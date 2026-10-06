// ALLY #0 接入的固定起點正常名字輸入驗證；入口與限制見 docs/re/038 §34。
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
func loadPNG(p string) *image.NRGBA {
	f, e := os.Open(p)
	check(e)
	im, e := png.Decode(f)
	check(e)
	check(f.Close())
	out := image.NewNRGBA(im.Bounds())
	draw.Draw(out, out.Bounds(), im, im.Bounds().Min, draw.Src)
	return out
}
func writePNG(p string, im image.Image) {
	f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	check(e)
	check(png.Encode(f, im))
	check(f.Close())
}

func compose(o *oracle.Oracle, tr *translator.Translator, hd *theme.Theme) *image.NRGBA {
	w, h, rgb := o.ScreenRGB()
	im := image.NewNRGBA(image.Rect(0, 0, w*3, h*3))
	for y := 0; y < h*3; y++ {
		for x := 0; x < w*3; x++ {
			i, j := 3*((y/3)*w+x/3), 4*(y*w*3+x)
			copy(im.Pix[j:j+3], rgb[i:i+3])
			im.Pix[j+3] = 255
		}
	}
	over := image.NewNRGBA(im.Bounds())
	if hd.Draw(over.Pix, 3) {
		draw.Draw(im, im.Bounds(), over, image.Point{}, draw.Over)
	}
	clear(over.Pix)
	if tr != nil && tr.Layer.Draw(over.Pix, 3, tr.MissingGlyph) {
		draw.Draw(im, im.Bounds(), over, image.Point{}, draw.Over)
	}
	return im
}

