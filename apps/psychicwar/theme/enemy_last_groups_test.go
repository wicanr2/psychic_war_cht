package theme

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/dosgolem/oracle"
)

// 八份原版實跑來源及畫面來自獨立探針，不由載入器產生期望值。
func TestLastEnemyOriginalSources(t *testing.T) {
	orig, base := os.Getenv("PSYCHICWAR_TEST_ORIG"), os.Getenv("PSYCHICWAR_LAST_ENEMY_PROOF")
	if orig == "" || base == "" {
		t.Skip("需最後兩組原版受控實跑證據")
	}
	read := func(p string) []byte {
		v, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	for _, bank := range []int{2, 8} {
		for phase, target := range []int{13, 14, 13, 12} {
			prefix := filepath.Join(base, fmt.Sprintf("bank%02d-group4-y160-body%d", bank, phase))
			var rows []struct {
				Raw string
				AX  uint16
			}
			if e := json.Unmarshal(read(prefix+".json"), &rows); e != nil || len(rows) != 1 {
				t.Fatal("原版收據", e)
			}
			raw, e := hex.DecodeString(rows[0].Raw)
			if e != nil {
				t.Fatal(e)
			}
			s, e := loadEnemy(orig, fmt.Sprintf("ENEMY%02d.PBL", bank), target)
			if e != nil {
				t.Fatal(e)
			}
			before := read(prefix + "-before.frame")
			regs := oracle.Regs{AX: rows[0].AX, CX: 0x0826, DX: 0x0304}
			s.blitWithFrame(regs, func() []byte { return raw }, func() []byte { return before })
			if !s.active || !s.inFlight {
				t.Fatal("拒絕原版四階段", bank, phase)
			}
			if bank == 8 {
				if s.h != 24 || len(s.packed) != 288 {
					t.Fatal("短圖被拉高")
				}
				changed := append([]byte(nil), raw...)
				for i := 288; i < 384; i++ {
					changed[i] ^= byte(i)
				}
				s.blitWithFrame(regs, func() []byte { return changed }, func() []byte { return before })
				if !s.active {
					t.Fatal("未知尾段影響已知身體")
				}
			}
			wrong := append([]byte(nil), raw...)
			wrong[0] ^= 1
			s.blitWithFrame(regs, func() []byte { return wrong }, func() []byte { return before })
			if s.active {
				t.Fatal("接受已知區域錯來源")
			}
		}
	}
}

func TestShortEnemyBattlePartial(t *testing.T) {
	orig, planPath, proof := os.Getenv("PSYCHICWAR_TEST_ORIG"), os.Getenv("PSYCHICWAR_ENEMY360_PLAN"), os.Getenv("PSYCHICWAR_LAST_ENEMY_PROOF")
	if orig == "" || planPath == "" || proof == "" {
		t.Skip("需短圖原版來源與獨立清單")
	}
	read := func(p string) []byte {
		v, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	var plan struct {
		Theme, Mask string
		Profiles    []struct {
			Bank, Group int
			Work        string
		}
	}
	if e := json.Unmarshal(read(planPath), &plan); e != nil {
		t.Fatal(e)
	}
	hd, _, e := LoadTheme(plan.Theme, orig, "", 3)
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range plan.Profiles {
		if p.Bank == 8 && p.Group == 4 {
			hd.battle.selectWork(read(p.Work)[:512])
		}
	}
	b := hd.battle.active
	if b == nil {
		t.Fatal("短圖來源組缺項")
	}
	mask := read(plan.Mask)
	var colors [256][3]uint8
	for i, c := range mazeColors {
		colors[i] = c
	}
	for phase := 0; phase < 4; phase++ {
		prefix := filepath.Join(proof, fmt.Sprintf("bank08-group4-y160-body%d", phase))
		var rows []struct {
			Raw string
			AX  uint16
		}
		if e := json.Unmarshal(read(prefix+".json"), &rows); e != nil {
			t.Fatal(e)
		}
		raw, e := hex.DecodeString(rows[0].Raw)
		if e != nil {
			t.Fatal(e)
		}
		native := read(prefix + "-before.frame")
		before := append([]byte(nil), b.base...)
		for y := 0; y < 32; y++ {
			copy(before[(152+y)*320+32:(152+y)*320+56], native[(152+y)*320+32:(152+y)*320+56])
		}
		target := append([]byte(nil), before...)
		for i, v := range raw {
			at := (152+i*2/24)*320 + 32 + i*2%24
			target[at] ^= v >> 4
			target[at+1] ^= v & 15
		}
		for y := 176; y < 184; y++ {
			for x := 32; x < 56; x++ {
				before[y*320+x] = byte((x + y) % 16)
				target[y*320+x] = byte((x + 3*y) % 16)
			}
		}
		b.reset()
		selectSingleAllyFixture(t, b, orig)
		b.frame(target, colors, 0x0d, mask)
		expected := append([]byte(nil), b.plane...)
		b.reset()
		selectSingleAllyFixture(t, b, orig)
		b.frame(before, colors, 0x0d, mask)
		prior := append([]byte(nil), b.plane...)
		if len(expected) != 768*120*4 || len(prior) != len(expected) {
			t.Fatal("短圖無法冷載")
		}
		regs := oracle.Regs{AX: rows[0].AX, CX: 0x0826, DX: 0x0304}
		b.blit(regs, raw, before)
		if !b.pending {
			t.Fatal("拒絕已證實身體差分")
		}
		for _, count := range []int{0, 1, 7, 8, 15, 16, 23, 24, 31, 32} {
			mid := append([]byte(nil), before...)
			for y := 0; y < count; y++ {
				copy(mid[(152+y)*320+32:(152+y)*320+56], target[(152+y)*320+32:(152+y)*320+56])
			}
			selectSingleAllyFixture(t, b, orig)
			b.frame(mid, colors, 0x0d, mask)
			if b.vector(mid).Cmp(b.vector(before)) == 0 {
				if !bytes.Equal(b.plane, prior) {
					t.Fatal("尚未改圖時顯示不同")
				}
				continue
			}
			for cy := 0; cy < 5; cy++ {
				for cx := 0; cx < 32; cx++ {
					valid := !b.ignoredPixel(32+cx*8, 144+cy*8)
					for y := 0; y < 8; y++ {
						at := (144+cy*8+y)*320 + 32 + cx*8
						valid = valid && bytes.Equal(mid[at:at+8], target[at:at+8])
					}
					for y := 0; y < 24; y++ {
						at := ((cy*24+y)*768 + cx*24) * 4
						want := expected[at : at+96]
						if !valid {
							want = make([]byte, 96)
						}
						if !bytes.Equal(b.plane[at:at+96], want) {
							t.Fatal("短圖中途格不同", phase, count, cx, cy)
						}
					}
				}
			}
		}
		wrong := append([]byte(nil), raw...)
		wrong[0] ^= 1
		b.blit(regs, wrong, before)
		if b.pending {
			t.Fatal("接受已知區外錯差分")
		}
	}
}

func TestEnemyAliasPNGGuard(t *testing.T) {
	orig, planPath := os.Getenv("PSYCHICWAR_TEST_ORIG"), os.Getenv("PSYCHICWAR_ENEMY360_PLAN")
	if orig == "" || planPath == "" {
		t.Skip("需360定稿主題")
	}
	var plan struct{ Theme string }
	b, e := os.ReadFile(planPath)
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, &plan); e != nil {
		t.Fatal(e)
	}
	b, e = os.ReadFile(filepath.Join(plan.Theme, "manifest.json"))
	if e != nil {
		t.Fatal(e)
	}
	var m ThemeManifest
	if e = json.Unmarshal(b, &m); e != nil {
		t.Fatal(e)
	}
	dir := t.TempDir()
	files, e := os.ReadDir(plan.Theme)
	if e != nil {
		t.Fatal(e)
	}
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".png" {
			p := filepath.Join(dir, f.Name())
			data, e := os.ReadFile(filepath.Join(plan.Theme, f.Name()))
			if e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(p, data, 0600); e != nil {
				t.Fatal(e)
			}
		}
	}
	write := func() {
		data, e := json.Marshal(m)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	for i := range m.Entries {
		if m.Entries[i].PBL == "ENEMY02.PBL" && m.Entries[i].Image == 13 {
			m.Entries[i].PNG = "ENEMY02-14.png"
		}
	}
	write()
	if _, _, e = LoadTheme(dir, orig, "", 3); e == nil {
		t.Fatal("接受不同HD姿勢的原版別名")
	}
	for i := range m.Entries {
		if m.Entries[i].PBL == "ENEMY02.PBL" && m.Entries[i].Image == 13 {
			m.Entries[i].PNG = "ENEMY02-13.png"
		}
	}
	write()
	hd, _, e := LoadTheme(dir, orig, "", 3)
	if e != nil {
		t.Fatal(e)
	}
	if hd.battle == nil || len(hd.battle.profiles) != 60 {
		t.Fatal("來源組缺項")
	}
}
