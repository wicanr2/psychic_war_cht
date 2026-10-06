package theme

// ALLY 的唯讀出現辨識；#0見024 §1.2／§1.15，#1見§1.16／§1.17，#2見§1.18／§1.21。
import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/dosgolem/xlate"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
)

type spritePresence struct {
	indexed, packed []byte
	x, y, w, h      int
	deltas          []spriteDelta
	active          bool
	inFlight        bool
	retainDelta     bool // 024 §1.6.1／§1.7.1：ENEMY00 #0–#2與ENEMY01 #6–#8。
}

type spriteDelta struct {
	from, packed []byte
}

func newSprite(px []byte, x, y, w, h int) *spritePresence {
	s := &spritePresence{indexed: px, packed: make([]byte, len(px)/2), x: x, y: y, w: w, h: h}
	for i := range s.packed {
		s.packed[i] = px[i*2]<<4 | px[i*2+1]
	}
	return s
}

func (s *spritePresence) slot() [4]int { return [4]int{s.x, s.y, s.w, s.h} }

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func loadAlly0(orig string) (*spritePresence, error) {
	return loadAlly(orig, 0)
}

func loadAlly(orig string, image int) (*spritePresence, error) {
	if image != 0 && image != 1 && image != 2 {
		return nil, fmt.Errorf("主題 ALLY #%d 尚未證實", image)
	}
	b, err := os.ReadFile(filepath.Join(orig, "ALLY.PBL"))
	if err != nil {
		return nil, fmt.Errorf("主題戰友來源：%w", err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(b)) != "c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219" {
		return nil, fmt.Errorf("主題 ALLY.PBL 的 SHA-256 不符")
	}
	offsets, err := pbl.Offsets(b)
	if err != nil || len(offsets) != 31 {
		return nil, fmt.Errorf("主題 ALLY.PBL 圖數不符")
	}
	w, h, px, err := pbl.Decode(b, image)
	if err != nil || w != 24 || h != 32 {
		return nil, fmt.Errorf("主題 ALLY #%d 尺寸或解碼不符", image)
	}
	x := 264
	if image == 1 || image == 2 {
		x = 232
	}
	return newSprite(px, x, 152, 24, 32), nil
}

func (s *spritePresence) frame(indexed []byte, anchor xlate.Watcher) bool {
	if len(indexed) != 320*200 {
		s.active = false
		s.inFlight = false
		return false
	}
	// 錨點消失時不保留上一個畫面的來源認定。
	want, err := pbl.Region(indexed, 320, 200, anchor.X, anchor.Y, anchor.W, anchor.H)
	if err != nil || !bytes.Equal(want, anchor.Want) {
		s.active = false
		s.inFlight = false
		return false
	}
	// 原版逐步改圖；這時完整前一張仍可能吻合，不得推翻入口已確認的新來源。
	if s.inFlight {
		return false
	}
	px, err := pbl.Region(indexed, 320, 200, s.x, s.y, s.w, s.h)
	if err == nil && bytes.Equal(px, s.indexed) {
		s.active = true
		return true
	}
	return false
}

func (s *spritePresence) blit(r oracle.Regs, read func() []byte) {
	x, y := int(r.CX>>8)*4, int(r.CX&255)*4
	w, h := int(r.DX>>8)*8, int(r.DX&255)*8
	if x <= s.x && y <= s.y && x+w >= s.x+s.w && y+h >= s.y+s.h {
		s.active = false
		s.inFlight = true
		if r.AX&255 == 0 && x == s.x && y == s.y && w == s.w && h == s.h {
			s.active = bytes.Equal(read(), s.packed)
		}
	}
}

// 只辨識已證實的完整前姿勢及來源差分；實際 XOR 仍由原版執行。
func (s *spritePresence) blitWithFrame(r oracle.Regs, read, before func() []byte) {
	s.blitWithSource(r, read, before, nil)
}

func (s *spritePresence) blitWithSource(r oracle.Regs, read, before func() []byte, prior []byte) {
	s.blit(r, read)
	if r.AX&255 != 1 || len(s.deltas) == 0 || int(r.CX>>8)*4 != s.x || int(r.CX&255)*4 != s.y ||
		int(r.DX>>8)*8 != s.w || int(r.DX&255)*8 != s.h {
		return
	}
	frame := before()
	if len(frame) != 64000 {
		return
	}
	px, err := pbl.Region(frame, 320, 200, s.x, s.y, s.w, s.h)
	if err != nil {
		return
	}
	for _, delta := range s.deltas {
		if (bytes.Equal(px, delta.from) || s.retainDelta && bytes.Equal(prior, delta.from)) && bytes.Equal(read(), delta.packed) {
			s.active = true
			return
		}
	}
}

// 保存當次入口之前的唯一身份；未知覆蓋的清除不能影響後續組的前身份快照。
func (t *Theme) blitSprites(r oracle.Regs, read, before func() []byte) {
	prior := make(map[[4]int][]byte)
	for i := range t.groups {
		s := t.groups[i].sprite
		if s == nil || !s.retainDelta || r.AX&255 != 1 ||
			int(r.CX>>8)*4 != s.x || int(r.CX&255)*4 != s.y ||
			int(r.DX>>8)*8 != s.w || int(r.DX&255)*8 != s.h {
			continue
		}
		if _, ok := prior[s.slot()]; !ok {
			prior[s.slot()] = t.trustedPredecessor(s.slot(), before())
		}
	}
	for i := range t.groups {
		if s := t.groups[i].sprite; s != nil {
			s.blitWithSource(r, read, before, prior[s.slot()])
		}
	}
}

func (t *Theme) trustedPredecessor(slot [4]int, indexed []byte) []byte {
	if len(indexed) != 64000 {
		return nil
	}
	var found *spritePresence
	for i := range t.groups {
		g := &t.groups[i]
		s := g.sprite
		if s == nil || s.slot() != slot || !s.active {
			continue
		}
		if found != nil || s.inFlight || !s.retainDelta {
			return nil
		}
		anchor, err := pbl.Region(indexed, 320, 200, g.watch.X, g.watch.Y, g.watch.W, g.watch.H)
		if err != nil || !bytes.Equal(anchor, g.watch.Want) {
			return nil
		}
		found = s
	}
	if found == nil {
		return nil
	}
	px, err := pbl.Region(indexed, 320, 200, slot[0], slot[1], slot[2], slot[3])
	if err != nil {
		return nil
	}
	for i := range t.groups {
		s := t.groups[i].sprite
		if s == nil || s.slot() != slot {
			continue
		}
		// 完整原圖比持續身份更強；也查未登記圖面的已知前圖。
		if bytes.Equal(px, s.indexed) && !bytes.Equal(px, found.indexed) {
			return nil
		}
		for _, d := range s.deltas {
			if bytes.Equal(px, d.from) && !bytes.Equal(px, found.indexed) {
				return nil
			}
		}
	}
	return found.indexed
}

// Attach 必須在原版推進前呼叫。讀檔仍使用同一個 Oracle，不重複追加掛鉤。
func (t *Theme) Attach(o *oracle.Oracle) error {
	if t == nil {
		return nil
	}
	if o == nil {
		return fmt.Errorf("主題不能接上空的 Oracle")
	}
	if t.attached == o {
		return nil
	}
	if t.attached != nil {
		return fmt.Errorf("同一主題不能接上第二個 Oracle")
	}
	t.attached = o
	hasSprite := false
	for i := range t.groups {
		if t.groups[i].sprite != nil {
			hasSprite = true
		}
	}
	if !hasSprite {
		return nil
	}
	o.OnCall(oracle.Addr{Seg: 0x0161, Off: 0x8705}, func(o *oracle.Oracle) {
		r := o.Regs()
		var packed []byte
		var indexed []byte
		read := func() []byte {
			if packed == nil {
				a := oracle.Addr{Seg: r.DS, Off: r.BX}
				if a.Linear() > 0xA0000-384 {
					return nil
				}
				packed = o.Bytes(a, 384)
			}
			return packed
		}
		before := func() []byte {
			if indexed == nil {
				indexed = o.Indexed()
			}
			return indexed
		}
		t.blitSprites(r, read, before)
	})
	o.OnCall(oracle.Addr{Seg: 0x0161, Off: 0x8751}, func(*oracle.Oracle) { t.finishBlit() })
	return nil
}

func (t *Theme) finishBlit() {
	for i := range t.groups {
		if s := t.groups[i].sprite; s != nil {
			s.inFlight = false
		}
	}
}
