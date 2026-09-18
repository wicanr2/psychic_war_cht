package translator

// docs/spec/011 §5 第 1 項：每一筆圖檔疊字都要在「原版圖塊」上被觸發、蓋在該蓋的位置。
//
// 用合成畫面驗證：把解出來的原版圖貼到空白畫面上該在的座標，跑一次 Frame，
// 檢查 watcher 有沒有比對到、疊字的幾何對不對、字有沒有畫在區塊裡。
// 這樣連玩不到的房間（大部分招牌）也驗得到——像素仍然是原版的，只有「那個房間會不會出現」沒驗。

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/dosgolem/xlate"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
)

var ega = [16][3]uint8{
	{0, 0, 0}, {0, 0, 0xAA}, {0, 0xAA, 0}, {0, 0xAA, 0xAA},
	{0xAA, 0, 0}, {0xAA, 0, 0xAA}, {0xAA, 0x55, 0}, {0xAA, 0xAA, 0xAA},
	{0x55, 0x55, 0x55}, {0x55, 0x55, 0xFF}, {0x55, 0xFF, 0x55}, {0x55, 0xFF, 0xFF},
	{0xFF, 0x55, 0x55}, {0xFF, 0x55, 0xFF}, {0xFF, 0xFF, 0x55}, {0xFF, 0xFF, 0xFF},
}

func TestBakedEntriesOnSyntheticScreen(t *testing.T) {
	root := "../../.."
	orig := filepath.Join(root, "workplace/original/psychic-war")
	if _, err := os.Stat(orig); err != nil {
		t.Skip("缺原版目錄")
	}
	entries, err := LoadBaked(filepath.Join(root, "text"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Skip("text/baked.json 沒有條目")
	}
	f24, err := xlate.LoadFont(filepath.Join(root, "font/cjk24.golemfnt"))
	if err != nil {
		t.Skip("缺字型子集")
	}
	f16, err := xlate.LoadFont(filepath.Join(root, "font/cjk16.golemfnt"))
	if err != nil {
		t.Skip("缺字型子集")
	}
	const W, H, S = 320, 200, 3
	for _, e := range entries {
		e := e
		t.Run(e.Key, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(orig, e.File))
			if err != nil {
				t.Skip("缺 " + e.File)
			}
			iw, ih, ipx, err := pbl.Decode(data, e.Image)
			if err != nil {
				t.Fatal(err)
			}
			idx := make([]uint8, W*H)
			for y := 0; y < ih; y++ {
				for x := 0; x < iw; x++ {
					sx, sy := e.Screen[0]+x, e.Screen[1]+y
					if sx >= 0 && sx < W && sy >= 0 && sy < H {
						idx[sy*W+sx] = ipx[y*iw+x]
					}
				}
			}
			rgb := make([]uint8, 3*W*H)
			for i, v := range idx {
				c := ega[v&0xF]
				rgb[3*i], rgb[3*i+1], rgb[3*i+2] = c[0], c[1], c[2]
			}
			tr := NewTranslator(map[string]TextEntry{}, f24, f16, S, nil)
			tr.AttachBaked([]BakedEntry{e}, orig)
			if tr.Layer.Watchers() != 1 {
				t.Fatalf("watcher 沒登記（%d）", tr.Layer.Watchers())
			}
			tr.Layer.Frame(idx, rgb)
			if len(tr.Layer.Stamps) != 1 {
				t.Fatalf("原版圖塊在畫面上時應該蓋中文，卻有 %d 筆", len(tr.Layer.Stamps))
			}
			s := tr.Layer.Stamps[0]
			wantX, wantY := e.Screen[0]+e.Text[0], e.Screen[1]+e.Text[1]
			if s.X != wantX || s.Y != wantY || s.Cells != e.Text[2] {
				t.Fatalf("幾何不對：(%d,%d) %d 格，要 (%d,%d) %d 格", s.X, s.Y, s.Cells, wantX, wantY, e.Text[2])
			}
			if s.State != xlate.Shown {
				t.Fatalf("定色後應該顯示，state=%v", s.State)
			}
			dst := make([]uint8, 4*W*S*H*S)
			if !tr.Layer.Draw(dst, S, nil) {
				t.Fatal("沒有畫出任何東西")
			}
			ink := 0
			for y := s.Y * S; y < (s.Y+s.CellH)*S; y++ {
				for x := s.X * S; x < (s.X+s.Cells*s.CellW)*S; x++ {
					i := 4 * (y*W*S + x)
					if dst[i+3] != 0 {
						ink++
					}
				}
			}
			if ink == 0 {
				t.Fatal("中文沒有畫進矩形裡")
			}
			// 反向對照：畫面是空白時不該蓋
			blank := make([]uint8, W*H)
			blankRGB := make([]uint8, 3*W*H)
			tr2 := NewTranslator(map[string]TextEntry{}, f24, f16, S, nil)
			tr2.AttachBaked([]BakedEntry{e}, orig)
			tr2.Layer.Frame(blank, blankRGB)
			if len(tr2.Layer.Stamps) != 0 {
				t.Fatal("空白畫面不該觸發")
			}
		})
	}
}
