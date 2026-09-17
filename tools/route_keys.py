"""迷宮路線 → 按鍵序列（issue #8，給重播檔 replay/*.json 用）。

    tools/py.sh tools/route_keys.py <起始朝向 N|E|S|W> <路線>

路線是每一步要走的方向，例如 `SSEEN`。每一步先轉向（右轉 right、左轉 left、向後轉 down），再按 up 前進。
朝向與轉向依 docs/re/011：`0x16970` 0 北 1 東 2 南 3 西；right 使朝向 ＋1，left －1，down ＋2。
輸出逗號分隔的鍵名（probe -press 的名稱）與鍵數。
"""
import sys

TURN = {0: [], 1: ["right"], 2: ["down"], 3: ["left"]}


def keys_for(start, route):
    facing = "NESW".index(start)
    keys = []
    for d in route:
        target = "NESW".index(d)
        keys += TURN[(target - facing) % 4]
        keys.append("up")
        facing = target
    return keys


def main(argv):
    if len(argv) != 2 or argv[0] not in "NESW" or any(c not in "NESW" for c in argv[1]):
        print(__doc__)
        return 2
    keys = keys_for(argv[0], argv[1])
    print(",".join(keys))
    print(len(keys))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
