"""研究038 §71：從正常開局步行到BBS與Radar房間，尋找未接入原版圖。"""
import hashlib
import json
import platform
from pathlib import Path
import re
import subprocess
import sys

from explore_sprite_sources import strict_pbl, crop, run, digest


ROOT = Path('/src')
OUT = ROOT / 'workplace/hd'
ORIG = Path('/orig/psychic-war')
PREFIX = 'room-nearby-explore-v1-20261003'


def main():
    assert not list(OUT.glob(PREFIX+'*')), '拒絕覆寫'
    assert OUT.stat().st_uid == OUT.stat().st_gid == 1000
    inputs = {}

    def read(path):
        assert path.stat().st_uid == path.stat().st_gid == 1000
        b = path.read_bytes()
        key = str(path.relative_to(ROOT)) if path.is_relative_to(ROOT) else str(path)
        inputs[key] = hashlib.sha256(b).hexdigest()
        return b

    receipt = json.loads(read(OUT/'companion-explore-v2-20261003.json'))
    binary = OUT/'companion-explore-v2-20261003-probe.bin'
    assert hashlib.sha256(read(binary)).hexdigest() == receipt['binary_sha256']
    start = ROOT/'workplace/states/07-first-play.state'
    read(start)
    for path in [Path(__file__), ROOT/'tools/hd/explore_sprite_sources.py',
                 ROOT/'docs/re/011-observation-addresses.md', ORIG/'PW.EXE']:
        read(path)
    poses = []
    archives = {}
    for path in sorted(ORIG.glob('*.PBL')):
        entries = strict_pbl(read(path))
        archives[path.name] = len(entries)
        poses += [(f'{path.name}:{i}',w,h,pixels,offset,end) for i,w,h,pixels,offset,end in entries]
    assert len(archives) == 26 and len(poses) == 537
    routes = [('bbs',[('left',42500000),('up',48500000)]),
              ('radar',[('up',42500000),('right',48500000),('up',54500000)])]
    result = []
    for label, keys in routes:
        stem = OUT/(PREFIX+'-'+label)
        frame = stem.with_suffix('.frame')
        png = stem.with_suffix('.png')
        state = stem.with_suffix('.state')
        samples = []
        at_steps = [42000001,48000000,54000000,60000000,65999999]
        argv = [str(binary),'-load-state',str(start),'-root',str(ORIG),'-steps','66000000',
                '-dump-screen',str(frame),'-dump-screen-png',str(png),
                '-save-state','65999999:'+str(state),
                '-hold',','.join(f'{key}@{at}+3000000' for key,at in keys),
                '-hold-typematic=false',
                '-shots',','.join(f'{at}:{stem}-step{at}.frame' for at in at_steps),
                '-peek','lin:57ef:2,lin:16966:20']
        execution = run(argv,stem.with_suffix('.log'))
        text = read(stem.with_suffix('.log')).decode()
        irq = re.search(r'硬體鍵盤：送出 IRQ1 (\d+) 次',text)
        assert irq and int(irq.group(1)) == len(keys)*2, '正常按鍵未實際送出'
        for path in [state,png]:
            read(path)
        for at, path in [(at,Path(str(stem)+f'-step{at}.frame')) for at in at_steps]+[(66000000,frame)]:
            f = read(path)
            assert len(f)==64000 and max(f)<16
            region = crop(f,4,124,72,72)
            matches = [key for key,w,h,pixels,o,end in poses if (w,h,pixels)==(72,72,region)]
            samples.append({'step':at,'frame':str(path.relative_to(ROOT)),
                            'room_rect':[4,124,72,72],'whole_matches':matches,
                            'region_sha256':hashlib.sha256(region).hexdigest()})
        result.append({'label':label,'keys':keys,'irq1_count':int(irq.group(1)),
                       'execution':execution,'state_save_step':65999999,
                       'saved_state':str(state.relative_to(ROOT)),'samples':samples})
        print(label,[(s['step'],s['whole_matches']) for s in samples],flush=True)
    output=OUT/(PREFIX+'.json')
    with output.open('x') as file:
        json.dump({'scope':'07-first-play正常步行到鄰近房間；僅探索完整來源',
                   'python_version':platform.python_version(),'probe_binary_sha256':digest(binary),
                   'inputs_sha256':inputs,'archives':archives,'routes':result,
                   'limits':'不注入RAM、seed或座標；標籤來自既有地圖線索，房間身份與實際座標依原版輸出查證。未觀察繪製掛鉤、未驗HD；state比最終frame早一指令。'},file,ensure_ascii=False,indent=2)
        file.write('\n')


if __name__=='__main__':
    main()
