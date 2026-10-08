package theme

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// 正式候選PNG與原版COPY畫格；期望由外部獨立像素工具計算。
func TestExpandedRoomsCandidateArtOutputs(t *testing.T) {
	planPath, output, orig := os.Getenv("PSYCHICWAR_ROOM63_ART_PLAN"), os.Getenv("PSYCHICWAR_ROOM63_ART_OUTPUT"), os.Getenv("PSYCHICWAR_TEST_ORIG")
	if planPath == "" || output == "" || orig == "" {
		t.Skip("需明示正式房間候選及獨立圖面輸出")
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
			Frame, PNG string
			Entry      ThemeEntry
		}
	}
	if e := json.Unmarshal(read(planPath), &plan); e != nil {
		t.Fatal(e)
	}
	if plan.Schema != "psychic-war-room63-art-plan/1" || len(plan.Rows) != 63 {
		t.Fatal("房間候選清單不符")
	}
	dir := t.TempDir()
	entries := make([]ThemeEntry, 0, 63)
	copied := map[string]bool{}
	for _, r := range plan.Rows {
		e := r.Entry
		name := filepath.Base(r.PNG)
		e.PNG = name
		entries = append(entries, e)
		if !copied[name] {
			if er := os.WriteFile(filepath.Join(dir, name), read(r.PNG), 0644); er != nil {
				t.Fatal(er)
			}
			copied[name] = true
		}
	}
	manifest := ThemeManifest{Schema: "psychic-war-theme/2", Name: "candidate-room63", Scale: 3, Entries: entries}
	raw, e := json.Marshal(manifest)
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
	if e = os.Mkdir(output, 0755); e != nil {
		t.Fatal(e)
	}
	for _, r := range plan.Rows {
		hd.ResetForLoad()
		frame := read(r.Frame)
		if len(frame) != 64000 {
			t.Fatal("原版画格大小")
		}
		hd.Layer.Frame(frame, make([]byte, 64000*3))
		rgba := make([]byte, 960*600*4)
		hd.Draw(rgba, 3)
		full := &image.NRGBA{Pix: rgba, Stride: 960 * 4, Rect: image.Rect(0, 0, 960, 600)}
		crop := image.NewNRGBA(image.Rect(0, 0, 240, 240))
		draw.Draw(crop, crop.Rect, full, image.Pt(0, 360), draw.Src)
		if bytes.Equal(crop.Pix, make([]byte, len(crop.Pix))) {
			t.Fatal("房間候選未畫", r.PBL, r.Image)
		}
		path := filepath.Join(output, fmt.Sprintf("%s-%02d.png", r.PBL, r.Image))
		f, er := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if er != nil {
			t.Fatal(er)
		}
		er = png.Encode(f, crop)
		ce := f.Close()
		if er != nil || ce != nil {
			t.Fatal(er, ce)
		}
		bad := append([]byte(nil), frame...)
		bad[124*320+4] ^= 1
		hd.Layer.Frame(bad, make([]byte, 64000*3))
		negative := make([]byte, len(rgba))
		hd.Draw(negative, 3)
		if !bytes.Equal(negative, make([]byte, len(negative))) {
			t.Fatal("失配後殘圖", r.PBL, r.Image)
		}
	}
	// 完整主題亦需實際載入，保持迷宮與效果模型接入。
	complete, notice, e := LoadTheme(plan.Theme, orig, "", 3)
	if e != nil || notice != "" || complete.maze == nil || complete.battle == nil {
		t.Fatal("完整主題載入", notice, e)
	}
}

