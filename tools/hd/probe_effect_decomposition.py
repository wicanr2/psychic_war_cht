"""研究 038 §38 原型：檢查能否由原版場景唯一還原效果，不序列化 HD 狀態。"""
import argparse
import hashlib
import json
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent.parent))
import pbl


def require(ok, message):
    if not ok:
        raise ValueError(message)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--probe", required=True)
    parser.add_argument("--verified", required=True)
    parser.add_argument("--original", default="/orig/psychic-war")
    parser.add_argument("--out", required=True)
    args = parser.parse_args()
    prefix, output = pathlib.Path(args.probe), pathlib.Path(args.out)
    require(not output.exists(), "拒絕覆寫")
    receipt_path, verified_path = pathlib.Path(str(prefix) + ".json"), pathlib.Path(args.verified)
    receipt = json.loads(receipt_path.read_text())
    verified = json.loads(verified_path.read_text())
    require(sha(receipt_path.read_bytes()) == verified["inputs_sha256"][str(receipt_path)], "原版收據未經獨立核對")
    require(len(verified["results"]) == 1081 and receipt["seed_before"] == "5447", "驗證範圍或種子不同")
    inputs = {str(receipt_path): sha(receipt_path.read_bytes()), str(verified_path): sha(verified_path.read_bytes()), __file__: sha(pathlib.Path(__file__).read_bytes())}
    assets = {}
    for name, count in (("BEAM", 12), ("FIGHT", 12), ("ENEMY00", 30), ("ALLY", 31), ("SCREEN", 5), ("MENU", 1)):
        path = pathlib.Path(args.original) / (name + ".PBL")
        data = path.read_bytes()
        require(sha(data) == verified["inputs_sha256"][str(path)], "來源 PBL 不同")
        inputs[str(path)] = sha(data)
        entries = pbl.images(data)
        require(len(entries) == count, "實際圖數不同")
        for n, offset, length in entries:
            w, h, pixels = pbl.decode(data, offset)
            assets[name, n] = (w, h, bytes(pixels))
    exe_path = pathlib.Path("workplace/ida/PW_UNP.EXE")
    executable = exe_path.read_bytes()
    require(sha(executable) == verified["inputs_sha256"][str(exe_path)], "靜態遮罩來源不同")
    inputs[str(exe_path)] = sha(executable)
    offset = int.from_bytes(executable[8:10], "little") * 16 + int.from_bytes(executable[22:24], "little") * 16 + 0x4E36
    stencil = executable[offset:offset + 32]
    assets["MASK", 0] = (16, 16, bytes(10 if stencil[y * 2 + x // 8] & (128 >> (x % 8)) else 0 for y in range(16) for x in range(16)))
    base = bytearray(64000)

    def paint(dst, name, n, x, y):
        w, h, pixels = assets[name, n]
        for row in range(h):
            dst[(y + row) * 320 + x:(y + row) * 320 + x + w] = pixels[row * w:(row + 1) * w]

    for n in range(5):
        paint(base, "SCREEN", n, 0, n * 40)
    paint(base, "MENU", 0, 160, 4)
    paint(base, "ALLY", 0, 264, 152)

    def vector(frame):
        # 32≤x<288，144≤y<184；原版 4bpp，固定 40,960 個二進位方程。
        pixels = b"".join(frame[y * 320 + 32:y * 320 + 288] for y in range(144, 184))
        require(len(pixels) == 10240 and all(v < 16 for v in pixels), "場景尺寸或色號不符")
        return int.from_bytes(bytes(pixels[i] << 4 | pixels[i + 1] for i in range(0, len(pixels), 2)), "little")

    slots = {}
    for result in verified["results"]:
        name = "MASK" if result.get("kind") == "mask" else result["name"]
        slot = (name, *result["rect"])
        choices = slots.setdefault(slot, set())
        if name == "MASK":
            choices.add(0)
        else:
            choices.update(n for n in (result["from"], result["to"]) if n is not None)
    variables, raw_vectors = [], []
    for slot, choices in sorted(slots.items()):
        name, x, y, w, h = slot
        for n in sorted(choices):
            aw, ah, pixels = assets[name, n]
            require((aw, ah) == (w, h), "變數來源尺寸不同")
            delta = bytearray(64000)
            for row in range(h):
                for col in range(w):
                    i = (y + row) * 320 + x + col
                    delta[i] = pixels[row * w + col]
                    if slot == ("ENEMY00", 32, 152, 24, 32):
                        delta[i] ^= base[i]
            variables.append({"name": name, "rect": list(slot[1:]), "image": n})
            raw_vectors.append(vector(delta))
    basis = {}
    for n, v in enumerate(raw_vectors):
        combinations = 1 << n
        while v:
            pivot = v.bit_length() - 1
            if pivot not in basis:
                basis[pivot] = (v, combinations)
                break
            other, mask = basis[pivot]
            v ^= other
            combinations ^= mask
    rank = len(basis)
    require(rank == len(variables), f"分解不唯一：秩 {rank}／變數 {len(variables)}；不得猜選")

    def solve(v):
        result = 0
        while v:
            pivot = v.bit_length() - 1
            if pivot not in basis:
                return None
            other, mask = basis[pivot]
            v ^= other
            result ^= mask
        # 同位置最多一張完整姿勢；可解的 XOR 差分不一定是可顯示的合法場景。
        occupied = set()
        for n, variable in enumerate(variables):
            if not result & (1 << n):
                continue
            slot = (variable["name"], *variable["rect"])
            if slot in occupied:
                return None
            occupied.add(slot)
        return result

    index = {(v["name"], *v["rect"], v["image"]): n for n, v in enumerate(variables)}
    state = 1 << index[("ENEMY00", 32, 152, 24, 32, 4)]
    base_vector = vector(base)
    samples = []
    for result in verified["results"]:
        kind = "mask" if result.get("kind") == "mask" else "event"
        n = result["event"]
        event = receipt["mask_events" if kind == "mask" else "events"][n]
        for side in ("before", "after"):
            if side == "after":
                if kind == "mask":
                    state ^= 1 << index[("MASK", *result["rect"], 0)]
                else:
                    for phase in (result["from"], result["to"]):
                        if phase is not None:
                            state ^= 1 << index[(result["name"], *result["rect"], phase)]
            path = pathlib.Path(f"{prefix}-{kind}{n:03d}-{side}.frame")
            frame = path.read_bytes()
            h = sha(frame)
            require(h == event[side + "_sha256"] == verified["inputs_sha256"][str(path)], "實際原版幀雜湊不同")
            inputs[str(path)] = h
            decoded = solve(vector(frame) ^ base_vector)
            require(decoded == state, f"{kind}{n} {side} 分解與獨立已證實的呼叫方向不同")
            samples.append({"kind": kind, "event": n, "side": side, "step": event[("entry" if side == "before" else "return") + "_step"], "frame_sha256": h, "active_variables": [i for i in range(len(variables)) if state & (1 << i)], "mismatch": 0})
    require(state == 0, "結束仍有敵人或效果殘留")
    unknown = bytearray(base)
    unknown[144 * 320 + 280] ^= 1
    require(solve(vector(unknown) ^ base_vector) is None, "未知來源負對照竟然被辨識")
    pair = next([index[(*slot, n)] for n in sorted(choices)[:2]] for slot, choices in slots.items() if len(choices) > 1)
    require(solve(raw_vectors[pair[0]] ^ raw_vectors[pair[1]]) is None, "同位置雙姿勢負對照竟然被允許")
    with output.open("x") as f:
        json.dump({"schema": "psychic-war-effect-decomposition-prototype/1", "scope": "固定正常攻擊的 2162 個完整原版幀；唯一分解與讀檔重建候選方法", "inputs_sha256": inputs, "domain": [32, 144, 256, 40], "rank": rank, "variables": variables, "samples": samples, "negative_unknown_rejected": True, "negative_multiple_poses_rejected": True, "limits": "原型，未驗貼圖中途；不代表 HD 合成、正式讀檔或所有場景，未知／非唯一分解須回退原版"}, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print(f"唯一分解：秩 {rank}/{len(variables)}；2162 個完整幀吻合已證實方向，兩項負對照被拒絕")


if __name__ == "__main__":
    main()
