"""另存25筆主題，只選入已審查的歐格斯首姿勢。入口研究038 §80。"""
from pathlib import Path
import argparse
import hashlib
import json
import shutil


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base', type=Path, required=True)
    parser.add_argument('--review', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True)
    args = parser.parse_args()
    base, review, out = args.base.resolve(), args.review.resolve(), args.out.resolve()
    if not base.is_dir() or not review.is_dir() or out.exists() or not out.parent.is_dir():
        parser.error('來源需存在，輸出需為新目錄且父目錄已存在')
    assert (out.parent.stat().st_uid, out.parent.stat().st_gid) == (1000, 1000)
    sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
    accepted = json.loads((review/'art-review.json').read_text())
    assert accepted['accepted_images'] == [6]
    assert accepted['status'] == 'accepted single-pose art and limited normal-source presentation'
    verified = json.loads((review/'independent.json').read_text())
    assert len(verified['transitions']) == 8 and len(verified['samples']) == 17
    assert all(row['full_actor_rgba_mismatch'] == row['actor_composition_mismatch'] ==
               row['full_text_mismatch'] == 0 for row in verified['samples'])
    visible = [row for row in verified['samples'] if row['enemy_cells']]
    assert len(visible) == 1 and visible[0]['source_gate_target'] == 6
    reload = json.loads((review/'reload-verification-v2.json').read_text())
    assert len(reload['rows']) == 4
    assert all(row['machine_fields'] == 44 and row['dos_gob_equal'] and
               row['full_projection_mismatch'] == 0 for row in reload['rows'])
    manifest = json.loads((base/'manifest.json').read_text())
    assert len(manifest['entries']) == 25
    entries = manifest['entries']
    selected = [e for e in entries if e['pbl'] == 'ENEMY01.PBL' and e['image'] == 6]
    assert len(selected) == 1 and selected[0]['at'] == [32, 152]
    name = selected[0]['png']
    assert name == 'ENEMY01-06.png'
    staged = review/'selected-theme'
    assert sha(staged/name) == accepted['accepted_sha256'][name]
    names = ['manifest.json'] + [e['png'] for e in entries]
    assert len(set(names)) == 26
    assert all(Path(n).name == n and (staged/n).is_file() for n in names)
    changed = [n for n in names if sha(base/n) != sha(staged/n)]
    assert changed == [name], changed
    out.mkdir()
    for n in names:
        shutil.copyfile(staged/n, out/n)
    assert all((out/n).read_bytes() == (staged/n).read_bytes() for n in names)
    sources = [Path(__file__), review/'art-review.json', review/'independent.json',
               review/'reload-verification-v2.json']
    sources += [base/n for n in names] + [staged/n for n in names]
    result = {'scope': 'local25 theme; reviewed Oogus pose6 only; poses7/8 and overlapping HD actions incomplete',
              'base': str(base), 'entries': 25, 'changed_assets': changed,
              'manifest_sha256': sha(out/'manifest.json'),
              'inputs': {str(p): sha(p) for p in sources},
              'outputs': {n: sha(out/n) for n in names}}
    (out/'selection.json').write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print('另存25筆主題；僅換歐格斯首姿勢，其他24張及manifest相同。')


if __name__ == '__main__':
    main()
