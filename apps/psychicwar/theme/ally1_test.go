package theme

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/dosgolem/oracle"
)

// 真實F2→Team up→命名frame的來源及遮格回歸，證據入口研究038 §130–131。
func TestAlly1JoinedSourceLifecycle(t *testing.T) {
	testAllyJoinedSourceLifecycle(t, 1, "8c25534e872e9e435681b9d7948c9279ae5eee02d8bd53ee0c18d688a98489a6", "recruit-shulosu-", []string{"f2", "team", "name"})
}

// 024 §1.18：使用獨立原版Minton收據，不能只由相鄰ALLY圖號推出來源。
func TestAlly2JoinedSourceLifecycle(t *testing.T) {
	testAllyJoinedSourceLifecycle(t, 2, "c0743ac3fa9ff26ea58843c2ceb9765803662214cd71ae89d62954f69202b985", "ally2-minton-", []string{"f2-v1-20261005", "teamup-v1-20261005", "name-v1-20261005"})
}

func testAllyJoinedSourceLifecycle(t *testing.T, index int, packedSHA, prefix string, labels []string) {
	t.Helper()
	orig := os.Getenv("PSYCHICWAR_TEST_ORIG")
	frames := os.Getenv("PSYCHICWAR_ALLY1_EVIDENCE")
	if orig == "" || frames == "" {
		t.Skip("需明示原版與正常ALLY #1招募收據")
	}
	dir := t.TempDir()
	themeTestPNG(t, dir, "ally.png", 72, 96)
	entry := ThemeEntry{PBL: "ALLY.PBL", Image: index, At: []int{232, 152}, PNG: "ally.png", Kind: "redraw", Match: []int{248, 0, 72, 40}}
	kai := entry
	kai.Image, kai.At = 0, []int{264, 152}
	other := entry
	other.Image = 3 - index
	load := func(entries []ThemeEntry) (*Theme, error) {
		b, err := json.Marshal(ThemeManifest{Schema: "psychic-war-theme/1", Name: "joined", Scale: 3, Entries: entries})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); err != nil {
			t.Fatal(err)
		}
		hd, _, err := LoadTheme(dir, orig, "", 3)
		return hd, err
	}
	hd, err := load([]ThemeEntry{entry, kai, other})
	if err != nil {
		t.Fatal(err)
	}
	s := hd.groups[0].sprite
	if fmt.Sprintf("%x", sha256.Sum256(s.packed)) != packedSHA {
		t.Fatal("未載入獨立原版收據的ALLY來源", index)
	}
	readFrame := func(label string) []byte {
		labelIndex := map[string]int{"f2": 0, "team": 1, "name": 2}[label]
		p, err := os.ReadFile(filepath.Join(frames, prefix+labels[labelIndex]+"-event-end.frame"))
		if err != nil || len(p) != 64000 {
			t.Fatal("正常原版frame無效", label, err)
		}
		return p
	}
	render := func(frame []byte) []byte {
		hd.frameSprites(frame)
		hd.Layer.Frame(frame, make([]byte, 64000*3))
		p := make([]byte, 960*600*4)
		hd.Draw(p, 3)
		return p
	}
	alpha := func(p []byte, x, y int) byte { return p[4*(y*3*960+x*3)+3] }
	for _, label := range []string{"f2", "team"} {
		p := render(readFrame(label))
		if s.active || alpha(p, 244, 167) != 0 {
			t.Fatal("命名前的空白隊伍欄誤認ALLY #1", label)
		}
	}
	joined := readFrame("name")
	initial := render(joined)
	if !s.active || !hd.groups[1].sprite.active || hd.groups[2].sprite.active || s == hd.groups[1].sprite || alpha(initial, 244, 167) != 128 {
		t.Fatal("正常加入後#1與#0未各自顯示")
	}
	otherName := "recruit-shulosu-name-event-end.frame"
	if index == 1 {
		otherName = "ally2-minton-name-v1-20261005-event-end.frame"
	}
	otherFrame, err := os.ReadFile(filepath.Join(frames, otherName))
	if err != nil || len(otherFrame) != 64000 {
		t.Fatal("同位置另一正常ALLY來源收據無效", err)
	}
	render(otherFrame)
	if s.active || !hd.groups[2].sprite.active || !hd.groups[1].sprite.active {
		t.Fatal("同位置#1／#2切換未清除前來源或誤認新來源")
	}
	if !bytes.Equal(render(joined), initial) || !s.active || hd.groups[2].sprite.active {
		t.Fatal("回到完整原圖後未切回唯一來源")
	}
	// 新角色不能在其他原版區域留下圖素。
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			inside := y >= 152 && y < 184 && (x >= 232 && x < 256 || x >= 264 && x < 288)
			if !inside && alpha(initial, x, y) != 0 {
				t.Fatal("隊伍圖像越過原版位置", x, y)
			}
		}
	}
	covered := append([]byte(nil), joined...)
	covered[169*320+245] ^= 1
	p := render(covered)
	if !s.active || alpha(p, 244, 169) != 0 || alpha(p, 252, 169) != 128 || alpha(p, 276, 169) != 128 {
		t.Fatal("單像素未遮完整8×8格，或干擾相鄰格／#0")
	}
	if !bytes.Equal(render(joined), initial) {
		t.Fatal("恢復原版後未恢復完整HD圖面")
	}
	cleared := append([]byte(nil), joined...)
	for y := 152; y < 184; y++ {
		clear(cleared[y*320+232 : y*320+256])
	}
	p = render(cleared)
	for y := 152; y < 184; y++ {
		for x := 232; x < 256; x++ {
			if alpha(p, x, y) != 0 {
				t.Fatal("清空原圖後留下ALLY #1殘影")
			}
		}
	}
	if !hd.groups[1].sprite.active || alpha(p, 276, 169) != 128 {
		t.Fatal("清空#1干擾#0")
	}
	// 未知XOR不猜身份；入口到返回期間也不能由殘留完整舊圖重啟。
	render(joined)
	hd.blitSprites(oracle.Regs{AX: 1, CX: 0x3A26, DX: 0x0304},
		func() []byte { t.Fatal("未登記ALLY差分仍讀來源"); return nil }, func() []byte { return joined })
	if alpha(render(joined), 244, 167) != 0 || s.active {
		t.Fatal("貼圖進行中沿用舊ALLY #1來源")
	}
	hd.finishBlit()
	if alpha(render(covered), 252, 169) != 0 || s.active {
		t.Fatal("未知XOR後由局部吻合猜出ALLY #1")
	}
	if !bytes.Equal(render(joined), initial) {
		t.Fatal("貼圖完成後完整原圖未重建")
	}
	noAnchor := append([]byte(nil), joined...)
	noAnchor[248] ^= 1
	if p = render(noAnchor); s.active || alpha(p, 244, 167) != 0 {
		t.Fatal("背景錨點失配未撤下新ALLY來源")
	}
	render(joined)
	hd.Enabled = false
	hd.ResetForLoad()
	if s.active || hd.groups[1].sprite.active || hd.Enabled {
		t.Fatal("載回保留角色身份或改HD開關")
	}
	p = render(joined)
	if !bytes.Equal(p, make([]byte, len(p))) {
		t.Fatal("HD停用仍輸出圖素")
	}
	hd.Enabled = true
	if !bytes.Equal(render(joined), initial) {
		t.Fatal("冷載後未由完整原圖重建")
	}
	for _, mutate := range []func(*ThemeEntry){
		func(e *ThemeEntry) { e.Image = 0 },
		func(e *ThemeEntry) { e.Image = 3 },
		func(e *ThemeEntry) { e.Image = 16 },
		func(e *ThemeEntry) { e.At = []int{264, 152} },
		func(e *ThemeEntry) { e.At = []int{136, 8} },
		func(e *ThemeEntry) { e.At = []int{228, 152} },
		func(e *ThemeEntry) { e.Match = nil },
		func(e *ThemeEntry) { e.Match = []int{0, 0, 320, 40} },
		func(e *ThemeEntry) { e.Match = []int{0, 0, 160, 40} },
		func(e *ThemeEntry) { e.Src = []int{0, 0, 16, 16} },
	} {
		bad := entry
		mutate(&bad)
		if _, err := load([]ThemeEntry{bad}); err == nil {
			t.Fatal("未證實來源／位置／錨點未拒絕", bad)
		}
	}
	if index == 2 {
		bad := entry
		bad.At = []int{128, 16}
		if _, err := load([]ThemeEntry{bad}); err == nil {
			t.Fatal("未驗ALLY #2道具位置偏移未拒絕")
		}
	}
	if _, err := load([]ThemeEntry{entry, entry}); err == nil {
		t.Fatal("同位置重複ALLY #1來源未拒絕")
	}
	if _, err := loadAlly(orig, 3); err == nil {
		t.Fatal("來源載入器擴張到未READY圖號")
	}
}
