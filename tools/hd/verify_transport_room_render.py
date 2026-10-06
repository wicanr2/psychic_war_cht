"""24筆交通主題、正式文本與字型的獨立完整像素核對；研究038 §69。"""
import json
from pathlib import Path

from verify_next_body_plane import W, H, S, FW, FH, pbl, region, rgba, over, require, sha
from verify_over_runtime import read, load_font, render_text
from verify_over_frontend import compose, mismatch
from verify_anchor_render import art_plane, verify_hashes


def assets_from_files(inputs):
    root = Path("workplace/hd/theme-transport-room-v1-20261003")
    manifest = json.loads(read(root / "manifest.json", inputs))
    require(len(manifest["entries"]) == 24, "不是24筆主題")
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


def formal_text(inputs):
    result = {}
    for name in ("CODEH.BIN.json", "I_MAP00.BIN.json", "I_MAP01.BIN.json", "I_MENU01.BIN.json", "I_MENUH.BIN.json", "baked.json"):
        doc = json.loads(read(Path("text") / name, inputs))
        for entry in doc["entries"]:
            if "translation" in entry:
                key = entry["key"]
                require(key not in result or result[key] == entry["translation"], "正式來源鍵不唯一")
                result[key] = entry["translation"]
    return result


def formal_errors(side, formal):
    return [s["key"] for s in side["stamps"] if s.get("text") and
            (s["key"] not in formal or s["text"].strip() != formal[s["key"]].strip())]


