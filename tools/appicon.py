"""產生發行包的圖示（docs/spec/021 §4）。

    tools/py.sh tools/appicon.py <輸出.png> [邊長]

圖示是自製的：深色底、青色外框，中間用專案自己的 `font/cjk24.golemfnt` 畫「銀河」兩字。
**不用原版的 Logo**——那是原版素材，發行包不得含（`CLAUDE.md` [HARD]）。
"""
import pathlib
import struct
import sys
import zlib

BG = (0, 0, 42)
FRAME = (0, 170, 170)
INK = (255, 255, 255)


def load_font(path):
    b = pathlib.Path(path).read_bytes()
    if b[:8] != b"GOLEMFNT":
        raise SystemExit("%s 不是 GOLEMFNT 字型" % path)
    w, h = struct.unpack_from("<HH", b, 8)
    n = struct.unpack_from("<I", b, 12)[0]
    stride = (w + 7) // 8
    out, off = {}, 16
    for _ in range(n):
        cp = struct.unpack_from("<I", b, off)[0]
        out[chr(cp)] = b[off + 5:off + 5 + h * stride]
        off += 5 + h * stride
    return w, h, stride, out


def write_png(path, size, px):
    raw = b"".join(b"\0" + bytes(c for x in range(size) for c in px[y * size + x]) for y in range(size))

    def chunk(tag, body):
        c = tag + body
        return struct.pack(">I", len(body)) + c + struct.pack(">I", zlib.crc32(c) & 0xFFFFFFFF)

    pathlib.Path(path).write_bytes(
        b"\x89PNG\r\n\x1a\n"
        + chunk(b"IHDR", struct.pack(">IIBBBBB", size, size, 8, 2, 0, 0, 0))
        + chunk(b"IDAT", zlib.compress(raw, 9))
        + chunk(b"IEND", b""))


def main(argv):
    out = argv[1] if len(argv) > 1 else "icon.png"
    size = int(argv[2]) if len(argv) > 2 else 256
    root = pathlib.Path(__file__).resolve().parent.parent
    fw, fh, stride, glyphs = load_font(root / "font" / "cjk24.golemfnt")

    px = [BG] * (size * size)
    b = max(2, size // 32)  # 外框寬
    for y in range(size):
        for x in range(size):
            if x < b or y < b or x >= size - b or y >= size - b:
                px[y * size + x] = FRAME

    text = "銀河"
    missing = [c for c in text if c not in glyphs]
    if missing:
        raise SystemExit("字型子集裡沒有 %s（把它加進 text/ 的資料檔再重烘）" % "".join(missing))
    scale = max(1, (size - 4 * b) // (fw * len(text)))
    tw, th = fw * scale * len(text), fh * scale
    ox, oy = (size - tw) // 2, (size - th) // 2
    for i, ch in enumerate(text):
        g = glyphs[ch]
        for gy in range(fh):
            row = g[gy * stride:(gy + 1) * stride]
            for gx in range(fw):
                if not (row[gx // 8] >> (7 - gx % 8)) & 1:
                    continue
                for sy in range(scale):
                    for sx in range(scale):
                        x = ox + (i * fw + gx) * scale + sx
                        y = oy + gy * scale + sy
                        if 0 <= x < size and 0 <= y < size:
                            px[y * size + x] = INK
    write_png(out, size, px)
    print("%s %d×%d" % (out, size, size))


if __name__ == "__main__":
    main(sys.argv)
