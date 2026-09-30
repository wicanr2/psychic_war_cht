"""把 ENEMY*.PBL 的 sprite 分組，產生成組重繪的指示（docs/spec/024 §3.2）。

    tools/py.sh tools/hd/enemy_groups.py <輸出.md> [PBL 名…]

`ENEMY*` 不是一張圖一個主體：連續數張是**同一角色的不同動作幀**。
逐張送給畫圖模型，角色的造型會在幀與幀之間漂移，動畫起來會閃。

分組判準是相鄰張的不同像素比例：低於 35% 當成同一角色。
尺寸換掉也一定是分界（24×32 的角色 → 16×16 的小圖塊）。

輸出的指示把每一組列成一筆，要求「同一組用同一個造型、只有姿勢不同」。
"""
import json
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent.parent))
import pbl  # noqa: E402

SAME = 0.35  # 不同像素比例低於這個值 → 同一角色


def groups(path):
    data = path.read_bytes()
    imgs = []
    for i, o, _ in pbl.images(data):
        w, h, px = pbl.decode(data, o)
        imgs.append((i, w, h, bytes(px)))
    out, cur = [], [imgs[0]]
    for a, b in zip(imgs, imgs[1:]):
        same = (a[1], a[2]) == (b[1], b[2])
        if same:
            d = sum(1 for x, y in zip(a[3], b[3]) if x != y) / len(a[3])
            same = d < SAME
        if same:
            cur.append(b)
        else:
            out.append(cur); cur = [b]
    out.append(cur)
    return out


def main(argv):
    if len(argv) < 2:
        raise SystemExit(__doc__)
    out = argv[1]
    names = argv[2:] or ["ENEMY%02d" % i for i in range(12)]
    L = ["# ENEMY 系列的 HD 重繪（成組）", "",
         "**同一組是同一個角色的不同動作幀。**一組要用同一個造型重畫，只有姿勢不同；",
         "逐張各畫各的會讓角色在動畫中變形。", "",
         "| 檔 | 組 | 圖號 | 尺寸 | 參照圖 | 輸出 |", "|---|---|---|---|---|---|"]
    total = ngroup = 0
    for name in names:
        p = pathlib.Path("workplace/original/psychic-war/%s.PBL" % name)
        if not p.exists():
            continue
        for gi, g in enumerate(groups(p)):
            ids = [im[0] for im in g]
            w, h = g[0][1], g[0][2]
            refs = " ".join("`workplace/hd/ref/%s-%02d.png`" % (name, i) for i in ids)
            dsts = " ".join("`workplace/hd/art-in/%s-%02d.png`" % (name, i) for i in ids)
            L.append("| %s | %d | %s | **%d×%d** | %s | %s |"
                     % (name, gi, ",".join(str(i) for i in ids), w * 3, h * 3, refs, dsts))
            total += len(ids); ngroup += 1
    L += ["", "同一列的每個輸出都要**剛好**是該列標的尺寸，一個像素都不能差。", ""]
    pathlib.Path(out).write_text("\n".join(L) + "\n", encoding="utf-8")
    print("%s：%d 檔 %d 組 %d 張" % (out, len(names), ngroup, total))


if __name__ == "__main__":
    main(sys.argv)
