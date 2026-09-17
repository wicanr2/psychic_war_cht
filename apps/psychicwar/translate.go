package psychicwar

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/overlay"
)

// 轉譯層的位址（docs/spec/008 §3.1）。全部是這一支 PW.EXE 專屬的。
const (
	codeSeg      = 0x0161
	addrLineLoop = 0x6289 // sub_16799 迴圈頭：BX 字元指標（ds）、CH 剩下字數
	addrGlyph    = 0x6158 // sub_16629 內 call sub_1884B：BX 仍是呼叫端的字元指標
	retLineLoop  = 0x628E // sub_16629 由 sub_16799 呼叫時的返回位址
	addrScrolled = 0x6276 // sub_16783 的 retn：訊息框已上移 2 像素
	addrCursor   = 0x610E // 低位元組 Y、高位元組 X，單位 4 像素
	linArea      = 0x16966

	linMenuH    = 0x13A16
	sizeMenuH   = 0xE00
	linMenuArea = 0x12316
	sizeMenuA   = 0xB00
)

// 訊息框（docs/spec/008 §3.4）。
const (
	boxX0, boxY0, boxX1, boxY1 = 8, 88, 136, 120
)

// TextEntry 是文本檔的一則（docs/spec/007 §4），轉譯層用得到的欄位。
type TextEntry struct {
	Key         string `json:"key"`
	Kind        string `json:"kind"`
	Width       int    `json:"width"`
	Translation string `json:"translation"`
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

// LineWidths 回一則的行寬（docs/spec/008 §3.3）。
func LineWidths(kind string) []int {
	if kind == "option" {
		return []int{10}
	}
	return []int{16, 15}
}

type printing struct {
	stamp *overlay.Stamp
	last  uint32 // 這一行最後一個字元的線性位址
}

// Translator 在原版印字時建立中文疊字（docs/spec/008）。
type Translator struct {
	Entries map[string]TextEntry
	Layer   overlay.Layer
	log     io.Writer

	lastLin uint32
	lastCH  uint8
	active  []printing
	noted   map[string]bool
}

// NewTranslator 建立轉譯層；log 可為 nil。
func NewTranslator(entries map[string]TextEntry, log io.Writer) *Translator {
	t := &Translator{Entries: entries, log: log, noted: map[string]bool{}}
	t.Layer.OnDrop = func(s *overlay.Stamp, why string) {
		t.event(map[string]any{"event": "drop", "key": s.Key, "why": why, "x": s.X, "y": s.Y})
	}
	return t
}

func (t *Translator) event(m map[string]any) {
	if t.log == nil {
		return
	}
	b, _ := json.Marshal(m)
	fmt.Fprintln(t.log, string(b))
}

func (t *Translator) once(kind, key string, extra map[string]any) {
	if t.noted[kind+key] {
		return
	}
	t.noted[kind+key] = true
	m := map[string]any{"event": kind, "key": key}
	for k, v := range extra {
		m[k] = v
	}
	t.event(m)
}

// Attach 在 oracle 上掛三個掛鉤。
func (t *Translator) Attach(o *oracle.Oracle) {
	o.OnCall(oracle.Addr{Seg: codeSeg, Off: addrLineLoop}, t.onLine)
	o.OnCall(oracle.Addr{Seg: codeSeg, Off: addrGlyph}, t.onGlyph)
	o.OnCall(oracle.Addr{Seg: codeSeg, Off: addrScrolled}, func(*oracle.Oracle) {
		t.Layer.Scroll(boxX0, boxY0, boxX1, boxY1, -2)
	})
}

func (t *Translator) onLine(o *oracle.Oracle) {
	r := o.Regs()
	lin := uint32(r.DS)*16 + uint32(r.BX)
	ch := uint8(r.CX >> 8)
	// 迴圈頭每字命中一次：位址加 1 且 CH 少 1 才是同一行。訊息第 2 行的起點緊接第 1 行，只看位址會併成一行。
	cont := lin == t.lastLin+1 && ch == t.lastCH-1
	t.lastLin, t.lastCH = lin, ch
	if cont {
		return
	}
	area := o.Word(oracle.Addr{Seg: linArea >> 4, Off: linArea & 0xF})
	file, off, ok := MenuSource(lin, area)
	if !ok {
		return
	}
	e, line, ok := FindLine(t.Entries, file, off)
	if !ok {
		return
	}
	if e.Translation == "" {
		t.once("missing-translation", e.Key, nil)
		return
	}
	lines, err := overlay.Layout(e.Translation, LineWidths(e.Kind))
	if err != nil {
		t.once("too-long", e.Key, nil)
	}
	cells := int(r.CX >> 8)
	cur := o.Word(oracle.Addr{Seg: codeSeg, Off: addrCursor})
	s := &overlay.Stamp{Key: e.Key, X: int(cur>>8) * 4, Y: int(cur&0xFF) * 4, Cells: cells, Text: lines[line]}
	t.Layer.Add(s)
	t.active = append(t.active, printing{stamp: s, last: lin + uint32(cells) - 1})
	t.event(map[string]any{"event": "stamp", "key": e.Key, "line": line, "x": s.X, "y": s.Y, "cells": cells, "text": string(s.Text)})
}

func (t *Translator) onGlyph(o *oracle.Oracle) {
	if len(t.active) == 0 {
		return
	}
	r := o.Regs()
	if o.Word(oracle.Addr{Seg: r.SS, Off: r.SP + 8}) != retLineLoop {
		return
	}
	lin := uint32(r.DS)*16 + uint32(r.BX)
	keep := t.active[:0]
	for _, p := range t.active {
		if p.last == lin {
			p.stamp.State = overlay.Pending
			continue
		}
		keep = append(keep, p)
	}
	t.active = keep
}

// Frame 在每次跑完一段機器時間後呼叫：定色、檢查失效。
func (t *Translator) Frame(o *oracle.Oracle) {
	_, _, rgb := o.ScreenRGB()
	t.Layer.Frame(o.Indexed(), rgb)
}

// MissingGlyph 記一筆缺字。
func (t *Translator) MissingGlyph(r rune) { t.once("missing-glyph", string(r), nil) }
