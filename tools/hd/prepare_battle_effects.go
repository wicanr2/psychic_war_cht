// 研究 038 §37：將生成候選轉成實際尺寸並製作對照，不改造型或裁切。
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
)

func main() {
	version := flag.String("version", "v1", "全新候選版本 v1、v2 或 v3")
	referenceVersion := flag.String("reference-version", "v1", "明示參照版本；v3 使用已驗戰鬥色盤")
	flag.Parse()
	if *version != "v1" && *version != "v2" && *version != "v3" {
		log.Fatal("未知候選版本")
	}
	check := func(e error) {
		if e != nil {
			log.Fatal(e)
		}
	}
	readImage := func(p string) image.Image {
		f, e := os.Open(p)
		check(e)
		defer f.Close()
		im, e := png.Decode(f)
		check(e)
		return im
	}
	writeImage := func(p string, im image.Image) {
		f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		check(e)
		check(png.Encode(f, im))
		check(f.Close())
	}
	hash := func(p string) string { b, e := os.ReadFile(p); check(e); return fmt.Sprintf("%x", sha256.Sum256(b)) }
	var rows []map[string]any
	b, e := os.ReadFile("workplace/hd/battle-effects-generated-" + *version + "-20261001.json")
	check(e)
	check(json.Unmarshal(b, &rows))
	canvas := image.NewNRGBA(image.Rect(0, 0, 4*192, 3*192))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.NRGBA{24, 24, 32, 255}}, image.Point{}, draw.Src)
	for n, row := range rows {
		id := row["id"].(string)
		src := row["local_path"].(string)
		im := readImage(src)
		bounds := im.Bounds()
		if hash(src) != row["sha256"] {
			log.Fatal("生成來源雜湊不同")
		}
		dst := image.NewNRGBA(image.Rect(0, 0, 48, 48))
		alphaMin, alphaMax := uint32(65535), uint32(0)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, a := im.At(x, y).RGBA()
				if a < alphaMin {
					alphaMin = a
				}
				if a > alphaMax {
					alphaMax = a
				}
			}
		}
		if alphaMin != 0 || alphaMax == 0 {
			log.Fatal("生成圖沒有有效透明通道")
		}
		// 面積濾波在預乘色值上平均，再還原非預乘 PNG，保留真實 alpha。
		for y := 0; y < 48; y++ {
			for x := 0; x < 48; x++ {
				x0, x1 := float64(x)*float64(bounds.Dx())/48, float64(x+1)*float64(bounds.Dx())/48
				y0, y1 := float64(y)*float64(bounds.Dy())/48, float64(y+1)*float64(bounds.Dy())/48
				var r, g, b, a, area float64
				for sy := int(math.Floor(y0)); sy < int(math.Ceil(y1)); sy++ {
					for sx := int(math.Floor(x0)); sx < int(math.Ceil(x1)); sx++ {
						weight := (math.Min(x1, float64(sx+1)) - math.Max(x0, float64(sx))) * (math.Min(y1, float64(sy+1)) - math.Max(y0, float64(sy)))
						rr, gg, bb, aa := im.At(bounds.Min.X+sx, bounds.Min.Y+sy).RGBA()
						r += float64(rr) * weight
						g += float64(gg) * weight
						b += float64(bb) * weight
						a += float64(aa) * weight
						area += weight
					}
				}
				if a > 0 {
					dst.SetNRGBA(x, y, color.NRGBA{uint8(math.Round(r / a * 255)), uint8(math.Round(g / a * 255)), uint8(math.Round(b / a * 255)), uint8(math.Round(a / area / 65535 * 255))})
				}
			}
		}
		out := filepath.Join("workplace/hd/art-in", "battle-effect-"+id+"-48-"+*version+"-20261001.png")
		writeImage(out, dst)
		refName := "ENEMY00-" + id
		if id == "mask" {
			refName = "MASK-4E36"
		}
		refPath := "workplace/hd/" + refName + "-reference-" + *referenceVersion + "-20261001.png"
		ref := readImage(refPath)
		for y := 0; y < 192; y++ {
			for x := 0; x < 192; x++ {
				canvas.Set(n*192+x, y, ref.At((x/12)*16, (y/12)*16))
				c := dst.NRGBAAt(x/4, y/4)
				base := canvas.NRGBAAt(n*192+x, 192+y)
				a := uint32(c.A)
				canvas.SetNRGBA(n*192+x, 192+y, color.NRGBA{uint8((uint32(c.R)*a + uint32(base.R)*(255-a) + 127) / 255), uint8((uint32(c.G)*a + uint32(base.G)*(255-a) + 127) / 255), uint8((uint32(c.B)*a + uint32(base.B)*(255-a) + 127) / 255), 255})
				if x < 48 && y < 48 {
					canvas.Set(n*192+x+72, 384+y+72, dst.At(x, y))
				}
			}
		}
		row["reference_sha256"] = hash(refPath)
		row["output_path"] = out
		row["output_sha256"] = hash(out)
		row["source_dimensions"] = []int{bounds.Dx(), bounds.Dy()}
		row["output_dimensions"] = []int{48, 48}
		row["alpha_range_16bit"] = []uint32{alphaMin, alphaMax}
		row["conversion"] = "全圖面積縮放，沒有裁切／平移；真實 alpha 保留"
		row["quality_status"] = "候選，構圖及動畫未通過，不接入 production"
	}
	writeImage("workplace/hd/battle-effects-candidates-comparison-"+*version+"-20261001.png", canvas)
	b, e = json.MarshalIndent(map[string]any{"tool_sha256": hash("tools/hd/prepare_battle_effects.go"), "rows": rows, "comparison_rows": "原版放大、48×48 候選放大、實際 48×48；列內依序 #18／#19／#20／CS:4E36", "limits": "轉檔與透明通道核對不代表美術或正常執行期驗收"}, "", "  ")
	check(e)
	f, e := os.OpenFile("workplace/hd/battle-effects-candidates-"+*version+"-20261001.json", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	check(e)
	_, e = f.Write(append(b, '\n'))
	check(e)
	check(f.Close())
	fmt.Println("四張透明候選轉成 48×48，並排對照已保存；未接入正式遊戲")
}
