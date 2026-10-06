"""保存治療房間來源收據與精確工具快照；入口研究038 §65。"""
import hashlib
import json
import pathlib

ROOT = pathlib.Path("/src")
HD = ROOT / "workplace/hd"


def sha(data):
    return hashlib.sha256(data).hexdigest()


def main():
    target = HD / "source-room-normal-v1-20261002"
    out = HD / "room-normal-source-manifest-v1-20261002.json"
    if target.exists() or out.exists():
        raise ValueError("拒絕覆寫來源快照")
    prior_path = HD / "checkpoints-source-manifest-v1-20261002.json"
    prior = json.loads(prior_path.read_text())
    for source, row in prior["files"].items():
        path = ROOT / row.get("snapshot", source)
        if sha(path.read_bytes()) != row["sha256"]:
            raise ValueError("前批來源改變：" + source)
    paths = {ROOT / key for key in prior["files"] if key.endswith(".go") or pathlib.Path(key).name in ("go.mod", "go.sum")}
    originals = {}
    receipts = [HD / "room-normal-v2-20261002.json", HD / "room-normal-independent-v1-20261002.json"]
    for receipt in receipts:
        doc = json.loads(receipt.read_text())
        paths.add(receipt)
        for key, expected in doc["inputs_sha256"].items():
            path = pathlib.Path(key)
            if key.startswith("/orig/"):
                path = ROOT / "workplace/original" / path.relative_to("/orig")
                if sha(path.read_bytes()) != expected:
                    raise ValueError("原版輸入改變：" + key)
                originals[key] = expected
                continue
            if not path.is_absolute():
                path = ROOT / path
            if sha(path.read_bytes()) != expected:
                raise ValueError("收據輸入改變：" + key)
            paths.add(path)
    paths.update(HD.glob("room-normal-v2-20261002*"))
    paths.update(ROOT / key for key in ["tools/hd/observe_room.go", "tools/hd/verify_room.py", "tools/hd/preserve_room.py", "tools/pbl.py", "CONTEXT.md", "WORKLOG.md", "docs/re/038-hd-theme-feasibility.md", "docs/spec/024-hd-theme.md", "docs/worklist.json", "workplace/hd/art-in/ROOM0-08.png", "workplace/hd/ref/ROOM0-08.png", "workplace/hd/spec-ROOM0.md", "workplace/hd/preserve-room-source-v1-20261002.py", "workplace/hd/observe-room-source-v1-20261002.go", "workplace/hd/observe-room-v1-20261002.bin", "workplace/hd/observe-room-v2-20261002.bin", "workplace/hd/room-normal-overlay-v1-20261002.json", "workplace/hd/dat-runtime-v1-20261002.go.work"])
    prepared = []
    for path in sorted(paths):
        relative = path.relative_to(ROOT)
        if not path.is_file() or "original" in relative.parts:
            raise ValueError("來源形態或散布邊界不符：" + str(path))
        stat = path.stat()
        if (stat.st_uid, stat.st_gid) != (1000, 1000):
            raise ValueError("來源擁有權不符：" + str(path))
        prepared.append((relative, path.read_bytes()))
    if (HD.stat().st_uid, HD.stat().st_gid) != (1000, 1000):
        raise ValueError("輸出目錄擁有權不符")
    target.mkdir()
    rows = {}
    for relative, data in prepared:
        snapshot = target / relative
        snapshot.parent.mkdir(parents=True, exist_ok=True)
        with snapshot.open("xb") as f:
            f.write(data)
        rows[str(relative)] = {"snapshot": str(snapshot.relative_to(ROOT)), "sha256": sha(data), "size": len(data), "uid": snapshot.stat().st_uid, "gid": snapshot.stat().st_gid}
    result = {"scope": "正常治療路線的原版來源證據、初版失敗來源／二進位與精確Go依賴快照；全部本機研究輸入", "files": rows, "originals_sha256": originals, "prior_manifest": {"path": str(prior_path.relative_to(ROOT)), "sha256": sha(prior_path.read_bytes()), "files_checked_unchanged": len(prior["files"])}, "limits": "不是HD正式接入或美術驗收；保存既有ROOM0候選只支持來源追溯，不證明造型或公開散布權利"}
    with out.open("x", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print("房間來源保全", len(rows), "項", sum(row["size"] for row in rows.values()), "bytes；前批", len(prior["files"]), "項未變")


if __name__ == "__main__":
    main()
