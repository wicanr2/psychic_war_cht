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

// 舊場景fixture的已確認基底：ALLY #0、槽0、driver2。
// 不用這個受控設定宣稱正常玩家來源；實際Oracle讀取另由正常保存點驗證。
func selectSingleAllyFixture(t *testing.T, b *battleTheme, orig string) {
	t.Helper()
	if b.profiles != nil {
		b = b.active
	}
	if b == nil || b.party == nil {
		return
	}
	ally, err := loadAlly0(orig)
	if err != nil {
		t.Fatal(err)
	}
	b.selectPartyData([4][]byte{ally.packed}, 0, 2)
	if !b.partyValid {
		t.Fatal("fixture已确认ALLY0基底未選入")
	}
}

// 原版RAM合成及實際COPY輸出由外部探針提供；完整RGBA另由Python核對。
func TestAllyEquipmentBatch(t *testing.T) {
	orig, proofPath, dir, out := os.Getenv("PSYCHICWAR_TEST_ORIG"), os.Getenv("PSYCHICWAR_ALLY31_PROOF"), os.Getenv("PSYCHICWAR_ALLY31_THEME"), os.Getenv("PSYCHICWAR_ALLY31_OUT")
	if orig == "" || proofPath == "" || dir == "" || out == "" {
		t.Skip("需192組原版装備合成與48人物原位")
	}
	read := func(p string) []byte {
		v, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	var proof struct {
		Rows []struct {
			Actor, Equipment, Slot int
			Prefix                 string
			Rect                   [4]int
		}
	}
	if e := json.Unmarshal(read(proofPath), &proof); e != nil {
		t.Fatal(e)
	}
	hd, _, e := LoadTheme(dir, orig, "", 3)
	if e != nil {
		t.Fatal(e)
	}
	old, _, e := LoadTheme(filepath.Join(filepath.Dir(dir), "theme-enemy360-ally27-effects-maze-B-v1-20261006"), orig, "", 3)
	if e != nil {
		t.Fatal(e)
	}
	if hd.party == nil || len(hd.party.assets) != 60 {
		t.Fatal("缺唯一合成身份")
	}
	render := func(h *Theme, frame []byte) []byte {
		h.frameSprites(frame)
		rgb := make([]byte, len(frame)*3)
		for i, v := range frame {
			c := mazeColors[v]
			copy(rgb[i*3:i*3+3], c[:])
		}
		h.Layer.Frame(frame, rgb)
		p := make([]byte, 960*600*4)
		h.Draw(p, 3)
		return p
	}
	save := func(path string, p []byte, w, h int) {
		f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if e != nil {
			t.Fatal(e)
		}
		if e = png.Encode(f, &image.NRGBA{Pix: p, Stride: w * 4, Rect: image.Rect(0, 0, w, h)}); e != nil {
			t.Fatal(e)
		}
		if e = f.Close(); e != nil {
			t.Fatal(e)
		}
	}
	rows := []map[string]any{}
	bodies := map[[2]int]bool{}
	for _, r := range proof.Rows {
		for _, phase := range []string{"body", "after"} {
			if phase == "body" && bodies[[2]int{r.Actor, r.Slot}] {
				continue
			}
			bodies[[2]int{r.Actor, r.Slot}] = true
			frame := read(r.Prefix + "-" + phase + ".frame")
			hd.ResetForLoad()
			old.ResetForLoad()
			actual, prior := render(hd, frame), render(old, frame)
			hd.ResetForLoad()
			if !bytes.Equal(render(hd, frame), actual) {
				t.Fatal("冷載不同", r.Actor, r.Equipment, r.Slot, phase)
			}
			if phase == "after" && bytes.Equal(actual, prior) {
				t.Fatal("裝備未繪製", r.Actor, r.Equipment, r.Slot)
			}
			hd.Enabled = false
			p := make([]byte, len(actual))
			if hd.Draw(p, 3) || !bytes.Equal(p, make([]byte, len(p))) {
				t.Fatal("關閉仍畫")
			}
			hd.Enabled = true
			path := filepath.Join(out, fmt.Sprintf("actor%02d-equipment%02d-slot%d-%s.png", r.Actor, r.Equipment, r.Slot, phase))
			oldPath := path + "-old.png"
			save(path, actual, 960, 600)
			save(oldPath, prior, 960, 600)
			rows = append(rows, map[string]any{"actor": r.Actor, "equipment": r.Equipment, "slot": r.Slot, "phase": phase, "frame": r.Prefix + "-" + phase + ".frame", "actual": path, "old": oldPath})
			wrong := append([]byte(nil), frame...)
			wrong[248] ^= 1
			render(hd, wrong)
			for _, g := range hd.groups {
				if g.sprite != nil && g.sprite.active && allyPartySlot([]int{g.sprite.x, g.sprite.y}) == r.Slot {
					t.Fatal("錨點失配仍有隊伍圖面")
				}
			}
		}
		for _, bad := range []ThemeEntry{{PBL: "ALLY.PBL", Image: r.Equipment, At: []int{r.Rect[0] + 4, 152}, Kind: "redraw", Match: []int{248, 0, 72, 40}}, {PBL: "ALLY.PBL", Image: r.Equipment, At: r.Rect[:2], Kind: "redraw"}, {PBL: "ALLY.PBL", Image: r.Equipment, At: r.Rect[:2], Kind: "original", Match: []int{248, 0, 72, 40}}} {
			if _, _, e := allyPosition(bad); e == nil {
				t.Fatal("接受未READY裝備")
			}
		}
	}
	if len(rows) != 240 || len(bodies) != 48 {
		t.Fatal("缺原版COPY")
	}
	// 已知敵人來源、原版compound與已證實效果，抽測B在武器上方及冷載。
	var plan struct {
		Mask     string
		Profiles []struct {
			Bank, Group int
			Work        string
			Cases       []struct {
				Frame  string
				Active []int
			}
			Variables []struct {
				Name  string
				Image int
				Rect  [4]int
			}
		}
	}
	if e := json.Unmarshal(read(os.Getenv("PSYCHICWAR_ENEMY360_PLAN")), &plan); e != nil {
		t.Fatal(e)
	}
	var colors [256][3]uint8
	for i, c := range mazeColors {
		colors[i] = c
	}
	mask := read(plan.Mask)
	effects := []map[string]any{}
	tested := 0
	for _, profile := range plan.Profiles {
		if profile.Bank != 0 || profile.Group != 2 {
			continue
		}
		work := read(profile.Work)[:512]
		for _, r := range proof.Rows {
			if r.Slot != 0 {
				continue
			}
			packed := read(r.Prefix + "-after.packed")
			for sample := 0; sample < 3; sample++ {
				hd.ResetForLoad()
				hd.battle.selectWork(work)
				b := hd.battle.active
				b.selectPartyData([4][]byte{packed}, 0, 2)
				if !b.partyValid || !b.prepare(mask) {
					t.Fatal("已確認隊伍未選入")
				}
				frame := read(profile.Cases[sample].Frame)
				// fixture原版基底為ALLY0；用原版合成替換該位置，既有XOR保持。
				base0, e := loadAlly0(orig)
				if e != nil {
					t.Fatal(e)
				}
				a := hd.party.assets[sha256.Sum256(packed)]
				for y := 0; y < 32; y++ {
					for x := 0; x < 24; x++ {
						i := (152+y)*320 + 264 + x
						frame[i] ^= base0.indexed[y*24+x] ^ a.indexed[y*24+x]
					}
				}
				b.frame(frame, colors, 0x0d, mask)
				if len(b.plane) != 768*120*4 || b.solve(frame) == nil {
					t.Fatal("裝備B場景無法冷解", r.Actor, r.Equipment, sample)
				}
				value := new(big.Int).Set(b.value)
				pixels := append([]byte(nil), b.plane...)
				b.selectPartyData([4][]byte{packed}, 0, 1)
				b.frame(frame, colors, 0x0d, mask)
				if b.plane != nil {
					t.Fatal("接受未知driver")
				}
				wrong := append([]byte(nil), packed...)
				wrong[0] ^= 1
				b.selectPartyData([4][]byte{wrong}, 0, 2)
				b.frame(frame, colors, 0x0d, mask)
				if b.plane != nil {
					t.Fatal("接受未知隊伍工作源")
				}
				b.selectPartyData([4][]byte{packed}, 4, 2)
				if b.partyValid {
					t.Fatal("接受未知人數")
				}
				hd.ResetForLoad()
				hd.battle.selectWork(work)
				b.selectPartyData([4][]byte{packed}, 0, 2)
				b.frame(frame, colors, 0x0d, mask)
				if !bytes.Equal(b.plane, pixels) || b.value.Cmp(value) != 0 {
					t.Fatal("裝備B冷載不同")
				}
				tested++
				if sample == 2 {
					path := filepath.Join(out, fmt.Sprintf("effect-actor%02d-equipment%02d.png", r.Actor, r.Equipment))
					save(path, pixels, 768, 120)
					framePath := path + ".frame"
					if e = os.WriteFile(framePath, frame, 0644); e != nil {
						t.Fatal(e)
					}
					effects = append(effects, map[string]any{"actor": r.Actor, "equipment": r.Equipment, "actual": path, "frame": framePath, "active": profile.Cases[sample].Active, "variables": profile.Variables})
				}
			}
		}
	}
	if tested != 144 {
		t.Fatal("缺裝備B樣本")
	}
	v, e := json.MarshalIndent(map[string]any{"status": "PASS_192_EQUIPMENT_48_BODY_COPY_144_B_SCENES", "renders": rows, "effects": effects, "limits": "Controlled original source and synthetic B scenes. Normal selection/GUI/DAT/fullHD pending."}, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(out, "render.json"), append(v, '\n'), 0644); e != nil {
		t.Fatal(e)
	}
}
