"""以獨立 Python PBL 解碼核對正常原版循環；入口見研究 038 §36。"""
import argparse
import hashlib
import json
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent.parent))
import pbl


def require(ok, message):
    if not ok:
        raise ValueError(message)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def region(frame):
    require(len(frame) == 64000, "原版畫面尺寸不符")
    return b"".join(frame[y * 320 + 32:y * 320 + 56] for y in range(152, 184))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--probe", required=True, help="原版觀察輸出前綴")
    parser.add_argument("--original", default="/orig/psychic-war")
    parser.add_argument("--out", required=True, help="全新核對收據")
    args = parser.parse_args()
    prefix, out = pathlib.Path(args.probe), pathlib.Path(args.out)
    require(not out.exists(), "拒絕覆寫核對收據")
    receipt_path = pathlib.Path(str(prefix) + ".json")
    receipt = json.loads(receipt_path.read_text())
    source_path = pathlib.Path(args.original) / "ENEMY00.PBL"
    source = source_path.read_bytes()
    require(sha(source) == "8241b0ea73b1e7402b13adc434f10a5b1288f88f01fcfc3ba5710e4923fe6067", "來源 PBL 不符")
    # 獨立解碼，不使用 Go 載入器或其解碼結果。
    poses = {}
    entries = pbl.images(source)
    require(len(entries) == 30, "來源圖數不符")
    for n in (3, 4, 5):
        width, height, pixels = pbl.decode(source, entries[n][1])
        require((width, height) == (24, 32), "角色尺寸不符")
        poses[n] = bytes(pixels)
    require(receipt["seed_before"] == "86AF", "未固定種子")
    events = receipt["events"]
    sequence = [3, 4, 5, 4, 3, 4, 5, 4, 3, 4, 5, 4, 3, 4, 5, 4]
    require(len(events) == len(sequence), "有界來源數不符")
    inputs = {str(receipt_path): sha(receipt_path.read_bytes()), str(source_path): sha(source), __file__: sha(pathlib.Path(__file__).read_bytes())}
    rows = []
    previous = None
    for i, (event, target) in enumerate(zip(events, sequence)):
        paths = [pathlib.Path(f"{prefix}-event{i:02d}-{suffix}") for suffix in ("before.frame", "after.frame", "source.bin")]
        before, after, raw = [p.read_bytes() for p in paths]
        for p in paths:
            inputs[str(p)] = sha(p.read_bytes())
        require(sha(before) == event["before_sha256"] and sha(after) == event["after_sha256"] and sha(raw) == event["source_sha256"], "觀察檔案雜湊不符")
        require(len(raw) == 384, "來源長度不符")
        require(event["to_images"] == [target], "原版姿勢次序不符")
        require(region(after) == poses[target], "獨立解碼與返回姿勢不符")
        mode = 0 if i == 0 else 1
        require(event["al"] == mode, "模式不符")
        require(event["entry_regs"]["CX"] == 0x0826 and event["entry_regs"]["DX"] == 0x0304, "位置／尺寸不符")
        if previous is not None:
            require(event["from_images"] == [previous] and region(before) == poses[previous], "前一姿勢不符")
        unpacked = bytes(v for value in raw for v in (value >> 4, value & 15))
        expected = unpacked if mode == 0 else bytes(a ^ b for a, b in zip(region(before), unpacked))
        require(expected == region(after), "來源或 XOR 結果不符")
        outside = sum(a != b for j, (a, b) in enumerate(zip(before, after)) if not (32 <= j % 320 < 56 and 152 <= j // 320 < 184))
        require(outside == 0, "角色矩形外變動")
        wrong = sum(a != b for a, b in zip(unpacked, region(after))) if mode else sum(a != b for a, b in zip(poses[4], region(after)))
        require(wrong > 0, "錯誤完整圖／動作負對照無效")
        rows.append({"event": i, "from": previous, "to": target, "source_mismatch": 0, "outside_changed_pixels": outside, "negative_wrong_mode_pixels": wrong})
        previous = target
    require(receipt["end_indexed_sha256"] == "d0616fd57d1f7d078ea8abae989e6e901ff069f23890a0906e8cea845fb2e7d0", "舊正常終點不同")
    with out.open("x") as f:
        json.dump({"scope": "此正常遭遇 16 次來源與完整循環的獨立原版驗證", "inputs_sha256": inputs, "results": rows, "limits": "不證明 HD 圖面、美術品質、全部敵人或效果"}, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print("16 次原版動作、完整循環、來源及反向對照通過")


if __name__ == "__main__":
    main()
