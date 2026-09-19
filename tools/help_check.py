"""F4 說明頁的逐像素驗收（docs/spec/012 §5 第 3 項、docs/spec/022 §6）。

    tools/py.sh tools/help_check.py <截圖.png> [--text text] [--font font] [--scale 3]
                                    [--prot 答案] [--dump 期望值.png]

期望值由 text/help.json（內容 ＋ `keycaps` ＋ `layout`）與字型排算，不看前端自己的輸出。
排版與 apps/psychicwar/helppage.go 相同：分段規則（docs/spec/022 §3.3）、欄位分配、
三個繪圖指令 rect／text／chip 兩邊各實作一次，常數只有 help.json 一份。

`--prot` 是防拷畫面時多出來的那一列答案（docs/spec/013 §2.2）；不給就當成沒有題目。
`--dump` 把期望值寫成 PNG，用來跟設計稿對照。
"""
import json
import pathlib
import struct
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import pbl  # noqa: E402

ROOT = pathlib.Path(__file__).resolve().parent.parent


def load_font(path):
    b = pathlib.Path(path).read_bytes()
    assert b[:8] == b"GOLEMFNT", path
    w, h, n = struct.unpack_from("<HHI", b, 8)
    rb = (w + 7) // 8
    out, off = {}, 16
    for _ in range(n):
        cp = struct.unpack_from("<I", b, off)[0]
        off += 5
        out[chr(cp)] = b[off:off + rb * h]
        off += rb * h
    return {"w": w, "h": h, "rb": rb, "g": out}


# ---------------------------------------------------------------- 幾何（只用前進量，不看字模）

