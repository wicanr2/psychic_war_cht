# 033：沒有 Mac，在 Linux 上交叉編出 macOS 版

狀態：量測紀錄（2026-09-19）。對應 issue #35。規格 `docs/spec/021`（READY）§4、§5 第 5 項；
方法 skill `osxcross-macos-cross-build`。

## 1. 結論

**`dist-all/PsychicWar-<版本>-macos.zip` 可以產出，靜態驗收五道全過。**
⚠ **這不是「macOS 版完成」**——Linux 上執行不了 macOS binary，這一輪**一次都沒有實際執行過**
這支程式。結構過關只代表「不會因為格式、架構、簽章、相依而開不起來」，**不代表功能正常**。

| 項目 | 結果 |
|---|---|
| 工具鏈 | `psychicwar-osxcross`（`crazymax/osxcross:15.5-debian` ＋ Ubuntu 24.04 ＋ 專案鎖定的 go1.24.13）|
| 目標三元組 | `arm64-apple-darwin24.5`、`x86_64-apple-darwin24.5`（`OSXCROSS_TARGET` 由 `osxcross-conf` 讀）|
| SDK／連結器 | MacOSX15.5.sdk、`OSXCROSS_LINKER_VERSION=711` |
| 最低系統版本 | `minos 11.0`（`sdk 15.5`）|
| universal 執行檔 | 16,065,874 bytes，`x86_64 arm64` |
| arm64 ad-hoc 簽章 | **有** `LC_CODE_SIGNATURE`（x86_64 沒有，正常）|
| 動態相依 | 11 項，全在 `/usr/lib/` 與 `/System/Library/` |
| 內容證據 | `Application Support` 2 處、`cjk24.golemfnt` 4 處、`Resources` 10 處 |
| 發行包 | `dist-all/PsychicWar-<版本>-macos.zip`，6,301,494 bytes（`-with-data` 版 6,541,839） |
| 建置時間 | 整支腳本冷 Go 快取 84 秒、暖快取 7.5 秒；工具鏈 image 另外 116 秒（只有第一次，要網路）。`--cpus 4`，14 核機器 load ≈ 7 |
| 重現性 | 連續兩次建置的 universal 執行檔**逐位元組相同**（`-trimpath`）|
| **實機驗收** | **沒有做**（沒有 Mac）|

## 2. 產物

```
PsychicWar.app/
  Contents/Info.plist
  Contents/MacOS/psychicwar          ← universal（x86_64 ＋ arm64）
  Contents/Resources/text/           ← 56 個 JSON（譯文與原文欄位）
  Contents/Resources/font/           ← cjk16.golemfnt、cjk24.golemfnt
  Contents/Resources/psychicwar.icns
  Contents/Resources/README.md、LICENSE
```

原版素材不在裡面（`CLAUDE.md` [HARD]）。玩家把含 `PW.EXE` 的目錄放進
`Contents/Resources/original`，或用 `-orig` 指過去——兩條路都已經在
`apps/psychicwar/paths.go` 的 `OrigDir()` 裡，這次不必改程式。
資料檔同理走 `../Resources/<名>`（`docs/spec/021` §3.1 第 2 條），存檔走
`~/Library/Application Support/PsychicWar`（§3.2）。

`Info.plist` 的 `CFBundleVersion` 放完整的 `git describe` 字串；
`CFBundleShortVersionString` 慣例是 `x.y.z`，還沒有標籤時取不到數字就填 `0.0.0`。

## 3. 怎麼做的

三支新檔：

| 檔案 | 做什麼 |
|---|---|
| `tools/docker/osxcross.Dockerfile` | 工具鏈 image |
| `tools/macos-pack.sh <版本>` | 兩弧各編一次 → `lipo` → 組 `.app` → 驗收 → `zip` |
| `tools/macos-verify.sh <執行檔或 .app>` | 五道靜態驗收 |

`tools/package.sh` 的 `do_macos()` 本來就呼叫 `tools/macos-pack.sh "$VER"`，介面對得上，沒有改。

### 3.1 image 為什麼要另外一個

`psychicwar-go-ebiten` 的底是 `golang:1.24-bookworm`（Debian 12，glibc 2.36），
而 `crazymax/osxcross:15.5-debian` 裡的執行檔要 glibc 2.38——直接疊上去 `osxcross-conf`
就報 `version GLIBC_2.38 not found`。所以改成 Ubuntu 24.04 起底，
再把**同一份** Go 工具鏈從 `psychicwar-go-ebiten` 整個 `COPY` 過來：
版本仍然鎖在 go1.24.13，與 Linux 版走的是同一支編譯器。

