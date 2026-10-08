"""Docker/Xvfb內從實際AppImage正常開機，錄下遊玩與Shift+F5原版／HD切換。"""
from pathlib import Path
import argparse,hashlib,json,os,shutil,subprocess,tempfile,time
p=argparse.ArgumentParser();p.add_argument('package',type=Path);p.add_argument('out',type=Path);p.add_argument('--external-recorder',action='store_true');a=p.parse_args()
assert a.package.is_file() and not a.out.exists();a.out.mkdir(parents=True);O=a.out.resolve();(O/'saves').mkdir()
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
def cmd(*args):return subprocess.check_output(args,text=True,timeout=12).strip()
events=[];start=0;record_start=None;window='';rec=None
def quiet():
 last=None;same=0;end=time.monotonic()+15
 while time.monotonic()<end:
  raw=subprocess.check_output(['import','-window',window,'-depth','8','RGB:-'],timeout=10)
  digest=hashlib.sha256(raw).digest();same=same+1 if digest==last else 0;last=digest
  if same>=2:return
  time.sleep(.2)
def event(kind,**fields):events.append(dict(kind=kind,wall=time.monotonic()-start,video_time=None if record_start is None else time.monotonic()-record_start,**fields))
def key(k,wait=1,shift=False,settle=True):
 if settle:quiet()
 if shift:cmd('xdotool','keydown','Shift_L')
 cmd('xdotool','keydown',k);time.sleep(.2);cmd('xdotool','keyup',k)
 if shift:cmd('xdotool','keyup','Shift_L')
 event('key',key=k,shift=shift);time.sleep(wait)
def shot(name):cmd('import','-window',window,'-depth','8','PNG24:'+str(O/(name+'.png')))
with tempfile.TemporaryDirectory(prefix='pw-live-') as td:
 root=Path(td);subprocess.run([str(a.package.resolve()),'--appimage-extract'],cwd=root,check=True,stdout=subprocess.DEVNULL,timeout=40)
 app=root/'squashfs-root';manifest=json.loads((app/'PACKAGE-MANIFEST.json').read_text());assert manifest['rights']=='local-only-original-and-all-HD'
 assert cmd(str(app/'AppRun'),'-version')==manifest['version']
 argv=[str(app/'AppRun'),'-audio','null','-speed','1','-scratch',str(O/'saves'),'-record',str(O/'record.json'),'-stats',str(O/'stats.jsonl'),'-quit-after','180s']
 log=(O/'frontend.log').open('x');proc=subprocess.Popen(argv,cwd='/tmp',stdout=log,stderr=subprocess.STDOUT,env=dict(os.environ,PSYCHICWAR_KEYLOG='1'));start=time.monotonic()
 try:
  while proc.poll() is None and time.monotonic()-start<30:
   try:window=cmd('xdotool','search','--onlyvisible','--name','Psychic War').splitlines()[0];break
   except (subprocess.CalledProcessError,IndexError):time.sleep(.2)
  assert window;cmd('xdotool','windowfocus',window);geometry=cmd('xdotool','getwindowgeometry','--shell',window);assert 'WIDTH=960' in geometry and 'HEIGHT=600' in geometry
  time.sleep(max(0,20-(time.monotonic()-start)))
  for k in ['space','Return','Return','space','Return','k','a','i','Return']:key(k)
  quiet();shot('maze-hd-before');key('F10',.5);shutil.copyfile(O/'saves/quick.state',O/'before-live.state')
  g=dict(line.split('=',1) for line in geometry.splitlines() if '=' in line);display=os.environ['DISPLAY'].split('.')[0]+'.0+'+g['X']+','+g['Y']
  ffargv=['ffmpeg','-y','-loglevel','warning','-thread_queue_size','512','-f','x11grab','-framerate','25','-video_size','960x600','-i',display,'-t','72','-an','-threads','2','-c:v','libx264','-preset','veryfast','-crf','20','-pix_fmt','yuv420p',str(O/'gameplay.mp4')]
  if a.external_recorder:
   (O/'capture-ready.json').write_text(json.dumps(dict(display=display,argv=ffargv)))
   deadline=time.monotonic()+30
   while not (O/'capture-started.json').exists() and time.monotonic()<deadline:time.sleep(.1)
   assert (O/'capture-started.json').exists();record_start=json.loads((O/'capture-started.json').read_text())['monotonic']
  else:
   rlog=(O/'capture.log').open('x');rec=subprocess.Popen(ffargv,stdout=rlog,stderr=subprocess.STDOUT);record_start=time.monotonic()
  event('record-start',HD=True)
  time.sleep(4);key('F5',3,shift=True,settle=False);shot('maze-original-live');event('mode',HD=False)
  key('F5',3,shift=True,settle=False);shot('maze-hd-live');event('mode',HD=True)
  key('Left',2,settle=False);key('Right',2,settle=False)
  for n in range(6):key('Up',2.2,settle=False);event('move',number=n+1)
  cmd('xdotool','keydown','space');event('attack-hold')
  time.sleep(5);key('F5',3,shift=True,settle=False);shot('battle-original-live');event('mode',HD=False)
  key('F5',3,shift=True,settle=False);shot('battle-hd-live');event('mode',HD=True)
  time.sleep(14);cmd('xdotool','keyup','space');event('attack-release');shot('after-battle')
  key('F4',3,settle=False);shot('help-live');key('F4',1,settle=False)
  remaining=68-(time.monotonic()-record_start)
  if remaining>0:time.sleep(remaining)
  key('F10',.5,settle=False)
  for suffix in ['state','json','state.xlate.json']:shutil.copyfile(O/'saves'/('quick.'+suffix),O/('after-live.'+suffix))
  if a.external_recorder:
   deadline=time.monotonic()+25
   while not (O/'capture-finished.json').exists() and time.monotonic()<deadline:time.sleep(.1)
   assert json.loads((O/'capture-finished.json').read_text())['returncode']==0
  else:rec.wait(timeout=20);assert rec.returncode==0;rlog.close()
  proc.wait(timeout=110);assert proc.returncode==0
 finally:
  if rec is not None and rec.poll() is None:rec.send_signal(2);rec.wait(timeout=10)
  if proc.poll() is None:proc.terminate();proc.wait(timeout=5)
  log.close()
  (O/'execution.json').write_text(json.dumps(dict(version=manifest['version'],build_head=manifest['build_head'],argv=argv,capture_argv=locals().get('ffargv'),returncode=proc.returncode,events=events,package_sha256=sha(a.package),capture_sha256=sha(O/'gameplay.mp4') if (O/'gameplay.mp4').exists() else None,script_sha256=sha(Path(__file__)),initial='Normal boot; no imported state or RAM/position/seed injection',limits='Actual packaged gameplay, null device audio; promo music uses the designated original DOSBox-X recording.'),ensure_ascii=False,indent=2)+'\n')
print('實際AppImage正常遊玩72秒錄影、原版／HD切換及F10完成。')
