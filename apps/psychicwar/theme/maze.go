package theme

// 遊戲專屬來源與生命週期，規格024 §1.13.3。原版資料只讀。
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

	"github.com/wicanr2/dosgolem/xlate"
)

type MazeEntry struct {
	File    string `json:"file"`
	Atlas   string `json:"atlas"`
	Columns int    `json:"columns"`
	Kind    string `json:"kind"`
}

func (e *MazeEntry) UnmarshalJSON(b []byte) error {
	type entry MazeEntry
	var value entry
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&value); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(b, &fields); err != nil {
		return err
	}
	for _, name := range []string{"file", "atlas", "columns", "kind"} {
		v, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return fmt.Errorf("迷宮資料缺少 %s", name)
		}
	}
	*e = MazeEntry(value)
	return nil
}

// 實際AC／DAC映射色彩，研究038 §140；不是原始DAC的索引順序。
var mazeColors = [16][3]byte{
	{0, 0, 0}, {0, 0, 170}, {170, 85, 0}, {0, 170, 170},
	{170, 0, 0}, {170, 0, 170}, {170, 85, 0}, {170, 170, 170},
	{0, 0, 0}, {85, 85, 255}, {255, 255, 85}, {85, 255, 255},
	{255, 85, 85}, {255, 85, 255}, {255, 255, 85}, {255, 255, 255},
}

type mazeTheme struct {
	source, base, table []byte
	rooms               [][]byte
	atlas               *image.NRGBA
	scale               int
	known               bool
}

func loadMaze(root, orig string, e MazeEntry, scale int, base []byte) (*mazeTheme, error) {
	if e.File != "MAZE.BIN" || e.Columns != 16 || e.Kind != "redraw" {
		return nil, fmt.Errorf("迷宮只接受MAZE.BIN、16欄及redraw")
	}
	source, err := os.ReadFile(filepath.Join(orig, e.File))
	if err != nil {
		return nil, err
	}
	if len(source) != 2048 || fmt.Sprintf("%x", sha256.Sum256(source)) != "8b3b08ef483ff8c64beb04c69c861763c1258afba2348a7058e828bcf4060756" {
		return nil, fmt.Errorf("MAZE.BIN尺寸或SHA-256不符")
	}
	path, err := themeAssetPath(root, e.Atlas)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil || cfg.Width != 64*scale || cfg.Height != 64*scale {
		return nil, fmt.Errorf("迷宮圖集尺寸不符")
	}
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	atlas := image.NewNRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
	draw.Draw(atlas, atlas.Bounds(), img, img.Bounds().Min, draw.Src)
	for i := 3; i < len(atlas.Pix); i += 4 {
		if atlas.Pix[i] != 255 {
			return nil, fmt.Errorf("迷宮圖集須全部不透明")
		}
	}
	m := &mazeTheme{source: source, base: append([]byte(nil), base...), atlas: atlas, scale: scale}
	for _, n := range []int{0, 2, 3, 8} {
		px, err := loadRoomImage(orig, n)
		if err != nil {
			return nil, err
		}
		m.rooms = append(m.rooms, px)
	}
	return m, nil
}

func mazeRowKey(row int) string { return fmt.Sprintf("theme-maze-row-%02d", row) }

func (m *mazeTheme) clear(l *xlate.Layer) {
	for row := 0; row < 10; row++ {
		l.DropArt(mazeRowKey(row))
	}
	m.known, m.table = false, nil
}

func mazeView(frame []byte) []byte {
	view := make([]byte, 72*72)
	for y := 0; y < 72; y++ {
		copy(view[y*72:(y+1)*72], frame[(124+y)*320+4:(124+y)*320+76])
	}
	return view
}

func (m *mazeTheme) original(table []byte) []byte {
	view := make([]byte, 72*72)
	for y := 0; y < 72; y++ {
		for x := 0; x < 72; x++ {
			slot := int(table[y/4*18+x/4])
			v := m.source[slot*8+(y%4)*2+x%4/2]
			if x%2 == 0 {
				v >>= 4
			}
			view[y*72+x] = v & 15
		}
	}
	return view
}

func (m *mazeTheme) frame(l *xlate.Layer, indexed, rgb []byte, mode byte, source, table []byte) {
	valid := mode == 0x0d && len(indexed) == 64000 && len(rgb) == 64000*3 && len(table) == 324 && bytes.Equal(source, m.source)
	if valid {
		for y := 0; y < 40 && valid; y++ {
			valid = bytes.Equal(indexed[y*320+248:(y+1)*320], m.base[y*320+248:(y+1)*320])
		}
	}
	if !valid {
		m.clear(l)
		return
	}
	view := mazeView(indexed)
	for _, room := range m.rooms {
		if bytes.Equal(view, room) {
			m.clear(l)
			return
		}
	}
	nonblack := false
	for y := 0; y < 72; y++ {
		for x := 0; x < 72; x++ {
			i := (124+y)*320 + 4 + x
			c := indexed[i]
			if c >= 16 || !bytes.Equal(rgb[i*3:i*3+3], mazeColors[c][:]) {
				m.clear(l)
				return
			}
			nonblack = nonblack || rgb[i*3] != 0 || rgb[i*3+1] != 0 || rgb[i*3+2] != 0
		}
	}
	if !nonblack {
		m.clear(l)
		return
	}
	if m.known && bytes.Equal(table, m.table) {
		return // 通用204逐格決定局部遮擋與恢復。
	}
	m.clear(l)
	original := m.original(table)
	if !bytes.Equal(view, original) {
		return
	}
	ref := append([]byte(nil), m.base...)
	for y := 0; y < 72; y++ {
		copy(ref[(124+y)*320+4:(124+y)*320+76], original[y*72:(y+1)*72])
	}
	pix := make([]byte, 80*80*m.scale*m.scale*4)
	for y := 0; y < 72*m.scale; y++ {
		for x := 0; x < 72*m.scale; x++ {
			slot := int(table[y/(4*m.scale)*18+x/(4*m.scale)])
			ax := slot%16*4*m.scale + x%(4*m.scale)
			ay := slot/16*4*m.scale + y%(4*m.scale)
			i := (ay*m.atlas.Stride + ax*4)
			j := ((y+4*m.scale)*80*m.scale + x + 4*m.scale) * 4
			copy(pix[j:j+4], m.atlas.Pix[i:i+4])
		}
	}
	for row := 0; row < 10; row++ {
		r := make([]byte, 80*8)
		for y := 0; y < 8; y++ {
			copy(r[y*80:(y+1)*80], ref[(120+row*8+y)*320:(120+row*8+y)*320+80])
		}
		s := &xlate.Stamp{Key: mazeRowKey(row), Art: true, Order: -1, X: 0, Y: 120 + row*8,
			Cells: 10, CellW: 8, CellH: 8, PixScale: m.scale, Reference: r,
			Pix: pix[row*8*m.scale*80*m.scale*4 : (row+1)*8*m.scale*80*m.scale*4]}
		if err := l.AddArt(s); err != nil {
			m.clear(l)
			return
		}
	}
	m.table, m.known = append([]byte(nil), table...), true
}
