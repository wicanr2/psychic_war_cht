// psychicwar 是《銀河超能力戰記》的可遊玩前端雛形（docs/spec/006）：原版 PW.EXE 跑在 dosgolem 上，
// 開視窗、接鍵盤與聲音，速度以牆上時間對齊 DOSBox 相容 cycles（預設 750，docs/spec/004）。
//
//	psychicwar -orig workplace/original/psychic-war
package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/translator"
	"github.com/wicanr2/dosgolem/xlate"
)

const sampleRate = 44100

// keyLog 由環境變數 PSYCHICWAR_KEYLOG=1 打開：把收到的按鍵印到標準錯誤（除錯用）。
var keyLog = os.Getenv("PSYCHICWAR_KEYLOG") == "1"

type game struct {
	o         *oracle.Oracle
	audio     *oracle.Audio
	ring      *psychicwar.Ring
	pacer     psychicwar.Pacer
	scale     int
	screen    *ebiten.Image
	rgba      []byte
	start     time.Time
	startCyc  uint64
	quitAfter time.Duration

	wav         *os.File // -wav：送出的取樣全部另存（16 位元單聲道 PCM，結束時補檔頭）
	wavSamples  int
	stats       *os.File
	lastStat    time.Time
	intercepted int
	exitErr     error

	tr      *translator.Translator // -text：中文疊字（docs/spec/008、009），nil ＝ 停用
	over    *ebiten.Image
	overPix []byte
}

func (g *game) machineMs() float64 {
	return float64(g.o.Cycles()-g.startCyc) / float64(g.pacer.PerMs)
}

func (g *game) Update() error {
	if g.start.IsZero() { // 牆上時鐘從第一次 Update 起算：建立視窗的時間不算落後
		g.start, g.startCyc, g.lastStat = time.Now(), g.o.Cycles(), time.Now()
		g.audio.Render() // 丟掉載入到現在的機器時間
	}
	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		if keyLog {
			sc, ok := psychicwar.ScanCode(k)
			log.Printf("按下 %v → %02X %v（第 %d 步）", k, sc, ok, g.o.Steps())
		}
		if psychicwar.Intercepted(k) {
			g.intercepted++
			continue
		}
		if sc, ok := psychicwar.ScanCode(k); ok {
			g.o.KeyDown(sc)
		}
	}
	for _, k := range inpututil.AppendJustReleasedKeys(nil) {
		if sc, ok := psychicwar.ScanCode(k); ok {
			g.o.KeyUp(sc)
		}
	}
	wall := time.Since(g.start)
	if n := g.pacer.Cycles(float64(wall.Microseconds())/1000, g.machineMs()); n > 0 {
		if err := g.o.RunCycles(n); err != nil {
			var exit *oracle.ExitError
			if errors.As(err, &exit) {
				return ebiten.Termination
			}
			g.exitErr = err
			return err
		}
	}
	if g.tr != nil {
		g.tr.Frame(g.o)
	}
	pcm := g.audio.Render()
	g.ring.Write(pcm)
	if g.wav != nil {
		_ = binary.Write(g.wav, binary.LittleEndian, pcm)
		g.wavSamples += len(pcm)
	}
	if g.stats != nil && time.Since(g.lastStat) >= time.Second {
		g.writeStats(wall)
	}
	if g.quitAfter > 0 && wall >= g.quitAfter {
		if g.stats != nil {
			g.writeStats(wall)
		}
		return ebiten.Termination
	}
	return nil
}

func (g *game) writeStats(wall time.Duration) {
	g.lastStat = time.Now()
	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	under, over := g.ring.Stats()
	line, _ := json.Marshal(map[string]any{
		"wall_ms":     wall.Milliseconds(),
		"machine_ms":  int64(g.machineMs()),
		"ticks":       g.o.Ticks(),
		"underruns":   under,
		"overflows":   over,
		"dropped_ms":  int64(g.pacer.DroppedMs),
		"intercepted": g.intercepted,
		"cpu_ms":      (ru.Utime.Nano() + ru.Stime.Nano()) / 1e6,
	})
	fmt.Fprintln(g.stats, string(line))
}

