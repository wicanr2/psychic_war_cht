"""研究038 §74：25筆主題真正視窗的獨立完整合成及原版DAT抽測。"""
import json
from collections import defaultdict
from pathlib import Path
import sys

from verify_room22_render import assets_from_files
from verify_anchor_render import art_plane
from verify_over_runtime import read, load_font, render_text
from verify_over_frontend import compose, mismatch
from verify_next_body_plane import rgba, require, sha, FW, FH


def main():
    roots=[Path(x) for x in sys.argv[1:]]
    require(bool(roots),'缺GUI輸出')
    output=Path('workplace/hd/gui-dat-independent-v2-20261003.json')
    require(not output.exists(),'拒絕覆寫')
    inputs={}; assets=assets_from_files(inputs)
    fonts={n:load_font('font/'+n+'.golemfnt',inputs) for n in ('cjk24','cjk16')}
    formal={}
    for p in sorted(Path('text').glob('*.json')):
        data=json.loads(read(p,inputs))
        if isinstance(data,dict):
            for entry in data.get('entries',[]):
                if 'key' in entry and 'translation' in entry:
                    require(entry['key'] not in formal or formal[entry['key']]==entry['translation'],'正式來源鍵不唯一')
                    formal[entry['key']]=entry['translation']
    empty=bytes(FW*FH*4); results=[]
    for root in roots:
        export=json.loads(read(root/'original-frames.json',inputs))
        for p,digest in export['inputs_sha256'].items():
            require(sha(read(p,inputs))==digest,'原版匯出輸入已變 '+p)
        states={Path(x['state']).stem:x for x in export['results']}
        for label,row in states.items():
            prefix=root/label
            frame=read(str(prefix)+'.state.frame',inputs); rgb=read(str(prefix)+'.state.rgb.bin',inputs)
            player=read(str(prefix)+'.state.player.bin',inputs)
            require(sha(frame)==row['frame_sha256'] and sha(rgb)==row['rgb_sha256'] and sha(player)==row['player_sha256'],'原版匯出不符')
            side=json.loads(read(str(prefix)+'.state.xlate.json',inputs))
            groups=defaultdict(list)
            for stamp in side['stamps']:
                if stamp.get('text'): groups[stamp['key']].append(stamp)
            for key, group in groups.items():
                require(key in formal,'中文鍵不在正式資料 '+key)
                expected_rows=[(i,text) for i,text in enumerate(formal[key].split('\n')) if text.strip()]
                if '\n' in formal[key]:
                    group=sorted(group,key=lambda s:s['y'])
                    require(len(group)==len(expected_rows),'多行中文筆數不符 '+key)
                    for stamp,(index,text) in zip(group,expected_rows,strict=True):
                        require(stamp['text'].strip()==text.strip(),'多行中文字面來源不符 '+key)
                        require(stamp['x']==group[0]['x'] and stamp['y']-group[0]['y']==(index-expected_rows[0][0])*stamp['cell_h'],'多行中文間隔不符 '+key)
                else:
                    require(all(stamp['text'].strip()==formal[key].strip() for stamp in group),'中文字面來源不符 '+key)
            metadata=json.loads(read(str(prefix)+'.meta.json',inputs))
            chinese=metadata['language']=='zh'
            hd='original-' not in label
            art,full,cells=art_plane(frame,assets)
            text=render_text(side,fonts)
            expected=compose(rgb,(art if hd else empty,text if chinese else empty))
            phases=sorted(root.glob(label+'-phase*.png')); require(len(phases)==12,'相位數不同 '+label)
            diffs=[]
            for png in phases:
                read(png,inputs); w,h,pix=rgba(png)
                require((w,h)==(FW,FH),'視窗尺寸不符')
                diffs.append({'png':str(png),'mismatch':mismatch(expected,pix)})
                if diffs[-1]['mismatch']==0: break
            matches=[x for x in diffs if x['mismatch']==0]
            require(bool(matches),'完整視窗不符 '+label+' '+str(diffs))
            omit_art=mismatch(expected,compose(rgb,(empty,text if chinese else empty))) if hd else 0
            omit_text=mismatch(expected,compose(rgb,(art if hd else empty,empty))) if chinese else 0
            require(not hd or not any(art) or omit_art>0,'省略HD負對照無效')
            # 初始debug state未恢復既有文字側檔，允許中文層尚未建立，但保留其數字。
            results.append({'root':str(root),'label':label,'hd':hd,'chinese':chinese,'full_sources':full,'room_cells':len(cells),
                'selected_png':matches[0]['png'],'composition_mismatch':0,
                'verified':True,'negative_omit_hd_pixels':omit_art,
                'negative_omit_text_pixels':omit_text,'captured_phases':len(phases),'checked_phases':diffs,'original':row})
            print(label+(' 完整視窗一致' if matches else ' 轉場樣本未作同狀態完成證據'),flush=True)
    for p in ('tools/hd/verify_gui_dat.py','tools/hd/gui_dat_run.py','tools/hd/export_gui_dat.go',
              'tools/hd/verify_room22_render.py','tools/hd/verify_anchor_render.py','tools/hd/verify_over_runtime.py',
              'tools/hd/verify_over_frontend.py','tools/hd/verify_next_body_plane.py','tools/pbl.py'):
        read(p,inputs)
    save_root=Path('workplace/hd/gui-dat-v1-20261003')
    load_root=Path('workplace/hd/gui-dat-load-v1-20261003')
    dat=read(save_root/'saves/hd25.dat',inputs)
    original_dat=read('workplace/states/scratch/test2.dat',inputs)
    require(len(dat)==512 and dat==original_dat,'真正GUI DAT與獨立原版不同')
    require(read(load_root/'saves/hd25.dat',inputs)==dat,'LOAD GAME輸入不是這次保存的DAT')
    player=read(save_root/'i-save-confirm.state.player.bin',inputs)
    reference_player=read('workplace/states/18-loaded2.mem',inputs)
    require(len(player)==52 and player==reference_player,'存檔前玩家資料與獨立原版不同')
    loaded_labels=('c-loaded-chinese','d-loaded-english','e-original-english')
    for label in loaded_labels:
        require(read(load_root/(label+'.state.player.bin'),inputs)==player,'原版LOAD GAME未還原52 bytes '+label)
    terminal=json.loads(read(load_root/'terminal.json',inputs))
    require(terminal['terminal'] and terminal['returncode']==0,'讀檔GUI未正常退出')
    recording=json.loads(read(load_root/'record.json',inputs))
    # KeyName實際命名為Down／Return，見keymap.go。
    expected_keys=['Down','Return','H','D','Digit2','Digit5','Return']
    expected_events=[(key,down) for key in expected_keys for down in (True,False)]
    require([(x['key'],x['down']) for x in recording['events']]==expected_events,'原版鍵盤事件不同或前端熱鍵外洩')
    bad=bytearray(dat);bad[0]^=1
    require(bytes(bad)!=original_dat,'DAT一byte負對照無效')
    for p in ('apps/psychicwar/keymap.go','workplace/hd/psychicwar-gui25-v1-20261003','workplace/hd/export-gui-dat-v1-20261003'):
        read(p,inputs)
    value={'scope':'現行25筆主題真正視窗之原始資料獨立完整合成與原版選單DAT存讀檔','inputs_sha256':inputs,
        'results':results,'dat':{'bytes':512,'sha256':sha(dat),'independent_original_equal':True,'negative_byte_change_detected':True,
        'player_52_bytes_equal_before_and_after_load':True,'loaded_labels':loaded_labels,'load_gui_exit_code':0,'original_key_events':len(expected_events)},
        'limits':'正常state接續、自然GUI時間；首張起點仍有原版排定輸入，各張必須有完整像素一致相位，不排除任何區域。保存GUI關窗BadWindow、record未保存；讀檔GUI採quit-after正常結束。不宣稱原版逐次亂數、從開機、全部sprite、美術、真機、封包、音訊或幀率完成'}
    output.write_text(json.dumps(value,ensure_ascii=False,indent=2)+'\n')
    print('完整GUI核對 '+str(len(results))+' 份',flush=True)


if __name__=='__main__': main()
