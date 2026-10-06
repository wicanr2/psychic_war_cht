// 只讀第一個一般貼圖間隙的函式進入點與畫面，保留原始執行位置。
package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/wicanr2/dosgolem/oracle"
	"log"
	"os"
	"runtime"
)

func main() {
	const out = "workplace/hd/effect-gap-v2-20261001.json"
	if _, e := os.Lstat(out); !os.IsNotExist(e) {
		log.Fatal("拒絕覆寫")
	}
	check := func(e error) {
		if e != nil {
			log.Fatal(e)
		}
	}
	read := func(p string) []byte { b, e := os.ReadFile(p); check(e); return b }
	hash := func(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
	const start = "workplace/states/08-encounter.state"
	if hash(read(start)) != "02d5fdba9181cb7fbec6282d74efbc25396562ba1403893cdf4c579ddfe770ba" {
		log.Fatal("起點不符")
	}
	o, e := oracle.Load("/orig/psychic-war/PW.EXE", "/orig/psychic-war")
	check(e)
	defer o.Close()
	check(o.LoadStateFile(start))
	var hits []map[string]any
	counts := map[string]int{}
	var functions struct {
		Functions []struct {
			Start string `json:"start"`
			End   string `json:"end"`
			Name  string `json:"name"`
		} `json:"functions"`
	}
	check(json.Unmarshal(read("workplace/ida/hd-effects-20261001/effects-functions.json"), &functions))
	var last []byte
	var previous map[string]any
	for _, f := range functions.Functions {
		var ea uint32
		_, e := fmt.Sscanf(f.Start, "%X", &ea)
		check(e)
		if ea < 0x10510 || ea >= 0x20510 {
			continue
		}
		o.OnCall(oracle.Addr{Seg: 0x161, Off: uint16(ea - 0x10510)}, func(o *oracle.Oracle) {
			if o.Steps() < 86702328 || o.Steps() >= 86742497 {
				return
			}
			key := f.Start
			counts[key]++
			now := o.Indexed()
			diff := 0
			if last != nil {
				for i, v := range now {
					if v != last[i] {
						diff++
					}
				}
			}
			hit := map[string]any{"step": o.Steps(), "ida_ea": key, "runtime_ip": o.IP(), "function": f.Name, "regs": o.Regs(), "changed_since_previous_entry": diff, "previous_entry": previous}
			if diff > 0 {
				check(os.WriteFile(fmt.Sprintf("workplace/hd/effect-gap-v2-%d-before.frame", o.Steps()), last, 0644))
				check(os.WriteFile(fmt.Sprintf("workplace/hd/effect-gap-v2-%d-after.frame", o.Steps()), now, 0644))
			}
			if len(hits) < 512 {
				hits = append(hits, hit)
			}
			previous = map[string]any{"step": o.Steps(), "ida_ea": key, "runtime_ip": o.IP(), "function": f.Name, "regs": o.Regs()}
			last = now
		})
	}
	check(o.Run(86500000 - o.Steps()))
	o.KeyDown(0x39)
	check(o.Run(86742497 - o.Steps()))
	b, e := json.MarshalIndent(map[string]any{"go_version": runtime.Version(), "state_sha256": hash(read(start)), "exe_sha256": hash(read("/orig/psychic-war/PW.EXE")), "tool_sha256": hash(read("tools/hd/observe_effect_gap.go")), "address_space": "WriteHit.At 為 oracle.ToIDA 映射；記憶體寫入為執行期線性位址，regs 為當次可見暫存器", "counts": counts, "first_hits": hits}, "", "  ")
	check(e)
	check(os.WriteFile(out, append(b, '\n'), 0644))
	fmt.Println(counts)
}
