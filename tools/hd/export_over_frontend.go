// 只讀正式前端F10保存狀態，匯出原版索引及RGB；研究038 §52。
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/wicanr2/dosgolem/oracle"
)

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}
func hash(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func read(p string) []byte { b, e := os.ReadFile(p); check(e); return b }
func write(p string, b []byte) {
	f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(e)
	_, e = f.Write(b)
	check(e)
	check(f.Close())
}
func main() {
	out := flag.String("out", "", "實際前端或pwstep產物目錄")
	flag.Parse()
	if *out == "" {
		log.Fatal("缺輸出目錄")
	}
	paths, e := filepath.Glob(filepath.Join(*out, "*.state"))
	check(e)
	inputs := map[string]string{}
	inputs["tools/hd/export_over_frontend.go"] = hash(read("tools/hd/export_over_frontend.go"))
	rows := []map[string]any{}
	for _, p := range paths {
		inputs[p] = hash(read(p))
		o, e := oracle.Load("/orig/psychic-war/PW.EXE", "/orig/psychic-war")
		check(e)
		check(o.LoadStateFile(p))
		w, h, rgb := o.ScreenRGB()
		if w != 320 || h != 200 {
			log.Fatal("原版尺寸不同")
		}
		indexed := o.Indexed()
		write(p+".frame", indexed)
		write(p+".rgb.bin", rgb)
		rows = append(rows, map[string]any{"state": p, "regs": o.Regs(), "steps": o.Steps(), "cycles": o.Cycles(), "seed": fmt.Sprintf("%04X", o.Word(oracle.Addr{Seg: 0x161, Off: 0x41df})), "area": o.Word(oracle.Addr{Seg: 0x1696, Off: 6}), "x": o.Word(oracle.Addr{Seg: 0x1696, Off: 8}), "y": o.Word(oracle.Addr{Seg: 0x1696, Off: 10}), "direction": o.Word(oracle.Addr{Seg: 0x1696, Off: 16}), "frame_sha256": hash(indexed), "rgb_sha256": hash(rgb), "ram_below_a0000_sha256": hash(o.Bytes(oracle.Addr{}, 0xa0000)), "ram_bus_terminal_sha256": hash(o.Bytes(oracle.Addr{}, 1<<20))})
		o.Close()
	}
	if len(rows) == 0 {
		log.Fatal("無保存狀態")
	}
	b, e := json.MarshalIndent(map[string]any{"scope": "實際前端保存狀態的原版320×200索引與RGB，唯讀匯出", "inputs_sha256": inputs, "results": rows, "limits": "未推進遊戲，不把視窗自然時間當成固定種子同狀態重播"}, "", "  ")
	check(e)
	write(filepath.Join(*out, "original-frames.json"), append(b, '\n'))
	fmt.Println("原版保存狀態匯出", len(rows), "份")
}
