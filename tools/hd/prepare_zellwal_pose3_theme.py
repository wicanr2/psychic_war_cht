"""研究038 §92：準備及選入已審查ENEMY03 #3，Docker專用。"""
import argparse
import hashlib
import json
from pathlib import Path
import shutil


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--select", action="store_true")
    args = parser.parse_args()
    repo = Path("/src")
    review = repo / "workplace/hd/art-zellwal-group1-review-v1-20261003"
    base = repo / "workplace/hd/theme-group1-pose3-v1-20261003"
    candidate = review / "selected-theme"
    target = repo / "workplace/hd/theme-zellwal-pose3-v1-20261003" if args.select else candidate
    assert not target.exists()
    assert (review.stat().st_uid, review.stat().st_gid) == (1000, 1000)
    art = json.loads((review / "art-review.json").read_text())
    assert art["formal_art_accepted"] and art["accepted_images"] == [3]
    inputs = {}
    for path, expected in art["inputs_sha256"].items():
        p = Path(path)
        assert sha(p) == expected, path
        inputs[path] = expected
    manifest = json.loads((base / "manifest.json").read_text())
    assert len(manifest["entries"]) == 25
    manifest["entries"].append({"pbl": "ENEMY03.PBL", "image": 3, "at": [32, 152],
                                "png": "ENEMY03-03.png", "kind": "redraw"})
    if args.select:
        rows = []
        for branch in ("selected-attack", "selected-death"):
            p = review / branch / "independent.json"
            proof = json.loads(p.read_text())
            rows.extend(proof["rows"])
            inputs[str(p)] = sha(p)
            for row in proof["rows"]:
                assert all(row[key] == 0 for key in ("whole_plane_difference",
                    "whole_composition_difference", "text_difference", "reload_difference"))
        assert len(rows) == 33 and sum(bool(row["new_enemy_cells"]) for row in rows) == 2
        assert json.loads((candidate / "manifest.json").read_text()) == manifest
    target.mkdir()
    outputs = {}
    for entry in manifest["entries"]:
        source = (candidate / entry["png"] if args.select else
                  repo / "workplace/hd/redraw/ENEMY03-group1-v3-frame-03-20261003.png"
                  if entry["pbl"] == "ENEMY03.PBL" else base / entry["png"])
        dest = target / entry["png"]
        shutil.copyfile(source, dest)
        assert sha(source) == sha(dest) and dest.stat().st_uid == 1000
        outputs[str(dest)] = sha(dest)
        inputs[str(source)] = sha(source)
    dest = target / "manifest.json"
    dest.write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n")
    outputs[str(dest)] = sha(dest)
    inputs[str(Path(__file__))] = sha(Path(__file__))
    inputs[str(base / "manifest.json")] = sha(base / "manifest.json")
    record = {"scope": "local26; accepts ENEMY03 #3 only; poses4/5 incomplete",
              "entries": 26, "base": str(base), "accepted_images": [3],
              "changed_assets": ["ENEMY03-03.png"], "inputs_sha256": inputs,
              "outputs_sha256": outputs, "normal_gate_passed": args.select,
              "limits": "specified normal state paths; overlap, GUI/DAT, fullsprites and release incomplete"}
    (target / "selection.json").write_text(json.dumps(record, ensure_ascii=False, indent=2) + "\n")
    if args.select:
        (review / "selection.json").write_text(json.dumps(record, ensure_ascii=False, indent=2) + "\n")
    print(target, "26 entries; selected" if args.select else "26 entries; unselected candidate")


if __name__ == "__main__":
    main()
