#!/usr/bin/env bash
# 推廣片工具的 docker 包裝（skill `game-promo-video-ffmpeg`）。
#
#   tools/video.sh <指令…>        # 在 /src 底下跑，image 是 psychicwar-video
#
# CPU 限 2 核（ffmpeg 會吃滿所有核）。第一次會 build image（要網路）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# 預設是本專案自己的 image。這台機器上已經有別的專案建好的同類 image
# （ffmpeg ＋ ImageMagick ＋ Noto CJK 一樣齊），要借用就設 PSYCHICWAR_VIDEO_IMAGE。
# ⚠ 借來的只**執行**，不清理、不覆寫——那是別人的東西（`CLAUDE.md` 的 docker 硬規則）。
IMAGE="${PSYCHICWAR_VIDEO_IMAGE:-psychicwar-video}"
if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
  [ "$IMAGE" = psychicwar-video ] || { echo "找不到 image $IMAGE" >&2; exit 2; }
  echo "[video] 第一次使用，先 build $IMAGE（要網路）" >&2
  docker build -t "$IMAGE" -f "$ROOT/tools/docker/video.Dockerfile" "$ROOT/tools/docker"
fi
exec timeout "${PSYCHICWAR_TIMEOUT:-20m}" docker run --rm --network none \
  --memory 2g --cpus 2 --pids-limit 64 --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -e HOME=/tmp -v "$ROOT:/src" -w /src "$IMAGE" "$@"
