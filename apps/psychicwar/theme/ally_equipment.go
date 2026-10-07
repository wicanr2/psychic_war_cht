package theme

// 024 §1.54：裝備是原版工作區中的完整合成身份，原版RAM只讀。
import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"

	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
)

type partyAsset struct {
	indexed []byte
	groups  [4]*themeGroup
}

type partyTheme struct {
	assets     map[[32]byte]*partyAsset
	base       []byte
	background []themeGroup
}

func allyPartySlot(at []int) int {
	if len(at) == 2 && at[1] == 152 {
		for slot := 0; slot < 4; slot++ {
			if at[0] == 264-slot*32 {
				return slot
			}
		}
	}
	return -1
}

func allyEquipmentEntry(e ThemeEntry) bool {
	return e.PBL == "ALLY.PBL" && e.Image >= 12 && e.Image < 16
}

// 最小alpha及反算前景；重建既有綠底，保留武器曲邊。
func allyEquipmentMatte(src *image.NRGBA) (*image.NRGBA, error) {
	key := [3]byte{85, 255, 85}
	if src.NRGBAAt(0, 0).A != 255 || !bytes.Equal(src.Pix[:3], key[:]) {
		return nil, fmt.Errorf("ALLY裝備PNG底色不符")
	}
	out := image.NewNRGBA(src.Bounds())
	for i := 0; i < len(src.Pix); i += 4 {
		if src.Pix[i+3] != 255 {
			return nil, fmt.Errorf("ALLY裝備PNG須保持已驗不透明綠底")
		}
		alpha := 0.0
		for c, k := range key {
			v := src.Pix[i+c]
			a := 0.0
			if v > k {
				a = float64(v-k) / float64(255-k)
			} else if v < k {
				a = float64(k-v) / float64(k)
			}
			alpha = math.Max(alpha, a)
		}
		a := int(math.Ceil(alpha*255 - 1e-10))
		out.Pix[i+3] = byte(a)
		if a == 0 {
			continue
		}
		for c, k := range key {
			v := math.Round(float64(int(src.Pix[i+c])*255-int(k)*(255-a)) / float64(a))
			out.Pix[i+c] = byte(math.Max(0, math.Min(255, v)))
		}
	}
	return out, nil
}

func allyEquipmentCompose(body, weapon *image.NRGBA) *image.NRGBA {
	out := image.NewNRGBA(body.Bounds())
	copy(out.Pix, body.Pix)
	for i := 0; i < len(out.Pix); i += 4 {
		a := int(weapon.Pix[i+3])
		for c := 0; c < 3; c++ {
			out.Pix[i+c] = byte((int(weapon.Pix[i+c])*a + int(body.Pix[i+c])*(255-a) + 127) / 255)
		}
	}
	return out
}

