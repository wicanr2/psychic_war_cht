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
	"image"
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
	"github.com/wicanr2/dosgolem/xlate"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/translator"
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

	// 輔助熱鍵（docs/spec/012）
	help      bool   // F1：說明頁顯示中
	english   bool   // F2：切到英文原文（疊字不畫）
	toast     string // F10／F11 的提示
	toastTill time.Time
	quickDir  string // 即時存檔放哪（＝ -scratch）
	origDir   string
	textDir   string
	baked     []translator.BakedEntry
	fontHelp  *xlate.Font
	helpLines []string
	cheat     bool // -cheat：打開 F5／F6（docs/spec/014）
	amap      *psychicwar.AutoMap
	mouseKey  ebiten.Key // 滑鼠按住時送出的鍵（docs/spec/018）
	mouseDown bool
	rec       *psychicwar.Recording // -record：輸入錄製（docs/spec/019）
	recPath   string
	showMap   bool // F3：自動地圖顯示中（docs/spec/015）
}

// 自動地圖的版面（docs/spec/015 §2）：一格 12 像素、左上角 (24, 24)、最多 32×32 格。
const (
	mapCell  = 12
	mapOX    = 24
	mapOY    = 24
	mapCells = 32
	mapHeadY = 4
)

// noteMap 每一幀記一次目前的格子。
func (g *game) noteMap() {
	if g.amap == nil {
		return
	}
	g.amap.Note(g.word(psychicwar.AddrArea), g.word(psychicwar.AddrMapX), g.word(psychicwar.AddrMapY))
}

// drawMap 畫 F3 自動地圖。
func (g *game) drawMap(dst *ebiten.Image) {
	if !g.showMap || g.amap == nil || g.over == nil || g.fontHelp == nil {
		return
	}
	w, h := 320*g.scale, 200*g.scale
	area, x, y := g.word(psychicwar.AddrArea), g.word(psychicwar.AddrMapX), g.word(psychicwar.AddrMapY)
	clear(g.overPix)
	// 底色：整頁不透明黑
	for i := 0; i < w*h; i++ {
		g.overPix[4*i+3] = 0xFF
	}
	put := func(px, py int, c [3]uint8) {
		if px < 0 || px >= w || py < 0 || py >= h {
			return
		}
		i := 4 * (py*w + px)
		g.overPix[i], g.overPix[i+1], g.overPix[i+2], g.overPix[i+3] = c[0], c[1], c[2], 0xFF
	}
	box := func(cx, cy int, c [3]uint8) {
		for dy := 0; dy < mapCell-2; dy++ {
			for dx := 0; dx < mapCell-2; dx++ {
				put(mapOX+cx*mapCell+dx, mapOY+cy*mapCell+dy, c)
			}
		}
	}
	grey := [3]uint8{0xAA, 0xAA, 0xAA}
	white := [3]uint8{0xFF, 0xFF, 0xFF}
	for _, c := range g.amap.Cells(area) {
		if int(c[0]) < mapCells && int(c[1]) < mapCells {
			box(int(c[0]), int(c[1]), grey)
		}
	}
	if int(x) < mapCells && int(y) < mapCells {
		box(int(x), int(y), white)
	}
	head := fmt.Sprintf(psychicwar.MapHeader,
		area, x, y, g.amap.Count(area), psychicwar.FacingMark(g.word(psychicwar.AddrFacing)))
	psychicwar.DrawTextPage(g.overPix, w, mapHeadY*g.scale+8*g.scale, g.fontHelp, []string{head},
		8*g.scale, white, [3]uint8{0, 0, 0}, 0)
	g.over.WritePixels(g.overPix)
	dst.DrawImage(g.over, nil)
}

// word 讀一個線性位址的字組。
func (g *game) word(lin uint32) uint16 {
	return g.o.Word(oracle.Addr{Seg: uint16(lin >> 4), Off: uint16(lin & 0xF)})
}

func (g *game) setWord(lin uint32, v uint16) {
	g.o.SetWord(oracle.Addr{Seg: uint16(lin >> 4), Off: uint16(lin & 0xF)}, v)
}

// cheatFull 把 HP 與能量補到上限（docs/spec/014 §3）。
func (g *game) cheatFull() string {
	done := 0
	for _, p := range [][2]uint32{{psychicwar.AddrHP, psychicwar.AddrHPMax}, {psychicwar.AddrEnergy, psychicwar.AddrEnergyMax}} {
		if v := psychicwar.FullValue(g.word(p[0]), g.word(p[1])); v != 0 {
			g.setWord(p[0], v)
			done++
		}
	}
	if done == 0 {
		return "沒東西可補"
	}
	return "HP 與能量補滿"
}

