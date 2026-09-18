// Package pbl 解碼原版的 .PBL 圖檔（docs/spec/010）。
//
// 格式是遊戲專屬的，所以留在本 repo（CLAUDE.md 的分層判準）。
// 只讀玩家自備的原版檔，不散布任何原版資料。
package pbl

import (
	"encoding/binary"
	"fmt"
)

// Offsets 回檔案裡每張圖的偏移（spec 010 §2）。
func Offsets(data []byte) ([]int, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("pbl：檔案太短（%d bytes）", len(data))
	}
	t := int(binary.LittleEndian.Uint16(data))
	if t < 2 || t%2 != 0 || t > len(data) {
		return nil, fmt.Errorf("pbl：表長不合理（%d）", t)
	}
	offs := make([]int, t/2)
	for i := range offs {
		o := int(binary.LittleEndian.Uint16(data[2*i:]))
		if o < 0 || o+18 > len(data) {
			return nil, fmt.Errorf("pbl：第 %d 張的偏移越界（%d）", i, o)
		}
		offs[i] = o
	}
	return offs, nil
}

// Decode 解出第 n 張圖，回寬、高（像素）與色號陣列（一格一個色號，長度 寬×高）。
//
// EGA 下不套圖裡的 16 bytes 對照表——那是 4 色模式用的（spec 010 §3.2）。
func Decode(data []byte, n int) (w, h int, px []uint8, err error) {
	offs, err := Offsets(data)
	if err != nil {
		return 0, 0, nil, err
	}
	if n < 0 || n >= len(offs) {
		return 0, 0, nil, fmt.Errorf("pbl：沒有第 %d 張（共 %d 張）", n, len(offs))
	}
	off := offs[n]
	cw, ch := int(data[off]), int(data[off+1])
	need := cw * ch * 32
	if need == 0 {
		return 0, 0, nil, fmt.Errorf("pbl：第 %d 張的尺寸是 0", n)
	}
	out := make([]byte, 0, need)
	si := off + 18
	if si >= len(data) {
		return 0, 0, nil, fmt.Errorf("pbl：第 %d 張沒有資料", n)
	}
	prev := data[si]
	si++
	for len(out) < need && si < len(data) {
		cur := data[si]
		si++
		if cur == prev { // 重複段
			if si >= len(data) {
				break
			}
			cnt := int(data[si])
			si++
			if cnt > need-len(out) {
				cnt = need - len(out)
			}
			for i := 0; i < cnt; i++ {
				out = append(out, prev)
			}
			if len(out) >= need || si >= len(data) {
				break
			}
			prev = data[si]
			si++
			continue
		}
		out = append(out, prev)
		prev = cur
	}
	for len(out) < need { // 資料提前用完：補 0，呼叫端會在比對時發現
		out = append(out, 0)
	}
	w, h = cw*8, ch*8
	px = make([]uint8, w*h)
	for y := 0; y < h; y++ {
		row := out[y*(w/2) : (y+1)*(w/2)]
		for x2, b := range row {
			px[y*w+2*x2] = b >> 4
			px[y*w+2*x2+1] = b & 0xF
		}
	}
	return w, h, px, nil
}

// Region 從 Decode 的色號陣列裡取一塊。
func Region(px []uint8, w, h, x, y, rw, rh int) ([]uint8, error) {
	if x < 0 || y < 0 || rw <= 0 || rh <= 0 || x+rw > w || y+rh > h {
		return nil, fmt.Errorf("pbl：區塊 (%d,%d,%d,%d) 超出 %d×%d", x, y, rw, rh, w, h)
	}
	out := make([]uint8, 0, rw*rh)
	for r := 0; r < rh; r++ {
		out = append(out, px[(y+r)*w+x:(y+r)*w+x+rw]...)
	}
	return out, nil
}