func (g *game) Draw(dst *ebiten.Image) {
	w, h, rgb := g.o.ScreenRGB()
	if w != 320 || h != 200 {
		return
	}
	for i := 0; i < w*h; i++ {
		g.rgba[4*i], g.rgba[4*i+1], g.rgba[4*i+2], g.rgba[4*i+3] = rgb[3*i], rgb[3*i+1], rgb[3*i+2], 0xFF
	}
	g.screen.WritePixels(g.rgba)
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(float64(g.scale), float64(g.scale))
	op.Filter = ebiten.FilterNearest
	dst.DrawImage(g.screen, &op)
	if g.tr != nil {
		clear(g.overPix)
		if g.tr.Layer.Draw(g.overPix, g.scale, g.tr.MissingGlyph) {
			g.over.WritePixels(g.overPix)
			dst.DrawImage(g.over, nil)
		}
	}
}

func (g *game) Layout(int, int) (int, int) { return 320 * g.scale, 200 * g.scale }

// stereoReader 把環形緩衝的單聲道取樣轉成 Ebiten 要的 16 位元雙聲道小端位元組。
type stereoReader struct {
	ring *psychicwar.Ring
	mono []int16
}

func (s *stereoReader) Read(p []byte) (int, error) {
	n := len(p) / 4
	if n == 0 {
		return 0, nil
	}
	if cap(s.mono) < n {
		s.mono = make([]int16, n)
	}
	s.ring.Read(s.mono[:n])
	for i, v := range s.mono[:n] {
		binary.LittleEndian.PutUint16(p[4*i:], uint16(v))
		binary.LittleEndian.PutUint16(p[4*i+2:], uint16(v))
	}
	return 4 * n, nil
}

// nullSink 沒有音效卡時照牆上時間把取樣讀走（docs/spec/006 §3.5），欠載照樣計數。
func nullSink(ring *psychicwar.Ring, stop <-chan struct{}) {
	t := time.NewTicker(10 * time.Millisecond)
	defer t.Stop()
	last := time.Now()
	carry := 0.0
	buf := make([]int16, sampleRate)
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			// 用 time.Now()，不用 ticker 送來的時間：CPU 被搶時 ticker 的時間戳可能早於上一次，算出負數（實測 panic）。
			now := time.Now()
			exact := now.Sub(last).Seconds()*sampleRate + carry
			last = now
			if exact < 0 {
				exact = 0
			}
			n := int(exact)
			carry = exact - float64(n)
			if n > len(buf) {
				n, carry = len(buf), 0
			}
			ring.Read(buf[:n])
		}
	}
}

// finishWAV 補上 WAV 檔頭（44 bytes，16 位元單聲道）。
func finishWAV(g *game) {
	h := make([]byte, 0, 44)
	put32 := func(v uint32) { h = binary.LittleEndian.AppendUint32(h, v) }
	put16 := func(v uint16) { h = binary.LittleEndian.AppendUint16(h, v) }
	data := uint32(2 * g.wavSamples)
	h = append(h, "RIFF"...)
	put32(36 + data)
	h = append(h, "WAVEfmt "...)
	put32(16)
	put16(1)
	put16(1)
	put32(sampleRate)
	put32(2 * sampleRate)
	put16(2)
	put16(16)
	h = append(h, "data"...)
	put32(data)
	_, _ = g.wav.WriteAt(h, 0)
	_ = g.wav.Close()
}

