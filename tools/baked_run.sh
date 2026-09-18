#!/usr/bin/env bash
# 圖檔內嵌文字的逐像素驗收（docs/spec/011 §5 第 2 項）。
#
#   tools/baked_run.sh [狀態檔] [動作]                    # 預設 states/07-first-play.state、wait:2000
#   PSYCHICWAR_WITHOUT=<key> tools/baked_run.sh           # 反向對照：那一筆當作沒有譯文
#
# 1. dosgolem cmd/step 從狀態檔跑出原版畫面（320×200）
# 2. cmd/pwstep 以同一狀態、同一動作跑出 960×600 截圖與轉譯紀錄
# 3. tools/overlay_check.py --baked 逐像素比（期望值由原版畫面推算）
# 產出 workplace/overlay/baked[-without]/：ref.png、shot.png、text.jsonl、result.txt
# ⚠ 先 build：workplace/bin/step、workplace/bin/pwstep
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE="${1:-states/07-first-play.state}"
DO="${2:-wait:2000}"
for b in pwstep step; do [[ -x "$ROOT/workplace/bin/$b" ]] || { echo "先 build workplace/bin/$b" >&2; exit 2; }; done
WITHOUT="${PSYCHICWAR_WITHOUT:-}"
OUT="workplace/overlay/baked"; TEXT=text; args=()
if [[ -n "$WITHOUT" ]]; then
  OUT="workplace/overlay/baked-without"; TEXT="$OUT/text"; args=(--without "$WITHOUT")
fi
rm -rf "$ROOT/$OUT"; mkdir -p "$ROOT/$OUT"
if [[ -n "$WITHOUT" ]]; then
  cp -r "$ROOT/text" "$ROOT/$TEXT"
  "$ROOT/tools/py.sh" -c '
import json, pathlib, sys
key = sys.argv[1]; p = pathlib.Path("/src") / sys.argv[2] / "baked.json"
doc = json.loads(p.read_text(encoding="utf-8"))
for e in doc["entries"]:
    if e["key"] == key:
        e["translation"] = ""
p.write_text(json.dumps(doc, ensure_ascii=False, indent=1) + "\n", encoding="utf-8")' "$WITHOUT" "$TEXT"
fi
timeout 20m docker run --rm --network none --memory 2g --cpus 2 --pids-limit 128 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp \
  -v "$ROOT:/src" -v "$ROOT/workplace/original:/src/workplace/original:ro" -v "$ROOT/workplace/original:/orig:ro" \
  -w /src psychicwar-go-ebiten sh -c "
    set -e
    workplace/bin/step -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war -scratch /tmp/s1 \
      -load-state workplace/$STATE -cycles 750 -do '$DO' -shot $OUT/ref.png -scale 1 > $OUT/step.log 2>&1
    workplace/bin/pwstep -orig /orig/psychic-war -scratch /tmp/s2 -load-state workplace/$STATE \
      -do '$DO' -text $TEXT -text-log $OUT/text.jsonl -shot $OUT/shot.png > $OUT/pwstep.log 2>&1
  "
set +e
"$ROOT/tools/py.sh" tools/overlay_check.py --baked "$OUT/ref.png" "$OUT/shot.png" --text "$TEXT" "${args[@]}" | tee "$ROOT/$OUT/result.txt"
exit "${PIPESTATUS[0]}"
