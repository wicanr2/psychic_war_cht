// 第一批主題的實際載入、正常 F3 重播與遊戲狀態比較；docs/re/038 §33。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
	"time"

	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/dosgolem/xlate"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/theme"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/translator"
)

func main() {
	dir := flag.String("theme", "", "主題目錄")
	out := flag.String("out", "", "新的收據前綴，不覆寫")
	flag.Parse()
	if *dir == "" || *out == "" {
		log.Fatal("必須指定 -theme 與 -out")
	}
	for _, ext := range []string{".json", "-off.png", "-on.png", "-initial.png"} {
		if _, err := os.Lstat(*out + ext); !os.IsNotExist(err) {
			log.Fatal("拒絕覆寫：", *out+ext)
		}
	}
	orig := "/orig/psychic-war"
	start := "workplace/hd/redraw/replay-encounter.state"
	inputs := map[string]string{}
	for _, p := range []string{start, start + ".xlate.json", "tools/hd/verify_theme.go", "apps/psychicwar/theme/theme.go", "docs/spec/024-hd-theme.md",
		"worktrees/dosgolem/xlate/art.go", "worktrees/dosgolem/xlate/layer.go", "worktrees/dosgolem/xlate/watch.go", "font/cjk24.golemfnt", "font/cjk16.golemfnt",
		"workplace/hd/redraw/layout-v2-20261001-background.png", "workplace/hd/redraw/art-plane-v2-20261001-0-0-composed.png",
		filepath.Join(orig, "PW.EXE"), filepath.Join(orig, "SCREEN.PBL"), filepath.Join(orig, "MENU.PBL"), filepath.Join(*dir, "manifest.json")} {
		inputs[p] = hash(read(p))
	}
	files, err := filepath.Glob(filepath.Join(*dir, "*.png"))
	check(err)
	for _, p := range files {
		inputs[p] = hash(read(p))
	}
	textFiles, err := filepath.Glob("text/*.json")
	check(err)
	for _, p := range textFiles {
		inputs[p] = hash(read(p))
	}
	if inputs[start] != "782405fd3ca55e61fafd3daa294b299911f1e7984199b62e71ee79f410ccd2b8" {
		log.Fatal("正常起點已改變")
	}
	var results []map[string]any
	var expectedMem, expectedFrame []byte
	var expectedRegs oracle.Regs
	var expectedSteps, expectedCycles uint64
	var frameCost, drawCost time.Duration
	for _, enabled := range []bool{false, true} {
		o, err := oracle.Load(filepath.Join(orig, "PW.EXE"), orig)
		check(err)
		check(o.LoadStateFile(start))
		o.SetScratch("/tmp/pw-theme-verify")
		o.SetDOSBoxCycles(750)
		seed := o.Word(oracle.Addr{Seg: 0x0161, Off: 0x41DF})
		if seed != 0xCAF0 {
			log.Fatal("起點種子不符")
		}
		entries, err := translator.LoadText("text")
		check(err)
		f24, err := xlate.LoadFont("font/cjk24.golemfnt")
		check(err)
		f16, err := xlate.LoadFont("font/cjk16.golemfnt")
		check(err)
		tr := translator.NewTranslator(entries, f24, f16, 3, nil)
		check(tr.Layer.Restore(read(start+".xlate.json"), tr.Fonts()))
		tr.Attach(o)
		baked, err := translator.LoadBaked("text")
		check(err)
		tr.AttachBaked(baked, orig)
		selected := ""
		if enabled {
			selected = *dir
		}
		hd, notice, err := theme.LoadTheme(selected, orig, "theme", 3)
		check(err)
		if notice != "" {
			log.Fatal(notice)
		}
		before := o.Save()
		hd.Frame(o)
		tr.Frame(o)
		if len(o.SearchChanged(before)) != 0 {
			log.Fatal("顯示圖面改寫原版記憶體")
		}
		if enabled {
			initial := compose(o, tr, hd)
			f, err := os.Open("workplace/hd/redraw/art-plane-v2-20261001-0-0-composed.png")
			check(err)
			old, err := png.Decode(f)
			check(err)
			check(f.Close())
			oldRGBA := image.NewNRGBA(initial.Bounds())
			draw.Draw(oldRGBA, oldRGBA.Bounds(), old, image.Point{}, draw.Src)
			if !bytes.Equal(initial.Pix, oldRGBA.Pix) {
				log.Fatal("正式載入畫面與獨立已驗合成不同")
			}
			writePNG(*out+"-initial.png", initial)
		}
		acts, err := oracle.ParseActions("wait:500,wait:2000")
		check(err)
		acts = append([]oracle.Action{{Kind: oracle.ActionTap, Scan: 0x3D, MS: 150}}, acts...)
		check(o.RunActions(acts, 0, func() { hd.Frame(o); tr.Frame(o) }))
		mem, frame, regs, steps, cycles := o.Bytes(oracle.Addr{}, 1<<20), append([]byte(nil), o.Indexed()...), o.Regs(), o.Steps(), o.Cycles()
		if !enabled {
			expectedMem, expectedFrame, expectedRegs, expectedSteps, expectedCycles = mem, frame, regs, steps, cycles
		} else {
			if !bytes.Equal(expectedMem, mem) || !bytes.Equal(expectedFrame, frame) || expectedRegs != regs || expectedSteps != steps || expectedCycles != cycles {
				log.Fatal("開 HD 與原版的記憶體／畫面／暫存器／步數／cycles 不同")
			}
			// 再載入同一存檔並重登記，禁止沿用逃離戰鬥後的遮格。
			check(o.LoadStateFile(start))
			hd.ResetForLoad()
			hd.Frame(o)
			tr.ResetForLoad()
			check(tr.Layer.Restore(read(start+".xlate.json"), tr.Fonts()))
			tr.AttachBaked(baked, orig)
			tr.Frame(o)
			if !bytes.Equal(compose(o, tr, hd).Pix, readPNG(*out+"-initial.png").Pix) {
				log.Fatal("實際載回狀態後主題畫面未恢復")
			}
			// 成本採固定工作量；不宣稱高負載環境的即時幀率。
			buf := make([]byte, 960*600*4)
			now := time.Now()
			for n := 0; n < 100; n++ {
				hd.Frame(o)
			}
			frameCost = time.Since(now) / 100
			now = time.Now()
			for n := 0; n < 100; n++ {
				clear(buf)
				hd.Draw(buf, 3)
			}
			drawCost = time.Since(now) / 100
			// 還原已驗終點供輸出，不推進或重擲原版。
			check(o.LoadStateFile(start))
			hd.ResetForLoad()
			tr.ResetForLoad()
			check(tr.Layer.Restore(read(start+".xlate.json"), tr.Fonts()))
			tr.AttachBaked(baked, orig)
			check(o.RunActions(acts, 0, func() { hd.Frame(o); tr.Frame(o) }))
		}
		label := "off"
		if enabled {
			label = "on"
		}
		writePNG(*out+"-"+label+".png", compose(o, tr, hd))
		results = append(results, map[string]any{"hd": enabled, "seed_before": "CAF0", "seed_method": "先載入固定 SHA-256 的正常遭遇 state，執行前唯讀核對 0161:41DF；兩側輸入相同，不改 seed", "steps": steps, "cycles": cycles, "ram_sha256": hash(mem), "indexed_sha256": hash(frame), "regs": regs})
		o.Close()
	}
	doc := map[string]any{"go_version": runtime.Version(), "inputs_sha256": inputs, "results": results, "initial_comparison": "與 §32 獨立已驗合成逐位元組相同", "state_comparison": "完整 1 MiB 可讀記憶體、原版畫面、暫存器、指令數與 cycles 相同", "reload": "實際載回同一 state 後重登記及畫面比較通過", "theme_frame_ms": float64(frameCost) / float64(time.Millisecond), "theme_draw_ms": float64(drawCost) / float64(time.Millisecond), "limits": "第一批本機主題；未測原版 DAT 存檔、實際按鍵切換、全部 sprite 或跨平台幀率"}
	b, err := json.MarshalIndent(doc, "", "  ")
	check(err)
	check(os.WriteFile(*out+".json", append(b, '\n'), 0644))
	fmt.Println("正常 F3 重播、完整記憶體及主題載回驗證通過")
}