func parseCycles(s string) (uint64, error) {
	switch strings.ToLower(s) {
	case "xt":
		return 240, nil
	case "at8":
		return 750, nil
	case "at12":
		return 1510, nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil || v == 0 {
		return 0, fmt.Errorf("-cycles 看不懂：%q", s)
	}
	return v, nil
}

func main() {
	orig := flag.String("orig", "", "含 PW.EXE 的原版目錄（玩家自備）")
	cyclesFlag := flag.String("cycles", "750", "每毫秒 cycles，或 xt／at8／at12（docs/spec/004）")
	scale := flag.Int("scale", 3, "整數倍放大")
	adlib := flag.Bool("adlib", false, "388h 上有 OPL2（遊戲改走 .MID）")
	scratch := flag.String("scratch", "saves", "遊戲存檔寫到這裡")
	loadState := flag.String("load-state", "", "從 probe 狀態檔開始（除錯、測試用）")
	audioOut := flag.String("audio", "ebiten", "ebiten（音效卡）或 null（照牆上時間丟棄）")
	statsPath := flag.String("stats", "", "每秒寫一行 JSON 量測")
	quitAfter := flag.Duration("quit-after", 0, "牆上時間到了自己結束（自動驗收用）")
	wavPath := flag.String("wav", "", "把送給音效卡的取樣另存成 WAV（驗證聲音內容用）")
	textDir := flag.String("text", "text", "文本檔目錄（docs/spec/007）；空字串停用中文疊字")
	fontDir := flag.String("font", "font", "中文字型子集目錄：cjk24.golemfnt、cjk16.golemfnt（tools/font/bake.sh）")
	textLog := flag.String("text-log", "", "轉譯紀錄（JSON Lines）")
	flag.Parse()
	if *orig == "" {
		flag.Usage()
		os.Exit(2)
	}
	perMs, err := parseCycles(*cyclesFlag)
	if err != nil {
		log.Fatal(err)
	}
	o, err := oracle.Load(filepath.Join(*orig, "PW.EXE"), *orig)
	if err != nil {
		log.Fatal(err)
	}
	defer o.Close()
	if err := os.MkdirAll(*scratch, 0o755); err != nil {
		log.Fatal(err)
	}
	o.SetScratch(*scratch)
	if *loadState != "" {
		if err := o.LoadStateFile(*loadState); err != nil {
			log.Fatal(err)
		}
	}
	// 狀態檔會還原時鐘設定，所以速度與 AdLib 在載入之後才設。
	o.SetAdLib(*adlib)
	o.SetDOSBoxCycles(perMs)

	g := &game{
		o: o, ring: psychicwar.NewRing(sampleRate / 2), pacer: psychicwar.Pacer{PerMs: perMs},
		scale: *scale, screen: ebiten.NewImage(320, 200), rgba: make([]byte, 4*320*200),
		quitAfter: *quitAfter,
	}
	g.audio = o.NewAudio(sampleRate)
	if *textDir != "" {
		if *scale%3 != 0 {
			log.Printf("-scale %d 不是 3 的倍數，停用中文疊字（docs/spec/008 §3.5）", *scale)
		} else {
			entries, err := translator.LoadText(*textDir)
			if err != nil {
				log.Fatal(err)
			}
			f24, err := xlate.LoadFont(filepath.Join(*fontDir, "cjk24.golemfnt"))
			if err != nil {
				log.Fatal(err)
			}
			f16, err := xlate.LoadFont(filepath.Join(*fontDir, "cjk16.golemfnt"))
			if err != nil {
				log.Fatal(err)
			}
			var w io.Writer
			if *textLog != "" {
				f, err := os.Create(*textLog)
				if err != nil {
					log.Fatal(err)
				}
				defer f.Close()
				w = f
			}
			g.tr = translator.NewTranslator(entries, f24, f16, *scale, w)
			g.tr.Attach(o)
			g.over = ebiten.NewImage(320**scale, 200**scale)
			g.overPix = make([]byte, 4*320**scale*200**scale)
		}
	}
	g.ring.Write(make([]int16, sampleRate/20)) // 預填 50 ms 靜音
	if *statsPath != "" {
		if g.stats, err = os.Create(*statsPath); err != nil {
			log.Fatal(err)
		}
		defer g.stats.Close()
	}

	if *wavPath != "" {
		if g.wav, err = os.Create(*wavPath); err != nil {
			log.Fatal(err)
		}
		_, _ = g.wav.Write(make([]byte, 44)) // 檔頭結束時補
		defer finishWAV(g)
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	switch *audioOut {
	case "null":
		wg.Add(1)
		go func() { defer wg.Done(); nullSink(g.ring, stop) }()
	case "ebiten":
		ctx := audio.NewContext(sampleRate)
		p, err := ctx.NewPlayer(io.Reader(&stereoReader{ring: g.ring}))
		if err != nil {
			log.Fatal(err)
		}
		p.SetBufferSize(50 * time.Millisecond)
		p.Play()
	default:
		log.Fatalf("-audio 要是 ebiten 或 null：%q", *audioOut)
	}

	ebiten.SetWindowTitle("銀河超能力戰記 Psychic War")
	ebiten.SetWindowSize(320**scale, 200**scale)
	ebiten.SetTPS(60)
	runErr := ebiten.RunGameWithOptions(g, &ebiten.RunGameOptions{})
	close(stop)
	wg.Wait()
	if runErr != nil {
		log.Fatal(runErr)
	}
}
