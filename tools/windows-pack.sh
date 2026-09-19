#!/usr/bin/env bash
# Windows 發行包（docs/spec/021 §1、§4；issue #39）：交叉編出 win64 執行檔 →
# 組 portable 目錄 → `dist-all/PsychicWar-<版本>-win64.zip`。
# PSYCHICWAR_PACK_WITH_DATA=1 另外出含原版素材的本機完整版（絕不推 git、絕不上傳）。
#
#   tools/windows-pack.sh [版本]      # 版本預設 git describe --tags --always --dirty
#   tools/package.sh windows          # 同上，發行包的統一入口
#
# **不需要 mingw**：Ebiten v2.9.9 的 Windows 後端用 purego 在執行期載 DLL，
# `CGO_ENABLED=0 GOOS=windows go build` 就編得出來（實測，docs/re/035 §3.1）。
# 所以這支沿用 `tools/go-ebiten.sh` 的同一份 Go 工具鏈（`psychicwar-go-ebiten`，go1.24.13），
# 與 Linux／macOS 版是同一支編譯器。macOS 那邊非要 osxcross 不可，是因為 Cocoa 後端要 cgo。
#
# `-H windowsgui` 讓程式以 GUI 子系統連結，雙擊不會多開一個主控台視窗。
# 代價是**錯誤訊息沒有地方可去**（`rulebook/82` 第 1 點：同一段訊息在不同 OS 是不同嚴重度）。
# 所以包裡附 `troubleshoot.bat`：它把 stderr 導進 `psychicwar-log.txt` 再 `pause`，
# 玩家看得到「缺原版」「缺字型」這類訊息。詳見 docs/re/035 §4。
#
# ⚠ 這支跑完只做得到「wine 底下起得來」的驗收（`tools/windows-verify.sh --run`）。
# **沒有在真正的 Windows 上跑過**，wine 過關不等於 Windows 過關。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

VER="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
IMAGE="${PSYCHICWAR_GO_IMAGE:-psychicwar-go-ebiten}"
CPUS="${PSYCHICWAR_WIN_CPUS:-4}"
APP_NAME="PsychicWar"
EXE_NAME="PsychicWar.exe"

DIST="dist-all"
STAGE="workplace/pkg-stage/win64"   # 中間產物放 workplace，壓完就清
PKG="$STAGE/$APP_NAME"
ORIG="workplace/original/psychic-war"
if [ "${PSYCHICWAR_PACK_WITH_DATA:-}" = 1 ]; then
  ZIP="$DIST/$APP_NAME-$VER-with-data-win64.zip"
  GLOB="$DIST/$APP_NAME-*-with-data-win64.zip"
else
  ZIP="$DIST/$APP_NAME-$VER-win64.zip"
  GLOB="$DIST/$APP_NAME-*-win64.zip"
fi
mkdir -p "$DIST"

# --- 1. 交叉編 -------------------------------------------------------------------
if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
  echo "[windows-pack] 第一次使用，先 build $IMAGE" >&2
  docker build -t "$IMAGE" -f "$ROOT/tools/docker/go-ebiten.Dockerfile" "$ROOT/tools/docker"
fi
mkdir -p "$ROOT/workplace/gocache" "$ROOT/workplace/gomodcache"
rm -rf "$STAGE"
mkdir -p "$PKG"

timeout "${PSYCHICWAR_WIN_TIMEOUT:-20m}" docker run --rm --network none \
  --memory 4g --cpus "$CPUS" --pids-limit 512 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -e HOME=/tmp \
  -e "PW_VER=$VER" -e "PW_OUT=$PKG/$EXE_NAME" \
  -v "$ROOT:/src" \
  -v "$ROOT/workplace/gocache:/gocache" -v "$ROOT/workplace/gomodcache:/gomodcache" \
  -w /src "$IMAGE" \
  bash -c '
    set -euo pipefail
    export GOCACHE=/gocache GOMODCACHE=/gomodcache GOPATH=/tmp/gopath
    export GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local GOWORK=off GOFLAGS=-mod=mod
    export CGO_ENABLED=0 GOOS=windows GOARCH=amd64
    go build -trimpath -ldflags "-s -w -H windowsgui -X main.version=$PW_VER" \
      -o "/src/$PW_OUT" ./cmd/psychicwar
    ls -l "/src/$PW_OUT"
  '

