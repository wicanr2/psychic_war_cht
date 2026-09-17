// Package translator 是《銀河超能力戰記》的轉譯層：位址、文本 key 換算與掛鉤（docs/spec/008、009）。不依賴 Ebiten。
package translator

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/dosgolem/xlate"
)

// 轉譯層的位址（docs/spec/008 §3.1、009 §4）。全部是這一支 PW.EXE 專屬的。
const (
	codeSeg = 0x0161
	codeLin = 0x1610 // 段 0161 的線性起點

	addrLineLoop = 0x6289 // A1 sub_16799 迴圈頭：BX 字元指標（ds）、CH 剩下字數
	addrDSLoop   = 0x62A2 // A2 sub_167B2 迴圈頭：ds:BX 到 0
	addrCSLoop   = 0x62AF // A3 sub_167BF 迴圈頭：cs:BX 到 0
	addrGlyph    = 0x6158 // sub_16629 內 call sub_1884B：[SS:SP+8] 是 sub_16629 的返回位址
	retLineLoop  = 0x628E // A1 迴圈呼叫 sub_16629 的返回位址（BX ＝ 字元）
	retDSLoop    = 0x62AD // A2（迴圈先 inc bx：BX ＝ 字元 ＋1）
	retCSLoop    = 0x62BB // A3（BX ＝ 字元 ＋1）
	addrScroll   = 0x6273 // sub_16783 進入點：訊息框開始上移 2 像素（逐列搬動，可能跨好幾幀）
	addrScrolled = 0x6276 // sub_16783 的 retn：訊息框已上移 2 像素
	addrCursor   = 0x610E // 低位元組 Y、高位元組 X，單位 4 像素
	addrSmall    = 0xB0F1 // B sub_1B601：AL 字碼、SI／DI 位置、[SS:SP] 返回位址
	retSmallEcho = 0xB055 // 輸入回顯與選擇游標
	addrB07D     = 0xB07D // B016 區塊，第 1 行由 B058 從 20 bytes 的行複製進來
	addrEnemyBuf = 0x3A4C // 戰鬥時從 I_ENMY 複製來的敵人名稱
	linArea      = 0x16966

	linMenuH    = 0x13A16
	sizeMenuH   = 0xE00
	linMenuArea = 0x12316
	sizeMenuA   = 0xB00
	linCodeArea = 0x11C16
	sizeCodeA   = 0x700
	linCodeH    = 0x14816
	sizeCodeH   = 0x1200
	linScript   = 0x16816
	sizeScript  = 0x100
)

// 訊息框（docs/spec/008 §3.4）。
const boxX0, boxY0, boxX1, boxY1 = 8, 88, 136, 120

// TextEntry 是文本檔的一則（docs/spec/007 §4），轉譯層用得到的欄位。
type TextEntry struct {
	Key         string `json:"key"`
	Kind        string `json:"kind"`
	Font        string `json:"font"`
	Width       int    `json:"width"`
	Original    string `json:"original"`
	Translation string `json:"translation"`
	SameAs      string `json:"same_as"`
	// Translatable 為 false：原文沒有可見字元（全是空白或控制碼），不需要譯文，照原版顯示。缺欄位視為 true。
	Translatable *bool `json:"translatable"`
}

// LoadText 讀 text/ 下 schema 為 psychic-war-text/1 的檔。
func LoadText(dir string) (map[string]TextEntry, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	out := map[string]TextEntry{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var doc struct {
			Schema  string      `json:"schema"`
			Entries []TextEntry `json:"entries"`
		}
		if err := json.Unmarshal(b, &doc); err != nil {
			return nil, fmt.Errorf("%s：%w", f, err)
		}
		if doc.Schema != "psychic-war-text/1" {
			continue
		}
		for _, e := range doc.Entries {
			out[e.Key] = e
		}
	}
	return out, nil
}

// DecodeShown 把原文表示法（`{XX}` 是非 ASCII 可印位元組）換回位元組。
func DecodeShown(s string) []byte {
	var out []byte
	for i := 0; i < len(s); {
		if s[i] == '{' && i+3 < len(s) && s[i+3] == '}' {
			var v byte
			if _, err := fmt.Sscanf(s[i+1:i+3], "%02X", &v); err == nil {
				out = append(out, v)
				i += 4
				continue
			}
		}
		out = append(out, s[i])
		i++
	}
	return out
}

func printable(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for _, c := range b {
		if c >= 0x20 {
			out = append(out, c)
		}
	}
	return out
}

