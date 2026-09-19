package psychicwar

// F1 說明頁（docs/spec/012 §3）。內容是這一款遊戲專屬的，留在本 repo。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wicanr2/dosgolem/xlate"
)

// HelpCols 是舊版面（整頁置中、一格 24×24）每行的格數，DrawTextPage 的退路還在用。
// 說明頁本身的版面已經改成 docs/spec/022 §4，行寬與行數的上限由 CheckHelp 實算。
const HelpCols = 38

// 說明頁內容在 text/help.json（LoadHelp 讀），不寫死在程式裡：驗收工具要用同一份算期望值。
// 版面常數（`layout`）與鍵名清單（`keycaps`）也在同一份，理由同上（docs/spec/022 §6 第 1 項）。
// LoadHelp 讀 <dir>/help.json 的說明頁內容與版面（docs/spec/012 §3、docs/spec/022）。
func LoadHelp(dir string) ([]string, error) {
	b, err := os.ReadFile(filepath.Join(dir, "help.json"))
	if err != nil {
		return nil, err
	}
	var doc struct {
		Schema  string     `json:"schema"`
		Lines   []string   `json:"lines"`
		Keycaps []string   `json:"keycaps"`
		Layout  HelpLayout `json:"layout"`
		Label   string     `json:"protection_label"`
		Map     string     `json:"map_header"`
		ASCII   string     `json:"ascii_only_toast"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	if doc.Schema != "psychic-war-help/1" {
		return nil, fmt.Errorf("help.json 的 schema 不是 psychic-war-help/1：%q", doc.Schema)
	}
	// 版面缺了就不要硬畫：程式裡另外寫一份預設值，等於讓資料與程式各有一套常數，
	// 驗收工具與前端會算出不同的圖而兩邊都不報錯。
	if doc.Layout.PageW <= 0 || doc.Layout.PageHeight <= 0 || doc.Layout.FontBodyW <= 0 {
		return nil, fmt.Errorf("help.json 沒有 layout（docs/spec/022 §4）")
	}
	ProtectionLabel = doc.Label
	if doc.Map != "" {
		MapHeader = doc.Map
	}
	if doc.ASCII != "" {
		ASCIIOnlyToast = doc.ASCII
	}
	helpTextDir = dir // 內文字型（cjk16）的備用找法：help.json 同層的 font/
	d := &HelpDoc{Lines: doc.Lines, Keycaps: sortKeycaps(doc.Keycaps), Layout: doc.Layout}
	if err := CheckHelpDoc(d); err != nil {
		return nil, err
	}
	helpDoc = d
	return doc.Lines, nil
}

// ProtectionLabel 是防拷畫面那一行的標籤（LoadHelp 讀進來）。
var ProtectionLabel = "　本題答案："

// MapHeader 是 F3 自動地圖的標題樣板（LoadHelp 讀進來；docs/spec/015 §2）。
var MapHeader = "區域 %d　座標 (%d, %d)　已走 %d 格　朝向 %c"

// ASCIIOnlyToast 是收到非 ASCII 輸入時的提示（docs/spec/017 §3）。
// 中文輸入法送出的是字元不是掃描碼，本來就進不到遊戲裡——玩家按半天沒有反應，
// 畫面上要說出為什麼，不能讓被擋掉的輸入靜默消失。
var ASCIIOnlyToast = "名字只能用英數字（原版的限制）"

// CheckHelp 檢查說明頁排得下；排不下回錯（建置期就擋住，docs/spec/012 §5 第 1 項）。
// 用目前載入的版面算，所以要先 LoadHelp。
func CheckHelp(lines []string) error {
	if helpDoc == nil {
		return fmt.Errorf("說明頁還沒載入（先呼叫 LoadHelp）")
	}
	return CheckHelpDoc(&HelpDoc{Lines: lines, Keycaps: helpDoc.Keycaps, Layout: helpDoc.Layout})
}

// CheckHelpDoc 用同一套版面常數實算，不是數行數（docs/spec/022 §5）。
//
// 新版面的高度不是行數乘以固定列距：標頭與內文的列距不同、段落之間有間距、
// 欄位分配是算出來的。數行數會在「還有幾十像素可用」與「已經超出畫布」之間給出相同的答案，
// 所以這裡直接把版面排一次，看最下面那個像素落在哪裡。防拷那一列一定算進去——
// 它是保留席位，平常不畫，但不能沒有位置。
func CheckHelpDoc(d *HelpDoc) error {
	L := d.Layout
	if len(d.Lines) == 0 {
		return fmt.Errorf("說明頁沒有內容")
	}
	ops := helpGeometry(d, ProtectionLabel+"ANSWER")
	limit := L.PageW - L.Margin
	for _, op := range ops {
		if op.Kind == opRect {
			continue
		}
		fw := L.FontBodyW
		if op.Big {
			fw = L.FontHeadW
		}
		if r := op.X + helpWidth(fw, op.S); r > limit {
			return fmt.Errorf("說明頁「%s」畫到 x=%d，超出可用寬度 %d", op.S, r, limit)
		}
	}
	if b := helpBottom(L, ops); b > L.PageHeight {
		return fmt.Errorf("說明頁內容底緣 y=%d，超出畫布高度 %d", b, L.PageHeight)
	}
	return nil
}

// MissingHelpGlyphs 回說明頁用到、但字型沒有的字。
func MissingHelpGlyphs(lines []string, f *xlate.Font) []rune {
	var out []rune
	seen := map[rune]bool{}
	for _, s := range lines {
		for _, r := range s {
			if r == ' ' || r == '　' || seen[r] || f == nil {
				continue
			}
			if _, ok := f.Glyphs[r]; !ok {
				seen[r] = true
				out = append(out, r)
			}
		}
	}
	return out
}

// DrawTextPage 把整頁文字畫進放大後的 RGBA（寬 w 像素）。
//
// 一格 cell×cell 像素，字模置左上；fg、bg 是 RGB。bgAlpha 0 表示不填背景（只畫字）。
//
// 內容剛好是 help.json 那一頁時（可能尾端多了防拷答案）改走 docs/spec/022 的版面：
// 說明頁有兩種字級、外框、橫線與反白鍵帽，fg／bg 兩個顏色表達不了。版面規則集中在
// helppage.go，呼叫端只交內容——驗收工具才有單一的期望值來源。地圖標題與提示列不受影響。
func DrawTextPage(dst []uint8, w, h int, f *xlate.Font, lines []string, cell int, fg, bg [3]uint8, bgAlpha uint8) {
	if f == nil {
		return
	}
	if ok, prot := helpPageLines(lines); ok && drawHelpPage(dst, w, h, f, prot) {
		return
	}
	if bgAlpha > 0 {
		for i := 0; i < w*h; i++ {
			dst[4*i], dst[4*i+1], dst[4*i+2], dst[4*i+3] = bg[0], bg[1], bg[2], bgAlpha
		}
	}
	rb := (f.W + 7) / 8
	sx := (w - HelpCols*cell) / 2
	sy := (h - len(lines)*cell) / 2
	if sx < 0 {
		sx = 0
	}
	if sy < 0 {
		sy = 0
	}
	scale := cell / f.W
	if scale < 1 {
		scale = 1
	}
	for row, s := range lines {
		pen := 0 // 目前的水平位置，單位是半格
		for _, r := range s {
			adv := 2
			if r < 0x80 {
				adv = 1 // 半形字佔半格
			}
			g, ok := f.Glyphs[r]
			if !ok {
				pen += adv
				continue
			}
			x0, y0 := sx+pen*cell/2, sy+row*cell
			pen += adv
			for gy := 0; gy < f.H; gy++ {
				for gx := 0; gx < f.W; gx++ {
					if g[gy*rb+gx/8]&(0x80>>(gx%8)) == 0 {
						continue
					}
					for py := 0; py < scale; py++ {
						for px := 0; px < scale; px++ {
							x, y := x0+gx*scale+px, y0+gy*scale+py
							if x < 0 || x >= w || y < 0 || y >= h {
								continue
							}
							i := y*w + x
							dst[4*i], dst[4*i+1], dst[4*i+2], dst[4*i+3] = fg[0], fg[1], fg[2], 0xFF
						}
					}
				}
			}
		}
	}
}

// NonASCII 回報這批文字輸入裡有沒有非 ASCII 字元（docs/spec/017 §3）。
//
// 前端把 ebiten.Key 對到掃描碼送給原版；中文輸入法送的是**字元**，不是掃描碼，
// 所以它本來就進不到遊戲裡。玩家按半天沒有反應時，畫面上要說出為什麼。
func NonASCII(rs []rune) bool {
	for _, r := range rs {
		if r > 0x7F {
			return true
		}
	}
	return false
}
