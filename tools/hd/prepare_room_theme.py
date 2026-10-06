"""複製既有二十筆主題並接入限定治療房間候選；研究038 §66。"""
import hashlib
import json
from pathlib import Path


def main():
    old = Path("workplace/hd/theme-over-v1-20261002")
    new = Path("workplace/hd/theme-room-v1-20261003")
    if new.exists():
        raise ValueError("拒絕覆寫主題")
    manifest = json.loads((old / "manifest.json").read_text())
    if len(manifest["entries"]) != 20:
        raise ValueError("既有主題不是二十筆")
    sources = [(old / entry["png"], entry["png"]) for entry in manifest["entries"]]
    sources.append((Path("workplace/hd/art-in/ROOM0-08.png"), "ROOM0-08.png"))
    for source, name in sources:
        if not source.is_file() or (source.stat().st_uid, source.stat().st_gid) != (1000, 1000):
            raise ValueError("來源形態或擁有權不符：" + str(source))
        if Path(name).is_absolute() or ".." in Path(name).parts:
            raise ValueError("主題相對路徑不符")
    manifest["name"] = "hd-room-v1"
    manifest["entries"].append({"pbl": "ROOM0.PBL", "image": 8, "at": [4, 124], "png": "ROOM0-08.png", "kind": "redraw"})
    new.mkdir()
    copies = {}
    for source, name in sources:
        target = new / name
        target.parent.mkdir(parents=True, exist_ok=True)
        data = source.read_bytes()
        with target.open("xb") as f:
            f.write(data)
        copies[str(source)] = {"target": str(target), "sha256": hashlib.sha256(data).hexdigest(), "size": len(data)}
    (new / "manifest.json").write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n")
    (new / "source-copies.json").write_text(json.dumps(copies, ensure_ascii=False, indent=2) + "\n")
    print("新本機主題21筆，舊20筆與ROOM0-08候選bytes保持不變")


if __name__ == "__main__":
    main()
