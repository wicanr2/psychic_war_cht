"""`.PBL` 圖檔解碼（issue #18）：偏移表 → 每張圖的 RLE 解壓 → 4bpp 色號陣列。

    tools/py.sh tools/pbl.py list <檔案…>                     # 每張圖的尺寸與壓縮率
    tools/py.sh tools/pbl.py dump <檔案> <輸出目錄> [圖號…]    # 每張圖存 .idx（色號，寬×高 bytes）與 .png
    tools/py.sh tools/pbl.py find <ref.png> <檔案…>            # 在畫面上找出每張圖畫在哪（色號比對）
    tools/py.sh tools/pbl.py check <檔案…>                     # 每張圖：吃掉的輸入 vs 長度、輸出長度（spec 010 §6 第 1 項）

格式（`docs/spec/010`，證據見 `docs/re/023`）：

    u16 表長 T（＝第 1 張圖的偏移，張數 ＝ T÷2）
    u16 offset[0..T/2-1]                 檔案內的絕對偏移
    每張圖：u8 w、u8 h（w 是寬度÷8、h 是高度÷8）、u8 map[16]（CGA 4 色對照表）、RLE 資料
    解壓後 w×h×32 bytes ＝ (w×8)×(h×8) 像素，一 byte 兩個像素（高半位元組在左），逐列排列

RLE（`sub_18B76`，`0161:8666`）：讀一個 byte 當「前一個」，再讀下一個；
兩個相同就是重複段，後面接一個 byte 當次數（寫次數次，兩個相同的 byte 不另外寫）；
不同就把前一個寫出去。

色號對照表只在 CGA 4 色模式用（`sub_18C62` 把一 byte 的兩個半位元組各查一次表）：
`sub_18B76` 在顯示模式旗標 `cs:863E` 是 2 或 5 時跳過轉換，EGA 走的就是那條路，
所以 EGA 下表沒有作用（表的內容全部落在 0–3，也證明它是給 4 色模式用的）。
"""
import json
import pathlib
import struct
import sys
import zlib

# EGA 預設 16 色（200 線 RGBI），與 docs/re/005 的色盤一致
EGA = [
    (0x00, 0x00, 0x00), (0x00, 0x00, 0xAA), (0x00, 0xAA, 0x00), (0x00, 0xAA, 0xAA),
    (0xAA, 0x00, 0x00), (0xAA, 0x00, 0xAA), (0xAA, 0x55, 0x00), (0xAA, 0xAA, 0xAA),
    (0x55, 0x55, 0x55), (0x55, 0x55, 0xFF), (0x55, 0xFF, 0x55), (0x55, 0xFF, 0xFF),
    (0xFF, 0x55, 0x55), (0xFF, 0x55, 0xFF), (0xFF, 0xFF, 0x55), (0xFF, 0xFF, 0xFF),
]


