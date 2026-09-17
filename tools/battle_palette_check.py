#!/usr/bin/env python3
"""battle_palette_check.py — 攻擊雷射上的色號 2、A 顏色驗證（issue #4）。

    tools/py.sh tools/battle_palette_check.py <dosgolem 目錄> <DOSBox-X battle 目錄>

- dosgolem 目錄：probe `-dump-at` 在攻擊期間寫出的 `<步數>.png`（解色後）與 `<步數>.bin`（色號）。
- DOSBox-X 目錄：`tools/dosboxx-ref.sh` 錄的 `NNN.rgb`（640×400）。

兩邊亂數不同、錄影時間也不同，整張畫面對不齊。所以只看 dosgolem 用到色號 2、A 的像素：
對每一格 dosgolem 畫面，在 DOSBox-X 的每一格裡算這些像素的 RGB 有幾個相同，取最好的一對。

反向對照：同樣的像素改用 EGA 預設色（2 ＝ `00AA00`、A ＝ `55FF55`，也就是修正前 `-dump-ega` 的顏色）算一次。
修正有效的話，實際色盤的吻合率應該遠高於預設色。
"""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from frame_compare import H, W, load_png_rgb, load_rgb2x  # noqa: E402

DEFAULT = {2: bytes.fromhex("00aa00"), 10: bytes.fromhex("55ff55")}


def main(argv):
    if len(argv) != 3:
        print(__doc__)
        return 2
    gdir, ddir = Path(argv[1]), Path(argv[2])
    dosbox = [(p.name, load_rgb2x(p)[0]) for p in sorted(ddir.glob("*.rgb"))]
    if not dosbox:
        raise SystemExit(f"{ddir} 沒有 .rgb")
    rows = []
    for png in sorted(gdir.glob("*.png")):
        idx = png.with_suffix(".bin").read_bytes()
        if len(idx) != W * H:
            continue
        pts = [i for i, v in enumerate(idx) if v in (2, 10)]
        if len(pts) < 20:
            continue
        rgb = load_png_rgb(png)
        best = max(((sum(1 for i in pts if d[i] == rgb[i]), name, d) for name, d in dosbox), key=lambda t: t[0])
        hit, name, d = best
        default_hit = sum(1 for i in pts if d[i] == DEFAULT[idx[i]])
        colors = {v: rgb[next(i for i in pts if idx[i] == v)].hex() for v in (2, 10) if any(idx[i] == v for i in pts)}
        rows.append((png.stem, len(pts), hit, default_hit, name, colors))
    print("| dosgolem 步數 | 色號 2／A 像素 | 最佳 DOSBox-X 格 | 實際色盤吻合 | EGA 預設色吻合 | dosgolem 顏色 |")
    print("|---:|---:|---|---:|---:|---|")
    for step, n, hit, dhit, name, colors in rows:
        print(f"| {int(step):,} | {n} | {name} | {hit}（{hit / n:.0%}） | {dhit}（{dhit / n:.0%}） | "
              f"2＝`{colors.get(2, '—')}`、A＝`{colors.get(10, '—')}` |")
    if not rows:
        print("沒有含色號 2／A（≥ 20 像素）的 dosgolem 畫面")
        return 1
    total = sum(r[1] for r in rows)
    print(f"\n合計 {len(rows)} 格、{total} 像素：實際色盤吻合 {sum(r[2] for r in rows) / total:.1%}，"
          f"EGA 預設色吻合 {sum(r[3] for r in rows) / total:.1%}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
