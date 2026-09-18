package psychicwar

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// docs/spec/018 §4 第 3、4 項：四個熱區各自對到正確的鍵，邊界差一像素就不同。
func TestHotspotAt(t *testing.T) {
	cases := []struct {
		x, y int
		want ebiten.Key
		in   bool
	}{
		{200, 8, ebiten.KeyUp, true},
		{172, 30, ebiten.KeyLeft, true},
		{220, 30, ebiten.KeyRight, true},
		{200, 56, ebiten.KeyDown, true},
		{200, 70, ebiten.KeyEscape, true},
		{167, 8, 0, false},  // 左緣外一像素
		{232, 8, 0, false},  // 右緣（半開區間，不含）
		{200, 80, 0, false}, // 下緣（不含）
		{20, 150, 0, false}, // 畫面左下角
	}
	for _, c := range cases {
		h := HotspotAt(c.x, c.y)
		if c.in {
			if h == nil {
				t.Errorf("(%d,%d) 應該在熱區裡", c.x, c.y)
				continue
			}
			if h.Key != c.want {
				t.Errorf("(%d,%d) 對到 %v，要 %v", c.x, c.y, h.Key, c.want)
			}
		} else if h != nil {
			t.Errorf("(%d,%d) 不該在熱區裡，卻對到 %s", c.x, c.y, h.Name)
		}
	}
	// 邊界：X0 在裡面、X0-1 在外面
	for _, h := range Hotspots {
		if HotspotAt(h.X0, h.Y0) == nil {
			t.Errorf("%s 的左上角應該算在裡面", h.Name)
		}
		if HotspotAt(h.X0-1, h.Y0) != nil && HotspotAt(h.X0-1, h.Y0).Name == h.Name {
			t.Errorf("%s 的左緣外一像素不該算進來", h.Name)
		}
	}
}