// 原始來源與正對照frame來自63次原版COPY及獨立Python收據。
func TestExpandedRoomsOriginalSourcesAndConditions(t *testing.T) {
	orig, proofPath := os.Getenv("PSYCHICWAR_TEST_ORIG"), os.Getenv("PSYCHICWAR_ROOM63_PROOF")
	if orig == "" || proofPath == "" {
		t.Skip("需明示63原版COPY來源及收據")
	}
	read := func(path string) []byte {
		b, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	var proof struct {
		Rows []struct {
			PBL   string
			Image int
			Rect  [4]int
			SHA   string `json:"source_sha256"`
		}
	}
	if e := json.Unmarshal(read(proofPath), &proof); e != nil {
		t.Fatal(e)
	}
	if len(proof.Rows) != 63 {
		t.Fatal("原版來源數不同")
	}
	for _, row := range proof.Rows {
		t.Run(fmt.Sprintf("%s/%d", row.PBL, row.Image), func(t *testing.T) {
			px, e := loadRoomImageNamed(orig, row.PBL, row.Image, true)
			if e != nil {
				t.Fatal(e)
			}
			if len(px) != 72*72 || fmt.Sprintf("%x", sha256.Sum256(battlePack(px))) != row.SHA || row.Rect != [4]int{4, 124, 72, 72} {
				t.Fatal("原始來源／原位不同")
			}
			frame, e := themeBackground(orig)
			if e != nil {
				t.Fatal(e)
			}
			for y := 0; y < 72; y++ {
				copy(frame[(124+y)*320+4:(124+y)*320+76], px[y*72:(y+1)*72])
			}
			dir := t.TempDir()
			themeTestPNG(t, dir, "room.png", 216, 216)
			entry := ThemeEntry{PBL: row.PBL, Image: row.Image, At: []int{4, 124}, PNG: "room.png", Kind: "redraw", Match: []int{4, 124, 72, 72}}
			m := ThemeManifest{Schema: "psychic-war-theme/2", Name: "room63", Scale: 3, Entries: []ThemeEntry{entry}}
			write := func() {
				b, e := json.Marshal(m)
				if e != nil {
					t.Fatal(e)
				}
				if e = os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); e != nil {
					t.Fatal(e)
				}
			}
			write()
			hd, notice, e := LoadTheme(dir, orig, "", 3)
			if e != nil || notice != "" || hd == nil {
				t.Fatal("schema2原圖", notice, e)
			}
			render := func(f []byte) []byte {
				hd.Layer.Frame(f, make([]byte, 64000*3))
				dst := make([]byte, 960*600*4)
				hd.Draw(dst, 3)
				return dst
			}
			initial := render(frame)
			if bytes.Equal(initial, make([]byte, len(initial))) {
				t.Fatal("完整內容正對照未畫")
			}
			bad := append([]byte(nil), frame...)
			bad[124*320+4] ^= 1
			if !bytes.Equal(render(bad), make([]byte, len(initial))) {
				t.Fatal("單像素失配仍畫")
			}
			if !bytes.Equal(render(frame), initial) {
				t.Fatal("恢復失配後不同")
			}
			hd.ResetForLoad()
			if !bytes.Equal(render(frame), initial) {
				t.Fatal("讀檔重建不同")
			}
			hd.Enabled = false
			if !bytes.Equal(render(frame), make([]byte, len(initial))) {
				t.Fatal("HD關閉仍畫")
			}
			m.Entries[0].At = []int{0, 120}
			write()
			if _, _, e = LoadTheme(dir, orig, "", 3); e == nil {
				t.Fatal("錯原位被接受")
			}
			m.Entries[0] = entry
			m.Schema = "psychic-war-theme/1"
			write()
			_, _, e = LoadTheme(dir, orig, "", 3)
			if (e == nil) != (row.PBL == "ROOM0.PBL" && validRoomImage(row.Image)) {
				t.Fatal("schema1契約改變", e)
			}
		})
	}
}

func TestRoom14And17AliasSingleRegistration(t *testing.T) {
	orig := os.Getenv("PSYCHICWAR_TEST_ORIG")
	if orig == "" {
		t.Skip("需真實ROOM0原圖")
	}
	dir := t.TempDir()
	themeTestPNG(t, dir, "a.png", 216, 216)
	themeTestPNG(t, dir, "b.png", 216, 216)
	first := ThemeEntry{PBL: "ROOM0.PBL", Image: 14, At: []int{4, 124}, PNG: "a.png", Kind: "redraw", Match: []int{4, 124, 72, 72}}
	second := first
	second.Image = 17
	second.PNG = "b.png"
	m := ThemeManifest{Schema: "psychic-war-theme/2", Name: "room-alias", Scale: 3, Entries: []ThemeEntry{first, second}}
	write := func() {
		b, e := json.Marshal(m)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); e != nil {
			t.Fatal(e)
		}
	}
	write()
	hd, _, e := LoadTheme(dir, orig, "", 3)
	if e != nil || hd == nil {
		t.Fatal(e)
	}
	if hd.Layer.Watchers() != 1 || len(hd.groups) != 1 {
		t.Fatal("別名重複繪製")
	}
	b, e := os.ReadFile(filepath.Join(dir, "b.png"))
	if e != nil {
		t.Fatal(e)
	}
	b[len(b)-1] ^= 1
	if e = os.WriteFile(filepath.Join(dir, "b.png"), b, 0644); e != nil {
		t.Fatal(e)
	}
	if _, _, e = LoadTheme(dir, orig, "", 3); e == nil {
		t.Fatal("不同別名PNG被接受")
	}
	if e = os.WriteFile(filepath.Join(dir, "b.png"), b[:len(b)-1], 0644); e != nil {
		t.Fatal(e)
	}
	m.Entries[1] = first
	write()
	if _, _, e = LoadTheme(dir, orig, "", 3); e == nil {
		t.Fatal("同圖號重複被接受")
	}
}
