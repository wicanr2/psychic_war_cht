#!/usr/bin/env bash
# 中文疊字實跑驗收（docs/spec/008 §4 第 2 項）。
#
#   tools/frontend-overlay-check.sh wall|launchpad [--expect-english KEY]
#
# 1. 原版參照：probe 跑同一狀態與按鍵，存色號畫面與 RGB（workplace/overlay/ref-<情境>.*，已存在就沿用）。
# 2. Xvfb 跑前端（scale 3、-audio null），送鍵，等轉譯紀錄出現全部目標 key 的 stamp 後 2 秒截圖。
#    --expect-english KEY：用暫存的 text 副本，把 KEY 的譯文清空（反向對照：該行應該是原版英文、紀錄有 missing-translation）。
# 3. tools/overlay_check.py 逐像素比。
# 產出 workplace/overlay/check-<情境>[-rev]/：shot.png、text.log、frontend.log、result.txt
# 狀態檔先用 probe 產生：workplace/overlay/wall-before.state、launchpad-before.state（docs/re/018）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SC="${1:?情境：wall 或 launchpad}"; shift
EN=""
[[ "${1:-}" == "--expect-english" ]] && EN="$2"
WP="$ROOT/workplace/overlay"
OUT="$WP/check-$SC${EN:+-rev}"
rm -rf "$OUT"; mkdir -p "$OUT"
[[ -x "$ROOT/workplace/bin/psychicwar" ]] || { echo "先 build 前端到 workplace/bin/psychicwar" >&2; exit 2; }

case "$SC" in
  wall)      STATE=wall-before.state;      REF_ARGS=(-press up -press-at 58000100 -steps 61000001); REF_AT=61000000; KEYS="Up"; KEYS_WANT="I_MENUH.BIN:0530" ;;
  launchpad) STATE=launchpad-before.state; REF_ARGS=(-steps 565000001); REF_AT=565000000; KEYS=""; KEYS_WANT="I_MENUH.BIN:09F0 I_MENUH.BIN:0A10 I_MENUH.BIN:0A40 I_MENUH.BIN:0A60 I_MENUH.BIN:0A80 I_MENUH.BIN:0A90" ;;
  *) echo "情境要是 wall 或 launchpad" >&2; exit 2 ;;
esac
[[ -f "$WP/$STATE" ]] || { echo "缺 $WP/$STATE" >&2; exit 2; }

if [[ ! -f "$WP/ref-$SC.frame" ]]; then
  DOSGOLEM_ORIG="$ROOT/workplace/original" DOSGOLEM_EXTRA_MOUNT="$ROOT/workplace:/wp" DOSGOLEM_CPUS=2 \
    "$ROOT/worktrees/dosgolem/tools/go.sh" run ./cmd/probe -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war \
    -load-state "/wp/overlay/$STATE" "${REF_ARGS[@]}" -shots "$REF_AT:/wp/overlay/ref-$SC.frame" -dump-at "$REF_AT:/wp/overlay/ref-$SC.rgb.png" \
    > "$WP/ref-$SC.log" 2>&1
fi

TEXT=text
if [[ -n "$EN" ]]; then
  TEXT="workplace/overlay/text-rev"
  rm -rf "$ROOT/$TEXT"; cp -r "$ROOT/text" "$ROOT/$TEXT"
  "$ROOT/tools/py.sh" -c '
import json, pathlib, sys
key = sys.argv[1]; d = pathlib.Path(sys.argv[2])
p = d / (key.split(":")[0] + ".json")
doc = json.loads(p.read_text(encoding="utf-8"))
for e in doc["entries"]:
    if e["key"] == key:
        e["translation"] = ""
p.write_text(json.dumps(doc, ensure_ascii=False), encoding="utf-8")' "$EN" "$TEXT"
fi

REL="workplace/overlay/check-$SC${EN:+-rev}"
PSYCHICWAR_TIMEOUT=10m PSYCHICWAR_SH="
set -eu
cd /src
workplace/bin/psychicwar -orig workplace/original/psychic-war -audio null -scratch /tmp/saves -load-state workplace/overlay/$STATE \
  -text $TEXT -text-log $REL/text.log -quit-after 120s > $REL/frontend.log 2>&1 &
PID=\$!
for i in \$(seq 1 100); do W=\$(xdotool search --name 'Psychic War' 2>/dev/null | head -1 || true); [ -n \"\$W\" ] && break; sleep 0.1; done
eval \"\$(xdotool getwindowgeometry --shell \"\$W\")\"
xdotool mousemove \$((X + WIDTH / 2)) \$((Y + HEIGHT / 2))
sleep 2
for k in $KEYS; do xdotool keydown \$k; sleep 0.15; xdotool keyup \$k; sleep 0.5; done
ok=0
for i in \$(seq 1 600); do
  kill -0 \$PID 2>/dev/null || { echo '前端已結束'; tail -5 $REL/frontend.log; exit 3; }
  all=1
  for key in $KEYS_WANT; do grep -qF \"\\\"key\\\":\\\"\$key\\\"\" $REL/text.log 2>/dev/null || { all=0; break; }; done
  [ \$all = 1 ] && { ok=1; break; }
  sleep 0.2
done
[ \$ok = 1 ] || { echo '等不到全部目標 key 的紀錄'; }
sleep 2
import -window \"\$W\" $REL/shot.png
convert $REL/shot.png -depth 8 rgb:$REL/shot.rgb
convert workplace/overlay/ref-$SC.rgb.png -depth 8 rgb:$REL/ref.rgb
kill \$PID; wait \$PID 2>/dev/null || true
" "$ROOT/tools/go-ebiten.sh"

set +e
"$ROOT/tools/py.sh" tools/overlay_check.py "$SC" workplace/original/psychic-war "$TEXT" "workplace/overlay/ref-$SC.frame" "$REL/ref.rgb" "$REL/shot.rgb" ${EN:+--expect-english "$EN"} | tee "$OUT/result.txt"
rc=${PIPESTATUS[0]}
echo "轉譯紀錄：$(grep -c '"stamp"' "$OUT/text.log" 2>/dev/null) 筆 stamp、$(grep -c 'missing-translation' "$OUT/text.log" 2>/dev/null) 筆缺譯文" | tee -a "$OUT/result.txt"
exit $rc
