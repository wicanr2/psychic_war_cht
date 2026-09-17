#!/usr/bin/env bash
# Python 走 docker，不用主機的 Python。
#
#   tools/py.sh tools/worklist.py verify
#   tools/py.sh -m unittest tools/test_worklist.py
#
# 整個 repo 掛在 /src（可寫，render 要寫 docs/worklist.md）；不掛任何原版素材。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${PSYCHICWAR_PY_IMAGE:-python:3.13-alpine}"

exec timeout "${PSYCHICWAR_TIMEOUT:-5m}" docker run --rm -i --network none \
  --memory 512m --cpus 1 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -e HOME=/tmp -e PYTHONDONTWRITEBYTECODE=1 \
  -v "$ROOT:/src" -w /src "$IMAGE" python3 "$@"
