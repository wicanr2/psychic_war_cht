// 正常方向鍵、等待、F3；原版FIFO與Oracle實際送鍵對拍，入口研究038 §49。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"reflect"

	"github.com/wicanr2/dosgolem/internal/cpu"
	"github.com/wicanr2/dosgolem/internal/dos"
	"github.com/wicanr2/dosgolem/internal/machine"
	"github.com/wicanr2/dosgolem/internal/state"
	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/dosgolem/xlate"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
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
	f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
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
func pngWrite(p string, pixels []byte) {
	f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	check(e)
	check(png.Encode(f, &image.NRGBA{Pix: pixels, Stride: 960 * 4, Rect: image.Rect(0, 0, 960, 600)}))
	check(f.Close())
}
func regs(c *cpu.CPU) oracle.Regs {
	return oracle.Regs{AX: c.R[cpu.AX], BX: c.R[cpu.BX], CX: c.R[cpu.CX], DX: c.R[cpu.DX], SI: c.R[cpu.SI], DI: c.R[cpu.DI], BP: c.R[cpu.BP], SP: c.R[cpu.SP], DS: c.Seg[cpu.DS], ES: c.Seg[cpu.ES], SS: c.Seg[cpu.SS], CS: c.Seg[cpu.CS], IP: c.IP, Flags: c.Flags}
}

type fingerprint struct {
	Regs                oracle.Regs
	Steps, Cycles, IRQ1 uint64
	RAM, Frame          string
}

func machineFP(m *machine.Machine) fingerprint {
	return fingerprint{regs(m.CPU), m.Steps, m.CPU.Cycles, m.IRQ1Delivered(), hash(m.Mem[:0xa0000]), hash(m.Indexed())}
}
func oracleFP(o *oracle.Oracle) fingerprint {
	return fingerprint{o.Regs(), o.Steps(), o.Cycles(), o.IRQ1Delivered(), hash(o.Bytes(oracle.Addr{}, 0xa0000)), hash(o.Indexed())}
}

type keyEvent struct {
	Step uint64
	Code uint8
}
type bodyEvent struct {
	Entry, Return         uint64
	Regs                  oracle.Regs
	Before, After, Source string
	To                    []int
}
type sample struct {
	Step  uint64
	Name  string
	Event int
	Kind  string
}

