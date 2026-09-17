#!/usr/bin/env bash
# 依重播檔產生狀態檔檢查點（issue #2、#8；格式見 docs/spec/003）。
#
#   tools/states.sh [--check] [重播檔]     # 預設 replay/title-to-first-save.json
#
# 每一段從上一段的狀態檔接著跑、送一組按鍵、在固定指令數存狀態（-save-state）並存一格畫面，
# 另存經屬性暫存器與 DAC 解色的 <段>.rgb.png。--check 比對 tools/states.expected（決定性檢查）。
# `expect_same_frame_as` 有值時，這一段的畫面必須與指定段完全相同（例如讀檔後回到存檔時的畫面）。
# `expect_same_memory_as` 有值時，這一段在 save_at 傾印的 `memory_ranges` 必須與指定段完全相同（<段>.mem）。
#
# PSYCHICWAR_PROBE_EXTRA：每段額外加的 probe 參數（觀測用，不改變執行），`{name}` 代換成段名。
#   例：PSYCHICWAR_PROBE_EXTRA='-regs-at 0161:6289 -regs-max 5000' tools/states.sh（tools/print_trace.py 用）
# PSYCHICWAR_PROBE_ARGS_DIR：repo 內的目錄；有 <段名>.args（一行一個參數）就附加到該段。
# ⚠ 按鍵時機就是亂數的一部分（docs/re/004）：改任何一段，之後所有段的畫面都可能變，要一起更新期望值。
# ⚠ 狀態檔綁 dosgolem 版本。換版本後先跑 --check。
# ⚠ 暫存層（遊戲存檔）每次執行前清空，重播必須自己產生它要讀的存檔。
# ⚠ 本 repo 不含原版；讀 workplace/original/psychic-war/（玩家自備、唯讀掛載）。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOLEM="${PSYCHICWAR_DOSGOLEM:-$ROOT/worktrees/dosgolem}"
WP="$ROOT/workplace"
ORIG="${PSYCHICWAR_ORIG_DIR:-$WP/original}"
OUT="$WP/states"
EXPECTED="$ROOT/tools/states.expected"
CHECK=0
if [[ "${1:-}" == "--check" ]]; then CHECK=1; shift; fi
REPLAY="${1:-replay/title-to-first-save.json}"

die() { echo "tools/states.sh：$*" >&2; exit 2; }
[[ -f "$ORIG/psychic-war/PW.EXE" ]] || die "找不到 $ORIG/psychic-war/PW.EXE（先把 DOS 版解壓到 workplace/original/）"
[[ -x "$GOLEM/tools/go.sh" ]] || die "找不到 $GOLEM/tools/go.sh（先 git clone dosgolem 到 worktrees/）"
[[ -f "$ROOT/$REPLAY" ]] || die "找不到重播檔 $REPLAY"
mkdir -p "$OUT"
rm -rf "$OUT/scratch"
mkdir -p "$OUT/scratch"

probe() {
  DOSGOLEM_ORIG="$ORIG" DOSGOLEM_EXTRA_MOUNT="$WP:/wp" \
  DOSGOLEM_CPUS="${PSYCHICWAR_CPUS:-2}" DOSGOLEM_TIMEOUT="${PSYCHICWAR_TIMEOUT:-15m}" \
    "$GOLEM/tools/go.sh" run ./cmd/probe -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war "$@"
}