// LineWidths 回一則的行寬（docs/spec/009 §3；與 tools/text_extract.py line_widths 相同）。
func LineWidths(e TextEntry) []int {
	switch e.Kind {
	case "message", "menu":
		return []int{16, 15}
	case "option":
		return []int{10}
	case "block":
		w := make([]int, e.Width/20)
		for i := range w {
			w[i] = 20
		}
		return w
	}
	return []int{len(printable(DecodeShown(e.Original)))}
}

// lineBytes 回原文第 line 行的可印位元組（選項標籤補空白到 10）。
func lineBytes(e TextEntry, line int) []byte {
	raw := printable(DecodeShown(e.Original))
	if e.Kind == "option" {
		for len(raw) < 10 {
			raw = append(raw, ' ')
		}
	}
	ws := LineWidths(e)
	if line >= len(ws) {
		return nil
	}
	start := 0
	for i := 0; i < line; i++ {
		start += ws[i]
	}
	end := start + ws[line]
	if start > len(raw) {
		return nil
	}
	if end > len(raw) {
		end = len(raw)
	}
	return raw[start:end]
}

// TransparentCells：譯文這一格是半形空白、原文同一格也是空白 → 透明（docs/spec/009 §3）。
func TransparentCells(text []rune, orig []byte) []bool {
	out := make([]bool, len(orig))
	for i := range orig {
		r := ' '
		if i < len(text) {
			r = text[i]
		}
		out[i] = r == ' ' && orig[i] == ' '
	}
	return out
}

// MenuSource 把 I_MENU 載入區的線性位址換成來源檔與偏移（docs/spec/008 §3.1）。
func MenuSource(lin uint32, area uint16) (file string, off int, ok bool) {
	switch {
	case lin >= linMenuH && lin < linMenuH+sizeMenuH:
		return "I_MENUH.BIN", int(lin - linMenuH), true
	case lin >= linMenuArea && lin < linMenuArea+sizeMenuA:
		return fmt.Sprintf("I_MENU%02d.BIN", area), int(lin - linMenuArea), true
	}
	return "", 0, false
}

// FindLine 依偏移找 key 與這是第幾行（docs/spec/008 §3.2）：o（第 1 行）、o−16（第 2 行）、o−6（選項）。
func FindLine(entries map[string]TextEntry, file string, off int) (e TextEntry, line int, ok bool) {
	try := func(o int, kinds ...string) (TextEntry, bool) {
		e, ok := entries[fmt.Sprintf("%s:%04X", file, o)]
		if !ok {
			return e, false
		}
		for _, k := range kinds {
			if e.Kind == k {
				return e, true
			}
		}
		return e, false
	}
	if e, ok := try(off, "message", "menu", "option"); ok {
		return e, 0, true
	}
	if e, ok := try(off-16, "message", "menu"); ok {
		return e, 1, true
	}
	if e, ok := try(off-6, "option"); ok {
		return e, 0, true
	}
	return TextEntry{}, 0, false
}

type exeEntry struct {
	addr uint16
	e    TextEntry
}

// exeWidths 回 PW.EXE 一則在記憶體裡的行寬（block 與 20 bytes 的行以 20 切；其他是整則一行）。
func exeWidths(e TextEntry) []int {
	if e.Kind == "block" {
		return LineWidths(e)
	}
	return []int{len(printable(DecodeShown(e.Original)))}
}

// Translator 在原版印字時建立中文疊字（docs/spec/008、009）。
type Translator struct {
	Entries map[string]TextEntry
	Layer   xlate.Layer
	Font24  *xlate.Font
	Font16  *xlate.Font
	Scale   int
	log     io.Writer

	exe       []exeEntry
	byContent map[string]map[string]string // 來源檔 → 去尾空白的可印原文 → key

	trackMenu, trackDS, trackCS xlate.LineTracker
	active                      []printing
	noted                       map[string]bool
	scrolling                   bool // 訊息框捲動一步進行中（addrScroll 到 addrScrolled）
}

type printing struct {
	stamp *xlate.Stamp
	ret   uint16 // A 路徑：哪一個迴圈畫的
	last  uint32 // A 路徑：最後一個可印字元的線性位址
	key   string // B 路徑：條目
	line  int
}

