"""研究038 §107：由原始貼圖來源重建角色區，核對正常來源身份與負對照。"""
from pathlib import Path
import copy
import hashlib
import json
import sys
sys.path.insert(0, '/src/tools')
import pbl

work=Path(sys.argv[1])
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
trace=json.loads((work/'trace.json').read_text())
for path,want in trace['inputs_sha256'].items():assert sha(path)==want,path
assert trace['inputs_sha256']['/orig/psychic-war/PW.EXE']=='88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49'
archive=Path('/orig/psychic-war/ENEMY00.PBL').read_bytes()
poses={i:bytes(pbl.decode(archive,pbl.images(archive)[i][1])[2]) for i in range(3)}
packed={i:bytes((px[j]<<4)|px[j+1] for j in range(0,len(px),2)) for i,px in poses.items()}
crop=lambda frame:bytes(v for y in range(152,184) for v in frame[y*320+32:y*320+56])
initial=(work/'initial.frame').read_bytes()
assert len(initial)==64000 and crop(initial)==poses[0]

def reconstruct(events):
    model=bytearray(poses[0])
    logical=0
    rows=[]
    for index,event in enumerate(events):
        assert bytes(model)==bytes.fromhex(event['before_hex']), f'unobserved region change before event {index}'
        x,y,w,h=event['rect'];kind=event['kind'];mode=event['regs']['AX']&255
        assert event['regs']['CS']==0x161
        raw=bytes.fromhex(event['packed_hex'])
        if kind=='packed':
            assert mode in (0,1) and event['regs']['IP']==0x8705 and len(raw)==w*h//2
        else:
            assert kind=='mask-xor10' and event['regs']['IP']==0x8260
            assert event['regs']['DS']==0x161 and event['regs']['DX']==0x4e36
            assert w==h==16 and len(raw)==32
            assert raw==bytes.fromhex('0fe01038403c402e4b46bb46ba47bac7bdc7bfc747c65bce291810580fe00000')
            pos=event['regs']['BX']-0xc000
            assert pos>=0 and (x,y)==(pos%80*4,pos//80)
        before=logical
        full_before=[i for i,px in poses.items() if bytes(model)==px]
        covering=x<=32 and y<=152 and x+w>=56 and y+h>=184
        recognized=False
        if covering and kind=='packed':
            logical=None
            if (x,y,w,h)==(32,152,24,32):
                if mode==0:
                    matches=[i for i,source in packed.items() if raw==source]
                elif before is not None:
                    matches=[i for i,source in packed.items() if abs(i-before)==1 and raw==bytes(a^b for a,b in zip(packed[before],source))]
                else:
                    matches=[]
                assert len(matches)<=1
                if matches:logical=matches[0];recognized=True
        for yy in range(max(y,152),min(y+h,184)):
            for xx in range(max(x,32),min(x+w,56)):
                target=(yy-152)*24+xx-32
                if kind=='packed':
                    at=(yy-y)*w+xx-x
                    value=raw[at//2]>>(4 if at%2==0 else 0)&15
                    model[target]=(model[target]^value) if mode else value
                elif raw[(yy-y)*2+(xx-x)//8] & (0x80>>((xx-x)%8)):
                    model[target]^=10
        assert bytes(model)==bytes.fromhex(event['after_hex']), f'packed source model mismatch at event {index}'
        visible=[]
        if logical is not None:
            px=poses[logical]
            for cy in range(0,32,8):
                for cx in range(0,24,8):
                    want=bytes(v for yy in range(cy,cy+8) for v in px[yy*24+cx:yy*24+cx+8])
                    actual=bytes(v for yy in range(cy,cy+8) for v in model[yy*24+cx:yy*24+cx+8])
                    if any(want) and actual==want:visible.append([cx+32,cy+152])
        rows.append({'event':index,'kind':kind,'entry':event['entry'],'return':event['return'],'rect':event['rect'],'mode':mode,
                     'source_DS_BX':[event['regs']['DS'],event['regs']['BX']], 'logical_before':before,
                     'logical_after':logical,'full_pose_before':full_before,'recognized_body_source':recognized,
                     'eligible_original8x8_cells':visible,'original_region_source_model_mismatch':0})
    return rows,bytes(model)

rows,model=reconstruct(trace['events'])
assert model==crop((work/'observed.frame').read_bytes())==crop((work/'control.frame').read_bytes())
assert any(r['logical_after']==2 and r['eligible_original8x8_cells'] for r in rows)
assert [r['logical_after'] for r in rows if r['recognized_body_source']]==[1,2,1,0]*4
negative={}
bad=copy.deepcopy(trace['events']);raw=bytearray.fromhex(bad[0]['packed_hex']);raw[0]^=1;bad[0]['packed_hex']=raw.hex()
# The first event may extend outside the actor. Choose an intersecting source pixel.
x,y,w,h=bad[0]['rect'];at=(max(y,152)-y)*w+max(x,32)-x
raw=bytearray.fromhex(trace['events'][0]['packed_hex']);raw[at//2]^=(16 if at%2==0 else 1);bad[0]['packed_hex']=raw.hex()
try:reconstruct(bad)
except AssertionError:negative['changed_source_pixel']=True
else:raise AssertionError('changed source negative passed')
bad=copy.deepcopy(trace['events']);changed=next(e for e in bad if e['kind']=='packed' and e['regs']['AX']&255==1);changed['regs']['AX']&=0xff00
try:reconstruct(bad)
except AssertionError:negative['wrong_xor_mode']=True
else:raise AssertionError('wrong mode negative passed')
bad=copy.deepcopy(trace['events']);bad.pop(0)
try:reconstruct(bad)
except AssertionError:negative['missing_intersecting_event']=True
else:raise AssertionError('missing event negative passed')
bad=copy.deepcopy(trace['events']);i=next(i for i,e in enumerate(bad) if e['kind']=='mask-xor10');bad.pop(i)
try:reconstruct(bad)
except AssertionError:negative['missing_mask_event']=True
else:raise AssertionError('missing mask negative passed')
machine=json.loads((work/'machine.json').read_text())
assert machine['field_count']==44 and machine['machine_fields_equal'] and machine['dos_gob_equal']
assert machine['negative_cpu_changed'] and machine['negative_port_changed'] and machine['negative_ram_changed']
inputs={str(path):sha(path) for path in [work/'trace.json',work/'machine.json',work/'initial.frame',work/'observed.frame',work/'control.frame',Path(__file__),Path(pbl.__file__)]}
result={'scope':'normal saved full-pose0 checkpoint, no-input wait, independent all-intersecting source model; prototype source identity only',
        'rows':rows,'negative_controls':negative,'machine_fields_equal':44,'dos_section_equal':True,
        'pose2_eligible_occurrences':sum(r['logical_after']==2 and bool(r['eligible_original8x8_cells']) for r in rows),
        'known_body_transitions_without_full_predecessor':sum(r['recognized_body_source'] and not r['full_pose_before'] for r in rows),
        'inputs_sha256':inputs,'limits':'Trusted logical identity derives only from initial full PBL pose and known adjacent raw sources, never current runtime HD flags. Unknown covering writes clear identity. No production relaxation, HD artwork, full animation or GUI/save/release claim.'}
out=work/(sys.argv[2] if len(sys.argv)>2 else 'independent.json');assert not out.exists()
out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({key:result[key] for key in ['pose2_eligible_occurrences','known_body_transitions_without_full_predecessor','negative_controls','machine_fields_equal']}))
