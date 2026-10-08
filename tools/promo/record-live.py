"""既有FFmpeg容器透過唯讀X11 socket錄影；與GUI容器以有界檔案訊號同步。"""
from pathlib import Path
import argparse,json,subprocess,time
p=argparse.ArgumentParser();p.add_argument('capture',type=Path);a=p.parse_args();O=a.capture;deadline=time.monotonic()+140
while not (O/'capture-ready.json').exists() and time.monotonic()<deadline:time.sleep(.1)
assert (O/'capture-ready.json').exists();ready=json.loads((O/'capture-ready.json').read_text());argv=ready['argv'];log=(O/'capture.log').open('x')
proc=subprocess.Popen(argv,stdout=log,stderr=subprocess.STDOUT);started=time.monotonic();tmp=O/'capture-started.tmp';tmp.write_text(json.dumps(dict(monotonic=started,argv=argv)));tmp.rename(O/'capture-started.json')
try:proc.wait(timeout=100);assert proc.returncode==0
finally:
 if proc.poll() is None:proc.send_signal(2);proc.wait(timeout=10)
 log.close();tmp=O/'capture-finished.tmp';tmp.write_text(json.dumps(dict(returncode=proc.returncode,seconds=time.monotonic()-started)));tmp.rename(O/'capture-finished.json')
print('72秒X11實際畫面錄影完成。')
