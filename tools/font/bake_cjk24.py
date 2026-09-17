"""烘製 24×24 中文點陣子集（docs/spec/008 §4）。

    tools/font/bake.sh            # docker 包裝（需要網路裝 Pillow）
    python3 bake_cjk24.py <STD.24M> <Noto TC .otf> <text 目錄> <輸出 .bin> [額外字…]

- 字集：text/*.json 所有 translation 的字元，加額外字。空白不烘。
- 來源：Big5 常用／次常用漢字用倚天 24 點明體 STD.24M（ETUNPACK 解壓，tools/font/etunpack.py）；
  其餘（標點、全形符號、ASCII）用 Noto Sans CJK TC 以 Pillow 點陣化。**fallback 字數是品質指標**，會印出來。
- 驗證 oracle：STD.24M 第 0 字必須是「一」（只有一條橫線），「中」要看得出中間一豎。
- 輸出格式：`PWCJK24\\0`、u32 字數，之後每字 u32 碼點、u8 來源（0 倚天、1 Noto）、72 bytes（每列 3 bytes，MSB 在左）。

⚠ 倚天中文系統是商業軟體，字模的散布授權未確認；本 repo 維持 private 期間可以放子集，公開前要換來源或取得授權（docs/spec/008 §4）。
"""
import json
import pathlib
import struct
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import etunpack  # noqa: E402

N_COMMON = 5401


def big5_index(ch):
    """倚天 STDFONT 的字序（knowledge-base eten-bitmap-font 的分區公式）；不是漢字區回 None。"""
    try:
        b = ch.encode("big5")
    except UnicodeEncodeError:
        return None
    if len(b) != 2:
        return None
    hi, lo = b

    def raw(h, l):
        return (h - 0xA1) * 157 + ((l - 0x40) if l < 0x7F else (l - 0x62))

    r = raw(hi, lo)
    if r < raw(0xA4, 0x40):
        return None  # 符號區：24 點沒有對應檔
    if r <= raw(0xC6, 0x7E):
        return r - raw(0xA4, 0x40)
    if r >= raw(0xC9, 0x40):
        return N_COMMON + (r - raw(0xC9, 0x40))
    return None


def eten_glyph(data, idx):
    g = data[idx * 72:(idx + 1) * 72]
    return bytes(g) if len(g) == 72 else None


def noto_glyph(font, ch):
    from PIL import Image, ImageDraw
    img = Image.new("L", (24, 24), 0)
    d = ImageDraw.Draw(img)
    box = d.textbbox((0, 0), ch, font=font)
    w, h = box[2] - box[0], box[3] - box[1]
    # 全形字置中；標點照字身位置（用字型的 ascent 對齊，不做垂直置中）
    x = (24 - w) // 2 - box[0]
    d.text((x, 0), ch, font=font, fill=255)
    out = bytearray(72)
    for y in range(24):
        for xx in range(24):
            if img.getpixel((xx, y)) >= 128:
                out[y * 3 + xx // 8] |= 0x80 >> (xx % 8)
    return bytes(out)


def art(g):
    return "\n".join("".join("#" if g[y * 3 + x // 8] & (0x80 >> (x % 8)) else "." for x in range(24)) for y in range(24))


def main(argv):
    if len(argv) < 5:
        print(__doc__)
        return 2
    std24, noto, text_dir, out = argv[1:5]
    extra = "".join(argv[5:])
    _, _, _, data, _ = etunpack.unpack(std24)
    first = eten_glyph(data, 0)
    rows = [first[y * 3:y * 3 + 3] for y in range(24)]
    filled = [r for r in rows if any(r)]
    if not (1 <= len(filled) <= 3):
        print("STD.24M 第 0 字不像「一」（有筆畫的列數 %d），索引或解壓有問題：\n%s" % (len(filled), art(first)), file=sys.stderr)
        return 1
    chars = set(extra)
    for p in sorted(pathlib.Path(text_dir).glob("*.json")):
        if p.name == "sources.json":
            continue
        for e in json.loads(p.read_text(encoding="utf-8")).get("entries", []):
            chars.update(e.get("translation", ""))
    chars = sorted(c for c in chars if not c.isspace())
    from PIL import ImageFont
    font = ImageFont.truetype(noto, 22)
    glyphs, fallback = [], []
    for ch in chars:
        idx = big5_index(ch)
        g = eten_glyph(data, idx) if idx is not None else None
        if g is not None and any(g):
            glyphs.append((ord(ch), 0, g))
        else:
            glyphs.append((ord(ch), 1, noto_glyph(font, ch)))
            fallback.append(ch)
    blob = bytearray(b"PWCJK24\0" + struct.pack("<I", len(glyphs)))
    for cp, src, g in glyphs:
        blob += struct.pack("<IB", cp, src) + g
    pathlib.Path(out).write_bytes(bytes(blob))
    listing = pathlib.Path(out).with_suffix(".txt")
    listing.write_text("".join(chars) + "\n", encoding="utf-8")
    zhong = next((g for cp, src, g in glyphs if cp == ord("中")), None)
    print("烘出 %d 字：倚天 %d、Noto fallback %d（%s）→ %s" % (len(glyphs), len(glyphs) - len(fallback), len(fallback), "".join(fallback), out))
    if zhong:
        print("「中」：\n" + art(zhong))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
