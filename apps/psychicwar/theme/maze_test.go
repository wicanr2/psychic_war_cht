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

	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/dosgolem/xlate"
)

func mazeRealTheme(t *testing.T) (*Theme, string, string) {
	t.Helper()
	orig, proto := os.Getenv("PSYCHICWAR_TEST_ORIG"), os.Getenv("PSYCHICWAR_TEST_MAZE")
	if orig == "" || proto == "" {
		t.Skip("需明示原版來源及独立迷宮原型")
	}
	base, err := themeBackground(orig)
	if err != nil {
		t.Fatal(err)
	}
	m, err := loadMaze(proto, orig, MazeEntry{File: "MAZE.BIN", Atlas: "MAZE-atlas.png", Columns: 16, Kind: "redraw"}, 3, base)
	if err != nil {
		t.Fatal(err)
	}
	return &Theme{Enabled: true, Layer: xlate.Layer{W: 320, H: 200}, maze: m}, orig, proto
}

func mazeRGB(f []byte) []byte {
	out := make([]byte, len(f)*3)
	for i, c := range f {
		copy(out[i*3:i*3+3], mazeColors[c][:])
	}
	return out
}

func mazeRender(hd *Theme, f, rgb, table []byte) []byte {
	hd.maze.frame(&hd.Layer, f, rgb, 0x0d, hd.maze.source, table)
	hd.Layer.Frame(f, rgb)
	out := make([]byte, 960*600*4)
	for y := 0; y < 600; y++ {
		for x := 0; x < 960; x++ {
			i := y/3*320 + x/3
			copy(out[(y*960+x)*4:(y*960+x)*4+3], rgb[i*3:i*3+3])
			out[(y*960+x)*4+3] = 255
		}
	}
	hd.Draw(out, 3)
	return out
}

