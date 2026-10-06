"""研究 038 §40：獨立核對有限原版按鍵樣本的貼圖來源、運算及矩形外像素。

不推定防護盾是否有效，也不以沒有新圖族宣稱所有能力或 sprite 已驗。
"""
import argparse
import hashlib
import json
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import pbl


def digest(data):
    return hashlib.sha256(data).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--out', required=True)
    args = parser.parse_args()
    output = Path(args.out)
    assert not output.exists() and output.parent.is_dir()
    hd = Path('workplace/hd')
    inputs = {__file__: digest(Path(__file__).read_bytes()), 'tools/pbl.py': digest(Path('tools/pbl.py').read_bytes())}
    assets = []
    for name, count in (('BEAM', 12), ('FIGHT', 12), ('ENEMY00', 30), ('ALLY', 31)):
        path = Path('/orig/psychic-war', name+'.PBL')
        data = path.read_bytes()
        inputs[str(path)] = digest(data)
        entries = pbl.images(data)
        assert len(entries) == count
        for n, off, _ in entries:
            w, h, pixels = pbl.decode(data, off)
            packed = bytes(pixels[i]<<4 | pixels[i+1] for i in range(0, len(pixels), 2))
            assets.append((name, n, w, h, packed))
    exe_path = Path('workplace/ida/PW_UNP.EXE')
    executable = exe_path.read_bytes()
    inputs[str(exe_path)] = digest(executable)
    offset = int.from_bytes(executable[8:10], 'little')*16+int.from_bytes(executable[22:24], 'little')*16+0x4E36
    stencil = executable[offset:offset+32]
    assert digest(stencil) == 'e1aa2b9ddb6488a70d573ca3a03e71028cd8a2dafa68087c6f6f3d7c60950a5a'
    results = []
    sequences = {}
    for scenario in ('enter', 'none', 'space-enter'):
        prefix = hd/f'battle-key-{scenario}-v1-20261001'
        rp = Path(str(prefix)+'.json')
        receipt = json.loads(rp.read_text())
        inputs[str(rp)] = digest(rp.read_bytes())
        source_version = 'v2' if scenario == 'space-enter' else 'v1'
        source_path = hd/f'observe_battle_effects-keys-{source_version}-20261001.go'
        assert digest(source_path.read_bytes()) == receipt['inputs_sha256']['tools/hd/observe_battle_effects.go']
        inputs[str(source_path)] = digest(source_path.read_bytes())
        for name, expected in receipt['inputs_sha256'].items():
            if name == 'tools/hd/observe_battle_effects.go':
                continue
            assert digest(Path(name).read_bytes()) == expected, name
            inputs[name] = expected
        assert receipt['seed_before'] == '5447' and receipt['end_steps'] == 112000000
        assert receipt['inputs_sha256']['workplace/states/08-encounter.state'] == '02d5fdba9181cb7fbec6282d74efbc25396562ba1403893cdf4c579ddfe770ba'
        assert not receipt['unknown_source_counts'] and not receipt['unknown_mask_sources']
        events = receipt['events'] or []
        masks = receipt['mask_events'] or []
        assert len(events) == sum(receipt['all_geometry_counts'].values())
        rows = []
        for kind, records in (('event', events), ('mask', masks)):
            for index, event in enumerate(records):
                x, y, w, h = event['rect']
                assert 0 <= x < x+w <= 320 and 0 <= y < y+h <= 200
                files = [Path(f'{prefix}-{kind}{index:03d}-{suffix}') for suffix in ('before.frame', 'after.frame', 'source.bin')]
                before, after, raw = [p.read_bytes() for p in files]
                for path in files:
                    inputs[str(path)] = digest(path.read_bytes())
                assert len(before) == len(after) == 64000
                assert digest(before) == event['before_sha256'] and digest(after) == event['after_sha256'] and digest(raw) == event['source_sha256']
                expected = bytearray(before)
                if kind == 'mask':
                    assert raw == stencil and (w, h) == (16, 16) and event['xor_color'] == 10
                    for yy in range(h):
                        for xx in range(w):
                            if raw[yy*2+xx//8] & (128 >> (xx%8)):
                                expected[(y+yy)*320+x+xx] ^= 10
                else:
                    labels = [f'{name}#{n}' for name, n, aw, ah, packed in assets if (aw, ah) == (w, h) and packed == raw]
                    pairs = []
                    for i, (name, n, aw, ah, a) in enumerate(assets):
                        if (aw, ah) != (w, h):
                            continue
                        for name2, n2, bw, bh, b in assets[i+1:]:
                            if name != name2 or (bw, bh) != (w, h):
                                continue
                            if bytes(c^d for c, d in zip(a, b)) == raw:
                                pairs.append(f'{name}#{n}^{name}#{n2}')
                    assert set(labels) == set(event['source_images'] or [])
                    assert set(pairs) == set(event['source_pairs'] or []) and (labels or pairs)
                    assert event['al'] in (0, 1), '未證實貼圖模式'
                    for yy in range(h):
                        for xx in range(w):
                            source_index = yy*w+xx
                            value = raw[source_index//2] >> 4 if source_index%2 == 0 else raw[source_index//2] & 15
                            pos = (y+yy)*320+x+xx
                            expected[pos] = before[pos]^value if event['al'] == 1 else value
                assert bytes(expected) == after, (scenario, kind, index)
                changed = sum(a != b for a, b in zip(before, after))
                assert changed > 0, '省略負對照不能分辨輸出'
                rows.append({'kind': kind, 'event': index, 'rect': event['rect'], 'source_mismatch': 0, 'whole_frame_mismatch': 0, 'omit_negative_pixels': changed})
        sequences[scenario] = [(e['source_sha256'], e['before_sha256'], e['after_sha256']) for e in events]
        results.append({'scenario': scenario, 'input': receipt['input'], 'events': len(events), 'masks': len(masks), 'end_cycles': receipt['end_cycles'], 'end_indexed_sha256': receipt['end_indexed_sha256'], 'end_ram_sha256': receipt['end_ram_sha256'], 'results': rows})
    with output.open('x') as f:
        json.dump({'scope': '固定原版首場遭遇，Enter／none／space+Enter 三個按鍵樣本；獨立來源與逐貼圖矩形外核對，不驗能力語意或 HD', 'inputs_sha256': inputs, 'results': results, 'enter_none_source_and_frames_equal': sequences['enter'] == sequences['none'], 'limits': '相同畫面不證明完整狀態或防護效果相同；F1/F2缺前提，未在本批執行。未證明FIGHT #4–11或ALLY #12–15正常路徑。'}, f, ensure_ascii=False, indent=2)
        f.write('\n')
    print([(r['scenario'], r['events'], r['masks']) for r in results])


if __name__ == '__main__':
    main()
