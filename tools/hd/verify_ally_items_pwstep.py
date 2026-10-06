"""研究038 §101：十二份pwstep的原版來源、中文與完整兩語圖面獨立核對。"""
import json
from pathlib import Path

from verify_ally_items import assets_from_files, art_plane, OUT, THEME
from verify_next_body_plane import FW, FH, region, rgba, require, sha
from verify_over_runtime import read, load_font, render_text
from verify_over_frontend import compose, mismatch


def main():
    output = OUT / 'pwstep-independent-v1.json'
    require(not output.exists(), '拒絕覆寫')
    inputs = {}
    run = json.loads(read(OUT / 'pwstep-v1/execution.json', inputs))
    export = json.loads(read(OUT / 'pwstep-v1/original-frames.json', inputs))
    for doc in (run, export):
        for path, expected in doc['inputs_sha256'].items():
            require(sha(read(path, inputs)) == expected, '輸入已變更：' + path)
    require(len(run['results']) == len(export['results']) == 12, '不是十二份前端產物')
    assets, decoded = assets_from_files(inputs)
    prior_root = Path('workplace/hd/theme-kasuruji-pose0-v1-20261003')
    prior = json.loads(read(prior_root / 'manifest.json', inputs))
    require(len(prior['entries']) == 26, '舊主題筆數不同')
    prior_assets = []
    for entry, current in zip(prior['entries'], assets[:26], strict=True):
        require({k: v for k, v in entry.items() if k != 'match'} == {k: v for k, v in current[0].items() if k != 'match'}, '舊主題位置／來源已變')
        require(read(prior_root / entry['png'], inputs) == read(THEME / entry['png'], inputs), 'PNG已變')
        prior_assets.append((entry, *current[1:]))
    exported = {r['state']: r for r in export['results']}
    fonts = {n: load_font('font/' + n + '.golemfnt', inputs) for n in ('cjk24', 'cjk16')}
    empty = bytes(FW * FH * 4)
    controls, text_controls, rows = {}, {}, []
    for row in run['results']:
        state, label = row['state'], row['label']
        read(state, inputs)
        fields = {k: v for k, v in exported[state].items() if k != 'state'}
        require(label not in controls or controls[label] == fields, '前端兩語或主題改变原版狀態')
        controls[label] = fields
        frame, rgb = read(state + '.frame', inputs), read(state + '.rgb.bin', inputs)
        require(sha(frame) == fields['frame_sha256'] and sha(rgb) == fields['rgb_sha256'], '原版匯出已變')
        require(region(frame, 128, 8, 24, 32) == decoded['ALLY.PBL', 0][2], '道具肖像來源不完整')
        art, cells = art_plane(frame, prior_assets if row['mode'] == 'prior26' else assets)
        text = empty
        if row['language'] == 'chinese':
            data = read(state + '.xlate.json', inputs)
            require(label not in text_controls or text_controls[label] == data, '主題改變中文快照')
            text_controls[label] = data
            side = json.loads(data)
            require(any(st['state'] == 2 and any('\u4e00' <= c <= '\u9fff' for c in st['text']) for st in side['stamps']), '中文模式沒有可見中文')
            text = render_text(side, fonts)
        enabled = row['mode'] != 'off'
        expected = compose(rgb, (art if enabled else empty, text))
        read(row['png'], inputs)
        w, h, actual = rgba(row['png'])
        delta = mismatch(expected, actual)
        require((w, h) == (FW, FH) and delta == 0, f'前端獨立全屏不符：{row["png"]}，差{delta}')
        negative = 0
        if row['mode'] == 'candidate':
            omit, _ = art_plane(frame, assets, omit_top=True)
            negative = mismatch(expected, compose(rgb, (omit, text)))
            require(negative > 0 and {tuple(c[:2]) for c in cells} == {(128, 8), (264, 152)}, '兩位置或省略肖像負對照無效')
        rows.append(dict(label=label, language=row['language'], mode=row['mode'], composition_mismatch=0, original_fields_equal=True, negative_omit_top_pixels=negative))
        print(row['png'], '獨立全屏差0', flush=True)
    output.write_text(json.dumps(dict(results=rows, inputs_sha256=inputs, limits='兩個正常checkpoint接續的pwstep；不是Ebiten實際視窗、原版DAT或封包驗收'), ensure_ascii=False, indent=2) + '\n')


if __name__ == '__main__':
    main()
