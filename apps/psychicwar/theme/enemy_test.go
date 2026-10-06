package theme

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/dosgolem/oracle"
)

func TestEnemyPoseExclusiveAndXOR(t *testing.T) {
	dir := t.TempDir()
	themeTestPNG(t, dir, "pose3.png", 72, 96)
	green := image.NewNRGBA(image.Rect(0, 0, 72, 96))
	for y := 0; y < 96; y++ {
		for x := 0; x < 72; x++ {
			green.SetNRGBA(x, y, color.NRGBA{R: 10, G: 220, B: 60, A: 128})
		}
	}
	f, e := os.Create(filepath.Join(dir, "pose4.png"))
	if e != nil {
		t.Fatal(e)
	}
	if e = png.Encode(f, green); e != nil {
		t.Fatal(e)
	}
	if e = f.Close(); e != nil {
		t.Fatal(e)
	}
	base3, base4 := make([]byte, 64000), make([]byte, 64000)
	for y := 152; y < 184; y++ {
		for x := 32; x < 56; x++ {
			base3[y*320+x] = 2
			base4[y*320+x] = 2
		}
	}
	base4[152*320+32] = 3
	var groups []themeGroup
	for n, base := range [][]byte{base3, base4} {
		px := make([]byte, 768)
		for y := 0; y < 32; y++ {
			copy(px[y*24:(y+1)*24], base[(152+y)*320+32:(152+y)*320+56])
		}
		name := "pose3.png"
		if n == 1 {
			name = "pose4.png"
		}
		g, e := loadThemeEntry(dir, n, ThemeEntry{PBL: "ENEMY00.PBL", Image: 3 + n, At: []int{32, 152}, PNG: name, Kind: "redraw"}, 3, base)
		if e != nil {
			t.Fatal(e)
		}
		g.sprite = newSprite(px, 32, 152, 24, 32)
		groups = append(groups, g)
	}
	s3, s4 := groups[0].sprite, groups[1].sprite
	s4.deltas = []spriteDelta{{from: s3.indexed, packed: make([]byte, 384)}}
	for i := range s4.deltas[0].packed {
		s4.deltas[0].packed[i] = s3.packed[i] ^ s4.packed[i]
	}
	hd := &Theme{Enabled: true, groups: groups}
	hd.Layer.W, hd.Layer.H = 320, 200
	hd.ResetForLoad()
	render := func(idx []byte) []byte {
		hd.frameSprites(idx)
		hd.Layer.Frame(idx, make([]byte, 64000*3))
		out := make([]byte, 960*600*4)
		hd.Draw(out, 3)
		return out
	}
	shared := 4 * (160*3*960 + 40*3)
	a := render(base3)
	if !s3.active || s4.active || !bytes.Equal(a[shared:shared+4], []byte{200, 120, 40, 128}) {
		t.Fatal("第一動作有其他動作混入共用格")
	}
	b := render(base4)
	if s3.active || !s4.active || !bytes.Equal(b[shared:shared+4], []byte{10, 220, 60, 128}) {
		t.Fatal("切換後舊動作仍疊到共用格")
	}
	if !bytes.Equal(render(base3), a) {
		t.Fatal("切回前一動作未完整恢復")
	}
	r := oracle.Regs{AX: 1, CX: 0x0826, DX: 0x0304}
	s3.blitWithFrame(r, func() []byte { t.Fatal("完整圖來源不該解釋為 XOR"); return nil }, func() []byte { return nil })
	s4.blitWithFrame(r, func() []byte { return s4.deltas[0].packed }, func() []byte { return base3 })
	started := render(base3)
	if s3.active || !s4.active || !s4.inFlight || !bytes.Equal(started[shared:shared+4], []byte{10, 220, 60, 128}) {
		t.Fatal("貼圖首指令的完整舊畫面推翻已知新來源")
	}
	partial := append([]byte(nil), base4...)
	partial[160*320+32] = 7
	c := render(partial)
	if s3.active || !s4.active || !bytes.Equal(c[shared:shared+4], []byte{10, 220, 60, 128}) {
		t.Fatal("已證實差分未保持未被遮格")
	}
	if !bytes.Equal(render(base4), b) {
		t.Fatal("差分動作恢復後留遮格")
	}
	hd.finishBlit()
	if s3.inFlight || s4.inFlight || !bytes.Equal(render(base4), b) {
		t.Fatal("返回閘門或完成動作不符")
	}
	s4.blitWithFrame(r, func() []byte { return []byte{9} }, func() []byte { return base3 })
	if s4.active {
		t.Fatal("錯誤 XOR 來源啟用目標")
	}
	s4.blitWithFrame(r, func() []byte { return s4.deltas[0].packed }, func() []byte { return base4 })
	if s4.active {
		t.Fatal("錯誤前一動作啟用目標")
	}
	r.AX = 2
	s4.active = true
	s4.blitWithFrame(r, func() []byte { t.Fatal("未知 AL 讀來源"); return nil }, func() []byte { t.Fatal("未知 AL 讀原圖"); return nil })
	if s4.active {
		t.Fatal("未知模式保留動作來源")
	}
	r.AX = 0
	s3.blitWithFrame(r, func() []byte { return s3.packed }, func() []byte { t.Fatal("完整圖不需讀前一動作"); return nil })
	if !s3.active {
		t.Fatal("完整來源未啟用")
	}
	hd.Enabled = false
	hd.ResetForLoad()
	if s3.active || s4.active || hd.Enabled {
		t.Fatal("讀檔保留動作或修改開關")
	}
	hd.Enabled = true
	if !bytes.Equal(render(base4), b) {
		t.Fatal("載回中途動作未重建")
	}
}

