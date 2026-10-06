package theme

import (
	"bytes"
	"encoding/json"
	"github.com/wicanr2/dosgolem/oracle"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllyItemsIndependentPositions(t *testing.T) {
	orig := os.Getenv("PSYCHICWAR_TEST_ORIG")
	if orig == "" {
		t.Skip("需明示原版來源")
	}
	dir := t.TempDir()
	themeTestPNG(t, dir, "ally.png", 72, 96)
	bottom := ThemeEntry{PBL: "ALLY.PBL", Image: 0, At: []int{264, 152}, PNG: "ally.png", Kind: "redraw", Match: []int{248, 0, 72, 40}}
	top := bottom
	top.At = []int{128, 8}
	load := func(entries []ThemeEntry) (*Theme, error) {
		b, err := json.Marshal(ThemeManifest{Schema: "psychic-war-theme/1", Name: "items", Scale: 3, Entries: entries})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); err != nil {
			t.Fatal(err)
		}
		hd, _, err := LoadTheme(dir, orig, "", 3)
		return hd, err
	}
	hd, err := load([]ThemeEntry{bottom, top})
	if err != nil {
		t.Fatal(err)
	}
	base, err := themeBackground(orig)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range hd.groups {
		s := g.sprite
		for y := 0; y < s.h; y++ {
			copy(base[(s.y+y)*320+s.x:], s.indexed[y*s.w:(y+1)*s.w])
		}
	}
	hd.frameSprites(base)
	if !hd.groups[0].sprite.active || !hd.groups[1].sprite.active || hd.groups[0].sprite == hd.groups[1].sprite {
		t.Fatal("同圖的兩個位置必須獨立並同時可見")
	}
	// 冷載只保留下方完整來源，不能把上方認定共享給下方或反向共享。
	base[8*320+128] ^= 1
	hd.ResetForLoad()
	hd.frameSprites(base)
	if !hd.groups[0].sprite.active || hd.groups[1].sprite.active {
		t.Fatal("兩位置的來源認定互相干擾")
	}
	for _, mutate := range []func(*ThemeEntry){
		func(e *ThemeEntry) { e.Match = nil },
		func(e *ThemeEntry) { e.Match = []int{0, 0, 160, 40} },
		func(e *ThemeEntry) { e.At = []int{128, 12} },
		func(e *ThemeEntry) { e.Image = 12 },
		func(e *ThemeEntry) { e.Src = []int{0, 0, 16, 16} },
	} {
		bad := top
		mutate(&bad)
		if _, err := load([]ThemeEntry{bad}); err == nil {
			t.Fatal("未證實的位置、錨點、來源應拒絕")
		}
	}
	if _, err := load([]ThemeEntry{bottom, top, top}); err == nil {
		t.Fatal("同位置重複來源應拒絕")
	}
}

