"""文字覆蓋率（issue #19，docs/spec/007 §6）：實跑印出的文字換算成文本 key，輸出四個數字。

    tools/py.sh tools/text_coverage.py <紀錄目錄> [--text text] [--json out.json]

紀錄目錄由 tools/text_coverage.sh 產生（每段一份 .log，加 replay.json）。
「觸發了卻不在文本檔」不為 0 時結束碼 1。
"""
import collections
import json
import pathlib
import re
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import text_extract as te  # noqa: E402

TEXT_SOURCES = re.compile(r"^(I_MENU(H|\d\d)|I_ENMY\d\d|CODE(H|\d+))\.BIN$", re.I)
CS_LIN = 0x1610  # 執行期段 0161 的線性起點
ENEMY_BUF = 0x3A4C  # 戰鬥時從 I_ENMY 複製來的名稱（cs:3A4A 起 80 bytes，名稱在 +2）
B07D_LINE1 = range(0xB07D, 0xB07D + 20)
# 執行期緩衝區：腳本把數字轉成字串的地方（實跑只見到數字與空白）
RUNTIME_DS = [(0x16816, 0x16916, "腳本緩衝區（數字）")]
# 小字型：返回位址 → 來源指標怎麼取（docs/spec/007 §6.2）
SMALL_PTR = {"B02A": "bx", "07A0": "bx", "07D1": "bx", "653A": "bx", "0BB1": "bx-1", "1259": "bx-1", "654E": "dx", "B055": "input"}

READ = re.compile(r"#(\d+)\s+(\S+)\s+handle=\w+ → \w+:\w+ 要 \d+ 得 (\d+)（線性 (\w+)–\w+）")
REGS_HEAD = re.compile(r"^0161:(\w{4}) 執行時的暫存器")
REGS = re.compile(r"#(\d+) AX=(\w{4}) BX=(\w{4}) CX=(\w{4}) DX=\w{4} SI=\w{4} DI=\w{4} BP=\w{4} DS=(\w{4})")
CALL = re.compile(r"#(\d+)\s+由 \w{4}:(\w{4}) .*AX=(\w{4}) BX=(\w{4}) CX=\w{4} DX=(\w{4})")


def show(chars):
    return te.text(bytes(chars))


def has_letters(chars):
    return any(0x41 <= c <= 0x5A or 0x61 <= c <= 0x7A or c >= 0x80 for c in chars)


def parse_log(path):
    """回 (讀檔事件, FONT.BIN 字串命中, 小字型命中)，都依步數排序。"""
    reads, strs, small = [], [], []
    section, cur = None, None
    for line in path.read_text(encoding="utf-8").splitlines():
        if line.startswith("讀檔（只列"):
            section = "reads"
            continue
        m = REGS_HEAD.match(line)
        if m:
            section, cur = "regs", m.group(1).upper()
            continue
        if line.startswith("0161:B0F1 被呼叫"):
            section = "call"
            continue
        if not line.startswith("  "):
            section = None if line.strip() else section
            continue
        if section == "reads":
            m = READ.search(line)
            if m:
                reads.append((int(m.group(1)), m.group(2).upper(), int(m.group(4), 16), int(m.group(3))))
        elif section == "regs":
            m = REGS.search(line)
            if m:
                strs.append((int(m.group(1)), cur, int(m.group(2), 16) & 0xFF, int(m.group(3), 16), int(m.group(4), 16), int(m.group(5), 16)))
        elif section == "call":
            m = CALL.search(line)
            if m:
                small.append((int(m.group(1)), m.group(2).upper(), int(m.group(3), 16) & 0xFF, int(m.group(4), 16), int(m.group(5), 16)))
    strs.sort()
    return sorted(reads), strs, small


