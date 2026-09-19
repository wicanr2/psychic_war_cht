"""F4 說明頁的逐像素驗收（docs/spec/012 §5 第 3 項）：期望值由 text/help.json 與字型排算，不看前端自己的輸出。

    tools/py.sh tools/help_check.py <截圖.png> [--text text] [--font font] [--scale 3]

排版與 apps/psychicwar/help.go 的 DrawTextPage 相同：
一格 8×scale 像素、半形字（< 0x80）佔半格、整頁置中（38 格寬、行數 ＝ 內容行數）、背景色號 0、前景色號 15。
"""
import json
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import pbl  # noqa: E402

ROOT = pathlib.Path(__file__).resolve().parent.parent
COLS, ROWS = 38, 22


def load_font(path):
    b = pathlib.Path(path).read_bytes()
    assert b[:8] == b"GOLEMFNT", path
    import struct
    w, h, n = struct.unpack_from("<HHI", b, 8)
    rb = (w + 7) // 8
    out, off = {}, 16
    for _ in range(n):
        cp = struct.unpack_from("<I", b, off)[0]
        off += 5
        out[chr(cp)] = b[off:off + rb * h]
        off += rb * h
    return w, h, rb, out


def expected(lines, font, scale, w, h):
    fw, fh, rb, glyphs = font
    cell = 8 * scale
    px = bytearray(w * h)  # 色號 0（背景）
    sx = (w - COLS * cell) // 2
    sy = (h - len(lines) * cell) // 2
    sx, sy = max(sx, 0), max(sy, 0)
    gs = max(cell // fw, 1)
    for row, s in enumerate(lines):
        pen = 0
        for ch in s:
            adv = 1 if ord(ch) < 0x80 else 2
            g = glyphs.get(ch)
            if g is None:
                pen += adv
                continue
            x0, y0 = sx + pen * cell // 2, sy + row * cell
            pen += adv
            for gy in range(fh):
                for gx in range(fw):
                    if not g[gy * rb + gx // 8] & (0x80 >> (gx % 8)):
                        continue
                    for py in range(gs):
                        for pxi in range(gs):
                            x, y = x0 + gx * gs + pxi, y0 + gy * gs + py
                            if 0 <= x < w and 0 <= y < h:
                                px[y * w + x] = 15
    return px


def main(argv):
    if not argv:
        print(__doc__)
        return 2
    shot = argv[0]
    text_dir = ROOT / (argv[argv.index("--text") + 1] if "--text" in argv else "text")
    font_dir = ROOT / (argv[argv.index("--font") + 1] if "--font" in argv else "font")
    scale = int(argv[argv.index("--scale") + 1]) if "--scale" in argv else 3
    doc = json.loads((text_dir / "help.json").read_text(encoding="utf-8"))
    lines = doc["lines"]
    if len(lines) > ROWS:
        print("說明頁 %d 行，超過 %d 行" % (len(lines), ROWS))
        return 1
    w, h, spx = pbl.screen_indices(ROOT / shot)
    if (w, h) != (320 * scale, 200 * scale):
        print("截圖大小 %d×%d，不是 %d×%d" % (w, h, 320 * scale, 200 * scale))
        return 1
    exp = expected(lines, load_font(font_dir / "cjk24.golemfnt"), scale, w, h)
    bad = sum(1 for i in range(w * h) if exp[i] != spx[i])
    print("說明頁：%d 行，不同像素 %d / %d" % (len(lines), bad, w * h))
    return 0 if bad == 0 else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
