"""兩個完整陣亡畫面三行中文對正式文本；研究038 §51，非全文驗收。"""
import hashlib
import json
from pathlib import Path


def sha(data):
    return hashlib.sha256(data).hexdigest()


inputs = {}


def read(path):
    p = Path(path)
    data = p.read_bytes()
    inputs[str(p)] = sha(data)
    return json.loads(data)


out = Path("workplace/hd/over-text-verification-v1-20261002.json")
assert not out.exists(), "拒絕覆寫"
doc = read("workplace/hd/over-runtime-verification-v1-20261002.json")
entries = {e["key"]: e["translation"] for e in read("text/PW.EXE.json")["entries"]}
keys = ["PW.EXE:cs:08D7", "PW.EXE:cs:0901", "PW.EXE:cs:092B"]
rows = []
for row in doc["routes"][0]["samples"]:
    if not row["full_over"]:
        continue
    name = row["sample"]["Name"]
    side = read(name + ".state.xlate.json")
    active = {st["key"]: st for st in side["stamps"] if st["state"] == 2}
    assert set(active) == set(keys), "三行正式訊息不完整"
    for i, key in enumerate(keys):
        st = active[key]
        assert st["text"] == entries[key], "陣亡中文與正式譯文不同"
        assert (st["x"], st["y"], st["cell_w"], st["cell_h"], st["font"]) == (0, 128 + i*8, 8, 8, "cjk24")
    assert entries[keys[0]] + "錯" != active[keys[0]]["text"], "改譯文負對照無效"
    rows.append({"sample": name, "keys": keys, "translation_mismatch": 0, "negative_changed_translation_count": 1})
assert len(rows) == 2, "完整陣亡樣本數不同"
p = Path(__file__).resolve().relative_to(Path.cwd())
inputs[str(p)] = sha(p.read_bytes())
with out.open("x") as f:
    json.dump({"scope": "兩個完整陣亡場景的三行中文內容", "inputs_sha256": inputs, "results": rows,
               "limits": "正式文本與實際中文快照對照；字型及合成另由正常21幀收據證實，不外推全文、GUI或美術"}, f, ensure_ascii=False, indent=2)
    f.write("\n")
print("兩個陣亡場景、三行正式中文內容不符0")
