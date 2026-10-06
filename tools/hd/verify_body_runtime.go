// 研究038 §108：正式Theme接續正常第0張狀態；原版資料只讀。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

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
func main() {
	const work = "/src/workplace/hd/body-runtime-v1-20261004"
	const orig = "/orig/psychic-war"
	const themeDir = "/src/workplace/hd/theme-kasuruji-pose2-v1-20261004"
	const start = "/src/workplace/hd/next-body-normal-kasuruji-v3-20261001-event00.state"
	const comparator = "/src/workplace/hd/maze-saved-state-independent-v1-20261003.bin"
	const evidence = "/src/workplace/hd/body-context-v2-20261004/trace.json"
	if hash(read(start)) != "2f64628424a694fed4aa0d0669cc76e6e31b1e1920409c6719593ad7d373b30a" {
		log.Fatal("原版起點不符")
	}
	inputs := map[string]string{}
	for _, p := range []string{start, evidence, comparator, orig + "/PW.EXE", themeDir + "/manifest.json", "/src/tools/hd/verify_body_runtime.go"} {
		inputs[p] = hash(read(p))
	}
	var source struct {
		Events []struct {
			Entry, Return uint64
			Rect          [4]int
			Kind          string
		}
		EndStep uint64 `json:"end_step"`
	}
	check(json.Unmarshal(read(evidence), &source))
	type point struct {
		Label string
		Step  uint64
	}
	points := []point{{"initial", 237710721}}
	sequence := []string{"pose1", "pose2", "pose1-back", "pose0-back"}
	var reloadStep uint64
	for _, event := range source.Events {
		if event.Kind != "packed" || event.Rect != [4]int{32, 152, 24, 32} {
			continue
		}
		if len(sequence) == 0 {
			break
		}
		label := sequence[0]
		sequence = sequence[1:]
		points = append(points, point{label + "-entry", event.Entry + 1}, point{label + "-mid", event.Entry + 1000}, point{label + "-return", event.Return + 1})
		if label == "pose2" {
			reloadStep = event.Return + 1
		}
	}
	points = append(points, point{"final", source.EndStep})
	if len(points) != 14 || reloadStep == 0 {
		log.Fatal("原版取樣計畫不符")
	}
	load := func(scratch string) *oracle.Oracle {
		o, e := oracle.Load(orig+"/PW.EXE", orig)
		check(e)
		check(o.LoadStateFile(start))
		o.SetScratch(filepath.Join(work, scratch))
		return o
	}
	o, control := load("hd-scratch"), load("control-scratch")
	defer o.Close()
	defer control.Close()
	hd, notice, e := theme.LoadTheme(themeDir, orig, "", 3)
	check(e)
	if hd == nil || notice != "" {
		log.Fatal("主題未接入")
	}
	check(hd.Attach(o))
	hd.Frame(o)
	seed := o.Word(oracle.Addr{Seg: 0x161, Off: 0x41df})
	var rows []map[string]any
	capture := func(label string, a, b *oracle.Oracle) {
		if !bytes.Equal(a.Indexed(), b.Indexed()) || a.Regs() != b.Regs() || a.Steps() != b.Steps() || a.Cycles() != b.Cycles() || !bytes.Equal(a.Bytes(oracle.Addr{}, 0xa0000), b.Bytes(oracle.Addr{}, 0xa0000)) {
			log.Fatal("HD改變原版")
		}
		prefix := filepath.Join(work, label)
		check(a.SaveStateFile(prefix + ".state"))
		check(b.SaveStateFile(prefix + "-control.state"))
		result, e := exec.Command(comparator, prefix+".state", prefix+"-control.state", prefix+"-machine.json").CombinedOutput()
		if e != nil {
			log.Fatalf("完整state比較：%v，%s", e, result)
		}
		hd.Frame(a)
		plane := make([]byte, 960*600*4)
		hd.Enabled = true
		hd.Draw(plane, 3)
		write(prefix+"-plane.rgba", plane)
		write(prefix+".frame", a.Indexed())
		w, h, rgb := a.ScreenRGB()
		if w != 320 || h != 200 {
			log.Fatal("原版畫面大小不符")
		}
		write(prefix+".rgb", rgb)
		hd.Enabled = false
		off := make([]byte, len(plane))
		if hd.Draw(off, 3) || !bytes.Equal(off, make([]byte, len(off))) {
			log.Fatal("HD關閉仍画圖")
		}
		hd.Enabled = true
		again := make([]byte, len(plane))
		hd.Draw(again, 3)
		if !bytes.Equal(again, plane) {
			log.Fatal("HD重啟不同")
		}
		rows = append(rows, map[string]any{"label": label, "step": a.Steps(), "cycles": a.Cycles(), "prefix": prefix, "toggle_equal": true, "original_frame_sha256": hash(a.Indexed())})
	}
	for _, p := range points {
		if p.Step < o.Steps() {
			log.Fatal("取樣順序不同")
		}
		check(o.Run(p.Step - o.Steps()))
		check(control.Run(p.Step - control.Steps()))
		capture(p.Label, o, control)
	}
	// 成功LoadStateFile及ResetForLoad後，從真正完整原圖重新建立身份並接續。
	check(o.LoadStateFile(start))
	check(control.LoadStateFile(start))
	hd.ResetForLoad()
	hd.Frame(o)
	check(o.Run(reloadStep - o.Steps()))
	check(control.Run(reloadStep - control.Steps()))
	capture("reload-full-next-pose2", o, control)
	// 真正載回受遮擋、貼圖中途的狀態，不能沿用之前的有效動作。
	mid := filepath.Join(work, "pose2-mid.state")
	inputs[mid] = hash(read(mid))
	check(o.LoadStateFile(mid))
	check(control.LoadStateFile(mid))
	hd.ResetForLoad()
	capture("reload-mid", o, control)
	check(o.Run(100000))
	check(control.Run(100000))
	capture("reload-mid-next", o, control)
	self, e := os.Executable()
	check(e)
	inputs[self] = hash(read(self))
	for _, p := range []string{"ally.go", "enemy.go", "theme.go"} {
		path := "/src/apps/psychicwar/theme/" + p
		inputs[path] = hash(read(path))
	}
	doc := map[string]any{"scope": "formal Theme normal complete-pose0 checkpoint continuation, 14 samples, genuine full and partial state reloads; no artificial game writes", "go_version": runtime.Version(), "seed_before": fmt.Sprintf("%04X", seed), "seed_method": "same immutable normal saved state, read before execution; no writes or rerolls", "theme": themeDir, "inputs_sha256": inputs, "samples": rows, "limits": "Not fromboot, actual Ebiten window, text, DAT, whole animation, platform or package evidence. New pose2 remains candidate until reviewed."}
	b, e := json.MarshalIndent(doc, "", "  ")
	check(e)
	write(filepath.Join(work, "runtime.json"), append(b, '\n'))
	fmt.Println(len(rows), "正常／中途／真正載回樣本，原版完整state核對通過")
}
