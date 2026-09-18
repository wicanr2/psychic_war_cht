"""檢查 `text/baked.json` 的每一筆（issue #25，docs/spec/011 §3）。

    tools/py.sh tools/baked_lint.py [--text text] [--orig workplace/original/psychic-war] [--file <另一份 json>]

逐筆檢查：

1. `file` 存在、`image` 在範圍內；`region`、`text` 都完全落在那張圖裡。
2. 中文蓋的矩形（`text` 的 x、y 加上格數×格寬、格高）要**包住** `region` 裡的英文墨跡
   （墨跡 ＝ 區塊裡出現次數最少的那些色號的像素；只檢查「英文有沒有被蓋到」，不看是什麼字）。
3. `translation` 非空、字數 ≤ 格數、每個字在字型子集裡（`font/charset.txt`）。
4. 同一張圖的不同筆不可以重疊（會互相移除）。
5. `original` 要在清冊 `text/baked-inventory.json` 的同一張圖裡找得到（大小寫、空白不計）。
6. 頂層 `equivalent` 宣告的「這張圖的文字像素與另一張相同」要真的成立：
   兩張圖各自換算到畫面座標之後，`covers` 列出的每一塊逐像素相同。
   面板同時來自 `SCREEN.PBL` 與 `MENU.PBL`，一組 watcher 就會蓋到兩者（`docs/re/024` §2），
   宣告出來報表才不會把另一張算成「還沒疊」。

全部通過結束碼 0；有問題列出來並回 1。
"""
import json
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import pbl  # noqa: E402

ROOT = pathlib.Path(__file__).resolve().parent.parent
CELL = {"cjk24": (8, 8), "cjk16": (6, 7)}


def ink(px, w, x, y, bw, bh):
    """回區塊裡「字」的像素位置：出現最少的色號當墨跡（招牌是大面積底色＋少量字）。"""
    cnt = {}
    for r in range(bh):
        for c in range(bw):
            v = px[(y + r) * w + x + c]
            cnt[v] = cnt.get(v, 0) + 1
    if len(cnt) < 2:
        return []
    least = sorted(cnt, key=lambda v: cnt[v])[0]
    return [(x + c, y + r) for r in range(bh) for c in range(bw) if px[(y + r) * w + x + c] == least]


