"""研究038 §102：正式視窗完整27筆圖面及中文字面獨立核對；不排除畫面區域。"""
import json
from pathlib import Path
import sys

from verify_ally_items import assets_from_files, art_plane
from verify_over_runtime import read, load_font, render_text
from verify_over_frontend import compose, mismatch
from verify_next_body_plane import region, rgba, require, sha, FW, FH


def main():
    root = Path(sys.argv[1])
    output = root / 'verified.json'
    require(not output.exists(), '拒絕覆寫核對收據')
    inputs = {}
    execution = json.loads(read(root / 'execution.json', inputs))
    terminal = json.loads(read(root / 'terminal.json', inputs))
    export = json.loads(read(root / 'original-frames.json', inputs))
    for doc in (execution, export):
        for path, expected in doc['inputs_sha256'].items():
            data = read(path, inputs)
            if sha(data) != expected and path == '/src/tools/hd/gui_ally_items_run.py':
                candidates = list(root.parent.glob('gui-run-v*-source.py'))
                preserved = next((p for p in candidates if sha(p.read_bytes()) == expected), None)
                require(preserved is not None, '沒有相符的執行工具來源副本')
                data = read(preserved, inputs)
            require(sha(data) == expected, '輸入已變：' + path)
    require(terminal['terminal'] and terminal['returncode'] == 0, '正式前端未正常退出')
    captured = {r['label']: r for r in terminal['actions'] if r['kind'] == 'captured'}
    originals = {Path(r['state']).stem: r for r in export['results']}
    assets, decoded = assets_from_files(inputs)
    fonts = {n: load_font('font/' + n + '.golemfnt', inputs) for n in ('cjk24', 'cjk16')}
    formal = {}
    for path in sorted(Path('text').glob('*.json')):
        document = json.loads(read(path, inputs))
        if isinstance(document, dict):
            for entry in document.get('entries', []):
                if 'key' in entry and 'translation' in entry:
                    require(entry['key'] not in formal or formal[entry['key']] == entry['translation'], '正式中文鍵不唯一')
                    formal[entry['key']] = entry['translation']
    help_data = json.loads(read('text/help.json', inputs))
    gear = int(execution['argv'][execution['argv'].index('-speed') + 1])
    read('apps/psychicwar/help.go', inputs)
    read('apps/psychicwar/speed.go', inputs)
    read('cmd/psychicwar/main.go', inputs)

    def with_badge(pixels):
        # 023 §7：正式速度字樣、24格、HelpCols38；完整畫面含前端標記。
        if gear == 1:
            return pixels
        message = help_data['speed_gear'] % gear
        result = bytearray(pixels)
        cell, box_w = 24, (len(message) + 1) * 24
        for y in range(cell):
            begin = 4 * y * FW
            result[begin:begin + box_w * 4] = bytes((0, 0, 0, 255)) * box_w
        w, h, glyphs = fonts['cjk24']
        pen = 0
        for character in message:
            x0 = (FW - 38 * cell) // 2 + pen * cell // 2
            pen += 1 if ord(character) < 128 else 2
            glyph = glyphs.get(character)
            if glyph is None:
                continue
            for y in range(h):
                for x in range(w):
                    if x0 + x < box_w and y < cell and glyph[y * ((w + 7) // 8) + x // 8] & (0x80 >> (x % 8)):
                        at = 4 * (y * FW + x0 + x)
                        result[at:at + 4] = bytes((85, 255, 85, 255))
        return result
    empty = bytes(FW * FH * 4)
    rows = []
    for label, capture in captured.items():
        prefix = root / label
        frame = read(str(prefix) + '.state.frame', inputs)
        rgb = read(str(prefix) + '.state.rgb.bin', inputs)
        player = read(str(prefix) + '.state.player.bin', inputs)
        original = originals[label]
        require(sha(frame) == original['frame_sha256'] and sha(rgb) == original['rgb_sha256'] and sha(player) == original['player_sha256'], '原版零步匯出已變')
        side = json.loads(read(str(prefix) + '.state.xlate.json', inputs))
        for stamp in side['stamps']:
            if stamp.get('text'):
                require(stamp['key'] in formal and stamp['text'].strip() in [line.strip() for line in formal[stamp['key']].split('\n')], '中文字面不在正式資料：' + stamp['key'])
        meta = json.loads(read(str(prefix) + '.meta.json', inputs))
        require(meta['language'] == capture['language'], '實際語言中繼資料不符')
        chinese = meta['language'] == 'zh'
        art, cells = art_plane(frame, assets)
        text = render_text(side, fonts) if chinese else empty
        expected = with_badge(compose(rgb, (art if capture['hd'] else empty, text)))
        phases = sorted(root.glob(label + '-phase*.png'))
        require(len(phases) == 12, '不是十二份實際視窗相位')
        results = []
        for path in phases:
            read(path, inputs)
            w, h, pixels = rgba(path)
            require((w, h) == (FW, FH), '視窗尺寸不符')
            results.append(dict(png=str(path), mismatch=mismatch(expected, pixels)))
        selected = next((r for r in results if r['mismatch'] == 0), None)
        omit, _ = art_plane(frame, assets, omit_top=True)
        negative = mismatch(expected, with_badge(compose(rgb, (omit, text)))) if capture['hd'] else 0
        top_complete = region(frame, 128, 8, 24, 32) == decoded['ALLY.PBL', 0][2]
        if selected and top_complete and capture['hd']:
            require(negative > 0 and {tuple(c[:2]) for c in cells} == {(128, 8), (264, 152)}, '兩位置HD或省略新肖像負對照無效')
        altered = bytearray(expected)
        altered[0] ^= 1
        require(mismatch(expected, altered) == 1, '單像素負對照無效')
        rows.append(dict(label=label, hd=capture['hd'], language=meta['language'], top_complete=top_complete, ally_cells=cells, verified=selected is not None, selected_png=selected['png'] if selected else None, composition_mismatch=0 if selected else min(r['mismatch'] for r in results), negative_omit_top_pixels=negative, phases=results, original=original))
        print(label, '完整視窗差0' if selected else '未通過，最少差' + str(rows[-1]['composition_mismatch']), flush=True)
    read('tools/hd/verify_gui_ally_items.py', inputs)
    read('tools/hd/gui_ally_items_run.py', inputs)
    incomplete = sorted(set(originals) - set(captured))
    output.write_text(json.dumps(dict(scope='正常checkpoint後正式Ebiten視窗與F10零步匯出之獨立完整合成；未排除畫面區域', inputs_sha256=inputs, results=rows, verified_count=sum(r['verified'] for r in rows), total=len(rows) + len(incomplete), incomplete=incomplete, limits='十二相位至少一份完整吻合才通過；不等同每相位、全程、原版DAT或正式封包'), ensure_ascii=False, indent=2) + '\n')
    require(not incomplete, '有未完成的視窗擷取，收據已保存')
    require(all(r['verified'] for r in rows), '正式視窗完整畫面仍未通過，收據已保存')
    require(any(r['hd'] and r['language'] == 'en' and r['top_complete'] for r in rows) and any(r['hd'] and r['language'] == 'zh' and r['top_complete'] for r in rows), '缺兩語新肖像HD完整樣本')


if __name__ == '__main__':
    main()
