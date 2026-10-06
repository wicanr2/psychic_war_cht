// 原版14-minton1正常攻擊的小圖塊來源觀察，研究038 §56。
// 只在臨時dosgolem子模組建置，不修改正式API或原版RAM。
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"syscall"

	"github.com/wicanr2/dosgolem/internal/cpu"
	"github.com/wicanr2/dosgolem/internal/dos"
	"github.com/wicanr2/dosgolem/internal/machine"
	"github.com/wicanr2/dosgolem/internal/state"
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

type fingerprint struct {
	R                   [8]uint16
	Seg                 [4]uint16
	IP, Flags           uint16
	Steps, Cycles, IRQ1 uint64
	RAM, Frame          string
}

func fp(m *machine.Machine) fingerprint {
	return fingerprint{m.CPU.R, m.CPU.Seg, m.CPU.IP, m.CPU.Flags, m.Steps, m.CPU.Cycles, m.IRQ1Delivered(), hash(m.Snapshot().Mem()), hash(m.Indexed())}
}

type route struct {
	Name     string   `json:"name"`
	From     string   `json:"from"`
	Keys     []string `json:"keys"`
	KeyAt    uint64   `json:"key_at"`
	KeyEvery uint64   `json:"key_every"`
	Holds    []string `json:"holds"`
}
type event struct {
	Entry      uint64            `json:"entry_step"`
	Return     uint64            `json:"return_step"`
	Regs       map[string]uint16 `json:"entry_regs"`
	Rect       [4]int            `json:"rect"`
	AL         uint16            `json:"al"`
	Before     string            `json:"before"`
	After      string            `json:"after"`
	Source     string            `json:"source"`
	BeforeHash string            `json:"before_sha256"`
	AfterHash  string            `json:"after_sha256"`
	SourceHash string            `json:"source_sha256"`
	Mismatch   int               `json:"source_mismatch"`
	Outside    int               `json:"outside_changed_pixels"`
	Changed    int               `json:"changed_pixels"`
}

