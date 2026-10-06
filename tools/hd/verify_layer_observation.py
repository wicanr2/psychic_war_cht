"""核對受控圖層蒐證；通過只表示觀察可重現，不表示 HD 圖層完成。

    tools/py.sh tools/hd/verify_layer_observation.py
    tools/py.sh tools/hd/verify_layer_observation.py --require-hd-contract

來源重生入口：docs/re/038 §23。第二個命令檢查尚未實作的恢復要求，
目前應非零結束；不得將它當成現有遊戲產品故障。
"""
import argparse
import hashlib
import json
from pathlib import Path
import sys

def require(condition, reason):
    if not condition:
        raise ValueError(reason)

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def check_backlinks(project):
    evidence_path = project / 'docs/re/038-hd-theme-feasibility.md'
    spec_path = project / 'docs/spec/024-hd-theme.md'
    plane_path = project / 'worktrees/dosgolem/docs/spec/204-art-plane.md'
    earlier, current = evidence_path.read_text(encoding='utf-8').split('## 23. ', 1)
    marker = '【HD-LAYER-01】'
    require(marker in earlier and marker in current, '研究缺少圖層契約的前後回查')
    for path in (spec_path, plane_path):
        require(marker in path.read_text(encoding='utf-8'), '草案缺圖層契約訂正：' + str(path))
    for symbol in ('(*Layer).Frame', '(*Stamp).setTransparent', '(*Layer).alive', 'checkWatchers'):
        require(symbol in current, '證據缺原始 Go 符號：' + symbol)
    return [evidence_path, spec_path, plane_path]

def verify(probe, project):
    backlinks = check_backlinks(project)
    source = probe / 'hd-layer-recovery-observation.json'
    doc = json.loads(source.read_text(encoding='utf-8'))
    require(doc['scale'] == 3 and doc['dosgolem_version'] == 'f8c1a6e', '倍率或工具版本不符')
    for path, expected in doc['files_sha256'].items():
        require(sha(Path(path)) == expected, '蒐證輸入雜湊已變：' + path)
    baseline = bytes(2 if i in (0, 8, 128, 136) else 1 for i in range(256))
    require((probe / 'hd-layer-baseline.idx').read_bytes() == baseline, '人工基準矩陣不符')
    rows = {}
    files = [source, probe / 'hd-layer-baseline.idx', Path(__file__)] + backlinks
    expected_cases = {'ordinary_partial_restore', 'watcher_multirow_restore',
                      'watcher_single_row_restore', 'partial_match_late_baseline'}
    require({x['name'] for x in doc['observations']} == expected_cases, '受控案例集合不符')
    for item in doc['observations']:
        name = item['name']
        phases = {}
        for phase in item['phases']:
            label = phase['name']
            idx = Path(phase['input']).read_bytes()
            rgba = Path(phase['rgba']).read_bytes()
            require(len(idx) == 256 and len(rgba) == 48*48*4, '色號或 RGBA 尺寸不符')
            expected_input = bytearray(baseline)
            if label == 'obscured':
                expected_input[0] = 3
            elif label == 'late_admission':
                for y in range(8, 16):
                    for x in range(8):
                        expected_input[y*16+x] = 3
            require(idx == expected_input, '人工階段輸入不符：' + name + '/' + label)
            missing = overdraw = visible = 0
            for y in range(48):
                for x in range(48):
                    alpha = rgba[4*(y*48+x)+3]
                    visible += alpha != 0
                    full_rows = name in ('watcher_multirow_restore', 'partial_match_late_baseline')
                    expected_covered = full_rows or y < 24
                    current_expected = expected_covered
                    if name == 'ordinary_partial_restore' and label in ('obscured', 'restored'):
                        current_expected = y < 24 and x >= 24
                    if name == 'watcher_multirow_restore' and label in ('obscured', 'restored', 'snapshot_restored'):
                        current_expected = y >= 24
                    if name == 'watcher_single_row_restore' and label == 'obscured':
                        current_expected = False
                    require(alpha == (255 if current_expected else 0),
                            f'實際繪圖遮罩不符既有行為：{name}/{label} ({x},{y})')
                    if label in ('restored', 'snapshot_restored'):
                        missing += expected_covered and alpha == 0
                    if alpha and idx[(y//3)*16+x//3] != baseline[(y//3)*16+x//3]:
                        overdraw += 1
            phases[label] = {'visible_scaled_pixels': visible,
                             'missing_after_baseline_restore': missing,
                             'covered_nonbaseline_pixels': overdraw,
                             'reported_rows': phase['active_rows'], 'make_calls': phase['make_calls']}
            files += [Path(phase[key]) for key in ('input', 'rgba', 'snapshot')]
        rows[name] = phases
    require(rows['ordinary_partial_restore']['restored']['missing_after_baseline_restore'] == 576,
            '普通疊字未恢復行為已變，須重新審查')
    require(rows['watcher_multirow_restore']['restored']['missing_after_baseline_restore'] == 1152,
            '多列缺列行為已變，須重新審查')
    require(rows['watcher_multirow_restore']['snapshot_restored']['missing_after_baseline_restore'] == 1152,
            '快照後的缺列行為已變，須重新審查')
    require(rows['watcher_single_row_restore']['restored']['missing_after_baseline_restore'] == 0,
            '單列正對照無法恢復')
    require(rows['partial_match_late_baseline']['late_admission']['covered_nonbaseline_pixels'] == 576,
            '晚加入的未驗證覆蓋行為已變，須重新審查')
    return {'status': '受控蒐證已核對；不是 HD 實作驗收或原版 parity',
            'source_basis': '16×16 人工色號矩陣及既有 xlate API，非原版 EXE／PBL／玩家路徑',
            'observations': rows,
            'hd_contract_gate': '不通過：格恢復、多列恢復、首次完整基準尚缺正式契約與實作',
            'backlink': '【HD-LAYER-01】；研究 §18→§23、024 §9、dosgolem 204 §2.6',
            'limits': '未實作 Art／Pix，未選定遮罩粒度，未測實際遊戲或效能',
            'files_sha256': {str(path): sha(path) for path in files}}

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--probe-dir', type=Path, default=Path('workplace/probe'))
    parser.add_argument('--output', type=Path, default=Path('workplace/probe/hd-layer-recovery-verification.json'))
    parser.add_argument('--project-root', type=Path, default=Path(__file__).resolve().parents[2],
                        help='回查文件所在的專案根目錄；反向對照可指定獨立副本')
    parser.add_argument('--require-hd-contract', action='store_true')
    args = parser.parse_args()
    try:
        receipt = verify(args.probe_dir, args.project_root)
    except (OSError, ValueError, KeyError) as exc:
        print('蒐證核對失敗：' + str(exc), file=sys.stderr)
        return 2
    args.output.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(json.dumps(receipt['observations'], ensure_ascii=False))
    if args.require_hd_contract:
        print('HD 草案契約尚未通過；受控缺口已重現，沒有正式圖面實作。', file=sys.stderr)
        return 1
    print('受控觀察核對通過；HD 契約仍未通過。')
    return 0

if __name__ == '__main__':
    raise SystemExit(main())
