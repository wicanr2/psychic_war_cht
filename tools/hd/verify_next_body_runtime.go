// 正常來源保存狀態的限定接入核對，非完整HD玩家試玩；入口研究038 §48。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
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
func hash(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func read(p string) []byte { b, e := os.ReadFile(p); check(e); return b }
func write(p string, b []byte) {
	f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	check(e)
	_, e = f.Write(b)
	check(e)
	check(f.Close())
}
func end(o *oracle.Oracle) map[string]any {
	return map[string]any{"regs": o.Regs(), "steps": o.Steps(), "cycles": o.Cycles(), "indexed": hash(o.Indexed()), "ram_bus": hash(o.Bytes(oracle.Addr{}, 1<<20))}
}
func main() {
	out := flag.String("out", "", "全新輸出前綴")
	flag.Parse()
	if *out == "" {
		log.Fatal("缺輸出")
	}
	paths, e := filepath.Glob(*out + "*")
	check(e)
	if len(paths) != 0 {
		log.Fatal("拒絕覆寫")
	}
	const orig = "/orig/psychic-war"
	const themeDir = "workplace/hd/theme-next-bodies-v1-20261001"
	hd, notice, e := theme.LoadTheme(themeDir, orig, "theme", 3)
	check(e)
	if notice != "" || hd == nil {
		log.Fatal("未載入主題")
	}
	a, e := oracle.Load(orig+"/PW.EXE", orig)
	check(e)
	defer a.Close()
	b, e := oracle.Load(orig+"/PW.EXE", orig)
	check(e)
	defer b.Close()
	check(hd.Attach(b))
	inputs := map[string]string{}
	for _, p := range []string{"tools/hd/verify_next_body_runtime.go", "apps/psychicwar/theme/enemy.go", "apps/psychicwar/theme/theme.go", "apps/psychicwar/theme/ally.go", themeDir + "/manifest.json"} {
		inputs[p] = hash(read(p))
	}
	files, e := filepath.Glob(themeDir + "/*.png")
	check(e)
	for _, p := range files {
		inputs[p] = hash(read(p))
	}
	rows := []map[string]any{}
	for _, label := range []string{"kasuruji", "minton"} {
		for i := 0; i < 16; i++ {
			p := fmt.Sprintf("workplace/hd/next-body-%s-v2-20261001-event%02d", label, i)
			state := p + ".state"
			inputs[state] = hash(read(state))
			check(a.LoadStateFile(state))
			check(b.LoadStateFile(state))
			a.SetScratch("/tmp/pw-next-body-runtime-a")
			b.SetScratch("/tmp/pw-next-body-runtime-b")
			seed := a.Word(oracle.Addr{Seg: 0x161, Off: 0x41df})
			if b.Word(oracle.Addr{Seg: 0x161, Off: 0x41df}) != seed {
				log.Fatal("執行前seed不同")
			}
			if !bytes.Equal(a.Indexed(), read(p+"-after.frame")) {
				log.Fatal("實際載回state畫面不同")
			}
			hd.Enabled = true
			hd.ResetForLoad()
			hd.Frame(b)
			pixels := make([]byte, 960*600*4)
			hd.Draw(pixels, 3)
			name := fmt.Sprintf("%s-%s-%02d.png", *out, label, i)
			f, e := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
			check(e)
			check(png.Encode(f, &image.NRGBA{Pix: pixels, Stride: 960 * 4, Rect: image.Rect(0, 0, 960, 600)}))
			check(f.Close())
			hd.Enabled = false
			off := make([]byte, len(pixels))
			if hd.Draw(off, 3) || !bytes.Equal(off, make([]byte, len(off))) {
				log.Fatal("HD關閉仍輸出")
			}
			hd.ResetForLoad()
			hd.Enabled = true
			hd.Frame(b)
			restored := make([]byte, len(pixels))
			hd.Draw(restored, 3)
			if !bytes.Equal(pixels, restored) {
				log.Fatal("重新登記來源不同")
			}
			check(a.Run(100000))
			check(b.Run(100000))
			hd.Frame(b)
			ae, be := end(a), end(b)
			if !reflect.DeepEqual(ae, be) {
				log.Fatal("HD干擾原版接續終點")
			}
			rows = append(rows, map[string]any{"label": label, "event": i, "state_sha256": inputs[state], "seed_before": fmt.Sprintf("%04X", seed), "plane_png": name, "plane_png_sha256": hash(read(name)), "reload_and_toggle_equal": true, "continuation_steps": 100000, "without_hd": ae, "with_hd": be})
		}
	}
	data, e := json.MarshalIndent(map[string]any{"scope": "32個正常來源返回state的主題接入／重登記／開關及等步數原版接續抽測", "inputs_sha256": inputs, "results": rows, "limits": "不代替從迷宮到戰鬥的完整HD正常玩家路徑，未驗美術、特效與中途輸出"}, "", "  ")
	check(e)
	write(*out+".json", append(data, '\n'))
	fmt.Println("32個實際載回狀態、HD開關／重登記與原版接續終點一致")
}