func loadAllyEquipment(root, orig string, m ThemeManifest, t *Theme, base []byte) (*partyTheme, error) {
	needed := false
	for _, e := range m.Entries {
		needed = needed || allyEquipmentEntry(e)
	}
	if !needed {
		return nil, nil
	}
	if m.Schema != "psychic-war-theme/2" {
		return nil, fmt.Errorf("ALLY裝備需要主題/2")
	}
	data, err := os.ReadFile(filepath.Join(orig, "ALLY.PBL"))
	if err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != "c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219" {
		return nil, fmt.Errorf("ALLY裝備來源SHA不符")
	}
	var raw [16][]byte
	var hd [12]*image.NRGBA
	for n := 0; n < 16; n++ {
		w, h, px, e := pbl.Decode(data, n)
		if e != nil || w != 24 || h != 32 {
			return nil, fmt.Errorf("ALLY装備來源尺寸不符")
		}
		raw[n] = px
	}
	for _, e := range m.Entries {
		if e.PBL != "ALLY.PBL" || e.Image < 0 || e.Image >= 12 {
			continue
		}
		im, er := battlePNG(root, e.PNG, 24, 32, m.Scale)
		if er != nil {
			return nil, er
		}
		for i := 3; i < len(im.Pix); i += 4 {
			if im.Pix[i] != 255 {
				return nil, fmt.Errorf("裝備合成人物須不透明")
			}
		}
		if hd[e.Image] != nil && !bytes.Equal(hd[e.Image].Pix, im.Pix) {
			return nil, fmt.Errorf("同一ALLY人物PNG像素不一致")
		}
		hd[e.Image] = im
	}
	p := &partyTheme{assets: map[[32]byte]*partyAsset{}, base: append([]byte(nil), base...)}
	for n := 0; n < 12; n++ {
		if hd[n] == nil {
			return nil, fmt.Errorf("裝備合成缺ALLY #%d人物PNG", n)
		}
		key := sha256.Sum256(battlePack(raw[n]))
		if p.assets[key] != nil {
			return nil, fmt.Errorf("ALLY人物身份不唯一")
		}
		p.assets[key] = &partyAsset{indexed: raw[n]}
	}
	for i := range t.groups {
		g := &t.groups[i]
		if g.sprite != nil {
			slot := allyPartySlot([]int{g.sprite.x, g.sprite.y})
			if slot >= 0 {
				if a := p.assets[sha256.Sum256(g.sprite.packed)]; a != nil {
					a.groups[slot] = g
				}
			}
		}
		// 只保存正式SCREEN/MENU；隊員由當幀工作源選取。
		for n, e := range m.Entries {
			if (e.PBL == "SCREEN.PBL" || e.PBL == "MENU.PBL") && g.watch.Key == fmt.Sprintf("theme-%03d", n) {
				p.background = append(p.background, *g)
			}
		}
	}
	seen := map[[2]int]bool{}
	for i, e := range m.Entries {
		if !allyEquipmentEntry(e) {
			continue
		}
		if _, _, er := allyPosition(e); er != nil {
			return nil, er
		}
		slot := allyPartySlot(e.At)
		if seen[[2]int{e.Image, slot}] {
			return nil, fmt.Errorf("ALLY裝備原位重複")
		}
		seen[[2]int{e.Image, slot}] = true
		im, er := battlePNG(root, e.PNG, 24, 32, m.Scale)
		if er != nil {
			return nil, er
		}
		matte, er := allyEquipmentMatte(im)
		if er != nil {
			return nil, er
		}
		for actor := 0; actor < 12; actor++ {
			px := append([]byte(nil), raw[actor]...)
			for k, v := range raw[e.Image] {
				if v != 2 && v != 10 {
					px[k] = v
				}
			}
			s := newSprite(px, e.At[0], e.At[1], 24, 32)
			key := sha256.Sum256(s.packed)
			a := p.assets[key]
			if a == nil {
				a = &partyAsset{indexed: px}
				p.assets[key] = a
			} else if !bytes.Equal(a.indexed, px) {
				return nil, fmt.Errorf("ALLY合成SHA碰撞")
			}
			if a.groups[slot] != nil {
				return nil, fmt.Errorf("ALLY合成身份／原位不唯一")
			}
			ref := append([]byte(nil), base...)
			for y := 0; y < 32; y++ {
				copy(ref[(152+y)*320+s.x:(152+y)*320+s.x+24], px[y*24:(y+1)*24])
			}
			g, er := loadThemeEntry(root, len(m.Entries)+i*12+actor, e, m.Scale, ref)
			if er != nil {
				return nil, er
			}
			combined := allyEquipmentCompose(hd[actor], matte)
			for _, row := range g.rows {
				// 四個已證實隊伍位置皆對齊全域8×8。
				start := (row.Y - 152) * m.Scale * combined.Stride
				row.Pix = combined.Pix[start : start+8*m.Scale*combined.Stride]
			}
			g.sprite = s
			// 存在獨立配置，避免追加slice後使指標引用舊副本。
			a.groups[slot] = &g
			t.groups = append(t.groups, g)
		}
	}
	return p, nil
}

func (b *battleTheme) selectParty(o *oracle.Oracle) {
	if b.party == nil {
		return
	}
	b.partyValid = false
	if o.Byte(oracle.Addr{Seg: 0x161, Off: 0x812e}) != 2 {
		b.clearPrediction()
		return
	}
	ptr := o.Word(oracle.Addr{Seg: 0x1175, Off: 0xaee6})
	start := o.Word(oracle.Addr{Seg: 0x1175, Off: 0xaf02})
	if ptr > 0xffff-0x46 {
		b.clearPrediction()
		return
	}
	count := int(o.Byte(oracle.Addr{Seg: 0x1175, Off: ptr + 0x46}) & 3)
	var work [4][]byte
	for slot := 0; slot <= count; slot++ {
		offset := int(start) + slot*0x240
		if offset > 0x10000-384 {
			b.clearPrediction()
			return
		}
		work[slot] = o.Bytes(oracle.Addr{Seg: 0x1175, Off: uint16(offset)}, 384)
	}
	b.selectPartyData(work, count, 2)
}

func (b *battleTheme) selectPartyData(work [4][]byte, count int, driver byte) {
	if b.party == nil {
		return
	}
	b.partyValid = false
	if driver != 2 || count < 0 || count > 3 {
		b.clearPrediction()
		return
	}
	var keys [4][32]byte
	var selected [4]*themeGroup
	for slot := 0; slot <= count; slot++ {
		if len(work[slot]) != 384 {
			b.clearPrediction()
			return
		}
		keys[slot] = sha256.Sum256(work[slot])
		a := b.party.assets[keys[slot]]
		if a == nil || a.groups[slot] == nil {
			b.clearPrediction()
			return
		}
		selected[slot] = a.groups[slot]
	}
	b.partyValid = true
	if keys == b.partyKeys {
		return
	}
	b.clearPrediction()
	b.partyKeys = keys
	b.base = append([]byte(nil), b.party.base...)
	b.background = append([]themeGroup(nil), b.party.background...)
	for slot := 0; slot <= count; slot++ {
		a := b.party.assets[keys[slot]]
		x := 264 - slot*32
		for y := 0; y < 32; y++ {
			copy(b.base[(152+y)*320+x:(152+y)*320+x+24], a.indexed[y*24:(y+1)*24])
		}
		b.background = append(b.background, *selected[slot])
	}
	if b.maskReady {
		b.baseVector = b.vector(b.base)
	}
}
