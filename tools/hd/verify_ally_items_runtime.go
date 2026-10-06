// 研究038 §101：正常名字、道具、Forget it及移動，原版／中文原版／中文HD三分支。
// 以dosgolem cmd/probe的Go覆映射建置；不改原版RAM或正式執行器。
package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"

	"github.com/wicanr2/dosgolem/internal/dos"
	"github.com/wicanr2/dosgolem/internal/machine"
	"github.com/wicanr2/dosgolem/internal/state"
	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/dosgolem/xlate"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/theme"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/translator"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
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
func saved(p string) (machineSaved, []byte) {
	z, e := gzip.NewReader(bytes.NewReader(read(p)))
	check(e)
	defer z.Close()
	section := func() []byte {
		var n uint64
		check(binary.Read(z, binary.LittleEndian, &n))
		if n > 32<<20 {
			log.Fatal("區段越界")
		}
		b := make([]byte, n)
		_, e := io.ReadFull(z, b)
		check(e)
		return b
	}
	mb, db := section(), section()
	tail, e := io.ReadAll(z)
	check(e)
	if len(tail) != 0 {
		log.Fatal("未解析尾端")
	}
	var s machineSaved
	check(gob.NewDecoder(bytes.NewReader(mb)).Decode(&s))
	return s, db
}
func compareState(a, b string) {
	am, ad := saved(a)
	bm, bd := saved(b)
	if !reflect.DeepEqual(am, bm) || !bytes.Equal(ad, bd) {
		log.Fatalf("完整state不符：%s / %s", a, b)
	}
	bad := bm
	bad.R[0] ^= 1
	if reflect.DeepEqual(am, bad) {
		log.Fatal("CPU負對照未檢出")
	}
	bad = bm
	bad.Mem = append([]byte(nil), bm.Mem...)
	bad.Mem[0] ^= 1
	if reflect.DeepEqual(am, bad) {
		log.Fatal("RAM負對照未檢出")
	}
	bad = bm
	bad.Ports = map[uint16]uint8{}
	for k, v := range bm.Ports {
		bad.Ports[k] = v
	}
	bad.Ports[0x3ce] ^= 1
	if reflect.DeepEqual(am, bad) {
		log.Fatal("port負對照未檢出")
	}
}

type fingerprint struct {
	Regs                oracle.Regs
	Steps, Cycles, IRQ1 uint64
	Bus, Frame          string
}

func fp(o *oracle.Oracle) fingerprint {
	return fingerprint{o.Regs(), o.Steps(), o.Cycles(), o.IRQ1Delivered(), hash(o.Bytes(oracle.Addr{}, 1<<20)), hash(o.Indexed())}
}

type world struct {
	m  *machine.Machine
	o  *oracle.Oracle
	tr *translator.Translator
	hd *theme.Theme
}

const orig = "/orig/psychic-war"
const dir = "workplace/hd/theme-ally-items-v1-20261004"
const output = "workplace/hd/ally-items-v1-20261004/runtime-v2"

