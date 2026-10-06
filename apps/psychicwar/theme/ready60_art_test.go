package theme

// 研究038 §162：獨立原版PBL與候選PNG期望，不把合成畫面稱為正常玩家證據。
import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestReady60ArtBatch(t *testing.T) {
	path := os.Getenv("PSYCHICWAR_READY60_ART_PLAN")
	orig := os.Getenv("PSYCHICWAR_TEST_ORIG")
	if path == "" || orig == "" {
		t.Skip("需明示60張獨立素材期望與原版")
	}
	var plan struct {
		Schema, Theme string
		BodyTheme     string `json:"body_theme"`
		Rows          []struct {
			PBL      string
			Image    int
			Frame    string
			FrameSHA string `json:"frame_sha256"`
			Expected string `json:"expected_sha256"`
			Covered  string `json:"covered_sha256"`
			Cover    []int  `json:"cover_at"`
			Cells    int    `json:"valid_cells"`
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &plan); err != nil {
		t.Fatal(err)
	}
	wantCount := 60
	if plan.Schema == "psychic-war-ready174-art-fixtures/1" {
		wantCount = 174
	} else if plan.Schema != "psychic-war-ready60-art-fixtures/1" {
		t.Fatal("期望格式不符")
	}
	if len(plan.Rows) != wantCount {
		t.Fatal("期望範圍不符")
	}
	all, notice, err := LoadTheme(plan.Theme, orig, "", 3)
	if err != nil || all == nil || notice != "" {
		t.Fatal("正式主題載入失敗", notice, err)
	}
	hd, notice, err := LoadTheme(plan.BodyTheme, orig, "", 3)
	if err != nil || hd == nil || notice != "" || len(hd.groups) != wantCount {
		t.Fatal("完整身體批次載入失敗", notice, err)
	}
	digest := func(p []byte) string { return fmt.Sprintf("%x", sha256.Sum256(p)) }
	render := func(frame []byte) []byte {
		before := append([]byte(nil), frame...)
		hd.frameSprites(frame)
		hd.Layer.Frame(frame, make([]byte, 64000*3))
		p := make([]byte, 960*600*4)
		hd.Draw(p, 3)
		if !bytes.Equal(frame, before) {
			t.Fatal("修改原版frame")
		}
		return p
	}
	empty := make([]byte, 960*600*4)
	for _, row := range plan.Rows {
		t.Run(fmt.Sprintf("%s/%02d", row.PBL, row.Image), func(t *testing.T) {
			frame, err := os.ReadFile(row.Frame)
			if err != nil || len(frame) != 64000 || digest(frame) != row.FrameSHA {
				t.Fatal("原版合成來源不符", err)
			}
			hd.Enabled = true
			hd.ResetForLoad()
			got := render(frame)
			if digest(got) != row.Expected || bytes.Equal(got, empty) {
				t.Fatal("獨立整圖面／省略負對照不符")
			}
			hd.Enabled = false
			if !bytes.Equal(render(frame), empty) {
				t.Fatal("關閉仍有HD")
			}
			hd.Enabled = true
			if digest(render(frame)) != row.Expected {
				t.Fatal("重開不同")
			}
			cover := append([]byte(nil), frame...)
			cover[row.Cover[1]*320+row.Cover[0]] ^= 1
			if digest(render(cover)) != row.Covered {
				t.Fatal("整個8×8遮格不同")
			}
			if digest(render(frame)) != row.Expected {
				t.Fatal("遮格未恢復")
			}
			noAnchor := append([]byte(nil), frame...)
			noAnchor[248] ^= 1
			if !bytes.Equal(render(noAnchor), empty) {
				t.Fatal("錨點消失仍顯示角色")
			}
			hd.ResetForLoad()
			if digest(render(frame)) != row.Expected {
				t.Fatal("冷載完整來源未恢復")
			}
			wrong := append([]byte(nil), got...)
			for i := 3; i < len(wrong); i += 4 {
				if wrong[i] != 0 {
					wrong[i-3] ^= 1
					break
				}
			}
			if digest(wrong) == row.Expected {
				t.Fatal("單像素負對照無效")
			}
		})
	}
	// 舊主題與批次候選的倍率fallback一樣，不讀不存在的PNG。
	stopped, notice, err := LoadTheme(plan.Theme, orig, "", 2)
	if err != nil || stopped != nil || notice == "" {
		t.Fatal("倍率fallback不符", err)
	}
	if output := os.Getenv("PSYCHICWAR_READY60_ART_OUT"); output != "" {
		manifestBytes, err := os.ReadFile(filepath.Join(plan.Theme, "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		var manifest ThemeManifest
		if err = json.Unmarshal(manifestBytes, &manifest); err != nil {
			t.Fatal(err)
		}
		pngPaths, err := filepath.Glob(filepath.Join(plan.Theme, "*.png"))
		if err != nil {
			t.Fatal(err)
		}
		receipt := map[string]any{"status": "PASS_SYNTHETIC_BODY_ART_PLANES", "manifest_entries": len(manifest.Entries), "png_count": len(pngPaths), "body_sources": wantCount, "full_plane_checks": wantCount, "cover_restore_checks": wantCount, "anchor_checks": wantCount, "toggle_checks": wantCount, "negative_controls": "omit and single-pixel; all effective", "plan_sha256": digest(raw), "limits": "Synthetic independent original PBL fixtures only; not normal player path, GUI, DAT or complete animation."}
		data, err := json.MarshalIndent(receipt, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.OpenFile(filepath.Join(output, "render.json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.Write(append(data, '\n')); err != nil {
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
