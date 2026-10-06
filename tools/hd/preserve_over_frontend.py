"""保全OVER正式前端、失敗診斷及獨立驗證來源；研究038 §52，全數留本機。"""
import hashlib
import json
from pathlib import Path
import shutil


def sha(data):
    return hashlib.sha256(data).hexdigest()


def main():
    root = Path("workplace/hd")
    target = root / "source-over-frontend-v1-20261002"
    manifest = root / "over-frontend-source-manifest-v1-20261002.json"
    assert not target.exists() and not manifest.exists(), "拒絕覆寫"
    assert (root.stat().st_uid, root.stat().st_gid) == (1000, 1000)
    prior = root / "over-source-manifest-v1-20261002.json"
    old = json.loads(prior.read_text())
    for source, row in old["files"].items():
        saved = Path(row.get("snapshot") or source)
        assert saved.is_file() and sha(saved.read_bytes()) == row["sha256"], str(saved)
    receipt = root / "over-frontend-verification-v4-20261002.json"
    result = json.loads(receipt.read_text())
    paths = {prior, receipt, Path("tools/hd/preserve_over_frontend.py")}
    originals = {}
    for name, expected in result["inputs_sha256"].items():
        source = Path(name)
        assert source.is_file() and sha(source.read_bytes()) == expected, name
        if name.startswith("/orig/"):
            originals[name] = expected
        elif name.startswith("/src/"):
            paths.add(source.relative_to("/src"))
        else:
            assert not source.is_absolute(), name
            paths.add(source)
    for pattern in (
        "over-frontend-*20261002*", "over-pwstep-*20261002*",
        "over-draw-*20261002*", "over-ui-*20261002*", "over-palette-*20261002*",
        "observe-over-frontend-*20261002*", "psychicwar-over-ui-*20261002",
        "pwstep-over-ui-*20261002", "export-over-frontend-*20261002",
        "theme-over-v1-20261002",
    ):
        for source in root.glob(pattern):
            if source.is_file():
                paths.add(source)
            elif source.is_dir():
                paths.update(p for p in source.rglob("*") if p.is_file())
    for pattern in (
        "tools/hd/*over*", "apps/psychicwar/theme/*.go",
        "apps/psychicwar/translator/*.go", "worktrees/dosgolem/xlate/*.go",
        "worktrees/dosgolem/oracle/*.go", "worktrees/dosgolem/internal/**/*.go",
    ):
        paths.update(p for p in Path(".").glob(pattern) if p.is_file())
    paths.update(Path(p) for p in (
        "CONTEXT.md", "WORKLOG.md", "docs/re/038-hd-theme-feasibility.md",
        "docs/spec/024-hd-theme.md", "docs/worklist.json", "docs/worklist.md",
        "worktrees/dosgolem/docs/spec/202-translation-overlay.md",
        "cmd/psychicwar/main.go", "cmd/pwstep/main.go",
        "apps/psychicwar/shift_input.go", "apps/psychicwar/shift_input_test.go",
        "go.mod", "go.sum", "worktrees/dosgolem/go.mod",
        "tools/pbl.py", "tools/docker/go-ebiten.Dockerfile",
    ))
    for p in paths:
        assert p.is_file() and not p.is_absolute() and ".." not in p.parts, str(p)
        s = p.stat()
        assert (s.st_uid, s.st_gid) == (1000, 1000), str(p)
    target.mkdir()
    files = {}
    for source in sorted(paths):
        data = source.read_bytes()
        dest = target / source
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(source, dest)
        assert sha(dest.read_bytes()) == sha(data)
        files[str(source)] = {
            "sha256": sha(data), "size": len(data), "snapshot": str(dest),
            "uid": 1000, "gid": 1000,
        }
    doc = {
        "scope": "OVER正式Linux視窗及pwstep有限驗證，含失敗版本與全部游標相位",
        "files": files, "originals_sha256": originals,
        "prior_manifest": {"path": str(prior), "sha256": sha(prior.read_bytes()),
                           "verified_snapshots": len(old["files"])},
        "limits": "全部留本機，不覆寫舊來源。歷史文件用snapshot回查；本批未驗DAT、正式封包、美術或全部sprite。",
    }
    with manifest.open("x") as f:
        json.dump(doc, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print("新來源", len(files), "項", sum(r["size"] for r in files.values()),
          "bytes；舊", len(old["files"]), "份快照未變")


if __name__ == "__main__":
    main()
