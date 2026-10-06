"""保全四檢查點的正常中文輸入、控制來源與實際回歸；研究038 §64。"""
import hashlib
import json
from pathlib import Path
import shutil


def sha(data):
    return hashlib.sha256(data).hexdigest()


def main():
    root = Path("workplace/hd")
    target = root / "source-checkpoints-v1-20261002"
    manifest = root / "checkpoints-source-manifest-v1-20261002.json"
    assert not target.exists() and not manifest.exists(), "拒絕覆寫"
    assert (root.stat().st_uid, root.stat().st_gid) == (1000, 1000)
    receipt = root / "checkpoints-verification-v1-20261002.json"
    doc = json.loads(receipt.read_text())
    paths = {receipt, Path(__file__).resolve().relative_to(Path.cwd())}
    originals = {}
    for name, expected in doc["inputs_sha256"].items():
        p = Path(name)
        assert p.is_file() and sha(p.read_bytes()) == expected, name
        if name.startswith("/orig/"):
            originals[name] = expected
        elif name.startswith("/src/"):
            paths.add(p.relative_to("/src"))
        else:
            assert not p.is_absolute() and ".." not in p.parts, name
            paths.add(p)
    historical = json.loads((root / "source-before-theme-20261001/manifest.json").read_text())["files"]["cmd/pwstep/main.go"]
    p = Path(historical["snapshot"])
    assert sha(p.read_bytes()) == historical["sha256"]
    paths.add(p)
    for pattern in ["checkpoints-*20261002*", "pwstep-checkpoints-*-v1-20261002", "export-checkpoints-v1-20261002"]:
        for p in root.glob(pattern):
            if p.is_file():
                paths.add(p)
            elif p.is_dir():
                paths.update(f for f in p.rglob("*") if f.is_file())
    for pattern in ["apps/psychicwar/theme/*.go", "apps/psychicwar/translator/*.go", "apps/psychicwar/pbl/*.go", "worktrees/dosgolem/oracle/*.go", "worktrees/dosgolem/xlate/*.go", "worktrees/dosgolem/internal/**/*.go"]:
        paths.update(p for p in Path(".").glob(pattern) if p.is_file())
    paths.update(Path(p) for p in ["tools/hd/prepare_checkpoint_layers.go", "tools/hd/checkpoints_run.py", "tools/hd/verify_checkpoints.py", "tools/hd/export_over_frontend.go", "cmd/pwstep/main.go", "go.mod", "go.sum", "worktrees/dosgolem/go.mod", "workplace/hd/dat-runtime-v1-20261002.go.work", "CONTEXT.md", "WORKLOG.md", "docs/spec/024-hd-theme.md", "docs/re/038-hd-theme-feasibility.md", "docs/worklist.json", "docs/worklist.md", "tools/docker/go-ebiten.Dockerfile"])
    for p in Path("/orig/psychic-war").iterdir():
        if p.is_file() and (p.name == "PW.EXE" or p.suffix.upper() in [".PBL", ".BIN"]):
            originals[str(p)] = sha(p.read_bytes())
    prior = root / "dat-source-manifest-v1-20261002.json"
    old = json.loads(prior.read_text())
    for name, row in old["files"].items():
        assert sha(Path(row["snapshot"]).read_bytes()) == row["sha256"], name
    for p in paths:
        assert p.is_file() and not p.is_absolute() and ".." not in p.parts, str(p)
        assert (p.stat().st_uid, p.stat().st_gid) == (1000, 1000), str(p)
    target.mkdir()
    files = {}
    for p in sorted(paths):
        b = p.read_bytes()
        dest = target / p
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(p, dest)
        assert sha(dest.read_bytes()) == sha(b)
        files[str(p)] = {"sha256": sha(b), "size": len(b), "snapshot": str(dest), "uid": 1000, "gid": 1000}
    result = {"scope": "四正常檢查點、兩語言、歷史main控制與現行逐步工具共16 PNG；正常中文Layer與第17份選HD敏感性負對照", "files": files, "originals_sha256": originals, "prior_manifest": {"path": str(prior), "sha256": sha(prior.read_bytes()), "verified_snapshots": len(old["files"])}, "tool_versions": {"go": "1.24.13", "runner_python": "3.11.2", "independent_python": "3.13.15"}, "limits": "全部留本機。歷史main共用目前依賴，只驗HD接合與獨立原版RGB／字型合成；不是完整歷史工具鏈、GUI、封包、性能或全部sprite美術。文件以snapshot回查。"}
    with manifest.open("x") as f:
        json.dump(result, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print("保全", len(files), "項", sum(r["size"] for r in files.values()), "bytes；前批", len(old["files"]), "份來源未變")


if __name__ == "__main__":
    main()
