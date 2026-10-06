"""核對首次迷宮右側 ALLY #0 貼圖；不證明全部角色或 HD 動畫完成。

    tools/py.sh tools/hd/verify_ally0.py

來源重生、固定狀態與位址基準見 docs/re/038 §22。
"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))
import pbl

STATE_SHA = 'ac8eec0bbc533b267782298c298ce4a60440d0568266e665c8a5badbf9e0ca4a'
ALLY_SHA = 'c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219'
END_SHA = 'a17c8610ac338deddd3060238b4debe3e837df6c6f8c685d7d9162c0e4cedfc7'
ENCOUNTER_SHA = 'd0616fd57d1f7d078ea8abae989e6e901ff069f23890a0906e8cea845fb2e7d0'

def require(condition, message):
    if not condition:
        raise ValueError(message)

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def tile(frame):
    require(len(frame) == 64000, '原版畫面必須為 320×200 色號陣列')
    return b''.join(frame[y*320+264:y*320+288] for y in range(152, 184))

def discover(original, frame):
    """僅搜尋既有畫面的右下區域；不用圖塊名稱猜角色身分。"""
    hits = []
    total = eligible = 0
    input_hashes = {}
    for path in sorted(original.glob('*.PBL')):
        data = path.read_bytes()
        input_hashes[str(path)] = sha(path)
        for image, offset, _ in pbl.images(data):
            total += 1
            w, h, pixels = pbl.decode(data, offset)
            if w > 64 or h > 48 or sum(bool(v) for v in pixels) < 20:
                continue
            eligible += 1
            for mirror in (False, True):
                rows = [bytes(pixels[y*w:(y+1)*w]) for y in range(h)]
                if mirror:
                    rows = [row[::-1] for row in rows]
                for y in range(130, min(190, 200-h)+1):
                    for x in range(240, min(310, 320-w)+1):
                        if all(frame[(y+r)*320+x:(y+r)*320+x+w] == rows[r] for r in range(h)):
                            hits.append({'source': path.name, 'image': image, 'file_offset': offset,
                                         'size': [w, h], 'at': [x, y], 'mirror_x': mirror})
    return {'total_images': total, 'eligible_images': eligible,
            'limits': '尺寸≤64×48、非黑像素≥20；左上角 x=240–310、y=130–190；完整矩形、原向或水平鏡像',
            'hits': hits, 'inputs_sha256': input_hashes}

def verify(args):
    probe = args.probe_dir
    source = args.original_dir / 'ALLY.PBL'
    require(sha(source) == ALLY_SHA, '原版 ALLY.PBL 雜湊不符')
    require(sha(args.state) == STATE_SHA, '固定名字狀態雜湊不符')
    seed_file = probe / 'hd-rightactor-initial-seed.json'
    seed = json.loads(seed_file.read_text(encoding='utf-8'))
    require(seed['sha256'] == STATE_SHA and seed['seed_cs_41df'] == '86AF'
            and seed['steps'] == 35000000, '執行前固定種子收據不符')
    log_file = probe / 'hd-rightactor-name.log'
    log = log_file.read_text(encoding='utf-8')
    for address, step in [('8705', 38648892), ('8751', 38673433)]:
        section = log.split(f'0161:{address} 執行時的暫存器', 1)[1]
        require(re.search(rf'#{step} AX=4200 BX=96C6 CX=4226 DX=0304 .* DS=1175 ', section),
                f'原版 {address} 的貼圖參數不符')
    data = source.read_bytes()
    image, offset, _ = pbl.images(data)[0]
    w, h, pixels = pbl.decode(data, offset)
    require(image == 0 and (w, h) == (24, 32), '原版 ALLY #0 圖號或尺寸不符')
    expected = bytes(pixels)
    before_path, after_path = [probe / f'hd-rightactor-{stage}.frame' for stage in ('before', 'after')]
    before, after = before_path.read_bytes(), after_path.read_bytes()
    require(tile(after) == expected, '貼圖後畫面與獨立 ALLY #0 期望值不同')
    raw_path = probe / 'hd-rightactor-source.bin'
    raw = raw_path.read_bytes()
    packed = bytes((expected[i] << 4) | expected[i+1] for i in range(0, len(expected), 2))
    require(len(raw) == 384 and raw == packed, 'DS:BX 來源不符原版 ALLY #0 完整圖')
    outside = sum(before[y*320+x] != after[y*320+x] for y in range(200) for x in range(320)
                  if not (264 <= x < 288 and 152 <= y < 184))
    require(outside == 0, f'貼圖矩形外變動 {outside} 像素')
    changed = sum(a != b for a, b in zip(tile(before), expected))
    require(changed > 0, '忽略貼圖的負對照沒有區辨力')
    end = probe / 'hd-rightactor-name-end.frame'
    require(sha(end) == END_SHA, '正常名字重播終點雜湊不符')
    encounter = args.encounter
    require(sha(encounter) == ENCOUNTER_SHA, '已驗中文遭遇戰原版畫面雜湊不符')
    require(tile(end.read_bytes()) == expected and tile(encounter.read_bytes()) == expected,
            '迷宮或遭遇戰的右側圖塊不符 ALLY #0')
    discovery = discover(args.original_dir, encounter.read_bytes())
    require(len(discovery['hits']) == 1 and discovery['hits'][0]['source'] == 'ALLY.PBL'
            and discovery['hits'][0]['image'] == 0, '限定區域內完整圖匹配不唯一或不同')
    project = Path(__file__).resolve().parents[2]
    evidence = project / 'docs/re/038-hd-theme-feasibility.md'
    spec = project / 'docs/spec/024-hd-theme.md'
    marker = '【HD-ALLY0-01】'
    earlier, current = evidence.read_text(encoding='utf-8').split('## 22. ', 1)
    require(marker in earlier and marker in current and marker in spec.read_text(encoding='utf-8'),
            '舊研究或草案缺少右側來源訂正回查')
    for address in ('0161:8705', '0161:8751', '1175:96C6', '0x1AE16', '0x003E'):
        require(address in current, '新來源證據缺少原始定位 ' + address)
    files = [source, args.state, seed_file, log_file, before_path, after_path, raw_path, end,
             encounter, Path(__file__), project / 'tools/pbl.py', evidence, spec,
             project / 'workplace/hd/inspect_state_seed.go', project / 'workplace/bin/probe-hd-native']
    return {'status': 'confirmed：限首次右側原版貼圖與兩張後續正常畫面，尚未驗 HD 動畫',
            'tool': 'dosgolem f8c1a6e／Go 1.24.13／Python ' + sys.version.split()[0] + '／tools/pbl.py',
            'address_space': '執行期段:偏移；PBL 檔案偏移及圖號；原版色號畫面座標，非 IDA ea',
            'initial_state': seed,
            'inputs': {'press': '25,1E,17,enter', 'press_at': 35500000,
                       'press_every': 500000, 'end_step': 42000001},
            'original_location': {'entry': '0161:8705', 'return': '0161:8751',
                                  'entry_step': 38648892, 'return_step': 38673433,
                                  'DS:BX': '1175:96C6', 'source_linear': '0x1AE16', 'AL': '00h',
                                  'CX': '4226h', 'DX': '0304h', 'pbl_offset': offset, 'image': 0},
            'additional_semantics': '384 bytes 完整 ALLY #0 在 (264,152) 畫成 24×32；不推論角色身分',
            'evidence_level': '已證實，限該次參數、來源與前後畫面',
            'results': {'source_byte_mismatch': 0, 'after_original_mismatch': 0,
                        'outside_changed_pixels': outside, 'negative_ignore_blit_pixels': changed},
            'discovery': discovery, 'files_sha256': {str(f): sha(f) for f in files},
            'backlink': marker + '；docs/re/038 §21→§22、docs/spec/024 §8',
            'limits': '不改原版；未驗全部 ALLY 圖、動作、局部圖塊、正式主題或存讀檔。'}

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--probe-dir', type=Path, default=Path('workplace/probe'))
    parser.add_argument('--original-dir', type=Path, default=Path('workplace/original/psychic-war'))
    parser.add_argument('--state', type=Path, default=Path('workplace/states/06-name.state'))
    parser.add_argument('--encounter', type=Path, default=Path('workplace/hd/redraw/replay-encounter.frame'))
    parser.add_argument('--output', type=Path, default=Path('workplace/probe/hd-ally0-verification.json'))
    args = parser.parse_args()
    try:
        receipt = verify(args)
        args.output.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    except (OSError, ValueError, KeyError, IndexError) as exc:
        print('驗證失敗：' + str(exc), file=sys.stderr)
        return 1
    print('ALLY #0：來源不符 0、原圖不符 0、貼圖外變動 0；忽略貼圖負對照 '
          + str(receipt['results']['negative_ignore_blit_pixels']) + ' 像素；寫出 ' + str(args.output))
    return 0

if __name__ == '__main__':
    raise SystemExit(main())
