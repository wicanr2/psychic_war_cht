"""研究 038 §39：盤點 sprite 的色盤敏感色號，建立既有主畫面／戰鬥色盤參照。

不是所有輸出路徑的色盤 oracle；005 已證實戰鬥 2／A 為棕／黃、8 為黑。
"""
import argparse
import hashlib
import json
from pathlib import Path
import struct
import sys
import zlib

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import pbl


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def write_reference(path, w, h, pixels, palette):
    assert not path.exists() and path.parent.is_dir()
    def chunk(kind, data):
        return struct.pack('>I', len(data))+kind+data+struct.pack('>I', zlib.crc32(kind+data))
    rows = bytearray()
    for y in range(h*16):
        rows.append(0)
        for x in range(w*16):
            rows.extend(palette[pixels[(y//16)*w+x//16]])
    payload = b'\x89PNG\r\n\x1a\n'+chunk(b'IHDR', struct.pack('>IIBBBBB', w*16, h*16, 8, 2, 0, 0, 0))+chunk(b'IDAT', zlib.compress(rows))+chunk(b'IEND', b'')
    with path.open('xb') as f:
        f.write(payload)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--out', required=True)
    args = parser.parse_args()
    output = Path(args.out)
    assert not output.exists() and output.parent.is_dir()
    palette = list(pbl.EGA)
    palette[2], palette[8], palette[10] = (170, 85, 0), (0, 0, 0), (255, 255, 85)
    evidence = Path('docs/re/005-ega-palette-rgb-parity.md')
    # 同狀態原版 RGB 與色號樣本：先驗戰鬥區無中文字，再確認 2／A，避免重犯預設色盤錯誤。
    prefix = Path('workplace/hd/battle-effects-prototype-v4-20261001-step86702889')
    frame_path = Path(str(prefix)+'.frame')
    rgb_path = Path(str(prefix)+'-original-cht.png')
    frame = frame_path.read_bytes()
    w, h, channels, rows, pal = pbl.read_png(rgb_path)
    assert (w, h) == (960, 600) and channels in (3, 4) and pal is None
    observed = {2: set(), 10: set()}
    for y in range(144, 184):
        for x in range(32, 288):
            color = frame[y*320+x]
            if color in observed:
                observed[color].add(tuple(rows[y*3][x*3*channels:x*3*channels+3]))
    assert all(observed[c] == {palette[c]} for c in observed), observed
    inputs = {str(evidence): sha(evidence), str(frame_path): sha(frame_path), str(rgb_path): sha(rgb_path), __file__: sha(Path(__file__)), 'tools/pbl.py': sha(Path('tools/pbl.py'))}
    assets, inventory, references = {}, [], {}
    expected_counts = {'BEAM': 12, 'FIGHT': 12, 'ALLY': 31, **{f'ENEMY{n:02d}': 30 for n in range(12)}}
    for name, count in expected_counts.items():
        path = Path('/orig/psychic-war', name+'.PBL')
        data = path.read_bytes()
        inputs[str(path)] = sha(path)
        entries = pbl.images(data)
        assert len(entries) == count
        for n, off, _ in entries:
            sw, sh, pixels = pbl.decode(data, off)
            assets[name, n] = (sw, sh, pixels)
            counts = {str(c): pixels.count(c) for c in (2, 8, 10)}
            if any(counts.values()):
                reference = Path(f'workplace/hd/{name}-{n:02d}-reference-battle-v1-20261001.png')
                write_reference(reference, sw, sh, pixels, palette)
                references[str(reference)] = sha(reference)
                inventory.append({'pbl': name, 'image': n, 'dimensions': [sw, sh], 'counts': counts, 'reference': str(reference)})
    for n in (18, 19, 20):
        reference = Path(f'workplace/hd/ENEMY00-{n}-reference-v3-20261001.png')
        write_reference(reference, *assets['ENEMY00', n], palette)
        references[str(reference)] = sha(reference)
    executable = Path('workplace/ida/PW_UNP.EXE')
    b = executable.read_bytes()
    inputs[str(executable)] = sha(executable)
    offset = int.from_bytes(b[8:10], 'little')*16+int.from_bytes(b[22:24], 'little')*16+0x4E36
    stencil = b[offset:offset+32]
    assert hashlib.sha256(stencil).hexdigest() == 'e1aa2b9ddb6488a70d573ca3a03e71028cd8a2dafa68087c6f6f3d7c60950a5a'
    pixels = [10 if stencil[y*2+x//8] & (128 >> (x%8)) else 0 for y in range(16) for x in range(16)]
    reference = Path('workplace/hd/MASK-4E36-reference-v3-20261001.png')
    write_reference(reference, 16, 16, pixels, palette)
    references[str(reference)] = sha(reference)
    with output.open('x') as f:
        json.dump({'scope': '415 個 PBL sprite 的色號盤點；僅建立已驗主畫面／戰鬥色盤參照，不證明每圖的實際輸出路徑', 'inputs_sha256': inputs, 'source_count': sum(expected_counts.values()), 'affected': inventory, 'palette': palette, 'reference_sha256': references, 'observed_battle_rgb': {str(c): list(next(iter(observed[c]))) for c in observed}, 'limits': 'Game Over 等其他狀態可改色盤；候選是否錯色仍須對照個別輸出路徑。不把含色號本身當成美術缺陷。'}, f, ensure_ascii=False, indent=2)
        f.write('\n')
    print(f'盤點 {sum(expected_counts.values())} 圖，色盤敏感 {len(inventory)} 圖；輸出 {len(references)} 張正確戰鬥色盤參照')


if __name__ == '__main__':
    main()
