#!/usr/bin/env bash
# 統一發行入口；全部建置與組裝均在Docker。需乾淨HEAD及同名tag。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
VER="${1:?請給完整版號 v.1.0.0-YYYYMMDD}"
[[ "$VER" =~ ^v\.[0-9]+\.[0-9]+\.[0-9]+-[0-9]{8}$ ]] || exit 2
[[ -z "$(git status --porcelain --untracked-files=normal)" ]] || { echo '工作樹必須乾淨' >&2; exit 2; }
[[ "$(git describe --exact-match --tags HEAD)" = "$VER" ]] || exit 2
HEAD="$(git rev-parse HEAD)"
THEME="${PSYCHICWAR_RELEASE_THEME:?指定已驗證的本機完整HD主題}"
[[ "$THEME" = /* ]] || THEME="$ROOT/$THEME"
test -d "$THEME"; test -f "$THEME/manifest.json"
test -d "$ROOT/workplace/original"; test -f "$ROOT/workplace/original/psychic-war/PW.EXE"
test -d /home/anr2/go/pkg/mod
GO_IMAGE=psychicwar-go-ebiten:latest
MAC_IMAGE=psychicwar-osxcross:go1.24.13-15.5-r2
APP_IMAGE=hr-appimage:runtime-recovery-r1
docker image inspect "$GO_IMAGE" "$MAC_IMAGE" "$APP_IMAGE" >/dev/null
COMMON=(--rm --user "$(id -u):$(id -g)" --cpus 2 --pids-limit 512 --network none --log-opt max-size=10m --log-opt max-file=3 -e HOME=/tmp -e GOMAXPROCS=2 -e GOMODCACHE=/gomod -e GOCACHE=/src/workplace/finish-gocache-20261008 -e GOPROXY=off -e GOTOOLCHAIN=local -e GOWORK=off -e "PW_VER=$VER" -e "PW_HEAD=$HEAD" -v "$ROOT:/src" -v "$ROOT/workplace/original:/src/workplace/original:ro" -v /home/anr2/go/pkg/mod:/gomod:ro -w /src)
timeout 10m docker run "${COMMON[@]}" --memory 4g --name psychicwar-release-build "$GO_IMAGE" bash -c '
 set -euo pipefail
 stage="workplace/release-stage/$PW_VER"; test ! -e "$stage"; mkdir -p "$stage"
 go build -p 1 -trimpath -ldflags "-s -w -X main.version=$PW_VER" -o "$stage/linux" ./cmd/psychicwar
 CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -p 1 -trimpath -ldflags "-s -w -H windowsgui -X main.version=$PW_VER" -o "$stage/windows.exe" ./cmd/psychicwar
 python3 tools/appicon.py "$stage/icon.png"
 '
timeout 15m docker run "${COMMON[@]}" --memory 4g --name psychicwar-release-mac "$MAC_IMAGE" bash -c '
 set -euo pipefail
 eval "$(osxcross-conf)"
 export CGO_ENABLED=1 GOOS=darwin MACOSX_DEPLOYMENT_TARGET=11.0
 for arch in arm64 amd64; do
  case $arch in arm64) pre=arm64-apple-$OSXCROSS_TARGET;; amd64) pre=x86_64-apple-$OSXCROSS_TARGET;; esac
  GOARCH=$arch CC=$pre-clang CXX=$pre-clang++ CGO_CFLAGS=-mmacosx-version-min=11.0 CGO_LDFLAGS=-mmacosx-version-min=11.0 go build -p 1 -trimpath -ldflags "-s -w -X main.version=$PW_VER" -o "/tmp/psychicwar-$arch" ./cmd/psychicwar
 done
 x86_64-apple-$OSXCROSS_TARGET-lipo -create /tmp/psychicwar-arm64 /tmp/psychicwar-amd64 -output "workplace/release-stage/$PW_VER/macos"
 x86_64-apple-$OSXCROSS_TARGET-lipo -info "workplace/release-stage/$PW_VER/macos"
 '
timeout 10m docker run "${COMMON[@]}" --memory 3g --name psychicwar-release-stage -v "$ROOT/workplace/original:/orig:ro" -v "$THEME:/theme:ro" "$GO_IMAGE" python3 tools/release-stage.py "$VER" "$HEAD" /theme
test -d "$ROOT/workplace/release-stage/$VER"; test -d "$ROOT/dist-all/$VER"
timeout 10m docker run --rm --user "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 128 --network none --log-opt max-size=10m --log-opt max-file=3 -e "PW_VER=$VER" -v "$ROOT/workplace/release-stage/$VER:/stage:ro" -v "$ROOT/dist-all/$VER:/out" --name psychicwar-release-appimage "$APP_IMAGE" sh -c '
 set -eu
 for kind in patch full-local; do
  suffix=""; [ "$kind" = patch ] || suffix=-with-data
  mksquashfs "/stage/$kind/PsychicWar.AppDir" "/tmp/$kind.squashfs" -processors 2 -root-owned -noappend -no-progress -comp zstd -Xcompression-level 15
  target="/out/$kind/PsychicWar-$PW_VER$suffix-x86_64.AppImage"; test ! -e "$target"
  cat /opt/runtime-x86_64 "/tmp/$kind.squashfs" > "$target"; chmod +x "$target"
 done
 '
echo "封包已建立：dist-all/$VER。逐包smoke及SHA清單另由驗收工具產生。"
