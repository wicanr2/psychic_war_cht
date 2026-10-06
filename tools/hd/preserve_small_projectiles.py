"""研究038 §56：保全正常小圖塊來源、工具與產物，原版素材不複製。"""
import hashlib
import json
import os
from pathlib import Path


def sha(p):
    return hashlib.sha256(Path(p).read_bytes()).hexdigest()


def owner(p):
    s = Path(p).stat()
    assert (s.st_uid, s.st_gid) == (1000, 1000), str(p)


def main():
    root = Path("workplace/hd")
    snapshot = root / "source-small-projectiles-v1-20261002"
    manifest = root / "small-projectiles-source-manifest-v1-20261002.json"
    receipt = root / "small-projectiles-minton-independent-v1-20261002.json"
    result = json.loads(receipt.read_text())
    assert not snapshot.exists() and not manifest.exists(), "拒絕覆寫"
    owner(root)
    files = set(result["files_sha256"])
    originals = dict(result["original_archives_sha256"])
    for f, wanted in result["files_sha256"].items():
        assert sha(f) == wanted, f
        if f.startswith("/orig/"):
            originals[f] = wanted
            files.remove(f)
    files.update(str(p) for p in root.glob("small-projectiles-minton-v1-20261002*"))
    files.update(str(p) for p in root.glob("small-projectiles-build-v1-20261002.*"))
    files.update((str(receipt), str(root / "observe-small-projectiles-v1-20261002")))
    mutable = {"tools/hd/observe_small_projectiles.go", "tools/hd/verify_small_projectiles.py",
               "tools/hd/preserve_small_projectiles.py", "tools/pbl.py", "go.mod", "go.sum",
               "replay/title-to-first-save.json", "worktrees/dosgolem/go.mod",
               "CONTEXT.md", "AGENTS.md", "docs/spec/024-hd-theme.md",
               "docs/re/038-hd-theme-feasibility.md"}
    for directory, dirs, names in os.walk("worktrees/dosgolem"):
        dirs[:] = [d for d in dirs if d not in (".git", "workplace", "docs", "cmd", "tools", "vendor")]
        for name in names:
            if name.endswith(".go") and not name.endswith("_test.go"):
                mutable.add(str(Path(directory) / name))
    files.update(mutable)
    for f in originals:
        assert sha(f) == originals[f]
    # 所有來源與UID先預檢，避免失敗時留下半套快照。
    for f in files:
        p = Path(f)
        assert not p.is_absolute() and ".." not in p.parts and p.is_file(), f
        owner(p)
    snapshot.mkdir()
    rows = {}
    for f in sorted(files):
        p = Path(f)
        row = {"sha256": sha(p), "size": p.stat().st_size}
        if f in mutable:
            dest = snapshot / p
            dest.parent.mkdir(parents=True, exist_ok=True)
            with dest.open("xb") as out:
                out.write(p.read_bytes())
            assert sha(dest) == row["sha256"]
            owner(dest)
            row["snapshot"] = str(dest)
        rows[f] = row
    binary = root / "observe-small-projectiles-v1-20261002"
    original = json.loads((root / "small-projectiles-minton-v1-20261002.json").read_text())
    assert sha(binary) == original["binary_sha256"]
    doc = {"scope": "原版正常敏頓小圖塊研究；全部state／來源／產物只留本機",
           "go_image": "sha256:1a6d4452c65dea36aac2e2d606b01b4a029ec90cc1ae53890540ce6173ea77ac",
           "go_version": original["go_version"], "python_version": result["python_version"],
           "files": rows, "originals_sha256": originals,
           "snapshot_note": "可變程式及文件回查snapshot；收尾補記可更新現況文件，不覆寫本清單"}
    with manifest.open("x") as out:
        json.dump(doc, out, ensure_ascii=False, indent=2)
        out.write("\n")
    owner(manifest)
    print(json.dumps({"files": len(rows), "bytes": sum(r["size"] for r in rows.values()),
                      "mutable_snapshots": sum("snapshot" in r for r in rows.values()),
                      "original_hashes": len(originals)}, ensure_ascii=False))


if __name__ == "__main__":
    main()
