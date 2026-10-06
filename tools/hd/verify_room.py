"""正常治療房間的獨立來源與座標核對；入口研究038 §65。"""
import hashlib
import json
import pathlib
import platform
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent.parent))
import pbl


def require(ok, message):
    if not ok:
        raise ValueError(message)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def crop(frame, x, y, w, h):
    require(len(frame) == 64000, "原版畫面尺寸不符")
    require(0 <= x <= 320 - w and 0 <= y <= 200 - h, "矩形越界")
    return b"".join(frame[j * 320 + x:j * 320 + x + w] for j in range(y, y + h))


def main():
    prefix = pathlib.Path("workplace/hd/room-normal-v2-20261002")
    out = pathlib.Path("workplace/hd/room-normal-independent-v1-20261002.json")
    require(not out.exists(), "拒絕覆寫獨立收據")
    inputs = {}

    def read(path):
        path = pathlib.Path(path)
        data = path.read_bytes()
        inputs[str(path)] = sha(data)
        require((path.stat().st_uid, path.stat().st_gid) == (1000, 1000), "擁有權不符：" + str(path))
        return data

    doc = json.loads(read(str(prefix) + ".json"))
    for path, expected in doc["inputs_sha256"].items():
        require(sha(read(path)) == expected, "原版或工具來源變更：" + path)
    poses, archive_counts, offsets, archive_bytes = {}, {}, {}, {}
    for path in sorted(pathlib.Path("/orig/psychic-war").glob("*.PBL")):
        data = read(path)
        archive_bytes[path.name] = data
        entries = pbl.images(data)
        archive_counts[path.name] = len(entries)
        for image, offset, size in entries:
            w, h, pixels = pbl.decode(data, offset)
            key = f"{path.name}:{image}"
            poses[key] = (w, h, bytes(pixels))
            offsets[key] = (offset, size)
    require(len(archive_counts) == 26 and len(poses) == 537, "原版圖庫數量不符")
    require(archive_counts == doc["archives"] and len(poses) == doc["pose_count"], "兩解碼器圖庫清冊不符")
    require(doc["observed_end"] == doc["control_end"] and doc["existing_state_equal"], "原版終點不符")
    require(doc["seed_before"] == "A48C", "固定起點種子不符")
    require(len(doc["keys_irq1"]) == 18 and doc["observed_end"]["IRQ1"] == 18, "九鍵IRQ1不符")
    require(doc["route"]["Keys"] == ["down", "up", "up", "right", "up", "up", "right", "up", "enter"], "正常按鍵路線不符")
    require(len(doc["events"]) == 1, "本條路線實際貼圖數變更")
    rows = []
    for event in doc["events"]:
        raw, before, after = (read(event[key]) for key in ("Source", "Before", "After"))
        x, y, w, h = (event[key] for key in ("X", "Y", "W", "H"))
        require(event["SourceKind"] == "rle-copy", "本條路線來源不是串流RLE")
        rw, rh, pixels = pbl.decode(raw, 0)
        require((rw, rh) == (w, h), "RLE尺寸不符")
        pixels = bytes(pixels)
        matches = sorted(key for key, pose in poses.items() if pose == (w, h, pixels))
        require(matches == ["ROOM0.PBL:8"] == event["SourceMatches"], "來源不唯一或不是原版房間8")
        key = matches[0]
        offset, size = offsets[key]
        require(raw == archive_bytes["ROOM0.PBL"][offset:offset + len(raw)] and len(raw) <= size, "執行期RLE與原始檔案bytes不符")
        require((x, y, w, h) == (4, 124, 72, 72) and event["CX"] == 0x011f, "原版房間座標不符")
        require((event["DS"], event["BX"]) == (0x161, 0x92be), "原始來源DS:BX不符")
        require(crop(after, x, y, w, h) == pixels, "返回畫面與獨立解碼不符")
        previous = crop(before, x, y, w, h)
        full_before = sorted(key for key, pose in poses.items() if pose == (w, h, previous))
        require(full_before == event["FullBefore"] and matches == event["FullAfter"], "完整畫面來源匹配不符")
        outside = sum(a != b for j, (a, b) in enumerate(zip(before, after)) if not (x <= j % 320 < x + w and y <= j // 320 < y + h))
        require(outside == 0, "房間矩形以外被改動")
        shifted = crop(after, x + 1, y, w, h)
        negative_shift = sum(a != b for a, b in zip(shifted, pixels))
        wrong = poses["ROOM0.PBL:7"][2]
        require(len(wrong) == len(pixels), "錯圖號負對照尺寸不符")
        negative_image = sum(a != b for a, b in zip(wrong, pixels))
        require(negative_shift > 0 and negative_image > 0, "座標或圖號負對照無效")
        changed = bytearray(pixels)
        changed[0] ^= 1
        require(sum(a != b for a, b in zip(changed, pixels)) == 1, "單像素比較負對照無效")
        read(event["State"])
        rows.append({"source": key, "file_offset": offset, "compressed_bytes_consumed": len(raw), "entry_step": event["Entry"], "return_step": event["Return"], "at": [x, y], "size": [w, h], "source_linear_runtime": event["DS"] * 16 + event["BX"], "rle_file_bytes_mismatch": 0, "original_pixels_mismatch": 0, "outside_changed_pixels": outside, "negative_shift_pixels": negative_shift, "negative_wrong_image_pixels": negative_image, "negative_single_pixel": 1})
    end = read(str(prefix) + "-end.frame")
    require(end == read("workplace/states/13-healed.frame"), "獨立原版終點畫面不符")
    require(sha(end) == doc["observed_end"]["Frame"], "終點畫面指紋不符")
    # 房間來源在完整繪製後出現；治療結束仍在原位置，並未縮小或平移。
    require(crop(end, 4, 124, 72, 72) == poses["ROOM0.PBL:8"][2], "治療終點完整房間不符")
    read(str(prefix) + "-end.state")
    read("workplace/hd/observe-room-v2-20261002.bin")
    read(__file__)
    result = {"scope": "既有正常九鍵治療路線的房間RLE原始來源、座標與完整返回畫面", "python_version": platform.python_version(), "inputs_sha256": inputs, "results": rows, "archive_count": 26, "image_count": 537, "normal_end_frame_mismatch": 0, "limits": "不證明其他房間、HD接入、美術、中文字面、GUI或動畫；CPU／RAM一致來自原版探針三方比較，獨立核對只重算PBL來源及保存畫面"}
    with out.open("x", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print(json.dumps(rows, ensure_ascii=False))


if __name__ == "__main__":
    main()
