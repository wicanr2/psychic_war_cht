// 獨立核對實際視窗的原座標人物／框線、語言切換及快速存讀檔；研究 §33。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
)

func main() {
	out := flag.String("out", "", "GUI 擷取目錄")
	allyPath := flag.String("ally", "", "額外核對 ALLY #0 PNG")
	bin := flag.String("bin", "workplace/hd/psychicwar-theme-v1", "實際擷取的執行檔")
	flag.Parse()
	if *out == "" {
		log.Fatal("必須指定 -out")
	}
	receipt := filepath.Join(*out, "verified.json")
	if _, e := os.Lstat(receipt); !os.IsNotExist(e) {
		log.Fatal("拒絕覆寫收據")
	}
	hd := loadPNG("workplace/hd/redraw/layout-v2-20261001-background.png")
	base := read("workplace/hd/bg.idx")
	if len(base) != 320*200 {
		log.Fatal("原版背景長度不符")
	}
	inputs := map[string]string{}
	for _, p := range []string{"workplace/hd/bg.idx", "workplace/hd/redraw/layout-v2-20261001-background.png", "tools/hd/frontend_check.sh", "tools/hd/verify_frontend.go", *bin, "cmd/psychicwar/main.go", "apps/psychicwar/theme/theme.go", "docs/spec/024-hd-theme.md", "text/help.json", "font/cjk24.golemfnt", "font/cjk16.golemfnt"} {
		inputs[p] = hash(read(p))
	}
	var ally *image.NRGBA
	var allyPixels []byte
	if *allyPath != "" {
		f, e := os.Open(*allyPath)
		check(e)
		im, e := png.Decode(f)
		check(e)
		check(f.Close())
		if im.Bounds().Dx() != 72 || im.Bounds().Dy() != 96 {
			log.Fatal("ALLY PNG 尺寸不符")
		}
		ally = image.NewNRGBA(im.Bounds())
		draw.Draw(ally, ally.Bounds(), im, im.Bounds().Min, draw.Src)
		inputs[*allyPath] = hash(read(*allyPath))
		path := "/orig/psychic-war/ALLY.PBL"
		inputs[path] = hash(read(path))
		_, _, allyPixels, e = pbl.Decode(read(path), 0)
		check(e)
	}
	var observed []byte
	var indexed [2][]byte
	var originals, expectedHD [2]*image.NRGBA
	for n, name := range []string{"before.state", "after.state"} {
		p := filepath.Join(*out, name)
		inputs[p] = hash(read(p))
		o, e := oracle.Load("/orig/psychic-war/PW.EXE", "/orig/psychic-war")
		check(e)
		check(o.LoadStateFile(p))
		got := o.Bytes(oracle.Addr{Seg: 0x1696, Off: 6}, 52)
		if observed == nil {
			observed = got
		} else if !bytes.Equal(observed, got) {
			log.Fatal("讀檔後區域／座標／角色數值不同")
		}
		indexed[n] = append([]byte(nil), o.Indexed()...)
		w, h, rgb := o.ScreenRGB()
		if w != 320 || h != 200 {
			log.Fatal("原版畫面尺寸不符")
		}
		img := image.NewNRGBA(hd.Bounds())
		for y := 0; y < 600; y++ {
			for x := 0; x < 960; x++ {
				i, j := 3*((y/3)*320+x/3), 4*(y*960+x)
				copy(img.Pix[j:j+3], rgb[i:i+3])
				img.Pix[j+3] = 255
			}
		}
		originals[n] = img
		canvas := image.NewNRGBA(hd.Bounds())
		copy(canvas.Pix, img.Pix)
		for y := 0; y < 200; y += 8 {
			for x := 0; x < 320; x += 8 {
				if visible(indexed[n], base, x, y) {
					r := image.Rect(x*3, y*3, (x+8)*3, (y+8)*3)
					draw.Draw(canvas, r, hd, r.Min, draw.Src)
				}
			}
		}
		if ally != nil {
			region, e := pbl.Region(indexed[n], 320, 200, 264, 152, 24, 32)
			check(e)
			if !bytes.Equal(region, allyPixels) {
				log.Fatal("本批 GUI 起點沒有完整 ALLY #0")
			}
			for yy := 0; yy < 32; yy += 8 {
				for xx := 0; xx < 24; xx += 8 {
					ink := false
					for y := yy; y < yy+8; y++ {
						for x := xx; x < xx+8; x++ {
							if allyPixels[y*24+x] != 0 {
								ink = true
							}
						}
					}
					if ink {
						r := image.Rect((264+xx)*3, (152+yy)*3, (264+xx+8)*3, (152+yy+8)*3)
						draw.Draw(canvas, r, ally, image.Pt(xx*3, yy*3), draw.Over)
					}
				}
			}
		}
		expectedHD[n] = canvas
		o.Close()
	}
	shots := map[string]*image.NRGBA{}
	shotIndexed := map[string][]byte{}
	shotExpected := map[string]*image.NRGBA{}
	diffs := map[string]int{}
	roi := image.Rect(696, 0, 960, 432)
	for _, name := range []string{"a-hd-chinese", "b-hd-english", "c-original-english", "d-original-chinese", "e-hd-chinese", "f-original-before-move", "g-moved", "h-loaded-original", "i-loaded-hd"} {
		p := filepath.Join(*out, name+".png")
		inputs[p] = hash(read(p))
		img := loadPNG(p)
		shots[name] = img
		n := 0
		if name == "h-loaded-original" || name == "i-loaded-hd" {
			n = 1
		}
		want := originals[n]
		if bytes.Contains([]byte(name), []byte("-hd")) {
			want = expectedHD[n]
		}
		statePath := filepath.Join(*out, name+".state")
		if _, e := os.Stat(statePath); e == nil {
			inputs[statePath] = hash(read(statePath))
			o, e := oracle.Load("/orig/psychic-war/PW.EXE", "/orig/psychic-war")
			check(e)
			check(o.LoadStateFile(statePath))
			idx := append([]byte(nil), o.Indexed()...)
			shotIndexed[name] = idx
			_, _, rgb := o.ScreenRGB()
			original := image.NewNRGBA(hd.Bounds())
			for y := 0; y < 600; y++ {
				for x := 0; x < 960; x++ {
					i, j := 3*((y/3)*320+x/3), 4*(y*960+x)
					copy(original.Pix[j:j+3], rgb[i:i+3])
					original.Pix[j+3] = 255
				}
			}
			canvas := image.NewNRGBA(hd.Bounds())
			copy(canvas.Pix, original.Pix)
			for y := 0; y < 200; y += 8 {
				for x := 0; x < 320; x += 8 {
					if visible(idx, base, x, y) {
						r := image.Rect(x*3, y*3, (x+8)*3, (y+8)*3)
						draw.Draw(canvas, r, hd, r.Min, draw.Src)
					}
				}
			}
			if ally != nil {
				region, e := pbl.Region(idx, 320, 200, 264, 152, 24, 32)
				check(e)
				if !bytes.Equal(region, allyPixels) {
					log.Fatal("逐張 GUI state 沒有完整 ALLY #0")
				}
				for yy := 0; yy < 32; yy += 8 {
					for xx := 0; xx < 24; xx += 8 {
						ink := false
						for y := yy; y < yy+8; y++ {
							for x := xx; x < xx+8; x++ {
								if allyPixels[y*24+x] != 0 {
									ink = true
								}
							}
						}
						if ink {
							r := image.Rect((264+xx)*3, (152+yy)*3, (264+xx+8)*3, (152+yy+8)*3)
							draw.Draw(canvas, r, ally, image.Pt(xx*3, yy*3), draw.Over)
						}
					}
				}
			}
			shotExpected[name] = canvas
			want = original
			if bytes.Contains([]byte(name), []byte("-hd")) {
				want = canvas
			}
			o.Close()
		}
		diffs[name] = different(img, want, roi)
		if diffs[name] != 0 {
			log.Fatalf("%s 原版位置人物與框線不符 %d 像素", name, diffs[name])
		}
		if ally != nil {
			key := name + " ally"
			diffs[key] = different(img, want, image.Rect(264*3, 152*3, 288*3, 184*3))
			if diffs[key] != 0 {
				log.Fatalf("%s 戰友圖不符 %d 像素", name, diffs[key])
			}
		}
	}
	menu := image.Rect(480, 12, 744, 228)
	if ally != nil {
		diffs["ally_hd_negative"] = different(shots["b-hd-english"], shots["c-original-english"], image.Rect(264*3, 152*3, 288*3, 184*3))
		if diffs["ally_hd_negative"] == 0 {
			log.Fatal("戰友 HD 負對照沒有差異")
		}
	}
	for _, pair := range [][2]string{{"a-hd-chinese", "b-hd-english"}, {"c-original-english", "d-original-chinese"}} {
		key := pair[0] + " vs " + pair[1]
		diffs[key] = different(shots[pair[0]], shots[pair[1]], menu)
		if diffs[key] == 0 {
			log.Fatal("語言切換未改動面板文字：", key)
		}
	}
	for _, pair := range [][2]string{{"a-hd-chinese", "e-hd-chinese"}, {"f-original-before-move", "h-loaded-original"}} {
		key := pair[0] + " vs " + pair[1]
		predicted := shots[pair[0]]
		if pair[0] == "a-hd-chinese" && shotIndexed[pair[0]] != nil && shotIndexed[pair[1]] != nil {
			predicted = image.NewNRGBA(hd.Bounds())
			copy(predicted.Pix, shots[pair[0]].Pix)
			for y := 0; y < 200; y += 8 {
				for x := 0; x < 320; x += 8 {
					if visible(shotIndexed[pair[0]], base, x, y) != visible(shotIndexed[pair[1]], base, x, y) {
						r := image.Rect(x*3, y*3, (x+8)*3, (y+8)*3)
						draw.Draw(predicted, r, shotExpected[pair[1]], r.Min, draw.Src)
					}
				}
			}
		}
		diffs[key] = different(predicted, shots[pair[1]], shots[pair[0]].Bounds())
		// 每張 F10 state 與三秒後視窗截圖不是同一幀。跨時間 HD 整屏差異只記錄，
		// 不能據此宣稱同狀態對拍；人物／框線與角色區域已在逐張循環獨立核對。
		if pair[0] == "a-hd-chinese" && ally != nil {
			continue
		}
		if diffs[key] != 0 {
			log.Fatalf("切回或讀回後畫面不同：%s，%d 像素", key, diffs[key])
		}
	}
	// 由兩份原版色號推導完整 8×8 遮格轉換，不硬編碼差異位置或容許像素數。
	transition := image.NewNRGBA(hd.Bounds())
	copy(transition.Pix, shots["e-hd-chinese"].Pix)
	if shotIndexed["e-hd-chinese"] != nil && shotIndexed["i-loaded-hd"] != nil {
		indexed[0], indexed[1] = shotIndexed["e-hd-chinese"], shotIndexed["i-loaded-hd"]
		expectedHD[1] = shotExpected["i-loaded-hd"]
	}
	var changedCells [][2]int
	for y := 0; y < 200; y += 8 {
		for x := 0; x < 320; x += 8 {
			if visible(indexed[0], base, x, y) == visible(indexed[1], base, x, y) {
				continue
			}
			changedCells = append(changedCells, [2]int{x, y})
			r := image.Rect(x*3, y*3, (x+8)*3, (y+8)*3)
			draw.Draw(transition, r, expectedHD[1], r.Min, draw.Src)
		}
	}
	diffs["reload_raw_delta"] = different(shots["e-hd-chinese"], shots["i-loaded-hd"], hd.Bounds())
	diffs["reload_after_original_grid_transition"] = different(transition, shots["i-loaded-hd"], hd.Bounds())
	if diffs["reload_after_original_grid_transition"] != 0 && ally == nil {
		log.Fatal("讀檔後的差異不符合原版完整格遮罩轉換")
	}
	diffs["movement_negative"] = different(shots["f-original-before-move"], shots["g-moved"], shots["g-moved"].Bounds())
	if diffs["movement_negative"] == 0 {
		log.Fatal("負對照：移動沒有改動畫面")
	}
	for _, name := range []string{"stats.jsonl", "frontend.log", "saves/quick.json", "saves/quick.state.xlate.json"} {
		p := filepath.Join(*out, name)
		inputs[p] = hash(read(p))
	}
	result := map[string]any{"status": "實際 Linux 視窗切換及 F10／F11 通過", "inputs_sha256": inputs, "differences": diffs, "original_mask_transition_cells": changedCells, "save_observation": "執行期線性 0x16966 起 52 bytes 相同；未比較整份自然運行 RAM 或原版 DAT", "hd_roi": [4]int{232, 0, 88, 144}, "limits": "保存正常玩家起點的 Xvfb 前端抽測；不是從開機完整重跑、發行包、全部美術／sprite、真機或跨平台驗收。音訊使用 null，不宣稱音訊／幀率通過。"}
	if ally != nil {
		result["status"] = "實際 Linux 視窗的 ALLY #0、人物／框線區域、切換與 F10／F11 位置抽測通過"
		result["ally_roi"] = [4]int{264, 152, 24, 32}
		result["whole_screen_limit"] = "逐張 state 與稍後視窗截圖非同一幀；跨時間 HD 整屏差異僅記錄，不作本批整屏同狀態完成證據。完整狀態不變另見固定種子 ally-runtime 收據。"
	}
	b, e := json.MarshalIndent(result, "", "  ")
	check(e)
	check(os.WriteFile(receipt, append(b, '\n'), 0644))
	log.Print("實際視窗像素、語言、HD、移動負對照及讀檔位置核對通過")
}

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}
func read(p string) []byte { b, e := os.ReadFile(p); check(e); return b }
func hash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func visible(idx, base []byte, x, y int) bool {
	for row := y; row < y+8; row++ {
		i := row*320 + x
		if !bytes.Equal(idx[i:i+8], base[i:i+8]) {
			return false
		}
	}
	return true
}
func loadPNG(p string) *image.NRGBA {
	f, e := os.Open(p)
	check(e)
	img, e := png.Decode(f)
	check(e)
	check(f.Close())
	if img.Bounds().Dx() != 960 || img.Bounds().Dy() != 600 {
		log.Fatal("截圖尺寸不符：", p)
	}
	out := image.NewNRGBA(img.Bounds())
	draw.Draw(out, out.Bounds(), img, img.Bounds().Min, draw.Src)
	return out
}
func different(a, b *image.NRGBA, r image.Rectangle) int {
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			i := a.PixOffset(x, y)
			j := b.PixOffset(x, y)
			if !bytes.Equal(a.Pix[i:i+3], b.Pix[j:j+3]) {
				n++
			}
		}
	}
	return n
}
