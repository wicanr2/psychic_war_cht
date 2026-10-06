#!/usr/bin/env bash
# 烘製 font/cjk24.golemfnt、font/cjk16.golemfnt（docs/spec/009 §2）。需要網路在容器裡裝 Pillow。
#   tools/font/bake.sh [額外字…]
# 字型來源放 workplace/font-src/：STD.24M（倚天 3.53，ETUNPACK 壓縮）、STDFONT.15、SPCFONT.15、NotoSansCJKtc-Regular.otf。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SRC="$ROOT/workplace/font-src"
for f in STD.24M STDFONT.15 SPCFONT.15 NotoSansCJKtc-Regular.otf; do
  [[ -f "$SRC/$f" ]] || { echo "tools/font/bake.sh：缺 $SRC/$f" >&2; exit 2; }
done
if [[ -e "$ROOT/font" ]]; then
  [[ -d "$ROOT/font" && "$(stat -c %u:%g "$ROOT/font")" == "$(id -u):$(id -g)" ]] || { echo "font 目錄形態或擁有權不符" >&2; exit 2; }
fi
for f in cjk24.golemfnt cjk16.golemfnt charset.txt; do
  [[ ! -e "$ROOT/font/$f" || ( -f "$ROOT/font/$f" && "$(stat -c %u:%g "$ROOT/font/$f")" == "$(id -u):$(id -g)" ) ]] || { echo "font/$f 形態或擁有權不符" >&2; exit 2; }
done
ORIGINAL=()
if [[ -e "$ROOT/workplace/original" ]]; then
  [[ -d "$ROOT/workplace/original" ]] || { echo "原版掛載來源不是目錄" >&2; exit 2; }
  ORIGINAL=(-v "$ROOT/workplace/original:/src/workplace/original:ro")
fi
mkdir -p "$ROOT/font"
exec timeout 10m docker run --rm --memory 1g --cpus 1 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -e HOME=/tmp -e PYTHONDONTWRITEBYTECODE=1 \
  -v "$ROOT:/src" -v "$SRC:/font-src:ro" "${ORIGINAL[@]}" -w /src python:3.13-slim \
  sh -c 'pip install -q --target /tmp/pk pillow==11.3.0 >/dev/null 2>&1 && PYTHONPATH=/tmp/pk python3 tools/font/bake_cjk.py /font-src /font-src/NotoSansCJKtc-Regular.otf text font "$@"' sh "$@"
