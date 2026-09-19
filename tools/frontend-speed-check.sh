#!/usr/bin/env bash
# 速度檔位的實跑驗收（docs/spec/023 §8 第 2、3 項）。
#
#   tools/frontend-speed-check.sh [檔位…]        # 預設 1 3
#   產出 workplace/fe/speed/：idle-*.jsonl、walk-*.jsonl、battle-*.jsonl、report.txt
#
# 三個量測，都在 Xvfb 裡跑真正的前端：
#
#   idle   **不按任何鍵**跑固定秒數，比跑到的指令數。沒有輸入所以完全決定性，
#          這是核心宣稱最乾淨的量法（docs/spec/023 §8 第 3 項）：
#          迷宮（07-first-play）要差約檔位倍數，戰鬥中（08-encounter）**要一樣**。
#   walk   從 07-first-play.state 按住 Up 走迷宮。換格子時前端會寫一行量測，
#          比「每秒一行」量得出「走一格幾秒」。走到第 6 格會遭遇，之後自動降回原速。
#   battle 從 08-encounter.state 按住空白鍵打完第一場。前端在戰鬥結束時寫一行量測。
#          比的是**戰鬥區間的「機器毫秒 ÷ 牆上毫秒」要是 1.00**，不是比指令數——
#          xdotool 的按鍵落在哪個指令數有抖動，而計時器相位會讓同一場戰鬥差到 ±30%
#          （docs/re/010 §4.5），拿指令數比等於在量 xdotool。
#
# ⚠ 先 build：tools/go-ebiten.sh build -o /src/workplace/bin/psychicwar ./cmd/psychicwar
# ⚠ 先跑過 tools/states.sh（要有 07-first-play.state 與 08-encounter.state）。
# ⚠ 主機忙的時候前端追不上牆上時間，走路那一段的秒數會偏長（偏向「加速幅度看起來比較小」）。
#   通過就算數，沒通過要在低負載重量（同 docs/re/010 §4.6 的規則）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GEARS=("$@"); [[ ${#GEARS[@]} -gt 0 ]] || GEARS=(1 3)
OUT="$ROOT/workplace/fe/speed"
WALK_SEC="${PSYCHICWAR_WALK_SEC:-30}"
BATTLE_SEC="${PSYCHICWAR_BATTLE_SEC:-60}"
IDLE_SEC="${PSYCHICWAR_IDLE_SEC:-20}"

[[ -x "$ROOT/workplace/bin/psychicwar" ]] || { echo "先 build 前端到 workplace/bin/psychicwar" >&2; exit 2; }
for s in 07-first-play 08-encounter; do
  [[ -s "$ROOT/workplace/states/$s.state" ]] || { echo "缺 workplace/states/$s.state，先跑 tools/states.sh" >&2; exit 2; }
done
rm -rf "$OUT"; mkdir -p "$OUT"

# run <名稱> <狀態檔> <按住的鍵；none ＝ 不按> <檔位> <秒數>
run() {
  local name="$1" state="$2" key="$3" gear="$4" sec="$5" hold=""
  [[ "$key" == "none" ]] || hold="xdotool keydown $key"
  echo "[speed] $name 檔位 $gear（$sec 秒）" >&2
  PSYCHICWAR_TIMEOUT=15m PSYCHICWAR_SH="
set -eu
cd /src
workplace/bin/psychicwar -orig workplace/original/psychic-war -audio null -text off \
  -scratch /tmp/saves -speed $gear -load-state workplace/states/$state.state \
  -stats workplace/fe/speed/$name-$gear.jsonl -quit-after ${sec}s \
  > workplace/fe/speed/$name-$gear.log 2>&1 &
PID=\$!
for i in \$(seq 1 100); do W=\$(xdotool search --name 'Psychic War' 2>/dev/null | head -1 || true); [ -n \"\$W\" ] && break; sleep 0.1; done
[ -n \"\$W\" ] || { echo '找不到視窗'; tail -5 workplace/fe/speed/$name-$gear.log; exit 3; }
eval \"\$(xdotool getwindowgeometry --shell \"\$W\")\"
xdotool mousemove \$((X + WIDTH / 2)) \$((Y + HEIGHT / 2))
sleep 3                      # 等前端把載入到現在的機器時間丟掉、時鐘對齊
$hold                        # 按住不放：走路靠按鍵重複，攻擊本來就要按住（docs/re/009）
wait \$PID || true
" "$ROOT/tools/go-ebiten.sh" >/dev/null
}

for g in "${GEARS[@]}"; do
  run idle-maze 07-first-play none "$g" "$IDLE_SEC"
  run idle-battle 08-encounter none "$g" "$IDLE_SEC"
  run walk 07-first-play Up "$g" "$WALK_SEC"
  run battle 08-encounter space "$g" "$BATTLE_SEC"
done

{
  echo "== 不按鍵跑 $IDLE_SEC 秒（決定性；迷宮要差檔位倍數，戰鬥中要一樣）"
  "$ROOT/tools/py.sh" tools/speed_report.py idle \
    $(printf 'workplace/fe/speed/idle-maze-%s.jsonl ' "${GEARS[@]}") \
    $(printf 'workplace/fe/speed/idle-battle-%s.jsonl ' "${GEARS[@]}")
  echo
  echo "== 走路（07-first-play，按住 Up）"
  "$ROOT/tools/py.sh" tools/speed_report.py walk $(printf 'workplace/fe/speed/walk-%s.jsonl ' "${GEARS[@]}")
  echo
  echo "== 戰鬥（08-encounter，按住空白鍵）"
  "$ROOT/tools/py.sh" tools/speed_report.py battle $(printf 'workplace/fe/speed/battle-%s.jsonl ' "${GEARS[@]}")
} | tee "$OUT/report.txt"
