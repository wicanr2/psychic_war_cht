package psychicwar

import "github.com/hajimehoshi/ebiten/v2"

// 按鍵對應（docs/spec/006 §3.4）：Ebiten 按鍵 → IBM PC XT 第 1 組掃描碼。
var scanCodes = map[ebiten.Key]uint8{
	ebiten.KeyEscape: 0x01, ebiten.KeyMinus: 0x0C, ebiten.KeyEqual: 0x0D, ebiten.KeyBackspace: 0x0E, ebiten.KeyTab: 0x0F,
	ebiten.KeyEnter: 0x1C, ebiten.KeyNumpadEnter: 0x1C, ebiten.KeyControlLeft: 0x1D, ebiten.KeyControlRight: 0x1D,
	ebiten.KeyShiftLeft: 0x2A, ebiten.KeyShiftRight: 0x36, ebiten.KeyAltLeft: 0x38, ebiten.KeyAltRight: 0x38,
	ebiten.KeySpace: 0x39, ebiten.KeyComma: 0x33, ebiten.KeyPeriod: 0x34, ebiten.KeySlash: 0x35,
	ebiten.KeyArrowUp: 0x48, ebiten.KeyArrowLeft: 0x4B, ebiten.KeyArrowRight: 0x4D, ebiten.KeyArrowDown: 0x50,
	ebiten.KeyNumpad8: 0x48, ebiten.KeyNumpad4: 0x4B, ebiten.KeyNumpad6: 0x4D, ebiten.KeyNumpad2: 0x50,
	ebiten.KeyF4: 0x3E, ebiten.KeyF5: 0x3F, ebiten.KeyF6: 0x40, ebiten.KeyF7: 0x41, ebiten.KeyF8: 0x42, ebiten.KeyF9: 0x43,
}

func init() {
	for i, k := range []ebiten.Key{ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5,
		ebiten.KeyDigit6, ebiten.KeyDigit7, ebiten.KeyDigit8, ebiten.KeyDigit9, ebiten.KeyDigit0} {
		scanCodes[k] = uint8(0x02 + i)
	}
	rows := []struct {
		keys  string
		first uint8
	}{{"QWERTYUIOP", 0x10}, {"ASDFGHJKL", 0x1E}, {"ZXCVBNM", 0x2C}}
	for _, row := range rows {
		for i, c := range row.keys {
			scanCodes[ebiten.KeyA+ebiten.Key(c-'A')] = row.first + uint8(i)
		}
	}
}

// intercepted 是前端攔下、不送進遊戲的鍵（留給 M5 的說明、語言、地圖、即時存檔）。
var intercepted = map[ebiten.Key]bool{ebiten.KeyF1: true, ebiten.KeyF2: true, ebiten.KeyF3: true, ebiten.KeyF10: true}

// ScanCode 回按鍵的掃描碼；沒有對應或被攔下時 ok 為 false。
func ScanCode(k ebiten.Key) (code uint8, ok bool) {
	if intercepted[k] {
		return 0, false
	}
	code, ok = scanCodes[k]
	return code, ok
}

// Intercepted 回報這個鍵是不是前端自己要處理的（F1／F2／F3／F10）。
func Intercepted(k ebiten.Key) bool { return intercepted[k] }
