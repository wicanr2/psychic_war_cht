"""烘製中文點陣子集（docs/spec/009 §2）：24×24 給 FONT.BIN 路徑、16×15 給小字型路徑。

    tools/font/bake.sh [額外字…]
    python3 bake_cjk24.py <字型來源目錄> <Noto TC .otf> <text 目錄> <輸出目錄> [額外字…]

- 字集：text/*.json（schema psychic-war-text/1 與 psychic-war-baked/1）所有 translation 的字元，加額外字。空白不烘。
- 24×24：Big5 漢字用倚天 24 點明體 STD.24M（ETUNPACK 解壓）；其餘用 Noto Sans CJK TC 點陣化。
- 16×15：Big5 漢字用倚天 STDFONT.15、全形符號用 SPCFONT.15；其餘用 Noto 點陣化。
- 驗證 oracle：兩份倚天字型的「一」都只有一到三列有筆畫，「中」印出來看。
- **fallback 字數是品質指標**，兩種尺寸都會印出來。
- **缺字就失敗**（結束碼 1）：Noto 也沒有的字（點陣化結果是空白，或與 Noto 的缺字方框 U+E000 相同）列出來，不寫輸出檔。
- 輸出 GOLEMFNT（dosgolem 規格 202 §2.1）：`GOLEMFNT`、u16 W、u16 H、u32 字數，每字 u32 碼點、u8 來源（0 倚天、1 Noto）、字模（每列 (W+7)/8 bytes，MSB 在左）。
  檔名 cjk24.golemfnt、cjk16.golemfnt，另寫字集清單 charset.txt。

⚠ 倚天中文系統是商業軟體，字模的散布授權未確認；本 repo private 期間放子集，公開前要換來源或取得授權。
"""
import json
import pathlib
import struct
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import etunpack  # noqa: E402

N_COMMON = 5401


def big5_raw(ch):
    try:
        b = ch.encode("big5")
    except UnicodeEncodeError:
        return None
    if len(b) != 2:
        return None
    hi, lo = b
    return (hi - 0xA1) * 157 + ((lo - 0x40) if lo < 0x7F else (lo - 0x62))


def raw_of(hi, lo):
    return (hi - 0xA1) * 157 + ((lo - 0x40) if lo < 0x7F else (lo - 0x62))


def eten_index(ch):
    """回（'spc' 或 'std', 索引）；不是倚天有的字回 None（knowledge-base eten-bitmap-font 的分區公式）。"""
    r = big5_raw(ch)
    if r is None:
        return None
    if r <= raw_of(0xA3, 0xBF):
        return ("spc", r)
    if raw_of(0xA4, 0x40) <= r <= raw_of(0xC6, 0x7E):
        return ("std", r - raw_of(0xA4, 0x40))
    if r >= raw_of(0xC9, 0x40):
        return ("std", N_COMMON + (r - raw_of(0xC9, 0x40)))
    return None


def glyph_at(data, idx, w, h):
    n = (w + 7) // 8 * h
    g = data[idx * n:(idx + 1) * n]
    return bytes(g) if len(g) == n and any(g) else None


