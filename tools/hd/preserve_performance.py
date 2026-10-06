"""研究038 §105：保存計時副本的實際依賴，重用已核對的精確來源。"""
from pathlib import Path
import hashlib
import json
import shutil
import subprocess
import sys

root = Path('/src')
work = Path(sys.argv[1])
build = work.parent
base_path = root/'workplace/hd/gui-ally-items-v1-20261004/source-manifest.json'
base = json.loads(base_path.read_text())
known = {row['source']: row for row in base['files']}
sha = lambda p: hashlib.sha256(Path(p).read_bytes()).hexdigest()
assert sha(root/'cmd/psychicwar/main.go') == sha(build/'main-base.go')
text = (build/'build-deps.json').read_text()
decoder = json.JSONDecoder()
packages = []
while text.strip():
    obj, end = decoder.raw_decode(text.lstrip())
    packages.append(obj)
    text = text.lstrip()[end:]
sources = set()
for package in packages:
    if package.get('Standard'):
        continue
    folder = Path(package['Dir'])
    for group in ['GoFiles', 'CgoFiles', 'CFiles', 'CXXFiles', 'HFiles', 'SFiles', 'EmbedFiles']:
        sources.update(folder/name for name in package.get(group, []))
    if package.get('Module', {}).get('GoMod'):
        sources.add(Path(package['Module']['GoMod']))
dependency_count = len(sources)
sources.update(root/p for p in ['go.mod', 'go.sum', 'workplace/hd/dat-runtime-v1-20261002.go.work'])
for folder in ['font', 'text', 'workplace/hd/theme-ally-items-v1-20261004']:
    sources.update(p for p in (root/folder).rglob('*') if p.is_file())
sources.update(root/'tools/hd'/f'{name}.py' for name in [
    'prepare_performance', 'run_performance', 'verify_performance', 'preserve_performance'])
snapshots = work/'source-snapshot'
assert not snapshots.exists()
snapshots.mkdir()
records = []
for source in sorted(sources):
    actual = build/'main-measured.go' if source == root/'cmd/psychicwar/main.go' else source
    digest = sha(actual)
    old = known.get(str(source))
    if actual == source and old and old['sha256'] == digest:
        snapshot = root/old['snapshot']
        assert sha(snapshot) == digest
        reused = True
    else:
        relative = Path('repo')/actual.relative_to(root) if actual.is_relative_to(root) else Path('gomod')/actual.relative_to('/gomod')
        snapshot = snapshots/relative
        snapshot.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(actual, snapshot)
        assert sha(snapshot) == digest
        reused = False
    assert snapshot.stat().st_uid == 1000 and snapshot.stat().st_gid == 1000
    records.append({'source': str(source), 'actual_source': str(actual),
                    'snapshot': str(snapshot), 'sha256': digest,
                    'bytes': actual.stat().st_size, 'reused': reused})
originals = {}
for name, digest in base['originals_sha256'].items():
    assert sha(name) == digest
    originals[name] = digest
versions = {label: subprocess.check_output(command, text=True).strip() for label, command in {
    'go': ['go', 'version'], 'python': ['python3', '--version'],
    'binary': ['go', 'version', '-m', str(build/'psychicwar-measured.bin')],
}.items()}
result = {'scope': 'current27 local measured frontend; actual nonstandard build sources including overlay',
          'versions': versions, 'base_manifest': str(base_path), 'base_manifest_sha256': sha(base_path),
          'actual_nonstandard_dependency_files': dependency_count, 'files': records,
          'originals_sha256': originals, 'binary_sha256': sha(build/'psychicwar-measured.bin'),
          'reused_files': sum(r['reused'] for r in records),
          'new_files': sum(not r['reused'] for r in records)}
(work/'source-manifest.json').write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
print(json.dumps({key: result[key] for key in ['actual_nonstandard_dependency_files', 'reused_files', 'new_files']}))
