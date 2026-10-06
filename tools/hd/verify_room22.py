"""正常交通房間的獨立原始來源、返回畫面與座標核對；研究038 §70。"""
import hashlib
import json
import platform
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))
import pbl


def require(ok, message):
    if not ok:
        raise ValueError(message)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def crop(frame, x, y, w, h):
    require(len(frame) == 64000, "原版畫面尺寸不符")
    require(0 <= x <= 320 - w and 0 <= y <= 200 - h, "原版矩形越界")
    return b"".join(frame[j * 320 + x:j * 320 + x + w] for j in range(y, y + h))


def rle_extent(data, offset, w, h):
    """依既有格式重算讀取範圍；最後literal可能要讀下一byte才能輸出prev。"""
    need, si = w * h // 2, offset + 18
    require(si < len(data), "RLE首byte不足")
    prev = data[si]
    si += 1
    packed = bytearray()
    while len(packed) < need:
        require(si < len(data), "RLE來源不足，禁止零填充")
        cur = data[si]
        si += 1
        if cur == prev:
            require(si < len(data), "RLE重複數量不足")
            count = data[si]
            si += 1
            packed.extend(bytes([prev]) * min(count, need - len(packed)))
            if len(packed) == need:
                break
            require(si < len(data), "RLE下一段不足")
            prev = data[si]
            si += 1
        else:
            packed.append(prev)
            prev = cur
    return si - offset, bytes(v for b in packed for v in (b >> 4, b & 15))


