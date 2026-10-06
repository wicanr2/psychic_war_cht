"""獨立核對正式 204 的正常保存幀輸出；入口 docs/re/038 §32。

在 Docker 中執行。本工具不呼叫 Go 圖面或把成果截圖當答案。
"""
import argparse
import hashlib
import json
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import pbl

ROOT = Path(__file__).resolve().parents[2]
W, H, S = 320, 200, 3
FW, FH = W*S, H*S


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def rgba(path):
    w,h,ch,rows,palette = pbl.read_png(path)
    assert (w,h)==(FW,FH) and ch in (3,4) and palette is None, path
    out=bytearray()
    for row in rows:
        for x in range(w):
            out.extend(row[ch*x:ch*x+3])
            out.append(row[ch*x+3] if ch==4 else 255)
    return out


def pixels_differ(a,b):
    assert len(a)==len(b)==FW*FH*4
    return sum(a[i:i+4]!=b[i:i+4] for i in range(0,len(a),4))


def cells(frame,base):
    return {(x,y) for y in range(0,H,8) for x in range(0,W,8)
            if all(frame[yy*W+x:yy*W+x+8]==base[yy*W+x:yy*W+x+8] for yy in range(y,y+8))}


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--receipt',required=True)
    parser.add_argument('--output',required=True,help='新收據，不覆寫')
    parser.add_argument('--tests',default='workplace/hd/art-plane-tests-20261001.jsonl')
    parser.add_argument('--benchmark',default='workplace/hd/art-plane-benchmark-20261001.txt')
    parser.add_argument('--source-snapshot',help='核對舊版時使用明確且逐檔雜湊相符的來源快照 manifest')
    args=parser.parse_args()
    result=ROOT/args.output
    assert result.parent.is_dir() and not result.exists(), '輸出已存在或目錄不存在'
    receipt_path=ROOT/args.receipt
    receipt=json.loads(receipt_path.read_text())
    snapshot=json.loads((ROOT/args.source_snapshot).read_text())['files'] if args.source_snapshot else {}
    archived_sources={}
    assert receipt['cell']==[8,8] and len(receipt['phases'])==6
    for key in ('inputs_sha256','outputs_sha256'):
        for name,expected in receipt[key].items():
            if sha(ROOT/name)!=expected:
                entry=snapshot.get(name)
                assert key=='inputs_sha256' and entry and entry['sha256']==expected and sha(ROOT/entry['snapshot'])==expected, '來源或輸出雜湊不符：'+name
                archived_sources[name]=entry
    old_path=ROOT/'workplace/hd/redraw/normal-recovery-20261001.json'
    old=json.loads(old_path.read_text())
    for name,expected in old['inputs_sha256'].items():
        assert sha(ROOT/name)==expected, '正常路徑來源已變：'+name
    base=(ROOT/'workplace/hd/bg.idx').read_bytes()
    reconstructed=bytearray(W*H)
    for name,count in (('SCREEN',5),('MENU',1)):
        data=(ROOT/f'workplace/original/psychic-war/{name}.PBL').read_bytes()
        images=list(pbl.images(data));assert len(images)==count
        for index,off,_ in images:
            sw,sh,px=pbl.decode(data,off)
            x,y=(0,index*40) if name=='SCREEN' else (160,4)
            for yy in range(sh):
                start=(y+yy)*W+x
                reconstructed[start:start+sw]=bytes(px[yy*sw:(yy+1)*sw])
    assert reconstructed==base, '原版背景重建不符'
    hd=rgba(ROOT/'workplace/hd/redraw/layout-v2-20261001-background.png')
    observations=[]
    by_branch={'escape':[],'control':[]}
    final_expected=None
    for phase in receipt['phases']:
        assert phase['rows']==25 and phase['make_calls']==1, '多列重複或缺列'
        frame=(ROOT/phase['input']).read_bytes();assert len(frame)==W*H
        assert frame[:W*40]==base[:W*40], '本批整組啟用區不相符'
        visible=cells(frame,base)
        expected_art=bytearray(FW*FH*4)
        for x,y in visible:
            for yy in range(y*S,(y+8)*S):
                start=4*(yy*FW+x*S);end=start+8*S*4
                expected_art[start:end]=hd[start:end]
        actual_art=(ROOT/phase['art']).read_bytes()
        art_bad=pixels_differ(actual_art,expected_art)
        assert art_bad==0, (phase['scene'],'圖面不符',art_bad)
        prefix=ROOT/'workplace/hd/redraw'/phase['scene']
        original=rgba(Path(str(prefix)+'-original.png'))
        chinese=rgba(Path(str(prefix)+'-chinese.png'))
        overlay=rgba(ROOT/phase['overlay'])
        assert set(overlay[3::4]) <= {0,255}, '本批中文不是已驗證的不透明覆繪'
        expected=bytearray(original)
        for i in range(0,len(expected),4):
            if expected_art[i+3]:expected[i:i+4]=expected_art[i:i+4]
            if overlay[i+3]:expected[i:i+4]=overlay[i:i+4]
        actual=(ROOT/phase['composed']).read_bytes()
        combined_bad=pixels_differ(actual,expected)
        assert combined_bad==0, (phase['scene'],'合成不符',combined_bad)
        assert rgba(ROOT/phase['png'])==actual, 'PNG 不等於實際 RGBA'
        chinese_bad=dynamic_bad=0
        for yy in range(FH):
            for xx in range(FW):
                i=4*(yy*FW+xx)
                if overlay[i+3]==255:
                    chinese_bad+=actual[i:i+4]!=overlay[i:i+4]
                if (xx//24*8,yy//24*8) not in visible:
                    dynamic_bad+=actual[i:i+4]!=chinese[i:i+4]
        assert chinese_bad==dynamic_bad==0
        by_branch[phase['branch']].append(visible)
        observations.append({'branch':phase['branch'],'scene':phase['scene'],'visible_cells':len(visible),
                             'art_mismatch':art_bad,'composed_mismatch':combined_bad,
                             'chinese_mismatch':chinese_bad,'dynamic_mismatch':dynamic_bad})
        if phase['scene']=='replay-escape-settled-20261001':final_expected=expected
    restored=by_branch['escape'][-1]-by_branch['escape'][0]
    control_restored=by_branch['control'][-1]-by_branch['control'][0]
    nonblack=sum(base[yy*W+xx]!=0 for x,y in restored for yy in range(y,y+8) for xx in range(x,x+8))
    assert nonblack==47 and len(restored)*64==1216 and not control_restored
    negative=(ROOT/receipt['negative']).read_bytes()
    negative_bad=pixels_differ(negative,final_expected)
    assert negative_bad==423, ('永久遮格反向對照未檢出已知恢復缺口',negative_bad)
    tests_path=ROOT/args.tests
    tests=[json.loads(s) for s in tests_path.read_text().splitlines()]
    assert not any(v['Action']=='fail' for v in tests)
    assert any(v['Action']=='pass' and v.get('Package')=='github.com/wicanr2/dosgolem/xlate' and 'Test' not in v for v in tests)
    passed=sum(v['Action']=='pass' and 'Test' in v for v in tests)
    benchmark=ROOT/args.benchmark
    assert 'PASS' in benchmark.read_text()
    output={'status':'正式 204 圖面之有限驗證通過；尚未驗收正式主題、前端與完整 sprite',
            'go_receipt':args.receipt,'go_receipt_sha256':sha(receipt_path),
            'verifier_sha256':sha(Path(__file__)),'pbl_background_mismatch':0,
            'observations':observations,'restored_original_pixels':len(restored)*64,
            'restored_nonblack_background_pixels':nonblack,'control_restored_original_pixels':len(control_restored)*64,
            'permanent_hole_negative_mismatch':negative_bad,'test_and_subtest_passes':passed,
            'test_log_sha256':sha(tests_path),'benchmark_sha256':sha(benchmark),
            'archived_sources':archived_sources,
            'limits':'本批正常幀來源沿用研究 §29；只讀保存畫面，不執行新玩家重播，不宣稱完整機器狀態或跨平台完成。'}
    with result.open('x',encoding='utf-8') as f:
        json.dump(output,f,ensure_ascii=False,indent=2);f.write('\n')
    print(json.dumps({'normal_frames':len(observations),'nonblack_restored':nonblack,
                      'negative_mismatch':negative_bad,'tests_and_subtests_passed':passed},ensure_ascii=False))


if __name__=='__main__':
    main()
