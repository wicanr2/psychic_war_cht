"""研究038 §53.1的兩張ENEMY03候選修整核對；Docker限定，不進正式執行期。"""
import argparse
import json
from pathlib import Path
import shutil
import subprocess
import sys

sys.path.insert(0, "tools/hd")
import small_tile_assets as base

ROOT = base.ROOT
PREFIX = "ENEMY03-small-boundary"
VERSION = "v2-20261002"


def path(label):
    return ROOT / f"{PREFIX}-{label}-{VERSION}.json"


def small(index, version):
    return ROOT / f"ENEMY03-small-{index}-single-48-{version}.png"


def rgba(file):
    base.owner(file)
    base.png_crc(file)
    w, h, ch, rows, pal = base.pbl.read_png(file)
    assert ch == 4 and pal is None and w == h
    raw = b"".join(rows)
    base.transparent(raw[3::4])
    return w, h, raw


def convert():
    rows = json.loads(path("selected").read_text())["outputs"]
    assert sorted(r["index"] for r in rows) == [20, 24]
    for r in rows:
        source = Path("/generated") / Path(r["default_path"]).name
        target = Path(r["project_path"])
        assert target.parent == ROOT and source.is_file() and not target.exists()
        shutil.copyfile(source, target)
        assert target.read_bytes() == source.read_bytes()
        dest = small(r["index"], VERSION)
        assert not dest.exists()
        subprocess.run(["convert", str(target), "-filter", "Lanczos", "-resize", "48x48!", "-depth", "8", "PNG32:"+str(dest)], check=True)
        base.owner(target)
        base.owner(dest)
    print("兩張原生圖及完整畫布48×48已另存")


def verify():
    old_manifest = ROOT / "ENEMY03-small-single-source-manifest-v1-20261002.json"
    old = json.loads(old_manifest.read_text())
    for file, r in old["files"].items():
        assert base.digest(Path(file).read_bytes()) == r["sha256"], file
    refs_path = base.path("ENEMY03", "references")
    refs = json.loads(refs_path.read_text())
    source = Path(refs["source"]).read_bytes()
    assert base.digest(source) == refs["source_sha256"]
    entries = base.pbl.images(source)
    old_selected = json.loads(base.path("ENEMY03", "selected").read_text())["outputs"]
    jobs = json.loads(path("generation-jobs").read_text())["jobs"]
    rows = json.loads(path("selected").read_text())["outputs"]
    assert sorted(r["index"] for r in rows) == [20,24]
    files, records, png_count = {}, [], 0
    board = bytearray(432*288*3)
    def record(file):
        base.owner(file)
        files[str(file)] = {"sha256":base.digest(file.read_bytes()), "size":file.stat().st_size}
    for slot, r in enumerate(rows):
        index = r["index"]
        ref = next(x for x in refs["references"] if x["index"] == index)
        _, offset, _ = entries[index]
        assert offset == ref["pbl_offset"]
        w, h, pixels = base.pbl.decode(source, offset)
        assert (w,h) == (16,16) and base.digest(pixels) == ref["decoded_sha256"]
        ref_file = Path(ref["reference"])
        base.png_crc(ref_file); png_count += 1; record(ref_file)
        rw,rh,ch,ref_rows,pal = base.pbl.read_png(ref_file)
        expected = b"".join(bytes(refs["palette"][pixels[(y//24)*16+x//24]]) for y in range(384) for x in range(384))
        assert (rw,rh,ch,pal) == (384,384,3,None) and b"".join(ref_rows) == expected
        job = next(x for x in jobs if x["index"] == index)
        prior = next(x for x in old_selected if x["index"] == index)
        assert job["refs"] == ["/home/anr2/cht/psychic-war/"+str(ref_file), "/home/anr2/cht/psychic-war/"+prior["project_path"]]
        wanted = [v*3 for v in ref["source_bbox"]]
        versions = []
        for tag, native in (("v1-20261002",Path(prior["project_path"])), (VERSION,Path(r["project_path"]))):
            record(native);rgba(native);png_count += 1
            file = small(index,tag);record(file)
            w,h,raw = rgba(file);png_count += 1
            assert (w,h) == (48,48)
            alpha = raw[3::4]
            box = base.bbox([(i%48,i//48) for i,a in enumerate(alpha) if a >= 128])
            mismatch = sum((a>=128) != (pixels[(i//48//3)*16+i%48//3]!=0) for i,a in enumerate(alpha))
            versions.append({"version":tag,"bbox":box,"bbox_absolute_error":sum(abs(a-b) for a,b in zip(box,wanted)),"occupied_mask_different_pixels":mismatch,"rgba":raw})
        changed_outside = sum(versions[0]["rgba"][i:i+4]!=versions[1]["rgba"][i:i+4] for i in range(0,48*36*4,4))
        for y in range(144):
            for x in range(144):
                colors = [bytes(refs["palette"][pixels[(y//9)*16+x//9]])]
                for v in versions:
                    at=((y//3)*48+x//3)*4; a=v["rgba"][at+3]
                    colors.append(bytes((v["rgba"][at+c]*a+127)//255 for c in range(3)))
                for col,color in enumerate(colors):
                    dest=((slot*144+y)*432+col*144+x)*3
                    board[dest:dest+3]=color
        for v in versions:del v["rgba"]
        records.append({"index":index,"expected_bbox":wanted,"versions":versions,"changed_rgba_pixels_above_row36":changed_outside,
                        "improved_bbox":versions[1]["bbox_absolute_error"]<versions[0]["bbox_absolute_error"],"new_bbox_exact":versions[1]["bbox"]==wanted})
    comparison=ROOT/f"{PREFIX}-comparison-{VERSION}.png"
    assert not comparison.exists()
    palette=list(dict.fromkeys(bytes(board[i:i+3]) for i in range(0,len(board),3)))
    mapping={c:i for i,c in enumerate(palette)}
    base.pbl.png(comparison,432,288,[mapping[bytes(board[i:i+3])] for i in range(0,len(board),3)],palette)
    base.png_crc(comparison);png_count += 1;record(comparison)
    w,h,ch,rr,pal=base.pbl.read_png(comparison)
    assert (w,h,ch,pal)==(432,288,3,None) and b"".join(rr)==board
    tool=Path(__file__)
    snapshot=ROOT/f"{PREFIX}-tool-source-{VERSION}.py"
    with snapshot.open("xb") as f:f.write(tool.read_bytes())
    for file in (path("generation-jobs"),path("selected"),old_manifest,Path("tools/hd/refine_small_tile.py"),snapshot):record(file)
    result={"scope":"兩張局部修整候選，不當正式美術或接入完成","frames":records,"png_crc_decode_count":png_count,"old_unchanged_files":len(old["files"]),"reference_and_comparison_rgb_mismatch":0,"formal_art_accepted":False,"python_version":sys.version.split()[0],"source_sha256":refs["source_sha256"]}
    base.write_json(path("verification"),result);record(path("verification"))
    base.write_json(path("source-manifest"),{"files":files,"parent_manifest":str(old_manifest),"source_sha256":refs["source_sha256"],"scope":"全部留本機；v1未覆寫，工具依賴由parent_manifest快照回查"})
    print(json.dumps(result,ensure_ascii=False))


if __name__ == "__main__":
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command",choices=("convert","verify"))
    args=parser.parse_args();base.owner(ROOT)
    convert() if args.command=="convert" else verify()
