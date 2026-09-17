#!/usr/bin/env python3
"""frames.py — 把 probe `-shots` 的色號陣列整理成摘要與總覽圖。

    tools/py.sh tools/frames.py summary <目錄>            # 每格：步數、非零像素、雜湊，標出變化點
    tools/py.sh tools/frames.py montage <輸出.png> <格…>   # 用 EGA 預設 16 色拼總覽圖（2 欄）

⚠ `-shots` 寫出的檔案副檔名是 .png，內容其實是 320×200 的色號陣列（64,000 bytes）。
⚠ 顏色用 EGA 預設 16 色，**不是**遊戲設定的色盤（見 issue #4），只能看圖形不能驗顏色。
"""
import hashlib
import re
import struct
import sys
import zlib
from pathlib import Path

W, H = 320, 200
EGA = [(0, 0, 0), (0, 0, 170), (0, 170, 0), (0, 170, 170), (170, 0, 0), (170, 0, 170), (170, 85, 0),
       (170, 170, 170), (85, 85, 85), (85, 85, 255), (85, 255, 85), (85, 255, 255), (255, 85, 85),
       (255, 85, 255), (255, 255, 85), (255, 255, 255)]


def step_of(p):
    m = re.search(r"(\d+)", p.stem)
    return int(m.group(1)) if m else -1


def read_frame(p):
    b = p.read_bytes()
    if len(b) != W * H:
        raise ValueError(f"{p}：{len(b)} bytes，不是 {W}×{H} 色號陣列")
    return b


def write_png(path, w, h, rgb):
    raw = b"".join(b"\0" + rgb[y * w * 3:(y + 1) * w * 3] for y in range(h))

    def chunk(t, d):
        return struct.pack(">I", len(d)) + t + d + struct.pack(">I", zlib.crc32(t + d) & 0xFFFFFFFF)

    Path(path).write_bytes(b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 2, 0, 0, 0))
                           + chunk(b"IDAT", zlib.compress(raw)) + chunk(b"IEND", b""))


def summary(d):
    prev = None
    for p in sorted(Path(d).glob("*.png"), key=step_of):
        b = read_frame(p)
        h = hashlib.sha256(b).hexdigest()[:12]
        nz = sum(1 for x in b if x)
        mark = "" if h == prev else "  ← 變化"
        print(f"{step_of(p):>12}  非零 {nz:>6}  {h}{mark}")
        prev = h


def montage(out, frames):
    tiles = [b"".join(bytes(EGA[i & 15]) for i in read_frame(Path(f))) for f in frames]
    cols = 2 if len(tiles) > 1 else 1
    rows = (len(tiles) + cols - 1) // cols
    big = bytearray(b"\x40" * (W * cols * H * rows * 3))
    for k, t in enumerate(tiles):
        ox, oy = (k % cols) * W, (k // cols) * H
        for y in range(H):
            s = (oy + y) * W * cols * 3 + ox * 3
            big[s:s + W * 3] = t[y * W * 3:(y + 1) * W * 3]
    write_png(out, W * cols, H * rows, bytes(big))
    print(f"寫出 {out}（{len(tiles)} 格）")


if __name__ == "__main__":
    if len(sys.argv) >= 3 and sys.argv[1] == "summary":
        summary(sys.argv[2])
    elif len(sys.argv) >= 4 and sys.argv[1] == "montage":
        montage(sys.argv[2], sys.argv[3:])
    else:
        print(__doc__)
        sys.exit(2)
