"""中文疊字的實跑驗收（docs/spec/008 §4 第 2 項）：期望畫面以原版算，不用前端自己的紀錄。

    tools/py.sh tools/overlay_check.py <情境> <原版目錄> <text 目錄> <參照.frame> <參照.rgb> <截圖.rgb> [--expect-english KEY]

- 原版位置：把這一行英文（文本檔 original）用 FONT.BIN 的 8×8 字模排出遮罩，在參照畫面（probe 跑同一狀態、同一按鍵的色號畫面）
  上找「遮罩為 1 的像素全是前景、為 0 的全是背景」的位置（x、y 都是 4 的倍數）。必須剛好一處。
- 顏色：該行範圍內出現最多的色號是背景、第二多是前景；RGB 從參照的 RGB 畫面取。
- 期望區塊：`--expect-english KEY` 的那一則是原版 RGB 放大 3 倍；其他則依 docs/spec/008 §3.3 排版，
  font/cjk24.bin 的字模以前景色畫在背景色上。
- 截圖是 960×600 的 RGB（`convert shot.png shot.rgb`）。每一行印出不同像素數；全部為 0 才結束碼 0。
"""
import json
import pathlib
import struct
import sys

W, H, S = 320, 200, 3
TARGETS = {
    "wall": [("I_MENUH.BIN:0530", 0), ("I_MENUH.BIN:0530", 1)],
    "launchpad": [("I_MENUH.BIN:09F0", 0), ("I_MENUH.BIN:09F0", 1), ("I_MENUH.BIN:0A10", 0), ("I_MENUH.BIN:0A40", 0),
                  ("I_MENUH.BIN:0A60", 0), ("I_MENUH.BIN:0A80", 0), ("I_MENUH.BIN:0A90", 0)],
}


def load_entries(text_dir):
    out = {}
    for p in pathlib.Path(text_dir).glob("*.json"):
        doc = json.loads(p.read_text(encoding="utf-8"))
        if doc.get("schema") == "psychic-war-text/1":
            for e in doc["entries"]:
                out[e["key"]] = e
    return out


def decode(shown):
    raw, i = bytearray(), 0
    while i < len(shown):
        if shown[i] == "{" and shown[i + 3:i + 4] == "}":
            raw.append(int(shown[i + 1:i + 3], 16))
            i += 4
        else:
            raw.append(ord(shown[i]))
            i += 1
    return bytes(raw)


def line_bytes(e, line):
    raw = decode(e["original"])
    if e["kind"] == "option":
        return raw.ljust(10, b" ")[:10]
    return raw[:16] if line == 0 else raw[16:31]


def layout(text, widths):
    out, line = [[] for _ in widths], 0
    for ch in text:
        if line >= len(widths):
            break
        if ch == "\n":
            line += 1
            continue
        if len(out[line]) == widths[line]:
            line += 1
            if line >= len(widths):
                break
        out[line].append(ch)
    return out


def find(frame, font8, chars):
    """回唯一的 (x, y, bg, fg)；找不到或不唯一回 None 與候選數。"""
    n = len(chars)
    mask = [[(font8[(c - 0x20) * 8 + r] >> (7 - b)) & 1 if c >= 0x20 else 0 for b in range(8)] for c in chars for r in range(8)]
    hits = []
    for y in range(0, H - 8 + 1, 4):
        for x in range(0, W - 8 * n + 1, 4):
            bg = fg = None
            ok = True
            for ci in range(n):
                for r in range(8):
                    bits = mask[ci * 8 + r]
                    row = y + r
                    for b in range(8):
                        v = frame[row * W + x + ci * 8 + b]
                        if bits[b]:
                            if fg is None:
                                fg = v
                            elif v != fg:
                                ok = False
                                break
                        else:
                            if bg is None:
                                bg = v
                            elif v != bg:
                                ok = False
                                break
                    if not ok:
                        break
                if not ok:
                    break
            if ok and fg is not None and fg != bg:
                hits.append((x, y, bg, fg))
    return hits


def load_cjk(path):
    b = pathlib.Path(path).read_bytes()
    n = struct.unpack_from("<I", b, 8)[0]
    out = {}
    for i in range(n):
        o = 12 + i * 77
        cp = struct.unpack_from("<I", b, o)[0]
        out[chr(cp)] = b[o + 5:o + 77]
    return out


def main(argv):
    if len(argv) < 6:
        print(__doc__)
        return 2
    scenario, orig, text_dir, ref_frame, ref_rgb, shot_rgb = argv[:6]
    english = argv[argv.index("--expect-english") + 1] if "--expect-english" in argv else None
    entries = load_entries(text_dir)
    font8 = (pathlib.Path(orig) / "FONT.BIN").read_bytes()
    cjk = load_cjk("font/cjk24.bin")
    frame = pathlib.Path(ref_frame).read_bytes()
    rgb = pathlib.Path(ref_rgb).read_bytes()
    shot = pathlib.Path(shot_rgb).read_bytes()
    if len(frame) != W * H or len(rgb) != 3 * W * H or len(shot) != 3 * W * S * H * S:
        print("尺寸不對：frame %d、rgb %d、截圖 %d" % (len(frame), len(rgb), len(shot)))
        return 2
    total_bad = 0
    for key, line in TARGETS[scenario]:
        e = entries[key]
        chars = line_bytes(e, line)
        hits = find(frame, font8, chars)
        if len(hits) != 1:
            print("%s 第 %d 行「%s」：原版畫面上找到 %d 處，不是 1 處" % (key, line, chars.decode("latin-1"), len(hits)))
            total_bad += 1
            continue
        x, y, bg, fg = hits[0]
        n = len(chars)
        rgb_of = lambda c: next(rgb[3 * i:3 * i + 3] for i in range(W * H) if frame[i] == c)  # noqa: E731
        bg_rgb, fg_rgb = rgb_of(bg), rgb_of(fg)
        bw, bh = n * 8 * S, 8 * S
        exp = bytearray(3 * bw * bh)
        if key == english:
            for yy in range(bh):
                for xx in range(bw):
                    i = 3 * ((y + yy // S) * W + x + xx // S)
                    exp[3 * (yy * bw + xx):3 * (yy * bw + xx) + 3] = rgb[i:i + 3]
        else:
            for i in range(bw * bh):
                exp[3 * i:3 * i + 3] = bg_rgb
            widths = [10] if e["kind"] == "option" else [16, 15]
            cells = layout(e["translation"], widths)[line]
            for ci, ch in enumerate(cells[:n]):
                g = cjk.get(ch)
                if g is None:
                    continue
                for gy in range(24):
                    for gx in range(24):
                        if g[gy * 3 + gx // 8] & (0x80 >> (gx % 8)):
                            p = 3 * (gy * bw + ci * 24 + gx)
                            exp[p:p + 3] = fg_rgb
        bad = 0
        for yy in range(bh):
            for xx in range(bw):
                p = 3 * (yy * bw + xx)
                q = 3 * ((y * S + yy) * W * S + x * S + xx)
                if exp[p:p + 3] != shot[q:q + 3]:
                    bad += 1
        total_bad += bad != 0
        print("%s 第 %d 行 @ (%d,%d) %d 格 背景 %d 前景 %d：%s，不同像素 %d / %d" % (
            key, line, x, y, n, bg, fg, "原版英文" if key == english else "中文", bad, bw * bh))
    return 0 if total_bad == 0 else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
