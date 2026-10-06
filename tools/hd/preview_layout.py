"""離線排版原型：人物照原版定位，框線在前，中文最後繪製。

入口及邊界見 docs/re/038 §31、docs/spec/024 §3.6。只在 Docker 執行。
保留所有舊圖與收據；不接入 DRAFT 的正式遊戲路徑。
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import struct
import sys
import zlib

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import pbl

ROOT = Path(__file__).resolve().parents[2]
W, H, SCALE = 960, 600, 3
GIRL = (232, 0, 88, 144)
# 原版人物前方的框架，人工核對 SCREEN 的原始色號及參照圖。
# 含黑色內部，讓控制台的開口也遮住後方人物；不是黑色透明化。
# 此輪廓屬強證據的美術分區，不是由遊戲判斷得出的物件邊界。
FRONT = ((232, 0, 88, 6), (232, 131, 14, 13),
         (246, 133, 28, 11), (274, 129, 42, 15), (316, 123, 4, 21))


def load(rel, size=None, channels=None):
    path = ROOT / rel
    w, h, ch, rows, palette = pbl.read_png(path)
    if palette is not None or (size and (w, h) != size) or (channels and ch != channels):
        raise ValueError(f"素材尺寸或像素格式不符：{rel}")
    rgba = bytearray()
    for row in rows:
        for x in range(w):
            rgba.extend(row[ch*x:ch*x+3])
            rgba.append(row[ch*x+3] if ch == 4 else 255)
    return w, h, rgba


def blend(dst, src, width=W, height=H, at=(0, 0)):
    for y in range(height):
        for x in range(width):
            s = 4*(y*width+x)
            a = src[s+3]
            if not a:
                continue
            d = 4*((y+at[1])*W+x+at[0])
            for c in range(3):
                dst[d+c] = (src[s+c]*a+dst[d+c]*(255-a)+127)//255


def png(path, rgba, width=W, height=H, rgb=False):
    ch = 3 if rgb else 4
    pixels = bytes(v for i, v in enumerate(rgba) if not rgb or i % 4 != 3)
    raw = b"".join(b"\0"+pixels[y*width*ch:(y+1)*width*ch] for y in range(height))
    def chunk(tag, data):
        return struct.pack(">I", len(data))+tag+data+struct.pack(">I", zlib.crc32(tag+data)&0xffffffff)
    path.write_bytes(b"\x89PNG\r\n\x1a\n"+chunk(b"IHDR", struct.pack(">IIBBBBB", width,height,8,2 if rgb else 6,0,0,0))
                     +chunk(b"IDAT",zlib.compress(raw,9))+chunk(b"IEND",b""))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, help="既有目錄內的新檔前綴，不得覆寫")
    parser.add_argument("--scene", default="workplace/hd/redraw/replay-encounter")
    args = parser.parse_args()
    prefix = Path(args.output).resolve()
    if not prefix.parent.is_dir() or prefix.parent.stat().st_uid != os.getuid():
        raise ValueError("輸出目錄必須已存在且由目前使用者擁有")
    suffixes = ("-background.png", "-foreground.png", "-reversed.png", "-chinese.png", "-comparison.png", ".json")
    outputs = [Path(str(prefix)+s) for s in suffixes]
    if any(p.exists() for p in outputs):
        raise ValueError("輸出已存在，請用新前綴保留既有證據")

    sources = ["tools/hd/preview_layout.py", "tools/pbl.py", "workplace/hd/bg.idx",
               "workplace/original/psychic-war/SCREEN.PBL", "workplace/original/psychic-war/MENU.PBL",
               "workplace/hd/full-near3-rgb.png", "workplace/hd/redraw/girl-raw.png",
               "workplace/hd/redraw/chrome-masked.png", "workplace/hd/redraw/dedither.png",
               "workplace/hd/redraw/frames.png", "workplace/hd/redraw/MENU-00-vec.png",
               "workplace/hd/redraw/full-hd-rgb.png"]
    sources += [args.scene+s for s in (".frame", "-original.png", "-overlay.png", "-chinese.png")]
    hashes = {p: hashlib.sha256((ROOT/p).read_bytes()).hexdigest() for p in sources}
    idx = (ROOT/"workplace/hd/bg.idx").read_bytes()
    if len(idx) != 64000:
        raise ValueError("原版背景須為 320×200 色號")
    rebuilt = bytearray(64000)
    for name, count in (("SCREEN",5),("MENU",1)):
        data = (ROOT/f"workplace/original/psychic-war/{name}.PBL").read_bytes()
        entries = list(pbl.images(data))
        if len(entries) != count:
            raise ValueError("原版 PBL 圖數不符")
        for i, off, _ in entries:
            sw,sh,px = pbl.decode(data,off)
            at = (0,i*40) if name == "SCREEN" else (160,4)
            if (sw,sh) != ((320,40) if name == "SCREEN" else (88,72)):
                raise ValueError("原版 PBL 尺寸不符")
            for y in range(sh):
                start = (y+at[1])*320+at[0]
                rebuilt[start:start+sw] = bytes(px[y*sw:(y+1)*sw])
    if rebuilt != idx:
        raise ValueError("背景不是原版 SCREEN／MENU 的原座標重建")

    _,_,original = load("workplace/hd/full-near3-rgb.png", (W,H),3)
    _,_,girl = load("workplace/hd/redraw/girl-raw.png", (264,456),4)
    # 固定原版 x=232，不使用舊版的等比縮小／靠右平移。
    girl = girl[:264*432*4]
    behind = bytearray(original)
    blend(behind,girl,264,432,(696,0))
    foreground = bytearray(W*H*4)
    _,_,chrome = load("workplace/hd/redraw/chrome-masked.png",(W,H),4)
    blend(foreground,chrome)
    # blend 只改 RGB；圖層本身的 alpha 也須保留。
    foreground[3::4] = chrome[3::4]
    for rx,ry,rw,rh in FRONT:
        for y in range(ry*SCALE,(ry+rh)*SCALE):
            for x in range(rx*SCALE,(rx+rw)*SCALE):
                j = 4*(y*W+x)
                foreground[j:j+4] = original[j:j+4]
    blend(behind,foreground)
    for rel, at, size in (("dedither.png",(0,0),(W,H)), ("frames.png",(0,0),(W,H)),
                          ("MENU-00-vec.png",(480,12),(264,216))):
        ww,hh,layer = load("workplace/hd/redraw/"+rel,size,4)
        blend(behind,layer,ww,hh,at)
    correct = behind
    reversed_order = bytearray(correct)
    blend(reversed_order,girl,264,432,(696,0))

    _,_,old = load("workplace/hd/redraw/full-hd-rgb.png",(W,H),3)
    outside = sum(correct[i:i+3] != old[i:i+3] for y in range(H) for x in range(W)
                  if not (696 <= x < 960 and y < 432) for i in [4*(y*W+x)])
    if outside:
        raise ValueError(f"人物區域之外意外改動 {outside} 個像素")
    _,_,frames = load("workplace/hd/redraw/frames.png",(W,H),4)
    frame_bad = negative = overlap = 0
    for rx,ry,rw,rh in FRONT:
        for y in range(ry*3,(ry+rh)*3):
            for x in range(rx*3,(rx+rw)*3):
                j = 4*(y*W+x)
                # 期望來自原版座標，不從成果截圖取答案。
                a = frames[j+3]
                expected = bytes((frames[j+c]*a+original[j+c]*(255-a)+127)//255 for c in range(3))
                frame_bad += correct[j:j+3] != expected
                negative += reversed_order[j:j+3] != expected
                overlap += girl[4*(y*264+x-696)+3] != 0
    if frame_bad or not negative or not overlap:
        raise ValueError("框線前置驗收或錯序反向對照失敗")

    current = (ROOT/(args.scene+".frame")).read_bytes()
    if len(current) != 64000:
        raise ValueError("場景須為 320×200 色號")
    _,_,base = load(args.scene+"-original.png",(W,H),3)
    _,_,overlay = load(args.scene+"-overlay.png",(W,H),4)
    _,_,chinese = load(args.scene+"-chinese.png",(W,H),3)
    visible = set()
    for y in range(0,200,8):
        for x in range(0,320,8):
            if all(current[yy*320+xx] == idx[yy*320+xx] for yy in range(y,y+8) for xx in range(x,x+8)):
                visible.add((x//8,y//8))
    output = bytearray(base)
    for y in range(H):
        for x in range(W):
            if (x//24,y//24) in visible:
                j = 4*(y*W+x)
                output[j:j+3] = correct[j:j+3]
    blend(output,overlay)
    dynamic_bad = text_bad = 0
    for y in range(H):
        for x in range(W):
            j = 4*(y*W+x)
            if (x//24,y//24) not in visible:
                dynamic_bad += output[j:j+3] != chinese[j:j+3]
            if overlay[j+3] == 255:
                text_bad += output[j:j+3] != overlay[j:j+3]
    if dynamic_bad or text_bad:
        raise ValueError("原版動態或中文被 HD 圖層覆蓋")

    comparison = bytearray()
    for y in range(H):
        comparison.extend(chinese[y*W*4:(y+1)*W*4])
        comparison.extend(output[y*W*4:(y+1)*W*4])
    for path, pixels, ww in zip(outputs[:5],(correct,foreground,reversed_order,output,comparison),(W,W,W,W,W*2)):
        png(path,pixels,ww,H,rgb=path != outputs[1])
    receipt = {"status":"DRAFT 離線排版原型；尚未接入正式執行期", "mask_cell":[8,8],
               "girl_at_original":[232,0], "girl_size_original":[88,144],
               "girl_transform":"原尺寸，僅裁掉 y≥144；不縮小或平移", "foreground_rects_original":FRONT,
               "foreground_evidence":"強證據：人工核對原版框架分區；不是全部美術輪廓驗收",
               "order":["原版底圖","美女圖","框架／框線／控制元件","原版動態格","中文"],
               "checks":{"pbl_background_mismatch":0,"outside_girl_changes":outside,
                         "foreground_mismatch":frame_bad,"reversed_order_mismatch":negative,
                         "girl_frame_overlap_pixels":overlap,"dynamic_mismatch":dynamic_bad,"chinese_mismatch":text_bad},
               "inputs_sha256":hashes,
               "outputs_sha256":{str(p.relative_to(ROOT)):hashlib.sha256(p.read_bytes()).hexdigest() for p in outputs[:5]},
               "limits":"下緣與頂緣框架暫沿用原版像素，五條既有 HD 漸層覆於其上；美術品質與完整手繪框架尚未驗收。單幀不證明切換、存讀檔或跨幀恢復。"}
    outputs[5].write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+"\n",encoding="utf-8")
    print(json.dumps(receipt["checks"],ensure_ascii=False))


if __name__ == "__main__":
    main()
