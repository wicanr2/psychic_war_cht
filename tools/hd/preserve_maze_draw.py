"""研究038 §72：保存本批MAZE繪製證據及精確工具／依賴；原版只記SHA。"""
import hashlib
import json
from pathlib import Path
import shutil


ROOT = Path('/src')
OUT = ROOT/'workplace/hd'


def sha(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream,'sha256').hexdigest()


def main():
    dest = OUT/'source-maze-draw-v1-20261003'
    manifest = OUT/'maze-draw-source-manifest-v1-20261003.json'
    assert not dest.exists() and not manifest.exists(), '拒絕覆寫'
    assert OUT.stat().st_uid == OUT.stat().st_gid == 1000
    selected = set()
    for pattern in ['maze-draw-build-v1-20261003*','maze-draw-build-v2-20261003*',
                    'maze-draw-source-v1-20261003*','maze-draw-source-v2-20261003*',
                    'maze-draw-verification-v1-20261003.json',
                    'maze-draw-saved-state-independent-v1-20261003.json',
                    'maze-table-verification-v1-20261003.json']:
        selected.update(OUT.glob(pattern))
    selected.update((ROOT/'workplace/ida/hd-maze-20261003').glob('*.json'))
    selected.update((ROOT/'workplace/ida/hd-maze-20261003').glob('*.log'))
    selected.update((ROOT/'workplace/ida/hd-maze-20261003').glob('*.py'))
    originals = {}
    for receipt in list(selected):
        if receipt.suffix != '.json':
            continue
        value = json.loads(receipt.read_text())
        for name, expected in value.get('inputs_sha256',{}).items():
            path = Path(name) if name.startswith('/') else ROOT/name
            assert path.is_file() and sha(path) == expected, '來源已變：'+name
            if name.startswith('/orig/'):
                originals[name] = expected
            else:
                assert path.is_relative_to(ROOT), name
                selected.add(path)
    for name in ['tools/ida/maze.py','tools/hd/observe_maze_draw.go',
                 'tools/hd/verify_maze_draw.py','tools/hd/verify_maze_table.py',
                 'tools/hd/preserve_maze_draw.py','CONTEXT.md','WORKLOG.md','AGENTS.md',
                 'docs/re/038-hd-theme-feasibility.md','docs/spec/024-hd-theme.md',
                 'docs/worklist.json','go.mod','go.sum',
                 'workplace/ida/PW_UNP.EXE.i64',
                 'workplace/hd/maze-saved-state-independent-v1-20261003.bin',
                 'workplace/hd/maze-saved-state-independent-v1-20261003.go',
                 'workplace/hd/maze-saved-state-independent-v1-20261003.json',
                 'workplace/hd/room-nearby-explore-v1-20261003-bbs.state',
                 'workplace/hd/maze-source-manifest-v1-20261003.json']:
        selected.add(ROOT/name)
    for name in ['PW_UNP.EXE']:
        path=ROOT/'workplace/ida'/name
        originals[str(path)] = sha(path)
    prior_path=OUT/'maze-source-manifest-v1-20261003.json'
    prior=json.loads(prior_path.read_text())
    for item in prior['files']:
        p=ROOT/item['snapshot']
        assert p.is_file() and p.stat().st_size==item['size'] and sha(p)==item['sha256']
        assert p.stat().st_uid==p.stat().st_gid==1000
    for p in selected:
        assert p.is_file() and p.stat().st_uid==p.stat().st_gid==1000,str(p)
        assert p.is_relative_to(ROOT)
    dest.mkdir()
    files=[]
    for p in sorted(selected):
        relative=p.relative_to(ROOT); target=dest/relative; target.parent.mkdir(parents=True,exist_ok=True)
        shutil.copyfile(p,target)
        item={'source':str(relative),'snapshot':str(target.relative_to(ROOT)),
              'size':p.stat().st_size,'sha256':sha(p)}
        assert sha(target)==item['sha256'] and target.stat().st_uid==target.stat().st_gid==1000
        files.append(item)
    value={'scope':'MAZE原版648圖塊、2592列中途、slot表及IDA精確來源；不是正式HD',
           'count':len(files),'total_bytes':sum(x['size'] for x in files),
           'original_hashes_only':originals,'files':files,'prior_manifest':str(prior_path.relative_to(ROOT)),
           'prior_snapshots_verified':len(prior['files']),
           'limits':'本機專用。文件保存收尾補記前bytes；未將原版輸入或資料庫上傳。'}
    with manifest.open('x') as stream:
        json.dump(value,stream,ensure_ascii=False,indent=2);stream.write('\n')
    print(value['count'],value['total_bytes'],'來源保存，前批',value['prior_snapshots_verified'],'份未變')


if __name__ == '__main__':
    main()