def main():
    output = Path("workplace/hd/transport-room-render-verification-v1-20261003.json")
    require(not output.exists(), "拒絕覆寫")
    inputs = {}
    assets = assets_from_files(inputs)
    fonts = {name: load_font("font/" + name + ".golemfnt", inputs) for name in ("cjk24", "cjk16")}
    formal = formal_text(inputs)
    empty = bytes(FW * FH * 4)
    normal = json.loads(read("workplace/hd/transport-room-runtime-v1-20261003.json", inputs))
    verify_hashes(normal, inputs)
    start = json.loads(read("workplace/hd/transport-runtime-start-v1-20261003.json", inputs))
    verify_hashes(start, inputs)
    source = json.loads(read("workplace/hd/transport-room-source-v3-20261003.json", inputs))
    verify_hashes(source, inputs)
    require(normal["keys_irq1"] == 20 and normal["manifest_entries"] == 24 and len(normal["samples"]) == 21, "正常路線未閉合")
    events = {step: event for event in source["events"] for step in (event["Entry"], event["Return"])}
    rows, full_counts, seen_texts = [], {}, set()
    previous_room = None
    negative_formal = 0
    for row in normal["samples"]:
        prefix, step = row["prefix"], row["step"]
        frame = read(prefix + ".frame", inputs)
        require(sha(frame) == row["original"]["Frame"], "原版frame來源不符")
        read(prefix + ".state", inputs)
        if step in events:
            event = events[step]
            original = event["Before"] if step == event["Entry"] else event["After"]
            require(frame == read(original, inputs), "正常原版與独立來源探針畫面不同")
        if step == 614000000:
            require(frame == read("workplace/states/16-sivad.frame", inputs), "交通終點畫面不符")
        expected, full, cells = art_plane(frame, assets)
        actual = read(prefix + "-plane.rgba", inputs)
        require(actual == expected, f"正常全24筆圖面不符：{prefix}，差{mismatch(actual, expected)}")
        room_keys = [tuple(key) for key in full if key[0] == "ROOM0.PBL"]
        require(len(room_keys) <= 1, "同時顯示多個完整房間")
        if step in events and step == events[step]["Return"]:
            number = int(events[step]["SourceMatches"][0].split(":")[1])
            require(room_keys == [("ROOM0.PBL", number)], "三個完整返回房間未顯示HD")
        side_data = read(prefix + ".state.xlate.json", inputs)
        side = json.loads(side_data)
        errors = formal_errors(side, formal)
        require(not errors, "中文字面不符正式來源：" + str(errors))
        seen_texts.update(s["key"] for s in side["stamps"] if s.get("text"))
        bad_formal = dict(formal)
        bad_formal["I_MENUH.BIN:0850"] = "錯誤負對照"
        negative_formal += len(formal_errors(side, bad_formal))
        text = render_text(side, fonts)
        require(text == read(prefix + "-text.rgba", inputs), "正式字型完整繪製不符")
        rgb = read(prefix + "-rgb.bin", inputs)
        expected_png = compose(rgb, (expected, text))
        read(prefix + "-hd-chinese.png", inputs)
        w, h, actual_png = rgba(prefix + "-hd-chinese.png")
        require((w, h) == (FW, FH) and actual_png == expected_png, "正常完整合成不符")
        omitted = 0
        if room_keys:
            key = room_keys[0]
            full_counts[key[1]] = full_counts.get(key[1], 0) + 1
            no_room, _, _ = art_plane(frame, assets, omit_sources=(key,))
            omitted = mismatch(expected_png, compose(rgb, (no_room, text)))
            require(omitted > 0, "完整房間省略負對照無效")
            previous_room, _, _ = art_plane(frame, [a for a in assets if (a[0]["pbl"], a[0]["image"]) == key])
        stale_pixels = 0
        if step >= 617000000:
            require(not room_keys and not cells and previous_room is not None, "離房清除樣本仍有房間或缺前圖")
            stale = bytearray(expected)
            for offset in range(0, len(stale), 4):
                over(stale, offset, previous_room[offset:offset + 4])
            stale_pixels = mismatch(expected_png, compose(rgb, (stale, text)))
            require(stale_pixels > 0, "故意保留最後房間負對照無效")
        rows.append({"step": step, "eligible_full_sources": full, "room_cells": cells,
                     "plane_mismatch": 0, "formal_text_mismatch": 0, "font_mismatch": 0, "composition_mismatch": 0,
                     "negative_omit_room_pixels": omitted, "negative_stale_room_pixels": stale_pixels})
    require(set(full_counts) == {0, 2, 3} and negative_formal > 0, "三房間或正式文本負對照未閉合")
    print("正常21圖面／正式中文／字型／完整合成不符0，三房間省略與離房殘留負對照有效", flush=True)

    reload_doc = json.loads(read("workplace/hd/transport-room-reload-v1-20261003.json", inputs))
    verify_hashes(reload_doc, inputs)
    require(len(reload_doc["results"]) == 4, "載回數量不同")
    reload_rows = []
    for row in reload_doc["results"]:
        require(row["reload_and_toggle_equal"] and row["without_hd"] == row["with_hd"], "載回原版或重建不符")
        for phase, frame_path, plane_path in (("before", row["source"] + ".frame", row["prefix"] + "-before-plane.rgba"),
                                             ("after", row["prefix"] + ".frame", row["prefix"] + "-plane.rgba")):
            frame = read(frame_path, inputs)
            expected, full, cells = art_plane(frame, assets)
            require(read(plane_path, inputs) == expected, "載回完整24筆圖面不符")
            reload_rows.append({"sample": row["sample"], "phase": phase, "plane_mismatch": 0,
                                "eligible_full_sources": full, "room_cells": cells})
    print("四份載回及接續八圖面不符0", flush=True)

    root = Path("workplace/hd/transport-room-pwstep-v1-20261003")
    frontend = json.loads(read(root / "execution.json", inputs))
    export = json.loads(read(root / "original-frames.json", inputs))
    verify_hashes(frontend, inputs)
    verify_hashes(export, inputs)
    require(len(frontend["results"]) == len(export["results"]) == 16, "實際pwstep不是16份")
    exported = {r["state"]: r for r in export["results"]}
    controls, text_controls, frontend_rows = {}, {}, []
    for row in frontend["results"]:
        state, label = row["state"], row["label"]
        fp = {k: v for k, v in exported[state].items() if k != "state"}
        require(label not in controls or controls[label] == fp, "實際pwstep原版終點不同")
        controls[label] = fp
        frame = read(state + ".frame", inputs)
        rgb = read(state + ".rgb.bin", inputs)
        art, full, cells = art_plane(frame, assets)
        keys = [key for key in full if key[0] == "ROOM0.PBL"]
        require(keys == ([] if label == "clear" else [["ROOM0.PBL", int(label[-1])]]), "實際pwstep房間與來源不同")
        if row["language"] == "chinese":
            side_data = read(state + ".xlate.json", inputs)
            require(label not in text_controls or text_controls[label] == side_data, "pwstep HD改動中文快照")
            text_controls[label] = side_data
            side = json.loads(side_data)
            require(not formal_errors(side, formal), "pwstep中文字面不符正式來源")
            text = render_text(side, fonts)
        else:
            text = empty
        expected = compose(rgb, (art if row["hd"] else empty, text))
        read(row["png"], inputs)
        w, h, actual = rgba(row["png"])
        require((w, h) == (FW, FH) and actual == expected, "pwstep完整合成不符：" + row["png"])
        negative = 0
        if row["hd"] and keys:
            no_room, _, _ = art_plane(frame, assets, omit_sources=(tuple(keys[0]),))
            negative = mismatch(expected, compose(rgb, (no_room, text)))
            require(negative > 0, "pwstep房間省略負對照無效")
        frontend_rows.append({"png": row["png"], "label": label, "language": row["language"], "hd": row["hd"],
                              "composition_mismatch": 0, "negative_omit_room_pixels": negative})
    print("16份實際pwstep原版終點／中文快照／正式字型完整合成不符0", flush=True)
    for path in (__file__, "tools/hd/verify_anchor_render.py", "tools/hd/verify_next_body_plane.py",
                 "tools/hd/verify_over_runtime.py", "tools/hd/verify_over_frontend.py", "tools/pbl.py"):
        read(path, inputs)
    with output.open("x") as f:
        json.dump({"scope": "全部24筆主題，21正常樣本、4份state載回前後及16份正式pwstep独立完整核對",
                   "inputs_sha256": inputs, "normal": rows, "reload": reload_rows, "pwstep": frontend_rows,
                   "room_full_sample_counts": full_counts, "formal_text_keys": sorted(seen_texts),
                   "negative_formal_source_mismatches": negative_formal,
                   "limits": "無排除角色區域；只驗既有九鍵交通與首Up，非全部房間／sprite、美術、GUI、幀率或正式封包驗收。"},
                  f, ensure_ascii=False, indent=2)
        f.write("\n")


if __name__ == "__main__":
    main()
