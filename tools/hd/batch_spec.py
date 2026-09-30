"""產生一批圖檔的重繪指示（docs/spec/024 §3.2）。

    tools/py.sh tools/hd/batch_spec.py <PBL 名> <輸出.md> [跳過的圖號,…]

每張圖列出：參照圖、輸出路徑、精確尺寸、以及**不能畫東西的文字框**
（`text/baked.json` 的內嵌英文，執行時會被中文疊字蓋掉；重繪要留成乾淨的招牌底，
畫了英文就會透出來）。

⚠ 有些圖**整張就是一條文字**（`OPEN` #7–#10 是開場字幕條，72×8），
那種要用「跳過的圖號」排除，不能重繪——整條已經疊了中文，重畫沒有意義而且會蓋掉它。
"""
import json
import pathlib
import sys


def main(argv):
    if len(argv) < 3:
        raise SystemExit(__doc__)
    name, out = argv[1], argv[2]
    skip = {int(v) for v in argv[3].split(",")} if len(argv) > 3 and argv[3] else set()
    batches = json.loads(pathlib.Path("workplace/hd/batches.json").read_text(encoding="utf-8"))
    baked = json.loads(pathlib.Path("text/baked.json").read_text(encoding="utf-8"))
    texts = {}
    for e in baked["entries"]:
        if e["file"] == name + ".PBL":
            texts.setdefault(e["image"], []).append((e["region"], e["original"]))

    items = [b for b in batches if b["file"] == name and b["image"] not in skip]
    if not items:
        raise SystemExit("batches.json 裡沒有 %s" % name)

    L = ["# %s 的 HD 重繪（共 %d 張）" % (name, len(items)), ""]
    L.append("每張都是獨立的一張圖。**輸出尺寸不能差一個像素。**")
    L.append("")
    L.append("| # | 參照圖 | 輸出 | 尺寸 | 不能畫東西的文字框（圖內座標 x,y,w,h ×3 是輸出座標） |")
    L.append("|---|---|---|---|---|")
    for b in items:
        t = texts.get(b["image"], [])
        cell = "，".join("%s = `%s`" % (",".join(str(v) for v in r), o) for r, o in t) or "—"
        L.append("| %d | `%s` | `%s` | **%d×%d** | %s |"
                 % (b["image"], b["ref"], b["dst"], b["out_w"], b["out_h"], cell))
    L += ["", "文字框那幾塊要畫成**乾淨的招牌底**（跟周圍一致的平面），不要畫任何字。",
          "遊戲執行時中文會蓋在那些位置上，你畫了英文就會從中文底下透出來。", ""]
    pathlib.Path(out).write_text("\n".join(L) + "\n", encoding="utf-8")
    print("%s：%d 張，其中 %d 張有內嵌文字%s"
          % (out, len(items), sum(1 for b in items if b["image"] in texts),
             ("；跳過 %s" % ",".join(str(i) for i in sorted(skip))) if skip else ""))


if __name__ == "__main__":
    main(sys.argv)
