// 原版選單DAT存讀檔與HD開關對照；入口研究038 §63。僅研究覆映射建置。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/wicanr2/dosgolem/internal/dos"
	"github.com/wicanr2/dosgolem/internal/machine"
	"github.com/wicanr2/dosgolem/internal/state"
	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/dosgolem/xlate"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/theme"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/translator"
)

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
func jsonWrite(p string, v any) {
	b, e := json.MarshalIndent(v, "", "  ")
	check(e)
	write(p, append(b, '\n'))
}
func scan(k string) uint8 {
	if v, ok := map[string]uint8{"up": 0x48, "down": 0x50, "right": 0x4d, "left": 0x4b, "esc": 1, "enter": 0x1c, "space": 0x39}[k]; ok {
		return v
	}
	n, e := strconv.ParseUint(k, 16, 8)
	check(e)
	return uint8(n)
}

type segment struct {
	Name, From string
	SaveAt     uint64 `json:"save_at"`
	KeyAt      uint64 `json:"key_at"`
	KeyEvery   uint64 `json:"key_every"`
	Keys       []string
}
type fingerprint struct {
	Regs                oracle.Regs
	Steps, Cycles, IRQ1 uint64
	RAM, Bus, Frame     string
}

func fp(o *oracle.Oracle) fingerprint {
	return fingerprint{o.Regs(), o.Steps(), o.Cycles(), o.IRQ1Delivered(), hash(o.Bytes(oracle.Addr{}, 0xa0000)), hash(o.Bytes(oracle.Addr{}, 1<<20)), hash(o.Indexed())}
}
func comparable(a, b fingerprint) bool { a.Bus = ""; b.Bus = ""; return reflect.DeepEqual(a, b) }

// 磁碟state未保存IRQ1Delivered；只在同一重播的三分支間核對該計數。
func savedStateEqual(a, b fingerprint) bool { a.IRQ1 = 0; b.IRQ1 = 0; return comparable(a, b) }