def merge_strings(hits):
    """迴圈頭的逐字命中併成一次印字：回 [(步數, 常式, 起點線性, 字元)]。第 k ≥ 1 次命中的 AL 是第 k−1 個字元。"""
    out = []
    for step, rt, al, bx, cx, ds in hits:
        lin = (CS_LIN + bx) if rt == "62AF" else ds * 16 + bx
        ch = cx >> 8
        # AL ＝ 0 表示上一個字串的結尾剛被讀掉：這一次是**新的字串**的開頭。
        # 不擋的話，記憶體裡相鄰的下一個字串（起點 ＝ 上一個位址 ＋1）會被併成同一次印字，
        # 觸發則數少算（Game Over 的三行只算一則）。
        if out and al != 0:
            o = out[-1]
            if o["rt"] == rt and lin == o["last"] + 1 and (rt != "6289" or ch == o["ch"] - 1):
                o["chars"].append(al)
                o["last"], o["ch"] = lin, ch
                continue
        out.append({"step": step, "rt": rt, "start": lin, "last": lin, "ch": ch, "chars": []})
    return out


class Coverage:
    def __init__(self, text_dir):
        self.have = te.load_text(text_dir)
        self.exe = sorted((int(k.split(":")[2], 16), e) for k, e in self.have.items() if k.startswith("PW.EXE:"))
        self.triggered = collections.Counter()
        self.missing = []
        self.runtime = collections.OrderedDict()

    def root(self, key):
        e = self.have[key]
        return e["same_as"] or key

    def hit(self, key):
        self.triggered[self.root(key)] += 1

    def exe_entry(self, off):
        for addr, e in self.exe:
            if addr <= off < addr + e["width"]:
                return e
        return None

    def loaded_at(self, loads, lin):
        for step, name, start, got in reversed(loads):
            if start <= lin < start + got:
                return name, lin - start
        return None, None

    def font8(self, seg, s, loads):
        chars, lin = s["chars"], s["start"]
        if s["rt"] == "62AF":
            off = lin - CS_LIN
            if off == ENEMY_BUF:
                enmy = [n for _, n, _, _ in loads if n.startswith("I_ENMY")]
                name = show(chars).rstrip()
                for k, e in self.have.items():
                    if enmy and k.startswith(enmy[-1] + ":") and e["original"].rstrip() == name:
                        return self.hit(k)
                return self.miss(seg, s, "敵人名稱對不到最近載入的 %s" % (enmy[-1] if enmy else "I_ENMY"))
            e = self.exe_entry(off)
            if e:
                return self.hit(e["key"])
            if has_letters(chars):
                return self.miss(seg, s, "PW.EXE cs:%04X 不在文本檔" % off)
            return self.note_runtime("cs:%04X" % off, chars, "程式碼段（無文字）")
        name, off = self.loaded_at(loads, lin)
        if name and TEXT_SOURCES.match(name):
            name = name.replace(".bin", ".BIN")
            for k in (off, off - 16, off - 6):
                key = "%s:%04X" % (name.upper(), k)
                if key in self.have:
                    return self.hit(key)
            if has_letters(chars):
                return self.miss(seg, s, "%s 偏移 %04X 換算不到 key" % (name, off))
            return self.note_runtime("%s:%04X" % (name, off), chars, "文字來源內的非文字")
        for lo, hi, why in RUNTIME_DS:
            if lo <= lin < hi:
                if not has_letters(chars):
                    return self.note_runtime("lin %05X" % lin, chars, why)
                maps = [n for _, n, _, _ in loads if n.startswith("I_MAP")]
                shown = show(chars).rstrip()
                for k, e in self.have.items():
                    if maps and k.startswith(maps[-1] + ":") and e["original"].rstrip() == shown:
                        return self.hit(k)
                return self.miss(seg, s, "緩衝區內容對不到最近載入的 %s 地點名稱" % (maps[-1] if maps else "I_MAP"))
        if has_letters(chars):
            return self.miss(seg, s, "線性 %05X 不屬於任何文字來源（最近載入：%s）" % (lin, name))
        return self.note_runtime("lin %05X" % lin, chars, "未知緩衝區（無文字）")

    def small_font(self, seg, hits):
        line1 = []
        for step, ret, al, bx, dx in hits:
            how = SMALL_PTR.get(ret)
            if how is None:
                self.missing.append({"segment": seg, "step": step, "where": "B0F1 由 %s" % ret, "text": show([al]), "why": "未知的小字型呼叫端"})
                continue
            if how == "input":
                self.note_runtime("B055", [al], "輸入回顯與選擇游標")
                continue
            ptr = {"bx": bx, "bx-1": bx - 1, "dx": dx}[how]
            if ptr in B07D_LINE1:
                line1.append((ptr, al, step))
                if ptr == B07D_LINE1[-1]:
                    self.b07d_line(seg, line1)
                    line1 = []
                continue
            e = self.exe_entry(ptr)
            if e:
                self.hit(e["key"])
            elif al != 0x20:
                self.missing.append({"segment": seg, "step": step, "where": "cs:%04X" % ptr, "text": show([al]), "why": "小字型指標不在文本檔"})

    def b07d_line(self, seg, line1):
        shown = show([al for _, al, _ in line1])
        for addr, e in self.exe:
            if e["kind"] == "line" and e["width"] == 20 and e["original"] == shown:
                return self.hit(e["key"])
        if shown.strip():
            self.missing.append({"segment": seg, "step": line1[0][2], "where": "cs:B07D 第 1 行", "text": shown, "why": "對不到 20 bytes 行"})

    def miss(self, seg, s, why):
        self.missing.append({"segment": seg, "step": s["step"], "where": "%s @ %05X" % (s["rt"], s["start"]), "text": show(s["chars"]), "why": why})

    def note_runtime(self, where, chars, why):
        k = (where, why)
        t = show(chars)
        self.runtime.setdefault(k, set()).add(t)


