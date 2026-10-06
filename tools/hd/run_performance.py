"""研究038 §105：有界Xvfb A/B成本抽測，只用既有正常checkpoint。"""
from pathlib import Path
import argparse
import hashlib
import json
import os
import shutil
import subprocess
import sys
import time

args = argparse.ArgumentParser()
args.add_argument('work', type=Path)
args.add_argument('--binary', type=Path)
options = args.parse_args()
root = Path('/src')
work = options.work
fixture = root/'workplace/hd/gui-ally-items-v1-20261004/run-v5/e-forget-hd-chinese.state'
side = Path(str(fixture)+'.xlate.json')
binary = options.binary or work/'psychicwar-measured.bin'
assert binary.is_file() and fixture.is_file() and side.is_file()
assert os.environ.get('DISPLAY')

def command(*args):
    return subprocess.check_output(args,text=True,timeout=10).strip()

for mode in ['absent','loaded-off','on']:
    out = work/mode
    assert not out.exists()
    out.mkdir()
    saves = out/'saves'
    saves.mkdir()
    start_load = float(Path('/proc/loadavg').read_text().split()[0])
    assert start_load < 7, f'load {start_load} exceeds contract'
    selection = '' if mode=='absent' else 'workplace/hd/theme-ally-items-v1-20261004'
    argv = [str(binary),'-orig','/orig/psychic-war','-audio','null','-speed','1','-scratch',str(saves),
            '-load-state',str(fixture),'-text','text','-font','font','-theme',selection,
            '-stats',str(out/'stats.jsonl'),'-record',str(out/'record.json'),'-quit-after','40s']
    (out/'execution.json').write_text(json.dumps({'argv':argv,'start_load':start_load,
        'inputs_sha256':{str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in [binary,fixture,side,Path(__file__)]},
        'interval_wall_ms':[10000,30000],'limits':'Normal inventory checkpoint then real F11; local measured build, static view with cursor changes; no battle/package/real GPU/audio.'},indent=2)+'\n')
    log = (out/'frontend.log').open('w')
    process = subprocess.Popen(argv,stdout=log,stderr=subprocess.STDOUT)
    events = []
    try:
        window = ''
        deadline = time.monotonic()+35
        while time.monotonic() < deadline and process.poll() is None:
            try:
                window = command('xdotool','search','--name','Psychic War').splitlines()[0]
                break
            except (subprocess.CalledProcessError,IndexError):
                time.sleep(.2)
        assert window
        command('xdotool','windowfocus',window)
        def key(name,shift=False):
            if shift: command('xdotool','keydown','Shift_L')
            command('xdotool','keydown',name)
            time.sleep(.18)
            command('xdotool','keyup',name)
            if shift: command('xdotool','keyup','Shift_L')
            time.sleep(1)
            events.append({'key':name,'shift':shift})
        time.sleep(2)
        key('F10')
        shutil.copyfile(fixture,saves/'quick.state')
        shutil.copyfile(side,saves/'quick.state.xlate.json')
        key('F11')
        if mode=='loaded-off': key('F5',True)
        # All mode/restore keys precede the fixed 10..30s measurement window.
        deadline = time.monotonic()+60
        while time.monotonic() < deadline and process.poll() is None:
            lines = (out/'stats.jsonl').read_text().splitlines()
            ticks = [json.loads(line) for line in lines if line.strip()]
            if ticks and ticks[-1]['wall_ms'] >= 32000:
                key('F10')
                for filename in ['quick.state','quick.state.xlate.json','quick.json']:
                    shutil.copyfile(saves/filename,out/filename)
                command('import','-window',window,'-depth','8','PNG24:'+str(out/'window.png'))
                break
            time.sleep(.25)
        code = process.wait(timeout=30)
        assert code == 0
        (out/'terminal.json').write_text(json.dumps({'exit_code':code,'events':events,'window':window},indent=2)+'\n')
        print(mode,'completed',flush=True)
    finally:
        if process.poll() is None:
            process.terminate()
            try: process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=5)
        log.close()
