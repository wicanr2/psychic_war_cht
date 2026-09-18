#!/usr/bin/env bash
# AppDir → AppImage（docs/spec/021 §4）：官方 type2 runtime 串接 squashfs 映像。
#
#   tools/appimagetool.sh <AppDir> <輸出.AppImage>
#
# image 是本專案自己的 `psychicwar-appimage`，第一次自動 build（那一步要網路，之後不用）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APPDIR="${1:?AppDir}"; OUT="${2:?輸出路徑}"
IMAGE="psychicwar-appimage"
if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
  echo "[appimage] 第一次使用，先 build $IMAGE（要網路）" >&2
  docker build -t "$IMAGE" -f "$ROOT/tools/docker/appimage.Dockerfile" "$ROOT/tools/docker"
fi
timeout 15m docker run --rm --network none --memory 2g --cpus 2 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp \
  -v "$ROOT:/src" -w /src "$IMAGE" sh -c "
set -eu
mksquashfs '$APPDIR' /tmp/app.squashfs -root-owned -noappend -no-progress -comp zstd -Xcompression-level 19
cat /opt/runtime-x86_64 /tmp/app.squashfs > '$OUT'
chmod +x '$OUT'
file '$OUT'
"
