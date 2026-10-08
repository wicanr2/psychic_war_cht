#!/usr/bin/env bash
# 平台封包統一由package.sh建立，避免舊版號與平行交付入口。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exec "$ROOT/tools/package.sh" "${1:?請給完整版號及PSYCHICWAR_RELEASE_THEME}"
