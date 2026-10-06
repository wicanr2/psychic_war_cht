#!/bin/sh
# 僅在既有Xvfb容器內執行；研究038 §52，所有產物留本機。
set -eu
test -n "${DISPLAY:-}"
OUT="$1"
BIN="$2"
THEME="workplace/hd/theme-over-v1-20261002"
test -f "$BIN"
test -d "$THEME"
test ! -e "$OUT"
test "$(stat -c %u "$(dirname "$OUT")")" = "$(id -u)"
mkdir "$OUT"
mkdir "$OUT/saves"
PID=""
trap 'if [ -n "$PID" ]; then kill "$PID" 2>/dev/null || true; wait "$PID" 2>/dev/null || true; fi' EXIT INT TERM
PSYCHICWAR_KEYLOG=1 "$BIN" -orig /orig/psychic-war -audio null -speed 1 \
  -scratch "$OUT/saves" -load-state workplace/hd/over-runtime-v1-20261002-event07.state \
  -theme "$THEME" -text text -font font -stats "$OUT/stats.jsonl" \
  -text-log "$OUT/text.jsonl" -record "$OUT/record.json" -quit-after 240s > "$OUT/frontend.log" 2>&1 &
PID=$!
W=""
for i in $(seq 1 200); do
  W=$(xdotool search --name 'Psychic War' 2>/dev/null | head -1 || true)
  test -z "$W" || break
  sleep 0.1
done
test -n "$W"
xdotool windowfocus "$W"
eval "$(xdotool getwindowgeometry --shell "$W")"
test "$WIDTH" = 960
test "$HEIGHT" = 600
xdotool mousemove "$((X+WIDTH/2))" "$((Y+HEIGHT/2))"
key() { xdotool keydown "$1"; sleep 0.18; xdotool keyup "$1"; sleep 3; }
hdkey() { xdotool keydown Shift_L; xdotool keydown F5; sleep 0.18; xdotool keyup F5; xdotool keyup Shift_L; sleep 3; }
shot() {
  for file in quick.state quick.state.xlate.json quick.json quick.map.json; do
    test ! -f "$OUT/saves/$file" || cp "$OUT/saves/$file" "$OUT/.shot-$file"
  done
  key F10
  cp "$OUT/saves/quick.state" "$OUT/$1.state"
  cp "$OUT/saves/quick.state.xlate.json" "$OUT/$1.state.xlate.json"
  cp "$OUT/saves/quick.json" "$OUT/$1.meta.json"
  import -window "$W" -define png:color-type=2 -depth 8 "PNG24:$OUT/$1.png"
  if test "$1" = g-after-enter; then
    # 保存兩種原版游標相位，驗證器仍要求完整畫面逐像素相等。
    for sample in $(seq 1 30); do
      import -window "$W" -define png:color-type=2 -depth 8 "PNG24:$OUT/$1-phase-$sample.png"
      sleep 0.1
    done
  fi
  printf '%s\n' "$1 已擷取"
  for file in quick.state quick.state.xlate.json quick.json quick.map.json; do
    if test -f "$OUT/.shot-$file"; then
      cp "$OUT/.shot-$file" "$OUT/saves/$file"
      rm "$OUT/.shot-$file"
    else
      rm -f "$OUT/saves/$file"
    fi
  done
}
# 等原版自行畫完陣亡人物及三行中文，純看畫面決定何時送鍵。
for i in $(seq 1 240); do
  kill -0 "$PID"
  import -window "$W" -define png:color-type=2 -depth 8 "PNG24:$OUT/wait.png"
  D=$(compare -metric AE workplace/hd/over-runtime-v1-20261002-scene01-hd-chinese.png "$OUT/wait.png" null: 2>&1 || true)
  test "$D" != 0 || break
  sleep 0.5
done
test "$D" = 0
rm "$OUT/wait.png"
shot a-hd-chinese
key F5; shot b-hd-english
hdkey; shot c-original-english
key F5; shot d-original-chinese
hdkey; shot e-hd-chinese
key F10
cp "$OUT/saves/quick.state" "$OUT/before.state"
cp "$OUT/saves/quick.state.xlate.json" "$OUT/before.state.xlate.json"
hdkey; shot f-original-before-enter
key Return
sleep 5
shot g-after-enter
key F11; shot h-loaded-original
hdkey; shot i-loaded-hd
printf '%s\n' '九張視窗已擷取，等待正式前端正常退出並保存按鍵紀錄。'
wait "$PID"
PID=""
test -s "$OUT/record.json"
printf '%s\n' '視窗擷取完成；尚須原版來源與獨立像素核對。' > "$OUT/status.txt"
