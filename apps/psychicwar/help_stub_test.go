package psychicwar

import "github.com/wicanr2/dosgolem/xlate"

// Font8x8Stub 是測試用的 8×8 字型，只有一個「A」。
var Font8x8Stub = xlate.Font{W: 8, H: 8, Glyphs: map[rune][]byte{
	'A': {0x18, 0x24, 0x42, 0x7E, 0x42, 0x42, 0x42, 0x00},
}}
