// 研究 038 §38：唯讀、可丟棄的效果分解／重疊／讀檔原型，不接入正式前端。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/dosgolem/xlate"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/theme"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/translator"
)

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}
func read(p string) []byte { b, e := os.ReadFile(p); check(e); return b }
func hash(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func write(p string, b []byte) {
	f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(e)
	_, e = f.Write(b)
	check(e)
	check(f.Close())
}
func writePNG(p string, im image.Image) {
	f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(e)
	check(png.Encode(f, im))
	check(f.Close())
}
func loadPNG(p string) *image.NRGBA {
	f, e := os.Open(p)
	check(e)
	im, e := png.Decode(f)
	check(e)
	check(f.Close())
	dst := image.NewNRGBA(im.Bounds())
	draw.Draw(dst, dst.Bounds(), im, im.Bounds().Min, draw.Src)
	return dst
}

type variable struct {
	Name  string `json:"name"`
	Rect  [4]int `json:"rect"`
	Image int    `json:"image"`
}
type sample struct {
	Kind   string `json:"kind"`
	Event  int    `json:"event"`
	Side   string `json:"side"`
	Step   uint64 `json:"step"`
	SHA    string `json:"frame_sha256"`
	Active []int  `json:"active_variables"`
}
type profile struct {
	Rank      int               `json:"rank"`
	Variables []variable        `json:"variables"`
	Samples   []sample          `json:"samples"`
	Inputs    map[string]string `json:"inputs_sha256"`
}
type basisRow struct{ Vector, Combination *big.Int }
type basis struct {
	Rows      map[int]basisRow
	Ambiguous *big.Int
	Mask      *big.Int
}
type asset struct {
	W, H   int
	Pixels []byte
	HD     *image.NRGBA
}
type scene struct {
	Variables       []variable
	Assets          []asset
	Base            []byte
	BaseHD          *image.NRGBA
	Basis           basis
	Cuts            []basis
	All, BaseVector *big.Int
	Inputs          map[string]string
}
type decoded struct {
	Value, Known *big.Int
	Complete     bool
	Alternatives int
}

func vector(frame []byte) *big.Int {
	if len(frame) != 64000 {
		log.Fatal("色號畫面尺寸不符")
	}
	packed := make([]byte, 5120)
	k := 0
	for y := 144; y < 184; y++ {
		for x := 32; x < 288; x += 2 {
			a, b := frame[y*320+x], frame[y*320+x+1]
			if a > 15 || b > 15 {
				log.Fatal("色號超出 4bpp")
			}
			packed[k] = a<<4 | b
			k++
		}
	}
	for i, j := 0, len(packed)-1; i < j; i, j = i+1, j-1 {
		packed[i], packed[j] = packed[j], packed[i]
	}
	return new(big.Int).SetBytes(packed)
}
func makeBasis(vectors []*big.Int, mask *big.Int) basis {
	b := basis{Rows: map[int]basisRow{}, Ambiguous: new(big.Int), Mask: mask}
	for n, raw := range vectors {
		v := new(big.Int).Set(raw)
		if mask != nil {
			v.And(v, mask)
		}
		comb := new(big.Int).SetBit(new(big.Int), n, 1)
		for v.Sign() != 0 {
			pivot := v.BitLen() - 1
			row, exists := b.Rows[pivot]
			if !exists {
				b.Rows[pivot] = basisRow{v, comb}
				break
			}
			v.Xor(v, row.Vector)
			comb.Xor(comb, row.Combination)
		}
		if v.Sign() == 0 {
			b.Ambiguous.Or(b.Ambiguous, comb)
		}
	}
	return b
}
func (b basis) solve(raw *big.Int) (*big.Int, bool) {
	v := new(big.Int).Set(raw)
	if b.Mask != nil {
		v.And(v, b.Mask)
	}
	result := new(big.Int)
	for v.Sign() != 0 {
		row, ok := b.Rows[v.BitLen()-1]
		if !ok {
			return nil, false
		}
		v.Xor(v, row.Vector)
		result.Xor(result, row.Combination)
	}
	return result, true
}
func (s *scene) legal(value, known *big.Int) bool {
	used := map[string]bool{}
	for n, v := range s.Variables {
		if known.Bit(n) == 0 || value.Bit(n) == 0 {
			continue
		}
		key := fmt.Sprintf("%s:%v", v.Name, v.Rect)
		if used[key] {
			return false
		}
		used[key] = true
	}
	return true
}
func (s *scene) decode(frame []byte) (decoded, bool) {
	difference := new(big.Int).Xor(vector(frame), s.BaseVector)
	if value, ok := s.Basis.solve(difference); ok && s.legal(value, s.All) {
		return decoded{value, new(big.Int).Set(s.All), true, 1}, true
	}
	var value, known *big.Int
	alternatives := 0
	for _, b := range s.Cuts {
		candidate, ok := b.solve(difference)
		if !ok {
			continue
		}
		fixed := new(big.Int).AndNot(s.All, b.Ambiguous)
		if !s.legal(candidate, fixed) {
			continue
		}
		if value == nil {
			value = new(big.Int).Set(candidate)
			known = fixed
		} else {
			known.And(known, fixed)
			known.AndNot(known, new(big.Int).Xor(value, candidate))
		}
		alternatives++
	}
	if value == nil {
		return decoded{}, false
	}
	value.And(value, known)
	return decoded{value, known, false, alternatives}, true
}

func newScene(p profile, dir, original, maskPath string, inputs map[string]string) *scene {
	s := &scene{Variables: p.Variables, Base: make([]byte, 64000), BaseHD: image.NewNRGBA(image.Rect(0, 0, 960, 600)), All: new(big.Int), Inputs: inputs}
	assets := map[string][]asset{}
	for _, source := range []struct {
		Name  string
		Count int
	}{{"SCREEN", 5}, {"MENU", 1}, {"ALLY", 31}, {"BEAM", 12}, {"FIGHT", 12}, {"ENEMY00", 30}} {
		path := filepath.Join(original, source.Name+".PBL")
		b := read(path)
		if hash(b) != p.Inputs[path] {
			log.Fatal("PBL 與獨立核對輸入不同")
		}
		inputs[path] = hash(b)
		offs, e := pbl.Offsets(b)
		check(e)
		if len(offs) != source.Count {
			log.Fatal("實際圖數不同")
		}
		for n := range offs {
			w, h, px, e := pbl.Decode(b, n)
			check(e)
			assets[source.Name] = append(assets[source.Name], asset{W: w, H: h, Pixels: px})
		}
	}
	paint := func(a asset, x, y int) {
		for row := 0; row < a.H; row++ {
			copy(s.Base[(y+row)*320+x:(y+row)*320+x+a.W], a.Pixels[row*a.W:(row+1)*a.W])
		}
	}
	for n := 0; n < 5; n++ {
		paint(assets["SCREEN"][n], 0, n*40)
	}
	paint(assets["MENU"][0], 160, 4)
	paint(assets["ALLY"][0], 264, 152)
	var m theme.ThemeManifest
	mp := filepath.Join(dir, "manifest.json")
	check(json.Unmarshal(read(mp), &m))
	inputs[mp] = hash(read(mp))
	for _, entry := range m.Entries {
		if entry.PBL != "SCREEN.PBL" && entry.PBL != "MENU.PBL" && entry.PBL != "ALLY.PBL" {
			continue
		}
		path := filepath.Join(dir, entry.PNG)
		im := loadPNG(path)
		inputs[path] = hash(read(path))
		draw.Draw(s.BaseHD, image.Rect(entry.At[0]*3, entry.At[1]*3, entry.At[0]*3+im.Bounds().Dx(), entry.At[1]*3+im.Bounds().Dy()), im, image.Point{}, draw.Over)
	}
	for i := 3; i < len(s.BaseHD.Pix); i += 4 {
		if s.BaseHD.Pix[i] != 255 {
			log.Fatal("原型基底必須不透明，避免殘留原版角色／效果")
		}
	}
	exePath := "workplace/ida/PW_UNP.EXE"
	exe := read(exePath)
	if hash(exe) != p.Inputs[exePath] {
		log.Fatal("靜態遮罩來源不同")
	}
	inputs[exePath] = hash(exe)
	off := int(binary.LittleEndian.Uint16(exe[8:10]))*16 + int(binary.LittleEndian.Uint16(exe[22:24]))*16 + 0x4E36
	mask := exe[off : off+32]
	maskPx := make([]byte, 256)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if mask[y*2+x/8]&(128>>uint(x%8)) != 0 {
				maskPx[y*16+x] = 10
			}
		}
	}
	assets["MASK"] = []asset{{W: 16, H: 16, Pixels: maskPx}}
	var vectors []*big.Int
	rectangles := map[[4]int]bool{}
	for n, v := range p.Variables {
		a := assets[v.Name][v.Image]
		if a.W != v.Rect[2] || a.H != v.Rect[3] {
			log.Fatal("變數尺寸不符")
		}
		var path string
		switch {
		case v.Name == "MASK":
			path = maskPath
		case v.Name == "ENEMY00" && a.W == 16:
			path = fmt.Sprintf("workplace/hd/art-in/battle-effect-%d-48-v2-20261001.png", v.Image)
		case v.Name == "ENEMY00":
			path = filepath.Join(dir, fmt.Sprintf("ENEMY00-%02d.png", v.Image))
		default:
			path = fmt.Sprintf("workplace/hd/art-in/%s-%02d.png", v.Name, v.Image)
		}
		a.HD = loadPNG(path)
		if a.HD.Bounds().Dx() != a.W*3 || a.HD.Bounds().Dy() != a.H*3 {
			log.Fatal("候選 PNG 尺寸不符")
		}
		inputs[path] = hash(read(path))
		s.Assets = append(s.Assets, a)
		delta := make([]byte, 64000)
		x, y := v.Rect[0], v.Rect[1]
		for row := 0; row < a.H; row++ {
			for col := 0; col < a.W; col++ {
				i := (y+row)*320 + x + col
				delta[i] = a.Pixels[row*a.W+col]
				if v.Name == "ENEMY00" && a.W == 24 {
					delta[i] ^= s.Base[i]
				}
			}
		}
		vectors = append(vectors, vector(delta))
		rectangles[v.Rect] = true
		s.All.SetBit(s.All, n, 1)
	}
	s.Basis = makeBasis(vectors, nil)
	s.BaseVector = vector(s.Base)
	if len(s.Basis.Rows) != len(p.Variables) || len(s.Basis.Rows) != p.Rank {
		log.Fatal("Go 分解不唯一或與獨立秩不同")
	}
	var rects [][4]int
	for r := range rectangles {
		rects = append(rects, r)
	}
	sort.Slice(rects, func(i, j int) bool { return fmt.Sprint(rects[i]) < fmt.Sprint(rects[j]) })
	for _, r := range rects {
		allowed := make([]byte, 64000)
		for y := 144; y < 184; y++ {
			for x := 32; x < 288; x++ {
				if x < r[0] || x >= r[0]+r[2] || y < r[1] || y >= r[1]+r[3] {
					allowed[y*320+x] = 15
				}
			}
		}
		s.Cuts = append(s.Cuts, makeBasis(vectors, vector(allowed)))
	}
	return s
}
func (s *scene) model(d decoded) []byte {
	dst := append([]byte(nil), s.Base...)
	for n, v := range s.Variables {
		if d.Value.Bit(n) == 0 {
			continue
		}
		a := s.Assets[n]
		x, y := v.Rect[0], v.Rect[1]
		for row := 0; row < a.H; row++ {
			for col := 0; col < a.W; col++ {
				i := (y+row)*320 + x + col
				value := a.Pixels[row*a.W+col]
				if v.Name == "ENEMY00" && a.W == 24 {
					value ^= s.Base[i]
				}
				dst[i] ^= value
			}
		}
	}
	return dst
}
func rawImage(o *oracle.Oracle) *image.NRGBA {
	w, h, rgb := o.ScreenRGB()
	if w != 320 || h != 200 {
		log.Fatal("原版畫面尺寸不符")
	}
	im := image.NewNRGBA(image.Rect(0, 0, 960, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 960; x++ {
			i, j := 3*((y/3)*320+x/3), 4*(y*960+x)
			copy(im.Pix[j:j+3], rgb[i:i+3])
			im.Pix[j+3] = 255
		}
	}
	return im
}
func ink(a asset, x, y int) bool {
	x = x / 24 * 8
	y = y / 24 * 8
	for yy := y; yy < y+8 && yy < a.H; yy++ {
		for xx := x; xx < x+8 && xx < a.W; xx++ {
			if a.Pixels[yy*a.W+xx] != 0 {
				return true
			}
		}
	}
	return false
}
func (s *scene) render(o *oracle.Oracle, mode string) (*image.NRGBA, map[string]any) {
	actual := o.Indexed()
	out := rawImage(o)
	d, ok := s.decode(actual)
	if !ok {
		return out, map[string]any{"recognized": false, "fallback": "未知場景，原版完整回退"}
	}
	expected := s.model(d)
	hd := image.NewNRGBA(s.BaseHD.Bounds())
	copy(hd.Pix, s.BaseHD.Pix)
	hidden := make([]bool, 1000)
	active := []int{}
	for n, v := range s.Variables {
		if d.Known.Bit(n) == 0 {
			r := v.Rect
			for y := r[1] / 8; y < (r[1]+r[3]+7)/8; y++ {
				for x := r[0] / 8; x < (r[0]+r[2]+7)/8; x++ {
					hidden[y*40+x] = true
				}
			}
		}
		if d.Value.Bit(n) != 0 {
			active = append(active, n)
			if v.Name == "ENEMY00" && v.Rect[2] == 24 {
				r := v.Rect
				draw.Draw(hd, image.Rect(r[0]*3, r[1]*3, (r[0]+r[2])*3, (r[1]+r[3])*3), s.Assets[n].HD, image.Point{}, draw.Over)
			}
		}
	}
	order := func(n int) int {
		v := s.Variables[n]
		switch v.Name {
		case "FIGHT":
			return 0
		case "BEAM":
			return 1
		case "ENEMY00":
			return 2
		default:
			return 3
		}
	}
	var fx []int
	for _, n := range active {
		if !(s.Variables[n].Name == "ENEMY00" && s.Variables[n].Rect[2] == 24) {
			fx = append(fx, n)
		}
	}
	sort.SliceStable(fx, func(i, j int) bool { return order(fx[i]) < order(fx[j]) })
	var transmission []float64
	if mode == "screen" {
		transmission = make([]float64, 960*600*3)
		for i := range transmission {
			transmission[i] = 1
		}
	}
	for _, n := range fx {
		v, a := s.Variables[n], s.Assets[n]
		for y := 0; y < a.H*3; y++ {
			for x := 0; x < a.W*3; x++ {
				if !ink(a, x, y) {
					continue
				}
				c := a.HD.NRGBAAt(x, y)
				if c.A == 0 {
					continue
				}
				j := 4 * ((v.Rect[1]*3+y)*960 + v.Rect[0]*3 + x)
				for channel, value := range []uint8{c.R, c.G, c.B} {
					if mode == "screen" {
						t := (j/4)*3 + channel
						transmission[t] *= 1 - float64(value)*float64(c.A)/(255*255)
					} else {
						hd.Pix[j+channel] = uint8((uint32(value)*uint32(c.A) + uint32(hd.Pix[j+channel])*(255-uint32(c.A)) + 127) / 255)
					}
				}
			}
		}
	}
	if mode == "screen" {
		for i, t := range transmission {
			j := (i/3)*4 + i%3
			hd.Pix[j] = uint8(math.Round(255 - (255-float64(hd.Pix[j]))*t))
		}
	}
	matched, blocked := 0, 0
	for cy := 0; cy < 25; cy++ {
		for cx := 0; cx < 40; cx++ {
			same := !hidden[cy*40+cx]
			for y := cy * 8; y < cy*8+8 && same; y++ {
				if !bytes.Equal(actual[y*320+cx*8:y*320+cx*8+8], expected[y*320+cx*8:y*320+cx*8+8]) {
					same = false
				}
			}
			if !same {
				blocked++
				continue
			}
			matched++
			r := image.Rect(cx*24, cy*24, cx*24+24, cy*24+24)
			draw.Draw(out, r, hd, r.Min, draw.Over)
		}
	}
	return out, map[string]any{"recognized": true, "complete": d.Complete, "consistent_cuts": d.Alternatives, "active_variables": active, "matched_cells": matched, "blocked_cells": blocked, "unknown_variables": new(big.Int).AndNot(s.All, d.Known).Text(16)}
}
func addText(im *image.NRGBA, tr *translator.Translator) {
	over := image.NewNRGBA(im.Bounds())
	if tr.Layer.Draw(over.Pix, 3, tr.MissingGlyph) {
		draw.Draw(im, im.Bounds(), over, image.Point{}, draw.Over)
	}
}
func loadTranslator(o *oracle.Oracle) *translator.Translator {
	entries, e := translator.LoadText("text")
	check(e)
	f24, e := xlate.LoadFont("font/cjk24.golemfnt")
	check(e)
	f16, e := xlate.LoadFont("font/cjk16.golemfnt")
	check(e)
	tr := translator.NewTranslator(entries, f24, f16, 3, nil)
	tr.Attach(o)
	baked, e := translator.LoadBaked("text")
	check(e)
	tr.AttachBaked(baked, "/orig/psychic-war")
	return tr
}

func main() {
	out := flag.String("out", "", "全新輸出前綴")
	profilePath := flag.String("profile", "workplace/hd/battle-effects-decomposition-v1-20261001.json", "已獨立核對的原型 profile")
	dir := flag.String("theme", "workplace/hd/theme-enemy00-v1-20261001", "本機候選主題")
	maskPath := flag.String("mask", "workplace/hd/art-in/battle-effect-mask-48-v2-20261001.png", "內建 0Ah 遮罩候選，明示新版本而不覆寫舊素材")
	flag.Parse()
	if *out == "" {
		log.Fatal("必須指定 -out")
	}
	if paths, e := filepath.Glob(*out + "*"); e != nil || len(paths) > 0 {
		log.Fatal("拒絕覆寫")
	}
	inputs := map[string]string{*profilePath: hash(read(*profilePath)), "tools/hd/prototype_battle_effects.go": hash(read("tools/hd/prototype_battle_effects.go"))}
	var p profile
	check(json.Unmarshal(read(*profilePath), &p))
	if len(p.Samples) != 2162 {
		log.Fatal("完整幀數不符")
	}
	s := newScene(p, *dir, "/orig/psychic-war", *maskPath, inputs)
	orig := "/orig/psychic-war"
	start := "workplace/states/06-name.state"
	encounter := "workplace/states/08-encounter.state"
	inputs[start] = hash(read(start))
	inputs[encounter] = hash(read(encounter))
	inputs[orig+"/PW.EXE"] = hash(read(orig + "/PW.EXE"))
	if inputs[start] != "ac8eec0bbc533b267782298c298ce4a60440d0568266e665c8a5badbf9e0ca4a" || inputs[encounter] != "02d5fdba9181cb7fbec6282d74efbc25396562ba1403893cdf4c579ddfe770ba" {
		log.Fatal("正常起點不符")
	}
	o, e := oracle.Load(orig+"/PW.EXE", orig)
	check(e)
	defer o.Close()
	check(o.LoadStateFile(start))
	o.SetScratch("/tmp/pw-effect-prototype")
	if o.Word(oracle.Addr{Seg: 0x161, Off: 0x41DF}) != 0x86AF {
		log.Fatal("名字起點種子不符")
	}
	tr := loadTranslator(o)
	runTo := func(step uint64) {
		if step < o.Steps() {
			log.Fatal("指令數倒退")
		}
		for o.Steps() < step {
			n := step - o.Steps()
			if n > 50000 {
				n = 50000
			}
			check(o.Run(n))
			tr.Frame(o)
		}
	}
	for i, scan := range []uint8{0x25, 0x1E, 0x17, 0x1C} {
		runTo(35500000 + uint64(i)*1000000)
		o.KeyDown(scan)
		runTo(36000000 + uint64(i)*1000000)
		o.KeyUp(scan)
	}
	for i := 0; i < 6; i++ {
		runTo(43000000 + uint64(i)*8000000)
		o.KeyDown(0x48)
		step := 47000000 + uint64(i)*8000000
		if step > 86000000 {
			step = 86000000
		}
		runTo(step)
		if step < 86000000 {
			o.KeyUp(0x48)
		}
	}
	checkpoint, e := oracle.Load(orig+"/PW.EXE", orig)
	check(e)
	check(checkpoint.LoadStateFile(encounter))
	if len(checkpoint.SearchChanged(o.Save())) != 0 || o.Regs() != checkpoint.Regs() || o.Cycles() != checkpoint.Cycles() || !bytes.Equal(o.Indexed(), checkpoint.Indexed()) {
		log.Fatal("正常名字／前進未接到同狀態遭遇")
	}
	checkpoint.Close()
	// 載回已核對的同狀態起點，保留其原有未來按鍵佇列；中文來源取自上述正常輸出。
	check(o.LoadStateFile(encounter))
	if o.Word(oracle.Addr{Seg: 0x161, Off: 0x41DF}) != 0x5447 {
		log.Fatal("攻擊起點種子不符")
	}
	runPlain := runTo
	spaceDown := false
	runTo = func(step uint64) {
		if !spaceDown && step >= 86500000 {
			runPlain(86500000)
			o.KeyDown(0x39)
			spaceDown = true
		}
		runPlain(step)
	}
	selected := map[uint64]string{}
	seen := map[string]bool{}
	maximum, maxStep := -1, uint64(0)
	for _, sample := range p.Samples {
		if sample.Side != "after" {
			continue
		}
		fx := 0
		for _, n := range sample.Active {
			v := p.Variables[n]
			label := v.Name
			if label == "ENEMY00" && v.Rect[2] == 24 {
				continue
			}
			if !seen[label] {
				selected[sample.Step] = "首次 " + label
				seen[label] = true
			}
			fx++
		}
		if fx > maximum {
			maximum, maxStep = fx, sample.Step
		}
	}
	selected[maxStep] = "最多同時效果"
	selected[p.Samples[len(p.Samples)-1].Step] = "效果全部清除"
	// 各種類第一個貼圖的中途；不挑選會通過的時點。
	partial := map[uint64]string{}
	done := map[string]bool{}
	for _, sample := range p.Samples {
		if sample.Side != "before" {
			continue
		}
		key := sample.Kind
		if key == "event" {
			for _, after := range p.Samples {
				if after.Kind == sample.Kind && after.Event == sample.Event && after.Side == "after" {
					previous := map[int]bool{}
					for _, n := range sample.Active {
						previous[n] = true
					}
					for _, n := range append(append([]int(nil), after.Active...), sample.Active...) {
						current := false
						for _, a := range after.Active {
							if a == n {
								current = true
								break
							}
						}
						if previous[n] == current {
							continue
						}
						v := p.Variables[n]
						if v.Name == "FIGHT" || v.Name == "BEAM" || v.Name == "ENEMY00" && v.Rect[2] == 16 {
							key = v.Name
							break
						}
					}
					break
				}
			}
		}
		if key == "event" || done[key] {
			continue
		}
		done[key] = true
		offsets := []uint64{1, 128, 256}
		if key != "mask" {
			offsets = []uint64{1, 1000, 4000, 8000}
		}
		for _, delta := range offsets {
			partial[sample.Step+delta] = key + " 中途"
		}
	}
	targets := map[uint64]*sample{}
	for i := range p.Samples {
		sample := &p.Samples[i]
		targets[sample.Step] = sample
	}
	var steps []uint64
	for step := range targets {
		steps = append(steps, step)
	}
	for step := range partial {
		if targets[step] == nil {
			steps = append(steps, step)
		}
	}
	sort.Slice(steps, func(i, j int) bool { return steps[i] < steps[j] })
	var results []map[string]any
	completeCount, partialCount := 0, 0
	resumedPartial := false
	for _, step := range steps {
		runTo(step)
		frame := o.Indexed()
		if sample := targets[step]; sample != nil {
			if hash(frame) != sample.SHA {
				log.Fatalf("正常完整幀 %d 與原版收據不同", step)
			}
			d, ok := s.decode(frame)
			want := new(big.Int)
			for _, n := range sample.Active {
				want.SetBit(want, n, 1)
			}
			if !ok || !d.Complete || d.Value.Cmp(want) != 0 {
				log.Fatal("Go 原型分解與獨立來源方向不同")
			}
			completeCount++
		}
		label, render := selected[step]
		if text, ok := partial[step]; ok {
			label, render = text, true
			partialCount++
		}
		if !render {
			continue
		}
		alpha, am := s.render(o, "alpha")
		screen, sm := s.render(o, "screen")
		raw := rawImage(o)
		addText(alpha, tr)
		addText(screen, tr)
		addText(raw, tr)
		prefix := fmt.Sprintf("%s-step%d", *out, step)
		writePNG(prefix+"-original-cht.png", raw)
		writePNG(prefix+"-alpha.png", alpha)
		writePNG(prefix+"-screen.png", screen)
		write(prefix+".frame", frame)
		check(o.SaveStateFile(prefix + ".state"))
		textState, e := tr.Layer.Snapshot()
		check(e)
		write(prefix+".xlate.json", textState)
		clone, e := oracle.Load(orig+"/PW.EXE", orig)
		check(e)
		check(clone.LoadStateFile(prefix + ".state"))
		loaded := loadTranslator(clone)
		check(loaded.Layer.Restore(textState, loaded.Fonts()))
		loaded.Frame(clone)
		ca, _ := s.render(clone, "alpha")
		cs, _ := s.render(clone, "screen")
		addText(ca, loaded)
		addText(cs, loaded)
		if !bytes.Equal(ca.Pix, alpha.Pix) || !bytes.Equal(cs.Pix, screen.Pix) || !bytes.Equal(clone.Indexed(), frame) || clone.Regs() != o.Regs() || clone.Cycles() != o.Cycles() || len(clone.SearchChanged(o.Save())) != 0 {
			log.Fatal("實際讀檔原型、RAM、暫存器或畫面不符")
		}
		resume := label == "最多同時效果" || !resumedPartial && !am["complete"].(bool)
		if resume {
			if !am["complete"].(bool) {
				resumedPartial = true
			}
			check(clone.Run(112000000 - clone.Steps()))
			if hash(clone.Indexed()) != "04449b264486b8daabd02398d4ca679c59218fa74cca11280f65081ac0adad63" || clone.Cycles() != 515130237 {
				log.Fatal("讀回繪製中途／重疊狀態後的正常攻擊終點不同")
			}
			if hash(clone.Bytes(oracle.Addr{}, 1<<20)) != "5e60e554b31c36138aac29ad8e8b4d5c88abb27d02b6ae43c9fc6cd252005ab1" {
				log.Fatal("讀檔後攻擊終點記憶體不同")
			}
		}
		clone.Close()
		row := map[string]any{"step": step, "label": label, "alpha": am, "screen": sm, "indexed_sha256": hash(frame), "alpha_rgba_sha256": hash(alpha.Pix), "screen_rgba_sha256": hash(screen.Pix), "reload_mismatch": 0, "resumed_to_original_end": resume, "cpu_regs": o.Regs(), "cycles": o.Cycles(), "prefix": prefix}
		results = append(results, row)
		if label == "最多同時效果" {
			contact := image.NewNRGBA(image.Rect(0, 0, 2448, 144))
			for n, im := range []*image.NRGBA{raw, alpha, screen} {
				draw.Draw(contact, image.Rect(n*816, 0, (n+1)*816, 144), im, image.Pt(24*3, 140*3), draw.Src)
			}
			writePNG(*out+"-overlap-comparison.png", contact)
		}
	}
	runTo(112000000)
	// 舊收據的 1MiB 匯流排讀取會載入 VGA 鎖存器；只在終點複本上取得該指標。
	check(o.SaveStateFile(*out + "-end.state"))
	endClone, e := oracle.Load(orig+"/PW.EXE", orig)
	check(e)
	check(endClone.LoadStateFile(*out + "-end.state"))
	endRAM := hash(endClone.Bytes(oracle.Addr{}, 1<<20))
	endClone.Close()
	endFrame := hash(o.Indexed())
	if endRAM != "5e60e554b31c36138aac29ad8e8b4d5c88abb27d02b6ae43c9fc6cd252005ab1" || endFrame != "04449b264486b8daabd02398d4ca679c59218fa74cca11280f65081ac0adad63" || o.Cycles() != 515130237 {
		log.Fatal("原型改變正常攻擊終點")
	}
	b, e := json.MarshalIndent(map[string]any{"scope": "024 §1.5 DRAFT 的可丟棄重疊與讀檔原型", "go_version": runtime.Version(), "inputs_sha256": inputs, "complete_frames_verified": completeCount, "partial_samples": partialCount, "rank": len(s.Basis.Rows), "projected_rectangles": len(s.Cuts), "results": results, "normal_name_seed": "86AF", "attack_seed": "5447", "input": "正常名字與六次前進到相同 08；載回同狀態起點的原有佇列，86,500,000 按住空白鍵至 112,000,000", "end_steps": o.Steps(), "end_cycles": o.Cycles(), "end_ram_sha256": endRAM, "end_indexed_sha256": endFrame, "limits": "原型兩種光效尚未由使用者定案，不改正式程式／schema；中途以所有一致單一矩形投影的共識顯示，歧義圖面及不符 8×8 格回退原版；未證明所有敵人／效果或正式 GUI／封包"}, "", "  ")
	check(e)
	write(*out+".json", append(b, '\n'))
	fmt.Printf("正常中文原型：完整 %d 幀、中途 %d 幀、%d 份實際讀檔；完整終點未改變\n", completeCount, partialCount, len(results))
}
