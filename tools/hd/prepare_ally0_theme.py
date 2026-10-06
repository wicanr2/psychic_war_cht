"""另存25筆主題，只更新已核對的ALLY #0素材。入口研究038 §78。"""
from pathlib import Path
import argparse
import hashlib
import json
import shutil
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import pbl


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base", type=Path, required=True)
    parser.add_argument("--asset", type=Path, required=True)
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()
    base, asset, out = args.base.resolve(), args.asset.resolve(), args.out.resolve()
    if not base.is_dir() or not asset.is_file() or out.exists() or not out.parent.is_dir():
        parser.error("來源需存在，輸出需為新目錄且父目錄已存在")
    manifest_path = base / "manifest.json"
    manifest = json.loads(manifest_path.read_text())
    entries = manifest["entries"]
    assert len(entries) == 25
    ally = [e for e in entries if e["pbl"] == "ALLY.PBL"]
    assert len(ally) == 1 and ally[0]["image"] == 0 and ally[0]["at"] == [264, 152]
    assert ally[0]["png"] == "ALLY-00.png"
    w, h, _, _, _ = pbl.read_png(asset)
    assert (w, h) == (72, 96)
    names = ["manifest.json"] + [e["png"] for e in entries]
    assert len(set(names)) == len(names)
    for name in names:
        assert Path(name).name == name and (base / name).is_file()
    out.mkdir()
    for name in names:
        shutil.copyfile(base / name, out / name)
    shutil.copyfile(asset, out / "ALLY-00.png")
    sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
    changed = [e["png"] for e in entries if sha(base / e["png"]) != sha(out / e["png"])]
    assert changed == ["ALLY-00.png"]
    assert (out / "manifest.json").read_bytes() == manifest_path.read_bytes()
    receipt = {
        "scope": "local theme selection after ALLY #0 art and normal-path review; not full HD acceptance",
        "base": str(base), "asset": str(asset), "changed_assets": changed,
        "entries": len(entries), "manifest_sha256": sha(manifest_path),
        "inputs": {str(p): sha(p) for p in [asset, Path(__file__), manifest_path] + [base / e["png"] for e in entries]},
        "outputs": {name: sha(out / name) for name in names},
    }
    (out / "selection.json").write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n")
    print("另存25筆主題；僅ALLY #0素材更新，其他24筆與manifest相同。")


if __name__ == "__main__":
    main()
