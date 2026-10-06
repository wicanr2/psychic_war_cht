"""研究038 §73：讀兩份原型清單，獨立驗證資料表示、格邊與全域8×8遮罩。"""
import copy
import hashlib
import json
from pathlib import Path

from mask import read_png
from explore_sprite_sources import strict_pbl


ROOT=Path('/src')
OUT=ROOT/'workplace/hd/maze-theme-prototype-v2-20261003'


def png(path):
    w,h,ch,rows=read_png(path)
    assert ch==4
    return w,h,b''.join(rows)


def load(folder,manifest):
    assert manifest['schema']=='psychic-war-maze-prototype/1'
    assert manifest['source']=='MAZE.BIN' and manifest['scale']==3
    assert manifest['kind']=='geometry-candidate'
    def image(name):
        p=Path(name)
        assert not p.is_absolute() and '..' not in p.parts
        p=(folder/p).resolve();assert p.is_relative_to(folder.resolve())
        return png(p)
    if manifest['storage']=='atlas':
        assert set(manifest)=={'schema','source','scale','kind','limits','storage','png','columns','rows','slot_count'}
        assert (manifest['columns'],manifest['rows'],manifest['slot_count'])==(16,16,256)
        w,h,data=image(manifest['png']);assert (w,h)==(192,192)
        return [b''.join(data[((i//16*12+y)*192+i%16*12)*4:((i//16*12+y)*192+i%16*12+12)*4] for y in range(12)) for i in range(256)]
    assert manifest['storage']=='files'
    assert set(manifest)=={'schema','source','scale','kind','limits','storage','tiles'}
    assert len(manifest['tiles'])==256
    result=[None]*256
    for entry in manifest['tiles']:
        assert set(entry)=={'slot','png'} and type(entry['slot']) is int
        slot=entry['slot'];assert 0<=slot<256 and result[slot] is None
        w,h,data=image(entry['png']);assert (w,h)==(12,12)
        result[slot]=data
    assert all(p is not None for p in result)
    return result


def unpack(b):
    return bytes(v for c in b for v in (c>>4,c&15))


def reference(base,slots,maze):
    value=bytearray(base)
    for i,slot in enumerate(slots):
        tile=unpack(maze[slot*8:slot*8+8]);x,y=4+i%18*4,124+i//18*4
        for row in range(4):
            value[(y+row)*320+x:(y+row)*320+x+4]=tile[row*4:row*4+4]
    return value


def padded(slots,tiles):
    plane=bytearray(240*240*4)
    for i,slot in enumerate(slots):
        x,y=12+i%18*12,12+i//18*12;tile=tiles[slot]
        for row in range(12):
            at=((y+row)*240+x)*4;plane[at:at+48]=tile[row*48:row*48+48]
    return plane


def masked(frame,ref,plane):
    result=bytearray(len(plane));valid=[]
    for col in range(10):
        for row in range(10):
            x,y=col*8,120+row*8
            actual=b''.join(frame[(y+i)*320+x:(y+i)*320+x+8] for i in range(8))
            want=b''.join(ref[(y+i)*320+x:(y+i)*320+x+8] for i in range(8))
            if actual==want:
                valid.append((x,y))
                for line in range(24):
                    at=((row*24+line)*240+col*24)*4
                    result[at:at+96]=plane[at:at+96]
    return result,sorted(valid)


def mismatch(a,b):
    assert len(a)==len(b)
    return sum(a[i:i+4]!=b[i:i+4] for i in range(0,len(a),4))


def main():
    target=OUT/'verification.json';assert not target.exists()
    inputs={}
    def read(path):
        value=path.read_bytes();inputs[str(path)]=hashlib.sha256(value).hexdigest();return value
    read(Path(__file__))
    a=json.loads(read(OUT/'atlas-manifest.json'));b=json.loads(read(OUT/'files-manifest.json'))
    atlas=load(OUT,a);files=load(OUT,b);assert atlas==files
    for p in [OUT/'MAZE-atlas.png']+sorted((OUT/'tiles').glob('*.png')):read(p)
    receipt=json.loads(read(OUT/'receipt.json'))
    for name,expected in receipt['inputs_sha256'].items():
        assert hashlib.sha256(read(Path(name))).hexdigest()==expected
    maze=read(Path('/orig/psychic-war/MAZE.BIN'))
    pal=read(ROOT/'workplace/hd/room-nearby-explore-v1-20261003-bbs-step65999999.pal')
    palette=[tuple(pal[i*3:i*3+3]) for i in range(16)]
    for slot,tile in enumerate(atlas):
        original=unpack(maze[slot*8:slot*8+8])
        for y in range(12):
            for x in range(12):
                at=(y*12+x)*4;assert tile[at+3]==255
                if x in (0,11) or y in (0,11):
                    assert tile[at:at+4]==bytes((*palette[original[y//3*4+x//3]],255))
    negatives={}
    for label,edited in [('missing_slot',copy.deepcopy(b)),('duplicate_slot',copy.deepcopy(b)),('wrong_atlas_layout',copy.deepcopy(a)),('path_escape',copy.deepcopy(a))]:
        if label=='missing_slot':edited['tiles'].pop()
        if label=='duplicate_slot':edited['tiles'][1]['slot']=0
        if label=='wrong_atlas_layout':edited['columns']=15
        if label=='path_escape':edited['png']='../MAZE-atlas.png'
        try:
            load(OUT,edited)
        except (AssertionError,FileNotFoundError):
            negatives[label]=True
        else:
            raise AssertionError('負對照無效：'+label)
    base=bytearray(64000)
    room_sources=[]
    for name in ['SCREEN.PBL','MENU.PBL','ROOM0.PBL']:
        archive=strict_pbl(read(Path('/orig/psychic-war')/name))
        if name=='ROOM0.PBL':
            room_sources=[(i,pixels) for i,w,h,pixels,_,_ in archive if i in [0,2,3,8,22]]
            continue
        for i,w,h,pixels,_,_ in archive:
            x,y=(160,4) if name=='MENU.PBL' else (0,i*40)
            for row in range(h):base[(y+row)*320+x:(y+row)*320+x+w]=pixels[row*w:(row+1)*w]
    catalog=json.loads(read(ROOT/'workplace/hd/maze-table-verification-v1-20261003.json'))
    samples=[]
    for sample in catalog['samples']:
        frame=read(Path(sample['frame']));slots=sample['slots']
        ref=reference(base,slots,maze);pa=padded(slots,atlas);pf=padded(slots,files)
        output,cells=masked(frame,ref,pa);other,other_cells=masked(frame,ref,pf)
        assert output==other and cells==other_cells
        w,h,actual=png(OUT/f'{sample["label"]}-plane.png');assert (w,h)==(240,240)
        assert output==actual
        original=b''.join(frame[y*320+4:y*320+76] for y in range(124,196))
        overlaps=[i for i,px in room_sources if px==original]
        changed=bytearray(frame);changed[120*320]^=1
        hidden,changed_cells=masked(changed,ref,pa)
        assert len(changed_cells)==99 and (0,120) not in changed_cells
        assert mismatch(hidden,output)==144
        restored,restored_cells=masked(frame,ref,pa);assert restored==output and len(restored_cells)==100
        assert all(output[(y*240+x)*4+3]==0 for y in range(240) for x in range(240) if not (12<=x<228 and 12<=y<228))
        negatives['padding_pixel_invalidates_global_cell']=144
        samples.append({'label':sample['label'],'valid_cells':len(cells),'plane_mismatch':0,'room0_content_overlaps':overlaps,'invalidated_cell':[0,120],'restored_cells':len(restored_cells)})
    # 648份實際貼圖後畫面；兩輪source表由原版AL序列導出，非偽造中途幀。
    table=catalog['draw_cycles'];draw=json.loads(read(ROOT/'workplace/hd/maze-draw-source-v2-20261003.json'))
    stream_path=Path(draw['frame_stream']);inputs[str(stream_path)]=draw['frame_stream_sha256']
    contexts=[(reference(base,g['slots'],maze),padded(g['slots'],atlas),padded(g['slots'],files)) for g in table]
    traces=[]
    with stream_path.open('rb') as stream:
        for index,event in enumerate(draw['events']):
            stream.seek(event['FrameOffset']+64000);frame=stream.read(64000);assert len(frame)==64000
            ref,pa,pf=contexts[index//324]
            left,cells=masked(frame,ref,pa);right,others=masked(frame,ref,pf)
            assert left==right and cells==others
            traces.append({'event':index,'step':event['Return'],'valid_cells':len(cells),'atlas_files_mismatch':0})
    assert hashlib.sha256(stream_path.read_bytes()).hexdigest()==draw['frame_stream_sha256']
    result={'scope':'兩份實際原型清單／256PNG／atlas，原始格邊及8×8遮罩；不是正式執行期驗收','inputs_sha256':inputs,'slot_count':256,'opaque_tiles':256,'edge_mismatch':0,'samples':samples,'actual_after_frames':traces,'negative_controls':negatives,'limits':'只驗三份正常狀態和648個原始after frame；未驗正式載入器、GUI、其他場景有效條件或ROOM優先序。藝術候選未定版。'}
    target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print('256格／兩份清單／648份after畫面相同；格邊及全域8×8遮罩核對通過；負對照有效。')


if __name__=='__main__':main()
