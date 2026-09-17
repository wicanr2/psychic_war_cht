"""文字抽取雛形（issue #16，docs/re/015）：從玩家自備的 I_MENUnn.BIN／I_MENUH.BIN 列出所有字串與偏移。

    tools/py.sh tools/text_extract.py menu <I_MENU 檔案…> [--json out.json]
    tools/py.sh tools/text_extract.py enmy <I_ENMY 檔案…> [--json out.json]
    tools/py.sh tools/text_extract.py code <CODE 檔案…> [--json out.json]

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
import json
import pathlib
import sys

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


def main(argv):
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
