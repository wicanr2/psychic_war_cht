"""原版保存畫面的背景錨點獨立來源核對；研究038 §67。"""
import json
from pathlib import Path

from verify_next_body_plane import W, H, pbl, region, require, sha
from verify_over_runtime import read


def main():
    output = Path("workplace/hd/background-anchor-source-v1-20261003.json")
    require(not output.exists(), "拒絕覆寫")
    inputs, base = {}, bytearray(W * H)
    for name in ("SCREEN.PBL", "MENU.PBL"):
        data = read(Path("/orig/psychic-war") / name, inputs)
        for image, offset, _ in pbl.images(data):
            w, h, pixels = pbl.decode(data, offset)
            x, y = (0, image * 40) if name == "SCREEN.PBL" else (160, 4)
            for yy in range(h):
                base[(y + yy) * W + x:(y + yy) * W + x + w] = pixels[yy * w:(yy + 1) * w]
    want = region(base, 0, 0, 160, 40)
    old_want = region(base, 0, 0, 320, 40)
    paths = {}
    replay = json.loads(read("replay/title-to-first-save.json", inputs))
    # 實際frame來自既有原版重播，不讀state旗標推定顯示。
    for segment in replay["segments"]:
        path = Path("workplace/states") / (segment["name"] + ".frame")
        require(path.is_file(), "原版檢查點不存在：" + str(path))
        paths[str(path)] = "original-checkpoint"
    for receipt in ("room-runtime-v1-20261003.json", "next-body-normal-minton-v3-20261001.json",
                    "next-body-normal-kasuruji-v3-20261001.json",
                    "sivad-normal-v1-20261002.json", "over-runtime-v1-20261002.json"):
        path = Path("workplace/hd") / receipt
        require(path.is_file(), "原版收據不存在：" + str(path))
        doc = json.loads(read(path, inputs))
        for sample in doc.get("samples", []):
            prefix = sample.get("prefix", sample.get("Name"))
            if prefix:
                frame = Path(prefix + ".frame")
                require(frame.is_file(), "取樣frame不存在：" + str(frame))
                paths[str(frame)] = receipt
    rows = []
    for path, source in sorted(paths.items()):
        frame = read(path, inputs)
        require(len(frame) == W * H, "原版畫面尺寸不符")
        small = region(frame, 0, 0, 160, 40)
        old = region(frame, 0, 0, 320, 40)
        points = [(x, y) for y in range(40) for x in range(320) if frame[y * W + x] != base[y * W + x]]
        rows.append({"frame": path, "source": source, "candidate_match": small == want,
                     "old_match": old == old_want,
                     "candidate_mismatch_pixels": sum(a != b for a, b in zip(small, want)),
                     "old_mismatch_pixels": len(points),
                     "outside_menu_mismatch_pixels": sum(not (160 <= x < 248 and 4 <= y < 76) for x, y in points)})
    indexed = {row["frame"]: row for row in rows}
    for name in ("01-title", "03-protection"):
        require(not indexed["workplace/states/" + name + ".frame"]["candidate_match"], "候選錨點誤認開場：" + name)
    for name in ("05-select", "06-name", "07-first-play"):
        row = indexed["workplace/states/" + name + ".frame"]
        require(row["candidate_match"] and row["old_match"], "原版背景既有匹配條件不符：" + name)
    for sample in (8, 9):
        row = indexed[f"workplace/hd/room-runtime-v1-20261003-sample{sample:02d}.frame"]
        require(row["candidate_match"] and row["old_mismatch_pixels"] == 588 and row["outside_menu_mismatch_pixels"] == 0,
                "治療回退來源與既有證據不符")
    changed = bytearray(want)
    changed[0] ^= 1
    require(changed != want, "錨點單像素負對照無效")
    read(__file__, inputs)
    read("tools/pbl.py", inputs)
    read("tools/hd/verify_next_body_plane.py", inputs)
    with output.open("x") as f:
        json.dump({"scope": "候選左側160×40與舊全寬錨點的原版保存畫面辨識，非HD或GUI驗收",
                   "inputs_sha256": inputs, "candidate_rect": [0, 0, 160, 40], "old_rect": [0, 0, 320, 40],
                   "reference_sha256": sha(want), "results": rows, "negative_single_pixel": 1,
                   "limits": "有限原版抽樣，不推定所有場景均穩定；未改正式loader或候選主題"},
                  f, ensure_ascii=False, indent=2)
        f.write("\n")
    print("原版錨點來源", len(rows), "份，候選匹配", sum(r["candidate_match"] for r in rows),
          "，舊匹配", sum(r["old_match"] for r in rows), flush=True)


if __name__ == "__main__":
    main()