def main(argv):
    if not argv:
        print(__doc__)
        return 2
    text_dir, out_json = "text", None
    if "--text" in argv:
        i = argv.index("--text")
        text_dir = argv[i + 1]
        del argv[i:i + 2]
    if "--json" in argv:
        i = argv.index("--json")
        out_json = argv[i + 1]
        del argv[i:i + 2]
    logdir = pathlib.Path(argv[0])
    replay = json.loads((logdir / "replay.json").read_text(encoding="utf-8"))
    cov = Coverage(text_dir)
    final_loads = {}
    for s in replay["segments"]:
        log = logdir / (s["name"] + ".log")
        if not log.exists():
            print("缺紀錄：%s" % log, file=sys.stderr)
            return 2
        reads, strs, small = parse_log(log)
        loads = list(final_loads.get(s["from"], []))
        events = [(st, "read", (st, n, lin, got)) for st, n, lin, got in reads]
        events += [(x["step"], "str", x) for x in merge_strings(strs)]
        for step, kind, x in sorted(events, key=lambda e: (e[0], e[1] != "read")):
            if kind == "read":
                loads.append(x)
            else:
                cov.font8(s["name"], x, loads)
        cov.small_font(s["name"], small)
        final_loads[s["name"]] = loads

    todo = te.to_translate(cov.have)
    todo_keys = {e["key"] for e in todo}
    done = [e for e in todo if e["translation"]]
    trig = set(cov.triggered)
    trig_todo = trig & todo_keys
    result = {
        "to_translate": len(todo), "translated": len(done),
        "triggered": len(trig), "triggered_to_translate": len(trig_todo),
        "triggered_not_in_text": len(cov.missing),
        "triggered_keys": sorted(trig),
        "missing": cov.missing,
        "runtime": [{"where": w, "why": y, "texts": sorted(t)} for (w, y), t in cov.runtime.items()],
    }
    print("要翻 %d、已翻 %d、實跑觸發 %d 則（其中要翻 %d）、觸發了卻不在文本檔 %d" % (
        result["to_translate"], result["translated"], result["triggered"], result["triggered_to_translate"], result["triggered_not_in_text"]))
    by_src = collections.Counter(k.split(":")[0] for k in trig)
    print("觸發的來源：" + "、".join("%s %d" % kv for kv in sorted(by_src.items())))
    print("執行期產生（不列入文本）：")
    for r in result["runtime"]:
        print("  %-22s %-16s %s" % (r["where"], r["why"], " | ".join(r["texts"][:6]) + (" …" if len(r["texts"]) > 6 else "")))
    for m in cov.missing[:40]:
        print("不在文本檔：%(segment)s #%(step)d %(where)s「%(text)s」%(why)s" % m)
    if out_json:
        pathlib.Path(out_json).write_text(json.dumps(result, ensure_ascii=False, indent=1) + "\n", encoding="utf-8")
    return 1 if cov.missing else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
