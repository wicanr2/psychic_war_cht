"""研究038 §71：正常BBS路線唯讀監看MAZE.BIN圖塊選擇。"""
from collections import Counter
import argparse
import hashlib
import json
from pathlib import Path
import platform
import subprocess

from compare_maze_saved import compare_saved


ROOT=Path('/src')
OUT=ROOT/'workplace/hd'
ORIG=Path('/orig/psychic-war')
PREFIX='maze-read-source-v1-20261003'


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument('--verify-existing-v1',action='store_true',help='只核對已終止的v1原版輸出，不重跑')
    args=parser.parse_args()
    if args.verify_existing_v1:
        assert not (OUT/(PREFIX+'.json')).exists(), '拒絕覆寫收據'
    else:
        assert not list(OUT.glob(PREFIX+'*')), '拒絕覆寫'
    assert OUT.stat().st_uid==OUT.stat().st_gid==1000
    inputs={}

    def read(path):
        assert path.stat().st_uid==path.stat().st_gid==1000
        data=path.read_bytes()
        key=str(path.relative_to(ROOT)) if path.is_relative_to(ROOT) else str(path)
        inputs[key]=hashlib.sha256(data).hexdigest()
        return data

    tiles=json.loads(read(OUT/'maze-tile-sources-v2-20261003.json'))
    exploration=json.loads(read(OUT/'room-nearby-explore-v1-20261003.json'))
    binary=OUT/'companion-explore-v2-20261003-probe.bin'
    assert hashlib.sha256(read(binary)).hexdigest()==exploration['probe_binary_sha256']
    original=read(ORIG/'MAZE.BIN')
    read(ORIG/'PW.EXE')
    read(ROOT/'workplace/states/07-first-play.state')
    for path in [Path(__file__),ROOT/'worktrees/dosgolem/cmd/probe/observe.go',
                 ROOT/'worktrees/dosgolem/internal/machine/machine.go']:
        read(path)
    address=tiles['ram_locations'][0]['source_linear_addresses'][0]
    assert address==0x12e16
    route=next(row for row in exploration['routes'] if row['label']=='bbs')
    assert route['keys']==[['left',42500000],['up',48500000]] and route['irq1_count']==4
    argv=route['execution']['argv'].copy()
    for flag in ['-dump-screen','-dump-screen-png','-save-state','-shots']:
        i=argv.index(flag)
        del argv[i:i+2]
    argv+=['-dump-screen',str(OUT/(PREFIX+'.frame')),
           '-save-state','65999999:'+str(OUT/(PREFIX+'.state')),
           '-dump-mem','0-a0000:'+str(OUT/(PREFIX+'.ram.bin')),
           '-read-watch',f'{address:x}-{address+len(original)-1:x}',
           '-read-watch-grain','8','-read-watch-file',str(OUT/(PREFIX+'.reads.txt'))]
    log=OUT/(PREFIX+'.log')
    if not args.verify_existing_v1:
        with log.open('xb') as file:
            process=subprocess.run(argv,stdout=file,stderr=subprocess.STDOUT,timeout=90)
        assert process.returncode==0
    else:
        assert '停止原因：跑滿 66000000 道指令上限' in log.read_text()
    read(log)
    frame=read(OUT/(PREFIX+'.frame'))
    assert frame==read(OUT/'room-nearby-explore-v1-20261003-bbs.frame')
    snapshot=read(OUT/(PREFIX+'.state'))
    baseline=OUT/'room-nearby-explore-v1-20261003-bbs.state'
    compressed_bytes_equal=snapshot==read(baseline)
    saved_comparison=compare_saved(ROOT,OUT,baseline,OUT/(PREFIX+'.state'),inputs)
    ram=read(OUT/(PREFIX+'.ram.bin'))
    assert len(ram)==0xa0000 and ram[address:address+len(original)]==original
    events=[]
    for line in read(OUT/(PREFIX+'.reads.txt')).decode().splitlines():
        step,addr,slot,pc=line.split()
        step,addr,slot=int(step),int(addr,16),int(slot)
        assert address<=addr<address+len(original) and slot==(addr-address)//8 and 0<=slot<256
        events.append({'step':step,'linear_source':addr,'slot':slot,'runtime_read_pc':pc,
                       'file_offset':slot*8,'tile_sha256':tiles['slots'][slot]['pixels_sha256']})
    assert 0<len(events)<200000, '未觀察到MAZE讀取或達收集上限'
    pc_counts=Counter(event['runtime_read_pc'] for event in events)
    output=OUT/(PREFIX+'.json')
    with output.open('x') as file:
        json.dump({'scope':'正常BBS路線MAZE圖塊讀取，唯讀觀察／既有原版終點完整state欄位與frame相同',
                   'python_version':platform.python_version(),'inputs_sha256':inputs,
                   'argv':argv,'returncode':0,'runtime_address_space':'x86實模式線性來源位址及CS:IP讀取觀察點',
                   'source_linear_start':address,'source_size':len(original),'read_grain':8,
                   'transition_count':len(events),'runtime_read_pc_counts':dict(pc_counts),
                   'unique_observed_slots':sorted({event['slot'] for event in events}),
                   'events':events,'frame_mismatch':0,'compressed_state_bytes_equal':compressed_bytes_equal,
                   'saved_state_comparison':saved_comparison,
                   'limits':'read-watch只記圖塊切換，連續相同slot會合併，不是完整貼圖次數。讀取PC不是函式入口；尚未還原繪製中途或正式HD契約。'},file,ensure_ascii=False,indent=2)
        file.write('\n')
    print(json.dumps({'read_transitions':len(events),'runtime_read_pc_counts':dict(pc_counts),
                      'unique_slots':len({event['slot'] for event in events}),
                      'frame_and_full_saved_state_fields_equal':True,'result':str(output)},ensure_ascii=False))


if __name__=='__main__':
    main()
