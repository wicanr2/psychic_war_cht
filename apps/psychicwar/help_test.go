package psychicwar

import "testing"

// docs/spec/017 §3：非 ASCII 的文字輸入要被認出來（前端據此顯示提示）。
func TestNonASCII(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"TESTER", false},
		{"play3", false},
		{"中", true},
		{"a中b", true},
		{"　", true}, // 全形空白也是非 ASCII
	}
	for _, c := range cases {
		if got := NonASCII([]rune(c.in)); got != c.want {
			t.Errorf("NonASCII(%q) = %v，要 %v", c.in, got, c.want)
		}
	}
}
