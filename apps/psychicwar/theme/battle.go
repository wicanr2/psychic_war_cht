package theme

// 024 §1.49–§1.50：已知場景的唯讀來源、冷載及B透光；未知場景保留原版。
import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"math/big"
	"os"
	"path/filepath"

	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
)

type BuiltinMaskEntry struct {
	ID    string `json:"id"`
	At    []int  `json:"at"`
	PNG   string `json:"png"`
	Kind  string `json:"kind"`
	Match []int  `json:"match"`
}

func (e *BuiltinMaskEntry) UnmarshalJSON(b []byte) error {
	type entry BuiltinMaskEntry
	var v entry
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&v); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(b, &fields); err != nil {
		return err
	}
	for _, name := range []string{"id", "at", "png", "kind", "match"} {
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("內建遮罩缺少 %s", name)
		}
	}
	*e = BuiltinMaskEntry(v)
	return nil
}

const battleMaskSHA = "e1aa2b9ddb6488a70d573ca3a03e71028cd8a2dafa68087c6f6f3d7c60950a5a"
const battleEXESHA = "88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49"

type battleAsset struct {
	name       string
	image      int
	rect       [4]int
	body       bool
	hidden     bool // 原版墨跡全部落在短圖未知格。
	px, packed []byte
	hd         *image.NRGBA
}
type battleBasisRow struct{ vector, combination *big.Int }
type battleTheme struct {
	profiles     map[[32]byte]*battleTheme
	beamProfiles map[[32]byte]map[[32]byte]*battleTheme
	active       *battleTheme
	base         []byte
	assets       []battleAsset
	background   []themeGroup
	scale        int
	basis        map[int]battleBasisRow
	baseVector   *big.Int
	value        *big.Int
	pendingValue *big.Int
	pending      bool
	plane        []byte
	maskReady    bool
	ignored      [4]int // 短圖尾段保持原版，不參與冷解或覆繪。
	party        *partyTheme
	partyKeys    [4][32]byte
	partyValid   bool
}

func battleEffectEntry(e ThemeEntry) bool {
	return e.PBL == "BEAM.PBL" || e.PBL == "FIGHT.PBL" || knownEnemySource(e.PBL) && e.Image >= 15
}

// 原位白名單來自研究038 §180／§196及原版17槽與八向偏移。
func battlePosition(name string, n, x, y int) bool {
	if knownEnemySource(name) {
		return n >= 15 && n < 30 && enemySmallGroupAllowed(name, (n-15)/3) && (y == 160 || y == 168) && x >= 40 && x <= 264 && (x-40)%16 == 0
	}
	switch name {
	case "BEAM.PBL":
		return n >= 0 && n < 12 && y == 160 && x >= 40 && x <= 248 && (x-40)%16 == 0
	case "FIGHT.PBL":
		if n >= 0 && n <= 1 {
			return y == 144 && x >= 160 && x <= 256 && (x-160)%32 == 0
		}
		if n >= 2 && n <= 3 {
			return y == 144 && x == 40
		}
		return n >= 4 && n < 12 && x >= 144 && x <= 240 && (x-144)%32 == 0 && y == 152+(n%2)*16
	case "MASK":
		return n == 0 && x >= 32 && x <= 272 && (y == 160 && (x-32)%16 == 0 || (y == 152 || y == 168) && (x-32)%8 == 0)
	}
	return false
}

func enemySmallGroupAllowed(name string, group int) bool {
	return group >= 0 && group < 5
}

// 只取三段已證實EGA來源；192-byte槽的CGA尾段不參與身份辨識。
func battleSourceKey(work []byte) ([32]byte, bool) {
	if len(work) != 512 {
		return [32]byte{}, false
	}
	key := make([]byte, 0, 384)
	for _, off := range []int{0, 192, 384} {
		key = append(key, work[off:off+128]...)
	}
	return sha256.Sum256(key), true
}

func (b *battleTheme) selectWork(work []byte) {
	b.selectWorkWithBeam(work, nil)
}

