package theme

import "bytes"

// DrawOriginalLabels保留無字重繪的原位英文前景（024 §1.39）。
// 只在完整原圖仍在畫面、且當幀HD像素全不透明時補原字模。
func (t *Theme) DrawOriginalLabels(dst []byte, scale int, indexed, rgb []byte) bool {
	if t == nil || !t.Enabled || len(t.originalElevator) != 72*72 || scale <= 0 ||
		len(indexed) != 320*200 || len(rgb) != 320*200*3 ||
		scale > int(^uint(0)>>1)/(320*200*4)/scale || len(dst) != 320*200*scale*scale*4 {
		return false
	}
	for y := 0; y < 72; y++ {
		if !bytes.Equal(indexed[(124+y)*320+4:(124+y)*320+76], t.originalElevator[y*72:(y+1)*72]) {
			return false
		}
	}
	drawn := false
	for y := 11; y < 16; y++ {
		for x := 8; x < 45; x++ {
			if t.originalElevator[y*72+x] != 12 {
				continue
			}
			source := 3 * ((124+y)*320 + 4 + x)
			for yy := (124 + y) * scale; yy < (125+y)*scale; yy++ {
				for xx := (4 + x) * scale; xx < (5+x)*scale; xx++ {
					at := 4 * (yy*320*scale + xx)
					if dst[at+3] == 255 {
						copy(dst[at:at+3], rgb[source:source+3])
						drawn = true
					}
				}
			}
		}
	}
	return drawn
}
