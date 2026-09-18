"""把 `text/baked.json` 的一筆畫成預覽 PNG（原版圖 ＋ 中文疊字，放大 3 倍）。

    tools/py.sh tools/baked_preview.py <key…> [--out workplace/pbl/preview] [--text text] [--file <另一份 json>]
    tools/py.sh tools/baked_preview.py --all

畫法與執行期相同（docs/spec/011 §3）：整個中文矩形填背景色（區塊裡最多的色號），
再以前景色（第二多）畫字模；一格 8×8（cjk24）或 6×7（cjk16，字模偏移 (1,3)）。
預覽只是給人看的，逐像素驗收在 `tools/baked_run.sh` 與 Go 的合成畫面測試。
"""
import json
import pathlib
import struct
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import pbl  # noqa: E402

ROOT = pathlib.Path(__file__).resolve().parent.parent
S = 3
GEOM = {"cjk24": (8, 8, "cjk24", 0, 0), "cjk16": (6, 7, "cjk16", 1, 3)}


def load_font(path):
    b = pathlib.Path(path).read_bytes()
    assert b[:8] == b"GOLEMFNT", path
    w, h, n = struct.unpack_from("<HHI", b, 8)
    rb = (w + 7) // 8
    out, off = {}, 16
    for _ in range(n):
        cp = struct.unpack_from("<I", b, off)[0]
        off += 5
        out[chr(cp)] = b[off:off + rb * h]
        off += rb * h
    return w, h, rb, out


def render(px, w, h, e, fonts):
    """回放大 3 倍的 RGB bytes。"""
    cw, ch, fname, gx, gy = GEOM[e.get("font") or "cjk24"]
    tx, ty, cells = e["text"]
    counts = {}
    for r in range(ch):
        for c in range(cells * cw):
            x, y = tx + c, ty + r
            if 0 <= x < w and 0 <= y < h:
                v = px[y * w + x]
                counts[v] = counts.get(v, 0) + 1
    order = sorted(counts, key=lambda v: -counts[v])
    bg = order[0] if order else 0
    fg = order[1] if len(order) > 1 else bg
    out = bytearray(3 * w * S * h * S)
    for y in range(h * S):
        for x in range(w * S):
            v = px[(y // S) * w + (x // S)]
            i = 3 * (y * w * S + x)
            out[i:i + 3] = bytes(pbl.EGA[v])
    def put(x, y, v):
        if 0 <= x < w * S and 0 <= y < h * S:
            i = 3 * (y * w * S + x)
            out[i:i + 3] = bytes(pbl.EGA[v])
    for r in range(ch * S):
        for c in range(cells * cw * S):
            put(tx * S + c, ty * S + r, bg)
    fw, fh, rb, glyphs = fonts[fname]
    for i, cch in enumerate(e["translation"][:cells]):
        g = glyphs.get(cch)
        if cch == " " or g is None:
            continue
        for r in range(fh):
            for b in range(fw):
                if g[r * rb + b // 8] & (0x80 >> (b % 8)):
                    x = tx * S + i * cw * S + gx + b
                    y = ty * S + gy + r
                    if x < (tx + (i + 1) * cw) * S:
                        put(x, y, fg)
    return out


def main(argv):
    text_dir = ROOT / (argv[argv.index("--text") + 1] if "--text" in argv else "text")
    out_dir = ROOT / (argv[argv.index("--out") + 1] if "--out" in argv else "workplace/pbl/preview")
    keys = [a for a in argv if not a.startswith("--") and argv[argv.index(a) - 1] not in ("--text", "--out")]
    src = ROOT / argv[argv.index("--file") + 1] if "--file" in argv else text_dir / "baked.json"
    doc = json.loads(src.read_text(encoding="utf-8"))
    fonts = {n: load_font(ROOT / "font" / (n + ".golemfnt")) for n in ("cjk24", "cjk16")}
    orig = ROOT / "workplace/original/psychic-war"
    out_dir.mkdir(parents=True, exist_ok=True)
    cache = {}
    n = 0
    for e in doc["entries"]:
        if keys and e["key"] not in keys and "--all" not in argv:
            continue
        if e["file"] not in cache:
            cache[e["file"]] = (orig / e["file"]).read_bytes()
        data = cache[e["file"]]
        w, h, px = pbl.decode(data, pbl.images(data)[e["image"]][1])
        rgb = render(px, w, h, e, fonts)
        name = e["key"].replace(":", "_") + ".png"
        raw = b"".join(b"\0" + bytes(rgb[3 * (y * w * S):3 * ((y + 1) * w * S)]) for y in range(h * S))
        import zlib

        def chunk(tag, body):
            c = tag + body
            return struct.pack(">I", len(body)) + c + struct.pack(">I", zlib.crc32(c) & 0xFFFFFFFF)

        (out_dir / name).write_bytes(b"\x89PNG\r\n\x1a\n"
                                    + chunk(b"IHDR", struct.pack(">IIBBBBB", w * S, h * S, 8, 2, 0, 0, 0))
                                    + chunk(b"IDAT", zlib.compress(bytes(raw), 9))
                                    + chunk(b"IEND", b""))
        print(e["key"], "→", out_dir / name)
        n += 1
    if n == 0:
        print("沒有符合的 key（用 --all 畫全部）")
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
