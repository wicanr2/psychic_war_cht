"""研究038 §108：由原版來源獨立核對正式27筆圖面與載回。"""
from pathlib import Path
import json,sys
sys.path.insert(0,'/src/tools/hd')
import verify_ally_items as art
from verify_next_body_plane import region,sha,require
from verify_over_frontend import mismatch
from verify_over_runtime import read

root=Path('/src')
work=root/'workplace/hd/body-runtime-v1-20261004'
art.THEME=root/'workplace/hd/theme-kasuruji-pose2-v1-20261004'
inputs={}
doc=json.loads(read(work/'runtime.json',inputs))
for p,want in doc['inputs_sha256'].items():require(sha(read(p,inputs))==want,'執行來源變動：'+p)
source=json.loads(read(root/'workplace/hd/body-context-v2-20261004/trace.json',inputs))
proof=json.loads(read(root/'workplace/hd/body-context-v2-20261004/independent-v2.json',inputs))
for p,want in proof['inputs_sha256'].items():require(sha(read(p,inputs))==want,'來源模型輸入變動：'+p)
assets,decoded=art.assets_from_files(inputs)
before=root/'workplace/hd/theme-ally-items-v1-20261004'
old=json.loads(read(before/'manifest.json',inputs))
new=json.loads(read(art.THEME/'manifest.json',inputs))
require(old['entries']==new['entries'] and len(new['entries'])==27,'位置或清單改動')
for e in old['entries']:
    if (e['pbl'],e['image'])!=('ENEMY00.PBL',2):
        require(read(before/e['png'],inputs)==read(art.THEME/e['png'],inputs),'其他素材改動')
require(read(art.THEME/'ENEMY00-02.png',inputs)==read(root/'workplace/hd/redraw/ENEMY00-group0-v13-frame-02-20261004.png',inputs),'非已初審v13')
events=[r for r in proof['rows'] if r['recognized_body_source']]
initial=read(work/'initial.frame',inputs)
require(region(initial,32,152,24,32)==decoded['ENEMY00.PBL',0][2],'起點角色不完整')
require(region(initial,264,152,24,32)==decoded['ALLY.PBL',0][2],'起點盟友不完整')
rows=[]
poses2=0
for row in doc['samples']:
    label,step=row['label'],row['step']
    frame=read(row['prefix']+'.frame',inputs)
    require(sha(frame)==row['original_frame_sha256'],'原版frame已變動')
    machine=json.loads(read(row['prefix']+'-machine.json',inputs))
    require(machine['field_count']==44 and machine['machine_fields_equal'] and machine['dos_gob_equal'],'原版完整state不符')
    require(all(machine[k] for k in ['negative_cpu_changed','negative_port_changed','negative_ram_changed']),'機器負對照無效')
    active=set()
    pose=None
    if not label.startswith('reload-mid'):
        pose=0
        for event in events:
            if event['entry']<step:pose=event['logical_after']
        active={('ENEMY00.PBL',pose,32,152),('ALLY.PBL',0,264,152)}
    actual=read(row['prefix']+'-plane.rgba',inputs)
    expected,_=art.art_plane(frame,assets,active)
    require(mismatch(actual,expected)==0,'完整27筆圖面不符：'+label)
    body_cells=[]
    negative=0
    if pose is not None:
        src=decoded['ENEMY00.PBL',pose][2]
        for y in range(0,32,8):
            for x in range(0,24,8):
                want=b''.join(src[yy*24+x:yy*24+x+8] for yy in range(y,y+8))
                if any(want) and region(frame,x+32,y+152,8,8)==want:body_cells.append([x+32,y+152])
    if pose==2:
        omitted=[a for a in assets if (a[0]['pbl'],a[0]['image'])!=('ENEMY00.PBL',2)]
        omit,_=art.art_plane(frame,omitted,active)
        negative=mismatch(expected,omit)
        require(negative>0 and body_cells,'第2張省略負對照無效：'+label)
        poses2+=1
    rows.append({'label':label,'step':step,'source_pose':pose,'body_cells':body_cells,
                 'whole_plane_mismatch':0,'negative_omit_pose2_pixels':negative,'machine_fields_equal':44})
require(len(rows)==17 and poses2>=4,'正常及載回範圍不足')
require(read(work/'pose2-return-plane.rgba',inputs)==read(work/'reload-full-next-pose2-plane.rgba',inputs),'真正完整起點載回接續圖面不同')
require(read(work/'pose2-return.frame',inputs)==read(work/'reload-full-next-pose2.frame',inputs),'真正載回接續原版frame不同')
for name in ['verify_body_runtime.py','verify_ally_items.py','verify_next_body_plane.py','verify_over_frontend.py','verify_over_runtime.py','explore_sprite_sources.py']:
    p=root/'tools/hd'/name;inputs[str(p)]=sha(p.read_bytes())
out=work/'independent.json';require(not out.exists(),'拒絕覆寫')
out.write_text(json.dumps({'scope':'formal Theme 17 full960x600 art planes, normal saved full-pose0 continuation and genuine full/partial reloads',
                          'results':rows,'normal_pose2_samples':poses2,'inputs_sha256':inputs,
                          'limits':'No text/window/DAT/package/full-animation claim. Cold partial reload correctly loses identity and falls back until full original source.'},ensure_ascii=False,indent=2)+'\n')
print('17份完整圖面差0，第2張',poses2,'份正常／載回HD可見，省略負對照有效')
