"""把原版的硬斜角框線掃出來，改成平滑漸層的 HD 框線（docs/spec/024 §3.1）。

    tools/py.sh tools/hd/bevels.py <bg.idx> <寬> <高> <輸出.json> [最短長度]

原版的框線是「一列亮色壓一列暗色」（例如色 9 上面、色 1 下面）——
那是 16 色下表現立體感的手法。整數倍放大會把它變成一條 3 px 亮加 3 px 暗的硬邊。

這支工具掃出這種配對，輸出 `render.py` 吃的三段漸層基元：
上緣高光（亮色再提一階）→ 本體（原版的亮色）→ 下緣陰影（原版的暗色）。
**位置與厚度完全照原版**，所以不影響對位；改變的只有立體感的畫法。
"""
import json
import pathlib
import sys

# EGA 的亮／暗配對（高 3 bit 相同，bit 3 決定亮暗）
PAIRS = {(9, 1), (11, 3), (15, 7), (12, 4), (10, 2), (13, 5), (14, 6), (7, 8), (8, 0)}

# 每個亮色再往上提一階當高光，讓框線有「上緣受光」的立體感
HILIGHT = {9: 11, 11: 15, 15: 15, 12: 14, 10: 15, 13: 15, 14: 15, 7: 15, 8: 7}


def runs(idx, w, h, y, min_len):
    """回這一列所有長度 ≥ min_len 的同色區段。"""
    out, x = [], 0
    while x < w:
        c = idx[y * w + x]
        n = 1
        while x + n < w and idx[y * w + x + n] == c:
            n += 1
        if c and n >= min_len:
            out.append((x, n, c))
        x += n
    return out


def main(argv):
    if len(argv) < 5:
        raise SystemExit(__doc__)
    idx = pathlib.Path(argv[1]).read_bytes()
    w, h, out = int(argv[2]), int(argv[3]), argv[4]
    min_len = int(argv[5]) if len(argv) > 5 else 24

    shapes = []
    used = set()
    for y in range(h - 1):
        for x, n, c in runs(idx, w, h, y, min_len):
            # 下一列同一段是不是配對的暗色
            below = {idx[(y + 1) * w + x + i] for i in range(n)}
            if len(below) != 1:
                continue
            d = below.pop()
            if (c, d) not in PAIRS:
                continue
            if any((x + i, y) in used for i in range(n)):
                continue
            for i in range(n):
                used.add((x + i, y))
                used.add((x + i, y + 1))
            shapes.append({
                "kind": "rect",
                "at": [x, y],
                "size": [n, 2],
                "gradient": [HILIGHT[c], c, d],
                "dir": "v",
                "why": "原版 y=%d 的 %d→%d 硬斜角框線" % (y, c, d),
            })

    spec = {
        "size": [w, h],
        "note": ("自動掃出來的框線（tools/hd/bevels.py）。原版用「一列亮色壓一列暗色」"
                 "表現立體感，這裡改成同色系的平滑漸層。位置與厚度照原版，不影響對位。"),
        "shapes": shapes,
    }
    pathlib.Path(out).write_text(json.dumps(spec, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print("%s：掃出 %d 條框線（最短 %d px）" % (out, len(shapes), min_len))
    for s in shapes[:12]:
        print("   y=%3d x=%3d 長 %3d  色 %d→%d→%d" % (s["at"][1], s["at"][0], s["size"][0], *s["gradient"]))
    if len(shapes) > 12:
        print("   …其餘 %d 條" % (len(shapes) - 12))


if __name__ == "__main__":
    main(sys.argv)
