// 正常重播第二、第三場遭遇的有界身體來源探針；入口見研究038 §48。
// 研究工具需在臨時 dosgolem 子模組建置，以沿用 probe 的內部按鍵佇列。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"

	"github.com/wicanr2/dosgolem/internal/cpu"
	"github.com/wicanr2/dosgolem/internal/dos"
	"github.com/wicanr2/dosgolem/internal/machine"
	"github.com/wicanr2/dosgolem/internal/state"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
)

type segment struct {
	Name     string   `json:"name"`
	From     string   `json:"from"`
	Keys     []string `json:"keys"`
	KeyAt    uint64   `json:"key_at"`
	KeyEvery uint64   `json:"key_every"`
	Holds    []string `json:"holds"`
}
type fingerprint struct {
	R                   [8]uint16
	Seg                 [4]uint16
	IP, Flags           uint16
	Steps, Cycles, IRQ1 uint64
	RAM, Frame          string
}

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
func region(b []byte) []byte { p, e := pbl.Region(b, 320, 200, 32, 152, 24, 32); check(e); return p }
func fp(m *machine.Machine) fingerprint {
	return fingerprint{m.CPU.R, m.CPU.Seg, m.CPU.IP, m.CPU.Flags, m.Steps, m.CPU.Cycles, m.IRQ1Delivered(), hash(m.Snapshot().Mem()), hash(m.Indexed())}
}

