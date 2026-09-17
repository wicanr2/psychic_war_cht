"""文字抽取雛形（issue #16，docs/re/015）：從玩家自備的 I_MENUnn.BIN／I_MENUH.BIN 列出所有字串與偏移。

    tools/py.sh tools/text_extract.py menu <I_MENU 檔案…> [--json out.json]
    tools/py.sh tools/text_extract.py enmy <I_ENMY 檔案…> [--json out.json]
    tools/py.sh tools/text_extract.py code <CODE 檔案…> [--json out.json]
    tools/py.sh tools/text_extract.py build <原版目錄> [--text text]   # 產生／合併 text/*.json（docs/spec/007）
    tools/py.sh tools/text_extract.py check <原版目錄> [--text text]   # 原文逐筆可重建
    tools/py.sh tools/text_extract.py stats [--text text]

格式（docs/re/015 §2）：以 16 bytes 為一列。
- 訊息：32 bytes ＝ 31 字元文字 ＋ 1 byte 類型碼（'0'、'p'、'P' 等）。
- 選單：32 bytes 提問（類型碼 'q'、'Q'、'1'、'2'…）後接選項列，每列 16 bytes ＝ 6 bytes 條件區（常見 2 bytes 條件＋4 bytes 0，也見過兩組條件）＋ 10 字元標籤；
  以「00 00 00 ＋ 13 個空白」（或只有空白與 00 的列）結束。選項前綴的 4 bytes 也見過空白（I_MENUH 第一個選單）；
  標籤中間出現 00 時截斷（I_MENU00 的 Cancel）。
I_ENMY：80 bytes 一筆，偏移 2 起 10 字元名字（其餘是數值，未解）。
CODE：腳本位元組碼；指令 81h 後接內嵌字串到 00 為止，字串裡可夾 10h–1Fh 的顏色控制碼（推定：以實跑印字追蹤核對，docs/re/015）。
字碼 20h–7Eh 是 ASCII；80h 以上是遊戲自訂字模（地名等），以 {XX} 表示。
⚠ 抽出的原文是原版素材，輸出只放 workplace/（gitignore），不進版控。
"""
import hashlib
import json
import pathlib
import struct
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import unexepack  # noqa: E402

MENU_TYPES = b"qQ123456789"
END = b"\x00\x00\x00" + b" " * 13


def text(b):
    return "".join(chr(c) if 0x20 <= c < 0x7F else "{%02X}" % c for c in b)


def is_text(b):
    return all(0x20 <= c < 0x7F or c >= 0x80 for c in b)


def parse_menu_file(data, name):
    out, unknown, pos = [], [], 0
    while pos + 16 <= len(data):
        rec = data[pos:pos + 32]
        if len(rec) == 32 and is_text(rec[:31]) and rec[31] in MENU_TYPES:
            item = {"file": name, "offset": pos, "kind": "menu", "type": chr(rec[31]), "text": text(rec[:31]), "options": []}
            pos += 32
            while pos + 16 <= len(data):
                row = data[pos:pos + 16]
                if row == END or not row.strip(b" \x00"):  # 結束列：00 00 00 ＋空白，或只有空白與 00
                    pos += 16
                    break
                label = row[6:].split(b"\x00")[0]
                if not label or not is_text(label) or is_text(row[:6]):  # 前綴全是文字就不是選項
                    break
                item["options"].append({"offset": pos, "cond": row[:6].hex(), "text": text(label)})
                pos += 16
            out.append(item)
        elif len(rec) == 32 and is_text(rec[:31]) and rec[31] >= 0x20:
            out.append({"file": name, "offset": pos, "kind": "message", "type": chr(rec[31]), "text": text(rec[:31])})
            pos += 32
        else:
            if data[pos:pos + 16].strip(b" \x00"):
                unknown.append(pos)
            pos += 16
    return out, unknown


def parse_enmy_file(data, name):
    out = []
    for pos in range(0, len(data) - 79, 80):
        n = data[pos + 2:pos + 12]
        if is_text(n.rstrip(b"\x00")) and n.strip(b" \x00"):
            out.append({"file": name, "offset": pos + 2, "kind": "enemy-name", "text": text(n.rstrip(b"\x00"))})
    return out, []


