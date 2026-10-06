// Sivad已存正常state → 原版陣亡結束人物的有界來源探針；入口研究038 §51。
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
func jsonWrite(p string, v any) {
	b, e := json.MarshalIndent(v, "", "  ")
	check(e)
	write(p, append(b, '\n'))
}

type fingerprint struct {
	R                   [8]uint16
	Seg                 [4]uint16
	IP, Flags           uint16
	Steps, Cycles, IRQ1 uint64
	RAM, Frame          string
}

func fp(m *machine.Machine) fingerprint {
	c := m.CPU
	return fingerprint{c.R, c.Seg, c.IP, c.Flags, m.Steps, c.Cycles, m.IRQ1Delivered(), hash(m.Mem[:0xa0000]), hash(m.Indexed())}
}

type keyEvent struct {
	Step uint64
	Code uint8
}
type bodyEvent struct {
	Entry, Return                uint64
	AX, BX, CX, DX, DS           uint16
	Before, After, Source, State string
	FullBefore, FullAfter        []string
	W, H                         int
}
type sample struct {
	Step         uint64
	Event        int
	Kind, Prefix string
}
type pose struct {
	Key    string
	W, H   int
	Pixels []byte
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
	const start = "workplace/states/17-saved2.state"
	const end = uint64(705000000)
	inputs := map[string]string{}
	for _, p := range []string{start, orig + "/PW.EXE", "tools/hd/observe_over_scene.go", "replay/title-to-first-save.json"} {
		inputs[p] = hash(read(p))
	}
	if inputs[start] != "3c2ae7470600c61217060dc76843aed7c4097ac6e4d80a778cf3282b8afbc7e4" || inputs[orig+"/PW.EXE"] != "88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49" {
		log.Fatal("固定來源不符")
	}
	poses := []pose{}
	p := orig + "/OVER.PBL"
	b := read(p)
	inputs[p] = hash(b)
	if len(b) < 2 || binary.LittleEndian.Uint16(b[:2])/2 != 1 {
		log.Fatal("OVER圖數不符")
	}
	w, h, px, e := pbl.Decode(b, 0)
	check(e)
	if w != 64 || h != 64 {
		log.Fatal("OVER尺寸不符")
	}
	poses = append(poses, pose{"OVER:0", w, h, px})
	matches := func(frame []byte, w, h int) []string {
		px, e := pbl.Region(frame, 320, 200, 128, 48, w, h)
		check(e)
		ids := []string{}
		for _, p := range poses {
			if p.W == w && p.H == h && bytes.Equal(px, p.Pixels) {
				ids = append(ids, p.Key)
			}
		}
		return ids
	}
	newMachine := func() (*machine.Machine, *dos.DOS, uint16) {
		m := machine.New()
		d := dos.New(m, orig)
		d.Install()
		m.KeyEvery = 3000000
		m.SetNextKey(655500000)
		m.QueueKey(0x48)
		m.QueueKey(0x48)
		check(state.Load(start, m, d))
		d.Root = orig
		d.Scratch = "/tmp/pw-sivad-body"
		m.HoldKey(0x1c, 695500000, 500000, false)
		return m, d, m.Read16(0x1610 + 0x41df)
	}
	m, d, seed := newMachine()
	defer d.Close()
	initial := fp(m)
	location := func() map[string]uint16 {
		return map[string]uint16{"area": m.Read16(0x16966), "x": m.Read16(0x16968), "y": m.Read16(0x1696a), "facing": m.Read16(0x16970), "player_hp": m.Read16(0x16990), "energy": m.Read16(0x16994)}
	}
	startLocation := location()
	if startLocation["area"] != 1 || startLocation["x"] != 5 || startLocation["y"] != 6 {
		log.Fatal("正常Sivad起點不同")
	}
	tape := []keyEvent{}
	codes := []uint8{0x48, 0xc8, 0x48, 0xc8, 0x1c, 0x9c}
	events := []bodyEvent{}
	samples := []sample{}
	var pending *bodyEvent
	var before, raw []byte
	total, entered, returned := 0, 0, 0
	allCalls := []map[string]any{}
	sceneSamples := []map[string]any{}
	lastFull := false
	firstFull := uint64(0)
	lastBefore := []byte(nil)

	for m.Steps < end {
		c := m.CPU
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x8705 {
			if len(allCalls) >= 2048 {
				log.Fatal("一般貼圖摘要超過有界限制")
			}
			allCalls = append(allCalls, map[string]any{"step": m.Steps, "AX": c.R[cpu.AX], "CX": c.R[cpu.CX], "DX": c.R[cpu.DX], "DS": c.Seg[cpu.DS], "BX": c.R[cpu.BX]})
		}
		if m.Steps%100000 == 0 {
			frame := m.Indexed()
			full := len(matches(frame, 64, 64)) == 1
			if full != lastFull || (firstFull != 0 && m.Steps == 695000000) {
				p := fmt.Sprintf("%s-scene%02d", *out, len(sceneSamples))
				write(p+".frame", frame)
				check(state.Save(p+".state", m, d))
				if full && firstFull == 0 {
					firstFull = m.Steps
					write(p+"-previous.frame", lastBefore)
				}
				sceneSamples = append(sceneSamples, map[string]any{"step": m.Steps, "prefix": p, "full_over": full})
			}
			lastBefore = frame
			lastFull = full
		}
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x47d4 {
			entered++
		}
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x47ad {
			returned++
		}
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x8705 && c.R[cpu.CX] == 0x200c && c.R[cpu.DX] == 0x0808 {
			total++
			if len(events) < 4 {
				if pending != nil {
					log.Fatal("巢狀身體貼圖")
				}
				w, h := int(c.R[cpu.DX]>>8)*8, int(c.R[cpu.DX]&255)*8
				pending = &bodyEvent{Entry: m.Steps, AX: c.R[cpu.AX], BX: c.R[cpu.BX], CX: c.R[cpu.CX], DX: c.R[cpu.DX], DS: c.Seg[cpu.DS], W: w, H: h}
				before = m.Indexed()
				addr := uint32(pending.DS)*16 + uint32(pending.BX)
				size := uint32(w * h / 2)
				if addr > 0xa0000-size {
					log.Fatal("來源越界")
				}
				raw = append([]byte(nil), m.Mem[addr:addr+size]...)
				pending.FullBefore = matches(before, w, h)
			}
		}
		if pending != nil && (m.Steps == pending.Entry+1 || m.Steps == pending.Entry+512 || m.Steps == pending.Entry+10000 || m.Steps == pending.Entry+50000) {
			p := fmt.Sprintf("%s-partial%02d-%d", *out, len(events), m.Steps-pending.Entry)
			write(p+".frame", m.Indexed())
			check(state.Save(p+".state", m, d))
			samples = append(samples, sample{m.Steps, len(events), "partial", p})
		}
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x8751 && pending != nil {
			i := len(events)
			p := fmt.Sprintf("%s-event%02d", *out, i)
			pending.Return = m.Steps
			pending.Before = p + "-before.frame"
			pending.After = p + ".frame"
			pending.Source = p + "-source.bin"
			pending.State = p + ".state"
			after := m.Indexed()
			pending.FullAfter = matches(after, pending.W, pending.H)
			write(pending.Before, before)
			write(pending.After, after)
			write(pending.Source, raw)
			check(state.Save(pending.State, m, d))
			events = append(events, *pending)
			samples = append(samples, sample{m.Steps, i, "complete", p})
			pending = nil
		}
		previous := m.IRQ1Delivered()
		at := m.Steps
		check(m.Step())
		if m.IRQ1Delivered() != previous {
			if len(tape) >= len(codes) {
				log.Fatal("非預期IRQ1")
			}
			tape = append(tape, keyEvent{at, codes[len(tape)]})
		}
	}
	if len(tape) != len(codes) || pending != nil {
		log.Fatal("方向鍵未送完或中途貼圖未返回")
	}
	observed := fp(m)
	write(*out+"-end.frame", m.Indexed())
	check(state.Save(*out+"-end.state", m, d))
	control, cd, controlSeed := newMachine()
	defer cd.Close()
	if controlSeed != seed || fp(control) != initial {
		log.Fatal("對照初始狀態不同")
	}
	for control.Steps < end {
		check(control.Step())
	}
	baseline := fp(control)
	if !reflect.DeepEqual(observed, baseline) {
		log.Fatal("觀察干擾原版")
	}
	jsonWrite(*out+".json", map[string]any{"tool": "dosgolem f8c1a6e原版；正常FIFO方向鍵，無HD實作", "go_version": runtime.Version(), "inputs_sha256": inputs, "address_space": "原版執行期CS:IP／DS:BX；320×200色號座標", "start": start, "seed_before": fmt.Sprintf("%04X", seed), "seed_method": "兩側同雜湊state，執行前唯讀核對CS:41DF，未重擲", "key_at": 655500000, "key_every": 3000000, "route": []string{"up", "up"}, "keys_irq1": tape, "end_step": end, "start_location": startLocation, "end_location": location(), "battle_entries": entered, "battle_returns": returned, "total_over_calls": total, "events": events, "samples": samples, "all_blit_calls": allCalls, "scene_samples": sceneSamples, "first_full_step": firstFull, "enter_at": 695500000, "enter_duration": 500000, "initial": initial, "observed_end": observed, "control_end": baseline, "limits": "原版Sivad陣亡結束人物來源，逐100000步原版完整OVER出現觀察；695500000正常Enter，尚未驗HD／中文或全部結束圖像"})
	fmt.Printf("原版OVER：人物%d/%d，戰鬥進入%d／返回%d，方向鍵%d，觀察終點相同；seed %04X\n", len(events), total, entered, returned, len(tape), seed)
}
