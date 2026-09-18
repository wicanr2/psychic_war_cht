#!/usr/bin/env bash
# macOS 發行包（docs/spec/021 §1、§4）：兩弧各編一次 → lipo 合成 universal →
# 組 `PsychicWar.app` → `dist/PsychicWar-<版本>-macos.zip`。
#
#   tools/macos-pack.sh [版本]        # 版本預設 git describe --tags --always --dirty
#   tools/package.sh macos            # 同上，發行包的統一入口
#
# 走 `tools/docker/osxcross.Dockerfile` 的 `psychicwar-osxcross`（第一次自動 build，
# 那一步要網路；之後一律 `--network none`）。Linux 版走的是 `tools/go-ebiten.sh` 的
# 同一份 Go 工具鏈，兩邊的 `-ldflags` 要一起改（`tools/package.sh` 的 `build_linux`）。
#
# 為什麼要 osxcross 而不是 `CGO_ENABLED=0`：Ebiten 的 macOS 後端走 Cocoa／OpenGL／
# Metal，**一定要 cgo**（實測 `CGO_ENABLED=0` 會噴 `undefined: glfw.Window`、
# `v.initDisplayLink undefined`）。
#
# ⚠ **這支跑完只做得到靜態驗收**（`tools/macos-verify.sh`，docs/spec/021 §5 第 5 項）。
# Linux 上執行不了 macOS binary，**結構過關不等於功能正常**，真機驗收沒有做。
#
# ⚠ bundle 不簽章：`_CodeSignature` 要 `codesign`，Linux 上做不出來。執行檔本身的
# ad-hoc 簽章由 ld64 在連結 arm64 時補上（verify 的第 2 道在驗這個）。「未簽」勝過
# 「壞簽」——壞簽是直接被拒絕，未簽只是首次開啟要右鍵 →「打開」。
#
# 發行包不含原版素材（`CLAUDE.md` [HARD]）：玩家把含 `PW.EXE` 的目錄放進
# `PsychicWar.app/Contents/Resources/original`，或用 `-orig` 指過去。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

VER="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
IMAGE="${PSYCHICWAR_MAC_IMAGE:-psychicwar-osxcross}"
MIN="${PSYCHICWAR_MACOS_MIN:-11.0}"
CPUS="${PSYCHICWAR_MAC_CPUS:-4}"
APP_NAME="PsychicWar"
EXE_NAME="psychicwar"

DIST="dist"
STAGE="$DIST/stage/macos"
APP="$STAGE/$APP_NAME.app"
ZIP="$DIST/$APP_NAME-$VER-macos.zip"

# --- 1. 工具鏈 image（缺了才 build，build 要網路）--------------------------------
if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
  if ! docker image inspect psychicwar-go-ebiten >/dev/null 2>&1; then
    echo "[macos-pack] 先 build psychicwar-go-ebiten（osxcross image 要從它搬 Go 工具鏈）" >&2
    docker build -t psychicwar-go-ebiten -f "$ROOT/tools/docker/go-ebiten.Dockerfile" "$ROOT/tools/docker"
  fi
  echo "[macos-pack] 第一次使用，先 build $IMAGE" >&2
  docker build -t "$IMAGE" -f "$ROOT/tools/docker/osxcross.Dockerfile" "$ROOT/tools/docker"
fi
mkdir -p "$ROOT/workplace/gocache" "$ROOT/workplace/gomodcache"

