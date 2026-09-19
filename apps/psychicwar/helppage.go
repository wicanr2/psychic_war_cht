package psychicwar

// F4 說明頁的版面（docs/spec/022，方案 B：標題色帶 ＋ 雙欄 ＋ 通欄尾段）。
//
// 版面常數全部在 text/help.json 的 `layout`，Go 與 tools/help_check.py 讀同一份：
// 驗收工具的期望值要能從資料算出來，兩邊各自寫死常數就會算出不同的圖而誰都不知道。
//
// 繪圖收斂成三個指令（rect／text／chip），先算出一串指令再光柵化。分成兩段是因為
// 幾何只用字型的「前進量」（半形半格、全形一格），不看字模——CheckHelp 在建置期
// 還沒載入字型，也要能算出內容底緣。

import (
	"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/wicanr2/dosgolem/xlate"
)

// egaRGB 是 EGA 預設 16 色（200 線 RGBI），與 tools/pbl.py 的 EGA 表、docs/re/005 的色盤一致。
// 說明頁只能用這 16 色：驗收是用 pbl.screen_indices 反查色號，出現別的顏色它會直接結束。
var egaRGB = [16][3]uint8{
	{0x00, 0x00, 0x00}, {0x00, 0x00, 0xAA}, {0x00, 0xAA, 0x00}, {0x00, 0xAA, 0xAA},
	{0xAA, 0x00, 0x00}, {0xAA, 0x00, 0xAA}, {0xAA, 0x55, 0x00}, {0xAA, 0xAA, 0xAA},
	{0x55, 0x55, 0x55}, {0x55, 0x55, 0xFF}, {0x55, 0xFF, 0x55}, {0x55, 0xFF, 0xFF},
	{0xFF, 0x55, 0x55}, {0xFF, 0x55, 0xFF}, {0xFF, 0xFF, 0x55}, {0xFF, 0xFF, 0xFF},
}

// HelpLayout 是說明頁的版面常數（text/help.json 的 `layout`，docs/spec/022 §4）。
// 單位是 960×600 的設計座標；畫布更大時整數倍放大。
type HelpLayout struct {
	PageW         int `json:"page_w"`
	PageHeight    int `json:"page_h"`
	Margin        int `json:"margin"`
	BandY         int `json:"band_y"`
	BandH         int `json:"band_h"`
	ColW          int `json:"col_w"`
	Gutter        int `json:"gutter"`
	BodyY         int `json:"body_y"`
	Pitch         int `json:"pitch"`
	HeadPitch     int `json:"head_pitch"`
	Gap           int `json:"gap"`
	TailGap       int `json:"tail_gap"`
	TailDY        int `json:"tail_dy"`
	Rule          int `json:"rule"`
	RuleInset     int `json:"rule_inset"`
	ChipPad       int `json:"chip_pad"`
	ProtGap       int `json:"prot_gap"`
	FontHeadW     int `json:"font_head_w"`
	FontHeadH     int `json:"font_head_h"`
	FontBodyW     int `json:"font_body_w"`
	FontBodyH     int `json:"font_body_h"`
	ColorBG       int `json:"color_bg"`
	ColorBand     int `json:"color_band"`
	ColorTitle    int `json:"color_title"`
	ColorHead     int `json:"color_head"`
	ColorBody     int `json:"color_body"`
	ColorRule     int `json:"color_rule"`
	ColorChip     int `json:"color_chip"`
	ColorChipText int `json:"color_chip_text"`
	ColorProt     int `json:"color_prot"`
}

// HelpDoc 是說明頁的內容與版面（LoadHelp 讀進來的整份 help.json）。
type HelpDoc struct {
	Lines   []string
	Keycaps []string // 已排成最長優先：F10 要排在 F1 前面，否則會被切成 F1 ＋ 0
	Layout  HelpLayout
}

// helpDoc 是目前這一份說明頁。DrawTextPage 靠它認出「這一頁是說明頁」，
// 所以版面規則全部留在本檔，前端只負責把內容交過來。
var helpDoc *HelpDoc

// sortKeycaps 把鍵名排成最長優先（長度相同時維持資料裡的順序）。
func sortKeycaps(caps []string) []string {
	out := append([]string(nil), caps...)
	sort.SliceStable(out, func(i, j int) bool { return len(out[i]) > len(out[j]) })
	return out
}

