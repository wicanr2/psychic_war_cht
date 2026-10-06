"""原版陣亡人物的独立場景來源／位置／正常Enter清除核對；研究038 §51。"""
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


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", required=True)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    output = pathlib.Path(args.out)
    require(args.check or not output.exists(), "拒絕覆寫收據")
    inputs = {}

    def read(name):
        p = pathlib.Path(name)
        b = p.read_bytes()
        inputs[str(p)] = sha(b)
        return b

    prefix = "workplace/hd/over-scene-source-v2-20261002"
    doc = json.loads(read(prefix + ".json"))
    require(doc["observed_end"] == doc["control_end"], "觀察與控制組原版終點不同")
    require(doc["seed_before"] == "F95B" and doc["route"] == ["up", "up"], "正常原版輸入不符")
    require([k["Code"] for k in doc["keys_irq1"]] == [0x48, 0xc8, 0x48, 0xc8, 0x1c, 0x9c], "方向鍵／Enter送鍵不符")
    for name, expected in doc["inputs_sha256"].items():
        require(sha(read(name)) == expected, "來源變更：" + name)
    poses = {}
    for path in sorted(pathlib.Path("/orig/psychic-war").glob("*.PBL")):
        b = read(path)
        for image, offset, _ in pbl.images(b):
            w, h, px = pbl.decode(b, offset)
            if (w, h) == (64, 64):
                poses[f"{path.name}:{image}"] = bytes(px)
    target = poses["OVER.PBL:0"]
    require(inputs["/orig/psychic-war/OVER.PBL"] == "57673c27c3c0b141182a8924ac2f2de8490f43b1958369926861f0eaf02b1d1c", "原版OVER版本不符")

    def region(frame, x=128, y=48):
        require(len(frame) == 64000, "原版畫面長度不符")
        return b"".join(frame[(y + j) * 320 + x:(y + j) * 320 + x + 64] for j in range(64))

    def all_positions(frame):
        return [[x, y] for y in range(137) for x in range(257)
                if frame[y * 320 + x:y * 320 + x + 64] == target[:64] and region(frame, x, y) == target]

    scenes = doc["scene_samples"]
    require([s["full_over"] for s in scenes] == [True, True, False], "實際出現／清除樣本不同")
    rows = []
    for i, sample in enumerate(scenes):
        frame = read(sample["prefix"] + ".frame")
        read(sample["prefix"] + ".state")
        observed = region(frame)
        matched = sorted(k for k, px in poses.items() if px == observed)
        if sample["full_over"]:
            require(matched == ["OVER.PBL:0"] and all_positions(frame) == [[128, 48]], "完整來源或唯一位置不符")
        else:
            require(not matched and not all_positions(frame), "正常Enter後仍有原版OVER")
        shifted = region(frame, 129, 48)
        negative_position = sum(a != b for a, b in zip(shifted, target))
        negative_omit = sum(v != 0 for v in target)
        require(not sample["full_over"] or negative_position > 0 and negative_omit > 0, "位置／省略負對照無效")
        rows.append({"sample": sample, "matched_source": matched, "source_mismatch": 0,
                     "negative_shift_pixels": negative_position, "negative_omit_pixels": negative_omit})
    previous = read(scenes[0]["prefix"] + "-previous.frame")
    require(region(previous) != target, "首次完整出現之前已完整吻合")
    old = json.loads(read("workplace/hd/sivad-body-source-v1-20261002.json"))
    require(sha(read(scenes[1]["prefix"] + ".frame")) == old["observed_end"]["Frame"], "695M終點與前輪原版不同")
    read(prefix + "-end.state")
    require(sha(read(prefix + "-end.frame")) == doc["observed_end"]["Frame"], "新原版終點收據不同")
    read(__file__)
    read("tools/pbl.py")
    result = {"scope": "正常Sivad陣亡後OVER完整出現與正常Enter後清除的獨立原版場景核對",
              "inputs_sha256": inputs, "address_space": doc["address_space"], "seed_before": "F95B", "results": rows,
              "first_full_sample_step": doc["first_full_step"], "enter_at": doc["enter_at"], "observed_control_end_equal": True,
              "limits": "場景每100000步取樣；不宣稱精確首次寫入步數。461次一般8705摘要未捕捉64×64整張OVER呼叫，繪製路徑未知；以正常原版場景及獨立唯一PBL來源閉合內容比對契約。不是HD／中文／美術驗收。"}
    if args.check:
        require(json.loads(output.read_text()) == result, "獨立收據重生不同")
    else:
        with output.open("x") as f:
            json.dump(result, f, ensure_ascii=False, indent=2)
            f.write("\n")
    print("OVER原版完整來源唯一匹配(128,48)；兩個出現、一個Enter後清除樣本，來源不符0")


if __name__ == "__main__":
    main()
