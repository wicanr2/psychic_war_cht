// 正常九鍵交通及首鍵Up離開到達平台的原版／HD兩側驗收；研究038 §69。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/wicanr2/dosgolem/internal/dos"
	"github.com/wicanr2/dosgolem/internal/machine"
	"github.com/wicanr2/dosgolem/internal/state"
	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/dosgolem/xlate"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/theme"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/translator"
)

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}
func read(p string) []byte { b, e := os.ReadFile(p); check(e); return b }
func hash(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func write(p string, b []byte) {
	f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(e)
	_, e = f.Write(b)
	check(e)
	check(f.Close())
}
func jsonWrite(p string, v any) {
	b, e := json.MarshalIndent(v, "", "  ")
	check(e)
	write(p, append(b, '\n'))
}
func pngWrite(p string, pix []byte) {
	f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(e)
	check(png.Encode(f, &image.NRGBA{Pix: pix, Stride: 960 * 4, Rect: image.Rect(0, 0, 960, 600)}))
	check(f.Close())
}

type fingerprint struct {
	Regs                oracle.Regs
	Steps, Cycles, IRQ1 uint64
	RAM, Frame          string
}

func fp(o *oracle.Oracle) fingerprint {
	return fingerprint{o.Regs(), o.Steps(), o.Cycles(), o.IRQ1Delivered(), hash(o.Bytes(oracle.Addr{}, 0xa0000)), hash(o.Indexed())}
}

type world struct {
	m  *machine.Machine
	o  *oracle.Oracle
	tr *translator.Translator
	hd *theme.Theme
}

func main() {
	const orig = "/orig/psychic-war"
	const start = "workplace/states/15-minton2.state"
	const side = "workplace/hd/sivad-translation-v1-20261002-15-minton2.layer.json"
	const dir = "workplace/hd/theme-transport-room-v1-20261003"
	const out = "workplace/hd/transport-room-runtime-v1-20261003"
	paths, e := filepath.Glob(out + "*")
	check(e)
	if len(paths) > 0 {
		log.Fatal("拒絕覆寫正常接入收據")
	}
	inputs := map[string]string{}
	for _, p := range []string{start, side, "workplace/hd/transport-runtime-start-v1-20261003.json", "workplace/states/16-sivad.state", "workplace/states/16-sivad.frame", "replay/title-to-first-save.json", "tools/hd/verify_transport_room_runtime.go", "workplace/hd/transport-room-ready-spec-v1-20261003.txt", orig + "/PW.EXE", orig + "/ROOM0.PBL", dir + "/manifest.json"} {
		inputs[p] = hash(read(p))
	}
	if inputs[start] != "b2a06cb338d9e150a3b8931fdd451381da019668df3c5979672c14d80d49c8a9" || inputs[orig+"/PW.EXE"] != "88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49" {
		log.Fatal("原版固定起點不符")
	}
	for _, pattern := range []string{dir + "/*.png", "font/*.golemfnt", "text/*.json", "apps/psychicwar/theme/*.go", "apps/psychicwar/translator/*.go"} {
		ps, e := filepath.Glob(pattern)
		check(e)
		for _, p := range ps {
			inputs[p] = hash(read(p))
		}
	}
	entries, e := translator.LoadText("text")
	check(e)
	f24, e := xlate.LoadFont("font/cjk24.golemfnt")
	check(e)
	f16, e := xlate.LoadFont("font/cjk16.golemfnt")
	check(e)
	baked, e := translator.LoadBaked("text")
	check(e)
	worlds := []world{}
	for mode := 0; mode < 3; mode++ {
		m := machine.New()
		d := dos.New(m, orig)
		d.Install()
		m.KeyEvery = 3000000
		m.SetNextKey(520500000)
		for _, sc := range []uint8{0x48, 0x48, 0x48, 0x48, 0x48, 0x4b, 0x48, 0x50, 0x1c} {
			m.QueueKey(sc)
		}
		check(state.Load(start, m, d))
		d.Root = orig
		d.Scratch = "/tmp/pw-transport-runtime"
		o := oracle.ResearchWrapMachine(m, d)
		defer o.Close()
		if o.Word(oracle.Addr{Seg: 0x161, Off: 0x41df}) != 0xf95b {
			log.Fatal("執行前seed不符")
		}
		w := world{m: m, o: o}
		if mode > 0 {
			w.tr = translator.NewTranslator(entries, f24, f16, 3, nil)
			w.tr.Attach(o)
			check(w.tr.Layer.Restore(read(side), w.tr.Fonts()))
			w.tr.AttachBaked(baked, orig)
			w.hd, _, e = theme.LoadTheme(dir, orig, "theme", 3)
			check(e)
			if w.hd == nil {
				log.Fatal("未載入24筆主題")
			}
			w.hd.Enabled = mode == 2
			check(w.hd.Attach(o))
			w.hd.Frame(o)
			w.tr.Frame(o)
		}
		worlds = append(worlds, w)
	}
	points := []uint64{520000000, 556515520, 556515521, 556616000, 556911924, 562000000, 569118554, 569118555, 569200000, 569512925, 569827225, 570000000, 570303671, 570309346, 570400000, 570705341, 578000000, 614000000, 615000000, 617000000, 620000000}
	rows := []map[string]any{}
	for i, target := range points {
		for worlds[0].o.Steps() < target {
			n := target - worlds[0].o.Steps()
			if n > 100000 {
				n = 100000
			}
			for _, w := range worlds {
				check(w.o.Run(n))
				if w.hd != nil {
					w.hd.Frame(w.o)
					w.tr.Frame(w.o)
				}
			}
		}
		baseline := fp(worlds[0].o)
		for _, w := range worlds[1:] {
			if fp(w.o) != baseline {
				log.Fatalf("三分支原版取樣不符：%d", target)
			}
		}
		if target == 614000000 {
			o, e := oracle.Load(orig+"/PW.EXE", orig)
			check(e)
			check(o.LoadStateFile("workplace/states/16-sivad.state"))
			v := fp(o)
			v.IRQ1 = baseline.IRQ1
			if v != baseline {
				log.Fatal("交通終點與既有原版state不同")
			}
			o.Close()
		}
		p := fmt.Sprintf("%s-sample%02d", out, i)
		write(p+".frame", worlds[0].o.Indexed())
		var priorText, priorSide []byte
		for mode, w := range worlds[1:] {
			w.hd.Frame(w.o)
			w.tr.Frame(w.o)
			plane, text := make([]byte, 960*600*4), make([]byte, 960*600*4)
			w.hd.Draw(plane, 3)
			w.tr.Layer.Draw(text, 3, w.tr.MissingGlyph)
			layer, e := w.tr.Layer.Snapshot()
			check(e)
			if mode == 0 {
				priorText = text
				priorSide = layer
				if !bytes.Equal(plane, make([]byte, len(plane))) {
					log.Fatal("HD關閉仍輸出圖面")
				}
			} else {
				if !bytes.Equal(text, priorText) || !bytes.Equal(layer, priorSide) {
					log.Fatal("HD改動中文Layer")
				}
				write(p+"-plane.rgba", plane)
				write(p+"-text.rgba", text)
				write(p+".state.xlate.json", layer)
				check(w.o.SaveStateFile(p + ".state"))
				_, _, rgb := w.o.ScreenRGB()
				write(p+"-rgb.bin", rgb)
				composed := image.NewNRGBA(image.Rect(0, 0, 960, 600))
				for y := 0; y < 600; y++ {
					for x := 0; x < 960; x++ {
						a, b := 3*((y/3)*320+x/3), 4*(y*960+x)
						copy(composed.Pix[b:b+3], rgb[a:a+3])
						composed.Pix[b+3] = 255
					}
				}
				draw.Draw(composed, composed.Bounds(), &image.NRGBA{Pix: plane, Stride: 960 * 4, Rect: composed.Bounds()}, image.Point{}, draw.Over)
				draw.Draw(composed, composed.Bounds(), &image.NRGBA{Pix: text, Stride: 960 * 4, Rect: composed.Bounds()}, image.Point{}, draw.Over)
				pngWrite(p+"-hd-chinese.png", composed.Pix)
			}
		}
		rows = append(rows, map[string]any{"step": target, "prefix": p, "original": baseline, "chinese_equal": true, "text_stamp_count": len(worlds[2].tr.Layer.Stamps)})
		if target == 614000000 {
			for _, w := range worlds {
				w.m.SetNextKey(614500000)
				w.m.QueueKey(0x48)
			}
		}
	}
	if worlds[0].o.IRQ1Delivered() != 20 {
		log.Fatal("正常十鍵IRQ1不是20次")
	}
	endBus := hash(worlds[0].o.Bytes(oracle.Addr{}, 1<<20))
	for _, w := range worlds[1:] {
		if hash(w.o.Bytes(oracle.Addr{}, 1<<20)) != endBus {
			log.Fatal("原版終點完整匯流排不同")
		}
	}
	jsonWrite(out+".json", map[string]any{"scope": "正常九鍵交通及17-saved2首鍵Up，原版／HD關閉／HD開啟三分支同seed接續", "inputs_sha256": inputs, "seed_before": "F95B", "seed_method": "執行前同固定雜湊state，唯讀核對；未改RAM或重擲", "theme": dir, "manifest_entries": 24, "samples": rows, "keys_irq1": 20, "end": fp(worlds[0].o), "end_bus_sha256": endBus, "limits": "正常原版及中文覆繪接入抽測；未驗GUI、美術、其他房間、完整sprite、幀率或正式交付"})
	fmt.Println("正常交通21取樣，三分支原版終點與HD兩側中文字面相同；十鍵20 IRQ1")
}
