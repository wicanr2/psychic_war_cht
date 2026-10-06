"""核對正常遭遇戰前三次貼圖；只證明 ENEMY00 #3–#5，未驗 HD。

    tools/py.sh tools/hd/verify_enemy_xor.py

先依 docs/re/038 §20 的原版重播命令產生收據。本工具不推進遊戲，
期望值來自獨立 PBL 解碼；不以攔截器訊號或 HD 圖作 oracle。
"""
import argparse
import datetime
import hashlib
import json
import pathlib
import re
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent.parent))
import pbl  # noqa: E402

CALLS = [
    (83190318, 83214859, "1175:A686", 0, 3, "hd-xor-source0.bin"),
    (83373343, 83397692, "1175:A8C6", 1, 4, "hd-xor-source1.bin"),
    (83556097, 83580446, "0161:344A", 1, 5, "hd-xor-source2.bin"),
]
END_SHA256 = "d0616fd57d1f7d078ea8abae989e6e901ff069f23890a0906e8cea845fb2e7d0"


def require(condition, message):
    if not condition:
        raise ValueError(message)


def sha256(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def tile(frame):
    require(len(frame) == 64000, "畫面不是 320×200 色號陣列")
    return b"".join(frame[y * 320 + 32:y * 320 + 56] for y in range(152, 184))


def packed(pixels):
    return bytes((pixels[i] << 4) | pixels[i + 1] for i in range(0, len(pixels), 2))


def unpacked(data):
    return bytes(v for byte in data for v in (byte >> 4, byte & 15))


def check_backlinks(root):
    evidence_path = root / "docs/re/038-hd-theme-feasibility.md"
    older_path = root / "docs/spec/024-hd-theme.md"
    evidence = evidence_path.read_text(encoding="utf-8")
    older = older_path.read_text(encoding="utf-8")
    earlier, current = evidence.split("## 20. ", 1)
    marker = "【HD-XOR-01】"
    require(marker in earlier, "§18 缺少新證據回查標記")
    require(marker in older and "§20" in older, "舊規格缺少差分訂正或證據入口")
    for address in ("0161:8705", "1175:A8C6", "0161:344A", "0x18C15", "0x10510"):
        require(address in current, f"新證據缺原始定位 {address}")
    return {"module": "DOS／PW_UNP.EXE；輸入雜湊見證據 §17.2",
            "entry": "執行期 0161:8705／IDA ea 0x18C15（base 0x10510）",
            "sources_in_fixed_state": ["1175:A8C6", "0161:344A"],
            "evidence": str(evidence_path.relative_to(root)) + " §20",
            "older": str(older_path.relative_to(root)) + " §8",
            "required_marker": marker, "checked": True}


def verify(probe, original, state, binary):
    backlinks = check_backlinks(pathlib.Path(__file__).resolve().parents[2])
    previous = probe / "hd-draw-observation-20260930.json"
    history = json.loads(previous.read_text(encoding="utf-8"))
    source = original / "ENEMY00.PBL"
    require(sha256(source) == history["source_inputs"]["ENEMY00.PBL"], "原版 PBL 雜湊已變")
    require(sha256(state) == history["initial_state"]["sha256"], "初始狀態雜湊已變")
    require(history["initial_state"]["seed"] == "0x86AF", "先前同一狀態的種子收據不符")
    require(history["inputs"] == {"press": ",".join(["up"] * 11), "press_at": 43000000,
                                   "press_every": 4000000, "end_step": 86000000},
            "先前的正常重播輸入收據不符")
    log = (probe / "hd-xor-return.log").read_text(encoding="utf-8")
    for enter, leave, pointer, mode, _, _ in CALLS:
        segment, offset = pointer.split(":")
        for step in (enter, leave):
            pattern = rf"#{step} AX=080{mode} BX={offset} CX=0826 DX=0304 .* DS={segment} "
            require(re.search(pattern, log) is not None, f"貼圖暫存器收據缺少或不符 #{step}")
    data = source.read_bytes()
    decoded = {i: pbl.decode(data, offset) for i, offset, _ in pbl.images(data)}
    paths = [source, state, binary, previous, original / "PW.EXE"]
    results = []
    for n, (enter, leave, pointer, mode, image, raw_name) in enumerate(CALLS):
        before_path = probe / f"hd-xor-before{n}.frame"
        after_path = probe / f"hd-xor-after{n}.frame"
        raw_path = probe / raw_name
        before, after = before_path.read_bytes(), after_path.read_bytes()
        before_tile, after_tile = tile(before), tile(after)
        width, height, pixels = decoded[image]
        require((width, height) == (24, 32), f"原圖 #{image} 的尺寸不符")
        expected = bytes(pixels)
        mismatch = sum(a != b for a, b in zip(after_tile, expected))
        require(mismatch == 0, f"動作 #{image} 與原版不同 {mismatch} 像素")
        outside = sum(before[y * 320 + x] != after[y * 320 + x]
                      for y in range(200) for x in range(320)
                      if not (32 <= x < 56 and 152 <= y < 184))
        require(outside == 0, f"貼圖外變動 {outside} 像素")
        raw = raw_path.read_bytes()
        require(len(raw) == 384, f"{raw_name} 不是 384 bytes")
        if mode == 0:
            require(raw == packed(expected), "第一筆來源與原版完整圖不符")
            ignored = None
        else:
            prior = bytes(decoded[image - 1][2])
            require(before_tile == prior, "差分前的畫面不符前一個原版動作")
            delta = bytes(a ^ b for a, b in zip(prior, expected))
            require(unpacked(raw) == delta, "來源不符兩動作的完整 XOR 差分")
            require(bytes(a ^ b for a, b in zip(before_tile, unpacked(raw))) == after_tile,
                    "XOR 重建與原版實際畫面不符")
            ignored = sum(a != b for a, b in zip(before_tile, after_tile))
            require(ignored > 0, "忽略差分的負對照沒有區辨力")
        results.append({"enter_step": enter, "return_step": leave,
                        "source_ds_bx": pointer, "al": mode, "original_image": image,
                        "width": width, "height": height, "at": [32, 152],
                        "original_mismatch_pixels": mismatch, "outside_changed_pixels": outside,
                        "negative_ignore_delta_pixels": ignored})
        paths.extend([before_path, after_path, raw_path])
    end = probe / "hd-xor-normal-end.frame"
    require(sha256(end) == END_SHA256, "正常重播終點與既有收據不同")
    paths.extend([end, probe / "hd-xor-return.log", probe / "hd-xor-normal.log"])
    project = pathlib.Path(__file__).resolve().parents[2]
    paths.extend([pathlib.Path(__file__).resolve(), project / "docs/re/038-hd-theme-feasibility.md",
                  project / "docs/spec/024-hd-theme.md"])
    return {
        "schema": "psychic-war-enemy-xor-observation/1",
        "recorded_at_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
        "status": "confirmed：只限正常遭遇戰前三次貼圖來源、XOR 重建與實際畫面",
        "tool": "dosgolem f8c1a6e／Go 1.24.13；tools/pbl.py；Python " + sys.version.split()[0],
        "address_space": "執行期段:偏移；PBL 檔名／圖號；畫面色號座標，非 IDA ea",
        "initial_state": history["initial_state"],
        "seed_evidence": "先前同一狀態的 CS:41DF 唯讀收據；本輪再次核對完整 state SHA-256，載入即固定",
        "inputs": history["inputs"],
        "files_sha256": {str(p): sha256(p) for p in paths},
        "observations": results,
        "backlinks": backlinks,
        "normal_end_sha256": END_SHA256,
        "limits": "未驗 HD、全角色、全部 AL 模式或小圖塊用途；不改遊戲記憶體。沒有重新推進亂數或挑選通過結果。",
    }


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--probe-dir", type=pathlib.Path, default=pathlib.Path("workplace/probe"))
    parser.add_argument("--original-dir", type=pathlib.Path,
                        default=pathlib.Path("workplace/original/psychic-war"))
    parser.add_argument("--state", type=pathlib.Path, default=pathlib.Path("workplace/states/07-first-play.state"))
    parser.add_argument("--binary", type=pathlib.Path, default=pathlib.Path("workplace/bin/probe-hd-native"))
    parser.add_argument("--output", type=pathlib.Path,
                        default=pathlib.Path("workplace/probe/hd-enemy-xor.json"))
    args = parser.parse_args(argv)
    try:
        receipt = verify(args.probe_dir, args.original_dir, args.state, args.binary)
        args.output.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    except (OSError, ValueError, KeyError) as exc:
        print(f"驗證失敗：{exc}", file=sys.stderr)
        return 1
    negatives = "／".join(str(row["negative_ignore_delta_pixels"])
                         for row in receipt["observations"][1:])
    print(f"ENEMY00 #3–#5：原圖不符 0、貼圖外變動 0；忽略差分的負對照 {negatives} 像素。寫出 {args.output}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
