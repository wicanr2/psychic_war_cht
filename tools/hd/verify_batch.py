"""驗證一批重繪圖的尺寸與存在（docs/spec/024 §6）。

    tools/py.sh tools/hd/verify_batch.py <PBL 名…>
"""
import json
import pathlib
import struct
import sys


def size(path):
    b = path.read_bytes()
    if b[:8] != b"\x89PNG\r\n\x1a\n":
        return None
    return struct.unpack_from(">II", b, 16)


def main(argv):
    batches = json.loads(pathlib.Path("workplace/hd/batches.json").read_text(encoding="utf-8"))
    names = argv[1:] or sorted({b["file"] for b in batches})
    rc = 0
    for name in names:
        items = [b for b in batches if b["file"] == name]
        miss, wrong, ok = [], [], 0
        for b in items:
            p = pathlib.Path(b["dst"])
            if not p.exists():
                miss.append(b["image"]); continue
            s = size(p)
            if s != (b["out_w"], b["out_h"]):
                wrong.append((b["image"], s, (b["out_w"], b["out_h"]))); continue
            ok += 1
        print("%-7s %3d/%-3d 尺寸正確" % (name, ok, len(items)), end="")
        if miss:
            print("　缺 %d 張：%s" % (len(miss), ",".join(str(i) for i in miss[:12])), end="")
        if wrong:
            print("　尺寸錯 %d 張：%s" % (len(wrong), "; ".join("#%d 是 %s 應該 %s" % w for w in wrong[:5])), end="")
        print()
        if miss or wrong:
            rc = 1
    return rc


if __name__ == "__main__":
    sys.exit(main(sys.argv))
