"""研究038 §73：原始位址／證據等級與較早ROOM22規格回填護欄。"""
import hashlib
import json
from pathlib import Path


def main():
    root=Path('/src');out=root/'workplace/hd/maze-room-backlink-v1-20261003.json'
    assert not out.exists()
    evidence=root/'docs/re/038-hd-theme-feasibility.md'
    consumer=root/'docs/spec/024-hd-theme.md'
    receipt=root/'workplace/hd/maze-room-overlap-v1-20261003.json'
    e=evidence.read_text();s=consumer.read_text();r=json.loads(receipt.read_text())
    binary_sha='88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49'
    assert r['inputs_sha256']['/orig/psychic-war/PW.EXE']==binary_sha
    midpoint=r['samples'][1]
    assert midpoint['step']==615210000 and midpoint['runtime_pc']==[0x161,0x5002]
    assert midpoint['maze_row_return_pc'] and midpoint['table_room0_matches']==[22]
    assert sum(x['maze_view_mismatch']==0 and x['actual_room0_matches']==[22] for x in r['samples'])==8
    older=s[s.index('### 1.12'):s.index('### 1.13')]
    marker='MAZE／ROOM0 #22內容別名'
    assert marker in older and '0161:5002' in older
    assert 'confirmed' in older and '強推論' in older
    assert marker in e and binary_sha in e and 'runtime0161:5002' in e
    negative_missing_marker=marker not in older.replace(marker,'')
    assert negative_missing_marker
    value={'scope':'原始鍵與§1.12回填；不驗正式MAZE優先序',
           'binary_sha256':binary_sha,'runtime_address':'0161:5002',
           'confirmed':'615210000步MAZE第一列返回，表目標與ROOM0 #22相同',
           'strong_inference':'本路線完整#22可由MAZE繪製產生',
           'evidence':str(evidence.relative_to(root)),
           'consumer':str(consumer.relative_to(root))+' §1.12',
           'required_marker':marker,'negative_missing_marker':negative_missing_marker,
           'inputs_sha256':{str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in [Path(__file__),evidence,consumer,receipt]}}
    out.write_text(json.dumps(value,ensure_ascii=False,indent=2)+'\n')
    print('原始0161:5002／confirmed與強推論分級／§1.12回填護欄通過。')


if __name__=='__main__':main()
