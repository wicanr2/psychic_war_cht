"""獨立PBL／PNG／字型核對完整21筆圖面、正常／載回／pwstep；研究038 §66。"""
import json
from pathlib import Path

from verify_next_body_plane import W, H, S, FW, FH, pbl, region, rgba, over, require, sha
from verify_over_runtime import read, load_font, render_text
from verify_over_frontend import compose, mismatch


def assets_from_files(inputs):
    root = Path("workplace/hd/theme-room-v1-20261003")
    manifest = json.loads(read(root / "manifest.json", inputs))
    require(len(manifest["entries"]) == 21, "不是21筆主題")
    base, decoded = bytearray(W * H), {}
    for name in ("SCREEN.PBL", "MENU.PBL", "ALLY.PBL", "ENEMY00.PBL", "ENEMY01.PBL", "OVER.PBL", "ROOM0.PBL"):
        data = read(Path("/orig/psychic-war") / name, inputs)
        for image, offset, _ in pbl.images(data):
            w, h, pixels = pbl.decode(data, offset)
            decoded[name, image] = w, h, bytes(pixels)
            if name in ("SCREEN.PBL", "MENU.PBL"):
                x, y = (0, image * 40) if name == "SCREEN.PBL" else (160, 4)
                for yy in range(h):
                    base[(y + yy) * W + x:(y + yy) * W + x + w] = pixels[yy * w:(yy + 1) * w]
    assets = []
    for entry in manifest["entries"]:
        name = entry["pbl"]
        w, h, source = decoded[name, entry["image"]]
        path = root / entry["png"]
        read(path, inputs)
        pw, ph, pixels = rgba(path)
        require((pw, ph) == (w * S, h * S), "候選尺寸不同")
        reference = bytearray(W * H) if name == "OVER.PBL" else bytearray(base)
        if name not in ("SCREEN.PBL", "MENU.PBL"):
            x, y = entry["at"]
            for yy in range(h):
                reference[(y + yy) * W + x:(y + yy) * W + x + w] = source[yy * w:(yy + 1) * w]
        assets.append((entry, w, h, source, pixels, reference))
    return assets


