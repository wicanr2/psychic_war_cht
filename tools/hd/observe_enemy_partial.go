// 只讀正常 #3→#4 貼圖中的分段原版畫面；研究 §35 的補充原版證據。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/wicanr2/dosgolem/oracle"
	"log"
	"os"
	"runtime"
)

func main() {
	const out = "workplace/hd/enemy-partial-observation-20261001.json"
	if _, e := os.Lstat(out); !os.IsNotExist(e) {
		log.Fatal("拒絕覆寫")
	}
	orig := "/orig/psychic-war"
	start := "workplace/states/07-first-play.state"
	read := func(p string) []byte {
		b, e := os.ReadFile(p)
		if e != nil {
			log.Fatal(e)
		}
		return b
	}
	sh := func(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
	if sh(read(start)) != "56c489deaa57cd26fbd4f39f374e87db2f51e7d1275c440c9805321fb09bd532" {
		log.Fatal("起點不符")
	}
	o, e := oracle.Load(orig+"/PW.EXE", orig)
	if e != nil {
		log.Fatal(e)
	}
	defer o.Close()
	if e = o.LoadStateFile(start); e != nil {
		log.Fatal(e)
	}
	if o.Word(oracle.Addr{Seg: 0x161, Off: 0x41DF}) != 0x86AF {
		log.Fatal("執行前種子不符")
	}
	run := func(n uint64) {
		if e := o.Run(n - o.Steps()); e != nil {
			log.Fatal(e)
		}
	}
	for i := 0; i < 6; i++ {
		run(43000000 + uint64(i)*8000000)
		o.KeyDown(0x48)
		if i < 5 {
			run(47000000 + uint64(i)*8000000)
			o.KeyUp(0x48)
		}
	}
	before := read("workplace/probe/hd-xor-before1.frame")
	after := read("workplace/probe/hd-xor-after1.frame")
	run(83373343)
	if !bytes.Equal(o.Indexed(), before) {
		log.Fatal("起點前畫面不符")
	}
	var samples []map[string]any
	for _, step := range []uint64{83373344, 83374343, 83377343, 83381343, 83385343, 83389343, 83393343, 83397692} {
		run(step)
		px := o.Indexed()
		dBefore, dAfter, outside := 0, 0, 0
		for i, v := range px {
			in := i%320 >= 32 && i%320 < 56 && i/320 >= 152 && i/320 < 184
			if in {
				if v != before[i] {
					dBefore++
				}
				if v != after[i] {
					dAfter++
				}
			} else if v != before[i] {
				outside++
			}
		}
		samples = append(samples, map[string]any{"step": step, "regs": o.Regs(), "from_pose3_pixels": dBefore, "from_pose4_pixels": dAfter, "outside_changed_pixels": outside, "indexed_sha256": sh(px)})
	}
	doc := map[string]any{"tool": runtime.Version(), "seed_before": "86AF", "state_sha256": sh(read(start)), "address_space": "執行期段:偏移／原版色號座標", "samples": samples, "limits": "只證明這次正常 #3→#4 的分段貼圖；不解釋全部 AL 或循環"}
	b, e := json.MarshalIndent(doc, "", "  ")
	if e != nil {
		log.Fatal(e)
	}
	if e = os.WriteFile(out, append(b, '\n'), 0644); e != nil {
		log.Fatal(e)
	}
	fmt.Println(string(b))
}
