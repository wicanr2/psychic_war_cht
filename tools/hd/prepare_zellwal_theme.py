"""現行25筆旁新增ENEMY03三張技術候選；研究038 §91，美術未接受。"""
import hashlib
import json
import pathlib
import shutil
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent.parent))
import pbl

root = pathlib.Path("workplace/hd")
old = root / "theme-group1-pose3-v1-20261003"
new = root / "theme-zellwal-candidate-v1-20261003"
assert not new.exists(), "拒絕覆寫主題"
assert old.is_dir() and root.stat().st_uid == 1000
manifest = json.loads((old / "manifest.json").read_text())
assert len(manifest["entries"]) == 25
selected = [(old / e["png"], e["png"]) for e in manifest["entries"]]
for image in (3, 4, 5):
    source = root / "redraw" / f"ENEMY03-group1-frame-{image % 3:02d}.png"
    w, h, _, _, _ = pbl.read_png(source)
    assert (w, h) == (72, 96), "候選尺寸不符"
    name = f"ENEMY03-{image:02d}.png"
    selected.append((source, name))
    manifest["entries"].append({"pbl": "ENEMY03.PBL", "image": image, "at": [32, 152], "png": name, "kind": "redraw"})
assert all(p.is_file() for p, _ in selected)
inputs = {str(old / "manifest.json"): hashlib.sha256((old / "manifest.json").read_bytes()).hexdigest()}
inputs[__file__] = hashlib.sha256(pathlib.Path(__file__).read_bytes()).hexdigest()
new.mkdir()
for source, name in selected:
    shutil.copyfile(source, new / name)
    inputs[str(source)] = hashlib.sha256(source.read_bytes()).hexdigest()
    assert (new / name).stat().st_uid == 1000
for name, data in (("manifest.json", manifest), ("asset-map.json", {"inputs_sha256": inputs, "limits": "僅本機技術接入候選；美術及散布權利未驗"})):
    with (new / name).open("x") as f:
        json.dump(data, f, ensure_ascii=False, indent=2)
        f.write("\n")
print("新技術候選28筆；現行25筆未覆寫，美術仍未接受")