// helpWidth 回一段字用寬 fw 的字型佔幾個像素：半形半格、全形一格（docs/spec/022 §7）。
// 只看字碼不看字模，所以沒有字型也算得出來。
func helpWidth(fw int, s string) int {
	n := 0
	for _, r := range s {
		if r < 0x80 {
			n += fw / 2
		} else {
			n += fw
		}
	}
	return n
}

// helpSec 是一個段落：標頭、同列的尾巴、內文各列（docs/spec/022 §3.3）。
type helpSec struct {
	Head string
	Rest string
	Body []string
}

// parseHelpLines 照 docs/spec/022 §3.3 的四條規則把 lines 分段。
func parseHelpLines(lines []string) (string, []helpSec) {
	if len(lines) == 0 {
		return "", nil
	}
	title := strings.Trim(lines[0], "　 ")
	var secs []helpSec
	for _, s := range lines[1:] {
		if strings.Trim(s, "　 ") == "" {
			continue // 空行丟掉：段落間距由版面決定
		}
		if strings.HasPrefix(s, "【") {
			head, rest := s, ""
			if i := strings.Index(s, "】"); i >= 0 {
				head, rest = s[:i+len("】")], strings.Trim(s[i+len("】"):], "　 ")
			}
			secs = append(secs, helpSec{Head: head, Rest: rest})
		} else if len(secs) > 0 {
			secs[len(secs)-1].Body = append(secs[len(secs)-1].Body, s)
		}
	}
	return title, secs
}

// helpItem 是段落展開後的一列：標頭或內文。
type helpItem struct {
	Head bool
	S    string
	Tail string // 標頭同列的尾巴；放不下時會自成一列內文
}

// helpItems 把一個段落展開成列。標頭與尾巴同列放不進欄寬時，尾巴自成一列。
func helpItems(L HelpLayout, sec helpSec, colw int) []helpItem {
	out := []helpItem{{Head: true, S: sec.Head}}
	if sec.Rest != "" {
		if helpWidth(L.FontHeadW, sec.Head)+L.TailGap+helpWidth(L.FontBodyW, sec.Rest) <= colw {
			out[0].Tail = sec.Rest
		} else {
			out = append(out, helpItem{S: "　" + sec.Rest})
		}
	}
	for _, s := range sec.Body {
		out = append(out, helpItem{S: s})
	}
	return out
}

// 繪圖指令（docs/spec/022 §6）。
const (
	opRect = iota
	opText
	opChip
)

type helpOp struct {
	Kind   int
	X, Y   int
	W, H   int    // opRect
	Big    bool   // true ＝ 標頭字型 cjk24，false ＝ 內文字型 cjk16
	S      string // opText、opChip
	Color  int    // 文字色；opRect 是填色；opChip 是底色
	Color2 int    // opChip 的字色
	Pad    int    // opChip 的外擴
}

// splitHelpCaps 把一行切成「一般文字」與「鍵名」兩種段落，鍵名要畫成反白鍵帽。
// 比對最長優先，否則 F10 會被切成 F1 ＋ 0。
func splitHelpCaps(s string, caps []string, on bool) []helpOp {
	if !on || len(caps) == 0 {
		return []helpOp{{Kind: opText, S: s}}
	}
	rs := []rune(s)
	var out []helpOp
	buf := ""
	flush := func() {
		if buf != "" {
			out = append(out, helpOp{Kind: opText, S: buf})
			buf = ""
		}
	}
	for i := 0; i < len(rs); {
		hit := ""
		for _, k := range caps {
			n := len([]rune(k))
			if i+n <= len(rs) && string(rs[i:i+n]) == k {
				hit = k
				break
			}
		}
		if hit == "" {
			buf += string(rs[i])
			i++
			continue
		}
		flush()
		out = append(out, helpOp{Kind: opChip, S: hit})
		i += len([]rune(hit))
	}
	flush()
	return out
}

// appendHelpLine 畫一行內文：鍵名變鍵帽，其餘照畫。
func appendHelpLine(ops []helpOp, doc *HelpDoc, s string, x, y, color int, caps bool) []helpOp {
	L := doc.Layout
	pen := x
	for _, seg := range splitHelpCaps(s, doc.Keycaps, caps) {
		seg.X, seg.Y = pen, y
		if seg.Kind == opChip {
			seg.Color, seg.Color2, seg.Pad = L.ColorChip, L.ColorChipText, L.ChipPad
		} else {
			seg.Color = color
		}
		ops = append(ops, seg)
		pen += helpWidth(L.FontBodyW, seg.S)
	}
	return ops
}

