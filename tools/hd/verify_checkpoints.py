"""由原版RGB及GOLEMFNT獨立核對四檢查點；入口研究038 §64。"""
import json
from pathlib import Path

from verify_next_body_normal import load_font, render_text, read
from verify_next_body_plane import require, rgba, sha, FW, FH
from verify_over_frontend import compose, mismatch


def main():
    root = Path("workplace/hd/checkpoints-runtime-v1-20261002")
    output = Path("workplace/hd/checkpoints-verification-v1-20261002.json")
    require(not output.exists(), "拒絕覆寫")
    inputs = {}
    run = json.loads(read(root / "execution.json", inputs))
    export = json.loads(read(root / "original-frames.json", inputs))
    for doc in [run, export]:
        for name, expected in doc["inputs_sha256"].items():
            require(sha(read(name, inputs)) == expected, "來源已變更：" + name)
    require(len(run["results"]) == len(export["results"]) == 17, "實際輸出數量不同")
    original = {row["state"]: row for row in export["results"]}
    rows = {(r["mode"], r["checkpoint"], r["language"]): r for r in run["results"]}
    fonts = {n: load_font("font/" + n + ".golemfnt", inputs) for n in ["cjk24", "cjk16"]}
    empty = bytes(FW * FH * 4)
    results = []
    for checkpoint in ["01-title", "03-protection", "05-select", "07-first-play"]:
        for language in ["original", "chinese"]:
            control, current = [rows[m, checkpoint, language] for m in ["control", "current"]]
            before, after = [original[r["state"]] for r in [control, current]]
            for field in ["regs", "steps", "cycles", "seed", "frame_sha256", "rgb_sha256", "ram_below_a0000_sha256", "ram_bus_terminal_sha256"]:
                require(before[field] == after[field], "未選HD改變原版：" + checkpoint + " " + language + " " + field)
            read(control["state"], inputs); read(current["state"], inputs)
            expected_raw = compose(read(current["state"] + ".rgb.bin", inputs), ())
            require(sha(read(current["state"] + ".frame", inputs)) == after["frame_sha256"], "原版frame匯出不同")
            text = empty
            if language == "chinese":
                side = json.loads(read(current["state"] + ".xlate.json", inputs))
                require(read(control["state"] + ".xlate.json", inputs) == read(current["state"] + ".xlate.json", inputs), "HD接合改變中文Layer")
                text = render_text(side, fonts)
                require(checkpoint == "01-title" or any(st["state"] == 2 and any("\u4e00" <= ch <= "\u9fff" for ch in st["text"]) for st in side["stamps"]), "中文分支沒有正常中文字面：" + checkpoint)
            expected = compose(read(current["state"] + ".rgb.bin", inputs), (text,))
            read(control["png"], inputs); read(current["png"], inputs)
            w, h, before_pixels = rgba(control["png"])
            require((w, h) == (FW, FH), "控制PNG尺寸不同")
            w, h, current_pixels = rgba(current["png"])
            require((w, h) == (FW, FH), "現行PNG尺寸不同")
            require(mismatch(before_pixels, current_pixels) == 0, "加入HD前後未選主題畫面不同")
            require(mismatch(expected, current_pixels) == 0, "原版RGB／字型獨立合成不同")
            omit = mismatch(expected, expected_raw)
            require(language != "chinese" or checkpoint == "01-title" or omit > 0, "省略中文負對照無效")
            results.append({"checkpoint": checkpoint, "language": language, "control_current_mismatch": 0, "independent_composition_mismatch": 0, "original_state_equal": True, "negative_omit_text_pixels": omit, "png": current["png"], "steps": after["steps"], "cycles": after["cycles"]})
    plain = rows["current", "07-first-play", "chinese"]
    hd = rows["current-hd", "07-first-play", "chinese"]
    a, b = original[plain["state"]], original[hd["state"]]
    for field in ["regs", "steps", "cycles", "seed", "frame_sha256", "rgb_sha256", "ram_below_a0000_sha256", "ram_bus_terminal_sha256"]:
        require(a[field] == b[field], "選主題負對照改變原版：" + field)
    require(read(plain["state"] + ".xlate.json", inputs) == read(hd["state"] + ".xlate.json", inputs), "選主題改變中文Layer")
    read(hd["png"], inputs)
    w, h, pixels_hd = rgba(hd["png"])
    require((w, h) == (FW, FH), "HD負對照尺寸不同")
    negative_hd = mismatch(rgba(plain["png"])[2], pixels_hd)
    require(negative_hd > 0, "真正選HD的敏感性負對照無效")
    changed = bytearray(rgba(plain["png"])[2]); changed[0] ^= 1
    require(mismatch(changed, rgba(plain["png"])[2]) == 1, "單像素比較負對照無效")
    for file in [__file__, "tools/hd/verify_next_body_normal.py", "tools/hd/verify_next_body_plane.py", "tools/hd/verify_over_frontend.py", "tools/hd/verify_over_runtime.py", "tools/pbl.py"]:
        read(file, inputs)
    with output.open("x") as f:
        json.dump({"inputs_sha256": inputs, "results": results, "pngs_verified": 16, "negative_selected_hd_pixels": negative_hd, "negative_single_pixel_mismatch": 1, "limits": "024 §6.1有限逐步工具回歸。歷史main與現行程式共用目前依賴，未驗整個歷史工具鏈、真正視窗、封包、幀率或完整sprite美術。標題Logo保留原版。"}, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print("四檢查點八組、16 PNG回歸／獨立合成不符0，選HD負對照", negative_hd)


if __name__ == "__main__":
    main()
