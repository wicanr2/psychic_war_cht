package theme

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestSceneEffectsOriginalBoundaries(t *testing.T) {
	root, orig, out := os.Getenv("PSYCHICWAR_SCENE_EFFECT_THEME"), os.Getenv("PSYCHICWAR_TEST_ORIG"), os.Getenv("PSYCHICWAR_SCENE_EFFECT_OUTPUT")
	if root == "" || orig == "" || out == "" {
		t.Skip("需明示原版92組XOR與實際候選")
	}
	B := filepath.Join(filepath.Dir(out))
	read := func(p string) []byte {
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	var manifest ThemeManifest
	if e := json.Unmarshal(read(filepath.Join(root, "manifest.json")), &manifest); e != nil {
		t.Fatal(e)
	}
	effects, e := loadSceneEffects(root, orig, manifest)
	if e != nil {
		t.Fatal(e)
	}
	if effects == nil || len(effects.sources) != 92 {
		t.Fatal("XOR原位數", effects)
	}
	var palette [256][3]uint8
	if e := json.Unmarshal(read(filepath.Join(B, "hd-completion-opening-parent-native-v1-20261008/palette.json")), &palette); e != nil {
		t.Fatal(e)
	}
	type row struct {
		PBL    string          `json:"pbl"`
		Image  int             `json:"image"`
		Rect   []int           `json:"rect"`
		Prefix string          `json:"prefix"`
		Mode   json.RawMessage `json:"mode"`
	}
	var cases []row
	for _, name := range []string{"hd-completion-static-scenes-native-v1-20261008", "hd-completion-end1-native-v2-20261008"} {
		var plan struct {
			Rows []row `json:"rows"`
		}
		if e := json.Unmarshal(read(filepath.Join(B, name, "execution.json")), &plan); e != nil {
			t.Fatal(e)
		}
		for _, r := range plan.Rows {
			if string(r.Mode) != "1" && string(r.Mode) != "\"XOR\"" {
				continue
			}
			if r.PBL == "" {
				r.PBL = "END1.PBL"
			}
			cases = append(cases, r)
		}
	}
	if len(cases) != 92 {
		t.Fatal("原版案例數", len(cases))
	}
	if e := os.Mkdir(out, 0755); e != nil {
		t.Fatal(e)
	}
	write := func(name string, pix []byte) {
		f, e := os.OpenFile(filepath.Join(out, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if e != nil {
			t.Fatal(e)
		}
		e = png.Encode(f, &image.NRGBA{Pix: pix, Stride: 960 * 4, Rect: image.Rect(0, 0, 960, 600)})
		ce := f.Close()
		if e != nil || ce != nil {
			t.Fatal(e, ce)
		}
	}
	// 已知OPEN0基底使用真實HD；結局此診斷初態無已知人物時保留原版基底。
	minimal := t.TempDir()
	var entries []ThemeEntry
	for _, a := range manifest.Entries {
		if a.PBL == "OPEN.PBL" && a.Image == 0 {
			entries = append(entries, a)
			if e := os.WriteFile(filepath.Join(minimal, a.PNG), read(filepath.Join(root, a.PNG)), 0644); e != nil {
				t.Fatal(e)
			}
		}
	}
	m := manifest
	m.Entries = entries
	m.Maze = nil
	m.BuiltinMasks = nil
	raw, _ := json.Marshal(m)
	if e := os.WriteFile(filepath.Join(minimal, "manifest.json"), raw, 0644); e != nil {
		t.Fatal(e)
	}
	hd, _, e := LoadTheme(minimal, orig, "", 3)
	if e != nil {
		t.Fatal(e)
	}
	for n, r := range cases {
		var source *sceneEffect
		for i := range effects.sources {
			s := &effects.sources[i]
			if s.entry.PBL == r.PBL && s.entry.Image == r.Image && s.x == r.Rect[0] && s.y == r.Rect[1] {
				source = s
				break
			}
		}
		if source == nil {
			t.Fatal(r)
		}
		before, after := read(r.Prefix+"-before.frame"), read(r.Prefix+"-after.frame")
		beforeHash, packedHash, artHash := sha256.Sum256(before), sha256.Sum256(source.packed), sha256.Sum256(source.art.Pix)
		base := nativeRGBA(before, palette)
		hd.ResetForLoad()
		hd.Layer.Frame(before, make([]byte, 64000*3))
		hd.Draw(base, 3)
		if r.PBL == "OPEN.PBL" {
			if !sceneEffectStartsVisible(hd, source, before) || sceneEffectStartsVisible(hd, source, after) {
				t.Fatal("冷載未辨識首圖與撤圖方向", r)
			}
		}
		effects.reset()
		if !effects.apply(source, source.packed, before, base, palette) || !bytes.Equal(effects.expected, after) {
			t.Fatal("原版XOR結果", r)
		}
		effects.frame(after, palette)
		actual := nativeRGBA(after, palette)
		if !effects.draw(actual, 3) {
			t.Fatal("未覆繪", r)
		}
		write(fmt.Sprintf("case%02d-base.png", n), base)
		write(fmt.Sprintf("case%02d.png", n), actual)
		bad := append([]byte(nil), after...)
		bad[source.y*320+source.x] ^= 1
		effects.frame(bad, palette)
		cell := (source.y/8)*40 + source.x/8
		if effects.valid[cell] {
			t.Fatal("局部失配保留舊格")
		}
		// 失效格不因下一幀像素巧合恢復。
		effects.frame(after, palette)
		if effects.valid[cell] {
			t.Fatal("失效格自行復活")
		}
		effects.reset()
		effects.apply(source, source.packed, before, base, palette)
		effects.frame(after, palette)
		if !effects.apply(source, source.packed, after, actual, palette) || !bytes.Equal(effects.expected, before) || effects.active != nil {
			t.Fatal("雙XOR未撤圖")
		}
		effects.frame(before, palette)
		restored := nativeRGBA(before, palette)
		effects.draw(restored, 3)
		if !bytes.Equal(restored, base) {
			t.Fatal("HD基底未恢復", r)
		}
		// XOR期間，無關格的原版輸出仍可更新；撤圖不得把舊畫面寫回顯示層。
		effects.reset()
		effects.apply(source, source.packed, before, base, palette)
		changedFrame := append([]byte(nil), after...)
		changedFrame[0] ^= 1
		effects.frame(changedFrame, palette)
		changedArt := nativeRGBA(changedFrame, palette)
		effects.draw(changedArt, 3)
		if !effects.apply(source, source.packed, changedFrame, changedArt, palette) || effects.active != nil {
			t.Fatal("無關格更新影響撤圖方向")
		}
		wantFrame := append([]byte(nil), before...)
		wantFrame[0] ^= 1
		effects.frame(wantFrame, palette)
		got := nativeRGBA(wantFrame, palette)
		effects.draw(got, 3)
		want := append([]byte(nil), base...)
		for yy := 0; yy < 3; yy++ {
			for xx := 0; xx < 3; xx++ {
				i := (yy*960 + xx) * 4
				c := palette[wantFrame[0]]
				copy(want[i:i+3], c[:])
				want[i+3] = 255
			}
		}
		if !bytes.Equal(got, want) {
			t.Fatal("撤圖覆蓋了原版的新輸出", r)
		}
		effects.reset()
		if effects.draw(restored, 3) {
			t.Fatal("冷載保留上一張")
		}
		changed := palette
		changed[1][0] ^= 1
		effects.apply(source, source.packed, before, base, palette)
		effects.frame(after, changed)
		if effects.hd != nil {
			t.Fatal("色盤失配保留")
		}
		malformed := append([]byte(nil), source.packed...)
		malformed[0] ^= 1
		if effects.apply(source, malformed, before, base, palette) || effects.hd != nil {
			t.Fatal("未知來源未退回")
		}
		if sha256.Sum256(before) != beforeHash || sha256.Sum256(source.packed) != packedHash || sha256.Sum256(source.art.Pix) != artHash {
			t.Fatal("唯讀輸入被修改")
		}
	}
	if _, _, _, _, e := sceneEffectPosition(ThemeEntry{PBL: "END1.PBL", Image: 18, At: []int{308, 155}, Kind: "redraw"}); e == nil {
		t.Fatal("越界來源")
	}
}
