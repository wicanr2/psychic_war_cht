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
	// F1／F2／F3 是原版自己的功能鍵（使用者實測 2026-09-19：三個都有作用）。
	// 少了這三個掃描碼，就算不攔也送不進遊戲。
	ebiten.KeyF1: 0x3B, ebiten.KeyF2: 0x3C, ebiten.KeyF3: 0x3D,
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

// intercepted 是前端攔下、不送進遊戲的鍵（docs/spec/012 §1）：
// F4 說明、F5 語言、F6 地圖、F7／F8 作弊（docs/spec/014）、F10 存檔、F11 讀檔。
//
// **F1／F2／F3 不在這裡**：那三個是原版自己的功能鍵，攔下來等於拿掉遊戲功能。
// F9 也留給原版。
var intercepted = map[ebiten.Key]bool{
	ebiten.KeyF4: true, ebiten.KeyF5: true, ebiten.KeyF6: true,
	ebiten.KeyF7: true, ebiten.KeyF8: true, ebiten.KeyF10: true, ebiten.KeyF11: true}

// ScanCode 回按鍵的掃描碼；沒有對應或被攔下時 ok 為 false。
func ScanCode(k ebiten.Key) (code uint8, ok bool) {
	if intercepted[k] {
		return 0, false
	}
	code, ok = scanCodes[k]
	return code, ok
}

// Intercepted 回報這個鍵是不是前端自己要處理的（F4–F8、F10、F11）。
func Intercepted(k ebiten.Key) bool { return intercepted[k] }

// KeyName 回這個鍵在錄製檔與 dosgolem 動作腳本裡的寫法（docs/spec/019 §3）。
// 用 ebiten 的 String()：方向鍵是 ArrowUp，dosgolem 認的是 Up，所以要換一下。
func KeyName(k ebiten.Key) string {
	switch k {
	case ebiten.KeyArrowUp:
		return "Up"
	case ebiten.KeyArrowDown:
		return "Down"
	case ebiten.KeyArrowLeft:
		return "Left"
	case ebiten.KeyArrowRight:
		return "Right"
	case ebiten.KeyEnter:
		return "Return"
	case ebiten.KeyEscape:
		return "Esc"
	case ebiten.KeySpace:
		return "Space"
	case ebiten.KeyBackspace:
		return "Backspace"
	}
	return k.String()
}
