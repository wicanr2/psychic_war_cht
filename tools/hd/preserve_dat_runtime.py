"""保全原版DAT正常重播、失敗診斷與獨立核對來源；入口研究038 §63。"""
import hashlib
import json
from pathlib import Path
import shutil


def sha(data):
    return hashlib.sha256(data).hexdigest()


def main():
    root = Path("workplace/hd")
    target = root / "source-dat-runtime-v1-20261002"
    manifest = root / "dat-source-manifest-v1-20261002.json"
    assert not target.exists() and not manifest.exists(), "拒絕覆寫"
    assert (root.stat().st_uid, root.stat().st_gid) == (1000, 1000)
    independent = root / "dat-independent-v1-20261002.json"
    result = json.loads(independent.read_text())
    paths = {independent, Path(__file__).resolve().relative_to(Path.cwd())}
    originals = {}
    for name, expected in result["inputs_sha256"].items():
        source = Path(name)
        assert source.is_file() and sha(source.read_bytes()) == expected, name
        if name.startswith("/orig/"):
            originals[name] = expected
        elif name.startswith("/src/"):
            paths.add(source.relative_to("/src"))
        else:
            assert not source.is_absolute() and ".." not in source.parts, name
            paths.add(source)
    for pattern in ["dat-runtime-*20261002*", "dat-independent-v1-20261002.json"]:
        for p in root.glob(pattern):
            if p.is_file():
                paths.add(p)
            elif p.is_dir():
                paths.update(q for q in p.rglob("*") if q.is_file())
    paths.update(Path(p) for p in [
        "tools/hd/verify_dat_runtime.go", "tools/hd/verify_dat_runtime.py",
        "workplace/hd/normal-chain-oracle-bridge-v1-20261001.go",
        "worktrees/dosgolem/workplace/hd_dat_runtime_v1_20261002.go",
        "CONTEXT.md", "WORKLOG.md", "docs/spec/024-hd-theme.md",
        "docs/re/038-hd-theme-feasibility.md", "docs/worklist.json", "docs/worklist.md",
        "tools/docker/go-ebiten.Dockerfile",
    ])
    prior = root / "redraw/ENEMY00-ENEMY11-small-inventory-source-manifest-v1-20261002.json"
    prior_data = json.loads(prior.read_text())
    for name, row in prior_data["files"].items():
        p = Path(row.get("snapshot", name))
        assert sha(p.read_bytes()) == row["sha256"], name
    for source in paths:
        assert source.is_file() and not source.is_absolute() and ".." not in source.parts, str(source)
        st = source.stat()
        assert (st.st_uid, st.st_gid) == (1000, 1000), str(source)
    files = {}
    target.mkdir()
    for source in sorted(paths):
        st = source.stat()
        data = source.read_bytes()
        dest = target / source
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, dest)
        assert sha(dest.read_bytes()) == sha(data)
        assert dest.stat().st_mtime_ns == st.st_mtime_ns
        files[str(source)] = {"sha256": sha(data), "size": len(data), "snapshot": str(dest), "mtime_ns": st.st_mtime_ns, "uid": 1000, "gid": 1000}
    doc = {"scope": "二十筆HD主題的兩次原版DAT存檔與兩次LOAD GAME，原版／HD關閉／HD開啟12段144取樣及失敗版本", "files": files, "originals_sha256": originals, "prior_manifest": {"path": str(prior), "sha256": sha(prior.read_bytes()), "verified_snapshots": len(prior_data["files"])}, "tool_versions": {"go": "1.24.13", "python": "3.13.15", "docker_image_id": "sha256:083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7"}, "limits": "全部留本機。metadata快照用於還原FindFirst測試輸入，未修改原版RAM、seed或EXE；不代表GUI、封包、美術或全部sprite通過。進度文件以本次snapshot回查。"}
    with manifest.open("x") as f:
        json.dump(doc, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print("保全", len(files), "項", sum(r["size"] for r in files.values()), "bytes；前批", len(prior_data["files"]), "份來源未變")


if __name__ == "__main__":
    main()