// cheatWeakenEnemy 把敵人 HP 設成 1（只在戰鬥中）。
func (g *game) cheatWeakenEnemy() string {
	if !psychicwar.EnemyHPSane(g.word(psychicwar.AddrEnemyHP)) {
		return "現在不是戰鬥"
	}
	g.setWord(psychicwar.AddrEnemyHP, 1)
	return "敵人剩 1 點"
}

// 狀態檔格式的版本字串：dosgolem 沒有版本常數，格式換了就手動升這個版號（docs/spec/012 §4）。
const stateFormat = "dosgolem-state/1"

// hotkeys 處理 F1／F2／F10／F11（docs/spec/012）。回傳有沒有處理掉。
func (g *game) hotkeys(k ebiten.Key) bool {
	switch k {
	case ebiten.KeyF1:
		g.help = !g.help
		return true
	case ebiten.KeyF3:
		g.showMap = !g.showMap
		return true
	case ebiten.KeyF2:
		g.english = !g.english
		g.showToast(map[bool]string{true: "英文原文", false: "中文"}[g.english])
		return true
	case ebiten.KeyF5:
		if g.cheat {
			g.showToast(g.cheatFull())
		}
		return true
	case ebiten.KeyF6:
		if g.cheat {
			g.showToast(g.cheatWeakenEnemy())
		}
		return true
	case ebiten.KeyF10:
		g.showToast(g.quickSave())
		return true
	case ebiten.KeyF11:
		g.showToast(g.quickLoad())
		return true
	}
	return false
}

func (g *game) showToast(s string) {
	if s == "" {
		return
	}
	g.toast, g.toastTill = s, time.Now().Add(2*time.Second)
}

// quickSave 存狀態、疊字層快照與中繼資料；回畫面上要顯示的一行字。
func (g *game) quickSave() string {
	base := filepath.Join(g.quickDir, "quick.state")
	if err := g.o.SaveStateFile(base); err != nil {
		return "存檔失敗：" + err.Error()
	}
	if g.tr != nil {
		if b, err := g.tr.Layer.Snapshot(); err == nil {
			_ = os.WriteFile(base+".xlate.json", b, 0o644)
		}
	}
	exe, err := psychicwar.FileSHA256(filepath.Join(g.origDir, "PW.EXE"))
	if err != nil {
		return "存檔失敗：" + err.Error()
	}
	text, _ := psychicwar.DirSHA256(g.textDir)
	lang := "zh"
	if g.english {
		lang = "en"
	}
	if b, err := json.Marshal(g.amap); err == nil { // 自動地圖跟著即時存檔（docs/spec/015 §3）
		_ = os.WriteFile(filepath.Join(g.quickDir, "quick.map.json"), b, 0o644)
	}
	if err := psychicwar.WriteQuickMeta(filepath.Join(g.quickDir, "quick.json"),
		psychicwar.NewQuickMeta(stateFormat, exe, text, lang)); err != nil {
		return "存檔失敗：" + err.Error()
	}
	return "已存檔"
}

