#!/usr/bin/env python3
"""worklist.py — 未完成項的權威是 docs/worklist.json。

    tools/py.sh tools/worklist.py verify          # 逐條問「還沒做完嗎」，有過期條目就 exit 1
    tools/py.sh tools/worklist.py render          # 產生 docs/worklist.md
    tools/py.sh tools/worklist.py issues <outdir> # 每條輸出一份 issue 內文，給 gh 建 issue 用

verify 為真＝這一條仍未完成。規則見 ~/.claude/rulebook/61。
"""
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
DATA_PATH = ROOT / "docs/worklist.json"
SCANNED = {".go", ".py", ".sh", ".md", ".json"}


def load(root=ROOT):
    return json.loads((root / "docs/worklist.json").read_text(encoding="utf-8"))


def is_test_file(f):
    return "_test." in f.name or f.name.startswith("test_")


def hit(v, root=ROOT):
    """回傳第一個命中 pattern 的產品檔（測試檔不算），沒有就回 None。"""
    expr = re.compile(v["pattern"])
    for target in v["paths"]:
        p = root / target
        if not p.exists():
            continue
        files = sorted(p.rglob("*")) if p.is_dir() else [p]
        for f in files:
            if not f.is_file() or is_test_file(f):
                continue
            if p.is_dir() and f.suffix not in SCANNED:
                continue
            if expr.search(f.read_text(encoding="utf-8", errors="ignore")):
                return str(f.relative_to(root))
    return None


def still_open(item, root=ROOT):
    v = item["verify"]
    kind = v["kind"]
    if kind == "manual":
        return True, "要人判"
    if kind == "json_len":
        field = json.loads((root / v["path"]).read_text(encoding="utf-8"))[v["field"]]
        return len(field) <= v["max"], f'{v["path"]} 的 {v["field"]} 有 {len(field)} 項'
    where = hit(v, root)
    if kind == "present":
        return bool(where), f"自承還在 {where}" if where else f'找不到 {v["pattern"]}'
    if kind == "absent":
        return not where, f"已經出現在 {where}" if where else "還沒出現"
    raise ValueError(f'{item["id"]}：不認識的 verify kind {kind!r}')


def check_schema(data):
    done_ids = set()
    for d in data.get("done", []):
        for key in ("id", "issue", "title", "evidence", "date"):
            if key not in d:
                raise ValueError(f'done {d.get("id", "?")} 缺欄位 {key}')
        done_ids.add(d["id"])
    ids = set()
    for item in data["items"]:
        for key in ("id", "milestone", "labels", "title", "body", "acceptance", "verify"):
            if key not in item:
                raise ValueError(f'{item.get("id", "?")} 缺欄位 {key}')
        if item["id"] in ids or item["id"] in done_ids:
            raise ValueError(f'id 重複：{item["id"]}')
        ids.add(item["id"])
        if item["milestone"] not in data["milestones"]:
            raise ValueError(f'{item["id"]}：milestone {item["milestone"]} 沒定義')
        for label in item["labels"]:
            if label not in data["labels"]:
                raise ValueError(f'{item["id"]}：label {label} 沒定義')
    for item in data["items"]:
        for dep in split_deps(item.get("blocked_by", "")):
            if dep not in ids and dep not in done_ids:
                raise ValueError(f'{item["id"]}：blocked_by {dep} 不存在')


def split_deps(s):
    return [d.strip() for d in s.split(",") if d.strip()]


def cmd_verify(data):
    stale = 0
    manual = 0
    for item in data["items"]:
        open_, why = still_open(item)
        if item["verify"]["kind"] == "manual":
            manual += 1
        if not open_:
            stale += 1
        mark = "仍未完成" if open_ else "**可能已完成**"
        print(f'{item["milestone"]} {item["id"]:<26} {mark:<10} {why}')
    print(f'\n共 {len(data["items"])} 條；要人判 {manual} 條；可能已完成 {stale} 條')
    return 1 if stale else 0


def issue_ref(data, item_id):
    for item in data["items"] + data.get("done", []):
        if item["id"] == item_id and item.get("issue"):
            return f'#{item["issue"]}'
    return f"`{item_id}`"


def issue_body(data, item):
    v = item["verify"]
    lines = [
        item["body"],
        "",
        "## 驗收",
        "",
        item["acceptance"],
        "",
    ]
    deps = split_deps(item.get("blocked_by", ""))
    if deps:
        lines += ["## 前置", "", "、".join(issue_ref(data, d) for d in deps), ""]
    if v["kind"] == "manual":
        sig = f'manual（{v["note"]}）'
    else:
        sig = f'`{v["kind"]}` `{v["pattern"]}` @ {", ".join(v["paths"])}（{v["note"]}）'
    lines += [
        "## 追蹤",
        "",
        f'- worklist id：`{item["id"]}`（權威在 `docs/worklist.json`）',
        f"- 完成訊號：{sig}",
    ]
    return "\n".join(lines) + "\n"


def cmd_render(data):
    out = [
        "# Worklist",
        "",
        "> 由 `tools/worklist.py render` 從 `docs/worklist.json` 產生，**不要手改**。",
        "> 條目做完就從 JSON 移走或改 verify，不是在這裡打勾。",
        "",
    ]
    for ms, desc in data["milestones"].items():
        items = [i for i in data["items"] if i["milestone"] == ms]
        out += [f"## {ms}：{desc}", "", "| issue | id | 標題 | label | 前置 | 完成訊號 |", "|---|---|---|---|---|---|"]
        for i in items:
            num = f'#{i["issue"]}' if i.get("issue") else "—"
            deps = "、".join(issue_ref(data, d) for d in split_deps(i.get("blocked_by", ""))) or "—"
            kind = i["verify"]["kind"]
            out.append(f'| {num} | `{i["id"]}` | {i["title"]} | {", ".join(i["labels"])} | {deps} | {kind} |')
        out.append("")
    done = data.get("done", [])
    if done:
        out += ["## 已完成", "", "| issue | id | 標題 | 證據 | 日期 |", "|---|---|---|---|---|"]
        for d in done:
            out.append(f'| #{d["issue"]} | `{d["id"]}` | {d["title"]} | {d["evidence"]} | {d["date"]} |')
        out.append("")
    (ROOT / "docs/worklist.md").write_text("\n".join(out), encoding="utf-8")
    print("寫出 docs/worklist.md")
    return 0


def cmd_issues(data, outdir):
    d = Path(outdir)
    d.mkdir(parents=True, exist_ok=True)
    index = []
    for item in data["items"]:
        (d / f'{item["id"]}.md').write_text(issue_body(data, item), encoding="utf-8")
        index.append("\t".join([item["id"], item["milestone"], ",".join(item["labels"]),
                                str(item.get("issue") or ""), item["title"]]))
    (d / "index.tsv").write_text("\n".join(index) + "\n", encoding="utf-8")
    print(f'寫出 {len(index)} 份 issue 內文到 {d}')
    return 0


def main(argv):
    data = load()
    check_schema(data)
    if len(argv) < 2:
        print(__doc__)
        return 2
    if argv[1] == "verify":
        return cmd_verify(data)
    if argv[1] == "render":
        return cmd_render(data)
    if argv[1] == "issues" and len(argv) == 3:
        return cmd_issues(data, argv[2])
    print(__doc__)
    return 2


if __name__ == "__main__":
    sys.exit(main(sys.argv))
