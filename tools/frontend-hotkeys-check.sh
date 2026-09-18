#!/usr/bin/env bash
# 輔助熱鍵的實跑驗收（docs/spec/012 §5 第 2–4 項）：F2 切換語言、F1 說明頁、F10／F11 即時存檔。
#
#   tools/frontend-hotkeys-check.sh [狀態檔]        # 預設 states/07-first-play.state
#
# 流程（Xvfb 裡的前端，scale 3、-audio null）：
#   a 中文 → F2 → b 英文 → F2 → c 回中文 → F1 → d 說明頁 → F1 → e 關閉
#   → F10 存檔 → 往前走一步 → f 移動後 → F11 讀回 → g 讀檔後
# 比對：
#   b 與 cmd/step 的原版畫面放大 3 倍逐像素相同（英文模式真的沒畫疊字）
#   c、e、g 與 a 逐像素相同
#   d 與 tools/help_check.py 由 text/help.json 算出的期望值逐像素相同
#   讀檔前後的觀測變數（lin:16966:52）相同（probe 讀兩個狀態檔）
# 產出 workplace/hotkeys/：各步驟的 PNG、result.txt
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE="${1:-states/07-first-play.state}"
for b in psychicwar step; do [[ -x "$ROOT/workplace/bin/$b" ]] || { echo "先 build workplace/bin/$b" >&2; exit 2; }; done
OUT=workplace/hotkeys
rm -rf "$ROOT/$OUT"; mkdir -p "$ROOT/$OUT/saves"

PSYCHICWAR_TIMEOUT=10m PSYCHICWAR_SH="
set -eu
cd /src
workplace/bin/step -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war -scratch /tmp/s1 -load-state workplace/$STATE -cycles 750 -do 'wait:2000' -shot $OUT/ref.png -scale 3 > $OUT/step.log 2>&1
workplace/bin/psychicwar -orig /orig/psychic-war -audio null -scratch $OUT/saves -load-state workplace/$STATE -text text -text-log $OUT/text.jsonl -quit-after 120s > $OUT/frontend.log 2>&1 &
PID=\$!
for i in \$(seq 1 100); do W=\$(xdotool search --name 'Psychic War' 2>/dev/null | head -1 || true); [ -n \"\$W\" ] && break; sleep 0.1; done
eval \"\$(xdotool getwindowgeometry --shell \"\$W\")\"
xdotool mousemove \$((X + WIDTH / 2)) \$((Y + HEIGHT / 2))
shot() { import -window \"\$W\" -define png:color-type=2 -depth 8 PNG24:$OUT/\$1.png; }
sleep 5
shot a-chinese
xdotool key F2; sleep 3; shot b-english
xdotool key F2; sleep 3; shot c-back
xdotool key F1; sleep 2; shot d-help
xdotool key F1; sleep 3; shot e-closed
xdotool key F10; sleep 3
cp $OUT/saves/quick.state $OUT/saves/before.state
xdotool keydown Up; sleep 0.15; xdotool keyup Up; sleep 4; shot f-moved
xdotool key F11; sleep 4; shot g-loaded
xdotool key F10; sleep 3
cp $OUT/saves/quick.state $OUT/saves/after.state
# 反向對照：把中繼資料的 PW.EXE 雜湊改掉，F11 應該拒絕、畫面不動（docs/spec/012 §5 第 4 項）
sed -i 's/\"exe_sha256\": \"[0-9a-f]*\"/\"exe_sha256\": \"deadbeef\"/' $OUT/saves/quick.json
xdotool keydown Up; sleep 0.15; xdotool keyup Up; sleep 4; shot h-moved2
xdotool key F11; sleep 4; shot i-refused
kill \$PID; wait \$PID 2>/dev/null || true
" "$ROOT/tools/go-ebiten.sh"

cd "$ROOT"
fail=0
say() { echo "$1"; echo "$1" >> "$ROOT/$OUT/result.txt"; }
: > "$ROOT/$OUT/result.txt"
cmp_png() { # <名稱> <a.png> <b.png>
  local n d
  d=$(tools/py.sh -c '
import sys, pathlib
sys.path.insert(0, "/src/tools")
import pbl
w1, h1, a = pbl.screen_indices("/src/" + sys.argv[1])
w2, h2, b = pbl.screen_indices("/src/" + sys.argv[2])
print("尺寸不同" if (w1, h1) != (w2, h2) else sum(1 for i in range(w1 * h1) if a[i] != b[i]))' "$2" "$3")
  say "$1：不同像素 $d"
  [[ "$d" == "0" ]] || fail=1
}
cmp_png "F2 英文模式 vs 原版畫面" "$OUT/b-english.png" "$OUT/ref.png"
cmp_png "F2 切回中文 vs 切換前" "$OUT/c-back.png" "$OUT/a-chinese.png"
cmp_png "F1 關閉後 vs 開啟前" "$OUT/e-closed.png" "$OUT/c-back.png"
cmp_png "F11 讀檔後 vs 存檔時" "$OUT/g-loaded.png" "$OUT/a-chinese.png"
cmp_png "反向對照：雜湊被改過時 F11 拒絕（畫面不動）" "$OUT/i-refused.png" "$OUT/h-moved2.png"
set +e
out=$(tools/py.sh tools/help_check.py "$OUT/d-help.png"); rc=$?
set -e
say "F1 說明頁：$out"
[[ $rc -eq 0 ]] || fail=1
# 讀檔前後的觀測變數（區域、座標、朝向、地點、角色數值：lin:16966 起 52 bytes，docs/re/011）
if [[ -x "$ROOT/workplace/bin/probe" ]]; then
  for w in before after; do
    timeout 10m docker run --rm --network none --memory 2g --cpus 1 --pids-limit 64 \
      --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp \
      -v "$ROOT:/src" -v "$ROOT/workplace/original:/orig:ro" -w /src psychicwar-go-ebiten \
      workplace/bin/probe -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war \
      -load-state "$OUT/saves/$w.state" -steps 0 -dump-mem "16966-1699A:$OUT/saves/$w.mem" > "$OUT/saves/$w.log" 2>&1 || true
  done
  mem=$(tools/py.sh -c '
import pathlib, sys
a = pathlib.Path("/src/" + sys.argv[1])
b = pathlib.Path("/src/" + sys.argv[2])
if not a.exists() or not b.exists():
    print("沒有傾印檔")
else:
    print("相同" if a.read_bytes() == b.read_bytes() else "不同")' "$OUT/saves/before.mem" "$OUT/saves/after.mem")
  say "讀檔前後的觀測變數：$mem"
  [[ "$mem" == "相同" ]] || fail=1
else
  say "讀檔前後的觀測變數：跳過（缺 workplace/bin/probe）"
fi
exit $fail
