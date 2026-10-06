"""研究038 §74：GUI DAT批次的精確來源版本與收據保全。"""
import hashlib
import json
from pathlib import Path
import shutil

root=Path('/src'); hd=root/'workplace/hd'
target=hd/'source-gui-dat-v2-20261003'
output=hd/'gui-dat-source-manifest-v2-20261003.json'
assert not target.exists() and not output.exists()
assert hd.stat().st_uid==1000
files=set(); originals={}; resolved={}
receipts=[hd/'gui-dat-independent-v2-20261003.json']
for name in ('gui-dat-v1-20261003','gui-dat-load-v1-20261003'):
    folder=hd/name
    files.update(p for p in folder.rglob('*') if p.is_file())
    receipts += [folder/'execution.json',folder/'original-frames.json']
for receipt in receipts:
    value=json.loads(receipt.read_text()); files.add(receipt)
    for name, expected in value['inputs_sha256'].items():
        source=Path(name)
        if not source.is_absolute(): source=root/source
        actual=hashlib.sha256(source.read_bytes()).hexdigest()
        if actual!=expected:
            assert str(source)==str(root/'tools/hd/gui_dat_run.py'), name
            source=hd/'gui-dat-run-source-v1-20261003.py'
            assert hashlib.sha256(source.read_bytes()).hexdigest()==expected,name
        resolved[name+'@'+expected]=str(source)
        if source.is_relative_to(Path('/orig')): originals[name]=expected
        else:
            assert source.is_relative_to(root), str(source)
            files.add(source)
for folder in ('cmd/psychicwar','apps/psychicwar','worktrees/dosgolem/oracle','worktrees/dosgolem/xlate','worktrees/dosgolem/pbl','worktrees/dosgolem/internal'):
    files.update(p for p in (root/folder).rglob('*.go') if p.is_file())
for name in ('go.mod','go.sum','worktrees/dosgolem/go.mod','docs/re/038-hd-theme-feasibility.md','docs/spec/024-hd-theme.md','CONTEXT.md','WORKLOG.md','docs/worklist.json','tools/hd/gui_dat_run.py','tools/hd/export_gui_dat.go','tools/hd/verify_gui_dat.py','tools/hd/preserve_gui_dat.py'):
    files.add(root/name)
files.add(hd/'dat-runtime-v1-20261002.go.work')
files.add(hd/'gui-dat-run-source-v1-20261003.py')
files.add(hd/'gui-dat-verifier-source-v1-20261003.py')
files.add(hd/'gui-dat-verifier-stop-v1-20261003.json')
files.add(hd/'gui-dat-preserver-source-v1-20261003.py')
optional_missing=[]
for name in ('worktrees/dosgolem/go.sum','workplace/hd/dat-runtime-v1-20261002.go.work.sum'):
    p=root/name
    if p.is_file(): files.add(p)
    else: optional_missing.append(name)
for name in ('gui25-xvfb-v1-20261003.log','gui25-load-xvfb-v1-20261003.log'):
    files.add(hd/name)
target.mkdir(); rows=[]
for source in sorted(files):
    st=source.stat(); assert st.st_uid==1000 and st.st_gid==1000,str(source)
    relative=source.relative_to(root); dest=target/relative
    dest.parent.mkdir(parents=True,exist_ok=True); shutil.copyfile(source,dest)
    sha=hashlib.sha256(source.read_bytes()).hexdigest()
    assert hashlib.sha256(dest.read_bytes()).hexdigest()==sha
    assert dest.stat().st_uid==1000 and dest.stat().st_gid==1000
    rows.append({'source':str(relative),'snapshot':str(dest.relative_to(root)),'sha256':sha,'bytes':st.st_size})
value={'scope':'GUI DAT兩版工具、實際視窗、收據與建置來源；原版僅記雜湊','originals_sha256':originals,'resolved_versions':resolved,'optional_missing':optional_missing,
       'file_count':len(rows),'total_bytes':sum(x['bytes'] for x in rows),'files':rows}
output.write_text(json.dumps(value,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({'file_count':value['file_count'],'total_bytes':value['total_bytes']},ensure_ascii=False))
