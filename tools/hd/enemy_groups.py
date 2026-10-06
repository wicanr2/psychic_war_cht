"""把 ENEMY*.PBL 的 sprite 分組，產生成組重繪的指示（docs/spec/024 §3.2）。

    tools/py.sh tools/hd/enemy_groups.py <輸出.md> [PBL 名…]

`ENEMY*` 中有同一角色的動作幀；連續圖號與像素相似度只能提供候選分組。
確認同一角色後應成組重畫，避免造型在幀與幀之間漂移。

候選分組判準是相鄰張的不同像素比例：低於 35% 暫列同組。
尺寸換掉也一定是分界（24×32 的角色 → 16×16 的小圖塊）。

每組的角色／動作語意都是假說，繪圖前須檢視參照；不是原版語意已證實的清冊。
"""
import hashlib
import json
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent.parent))
import pbl  # noqa: E402

SAME = 0.35  # 視覺候選門檻；不證明角色或動作身分


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
    L = ["# ENEMY 系列的 HD 重繪（候選分組）", "",
         "**⚠ 假說：像素相似度不證明同一角色。**先檢視參照圖，再確認哪些是同角色動作；",
         "確認後同組用同一個造型重畫。尺寸與圖號來自原版資料，角色／動作語意仍未證實。", "",
         "工具：tools/hd/enemy_groups.py／Python %s；來源基準：PBL 檔名、圖號、解碼色號陣列。" % sys.version.split()[0],
         "門檻：相鄰圖尺寸相同且不同像素比例 < %.2f；只作視覺候選。" % SAME, "",
         "| 檔 | 候選組 | 原始圖號 | 尺寸 | 等級 | 參照圖 | 輸出 |", "|---|---|---|---|---|---|---|"]
    sources = []
    total = ngroup = 0
    for name in names:
        p = pathlib.Path("workplace/original/psychic-war/%s.PBL" % name)
        if not p.exists():
            raise SystemExit("找不到原始輸入：%s" % p)
        sources.append("- `%s`：SHA-256 `%s`" % (p, hashlib.sha256(p.read_bytes()).hexdigest()))
        for gi, g in enumerate(groups(p)):
            ids = [im[0] for im in g]
            w, h = g[0][1], g[0][2]
            refs = " ".join("`workplace/hd/ref/%s-%02d.png`" % (name, i) for i in ids)
            dsts = " ".join("`workplace/hd/art-in/%s-%02d.png`" % (name, i) for i in ids)
            L.append("| %s | %d | %s | **%d×%d** | **⚠ 假說** | %s | %s |"
                     % (name, gi, ",".join(str(i) for i in ids), w * 3, h * 3, refs, dsts))
            total += len(ids); ngroup += 1
    L += ["", "同一列的每個輸出都要**剛好**是該列標的尺寸，一個像素都不能差。", "",
          "## 原始輸入", ""] + sources + [""]
    pathlib.Path(out).write_text("\n".join(L) + "\n", encoding="utf-8")
    print("%s：%d 檔 %d 組 %d 張" % (out, len(names), ngroup, total))


if __name__ == "__main__":
    main(sys.argv)
