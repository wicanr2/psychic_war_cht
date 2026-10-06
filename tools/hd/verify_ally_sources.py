"""研究038 §100：正常ALLY来源的獨立原始資料核對，不編修圖片。"""
import base64
import hashlib
import json
from pathlib import Path

from explore_sprite_sources import strict_pbl, crop


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def main():
    root = Path('/src')
    out = root / 'workplace/ida/hd-ally-portrait-20261004'
    target = out / 'ally-sources-independent-20261004.json'
    assert not target.exists(), '拒絕覆寫'
    archive = Path('/orig/psychic-war/ALLY.PBL')
    poses = strict_pbl(archive.read_bytes())
    assert len(poses) == 31
    inputs = {str(archive): sha(archive)}
    rows = []
    labels = [('name-v3',42000000,8,[(264,152)]),
              ('ask',58000000,4,[]),('items',58000000,6,[(128,8)]),
              ('esp-v2',66000000,8,[])]
    for label, step, irq, positions in labels:
        prefix = out / f'ally-sources-{label}-20261004'
        data = json.loads(prefix.with_suffix('.json').read_text())
        model_path = Path(str(prefix)+'-machine.json')
        model = json.loads(model_path.read_text())
        frame = Path(str(prefix)+'-end.frame').read_bytes()
        assert len(model) == 44 and model['Steps'] == step
        assert data['observed_end'] == data['control_end']
        assert data['full_saved_state_equal'] and data['register_negative_control']
        assert data['observed_end']['IRQ1'] == irq
        assert data['seed_before'] == '86AF'
        assert [(r['X'],r['Y']) for r in data['ally_draws']] == positions
        for name, expected in data['inputs_sha256'].items():
            path = Path(name)
            if not path.is_absolute():
                path = root / path
            assert sha(path) == expected, name
            inputs[str(path)] = expected
        draws = []
        for draw in data['ally_draws']:
            path = out / Path(draw['File']).name
            raw = path.read_bytes()
            assert sha(path) == draw['PackedSHA256'] and len(raw) == 384
            decoded = bytes(c for b in raw for c in (b>>4,b&15))
            matches = [i for i,w,h,p,o,end in poses
                       if (w,h,p) == (draw['W'],draw['H'],decoded)]
            assert matches == draw['Matches'] == [0]
            actual = crop(frame, draw['X'],draw['Y'],draw['W'],draw['H'])
            assert actual == decoded, '完整原版終點肖像不同'
            bad = bytearray(decoded)
            bad[0] ^= 1
            assert sum(a!=b for a,b in zip(actual,bad)) == 1
            context = draw['Context']
            assert context['AX']&255 == 0 and context['CS'] == 0x161
            assert context['IP'] == 0x8705 and context['DX'] == 0x0304
            assert context['DS'] == 0x1175
            inputs[str(path)] = sha(path)
            draws.append({'index':0,'position':[draw['X'],draw['Y']],
                          'pixels_compared':768,'mismatch':0,'negative_mismatch':1})
        for load in data['ally_loads']:
            assert load['AX'] == 0x0c00
            assert load['Stack'][0] == 0x2cee
        if label == 'name-v3':
            reference = root/'workplace/states/07-first-play.frame'
            assert frame == reference.read_bytes()
            inputs[str(reference)] = sha(reference)
        else:
            replay = out/f'native-{label}-end-20261004.frame'
            assert frame == replay.read_bytes(), '載回原版一指令後畫面不同'
            inputs[str(replay)] = sha(replay)
        for path in [prefix.with_suffix('.json'),model_path,
                     Path(str(prefix)+'-end.frame'),
                     Path(str(prefix)+'-observed.state'),Path(str(prefix)+'-control.state')]:
            inputs[str(path)] = sha(path)
        rows.append({'route':label,'end_step':step,'irq1':irq,'ally_loads':len(data['ally_loads']),
                     'ally_draws':draws,'machine_fields_equal':44,'dos_section_equal':True,
                     'limits':'只證實本正常鍵序，未觸發#30不表示未使用'})
    wrapper_path = out/'normal-load-wrapper.json'
    wrapper = json.loads(wrapper_path.read_text())
    memory = base64.b64decode(json.loads((out/'ally-sources-name-v3-20261004-machine.json').read_text())['Mem'])
    # IDA seg002 base0x10510、runtime CS0161分別列出，逐bytes核對。
    static = bytes.fromhex(wrapper['span_bytes'])
    assert memory[0x1610+0x2ce3:0x1610+0x2ce3+len(static)] == static
    call = next(r for r in wrapper['decoded_instructions'] if r['ida_ea']=='0x131fb')
    assert call['bytes']=='e87859' and call['decoded_mnemonic']=='call'
    assert call['decoded_operands'][0]['addr']=='0x8666'
    inputs[str(wrapper_path)] = sha(wrapper_path)
    for path in [Path(__file__),root/'tools/hd/explore_sprite_sources.py',
                 root/'tools/ida/ally_portrait.py',out/'machine-schema.go']:
        inputs[str(path)] = sha(path)
    result = {'scope':'正常開局及三項選單的原版ALLY來源，不是HD或GUI驗收',
              'inputs_sha256':inputs,'archive_count':31,'routes':rows,
              'loader_wrapper':{'IDA_span':['0x131f3','0x13204'],
                                'runtime_span':['0161:2ce3','0161:2cf4'],
                                'bytes_equal':True,'actual_return':'0161:2cee'},
              'next_gate':'ALLY #0道具肖像的新位置及顯示生命週期契約；#30用途仍未知'}
    target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print('四條正常路線：44機器欄位及DOS相同；ALLY #0兩位置各768像素差0，負對照差1')


if __name__ == '__main__':
    main()
