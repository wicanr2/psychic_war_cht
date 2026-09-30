"""把幾何基元描述畫成 HD 主題的圖（docs/spec/024 §3.1）。

    tools/py.sh tools/hd/render.py <基元.json> <輸出.png> [倍率]

基元用**原版像素座標**描述（可以是小數），顏色用 EGA 色號——這樣重繪的配色
自動跟原版一致，不用另外維護一份色表。

畫的時候以 SS×SS 超取樣求覆蓋率再合成，所以斜邊是抗鋸齒的；原版靠塊狀感
撐起來的形狀（例如箭頭的頭與柄隔著 3 px）要在基元裡就接起來，不是這裡的事
（docs/re/038 §11.4）。
"""
import json
import pathlib
import struct
import sys
import zlib

SS = 4  # 每個輸出像素的超取樣邊長

EGA = [
    (0x00, 0x00, 0x00), (0x00, 0x00, 0xAA), (0x00, 0xAA, 0x00), (0x00, 0xAA, 0xAA),
    (0xAA, 0x00, 0x00), (0xAA, 0x00, 0xAA), (0xAA, 0x55, 0x00), (0xAA, 0xAA, 0xAA),
    (0x55, 0x55, 0x55), (0x55, 0x55, 0xFF), (0x55, 0xFF, 0x55), (0x55, 0xFF, 0xFF),
    (0xFF, 0x55, 0x55), (0xFF, 0x55, 0xFF), (0xFF, 0xFF, 0x55), (0xFF, 0xFF, 0xFF),
]


def color(spec):
    """色號或 [r,g,b]。色號讓重繪自動跟原版同色。"""
    if isinstance(spec, int):
        return EGA[spec]
    return tuple(spec[:3])


def polygons(shape):
    """把一個基元展開成若干多邊形（原版座標）。"""
    k = shape["kind"]
    if k == "poly":
        return [[tuple(p) for p in shape["points"]]]
    if k == "rect":
        x, y = shape["at"]
        w, h = shape["size"]
        return [[(x, y), (x + w, y), (x + w, y + h), (x, y + h)]]
    if k == "grid":
        # 規則格線：在 rect 範圍內，每 step 畫一條 width 寬的線
        x0, y0, w, h = shape["rect"]
        sx, sy = shape["step"]
        ox, oy = shape.get("offset", [0, 0])
        t = shape.get("width", 1)
        out = []
        if sx:
            x = x0 + ox
            while x < x0 + w:
                out.append([(x, y0), (x + t, y0), (x + t, y0 + h), (x, y0 + h)])
                x += sx
        if sy:
            y = y0 + oy
            while y < y0 + h:
                out.append([(x0, y), (x0 + w, y), (x0 + w, y + t), (x0, y + t)])
                y += sy
        return out
    raise SystemExit("不認得的基元 kind=%s" % k)


def fill(cov, W, H, poly, scale):
    """掃描線填多邊形，累加到 cov（長度 W*SS × H*SS 的覆蓋圖）。"""
    pts = [(x * scale * SS, y * scale * SS) for x, y in poly]
    ys = [p[1] for p in pts]
    y0, y1 = max(0, int(min(ys))), min(H * SS - 1, int(max(ys)) + 1)
    n = len(pts)
    for sy in range(y0, y1 + 1):
        yc = sy + 0.5
        xs = []
        for i in range(n):
            ax, ay = pts[i]
            bx, by = pts[(i + 1) % n]
            if (ay <= yc) == (by <= yc):
                continue
            xs.append(ax + (yc - ay) * (bx - ax) / (by - ay))
        xs.sort()
        for i in range(0, len(xs) - 1, 2):
            a, b = int(xs[i] + 0.5), int(xs[i + 1] + 0.5)
            row = sy * W * SS
            for sx in range(max(0, a), min(W * SS, b)):
                cov[row + sx] = 1


def write_png(path, w, h, rgba):
    raw = b"".join(b"\0" + bytes(rgba[y * w * 4:(y + 1) * w * 4]) for y in range(h))

    def chunk(tag, body):
        c = tag + body
        return struct.pack(">I", len(body)) + c + struct.pack(">I", zlib.crc32(c) & 0xFFFFFFFF)

    pathlib.Path(path).write_bytes(
        b"\x89PNG\r\n\x1a\n"
        + chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 6, 0, 0, 0))
        + chunk(b"IDAT", zlib.compress(raw, 9))
        + chunk(b"IEND", b""))


def main(argv):
    if len(argv) < 3:
        raise SystemExit(__doc__)
    spec = json.loads(pathlib.Path(argv[1]).read_text(encoding="utf-8"))
    out = argv[2]
    scale = int(argv[3]) if len(argv) > 3 else 3
    sw, sh = spec["size"]
    W, H = sw * scale, sh * scale
    rgba = bytearray(W * H * 4)

    # 底：省略時全透明，讓原版透出來
    bg = spec.get("background")
    if bg is not None:
        r, g, b = color(bg)
        for i in range(W * H):
            rgba[4*i:4*i+4] = bytes((r, g, b, 0xFF))

    for shape in spec["shapes"]:
        # 漸層：原版的框線是「亮色壓暗色」的硬斜角，HD 版畫成平滑過渡
        grad = shape.get("gradient")
        if grad:
            stops = [color(c) for c in grad]  # 2 段或多段，沿短邊等距
            vertical = shape.get("dir", "v") == "v"
        else:
            r, g, b = color(shape["color"])
        cov = bytearray(W * SS * H * SS)
        ps = polygons(shape)
        for poly in ps:
            fill(cov, W, H, poly, scale)
        if grad:
            xs = [p[0] for poly in ps for p in poly]
            ys = [p[1] for poly in ps for p in poly]
            g0, g1 = (min(ys), max(ys)) if vertical else (min(xs), max(xs))
            g0, g1 = g0 * scale, max(g1 * scale, g0 * scale + 1)
        for y in range(H):
            for x in range(W):
                n = 0
                for dy in range(SS):
                    row = (y * SS + dy) * W * SS + x * SS
                    n += sum(cov[row:row + SS])
                if not n:
                    continue
                a = n / (SS * SS)
                if grad:
                    t = ((y if vertical else x) + 0.5 - g0) / (g1 - g0)
                    t = 0.0 if t < 0 else (1.0 if t > 1 else t)
                    seg = t * (len(stops) - 1)
                    k0 = min(int(seg), len(stops) - 2)
                    u = seg - k0
                    a0, a1 = stops[k0], stops[k0 + 1]
                    r, g, b = (int(a0[k] + (a1[k] - a0[k]) * u + 0.5) for k in range(3))
                i = 4 * (y * W + x)
                o = rgba[i + 3] / 255.0
                na = a + o * (1 - a)
                for c, v in enumerate((r, g, b)):
                    rgba[i + c] = int((v * a + rgba[i + c] * o * (1 - a)) / na + 0.5)
                rgba[i + 3] = int(na * 255 + 0.5)

    write_png(out, W, H, rgba)
    print("%s %d×%d（%d 個基元，%d 倍）" % (out, W, H, len(spec["shapes"]), scale))


if __name__ == "__main__":
    main(sys.argv)
