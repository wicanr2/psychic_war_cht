package translator

import (
	"reflect"
	"testing"

	"github.com/wicanr2/dosgolem/xlate"
)

// docs/spec/008 §4、009 §6 第 1 項：來源換算與透明格。

func TestMenuSourceAndFindLine(t *testing.T) {
	entries := map[string]TextEntry{
		"I_MENUH.BIN:0530":  {Key: "I_MENUH.BIN:0530", Kind: "message"},
		"I_MENU01.BIN:04F0": {Key: "I_MENU01.BIN:04F0", Kind: "message"},
		"I_MENUH.BIN:0A10":  {Key: "I_MENUH.BIN:0A10", Kind: "option"},
	}
	cases := []struct {
		lin  uint32
		area uint16
		key  string
		line int
	}{
		{0x13A16 + 0x530, 0, "I_MENUH.BIN:0530", 0},
		{0x13A16 + 0x540, 0, "I_MENUH.BIN:0530", 1},
		{0x12316 + 0x4F0, 1, "I_MENU01.BIN:04F0", 0},
		{0x13A16 + 0xA16, 0, "I_MENUH.BIN:0A10", 0},
	}
	for _, c := range cases {
		file, off, ok := MenuSource(c.lin, c.area)
		if !ok {
			t.Fatalf("%05X 不在 I_MENU 載入區", c.lin)
		}
		e, line, ok := FindLine(entries, file, off)
		if !ok || e.Key != c.key || line != c.line {
			t.Errorf("%05X → %s 第 %d 行（%v），要 %s 第 %d 行", c.lin, e.Key, line, ok, c.key, c.line)
		}
	}
	file, off, _ := MenuSource(0x12316+0x4F0, 0) // 反向對照：區域 0 對到 I_MENU00，沒有這則
	if _, _, ok := FindLine(entries, file, off); ok {
		t.Error("區域 0 不該找到 I_MENU01 的訊息")
	}
}

func testTranslator() *Translator {
	entries := map[string]TextEntry{
		"CODEH.BIN:02D5":    {Key: "CODEH.BIN:02D5", Kind: "inline", Original: "West    "},
		"CODE3.BIN:0100":    {Key: "CODE3.BIN:0100", Kind: "inline", Original: "{17}{19}Hello"},
		"I_MAP01.BIN:0228":  {Key: "I_MAP01.BIN:0228", Kind: "place", Original: "{95}{A5}{B5}{C5}{D5}{96}{A6} "},
		"I_ENMY00.BIN:0052": {Key: "I_ENMY00.BIN:0052", Kind: "enemy-name", Original: "Shulosu "},
		"PW.EXE:cs:4AFB":    {Key: "PW.EXE:cs:4AFB", Kind: "inline", Original: "{18}{15}I..I Surrender!{17}"},
		"PW.EXE:cs:65E5":    {Key: "PW.EXE:cs:65E5", Kind: "block", Width: 60, Original: "MATCH THE ALLY WITH   THE BASE TO GAIN   SECURITY CLEARANCE "},
		"PW.EXE:cs:66F7":    {Key: "PW.EXE:cs:66F7", Kind: "names", Width: 8, Original: " ZUPREEN"},
		"PW.EXE:cs:66FF":    {Key: "PW.EXE:cs:66FF", Kind: "names", Width: 8, Original: " SHULOSU"},
		"PW.EXE:cs:8851":    {Key: "PW.EXE:cs:8851", Kind: "line", Width: 20, Original: "   INSERT DISK-?    "},
		"PW.EXE:cs:B07D":    {Key: "PW.EXE:cs:B07D", Kind: "block", Width: 60, Original: "                       PRESS  ANY KEY        WHEN READY     "},
		"PW.EXE:cs:0BD7":    {Key: "PW.EXE:cs:0BD7", Kind: "line", Width: 20, Original: "IBM VERSION by      "},
	}
	return NewTranslator(entries, nil, nil, 3, nil)
}

func TestDSSource(t *testing.T) {
	tr := testTranslator()
	if e, ok := tr.DSSource(0x14816+0x2D5, []byte("West    "), 0); !ok || e.Key != "CODEH.BIN:02D5" {
		t.Errorf("CODEH：%v %v", e.Key, ok)
	}
	if e, ok := tr.DSSource(0x11C16+0x100, []byte("\x17\x19Hello"), 3); !ok || e.Key != "CODE3.BIN:0100" {
		t.Errorf("CODE3：%v %v", e.Key, ok)
	}
	if _, ok := tr.DSSource(0x11C16+0x100, []byte("\x17\x19Hello"), 2); ok {
		t.Error("區域 2 不該對到 CODE3")
	}
	if _, ok := tr.DSSource(0x11C16+0x700, nil, 3); ok {
		t.Error("載入視窗 700h 之外不該算 CODE3")
	}
	place := []byte{0x95, 0xA5, 0xB5, 0xC5, 0xD5, 0x96, 0xA6, ' '}
	if e, ok := tr.DSSource(0x16816, place, 1); !ok || e.Key != "I_MAP01.BIN:0228" {
		t.Errorf("地點名稱：%v %v", e.Key, ok)
	}
	if _, ok := tr.DSSource(0x16818, []byte("  30"), 1); ok {
		t.Error("數字不該換算成文本")
	}
}

