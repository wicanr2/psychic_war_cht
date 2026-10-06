"""保全交通房間來源、READY限定實作與單元結果；研究038 §68。"""
import hashlib
import json
from pathlib import Path

ROOT = Path("/src")
HD = ROOT / "workplace/hd"


def sha(data):
    return hashlib.sha256(data).hexdigest()


def main():
    target = HD / "source-transport-room-v1-20261003"
    output = HD / "transport-room-source-manifest-v1-20261003.json"
    if target.exists() or output.exists():
        raise ValueError("拒絕覆寫")
    prior_path = HD / "background-anchor-source-manifest-v1-20261003.json"
    prior = json.loads(prior_path.read_text())
    for source, row in prior["files"].items():
        path = ROOT / row["snapshot"]
        if sha(path.read_bytes()) != row["sha256"]:
            raise ValueError("前批來源快照改變：" + source)
    paths = {ROOT / key for key in prior["files"] if key.endswith(".go") or Path(key).name in ("go.mod", "go.sum")}
    for pattern in ("apps/psychicwar/*.go", "cmd/pwstep/*.go", "cmd/psychicwar/*.go"):
        paths.update(ROOT.glob(pattern))
    originals = {}
    receipts = [HD / name for name in (
        "transport-room-source-v3-20261003.json",
        "transport-room-independent-v2-20261003.json", "transport-room-grid-v1-20261003.json",
    )]
    for receipt in receipts:
        doc = json.loads(receipt.read_text())
        paths.add(receipt)
        for key, expected in doc["inputs_sha256"].items():
            path = Path(key)
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
    paths.update(p for p in HD.glob("*transport*20261003*") if p.is_file())
    paths.update(p for p in (HD / "theme-transport-room-v1-20261003").rglob("*") if p.is_file())
    paths.update(ROOT.glob("apps/psychicwar/theme/*.go"))
    paths.update(ROOT / key for key in (
        "tools/hd/preserve_transport_rooms.py", "tools/hd/observe_transport_rooms.go",
        "tools/hd/verify_transport_rooms.py", "tools/hd/verify_transport_room_grid.py",
        "tools/hd/prepare_transport_room_theme.py", "tools/hd/verify_room_grid.py",
        "CONTEXT.md", "WORKLOG.md", "docs/worklist.json", "docs/worklist.md",
        "docs/re/038-hd-theme-feasibility.md", "docs/spec/024-hd-theme.md",
        "workplace/hd/dat-runtime-v1-20261002.go.work",
    ))
    prepared = []
    for path in sorted(paths):
        relative = path.relative_to(ROOT)
        if not path.is_file() or "original" in relative.parts:
            raise ValueError("來源形態或散布邊界不符：" + str(path))
        if (path.stat().st_uid, path.stat().st_gid) != (1000, 1000):
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
        rows[str(relative)] = {"snapshot": str(snapshot.relative_to(ROOT)), "sha256": sha(data), "size": len(data),
                               "uid": snapshot.stat().st_uid, "gid": snapshot.stat().st_gid}
    with output.open("x") as f:
        json.dump({"scope": "正常交通三房間來源、限定READY實作及單元驗收精確來源，全部留本機",
                   "files": rows, "originals_sha256": originals,
                   "prior_manifest": {"path": str(prior_path.relative_to(ROOT)), "sha256": sha(prior_path.read_bytes()),
                                      "files_checked_unchanged": len(prior["files"])},
                   "limits": "ROOM0 #0／#3／#2原版來源與限定載入通過；新24筆主題尚未驗正常HD／中文整屏、載回接續、pwstep、美術、GUI或正式交付。"},
                  f, ensure_ascii=False, indent=2)
        f.write("\n")
    print("交通房間保全", len(rows), "項", sum(r["size"] for r in rows.values()), "bytes；前批", len(prior["files"]), "未變")


if __name__ == "__main__":
    main()