// §1.55：新主題同時要求敵人與BEAM的三槽原始bytes，舊主題只沿敵人來源。
func (b *battleTheme) selectWorkWithBeam(work, beam []byte) {
	if b.profiles == nil {
		return
	}
	key, ok := battleSourceKey(work)
	var next *battleTheme
	if ok {
		if b.beamProfiles == nil {
			next = b.profiles[key]
		} else if beamKey, valid := battleSourceKey(beam); valid {
			next = b.beamProfiles[beamKey][key]
		}
	}
	if next != b.active && b.active != nil {
		b.active.clearPrediction()
	}
	b.active = next
}

func (b *battleTheme) selectSource(o *oracle.Oracle) {
	if b.profiles == nil {
		b.selectParty(o)
		return
	}
	a := oracle.Addr{Seg: 0x1175, Off: o.Word(oracle.Addr{Seg: 0x1175, Off: 0xaf12})}
	if a.Linear() > 0xa0000-512 {
		b.selectWork(nil)
		return
	}
	var beam []byte
	if b.beamProfiles != nil {
		ba := oracle.Addr{Seg: 0x1175, Off: o.Word(oracle.Addr{Seg: 0x1175, Off: 0xaf0c})}
		if ba.Linear() <= 0xa0000-512 {
			beam = o.Bytes(ba, 512)
		}
	}
	b.selectWorkWithBeam(o.Bytes(a, 512), beam)
	if b.active != nil {
		b.active.selectParty(o)
	}
}

