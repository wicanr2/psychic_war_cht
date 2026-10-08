package theme

import (
	"bytes"
	"fmt"
	"image"
	"math"

	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
)

// 024 §1.58：原版仍執行XOR；此模型只保存派生顯示資料。
type sceneEffect struct {
	entry           ThemeEntry
	x, y, w, h      int
	indexed, packed []byte
	art             *image.NRGBA
}
type sceneEffects struct {
	sources               []sceneEffect
	expected, hd, restore []byte
	valid                 [1000]bool
	active                *sceneEffect
	palette               [256][3]uint8
}

func sceneEffectEntry(e ThemeEntry) bool {
	return e.PBL == "OPEN.PBL" && e.Image == 4 || e.PBL == "END1.PBL" && e.Image == 18
}

var openingCursorPositions = [16][2]int{{136, 152}, {132, 152}, {128, 152}, {124, 148}, {120, 148}, {116, 148}, {112, 144}, {108, 144}, {104, 144}, {100, 148}, {96, 148}, {92, 148}, {88, 152}, {84, 152}, {80, 152}, {76, 152}}

func sceneEffectPosition(e ThemeEntry) (x, y, w, h int, err error) {
	err = fmt.Errorf("場景XOR來源或原位不符")
	if len(e.At) != 2 || e.Kind != "redraw" || e.Scaler != "" {
		return
	}
	x, y = e.At[0], e.At[1]
	if e.PBL == "OPEN.PBL" && e.Image == 4 {
		w, h = 16, 16
		found := false
		for _, p := range openingCursorPositions {
			if x == p[0] && y == p[1] {
				found = true
			}
		}
		if !found {
			return
		}
	} else if e.PBL == "END1.PBL" && e.Image == 18 {
		w, h = 16, 8
		if x < 4 || x > 304 || x%4 != 0 || y != 155 {
			return
		}
	} else {
		return
	}
	if e.Src != nil && !equalInts(e.Src, []int{0, 0, w, h}) || e.Match != nil && !equalInts(e.Match, []int{x, y, w, h}) {
		return
	}
	err = nil
	return
}
func loadSceneEffects(root, orig string, m ThemeManifest) (*sceneEffects, error) {
	b := &sceneEffects{}
	cache := map[string]*image.NRGBA{}
	seen := map[string]bool{}
	for _, e := range m.Entries {
		if !sceneEffectEntry(e) {
			continue
		}
		if m.Schema != "psychic-war-theme/2" {
			return nil, fmt.Errorf("場景XOR只支援theme/2")
		}
		x, y, w, h, err := sceneEffectPosition(e)
		if err != nil {
			return nil, err
		}
		key := fmt.Sprintf("%s:%d:%d:%d", e.PBL, e.Image, x, y)
		if seen[key] {
			return nil, fmt.Errorf("場景XOR原位重複")
		}
		seen[key] = true
		px, err := sceneEffectSource(orig, e)
		if err != nil {
			return nil, err
		}
		art, err := battlePNG(root, e.PNG, w, h, 3, cache)
		if err != nil {
			return nil, err
		}
		s := sceneEffect{entry: e, x: x, y: y, w: w, h: h, indexed: px, packed: make([]byte, len(px)/2), art: art}
		for i := range s.packed {
			s.packed[i] = px[i*2]<<4 | px[i*2+1]
		}
		b.sources = append(b.sources, s)
	}
	if len(b.sources) == 0 {
		return nil, nil
	}
	return b, nil
}
func sceneEffectSource(orig string, e ThemeEntry) ([]byte, error) {
	raw, err := sceneArchive(orig, e.PBL)
	if err != nil {
		return nil, err
	}
	w, h, px, err := pbl.Decode(raw, e.Image)
	if err != nil {
		return nil, err
	}
	if e.PBL == "OPEN.PBL" && (w != 16 || h != 16) || e.PBL == "END1.PBL" && (w != 8 || h != 16) {
		return nil, fmt.Errorf("場景XOR archive形狀不符")
	}
	return px, nil
}
func (b *sceneEffects) reset() {
	if b == nil {
		return
	}
	b.expected = nil
	b.hd = nil
	b.restore = nil
	b.active = nil
	b.valid = [1000]bool{}
}
func nativeRGBA(indexed []byte, palette [256][3]uint8) []byte {
	out := make([]byte, 960*600*4)
	if len(indexed) != 64000 {
		return out
	}
	for y := 0; y < 600; y++ {
		for x := 0; x < 960; x++ {
			i := (y*960 + x) * 4
			c := palette[indexed[(y/3)*320+x/3]]
			copy(out[i:i+3], c[:])
			out[i+3] = 255
		}
	}
	return out
}
func (b *sceneEffects) apply(s *sceneEffect, packed, before, base []byte, palette [256][3]uint8) bool {
	if len(before) != 64000 || len(base) != 960*600*4 || !bytes.Equal(packed, s.packed) {
		b.reset()
		return false
	}
	expected := append([]byte(nil), before...)
	for y := 0; y < s.h; y++ {
		for x := 0; x < s.w; x++ {
			expected[(s.y+y)*320+s.x+x] ^= s.indexed[y*s.w+x]
		}
	}
	var priorRegion []byte
	if len(b.expected) == 64000 {
		priorRegion, _ = pbl.Region(b.expected, 320, 200, s.x, s.y, s.w, s.h)
	}
	currentRegion, _ := pbl.Region(before, 320, 200, s.x, s.y, s.w, s.h)
	erase := b.active != nil && b.active.entry.PBL == s.entry.PBL && b.active.x == s.x && b.active.y == s.y && bytes.Equal(currentRegion, priorRegion) && b.palette == palette
	if erase {
		base = append([]byte(nil), base...)
		nativeAfter := nativeRGBA(expected, palette)
		for cy := s.y / 8; cy <= (s.y+s.h-1)/8; cy++ {
			for cx := s.x / 8; cx <= (s.x+s.w-1)/8; cx++ {
				from := nativeAfter
				if b.valid[cy*40+cx] {
					from = b.restore
				}
				for yy := cy * 24; yy < cy*24+24; yy++ {
					i := (yy*960 + cx*24) * 4
					copy(base[i:i+24*4], from[i:i+24*4])
				}
			}
		}
		b.active = nil
		b.restore = nil
	} else {
		b.restore = append([]byte(nil), base...)
		b.active = s
		base = append([]byte(nil), base...)
	}
	native := nativeRGBA(before, palette)
	if erase {
		native = nativeRGBA(expected, palette)
	}
	b.valid = [1000]bool{}
	for cy := 0; cy < 25; cy++ {
		for cx := 0; cx < 40; cx++ {
			cell := cy*40 + cx
			own := false
			ink := false
			for y := cy * 8; y < cy*8+8; y++ {
				for x := cx * 8; x < cx*8+8; x++ {
					if x >= s.x && x < s.x+s.w && y >= s.y && y < s.y+s.h && s.indexed[(y-s.y)*s.w+x-s.x] != 0 {
						ink = true
					}
					i := (y*3*960 + x*3) * 4
					if !bytes.Equal(base[i:i+4], native[i:i+4]) {
						own = true
					}
				}
			}
			if !erase && ink {
				own = true
				for y := cy * 24; y < cy*24+24; y++ {
					for x := cx * 24; x < cx*24+24; x++ {
						xx, yy := x-s.x*3, y-s.y*3
						if xx < 0 || yy < 0 || xx >= s.w*3 || yy >= s.h*3 {
							continue
						}
						i := (y*960 + x) * 4
						j := yy*s.art.Stride + xx*4
						for c := 0; c < 3; c++ {
							v := float64(s.art.Pix[j+c]) * float64(s.art.Pix[j+3]) / (255 * 255)
							base[i+c] = byte(math.Round(255 - (255-float64(base[i+c]))*(1-v)))
						}
						base[i+3] = 255
					}
				}
			}
			b.valid[cell] = own
		}
	}
	b.expected = expected
	b.hd = base
	b.palette = palette
	return true
}
func (b *sceneEffects) frame(indexed []byte, palette [256][3]uint8) {
	if b == nil || b.expected == nil {
		return
	}
	if len(indexed) != 64000 || palette != b.palette {
		b.reset()
		return
	}
	for cell, valid := range b.valid {
		if !valid {
			continue
		}
		x, y := (cell%40)*8, (cell/40)*8
		for yy := y; yy < y+8; yy++ {
			if !bytes.Equal(indexed[yy*320+x:yy*320+x+8], b.expected[yy*320+x:yy*320+x+8]) {
				b.valid[cell] = false
				break
			}
		}
	}
}
func (b *sceneEffects) draw(dst []byte, scale int) bool {
	if b == nil || scale != 3 || len(dst) != 960*600*4 || b.hd == nil {
		return false
	}
	drawn := false
	for cell, valid := range b.valid {
		if !valid {
			continue
		}
		x, y := (cell%40)*24, (cell/40)*24
		for yy := y; yy < y+24; yy++ {
			i := (yy*960 + x) * 4
			copy(dst[i:i+24*4], b.hd[i:i+24*4])
		}
		drawn = true
	}
	return drawn
}
func (b *sceneEffects) observe(t *Theme, o *oracle.Oracle, end bool) {
	r := o.Regs()
	x, y, w, h := int(r.CX>>8)*4, int(r.CX&255)*4, int(r.DX>>8)*8, int(r.DX&255)*8
	var addr oracle.Addr
	if end {
		ch := int(r.CX >> 8)
		x, y, w, h = 4+4*(ch/2-17), 155, 16, 8
		if r.AX&255 != 11 || ch%2 != 0 || r.DS != 0x1175 {
			b.reset()
			return
		}
		ptr := o.Bytes(oracle.Addr{Seg: r.DS, Off: 0xaec6}, 2)
		if len(ptr) != 2 {
			b.reset()
			return
		}
		off := int(ptr[0]) | int(ptr[1])<<8
		off += 0x3a80
		if off > 65536-64 {
			b.reset()
			return
		}
		addr = oracle.Addr{Seg: r.DS, Off: uint16(off)}
	} else {
		if r.AX&255 != 1 {
			b.reset()
			return
		}
		addr = oracle.Addr{Seg: r.DS, Off: r.BX}
	}
	for i := range b.sources {
		s := &b.sources[i]
		if s.x != x || s.y != y || s.w != w || s.h != h || (s.entry.PBL == "END1.PBL") != end {
			continue
		}
		if addr.Linear() > uint32(0xa0000-len(s.packed)) {
			b.reset()
			return
		}
		packed := o.Bytes(addr, len(s.packed))
		if !bytes.Equal(packed, s.packed) {
			b.reset()
			return
		}
		before := o.Indexed()
		if b.expected == nil && !sceneEffectStartsVisible(t, s, before) {
			b.reset()
			return
		}
		palette := o.Palette()
		t.Frame(o)
		base := nativeRGBA(before, palette)
		t.Layer.Draw(base, 3, nil)
		if t.battle != nil {
			t.battle.draw(base, 3)
		}
		b.draw(base, 3)
		b.apply(s, packed, before, base, palette)
		return
	}
	b.reset()
}

