"""把重繪的整屏外框壓回功能遮罩（docs/spec/024 §3.3）。

    tools/py.sh tools/hd/mask.py <bg.idx> <寬> <高> <重繪.png> <輸出.png> [倍率]

原版背景是黑色的地方，遊戲會在上面畫迷宮視野、訊息框的字、狀態數值與角色。
重繪的外框**不能佔住那些像素**，否則訊息還沒出現時會看到一塊裝飾，
訊息出現時又整塊消失（逐格失效會處理，但畫面會閃）。

所以流程是：畫圖代理畫一張不透明的整屏 → 這支工具把「原版是黑的」那些像素
改成全透明 → 疊字層只畫剩下的部分。**正確性由這支工具保證，不靠代理自律。**
"""
import pathlib
import struct
import sys
import zlib


def read_png(path):
    b = pathlib.Path(path).read_bytes()
    assert b[:8] == b"\x89PNG\r\n\x1a\n", "不是 PNG"
    i, w, h, ch, idat = 8, 0, 0, 0, b""
    while i < len(b):
        n = struct.unpack_from(">I", b, i)[0]
        tag = b[i+4:i+8]
        body = b[i+8:i+8+n]
        if tag == b"IHDR":
            w, h, depth, ctype = struct.unpack_from(">IIBB", body, 0)
            assert depth == 8 and ctype in (2, 6), "只支援 8 位元 RGB／RGBA（ctype=%d）" % ctype
            ch = 3 if ctype == 2 else 4
        elif tag == b"IDAT":
            idat += body
        i += 12 + n
    raw = zlib.decompress(idat)
    stride = w * ch
    rows, prev = [], bytearray(stride)
    p = 0
    for _ in range(h):
        f = raw[p]; line = bytearray(raw[p+1:p+1+stride]); p += 1 + stride
        for x in range(stride):
            a = line[x-ch] if x >= ch else 0
            c = prev[x]
            d = prev[x-ch] if x >= ch else 0
            if f == 1: line[x] = (line[x] + a) & 255
            elif f == 2: line[x] = (line[x] + c) & 255
            elif f == 3: line[x] = (line[x] + (a + c) // 2) & 255
            elif f == 4:
                pp = a + c - d
                pa, pb, pc = abs(pp-a), abs(pp-c), abs(pp-d)
                line[x] = (line[x] + (a if pa <= pb and pa <= pc else c if pb <= pc else d)) & 255
        rows.append(line); prev = line
    return w, h, ch, rows


def write_png(path, w, h, rgba):
    raw = b"".join(b"\0" + bytes(rgba[y*w*4:(y+1)*w*4]) for y in range(h))

    def chunk(tag, body):
        c = tag + body
        return struct.pack(">I", len(body)) + c + struct.pack(">I", zlib.crc32(c) & 0xFFFFFFFF)

    pathlib.Path(path).write_bytes(
        b"\x89PNG\r\n\x1a\n"
        + chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 6, 0, 0, 0))
        + chunk(b"IDAT", zlib.compress(raw, 9))
        + chunk(b"IEND", b""))


def main(argv):
    if len(argv) < 6:
        raise SystemExit(__doc__)
    idx = pathlib.Path(argv[1]).read_bytes()
    w, h = int(argv[2]), int(argv[3])
    src, out = argv[4], argv[5]
    scale = int(argv[6]) if len(argv) > 6 else 3

    W, H, ch, rows = read_png(src)
    if (W, H) != (w * scale, h * scale):
        raise SystemExit("重繪圖是 %d×%d，應該是 %d×%d" % (W, H, w * scale, h * scale))

    rgba = bytearray(W * H * 4)
    kept = 0
    for y in range(H):
        r = rows[y]
        sy = y // scale
        for x in range(W):
            if idx[sy * w + x // scale] == 0:
                continue  # 原版是黑的：留給遊戲，全透明
            i = 4 * (y * W + x)
            rgba[i:i+3] = r[ch*x:ch*x+3]
            rgba[i+3] = 0xFF
            kept += 1
    write_png(out, W, H, rgba)
    print("%s %d×%d：保留 %d px（%.1f%%），其餘遮成透明留給遊戲"
          % (out, W, H, kept, 100 * kept / (W * H)))


if __name__ == "__main__":
    main(sys.argv)
