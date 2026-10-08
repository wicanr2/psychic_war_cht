"""目前前端的正常開機、移動與戰鬥抽樣；需Docker/Xvfb。"""
from pathlib import Path
import argparse,hashlib,json,os,shutil,subprocess,time
p=argparse.ArgumentParser();p.add_argument('binary');p.add_argument('theme');p.add_argument('out');a=p.parse_args()
O=Path(a.out);assert not O.exists();O.mkdir();(O/'saves').mkdir()
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
argv=[a.binary,'-orig','/orig/psychic-war','-text','/src/text','-font','/src/font','-theme',a.theme,'-scratch',str(O/'saves'),'-audio','null','-speed','1','-record',str(O/'record.json'),'-quit-after','160s']
events=[];log=(O/'frontend.log').open('x');proc=subprocess.Popen(argv,stdout=log,stderr=subprocess.STDOUT,env=dict(os.environ,PSYCHICWAR_KEYLOG='1'));start=time.monotonic()
def cmd(*x):return subprocess.check_output(x,text=True,timeout=10).strip()
def quiet():
 # 原版大型COPY仍在進行時不送下一個鍵；比較像素，排除PNG時間標籤。
 last=None;same=0;deadline=time.monotonic()+15
 while time.monotonic()<deadline:
  raw=subprocess.check_output(['import','-window',window,'-depth','8','RGB:-'],timeout=10)
  digest=hashlib.sha256(raw).digest();same=same+1 if digest==last else 0;last=digest
  if same>=2:return
  time.sleep(.2)
def key(k,wait=1):
 quiet()
 cmd('xdotool','keydown',k);time.sleep(.18);cmd('xdotool','keyup',k);time.sleep(wait);events.append(dict(key=k,wall=time.monotonic()-start))
def shot(label):
 cmd('import','-window',window,'-define','png:color-type=2','-depth','8','PNG24:'+str(O/(label+'.png')));events.append(dict(shot=label,wall=time.monotonic()-start))
try:
 window=''
 while proc.poll() is None and time.monotonic()-start<20:
  try:window=cmd('xdotool','search','--name','Psychic War').splitlines()[0];break
  except (subprocess.CalledProcessError,IndexError):time.sleep(.2)
 assert window;cmd('xdotool','windowfocus',window);assert 'WIDTH=960' in cmd('xdotool','getwindowgeometry','--shell',window)
 for n in range(20):shot(f'opening{n:02d}');time.sleep(.7)
 time.sleep(max(0,20-(time.monotonic()-start)));shot('title')
 for n,k in enumerate(['space','Return','Return','space','Return','k','a','i','Return']):key(k);shot(f'boot{n:02d}')
 shot('maze-hd');cmd('xdotool','keydown','Shift_L');key('F5');cmd('xdotool','keyup','Shift_L');shot('maze-original')
 cmd('xdotool','keydown','Shift_L');key('F5');cmd('xdotool','keyup','Shift_L');key('F4');shot('help');key('F4')
 for n in range(6):key('Up',2.5);shot(f'move{n:02d}')
 cmd('xdotool','keydown','space')
 for n in range(12):shot(f'battle{n:02d}');time.sleep(.6)
 cmd('xdotool','keyup','space');key('F10',.5)
 for suffix in ['state','json','state.xlate.json']:shutil.copyfile(O/'saves'/('quick.'+suffix),O/('final.'+suffix))
 proc.wait(timeout=105);assert proc.returncode==0
finally:
 if proc.poll() is None:proc.terminate();proc.wait(timeout=5)
 log.close();(O/'execution.json').write_text(json.dumps(dict(argv=argv,returncode=proc.returncode,natural_exit=proc.returncode==0,events=events,inputs_sha256={str(p):sha(p) for p in [Path(__file__),Path(a.binary),Path(a.theme)/'manifest.json']},limits='Normal boot and original-key sampling; not complete playthrough or audio/hardware acceptance.'),ensure_ascii=False,indent=2)+'\n')
print('目前版本正常開機GUI擷取完成。')
