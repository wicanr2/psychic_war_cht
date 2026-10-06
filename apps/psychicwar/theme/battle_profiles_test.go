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

// 期望來源、原位、原版frame及活躍圖號由獨立Python／原版受控工作區給定。
func TestEnemySmallProfiles(t *testing.T) {
	path, orig := os.Getenv("PSYCHICWAR_SMALL_PLAN"), os.Getenv("PSYCHICWAR_TEST_ORIG")
	if path == "" || orig == "" {
		t.Skip("需明示58組獨立來源及合成清單")
	}
	read := func(p string) []byte {
		v, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	var plan struct {
		Schema, Theme, Mask string
		Profiles            []struct {
			Bank, Group int
			Rank        int
			Selector    string `json:"selector_sha256"`
			Work        string
			Variables   []struct {
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
	if e := json.Unmarshal(read(path), &plan); e != nil {
		t.Fatal(e)
	}
	if plan.Schema != "psychic-war-small-profile-plan/1" || (len(plan.Profiles) != 58 && len(plan.Profiles) != 60) {
		t.Fatal("獨立範圍不符")
	}
	hd, notice, e := LoadTheme(plan.Theme, orig, "", 3)
	if e != nil || notice != "" || hd == nil || hd.battle == nil {
		t.Fatal("批次小圖載入", notice, e)
	}
	root := hd.battle
	if len(root.profiles) != len(plan.Profiles) {
		t.Fatal("來源組缺項")
	}
	mask := read(plan.Mask)
	var colors [256][3]uint8
	for i, c := range mazeColors {
		colors[i] = c
	}
	output := os.Getenv("PSYCHICWAR_SMALL_OUT")
	renders := []map[string]any{}
	for _, p := range plan.Profiles {
		raw := read(p.Work)
		root.selectWork(raw[:512])
		b := root.active
		key, ok := battleSourceKey(raw[:512])
		rank := p.Rank
		if rank == 0 {
			rank = len(p.Variables)
		}
		if !ok || fmt.Sprintf("%x", key) != p.Selector || b == nil || !b.prepare(mask) || len(b.assets) != len(p.Variables) || len(b.basis) != rank {
			t.Fatalf("來源／滿秩不同 bank%d group%d", p.Bank, p.Group)
		}
		index := map[string]int{}
		for i, a := range b.assets {
			index[fmt.Sprintf("%s:%d:%v", a.name, a.image, a.rect)] = i
		}
		mapped := make([]int, len(p.Variables))
		for i, v := range p.Variables {
			j, ok := index[fmt.Sprintf("%s:%d:%v", v.Name, v.Image, v.Rect)]
			if !ok {
				t.Fatal("獨立來源／原位缺項", v)
			}
			mapped[i] = j
			a := b.assets[j]
			if fmt.Sprintf("%x", sha256.Sum256(a.packed)) != v.SHA {
				t.Fatal("原版來源不同", v)
			}
			value := new(big.Int).SetBit(new(big.Int), j, 1)
			got := b.solve(b.model(value))
			if v.Hidden {
				value.SetBit(value, j, 0)
			}
			if got == nil || got.Cmp(value) != 0 {
				t.Fatal("已核對單源無法冷解", v)
			}
		}
		for n, c := range p.Cases {
			frame := read(c.Frame)
			if fmt.Sprintf("%x", sha256.Sum256(frame)) != c.SHA256 {
				t.Fatal("獨立原版frame雜湊不同")
			}
			want := new(big.Int)
			for _, i := range c.Active {
				want.SetBit(want, mapped[i], 1)
			}
			got := b.solve(frame)
			if got == nil || got.Cmp(want) != 0 || b.vector(b.model(got)).Cmp(b.vector(frame)) != 0 {
				t.Fatal("独立冷解不同", p.Bank, p.Group, n)
			}
			root.frame(frame, colors, 0x0d, mask)
			if len(b.plane) != 768*120*4 {
				t.Fatal("B合成缺失")
			}
			pixels := append([]byte(nil), b.plane...)
			hd.ResetForLoad()
			if root.active != nil {
				t.Fatal("讀檔保留舊來源組")
			}
			root.selectWork(raw[:512])
			root.frame(frame, colors, 0x0d, mask)
			if !bytes.Equal(pixels, b.plane) {
				t.Fatal("切組／讀檔冷載不同")
			}
			dst := make([]byte, 960*600*4)
			hd.Enabled = false
			if hd.Draw(dst, 3) {
				t.Fatal("HD關閉仍繪製")
			}
			hd.Enabled = true
			if n == 2 && output != "" {
				file := filepath.Join(output, fmt.Sprintf("bank%02d-group%d.png", p.Bank, p.Group))
				f, e := os.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
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
				renders = append(renders, map[string]any{"bank": p.Bank, "group": p.Group, "case": n, "png": file, "RGBA_sha256": fmt.Sprintf("%x", sha256.Sum256(pixels))})
			}
			wrong := append([]byte(nil), raw[:512]...)
			wrong[0] ^= 1
			root.selectWork(wrong)
			if root.active != nil || root.draw(make([]byte, len(dst)), 3) {
				t.Fatal("錯工作源仍繪製")
			}
			root.selectWork(raw[:512])
		}
	}
	if output != "" {
		v, e := json.MarshalIndent(map[string]any{"status": "PASS_ENEMY_PROFILES_LIMITED", "profiles": len(plan.Profiles), "synthetic_scenes": len(plan.Profiles) * 3, "wrong_workspace_negatives": len(plan.Profiles) * 3, "renders": renders, "limits": "Independent source models and synthetic scenes. Normal GUI / actualmid-blit / DAT separate."}, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(output, "render.json"), append(v, '\n'), 0644); e != nil {
			t.Fatal(e)
		}
	}
}
