package theme

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestAllySmallBatch(t *testing.T) {
	orig, proofPath, newDir, output := os.Getenv("PSYCHICWAR_TEST_ORIG"), os.Getenv("PSYCHICWAR_ALLY27_PROOF"), os.Getenv("PSYCHICWAR_ALLY27_THEME"), os.Getenv("PSYCHICWAR_ALLY27_OUT")
	if orig == "" || proofPath == "" || newDir == "" || output == "" {
		t.Skip("需十五小圖原版31原位與獨立輸出")
	}
	read := func(p string) []byte {
		v, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	var proof struct {
		Rows []struct {
			Image, Slot int
			Rect        [4]int
			Prefix      string
		}
	}
	if e := json.Unmarshal(read(proofPath), &proof); e != nil {
		t.Fatal(e)
	}
	hd, _, e := LoadTheme(newDir, orig, "", 3)
	if e != nil {
		t.Fatal(e)
	}
	old, _, e := LoadTheme(filepath.Join(filepath.Dir(newDir), "theme-enemy360-ally12-effects-maze-B-v1-20261006"), orig, "", 3)
	if e != nil {
		t.Fatal(e)
	}
	render := func(h *Theme, frame []byte) []byte {
		h.frameSprites(frame)
		rgb := make([]byte, 3*len(frame))
		for i, v := range frame {
			c := mazeColors[v]
			copy(rgb[i*3:i*3+3], c[:])
		}
		h.Layer.Frame(frame, rgb)
		p := make([]byte, 960*600*4)
		h.Draw(p, 3)
		return p
	}
	rows := []map[string]any{}
	for _, r := range proof.Rows {
		s, e := loadAlly(orig, r.Image)
		if e != nil || s.w != 16 || s.h != 16 {
			t.Fatal("小圖來源尺寸", r.Image, e)
		}
		pos := ThemeEntry{Image: r.Image, At: r.Rect[:2], Match: []int{248, 0, 72, 40}}
		x, y, e := allyPosition(pos)
		if e != nil || x != r.Rect[0] || y != r.Rect[1] {
			t.Fatal("原版位置", r.Image, r.Slot, e)
		}
		after := read(r.Prefix + "-after.frame")
		hd.ResetForLoad()
		old.ResetForLoad()
		actual, prior := render(hd, after), render(old, after)
		if bytes.Equal(actual, prior) {
			t.Fatal("新增小圖未畫", r.Image, r.Slot)
		}
		hd.ResetForLoad()
		if !bytes.Equal(render(hd, after), actual) {
			t.Fatal("冷載不一致")
		}
		hd.Enabled = false
		off := make([]byte, len(actual))
		if hd.Draw(off, 3) || !bytes.Equal(off, make([]byte, len(off))) {
			t.Fatal("關閉仍畫")
		}
		hd.Enabled = true
		for _, bad := range []ThemeEntry{{Image: r.Image, At: []int{x + 4, y}, Match: pos.Match}, {Image: r.Image, At: pos.At}, {Image: r.Image, At: pos.At, Match: pos.Match, Src: []int{0, 0, 24, 32}}} {
			if _, _, e := allyPosition(bad); e == nil {
				t.Fatal("接受未證實位置／尺寸／錨點")
			}
		}
		wrong := append([]byte(nil), after...)
		wrong[0*320+248] ^= 1
		render(hd, wrong)
		for _, g := range hd.groups {
			if g.sprite != nil && g.sprite.w == 16 && g.sprite.active {
				t.Fatal("錨點失配仍保持小圖")
			}
		}
		paths := []string{}
		for i, p := range [][]byte{actual, prior} {
			path := filepath.Join(output, fmt.Sprintf("ally%02d-slot%d-%d.png", r.Image, r.Slot, i))
			f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
			if e != nil {
				t.Fatal(e)
			}
			e = png.Encode(f, &image.NRGBA{Pix: p, Stride: 960 * 4, Rect: image.Rect(0, 0, 960, 600)})
			if e != nil {
				t.Fatal(e)
			}
			if e = f.Close(); e != nil {
				t.Fatal(e)
			}
			paths = append(paths, path)
		}
		rows = append(rows, map[string]any{"image": r.Image, "slot": r.Slot, "frame": r.Prefix + "-after.frame", "actual": paths[0], "old": paths[1]})
	}
	if len(rows) != 31 {
		t.Fatal("缺原位")
	}
	for _, n := range []int{-1, 12, 13, 14, 15, 31} {
		if _, e := loadAlly(orig, n); e == nil {
			t.Fatal("接受未READY來源", n)
		}
	}
	v, e := json.MarshalIndent(map[string]any{"status": "PASS_15_ALLY_SMALL_31_POSITIONS", "renders": rows, "limits": "Controlledoriginalonly.NormalGUI/DAT/remainingmaskpending."}, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(output, "render.json"), append(v, '\n'), 0644); e != nil {
		t.Fatal(e)
	}
}
