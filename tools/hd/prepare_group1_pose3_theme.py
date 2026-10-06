"""研究038 §86：只選入已審查及正常驗證的ENEMY00 #3；Docker專用。"""
from pathlib import Path
import hashlib,json,shutil
root=Path('/src');review=root/'workplace/hd/art-group1-pose3-review-v1-20261003'
base=root/'workplace/hd/theme-oogus-pose6-v2-20261003'
target=root/'workplace/hd/theme-group1-pose3-v1-20261003'
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
assert not target.exists()
assert(review.stat().st_uid,review.stat().st_gid)==(1000,1000)
art=json.loads((review/'art-review.json').read_text())
proof=json.loads((review/'independent.json').read_text())
runtime=json.loads((review/'normal.json').read_text())
assert art['image']==3 and art['origin']==[32,152] and art['size']==[72,96]
assert art['candidate_ink_bounds_threshold60']==art['original_ink_bounds_hd']==[0,0,65,95]
assert art['reference_pixels_verified']==248832 and art['reference_mismatch']==0 and art['negative_reference_pixel']==1
assert len(proof['rows'])==16 and proof['new_pose3_occurrences']==4
assert proof['original_HD_branches_equal'] and proof['partial_checks']==26 and proof['runtime_reload_states']==16
assert all(r['complete_composition_mismatch']==0 and r['negative_omit_enemy_pixels']>0 and r['negative_wrong_pose_pixels']>0 for r in proof['rows'])
assert all(r['hd_mismatch']==0 for r in runtime['partial_checks'])
inputs={}
for doc in [art,proof,runtime]:
    for path,h in doc['inputs_sha256'].items():
        p=Path(path);p=p if p.is_absolute() else root/p
        assert p.is_file() and sha(p)==h,str(p)
        inputs[str(p)]=h
manifest=json.loads((base/'manifest.json').read_text());assert len(manifest['entries'])==25
candidate=review/'theme';assert sha(candidate/'manifest.json')==sha(base/'manifest.json')
for e in manifest['entries']:
    p=candidate/e['png']
    if e['pbl']=='ENEMY00.PBL' and e['image']==3:
        assert sha(p)==art['inputs_sha256']['/src/workplace/hd/redraw/ENEMY00-group1-v5-frame-03-20261003.png']
    else:assert sha(p)==sha(base/e['png']),e['png']
target.mkdir();outputs={}
for name in ['manifest.json']+[e['png'] for e in manifest['entries']]:
    src=candidate/name;dst=target/name;dst.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(src,dst)
    assert sha(dst)==sha(src) and(dst.stat().st_uid,dst.stat().st_gid)==(1000,1000)
    outputs[str(dst)]=sha(dst)
inputs[str(Path(__file__))]=sha(Path(__file__))
accepted=dict(art);accepted.update({'normal_gate':'passed','formal_art_accepted':True,'accepted_images':[3],'normal_samples':16,'new_pose3_occurrences':4,'limits':'single pose3 art and specified normal/checkpoint paths; other poses4/5, GUI/DAT/fullsprites/release incomplete'})
(review/'art-review-accepted.json').write_text(json.dumps(accepted,ensure_ascii=False,indent=2)+'\n')
selection={'scope':'local25 theme; acceptedENEMY00 image3 only; other4/5 art incomplete','entries':25,'base':str(base),'changed_assets':['ENEMY00-03.png'],'accepted_images':[3],'inputs_sha256':inputs,'outputs_sha256':outputs,'manifest_sha256':outputs[str(target/'manifest.json')]}
(target/'selection.json').write_text(json.dumps(selection,ensure_ascii=False,indent=2)+'\n')
(review/'selection.json').write_text(json.dumps(selection,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({'theme':str(target),'entries':25,'changed_assets':selection['changed_assets'],'formal_enemy_art': '5/360'}))
