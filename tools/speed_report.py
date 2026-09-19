#!/usr/bin/env python3
"""speed_report.py — 速度檔位驗收的數字（docs/spec/023 §8）。

    tools/py.sh tools/speed_report.py idle   <stats.jsonl…>
    tools/py.sh tools/speed_report.py walk   <stats.jsonl…>
    tools/py.sh tools/speed_report.py battle <stats.jsonl…>

讀前端 `-stats` 寫的 JSON Lines。事件欄 `event`：
tick（每秒）、cell（換格子）、battle-in／battle-out（進出戰鬥）、gear（換檔）、quit。

walk：從**走到的第一格**到最後一格的牆上秒數與每格平均，外加同一段的
`機器毫秒 ÷ 牆上毫秒`（＝實際生效的倍率）。第一筆 cell 是載入狀態檔時的所在格、
牆上時間 0，不是走出來的，要扣掉——不扣的話量到的是「等前端起來的時間」。
battle：戰鬥結束事件的指令數與牆上毫秒，以及戰鬥區間的倍率——
**戰鬥區間的倍率一定要是 1.00，不管玩家選了哪一檔**，那就是這次改動的核心宣稱。
"""
import json
import sys


def rows(path):
    out = []
    with open(path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line:
                out.append(json.loads(line))
    return out


def ratio(a, b):
    dw = b["wall_ms"] - a["wall_ms"]
    return (b["machine_ms"] - a["machine_ms"]) / dw if dw else float("nan")


def walk(paths):
    print(f"{'檔案':<28} {'檔位':>4} {'格數':>4} {'秒':>7} {'每格秒':>7} {'機器÷牆上':>9}")
    for p in paths:
        r = rows(p)
        cells = [x for x in r if x.get("event") == "cell"][1:]  # 第一筆是起點，不是走出來的
        gear = r[-1]["gear"] if r else 0
        if len(cells) < 2:
            print(f"{p.split('/')[-1]:<28} {gear:>4} {len(cells):>4}  （格數不足，量不出來）")
            continue
        a, b = cells[0], cells[-1]
        sec = (b["wall_ms"] - a["wall_ms"]) / 1000
        n = len(cells) - 1
        print(f"{p.split('/')[-1]:<28} {gear:>4} {n:>4} {sec:>7.2f} {sec / n:>7.3f} {ratio(a, b):>9.2f}")


def battle(paths):
    print(f"{'檔案':<28} {'檔位':>4} {'結束指令數':>12} {'結束牆上毫秒':>12} {'戰鬥區間機器÷牆上':>17}")
    for p in paths:
        r = rows(p)
        gear = r[-1]["gear"] if r else 0
        out = next((x for x in r if x.get("event") == "battle-out"), None)
        if out is None:
            print(f"{p.split('/')[-1]:<28} {gear:>4}  （沒有 battle-out，戰鬥沒打完）")
            continue
        first = r[0]
        print(f"{p.split('/')[-1]:<28} {gear:>4} {out['steps']:>12} {out['wall_ms']:>12} {ratio(first, out):>17.3f}")


def idle(paths):
    """不按任何鍵跑固定秒數：跑到的指令數。

    沒有輸入，所以指令數只由「牆上時間換多少 cycles」決定。迷宮裡要差檔位倍數，
    戰鬥中兩個檔位要一樣——那就是「戰鬥不受檔位影響」最直接的樣子。
    """
    print(f"{'檔案':<28} {'檔位':>4} {'牆上秒':>7} {'機器秒':>7} {'跑到的指令數':>14} {'機器÷牆上':>9}")
    for p in paths:
        r = rows(p)
        q = next((x for x in reversed(r) if x.get("event") == "quit"), r[-1] if r else None)
        if q is None:
            print(f"{p.split('/')[-1]:<28}  （沒有量測）")
            continue
        print(f"{p.split('/')[-1]:<28} {q['gear']:>4} {q['wall_ms'] / 1000:>7.2f} "
              f"{q['machine_ms'] / 1000:>7.2f} {q['steps']:>14} {q['machine_ms'] / q['wall_ms']:>9.2f}")


MODES = {"walk": walk, "battle": battle, "idle": idle}

if __name__ == "__main__":
    if len(sys.argv) < 3 or sys.argv[1] not in MODES:
        print(__doc__)
        sys.exit(2)
    MODES[sys.argv[1]](sys.argv[2:])
