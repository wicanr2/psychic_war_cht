"""docs/spec/007 §7 第 8 項：文字覆蓋率。需要 tools/text_coverage.sh 的實跑紀錄，缺就 skip。

    tools/py.sh -m unittest tools/test_text_coverage.py
"""
import contextlib
import io
import json
import pathlib
import shutil
import sys
import tempfile
import unittest

ROOT = pathlib.Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "tools"))
import text_coverage as tc  # noqa: E402

LOGS = ROOT / "workplace/coverage/title-to-first-save"
TEXT = ROOT / "text"


def run(text_dir, tmp):
    out = pathlib.Path(tmp) / "cov.json"
    buf = io.StringIO()
    with contextlib.redirect_stdout(buf):
        code = tc.main([str(LOGS), "--text", str(text_dir), "--json", str(out)])
    return code, json.loads(out.read_text(encoding="utf-8")), buf.getvalue()


@unittest.skipUnless((LOGS / "replay.json").is_file() and TEXT.is_dir(), "沒有覆蓋率紀錄（tools/text_coverage.sh）或 text/")
class CoverageTest(unittest.TestCase):
    def test_route_has_no_missing(self):
        with tempfile.TemporaryDirectory() as tmp:
            code, r, _ = run(TEXT, tmp)
        self.assertEqual(code, 0)
        self.assertEqual(r["triggered_not_in_text"], 0)
        for key in ("I_MENUH.BIN:09F0", "PW.EXE:cs:65E5", "CODEH.BIN:02B4", "I_MAP01.BIN:0228"):
            self.assertIn(key, r["triggered_keys"])

    def test_removed_entry_is_reported(self):
        with tempfile.TemporaryDirectory() as tmp:
            t2 = pathlib.Path(tmp) / "text"
            shutil.copytree(TEXT, t2)
            p = t2 / "I_MENUH.BIN.json"
            doc = json.loads(p.read_text(encoding="utf-8"))
            doc["entries"] = [e for e in doc["entries"] if e["key"] != "I_MENUH.BIN:09F0"]
            p.write_text(json.dumps(doc, ensure_ascii=False), encoding="utf-8")
            code, r, _ = run(t2, tmp)
        self.assertEqual(code, 1)
        self.assertGreaterEqual(r["triggered_not_in_text"], 1)
        self.assertTrue(any("09F0" in m["why"] for m in r["missing"]), r["missing"][:3])


if __name__ == "__main__":
    unittest.main()