// 期望由獨立Python原始資料合成，不從Go迷宮實作反填。
func TestMazeIndependentSceneAndLifecycle(t *testing.T) {
	hd, _, proto := mazeRealTheme(t)
	research := os.Getenv("PSYCHICWAR_TEST_MAZE_RESEARCH")
	if research == "" {
		t.Skip("需正常狀態研究輸入")
	}
	read := func(p string) []byte {
		t.Helper()
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	var saved struct {
		Table []byte `json:"maze_table"`
	}
	if err := json.Unmarshal(read(filepath.Join(research, "pionn-rusteck-exit-v1-maze-read-v1.json")), &saved); err != nil {
		t.Fatal(err)
	}
	f := read(filepath.Join(research, "pionn-rusteck-exit-v1-event-end.frame"))
	rgb := mazeRGB(f)
	img, err := png.Decode(bytes.NewReader(read(filepath.Join(proto, "rusteck-prototype.png"))))
	if err != nil {
		t.Fatal(err)
	}
	want := image.NewNRGBA(image.Rect(0, 0, 960, 600))
	draw.Draw(want, want.Bounds(), img, img.Bounds().Min, draw.Src)
	initial := mazeRender(hd, f, rgb, saved.Table)
	if !bytes.Equal(initial, want.Pix) || !hd.maze.known || len(hd.Layer.Art) != 10 {
		t.Fatal("独立整屏合成不符")
	}
	changed := append([]byte(nil), f...)
	changed[124*320+4] ^= 1
	after := mazeRender(hd, changed, mazeRGB(changed), saved.Table)
	if !hd.maze.known {
		t.Fatal("局部遮擋撤銷整張來源")
	}
	for y := 0; y < 600; y++ {
		for x := 0; x < 960; x++ {
			i := (y*960 + x) * 4
			if x < 24 && y >= 360 && y < 384 {
				c := mazeColors[changed[y/3*320+x/3]]
				if !bytes.Equal(after[i:i+4], []byte{c[0], c[1], c[2], 255}) {
					t.Fatal("失配格未回原版")
				}
			} else if !bytes.Equal(after[i:i+4], initial[i:i+4]) {
				t.Fatal("局部失配改動其他格")
			}
		}
	}
	if !bytes.Equal(mazeRender(hd, f, rgb, saved.Table), initial) {
		t.Fatal("恢復不符")
	}
	hd.ResetForLoad()
	mazeRender(hd, changed, mazeRGB(changed), saved.Table)
	if hd.maze.known || len(hd.Layer.Art) != 0 {
		t.Fatal("冷載局部誤取得來源")
	}
	mazeRender(hd, f, rgb, saved.Table)
	table := append([]byte(nil), saved.Table...)
	table[0] ^= 1
	mazeRender(hd, changed, mazeRGB(changed), table)
	if hd.maze.known || len(hd.Layer.Art) != 0 {
		t.Fatal("改表留下舊圖")
	}
	for _, name := range []string{"palette", "source", "room", "anchor", "blank", "mode"} {
		t.Run(name, func(t *testing.T) {
			mazeRender(hd, f, rgb, saved.Table)
			ff, rr, ss := append([]byte(nil), f...), append([]byte(nil), rgb...), append([]byte(nil), hd.maze.source...)
			mode := byte(0x0d)
			switch name {
			case "palette":
				rr[(124*320+4)*3] ^= 1
			case "source":
				ss[0] ^= 1
			case "anchor":
				ff[248] ^= 1
			case "mode":
				mode = 0x13
			case "room":
				for y := 0; y < 72; y++ {
					copy(ff[(124+y)*320+4:(124+y)*320+76], hd.maze.rooms[1][y*72:(y+1)*72])
				}
				rr = mazeRGB(ff)
			case "blank":
				for y := 0; y < 72; y++ {
					clear(ff[(124+y)*320+4 : (124+y)*320+76])
				}
				rr = mazeRGB(ff)
			}
			hd.maze.frame(&hd.Layer, ff, rr, mode, ss, saved.Table)
			if hd.maze.known || len(hd.Layer.Art) != 0 {
				t.Fatal("失效來源仍有圖面")
			}
		})
	}
	hd.Enabled = false
	mazeRender(hd, f, rgb, saved.Table)
	plane := make([]byte, 960*600*4)
	if hd.Draw(plane, 3) || !bytes.Equal(plane, make([]byte, len(plane))) {
		t.Fatal("HD停用仍畫")
	}
	hd.ResetForLoad()
	if hd.Enabled || hd.maze.known {
		t.Fatal("讀檔改開關或沿用來源")
	}
	hd.Enabled = true
	if !bytes.Equal(mazeRender(hd, f, rgb, saved.Table), initial) {
		t.Fatal("載回重建不符")
	}
}

func TestMazeNormalStatesReadOnly(t *testing.T) {
	hd, orig, _ := mazeRealTheme(t)
	research := os.Getenv("PSYCHICWAR_TEST_MAZE_RESEARCH")
	if research == "" {
		t.Skip("需正常狀態研究輸入")
	}
	b, err := os.ReadFile("../../../workplace/hd/maze-room-overlap-v1-20261003.json")
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Samples []struct {
			State    string `json:"state"`
			Mismatch int    `json:"maze_view_mismatch"`
		} `json:"samples"`
	}
	if err = json.Unmarshal(b, &data); err != nil {
		t.Fatal(err)
	}
	states := []string{}
	eligible := []bool{}
	for _, s := range data.Samples {
		states = append(states, s.State)
		eligible = append(eligible, s.Mismatch == 0)
	}
	for _, s := range []string{"launch", "rusteck", "rusteck-exit"} {
		states = append(states, filepath.Join(research, "pionn-"+s+"-v1-event-observed.state"))
		eligible = append(eligible, s == "rusteck-exit")
	}
	o, err := oracle.Load(filepath.Join(orig, "PW.EXE"), orig)
	if err != nil {
		t.Fatal(err)
	}
	active := 0
	// 選用本機收據供既有獨立44欄位比較器核對埠、時鐘與DOS。
	audit := os.Getenv("PSYCHICWAR_TEST_MAZE_STATE_PREFIX")
	save := func(label string, machine *oracle.Oracle) {
		t.Helper()
		if audit == "" {
			return
		}
		path := audit + "-" + label + ".state"
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("拒絕覆寫狀態收據", path)
		}
		if err := machine.SaveStateFile(path); err != nil {
			t.Fatal(err)
		}
	}
	for i, state := range states {
		if err = o.LoadStateFile(state); err != nil {
			t.Fatal(err)
		}
		before := o.Indexed()
		regs, steps, cycles := o.Regs(), o.Steps(), o.Cycles()
		ram := o.Bytes(oracle.Addr{}, 0xa0000)
		save(fmt.Sprintf("%02d-before", i), o)
		hd.ResetForLoad()
		hd.Frame(o)
		save(fmt.Sprintf("%02d-after", i), o)
		if hd.maze.known != eligible[i] {
			t.Fatal("正常來源認定不符", i)
		}
		if hd.maze.known {
			active++
		}
		if !bytes.Equal(before, o.Indexed()) || regs != o.Regs() || steps != o.Steps() || cycles != o.Cycles() || !bytes.Equal(ram, o.Bytes(oracle.Addr{}, 0xa0000)) {
			t.Fatal("顯示改變原版狀態", i)
		}
	}
	if len(states) != 16 || active != 9 {
		t.Fatal("正常樣本數不符", len(states), active)
	}
	// 已確認原版狀態後再走相同正常指令，顯示觀察不能影響結果。
	control, err := oracle.Load(filepath.Join(orig, "PW.EXE"), orig)
	if err != nil {
		t.Fatal(err)
	}
	if err = control.LoadStateFile(states[len(states)-1]); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 10; n++ {
		hd.Frame(o)
		if err = o.Run(1000); err != nil {
			t.Fatal(err)
		}
		if err = control.Run(1000); err != nil {
			t.Fatal(err)
		}
	}
	if o.Regs() != control.Regs() || o.Steps() != control.Steps() || o.Cycles() != control.Cycles() || !bytes.Equal(o.Indexed(), control.Indexed()) || !bytes.Equal(o.Bytes(oracle.Addr{}, 0xa0000), control.Bytes(oracle.Addr{}, 0xa0000)) {
		t.Fatal("正常推進與控制不同")
	}
	save("continuation-observed", o)
	save("continuation-control", control)
}

