"""敵人16×16小圖塊參照、候選轉檔與來源核對；研究038 §53，Docker限定。"""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import sys
import struct
import zlib

sys.path.insert(0, "tools")
import pbl

ROOT = Path("workplace/hd/redraw")
VERSION = "v1-20261002"


def digest(data):
    return hashlib.sha256(data).hexdigest()


def write_json(path, data):
    with path.open("x") as f:
        json.dump(data, f, ensure_ascii=False, indent=2)
        f.write("\n")


def owner(path):
    s = path.stat()
    assert (s.st_uid, s.st_gid) == (1000, 1000), str(path)


def path(name, label):
    return ROOT / f"{name}-small-single-{label}-{VERSION}.json"


def bbox(points):
    assert points, "空圖不作造型猜測"
    return [min(p[0] for p in points), min(p[1] for p in points),
            max(p[0] for p in points)+1, max(p[1] for p in points)+1]


def png_crc(file):
    data = Path(file).read_bytes()
    assert data[:8] == b"\x89PNG\r\n\x1a\n"
    pos = 8
    while pos < len(data):
        n = struct.unpack_from(">I", data, pos)[0]
        end = pos + 12 + n
        assert end <= len(data)
        chunk = data[pos+4:pos+8+n]
        assert zlib.crc32(chunk) & 0xffffffff == struct.unpack_from(">I", data, pos+8+n)[0]
        pos = end
        if chunk[:4] == b"IEND":
            assert n == 0 and pos == len(data)
            return
    raise AssertionError("PNG缺IEND")


def transparent(alpha):
    assert min(alpha) == 0 and max(alpha) > 128


