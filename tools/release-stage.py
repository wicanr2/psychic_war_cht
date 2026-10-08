"""Docker內組裝兩種封包；公開採明確清單，原版及HD僅本機。"""
from pathlib import Path
import argparse,hashlib,json,os,plistlib,re,shutil,struct,zipfile
p=argparse.ArgumentParser();p.add_argument('version');p.add_argument('head');p.add_argument('theme');a=p.parse_args()
assert re.fullmatch(r'v\.[0-9]+\.[0-9]+\.[0-9]+-[0-9]{8}',a.version)
R=Path('/src');B=R/'workplace/release-stage'/a.version;D=R/'dist-all'/a.version;T=Path(a.theme);O=Path('/orig/psychic-war')
assert (T/'manifest.json').is_file() and (O/'PW.EXE').is_file() and B.stat().st_uid==os.getuid()
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
def files(root):return sorted(p for p in root.rglob('*') if p.is_file())
def zipfiles(paths,base,out):
 assert not out.exists()
 with zipfile.ZipFile(out,'x',zipfile.ZIP_DEFLATED,compresslevel=6) as z:
  for f in paths:z.write(f,f.relative_to(base))
def data(root,full):
 root.mkdir(parents=True,exist_ok=True)
 for directory,pattern in [('text','*.json'),('font','*.golemfnt')]:
  (root/directory).mkdir()
  for f in (R/directory).glob(pattern):shutil.copyfile(f,root/directory/f.name)
 for f in ['README.md','LICENSE']:shutil.copyfile(R/f,root/f)
 notices=root/'third-party-licenses';notices.mkdir()
 shutil.copyfile('/usr/local/go/LICENSE',notices/'Go-LICENSE')
 shutil.copyfile(R/'worktrees/dosgolem/LICENSE',notices/'dosgolem-LICENSE')
 for module,version in re.findall(r'^\s*([^\s]+)\s+(v[^\s]+)',(R/'go.mod').read_text(),re.M):
  directory=Path('/gomod')/(module+'@'+version)
  if directory.is_dir():
   for f in directory.glob('LICENSE*'):
    if f.is_file():shutil.copyfile(f,notices/(module.replace('/','_')+'-'+f.name))
 (root/'START.txt').write_text('銀河超能力戰記 '+a.version+'\n\n'+('本機完整版含原版及HD，禁止上傳。使用Start-HD啟動器或 -theme 指向theme/hd。\n' if full else '請自行合法持有DOS英文版。使用 -orig 指向含PW.EXE的資料目錄。\n')+'存檔：Linux使用者資料目錄；Windows %APPDATA%/PsychicWar；macOS ~/Library/Application Support/PsychicWar。\nShift+F5切換已選HD；F5中英；F4說明；F10/F11即時存讀。\nmacOS未簽章，僅結構驗證；Windows以Wine驗證，真機另待驗。\n',encoding='utf-8-sig')
 if full:shutil.copytree(O,root/'original');shutil.copytree(T,root/'theme/hd')
def manifest(root,platform,full):
 rows={str(f.relative_to(root)):{'size':f.stat().st_size,'sha256':sha(f)} for f in files(root)}
 if not full:
  assert not (root/'original').exists() and not (root/'theme').exists()
  names={f.name.casefold() for f in files(O)}
  assert not any(f.name.casefold() in names or f.suffix.casefold() in ['.pbl','.mid','.ibm','.wav','.state','.dat'] for f in files(root))
 (root/'PACKAGE-MANIFEST.json').write_text(json.dumps(dict(schema='psychic-war-package/1',version=a.version,build_head=a.head,platform=platform,architecture='arm64+amd64' if platform=='macos' else 'amd64',rights='local-only-original-and-all-HD' if full else 'public-code-and-approved-assets',files=rows),ensure_ascii=False,indent=2)+'\n')
for full in [False,True]:
 kind='full-local' if full else 'patch';dest=D/kind;dest.mkdir(parents=True,exist_ok=True);suffix='-with-data' if full else '';stage=B/kind;stage.mkdir()
 app=stage/'PsychicWar.AppDir';data(app/'usr/bin',full);shutil.copyfile(B/'linux',app/'usr/bin/psychicwar');(app/'usr/bin/psychicwar').chmod(0o755)
 theme=' -theme "$HERE/usr/bin/theme/hd"' if full else ''
 (app/'AppRun').write_text('#!/bin/sh\nHERE="$(dirname "$(readlink -f "$0")")"\nexec "$HERE/usr/bin/psychicwar"'+theme+' "$@"\n');(app/'AppRun').chmod(0o755)
 (app/'psychicwar.desktop').write_text('[Desktop Entry]\nType=Application\nName=Psychic War\nExec=psychicwar\nIcon=psychicwar\nCategories=Game;\nTerminal=false\n');shutil.copyfile(B/'icon.png',app/'psychicwar.png');manifest(app,'linux',full)
 win=stage/'PsychicWar';data(win,full);shutil.copyfile(B/'windows.exe',win/'PsychicWar.exe')
 if full:(win/'Start-HD.bat').write_bytes(b'@echo off\r\ncd /d "%~dp0"\r\nstart "" "%~dp0PsychicWar.exe" -theme "%~dp0theme\\hd"\r\n')
 manifest(win,'windows',full);zipfiles(files(win),stage,dest/('PsychicWar-'+a.version+suffix+'-win64.zip'))
 mac=stage/'PsychicWar.app';res=mac/'Contents/Resources';exe=mac/'Contents/MacOS';exe.mkdir(parents=True);data(res,full);shutil.copyfile(B/'macos',exe/'psychicwar');(exe/'psychicwar').chmod(0o755)
 info=dict(CFBundleDevelopmentRegion='zh_TW',CFBundleDisplayName='銀河超能力戰記',CFBundleExecutable='psychicwar',CFBundleIdentifier='io.github.wicanr2.psychicwar',CFBundleName='PsychicWar',CFBundlePackageType='APPL',CFBundleShortVersionString=a.version[2:].split('-')[0],CFBundleVersion=a.version,LSMinimumSystemVersion='11.0',NSHighResolutionCapable=True)
 icon=(B/'icon.png').read_bytes();iw,ih=struct.unpack('>II',icon[16:24]);assert iw==ih
 tag={64:b'icp6',128:b'ic07',256:b'ic08',512:b'ic09',1024:b'ic10'}[iw];body=tag+struct.pack('>I',len(icon)+8)+icon
 (res/'psychicwar.icns').write_bytes(b'icns'+struct.pack('>I',len(body)+8)+body);info['CFBundleIconFile']='psychicwar'
 if full:
  (exe/'psychicwar').rename(exe/'psychicwar.bin')
  (exe/'psychicwar').write_text('#!/bin/sh\nHERE="$(cd "$(dirname "$0")" && pwd)"\nexec "$HERE/psychicwar.bin" -theme "$HERE/../Resources/theme/hd" "$@"\n');(exe/'psychicwar').chmod(0o755)
 (mac/'Contents/Info.plist').write_bytes(plistlib.dumps(info));manifest(mac,'macos',full);paths=files(mac)
 if full:
  script=stage/'Start-HD.command';script.write_text('#!/bin/sh\nHERE="$(cd "$(dirname "$0")" && pwd)"\nexec "$HERE/PsychicWar.app/Contents/MacOS/psychicwar" "$@"\n');script.chmod(0o755);paths.append(script)
 zipfiles(paths,stage,dest/('PsychicWar-'+a.version+suffix+'-macos.zip'))
print('兩類三平台staging完成；AppImage由既有runtime組裝。')
