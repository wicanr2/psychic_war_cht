"""獨立核對正常戰鬥的來源、方向與 XOR 場景；研究 038 §37 的原版證據工具。"""
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


def region(frame, rect):
    require(len(frame) == 64000, "色號畫面尺寸不符")
    x, y, w, h = rect
    return b"".join(frame[row * 320 + x:row * 320 + x + w] for row in range(y, y + h))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--probe", required=True, help="原版輸出前綴")
    parser.add_argument("--original", default="/orig/psychic-war")
    parser.add_argument("--out", required=True, help="新的驗證收據")
    args = parser.parse_args()
    prefix, out, original = pathlib.Path(args.probe), pathlib.Path(args.out), pathlib.Path(args.original)
    require(not out.exists(), "拒絕覆寫收據")
    rp = pathlib.Path(str(prefix) + ".json")
    receipt = json.loads(rp.read_text())
    inputs = {str(rp): sha(rp.read_bytes()), __file__: sha(pathlib.Path(__file__).read_bytes())}
    assets = {}
    for name, count in (("BEAM", 12), ("FIGHT", 12), ("ENEMY00", 30), ("ALLY", 31), ("SCREEN", 5), ("MENU", 1)):
        path = original / (name + ".PBL")
        data = path.read_bytes()
        inputs[str(path)] = sha(data)
        entries = pbl.images(data)
        require(len(entries) == count, "來源圖數不符：" + name)
        for n, off, length in entries:
            w, h, pixels = pbl.decode(data, off)
            assets[(name, n)] = (w, h, bytes(pixels))
    require(receipt["seed_before"] == "5447" and not receipt["unknown_source_counts"], "未固定種子或來源尚未完全辨識")
    events = receipt["events"]
    require(len(events) == sum(receipt["all_geometry_counts"].values()) == 975, "實際貼圖數不符")
    executable = pathlib.Path("workplace/ida/PW_UNP.EXE")
    exe = executable.read_bytes()
    inputs[str(executable)] = sha(exe)
    require(inputs[str(executable)] == "fd5115b91f014c47a1fb1a6e2ecfd645e293264fe614a4b350e92637cad2fbd9", "解壓後執行檔不符")
    mask_offset = int.from_bytes(exe[8:10], "little") * 16 + int.from_bytes(exe[22:24], "little") * 16 + 0x4E36
    stencil = exe[mask_offset:mask_offset + 32]
    require(len(stencil) == 32, "遮罩長度不符")
    mask_phases = set()
    canvas = bytearray(64000)

    def paint(dst, label, x, y, xor=False):
        w, h, pixels = assets[label]
        for row in range(h):
            start = (y + row) * 320 + x
            for col in range(w):
                value = pixels[row * w + col]
                if xor:
                    dst[start + col] ^= value
                else:
                    dst[start + col] = value

    for n in range(5):
        paint(canvas, ("SCREEN", n), 0, n * 40)
    paint(canvas, ("MENU", 0), 160, 4)
    paint(canvas, ("ALLY", 0), 264, 152)
    static = bytes(canvas)
    # 固定遭遇起點的完整原圖確認 #4；後續由來源推進，不從畫面猜選。
    phases = {("ENEMY00", 32, 152, 24, 32): 4}
    rows = []
    domain = (32, 144, 256, 40)

    def model():
        result = bytearray(static)
        actor = phases.get(("ENEMY00", 32, 152, 24, 32))
        if actor is not None:
            paint(result, ("ENEMY00", actor), 32, 152)
        for (name, x, y, w, h), image in phases.items():
            if (x, y, w, h) == (32, 152, 24, 32) or image is None:
                continue
            paint(result, (name, image), x, y, xor=True)
        for x, y, w, h in mask_phases:
            for row in range(16):
                for col in range(16):
                    if stencil[row * 2 + col // 8] & (0x80 >> (col % 8)):
                        result[(y + row) * 320 + x + col] ^= 10
        return bytes(result)

    timeline = [(e["entry_step"], "event", i, e) for i, e in enumerate(events)]
    timeline += [(e["entry_step"], "mask", i, e) for i, e in enumerate(receipt["mask_events"])]
    previous_return = 0
    for step, kind, i, event in sorted(timeline):
        require(step > previous_return, "輸出呼叫重疊或亂序")
        previous_return = event["return_step"]
        if kind == "mask":
            paths = [pathlib.Path(f"{prefix}-mask{i:03d}-{suffix}") for suffix in ("before.frame", "after.frame", "source.bin")]
            before, after, raw = [p.read_bytes() for p in paths]
            for p in paths:
                inputs[str(p)] = sha(p.read_bytes())
            require(sha(before) == event["before_sha256"] and sha(after) == event["after_sha256"] and sha(raw) == event["source_sha256"], "遮罩輸出雜湊不同")
            require(raw == stencil and event["xor_color"] == 10, "遮罩來源或顏色不同")
            rect = tuple(event["rect"])
            x, y, w, h = rect
            require((w, h) == (16, 16), "遮罩尺寸不符")
            expected = bytearray(before)
            for row in range(16):
                for col in range(16):
                    if raw[row * 2 + col // 8] & (0x80 >> (col % 8)):
                        expected[(y + row) * 320 + x + col] ^= 10
            require(bytes(expected) == after, "獨立位元遮罩與原版畫面不符")
            require(region(model(), domain) == region(before, domain), f"遮罩 {i} 之前場景模型不符")
            if rect in mask_phases:
                mask_phases.remove(rect)
            else:
                mask_phases.add(rect)
            require(region(model(), domain) == region(after, domain), f"遮罩 {i} 之後場景模型不符")
            omit = sum(a != b for a, b in zip(before, after))
            require(omit == sum(v.bit_count() for v in stencil) > 0, "遮罩省略負對照無效")
            rows.append({"kind": kind, "event": i, "rect": rect, "source_mismatch": 0, "scene_model_mismatch": 0, "negative_omit_pixels": omit})
            continue
        paths = [pathlib.Path(f"{prefix}-event{i:03d}-{suffix}") for suffix in ("before.frame", "after.frame", "source.bin")]
        before, after, raw = [p.read_bytes() for p in paths]
        for p in paths:
            inputs[str(p)] = sha(p.read_bytes())
        require(sha(before) == event["before_sha256"] and sha(after) == event["after_sha256"] and sha(raw) == event["source_sha256"], "原始輸出雜湊不同")
        rect = tuple(event["rect"])
        x, y, w, h = rect
        require(event["al"] == 1 and len(raw) == w * h // 2, "模式／來源長度不符")
        previous_region, next_region = region(before, rect), region(after, rect)
        unpacked = bytes(v for value in raw for v in (value >> 4, value & 15))
        require(bytes(a ^ b for a, b in zip(previous_region, unpacked)) == next_region, "來源 XOR 與原版返回圖不符")
        outside = sum(a != b for j, (a, b) in enumerate(zip(before, after)) if not (x <= j % 320 < x + w and y <= j // 320 < y + h))
        require(outside == 0, "角色矩形外被變更")
        require(region(model(), domain) == region(before, domain), f"事件 {i} 之前場景模型不符")
        labels = event["source_images"]
        pairs = event["source_pairs"] or []
        choices = []
        for label in labels:
            name, number = label.split("#")
            number = int(number)
            aw, ah, px = assets[(name, number)]
            require((aw, ah) == (w, h), "完整來源尺寸不同")
            packed = bytes(px[j] << 4 | px[j + 1] for j in range(0, len(px), 2))
            require(packed == raw, "完整來源 bytes 不同")
            key = (name, x, y, w, h)
            previous = phases.get(key)
            if previous is None:
                choices.append((key, number))
            elif previous == number:
                choices.append((key, None))
        for pair in pairs:
            a, b = pair.split("^")
            name, ai = a.split("#")
            other, bi = b.split("#")
            require(name == other, "來源檔混合")
            ai, bi = int(ai), int(bi)
            pa, pb = assets[(name, ai)][2], assets[(name, bi)][2]
            delta = bytes(u ^ v for u, v in zip(pa, pb))
            packed = bytes(delta[j] << 4 | delta[j + 1] for j in range(0, len(delta), 2))
            require(packed == raw, "獨立差分來源 bytes 不同")
            key = (name, x, y, w, h)
            previous = phases.get(key)
            if previous in (ai, bi):
                choices.append((key, bi if previous == ai else ai))
        choices = list(dict.fromkeys(choices))
        require(len(choices) == 1, f"事件 {i} 轉換方向未知或有歧義：{choices}")
        key, target = choices[0]
        previous = phases.get(key)
        phases[key] = target
        require(region(model(), domain) == region(after, domain), f"事件 {i} 之後場景模型不符")
        omit = sum(a != b for a, b in zip(previous_region, next_region))
        require(omit > 0, "省略貼圖負對照無效")
        rows.append({"event": i, "name": key[0], "rect": rect, "from": previous, "to": target, "source_mismatch": 0, "outside_changed_pixels": outside, "scene_model_mismatch": 0, "negative_omit_pixels": omit})
    require(not [image for image in phases.values() if image is not None] and not mask_phases, "最後仍有敵人或特效殘留")
    with out.open("x") as f:
        json.dump({"scope": "此正常攻擊 975 次貼圖及全部位元遮罩；獨立來源、方向、角色＋XOR 特效場景及消除驗證", "inputs_sha256": inputs, "results": rows, "limits": "不是 HD 品質或其他戰鬥／特效驗收"}, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print(f"975 次貼圖與 {len(receipt['mask_events'])} 次位元遮罩的來源、方向、XOR 場景及特效消除通過")


if __name__ == "__main__":
    main()