func battlePNG(root, name string, w, h, scale int, caches ...map[string]*image.NRGBA) (*image.NRGBA, error) {
	images := map[string]*image.NRGBA{}
	if len(caches) > 0 && caches[0] != nil {
		images = caches[0]
	}
	path, err := themeAssetPath(root, name)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%s:%d:%d:%d", path, w, h, scale)
	if cached := images[key]; cached != nil {
		return cached, nil // 所有效果只讀hd.Pix；每次LoadTheme各自持有完整來源組。
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil || cfg.Width != w*scale || cfg.Height != h*scale {
		return nil, fmt.Errorf("效果PNG %s尺寸或解碼不符", name)
	}
	if _, err = f.Seek(0, 0); err != nil {
		return nil, err
	}
	im, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	out := image.NewNRGBA(image.Rect(0, 0, w*scale, h*scale))
	draw.Draw(out, out.Bounds(), im, im.Bounds().Min, draw.Src)
	images[key] = out
	return out, nil
}

func loadBattleTheme(root, orig string, m ThemeManifest, groups []themeGroup, base []byte) (*battleTheme, error) {
	images := map[string]*image.NRGBA{}
	expanded := false
	beamExpanded := false
	beamGroups := map[int]bool{}
	for _, e := range m.Entries {
		if e.PBL == "BEAM.PBL" {
			if e.Image < 0 || e.Image >= 12 {
				return nil, fmt.Errorf("BEAM圖號越界")
			}
			beamGroups[e.Image/3] = true
			beamExpanded = beamExpanded || e.Image >= 3
		}
		if knownEnemySource(e.PBL) && e.Image >= 15 {
			expanded = expanded || e.PBL != "ENEMY00.PBL" || e.Image < 21 || e.Image > 23 || len(e.At) == 2 && e.At[1] != 160
		}
	}
	if !expanded && !beamExpanded {
		return loadBattleProfile(root, orig, m, groups, base, "ENEMY00.PBL", 2, images)
	}
	b := &battleTheme{profiles: map[[32]byte]*battleTheme{}}
	selected := map[string]map[int]bool{}
	for _, e := range m.Entries {
		if !knownEnemySource(e.PBL) || e.Image < 15 {
			continue
		}
		g := (e.Image - 15) / 3
		if !enemySmallGroupAllowed(e.PBL, g) || e.Image >= 30 {
			return nil, fmt.Errorf("尚未READY的敵人小圖 %s #%d", e.PBL, e.Image)
		}
		if selected[e.PBL] == nil {
			selected[e.PBL] = map[int]bool{}
		}
		selected[e.PBL][g] = true
	}
	if beamExpanded && len(selected) == 0 {
		return nil, fmt.Errorf("完整BEAM主題需明示已READY的敵人來源組")
	}
	manifests := []ThemeManifest{m}
	beamKeys := [][32]byte{}
	if beamExpanded {
		b.beamProfiles = map[[32]byte]map[[32]byte]*battleTheme{}
		data, err := os.ReadFile(filepath.Join(orig, "BEAM.PBL"))
		if err != nil {
			return nil, err
		}
		manifests = nil
		for g := 0; g < 4; g++ {
			if !beamGroups[g] {
				continue
			}
			part := m
			part.Entries = nil
			for _, e := range m.Entries {
				if e.PBL != "BEAM.PBL" || e.Image/3 == g {
					part.Entries = append(part.Entries, e)
				}
			}
			key := make([]byte, 0, 384)
			for n := g * 3; n < g*3+3; n++ {
				w, h, px, err := pbl.Decode(data, n)
				if err != nil || w != 16 || h != 16 {
					return nil, fmt.Errorf("BEAM工作源尺寸或解碼不符")
				}
				key = append(key, battlePack(px)...)
			}
			hash := sha256.Sum256(key)
			if b.beamProfiles[hash] != nil {
				return nil, fmt.Errorf("BEAM工作源身份重複")
			}
			b.beamProfiles[hash] = map[[32]byte]*battleTheme{}
			beamKeys = append(beamKeys, hash)
			manifests = append(manifests, part)
		}
	}
	for name, gs := range selected {
		data, err := os.ReadFile(filepath.Join(orig, name))
		if err != nil {
			return nil, err
		}
		for g := range gs {
			key := make([]byte, 0, 384)
			for _, n := range []int{15 + 3*g, 17 + 3*g, 16 + 3*g} {
				w, h, px, err := pbl.Decode(data, n)
				if err != nil || w != 16 || h != 16 {
					return nil, fmt.Errorf("敵人工作源尺寸或解碼不符")
				}
				key = append(key, battlePack(px)...)
			}
			hash := sha256.Sum256(key)
			if b.profiles[hash] != nil {
				return nil, fmt.Errorf("敵人工作源身份重複")
			}
			for n, part := range manifests {
				profile, err := loadBattleProfile(root, orig, part, groups, base, name, g, images)
				if err != nil {
					return nil, err
				}
				if n == 0 {
					b.profiles[hash] = profile
				}
				if beamExpanded {
					b.beamProfiles[beamKeys[n]][hash] = profile
				}
			}
		}
	}
	return b, nil
}

func loadBattleProfile(root, orig string, m ThemeManifest, groups []themeGroup, base []byte, enemy string, group int, caches ...map[string]*image.NRGBA) (*battleTheme, error) {
	images := map[string]*image.NRGBA{}
	if len(caches) > 0 {
		images = caches[0]
	}
	needed := len(m.BuiltinMasks) != 0
	for _, e := range m.Entries {
		needed = needed || battleEffectEntry(e)
	}
	if !needed {
		return nil, nil
	}
	if m.Schema != "psychic-war-theme/2" {
		return nil, fmt.Errorf("戰鬥效果需要主題/2")
	}
	exe, err := os.ReadFile(filepath.Join(orig, "PW.EXE"))
	if err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(exe)) != battleEXESHA {
		return nil, fmt.Errorf("戰鬥遮罩PW.EXE來源SHA-256不符")
	}
	b := &battleTheme{base: append([]byte(nil), base...), scale: m.Scale}
	if enemy == "ENEMY08.PBL" && group == 4 {
		b.ignored = [4]int{32, 176, 24, 8}
	}
	ally, err := loadAlly0(orig)
	if err != nil {
		return nil, err
	}
	for y := 0; y < 32; y++ {
		copy(b.base[(152+y)*320+264:(152+y)*320+288], ally.indexed[y*24:(y+1)*24])
	}
	seen := map[string]bool{}
	add := func(a battleAsset, pngName string) error {
		key := fmt.Sprintf("%s:%d:%v", a.name, a.image, a.rect)
		if seen[key] {
			return fmt.Errorf("重複效果來源 %s", key)
		}
		seen[key] = true
		a.hd, err = battlePNG(root, pngName, a.rect[2], a.rect[3], m.Scale, images)
		if err != nil {
			return err
		}
		if a.px != nil {
			a.packed = battlePack(a.px)
		}
		b.assets = append(b.assets, a)
		return nil
	}
	files := map[string][]byte{}
	alias12 := false
	for _, e := range m.Entries {
		alias12 = alias12 || e.PBL == "ENEMY02.PBL" && e.Image == 12
	}
	for _, e := range m.Entries {
		if knownEnemySource(e.PBL) && (e.PBL != enemy || e.Image < 15 && e.Image/3 != group || e.Image >= 15 && (e.Image-15)/3 != group) {
			continue
		}
		body := e.PBL == enemy && e.Image >= group*3 && e.Image < group*3+3
		if !body && !battleEffectEntry(e) {
			continue
		}
		w, h := 16, 16
		if body || e.PBL == "FIGHT.PBL" && e.Image < 4 {
			w, h = 24, 32
		}
		if body {
			h = enemyHeight(enemy, e.Image)
		}
		if !body && (len(e.At) != 2 || !battlePosition(e.PBL, e.Image, e.At[0], e.At[1]) || e.Kind != "redraw" || !equalInts(e.Match, []int{248, 0, 72, 40}) || e.Src != nil && !equalInts(e.Src, []int{0, 0, w, h}) || e.Scaler != "") {
			return nil, fmt.Errorf("尚未READY的效果来源／原位 %s #%d", e.PBL, e.Image)
		}
		data, ok := files[e.PBL]
		if !ok {
			expected := map[string]string{"BEAM.PBL": "ac1cdff92a20a1ed68de7a228835af6962fa103751a2c427c21cff43abb46045", "FIGHT.PBL": "5dbced5104985c23ba085855fed6d0a6ff27bdac2a180dd3b42eef8d8926ae9d", "ENEMY00.PBL": enemySourceHashes["ENEMY00.PBL"]}[e.PBL]
			if expected == "" {
				expected = enemySourceHashes[e.PBL]
				if expected == "" {
					return nil, fmt.Errorf("尚未READY的效果圖庫 %s", e.PBL)
				}
			}
			data, err = os.ReadFile(filepath.Join(orig, e.PBL))
			if err != nil {
				return nil, err
			}
			offsets, e2 := pbl.Offsets(data)
			count := 12
			if knownEnemySource(e.PBL) {
				count = 30
			}
			if e2 != nil || len(offsets) != count || fmt.Sprintf("%x", sha256.Sum256(data)) != expected {
				return nil, fmt.Errorf("效果%s來源SHA-256或圖數不符", e.PBL)
			}
			files[e.PBL] = data
		}
		pw, ph, px, e2 := pbl.Decode(data, e.Image)
		if e2 != nil || pw != w || ph != h {
			return nil, fmt.Errorf("效果原版尺寸或解碼不符")
		}
		if enemy == "ENEMY02.PBL" && group == 4 && alias12 && e.Image == 13 {
			continue // 等價來源的PNG已由主題載入器逐bytes核對。
		}
		if err = add(battleAsset{name: e.PBL, image: e.Image, rect: [4]int{e.At[0], e.At[1], w, h}, body: body, px: px}, e.PNG); err != nil {
			return nil, err
		}
	}
	for _, e := range m.BuiltinMasks {
		if e.ID != "battle-mask-4e36" || len(e.At) != 2 || !battlePosition("MASK", 0, e.At[0], e.At[1]) || e.Kind != "redraw" || !equalInts(e.Match, []int{248, 0, 72, 40}) {
			return nil, fmt.Errorf("內建遮罩來源或原位不符")
		}
		if err = add(battleAsset{name: "MASK", rect: [4]int{e.At[0], e.At[1], 16, 16}}, e.PNG); err != nil {
			return nil, err
		}
	}
	// 背景圖面直接沿正式載入結果，保留原版格的透明判準。
	keys := map[string]bool{}
	for i, e := range m.Entries {
		if e.PBL == "SCREEN.PBL" || e.PBL == "MENU.PBL" || e.PBL == "ALLY.PBL" && e.Image == 0 && equalInts(e.At, []int{264, 152}) {
			keys[fmt.Sprintf("theme-%03d", i)] = true
		}
	}
	for _, g := range groups {
		if keys[g.watch.Key] {
			b.background = append(b.background, g)
		}
	}
	return b, nil
}

