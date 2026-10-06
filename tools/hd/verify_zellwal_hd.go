// 研究038 §91：既有原版正常按鍵的HD／中文、中途及state載回核對。
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
	"runtime"
	"sort"

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
func pngWrite(p string, b []byte) {
	f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(e)
	check(png.Encode(f, &image.NRGBA{Pix: b, Stride: 960 * 4, Rect: image.Rect(0, 0, 960, 600)}))
	check(f.Close())
}

type fingerprint struct {
	Steps, Cycles uint64
	Regs          oracle.Regs
	RAM, Frame    string
}

func fp(o *oracle.Oracle) fingerprint {
	return fingerprint{o.Steps(), o.Cycles(), o.Regs(), hash(o.Bytes(oracle.Addr{}, 1<<20)), hash(o.Indexed())}
}

type key struct {
	At   uint64
	Scan uint8
	Down bool
}
type event struct {
	Entry, Return                uint64
	Regs                         oracle.Regs
	X, Y, W, H                   int
	Before, After, Source, State string
}
type model struct {
	Inputs                           map[string]string `json:"inputs_sha256"`
	Initial, ObservedEnd, ControlEnd fingerprint
	Keys                             []key `json:"key_events"`
	Events                           []event
	Seed                             string `json:"seed_before"`
}
type sample struct {
	Step  uint64
	Event int
	Kind  string
}