def images(data):
    """回 [(圖號, 偏移, 長度)]。"""
    t = struct.unpack_from("<H", data, 0)[0]
    if t < 2 or t % 2 or t > len(data):
        raise ValueError("表長不合理：%d" % t)
    offs = [struct.unpack_from("<H", data, 2 * i)[0] for i in range(t // 2)]
    out = []
    for i, o in enumerate(offs):
        end = offs[i + 1] if i + 1 < len(offs) else len(data)
        out.append((i, o, end - o))
    return out


def decode(data, off, remap=False):
    """回 (寬, 高, 色號陣列)；色號陣列是寬×高 bytes，一格一個色號。

    remap=True 才套色號對照表（CGA 4 色模式；EGA 不套，見模組說明）。
    """
    w, h = data[off], data[off + 1]
    table = data[off + 2:off + 18]
    need = w * h * 32
    si = off + 18
    out = bytearray()
    prev = data[si]
    si += 1
    while len(out) < need and si < len(data):
        cur = data[si]
        si += 1
        if cur == prev:
            n = data[si]
            si += 1
            out += bytes([prev]) * min(n, need - len(out))
            if len(out) >= need or si >= len(data):
                break
            prev = data[si]
            si += 1
        else:
            out.append(prev)
            prev = cur
    out = bytes(out[:need]).ljust(need, b"\0")
    if remap:
        out = bytes((table[b >> 4] << 4) | (table[b & 0xF] & 0xF) for b in out)
    W, H = w * 8, h * 8
    px = bytearray(W * H)
    for y in range(H):
        row = out[y * (W // 2):(y + 1) * (W // 2)]
        o = y * W
        for x2, b in enumerate(row):
            px[o + 2 * x2] = b >> 4
            px[o + 2 * x2 + 1] = b & 0xF
    return W, H, px


def png(path, w, h, px, palette=EGA):
    raw = b"".join(b"\0" + bytes(b for i in range(w) for b in palette[px[y * w + i]]) for y in range(h))

    def chunk(tag, body):
        c = tag + body
        return struct.pack(">I", len(body)) + c + struct.pack(">I", zlib.crc32(c) & 0xFFFFFFFF)

    path.write_bytes(b"\x89PNG\r\n\x1a\n"
                     + chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 2, 0, 0, 0))
                     + chunk(b"IDAT", zlib.compress(raw, 9))
                     + chunk(b"IEND", b""))


def read_png(path):
    """讀 8 位元 RGB／RGBA／調色盤 PNG，回 (寬, 高, 每格 bytes, 每列 bytes, 調色盤或 None)。

    調色盤 PNG（`import` 預設就是這種）回 ch=1，呼叫端要自己查 PLTE。
    """
    b = pathlib.Path(path).read_bytes()
    assert b[:8] == b"\x89PNG\r\n\x1a\n", "不是 PNG"
    i, idat, w, plte = 8, b"", None, None
    while i < len(b):
        n = struct.unpack_from(">I", b, i)[0]
        tag = b[i + 4:i + 8]
        body = b[i + 8:i + 8 + n]
        if tag == b"IHDR":
            w, h, depth, ctype = struct.unpack_from(">IIBB", body, 0)
            assert depth == 8 and ctype in (2, 3, 6), "只支援 8 位元 RGB／RGBA／調色盤（depth=%d ctype=%d）" % (depth, ctype)
            ch = {2: 3, 3: 1, 6: 4}[ctype]
        elif tag == b"PLTE":
            plte = [tuple(body[3 * k:3 * k + 3]) for k in range(len(body) // 3)]
        elif tag == b"IDAT":
            idat += body
        elif tag == b"IEND":
            break
        i += 12 + n
    raw = zlib.decompress(idat)
    stride = w * ch
    rows, prev = [], bytes(stride)
    p = 0
    for _ in range(h):
        f = raw[p]
        line = bytearray(raw[p + 1:p + 1 + stride])
        p += 1 + stride
        for x in range(stride):
            a = line[x - ch] if x >= ch else 0
            bb = prev[x]
            c = prev[x - ch] if x >= ch else 0
            if f == 1:
                line[x] = (line[x] + a) & 0xFF
            elif f == 2:
                line[x] = (line[x] + bb) & 0xFF
            elif f == 3:
                line[x] = (line[x] + (a + bb) // 2) & 0xFF
            elif f == 4:
                pa, pb, pc = abs(bb - c), abs(a - c), abs(a + bb - 2 * c)
                line[x] = (line[x] + (a if pa <= pb and pa <= pc else bb if pb <= pc else c)) & 0xFF
        rows.append(bytes(line))
        prev = line
    return w, h, ch, rows, plte


def screen_indices(path):
    """畫面 PNG → 色號陣列（用 EGA 預設 16 色反查；出現別的顏色就報錯）。"""
    w, h, ch, rows, plte = read_png(path)
    back = {c: i for i, c in enumerate(EGA)}
    px = bytearray(w * h)
    for y in range(h):
        r = rows[y]
        for x in range(w):
            c = plte[r[x]] if ch == 1 else (r[ch * x], r[ch * x + 1], r[ch * x + 2])
            if c not in back:
                raise SystemExit("畫面有非 EGA 預設色：(%d,%d) %s" % (x, y, c))
            px[y * w + x] = back[c]
    return w, h, px


def cmd_list(paths):
    for p in paths:
        data = pathlib.Path(p).read_bytes()
        try:
            items = images(data)
        except ValueError as e:
            print("%s：%s" % (p, e))
            continue
        print("== %s %d bytes，%d 張" % (pathlib.Path(p).name, len(data), len(items)))
        for i, off, size in items:
            w, h = data[off], data[off + 1]
            print("  #%-2d @%05X %5d bytes  %3d×%-3d 格 ＝ %4d×%-4d 像素  解壓 %6d（%.0f%%）"
                  % (i, off, size, w, h, w * 8, h * 8, w * h * 32, 100.0 * size / max(1, w * h * 32)))


def cmd_dump(path, outdir, which):
    data = pathlib.Path(path).read_bytes()
    out = pathlib.Path(outdir)
    out.mkdir(parents=True, exist_ok=True)
    name = pathlib.Path(path).stem
    for i, off, _ in images(data):
        if which and i not in which:
            continue
        w, h, px = decode(data, off)
        (out / ("%s-%02d.idx" % (name, i))).write_bytes(bytes([w & 0xFF, w >> 8, h & 0xFF, h >> 8]) + bytes(px))
        png(out / ("%s-%02d.png" % (name, i)), w, h, px)
        print("%s #%d %d×%d → %s" % (name, i, w, h, out / ("%s-%02d.png" % (name, i))))


def cmd_check(paths):
    bad = 0
    total = 0
    for p in paths:
        data = pathlib.Path(p).read_bytes()
        for i, off, size in images(data):
            total += 1
            w, h = data[off], data[off + 1]
            need = w * h * 32
            si = off + 18
            n = 0
            prev = data[si]
            si += 1
            while n < need and si < len(data):
                cur = data[si]
                si += 1
                if cur == prev:
                    n = min(n + data[si], need)   # 原版以 cx 計數，最後一段超出就停
                    si += 1
                    if n >= need or si >= len(data):
                        break
                    prev = data[si]
                    si += 1
                else:
                    n += 1
                    prev = cur
            used = si - off
            if abs(used - size) > 2 or n < need:
                bad += 1
                print("%s #%d：長度 %d、吃掉 %d、輸出 %d/%d" % (pathlib.Path(p).name, i, size, used, n, need))
    print("%d 張，異常 %d 張" % (total, bad))
    return 1 if bad else 0


def cmd_find(ref, paths):
    sw, sh, spx = screen_indices(ref)
    for p in paths:
        data = pathlib.Path(p).read_bytes()
        for i, off, _ in images(data):
            w, h, px = decode(data, off)
            hit = None
            for y in range(0, sh - h + 1):
                for x in range(0, sw - w + 1):
                    if all(spx[(y + r) * sw + x:(y + r) * sw + x + w] == px[r * w:(r + 1) * w] for r in range(h)):
                        hit = (x, y)
                        break
                if hit:
                    break
            print(json.dumps({"file": pathlib.Path(p).name, "image": i, "w": w, "h": h,
                              "at": hit}, ensure_ascii=False))


def main(argv):
    if len(argv) < 3:
        print(__doc__)
        return 2
    cmd = argv[1]
    if cmd == "list":
        cmd_list(argv[2:])
    elif cmd == "dump":
        cmd_dump(argv[2], argv[3], [int(x) for x in argv[4:]])
    elif cmd == "find":
        cmd_find(argv[2], argv[3:])
    elif cmd == "check":
        return cmd_check(argv[2:])
    else:
        print(__doc__)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
