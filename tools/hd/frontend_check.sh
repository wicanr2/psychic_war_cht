#!/bin/sh
# 已有 Xvfb 的 Go Docker 容器內執行；不可在主機執行工作負載。
# 入口與第一批驗證限制見 docs/re/038 §33。
set -eu
test -n "${DISPLAY:-}"
test "$#" -ge 1
test "$#" -le 3
OUT="$1"
BIN="${2:-workplace/hd/psychicwar-theme-v1}"
THEME="${3:-workplace/hd/theme-v1-20261001}"
test -f "$BIN"
test -d "$THEME"
test ! -e "$OUT"
test "$(stat -c %u "$(dirname "$OUT")")" = "$(id -u)"
mkdir "$OUT"
mkdir "$OUT/saves"
PID=""
trap 'if [ -n "$PID" ]; then kill "$PID" 2>/dev/null || true; wait "$PID" 2>/dev/null || true; fi' EXIT INT TERM
PSYCHICWAR_KEYLOG=1 "$BIN" -orig /orig/psychic-war \
  -audio null -scratch "$OUT/saves" -load-state workplace/states/07-first-play.state \
  -theme "$THEME" -text text -stats "$OUT/stats.jsonl" \
  -quit-after 90s > "$OUT/frontend.log" 2>&1 &
PID=$!
W=""
for i in $(seq 1 100); do
  W=$(xdotool search --name 'Psychic War' 2>/dev/null | head -1 || true)
  test -z "$W" || break
  sleep 0.1
done
test -n "$W"
xdotool windowfocus "$W"
eval "$(xdotool getwindowgeometry --shell "$W")"
xdotool mousemove "$((X+WIDTH/2))" "$((Y+HEIGHT/2))"
shot() {
  # 擷取用 F10 不得覆蓋玩家路徑中原本供 F11 讀回的存檔。
  for file in quick.state quick.state.xlate.json quick.json quick.map.json; do
    test ! -f "$OUT/saves/$file" || cp "$OUT/saves/$file" "$OUT/.shot-$file"
  done
  xdotool keydown F10; sleep 0.18; xdotool keyup F10
  sleep 3
  cp "$OUT/saves/quick.state" "$OUT/$1.state"
  import -window "$W" -define png:color-type=2 -depth 8 "PNG24:$OUT/$1.png"
  for file in quick.state quick.state.xlate.json quick.json quick.map.json; do
    if test -f "$OUT/.shot-$file"; then
      cp "$OUT/.shot-$file" "$OUT/saves/$file"
      rm "$OUT/.shot-$file"
    else
      rm -f "$OUT/saves/$file"
    fi
  done
}
key() { xdotool keydown "$1"; sleep 0.18; xdotool keyup "$1"; sleep 3; }
hdkey() { xdotool keydown Shift_L; xdotool keydown F5; sleep 0.18; xdotool keyup F5; xdotool keyup Shift_L; sleep 3; }
sleep 5
shot a-hd-chinese
key F5; shot b-hd-english
hdkey; shot c-original-english
key F5; shot d-original-chinese
hdkey; shot e-hd-chinese
key F10
cp "$OUT/saves/quick.state" "$OUT/before.state"
hdkey; shot f-original-before-move
key Up; shot g-moved
key F11; shot h-loaded-original
key F10
cp "$OUT/saves/quick.state" "$OUT/after.state"
hdkey; shot i-loaded-hd
kill "$PID"
wait "$PID" 2>/dev/null || true
PID=""
printf '%s\n' '前端切換、移動及 F10／F11 已擷取；尚須獨立像素與存檔位置核對。' > "$OUT/status.txt"