def art_plane(frame, assets, omit_room=False, force_room=False):
    """本批完整來源直接可判定；不從實際圖面或內部active旗標取得期望。"""
    require(len(frame) == W * H, "原版畫面尺寸不同")
    plane = bytearray(FW * FH * 4)
    full, room_cells = [], []
    for entry, w, h, source, pixels, reference in assets:
        name = entry["pbl"]
        x, y = entry["at"]
        if name == "ROOM0.PBL" and omit_room:
            continue
        if name in ("ROOM0.PBL", "OVER.PBL"):
            if region(frame, x, y, w, h) != source and not (force_room and name == "ROOM0.PBL"):
                continue
        else:
            if region(frame, 0, 0, W, 40) != region(reference, 0, 0, W, 40):
                continue
            if name not in ("SCREEN.PBL", "MENU.PBL") and region(frame, x, y, w, h) != source:
                continue
        if name not in ("SCREEN.PBL", "MENU.PBL"):
            full.append([name, entry["image"]])
        for cy in range(y // 8 * 8, (y + h + 7) // 8 * 8, 8):
            for cx in range(x // 8 * 8, (x + w + 7) // 8 * 8, 8):
                ref = region(reference, cx, cy, 8, 8)
                if region(frame, cx, cy, 8, 8) != ref:
                    continue
                if name == "ROOM0.PBL":
                    ink = region(reference, max(x, cx), max(y, cy), min(x + w, cx + 8) - max(x, cx), min(y + h, cy + 8) - max(y, cy))
                else:
                    ink = ref
                if name not in ("SCREEN.PBL", "MENU.PBL") and not any(ink):
                    continue
                if name == "ROOM0.PBL":
                    room_cells.append([cx, cy])
                for yy in range(max(y, cy) * S, min(y + h, cy + 8) * S):
                    for xx in range(max(x, cx) * S, min(x + w, cx + 8) * S):
                        offset = 4 * ((yy - y * S) * w * S + xx - x * S)
                        over(plane, 4 * (yy * FW + xx), pixels[offset:offset + 4])
    return plane, full, room_cells


def verify_hashes(doc, inputs):
    for path, expected in doc["inputs_sha256"].items():
        require(sha(read(path, inputs)) == expected, "來源已變更：" + path)


def main():
    output = Path("workplace/hd/room-render-verification-v2-20261003.json")
    require(not output.exists(), "拒絕覆寫")
    inputs = {}
    assets = assets_from_files(inputs)
    room = next(a for a in assets if a[0]["pbl"] == "ROOM0.PBL")
    fonts = {name: load_font("font/" + name + ".golemfnt", inputs) for name in ("cjk24", "cjk16")}
    empty = bytes(FW * FH * 4)
    rows = []
    normal = json.loads(read("workplace/hd/room-runtime-v1-20261003.json", inputs))
    verify_hashes(normal, inputs)
    require(normal["keys_irq1"] == 20 and normal["manifest_entries"] == 21 and len(normal["samples"]) == 13, "正常路線未閉合")
    for row in normal["samples"]:
        prefix = row["prefix"]
        frame = read(prefix + ".frame", inputs)
        require(sha(frame) == row["original"]["Frame"], "實際原版frame來源不同")
        read(prefix + ".state", inputs)
        expected, full, cells = art_plane(frame, assets)
        actual = read(prefix + "-plane.rgba", inputs)
        require(actual == expected, f"正常完整21筆圖面不符：{prefix}，差{mismatch(actual, expected)}")
        ally = next(a for a in assets if a[0]["pbl"] == "ALLY.PBL")
        require(region(frame, 264, 152, 24, 32) == ally[3], "本批盟友原圖不完整")
        require(not any(name.startswith("ENEMY") for name, _ in full), "本批出現完整敵人來源")
        side = json.loads(read(prefix + ".state.xlate.json", inputs))
        text = render_text(side, fonts)
        require(text == read(prefix + "-text.rgba", inputs), "正式字型獨立繪製不符")
        rgb = read(prefix + "-rgb.bin", inputs)
        expected_png = compose(rgb, (expected, text))
        read(prefix + "-hd-chinese.png", inputs)
        w, h, actual_png = rgba(prefix + "-hd-chinese.png")
        require((w, h) == (FW, FH) and actual_png == expected_png, "正常獨立完整合成不符")
        no_room, _, _ = art_plane(frame, assets, omit_room=True)
        omitted = mismatch(expected_png, compose(rgb, (no_room, text)))
        full_room = region(frame, 4, 124, 72, 72) == room[3]
        require((omitted > 0) == full_room, "房間省略負對照未生效或殘留")
        ghost = 0
        if row["step"] >= 349000000:
            stale, _, _ = art_plane(frame, assets, force_room=True)
            ghost = mismatch(expected, stale)
            require(ghost > 0, "故意保留房間負對照無效")
        rows.append({"step": row["step"], "eligible_full_sources": full, "ally_source_complete": True, "room_cells": cells,
                     "plane_mismatch": 0, "text_mismatch": 0, "composition_mismatch": 0,
                     "negative_omit_room_pixels": omitted, "negative_stale_room_pixels": ghost})
    require(sum(bool(r["room_cells"]) for r in rows) == 4, "完整房間樣本不是四份")
    print("正常十三份完整21筆圖面／中文／合成不符0", flush=True)

    reload_doc = json.loads(read("workplace/hd/room-reload-v1-20261003.json", inputs))
    verify_hashes(reload_doc, inputs)
    require(len(reload_doc["results"]) == 4, "載回數量不同")
    reload_rows = []
    for row in reload_doc["results"]:
        require(row["reload_and_toggle_equal"] and row["without_hd"] == row["with_hd"], "載回或原版接續不符")
        for phase, frame_path, plane_path in (
            ("before", row["source"] + ".frame", row["prefix"] + "-before-plane.rgba"),
            ("after", row["prefix"] + ".frame", row["prefix"] + "-plane.rgba"),
        ):
            frame = read(frame_path, inputs)
            expected, full, cells = art_plane(frame, assets)
            actual = read(plane_path, inputs)
            require(actual == expected, "載回獨立完整圖面不符")
            reload_rows.append({"sample": row["sample"], "phase": phase, "eligible_full_sources": full,
                                "room_cells": cells, "plane_mismatch": 0})
    print("四份載回及接續八圖面不符0", flush=True)

    root = Path("workplace/hd/room-pwstep-v1-20261003")
    frontend = json.loads(read(root / "execution.json", inputs))
    export = json.loads(read(root / "original-frames.json", inputs))
    verify_hashes(frontend, inputs)
    verify_hashes(export, inputs)
    require(len(frontend["results"]) == len(export["results"]) == 8, "實際pwstep不是八份")
    exported = {row["state"]: row for row in export["results"]}
    controls, text_controls, frontend_rows = {}, {}, []
    for row in frontend["results"]:
        state = row["state"]
        fp = {k: v for k, v in exported[state].items() if k != "state"}
        label = row["label"]
        require(label not in controls or controls[label] == fp, "實際pwstep原版終點不符")
        controls[label] = fp
        frame = read(state + ".frame", inputs)
        rgb = read(state + ".rgb.bin", inputs)
        art, full, cells = art_plane(frame, assets)
        require(bool(cells) == (label == "full"), "實際pwstep房間完整性不符")
        if row["language"] == "chinese":
            side_data = read(state + ".xlate.json", inputs)
            require(label not in text_controls or text_controls[label] == side_data, "實際pwstep HD改變中文字面")
            text_controls[label] = side_data
            text = render_text(json.loads(side_data), fonts)
        else:
            text = empty
        expected = compose(rgb, (art if row["hd"] else empty, text))
        read(row["png"], inputs)
        w, h, actual = rgba(row["png"])
        require((w, h) == (FW, FH) and actual == expected, "實際pwstep獨立完整合成不符：" + row["png"])
        omit_room, _, _ = art_plane(frame, assets, omit_room=True)
        negative = mismatch(expected, compose(rgb, (omit_room, text))) if row["hd"] else 0
        require(not row["hd"] or label != "full" or negative > 0, "pwstep房間省略負對照無效")
        frontend_rows.append({"png": row["png"], "label": label, "language": row["language"], "hd": row["hd"],
                              "composition_mismatch": 0, "negative_omit_room_pixels": negative})
    print("八份實際pwstep原版終點／中文快照／獨立完整合成不符0", flush=True)
    for name in (__file__, "tools/hd/verify_next_body_plane.py", "tools/hd/verify_over_runtime.py",
                 "tools/hd/verify_over_frontend.py", "tools/pbl.py"):
        read(name, inputs)
    with output.open("x") as f:
        json.dump({"scope": "完整21筆主題，13正常樣本、4份state載回前後及8份實際pwstep獨立像素核對",
                   "inputs_sha256": inputs, "normal": rows, "reload": reload_rows, "pwstep": frontend_rows,
                   "limits": "無排除角色區域；本批ALLY始終完整、無完整敵人姿勢，不證明其他場景的身份或動作覆蓋。GUI、美術、其他房間、全部sprite、幀率及實際封包未驗。"},
                  f, ensure_ascii=False, indent=2)
        f.write("\n")


if __name__ == "__main__":
    main()
