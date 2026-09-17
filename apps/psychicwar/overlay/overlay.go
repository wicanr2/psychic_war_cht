// Package overlay 是放大畫布上的中文疊字層（docs/spec/008 §3.3–3.5）。
//
// 不認識 dosgolem 也不認識 Ebiten：輸入是原版 320×200 的色號與 RGB，輸出是放大後的 RGBA。
// 位址與「什麼時候蓋一筆」由呼叫端（apps/psychicwar 的轉譯層）決定。
package overlay

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

// 原版畫面與字格。
const (
	ScreenW = 320
	ScreenH = 200
	Cell    = 8  // 原版 FONT.BIN 一個字格
	Glyph   = 24 // 字模大小
)

// Font 是 font/cjk24.bin 的內容：碼點 → 72 bytes（每列 3 bytes，MSB 在左）。
type Font map[rune][72]byte

// LoadFont 讀 tools/font/bake.sh 產生的字型子集。
func LoadFont(path string) (Font, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) < 12 || string(b[:8]) != "PWCJK24\x00" {
		return nil, fmt.Errorf("%s 不是 PWCJK24 字型", path)
	}
	n := int(binary.LittleEndian.Uint32(b[8:]))
	if len(b) != 12+n*77 {
		return nil, fmt.Errorf("%s 長度 %d，%d 字應該是 %d", path, len(b), n, 12+n*77)
	}
	f := make(Font, n)
	for i := 0; i < n; i++ {
		o := 12 + i*77
		var g [72]byte
		copy(g[:], b[o+5:o+77])
		f[rune(binary.LittleEndian.Uint32(b[o:]))] = g
	}
	return f, nil
}

// ErrTooLong 表示譯文放不下，多的字被截掉。
var ErrTooLong = errors.New("譯文過長")

// Layout 把譯文排進各行的格數（一個字元一格；`\n` 強制換行）。放不下時回 ErrTooLong 與截掉後的結果。
func Layout(text string, widths []int) ([][]rune, error) {
	out := make([][]rune, len(widths))
	line := 0
	var err error
	for _, r := range text {
		if line >= len(widths) {
			err = ErrTooLong
			break
		}
		if r == '\n' {
			line++
			continue
		}
		if len(out[line]) == widths[line] {
			line++
			if line >= len(widths) {
				err = ErrTooLong
				break
			}
		}
		out[line] = append(out[line], r)
	}
	return out, err
}

// State 是一筆疊字的狀態。
type State int

const (
	Printing State = iota // 原版還在畫這一行
	Pending               // 畫完了，等下一幀定色
	Shown                 // 已定色、顯示中
)

// Stamp 是一行中文：左上角 (X, Y)、Cells 格，原版像素座標。
type Stamp struct {
	Key    string
	X, Y   int
	Cells  int
	Text   []rune
	State  State
	FG, BG [3]uint8
	hash   uint64
	misses int
}

// Rect 回這一筆蓋住的原版像素範圍 [x0,x1)×[y0,y1)。
func (s *Stamp) Rect() (x0, y0, x1, y1 int) { return s.X, s.Y, s.X + s.Cells*Cell, s.Y + Cell }

// Layer 是目前所有疊字。
type Layer struct {
	Stamps []*Stamp
	// OnDrop 在一筆被移除時呼叫（原因：overlap、scroll、changed）。可為 nil。
	OnDrop func(s *Stamp, why string)
}

func overlap(a, b *Stamp) bool {
	ax0, ay0, ax1, ay1 := a.Rect()
	bx0, by0, bx1, by1 := b.Rect()
	return ax0 < bx1 && bx0 < ax1 && ay0 < by1 && by0 < ay1
}

// Add 加一筆；與它重疊的舊疊字移除。
func (l *Layer) Add(s *Stamp) {
	keep := l.Stamps[:0]
	for _, old := range l.Stamps {
		if overlap(old, s) {
			l.drop(old, "overlap")
			continue
		}
		keep = append(keep, old)
	}
	l.Stamps = append(keep, s)
}

func (l *Layer) drop(s *Stamp, why string) {
	if l.OnDrop != nil {
		l.OnDrop(s, why)
	}
}

// Scroll 把左上角落在 [x0,x1)×[y0,y1) 的疊字移 dy 像素；移出上緣的移除（docs/spec/008 §3.4）。
func (l *Layer) Scroll(x0, y0, x1, y1, dy int) {
	keep := l.Stamps[:0]
	for _, s := range l.Stamps {
		if s.X >= x0 && s.X < x1 && s.Y >= y0 && s.Y < y1 {
			s.Y += dy
			if s.Y < y0 || s.Y+Cell > y1 {
				l.drop(s, "scroll")
				continue
			}
		}
		keep = append(keep, s)
	}
	l.Stamps = keep
}

