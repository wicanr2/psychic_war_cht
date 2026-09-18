#!/usr/bin/env bash
# F3 自動地圖的實跑驗收（docs/spec/015 §4 第 2、3 項）。
#
#   tools/frontend-map-check.sh [狀態檔] [走幾步]     # 預設 states/07-first-play.state、3 步
#
# 流程：Xvfb 裡從狀態檔啟動 → 往前走 N 步 → 按 F3 截圖 → 再按 F3 關掉 → F10 存檔。
# 比對（tools/map_check.py）：
#   目前格子（由存檔讀出的 X、Y）在畫面上是白色；走過的格子是灰色；沒走過的格子是黑色（反向對照）。
# 記憶體：開關 F3 前後的 lin:16966 起 52 bytes 相同（用 F10 存檔 ＋ probe 比）。
# 產出 workplace/automap/：map.png、開關前後的記憶體傾印、result.txt
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE="${1:-states/07-first-play.state}"
STEPS="${2:-3}"
for b in psychicwar probe; do [[ -x "$ROOT/workplace/bin/$b" ]] || { echo "先 build workplace/bin/$b" >&2; exit 2; }; done
OUT=workplace/automap
rm -rf "$ROOT/$OUT"; mkdir -p "$ROOT/$OUT/saves"

PSYCHICWAR_TIMEOUT=10m PSYCHICWAR_SH="
set -eu
cd /src
workplace/bin/psychicwar -orig /orig/psychic-war -audio null -scratch $OUT/saves -load-state workplace/$STATE -text text -quit-after 90s > $OUT/frontend.log 2>&1 &
PID=\$!
for i in \$(seq 1 100); do W=\$(xdotool search --name 'Psychic War' 2>/dev/null | head -1 || true); [ -n \"\$W\" ] && break; sleep 0.1; done
sleep 5
for i in \$(seq 1 $STEPS); do xdotool keydown Up; sleep 0.15; xdotool keyup Up; sleep 3; done
xdotool key F10; sleep 3; cp $OUT/saves/quick.state $OUT/saves/before.state
xdotool key F3; sleep 2
import -window \"\$W\" -define png:color-type=2 -depth 8 PNG24:$OUT/map.png
xdotool key F3; sleep 2
xdotool key F10; sleep 3; cp $OUT/saves/quick.state $OUT/saves/after.state
cp $OUT/saves/quick.map.json $OUT/quick.map.json
kill \$PID; wait \$PID 2>/dev/null || true
" "$ROOT/tools/go-ebiten.sh"

cd "$ROOT"
for w in before after; do
  timeout 10m docker run --rm --network none --memory 2g --cpus 1 --pids-limit 64 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp \
    -v "$ROOT:/src" -v "$ROOT/workplace/original:/orig:ro" -w /src psychicwar-go-ebiten \
    workplace/bin/probe -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war \
    -load-state "$OUT/saves/$w.state" -steps 0 -dump-mem "16966-1699A:$OUT/$w.mem" > "$OUT/$w-probe.log" 2>&1 || true
done
set +e
tools/py.sh tools/map_check.py "$OUT/map.png" "$OUT/quick.map.json" "$OUT/after.mem" | tee "$ROOT/$OUT/result.txt"
rc=$?
mem=$(tools/py.sh -c '
import pathlib, sys
a = pathlib.Path("/src/" + sys.argv[1]).read_bytes()
b = pathlib.Path("/src/" + sys.argv[2]).read_bytes()
print("相同" if a == b else "不同")' "$OUT/before.mem" "$OUT/after.mem")
echo "開關 F3 前後的觀測變數：$mem" | tee -a "$ROOT/$OUT/result.txt"
[[ "$mem" == "相同" ]] || rc=1
exit $rc