// DOS檔名不分大小寫；主機落地名稱依既有resolveWrite可能為小寫。
func datPath(dir, name string) string {
	rows, e := os.ReadDir(dir)
	check(e)
	found := ""
	for _, row := range rows {
		if strings.EqualFold(row.Name(), name) {
			if row.IsDir() || found != "" {
				log.Fatal("DAT檔名不唯一或是目錄")
			}
			found = filepath.Join(dir, row.Name())
		}
	}
	if found == "" {
		log.Fatal("沒有原版DAT：", dir, " ", name)
	}
	return found
}
func attachText(o *oracle.Oracle, side []byte) *translator.Translator {
	e, e1 := translator.LoadText("text")
	check(e1)
	f24, e1 := xlate.LoadFont("font/cjk24.golemfnt")
	check(e1)
	f16, e1 := xlate.LoadFont("font/cjk16.golemfnt")
	check(e1)
	t := translator.NewTranslator(e, f24, f16, 3, nil)
	t.Attach(o)
	if len(side) > 0 {
		check(t.Layer.Restore(side, t.Fonts()))
	}
	b, e1 := translator.LoadBaked("text")
	check(e1)
	t.AttachBaked(b, "/orig/psychic-war")
	return t
}
func open(s segment, scratch string) *oracle.Oracle {
	m := machine.New()
	d := dos.New(m, "/orig/psychic-war")
	d.Install()
	if len(s.Keys) > 0 {
		m.KeyEvery = s.KeyEvery
		if m.KeyEvery == 0 {
			m.KeyEvery = 500000
		} // probe未給press-every時的原版重播預設值。
		m.SetNextKey(s.KeyAt)
		for _, k := range s.Keys {
			m.QueueKey(scan(k))
		}
	}
	check(state.Load("workplace/states/"+s.From+".state", m, d))
	d.Root = "/orig/psychic-war"
	d.Scratch = scratch
	return oracle.ResearchWrapMachine(m, d)
}
func pngWrite(p string, pix []byte) {
	f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	check(e)
	check(png.Encode(f, &image.NRGBA{Pix: pix, Stride: 960 * 4, Rect: image.Rect(0, 0, 960, 600)}))
	check(f.Close())
}
func compose(o *oracle.Oracle, art, text []byte) []byte {
	w, h, rgb := o.ScreenRGB()
	if w != 320 || h != 200 {
		log.Fatal("畫面尺寸不符")
	}
	im := image.NewNRGBA(image.Rect(0, 0, 960, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 960; x++ {
			i, j := 3*((y/3)*320+x/3), 4*(y*960+x)
			copy(im.Pix[j:j+3], rgb[i:i+3])
			im.Pix[j+3] = 255
		}
	}
	for _, pix := range [][]byte{art, text} {
		draw.Draw(im, im.Bounds(), &image.NRGBA{Pix: pix, Stride: 960 * 4, Rect: im.Bounds()}, image.Point{}, draw.Over)
	}
	return im.Pix
}
func main() {
	out := flag.String("out", "", "全新輸出目錄")
	flag.Parse()
	if *out == "" {
		log.Fatal("缺輸出")
	}
	if _, e := os.Stat(*out); !os.IsNotExist(e) {
		log.Fatal("拒絕覆寫")
	}
	check(os.Mkdir(*out, 0755))
	const orig = "/orig/psychic-war"
	const themeDir = "workplace/hd/theme-over-v1-20261002"
	inputs := map[string]string{}
	for _, p := range []string{"tools/hd/verify_dat_runtime.go", "workplace/hd/normal-chain-oracle-bridge-v1-20261001.go", "replay/title-to-first-save.json", orig + "/PW.EXE", "go.mod", "go.sum", "worktrees/dosgolem/go.mod", themeDir + "/manifest.json"} {
		inputs[p] = hash(read(p))
	}
	if inputs[orig+"/PW.EXE"] != "88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49" {
		log.Fatal("EXE來源不符")
	}
	for _, pattern := range []string{"text/*.json", "font/*.golemfnt", themeDir + "/*.png", "apps/psychicwar/theme/*.go", "apps/psychicwar/translator/*.go", "worktrees/dosgolem/oracle/*.go", "worktrees/dosgolem/xlate/*.go", "worktrees/dosgolem/internal/**/*.go", orig + "/*.PBL", orig + "/*.BIN"} {
		paths, e := filepath.Glob(pattern)
		check(e)
		for _, p := range paths {
			inputs[p] = hash(read(p))
		}
	}
	var plan struct{ Segments []segment }
	check(json.Unmarshal(read("replay/title-to-first-save.json"), &plan))
	byName := map[string]segment{}
	for _, s := range plan.Segments {
		byName[s.Name] = s
	}
	fixed := map[string]string{
		"05-select":     "10fc7ec0faf9e5b67b187fac4a9261c90f757c8f9f0aa84f2ddaa57a47d3c9ac",
		"09-battle-won": "17e2c2c94e06414eb17bfa28743bd509604f3f6d81765f3b8c19b8c89406a428",
		"10-saved":      "abdbb98f66010d5680d3f3bfecead2c28ffd4f7f6fd625fc3b3103a62bcbd056",
		"11-loaded":     "350cdb574b8558899261647b7cfa09b85aba7605081302ddb88a01aab93111fd",
		"16-sivad":      "ae4a30d999553d223a6376deb533ee98cd1aaf043cbfb782f0b3eb9cac01df1e",
		"17-saved2":     "3c2ae7470600c61217060dc76843aed7c4097ac6e4d80a778cf3282b8afbc7e4",
		"18-loaded2":    "ec8efa5920dd122d8718bf3f6fefbc61991dc2b50efbeed45739c1032ab36f15",
	}
	for n, want := range fixed {
		p := "workplace/states/" + n + ".state"
		got := hash(read(p))
		if got != want {
			log.Fatal("固定state來源不符：", n)
		}
		inputs[p] = got
	}
	for _, n := range []string{"TEST.DAT", "TEST2.DAT"} {
		if _, e := os.Stat(filepath.Join(orig, n)); !os.IsNotExist(e) {
			log.Fatal("原版資料夾已有測試DAT，拒絕隱藏fallback")
		}
	}
	for _, n := range []string{"test.dat", "test2.dat"} {
		p := "workplace/states/scratch/" + n
		inputs[p] = hash(read(p))
	}
	// SELECT既有字面由04-cleared正常空白鍵重畫，沒有人工合成譯文。
	startPath := "workplace/states/04-cleared.state"
	inputs[startPath] = hash(read(startPath))
	o := open(byName["05-select"], "/tmp/pw-hd-dat-select")
	tr := attachText(o, nil)
	for o.Steps() < byName["05-select"].SaveAt {
		n := byName["05-select"].SaveAt - o.Steps()
		if n > 100000 {
			n = 100000
		}
		check(o.Run(n))
		tr.Frame(o)
	}
	b, e := oracle.Load(orig+"/PW.EXE", orig)
	check(e)
	check(b.LoadStateFile("workplace/states/05-select.state"))
	if !savedStateEqual(fp(o), fp(b)) {
		log.Fatalf("正常SELECT中文重播與原版不符：actual=%+v expected=%+v", fp(o), fp(b))
	}
	b.Close()
	selectLayer, e := tr.Layer.Snapshot()
	check(e)
	write(filepath.Join(*out, "05-select.layer.json"), selectLayer)
	o.Close()
	layers := map[string][]byte{"05-select": selectLayer}
	for n, p := range map[string]string{"09-battle-won": "workplace/hd/normal-chain-translation-v1-20261001-09-battle-won.layer.json", "16-sivad": "workplace/hd/sivad-translation-v1-20261002-16-sivad.layer.json"} {
		layers[n] = read(p)
		inputs[p] = hash(layers[n])
	}
	model := map[string][]fingerprint{}
	modelDAT := map[string][]byte{}
	modelText := map[string][]string{}
	results := []map[string]any{}
	for _, mode := range []string{"original", "disabled", "enabled"} {
		scratch := filepath.Join(*out, mode)
		check(os.Mkdir(scratch, 0755))
		for _, name := range []string{"10-saved", "11-loaded", "17-saved2", "18-loaded2"} {
			s := byName[name]
			o := open(s, scratch)
			seed := o.Word(oracle.Addr{Seg: 0x161, Off: 0x41df})
			start := fp(o)
			var hd *theme.Theme
			var tr *translator.Translator
			if mode != "original" {
				tr = attachText(o, layers[s.From])
				hd, _, e = theme.LoadTheme(themeDir, orig, "theme", 3)
				check(e)
				if hd == nil {
					log.Fatal("主題未載入")
				}
				hd.Enabled = mode == "enabled"
				check(hd.Attach(o))
				hd.Frame(o)
				tr.Frame(o)
			}
			marks := []uint64{}
			for i := range s.Keys {
				at := s.KeyAt + uint64(i)*s.KeyEvery + 700000
				if at < s.SaveAt {
					marks = append(marks, at)
				}
			}
			marks = append(marks, s.SaveAt)
			rows := []fingerprint{}
			textHashes := []string{}
			sampleRows := []map[string]any{}
			visible := 0
			for i, at := range marks {
				for o.Steps() < at {
					n := at - o.Steps()
					if n > 100000 {
						n = 100000
					}
					check(o.Run(n))
					if hd != nil {
						hd.Frame(o)
						tr.Frame(o)
					}
				}
				f := fp(o)
				rows = append(rows, f)
				if mode != "original" {
					if !comparable(f, model[name][i]) {
						log.Fatalf("正常原版與HD終點／取樣不同：%s %s %d", mode, name, i)
					}
				}
				art, text := make([]byte, 960*600*4), make([]byte, 960*600*4)
				if hd != nil {
					hd.Frame(o)
					tr.Frame(o)
					hd.Draw(art, 3)
					tr.Layer.Draw(text, 3, tr.MissingGlyph)
					if mode == "enabled" {
						for j := 3; j < len(art); j += 4 {
							if art[j] > 0 {
								visible++
								break
							}
						}
					}
					textHashes = append(textHashes, hash(text))
					if mode == "enabled" && hash(text) != modelText[name][i] {
						log.Fatal("HD改變中文字面")
					}
				}
				p := filepath.Join(*out, fmt.Sprintf("%s-%s-%02d", mode, name, i))
				write(p+".frame", o.Indexed())
				if i == len(marks)-1 {
					pngWrite(p+".png", compose(o, art, text))
					if tr != nil {
						side, e := tr.Layer.Snapshot()
						check(e)
						write(p+".xlate.json", side)
					}
					check(o.SaveStateFile(p + ".state"))
				}
				sampleRows = append(sampleRows, map[string]any{"step": at, "fingerprint": f, "frame": p + ".frame", "hd_plane_sha256": hash(art), "text_plane_sha256": hash(text)})
			}
			baseline, e := oracle.Load(orig+"/PW.EXE", orig)
			check(e)
			check(baseline.LoadStateFile("workplace/states/" + name + ".state"))
			if !savedStateEqual(fp(o), fp(baseline)) {
				write(filepath.Join(*out, mode+"-"+name+"-actual-ram.bin"), o.Bytes(oracle.Addr{}, 0xa0000))
				write(filepath.Join(*out, mode+"-"+name+"-expected-ram.bin"), baseline.Bytes(oracle.Addr{}, 0xa0000))
				log.Fatalf("正常DAT重播與既有原版state不同：%s %s actual=%+v expected=%+v", mode, name, fp(o), fp(baseline))
			}
			baseline.Close()
			// DAT的載回以既有獨立原版state作期望；52 bytes玩家數值範圍也另存。
			player := o.Bytes(oracle.Addr{Seg: 0x1696, Off: 6}, 52)
			write(filepath.Join(*out, mode+"-"+name+"-player.bin"), player)
			// VGA匯流排影子只在共同畫面讀取後的終點比較；中途只比較A0000h以下RAM。
			o.ScreenRGB()
			end := fp(o)
			if mode != "original" && !reflect.DeepEqual(end, model[name][len(marks)-1]) {
				log.Fatal("HD改變完整終止匯流排：", mode, " ", name)
			}
			rows[len(rows)-1] = end
			o.Close()
			datName := "TEST.DAT"
			if name == "17-saved2" || name == "18-loaded2" {
				datName = "TEST2.DAT"
			}
			actualDAT := datPath(scratch, datName)
			datName = filepath.Base(actualDAT)
			dat := read(actualDAT)
			if len(dat) != 512 {
				log.Fatal("DAT不是512 bytes")
			}
			// 檔案修改時間也是FindFirst的原版輸入。只固定新測試檔的主機metadata，原版RAM不改。
			refPath := "workplace/states/scratch/" + datName
			if !bytes.Equal(dat, read(refPath)) {
				log.Fatal("新DAT與既有獨立原版DAT不同")
			}
			refInfo, e := os.Stat(refPath)
			check(e)
			check(os.Chtimes(actualDAT, refInfo.ModTime(), refInfo.ModTime()))
			if mode == "original" {
				model[name] = rows
				modelDAT[name] = dat
			} else {
				if !bytes.Equal(dat, modelDAT[name]) {
					log.Fatal("HD改變DAT bytes")
				}
				if mode == "disabled" {
					modelText[name] = textHashes
				}
			}
			if mode == "enabled" && visible == 0 {
				log.Fatal("HD分支沒有任何圖面，拒絕空測")
			}
			results = append(results, map[string]any{"mode": mode, "name": name, "replay_segment": s, "seed_before": fmt.Sprintf("%04X", seed), "start": start, "end": end, "samples": sampleRows, "dat": filepath.Join(scratch, datName), "dat_size": len(dat), "dat_sha256": hash(dat), "dat_matches_original": true, "existing_state_equal": true, "player_bytes_sha256": hash(player), "hd_visible_samples": visible})
			results[len(results)-1]["reference_dat"] = refPath
			results[len(results)-1]["reference_dat_mtime_utc"] = refInfo.ModTime().UTC().Format("2006-01-02T15:04:05.999999999Z")
			jsonWrite(filepath.Join(*out, mode+"-"+name+".json"), results[len(results)-1])
			fmt.Println(mode, name, "正常DAT重播、原版state與512 bytes通過")
		}
	}
	bad := append([]byte(nil), modelDAT["10-saved"]...)
	bad[0] ^= 1
	if bytes.Equal(bad, modelDAT["10-saved"]) {
		log.Fatal("DAT負對照無效")
	}
	jsonWrite(filepath.Join(*out, "verification.json"), map[string]any{"tool": "dosgolem f8c1a6e，正常FIFO研究包裝；沒有修改正式API／RAM／seed", "inputs_sha256": inputs, "results": results, "negative_control_dat_byte_change_rejected": true, "limits": "兩段存檔與兩段新執行器LOAD GAME讀DAT；由既有正常state接續，不是從開機GUI、正式封包、完整HD或全文驗收。HD角色與其他sprite仍有未完成項。"})
}
