#!/usr/bin/env bash
# 滑鼠的實跑驗收（docs/spec/018 §4 第 1、2 項）。
#
#   tools/frontend-mouse-check.sh [狀態檔]        # 預設 states/07-first-play.state
#
# 同一個狀態檔跑三次：
#   A 用滑鼠點「前進」熱區（原版像素 (200, 8)，乘上 scale）
#   B 用鍵盤按 Up
#   C 點熱區外面（畫面左下角）
# 各自 F10 存檔，再用 probe 讀觀測變數（lin:16966 起 52 bytes）比對：
#   A == B（點選與按鍵走到同一個狀態）、C 與「沒點」相同（熱區外不送鍵）。
# 產出 workplace/mouse/：a.mem、b.mem、c.mem、result.txt
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE="${1:-states/07-first-play.state}"
for b in psychicwar probe; do [[ -x "$ROOT/workplace/bin/$b" ]] || { echo "先 build workplace/bin/$b" >&2; exit 2; }; done
OUT=workplace/mouse
rm -rf "$ROOT/$OUT"; mkdir -p "$ROOT/$OUT"

# $1=標籤 $2=動作（在容器內的 shell 片段）
run_one() {
  local tag="$1" act="$2"
  mkdir -p "$ROOT/$OUT/$tag"
  PSYCHICWAR_TIMEOUT=8m PSYCHICWAR_SH="
set -eu
cd /src
workplace/bin/psychicwar -orig /orig/psychic-war -audio null -scratch $OUT/$tag \
  -text text -load-state workplace/$STATE -quit-after 40s > $OUT/$tag/frontend.log 2>&1 &
PID=\$!
for i in \$(seq 1 200); do W=\$(xdotool search --name 'Psychic War' 2>/dev/null | head -1 || true); [ -n \"\$W\" ] && break; sleep 0.1; done
eval \"\$(xdotool getwindowgeometry --shell \"\$W\")\"
sleep 6
tap() { xdotool keydown \"\$1\"; sleep 0.15; xdotool keyup \"\$1\"; }
# 滑鼠點擊也要按住 0.15 秒：Ebiten 每 1/60 秒輪詢一次，太快的按下放開會整個漏掉
click() { xdotool mousemove --sync \$((X + \$1)) \$((Y + \$2)); xdotool mousedown 1; sleep 0.15; xdotool mouseup 1; }
$act
sleep 4
tap F10; sleep 3
kill \$PID; wait \$PID 2>/dev/null || true
" "$ROOT/tools/go-ebiten.sh"
  timeout 10m docker run --rm --network none --memory 2g --cpus 1 --pids-limit 64 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp \
    -v "$ROOT:/src" -v "$ROOT/workplace/original:/orig:ro" -w /src psychicwar-go-ebiten \
    workplace/bin/probe -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war \
    -load-state "$OUT/$tag/quick.state" -steps 0 -dump-mem "16966-1699A:$OUT/$tag.mem" \
    > "$ROOT/$OUT/$tag-probe.log" 2>&1 || true
}

# scale 預設 3：熱區 (200, 8) → 視窗座標 (600, 24)
run_one a 'click 600 24'
run_one b 'tap Up'
run_one c 'click 60 540'   # 畫面左下角，熱區外
run_one d ''               # 什麼都不做（c 的對照）

cd "$ROOT"
set +e
tools/py.sh -c '
import pathlib, sys
root = pathlib.Path("/src")
def rd(p):
    q = root / p
    return q.read_bytes() if q.exists() else None
a, b, c, d = rd(sys.argv[1]), rd(sys.argv[2]), rd(sys.argv[3]), rd(sys.argv[4])
bad = 0
for name, v in [("點前進", a), ("按 Up", b), ("點熱區外", c), ("什麼都不做", d)]:
    if v is None:
        print("%s：沒有存檔（前端沒跑起來？）" % name)
        bad += 1
print("點前進 vs 按 Up：%s" % ("相同" if a and b and a == b else "不同"))
if not (a and b and a == b):
    bad += 1
print("點熱區外 vs 什麼都不做：%s" % ("相同" if c and d and c == d else "不同"))
if not (c and d and c == d):
    bad += 1
print("不符 %d 項" % bad)
sys.exit(1 if bad else 0)' "$OUT/a.mem" "$OUT/b.mem" "$OUT/c.mem" "$OUT/d.mem" | tee "$ROOT/$OUT/result.txt"
exit "${PIPESTATUS[0]}"
