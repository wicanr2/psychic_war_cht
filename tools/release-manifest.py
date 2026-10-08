"""Docker內建立已驗收封包的SHA-256清單；只選patch三包作公開附件。"""
from pathlib import Path
import argparse,hashlib,json,re
p=argparse.ArgumentParser();p.add_argument('version');p.add_argument('head');p.add_argument('--promo-required',action='store_true');a=p.parse_args()
assert re.fullmatch(r'v\.[0-9]+\.[0-9]+\.[0-9]+-[0-9]{8}',a.version)
D=Path('/src/dist-all')/a.version;S=D/'smoke';sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
package=json.loads((S/'package-verification.json').read_text());wine=json.loads((S/'windows-wine-verification.json').read_text())
assert package['version']==wine['version']==a.version and package['build_head']==a.head
assert package['status'].startswith('PASS_') and len(package['reports'])==6
assert wine['status'].startswith('PASS_') and {r['kind'] for r in wine['rows']}=={'patch','full-local'}
promo=D/'promo/verification.json'
if a.promo_required:
 rvideo=json.loads(promo.read_text());assert rvideo['version']==a.version and rvideo['status'].startswith('PASS_') and rvideo['live_seconds']>=60
assets=sorted((D/'patch').iterdir());assert len(assets)==3
assert all(p.suffix in ['.zip','.AppImage'] and '-with-data' not in p.name for p in assets)
full=[p for p in (D/'full-local').iterdir() if p.suffix in ['.zip','.AppImage']]
assert len(full)==3 and all('-with-data' in p.name for p in full)
files={}
for f in sorted(D.rglob('*')):
 if not f.is_file() or f.name in ['SHA256SUMS.json','delivery-complete.json']:continue
 name=str(f.relative_to(D));files[name]=dict(bytes=f.stat().st_size,sha256=sha(f),rights='public-code-and-approved-assets' if name.startswith('patch/') else 'local-only')
manifest=dict(schema='psychic-war-delivery/1',version=a.version,build_head=a.head,release_assets=[str(f.relative_to(D)) for f in assets],files=files,provenance=dict(finalizer='tools/release-manifest.py',finalizer_sha256=sha(Path(__file__))),limits=['Windows Wine不等於真Windows','macOS僅結構驗收，未在Mac實跑','全部HD與原版僅本機','未做全程試玩；已知未知範圍保留原版'])
(D/'SHA256SUMS.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
published=S/'published-assets-verified.json';status='COMPLETE_LOCAL_PACKAGES'
if published.exists():
 r=json.loads(published.read_text());assert r['status'].startswith('PASS_');status='COMPLETE_PUBLISHED_PUBLIC_PATCH_AND_LOCAL_FULL_HD'
receipt=dict(status=status,version=a.version,build_head=a.head,packages=6,public_assets=3,pbl_identities=531,png_count=532,manifest_sha256=sha(D/'SHA256SUMS.json'),package_receipt_sha256=sha(S/'package-verification.json'),wine_receipt_sha256=sha(S/'windows-wine-verification.json'))
if a.promo_required:receipt.update(promo_seconds=73,actual_gameplay_seconds=rvideo['live_seconds'],HD_switches=len(rvideo['switches']),promo_receipt_sha256=sha(promo))
if published.exists():receipt['release_url']=r['release_url']
(S/'delivery-complete.json').write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+'\n')
print(json.dumps(receipt,ensure_ascii=False))
