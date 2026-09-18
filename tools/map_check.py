"""F3 自動地圖的畫面比對（docs/spec/015 §4 第 2、3 項）。

    tools/py.sh tools/map_check.py <地圖畫面.png> <quick.map.json> <觀測變數.mem>

期望值不看前端自己畫了什麼，是從**存檔讀出來的座標**與**存下來的地圖 JSON**推出來的：

- 目前格子（記憶體的 X、Y）在畫面上要是白色；
- `quick.map.json` 裡走過的格子要是灰色（目前格子除外）；
- 沒走過的格子要是黑色（反向對照）。

版面與 `cmd/psychicwar` 的 `drawMap` 相同：一格 12 像素、左上角 (24, 24)、每格畫 10×10。
"""
import json
import pathlib
import struct
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import pbl  # noqa: E402

CELL, OX, OY = 12, 24, 24
GREY, WHITE, BLACK = (0xAA, 0xAA, 0xAA), (0xFF, 0xFF, 0xFF), (0, 0, 0)


def main(argv):
    if len(argv) < 3:
        print(__doc__)
        return 2
    shot, mapjson, mem = argv[0], argv[1], argv[2]
    root = pathlib.Path(__file__).resolve().parent.parent
    w, h, ch, rows, plte = pbl.read_png(root / shot)
    b = (root / mem).read_bytes()
    area, x, y = struct.unpack_from("<HHH", b, 0)[0], struct.unpack_from("<H", b, 2)[0], struct.unpack_from("<H", b, 4)[0]
    doc = json.loads((root / mapjson).read_text(encoding="utf-8"))
    cells = [tuple(c) for c in doc.get("areas", {}).get(str(area), [])]

    def px(cx, cy):
        """格子中心的顏色。"""
        sx, sy = OX + cx * CELL + (CELL - 2) // 2, OY + cy * CELL + (CELL - 2) // 2
        if sx >= w or sy >= h:
            return None
        r = rows[sy]
        return plte[r[sx]] if ch == 1 else (r[ch * sx], r[ch * sx + 1], r[ch * sx + 2])

    bad = 0
    cur = px(x, y)
    print("區域 %d，目前格子 (%d, %d) 的顏色 %s（要 %s）" % (area, x, y, cur, WHITE))
    if cur != WHITE:
        bad += 1
    seen_ok = 0
    for cx, cy in cells:
        if (cx, cy) == (x, y):
            continue
        c = px(cx, cy)
        if c == GREY:
            seen_ok += 1
        else:
            bad += 1
            print("走過的格子 (%d, %d) 顏色 %s，要 %s" % (cx, cy, c, GREY))
    print("走過的格子 %d 個（不含目前格），顏色正確 %d 個" % (max(len(cells) - 1, 0), seen_ok))
    # 反向對照：挑幾個沒走過的格子
    unseen = [(cx, cy) for cy in range(8) for cx in range(8) if (cx, cy) not in cells][:5]
    for cx, cy in unseen:
        c = px(cx, cy)
        if c != BLACK:
            bad += 1
            print("沒走過的格子 (%d, %d) 顏色 %s，要 %s" % (cx, cy, c, BLACK))
    print("沒走過的格子抽驗 %d 個" % len(unseen))
    print("不符 %d 項" % bad)
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
