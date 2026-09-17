#!/usr/bin/env bash
# 決定性測試（issue #7）：從 08-encounter 狀態檔開始打第一場戰鬥。
#
#   tools/determinism.sh
#
#   A1、A2：同一步（86,500,000）按空白鍵攻擊 → 第 99,000,000 步的整個 1 MB 記憶體與畫面必須逐位元組相同
#   B     ：晚 1,000,000 步才按 → 必須與 A1 不同（反向對照：證明比對真的看得出差異）
#
# B 會不同，是因為等待按鍵的迴圈每圈都推進一次亂數（docs/re/004）。
# 需要先跑過 tools/states.sh（要有 workplace/states/08-encounter.state）。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOLEM="${PSYCHICWAR_DOSGOLEM:-$ROOT/worktrees/dosgolem}"
WP="$ROOT/workplace"
OUT="$WP/determinism"
[[ -s "$WP/states/08-encounter.state" ]] || { echo "tools/determinism.sh：先跑 tools/states.sh" >&2; exit 2; }
mkdir -p "$OUT"
rm -f "$OUT"/*.mem "$OUT"/*.frame

run() {
  local name="$1" at="$2"
  DOSGOLEM_ORIG="$WP/original" DOSGOLEM_EXTRA_MOUNT="$WP:/wp" DOSGOLEM_CPUS="${PSYCHICWAR_CPUS:-2}" \
    "$GOLEM/tools/go.sh" run ./cmd/probe -load-state /wp/states/08-encounter.state \
    -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war -steps 99000000 \
    -press space -press-at "$at" \
    -dump-mem "00000-FFFFF:/wp/determinism/$name.mem" \
    -shots "95000000:/wp/determinism/$name-95.frame" > "$OUT/$name.log" 2>&1
  [[ -s "$OUT/$name.mem" && -s "$OUT/$name-95.frame" ]] || { echo "$name 沒有產出（見 $OUT/$name.log）" >&2; exit 1; }
}
run A1 86500000
run A2 86500000
run B  87500000

h() { sha256sum "$1" | cut -c1-16; }
fail=0
if [[ "$(h "$OUT/A1.mem")" == "$(h "$OUT/A2.mem")" && "$(h "$OUT/A1-95.frame")" == "$(h "$OUT/A2-95.frame")" ]]; then
  echo "同輸入：A1 與 A2 的記憶體與戰鬥中畫面相同（$(h "$OUT/A1.mem")）"
else
  echo "✗ 同輸入卻不同：A1 $(h "$OUT/A1.mem") vs A2 $(h "$OUT/A2.mem")"; fail=1
fi
if [[ "$(h "$OUT/A1.mem")" != "$(h "$OUT/B.mem")" ]]; then
  echo "反向對照：B（晚 100 萬步按）與 A1 不同，差異 $(cmp -l "$OUT/A1.mem" "$OUT/B.mem" | wc -l) bytes"
else
  echo "✗ 反向對照失敗：B 與 A1 相同，比對可能沒有在看"; fail=1
fi
exit $fail
