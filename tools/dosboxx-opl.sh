#!/usr/bin/env bash
# DOSBox-X 擷取標題音樂的 raw OPL 暫存器序列（.dro），給 OPL2 事件層比對（issue #5，docs/spec/005）。
#
#   tools/dosboxx-opl.sh [秒數]      產出 workplace/dosboxx-opl/capture/*.dro
#
# autoexec 用 `DX-CAPTURE /O /-D PW.EXE`：遊戲一開始就擷取，第一個音起寫檔；遊戲結束時收尾寫標頭。
# 秒數到了照正常流程走進迷宮再送 Ctrl+Q（說明書：回 DOS），讓遊戲正常結束——直接關掉 DOSBox-X 的話標頭不會寫（指令數 0）。
# 擷取因此包含標題之後的音樂；比對取與 dosgolem 標題紀錄重疊的前綴。
# sbtype=sb1、oplmode=opl2：388h 上有 OPL2，遊戲走 .MID 路徑（對應 dosgolem 的 -adlib）。
# ⚠ Xvfb 沒有視窗管理員，焦點跟著滑鼠：先把滑鼠移進 DOSBox-X 視窗再送鍵。
# ⚠ 本 repo 不含原版；讀 workplace/original/psychic-war/（唯讀掛載，容器內複製一份再跑）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${PSYCHICWAR_DOSBOXX_IMAGE:-civ1-dosboxx-input:20260830}"
GAME="${PSYCHICWAR_ORIG_DIR:-$ROOT/workplace/original}/psychic-war"
OUT="$ROOT/workplace/dosboxx-opl"
SECS="${1:-90}"
[[ -f "$GAME/PW.EXE" ]] || { echo "找不到 $GAME/PW.EXE" >&2; exit 2; }
rm -rf "$OUT"; mkdir -p "$OUT/capture"
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
captures=/out/capture
show recorded filename=false
[render]
aspect=false
scaler=none
[cpu]
core=normal
cputype=286
cycles=fixed 8000
[sblaster]
sbtype=sb1
oplmode=opl2
[gus]
gus=false
[speaker]
pcspeaker=true
[autoexec]
mount c /tmp/game
c:
DX-CAPTURE /O /-D PW.EXE
CONF
dosbox-x -conf /tmp/dosbox.conf -nomenu >/tmp/dbx.log 2>&1 &
DBX=\$!
sleep 6; for i in \$(seq 1 300); do WIN=\$(xdotool search --name 'DOSBox-X' 2>/dev/null | tail -1 || true); [ -n "\$WIN" ] && break; sleep 0.1; done
eval "\$(xdotool getwindowgeometry --shell "\$WIN")"
xdotool mousemove \$((X + WIDTH / 2)) \$((Y + HEIGHT / 2))
sleep $SECS
import -window "\$WIN" /out/before-quit.png
shot() { import -window "\$WIN" png:- | md5sum | cut -c1-16; }
key() { h0=\$(shot); xdotool key --clearmodifiers "\$1"; for i in \$(seq 1 160); do [ "\$(shot)" != "\$h0" ] && break; sleep 0.25; done; sleep "\$2"; }
# 標題畫面按 Ctrl+Q 只會被當成一般按鍵；走進迷宮再按才會回 DOS（擷取在程式結束時收尾）
key space 3; key Return 3; key Return 3; key space 5; key Return 3
key k 1; key a 1; key i 1; key Return 6
xdotool key --clearmodifiers ctrl+q
sleep 4
xdotool key --clearmodifiers space   # GAME OVER 畫面：Hit any key to go back to DOS
sleep 6
import -window "\$WIN" /out/after-quit.png || true
kill \$DBX 2>/dev/null || true
sleep 1
grep -i "opl\|capture" /tmp/dbx.log >&2 || true
ls -l /out/capture >&2
EOF
timeout "$((SECS + 120))" docker run --rm --network none \
  --memory 2g --cpus 2 --pids-limit 128 --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -v "$GAME:/orig:ro" -v "$OUT:/out" \
  "$IMAGE" bash /out/.run.sh 2>&1 | grep -v -E 'XGetInputFocus|^$' || true
rm -f "$OUT/.run.sh"