SDK 的授權只允許在 Apple 硬體上使用，**image 只留本機，不上傳也不散布**；
發行包裡不含 SDK 的任何內容。

### 3.2 為什麼一定要 cgo

Ebiten 的 macOS 後端走 Cocoa／OpenGL／Metal。`CGO_ENABLED=0` 編不過，實測輸出：

```
internal/graphicsdriver/opengl/graphics_macos.go:26:15: undefined: glfw.Window
internal/graphicsdriver/metal/view_macos.go:57:14: v.initDisplayLink undefined
```

所以 `CC` 要指到 osxcross 的 clang wrapper。Go 的交叉編譯不是 autoconf 那一套，
但 skill 的三條照樣成立：**每弧各編一次最後 `lipo`**、**前綴問 `osxcross-conf` 不寫死**、
**arm64 要有 `LC_CODE_SIGNATURE`**。

建置本體：

```sh
export CGO_ENABLED=1 GOOS=darwin MACOSX_DEPLOYMENT_TARGET=11.0
for arch in arm64 amd64; do
  pre=…-apple-$OSXCROSS_TARGET
  env GOARCH=$arch CC=$pre-clang CXX=$pre-clang++ \
      CGO_CFLAGS=-mmacosx-version-min=11.0 CGO_LDFLAGS=-mmacosx-version-min=11.0 \
    go build -trimpath -ldflags "-s -w -X main.version=$VER" -o /tmp/psychicwar-$arch ./cmd/psychicwar
done
x86_64-apple-$OSXCROSS_TARGET-lipo -create /tmp/psychicwar-{arm64,amd64} -output …/MacOS/psychicwar
```

`GOPROXY=off`、`--network none`：模組全部來自 `workplace/gomodcache`，darwin 版不需要新模組。

### 3.3 圖示

`.icns` 自己組（header `icns` ＋ 總長度，接著每張 type(4)＋長度(4)＋PNG），
每一張都是 PNG 內嵌：

| 型別 | 邊長 | 大小 |
|---|---:|---:|
| `icp6` | 64 | 367 bytes |
| `ic07` | 128 | 521 bytes |
| `ic08` | 256 | 1,015 bytes |
| `ic09` | 512 | 3,045 bytes |
| `ic10` | 1024 | 10,583 bytes |

image 裡有 `png2icns`（icnsutils），但**沒有用**：實測它把 `ic07`／`ic08`／`ic09`
寫成 raw ARGB（它自己的輸出訊息就是 `Using icns type 'ic09' (ARGB)`），
只有 48 以下走 `is32`／`il32` ＋ 遮罩。**哪一種在現行 macOS 上畫得出來，這裡沒有辦法驗**——
選 PNG 內嵌是因為那是 Apple 自家 `iconutil` 的產出形式（二手認知，未實機確認）。
圖示畫不出來不影響程式能不能跑，實機一看就知道，優先度低。

16 與 32 這兩個尺寸沒有做：圖示是拿 24×24 的字模畫「銀河」兩字，
`tools/appicon.py` 的放大倍率在那兩個尺寸會退到 1，字會被裁掉。小尺寸交給 macOS 自己縮。

## 4. 驗收：驗到了什麼

`rulebook/82` 的硬規則是「驗**實際打包產物**」，所以下面兩份都跑：
`workplace/pkg-stage/macos/PsychicWar.app`（`macos-pack.sh` 內建，不過就不產 zip）與
**解開 zip 之後**的 `workplace/macos-check/PsychicWar.app`。兩份輸出相同。

| 道 | 檢查 | 結果 |
|---|---|---|
| 1 | `lipo -info` 雙弧 | `x86_64 arm64` |
| 2 | arm64 的 `LC_CODE_SIGNATURE` | 有（x86_64 無，正常）|
| 3 | `minos` 與宣稱一致 | 兩弧都是 `11.0`，`sdk 15.5` |
| 4 | 動態相依只在系統目錄 | 兩弧各 11 項，全數落在 `/usr/lib/`、`/System/Library/` |
| 5 | 內容證據 | `Application Support` 2、`cjk24.golemfnt` 4、`Resources` 10 |

第 4 道的完整清單（arm64）：`libresolv.9`、`libobjc.A`、`libSystem.B`、
`Cocoa`、`AppKit`、`Foundation`、`CoreFoundation`、`CoreGraphics`、`CoreServices`、
`CoreVideo`、`IOKit`。

