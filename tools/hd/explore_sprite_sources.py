"""研究038 §71：由正常state與真實F2鍵探索其他sprite；不修改正式程式。"""
import argparse
import hashlib
import json
import os
import platform
from pathlib import Path
import struct
import subprocess
import time


ROOT = Path('/src')
OUT = ROOT / 'workplace/hd'
PREFIX = 'companion-explore-v2-20261003'
ORIG = Path('/orig/psychic-war')


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def strict_pbl(data):
    table = struct.unpack_from('<H', data)[0]
    assert table >= 2 and table % 2 == 0 and table <= len(data)
    offsets = struct.unpack_from('<' + 'H' * (table // 2), data)
    assert offsets[0] == table and list(offsets) == sorted(offsets)
    result = []
    for number, offset in enumerate(offsets):
        width, height = data[offset] * 8, data[offset + 1] * 8
        assert width > 0 and height > 0
        need = width * height // 2
        cursor = offset + 18
        previous = data[cursor]
        cursor += 1
        decoded = bytearray()
        while len(decoded) < need:
            assert cursor < len(data), '缺少RLE byte，禁止零填充'
            current = data[cursor]
            cursor += 1
            if current == previous:
                assert cursor < len(data), '缺少RLE次數'
                count = data[cursor]
                cursor += 1
                decoded.extend([previous] * min(count, need - len(decoded)))
                if len(decoded) < need:
                    assert cursor < len(data), '缺少下一個RLE byte'
                    previous = data[cursor]
                    cursor += 1
            else:
                decoded.append(previous)
                previous = current
        pixels = bytes(c for byte in decoded for c in (byte >> 4, byte & 15))
        assert len(pixels) == width * height
        result.append((number, width, height, pixels, offset, cursor))
    return result


def run(argv, log):
    assert not log.exists(), '拒絕覆寫log'
    started = time.monotonic()
    with log.open('xb') as file:
        process = subprocess.run(argv, stdout=file, stderr=subprocess.STDOUT, timeout=90)
    row = {'argv': argv, 'returncode': process.returncode,
           'log': str(log.relative_to(ROOT)), 'log_sha256': digest(log),
           'elapsed_seconds': time.monotonic() - started}
    assert process.returncode == 0, row
    return row


def crop(frame, x, y, width, height):
    assert len(frame) == 64000 and 0 <= x <= 320-width and 0 <= y <= 200-height
    return b''.join(frame[(y+row)*320+x:(y+row)*320+x+width] for row in range(height))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--phase', choices=['initial'], required=True)
    args = parser.parse_args()
    assert OUT.is_dir() and OUT.stat().st_uid == OUT.stat().st_gid == 1000
    assert not list(OUT.glob(PREFIX + '*')), '拒絕覆寫舊探索'
    inputs = {}

    def remember(path):
        assert path.stat().st_uid == path.stat().st_gid == 1000
        key = str(path.relative_to(ROOT)) if path.is_relative_to(ROOT) else str(path)
        inputs[key] = digest(path)

    tool = Path(__file__)
    remember(tool)
    for path in sorted((ROOT / 'worktrees/dosgolem/cmd/probe').glob('*.go')):
        remember(path)
    for path in [ROOT / 'go.mod', ROOT / 'go.sum', ROOT / 'worktrees/dosgolem/go.mod',
                 ROOT / 'workplace/hd/dat-runtime-v1-20261002.go.work',
                 ROOT / 'replay/title-to-first-save.json', ORIG / 'PW.EXE']:
        remember(path)
    optional_sum = ROOT / 'worktrees/dosgolem/go.sum'
    absent_source_files = []
    if optional_sum.exists():
        remember(optional_sum)
    else:
        absent_source_files.append(str(optional_sum.relative_to(ROOT)))
    bank = []
    counts = {}
    for path in sorted(ORIG.glob('*.PBL')):
        remember(path)
        entries = strict_pbl(path.read_bytes())
        counts[path.name] = len(entries)
        for number, width, height, pixels, offset, end in entries:
            bank.append({'key': f'{path.name}:{number}', 'width': width, 'height': height,
                         'pixels': pixels, 'offset': offset, 'rle_read_end': end})
    assert len(counts) == 26 and len(bank) == 537
    binary = OUT / (PREFIX + '-probe.bin')
    build = run(['go', 'build', '-buildvcs=false', '-o', str(binary), './cmd/probe'],
                OUT / (PREFIX + '-build.log'))
    remember(binary)
    versions = {'python': platform.python_version(),
                'go': subprocess.check_output(['go', 'version'], text=True).strip()}
    inventory = []

    def identify(path):
        remember(path)
        frame = path.read_bytes()
        assert len(frame) == 64000 and max(frame) < 16
        matches = []
        for entry in bank:
            name = entry['key'].split(':')[0]
            if name not in ['ALLY.PBL', 'ROOM0.PBL', 'ROOM1.PBL', 'MAP.PBL']:
                continue
            x, y = (264,152) if name == 'ALLY.PBL' else ((0,0) if name == 'MAP.PBL' else (4,124))
            width, height = entry['width'], entry['height']
            if x + width > 320 or y + height > 200:
                continue
            region = crop(frame, x, y, width, height)
            if region == entry['pixels']:
                shifted = crop(frame, x+1, y, width, height)
                shift_difference = sum(a != b for a,b in zip(region, shifted, strict=True))
                changed = bytearray(region)
                changed[0] ^= 1
                assert bytes(changed) != entry['pixels']
                matches.append({'key': entry['key'], 'rect': [x,y,width,height],
                                'full_mismatch':0, 'negative_shift_pixels':shift_difference,
                                'negative_one_pixel':1})
        inventory.append({'frame': str(path.relative_to(ROOT)), 'matches':matches})
        return matches

    for path in sorted((ROOT / 'workplace/states').glob('*.frame')):
        identify(path)
    runs = []
    for label, start_step, at in [('07-first-play', 42000000, 42500000),
                                  ('09-battle-won',112000000,112500000)]:
        state = ROOT / 'workplace/states' / (label+'.state')
        remember(state)
        for kind in ['control', 'f2']:
            stem = OUT / (PREFIX+'-'+label+'-'+kind)
            frame = stem.with_suffix('.frame')
            output_state = stem.with_suffix('.state')
            end = start_step+12000000
            argv = [str(binary), '-load-state', str(state), '-root', str(ORIG), '-steps', str(end),
                    '-dump-screen', str(frame), '-dump-screen-png', str(stem.with_suffix('.png')),
                    '-save-state', str(end-1)+':'+str(output_state),
                    '-peek', 'lin:57ef:2,lin:16966:12']
            if kind == 'f2':
                argv += ['-hold','3c@'+str(at)+'+3000000','-hold-typematic=false']
            execution = run(argv, stem.with_suffix('.log'))
            for path in [output_state, stem.with_suffix('.png')]:
                remember(path)
            execution.update({'start_state':str(state.relative_to(ROOT)), 'start_step':start_step,
                              'end_step':end, 'state_save_step':end-1,
                              'branch':kind, 'frame':str(frame.relative_to(ROOT)),
                              'output_state':str(output_state.relative_to(ROOT)),
                              'matches':identify(frame)})
            runs.append(execution)
            print(label,kind,execution['matches'],flush=True)
    rows=[]
    for index in range(0,len(runs),2):
        a,b=runs[index:index+2]
        pixels_a=(ROOT/a['frame']).read_bytes()
        pixels_b=(ROOT/b['frame']).read_bytes()
        rows.append({'start_state':a['start_state'],
                     'f2_vs_control_changed_pixels':sum(x!=y for x,y in zip(pixels_a,pixels_b,strict=True))})
    result={'scope':'正常檢查點完整內容盤點與真實F2輸入探索；不是正式HD驗收',
            'versions':versions,'inputs_sha256':inputs,'absent_source_files':absent_source_files,
            'archives':counts,'build':build,
            'binary_sha256':digest(binary),'inventory':inventory,'runs':runs,'comparisons':rows,
            'limits':'只比已知原版座標；不聲稱未匹配來源不存在。未觀察精確貼圖入口或IRQ1事件序列，不猜測角色身份。控制與F2是不同正常輸入，不要求終點相同。probe的state在截止前一指令保存，PNG／frame是實際截止畫面，不冒稱兩者同幀。'}
    path=OUT/(PREFIX+'.json')
    with path.open('x') as file:
        json.dump(result,file,ensure_ascii=False,indent=2)
        file.write('\n')
    print(json.dumps({'comparisons':rows,'inventory_frames':len(inventory),'result':str(path)},ensure_ascii=False))


if __name__=='__main__':
    main()
