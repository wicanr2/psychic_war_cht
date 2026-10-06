// 只讀原版正常遭遇的連續貼圖；由原始來源及前後畫面核對動作，禁止用推測替代證據。
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
	"runtime"

	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
)

func main() {
	out := flag.String("out", "", "全新輸出前綴")
	flag.Parse()
	if *out == "" {
		log.Fatal("必須指定 -out")
	}
	if ps, e := filepath.Glob(*out + "*"); e != nil || len(ps) != 0 {
		log.Fatal("拒絕覆寫輸出")
	}
	check := func(e error) {
		if e != nil {
			log.Fatal(e)
		}
	}
	read := func(p string) []byte { b, e := os.ReadFile(p); check(e); return b }
	hash := func(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
	write := func(p string, b []byte) {
		f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		check(e)
		_, e = f.Write(b)
		check(e)
		check(f.Close())
	}
	const orig = "/orig/psychic-war"
	const start = "workplace/states/07-first-play.state"
	inputs := map[string]string{}
	for _, p := range []string{start, orig + "/PW.EXE", orig + "/ENEMY00.PBL", "tools/hd/observe_enemy_cycle.go"} {
		inputs[p] = hash(read(p))
	}
	if inputs[start] != "56c489deaa57cd26fbd4f39f374e87db2f51e7d1275c440c9805321fb09bd532" || inputs[orig+"/ENEMY00.PBL"] != "8241b0ea73b1e7402b13adc434f10a5b1288f88f01fcfc3ba5710e4923fe6067" {
		log.Fatal("固定來源不符")
	}
	b := read(orig + "/ENEMY00.PBL")
	poses := map[int][]byte{}
	for n := 0; n < 30; n++ {
		w, h, px, e := pbl.Decode(b, n)
		check(e)
		if w == 24 && h == 32 {
			poses[n] = px
		}
	}
	matches := func(px []byte) []int {
		ids := []int{}
		for n := 0; n < 30; n++ {
			if bytes.Equal(px, poses[n]) {
				ids = append(ids, n)
			}
		}
		return ids
	}
	o, e := oracle.Load(orig+"/PW.EXE", orig)
	check(e)
	defer o.Close()
	check(o.LoadStateFile(start))
	o.SetScratch("/tmp/pw-enemy-cycle")
	if o.Word(oracle.Addr{Seg: 0x161, Off: 0x41DF}) != 0x86AF {
		log.Fatal("執行前 seed 不符")
	}
	var entry oracle.Regs
	var enter uint64
	var before, raw []byte
	var events []map[string]any
	o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x8705}, func(o *oracle.Oracle) {
		r := o.Regs()
		if r.CX != 0x0826 || r.DX != 0x0304 {
			return
		}
		if before != nil || len(events) >= 64 {
			log.Fatal("貼圖巢狀或超過有界範圍")
		}
		entry, enter = r, o.Steps()
		before = o.Indexed()
		a := oracle.Addr{Seg: r.DS, Off: r.BX}
		if a.Linear() > 0xA0000-384 {
			log.Fatal("來源超過 RAM")
		}
		raw = o.Bytes(a, 384)
	})
	o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x8751}, func(o *oracle.Oracle) {
		if before == nil {
			return
		}
		after := o.Indexed()
		from, e := pbl.Region(before, 320, 200, 32, 152, 24, 32)
		check(e)
		to, e := pbl.Region(after, 320, 200, 32, 152, 24, 32)
		check(e)
		mode := entry.AX & 255
		mismatch := 0
		for i, v := range to {
			expected := raw[i/2]
			if i%2 == 0 {
				expected >>= 4
			} else {
				expected &= 15
			}
			if mode == 1 {
				expected ^= from[i]
			}
			if v != expected {
				mismatch++
			}
		}
		outside := 0
		for i, v := range after {
			if (i%320 < 32 || i%320 >= 56 || i/320 < 152 || i/320 >= 184) && v != before[i] {
				outside++
			}
		}
		if mode > 1 || mismatch != 0 || outside != 0 {
			log.Fatal("貼圖模式或來源推導與原版不符")
		}
		n := len(events)
		prefix := fmt.Sprintf("%s-event%02d", *out, n)
		write(prefix+"-before.frame", before)
		write(prefix+"-after.frame", after)
		write(prefix+"-source.bin", raw)
		events = append(events, map[string]any{"entry_step": enter, "return_step": o.Steps(), "entry_regs": entry, "return_regs": o.Regs(), "cycles": o.Cycles(), "al": mode, "from_images": matches(from), "to_images": matches(to), "before_sha256": hash(before), "after_sha256": hash(after), "source_sha256": hash(raw), "source_mismatch": mismatch, "outside_changed_pixels": outside})
		before = nil
	})
	run := func(n uint64) { check(o.Run(n - o.Steps())) }
	for i := 0; i < 6; i++ {
		run(43000000 + uint64(i)*8000000)
		o.KeyDown(0x48)
		if i < 5 {
			run(47000000 + uint64(i)*8000000)
			o.KeyUp(0x48)
		}
	}
	run(86000000)
	if len(events) < 5 || before != nil {
		log.Fatal("不足以觀察循環或終點仍在貼圖")
	}
	mem := o.Bytes(oracle.Addr{}, 1<<20)
	doc := map[string]any{"tool": "dosgolem f8c1a6e 原版唯讀掛鉤", "go_version": runtime.Version(), "inputs_sha256": inputs, "seed_before": "86AF", "seed_method": "固定雜湊 state，執行前唯讀核對", "address_space": "dosgolem 執行期 CS:IP／DS:BX；原版 320×200 色號座標", "input_prefix": "Up 按下 43,000,000 起每 8,000,000 一次，前五次 4,000,000 後放開；止於 86,000,000，第六次放開尚未到達", "events": events, "end_steps": o.Steps(), "end_cycles": o.Cycles(), "end_regs": o.Regs(), "end_indexed_sha256": hash(o.Indexed()), "end_ram_sha256": hash(mem), "limits": "此正常遭遇的有界連續動作觀察；其他敵人、攻擊、效果及時序尚未涵蓋"}
	jsonBytes, e := json.MarshalIndent(doc, "", "  ")
	check(e)
	write(*out+".json", append(jsonBytes, '\n'))
	fmt.Printf("原版連續 %d 次貼圖的來源、前後畫面與區外不變通過\n", len(events))
}
