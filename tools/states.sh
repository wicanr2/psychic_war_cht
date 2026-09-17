#!/usr/bin/env bash
# 產生 M0 的狀態檔檢查點（issue #2）。
#
#   tools/states.sh            # 從開機一路跑，產出 workplace/states/*.state 與畫面
#   tools/states.sh --check    # 同上，並比對 tools/states.expected（決定性檢查）
#
# 每一段從上一段的狀態檔接著跑、送一組按鍵、在固定指令數存狀態（-save-state）
# 並存一格畫面。四個 M0 檢查點是：
#
#   01-title        標題畫面，等按鍵
#   03-protection   防拷：ENTER ESP POWER 輸入框
#   06-name         ENTER YOUR NAME 輸入框（主畫面）
#   07-first-play   第一人稱迷宮，第一個可操作畫面
#
# 其餘三段（02-match、04-cleared、05-select）是中間畫面，一併保留。
# 08-encounter 是往前走 11 步後遇到第一個敵人（Shulosu），給決定性測試與 #8 的重播用。
#
# ⚠ 按鍵時機就是亂數的一部分：防拷問哪個盟友會隨按鍵的指令數改變（docs/re/003）。
#   改任何一段的時機，後面所有檢查點的畫面都會變，要一起更新 tools/states.expected。
# ⚠ 狀態檔綁 dosgolem 版本。換版本後先跑 --check。
# ⚠ 本 repo 不含原版；讀 workplace/original/psychic-war/（玩家自備、唯讀掛載）。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOLEM="${PSYCHICWAR_DOSGOLEM:-$ROOT/worktrees/dosgolem}"
WP="$ROOT/workplace"
ORIG="${PSYCHICWAR_ORIG_DIR:-$WP/original}"
OUT="$WP/states"
EXPECTED="$ROOT/tools/states.expected"
CHECK=0
[[ "${1:-}" == "--check" ]] && CHECK=1

die() { echo "tools/states.sh：$*" >&2; exit 2; }
[[ -f "$ORIG/psychic-war/PW.EXE" ]] || die "找不到 $ORIG/psychic-war/PW.EXE（先把 DOS 版解壓到 workplace/original/）"
[[ -x "$GOLEM/tools/go.sh" ]] || die "找不到 $GOLEM/tools/go.sh（先 git clone dosgolem 到 worktrees/）"
mkdir -p "$OUT"

probe() {
  DOSGOLEM_ORIG="$ORIG" DOSGOLEM_EXTRA_MOUNT="$WP:/wp" \
  DOSGOLEM_CPUS="${PSYCHICWAR_CPUS:-2}" DOSGOLEM_TIMEOUT="${PSYCHICWAR_TIMEOUT:-15m}" \
    "$GOLEM/tools/go.sh" run ./cmd/probe -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war "$@"
}

# 名稱 | 從哪一段接 | 存檔步數 | 按鍵 | 第一個鍵的步數 | 鍵距
STAGES=(
  "01-title||12000000|||"
  "02-match|01-title|15000000|space|12500000|"
  "03-protection|02-match|18000000|enter|15500000|"
  "04-cleared|03-protection|22000000|enter|18500000|"
  "05-select|04-cleared|32000000|space|22500000|"
  "06-name|05-select|35000000|enter|32500000|"
  "07-first-play|06-name|42000000|25,1E,17,enter|35500000|500000"
  "08-encounter|07-first-play|86000000|up,up,up,up,up,up,up,up,up,up,up|43000000|4000000"
)

manifest="$OUT/manifest.tsv"
: > "$manifest"
for row in "${STAGES[@]}"; do
  IFS='|' read -r name from at keys key_at every <<<"$row"
  # -dump-at 另存一份經屬性暫存器與 DAC 解色的 PNG（<段>.rgb.png，另附同內容色號 .rgb.bin），給 RGB 比對用
  args=(-steps "$((at + 1))" -save-state "$at:/wp/states/$name.state" -shots "$at:/wp/states/$name.frame"
        -dump-at "$at:/wp/states/$name.rgb.png")
  [[ -n "$from" ]] && args+=(-load-state "/wp/states/$from.state")
  if [[ -n "$keys" ]]; then
    args+=(-press "$keys" -press-at "$key_at")
    [[ -n "$every" ]] && args+=(-press-every "$every")
  fi
  echo "[$name] 從 ${from:-開機} 跑到第 $at 道指令（按鍵：${keys:-無}）" >&2
  probe "${args[@]}" > "$OUT/$name.log" 2>&1 || { tail -20 "$OUT/$name.log" >&2; die "$name 失敗"; }
  [[ -s "$OUT/$name.state" && -s "$OUT/$name.frame" && -s "$OUT/$name.rgb.png" ]] || die "$name 沒有產出狀態檔或畫面（見 $OUT/$name.log）"
  hash=$(sha256sum "$OUT/$name.frame" | cut -c1-16)
  printf '%s\t%s\t%s\n' "$name" "$at" "$hash" >> "$manifest"
done

# 每格畫面轉成 PNG（EGA 預設 16 色；顏色未驗證，見 issue #4）
for row in "${STAGES[@]}"; do
  name=${row%%|*}
  "$ROOT/tools/py.sh" tools/frames.py montage "workplace/states/$name.png" "workplace/states/$name.frame" >/dev/null
done

column -t "$manifest"

if (( CHECK )); then
  [[ -f "$EXPECTED" ]] || die "沒有 $EXPECTED"
  if diff <(cut -f1-3 "$EXPECTED") "$manifest"; then
    echo "決定性檢查：全部畫面雜湊與 tools/states.expected 相同" >&2
  else
    die "決定性檢查失敗：畫面雜湊與 tools/states.expected 不同"
  fi
fi
