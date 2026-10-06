"""在既有工具容器執行兩份逐步工具的四檢查點；入口研究038 §64。"""
import hashlib
import json
from pathlib import Path
import subprocess


def sha(data):
    return hashlib.sha256(data).hexdigest()


def main():
    hd = Path("workplace/hd")
    root = hd / "checkpoints-runtime-v1-20261002"
    assert not root.exists(), "拒絕覆寫"
    assert (hd.stat().st_uid, hd.stat().st_gid) == (1000, 1000)
    source = hd / "checkpoints-input-v1-20261002"
    preparation = json.loads((source / "verification.json").read_text())
    inputs = dict(preparation["inputs_sha256"])
    inputs[str(source / "verification.json")] = sha((source / "verification.json").read_bytes())
    inputs[str(Path(__file__).resolve())] = sha(Path(__file__).read_bytes())
    bins = {mode: hd / ("pwstep-checkpoints-" + mode + "-v1-20261002") for mode in ["control", "current"]}
    exporter = hd / "export-checkpoints-v1-20261002"
    for p in [*bins.values(), exporter, hd / "checkpoints-control-overlay-v1-20261002.json", hd / "source-before-theme-20261001/manifest.json"]:
        assert p.is_file()
        inputs[str(p)] = sha(p.read_bytes())
    root.mkdir()
    rows = []
    for name in ["01-title", "03-protection", "05-select", "07-first-play"]:
        start = source / (name + ".state")
        side = Path(str(start) + ".xlate.json")
        for p in [start, side]:
            inputs[str(p)] = sha(p.read_bytes())
        for language in ["original", "chinese"]:
            for mode in ["control", "current"]:
                label = mode + "-" + name + "-" + language
                scratch = root / (label + "-scratch")
                target = root / (label + ".state")
                command = [str(bins[mode]), "-orig", "/orig/psychic-war", "-load-state", str(start), "-do", "wait:1", "-scale", "3", "-text", "text" if language == "chinese" else "", "-font", "font", "-scratch", str(scratch), "-save-state", str(target), "-shot", str(root / (label + ".png")), "-text-log", str(root / (label + "-text.jsonl"))]
                if mode == "current":
                    command += ["-theme", ""]
                result = subprocess.run(command, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=45)
                (root / (label + ".log")).write_bytes(result.stdout + result.stderr)
                assert result.returncode == 0, (label, result.returncode, result.stderr.decode())
                assert target.is_file() and (root / (label + ".png")).is_file()
                rows.append({"label": label, "mode": mode, "checkpoint": name, "language": language, "command": command, "state": str(target), "png": str(root / (label + ".png")), "exit_code": result.returncode})
                print(label, "完成", flush=True)
    label = "current-07-first-play-chinese-hd"
    command = [str(bins["current"]), "-orig", "/orig/psychic-war", "-load-state", str(source / "07-first-play.state"), "-do", "wait:1", "-scale", "3", "-text", "text", "-font", "font", "-scratch", str(root / (label + "-scratch")), "-save-state", str(root / (label + ".state")), "-shot", str(root / (label + ".png")), "-text-log", str(root / (label + "-text.jsonl")), "-theme", "workplace/hd/theme-over-v1-20261002"]
    result = subprocess.run(command, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=45)
    (root / (label + ".log")).write_bytes(result.stdout + result.stderr)
    assert result.returncode == 0, result.stderr.decode()
    hd_manifest = Path("workplace/hd/theme-over-v1-20261002/manifest.json")
    inputs[str(hd_manifest)] = sha(hd_manifest.read_bytes())
    for entry in json.loads(hd_manifest.read_text())["entries"]:
        p = hd_manifest.parent / entry["png"]
        inputs[str(p)] = sha(p.read_bytes())
    rows.append({"label": label, "mode": "current-hd", "checkpoint": "07-first-play", "language": "chinese", "command": command, "state": str(root / (label + ".state")), "png": str(root / (label + ".png")), "exit_code": result.returncode})
    result = subprocess.run([str(exporter), "-out", str(root)], stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=45)
    (root / "export.log").write_bytes(result.stdout + result.stderr)
    assert result.returncode == 0, result.stderr.decode()
    with (root / "execution.json").open("x") as f:
        json.dump({"inputs_sha256": inputs, "results": rows, "scope": "四正常中文state、兩語言、歷史main控制版本與現行逐步工具，各wait:1、未選主題；另初始迷宮選HD作敏感性負對照", "limits": "共用現行dosgolem／翻譯器／字型，不是整個歷史工具鏈重建；這是逐步工具，未驗真正視窗、封包或即時幀率"}, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print("17個實際逐步輸出與原版state匯出完成，仍須獨立像素核對")


if __name__ == "__main__":
    main()
