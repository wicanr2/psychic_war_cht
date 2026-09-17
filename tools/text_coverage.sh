#!/usr/bin/env bash
# 文字覆蓋率的實跑紀錄（issue #19，docs/spec/007 §6.1）。
#
#   tools/text_coverage.sh [重播檔]      # 預設 replay/title-to-first-save.json
#   tools/py.sh tools/text_coverage.py workplace/coverage/<重播檔名>
#
# 用 tools/states.sh 重跑重播（--check 驗畫面雜湊，順便確認紀錄沒有改變執行），
# 每段 probe 加讀檔紀錄、三支 FONT.BIN 字串函式的暫存器、小字型單字元的呼叫端；
# 紀錄複製到 workplace/coverage/<重播檔名>/。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPLAY="${1:-replay/title-to-first-save.json}"
NAME="$(basename "$REPLAY" .json)"
OUT="$ROOT/workplace/coverage/$NAME"
PSYCHICWAR_PROBE_EXTRA='-reads-of .BIN -regs-at 0161:6289,0161:62A2,0161:62AF -regs-max 200000 -call-args 0161:B0F1:1:0:18000000000000000000 -arg-regs' \
  "$ROOT/tools/states.sh" --check "$REPLAY"
mkdir -p "$OUT"
names=$("$ROOT/tools/py.sh" -c 'import json,sys; print(" ".join(s["name"] for s in json.load(open(sys.argv[1]))["segments"]))' "$REPLAY")
for n in $names; do cp "$ROOT/workplace/states/$n.log" "$OUT/"; done
cp "$ROOT/$REPLAY" "$OUT/replay.json"
echo "紀錄在 $OUT"
