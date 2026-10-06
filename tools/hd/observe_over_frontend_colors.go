// 正式前端陣亡文字的原版RGB／定色診斷，研究038 §52。
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/dosgolem/xlate"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/translator"
	"log"
	"os"
	"strings"
)

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}
func main() {
	output := flag.String("out", "workplace/hd/over-frontend-colors-v1-20261002.json", "全新診斷結果")
	flag.Parse()
	out := *output
	if _, e := os.Stat(out); !os.IsNotExist(e) {
		log.Fatal("拒絕覆寫")
	}
	o, e := oracle.Load("/orig/psychic-war/PW.EXE", "/orig/psychic-war")
	check(e)
	defer o.Close()
	check(o.LoadStateFile("workplace/hd/over-runtime-v1-20261002-event07.state"))
	o.SetAdLib(true)
	o.SetDOSBoxCycles(750)
	audio := o.NewAudio(44100)
	entries, e := translator.LoadText("text")
	check(e)
	f24, e := xlate.LoadFont("font/cjk24.golemfnt")
	check(e)
	f16, e := xlate.LoadFont("font/cjk16.golemfnt")
	check(e)
	tr := translator.NewTranslator(entries, f24, f16, 3, nil)
	tr.Attach(o)
	rows := []map[string]any{}
	last := ""
	for i := 0; i < 1600; i++ {
		check(o.RunCycles(12500))
		w, h, rgb := o.ScreenRGB()
		idx := o.Indexed()
		if w != 320 || h != 200 {
			log.Fatal("尺寸不同")
		}
		tr.Frame(o)
		audio.Render()
		stamps := []map[string]any{}
		for _, s := range tr.Layer.Stamps {
			if strings.HasPrefix(s.Key, "PW.EXE:cs:09") || s.Key == "PW.EXE:cs:08D7" {
				stamps = append(stamps, map[string]any{"key": s.Key, "state": s.State, "fg": s.FG, "bg": s.BG, "transparent": s.Transparent})
			}
		}
		scan := []map[string]any{}
		for _, y := range []int{128, 136, 144} {
			counts := map[byte]int{}
			colors := map[byte][3]byte{}
			for p := y * 320; p < (y+8)*320; p++ {
				counts[idx[p]]++
				colors[idx[p]] = [3]byte{rgb[p*3], rgb[p*3+1], rgb[p*3+2]}
			}
			scan = append(scan, map[string]any{"y": y, "counts": counts, "colors": colors})
		}
		key, e := json.Marshal(map[string]any{"stamps": stamps, "scan": scan})
		check(e)
		if string(key) != last {
			last = string(key)
			rows = append(rows, map[string]any{"steps": o.Steps(), "cycles": o.Cycles(), "stamps": stamps, "scan": scan, "indexed_sha256": fmt.Sprintf("%x", sha256.Sum256(idx)), "rgb_sha256": fmt.Sprintf("%x", sha256.Sum256(rgb))})
		}
	}
	b, e := json.MarshalIndent(map[string]any{"scope": "原版state正常接續、固定12500cycles每次、三行中文與原版RGB診斷", "start": "workplace/hd/over-runtime-v1-20261002-event07.state", "rows": rows}, "", "  ")
	check(e)
	f, e := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	check(e)
	_, e = f.Write(append(b, '\n'))
	check(e)
	check(f.Close())
	fmt.Println("原版顏色與疊字變更", len(rows), "筆")
}