def noto_glyph(font, ch, w, h):
    from PIL import Image, ImageDraw
    img = Image.new("L", (w, h), 0)
    d = ImageDraw.Draw(img)
    box = d.textbbox((0, 0), ch, font=font)
    x = (w - (box[2] - box[0])) // 2 - box[0]
    # 垂直位置以「中」的字框置中，所有補字共用同一條基線（句點、底線不會被推出字格）
    em = d.textbbox((0, 0), "中", font=font)
    y = (h - (em[3] - em[1])) // 2 - em[1]
    d.text((x, y), ch, font=font, fill=255)
    rb = (w + 7) // 8
    out = bytearray(rb * h)
    for y in range(h):
        for xx in range(w):
            if img.getpixel((xx, y)) >= 128:
                out[y * rb + xx // 8] |= 0x80 >> (xx % 8)
    return bytes(out)


def art(g, w, h):
    rb = (w + 7) // 8
    return "\n".join("".join("#" if g[y * rb + x // 8] & (0x80 >> (x % 8)) else "." for x in range(w)) for y in range(h))


def check_one(g, w, h, name):
    rb = (w + 7) // 8
    rows = sum(1 for y in range(h) if any(g[y * rb:(y + 1) * rb]))
    if not 1 <= rows <= 3:
        raise SystemExit("%s 的「一」有 %d 列筆畫，索引或解壓有問題：\n%s" % (name, rows, art(g, w, h)))


def write(path, w, h, glyphs):
    blob = bytearray(b"GOLEMFNT" + struct.pack("<HHI", w, h, len(glyphs)))
    for cp, src, g in glyphs:
        blob += struct.pack("<IB", cp, src) + g
    pathlib.Path(path).write_bytes(bytes(blob))


def main(argv):
    if len(argv) < 5:
        print(__doc__)
        return 2
    src, noto, text_dir, out = pathlib.Path(argv[1]), argv[2], argv[3], pathlib.Path(argv[4])
    extra = "".join(argv[5:])
    _, _, _, std24, _ = etunpack.unpack(str(src / "STD.24M"))
    std15 = (src / "STDFONT.15").read_bytes()
    spc15 = (src / "SPCFONT.15").read_bytes()
    check_one(glyph_at(std24, 0, 24, 24), 24, 24, "STD.24M")
    check_one(glyph_at(std15, 0, 16, 15), 16, 15, "STDFONT.15")

    chars = set(extra)
    for p in sorted(pathlib.Path(text_dir).glob("*.json")):
        doc = json.loads(p.read_text(encoding="utf-8"))
        if doc.get("schema") not in ("psychic-war-text/1", "psychic-war-baked/1"):
            continue
        for e in doc["entries"]:
            chars.update(e.get("translation", ""))
    chars = sorted(c for c in chars if not c.isspace())

    from PIL import ImageFont
    noto24 = ImageFont.truetype(noto, 22)
    noto16 = ImageFont.truetype(noto, 14)
    out.mkdir(parents=True, exist_ok=True)
    report, missing, baked = [], set(), []
    for name, w, h, noto_font in (("cjk24", 24, 24, noto24), ("cjk16", 16, 15, noto16)):
        glyphs, fallback = [], []
        for ch in chars:
            idx = eten_index(ch)
            g = None
            if idx is not None:
                kind, i = idx
                if w == 24:
                    g = glyph_at(std24, i, 24, 24) if kind == "std" else None
                else:
                    g = glyph_at(std15 if kind == "std" else spc15, i, 16, 15)
            if g is not None:
                glyphs.append((ord(ch), 0, g))
            else:
                g = noto_glyph(noto_font, ch, w, h)
                if not any(g) or g == noto_glyph(noto_font, "\ue000", w, h):
                    missing.add(ch)
                glyphs.append((ord(ch), 1, g))
                fallback.append(ch)
        baked.append((name, w, h, glyphs))
        report.append("%s：%d 字，倚天 %d、Noto fallback %d（%s）" % (name, len(glyphs), len(glyphs) - len(fallback), len(fallback), "".join(fallback)))
        zhong = next((g for cp, _, g in glyphs if cp == ord("中")), None)
        if zhong:
            report.append("「中」%d×%d：\n%s" % (w, h, art(zhong, w, h)))
    if missing:
        print("缺字 %d：%s（兩套字型都沒有，沒有寫輸出檔）" % (len(missing), " ".join("%s U+%04X" % (c, ord(c)) for c in sorted(missing))))
        return 1
    for name, w, h, glyphs in baked:  # 兩種尺寸都沒有缺字才寫，不留半套輸出
        write(out / (name + ".golemfnt"), w, h, glyphs)
    (out / "charset.txt").write_text("".join(chars) + "\n", encoding="utf-8")
    print("\n".join(report))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
