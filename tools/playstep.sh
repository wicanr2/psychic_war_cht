#!/usr/bin/env bash
# 模擬人手試玩的一步（R6 第 5 項）：包 cmd/pwstep，狀態、截圖、轉譯紀錄依步數編號存在同一個試玩目錄。
#
#   tools/playstep.sh <試玩名> new "<動作>"        # 從開機開始，存成第 001 步
#   tools/playstep.sh <試玩名> next "<動作>"       # 從最後一步接著跑
#   tools/playstep.sh <試玩名> from <NNN> "<動作>" # 從第 NNN 步分岔（後面的步數會被覆蓋）
#
# 動作腳本是 dosgolem 規格 201-step-actions §2.1：tap:<鍵>[:ms]、hold:<鍵>:<ms>、wait:<ms>、type:<文字>，逗號分隔；
# 鍵名 Return、Space、Esc、Up、Down、Left、Right、Backspace、字母、數字（不分大小寫）。毫秒是機器時間（750 cycles）。
#
# 產出 workplace/playtest/<試玩名>/：NNN.state（＋.xlate.json）、NNN.png（960×600，原版放大＋中文疊字）、
#   NNN.text.jsonl（這一步的轉譯紀錄）、NNN.do（這一步的動作）、saves/（遊戲存檔）。
# 結束印出：步數、截圖路徑、pwstep 摘要、這一步的轉譯紀錄（缺譯文、過長、缺字會出現在這裡）。
# ⚠ 先 build：tools/go-ebiten.sh build -o /src/workplace/bin/pwstep ./cmd/pwstep
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NAME="${1:?試玩名}"; MODE="${2:?new／next／from}"
case "$MODE" in
  new) FROM=""; DO="${3:?動作}"; N=1 ;;
  next) DO="${3:?動作}"
    last=$(ls "$ROOT/workplace/playtest/$NAME"/[0-9][0-9][0-9].state 2>/dev/null | sort | tail -1 || true)
    [[ -n "$last" ]] || { echo "還沒有任何一步，先用 new" >&2; exit 2; }
    FROM=$(basename "$last" .state); N=$((10#$FROM + 1)) ;;
  from) FROM="${3:?步數}"; DO="${4:?動作}"; FROM=$(printf %03d $((10#$FROM))); N=$((10#$FROM + 1)) ;;
  *) echo "模式要是 new、next 或 from" >&2; exit 2 ;;
esac
[[ -x "$ROOT/workplace/bin/pwstep" ]] || { echo "先 build workplace/bin/pwstep" >&2; exit 2; }
[[ -d "$ROOT/workplace/original/psychic-war" ]] || { echo "缺原版 workplace/original/psychic-war" >&2; exit 2; }
DIR="workplace/playtest/$NAME"; mkdir -p "$ROOT/$DIR/saves"
OUT=$(printf %03d "$N")
if [[ -n "$FROM" ]]; then
  [[ -f "$ROOT/$DIR/$FROM.state" ]] || { echo "沒有 $DIR/$FROM.state" >&2; exit 2; }
  SRC=(-load-state "$DIR/$FROM.state")
else
  SRC=(-new)
fi
printf '%s\n' "$DO" > "$ROOT/$DIR/$OUT.do"
timeout 10m docker run --rm --network none --memory 1g --cpus 1 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp \
  -v "$ROOT:/src" -v "$ROOT/workplace/original:/src/workplace/original:ro" -v "$ROOT/workplace/original:/orig:ro" \
  -w /src psychicwar-go-ebiten \
  workplace/bin/pwstep -orig /orig/psychic-war "${SRC[@]}" -do "$DO" -scratch "$DIR/saves" \
    -save-state "$DIR/$OUT.state" -shot "$DIR/$OUT.png" -text-log "$DIR/$OUT.text.jsonl"
echo "第 $OUT 步（從 ${FROM:-開機}）：$DO"
echo "截圖：$DIR/$OUT.png"
echo "轉譯紀錄："
cat "$ROOT/$DIR/$OUT.text.jsonl" 2>/dev/null || true