func queue(w world, at, every uint64, keys ...uint8) {
	w.m.KeyEvery = every
	w.m.SetNextKey(at)
	for _, k := range keys {
		w.m.QueueKey(k)
	}
}
func frame(w world) {
	if w.hd != nil {
		w.hd.Frame(w.o)
		w.tr.Frame(w.o)
	}
}
func planes(w world) ([]byte, []byte, []byte) {
	p, t := make([]byte, 960*600*4), make([]byte, 960*600*4)
	w.hd.Draw(p, 3)
	w.tr.Layer.Draw(t, 3, w.tr.MissingGlyph)
	s, e := w.tr.Layer.Snapshot()
	check(e)
	return p, t, s
}
func compose(p string, rgb []byte, layers ...[]byte) {
	im := image.NewNRGBA(image.Rect(0, 0, 960, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 960; x++ {
			a, b := 3*((y/3)*320+x/3), 4*(y*960+x)
			copy(im.Pix[b:b+3], rgb[a:a+3])
			im.Pix[b+3] = 255
		}
	}
	for _, l := range layers {
		draw.Draw(im, im.Bounds(), &image.NRGBA{Pix: l, Stride: 960 * 4, Rect: im.Bounds()}, image.Point{}, draw.Over)
	}
	f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(e)
	check(png.Encode(f, im))
	check(f.Close())
}
func capture(p string, w world) []byte {
	plane, text, side := planes(w)
	_, _, rgb := w.o.ScreenRGB()
	write(p+".frame", w.o.Indexed())
	write(p+"-rgb.bin", rgb)
	write(p+"-plane.rgba", plane)
	write(p+"-text.rgba", text)
	write(p+".state.xlate.json", side)
	compose(p+"-english.png", rgb, plane)
	compose(p+"-chinese.png", rgb, plane, text)
	return plane
}
func main() {
	old, e := filepath.Glob(output + "*")
	check(e)
	if len(old) != 0 {
		log.Fatal("拒絕覆寫runtime收據")
	}
	start := "workplace/states/06-name.state"
	if hash(read(start)) != "ac8eec0bbc533b267782298c298ce4a60440d0568266e665c8a5badbf9e0ca4a" || hash(read(orig+"/PW.EXE")) != "88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49" {
		log.Fatal("固定原版身份不同")
	}
	inputs := map[string]string{}
	for _, pattern := range []string{start, orig + "/*.PBL", orig + "/PW.EXE", dir + "/*", "font/*.golemfnt", "text/*.json", "apps/psychicwar/theme/*.go", "apps/psychicwar/translator/*.go", "tools/hd/verify_ally_items_runtime.go"} {
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
	attach := func(w *world, enabled bool) {
		w.tr = translator.NewTranslator(entries, f24, f16, 3, nil)
		w.tr.Attach(w.o)
		w.tr.AttachBaked(baked, orig)
		w.hd, _, e = theme.LoadTheme(dir, orig, "", 3)
		check(e)
		w.hd.Enabled = enabled
		check(w.hd.Attach(w.o))
		check(w.hd.Attach(w.o))
		frame(*w)
	}
	worlds := []world{}
	for mode := 0; mode < 3; mode++ {
		m := machine.New()
		d := dos.New(m, orig)
		d.Install()
		w := world{m: m}
		queue(w, 35500000, 500000, 0x25, 0x1e, 0x17, 0x1c)
		check(state.Load(start, m, d))
		d.Root = orig
		d.Scratch = "/tmp/pw-ally-items"
		w.o = oracle.ResearchWrapMachine(m, d)
		defer w.o.Close()
		if w.o.Word(oracle.Addr{Seg: 0x161, Off: 0x41df}) != 0x86af {
			log.Fatal("起始seed不同")
		}
		if mode > 0 {
			attach(&w, mode == 2)
		}
		worlds = append(worlds, w)
	}
	events := []map[string]any{}
	worlds[0].o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x8705}, func(o *oracle.Oracle) {
		r := o.Regs()
		if r.CX == 0x2002 && r.DX == 0x0304 {
			raw := o.Bytes(oracle.Addr{Seg: r.DS, Off: r.BX}, 384)
			p := fmt.Sprintf("%s-source%02d.bin", output, len(events))
			write(p, raw)
			events = append(events, map[string]any{"step": o.Steps(), "event": "entry", "regs": r, "source": p, "sha256": hash(raw)})
		}
	})
	worlds[0].o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x8751}, func(o *oracle.Oracle) {
		if len(events) > 0 && events[len(events)-1]["event"] == "entry" {
			events = append(events, map[string]any{"step": o.Steps(), "event": "return"})
		}
	})
	points := []uint64{35000000, 42000000, 48000000, 54799098, 54799099, 54800000, 54805000, 54825000, 55000000, 58000000, 66000000, 72000000, 80000000}
	rows := []map[string]any{}
	for i, target := range points {
		for worlds[0].o.Steps() < target {
			n := target - worlds[0].o.Steps()
			if n > 100000 {
				n = 100000
			}
			for _, w := range worlds {
				check(w.o.Run(n))
				frame(w)
			}
		}
		base := fp(worlds[0].o)
		for _, w := range worlds[1:] {
			if fp(w.o) != base {
				log.Fatalf("原版三分支不同：%d", target)
			}
		}
		p := fmt.Sprintf("%s-sample%02d", output, i)
		for mode, w := range worlds {
			check(w.o.SaveStateFile(fmt.Sprintf("%s-mode%d.state", p, mode)))
		}
		compareState(p+"-mode0.state", p+"-mode1.state")
		compareState(p+"-mode0.state", p+"-mode2.state")
		plane := capture(p, worlds[2])
		disabled, text, side := planes(worlds[1])
		_, text2, side2 := planes(worlds[2])
		if !bytes.Equal(disabled, make([]byte, len(disabled))) || !bytes.Equal(text, text2) || !bytes.Equal(side, side2) {
			log.Fatal("HD關閉圖面或中文不同")
		}
		worlds[2].hd.Enabled = false
		off, _, _ := planes(worlds[2])
		if !bytes.Equal(off, disabled) {
			log.Fatal("HD切換未清空")
		}
		worlds[2].hd.Enabled = true
		on, _, _ := planes(worlds[2])
		if !bytes.Equal(on, plane) {
			log.Fatal("HD切換恢復不同")
		}
		// 真正冷載保存產物，兩側使用同一份state，HD來源身份不序列化。
		reloads := []world{}
		for mode := 0; mode < 2; mode++ {
			m := machine.New()
			d := dos.New(m, orig)
			d.Install()
			check(state.Load(p+"-mode2.state", m, d))
			d.Root = orig
			d.Scratch = "/tmp/pw-ally-items-reload"
			w := world{m: m, o: oracle.ResearchWrapMachine(m, d)}
			if mode == 1 {
				attach(&w, true)
				check(w.tr.Layer.Restore(side2, w.tr.Fonts()))
				w.hd.ResetForLoad()
				frame(w)
			}
			reloads = append(reloads, w)
		}
		capture(p+"-reload", reloads[1])
		for _, w := range reloads {
			check(w.o.Run(100000))
			frame(w)
		}
		for mode, w := range reloads {
			check(w.o.SaveStateFile(fmt.Sprintf("%s-reload-mode%d.state", p, mode)))
		}
		compareState(p+"-reload-mode0.state", p+"-reload-mode1.state")
		if fp(reloads[0].o) != fp(reloads[1].o) {
			log.Fatal("載回接續HD干擾原版")
		}
		capture(p+"-reload-next", reloads[1])
		for _, w := range reloads {
			w.o.Close()
		}
		rows = append(rows, map[string]any{"step": target, "prefix": p, "original": base, "machine_fields": reflect.TypeOf(machineSaved{}).NumField(), "dos_equal": true, "chinese_equal": true, "cold_reload_next_steps": 100000})
		fmt.Printf("正常取樣 %d：原版三分支、完整state與載回接續相同\n", target)
		switch target {
		case 42000000:
			for _, w := range worlds {
				queue(w, 42500000, 3000000, 0x01, 0x50, 0x1c)
			}
		case 58000000:
			for _, w := range worlds {
				queue(w, 58500000, 3000000, 0x50, 0x1c)
			}
		case 72000000:
			for _, w := range worlds {
				queue(w, 73500000, 3000000, 0x48)
			}
		}
	}
	if worlds[0].o.IRQ1Delivered() != 20 {
		log.Fatal("十鍵IRQ不是20次")
	}
	jsonWrite(output+".json", map[string]any{"scope": "正常名字→道具→Forget it→Up，三分支13取樣及真正冷載接續；不是GUI或封包驗收", "go_version": runtime.Version(), "seed": "86AF", "seed_method": "固定原版state，未改值或重擲", "theme": dir, "entries": 27, "inputs_sha256": inputs, "samples": rows, "events": events, "irq1": 20, "full_machine_negative_controls": []string{"CPU", "RAM", "port"}, "limits": "HD圖面與中文須由獨立核對器驗收；其他sprite、GUI、DAT與完整HD交付未完成"})
}
