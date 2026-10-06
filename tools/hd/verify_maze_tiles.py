"""研究038 §71：MAZE.BIN的4×4原版圖塊、正常視野與載入記憶體獨立核對。"""
from collections import Counter
import hashlib
import json
from pathlib import Path
import platform
import subprocess
import time


ROOT = Path('/src')
OUT = ROOT/'workplace/hd'
PREFIX = 'maze-tile-sources-v2-20261003'
ORIG = Path('/orig/psychic-war')


def main():
    assert not list(OUT.glob(PREFIX+'*')), '拒絕覆寫'
    assert OUT.stat().st_uid == OUT.stat().st_gid == 1000
    inputs={}

    def read(path):
        assert path.stat().st_uid == path.stat().st_gid == 1000
        b=path.read_bytes()
        key=str(path.relative_to(ROOT)) if path.is_relative_to(ROOT) else str(path)
        inputs[key]=hashlib.sha256(b).hexdigest()
        return b

    raw=read(ORIG/'MAZE.BIN')
    assert len(raw)==2048 and hashlib.sha256(raw).hexdigest()=='8b3b08ef483ff8c64beb04c69c861763c1258afba2348a7058e828bcf4060756'
    read(ORIG/'PW.EXE')
    for path in [Path(__file__),ROOT/'tools/hd/explore_room_sources.py',
                 ROOT/'tools/hd/explore_sprite_sources.py',ROOT/'docs/re/011-observation-addresses.md']:
        read(path)
    atlas={}
    slots=[]
    for slot in range(256):
        pixels=bytes(value for b in raw[slot*8:slot*8+8] for value in (b>>4,b&15))
        atlas.setdefault(pixels,[]).append(slot)
        slots.append({'slot':slot,'file_offset':slot*8,'size':[4,4],
                      'pixels_sha256':hashlib.sha256(pixels).hexdigest(),
                      'nonzero_pixels':sum(bool(value) for value in pixels)})

    def region(frame,x,y,w=4,h=4):
        assert len(frame)==64000 and 0<=x<=320-w and 0<=y<=200-h
        return b''.join(frame[(y+r)*320+x:(y+r)*320+x+w] for r in range(h))

    rows=[]
    frame_paths=sorted((ROOT/'workplace/states').glob('*.frame'))
    for label in ['bbs','radar']:
        frame_paths+=sorted(OUT.glob('room-nearby-explore-v1-20261003-'+label+'*.frame'))
    unique_tile_shapes=set()
    for path in frame_paths:
        frame=read(path)
        assert len(frame)==64000 and max(frame)<16
        cells=[]
        for y in range(124,196,4):
            for x in range(4,76,4):
                pixels=region(frame,x,y)
                matches=atlas.get(pixels,[])
                if matches:
                    unique_tile_shapes.add(pixels)
                cells.append({'x':x,'y':y,'slots':matches,
                              'pixels_sha256':hashlib.sha256(pixels).hexdigest()})
        unmatched=[cell for cell in cells if not cell['slots']]
        if not unmatched:
            rebuilt=bytearray(72*72)
            for cell in cells:
                tile=raw[cell['slots'][0]*8:cell['slots'][0]*8+8]
                decoded=bytes(value for b in tile for value in (b>>4,b&15))
                for row in range(4):
                    start=(cell['y']-124+row)*72+cell['x']-4
                    rebuilt[start:start+4]=decoded[row*4:(row+1)*4]
            expected=region(frame,4,124,72,72)
            assert bytes(rebuilt)==expected
            one_pixel=bytearray(rebuilt)
            one_pixel[0]^=1
            assert sum(a!=b for a,b in zip(one_pixel,expected,strict=True))==1
            shifted=sum(region(frame,cell['x']+1,cell['y']) not in atlas for cell in cells)
            source_changed=bytearray(raw)
            used=next((cell for cell in cells if any(region(frame,cell['x'],cell['y']))),cells[0])
            source_changed[used['slots'][0]*8]^=0x80
            changed=source_changed[used['slots'][0]*8:used['slots'][0]*8+8]
            changed_pixels=bytes(value for b in changed for value in (b>>4,b&15))
            assert changed_pixels!=region(frame,used['x'],used['y'])
            full={'rebuild_mismatch':0,'negative_one_pixel':1,
                  'negative_shift_unmatched_cells':shifted,'negative_source_tile_changed_pixels':1}
        else:
            full=None
        rows.append({'frame':str(path.relative_to(ROOT)), 'viewport':[4,124,72,72],
                     'cells':cells,'matched_cells':324-len(unmatched),'unmatched_cells':len(unmatched),
                     'nonzero_cells':sum(any(region(frame,cell['x'],cell['y'])) for cell in cells),
                     'unique_source_cells':sum(len(cell['slots'])==1 for cell in cells),
                     'ambiguous_source_cells':sum(len(cell['slots'])>1 for cell in cells),
                     'complete_rebuild':full})
    for required in ['workplace/states/07-first-play.frame',
                     'workplace/hd/room-nearby-explore-v1-20261003-bbs.frame',
                     'workplace/hd/room-nearby-explore-v1-20261003-radar.frame']:
        sample=next(row for row in rows if row['frame']==required)
        assert sample['matched_cells']==324 and sample['complete_rebuild']['negative_shift_unmatched_cells']>0

    receipt=json.loads(read(OUT/'companion-explore-v2-20261003.json'))
    probe=OUT/'companion-explore-v2-20261003-probe.bin'
    assert hashlib.sha256(read(probe)).hexdigest()==receipt['binary_sha256']
    locations=[]
    for label, state, step in [('initial',ROOT/'workplace/states/07-first-play.state',42000000),
                              ('bbs',OUT/'room-nearby-explore-v1-20261003-bbs.state',65999999),
                              ('radar',OUT/'room-nearby-explore-v1-20261003-radar.state',65999999)]:
        read(state)
        dump=OUT/(PREFIX+'-'+label+'.ram.bin')
        log=OUT/(PREFIX+'-'+label+'.log')
        argv=[str(probe),'-load-state',str(state),'-root',str(ORIG),'-steps',str(step),
              '-dump-mem','0-a0000:'+str(dump)]
        started=time.monotonic()
        with log.open('xb') as file:
            proc=subprocess.run(argv,stdout=file,stderr=subprocess.STDOUT,timeout=60)
        assert proc.returncode==0
        execution={'argv':argv,'returncode':0,'log':str(log.relative_to(ROOT)),
                   'elapsed_seconds':time.monotonic()-started}
        read(log)
        memory=read(dump)
        assert len(memory)==0xa0000, '原版RAM傾印長度不同'
        addresses=[]
        offset=0
        while True:
            found=memory.find(raw,offset)
            if found<0:break
            addresses.append(found)
            offset=found+1
        assert len(addresses)==1,'載入RAM中的MAZE完整來源不唯一'
        locations.append({'label':label,'state':str(state.relative_to(ROOT)),
                          'steps_advanced':0,'ram':str(dump.relative_to(ROOT)),
                          'source_linear_addresses':addresses,'execution':execution})
    assert len({tuple(row['source_linear_addresses']) for row in locations})==1
    output=OUT/(PREFIX+'.json')
    summary={'scope':'MAZE.BIN原版4×4內容與正常72×72視野、唯讀RAM來源',
             'python_version':platform.python_version(),'inputs_sha256':inputs,
             'source_size':2048,'tile_size':[4,4],'slots':slots,'unique_tile_shapes':len(atlas),
             'observed_tile_shapes':len(unique_tile_shapes),'samples':rows,'ram_locations':locations,
             'limits':'4×4格式與完整視野的內容重建已證實。相同bytes的多個slot只列歧義，未猜選擇器身份；精確繪製入口、動作中途與HD素材／正式契約仍未知。未改RAM、seed或遊戲規則，沒有正式HD接入。'}
    with output.open('x') as file:
        json.dump(summary,file,ensure_ascii=False,indent=2)
        file.write('\n')
    print(json.dumps({'frames':len(rows),'complete_nonblank_viewports':sum(row['matched_cells']==324 and row['nonzero_cells']>0 for row in rows),
                      'unique_tile_shapes':len(atlas),'observed_tile_shapes':len(unique_tile_shapes),
                      'ram_linear_addresses':locations[0]['source_linear_addresses'],
                      'result':str(output)},ensure_ascii=False))


if __name__=='__main__':
    main()
