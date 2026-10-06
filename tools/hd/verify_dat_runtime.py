"""獨立核對原版DAT、玩家資料與正常畫面；研究038 §63，所有產物留本機。"""
import copy
import hashlib
import json
from pathlib import Path
import struct
import zlib


def sha(data):
    return hashlib.sha256(data).hexdigest()


def equal_payload(actual, expected):
    assert len(actual) == 512 and actual == expected, "DAT bytes不同"


def png_check(path):
    data = path.read_bytes()
    assert data[:8] == b"\x89PNG\r\n\x1a\n"
    pos, compressed = 8, bytearray()
    end = False
    while pos < len(data):
        size = struct.unpack_from(">I", data, pos)[0]
        kind = data[pos + 4:pos + 8]
        body = data[pos + 8:pos + 8 + size]
        assert pos + 12 + size <= len(data)
        assert zlib.crc32(kind + body) & 0xffffffff == struct.unpack_from(">I", data, pos + 8 + size)[0]
        if kind == b"IHDR":
            w, h, depth, color, method, filt, interlace = struct.unpack(">IIBBBBB", body)
            assert (w, h, depth, color, method, filt, interlace) == (960, 600, 8, 2, 0, 0, 0)
        if kind == b"IDAT":
            compressed.extend(body)
        pos += size + 12
        if kind == b"IEND":
            assert size == 0 and pos == len(data)
            end = True
            break
    assert end
    raw = zlib.decompress(compressed)
    assert len(raw) == (960 * 3 + 1) * 600
    assert all(raw[y * (960 * 3 + 1)] <= 4 for y in range(600))


def main():
    root = Path("workplace/hd/dat-runtime-v6-20261002")
    out = Path("workplace/hd/dat-independent-v1-20261002.json")
    assert not out.exists(), "拒絕覆寫"
    receipt = root / "verification.json"
    data = json.loads(receipt.read_text())
    inputs = dict(data["inputs_sha256"])
    inputs[str(receipt)] = sha(receipt.read_bytes())
    inputs[__file__] = sha(Path(__file__).read_bytes())
    for name, expected in inputs.items():
        assert sha(Path(name).read_bytes()) == expected, name
    plan = {s["name"]: s for s in json.loads(Path("replay/title-to-first-save.json").read_text())["segments"]}
    names = ["10-saved", "11-loaded", "17-saved2", "18-loaded2"]
    modes = ["original", "disabled", "enabled"]
    rows = {(r["mode"], r["name"]): r for r in data["results"]}
    assert len(rows) == len(data["results"]) == 12
    assert set(rows) == {(m, n) for m in modes for n in names}
    pngs, sample_count, verified = [], 0, []
    for name in names:
        original = rows["original", name]
        s = plan[name]
        marks = [s["key_at"] + i * s["key_every"] + 700000 for i in range(len(s["keys"]))]
        marks = [at for at in marks if at < s["save_at"]] + [s["save_at"]]
        expected_frame = Path("workplace/states") / (name + ".frame")
        expected_player = Path("workplace/states") / (name + ".mem")
        assert len(expected_frame.read_bytes()) == 64000
        assert len(expected_player.read_bytes()) == 52
        inputs[str(expected_frame)] = sha(expected_frame.read_bytes())
        inputs[str(expected_player)] = sha(expected_player.read_bytes())
        for mode in modes:
            row = rows[mode, name]
            assert row["replay_segment"]["Keys"] == s["keys"]
            assert row["seed_before"] == original["seed_before"]
            assert row["start"] == original["start"], (mode, name, "起點不同")
            assert row["end"] == original["end"], (mode, name, "終點不同")
            assert row["end"]["Steps"] == s["save_at"]
            assert row["end"]["IRQ1"] == len(s["keys"]) * 2
            assert [r["step"] for r in row["samples"]] == marks
            assert len(row["samples"]) == len(original["samples"])
            sample_count += len(marks)
            for r, ref in zip(row["samples"], original["samples"]):
                a, b = copy.deepcopy(r["fingerprint"]), copy.deepcopy(ref["fingerprint"])
                a.pop("Bus"); b.pop("Bus")
                assert a == b, (mode, name, r["step"])
                path = Path(r["frame"])
                assert len(path.read_bytes()) == 64000
                assert sha(path.read_bytes()) == r["fingerprint"]["Frame"]
                assert path.read_bytes() == Path(ref["frame"]).read_bytes()
                if mode == "enabled":
                    disabled = rows["disabled", name]["samples"][row["samples"].index(r)]
                    assert r["text_plane_sha256"] == disabled["text_plane_sha256"]
            last = Path(row["samples"][-1]["frame"])
            assert last.read_bytes() == expected_frame.read_bytes(), (mode, name, "原版畫面不同")
            player = root / (mode + "-" + name + "-player.bin")
            assert player.read_bytes() == expected_player.read_bytes(), (mode, name, "原版玩家資料不同")
            dat, ref_dat = Path(row["dat"]), Path(row["reference_dat"])
            equal_payload(dat.read_bytes(), ref_dat.read_bytes())
            assert sha(dat.read_bytes()) == row["dat_sha256"]
            assert dat.stat().st_mtime_ns == ref_dat.stat().st_mtime_ns
            if mode == "enabled":
                assert row["hd_visible_samples"] > 0, "HD空測"
            else:
                assert row["hd_visible_samples"] == 0
            shot = last.with_suffix(".png")
            png_check(shot)
            pngs.append(str(shot))
            verified.append({"mode": mode, "name": name, "samples": len(marks), "dat": str(dat), "dat_sha256": row["dat_sha256"], "old_frame_equal": True, "old_player_52_bytes_equal": True, "same_runtime_end": True})
    bad = bytearray(Path(rows["original", "10-saved"]["dat"]).read_bytes())
    bad[0] ^= 1
    rejected = False
    try:
        equal_payload(bytes(bad), Path(rows["original", "10-saved"]["reference_dat"]).read_bytes())
    except AssertionError:
        rejected = True
    assert rejected
    negative_root = Path("workplace/hd/dat-runtime-v5-20261002")
    actual = (negative_root / "original-11-loaded-actual-ram.bin").read_bytes()
    expected = (negative_root / "original-11-loaded-expected-ram.bin").read_bytes()
    differing = [i for i, (a, b) in enumerate(zip(actual, expected)) if a != b]
    assert differing == [0x1096, 0x1097, 0x1098]
    result = {"inputs_sha256": inputs, "results": verified, "samples_total": sample_count, "samples_per_mode": sample_count // 3, "pngs": pngs, "dat_byte_negative_rejected": rejected, "unfixed_mtime_negative_ram_addresses": [hex(i) for i in differing], "limits": "正常原版state接續的DAT選單流程及獨立舊畫面／52 bytes玩家資料。PNG僅CRC、解壓與尺寸核對及另行目視，未宣稱獨立完整HD合成、GUI、正式封包或所有sprite通過。"}
    with out.open("x") as f:
        json.dump(result, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print("獨立核對12分支、", sample_count, "取樣、12 PNG通過；DAT與檔案時間負對照有效")


if __name__ == "__main__":
    main()
