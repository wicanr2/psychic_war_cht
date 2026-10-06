// 研究 038 §38：以兩個同狀態複本核對 VGA 匯流排讀取的觀察副作用。
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/wicanr2/dosgolem/oracle"
	"log"
	"os"
	"runtime"
)

func main() {
	state := flag.String("state", "workplace/hd/battle-effects-prototype-v2-20261001-step86539215.state", "相同中途狀態")
	out := flag.String("out", "", "全新收據檔名")
	flag.Parse()
	must := func(e error) {
		if e != nil {
			log.Fatal(e)
		}
	}
	hash := func(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
	read := func(path string) []byte { b, e := os.ReadFile(path); must(e); return b }
	if *out == "" {
		log.Fatal("必須指定 -out")
	}
	orig := "/orig/psychic-war"
	expected := "9af6bbbe75a7c07792a4f76ba69da5bee3155b12d3c649730c4563d7bc2f02e6"
	rows := []map[string]any{}
	var clean []byte
	differences := 0
	for _, busRead := range []bool{false, true} {
		o, e := oracle.Load(orig+"/PW.EXE", orig)
		must(e)
		must(o.LoadStateFile(*state))
		start := o.Steps()
		if start != 86539215 {
			log.Fatal("中途狀態起點不同")
		}
		before := hash(o.Indexed())
		if busRead {
			_ = o.Bytes(oracle.Addr{}, 1<<20)
		}
		if before != hash(o.Indexed()) {
			log.Fatal("讀取當下已改畫面")
		}
		must(o.Run(86562564 - start))
		frame := o.Indexed()
		if !busRead {
			if hash(frame) != expected {
				log.Fatal("乾淨分支與獨立原版終點不符")
			}
			clean = append([]byte(nil), frame...)
		} else {
			for i, c := range frame {
				if c != clean[i] {
					differences++
				}
			}
			if differences == 0 {
				log.Fatal("負對照未呈現副作用，不能確認原因")
			}
		}
		rows = append(rows, map[string]any{"bus_read_1mib": busRead, "start_steps": start, "end_steps": o.Steps(), "end_cycles": o.Cycles(), "indexed_sha256": hash(frame), "cpu_regs": o.Regs()})
		o.Close()
	}
	b, e := json.MarshalIndent(map[string]any{"scope": "同一 FIGHT 繪製中途狀態的有限 A/B；只確認匯流排觀察副作用，不深入硬體時序", "go_version": runtime.Version(), "inputs_sha256": map[string]string{*state: hash(read(*state)), orig + "/PW.EXE": hash(read(orig + "/PW.EXE")), "tools/hd/probe_vga_observation.go": hash(read("tools/hd/probe_vga_observation.go"))}, "expected_independent_indexed_sha256": expected, "rows": rows, "pixel_differences": differences}, "", "  ")
	must(e)
	f, e := os.OpenFile(*out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	must(e)
	_, e = f.Write(append(b, '\n'))
	must(e)
	must(f.Close())
	fmt.Printf("觀察副作用 A/B：乾淨分支符合原版，1MiB 匯流排讀取分支差 %d 像素\n", differences)
}
