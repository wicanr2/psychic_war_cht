"""實際pwstep載回房間完整／清除樣本，兩語言與HD兩側；研究038 §67。"""
import hashlib
import json
from pathlib import Path
import subprocess


def main():
    hd = Path("workplace/hd")
    root = hd / "room-anchor-pwstep-v1-20261003"
    assert not root.exists(), "拒絕覆寫"
    assert (hd.stat().st_uid, hd.stat().st_gid) == (1000, 1000)
    binary = hd / "pwstep-room-anchor-v1-20261003"
    exporter = hd / "export-checkpoints-v1-20261002"
    paths = [binary, exporter, Path(__file__)]
    for p in paths:
        assert p.is_file(), p
    inputs = {str(p): hashlib.sha256(p.read_bytes()).hexdigest() for p in paths}
    root.mkdir()
    rows = []
    for sample, label in ((9, "full"), (12, "clear")):
        start = hd / f"room-anchor-runtime-v1-20261003-sample{sample:02d}.state"
        for p in (start, Path(str(start) + ".xlate.json")):
            inputs[str(p)] = hashlib.sha256(p.read_bytes()).hexdigest()
        for language in ("original", "chinese"):
            for enabled in (False, True):
                name = f"{label}-{language}-{'hd' if enabled else 'off'}"
                target = root / (name + ".state")
                command = [str(binary), "-orig", "/orig/psychic-war", "-load-state", str(start),
                           "-do", "wait:1", "-scale", "3", "-text", "text" if language == "chinese" else "",
                           "-font", "font", "-theme", str(hd / "theme-room-anchor-v1-20261003") if enabled else "",
                           "-scratch", str(root / (name + "-scratch")), "-save-state", str(target),
                           "-shot", str(root / (name + ".png")), "-text-log", str(root / (name + "-text.jsonl"))]
                result = subprocess.run(command, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=45)
                (root / (name + ".log")).write_bytes(result.stdout + result.stderr)
                assert result.returncode == 0, (name, result.stderr.decode())
                assert target.is_file() and (root / (name + ".png")).is_file()
                rows.append({"label": label, "language": language, "hd": enabled, "sample": sample,
                             "state": str(target), "png": str(root / (name + ".png")), "command": command,
                             "exit_code": result.returncode})
                print(name, "完成", flush=True)
    command = [str(exporter), "-out", str(root)]
    result = subprocess.run(command, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=45)
    (root / "export.log").write_bytes(result.stdout + result.stderr)
    assert result.returncode == 0, result.stderr.decode()
    with (root / "execution.json").open("x") as f:
        json.dump({"inputs_sha256": inputs, "results": rows, "export_command": command,
                   "scope": "正常房間完整及清除state，兩語言HD兩側共八份實際pwstep輸出",
                   "limits": "保存正常state接續wait:1，未驗正式視窗、即時幀率、美術與封包"},
                  f, ensure_ascii=False, indent=2)
        f.write("\n")


if __name__ == "__main__":
    main()
