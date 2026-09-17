#!/usr/bin/env bash
# Go ＋ Ebiten 走 docker（docs/spec/006）：帶 X11／GL／ALSA 標頭檔的 image，容器內前景啟動 Xvfb。
#
#   tools/go-ebiten.sh test ./apps/...
#   tools/go-ebiten.sh build -o /src/workplace/bin/psychicwar ./cmd/psychicwar
#   PSYCHICWAR_GO_NETWORK=1 tools/go-ebiten.sh mod tidy      # 第一次抓模組才需要網路
#   PSYCHICWAR_SH='指令' tools/go-ebiten.sh                  # 在同一個 Xvfb 裡跑任意 shell（前端實跑、截圖）
#
# image 是本專案自己的 `psychicwar-go-ebiten`（tools/docker/go-ebiten.Dockerfile），第一次自動 build。
# 整個 repo 掛在 /src（含 worktrees/dosgolem，go.mod 以 replace 指到它）；workplace/original 另外唯讀掛載覆蓋。
# 只清理自己建立的 container（--rm）；不碰任何共用 docker 資源。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="psychicwar-go-ebiten"
if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
  echo "[go-ebiten] 第一次使用，先 build $IMAGE" >&2
  docker build -t "$IMAGE" -f "$ROOT/tools/docker/go-ebiten.Dockerfile" "$ROOT/tools/docker"
fi
mkdir -p "$ROOT/workplace/gocache" "$ROOT/workplace/gomodcache"

NET=(--network none)
[[ "${PSYCHICWAR_GO_NETWORK:-}" == "1" ]] && NET=()
ORIG=()
# 原版也掛在 /orig：probe 產生的狀態檔記著 /orig/psychic-war，載入後遊戲照那個路徑開檔
[[ -d "$ROOT/workplace/original" ]] && ORIG=(-v "$ROOT/workplace/original:/src/workplace/original:ro" -v "$ROOT/workplace/original:/orig:ro")

exec timeout "${PSYCHICWAR_TIMEOUT:-15m}" docker run --rm "${NET[@]}" \
  --memory "${PSYCHICWAR_MEMORY:-3g}" --cpus "${PSYCHICWAR_CPUS:-2}" --pids-limit 512 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  -v "$ROOT:/src" "${ORIG[@]}" \
  -v "$ROOT/workplace/gocache:/gocache" -v "$ROOT/workplace/gomodcache:/gomodcache" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomodcache -e HOME=/tmp -e GOFLAGS=-mod=mod \
  -e PSYCHICWAR_SH="${PSYCHICWAR_SH:-}" \
  -w /src "$IMAGE" sh -c '
    set -eu
    mkdir -p /tmp/.X11-unix
    Xvfb :99 -screen 0 1280x800x24 -nolisten tcp -ac >/tmp/xvfb.log 2>&1 &
    xvfb_pid=$!
    trap "kill $xvfb_pid 2>/dev/null || true" EXIT INT TERM
    i=0; while [ ! -S /tmp/.X11-unix/X99 ] && [ $i -lt 50 ]; do sleep 0.1; i=$((i+1)); done
    [ -S /tmp/.X11-unix/X99 ] || { cat /tmp/xvfb.log >&2; exit 1; }
    export DISPLAY=:99
    if [ -n "$PSYCHICWAR_SH" ]; then sh -c "$PSYCHICWAR_SH"; else go "$@"; fi
  ' go-ebiten "$@"
