#!/usr/bin/env bash
# DOSBox-X 以固定 cycles 量第一場戰鬥（按住攻擊）並錄影（issue #9 的速度參照）。
#
#   tools/dosboxx-battle-speed.sh <cycles> [按住秒數]
#       240 ＝ DOSBox-X「8088 XT 4.77MHz」預設；750 ＝「286 8MHz」預設（說明書硬體配備：IBM PC XT／AT）
#   產出 workplace/dosboxx-speed/c<cycles>/：frames/NNNN.png（4 fps，第 0 格＝按下空白鍵前 1 秒）、title.txt、log
#
# 走法：以 8000 cycles 照 tools/dosboxx-ref.sh 的時序走到遭遇，再用 host 鍵（Linux 預設 F12）＋減號把 cycles 一次降到目標值
# （cycledown ＝ 8000 − 目標），從視窗標題讀回 cycles 確認，然後按住空白鍵開打。
#
# ⚠ 不要整段都用低 cycles 跑、再用「dosgolem 指令數 ÷ cycles」估每一段要等多久：實測 750 cycles 走了 216 秒
#   還停在主選單，按鍵全部落空。開場各段的長短不是指令數能換算的（計時器等待、主機負載）。
# ⚠ host 鍵要「按住 F12 → 點減號 → 放開」分開送；hostkey=ctrlalt 配 xdotool 的組合鍵實測不生效。
# ⚠ Xvfb 沒有視窗管理員，焦點跟著滑鼠：先把滑鼠移進 DOSBox-X 視窗再送鍵。
# ⚠ 本 repo 不含原版；讀 workplace/original/psychic-war/（唯讀掛載，容器內複製一份再跑）。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${PSYCHICWAR_DOSBOXX_IMAGE:-civ1-dosboxx-input:20260830}"
GAME="${PSYCHICWAR_ORIG_DIR:-$ROOT/workplace/original}/psychic-war"
CYCLES="${1:?要給 cycles（240 或 750）}"
HOLD="${2:-}"
OUT="$ROOT/workplace/dosboxx-speed/c$CYCLES"

[[ -f "$GAME/PW.EXE" ]] || { echo "找不到 $GAME/PW.EXE" >&2; exit 2; }
(( CYCLES > 0 && CYCLES < 8000 )) || { echo "cycles 要在 1–7999" >&2; exit 2; }
# 戰鬥約 1,220 萬 cycles（dosgolem 規格 198 量測，戰鬥中沒有字串指令）：按住到預估時間 × 1.4 ＋ 5 秒
[[ -n "$HOLD" ]] || HOLD=$(awk -v c="$CYCLES" 'BEGIN { printf "%d", 12200 / c * 1.4 + 5 }')
REC=$((HOLD + 10))
DOWN=$((8000 - CYCLES))

rm -rf "$OUT"; mkdir -p "$OUT/frames"
cat > "$OUT/.run.sh" <<EOF
set -euo pipefail
export HOME=/tmp DISPLAY=:99
Xvfb :99 -screen 0 1024x768x24 >/tmp/xvfb.log 2>&1 &
sleep 2
mkdir -p /tmp/game && cp -r /orig/. /tmp/game/ && chmod -R u+w /tmp/game
cat > /tmp/dosbox.conf <<CONF
[sdl]
output=surface
autolock=false
windowresolution=original
[dosbox]
machine=svga_s3
memsize=4
[render]
aspect=false
scaler=none
[cpu]
core=normal
cputype=286
cycles=fixed 8000
cycledown=$DOWN
[sblaster]
sbtype=none
oplmode=none
[gus]
gus=false
[speaker]
pcspeaker=true
[autoexec]
mount c /tmp/game
c:
PW.EXE
CONF
dosbox-x -conf /tmp/dosbox.conf -nomenu >/tmp/dbx.log 2>&1 &
DBX=\$!
sleep 6; for i in \$(seq 1 300); do WIN=\$(xdotool search --name 'DOSBox-X' 2>/dev/null | tail -1 || true); [ -n "\$WIN" ] && break; sleep 0.1; done
eval "\$(xdotool getwindowgeometry --shell "\$WIN")"
xdotool mousemove \$((X + WIDTH / 2)) \$((Y + HEIGHT / 2))
t0=\$(date +%s)
note() { echo "[\$((\$(date +%s) - t0))s] \$*" >&2; }
xdotool windowfocus --sync "\$WIN" || true
mkdir -p /out/steps
shot() { import -window "\$WIN" png:- | md5sum | cut -c1-16; }
n=0
# 送一鍵，等畫面跟著變（最多 40 秒），再等 settle 秒；每一步截圖存 steps/，時序錯了看得出卡在哪一步
key() {
  h0=\$(shot); xdotool key --clearmodifiers "\$1"
  for i in \$(seq 1 160); do [ "\$(shot)" != "\$h0" ] && break; sleep 0.25; done
  sleep "\$2"; n=\$((n + 1))
  import -window "\$WIN" "/out/steps/\$(printf %02d \$n)-\$1.png"
  note "key \$1（畫面變化後再等 \$2 秒）"
}
# 標題出現約在視窗出現後 12–18 秒（tools/dosboxx-ref.sh 實跑的時序）；太早按空白鍵會被吃掉
sleep 18
import -window "\$WIN" /out/title.png
key space 3; key Return 3; key Return 3; key space 5; key Return 3
key k 1; key a 1; key i 1
key Return 6
for s in \$(seq 1 11); do key Up 1; done
sleep 2
import -window "\$WIN" /out/encounter.png
xdotool keydown F12; sleep 0.3; xdotool key minus; sleep 0.3; xdotool keyup F12; sleep 1
xdotool getwindowname "\$WIN" | tee /out/title.txt >&2
grep -q "$CYCLES" /out/title.txt || { note "視窗標題沒有 $CYCLES，cycles 沒降下來"; kill \$DBX; exit 3; }
note "cycles 已降到 $CYCLES；錄影 $REC 秒、第 1 秒起按住空白鍵 $HOLD 秒"
ffmpeg -loglevel error -f x11grab -framerate 4 -video_size \${WIDTH}x\${HEIGHT} -i ":99.0+\${X},\${Y}" \
  -t $REC -start_number 0 "/out/frames/%04d.png" &
FF=\$!
sleep 1
xdotool keydown space; note "按下空白鍵"; sleep $HOLD; xdotool keyup space
note "放開空白鍵"
wait \$FF || true
note "錄影結束"
kill \$DBX 2>/dev/null || true
EOF

TOTAL=150
echo "tools/dosboxx-battle-speed.sh：cycles=$CYCLES，走到遭遇約 $TOTAL 秒，錄影 $REC 秒" >&2
timeout "$((TOTAL + REC + 120))" docker run --rm --network none \
  --memory 2g --cpus 2 --pids-limit 128 --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -v "$GAME:/orig:ro" -v "$OUT:/out" \
  "$IMAGE" bash /out/.run.sh 2>&1 | grep -v -E 'XGetInputFocus|^$' | tee "$OUT/log" || true
rm -f "$OUT/.run.sh"
echo "錄到 $(ls "$OUT/frames" | wc -l) 格" >&2
