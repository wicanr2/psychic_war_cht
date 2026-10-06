package theme

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func overTestTheme(t *testing.T) (*Theme, []byte) {
	t.Helper()
	orig := os.Getenv("PSYCHICWAR_TEST_ORIG")
	if orig == "" {
		t.Skip("真實素材驗證需明示 PSYCHICWAR_TEST_ORIG")
	}
	px, err := loadOver0(orig)
	if err != nil {
		t.Fatal(err)
	}
	if len(px) != 4096 {
		t.Fatal("OVER 原圖長度不符")
	}
	dir := t.TempDir()
	themeTestPNG(t, dir, "over.png", 192, 192)
	m := ThemeManifest{Schema: "psychic-war-theme/1", Name: "over", Scale: 3, Entries: []ThemeEntry{{PBL: "OVER.PBL", Image: 0, At: []int{128, 48}, PNG: "over.png", Kind: "redraw"}}}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
	hd, notice, err := LoadTheme(dir, orig, "theme", 3)
	if err != nil || notice != "" || hd == nil {
		t.Fatal("OVER 載入不符", err, notice)
	}
	frame := make([]byte, 64000)
	for y := 0; y < 64; y++ {
		copy(frame[(48+y)*320+128:(48+y)*320+192], px[y*64:(y+1)*64])
	}
	return hd, frame
}

func TestOverFullPresenceClearAndReload(t *testing.T) {
	hd, frame := overTestTheme(t)
	render := func(frame []byte) []byte {
		hd.Layer.Frame(frame, make([]byte, 64000*3))
		out := make([]byte, 960*600*4)
		hd.Draw(out, 3)
		return out
	}
	initial := render(frame)
	if bytes.Equal(initial, make([]byte, len(initial))) || len(hd.Layer.Art) != 8 {
		t.Fatal("完整人物未出現")
	}
	// 逐格期望直接來自原圖；全黑格與人物矩形外均須透明。
	inkCells := 0
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			want := byte(0)
			if x >= 128 && x < 192 && y >= 48 && y < 112 {
				for yy := y / 8 * 8; yy < y/8*8+8; yy++ {
					for xx := x / 8 * 8; xx < x/8*8+8; xx++ {
						if frame[yy*320+xx] != 0 {
							want = 128
						}
					}
				}
			}
			if initial[4*(y*3*960+x*3)+3] != want {
				t.Fatal("原版空格或矩形外透明不符", x, y)
			}
			if want != 0 {
				inkCells++
			}
		}
	}
	if inkCells == 0 {
		t.Fatal("透明期望沒有角色墨跡")
	}
	assertEmpty := func(got []byte, why string) {
		t.Helper()
		if !bytes.Equal(got, make([]byte, len(got))) {
			t.Fatal(why)
		}
	}
	for _, point := range []int{48*320 + 128, 80*320 + 160, 111*320 + 191} {
		bad := append([]byte(nil), frame...)
		bad[point] ^= 1
		assertEmpty(render(bad), "一像素失配仍留下整組人物")
		if !bytes.Equal(render(frame), initial) {
			t.Fatal("失配恢復後人物未恢復")
		}
	}
	assertEmpty(render(make([]byte, 64000)), "原版清空後人物殘留")
	assertEmpty(render(nil), "錯尺寸仍留下人物")
	if !bytes.Equal(render(frame), initial) {
		t.Fatal("完整原版恢復後圖面不同")
	}
	hd.Enabled = false
	assertEmpty(render(frame), "HD 停用仍輸出")
	hd.ResetForLoad()
	if hd.Enabled || len(hd.Layer.Art) != 0 || hd.Layer.Watchers() != 1 {
		t.Fatal("讀檔重登記保留舊圖面或改變開關")
	}
	hd.Enabled = true
	partial := append([]byte(nil), frame...)
	partial[80*320+160] ^= 1
	assertEmpty(render(partial), "讀檔後部分吻合誤恢復人物")
	if !bytes.Equal(render(frame), initial) {
		t.Fatal("讀檔後完整原圖未重建人物")
	}
}

func TestOverRejectWrongSourceAndGeometry(t *testing.T) {
	orig := os.Getenv("PSYCHICWAR_TEST_ORIG")
	if orig == "" {
		t.Skip("真實素材驗證需明示 PSYCHICWAR_TEST_ORIG")
	}
	if _, err := loadOver0(t.TempDir()); err == nil {
		t.Fatal("缺來源未拒絕")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "OVER.PBL"), []byte{2, 0}, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadOver0(dir); err == nil {
		t.Fatal("錯來源雜湊未拒絕")
	}
	themeTestPNG(t, dir, "over.png", 192, 192)
	e := ThemeEntry{PBL: "OVER.PBL", Image: 0, At: []int{128, 48}, PNG: "over.png", Kind: "redraw"}
	base := make([]byte, 64000)
	for _, change := range []func(*ThemeEntry){
		func(e *ThemeEntry) { e.Image = 1 },
		func(e *ThemeEntry) { e.At = []int{120, 48} },
		func(e *ThemeEntry) { e.Src = []int{0, 0, 64, 56} },
		func(e *ThemeEntry) { e.Match = []int{0, 0, 320, 40} },
		func(e *ThemeEntry) { e.Match = []int{128, 48, 56, 64} },
		func(e *ThemeEntry) { e.PNG = "../over.png" },
	} {
		bad := e
		change(&bad)
		if _, err := loadThemeEntry(dir, 0, bad, 3, base); err == nil {
			t.Fatal("未授權來源／矩形被接受", bad)
		}
	}
	e.Src, e.Match = []int{0, 0, 64, 64}, []int{128, 48, 64, 64}
	if _, err := loadThemeEntry(dir, 0, e, 3, base); err != nil {
		t.Fatal("明示完整矩形遭拒絕", err)
	}
	themeTestPNG(t, dir, "wrong.png", 64, 64)
	e.PNG = "wrong.png"
	if _, err := loadThemeEntry(dir, 0, e, 3, base); err == nil {
		t.Fatal("錯候選尺寸未拒絕")
	}
}
