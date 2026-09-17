# 006：可遊玩前端雛形

狀態：**READY**
日期：2026-09-17
對應：issue #9（節拍）、#10（視窗）、#11（鍵盤）、#13（音訊）；速度依據 `docs/spec/004`；dosgolem 規格 `199-live-session-api`

## 1. 目的

原版 `PW.EXE` 跑在 dosgolem 上，開視窗讓人玩：畫面、鍵盤、聲音，速度對齊 AT 8 MHz。
這一版是雛形：不含中文、不含輔助功能（F1／F2／F3／F10 先攔下不送進遊戲，功能留給 M5）。

## 2. 技術選擇

| 項目 | 選擇 | 理由 |
|---|---|---|
| 語言與框架 | Go ＋ Ebiten v2.9.9 | dosgolem 是 Go；Ebiten 在 Linux／Windows／macOS 都能出視窗與聲音，本機其他專案已驗證同版本 |
| 位置 | `cmd/psychicwar/`（主程式）、`apps/psychicwar/`（按鍵對應、節拍、音訊緩衝，可單元測試） | CLAUDE.md 預定目錄 |
| module | `github.com/wicanr2/psychic_war_cht`，`replace github.com/wicanr2/dosgolem => ./worktrees/dosgolem` | 前端需要 dosgolem 本機分支的 `oracle` 即時介面（規格 `199`） |
| 建置 | docker image `psychicwar-go-ebiten`（`tools/docker/go-ebiten.Dockerfile`：golang 1.24 ＋ X11／GL／ALSA 標頭 ＋ Xvfb、xdotool、imagemagick），包裝 `tools/go-ebiten.sh` | 硬規則：建置測試走 docker |
| 支援平台 | 本輪只驗 Linux（docker＋Xvfb）；Windows／macOS 交叉編譯屬 #35 | |

## 3. 規格

### 3.1 啟動參數

`psychicwar -orig <含 PW.EXE 的目錄> [選項]`

| 參數 | 預設 | 意義 |
|---|---|---|
| `-cycles` | `750` | 每毫秒 cycles，也接受 `xt`／`at8`／`at12` |
| `-scale` | `3` | 整數倍放大（nearest） |
| `-adlib` | false | 388h 上有 OPL2（遊戲改走 `.MID`） |
| `-scratch` | `./saves` | 遊戲存檔寫到這裡（原版目錄唯讀） |
| `-load-state` | 空 | 從 probe 狀態檔開始（除錯、測試用） |
| `-audio` | `ebiten` | `ebiten`：音效卡；`null`：丟棄但照牆上時間消耗（沒有音效卡的容器用） |
| `-stats` | 空 | 每秒把量測寫一行 JSON 到這個檔（§3.5） |
| `-quit-after` | 0 | 大於 0 時，牆上時間到了就自己結束（自動驗收用） |
| `-wav` | 空 | 把每次 `Render()` 送出的取樣另存成 WAV（44,100 Hz 單聲道 16 位元），驗證聲音內容用 |

### 3.2 節拍（牆上時間對齊）

- 每次 `Update`（Ebiten 60 TPS）：`目標機器毫秒 ＝ 啟動後經過的牆上毫秒`；
  `要跑的 cycles ＝ (目標 − 已跑機器毫秒) × cycles`，單次最多 100 ms 份（落後太多時放棄追趕，並計一次 `dropped_ms`）。
- 機器毫秒由 `Cycles()` 換算，不另外累加（避免兩個時鐘漂移）。

### 3.3 畫面

- 每次 `Draw` 取 `ScreenRGB()`，寫進 320×200 的 Ebiten 影像，以整數倍 nearest 放大到視窗。視窗大小固定 `320×scale × 200×scale`。

### 3.4 鍵盤

