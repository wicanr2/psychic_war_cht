"""獨立核對正常遭遇的原版差分、8×8敵人遮格與中文合成；入口研究038 §51。"""
import argparse
import json
import pathlib
import struct

from verify_next_body_plane import FW, FH, S, W, H, over, pbl, region, require, rgba, sha


def load_font(path, inputs):
    data = read(path, inputs)
    require(data[:8] == b"GOLEMFNT", "字型格式不同")
    w, h, n = struct.unpack_from("<HHI", data, 8)
    size = h * ((w + 7) // 8)
    require(len(data) == 16 + n * (size + 5), "字型長度不同")
    glyphs = {}
    for i in range(n):
        at = 16 + i * (size + 5)
        code = struct.unpack_from("<I", data, at)[0]
        glyphs[chr(code)] = data[at + 5:at + 5 + size]
    return w, h, glyphs


def read(path, inputs):
    path = pathlib.Path(path)
    data = path.read_bytes()
    inputs[str(path)] = sha(data)
    return data


def render_text(side, fonts):
    plane = bytearray(FW * FH * 4)
    for stamp in side["stamps"]:
        if stamp["state"] != 2:
            continue
        x, y = stamp["x"] * S, stamp["y"] * S
        cw, ch = stamp["cell_w"] * S, stamp["cell_h"] * S
        transparent = stamp.get("transparent", [])
        for i in range(stamp["cells"]):
            if i < len(transparent) and transparent[i]:
                continue
            cx = x + i * cw
            bg = bytes(stamp["bg"] + [255]) * cw
            require(0 <= cx and cx + cw <= FW and 0 <= y and y + ch <= FH, "中文格越界")
            for yy in range(y, y + ch):
                at = 4 * (yy * FW + cx)
                plane[at:at + cw * 4] = bg
            if i >= len(stamp["text"]) or stamp["text"][i] in (" ", "　") or not stamp.get("font"):
                continue
            w, h, glyphs = fonts[stamp["font"]]
            glyph = glyphs.get(stamp["text"][i])
            require(glyph is not None, "中文缺字")
            k = stamp["glyph_scale"] or S // 3
            for gy in range(h):
                for gx in range(w):
                    if not glyph[gy * ((w + 7) // 8) + gx // 8] & (0x80 >> (gx % 8)):
                        continue
                    for dy in range(k):
                        ly = stamp["glyph_y"] + gy * k + dy
                        for dx in range(k):
                            lx = stamp["glyph_x"] + gx * k + dx
                            if 0 <= lx < cw and 0 <= ly < ch:
                                at = 4 * ((y + ly) * FW + cx + lx)
                                plane[at:at + 4] = bytes(stamp["fg"] + [255])
    return plane


def art_plane(frame, assets, active_enemy):
    plane = bytearray(FW * FH * 4)
    enemy_cells = []
    anchor = region(frame, 0, 0, 320, 40) == region(assets[0][5], 0, 0, 320, 40)
    for entry, w, h, source, pixels, reference in assets:
        if entry["pbl"] == "OVER.PBL":
            if region(frame, 128, 48, 64, 64) != source:
                continue
        elif not anchor:
            continue
        if entry["pbl"] == "ALLY.PBL":
            # 本批未保存覆蓋盟友的全部貼圖事件，不猜盟友的持續身份。
            continue
        if entry["pbl"] == "ENEMY00.PBL" or (entry["pbl"] == "ENEMY01.PBL" and entry["image"] != active_enemy):
            continue
        x, y = entry["at"]
        for cy in range(y // 8 * 8, (y + h + 7) // 8 * 8, 8):
            for cx in range(x // 8 * 8, (x + w + 7) // 8 * 8, 8):
                ref = region(reference, cx, cy, 8, 8)
                if region(frame, cx, cy, 8, 8) != ref or (entry["pbl"] in ("ENEMY01.PBL", "OVER.PBL") and not any(ref)):
                    continue
                if entry["pbl"] == "ENEMY01.PBL":
                    enemy_cells.append([cx, cy])
                for yy in range(max(cy, y) * S, min(cy + 8, y + h) * S):
                    for xx in range(max(cx, x) * S, min(cx + 8, x + w) * S):
                        at = 4 * ((yy - y * S) * w * S + xx - x * S)
                        over(plane, 4 * (yy * FW + xx), pixels[at:at + 4])
    return plane, enemy_cells


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", required=True)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    output = pathlib.Path(args.out)
    require(args.check or not output.exists(), "拒絕覆寫")
    inputs = {}
    root = pathlib.Path("workplace/hd/theme-over-v1-20261002")
    manifest = json.loads(read(root / "manifest.json", inputs))
    base, decoded = bytearray(W * H), {}
    for name in ("SCREEN.PBL", "MENU.PBL", "ALLY.PBL", "ENEMY00.PBL", "ENEMY01.PBL", "OVER.PBL"):
        data = read(pathlib.Path("/orig/psychic-war") / name, inputs)
        for image, offset, _ in pbl.images(data):
            w, h, source = pbl.decode(data, offset)
            decoded[name, image] = w, h, bytes(source)
            if name in ("SCREEN.PBL", "MENU.PBL"):
                x, y = (0, image * 40) if name == "SCREEN.PBL" else (160, 4)
                for yy in range(h):
                    base[(y + yy) * W + x:(y + yy) * W + x + w] = source[yy * w:(yy + 1) * w]
    assets = []
    for entry in manifest["entries"]:
        w, h, source = decoded[entry["pbl"], entry["image"]]
        path = root / entry["png"]
        read(path, inputs)
        pw, ph, pixels = rgba(path)
        require((pw, ph) == (w * S, h * S), "候選尺寸不同")
        reference = bytearray(W * H) if entry["pbl"] == "OVER.PBL" else bytearray(base)
        if entry["pbl"] in ("ENEMY00.PBL", "ENEMY01.PBL", "ALLY.PBL", "OVER.PBL"):
            x, y = entry["at"]
            for yy in range(h):
                reference[(y + yy) * W + x:(y + yy) * W + x + w] = source[yy * w:(yy + 1) * w]
        assets.append((entry, w, h, source, pixels, reference))
    poses = {i: decoded["ENEMY01.PBL", i][2] for i in range(6,9)}
    packed = {i: bytes((p[j] << 4) | p[j + 1] for j in range(0, len(p), 2)) for i, p in poses.items()}
    fonts = {name: load_font("font/" + name + ".golemfnt", inputs) for name in ("cjk24", "cjk16")}
    entries = {}
    for name in ("I_MENU01.BIN", "I_ENMY01.BIN"):
        for entry in json.loads(read("text/" + name + ".json", inputs))["entries"]:
            entries[entry["key"]] = entry["translation"]
    routes = []
    for label in ("oogus",):
        prefix = "workplace/hd/over-runtime-v1-20261002"
        doc = json.loads(read(prefix + ".json", inputs))
        for path, expected_hash in doc["inputs_sha256"].items():
            require(sha(read(path, inputs)) == expected_hash, "正常來源改變：" + path)
        off, on = doc["results"]
        require(not off["hd"] and on["hd"], "HD兩側順序不同")
        require(off["end"] == on["end"] == doc["model_end"] and off["ram_bus_sha256"] == on["ram_bus_sha256"], "原版終點不同")
        chain = json.loads(read("workplace/hd/sivad-translation-v1-20261002.json", inputs))
        require(all(row["baseline_equal"] for row in chain["results"]), "正常中文鏈不符")
        origin = next(row for row in chain["results"] if row["layer_json"] == doc["initial_xlate"])
        require(sha(read(doc["initial_xlate"], inputs)) == origin["layer_sha256"], "中文起點不是正常來源")
        targets = []
        transitions = []
        for event in doc["events"]:
            before, after, raw = (read(event[name], inputs) for name in ("Before", "After", "Source"))
            require(len(before) == len(after) == W * H and len(raw) == 384, "原版來源尺寸不同")
            r = event["Regs"]
            require(r["CS"] == 0x161 and r["IP"] == 0x8705 and r["CX"] == 0x0826 and r["DX"] == 0x0304, "原版位置不同")
            mode = r["AX"] & 255
            require(mode in (0, 1), "未知原版模式")
            unpacked = bytes(v for b in raw for v in (b >> 4, b & 15))
            expected = bytearray(before)
            prior = region(before, 32, 152, 24, 32)
            for yy in range(32):
                at = (152 + yy) * W + 32
                expected[at:at + 24] = bytes((prior[yy * 24 + x] if mode else 0) ^ unpacked[yy * 24 + x] for x in range(24))
            require(expected == after, "原版差分或矩形外變動不符")
            wrong_mode = bytes((0 if mode else prior[i]) ^ value for i, value in enumerate(unpacked))
            mode_negative = sum(a != b for a, b in zip(wrong_mode, region(after, 32, 152, 24, 32)))
            require(mode != 1 or mode_negative > 0, "差分錯模式負對照無效")
            full_before = [i for i, pose in poses.items() if pose == prior]
            target = [i for i, data in packed.items() if data == raw] if mode == 0 else [i for j in full_before for i in range(j // 3 * 3, j // 3 * 3 + 3) if abs(i - j) == 1 and bytes(a ^ b for a, b in zip(packed[j], packed[i])) == raw]
            require(len(target) <= 1, "敵人來源不唯一")
            targets.append(target[0] if target else None)
            transitions.append({"mode": mode, "full_before": full_before, "confirmed_target": target, "full_after": event["To"], "original_mismatch": 0, "negative_wrong_mode_pixels": mode_negative})
        rows = []
        for sample, a, b in zip(doc["samples"], off["phases"], on["phases"], strict=True):
            frame = read(sample["Name"] + ".frame", inputs)
            require(sha(frame) == a["original_frame_sha256"] == b["original_frame_sha256"], "原版取樣不同")
            require(a["text_plane_sha256"] == b["text_plane_sha256"], "HD改變中文")
            active = targets[sample["Event"]] if sample["Event"] >= 0 else None
            expected, cells = art_plane(frame, assets, active)
            read(sample["Name"] + "-plane.png", inputs)
            pw, ph, actual = rgba(sample["Name"] + "-plane.png")
            require((pw, ph) == (FW, FH) and sha(actual) == b["hd_plane_sha256"], "HD圖面來源不同")
            # 盟友完整生存期另有§34證據，本批不以未保存的覆蓋事件猜出期望。
            for yy in range(152 * S, 184 * S):
                at, end = 4 * (yy * FW + 264 * S), 4 * (yy * FW + 288 * S)
                expected[at:end] = actual[at:end]
            mismatch = sum(actual[i:i + 4] != expected[i:i + 4] for i in range(0, len(actual), 4))
            require(mismatch == 0, "8×8期望不同：" + sample["Name"] + f" ({mismatch})")
            side = json.loads(read(sample["Name"] + ".state.xlate.json", inputs))
            expected_text = render_text(side, fonts)
            read(sample["Name"] + "-text.png", inputs)
            tw, th, text = rgba(sample["Name"] + "-text.png")
            require((tw, th) == (FW, FH) and sha(text) == a["text_plane_sha256"] and text == expected_text, "獨立字型渲染不同")
            key = "I_ENMY01.BIN:00A2"
            if sample["Kind"] == "complete":
                require(any(st["key"] == key and st["text"] == entries[key] and st["state"] == 2 for st in side["stamps"]), "敵人名稱中文不符")
            rgb = read(sample["Name"] + "-rgb.bin", inputs)
            require(len(rgb) == W * H * 3, "RGB尺寸不同")
            composed = bytearray()
            for y in range(H):
                row = b"".join(rgb[3 * (y * W + x):3 * (y * W + x) + 3] + b"\xff" for x in range(W) for _ in range(S))
                composed.extend(row * S)
            for plane in (expected, expected_text):
                for i in range(0, len(plane), 4):
                    if plane[i + 3]:
                        over(composed, i, plane[i:i + 4])
            read(sample["Name"] + "-hd-chinese.png", inputs)
            cw, ch, combined = rgba(sample["Name"] + "-hd-chinese.png")
            require((cw, ch) == (FW, FH) and combined == composed, "獨立合成不同")
            omitted = sum(actual[4 * (yy * FW + xx) + 3] != 0 for cx, cy in cells for yy in range(cy * S, (cy + 8) * S) for xx in range(cx * S, (cx + 8) * S))
            require(not cells or omitted > 0, "省略負對照無效")
            ghost_negative = 0
            if sample["Kind"] == "end" and doc["escape_returns"] == 1:
                # 走廊背景會正常蓋回敵人區；已核對的期望是背景，不能要求圖面全透明。
                require(not cells and active is None, "返回迷宮後仍認定HD敵人")
                ghost = next(asset for asset in assets if asset[0]["pbl"] == "ENEMY01.PBL" and asset[0]["image"] == 7)
                ghost_plane = bytearray(actual)
                for yy in range(ghost[2] * S):
                    for xx in range(ghost[1] * S):
                        at = 4 * (((152 * S + yy) * FW) + 32 * S + xx)
                        gi = 4 * (yy * ghost[1] * S + xx)
                        over(ghost_plane, at, ghost[4][gi:gi + 4])
                ghost_negative = sum(ghost_plane[i:i + 4] != actual[i:i + 4] for i in range(0, len(actual), 4))
                require(ghost_negative > 0, "故意保留敵人負對照無效")
            text_negative = sum(text[i + 3] != 0 for i in range(0, len(text), 4))
            require(sample["Kind"] != "complete" or text_negative > 0, "戰鬥中文省略負對照無效")
            full_over = region(frame, 128, 48, 64, 64) == decoded["OVER.PBL", 0][2]
            without_over, _ = art_plane(frame, [a for a in assets if a[0]["pbl"] != "OVER.PBL"], active)
            for yy in range(152*S,184*S):
                at, end = 4*(yy*FW+264*S),4*(yy*FW+288*S)
                without_over[at:end] = actual[at:end]
            over_ink = sum(actual[i:i+4] != without_over[i:i+4] for i in range(0,len(actual),4))
            require(not full_over or over_ink > 0, "OVER省略負對照無效")
            if sample["Kind"] in ("over-scene", "end") and not full_over:
                require(over_ink == 0, "清除或中途仍留OVER殘片")
            rows.append({"sample": sample, "target": active, "enemy_cells": cells, "plane_mismatch_outside_ally": mismatch, "text_mismatch": 0, "composition_mismatch": 0, "negative_omit_enemy_pixels": omitted, "negative_omit_text_pixels": text_negative, "negative_ghost_enemy_pixels": ghost_negative, "full_over": full_over, "negative_omit_over_pixels": over_ink})
        routes.append({"label": label, "escape_returns": doc["escape_returns"], "transitions": transitions, "samples": rows})
        print(label, "來源", len(transitions), "取樣", len(rows), "原版／圖面／中文／合成不符0", flush=True)
    # 真實保存狀態的載回收據另驗；期望仍由原版PBL與候選推導。
    reload_doc = json.loads(read("workplace/hd/over-reload-v1-20261002.json", inputs))
    for path, expected_hash in reload_doc["inputs_sha256"].items():
        require(sha(read(path, inputs)) == expected_hash, "載回來源改變：" + path)
    require(len(reload_doc["results"]) == 4, "載回樣本數不同")
    reload_rows = []
    full_plane = None
    for row in reload_doc["results"]:
        require(row["reload_and_toggle_equal"] and row["with_hd"] == row["without_hd"], "載回或原版接續不同")
        frame = read(row["frame"], inputs)
        read(row["state"], inputs)
        expected, _ = art_plane(frame, assets, None)
        read(row["plane_png"], inputs)
        pw, ph, actual = rgba(row["plane_png"])
        require((pw, ph) == (FW, FH) and actual == expected, "載回獨立完整圖面不同")
        full_over = region(frame, 128, 48, 64, 64) == decoded["OVER.PBL", 0][2]
        without_over, _ = art_plane(frame, [a for a in assets if a[0]["pbl"] != "OVER.PBL"], None)
        ink = sum(actual[i:i+4] != without_over[i:i+4] for i in range(0,len(actual),4))
        if full_over:
            require(ink > 0, "載回省略OVER負對照無效")
            full_plane = actual
        else:
            require(ink == 0, "清除場景載回留殘片")
        stale_negative = 0
        if not full_over and full_plane is not None:
            stale = bytearray(actual)
            for i in range(0,len(actual),4):
                if full_plane[i+3]:
                    over(stale,i,full_plane[i:i+4])
            stale_negative = sum(actual[i:i+4] != stale[i:i+4] for i in range(0,len(actual),4))
            require(stale_negative > 0, "故意保留人物負對照無效")
        reload_rows.append({"sample": row["sample"], "full_over": full_over, "plane_mismatch_pixels": 0,
                            "negative_omit_over_pixels": ink, "negative_stale_over_pixels": stale_negative})
    require(sum(row["full_over"] for row in reload_rows) == 2, "兩個完整人物場景未閉合")
    for name in (__file__, "tools/hd/verify_sivad_normal.py", "tools/hd/verify_next_body_plane.py", "tools/pbl.py"):
        read(name, inputs)
    result = {"scope": "正常Sivad兩次方向鍵、原版陣亡及正常Enter，8次身體與21幀抽樣；另4個原版state實際載回", "inputs_sha256": inputs, "routes": routes, "reload_results": reload_rows, "limits": "正常路線圖面獨立核對排除盟友(264,152,24,32)的持續身份；該區只核對實際合成。中文從正常印字與正式文本／字型核對，非全文或GUI驗收；Sivad原版HP0返回一次，Enter清除OVER；首完整敵人姿勢以外重疊回退，不推定逃走規則或全姿勢HD完成；美術未驗。"}
    if args.check:
        require(json.loads(output.read_text()) == result, "獨立收據重生不同")
    else:
        with output.open("x") as f:
            json.dump(result, f, ensure_ascii=False, indent=2)
            f.write("\n")


if __name__ == "__main__":
    main()
