"""研究 038 §38：獨立核對原型戰鬥區的合成、原點 8×8 遮格及未知格回退。

只核對原型，不驗收造型；中途的來源推定仍引用 Go 收據，不宣稱獨立證明其唯一性。
"""
import argparse
import hashlib
import json
import math
from pathlib import Path
import sys
from collections import Counter

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import pbl


def digest(data):
    return hashlib.sha256(data).hexdigest()


def png(path):
    w, h, channels, rows, palette = pbl.read_png(path)
    assert channels in (3, 4) and palette is None, path
    data = bytearray()
    for row in rows:
        for x in range(w):
            data.extend(row[x*channels:x*channels+3])
            data.append(row[x*channels+3] if channels == 4 else 255)
    return w, h, data


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--receipt', required=True)
    parser.add_argument('--out', required=True)
    parser.add_argument('--source-snapshot', help='現行來源已有變更時，明示舊收據的逐檔快照 manifest')
    args = parser.parse_args()
    output = Path(args.out)
    assert not output.exists() and output.parent.is_dir()
    receipt_path = Path(args.receipt)
    receipt = json.loads(receipt_path.read_text())
    inputs = {str(receipt_path): digest(receipt_path.read_bytes()), __file__: digest(Path(__file__).read_bytes())}
    archived_inputs = {}
    snapshot = json.loads(Path(args.source_snapshot).read_text())['files'] if args.source_snapshot else {}
    for name, expected in receipt['inputs_sha256'].items():
        path = Path(name)
        if not path.is_file() or digest(path.read_bytes()) != expected:
            key = name.removeprefix('/src/').replace('/orig/', 'workplace/original/', 1)
            entry = snapshot.get(key)
            assert entry and entry['sha256'] == expected and digest(Path(entry['snapshot']).read_bytes()) == expected, name
            archived_inputs[name] = entry
        inputs[name] = expected
    profile_path = next(Path(n) for n in inputs if 'decomposition-v1' in n)
    profile = json.loads(profile_path.read_text())
    complete = {s['step']: s for s in profile['samples']}
    variables = profile['variables']
    assets = {}
    for name in ('SCREEN', 'MENU', 'ALLY', 'ENEMY00', 'BEAM', 'FIGHT'):
        data = Path('/orig/psychic-war', name+'.PBL').read_bytes()
        for n, off, _ in pbl.images(data):
            w, h, pixels = pbl.decode(data, off)
            assets[name, n] = (w, h, bytes(pixels))
    executable = Path('workplace/ida/PW_UNP.EXE').read_bytes()
    offset = int.from_bytes(executable[8:10], 'little')*16 + int.from_bytes(executable[22:24], 'little')*16 + 0x4E36
    stencil = executable[offset:offset+32]
    assets['MASK', 0] = (16, 16, bytes(10 if stencil[y*2+x//8] & (128 >> (x%8)) else 0 for y in range(16) for x in range(16)))
    base = bytearray(64000)
    def paste(dst, asset, x, y):
        w, h, pixels = asset
        for yy in range(h):
            start = (y+yy)*320+x
            dst[start:start+w] = pixels[yy*w:(yy+1)*w]
    for n in range(5):
        paste(base, assets['SCREEN', n], 0, n*40)
    paste(base, assets['MENU', 0], 160, 4)
    paste(base, assets['ALLY', 0], 264, 152)
    theme_dir = Path(next(n for n in inputs if n.endswith('/manifest.json'))).parent
    hd_base = bytearray(960*600*4)
    def over(dst, image, ox, oy):
        w, h, src = image
        for yy in range(h):
            for xx in range(w):
                j, k = 4*((oy+yy)*960+ox+xx), 4*(yy*w+xx)
                a = src[k+3]*257
                if a == 65535:
                    dst[j:j+4] = src[k:k+4]
                elif a:
                    assert dst[j+3] == 255, '本原型只允許不透明基底上的透明角色'
                    for c in range(3):
                        premult = src[k+c]*a//255
                        dst[j+c] = ((dst[j+c]*257*(65535-a)+premult*65535)//65535) >> 8
    for n in range(5):
        over(hd_base, png(theme_dir/f'SCREEN-{n:02d}.png'), 0, n*120)
    over(hd_base, png(theme_dir/'MENU-00.png'), 480, 12)
    over(hd_base, png(theme_dir/'ALLY-00.png'), 792, 456)
    assert set(hd_base[3::4]) == {255}
    hd_assets = []
    for v in variables:
        name, n, rect = v['name'], v['image'], v['rect']
        if name == 'MASK':
            candidates = [Path(n) for n in inputs if 'battle-effect-mask-48-' in n and n.endswith('.png')]
            assert len(candidates) == 1, '遮罩候選來源不唯一'
            path = candidates[0]
        elif name == 'ENEMY00' and rect[2] == 16:
            path = Path(f'workplace/hd/art-in/battle-effect-{n}-48-v2-20261001.png')
        elif name == 'ENEMY00':
            path = theme_dir/f'ENEMY00-{n:02d}.png'
        else:
            path = Path(f'workplace/hd/art-in/{name}-{n:02d}.png')
        hd_assets.append(png(path))
    observations = []
    negative_difference = 0
    for row in receipt['results']:
        prefix = Path(row['prefix'])
        frame = Path(str(prefix)+'.frame').read_bytes()
        assert digest(frame) == row['indexed_sha256']
        inputs[str(prefix)+'.frame'] = digest(frame)
        metadata = row['screen']
        active = metadata['active_variables']
        if row['step'] in complete:
            assert active == complete[row['step']]['active_variables']
            assert metadata['complete']
        model = bytearray(base)
        hidden = set()
        unknown = int(metadata['unknown_variables'], 16)
        for n, v in enumerate(variables):
            x, y, w, h = v['rect']
            if unknown >> n & 1:
                hidden.update((cx, cy) for cy in range(y//8, (y+h+7)//8) for cx in range(x//8, (x+w+7)//8))
            if n not in active:
                continue
            _, _, pixels = assets[v['name'], v['image']]
            for yy in range(h):
                for xx in range(w):
                    j = (y+yy)*320+x+xx
                    delta = pixels[yy*w+xx]
                    if v['name'] == 'ENEMY00' and w == 24:
                        delta ^= base[j]
                    model[j] ^= delta
        visible = {(cx, cy) for cy in range(25) for cx in range(40) if (cx, cy) not in hidden and all(frame[y*320+cx*8:y*320+cx*8+8] == model[y*320+cx*8:y*320+cx*8+8] for y in range(cy*8, cy*8+8))}
        assert len(visible) == metadata['matched_cells']
        fx = [n for n in active if not (variables[n]['name'] == 'ENEMY00' and variables[n]['rect'][2] == 24)]
        fx.sort(key=lambda n: {'FIGHT': 0, 'BEAM': 1, 'ENEMY00': 2, 'MASK': 3}[variables[n]['name']])
        text = json.loads(Path(str(prefix)+'.xlate.json').read_text())
        assert all(s['y']+s['cell_h'] <= 144 or s['y'] >= 184 for s in text['stamps']), '戰鬥區出現文字，需另建立獨立字形期望'
        original_path = Path(str(prefix)+'-original-cht.png')
        rw, rh, original = png(original_path)
        assert (rw, rh) == (960, 600)
        inputs[str(original_path)] = digest(original_path.read_bytes())
        for mode in ('alpha', 'screen'):
            hd = bytearray(hd_base)
            for n in active:
                v = variables[n]
                if v['name'] == 'ENEMY00' and v['rect'][2] == 24:
                    over(hd, hd_assets[n], v['rect'][0]*3, v['rect'][1]*3)
            transmission = {}
            for n in fx:
                v = variables[n]
                x, y, w, h = v['rect']
                _, _, ink = assets[v['name'], v['image']]
                _, _, pixels = hd_assets[n]
                for yy in range(h*3):
                    for xx in range(w*3):
                        cellx, celly = xx//24*8, yy//24*8
                        if not any(ink[i*w+j] for i in range(celly, min(celly+8, h)) for j in range(cellx, min(cellx+8, w))):
                            continue
                        k, j = 4*(yy*w*3+xx), 4*((y*3+yy)*960+x*3+xx)
                        a = pixels[k+3]
                        for c in range(3):
                            if mode == 'alpha':
                                hd[j+c] = (pixels[k+c]*a+hd[j+c]*(255-a)+127)//255
                            else:
                                transmission[j+c] = transmission.get(j+c, 1.0)*(1-pixels[k+c]*a/65025)
            for j, t in transmission.items():
                hd[j] = math.floor(255-(255-hd[j])*t+0.5)
            w, h, actual = png(Path(str(prefix)+'-'+mode+'.png'))
            assert (w, h) == (960, 600)
            inputs[str(prefix)+'-'+mode+'.png'] = digest(Path(str(prefix)+'-'+mode+'.png').read_bytes())
            mismatch = 0
            pairs = Counter()
            for yy in range(144*3, 184*3):
                for xx in range(32*3, 288*3):
                    j = 4*(yy*960+xx)
                    if (xx//24, yy//24) in visible:
                        expected = hd[j:j+4]
                    else:
                        # 原版使用遊戲自設色盤；此區已確認無中文，引用同狀態原版 RGB。
                        expected = original[j:j+4]
                    if actual[j:j+4] != expected:
                        mismatch += 1
                        pairs[(tuple(actual[j:j+4]), tuple(expected), (xx//24, yy//24) in visible)] += 1
            assert mismatch == 0, (row['step'], mode, mismatch, pairs.most_common(5))
            if row['label'] == '最多同時效果' and mode == 'screen':
                # 省略所有特效：獨立候選基底不能冒充已驗重疊成果。
                negative_difference = sum(actual[j:j+4] != hd_base[j:j+4] for yy in range(480, 528) for xx in range(120, 744) if (xx//24, yy//24) in visible for j in [4*(yy*960+xx)])
        observations.append({'step': row['step'], 'complete': metadata['complete'], 'matched_cells': len(visible), 'battle_region_mismatch': 0})
    assert negative_difference > 0
    with output.open('x') as f:
        json.dump({'scope': '21 個原型畫格的戰鬥區域；兩種混色、8×8 遮格、未知格原版回退。中途來源推定引用 Go，未獨立證明唯一性；未驗戰鬥區外、中文、美術或正式前端', 'inputs_sha256': inputs, 'archived_inputs': archived_inputs, 'observations': observations, 'omit_effects_negative_pixels': negative_difference}, f, ensure_ascii=False, indent=2)
        f.write('\n')
    print(f'獨立合成：{len(observations)} 幀 × 2 種混色，戰鬥區不符 0；省略特效負對照差 {negative_difference}')


if __name__ == '__main__':
    main()
