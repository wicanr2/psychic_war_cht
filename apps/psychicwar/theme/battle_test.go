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

	"github.com/wicanr2/dosgolem/oracle"
)

func TestBuiltinMaskRequiredFields(t *testing.T) {
	valid := map[string]any{"id": "battle-mask-4e36", "at": []int{32, 152}, "png": "mask.png", "kind": "redraw", "match": []int{248, 0, 72, 40}}
	for _, key := range []string{"id", "at", "png", "kind", "match"} {
		for _, null := range []bool{false, true} {
			copy := map[string]any{}
			for k, v := range valid {
				copy[k] = v
			}
			if null {
				copy[key] = nil
			} else {
				delete(copy, key)
			}
			b, _ := json.Marshal(copy)
			var e BuiltinMaskEntry
			if json.Unmarshal(b, &e) == nil {
				t.Fatalf("接受缺少／null %s", key)
			}
		}
	}
	valid["offset"] = 0x4e36
	b, _ := json.Marshal(valid)
	var e BuiltinMaskEntry
	if json.Unmarshal(b, &e) == nil {
		t.Fatal("接受任意位址欄位")
	}
}

func TestMintonProductionScene(t *testing.T) {
	path, orig := os.Getenv("PSYCHICWAR_MINTON_PLAN"), os.Getenv("PSYCHICWAR_TEST_ORIG")
	if path == "" || orig == "" {
		t.Skip("需明示獨立敏頓原版來源及場景清單")
	}
	read := func(p string) []byte {
		v, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	var plan struct {
		Schema, Theme string
		Normal        string `json:"normal_trace"`
		Mask          string `json:"mask_source"`
		Variables     []struct {
			Name  string
			Image int
			Rect  [4]int
		}
		Samples []struct {
			Kind   string
			Event  int
			Side   string
			Step   uint64
			SHA    string `json:"frame_sha256"`
			Active []int  `json:"active_variables"`
		}
		Renders []struct {
			Sample int
			Step   uint64
			Source string `json:"source_frame"`
		} `json:"render_samples"`
	}
	if e := json.Unmarshal(read(path), &plan); e != nil {
		t.Fatal(e)
	}
	if plan.Schema != "psychic-war-minton-production-plan/1" || len(plan.Variables) != 172 || len(plan.Samples) != 2238 && len(plan.Samples) != 1928 {
		t.Fatal("獨立範圍不符")
	}
	selection := plan.Theme
	if override := os.Getenv("PSYCHICWAR_MINTON_THEME"); override != "" {
		selection = override
	}
	hd, notice, err := LoadTheme(selection, orig, "", 3)
	if err != nil || hd == nil || notice != "" || hd.battle == nil {
		t.Fatal("正式效果載入", notice, err)
	}
	b := hd.battle
	extendedEffects := b.beamProfiles != nil
	variables := 172
	if b.profiles != nil {
		var found *battleTheme
		for _, profile := range b.profiles {
			for _, a := range profile.assets {
				if a.name == "ENEMY00.PBL" && a.image == 6 && a.body {
					if found != nil {
						t.Fatal("敏頓來源組不唯一")
					}
					found = profile
				}
			}
		}
		if found == nil {
			t.Fatal("敏頓來源組缺失")
		}
		b = found
		variables = 217
		if extendedEffects {
			variables = 255 // §1.55：舊172原版真值映射保持；新增38個已驗FIGHT原位。
		}
	}
	selectSingleAllyFixture(t, b, orig)
	mask := read(plan.Mask)
	if !b.prepare(mask) || len(b.assets) != variables || len(b.basis) != variables {
		t.Fatal("原版遮罩／滿秩來源不符")
	}
	key := func(name string, n int, r [4]int) string { return fmt.Sprintf("%s:%d:%v", name, n, r) }
	index := map[string]int{}
	for n, a := range b.assets {
		index[key(a.name, a.image, a.rect)] = n
	}
	truth := func(active []int) *big.Int {
		v := new(big.Int)
		for _, n := range active {
			a := plan.Variables[n]
			name := a.Name
			if name != "MASK" {
				name += ".PBL"
			}
			j, ok := index[key(name, a.Image, a.Rect)]
			if !ok {
				t.Fatal("來源對應缺項", a)
			}
			v.SetBit(v, j, 1)
		}
		return v
	}
	type event struct {
		Entry                 uint64 `json:"entry_step"`
		Return                uint64 `json:"return_step"`
		Before, After, Source string
		Regs                  map[string]uint16 `json:"entry_regs"`
		Rect                  [4]int
		AL                    byte `json:"al"`
	}
	var trace struct {
		Events []event
		Masks  []event `json:"mask_events"`
	}
	if err = json.Unmarshal(read(plan.Normal), &trace); err != nil {
		t.Fatal(err)
	}
	if !(len(trace.Events) == 1009 && len(trace.Masks) == 110 && len(plan.Samples) == 2238 || len(trace.Events) == 874 && len(trace.Masks) == 90 && len(plan.Samples) == 1928) {
		t.Fatal("原版事件範圍不符")
	}
	digest := func(v []byte) string { return fmt.Sprintf("%x", sha256.Sum256(v)) }
	for _, s := range plan.Samples {
		var e event
		if s.Kind == "MASK" {
			e = trace.Masks[s.Event]
		} else {
			e = trace.Events[s.Event]
		}
		p := e.After
		if s.Side == "before" {
			p = e.Before
		}
		frame := read(p)
		if digest(frame) != s.SHA {
			t.Fatal("原版保存畫面SHA不符")
		}
		want := truth(s.Active)
		got := b.solve(frame)
		if got == nil || got.Cmp(want) != 0 || battleVector(b.model(got)).Cmp(battleVector(frame)) != 0 {
			t.Fatalf("完整冷載／原始素材模型不同 step%d", s.Step)
		}
	}
	checkEvent := func(e event, isMask bool) {
		before, after, raw := read(e.Before), read(e.After), read(e.Source)
		want := b.solve(after)
		if want == nil {
			t.Fatal("原版after不可解")
		}
		r := oracle.Regs{AX: e.Regs["AX"], BX: e.Regs["BX"], CX: e.Regs["CX"], DX: e.Regs["DX"], DS: e.Regs["DS"]}
		if isMask {
			b.mask(r, raw, before)
		} else {
			b.blit(r, raw, before)
		}
		if !b.pending || b.value == nil || b.value.Cmp(want) != 0 {
			t.Fatalf("來源轉換／前姿勢不同 step%d mask%v", e.Entry, isMask)
		}
		bad := append([]byte(nil), raw...)
		bad[0] ^= 1
		if isMask {
			b.mask(r, bad, before)
		} else {
			b.blit(r, bad, before)
		}
		if b.value != nil || b.pending {
			t.Fatal("錯來源仍認定效果")
		}
		b.finish()
	}
	for _, e := range trace.Events {
		checkEvent(e, false)
	}
	for _, e := range trace.Masks {
		checkEvent(e, true)
	}
	var colors [256][3]uint8
	for n, c := range mazeColors {
		colors[n] = c
	}
	// 原版來源的前後frame間只改第一個變動像素，驗中途預測的全域8×8回退。
	partialChecks := 0
	for _, e := range append(append([]event(nil), trace.Events...), trace.Masks...) {
		if partialChecks >= 12 {
			break
		}
		before, after := read(e.Before), read(e.After)
		r := oracle.Regs{AX: e.Regs["AX"], BX: e.Regs["BX"], CX: e.Regs["CX"], DX: e.Regs["DX"], DS: e.Regs["DS"]}
		isMask := r.DS == 0x161 && r.DX == 0x4e36
		if isMask {
			b.mask(r, read(e.Source), before)
		} else {
			b.blit(r, read(e.Source), before)
		}
		partial := append([]byte(nil), before...)
		changed := false
		for i := range partial {
			if partial[i] != after[i] {
				partial[i] = after[i]
				changed = true
				break
			}
		}
		if !changed {
			continue
		}
		selectSingleAllyFixture(t, b, orig)
		b.frame(partial, colors, 0x0d, mask)
		if b.plane == nil {
			t.Fatal("已知中途整層消失")
		}
		expected := b.model(b.value)
		blocked, visible := 0, 0
		for cy := 0; cy < 5; cy++ {
			for cx := 0; cx < 32; cx++ {
				match := true
				for y := 0; y < 8; y++ {
					i := (144+cy*8+y)*320 + 32 + cx*8
					if !bytes.Equal(partial[i:i+8], expected[i:i+8]) {
						match = false
					}
				}
				for y := 0; y < 24; y++ {
					for x := 0; x < 24; x++ {
						alpha := b.plane[((cy*24+y)*768+cx*24+x)*4+3]
						if match && alpha != 255 || !match && alpha != 0 {
							t.Fatal("中途未按全域8×8回退")
						}
					}
				}
				if match {
					visible++
				} else {
					blocked++
				}
			}
		}
		if blocked == 0 || visible == 0 {
			t.Fatal("中途正／負對照範圍不符")
		}
		partialChecks++
		b.finish()
		selectSingleAllyFixture(t, b, orig)
		b.frame(after, colors, 0x0d, mask)
		if b.plane == nil {
			t.Fatal("完成後HD未恢復")
		}
	}
	if partialChecks != 12 {
		t.Fatal("中途樣本不足")
	}
	output := os.Getenv("PSYCHICWAR_MINTON_OUT")
	rows := []map[string]any{}
	for _, r := range plan.Renders {
		frame := read(r.Source)
		selectSingleAllyFixture(t, b, orig)
		b.frame(frame, colors, 0x0d, mask)
		if len(b.plane) == 0 {
			t.Fatal("完整效果畫面被拒絕")
		}
		pixels := append([]byte(nil), b.plane...)
		hd.ResetForLoad()
		selectSingleAllyFixture(t, b, orig)
		b.frame(frame, colors, 0x0d, mask)
		if !bytes.Equal(pixels, b.plane) {
			t.Fatal("實際ResetForLoad冷載不同")
		}
		all := make([]byte, 960*600*4)
		hd.Layer.Frame(frame, make([]byte, 64000*3))
		hd.Draw(all, 3)
		hd.Enabled = false
		if hd.Draw(make([]byte, len(all)), 3) {
			t.Fatal("關閉仍畫HD")
		}
		hd.Enabled = true
		if output != "" {
			p := filepath.Join(output, fmt.Sprintf("sample%02d.png", r.Sample))
			f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
			if e != nil {
				t.Fatal(e)
			}
			e = png.Encode(f, &image.NRGBA{Pix: pixels, Stride: 768 * 4, Rect: image.Rect(0, 0, 768, 120)})
			if e != nil {
				t.Fatal(e)
			}
			if e = f.Close(); e != nil {
				t.Fatal(e)
			}
			rows = append(rows, map[string]any{"sample": r.Sample, "step": r.Step, "source_frame": r.Source, "png": p, "RGBA_sha256": digest(pixels)})
		}
	}
	// 完整冷載不能把未知像素或未知身體猜成同一場景。
	frame := read(plan.Renders[0].Source)
	bad := append([]byte(nil), frame...)
	bad[180*320+280] ^= 1
	if b.solve(bad) != nil {
		t.Fatal("未知像素未拒絕")
	}
	wrongMask := append([]byte(nil), mask...)
	wrongMask[0] ^= 1
	selectSingleAllyFixture(t, b, orig)
	b.frame(frame, colors, 0x0d, wrongMask)
	if b.plane != nil || b.value != nil {
		t.Fatal("錯遮罩來源未回退")
	}
	selectSingleAllyFixture(t, b, orig)
	b.frame(frame, colors, 0x0d, mask)
	if b.plane == nil {
		t.Fatal("原版遮罩恢復未回來")
	}
	noAnchor := append([]byte(nil), frame...)
	noAnchor[248] ^= 1
	selectSingleAllyFixture(t, b, orig)
	b.frame(noAnchor, colors, 0x0d, mask)
	if b.plane != nil {
		t.Fatal("錨點錯誤仍畫效果")
	}
	if output != "" {
		doc := map[string]any{"status": "PASS_MINTON_PRODUCTION_COLD_AND_SOURCE", "full_boundaries": len(plan.Samples), "source_transitions": len(trace.Events) + len(trace.Masks), "wrong_source_negatives": len(trace.Events) + len(trace.Masks), "variables": len(b.assets), "synthetic_partial_checks": partialChecks, "render_rows": rows, "limits": "Stored original normal boundaries and independent truth; partial checks are synthetic. Fresh GUI and actual mid-blit/DAT require separate receipts."}
		v, e := json.MarshalIndent(doc, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(output, "render.json"), append(v, '\n'), 0644); e != nil {
			t.Fatal(e)
		}
	}
}