def parse_code_file(data, name):
    out, pos = [], 0
    while True:
        pos = data.find(b"\x81", pos)
        if pos < 0:
            break
        end = data.find(b"\x00", pos + 1)
        body = data[pos + 1:end] if end > 0 else b""
        letters = sum(1 for c in body if 0x41 <= c <= 0x7A)
        if len(body) >= 2 and letters >= 2 and all(0x10 <= c < 0x7F or c >= 0x80 for c in body):
            shown = "".join("{%02X}" % c if c < 0x20 or c >= 0x7F else chr(c) for c in body)
            out.append({"file": name, "offset": pos + 1, "kind": "inline", "text": shown})
            pos = end
        pos += 1
    return out, []


PW_EXE_SHA256 = "88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49"
PW_UNP_SHA256 = "fd5115b91f014c47a1fb1a6e2ecfd645e293264fe614a4b350e92637cad2fbd9"
MENU_FILES = ["I_MENUH.BIN"] + ["I_MENU%02d.BIN" % i for i in range(12)]
ENMY_FILES = ["I_ENMY%02d.BIN" % i for i in range(12)]
MAP_FILES = ["I_MAP%02d.BIN" % i for i in range(12)]
CODE_FILES = ["CODEH.BIN"] + ["CODE%d.BIN" % i for i in range(12)]
CODE_WINDOW = {"CODEH.BIN": 0x1200}  # 其餘 CODEnn 0x700（docs/spec/007 §3.3：後載入的 I_MENUnn／FONT.BIN 蓋掉尾端）
SEG_BASE = 0x510  # PW_UNP.EXE 映像內，執行期段 0161 的起點（IDA 10510h）

# docs/spec/007 §3.4：（位址, 種類, 長度（None ＝ 到 00）, 行寬, 字型, 簽章）
# 簽章：("call", 位置, 目標) ＝ 位置是 BB <位址> E8 <到目標>；("bytes", 位置, hex)；("after", 位置, 目標) ＝ 位置是 E8 <到目標>、字串緊接其後
EXE_ITEMS = (
    [(a, "block", 160, [20] * 8, "font6", ("call", s, 0x0788)) for a, s in ((0x0287, 0x01D7), (0x0327, 0x0204), (0x03C7, 0x0228))]
    + [(0x0467, "line", None, None, "font6", ("bytes", 0x0251, "bb6704")),
       (0x0495, "line", None, None, "font6", ("bytes", 0x025D, "bb9504"))]
    + [(0x0BD7 + 20 * i, "line", 20, None, "font6", ("bytes", 0x0B50, "bbd70b")) for i in range(47)]
    + [(a, "block", 60, [20] * 3, "font6", ("call", s, 0xB016)) for a, s in (
        (0x65E5, 0x642D), (0x6621, 0x643F), (0x656D, 0x64E3), (0x65A9, 0x6560), (0x89B3, 0x8942), (0x8A4D, 0x89FA),
        (0x8AEF, 0x8ACE), (0xB07D, 0xB06E))]
    + [(0x665D, "block", 60, [20] * 3, "font6", ("bytes", 0x6403, "bb5d66")),
       (0x6699, "block", 60, [20] * 3, "font6", ("bytes", 0x6409, "bb9966"))]
    + [(a, "line", 20, None, "font6", ("call", s, 0xB058)) for a, s in (
        (0x8851, 0x8844), (0x88A8, 0x889B), (0x88D1, 0x88C4), (0x88FA, 0x88ED), (0x8923, 0x8916))]
    + [(0x66F7 + 8 * i, "names", 8, None, "font6", ("bytes", 0x6511, "81c3f766")) for i in range(11)]
    + [(0x674F + 8 * i, "names", 8, None, "font6", ("bytes", 0x6522, "81c24f67")) for i in range(7)]
    + [(a, "inline", None, None, "font8", ("call", s, 0x62AF)) for a, s in (
        (0x08D7, 0x0888), (0x0901, 0x0895), (0x092B, 0x08A2), (0x0955, 0x08C3), (0x097F, 0x08D0),
        (0x0A40, 0x0A07), (0x0A6A, 0x0A14), (0x0A94, 0x0A21))]
    + [(0x4AC3, "inline", None, None, "font8", ("after", 0x4AC0, 0x6294)),
       (0x4AFB, "inline", None, None, "font8", ("after", 0x4AF8, 0x6294))]
)


