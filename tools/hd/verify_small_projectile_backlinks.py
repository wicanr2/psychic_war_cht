"""研究038 §56.3：檢查小圖塊來源回填標記，缺標記即拒絕。"""
import hashlib
import json
from pathlib import Path
import sys

sys.path.insert(0, "tools")
import pbl

MARK = "【HD-SMALL-02】"
ARCHIVE_SHA = "8241b0ea73b1e7402b13adc434f10a5b1288f88f01fcfc3ba5710e4923fe6067"
OFFSETS = {21: 0x1924, 22: 0x19A1, 23: 0x1A1B}


def sha(p):
    return hashlib.sha256(Path(p).read_bytes()).hexdigest()


def check(spec, research):
    older = research[research.index("## 20."):research.index("## 21.")]
    current = research[research.index("## 56."):]
    effect = spec[spec.index("### 1.5"):spec.index("### 1.6")]
    assert MARK in older and "§56" in older and "0161:8705" in older
    assert MARK in effect and "DRAFT" in effect and "209" in effect
    assert MARK in current and ARCHIVE_SHA in current and "confirmed" in current
    for index, offset in OFFSETS.items():
        assert f"#{index}／0x{offset:04X}" in current


def main():
    root = Path("workplace/hd")
    out = root / "small-projectiles-backlinks-v1-20261002.json"
    assert not out.exists(), "拒絕覆寫"
    spec_path = Path("docs/spec/024-hd-theme.md")
    research_path = Path("docs/re/038-hd-theme-feasibility.md")
    spec, research = spec_path.read_text(), research_path.read_text()
    check(spec, research)
    try:
        check(spec.replace(MARK, ""), research)
    except AssertionError:
        missing_marker_rejected = True
    else:
        raise AssertionError("缺標記負對照未拒絕")
    archive = Path("/orig/psychic-war/ENEMY00.PBL")
    assert sha(archive) == ARCHIVE_SHA
    data = archive.read_bytes()
    entries = pbl.images(data)
    rows = []
    for index, offset in OFFSETS.items():
        assert entries[index][1] == offset
        w, h, pixels = pbl.decode(data, offset)
        assert (w, h) == (16, 16)
        rows.append({"archive": str(archive), "archive_sha256": ARCHIVE_SHA,
                     "index": index, "file_offset": offset,
                     "decoded_sha256": hashlib.sha256(pixels).hexdigest()})
    verified = root / "small-projectiles-minton-independent-v1-20261002.json"
    result = json.loads(verified.read_text())
    assert result["general_event_count"] == result["known_source_events"] == 1009
    assert not result["unknown_source_events"] and not result["ambiguous_source_events"]
    for index in OFFSETS:
        assert result["tile_families"][f"ENEMY00#{index}"] > 0
    sources = {}
    for label, path in (("guard.py", Path(__file__)), ("decoder.py", Path("tools/pbl.py")),
                        ("spec.md", spec_path), ("research.md", research_path)):
        snap = root / f"small-projectiles-backlink-source-v1-20261002-{label}"
        with snap.open("xb") as file:
            file.write(path.read_bytes())
        assert sha(snap) == sha(path)
        sources[str(path)] = {"sha256": sha(path), "snapshot": str(snap)}
    doc = {"scope": "三個小圖塊來源回填，非HD完成", "resolved_keys": rows,
           "missing_marker_negative_rejected": missing_marker_rejected,
           "independent_receipt_sha256": sha(verified), "sources": sources}
    with out.open("x") as file:
        json.dump(doc, file, ensure_ascii=False, indent=2)
        file.write("\n")
    print("三個來源鍵及較早回填通過；缺標記負對照拒絕；024 §1.5仍DRAFT")


if __name__ == "__main__":
    main()
