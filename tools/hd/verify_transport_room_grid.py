"""三張正常交通房間的原點8×8格與候選尺寸；研究038 §68。"""
import hashlib
import json
from pathlib import Path

from verify_room_grid import reference, region, valid_cells, require, pbl


def main():
    output = Path("workplace/hd/transport-room-grid-v1-20261003.json")
    require(not output.exists(), "拒絕覆寫")
    inputs = {}

    def read(path):
        path = Path(path)
        data = path.read_bytes()
        require((path.stat().st_uid, path.stat().st_gid) == (1000, 1000), "來源擁有權不符")
        inputs[str(path)] = hashlib.sha256(data).hexdigest()
        return data

    doc = json.loads(read("workplace/hd/transport-room-source-v3-20261003.json"))
    original = read("/orig/psychic-war/ROOM0.PBL")
    entries = pbl.images(original)
    rows = []
    for event in doc["events"]:
        number = int(event["SourceMatches"][0].split(":")[1])
        require(number in (0, 2, 3), "交通來源範圍不同")
        base, _ = reference(Path("/orig/psychic-war"), inputs)
        base = bytearray(base)
        w, h, pixels = pbl.decode(original, entries[number][1])
        require((w, h) == (72, 72), "原版尺寸不同")
        for y in range(h):
            base[(124 + y) * 320 + 4:(124 + y) * 320 + 76] = pixels[y * w:(y + 1) * w]
        frame = read(event["After"])
        cells = valid_cells(frame, base)
        require(len(cells) == 100, "原版房間與圖外邊界不符")
        boundary = [(x, y) for x, y in cells if x in (0, 72) or y in (120, 192)]
        require(len(boundary) == 36, "原点邊界格數不同")
        altered = bytearray(frame)
        altered[120 * 320] ^= 1
        require(region(altered, 4, 124, 72, 72) == bytes(pixels), "外緣負對照改到圖內")
        other = valid_cells(altered, base)
        require(len(other) == 99 and (0, 120) not in other, "外緣失配負對照無效")
        candidate = Path(f"workplace/hd/art-in/ROOM0-{number:02d}.png")
        read(candidate)
        cw, ch, channels, data, palette = pbl.read_png(candidate)
        require((cw, ch) == (216, 216) and channels in (3, 4), "候選尺寸或色彩格式不符")
        require(len(data) == 216 and all(len(row) == 216 * channels for row in data), "候選完整解碼不符")
        rows.append({"image": number, "frame": event["After"], "candidate": str(candidate),
                     "original_rect": [4, 124, 72, 72], "padded_rect": [0, 120, 80, 80],
                     "valid_cells": 100, "boundary_cells": 36, "negative_halo_valid_cells": 99,
                     "candidate_size": [216, 216], "candidate_channels": channels})
    require(sorted(r["image"] for r in rows) == [0, 2, 3], "三張來源未齊")
    read(__file__)
    read("tools/hd/verify_room_grid.py")
    read("tools/pbl.py")
    with output.open("x") as f:
        json.dump({"scope": "正常交通三房間的原版位置、原點8×8格及候選技術尺寸", "inputs_sha256": inputs,
                   "results": rows, "limits": "未驗HD生命週期、正式合成、中文字面、美術、GUI與完整sprite。"},
                  f, ensure_ascii=False, indent=2)
        f.write("\n")
    print("三房間各100格、36邊界；外緣負對照99，三候選216×216完整解碼通過")


if __name__ == "__main__":
    main()