// emitHelpGroup 把一組段落排進寬 colw 的欄裡，回新的指令清單與排完後的 y。
func emitHelpGroup(ops []helpOp, doc *HelpDoc, group []helpSec, x, colw, y int) ([]helpOp, int) {
	L := doc.Layout
	for si, sec := range group {
		if si > 0 {
			y += L.Gap
		}
		for _, it := range helpItems(L, sec, colw) {
			if it.Head {
				ops = append(ops, helpOp{Kind: opText, Big: true, S: it.S, X: x, Y: y, Color: L.ColorHead})
				if it.Tail != "" {
					ops = appendHelpLine(ops, doc, it.Tail,
						x+helpWidth(L.FontHeadW, it.S)+L.TailGap, y+L.TailDY, L.ColorBody, true)
				}
				y += L.HeadPitch
			} else {
				ops = appendHelpLine(ops, doc, it.S, x, y, L.ColorBody, true)
				y += L.Pitch
			}
		}
	}
	return ops, y
}

// helpGeometry 算出整頁的繪圖指令（docs/spec/022 §4）。prot 是空字串就不留防拷那一列。
//
// 欄位分配是算出來的：任何一行超過欄寬的段落整段移到下面通欄，其餘依高度取兩邊差最小的
// 切點。切點唯一（相同差值取最小的 k），所以同一份資料一定算出同一張圖。
func helpGeometry(doc *HelpDoc, prot string) []helpOp {
	L := doc.Layout
	title, secs := parseHelpLines(doc.Lines)
	colX := [2]int{L.Margin, L.Margin + L.ColW + L.Gutter}
	right := colX[1] + L.ColW

	ops := []helpOp{{Kind: opRect, X: 0, Y: 0, W: L.PageW, H: L.PageHeight, Color: L.ColorBG}}
	ops = append(ops, helpOp{Kind: opRect, X: L.Margin, Y: L.BandY, W: right - L.Margin, H: L.BandH, Color: L.ColorBand})
	ops = append(ops, helpOp{Kind: opText, Big: true, S: title, Color: L.ColorTitle,
		X: (L.PageW - helpWidth(L.FontHeadW, title)) / 2,
		Y: L.BandY + (L.BandH-L.FontHeadH)/2})

	// 一行都放不進欄寬的段落整段通欄。
	var wide, narrow []helpSec
	for _, s := range secs {
		w := 0
		for _, b := range s.Body {
			if x := helpWidth(L.FontBodyW, b); x > w {
				w = x
			}
		}
		if s.Rest != "" {
			if x := helpWidth(L.FontBodyW, "　"+s.Rest); x > w {
				w = x
			}
		}
		if w > L.ColW {
			wide = append(wide, s)
		} else {
			narrow = append(narrow, s)
		}
	}

	hs := make([]int, len(narrow))
	for i, s := range narrow {
		hs[i] = L.HeadPitch + L.Pitch*(len(helpItems(L, s, L.ColW))-1)
	}
	best, cut := -1, 0
	for k := 1; k < len(narrow); k++ {
		lh, rh := L.Gap*(k-1), L.Gap*(len(narrow)-k-1)
		for _, v := range hs[:k] {
			lh += v
		}
		for _, v := range hs[k:] {
			rh += v
		}
		d := lh - rh
		if d < 0 {
			d = -d
		}
		if best < 0 || d < best {
			best, cut = d, k
		}
	}

	bottom := L.BodyY
	for ci, group := range [2][]helpSec{narrow[:cut], narrow[cut:]} {
		var y int
		ops, y = emitHelpGroup(ops, doc, group, colX[ci], L.ColW, L.BodyY)
		if y > bottom {
			bottom = y
		}
	}
	ops = append(ops, helpOp{Kind: opRect, X: colX[0] + L.ColW + (L.Gutter-L.Rule)/2, Y: L.BodyY,
		W: L.Rule, H: bottom - L.BodyY - L.RuleInset, Color: L.ColorRule})

	y := bottom + L.Gap
	ops = append(ops, helpOp{Kind: opRect, X: L.Margin, Y: y, W: right - L.Margin, H: L.Rule, Color: L.ColorRule})
	y += L.Gap
	ops, y = emitHelpGroup(ops, doc, wide, L.Margin, right-L.Margin, y)

	// 防拷那一列是**保留席位**：沒有題目時不畫，上面的座標一個都不變（docs/spec/022 §5）。
	if prot != "" {
		y += L.ProtGap
		ops = append(ops, helpOp{Kind: opRect, X: L.Margin, Y: y, W: right - L.Margin, H: L.Rule, Color: L.ColorRule})
		ops = appendHelpLine(ops, doc, prot, L.Margin, y+L.ProtGap, L.ColorProt, false)
	}
	return ops
}

