"""在既有畫面截圖裡找出哪一張有某筆圖檔疊字的原版圖塊（docs/spec/011 §5 第 2 項的前置）。

    tools/py.sh tools/baked_find.py <截圖…> [--text text] [--orig workplace/original/psychic-war] [--key <key>]

圖檔疊字大多在玩不到或還沒走到的房間，要做逐像素實跑驗收得先知道「哪個狀態的畫面上有這張圖」。
這支工具拿 `.PBL` 解出來的原版圖塊，去比每張截圖上 `screen` 座標的像素（960×600 的前端截圖會自動除以 3）。

判準與 `xlate` 的 watcher 相同：先抽 8 個點，全中才逐像素比。比的是**原版像素**，
所以截圖上如果已經蓋了中文就不會命中——先用還沒做疊字時留下的截圖找位置。

輸出每張截圖命中的 key，最後列出每個 key 第一次出現在哪一張。
"""
import json
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import pbl  # noqa: E402

ROOT = pathlib.Path(__file__).resolve().parent.parent


def probe_points(w, h):
    """圖塊上的 8 個取樣點（四角內縮 ＋ 邊中點）。"""
    x0, y0, x1, y1 = 1, 1, max(w - 2, 0), max(h - 2, 0)
    xm, ym = w // 2, h // 2
    return [(x0, y0), (x1, y0), (x0, y1), (x1, y1), (xm, y0), (xm, y1), (x0, ym), (x1, ym)]


def main(argv):
    shots = [a for a in argv if not a.startswith("--")]
    if not shots:
        print(__doc__)
        return 2
    text_dir = ROOT / (argv[argv.index("--text") + 1] if "--text" in argv else "text")
    orig = ROOT / (argv[argv.index("--orig") + 1] if "--orig" in argv else "workplace/original/psychic-war")
    only = argv[argv.index("--key") + 1] if "--key" in argv else None
    entries = json.loads((text_dir / "baked.json").read_text(encoding="utf-8"))["entries"]
    if only:
        entries = [e for e in entries if e["key"] == only]
    art, offs = {}, {}
    for e in entries:
        k = (e["file"], e["image"])
        if k in art:
            continue
        data = (orig / e["file"]).read_bytes()
        if e["file"] not in offs:
            offs[e["file"]] = {i: off for i, off, _ in pbl.images(data)}
        art[k] = pbl.decode(data, offs[e["file"]][e["image"]])
    first = {}
    for s in shots:
        p = pathlib.Path(s)
        if not p.is_absolute():
            p = ROOT / s
        w, h, ch, rows, plte = pbl.read_png(p)
        scale = w // 320
        if scale < 1 or w % 320:
            print("%s：寬 %d 不是 320 的整數倍，跳過" % (s, w))
            continue

        def rgb(x, y):
            r = rows[y * scale]
            x *= scale
            return plte[r[x]] if ch == 1 else (r[ch * x], r[ch * x + 1], r[ch * x + 2])

        hit = []
        for e in entries:
            iw, ih, ipx = art[(e["file"], e["image"])]
            sx, sy = e["screen"]
            if sx < 0 or sy < 0 or sx + iw > 320 or sy + ih > 200:
                continue
            ok = True
            for x, y in probe_points(iw, ih):
                if rgb(sx + x, sy + y) != pbl.EGA[ipx[y * iw + x] & 0xF]:
                    ok = False
                    break
            if not ok:
                continue
            for y in range(ih):
                for x in range(iw):
                    if rgb(sx + x, sy + y) != pbl.EGA[ipx[y * iw + x] & 0xF]:
                        ok = False
                        break
                if not ok:
                    break
            if ok:
                hit.append(e["key"])
                first.setdefault(e["key"], s)
        if hit:
            print("%s：%s" % (s, "、".join(hit)))
    print("---")
    for e in entries:
        print("%-28s %s" % (e["key"], first.get(e["key"], "沒有截圖有這張圖")))
    print("%d 筆裡有 %d 筆找得到畫面" % (len(entries), len(first)))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
