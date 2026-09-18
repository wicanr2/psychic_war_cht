#!/usr/bin/env bash
# 作弊熱鍵的實跑驗收（docs/spec/014 §4 第 2、3 項）。
#
#   tools/frontend-cheat-check.sh [狀態檔]      # 預設 states/08-encounter.state（戰鬥中）
#
# 兩輪：帶 -cheat 與不帶。每輪都是「按 F5、F6 → F10 存檔」，再用 probe 從存檔讀出
# HP（lin 16990）、HP 上限（1698E）、能量（16994）、能量上限（16992）、敵人 HP（509C）。
# 帶 -cheat：HP ＝ 上限、能量 ＝ 上限、敵人 HP ＝ 1。不帶：三個值與按鍵前相同。
# 產出 workplace/cheat/：各輪的 quick.state 與 .mem、result.txt
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE="${1:-states/08-encounter.state}"
for b in psychicwar probe; do [[ -x "$ROOT/workplace/bin/$b" ]] || { echo "先 build workplace/bin/$b" >&2; exit 2; }; done
OUT=workplace/cheat
rm -rf "$ROOT/$OUT"; mkdir -p "$ROOT/$OUT"

run() { # <標籤> <額外旗標>
  PSYCHICWAR_TIMEOUT=8m PSYCHICWAR_SH="
set -eu
cd /src
workplace/bin/psychicwar -orig /orig/psychic-war -audio null -scratch $OUT/$1 -load-state workplace/$STATE -text text $2 -quit-after 40s > $OUT/$1.log 2>&1 &
PID=\$!
for i in \$(seq 1 100); do W=\$(xdotool search --name 'Psychic War' 2>/dev/null | head -1 || true); [ -n \"\$W\" ] && break; sleep 0.1; done
sleep 4
xdotool key F5; sleep 1; xdotool key F6; sleep 1
xdotool key F10; sleep 3
import -window \"\$W\" -define png:color-type=2 -depth 8 PNG24:$OUT/$1.png
kill \$PID; wait \$PID 2>/dev/null || true
" "$ROOT/tools/go-ebiten.sh"
  timeout 10m docker run --rm --network none --memory 2g --cpus 1 --pids-limit 64 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp \
    -v "$ROOT:/src" -v "$ROOT/workplace/original:/orig:ro" -w /src psychicwar-go-ebiten \
    workplace/bin/probe -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war \
    -load-state "$OUT/$1/quick.state" -steps 0 \
    -dump-mem "1698E-16996:$OUT/$1-vars.bin,509C-509E:$OUT/$1-enemy.bin" > "$OUT/$1-probe.log" 2>&1 || true
}
run cheat -cheat
run plain ""
"$ROOT/tools/py.sh" -c '
import pathlib, struct
out = pathlib.Path("/src/workplace/cheat")
for tag in ("cheat", "plain"):
    v = (out / (tag + "-vars.bin")).read_bytes()
    e = (out / (tag + "-enemy.bin")).read_bytes()
    hpmax, hp, enmax, en = struct.unpack("<HHHH", v[:8])
    enemy = struct.unpack("<H", e[:2])[0]
    print("%-6s HP %d/%d 能量 %d/%d 敵人 HP %d" % (tag, hp, hpmax, en, enmax, enemy))
' | tee "$ROOT/$OUT/result.txt"
