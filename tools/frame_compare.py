#!/usr/bin/env python3
"""frame_compare.py — dosgolem 色號陣列 vs DOSBox-X 畫面（issue #3）。

    tools/py.sh tools/frame_compare.py <golem.frame> <dosboxx.rgb> [<dosboxx.rgb> …] [--json out.json] [--diff out.png]

- golem.frame：320×200 色號（tools/states.sh 產出）。
- dosboxx.rgb：640×400 raw RGB（tools/dosboxx-ref.sh 產出）。給多張時挑不一致像素最少的那張。

比對的是**版面**，不是顏色：dosgolem 目前的畫面輸出沒有套用遊戲的色盤（issue #4），
所以對每個 dosgolem 色號取它在 DOSBox-X 最常對到的 RGB，當作兩邊的對應；
與對應不符的像素才算不一致。

同時檢查：
- DOSBox-X 畫面每個 2×2 區塊是否同色（不是的話縮放不是整數倍，比對無效）。
- 對應是否一對一（兩個色號對到同一個 RGB，代表遊戲把兩個色號設成同色，或對應有錯）。

得到的「色號 → RGB」就是 issue #4 修正色盤時的參照答案。
"""
import collections
import json
import struct
import sys
import zlib
from pathlib import Path

W, H = 320, 200


def load_frame(p):
    b = Path(p).read_bytes()
    if len(b) != W * H:
        raise SystemExit(f"{p}：{len(b)} bytes，不是 320×200 色號陣列")
    return b


def load_rgb2x(p):
    b = Path(p).read_bytes()
    if len(b) != 2 * W * 2 * H * 3:
        raise SystemExit(f"{p}：{len(b)} bytes，不是 640×400 RGB")
    px = []
    nonuniform = 0
    for y in range(H):
        for x in range(W):
            blk = set()
            for dy in (0, 1):
                for dx in (0, 1):
                    o = ((2 * y + dy) * 2 * W + (2 * x + dx)) * 3
                    blk.add(b[o:o + 3])
            if len(blk) != 1:
                nonuniform += 1
            o = (2 * y * 2 * W + 2 * x) * 3
            px.append(bytes(b[o:o + 3]))
    return px, nonuniform


def compare(golem, rgb):
    joint = collections.defaultdict(collections.Counter)
    for v, c in zip(golem, rgb):
        joint[v][c] += 1
    mapping = {v: cnt.most_common(1)[0][0] for v, cnt in joint.items()}
    bad = [i for i, (v, c) in enumerate(zip(golem, rgb)) if mapping[v] != c]
    inv = collections.defaultdict(list)
    for v, c in mapping.items():
        inv[c].append(v)
    merged = {c.hex(): vs for c, vs in inv.items() if len(vs) > 1}
    return mapping, bad, merged, joint


def bbox(idx):
    if not idx:
        return None
    xs = [i % W for i in idx]
    ys = [i // W for i in idx]
    return [min(xs), min(ys), max(xs), max(ys)]


def clusters(idx, gap=8):
    """把不一致像素粗分成幾塊（以列為單位合併相鄰列），回傳每塊的外框與像素數。"""
    rows = collections.defaultdict(list)
    for i in idx:
        rows[i // W].append(i)
    out = []
    cur = []
    last = None
    for y in sorted(rows):
        if last is not None and y - last > gap:
            out.append(cur)
            cur = []
        cur.extend(rows[y])
        last = y
    if cur:
        out.append(cur)
    return [{"bbox": bbox(c), "pixels": len(c)} for c in out]


def write_diff(path, golem, rgb, bad):
    badset = set(bad)
    out = bytearray()
    for i, c in enumerate(rgb):
        if i in badset:
            out += b"\xff\x00\xff"
        else:
            out += bytes(x // 3 for x in c)
    raw = b"".join(b"\0" + bytes(out[y * W * 3:(y + 1) * W * 3]) for y in range(H))

    def chunk(t, d):
        return struct.pack(">I", len(d)) + t + d + struct.pack(">I", zlib.crc32(t + d) & 0xFFFFFFFF)

    Path(path).write_bytes(b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", W, H, 8, 2, 0, 0, 0))
                           + chunk(b"IDAT", zlib.compress(raw)) + chunk(b"IEND", b""))


def main(argv):
    args = argv[1:]
    out_json = out_diff = None
    if "--json" in args:
        i = args.index("--json")
        out_json = args[i + 1]
        del args[i:i + 2]
    if "--diff" in args:
        i = args.index("--diff")
        out_diff = args[i + 1]
        del args[i:i + 2]
    if len(args) < 2:
        print(__doc__)
        return 2
    golem = load_frame(args[0])
    best = None
    tried = []
    for cand in args[1:]:
        rgb, nonuni = load_rgb2x(cand)
        mapping, bad, merged, joint = compare(golem, rgb)
        tried.append({"file": Path(cand).name, "mismatch": len(bad), "nonuniform_2x2": nonuni})
        if best is None or len(bad) < len(best[2]):
            best = (cand, rgb, bad, mapping, merged, nonuni)
    cand, rgb, bad, mapping, merged, nonuni = best
    result = {
        "golem_frame": Path(args[0]).name,
        "dosboxx_best": Path(cand).name,
        "candidates": tried,
        "nonuniform_2x2": nonuni,
        "mismatch_pixels": len(bad),
        "mismatch_regions": clusters(bad),
        "index_to_rgb": {f"{v:02X}": c.hex() for v, c in sorted(mapping.items())},
        "rgb_shared_by_indices": merged,
    }
    for t in tried:
        print(f'  候選 {t["file"]}：不一致 {t["mismatch"]}，2×2 不同色區塊 {t["nonuniform_2x2"]}')
    print(f'{Path(args[0]).name} vs {Path(cand).name}：不一致 {len(bad)} / {W * H} 像素')
    for r in result["mismatch_regions"]:
        print(f'  區塊 {r["bbox"]}：{r["pixels"]} 像素')
    print("  色號 → RGB：", result["index_to_rgb"])
    if merged:
        print("  ⚠ 多個色號對到同一個 RGB：", merged)
    if out_json:
        Path(out_json).write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    if out_diff:
        write_diff(out_diff, golem, rgb, bad)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
