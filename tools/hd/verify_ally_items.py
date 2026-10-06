"""研究038 §101：只讀原版PBL、來源事件、PNG及字型，獨立核對完整27筆圖面。"""
import json
from pathlib import Path

from explore_sprite_sources import strict_pbl
from verify_next_body_plane import W, H, S, FW, FH, region, rgba, over, require, sha
from verify_over_runtime import read, load_font, render_text
from verify_over_frontend import compose, mismatch

OUT = Path('workplace/hd/ally-items-v1-20261004')
THEME = Path('workplace/hd/theme-ally-items-v1-20261004')


def assets_from_files(inputs):
    manifest = json.loads(read(THEME / 'manifest.json', inputs))
    require(len(manifest['entries']) == 27, '不是27筆主題')
    base, decoded = bytearray(W * H), {}
    for name in sorted({e['pbl'] for e in manifest['entries']}):
        for n, w, h, pixels, _, _ in strict_pbl(read(Path('/orig/psychic-war') / name, inputs)):
            decoded[name, n] = w, h, pixels
            if name in ('SCREEN.PBL', 'MENU.PBL'):
                x, y = (0, n * 40) if name == 'SCREEN.PBL' else (160, 4)
                for yy in range(h):
                    base[(y + yy) * W + x:(y + yy) * W + x + w] = pixels[yy * w:(yy + 1) * w]
    assets = []
    for entry in manifest['entries']:
        name = entry['pbl']
        w, h, source = decoded[name, entry['image']]
        path = THEME / entry['png']
        read(path, inputs)
        pw, ph, pixels = rgba(path)
        require((pw, ph) == (w * S, h * S), 'PNG尺寸不同')
        reference = bytearray(W * H) if name == 'OVER.PBL' else bytearray(base)
        if name not in ('SCREEN.PBL', 'MENU.PBL'):
            x, y = entry['at']
            for yy in range(h):
                reference[(y + yy) * W + x:(y + yy) * W + x + w] = source[yy * w:(yy + 1) * w]
        assets.append((entry, w, h, source, pixels, reference))
    return assets, decoded