// 冷載缺少方向時，不把原版的撤圖誤畫成新效果。
// 只有已知COPY基底或完整黑底能證實首個可見方向。
func sceneEffectStartsVisible(t *Theme, s *sceneEffect, before []byte) bool {
	if len(before) != 64000 {
		return false
	}
	blank := true
	for y := 0; y < s.h; y++ {
		for x := 0; x < s.w; x++ {
			if before[(s.y+y)*320+s.x+x] != 0 {
				blank = false
			}
		}
	}
	if blank {
		return true
	}
	after := append([]byte(nil), before...)
	for y := 0; y < s.h; y++ {
		for x := 0; x < s.w; x++ {
			after[(s.y+y)*320+s.x+x] ^= s.indexed[y*s.w+x]
		}
	}
	knownBefore, knownAfter := false, false
	for _, g := range t.groups {
		w := g.watch
		if !g.sceneCopy || w.X >= s.x+s.w || w.X+w.W <= s.x || w.Y >= s.y+s.h || w.Y+w.H <= s.y {
			continue
		}
		a, err := pbl.Region(before, 320, 200, w.X, w.Y, w.W, w.H)
		if err == nil && bytes.Equal(a, w.Want) {
			knownBefore = true
		}
		a, err = pbl.Region(after, 320, 200, w.X, w.Y, w.W, w.H)
		if err == nil && bytes.Equal(a, w.Want) {
			knownAfter = true
		}
	}
	return knownBefore && !knownAfter
}