class Fail(Exception):
    pass


def sha(b):
    return hashlib.sha256(b).hexdigest()


def translatable(b):
    return any(0x21 <= c < 0x7F or c >= 0x80 for c in b)


def entry(key, kind, font, raw, lines=None, **extra):
    e = {"key": key, "kind": kind, "font": font, "width": len(raw)}
    if lines:
        e["lines"] = lines
    e.update(extra)
    e.update({"translatable": translatable(raw), "reachable": True, "same_as": "", "original": text(raw), "translation": "", "note": ""})
    return e


def extract_menu(data, name):
    items, unknown = parse_menu_file(data, name)
    out = []
    for it in items:
        key = "%s:%04X" % (name, it["offset"])
        raw = data[it["offset"]:it["offset"] + 31]
        out.append(entry(key, it["kind"], "font8", raw, [16, 15], type=it["type"]))
        for op in it.get("options", []):
            lab = data[op["offset"] + 6:op["offset"] + 16].split(b"\x00")[0]
            e = entry("%s:%04X" % (name, op["offset"]), "option", "font8", lab, menu=key)
            e["width"] = 10  # 標籤欄 10 字，遇 00 截斷
            out.append(e)
    return out, unknown


def extract_enmy(data, name):
    out = []
    for pos in range(0, len(data) - 79, 80):
        raw = data[pos + 2:pos + 12].split(b"\x00")[0]
        if raw.strip(b" "):
            e = entry("%s:%04X" % (name, pos + 2), "enemy-name", "font8", raw)
            e["width"] = 10
            out.append(e)
    return out, []


def extract_map(data, name):
    """地點名稱表：偏移 200h 起 64 × 8 bytes（docs/spec/007 §3.5）。"""
    out = []
    for i in range(64):
        pos = 0x200 + 8 * i
        raw = data[pos:pos + 8].split(b"\x00")[0]
        if raw.strip(b" "):
            e = entry("%s:%04X" % (name, pos), "place", "font8", raw, place=i)
            e["width"] = 8
            out.append(e)
    return out, []


def extract_code(data, name):
    out, bad, seen = [], [], {}
    for pos, c in enumerate(data):
        if c != 0x81:
            continue
        end = data.find(b"\x00", pos + 1)
        body = data[pos + 1:end] if end >= 0 else b""
        if end < 0 or end - pos - 1 > 80 or not all(0x10 <= x < 0x7F or x >= 0x80 for x in body):
            bad.append(pos)
            continue
        e = entry("%s:%04X" % (name, pos + 1), "inline", "font8", body)
        e["reachable"] = pos + 1 < CODE_WINDOW.get(name, 0x700)
        if body in seen and translatable(body):
            e["same_as"] = seen[body]
        seen.setdefault(body, e["key"])
        out.append(e)
    return out, bad


