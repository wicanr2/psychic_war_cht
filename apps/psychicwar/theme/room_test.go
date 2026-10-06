package theme

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func roomTestTheme(t *testing.T) (*Theme, []byte, string, string) {
	t.Helper()
	orig := os.Getenv("PSYCHICWAR_TEST_ORIG")
	if orig == "" {
		t.Skip("真實素材驗證需明示 PSYCHICWAR_TEST_ORIG")
	}
	px, err := loadRoom8(orig)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := themeBackground(orig)
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 72; y++ {
		copy(frame[(124+y)*320+4:(124+y)*320+76], px[y*72:(y+1)*72])
	}
	dir := t.TempDir()
	themeTestPNG(t, dir, "room.png", 216, 216)
	m := ThemeManifest{Schema: "psychic-war-theme/1", Name: "room", Scale: 3, Entries: []ThemeEntry{{PBL: "ROOM0.PBL", Image: 8, At: []int{4, 124}, PNG: "room.png", Kind: "redraw"}}}
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

func TestRoomOriginalGridEdgesAndRecovery(t *testing.T) {
	hd, frame, _, _ := roomTestTheme(t)
	render := func(f []byte) []byte {
		hd.Layer.Frame(f, make([]byte, 64000*3))
		out := make([]byte, 960*600*4)
		hd.Draw(out, 3)
		return out
	}
	initial := render(frame)
	if len(hd.Layer.Art) != 10 || bytes.Equal(initial, make([]byte, len(initial))) {
		t.Fatal("房間十列未完整繪製")
	}
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			want := byte(0)
			if x >= 4 && x < 76 && y >= 124 && y < 196 {
				for yy := y / 8 * 8; yy < y/8*8+8; yy++ {
					for xx := x / 8 * 8; xx < x/8*8+8; xx++ {
						if xx >= 4 && xx < 76 && yy >= 124 && yy < 196 && frame[yy*320+xx] != 0 {
							want = 128
						}
					}
				}
			}
			if initial[4*(y*3*960+x*3)+3] != want {
				t.Fatal("原點格／矩形透明邊界不符", x, y, want)
			}
		}
	}
	// 圖外像素不改完整匹配，只應遮掉原點格的一角。
	halo := append([]byte(nil), frame...)
	halo[120*320] ^= 1
	after := render(halo)
	removed := 0
	for y := 0; y < 600; y++ {
		for x := 0; x < 960; x++ {
			i := 4 * (y*960 + x)
			if x < 24 && y >= 360 && y < 384 {
				if initial[i+3] != 0 {
					removed++
				}
				if !bytes.Equal(after[i:i+4], []byte{0, 0, 0, 0}) {
					t.Fatal("圖外失配未遮同一原點格")
				}
			} else if !bytes.Equal(after[i:i+4], initial[i:i+4]) {
				t.Fatal("圖外失配改動其他格")
			}
		}
	}
	if removed == 0 {
		t.Fatal("圖外負對照沒有可見像素")
	}
	if !bytes.Equal(render(frame), initial) {
		t.Fatal("外緣恢復後圖面不同")
	}
	inside := append([]byte(nil), frame...)
	inside[124*320+4] ^= 1
	for _, f := range [][]byte{inside, make([]byte, 64000), nil} {
		if !bytes.Equal(render(f), make([]byte, len(initial))) {
			t.Fatal("完整原圖失配／清空／錯尺寸仍留房間")
		}
	}
	if !bytes.Equal(render(frame), initial) {
		t.Fatal("完整房間恢復失敗")
	}
	hd.Enabled = false
	if !bytes.Equal(render(frame), make([]byte, len(initial))) {
		t.Fatal("停用HD仍畫房間")
	}
	hd.ResetForLoad()
	if hd.Enabled || len(hd.Layer.Art) != 0 || hd.Layer.Watchers() != 1 {
		t.Fatal("載回重建未保留開關或仍留舊圖")
	}
	hd.Enabled = true
	if !bytes.Equal(render(frame), initial) {
		t.Fatal("載回原圖後無法恢復")
	}
}

func TestRoomRejectUnprovenEntries(t *testing.T) {
	_, _, dir, orig := roomTestTheme(t)
	cases := map[string]func(*ThemeEntry){
		"image":             func(e *ThemeEntry) { e.Image = 7 },
		"position":          func(e *ThemeEntry) { e.At = []int{0, 120} },
		"crop":              func(e *ThemeEntry) { e.Src = []int{0, 0, 64, 72} },
		"background-anchor": func(e *ThemeEntry) { e.Match = []int{0, 0, 320, 40} },
		"other-archive":     func(e *ThemeEntry) { e.PBL = "ROOM1.PBL" },
	}
	for name, alter := range cases {
		t.Run(name, func(t *testing.T) {
			entry := ThemeEntry{PBL: "ROOM0.PBL", Image: 8, At: []int{4, 124}, PNG: "room.png", Kind: "redraw"}
			alter(&entry)
			m := ThemeManifest{Schema: "psychic-war-theme/1", Name: "room", Scale: 3, Entries: []ThemeEntry{entry}}
			b, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); err != nil {
				t.Fatal(err)
			}
			if _, _, err = LoadTheme(dir, orig, "theme", 3); err == nil {
				t.Fatal("接受未證實房間條目")
			}
		})
	}
	entry := ThemeEntry{PBL: "ROOM0.PBL", Image: 8, At: []int{4, 124}, PNG: "room.png", Kind: "redraw"}
	m := ThemeManifest{Schema: "psychic-war-theme/1", Name: "room", Scale: 3, Entries: []ThemeEntry{entry, entry}}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err = LoadTheme(dir, orig, "theme", 3); err == nil {
		t.Fatal("接受重複ROOM列")
	}
}

func TestRoomRejectWrongSourceVersion(t *testing.T) {
	_, _, dir, orig := roomTestTheme(t)
	bad := t.TempDir()
	for _, name := range []string{"SCREEN.PBL", "MENU.PBL", "ROOM0.PBL"} {
		b, err := os.ReadFile(filepath.Join(orig, name))
		if err != nil {
			t.Fatal(err)
		}
		if name == "ROOM0.PBL" {
			b[len(b)-1] ^= 1
		}
		if err = os.WriteFile(filepath.Join(bad, name), b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := LoadTheme(dir, bad, "theme", 3); err == nil {
		t.Fatal("接受錯版本原版PBL")
	}
}
