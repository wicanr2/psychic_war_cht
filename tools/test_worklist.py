"""worklist.py 的正反對照：訊號在時報未完成，拿掉訊號時真的開口。

    tools/py.sh -m unittest tools/test_worklist.py
"""
import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import worklist  # noqa: E402


def item(kind, **v):
    return {"id": "x", "verify": {"kind": kind, **v}}


class VerifyTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        (self.root / "docs/re").mkdir(parents=True)

    def tearDown(self):
        self.tmp.cleanup()

    def write(self, rel, text):
        p = self.root / rel
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text(text, encoding="utf-8")

    def test_absent_title_pattern_both_directions(self):
        it = item("absent", paths=["docs/re"], pattern=r"(?m)^# \d{3}：.*印字常式")
        # 內文提到「印字常式」不算：標題才算
        self.write("docs/re/001-first.md", "# 001：初探\n\n印字常式在哪還不知道\n")
        self.assertTrue(worklist.still_open(it, self.root)[0])
        self.write("docs/re/002-print.md", "# 002：印字常式\n")
        self.assertFalse(worklist.still_open(it, self.root)[0])

    def test_missing_path_is_open(self):
        it = item("absent", paths=["no/such"], pattern="x")
        self.assertTrue(worklist.still_open(it, self.root)[0])

    def test_test_files_do_not_close_items(self):
        it = item("absent", paths=["tools"], pattern="save-state")
        self.write("tools/test_states.py", "save-state")
        self.assertTrue(worklist.still_open(it, self.root)[0])
        self.write("tools/states.sh", "probe -save-state 1:a")
        self.assertFalse(worklist.still_open(it, self.root)[0])

    def test_present_both_directions(self):
        it = item("present", paths=["cmd"], pattern="還沒有中文")
        self.write("cmd/a.go", "// 還沒有中文\n")
        self.assertTrue(worklist.still_open(it, self.root)[0])
        self.write("cmd/a.go", "// 有了\n")
        self.assertFalse(worklist.still_open(it, self.root)[0])

    def test_single_file_path(self):
        it = item("absent", paths=["LICENSE"], pattern="RRSAL-1.0")
        self.assertTrue(worklist.still_open(it, self.root)[0])
        self.write("LICENSE", "RRSAL-1.0\n")
        self.assertFalse(worklist.still_open(it, self.root)[0])

    def test_json_len_both_directions(self):
        it = item("json_len", path="text/a.json", field="items", max=0)
        self.write("text/a.json", json.dumps({"items": []}))
        self.assertTrue(worklist.still_open(it, self.root)[0])
        self.write("text/a.json", json.dumps({"items": [1]}))
        self.assertFalse(worklist.still_open(it, self.root)[0])

    def test_manual_always_open(self):
        self.assertTrue(worklist.still_open(item("manual", note="n"), self.root)[0])


class RepoWorklistTest(unittest.TestCase):
    def test_schema_valid(self):
        worklist.check_schema(worklist.load())

    def test_no_item_closed_by_current_tree(self):
        # 剛建立時每一條都應該未完成；有條目在現在的樹上被判完成＝pattern 寫壞了
        data = worklist.load()
        closed = [i["id"] for i in data["items"] if not worklist.still_open(i)[0]]
        self.assertEqual(closed, [], f"pattern 在現在的樹上誤中：{closed}")


if __name__ == "__main__":
    unittest.main()
