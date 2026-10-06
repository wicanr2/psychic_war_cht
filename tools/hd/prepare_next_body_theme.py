"""複製既有十筆主題與六張候選；入口研究038 §48，全部留本機。"""
import hashlib
import json
import pathlib
import shutil

root = pathlib.Path("workplace/hd")
old = root / "theme-enemy00-v1-20261001"
new = root / "theme-next-bodies-v1-20261001"
assert not new.exists(), "拒絕覆寫主題"
assert old.is_dir() and new.parent.is_dir()
assert (new.parent.stat().st_uid, new.parent.stat().st_gid) == (1000, 1000)
manifest = json.loads((old / "manifest.json").read_text())
assert len(manifest["entries"]) == 10
inputs = {}
selected = []
for entry in manifest["entries"]:
    selected.append((old / entry["png"], entry["png"]))
for image in (0, 1, 2, 6, 7, 8):
    source = root / "redraw" / f"ENEMY00-group{image // 3}-frame-{image % 3:02d}.png"
    name = f"ENEMY00-{image:02d}.png"
    assert source.is_file()
    selected.append((source, name))
    manifest["entries"].append({"pbl": "ENEMY00.PBL", "image": image, "at": [32, 152], "png": name, "kind": "redraw"})
assert all(source.is_file() for source, _ in selected)
new.mkdir()
for source, name in selected:
    shutil.copyfile(source, new / name)
    inputs[str(source)] = hashlib.sha256(source.read_bytes()).hexdigest()
inputs[str(old / "manifest.json")] = hashlib.sha256((old / "manifest.json").read_bytes()).hexdigest()
inputs[__file__] = hashlib.sha256(pathlib.Path(__file__).read_bytes()).hexdigest()
for name, data in (("manifest.json", manifest), ("asset-map.json", {"inputs_sha256": inputs, "limits": "僅本機候選，正式美術及散布權利未驗"})):
    with (new / name).open("x") as f:
        json.dump(data, f, ensure_ascii=False, indent=2)
        f.write("\n")
print("本機主題16筆，六張新姿勢複製完成，舊主題未覆寫")