// NewTranslator 建立轉譯層；log 可為 nil。scale 是放大倍率（3 的倍數）。
func NewTranslator(entries map[string]TextEntry, font24, font16 *xlate.Font, scale int, log io.Writer) *Translator {
	t := &Translator{Entries: entries, Font24: font24, Font16: font16, Scale: scale, log: log,
		noted: map[string]bool{}, byContent: map[string]map[string]string{}}
	if font24 != nil {
		font24.Name = "cjk24"
	}
	if font16 != nil {
		font16.Name = "cjk16"
	}
	t.Layer.OnDrop = func(s *xlate.Stamp, why string) {
		t.event(map[string]any{"event": "drop", "key": s.Key, "why": why, "x": s.X, "y": s.Y})
	}
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys) // 內容比對撞名時取固定的一則
	for _, k := range keys {
		e := entries[k]
		src := k[:strings.Index(k, ":")]
		if src == "PW.EXE" {
			var off uint16
			fmt.Sscanf(k[len("PW.EXE:cs:"):], "%04X", &off)
			t.exe = append(t.exe, exeEntry{addr: off, e: e})
			if e.Kind == "line" && e.Width == 20 {
				t.content(src, e)
			}
			continue
		}
		if e.Kind == "place" || e.Kind == "enemy-name" {
			t.content(src, e)
		}
	}
	sort.Slice(t.exe, func(i, j int) bool { return t.exe[i].addr < t.exe[j].addr })
	return t
}

func (t *Translator) content(src string, e TextEntry) {
	if t.byContent[src] == nil {
		t.byContent[src] = map[string]string{}
	}
	c := strings.TrimRight(string(printable(DecodeShown(e.Original))), " ")
	if _, dup := t.byContent[src][c]; !dup {
		t.byContent[src][c] = e.Key
	}
}

func (t *Translator) event(m map[string]any) {
	if t.log == nil {
		return
	}
	b, _ := json.Marshal(m)
	fmt.Fprintln(t.log, string(b))
}

func (t *Translator) once(kind, key string) {
	if t.noted[kind+key] {
		return
	}
	t.noted[kind+key] = true
	t.event(map[string]any{"event": kind, "key": key})
}

// translation 回一則實際用的譯文（same_as 用根的譯文）。
func (t *Translator) translation(e TextEntry) string {
	if e.SameAs != "" {
		return t.Entries[e.SameAs].Translation
	}
	return e.Translation
}

func (t *Translator) byName(src string, str []byte) (TextEntry, bool) {
	k, ok := t.byContent[src][strings.TrimRight(string(printable(str)), " ")]
	if !ok {
		return TextEntry{}, false
	}
	return t.Entries[k], true
}

// DSSource 換算 A2 的來源（docs/spec/009 §4.1）。str 是從起點讀到 0 的位元組。
func (t *Translator) DSSource(lin uint32, str []byte, area uint16) (TextEntry, bool) {
	switch {
	case lin >= linCodeArea && lin < linCodeArea+sizeCodeA:
		e, ok := t.Entries[fmt.Sprintf("CODE%d.BIN:%04X", area, lin-linCodeArea)]
		return e, ok
	case lin >= linCodeH && lin < linCodeH+sizeCodeH:
		e, ok := t.Entries[fmt.Sprintf("CODEH.BIN:%04X", lin-linCodeH)]
		return e, ok
	case lin >= linScript && lin < linScript+sizeScript:
		return t.byName(fmt.Sprintf("I_MAP%02d.BIN", area), str)
	}
	return TextEntry{}, false
}

// CSSource 換算 A3 的來源（docs/spec/009 §4.2）。
func (t *Translator) CSSource(off uint16, str []byte, area uint16) (TextEntry, bool) {
	if off == addrEnemyBuf {
		return t.byName(fmt.Sprintf("I_ENMY%02d.BIN", area), str)
	}
	e, ok := t.Entries[fmt.Sprintf("PW.EXE:cs:%04X", off)]
	return e, ok
}

// SmallPointer 依小字型呼叫端的返回位址取字元指標（docs/spec/009 §4.3）。
func SmallPointer(ret, bx, dx uint16) (p uint16, ok bool) {
	switch ret {
	case 0xB02A, 0x07A0, 0x07D1, 0x653A:
		return bx, true
	case 0x0BB1, 0x1259:
		return bx - 1, true
	case 0x654E:
		return dx, true
	}
	return 0, false
}

