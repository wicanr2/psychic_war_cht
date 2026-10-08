package theme

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestFixedSceneReadyPositions(t *testing.T) {
	cases := []struct {
		pbl   string
		n     int
		at    []int
		valid bool
	}{
		{"MAP.PBL", 17, []int{4, 4}, true}, {"MAP.PBL", 18, []int{4, 4}, false},
		{"OPEN.PBL", 3, []int{24, 120}, true}, {"OPEN.PBL", 4, []int{136, 152}, false},
		{"OPEN.PBL", 6, []int{56, 152}, true}, {"END0.PBL", 1, []int{56, 80}, true},
		{"END1.PBL", 0, []int{8, 152}, true}, {"END1.PBL", 0, []int{4, 152}, false},
		{"END1.PBL", 1, []int{4, 152}, true}, {"END1.PBL", 1, []int{8, 152}, false},
		{"END1.PBL", 17, []int{296, 152}, true}, {"END1.PBL", 17, []int{300, 152}, false},
		{"END1.PBL", 18, []int{4, 155}, false},
	}
	for _, c := range cases {
		e := ThemeEntry{PBL: c.pbl, Image: c.n, At: c.at, Kind: "redraw"}
		if _, _, _, _, err := fixedScenePosition(e); (err == nil) != c.valid {
			t.Fatal(c, err)
		}
	}
}

