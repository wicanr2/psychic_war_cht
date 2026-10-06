// 由正常名字／戰鬥／存檔／治療重播建立中文Layer，逐段核對既有原版；入口研究038 §50。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wicanr2/dosgolem/internal/dos"
	"github.com/wicanr2/dosgolem/internal/machine"
	"github.com/wicanr2/dosgolem/internal/state"
	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/dosgolem/xlate"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/translator"
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
func scan(k string) uint8 {
	if sc, ok := map[string]uint8{"up": 0x48, "right": 0x4d, "down": 0x50, "left": 0x4b, "esc": 1, "enter": 0x1c, "space": 0x39}[k]; ok {
		return sc
	}
	sc, e := strconv.ParseUint(k, 16, 8)
	check(e)
	return uint8(sc)
}
func steps(s string, ips float64) uint64 {
	if text, ok := strings.CutSuffix(s, "ms"); ok {
		value, e := strconv.ParseFloat(text, 64)
		check(e)
		return uint64(value / 1000 * ips)
	}
	n, e := strconv.ParseUint(s, 10, 64)
	check(e)
	return n
}

type segment struct {
	Name, From  string
	SaveAt      uint64 `json:"save_at"`
	KeyAt       uint64 `json:"key_at"`
	KeyEvery    uint64 `json:"key_every"`
	Keys, Holds []string
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
	inputs := map[string]string{}
	for _, p := range []string{"tools/hd/rebuild_sivad_translation.go", "workplace/hd/normal-chain-oracle-bridge-v1-20261001.go", "replay/title-to-first-save.json", "workplace/hd/normal-chain-translation-v1-20261001.json", "workplace/hd/normal-chain-translation-v1-20261001-13-healed.layer.json"} {
		inputs[p] = hash(read(p))
	}
	for _, pattern := range []string{"text/*.json", "font/*.golemfnt", "apps/psychicwar/translator/*.go"} {
		ps, e := filepath.Glob(pattern)
		check(e)
		for _, p := range ps {
			inputs[p] = hash(read(p))
		}
	}
	var plan struct{ Segments []segment }
	check(json.Unmarshal(read("replay/title-to-first-save.json"), &plan))
	byName := map[string]segment{}
	for _, s := range plan.Segments {
		byName[s.Name] = s
	}
	layers := map[string][]byte{"13-healed": read("workplace/hd/normal-chain-translation-v1-20261001-13-healed.layer.json")}
	rows := []map[string]any{}
	entries, e := translator.LoadText("text")
	check(e)
	f24, e := xlate.LoadFont("font/cjk24.golemfnt")
	check(e)
	f16, e := xlate.LoadFont("font/cjk16.golemfnt")
	check(e)
	baked, e := translator.LoadBaked("text")
	check(e)
	for _, name := range []string{"14-minton1", "15-minton2", "16-sivad", "17-saved2"} {
		s, ok := byName[name]
		if !ok {
			log.Fatal("缺正常段落")
		}
		start := "workplace/states/" + s.From + ".state"
		expected := "workplace/states/" + s.Name + ".state"
		inputs[start] = hash(read(start))
		inputs[expected] = hash(read(expected))
		m := machine.New()
		d := dos.New(m, orig)
		d.Install()
		if len(s.Keys) != 0 {
			m.KeyEvery = s.KeyEvery
			m.SetNextKey(s.KeyAt)
			for _, k := range s.Keys {
				m.QueueKey(scan(k))
			}
		}
		check(state.Load(start, m, d))
		d.Root = orig
		d.Scratch = "/tmp/pw-normal-chain"
		for _, spec := range s.Holds {
			k, time, ok := strings.Cut(spec, "@")
			at, dur, ok2 := strings.Cut(time, "+")
			if !ok || !ok2 {
				log.Fatal("按住格式不符")
			}
			m.HoldKey(scan(k), steps(at, m.InstructionsPerSecond()), steps(dur, m.InstructionsPerSecond()), true)
		}
		o := oracle.ResearchWrapMachine(m, d)
		tr := translator.NewTranslator(entries, f24, f16, 3, nil)
		tr.Attach(o)
		if side := layers[s.From]; len(side) != 0 {
			check(tr.Layer.Restore(side, tr.Fonts()))
		} else if s.From != "06-name" {
			log.Fatal("缺上一正常段落中文Layer")
		}
		tr.AttachBaked(baked, orig)
		seed := o.Word(oracle.Addr{Seg: 0x161, Off: 0x41df})
		for o.Steps() < s.SaveAt {
			n := s.SaveAt - o.Steps()
			if n > 100000 {
				n = 100000
			}
			check(o.Run(n))
			tr.Frame(o)
		}
		baseline, e := oracle.Load(orig+"/PW.EXE", orig)
		check(e)
		check(baseline.LoadStateFile(expected))
		if o.Regs() != baseline.Regs() || o.Steps() != baseline.Steps() || o.Cycles() != baseline.Cycles() || !bytes.Equal(o.Indexed(), baseline.Indexed()) || !bytes.Equal(o.Bytes(oracle.Addr{}, 0xa0000), baseline.Bytes(oracle.Addr{}, 0xa0000)) {
			log.Fatal("正常中文重播與既有state不符：", name)
		}
		side, e := tr.Layer.Snapshot()
		check(e)
		layers[name] = side
		p := *out + "-" + name
		write(p+".layer.json", side)
		write(p+".frame", o.Indexed())
		pixels := make([]byte, 960*600*4)
		tr.Layer.Draw(pixels, 3, tr.MissingGlyph)
		write(p+"-text.rgba", pixels)
		rows = append(rows, map[string]any{"name": name, "from": s.From, "seed_before": fmt.Sprintf("%04X", seed), "input": s, "layer_json": p + ".layer.json", "layer_sha256": hash(side), "steps": o.Steps(), "cycles": o.Cycles(), "state_regs": o.Regs(), "original_frame_sha256": hash(o.Indexed()), "ram_below_a0000_sha256": hash(o.Bytes(oracle.Addr{}, 0xa0000)), "baseline_state_sha256": inputs[expected], "baseline_equal": true})
		baseline.Close()
		o.Close()
		fmt.Println(name, "正常中文Layer與既有原版state相同")
	}
	b, e := json.MarshalIndent(map[string]any{"tool": "dosgolem f8c1a6e；研究編譯覆映射包裝正常FIFO機器，未修改正式API", "inputs_sha256": inputs, "results": rows, "limits": "由已驗13-healed中文Layer接續四條既有正常段落到Sivad；原版狀態比較限CPU／cycles／畫面／A0000h以下RAM，中文Layer來自正常印字，不證明全文或全遊戲驗收"}, "", "  ")
	check(e)
	write(*out+".json", append(b, '\n'))
}
