#!/usr/bin/env bash
# 發行包（docs/spec/021）。
#
#   tools/package.sh linux      # dist/psychicwar-<版本>-linux-x86_64.tar.gz
#   tools/package.sh appimage   # dist/PsychicWar-<版本>-x86_64.AppImage
#   tools/package.sh macos      # dist/PsychicWar-<版本>-macos.zip（universal）
#   tools/package.sh all
#
# 建置一律在 docker 裡（`--rm`、目前 UID/GID、預設 --network none）。
# **發行包不含原版素材**：PW.EXE、.PBL、.BIN、.MID、磁碟映像都不進去，玩家自備（docs/spec/021 §1）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
TARGET="${1:-all}"
VER="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
DIST="dist"
mkdir -p "$DIST"

# 發行包裡的資料：text/ 與 font/ 是自製素材（譯文、字型子集），可以散布。
# text/ 的 original 欄位是原版字串——使用者定案 2026-09-19 直接帶（docs/spec/021 §2）。
stage_data() { # $1 = 目的目錄
  mkdir -p "$1/text" "$1/font"
  cp text/*.json "$1/text/"
  cp font/*.golemfnt "$1/font/"
  cp README.md LICENSE "$1/"
}

build_linux() { # $1 = 輸出執行檔路徑（repo 相對）
  PSYCHICWAR_TIMEOUT=20m tools/go-ebiten.sh build \
    -ldflags "-s -w -X main.version=$VER" -o "/src/$1" ./cmd/psychicwar
}

do_linux() {
  local name="psychicwar-$VER-linux-x86_64"
  local stage="$DIST/stage/$name"
  rm -rf "$stage"; mkdir -p "$stage"
  build_linux "$stage/psychicwar"
  stage_data "$stage"
  tar -C "$DIST/stage" -czf "$DIST/$name.tar.gz" "$name"
  echo "$DIST/$name.tar.gz"
}

do_appimage() {
  local app="$DIST/stage/PsychicWar.AppDir"
  rm -rf "$app"; mkdir -p "$app/usr/bin"
  build_linux "$app/usr/bin/psychicwar"
  stage_data "$app/usr/bin"
  # AppRun 要把 cwd 無關的事都處理掉：資料在執行檔旁（docs/spec/021 §3.1 第 1 條），
  # 存檔由程式自己放到使用者資料目錄（§3.2），所以這裡只要轉呼叫。
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
  tools/py.sh tools/appicon.py "$app/psychicwar.png"
  local out="$DIST/PsychicWar-$VER-x86_64.AppImage"
  ARCH=x86_64 tools/appimagetool.sh "$app" "$out"
  echo "$out"
}

do_macos() {
  tools/macos-pack.sh "$VER"
}

case "$TARGET" in
  linux) do_linux ;;
  appimage) do_appimage ;;
  macos) do_macos ;;
  all) do_linux; do_appimage; do_macos ;;
  *) echo "目標要是 linux、appimage、macos 或 all" >&2; exit 2 ;;
esac
