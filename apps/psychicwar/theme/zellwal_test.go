package theme

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/dosgolem/oracle"
)

func TestZellwalRealSourceAndManifest(t *testing.T) {
	orig := os.Getenv("PSYCHICWAR_TEST_ORIG")
	if orig == "" {
		t.Skip("真實素材驗證需明示 PSYCHICWAR_TEST_ORIG")
	}
	for _, n := range []int{3, 4, 5} {
		s, err := loadEnemy(orig, "ENEMY03.PBL", n)
		if err != nil {
			t.Fatal(err)
		}
		want := 1
		if n == 4 {
			want = 2
		}
		if s.slot() != [4]int{32, 152, 24, 32} || len(s.packed) != 384 || len(s.deltas) != want {
			t.Fatal("新來源尺寸或四條限定差分不符", n)
		}
		for _, delta := range s.deltas {
			frame := make([]byte, 64000)
			copyBody := func(px []byte) {
				for y := 0; y < 32; y++ {
					copy(frame[(152+y)*320+32:(152+y)*320+56], px[y*24:(y+1)*24])
				}
			}
			copyBody(delta.from)
			r := oracle.Regs{AX: 1, CX: 0x0826, DX: 0x0304}
			s.blitWithFrame(r, func() []byte { return delta.packed }, func() []byte { return frame })
			if !s.active {
				t.Fatal("已證實差分未啟用", n)
			}
			other, err := loadEnemy0(orig, n)
			if err != nil {
				t.Fatal(err)
			}
			copyBody(other.indexed)
			s.blitWithFrame(r, func() []byte { return delta.packed }, func() []byte { return frame })
			if s.active {
				t.Fatal("同圖號跨檔前姿勢被接受", n)
			}
			copyBody(delta.from)
			frame[160*320+40] ^= 1
			s.blitWithFrame(r, func() []byte { return delta.packed }, func() []byte { return frame })
			if s.active {
				t.Fatal("重疊前姿勢被接受", n)
			}
		}
	}
	for _, n := range []int{-1, 15, 29, 30} {
		if _, err := loadEnemy(orig, "ENEMY03.PBL", n); err == nil {
			t.Fatal("未證實圖號未拒絕", n)
		}
	}
	for _, name := range []string{"ENEMY02.PBL", "../ENEMY03.PBL", "enemy03.pbl"} {
		if _, err := loadEnemy(orig, name, 3); err == nil {
			t.Fatal("未證實檔名未拒絕", name)
		}
	}
	bad := t.TempDir()
	if err := os.WriteFile(filepath.Join(bad, "ENEMY03.PBL"), []byte{0}, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadEnemy(bad, "ENEMY03.PBL", 3); err == nil {
		t.Fatal("錯雜湊未拒絕")
	}
	dir := t.TempDir()
	themeTestPNG(t, dir, "enemy.png", 72, 96)
	e := ThemeEntry{PBL: "ENEMY03.PBL", Image: 3, At: []int{32, 152}, PNG: "enemy.png", Kind: "redraw"}
	load := func(entry ThemeEntry) error {
		m := ThemeManifest{Schema: "psychic-war-theme/1", Name: "zellwal", Scale: 3, Entries: []ThemeEntry{entry}}
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); err != nil {
			t.Fatal(err)
		}
		_, _, err = LoadTheme(dir, orig, "theme", 3)
		return err
	}
	if err := load(e); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*ThemeEntry){
		func(e *ThemeEntry) { e.Image = 15 },
		func(e *ThemeEntry) { e.At = []int{40, 152} },
		func(e *ThemeEntry) { e.Src = []int{0, 0, 16, 16} },
		func(e *ThemeEntry) { e.Match = []int{32, 152, 24, 32} },
	} {
		x := e
		change(&x)
		if load(x) == nil {
			t.Fatal("未授權的清單變更被接受", x)
		}
	}
}