func compose(o *oracle.Oracle, tr *translator.Translator, hd *theme.Theme) *image.NRGBA {
	w, h, rgb := o.ScreenRGB()
	img := image.NewNRGBA(image.Rect(0, 0, w*3, h*3))
	for y := 0; y < h*3; y++ {
		for x := 0; x < w*3; x++ {
			i, j := 3*((y/3)*w+x/3), 4*(y*w*3+x)
			copy(img.Pix[j:j+3], rgb[i:i+3])
			img.Pix[j+3] = 255
		}
	}
	over := image.NewNRGBA(img.Bounds())
	if hd.Draw(over.Pix, 3) {
		draw.Draw(img, img.Bounds(), over, image.Point{}, draw.Over)
	}
	clear(over.Pix)
	if tr.Layer.Draw(over.Pix, 3, tr.MissingGlyph) {
		draw.Draw(img, img.Bounds(), over, image.Point{}, draw.Over)
	}
	return img
}
func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
func read(p string) []byte { b, e := os.ReadFile(p); check(e); return b }
func hash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func readPNG(p string) *image.NRGBA {
	f, e := os.Open(p)
	check(e)
	img, e := png.Decode(f)
	check(e)
	check(f.Close())
	out := image.NewNRGBA(img.Bounds())
	draw.Draw(out, out.Bounds(), img, img.Bounds().Min, draw.Src)
	return out
}
func writePNG(p string, img image.Image) {
	f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	check(e)
	check(png.Encode(f, img))
	check(f.Close())
}