# --- 2. 資料與說明 ---------------------------------------------------------------
# text/ 與 font/ 是自製素材（譯文、字型子集），放執行檔旁，
# 由 apps/psychicwar/paths.go 的第 1 條（執行檔所在目錄）找到（docs/spec/021 §3.1）。
mkdir -p "$PKG/text" "$PKG/font"
cp text/*.json "$PKG/text/"
cp font/*.golemfnt "$PKG/font/"
cp README.md LICENSE "$PKG/"

if [ "${PSYCHICWAR_PACK_WITH_DATA:-}" = 1 ]; then
  # 本機完整版：原版素材放執行檔旁的 original\，由 OrigDir() 找到，玩家不必給 -orig。
  # 這種包絕不推 git、絕不上傳（docs/spec/021 §1.1）。
  [ -d "$ORIG" ] || { echo "缺原版 $ORIG" >&2; exit 2; }
  mkdir -p "$PKG/original"
  cp -r "$ORIG"/. "$PKG/original/"
else
  # 可散布版不得夾帶原版檔（CLAUDE.md [HARD]）。判準是原版目錄裡實際有哪些檔名，不是猜副檔名。
  if [ -d "$ORIG" ]; then
    leak=$(cd "$PKG" && for n in $(cd "$ROOT/$ORIG" && ls); do find . -name "$n" -print; done)
    [ -z "$leak" ] || { echo "可散布的包裡夾帶原版檔：$leak" >&2; exit 1; }
  fi
fi

# troubleshoot.bat：`-H windowsgui` 的程式沒有主控台，log.Fatal 的訊息寫不到任何地方。
# 這支用重新導向把 stderr 接成檔案；GUI 子系統的行程照樣繼承 cmd 給的控制代碼，
# 所以導到檔案就收得到（實測，docs/re/035 §4）。內容全部 ASCII：.bat 由 cmd.exe
# 以系統 ANSI 代碼頁逐行解讀，中文寫在這裡在 cp950 以外的機器上會變亂碼。
cat > "$PKG/troubleshoot.bat" <<'BAT'
@echo off
rem Run the game with stderr captured, then show the log.
rem Use this when double-clicking PsychicWar.exe does nothing.
cd /d "%~dp0"
echo Running PsychicWar.exe ... log goes to psychicwar-log.txt
PsychicWar.exe %* > psychicwar-log.txt 2>&1
echo Exit code: %ERRORLEVEL%
echo ---------------- psychicwar-log.txt ----------------
type psychicwar-log.txt
pause
BAT

# 說明檔用 UTF-8 ＋ BOM：記事本沒有 BOM 時會拿 ANSI 代碼頁去猜，中文變亂碼。
# 檔名維持 ASCII，避開 zip 的檔名編碼問題。
{
  printf '\xEF\xBB\xBF'
  cat <<TXT
銀河超能力戰記 繁體中文化　Windows 版 $VER

一、怎麼開始
  1. 解開整個資料夾（不要只把 PsychicWar.exe 拉出來，text\\ 與 font\\ 要在它旁邊）。
  2. 把你自己那份原版（含 PW.EXE 的目錄）整個複製成這個資料夾底下的 original\\。
     例：PsychicWar\\original\\PW.EXE
  3. 雙擊 PsychicWar.exe。

  原版放別的地方也可以，用命令列指過去：
     PsychicWar.exe -orig D:\\路徑\\到\\原版

二、雙擊沒反應？
  這支程式是視窗程式（GUI），沒有主控台可以印訊息，所以「缺原版」「缺字型」
  這類錯誤在雙擊時看不到。改成雙擊 troubleshoot.bat：它會把訊息寫進
  psychicwar-log.txt 並停在畫面上。

  最常見的原因就是第一項的第 2 步沒做（找不到 PW.EXE）。

三、存檔放哪
  %APPDATA%\\PsychicWar
  （遊戲存檔 <名字>.DAT 與 F10 的即時存檔都在這裡。解開的資料夾本身不會被寫入，
   所以放在 Program Files 或唯讀磁碟也沒問題。）

四、熱鍵
  進遊戲後叫出說明頁就看得到完整清單（按哪個鍵寫在 README.md 裡）。
  這裡不重抄一份；熱鍵改過，抄本就會變成錯的。

五、已知限制
  這一版沒有在真正的 Windows 上跑過，只在 Linux 的 wine 底下驗過起得來、
  讀得到資料檔、畫得出中文。遇到問題請回報。
TXT
} > "$PKG/troubleshoot.txt"

# --- 3. 靜態驗收（過不了就不要產出 zip）------------------------------------------
tools/windows-verify.sh "$PKG"

# --- 4. zip ----------------------------------------------------------------------
# 同一個平台只留最新一份（docs/DEV-SETUP.md 的 dist-all 慣例）。
for old in $GLOB; do
  [ -e "$old" ] || continue
  case "$old" in *-with-data-win64.zip) [ "${PSYCHICWAR_PACK_WITH_DATA:-}" = 1 ] || continue ;; esac
  [ "$old" = "$ZIP" ] || { echo "[windows-pack] 清掉舊的 $old"; rm -f "$old"; }
done
rm -f "$ZIP"
( cd "$STAGE" && zip -qry "$ROOT/$ZIP" "$APP_NAME" )
rm -rf "$STAGE"          # staging 自清，別把解開的包留在磁碟
rmdir workplace/pkg-stage 2>/dev/null || true
echo "$ZIP"
