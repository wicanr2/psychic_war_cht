// 正常九鍵治療及首鍵Up離房的原版／HD兩側驗收；研究038 §66。
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
	RAM, Bus, Frame     string
}

func fp(o *oracle.Oracle) fingerprint {
	return fingerprint{o.Regs(), o.Steps(), o.Cycles(), o.IRQ1Delivered(), hash(o.Bytes(oracle.Addr{}, 0xa0000)), hash(o.Bytes(oracle.Addr{}, 1<<20)), hash(o.Indexed())}
}

type world struct {
	m  *machine.Machine
	o  *oracle.Oracle
	tr *translator.Translator
	hd *theme.Theme
}

func main() {
	const orig = "/orig/psychic-war"
	const start = "workplace/states/12-battle2.state"
	const side = "workplace/hd/normal-chain-translation-v1-20261001-12-battle2.layer.json"
	const dir = "workplace/hd/theme-room-v1-20261003"
	const out = "workplace/hd/room-runtime-v1-20261003"
	paths, e := filepath.Glob(out + "*")
	check(e)
	if len(paths) > 0 {
		log.Fatal("拒絕覆寫正常接入收據")
	}
	inputs := map[string]string{}
	for _, p := range []string{start, side, "replay/title-to-first-save.json", "tools/hd/verify_room_runtime.go", "workplace/hd/room-ready-spec-v1-20261003.txt", orig + "/PW.EXE", orig + "/ROOM0.PBL", dir + "/manifest.json"} {
		inputs[p] = hash(read(p))
	}
	if inputs[start] != "94912dee5881b8f7b878fed44465a7afdbbe47d2affef5a19226a898e5f60993" || inputs[orig+"/PW.EXE"] != "88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49" {
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
		m.SetNextKey(280500000)
		for _, sc := range []uint8{0x50, 0x48, 0x48, 0x4d, 0x48, 0x48, 0x4d, 0x48, 0x1c} {
			m.QueueKey(sc)
		}
		check(state.Load(start, m, d))
		d.Root = orig
		d.Scratch = "/tmp/pw-room-runtime"
		o := oracle.ResearchWrapMachine(m, d)
		defer o.Close()
		if o.Word(oracle.Addr{Seg: 0x161, Off: 0x41df}) != 0xa48c {
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
				log.Fatal("未載入21筆主題")
			}
			w.hd.Enabled = mode == 2
			check(w.hd.Attach(o))
			w.hd.Frame(o)
			w.tr.Frame(o)
		}
		worlds = append(worlds, w)
	}
	points := []uint64{280000000, 322515156, 322515157, 322515668, 322525156, 322565156, 322715156, 322911294, 332000000, 345000000, 346000000, 349000000, 351000000}
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
		if target == 345000000 {
			o, e := oracle.Load(orig+"/PW.EXE", orig)
			check(e)
			check(o.LoadStateFile("workplace/states/13-healed.state"))
			v := fp(o)
			v.IRQ1 = baseline.IRQ1
			if v != baseline {
				log.Fatal("治療終點與既有原版state不同")
			}
			o.Close()
		}
		for _, step := range []uint64{346000000, 349000000} {
			if target == step {
				p := fmt.Sprintf("workplace/hd/room-exit-v1-20261003-%d.frame", step)
				inputs[p] = hash(read(p))
				if !bytes.Equal(worlds[0].o.Indexed(), read(p)) {
					log.Fatal("正常Up與獨立原版probe不同")
				}
			}
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
		if target == 345000000 {
			for _, w := range worlds {
				w.m.SetNextKey(345500000)
				w.m.QueueKey(0x48)
			}
		}
	}
	if worlds[0].o.IRQ1Delivered() != 20 {
		log.Fatal("正常十鍵IRQ1不是20次")
	}
	jsonWrite(out+".json", map[string]any{"scope": "正常九鍵治療及14-minton1首鍵Up，原版／HD關閉／HD開啟三分支同seed接續", "inputs_sha256": inputs, "seed_before": "A48C", "seed_method": "執行前同固定雜湊state，唯讀核對；未改RAM或重擲", "theme": dir, "manifest_entries": 21, "samples": rows, "keys_irq1": 20, "end": fp(worlds[0].o), "limits": "正常原版及中文覆繪接入抽測；未驗GUI、美術、其他房間、完整sprite、幀率或正式交付"})
	fmt.Println("正常房間13取樣，三分支原版終點與HD兩側中文字面相同；十鍵20 IRQ1")
}
