"""把分頭寫的圖檔疊字資料合併進 `text/baked.json`（issue #25）。

    tools/py.sh tools/baked_merge.py <來源.json…> [--text text] [--dry-run]

- 逐筆檢查再寫入：`tools/baked_lint.py` 的規則由合併後的整份檔案再跑一次（呼叫端負責）。
- key 重複時**後來的覆蓋先前的**，並列出來。
- 寫入前先排序（檔名、圖號、中文的 y、x），diff 才看得懂。
"""
import json
import pathlib
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent


def main(argv):
    srcs = [a for a in argv if not a.startswith("--")]
    text_dir = ROOT / (argv[argv.index("--text") + 1] if "--text" in argv else "text")
    if not srcs:
        print(__doc__)
        return 2
    dst = text_dir / "baked.json"
    doc = json.loads(dst.read_text(encoding="utf-8"))
    by_key = {e["key"]: e for e in doc["entries"]}
    added = replaced = 0
    for s in srcs:
        p = ROOT / s
        src = json.loads(p.read_text(encoding="utf-8"))
        if src.get("schema") != "psychic-war-baked/1":
            print("%s：schema 不是 psychic-war-baked/1，跳過" % s)
            continue
        for e in src["entries"]:
            if e["key"] in by_key:
                if by_key[e["key"]] != e:
                    print("覆蓋：%s（來自 %s）" % (e["key"], p.name))
                    replaced += 1
            else:
                added += 1
            by_key[e["key"]] = e
    entries = sorted(by_key.values(), key=lambda e: (e["file"], e["image"], e["text"][1], e["text"][0]))
    doc["entries"] = entries
    print("合併後 %d 筆（新增 %d、覆蓋 %d）" % (len(entries), added, replaced))
    if "--dry-run" in argv:
        return 0
    dst.write_text(json.dumps(doc, ensure_ascii=False, indent=1) + "\n", encoding="utf-8")
    print("寫出", dst)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
