"""把錄製檔（`docs/spec/019`）轉成 probe 的 `-hold` 參數。

    tools/py.sh tools/replay_record.py <錄製檔> [--shift <事件序號>:<指令數>]

輸出一行，可以直接餵給 probe：

    -hold 'Up@138401000+3300,Space@138500000+120000'

⚠ 一定要**逗號分隔成一個旗標**：Go 的 flag 對重複的 -hold 只留一個，
給四個 -hold 只會有一組生效，而畫面上看起來就只是「重播結果不一樣」。

用 `-hold` 不用 `-press`：後者走 FIFO 佇列而且會節流，排得太密的鍵會被延後送出
（`docs/spec/003` §3），錄下來的時機就白記了。`-hold` 是定時送出、不節流。

`--shift` 把第 N 筆事件的指令數加上一個量，給反向對照用（`docs/spec/019` §5 第 2 項）：
**證明這個比對真的看得到差異**，不是兩邊都沒動所以「相同」。
"""
import json
import pathlib
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent


def main(argv):
    args = [a for a in argv if not a.startswith("--")]
    if not args:
        print(__doc__)
        return 2
    p = ROOT / args[0] if not pathlib.Path(args[0]).is_absolute() else pathlib.Path(args[0])
    doc = json.loads(p.read_text(encoding="utf-8"))
    if doc.get("schema") != "psychic-war-recording/1":
        print("schema 不是 psychic-war-recording/1：%s" % doc.get("schema"))
        return 2
    events = [dict(e) for e in doc["events"]]
    if "--shift" in argv:
        idx, delta = argv[argv.index("--shift") + 1].split(":")
        i = int(idx)
        if not 0 <= i < len(events):
            print("--shift 的事件序號 %d 超出範圍（共 %d 筆）" % (i, len(events)))
            return 2
        events[i]["step"] += int(delta)

    # 按下與放開配對成一段「按住」。同一個鍵重疊按下時以先進先出配對。
    pending, holds, unpaired = {}, [], 0
    for e in events:
        k = e["key"]
        if e["down"]:
            pending.setdefault(k, []).append(e["step"])
        else:
            q = pending.get(k) or []
            if not q:
                unpaired += 1
                continue
            start = q.pop(0)
            span = e["step"] - start
            if span <= 0:
                span = 1  # 同一步按下又放開：長度至少 1，不然 probe 會拒絕
            holds.append((start, k, span))
    for k, q in pending.items():
        unpaired += len(q)
    holds.sort()
    print("-hold '%s'" % ",".join("%s@%d+%d" % (k, s, span) for s, k, span in holds))
    if unpaired:
        print("# ⚠ 有 %d 筆事件沒有配對（按下沒放開或反過來）" % unpaired, file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