// SmallEntry 找指標 p 所在的 PW.EXE 條目與行、欄；mem 讀段 0161 的記憶體（B07D 第 1 行的內容比對用）。
func (t *Translator) SmallEntry(p uint16, mem func(off uint16, n int) []byte) (e TextEntry, line, col, width int, ok bool) {
	i := sort.Search(len(t.exe), func(i int) bool { return t.exe[i].addr > p }) - 1
	if i < 0 {
		return
	}
	x := t.exe[i]
	ws := exeWidths(x.e)
	off, total := int(p-x.addr), 0
	for _, w := range ws {
		total += w
	}
	if off >= total {
		return
	}
	for line = 0; off >= ws[line]; line++ {
		off -= ws[line]
	}
	e, col, width = x.e, off, ws[line]
	if x.addr == addrB07D && line == 0 {
		src, found := t.byContent["PW.EXE"][strings.TrimRight(string(mem(addrB07D, 20)), " ")]
		if !found {
			return TextEntry{}, 0, 0, 0, false
		}
		e = t.Entries[src]
	}
	return e, line, col, width, true
}

// Attach 在 oracle 上掛掛鉤。
func (t *Translator) Attach(o *oracle.Oracle) {
	at := func(off uint16) oracle.Addr { return oracle.Addr{Seg: codeSeg, Off: off} }
	o.OnCall(at(addrLineLoop), t.onMenuLine)
	o.OnCall(at(addrDSLoop), func(o *oracle.Oracle) { t.onStringLoop(o, addrDSLoop) })
	o.OnCall(at(addrCSLoop), func(o *oracle.Oracle) { t.onStringLoop(o, addrCSLoop) })
	o.OnCall(at(addrGlyph), t.onGlyph)
	o.OnCall(at(addrScroll), func(*oracle.Oracle) { t.scrolling = true })
	o.OnCall(at(addrScrolled), func(*oracle.Oracle) {
		t.scrolling = false
		t.Layer.Scroll(boxX0, boxY0, boxX1, boxY1, -2)
	})
	// 捲動一步沒做完時，訊息框裡是搬到一半的畫面，不能拿來判斷失效（spec 009 §4.6）。
	t.Layer.Frozen = func(s *xlate.Stamp) bool {
		return t.scrolling && s.X >= boxX0 && s.X+s.Cells*s.CellW <= boxX1 && s.Y >= boxY0 && s.Y < boxY1
	}
	o.OnCall(at(addrSmall), t.onSmall)
}

func area(o *oracle.Oracle) uint16 {
	return o.Word(oracle.Addr{Seg: uint16(linArea >> 4), Off: uint16(linArea & 0xF)})
}

// NewStamp 建一筆疊字（沒有譯文回 nil）；A 路徑 small=false，B 路徑 small=true（docs/spec/009 §2）。
func (t *Translator) NewStamp(e TextEntry, line, x, y, cells int, small bool) *xlate.Stamp {
	if e.Translatable != nil && !*e.Translatable {
		return nil // 空白行（清掉舊字用）：原版像素照常顯示，不算缺譯文（docs/spec/009 §3）
	}
	tr := t.translation(e)
	if tr == "" {
		t.once("missing-translation", e.Key)
		return nil
	}
	if tr == e.Original {
		return nil // 保留原文：原版像素照常顯示（docs/spec/009 §3）
	}
	ws := LineWidths(e)
	if e.Kind == "line" && len(ws) == 1 && ws[0] != cells {
		ws = []int{cells}
	}
	lines, err := xlate.Layout(tr, ws)
	if err != nil {
		t.once("too-long", e.Key)
	}
	var text []rune
	if line < len(lines) {
		text = lines[line]
	}
	s := &xlate.Stamp{Key: e.Key, X: x, Y: y, Cells: cells, Text: text,
		Transparent: TransparentCells(text, lineBytes(e, line))}
	if small {
		s.CellW, s.CellH, s.Font = 6, 7, t.Font16
		s.GlyphX, s.GlyphY, s.GlyphScale = t.Scale/3, 3*t.Scale/3, t.Scale/3
	} else {
		s.CellW, s.CellH, s.Font = 8, 8, t.Font24
	}
	t.Layer.Add(s)
	t.event(map[string]any{"event": "stamp", "key": e.Key, "line": line, "x": x, "y": y, "cells": cells, "text": string(text)})
	return s
}

func cursor(o *oracle.Oracle) (x, y int) {
	cur := o.Word(oracle.Addr{Seg: codeSeg, Off: addrCursor})
	return int(cur>>8) * 4, int(cur&0xFF) * 4
}

