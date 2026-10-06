"""研究038 §73：不推進指令，讀取既有正常ROOM22狀態的MAZE表與畫面重疊。"""
import hashlib
import json
from pathlib import Path
import subprocess

from explore_sprite_sources import strict_pbl


ROOT=Path('/src')
OUT=ROOT/'workplace/hd'
PREFIX='maze-room-overlap-v1-20261003'


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    assert not list(OUT.glob(PREFIX+'*')),'拒絕覆寫'
    assert OUT.stat().st_uid==OUT.stat().st_gid==1000
    inputs={}
    def read(path):
        value=path.read_bytes();inputs[str(path)]=hashlib.sha256(value).hexdigest();return value
    read(Path(__file__));read(ROOT/'tools/hd/explore_sprite_sources.py')
    source=json.loads(read(OUT/'room22-runtime-v1-20261003.json'))
    archive=json.loads(read(OUT/'room22-source-manifest-v1-20261003.json'))
    archived=archive['files']
    if isinstance(archived,list):
        expected={x['source']:x for x in archived}
    else:
        expected=archived
    tool=OUT/'companion-explore-v2-20261003-probe.bin'
    compiled=json.loads(read(OUT/'companion-explore-v2-20261003.json'))
    assert hashlib.sha256(read(tool)).hexdigest()==compiled['binary_sha256']
    orig=Path('/orig/psychic-war')
    maze=read(orig/'MAZE.BIN');assert len(maze)==2048
    assert hashlib.sha256(maze).hexdigest()=='8b3b08ef483ff8c64beb04c69c861763c1258afba2348a7058e828bcf4060756'
    read(orig/'PW.EXE')
    room=strict_pbl(read(orig/'ROOM0.PBL'))
    assert len(room)==31
    samples=[]
    for ordinal,sample in enumerate(source['samples']):
        original=sample['original'];start=ROOT/(sample['prefix']+'.state')
        key=str(start.relative_to(ROOT));assert key in expected
        assert hashlib.sha256(read(start)).hexdigest()==expected[key]['sha256']
        stem=OUT/f'{PREFIX}-sample{ordinal:02d}'
        frame_path=stem.with_suffix('.frame');ram_path=stem.with_suffix('.ram.bin');log_path=stem.with_suffix('.log')
        argv=[str(tool),'-load-state',str(start),'-root',str(orig),'-steps',str(sample['step']),
              '-dump-screen',str(frame_path),'-dump-mem','0-a0000:'+str(ram_path)]
        with log_path.open('xb') as stream:
            process=subprocess.run(argv,stdout=stream,stderr=subprocess.STDOUT,timeout=40)
        assert process.returncode==0,log_path
        read(log_path);frame=read(frame_path);ram=read(ram_path)
        assert len(frame)==64000 and len(ram)==0xa0000
        assert hashlib.sha256(frame).hexdigest()==original['Frame']
        assert hashlib.sha256(ram).hexdigest()==original['RAM']
        assert ram[0x12e16:0x13616]==maze
        table=ram[0x468b:0x468b+324]
        desired=bytearray(72*72)
        for i,slot in enumerate(table):
            raw=maze[slot*8:slot*8+8]
            pixels=bytes(v for b in raw for v in (b>>4,b&15))
            x,y=i%18*4,i//18*4
            for row in range(4):desired[(y+row)*72+x:(y+row)*72+x+4]=pixels[row*4:row*4+4]
        actual=b''.join(frame[y*320+4:y*320+76] for y in range(124,196))
        matching=[i for i,w,h,pixels,_,_ in room if (w,h)==(72,72) and pixels==actual]
        table_rooms=[i for i,w,h,pixels,_,_ in room if (w,h)==(72,72) and pixels==desired]
        regs=original['Regs']
        samples.append({'sample':ordinal,'step':sample['step'],'state':str(start),'argv':argv,
                        'frame':str(frame_path),'ram':str(ram_path),'original_frame_ram_equal':True,
                        'runtime_pc':[regs['CS'],regs['IP']],
                        'maze_row_return_pc':regs['CS']==0x161 and regs['IP'] in [0x5002,0x5010,0x501e,0x502c],
                        'actual_room0_matches':matching,'table_room0_matches':table_rooms,
                        'maze_view_mismatch':sum(a!=b for a,b in zip(actual,desired)),
                        'table_sha256':hashlib.sha256(table).hexdigest(),'slots':list(table)})
    result={'scope':'既有正常Up/Down/Up的13份state不推進指令，核對MAZE表與ROOM圖面重疊',
            'inputs_sha256':inputs,'samples':samples,
            'limits':'不重跑玩家輸入，不宣稱正式優先序。已保存PC及table是本批正常state證據，不能外推其他ROOM/場景。'}
    (OUT/(PREFIX+'.json')).write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print([(s['sample'],s['actual_room0_matches'],s['table_room0_matches'],s['maze_view_mismatch'],s['maze_row_return_pc']) for s in samples])


if __name__=='__main__':main()