func battlePack(px []byte) []byte {
	p := make([]byte, len(px)/2)
	for i := range p {
		p[i] = px[2*i]<<4 | px[2*i+1]
	}
	return p
}
func battleVector(frame []byte) *big.Int {
	p := make([]byte, 5120)
	k := 0
	for y := 144; y < 184; y++ {
		for x := 32; x < 288; x += 2 {
			p[len(p)-1-k] = frame[y*320+x]<<4 | frame[y*320+x+1]
			k++
		}
	}
	return new(big.Int).SetBytes(p)
}

func (b *battleTheme) ignoredPixel(x, y int) bool {
	return x >= b.ignored[0] && x < b.ignored[0]+b.ignored[2] && y >= b.ignored[1] && y < b.ignored[1]+b.ignored[3]
}

func (b *battleTheme) vector(frame []byte) *big.Int {
	if b.ignored[2] == 0 {
		return battleVector(frame)
	}
	known := append([]byte(nil), frame...)
	for y := b.ignored[1]; y < b.ignored[1]+b.ignored[3]; y++ {
		clear(known[y*320+b.ignored[0] : y*320+b.ignored[0]+b.ignored[2]])
	}
	return battleVector(known)
}

func (b *battleTheme) sourceEqual(a, c []byte, rect [4]int) bool {
	if len(a) != len(c) {
		return false
	}
	for i, v := range a {
		x, y := rect[0]+i*2%rect[2], rect[1]+i*2/rect[2]
		if !b.ignoredPixel(x, y) && v != c[i] {
			return false
		}
	}
	return true
}

