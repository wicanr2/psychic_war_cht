"""研究038 §140：正式視窗迷宮、正常轉向、HD開關及F11。限容器。"""
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time

out, binary, initial, theme = map(Path, sys.argv[1:])
assert not out.exists() and os.environ.get('DISPLAY')
out.mkdir()
(out / 'saves').mkdir()
events = []


def command(*args):
    return subprocess.check_output(args, text=True, timeout=5).strip()


def key(name, shift=False):
    if shift:
        command('xdotool', 'keydown', 'Shift_L')
    command('xdotool', 'keydown', name)
    time.sleep(.20)
    command('xdotool', 'keyup', name)
    if shift:
        command('xdotool', 'keyup', 'Shift_L')
    time.sleep(1.5)
    events.append({'key': name, 'shift': shift})


def shot(label, hd):
    key('F10')
    for source, suffix in [('quick.state', '.state'), ('quick.state.xlate.json', '.state.xlate.json'), ('quick.json', '.meta.json')]:
        shutil.copyfile(out / 'saves' / source, out / (label + suffix))
    for phase in range(8):
        path = out / f'{label}-phase{phase:02}.png'
        command('import', '-window', window, '-depth', '8', 'PNG24:' + str(path))
        time.sleep(.1)
    events.append({'capture': label, 'hd': hd})
    print('擷取 ' + label, flush=True)


argv = [str(binary), '-orig', '/orig/psychic-war', '-audio', 'null', '-speed', '1',
        '-scratch', str(out / 'saves'), '-load-state', str(initial), '-text', 'text',
        '-font', 'font', '-theme', str(theme), '-record', str(out / 'record.json'),
        '-quit-after', '3m']
(out / 'execution.json').write_text(json.dumps({'argv': argv, 'load': Path('/proc/loadavg').read_text(),
    'scope': '正常checkpoint後正式鍵盤轉向，HD切換及F11；非從開機／完整HD／封包驗收'}, ensure_ascii=False, indent=2) + '\n')
log = (out / 'frontend.log').open('w')
process = subprocess.Popen(argv, stdout=log, stderr=subprocess.STDOUT)
try:
    deadline = time.monotonic() + 10
    window = ''
    while time.monotonic() < deadline and process.poll() is None:
        try:
            window = command('xdotool', 'search', '--name', 'Psychic War').splitlines()[0]
            break
        except (subprocess.CalledProcessError, IndexError):
            time.sleep(.2)
    assert window, '沒有正式前端視窗'
    command('xdotool', 'windowfocus', window)
    geometry = command('xdotool', 'getwindowgeometry', '--shell', window)
    assert 'WIDTH=960' in geometry and 'HEIGHT=600' in geometry
    (out / 'window.txt').write_text(geometry + '\n')
    time.sleep(2)
    shot('a-initial-hd', True)
    key('Left')
    shot('b-left-hd', True)
    checkpoint = out / 'load-point'
    checkpoint.mkdir()
    for p in (out / 'saves').glob('quick*'):
        shutil.copyfile(p, checkpoint / p.name)
    key('F5', True)
    shot('c-original', False)
    key('F5', True)
    shot('d-hd-again', True)
    key('Right')
    for p in checkpoint.glob('quick*'):
        shutil.copyfile(p, out / 'saves' / p.name)
    key('F11')
    shot('e-loaded-hd', True)
finally:
    if process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait()
    log.close()
    (out / 'terminal.json').write_text(json.dumps({'returncode': process.returncode,
        'terminal': True, 'actions': events}, ensure_ascii=False, indent=2) + '\n')
