"""驗證一批重繪圖的尺寸與存在（docs/spec/024 §6）。

    tools/py.sh tools/hd/verify_batch.py <PBL 名…>
    tools/py.sh tools/hd/verify_batch.py OPEN --skip OPEN:7,8,9,10

只有明確列出的圖號會排除，輸出仍列出排除清單。未指定排除時維持完整檢查。
本工具只檢查檔案存在與 PNG 標頭尺寸，不代表美術或遊戲驗收通過。
"""
import argparse
import json
import pathlib
import struct
import sys


def size(path):
    b = path.read_bytes()
    if len(b) < 24 or b[:8] != b"\x89PNG\r\n\x1a\n" or b[12:16] != b"IHDR":
        return None
    return struct.unpack_from(">II", b, 16)


def main(argv):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("names", nargs="*", help="PBL 名；省略時檢查全部")
    parser.add_argument("--skip", action="append", default=[], metavar="PBL:圖號,…",
                        help="明確排除的圖號，可重複指定；不會自動略過缺檔")
    args = parser.parse_args(argv[1:])
    batches = json.loads(pathlib.Path("workplace/hd/batches.json").read_text(encoding="utf-8"))
    known = {b["file"] for b in batches}
    names = args.names or sorted(known)
    for name in names:
        if name not in known:
            parser.error("batches.json 裡沒有 %s" % name)
    skips = {}
    for entry in args.skip:
        try:
            name, values = entry.split(":", 1)
            ids = {int(v) for v in values.split(",")}
        except ValueError:
            parser.error("排除項格式應為 PBL:圖號,…：%s" % entry)
        if name not in names:
            parser.error("排除項 %s 不在本次檢查批次中" % name)
        available = {b["image"] for b in batches if b["file"] == name}
        if not ids <= available:
            parser.error("%s 的排除清單含未知圖號：%s" %
                         (name, ",".join(str(i) for i in sorted(ids - available))))
        skips.setdefault(name, set()).update(ids)
    for name, ids in skips.items():
        if ids == {b["image"] for b in batches if b["file"] == name}:
            parser.error("不能排除 %s 的全部圖號；請只選擇要檢查的批次" % name)
    rc = 0
    for name in names:
        excluded = skips.get(name, set())
        items = [b for b in batches if b["file"] == name and b["image"] not in excluded]
        miss, wrong, ok = [], [], 0
        for b in items:
            p = pathlib.Path(b["dst"])
            if not p.exists():
                miss.append(b["image"]); continue
            try:
                s = size(p)
            except OSError:
                s = None
            if s != (b["out_w"], b["out_h"]):
                wrong.append((b["image"], s, (b["out_w"], b["out_h"]))); continue
            ok += 1
        print("%-7s %3d/%-3d 尺寸正確" % (name, ok, len(items)), end="")
        if excluded:
            print("　明確排除 %d 張：%s" %
                  (len(excluded), ",".join(str(i) for i in sorted(excluded))), end="")
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
