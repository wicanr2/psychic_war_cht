"""複製已驗證24筆主題並增加ROOM0 #22，全部留本機；研究038 §70。"""
import hashlib
import json
from pathlib import Path


def main():
    old = Path("workplace/hd/theme-transport-room-v1-20261003")
    target = Path("workplace/hd/theme-room22-v1-20261003")
    if target.exists():
        raise ValueError("拒絕覆寫")
    manifest = json.loads((old / "manifest.json").read_text())
    if len(manifest["entries"]) != 24:
        raise ValueError("來源不是已驗證24筆主題")
    sources = [(old / e["png"], e["png"]) for e in manifest["entries"]]
    for number in (22,):
        name = f"ROOM0-{number:02d}.png"
        sources.append((Path("workplace/hd/art-in") / name, name))
        manifest["entries"].append({"pbl": "ROOM0.PBL", "image": number, "at": [4, 124],
                                    "png": name, "kind": "redraw"})
    if len({name for _, name in sources}) != 25:
        raise ValueError("重複PNG路徑")
    prepared = []
    for path, name in sources:
        if not path.is_file() or (path.stat().st_uid, path.stat().st_gid) != (1000, 1000):
            raise ValueError("來源形態或擁有權不符")
        if Path(name).is_absolute() or ".." in Path(name).parts:
            raise ValueError("主題路徑越界")
        prepared.append((path, name, path.read_bytes()))
    if (target.parent.stat().st_uid, target.parent.stat().st_gid) != (1000, 1000):
        raise ValueError("輸出根擁有權不符")
    target.mkdir()
    copies = {}
    for source, name, data in prepared:
        output = target / name
        output.parent.mkdir(parents=True, exist_ok=True)
        with output.open("xb") as f:
            f.write(data)
        copies[str(source)] = {"target": str(output), "sha256": hashlib.sha256(data).hexdigest(), "size": len(data)}
    manifest["name"] = "hd-room22-v1"
    (target / "manifest.json").write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n")
    (target / "source-copies.json").write_text(json.dumps(copies, ensure_ascii=False, indent=2) + "\n")
    print("新本機25筆主題已建立，既有24筆與ROOM0 #22候選PNG bytes保持不變")


if __name__ == "__main__":
    main()
