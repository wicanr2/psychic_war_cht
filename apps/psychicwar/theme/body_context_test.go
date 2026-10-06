package theme

import (
	"bytes"
	"os"
	"testing"

	"github.com/wicanr2/dosgolem/oracle"
)

// 024 §1.6.1：使用已核對的實際原圖，避免共用格錯姿勢仍可過關。
func TestBodyContextSourceLifecycle(t *testing.T) {
	testBodyContextSourceLifecycle(t, "ENEMY00.PBL", 0)
}

// 024 §1.7.1／§1.23：歐格斯與ENEMY00其他組各自保留已證實的差分身份。
func TestOogusContextSourceLifecycle(t *testing.T) {
	testBodyContextSourceLifecycle(t, "ENEMY01.PBL", 6)
}

// 024 §1.14.1：葛雷戈林四條原版差分、受遮擋來源及清除規則。
func TestZellwalContextSourceLifecycle(t *testing.T) {
	testBodyContextSourceLifecycle(t, "ENEMY03.PBL", 3)
}

func testBodyContextSourceLifecycle(t *testing.T, source string, base int) {
	orig := os.Getenv("PSYCHICWAR_TEST_ORIG")
	dir := os.Getenv("PSYCHICWAR_TEST_THEME")
	if orig == "" || dir == "" {
		t.Skip("需要明示原版與27筆主題")
	}
	for _, name := range []string{"partial", "reverse", "next", "reverse-next", "wrong-source", "unknown-cover", "unknown-mode", "reload", "ambiguous", "cross-group", "anchor", "in-flight", "complete-contradiction", "missing-source", "off", "other-ready-group"} {
		t.Run(name, func(t *testing.T) {
			hd, _, err := LoadTheme(dir, orig, "", 3)
			if err != nil {
				t.Fatal(err)
			}
			poses := make(map[int]*spritePresence)
			for n := 0; n < 9; n++ {
				file, image := "ENEMY00.PBL", n
				if n < 3 {
					file, image = source, n+base
				}
				s, err := loadEnemy(orig, file, image)
				if err != nil {
					t.Fatal(err)
				}
				for _, g := range hd.groups {
					if g.sprite != nil && g.sprite.slot() == s.slot() && bytes.Equal(g.sprite.indexed, s.indexed) {
						poses[n] = g.sprite
					}
				}
			}
			frame, err := themeBackground(orig)
			if err != nil {
				t.Fatal(err)
			}
			put := func(n int) {
				for y := 0; y < 32; y++ {
					copy(frame[(152+y)*320+32:(152+y)*320+56], poses[n].indexed[y*24:(y+1)*24])
				}
			}
			from, to := 0, 1
			if name == "reverse" {
				from, to = 1, 0
			}
			if name == "next" {
				from, to = 1, 2
			}
			if name == "reverse-next" {
				from, to = 2, 1
			}
			if name == "other-ready-group" {
				from, to = 3, 4
			}
			put(from)
			hd.frameSprites(frame)
			if !poses[from].active {
				t.Fatal("完整起點未辨識")
			}
			frame[152*320+32] ^= 10
			r := oracle.Regs{AX: 1, CX: 0x0826, DX: 0x0304}
			raw := make([]byte, 384)
			for i := range raw {
				raw[i] = poses[from].packed[i] ^ poses[to].packed[i]
			}
			want := name == "partial" || name == "reverse" || name == "next" || name == "reverse-next" || name == "off" || name == "other-ready-group"
			switch name {
			case "wrong-source":
				raw[0] ^= 1
			case "unknown-cover":
				hd.blitSprites(oracle.Regs{CX: 0x0826, DX: 0x0304}, func() []byte { return make([]byte, 384) }, func() []byte { return frame })
				hd.finishBlit()
			case "unknown-mode":
				r.AX = 2
			case "reload":
				hd.ResetForLoad()
			case "ambiguous":
				poses[3].active = true
			case "cross-group":
				poses[from].active = false
				poses[3].active = true
			case "anchor":
				frame[248] ^= 1
			case "in-flight":
				poses[from].inFlight = true
			case "complete-contradiction":
				put(1)
				to = 0 // 原版完整前姿勢比殘留的來源身份更強。
				want = true
			case "missing-source":
				raw = nil
			case "off":
				hd.Enabled = false
			}
			hd.blitSprites(r, func() []byte { return raw }, func() []byte { return frame })
			if poses[to].active != want {
				t.Fatalf("目標%d active=%v，期望%v", to, poses[to].active, want)
			}
			active := 0
			for _, s := range poses {
				if s.active {
					active++
				}
			}
			if active > 1 {
				t.Fatal("同位置多個動作同時有效")
			}
			if want {
				put(to)
				frame[152*320+32] ^= 10
				hd.frameSprites(frame)
				hd.Layer.Frame(frame, make([]byte, 64000*3))
				plane := make([]byte, 960*600*4)
				hd.Draw(plane, 3)
				if name == "off" {
					if !bytes.Equal(plane, make([]byte, len(plane))) {
						t.Fatal("HD關閉仍畫圖")
					}
					hd.Enabled = true
					hd.Draw(plane, 3)
				}
				p := 4 * ((152*3)*960 + 32*3)
				if !bytes.Equal(plane[p:p+4], []byte{0, 0, 0, 0}) {
					t.Fatal("被改畫格仍有HD")
				}
				visible := false
				for y := 152 * 3; y < 184*3; y++ {
					for x := 32 * 3; x < 56*3; x++ {
						visible = visible || plane[(y*960+x)*4+3] != 0
					}
				}
				if !visible {
					t.Fatal("其他吻合格沒有HD")
				}
				hd.finishBlit()
				hd.ResetForLoad()
				hd.frameSprites(frame)
				if poses[to].active || poses[to].inFlight {
					t.Fatal("冷載受遮擋姿勢保留身份")
				}
			}
		})
	}
}
