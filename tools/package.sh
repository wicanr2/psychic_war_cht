#!/usr/bin/env bash
# 發行包（docs/spec/021）。**產出：`dist-all/`**，每個平台只留最新一份。
#
#   tools/package.sh appimage   # Linux：PsychicWar-<版本>-x86_64.AppImage
#   tools/package.sh macos      # PsychicWar-<版本>-macos.zip（universal）
#   tools/package.sh promo      # 把推廣片收進 dist-all（要先跑 tools/promo/make.sh）
#   tools/package.sh all
#
#   PSYCHICWAR_WITH_DATA=1 tools/package.sh all    # 另外再出一份含原版素材的本機完整版
#
# 兩種變體（使用者定案 2026-09-19）：
#   *-x86_64.AppImage / *-macos.zip             可散布，**不含原版素材**，玩家自備
#   *-with-data-x86_64.AppImage / *-macos.zip   含原版素材，**純本機自用**，絕不推 git、絕不上傳
# `dist-all/` 整個 gitignore。可散布版產出後會掃一次有沒有夾帶原版檔（leak-scan）。
#
# Linux 只出 AppImage，不出 tar.gz（kb `mac-app-cross-pack`「產物統一放 dist-all/」的 ship matrix）。
# 建置一律在 docker 裡（`--rm`、目前 UID/GID、預設 `--network none`）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
TARGET="${1:-all}"
VER="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
DIST="dist-all"
STAGE="workplace/pkg-stage"      # 中間產物放 workplace，壓完就清，不留在 dist-all
ORIG="workplace/original/psychic-war"
mkdir -p "$DIST"

# keep_latest：同一個平台只留最新一份（kb 的慣例）。$1 是 glob，$2 是這次要留的檔名。
keep_latest() {
  local f
  for f in $1; do
    [ -e "$f" ] || continue
    [ "$f" = "$2" ] || { echo "[package] 清掉舊的 $f"; rm -f "$f"; }
  done
}

# stage_data：發行包裡的資料。text/ 與 font/ 是自製素材（譯文、字型子集）。
# text/ 的 original 欄位是原版字串，使用者定案 2026-09-19 直接帶（docs/spec/021 §2）。
stage_data() { # $1 = 目的目錄
  mkdir -p "$1/text" "$1/font"
  cp text/*.json "$1/text/"
  cp font/*.golemfnt "$1/font/"
  cp README.md LICENSE "$1/"
}

# stage_orig：本機完整版才呼叫。原版素材放執行檔旁的 original/，
# 由 apps/psychicwar/paths.go 的 OrigDir() 找到，玩家不必給 -orig。
stage_orig() { # $1 = 目的目錄
  [ -d "$ORIG" ] || { echo "缺原版 $ORIG" >&2; exit 2; }
  mkdir -p "$1/original"
  cp -r "$ORIG"/. "$1/original/"   # 要的是 original/PW.EXE，不是 original/psychic-war/PW.EXE
}

# leak_scan：可散布的包裡不可以有原版檔（CLAUDE.md [HARD]）。
# 判準是原版目錄裡實際有哪些檔名，不是猜副檔名。
leak_scan() { # $1 = 要掃的目錄
  local names hit
  names=$(cd "$ORIG" 2>/dev/null && ls) || return 0
  hit=$(cd "$1" && for n in $names; do find . -name "$n" -print; done)
  [ -z "$hit" ] || { echo "[package] 可散布的包裡夾帶原版檔：$hit" >&2; exit 1; }
}

build_linux() { # $1 = 輸出執行檔路徑（repo 相對）
  PSYCHICWAR_TIMEOUT=20m tools/go-ebiten.sh build \
    -ldflags "-s -w -X main.version=$VER" -o "/src/$1" ./cmd/psychicwar
}

# appdir：組 AppDir。$1 = AppDir 路徑，$2 = with-data 就塞原版素材
appdir() {
  local app="$1"
  rm -rf "$app"; mkdir -p "$app/usr/bin"
  build_linux "$app/usr/bin/psychicwar"
  stage_data "$app/usr/bin"
  [ "${2:-}" = with-data ] && stage_orig "$app/usr/bin"
  # AppRun 只轉呼叫：資料在執行檔旁（docs/spec/021 §3.1 第 1 條），
  # 存檔由程式自己放到使用者資料目錄（§3.2），squashfs 是唯讀的。
  cat > "$app/AppRun" <<'SH'
#!/bin/sh
HERE="$(dirname "$(readlink -f "$0")")"
exec "$HERE/usr/bin/psychicwar" "$@"
SH
  chmod +x "$app/AppRun"
  cat > "$app/psychicwar.desktop" <<'DESKTOP'
[Desktop Entry]
Type=Application
Name=Psychic War
Comment=銀河超能力戰記 繁體中文化
Exec=psychicwar
Icon=psychicwar
Categories=Game;
Terminal=false
DESKTOP
  tools/py.sh tools/appicon.py "$app/psychicwar.png" >/dev/null
}

do_appimage() {
  local app="$STAGE/PsychicWar.AppDir"
  local out="$DIST/PsychicWar-$VER-x86_64.AppImage"
  appdir "$app"
  leak_scan "$app"
  keep_latest "$DIST/PsychicWar-*-x86_64.AppImage" "$out"
  ARCH=x86_64 tools/appimagetool.sh "$app" "$out"
  rm -rf "$app"
  echo "$out"
  if [ "${PSYCHICWAR_WITH_DATA:-}" = 1 ]; then
    local outd="$DIST/PsychicWar-$VER-with-data-x86_64.AppImage"
    appdir "$app" with-data
    keep_latest "$DIST/PsychicWar-*-with-data-x86_64.AppImage" "$outd"
    ARCH=x86_64 tools/appimagetool.sh "$app" "$outd"
    rm -rf "$app"
    echo "$outd"
  fi
}

do_macos() {
  tools/macos-pack.sh "$VER"
  [ "${PSYCHICWAR_WITH_DATA:-}" = 1 ] && PSYCHICWAR_PACK_WITH_DATA=1 tools/macos-pack.sh "$VER"
  return 0
}

# 推廣片也歸到 dist-all（kb 的典型內容就含 *-promo.mp4）。
do_promo() {
  local src=workplace/promo/psychic-war-promo.mp4
  local out="$DIST/psychic-war-$VER-promo.mp4"
  [ -f "$src" ] || { echo "還沒有推廣片，先跑 tools/video.sh sh tools/promo/make.sh" >&2; return 0; }
  keep_latest "$DIST/psychic-war-*-promo.mp4" "$out"
  cp "$src" "$out"
  echo "$out"
}

case "$TARGET" in
  appimage) do_appimage ;;
  macos) do_macos ;;
  promo) do_promo ;;
  all) do_appimage; do_macos; do_promo ;;
  *) echo "目標要是 appimage、macos、promo 或 all" >&2; exit 2 ;;
esac
rmdir "$STAGE" 2>/dev/null || true
echo "== dist-all"
ls -la "$DIST"
