// ENEMY00 #3–#5 正常往返循環驗證；原版證據 §20／§36，範圍 024 §1.3／§1.4。
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
	addText(im, tr)
	return im
}
func addText(im *image.NRGBA, tr *translator.Translator) {
	if tr == nil {
		return
	}
	over := image.NewNRGBA(im.Bounds())
	if tr.Layer.Draw(over.Pix, 3, tr.MissingGlyph) {
		draw.Draw(im, im.Bounds(), over, image.Point{}, draw.Over)
	}
}
func different(a, b *image.NRGBA, inside bool) int {
	n := 0
	for y := 0; y < 600; y++ {
		for x := 0; x < 960; x++ {
			in := x >= 32*3 && x < 56*3 && y >= 152*3 && y < 184*3
			if in != inside {
				continue
			}
			i := 4 * (y*960 + x)
			if !bytes.Equal(a.Pix[i:i+4], b.Pix[i:i+4]) {
				n++
			}
		}
	}
	return n
}
func place(im *image.NRGBA, asset *image.NRGBA, original []byte) {
	placeMatching(im, asset, original, nil)
}

func placeMatching(im *image.NRGBA, asset *image.NRGBA, original, current []byte) {
	for yy := 0; yy < 32; yy += 8 {
		for xx := 0; xx < 24; xx += 8 {
			ink, match := false, true
			for y := yy; y < yy+8; y++ {
				for x := xx; x < xx+8; x++ {
					if original[(152+y)*320+32+x] != 0 {
						ink = true
					}
					if current != nil && current[(152+y)*320+32+x] != original[(152+y)*320+32+x] {
						match = false
					}
				}
			}
			if ink && match {
				r := image.Rect((32+xx)*3, (152+yy)*3, (32+xx+8)*3, (152+yy+8)*3)
				draw.Draw(im, r, asset, image.Pt(xx*3, yy*3), draw.Over)
			}
		}
	}
}

