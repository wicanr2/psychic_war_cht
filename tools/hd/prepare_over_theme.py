"""已驗十九筆主題加陣亡人物新重繪候選；研究038 §51，僅限本機。"""
import hashlib
import json
import pathlib
import shutil
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent.parent))
import pbl

root = pathlib.Path("workplace/hd")
old = root / "theme-sivad-v1-20261002"
new = root / "theme-over-v1-20261002"
assert not new.exists(), "拒絕覆寫主題"
assert root.is_dir() and root.stat().st_uid == 1000
manifest = json.loads((old / "manifest.json").read_text())
assert len(manifest["entries"]) == 19
selected = [(old / e["png"], e["png"]) for e in manifest["entries"]]
source = root / "redraw/OVER-00-hd-v1-20261002.png"
w, h, channels, _, palette = pbl.read_png(source)
assert (w, h, channels) == (192, 192, 4) and palette is None
selected.append((source, "OVER-00.png"))
manifest["entries"].append({"pbl": "OVER.PBL", "image": 0, "at": [128, 48], "png": "OVER-00.png", "kind": "redraw"})
manifest["name"] = "hd-over-local"
provenance = [old / "manifest.json", pathlib.Path(__file__).resolve(), source,
              root / "redraw/OVER-00-hd-native-v1-20261002.png",
              root / "redraw/OVER-00-generation-job-v1-20261002.json",
              root / "redraw/OVER-00-selected-v1-20261002.json"]
assert all(p.is_file() and p.stat().st_uid == 1000 for p in provenance + [p for p, _ in selected])
inputs = {str(p.resolve().relative_to(pathlib.Path.cwd())): hashlib.sha256(p.read_bytes()).hexdigest()
          for p in provenance + [p for p, _ in selected]}
new.mkdir()
for source, name in selected:
    shutil.copyfile(source, new / name)
    assert (new / name).stat().st_uid == 1000
for name, data in (("manifest.json", manifest), ("asset-map.json", {"inputs_sha256": inputs, "limits": "本機技術接入；候選美術與散布權利未驗，舊十九筆及原候選保留"})):
    with (new / name).open("x") as f:
        json.dump(data, f, ensure_ascii=False, indent=2)
        f.write("\n")
print("新本機主題20筆；舊19筆及原候選未覆寫")
