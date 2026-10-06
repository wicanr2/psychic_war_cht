// 原版正常 BBS 路線的唯讀 MAZE 圖塊觀察；入口研究038 §72。
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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
	write(p, append(b, 10))
}
func crop(frame []byte, x, y, w, h int) []byte {
	b := make([]byte, 0, w*h)
	for row := 0; row < h; row++ {
		b = append(b, frame[(y+row)*320+x:(y+row)*320+x+w]...)
	}
	return b
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

type row struct {
	Step       uint64
	ReturnIP   uint16
	SI, DI, DX uint16
	Tile       []byte
}
type event struct {
	Entry, Return                                           uint64
	Slot, X, Y                                              int
	AX, DX, DS, SourcePointer, ReturnIP, ReturnSS, ReturnSP uint16
	LinearSource                                            uint32
	Source                                                  []byte
	FrameOffset                                             int64
	Rows                                                    []row
	SourceMismatch, OutsideChanged                          int
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
	const start = "workplace/states/07-first-play.state"
	const oldFrame = "workplace/hd/room-nearby-explore-v1-20261003-bbs.frame"
	inputs := map[string]string{}
	for _, p := range []string{start, oldFrame, orig + "/PW.EXE", orig + "/MAZE.BIN", "tools/hd/observe_maze_draw.go", "workplace/ida/hd-maze-20261003/decode.json", "workplace/ida/hd-maze-20261003/row.json"} {
		inputs[p] = hash(read(p))
	}
	if inputs[start] != "56c489deaa57cd26fbd4f39f374e87db2f51e7d1275c440c9805321fb09bd532" || inputs[orig+"/PW.EXE"] != "88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49" || inputs[orig+"/MAZE.BIN"] != "8b3b08ef483ff8c64beb04c69c861763c1258afba2348a7058e828bcf4060756" {
		log.Fatal("固定來源不符")
	}
	raw := read(orig + "/MAZE.BIN")
	if len(raw) != 2048 {
		log.Fatal("MAZE尺寸不符")
	}
	var ida struct {
		OriginalRegionBytes string `json:"original_region_bytes"`
	}
	check(json.Unmarshal(read("workplace/ida/hd-maze-20261003/decode.json"), &ida))
	code, e := hex.DecodeString(ida.OriginalRegionBytes)
	check(e)
	newMachine := func() (*machine.Machine, *dos.DOS) {
		m := machine.New()
		d := dos.New(m, orig)
		d.Install()
		check(state.Load(start, m, d))
		d.Root = orig
		d.Scratch = "/tmp/pw-maze-draw"
		m.HoldKey(0x4b, 42500000, 3000000, false)
		m.HoldKey(0x48, 48500000, 3000000, false)
		if !bytes.Equal(m.Mem[0x1610+0x4fd6:0x1610+0x4fd6+len(code)], code) {
			log.Fatal("原版程式與IDA bytes不符")
		}
		return m, d
	}
	m, d := newMachine()
	defer d.Close()
	initial := fp(m)
	seed := binary.LittleEndian.Uint16(m.Mem[0x57ef:0x57f1])
	if m.Steps != 42000000 || seed != 0x86af {
		log.Fatal("原版初始state不符")
	}
	file, e := os.OpenFile(*out+"-frames.bin", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(e)
	stream := bufio.NewWriter(file)
	events := []event{}
	keys := []uint64{}
	var pending *event
	var before []byte
	for m.Steps < 66000000 {
		c := m.CPU
		if c.Seg[cpu.CS] == 0x161 && c.IP == 0x4fd6 {
			if pending != nil || len(events) >= 1024 {
				log.Fatal("巢狀或超過有界圖塊次數")
			}
			id := int(c.R[cpu.AX] & 255)
			x, y := int(c.R[cpu.DX]>>8)*4, int(c.R[cpu.DX]&255)*4
			if x < 4 || y < 124 || x+4 > 76 || y+4 > 196 {
				log.Fatalf("未知圖塊座標 %d,%d", x, y)
			}
			ds := c.Seg[cpu.DS]
			pa := uint32(ds)*16 + 0xaed4
			pointer := binary.LittleEndian.Uint16(m.Mem[pa : pa+2])
			linear := uint32(ds)*16 + uint32(pointer) + uint32(id*8)
			if !bytes.Equal(m.Mem[linear:linear+8], raw[id*8:id*8+8]) {
				log.Fatal("原始slot來源不同")
			}
			ss, sp := c.Seg[cpu.SS], c.R[cpu.SP]
			ret := binary.LittleEndian.Uint16(m.Mem[uint32(ss)*16+uint32(sp) : uint32(ss)*16+uint32(sp)+2])
			pending = &event{Entry: m.Steps, Slot: id, X: x, Y: y, AX: c.R[cpu.AX], DX: c.R[cpu.DX], DS: ds, SourcePointer: pointer, LinearSource: linear, Source: append([]byte(nil), raw[id*8:id*8+8]...), ReturnIP: ret, ReturnSS: ss, ReturnSP: sp + 2, FrameOffset: int64(len(events) * 128000)}
			before = m.Indexed()
			_, e = stream.Write(before)
			check(e)
		}
		if pending != nil && c.Seg[cpu.CS] == 0x161 {
			rowID := map[uint16]int{0x5002: 1, 0x5010: 2, 0x501e: 3, 0x502c: 4}[c.IP]
			if rowID > 0 {
				if len(pending.Rows)+1 != rowID || c.R[cpu.SI] != uint16(pending.X) || c.R[cpu.DI] != uint16(pending.Y+rowID-1) {
					log.Fatal("列完成順序或SI/DI座標不符")
				}
				frame := m.Indexed()
				pending.Rows = append(pending.Rows, row{m.Steps, c.IP, c.R[cpu.SI], c.R[cpu.DI], c.R[cpu.DX], crop(frame, pending.X, pending.Y, 4, 4)})
			}
			if c.IP == pending.ReturnIP && c.Seg[cpu.SS] == pending.ReturnSS && c.R[cpu.SP] == pending.ReturnSP {
				if len(pending.Rows) != 4 {
					log.Fatal("返回未完成四列")
				}
				after := m.Indexed()
				got := crop(after, pending.X, pending.Y, 4, 4)
				for i, b := range got {
					want := pending.Source[i/2] >> 4
					if i%2 == 1 {
						want = pending.Source[i/2] & 15
					}
					if b != want {
						pending.SourceMismatch++
					}
				}
				for i, b := range after {
					if !(pending.X <= i%320 && i%320 < pending.X+4 && pending.Y <= i/320 && i/320 < pending.Y+4) && b != before[i] {
						pending.OutsideChanged++
					}
				}
				if pending.SourceMismatch != 0 || pending.OutsideChanged != 0 {
					log.Fatalf("原版圖塊copy語意不同 %#v", pending)
				}
				pending.Return = m.Steps
				_, e = stream.Write(after)
				check(e)
				events = append(events, *pending)
				pending = nil
			}
		}
		if m.Steps == 65999999 {
			check(state.Save(*out+"-observed.state", m, d))
		}
		at, irq := m.Steps, m.IRQ1Delivered()
		check(m.Step())
		if m.IRQ1Delivered() != irq {
			keys = append(keys, at)
		}
		if d.Exited || c.Halted {
			log.Fatal("正常路線意外退出")
		}
	}
	check(stream.Flush())
	check(file.Close())
	if pending != nil || len(events) == 0 || len(keys) != 4 || !bytes.Equal(m.Indexed(), read(oldFrame)) {
		log.Fatalf("終點／事件不符 events=%d IRQ1=%d", len(events), len(keys))
	}
	observed := fp(m)
	write(*out+"-end.frame", m.Indexed())
	control, cd := newMachine()
	defer cd.Close()
	if fp(control) != initial {
		log.Fatal("控制起點不同")
	}
	for control.Steps < 66000000 {
		check(control.Step())
	}
	if fp(control) != observed || !reflect.DeepEqual(m.Snapshot(), control.Snapshot()) {
		log.Fatal("觀察干擾完整原版機器快照")
	}
	jsonWrite(*out+".json", map[string]any{"scope": "MAZE正常BBS圖塊入口／四列返回／完整返回及無觀察控制", "go_version": runtime.Version(), "inputs_sha256": inputs, "address_space": "DOS runtime CS:IP 0161:4FD6 entry, 5002/5010/501E/502C row returns; DS:AED4 pointer; original screen pixels", "ida_address_note": "IDA 0x154E6 corresponds to runtime 0161:4FD6 by exact bytes; original database has no function boundary here", "seed": fmt.Sprintf("%04X", seed), "initial": initial, "events": events, "keys_irq1": keys, "observed_end": observed, "control_end": fp(control), "complete_machine_snapshot_equal": true, "frame_stream": *out + "-frames.bin", "frame_stream_sha256": hash(read(*out + "-frames.bin")), "limits": "only normal initial left/up BBS route; no HD, source data format, full maze geometry or art completion claim"})
	fmt.Printf("MAZE: %d原版圖塊、%d列完成、4次IRQ1，正常frame與完整機器快照相同\n", len(events), len(events)*4)
}
