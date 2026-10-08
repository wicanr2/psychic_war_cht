package theme

// 024 §1.57：只在原位完整內容吻合時顯示固定場景，原版資料保持唯讀。
import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
)

func isSceneSource(name string) bool {
	return name == "MAP.PBL" || name == "OPEN.PBL" || name == "END0.PBL" || name == "END1.PBL"
}

func fixedScenePosition(e ThemeEntry) (x, y, w, h int, err error) {
	err = fmt.Errorf("%s #%d不是READY的完整COPY來源或原位", e.PBL, e.Image)
	if len(e.At) != 2 || e.Image < 0 {
		return
	}
	switch e.PBL {
	case "MAP.PBL":
		if e.Image >= 18 {
			return
		}
		x, y, w, h = 4, 4, 152, 80
	case "OPEN.PBL":
		switch {
		case e.Image < 4:
			x, y, w, h = 24, 120, 128, 64
		case e.Image == 5 || e.Image == 6:
			x, y, w, h = 56, 152, 8, 8
		default:
			return
		}
	case "END0.PBL":
		if e.Image >= 2 {
			return
		}
		x, y, w, h = 56, 80, 48, 48
	case "END1.PBL":
		if e.Image >= 18 {
			return
		}
		w, h = 24, 32
		if e.Image < 4 {
			w = 72
		}
		x, y = e.At[0], 152
		if x < 4 || x+w > 320 || x%4 != 0 {
			return
		}
		if e.Image < 14 && ((x/4-1)%2 == 0) != (e.Image%2 == 1) {
			return
		}
	default:
		return
	}
	if e.At[0] != x || e.At[1] != y || e.Kind != "redraw" || e.Scaler != "" ||
		(e.Src != nil && !equalInts(e.Src, []int{0, 0, w, h})) ||
		(e.Match != nil && !equalInts(e.Match, []int{x, y, w, h})) {
		return
	}
	err = nil
	return
}

// 024 §1.59：OPEN5與大圖裁切相同；OPEN6只替換已證實的同一區塊。
func (t *Theme) loadOpeningParents(root, orig string, m ThemeManifest) error {
	entries := map[int]ThemeEntry{}
	for _, e := range m.Entries {
		if e.PBL == "OPEN.PBL" && (e.Image == 3 || e.Image == 5 || e.Image == 6) {
			entries[e.Image] = e
		}
	}
	parent, ok := entries[3]
	if !ok {
		return nil
	}
	_, _, original, err := loadSceneImage(orig, parent)
	if err != nil {
		return err
	}
	cache := map[string]*image.NRGBA{}
	base, err := battlePNG(root, parent.PNG, 128, 64, 3, cache)
	if err != nil {
		return err
	}
	if child, ok := entries[5]; ok {
		_, _, px, err := loadSceneImage(orig, child)
		if err != nil {
			return err
		}
		art, err := battlePNG(root, child.PNG, 8, 8, 3, cache)
		if err != nil {
			return err
		}
		for y := 0; y < 8; y++ {
			if !bytes.Equal(original[(32+y)*128+32:(32+y)*128+40], px[y*8:(y+1)*8]) {
				return fmt.Errorf("OPEN5原版親子來源不符")
			}
		}
		for y := 0; y < 24; y++ {
			if !bytes.Equal(base.Pix[(96+y)*base.Stride+96*4:(96+y)*base.Stride+120*4], art.Pix[y*art.Stride:y*art.Stride+24*4]) {
				return fmt.Errorf("OPEN5必須共用OPEN3的同位置HD紋理")
			}
		}
	}
	child, ok := entries[6]
	if !ok {
		return nil
	}
	_, _, px, err := loadSceneImage(orig, child)
	if err != nil {
		return err
	}
	art, err := battlePNG(root, child.PNG, 8, 8, 3, cache)
	if err != nil {
		return err
	}
	ref := make([]byte, 64000)
	for y := 0; y < 64; y++ {
		copy(ref[(120+y)*320+24:(120+y)*320+152], original[y*128:(y+1)*128])
	}
	for y := 0; y < 8; y++ {
		copy(ref[(152+y)*320+56:(152+y)*320+64], px[y*8:(y+1)*8])
	}
	g, err := loadThemeEntry(root, len(m.Entries), parent, 3, ref, true)
	if err != nil {
		return err
	}
	g.sceneCopy = true
	for _, row := range g.rows {
		if row.Y != 152 {
			continue
		}
		stride := row.Cells * row.CellW * row.PixScale * 4
		for y := 0; y < 24; y++ {
			at := y*stride + (56-row.X)*3*4
			copy(row.Pix[at:at+24*4], art.Pix[y*art.Stride:y*art.Stride+24*4])
		}
	}
	t.groups = append(t.groups, g)
	return nil
}

func loadSceneImage(orig string, e ThemeEntry) (w, h int, px []byte, err error) {
	_, _, w, h, err = fixedScenePosition(e)
	if err != nil {
		return
	}
	raw, err := sceneArchive(orig, e.PBL)
	if err != nil {
		return
	}
	var aw, ah int
	aw, ah, px, err = pbl.Decode(raw, e.Image)
	if err != nil || aw != w || ah != h {
		err = fmt.Errorf("%s #%d原始形狀不符", e.PBL, e.Image)
		return
	}
	return
}

func sceneArchive(orig, name string) ([]byte, error) {
	hashes := map[string]string{
		"MAP.PBL":  "72864a618cbe72b6a53f73d17dae6eb2578724dad23f2966e1885eb4847ae4df",
		"OPEN.PBL": "928f248ed567f2bed44ed7f186e5a0160371a61f2db334c1c8d896899463d822",
		"END0.PBL": "309bc39b16add885b3dd869e31f397bd7d5487fb357a2ead78e3a7c0c4c4358d",
		"END1.PBL": "0b26a40711cdcd230cf88d1977b615a3e7eb2181e299335549172050cbdd8a02",
	}
	counts := map[string]int{"MAP.PBL": 18, "OPEN.PBL": 11, "END0.PBL": 2, "END1.PBL": 19}
	raw, err := os.ReadFile(filepath.Join(orig, name))
	if err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != hashes[name] {
		return nil, fmt.Errorf("%s原始SHA-256不符", name)
	}
	offsets, err := pbl.Offsets(raw)
	if err != nil || len(offsets) != counts[name] {
		return nil, fmt.Errorf("%s實際archive圖數不符", name)
	}
	return raw, nil
}
