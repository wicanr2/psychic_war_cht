#!/usr/bin/env bash
# 中文疊字逐像素驗收（docs/spec/009 §6 第 2 項）。
#
#   tools/overlay_run.sh [情境…]              # 預設跑 tools/overlay_cases.json 的全部情境
#   PSYCHICWAR_WITHOUT=1 tools/overlay_run.sh  # 反向對照：每個情境的第一個 target 當作沒有譯文
#
# 每個情境：
#   1. dosgolem cmd/step 從起始狀態跑動作腳本，存 320×200 參照畫面（原版英文）
#   2. cmd/pwstep 以同一狀態、同一動作，存 960×600 截圖（原版放大＋疊字層）與轉譯紀錄
#   3. tools/overlay_check.py 逐像素比
# 產出 workplace/overlay/run/<情境>/：ref.png、shot.png、text.jsonl、result.txt
# ⚠ 先 build：tools/go-ebiten.sh build -o /src/workplace/bin/pwstep ./cmd/pwstep；
#   tools/go-ebiten.sh build -o /src/workplace/bin/step github.com/wicanr2/dosgolem/cmd/step
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
for b in pwstep step; do [[ -x "$ROOT/workplace/bin/$b" ]] || { echo "先 build workplace/bin/$b" >&2; exit 2; }; done
CASES=("$@")
if [[ ${#CASES[@]} -eq 0 ]]; then
  mapfile -t CASES < <("$ROOT/tools/py.sh" -c 'import json; print("\n".join(json.load(open("/src/tools/overlay_cases.json"))["cases"]))')
fi
fail=0
for c in "${CASES[@]}"; do
  row=$("$ROOT/tools/py.sh" -c 'import json,sys; x=json.load(open("/src/tools/overlay_cases.json"))["cases"][sys.argv[1]]; print(x["state"]+"|"+x["do"]+"|"+(x["targets"][0][0] if x["targets"] else ""))' "$c")
  IFS='|' read -r state do first <<<"$row"
  suffix=""; without=()
  if [[ "${PSYCHICWAR_WITHOUT:-}" == "1" ]]; then suffix="-without"; without=(--without "$first"); fi
  OUT="workplace/overlay/run/$c$suffix"
  rm -rf "$ROOT/$OUT"; mkdir -p "$ROOT/$OUT"
  TEXT=text
  if [[ -n "$suffix" ]]; then
    TEXT="$OUT/text"; cp -r "$ROOT/text" "$ROOT/$TEXT"
    "$ROOT/tools/py.sh" -c '
import json, pathlib, sys
key = sys.argv[1]; d = pathlib.Path("/src") / sys.argv[2]
p = d / (key.rsplit(":", 1)[0].split(":cs")[0] + ".json")
doc = json.loads(p.read_text(encoding="utf-8"))
for e in doc["entries"]:
    if e["key"] == key:
        e["translation"] = ""
p.write_text(json.dumps(doc, ensure_ascii=False), encoding="utf-8")' "$first" "$TEXT"
  fi
  timeout 20m docker run --rm --network none --memory 2g --cpus 2 --pids-limit 128 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp \
    -v "$ROOT:/src" -v "$ROOT/workplace/original:/src/workplace/original:ro" -v "$ROOT/workplace/original:/orig:ro" -w /src psychicwar-go-ebiten sh -c "
      set -e
      workplace/bin/step -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war -scratch /tmp/s1 \
        -load-state workplace/$state -cycles 750 -do '$do' -shot $OUT/ref.png -scale 1 > $OUT/step.log 2>&1
      workplace/bin/pwstep -orig /orig/psychic-war -scratch /tmp/s2 -load-state workplace/$state \
        -do '$do' -text $TEXT -text-log $OUT/text.jsonl -shot $OUT/shot.png > $OUT/pwstep.log 2>&1
    "
  set +e
  "$ROOT/tools/py.sh" tools/overlay_check.py "$c" "$OUT/ref.png" "$OUT/shot.png" --text "$TEXT" "${without[@]}" > "$ROOT/$OUT/result.txt" 2>&1
  rc=$?
  set -e
  echo "== $c$suffix（結束碼 $rc）"; cat "$ROOT/$OUT/result.txt"
  [[ $rc -eq 0 ]] || fail=1
done
exit $fail
