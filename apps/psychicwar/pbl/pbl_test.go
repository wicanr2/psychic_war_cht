package pbl

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// docs/spec/011 §5 第 1 項：Go 版解碼與 tools/pbl.py 的輸出逐 byte 相同。
// 原版是玩家自備的，缺檔就 skip（CLAUDE.md 的硬規則）。
func TestDecodeMatchesPythonDump(t *testing.T) {
	root := "../../.."
	src := filepath.Join(root, "workplace/original/psychic-war/SCREEN.PBL")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Skip("缺原版 SCREEN.PBL")
	}
	for n := 0; n < 5; n++ {
		ref, err := os.ReadFile(filepath.Join(root, "workplace/pbl", "SCREEN-0"+string(rune('0'+n))+".idx"))
		if err != nil {
			t.Skip("缺 tools/pbl.py 的輸出（先跑 tools/py.sh tools/pbl.py dump）")
		}
		w, h, px, err := Decode(data, n)
		if err != nil {
			t.Fatalf("第 %d 張解不開：%v", n, err)
		}
		rw := int(binary.LittleEndian.Uint16(ref[0:]))
		rh := int(binary.LittleEndian.Uint16(ref[2:]))
		if w != rw || h != rh {
			t.Fatalf("第 %d 張尺寸不同：%d×%d vs %d×%d", n, w, h, rw, rh)
		}
		if string(px) != string(ref[4:]) {
			diff := 0
			for i := range px {
				if px[i] != ref[4+i] {
					diff++
				}
			}
			t.Fatalf("第 %d 張色號不同：%d／%d 格", n, diff, len(px))
		}
	}
}

func TestOffsetsRejectsGarbage(t *testing.T) {
	for _, bad := range [][]byte{{}, {1}, {3, 0, 9, 9}, {2, 0}} {
		if _, err := Offsets(bad); err == nil {
			t.Errorf("%v 應該回錯", bad)
		}
	}
}

func TestRegionBounds(t *testing.T) {
	px := make([]uint8, 16*16)
	for i := range px {
		px[i] = uint8(i % 7)
	}
	if _, err := Region(px, 16, 16, 12, 0, 8, 4); err == nil {
		t.Error("超出右緣應該回錯")
	}
	got, err := Region(px, 16, 16, 2, 1, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []uint8{px[1*16+2], px[1*16+3], px[1*16+4], px[2*16+2], px[2*16+3], px[2*16+4]}
	if string(got) != string(want) {
		t.Errorf("取出來的區塊不對：%v vs %v", got, want)
	}
}
