#!/usr/bin/env bash
# 真實音訊裝置上的欠載量測（docs/spec/020）。
#
#   tools/audio-device-check.sh [adlib|speaker] [秒數]      # 預設 adlib、65
#
# ⚠ **這會從喇叭發出聲音**（量測期間播遊戲音樂）。沒有真的送到裝置就量不到裝置緩衝，
#   而裝置緩衝正是玩家會聽到斷音的地方（`-audio null` 那條路徑不開裝置，量不到）。
#
# 接法：掛主機的 PipeWire／PulseAudio socket，容器內 ALSA 走 pulse plugin。
# **不獨佔音效卡**——直接掛 /dev/snd 開 hw:0 會跟主機的音訊伺服器搶裝置。
# ALSA 的 null PCM 也不行：它不按取樣率消耗資料而是立刻吸收，量到的是它自己的行為
# （`docs/re/026` §6：65 秒 290,203 次欠載）。
#
# 產出 workplace/audio-device/<模式>/：stats.jsonl、frontend.log、load-before/after.txt、result.txt
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="${1:-adlib}"
SECS="${2:-65}"
case "$MODE" in
  adlib)   ADLIB="-adlib" ;;
  speaker) ADLIB="" ;;
  *) echo "模式要是 adlib 或 speaker" >&2; exit 2 ;;
esac
[[ -x "$ROOT/workplace/bin/psychicwar" ]] || { echo "先 build workplace/bin/psychicwar" >&2; exit 2; }
SOCK="/run/user/$(id -u)/pulse/native"
[[ -S "$SOCK" ]] || { echo "找不到 PulseAudio socket：$SOCK（主機有在跑 PipeWire 或 PulseAudio 嗎）" >&2; exit 2; }

# image 少了 pulse plugin 的話，前端會噴 ALSA 的參數錯誤，看不出真正的原因。先擋下來。
if ! docker run --rm --network none --memory 256m --cpus 1 --pids-limit 32 \
     --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp \
     psychicwar-go-ebiten test -e /usr/lib/x86_64-linux-gnu/alsa-lib/libasound_module_pcm_pulse.so; then
  echo "image 裡沒有 ALSA 的 pulse plugin。重建：" >&2
  echo "  docker build -t psychicwar-go-ebiten -f tools/docker/go-ebiten.Dockerfile tools/docker" >&2
  exit 2
fi

OUT="workplace/audio-device/$MODE"
rm -rf "$ROOT/$OUT"; mkdir -p "$ROOT/$OUT/saves"
uptime | tee "$ROOT/$OUT/load-before.txt"

PSYCHICWAR_TIMEOUT="$((SECS / 60 + 6))m" \
PSYCHICWAR_DOCKER_EXTRA="-v $SOCK:/tmp/pulse-native -e PULSE_SERVER=unix:/tmp/pulse-native" \
PSYCHICWAR_SH="
set -eu
cd /src
printf 'pcm.!default { type pulse }\nctl.!default { type pulse }\n' > /tmp/.asoundrc
workplace/bin/psychicwar -orig /orig/psychic-war $ADLIB -audio ebiten \
  -scratch $OUT/saves -stats $OUT/stats.jsonl -quit-after ${SECS}s > $OUT/frontend.log 2>&1 || echo \"前端結束碼 \$?\"
" "$ROOT/tools/go-ebiten.sh" >/dev/null 2>&1 || true
uptime | tee "$ROOT/$OUT/load-after.txt"

cd "$ROOT"
[[ -s "$OUT/stats.jsonl" ]] || { echo "沒有量到東西，看 $OUT/frontend.log" >&2; tail -5 "$OUT/frontend.log" >&2; exit 1; }
set +e
tools/py.sh -c '
import json, pathlib, sys
rows = [json.loads(l) for l in (pathlib.Path("/src") / sys.argv[1]).read_text(encoding="utf-8").splitlines() if l.strip()]
if len(rows) < 10:
    print("樣本只有 %d 秒，太短" % len(rows)); sys.exit(1)
prev, seg = 0, []
for r in rows:
    u = r.get("underruns", 0); seg.append(u - prev); prev = u
b = rows[-1]
mach = b["machine_ms"] / b["wall_ms"]
after = sum(seg[1:])
print("樣本 %d 秒；欠載總數 %d（開場第 1 秒 %d，之後 %d）" % (len(rows), prev, seg[0], after))
print("放棄追趕 %d ms、溢位 %d、CPU %.2f 核、機器／牆上 %.4f" % (b["dropped_ms"], b["overflows"], b["cpu_ms"]/b["wall_ms"], mach))
bad = 0
if after:
    worst = sorted(((v, i) for i, v in enumerate(seg) if i and v), reverse=True)[:5]
    print("第 2 秒之後有欠載的秒（次數, 第幾秒）：", worst)
    bad += 1
# 反向對照：機器時間沒跟上的話，「欠載 0」可能只是因為它跑得慢、產得少（docs/spec/020 §4 第 4 項）
if abs(mach - 1) > 0.01:
    print("⚠ 機器時間／牆上時間 %.4f 偏離 1 超過 1%%，這次的「欠載 0」不算數" % mach)
    bad += 1
print("不符 %d 項" % bad)
sys.exit(1 if bad else 0)' "$OUT/stats.jsonl" | tee "$ROOT/$OUT/result.txt"
exit "${PIPESTATUS[0]}"
