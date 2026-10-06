"""研究038 §73：可丟棄MAZE資料表示與幾何重繪原型，不修改正式載入器。"""
import hashlib
import json
import math
from pathlib import Path

from render import fill, write_png
from mask import read_png
from verify_maze_draw import unpack
from explore_sprite_sources import strict_pbl


ROOT = Path('/src')
OUT = ROOT/'workplace/hd/maze-theme-prototype-v2-20261003'
SCALE = 3


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def distance(p,a,b):
    vx,vy=b[0]-a[0],b[1]-a[1]
    if vx==vy==0:
        return math.hypot(p[0]-a[0],p[1]-a[1])
    t=max(0,min(1,((p[0]-a[0])*vx+(p[1]-a[1])*vy)/(vx*vx+vy*vy)))
    return math.hypot(p[0]-a[0]-t*vx,p[1]-a[1]-t*vy)


def simplify(points):
    if len(points)<3:
        return points
    distances=[distance(p,points[0],points[-1]) for p in points[1:-1]]
    largest=max(distances)
    if largest<=0.51:
        return [points[0],points[-1]]
    split=distances.index(largest)+1
    return simplify(points[:split+1])[:-1]+simplify(points[split:])


def loops(tile,color):
    # 由色區建立順時針邊，孔洞與外框用偶奇填色合成。
    edges=set()
    for y in range(4):
        for x in range(4):
            if tile[y*4+x]!=color:
                continue
            p=[(x,y),(x+1,y),(x+1,y+1),(x,y+1)]
            for a,b in zip(p,p[1:]+p[:1]):
                if (b,a) in edges:
                    edges.remove((b,a))
                else:
                    edges.add((a,b))
    paths=[]
    directions={(1,0):0,(0,1):1,(-1,0):2,(0,-1):3}
    while edges:
        a,b=min(edges); edges.remove((a,b)); points=[a,b]
        while b!=points[0]:
            choices=[end for start,end in edges if start==b]
            assert choices,'輪廓未閉合'
            old=directions[(b[0]-a[0],b[1]-a[1])]
            # 在對角接觸點優先右轉，保持像素色區的獨立輪廓。
            rank={1:0,0:1,3:2,2:3}
            end=min(choices,key=lambda p:rank[(directions[(p[0]-b[0],p[1]-b[1])]-old)%4])
            edges.remove((b,end));a,b=b,end;points.append(b)
        paths.append(points)
    return paths


