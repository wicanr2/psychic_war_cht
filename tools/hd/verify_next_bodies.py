"""正常後續遭遇的獨立來源核對；入口見研究038 §48。"""
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
    require(len(frame) == 64000, "畫面尺寸不符")
    return b"".join(frame[y * 320 + 32:y * 320 + 56] for y in range(152, 184))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", required=True)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    out = pathlib.Path(args.out)
    require(args.check or not out.exists(), "拒絕覆寫收據")
    inputs, poses = {}, {}
    for archive in range(12):
        path = pathlib.Path(f"/orig/psychic-war/ENEMY{archive:02d}.PBL")
        data = path.read_bytes()
        inputs[str(path)] = sha(data)
        entries = pbl.images(data)
        require(len(entries) == 30, "原版圖數不符")
        for image, offset, _ in entries:
            w, h, pixels = pbl.decode(data, offset)
            if (w, h) == (24, 32):
                poses[f"ENEMY{archive:02d}:{image}"] = bytes(pixels)
    rows = []
    for label, segment in (("kasuruji", "12-battle2"), ("minton", "14-minton1")):
        prefix = pathlib.Path(f"workplace/hd/next-body-{label}-v2-20261001")
        path = pathlib.Path(str(prefix) + ".json")
        doc = json.loads(path.read_text())
        inputs[str(path)] = sha(path.read_bytes())
        require(doc["route"]["name"] == segment, "正常段落不符")
        require(doc["observed_end"] == doc["unobserved_end"], "掛鉤干擾原版")
        require(1 <= len(doc["events"]) <= 16, "超過有界觀察範圍")
        for path_text, expected in doc["inputs_sha256"].items():
            source = pathlib.Path(path_text)
            require(sha(source.read_bytes()) == expected, "輸入來源變更：" + path_text)
            inputs[path_text] = expected
        results = []
        logical = None
        for i, event in enumerate(doc["events"]):
            files = [pathlib.Path(f"{prefix}-event{i:02d}-{s}") for s in ("before.frame", "after.frame", "source.bin")]
            before, after, raw = [p.read_bytes() for p in files]
            state = pathlib.Path(f"{prefix}-event{i:02d}.state")
            for p in [*files, state]:
                inputs[str(p)] = sha(p.read_bytes())
            require((sha(before), sha(after), sha(raw)) == (event["before_sha256"], event["after_sha256"], event["source_sha256"]), "事件來源改變")
            require(len(raw) == 384, "打包來源長度不符")
            from_pixels, to_pixels = region(before), region(after)
            targets = sorted(k for k, v in poses.items() if v == to_pixels)
            origins = sorted(k for k, v in poses.items() if v == from_pixels)
            require(targets == sorted(event["to_images"]), "完整返回姿勢索引不符")
            require(origins == sorted(event["from_images"]), "起始姿勢不符")
            mode = event["al"]
            require(mode in (0, 1), "未知貼圖模式")
            require(event["entry_regs"]["CX"] == 0x0826 and event["entry_regs"]["DX"] == 0x0304, "位置／尺寸不符")
            unpacked = bytes(v for b in raw for v in (b >> 4, b & 15))
            expected = unpacked if mode == 0 else bytes(a ^ b for a, b in zip(from_pixels, unpacked))
            require(to_pixels == expected, "打包來源與原版輸出不符")
            previous = logical
            if mode == 0:
                candidates = sorted(k for k, v in poses.items() if v == unpacked)
            else:
                require(logical is not None, "尚未建立完整來源的身體起點")
                candidates = sorted(k for k, v in poses.items() if bytes(a ^ b for a, b in zip(poses[logical], v)) == unpacked)
            require(len(candidates) == 1, "身體差分來源有歧義或未知")
            logical = candidates[0]
            require(not targets or targets == [logical], "完整畫面與差分來源不同")
            require(not origins or origins == [previous], "完整前姿勢與已追蹤來源不同")
            # 殘留色號只記為「身體外的XOR成分」，不把它猜成已辨識特效。
            residual_after = bytes(a ^ b for a, b in zip(to_pixels, poses[logical]))
            if mode == 1:
                residual_before = bytes(a ^ b for a, b in zip(from_pixels, poses[previous]))
                require(residual_before == residual_after, "身體差分改動了額外成分")
            outside = sum(a != b for j, (a, b) in enumerate(zip(before, after)) if not (32 <= j % 320 < 56 and 152 <= j // 320 < 184))
            require(outside == 0, "身體區外有變動")
            wrong = bytes(a ^ b for a, b in zip(from_pixels, unpacked)) if mode == 0 else unpacked
            negative = sum(a != b for a, b in zip(wrong, to_pixels))
            # 完整圖貼在全黑矩形時兩種模式結果相同；這種樣本不宣稱錯模式負對照有效。
            shifted = bytes(b for y in range(32) for b in to_pixels[y * 24 + 1:(y + 1) * 24] + b"\0")
            position_negative = sum(a != b for a, b in zip(shifted, to_pixels))
            require(position_negative > 0, "位置負對照無效")
            wrong_pose = next(v for k, v in poses.items() if k != logical and v != poses[logical])
            pose_negative = sum(a != b for a, b in zip(wrong_pose, poses[logical]))
            require(pose_negative > 0, "錯姿勢負對照無效")
            results.append({"event": i, "full_frame_from": origins, "full_frame_to": targets, "source_from": previous, "source_to": logical, "al": mode, "source_mismatch": 0, "outside_changed_pixels": outside, "residual_nonzero_pixels": sum(v != 0 for v in residual_after), "negative_wrong_mode_pixels": negative, "negative_shift_pixels": position_negative, "negative_wrong_pose_pixels": pose_negative})
        first = 0 if label == "kasuruji" else 6
        expected_ids = [first + n for n in (0, 1, 2, 1, 0, 1, 2, 1, 0, 1, 2, 1, 0, 1, 2, 1)]
        require([row["source_to"] for row in results] == [f"ENEMY00:{n}" for n in expected_ids], "原版身體往返來源與已記錄序列不同")
        end_path = pathlib.Path(str(prefix) + "-end.frame")
        baseline = pathlib.Path(f"workplace/states/{segment}.frame")
        inputs[str(end_path)] = sha(end_path.read_bytes())
        inputs[str(baseline)] = sha(baseline.read_bytes())
        require(end_path.read_bytes() == baseline.read_bytes(), "既有正常重播終點畫面不同")
        require(inputs[str(end_path)] == doc["observed_end"]["Frame"], "終點收據不同")
        rows.append({"segment": segment, "seed_before": doc["seed_before"], "actual_samples": len(results), "total_body_calls": doc["total_body_calls"], "events": results, "normal_baseline_frame_mismatch": 0, "observed_unobserved_end_equal": True})
    inputs[__file__] = sha(pathlib.Path(__file__).read_bytes())
    result = {"scope": "兩條既有正常玩家路線的有界身體來源與原版終點畫面核對", "inputs_sha256": inputs, "results": rows, "limits": "不證明HD美術、其他敵人、全部動作或效果；對照RAM屬掛鉤／無掛鉤同版原版，不等於所有既有state欄位相同"}
    if args.check:
        require(json.loads(out.read_text()) == result, "核對收據重生不同")
    else:
        with out.open("x") as f:
            json.dump(result, f, ensure_ascii=False, indent=2)
            f.write("\n")
    for row in rows:
        print(row["segment"], row["seed_before"], row["actual_samples"], [r["source_to"] for r in row["events"]])


if __name__ == "__main__":
    main()
