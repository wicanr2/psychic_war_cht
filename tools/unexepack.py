#!/usr/bin/env python3
"""unexepack.py — 把 Microsoft EXEPACK 壓縮的 MZ 執行檔解成一般 MZ（帶重定位表）。

    tools/py.sh tools/unexepack.py <輸入.EXE> <輸出.EXE> [--verify <記憶體傾印> <載入段 hex>]

--verify 拿執行器在「真正進入點第一道指令」那一步傾印的映像來比：
把重定位以同一個載入段套進解壓結果，逐位元組必須完全相同。
兩個獨立的來源（靜態解壓、執行期的解壓器自己解）一致，才算解對。

格式依公開的 EXEPACK 文件：映像尾端的段放 EXEPACK 標頭與解壓器，
壓縮資料從映像結尾往前讀；指令 B0/B1 是填滿、B2/B3 是複製，最低位元 1 表示最後一段；
重定位表緊接在解壓器的錯誤訊息之後，分 16 組、每組一個數量加上 N 個位移，段 = 組號 × 1000h。
"""
import struct
import sys
from pathlib import Path

ERR = b"Packed file is corrupt"


class Bad(Exception):
    pass


def parse_mz(b):
    if b[:2] not in (b"MZ", b"ZM"):
        raise Bad("不是 MZ 執行檔")
    (magic, last, pages, nrel, hdrpara, minalloc, maxalloc,
     ss, sp, csum, ip, cs, reloff, ovl) = struct.unpack("<2s13H", b[:28])
    size = (pages - 1) * 512 + last if last else pages * 512
    return {"nrel": nrel, "hdr": hdrpara * 16, "min": minalloc, "max": maxalloc,
            "ss": ss, "sp": sp, "ip": ip, "cs": cs, "size": size}


def unpack(b):
    mz = parse_mz(b)
    if mz["nrel"]:
        raise Bad("有重定位表，不像 EXEPACK")
    img = b[mz["hdr"]:mz["size"]]
    hoff = mz["cs"] * 16
    h = img[hoff:]
    # 標頭有 16 與 18 bytes 兩種：簽章 "RB" 在 +14 或 +16
    if h[14:16] == b"RB":
        real_ip, real_cs, _mem, xsize, real_sp, real_ss, dest_len = struct.unpack("<7H", h[:14])
        skip, hlen = 1, 16
    elif h[16:18] == b"RB":
        real_ip, real_cs, _mem, xsize, real_sp, real_ss, dest_len, skip = struct.unpack("<8H", h[:16])
        hlen = 18
    else:
        raise Bad("找不到 EXEPACK 簽章 RB")
    if mz["ip"] != hlen:
        raise Bad(f"進入點 IP={mz['ip']:04X} 與標頭長度 {hlen} 不符")
    e = h.find(ERR)
    if e < 0 or e > xsize:
        raise Bad("找不到解壓器的錯誤訊息，無法定位重定位表")

    # 重定位表
    p = e + len(ERR)
    relocs = []
    for group in range(16):
        (n,) = struct.unpack_from("<H", h, p)
        p += 2
        for _ in range(n):
            (off,) = struct.unpack_from("<H", h, p)
            p += 2
            relocs.append((group * 0x1000, off))
    if p != xsize:
        raise Bad(f"重定位表結束在 +{p:X}，EXEPACK 區段長 {xsize:X}：版面不對")

    # 解壓：就地、從尾端往前
    src = hoff - (skip - 1) * 16
    out_len = dest_len * 16
    buf = bytearray(max(out_len, len(img)))
    buf[:hoff] = img[:hoff]
    while src > 0 and buf[src - 1] == 0xFF:
        src -= 1
    dst = out_len
    while True:
        if src < 3:
            raise Bad("壓縮資料在指令結束前用完")
        src -= 1
        cmd = buf[src]
        src -= 2
        length = buf[src] | buf[src + 1] << 8
        kind = cmd & 0xFE
        if kind == 0xB0:
            src -= 1
            fill = buf[src]
            dst -= length
            buf[dst:dst + length] = bytes([fill]) * length
        elif kind == 0xB2:
            src -= length
            dst -= length
            buf[dst:dst + length] = bytes(buf[src:src + length])
        else:
            raise Bad(f"不認識的指令 {cmd:02X}（src={src:X}）")
        if dst < src:
            raise Bad("輸出指標跑到輸入指標前面")
        if cmd & 1:
            break
    if dst != src:
        # 剩下的前段是未壓縮、原地不動的資料
        pass
    image = bytes(buf[:out_len])
    return {"image": image, "relocs": relocs, "ip": real_ip, "cs": real_cs, "sp": real_sp,
            "ss": real_ss, "min": mz["min"], "max": mz["max"], "packed_image_len": len(img)}


def build_mz(u):
    nrel = len(u["relocs"])
    hdr_len = 28 + nrel * 4
    hdr_para = (hdr_len + 15) // 16
    total = hdr_para * 16 + len(u["image"])
    pages = (total + 511) // 512
    last = total % 512
    extra = u["packed_image_len"] + u["min"] * 16 - len(u["image"])
    minalloc = max(0, (extra + 15) // 16)
    head = struct.pack("<2s13H", b"MZ", last, pages, nrel, hdr_para, minalloc, u["max"],
                       u["ss"], u["sp"], 0, u["ip"], u["cs"], 28, 0)
    rel = b"".join(struct.pack("<HH", off, seg) for seg, off in u["relocs"])
    return (head + rel).ljust(hdr_para * 16, b"\0") + u["image"]


def verify(u, dump_path, load_seg):
    mem = bytearray(u["image"])
    for seg, off in u["relocs"]:
        at = seg * 16 + off
        (v,) = struct.unpack_from("<H", mem, at)
        struct.pack_into("<H", mem, at, (v + load_seg) & 0xFFFF)
    dump = Path(dump_path).read_bytes()
    n = min(len(dump), len(mem))
    diff = [i for i in range(n) if dump[i] != mem[i]]
    return n, diff


def main(argv):
    if len(argv) not in (3, 6) or (len(argv) == 6 and argv[3] != "--verify"):
        print(__doc__)
        return 2
    u = unpack(Path(argv[1]).read_bytes())
    Path(argv[2]).write_bytes(build_mz(u))
    print(f"解壓映像 {len(u['image'])} bytes，重定位 {len(u['relocs'])} 筆，"
          f"進入點 {u['cs']:04X}:{u['ip']:04X}，堆疊 {u['ss']:04X}:{u['sp']:04X} → {argv[2]}")
    if len(argv) == 6:
        n, diff = verify(u, argv[4], int(argv[5], 16))
        print(f"與傾印比對 {n} bytes：不一致 {len(diff)} 處" + (f"，前 10 處 {[hex(i) for i in diff[:10]]}" if diff else ""))
        return 1 if diff else 0
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main(sys.argv))
    except Bad as e:
        print(f"unexepack：{e}", file=sys.stderr)
        sys.exit(1)