func main() {
	dir := flag.String("theme", "", "含 ALLY #0 的本機主題")
	out := flag.String("out", "", "新的收據前綴，不覆寫")
	flag.Parse()
	if *dir == "" || *out == "" {
		log.Fatal("必須指定 -theme 與 -out")
	}
	for _, ext := range []string{".json", "-original.png", "-hd.png", "-background.png", "-end.state"} {
		if _, e := os.Lstat(*out + ext); !os.IsNotExist(e) {
			log.Fatal("拒絕覆寫 ", *out+ext)
		}
	}
	orig, start := "/orig/psychic-war", "workplace/states/06-name.state"
	if hash(read(start)) != "ac8eec0bbc533b267782298c298ce4a60440d0568266e665c8a5badbf9e0ca4a" {
		log.Fatal("起點雜湊改變")
	}
	inputs := map[string]string{}
	for _, p := range []string{start, "tools/hd/verify_ally_runtime.go", "apps/psychicwar/theme/ally.go", "apps/psychicwar/theme/theme.go", "apps/psychicwar/theme/theme_test.go", "cmd/psychicwar/main.go", "cmd/pwstep/main.go", "docs/spec/024-hd-theme.md", "worktrees/dosgolem/xlate/art.go", "worktrees/dosgolem/xlate/watch.go", "worktrees/dosgolem/xlate/layer.go", filepath.Join(orig, "ALLY.PBL"), filepath.Join(orig, "PW.EXE"), filepath.Join(*dir, "manifest.json"), filepath.Join(*dir, "ALLY-00.png")} {
		inputs[p] = hash(read(p))
	}
	files, e := filepath.Glob(filepath.Join(*dir, "*.png"))
	check(e)
	for _, p := range files {
		inputs[p] = hash(read(p))
	}
	for _, pattern := range []string{"text/*.json", "font/*.golemfnt"} {
		files, e = filepath.Glob(pattern)
		check(e)
		for _, p := range files {
			inputs[p] = hash(read(p))
		}
	}
	var mem, frame []byte
	var regs oracle.Regs
	var steps, cycles uint64
	results := []map[string]any{}
	for _, enabled := range []bool{false, true} {
		o, e := oracle.Load(filepath.Join(orig, "PW.EXE"), orig)
		check(e)
		check(o.LoadStateFile(start))
		o.SetScratch("/tmp/pw-ally-runtime")
		if o.Word(oracle.Addr{Seg: 0x161, Off: 0x41DF}) != 0x86AF || o.Steps() != 35000000 {
			log.Fatal("執行前種子或步數不符")
		}
		var hd *theme.Theme
		if enabled {
			var notice string
			hd, notice, e = theme.LoadTheme(*dir, orig, "theme", 3)
			check(e)
			if notice != "" {
				log.Fatal(notice)
			}
			check(hd.Attach(o))
			check(hd.Attach(o))
		}
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
		var hits []map[string]any
		o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x8705}, func(o *oracle.Oracle) {
			r := o.Regs()
			if r.AX&255 == 0 && r.CX == 0x4226 && r.DX == 0x0304 {
				hits = append(hits, map[string]any{"step": o.Steps(), "regs": r, "packed_sha256": hash(o.Bytes(oracle.Addr{Seg: r.DS, Off: r.BX}, 384))})
			}
		})
		runTo := func(target uint64) {
			for o.Steps() < target {
				n := target - o.Steps()
				if n > 50000 {
					n = 50000
				}
				check(o.Run(n))
				hd.Frame(o)
				tr.Frame(o)
			}
		}
		// 正常鍵盤事件；與歷史 probe 的節流佇列分開記錄，不冒稱逐指令時序相同。
		for i, scan := range []uint8{0x25, 0x1E, 0x17, 0x1C} {
			runTo(35500000 + uint64(i)*1000000)
			o.KeyDown(scan)
			runTo(36000000 + uint64(i)*1000000)
			o.KeyUp(scan)
		}
		runTo(42000001)
		m, f, r, st, cy := o.Bytes(oracle.Addr{}, 1<<20), append([]byte(nil), o.Indexed()...), o.Regs(), o.Steps(), o.Cycles()
		if !enabled {
			mem, frame, regs, steps, cycles = m, f, r, st, cy
			writePNG(*out+"-original.png", compose(o, tr, nil))
		} else {
			if !bytes.Equal(mem, m) || !bytes.Equal(frame, f) || regs != r || steps != st || cycles != cy {
				log.Fatal("HD 接入改變原版狀態")
			}
			_, _, px, e := pbl.Decode(read(filepath.Join(orig, "ALLY.PBL")), 0)
			check(e)
			actual, e := pbl.Region(f, 320, 200, 264, 152, 24, 32)
			check(e)
			if !bytes.Equal(actual, px) || len(hits) == 0 {
				log.Fatal("正常路徑未貼 ALLY #0")
			}
			bg, notice, e := theme.LoadTheme("workplace/hd/theme-v1-20261001", orig, "theme", 3)
			check(e)
			if notice != "" {
				log.Fatal(notice)
			}
			bg.Frame(o)
			before := o.Save()
			hd.Frame(o)
			if len(o.SearchChanged(before)) != 0 {
				log.Fatal("Frame 寫原版 RAM")
			}
			base, got := compose(o, nil, bg), compose(o, nil, hd)
			expect := image.NewNRGBA(base.Bounds())
			copy(expect.Pix, base.Pix)
			asset := loadPNG(filepath.Join(*dir, "ALLY-00.png"))
			// 獨立推導每個含原版墨跡的 8 格；不呼叫載入器取得期望圖面。
			for yy := 0; yy < 32; yy += 8 {
				for xx := 0; xx < 24; xx += 8 {
					ink := false
					for y := yy; y < yy+8; y++ {
						for x := xx; x < xx+8; x++ {
							if px[y*24+x] != 0 {
								ink = true
							}
						}
					}
					if ink {
						rect := image.Rect((264+xx)*3, (152+yy)*3, (264+xx+8)*3, (152+yy+8)*3)
						draw.Draw(expect, rect, asset, image.Pt(xx*3, yy*3), draw.Over)
					}
				}
			}
			if !bytes.Equal(expect.Pix, got.Pix) {
				log.Fatal("HD 與獨立資產／格線期望不同")
			}
			changed, outside := 0, 0
			for y := 0; y < 600; y++ {
				for x := 0; x < 960; x++ {
					j := 4 * (y*960 + x)
					if !bytes.Equal(got.Pix[j:j+4], base.Pix[j:j+4]) {
						changed++
						if x < 264*3 || x >= 288*3 || y < 152*3 || y >= 184*3 {
							outside++
						}
					}
				}
			}
			if changed == 0 || outside != 0 {
				log.Fatal("負對照無差異或更動角色區外")
			}
			full := compose(o, tr, hd)
			writePNG(*out+"-hd.png", full)
			writePNG(*out+"-background.png", compose(o, tr, bg))
			check(o.SaveStateFile(*out + "-end.state"))
			check(o.Run(100000))
			check(o.LoadStateFile(*out + "-end.state"))
			hd.ResetForLoad()
			hd.Frame(o)
			if !bytes.Equal(compose(o, nil, hd).Pix, got.Pix) {
				log.Fatal("實際 state 載回角色未恢復")
			}
			results = append(results, map[string]any{"independent_asset_mismatch": 0, "outside_changed_pixels": outside, "negative_omit_ally_pixels": changed, "state_reload": "相同", "full_ram_and_regs": "相同"})
		}
		results = append(results, map[string]any{"hd": enabled, "seed_before": "86AF", "seed_method": "執行前載入固定雜湊 state，唯讀核對；未寫 seed", "steps": st, "cycles": cy, "ram_sha256": hash(m), "indexed_sha256": hash(f), "regs": r, "normal_ally_blits": hits})
		o.Close()
	}
	doc := map[string]any{"go_version": runtime.Version(), "inputs_sha256": inputs, "results": results, "inputs": "35,500,000 起每 1,000,000 指令按 K/A/I/Enter；每鍵 500,000 指令後放開；至 42,000,001", "limits": "限 ALLY #0 技術接入；造型比例仍需美術審查，非全部 sprite、原版 DAT 或跨平台驗收"}
	b, e := json.MarshalIndent(doc, "", "  ")
	check(e)
	f, e := os.OpenFile(*out+".json", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(e)
	_, e = f.Write(append(b, '\n'))
	check(e)
	check(f.Close())
	fmt.Println("ALLY #0 正常名字輸入、完整狀態、獨立 PNG 格線、區外負對照與實際讀檔通過")
}
