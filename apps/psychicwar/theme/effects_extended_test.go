package theme

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math/big"
	"os"
	"path/filepath"
	"testing"
)

// 正對照的位置由原版600輸出清單提供；此處另驗受控但未核准的位置拒絕。
func TestExtendedEffectPositionsRejectUnsupported(t *testing.T) {
	for _, p := range []struct {
		name    string
		n, x, y int
	}{
		{"BEAM.PBL", 3, 8, 160}, {"BEAM.PBL", 3, 24, 160}, {"BEAM.PBL", 3, 264, 160},
		{"BEAM.PBL", 12, 40, 160}, {"BEAM.PBL", -1, 40, 160},
		{"FIGHT.PBL", 4, 144, 168}, {"FIGHT.PBL", 5, 144, 152},
		{"FIGHT.PBL", 0, 144, 144}, {"FIGHT.PBL", 2, 160, 144},
		{"FIGHT.PBL", 11, 240, 160}, {"FIGHT.PBL", 12, 240, 168},
	} {
		if battlePosition(p.name, p.n, p.x, p.y) {
			t.Fatal("接受未READY位置", p)
		}
	}
}

func TestExtendedEffectProfilesIndependent(t *testing.T) {
	planPath, orig, out := os.Getenv("PSYCHICWAR_EFFECTS_PLAN"), os.Getenv("PSYCHICWAR_TEST_ORIG"), os.Getenv("PSYCHICWAR_EFFECTS_OUT")
	if planPath == "" || orig == "" || out == "" {
		t.Skip("需明示240組獨立原始來源、場景與輸出")
	}
	read := func(p string) []byte {
		v, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	var plan struct {
		Schema, Theme, Mask string
		Profiles            []struct {
			Bank, Group, Beam, Rank int
			Work, BeamWork          string
			Variables               []struct {
				Name   string
				Image  int
				Rect   [4]int
				SHA    string `json:"source_sha256"`
				Hidden bool
			}
			Cases []struct {
				Frame, SHA256 string
				Active        []int
			}
		}
	}
	if err := json.Unmarshal(read(planPath), &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Schema != "psychic-war-full-effects-plan/1" || len(plan.Profiles) != 240 {
		t.Fatal("獨立範圍不符")
	}
	hd, notice, err := LoadTheme(plan.Theme, orig, "", 3)
	if err != nil || notice != "" || hd == nil || hd.battle == nil {
		t.Fatal("完整效果載入", notice, err)
	}
	root := hd.battle
	if len(root.profiles) != 60 || len(root.beamProfiles) != 4 || len(root.allProfiles()) != 240 {
		t.Fatal("工作源組缺項")
	}
	mask := read(plan.Mask)
	var palette [256][3]uint8
	for i, c := range mazeColors {
		palette[i] = c
	}
	renders := []map[string]any{}
	for _, p := range plan.Profiles {
		work, beam := read(p.Work)[:512], read(p.BeamWork)[:512]
		root.selectWorkWithBeam(work, beam)
		b := root.active
		if b == nil {
			t.Fatal("原版工作源未選入", p.Bank, p.Group, p.Beam)
		}
		selectSingleAllyFixture(t, root, orig)
		if !b.prepare(mask) || len(b.assets) != len(p.Variables) || len(b.basis) != p.Rank {
			t.Fatal("獨立滿秩或變數不符", p.Bank, p.Group, p.Beam)
		}
		indices := map[string]int{}
		for n, a := range b.assets {
			indices[fmt.Sprintf("%s:%d:%v", a.name, a.image, a.rect)] = n
		}
		mapped := make([]int, len(p.Variables))
		for n, v := range p.Variables {
			index, ok := indices[fmt.Sprintf("%s:%d:%v", v.Name, v.Image, v.Rect)]
			if !ok {
				t.Fatal("原版來源／原位缺項", v)
			}
			mapped[n] = index
			if fmt.Sprintf("%x", sha256.Sum256(b.assets[index].packed)) != v.SHA {
				t.Fatal("原始來源不同", v)
			}
		}
		for n, c := range p.Cases {
			frame := read(c.Frame)
			if fmt.Sprintf("%x", sha256.Sum256(frame)) != c.SHA256 {
				t.Fatal("獨立frame不同")
			}
			want := new(big.Int)
			for _, index := range c.Active {
				want.SetBit(want, mapped[index], 1)
			}
			value := b.solve(frame)
			if value == nil || value.Cmp(want) != 0 {
				t.Fatal("獨立原版像素冷解不同", p.Bank, p.Group, p.Beam, n)
			}
			root.frame(frame, palette, 0x0d, mask)
			if len(b.plane) != 768*120*4 {
				t.Fatal("B完整圖面缺失")
			}
			before := append([]byte(nil), b.plane...)
			hd.ResetForLoad()
			if root.active != nil {
				t.Fatal("讀檔保留來源")
			}
			for _, other := range root.allProfiles() {
				if other.maskReady || other.value != nil || other.plane != nil {
					t.Fatal("讀檔遺留其他BEAM組")
				}
			}
			root.selectWorkWithBeam(work, beam)
			selectSingleAllyFixture(t, root, orig)
			root.frame(frame, palette, 0x0d, mask)
			if !bytes.Equal(before, b.plane) {
				t.Fatal("讀檔重建RGBA不同")
			}
			wrong := append([]byte(nil), beam...)
			wrong[0] ^= 1
			root.selectWorkWithBeam(work, wrong)
			if root.active != nil || len(b.plane) != 0 {
				t.Fatal("錯BEAM來源保留舊圖面")
			}
			root.selectWorkWithBeam(work, beam)
			selectSingleAllyFixture(t, root, orig)
			root.frame(frame, palette, 0x0d, mask)
			cga := append([]byte(nil), beam...)
			cga[128] ^= 1
			root.selectWorkWithBeam(work, cga)
			if root.active != b {
				t.Fatal("CGA尾段誤改EGA身份")
			}
			{
				path := filepath.Join(out, fmt.Sprintf("bank%02d-group%d-beam%d-case%d.png", p.Bank, p.Group, p.Beam, n))
				f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(f, &image.NRGBA{Pix: b.plane, Stride: 768 * 4, Rect: image.Rect(0, 0, 768, 120)})
				if err != nil {
					t.Fatal(err)
				}
				if err = f.Close(); err != nil {
					t.Fatal(err)
				}
				renders = append(renders, map[string]any{"bank": p.Bank, "group": p.Group, "beam": p.Beam, "case": n, "png": path, "RGBA_sha256": fmt.Sprintf("%x", sha256.Sum256(b.plane))})
			}
		}
	}
	raw, err := json.MarshalIndent(map[string]any{"status": "PASS_FULL_EFFECTS_CONTROLLED_SOURCE_MODELS", "profiles": 240, "synthetic_scenes": 720, "renders": renders, "limits": "原始來源、合成場景、冷解與讀檔重建；正常GUI、DAT及完整HD另驗。"}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(out, "render.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
