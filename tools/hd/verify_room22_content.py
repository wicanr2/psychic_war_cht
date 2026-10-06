"""研究038 §70：正常首Up完整內容與原點格的獨立核對；不推定繪製入口。"""
import hashlib
import json
import platform
from pathlib import Path

from verify_room_grid import pbl, reference, valid_cells, region, require


def main():
    output = Path('workplace/hd/room22-content-independent-v1-20261003.json')
    require(not output.exists(), '拒絕覆寫')
    inputs = {}

    def read(path):
        path = Path(path)
        require((path.stat().st_uid, path.stat().st_gid) == (1000, 1000), '來源擁有權不符')
        b = path.read_bytes()
        inputs[str(path)] = hashlib.sha256(b).hexdigest()
        return b

    prefix = 'workplace/hd/room22-source-v3-20261003'
    doc = json.loads(read(prefix + '.json'))
    for path, digest in doc['inputs_sha256'].items():
        require(hashlib.sha256(read(path)).hexdigest() == digest, '探針來源已變：' + path)
    require(doc['observed_end'] == doc['control_end'] and doc['existing_state_equal'], '原版三方終點不同')
    require(doc['route']['Keys'] == ['up'] and doc['route']['key_every'] == 3000000, '正常首Up輸入不同')
    require(doc['seed_before'] == 'F95B' and len(doc['keys_irq1']) == 2, '種子或送键次數不同')
    require(not doc['events'] and doc['known_drawing_entry_count'] == 0, '本批已知入口觀察結果不同')
    poses = {}
    archives = {}
    for path in sorted(Path('/orig/psychic-war').glob('*.PBL')):
        data = read(path)
        entries = pbl.images(data)
        archives[path.name] = len(entries)
        for number, offset, size in entries:
            w, h, pixels = pbl.decode(data, offset)
            poses[f'{path.name}:{number}'] = (w, h, bytes(pixels))
    require(archives == doc['archives'] and len(archives) == 26 and len(poses) == 537, '原版圖庫不同')
    w, h, pixels = poses['ROOM0.PBL:22']
    require((w, h) == (72, 72), 'ROOM0 #22原版尺寸不同')
    base, _ = reference(Path('/orig/psychic-war'), inputs)
    base = bytearray(base)
    for y in range(72):
        base[(124+y)*320+4:(124+y)*320+76] = pixels[y*72:(y+1)*72]
    rows = []
    for sample in doc['region_content_changes_10000_step_samples']:
        frame = read(sample['frame'])
        cropped = region(frame, 4, 124, 72, 72)
        require(len(frame) == 64000, '畫面長度不同')
        matches = sorted(k for k, v in poses.items() if v == (72, 72, cropped))
        require(matches == sample['full_region_matches'], '獨立完整內容配對不同')
        require(hashlib.sha256(cropped).hexdigest() == sample['region_sha256'], '區域SHA不同')
        if sample['state']:
            read(sample['state'])
        rows.append({'step': sample['step'], 'frame': sample['frame'], 'full_matches': matches,
                     'mismatch': 0, 'state': sample['state']})
    require(rows[0]['full_matches'] == ['ROOM0.PBL:2'], '正常起點不是降落平台')
    require(rows[-1]['full_matches'] == ['ROOM0.PBL:22'] and rows[-1]['state'], '正常完整ROOM0 #22缺少')
    require(all(not row['full_matches'] for row in rows[1:-1]), '中途被誤認完整原圖')
    end = read(prefix + '-end.frame')
    read(prefix + '-end.state')
    require(end == read('workplace/hd/transport-room-runtime-v1-20261003-sample20.frame'), '前批正常終點不同')
    require(region(end, 4, 124, 72, 72) == pixels, '正常終點不是原版ROOM0 #22')
    grid = []
    for frame_path in (rows[-1]['frame'], prefix + '-end.frame'):
        frame = read(frame_path)
        cells = valid_cells(frame, base)
        require(len(cells) == 100, '原版完整圖或外緣原點格不同')
        boundary = [(x,y) for x,y in cells if x in (0,72) or y in (120,192)]
        require(len(boundary) == 36, '邊界格數不同')
        changed = bytearray(frame)
        changed[120*320] ^= 1
        require(region(changed,4,124,72,72) == pixels, '外緣負對照改到原圖')
        require(len(valid_cells(changed,base)) == 99, '外緣失配負對照無效')
        shifted = sum(a != b for a,b in zip(region(frame,5,124,72,72),pixels,strict=True))
        wrong = sum(a != b for a,b in zip(poses['ROOM0.PBL:2'][2],pixels,strict=True))
        require(shifted > 0 and wrong > 0, '位移或錯圖號負對照無效')
        grid.append({'frame':frame_path,'valid_cells':100,'boundary_cells':36,'negative_halo_cells':99,
                     'negative_shift_pixels':shifted,'negative_wrong_image_pixels':wrong})
    candidate = Path('workplace/hd/art-in/ROOM0-22.png')
    read(candidate)
    cw,ch,channels,data,palette = pbl.read_png(candidate)
    require((cw,ch) == (216,216) and channels in (3,4), '候選尺寸或格式不同')
    require(len(data)==216 and all(len(row)==216*channels for row in data), '候選完整解碼不同')
    execution = json.loads(read('workplace/hd/room22-source-execution-v3-20261003.json'))
    require(execution['returncode']==0, '實際原版執行未通過')
    read(execution['log'])
    require(hashlib.sha256(read('workplace/hd/observe-room22-v3-20261003.bin')).hexdigest()==execution['binary_sha256'], '實際二進位不同')
    for path in (__file__,'tools/pbl.py','tools/hd/verify_room_grid.py','workplace/hd/room22-source-overlay-v3-20261003.json'):
        read(path)
    with output.open('x') as f:
        json.dump({'scope':'正常首Up由ROOM0 #2經中途到#22的完整內容及8×8格獨立核對',
                   'python_version':platform.python_version(),'inputs_sha256':inputs,'samples':rows,'grid':grid,
                   'known_drawing_entry_count':0,'sample_interval_steps':10000,'first_complete_sample_step':rows[-1]['step'],
                   'candidate':{'path':str(candidate),'size':[cw,ch],'channels':channels},
                   'limits':'完整內容來源與原版位置已證實；精確繪製入口、第一次寫入步數及原始RLE搬移仍未知。未驗HD、美術、其他sprite或GUI。'},f,ensure_ascii=False,indent=2)
        f.write('\n')
    print('正常19區域變動樣本匹配一致，中途17個未知；#22兩份100格／36邊界，外緣負對照99', grid)


if __name__ == '__main__':
    main()