def width(fw, s):
    """一段字佔幾個像素：半形半格、全形一格。"""
    return sum(fw // 2 if ord(c) < 0x80 else fw for c in s)


def parse(lines):
    """docs/spec/022 §3.3 的四條分段規則。"""
    title = lines[0].strip("　 ") if lines else ""
    secs = []
    for s in lines[1:]:
        if not s.strip("　 "):
            continue
        if s.startswith("【"):
            i = s.find("】")
            head, rest = (s, "") if i < 0 else (s[:i + 1], s[i + 1:].strip("　 "))
            secs.append({"head": head, "rest": rest, "body": []})
        elif secs:
            secs[-1]["body"].append(s)
    return title, secs


def items(L, sec, colw):
    """段落展開成列；標頭同列放不下的尾巴自成一列。"""
    out = [{"head": True, "s": sec["head"], "tail": ""}]
    if sec["rest"]:
        if width(L["font_head_w"], sec["head"]) + L["tail_gap"] + width(L["font_body_w"], sec["rest"]) <= colw:
            out[0]["tail"] = sec["rest"]
        else:
            out.append({"head": False, "s": "　" + sec["rest"], "tail": ""})
    out += [{"head": False, "s": s, "tail": ""} for s in sec["body"]]
    return out


def split_caps(s, caps, on=True):
    """切出要畫成反白鍵帽的鍵名。比對最長優先，否則 F10 會被切成 F1 ＋ 0。"""
    if not on or not caps:
        return [("text", s)]
    out, i, buf = [], 0, ""
    while i < len(s):
        hit = next((k for k in caps if s.startswith(k, i)), None)
        if hit is None:
            buf += s[i]
            i += 1
            continue
        if buf:
            out.append(("text", buf))
            buf = ""
        out.append(("chip", hit))
        i += len(hit)
    if buf:
        out.append(("text", buf))
    return out


def line_ops(L, caps, s, x, y, color, use_caps=True):
    ops, pen = [], x
    for kind, seg in split_caps(s, caps, use_caps):
        if kind == "chip":
            ops.append(("chip", pen, y, False, seg, L["color_chip"], L["color_chip_text"], L["chip_pad"]))
        else:
            ops.append(("text", pen, y, False, seg, color, 0, 0))
        pen += width(L["font_body_w"], seg)
    return ops


def geometry(doc, prot=""):
    """整頁的繪圖指令（docs/spec/022 §4，方案 B）。"""
    L, caps = doc["layout"], sorted(doc.get("keycaps", []), key=len, reverse=True)
    title, secs = parse(doc["lines"])
    colx = [L["margin"], L["margin"] + L["col_w"] + L["gutter"]]
    right = colx[1] + L["col_w"]

    ops = [("rect", 0, 0, L["page_w"], L["page_h"], L["color_bg"]),
           ("rect", L["margin"], L["band_y"], right - L["margin"], L["band_h"], L["color_band"])]
    ops.append(("text", (L["page_w"] - width(L["font_head_w"], title)) // 2,
                L["band_y"] + (L["band_h"] - L["font_head_h"]) // 2,
                True, title, L["color_title"], 0, 0))

    wide, narrow = [], []
    for s in secs:
        w = max([width(L["font_body_w"], b) for b in s["body"]] +
                [width(L["font_body_w"], "　" + s["rest"]) if s["rest"] else 0])
        (wide if w > L["col_w"] else narrow).append(s)

    hs = [L["head_pitch"] + L["pitch"] * (len(items(L, s, L["col_w"])) - 1) for s in narrow]
    best, cut = None, 0
    for k in range(1, len(narrow)):
        lh = sum(hs[:k]) + L["gap"] * (k - 1)
        rh = sum(hs[k:]) + L["gap"] * (len(narrow) - k - 1)
        if best is None or abs(lh - rh) < best:
            best, cut = abs(lh - rh), k

    def emit(group, x, colw, y):
        out = []
        for si, sec in enumerate(group):
            if si:
                y += L["gap"]
            for it in items(L, sec, colw):
                if it["head"]:
                    out.append(("text", x, y, True, it["s"], L["color_head"], 0, 0))
                    if it["tail"]:
                        out += line_ops(L, caps, it["tail"],
                                        x + width(L["font_head_w"], it["s"]) + L["tail_gap"],
                                        y + L["tail_dy"], L["color_body"])
                    y += L["head_pitch"]
                else:
                    out += line_ops(L, caps, it["s"], x, y, L["color_body"])
                    y += L["pitch"]
        return out, y

    bottom = L["body_y"]
    for ci, group in enumerate([narrow[:cut], narrow[cut:]]):
        out, y = emit(group, colx[ci], L["col_w"], L["body_y"])
        ops += out
        bottom = max(bottom, y)
    ops.append(("rect", colx[0] + L["col_w"] + (L["gutter"] - L["rule"]) // 2, L["body_y"],
                L["rule"], bottom - L["body_y"] - L["rule_inset"], L["color_rule"]))

    y = bottom + L["gap"]
    ops.append(("rect", L["margin"], y, right - L["margin"], L["rule"], L["color_rule"]))
    y += L["gap"]
    out, y = emit(wide, L["margin"], right - L["margin"], y)
    ops += out
    if prot:
        y += L["prot_gap"]
        ops.append(("rect", L["margin"], y, right - L["margin"], L["rule"], L["color_rule"]))
        ops += line_ops(L, caps, prot, L["margin"], y + L["prot_gap"], L["color_prot"], use_caps=False)
    return ops


def content_bottom(L, ops):
    """內容底緣（背景那一塊不算）。"""
    b = 0
    for op in ops[1:]:
        if op[0] == "rect":
            b = max(b, op[2] + op[4])
        else:
            b = max(b, op[2] + (L["font_head_h"] if op[3] else L["font_body_h"]))
    return b


# ---------------------------------------------------------------- 光柵化（三個指令）

def ink_box(f, s):
    lo = top = hi = bot = None
    pen = 0
    for ch in s:
        g = f["g"].get(ch)
        if g:
            for gy in range(f["h"]):
                for gx in range(f["w"]):
                    if g[gy * f["rb"] + gx // 8] & (0x80 >> (gx % 8)):
                        x = pen + gx
                        lo = x if lo is None else min(lo, x)
                        hi = x if hi is None else max(hi, x)
                        top = gy if top is None else min(top, gy)
                        bot = gy if bot is None else max(bot, gy)
        pen += f["w"] // 2 if ord(ch) < 0x80 else f["w"]
    return None if lo is None else (lo, top, hi, bot)


def fill(px, w, h, u, x, y, rw, rh, c):
    for yy in range(y * u, (y + rh) * u):
        if 0 <= yy < h:
            row = yy * w
            for xx in range(x * u, (x + rw) * u):
                if 0 <= xx < w:
                    px[row + xx] = c


def draw_text(px, w, h, u, f, s, x, y, c):
    pen = x
    for ch in s:
        g = f["g"].get(ch)
        if g:
            for gy in range(f["h"]):
                for gx in range(f["w"]):
                    if not g[gy * f["rb"] + gx // 8] & (0x80 >> (gx % 8)):
                        continue
                    for py in range(u):
                        for pxi in range(u):
                            xx, yy = (pen + gx) * u + pxi, (y + gy) * u + py
                            if 0 <= xx < w and 0 <= yy < h:
                                px[yy * w + xx] = c
        pen += f["w"] // 2 if ord(ch) < 0x80 else f["w"]


def expected(doc, f24, f16, w, h, prot=""):
    L = doc["layout"]
    u = w // L["page_w"]
    px = bytearray(w * h)
    for op in geometry(doc, prot):
        if op[0] == "rect":
            fill(px, w, h, u, op[1], op[2], op[3], op[4], op[5])
            continue
        _, x, y, big, s, c, c2, pad = op
        f = f24 if big else f16
        if op[0] == "chip":
            box = ink_box(f, s)
            if box:
                lo, top, hi, bot = box
                fill(px, w, h, u, x + lo - pad, y + top - pad,
                     hi - lo + 1 + 2 * pad, bot - top + 1 + 2 * pad, c)
            draw_text(px, w, h, u, f, s, x, y, c2)
        else:
            draw_text(px, w, h, u, f, s, x, y, c)
    return px


def arg(argv, name, default=None):
    return argv[argv.index(name) + 1] if name in argv else default


def main(argv):
    if not argv:
        print(__doc__)
        return 2
    shot = argv[0]
    text_dir = ROOT / arg(argv, "--text", "text")
    font_dir = ROOT / arg(argv, "--font", "font")
    scale = int(arg(argv, "--scale", "3"))
    prot = arg(argv, "--prot", "")
    doc = json.loads((text_dir / "help.json").read_text(encoding="utf-8"))
    L = doc["layout"]
    if prot:
        prot = doc["protection_label"] + prot
    f24 = load_font(font_dir / "cjk24.golemfnt")
    f16 = load_font(font_dir / "cjk16.golemfnt")
    for f, kw, kh in ((f24, "font_head_w", "font_head_h"), (f16, "font_body_w", "font_body_h")):
        if (f["w"], f["h"]) != (L[kw], L[kh]):
            print("字型 %d×%d 與 layout 的 %d×%d 對不上" % (f["w"], f["h"], L[kw], L[kh]))
            return 1
    bottom = content_bottom(L, geometry(doc, doc["protection_label"] + "ANSWER"))
    if bottom > L["page_h"]:
        print("說明頁內容底緣 y=%d，超出畫布高度 %d" % (bottom, L["page_h"]))
        return 1
    w, h = 320 * scale, 200 * scale
    exp = expected(doc, f24, f16, w, h, prot)
    dump = arg(argv, "--dump")
    if dump:
        pbl.png(ROOT / dump, w, h, exp)
    sw, sh, spx = pbl.screen_indices(ROOT / shot)
    if (sw, sh) != (w, h):
        print("截圖大小 %d×%d，不是 %d×%d" % (sw, sh, w, h))
        return 1
    bad = sum(1 for i in range(w * h) if exp[i] != spx[i])
    print("說明頁：內容底緣 y=%d／%d，不同像素 %d / %d" % (bottom, L["page_h"], bad, w * h))
    return 0 if bad == 0 else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
