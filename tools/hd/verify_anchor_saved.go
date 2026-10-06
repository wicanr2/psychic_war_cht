// 左側錨點的原版檢查點與角色保存幀回歸；研究038 §67。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"

	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/theme"
)

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}
func read(p string) []byte { b, e := os.ReadFile(p); check(e); return b }
func hash(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func write(p string, b []byte) {
	f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(e)
	_, e = f.Write(b)
	check(e)
	check(f.Close())
}
func fp(o *oracle.Oracle) map[string]any {
	return map[string]any{"regs": o.Regs(), "steps": o.Steps(), "cycles": o.Cycles(), "frame": hash(o.Indexed()), "ram_below_a0000": hash(o.Bytes(oracle.Addr{}, 0xa0000))}
}
func main() {
	const orig = "/orig/psychic-war"
	const plan = "workplace/hd/background-anchor-saved-input-v1-20261003.json"
	const dir = "workplace/hd/theme-room-anchor-v1-20261003"
	const out = "workplace/hd/background-anchor-saved-v1-20261003"
	files, e := filepath.Glob(out + "*")
	check(e)
	if len(files) > 0 {
		log.Fatal("拒絕覆寫")
	}
	var doc struct {
		Inputs map[string]string               `json:"inputs_sha256"`
		Cases  []struct{ Frame, State string } `json:"cases"`
	}
	check(json.Unmarshal(read(plan), &doc))
	inputs := doc.Inputs
	for p, want := range inputs {
		if p[0] == '/' {
			continue
		}
		if hash(read(p)) != want {
			log.Fatal("輸入已變更：", p)
		}
	}
	for _, p := range []string{plan, "tools/hd/verify_anchor_saved.go", orig + "/PW.EXE", dir + "/manifest.json", "apps/psychicwar/theme/theme.go", "apps/psychicwar/theme/ally.go", "apps/psychicwar/theme/enemy.go"} {
		inputs[p] = hash(read(p))
	}
	rows := []map[string]any{}
	for i, c := range doc.Cases {
		o, e := oracle.Load(orig+"/PW.EXE", orig)
		check(e)
		check(o.LoadStateFile(c.State))
		o.SetScratch("/tmp/pw-anchor-saved")
		if !bytes.Equal(o.Indexed(), read(c.Frame)) {
			log.Fatal("原版保存frame與state不符：", c.Frame)
		}
		before := fp(o)
		hd, notice, e := theme.LoadTheme(dir, orig, "theme", 3)
		check(e)
		if hd == nil || notice != "" {
			log.Fatal("主題未載入")
		}
		check(hd.Attach(o))
		hd.Frame(o)
		plane := make([]byte, 960*600*4)
		hd.Draw(plane, 3)
		hd.Enabled = false
		off := make([]byte, len(plane))
		if hd.Draw(off, 3) || !bytes.Equal(off, make([]byte, len(off))) {
			log.Fatal("停用仍有圖面")
		}
		hd.ResetForLoad()
		hd.Enabled = true
		hd.Frame(o)
		again := make([]byte, len(plane))
		hd.Draw(again, 3)
		if !bytes.Equal(plane, again) {
			log.Fatal("重建圖面不符")
		}
		if !reflect.DeepEqual(before, fp(o)) {
			log.Fatal("HD修改原版保存狀態")
		}
		p := fmt.Sprintf("%s-sample%02d-plane.rgba", out, i)
		write(p, plane)
		rows = append(rows, map[string]any{"case": i, "frame": c.Frame, "state": c.State, "plane": p, "plane_sha256": hash(plane), "original_unchanged": true, "reload_and_toggle_equal": true, "original": before})
		o.Close()
	}
	b, e := json.MarshalIndent(map[string]any{"scope": "十九原版檢查點與四組完整角色來源，實際state載回與主題重建，不推進遊戲", "inputs_sha256": inputs, "results": rows, "limits": "原版RAM僅比較A0000h以下；圖面仍須獨立核對，不是正常鍵序或GUI替代品"}, "", "  ")
	check(e)
	write(out+".json", append(b, '\n'))
	fmt.Println("原版保存狀態回歸", len(rows), "份，圖面開關與重建相同")
}
