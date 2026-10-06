// 研究038 §108：將實際正式圖面與原版RGB合成可檢視PNG，不修改美術素材。
package main

import (
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
)

func main() {
	const work = "/src/workplace/hd/body-runtime-v1-20261004"
	check := func(e error) {
		if e != nil {
			log.Fatal(e)
		}
	}
	read := func(p string) []byte { b, e := os.ReadFile(p); check(e); return b }
	for _, name := range []string{"pose2-return", "reload-full-next-pose2"} {
		rgb := read(filepath.Join(work, name+".rgb"))
		plane := read(filepath.Join(work, name+"-plane.rgba"))
		if len(rgb) != 320*200*3 || len(plane) != 960*600*4 {
			log.Fatal("原版RGB或正式圖面長度不符")
		}
		img := image.NewNRGBA(image.Rect(0, 0, 960, 600))
		for y := 0; y < 600; y++ {
			for x := 0; x < 960; x++ {
				pi := 4 * (y*960 + x)
				bi := 3 * ((y/3)*320 + x/3)
				a := uint32(plane[pi+3])
				for c := 0; c < 3; c++ {
					img.Pix[pi+c] = byte((uint32(plane[pi+c])*a + uint32(rgb[bi+c])*(255-a) + 127) / 255)
				}
				img.Pix[pi+3] = 255
			}
		}
		f, e := os.OpenFile(filepath.Join(work, name+"-preview.png"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		check(e)
		check(png.Encode(f, img))
		check(f.Close())
	}
}