func TestEnemyRealSourceAndManifest(t *testing.T) {
	orig := os.Getenv("PSYCHICWAR_TEST_ORIG")
	if orig == "" {
		t.Skip("真實素材驗證需明示 PSYCHICWAR_TEST_ORIG")
	}
	for n := 0; n <= 14; n++ {
		s, e := loadEnemy0(orig, n)
		if e != nil {
			t.Fatal(e)
		}
		if s.slot() != [4]int{32, 152, 24, 32} || len(s.packed) != 384 {
			t.Fatal("角色原座標或來源長度不符")
		}
		if len(s.deltas) != []int{1, 2, 1}[n%3] || len(s.deltas[0].packed) != 384 {
			t.Fatal("已證實差分資料缺失")
		}
		for _, delta := range s.deltas {
			frame := make([]byte, 64000)
			for y := 0; y < 32; y++ {
				copy(frame[(152+y)*320+32:(152+y)*320+56], delta.from[y*24:(y+1)*24])
			}
			r := oracle.Regs{AX: 1, CX: 0x0826, DX: 0x0304}
			s.blitWithFrame(r, func() []byte { return delta.packed }, func() []byte { return frame })
			if !s.active {
				t.Fatal("已證實往返差分未啟用", n)
			}
			other, err := loadEnemy0(orig, (n+3)%15)
			if err != nil {
				t.Fatal(err)
			}
			for y := 0; y < 32; y++ {
				copy(frame[(152+y)*320+32:(152+y)*320+56], other.indexed[y*24:(y+1)*24])
			}
			s.blitWithFrame(r, func() []byte { return delta.packed }, func() []byte { return frame })
			if s.active {
				t.Fatal("跨角色前姿勢被接受", n)
			}
		}
	}
	for _, n := range []int{-1, 15, 29, 30} {
		if _, e := loadEnemy0(orig, n); e == nil {
			t.Fatal("未證實圖號未拒絕", n)
		}
	}
	if _, e := loadEnemy0(t.TempDir(), 3); e == nil {
		t.Fatal("缺敵人來源未拒絕")
	}
	dir := t.TempDir()
	themeTestPNG(t, dir, "enemy.png", 72, 96)
	e := ThemeEntry{PBL: "ENEMY00.PBL", Image: 3, At: []int{32, 152}, PNG: "enemy.png", Kind: "redraw"}
	m := ThemeManifest{Schema: "psychic-war-theme/1", Name: "enemy", Scale: 3, Entries: []ThemeEntry{e}}
	write := func() {
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	write()
	if _, _, err := LoadTheme(dir, orig, "theme", 3); err != nil {
		t.Fatal(err)
	}
	m.Entries = append(m.Entries, e)
	write()
	if _, _, err := LoadTheme(dir, orig, "theme", 3); err == nil {
		t.Fatal("重複角色來源未拒絕")
	}
	for _, change := range []func(*ThemeEntry){func(e *ThemeEntry) { e.Image = 15 }, func(e *ThemeEntry) { e.At = []int{36, 152} }, func(e *ThemeEntry) { e.Src = []int{0, 0, 16, 16} }, func(e *ThemeEntry) { e.Match = []int{32, 152, 24, 32} }} {
		bad := e
		change(&bad)
		if _, err := loadThemeEntry(dir, 0, bad, 3, make([]byte, 64000)); err == nil {
			t.Fatal("未證實敵人來源條件未拒絕")
		}
	}
}
