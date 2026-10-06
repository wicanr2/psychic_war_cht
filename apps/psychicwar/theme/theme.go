package theme

// SCREEN／MENU 主題載入與顯示生命週期（docs/spec/024 §4.1）。
// 原版資料只讀；圖面與文字使用獨立的 Layer，避免語言開關連帶關閉 HD。
import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/dosgolem/xlate"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
)

type ThemeEntry struct {
	PBL    string `json:"pbl"`
	Image  int    `json:"image"`
	Src    []int  `json:"src,omitempty"`
	At     []int  `json:"at"`
	PNG    string `json:"png"`
	Kind   string `json:"kind"`
	Scaler string `json:"scaler,omitempty"`
	Match  []int  `json:"match,omitempty"`
}

type ThemeManifest struct {
	Schema       string             `json:"schema"`
	Name         string             `json:"name"`
	Title        string             `json:"title,omitempty"`
	Scale        int                `json:"scale"`
	Entries      []ThemeEntry       `json:"entries"`
	Maze         *MazeEntry         `json:"maze,omitempty"`
	BuiltinMasks []BuiltinMaskEntry `json:"builtin_masks,omitempty"`
}

// 缺少圖號不能默認成 #0；同樣拒絕 null 與未知欄位。
func (e *ThemeEntry) UnmarshalJSON(b []byte) error {
	type entry ThemeEntry
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
	for _, name := range []string{"pbl", "image", "at", "png", "kind"} {
		v, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return fmt.Errorf("主題圖面缺少 %s", name)
		}
	}
	*e = ThemeEntry(value)
	return nil
}

type themeGroup struct {
	watch      xlate.Watcher
	rows       []*xlate.Stamp
	sprite     *spritePresence
	registered *xlate.Watcher
}

type Theme struct {
	Name               string
	Enabled            bool
	Layer              xlate.Layer
	groups             []themeGroup
	attached           *oracle.Oracle
	maze               *mazeTheme
	textBackgroundKeys map[string]bool
	originalElevator   []byte
	battle             *battleTheme
}

