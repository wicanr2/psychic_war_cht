package theme

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func transportRoomManifest(t *testing.T, dir string, entries []ThemeEntry) {
	t.Helper()
	b, err := json.Marshal(ThemeManifest{Schema: "psychic-war-theme/1", Name: "transport", Scale: 3, Entries: entries})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestTransportRoomExclusiveSourcesAndRecovery(t *testing.T) {
	orig := os.Getenv("PSYCHICWAR_TEST_ORIG")
	if orig == "" {
		t.Skip("需明示真實PBL素材")
	}
	dir := t.TempDir()
	ids := []int{0, 3, 2, 8, 22}
	entries := []ThemeEntry{}
	colors := []color.NRGBA{{R: 31, A: 128}, {G: 61, A: 128}, {B: 91, A: 128}, {R: 121, B: 91, A: 128}, {G: 121, B: 91, A: 128}}
	for index, id := range ids {
		name := fmt.Sprintf("room-%d.png", id)
		img := image.NewNRGBA(image.Rect(0, 0, 216, 216))
		for y := 0; y < 216; y++ {
			for x := 0; x < 216; x++ {
				img.SetNRGBA(x, y, colors[index])
			}
		}
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
		entries = append(entries, ThemeEntry{PBL: "ROOM0.PBL", Image: id, At: []int{4, 124}, PNG: name, Kind: "redraw"})
	}
	transportRoomManifest(t, dir, entries)
	hd, notice, err := LoadTheme(dir, orig, "theme", 3)
	if err != nil || notice != "" || hd == nil {
		t.Fatal(err, notice)
	}
	background, err := themeBackground(orig)
	if err != nil {
		t.Fatal(err)
	}
	render := func(frame []byte) []byte {
		hd.Layer.Frame(frame, make([]byte, 64000*3))
		out := make([]byte, 960*600*4)
		hd.Draw(out, 3)
		return out
	}
	empty := make([]byte, 960*600*4)
	for _, index := range []int{0, 1, 2, 4, 2, 3, 0} {
		pixels, err := loadRoomImage(orig, ids[index])
		if err != nil {
			t.Fatal(err)
		}
		frame := append([]byte(nil), background...)
		for y := 0; y < 72; y++ {
			copy(frame[(124+y)*320+4:(124+y)*320+76], pixels[y*72:(y+1)*72])
		}
		plane := render(frame)
		if bytes.Equal(plane, empty) {
			t.Fatal("完整房間沒有HD", ids[index])
		}
		// 快取可以保留舊列；以完整輸出像素判斷舊房間是否仍被畫出。
		var cells [10][10]bool
		for i, pixel := range pixels {
			if pixel != 0 {
				cells[(124+i/72)/8-15][(4+i%72)/8] = true
			}
		}
		want := make([]byte, len(empty))
		c := colors[index]
		for y := 124; y < 196; y++ {
			for x := 4; x < 76; x++ {
				if !cells[y/8-15][x/8] {
					continue
				}
				for yy := y * 3; yy < (y+1)*3; yy++ {
					for xx := x * 3; xx < (x+1)*3; xx++ {
						copy(want[4*(yy*960+xx):], []byte{c.R, c.G, c.B, c.A})
					}
				}
			}
		}
		if !bytes.Equal(plane, want) {
			t.Fatal("完整像素不符或切換後仍顯示其他房間", ids[index])
		}
		ink := -1
		for i, pixel := range pixels {
			if pixel != 0 {
				ink = i
				break
			}
		}
		if ink < 0 {
			t.Fatal("房間原圖沒有墨跡")
		}
		p := 4 * (((124+ink/72)*3)*960 + (4+ink%72)*3)
		if !bytes.Equal(plane[p:p+4], []byte{c.R, c.G, c.B, c.A}) {
			t.Fatal("原圖顯示了其他候選")
		}
		changed := append([]byte(nil), frame...)
		changed[(124+ink/72)*320+4+ink%72] ^= 1
		if !bytes.Equal(render(changed), empty) {
			t.Fatal("完整原圖失配仍留房間")
		}
		if !bytes.Equal(render(frame), plane) {
			t.Fatal("完整原圖恢復不同")
		}
		hd.Enabled = false
		hd.ResetForLoad()
		if hd.Enabled || !bytes.Equal(render(frame), empty) {
			t.Fatal("停用／重建後仍畫HD")
		}
		hd.Enabled = true
		if !bytes.Equal(render(frame), plane) {
			t.Fatal("載回重建圖面不同")
		}
	}
	if !bytes.Equal(render(make([]byte, 64000)), empty) {
		t.Fatal("原版清空後仍留房間")
	}
}

func TestTransportRoomRejectUnprovenAndDuplicates(t *testing.T) {
	_, _, dir, orig := roomTestTheme(t)
	for _, id := range []int{-1, 1, 7, 23, 31} {
		entry := ThemeEntry{PBL: "ROOM0.PBL", Image: id, At: []int{4, 124}, PNG: "room.png", Kind: "redraw"}
		transportRoomManifest(t, dir, []ThemeEntry{entry})
		if _, _, err := LoadTheme(dir, orig, "theme", 3); err == nil {
			t.Fatal("接受未證實圖號", id)
		}
	}
	entries := []ThemeEntry{}
	for _, id := range []int{0, 3, 2, 8, 3} {
		entries = append(entries, ThemeEntry{PBL: "ROOM0.PBL", Image: id, At: []int{4, 124}, PNG: "room.png", Kind: "redraw"})
	}
	transportRoomManifest(t, dir, entries)
	if _, _, err := LoadTheme(dir, orig, "theme", 3); err == nil {
		t.Fatal("接受重複房間圖號")
	}
}
