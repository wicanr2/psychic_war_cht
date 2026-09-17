"""印字追蹤（issue #15，docs/re/014）：重播實跑時，遊戲印出了哪些字串。

    tools/py.sh tools/print_trace.py plan   # 讀 workplace/states/<段>.log 的 -regs-at 紀錄，寫第二階段參數檔
    tools/py.sh tools/print_trace.py report # 讀第二階段傾印，輸出 workplace/print/trace.json 與摘要

流程（tools/print_trace.sh 串起來）：
1. states.sh 加 `-regs-at 0161:6289,0161:62A2,0161:62AF -regs-max 20000`，記下三支印字串函式每次被呼叫的步數與暫存器。
2. plan：同一段內每個（位址, 長度）在第一次呼叫的那一步傾印（sub_16799 讀 CH 個字元；另外兩支讀到 0，最多 128 bytes）。
   ⚠ 同一位址之後被改寫再印（戰鬥中的數字）時，報告沿用第一次的內容；翻譯對象是固定文字，不影響字串清單。
3. states.sh 第二次跑，帶 `workplace/print/args/<段>.args`，**同時保留 -regs-at**（report 從同一份紀錄檔讀呼叫；執行是決定性的，兩次步數相同）。
4. report：解出字串（字碼 < 20h 是控制碼，以 {XX} 表示），依段列出。

三支函式（執行期 CS＝0161）：
- 0161:6289 sub_16799：ds:[BX] 起 CH 個字元（I_MENU 訊息，一行 16 字元）
- 0161:62A2 sub_167B2：ds:[BX] 起到 0
- 0161:62AF sub_167BF：cs:[BX] 起到 0（程式內嵌字串）
"""
import json
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
STATES = ROOT / "workplace/states"
OUT = ROOT / "workplace/print"
REPLAY = ROOT / "replay/title-to-first-save.json"
ROUTINES = {"6289": "sub_16799", "62A2": "sub_167B2", "62AF": "sub_167BF"}
HIT = re.compile(r"#(\d+) AX=(\w{4}) BX=(\w{4}) CX=(\w{4}) DX=\w{4} SI=\w{4} DI=\w{4} BP=\w{4} DS=(\w{4}) ES=\w{4}")


def segments():
    return [s["name"] for s in json.load(open(REPLAY, encoding="utf-8"))["segments"]]


def hits(name):
    """回 [(步數, 常式, 線性位址, 長度)]。"""
    log = STATES / f"{name}.log"
    if not log.exists():
        return []
    raw, cur = [], None
    for line in log.read_text(encoding="utf-8").splitlines():
        m = re.match(r"0161:(\w{4}) 執行時的暫存器", line)
        if m:
            cur = m.group(1)
            continue
        m = HIT.search(line)
        if m and cur in ROUTINES:
            step, bx, cx, ds = int(m.group(1)), int(m.group(3), 16), int(m.group(4), 16), int(m.group(5), 16)
            seg = 0x0161 if cur == "62AF" else ds
            n = cx >> 8 if cur == "6289" else 128
            lin = seg * 16 + bx
            # 三個位址都是字串迴圈的迴圈頭，每個字元命中一次：同一常式、位址連續加 1 的命中併成同一個字串；\n            # sub_16799 另外要 CH 剛好少 1（訊息第二行的位址緊接第一行，只看位址會併成一行）
            ch = cx >> 8
            if raw and raw[-1][1] == ROUTINES[cur] and lin == raw[-1][4] + 1 and (cur != "6289" or ch == raw[-1][5] - 1):
                raw[-1][4], raw[-1][5] = lin, ch
                continue
            raw.append([step, ROUTINES[cur], lin, n, lin, ch])
    return [(st, r, lin, n) for st, r, lin, n, _, _ in raw]


def plan():
    args = OUT / "args"
    args.mkdir(parents=True, exist_ok=True)
    total = 0
    for name in segments():
        hs = hits(name)
        f = args / f"{name}.args"
        if not hs:
            f.unlink(missing_ok=True)
            continue
        first = {}
        for st, _, lin, n in hs:  # 同一段內同一位址只傾印第一次（戰鬥中的數字字串會重印上千次）
            first.setdefault((lin, n), st)
        spec = ";".join(f"{st}:lin:{lin:X}:{n}:/wp/print/dump/{name}-{lin:X}-{n}.bin" for (lin, n), st in first.items())
        f.write_text(f"-dump-mem-at\n{spec}\n", encoding="utf-8")
        total += len(hs)
        print(f"{name}: {len(hs)} 次呼叫，{len(first)} 個不同位址")
    (OUT / "dump").mkdir(parents=True, exist_ok=True)
    print(f"共 {total} 次")


def decode(b, routine, n):
    s = []
    for i, c in enumerate(b[:n]):
        if routine != "sub_16799" and c == 0:
            break
        s.append(chr(c) if 0x20 <= c < 0x7F else "{%02X}" % c)
    return "".join(s)


def char_hits(name):
    """單字元輸出 0161:6119 的命中數（AL ≥ 20h 才畫字）。"""
    log = STATES / f"{name}.log"
    n, on = 0, False
    for line in log.read_text(encoding="utf-8").splitlines():
        if line.startswith("0161:"):
            on = line.startswith("0161:6119 ")
            continue
        m = re.search(r"#\d+ AX=\w\w(\w\w) ", line)
        if on and m and int(m.group(1), 16) >= 0x20:
            n += 1
    return n


def report():
    rows, missing = [], 0
    for name in segments():
        for st, routine, lin, n in hits(name):
            p = OUT / "dump" / f"{name}-{lin:X}-{n}.bin"
            if not p.exists():
                missing += 1
                continue
            rows.append({"segment": name, "step": st, "routine": routine, "linear": f"{lin:05X}",
                         "text": decode(p.read_bytes(), routine, n)})
    json.dump(rows, open(OUT / "trace.json", "w", encoding="utf-8"), ensure_ascii=False, indent=1)
    chars = sum(char_hits(n) for n in segments())
    printable = sum(sum(1 for ch in re.sub(r"\{..\}", "", r["text"])) for r in rows)
    print(f"單字元輸出畫字 {chars} 次；三支字串函式涵蓋的字元（以第一次傾印估）{printable}")
    uniq = sorted({(r["routine"], r["text"]) for r in rows})
    by = {}
    for r in rows:
        by[r["routine"]] = by.get(r["routine"], 0) + 1
    print(f"呼叫 {len(rows)} 次（缺傾印 {missing}），不重複字串 {len(uniq)}；各常式 {by}")
    for routine, text in uniq:
        print(f"  {routine}  {text}")


if __name__ == "__main__":
    {"plan": plan, "report": report}[sys.argv[1]]()