def art_plane(frame, assets, active_positions=(), omit_top=False):
    require(len(frame) == W * H, '原版frame尺寸不同')
    plane, cells = bytearray(FW * FH * 4), []
    for entry, w, h, source, pixels, reference in assets:
        name = entry['pbl']
        x, y = entry['at']
        identity = (name, entry['image'], x, y)
        if omit_top and identity == ('ALLY.PBL', 0, 128, 8):
            continue
        anchor = entry.get('match', [128, 48, 64, 64] if name == 'OVER.PBL' else [4, 124, 72, 72] if name == 'ROOM0.PBL' else [0, 0, 320, 40])
        mx, my, mw, mh = anchor
        if region(frame, mx, my, mw, mh) != region(reference, mx, my, mw, mh):
            continue
        if name not in ('SCREEN.PBL', 'MENU.PBL') and region(frame, x, y, w, h) != source and identity not in active_positions:
            continue
        for cy in range(y // 8 * 8, (y + h + 7) // 8 * 8, 8):
            for cx in range(x // 8 * 8, (x + w + 7) // 8 * 8, 8):
                ref = region(reference, cx, cy, 8, 8)
                if region(frame, cx, cy, 8, 8) != ref:
                    continue
                ink = region(reference, max(x, cx), max(y, cy), min(x + w, cx + 8) - max(x, cx), min(y + h, cy + 8) - max(y, cy)) if name == 'ROOM0.PBL' else ref
                if name not in ('SCREEN.PBL', 'MENU.PBL') and not any(ink):
                    continue
                if name == 'ALLY.PBL':
                    cells.append([x, y, cx, cy])
                for yy in range(max(y, cy) * S, min(y + h, cy + 8) * S):
                    for xx in range(max(x, cx) * S, min(x + w, cx + 8) * S):
                        at = 4 * ((yy - y * S) * w * S + xx - x * S)
                        over(plane, 4 * (yy * FW + xx), pixels[at:at + 4])
    return plane, cells


def main():
    output = OUT / 'independent-v2.json'
    require(not output.exists(), '拒絕覆寫核對收據')
    inputs = {}
    doc = json.loads(read(OUT / 'runtime-v2.json', inputs))
    for p, h in doc['inputs_sha256'].items():
        require(sha(read(p, inputs)) == h, '執行輸入已變更：' + p)
    require(doc['irq1'] == 20 and doc['entries'] == 27 and len(doc['samples']) == 13, '正常路線未閉合')
    assets, decoded = assets_from_files(inputs)
    source = decoded['ALLY.PBL', 0][2]
    entries = [e for e in doc['events'] if e['event'] == 'entry']
    require(len(entries) == 1 and len(doc['events']) == 2, '新位置來源／返回不是唯一一組')
    event = entries[0]
    packed = read(event['source'], inputs)
    require(sha(packed) == event['sha256'] and bytes(c for b in packed for c in (b >> 4, b & 15)) == source, '原版新位置來源不是ALLY #0')
    require(event['regs']['AX'] & 255 == 0 and event['regs']['CX'] == 0x2002 and event['regs']['DX'] == 0x0304, '原版位置或貼圖模式不同')
    bad = bytearray(packed)
    bad[0] ^= 1
    require(bytes(c for b in bad for c in (b >> 4, b & 15)) != source, '錯來源負對照無效')
    fonts = {n: load_font('font/' + n + '.golemfnt', inputs) for n in ('cjk24', 'cjk16')}
    rows = []
    top_negatives = 0
    for row in doc['samples']:
        require(row['machine_fields'] == 44 and row['dos_equal'] and row['chinese_equal'], '完整狀態或中文不符')
        for phase in ('normal', 'reload', 'reload-next'):
            prefix = row['prefix'] + ('' if phase == 'normal' else '-' + phase)
            frame = read(prefix + '.frame', inputs)
            if phase == 'normal':
                require(sha(frame) == row['original']['Frame'], '原版取樣frame不符')
            active = {('ALLY.PBL', 0, 128, 8)} if phase == 'normal' and row['step'] > event['step'] else set()
            expected, cells = art_plane(frame, assets, active)
            actual = read(prefix + '-plane.rgba', inputs)
            delta = mismatch(actual, expected)
            require(delta == 0, f'完整27筆圖面不符：{prefix}，差{delta}')
            side = json.loads(read(prefix + '.state.xlate.json', inputs))
            text = render_text(side, fonts)
            require(text == read(prefix + '-text.rgba', inputs), '正式字型獨立圖面不符：' + prefix)
            rgb = read(prefix + '-rgb.bin', inputs)
            for language, layers in (('english', (expected,)), ('chinese', (expected, text))):
                path = prefix + '-' + language + '.png'
                read(path, inputs)
                pw, ph, png = rgba(path)
                require((pw, ph) == (FW, FH) and png == compose(rgb, layers), '獨立全屏合成不符：' + path)
            omitted, _ = art_plane(frame, assets, active, omit_top=True)
            negative = mismatch(expected, omitted)
            if any(c[:2] == [128, 8] for c in cells):
                require(negative > 0, '省略肖像負對照無效')
                top_negatives += 1
            if row['step'] in (58000000, 72000000) and phase == 'normal':
                require(region(frame, 128, 8, 24, 32) == source and region(frame, 264, 152, 24, 32) == source, '道具或Forget it後兩位置不完整')
                require({tuple(c[:2]) for c in cells} == {(128, 8), (264, 152)}, '兩位置HD未同時保留')
            altered = bytearray(expected)
            altered[0] ^= 1
            require(mismatch(expected, altered) == 1, '單像素負對照無效')
            rows.append(dict(step=row['step'], phase=phase, ally_cells=cells, plane_mismatch=0, text_mismatch=0, english_composition_mismatch=0, chinese_composition_mismatch=0, negative_omit_top_pixels=negative))
            print(prefix, '完整圖面／兩語合成差0', flush=True)
    require(top_negatives > 0, '沒有新肖像HD負對照')
    json_out = dict(scope='13份正常取樣及26份真正載回／接續；完整27筆圖面、正式字型與兩語合成，不是GUI驗收', samples=rows, entry_step=event['step'], return_step=doc['events'][1]['step'], inputs_sha256=inputs, limits='新主題未選入；GUI、原版DAT、其他sprite及完整交付未完成')
    output.write_text(json.dumps(json_out, ensure_ascii=False, indent=2) + '\n')


if __name__ == '__main__':
    main()