func TestMazeManifestCompatibility(t *testing.T) {
	_, orig, proto := mazeRealTheme(t)
	dir := t.TempDir()
	b, err := os.ReadFile(filepath.Join(proto, "theme-atlas-proposal.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m ThemeManifest
	if err = json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, e := range m.Entries {
		b, err := os.ReadFile(filepath.Join(proto, e.PNG))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, e.PNG), b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	b, err = os.ReadFile(filepath.Join(proto, m.Maze.Atlas))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, m.Maze.Atlas), b, 0644); err != nil {
		t.Fatal(err)
	}
	load := func(m ThemeManifest) (*Theme, error) {
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); err != nil {
			t.Fatal(err)
		}
		hd, _, err := LoadTheme(dir, orig, "", 3)
		return hd, err
	}
	hd, err := load(m)
	if err != nil {
		t.Fatal(err)
	}
	if hd.maze == nil || len(hd.groups) != 30 {
		t.Fatal("新版未排除ROOM22高光")
	}
	// 此主題會覆蓋完整原版迷宮，透明圖集不能悄悄留孔。
	mazeTest := *m.Maze
	themeTestPNG(t, dir, "transparent-atlas.png", 192, 192)
	mazeTest.Atlas = "transparent-atlas.png"
	bad := m
	bad.Maze = &mazeTest
	if _, err = load(bad); err == nil {
		t.Fatal("接受透明圖集")
	}
	mazeTest.Atlas = "MAZE-atlas.png"
	mazeTest.File = "OTHER.BIN"
	if _, err = load(bad); err == nil {
		t.Fatal("接受未知迷宮來源")
	}
	m.Schema = "psychic-war-theme/1"
	if _, err = load(m); err == nil {
		t.Fatal("舊版接受maze")
	}
	m.Maze = nil
	hd, err = load(m)
	if err != nil || hd.maze != nil || len(hd.groups) != 31 {
		t.Fatal("舊版相容性不符", err)
	}
	for _, bad := range []string{`null`, `{}`, `{"file":"MAZE.BIN","atlas":"MAZE-atlas.png","columns":16,"kind":"redraw","extra":0}`, `{"file":"MAZE.BIN","atlas":"../MAZE-atlas.png","columns":16,"kind":"redraw"}`, `{"file":"MAZE.BIN","atlas":"MAZE-atlas.png","columns":15,"kind":"redraw"}`} {
		var raw map[string]json.RawMessage
		b, _ := json.Marshal(m)
		json.Unmarshal(b, &raw)
		raw["schema"] = json.RawMessage(`"psychic-war-theme/2"`)
		raw["maze"] = json.RawMessage(bad)
		b, _ = json.Marshal(raw)
		if err = os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); err != nil {
			t.Fatal(err)
		}
		if _, _, err = LoadTheme(dir, orig, "", 3); err == nil {
			t.Fatal("接受不合法maze", bad)
		}
	}
}
