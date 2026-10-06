"""保存OVER接入的精確來源與產物；研究038 §51，全數留本機。"""
import hashlib
import json
import pathlib
import shutil


def sha(data):
    return hashlib.sha256(data).hexdigest()


root = pathlib.Path("workplace/hd")
out = root / "source-over-v1-20261002"
manifest = root / "over-source-manifest-v1-20261002.json"
assert not out.exists() and not manifest.exists(), "拒絕覆寫來源"
assert (root.stat().st_uid, root.stat().st_gid) == (1000, 1000)
prior = root / "sivad-source-manifest-v1-20261002.json"
old = json.loads(prior.read_text())
for name, row in old["files"].items():
    source = pathlib.Path(row.get("snapshot") or name)
    assert source.is_file(), str(source)
    assert sha(source.read_bytes()) == row["sha256"], "舊保存來源變更：" + str(source)

paths = {prior, pathlib.Path(__file__).relative_to("/src")}
originals = {}
receipt_names = ["over-source-v1-20261002.json", "over-scene-source-v2-20261002.json",
                 "over-source-verification-v1-20261002.json", "over-runtime-v1-20261002.json",
                 "over-reload-v1-20261002.json", "over-runtime-verification-v1-20261002.json", "over-text-verification-v1-20261002.json"]
for name in receipt_names:
    p = root / name
    doc = json.loads(p.read_text())
    paths.add(p)
    for source, expected in doc.get("inputs_sha256", {}).items():
        p = pathlib.Path(source)
        assert p.is_file() and sha(p.read_bytes()) == expected, "驗證來源已變更：" + source
        if source.startswith("/orig/"):
            originals[source] = expected
        elif source.startswith("/src/"):
            # 驗證器__file__記錄容器絕對路徑；映射回同一工作樹來源。
            paths.add(p.relative_to("/src"))
        else:
            assert not p.is_absolute(), "未知外部來源"
            paths.add(p)
for pattern in ("over-*", "observe-over-*-20261002", "verify-over-*-20261002", "verify-over-before-*-20261002.py",
                "theme-over-v1-20261002/*", "source-before-over-v1-20261002/**/*", "redraw/OVER-00*v1-20261002*", "art-in/OVER-00.png"):
    paths.update(p for p in root.glob(pattern) if p.is_file())
for pattern in ("tools/hd/*over*", "apps/psychicwar/theme/*.go", "apps/psychicwar/translator/*.go",
                "worktrees/dosgolem/internal/**/*.go", "worktrees/dosgolem/oracle/*.go", "worktrees/dosgolem/xlate/*.go"):
    paths.update(p for p in pathlib.Path(".").glob(pattern) if p.is_file())
paths.update(pathlib.Path(p) for p in ["CONTEXT.md", "WORKLOG.md", "docs/re/038-hd-theme-feasibility.md",
                                    "docs/spec/024-hd-theme.md", "docs/worklist.json", "docs/worklist.md",
                                    "go.mod", "go.sum", "worktrees/dosgolem/go.mod", "tools/pbl.py"])
assert all(p.is_file() and not p.is_absolute() and ".." not in p.parts and (p.stat().st_uid, p.stat().st_gid) == (1000, 1000) for p in paths), "來源集合檢查不符"
out.mkdir()
files = {}
for source in sorted(paths):
    assert not source.is_absolute() and ".." not in source.parts
    st = source.stat()
    assert (st.st_uid, st.st_gid) == (1000, 1000), str(source)
    data = source.read_bytes()
    target = out / source
    target.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(source, target)
    assert sha(target.read_bytes()) == sha(data)
    files[str(source)] = {"sha256": sha(data), "size": len(data), "uid": st.st_uid, "gid": st.st_gid, "snapshot": str(target)}
doc = {"scope": "OVER #0陣亡人物限定接入及正常HD／中文與實際載回驗證，所有來源留本機",
       "files": files, "originals_sha256": originals,
       "prior_manifest": {"path": str(prior), "sha256": sha(prior.read_bytes()), "verified_snapshots": len(old["files"])},
       "limits": "保存當時文件bytes，不覆寫舊來源；歷史文件以snapshot回查。原版仍在唯讀本機輸入，不進Git／公開包。正式美術及全部sprite／GUI／封包尚未完成。"}
with manifest.open("x") as f:
    json.dump(doc, f, ensure_ascii=False, indent=2)
    f.write("\n")
print("新來源", len(files), "項", sum(row["size"] for row in files.values()), "bytes；舊", len(old["files"]), "份來源快照未變")
