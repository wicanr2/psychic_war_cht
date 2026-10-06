"""Sivad 原版身體來源獨立核對；研究038 §50，只讀既有觀察資料。"""
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
    require(len(frame) == 64000, "原版畫面長度不符")
    return b"".join(frame[y * 320 + 32:y * 320 + 56] for y in range(152, 184))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", required=True)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    out = pathlib.Path(args.out)
    require(args.check or not out.exists(), "拒絕覆寫收據")
    inputs = {}

    def read(name):
        path = pathlib.Path(name)
        data = path.read_bytes()
        inputs[str(path)] = sha(data)
        return data

    prefix = "workplace/hd/sivad-body-source-v1-20261002"
    original = read(prefix + ".json")
    require(sha(original) == "79987c386258696348632f5638448ac21045c7ca3b7a154c423c1d5a98a020c3", "原始觀察收據變更")
    doc = json.loads(original)
    require(doc["observed_end"] == doc["control_end"], "觀察干擾原版終點")
    require(doc["route"] == ["up", "up"] and doc["seed_before"] == "F95B", "正常路線或種子不符")
    require(doc["battle_entries"] == doc["battle_returns"] == 1, "原版戰鬥進出不符")
    require(doc["end_location"]["player_hp"] == 0, "原版終點狀態不符")
    require(len(doc["events"]) == doc["total_body_calls"] == 8, "身體樣本數不符")
    for name, expected in doc["inputs_sha256"].items():
        require(sha(read(name)) == expected, "原始輸入變更：" + name)
    poses = {}
    for archive in range(12):
        data = read(f"/orig/psychic-war/ENEMY{archive:02d}.PBL")
        entries = pbl.images(data)
        require(len(entries) == 30, "原版PBL圖數不符")
        for image, offset, _ in entries:
            w, h, pixels = pbl.decode(data, offset)
            if image < 15 and (w, h) == (24, 32):
                poses[f"ENEMY{archive:02d}:{image}"] = bytes(pixels)
    rows, logical = [], None
    for i, event in enumerate(doc["events"]):
        before, after, raw = [read(event[k]) for k in ("Before", "After", "Source")]
        read(event["State"])
        require(event["Entry"] < event["Return"], "貼圖時間順序錯誤")
        require((event["CX"], event["DX"], event["W"], event["H"]) == (0x0826, 0x0304, 24, 32), "原版位置或尺寸不符")
        require(len(raw) == 384, "原始4bpp來源長度不符")
        a, b = region(before), region(after)
        origins = sorted(k for k, v in poses.items() if v == a)
        targets = sorted(k for k, v in poses.items() if v == b)
        require(origins == event["FullBefore"] and targets == event["FullAfter"], "觀察器完整姿勢與獨立PBL不符")
        mode = event["AX"] & 255
        require(mode in (0, 1), "未知原版貼圖模式")
        pixels = bytes(v for byte in raw for v in (byte >> 4, byte & 15))
        expected = pixels if mode == 0 else bytes(x ^ y for x, y in zip(a, pixels))
        require(expected == b, "來源與原版前後畫面不符")
        previous = logical
        if mode == 0:
            candidates = sorted(k for k, v in poses.items() if v == pixels)
        else:
            require(previous is not None, "未建立完整來源起點")
            candidates = sorted(k for k, v in poses.items() if bytes(x ^ y for x, y in zip(poses[previous], v)) == pixels)
        require(len(candidates) == 1, "差分來源未知或歧義")
        logical = candidates[0]
        require(not targets or targets == [logical], "完整姿勢與差分來源不同")
        require(not origins or origins == [previous], "完整前姿勢與來源追蹤不同")
        residual = bytes(x ^ y for x, y in zip(b, poses[logical]))
        if mode == 1:
            require(bytes(x ^ y for x, y in zip(a, poses[previous])) == residual, "額外XOR成分被身體差分改動")
        outside = sum(x != y for j, (x, y) in enumerate(zip(before, after)) if not (32 <= j % 320 < 56 and 152 <= j // 320 < 184))
        require(outside == 0, "貼圖矩形外有變動")
        wrong = bytes(x ^ y for x, y in zip(a, pixels)) if mode == 0 else pixels
        negative_mode = sum(x != y for x, y in zip(wrong, b))
        if mode == 1:
            require(negative_mode > 0, "差分錯模式負對照無效")
        shifted = b"".join(b[y * 24 + 1:(y + 1) * 24] + b"\0" for y in range(32))
        negative_shift = sum(x != y for x, y in zip(shifted, b))
        other = next(v for k, v in poses.items() if k != logical and v != poses[logical])
        negative_pose = sum(x != y for x, y in zip(other, poses[logical]))
        require(negative_shift > 0 and negative_pose > 0, "位置或來源負對照無效")
        rows.append({"event": i, "entry": event["Entry"], "return": event["Return"], "source_from": previous, "source_to": logical,
                     "full_frame_from": origins, "full_frame_to": targets, "al": mode, "raw_sha256": sha(raw), "source_mismatch": 0,
                     "outside_changed_pixels": outside, "residual_nonzero_pixels": sum(v != 0 for v in residual),
                     "negative_wrong_mode_pixels": negative_mode, "negative_shift_pixels": negative_shift, "negative_wrong_pose_pixels": negative_pose})
    require([r["source_to"] for r in rows] == [f"ENEMY01:{n}" for n in (6, 7, 8, 7, 6, 7, 8, 7)], "實際來源序列不同")
    partials = []
    require(len(doc["samples"]) == 16, "完整與中途樣本數不符")
    for sample in doc["samples"]:
        frame = read(sample["Prefix"] + ".frame")
        read(sample["Prefix"] + ".state")
        require(len(frame) == 64000, "樣本畫面長度不符")
        event = doc["events"][sample["Event"]]
        if sample["Kind"] == "partial":
            require(sample["Step"] - event["Entry"] in (1, 512), "中途步數不符")
            require(event["Entry"] < sample["Step"] < event["Return"], "中途樣本不在貼圖內")
            partials.append(sample)
        else:
            require(sample["Kind"] == "complete" and sample["Step"] == event["Return"], "完整樣本時間不符")
            require(frame == read(event["After"]), "完整樣本來源不同")
    require(len(partials) == 8, "中途樣本數不同")
    read(prefix + "-end.state")
    require(sha(read(prefix + "-end.frame")) == doc["observed_end"]["Frame"], "終點畫面收據不符")
    read(__file__)
    read("tools/pbl.py")
    result = {"scope": "Sivad正常方向鍵的8次身體原版來源與前後畫面獨立核對", "inputs_sha256": inputs,
              "seed_before": doc["seed_before"], "seed_method": doc["seed_method"], "address_space": doc["address_space"],
              "events": rows, "partial_samples": partials, "observed_control_end_equal": True,
              "battle_entries": 1, "battle_returns": 1, "end_player_hp": 0,
              "limits": "差分來源已證實；額外XOR成分用途未知。中途原版state僅保全，不代表HD中途輸出驗收；不證明HD／中文／美術、其他角色或全部效果。首筆黑底完整圖錯模式差0，不宣稱該負對照有效。"}
    if args.check:
        require(json.loads(out.read_text()) == result, "同資料重生收據不同")
    else:
        with out.open("x") as f:
            json.dump(result, f, ensure_ascii=False, indent=2)
            f.write("\n")
    print("Sivad", len(rows), "身體來源", [r["source_to"] for r in rows], "來源／矩形外不符0；8中途來源保全")


if __name__ == "__main__":
    main()
