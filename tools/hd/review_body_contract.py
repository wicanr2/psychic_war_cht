"""研究038 §108：首組來源唯一性及清除契約的證據審查。"""
from pathlib import Path
import hashlib,json,sys
sys.path.insert(0,'/src/tools')
import pbl
root=Path('/src')
out=root/'workplace/hd/body-runtime-v1-20261004'
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
assert out.parent.stat().st_uid==out.parent.stat().st_gid==1000
out.mkdir()
previous=root/'workplace/hd/body-context-v2-20261004/independent-v2.json'
proof=json.loads(previous.read_text())
for p,digest in proof['inputs_sha256'].items():assert sha(p)==digest
assert proof['machine_fields_equal']==44 and all(proof['negative_controls'].values())
assert [r['logical_after'] for r in proof['rows'] if r['recognized_body_source']]==[1,2,1,0]*4
inputs={str(previous):sha(previous),str(Path(__file__)):sha(__file__),str(Path(pbl.__file__)):sha(pbl.__file__)}
images={}
for name,nums in [('ENEMY00.PBL',range(9)),('ENEMY01.PBL',range(6,9)),('ENEMY03.PBL',[3])]:
    path=Path('/orig/psychic-war')/name
    raw=path.read_bytes();inputs[str(path)]=sha(path)
    for n in nums:
        w,h,px=pbl.decode(raw,pbl.images(raw)[n][1]);assert (w,h)==(24,32)
        images[(name,n)]=bytes((px[i]<<4)|px[i+1] for i in range(0,768,2))
assert len(set(images.values()))==len(images), 'same-position immutable sources ambiguous'
edge={0:[1],1:[0,2],2:[1]}
def targets(previous,mode,raw,invalidated=False):
    if invalidated or len(previous)!=1 or mode!=1:return []
    name,n=previous[0]
    if name!='ENEMY00.PBL' or n not in edge:return []
    return [j for j in edge[n] if raw==bytes(a^b for a,b in zip(images[(name,n)],images[(name,j)]))]
delta=lambda a,b:bytes(x^y for x,y in zip(images[('ENEMY00.PBL',a)],images[('ENEMY00.PBL',b)]))
cases=[('forward', [('ENEMY00.PBL',0)],1,delta(0,1),False,[1]),
       ('reverse', [('ENEMY00.PBL',1)],1,delta(0,1),False,[0]),
       ('next', [('ENEMY00.PBL',1)],1,delta(1,2),False,[2]),
       ('reverse-next', [('ENEMY00.PBL',2)],1,delta(1,2),False,[1]),
       ('no identity', [],1,delta(0,1),False,[]),
       ('wrong previous', [('ENEMY00.PBL',2)],1,delta(0,1),False,[]),
       ('cross group', [('ENEMY00.PBL',3)],1,delta(0,1),False,[]),
       ('cross file', [('ENEMY01.PBL',6)],1,delta(0,1),False,[]),
       ('ambiguous', [('ENEMY00.PBL',0),('ENEMY00.PBL',3)],1,delta(0,1),False,[]),
       ('unknown source', [('ENEMY00.PBL',0)],1,bytes(384),False,[]),
       ('unknown mode', [('ENEMY00.PBL',0)],2,delta(0,1),False,[]),
       ('cover cleared', [('ENEMY00.PBL',0)],1,delta(0,1),True,[]),
       ('reload cleared', [('ENEMY00.PBL',1)],1,delta(1,2),True,[])]
for label,prior,mode,raw,clear,want in cases:assert targets(prior,mode,raw,clear)==want,label
for n in edge:
    assert len({delta(n,j) for j in edge[n]})==len(edge[n])
doc={'scope':'evidence review of bounded ENEMY00 #0-2 identity contract, not production test',
     'unique_same_position_immutable_sources':len(images),'cases':[c[0] for c in cases],
     'four_directed_edges_unambiguous':True,'source_proof_sha256':sha(previous),
     'inputs_sha256':inputs,'limits':'Production must still verify source lifecycle, original8x8 cells, reset, anchors and all existing groups. No art acceptance, GUI or package claim.'}
with (out/'contract-review.json').open('x') as f:f.write(json.dumps(doc,ensure_ascii=False,indent=2)+'\n')
print('13契約案例及13份同位置來源唯一性審查通過')
