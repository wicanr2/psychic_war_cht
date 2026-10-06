"""研究038 §90：獨立核對正常Zellwal路線的原版來源與完整前後畫面。"""
import argparse
import hashlib
import json
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))
import pbl


def sha(data):
    return hashlib.sha256(data).hexdigest()


def region(frame, x, y, w, h):
    assert len(frame) == 64000
    return b"".join(frame[row * 320 + x:row * 320 + x + w] for row in range(y, y + h))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory")
    args = parser.parse_args()
    root = Path(args.directory)
    inputs = {}

    def read(path):
        path = Path(path)
        data = path.read_bytes()
        inputs[str(path)] = sha(data)
        return data

    doc = json.loads(read(root / "observation.json"))
    assert doc["observed_end"] == doc["control_end"]
    for name, expected in doc["inputs_sha256"].items():
        assert sha(read(name)) == expected, name
    end = read(root / "end.frame")
    assert sha(end) == doc["observed_end"]["Frame"]
    for name in ("observed-end.state", "control-end.state"):
        read(root / name)
    poses = {}
    for name in ("ALLY.PBL", *(f"ENEMY{i:02d}.PBL" for i in range(12))):
        data = read(Path("/orig/psychic-war") / name)
        entries = pbl.images(data)
        assert len(entries) == (31 if name == "ALLY.PBL" else 30)
        for image, offset, _ in entries:
            w, h, pixels = pbl.decode(data, offset)
            poses[f"{name}:{image}"] = (w, h, bytes(pixels))

    def matches(pixels, w, h):
        return sorted(k for k, (pw, ph, p) in poses.items() if (pw, ph) == (w, h) and p == pixels)

    rows, logical = [], {}
    for number, event in enumerate(doc["events"]):
        before, after, raw = [read(event[k]) for k in ("Before", "After", "Source")]
        read(event["State"])
        x, y, w, h = [event[k] for k in ("X", "Y", "W", "H")]
        assert event["Entry"] < event["Return"]
        assert x in (32, 264) and y == 152 and 0 < w <= 320 - x and 0 < h <= 48
        assert len(raw) == w * h // 2
        pixels = bytes(v for byte in raw for v in (byte >> 4, byte & 15))
        a, b = region(before, x, y, w, h), region(after, x, y, w, h)
        origins, targets, raw_full = matches(a, w, h), matches(b, w, h), matches(pixels, w, h)
        assert origins == event["FullBefore"] and targets == event["FullAfter"] and raw_full == event["RawFull"]
        mode = event["Regs"]["AX"] & 255
        assert mode in (0, 1), "未知原版貼圖模式"
        expected = bytearray(before)
        for row in range(h):
            for col in range(w):
                at, value = (y + row) * 320 + x + col, pixels[row * w + col]
                expected[at] = value if mode == 0 else before[at] ^ value
        assert bytes(expected) == after, "原版完整前後畫面不符"
        key = (x, y, w, h)
        previous = logical.get(key, origins)
        if mode == 0:
            transitions = [(None, target) for target in raw_full]
        else:
            transitions = [(origin, target) for origin in previous
                           for target, (pw, ph, px) in poses.items()
                           if (pw, ph) == (w, h)
                           and bytes(a ^ b for a, b in zip(poses[origin][2], px)) == pixels]
        candidates = sorted(set(target for _, target in transitions))
        if targets and candidates:
            assert all(target in candidates for target in targets)
        logical[key] = candidates
        wrong_raw = bytearray(raw)
        wrong_raw[0] ^= 0x10
        changed_pixels = bytes(v for byte in wrong_raw for v in (byte >> 4, byte & 15))
        wrong_result = changed_pixels if mode == 0 else bytes(a ^ b for a, b in zip(a, changed_pixels))
        negative_raw = sum(a != b for a, b in zip(wrong_result, b))
        assert negative_raw == 1
        wrong_mode = bytes(a ^ b for a, b in zip(a, pixels)) if mode == 0 else pixels
        negative_mode = sum(a != b for a, b in zip(wrong_mode, b))
        if mode == 1:
            assert negative_mode > 0
        rows.append({"event": number, "entry": event["Entry"], "return": event["Return"],
                     "rect": [x, y, w, h], "al": mode, "full_before": origins,
                     "full_after": targets, "raw_full": raw_full, "transitions": transitions,
                     "source_targets": candidates, "source_level": "confirmed" if len(candidates) == 1 else "unknown",
                     "complete_frame_difference": 0, "negative_raw_pixels": negative_raw,
                     "negative_wrong_mode_pixels": negative_mode})
    read(__file__)
    read("tools/pbl.py")
    result = {"scope": "original normal Zellwal route; independent PBL and entire 64000-byte before/after frames",
              "route": doc["route"], "seed_before": doc["seed_before"],
              "start_location": doc["start_location"], "end_location": doc["end_location"],
              "events": rows, "total_sprite_calls": doc["total_sprite_calls"],
              "battle_entries": doc["battle_entries"], "battle_returns": doc["battle_returns"],
              "end_fingerprint_equal": True, "inputs_sha256": inputs,
              "limits": "no HD, art, translation, full serialized-state or GUI acceptance; no event means no new source proof"}
    with (root / "independent.json").open("x") as f:
        json.dump(result, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print(root.name, "independent source events", len(rows), [r["source_targets"] for r in rows])


if __name__ == "__main__":
    main()