// LoadTheme 回 nil 表示未選主題或倍率不符；notice 由前端記錄一次。
func LoadTheme(selection, orig, themeRoot string, scale int) (*Theme, string, error) {
	if selection == "" {
		return nil, "", nil
	}
	dir := selection
	if !filepath.IsAbs(dir) && !strings.ContainsAny(dir, `/\`) {
		dir = filepath.Join(themeRoot, dir)
	}
	b, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, "", fmt.Errorf("讀取主題 %s：%w", selection, err)
	}
	var m ThemeManifest
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&m); err != nil {
		return nil, "", fmt.Errorf("主題清單：%w", err)
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return nil, "", fmt.Errorf("主題清單含多餘 JSON")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(b, &fields); err != nil {
		return nil, "", err
	}
	if maze, present := fields["maze"]; present && (m.Schema != "psychic-war-theme/2" || bytes.Equal(bytes.TrimSpace(maze), []byte("null"))) {
		return nil, "", fmt.Errorf("maze 僅接受新版主題的完整資料")
	}
	if masks, present := fields["builtin_masks"]; present && (m.Schema != "psychic-war-theme/2" || bytes.Equal(bytes.TrimSpace(masks), []byte("null"))) {
		return nil, "", fmt.Errorf("builtin_masks 僅接受新版主題的完整清單")
	}
	if (m.Schema != "psychic-war-theme/1" && m.Schema != "psychic-war-theme/2") || m.Name == "" || m.Scale <= 0 || len(m.Entries) == 0 {
		return nil, "", fmt.Errorf("主題清單的格式、名稱、倍率或圖面清單不合法")
	}
	if scale != m.Scale {
		return nil, fmt.Sprintf("主題 %s 需要 %d 倍，目前 %d 倍；沿用原版畫面", m.Name, m.Scale, scale), nil
	}
	// 每張來源最多 320×200，先確認放大後的尺寸乘法不溢位。
	if scale > int(^uint(0)>>1)/(320*200*4)/scale {
		return nil, "", fmt.Errorf("主題倍率過大")
	}
	base, err := themeBackground(orig)
	if err != nil {
		return nil, "", err
	}
	root, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return nil, "", err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, "", err
	}
	t := &Theme{Name: m.Name, Enabled: true, Layer: xlate.Layer{W: 320, H: 200}}
	roomLoaded := make(map[int]bool)
	sourceEntries := make(map[*spritePresence]ThemeEntry)
	seenAliases := make(map[int]bool)
	for i, e := range m.Entries {
		if e.PBL == "ENEMY02.PBL" && (e.Image == 12 || e.Image == 13) {
			if seenAliases[e.Image] {
				return nil, "", fmt.Errorf("ENEMY02別名圖號重複登記")
			}
			seenAliases[e.Image] = true
		}
		if battleEffectEntry(e) {
			continue // 024 §1.49：效果由獨立場景合成，不登記成單張角色。
		}
		ref := base
		var roomSource []byte
		var sprite *spritePresence
		if e.PBL == "ROOM0.PBL" {
			if roomLoaded[e.Image] {
				return nil, "", fmt.Errorf("ROOM0 #%d不能重複登記", e.Image)
			}
			roomLoaded[e.Image] = true
			px, err := loadRoomImage(orig, e.Image)
			if err != nil {
				return nil, "", err
			}
			roomSource = px
			ref = append([]byte(nil), base...)
			for y := 0; y < 72; y++ {
				copy(ref[(124+y)*320+4:(124+y)*320+76], px[y*72:(y+1)*72])
			}
		} else if e.PBL == "OVER.PBL" {
			px, err := loadOver0(orig)
			if err != nil {
				return nil, "", err
			}
			ref = make([]byte, 320*200)
			for y := 0; y < 64; y++ {
				copy(ref[(48+y)*320+128:(48+y)*320+192], px[y*64:(y+1)*64])
			}
		} else if e.PBL == "ALLY.PBL" {
			x, y, positionErr := allyPosition(e)
			if positionErr != nil {
				return nil, "", positionErr
			}
			sprite, err = loadAlly(orig, e.Image)
			if err != nil {
				return nil, "", err
			}
			sprite.x, sprite.y = x, y
		} else if knownEnemySource(e.PBL) {
			sprite, err = loadEnemy(orig, e.PBL, e.Image)
			if err != nil {
				return nil, "", err
			}
		}
		if sprite != nil {
			alias := false
			for _, prior := range t.groups {
				if prior.sprite != nil && prior.sprite.slot() == sprite.slot() && bytes.Equal(prior.sprite.indexed, sprite.indexed) {
					prev := sourceEntries[prior.sprite]
					if prev.PBL != "ENEMY02.PBL" || e.PBL != prev.PBL || !(prev.Image == 12 && e.Image == 13 || prev.Image == 13 && e.Image == 12) ||
						!equalInts(prev.At, e.At) || !equalInts(prev.Src, e.Src) || !equalInts(prev.Match, e.Match) || prev.Kind != e.Kind || prev.Scaler != e.Scaler {
						return nil, "", fmt.Errorf("同位置重複或相同的角色來源，不能辨識唯一動作")
					}
					path, e1 := themeAssetPath(root, e.PNG)
					oldPath, e2 := themeAssetPath(root, prev.PNG)
					if e1 != nil || e2 != nil {
						return nil, "", fmt.Errorf("別名PNG路徑不合法")
					}
					pixels, e1 := os.ReadFile(path)
					oldPixels, e2 := os.ReadFile(oldPath)
					if e1 != nil || e2 != nil || !bytes.Equal(pixels, oldPixels) {
						return nil, "", fmt.Errorf("ENEMY02 #12／#13別名必須使用相同PNG")
					}
					prior.sprite.deltas = append(prior.sprite.deltas, sprite.deltas...)
					alias = true
				}
			}
			if alias {
				continue
			}
			ref = append([]byte(nil), base...)
			for y := 0; y < sprite.h; y++ {
				copy(ref[(sprite.y+y)*320+sprite.x:(sprite.y+y)*320+sprite.x+sprite.w], sprite.indexed[y*sprite.w:(y+1)*sprite.w])
			}
		}
		g, err := loadThemeEntry(root, i, e, scale, ref)
		if err != nil {
			return nil, "", fmt.Errorf("主題 %s 第 %d 筆：%w", m.Name, i+1, err)
		}
		g.sprite = sprite
		if sprite != nil {
			sourceEntries[sprite] = e
		}
		if e.PBL == "ROOM0.PBL" && e.Image == 5 && e.Kind == "redraw" {
			if t.textBackgroundKeys == nil {
				t.textBackgroundKeys = make(map[string]bool)
			}
			t.textBackgroundKeys["ROOM0.PBL:5:elevator"] = true
			t.originalElevator = append([]byte(nil), roomSource...)
		}
		// 新版統一使用迷宮圖集，仍先驗證既有ROOM22資產。
		if m.Maze != nil && e.PBL == "ROOM0.PBL" && e.Image == 22 {
			continue
		}
		t.groups = append(t.groups, g)
	}
	if m.Maze != nil {
		t.maze, err = loadMaze(root, orig, *m.Maze, scale, base)
		if err != nil {
			return nil, "", fmt.Errorf("主題迷宮：%w", err)
		}
	}
	t.battle, err = loadBattleTheme(root, orig, m, t.groups, base)
	if err != nil {
		return nil, "", fmt.Errorf("主題戰鬥效果：%w", err)
	}
	t.ResetForLoad()
	return t, "", nil
}

func themeBackground(orig string) ([]byte, error) {
	base := make([]byte, 320*200)
	for _, source := range []struct {
		name, sha   string
		count, w, h int
	}{
		{"SCREEN.PBL", "302a9724ddbf1872cf6b6ccdad1e2611e0c94663e61b40cb1ee46322d78ddbd1", 5, 320, 40},
		{"MENU.PBL", "b4ef63944ca7a2f37fbd459356e064cab3f8d285296fc701d362ff99aed89641", 1, 88, 72},
	} {
		b, err := os.ReadFile(filepath.Join(orig, source.name))
		if err != nil {
			return nil, fmt.Errorf("主題原版來源 %s：%w", source.name, err)
		}
		hash := sha256.Sum256(b)
		if hex.EncodeToString(hash[:]) != source.sha {
			return nil, fmt.Errorf("主題原版來源 %s 的 SHA-256 不符", source.name)
		}
		offsets, err := pbl.Offsets(b)
		if err != nil || len(offsets) != source.count {
			return nil, fmt.Errorf("主題原版來源 %s 的圖數不符", source.name)
		}
		for n := range offsets {
			w, h, px, err := pbl.Decode(b, n)
			if err != nil || w != source.w || h != source.h {
				return nil, fmt.Errorf("主題原版來源 %s #%d 尺寸或解碼不符", source.name, n)
			}
			x, y := 0, n*40
			if source.name == "MENU.PBL" {
				x, y = 160, 4
			}
			for row := 0; row < h; row++ {
				copy(base[(y+row)*320+x:(y+row)*320+x+w], px[row*w:(row+1)*w])
			}
		}
	}
	return base, nil
}

// OVER是已證實的完整靜態場景來源（024 §1.8），不推定一般貼圖呼叫模式。
func loadOver0(orig string) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(orig, "OVER.PBL"))
	if err != nil {
		return nil, fmt.Errorf("主題結束人物來源：%w", err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(b)) != "57673c27c3c0b141182a8924ac2f2de8490f43b1958369926861f0eaf02b1d1c" {
		return nil, fmt.Errorf("主題OVER.PBL的SHA-256不符")
	}
	offsets, err := pbl.Offsets(b)
	if err != nil || len(offsets) != 1 {
		return nil, fmt.Errorf("主題OVER.PBL圖數不符")
	}
	w, h, px, err := pbl.Decode(b, 0)
	if err != nil || w != 64 || h != 64 {
		return nil, fmt.Errorf("主題OVER #0尺寸或解碼不符")
	}
	return px, nil
}

// 保留§1.9的原呼叫入口。
func loadRoom8(orig string) ([]byte, error) {
	return loadRoomImage(orig, 8)
}

func validRoomImage(image int) bool {
	return image == 0 || image == 2 || image == 3 || image == 5 || image == 8 || image == 22
}

// ROOM0限定完整原圖辨識（024 §1.9／§1.11／§1.12），圖外基準沿用SCREEN／MENU。
func loadRoomImage(orig string, image int) ([]byte, error) {
	if !validRoomImage(image) {
		return nil, fmt.Errorf("ROOM0圖號未證實：%d", image)
	}
	b, err := os.ReadFile(filepath.Join(orig, "ROOM0.PBL"))
	if err != nil {
		return nil, fmt.Errorf("主題房間來源：%w", err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(b)) != "2b2f58c9b716a54bf826dbc9c90237a32458fe49abab52869d5353bbff34d111" {
		return nil, fmt.Errorf("主題ROOM0.PBL的SHA-256不符")
	}
	offsets, err := pbl.Offsets(b)
	if err != nil || len(offsets) != 31 {
		return nil, fmt.Errorf("主題ROOM0.PBL圖數不符")
	}
	w, h, px, err := pbl.Decode(b, image)
	if err != nil || w != 72 || h != 72 {
		return nil, fmt.Errorf("主題ROOM0 #%d尺寸或解碼不符", image)
	}
	return px, nil
}

func loadThemeEntry(root string, index int, e ThemeEntry, scale int, base []byte) (themeGroup, error) {
	var g themeGroup
	w, h, ox, oy := 320, 40, 0, e.Image*40
	switch e.PBL {
	case "SCREEN.PBL":
		if e.Image < 0 || e.Image >= 5 {
			return g, fmt.Errorf("SCREEN 圖號越界")
		}
	case "MENU.PBL":
		if e.Image != 0 {
			return g, fmt.Errorf("MENU 圖號越界")
		}
		w, h, ox, oy = 88, 72, 160, 4
	case "OVER.PBL":
		if e.Image != 0 || (e.Src != nil && !equalInts(e.Src, []int{0, 0, 64, 64})) ||
			(e.Match != nil && !equalInts(e.Match, []int{128, 48, 64, 64})) {
			return g, fmt.Errorf("OVER只支援#0完整圖及原版人物矩形錨點")
		}
		w, h, ox, oy = 64, 64, 128, 48
	case "ROOM0.PBL":
		if !validRoomImage(e.Image) || (e.Src != nil && !equalInts(e.Src, []int{0, 0, 72, 72})) ||
			(e.Match != nil && !equalInts(e.Match, []int{4, 124, 72, 72})) {
			return g, fmt.Errorf("ROOM0只支援#0／#2／#3／#5／#8／#22完整圖及原版房間矩形錨點")
		}
		w, h, ox, oy = 72, 72, 4, 124
	case "ALLY.PBL":
		var err error
		ox, oy, err = allyPosition(e)
		if err != nil {
			return g, err
		}
		w, h = 24, 32
	case "ENEMY00.PBL", "ENEMY01.PBL", "ENEMY02.PBL", "ENEMY03.PBL", "ENEMY04.PBL", "ENEMY05.PBL", "ENEMY06.PBL", "ENEMY07.PBL", "ENEMY08.PBL", "ENEMY09.PBL", "ENEMY10.PBL", "ENEMY11.PBL":
		if _, _, err := enemySource(e.PBL, e.Image); err != nil {
			return g, err
		}
		h = enemyHeight(e.PBL, e.Image)
		if (e.Src != nil && !equalInts(e.Src, []int{0, 0, 24, h})) ||
			!validSpriteMatch(e.Match) {
			return g, fmt.Errorf("%s 只支援完整圖與已證實背景錨點", e.PBL)
		}
		if e.PBL == "ENEMY04.PBL" && !equalInts(e.Match, []int{248, 0, 72, 40}) {
			return g, fmt.Errorf("ENEMY04 必須明示已證實的右側背景錨點")
		}
		w, ox, oy = 24, 32, 152
	default:
		return g, fmt.Errorf("第一批尚未支援來源 %q", e.PBL)
	}
	if e.Kind != "redraw" && e.Kind != "original" {
		return g, fmt.Errorf("kind 必須是 redraw 或 original")
	}
	if e.Scaler != "" && e.Scaler != "nearest" && e.Scaler != "hq3x" {
		return g, fmt.Errorf("未知製作放大器 %q", e.Scaler)
	}
	src := []int{0, 0, w, h}
	if e.Src != nil {
		src = e.Src
	}
	if !validThemeRect(src, w, h) {
		return g, fmt.Errorf("來源矩形越界或格式不符")
	}
	x, y, rw, rh := ox+src[0], oy+src[1], src[2], src[3]
	if len(e.At) != 2 || e.At[0] != x || e.At[1] != y {
		return g, fmt.Errorf("at 必須沿用原版位置 (%d,%d)", x, y)
	}
	match := []int{0, 0, 320, 40}
	if e.PBL == "OVER.PBL" {
		match = []int{128, 48, 64, 64}
	} else if e.PBL == "ROOM0.PBL" {
		match = []int{4, 124, 72, 72}
	}
	if e.Match != nil {
		match = e.Match
	}
	if !validThemeRect(match, 320, 200) {
		return g, fmt.Errorf("比對矩形越界或格式不符")
	}
	path, err := themeAssetPath(root, e.PNG)
	if err != nil {
		return g, err
	}
	f, err := os.Open(path)
	if err != nil {
		return g, err
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		return g, fmt.Errorf("PNG %s：%w", e.PNG, err)
	}
	if cfg.Width != rw*scale || cfg.Height != rh*scale {
		return g, fmt.Errorf("PNG %s 尺寸 %d×%d，應為 %d×%d", e.PNG, cfg.Width, cfg.Height, rw*scale, rh*scale)
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return g, err
	}
	img, err := png.Decode(f)
	if err != nil {
		return g, err
	}
	// 補齊全畫面的 8×8 格；MENU y=4 的上下透明區也使用同一格線。
	x0, y0 := x/8*8, y/8*8
	x1, y1 := (x+rw+7)/8*8, (y+rh+7)/8*8
	padded := image.NewNRGBA(image.Rect(0, 0, (x1-x0)*scale, (y1-y0)*scale))
	draw.Draw(padded, image.Rect((x-x0)*scale, (y-y0)*scale, (x-x0+rw)*scale, (y-y0+rh)*scale), img, img.Bounds().Min, draw.Src)
	key := fmt.Sprintf("theme-%03d", index)
	want, _ := pbl.Region(base, 320, 200, match[0], match[1], match[2], match[3])
	g.watch = xlate.Watcher{Key: key, Art: true, X: match[0], Y: match[1], W: match[2], H: match[3], Want: want}
	for row := y0; row < y1; row += 8 {
		rowKey := fmt.Sprintf("%s-row-%03d", key, row)
		ref, _ := pbl.Region(base, 320, 200, x0, row, x1-x0, 8)
		start := (row - y0) * scale * padded.Stride
		g.rows = append(g.rows, &xlate.Stamp{Key: rowKey, Art: true, X: x0, Y: row, Cells: (x1 - x0) / 8, CellW: 8, CellH: 8, PixScale: scale, Order: index,
			Reference: ref, Pix: padded.Pix[start : start+8*scale*padded.Stride]})
		if e.PBL == "ALLY.PBL" || knownEnemySource(e.PBL) || e.PBL == "OVER.PBL" || e.PBL == "ROOM0.PBL" {
			// 原版全黑的格沒有角色證據，不能因清空後吻合而留下 HD 殘片。
			s := g.rows[len(g.rows)-1]
			s.Transparent = make([]bool, s.Cells)
			for cell := 0; cell < s.Cells; cell++ {
				s.Transparent[cell] = true
				for yy := 0; yy < 8; yy++ {
					for xx := cell * 8; xx < (cell+1)*8; xx++ {
						if e.PBL == "ROOM0.PBL" && (x0+xx < x || x0+xx >= x+rw || row+yy < y || row+yy >= y+rh) {
							continue
						}
						if ref[yy*(x1-x0)+xx] != 0 {
							s.Transparent[cell] = false
						}
					}
				}
			}
		}
		g.watch.ArtKeys = append(g.watch.ArtKeys, rowKey)
	}
	return g, nil
}

// §1.10／§1.15限定明示左右錨點；省略時仍沿用原全寬錨點。
func validSpriteMatch(r []int) bool {
	return r == nil || equalInts(r, []int{0, 0, 320, 40}) || equalInts(r, []int{0, 0, 160, 40}) || equalInts(r, []int{248, 0, 72, 40})
}

// 來源認定與圖面必須共用同一個已證實位置，不能只移動 PNG。
func allyPosition(e ThemeEntry) (int, int, error) {
	if e.Image < 0 || e.Image >= 12 || (e.Src != nil && !equalInts(e.Src, []int{0, 0, 24, 32})) || !validSpriteMatch(e.Match) {
		return 0, 0, fmt.Errorf("ALLY 只支援 #0–#11 完整圖與已證實背景錨點")
	}
	if e.Image >= 3 {
		if equalInts(e.At, []int{128, 8}) && equalInts(e.Match, []int{248, 0, 72, 40}) {
			return 128, 8, nil
		}
		return 0, 0, fmt.Errorf("ALLY #3–#11只支援道具原位置(128,8)，並明示右側錨點")
	}
	if e.Image == 2 {
		if equalInts(e.At, []int{232, 152}) && equalInts(e.Match, []int{248, 0, 72, 40}) {
			return 232, 152, nil
		}
		if equalInts(e.At, []int{128, 8}) && equalInts(e.Match, []int{248, 0, 72, 40}) {
			return 128, 8, nil
		}
		return 0, 0, fmt.Errorf("ALLY #2 只支援原位置 (232,152) 或道具位置 (128,8)，並明示右側錨點")
	}
	if e.Image == 1 {
		if equalInts(e.At, []int{232, 152}) && equalInts(e.Match, []int{248, 0, 72, 40}) {
			return 232, 152, nil
		}
		if equalInts(e.At, []int{128, 8}) && equalInts(e.Match, []int{248, 0, 72, 40}) {
			return 128, 8, nil
		}
		return 0, 0, fmt.Errorf("ALLY #1 只支援原位置 (232,152) 或道具位置 (128,8)，並明示右側錨點")
	}
	if equalInts(e.At, []int{264, 152}) {
		return 264, 152, nil
	}
	if equalInts(e.At, []int{128, 8}) && equalInts(e.Match, []int{248, 0, 72, 40}) {
		return 128, 8, nil
	}
	return 0, 0, fmt.Errorf("ALLY 只支援原位置 (264,152) 或明示右側錨點的道具位置 (128,8)")
}

func validThemeRect(r []int, w, h int) bool {
	return len(r) == 4 && r[0] >= 0 && r[1] >= 0 && r[2] > 0 && r[3] > 0 && r[2] <= w && r[3] <= h && r[0] <= w-r[2] && r[1] <= h-r[3]
}

func themeAssetPath(root, name string) (string, error) {
	if name == "" || filepath.IsAbs(name) || strings.Contains(name, `\`) {
		return "", fmt.Errorf("PNG 路徑不合法：%q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return "", fmt.Errorf("PNG 路徑含父目錄：%q", name)
		}
	}
	path, err := filepath.EvalSymlinks(filepath.Join(root, name))
	if err != nil {
		return "", fmt.Errorf("PNG %s：%w", name, err)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("PNG 路徑越過主題目錄：%q", name)
	}
	return path, nil
}

// ResetForLoad 保持使用者的開關，不保存或沿用上一畫面的遮格。
func (t *Theme) ResetForLoad() {
	if t == nil {
		return
	}
	t.Layer.ClearArt()
	t.Layer.UnwatchAll()
	if t.maze != nil {
		t.maze.clear(&t.Layer)
	}
	if t.battle != nil {
		t.battle.reset()
	}
	for i := range t.groups {
		g := &t.groups[i]
		if g.sprite != nil {
			g.sprite.active = false
			g.sprite.inFlight = false
		}
		w := g.watch
		w.Make = func() []*xlate.Stamp { return g.rows }
		if g.sprite != nil {
			w.Want = nil
		}
		g.registered = &w
		t.Layer.Watch(&w)
	}
}

func (t *Theme) Frame(o *oracle.Oracle) {
	if t == nil {
		return
	}
	w, h, rgb := o.ScreenRGB()
	if w != 320 || h != 200 {
		if t.battle != nil {
			t.battle.reset()
		}
		if t.maze != nil {
			t.maze.clear(&t.Layer)
		}
		for i := range t.groups {
			if t.groups[i].sprite != nil {
				t.groups[i].sprite.active = false
				t.groups[i].sprite.inFlight = false
			}
		}
		t.Layer.Frame(nil, nil)
		return
	}
	indexed := o.Indexed()
	if t.battle != nil {
		t.battle.selectSource(o)
		t.battle.frame(indexed, o.Palette(), o.VideoMode(), o.Bytes(oracle.Addr{Seg: 0x0161, Off: 0x4e36}, 32))
	}
	if t.maze != nil {
		t.maze.frame(&t.Layer, indexed, rgb, o.VideoMode(),
			o.Bytes(oracle.Addr{Seg: 0x1175, Off: 0x16c6}, 2048),
			o.Bytes(oracle.Addr{Seg: 0x0161, Off: 0x307b}, 324))
	}
	t.frameSprites(indexed)
	t.Layer.Frame(indexed, rgb)
}

func (t *Theme) frameSprites(indexed []byte) {
	selected := make(map[[4]int]*spritePresence)
	for i := range t.groups {
		g := &t.groups[i]
		if g.sprite != nil && g.sprite.frame(indexed, g.watch) {
			selected[g.sprite.slot()] = g.sprite
		}
	}
	for i := range t.groups {
		g := &t.groups[i]
		if g.sprite == nil {
			continue
		}
		if current := selected[g.sprite.slot()]; current != nil && current != g.sprite {
			g.sprite.active = false
		}
		g.registered.Want = nil
		if g.sprite.active {
			g.registered.Want = g.watch.Want
		}
	}
}

func (t *Theme) Draw(dst []byte, scale int) bool {
	if t == nil || !t.Enabled {
		return false
	}
	drawn := t.Layer.Draw(dst, scale, nil)
	if t.battle != nil && t.battle.draw(dst, scale) {
		drawn = true
	}
	return drawn
}

// TextBackground只授權已核對無字重繪來源的文字（024 §1.38）。
// 實際像素仍須由當幀原版Watcher及8×8格驗證後的圖面提供。
func (t *Theme) TextBackground(s *xlate.Stamp) bool {
	return t != nil && t.Enabled && s != nil && t.textBackgroundKeys[s.Key]
}

// PremultiplyRGBA 僅供 Ebiten 輸出；Theme 與 PNG 保持非預乘資料。
func PremultiplyRGBA(pix []byte) {
	for i := 0; i+3 < len(pix); i += 4 {
		a := uint32(pix[i+3])
		if a == 255 {
			continue
		}
		for c := 0; c < 3; c++ {
			pix[i+c] = uint8((uint32(pix[i+c])*a + 127) / 255)
		}
	}
}
