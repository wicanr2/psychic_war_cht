#!/usr/bin/env bash
# 圖檔內嵌文字在前端的驗收（docs/spec/011 §5 第 3 項）：Ebiten 合成的畫面與原版推算的期望值逐像素相同。
#
#   tools/frontend-baked-check.sh [狀態檔]        # 預設 states/07-first-play.state
#
# 1. 參照：dosgolem cmd/step 從同一狀態跑 2 秒，存 320×200 原版畫面
# 2. 前端：Xvfb 裡從同一狀態啟動（scale 3、-audio null），等轉譯紀錄出現全部 baked key 後截圖（PNG24）
# 3. tools/overlay_check.py --baked 逐像素比
# 產出 workplace/overlay/frontend/baked/：ref.png、shot.png、text.jsonl、frontend.log、result.txt
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE="${1:-states/07-first-play.state}"
for b in psychicwar step; do [[ -x "$ROOT/workplace/bin/$b" ]] || { echo "先 build workplace/bin/$b" >&2; exit 2; }; done
OUT="workplace/overlay/frontend/baked"
rm -rf "$ROOT/$OUT"; mkdir -p "$ROOT/$OUT"
KEYS=$("$ROOT/tools/py.sh" -c 'import json; print(" ".join(e["key"] for e in json.load(open("/src/text/baked.json"))["entries"]))')

PSYCHICWAR_TIMEOUT=10m PSYCHICWAR_SH="
set -eu
cd /src
workplace/bin/step -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war -scratch /tmp/s1 -load-state workplace/$STATE -cycles 750 -do 'wait:2000' -shot $OUT/ref.png -scale 1 > $OUT/step.log 2>&1
workplace/bin/psychicwar -orig /orig/psychic-war -audio null -scratch /tmp/s2 -load-state workplace/$STATE -text text -text-log $OUT/text.jsonl -quit-after 90s > $OUT/frontend.log 2>&1 &
PID=\$!
for i in \$(seq 1 100); do W=\$(xdotool search --name 'Psychic War' 2>/dev/null | head -1 || true); [ -n \"\$W\" ] && break; sleep 0.1; done
sleep 3
ok=0
for i in \$(seq 1 300); do
  kill -0 \$PID 2>/dev/null || { echo '前端已結束'; tail -5 $OUT/frontend.log; exit 3; }
  all=1
  for key in $KEYS; do grep -F \"\\\"key\\\":\\\"\$key\\\"\" $OUT/text.jsonl >/dev/null 2>&1 || { all=0; break; }; done
  [ \$all = 1 ] && { ok=1; break; }
  sleep 0.2
done
[ \$ok = 1 ] || echo '等不到全部 baked key 的紀錄'
sleep 1
import -window \"\$W\" -define png:color-type=2 -depth 8 PNG24:$OUT/shot.png
kill \$PID; wait \$PID 2>/dev/null || true
" "$ROOT/tools/go-ebiten.sh"

set +e
"$ROOT/tools/py.sh" tools/overlay_check.py --baked "$OUT/ref.png" "$OUT/shot.png" | tee "$ROOT/$OUT/result.txt"
exit "${PIPESTATUS[0]}"
