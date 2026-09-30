"""HD 主題的合成驗收（docs/spec/024 §6）。

    tools/py.sh tools/hd/check.py <bg.idx> <寬> <高> <原版.png> <合成.png> <owned.txt>

**不變量：在所有「有主」的矩形之外，原版是黑的像素必須還是黑的。**
那些像素是遊戲要畫迷宮視野、訊息、狀態數值與角色的地方；被 HD 素材佔住的話，
訊息還沒出現時會看到裝飾，出現時又整塊消失。

「有主」指那一塊由某個 HD 素材完整負責（人物、重畫的箭頭、去抖色的儀表區）。
素材在自己的矩形裡可以蓋掉黑色——執行期的逐格失效會處理遊戲真的畫上去的情況。

⚠ 「不給畫圖代理碰」與「沒有人可以畫」是兩件事，不要共用一份清單：
箭頭要擋著不給代理畫，但我們自己會重畫它。`mask.py` 吃的是前者，這支吃的是後者。
"""
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent.parent))
import pbl  # noqa: E402


def main(argv):
    if len(argv) < 7:
        raise SystemExit(__doc__)
    idx = pathlib.Path(argv[1]).read_bytes()
    sw, sh = int(argv[2]), int(argv[3])
    w, h, ca, ra, _ = pbl.read_png(pathlib.Path(argv[4]))
    _, _, cb, rb, _ = pbl.read_png(pathlib.Path(argv[5]))
    owned = []
    for line in pathlib.Path(argv[6]).read_text(encoding="utf-8").splitlines():
        line = line.split("#")[0].strip()
        if not line:
            continue
        x, y, ww, hh = (int(v) for v in line.split(","))
        owned.append((x, y, x + ww, y + hh))
    scale = w // sw

    changed = bad = 0
    hits = {}
    for y in range(h):
        r1, r2 = ra[y], rb[y]
        sy = y // scale
        for x in range(w):
            if r1[ca*x:ca*x+3] == r2[cb*x:cb*x+3]:
                continue
            changed += 1
            sx = x // scale
            if any(x0 <= sx < x1 and y0 <= sy < y1 for x0, y0, x1, y1 in owned):
                continue
            if idx[sy * sw + sx] == 0:
                bad += 1
                hits[(sx // 8 * 8, sy // 8 * 8)] = hits.get((sx // 8 * 8, sy // 8 * 8), 0) + 1

    print("改動像素 %d／%d（%.1f%%）" % (changed, w * h, 100 * changed / (w * h)))
    print("有主矩形 %d 個" % len(owned))
    if bad:
        print("❌ 無主的黑色像素被佔住：%d px" % bad)
        for (x, y), n in sorted(hits.items(), key=lambda kv: -kv[1])[:12]:
            print("     原版 (%3d,%3d) 這一格 %d px" % (x, y, n))
        return 1
    print("✅ 無主的黑色像素被佔住：0 px")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
