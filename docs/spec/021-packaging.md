# 021 — 發行包：Linux、AppImage、macOS

狀態：**READY**
日期：2026-09-19
對應 issue：#35
前置：`docs/spec/006`（前端）、`docs/spec/007`（文本檔）、`docs/spec/012`（熱鍵）；`rulebook/82`

---

## 1. 範圍

把前端（`cmd/psychicwar`）連同中文資料打包成玩家解開就能跑的東西：

| 產物 | 內容 |
|---|---|
| `psychicwar-<版本>-linux-x86_64.tar.gz` | 執行檔、`text/`、`font/`、`README.md`、`LICENSE` |
| `PsychicWar-<版本>-x86_64.AppImage` | 同上，包成單檔 |
| `PsychicWar-<版本>-macos.zip` | `PsychicWar.app`（universal：x86_64 ＋ arm64） |

**不含原版素材**：`PW.EXE`、`.PBL`、`.BIN`、`.MID`、磁碟映像一律不進發行包。玩家自備。

Windows 不在本輪（issue #35 另外追蹤）。

## 2. 文本檔的原文欄位（使用者定案 2026-09-19）

`text/*.json` 的 `original` 是原版的遊戲字串。轉譯層需要它來算原文長度與對位
（`translator.go` 的 `DecodeShown(e.Original)`），拿掉就不能疊字。

定案：**發行包直接帶完整的 `text/`**，不做「首次啟動從玩家的 `PW.EXE` 抽出」。
理由是沒有 `PW.EXE` 遊戲本來就跑不起來，發行包的前提已經是玩家擁有原版。

`CLAUDE.md` 的硬規則「含原文的文本檔不得進公開 repo」仍然成立，管的是 repo 本身。

## 3. 路徑解析

打包產物的 cwd 不是 repo 根目錄，AppImage 與 `.app` 的內容還是唯讀的（`rulebook/82` 第 2、4 點）。

### 3.1 資料檔（`-text`、`-font`）

旗標沒給時依序找，第一個存在的就用：

1. 執行檔所在目錄下的 `text/`／`font/`
2. macOS `.app` 的 `Contents/Resources/text`／`font`（＝執行檔目錄的 `../Resources/<名>`）
3. cwd 下的 `text/`／`font/`（從 repo 根目錄直接跑的情形）

旗標有給就照給的用，找不到就照現行行為報錯，不靜默回退。

### 3.2 存檔（`-scratch`）

旗標沒給時用使用者資料目錄，不是 cwd：

| 平台 | 位置 |
|---|---|
| Linux | `$XDG_DATA_HOME/psychicwar`，沒設就 `~/.local/share/psychicwar` |
| macOS | `~/Library/Application Support/PsychicWar` |

目錄以 `MkdirAll` 建（父層可能不存在）。遊戲存檔（`<名>.DAT`）與即時存檔都寫在這裡。

### 3.3 原版目錄（`-orig`）

旗標沒給時找執行檔所在目錄的 `original/`；還是沒有就照現行行為印用法並結束（結束碼 2），
訊息要講清楚「要自備原版，把含 `PW.EXE` 的目錄用 `-orig` 指過來」。

## 4. 建置

一律走 docker（`tools/package.sh`）。

| 目標 | 怎麼建 |
|---|---|
| Linux x86_64 | `tools/go-ebiten.sh` 的同一個 image，原生建 |
| AppImage | 用 Linux 的執行檔組 `AppDir`（`AppRun`、`.desktop`、圖示、資料），`appimagetool` 打包 |
| macOS | osxcross 交叉編譯（skill `osxcross-macos-cross-build`）。x86_64 與 arm64 各建一次，`lipo` 合成 universal |

Ebiten 需要 cgo：Linux 連 X11／GL，macOS 連 Cocoa／OpenGL／Metal framework。
兩邊都不能用 `CGO_ENABLED=0`。

版本字串由 `git describe --tags --always --dirty` 取，以 `-ldflags -X` 打進執行檔，
`-version` 旗標印出來。

## 5. 驗收

`rulebook/82` 的硬規則是「驗**實際打包產物**在**它自己的執行環境**」，所以每一項都解開產物再跑，
不驗 `workplace/bin/` 的建置輸出。

1. **Linux tar.gz**：解到一個空目錄，`cd` 到**別的**目錄執行（cwd 不是解開處），
   `-quit-after` 跑滿並存一張截圖，與檢查點 `07-first-play` 逐像素差 0。
2. **存檔落點**：同一次執行後，`$XDG_DATA_HOME/psychicwar`（測試時指到暫存目錄）底下要出現即時存檔；
   解開處的目錄不得被寫入（比對執行前後的檔案列表）。
3. **AppImage**：`--appimage-extract-and-run` 在唯讀情境下跑同一項，畫面與第 1 項相同、存檔一樣落在使用者目錄。
4. **反向對照**：把發行包裡的 `font/` 改名，執行要**明確報錯**指出缺字型，不是靜默跑出沒有中文的畫面。
5. **macOS**：`lipo -archs` 要列出 `x86_64 arm64`；`.app` 的結構（`Contents/MacOS`、`Contents/Resources`、
   `Info.plist`）齊全。**沒有 Mac 可以實跑，這一項只驗產物結構，不驗行為**，紀錄要照實寫。

## 6. 不做

- Windows（issue #35）。
- 簽章與公證（macOS Gatekeeper 會擋，README 要寫怎麼繞過）。
- 自動更新、安裝程式。