// quickLoad 版本相符才讀；不符就拒絕，不動目前的遊戲。
func (g *game) quickLoad() string {
	base := filepath.Join(g.quickDir, "quick.state")
	m, err := psychicwar.ReadQuickMeta(filepath.Join(g.quickDir, "quick.json"))
	if err != nil {
		return "沒有即時存檔"
	}
	exe, err := psychicwar.FileSHA256(filepath.Join(g.origDir, "PW.EXE"))
	if err != nil {
		return "讀檔失敗：" + err.Error()
	}
	text, _ := psychicwar.DirSHA256(g.textDir)
	ok, why := psychicwar.CheckQuickMeta(m, stateFormat, exe, text)
	if !ok {
		return "不能讀：" + why
	}
	if err := g.o.LoadStateFile(base); err != nil {
		return "讀檔失敗：" + err.Error()
	}
	g.startCyc, g.start = g.o.Cycles(), time.Now() // 牆上時間重新對齊，不然會狂追進度
	if g.tr != nil {
		g.tr.ResetForLoad()
		if b, err := os.ReadFile(base + ".xlate.json"); err == nil {
			_ = g.tr.Layer.Restore(b, g.tr.Fonts())
		}
		g.tr.AttachBaked(g.baked, g.origDir) // watcher 不進快照，讀檔後重新登記（dosgolem 規格 203 §2.3）
	}
	if b, err := os.ReadFile(filepath.Join(g.quickDir, "quick.map.json")); err == nil {
		m2 := psychicwar.NewAutoMap()
		if json.Unmarshal(b, m2) == nil {
			g.amap = m2
		}
	}
	g.english = m.Language == "en"
	if why != "" {
		return "已讀檔（" + why + "）"
	}
	return "已讀檔"
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
			g.hotkeys(k)
			continue
		}
		if sc, ok := psychicwar.ScanCode(k); ok {
			g.o.KeyDown(sc)
			g.record(k, true)
		}
	}
	// 非 ASCII 的文字輸入（中文輸入法）送的是字元不是掃描碼，本來就進不到遊戲裡。
	// 玩家按半天沒有反應，畫面上要說出為什麼——被擋掉的輸入不能靜默消失（docs/spec/017 §3）。
	if psychicwar.NonASCII(ebiten.AppendInputChars(nil)) {
		g.showToast(psychicwar.ASCIIOnlyToast)
	}
	g.mouse()
	for _, k := range inpututil.AppendJustReleasedKeys(nil) {
		if sc, ok := psychicwar.ScanCode(k); ok {
			g.o.KeyUp(sc)
			g.record(k, false)
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
	g.noteMap()
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
	if g.tr != nil && !g.english { // F2：英文模式不畫疊字（docs/spec/012 §2）
		clear(g.overPix)
		if g.tr.Layer.Draw(g.overPix, g.scale, g.tr.MissingGlyph) {
			g.over.WritePixels(g.overPix)
			dst.DrawImage(g.over, nil)
		}
	}
	g.drawMap(dst)
	g.drawHelp(dst)
	g.drawToast(dst)
}

// drawHelp 畫 F1 說明頁（docs/spec/012 §3）：蓋滿整個畫布，固定顏色。
func (g *game) drawHelp(dst *ebiten.Image) {
	if !g.help || g.over == nil || g.fontHelp == nil || len(g.helpLines) == 0 {
		return
	}
	w, h := 320*g.scale, 200*g.scale
	lines := g.helpLines
	if g.tr != nil && g.tr.InProtection() {
		if a := psychicwar.ProtectionAnswer(g.o.Bytes(oracle.Addr{Seg: psychicwar.ProtAnswerSeg, Off: psychicwar.ProtAnswerOff}, psychicwar.ProtAnswerLen)); a != "" {
			lines = append(append([]string{}, lines...), "", psychicwar.ProtectionLabel+a) // 防拷畫面才有（docs/spec/013 §2.2）
		}
	}
	clear(g.overPix)
	psychicwar.DrawTextPage(g.overPix, w, h, g.fontHelp, lines,
		8*g.scale, [3]uint8{0xFF, 0xFF, 0xFF}, [3]uint8{0, 0, 0}, 0xFF)
	g.over.WritePixels(g.overPix)
	dst.DrawImage(g.over, nil)
}

// drawToast 畫 F10／F11 的提示（兩秒）。
func (g *game) drawToast(dst *ebiten.Image) {
	if g.toast == "" || time.Now().After(g.toastTill) || g.over == nil || g.fontHelp == nil {
		return
	}
	w, h := 320*g.scale, 200*g.scale
	clear(g.overPix)
	psychicwar.DrawTextPage(g.overPix, w, 8*g.scale, g.fontHelp, []string{g.toast},
		8*g.scale, [3]uint8{0xFF, 0xFF, 0x55}, [3]uint8{0, 0, 0}, 0xFF)
	g.over.WritePixels(g.overPix)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, float64(h-8*g.scale))
	dst.DrawImage(g.over.SubImage(image.Rect(0, 0, w, 8*g.scale)).(*ebiten.Image), op)
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

// version 由 -ldflags -X main.version 打進來（docs/spec/021 §4）。
var version = "dev"

// notext 是 -text 的停用值。空字串現在表示「用預設位置」（docs/spec/021 §3.1），
// 所以停用疊字要另外給一個值。
const notext = "off"

func main() {
	orig := flag.String("orig", "", "含 PW.EXE 的原版目錄（玩家自備）")
	cyclesFlag := flag.String("cycles", "750", "每毫秒 cycles，或 xt／at8／at12（docs/spec/004）")
	scale := flag.Int("scale", 3, "整數倍放大")
	adlib := flag.Bool("adlib", false, "388h 上有 OPL2（遊戲改走 .MID）")
	scratch := flag.String("scratch", "", "遊戲存檔寫到這裡（預設是使用者資料目錄，docs/spec/021 §3.2）")
	loadState := flag.String("load-state", "", "從 probe 狀態檔開始（除錯、測試用）")
	audioOut := flag.String("audio", "ebiten", "ebiten（音效卡）或 null（照牆上時間丟棄）")
	statsPath := flag.String("stats", "", "每秒寫一行 JSON 量測")
	quitAfter := flag.Duration("quit-after", 0, "牆上時間到了自己結束（自動驗收用）")
	wavPath := flag.String("wav", "", "把送給音效卡的取樣另存成 WAV（驗證聲音內容用）")
	textDir := flag.String("text", "", "文本檔目錄（docs/spec/007）；預設找執行檔旁的 text/，"+notext+" 停用中文疊字")
	fontDir := flag.String("font", "", "中文字型子集目錄：cjk24.golemfnt、cjk16.golemfnt（預設找執行檔旁的 font/）")
	textLog := flag.String("text-log", "", "轉譯紀錄（JSON Lines）")
	cheat := flag.Bool("cheat", false, "打開作弊熱鍵 F5（補滿 HP 與能量）、F6（敵人剩 1 點），docs/spec/014")
	recordPath := flag.String("record", "", "把按鍵錄成重播檔（docs/spec/019）：記指令數不記時間，換一台機器也能重現")
	showVersion := flag.Bool("version", false, "印出版本後結束")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}
	// 發行包的路徑（docs/spec/021 §3）：旗標沒給時找執行檔旁，不是 cwd。
	if *orig == "" {
		*orig = psychicwar.OrigDir()
	}
	if *scratch == "" {
		*scratch = psychicwar.SaveDir("saves")
	}
	if *textDir == "" {
		*textDir = psychicwar.DataDir("text")
	} else if *textDir == notext {
		*textDir = ""
	}
	if *fontDir == "" {
		*fontDir = psychicwar.DataDir("font")
	}
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
	g.quickDir, g.origDir, g.textDir = *scratch, *orig, *textDir
	g.cheat = *cheat
	g.amap = psychicwar.NewAutoMap()
	if *recordPath != "" { // 輸入錄製（docs/spec/019）
		exe, err := psychicwar.FileSHA256(filepath.Join(*orig, "PW.EXE"))
		if err != nil {
			log.Fatal(err)
		}
		g.rec = psychicwar.NewRecording(exe, int(perMs), *loadState, o.Steps())
		g.recPath = *recordPath
	}
	if lines, err := psychicwar.LoadHelp(*textDir); err != nil { // 排不下或讀不到就不要進畫面（docs/spec/012 §5）
		log.Printf("讀不到說明頁（F1 停用）：%v", err)
	} else {
		g.helpLines = lines
	}
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
			g.fontHelp = f24
			if baked, err := translator.LoadBaked(*textDir); err != nil {
				log.Fatal(err)
			} else {
				g.baked = baked
				g.tr.AttachBaked(baked, *orig)
			}
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
	g.saveRecording()
	if runErr != nil {
		log.Fatal(runErr)
	}
}

