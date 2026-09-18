"""中文疊字逐像素驗收（docs/spec/009 §6 第 2 項）：期望值由原版畫面推算，不用轉譯層自己的紀錄。

    tools/py.sh tools/overlay_check.py <情境名> <參照.png> <截圖.png> [--text text] [--without KEY]
    tools/py.sh tools/overlay_check.py --baked <參照.png> <截圖.png> [--text text] [--keys k1,k2] [--without KEY]

- 參照：dosgolem `cmd/step` 以同一狀態、同一動作跑出的原版畫面（-scale 1，320×200）。
- 截圖：`cmd/pwstep` 以同一狀態、同一動作跑出的放大畫面加疊字層（-scale 3，960×600）。
- 情境與要檢查的 key、行寫在 tools/overlay_cases.json。
- 每一行：用原版字模（FONT.BIN 8×8，或 PW_UNP.EXE 內建的小字型 6×6）把英文原文排成遮罩，在參照畫面上找唯一符合的位置
  （遮罩為 1 的像素同一種顏色、為 0 的同一種顏色；透明格不看）。背景色 ＝ 該行非透明格出現最多的顏色，前景 ＝ 第二多。
- 期望區塊：非透明格填背景色，再依 docs/spec/009 §2 的幾何以 font/*.golemfnt 的字模畫前景色；透明格是參照畫面放大 3 倍。
  `--without KEY`（反向對照）：這一則當作沒有譯文，整行期望是參照畫面放大 3 倍。
- `--baked`：圖檔內嵌文字（docs/spec/011）。位置不用搜尋——`text/baked.json` 直接給了畫面座標與格數；
  期望區塊 ＝ 參照畫面該塊的背景色填滿再畫中文字模，顏色同樣取「最多／第二多」。
- 全部行都差 0 才結束碼 0。
"""
import json
import pathlib
import struct
import sys
import zlib

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import text_extract as te  # noqa: E402

ROOT = pathlib.Path(__file__).resolve().parent.parent
S = 3
GEOM = {  # 字型路徑 → (格寬, 格高, 中文字型, 字模偏移 x, y（放大後像素）)
    "font8": (8, 8, "cjk24", 0, 0),
    "font6": (6, 7, "cjk16", 1, 3),
}


