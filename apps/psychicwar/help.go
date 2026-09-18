package psychicwar

// F1 說明頁（docs/spec/012 §3）。內容是這一款遊戲專屬的，留在本 repo。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wicanr2/dosgolem/xlate"
)

// 說明頁的版面（放大 3 倍的畫布上，一格 24×24）。
const (
	HelpCols = 38 // 每行最多幾個中文字
	HelpRows = 22 // 最多幾行
)

// 說明頁內容在 text/help.json（LoadHelp 讀），不寫死在程式裡：驗收工具要用同一份算期望值。
// LoadHelp 讀 <dir>/help.json 的說明頁內容（docs/spec/012 §3）。
func LoadHelp(dir string) ([]string, error) {
	b, err := os.ReadFile(filepath.Join(dir, "help.json"))
	if err != nil {
		return nil, err
	}
	var doc struct {
		Schema string   `json:"schema"`
		Lines  []string `json:"lines"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	if doc.Schema != "psychic-war-help/1" {
		return nil, fmt.Errorf("help.json 的 schema 不是 psychic-war-help/1：%q", doc.Schema)
	}
	return doc.Lines, CheckHelp(doc.Lines)
}

// LineCells 回一行佔幾格（半形字佔半格，用兩倍整數避免小數）：回的是「半格數」。
func LineCells(s string) int {
	n := 0
	for _, r := range s {
		if r < 0x80 {
			n++ // 半形：半格
		} else {
			n += 2
		}
	}
	return n
}

// CheckHelp 檢查說明頁排得下；排不下回錯（建置期就擋住，docs/spec/012 §5 第 1 項）。
func CheckHelp(lines []string) error {
	if len(lines) > HelpRows {
		return fmt.Errorf("說明頁 %d 行，超過 %d 行", len(lines), HelpRows)
	}
	for i, s := range lines {
		if n := LineCells(s); n > 2*HelpCols {
			return fmt.Errorf("說明頁第 %d 行 %.1f 格，超過 %d 格：%s", i+1, float64(n)/2, HelpCols, s)
		}
	}
	return nil
}

// MissingHelpGlyphs 回說明頁用到、但字型沒有的字。
func MissingHelpGlyphs(lines []string, f *xlate.Font) []rune {
	var out []rune
	seen := map[rune]bool{}
	for _, s := range lines {
		for _, r := range s {
			if r == ' ' || r == '　' || seen[r] || f == nil {
				continue
			}
			if _, ok := f.Glyphs[r]; !ok {
				seen[r] = true
				out = append(out, r)
			}
		}
	}
	return out
}

// DrawTextPage 把整頁文字畫進放大後的 RGBA（寬 w 像素）。
//
// 一格 cell×cell 像素，字模置左上；fg、bg 是 RGB。bgAlpha 0 表示不填背景（只畫字）。
func DrawTextPage(dst []uint8, w, h int, f *xlate.Font, lines []string, cell int, fg, bg [3]uint8, bgAlpha uint8) {
	if f == nil {
		return
	}
	if bgAlpha > 0 {
		for i := 0; i < w*h; i++ {
			dst[4*i], dst[4*i+1], dst[4*i+2], dst[4*i+3] = bg[0], bg[1], bg[2], bgAlpha
		}
	}
	rb := (f.W + 7) / 8
	sx := (w - HelpCols*cell) / 2
	sy := (h - len(lines)*cell) / 2
	if sx < 0 {
		sx = 0
	}
	if sy < 0 {
		sy = 0
	}
	scale := cell / f.W
	if scale < 1 {
		scale = 1
	}
	for row, s := range lines {
		pen := 0 // 目前的水平位置，單位是半格
		for _, r := range s {
			adv := 2
			if r < 0x80 {
				adv = 1 // 半形字佔半格
			}
			g, ok := f.Glyphs[r]
			if !ok {
				pen += adv
				continue
			}
			x0, y0 := sx+pen*cell/2, sy+row*cell
			pen += adv
			for gy := 0; gy < f.H; gy++ {
				for gx := 0; gx < f.W; gx++ {
					if g[gy*rb+gx/8]&(0x80>>(gx%8)) == 0 {
						continue
					}
					for py := 0; py < scale; py++ {
						for px := 0; px < scale; px++ {
							x, y := x0+gx*scale+px, y0+gy*scale+py
							if x < 0 || x >= w || y < 0 || y >= h {
								continue
							}
							i := y*w + x
							dst[4*i], dst[4*i+1], dst[4*i+2], dst[4*i+3] = fg[0], fg[1], fg[2], 0xFF
						}
					}
				}
			}
		}
	}
}