func main() {
	label := flag.String("label", "", "kasuruji 或minton")
	out := flag.String("out", "", "全新輸出前綴")
	initialXlate := flag.String("initial-xlate", "", "正常重播產生的起始中文Layer；不提供時只驗新印字")
	flag.Parse()
	if *out == "" {
		log.Fatal("缺輸出")
	}
	found, e := filepath.Glob(*out + "*")
	check(e)
	if len(found) != 0 {
		log.Fatal("拒絕覆寫")
	}
	const orig = "/orig/psychic-war"
	const dir = "workplace/hd/theme-next-bodies-v1-20261001"
	from, name, startHash, seedWant, at := "10-saved", "12-battle2", "abdbb98f66010d5680d3f3bfecead2c28ffd4f7f6fd625fc3b3103a62bcbd056", uint16(0xa71d), uint64(242000000)
	if *label == "minton" {
		from, name, startHash, seedWant, at = "13-healed", "14-minton1", "d9c18d64a60ed678d34d72f00c199af51108c2a36931fdcaed650a110f23e7a0", 0xa48c, 386200000
	} else if *label != "kasuruji" {
		log.Fatal("未知正常路線")
	}
	start := "workplace/states/" + from + ".state"
	stop := at + 3000000
	inputs := map[string]string{}
	if *initialXlate != "" {
		inputs[*initialXlate] = hash(read(*initialXlate))
	}
	for _, p := range []string{start, "replay/title-to-first-save.json", orig + "/PW.EXE", orig + "/ENEMY00.PBL", "tools/hd/verify_next_body_normal.go", dir + "/manifest.json"} {
		inputs[p] = hash(read(p))
	}
	if inputs[start] != startHash || inputs[orig+"/PW.EXE"] != "88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49" || inputs[orig+"/ENEMY00.PBL"] != "8241b0ea73b1e7402b13adc434f10a5b1288f88f01fcfc3ba5710e4923fe6067" {
		log.Fatal("固定來源不符")
	}
	for _, pattern := range []string{"text/*.json", "font/*.golemfnt", dir + "/*.png", "apps/psychicwar/theme/*.go", "apps/psychicwar/translator/*.go"} {
		paths, e := filepath.Glob(pattern)
		check(e)
		for _, p := range paths {
			inputs[p] = hash(read(p))
		}
	}
	var plan struct {
		Segments []struct {
			Name, From string
			Keys       []string
			KeyAt      uint64 `json:"key_at"`
			KeyEvery   uint64 `json:"key_every"`
		}
	}
	check(json.Unmarshal(read("replay/title-to-first-save.json"), &plan))
	var codes []uint8
	var keyAt, keyEvery uint64
	scans := map[string]uint8{"up": 0x48, "right": 0x4d, "down": 0x50, "left": 0x4b}
	m := machine.New()
	d := dos.New(m, orig)
	d.Install()
	defer d.Close()
	for _, s := range plan.Segments {
		if s.Name == name {
			if s.From != from || s.KeyEvery != 3000000 {
				log.Fatal("正常路線不符")
			}
			keyAt, keyEvery = s.KeyAt, s.KeyEvery
			m.KeyEvery = keyEvery
			m.SetNextKey(keyAt)
			for _, k := range s.Keys {
				sc, ok := scans[k]
				if !ok {
					log.Fatal("未知方向鍵")
				}
				m.QueueKey(sc)
				codes = append(codes, sc, sc|128)
			}
		}
	}
	if len(codes) == 0 {
		log.Fatal("無正常方向鍵")
	}
	codes = append(codes, 0x3d, 0xbd)
	check(state.Load(start, m, d))
	d.Root = orig
	d.Scratch = "/tmp/pw-next-normal-model"
	m.HoldKey(0x3d, at, 500000, false)
	seed := m.Read16(0x1610 + 0x41df)
	if seed != seedWant {
		log.Fatal("執行前seed不符")
	}
	poses := map[int][]byte{}
	source := read(orig + "/ENEMY00.PBL")
	for id := 0; id < 9; id++ {
		_, _, px, e := pbl.Decode(source, id)
		check(e)
		poses[id] = px
	}
	matches := func(frame []byte) []int {
		px, e := pbl.Region(frame, 320, 200, 32, 152, 24, 32)
		check(e)
		ids := []int{}
		for id := 0; id < 9; id++ {
			if bytes.Equal(px, poses[id]) {
				ids = append(ids, id)
			}
		}
		return ids
	}
	tape := []keyEvent{}
	events := []bodyEvent{}
	samples := []sample{}
	var pending *bodyEvent
	var pendingFrame, raw []byte
	escape := 0
	for m.Steps < stop {
		c := m.CPU
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x47ad {
			escape++
		}
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x8705 && c.R[cpu.CX] == 0x0826 && c.R[cpu.DX] == 0x0304 && len(events) < 16 {
			if pending != nil {
				log.Fatal("巢狀貼圖")
			}
			pending = &bodyEvent{Entry: m.Steps, Regs: regs(c)}
			pendingFrame = m.Indexed()
			addr := uint32(c.Seg[cpu.DS])*16 + uint32(c.R[cpu.BX])
			if addr > 0xa0000-384 {
				log.Fatal("來源越界")
			}
			raw = append([]byte(nil), m.Mem[addr:addr+384]...)
		}
		if pending != nil && len(events) >= 1 && len(events) <= 4 && (m.Steps == pending.Entry+1 || m.Steps == pending.Entry+512) {
			p := fmt.Sprintf("%s-partial%02d-%d", *out, len(events), m.Steps-pending.Entry)
			write(p+".frame", m.Indexed())
			samples = append(samples, sample{m.Steps, p, len(events), "partial"})
		}
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x8751 && pending != nil {
			i := len(events)
			p := fmt.Sprintf("%s-event%02d", *out, i)
			after := m.Indexed()
			pending.Return = m.Steps
			pending.Before = p + "-before.frame"
			pending.After = p + ".frame"
			pending.Source = p + "-source.bin"
			pending.To = matches(after)
			write(pending.Before, pendingFrame)
			write(pending.After, after)
			write(pending.Source, raw)
			events = append(events, *pending)
			samples = append(samples, sample{m.Steps, p, i, "complete"})
			pending = nil
		}
		oldIRQ := m.IRQ1Delivered()
		step := m.Steps
		check(m.Step())
		if m.IRQ1Delivered() != oldIRQ {
			if len(tape) >= len(codes) {
				log.Fatal("多餘IRQ1")
			}
			tape = append(tape, keyEvent{step, codes[len(tape)]})
		}
	}
	if len(tape) != len(codes) || len(events) < 5 || pending != nil || escape > 1 {
		log.Fatalf("正常路徑未閉合：鍵%d/%d、身體%d、返回%d", len(tape), len(codes), len(events), escape)
	}
	model := machineFP(m)
	p := *out + "-end"
	write(p+".frame", m.Indexed())
	samples = append(samples, sample{stop, p, -1, "end"})
	expectedEnd := matches(m.Indexed())
	if escape == 1 && len(expectedEnd) != 0 {
		log.Fatal("脫離後仍是敵人完整圖")
	}
	results := []map[string]any{}
	var baselineRAM string
	for _, enabled := range []bool{false, true} {
		o, e := oracle.Load(orig+"/PW.EXE", orig)
		check(e)
		check(o.LoadStateFile(start))
		o.SetScratch("/tmp/pw-next-normal-oracle")
		if o.Word(oracle.Addr{Seg: 0x161, Off: 0x41df}) != seed {
			log.Fatal("Oracle seed不同")
		}
		entries, e := translator.LoadText("text")
		check(e)
		f24, e := xlate.LoadFont("font/cjk24.golemfnt")
		check(e)
		f16, e := xlate.LoadFont("font/cjk16.golemfnt")
		check(e)
		tr := translator.NewTranslator(entries, f24, f16, 3, nil)
		tr.Attach(o)
		if *initialXlate != "" {
			check(tr.Layer.Restore(read(*initialXlate), tr.Fonts()))
		}
		baked, e := translator.LoadBaked("text")
		check(e)
		tr.AttachBaked(baked, orig)
		hd, notice, e := theme.LoadTheme(dir, orig, "theme", 3)
		check(e)
		if notice != "" {
			log.Fatal(notice)
		}
		hd.Enabled = enabled
		check(hd.Attach(o))
		si, ki := 0, 0
		phaseRows := []map[string]any{}
		hd.Frame(o)
		tr.Frame(o)
		for si < len(samples) || ki < len(tape) {
			next := stop
			if si < len(samples) && samples[si].Step < next {
				next = samples[si].Step
			}
			if ki < len(tape) && tape[ki].Step < next {
				next = tape[ki].Step
			}
			for o.Steps() < next {
				n := next - o.Steps()
				if n > 100000 {
					n = 100000
				}
				check(o.Run(n))
				hd.Frame(o)
				tr.Frame(o)
			}
			if ki < len(tape) && tape[ki].Step == o.Steps() {
				k := tape[ki]
				if k.Code&128 == 0 {
					o.KeyDown(k.Code)
				} else {
					o.KeyUp(k.Code & 127)
				}
				ki++
			}
			if si < len(samples) && samples[si].Step == o.Steps() {
				s := samples[si]
				if !bytes.Equal(o.Indexed(), read(s.Name+".frame")) {
					log.Fatal("正常Oracle取樣與FIFO原版不同：", s.Name)
				}
				hd.Frame(o)
				tr.Frame(o)
				plane, text := make([]byte, 960*600*4), make([]byte, 960*600*4)
				hd.Draw(plane, 3)
				tr.Layer.Draw(text, 3, tr.MissingGlyph)
				textName := s.Name + "-text.png"
				if !enabled {
					pngWrite(textName, text)
				} else {
					f, e := os.Open(textName)
					check(e)
					im, e := png.Decode(f)
					check(e)
					check(f.Close())
					same := image.NewNRGBA(im.Bounds())
					draw.Draw(same, same.Bounds(), im, im.Bounds().Min, draw.Src)
					if !bytes.Equal(same.Pix, text) {
						log.Fatal("HD改變中文層")
					}
				}
				if enabled {
					pngWrite(s.Name+"-plane.png", plane)
					_, _, rgb := o.ScreenRGB()
					write(s.Name+"-rgb.bin", rgb)
					composed := image.NewNRGBA(image.Rect(0, 0, 960, 600))
					for y := 0; y < 600; y++ {
						for x := 0; x < 960; x++ {
							i, j := 3*((y/3)*320+x/3), 4*(y*960+x)
							copy(composed.Pix[j:j+3], rgb[i:i+3])
							composed.Pix[j+3] = 255
						}
					}
					draw.Draw(composed, composed.Bounds(), &image.NRGBA{Pix: plane, Stride: 960 * 4, Rect: composed.Bounds()}, image.Point{}, draw.Over)
					draw.Draw(composed, composed.Bounds(), &image.NRGBA{Pix: text, Stride: 960 * 4, Rect: composed.Bounds()}, image.Point{}, draw.Over)
					pngWrite(s.Name+"-hd-chinese.png", composed.Pix)
					check(o.SaveStateFile(s.Name + ".state"))
					side, e := tr.Layer.Snapshot()
					check(e)
					write(s.Name+".state.xlate.json", side)
				}
				phaseRows = append(phaseRows, map[string]any{"sample": s, "original_frame_sha256": hash(o.Indexed()), "text_plane_sha256": hash(text), "hd_plane_sha256": hash(plane)})
				si++
			}
		}
		end := oracleFP(o)
		if !reflect.DeepEqual(end, model) {
			log.Fatalf("FIFO與Oracle終點不同：model=%+v oracle=%+v", model, end)
		}
		ram := hash(o.Bytes(oracle.Addr{}, 1<<20))
		if enabled && ram != baselineRAM {
			log.Fatal("HD改變完整匯流排終點")
		}
		if !enabled {
			baselineRAM = ram
		}
		results = append(results, map[string]any{"hd": enabled, "end": end, "ram_bus_sha256": ram, "phases": phaseRows})
		o.Close()
	}
	jsonWrite(*out+".json", map[string]any{"tool": "dosgolem f8c1a6e；研究用FIFO與正式Oracle＋現行HD／中文層", "inputs_sha256": inputs, "initial_xlate": *initialXlate, "start": start, "seed_before": fmt.Sprintf("%04X", seed), "seed_method": "執行前固定原版state雜湊並唯讀核對；無重擲或記憶體注入", "route_name": name, "key_at": keyAt, "key_every": keyEvery, "keys_irq1": tape, "f3_start": at, "f3_duration": 500000, "end_step": stop, "events": events, "samples": samples, "model_end": model, "escape_returns": escape, "results": results, "limits": "正常迷宮到等待／F3輸入的限定抽測，返回次數依原版實測；非GUI或完整遊戲，美術及效果仍待驗"})
	fmt.Printf("%s：正常%d次身體貼圖、%d個中途樣本、F3返回%d次；HD兩側原版終點一致\n", *label, len(events), len(samples)-len(events)-1, escape)
}
