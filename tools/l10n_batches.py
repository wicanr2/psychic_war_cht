"""全文翻譯的分批與合併（issue #24，knowledge-base workflows/batch-subagent-localization.md）。

    tools/py.sh tools/l10n_batches.py prep <原版目錄> [--size 130]   # text/ → workplace/l10n/batches/NN.json
    tools/py.sh tools/l10n_batches.py merge                          # workplace/l10n/batches/NN.done.json → text/
    tools/py.sh tools/l10n_batches.py sweep <glossary.json>          # 合併後掃描：譯名表的英文名在譯文裡殘留、同一英文名多種譯法

prep：只收「要翻」的則（translatable、reachable、same_as 為空、還沒有譯文）。同一個來源檔的則放在一起、依偏移排序，
選項帶上所屬提問的原文當上下文。原文裡 80h 以上的自訂字模另附 ASCII 字形圖（FONT.BIN 8×8）。
merge：逐則核對 key 存在、原文沒被改、譯文通過 docs/spec/009 §3 的行寬與 Big5 檢查；不通過的整則不寫入並列出。
"""
import json
import pathlib
import re
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import text_extract as te  # noqa: E402

ROOT = pathlib.Path(__file__).resolve().parent.parent
BATCH = ROOT / "workplace/l10n/batches"


def glyph_art(raw, font8):
    rows = []
    for r in range(8):
        line = ""
        for c in raw:
            if c < 0x20:
                continue
            g = font8[(c - 0x20) * 8 + r]
            line += "".join("#" if g & (0x80 >> b) else "." for b in range(8))
        rows.append(line)
    return rows


def cmd_prep(orig, size):
    font8 = (pathlib.Path(orig) / "FONT.BIN").read_bytes()
    have = te.load_text(ROOT / "text")
    todo = [e for e in te.to_translate(have) if not e["translation"]]
    order = {}
    for e in todo:
        src, off = e["key"].rsplit(":", 1)
        order[e["key"]] = (src, int(off, 16))
    todo.sort(key=lambda e: order[e["key"]])
    BATCH.mkdir(parents=True, exist_ok=True)
    for old in BATCH.glob("*.json"):
        if not old.name.endswith(".done.json"):
            old.unlink()
    batches, cur = [], []
    for e in todo:
        # 選項不要和它的提問分到不同批
        if len(cur) >= size and not (e["kind"] == "option" and cur and cur[-1]["kind"] in ("menu", "option")):
            batches.append(cur)
            cur = []
        cur.append(e)
    if cur:
        batches.append(cur)
    for i, b in enumerate(batches, 1):
        items = []
        for e in b:
            raw = te.decode_shown(e["original"])
            it = {"key": e["key"], "kind": e["kind"], "font": e["font"], "lines": te.line_widths(e), "original": e["original"], "translation": ""}
            if e["kind"] == "option" and e.get("menu") in have:
                it["menu"] = have[e["menu"]]["original"]
            if any(c >= 0x80 for c in raw):
                it["glyph_art"] = glyph_art(raw, font8)
            items.append(it)
        (BATCH / ("%02d.json" % i)).write_text(json.dumps({"batch": i, "entries": items}, ensure_ascii=False, indent=1) + "\n", encoding="utf-8")
    print("要翻且沒有譯文 %d 則，分 %d 批（每批約 %d 則）→ %s" % (len(todo), len(batches), size, BATCH))


def cmd_merge():
    have = te.load_text(ROOT / "text")
    updates, problems = {}, []
    for p in sorted(BATCH.glob("*.done.json")):
        doc = json.loads(p.read_text(encoding="utf-8"))
        for it in doc["entries"]:
            k = it.get("key")
            e = have.get(k)
            tr = it.get("translation", "")
            if e is None:
                problems.append("%s：%s 不在文本檔" % (p.name, k))
                continue
            if it.get("original") != e["original"]:
                problems.append("%s：%s 原文被改動" % (p.name, k))
                continue
            if not tr:
                problems.append("%s：%s 沒有譯文" % (p.name, k))
                continue
            _, too = te.layout(tr, te.line_widths(e))
            if too and tr != e["original"]:
                problems.append("%s：%s 譯文過長（行寬 %s）：%s" % (p.name, k, te.line_widths(e), tr.replace("\n", "⏎")))
                continue
            bc = te.bad_chars(tr) if tr != e["original"] else []
            if bc:
                problems.append("%s：%s 非 Big5 字元 %s" % (p.name, k, "".join(bc)))
                continue
            if k in updates and updates[k] != tr:
                problems.append("%s：%s 在兩批有不同譯文" % (p.name, k))
            updates[k] = tr
    by_file = {}
    for k in updates:
        by_file.setdefault(k.split(":")[0], []).append(k)
    for src, keys in by_file.items():
        path = ROOT / "text" / (src + ".json")
        doc = json.loads(path.read_text(encoding="utf-8"))
        for e in doc["entries"]:
            if e["key"] in updates:
                e["translation"] = updates[e["key"]]
        path.write_text(json.dumps(doc, ensure_ascii=False, indent=1) + "\n", encoding="utf-8")
    for pr in problems[:60]:
        print(pr)
    print("寫入 %d 則；問題 %d 則" % (len(updates), len(problems)))
    return 1 if problems else 0


def cmd_sweep(glossary):
    g = json.loads(pathlib.Path(glossary).read_text(encoding="utf-8"))["entries"]
    have = te.load_text(ROOT / "text")
    issues = 0
    for item in g:
        en = item["en"]
        pat = re.compile(r"\b%s\b" % re.escape(en), re.I)
        left = [e["key"] for e in have.values() if e["translation"] and pat.search(e["translation"])]
        if left:
            issues += len(left)
            print("譯文殘留英文名 %s（應為 %s）：%s" % (en, item["zh"], " ".join(left[:8])))
        hits = [e for e in have.values() if e["translation"] and pat.search(te.decode_shown(e["original"]).decode("latin-1"))]
        missing = [e["key"] for e in hits if item["zh"] not in e["translation"]]
        if missing and len(missing) < len(hits):
            print("  %s：原文有、譯文沒有「%s」的 %d／%d 則：%s" % (en, item["zh"], len(missing), len(hits), " ".join(missing[:8])))
    print("殘留 %d 處" % issues)


def main(argv):
    if argv[:1] == ["prep"] and len(argv) >= 2:
        size = int(argv[argv.index("--size") + 1]) if "--size" in argv else 130
        cmd_prep(argv[1], size)
        return 0
    if argv == ["merge"]:
        return cmd_merge()
    if argv[:1] == ["sweep"] and len(argv) == 2:
        cmd_sweep(argv[1])
        return 0
    print(__doc__)
    return 2


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
