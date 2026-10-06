"""研究038 §92：唯讀核對ENEMY03三姿勢參照與候選，Docker專用。

只量測原始像素、輪廓與主要色區，不修改圖片，不自行判定美術接受。
"""
import hashlib
import json
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import pbl


def sha(data):
    return hashlib.sha256(data).hexdigest()


def rgb(path):
    w, h, ch, rows, palette = pbl.read_png(path)
    assert ch in (3, 4) and palette is None, path
    pixels = [tuple(row[x * ch:x * ch + 3]) for row in rows for x in range(w)]
    return w, h, pixels


def bounds(points):
    if not points:
        return None
    return [min(x for x, y in points), min(y for x, y in points),
            max(x for x, y in points), max(y for x, y in points)]


def metrics(pixels):
    groups = {k: [] for k in ("ink", "left_pad", "right_pad", "head_yellow", "right_hook")}
    stripes = {k: 0 for k in ("yellow", "white", "cyan")}
    for y in range(96):
        for x in range(72):
            r, g, b = pixels[y * 72 + x]
            cyan = r < 150 and g > 160 and b > 160
            white = min(r, g, b) > 170 and max(r, g, b) - min(r, g, b) < 70
            yellow = r > 170 and g > 160 and b < 150
            if max(r, g, b) > 60:
                groups["ink"].append((x, y))
            if (cyan or white) and 27 <= y < (63 if x < 36 else 72):
                groups["left_pad" if x < 36 else "right_pad"].append((x, y))
            if yellow and y < 36:
                groups["head_yellow"].append((x, y))
            if x >= 39 and 75 <= y < 87 and r > 120 and r > g * 1.5:
                groups["right_hook"].append((x, y))
            if 27 <= x < 36 and 63 <= y < 85:
                for key, match in (("yellow", yellow), ("white", white), ("cyan", cyan)):
                    stripes[key] += int(match)
    return {k: {"bounds_inclusive": bounds(v), "pixels": len(v)} for k, v in groups.items()} | {
        "lower_stripe_color_pixels": stripes}


def main():
    root = Path("/src")
    out = Path(sys.argv[1])
    assert not out.exists(), out
    d = root / "workplace/hd/redraw"
    original_path = Path("/orig/psychic-war/ENEMY03.PBL")
    original = original_path.read_bytes()
    assert sha(original) == "ad6e8183bbf6c6fc258693b1f0ac726593feff5d052b5da18ac715cc2e63e5c1"
    inputs = {str(original_path): sha(original)}
    rows = []
    for image in (3, 4, 5):
        _, offset, _ = pbl.images(original)[image]
        w, h, indices = pbl.decode(original, offset)
        assert (w, h) == (24, 32)
        ref = d / ("ENEMY03-group1-v2-reference-%02d-20261003.png" % image)
        rw, rh, reference = rgb(ref)
        expected = [pbl.EGA[indices[(y // 18) * w + x // 18]]
                    for y in range(h * 18) for x in range(w * 18)]
        assert (rw, rh) == (432, 576) and reference == expected
        changed = list(reference)
        changed[0] = (changed[0][0] ^ 1, *changed[0][1:])
        negative = sum(a != b for a, b in zip(changed, expected))
        assert negative == 1
        candidate = d / ("ENEMY03-group1-v3-frame-%02d-20261003.png" % image)
        cw, ch, pixels = rgb(candidate)
        assert (cw, ch) == (72, 96)
        nearest = [pbl.EGA[indices[(y // 3) * w + x // 3]]
                   for y in range(h * 3) for x in range(w * 3)]
        for p in (ref, candidate, Path(__file__)):
            inputs[str(p)] = sha(p.read_bytes())
        rows.append({"image": image, "file_offset": offset, "reference_pixels": rw * rh,
                     "reference_mismatch": 0, "negative_reference_pixel": negative,
                     "source_indices_sha256": sha(indices), "origin": [32, 152], "size": [72, 96],
                     "original_metrics": metrics(nearest), "candidate_metrics": metrics(pixels)})
    result = {"scope": "exact original references and read-only candidate measurements; no acceptance implied",
              "tool_versions": {"python": sys.version.split()[0], "decoder": "tools/pbl.py"},
              "source_basis": "PBL file offsets and decoded original color indices, no executable addresses",
              "inputs_sha256": inputs, "rows": rows,
              "limits": "RGB groups are navigation evidence; smoothing changes individual color boundaries; visual review and normal runtime evidence required"}
    out.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps(rows, ensure_ascii=False))


if __name__ == "__main__":
    main()
