// ROOM0正常來源state載回、主題重建與原版接續；研究038 §67。
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

// 終點共同讀取VGA匯流排後，不再接續這份執行器。
func end(o *oracle.Oracle) map[string]any {
	return map[string]any{"regs": o.Regs(), "steps": o.Steps(), "cycles": o.Cycles(), "frame": hash(o.Indexed()), "ram": hash(o.Bytes(oracle.Addr{}, 0xa0000)), "bus": hash(o.Bytes(oracle.Addr{}, 1<<20))}
}
func main() {
	const orig = "/orig/psychic-war"
	const dir = "workplace/hd/theme-room-anchor-v1-20261003"
	const normal = "workplace/hd/room-anchor-runtime-v1-20261003"
	const out = "workplace/hd/room-anchor-reload-v1-20261003"
	ps, e := filepath.Glob(out + "*")
	check(e)
	if len(ps) > 0 {
		log.Fatal("拒絕覆寫")
	}
	inputs := map[string]string{}
	for _, p := range []string{"tools/hd/verify_anchor_reload.go", normal + ".json", orig + "/PW.EXE", dir + "/manifest.json", "apps/psychicwar/theme/theme.go", "apps/psychicwar/theme/ally.go", "apps/psychicwar/theme/enemy.go"} {
		inputs[p] = hash(read(p))
	}
	files, e := filepath.Glob(dir + "/*.png")
	check(e)
	for _, p := range files {
		inputs[p] = hash(read(p))
	}
	rows := []map[string]any{}
	for _, sample := range []int{7, 9, 11, 12} {
		prefix := fmt.Sprintf("%s-sample%02d", normal, sample)
		inputs[prefix+".state"] = hash(read(prefix + ".state"))
		inputs[prefix+".frame"] = hash(read(prefix + ".frame"))
		a, e := oracle.Load(orig+"/PW.EXE", orig)
		check(e)
		b, e := oracle.Load(orig+"/PW.EXE", orig)
		check(e)
		check(a.LoadStateFile(prefix + ".state"))
		check(b.LoadStateFile(prefix + ".state"))
		a.SetScratch("/tmp/pw-room-reload-a")
		b.SetScratch("/tmp/pw-room-reload-b")
		if !bytes.Equal(a.Indexed(), read(prefix+".frame")) || !bytes.Equal(a.Indexed(), b.Indexed()) {
			log.Fatal("載回原版畫面不符")
		}
		seed := a.Word(oracle.Addr{Seg: 0x161, Off: 0x41df})
		if seed != b.Word(oracle.Addr{Seg: 0x161, Off: 0x41df}) {
			log.Fatal("載回seed不符")
		}
		hd, notice, e := theme.LoadTheme(dir, orig, "theme", 3)
		check(e)
		if hd == nil || notice != "" {
			log.Fatal("主題未載入")
		}
		check(hd.Attach(b))
		hd.Frame(b)
		plane := make([]byte, 960*600*4)
		hd.Draw(plane, 3)
		hd.Enabled = false
		off := make([]byte, len(plane))
		if hd.Draw(off, 3) || !bytes.Equal(off, make([]byte, len(off))) {
			log.Fatal("停用仍有圖面")
		}
		hd.ResetForLoad()
		hd.Enabled = true
		hd.Frame(b)
		restored := make([]byte, len(plane))
		hd.Draw(restored, 3)
		if !bytes.Equal(plane, restored) {
			log.Fatal("重登記圖面不符")
		}
		p := fmt.Sprintf("%s-sample%02d", out, sample)
		write(p+"-before-plane.rgba", plane)
		check(a.Run(100000))
		check(b.Run(100000))
		hd.Frame(b)
		next := make([]byte, len(plane))
		hd.Draw(next, 3)
		write(p+".frame", b.Indexed())
		write(p+"-plane.rgba", next)
		ae, be := end(a), end(b)
		if !reflect.DeepEqual(ae, be) {
			log.Fatal("HD干擾原版接續")
		}
		rows = append(rows, map[string]any{"sample": sample, "source": prefix, "prefix": p, "seed_before": fmt.Sprintf("%04X", seed), "continuation_steps": 100000, "reload_and_toggle_equal": true, "without_hd": ae, "with_hd": be})
		a.Close()
		b.Close()
	}
	doc := map[string]any{"scope": "ROOM0 #8正常來源兩份完整房間與兩份清除state載回、重建及各100000步原版接續", "inputs_sha256": inputs, "results": rows, "limits": "保存正常state抽測；不是正常鍵序或正式GUI的替代品，未驗美術與全部sprite"}
	b, e := json.MarshalIndent(doc, "", "  ")
	check(e)
	write(out+".json", append(b, '\n'))
	fmt.Println("四個真實state載回、重建與各100000步原版接續相同")
}