func (b *battleTheme) prepare(mask []byte) bool {
	if len(mask) != 32 || fmt.Sprintf("%x", sha256.Sum256(mask)) != battleMaskSHA {
		b.reset()
		return false
	}
	if b.maskReady {
		return true
	}
	b.basis = map[int]battleBasisRow{}
	for n := range b.assets {
		a := &b.assets[n]
		if a.name == "MASK" {
			a.px = make([]byte, 256)
			for y := 0; y < 16; y++ {
				for x := 0; x < 16; x++ {
					if mask[y*2+x/8]&(128>>uint(x%8)) != 0 {
						a.px[y*16+x] = 10
					}
				}
			}
			a.packed = battlePack(a.px)
		}
		frame := make([]byte, 64000)
		for y := 0; y < a.rect[3]; y++ {
			for x := 0; x < a.rect[2]; x++ {
				i := (a.rect[1]+y)*320 + a.rect[0] + x
				frame[i] = a.px[y*a.rect[2]+x]
				if a.body {
					frame[i] ^= b.base[i]
				}
			}
		}
		v := b.vector(frame)
		a.hidden = v.Sign() == 0 && b.ignored[2] != 0
		if a.hidden {
			continue
		}
		combination := new(big.Int).SetBit(new(big.Int), n, 1)
		for v.Sign() != 0 {
			pivot := v.BitLen() - 1
			row, ok := b.basis[pivot]
			if !ok {
				b.basis[pivot] = battleBasisRow{v, combination}
				break
			}
			v.Xor(v, row.vector)
			combination.Xor(combination, row.combination)
		}
		if v.Sign() == 0 {
			b.reset()
			return false
		}
	}
	b.baseVector = b.vector(b.base)
	b.maskReady = true
	return true
}
func (b *battleTheme) legal(value *big.Int) bool {
	used := map[string]bool{}
	for n, a := range b.assets {
		if value.Bit(n) == 0 {
			continue
		}
		key := fmt.Sprintf("%s:%v", a.name, a.rect)
		if a.body {
			key = "body"
		}
		if used[key] {
			return false
		}
		used[key] = true
	}
	return true
}
func (b *battleTheme) solve(frame []byte) *big.Int {
	if !b.maskReady || len(frame) != 64000 {
		return nil
	}
	v := new(big.Int).Xor(b.vector(frame), b.baseVector)
	value := new(big.Int)
	for v.Sign() != 0 {
		row, ok := b.basis[v.BitLen()-1]
		if !ok {
			return nil
		}
		v.Xor(v, row.vector)
		value.Xor(value, row.combination)
	}
	if !b.legal(value) {
		return nil
	}
	return value
}
func (b *battleTheme) model(value *big.Int) []byte {
	p := append([]byte(nil), b.base...)
	for n, a := range b.assets {
		if value.Bit(n) == 0 {
			continue
		}
		for y := 0; y < a.rect[3]; y++ {
			for x := 0; x < a.rect[2]; x++ {
				i := (a.rect[1]+y)*320 + a.rect[0] + x
				v := a.px[y*a.rect[2]+x]
				if a.body {
					v ^= b.base[i]
				}
				p[i] ^= v
			}
		}
	}
	return p
}
func (b *battleTheme) reset() {
	if b.profiles != nil {
		for _, p := range b.allProfiles() {
			p.reset()
		}
		b.active = nil
		return
	}
	b.value = nil
	b.pendingValue = nil
	b.pending = false
	b.plane = nil
	b.maskReady = false
	b.basis = nil
	b.baseVector = nil
	b.partyKeys = [4][32]byte{}
	b.partyValid = false
}

