// pwstep 是《銀河超能力戰記》的逐步操作（docs/spec/009 §5）：dosgolem 規格 201 的一步，加上中文轉譯層。
//
// 每一步是一次獨立的指令：載入狀態 → 依動作腳本推進機器時間（每 1/60 秒做一次疊字定色，與前端相同）→
// 存狀態、疊字層（<狀態>.xlate.json）與截圖（放大後的原版畫面加疊字層）。代理看截圖決定下一步。
//
//	pwstep -orig workplace/original/psychic-war -new -do "wait:12000" -save-state s1.state -shot s1.png
//	pwstep -orig workplace/original/psychic-war -load-state s1.state -do "tap:Space,wait:3000" -save-state s2.state -shot s2.png
package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/dosgolem/xlate"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/translator"
)

func main() {
	orig := flag.String("orig", "", "含 PW.EXE 的原版目錄（玩家自備）")
	newGame := flag.Bool("new", false, "從程式進入點開始（不給 -load-state 時必須給）")
	loadState := flag.String("load-state", "", "從這個狀態檔接著跑；同名 .xlate.json 存在就一起還原疊字層")
	do := flag.String("do", "", "動作腳本（dosgolem 規格 201 §2.1）")
	cycles := flag.Uint64("cycles", 750, "每毫秒 cycles（docs/spec/004）")
	saveState := flag.String("save-state", "", "存狀態檔（另存 <檔名>.xlate.json）")
	shot := flag.String("shot", "", "存截圖 PNG（放大後的原版畫面加疊字層）")
	scale := flag.Int("scale", 3, "放大倍率（3 的倍數）")
	textDir := flag.String("text", "text", "文本檔目錄；空字串停用轉譯層")
	fontDir := flag.String("font", "font", "字型子集目錄")
	textLog := flag.String("text-log", "", "這一步的轉譯紀錄（JSON Lines）；不給就印到標準輸出")
	scratch := flag.String("scratch", "workplace/pwstep-saves", "遊戲存檔寫到這裡")
	flag.Parse()
	if *orig == "" || *do == "" || (*loadState == "") == !*newGame {
		flag.Usage()
		os.Exit(2)
	}
	if *scale%3 != 0 {
		log.Fatalf("-scale 要是 3 的倍數：%d", *scale)
	}
	acts, err := oracle.ParseActions(*do)
	if err != nil {
		log.Fatal(err)
	}
	o, err := oracle.Load(filepath.Join(*orig, "PW.EXE"), *orig)
	if err != nil {
		log.Fatal(err)
	}
	defer o.Close()
	if err := os.MkdirAll(*scratch, 0o755); err != nil {
		log.Fatal(err)
	}
	o.SetScratch(*scratch)
	if *loadState != "" {
		if err := o.LoadStateFile(*loadState); err != nil {
			log.Fatal(err)
		}
	}
	o.SetDOSBoxCycles(*cycles)

	var tr *translator.Translator
	if *textDir != "" {
		entries, err := translator.LoadText(*textDir)
		if err != nil {
			log.Fatal(err)
		}
		f24, err := xlate.LoadFont(filepath.Join(*fontDir, "cjk24.golemfnt"))
		if err != nil {
			log.Fatal(err)
		}
		f16, err := xlate.LoadFont(filepath.Join(*fontDir, "cjk16.golemfnt"))
		if err != nil {
			log.Fatal(err)
		}
		w := os.Stdout
		if *textLog != "" {
			f, err := os.Create(*textLog)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			w = f
		}
		tr = translator.NewTranslator(entries, f24, f16, *scale, w)
		if *loadState != "" {
			if b, err := os.ReadFile(*loadState + ".xlate.json"); err == nil {
				if err := tr.Layer.Restore(b, tr.Fonts()); err != nil {
					log.Fatal(err)
				}
			}
		}
		tr.Attach(o)
		if baked, err := translator.LoadBaked(*textDir); err != nil {
			log.Fatal(err)
		} else {
			tr.AttachBaked(baked, *orig)
		}
	}

	startMs := float64(o.Cycles()) / float64(*cycles)
	runErr := o.RunActions(acts, 0, func() {
		if tr != nil {
			tr.Frame(o)
		}
	})
	var exit *oracle.ExitError
	if runErr != nil && !errors.As(runErr, &exit) {
		log.Fatal(runErr)
	}

	if *saveState != "" {
		if err := o.SaveStateFile(*saveState); err != nil {
			log.Fatal(err)
		}
		if tr != nil {
			b, err := tr.Layer.Snapshot()
			if err != nil {
				log.Fatal(err)
			}
			if err := os.WriteFile(*saveState+".xlate.json", b, 0o644); err != nil {
				log.Fatal(err)
			}
		}
	}
	if *shot != "" {
		if err := writeShot(o, tr, *scale, *shot); err != nil {
			log.Fatal(err)
		}
	}
	stamps := 0
	if tr != nil {
		stamps = len(tr.Layer.Stamps)
	}
	fmt.Fprintf(os.Stderr, "步數 %d，機器時間 %.0f → %.0f ms，疊字 %d 筆%s\n", o.Steps(), startMs,
		float64(o.Cycles())/float64(*cycles), stamps, map[bool]string{true: "，程式已結束", false: ""}[exit != nil])
}

// writeShot 存放大後的原版畫面，疊字層不透明的像素蓋上去（與前端 Draw 的合成相同）。
func writeShot(o *oracle.Oracle, tr *translator.Translator, scale int, path string) error {
	w, h, rgb := o.ScreenRGB()
	W, H := w*scale, h*scale
	img := image.NewNRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			i, j := 3*((y/scale)*w+x/scale), 4*(y*W+x)
			img.Pix[j], img.Pix[j+1], img.Pix[j+2], img.Pix[j+3] = rgb[i], rgb[i+1], rgb[i+2], 0xFF
		}
	}
	if tr != nil {
		over := make([]uint8, 4*W*H)
		tr.Layer.W, tr.Layer.H = w, h
		tr.Layer.Draw(over, scale, tr.MissingGlyph)
		for j := 0; j < len(over); j += 4 {
			if over[j+3] != 0 {
				copy(img.Pix[j:j+4], over[j:j+4])
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
