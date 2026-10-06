"""保全本輪精確來源及產物；入口研究038 §48。"""
import hashlib
import json
import pathlib

root = pathlib.Path(".").resolve()
work = root / "workplace/hd"
output = work / "next-bodies-source-manifest-v1-20261001.json"
assert not output.exists(), "拒絕覆寫來源manifest"
paths = set()
recorded_originals = {}
for relative in (
    "workplace/hd/next-body-kasuruji-v2-20261001.json",
    "workplace/hd/next-body-minton-v2-20261001.json",
    "workplace/hd/next-bodies-verification-v2-20261001.json",
    "workplace/hd/next-body-runtime-v1-20261001.json",
    "workplace/hd/next-body-plane-verification-v1-20261001.json",
    "workplace/hd/theme-next-bodies-v1-20261001/asset-map.json",
):
    receipt = root / relative
    paths.add(receipt)
    doc = json.loads(receipt.read_text())
    for name, expected in doc.get("inputs_sha256", {}).items():
        path = pathlib.Path(name)
        if name.startswith("/orig/"):
            assert hashlib.sha256(path.read_bytes()).hexdigest() == expected
            recorded_originals[name] = expected
            continue
        if name.startswith("/src/"):
            path = root / name[5:]
        elif not path.is_absolute():
            path = root / path
        assert hashlib.sha256(path.read_bytes()).hexdigest() == expected, name
        paths.add(path)
for pattern in (
    "next-body-kasuruji-v1-20261001-event*",
    "next-body-minton-v1-20261001-event*",
    "next-body-kasuruji-v2-20261001*",
    "next-body-minton-v2-20261001*",
    "next-body-runtime-v1-20261001*",
    "next-body-theme-tests-v[12]-20261001.jsonl",
    "next-bodies-before-*-20261001",
):
    paths.update(work.glob(pattern))
paths.update((work / "theme-next-bodies-v1-20261001").iterdir())
for relative in (
    "workplace/hd/observe-next-bodies-v1-20261001",
    "workplace/hd/observe-next-bodies-v2-20261001",
    "workplace/hd/next-bodies-observer-prefix-v1-20261001.go",
    "workplace/hd/next-bodies-verifier-fullpose-v1-20261001.py",
    "tools/pbl.py", "go.mod", "go.sum",
    "worktrees/dosgolem/go.mod",
    "docs/spec/024-hd-theme.md", "docs/re/038-hd-theme-feasibility.md",
    "tools/hd/observe_next_bodies.go", "tools/hd/verify_next_bodies.py",
    "tools/hd/prepare_next_body_theme.py", "tools/hd/verify_next_body_runtime.go",
    "tools/hd/verify_next_body_plane.py", "tools/hd/preserve_next_bodies.py",
):
    paths.add(root / relative)
for directory in (
    "apps/psychicwar/theme", "apps/psychicwar/pbl",
    "worktrees/dosgolem/oracle", "worktrees/dosgolem/xlate",
    "worktrees/dosgolem/internal/cpu", "worktrees/dosgolem/internal/machine",
    "worktrees/dosgolem/internal/dos", "worktrees/dosgolem/internal/state",
):
    paths.update((root / directory).glob("*.go"))
files = {}
for path in sorted(paths):
    assert path.is_file(), path
    relative = str(path.relative_to(root))
    # 原版／正常存檔只記雜湊；已有原始檔保持本機原位，不複製原版素材。
    data = path.read_bytes()
    st = path.stat()
    assert (st.st_uid, st.st_gid) == (1000, 1000), relative
    record = {"sha256": hashlib.sha256(data).hexdigest(), "size": len(data), "uid": st.st_uid, "gid": st.st_gid}
    if relative.startswith(("tools/", "apps/", "docs/", "worktrees/")) or relative in ("go.mod", "go.sum"):
        snapshot = work / ("next-bodies-source-" + hashlib.sha256(relative.encode()).hexdigest()[:16] + "-20261001")
        if snapshot.exists() and snapshot.read_bytes() != data:
            prior = snapshot.read_bytes()
            record["prior_partial_snapshot"] = {"path": str(snapshot.relative_to(root)), "sha256": hashlib.sha256(prior).hexdigest(), "size": len(prior)}
            snapshot = pathlib.Path(str(snapshot) + "-" + hashlib.sha256(data).hexdigest()[:16])
        if snapshot.exists():
            assert snapshot.read_bytes() == data, "拒絕改寫舊來源"
        else:
            with snapshot.open("xb") as f:
                f.write(data)
        record["snapshot"] = str(snapshot.relative_to(root))
    files[relative] = record
with output.open("x") as f:
    assert not (root / "worktrees/dosgolem/go.sum").exists(), "dosgolem輸入集合已變更，重新審查"
    json.dump({"scope": "後續正常身體來源與限定正式接入的來源／產物保全", "files": files, "originals_sha256": recorded_originals, "absent_inputs": ["worktrees/dosgolem/go.sum"], "limits": "僅本機，不證明全sprite、美術或完整HD玩家路徑；重跑以前核對保存來源"}, f, ensure_ascii=False, indent=2)
    f.write("\n")
print("來源保全", len(files), "項", sum(row["size"] for row in files.values()), "bytes")
