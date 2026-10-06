"""治療房間原點8×8格的獨立背景與邊界核對；研究038 §66。"""
import hashlib
import json
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent.parent))
import pbl


def require(ok, message):
    if not ok:
        raise ValueError(message)


def region(frame, x, y, w, h):
    return b"".join(frame[(y + yy) * 320 + x:(y + yy) * 320 + x + w] for yy in range(h))


def reference(orig, inputs=None):
    base = bytearray(64000)
    for name in ("SCREEN.PBL", "MENU.PBL"):
        data = (orig / name).read_bytes()
        if inputs is not None:
            inputs[str(orig / name)] = hashlib.sha256(data).hexdigest()
        for image, offset, _ in pbl.images(data):
            w, h, pixels = pbl.decode(data, offset)
            x, y = (0, image * 40) if name == "SCREEN.PBL" else (160, 4)
            for yy in range(h):
                base[(y + yy) * 320 + x:(y + yy) * 320 + x + w] = pixels[yy * w:(yy + 1) * w]
    data = (orig / "ROOM0.PBL").read_bytes()
    require(hashlib.sha256(data).hexdigest() == "2b2f58c9b716a54bf826dbc9c90237a32458fe49abab52869d5353bbff34d111", "ROOM0版本不符")
    if inputs is not None:
        inputs[str(orig / "ROOM0.PBL")] = hashlib.sha256(data).hexdigest()
    entries = pbl.images(data)
    require(len(entries) == 31, "ROOM0圖數不符")
    w, h, pixels = pbl.decode(data, entries[8][1])
    require((w, h) == (72, 72), "治療房間尺寸不符")
    for yy in range(72):
        base[(124 + yy) * 320 + 4:(124 + yy) * 320 + 76] = pixels[yy * 72:(yy + 1) * 72]
    return bytes(base), bytes(pixels)


def valid_cells(frame, base):
    require(len(frame) == len(base) == 64000, "原版畫面尺寸不符")
    return [(x, y) for y in range(120, 200, 8) for x in range(0, 80, 8)
            if region(frame, x, y, 8, 8) == region(base, x, y, 8, 8)]


def main():
    out = pathlib.Path("workplace/hd/room-grid-v1-20261003.json")
    require(not out.exists(), "拒絕覆寫收據")
    inputs = {}
    base, original = reference(pathlib.Path("/orig/psychic-war"), inputs)
    rows = []
    for suffix in ("event000-after.frame", "end.frame"):
        path = pathlib.Path("workplace/hd/room-normal-v2-20261002-" + suffix)
        frame = path.read_bytes()
        inputs[str(path)] = hashlib.sha256(frame).hexdigest()
        cells = valid_cells(frame, base)
        require(len(cells) == 100 and region(frame, 4, 124, 72, 72) == original, "正常房間或邊界基準不符")
        boundary = [(x, y) for x, y in cells if x in (0, 72) or y in (120, 192)]
        require(len(boundary) == 36, "邊界格數不符")
        altered = bytearray(frame)
        altered[120 * 320] ^= 1
        require(region(altered, 4, 124, 72, 72) == original, "外緣負對照改到完整圖")
        other = valid_cells(altered, base)
        require(len(other) == 99 and (0, 120) not in other, "原點格的外緣失配未被辨識")
        rows.append({"input": str(path), "padded_rect": [0, 120, 80, 80], "original_rect": [4, 124, 72, 72], "rows": 10, "columns": 10, "valid_cells": 100, "boundary_cells": 36, "negative_halo_change_valid_cells": 99, "source_position_unchanged": True})
    inputs[__file__] = hashlib.sha256(pathlib.Path(__file__).read_bytes()).hexdigest()
    with out.open("x", encoding="utf-8") as f:
        json.dump({"scope": "原版房間與SCREEN／MENU不可變基準的原點8×8邊界核對", "inputs_sha256": inputs, "results": rows, "limits": "只證明原版基準與格網；未驗HD合成、美術、離房生命週期或GUI"}, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print("兩張原版房間各100格基準相同，36個邊界格；外緣一像素負對照留下99格")


if __name__ == "__main__":
    main()
