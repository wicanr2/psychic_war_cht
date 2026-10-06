// 從正常鍵序重建四檢查點中文Layer，原版state只讀；研究038 §64。
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

type segment struct {
	Name, From string
	SaveAt     uint64 `json:"save_at"`
	KeyAt      uint64 `json:"key_at"`
	KeyEvery   uint64 `json:"key_every"`
	Keys       []string
}

func scan(s string) uint8 {
	if n, ok := map[string]uint8{"space": 0x39, "enter": 0x1c}[s]; ok {
		return n
	}
	n, e := strconv.ParseUint(s, 16, 8)
	check(e)
	return uint8(n)
}
func main() {
	out := flag.String("out", "", "全新輸入目錄")
	flag.Parse()
	if *out == "" {
		log.Fatal("缺輸出")
	}
	if _, e := os.Stat(*out); !os.IsNotExist(e) {
		log.Fatal("拒絕覆寫")
	}
	check(os.Mkdir(*out, 0755))
	const orig = "/orig/psychic-war"
	inputs := map[string]string{}
	for _, p := range []string{"tools/hd/prepare_checkpoint_layers.go", "workplace/hd/normal-chain-oracle-bridge-v1-20261001.go", "replay/title-to-first-save.json", orig + "/PW.EXE"} {
		inputs[p] = hash(read(p))
	}
	if inputs[orig+"/PW.EXE"] != "88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49" {
		log.Fatal("EXE來源不符")
	}
	fixed := map[string]string{"01-title": "bd38fe2f7a2381725d3317e4229fea842407ce3a6feba5494ddf946b04458c5e", "03-protection": "9ddfb4af2ffc283e84f4221521a9e8404e2500e0944bb6ea48bdc48caed62af9", "05-select": "10fc7ec0faf9e5b67b187fac4a9261c90f757c8f9f0aa84f2ddaa57a47d3c9ac", "07-first-play": "56c489deaa57cd26fbd4f39f374e87db2f51e7d1275c440c9805321fb09bd532"}
	for n, want := range fixed {
		p := "workplace/states/" + n + ".state"
		inputs[p] = hash(read(p))
		if inputs[p] != want {
			log.Fatal("固定state不同：", n)
		}
	}
	for _, pattern := range []string{"text/*.json", "font/*.golemfnt", "apps/psychicwar/translator/*.go"} {
		ps, e := filepath.Glob(pattern)
		check(e)
		for _, p := range ps {
			inputs[p] = hash(read(p))
		}
	}
	entries, e := translator.LoadText("text")
	check(e)
	f24, e := xlate.LoadFont("font/cjk24.golemfnt")
	check(e)
	f16, e := xlate.LoadFont("font/cjk16.golemfnt")
	check(e)
	baked, e := translator.LoadBaked("text")
	check(e)
	var plan struct{ Segments []segment }
	check(json.Unmarshal(read("replay/title-to-first-save.json"), &plan))
	byName := map[string]segment{}
	for _, s := range plan.Segments {
		byName[s.Name] = s
	}
	sides := map[string][]byte{}
	rows := []map[string]any{}
	// 標題Logo依既有定案沿用原版，尚未有可轉譯動態字面。
	title := translator.NewTranslator(entries, f24, f16, 3, nil)
	side, e := title.Layer.Snapshot()
	check(e)
	sides["01-title"] = side
	write(filepath.Join(*out, "01-title.state"), read("workplace/states/01-title.state"))
	write(filepath.Join(*out, "01-title.state.xlate.json"), side)
	for _, name := range []string{"02-match", "03-protection", "04-cleared", "05-select", "06-name", "07-first-play"} {
		s := byName[name]
		from := "workplace/states/" + s.From + ".state"
		expected := "workplace/states/" + s.Name + ".state"
		inputs[from] = hash(read(from))
		inputs[expected] = hash(read(expected))
		m := machine.New()
		d := dos.New(m, orig)
		d.Install()
		m.KeyEvery = s.KeyEvery
		if m.KeyEvery == 0 {
			m.KeyEvery = 500000
		}
		m.SetNextKey(s.KeyAt)
		for _, key := range s.Keys {
			m.QueueKey(scan(key))
		}
		check(state.Load(from, m, d))
		d.Root = orig
		d.Scratch = "/tmp/pw-hd-checkpoint-layer"
		o := oracle.ResearchWrapMachine(m, d)
		seed := o.Word(oracle.Addr{Seg: 0x161, Off: 0x41df})
		tr := translator.NewTranslator(entries, f24, f16, 3, nil)
		tr.Attach(o)
		check(tr.Layer.Restore(sides[s.From], tr.Fonts()))
		tr.AttachBaked(baked, orig)
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
			log.Fatal("正常中文重播與原版state不同：", name)
		}
		baseline.Close()
		side, e := tr.Layer.Snapshot()
		check(e)
		sides[name] = side
		if _, ok := fixed[name]; ok {
			write(filepath.Join(*out, name+".state"), read(expected))
			write(filepath.Join(*out, name+".state.xlate.json"), side)
		}
		rows = append(rows, map[string]any{"name": name, "from": s.From, "seed_before": fmt.Sprintf("%04X", seed), "input": s, "steps": o.Steps(), "cycles": o.Cycles(), "original_frame_sha256": hash(o.Indexed()), "ram_below_a0000_sha256": hash(o.Bytes(oracle.Addr{}, 0xa0000)), "layer_sha256": hash(side), "layer_stamp_count": len(tr.Layer.Stamps), "existing_state_equal": true})
		o.Close()
		fmt.Println(name, "正常中文Layer與原版state相同")
	}
	b, e := json.MarshalIndent(map[string]any{"inputs_sha256": inputs, "results": rows, "scope": "01-title保留原版Logo；六條既有正常鍵序重建至07-first-play，四個原版state bytes不變、中文Layer從實際印字產生", "limits": "磁碟state不保存IRQ1計數，不比較此欄；未驗GUI、HD合成、美術或正式包"}, "", "  ")
	check(e)
	write(filepath.Join(*out, "verification.json"), append(b, '\n'))
}
