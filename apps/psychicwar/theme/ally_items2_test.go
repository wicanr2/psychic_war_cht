package theme

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/dosgolem/oracle"
)

// 024 §1.21：獨立Python逐格期望、正常Enter及真正保存／載回接續。
func TestAlly2ItemsOriginalAndAttached(t *testing.T) {
	orig, evidence := os.Getenv("PSYCHICWAR_TEST_ORIG"), os.Getenv("PSYCHICWAR_ALLY2_ITEMS_EVIDENCE")
	if orig == "" || evidence == "" {
		t.Skip("需明示原版及研究038 §144道具來源")
	}
	type cell struct{ Image, X, Y int }
	var proof struct {
		Rows map[string]struct {
			Frame, SHA256 string
			Cells         []cell
		}
	}
	data, err := os.ReadFile(filepath.Join(evidence, "ally2-items-grid-proof-v1-20261005.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &proof); err != nil {
		t.Fatal(err)
	}
	read := func(name string) []byte {
		p, err := os.ReadFile(filepath.Join(evidence, name))
		if err != nil || len(p) != 64000 {
			t.Fatal(name, err)
		}
		return p
	}
	frames := map[string][]byte{}
	for name, row := range proof.Rows {
		p := read(filepath.Base(row.Frame))
		if fmt.Sprintf("%x", sha256.Sum256(p)) != row.SHA256 {
			t.Fatal("原版逐格收據SHA不符", name)
		}
		frames[name] = p
	}
	dir := t.TempDir()
	colors := []color.NRGBA{{R: 211, G: 12, B: 43, A: 255}, {R: 22, G: 213, B: 54, A: 255}, {R: 33, G: 24, B: 215, A: 255}}
	entries := []ThemeEntry{}
	for n, c := range colors {
		name := fmt.Sprintf("ally%d.png", n)
		img := image.NewNRGBA(image.Rect(0, 0, 72, 96))
		for y := 0; y < 96; y++ {
			for x := 0; x < 72; x++ {
				img.SetNRGBA(x, y, c)
			}
		}
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(f, img)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
		entries = append(entries, ThemeEntry{PBL: "ALLY.PBL", Image: n, At: []int{128, 8}, PNG: name, Kind: "redraw", Match: []int{248, 0, 72, 40}})
	}
	bottomKai, bottomMinton := entries[0], entries[2]
	bottomKai.At, bottomMinton.At = []int{264, 152}, []int{232, 152}
	entries = append(entries, bottomKai, bottomMinton)
	load := func(es []ThemeEntry) (*Theme, error) {
		b, err := json.Marshal(ThemeManifest{Schema: "psychic-war-theme/1", Name: "ally2-items-source-test", Scale: 3, Entries: es})
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); err != nil {
			t.Fatal(err)
		}
		hd, _, err := LoadTheme(dir, orig, "", 3)
		return hd, err
	}
	hd, err := load(entries)
	if err != nil {
		t.Fatal(err)
	}
	want := func(name string, removeX, removeY int) []byte {
		p := make([]byte, 960*600*4)
		for _, c := range proof.Rows[name].Cells {
			if c.X == removeX && c.Y == removeY {
				continue
			}
			ink := colors[c.Image]
			for y := c.Y * 3; y < (c.Y+8)*3; y++ {
				for x := c.X * 3; x < (c.X+8)*3; x++ {
					copy(p[4*(y*960+x):], []byte{ink.R, ink.G, ink.B, ink.A})
				}
			}
		}
		return p
	}
	render := func(frame []byte) []byte {
		copyFrame := append([]byte(nil), frame...)
		hd.frameSprites(frame)
		hd.Layer.Frame(frame, make([]byte, 64000*3))
		p := make([]byte, 960*600*4)
		hd.Draw(p, 3)
		if !bytes.Equal(frame, copyFrame) {
			t.Fatal("修改原版frame")
		}
		return p
	}
	for _, name := range []string{"kai", "minton", "list"} {
		if !bytes.Equal(render(frames[name]), want(name, -1, -1)) {
			t.Fatal("完整圖面不符", name)
		}
		if hd.groups[1].sprite.active {
			t.Fatal("錯認同位置ALLY #1", name)
		}
	}
	before := read("ally2-items-next-v1-20261005-event-draw000-before.frame")
	raw, err := os.ReadFile(filepath.Join(evidence, "ally2-items-next-v1-20261005-event-source000.bin"))
	if err != nil || len(raw) != 384 || fmt.Sprintf("%x", sha256.Sum256(raw)) != "c0743ac3fa9ff26ea58843c2ceb9765803662214cd71ae89d62954f69202b985" {
		t.Fatal("實際來源SHA不符", err)
	}
	render(frames["kai"])
	hd.blitSprites(oracle.Regs{AX: 0x2000, CX: 0x2002, DX: 0x0304}, func() []byte { return raw }, func() []byte { return before })
	if !bytes.Equal(render(before), want("list", -1, -1)) || hd.groups[0].sprite.active || !hd.groups[2].sprite.active {
		t.Fatal("AL00進行中未隱藏上方／保留下方來源")
	}
	hd.finishBlit()
	full := render(frames["minton"])
	expected := want("minton", -1, -1)
	if !bytes.Equal(full, expected) || bytes.Equal(full, want("list", -1, -1)) {
		t.Fatal("完整來源／省略上方負對照不符")
	}
	wrong := append([]byte(nil), expected...)
	for y := 8 * 3; y < 40*3; y++ {
		for x := 128 * 3; x < 152*3; x++ {
			wrong[4*(y*960+x)] ^= 1
		}
	}
	if bytes.Equal(full, wrong) {
		t.Fatal("錯圖負對照無效")
	}
	covered := append([]byte(nil), frames["minton"]...)
	covered[8*320+128] ^= 1
	if !bytes.Equal(render(covered), want("minton", 128, 8)) || !bytes.Equal(render(frames["minton"]), expected) {
		t.Fatal("8×8遮格／恢復不符")
	}
	hd.ResetForLoad()
	if !bytes.Equal(render(covered), want("list", -1, -1)) {
		t.Fatal("冷載部分肖像猜身份")
	}
	if !bytes.Equal(render(frames["minton"]), expected) {
		t.Fatal("冷載完整肖像未恢復")
	}
	hd.Enabled = false
	if !bytes.Equal(render(frames["minton"]), make([]byte, len(expected))) {
		t.Fatal("HD關閉仍輸出")
	}
	hd.Enabled = true
	noAnchor := append([]byte(nil), frames["minton"]...)
	noAnchor[248] ^= 1
	if !bytes.Equal(render(noAnchor), make([]byte, len(expected))) {
		t.Fatal("錨點失配仍輸出角色")
	}
	if !bytes.Equal(render(frames["list"]), want("list", -1, -1)) {
		t.Fatal("原版撤圖留下上方殘片")
	}

	o, err := oracle.Load(filepath.Join(orig, "PW.EXE"), orig)
	if err != nil {
		t.Fatal(err)
	}
	defer o.Close()
	control, err := oracle.Load(filepath.Join(orig, "PW.EXE"), orig)
	if err != nil {
		t.Fatal(err)
	}
	defer control.Close()
	state := filepath.Join(evidence, "ally2-see-items-menu-v1-20261005-event-observed.state")
	for _, machine := range []*oracle.Oracle{o, control} {
		if err = machine.LoadStateFile(state); err != nil {
			t.Fatal(err)
		}
	}
	hd.ResetForLoad()
	if err = hd.Attach(o); err != nil {
		t.Fatal(err)
	}
	seen := 0
	o.OnCall(oracle.Addr{Seg: 0x0161, Off: 0x8705}, func(machine *oracle.Oracle) {
		r := machine.Regs()
		if r.CX == 0x2002 && r.DX == 0x0304 && r.AX&255 == 0 {
			if !bytes.Equal(machine.Bytes(oracle.Addr{Seg: r.DS, Off: r.BX}, 384), raw) {
				t.Fatal("實際Enter來源與原版收據不同")
			}
			seen++
		}
	})
	for _, machine := range []*oracle.Oracle{o, control} {
		if err = machine.Run(500000); err != nil {
			t.Fatal(err)
		}
		machine.KeyDown(0x1c)
		if err = machine.Run(150000); err != nil {
			t.Fatal(err)
		}
		machine.KeyUp(0x1c)
		if err = machine.Run(1500000); err != nil {
			t.Fatal(err)
		}
	}
	check := func() {
		hd.Frame(o)
		p := make([]byte, len(expected))
		hd.Draw(p, 3)
		if !bytes.Equal(p, expected) || !bytes.Equal(o.Indexed(), control.Indexed()) || o.Regs() != control.Regs() || o.Steps() != control.Steps() || o.Cycles() != control.Cycles() {
			if prefix := os.Getenv("PSYCHICWAR_ALLY2_ITEMS_STATE_PREFIX"); prefix != "" {
				for name, machine := range map[string]*oracle.Oracle{"observed": o, "control": control} {
					path := prefix + "-failed-" + name + ".state"
					if _, err := os.Stat(path); !os.IsNotExist(err) {
						t.Fatal("拒絕覆寫失敗收據")
					}
					if err := machine.SaveStateFile(path); err != nil {
						t.Fatal(err)
					}
				}
			}
			t.Fatalf("正常Enter不符：圖面%v indexed%v regs%v steps%v cycles%v 實際貼圖%d",
				bytes.Equal(p, expected), bytes.Equal(o.Indexed(), control.Indexed()), o.Regs() == control.Regs(), o.Steps() == control.Steps(), o.Cycles() == control.Cycles(), seen)
		}
	}
	check()
	if seen != 1 {
		t.Fatal("實際正常貼圖次數不是1", seen)
	}
	prefix := os.Getenv("PSYCHICWAR_ALLY2_ITEMS_STATE_PREFIX")
	savePair := func(label string) {
		if prefix == "" {
			return
		}
		for name, machine := range map[string]*oracle.Oracle{"observed": o, "control": control} {
			q := prefix + "-" + label + "-" + name + ".state"
			if _, err := os.Stat(q); !os.IsNotExist(err) {
				t.Fatal("拒絕覆寫state收據")
			}
			if err := machine.SaveStateFile(q); err != nil {
				t.Fatal(err)
			}
		}
	}
	savePair("enter")
	restore := filepath.Join(dir, "actual.state")
	if err = o.SaveStateFile(restore); err != nil {
		t.Fatal(err)
	}
	if err = o.LoadStateFile(restore); err != nil {
		t.Fatal(err)
	}
	hd.ResetForLoad()
	if err = control.LoadStateFile(restore); err != nil {
		t.Fatal(err)
	}
	for _, machine := range []*oracle.Oracle{o, control} {
		if err = machine.Run(100000); err != nil {
			t.Fatal(err)
		}
	}
	check()
	savePair("reload")
	for _, mutate := range []func(*ThemeEntry){
		func(e *ThemeEntry) { e.At = []int{128, 16} },
		func(e *ThemeEntry) { e.Image = 3 },
		func(e *ThemeEntry) { e.Match = nil },
		func(e *ThemeEntry) { e.Match = []int{0, 0, 160, 40} },
		func(e *ThemeEntry) { e.Src = []int{0, 0, 16, 16} },
	} {
		bad := entries[2]
		mutate(&bad)
		if _, err := load([]ThemeEntry{bad}); err == nil {
			t.Fatal("未證實位置／來源／錨點／裁切未拒絕", bad)
		}
	}
	if _, err := load([]ThemeEntry{entries[2], entries[2]}); err == nil {
		t.Fatal("重複來源位置未拒絕")
	}
}
