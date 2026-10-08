"""在Docker內驗實際封包：內容SHA、權利、PE/Mach-O、Linux GUI、存檔與缺字型。"""
from pathlib import Path
import argparse,hashlib,json,os,plistlib,shutil,struct,subprocess,tempfile,time,zipfile
p=argparse.ArgumentParser();p.add_argument('version');p.add_argument('head');a=p.parse_args()
R=Path('/src');D=R/'dist-all'/a.version;S=D/'smoke';S.mkdir(exist_ok=True);reports=[]
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
def check_manifest(root):
 paths=list(root.rglob('PACKAGE-MANIFEST.json'));assert len(paths)==1
 mf=paths[0];base=mf.parent;m=json.loads(mf.read_text());assert m['version']==a.version and m['build_head']==a.head
 actual={str(f.relative_to(base)) for f in base.rglob('*') if f.is_file()};assert actual==set(m['files'])|{'PACKAGE-MANIFEST.json'}
 for name,row in m['files'].items():f=base/name;assert f.stat().st_size==row['size'] and sha(f)==row['sha256']
 if m['rights'].startswith('public'):
  assert not any(f.suffix.lower() in ['.pbl','.bin','.mid','.ibm','.wav','.dat','.state'] for f in base.rglob('*') if f.is_file())
  assert not list(base.rglob('PW.EXE')) and not list(base.rglob('manifest.json'))
 else:
  orig=next(base.rglob('original'));theme=next(base.rglob('hd'));assert sha(orig/'PW.EXE')==sha(Path('/orig/psychic-war/PW.EXE'))
  tm=json.loads((theme/'manifest.json').read_text());assert len(tm['entries'])==6875 and len(list(theme.glob('*.png')))==532
 assert list(base.rglob('LICENSE'));return m
def pe(raw):
 assert raw[:2]==b'MZ';off=struct.unpack_from('<I',raw,0x3c)[0];assert raw[off:off+4]==b'PE\0\0'
 assert struct.unpack_from('<H',raw,off+4)[0]==0x8664;opt=off+24;assert struct.unpack_from('<H',raw,opt)[0]==0x20b and struct.unpack_from('<H',raw,opt+68)[0]==2
 assert b'PsychicWar' in raw and a.version.encode() in raw;return 'PE64 amd64 GUI'
def macho(raw):
 magic,n=struct.unpack_from('>II',raw);assert magic==0xcafebabe and n==2;rows=[]
 for i in range(n):
  cpu,_,off,size,_=struct.unpack_from('>IIIII',raw,8+i*20);thin=raw[off:off+size];assert thin[:4]==b'\xcf\xfa\xed\xfe';nc=struct.unpack_from('<I',thin,16)[0];at=32;sig=False;minos=None;libs=[]
  for j in range(nc):
   cmd,length=struct.unpack_from('<II',thin,at);assert length>=8 and at+length<=len(thin)
   if cmd==0x1d:sig=True
   if cmd==0x32:minos=struct.unpack_from('<I',thin,at+12)[0]
   if cmd in [0xc,0x80000018,0x8000001f,0x80000023]:
    name=struct.unpack_from('<I',thin,at+8)[0];lib=thin[at+name:at+length].split(b'\0')[0].decode();assert lib.startswith(('/usr/lib/','/System/Library/'));libs.append(lib)
   at+=length
  assert cpu in [0x1000007,0x100000c] and minos==11<<16
  if cpu==0x100000c:assert sig
  rows.append(dict(cpu=cpu,signature=sig,minos='11.0',libraries=libs))
 assert {r['cpu'] for r in rows}=={0x1000007,0x100000c};assert all(s in raw for s in [b'Application Support',b'cjk24.golemfnt',a.version.encode()]);return rows
for kind in ['patch','full-local']:
 for file in sorted((D/kind).glob('*.zip')):
  with tempfile.TemporaryDirectory() as td:
   root=Path(td)
   with zipfile.ZipFile(file) as z:
    assert z.testzip() is None
    for f in z.infolist():assert not Path(f.filename).is_absolute() and '..' not in Path(f.filename).parts
    z.extractall(root)
   m=check_manifest(root)
   if m['platform']=='windows':details=pe((root/'PsychicWar/PsychicWar.exe').read_bytes())
   else:
    app=root/'PsychicWar.app';info=plistlib.loads((app/'Contents/Info.plist').read_bytes());assert info['CFBundleVersion']==a.version;exe=app/'Contents/MacOS/psychicwar';raw=exe.read_bytes()
    if raw.startswith(b'#!'):exe=Path(str(exe)+'.bin');raw=exe.read_bytes()
    details=macho(raw)
    with zipfile.ZipFile(file) as z:assert next(x for x in z.infolist() if x.filename.endswith(str(exe.relative_to(root)))).external_attr>>16&0o111
   reports.append(dict(file=str(file.relative_to(D)),sha256=sha(file),content='PASS',structure=details,rights=m['rights']))
