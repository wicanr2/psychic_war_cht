"""docs/spec/007 §6 的驗收。需要玩家自備的原版（workplace/original/psychic-war），缺檔就 skip。

    tools/py.sh -m unittest tools/test_text_extract.py
"""
import json
import pathlib
import shutil
import sys
import tempfile
import unittest

ROOT = pathlib.Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "tools"))
import text_extract as te  # noqa: E402

ORIG = ROOT / "workplace/original/psychic-war"
TRACE = ROOT / "workplace/print/trace.json"
HAVE_ORIG = (ORIG / "PW.EXE").is_file()

# 執行期載入位址（docs/re/014 §1、tools/print_trace.py 的實跑紀錄）
I_MENUH_LIN = 0x13A16
I_MENU_AREA_LIN = 0x12316


@unittest.skipUnless(HAVE_ORIG, "沒有原版（workplace/original/psychic-war），跳過")
class TextExtractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.tmp = pathlib.Path(tempfile.mkdtemp())
        cls.text = cls.tmp / "text"
        te.cmd_build(ORIG, cls.text)

    @classmethod
    def tearDownClass(cls):
        shutil.rmtree(cls.tmp)

    def entries(self):
        return te.load_text(self.text)

    def test_1_counts(self):
        kinds = {}
        for e in self.entries().values():
            src = e["key"].split(":")[0]
            group = "PW.EXE" if src == "PW.EXE" else src[:6]
            kinds[(group, e["kind"])] = kinds.get((group, e["kind"]), 0) + 1
        self.assertEqual(kinds[("I_MENU", "message")], 427)
        self.assertEqual(kinds[("I_MENU", "menu")], 95)
        self.assertEqual(kinds[("I_MENU", "option")], 273)
        self.assertEqual(kinds[("I_ENMY", "enemy-name")], 60)
        self.assertEqual(kinds[("I_MAP0", "place")] + kinds[("I_MAP1", "place")], 293)
        self.assertEqual(sum(v for (g, k), v in kinds.items() if g.startswith("CODE")), 146)
        self.assertEqual(sum(v for (g, k), v in kinds.items() if g == "PW.EXE"), len(te.EXE_ITEMS))

    def test_2_hash_mismatch_stops(self):
        orig2 = self.tmp / "orig2"
        shutil.copytree(ORIG, orig2)
        data = bytearray((orig2 / "I_MENU00.BIN").read_bytes())
        data[0x40] ^= 0x01
        (orig2 / "I_MENU00.BIN").write_bytes(bytes(data))
        with self.assertRaises(te.Fail) as cm:
            te.cmd_build(orig2, self.text)
        self.assertIn("I_MENU00.BIN", str(cm.exception))
        # 反向對照：沒改的副本照樣通過
        shutil.copy(ORIG / "I_MENU00.BIN", orig2 / "I_MENU00.BIN")
        te.cmd_build(orig2, self.text)

    def test_3_check(self):
        te.cmd_check(ORIG, self.text)
        text2 = self.tmp / "text-check"
        shutil.copytree(self.text, text2)
        p = text2 / "I_MENUH.BIN.json"
        doc = json.loads(p.read_text(encoding="utf-8"))
        doc["entries"][0]["original"] += "x"
        key = doc["entries"][0]["key"]
        p.write_text(json.dumps(doc, ensure_ascii=False), encoding="utf-8")
        with self.assertRaises(te.Fail):
            te.cmd_check(ORIG, text2)

    def test_4_merge_keeps_translation(self):
        text2 = self.tmp / "text-merge"
        shutil.copytree(self.text, text2)
        p = text2 / "I_MENUH.BIN.json"
        doc = json.loads(p.read_text(encoding="utf-8"))
        target = next(e for e in doc["entries"] if e["key"] == "I_MENUH.BIN:0530")
        self.assertEqual(target["original"], "You can't move  forward here.  ")
        target["translation"] = "這裡不能往前走。"
        p.write_text(json.dumps(doc, ensure_ascii=False), encoding="utf-8")
        te.cmd_build(ORIG, text2)
        self.assertEqual(te.load_text(text2)["I_MENUH.BIN:0530"]["translation"], "這裡不能往前走。")

    def test_5_exe_signature(self):
        saved = te.EXE_ITEMS
        try:
            addr, kind, n, lines, font, sig = saved[0]
            te.EXE_ITEMS = [(addr + 1, kind, n, lines, font, sig)] + list(saved[1:])
            with self.assertRaises(te.Fail) as cm:
                te.extract_exe((ORIG / "PW.EXE").read_bytes())
            self.assertIn("cs:%04X" % (addr + 1), str(cm.exception))
        finally:
            te.EXE_ITEMS = saved

    def test_7_code_window(self):
        have = self.entries()
        self.assertFalse(have["CODEH.BIN:32B4"]["reachable"])
        self.assertTrue(have["CODEH.BIN:02B4"]["reachable"])
        beyond = [e for e in have.values() if not e.get("reachable", True)]
        self.assertTrue(all(e["kind"] == "inline" and e["key"].startswith("CODE") for e in beyond))
        roots = {have[k]["original"] for k in have if have[k].get("reachable", True)}
        self.assertTrue(all(e["original"] in roots for e in beyond), "視窗外的內容都要在視窗內出現過")

    @unittest.skipUnless(TRACE.is_file(), "沒有 workplace/print/trace.json（tools/print_trace.py report）")
    def test_6_runtime_lines_have_keys(self):
        have = self.entries()
        trace = json.loads(TRACE.read_text(encoding="utf-8"))
        lines = {(int(x["linear"], 16), x["text"], x["segment"]) for x in trace if x["routine"] == "sub_16799"}
        self.assertGreater(len(lines), 40)
        missing = []
        for lin, shown, seg in sorted(lines):
            if lin >= I_MENUH_LIN:
                src, off = "I_MENUH.BIN", lin - I_MENUH_LIN
            else:
                src, off = "I_MENU%02d.BIN" % (1 if seg == "16-sivad" else 0), lin - I_MENU_AREA_LIN
            found = False
            # 訊息第 1 行（+0）、第 2 行（+16，15 字）、選項標籤（+6）
            for start, cut in ((off, slice(0, 16)), (off - 16, slice(16, 31)), (off - 6, slice(0, 10))):
                e = have.get("%s:%04X" % (src, start))
                if e and te_text_slice(e["original"], cut).rstrip() == shown.rstrip():
                    found = True
                    break
            if not found:
                missing.append((seg, "%05X" % lin, shown))
        self.assertEqual(missing, [])


def te_text_slice(shown, cut):
    """原文用 {XX} 表示非 ASCII；依原始位元組切片後再轉回表示法。"""
    raw, i = bytearray(), 0
    while i < len(shown):
        if shown[i] == "{" and i + 3 < len(shown) and shown[i + 3] == "}":
            raw.append(int(shown[i + 1:i + 3], 16))
            i += 4
        else:
            raw.append(ord(shown[i]))
            i += 1
    return te.text(bytes(raw[cut]))


if __name__ == "__main__":
    unittest.main()
