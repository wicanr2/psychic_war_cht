#!/usr/bin/env bash
# DOSBox-X 參照畫面（issue #3）：用獨立的第二個實作走到 M0 的四個檢查點，存畫面。
#
#   tools/dosboxx-ref.sh              # 產出 workplace/dosboxx/<檢查點>[-變體].{x11.png,rgb}
#
# 走的畫面與 tools/states.sh 相同（標題 → 防拷 → 輸入名字 → 第一個可操作畫面），
# 但 DOSBox-X 以牆上時間送鍵，**亂數不會對齊**：防拷問的盟友多半與 dosgolem 不同。
# 比對時把那一塊文字區當成已知差異（tools/frame_compare.py）。
#
# 每個檢查點存 X 視窗截圖 <名>.x11.png 與同內容的 raw RGB <名>.rgb。
# scaler=none、aspect=false 下視窗是 640×400，即 320×200 的整數 2 倍；
# tools/frame_compare.py 會先驗每個 2×2 區塊同色，才縮回 320×200 比對。
# DOSBox-X 2026.06.02 在這個設定下 Ctrl+F5 不產生截圖檔，所以不用它。
#
# 設定依 ~/.claude/knowledge-base/retro/dosbox-game-configs.md：cycles 固定，不用 auto。
# 音效卡設成沒有（sbtype=none、oplmode=none），與 dosgolem 預設的「沒有 AdLib」同一條路徑。
#
# ⚠ Xvfb 沒有視窗管理員，焦點跟著滑鼠：先把滑鼠移進 DOSBox-X 視窗再送鍵，
#   否則按鍵會安靜地送不到。
# ⚠ 本 repo 不含原版；讀 workplace/original/psychic-war/（唯讀掛載，容器內複製一份再跑）。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${PSYCHICWAR_DOSBOXX_IMAGE:-civ1-dosboxx-input:20260830}"
GAME="${PSYCHICWAR_ORIG_DIR:-$ROOT/workplace/original}/psychic-war"
OUT="$ROOT/workplace/dosboxx"

[[ -f "$GAME/PW.EXE" ]] || { echo "tools/dosboxx-ref.sh：找不到 $GAME/PW.EXE" >&2; exit 2; }
docker image inspect "$IMAGE" >/dev/null 2>&1 || { echo "tools/dosboxx-ref.sh：找不到 image $IMAGE" >&2; exit 2; }
mkdir -p "$OUT"
rm -f "$OUT"/*.png "$OUT"/*.rgb 2>/dev/null || true

cat > "$OUT/.run.sh" <<'EOF'
set -euo pipefail
export HOME=/tmp DISPLAY=:99
Xvfb :99 -screen 0 1024x768x24 >/tmp/xvfb.log 2>&1 &
sleep 2
mkdir -p /tmp/game
cp -r /orig/. /tmp/game/
chmod -R u+w /tmp/game
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
DBX=$!
sleep 6
WIN=$(xdotool search --name 'DOSBox-X' | tail -1)
eval "$(xdotool getwindowgeometry --shell "$WIN")"
xdotool mousemove $((X + WIDTH / 2)) $((Y + HEIGHT / 2))
xdotool windowfocus --sync "$WIN" || true
key() { xdotool key --clearmodifiers "$1"; sleep "${2:-3}"; }
snap() {
  # X 視窗截圖（原尺寸 640×400 ＝ 2 倍）＋同內容的 raw RGB，給 tools/frame_compare.py
  import -window "$WIN" "/out/$1.x11.png"
  convert "/out/$1.x11.png" -depth 8 "rgb:/out/$1.rgb"
  echo "[$1] 已擷取" >&2
}
# 標題出現的時點依主機速度而定，停太久會進開場故事：從第 6 秒起連抓六張，比對時挑一致的那張
sleep 6
for t in a b c d e f; do snap "01-title-$t"; sleep 1; done
key space 5                   # → 防拷說明框
key Return 6                  # → ENTER ESP POWER
snap 03-protection
key Return 6                  # 空白答案 → YOU ARE CLEARED
key space 20                  # → 主畫面 SELECT 選單
key Return 6                  # NEW GAME → ENTER YOUR NAME
snap 06-name
key k 1; key a 1; key i 1
key Return 15                 # → 第一人稱迷宮
# 迷宮畫面有週期性閃爍：抓三張，比對時挑相位一致的那張
for t in a b c; do snap "07-first-play-$t"; sleep 0.7; done
echo "視窗 ${WIDTH}x${HEIGHT}" >&2
kill $DBX 2>/dev/null || true
EOF

echo "tools/dosboxx-ref.sh：跑 DOSBox-X（約 2 分鐘，1 個容器、2 核）" >&2
timeout "${PSYCHICWAR_DOSBOXX_TIMEOUT:-400}" docker run --rm --network none \
  --memory 2g --cpus 2 --pids-limit 128 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  -v "$GAME:/orig:ro" -v "$OUT:/out" \
  "$IMAGE" bash /out/.run.sh 2>&1 | grep -v -E 'XGetInputFocus|^$' || true
rm -f "$OUT/.run.sh"
ls -l "$OUT"
