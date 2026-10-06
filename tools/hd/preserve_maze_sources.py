"""研究038 §71：保全MAZE來源、正常探索、失敗版本及精確工具鏈來源。"""
import hashlib
import json
from pathlib import Path
import shutil


ROOT=Path('/src')
OUT=ROOT/'workplace/hd'


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    target=OUT/'source-maze-v1-20261003'
    manifest=OUT/'maze-source-manifest-v1-20261003.json'
    assert not target.exists() and not manifest.exists(), '拒絕覆寫'
    assert OUT.stat().st_uid==OUT.stat().st_gid==1000
    prefixes=['companion-explore-','room-nearby-explore-','maze-tile-sources-',
              'maze-read-source-','maze-saved-state-independent-',
              'verify-maze-tiles-source-','observe-maze-reads-source-']
    sources={p for p in OUT.iterdir() if p.is_file() and any(p.name.startswith(prefix) for prefix in prefixes)}
    original_hashes={}
    receipts=[OUT/'companion-explore-v2-20261003.json',
              OUT/'room-nearby-explore-v1-20261003.json',
              OUT/'maze-tile-sources-v2-20261003.json',
              OUT/'maze-read-source-v1-20261003.json']
    for path in receipts:
        doc=json.loads(path.read_text())
        for name,digest in doc['inputs_sha256'].items():
            if name.startswith('/orig/'):
                source=ROOT/'workplace/original'/name.removeprefix('/orig/')
                assert source.is_file() and sha(source)==digest, '原始檔已變：'+name
                original_hashes[name]=digest
            else:
                source=ROOT/name
                assert source.is_file() and sha(source)==digest, '已驗來源已變：'+name
                sources.add(source)
    for path in (ROOT/'worktrees/dosgolem/internal').rglob('*.go'):
        sources.add(path)
    for relative in ['CONTEXT.md','WORKLOG.md','docs/worklist.json','docs/spec/024-hd-theme.md',
                     'docs/re/038-hd-theme-feasibility.md','tools/hd/preserve_maze_sources.py']:
        sources.add(ROOT/relative)
    previous=OUT/'room22-source-manifest-v1-20261003.json'
    prior=json.loads(previous.read_text())
    prior_checked=0
    for name,row in prior['files'].items():
        snapshot=ROOT/row['snapshot']
        assert snapshot.is_file() and sha(snapshot)==row['sha256'], '前批快照SHA已變：'+name
        assert snapshot.stat().st_size==row['size']
        assert snapshot.stat().st_uid==row['uid']==1000 and snapshot.stat().st_gid==row['gid']==1000
        prior_checked+=1
    assert prior_checked==645
    sources.add(previous)
    target.mkdir()
    rows=[]
    for source in sorted(sources):
        assert source.is_relative_to(ROOT)
        assert source.stat().st_uid==source.stat().st_gid==1000
        relative=source.relative_to(ROOT)
        destination=target/relative
        destination.parent.mkdir(parents=True,exist_ok=True)
        shutil.copyfile(source,destination)
        assert sha(source)==sha(destination)
        assert destination.stat().st_uid==destination.stat().st_gid==1000
        rows.append({'source':str(relative),'snapshot':str(destination.relative_to(ROOT)),
                     'size':source.stat().st_size,'sha256':sha(source)})
    data={'scope':'MAZE.BIN來源、正常F2／BBS／受阻Radar探索、完整state核對及失敗版本保全',
          'snapshot_root':str(target.relative_to(ROOT)),
          'count':len(rows),'total_bytes':sum(row['size'] for row in rows),
          'original_hashes_only':original_hashes,'files':rows,
          'prior_manifest':str(previous.relative_to(ROOT)),
          'prior_snapshots_verified':prior_checked,
          'limits':'保存本批收尾補記前文件bytes。原版檔只記SHA；所有state、RAM、PNG、候選與快照留本機。正式HD主題仍25筆，MAZE尚未接入。'}
    with manifest.open('x') as f:
        json.dump(data,f,ensure_ascii=False,indent=2)
        f.write('\n')
    print(json.dumps({'count':data['count'],'total_bytes':data['total_bytes'],
                      'manifest':str(manifest.relative_to(ROOT))},ensure_ascii=False))


if __name__=='__main__':
    main()
