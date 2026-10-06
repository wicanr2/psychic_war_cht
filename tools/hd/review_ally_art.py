"""研究038 §99：只讀全量ALLY參照及候選，不修改圖片或接受素材。"""
from pathlib import Path
import hashlib
import json
import subprocess
import sys

sys.path.insert(0, 'tools')
import pbl


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def rgb(path, width, height):
    size = subprocess.check_output(['identify', '-format', '%w %h', str(path)], timeout=10).decode()
    if size != f'{width} {height}':
        raise ValueError(f'{path}: unexpected size {size}')
    data = subprocess.check_output(['convert', str(path), '-depth', '8', 'rgb:-'], timeout=10)
    if len(data) != width * height * 3:
        raise ValueError(f'{path}: RGB length differs')
    return data


def bbox(points):
    if not points:
        return None
    return [min(x for x, y in points), min(y for x, y in points),
            max(x for x, y in points), max(y for x, y in points)]


def main():
    original = Path('/orig/psychic-war/ALLY.PBL')
    root = Path('workplace/hd')
    output = root / 'ally-art-review-v1-20261004.json'
    if output.exists():
        raise ValueError('Refusing to overwrite review')
    assert root.stat().st_uid == root.stat().st_gid == 1000
    assert sha(original) == 'c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219'
    data = original.read_bytes()
    images = list(pbl.images(data))
    assert len(images) == 31
    inputs = {str(original): sha(original), __file__: sha(Path(__file__))}
    rows = []
    for image, offset, _ in images:
        w, h, source = pbl.decode(data, offset)
        ref = root / 'ref' / f'ALLY-{image:02d}.png'
        candidate = root / 'art-in' / f'ALLY-{image:02d}.png'
        inputs[str(ref)] = sha(ref)
        inputs[str(candidate)] = sha(candidate)
        expected = b''.join(bytes(pbl.EGA[i]) for i in source)
        actual_ref = rgb(ref, w, h)
        assert actual_ref == expected, f'Original reference differs for #{image}'
        modified = bytearray(actual_ref)
        modified[0] ^= 1
        assert sum(actual_ref[i:i+3] != modified[i:i+3] for i in range(0, len(actual_ref), 3)) == 1
        cw, ch = w * 3, h * 3
        pixels = rgb(candidate, cw, ch)
        original_points = [(x, y) for y in range(ch) for x in range(cw) if source[y//3*w+x//3]]
        original_bbox = bbox(original_points)
        thresholds = {}
        for threshold in (16, 32, 60):
            points = [(x, y) for y in range(ch) for x in range(cw)
                      if max(pixels[(y*cw+x)*3:(y*cw+x)*3+3]) >= threshold]
            thresholds[str(threshold)] = {'bbox': bbox(points), 'nonblack_pixels': len(points),
                                          'bbox_equal': bbox(points) == original_bbox}
        rows.append({'image': image, 'file_offset': offset, 'source_size': [w, h],
                     'candidate_size': [cw, ch], 'original_reference_pixels': w*h,
                     'original_reference_mismatch': 0, 'negative_single_pixel': 1,
                     'original_scaled_bbox': original_bbox, 'original_scaled_nonblack_pixels': len(original_points),
                     'candidate_thresholds': thresholds, 'art_status': 'requires visual review',
                     'runtime_ready': image == 0})
    result = {'scope': '31 ALLY candidate measurements; not art/runtime approval', 'rows': rows,
              'summary': {'images': 31, 'size_and_reference_passed': 31,
                          'bbox_equal_by_threshold': {str(t): sum(r['candidate_thresholds'][str(t)]['bbox_equal'] for r in rows) for t in (16, 32, 60)},
                          'body_canvas_24x32': sum(r['source_size'] == [24, 32] for r in rows),
                          'small_canvas_16x16': sum(r['source_size'] == [16, 16] for r in rows)},
              'inputs_sha256': inputs,
              'limits': 'Canvas size does not prove purpose. RGB thresholds locate silhouette only; same bbox cannot prove pose/shape/colour fidelity. Runtime readiness is limited to spec024 1.2 source#0.'}
    output.write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps(result['summary'], ensure_ascii=False))
    for row in rows:
        print(row['image'], row['original_scaled_bbox'], row['candidate_thresholds']['32']['bbox'])


if __name__ == '__main__':
    main()
