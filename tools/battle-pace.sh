#!/usr/bin/env bash
# dosgolem 量第一場戰鬥的長度（issue #9；與 tools/dosboxx-battle-speed.sh 的 DOSBox-X 參照對照）。
#
#   tools/battle-pace.sh <cycles|xt|at8|at12> [按住起點]
#
# 從 workplace/states/08-encounter.state（第 86,000,000 步）接著跑，預設第 86,500,000 步起按住空白鍵，
# 監看敵人 HP（lin 0x509C）與玩家 HP（lin 0x16990）的寫入，找出敵人 HP 歸零的指令數，
# 再各跑一次到「按下」與「歸零」兩個時點，讀 probe 摘要的 DOSBox 相容 cycles，換算機器時間。
# 產出 workplace/pace/<cycles>/<按住起點>/：watch.log（敵人）、watch-player.log、press.log、hp0.log、summary.txt
#
# ⚠ 先跑過 tools/states.sh（要有 08-encounter.state）。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOLEM="${PSYCHICWAR_DOSGOLEM:-$ROOT/worktrees/dosgolem}"
WP="$ROOT/workplace"
SPEED="${1:?要給 cycles（數字或 xt／at8／at12）}"
PRESS="${2:-86500000}"
TYPEMATIC="${PSYCHICWAR_HOLD_TYPEMATIC:-true}"   # false：按住時不送重複按下碼（docs/re/010 §4.2 分岔的驗證用）
OUT="$WP/pace/$SPEED/$PRESS"
[[ "$TYPEMATIC" == "true" ]] || OUT="$WP/pace/$SPEED-notypematic/$PRESS"
[[ -s "$WP/states/08-encounter.state" ]] || { echo "找不到 08-encounter.state，先跑 tools/states.sh" >&2; exit 2; }
mkdir -p "$OUT"

probe() {
  DOSGOLEM_ORIG="$WP/original" DOSGOLEM_EXTRA_MOUNT="$WP:/wp" \
  DOSGOLEM_CPUS="${PSYCHICWAR_CPUS:-2}" DOSGOLEM_TIMEOUT="${PSYCHICWAR_TIMEOUT:-15m}" \
    "$GOLEM/tools/go.sh" run ./cmd/probe -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war \
    -load-state /wp/states/08-encounter.state -cycles "$SPEED" -hold-typematic="$TYPEMATIC" "$@"
}

# 按住 3,000 萬步：比任何速度下的戰鬥都長（戰鬥約 1,200 萬步）
HOLD="space@$PRESS+30000000"
# -watch 一次只收一段位址：敵人、玩家各跑一次
probe -steps "$((PRESS + 30000000))" -hold "$HOLD" -watch 509C-509D > "$OUT/watch.log" 2>&1
HP0=$(awk -v lo="$PRESS" '/\[watch\] #[0-9]+ 0509C: [0-9A-F]+ → 00 /{n=$2; sub("#","",n); if (n+0>=lo) {print n; exit}}' "$OUT/watch.log")
[[ -n "$HP0" ]] || { echo "敵人 HP 沒有歸零（見 $OUT/watch.log）" >&2; exit 1; }
probe -steps "$HP0" -hold "$HOLD" -watch 16990-16991 > "$OUT/watch-player.log" 2>&1
probe -steps "$PRESS" -hold "$HOLD" > "$OUT/press.log" 2>&1
probe -steps "$HP0" -hold "$HOLD" > "$OUT/hp0.log" 2>&1

cyc() { awk '/本次 DOSBox 相容 cycles/{sub("本次 DOSBox 相容 cycles ",""); split($0,a,"，"); print a[1]}' "$1"; }
C_PRESS=$(cyc "$OUT/press.log"); C_HP0=$(cyc "$OUT/hp0.log")
PER_MS=$(awk '/機器速度：DOSBox 相容/{sub(".*相容 ",""); print $1+0; exit}' "$OUT/hp0.log")
PLAYER=$(awk -v lo="$PRESS" -v hi="$HP0" '/\[watch\] #[0-9]+ 16990:/{n=$2; sub("#","",n); if (n+0>=lo && n+0<=hi) c++} END{print c+0}' "$OUT/watch-player.log")
LAST=$(awk -v hi="$HP0" '/\[watch\] #[0-9]+ 16990:/{n=$2; sub("#","",n); if (n+0<=hi) v=$6} END{print v}' "$OUT/watch-player.log")
awk -v s="$SPEED" -v p="$PRESS" -v h="$HP0" -v cp="$C_PRESS" -v ch="$C_HP0" -v pm="$PER_MS" -v pl="$PLAYER" -v last="$LAST" 'BEGIN {
  printf "speed=%s cycles_per_ms=%d press=%d hp0=%d battle_steps=%d battle_cycles=%d cycles_per_step=%.3f sec=%.2f player_hp_writes=%d player_hp_last=%s\n",
    s, pm, p, h, h-p, ch-cp, (ch-cp)/(h-p), (ch-cp)/(pm*1000), pl, (last==""?"—":last) }' | tee "$OUT/summary.txt"