**`Metal.framework` 不在清單裡是對的**：Ebiten v2.9.9 的 Metal 後端用
`purego.Dlopen("/System/Library/Frameworks/Metal.framework/Metal", …)` 在執行期載入
（`internal/graphicsdriver/metal/mtl/mtl_darwin.go:572`），不是連結期相依。
`CoreVideo` 在清單裡則對應 `initDisplayLink`。

第 5 道的三個字串各自對應一段非有不可的程式碼：`Application Support` ＝
`paths.go` 的 darwin 存檔落點、`cjk24.golemfnt` ＝ 中文字型子集、`Resources` ＝ `.app`
的資料搜尋路徑。反向樣本 `SDL2` 是 0 處——證明這個檢查不是「什麼都找得到」。

### 4.1 反向對照（證明驗收不是空轉）

| 餵給它 | 期望 | 實際 |
|---|---|---|
| `lipo -thin arm64` 拆出來的單弧執行檔 | 擋下 | 「缺少架構 x86_64」，結束碼 1 |
| universal 執行檔，但宣稱 `minos 12.0` | 擋下 | 「arm64 的 minos=11.0，與宣稱的 12.0 不符」，結束碼 1 |

### 4.2 沒有驗到的

**這一輪沒有任何一次執行這支程式。** 以下全部未知：

- 能不能開起來、畫面對不對、中文疊字在 macOS 上長什麼樣、音訊能不能出聲、鍵盤對應、節拍。
- Gatekeeper 的實際行為。bundle **沒有簽章**（`_CodeSignature` 要 `codesign`，Linux 上做不出來），
  也**沒有公證**（要上傳 Apple，一定要 Mac）。skill §6 的定調是「不簽勝過壞簽」：
  壞簽直接被拒絕，未簽只是首次開啟要右鍵 →「打開」。這句要寫進 README（#36）。
- 圖示在 Finder 與 Dock 裡的實際外觀。`.icns` 的結構是照格式組出來的，沒有在 macOS 上看過。
- Apple Silicon 與 Intel 兩種機器上的實跑差異。
- zip 的執行位元只在 Linux 的 `unzip` 上確認過（解開後仍是 `-rwxr-xr-x`），
  macOS 的 Archive Utility 沒試過。

要收掉這一段，唯一的辦法是一台 Mac（或 macOS CI）。

## 5. 踩到什麼（寫成規則）

- **`grep -q` 不要接在 `otool` 後面。** `grep -q` 找到就結束，上游的 `otool` 收到 SIGPIPE
  回 141，`pipefail` 把整條管線判成失敗——「簽章明明在」也會噴錯。機器閒的時候 `otool`
  早就寫完不會踩到，一忙就必中。**先整份收進變數再用 `case` 比。**
  （這條是從 `~/cht/kol/tools/verify_macos.sh` 的註解學來的，本輪照著寫，沒有再踩。）
- **`otool -L` 對 fat binary 會為每個架構印一行檔名標頭**，把它當成相依項的話檢查會永遠失敗、
  而且指著執行檔自己。要先 `lipo -thin` 拆單弧再查。
- **驗收腳本要認兩種 bundle 佈局**：`Contents/MacOS/<name>` 可能被換成包裝腳本、
  真正的執行檔改名 `<name>.bin`，`lipo` 對腳本會報 `can not figure out the architecture type`。
- **工具前綴帶 SDK 次版號**（15.5 → `darwin24.5`），寫死就會在別台或換 SDK 時整批找不到指令，
  而 clang 只轉述成「找不到指令」。一律讀 `OSXCROSS_TARGET`。
- **image 裡 `/osxcross/lib` 要進 `ldconfig`**，否則 `ld64` 起不來（`libxar.so.1`），
  症狀同樣是「找不到指令」。

## 6. 重現

```sh
tools/macos-pack.sh                       # 或 tools/package.sh macos
unzip -q dist-all/PsychicWar-<版本>-macos.zip -d workplace/macos-check
tools/macos-verify.sh workplace/macos-check/PsychicWar.app
```

反向對照：

```sh
PSYCHICWAR_MACOS_MIN=12.0 tools/macos-verify.sh workplace/macos-check/PsychicWar.app   # 要擋下
```

旋鈕：`PSYCHICWAR_MACOS_MIN`（預設 11.0）、`PSYCHICWAR_MAC_CPUS`（預設 4）、
`PSYCHICWAR_MAC_IMAGE`、`PSYCHICWAR_MAC_TIMEOUT`。
第一次跑會 build image（**那一步要網路**，之後一律 `--network none`）。