func main() {
	dir := flag.String("theme", "", "包含三個敵人動作的主題")
	out := flag.String("out", "", "新的唯一輸出前綴")
	flag.Parse()
	if *dir == "" || *out == "" {
		log.Fatal("必須指定 -theme、-out")
	}
	if matches, e := filepath.Glob(*out + "*"); e != nil || len(matches) != 0 {
		log.Fatal("拒絕覆寫輸出前綴")
	}
	orig, start := "/orig/psychic-war", "workplace/states/06-name.state"
	if hash(read(start)) != "ac8eec0bbc533b267782298c298ce4a60440d0568266e665c8a5badbf9e0ca4a" {
		log.Fatal("固定起點不符")
	}
	inputs := map[string]string{}
	paths := []string{start, "workplace/states/07-first-play.state", "tools/hd/verify_enemy_runtime.go", "docs/spec/024-hd-theme.md", "worktrees/dosgolem/xlate/art.go", "worktrees/dosgolem/xlate/layer.go", "worktrees/dosgolem/xlate/watch.go", filepath.Join(orig, "PW.EXE"), filepath.Join(orig, "ENEMY00.PBL"), filepath.Join(orig, "ALLY.PBL"), filepath.Join(*dir, "manifest.json"), "workplace/probe/hd-enemy-xor.json"}
	paths = append(paths, "tools/hd/observe_enemy_partial.go", "workplace/hd/enemy-partial-observation-20261001.json")
	for _, pattern := range []string{"apps/psychicwar/theme/*.go", "text/*.json", "font/*.golemfnt", filepath.Join(*dir, "*.png"), "workplace/probe/hd-xor-before*.frame", "workplace/probe/hd-xor-after*.frame", "workplace/probe/hd-xor-source*.bin"} {
		ps, e := filepath.Glob(pattern)
		check(e)
		paths = append(paths, ps...)
	}
	for _, p := range paths {
		inputs[p] = hash(read(p))
	}
	var baselineMem, baselineFrame []byte
	var baselineRegs oracle.Regs
	var baselineCycles uint64
	var baselinePhases []map[string]any
	var partialChecks []map[string]any
	var observation struct {
		Samples []struct {
			Step uint64 `json:"step"`
			SHA  string `json:"indexed_sha256"`
		} `json:"samples"`
	}
	check(json.Unmarshal(read("workplace/hd/enemy-partial-observation-20261001.json"), &observation))
	partialHashes := map[uint64]string{}
	for _, s := range observation.Samples {
		partialHashes[s.Step] = s.SHA
	}

	const cyclePrefix = "workplace/hd/enemy-cycle-observation-v1-20261001"
	var cycle struct {
		Events []struct {
			EntryStep  uint64 `json:"entry_step"`
			ReturnStep uint64 `json:"return_step"`
			From       []int  `json:"from_images"`
			To         []int  `json:"to_images"`
			AL         uint16 `json:"al"`
		} `json:"events"`
	}
	check(json.Unmarshal(read(cyclePrefix+".json"), &cycle))
	if len(cycle.Events) != 16 {
		log.Fatal("正常循環來源數不符")
	}
	for _, pattern := range []string{cyclePrefix + "*", "tools/hd/observe_enemy_cycle.go", "tools/hd/verify_enemy_cycle.py", "workplace/hd/enemy-cycle-independent-v1-20261001.json"} {
		ps, e := filepath.Glob(pattern)
		check(e)
		for _, p := range ps {
			inputs[p] = hash(read(p))
		}
	}
	partialEvents := map[uint64]int{}
	for _, sample := range observation.Samples {
		partialEvents[sample.Step] = 1
	}
	for _, n := range []int{2, 3, 4} {
		event := cycle.Events[n]
		for _, offset := range []uint64{1, 1000, 8000, 16000, 20000} {
			partialEvents[event.EntryStep+offset] = n
		}
		partialEvents[event.ReturnStep] = n
	}
	baselinePartial := map[uint64]string{}
	results := []map[string]any{}
	expectedEnter, expectedLeave := []uint64{}, []uint64{}
	for _, event := range cycle.Events {
		expectedEnter = append(expectedEnter, event.EntryStep)
		expectedLeave = append(expectedLeave, event.ReturnStep)
	}
	for _, enabled := range []bool{false, true} {
		o, e := oracle.Load(filepath.Join(orig, "PW.EXE"), orig)
		check(e)
		check(o.LoadStateFile(start))
		o.SetScratch("/tmp/pw-enemy-runtime")
		if o.Word(oracle.Addr{Seg: 0x161, Off: 0x41DF}) != 0x86AF {
			log.Fatal("執行前種子不符")
		}
		entries, e := translator.LoadText("text")
		check(e)
		f24, e := xlate.LoadFont("font/cjk24.golemfnt")
		check(e)
		f16, e := xlate.LoadFont("font/cjk16.golemfnt")
		check(e)
		tr := translator.NewTranslator(entries, f24, f16, 3, nil)
		tr.Attach(o)
		if b, e := os.ReadFile(start + ".xlate.json"); e == nil {
			check(tr.Layer.Restore(b, tr.Fonts()))
			inputs[start+".xlate.json"] = hash(b)
		} else if !os.IsNotExist(e) {
			check(e)
		}
		baked, e := translator.LoadBaked("text")
		check(e)
		tr.AttachBaked(baked, orig)
		var hd, bg *theme.Theme
		if enabled {
			var notice string
			hd, notice, e = theme.LoadTheme(*dir, orig, "theme", 3)
			check(e)
			if notice != "" {
				log.Fatal(notice)
			}
			check(hd.Attach(o))
			bg, notice, e = theme.LoadTheme("workplace/hd/theme-ally0-v1-20261001", orig, "theme", 3)
			check(e)
			if notice != "" {
				log.Fatal(notice)
			}
		}
		phases := []map[string]any{}
		pending := false
		var entry oracle.Regs
		o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x8705}, func(o *oracle.Oracle) {
			r := o.Regs()
			n := len(phases)
			if n >= len(cycle.Events) || r.CX != 0x0826 || r.DX != 0x0304 {
				return
			}
			if pending {
				log.Fatal("未完成前一筆貼圖")
			}
			entry = r
			pending = true
			if o.Steps() != expectedEnter[n] {
				log.Fatalf("來源事件 %d 步數不同 %d", n, o.Steps())
			}
			mode := cycle.Events[n].AL
			if r.AX&255 != mode {
				log.Fatal("已證實模式不符")
			}
			before := read(fmt.Sprintf("%s-event%02d-before.frame", cyclePrefix, n))
			source := read(fmt.Sprintf("%s-event%02d-source.bin", cyclePrefix, n))
			if !bytes.Equal(o.Indexed(), before) || !bytes.Equal(o.Bytes(oracle.Addr{Seg: r.DS, Off: r.BX}, 384), source) {
				log.Fatal("正常前畫面或來源與獨立已驗收據不同")
			}
		})
		o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x8751}, func(o *oracle.Oracle) {
			if !pending {
				return
			}
			pending = false
			n := len(phases)
			if o.Steps() != expectedLeave[n] {
				log.Fatal("返回步數不符")
			}
			original := append([]byte(nil), o.Indexed()...)
			expected := read(fmt.Sprintf("%s-event%02d-after.frame", cyclePrefix, n))
			if !bytes.Equal(original, expected) {
				log.Fatal("原版後畫面與獨立已驗收據不同")
			}
			phase := map[string]any{"image": cycle.Events[n].To[0], "entry_step": expectedEnter[n], "return_step": o.Steps(), "entry_regs": entry, "return_regs": o.Regs(), "cycles": o.Cycles(), "ram_sha256": hash(o.Bytes(oracle.Addr{}, 1<<20)), "indexed_sha256": hash(original)}
			before := o.Save()
			hd.Frame(o)
			tr.Frame(o)
			if len(o.SearchChanged(before)) != 0 {
				log.Fatal("顯示層寫原版 RAM")
			}
			if enabled {
				for _, key := range []string{"return_regs", "cycles", "ram_sha256", "indexed_sha256"} {
					a, _ := json.Marshal(phase[key])
					b, _ := json.Marshal(baselinePhases[n][key])
					if !bytes.Equal(a, b) {
						log.Fatal("動作狀態不符：", key)
					}
				}
				bg.Frame(o)
				base := compose(o, nil, bg)
				want := image.NewNRGBA(base.Bounds())
				copy(want.Pix, base.Pix)
				asset := loadPNG(filepath.Join(*dir, fmt.Sprintf("ENEMY00-%02d.png", cycle.Events[n].To[0])))
				place(want, asset, expected)
				addText(want, tr)
				got := compose(o, tr, hd)
				if !bytes.Equal(want.Pix, got.Pix) {
					log.Fatal("完整 HD 畫面與獨立 PNG／原版格線／中文期望不同")
				}
				baseText := compose(o, tr, bg)
				outside, changed := different(got, baseText, false), different(got, baseText, true)
				wrong := image.NewNRGBA(base.Bounds())
				copy(wrong.Pix, base.Pix)
				place(wrong, loadPNG(filepath.Join(*dir, fmt.Sprintf("ENEMY00-%02d.png", 3+(cycle.Events[n].To[0]-3+1)%3))), expected)
				addText(wrong, tr)
				negative := different(wrong, got, true)
				if outside != 0 || changed == 0 || negative == 0 {
					log.Fatal("區外更動或負對照無區辨力")
				}
				phase["asset_mismatch"] = 0
				phase["outside_changed_pixels"] = outside
				phase["negative_omit_enemy_pixels"] = changed
				phase["negative_wrong_pose_pixels"] = negative
				prefix := fmt.Sprintf("%s-event%02d-pose%d", *out, n, cycle.Events[n].To[0])
				writePNG(prefix+"-hd.png", got)
				writePNG(prefix+"-background.png", baseText)
				check(o.SaveStateFile(prefix + ".state"))
				snap, e := tr.Layer.Snapshot()
				check(e)
				check(os.WriteFile(prefix+".state.xlate.json", snap, 0644))
				hd.Enabled = false
				if !bytes.Equal(compose(o, tr, hd).Pix, compose(o, tr, nil).Pix) {
					log.Fatal("停用 HD 未回原版")
				}
				hd.Enabled = true
				if !bytes.Equal(compose(o, tr, hd).Pix, got.Pix) {
					log.Fatal("切回 HD 未恢復目前動作")
				}
			} else {
				writePNG(fmt.Sprintf("%s-event%02d-pose%d-original.png", *out, n, cycle.Events[n].To[0]), compose(o, tr, nil))
			}
			phases = append(phases, phase)
		})
		runTo := func(target uint64) {
			for o.Steps() < target {
				n := target - o.Steps()
				if n > 50000 {
					n = 50000
				}
				for step := range partialEvents {
					if step > o.Steps() && step < o.Steps()+n {
						n = step - o.Steps()
					}
				}
				check(o.Run(n))
				hd.Frame(o)
				tr.Frame(o)
				if eventIndex, sample := partialEvents[o.Steps()]; sample {
					expectedSHA := partialHashes[o.Steps()]
					idx := o.Indexed()
					if expectedSHA != "" && hash(idx) != expectedSHA {
						log.Fatal("正常中間原版畫面與唯讀觀察不同")
					}
					if !enabled {
						baselinePartial[o.Steps()] = hash(idx)
					} else if hash(idx) != baselinePartial[o.Steps()] {
						log.Fatal("HD 改變中途原版狀態")
					}
					if enabled {
						bg.Frame(o)
						base := compose(o, nil, bg)
						want := image.NewNRGBA(base.Bounds())
						copy(want.Pix, base.Pix)
						event := cycle.Events[eventIndex]
						ref := read(fmt.Sprintf("%s-event%02d-after.frame", cyclePrefix, eventIndex))
						placeMatching(want, loadPNG(filepath.Join(*dir, fmt.Sprintf("ENEMY00-%02d.png", event.To[0]))), ref, idx)
						addText(want, tr)
						wrong := image.NewNRGBA(base.Bounds())
						copy(wrong.Pix, base.Pix)
						placeMatching(wrong, loadPNG(filepath.Join(*dir, fmt.Sprintf("ENEMY00-%02d.png", event.From[0]))), ref, idx)
						addText(wrong, tr)
						got := compose(o, tr, hd)
						if !bytes.Equal(want.Pix, got.Pix) {
							log.Fatal("貼圖中途推翻新來源或未按完整 8 格遮擋")
						}
						partialChecks = append(partialChecks, map[string]any{"step": o.Steps(), "event": eventIndex, "target_image": event.To[0], "original_sha256": hash(idx), "hd_mismatch": 0, "negative_old_pose_pixels": different(wrong, got, true)})
					}
				}
			}
		}
		for i, scan := range []uint8{0x25, 0x1E, 0x17, 0x1C} {
			runTo(35500000 + uint64(i)*1000000)
			o.KeyDown(scan)
			runTo(36000000 + uint64(i)*1000000)
			o.KeyUp(scan)
		}
		runTo(42000001)
		// 不注入文字或狀態；從正常名字輸入接續，逐項核對與舊迷宮起點相同。
		checkpoint, e := oracle.Load(filepath.Join(orig, "PW.EXE"), orig)
		check(e)
		check(checkpoint.LoadStateFile("workplace/states/07-first-play.state"))
		check(checkpoint.Run(1))
		if !bytes.Equal(o.Bytes(oracle.Addr{}, 1<<20), checkpoint.Bytes(oracle.Addr{}, 1<<20)) || o.Regs() != checkpoint.Regs() ||
			!bytes.Equal(o.Indexed(), checkpoint.Indexed()) || o.Cycles() != checkpoint.Cycles() {
			log.Fatal("名字輸入未接到相同迷宮起點")
		}
		checkpoint.Close()
		// probe 十一鍵序列在 86,000,000 前的等價前綴；未達時間的後續鍵不作驗收。
		for i := 0; i < 6; i++ {
			runTo(43000000 + uint64(i)*8000000)
			o.KeyDown(0x48)
			target := 47000000 + uint64(i)*8000000
			if target > 86000000 {
				target = 86000000
			}
			runTo(target)
			if target < 86000000 {
				o.KeyUp(0x48)
			}
		}
		if len(phases) != len(cycle.Events) || o.Steps() != 86000000 {
			log.Fatal("正常路徑未完成 16 次往返動作")
		}
		mem, frame, regs, cycles := o.Bytes(oracle.Addr{}, 1<<20), append([]byte(nil), o.Indexed()...), o.Regs(), o.Cycles()
		if hash(frame) != "d0616fd57d1f7d078ea8abae989e6e901ff069f23890a0906e8cea845fb2e7d0" {
			log.Fatal("正常終點與既有原版收據不同")
		}
		if !enabled {
			baselineMem, baselineFrame, baselineRegs, baselineCycles = mem, frame, regs, cycles
			baselinePhases = phases
		} else {
			if !bytes.Equal(mem, baselineMem) || !bytes.Equal(frame, baselineFrame) || regs != baselineRegs || cycles != baselineCycles {
				log.Fatal("HD 接入改變原版完整狀態")
			}
			for n := 0; n < len(cycle.Events); n++ {
				prefix := fmt.Sprintf("%s-event%02d-pose%d", *out, n, cycle.Events[n].To[0])
				check(o.LoadStateFile(prefix + ".state"))
				hd.ResetForLoad()
				tr.ResetForLoad()
				check(tr.Layer.Restore(read(prefix+".state.xlate.json"), tr.Fonts()))
				tr.AttachBaked(baked, orig)
				hd.Frame(o)
				tr.Frame(o)
				if !bytes.Equal(compose(o, tr, hd).Pix, loadPNG(prefix+"-hd.png").Pix) {
					log.Fatal("載回目前動作後 HD 不同")
				}
			}
		}
		results = append(results, map[string]any{"hd": enabled, "seed_before": "86AF", "seed_method": "執行前固定雜湊 state 並唯讀核對；沒有寫 seed、重擲或挑選", "end_steps": 86000000, "end_cycles": cycles, "end_ram_sha256": hash(mem), "end_indexed_sha256": hash(frame), "end_regs": regs, "phases": phases})
		o.Close()
	}
	doc := map[string]any{"tool": "dosgolem f8c1a6e 與目前唯讀覆繪掛鉤", "go_version": runtime.Version(), "inputs_sha256": inputs, "results": results,
		"normal_name_prefix": "06-name 的正常 K/A/I/Enter；終點與 07-first-play 推進一指令的完整記憶體、暫存器、原版畫面與 cycles 相同；沒有注入覆繪答案",
		"input_prefix":       "名字按下 35,500,000 起，每 1,000,000 一鍵，各 500,000 後放開；前進按下 43,000,000 起，每 8,000,000 一鍵，各 4,000,000 後放開，至 86,000,000，未達時間的後續鍵不作驗收",
		"reload":             "16 個實際 state 載回及文字圖層還原後 HD 圖面完整相同", "limits": "ENEMY00 #3–#5 正常往返循環限定技術接入；候選造型與全部 sprite 仍待驗，非完整遊戲、GUI 同幀、幀率或發行包驗收"}
	doc["partial_checks"] = partialChecks
	b, e := json.MarshalIndent(doc, "", "  ")
	check(e)
	f, e := os.OpenFile(*out+".json", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	check(e)
	_, e = f.Write(append(b, '\n'))
	check(e)
	check(f.Close())
	fmt.Println("ENEMY00 #3–#5 正常 16 次往返動作、四方向中途遮格、完整狀態、切換與讀檔通過")
}
