# 021 — 發行包：AppImage、macOS、Windows

狀態：**READY**
日期：2026-09-19
對應 issue：#35
前置：`docs/spec/006`（前端）、`docs/spec/007`（文本檔）、`docs/spec/012`（熱鍵）；`rulebook/82`

---

## 1. 範圍

把前端（`cmd/psychicwar`）連同中文資料打包成玩家解開就能跑的東西。
**所有可交付的產物一律輸出到 `dist-all/`**（gitignore），每個平台只留最新一份
（kb `mac-app-cross-pack`「產物統一放 dist-all/」的組織慣例）。

| 產物 | 內容 |
|---|---|
| `PsychicWar-<版本>-x86_64.AppImage` | 執行檔、`text/`、`font/`、`README.md`、`LICENSE`，包成單檔 |
| `PsychicWar-<版本>-macos.zip` | `PsychicWar.app`（universal：x86_64 ＋ arm64） |
| `PsychicWar-<版本>-win64.zip` | `PsychicWar.exe` 與同樣的資料檔 |
| `psychic-war-<版本>-promo.mp4` | 推廣片（`docs/re/034`） |

**Linux 只出 AppImage**，不出 tar.gz。

### 1.1 兩種變體（使用者定案 2026-09-19）

| 變體 | 原版素材 | 用途 |
|---|---|---|
| `PsychicWar-<版本>-x86_64.AppImage`、`-macos.zip` | **不含** | 可散布。玩家自備 `PW.EXE`，用 `-orig` 指過去 |
| `PsychicWar-<版本>-with-data-x86_64.AppImage`、`-with-data-macos.zip` | **含** | **純本機自用。絕不推 git、絕不上傳。** 解開就能玩，不必給 `-orig` |

`PSYCHICWAR_WITH_DATA=1 tools/package.sh all` 才會多出 `-with-data` 那一份。

可散布版打包時做一次 **leak-scan**：拿原版目錄裡實際有哪些檔名去掃包的內容，
掃到就中止（`CLAUDE.md` [HARD]：不得散布原版素材）。判準是實際檔名，不是猜副檔名。

Windows 版已經做好（`docs/re/035`），與另外兩個平台走同一支 `tools/package.sh`。

## 2. 文本檔的原文欄位（使用者定案 2026-09-19）

`text/*.json` 的 `original` 是原版的遊戲字串。轉譯層需要它來算原文長度與對位
（`translator.go` 的 `DecodeShown(e.Original)`），拿掉就不能疊字。

定案：**發行包直接帶完整的 `text/`**，不做「首次啟動從玩家的 `PW.EXE` 抽出」。
理由是沒有 `PW.EXE` 遊戲本來就跑不起來，發行包的前提已經是玩家擁有原版。

原本「含原文的文本檔不得進公開 repo」那條硬規則，使用者於 2026-09-19 一併解除：repo 公開，原文欄位照留。

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

旗標沒給時找執行檔所在目錄的 `original/`；還是沒有就結束（結束碼 2）。

訊息**不是** `flag.Usage()`。整份旗標說明有 34 行，第一行是 `Usage of …`，
沒有一句話講「你要自己準備原版」，而這是玩家最常撞到的一件事。要印的是：
缺的是 `PW.EXE`、原版要自備、兩種指法（複製成執行檔旁的 `original`，或 `-orig` 指過去）、
例子照平台給（Windows 給 `PsychicWar.exe -orig D:\…`）。旗標清單留給 `-h`。

### 3.4 致命錯誤要讓玩家看得到（issue #45）

Windows 版以 `-H windowsgui` 連結，行程沒有主控台：`log.Fatal` 的訊息**一個字都不會出現**，
雙擊的人只看到什麼都沒發生。同一段訊息在 Linux 只是終端機上的一行字
（`rulebook/82` 第 1 點：同一段訊息在不同 OS 是不同嚴重度）。

所以致命錯誤一律走 `psychicwar.Fatal`（`apps/psychicwar/fatal.go`），三條出口同時走：

