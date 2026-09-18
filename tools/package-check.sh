#!/usr/bin/env bash
# 發行包的驗收（docs/spec/021 §5）：解開產物，在**別的 cwd** 跑，比檢查點畫面。
#
#   tools/package-check.sh <產物.tar.gz|產物.AppImage>
#
# 為什麼不驗 workplace/bin 的建置輸出：rulebook/82 的硬規則是「驗實際打包產物在它自己的執行環境」。
# 路徑解析（資料在執行檔旁、存檔在使用者資料目錄）只有解開之後才驗得到。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PKG="${1:?產物路徑}"
[[ -f "$ROOT/$PKG" || -f "$PKG" ]] || { echo "找不到 $PKG" >&2; exit 2; }
[[ -d "$ROOT/workplace/original/psychic-war" ]] || { echo "缺原版 workplace/original/psychic-war，skip" >&2; exit 0; }
OUT="workplace/package-check"
mkdir -p "$ROOT/$OUT"

# 容器內：解開到 /unpack，從 /elsewhere 執行（cwd 不是解開處），存檔導到 /home 底下的 XDG 目錄。
PSYCHICWAR_TIMEOUT=20m PSYCHICWAR_SH='
set -eu
cd /src
mkdir -p /tmp/unpack /tmp/elsewhere /tmp/xdg
case "'"$PKG"'" in
  *.tar.gz) tar -C /tmp/unpack -xzf "'"$PKG"'"; APPDIR=$(ls -d /tmp/unpack/*/) ;;
  *.AppImage) cp "'"$PKG"'" /tmp/app.AppImage; chmod +x /tmp/app.AppImage
              (cd /tmp/unpack && /tmp/app.AppImage --appimage-extract >/dev/null)
              APPDIR=/tmp/unpack/squashfs-root/usr/bin/ ;;
  *) echo "不認得的產物"; exit 2 ;;
esac
EXE="$APPDIR/psychicwar"
echo "== 版本：$($EXE -version)"
# 執行前的檔案列表：解開處不得被寫入（docs/spec/021 §5 第 2 項）
find /tmp/unpack -type f | sort > /tmp/before.txt

cd /tmp/elsewhere      # ← cwd 刻意不是解開處，也不是 repo
( sleep 3; import -window root /src/'"$OUT"'/pkg.png ) &
XDG_DATA_HOME=/tmp/xdg "$EXE" -orig /orig/psychic-war -load-state /src/'"$OUT"'/start.state \
  -audio null -quit-after 6s -text-log /src/'"$OUT"'/text.jsonl > /src/'"$OUT"'/run.log 2>&1 || echo "結束碼 $?"
wait
# 對照組：repo 裡建置的執行檔，同一個狀態檔、同一組旗標（驗「打包沒有改變行為」）
( sleep 3; import -window root /src/'"$OUT"'/repo.png ) &
cd /src
XDG_DATA_HOME=/tmp/xdg workplace/bin/psychicwar -orig /orig/psychic-war -load-state '"$OUT"'/start.state \
  -audio null -quit-after 6s > /src/'"$OUT"'/repo.log 2>&1 || echo "對照組結束碼 $?"
wait
cd /tmp/elsewhere
find /tmp/unpack -type f | sort > /tmp/after.txt
diff /tmp/before.txt /tmp/after.txt > /src/'"$OUT"'/unpack-diff.txt 2>&1 || true
ls -R /tmp/xdg > /src/'"$OUT"'/xdg.txt 2>&1 || true

# 反向對照（§5 第 4 項）：把 font/ 改名，要明確報錯，不是靜默跑出沒中文的畫面
mv "$APPDIR/font" "$APPDIR/font-off"
XDG_DATA_HOME=/tmp/xdg "$EXE" -orig /orig/psychic-war -load-state /src/'"$OUT"'/start.state \
  -audio null -quit-after 3s > /src/'"$OUT"'/nofont.log 2>&1 || true
mv "$APPDIR/font-off" "$APPDIR/font"
' "$ROOT/tools/go-ebiten.sh" >/dev/null 2>&1 || true

cd "$ROOT"
echo "=== 執行紀錄"; tail -5 "$OUT/run.log" 2>/dev/null || echo "（沒有）"
echo "=== 解開處有沒有被寫入（要是空的）"; cat "$OUT/unpack-diff.txt" 2>/dev/null
echo "=== 使用者資料目錄"; cat "$OUT/xdg.txt" 2>/dev/null
echo "=== 少了字型的反向對照（要有明確錯誤）"; tail -3 "$OUT/nofont.log" 2>/dev/null
echo "=== 發行包 vs repo 建置的畫面（要差 0）"
if [[ -f "$OUT/pkg.png" && -f "$OUT/repo.png" ]]; then
  docker run --rm --network none --memory 512m --cpus 1 --pids-limit 32 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp \
    -v "$ROOT/$OUT:/o:ro" -w /o psychicwar-go-ebiten \
    sh -c 'compare -metric AE pkg.png repo.png null: 2>&1; echo' || true
else
  echo "截圖沒出來：$(ls "$OUT"/*.png 2>/dev/null | tr "\n" " ")"
fi