func (t *Translator) onMenuLine(o *oracle.Oracle) {
	r := o.Regs()
	lin := uint32(r.DS)*16 + uint32(r.BX)
	cells := int(r.CX >> 8)
	if !t.trackMenu.Hit(lin, cells, true) {
		return
	}
	file, off, ok := MenuSource(lin, area(o))
	if !ok {
		return
	}
	e, line, ok := FindLine(t.Entries, file, off)
	if !ok {
		return
	}
	x, y := cursor(o)
	if s := t.NewStamp(e, line, x, y, cells, false); s != nil {
		t.active = append(t.active, printing{stamp: s, ret: retLineLoop, last: lin + uint32(cells) - 1})
	}
}

func (t *Translator) onStringLoop(o *oracle.Oracle, loop uint16) {
	r := o.Regs()
	lin := uint32(r.DS)*16 + uint32(r.BX)
	tr := &t.trackDS
	if loop == addrCSLoop {
		lin, tr = codeLin+uint32(r.BX), &t.trackCS
	}
	if !tr.Hit(lin, 0, false) {
		return
	}
	str := o.Bytes(oracle.Addr{Seg: uint16(lin >> 4), Off: uint16(lin & 0xF)}, 128)
	for i, c := range str {
		if c == 0 {
			str = str[:i]
			break
		}
	}
	var e TextEntry
	var ok bool
	if loop == addrCSLoop {
		e, ok = t.CSSource(r.BX, str, area(o))
	} else {
		e, ok = t.DSSource(lin, str, area(o))
	}
	if !ok {
		return
	}
	last, cells := -1, 0
	for i, c := range str {
		if c < 0x10 {
			t.once("unsupported-control", e.Key)
			return
		}
		if c >= 0x20 {
			last, cells = i, cells+1
		}
	}
	if cells == 0 {
		return
	}
	x, y := cursor(o)
	ret := uint16(retDSLoop)
	if loop == addrCSLoop {
		ret = retCSLoop
	}
	if s := t.NewStamp(e, 0, x, y, cells, false); s != nil {
		t.active = append(t.active, printing{stamp: s, ret: ret, last: lin + uint32(last)})
	}
}

func (t *Translator) onGlyph(o *oracle.Oracle) {
	if len(t.active) == 0 {
		return
	}
	r := o.Regs()
	ret := o.Word(oracle.Addr{Seg: r.SS, Off: r.SP + 8})
	var lin uint32
	switch ret {
	case retLineLoop:
		lin = uint32(r.DS)*16 + uint32(r.BX)
	case retDSLoop:
		lin = uint32(r.DS)*16 + uint32(r.BX) - 1
	case retCSLoop:
		lin = codeLin + uint32(r.BX) - 1
	default:
		return
	}
	t.finish(func(p printing) bool { return p.key == "" && p.ret == ret && p.last == lin })
}

func (t *Translator) finish(match func(printing) bool) {
	keep := t.active[:0]
	for _, p := range t.active {
		if match(p) {
			if p.stamp.State == xlate.Printing {
				p.stamp.State = xlate.Pending
			}
			continue
		}
		keep = append(keep, p)
	}
	t.active = keep
}

func (t *Translator) onSmall(o *oracle.Oracle) {
	r := o.Regs()
	ret := o.Word(oracle.Addr{Seg: r.SS, Off: r.SP})
	if ret == retSmallEcho {
		return
	}
	p, ok := SmallPointer(ret, r.BX, r.DX)
	if !ok {
		t.once("unknown-caller", fmt.Sprintf("B0F1 由 %04X", ret))
		return
	}
	mem := func(off uint16, n int) []byte { return o.Bytes(oracle.Addr{Seg: codeSeg, Off: off}, n) }
	e, line, col, width, ok := t.SmallEntry(p, mem)
	if !ok {
		return
	}
	if col == 0 {
		if s := t.NewStamp(e, line, int(r.SI), int(r.DI), width, true); s != nil {
			t.active = append(t.active, printing{stamp: s, key: e.Key, line: line})
		}
	}
	if col == width-1 {
		t.finish(func(pr printing) bool { return pr.key == e.Key && pr.line == line })
	}
}

// Frame 在每段機器時間之後呼叫：定色、檢查失效。
func (t *Translator) Frame(o *oracle.Oracle) {
	_, _, rgb := o.ScreenRGB()
	t.Layer.Frame(o.Indexed(), rgb)
}

// MissingGlyph 記一筆缺字。
func (t *Translator) MissingGlyph(r rune) { t.once("missing-glyph", string(r)) }

// Fonts 回快照還原用的字型表。
func (t *Translator) Fonts() map[string]*xlate.Font {
	return map[string]*xlate.Font{"cjk24": t.Font24, "cjk16": t.Font16}
}
