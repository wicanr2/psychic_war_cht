"""譯者自我檢查（唯讀）：一個批次的 .done.json 是否符合 docs/spec/009 §3 的行寬、Big5 字元與欄位不變。

    tools/py.sh tools/l10n_check.py workplace/l10n/batches/NN.done.json
"""
import json
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import text_extract as te  # noqa: E402

ROOT = pathlib.Path(__file__).resolve().parent.parent


def main(argv):
    if len(argv) != 1:
        print(__doc__)
        return 2
    done = json.loads(pathlib.Path(argv[0]).read_text(encoding="utf-8"))
    src = pathlib.Path(argv[0].replace(".done.json", ".json"))
    orig = {e["key"]: e for e in json.loads(src.read_text(encoding="utf-8"))["entries"]} if src.exists() else {}
    have = te.load_text(ROOT / "text")
    bad = 0
    for it in done["entries"]:
        k = it.get("key")
        e = have.get(k)
        msg = []
        if e is None:
            msg.append("key 不在文本檔")
        else:
            if it.get("original") != e["original"]:
                msg.append("original 被改動")
            tr = it.get("translation", "")
            if not tr:
                msg.append("沒有譯文")
            elif tr == e["original"]:
                pass  # 保留原文（含 {XX} 記號）：不檢查格數
            else:
                lines, too = te.layout(tr, te.line_widths(e))
                if too:
                    msg.append("過長：行寬 %s，你的分行 %s" % (te.line_widths(e), [len(x) for x in tr.split("\n")]))
                bc = te.bad_chars(tr)
                if bc:
                    msg.append("非 Big5 字元 %s" % "".join(bc))
        if orig and k in orig and any(it.get(f) != orig[k].get(f) for f in ("lines", "kind")):
            msg.append("lines 或 kind 被改動")
        if msg:
            bad += 1
            print("%s：%s" % (k, "；".join(msg)))
    missing = set(orig) - {it.get("key") for it in done["entries"]}
    if missing:
        bad += len(missing)
        print("少了 %d 則：%s" % (len(missing), " ".join(sorted(missing)[:10])))
    print("共 %d 則，問題 %d 則" % (len(done["entries"]), bad))
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
