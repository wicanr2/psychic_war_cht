"""研究038 §91：由原版frame／PBL、候選PNG及正式字型重建完整正常圖面。"""
import argparse
import json
from pathlib import Path

from verify_next_body_normal import load_font, render_text
from verify_next_body_plane import FW, FH, S, W, H, over, pbl, region, require, rgba, sha


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory")
    parser.add_argument("--entries", type=int, choices=(26, 28), default=28)
    args = parser.parse_args()
    root = Path(args.directory)
    inputs = {}

    def read(path):
        path = Path(path)
        data = path.read_bytes()
        inputs[str(path)] = sha(data)
        return data

    doc = json.loads(read(root / "runtime.json"))
    for path, expected in doc["inputs_sha256"].items():
        require(sha(read(path)) == expected, "來源變更：" + path)
    model = json.loads(read(doc["model"]))
    off, on = doc["results"]
    require(not off["hd"] and on["hd"], "HD控制側不符")
    require(off["end"] == on["end"] == model["observed_end"] == model["control_end"], "原版終點不符")
    require(doc["key_events"] == model["key_events"] and doc["seed_before"] == "F95B", "原版固定輸入不符")
    manifest = json.loads(read(Path(doc["theme"]) / "manifest.json"))
    require(len(manifest["entries"]) == args.entries, "候選主題筆數不符")
    decoded, base = {}, bytearray(W * H)
    for name in sorted({e["pbl"] for e in manifest["entries"]}):
        data = read(Path("/orig/psychic-war") / name)
        for image, offset, _ in pbl.images(data):
            w, h, px = pbl.decode(data, offset)
            decoded[name, image] = (w, h, bytes(px))
            if name in ("SCREEN.PBL", "MENU.PBL"):
                x, y = (0, image * 40) if name == "SCREEN.PBL" else (160, 4)
                for yy in range(h):
                    base[(y + yy) * W + x:(y + yy) * W + x + w] = bytes(px[yy * w:(yy + 1) * w])
    actors, assets = {}, []
    for entry in manifest["entries"]:
        name, image = entry["pbl"], entry["image"]
        w, h, original = decoded[name, image]
        sx, sy, rw, rh = entry.get("src", [0, 0, w, h])
        x, y = entry["at"]
        pxpath = Path(doc["theme"]) / entry["png"]
        read(pxpath)
        pw, ph, pixels = rgba(pxpath)
        require((pw, ph) == (rw * S, rh * S), "候選PNG尺寸不符")
        reference = bytearray(base)
        actor = name == "ALLY.PBL" or name.startswith("ENEMY")
        if actor or name in ("ROOM0.PBL", "OVER.PBL"):
            for yy in range(h):
                at = (y - sy + yy) * W + x - sx
                reference[at:at + w] = original[yy * w:(yy + 1) * w]
        match = entry.get("match", [4, 124, 72, 72] if name == "ROOM0.PBL"
                          else [128, 48, 64, 64] if name == "OVER.PBL" else [0, 0, 320, 40])
        key = f"{name}:{image}"
        if actor:
            actors[key] = {"name": name, "image": image, "rect": (x, y, w, h), "source": original,
                           "packed": bytes((original[i] << 4) | original[i + 1] for i in range(0, len(original), 2)),
                           "match": match, "want": region(reference, *match), "active": False, "return": 0}
        assets.append({"entry": entry, "key": key, "actor": actor, "reference": reference,
                       "pixels": pixels, "rect": (x, y, rw, rh), "match": match, "want": region(reference, *match)})
    # 只使用READY已證實的有向邊，不因相鄰圖號自行建立動作。
    edges = {"ENEMY00.PBL": {0: [1], 1: [0, 2], 2: [1], 3: [4], 4: [3, 5], 5: [4], 6: [7], 7: [6, 8], 8: [7]},
             "ENEMY01.PBL": {6: [7], 7: [6, 8], 8: [7]},
             "ENEMY03.PBL": {3: [4], 4: [3, 5], 5: [4]}}
    blits, frames = on["blits"], on["frames"]
    require(all("return" in e and e["step"] < e["return"] for e in blits), "角色貼圖未閉合")
    bi, states = 0, {}
    for tick in frames:
        frame = read(tick["frame"])
        require(len(frame) == 64000, "原版frame尺寸不符")
        while bi < len(blits) and blits[bi]["step"] <= tick["step"]:
            event = blits[bi]
            r = event["regs"]
            bx, by, bw, bh = (r["CX"] >> 8) * 4, (r["CX"] & 255) * 4, (r["DX"] >> 8) * 8, (r["DX"] & 255) * 8
            before, raw = read(event["before"]), read(event["source"])
            for key, actor in actors.items():
                x, y, w, h = actor["rect"]
                if not (bx <= x and by <= y and bx + bw >= x + w and by + bh >= y + h):
                    continue
                actor["active"], actor["return"] = False, event["return"]
                if (bx, by, bw, bh) != (x, y, w, h):
                    continue
                if r["AX"] & 255 == 0:
                    actor["active"] = raw == actor["packed"]
                elif r["AX"] & 255 == 1:
                    for source in edges.get(actor["name"], {}).get(actor["image"], []):
                        prev = decoded[actor["name"], source][2]
                        delta = bytes(((prev[i] << 4) | prev[i + 1]) ^ actor["packed"][i // 2] for i in range(0, len(prev), 2))
                        if region(before, x, y, w, h) == prev and raw == delta:
                            actor["active"] = True
            bi += 1
        full = {}
        for key, actor in actors.items():
            if region(frame, *actor["match"]) != actor["want"]:
                actor["active"], actor["return"] = False, 0
            elif tick["step"] >= actor["return"] and region(frame, *actor["rect"]) == actor["source"]:
                actor["active"] = True
                full[actor["rect"]] = key
        for key, actor in actors.items():
            if actor["rect"] in full and full[actor["rect"]] != key:
                actor["active"] = False
        states[tick["step"]] = sorted(k for k, a in actors.items() if a["active"])

    def art_plane(frame, active):
        plane, enemy_cells = bytearray(FW * FH * 4), []
        for asset in assets:
            entry = asset["entry"]
            if region(frame, *asset["match"]) != asset["want"] or (asset["actor"] and asset["key"] not in active):
                continue
            x, y, w, h = asset["rect"]
            for cy in range(y // 8 * 8, (y + h + 7) // 8 * 8, 8):
                for cx in range(x // 8 * 8, (x + w + 7) // 8 * 8, 8):
                    ref = region(asset["reference"], cx, cy, 8, 8)
                    if region(frame, cx, cy, 8, 8) != ref:
                        continue
                    if asset["actor"] or entry["pbl"] in ("ROOM0.PBL", "OVER.PBL"):
                        ink = ref if entry["pbl"] != "ROOM0.PBL" else bytes(
                            ref[yy * 8 + xx] for yy in range(8) for xx in range(8)
                            if x <= cx + xx < x + w and y <= cy + yy < y + h)
                        if not any(ink):
                            continue
                    if entry["pbl"] == "ENEMY03.PBL":
                        enemy_cells.append([cx, cy])
                    for yy in range(max(y, cy) * S, min(y + h, cy + 8) * S):
                        left, right = max(x, cx) * S, min(x + w, cx + 8) * S
                        pi = 4 * ((yy - y * S) * w * S + left - x * S)
                        source = asset["pixels"][pi:pi + 4 * (right - left)]
                        at = 4 * (yy * FW + left)
                        if all(source[i] == 255 for i in range(3, len(source), 4)):
                            plane[at:at + len(source)] = source
                        else:
                            for offset in range(0, len(source), 4):
                                if source[offset + 3]:
                                    over(plane, at + offset, source[offset:offset + 4])
        return plane, enemy_cells

    fonts = {name: load_font("font/" + name + ".golemfnt", inputs) for name in ("cjk24", "cjk16")}
    enemy_text = {e["key"]: e["translation"] for e in json.loads(read("text/I_ENMY03.BIN.json"))["entries"]}
    rows, name_count = [], 0
    for a, b in zip(off["samples"], on["samples"], strict=True):
        require(a["sample"] == b["sample"] and a["step"] == b["step"], "樣本錯配")
        prefix = Path(b["prefix"])
        frame = read(str(prefix) + ".frame")
        require(sha(frame) == a["frame_sha256"] == b["frame_sha256"], "HD改變原版frame")
        expected, cells = art_plane(frame, states[b["step"]])
        pw, ph, plane = rgba(str(prefix) + "-plane.png")
        read(str(prefix) + "-plane.png")
        require((pw, ph) == (FW, FH) and sha(plane) == b["plane_sha256"], "圖面來源不符")
        mismatch = sum(expected[i:i + 4] != plane[i:i + 4] for i in range(0, len(plane), 4))
        require(mismatch == 0, f"完整獨立8×8圖面不同：{prefix} {mismatch}")
        side = json.loads(read(str(prefix) + ".state.xlate.json"))
        expected_text = render_text(side, fonts)
        tw, th, text = rgba(str(prefix) + "-text.png")
        read(str(prefix) + "-text.png")
        require((tw, th) == (FW, FH) and text == expected_text and sha(text) == a["text_sha256"] == b["text_sha256"], "正式字型或HD兩側中文字面不符")
        names = [stamp for stamp in side["stamps"] if stamp["state"] == 2 and stamp["key"] in enemy_text]
        require(all(st["text"] == enemy_text[st["key"]] for st in names), "新敵人名稱與正式text資料不符")
        name_count += bool(names)
        rgb = read(str(prefix) + "-rgb.bin")
        require(len(rgb) == W * H * 3, "原版RGB尺寸不符")
        composed = bytearray()
        for y in range(H):
            row = b"".join(rgb[3 * (y * W + x):3 * (y * W + x) + 3] + b"\xff" for x in range(W) for _ in range(S))
            composed.extend(row * S)
        for layer in (expected, expected_text):
            for i in range(0, len(layer), 4):
                if layer[i + 3]:
                    over(composed, i, layer[i:i + 4])
        cw, ch, combined = rgba(str(prefix) + "-hd-chinese.png")
        read(str(prefix) + "-hd-chinese.png")
        require((cw, ch) == (FW, FH) and combined == composed, "完整獨立HD中文合成不符")
        omitted = sum(plane[4 * (yy * FW + xx) + 3] != 0 for cx, cy in cells
                      for yy in range(cy * S, (cy + 8) * S) for xx in range(cx * S, (cx + 8) * S))
        require(not cells or omitted > 0, "省略新敵人負對照無效")
        rw, rh, reload_plane = rgba(str(prefix) + "-reload-plane.png")
        read(str(prefix) + "-reload-plane.png")
        fresh = sorted(k for k, actor in actors.items() if region(frame, *actor["match"]) == actor["want"]
                       and region(frame, *actor["rect"]) == actor["source"])
        reload_expected, _ = art_plane(frame, fresh)
        require((rw, rh) == (FW, FH) and reload_plane == reload_expected, "實際載回的完整獨立圖面不符")
        require(sha(reload_plane) == doc["reloads"][b["sample"]]["reload_plane_sha256"], "載回圖面來源不符")
        rows.append({"sample": b["sample"], "kind": b["kind"], "step": b["step"],
                     "active_sources": states[b["step"]], "new_enemy_cells": cells,
                     "whole_plane_difference": 0, "whole_composition_difference": 0,
                     "text_difference": 0, "reload_difference": 0, "negative_omit_enemy_pixels": omitted,
                     "chinese_enemy_keys": [st["key"] for st in names],
                     "normal_reload_difference": sum(plane[i:i + 4] != reload_plane[i:i + 4] for i in range(0, len(plane), 4))})
    require(any(r["negative_omit_enemy_pixels"] > 0 for r in rows), "新來源未實際呈現HD")
    require(name_count > 0, "正常原版路線未呈現新敵人中文")
    for path in [__file__, "tools/hd/verify_next_body_normal.py", "tools/hd/verify_next_body_plane.py", "tools/pbl.py"]:
        read(path)
    result = {"scope": f"entire {args.entries}-entry HD plane, Chinese glyph plane and composition; no excluded regions",
              "inputs_sha256": inputs, "rows": rows, "chinese_enemy_samples": name_count,
              "independent_original_frame_ticks": len(frames), "original_slot_blits": len(blits),
              "limits": "art candidates not accepted; partial and overlap may fall back; cold-start text only, not fromboot, real GUI, DAT or full serialized-state acceptance"}
    with (root / "independent.json").open("x") as f:
        json.dump(result, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print(root.name, "complete compositions", len(rows), "differences0; new Chinese name", name_count,
          "new enemy HD", sum(bool(r["new_enemy_cells"]) for r in rows))


if __name__ == "__main__":
    main()
