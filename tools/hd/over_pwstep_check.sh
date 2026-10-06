#!/bin/sh
# Docker限定，研究038 §52；保存狀態接續，不代替視窗按鍵驗收。
set -eu
OUT="${1:-workplace/hd/over-pwstep-v1-20261002}"
BIN="${2:-workplace/hd/pwstep-over-ui-v1-20261002}"
START=workplace/hd/over-runtime-v1-20261002-scene01.state
test -f "$BIN"
test -f "$START"
test -f "$START.xlate.json"
test ! -e "$OUT"
test "$(stat -c %u workplace/hd)" = "$(id -u)"
mkdir "$OUT"
for phase in full clear; do
  ACTION=wait:1
  test "$phase" != clear || ACTION=tap:Enter:150,wait:20000
  for mode in off on; do
    THEME=""
    test "$mode" != on || THEME=workplace/hd/theme-over-v1-20261002
    LABEL="$phase-$mode"
    "$BIN" -orig /orig/psychic-war -load-state "$START" -do "$ACTION" \
      -theme "$THEME" -text text -font font -scratch "$OUT/saves-$LABEL" \
      -save-state "$OUT/$LABEL.state" -shot "$OUT/$LABEL.png" \
      -text-log "$OUT/$LABEL.text.jsonl" >"$OUT/$LABEL.stdout" 2>"$OUT/$LABEL.stderr"
    printf '%s\n' "$LABEL 完成"
  done
done