func main() {
	out := flag.String("out", "", "全新本機輸出前綴")
	flag.Parse()
	if *out == "" {
		log.Fatal("必須指定輸出")
	}
	paths, e := filepath.Glob(*out + "*")
	check(e)
	if len(paths) != 0 {
		log.Fatal("拒絕覆寫")
	}
	st, e := os.Stat(filepath.Dir(*out))
	check(e)
	u := st.Sys().(*syscall.Stat_t)
	if u.Uid != 1000 || u.Gid != 1000 {
		log.Fatal("輸出目錄UID/GID不符")
	}
	const orig = "/orig/psychic-war"
	const start = "workplace/states/13-healed.state"
	const prior = "workplace/hd/next-body-minton-v2-20261001.json"
	const replay = "replay/title-to-first-save.json"
	const stop = uint64(420000000)
	inputs := map[string]string{}
	for _, p := range []string{start, prior, replay, orig + "/PW.EXE", "tools/hd/observe_small_projectiles.go", "go.mod", "go.sum", "worktrees/dosgolem/go.mod"} {
		inputs[p] = hash(read(p))
	}
	if inputs[start] != "d9c18d64a60ed678d34d72f00c199af51108c2a36931fdcaed650a110f23e7a0" || inputs[orig+"/PW.EXE"] != "88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49" {
		log.Fatal("固定來源不符")
	}
	var plan struct {
		Schema   string  `json:"schema"`
		Segments []route `json:"segments"`
	}
	check(json.Unmarshal(read(replay), &plan))
	var cfg route
	for _, r := range plan.Segments {
		if r.Name == "14-minton1" {
			cfg = r
		}
	}
	if plan.Schema != "psychic-war-replay/1" || cfg.From != "13-healed" || cfg.KeyAt != 345500000 || cfg.KeyEvery != 3000000 || !reflect.DeepEqual(cfg.Keys, []string{"up", "right", "up", "up", "up", "up", "up"}) || !reflect.DeepEqual(cfg.Holds, []string{"space@382200000+30000000"}) {
		log.Fatal("正常路線契約不符")
	}
	var old struct {
		Observed   fingerprint `json:"observed_end"`
		Unobserved fingerprint `json:"unobserved_end"`
	}
	check(json.Unmarshal(read(prior), &old))
	if old.Observed != old.Unobserved || old.Observed.Steps != stop {
		log.Fatal("既有終點收據不符")
	}
	newMachine := func() (*machine.Machine, *dos.DOS) {
		m := machine.New()
		d := dos.New(m, orig)
		d.Install()
		m.KeyEvery = cfg.KeyEvery
		m.SetNextKey(cfg.KeyAt)
		scans := map[string]uint8{"up": 0x48, "right": 0x4d}
		for _, k := range cfg.Keys {
			m.QueueKey(scans[k])
		}
		check(state.Load(start, m, d))
		d.Root = orig
		d.Scratch = "/tmp/pw-small-projectiles"
		m.HoldKey(0x39, 382200000, 30000000, true)
		if m.Read16(0x1610+0x41df) != 0xa48c {
			log.Fatal("執行前seed不符")
		}
		return m, d
	}
	m, d := newMachine()
	defer d.Close()
	initial := m.Steps
	events := []event{}
	geometry := map[string]int{}
	var pending *event
	var before, raw []byte
	for m.Steps < stop {
		c := m.CPU
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x8705 {
			x, y, w, h := int(c.R[cpu.CX]>>8)*4, int(c.R[cpu.CX]&255)*4, int(c.R[cpu.DX]>>8)*8, int(c.R[cpu.DX]&255)*8
			geometry[fmt.Sprintf("%d,%d,%d,%d,AL=%02X", x, y, w, h, c.R[cpu.AX]&255)]++
			if pending != nil {
				log.Fatal("巢狀貼圖")
			}
			if (w == 16 && h == 16) || (w == 24 && h == 32) {
				if x+w > 320 || y+h > 200 || len(events) >= 4096 {
					log.Fatal("超過觀察範圍")
				}
				a := uint32(c.Seg[cpu.DS])*16 + uint32(c.R[cpu.BX])
				size := w * h / 2
				if a > 0xa0000-uint32(size) {
					log.Fatal("來源超過RAM")
				}
				raw = append([]byte(nil), m.Mem[a:a+uint32(size)]...)
				before = m.Indexed()
				pending = &event{Entry: m.Steps, Regs: map[string]uint16{"AX": c.R[cpu.AX], "BX": c.R[cpu.BX], "CX": c.R[cpu.CX], "DX": c.R[cpu.DX], "DS": c.Seg[cpu.DS]}, Rect: [4]int{x, y, w, h}, AL: c.R[cpu.AX] & 255}
			}
		}
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x8751 && pending != nil {
			after := m.Indexed()
			r := pending.Rect
			if pending.AL > 1 {
				log.Fatal("未知AL模式，停止推論")
			}
			for i, v := range after {
				want := before[i]
				x, y := i%320-r[0], i/320-r[1]
				if x >= 0 && y >= 0 && x < r[2] && y < r[3] {
					at := y*r[2] + x
					p := raw[at/2] & 15
					if at%2 == 0 {
						p = raw[at/2] >> 4
					}
					if pending.AL == 1 {
						want ^= p
					} else {
						want = p
					}
				} else if v != before[i] {
					pending.Outside++
				}
				if v != want {
					pending.Mismatch++
				}
				if v != before[i] {
					pending.Changed++
				}
			}
			if pending.Mismatch != 0 || pending.Outside != 0 {
				log.Fatal("原版輸出不符")
			}
			p := fmt.Sprintf("%s-event%04d", *out, len(events))
			pending.Before, pending.After, pending.Source = p+"-before.frame", p+"-after.frame", p+"-source.bin"
			write(pending.Before, before)
			write(pending.After, after)
			write(pending.Source, raw)
			pending.BeforeHash, pending.AfterHash, pending.SourceHash = hash(before), hash(after), hash(raw)
			pending.Return = m.Steps
			events = append(events, *pending)
			pending = nil
		}
		if d.Exited || c.Halted {
			log.Fatal("正常路徑意外終止")
		}
		check(m.Step())
	}
	if pending != nil || len(events) == 0 {
		log.Fatal("沒有完整樣本")
	}
	observed := fp(m)
	control, cd := newMachine()
	defer cd.Close()
	for control.Steps < stop {
		check(control.Step())
	}
	unobserved := fp(control)
	if observed != unobserved || observed != old.Observed {
		log.Fatal("觀察分支／既有正常終點不符")
	}
	write(*out+"-end.frame", m.Indexed())
	check(state.Save(*out+"-end.state", m, d))
	exe, e := os.Executable()
	check(e)
	doc := map[string]any{"tool": "dosgolem f8c1a6e 正常14-minton1唯讀觀察", "go_version": runtime.Version(), "binary_sha256": hash(read(exe)), "inputs_sha256": inputs, "address_space": "執行期CS:IP 0161:8705／8751、DS:BX；320×200原版色號座標", "seed_before": "A48C", "seed_method": "執行前同雜湊13-healed.state，兩側唯讀核對CS:41DF；不寫seed或重擲", "route": cfg, "start_steps": initial, "end_steps": stop, "events": events, "all_geometry_counts": geometry, "observed_end": observed, "unobserved_end": unobserved, "prior_end": old.Observed, "limits": "只驗一般16×16及24×32貼圖，不辨識內建位元遮罩；來源配對與方向須由獨立核對器確認。非完整HD／GUI／美術或能力語意驗收"}
	b, e := json.MarshalIndent(doc, "", "  ")
	check(e)
	write(*out+".json", append(b, '\n'))
	fmt.Printf("正常敏頓攻擊：%d次一般貼圖；觀察／不觀察／既有終點相同\n", len(events))
}