# --- 2. bundle 骨架與資料 --------------------------------------------------------
rm -rf "$STAGE"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources" "$STAGE/icon"
# text/ 與 font/ 是自製素材（譯文、字型子集），放 Contents/Resources，
# 由 apps/psychicwar/paths.go 的 `../Resources/<名>` 那一條找到（docs/spec/021 §3.1）。
mkdir -p "$APP/Contents/Resources/text" "$APP/Contents/Resources/font"
cp text/*.json "$APP/Contents/Resources/text/"
cp font/*.golemfnt "$APP/Contents/Resources/font/"
cp README.md LICENSE "$APP/Contents/Resources/"

# --- 3. 圖示 ---------------------------------------------------------------------
# 只做 64 以上：圖示是拿 24×24 的字模畫「銀河」兩字，更小的尺寸字會被裁掉
# （`tools/appicon.py` 的 scale 會退到 1）。小尺寸交給 macOS 自己縮。
ICON_SIZES=(64 128 256 512 1024)
for s in "${ICON_SIZES[@]}"; do
  tools/py.sh tools/appicon.py "$STAGE/icon/icon_$s.png" "$s" >/dev/null
done
# .icns 自己組：header `icns` ＋ 總長度，接著每張 type(4)＋長度(4，含這 8 bytes)＋PNG。
# 用 PNG 內嵌的型別碼（icp6／ic07／ic08／ic09／ic10 ＝ 64／128／256／512／1024），
# 這是現行 `iconutil` 產出的形式。png2icns（icnsutils）也在 image 裡，但它把
# ic07／ic08／ic09 寫成 raw ARGB，那是舊格式，所以不用它。
tools/py.sh - "$APP/Contents/Resources/$EXE_NAME.icns" "$STAGE/icon" <<'PY' >/dev/null
import pathlib, struct, sys
out, src = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
types = {64: b"icp6", 128: b"ic07", 256: b"ic08", 512: b"ic09", 1024: b"ic10"}
body = b""
for size, tag in sorted(types.items()):
    png = (src / ("icon_%d.png" % size)).read_bytes()
    body += tag + struct.pack(">I", len(png) + 8) + png
out.write_bytes(b"icns" + struct.pack(">I", len(body) + 8) + body)
print(out, len(body) + 8)
PY

# --- 4. 兩弧各編一次 → lipo ------------------------------------------------------
# 前綴帶 SDK 次版號（SDK 15.5 → darwin24.5），從 `osxcross-conf` 讀，不寫死。
timeout "${PSYCHICWAR_MAC_TIMEOUT:-30m}" docker run --rm --network none \
  --memory 6g --cpus "$CPUS" --pids-limit 512 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -e HOME=/tmp \
  -e "PW_VER=$VER" -e "PW_MIN=$MIN" -e "PW_OUT=$APP/Contents/MacOS/$EXE_NAME" \
  -v "$ROOT:/src" \
  -v "$ROOT/workplace/gocache:/gocache" -v "$ROOT/workplace/gomodcache:/gomodcache" \
  --tmpfs "/tmp:exec,uid=$(id -u),gid=$(id -g),size=2g" \
  -w /src "$IMAGE" \
  bash -c '
    set -euo pipefail
    eval "$(osxcross-conf)"
    export GOCACHE=/gocache GOMODCACHE=/gomodcache GOPATH=/tmp/gopath
    export GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local GOWORK=off GOFLAGS=-mod=mod
    export CGO_ENABLED=1 GOOS=darwin
    export MACOSX_DEPLOYMENT_TARGET=$PW_MIN
    for arch in arm64 amd64; do
      case $arch in
        arm64) pre=arm64-apple-$OSXCROSS_TARGET ;;
        amd64) pre=x86_64-apple-$OSXCROSS_TARGET ;;
      esac
      echo "[macos-pack] $arch（$pre）"
      env GOARCH=$arch CC=$pre-clang CXX=$pre-clang++ \
          CGO_CFLAGS="-mmacosx-version-min=$PW_MIN" \
          CGO_LDFLAGS="-mmacosx-version-min=$PW_MIN" \
        go build -trimpath -ldflags "-s -w -X main.version=$PW_VER" \
          -o /tmp/psychicwar-$arch ./cmd/psychicwar
    done
    x86_64-apple-$OSXCROSS_TARGET-lipo -create /tmp/psychicwar-arm64 /tmp/psychicwar-amd64 \
      -output "/src/$PW_OUT"
    x86_64-apple-$OSXCROSS_TARGET-lipo -info "/src/$PW_OUT"
  '
chmod +x "$APP/Contents/MacOS/$EXE_NAME"

# --- 5. Info.plist ---------------------------------------------------------------
# CFBundleShortVersionString 慣例是 x.y.z；版本字串是 git describe，沒有標籤時
# 只是一串雜湊，所以短版本取標籤裡的數字，取不到就 0.0.0，完整字串放 CFBundleVersion。
SHORT="$(printf '%s' "$VER" | sed -n 's/^v\{0,1\}\([0-9][0-9.]*\).*/\1/p')"
[[ -n "$SHORT" ]] || SHORT="0.0.0"
cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key><string>zh_TW</string>
	<key>CFBundleDisplayName</key><string>銀河超能力戰記</string>
	<key>CFBundleExecutable</key><string>$EXE_NAME</string>
	<key>CFBundleIconFile</key><string>$EXE_NAME</string>
	<key>CFBundleIdentifier</key><string>io.github.wicanr2.psychicwar</string>
	<key>CFBundleInfoDictionaryVersion</key><string>6.0</string>
	<key>CFBundleName</key><string>$APP_NAME</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleShortVersionString</key><string>$SHORT</string>
	<key>CFBundleVersion</key><string>$VER</string>
	<key>LSApplicationCategoryType</key><string>public.app-category.role-playing-games</string>
	<key>LSMinimumSystemVersion</key><string>$MIN</string>
	<key>NSHighResolutionCapable</key><true/>
	<key>NSSupportsAutomaticGraphicsSwitching</key><true/>
</dict>
</plist>
PLIST

# --- 6. 靜態驗收（過不了就不要產出 zip）------------------------------------------
tools/macos-verify.sh "$APP"

# --- 7. zip ----------------------------------------------------------------------
# zip 會保留 unix 權限位元，解開之後執行位元還在（macOS 的 Archive Utility 也認）。
rm -f "$ZIP"
( cd "$STAGE" && zip -qry "$ROOT/$ZIP" "$APP_NAME.app" )
echo "$ZIP"
