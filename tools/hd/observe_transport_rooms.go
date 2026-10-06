// 正常交通路線的唯讀貼圖來源探針；入口研究038 §68。
// 透過Go覆映射在dosgolem的probe套件建置，不修改正式執行器。
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
	"runtime"
	"sort"

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

type segment struct {
	Name, From  string
	SaveAt      uint64 `json:"save_at"`
	KeyAt       uint64 `json:"key_at"`
	KeyEvery    uint64 `json:"key_every"`
	Keys, Holds []string
}
type fingerprint struct {
	R                   [8]uint16
	Seg                 [4]uint16
	IP, Flags           uint16
	Steps, Cycles, IRQ1 uint64
	RAM, Bus, Frame     string
}

func fp(m *machine.Machine) fingerprint {
	c := m.CPU
	return fingerprint{c.R, c.Seg, c.IP, c.Flags, m.Steps, c.Cycles, m.IRQ1Delivered(), hash(m.Mem[:0xa0000]), hash(m.Snapshot().Mem()), hash(m.Indexed())}
}

type pose struct {
	Key    string
	W, H   int
	Pixels []byte
}
type event struct {
	SourceKind                           string
	Entry, Return                        uint64
	AX, BX, CX, DX, DS                   uint16
	X, Y, W, H                           int
	Source, Before, After, State         string
	SourceMatches, FullBefore, FullAfter []string
	SourceMismatch, OutsideChanged       int
}

func crop(frame []byte, x, y, w, h int) []byte {
	px, e := pbl.Region(frame, 320, 200, x, y, w, h)
	check(e)
	return px
}