def main():
    prefix = Path("workplace/hd/room22-source-v1-20261003")
    output = Path("workplace/hd/room22-independent-v1-20261003.json")
    require(not output.exists(), "拒絕覆寫")
    inputs = {}

    def read(path):
        path = Path(path)
        require((path.stat().st_uid, path.stat().st_gid) == (1000, 1000), "來源擁有權不符")
        data = path.read_bytes()
        inputs[str(path)] = sha(data)
        return data

    doc = json.loads(read(str(prefix) + ".json"))
    for path, expected in doc["inputs_sha256"].items():
        require(sha(read(path)) == expected, "來源已變更：" + path)
    require(doc["observed_end"] == doc["control_end"] and doc["existing_state_equal"], "原版終點不符")
    require(doc["route"]["Keys"] == ["up"], "首Up鍵序不符")
    require(len(doc["keys_irq1"]) == 2 and doc["observed_end"]["IRQ1"] == 2, "平台首Up未送達")
    archives, poses, offsets, counts = {}, {}, {}, {}
    for path in sorted(Path("/orig/psychic-war").glob("*.PBL")):
        data = read(path)
        archives[path.name] = data
        entries = pbl.images(data)
        counts[path.name] = len(entries)
        for number, offset, size in entries:
            w, h, pixels = pbl.decode(data, offset)
            key = f"{path.name}:{number}"
            poses[key] = (w, h, bytes(pixels))
            offsets[key] = (offset, size)
    require(counts == doc["archives"] and len(counts) == 26 and len(poses) == 537, "圖庫清冊不符")
    rows, room_rows = [], []
    for event in doc["events"]:
        raw, before, after = (read(event[k]) for k in ("Source", "Before", "After"))
        x, y, w, h = (event[k] for k in ("X", "Y", "W", "H"))
        if event["SourceKind"] == "rle-copy":
            rw, rh, pixels = pbl.decode(raw, 0)
            require((rw, rh) == (w, h), "RLE尺寸不符")
            pixels = bytes(pixels)
        else:
            require(event["SourceKind"] == "packed" and event["AX"] & 255 in (0, 1), "未知貼圖模式")
            pixels = bytes(v for b in raw for v in (b >> 4, b & 15))
            require(len(pixels) == w * h, "打包來源尺寸不符")
        matches = sorted(k for k, v in poses.items() if v == (w, h, pixels))
        require(matches == event["SourceMatches"], "來源解碼器結果不同")
        previous = crop(before, x, y, w, h)
        expected = bytes(a ^ b for a, b in zip(previous, pixels, strict=True)) if event["SourceKind"] == "packed" and event["AX"] & 255 == 1 else pixels
        require(crop(after, x, y, w, h) == expected, "原版完整返回畫面不符")
        outside = sum(a != b for i, (a, b) in enumerate(zip(before, after, strict=True))
                      if not (x <= i % 320 < x + w and y <= i // 320 < y + h))
        require(outside == 0, "貼圖矩形外被改動")
        full_before = sorted(k for k, v in poses.items() if v == (w, h, previous))
        full_after = sorted(k for k, v in poses.items() if v == (w, h, crop(after, x, y, w, h)))
        require(full_before == event["FullBefore"] and full_after == event["FullAfter"], "完整來源辨識不同")
        row = {"source_kind": event["SourceKind"], "sources": matches, "full_before": full_before,
               "full_after": full_after, "entry_step": event["Entry"], "return_step": event["Return"],
               "at": [x, y], "size": [w, h], "source_ds_bx": [event["DS"], event["BX"]],
               "source_linear_runtime": event["DS"] * 16 + event["BX"],
               "original_pixels_mismatch": 0, "outside_changed_pixels": 0}
        if any(k.startswith("ROOM") for k in matches):
            require(len(matches) == 1 and event["SourceKind"] == "rle-copy", "房間來源不唯一或非完整RLE")
            key = matches[0]
            name = key.split(":")[0]
            offset, size = offsets[key]
            extent, strict_pixels = rle_extent(archives[name], offset, w, h)
            require(strict_pixels == pixels, "禁止零填充的RLE解碼不符")
            require(len(raw) == extent and raw == archives[name][offset:offset + extent], "房間RLE原始檔bytes不符")
            require((x, y, w, h) == (4, 124, 72, 72), "房間原版位置或範圍不同")
            shift = sum(a != b for a, b in zip(crop(after, x + 1, y, w, h), pixels, strict=True))
            wrong_key = "ROOM0.PBL:8"
            wrong = sum(a != b for a, b in zip(poses[wrong_key][2], pixels, strict=True))
            require(shift > 0 and wrong > 0, "房間位移或錯圖號負對照無效")
            changed = bytearray(pixels)
            changed[0] ^= 1
            require(sum(a != b for a, b in zip(changed, pixels, strict=True)) == 1, "單像素負對照無效")
            require(event["State"], "房間返回state缺少")
            read(event["State"])
            row.update({"file_offset": offset, "archive_image_bytes": size, "rle_decoder_bytes": extent,
                        "lookahead_outside_image_bytes": max(0, extent - size), "rle_file_bytes_mismatch": 0,
                        "negative_shift_pixels": shift, "negative_wrong_image_pixels": wrong,
                        "negative_single_pixel": 1, "state": event["State"]})
            room_rows.append(row)
        rows.append(row)
    require(room_rows and len(room_rows) == 1 and room_rows[0]["sources"] == ["ROOM0.PBL:22"], "正常首Up未取得唯一ROOM0 #22完整來源")
    end = read(str(prefix) + "-end.frame")
    require(end == read("workplace/hd/transport-room-runtime-v1-20261003-sample20.frame") and sha(end) == doc["observed_end"]["Frame"], "交通終點畫面不符")
    require(crop(end, 4, 124, 72, 72) == poses["ROOM0.PBL:22"][2], "正常交通終點原版房間不符")
    read(str(prefix) + "-end.state")
    saved = []
    for name, key in (("16-sivad", "ROOM0.PBL:2"), ("19-zellwal", "ROOM0.PBL:2"),
                      ("17-saved2", "ROOM0.PBL:22"), ("18-loaded2", "ROOM0.PBL:22")):
        frame = read("workplace/states/" + name + ".frame")
        read("workplace/states/" + name + ".state")
        require(crop(frame, 4, 124, 72, 72) == poses[key][2], "既有完整保存畫面不符")
        saved.append({"checkpoint": name, "source": key, "at": [4, 124], "size": [72, 72], "mismatch": 0,
                      "level": "完整保存畫面匹配，單獨不證明正常繪製來源"})
    execution = json.loads(read("workplace/hd/room22-source-execution-v1-20261003.json"))
    require(execution["returncode"] == 0, "來源執行失敗")
    read(execution["log"])
    binary = read("workplace/hd/observe-room22-v1-20261003.bin")
    require(sha(binary) == execution["binary_sha256"], "實際二進位不同")
    for path in (__file__, "tools/pbl.py", "workplace/hd/room22-source-overlay-v1-20261003.json"):
        read(path)
    with output.open("x") as f:
        json.dump({"scope": "正常離開降落平台首Up的原版RLE來源、完整返回畫面與既有保存畫面匹配",
                   "python_version": platform.python_version(), "inputs_sha256": inputs, "results": rows,
                   "rooms": room_rows, "saved_frames": saved, "seed_before": doc["seed_before"],
                   "other_rle_calls_unknown": doc["other_rle_calls_unknown"],
                   "address_space": doc["address_space"], "archive_count": 26, "image_count": 537,
                   "limits": "只核對原版房間矩形的來源，其他RLE模式未知；未接入HD，其他房間／sprite來源未證實、中文、美術、GUI或完整sprite。"},
                  f, ensure_ascii=False, indent=2)
        f.write("\n")
    print(json.dumps({"events": len(rows), "rooms": room_rows, "saved_frame_matches": len(saved)}, ensure_ascii=False))


if __name__ == "__main__":
    main()