// helpBottom 回內容底緣（最下面一個像素的下一列）。背景那一塊不算。
func helpBottom(L HelpLayout, ops []helpOp) int {
	bottom := 0
	for i, op := range ops {
		b := 0
		switch {
		case i == 0: // 背景
			continue
		case op.Kind == opRect:
			b = op.Y + op.H
		case op.Big:
			b = op.Y + L.FontHeadH
		default:
			b = op.Y + L.FontBodyH
		}
		if b > bottom {
			bottom = b
		}
	}
	return bottom
}

// helpInkBox 量一段字的墨跡框（相對於起筆點）。ASCII 的墨跡不靠左，鍵帽要對齊只能量墨跡，
// 不能用前進量推（docs/spec/022 §7）。
func helpInkBox(f *xlate.Font, s string) (int, int, int, int, bool) {
	rb := (f.W + 7) / 8
	lo, top, hi, bot, ok := 0, 0, 0, 0, false
	pen := 0
	for _, r := range s {
		adv := f.W
		if r < 0x80 {
			adv = f.W / 2
		}
		if g, has := f.Glyphs[r]; has {
			for gy := 0; gy < f.H; gy++ {
				for gx := 0; gx < f.W; gx++ {
					if g[gy*rb+gx/8]&(0x80>>(gx%8)) == 0 {
						continue
					}
					x := pen + gx
					if !ok {
						lo, top, hi, bot, ok = x, gy, x, gy, true
						continue
					}
					if x < lo {
						lo = x
					}
					if x > hi {
						hi = x
					}
					if gy < top {
						top = gy
					}
					if gy > bot {
						bot = gy
					}
				}
			}
		}
		pen += adv
	}
	return lo, top, hi, bot, ok
}

// helpFill 填一塊實心矩形（座標是設計座標，unit 是整數倍放大）。
func helpFill(dst []uint8, w, h, unit, x, y, rw, rh, color int) {
	rgb := egaRGB[color&15]
	for yy := y * unit; yy < (y+rh)*unit; yy++ {
		if yy < 0 || yy >= h {
			continue
		}
		for xx := x * unit; xx < (x+rw)*unit; xx++ {
			if xx < 0 || xx >= w {
				continue
			}
			i := yy*w + xx
			dst[4*i], dst[4*i+1], dst[4*i+2], dst[4*i+3] = rgb[0], rgb[1], rgb[2], 0xFF
		}
	}
}

// helpText 逐字畫字模。
func helpText(dst []uint8, w, h, unit int, f *xlate.Font, s string, x, y, color int) {
	rgb := egaRGB[color&15]
	rb := (f.W + 7) / 8
	pen := x
	for _, r := range s {
		adv := f.W
		if r < 0x80 {
			adv = f.W / 2
		}
		if g, has := f.Glyphs[r]; has {
			for gy := 0; gy < f.H; gy++ {
				for gx := 0; gx < f.W; gx++ {
					if g[gy*rb+gx/8]&(0x80>>(gx%8)) == 0 {
						continue
					}
					for py := 0; py < unit; py++ {
						for px := 0; px < unit; px++ {
							xx, yy := (pen+gx)*unit+px, (y+gy)*unit+py
							if xx < 0 || xx >= w || yy < 0 || yy >= h {
								continue
							}
							i := yy*w + xx
							dst[4*i], dst[4*i+1], dst[4*i+2], dst[4*i+3] = rgb[0], rgb[1], rgb[2], 0xFF
						}
					}
				}
			}
		}
		pen += adv
	}
}