# 重播檔 → 一段一列：名稱、從哪段、存檔步數、按鍵、第一鍵、鍵距、暫存層、應相同的段、按住、記憶體應相同的段、傾印範圍
ROWS=$("$ROOT/tools/py.sh" -c '
import json, sys
d = json.load(open(sys.argv[1], encoding="utf-8"))
assert d.get("schema") == "psychic-war-replay/1", "schema 不對"
names = set()
ranges = d.get("memory_ranges", [])
for r in ranges:
    kind, addr, length = r.split(":")
    assert kind == "lin" and int(addr, 16) >= 0 and int(length) > 0, "memory_ranges 格式不對：%s" % r
for s in d["segments"]:
    s.setdefault("holds", [])
    s.setdefault("expect_same_memory_as", "")
    assert not s["expect_same_memory_as"] or (s["expect_same_memory_as"] in names and ranges), "%s 的 expect_same_memory_as 還沒出現或沒有 memory_ranges" % s["name"]
    for k in ("name", "from", "save_at", "keys", "key_at", "key_every", "scratch", "expect_same_frame_as"):
        assert k in s, "%s 缺 %s" % (s.get("name"), k)
    assert not s["from"] or s["from"] in names, "%s 的 from 還沒出現" % s["name"]
    assert not s["expect_same_frame_as"] or s["expect_same_frame_as"] in names, "%s 的 expect_same_frame_as 還沒出現" % s["name"]
    names.add(s["name"])
    print("|".join([s["name"], s["from"], str(s["save_at"]), ",".join(s["keys"]), str(s["key_at"]),
                    str(s["key_every"]), "1" if s["scratch"] else "", s["expect_same_frame_as"], ",".join(s["holds"]),
                    s["expect_same_memory_as"], ";".join(ranges)]))
' "$REPLAY")

manifest="$OUT/manifest.tsv"
: > "$manifest"
fail=0
while IFS='|' read -r name from at keys key_at every scratch same holds memsame ranges; do
  [[ -n "$name" ]] || continue
  args=(-steps "$((at + 1))" -save-state "$at:/wp/states/$name.state" -shots "$at:/wp/states/$name.frame"
        -dump-at "$at:/wp/states/$name.rgb.png")
  [[ -n "$from" ]] && args+=(-load-state "/wp/states/$from.state")
  if [[ -n "$keys" ]]; then
    args+=(-press "$keys" -press-at "$key_at")
    [[ "$every" != "0" ]] && args+=(-press-every "$every")
  fi
  [[ -n "$holds" ]] && args+=(-hold "$holds")
  [[ -n "$scratch" ]] && args+=(-scratch /wp/states/scratch)
  dumpspecs=()   # -dump-mem-at 只能給一次（Go 旗標保留最後一個），所有來源合併成一個
  if [[ -n "${PSYCHICWAR_PROBE_ARGS_DIR:-}" && -f "$ROOT/$PSYCHICWAR_PROBE_ARGS_DIR/$name.args" ]]; then
    mapfile -t extra < "$ROOT/$PSYCHICWAR_PROBE_ARGS_DIR/$name.args"   # 一行一個參數
    for ((j = 0; j < ${#extra[@]}; j++)); do
      if [[ "${extra[j]}" == "-dump-mem-at" ]]; then dumpspecs+=("${extra[j+1]}"); j=$((j + 1)); else args+=("${extra[j]}"); fi
    done
  fi
  if [[ -n "${PSYCHICWAR_PROBE_EXTRA:-}" ]]; then
    read -ra extra <<< "${PSYCHICWAR_PROBE_EXTRA//\{name\}/$name}"
    args+=("${extra[@]}")
  fi
  memshots=(); i=0
  if [[ -n "$ranges" ]]; then
    IFS=';' read -ra rs <<< "$ranges"
    for r in "${rs[@]}"; do memshots+=("$at:$r:/wp/states/$name.mem$i"); i=$((i + 1)); done
    dumpspecs+=("$(IFS=';'; echo "${memshots[*]}")")
  fi
  [[ ${#dumpspecs[@]} -gt 0 ]] && args+=(-dump-mem-at "$(IFS=';'; echo "${dumpspecs[*]}")")
  echo "[$name] 從 ${from:-開機} 跑到第 $at 道指令（按鍵：${keys:-無}${holds:+；按住 $holds}${scratch:+；暫存層}）" >&2
  probe "${args[@]}" > "$OUT/$name.log" 2>&1 || { tail -20 "$OUT/$name.log" >&2; die "$name 失敗"; }
  [[ -s "$OUT/$name.state" && -s "$OUT/$name.frame" && -s "$OUT/$name.rgb.png" ]] || die "$name 沒有產出狀態檔或畫面（見 $OUT/$name.log）"
  if [[ -n "$ranges" ]]; then
    : > "$OUT/$name.mem"
    for ((j = 0; j < i; j++)); do cat "$OUT/$name.mem$j" >> "$OUT/$name.mem" && rm -f "$OUT/$name.mem$j"; done
    [[ -s "$OUT/$name.mem" ]] || die "$name 沒有產出記憶體傾印"
  fi
  hash=$(sha256sum "$OUT/$name.frame" | cut -c1-16)
  printf '%s\t%s\t%s\n' "$name" "$at" "$hash" >> "$manifest"
  if [[ -n "$same" ]]; then
    other=$(awk -F'\t' -v n="$same" '$1==n{print $3}' "$manifest")
    if [[ "$hash" == "$other" ]]; then
      echo "[$name] 畫面與 $same 相同（$hash）" >&2
    else
      echo "[$name] ✗ 畫面應與 $same 相同：$hash ≠ $other" >&2
      fail=1
    fi
  fi
  if [[ -n "$memsame" ]]; then
    if cmp -s "$OUT/$name.mem" "$OUT/$memsame.mem"; then
      echo "[$name] 記憶體（$ranges）與 $memsame 相同" >&2
    else
      echo "[$name] ✗ 記憶體應與 $memsame 相同：" >&2
      cmp -l "$OUT/$name.mem" "$OUT/$memsame.mem" | head -5 >&2 || true
      fail=1
    fi
  fi
done <<< "$ROWS"

# 每格畫面轉成 PNG（EGA 預設 16 色，看版面用；正確顏色看 <段>.rgb.png）
while IFS='|' read -r name _; do
  [[ -n "$name" ]] || continue
  "$ROOT/tools/py.sh" tools/frames.py montage "workplace/states/$name.png" "workplace/states/$name.frame" >/dev/null
done <<< "$ROWS"

column -t "$manifest"
(( fail == 0 )) || die "expect_same_frame_as／expect_same_memory_as 檢查失敗"

if (( CHECK )); then
  [[ -f "$EXPECTED" ]] || die "沒有 $EXPECTED"
  if diff <(cut -f1-3 "$EXPECTED") "$manifest"; then
    echo "決定性檢查：全部畫面雜湊與 tools/states.expected 相同" >&2
  else
    die "決定性檢查失敗：畫面雜湊與 tools/states.expected 不同"
  fi
fi
