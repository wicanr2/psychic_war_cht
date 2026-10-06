"""研究038 §56：獨立核對正常敏頓攻擊的PBL來源與一般貼圖，不證明HD完成。"""
import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path
import sys

sys.path.insert(0, "tools")
import pbl


def digest(data):
    return hashlib.sha256(data).hexdigest()


def paint(before, raw, rect, mode):
    x, y, w, h = rect
    assert len(before) == 64000 and len(raw) == w*h//2
    assert 0 <= x <= 320-w and 0 <= y <= 200-h and mode in (0, 1)
    expected = bytearray(before)
    for row in range(h):
        for col in range(w):
            at = row*w+col
            pixel = raw[at//2] >> 4 if at % 2 == 0 else raw[at//2] & 15
            dst = (y+row)*320+x+col
            expected[dst] = before[dst] ^ pixel if mode == 1 else pixel
    return bytes(expected)


def verify(probe):
    receipt_path = Path(probe + ".json")
    receipt = json.loads(receipt_path.read_text())
    assert receipt["seed_before"] == "A48C"
    assert receipt["end_steps"] == 420000000
    assert receipt["observed_end"] == receipt["unobserved_end"] == receipt["prior_end"]
    source_files = {str(receipt_path): digest(receipt_path.read_bytes())}
    for f, wanted in receipt["inputs_sha256"].items():
        assert digest(Path(f).read_bytes()) == wanted, f
        source_files[f] = wanted
    assets = []
    archive_hashes = {}
    for f in sorted(Path("/orig/psychic-war").glob("*.PBL")):
        data = f.read_bytes()
        archive_hashes[str(f)] = digest(data)
        for index, offset, _ in pbl.images(data):
            w, h, pixels = pbl.decode(data, offset)
            if (w, h) not in ((16, 16), (24, 32)):
                continue
            packed = bytes(pixels[i] << 4 | pixels[i+1] for i in range(0, len(pixels), 2))
            assets.append({"archive": f.stem, "index": index, "w": w, "h": h,
                           "offset": offset, "packed": packed})
    complete, pairs = {}, {}
    for a in assets:
        key = (a["w"], a["h"], digest(a["packed"]))
        complete.setdefault(key, []).append(f'{a["archive"]}#{a["index"]}')
    for slot, a in enumerate(assets):
        for b in assets[slot+1:]:
            if a["archive"] != b["archive"] or (a["w"], a["h"]) != (b["w"], b["h"]):
                continue
            delta = bytes(x ^ y for x, y in zip(a["packed"], b["packed"]))
            key = (a["w"], a["h"], digest(delta))
            pairs.setdefault(key, []).append([f'{a["archive"]}#{a["index"]}',
                                              f'{b["archive"]}#{b["index"]}'])
    events = []
    known, unknown, ambiguous, body, tile = 0, [], [], 0, 0
    tile_families, tile_source_counts, tile_positions = Counter(), Counter(), set()
    negative = None
    previous_return = -1
    for number, e in enumerate(receipt["events"]):
        assert e["entry_step"] > previous_return and e["return_step"] > e["entry_step"]
        previous_return = e["return_step"]
        blobs = {}
        for label in ("before", "after", "source"):
            f = Path(e[label])
            blobs[label] = f.read_bytes()
            assert digest(blobs[label]) == e[label + "_sha256"]
            assert (f.stat().st_uid, f.stat().st_gid) == (1000, 1000)
            source_files[str(f)] = digest(blobs[label])
        before, after, raw = blobs["before"], blobs["after"], blobs["source"]
        assert len(after) == 64000
        x, y, w, h = e["rect"]
        r = e["entry_regs"]
        assert [r["CX"] >> 8 << 2, (r["CX"] & 255) << 2,
                r["DX"] >> 8 << 3, (r["DX"] & 255) << 3] == e["rect"]
        assert r["AX"] & 255 == e["al"]
        assert paint(before, raw, e["rect"], e["al"]) == after, number
        assert sum(a != b for a, b in zip(before, after)) == e["changed_pixels"]
        assert e["source_mismatch"] == e["outside_changed_pixels"] == 0
        key = (w, h, digest(raw))
        labels = complete.get(key, [])
        delta_pairs = pairs.get(key, [])
        choices = len(labels) + len(delta_pairs)
        if choices:
            known += 1
        else:
            unknown.append(number)
        if choices > 1:
            ambiguous.append(number)
        row = {"event": number, "entry_step": e["entry_step"],
               "return_step": e["return_step"], "rect": e["rect"], "al": e["al"],
               "full_images": labels, "unordered_xor_pairs": delta_pairs,
               "unique_source": choices == 1, "source_mismatch": 0, "outside_mismatch": 0}
        events.append(row)
        if (w, h) == (16, 16):
            tile += 1
            tile_positions.add((x, y))
            source_label = ",".join(labels) + ";" + ",".join("^".join(p) for p in delta_pairs)
            tile_source_counts[source_label] += 1
            for name in set(labels + [name for pair in delta_pairs for name in pair]):
                tile_families[name] += 1
            if negative is None and e["changed_pixels"] > 0 and x+4 <= 320-w:
                damaged = bytearray(raw)
                damaged[0] ^= 1
                damaged_diff = sum(a != b for a, b in zip(paint(before, damaged, e["rect"], e["al"]), after))
                shifted_diff = sum(a != b for a, b in zip(paint(before, raw, [x+4,y,w,h], e["al"]), after))
                assert damaged_diff > 0 and shifted_diff > 0
                negative = {"event": number, "wrong_source_pixels": damaged_diff,
                            "shifted_four_pixels": shifted_diff}
        else:
            body += 1
    assert events and negative is not None
    end = Path(probe + "-end.frame")
    assert digest(end.read_bytes()) == receipt["observed_end"]["Frame"]
    source_files[str(end)] = digest(end.read_bytes())
    source_files[probe + "-end.state"] = digest(Path(probe + "-end.state").read_bytes())
    return {"scope": "正常14-minton1一般貼圖來源，非完整HD或全部能力驗收",
            "python_version": sys.version.split()[0], "seed_before": "A48C",
            "address_space": receipt["address_space"], "events": events,
            "general_event_count": len(events), "tile_event_count": tile,
            "body_event_count": body, "known_source_events": known,
            "unknown_source_events": unknown, "ambiguous_source_events": ambiguous,
            "tile_families": dict(sorted(tile_families.items())),
            "tile_source_counts": dict(sorted(tile_source_counts.items())),
            "tile_positions": sorted(tile_positions), "negative": negative,
            "source_mismatch": 0, "full_frame_mismatch": 0,
            "observer_control_prior_end_equal": True,
            "original_archives_sha256": archive_hashes, "files_sha256": source_files,
            "limits": "XOR配對未辨識方向；內建遮罩、完整重疊場景、傷害／能力語意、HD與GUI未驗"}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--probe", required=True)
    parser.add_argument("--out", required=True)
    args = parser.parse_args()
    assert not Path(args.out).exists(), "拒絕覆寫"
    result = verify(args.probe)
    with Path(args.out).open("x") as file:
        json.dump(result, file, ensure_ascii=False, indent=2)
        file.write("\n")
    print(json.dumps({k: result[k] for k in ("general_event_count", "tile_event_count",
        "body_event_count", "known_source_events", "unknown_source_events", "ambiguous_source_events",
        "tile_families", "negative", "observer_control_prior_end_equal")}, ensure_ascii=False))
