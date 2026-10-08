"""Docker/Wine內驗實際Windows ZIP；不代表Windows真機。"""
from pathlib import Path
import argparse,json,os,shutil,subprocess,tempfile,time,zipfile
p=argparse.ArgumentParser();p.add_argument('version');a=p.parse_args();R=Path('/src');D=R/'dist-all'/a.version;S=D/'smoke';S.mkdir(exist_ok=True);rows=[]
def win(p):return 'Z:'+str(p).replace('/','\\')
for kind in ['patch','full-local']:
 file=next((D/kind).glob('*win64.zip'));out=S/('wine-visible-'+kind);assert not out.exists();out.mkdir()
 with tempfile.TemporaryDirectory(prefix='pw-wine-') as td:
  base=Path(td);prefix=base/'prefix';env=dict(os.environ,WINEPREFIX=str(prefix),WINEDEBUG='-all',LIBGL_ALWAYS_SOFTWARE='1',EBITENGINE_GRAPHICS_LIBRARY='opengl')
  with zipfile.ZipFile(file) as z:z.extractall(base)
  pkg=base/'PsychicWar';exe=pkg/'PsychicWar.exe'
  argv=['wine',str(exe),'-audio','null','-quit-after','25s']
  if kind=='patch':argv+=['-orig',win(Path('/orig/psychic-war'))]
  else:argv+=['-theme',win(pkg/'theme/hd')]
  log=(out/'frontend.log').open('x');proc=subprocess.Popen(argv,cwd='/tmp',env=env,stdout=log,stderr=subprocess.STDOUT);start=time.monotonic()
  try:
   window=''
   while proc.poll() is None and time.monotonic()-start<150:
    q=subprocess.run(['xdotool','search','--onlyvisible','--name','Psychic War'],capture_output=True,text=True)
    if q.returncode==0:window=q.stdout.splitlines()[0];break
    time.sleep(.2)
   assert window;startup_seconds=time.monotonic()-start;subprocess.run(['xdotool','windowfocus',window],check=True);time.sleep(1)
   subprocess.run(['xdotool','keydown','F10'],check=True);time.sleep(.25);subprocess.run(['xdotool','keyup','F10'],check=True);time.sleep(1)
   subprocess.run(['import','-window',window,'-depth','8','PNG24:'+str(out/'window.png')],check=True)
   proc.wait(timeout=45);assert proc.returncode==0
   saves=list(prefix.rglob('PsychicWar/quick.state'));assert len(saves)==1 and saves[0].stat().st_size>0
   shutil.copyfile(saves[0],out/'quick.state');assert not list(pkg.rglob('quick.state'))
   rows.append(dict(kind=kind,exit_code=0,startup_seconds=startup_seconds,F10_saved_to_AppData=True,package_writes=0,scope='Wine GUI；非Windows真機'))
  finally:
   for f in prefix.rglob('psychicwar-error.log'):
    shutil.copyfile(f,out/'error.log')
   if not window:
    subprocess.run(['import','-window','root',str(out/'desktop.png')],capture_output=True,timeout=10)
   (out/'execution.json').write_text(json.dumps(dict(argv=argv,returncode=proc.poll(),scope='Normal boot of actual packaged EXE; no imported Linux state'),indent=2)+'\n')
   if proc.poll() is None:proc.terminate();proc.wait(timeout=5)
   subprocess.run(['wineserver','-k'],env=env,capture_output=True,timeout=10);log.close()
(S/'windows-wine-verification.json').write_text(json.dumps(dict(status='PASS_ACTUAL_WINDOWS_ZIPS_WINE_GUI_AND_APPDATA_SAVE',version=a.version,rows=rows,limits='Linux容器Wine、軟體OpenGL及null音訊；真Windows輸入與裝置音訊仍屬#44。'),ensure_ascii=False,indent=2)+'\n');print('Windows兩個實際ZIP在Wine啟動、F10與AppData存檔通過。')