def main(argv):
    text_dir = ROOT / (argv[argv.index("--text") + 1] if "--text" in argv else "text")
    orig = ROOT / (argv[argv.index("--orig") + 1] if "--orig" in argv else "workplace/original/psychic-war")
    src = ROOT / argv[argv.index("--file") + 1] if "--file" in argv else text_dir / "baked.json"
    doc = json.loads(src.read_text(encoding="utf-8"))
    inv = json.loads((text_dir / "baked-inventory.json").read_text(encoding="utf-8"))["entries"]
    charset = set((ROOT / "font/charset.txt").read_text(encoding="utf-8").strip())
    inv_by = {}
    for e in inv:
        inv_by.setdefault((e["file"], e["image"]), []).extend(t["text"].upper().replace(" ", "") for t in e["texts"])
    cache, rects, bad = {}, {}, 0

    def fail(key, msg):
        nonlocal bad
        bad += 1
        print("%-34s %s" % (key, msg))

    for e in doc["entries"]:
        key = e["key"]
        p = orig / e["file"]
        if not p.exists():
            print("%-34s 缺原版 %s（跳過）" % (key, e["file"]))
            continue
        if e["file"] not in cache:
            cache[e["file"]] = p.read_bytes()
        data = cache[e["file"]]
        offs = pbl.images(data)
        if not 0 <= e["image"] < len(offs):
            fail(key, "圖號 %d 超出 %d 張" % (e["image"], len(offs)))
            continue
        w, h, px = pbl.decode(data, offs[e["image"]][1])
        rx, ry, rw, rh = e["region"]
        if rx < 0 or ry < 0 or rx + rw > w or ry + rh > h or rw <= 0 or rh <= 0:
            fail(key, "region 超出圖 %d×%d" % (w, h))
            continue
        cw, chh = CELL.get(e.get("font") or "cjk24", (8, 8))
        tx, ty, cells = e["text"]
        tw, th = cells * cw, chh
        if tx < 0 or ty < 0 or tx + tw > w or ty + th > h or cells <= 0:
            fail(key, "中文矩形 (%d,%d,%d,%d) 超出圖 %d×%d" % (tx, ty, tw, th, w, h))
            continue
        # 2：英文墨跡要被蓋住
        outside = [(x, y) for x, y in ink(px, w, rx, ry, rw, rh)
                   if not (tx <= x < tx + tw and ty <= y < ty + th)]
        if len(outside) > 2:  # 容忍兩點（招牌邊角的雜點）
            fail(key, "中文沒蓋住英文：%d 個墨跡點在外面，例如 %s" % (len(outside), outside[:4]))
        # 3：譯文
        tr = e["translation"]
        if not tr:
            fail(key, "沒有譯文")
        elif len(tr) > cells:
            fail(key, "譯文 %d 字，超過 %d 格" % (len(tr), cells))
        else:
            missing = [c for c in tr if c != " " and c not in charset]
            if missing:
                fail(key, "字型子集沒有：%s（先跑 tools/font/bake.sh）" % "".join(missing))
        # 4：同一張圖不可重疊
        r = (tx, ty, tx + tw, ty + th)
        for other, o in rects.get((e["file"], e["image"]), []):
            if r[0] < o[2] and o[0] < r[2] and r[1] < o[3] and o[1] < r[3]:
                fail(key, "與 %s 的中文矩形重疊" % other)
        rects.setdefault((e["file"], e["image"]), []).append((key, r))
        # 5：原文要在清冊裡
        want = e["original"].upper().replace(" ", "")
        texts = inv_by.get((e["file"], e["image"]), [])
        if want and texts and not any(want in t or t in want for t in texts):
            fail(key, "原文 %r 不在清冊的 %s" % (e["original"], texts))

    # 6：等價涵蓋的宣告要成立（同一塊像素來自兩張圖時，一組 watcher 就夠）
    by_key = {e["key"]: e for e in doc["entries"]}
    for q in doc.get("equivalent", []):
        tag = "%s:%d" % (q["file"], q["image"])
        p = orig / q["file"]
        if not p.exists():
            print("%-34s 缺原版 %s（跳過）" % (tag, q["file"]))
            continue
        if q["file"] not in cache:
            cache[q["file"]] = p.read_bytes()
        qw, qh, qpx = pbl.decode(cache[q["file"]], pbl.images(cache[q["file"]])[q["image"]][1])
        qsx, qsy = q["screen"]
        for k in q["covers"]:
            e = by_key.get(k)
            if e is None:
                fail(tag, "covers 指到不存在的 %s" % k)
                continue
            w, h, px = pbl.decode(cache[e["file"]], pbl.images(cache[e["file"]])[e["image"]][1])
            rx, ry, rw, rh = e["region"]
            # 兩張圖各自的圖內座標換算到畫面座標再比
            ax, ay = e["screen"][0] + rx, e["screen"][1] + ry
            bx, by = ax - qsx, ay - qsy
            if bx < 0 or by < 0 or bx + rw > qw or by + rh > qh:
                fail(tag, "%s 的區塊落在這張圖之外（圖內 (%d,%d) %d×%d，圖 %d×%d）" % (k, bx, by, rw, rh, qw, qh))
                continue
            diff = sum(1 for r in range(rh) for c in range(rw)
                       if px[(ry + r) * w + rx + c] != qpx[(by + r) * qw + bx + c])
            if diff:
                fail(tag, "%s 的區塊與這張圖不同：%d 個像素" % (k, diff))
    print("%d 筆，等價涵蓋 %d 張，問題 %d 筆" % (len(doc["entries"]), len(doc.get("equivalent", [])), bad))
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