| 出口 | 平台 | 內容 |
|---|---|---|
| stderr | 全部 | 與以前相同，從終端機或 `troubleshoot.bat` 啟動時看得到 |
| `<存檔目錄>/psychicwar-error.log` | 全部 | 時間、命令列、工作目錄、訊息本體。覆蓋寫，只留最後一次 |
| `MessageBoxW` | 只有 Windows | 訊息本體 ＋ 紀錄檔路徑。其他平台是 no-op |

- 缺資料檔的訊息要帶「怎麼修」，不能只給 `open …: no such file or directory`。
- **啟動成功就刪掉上一次的 `psychicwar-error.log`**，否則玩家會照著已經修好的問題追下去。
- `PSYCHICWAR_NO_DIALOG=1` 關掉彈窗。模態視窗會停在那裡等人按確定，無人看管的自動驗收
  會卡到 timeout 拿不到結束碼；彈窗本身另外驗（§5 第 7 項），不是靠這個變數繞過去不驗。

## 4. 建置

一律走 docker（`tools/package.sh`）。

| 目標 | 怎麼建 |
|---|---|
| AppImage | `tools/go-ebiten.sh` 的同一個 image 原生建執行檔，組 `AppDir`（`AppRun`、`.desktop`、圖示、資料），`tools/appimagetool.sh` 打包 |
| macOS | osxcross 交叉編譯（skill `osxcross-macos-cross-build`）。x86_64 與 arm64 各建一次，`lipo` 合成 universal |

Ebiten 需要 cgo：Linux 連 X11／GL，macOS 連 Cocoa／OpenGL／Metal framework。
兩邊都不能用 `CGO_ENABLED=0`。

中間的 staging 放 `workplace/pkg-stage/`，壓完就刪，不留在 `dist-all/`。

版本字串由 `git describe --tags --always --dirty` 取，以 `-ldflags -X` 打進執行檔，
`-version` 旗標印出來。

## 5. 驗收

`rulebook/82` 的硬規則是「驗**實際打包產物**在**它自己的執行環境**」，所以每一項都解開產物再跑，
不驗 `workplace/bin/` 的建置輸出。

1. **AppImage（可散布版）**：解到一個空目錄，`cd` 到**別的**目錄執行（cwd 不是解開處），
   `-quit-after` 跑滿並存一張截圖，與 repo 內建置的執行檔逐像素差 0。
2. **存檔落點**：同一次執行後，`$XDG_DATA_HOME/psychicwar`（測試時指到暫存目錄）底下要出現即時存檔；
   解開處的目錄不得被寫入（比對執行前後的檔案列表）。
3. **`-with-data` 變體**：在**沒有掛任何原版目錄**的容器裡跑，不給 `-orig` 要能啟動
   （靠 `OrigDir()` 找到包內的 `original/`）。反向對照：同一個容器跑可散布版要印 §3.3 的訊息、結束碼 2。
4. **反向對照**：把發行包裡的 `font/` 改名，執行要**明確報錯**指出缺字型，不是靜默跑出沒有中文的畫面。
5. **macOS**：`lipo -archs` 要列出 `x86_64 arm64`；`.app` 的結構（`Contents/MacOS`、`Contents/Resources`、
   `Info.plist`）齊全。**沒有 Mac 可以實跑，這一項只驗產物結構，不驗行為**，紀錄要照實寫。
6. **錯誤紀錄**（§3.4）：缺原版、缺字型各跑一次，`<存檔目錄>/psychicwar-error.log` 要寫得出來、
   內容看得懂。**反向對照**：正常啟動一次，結束後那個檔案不存在（上一次留下的也要被清掉）。
7. **彈窗**（Windows）：不給原版跑一次，`PSYCHICWAR_NO_DIALOG` 不設。判準是程式停在模態視窗上
   沒有自己結束、視窗樹裡有標題正確的視窗，並存一張截圖
   （`tools/windows-verify.sh <zip> --dialog`）。
8. **彈窗的反向對照**：同一個情境設 `PSYCHICWAR_NO_DIALOG=1`，程式要立刻結束、結束碼 2。

## 6. 不做

- 簽章與公證（macOS Gatekeeper 會擋，README 要寫怎麼繞過）。
- 自動更新、安裝程式。
