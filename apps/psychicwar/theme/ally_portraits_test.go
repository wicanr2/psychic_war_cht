package theme

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/dosgolem/oracle"
)

func TestAllyPortraitBatch(t *testing.T) {
	orig, path, output := os.Getenv("PSYCHICWAR_TEST_ORIG"), os.Getenv("PSYCHICWAR_ALLY12_PROOF"), os.Getenv("PSYCHICWAR_ALLY12_OUT")
	if orig == "" || path == "" || output == "" {
		t.Skip("需九張原版肖像來源與獨立輸出")
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
			Image       int
			Prefix, Raw string
			SHA         string `json:"source_sha256"`
		}
	}
	if e := json.Unmarshal(read(path), &proof); e != nil {
		t.Fatal(e)
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(path)))
	newDir := filepath.Join(root, "hd/theme-enemy360-ally12-effects-maze-B-v1-20261006")
	oldDir := filepath.Join(root, "hd/theme-enemy360-effects-maze-B-v1-20261006")
	hd, _, e := LoadTheme(newDir, orig, "", 3)
	if e != nil {
		t.Fatal(e)
	}
	old, _, e := LoadTheme(oldDir, orig, "", 3)
	if e != nil {
		t.Fatal(e)
	}
	render := func(h *Theme, frame []byte) []byte {
		h.frameSprites(frame)
		rgb := make([]byte, 3*len(frame))
		for i, value := range frame {
			c := mazeColors[value]
			copy(rgb[i*3:i*3+3], c[:])
		}
		h.Layer.Frame(frame, rgb)
		p := make([]byte, 960*600*4)
		h.Draw(p, 3)
		return p
	}
	rows := []map[string]any{}
	for _, r := range proof.Rows {
		if r.Image < 3 {
			continue
		}
		raw, e := hex.DecodeString(r.Raw)
		if e != nil {
			t.Fatal(e)
		}
		s, e := loadAlly(orig, r.Image)
		if e != nil || !bytes.Equal(raw, s.packed) || fmt.Sprintf("%x", sha256.Sum256(raw)) != r.SHA {
			t.Fatal("原版來源不同", r.Image, e)
		}
		before, after := read(r.Prefix+"-before.frame"), read(r.Prefix+"-after.frame")
		hd.ResetForLoad()
		old.ResetForLoad()
		render(hd, before)
		regs := oracle.Regs{AX: 0, CX: 0x2002, DX: 0x0304}
		hd.blitSprites(regs, func() []byte { return raw }, func() []byte { return before })
		hd.finishBlit()
		actual, prior := render(hd, after), render(old, after)
		if bytes.Equal(actual, prior) {
			t.Fatal("新肖像沒有繪製", r.Image)
		}
		hd.ResetForLoad()
		if !bytes.Equal(render(hd, after), actual) {
			t.Fatal("冷載肖像不同")
		}
		hd.Enabled = false
		off := make([]byte, len(actual))
		if hd.Draw(off, 3) || !bytes.Equal(off, make([]byte, len(off))) {
			t.Fatal("關閉HD仍繪製")
		}
		hd.Enabled = true
		wrong := append([]byte(nil), raw...)
		wrong[0] ^= 1
		s.blitWithFrame(regs, func() []byte { return wrong }, func() []byte { return before })
		if s.active {
			t.Fatal("接受錯誤肖像來源")
		}
		for _, p := range []struct{ At, Match []int }{{[]int{236, 152}, []int{248, 0, 72, 40}}, {[]int{128, 8}, nil}, {[]int{132, 8}, []int{248, 0, 72, 40}}} {
			if _, _, e := allyPosition(ThemeEntry{Image: r.Image, At: p.At, Match: p.Match}); e == nil {
				t.Fatal("接受未證實盟友位置", r.Image)
			}
		}
		paths := []string{}
		for n, p := range [][]byte{actual, prior} {
			name := filepath.Join(output, fmt.Sprintf("ally%02d-%d.png", r.Image, n))
			f, e := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
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
			paths = append(paths, name)
		}
		rows = append(rows, map[string]any{"image": r.Image, "actual": paths[0], "old": paths[1], "frame": r.Prefix + "-after.frame"})
	}
	if len(rows) != 9 {
		t.Fatal("肖像缺項")
	}
	for _, n := range []int{-1, 12, 15, 31} {
		if _, e := loadAlly(orig, n); e == nil {
			t.Fatal("接受非完整人物來源")
		}
	}
	v, e := json.MarshalIndent(map[string]any{"status": "PASS_ALLY9_LIMITED_PORTRAIT_RUNTIME", "renders": rows, "limits": "Actual controlled original source frames; not normal recruitment/party/GUI/DAT."}, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(output, "render.json"), append(v, '\n'), 0644); e != nil {
		t.Fatal(e)
	}
}
