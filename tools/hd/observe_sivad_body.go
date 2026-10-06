// Sivad已存正常state → 兩次Up的有界原版身體探針；入口研究038 §50。
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
	const end = uint64(695000000)
	inputs := map[string]string{}
	for _, p := range []string{start, orig + "/PW.EXE", "tools/hd/observe_sivad_body.go", "replay/title-to-first-save.json"} {
		inputs[p] = hash(read(p))
	}
	if inputs[start] != "3c2ae7470600c61217060dc76843aed7c4097ac6e4d80a778cf3282b8afbc7e4" || inputs[orig+"/PW.EXE"] != "88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49" {
		log.Fatal("固定來源不符")
	}
	poses := []pose{}
	for n := 0; n < 12; n++ {
		p := fmt.Sprintf("%s/ENEMY%02d.PBL", orig, n)
		b := read(p)
		inputs[p] = hash(b)
		if len(b) < 2 || binary.LittleEndian.Uint16(b[:2])/2 != 30 {
			log.Fatal("PBL圖數不符")
		}
		for id := 0; id < 15; id++ {
			w, h, px, e := pbl.Decode(b, id)
			check(e)
			poses = append(poses, pose{fmt.Sprintf("ENEMY%02d:%d", n, id), w, h, px})
		}
	}
	matches := func(frame []byte, w, h int) []string {
		px, e := pbl.Region(frame, 320, 200, 32, 152, w, h)
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
	codes := []uint8{0x48, 0xc8, 0x48, 0xc8}
	events := []bodyEvent{}
	samples := []sample{}
	var pending *bodyEvent
	var before, raw []byte
	total, entered, returned := 0, 0, 0
	for m.Steps < end {
		c := m.CPU
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x47d4 {
			entered++
		}
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x47ad {
			returned++
		}
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x8705 && c.R[cpu.CX] == 0x0826 && c.R[cpu.DX]>>8 == 3 && (c.R[cpu.DX]&255 == 3 || c.R[cpu.DX]&255 == 4) {
			total++
			if len(events) < 16 {
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
		if pending != nil && len(events) >= 1 && len(events) <= 4 && (m.Steps == pending.Entry+1 || m.Steps == pending.Entry+512) {
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
	jsonWrite(*out+".json", map[string]any{"tool": "dosgolem f8c1a6e原版；正常FIFO方向鍵，無HD實作", "go_version": runtime.Version(), "inputs_sha256": inputs, "address_space": "原版執行期CS:IP／DS:BX；320×200色號座標", "start": start, "seed_before": fmt.Sprintf("%04X", seed), "seed_method": "兩側同雜湊state，執行前唯讀核對CS:41DF，未重擲", "key_at": 655500000, "key_every": 3000000, "route": []string{"up", "up"}, "keys_irq1": tape, "end_step": end, "start_location": startLocation, "end_location": location(), "battle_entries": entered, "battle_returns": returned, "total_body_calls": total, "events": events, "samples": samples, "initial": initial, "observed_end": observed, "control_end": baseline, "limits": "新Sivad正常方向鍵來源，最多16筆身體及8中途樣本；無新攻擊鍵，不證明全部姿勢／特效、中文或正式HD接入"})
	fmt.Printf("Sivad：身體%d/%d，戰鬥進入%d／返回%d，方向鍵%d，觀察終點相同；seed %04X\n", len(events), total, entered, returned, len(tape), seed)
}