// drawHelpOps 把指令清單光柵化到放大後的 RGBA。
func drawHelpOps(dst []uint8, w, h, unit int, f24, f16 *xlate.Font, ops []helpOp) {
	for _, op := range ops {
		f := f16
		if op.Big {
			f = f24
		}
		switch op.Kind {
		case opRect:
			helpFill(dst, w, h, unit, op.X, op.Y, op.W, op.H, op.Color)
		case opText:
			helpText(dst, w, h, unit, f, op.S, op.X, op.Y, op.Color)
		case opChip:
			if lo, top, hi, bot, ok := helpInkBox(f, op.S); ok {
				helpFill(dst, w, h, unit, op.X+lo-op.Pad, op.Y+top-op.Pad,
					hi-lo+1+2*op.Pad, bot-top+1+2*op.Pad, op.Color)
			}
			helpText(dst, w, h, unit, f, op.S, op.X, op.Y, op.Color2)
		}
	}
}

// helpBody16 是說明頁內文用的 cjk16。
//
// 前端交給 DrawTextPage 的只有標頭字型，內文字型由本檔自己找：先看與前端相同的預設
// （執行檔旁的 font/，退到工作目錄下的 font/），再看 help.json 那個資料夾的同層 font/。
// ⚠ 前端的 -font 指到別的地方時這裡找不到，會退回舊的整頁文字畫法並留一行 log——
// 說明頁靜默變成另一個樣子，比缺字更難查。
var (
	helpBody16     *xlate.Font
	helpTextDir    string
	helpBody16Once sync.Once
	helpWarnOnce   sync.Once
)

func helpBodyFont() *xlate.Font {
	helpBody16Once.Do(func() {
		dirs := []string{DataDir("font")}
		if helpTextDir != "" {
			dirs = append(dirs, filepath.Join(filepath.Dir(helpTextDir), "font"))
		}
		for _, d := range dirs {
			if f, err := xlate.LoadFont(filepath.Join(d, "cjk16.golemfnt")); err == nil {
				helpBody16 = f
				return
			}
		}
		log.Printf("說明頁在 %v 讀不到 cjk16 字型，改用舊的整頁文字版面", dirs)
	})
	return helpBody16
}

// helpPageLines 認出「這一批文字就是說明頁」，順便把防拷答案那一列挑出來。
//
// 前端把 help.json 的內容原樣交過來，防拷畫面時在尾端多一個空行與答案列
// （cmd/psychicwar 的 drawHelp）。版面由本檔決定，所以認頁面也在本檔。
func helpPageLines(lines []string) (bool, string) {
	d := helpDoc
	if d == nil || len(d.Lines) == 0 || len(lines) < len(d.Lines) {
		return false, ""
	}
	for i, s := range d.Lines {
		if lines[i] != s {
			return false, ""
		}
	}
	switch n := len(lines) - len(d.Lines); n {
	case 0:
		return true, ""
	case 2:
		if lines[len(d.Lines)] == "" && strings.HasPrefix(lines[len(d.Lines)+1], ProtectionLabel) {
			return true, lines[len(d.Lines)+1]
		}
	}
	return false, ""
}

// drawHelpPage 畫說明頁；畫不了（沒有字型、畫布比版面小、字型尺寸與版面對不上）回 false，
// 由呼叫端退回舊的整頁文字畫法。
func drawHelpPage(dst []uint8, w, h int, f24 *xlate.Font, prot string) bool {
	d := helpDoc
	if d == nil || f24 == nil {
		return false
	}
	L := d.Layout
	if L.PageW <= 0 || L.PageHeight <= 0 {
		return false
	}
	f16 := helpBodyFont()
	if f16 == nil {
		return false
	}
	if f24.W != L.FontHeadW || f24.H != L.FontHeadH || f16.W != L.FontBodyW || f16.H != L.FontBodyH {
		helpWarnOnce.Do(func() {
			log.Printf("說明頁的字型尺寸（%d×%d、%d×%d）與 help.json 的 layout（%d×%d、%d×%d）對不上",
				f24.W, f24.H, f16.W, f16.H, L.FontHeadW, L.FontHeadH, L.FontBodyW, L.FontBodyH)
		})
		return false
	}
	unit := w / L.PageW
	if unit < 1 || h/L.PageHeight < unit {
		return false
	}
	drawHelpOps(dst, w, h, unit, f24, f16, helpGeometry(d, prot))
	return true
}
