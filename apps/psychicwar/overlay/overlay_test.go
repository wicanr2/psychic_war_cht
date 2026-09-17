package overlay

import (
	"errors"
	"testing"
)

// docs/spec/008 §4 第 1 項。

func TestLayout(t *testing.T) {
	got, err := Layout("這裡無法前進。", []int{16, 15})
	if err != nil || string(got[0]) != "這裡無法前進。" || len(got[1]) != 0 {
		t.Errorf("7 字：%q %q %v", string(got[0]), string(got[1]), err)
	}
	s18 := "一二三四五六七八九十一二三四五六七八"
	got, err = Layout(s18, []int{16, 15})
	if err != nil || len(got[0]) != 16 || string(got[1]) != "七八" {
		t.Errorf("18 字：%d %q %v", len(got[0]), string(got[1]), err)
	}
	got, err = Layout(s18+s18[:14*3], []int{16, 15}) // 32 字
	if !errors.Is(err, ErrTooLong) || len(got[0]) != 16 || len(got[1]) != 15 {
		t.Errorf("32 字：%d %d %v", len(got[0]), len(got[1]), err)
	}
	got, _ = Layout("甲\n乙", []int{16, 15})
	if string(got[0]) != "甲" || string(got[1]) != "乙" {
		t.Errorf("換行：%q %q", string(got[0]), string(got[1]))
	}
}

func TestScroll(t *testing.T) {
	var dropped []string
	l := &Layer{OnDrop: func(s *Stamp, why string) { dropped = append(dropped, s.Key+":"+why) }}
	in := &Stamp{Key: "in", X: 8, Y: 112, Cells: 16}
	top := &Stamp{Key: "top", X: 8, Y: 88, Cells: 16}
	out := &Stamp{Key: "out", X: 200, Y: 112, Cells: 4}
	l.Stamps = []*Stamp{in, top, out}
	l.Scroll(8, 88, 136, 120, -2)
	if in.Y != 110 || out.Y != 112 || len(l.Stamps) != 2 || len(dropped) != 1 || dropped[0] != "top:scroll" {
		t.Errorf("in.Y=%d out.Y=%d 剩 %d 筆 移除 %v", in.Y, out.Y, len(l.Stamps), dropped)
	}
}

func TestInvalidateAfterThreeFrames(t *testing.T) {
	idx := make([]uint8, ScreenW*ScreenH)
	rgb := make([]uint8, 3*ScreenW*ScreenH)
	idx[112*ScreenW+9] = 11
	l := &Layer{}
	s := &Stamp{Key: "k", X: 8, Y: 112, Cells: 2, State: Pending}
	l.Stamps = []*Stamp{s}
	l.Frame(idx, rgb)
	if s.State != Shown {
		t.Fatal("定色後應該顯示")
	}
	changed := append([]uint8(nil), idx...)
	changed[113*ScreenW+10] = 5
	l.Frame(changed, rgb)
	l.Frame(changed, rgb)
	if len(l.Stamps) != 1 {
		t.Fatal("連續 2 幀不同就被移除")
	}
	l.Frame(idx, rgb) // 恢復一次，重新計數
	l.Frame(changed, rgb)
	l.Frame(changed, rgb)
	if len(l.Stamps) != 1 {
		t.Fatal("恢復後又 2 幀不同就被移除")
	}
	l.Frame(changed, rgb)
	if len(l.Stamps) != 0 {
		t.Fatal("連續 3 幀不同應該移除")
	}
}

func TestColors(t *testing.T) {
	cell := make([]uint8, 64)
	for i := 0; i < 14; i++ {
		cell[i*3] = 11
	}
	if bg, fg := Colors(cell); bg != 0 || fg != 11 {
		t.Errorf("背景 %d 前景 %d，要 0、11", bg, fg)
	}
	if bg, fg := Colors(make([]uint8, 64)); bg != 0 || fg != 0 {
		t.Errorf("只有一種色號：背景 %d 前景 %d", bg, fg)
	}
}

func TestAddReplacesOverlap(t *testing.T) {
	l := &Layer{}
	l.Add(&Stamp{Key: "a", X: 8, Y: 112, Cells: 16})
	l.Add(&Stamp{Key: "b", X: 8, Y: 104, Cells: 16})
	l.Add(&Stamp{Key: "c", X: 16, Y: 112, Cells: 3})
	if len(l.Stamps) != 2 || l.Stamps[0].Key != "b" || l.Stamps[1].Key != "c" {
		t.Errorf("剩 %d 筆", len(l.Stamps))
	}
}

func TestDrawGlyph(t *testing.T) {
	var g [72]byte
	g[0] = 0x80 // 左上角一點
	font := Font{'一': g}
	l := &Layer{Stamps: []*Stamp{{X: 8, Y: 112, Cells: 2, Text: []rune("一x"), State: Shown, FG: [3]uint8{1, 2, 3}, BG: [3]uint8{9, 9, 9}}}}
	dst := make([]uint8, 4*960*600)
	var miss []rune
	l.Draw(dst, 3, font, func(r rune) { miss = append(miss, r) })
	at := func(x, y int) [3]uint8 { i := 4 * (y*960 + x); return [3]uint8{dst[i], dst[i+1], dst[i+2]} }
	if at(24, 336) != [3]uint8{1, 2, 3} || at(25, 336) != [3]uint8{9, 9, 9} || at(24+47, 336+23) != [3]uint8{9, 9, 9} {
		t.Errorf("字模點 %v、旁邊 %v、角落 %v", at(24, 336), at(25, 336), at(71, 359))
	}
	if len(miss) != 1 || miss[0] != 'x' {
		t.Errorf("缺字 %q", string(miss))
	}
}