def read_png(path):
    """讀 8 位元 RGB／RGBA、不交錯的 PNG，回 (w, h, RGB bytes)。"""
    b = pathlib.Path(path).read_bytes()
    assert b[:8] == b"\x89PNG\r\n\x1a\n", path
    pos, idat, w = 8, b"", 0
    while pos < len(b):
        n = struct.unpack(">I", b[pos:pos + 4])[0]
        typ, data = b[pos + 4:pos + 8], b[pos + 8:pos + 8 + n]
        if typ == b"IHDR":
            w, h, depth, ctype, _, _, inter = struct.unpack(">IIBBBBB", data)
            assert depth == 8 and ctype in (2, 6) and inter == 0, (path, depth, ctype, inter)
            bpp = 3 if ctype == 2 else 4
        elif typ == b"IDAT":
            idat += data
        pos += 12 + n
    raw = zlib.decompress(idat)
    stride = w * bpp
    out, prev = bytearray(), bytearray(stride)
    for y in range(h):
        f = raw[y * (stride + 1)]
        line = bytearray(raw[y * (stride + 1) + 1:(y + 1) * (stride + 1)])
        for i in range(stride):
            a = line[i - bpp] if i >= bpp else 0
            up = prev[i]
            c = prev[i - bpp] if i >= bpp else 0
            if f == 1:
                line[i] = (line[i] + a) & 255
            elif f == 2:
                line[i] = (line[i] + up) & 255
            elif f == 3:
                line[i] = (line[i] + (a + up) // 2) & 255
            elif f == 4:
                p = a + up - c
                pa, pb, pc = abs(p - a), abs(p - up), abs(p - c)
                line[i] = (line[i] + (a if pa <= pb and pa <= pc else up if pb <= pc else c)) & 255
        prev = line
        for x in range(w):
            out += line[x * bpp:x * bpp + 3]
    return w, h, bytes(out)


def load_golemfnt(path):
    b = pathlib.Path(path).read_bytes()
    assert b[:8] == b"GOLEMFNT", path
    w, h, n = struct.unpack_from("<HHI", b, 8)
    rb = (w + 7) // 8 * h
    out, o = {}, 16
    for _ in range(n):
        cp = struct.unpack_from("<I", b, o)[0]
        out[chr(cp)] = b[o + 5:o + 5 + rb]
        o += 5 + rb
    return w, h, out


def font8_masks(orig):
    f = (pathlib.Path(orig) / "FONT.BIN").read_bytes()
    return lambda c: [[(f[(c - 0x20) * 8 + r] >> (7 - x)) & 1 for x in range(8)] for r in range(8)]


def font6_masks(orig):
    d = (pathlib.Path(orig) / "PW.EXE").read_bytes()
    unp = te.unexepack.build_mz(te.unexepack.unpack(d))
    hdr = struct.unpack_from("<H", unp, 8)[0] * 16
    code = unp[hdr + te.SEG_BASE:hdr + te.SEG_BASE + 0x10000]

    def mask(c):
        ptr = struct.unpack_from("<H", code, 0xB13D + (c - 0x20) * 2)[0]
        rows = [[(code[ptr + r] >> (7 - x)) & 1 for x in range(6)] for r in range(6)]
        return rows + [[0] * 6]
    return mask


def line_chars(e, line):
    raw = [c for c in te.decode_shown(e["original"]) if c >= 0x20]
    widths = te.line_widths(e)
    start = sum(widths[:line])
    return raw[start:start + widths[line]]


def search(px, cm, n, cw, ch, skip):
    hits = []
    for y in range(0, 200 - ch + 1):
        for x in range(0, 320 - cw * n + 1):
            bg = fg = None
            ok = True
            for i in range(n):
                if skip[i]:
                    continue
                for r in range(ch):
                    for b in range(cw):
                        v = px(x + i * cw + b, y + r)
                        if cm[i][r][b]:
                            if fg is None:
                                fg = v
                            elif v != fg:
                                ok = False
                                break
                        elif bg is None:
                            bg = v
                        elif v != bg:
                            ok = False
                            break
                    if not ok:
                        break
                if not ok:
                    break
            if ok and fg is not None and fg != bg:
                hits.append((x, y))
    return hits


def check_baked(argv):
    """圖檔內嵌文字的逐像素驗收（docs/spec/011 §5 第 2 項）；argv[0] 是 --baked。"""
    ref_png, shot_png = argv[1], argv[2]
    text_dir = ROOT / (argv[argv.index("--text") + 1] if "--text" in argv else "text")
    without = argv[argv.index("--without") + 1] if "--without" in argv else None
    only = argv[argv.index("--keys") + 1].split(",") if "--keys" in argv else None
    doc = json.loads((text_dir / "baked.json").read_text(encoding="utf-8"))
    rw, rh, ref = read_png(ROOT / ref_png)
    sw, sh, shot = read_png(ROOT / shot_png)
    assert (rw, rh, sw, sh) == (320, 200, 960, 600), (rw, rh, sw, sh)
    px = lambda x, y: ref[3 * (y * 320 + x):3 * (y * 320 + x) + 3]  # noqa: E731
    fw, fh, glyphs = load_golemfnt(ROOT / "font/cjk24.golemfnt")
    rb = (fw + 7) // 8
    failed = 0
    for e in doc["entries"]:
        if only and e["key"] not in only:
            continue
        tr = "" if e["key"] == without else e["translation"]
        cells = e["text"][2]
        x = e["screen"][0] + e["text"][0]
        y = e["screen"][1] + e["text"][1]
        cw = ch = 8
        counts = {}
        for r in range(ch):
            for b in range(cells * cw):
                v = px(x + b, y + r)
                counts[v] = counts.get(v, 0) + 1
        order = sorted(counts, key=lambda v: -counts[v])
        bg = order[0]
        fg = order[1] if len(order) > 1 else order[0]
        bw, bh = cells * cw * S, ch * S
        exp = bytearray(3 * bw * bh)
        for yy in range(bh):
            for xx in range(bw):
                v = px(x + xx // S, y + yy // S) if not tr else bg
                exp[3 * (yy * bw + xx):3 * (yy * bw + xx) + 3] = v
        if tr:
            for i, c in enumerate(tr.ljust(cells)[:cells]):
                if c == " " or c not in glyphs:
                    continue
                g = glyphs[c]
                for r in range(fh):
                    for b in range(fw):
                        if g[r * rb + b // 8] & (0x80 >> (b % 8)):
                            xx, yy = i * cw * S + b, r
                            if xx < (i + 1) * cw * S and yy < bh:
                                exp[3 * (yy * bw + xx):3 * (yy * bw + xx) + 3] = fg
        bad = sum(1 for yy in range(bh) for xx in range(bw)
                  if exp[3 * (yy * bw + xx):3 * (yy * bw + xx) + 3] != shot[3 * ((y * S + yy) * 960 + x * S + xx):3 * ((y * S + yy) * 960 + x * S + xx) + 3])
        failed += bad != 0
        print("%-28s @ (%3d,%3d) %d 格：%s，不同像素 %d / %d" % (e["key"], x, y, cells, "中文" if tr else "原版英文", bad, bw * bh))
    return 0 if failed == 0 else 1


def main(argv):
    if argv and argv[0] == "--baked":
        return check_baked(argv)
    if len(argv) < 3:
        print(__doc__)
        return 2
    name, ref_png, shot_png = argv[:3]
    text_dir = str(ROOT / argv[argv.index("--text") + 1]) if "--text" in argv else str(ROOT / "text")
    without = argv[argv.index("--without") + 1] if "--without" in argv else None
    case = json.loads((ROOT / "tools/overlay_cases.json").read_text(encoding="utf-8"))["cases"][name]
    have = te.load_text(text_dir)
    orig = ROOT / "workplace/original/psychic-war"
    masks = {"font8": font8_masks(orig), "font6": font6_masks(orig)}
    fonts = {k: load_golemfnt(ROOT / "font" / (GEOM[k][2] + ".golemfnt")) for k in GEOM}
    rw, rh, ref = read_png(ROOT / ref_png)
    sw, sh, shot = read_png(ROOT / shot_png)
    assert (rw, rh, sw, sh) == (320, 200, 960, 600), (rw, rh, sw, sh)
    px = lambda x, y: ref[3 * (y * 320 + x):3 * (y * 320 + x) + 3]  # noqa: E731
    failed = 0
    for key, line in case["targets"]:
        e = have[key]
        root = have[e["same_as"]] if e["same_as"] else e
        tr = "" if key == without else root["translation"]
        cw, ch, fname, gx, gy = GEOM[e["font"]]
        chars = line_chars(e, line)
        n = len(chars)
        cells = te.layout(tr, te.line_widths(e))[0][line] if tr else ""
        cells = cells.ljust(n)
        transparent = [cells[i] == " " and chars[i] == 0x20 for i in range(n)] if tr else [False] * n
        cm = [masks[e["font"]](c) for c in chars]
        hits = search(px, cm, n, cw, ch, transparent)
        if not hits:  # 輸入框裡有玩家打的字：原文是空白的格子不比對，再找一次
            hits = search(px, cm, n, cw, ch, [chars[i] == 0x20 for i in range(n)])
        label = "%s 第 %d 行「%s」" % (key, line, bytes(chars).decode("latin-1"))
        if len(hits) != 1:
            print("%s：原版畫面上找到 %d 處，不是 1 處 %s" % (label, len(hits), hits[:4]))
            failed += 1
            continue
        x, y = hits[0]
        counts = {}
        for i in range(n):
            if transparent[i]:
                continue
            for r in range(ch):
                for b in range(cw):
                    v = px(x + i * cw + b, y + r)
                    counts[v] = counts.get(v, 0) + 1
        order = sorted(counts, key=lambda v: -counts[v])
        bg = order[0]
        fg = order[1] if len(order) > 1 else order[0]
        bw, bh = n * cw * S, ch * S
        exp = bytearray(3 * bw * bh)
        for yy in range(bh):
            for xx in range(bw):
                i = xx // (cw * S)
                v = px(x + xx // S, y + yy // S) if (not tr or transparent[i]) else bg
                exp[3 * (yy * bw + xx):3 * (yy * bw + xx) + 3] = v
        if tr:
            fw, fh, glyphs = fonts[e["font"]]
            rb = (fw + 7) // 8
            for i, c in enumerate(cells[:n]):
                if transparent[i] or c == " " or c not in glyphs:
                    continue
                g = glyphs[c]
                for r in range(fh):
                    for b in range(fw):
                        if g[r * rb + b // 8] & (0x80 >> (b % 8)):
                            xx, yy = i * cw * S + gx + b, gy + r
                            if xx < (i + 1) * cw * S and yy < bh:
                                exp[3 * (yy * bw + xx):3 * (yy * bw + xx) + 3] = fg
        bad = sum(1 for yy in range(bh) for xx in range(bw)
                  if exp[3 * (yy * bw + xx):3 * (yy * bw + xx) + 3] != shot[3 * ((y * S + yy) * 960 + x * S + xx):3 * ((y * S + yy) * 960 + x * S + xx) + 3])
        failed += bad != 0
        print("%s @ (%d,%d) %d 格：%s，不同像素 %d / %d" % (label, x, y, n, "中文" if tr else "原版英文", bad, bw * bh))
    return 0 if failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