func (b *battleTheme) allProfiles() []*battleTheme {
	seen := map[*battleTheme]bool{}
	var profiles []*battleTheme
	add := func(p *battleTheme) {
		if !seen[p] {
			seen[p] = true
			profiles = append(profiles, p)
		}
	}
	for _, p := range b.profiles {
		add(p)
	}
	for _, bank := range b.beamProfiles {
		for _, p := range bank {
			add(p)
		}
	}
	return profiles
}
func (b *battleTheme) clearPrediction() {
	if b.profiles != nil {
		if b.active != nil {
			b.active.clearPrediction()
		}
		return
	}
	b.value = nil
	b.pendingValue = nil
	b.pending = false
	b.plane = nil
}

// 前場景必須完整可解，來源須唯一對應一個合法轉換。
func (b *battleTheme) begin(rect [4]int, al byte, raw, before []byte, mask bool) {
	prior := b.solve(before)
	b.clearPrediction()
	if prior == nil {
		return
	}
	var chosen *big.Int
	for n, a := range b.assets {
		if a.hidden || a.rect != rect || (a.name == "MASK") != mask {
			continue
		}
		candidate := new(big.Int).Set(prior)
		if mask {
			if !bytes.Equal(raw, a.packed) {
				continue
			}
			candidate.SetBit(candidate, n, 1-candidate.Bit(n))
		} else {
			if al > 1 || al == 0 && !a.body {
				continue
			}
			var old *battleAsset
			oldIndex := -1
			for j, z := range b.assets {
				if z.name == a.name && z.rect == rect && prior.Bit(j) != 0 {
					old = &b.assets[j]
					oldIndex = j
				}
			}
			want := append([]byte(nil), a.packed...)
			if al == 1 && old != nil {
				if oldIndex == n {
					candidate.SetBit(candidate, n, 0)
				} else {
					for k := range want {
						want[k] ^= old.packed[k]
					}
					candidate.SetBit(candidate, oldIndex, 0)
					candidate.SetBit(candidate, n, 1)
				}
			} else {
				if oldIndex >= 0 {
					candidate.SetBit(candidate, oldIndex, 0)
				}
				candidate.SetBit(candidate, n, 1)
			}
			if !b.sourceEqual(raw, want, rect) {
				continue
			}
		}
		if !b.legal(candidate) {
			continue
		}
		if chosen != nil {
			b.clearPrediction()
			return
		}
		chosen = candidate
	}
	// 差分清除可以由同一已知圖號的完整來源辨識，不能猜未登記目標。
	if chosen != nil {
		b.value = chosen
		b.pendingValue = new(big.Int).Set(chosen)
		b.pending = true
	}
}
func (b *battleTheme) blit(r oracle.Regs, raw, before []byte) {
	if b.profiles != nil {
		if b.active != nil {
			b.active.blit(r, raw, before)
		}
		return
	}
	x, y, w, h := int(r.CX>>8)*4, int(r.CX&255)*4, int(r.DX>>8)*8, int(r.DX&255)*8
	if x >= 288 || x+w <= 32 || y >= 184 || y+h <= 144 {
		return
	}
	if w*h/2 > len(raw) || w <= 0 || h <= 0 {
		b.clearPrediction()
		return
	}
	if b.ignored[2] != 0 && [4]int{x, y, w, h} == [4]int{32, 152, 24, 32} {
		h = 24 // 原版尾96 bytes保持顯示，僅比較已證實的身體區域。
	}
	b.begin([4]int{x, y, w, h}, byte(r.AX), raw[:w*h/2], before, false)
}
func (b *battleTheme) mask(r oracle.Regs, raw, before []byte) {
	if b.profiles != nil {
		if b.active != nil {
			b.active.mask(r, raw, before)
		}
		return
	}
	if r.DS != 0x161 || r.DX != 0x4e36 || r.BX < 0xc000 || fmt.Sprintf("%x", sha256.Sum256(raw)) != battleMaskSHA {
		b.clearPrediction()
		return
	}
	px := make([]byte, 256)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if raw[y*2+x/8]&(128>>uint(x%8)) != 0 {
				px[y*16+x] = 10
			}
		}
	}
	p := int(r.BX) - 0xc000
	b.begin([4]int{p % 80 * 4, p / 80, 16, 16}, 1, battlePack(px), before, true)
}
func (b *battleTheme) finish() {
	if b.profiles != nil {
		if b.active != nil {
			b.active.finish()
		}
		return
	}
	b.pending = false
	b.pendingValue = nil
}