// Colors 回一塊色號裡出現最多（背景）與第二多（前景）的色號；只有一種時兩者相同。
func Colors(idx []uint8) (bg, fg uint8) {
	var count [256]int
	for _, v := range idx {
		count[v]++
	}
	best, second := -1, -1
	for c := 0; c < 256; c++ {
		if count[c] == 0 {
			continue
		}
		if best < 0 || count[c] > count[best] {
			best, second = c, best
		} else if second < 0 || count[c] > count[second] {
			second = c
		}
	}
	if best < 0 {
		return 0, 0
	}
	if second < 0 {
		second = best
	}
	return uint8(best), uint8(second)
}

func (s *Stamp) region(indexed []uint8) []uint8 {
	x0, y0, x1, y1 := s.Rect()
	out := make([]uint8, 0, (x1-x0)*(y1-y0))
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if x >= 0 && x < ScreenW && y >= 0 && y < ScreenH {
				out = append(out, indexed[y*ScreenW+x])
			}
		}
	}
	return out
}

func fnv(b []uint8) uint64 {
	h := uint64(14695981039346656037)
	for _, v := range b {
		h ^= uint64(v)
		h *= 1099511628211
	}
	return h
}

// Frame 在機器停下來之後呼叫一次：定色、檢查失效（docs/spec/008 §3.4）。indexed 是 320×200 色號，rgb 是同一幀的 RGB。
func (l *Layer) Frame(indexed, rgb []uint8) {
	keep := l.Stamps[:0]
	for _, s := range l.Stamps {
		switch s.State {
		case Pending:
			reg := s.region(indexed)
			bg, fg := Colors(reg)
			s.BG, s.FG = pick(s, indexed, rgb, bg), pick(s, indexed, rgb, fg)
			s.hash, s.misses, s.State = fnv(reg), 0, Shown
		case Shown:
			if fnv(s.region(indexed)) != s.hash {
				s.misses++
				if s.misses >= 3 {
					l.drop(s, "changed")
					continue
				}
			} else {
				s.misses = 0
			}
		}
		keep = append(keep, s)
	}
	l.Stamps = keep
}

// pick 從 rgb 取色號 c 在這一筆範圍內第一次出現的位置的顏色。
func pick(s *Stamp, indexed, rgb []uint8, c uint8) [3]uint8 {
	x0, y0, x1, y1 := s.Rect()
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if x >= 0 && x < ScreenW && y >= 0 && y < ScreenH && indexed[y*ScreenW+x] == c {
				i := 3 * (y*ScreenW + x)
				return [3]uint8{rgb[i], rgb[i+1], rgb[i+2]}
			}
		}
	}
	return [3]uint8{}
}

// Draw 把顯示中的疊字畫進放大後的 RGBA（寬 320×scale）。scale 必須是 3 的倍數。
// missing 對字型沒有的字呼叫（可為 nil）。回有沒有畫任何東西。
func (l *Layer) Draw(dst []uint8, scale int, font Font, missing func(r rune)) bool {
	if scale%3 != 0 {
		return false
	}
	k := scale / 3
	W := ScreenW * scale
	drew := false
	for _, s := range l.Stamps {
		if s.State != Shown {
			continue
		}
		drew = true
		x0, y0, x1, y1 := s.Rect()
		for y := y0 * scale; y < y1*scale; y++ {
			for x := x0 * scale; x < x1*scale; x++ {
				set(dst, W, x, y, s.BG)
			}
		}
		for i, r := range s.Text {
			if i >= s.Cells || r == ' ' || r == '　' {
				continue
			}
			g, ok := font[r]
			if !ok {
				if missing != nil {
					missing(r)
				}
				continue
			}
			cx, cy := (s.X+i*Cell)*scale, s.Y*scale
			for gy := 0; gy < Glyph; gy++ {
				for gx := 0; gx < Glyph; gx++ {
					if g[gy*3+gx/8]&(0x80>>(gx%8)) == 0 {
						continue
					}
					for dy := 0; dy < k; dy++ {
						for dx := 0; dx < k; dx++ {
							set(dst, W, cx+gx*k+dx, cy+gy*k+dy, s.FG)
						}
					}
				}
			}
		}
	}
	return drew
}

func set(dst []uint8, w, x, y int, c [3]uint8) {
	i := 4 * (y*w + x)
	if x < 0 || x >= w || i < 0 || i+3 >= len(dst) {
		return
	}
	dst[i], dst[i+1], dst[i+2], dst[i+3] = c[0], c[1], c[2], 0xFF
}
