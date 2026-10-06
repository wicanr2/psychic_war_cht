package theme

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func anchorTestTheme(t *testing.T, match []int, background bool) (*Theme, []byte, string, string) {
	t.Helper()
	orig := os.Getenv("PSYCHICWAR_TEST_ORIG")
	if orig == "" {
		t.Skip("需明示真實原版資料")
	}
	frame, err := themeBackground(orig)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	entries := []ThemeEntry{}
	if background {
		themeTestPNG(t, dir, "screen.png", 960, 120)
		themeTestPNG(t, dir, "menu.png", 264, 216)
		entries = append(entries, ThemeEntry{PBL: "SCREEN.PBL", Image: 0, At: []int{0, 0}, PNG: "screen.png", Kind: "redraw", Match: match}, ThemeEntry{PBL: "MENU.PBL", Image: 0, At: []int{160, 4}, PNG: "menu.png", Kind: "redraw", Match: match})
	}
	ally, err := loadAlly0(orig)
	if err != nil {
		t.Fatal(err)
	}
	enemy, err := loadEnemy(orig, "ENEMY00.PBL", 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []*spritePresence{ally, enemy} {
		for y := 0; y < s.h; y++ {
			copy(frame[(s.y+y)*320+s.x:(s.y+y)*320+s.x+s.w], s.indexed[y*s.w:(y+1)*s.w])
		}
	}
	themeTestPNG(t, dir, "ally.png", 72, 96)
	themeTestPNG(t, dir, "enemy.png", 72, 96)
	entries = append(entries, ThemeEntry{PBL: "ALLY.PBL", Image: 0, At: []int{264, 152}, PNG: "ally.png", Kind: "redraw", Match: match}, ThemeEntry{PBL: "ENEMY00.PBL", Image: 3, At: []int{32, 152}, PNG: "enemy.png", Kind: "redraw", Match: match})
	m := ThemeManifest{Schema: "psychic-war-theme/1", Name: "anchor", Scale: 3, Entries: entries}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
	hd, notice, err := LoadTheme(dir, orig, "theme", 3)
	if err != nil || notice != "" || hd == nil {
		t.Fatal(err, notice)
	}
	return hd, frame, dir, orig
}

func anchorRender(hd *Theme, frame []byte) []byte {
	hd.frameSprites(frame)
	hd.Layer.Frame(frame, make([]byte, 64000*3))
	plane := make([]byte, 960*600*4)
	hd.Draw(plane, 3)
	return plane
}

func TestBackgroundAnchorMenuChangeAndRecovery(t *testing.T) {
	hd, frame, _, _ := anchorTestTheme(t, []int{0, 0, 160, 40}, true)
	before := anchorRender(hd, frame)
	if bytes.Equal(before, make([]byte, len(before))) {
		t.Fatal("正對照沒有圖面")
	}
	changed := append([]byte(nil), frame...)
	changed[5*320+168] ^= 1
	after := anchorRender(hd, changed)
	removed := 0
	for y := 0; y < 600; y++ {
		for x := 0; x < 960; x++ {
			i := 4 * (y*960 + x)
			if x >= 168*3 && x < 176*3 && y < 8*3 {
				if before[i+3] != 0 {
					removed++
				}
				if !bytes.Equal(after[i:i+4], []byte{0, 0, 0, 0}) {
					t.Fatal("選單不符格未回原版")
				}
			} else if !bytes.Equal(before[i:i+4], after[i:i+4]) {
				t.Fatal("選單改畫清除了其他背景或角色", x, y)
			}
		}
	}
	if removed == 0 {
		t.Fatal("選單單像素負對照無效")
	}
	if !bytes.Equal(anchorRender(hd, frame), before) {
		t.Fatal("選單恢復圖面不同")
	}
	bad := append([]byte(nil), frame...)
	bad[0] ^= 1
	if !bytes.Equal(anchorRender(hd, bad), make([]byte, len(before))) {
		t.Fatal("左側錨點失配仍顯示HD")
	}
	if !bytes.Equal(anchorRender(hd, frame), before) {
		t.Fatal("左側錨點恢復圖面不同")
	}
	hd.Enabled = false
	hd.ResetForLoad()
	if hd.Enabled || len(hd.Layer.Art) != 0 {
		t.Fatal("重建未保持停用或留殘片")
	}
	hd.Enabled = true
	if !bytes.Equal(anchorRender(hd, frame), before) {
		t.Fatal("重新登記圖面不同")
	}
}

func TestBackgroundAnchorLegacyDefault(t *testing.T) {
	for _, match := range [][]int{nil, {0, 0, 320, 40}} {
		hd, frame, _, _ := anchorTestTheme(t, match, false)
		before := anchorRender(hd, frame)
		if bytes.Equal(before, make([]byte, len(before))) {
			t.Fatal("舊契約正對照無圖面")
		}
		frame[5*320+168] ^= 1
		if !bytes.Equal(anchorRender(hd, frame), make([]byte, len(before))) {
			t.Fatal("省略或明示全寬預設被改動")
		}
	}
}

func TestBackgroundAnchorRejectUnprovenSpriteMatches(t *testing.T) {
	_, _, dir, orig := anchorTestTheme(t, []int{0, 0, 160, 40}, false)
	for _, name := range []string{"ALLY.PBL", "ENEMY00.PBL"} {
		for _, match := range [][]int{{0, 0, 159, 40}, {1, 0, 160, 40}, {0, 1, 160, 40}, {0, 0, 160, 39}} {
			image, at, png := 0, []int{264, 152}, "ally.png"
			if name == "ENEMY00.PBL" {
				image, at, png = 3, []int{32, 152}, "enemy.png"
			}
			m := ThemeManifest{Schema: "psychic-war-theme/1", Name: "bad", Scale: 3, Entries: []ThemeEntry{{PBL: name, Image: image, At: at, PNG: png, Kind: "redraw", Match: match}}}
			b, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); err != nil {
				t.Fatal(err)
			}
			if _, _, err = LoadTheme(dir, orig, "theme", 3); err == nil {
				t.Fatal("未拒絕未證實sprite錨點", name, match)
			}
		}
	}
}
