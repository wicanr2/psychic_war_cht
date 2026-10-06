"""研究038 §107：保存角色來源觀察的實際依賴與精確收據。"""
from pathlib import Path
import hashlib
import json
import shutil
import subprocess
import sys

root=Path('/src')
work=Path(sys.argv[1])
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
assert work.stat().st_uid==work.stat().st_gid==1000
base_path=root/'workplace/hd/gui-ally-items-v1-20261004/source-manifest.json'
base=json.loads(base_path.read_text())
known={r['source']:r for r in base['files']}
text=(work/'build-deps.json').read_text()
decoder=json.JSONDecoder()
sources=set()
while text.strip():
    package,end=decoder.raw_decode(text.lstrip())
    text=text.lstrip()[end:]
    if package.get('Standard'):
        continue
    folder=Path(package['Dir'])
    for group in ['GoFiles','CgoFiles','CFiles','CXXFiles','HFiles','SFiles','EmbedFiles']:
        sources.update(folder/name for name in package.get(group,[]))
    if package.get('Module',{}).get('GoMod'):
        sources.add(Path(package['Module']['GoMod']))
dependency_count=len(sources)
sources.update(root/p for p in ['go.mod','go.sum','tools/pbl.py',
    'workplace/hd/dat-runtime-v1-20261002.go.work','tools/hd/observe_body_context.go',
    'tools/hd/verify_body_context.py','tools/hd/preserve_body_context.py',
    'tools/hd/compare_maze_saved.py','worktrees/dosgolem/internal/machine/state.go',
    'workplace/hd/maze-saved-state-independent-v1-20261003.go',
    'workplace/hd/maze-saved-state-independent-v1-20261003.bin'])
sources.update(p for p in work.iterdir() if p.is_file())
previous=root/'workplace/hd/body-context-v1-20261004'
sources.update(previous/name for name in ['observer-source.go','verifier-source-failed.py',
    'independent-failed.json','trace.json','machine.json','observe-body-context','build-deps.json'])
snapshot_root=work/'source-snapshot'
assert not snapshot_root.exists()
snapshot_root.mkdir()
records=[]
for source in sorted(sources):
    digest=sha(source)
    old=known.get(str(source))
    if old and old['sha256']==digest:
        snapshot=root/old['snapshot']
        assert sha(snapshot)==digest
        reused=True
    else:
        relative=Path('repo')/source.relative_to(root) if source.is_relative_to(root) else Path('gomod')/source.relative_to('/gomod')
        snapshot=snapshot_root/relative
        snapshot.parent.mkdir(parents=True,exist_ok=True)
        shutil.copyfile(source,snapshot)
        assert sha(snapshot)==digest
        reused=False
    assert snapshot.stat().st_uid==snapshot.stat().st_gid==1000
    records.append({'source':str(source),'snapshot':str(snapshot.relative_to(root)),
                    'sha256':digest,'bytes':source.stat().st_size,'reused':reused})
trace=json.loads((work/'trace.json').read_text())
originals={p:h for p,h in trace['inputs_sha256'].items() if p.startswith('/orig/') or p.endswith('.state')}
for path,digest in originals.items():
    assert sha(path)==digest
result={'scope':'original-only source observer actual nonstandard dependencies, exact verifier versions, receipts and reused complete-state comparator; no production frontend build',
    'actual_nonstandard_dependency_files':dependency_count,'files':records,
    'base_manifest':str(base_path),'base_manifest_sha256':sha(base_path),
    'originals_sha256':originals,'reused_files':sum(r['reused'] for r in records),
    'new_files':sum(not r['reused'] for r in records),'total_bytes':sum(r['bytes'] for r in records),
    'versions':{'go':subprocess.check_output(['go','version'],text=True).strip(),
                'python':subprocess.check_output(['python3','--version'],text=True).strip(),
                'observer':subprocess.check_output(['go','version','-m',str(work/'observe-body-context')],text=True).strip(),
                'comparator':subprocess.check_output(['go','version','-m',str(root/'workplace/hd/maze-saved-state-independent-v1-20261003.bin')],text=True).strip()}}
out=work/'source-manifest.json'
assert not out.exists()
out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({k:result[k] for k in ['actual_nonstandard_dependency_files','reused_files','new_files','total_bytes']}))
