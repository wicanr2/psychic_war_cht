// 將已合成、已核對排版的本機背景切成 024 的 SCREEN／MENU 資產。
// 不生成或重繪美術；不判定散布權利。入口見 docs/re/038 §33。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/theme"
)

func main() {
	source := flag.String("source", "", "已驗合成 PNG（960×600）")
	out := flag.String("out", "", "新的本機研究輸出目錄，禁止覆寫")
	flag.Parse()
	if *source == "" || *out == "" {
		log.Fatal("必須指定 -source 與 -out")
	}
	f, err := os.Open(*source)
	check(err)
	img, err := png.Decode(f)
	check(err)
	check(f.Close())
	if img.Bounds().Dx() != 960 || img.Bounds().Dy() != 600 {
		log.Fatal("背景必須為 960×600")
	}
	if _, err := os.Lstat(*out); !os.IsNotExist(err) {
		log.Fatal("拒絕覆寫輸出目錄：", *out)
	}
	check(os.Mkdir(*out, 0755))
	m := theme.ThemeManifest{Schema: "psychic-war-theme/1", Name: "hd", Title: "高解析度", Scale: 3}
	for i := 0; i < 6; i++ {
		x, y, w, h, file, n := 0, i*40, 320, 40, "SCREEN.PBL", i
		if i == 5 {
			x, y, w, h, file, n = 160, 4, 88, 72, "MENU.PBL", 0
		}
		name := fmt.Sprintf("%s-%02d.png", file[:len(file)-4], n)
		part := image.NewNRGBA(image.Rect(0, 0, w*3, h*3))
		draw.Draw(part, part.Bounds(), img, image.Pt(x*3, y*3), draw.Src)
		f, err := os.OpenFile(filepath.Join(*out, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		check(err)
		check(png.Encode(f, part))
		check(f.Close())
		m.Entries = append(m.Entries, theme.ThemeEntry{PBL: file, Image: n, At: []int{x, y}, PNG: name, Kind: "redraw", Match: []int{0, 0, 320, 40}})
	}
	b, err := json.MarshalIndent(m, "", "  ")
	check(err)
	check(os.WriteFile(filepath.Join(*out, "manifest.json"), append(b, '\n'), 0644))
	log.Printf("已切分六筆本機主題資產；完整美術與散布權利仍須另驗：%s", *out)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
