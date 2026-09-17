package psychicwar

import "testing"

// docs/spec/008 §4 第 1 項：key 換算。

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
	if _, _, ok := MenuSource(0x16816, 0); ok {
		t.Error("腳本緩衝區不該算 I_MENU")
	}
	// 反向對照：區域 0 時同一個位址對到 I_MENU00，沒有這則
	file, off, _ := MenuSource(0x12316+0x4F0, 0)
	if _, _, ok := FindLine(entries, file, off); ok {
		t.Error("區域 0 不該找到 I_MENU01 的訊息")
	}
}
