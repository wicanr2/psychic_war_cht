"""圖檔內嵌文字的覆蓋率（issue #18、#25）：清冊有幾則、疊了幾則、還差哪些。

    tools/py.sh tools/baked_report.py [--text text] [--json out.json]

清冊 `text/baked-inventory.json`（537 張圖，has_text 的列出圖上的英文），
已疊中文的在 `text/baked.json`（一筆 ＝ 一塊要蓋的區域）。
兩者以「檔案＋圖號」對應：同一張圖上的幾則文字可能合成一塊，也可能拆成好幾塊，
所以覆蓋率以**張數**為主、則數為輔。
"""
import collections
import json
import pathlib
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent


def main(argv):
    text_dir = ROOT / (argv[argv.index("--text") + 1] if "--text" in argv else "text")
    inv = json.loads((text_dir / "baked-inventory.json").read_text(encoding="utf-8"))["entries"]
    baked = json.loads((text_dir / "baked.json").read_text(encoding="utf-8"))["entries"]
    with_text = [e for e in inv if e["has_text"]]
    strings = sum(len(e["texts"]) for e in with_text)
    done_imgs = {(e["file"], e["image"]) for e in baked}
    translated = sum(1 for e in baked if e["translation"])
    by_file = collections.Counter(e["file"] for e in with_text)
    done_by_file = collections.Counter(f for f, _ in done_imgs)
    rows = []
    for f in sorted(by_file):
        rows.append({"file": f, "images_with_text": by_file[f], "images_done": done_by_file.get(f, 0)})
    out = {
        "images_total": len(inv),
        "images_with_text": len(with_text),
        "strings_in_inventory": strings,
        "images_covered": len(done_imgs),
        "regions": len(baked),
        "regions_translated": translated,
        "by_file": rows,
        "remaining": [{"file": e["file"], "image": e["image"], "kind": e["kind"],
                       "texts": [t["text"] for t in e["texts"]]}
                      for e in with_text if (e["file"], e["image"]) not in done_imgs],
    }
    print("圖 %d 張，含文字 %d 張（%d 則）；已疊中文 %d 張、%d 塊（有譯文 %d 塊）"
          % (out["images_total"], out["images_with_text"], out["strings_in_inventory"],
             out["images_covered"], out["regions"], out["regions_translated"]))
    for r in rows:
        print("  %-14s 含文字 %2d 張，已疊 %d 張" % (r["file"], r["images_with_text"], r["images_done"]))
    if "--json" in argv:
        p = ROOT / argv[argv.index("--json") + 1]
        p.write_text(json.dumps(out, ensure_ascii=False, indent=1) + "\n", encoding="utf-8")
        print("寫出", p)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