def prepare(name):
    source = Path("/orig/psychic-war") / (name+".PBL")
    data = source.read_bytes()
    entries = pbl.images(data)
    assert len(entries) == 30
    palette = list(pbl.EGA)
    palette[2], palette[8], palette[10] = (170,85,0), (0,0,0), (255,255,85)
    refs, jobs = [], []
    for index in range(15,30):
        _, offset, _ = entries[index]
        w, h, pixels = pbl.decode(data, offset)
        assert (w,h) == (16,16) and not set(pixels)&{2,8,10}
        box = bbox([(i%16,i//16) for i,c in enumerate(pixels) if c])
        reference = ROOT / f"{name}-small-{index}-single-reference-{VERSION}.png"
        assert not reference.exists()
        expanded = bytes(pixels[(y//24)*16+x//24] for y in range(384) for x in range(384))
        pbl.png(reference,384,384,expanded,palette)
        rw,rh,ch,rows,pal = pbl.read_png(reference)
        assert (rw,rh,ch,pal)==(384,384,3,None)
        assert b"".join(rows)==b"".join(bytes(palette[c]) for c in expanded)
        prompt = (
            "Use case: style-transfer. Faithful HD game asset: redraw the ONE attached original 16x16 small graphic tile. "
            "Keep the reference's exact separate and connected colored regions, their proportions and positions. "
            "Use smooth restrained flat cel-colored contours and subtle curves in place of coarse steps. "
            "This is an abstract colored graphic fragment of an existing game sprite, not an invitation to invent a creature, face, weapon or object. "
            "Preserve original colors, black openings and clipped edge contacts. Single square tile, no atlas. "
            f"Exact occupied box in a 16x16 canvas is [left,top,right,bottom]={box}, exclusive right and bottom. "
            f"Preserve transparent margins of left {box[0]}/16, top {box[1]}/16, right {16-box[2]}/16, bottom {16-box[3]}/16. "
            "Regions touching reference edges must touch the same canvas edges. Keep narrow parts narrow and all internal gaps. "
            "Do not center, expand, shrink, move, add padding or fill blank corners. "
            "Reference black becomes transparent empty space, including inner gaps. "
            "No glossy shading, extruded borders, glow, particles, new anatomy, invented details, text, labels, frames or background. "
            "Transparent background. Single square RGBA tile, delivered at 48x48. Faithful redraw of this one frame, no new animation pose."
        )
        if name == "ENEMY03" and index in (24,25,26):
            prompt += " Replace the alternating white/magenta dither region with a continuous pale pink intermediate tone. Keep its outer footprint and broad coral/blue regions; do not turn dither into repeated beads or rounded tiles."
        refs.append({"index":index,"pbl_offset":offset,"source_size":[w,h],"source_bbox":box,
                     "decoded_sha256":digest(pixels),"reference":str(reference),
                     "reference_sha256":digest(reference.read_bytes()),"reference_rgb_mismatch":0})
        jobs.append({"archive":name+".PBL","index":index,"source_bbox":box,"source_size":[16,16],
                     "target_size":[48,48],"refs":["/home/anr2/cht/psychic-war/"+str(reference)],"prompt":prompt})
    write_json(path(name,"references"),{"tool":"Python "+sys.version.split()[0],"source":str(source),
        "source_sha256":digest(data),"image_count":30,"palette":palette,"references":refs,
        "decoder_sha256":digest(Path("tools/pbl.py").read_bytes()),
        "scope":"來源偏移／尺寸／色號confirmed；角色、用途、動畫及原版合成模式未知"})
    write_json(path(name,"generation-jobs"),{"tool":"內建 image_gen","jobs":jobs})
    print(name,"十五張原版參照RGB不符0",[(r["index"],r["source_bbox"]) for r in refs])


def convert(name, generated):
    outputs = json.loads(Path(generated).read_text())["outputs"]
    owner(ROOT)
    for row in outputs:
        index = row["index"]
        assert 15 <= index < 30
        source = Path("/generated") / Path(row["default_path"]).name
        dest = Path(row["project_path"])
        assert dest.parent == ROOT and source.is_file() and not dest.exists()
        shutil.copyfile(source,dest)
        owner(dest)
        small = ROOT / f"{name}-small-{index}-single-48-{VERSION}.png"
        assert not small.exists()
        subprocess.run(["convert",str(dest),"-filter","Lanczos","-resize","48x48!",
                        "-depth","8","PNG32:"+str(small)],check=True)
        owner(small)
        print("已保存",index,digest(dest.read_bytes()))


def verify(name):
    refs = json.loads(path(name,"references").read_text())
    jobs = json.loads(path(name,"generation-jobs").read_text())["jobs"]
    selected_path = path(name,"selected")
    selected = json.loads(selected_path.read_text())["outputs"]
    assert sorted(r["index"] for r in selected)==list(range(15,30)), "批次未完成"
    data = Path(refs["source"]).read_bytes()
    assert digest(data)==refs["source_sha256"]
    assert digest(Path("tools/pbl.py").read_bytes())==refs["decoder_sha256"]
    entries = pbl.images(data)
    files, records, negatives = {}, [], 0
    board = bytearray(720*864*3)
    crc_count = 0
    def record(file):
        file=Path(file);owner(file)
        files[str(file)]={"sha256":digest(file.read_bytes()),"size":file.stat().st_size}
    for slot, r in enumerate(refs["references"]):
        index=r["index"];_,offset,_=entries[index];assert offset==r["pbl_offset"]
        w,h,pixels=pbl.decode(data,offset);assert (w,h)==(16,16) and digest(pixels)==r["decoded_sha256"]
        ref=Path(r["reference"]);record(ref);assert digest(ref.read_bytes())==r["reference_sha256"]
        png_crc(ref);crc_count += 1
        rw,rh,ch,rows,pal=pbl.read_png(ref);assert (rw,rh,ch,pal)==(384,384,3,None)
        rgb=b"".join(rows)
        expected=b"".join(bytes(refs["palette"][pixels[(y//24)*16+x//24]]) for y in range(384) for x in range(384))
        assert rgb==expected
        shifted=b"".join(row[3:]+row[:3] for row in rows)
        negatives+=sum(rgb[i:i+3]!=shifted[i:i+3] for i in range(0,len(rgb),3))
        job=next(j for j in jobs if j["index"]==index)
        assert job["source_bbox"]==r["source_bbox"] and job["refs"]==["/home/anr2/cht/psychic-war/"+str(ref)]
        output=next(s for s in selected if s["index"]==index)
        for file,is_small in ((Path(output["project_path"]),False),
                              (ROOT/f"{name}-small-{index}-single-48-{VERSION}.png",True)):
            record(file);png_crc(file);crc_count += 1
            pw,ph,ch,rows,pal=pbl.read_png(file)
            assert ch==4 and pal is None and pw==ph
            if is_small:assert (pw,ph)==(48,48)
            rgba=b"".join(rows);alpha=rgba[3::4];transparent(alpha)
            if is_small:
                box=bbox([(i%48,i//48) for i,a in enumerate(alpha) if a>=128])
                wanted=[v*3 for v in r["source_bbox"]]
                records.append({"index":index,"source_bbox_x3":wanted,"candidate_bbox":box,
                                "bbox_exact":box==wanted,"transparent_pixels":alpha.count(0),
                                "occupied_mask_different_pixels":sum((a>=128)!=(pixels[(i//48//3)*16+(i%48//3)]!=0) for i,a in enumerate(alpha))})
                # 展示圖採完整畫布；原版像素最近鄰，候選保留alpha並疊黑底。
                for y in range(144):
                    for x in range(144):
                        top=bytes(refs["palette"][pixels[(y//9)*16+x//9]])
                        at=((y//3)*48+x//3)*4
                        bottom=bytes((rgba[at+c]*rgba[at+3]+127)//255 for c in range(3))
                        for row, color in ((0,top),(1,bottom)):
                            dest=(((slot//5*2+row)*144+y)*720+slot%5*144+x)*3
                            board[dest:dest+3]=color
    assert negatives>0
    for file in (path(name,"references"),path(name,"generation-jobs"),selected_path,
                 Path("tools/hd/small_tile_assets.py"),Path("tools/pbl.py")):
        record(file)
    prior=ROOT/"ENEMY02-small-single-source-manifest-v1-20261001.json"
    old=json.loads(prior.read_text())
    for r in old["files"]:assert digest(Path(r["path"]).read_bytes())==r["sha256"]
    record(prior)
    for label, source in (("tool",Path("tools/hd/small_tile_assets.py")),("decoder",Path("tools/pbl.py"))):
        snap=ROOT/f"{name}-small-single-{label}-source-{VERSION}.py"
        with snap.open("xb") as f:f.write(source.read_bytes())
        assert snap.read_bytes()==source.read_bytes();record(snap)
        files[str(source)]["snapshot"]=str(snap)
    comparison=ROOT/f"{name}-small15-single-comparison-{VERSION}.png"
    assert not comparison.exists()
    colors=list(dict.fromkeys(bytes(board[i:i+3]) for i in range(0,len(board),3)))
    indices={color:i for i,color in enumerate(colors)}
    pbl.png(comparison,720,864,[indices[bytes(board[i:i+3])] for i in range(0,len(board),3)],colors)
    png_crc(comparison);crc_count += 1
    bw,bh,ch,rows,pal=pbl.read_png(comparison)
    assert (bw,bh,ch,pal)==(720,864,3,None) and b"".join(rows)==board
    record(comparison)
    try:transparent(bytes([255])*48*48)
    except AssertionError:opaque_rejected=True
    else:raise AssertionError("不透明負對照未被拒絕")
    result={"scope":"十五張單格候選技術核對，不當作正式美術或接入完成",
            "files":files,"frames":records,"source_sha256":refs["source_sha256"],
            "source_reference_rgb_mismatch":0,"reference_shift_negative_pixels":negatives,
            "new_bbox_exact_count":sum(r["bbox_exact"] for r in records),"formal_art_accepted":False,
            "png_crc_and_decode_count":crc_count,"opaque_negative_rejected":opaque_rejected,
            "comparison_rgb_mismatch":0,"python_version":sys.version.split()[0],
            "old_unchanged_files":len(old["files"]),"limits":"用途、動畫、原版透明合成、正常路徑及公開權利未驗"}
    write_json(path(name,"verification"),result)
    record(path(name,"verification"))
    # 全部來源保持本機；保全包括原版以外的參照、原生圖、48px與完整提示詞。
    snapshot={}
    for file,row in files.items():
        snapshot[file]=row
    write_json(path(name,"source-manifest"),{"files":snapshot,"originals_sha256":{refs["source"]:refs["source_sha256"]},
                                            "scope":"非破壞性來源清單，原版未複製、全部留本機"})
    print("十五張參照與30張候選PNG技術核對通過，外框精確",result["new_bbox_exact_count"],"/15；舊",len(old["files"]),"項未變")


if __name__=="__main__":
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command",choices=("prepare","convert","verify"))
    parser.add_argument("name",choices=[f"ENEMY{i:02}" for i in range(12)])
    parser.add_argument("--selected")
    args=parser.parse_args();owner(ROOT)
    if args.command=="prepare":prepare(args.name)
    elif args.command=="convert":convert(args.name,args.selected)
    else:verify(args.name)
