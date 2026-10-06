"""研究038 §108：保全正式Theme來源、實際建置依賴、素材與核對收據。"""
from pathlib import Path
import hashlib,json,shutil,subprocess

root=Path('/src'); work=root/'workplace/hd/body-runtime-v1-20261004'
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
known={}; bases={}
for name in ['gui-ally-items-v1-20261004','body-context-v2-20261004']:
    p=root/'workplace/hd'/name/'source-manifest.json';m=json.loads(p.read_text());bases[str(p)]=sha(p)
    known.update({r['source']:r for r in m['files']})
text=(work/'build-deps.json').read_text();decoder=json.JSONDecoder();sources=set()
while text.strip():
    p,end=decoder.raw_decode(text.lstrip());text=text.lstrip()[end:]
    if p.get('Standard'):continue
    folder=Path(p['Dir'])
    for group in ['GoFiles','CgoFiles','CFiles','CXXFiles','HFiles','SFiles','EmbedFiles']:
        sources.update(folder/name for name in p.get(group,[]))
    if p.get('Module',{}).get('GoMod'):sources.add(Path(p['Module']['GoMod']))
dependency_count=len(sources)
originals={}
for doc in ['runtime.json','independent.json','art-review-accepted.json','contract-review.json']:
    d=json.loads((work/doc).read_text())
    for p,h in d.get('inputs_sha256',{}).items():
        path=Path(p) if Path(p).is_absolute() else root/p
        assert sha(path)==h,p
        if p.startswith('/orig/') or p.endswith('.state') and 'next-body-normal-' in p:originals[str(path)]=h
        else:sources.add(path)
sources.update(p for p in work.iterdir() if p.is_file())
sources.update(p for p in (root/'workplace/hd/theme-kasuruji-pose2-v1-20261004').iterdir() if p.is_file())
sources.update(root/p for p in ['go.mod','go.sum','workplace/hd/dat-runtime-v1-20261002.go.work',
    'tools/hd/preserve_body_runtime.py','tools/hd/preview_body.go','tools/hd/compare_maze_saved.py',
    'workplace/hd/maze-saved-state-independent-v1-20261003.go','apps/psychicwar/theme/body_context_test.go',
    'tools/docker/go-ebiten.Dockerfile','tools/pbl.py'])
snapshot_root=work/'source-snapshot';assert not snapshot_root.exists();snapshot_root.mkdir()
records=[]
for source in sorted(sources):
    digest=sha(source);old=known.get(str(source))
    if old and old['sha256']==digest:
        snapshot=root/old['snapshot'];assert sha(snapshot)==digest;reused=True
    else:
        relative=Path('repo')/source.relative_to(root) if source.is_relative_to(root) else Path('gomod')/source.relative_to('/gomod')
        snapshot=snapshot_root/relative;snapshot.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(source,snapshot)
        assert sha(snapshot)==digest;reused=False
    assert snapshot.stat().st_uid==snapshot.stat().st_gid==1000
    records.append({'source':str(source),'snapshot':str(snapshot.relative_to(root)),'sha256':digest,'bytes':source.stat().st_size,'reused':reused})
doc={'scope':'actual production Theme runtime binary dependencies, bounded art acceptance and 17 normal/reload full-plane receipts; all local',
     'actual_nonstandard_dependency_files':dependency_count,'files':records,'base_manifests_sha256':bases,
     'originals_sha256':originals,'reused_files':sum(r['reused'] for r in records),'new_files':sum(not r['reused'] for r in records),
     'total_bytes':sum(r['bytes'] for r in records),
     'versions':{'go':subprocess.check_output(['go','version'],text=True).strip(),
                 'python':subprocess.check_output(['python3','--version'],text=True).strip(),
                 'binary':subprocess.check_output(['go','version','-m',str(work/'verify-body-runtime')],text=True).strip()}}
p=work/'source-manifest.json';assert not p.exists();p.write_text(json.dumps(doc,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({k:doc[k] for k in ['actual_nonstandard_dependency_files','reused_files','new_files','total_bytes']}))
