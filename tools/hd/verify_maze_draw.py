"""研究038 §72：由原始MAZE、完整前後畫面及列中途獨立核對圖塊copy。"""
import base64
import hashlib
import json
from pathlib import Path


ROOT = Path('/src')
OUT = ROOT / 'workplace/hd'
PREFIX = 'maze-draw-source-v2-20261003'


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def unpack(raw):
    return bytes(value for b in raw for value in (b >> 4, b & 15))


def crop(frame, x, y):
    return b''.join(frame[(y+r)*320+x:(y+r)*320+x+4] for r in range(4))


def apply(frame, x, y, tile):
    value = bytearray(frame)
    for row in range(4):
        value[(y+row)*320+x:(y+row)*320+x+4] = tile[row*4:row*4+4]
    return value


def mismatches(a, b):
    assert len(a) == len(b)
    return sum(x != y for x, y in zip(a, b))


def main():
    target = OUT / 'maze-draw-verification-v1-20261003.json'
    assert not target.exists(), '拒絕覆寫'
    receipt = OUT / (PREFIX + '.json')
    data = json.loads(receipt.read_text())
    maze_path = Path('/orig/psychic-war/MAZE.BIN')
    maze = maze_path.read_bytes()
    assert len(maze) == 2048 and digest(maze_path) == data['inputs_sha256'][str(maze_path)]
    frames_path = Path(data['frame_stream'])
    assert digest(frames_path) == data['frame_stream_sha256']
    assert frames_path.stat().st_size == len(data['events']) * 128000
    assert data['observed_end'] == data['control_end']
    assert data['complete_machine_snapshot_equal'] and len(data['keys_irq1']) == 4
    checked = []
    negatives = {}
    slots, coords, callers, source_bases = set(), set(), set(), set()
    with frames_path.open('rb') as stream:
        for ordinal, event in enumerate(data['events']):
            assert event['FrameOffset'] == ordinal * 128000
            before, after = stream.read(64000), stream.read(64000)
            assert len(before) == len(after) == 64000
            slot, x, y = event['Slot'], event['X'], event['Y']
            raw = maze[slot*8:slot*8+8]
            assert raw == base64.b64decode(event['Source'])
            assert 0 <= slot < 256 and x % 4 == y % 4 == 0
            assert 4 <= x <= 72 and 124 <= y <= 192
            assert event['LinearSource'] == event['DS']*16 + event['SourcePointer'] + slot*8
            assert event['Entry'] < event['Return'] and len(event['Rows']) == 4
            tile = unpack(raw)
            want = apply(before, x, y, tile)
            assert want == after, f'完整圖塊及區外不符：{ordinal}'
            old_tile = crop(before, x, y)
            previous = event['Entry']
            for row_id, row in enumerate(event['Rows'], 1):
                assert previous < row['Step'] < event['Return']
                previous = row['Step']
                assert row['ReturnIP'] == [0x5002, 0x5010, 0x501e, 0x502c][row_id-1]
                assert row['SI'] == x and row['DI'] == y + row_id - 1
                middle = base64.b64decode(row['Tile'])
                assert middle == tile[:row_id*4] + old_tile[row_id*4:]
                if row_id < 4 and 'premature_full_tile' not in negatives:
                    n = mismatches(middle, tile)
                    if n:
                        negatives['premature_full_tile'] = {'event': ordinal, 'row': row_id, 'mismatch': n}
            if not negatives.get('wrong_source_pixel'):
                bad = bytes([tile[0] ^ 1]) + tile[1:]
                negatives['wrong_source_pixel'] = {'event': ordinal, 'mismatch': mismatches(apply(before,x,y,bad),after)}
            if 'wrong_slot' not in negatives:
                bad_slot = next(s for s in range(256) if maze[s*8:s*8+8] != raw)
                negatives['wrong_slot'] = {'event': ordinal, 'slot': bad_slot, 'mismatch': mismatches(apply(before,x,y,unpack(maze[bad_slot*8:bad_slot*8+8])),after)}
            if 'shift_one_pixel' not in negatives:
                n = mismatches(apply(before, x+1, y, tile), after)
                if n:
                    negatives['shift_one_pixel'] = {'event': ordinal, 'mismatch': n}
            if 'omit_tile' not in negatives and before != after:
                negatives['omit_tile'] = {'event': ordinal, 'mismatch': mismatches(before, after)}
            slots.add(slot); coords.add((x,y)); callers.add(event['ReturnIP'])
            source_bases.add(event['DS']*16 + event['SourcePointer'])
            checked.append({'event': ordinal, 'entry': event['Entry'], 'return': event['Return'],
                            'slot': slot, 'at': [x,y], 'whole_frame_mismatch': 0, 'rows_checked': 4})
        assert stream.read() == b''
    assert set(negatives) == {'wrong_source_pixel','wrong_slot','shift_one_pixel','omit_tile','premature_full_tile'}
    assert all(n['mismatch'] > 0 for n in negatives.values())
    output = {
        'scope': '正常BBS路線原始4×4圖塊與逐列中途；不是HD驗收',
        'inputs_sha256': {str(p):digest(p) for p in [Path(__file__),receipt,maze_path,frames_path]},
        'event_count': len(checked), 'row_count': len(checked)*4,
        'unique_slots': sorted(slots), 'unique_coordinates': sorted(coords),
        'runtime_return_ips': sorted(callers), 'runtime_source_bases': sorted(source_bases),
        'whole_frame_mismatch': 0, 'rows_mismatch': 0, 'checked': checked, 'negative_controls': negatives,
        'limits': '僅正常開局左轉及前進。四列中途只核對圖塊矩形；完整返回核對整屏。不證明其他來源、ROOM優先序、主題格式、HD素材或GUI。',
    }
    with target.open('x') as stream:
        json.dump(output, stream, ensure_ascii=False, indent=2)
        stream.write('\n')
    print(json.dumps({k:output[k] for k in ['event_count','row_count','runtime_return_ips','runtime_source_bases','negative_controls']},ensure_ascii=False))


if __name__ == '__main__':
    main()
