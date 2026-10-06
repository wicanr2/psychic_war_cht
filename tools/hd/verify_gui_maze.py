"""研究038 §140：原始PBL／MAZE／slot表／文本獨立合成正式視窗。"""
import base64
import importlib.util
import json
from pathlib import Path
import subprocess
import sys

from verify_anchor_render import art_plane
from verify_next_body_plane import W, H, S, FW, FH, pbl, rgba, require, sha
from verify_over_runtime import load_font, render_text, read
from verify_over_frontend import compose, mismatch

out, theme = map(Path, sys.argv[1:])
inputs = {}
manifest = json.loads(read(theme / 'manifest.json', inputs))
require(manifest['schema'] == 'psychic-war-theme/2', '不是新版主題')
base, decoded = bytearray(W * H), {}
for name in sorted({e['pbl'] for e in manifest['entries']}):
    data = read(Path('/orig/psychic-war') / name, inputs)
    for image, offset, _ in pbl.images(data):
        w, h, pixels = pbl.decode(data, offset)
        decoded[name, image] = w, h, bytes(pixels)
        if name in ('SCREEN.PBL', 'MENU.PBL'):
            x, y = (0, image * 40) if name == 'SCREEN.PBL' else (160, 4)
            for yy in range(h):
                base[(y+yy)*W+x:(y+yy)*W+x+w] = pixels[yy*w:(yy+1)*w]
assets = []
for entry in manifest['entries']:
    if entry['pbl'] == 'ROOM0.PBL' and entry['image'] == 22:
        continue
    w, h, source = decoded[entry['pbl'], entry['image']]
    path = theme / entry['png']
    read(path, inputs)
    pw, ph, pixels = rgba(path)
    require((pw, ph) == (w*S, h*S), '候選尺寸不同')
    reference = bytearray(W*H) if entry['pbl'] == 'OVER.PBL' else bytearray(base)
    if entry['pbl'] not in ('SCREEN.PBL', 'MENU.PBL'):
        x, y = entry['at']
        for yy in range(h):
            reference[(y+yy)*W+x:(y+yy)*W+x+w] = source[yy*w:(yy+1)*w]
    assets.append((entry, w, h, source, pixels, reference))

spec = importlib.util.spec_from_file_location('native', 'tools/hd/prototype_maze_theme.py')
native = importlib.util.module_from_spec(spec)
spec.loader.exec_module(native)
native.mask.maze = read('/orig/psychic-war/MAZE.BIN', inputs)
atlas_path = theme / manifest['maze']['atlas']
read(atlas_path, inputs)
_, _, atlas = native.rgba_png(atlas_path)
tiles = [b''.join(atlas[((n//16*12+y)*192+n%16*12)*4:
                       ((n//16*12+y)*192+n%16*12+12)*4] for y in range(12)) for n in range(256)]
fonts = {n: load_font('font/'+n+'.golemfnt', inputs) for n in ('cjk24', 'cjk16')}
formal = {}
for path in Path('text').glob('*.json'):
    data = json.loads(read(path, inputs))
    if isinstance(data, dict):
        for entry in data.get('entries', []):
            if 'key' in entry and 'translation' in entry:
                formal[entry['key']] = entry['translation']
rows = []
empty = bytes(FW*FH*4)
reader = Path('workplace/ida/hd-ally-recruit-20261004/pionn-maze-state-read-v1.bin')
read(reader, inputs)
captures = [x for x in json.loads((out/'terminal.json').read_text())['actions'] if 'capture' in x]
for capture in captures:
    label = capture['capture']
    prefix = out / label
    state = Path(str(prefix)+'.state')
    frame = read(str(state)+'.frame', inputs)
    rgb = read(str(state)+'.rgb.bin', inputs)
    saved = json.loads(subprocess.check_output([str(reader.resolve()), str(state.resolve())]))
    require(base64.b64decode(saved['maze_source']) == native.mask.maze, '原始MAZE來源不同')
    slots = list(base64.b64decode(saved['maze_table']))
    art = native.scene(slots, tiles)
    small, cells = native.mask(frame, base, slots, art)
    require(len(cells) == 100, '正常穩定視野尚未完整重建')
    maze = bytearray(FW*FH*4)
    for y in range(240):
        maze[((360+y)*FW)*4:((360+y)*FW+240)*4] = small[y*240*4:(y+1)*240*4]
    upper, full, _ = art_plane(frame, assets)
    side = json.loads(read(str(prefix)+'.state.xlate.json', inputs))
    for stamp in side.get('stamps', []):
        if stamp.get('text'):
            require(stamp['key'] in formal and stamp['text'].strip() in formal[stamp['key']], '中文不是正式來源')
    meta = json.loads(read(str(prefix)+'.meta.json', inputs))
    text = render_text(side, fonts) if meta['language'] == 'zh' else empty
    planes = (maze, upper, text) if capture['hd'] else (text,)
    expected = compose(rgb, planes)
    phases = sorted(out.glob(label+'-phase*.png'))
    require(len(phases) == 8, '視窗相位數不符')
    diffs = []
    for path in phases:
        read(path, inputs)
        w, h, actual = rgba(path)
        require((w, h) == (FW, FH), '視窗尺寸不同')
        diffs.append({'png': str(path), 'mismatch': mismatch(expected, actual)})
        if diffs[-1]['mismatch'] == 0:
            break
    require(any(x['mismatch'] == 0 for x in diffs), '整屏不符 '+label+' '+str(diffs))
    omit_maze = mismatch(expected, compose(rgb, (upper, text))) if capture['hd'] else 0
    require(not capture['hd'] or omit_maze > 0, '移除迷宮負對照無效')
    rows.append({'label': label, 'hd': capture['hd'], 'cells': len(cells), 'diffs': diffs,
                 'omit_maze_mismatch': omit_maze, 'full_sources': full})
require(len(rows) == 5, 'GUI抽樣不足')
left = json.loads(subprocess.check_output([str(reader.resolve()), str((out/'b-left-hd.state').resolve())]))
restored = json.loads(subprocess.check_output([str(reader.resolve()), str((out/'e-loaded-hd.state').resolve())]))
require(left['direction'] == restored['direction'] and left['area'] == restored['area'], 'F11未恢復玩家位置／朝向')
require((out/'b-left-hd.state.frame').read_bytes() == (out/'e-loaded-hd.state.frame').read_bytes(), 'F11原版整屏未恢復')
receipt = {'scope': '正式GUI正常轉向、HD開關與F11；獨立原始資料整屏期望，非封包／全場景',
           'results': rows, 'f11_original_frame_restored': True, 'inputs_sha256': inputs,
           'source_text': Path(__file__).read_text()}
with (out/'independent-verification.json').open('x') as f:
    json.dump(receipt, f, ensure_ascii=False, indent=2)
    f.write('\n')
print('正式GUI 5/5整屏相同，迷宮省略負對照有效，F11恢復原版畫面')
