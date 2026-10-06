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

// 024 §1.19：期望格來自獨立Python解析與原版完整返回畫面，不由Layer自身產生。
func TestJaxemoOriginalEventsAndGrid(t *testing.T) {
	orig, evidence := os.Getenv("PSYCHICWAR_TEST_ORIG"), os.Getenv("PSYCHICWAR_JAXEMO_EVIDENCE")
	if orig == "" || evidence == "" {
		t.Skip("需明示原版與研究038 §142的正常Jaxemo收據")
	}
	var grid struct {
		Rows []struct {
			Index int
			Count int `json:"nonempty_valid_cells"`
			Cells []struct {
				X, Y, Mismatch int
				Nonblack       bool
			}
		}
		Hashes map[string]string `json:"inputs_sha256"`
	}
	readJSON := func(name string, dst any) {
		b, err := os.ReadFile(filepath.Join(evidence, name))
		if err != nil || json.Unmarshal(b, dst) != nil {
			t.Fatal("原版收據無效", name, err)
		}
	}
	readJSON("rusteck-sprites-grid-eligibility-v2-20261005.json", &grid)
	var events struct {
		Draws []struct {
			Context      oracle.Regs
			Return       struct{ Step uint64 }
			File         string
			BeforeFile   string
			AfterFile    string
			AfterState   string
			PackedSHA256 string
		} `json:"ally_draws"`
	}
	readJSON("rusteck-sprites-southup-v1-20261005-event.json", &events)
	if len(events.Draws) != 4 || len(grid.Rows) != 3 {
		t.Fatal("限定原版事件／期望格數不符")
	}
	read := func(path string, length int, hash string) []byte {
		b, err := os.ReadFile(filepath.Join(evidence, filepath.Base(path)))
		if err != nil || len(b) != length || hash != "" && fmt.Sprintf("%x", sha256.Sum256(b)) != hash {
			t.Fatal("原版資料長度或SHA不符", path, err)
		}
		return b
	}
	dir := t.TempDir()
	entries := make([]ThemeEntry, 3)
	colors := []color.NRGBA{{R: 201, G: 13, B: 44, A: 255}, {R: 17, G: 202, B: 55, A: 255}, {R: 26, G: 39, B: 203, A: 255}}
	for i := range entries {
		name := fmt.Sprintf("pose%d.png", i+3)
		img := image.NewNRGBA(image.Rect(0, 0, 72, 96))
		for y := 0; y < 96; y++ {
			for x := 0; x < 72; x++ {
				img.SetNRGBA(x, y, colors[i])
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
		entries[i] = ThemeEntry{PBL: "ENEMY04.PBL", Image: i + 3, At: []int{32, 152}, PNG: name, Kind: "redraw", Match: []int{248, 0, 72, 40}}
	}
	load := func(es []ThemeEntry) (*Theme, error) {
		b, err := json.Marshal(ThemeManifest{Schema: "psychic-war-theme/1", Name: "synthetic-source-test", Scale: 3, Entries: es})
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
		saved := append([]byte(nil), frame...)
		hd.frameSprites(frame)
		hd.Layer.Frame(frame, make([]byte, 64000*3))
		out := make([]byte, 960*600*4)
		hd.Draw(out, 3)
		if !bytes.Equal(frame, saved) {
			t.Fatal("覆繪修改原版frame")
		}
		return out
	}
	expected := func(i int, remove bool) []byte {
		out := make([]byte, 960*600*4)
		count := 0
		for _, cell := range grid.Rows[i].Cells {
			if cell.Mismatch != 0 || !cell.Nonblack || remove && cell.X == 32 && cell.Y == 152 {
				continue
			}
			count++
			c := colors[i]
			for y := cell.Y * 3; y < (cell.Y+8)*3; y++ {
				for x := cell.X * 3; x < (cell.X+8)*3; x++ {
					copy(out[4*(y*960+x):], []byte{c.R, c.G, c.B, c.A})
				}
			}
		}
		want := []int{12, 8, 8}[i]
		if remove {
			want--
		}
		if count != want || grid.Rows[i].Index != i+3 || grid.Rows[i].Count != []int{12, 8, 8}[i] {
			t.Fatal("獨立原版格期望不符", i, count)
		}
		return out
	}
	var after []byte
	for i, event := range events.Draws[:3] {
		before := read(event.BeforeFile, 64000, "")
		after = read(event.AfterFile, 64000, grid.Hashes[event.AfterFile])
		if grid.Hashes[event.AfterFile] == "" {
			t.Fatal("格收據缺少實際frame SHA")
		}
		raw := read(event.File, 384, event.PackedSHA256)
		hd.blitSprites(event.Context, func() []byte { return raw }, func() []byte { return before })
		for n, g := range hd.groups {
			if g.sprite.active != (n == i) || !g.sprite.inFlight {
				t.Fatal("實際原版入口未取得唯一姿勢", i, n)
			}
		}
		render(before) // 入口尚未返回，不得由完整舊圖推翻新身份。
		if !hd.groups[i].sprite.active {
			t.Fatal("貼圖進行中丟失已證實來源")
		}
		hd.finishBlit()
		want := expected(i, false)
		got := render(after)
		if !bytes.Equal(got, want) || bytes.Equal(got, expected((i+1)%3, false)) || bytes.Equal(got, make([]byte, len(got))) {
			t.Fatal("完整圖面／錯姿勢／省略來源負對照失效", i)
		}
		covered := append([]byte(nil), after...)
		covered[152*320+32] ^= 1
		if !bytes.Equal(render(covered), expected(i, true)) || !bytes.Equal(render(after), want) {
			t.Fatal("單像素未移除整個8×8格或未恢復", i)
		}
		hd.Enabled = false
		if !bytes.Equal(render(after), make([]byte, len(got))) || !hd.groups[i].sprite.active {
			t.Fatal("關閉HD未隱藏圖面或停止追蹤來源")
		}
		hd.Enabled = true
	}
	// 冷載受遮擋的#5不猜來源，完整#3才能重建。
	hd.ResetForLoad()
	if !bytes.Equal(render(after), make([]byte, 960*600*4)) {
		t.Fatal("冷載局部#5猜出來源")
	}
	first := events.Draws[0]
	full := read(first.AfterFile, 64000, grid.Hashes[first.AfterFile])
	if !bytes.Equal(render(full), expected(0, false)) {
		t.Fatal("冷載完整#3未重建")
	}
	// 使用正常#3返回state接續原版指令，驗正式Attach與8751返回閘門。
	o, err := oracle.Load(filepath.Join(orig, "PW.EXE"), orig)
	if err != nil {
		t.Fatal(err)
	}
	control, err := oracle.Load(filepath.Join(orig, "PW.EXE"), orig)
	if err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(evidence, filepath.Base(first.AfterState))
	if err = o.LoadStateFile(statePath); err != nil {
		t.Fatal(err)
	}
	if err = control.LoadStateFile(statePath); err != nil {
		t.Fatal(err)
	}
	hd.ResetForLoad()
	if err = hd.Attach(o); err != nil {
		t.Fatal(err)
	}
	hd.Frame(o)
	if !hd.groups[0].sprite.active {
		t.Fatal("實際正常state未取得完整#3")
	}
	for i := 1; i < 3; i++ {
		end := events.Draws[i].Return.Step + 1
		if end <= o.Steps() {
			t.Fatal("正常接續步數倒退")
		}
		delta := end - o.Steps()
		if err = o.Run(delta); err != nil {
			t.Fatal(err)
		}
		if err = control.Run(delta); err != nil {
			t.Fatal(err)
		}
		hd.Frame(o)
		out := make([]byte, 960*600*4)
		hd.Draw(out, 3)
		if !bytes.Equal(out, expected(i, false)) || !bytes.Equal(o.Indexed(), control.Indexed()) ||
			o.Regs() != control.Regs() || o.Steps() != control.Steps() || o.Cycles() != control.Cycles() {
			t.Fatal("正式Attach正常指令接續與獨立期望／原版控制不符", i)
		}
		prefix := os.Getenv("PSYCHICWAR_JAXEMO_STATE_PREFIX")
		if prefix != "" {
			for label, machine := range map[string]*oracle.Oracle{"observed": o, "control": control} {
				path := fmt.Sprintf("%s-%d-%s.state", prefix, i+3, label)
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatal("拒絕覆寫接續收據", path)
				}
				if err := machine.SaveStateFile(path); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	// §1.23的反向邊另由原始圖庫模型驗證；以下來源失效仍須拒絕。
	for _, name := range []string{"wrong-source", "unknown-cover", "unknown-mode", "load", "ambiguous", "cross-file", "anchor", "in-flight", "missing-source"} {
		t.Run(name, func(t *testing.T) {
			hd.ResetForLoad()
			from, to := 0, 1
			frame := append([]byte(nil), full...)
			s := hd.groups[from].sprite
			for y := 0; y < 32; y++ {
				copy(frame[(152+y)*320+32:], s.indexed[y*24:(y+1)*24])
			}
			render(frame)
			frame[152*320+32] ^= 1
			r := oracle.Regs{AX: 1, CX: 0x0826, DX: 0x0304}
			raw := make([]byte, 384)
			for i := range raw {
				raw[i] = s.packed[i] ^ hd.groups[to].sprite.packed[i]
			}
			switch name {
			case "wrong-source":
				raw[0] ^= 1
			case "unknown-cover":
				hd.blitSprites(oracle.Regs{CX: 0x0826, DX: 0x0304}, func() []byte { return make([]byte, 384) }, func() []byte { return frame })
				hd.finishBlit()
			case "unknown-mode":
				r.AX = 2
			case "load":
				hd.ResetForLoad()
			case "ambiguous":
				hd.groups[2].sprite.active = true
			case "cross-file":
				s.active = false
				other, err := loadEnemy(orig, "ENEMY00.PBL", 0)
				if err != nil {
					t.Fatal(err)
				}
				other.active = true
				hd.groups = append(hd.groups, themeGroup{sprite: other, watch: hd.groups[0].watch})
				defer func() { hd.groups = hd.groups[:3] }()
			case "anchor":
				frame[248] ^= 1
			case "in-flight":
				s.inFlight = true
			case "missing-source":
				raw = nil
			}
			hd.blitSprites(r, func() []byte { return raw }, func() []byte { return frame })
			for _, g := range hd.groups {
				if g.sprite.active {
					t.Fatal("未知／失效來源仍啟用", name)
				}
			}
		})
	}
	for _, mutate := range []func(*ThemeEntry){
		func(e *ThemeEntry) { e.Image = 15 }, func(e *ThemeEntry) { e.Image = 29 },
		func(e *ThemeEntry) { e.At = []int{36, 152} }, func(e *ThemeEntry) { e.Match = nil },
		func(e *ThemeEntry) { e.Match = []int{0, 0, 320, 40} }, func(e *ThemeEntry) { e.Match = []int{0, 0, 160, 40} },
		func(e *ThemeEntry) { e.Src = []int{0, 0, 16, 16} },
	} {
		bad := entries[0]
		mutate(&bad)
		if _, err := load([]ThemeEntry{bad}); err == nil {
			t.Fatal("未證實manifest未拒絕", bad)
		}
	}
	if _, err := load([]ThemeEntry{entries[0], entries[0]}); err == nil {
		t.Fatal("重複來源未拒絕")
	}
}
