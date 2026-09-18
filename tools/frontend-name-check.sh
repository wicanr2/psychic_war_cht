#!/usr/bin/env bash
# 輸入名字的實跑驗收（docs/spec/017 §4）。
#
#   tools/frontend-name-check.sh [狀態檔] [期望的名字]
#       預設 states/play3-047.state、play3
#
# 為什麼從狀態檔載入而不是從開機打字：前端是**實時**跑的，轉場比逐步操作（pwstep）快，
# 「等固定秒數再送鍵」會整串錯位——實測第 6 個 Return 之後畫面已經在迷宮，
# 名字被空的確認掉了，字母全送在迷宮裡。名字輸入本身由 pwstep 的逐步試玩涵蓋
# （workplace/playtest/play3 第 013–014 步），這支專驗前端這一層：**非 ASCII 要被擋，而且看得出來**。
#
# 流程：Xvfb 裡從狀態檔啟動 → F10 存檔 → 截圖 → 送一個中文碼位 → 截圖 → 再 F10 存檔。
# 比對：
#   1. 狀態檔裡的名字欄位（lin:1699A 起 8 bytes）等於期望的名字
#   2. 送非 ASCII 之後名字欄位**逐位元組相同**（沒有混進去）
#   3. 送之前與送之後的畫面不同（提示要看得見）
# 產出 workplace/nameentry/：before.mem、after.mem、before.png、toast.png、result.txt
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE="${1:-states/play3-047.state}"
WANT="${2:-play3}"
for b in psychicwar probe; do [[ -x "$ROOT/workplace/bin/$b" ]] || { echo "先 build workplace/bin/$b" >&2; exit 2; }; done
OUT=workplace/nameentry
rm -rf "$ROOT/$OUT"; mkdir -p "$ROOT/$OUT/saves"

PSYCHICWAR_TIMEOUT=10m PSYCHICWAR_SH="
set -eu
cd /src
workplace/bin/psychicwar -orig /orig/psychic-war -audio null -scratch $OUT/saves \
  -text text -load-state workplace/$STATE -quit-after 70s > $OUT/frontend.log 2>&1 &
PID=\$!
for i in \$(seq 1 200); do W=\$(xdotool search --name 'Psychic War' 2>/dev/null | head -1 || true); [ -n \"\$W\" ] && break; sleep 0.1; done
sleep 6
# ⚠ 按鍵要「按下 → 0.15 秒 → 放開」：xdotool key 的按下與放開只差幾毫秒，
# 落在 Ebiten 兩次輪詢（1/60 秒）之間就整個漏掉（tools/frontend-playthrough.sh 已記過）。
tap() { xdotool keydown \"\$1\"; sleep 0.15; xdotool keyup \"\$1\"; }
tap F10; sleep 3; cp $OUT/saves/quick.state $OUT/before.state
sleep 3   # 等 F10 的提示自己消失（兩秒），不然會跟非 ASCII 的提示混在一起
import -window \"\$W\" -define png:color-type=2 -depth 8 PNG24:$OUT/before.png
# 中文輸入法送的是字元不是掃描碼；xdotool 的 U+XXXX 走同一條路
# xdotool type 為了送出 Unicode 字元會暫時重映射鍵盤；key U4E2D 在某些版本上安靜地不送
if xdotool type --delay 120 中 2>$OUT/xdotool.err; then echo 1 > $OUT/sent; else echo 0 > $OUT/sent; fi
sleep 2
import -window \"\$W\" -define png:color-type=2 -depth 8 PNG24:$OUT/toast.png
sleep 3
tap F10; sleep 3; cp $OUT/saves/quick.state $OUT/after.state
kill \$PID; wait \$PID 2>/dev/null || true
" "$ROOT/tools/go-ebiten.sh"

cd "$ROOT"
for w in before after; do
  timeout 10m docker run --rm --network none --memory 2g --cpus 1 --pids-limit 64 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp \
    -v "$ROOT:/src" -v "$ROOT/workplace/original:/orig:ro" -w /src psychicwar-go-ebiten \
    workplace/bin/probe -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war \
    -load-state "$OUT/$w.state" -steps 0 -dump-mem "1699A-169A2:$OUT/$w.mem" > "$OUT/$w-probe.log" 2>&1 || true
done
set +e
tools/py.sh -c '
import pathlib, sys
sys.path.insert(0, "/src/tools")
import pbl
root = pathlib.Path("/src")
want = sys.argv[1]
a = (root / sys.argv[2]).read_bytes()
b = (root / sys.argv[3]).read_bytes()
got = a.decode("ascii", "replace").rstrip()
bad = 0
print("名字欄位 %r（要 %r）" % (got, want))
if got.lower() != want.lower():
    bad += 1
print("送非 ASCII 之後的名字欄位：%s" % ("相同" if a == b else "不同（被寫進去了）"))
if a != b:
    bad += 1
w1, h1, p1 = pbl.screen_indices(root / sys.argv[4], 3)
w2, h2, p2 = pbl.screen_indices(root / sys.argv[5], 3)
diff = sum(1 for i in range(min(len(p1), len(p2))) if p1[i] != p2[i])
if sys.argv[6] == "1":
    print("送非 ASCII 前後的畫面：不同像素 %d（提示要看得見）" % diff)
    if diff == 0:
        bad += 1
else:
    print("⚠ xdotool 在這個 X server 上送不出 Unicode，提示這一項沒驗到"
          "（判斷邏輯由 apps/psychicwar 的 TestNonASCII 涵蓋）")
print("不符 %d 項" % bad)
sys.exit(1 if bad else 0)' "$WANT" "$OUT/before.mem" "$OUT/after.mem" "$OUT/before.png" "$OUT/toast.png" \
  "$(cat "$ROOT/$OUT/sent" 2>/dev/null || echo 0)" | tee "$ROOT/$OUT/result.txt"
exit "${PIPESTATUS[0]}"