func main() {
	name := flag.String("segment", "", "12-battle2 或14-minton1")
	out := flag.String("out", "", "全新輸出前綴")
	flag.Parse()
	cfgs := map[string]struct {
		start, sha string
		stop       uint64
	}{
		"12-battle2": {"10-saved", "abdbb98f66010d5680d3f3bfecead2c28ffd4f7f6fd625fc3b3103a62bcbd056", 280000000},
		"14-minton1": {"13-healed", "d9c18d64a60ed678d34d72f00c199af51108c2a36931fdcaed650a110f23e7a0", 420000000},
	}
	cfg, ok := cfgs[*name]
	if !ok || *out == "" {
		log.Fatal("必須指定支援的正常段落及輸出")
	}
	paths, e := filepath.Glob(*out + "*")
	check(e)
	if len(paths) != 0 {
		log.Fatal("拒絕覆寫輸出")
	}
	const orig = "/orig/psychic-war"
	const replay = "replay/title-to-first-save.json"
	start := "workplace/states/" + cfg.start + ".state"
	inputs := map[string]string{}
	for _, p := range []string{start, replay, orig + "/PW.EXE", "tools/hd/observe_next_bodies.go"} {
		inputs[p] = hash(read(p))
	}
	if inputs[start] != cfg.sha || inputs[orig+"/PW.EXE"] != "88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49" {
		log.Fatal("固定輸入不符")
	}
	var plan struct {
		Schema   string    `json:"schema"`
		Segments []segment `json:"segments"`
	}
	check(json.Unmarshal(read(replay), &plan))
	if plan.Schema != "psychic-war-replay/1" {
		log.Fatal("重播格式不符")
	}
	var route segment
	for _, s := range plan.Segments {
		if s.Name == *name {
			route = s
		}
	}
	if route.From != cfg.start || route.KeyEvery != 3000000 || len(route.Holds) != 1 {
		log.Fatal("正常路線契約不符")
	}
	if (*name == "12-battle2" && route.Holds[0] != "space@238000000+30000000") || (*name == "14-minton1" && route.Holds[0] != "space@382200000+30000000") {
		log.Fatal("攻擊邊界不符")
	}
	poses := map[string][]byte{}
	for n := 0; n < 12; n++ {
		p := fmt.Sprintf("%s/ENEMY%02d.PBL", orig, n)
		b := read(p)
		inputs[p] = hash(b)
		if len(b) < 2 {
			log.Fatal("PBL標頭不足")
		}
		count := int(binary.LittleEndian.Uint16(b[:2])) / 2
		if count != 30 {
			log.Fatal("圖數不符")
		}
		for id := 0; id < count; id++ {
			w, h, px, e := pbl.Decode(b, id)
			check(e)
			if w == 24 && h == 32 {
				poses[fmt.Sprintf("ENEMY%02d:%d", n, id)] = px
			}
		}
	}
	matches := func(px []byte) []string {
		ids := []string{}
		for n := 0; n < 12; n++ {
			for id := 0; id < 30; id++ {
				key := fmt.Sprintf("ENEMY%02d:%d", n, id)
				if bytes.Equal(px, poses[key]) {
					ids = append(ids, key)
				}
			}
		}
		return ids
	}
	scans := map[string]uint8{"up": 0x48, "right": 0x4d, "down": 0x50, "left": 0x4b, "enter": 0x1c}
	newMachine := func() (*machine.Machine, *dos.DOS, uint16) {
		m := machine.New()
		d := dos.New(m, orig)
		d.Install()
		// 與 probe 相同：先設定 FIFO，再載入state；state不序列化鍵盤FIFO。
		m.KeyEvery = route.KeyEvery
		m.SetNextKey(route.KeyAt)
		for _, key := range route.Keys {
			sc, ok := scans[key]
			if !ok {
				log.Fatal("未支援按鍵")
			}
			m.QueueKey(sc)
		}
		check(state.Load(start, m, d))
		d.Root = orig
		d.Scratch = "/tmp/pw-next-bodies"
		if *name == "12-battle2" {
			m.HoldKey(0x39, 238000000, 30000000, true)
		} else {
			m.HoldKey(0x39, 382200000, 30000000, true)
		}
		seed := m.Read16(0x1610 + 0x41df)
		return m, d, seed
	}
	m, d, seed := newMachine()
	defer d.Close()
	var before, raw []byte
	var enter uint64
	var regs map[string]uint16
	events := []map[string]any{}
	total := 0
	for m.Steps < cfg.stop {
		c := m.CPU
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x8705 && c.R[cpu.CX] == 0x0826 && c.R[cpu.DX] == 0x0304 {
			total++
			if len(events) < 16 {
				if before != nil {
					log.Fatal("巢狀貼圖")
				}
				enter = m.Steps
				before = m.Indexed()
				regs = map[string]uint16{"AX": c.R[cpu.AX], "BX": c.R[cpu.BX], "CX": c.R[cpu.CX], "DX": c.R[cpu.DX], "DS": c.Seg[cpu.DS]}
				a := uint32(c.Seg[cpu.DS])*16 + uint32(c.R[cpu.BX])
				if a > 0xa0000-384 {
					log.Fatal("來源超過RAM")
				}
				raw = append([]byte(nil), m.Mem[a:a+384]...)
			}
		}
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x8751 && before != nil {
			after := m.Indexed()
			from, to := region(before), region(after)
			mode := regs["AX"] & 255
			mismatch, outside := 0, 0
			for i, v := range to {
				want := raw[i/2] & 15
				if i%2 == 0 {
					want = raw[i/2] >> 4
				}
				if mode == 1 {
					want ^= from[i]
				}
				if v != want {
					mismatch++
				}
			}
			for i, v := range after {
				if !(i%320 >= 32 && i%320 < 56 && i/320 >= 152 && i/320 < 184) && v != before[i] {
					outside++
				}
			}
			if mode > 1 || mismatch != 0 || outside != 0 {
				log.Fatal("原版貼圖契約不符")
			}
			p := fmt.Sprintf("%s-event%02d", *out, len(events))
			write(p+"-before.frame", before)
			write(p+"-after.frame", after)
			write(p+"-source.bin", raw)
			check(state.Save(p+".state", m, d))
			events = append(events, map[string]any{"entry_step": enter, "return_step": m.Steps, "entry_regs": regs, "al": mode, "from_images": matches(from), "to_images": matches(to), "before_sha256": hash(before), "after_sha256": hash(after), "source_sha256": hash(raw), "source_mismatch": mismatch, "outside_changed_pixels": outside})
			before = nil
		}
		if d.Exited || m.CPU.Halted {
			log.Fatal("正常路徑意外終止")
		}
		check(m.Step())
	}
	if len(events) == 0 || before != nil {
		log.Fatal("沒有身體樣本或尚在貼圖")
	}
	observed := fp(m)
	control, cd, cs := newMachine()
	defer cd.Close()
	if cs != seed {
		log.Fatal("對照seed不同")
	}
	for control.Steps < cfg.stop {
		check(control.Step())
	}
	unobserved := fp(control)
	if !reflect.DeepEqual(observed, unobserved) {
		log.Fatal("觀察干擾原版終點")
	}
	write(*out+"-end.frame", m.Indexed())
	doc := map[string]any{"tool": "dosgolem f8c1a6e 原版正常重播，非正式HD接入", "go_version": runtime.Version(), "inputs_sha256": inputs, "address_space": "執行期CS:IP、DS:BX；320×200原版色號座標", "route": route, "seed_before": fmt.Sprintf("%04X", seed), "seed_method": "兩側載入相同固定雜湊state，執行前唯讀核對CS:41DF", "normal_replay_end_step": cfg.stop, "events": events, "total_body_calls": total, "observed_end": observed, "unobserved_end": unobserved, "limits": "僅正常路線身體貼圖，上限16次、實際數量依events；少於16不證明完整循環。不證明全部敵人、效果、HD美術或正式接入"}
	b, e := json.MarshalIndent(doc, "", "  ")
	check(e)
	write(*out+".json", append(b, '\n'))
	fmt.Printf("%s：%d個身體事件、區外不變與觀察終點對照通過；seed %04X\n", *name, len(events), seed)
}
