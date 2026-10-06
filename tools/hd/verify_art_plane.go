// 正式 204 圖面層的已保存正常幀驗證；來源、限制見 docs/re/038 §32。
// 不執行遊戲、不重播或改寫 state；此工具只消費已保存的原版及中文畫面。
package main

import (
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

	"github.com/wicanr2/dosgolem/xlate"
)

const W, H, S = 320, 200, 3

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
func read(path string) []byte { b, e := os.ReadFile(path); check(e); return b }
func hash(b []byte) string    { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func picture(path string) *image.NRGBA {
	f, e := os.Open(path)
	check(e)
	defer f.Close()
	img, e := png.Decode(f)
	check(e)
	if img.Bounds().Dx() != W*S || img.Bounds().Dy() != H*S {
		log.Fatal("畫面尺寸不符：", path)
	}
	out := image.NewNRGBA(image.Rect(0, 0, W*S, H*S))
	draw.Draw(out, out.Bounds(), img, img.Bounds().Min, draw.Src)
	return out
}
func write(path string, b []byte) {
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	check(e)
	_, e = f.Write(b)
	check(e)
	check(f.Close())
}

type phase struct {
	Branch    string `json:"branch"`
	Scene     string `json:"scene"`
	Input     string `json:"input"`
	Overlay   string `json:"overlay"`
	Art       string `json:"art"`
	Composed  string `json:"composed"`
	PNG       string `json:"png"`
	Rows      int    `json:"rows"`
	MakeCalls int    `json:"make_calls"`
}

func main() {
	prefix := flag.String("out", "", "既有輸出目錄內的新前綴，禁止覆寫")
	flag.Parse()
	if *prefix == "" {
		log.Fatal("必須指定 -out")
	}
	info, e := os.Stat(filepath.Dir(*prefix))
	check(e)
	if !info.IsDir() {
		log.Fatal("輸出目錄不存在")
	}
	scenes := [][]string{
		{"replay-encounter", "replay-escape-20261001", "replay-escape-settled-20261001"},
		{"replay-encounter", "replay-noescape-20261001", "replay-noescape-settled-20261001"},
	}
	// 在任何寫入前檢查整批固定輸出；禁止「最後才發現既有收據」。
	for branch, list := range scenes {
		for index := range list {
			for _, ext := range []string{"-art.rgba", "-composed.rgba", "-composed.png"} {
				path := fmt.Sprintf("%s-%d-%d%s", *prefix, branch, index, ext)
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					log.Fatal("拒絕覆寫：", path)
				}
			}
		}
	}
	for _, ext := range []string{".json", "-permanent-hole.rgba"} {
		path := *prefix + ext
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			log.Fatal("拒絕覆寫：", path)
		}
	}
	base := read("workplace/hd/bg.idx")
	if len(base) != W*H {
		log.Fatal("背景色號尺寸不符")
	}
	hdPath := "workplace/hd/redraw/layout-v2-20261001-background.png"
	hd := picture(hdPath)
	inputs := map[string]string{}
	addInput := func(path string) { inputs[path] = hash(read(path)) }
	for _, path := range []string{"tools/hd/verify_art_plane.go", "worktrees/dosgolem/xlate/art.go",
		"worktrees/dosgolem/xlate/art_test.go", "worktrees/dosgolem/xlate/layer.go", "worktrees/dosgolem/xlate/watch.go",
		"worktrees/dosgolem/docs/spec/204-art-plane.md", "workplace/hd/bg.idx", hdPath,
		"workplace/hd/redraw/normal-recovery-20261001.json", "workplace/hd/source-before-art-20261001/manifest.json"} {
		addInput(path)
	}
	register := func(l *xlate.Layer, made *int) {
		keys := make([]string, H/8)
		for i := range keys {
			keys[i] = fmt.Sprintf("background-%02d", i)
		}
		l.Watch(&xlate.Watcher{Key: "hd-background", Art: true, ArtKeys: keys, X: 0, Y: 0, W: W, H: 40,
			Want: append([]byte(nil), base[:W*40]...), Make: func() []*xlate.Stamp {
				*made++
				rows := make([]*xlate.Stamp, H/8)
				for i := range rows {
					rows[i] = &xlate.Stamp{Key: keys[i], Y: i * 8, Cells: 40, CellW: 8, CellH: 8, Art: true, PixScale: S,
						Reference: base[i*8*W : (i+1)*8*W], Pix: hd.Pix[i*8*S*W*S*4 : (i+1)*8*S*W*S*4]}
				}
				return rows
			}})
	}
	compose := func(art []byte, srcPrefix string) *image.NRGBA {
		img := picture(srcPrefix + "-original.png")
		over := &image.NRGBA{Pix: art, Stride: W * S * 4, Rect: img.Bounds()}
		draw.Draw(img, img.Bounds(), over, image.Point{}, draw.Over)
		draw.Draw(img, img.Bounds(), picture(srcPrefix+"-overlay.png"), image.Point{}, draw.Over)
		return img
	}
	outputs := map[string]string{}
	writeOutput := func(path string, b []byte) { write(path, b); outputs[path] = hash(b) }
	var phases []phase
	for branch, list := range scenes {
		l := &xlate.Layer{W: W, H: H}
		made := 0
		register(l, &made)
		for index, scene := range list {
			src := "workplace/hd/redraw/" + scene
			for _, ext := range []string{".frame", "-original.png", "-overlay.png", "-chinese.png", ".state.xlate.json"} {
				addInput(src + ext)
			}
			idx := read(src + ".frame")
			if len(idx) != W*H {
				log.Fatal("原版畫面長度不符")
			}
			img := picture(src + "-original.png")
			rgb := make([]byte, W*H*3)
			for y := 0; y < H; y++ {
				for x := 0; x < W; x++ {
					copy(rgb[3*(y*W+x):3*(y*W+x)+3], img.Pix[4*(y*S*W*S+x*S):4*(y*S*W*S+x*S)+3])
				}
			}
			l.Frame(idx, rgb)
			art := make([]byte, W*H*S*S*4)
			l.Draw(art, S, nil)
			out := fmt.Sprintf("%s-%d-%d", *prefix, branch, index)
			composed := compose(art, src)
			writeOutput(out+"-art.rgba", art)
			writeOutput(out+"-composed.rgba", composed.Pix)
			f, e := os.OpenFile(out+"-composed.png", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
			check(e)
			check(png.Encode(f, composed))
			check(f.Close())
			outputs[out+"-composed.png"] = hash(read(out + "-composed.png"))
			name := "escape"
			if branch == 1 {
				name = "control"
			}
			phases = append(phases, phase{Branch: name, Scene: scene, Input: src + ".frame", Overlay: src + "-overlay.png",
				Art: out + "-art.rgba", Composed: out + "-composed.rgba", PNG: out + "-composed.png", Rows: len(l.Art), MakeCalls: made})
			if branch == 0 && index == 0 {
				// 受控錯誤消費者：把初始遮格當永久 Transparent，應漏掉正常 F3 恢復。
				bad := &xlate.Layer{W: W, H: H}
				badMade := 0
				register(bad, &badMade)
				bad.Frame(idx, rgb)
				for _, s := range bad.Art {
					s.Transparent = make([]bool, s.Cells)
					for cell := 0; cell < s.Cells; cell++ {
						s.Transparent[cell] = art[4*((s.Y*S)*W*S+cell*8*S)+3] == 0
					}
				}
				last := "workplace/hd/redraw/" + list[len(list)-1]
				bad.Frame(read(last+".frame"), rgb)
				badArt := make([]byte, len(art))
				bad.Draw(badArt, S, nil)
				writeOutput(*prefix+"-permanent-hole.rgba", compose(badArt, last).Pix)
			}
		}
	}
	receipt := map[string]any{"status": "正式 204 已保存正常幀驗證；不是正式主題／前端或全 sprite 完成",
		"go_version": runtime.Version(), "cell": [2]int{8, 8}, "phases": phases,
		"inputs_sha256": inputs, "outputs_sha256": outputs, "negative": *prefix + "-permanent-hole.rgba",
		"normal_source": "docs/re/038 §29；兩分支從執行前固定 CAF0h 的保存狀態取得。本次沒有執行原版或重擲亂數。",
		"limits":        "本工具只消費保存幀與獨立中文圖層；不證明實際前端切換、存讀檔、整款素材或平台完成。"}
	b, e := json.MarshalIndent(receipt, "", "  ")
	check(e)
	write(*prefix+".json", append(b, '\n'))
	log.Printf("正式圖面層：%d 個正常幀消費結果；收據 %s.json", len(phases), *prefix)
}
