// 原版正常遭遇後的有界按鍵及唯讀貼圖觀察；不啟用 HD 或改遊戲資料。
package main

import (
	"bytes"
	"crypto/sha256"
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

func main() {
	out := flag.String("out", "", "全新輸出前綴")
	deltas := flag.Bool("deltas", false, "另收集同來源檔的 XOR 差分與未知來源摘要")
	key := flag.String("key", "space", "正常原版按鍵 space／enter／f1／f2／none；缺前提不視為功能已驗")
	guardAt := flag.Uint64("guard-at", 0, "僅 space 分支可在指定絕對指令數再按住 Enter，原版雙鍵正常輸入")
	flag.Parse()
	scans := map[string]uint8{"space": 0x39, "enter": 0x1C, "f1": 0x3B, "f2": 0x3C, "none": 0}
	scan, knownKey := scans[*key]
	if !knownKey {
		log.Fatal("未知原版按鍵")
	}
	if *guardAt != 0 && (*key != "space" || *guardAt <= 86500000 || *guardAt >= 112000000) {
		log.Fatal("雙鍵時點或前置按鍵不符")
	}
	if *out == "" {
		log.Fatal("必須指定 -out")
	}
	if ps, e := filepath.Glob(*out + "*"); e != nil || len(ps) != 0 {
		log.Fatal("拒絕覆寫")
	}
	check := func(e error) {
		if e != nil {
			log.Fatal(e)
		}
	}
	read := func(p string) []byte { b, e := os.ReadFile(p); check(e); return b }
	hash := func(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
	write := func(p string, b []byte) {
		f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		check(e)
		_, e = f.Write(b)
		check(e)
		check(f.Close())
	}
	const orig = "/orig/psychic-war"
	const start = "workplace/states/08-encounter.state"
	inputs := map[string]string{}
	for _, p := range []string{start, orig + "/PW.EXE", orig + "/BEAM.PBL", orig + "/FIGHT.PBL", orig + "/ENEMY00.PBL", orig + "/ALLY.PBL", "tools/hd/observe_battle_effects.go"} {
		inputs[p] = hash(read(p))
	}
	if inputs[start] != "02d5fdba9181cb7fbec6282d74efbc25396562ba1403893cdf4c579ddfe770ba" {
		log.Fatal("固定原版起點不符")
	}
	type asset struct {
		Name        string
		Image, W, H int
		Packed      []byte
	}
	var assets []asset
	for _, name := range []string{"BEAM", "FIGHT", "ENEMY00", "ALLY"} {
		b := read(orig + "/" + name + ".PBL")
		offsets, e := pbl.Offsets(b)
		check(e)
		expectedCount := 12
		if name == "ENEMY00" {
			expectedCount = 30
		}
		if name == "ALLY" {
			expectedCount = 31
		}
		if len(offsets) != expectedCount {
			log.Fatal("實際圖數不符")
		}
		for n := range offsets {
			w, h, px, e := pbl.Decode(b, n)
			check(e)
			packed := make([]byte, len(px)/2)
			for i := range packed {
				packed[i] = px[i*2]<<4 | px[i*2+1]
			}
			assets = append(assets, asset{name, n, w, h, packed})
		}
	}
	pairs := map[string][]string{}
	if *deltas {
		for i, a := range assets {
			for j, b := range assets {
				if j <= i || a.Name != b.Name || a.W != b.W || a.H != b.H {
					continue
				}
				raw := make([]byte, len(a.Packed))
				for k := range raw {
					raw[k] = a.Packed[k] ^ b.Packed[k]
				}
				key := hash(raw)
				pairs[key] = append(pairs[key], fmt.Sprintf("%s#%d^%s#%d", a.Name, a.Image, b.Name, b.Image))
			}
		}
	}
	o, e := oracle.Load(orig+"/PW.EXE", orig)
	check(e)
	defer o.Close()
	check(o.LoadStateFile(start))
	o.SetScratch("/tmp/pw-effects")
	seed := o.Word(oracle.Addr{Seg: 0x161, Off: 0x41DF})
	var events []map[string]any
	var masks []map[string]any
	var maskPending map[string]any
	var maskBefore, maskSource []byte
	var maskRect [4]int
	geometry := map[string]int{}
	unknown := map[string]int{}
	unknownMask := map[string]int{}
	var pending map[string]any
	var before, source []byte
	var rect [4]int
	var mode uint16
	// IDA sub_1530A 的唯一已核對輸出呼叫；CS:4E36 為 32 bytes 位元遮罩。
	o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x8260}, func(o *oracle.Oracle) {
		r := o.Regs()
		if r.DS != 0x161 || r.DX != 0x4E36 {
			unknownMask[fmt.Sprintf("DS:DX=%04X:%04X", r.DS, r.DX)]++
			return
		}
		if maskPending != nil || pending != nil || r.BX < 0xC000 {
			log.Fatal("遮罩巢狀或座標不符")
		}
		pos := int(r.BX) - 0xC000
		x, y := (pos%80)*4, pos/80
		if x+16 > 320 || y+16 > 200 {
			log.Fatal("遮罩超出畫面")
		}
		maskBefore = o.Indexed()
		maskSource = o.Bytes(oracle.Addr{Seg: r.DS, Off: r.DX}, 32)
		maskRect = [4]int{x, y, 16, 16}
		maskPending = map[string]any{"entry_step": o.Steps(), "entry_regs": r, "rect": maskRect, "source_sha256": hash(maskSource), "source": "PW_UNP.EXE CS:4E36 32 bytes，IDA ea 15346", "xor_color": 10}
	})
	o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x4E34}, func(o *oracle.Oracle) {
		if maskPending == nil {
			return
		}
		after := o.Indexed()
		x, y := maskRect[0], maskRect[1]
		mismatch, outside, changed := 0, 0, 0
		for i, v := range after {
			expected := maskBefore[i]
			col, row := i%320-x, i/320-y
			if col >= 0 && col < 16 && row >= 0 && row < 16 {
				if maskSource[row*2+col/8]&(0x80>>uint(col%8)) != 0 {
					expected ^= 10
				}
			} else if v != maskBefore[i] {
				outside++
			}
			if v != expected {
				mismatch++
			}
			if v != maskBefore[i] {
				changed++
			}
		}
		n := len(masks)
		if n >= 2048 {
			log.Fatal("遮罩事件數超出上限")
		}
		prefix := fmt.Sprintf("%s-mask%03d", *out, n)
		write(prefix+"-before.frame", maskBefore)
		write(prefix+"-after.frame", after)
		write(prefix+"-source.bin", maskSource)
		maskPending["return_step"] = o.Steps()
		maskPending["return_regs"] = o.Regs()
		maskPending["before_sha256"] = hash(maskBefore)
		maskPending["after_sha256"] = hash(after)
		maskPending["source_mismatch"] = mismatch
		maskPending["outside_changed_pixels"] = outside
		maskPending["changed_pixels"] = changed
		masks = append(masks, maskPending)
		maskPending = nil
	})
	o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x8705}, func(o *oracle.Oracle) {
		r := o.Regs()
		x, y, w, h := int(r.CX>>8)*4, int(r.CX&255)*4, int(r.DX>>8)*8, int(r.DX&255)*8
		key := fmt.Sprintf("%d,%d,%d,%d,AL=%02X", x, y, w, h, r.AX&255)
		geometry[key]++
		if pending != nil {
			log.Fatal("貼圖巢狀")
		}
		if w != 16 && w != 24 || h != 16 && h != 32 {
			return
		}
		a := oracle.Addr{Seg: r.DS, Off: r.BX}
		size := w * h / 2
		if a.Linear() > 0xA0000-uint32(size) {
			return
		}
		raw := o.Bytes(a, size)
		labels := []string{}
		for _, asset := range assets {
			if w == asset.W && h == asset.H && bytes.Equal(raw, asset.Packed) {
				labels = append(labels, fmt.Sprintf("%s#%d", asset.Name, asset.Image))
			}
		}
		candidatePairs := pairs[hash(raw)]
		if len(labels) == 0 && len(candidatePairs) == 0 {
			if *deltas {
				key := fmt.Sprintf("%s,source=%s", key, hash(raw))
				unknown[key]++
				p := fmt.Sprintf("%s-unknown-%s.bin", *out, hash(raw))
				if _, e := os.Lstat(p); os.IsNotExist(e) {
					write(p, raw)
				}
			}
			return
		}
		if len(events) > 2048 {
			log.Fatal("超過有界事件數")
		}
		pending = map[string]any{"entry_step": o.Steps(), "entry_regs": r, "source_images": labels, "rect": [4]int{x, y, w, h}, "source_sha256": hash(raw)}
		pending["source_pairs"] = candidatePairs
		before = o.Indexed()
		source = raw
		rect = [4]int{x, y, w, h}
		mode = r.AX & 255
	})
	o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x8751}, func(o *oracle.Oracle) {
		if pending == nil {
			return
		}
		after := o.Indexed()
		x, y, w, h := rect[0], rect[1], rect[2], rect[3]
		prev, e := pbl.Region(before, 320, 200, x, y, w, h)
		check(e)
		next, e := pbl.Region(after, 320, 200, x, y, w, h)
		check(e)
		diff, outside, mismatch := 0, 0, 0
		for i, v := range next {
			expected := source[i/2]
			if i%2 == 0 {
				expected >>= 4
			} else {
				expected &= 15
			}
			if mode == 1 {
				expected ^= prev[i]
			}
			if v != expected {
				mismatch++
			}
			if v != prev[i] {
				diff++
			}
		}
		for i, v := range after {
			if (i%320 < x || i%320 >= x+w || i/320 < y || i/320 >= y+h) && v != before[i] {
				outside++
			}
		}
		pending["return_step"] = o.Steps()
		pending["return_regs"] = o.Regs()
		pending["al"] = mode
		pending["cycles"] = o.Cycles()
		pending["before_sha256"] = hash(before)
		pending["after_sha256"] = hash(after)
		pending["changed_pixels"] = diff
		pending["outside_changed_pixels"] = outside
		pending["source_mismatch"] = mismatch
		n := len(events)
		prefix := fmt.Sprintf("%s-event%03d", *out, n)
		write(prefix+"-before.frame", before)
		write(prefix+"-after.frame", after)
		write(prefix+"-source.bin", source)
		events = append(events, pending)
		pending = nil
	})
	run := func(step uint64) { check(o.Run(step - o.Steps())) }
	run(86500000)
	if scan != 0 {
		o.KeyDown(scan)
	}
	if *guardAt != 0 {
		run(*guardAt)
		o.KeyDown(0x1C)
	}
	run(112000000)
	if pending != nil || maskPending != nil {
		log.Fatal("終點在貼圖中")
	}
	check(o.SaveStateFile(*out + "-end.state"))
	doc := map[string]any{"tool": "dosgolem f8c1a6e 原版唯讀觀察", "go_version": runtime.Version(), "inputs_sha256": inputs, "address_space": "執行期 CS:IP 0161:8705／8751，來源 DS:BX；原版 320×200 色號座標", "seed_before": fmt.Sprintf("%04X", seed), "seed_method": "執行前以固定 state 雜湊鎖定，唯讀取得種子；不改值／重擲", "input": "既有正常 08-encounter 起點；86,500,000 按下空白鍵，保持至 112,000,000；state 既有佇列不改動", "events": events, "all_geometry_counts": geometry, "unknown_source_counts": unknown, "end_steps": o.Steps(), "end_cycles": o.Cycles(), "end_ram_sha256": hash(o.Bytes(oracle.Addr{}, 1<<20)), "end_indexed_sha256": hash(o.Indexed()), "end_regs": o.Regs(), "limits": "只收集吻合 BEAM／FIGHT 或同檔 XOR 的來源；差分配對不證明方向，未匹配來源不解釋，非完整特效或 HD 驗收"}
	b, e := json.MarshalIndent(doc, "", "  ")
	doc["mask_events"] = masks
	doc["key"] = *key
	doc["key_scan"] = scan
	doc["guard_at"] = *guardAt
	doc["unknown_mask_sources"] = unknownMask
	doc["input"] = fmt.Sprintf("既有正常 08-encounter 起點、5447h；86,500,000 按下 %s（掃描碼 %02X），保持至 112,000,000；none 不加按鍵，state 既有佇列均不改動", *key, scan)
	if *guardAt != 0 {
		doc["input"] = fmt.Sprintf("既有正常 08-encounter 起點、5447h；86,500,000 按住 space，%d 再按住 Enter，保持至 112,000,000；不改 state 既有佇列或遊戲資料", *guardAt)
	}
	doc["limits"] = "一般貼圖僅辨識已解碼 BEAM／FIGHT／ENEMY00 完整來源及同檔 XOR；另收集已證實 CS:4E36 位元遮罩。XOR 配對方向及完整場景須由獨立核對器確認；不是 HD 或所有戰鬥驗收"
	b, e = json.MarshalIndent(doc, "", "  ")
	check(e)
	write(*out+".json", append(b, '\n'))
	fmt.Printf("原版有限按鍵觀察：符合來源 %d 次，幾何／模式 %d 類\n", len(events), len(geometry))
}