func themeTestPNG(t *testing.T, dir, name string, w, h int) {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 200, G: 120, B: 40, A: 128})
		}
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestThemeMenuGridAndRecovery(t *testing.T) {
	dir := t.TempDir()
	themeTestPNG(t, dir, "menu.png", 264, 216)
	base := make([]byte, 320*200)
	entry := ThemeEntry{PBL: "MENU.PBL", Image: 0, At: []int{160, 4}, PNG: "menu.png", Kind: "redraw", Match: []int{0, 0, 8, 4}}
	g, err := loadThemeEntry(dir, 0, entry, 3, base)
	if err != nil {
		t.Fatal(err)
	}
	hd := &Theme{Enabled: true, groups: []themeGroup{g}}
	hd.Layer.W, hd.Layer.H = 320, 200
	hd.ResetForLoad()
	rgb := make([]byte, 320*200*3)
	hd.Layer.Frame(base, rgb)
	got := make([]byte, 320*200*3*3*4)
	if !hd.Draw(got, 3) {
		t.Fatal("未畫主題")
	}
	if len(hd.Layer.Art) != 10 || hd.Layer.Art[0].Y != 0 || hd.Layer.Art[9].Y != 72 {
		t.Fatal("MENU 未依全畫面 8 格補齊")
	}
	if got[4*(0*960+160*3)+3] != 0 || got[4*(4*3*960+160*3)+3] != 128 {
		t.Fatal("上下補格或透明度不符")
	}
	before := append([]byte(nil), got...)
	dynamic := append([]byte(nil), base...)
	dynamic[4*320+160] = 1
	hd.Layer.Frame(dynamic, rgb)
	clear(got)
	hd.Draw(got, 3)
	if got[4*(4*3*960+160*3)+3] != 0 || got[4*(8*3*960+160*3)+3] != 128 {
		t.Fatal("改動一像素應只遮原點對齊的完整格")
	}
	hd.Layer.Frame(base, rgb)
	clear(got)
	hd.Draw(got, 3)
	if !bytes.Equal(before, got) {
		t.Fatal("背景恢復後未恢復主題")
	}
	hd.Enabled = false
	clear(got)
	if hd.Draw(got, 3) {
		t.Fatal("停用仍繪圖")
	}
	hd.ResetForLoad()
	if hd.Enabled || len(hd.Layer.Art) != 0 || hd.Layer.Watchers() != 1 {
		t.Fatal("讀檔重登記改動開關或保留舊圖面")
	}
	hd.Enabled = true
	hd.Layer.Frame(base, rgb)
	clear(got)
	hd.Draw(got, 3)
	if !bytes.Equal(before, got) {
		t.Fatal("讀檔重登記未重建完整圖面")
	}
}

func TestThemeRejectInvalidAssets(t *testing.T) {
	dir := t.TempDir()
	themeTestPNG(t, dir, "ok.png", 960, 120)
	good := ThemeEntry{PBL: "SCREEN.PBL", Image: 0, At: []int{0, 0}, PNG: "ok.png", Kind: "redraw"}
	for _, name := range []string{"missing", "size", "position", "source", "rect", "match", "parent", "absolute", "symlink"} {
		t.Run(name, func(t *testing.T) {
			e := good
			switch name {
			case "missing":
				e.PNG = "missing.png"
			case "size":
				themeTestPNG(t, dir, "wrong.png", 1, 1)
				e.PNG = "wrong.png"
			case "position":
				e.At = []int{1, 0}
			case "source":
				e.Image = 5
			case "rect":
				e.Src = []int{0, 0, 321, 40}
			case "match":
				e.Match = []int{0, 0, 0, 40}
			case "parent":
				e.PNG = "../ok.png"
			case "absolute":
				e.PNG = filepath.Join(dir, "ok.png")
			case "symlink":
				outside := t.TempDir()
				themeTestPNG(t, outside, "outside.png", 960, 120)
				if err := os.Symlink(filepath.Join(outside, "outside.png"), filepath.Join(dir, "link.png")); err != nil {
					t.Fatal(err)
				}
				e.PNG = "link.png"
			}
			if _, err := loadThemeEntry(dir, 0, e, 3, make([]byte, 64000)); err == nil {
				t.Fatal("錯誤資產未拒絕")
			}
		})
	}
}

func TestThemeManifestAndScale(t *testing.T) {
	if hd, notice, err := LoadTheme("", "missing", "theme", 3); hd != nil || notice != "" || err != nil {
		t.Fatal("預設不應讀資料")
	}
	dir := t.TempDir()
	m := ThemeManifest{Schema: "psychic-war-theme/1", Name: "hd", Scale: 3, Entries: []ThemeEntry{{PBL: "SCREEN.PBL", Image: 0, At: []int{0, 0}, PNG: "missing.png", Kind: "redraw"}}}
	b, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
	if hd, notice, err := LoadTheme(dir, "missing", "theme", 2); hd != nil || notice == "" || err != nil {
		t.Fatal("倍率不符不應讀缺少的 PNG／PBL")
	}
	for _, invalid := range []string{string(b) + " {}", strings.Replace(string(b), `"name":"hd"`, `"unknown":true,"name":"hd"`, 1), strings.Replace(string(b), `"image":0,`, "", 1), strings.Replace(string(b), `"image":0`, `"image":null`, 1)} {
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(invalid), 0644); err != nil {
			t.Fatal(err)
		}
		if _, _, err := LoadTheme(dir, "missing", "theme", 2); err == nil {
			t.Fatal("清單錯誤未拒絕")
		}
	}
}

