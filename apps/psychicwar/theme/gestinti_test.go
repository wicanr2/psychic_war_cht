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

// 024 §1.20：十一正常原版事件的逐格期望由獨立Python收據提供。
func TestGestintiOriginalLoopAndAttached(t *testing.T) {
	orig, evidence := os.Getenv("PSYCHICWAR_TEST_ORIG"), os.Getenv("PSYCHICWAR_GESTINTI_EVIDENCE")
	if orig == "" || evidence == "" {
		t.Skip("需原版及研究038 §143的Gestinti來源收據")
	}
	readJSON := func(name string, value any) {
		b, err := os.ReadFile(filepath.Join(evidence, name))
		if err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(b, value); err != nil {
			t.Fatal(err)
		}
	}
	var proof struct {
		Rows []struct {
			EventIndex int `json:"event_index"`
			Index      int
			Count      int `json:"nonempty_valid_cells"`
			Cells      []struct {
				X, Y, Mismatch int
				Nonblack       bool
			}
		}
		Hashes map[string]string `json:"inputs_sha256"`
	}
	var events struct {
		Draws []struct {
			Context                                               oracle.Regs
			Return                                                struct{ Step uint64 }
			File, BeforeFile, AfterFile, AfterState, PackedSHA256 string
		} `json:"ally_draws"`
	}
	readJSON("jaxemo-attack-gestinti-body-source-v1-20261005.json", &proof)
	readJSON("jaxemo-attack-party-encounter-v1-20261005-event.json", &events)
	if len(proof.Rows) != 11 || len(events.Draws) != 13 {
		t.Fatal("正常原版事件數不符")
	}
	read := func(path string, length int) []byte {
		p, err := os.ReadFile(filepath.Join(evidence, filepath.Base(path)))
		if err != nil || len(p) != length || proof.Hashes[path] == "" || fmt.Sprintf("%x", sha256.Sum256(p)) != proof.Hashes[path] {
			t.Fatal("原版資料長度或SHA不符", path, err)
		}
		return p
	}
	dir := t.TempDir()
	colors := []color.NRGBA{{R: 211, G: 12, B: 43, A: 255}, {R: 22, G: 213, B: 54, A: 255}, {R: 33, G: 24, B: 215, A: 255}}
	entries := []ThemeEntry{}
	for i, c := range colors {
		name := fmt.Sprintf("pose%d.png", i+6)
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
		entries = append(entries, ThemeEntry{PBL: "ENEMY04.PBL", Image: i + 6, At: []int{32, 152}, PNG: name, Kind: "redraw", Match: []int{248, 0, 72, 40}})
	}
	// 同檔Jaxemo另組也登記，避免共用格或來源身份跨組污染。
	other := entries[0]
	other.Image = 3
	entries = append(entries, other)
	load := func(es []ThemeEntry) (*Theme, error) {
		b, err := json.Marshal(ThemeManifest{Schema: "psychic-war-theme/1", Name: "gestinti-source-test", Scale: 3, Entries: es})
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
	render := func(frame []byte) []byte {
		copyFrame := append([]byte(nil), frame...)
		hd.frameSprites(frame)
		hd.Layer.Frame(frame, make([]byte, 64000*3))
		p := make([]byte, 960*600*4)
		hd.Draw(p, 3)
		if !bytes.Equal(copyFrame, frame) {
			t.Fatal("HD修改原版frame")
		}
		return p
	}
	cut := func(row int) (int, int) {
		for _, cell := range proof.Rows[row].Cells {
			if cell.Mismatch == 0 && cell.Nonblack {
				return cell.X, cell.Y
			}
		}
		t.Fatal("來源沒有可見格")
		return 0, 0
	}
	want := func(row int, remove bool) []byte {
		p := make([]byte, 960*600*4)
		count := 0
		c := colors[proof.Rows[row].Index-6]
		cx, cy := cut(row)
		for _, cell := range proof.Rows[row].Cells {
			if cell.Mismatch != 0 || !cell.Nonblack || remove && cell.X == cx && cell.Y == cy {
				continue
			}
			count++
			for y := cell.Y * 3; y < (cell.Y+8)*3; y++ {
				for x := cell.X * 3; x < (cell.X+8)*3; x++ {
					copy(p[4*(y*960+x):], []byte{c.R, c.G, c.B, c.A})
				}
			}
		}
		wantCount := proof.Rows[row].Count
		if remove {
			wantCount--
		}
		if count != wantCount {
			t.Fatal("期望格數不符")
		}
		return p
	}
	var last []byte
	for i, row := range proof.Rows {
		event := events.Draws[row.EventIndex]
		before := read(event.BeforeFile, 64000)
		last = read(event.AfterFile, 64000)
		raw := read(event.File, 384)
		hd.blitSprites(event.Context, func() []byte { return raw }, func() []byte { return before })
		render(before)
		for n, g := range hd.groups {
			if g.sprite.active != (n == row.Index-6) || !g.sprite.inFlight {
				t.Fatal("來源／進行中／跨組身份不符", i, n)
			}
		}
		hd.finishBlit()
		got := render(last)
		expected := want(i, false)
		if !bytes.Equal(got, expected) || bytes.Equal(got, make([]byte, len(got))) {
			t.Fatal("十一完整HD圖面或省略來源負對照不符", i)
		}
		wrong := append([]byte(nil), expected...)
		for p := 0; p < len(wrong); p += 4 {
			if wrong[p+3] != 0 {
				wrong[p] ^= 1
			}
		}
		if bytes.Equal(got, wrong) {
			t.Fatal("錯姿勢色彩負對照無效")
		}
		covered := append([]byte(nil), last...)
		x, y := cut(i)
		covered[y*320+x] ^= 1
		if !bytes.Equal(render(covered), want(i, true)) || !bytes.Equal(render(last), expected) {
			t.Fatal("整個8×8格移除／恢復不符", i)
		}
	}
	hd.ResetForLoad()
	if !bytes.Equal(render(last), make([]byte, 960*600*4)) {
		t.Fatal("冷載部分#8猜身份")
	}
	first := events.Draws[0]
	full := read(first.AfterFile, 64000)
	if !bytes.Equal(render(full), want(0, false)) {
		t.Fatal("冷載完整#6未重建")
	}
	hd.Enabled = false
	if !bytes.Equal(render(full), make([]byte, 960*600*4)) {
		t.Fatal("HD關閉仍畫圖")
	}
	hd.Enabled = true
	// 真實正常state接續全部十次差分，正式Attach與原版無主題控制同指令推進。
	o, err := oracle.Load(filepath.Join(orig, "PW.EXE"), orig)
	if err != nil {
		t.Fatal(err)
	}
	control, err := oracle.Load(filepath.Join(orig, "PW.EXE"), orig)
	if err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(evidence, filepath.Base(first.AfterState))
	if err = o.LoadStateFile(state); err != nil {
		t.Fatal(err)
	}
	if err = control.LoadStateFile(state); err != nil {
		t.Fatal(err)
	}
	hd.ResetForLoad()
	if err = hd.Attach(o); err != nil {
		t.Fatal(err)
	}
	hd.Frame(o)
	for i, row := range proof.Rows[1:] {
		end := events.Draws[row.EventIndex].Return.Step + 1
		if end <= o.Steps() {
			t.Fatal("原版步數倒退")
		}
		steps := end - o.Steps()
		if err = o.Run(steps); err != nil {
			t.Fatal(err)
		}
		if err = control.Run(steps); err != nil {
			t.Fatal(err)
		}
		hd.Frame(o)
		p := make([]byte, 960*600*4)
		hd.Draw(p, 3)
		if !bytes.Equal(p, want(i+1, false)) || !bytes.Equal(o.Indexed(), control.Indexed()) || o.Regs() != control.Regs() || o.Steps() != control.Steps() || o.Cycles() != control.Cycles() {
			t.Fatal("正常Attach接續圖面或控制不符", i)
		}
	}
	if prefix := os.Getenv("PSYCHICWAR_GESTINTI_STATE_PREFIX"); prefix != "" {
		for label, machine := range map[string]*oracle.Oracle{"observed": o, "control": control} {
			path := prefix + "-" + label + ".state"
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("拒絕覆寫接續收據")
			}
			if err := machine.SaveStateFile(path); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, name := range []string{"cross-group", "wrong-source", "anchor", "load", "unknown-mode", "ambiguous"} {
		t.Run(name, func(t *testing.T) {
			hd.ResetForLoad()
			frame := append([]byte(nil), full...)
			render(frame)
			frame[152*320+32] ^= 1
			r := oracle.Regs{AX: 1, CX: 0x0826, DX: 0x0304}
			raw := read(events.Draws[1].File, 384)
			switch name {
			case "cross-group":
				hd.groups[0].sprite.active = false
				hd.groups[3].sprite.active = true
			case "wrong-source":
				raw[0] ^= 1
			case "anchor":
				frame[248] ^= 1
			case "load":
				hd.ResetForLoad()
			case "unknown-mode":
				r.AX = 2
			case "ambiguous":
				hd.groups[2].sprite.active = true
			}
			hd.blitSprites(r, func() []byte { return raw }, func() []byte { return frame })
			for _, g := range hd.groups {
				if g.sprite.active {
					t.Fatal("未知／跨組來源啟用", name)
				}
			}
		})
	}
	for _, n := range []int{-1, 15, 29, 30} {
		if _, err := loadEnemy(orig, "ENEMY04.PBL", n); err == nil {
			t.Fatal("未READY圖號未拒絕", n)
		}
	}
	for _, mutate := range []func(*ThemeEntry){func(e *ThemeEntry) { e.At = []int{36, 152} }, func(e *ThemeEntry) { e.Match = nil }, func(e *ThemeEntry) { e.Src = []int{0, 0, 16, 16} }} {
		e := entries[0]
		mutate(&e)
		if _, err := load([]ThemeEntry{e}); err == nil {
			t.Fatal("未證實位置／錨點／裁切未拒絕")
		}
	}
}
