"""單字元輸出的呼叫端統計（第 5 輪 #19，docs/re/014 §6）。

    tools/py.sh tools/char_callers.py [紀錄目錄，預設 workplace/print/callers]

讀 states.sh 加 `-call-args <位址>:1:0:<大數> -arg-regs` 的紀錄：每一次進入 sub_16629 的步數、返回位址（近呼叫 ＝ 呼叫端 IP）、AL。
AL ≥ 20h 算一個畫出的字元；10h–1Fh 是顏色；< 10h 是控制碼。依呼叫端分組輸出。
"""
import collections
import json
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
REPLAY = ROOT / "replay/title-to-first-save.json"
ROW = re.compile(r"#(\d+)\s+由 \w{4}:(\w{4}) .*AX=(\w{4}) BX=(\w{4})")


def main(argv):
    d = pathlib.Path(argv[0]) if argv else ROOT / "workplace/print/callers"
    names = [s["name"] for s in json.load(open(REPLAY, encoding="utf-8"))["segments"]]
    total = collections.Counter()
    per_seg = {}
    ctrl = collections.Counter()
    values = collections.defaultdict(collections.Counter)
    for n in names:
        f = d / f"{n}.log"
        if not f.exists():
            print(f"{n}: 沒有紀錄")
            continue
        c = collections.Counter()
        for line in f.read_text(encoding="utf-8").splitlines():
            m = ROW.search(line)
            if not m:
                continue
            ret, al = m.group(2).upper(), int(m.group(3), 16) & 0xFF
            if al >= 0x20:
                c[ret] += 1
                total[ret] += 1
                values[ret][al] += 1
            elif al < 0x10:
                ctrl[al] += 1
        per_seg[n] = c
        print(f"{n}: {sum(c.values())} 字 " + "、".join(f"{k}×{v}" for k, v in c.most_common()))
    print(f"合計 {sum(total.values())} 字；呼叫端（返回 IP，cs＝0161）：")
    for k, v in total.most_common():
        top = values[k].most_common(6)
        print(f"  0161:{k}（IDA {0x10510 + int(k, 16):X}）×{v}，不同字碼 {len(values[k])}：" + "、".join(f"{a:02X}h×{n}" for a, n in top))
    print("控制碼：" + "、".join(f"{k:02X}h×{v}" for k, v in sorted(ctrl.items())))


if __name__ == "__main__":
    main(sys.argv[1:])