func TestThemeRealBackground(t *testing.T) {
	orig := os.Getenv("PSYCHICWAR_TEST_ORIG")
	if orig == "" {
		t.Skip("真實素材驗證需明示 PSYCHICWAR_TEST_ORIG")
	}
	got, err := themeBackground(orig)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join(orig, "..", "..", "hd", "bg.idx"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, expected) {
		t.Fatal("由實際 PBL 重建的背景與獨立基準不同")
	}
	bad := t.TempDir()
	if err := os.WriteFile(filepath.Join(bad, "SCREEN.PBL"), []byte{2, 0, 1, 1}, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := themeBackground(bad); err == nil {
		t.Fatal("錯誤原版來源未拒絕")
	}
}

func TestThemePremultipliedOutput(t *testing.T) {
	p := []byte{200, 120, 40, 128, 10, 20, 30, 255, 10, 20, 30, 0}
	PremultiplyRGBA(p)
	if !bytes.Equal(p, []byte{100, 60, 20, 128, 10, 20, 30, 255, 0, 0, 0, 0}) {
		t.Fatal(p)
	}
}

func TestAllyGridPresenceAndRecovery(t *testing.T) {
	dir := t.TempDir()
	themeTestPNG(t, dir, "ally.png", 72, 96)
	base := make([]byte, 64000)
	// 左上格無原版墨跡：即使 HD 有顏色也不能在清空後留下殘影。
	for y := 152; y < 184; y++ {
		for x := 264; x < 288; x++ {
			if x >= 272 || y >= 160 {
				base[y*320+x] = 3
			}
		}
	}
	px := make([]byte, 24*32)
	for y := 0; y < 32; y++ {
		copy(px[y*24:(y+1)*24], base[(152+y)*320+264:(152+y)*320+288])
	}
	s := &spritePresence{indexed: px, packed: []byte{1, 2, 3}, x: 264, y: 152, w: 24, h: 32}
	e := ThemeEntry{PBL: "ALLY.PBL", At: []int{264, 152}, PNG: "ally.png", Kind: "redraw"}
	g, err := loadThemeEntry(dir, 0, e, 3, base)
	if err != nil {
		t.Fatal(err)
	}
	g.sprite = s
	hd := &Theme{Enabled: true, groups: []themeGroup{g}}
	hd.Layer.W, hd.Layer.H = 320, 200
	hd.ResetForLoad()
	render := func(frame []byte) []byte {
		g := &hd.groups[0]
		s.frame(frame, g.watch)
		g.registered.Want = nil
		if s.active {
			g.registered.Want = g.watch.Want
		}
		hd.Layer.Frame(frame, make([]byte, 64000*3))
		out := make([]byte, 960*600*4)
		hd.Draw(out, 3)
		return out
	}
	alpha := func(p []byte, x, y int) byte { return p[4*(y*3*960+x*3)+3] }
	initial := render(base)
	if !s.active || alpha(initial, 264, 152) != 0 || alpha(initial, 272, 152) != 128 {
		t.Fatal("載入既有角色辨識或空格透明不符")
	}
	covered := append([]byte(nil), base...)
	covered[153*320+273] = 5
	p := render(covered)
	if !s.active || alpha(p, 272, 152) != 0 || alpha(p, 280, 152) != 128 {
		t.Fatal("局部遮擋撤下整張角色或未遮完整格")
	}
	if !bytes.Equal(render(base), initial) {
		t.Fatal("恢復原版角色未恢復 HD")
	}
	clearFrame := append([]byte(nil), base...)
	for y := 152; y < 184; y++ {
		clear(clearFrame[y*320+264 : y*320+288])
	}
	if p = render(clearFrame); bytes.Count(p, []byte{0}) != len(p) {
		t.Fatal("角色清空留下 HD 殘影")
	}
	other := oracle.Regs{CX: 0x4226, DX: 0x0304}
	s.blit(other, func() []byte { return []byte{9} })
	if s.active {
		t.Fatal("另一個完整貼圖未撤銷角色")
	}
	if p = render(covered); alpha(p, 280, 152) != 0 {
		t.Fatal("未知換圖後局部吻合誤恢復角色")
	}
	s.blit(other, func() []byte { return s.packed })
	if !s.active || alpha(render(covered), 280, 152) != 128 {
		t.Fatal("已證實來源未恢復未被遮格")
	}
	other.AX = 1
	s.blit(other, func() []byte { t.Fatal("未知 AL 仍讀完整來源"); return nil })
	if s.active {
		t.Fatal("未知 AL 模式沿用完整圖來源")
	}
	s.active = true
	s.blit(oracle.Regs{CX: 0x0101, DX: 0x0101}, func() []byte { t.Fatal("無關貼圖讀角色來源"); return nil })
	if !s.active {
		t.Fatal("無關貼圖撤銷角色")
	}
	noAnchor := append([]byte(nil), base...)
	noAnchor[0] = 4
	render(noAnchor)
	if s.active {
		t.Fatal("背景失配沿用舊來源")
	}
	hd.Enabled = false
	hd.ResetForLoad()
	if s.active || hd.Enabled {
		t.Fatal("讀檔保留角色來源或改 HD 開關")
	}
	hd.Enabled = true
	if !bytes.Equal(render(base), initial) {
		t.Fatal("讀檔後未從原版重建角色")
	}
	for _, change := range []func(*ThemeEntry){
		func(e *ThemeEntry) { e.Image = 1 }, func(e *ThemeEntry) { e.At = []int{260, 152} },
		func(e *ThemeEntry) { e.Src = []int{0, 0, 16, 16} }, func(e *ThemeEntry) { e.Match = []int{264, 152, 24, 32} },
	} {
		bad := e
		change(&bad)
		if _, err := loadThemeEntry(dir, 0, bad, 3, base); err == nil {
			t.Fatal("未證實位置／來源未拒絕")
		}
	}
}

func TestAllyRealSourceAndAttach(t *testing.T) {
	orig := os.Getenv("PSYCHICWAR_TEST_ORIG")
	if orig == "" {
		t.Skip("真實素材驗證需明示 PSYCHICWAR_TEST_ORIG")
	}
	s, err := loadAlly0(orig)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.indexed) != 768 || len(s.packed) != 384 {
		t.Fatal("角色來源長度不符")
	}
	if _, err := loadAlly0(t.TempDir()); err == nil {
		t.Fatal("缺原版角色來源未拒絕")
	}
	a, err := oracle.Load(filepath.Join(orig, "PW.EXE"), orig)
	if err != nil {
		t.Fatal(err)
	}
	b, err := oracle.Load(filepath.Join(orig, "PW.EXE"), orig)
	if err != nil {
		t.Fatal(err)
	}
	hd := &Theme{groups: []themeGroup{{sprite: s}}}
	if hd.Attach(nil) == nil {
		t.Fatal("空 Oracle 未拒絕")
	}
	if err := hd.Attach(a); err != nil {
		t.Fatal(err)
	}
	if err := hd.Attach(a); err != nil {
		t.Fatal("同 Oracle 不可重複接上", err)
	}
	if hd.Attach(b) == nil {
		t.Fatal("第二個 Oracle 未拒絕")
	}
}
