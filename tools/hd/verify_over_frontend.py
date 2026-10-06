"""原版PBL／候選PNG／正式文本／字型獨立核對OVER前端；研究038 §52。"""
import json
from pathlib import Path

from verify_over_runtime import art_plane, load_font, render_text, read
from verify_next_body_plane import W, H, S, FW, FH, rgba, pbl, over, region, require, sha


def compose(rgb, planes):
    require(len(rgb) == W * H * 3, "原版RGB尺寸不同")
    result = bytearray()
    for y in range(H):
        row = b"".join(rgb[3*(y*W+x):3*(y*W+x)+3] + b"\xff" for x in range(W) for _ in range(S))
        result.extend(row*S)
    for plane in planes:
        for i in range(0, len(plane), 4):
            if plane[i+3]:
                over(result, i, plane[i:i+4])
    return result


def mismatch(a, b):
    require(len(a) == len(b), "對照尺寸不同")
    return sum(a[i:i+4] != b[i:i+4] for i in range(0, len(a), 4))


def main():
    output = Path("workplace/hd/over-frontend-verification-v4-20261002.json")
    require(not output.exists(), "拒絕覆寫")
    inputs = {}
    root = Path("workplace/hd/theme-over-v1-20261002")
    manifest = json.loads(read(root/"manifest.json", inputs))
    require(len(manifest["entries"]) == 20, "主題筆數不同")
    base, decoded = bytearray(W*H), {}
    for name in ("SCREEN.PBL", "MENU.PBL", "ALLY.PBL", "ENEMY00.PBL", "ENEMY01.PBL", "OVER.PBL"):
        data = read(Path("/orig/psychic-war")/name, inputs)
        for image, offset, _ in pbl.images(data):
            w, h, source = pbl.decode(data, offset)
            decoded[name, image] = w, h, bytes(source)
            if name in ("SCREEN.PBL", "MENU.PBL"):
                x, y = (0, image*40) if name == "SCREEN.PBL" else (160, 4)
                for yy in range(h):
                    base[(y+yy)*W+x:(y+yy)*W+x+w] = source[yy*w:(yy+1)*w]
    assets = []
    for entry in manifest["entries"]:
        w, h, source = decoded[entry["pbl"], entry["image"]]
        path = root/entry["png"]
        read(path, inputs)
        pw, ph, pixels = rgba(path)
        require((pw, ph) == (w*S, h*S), "候選尺寸不同")
        reference = bytearray(W*H) if entry["pbl"] == "OVER.PBL" else bytearray(base)
        if entry["pbl"] not in ("SCREEN.PBL", "MENU.PBL"):
            x, y = entry["at"]
            for yy in range(h):
                reference[(y+yy)*W+x:(y+yy)*W+x+w] = source[yy*w:(yy+1)*w]
        assets.append((entry, w, h, source, pixels, reference))
    fonts = {n: load_font("font/"+n+".golemfnt", inputs) for n in ("cjk24", "cjk16")}
    entries = {e["key"]: e["translation"] for e in json.loads(read("text/PW.EXE.json", inputs))["entries"]}
    keys = ["PW.EXE:cs:08D7", "PW.EXE:cs:0901", "PW.EXE:cs:092B"]
    original_over = decoded["OVER.PBL", 0][2]
    empty = bytes(FW*FH*4)

    def validate_export(export):
        for path, expected_hash in export["inputs_sha256"].items():
            require(sha(read(path, inputs)) == expected_hash, "匯出輸入已變更："+path)
        for row in export["results"]:
            path = row["state"]
            require(sha(read(path+".frame", inputs)) == row["frame_sha256"], "色號匯出已變更："+path)
            require(sha(read(path+".rgb.bin", inputs)) == row["rgb_sha256"], "RGB匯出已變更："+path)

    def check_image(path, hd, chinese, full):
        frame = read(str(path)+".state.frame", inputs)
        rgb = read(str(path)+".state.rgb.bin", inputs)
        read(str(path)+".state", inputs)
        side = json.loads(read(str(path)+".state.xlate.json", inputs))
        require((region(frame, 128, 48, 64, 64) == original_over) == full, "原版OVER完整性不同："+str(path))
        if full:
            active = {st["key"]: st for st in side["stamps"] if st["state"] == 2}
            require(set(active) == set(keys), "三行陣亡訊息不完整")
            for i, key in enumerate(keys):
                st = active[key]
                require(st["text"] == entries[key], "正式中文不同")
                require((st["x"], st["y"], st["cell_w"], st["cell_h"], st["font"]) == (0, 128+i*8, 8, 8, "cjk24"), "陣亡中文座標不同")
        art, _ = art_plane(frame, assets, None)
        text = render_text(side, fonts)
        expected = compose(rgb, (art if hd else empty, text if chinese else empty))
        candidates = [Path(str(path)+".png")]
        if path.name == "g-after-enter":
            phases = sorted(path.parent.glob(path.name+"-phase-*.png"))
            require(len(phases) == 30, "游標相位擷取數不同")
            candidates += phases
        phase_results = []
        for png in candidates:
            read(png, inputs)
            iw, ih, actual = rgba(png)
            require((iw, ih) == (FW, FH), "視窗尺寸不同")
            phase_results.append({"png": str(png), "mismatch": mismatch(expected, actual)})
        selected = next((r for r in phase_results if r["mismatch"] == 0), None)
        require(selected is not None, f"前端合成不符：{path} ({phase_results})")
        difference = selected["mismatch"]
        negative_over = mismatch(expected, compose(rgb, (empty, text if chinese else empty))) if hd and full else 0
        negative_text = mismatch(expected, compose(rgb, (art if hd else empty, empty))) if chinese and full else 0
        require(not (hd and full) or negative_over > 0, "省略人物負對照無效")
        require(not (chinese and full) or negative_text > 0, "省略中文負對照無效")
        return {"sample": str(path), "hd": hd, "chinese": chinese, "full_over": full,
                "composition_mismatch": difference, "negative_omit_over_pixels": negative_over,
                "negative_omit_text_pixels": negative_text, "selected_png": selected["png"],
                "captured_phases": phase_results}

    gui = Path("workplace/hd/over-frontend-v4-20261002")
    export = json.loads(read(gui/"original-frames.json", inputs))
    validate_export(export)
    require(len(export["results"]) == 10, "視窗保存狀態數不同")
    rows = []
    modes = [("a-hd-chinese", True, True), ("b-hd-english", True, False),
             ("c-original-english", False, False), ("d-original-chinese", False, True),
             ("e-hd-chinese", True, True), ("f-original-before-enter", False, True),
             ("g-after-enter", False, True), ("h-loaded-original", False, True), ("i-loaded-hd", True, True)]
    for label, hd, chinese in modes:
        meta = json.loads(read(gui/(label+".meta.json"), inputs))
        require(meta["language"] == ("zh" if chinese else "en"), "快速存檔語言不同")
        rows.append(check_image(gui/label, hd, chinese, label != "g-after-enter"))
    states = {Path(r["state"]).name: r for r in export["results"]}
    for label in ("h-loaded-original", "i-loaded-hd"):
        for key in ("area", "x", "y", "direction", "frame_sha256"):
            require(states[label+".state"][key] == states["before.state"][key], "F11原版位置或畫面不同")
    recording = json.loads(read(gui/"record.json", inputs))
    read(gui/"frontend.log", inputs)
    read(gui/"text.jsonl", inputs)
    require([(e["key"],e["down"]) for e in recording["events"]] == [("Return",True),("Return",False)], "前端組合鍵仍送到原版")

    step = Path("workplace/hd/over-pwstep-v3-20261002")
    step_export = json.loads(read(step/"original-frames.json", inputs))
    validate_export(step_export)
    require(len(step_export["results"]) == 4, "逐步保存狀態數不同")
    steps = {Path(r["state"]).name: r for r in step_export["results"]}
    step_rows = []
    for phase in ("full", "clear"):
        off, on = steps[phase+"-off.state"], steps[phase+"-on.state"]
        for key in ("regs", "steps", "cycles", "seed", "area", "x", "y", "direction", "frame_sha256", "rgb_sha256", "ram_below_a0000_sha256", "ram_bus_terminal_sha256"):
            require(off[key] == on[key], "逐步HD兩側原版不同："+phase+" "+key)
        require(read(step/(phase+"-off.state.xlate.json"), inputs) == read(step/(phase+"-on.state.xlate.json"), inputs), "HD影響逐步中文快照")
        for mode in ("off", "on"):
            step_rows.append(check_image(step/(phase+"-"+mode), mode == "on", True, phase == "full"))
    color_old, color_new = [json.loads(read(f"workplace/hd/over-frontend-colors-v{v}-20261002.json", inputs)) for v in (1, 2)]
    require(color_old["start"] == color_new["start"], "色盤重播起點不同")
    require(len(color_old["rows"]) == len(color_new["rows"]) == 83, "色盤原版取樣數不同")
    for before, after in zip(color_old["rows"], color_new["rows"]):
        for key in ("steps", "cycles", "indexed_sha256", "rgb_sha256", "scan"):
            require(before[key] == after[key], "色盤修正影響原版："+key)
    color_before = next(r for r in color_old["rows"] if r["steps"] == 670736610)
    color_after = next(r for r in color_new["rows"] if r["steps"] == 670736610)
    require([st["key"] for st in color_after["stamps"]] == keys, "色盤三行來源不同")
    require(all(st["fg"] == [0, 0, 0] for st in color_before["stamps"][:2]), "舊缺行負對照未重現")
    require(all(st["fg"] == [255, 255, 255] for st in color_after["stamps"]), "色盤恢復仍缺行")
    for file in ("tools/hd/verify_over_frontend.py", "tools/hd/over_frontend_check.sh", "tools/hd/over_pwstep_check.sh", "tools/hd/export_over_frontend.go", "tools/hd/verify_over_runtime.py", "tools/hd/verify_next_body_plane.py", "tools/pbl.py", "cmd/psychicwar/main.go", "cmd/pwstep/main.go", "apps/psychicwar/shift_input.go", "apps/psychicwar/shift_input_test.go", "worktrees/dosgolem/xlate/layer.go", "worktrees/dosgolem/xlate/layer_palette_test.go", "workplace/hd/psychicwar-over-ui-v3-20261002", "workplace/hd/pwstep-over-ui-v2-20261002", "workplace/hd/export-over-frontend-v1-20261002", "/orig/psychic-war/PW.EXE"):
        read(file, inputs)
    result = {"scope": "Linux正式視窗9種語言／HD／Enter／快速讀檔畫面及pwstep四組等價接續",
              "inputs_sha256": inputs, "gui_results": rows, "pwstep_results": step_rows,
              "f11_original_position_and_frame_equal": True, "pwstep_original_endpoints_equal": True,
              "palette_original_rows_equal": 83, "palette_restored_three_lines": True,
              "limits": "保存狀態接續與真正視窗按鍵抽測，非從開機或固定GUI時間亂數對拍；原版DAT、封包、真機、美術與全部sprite仍待驗。"}
    with output.open("x") as f:
        json.dump(result, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print("九張正式視窗、四組逐步合成不符0；F11位置／原版畫面及HD兩側逐步原版終點一致")


if __name__ == "__main__":
    main()