def check_sig(code, sig):
    kind, at, want = sig
    if kind == "bytes":
        return code[at:at + len(want) // 2].hex() == want
    if kind == "after":
        return code[at] == 0xE8 and (at + 3 + struct.unpack_from("<h", code, at + 1)[0]) & 0xFFFF == want
    return code[at] == 0xBB and code[at + 3] == 0xE8 and (at + 6 + struct.unpack_from("<h", code, at + 4)[0]) & 0xFFFF == want


def extract_exe(pw_exe):
    if sha(pw_exe) != PW_EXE_SHA256:
        raise Fail("PW.EXE 的 SHA-256 不符（docs/spec/007 §2），位址表不適用")
    unp = unexepack.build_mz(unexepack.unpack(pw_exe))
    if sha(unp) != PW_UNP_SHA256:
        raise Fail("解壓後的 PW_UNP.EXE SHA-256 不符（tools/unexepack.py 變了？）")
    hdr = struct.unpack_from("<H", unp, 8)[0] * 16
    code = unp[hdr + SEG_BASE:hdr + SEG_BASE + 0x10000]
    out, bad = [], []
    for addr, kind, n, lines, font, sig in EXE_ITEMS:
        # 簽章裡的位址要真的是這一則（call／bytes 形式：BB 後面的立即數；names、after 另外核對）
        ok = check_sig(code, sig)
        if ok and sig[0] == "call":
            ok = struct.unpack_from("<H", code, sig[1] + 1)[0] == addr
        if ok and sig[0] == "after":
            ok = addr == sig[1] + 3
        if not ok:
            bad.append("cs:%04X（簽章 %s @ %04X）" % (addr, sig[0], sig[1]))
            continue
        raw = code[addr:code.index(0, addr)] if n is None else code[addr:addr + n]
        out.append(entry("PW.EXE:cs:%04X" % addr, kind, font, raw, lines))
    if bad:
        raise Fail("PW_UNP.EXE 位元組簽章不符：" + "、".join(bad))
    return out


def extract_all(orig):
    orig = pathlib.Path(orig)
    need = ["PW.EXE"] + MENU_FILES + ENMY_FILES + MAP_FILES + CODE_FILES
    missing = [f for f in need if not (orig / f).is_file()]
    if missing:
        raise Fail("原版目錄 %s 缺檔：%s" % (orig, "、".join(missing)))
    sources = {"PW.EXE": (sha((orig / "PW.EXE").read_bytes()), extract_exe((orig / "PW.EXE").read_bytes()), [])}
    for files, fn in ((MENU_FILES, extract_menu), (ENMY_FILES, extract_enmy), (MAP_FILES, extract_map), (CODE_FILES, extract_code)):
        for f in files:
            data = (orig / f).read_bytes()
            items, unknown = fn(data, f)
            sources[f] = (sha(data), items, unknown)
    return sources


def pinned_hashes(text_dir):
    p = pathlib.Path(text_dir) / "sources.json"
    return json.loads(p.read_text(encoding="utf-8"))["sha256"] if p.exists() else {}


def cmd_build(orig, text_dir):
    sources = extract_all(orig)
    pins = pinned_hashes(text_dir)
    wrong = [f for f, (h, _, _) in sources.items() if f in pins and pins[f] != h]
    if wrong:
        raise Fail("來源檔的 SHA-256 與 text/sources.json 不符：" + "、".join(wrong))
    unknown = {f: u for f, (_, _, u) in sources.items() if u}
    if unknown:
        raise Fail("有判讀不了的位置：" + "；".join("%s %s" % (f, " ".join("%04X" % x for x in u[:8])) for f, u in unknown.items()))
    d = pathlib.Path(text_dir)
    d.mkdir(parents=True, exist_ok=True)
    total = 0
    for f, (h, items, _) in sources.items():
        path = d / (f + ".json")
        old = {e["key"]: e for e in json.loads(path.read_text(encoding="utf-8"))["entries"]} if path.exists() else {}
        new_keys = set()
        for e in items:
            new_keys.add(e["key"])
            if e["key"] in old:
                e["translation"] = old[e["key"]].get("translation", "")  # 譯文與備註以 key 保留
                e["note"] = old[e["key"]].get("note", "")
        for k, e in old.items():
            if k not in new_keys:
                e["orphan"] = True
                items.append(e)
        path.write_text(json.dumps({"schema": "psychic-war-text/1", "source": f, "sha256": h, "entries": items},
                                   ensure_ascii=False, indent=1) + "\n", encoding="utf-8")
        kinds = {}
        for e in items:
            kinds[e["kind"]] = kinds.get(e["kind"], 0) + 1
        total += len(items)
        print("%-14s %4d 則  %s" % (f, len(items), "、".join("%s %d" % kv for kv in sorted(kinds.items()))))
    (d / "sources.json").write_text(json.dumps({"schema": "psychic-war-text-sources/1",
                                                "sha256": {f: h for f, (h, _, _) in sources.items()}},
                                               ensure_ascii=False, indent=1) + "\n", encoding="utf-8")
    print("合計 %d 則" % total)


def load_text(text_dir):
    out = {}
    for p in sorted(pathlib.Path(text_dir).glob("*.json")):
        doc = json.loads(p.read_text(encoding="utf-8"))
        if doc.get("schema") != "psychic-war-text/1":
            continue  # sources.json、glossary.json 等不是文本檔
        for e in doc["entries"]:
            out[e["key"]] = e
    return out


def cmd_check(orig, text_dir):
    sources = extract_all(orig)
    have = load_text(text_dir)
    problems = []
    fresh = {}
    for _, items, _ in sources.values():
        for e in items:
            fresh[e["key"]] = e
    for k, e in fresh.items():
        if k not in have:
            problems.append("%s：text/ 缺這一則" % k)
        elif have[k]["original"] != e["original"]:
            problems.append("%s：原文與重新抽取不同" % k)
    for k, e in have.items():
        if k not in fresh and not e.get("orphan"):
            problems.append("%s：原版沒有這一則（沒標 orphan）" % k)
    for p in problems[:20]:
        print(p)
    if problems:
        raise Fail("check 不通過：%d 處" % len(problems))
    print("check 通過：%d 則原文可由原版重建" % len(fresh))


def to_translate(have):
    """要翻的則：有文字、在載入視窗內、不是別則的重複、不是孤兒（docs/spec/007 §5）。"""
    return [e for e in have.values() if e["translatable"] and e.get("reachable", True) and not e["same_as"] and not e.get("orphan")]


def cmd_stats(text_dir):
    have = load_text(text_dir)
    tr = to_translate(have)
    done = [e for e in tr if e["translation"]]
    print("總則數 %d、要翻 %d、已翻 %d（%.1f%%）、不在載入視窗 %d、重複 %d、孤兒 %d" % (
        len(have), len(tr), len(done), 100.0 * len(done) / len(tr) if tr else 0,
        sum(1 for e in have.values() if not e.get("reachable", True)), sum(1 for e in have.values() if e["same_as"]),
        sum(1 for e in have.values() if e.get("orphan"))))


def main(argv):
    text_dir = "text"
    if "--text" in argv:
        i = argv.index("--text")
        text_dir = argv[i + 1]
        del argv[i:i + 2]
    try:
        if argv[:1] == ["build"] and len(argv) == 2:
            cmd_build(argv[1], text_dir)
            return 0
        if argv[:1] == ["check"] and len(argv) == 2:
            cmd_check(argv[1], text_dir)
            return 0
        if argv == ["stats"]:
            cmd_stats(text_dir)
            return 0
    except Fail as e:
        print("tools/text_extract.py：%s" % e, file=sys.stderr)
        return 1
    if len(argv) < 2 or argv[0] not in ("menu", "enmy", "code"):
        print(__doc__)
        return 2
    out_json = None
    if "--json" in argv:
        i = argv.index("--json")
        out_json = argv[i + 1]
        del argv[i:i + 2]
    everything, total_unknown = [], 0
    for f in argv[1:]:
        p = pathlib.Path(f)
        data = p.read_bytes()
        items, unknown = {"menu": parse_menu_file, "enmy": parse_enmy_file, "code": parse_code_file}[argv[0]](data, p.name)
        everything += items
        total_unknown += len(unknown)
        msgs = sum(1 for i in items if i["kind"] == "message")
        menus = [i for i in items if i["kind"] == "menu"]
        opts = sum(len(m["options"]) for m in menus)
        print(f"{p.name}: {len(data)} bytes，訊息 {msgs}、選單 {len(menus)}（選項 {opts}），解不出的非空列 {len(unknown)}"
              + (f"：{', '.join('%04X' % u for u in unknown[:8])}{' …' if len(unknown) > 8 else ''}" if unknown else ""))
    if out_json:
        json.dump(everything, open(out_json, "w", encoding="utf-8"), ensure_ascii=False, indent=1)
    print(f"合計 {len(everything)} 筆，解不出的非空列 {total_unknown}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
