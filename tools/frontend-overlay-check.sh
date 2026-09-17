#!/usr/bin/env bash
# 中文疊字的前端實跑驗收（docs/spec/009 §6 第 3 項）：Ebiten 合成出來的畫面與原版推算的期望值逐像素相同。
#
#   tools/frontend-overlay-check.sh <情境> <xdotool 鍵名…>     # 例：wall Up、match space
#
# 情境取自 tools/overlay_cases.json（起始狀態、要檢查的 key）。
# 1. 參照：dosgolem cmd/step 以同一狀態、同一動作腳本跑出原版畫面（先跑 tools/overlay_run.sh <情境> 產生 ref.png 也可以）。
# 2. 前端：Xvfb 裡從同一狀態啟動（scale 3、-audio null），xdotool 送鍵（按住 0.15 秒），等轉譯紀錄出現全部目標 key 的 stamp 後 2 秒截圖。
# 3. tools/overlay_check.py 逐像素比。
# ⚠ 前端是牆上時間，時間線與 step 不同；只適合「畫完之後會停住」的畫面（訊息框、選單、說明區塊）。
# ⚠ 原版一律也掛在 /orig（狀態檔記著 /orig/psychic-war，見 tools/overlay_run.sh）。
# 產出 workplace/overlay/frontend/<情境>/：ref.png、shot.png、text.jsonl、frontend.log、result.txt
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CASE="${1:?情境名}"; shift
KEYS="$*"
for b in psychicwar step; do [[ -x "$ROOT/workplace/bin/$b" ]] || { echo "先 build workplace/bin/$b" >&2; exit 2; }; done
row=$("$ROOT/tools/py.sh" -c 'import json,sys; x=json.load(open("/src/tools/overlay_cases.json"))["cases"][sys.argv[1]]; print(x["state"]+"|"+x["do"]+"|"+" ".join(t[0] for t in x["targets"]))' "$CASE")
IFS='|' read -r state do targets <<<"$row"
OUT="workplace/overlay/frontend/$CASE"
rm -rf "$ROOT/$OUT"; mkdir -p "$ROOT/$OUT"

PSYCHICWAR_TIMEOUT=10m PSYCHICWAR_SH="
set -eu
cd /src
workplace/bin/step -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war -scratch /tmp/s1 -load-state workplace/$state -cycles 750 -do '$do' -shot $OUT/ref.png -scale 1 > $OUT/step.log 2>&1
workplace/bin/psychicwar -orig /orig/psychic-war -audio null -scratch /tmp/s2 -load-state workplace/$state -text text -text-log $OUT/text.jsonl -quit-after 120s > $OUT/frontend.log 2>&1 &
PID=\$!
for i in \$(seq 1 100); do W=\$(xdotool search --name 'Psychic War' 2>/dev/null | head -1 || true); [ -n \"\$W\" ] && break; sleep 0.1; done
eval \"\$(xdotool getwindowgeometry --shell \"\$W\")\"
xdotool mousemove \$((X + WIDTH / 2)) \$((Y + HEIGHT / 2))
sleep 2
for k in $KEYS; do xdotool keydown \$k; sleep 0.15; xdotool keyup \$k; sleep 0.5; done
ok=0
for i in \$(seq 1 600); do
  kill -0 \$PID 2>/dev/null || { echo '前端已結束'; tail -5 $OUT/frontend.log; exit 3; }
  all=1
  for key in $targets; do grep -F \"\\\"stamp\\\",\\\"key\\\":\\\"\$key\\\"\" $OUT/text.jsonl >/dev/null 2>&1 || { all=0; break; }; done
  [ \$all = 1 ] && { ok=1; break; }
  sleep 0.2
done
[ \$ok = 1 ] || echo '等不到全部目標 key 的紀錄'
sleep 2
import -window \"\$W\" -define png:color-type=2 -depth 8 PNG24:$OUT/shot.png
kill \$PID; wait \$PID 2>/dev/null || true
" "$ROOT/tools/go-ebiten.sh"

set +e
"$ROOT/tools/py.sh" tools/overlay_check.py "$CASE" "$OUT/ref.png" "$OUT/shot.png" | tee "$ROOT/$OUT/result.txt"
exit "${PIPESTATUS[0]}"