- Ebiten 按鍵剛按下 → `KeyDown(掃描碼)`；剛放開 → `KeyUp`。typematic 由 dosgolem 產生（規格 `199` §3.3），前端不自己重複。
- 對應表（`apps/psychicwar/keymap.go`）：方向鍵與數字鍵盤 8／4／6／2 → 48／4B／4D／50；Space 39；Enter 1C；Esc 01；Backspace 0E；
  A–Z、0–9、減號 → IBM PC XT 第 1 組掃描碼；Ctrl 1D、左 Shift 2A、右 Shift 36、Alt 38（Ctrl+Q 等組合鍵照送）。
- **F1、F2、F3、F10 不送進遊戲**，記錄次數（`stats` 的 `intercepted`）。
- 限制：按下與放開落在同一次輪詢（1/60 秒）之間時收不到。自動化送鍵要按住至少 0.15 秒（`docs/re/016` §3）。

### 3.5 音訊與量測

- 每次 `Update` 跑完後 `Audio.Render()`，取樣寫進環形緩衝（44,100 Hz 單聲道 16 位元，容量 0.5 秒）。
- 音效卡端（或 `null` 端）讀緩衝；不夠時補 0 並計一次 `underruns`。`null` 端每 10 ms 依牆上時間讀走應讀的取樣數。
- 啟動後先預填 50 ms 靜音，避免第一次讀就欠載。
- `-stats` 每秒一行：`wall_ms`、`machine_ms`、`ticks`（`Ticks()`）、`underruns`、`dropped_ms`、`intercepted`、`cpu_ms`（行程 CPU 時間）。

### 3.6 分層

- `apps/psychicwar`：`Pacer`（§3.2 的純計算）、`Ring`（環形緩衝＋欠載計數）、`ScanCode(ebiten.Key) (uint8, bool)`、`Intercepted(ebiten.Key) bool`。不 import Ebiten 的部分要能在沒有 X 的情況下測試（按鍵對應用 Ebiten 的 key 常數，不需要開視窗）。
- `cmd/psychicwar`：Ebiten 的 `Game`、音效串接、參數。

## 4. 驗收

1. 單元測試（`apps/psychicwar`）：
   - `Pacer`：牆上 1,000 ms、750 cycles → 要跑 750,000；落後 500 ms 時單次只給 100 ms 份並記 `dropped_ms`。
   - `Ring`：寫 1,000 讀 1,500 → 回 1,500 筆、後 500 筆為 0、`underruns` ＝ 1；反向對照：讀 1,000 時 `underruns` ＝ 0。
   - 按鍵對應：方向鍵、數字鍵盤、Space、Enter、Esc、A、Z、0、9 的掃描碼；F1／F2／F3／F10 被攔下。
2. 畫面：`-load-state workplace/states/03-protection.state` 啟動，Xvfb 截圖依 `scale` 取樣回 320×200，與 `03-protection.rgb.png` 逐像素比，不同像素 0。
3. 節拍：`-audio null -stats` 實跑 60 秒（從 `07-first-play` 狀態），`machine_ms` 與 `wall_ms` 差 ≤ 1%；`ticks` 增量換算成 72.0 Hz ±1%（分頻 16571 → 72.00 Hz）；記錄 `cpu_ms`／`wall_ms`。
4. 音訊：同一次 60 秒 `underruns` ＝ 0（前 1 秒不計）。
5. 鍵盤：Xvfb 裡以 xdotool 送真正的 X 按鍵（`tools/frontend-playthrough.sh`），從開機走完重播 01–09 段的流程（標題 → 防拷 → 名字 → 走 6 步 → 按住空白鍵），打贏 Shulosu（敵人圖像消失、玩家 HP 不為 0）；送 F1 後 `intercepted` ≥ 1 且畫面沒有反應。
6. 人手操作：記錄一次人手操作的結果（可以輸，要有紀錄）。沒有人手時寫明未做。

## 5. 不做

- 中文、疊字層、F1／F2／F3／F10 的功能。
- 滑鼠（#12）、輸入錄放（#14）、交叉編譯（#35）。
- 視窗縮放、全螢幕、垂直同步調整。
