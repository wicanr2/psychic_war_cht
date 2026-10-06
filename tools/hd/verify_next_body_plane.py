"""以原版PBL及PNG獨立核對32張8×8主題圖面；入口研究038 §48。"""
import argparse
import hashlib
import json
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent.parent))
import pbl

W, H, S = 320, 200, 3
FW, FH = W * S, H * S


def require(ok, message):
    if not ok:
        raise ValueError(message)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def rgba(path):
    w, h, channels, rows, palette = pbl.read_png(path)
    require(channels in (3, 4) and palette is None, "PNG通道不符")
    if channels == 4:
        return w, h, b"".join(rows)
    result = bytearray()
    for row in rows:
        for i in range(0, len(row), 3):
            result.extend(row[i:i + 3])
            result.append(255)
    return w, h, bytes(result)


def region(frame, x, y, w, h):
    return b"".join(frame[(y + yy) * W + x:(y + yy) * W + x + w] for yy in range(h))


def over(dst, index, source):
    alpha = source[3]
    if alpha == 0:
        return
    if alpha == 255:
        dst[index:index + 4] = source
        return
    old_alpha = dst[index + 3]
    denominator = alpha * 255 + old_alpha * (255 - alpha)
    for c in range(3):
        dst[index + c] = (source[c] * alpha * 255 + dst[index + c] * old_alpha * (255 - alpha) + denominator // 2) // denominator
    dst[index + 3] = (denominator + 127) // 255


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", required=True)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    output = pathlib.Path(args.out)
    require(args.check or not output.exists(), "拒絕覆寫")
    root = pathlib.Path("workplace/hd/theme-next-bodies-v1-20261001")
    manifest = json.loads((root / "manifest.json").read_text())
    receipt_path = pathlib.Path("workplace/hd/next-body-runtime-v1-20261001.json")
    runtime = json.loads(receipt_path.read_text())
    inputs = {str(receipt_path): sha(receipt_path.read_bytes()), str(root / "manifest.json"): sha((root / "manifest.json").read_bytes())}
    base = bytearray(W * H)
    decoded = {}
    for name in ("SCREEN.PBL", "MENU.PBL", "ALLY.PBL", "ENEMY00.PBL"):
        path = pathlib.Path("/orig/psychic-war") / name
        source = path.read_bytes()
        inputs[str(path)] = sha(source)
        for image, offset, _ in pbl.images(source):
            w, h, px = pbl.decode(source, offset)
            decoded[name, image] = (w, h, bytes(px))
            if name in ("SCREEN.PBL", "MENU.PBL"):
                x, y = (0, image * 40) if name == "SCREEN.PBL" else (160, 4)
                for yy in range(h):
                    base[(y + yy) * W + x:(y + yy) * W + x + w] = bytes(px[yy * w:(yy + 1) * w])
    assets = []
    for entry in manifest["entries"]:
        name, image = entry["pbl"], entry["image"]
        w, h, source = decoded[name, image]
        x, y = entry["at"]
        path = root / entry["png"]
        pw, ph, pixels = rgba(path)
        require((pw, ph) == (w * S, h * S), "候選尺寸不符")
        inputs[str(path)] = sha(path.read_bytes())
        reference = bytearray(base)
        sprite = name in ("ALLY.PBL", "ENEMY00.PBL")
        if sprite:
            for yy in range(h):
                reference[(y + yy) * W + x:(y + yy) * W + x + w] = source[yy * w:(yy + 1) * w]
        assets.append((entry, w, h, source, pixels, reference, sprite))
    rows = []
    for result in runtime["results"]:
        label, event = result["label"], result["event"]
        frame_path = pathlib.Path(f"workplace/hd/next-body-{label}-v2-20261001-event{event:02d}-after.frame")
        frame = frame_path.read_bytes()
        require(len(frame) == W * H, "原版畫面尺寸不符")
        inputs[str(frame_path)] = sha(frame)
        expected = bytearray(FW * FH * 4)
        anchor = region(frame, 0, 0, 320, 40) == region(base, 0, 0, 320, 40)
        require(anchor, "正常畫面錨點不同")
        selected = []
        body_cells = []
        for entry, w, h, source, pixels, reference, sprite in assets:
            x, y = entry["at"]
            if sprite and region(frame, x, y, w, h) != source:
                continue
            if entry["pbl"] == "ENEMY00.PBL":
                selected.append(entry["image"])
            for cy in range(y // 8 * 8, (y + h + 7) // 8 * 8, 8):
                for cx in range(x // 8 * 8, (x + w + 7) // 8 * 8, 8):
                    ref = region(reference, cx, cy, 8, 8)
                    if region(frame, cx, cy, 8, 8) != ref or (sprite and not any(ref)):
                        continue
                    if entry["pbl"] == "ENEMY00.PBL":
                        body_cells.append((cx, cy))
                    for yy in range(max(cy, y) * S, min(cy + 8, y + h) * S):
                        start_x, end_x = max(cx, x) * S, min(cx + 8, x + w) * S
                        for xx in range(start_x, end_x):
                            pi = 4 * ((yy - y * S) * w * S + xx - x * S)
                            over(expected, 4 * (yy * FW + xx), pixels[pi:pi + 4])
        path = pathlib.Path(result["plane_png"])
        rw, rh, actual = rgba(path)
        require((rw, rh) == (FW, FH) and sha(path.read_bytes()) == result["plane_png_sha256"], "正式圖面來源不符")
        inputs[str(path)] = sha(path.read_bytes())
        mismatch = sum(actual[i:i + 4] != expected[i:i + 4] for i in range(0, len(actual), 4))
        require(mismatch == 0, "正式8×8圖面與獨立期望不同")
        negative = sum(actual[4 * (yy * FW + xx) + 3] != 0 for cx, cy in body_cells for yy in range(cy * S, (cy + 8) * S) for xx in range(cx * S, (cx + 8) * S))
        require(not selected or negative > 0, "新姿勢省略負對照無效")
        require(result["with_hd"] == result["without_hd"] and result["reload_and_toggle_equal"], "原版接續／HD切換不符")
        rows.append({"label": label, "event": event, "selected_full_enemy": selected, "plane_mismatch_pixels": mismatch, "negative_omit_enemy_pixels": negative})
    inputs[__file__] = sha(pathlib.Path(__file__).read_bytes())
    doc = {"scope": "32份正常來源state的正式圖面，獨立原版PBL與候選PNG推導8×8期望", "inputs_sha256": inputs, "results": rows, "limits": "4份完整姿勢顯示，28份重疊來源載回後回退；非全程HD玩家驗收，未驗候選美術與中文覆繪"}
    if args.check:
        require(json.loads(output.read_text()) == doc, "獨立收據重生不同")
    else:
        with output.open("x") as f:
            json.dump(doc, f, ensure_ascii=False, indent=2)
            f.write("\n")
    print("32份圖面與獨立8×8期望不符0；完整姿勢", sum(bool(r["selected_full_enemy"]) for r in rows), "；重疊回退", sum(not r["selected_full_enemy"] for r in rows))


if __name__ == "__main__":
    main()