func main() {
	out := flag.String("out", "", "全新輸出目錄")
	modelPath := flag.String("model", "", "§90原版正常收據")
	themePath := flag.String("theme", "workplace/hd/theme-zellwal-candidate-v1-20261003", "限定來源的候選主題目錄")
	flag.Parse()
	if *out == "" || *modelPath == "" {
		log.Fatal("缺輸出或原版收據")
	}
	if _, e := os.Stat(*out); e == nil {
		log.Fatal("拒絕覆寫")
	}
	check(os.Mkdir(*out, 0755))
	const orig = "/orig/psychic-war"
	dir := *themePath
	const start = "workplace/hd/zellwal-sprite-route-v1-20261003/advance-east/observed-end.state"
	var m model
	check(json.Unmarshal(read(*modelPath), &m))
	// 原版JSON使用snake_case命名兩個終點；分開讀取避免零值形成假對照。
	var ends struct {
		A fingerprint `json:"observed_end"`
		B fingerprint `json:"control_end"`
	}
	check(json.Unmarshal(read(*modelPath), &ends))
	m.ObservedEnd, m.ControlEnd = ends.A, ends.B
	if m.ObservedEnd != m.ControlEnd || m.Seed != "F95B" || len(m.Events) < 5 {
		log.Fatal("原版基準不符")
	}
	inputs := map[string]string{*modelPath: hash(read(*modelPath))}
	for name, want := range m.Inputs {
		if hash(read(name)) != want {
			log.Fatal("原版來源變更：", name)
		}
		inputs[name] = want
	}
	for _, name := range []string{"tools/hd/verify_zellwal_hd.go", dir + "/manifest.json", start} {
		inputs[name] = hash(read(name))
	}
	for _, pattern := range []string{dir + "/*.png", "text/*.json", "font/*.golemfnt", "apps/psychicwar/theme/*.go", "apps/psychicwar/translator/*.go"} {
		paths, e := filepath.Glob(pattern)
		check(e)
		for _, p := range paths {
			inputs[p] = hash(read(p))
		}
	}
	samples := []sample{{m.Initial.Steps, -1, "initial"}}
	for i, e := range m.Events {
		if e.X == 32 {
			samples = append(samples, sample{e.Entry + 1, i, "partial1"}, sample{e.Entry + 512, i, "partial512"})
		}
		samples = append(samples, sample{e.Return, i, "complete"})
	}
	samples = append(samples, sample{m.ObservedEnd.Steps, -1, "end"})
	sort.Slice(samples, func(i, j int) bool { return samples[i].Step < samples[j].Step })
	create := func() *oracle.Oracle {
		o, e := oracle.Load(orig+"/PW.EXE", orig)
		check(e)
		check(o.LoadStateFile(start))
		o.SetScratch("/tmp/pw-zellwal-hd")
		if fp(o) != m.Initial || fmt.Sprintf("%04X", o.Word(oracle.Addr{Seg: 0x161, Off: 0x41df})) != m.Seed {
			log.Fatal("原版起始不同")
		}
		return o
	}
	all := []map[string]any{}
	for _, enabled := range []bool{false, true} {
		o := create()
		entries, e := translator.LoadText("text")
		check(e)
		f24, e := xlate.LoadFont("font/cjk24.golemfnt")
		check(e)
		f16, e := xlate.LoadFont("font/cjk16.golemfnt")
		check(e)
		tr := translator.NewTranslator(entries, f24, f16, 3, nil)
		tr.Attach(o)
		baked, e := translator.LoadBaked("text")
		check(e)
		tr.AttachBaked(baked, orig)
		hd, notice, e := theme.LoadTheme(dir, orig, "theme", 3)
		check(e)
		if hd == nil || notice != "" {
			log.Fatal("候選主題未載入")
		}
		hd.Enabled = enabled
		check(hd.Attach(o))
		frames := []map[string]any{}
		tick := func() {
			if enabled {
				p := filepath.Join(*out, fmt.Sprintf("frame%03d.frame", len(frames)))
				write(p, o.Indexed())
				frames = append(frames, map[string]any{"step": o.Steps(), "frame": p})
			}
			hd.Frame(o)
			tr.Frame(o)
		}
		tick()
		blits := []map[string]any{}
		o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x8705}, func(o *oracle.Oracle) {
			r := o.Regs()
			x, y, w, h := int(r.CX>>8)*4, int(r.CX&255)*4, int(r.DX>>8)*8, int(r.DX&255)*8
			// 所有覆蓋完整敵人或盟友slot的貼圖都記錄，避免忽略清除或背景回填。
			if !((x <= 32 && y <= 152 && x+w >= 56 && y+h >= 184) || (x <= 264 && y <= 152 && x+w >= 288 && y+h >= 184)) {
				return
			}
			a := oracle.Addr{Seg: r.DS, Off: r.BX}
			raw := []byte(nil)
			if a.Linear() <= 0xa0000-384 {
				raw = o.Bytes(a, 384)
			}
			if enabled {
				p := filepath.Join(*out, fmt.Sprintf("blit%02d", len(blits)))
				write(p+"-before.frame", o.Indexed())
				write(p+"-source.bin", raw)
				blits = append(blits, map[string]any{"step": o.Steps(), "regs": r, "before": p + "-before.frame", "source": p + "-source.bin"})
			}
		})
		o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x8751}, func(o *oracle.Oracle) {
			if enabled && len(blits) > 0 {
				last := blits[len(blits)-1]
				if _, ok := last["return"]; !ok {
					last["return"] = o.Steps()
				}
			}
		})
		ki := 0
		rows := []map[string]any{}
		for i, s := range samples {
			for o.Steps() < s.Step {
				next := s.Step
				if ki < len(m.Keys) && m.Keys[ki].At < next {
					next = m.Keys[ki].At
				}
				for o.Steps() < next {
					n := next - o.Steps()
					if n > 100000 {
						n = 100000
					}
					check(o.Run(n))
					tick()
				}
				if ki < len(m.Keys) && m.Keys[ki].At == o.Steps() {
					k := m.Keys[ki]
					if k.Down {
						o.KeyDown(k.Scan)
					} else {
						o.KeyUp(k.Scan)
					}
					ki++
				}
			}
			tick()
			frame := o.Indexed()
			p := filepath.Join(*out, fmt.Sprintf("sample%02d", i))
			plane, text := make([]byte, 960*600*4), make([]byte, 960*600*4)
			hd.Draw(plane, 3)
			tr.Layer.Draw(text, 3, tr.MissingGlyph)
			if s.Kind == "complete" && !bytes.Equal(frame, read(m.Events[s.Event].After)) {
				log.Fatal("正常原版完整樣本不同")
			}
			if s.Kind == "end" && hash(frame) != m.ObservedEnd.Frame {
				log.Fatal("原版終點畫面不同")
			}
			if !enabled {
				write(p+".frame", frame)
				pngWrite(p+"-text.png", text)
				if !bytes.Equal(plane, make([]byte, len(plane))) {
					log.Fatal("關閉HD仍繪圖")
				}
			}
			if enabled {
				if !bytes.Equal(frame, read(p+".frame")) {
					log.Fatal("HD改變原版樣本")
				}
				f, e := os.Open(p + "-text.png")
				check(e)
				im, e := png.Decode(f)
				check(e)
				check(f.Close())
				same := image.NewNRGBA(im.Bounds())
				draw.Draw(same, same.Bounds(), im, im.Bounds().Min, draw.Src)
				if !bytes.Equal(same.Pix, text) {
					log.Fatal("HD改變中文圖面")
				}
				pngWrite(p+"-plane.png", plane)
				_, _, rgb := o.ScreenRGB()
				write(p+"-rgb.bin", rgb)
				composed := image.NewNRGBA(image.Rect(0, 0, 960, 600))
				for yy := 0; yy < 600; yy++ {
					for xx := 0; xx < 960; xx++ {
						at := 4 * (yy*960 + xx)
						src := 3 * ((yy/3)*320 + xx/3)
						copy(composed.Pix[at:at+3], rgb[src:src+3])
						composed.Pix[at+3] = 255
					}
				}
				draw.Draw(composed, composed.Bounds(), &image.NRGBA{Pix: plane, Stride: 3840, Rect: composed.Bounds()}, image.Point{}, draw.Over)
				draw.Draw(composed, composed.Bounds(), &image.NRGBA{Pix: text, Stride: 3840, Rect: composed.Bounds()}, image.Point{}, draw.Over)
				pngWrite(p+"-hd-chinese.png", composed.Pix)
				check(o.SaveStateFile(p + ".state"))
				side, e := tr.Layer.Snapshot()
				check(e)
				write(p+".state.xlate.json", side)
				// 顯示開關不送入原版，切回必須保持當前圖面。
				hd.Enabled = false
				off := make([]byte, len(plane))
				if hd.Draw(off, 3) || !bytes.Equal(off, make([]byte, len(off))) {
					log.Fatal("HD關閉輸出不空")
				}
				hd.Enabled = true
				on := make([]byte, len(plane))
				hd.Draw(on, 3)
				if !bytes.Equal(plane, on) {
					log.Fatal("HD切回不同")
				}
			}
			rows = append(rows, map[string]any{"sample": i, "step": s.Step, "event": s.Event, "kind": s.Kind, "prefix": p, "frame_sha256": hash(frame), "plane_sha256": hash(plane), "text_sha256": hash(text)})
		}
		if ki != len(m.Keys) || fp(o) != m.ObservedEnd {
			log.Fatal("正常按鍵或原版終點不同")
		}
		all = append(all, map[string]any{"hd": enabled, "end": fp(o), "samples": rows, "blits": blits, "frames": frames})
		o.Close()
	}
	// 真正載回每份正常／中途state。這條路徑清除顯示身份，不能要求受遮角色與連續觀察相同。
	reloads := []map[string]any{}
	o := create()
	defer o.Close()
	hd, _, e := theme.LoadTheme(dir, orig, "theme", 3)
	check(e)
	check(hd.Attach(o))
	base := create()
	defer base.Close()
	for i, s := range samples {
		p := filepath.Join(*out, fmt.Sprintf("sample%02d", i))
		check(o.LoadStateFile(p + ".state"))
		check(base.LoadStateFile(p + ".state"))
		if !bytes.Equal(o.Indexed(), read(p+".frame")) {
			log.Fatal("載回原版frame不同")
		}
		hd.ResetForLoad()
		hd.Enabled = true
		hd.Frame(o)
		plane := make([]byte, 960*600*4)
		hd.Draw(plane, 3)
		pngWrite(p+"-reload-plane.png", plane)
		check(o.Run(100000))
		check(base.Run(100000))
		if fp(o) != fp(base) {
			log.Fatal("HD干擾載回接續")
		}
		reloads = append(reloads, map[string]any{"sample": i, "kind": s.Kind, "prefix": p, "continuation_steps": 100000, "end": fp(o), "reload_plane_sha256": hash(plane)})
	}
	jsonWrite(filepath.Join(*out, "runtime.json"), map[string]any{"scope": "normal Zellwal original keys with candidate HD and new Chinese output; partial state reloads", "go_version": runtime.Version(), "inputs_sha256": inputs, "model": *modelPath, "theme": dir, "seed_before": m.Seed, "key_events": m.Keys, "results": all, "reloads": reloads, "limits": "cold Chinese overlay only newly printed strings, not fromboot/DAT/GUI/art/full-overlap acceptance; reloads independently checked, not assumed identical to continuous display"})
	fmt.Printf("Zellwal正常%d樣本，HD兩側原版／中文一致，%d份state載回及100000步接續一致\n", len(samples), len(reloads))
}