// mouse 處理滑鼠點擊操作面板（docs/spec/018）。
//
// 按下送 KeyDown、放開送 KeyUp，與鍵盤走同一條路——**不模擬「按一下」**：
// 原版有些操作要按住（戰鬥時的攻擊），維持按住／放開的語意才不會漏掉這類用法。
func (g *game) mouse() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		if h := psychicwar.HotspotAt(mx/g.scale, my/g.scale); h != nil {
			if sc, ok := psychicwar.ScanCode(h.Key); ok {
				g.o.KeyDown(sc)
				g.mouseKey = h.Key
				g.mouseDown = true
			}
		}
	}
	if g.mouseDown && inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		if sc, ok := psychicwar.ScanCode(g.mouseKey); ok {
			g.o.KeyUp(sc)
		}
		g.mouseDown = false
	}
}

// record 記一筆按鍵（docs/spec/019 §4）。
// 熱鍵不會走到這裡——它們在 Update 裡就被 Intercepted 攔掉了，那是前端自己的功能，不進遊戲。
func (g *game) record(k ebiten.Key, down bool) {
	if g.rec == nil {
		return
	}
	g.rec.Add(g.o.Steps(), psychicwar.KeyName(k), down)
}

// saveRecording 在結束時把錄製寫出來。
func (g *game) saveRecording() {
	if g.rec == nil || g.recPath == "" {
		return
	}
	if err := g.rec.Save(g.recPath); err != nil {
		log.Printf("寫錄製檔失敗：%v", err)
		return
	}
	log.Printf("錄製 %d 筆事件 → %s", len(g.rec.Events), g.recPath)
}
