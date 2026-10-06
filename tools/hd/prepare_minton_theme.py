"""另存25筆主題，更新已審查的ENEMY00 #6–#8。入口研究038 §79。"""
from pathlib import Path
import argparse
import hashlib
import json
import shutil
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import pbl


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
    accepted = json.loads((review/'art-review.json').read_text())
    assert accepted['accepted_images'] == [6, 7, 8]
    assert accepted['status'] == 'accepted art and limited normal-source presentation'
    verified = json.loads((review/'independent.json').read_text())
    assert len(verified['transitions']) == 16 and len(verified['samples']) == 25
    assert all(row.get('full_actor_rgba_mismatch', 0) == 0 for row in verified['samples'])
    assert json.loads((review/'independent-end.json').read_text())['actor_rgba_mismatch'] == 0
    assert all(row['machine_fields'] == 44 and row['full_projection_mismatch'] == 0
               for row in json.loads((review/'reload-verification-v2.json').read_text())['rows'])
    manifest_path = base/'manifest.json'
    manifest = json.loads(manifest_path.read_text())
    entries = manifest['entries']
    assert len(entries) == 25
    selected = [e for e in entries if e['pbl']=='ENEMY00.PBL' and e['image'] in [6, 7, 8]]
    assert [e['image'] for e in selected] == [6, 7, 8]
    names = ['manifest.json'] + [e['png'] for e in entries]
    assert len(set(names)) == 26
    sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
    for e in selected:
        assert e['at'] == [32, 152] and e['png'] == f"ENEMY00-{e['image']:02d}.png"
        asset = review/'theme'/e['png']
        w, h, ch, _, _ = pbl.read_png(asset)
        assert (w, h, ch) == (72, 96, 3)
        assert sha(asset) == accepted['accepted_sha256'][e['png']]
    for name in names:
        assert Path(name).name == name and (base/name).is_file()
    out.mkdir()
    for name in names:
        shutil.copyfile(base/name, out/name)
    for e in selected:
        shutil.copyfile(review/'theme'/e['png'], out/e['png'])
    changed = [e['png'] for e in entries if sha(base/e['png']) != sha(out/e['png'])]
    assert changed == ['ENEMY00-06.png', 'ENEMY00-07.png', 'ENEMY00-08.png']
    assert (out/'manifest.json').read_bytes() == manifest_path.read_bytes()
    assert all((out/name).read_bytes() == (review/'theme'/name).read_bytes() for name in names)
    input_paths = [Path(__file__), manifest_path] + [base/e['png'] for e in entries]
    input_paths += [review/'theme'/e['png'] for e in selected]
    input_paths += [review/name for name in ['art-review.json','independent.json','independent-end.json','reload-verification-v2.json']]
    result = {'scope':'local25 theme, three reviewed Minton HD poses; not full HD or GUI/DAT acceptance',
              'entries':25, 'base':str(base), 'changed_assets':changed,
              'manifest_sha256':sha(manifest_path),
              'inputs':{str(p):sha(p) for p in input_paths},
              'outputs':{name:sha(out/name) for name in names}}
    (out/'selection.json').write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print('另存25筆主題；只換敏頓三個姿勢，其他22張及manifest相同。')


if __name__ == '__main__':
    main()
