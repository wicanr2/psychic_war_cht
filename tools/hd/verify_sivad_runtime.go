// Sivad正常來源state的接入／中途載回／接續核對；研究038 §50，非正常HD試玩替代品。
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
func read(p string) []byte { b, e := os.ReadFile(p); check(e); return b }
func hash(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func write(p string, b []byte) {
	f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	check(e)
	_, e = f.Write(b)
	check(e)
	check(f.Close())
}

// 完整VGA匯流排只在終止後讀取；此函式後不繼續執行該state。
func end(o *oracle.Oracle) map[string]any {
	return map[string]any{"regs": o.Regs(), "steps": o.Steps(), "cycles": o.Cycles(), "indexed": hash(o.Indexed()), "ram_below_a0000": hash(o.Bytes(oracle.Addr{}, 0xa0000)), "ram_bus": hash(o.Bytes(oracle.Addr{}, 1<<20))}
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
	const themeDir = "workplace/hd/theme-sivad-v1-20261002"
	const source = "workplace/hd/sivad-body-source-v1-20261002"
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
	type sample struct {
		Step         uint64
		Event        int
		Kind, Prefix string
	}
	var doc struct{ Samples []sample }
	check(json.Unmarshal(read(source+".json"), &doc))
	if len(doc.Samples) != 16 {
		log.Fatal("正常樣本數不符")
	}
	doc.Samples = append(doc.Samples, sample{695000000, -1, "terminal", source + "-end"})
	inputs := map[string]string{}
	for _, p := range []string{"tools/hd/verify_sivad_runtime.go", "apps/psychicwar/theme/enemy.go", "apps/psychicwar/theme/theme.go", "apps/psychicwar/theme/ally.go", source + ".json", "workplace/hd/sivad-body-verification-v1-20261002.json", themeDir + "/manifest.json", orig + "/PW.EXE"} {
		inputs[p] = hash(read(p))
	}
	files, e := filepath.Glob(themeDir + "/*.png")
	check(e)
	for _, p := range files {
		inputs[p] = hash(read(p))
	}
	rows := []map[string]any{}
	for i, s := range doc.Samples {
		state := s.Prefix + ".state"
		frame := s.Prefix + ".frame"
		inputs[state] = hash(read(state))
		inputs[frame] = hash(read(frame))
		check(a.LoadStateFile(state))
		check(b.LoadStateFile(state))
		a.SetScratch("/tmp/pw-sivad-a")
		b.SetScratch("/tmp/pw-sivad-b")
		seed := a.Word(oracle.Addr{Seg: 0x161, Off: 0x41df})
		if seed != b.Word(oracle.Addr{Seg: 0x161, Off: 0x41df}) {
			log.Fatal("執行前seed不同")
		}
		if a.Steps() != s.Step || !bytes.Equal(a.Indexed(), read(frame)) || !bytes.Equal(b.Indexed(), a.Indexed()) {
			log.Fatal("載回來源state不符")
		}
		hd.Enabled = true
		hd.ResetForLoad()
		hd.Frame(b)
		pixels := make([]byte, 960*600*4)
		hd.Draw(pixels, 3)
		name := fmt.Sprintf("%s-sample%02d.png", *out, i)
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
			log.Fatal("重新登記圖面不同")
		}
		check(a.Run(100000))
		check(b.Run(100000))
		hd.Frame(b)
		ae, be := end(a), end(b)
		if !reflect.DeepEqual(ae, be) {
			log.Fatal("HD干擾原版接續")
		}
		rows = append(rows, map[string]any{"sample": i, "event": s.Event, "kind": s.Kind, "frame": frame, "state": state, "step": s.Step, "seed_before": fmt.Sprintf("%04X", seed), "plane_png": name, "plane_png_sha256": hash(read(name)), "reload_and_toggle_equal": true, "continuation_steps": 100000, "without_hd": ae, "with_hd": be})
	}
	data, e := json.MarshalIndent(map[string]any{"scope": "Sivad正常原版來源17個state的HD接入與同狀態接續", "inputs_sha256": inputs, "results": rows, "limits": "8完整＋8中途＋1原版終點保存狀態；非正常方向鍵HD／中文或GUI驗收。讀取完整匯流排後不再接續該state。"}, "", "  ")
	check(e)
	write(*out+".json", append(data, '\n'))
	fmt.Println("17個實際state載回、HD切換及各100000步原版接續一致")
}
