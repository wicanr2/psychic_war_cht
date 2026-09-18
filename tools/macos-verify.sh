#!/usr/bin/env bash
# macOS 交叉編產物的靜態驗收（docs/spec/021 §5 第 5 項）。
#
#   tools/macos-verify.sh dist/stage/macos/PsychicWar.app
#   tools/macos-verify.sh workplace/bin/psychicwar-macos
#
# 參數可以是 `.app` 目錄或執行檔；路徑要在本 repo 底下（repo 唯讀掛進容器）。
#
# ⚠ **Linux 上執行不了 macOS binary。** 這支只驗結構：全過只代表「不會因為結構問題
# 開不起來」，**不代表功能正常**。真機驗收要另外做，目前沒有 Mac 可以做。
#
# 五道（skill `osxcross-macos-cross-build` §5）：
#   1. 雙弧：`lipo -info` 要同時列出 x86_64 與 arm64。
#   2. arm64 必須有 `LC_CODE_SIGNATURE`。ld64 連結 arm64 時會自己補 ad-hoc 簽章，
#      少了它使用者在 Apple Silicon 上會看到 `Killed: 9`，而檔案格式完全正常，
#      在 Linux 這端看不出任何異狀。x86_64 沒有這個限制。
#   3. 最低系統版本（`minos`）讀得出來，而且與宣稱的一致。
#   4. 動態相依只能落在 `/usr/lib/` 與 `/System/Library/`；其他都是連到編譯機才有的
#      路徑，玩家的 Mac 上不存在（`dyld: Library not loaded`）。
#      `otool -L` 對 fat binary 會為每個架構印一行檔名標頭，所以要先 `lipo -thin` 拆單弧。
#   5. 內容證據：`strings` 找這份程式一定會有的字串，證明「這份 binary 真的含有那段
#      程式碼」，不必開機也能判。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET="${1:?用法：macos-verify.sh <執行檔或 .app>}"
IMAGE="${PSYCHICWAR_MAC_IMAGE:-psychicwar-osxcross}"
MIN="${PSYCHICWAR_MACOS_MIN:-11.0}"

case "$TARGET" in /*) ABS="$TARGET" ;; *) ABS="$ROOT/$TARGET" ;; esac
[[ -e "$ABS" ]] || { echo "找不到 $ABS" >&2; exit 1; }
case "$ABS" in "$ROOT"/*) REL="${ABS#"$ROOT"/}" ;; *) echo "目標要在 repo 底下（$ROOT）" >&2; exit 1 ;; esac

docker image inspect "$IMAGE" >/dev/null 2>&1 || { echo "還沒有 $IMAGE，先跑 tools/macos-pack.sh" >&2; exit 1; }

exec timeout 300 docker run --rm --network none \
  --memory 2g --cpus 2 --pids-limit 128 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -e HOME=/tmp \
  -e "PW_TARGET=$REL" -e "PW_MIN=$MIN" \
  -v "$ROOT:/src:ro" -w /src \
  --tmpfs "/tmp:exec,uid=$(id -u),gid=$(id -g),size=512m" \
  "$IMAGE" \
  bash -c '
    set -euo pipefail
    eval "$(osxcross-conf)"
    T=x86_64-apple-$OSXCROSS_TARGET
    B=/src/$PW_TARGET
    # `.app` 就往 Contents/MacOS 找；打包流程常把那支換成包裝腳本、真正的執行檔改名
    # `<name>.bin`，`lipo` 對腳本會報 `can not figure out the architecture type`，
    # 所以兩種佈局都要認。
    if [ -d "$B" ]; then
      case "$B" in *.app) ;; *) echo "目錄只認 .app" >&2; exit 1 ;; esac
      name=$(basename "$B" .app)
      exe=""
      for c in "$B/Contents/MacOS/"*; do [ -f "$c" ] && exe="$c" && break; done
      [ -n "$exe" ] || { echo "$B/Contents/MacOS 裡沒有執行檔" >&2; exit 1; }
      case "$(file -b "$exe")" in
        *"shell script"*|*"text"*)
          [ -f "$exe.bin" ] || { echo "$exe 是包裝腳本，但找不到 $exe.bin" >&2; exit 1; }
          exe="$exe.bin" ;;
      esac
      echo "== .app：$B"
      for f in Contents/MacOS Contents/Resources Contents/Info.plist; do
        [ -e "$B/$f" ] || { echo "缺 $f" >&2; exit 1; }
        echo "-- 有 $f"
      done
      B="$exe"
    fi
    echo "== 執行檔：${B#/src/}"

    # 1. 雙弧
    info=$($T-lipo -info "$B")
    echo "-- $info"
    for a in x86_64 arm64; do
      case "$info" in *" $a"*) ;; *) echo "缺少架構 $a" >&2; exit 1 ;; esac
    done

    for a in arm64 x86_64; do
      $T-lipo -thin $a "$B" -output /tmp/$a
      # ⚠ 先整份收下來再比，**不要 `otool -l | grep -q`**：`grep -q` 找到就結束，
      # 上游的 otool 收到 SIGPIPE 回 141，`pipefail` 把整條管線判成失敗——
      # 於是「簽章明明在」也會噴錯。機器閒的時候 otool 早就寫完不會踩到，一忙就必中。
      load=$($T-otool -l /tmp/$a)

      # 2. arm64 的 ad-hoc 簽章
      case "$load" in
        *LC_CODE_SIGNATURE*) sig="有" ;;
        *) sig="無" ;;
      esac
      if [ "$a" = arm64 ] && [ "$sig" = 無 ]; then
        echo "arm64 沒有 LC_CODE_SIGNATURE（Apple Silicon 上會 Killed: 9）" >&2; exit 1
      fi

      # 3. 最低系統版本
      minos=$(printf "%s\n" "$load" | awk "/minos/{print \$2; exit}")
      [ -n "$minos" ] || { echo "$a 讀不到 minos" >&2; exit 1; }
      [ "$minos" = "$PW_MIN" ] || { echo "$a 的 minos=$minos，與宣稱的 $PW_MIN 不符" >&2; exit 1; }
      echo "-- $a：LC_CODE_SIGNATURE $sig、minos $minos"

      # 4. 動態相依
      libs=$($T-otool -L /tmp/$a)
      bad=$(printf "%s\n" "$libs" | tail -n +2 | awk "{print \$1}" \
            | grep -vE "^(/usr/lib/|/System/Library/)" || true)
      if [ -n "$bad" ]; then
        echo "$a 連到系統目錄以外的動態庫：" >&2; printf "%s\n" "$bad" >&2; exit 1
      fi
      echo "-- $a 的相依（$(printf "%s\n" "$libs" | tail -n +2 | wc -l) 項）全在 /usr/lib 與 /System/Library"
    done

    # 5. 內容證據：這三個字串對應這份程式一定會有的三段程式碼。
    #    Application Support ＝ macOS 的存檔落點（apps/psychicwar/paths.go 的 darwin 分支），
    #    cjk24.golemfnt ＝ 中文字型子集，Resources ＝ .app 的資料搜尋路徑。
    #    找不到就表示編到的不是這份原始碼，或那段程式碼沒被連進去。
    for s in "Application Support" "cjk24.golemfnt" "Resources"; do
      n=$(strings -a "$B" | grep -c -F "$s" || true)
      [ "$n" -gt 0 ] || { echo "binary 裡找不到 \"$s\"" >&2; exit 1; }
      echo "-- 內容證據 \"$s\"：$n 處"
    done

    echo "五道全過（**結構驗收，不是功能驗收**；Linux 上執行不了 macOS binary）"
  '
