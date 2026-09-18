#!/usr/bin/env bash
# 輸入錄放的驗收（docs/spec/019 §5）。
#
#   tools/record-replay-check.sh [狀態檔]        # 預設 states/07-first-play.state
#
# 1. 錄：Xvfb 裡從狀態檔啟動前端、送一串按鍵（含移動與按住），-record 寫出錄製檔
# 2. 存：錄完 F10 存檔，probe 讀出觀測變數（lin:16966 起 52 bytes）＝ 真值
# 3. 放：tools/replay_record.py 把錄製檔轉成 probe 的 -hold，從同一個狀態檔重播，再讀一次
# 4. 比：真值與重播結果要**逐位元組相同**
# 5. 反向對照：把第一個按鍵挪到重播範圍之外（等於那一鍵不生效），重播結果必須**不同**
#    （不做這一項的話，「相同」有可能只是因為兩邊都沒動）
# 產出 workplace/recplay/：rec.json、live.mem、replay.mem、shifted.mem、result.txt
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE="${1:-states/07-first-play.state}"
for b in psychicwar probe; do [[ -x "$ROOT/workplace/bin/$b" ]] || { echo "先 build workplace/bin/$b" >&2; exit 2; }; done
OUT=workplace/recplay
rm -rf "$ROOT/$OUT"; mkdir -p "$ROOT/$OUT/saves"

PSYCHICWAR_TIMEOUT=10m PSYCHICWAR_SH="
set -eu
cd /src
workplace/bin/psychicwar -orig /orig/psychic-war -audio null -scratch $OUT/saves \
  -text text -load-state workplace/$STATE -record $OUT/rec.json -quit-after 45s > $OUT/frontend.log 2>&1 &
PID=\$!
for i in \$(seq 1 200); do W=\$(xdotool search --name 'Psychic War' 2>/dev/null | head -1 || true); [ -n \"\$W\" ] && break; sleep 0.1; done
sleep 6
tap() { xdotool keydown \"\$1\"; sleep 0.15; xdotool keyup \"\$1\"; }
tap Up; sleep 3
tap Right; sleep 3
tap Up; sleep 3
xdotool keydown space; sleep 1.2; xdotool keyup space   # 按住：長度資訊要能錄到
sleep 3
xdotool key F10; sleep 1; tap F10; sleep 3              # F10 是熱鍵，錄製檔裡不該出現
cp $OUT/saves/quick.state $OUT/live.state
# ⚠ 不要 kill：錄製檔是在 RunGame 正常返回之後才寫的，送 SIGTERM 會讓它來不及寫
# （第一版就是 kill 掉，結果錄製檔根本不存在）。等 -quit-after 自己到期。
wait \$PID 2>/dev/null || true
" "$ROOT/tools/go-ebiten.sh"

cd "$ROOT"
[[ -f "$OUT/rec.json" ]] || { echo "沒有錄到東西（$OUT/rec.json 不存在）" >&2; exit 1; }

# 真值：錄製當下的狀態
probe_mem() { # $1=狀態檔 $2=輸出
  timeout 10m docker run --rm --network none --memory 2g --cpus 1 --pids-limit 64 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp \
    -v "$ROOT:/src" -v "$ROOT/workplace/original:/orig:ro" -w /src psychicwar-go-ebiten \
    workplace/bin/probe -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war \
    -load-state "$1" -steps 0 -dump-mem "16966-1699A:$2" > /dev/null 2>&1 || true
}
probe_mem "$OUT/live.state" "$OUT/live.mem"

# 錄製結束時的絕對指令數：重播要跑到同一個點才能比。
# ⚠ 不能「跑到最後一個事件之後再多跑一段」：多跑的步數會讓遊戲繼續處理輸入，
# 位置就多走一格（第一版多跑 300 萬步，x 差 1、朝向也不同，看起來像重播壞了）。
LIVE_STEP=$(timeout 5m docker run --rm --network none --memory 2g --cpus 1 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp \
  -v "$ROOT:/src" -v "$ROOT/workplace/original:/orig:ro" -w /src psychicwar-go-ebiten \
  workplace/bin/probe -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war \
  -load-state "$OUT/live.state" -steps 0 2>/dev/null | sed -n 's/.*第 \([0-9]*\) 道指令.*/\1/p' | head -1)
[[ -n "$LIVE_STEP" ]] || { echo "讀不到錄製結束時的指令數" >&2; exit 1; }

# 重播：同一個狀態檔 ＋ 錄下來的 -hold，跑到錄製結束存檔的那一刻
replay() { # $1=額外參數 $2=輸出
  local holds last
  holds=$(tools/py.sh tools/replay_record.py "$OUT/rec.json" $1 | head -1)
  # ⚠ probe 的 -steps 是**絕對**指令數，不是「再跑幾步」（tools/states.sh 用 $((at + 1))）。
  # 給相對值的話，它小於狀態檔本身的步數，按鍵一個都排不到——而畫面上看起來就只是
  # 「重播結果跟沒按鍵一樣」。正對照（戰鬥按住空白鍵）用絕對值跑，敵人 HP 38 → 20。
  last=$((LIVE_STEP + 1))
  timeout 15m docker run --rm --network none --memory 2g --cpus 2 --pids-limit 128 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp \
    -v "$ROOT:/src" -v "$ROOT/workplace/original:/orig:ro" -w /src psychicwar-go-ebiten \
    sh -c "workplace/bin/probe -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war \
      -load-state workplace/$STATE -steps $last $holds -hold-typematic=false -dump-mem '16966-1699A:$2'" > "$ROOT/$OUT/replay.log" 2>&1 || true
}
replay "" "$OUT/replay.mem"
# 反向對照：把第一個按鍵挪到重播範圍之外（等於那一鍵不生效）。
# ⚠ 不要只挪 50 萬步（約 0.67 秒）：遊戲那時候正在等按鍵，晚 0.67 秒按下的結果一樣，
# 反向對照會「相同」而看起來像比對失效。要選一個**必然**改變結果的擾動。
replay "--shift 0:20000000" "$OUT/shifted.mem"

set +e
tools/py.sh -c '
import json, pathlib, sys
root = pathlib.Path("/src")
def rd(p):
    q = root / p
    return q.read_bytes() if q.exists() else None
live, rep, shifted = rd(sys.argv[1]), rd(sys.argv[2]), rd(sys.argv[3])
rec = json.loads((root / sys.argv[4]).read_text(encoding="utf-8"))
bad = 0
print("錄到 %d 筆事件（%d 個按下）" % (len(rec["events"]), sum(1 for e in rec["events"] if e["down"])))
keys = {e["key"] for e in rec["events"]}
print("錄到的鍵：%s" % "、".join(sorted(keys)))
if any(k.startswith("F") and k[1:].isdigit() for k in keys):
    print("⚠ 熱鍵被錄進去了（不該出現）")
    bad += 1
if live is None or rep is None:
    print("缺存檔，沒得比")
    sys.exit(1)
print("重播 vs 錄製當下：%s" % ("相同" if live == rep else "不同"))
if live != rep:
    bad += 1
print("把第一個按鍵挪出重播範圍之後：%s（要不同，證明這個比對看得到差異）"
      % ("相同" if shifted == rep else "不同"))
if shifted == rep:
    bad += 1
print("不符 %d 項" % bad)
sys.exit(1 if bad else 0)' "$OUT/live.mem" "$OUT/replay.mem" "$OUT/shifted.mem" "$OUT/rec.json" | tee "$ROOT/$OUT/result.txt"
exit "${PIPESTATUS[0]}"