for kind in ['patch','full-local']:
 file=next((D/kind).glob('*.AppImage'))
 with tempfile.TemporaryDirectory() as td:
  root=Path(td);subprocess.run([str(file),'--appimage-extract'],cwd=root,check=True,stdout=subprocess.DEVNULL,timeout=40);app=root/'squashfs-root';m=check_manifest(app);exe=app/'usr/bin/psychicwar';assert exe.read_bytes()[:4]==b'\x7fELF'
  version=subprocess.check_output([str(app/'AppRun'),'-version'],cwd='/tmp',text=True,timeout=20).strip();assert version==a.version
  name='linux-'+kind;out=S/name;assert not out.exists();out.mkdir();xdg=out/'xdg';env=dict(os.environ,XDG_DATA_HOME=str(xdg));state=R/'workplace/ida/hd-ally-recruit-20261004/hd43-ally2-dat-save-v4-20261006/a-options-hd-chinese.state'
  argv=[str(app/'AppRun'),'-audio','null','-load-state',str(state),'-quit-after','24s']
  if kind=='patch':argv+=['-orig','/orig/psychic-war']
  log=(out/'frontend.log').open('x');proc=subprocess.Popen(argv,cwd='/tmp',env=env,stdout=log,stderr=subprocess.STDOUT);start=time.monotonic()
  try:
   window=''
   while proc.poll() is None and time.monotonic()-start<15:
    q=subprocess.run(['xdotool','search','--name','Psychic War'],capture_output=True,text=True)
    if q.returncode==0:window=q.stdout.splitlines()[0];break
    time.sleep(.2)
   assert window;subprocess.run(['xdotool','windowfocus',window],check=True)
   def key(k):
    subprocess.run(['xdotool','keydown',k],check=True);time.sleep(.18);subprocess.run(['xdotool','keyup',k],check=True);time.sleep(1)
   time.sleep(1)
   for k in ['Return','Return','r','e','l','Return']:key(k)
   key('F10');subprocess.run(['import','-window',window,'-depth','8','PNG24:'+str(out/'window.png')],check=True)
   proc.wait(timeout=30);assert proc.returncode==0
  finally:
   if proc.poll() is None:proc.terminate();proc.wait(timeout=5)
   log.close()
  save=xdg/'psychicwar';assert (save/'rel.dat').stat().st_size==512 and (save/'quick.state').stat().st_size>0
  assert not (save/'psychicwar-error.log').exists();check_manifest(app)
  # 真實執行檔缺字型應失敗並留下可讀紀錄，不能只有編譯成功。
  font=app/'usr/bin/font/cjk24.golemfnt';font.rename(font.with_suffix('.hidden'))
  bad=subprocess.run(argv,cwd='/tmp',env=env,capture_output=True,text=True,timeout=15);assert bad.returncode!=0
  error=save/'psychicwar-error.log';assert error.is_file() and error.stat().st_size>0
  (out/'missing-font.log').write_text(bad.stdout+bad.stderr)
  reports.append(dict(file=str(file.relative_to(D)),sha256=sha(file),version=version,content='PASS',GUI_exit=0,DAT_bytes=512,F10_save=str((save/'quick.state').relative_to(S)),package_writes=0,missing_font_negative=True,rights=m['rights']))
(S/'package-verification.json').write_text(json.dumps(dict(status='PASS_SIX_PACKAGE_CONTENT_RIGHTS_STRUCTURE_AND_LINUX_NATIVE_GUI_DAT',version=a.version,build_head=a.head,reports=reports,limits='macOS僅雙架構、簽章、最低版本、相依與資料結構；Windows實跑另見Wine收據；兩平台真機仍未驗。'),ensure_ascii=False,indent=2)+'\n');print('六包內容／權利／結構、Linux兩包實際GUI與DAT存檔、缺字型負對照通過。')