func TestCSSource(t *testing.T) {
	tr := testTranslator()
	if e, ok := tr.CSSource(0x3A4C, []byte("Shulosu "), 0); !ok || e.Key != "I_ENMY00.BIN:0052" {
		t.Errorf("敵人名稱：%v %v", e.Key, ok)
	}
	if _, ok := tr.CSSource(0x3A4C, []byte("Shulosu "), 1); ok {
		t.Error("區域 1 不該對到 I_ENMY00")
	}
	if e, ok := tr.CSSource(0x4AFB, nil, 0); !ok || e.Key != "PW.EXE:cs:4AFB" {
		t.Errorf("內嵌字串：%v %v", e.Key, ok)
	}
}

func TestSmallPointerAndEntry(t *testing.T) {
	tr := testTranslator()
	mem := func(off uint16, n int) []byte { return []byte("   INSERT DISK-?    ")[:n] }
	cases := []struct {
		ret, bx, dx      uint16
		key              string
		line, col, width int
	}{
		{0xB02A, 0x65E5, 0, "PW.EXE:cs:65E5", 0, 0, 20},
		{0xB02A, 0x65E5 + 45, 0, "PW.EXE:cs:65E5", 2, 5, 20},
		{0x653A, 0x66FF + 7, 0, "PW.EXE:cs:66FF", 0, 7, 8},
		{0x654E, 0, 0x66F7, "PW.EXE:cs:66F7", 0, 0, 8},
		{0x0BB1, 0x0BD7 + 1, 0, "PW.EXE:cs:0BD7", 0, 0, 20},
		{0xB02A, 0xB07D + 3, 0, "PW.EXE:cs:8851", 0, 3, 20}, // B07D 第 1 行：內容比對
		{0xB02A, 0xB07D + 21, 0, "PW.EXE:cs:B07D", 1, 1, 20},
	}
	for _, c := range cases {
		p, ok := SmallPointer(c.ret, c.bx, c.dx)
		if !ok {
			t.Fatalf("返回位址 %04X 應該認得", c.ret)
		}
		e, line, col, width, ok := tr.SmallEntry(p, mem)
		if !ok || e.Key != c.key || line != c.line || col != c.col || width != c.width {
			t.Errorf("%04X/%04X → %s 行 %d 欄 %d 寬 %d（%v），要 %s %d %d %d", c.ret, p, e.Key, line, col, width, ok, c.key, c.line, c.col, c.width)
		}
	}
	if _, ok := SmallPointer(0xB055, 0, 0); ok {
		t.Error("B055 是輸入回顯，不該有指標")
	}
	if _, _, _, _, ok := tr.SmallEntry(0x65E5+60, mem); ok {
		t.Error("區塊之外不該找到")
	}
}

func TestTransparentCellsAndStamp(t *testing.T) {
	if got := TransparentCells([]rune("  [甲 ]"), []byte("  [  ]")); !reflect.DeepEqual(got, []bool{true, true, false, false, true, false}) {
		t.Errorf("透明格 %v", got)
	}
	tr := testTranslator()
	e := TextEntry{Key: "PW.EXE:cs:8AEF", Kind: "block", Width: 60,
		Original:    "  ENTER YOUR NAME:                           [        ]     ",
		Translation: "  請輸入名字：\n\n     [        ]"}
	tr.Entries[e.Key] = e
	s := tr.NewStamp(e, 2, 100, 103, 20, true)
	if s == nil || s.CellW != 6 || s.CellH != 7 || s.GlyphX != 1 || s.GlyphY != 3 || s.GlyphScale != 1 {
		t.Fatalf("小字型疊字幾何 %+v", s)
	}
	for i := 6; i < 14; i++ {
		if !s.Transparent[i] {
			t.Errorf("輸入框內第 %d 格應該透明", i)
		}
	}
	if s.Transparent[5] || s.Transparent[14] {
		t.Error("方括號不該透明")
	}
	if tr.NewStamp(TextEntry{Key: "x", Kind: "option", Original: "Leave"}, 0, 0, 0, 10, false) != nil {
		t.Error("沒有譯文不該建疊字")
	}
	_ = xlate.Printing
}

func TestLineWidths(t *testing.T) {
	cases := []struct {
		e    TextEntry
		want []int
	}{
		{TextEntry{Kind: "message"}, []int{16, 15}},
		{TextEntry{Kind: "option"}, []int{10}},
		{TextEntry{Kind: "block", Width: 160}, []int{20, 20, 20, 20, 20, 20, 20, 20}},
		{TextEntry{Kind: "inline", Original: "{17}{19}Hello"}, []int{5}},
		{TextEntry{Kind: "place", Original: "{95}{A5}{B5}{C5}{D5}{96}{A6} "}, []int{8}},
	}
	for _, c := range cases {
		if got := LineWidths(c.e); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s：%v，要 %v", c.e.Kind, got, c.want)
		}
	}
}