func battleInk(a battleAsset, x, y int) bool {
	for yy := y / 8 * 8; yy < y/8*8+8 && yy < a.rect[3]; yy++ {
		for xx := x / 8 * 8; xx < x/8*8+8 && xx < a.rect[2]; xx++ {
			if a.px[yy*a.rect[2]+xx] != 0 {
				return true
			}
		}
	}
	return false
}
func (b *battleTheme) frame(actual []byte, palette [256][3]uint8, mode byte, mask []byte) {
	if b.profiles != nil {
		if b.active != nil {
			b.active.frame(actual, palette, mode, mask)
		}
		return
	}
	b.plane = nil
	if mode != 0x0d || len(actual) != 64000 || b.party != nil && !b.partyValid || !b.prepare(mask) {
		b.clearPrediction()
		return
	}
	for y := 0; y < 40; y++ {
		if !bytes.Equal(actual[y*320+248:(y+1)*320], b.base[y*320+248:(y+1)*320]) {
			b.clearPrediction()
			return
		}
	}
	if value := b.solve(actual); value != nil {
		b.value = value
	} else if b.pending && b.pendingValue != nil {
		b.value = new(big.Int).Set(b.pendingValue)
	} else {
		b.clearPrediction()
		return
	}
	expected := b.model(b.value)
	width, height := 256*b.scale, 40*b.scale
	hd := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := palette[b.base[(144+y/b.scale)*320+32+x/b.scale]]
			i := y*hd.Stride + x*4
			copy(hd.Pix[i:i+3], c[:])
			hd.Pix[i+3] = 255
		}
	}
	for _, g := range b.background {
		for _, r := range g.rows {
			if r.Y < 144 || r.Y >= 184 {
				continue
			}
			for cell := 0; cell < r.Cells; cell++ {
				if len(r.Transparent) > cell && r.Transparent[cell] {
					continue
				}
				x := r.X + cell*8
				if x < 32 || x+8 > 288 {
					continue
				}
				source := image.NRGBA{Pix: r.Pix, Stride: r.Cells * 8 * b.scale * 4, Rect: image.Rect(0, 0, r.Cells*8*b.scale, 8*b.scale)}
				dx, dy := (x-32)*b.scale, (r.Y-144)*b.scale
				draw.Draw(hd, image.Rect(dx, dy, dx+8*b.scale, dy+8*b.scale), &source, image.Pt(cell*8*b.scale, 0), draw.Over)
			}
		}
	}
	for n, a := range b.assets {
		if b.value.Bit(n) == 0 || !a.body {
			continue
		}
		for y := 0; y < a.rect[3]; y += 8 {
			for x := 0; x < a.rect[2]; x += 8 {
				if !battleInk(a, x, y) {
					continue
				}
				dx, dy := (a.rect[0]-32+x)*b.scale, (a.rect[1]-144+y)*b.scale
				draw.Draw(hd, image.Rect(dx, dy, dx+8*b.scale, dy+8*b.scale), a.hd, image.Pt(x*b.scale, y*b.scale), draw.Over)
			}
		}
	}
	trans := make([]float64, width*height*3)
	for i := range trans {
		trans[i] = 1
	}
	for n, a := range b.assets {
		if b.value.Bit(n) == 0 || a.body {
			continue
		}
		for y := 0; y < a.hd.Bounds().Dy(); y++ {
			for x := 0; x < a.hd.Bounds().Dx(); x++ {
				if !battleInk(a, x/b.scale, y/b.scale) {
					continue
				}
				c := a.hd.NRGBAAt(x, y)
				if c.A == 0 {
					continue
				}
				i := ((a.rect[1]-144)*b.scale+y)*width + (a.rect[0]-32)*b.scale + x
				for channel, v := range []uint8{c.R, c.G, c.B} {
					trans[i*3+channel] *= 1 - float64(v)*float64(c.A)/(255*255)
				}
			}
		}
	}
	for i, t := range trans {
		j := i/3*4 + i%3
		hd.Pix[j] = byte(math.Round(255 - (255-float64(hd.Pix[j]))*t))
	}
	for cy := 0; cy < 5; cy++ {
		for cx := 0; cx < 32; cx++ {
			same := true
			for y := 0; y < 8; y++ {
				i := (144+cy*8+y)*320 + 32 + cx*8
				if !bytes.Equal(actual[i:i+8], expected[i:i+8]) {
					same = false
					break
				}
			}
			if !same || b.ignoredPixel(32+cx*8, 144+cy*8) {
				for y := 0; y < 8*b.scale; y++ {
					start := (cy*8*b.scale+y)*hd.Stride + cx*8*b.scale*4
					clear(hd.Pix[start : start+8*b.scale*4])
				}
			}
		}
	}
	b.plane = hd.Pix
}
func (b *battleTheme) draw(dst []byte, scale int) bool {
	if b.profiles != nil {
		return b.active != nil && b.active.draw(dst, scale)
	}
	if scale != b.scale || len(b.plane) != 256*40*scale*scale*4 || len(dst) != 320*200*scale*scale*4 {
		return false
	}
	for y := 0; y < 40*scale; y++ {
		for x := 0; x < 256*scale; x++ {
			from := (y*256*scale + x) * 4
			if b.plane[from+3] == 0 {
				continue
			}
			to := ((144*scale+y)*320*scale + 32*scale + x) * 4
			copy(dst[to:to+4], b.plane[from:from+4])
		}
	}
	return true
}
