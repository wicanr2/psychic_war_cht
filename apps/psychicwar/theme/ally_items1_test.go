package theme

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/dosgolem/oracle"
)

// 024 §1.17：正常道具來源互換與三處角色的獨立生命週期。
func TestAlly1ItemsSourceLifecycle(t *testing.T) {
	orig := os.Getenv("PSYCHICWAR_TEST_ORIG")
	frames := os.Getenv("PSYCHICWAR_ALLY1_EVIDENCE")
	if orig == "" || frames == "" {
		t.Skip("需明示原版與正常ALLY #1道具收據")
	}
	dir := t.TempDir()
	themeTestPNG(t, dir, "ally.png", 72, 96)
	top := ThemeEntry{PBL: "ALLY.PBL", Image: 1, At: []int{128, 8}, PNG: "ally.png", Kind: "redraw", Match: []int{248, 0, 72, 40}}
	topKai := top
	topKai.Image = 0
	bottom := top
	bottom.At = []int{232, 152}
	bottomKai := topKai
	bottomKai.At = []int{264, 152}
	load := func(entries []ThemeEntry) (*Theme, error) {
		b, err := json.Marshal(ThemeManifest{Schema: "psychic-war-theme/1", Name: "items", Scale: 3, Entries: entries})
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); err != nil {
			t.Fatal(err)
		}
		hd, _, err := LoadTheme(dir, orig, "", 3)
		return hd, err
	}
	hd, err := load([]ThemeEntry{topKai, top, bottomKai, bottom})
	if err != nil {
		t.Fatal(err)
	}
	read := func(name string) []byte {
		p, err := os.ReadFile(filepath.Join(frames, name))
		if err != nil || len(p) != 64000 {
			t.Fatal(name, err)
		}
		return p
	}
	render := func(frame []byte) []byte {
		hd.frameSprites(frame)
		hd.Layer.Frame(frame, make([]byte, 192000))
		p := make([]byte, 960*600*4)
		hd.Draw(p, 3)
		return p
	}
	alpha := func(p []byte, x, y int) byte { return p[4*(y*3*960+x*3)+3] }
	menu := read("ally1-items-menu-v1/after.state.frame")
	render(menu)
	if !hd.groups[0].sprite.active || hd.groups[1].sprite.active || !hd.groups[2].sprite.active || !hd.groups[3].sprite.active {
		t.Fatal("第一名道具肖像與下方隊伍來源不同")
	}
	before := read("ally1-items-source-v1-draw000-before.frame")
	after := read("ally1-items-source-v1-draw000-after.frame")
	hd.blitSprites(oracle.Regs{AX: 0x2000, CX: 0x2002, DX: 0x0304},
		func() []byte { return hd.groups[1].sprite.packed }, func() []byte { return before })
	p := render(before)
	if alpha(p, 140, 23) != 0 || hd.groups[0].sprite.active || !hd.groups[1].sprite.active {
		t.Fatal("AL0進行中未撤下舊道具肖像，或提前顯示新肖像")
	}
	hd.finishBlit()
	initial := render(after)
	if hd.groups[0].sprite.active || !hd.groups[1].sprite.active || alpha(initial, 140, 23) != 128 {
		t.Fatal("貼圖完成後未切到唯一ALLY #1來源")
	}
	if !hd.groups[2].sprite.active || !hd.groups[3].sprite.active || alpha(initial, 244, 167) != 128 || alpha(initial, 276, 167) != 128 {
		t.Fatal("道具來源切換干擾下方兩名盟友")
	}
	covered := append([]byte(nil), after...)
	covered[25*320+141] ^= 1
	p = render(covered)
	if alpha(p, 140, 25) != 0 || alpha(p, 148, 25) != 128 || alpha(p, 244, 167) != 128 {
		t.Fatal("道具肖像單像素未遮完整8×8格，或干擾鄰格／隊伍")
	}
	if !bytes.Equal(render(after), initial) {
		t.Fatal("恢復原版後未恢復圖面")
	}
	hd.Enabled = false
	hd.ResetForLoad()
	if !bytes.Equal(render(after), make([]byte, len(initial))) {
		t.Fatal("HD停用仍畫道具肖像")
	}
	hd.Enabled = true
	if !bytes.Equal(render(after), initial) {
		t.Fatal("成功載回後未由完整原圖重建")
	}
	items := read("ally1-items-return-v1/after.state.frame")
	p = render(items)
	for y := 8; y < 40; y++ {
		for x := 128; x < 152; x++ {
			if alpha(p, x, y) != 0 {
				t.Fatal("正常道具列表清除肖像後留下HD殘影")
			}
		}
	}
	if alpha(p, 244, 167) != 128 || alpha(p, 276, 167) != 128 {
		t.Fatal("道具列表清除肖像干擾下方盟友")
	}
	for _, mutate := range []func(*ThemeEntry){
		func(e *ThemeEntry) { e.At = []int{136, 8} },
		func(e *ThemeEntry) { e.Match = nil },
		func(e *ThemeEntry) { e.Match = []int{0, 0, 160, 40} },
		func(e *ThemeEntry) { e.Src = []int{0, 0, 16, 16} },
	} {
		bad := top
		mutate(&bad)
		if _, err := load([]ThemeEntry{bad}); err == nil {
			t.Fatal("未證實的道具位置／錨點／裁切未拒絕", bad)
		}
	}
	if _, err := load([]ThemeEntry{top, top}); err == nil {
		t.Fatal("同來源／位置重複未拒絕")
	}
}