def redraw(tile,palette):
    rgba=[0]*(12*12*4)
    colors=sorted(set(tile))
    geometry=[]
    changed_paths=0
    for color in colors:
        combined=bytearray(48*48)
        for path in loops(tile,color):
            # 以碰到圖塊外緣的頂點為固定錨點，格內折線才簡化。
            anchors=[i for i,p in enumerate(path[:-1]) if 0 in p or 4 in p]
            if not anchors:
                anchors=[0,len(path[:-1])//2]
            ring=path[:-1]; poly=[]
            for a,b in zip(anchors,anchors[1:]+[anchors[0]+len(ring)]):
                points=[ring[i%len(ring)] for i in range(a,b+1)]
                reduced=simplify(points)
                changed_paths+=int(len(reduced)<len(points))
                poly.extend(reduced[:-1])
            if len(poly)<3:
                poly=ring
            coverage=bytearray(48*48)
            fill(coverage,12,12,poly,3)
            for i,v in enumerate(coverage):
                combined[i]^=v
            geometry.append({'color':color,'points':poly})
        for y in range(12):
            for x in range(12):
                n=sum(combined[(y*4+dy)*48+x*4:(y*4+dy)*48+x*4+4].count(1) for dy in range(4))
                if n==0:
                    continue
                # 每種色區的覆蓋率來自向量多邊形，不對原PNG做濾鏡。
                i=(y*12+x)*4
                for channel in range(3):
                    rgba[i+channel]+=round(palette[color][channel]*n/16)
                rgba[i+3]+=round(255*n/16)
    for y in range(12):
        for x in range(12):
            i=(y*12+x)*4
            # 外緣一個HD像素保持原色，防止相鄰4×4格出現接合縫。
            if x in (0,11) or y in (0,11):
                rgba[i:i+4]=bytes((*palette[tile[(y//3)*4+x//3]],255))
            else:
                # 多色輪廓簡化可能交疊；原型保留透明度總量與色區限制。
                total=rgba[i+3]
                if total==0:
                    rgba[i:i+4]=bytes((*palette[tile[(y//3)*4+x//3]],255))
                elif total!=255:
                    for ch in range(3):
                        rgba[i+ch]=min(255,round(rgba[i+ch]*255/total))
                    rgba[i+3]=255
    return bytearray(rgba),geometry,changed_paths


def rgba_png(path):
    w,h,ch,rows=read_png(path)
    assert ch==4
    return w,h,b''.join(rows)


def scene(slots,tiles):
    rgba=bytearray(216*216*4)
    for row in range(18):
        for col in range(18):
            tile=tiles[slots[row*18+col]]
            for y in range(12):
                at=((row*12+y)*216+col*12)*4
                rgba[at:at+48]=tile[y*48:y*48+48]
    return rgba


def make_base(orig,inputs):
    value=bytearray(64000)
    for name,n in [('SCREEN.PBL',5),('MENU.PBL',1)]:
        p=orig/name; inputs[str(p)]=sha(p); archive=strict_pbl(p.read_bytes())
        assert len(archive)==n
        for item in archive:
            index,w,h,pixels,_,_=item
            x,y=(160,4) if name=='MENU.PBL' else (0,index*40)
            for row in range(h):
                value[(y+row)*320+x:(y+row)*320+x+w]=pixels[row*w:(row+1)*w]
    return value


def mask(frame,base,slots,art):
    reference=bytearray(base)
    maze=mask.maze
    for row in range(18):
        for col in range(18):
            tile=unpack(maze[slots[row*18+col]*8:slots[row*18+col]*8+8])
            for y in range(4):
                at=(124+row*4+y)*320+4+col*4
                reference[at:at+4]=tile[y*4:y*4+4]
    output=bytearray(240*240*4)
    valid=[]
    for gy in range(120,200,8):
        for gx in range(0,80,8):
            if any(frame[(gy+y)*320+gx:(gy+y)*320+gx+8]!=reference[(gy+y)*320+gx:(gy+y)*320+gx+8] for y in range(8)):
                continue
            valid.append([gx,gy])
            for y in range(gy,max(gy,min(gy+8,196))):
                for x in range(max(gx,4),min(gx+8,76)):
                    if y<124:
                        continue
                    for sy in range(3):
                        source=((y-124)*3+sy)*216*4+(x-4)*12
                        dest=((y-120)*3+sy)*240*4+x*12
                        output[dest:dest+12]=art[source:source+12]
    return output,valid


def original_rgba(frame,palette,scale=3):
    data=bytearray(320*scale*200*scale*4)
    for y in range(200):
        row=b''.join(bytes((*palette[c],255))*scale for c in frame[y*320:y*320+320])
        for sy in range(scale):
            at=(y*scale+sy)*320*scale*4;data[at:at+len(row)]=row
    return data


def main():
    assert not OUT.exists(),'拒絕覆寫'
    assert OUT.parent.stat().st_uid==OUT.parent.stat().st_gid==1000
    orig=Path('/orig/psychic-war')
    inputs={}
    for p in [Path(__file__),ROOT/'tools/hd/render.py',ROOT/'tools/hd/mask.py',ROOT/'tools/hd/explore_sprite_sources.py',ROOT/'tools/hd/verify_maze_draw.py']:
        inputs[str(p)]=sha(p)
    maze_path=orig/'MAZE.BIN';maze=maze_path.read_bytes();inputs[str(maze_path)]=sha(maze_path)
    assert inputs[str(maze_path)]=='8b3b08ef483ff8c64beb04c69c861763c1258afba2348a7058e828bcf4060756'
    catalog_path=ROOT/'workplace/hd/maze-table-verification-v1-20261003.json'
    inputs[str(catalog_path)]=sha(catalog_path);catalog=json.loads(catalog_path.read_text())
    palette_path=ROOT/'workplace/hd/room-nearby-explore-v1-20261003-bbs-step65999999.pal'
    inputs[str(palette_path)]=sha(palette_path);pal=palette_path.read_bytes();assert len(pal)==768
    palette=[tuple(pal[i*3:i*3+3]) for i in range(16)]
    base=make_base(orig,inputs);mask.maze=maze
    OUT.mkdir();(OUT/'tiles').mkdir()
    tiles=[];geometry=[];stats=[]
    atlas=bytearray(192*192*4)
    for slot in range(256):
        source=unpack(maze[slot*8:slot*8+8]);tile,paths,changes=redraw(source,palette)
        tiles.append(tile);geometry.append({'slot':slot,'paths':paths})
        write_png(OUT/f'tiles/MAZE-{slot:03d}.png',12,12,tile)
        for y in range(12):
            at=((slot//16*12+y)*192+slot%16*12)*4;atlas[at:at+48]=tile[y*48:y*48+48]
        original=b''.join(bytes((*palette[source[y//3*4+x//3]],255)) for y in range(12) for x in range(12))
        stats.append({'slot':slot,'source_sha256':hashlib.sha256(source).hexdigest(),'vector_paths':len(paths),'simplified_runs':changes,'changed_pixels':sum(tile[i:i+4]!=original[i:i+4] for i in range(0,len(tile),4))})
    write_png(OUT/'MAZE-atlas.png',192,192,atlas)
    common={'schema':'psychic-war-maze-prototype/1','source':'MAZE.BIN','scale':3,'kind':'geometry-candidate','limits':'研究原型；正式載入器拒絕此schema'}
    atlas_manifest={**common,'storage':'atlas','png':'MAZE-atlas.png','columns':16,'rows':16,'slot_count':256}
    files_manifest={**common,'storage':'files','tiles':[{'slot':i,'png':f'tiles/MAZE-{i:03d}.png'} for i in range(256)]}
    for name,value in [('atlas-manifest.json',atlas_manifest),('files-manifest.json',files_manifest),('geometry.json',geometry)]:
        (OUT/name).write_text(json.dumps(value,ensure_ascii=False,indent=2)+'\n')
    # 實際從兩份PNG資料讀回，不從生成器中的tile陣列比自身。
    aw,ah,decoded_atlas=rgba_png(OUT/'MAZE-atlas.png');assert (aw,ah)==(192,192)
    from_atlas=[];from_files=[]
    for i in range(256):
        from_atlas.append(b''.join(decoded_atlas[((i//16*12+y)*192+i%16*12)*4:((i//16*12+y)*192+i%16*12+12)*4] for y in range(12)))
        w,h,decoded=rgba_png(OUT/f'tiles/MAZE-{i:03d}.png');assert (w,h)==(12,12);from_files.append(decoded)
    assert from_atlas==from_files
    samples=[]
    for sample in catalog['samples']:
        frame_path=Path(sample['frame']);inputs[str(frame_path)]=sha(frame_path);frame=frame_path.read_bytes()
        slots=sample['slots'];one=scene(slots,from_atlas);two=scene(slots,from_files);assert one==two
        plane,valid=mask(frame,base,slots,one)
        old=original_rgba(frame,palette);new=bytearray(old)
        for y in range(240):
            for x in range(240):
                at=(y*240+x)*4
                if plane[at+3]:
                    dest=((360+y)*960+x)*4;new[dest:dest+4]=plane[at:at+4]
        label=sample['label'];write_png(OUT/f'{label}-original.png',960,600,old);write_png(OUT/f'{label}-prototype.png',960,600,new)
        write_png(OUT/f'{label}-plane.png',240,240,plane)
        # 三倍視野並排，左為原始，右為幾何候選，無文字烘入玩家圖面。
        left=b''.join(old[((372+y)*960+12)*4:((372+y)*960+228)*4] for y in range(216))
        right=b''.join(new[((372+y)*960+12)*4:((372+y)*960+228)*4] for y in range(216))
        pair=b''.join(left[y*864:(y+1)*864]+right[y*864:(y+1)*864] for y in range(216))
        write_png(OUT/f'{label}-comparison.png',432,216,pair)
        samples.append({'label':label,'frame':str(frame_path),'valid_cells':valid,'atlas_files_mismatch':0,'changed_pixels':sum(old[i:i+4]!=new[i:i+4] for i in range(0,len(old),4))})
    (OUT/'receipt.json').write_text(json.dumps({'scope':'單張圖集／256PNG的可丟棄資料表示與幾何候選','inputs_sha256':inputs,'palette':palette,'tiles':stats,'samples':samples,'limits':'尚未驗正式載入器、其他場景、動作、ROOM優先序或正式美術；多色輪廓簡化可能改變內部色區，需要目視審查。'},ensure_ascii=False,indent=2)+'\n')
    print('原型已生成256格；兩種PNG存法讀回相同；改變',sum(s['changed_pixels']>0 for s in stats),'個圖塊；樣本',[(s['label'],len(s['valid_cells']),s['changed_pixels']) for s in samples])


if __name__=='__main__':
    main()