func TestOpeningParentArtAndRecovery(t *testing.T) {
	root, orig, output := os.Getenv("PSYCHICWAR_OPEN_PARENT_THEME"), os.Getenv("PSYCHICWAR_TEST_ORIG"), os.Getenv("PSYCHICWAR_OPEN_PARENT_OUTPUT")
	if root == "" || orig == "" || output == "" {
		t.Skip("需明示開場親子實際候選")
	}
	read := func(p string) []byte {
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	var m ThemeManifest
	if e := json.Unmarshal(read(filepath.Join(root, "manifest.json")), &m); e != nil {
		t.Fatal(e)
	}
	dir := t.TempDir()
	var selected []ThemeEntry
	for _, e := range m.Entries {
		if e.PBL == "OPEN.PBL" && (e.Image == 3 || e.Image == 5 || e.Image == 6) {
			selected = append(selected, e)
			if er := os.WriteFile(filepath.Join(dir, e.PNG), read(filepath.Join(root, e.PNG)), 0644); er != nil {
				t.Fatal(er)
			}
		}
	}
	if len(selected) != 3 {
		t.Fatal("親子來源数")
	}
	m.Entries = selected
	m.Maze = nil
	m.BuiltinMasks = nil
	raw, e := json.Marshal(m)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0644); e != nil {
		t.Fatal(e)
	}
	hd, notice, e := LoadTheme(dir, orig, "", 3)
	if e != nil || notice != "" {
		t.Fatal(notice, e)
	}
	if len(hd.groups) != 4 {
		t.Fatal("兩原始模板與兩小圖未建立")
	}
	if e = os.Mkdir(output, 0755); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 2; i++ {
		prefix := filepath.Join(filepath.Dir(output), "hd-completion-opening-parent-native-v1-20261008", fmt.Sprintf("case%02d", i))
		frame := read(prefix + "-after.frame")
		render := func(f []byte) []byte {
			hd.Layer.Frame(f, make([]byte, 64000*3))
			rgba := make([]byte, 960*600*4)
			hd.Draw(rgba, 3)
			return rgba
		}
		hd.ResetForLoad()
		positive := render(frame)
		full := &image.NRGBA{Pix: positive, Stride: 960 * 4, Rect: image.Rect(0, 0, 960, 600)}
		crop := image.NewNRGBA(image.Rect(0, 0, 384, 192))
		draw.Draw(crop, crop.Rect, full, image.Pt(72, 360), draw.Src)
		f, er := os.OpenFile(filepath.Join(output, fmt.Sprintf("case%02d.png", i)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if er != nil {
			t.Fatal(er)
		}
		er = png.Encode(f, crop)
		ce := f.Close()
		if er != nil || ce != nil {
			t.Fatal(er, ce)
		}
		bad := append([]byte(nil), frame...)
		bad[120*320+24] ^= 1
		negative := render(bad)
		for yy := 120 * 3; yy < 184*3; yy++ {
			for xx := 24 * 3; xx < 152*3; xx++ {
				if xx >= 56*3 && xx < 64*3 && yy >= 152*3 && yy < 160*3 {
					continue
				}
				k := (yy*960 + xx) * 4
				if !bytes.Equal(negative[k:k+4], make([]byte, 4)) {
					t.Fatal("親子大圖失配殘留")
				}
			}
		}
		if !bytes.Equal(render(frame), positive) {
			t.Fatal("失配恢復不同")
		}
		hd.ResetForLoad()
		if !bytes.Equal(render(frame), positive) {
			t.Fatal("讀檔重建不同")
		}
	}
}

func TestFixedSceneOriginalCopyArtOutputs(t *testing.T) {
	planPath, output, orig := os.Getenv("PSYCHICWAR_SCENE_COPY_PLAN"), os.Getenv("PSYCHICWAR_SCENE_COPY_OUTPUT"), os.Getenv("PSYCHICWAR_TEST_ORIG")
	if planPath == "" || output == "" || orig == "" {
		t.Skip("需明示816原版COPY及候選")
	}
	read := func(p string) []byte {
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	var plan struct {
		Schema, Theme string
		Rows          []struct {
			PBL        string
			Image      int
			Rect       [4]int
			Frame, PNG string
			Entry      ThemeEntry
		}
	}
	if e := json.Unmarshal(read(planPath), &plan); e != nil {
		t.Fatal(e)
	}
	if plan.Schema != "psychic-war-scene-copy-art-plan/1" || len(plan.Rows) != 816 {
		t.Fatal("COPY清單不同")
	}
	dir := t.TempDir()
	copied := map[string]bool{}
	entries := make([]ThemeEntry, 0, 816)
	for _, r := range plan.Rows {
		_, _, px, err := loadSceneImage(orig, r.Entry)
		if err != nil || len(px) != r.Rect[2]*r.Rect[3] {
			t.Fatal("原始來源", r.PBL, r.Image, err)
		}
		e := r.Entry
		e.PNG = filepath.Base(r.PNG)
		entries = append(entries, e)
		if !copied[e.PNG] {
			if err = os.WriteFile(filepath.Join(dir, e.PNG), read(r.PNG), 0644); err != nil {
				t.Fatal(err)
			}
			copied[e.PNG] = true
		}
	}
	manifest := ThemeManifest{Schema: "psychic-war-theme/2", Name: "copy816", Scale: 3, Entries: entries}
	raw, e := json.Marshal(manifest)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0644); e != nil {
		t.Fatal(e)
	}
	if e = os.Mkdir(output, 0755); e != nil {
		t.Fatal(e)
	}
	for i, r := range plan.Rows {
		// 隔離本次來源；大圖失配時仍可能有合法的小圖吻合，不能把它算成殘圖。
		selected := r.Entry
		selected.PNG = filepath.Base(r.PNG)
		manifest.Entries = []ThemeEntry{selected}
		raw, e = json.Marshal(manifest)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0644); e != nil {
			t.Fatal(e)
		}
		hd, notice, loadErr := LoadTheme(dir, orig, "", 3)
		if loadErr != nil || notice != "" {
			t.Fatal(notice, loadErr)
		}
		hd.ResetForLoad()
		frame := read(r.Frame)
		if len(frame) != 64000 {
			t.Fatal("原版畫格大小")
		}
		hd.Layer.Frame(frame, make([]byte, 64000*3))
		rgba := make([]byte, 960*600*4)
		hd.Draw(rgba, 3)
		x, y, w, h := r.Rect[0], r.Rect[1], r.Rect[2], r.Rect[3]
		x0, y0 := x/8*8, y/8*8
		cw, ch := ((x+w+7)/8*8-x0)*3, ((y+h+7)/8*8-y0)*3
		full := &image.NRGBA{Pix: rgba, Stride: 960 * 4, Rect: image.Rect(0, 0, 960, 600)}
		crop := image.NewNRGBA(image.Rect(0, 0, cw, ch))
		draw.Draw(crop, crop.Rect, full, image.Pt(x0*3, y0*3), draw.Src)
		if bytes.Equal(crop.Pix, make([]byte, len(crop.Pix))) {
			t.Fatal("候選未顯示", i, r.PBL, r.Image, r.Rect)
		}
		path := filepath.Join(output, fmt.Sprintf("case%03d.png", i))
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(f, crop)
		ce := f.Close()
		if err != nil || ce != nil {
			t.Fatal(err, ce)
		}
		bad := append([]byte(nil), frame...)
		bad[y*320+x] ^= 1
		hd.Layer.Frame(bad, make([]byte, 64000*3))
		negative := make([]byte, len(rgba))
		hd.Draw(negative, 3)
		// 本次來源失配必須撤圖；獨立RGBA工具仍比較完整候選及8×8遮罩。
		for yy := y * 3; yy < (y+h)*3; yy++ {
			for xx := x * 3; xx < (x+w)*3; xx++ {
				k := 4 * (yy*960 + xx)
				if !bytes.Equal(negative[k:k+4], make([]byte, 4)) {
					t.Fatal("失配殘圖", i)
				}
			}
		}
	}
	complete, notice, e := LoadTheme(plan.Theme, orig, "", 3)
	if e != nil || notice != "" || complete.battle == nil || complete.maze == nil {
		t.Fatal("完整候選載入", notice, e)
	}
}
