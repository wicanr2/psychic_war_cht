#!/usr/bin/env bash
# DOSBox-X 錄標題音樂（PC 喇叭或 AdLib 路徑），給 issue #6／#5 的音樂比對當參照。
#
#   tools/dosboxx-audio.sh [秒數] [speaker|adlib]
#       speaker（預設）：產出 workplace/dosboxx-audio/title.wav
#       adlib          ：產出 workplace/dosboxx-audio/title-adlib.wav
#
# 不按任何鍵：標題畫面會一直等，音樂照樣播（PC 喇叭版約 80 秒）。
# speaker：音效卡設成沒有（sbtype=none、oplmode=none），遊戲偵測不到 AdLib，走 .IBM 的 PC 喇叭路徑，與 dosgolem 預設相同。
# adlib  ：sbtype=sb1、oplmode=opl2，388h 上有 OPL2，遊戲改讀 .MID（對應 dosgolem 的 -adlib）。
#
# 錄音用 mapper 的 recwave：hostkey=ctrlalt 時是 Ctrl+Alt+W（DOSBox-X hardware.cpp 的 MK_w＋MMODHOST）。
# ⚠ Xvfb 沒有視窗管理員，焦點跟著滑鼠：先把滑鼠移進 DOSBox-X 視窗再送鍵，否則熱鍵安靜地送不到。
# ⚠ 本 repo 不含原版；讀 workplace/original/psychic-war/（唯讀掛載，容器內複製一份再跑）。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${PSYCHICWAR_DOSBOXX_IMAGE:-civ1-dosboxx-input:20260830}"
GAME="${PSYCHICWAR_ORIG_DIR:-$ROOT/workplace/original}/psychic-war"
OUT="$ROOT/workplace/dosboxx-audio"
SECS="${1:-95}"
MODE="${2:-speaker}"
case "$MODE" in
  speaker) SB="sbtype=none"; OPL="oplmode=none"; NAME="title.wav" ;;
  adlib)   SB="sbtype=sb1";  OPL="oplmode=opl2"; NAME="title-adlib.wav" ;;
  *) echo "tools/dosboxx-audio.sh：模式要是 speaker 或 adlib" >&2; exit 2 ;;
esac

[[ -f "$GAME/PW.EXE" ]] || { echo "tools/dosboxx-audio.sh：找不到 $GAME/PW.EXE" >&2; exit 2; }
docker image inspect "$IMAGE" >/dev/null 2>&1 || { echo "tools/dosboxx-audio.sh：找不到 image $IMAGE" >&2; exit 2; }
mkdir -p "$OUT"
rm -rf "$OUT"/capture "$OUT/$NAME"

cat > "$OUT/.run.sh" <<EOF
set -euo pipefail
export HOME=/tmp DISPLAY=:99
Xvfb :99 -screen 0 1024x768x24 >/tmp/xvfb.log 2>&1 &
sleep 2
mkdir -p /tmp/game /out/capture
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
hostkey=ctrlalt
captures=/out/capture
[render]
aspect=false
scaler=none
[cpu]
core=normal
cputype=286
cycles=fixed 8000
[mixer]
rate=22050
[sblaster]
$SB
$OPL
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
for i in \$(seq 1 50); do WIN=\$(xdotool search --name 'DOSBox-X' 2>/dev/null | tail -1 || true); [ -n "\$WIN" ] && break; sleep 0.1; done
eval "\$(xdotool getwindowgeometry --shell "\$WIN")"
xdotool mousemove \$((X + WIDTH / 2)) \$((Y + HEIGHT / 2))
xdotool key --clearmodifiers ctrl+alt+w
echo "開始錄音" >&2
sleep $SECS
xdotool key --clearmodifiers ctrl+alt+w
sleep 2
kill \$DBX 2>/dev/null || true
sleep 1
ls -l /out/capture >&2
EOF

echo "tools/dosboxx-audio.sh：跑 DOSBox-X（$MODE）錄 $SECS 秒（1 個容器、2 核）" >&2
timeout "$((SECS + 120))" docker run --rm --network none \
  --memory 2g --cpus 2 --pids-limit 128 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  -v "$GAME:/orig:ro" -v "$OUT:/out" \
  "$IMAGE" bash /out/.run.sh 2>&1 | grep -v -E 'XGetInputFocus|^$' || true
rm -f "$OUT/.run.sh"
wav=$(ls -t "$OUT"/capture/*.wav 2>/dev/null | head -1 || true)
[[ -n "$wav" && -s "$wav" ]] || { echo "tools/dosboxx-audio.sh：沒有錄到 WAV（熱鍵可能沒送到）" >&2; exit 1; }
cp "$wav" "$OUT/$NAME"
ls -l "$OUT/$NAME"
