"""研究038 §74：真正視窗的逐畫面鍵盤操作；只在Xvfb容器內執行。"""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time

out = Path(sys.argv[1])
binary = Path(sys.argv[2])
initial = Path(sys.argv[3])
assert not out.exists() and os.environ.get('DISPLAY')
assert out.parent.stat().st_uid == os.getuid()
out.mkdir(); (out/'saves').mkdir()
events = []
def run(*args):
    return subprocess.check_output(args, text=True).strip()
def command(*args):
    subprocess.run(args, check=True)
def key(name, shift=False):
    if shift: command('xdotool','keydown','Shift_L')
    command('xdotool','keydown',name); time.sleep(.22)
    command('xdotool','keyup',name)
    if shift: command('xdotool','keyup','Shift_L')
    time.sleep(3)
def shot(label):
    assert label.isascii() and label.replace('-','').isalnum()
    key('F10')
    for source, suffix in [('quick.state','.state'),('quick.state.xlate.json','.state.xlate.json'),('quick.json','.meta.json')]:
        shutil.copyfile(out/'saves'/source,out/(label+suffix))
    for n in range(12):
        command('import','-window',window,'-define','png:color-type=2','-depth','8','PNG24:'+str(out/f'{label}-phase{n:02}.png'))
        time.sleep(.12)
    print('擷取完成 '+label, flush=True)

argv = [str(binary),'-orig','/orig/psychic-war','-audio','null','-speed','1',
        '-scratch',str(out/'saves'),'-load-state',str(initial),'-text','text','-font','font',
        '-theme','workplace/hd/theme-room22-v1-20261003','-stats',str(out/'stats.jsonl'),
        '-text-log',str(out/'text.jsonl'),'-record',str(out/'record.json'),'-quit-after',os.environ.get('GUI_DAT_QUIT_AFTER','14m')]
(out/'execution.json').write_text(json.dumps({'argv':argv,'load':Path('/proc/loadavg').read_text().strip(),
    'inputs_sha256':{str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in (Path(__file__),binary,initial)},
    'limits':'正常state接續與視窗逐步操作；不宣稱固定GUI亂數或牆上時間對拍'},ensure_ascii=False,indent=2)+'\n')
env=dict(os.environ,PSYCHICWAR_KEYLOG='1')
log=(out/'frontend.log').open('w')
process=subprocess.Popen(argv,stdout=log,stderr=subprocess.STDOUT,env=env)
last=0
try:
    deadline=time.monotonic()+45
    window=''
    while time.monotonic()<deadline and process.poll() is None:
        try: window=run('xdotool','search','--name','Psychic War').splitlines()[0]
        except (subprocess.CalledProcessError,IndexError): time.sleep(.2); continue
        break
    assert window, '沒有前端視窗'
    command('xdotool','windowfocus',window)
    geometry=run('xdotool','getwindowgeometry','--shell',window)
    assert 'WIDTH=960' in geometry and 'HEIGHT=600' in geometry
    (out/'window.txt').write_text(geometry+'\n')
    time.sleep(5)
    shot('a-hd-room')
    inbox=out/'commands.json'
    while process.poll() is None:
        if inbox.exists():
            data=json.loads(inbox.read_text())
            if data['sequence']>last:
                assert data['sequence']==last+1
                for item in data['actions']:
                    if item['kind']=='key': key(item['key'],item.get('shift',False))
                    elif item['kind']=='shot': shot(item['label'])
                    elif item['kind']=='wait': time.sleep(min(item['seconds'],10))
                    elif item['kind']=='finish':
                        # 由前端既有quit-after正常結束，讓按鍵紀錄完成；不用XDestroyWindow。
                        process.wait(timeout=850); break
                    else: raise AssertionError('不支援的操作')
                    events.append(item)
                last=data['sequence']
                (out/'completed.json').write_text(json.dumps({'sequence':last,'actions':events},ensure_ascii=False,indent=2)+'\n')
                print('命令批次完成 '+str(last),flush=True)
        time.sleep(.2)
    print('前端終止碼 '+str(process.returncode),flush=True)
finally:
    if process.poll() is None:
        process.terminate()
        try: process.wait(timeout=10)
        except subprocess.TimeoutExpired: process.kill(); process.wait()
    log.close()
    (out/'terminal.json').write_text(json.dumps({'returncode':process.returncode,'last_sequence':last,'terminal':True},indent=2)+'\n')
