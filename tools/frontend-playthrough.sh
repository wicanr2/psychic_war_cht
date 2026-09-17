#!/usr/bin/env bash
# 前端鍵盤實測（docs/spec/006 §4 第 5 項）：Xvfb 裡從開機跑前端，用 xdotool 送真正的 X 按鍵，
# 走完重播 01–09 段的流程並打第一場戰鬥。
#
#   tools/frontend-playthrough.sh [按住空白鍵秒數]
#   PSYCHICWAR_PLAY_TEXT=0 tools/frontend-playthrough.sh   # 不開中文疊字（預設開，轉譯紀錄寫 text.jsonl）
#   產出 workplace/fe/play/：steps/NN-<鍵>.png（每一步）、checkpoints.txt（檢查點比對）、stats.jsonl、f1.txt、final.png、text.jsonl
#   勝負看 final.png（敵人消失、回到迷宮、玩家 HP 不為 0）
#
# 按鍵走 X → GLFW → Ebiten → inpututil → dosgolem KeyDown／KeyUp，與玩家在自己機器上按的是同一條路。
# 每一步先等畫面與重播檢查點（workplace/states/NN.rgb.png）相符再送鍵：750 cycles 下轉場很慢，
# 「畫面一變就送」會把轉場當成按鍵有反應（實測 Return 落在黑畫面、選單出現後字母全送在 SELECT 上）。
# ⚠ 每個鍵要「按下 → 0.15 秒 → 放開」：`xdotool key` 的按下與放開只差幾毫秒，落在 Ebiten 兩次輪詢（1/60 秒）之間，
#   inpututil 看不到這次按鍵，遊戲就收不到（實測名字欄全空）。
# ⚠ 先 build：tools/go-ebiten.sh build -o /src/workplace/bin/psychicwar ./cmd/psychicwar
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOLD="${1:-35}"
TEXTARGS="-text text -font font -text-log workplace/fe/play/text.jsonl"; TOLMIN=3000
# 檢查點是原版英文畫面；開疊字時中文區塊會多出上千個不同像素，檢查點只用來抓送鍵時機，所以容許值至少 3000
if [[ "${PSYCHICWAR_PLAY_TEXT:-1}" == "0" ]]; then TEXTARGS="-text ''"; TOLMIN=0; fi
[[ -x "$ROOT/workplace/bin/psychicwar" ]] || { echo "先 build 前端到 workplace/bin/psychicwar" >&2; exit 2; }
rm -rf "$ROOT/workplace/fe/play"; mkdir -p "$ROOT/workplace/fe/play/steps"
PSYCHICWAR_TIMEOUT=15m PSYCHICWAR_SH="
set -eu
cd /src
OUT=workplace/fe/play
workplace/bin/psychicwar -orig workplace/original/psychic-war -audio null -scratch /tmp/saves -stats \$OUT/stats.jsonl $TEXTARGS > \$OUT/frontend.log 2>&1 &
PID=\$!
for i in \$(seq 1 100); do W=\$(xdotool search --name 'Psychic War' 2>/dev/null | head -1 || true); [ -n \"\$W\" ] && break; sleep 0.1; done
eval \"\$(xdotool getwindowgeometry --shell \"\$W\")\"
xdotool mousemove \$((X + WIDTH / 2)) \$((Y + HEIGHT / 2))
t0=\$(date +%s); n=0
note() { echo \"[\$((\$(date +%s) - t0))s] \$*\"; }
shot() { import -window \"\$W\" png:- | md5sum | cut -c1-16; }
tap() { xdotool keydown \"\$1\"; sleep 0.15; xdotool keyup \"\$1\"; }
alive() { kill -0 \$PID 2>/dev/null || { note '前端已結束'; tail -5 \$OUT/frontend.log; exit 3; }; }
# 送鍵前等畫面靜止 2.5 秒（最多 60 秒）：750 cycles 下轉場很慢，選單還沒出來就送的鍵會被吃掉
quiet() { h=\$(shot); q=0; for i in \$(seq 1 240); do alive; sleep 0.25; h2=\$(shot); if [ \"\$h2\" = \"\$h\" ]; then q=\$((q + 1)); [ \$q -ge 10 ] && return 0; else q=0; h=\$h2; fi; done; note '畫面 60 秒沒有靜止'; }
wait_change() { h0=\$1; for i in \$(seq 1 20); do [ \"\$(shot)\" != \"\$h0\" ] && return 0; sleep 0.25; done; return 1; }
retries=0
# 送鍵：等畫面靜止 → 按下 0.15 秒放開 → 5 秒內畫面沒變就重送（最多 3 次，像真人再按一次）
key() { quiet; for try in 1 2 3; do h0=\$(shot); tap \"\$1\"; wait_change \"\$h0\" && break; retries=\$((retries + 1)); note \"key \$1 沒反應，重送\"; done; sleep \"\$2\"; n=\$((n + 1)); import -window \"\$W\" \$OUT/steps/\$(printf %02d \$n)-\$1.png; note \"key \$1\"; }
# 等畫面與重播檢查點相符（縮回 320×200 後不同像素 ≤ 容許值，給閃爍的游標與箭頭），最多 90 秒
wait_ref() { ref=workplace/states/\$1.rgb.png; tol=\$2; [ \$tol -lt $TOLMIN ] && tol=$TOLMIN; for i in \$(seq 1 180); do alive; import -window \"\$W\" /tmp/cur.png; convert /tmp/cur.png -sample 320x200! /tmp/cur320.png; d=\$(compare -metric AE \$ref /tmp/cur320.png null: 2>&1 || true); d=\${d%% *}; if [ \"\${d%.*}\" -le \"\$tol\" ] 2>/dev/null; then note \"畫面符合 \$1（差 \$d 像素）\"; echo \"\$1 \$d\" >> \$OUT/checkpoints.txt; return 0; fi; sleep 0.5; done; note \"等不到 \$1（最後差 \$d）\"; echo \"\$1 timeout \$d\" >> \$OUT/checkpoints.txt; }
press() { tap \"\$1\"; sleep \"\$2\"; n=\$((n + 1)); import -window \"\$W\" \$OUT/steps/\$(printf %02d \$n)-\$1.png; note \"key \$1\"; }
sleep 20
import -window \"\$W\" \$OUT/steps/00-title.png
press space 1
wait_ref 02-match 64; press Return 1
wait_ref 03-protection 2000;  # 防拷題目依亂數出，戰士與基地名稱會不同，只看版面
 press Return 1
wait_ref 04-cleared 64; press space 1
wait_ref 05-select 64; press Return 1
wait_ref 06-name 64; press k 0.5; press a 0.5; press i 0.5; press Return 1
wait_ref 07-first-play 64
# 攔截測試：F1 不送進遊戲，畫面不該變
h0=\$(shot); tap F1; sleep 2; [ \"\$(shot)\" = \"\$h0\" ] && echo 'F1 後畫面沒變' > \$OUT/f1.txt || echo 'F1 後畫面變了' > \$OUT/f1.txt
for s in 1 2 3 4 5 6; do press Up 2.5; done
note \"遭遇，按住空白鍵 $HOLD 秒\"
xdotool keydown space; sleep $HOLD; xdotool keyup space
note \"放開\"
sleep 3
import -window \"\$W\" \$OUT/final.png
# 轉向與 Esc 選單：畫面要變（方位名稱、選單框），截圖留給人看
h1=\$(shot); press Left 3; h2=\$(shot); press Escape 3; h3=\$(shot)
cp \$OUT/steps/\$(printf %02d \$((n - 1)))-Left.png \$OUT/turn.png; cp \$OUT/steps/\$(printf %02d \$n)-Escape.png \$OUT/esc.png
{ [ \"\$h1\" != \"\$h2\" ] && echo '轉向後畫面變了' || echo '轉向後畫面沒變'; [ \"\$h2\" != \"\$h3\" ] && echo 'Esc 後畫面變了' || echo 'Esc 後畫面沒變'; } > \$OUT/turn-esc.txt
kill \$PID; wait \$PID 2>/dev/null || true
note 完成
" "$ROOT/tools/go-ebiten.sh"
tail -1 "$ROOT/workplace/fe/play/stats.jsonl"
cat "$ROOT/workplace/fe/play/f1.txt" "$ROOT/workplace/fe/play/turn-esc.txt"
