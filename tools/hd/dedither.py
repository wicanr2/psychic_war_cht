"""把抖色棋盤格換成真正的中間色（docs/spec/024 §3.1）。

    tools/py.sh tools/hd/dedither.py <bg.idx> <寬> <高> <輸出.png> [倍率]

原版用兩色交錯表現中間色調。整數倍放大會把棋盤格一起放大，格子變大反而更明顯，
所以重繪要把它解成單一顏色。

判準是**四鄰中有 need 個同色、而且自己跟四鄰都不同色**（預設 need ＝ 3）。
棋盤格的兩個相位都符合；need ＝ 4 會在抖色區的邊緣留下一圈沒解到的格子，
所以預設放寬到 3。自己跟任一鄰居同色就跳過——那是線或塊的一部分，不是抖色。輸出是 RGBA 疊層，只有判定為抖色的像素不透明，其餘透明
讓原版透出來——所以這支工具不會動到非抖色的部分。
"""
import pathlib
import struct
import sys
import zlib

EGA = [
    (0x00, 0x00, 0x00), (0x00, 0x00, 0xAA), (0x00, 0xAA, 0x00), (0x00, 0xAA, 0xAA),
    (0xAA, 0x00, 0x00), (0xAA, 0x00, 0xAA), (0xAA, 0x55, 0x00), (0xAA, 0xAA, 0xAA),
    (0x55, 0x55, 0x55), (0x55, 0x55, 0xFF), (0x55, 0xFF, 0x55), (0x55, 0xFF, 0xFF),
    (0xFF, 0x55, 0x55), (0xFF, 0x55, 0xFF), (0xFF, 0xFF, 0x55), (0xFF, 0xFF, 0xFF),
]


def write_png(path, w, h, rgba):
    raw = b"".join(b"\0" + bytes(rgba[y * w * 4:(y + 1) * w * 4]) for y in range(h))

    def chunk(tag, body):
        c = tag + body
        return struct.pack(">I", len(body)) + c + struct.pack(">I", zlib.crc32(c) & 0xFFFFFFFF)

    pathlib.Path(path).write_bytes(
        b"\x89PNG\r\n\x1a\n"
        + chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 6, 0, 0, 0))
        + chunk(b"IDAT", zlib.compress(raw, 9))
        + chunk(b"IEND", b""))


def main(argv):
    if len(argv) < 5:
        raise SystemExit(__doc__)
    idx = pathlib.Path(argv[1]).read_bytes()
    w, h, out = int(argv[2]), int(argv[3]), argv[4]
    scale = int(argv[5]) if len(argv) > 5 else 3
    need = int(argv[6]) if len(argv) > 6 else 3  # 四鄰中要有幾個同色

    blend = {}
    n = 0
    for y in range(1, h - 1):
        for x in range(1, w - 1):
            p = idx[y * w + x]
            nb = [idx[y * w + x - 1], idx[y * w + x + 1],
                  idx[(y - 1) * w + x], idx[(y + 1) * w + x]]
            if p in nb:
                continue  # 同色的鄰居 ＝ 這是線或塊的一部分，不是抖色
            c = max(set(nb), key=nb.count)
            if nb.count(c) < need:
                continue
            a, b = EGA[p], EGA[c]
            blend[(x, y)] = tuple((a[i] + b[i]) // 2 for i in range(3))
            n += 1

    W, H = w * scale, h * scale
    rgba = bytearray(W * H * 4)
    for (x, y), (r, g, b) in blend.items():
        for dy in range(scale):
            row = (y * scale + dy) * W
            for dx in range(scale):
                i = 4 * (row + x * scale + dx)
                rgba[i:i+4] = bytes((r, g, b, 0xFF))
    write_png(out, W, H, rgba)
    print("%s %d×%d：判定為抖色的像素 %d（原圖 %d px 的 %.1f%%，四鄰門檻 %d）"
          % (out, W, H, n, w * h, 100 * n / (w * h), need))


if __name__ == "__main__":
    main(sys.argv)
