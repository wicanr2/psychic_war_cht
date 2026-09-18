"""量圖檔裡文字的位置（issue #25，給 text/baked.json 用）。

    tools/py.sh tools/baked_boxes.py <PBL 檔> <圖號…>            # 列出每張圖裡的文字候選框
    tools/py.sh tools/baked_boxes.py <PBL 檔> <圖號> --art <x> <y> <w> <h>   # 印出那一塊的色號圖
    tools/py.sh tools/baked_boxes.py <PBL 檔> <圖號…> --colors 1,9           # 換一組「字的顏色」再找

候選框的找法：把「亮色」（色號 7、11、13、14、15）的像素以 2 像素的間距連通起來，
框寬 ≥ 12、高 3–12 的就是候選（招牌上的字），輸出框、框內色號分布與建議的格子數。

⚠ 這只是量尺，不是判讀：框裡是什麼字看 `text/baked-inventory.json` 或直接看 PNG。
"""
import collections
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import pbl  # noqa: E402

BRIGHT = (7, 11, 13, 14, 15)


def boxes(px, w, h, colors=BRIGHT, gap=2, minw=12, minh=3, maxh=12):
    mask = [[1 if px[y * w + x] in colors else 0 for x in range(w)] for y in range(h)]
    seen = [[0] * w for _ in range(h)]
    out = []
    for y in range(h):
        for x in range(w):
            if not mask[y][x] or seen[y][x]:
                continue
            st, xs, ys = [(x, y)], [x], [y]
            seen[y][x] = 1
            while st:
                cx, cy = st.pop()
                for dy in range(-gap, gap + 1):
                    for dx in range(-gap, gap + 1):
                        nx, ny = cx + dx, cy + dy
                        if 0 <= nx < w and 0 <= ny < h and mask[ny][nx] and not seen[ny][nx]:
                            seen[ny][nx] = 1
                            st.append((nx, ny))
                            xs.append(nx)
                            ys.append(ny)
            bw, bh = max(xs) - min(xs) + 1, max(ys) - min(ys) + 1
            if bw >= minw and minh <= bh <= maxh:
                out.append((min(xs), min(ys), bw, bh))
    return sorted(out, key=lambda b: (b[1], b[0]))


def art(px, w, h, x0, y0, bw, bh):
    rows = []
    for y in range(y0, min(y0 + bh, h)):
        rows.append("".join("%X" % px[y * w + x] for x in range(x0, min(x0 + bw, w))))
    return "\n".join(rows)


def main(argv):
    if len(argv) < 2:
        print(__doc__)
        return 2
    path = pathlib.Path(argv[0])
    data = path.read_bytes()
    offs = pbl.images(data)
    nums = []
    for a in argv[1:]:
        if a.startswith("--"):
            break
        nums.append(int(a))
    colors = BRIGHT
    if "--colors" in argv:
        colors = tuple(int(v) for v in argv[argv.index("--colors") + 1].split(","))
    for n in nums:
        w, h, px = pbl.decode(data, offs[n][1])
        print("== %s #%d %d×%d" % (path.name, n, w, h))
        if "--art" in argv:
            i = argv.index("--art")
            x0, y0, bw, bh = (int(v) for v in argv[i + 1:i + 5])
            print(art(px, w, h, x0, y0, bw, bh))
            continue
        for x, y, bw, bh in boxes(px, w, h, colors):
            reg = [px[(y + r) * w + x + c] for r in range(bh) for c in range(bw)]
            cnt = collections.Counter(reg).most_common(3)
            print("   框 x=%2d y=%2d w=%2d h=%2d  色號 %s  放得下 %d 個中文字（8 像素一格）"
                  % (x, y, bw, bh, cnt, bw // 8))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
