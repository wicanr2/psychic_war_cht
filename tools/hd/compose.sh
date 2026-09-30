#!/usr/bin/env bash
# HD 主題的合成與驗收（docs/spec/024 §6）。
#
#   tools/hd/compose.sh [輸出目錄]
#
# 順序：原版 → 外框（遮罩後）→ 去抖色 → 向量基元 → 人物（去背）
# 最後跑 tools/hd/check.py：**在所有「有主」的矩形之外，原版是黑的像素必須還是黑的**。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT="${1:-workplace/hd/redraw}"
IMAGE="${PSYCHICWAR_VIDEO_IMAGE:-psychicwar-video}"
cd "$ROOT"

im() {
  timeout 5m docker run --rm --network none --memory 2g --cpus 2 --pids-limit 128 \
    --log-opt max-size=10m --log-opt max-file=3 \
    -u "$(id -u):$(id -g)" -e HOME=/tmp -v "$ROOT:/src" -w /src "$IMAGE" "$@"
}

KEEP=$(cat workplace/hd/keep-rects.txt)

echo "[1/6] 外框壓功能遮罩"
tools/py.sh tools/hd/mask.py workplace/hd/bg.idx 320 200 \
  workplace/hd/art-in/chrome.png "$OUT/chrome-masked.png" 3 $KEEP

echo "[2/6] 去抖色（限有主矩形）"
tools/py.sh tools/hd/dedither.py workplace/hd/bg.idx 320 200 \
  "$OUT/dedither.png" 3 3 workplace/hd/dedither-rects.txt

echo "[3/6] 框線（自動掃出硬斜角，改三段漸層）"
tools/py.sh tools/hd/bevels.py workplace/hd/bg.idx 320 200 workplace/hd/frames.json 12
tools/py.sh tools/hd/render.py workplace/hd/frames.json "$OUT/frames.png" 3

echo "[4/6] 向量基元"
tools/py.sh tools/hd/render.py theme/hd/draw/MENU-00.json "$OUT/MENU-00-vec.png" 3

echo "[5/6] 人物去背（邊緣填充，保留內部黑線）"
im convert workplace/hd/art-in/girl.png -alpha set -channel RGBA -fuzz 6% \
  -fill none -floodfill +0+0 black -fill none -floodfill +263+0 black \
  -fill none -floodfill +0+455 black -fill none -floodfill +263+455 black \
  +channel "$OUT/girl-raw.png"
# 框線底邊在原版 y=143（y=144 整列全黑），人物不能超過。等比縮到高 432 再靠右對齊，
# 免得壓在框線上（docs/re/038 §14）。
im convert "$OUT/girl-raw.png" -resize x432 "$OUT/girl-cut.png"

echo "[6/6] 合成"
im convert workplace/hd/full-near3.png \
  "$OUT/chrome-masked.png" -composite \
  "$OUT/dedither.png" -composite \
  "$OUT/frames.png" -composite \
  "$OUT/MENU-00-vec.png" -geometry +480+12 -composite \
  "$OUT/girl-cut.png" -gravity NorthEast -geometry +0+0 -composite +gravity \
  "$OUT/full-hd.png"
im convert "$OUT/full-hd.png" -type TrueColor "PNG24:$OUT/full-hd-rgb.png"

echo "[驗收]"
tools/py.sh tools/hd/check.py workplace/hd/bg.idx 320 200 \
  workplace/hd/full-near3-rgb.png "$OUT/full-hd-rgb.png" workplace/hd/owned-rects.txt
