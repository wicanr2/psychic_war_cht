// 研究038 §107：正常完整第0張起點，捕捉全部相交貼圖；不修改原版資料。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
func read(path string) []byte { b, err := os.ReadFile(path); check(err); return b }
func hash(b []byte) string    { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func write(path string, b []byte) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(err)
	_, err = f.Write(b)
	check(err)
	check(f.Close())
}
func crop(frame []byte) []byte {
	b, err := pbl.Region(frame, 320, 200, 32, 152, 24, 32)
	check(err)
	return b
}

type event struct {
	Kind   string      `json:"kind"`
	Entry  uint64      `json:"entry"`
	Return uint64      `json:"return"`
	Regs   oracle.Regs `json:"regs"`
	Rect   [4]int      `json:"rect"`
	Packed string      `json:"packed_hex"`
	Before string      `json:"before_hex"`
	After  string      `json:"after_hex"`
}

func main() {
	out := flag.String("out", "", "全新工作區")
	flag.Parse()
	if *out == "" {
		log.Fatal("缺輸出")
	}
	info, err := os.Stat(*out)
	check(err)
	if !info.IsDir() {
		log.Fatal("輸出不是目錄")
	}
	const orig = "/orig/psychic-war"
	const start = "/src/workplace/hd/next-body-normal-kasuruji-v3-20261001-event00.state"
	inputs := map[string]string{}
	for _, path := range []string{start, orig + "/PW.EXE", orig + "/ENEMY00.PBL", "/src/tools/hd/observe_body_context.go"} {
		inputs[path] = hash(read(path))
	}
	archive := read(orig + "/ENEMY00.PBL")
	if hash(archive) != "8241b0ea73b1e7402b13adc434f10a5b1288f88f01fcfc3ba5710e4923fe6067" {
		log.Fatal("原版PBL不符")
	}
	w, h, pose0, err := pbl.Decode(archive, 0)
	check(err)
	if w != 24 || h != 32 {
		log.Fatal("來源尺寸不符")
	}
	load := func(scratch string) *oracle.Oracle {
		o, err := oracle.Load(orig+"/PW.EXE", orig)
		check(err)
		check(o.LoadStateFile(start))
		o.SetScratch(filepath.Join(*out, scratch))
		return o
	}
	o := load("observed-scratch")
	defer o.Close()
	initial := o.Indexed()
	if !bytes.Equal(crop(initial), pose0) {
		log.Fatal("起點不是完整第0張")
	}
	initialStep := o.Steps()
	seed := o.Word(oracle.Addr{Seg: 0x161, Off: 0x41df})
	write(filepath.Join(*out, "initial.frame"), initial)
	var pending *event
	events := []event{}
	o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x8705}, func(o *oracle.Oracle) {
		r := o.Regs()
		x, y := int(r.CX>>8)*4, int(r.CX&255)*4
		w, h := int(r.DX>>8)*8, int(r.DX&255)*8
		if x >= 56 || y >= 184 || x+w <= 32 || y+h <= 152 {
			return
		}
		if pending != nil || len(events) >= 2000 {
			log.Fatal("巢狀或事件超出上限")
		}
		if x < 0 || y < 0 || x+w > 320 || y+h > 200 || w <= 0 || h <= 0 {
			log.Fatal("原版矩形越界")
		}
		a := oracle.Addr{Seg: r.DS, Off: r.BX}
		n := w * h / 2
		if a.Linear() > 0xa0000-uint32(n) {
			log.Fatal("原版來源越界")
		}
		pending = &event{Kind: "packed", Entry: o.Steps(), Regs: r, Rect: [4]int{x, y, w, h}, Packed: hex.EncodeToString(o.Bytes(a, n)), Before: hex.EncodeToString(crop(o.Indexed()))}
	})
	o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x8751}, func(o *oracle.Oracle) {
		if pending == nil || pending.Kind != "packed" {
			return
		}
		pending.Return = o.Steps()
		pending.After = hex.EncodeToString(crop(o.Indexed()))
		events = append(events, *pending)
		pending = nil
	})
	// 研究038 §37的已證實16×16位元遮罩，原版仍自行執行。
	o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x8260}, func(o *oracle.Oracle) {
		r := o.Regs()
		if r.DS != 0x161 || r.DX != 0x4e36 || r.BX < 0xc000 {
			log.Fatal("未支援遮罩來源或座標")
		}
		pos := int(r.BX) - 0xc000
		x, y := (pos%80)*4, pos/80
		if x >= 56 || y >= 184 || x+16 <= 32 || y+16 <= 152 {
			return
		}
		if pending != nil || len(events) >= 2000 || x+16 > 320 || y+16 > 200 {
			log.Fatal("遮罩巢狀、數量或矩形越界")
		}
		pending = &event{Kind: "mask-xor10", Entry: o.Steps(), Regs: r, Rect: [4]int{x, y, 16, 16}, Packed: hex.EncodeToString(o.Bytes(oracle.Addr{Seg: r.DS, Off: r.DX}, 32)), Before: hex.EncodeToString(crop(o.Indexed()))}
	})
	o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x4e34}, func(o *oracle.Oracle) {
		if pending == nil || pending.Kind != "mask-xor10" {
			return
		}
		pending.Return = o.Steps()
		pending.After = hex.EncodeToString(crop(o.Indexed()))
		events = append(events, *pending)
		pending = nil
	})
	check(o.Run(15000000))
	for i := 0; pending != nil && i < 100000; i++ {
		check(o.Run(1))
	}
	if pending != nil || len(events) == 0 {
		log.Fatal("仍在貼圖或沒有相交事件")
	}
	endStep := o.Steps()
	check(o.SaveStateFile(filepath.Join(*out, "observed.state")))
	write(filepath.Join(*out, "observed.frame"), o.Indexed())
	control := load("control-scratch")
	defer control.Close()
	if control.Steps() != initialStep || control.Word(oracle.Addr{Seg: 0x161, Off: 0x41df}) != seed {
		log.Fatal("控制起點不同")
	}
	check(control.Run(endStep - initialStep))
	check(control.SaveStateFile(filepath.Join(*out, "control.state")))
	write(filepath.Join(*out, "control.frame"), control.Indexed())
	if !bytes.Equal(o.Indexed(), control.Indexed()) || o.Regs() != control.Regs() || o.Cycles() != control.Cycles() || !bytes.Equal(o.Bytes(oracle.Addr{}, 0xa0000), control.Bytes(oracle.Addr{}, 0xa0000)) {
		log.Fatal("觀察干擾原版")
	}
	self, err := os.Executable()
	check(err)
	inputs[self] = hash(read(self))
	doc := map[string]any{"scope": "saved normal complete-pose0 checkpoint, no-input wait, intersecting 8705/8751 packed blits and 8260/4E34 masks; original-only observer",
		"address_space": "runtime CS:IP, DS:BX, 320x200 indexed coordinates", "initial_step": initialStep, "end_step": endStep, "seed_before": fmt.Sprintf("%04X", seed),
		"seed_method": "same immutable saved state, read before execution; no value writes or rerolls", "go_version": runtime.Version(), "events": events, "inputs_sha256": inputs,
		"observed_control_frame_registers_cycles_RAM_equal": true, "limits": "full 44-field/DOS comparison and independent pixel/source model are separate; not production HD or fromboot/player GUI evidence"}
	b, err := json.MarshalIndent(doc, "", "  ")
	check(err)
	write(filepath.Join(*out, "trace.json"), append(b, '\n'))
	fmt.Println("原版相交貼圖", len(events), "兩側終點frame/CPU/RAM相同，step", endStep)
}
