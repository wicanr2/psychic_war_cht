"""研究038 §101：實際pwstep道具頁與Forget it後，兩語HD／原版及舊26筆回歸。"""
import hashlib
import json
from pathlib import Path
import subprocess


def main():
    root = Path('workplace/hd/ally-items-v1-20261004')
    output = root / 'pwstep-v1'
    assert not output.exists(), '拒絕覆寫'
    binary, exporter = root / 'pwstep-v1.bin', root / 'export-v1.bin'
    assert binary.is_file() and exporter.is_file()
    output.mkdir()
    inputs = {str(p): hashlib.sha256(p.read_bytes()).hexdigest() for p in (binary, exporter, Path(__file__))}
    rows = []
    for sample, label in ((9, 'items'), (11, 'forget')):
        source = root / f'runtime-v2-sample{sample:02d}-mode2.state'
        side = root / f'runtime-v2-sample{sample:02d}.state.xlate.json'
        # 同一個真正state的前端中文字面；不修改state或遊戲資料。
        target_side = Path(str(source) + '.xlate.json')
        assert not target_side.exists()
        target_side.write_bytes(side.read_bytes())
        for p in (source, target_side):
            inputs[str(p)] = hashlib.sha256(p.read_bytes()).hexdigest()
        for language in ('original', 'chinese'):
            for mode in ('off', 'candidate', 'prior26'):
                name = f'{label}-{language}-{mode}'
                target = output / (name + '.state')
                theme = '' if mode == 'off' else 'workplace/hd/theme-ally-items-v1-20261004' if mode == 'candidate' else 'workplace/hd/theme-kasuruji-pose0-v1-20261003'
                command = [str(binary), '-orig', '/orig/psychic-war', '-load-state', str(source), '-do', 'wait:1', '-scale', '3', '-text', 'text' if language == 'chinese' else '', '-font', 'font', '-theme', theme, '-scratch', str(output / (name + '-scratch')), '-save-state', str(target), '-shot', str(output / (name + '.png')), '-text-log', str(output / (name + '-text.jsonl'))]
                process = subprocess.run(command, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=45)
                (output / (name + '.log')).write_bytes(process.stdout + process.stderr)
                assert process.returncode == 0, (name, process.stderr.decode())
                rows.append(dict(label=label, language=language, mode=mode, state=str(target), png=str(output / (name + '.png')), command=command, exit_code=0))
                print(name, '完成', flush=True)
    process = subprocess.run([str(exporter), '-out', str(output)], stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=45)
    (output / 'export.log').write_bytes(process.stdout + process.stderr)
    assert process.returncode == 0, process.stderr.decode()
    (output / 'execution.json').write_text(json.dumps(dict(inputs_sha256=inputs, results=rows, scope='兩個正常checkpoint接續1ms；十二份實際pwstep保存及兩語PNG，未驗正式Ebiten視窗或原版DAT'), ensure_ascii=False, indent=2) + '\n')


if __name__ == '__main__':
    main()
