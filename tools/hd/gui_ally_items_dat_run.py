"""研究038 §102：27筆主題的正常原版DAT存讀檔；容器Xvfb限定。"""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time


out, binary, initial, side = map(Path, sys.argv[1:])
assert not out.exists() and os.environ.get('DISPLAY')
out.mkdir()
(out / 'saves').mkdir()
dat_source = os.environ.get('GUI_DAT_SOURCE')
if dat_source:
    source = Path(dat_source)
    assert source.stat().st_size == 512
    shutil.copyfile(source, out / 'saves/hd27.dat')

events = []


def command(*args):
    return subprocess.check_output(args, text=True, timeout=10).strip()


def key(name, shift=False):
    if shift:
        command('xdotool', 'keydown', 'Shift_L')
    command('xdotool', 'keydown', name)
    time.sleep(.18)
    command('xdotool', 'keyup', name)
    if shift:
        command('xdotool', 'keyup', 'Shift_L')
    time.sleep(3)


def copy_quick(label):
    for source, suffix in (('quick.state', '.state'), ('quick.state.xlate.json', '.state.xlate.json'), ('quick.json', '.meta.json')):
        shutil.copyfile(out / 'saves' / source, out / (label + suffix))


def shot(label, hd):
    assert label.isascii() and label.replace('-', '').isalnum()
    key('F10')
    copy_quick(label)
    meta = json.loads((out / (label + '.meta.json')).read_text())
    rows = []
    for i in range(12):
        path = out / f'{label}-phase{i:02}.png'
        command('import', '-window', window, '-define', 'png:color-type=2', '-depth', '8', 'PNG24:' + str(path))
        rows.append(str(path))
        time.sleep(.12)
    events.append(dict(kind='captured', label=label, hd=hd, language=meta['language'], phases=rows))
    print('擷取完成 ' + label, flush=True)


argv = [str(binary), '-orig', '/orig/psychic-war', '-audio', 'null', '-speed', '1', '-scratch', str(out / 'saves'), '-load-state', str(initial), '-text', 'text', '-font', 'font', '-theme', 'workplace/hd/theme-ally-items-v1-20261004', '-stats', str(out / 'stats.jsonl'), '-text-log', str(out / 'text.jsonl'), '-record', str(out / 'record.json'), '-quit-after', os.environ.get('GUI_DAT_QUIT_AFTER', '6m')]
(out / 'execution.json').write_text(json.dumps(dict(argv=argv, load=Path('/proc/loadavg').read_text().strip(), inputs_sha256={str(p): hashlib.sha256(p.read_bytes()).hexdigest() for p in (Path(__file__), binary, initial, side, *([Path(dat_source)] if dat_source else []))}, limits='正常checkpoint後F11還原正式中文字面，再正常鍵盤操作；未驗從開機、自然亂數對拍或封包'), ensure_ascii=False, indent=2) + '\n')
log = (out / 'frontend.log').open('w')
process = subprocess.Popen(argv, stdout=log, stderr=subprocess.STDOUT, env=dict(os.environ, PSYCHICWAR_KEYLOG='1'))
last = 0
try:
    deadline = time.monotonic() + 45
    window = ''
    while time.monotonic() < deadline and process.poll() is None:
        try:
            window = command('xdotool', 'search', '--name', 'Psychic War').splitlines()[0]
        except (subprocess.CalledProcessError, IndexError):
            time.sleep(.2)
            continue
        break
    assert window, '沒有前端視窗'
    command('xdotool', 'windowfocus', window)
    geometry = command('xdotool', 'getwindowgeometry', '--shell', window)
    assert 'WIDTH=960' in geometry and 'HEIGHT=600' in geometry
    (out / 'window.txt').write_text(geometry + '\n')
    time.sleep(5)
    key('F10')
    shutil.copyfile(initial, out / 'saves/quick.state')
    shutil.copyfile(side, out / 'saves/quick.state.xlate.json')
    key('F11')
    shot(os.environ.get('GUI_DAT_INITIAL_LABEL', 'a-before-save-hd-chinese'), True)
    print('等待正常道具操作命令', flush=True)
    inbox = out / 'commands.json'
    while process.poll() is None:
        if inbox.exists():
            request = json.loads(inbox.read_text())
            if request['sequence'] > last:
                assert request['sequence'] == last + 1
                for item in request['actions']:
                    if item['kind'] == 'key':
                        key(item['key'], item.get('shift', False))
                    elif item['kind'] == 'shot':
                        shot(item['label'], item['hd'])
                    elif item['kind'] == 'wait':
                        time.sleep(min(item['seconds'], 10))
                    elif item['kind'] == 'copy-load-point':
                        target = out / item['label']
                        target.mkdir()
                        for p in (out / 'saves').glob('quick*'):
                            shutil.copyfile(p, target / p.name)
                    elif item['kind'] == 'restore-load-point':
                        for p in (out / item['label']).glob('quick*'):
                            shutil.copyfile(p, out / 'saves' / p.name)
                    elif item['kind'] == 'finish':
                        # F11 會重設正式前端的退出計時；涵蓋預設十分鐘期限。
                        # Docker 外層期限仍需包含最後一次 F11 前的全部操作。
                        process.wait(timeout=650)
                    else:
                        raise AssertionError('未知命令')
                    if item['kind'] != 'shot':
                        events.append(item)
                last = request['sequence']
                (out / 'completed.json').write_text(json.dumps(dict(sequence=last, actions=events), ensure_ascii=False, indent=2) + '\n')
                print('命令批次完成 ' + str(last), flush=True)
        time.sleep(.2)
    print('前端終止碼 ' + str(process.returncode), flush=True)
finally:
    if process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait()
    log.close()
    (out / 'terminal.json').write_text(json.dumps(dict(returncode=process.returncode, last_sequence=last, terminal=True, actions=events), ensure_ascii=False, indent=2) + '\n')
