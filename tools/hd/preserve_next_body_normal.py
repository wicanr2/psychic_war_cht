"""保全正常HD／中文抽測的精確來源與產物；入口研究038 §49。"""
import hashlib
import json
import pathlib

root = pathlib.Path.cwd()
work = root / "workplace/hd"
output = work / "next-body-normal-source-manifest-v1-20261001.json"
assert not output.exists(), "拒絕覆寫來源manifest"
paths = set()
originals = {}


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def resolve(name):
    if name.startswith("/orig/"):
        return pathlib.Path(name)
    if name.startswith("/src/"):
        return root / name[5:]
    path = pathlib.Path(name)
    return path if path.is_absolute() else root / path


for relative in (
    "workplace/hd/normal-chain-translation-v1-20261001.json",
    "workplace/hd/next-body-normal-kasuruji-v3-20261001.json",
    "workplace/hd/next-body-normal-minton-v3-20261001.json",
    "workplace/hd/next-body-normal-verification-v3-20261001.json",
):
    receipt = root / relative
    paths.add(receipt)
    for name, expected in json.loads(receipt.read_text())["inputs_sha256"].items():
        path = resolve(name)
        assert path.is_file() and digest(path) == expected, name
        if name.startswith("/orig/"):
            originals[name] = expected
        else:
            paths.add(path)

# 舊收據的工具已由新版本替換；以保存的實際bytes核對，不能讀現行工具冒稱舊來源未變。
legacy = {}
for label, version, source in (
    ("minton", "v1", "next-body-normal-observer-before-diagnostic-v1-20261001.go"),
    ("kasuruji", "v2", "next-body-normal-observer-before-translation-v2-20261001.go"),
):
    receipt = work / f"next-body-normal-{label}-{version}-20261001.json"
    paths.add(receipt)
    expected = json.loads(receipt.read_text())["inputs_sha256"]["tools/hd/verify_next_body_normal.go"]
    frozen = work / source
    assert digest(frozen) == expected, "舊工具保存來源不符"
    paths.add(frozen)
    legacy[str(receipt.relative_to(root))] = {"tool_snapshot": str(frozen.relative_to(root)), "sha256": expected}

prior = work / "next-bodies-source-manifest-v1-20261001.json"
paths.add(prior)
prior_doc = json.loads(prior.read_text())
for name, row in prior_doc["files"].items():
    path = root / row.get("snapshot", name)
    assert digest(path) == row["sha256"], "前輪保全來源改變：" + name
    paths.add(path)
for pattern in (
    "next-body-normal-*", "verify-next-body-normal-v*-20261001",
    "normal-chain-*", "rebuild-normal-translation-v1-20261001",
):
    paths.update(work.glob(pattern))
for name in (
    "tools/hd/verify_next_body_normal.go", "tools/hd/verify_next_body_normal.py",
    "tools/hd/rebuild_normal_translation.go", "tools/hd/preserve_next_body_normal.py",
    "CONTEXT.md", "WORKLOG.md", "docs/re/038-hd-theme-feasibility.md",
    "docs/spec/024-hd-theme.md", "docs/worklist.json", "docs/worklist.md",
    "go.mod", "go.sum", "worktrees/dosgolem/go.mod",
):
    paths.add(root / name)
for directory in (
    "apps/psychicwar/theme", "apps/psychicwar/translator", "apps/psychicwar/pbl",
    "worktrees/dosgolem/oracle", "worktrees/dosgolem/xlate",
    "worktrees/dosgolem/internal/cpu", "worktrees/dosgolem/internal/machine",
    "worktrees/dosgolem/internal/dos", "worktrees/dosgolem/internal/state",
):
    paths.update((root / directory).glob("*.go"))
files = {}
for path in sorted(paths):
    assert path.is_file(), path
    name = str(path.relative_to(root))
    data, st = path.read_bytes(), path.stat()
    assert (st.st_uid, st.st_gid) == (1000, 1000), name
    row = {"sha256": hashlib.sha256(data).hexdigest(), "size": len(data), "uid": st.st_uid, "gid": st.st_gid}
    if name.startswith(("tools/", "apps/", "docs/", "worktrees/")) or name in ("CONTEXT.md", "WORKLOG.md", "go.mod", "go.sum"):
        snapshot = work / ("normal-path-source-" + hashlib.sha256(name.encode()).hexdigest()[:16] + "-20261001")
        assert not snapshot.exists(), "拒絕覆寫舊來源"
        with snapshot.open("xb") as f:
            f.write(data)
        row["snapshot"] = str(snapshot.relative_to(root))
    files[name] = row
with output.open("x") as f:
    json.dump({"scope": "正常方向／等待／F3的原版來源、HD中文抽測及六段正常中文起點", "files": files,
               "originals_sha256": originals, "legacy_receipts": legacy,
               "prior_manifest": {"path": str(prior.relative_to(root)), "sha256": digest(prior), "verified_files": len(prior_doc["files"])},
               "limits": "本機原版輸入及state不可公開；不證明完整HD、所有sprite、美術或全遊戲中文驗收。"}, f, ensure_ascii=False, indent=2)
    f.write("\n")
print("來源保全", len(files), "項", sum(row["size"] for row in files.values()), "bytes", flush=True)
