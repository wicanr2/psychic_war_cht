#!/usr/bin/env python3
"""census_report.py — 把 IDA 普查 JSON 與 dosgolem 覆蓋率合成統計（issue #1）。

    tools/py.sh tools/census_report.py <census.json> <cov.json> <runtime_base hex> <image_len> [--int-table]

- IDA ea ＝ image_base ＋（執行期線性 − runtime_base），只算落在 [runtime_base, runtime_base＋image_len) 的覆蓋率。
- dosgolem 的覆蓋率記的是**指令起點**（`internal/machine` 在每道指令前標 CS:IP），
  所以「執行過的位址數」是指令數，不是位元組數。
"""
import collections
import json
import sys


def main(argv):
    if len(argv) < 5:
        print(__doc__)
        return 2
    census = json.load(open(argv[1]))
    cov = json.load(open(argv[2]))
    rbase = int(argv[3], 16)
    img_len = int(argv[4], 0)
    snap = census.get("after_seed") or census["before_seed"]
    base = census["image_base"]

    executed = set()
    outside = 0
    for sp in cov["spans"]:
        for lin in range(int(sp["start"], 16), int(sp["end"], 16)):
            if rbase <= lin < rbase + img_len:
                executed.add(base + lin - rbase)
            else:
                outside += 1

    funcs = sorted(snap["functions"], key=lambda f: f["start"])
    starts = [f["start"] for f in funcs]

    def owner(ea):
        lo, hi = 0, len(funcs)
        while lo < hi:
            mid = (lo + hi) // 2
            if funcs[mid]["start"] <= ea:
                lo = mid + 1
            else:
                hi = mid
        i = lo - 1
        if i >= 0 and funcs[i]["start"] <= ea < funcs[i]["end"]:
            return funcs[i]
        return None

    hit_funcs = set()
    orphan = 0
    for ea in executed:
        f = owner(ea)
        if f:
            hit_funcs.add(f["start"])
        else:
            orphan += 1

    print(f'輸入 SHA-256：{census["input_sha256"]}')
    print(f'函式：{len(funcs)}（IDA 標 FUNC_LIB：{sum(f["lib"] for f in funcs)}）')
    print(f'程式碼 {snap["counts"]["code_bytes"]} bytes、資料 {snap["counts"]["data_bytes"]} bytes、'
          f'未定義 {snap["counts"]["unknown_bytes"]} bytes；字串 {snap["strings"]}')
    print(f'覆蓋率：映像內執行過的指令起點 {len(executed)}，映像外 {outside}；'
          f'落在函式外的指令起點 {orphan}')
    print(f'執行過至少一道指令的函式：{len(hit_funcs)} / {len(funcs)}')
    if "seed" in census:
        print(f'覆蓋率種子：新建指令 {census["seed"]["planted_insns"]}；'
              f'函式 {len(census["before_seed"]["functions"])} → {len(census["after_seed"]["functions"])}')

    heads = collections.Counter(f["head"][:6] for f in funcs)
    print("函式開頭前 3 bytes 最常見：", heads.most_common(8))
    frames = sum(1 for f in funcs if f["head"].startswith("558bec"))
    print(f"以 push bp; mov bp,sp 開頭的函式：{frames}")
    size = collections.Counter()
    for f in funcs:
        size["<16" if f["size"] < 16 else "<64" if f["size"] < 64 else "<256" if f["size"] < 256 else ">=256"] += 1
    print("函式大小分布：", dict(size))

    sites = snap["int_sites"]
    by_op = collections.Counter("%02Xh" % s["operand"] for s in sites)
    ex_op = collections.Counter("%02Xh" % s["operand"] for s in sites if s["ea"] in executed)
    print("INT 呼叫點（全部／這次執行過）：",
          {k: f"{v}/{ex_op.get(k, 0)}" for k, v in sorted(by_op.items())})

    if "--int-table" in argv:
        print()
        print("| IDA 位址 | 段:位移 | 中斷 | bytes | 函式 | 這次執行過 | 反組譯 |")
        print("|---|---|---|---|---|---|---|")
        for s in sorted(sites, key=lambda s: s["ea"]):
            print(f'| `{s["ea"]:05X}` | `{s["segoff"]}` | `{s["operand"]:02X}h` | `{s["bytes"]}` | '
                  f'{s["func"] or "—"} | {"是" if s["ea"] in executed else "否"} | `{" ".join(s["disasm"].split())}` |')
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