func main() {
	out := flag.String("out", "", "全新輸出前綴")
	flag.Parse()
	if *out == "" {
		log.Fatal("缺輸出")
	}
	paths, e := filepath.Glob(*out + "*")
	check(e)
	if len(paths) > 0 {
		log.Fatal("拒絕覆寫")
	}
	const orig = "/orig/psychic-war"
	const start = "workplace/states/15-minton2.state"
	const expected = "workplace/states/16-sivad.state"
	inputs := map[string]string{}
	for _, p := range []string{start, expected, "replay/title-to-first-save.json", "tools/hd/observe_transport_rooms.go", orig + "/PW.EXE"} {
		inputs[p] = hash(read(p))
	}
	if inputs[start] != "b2a06cb338d9e150a3b8931fdd451381da019668df3c5979672c14d80d49c8a9" || inputs[expected] != "ae4a30d999553d223a6376deb533ee98cd1aaf043cbfb782f0b3eb9cac01df1e" || inputs[orig+"/PW.EXE"] != "88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49" {
		log.Fatal("固定原版來源不符")
	}
	var plan struct{ Segments []segment }
	check(json.Unmarshal(read("replay/title-to-first-save.json"), &plan))
	var route segment
	for _, s := range plan.Segments {
		if s.Name == "16-sivad" {
			route = s
		}
	}
	if route.From != "15-minton2" || route.SaveAt != 614000000 || route.KeyAt != 520500000 || route.KeyEvery != 3000000 || len(route.Holds) != 0 || fmt.Sprint(route.Keys) != "[up up up up up left up down enter]" {
		log.Fatal("正常交通鍵序不符")
	}
	poses := []pose{}
	archives := map[string]int{}
	files, e := filepath.Glob(orig + "/*.PBL")
	check(e)
	sort.Strings(files)
	for _, p := range files {
		b := read(p)
		inputs[p] = hash(b)
		if len(b) < 2 {
			log.Fatal("PBL標頭不足")
		}
		n := int(binary.LittleEndian.Uint16(b[:2])) / 2
		archives[filepath.Base(p)] = n
		for id := 0; id < n; id++ {
			w, h, px, e := pbl.Decode(b, id)
			check(e)
			poses = append(poses, pose{fmt.Sprintf("%s:%d", filepath.Base(p), id), w, h, px})
		}
	}
	match := func(px []byte, w, h int) []string {
		ids := []string{}
		for _, p := range poses {
			if p.W == w && p.H == h && bytes.Equal(px, p.Pixels) {
				ids = append(ids, p.Key)
			}
		}
		return ids
	}
	newMachine := func() (*machine.Machine, *dos.DOS) {
		m := machine.New()
		d := dos.New(m, orig)
		d.Install()
		m.KeyEvery = route.KeyEvery
		m.SetNextKey(route.KeyAt)
		scans := map[string]uint8{"up": 0x48, "down": 0x50, "right": 0x4d, "left": 0x4b, "enter": 0x1c}
		for _, k := range route.Keys {
			m.QueueKey(scans[k])
		}
		check(state.Load(start, m, d))
		d.Root = orig
		d.Scratch = "/tmp/pw-transport-normal"
		return m, d
	}
	m, d := newMachine()
	defer d.Close()
	initial := fp(m)
	seed := m.Read16(0x1610 + 0x41df)
	location := func() map[string]uint16 {
		return map[string]uint16{"area": m.Read16(0x16966), "x": m.Read16(0x16968), "y": m.Read16(0x1696a), "facing": m.Read16(0x16970), "hp": m.Read16(0x16990), "energy": m.Read16(0x16994)}
	}
	startLocation := location()
	events := []event{}
	otherRLE := []map[string]any{}
	keys := []map[string]uint64{}
	var pending *event
	var before, raw, unpacked []byte
	var returnIP, returnSP, returnSS uint16
	for m.Steps < route.SaveAt {
		c := m.CPU
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x8705 && c.R[cpu.CX] == 0x011f {
			if pending != nil || len(events) >= 256 {
				log.Fatal("巢狀或超過256次有界貼圖")
			}
			w, h := int(c.R[cpu.DX]>>8)*8, int(c.R[cpu.DX]&255)*8
			x, y := int(c.R[cpu.CX]>>8)*4, int(c.R[cpu.CX]&255)*4
			if w < 8 || h < 8 || x+w > 320 || y+h > 200 || c.R[cpu.AX]&255 > 1 {
				log.Fatalf("未支援貼圖範圍或模式：step=%d AX=%04X CX=%04X DX=%04X", m.Steps, c.R[cpu.AX], c.R[cpu.CX], c.R[cpu.DX])
			}
			a := uint32(c.Seg[cpu.DS])*16 + uint32(c.R[cpu.BX])
			n := uint32(w * h / 2)
			if a+n > 0xa0000 {
				log.Fatal("來源RAM越界")
			}
			raw = append([]byte(nil), m.Mem[a:a+n]...)
			unpacked = make([]byte, w*h)
			for i, b := range raw {
				unpacked[2*i] = b >> 4
				unpacked[2*i+1] = b & 15
			}
			before = m.Indexed()
			pending = &event{SourceKind: "packed", Entry: m.Steps, AX: c.R[cpu.AX], BX: c.R[cpu.BX], CX: c.R[cpu.CX], DX: c.R[cpu.DX], DS: c.Seg[cpu.DS], X: x, Y: y, W: w, H: h, SourceMatches: match(unpacked, w, h), FullBefore: match(crop(before, x, y, w, h), w, h)}
		}
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x8588 && c.R[cpu.CX] != 0x011f {
			otherRLE = append(otherRLE, map[string]any{"step": m.Steps, "ds": c.Seg[cpu.DS], "bx": c.R[cpu.BX], "cx": c.R[cpu.CX], "ax": c.R[cpu.AX], "level": "未知，本次只研究房間矩形；不宣稱其他呼叫為原始PBL copy"})
		}
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x8588 && c.R[cpu.CX] == 0x011f {
			if pending != nil || len(events) >= 256 {
				log.Fatal("巢狀或超過256次串流貼圖")
			}
			a := uint32(c.Seg[cpu.DS])*16 + uint32(c.R[cpu.BX])
			if a+0x1388 > 0xa0000 {
				log.Fatal("RLE來源越界")
			}
			blob := m.Mem[a : a+0x1388]
			w, h := int(blob[0])*8, int(blob[1])*8
			x, y := int(c.R[cpu.CX]>>8)*4, int(c.R[cpu.CX]&255)*4
			if w < 8 || h < 8 || x+w > 320 || y+h > 200 {
				log.Fatal("未支援RLE繪製矩形")
			}
			// 只保存本次實際消耗的RLE來源。
			si, n, prev := 19, 0, blob[18]
			for n < w*h/2 {
				if si >= len(blob) {
					log.Fatal("RLE輸入不足")
				}
				cur := blob[si]
				si++
				if cur == prev {
					if si >= len(blob) {
						log.Fatal("RLE計數不足")
					}
					n += int(blob[si])
					si++
					if n >= w*h/2 {
						break
					}
					if si >= len(blob) {
						log.Fatal("RLE後續不足")
					}
					prev = blob[si]
					si++
				} else {
					n++
					prev = cur
				}
			}
			raw = append([]byte(nil), blob[:si]...)
			archive := append([]byte{2, 0}, raw...)
			dw, dh, px, e := pbl.Decode(archive, 0)
			check(e)
			if dw != w || dh != h {
				log.Fatal("RLE尺寸不符")
			}
			unpacked = px
			before = m.Indexed()
			pending = &event{SourceKind: "rle-copy", Entry: m.Steps, AX: c.R[cpu.AX], BX: c.R[cpu.BX], CX: c.R[cpu.CX], DX: c.R[cpu.DX], DS: c.Seg[cpu.DS], X: x, Y: y, W: w, H: h, SourceMatches: match(unpacked, w, h), FullBefore: match(crop(before, x, y, w, h), w, h)}
			returnSS = c.Seg[cpu.SS]
			returnSP = c.R[cpu.SP] + 2
			returnIP = m.Read16(uint32(returnSS)*16 + uint32(c.R[cpu.SP]))
		}
		if pending != nil && c.Seg[cpu.CS] == 0x161 && ((pending.SourceKind == "packed" && c.IP == 0x8751) || (pending.SourceKind == "rle-copy" && c.IP == returnIP && c.Seg[cpu.SS] == returnSS && c.R[cpu.SP] == returnSP)) {
			v := pending
			after := m.Indexed()
			to := crop(after, v.X, v.Y, v.W, v.H)
			from := crop(before, v.X, v.Y, v.W, v.H)
			for i, b := range to {
				want := unpacked[i]
				if v.SourceKind == "packed" && v.AX&255 == 1 {
					want ^= from[i]
				}
				if b != want {
					v.SourceMismatch++
				}
			}
			for i, b := range after {
				if !(v.X <= i%320 && i%320 < v.X+v.W && v.Y <= i/320 && i/320 < v.Y+v.H) && b != before[i] {
					v.OutsideChanged++
				}
			}
			if v.SourceMismatch != 0 || v.OutsideChanged != 0 {
				p := *out + "-failure"
				write(p+"-source.bin", raw)
				write(p+"-before.frame", before)
				write(p+"-after.frame", after)
				check(state.Save(p+".state", m, d))
				jsonWrite(p+".json", map[string]any{"event": v, "inputs_sha256": inputs, "seed_before": fmt.Sprintf("%04X", seed), "completed_events": events, "steps": m.Steps, "original": fp(m)})
				log.Fatal("原版貼圖語意或區外不符")
			}
			p := fmt.Sprintf("%s-event%03d", *out, len(events))
			v.Return = m.Steps
			v.Source = p + "-source.bin"
			v.Before = p + "-before.frame"
			v.After = p + "-after.frame"
			v.FullAfter = match(to, v.W, v.H)
			write(v.Source, raw)
			write(v.Before, before)
			write(v.After, after)
			for _, id := range v.FullAfter {
				if len(id) >= 4 && id[:4] == "ROOM" {
					v.State = p + ".state"
					check(state.Save(v.State, m, d))
					break
				}
			}
			events = append(events, *v)
			pending = nil
		}
		at, irq := m.Steps, m.IRQ1Delivered()
		check(m.Step())
		if m.IRQ1Delivered() != irq {
			keys = append(keys, map[string]uint64{"step": at, "ordinal": uint64(len(keys))})
		}
		if d.Exited || c.Halted {
			log.Fatal("原版路線意外終止")
		}
	}
	if pending != nil || len(events) == 0 || len(keys) != 18 {
		log.Fatalf("貼圖／按鍵終止條件未通過：events=%d pending=%v IRQ1=%d end=%d", len(events), pending != nil, len(keys), m.Steps)
	}
	observed := fp(m)
	endLocation := location()
	write(*out+"-end.frame", m.Indexed())
	check(state.Save(*out+"-end.state", m, d))
	control, cd := newMachine()
	defer cd.Close()
	if fp(control) != initial || control.Read16(0x1610+0x41df) != seed {
		log.Fatal("無觀察控制起點不符")
	}
	for control.Steps < route.SaveAt {
		check(control.Step())
	}
	if fp(control) != observed {
		log.Fatal("唯讀觀察干擾原版")
	}
	baseline := machine.New()
	bd := dos.New(baseline, orig)
	bd.Install()
	defer bd.Close()
	check(state.Load(expected, baseline, bd))
	a, b := fp(baseline), observed
	a.IRQ1 = b.IRQ1
	if a != b {
		log.Fatal("既有原版終點不符，磁碟state不保存IRQ1計數")
	}
	jsonWrite(*out+".json", map[string]any{"tool": "dosgolem 原版正常FIFO路線，唯讀觀察", "go_version": runtime.Version(), "inputs_sha256": inputs, "archives": archives, "pose_count": len(poses), "address_space": "執行期CS:IP 0161:8588及其stack返回、0161:8705／8751、DS:BX原始打包來源；320×200原版像素座標", "route": route, "seed_before": fmt.Sprintf("%04X", seed), "seed_method": "同一固定雜湊state，執行前唯讀核對CS:41DF；未修改或重擲", "initial": initial, "start_location": startLocation, "end_location": endLocation, "events": events, "other_rle_calls_unknown": otherRLE, "keys_irq1": keys, "observed_end": observed, "control_end": fp(control), "existing_state_equal": true, "limits": "只證明本條正常交通路線房間矩形(4,124)的來源／位置／copy或XOR呼叫及無觀察終點一致；其餘RLE呼叫不推定模式，不證明全部房間、HD接入、美術、中文或GUI"})
	fmt.Printf("正常交通路線：%d貼圖、%d鍵IRQ1、seed%04X，既有與無觀察原版終點相同\n", len(events), len(keys), seed)
}
