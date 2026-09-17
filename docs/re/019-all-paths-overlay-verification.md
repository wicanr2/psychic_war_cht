# 019：所有印字路徑的疊字驗收、覆蓋率與前端實跑

狀態：量測紀錄（2026-09-17）。對應 issue #20、#21、#24。規格 `docs/spec/009`；dosgolem 規格 `201-step-actions`、`202-translation-overlay`
（分支 `psychic-war/r5-text`，未推上游）。

## 1. 結論

| 項目 | 結果 |
|---|---|
| 單元測試（本 repo `apps/...`、dosgolem 全套 `go test -count=1 ./...`） | 通過 |
| 逐像素（`pwstep`，8 個情境、13 行，涵蓋 A1／A2／A3／B） | 13 行全部差 0 |
| 反向對照（每個情境拿掉第一個目標那一則的譯文） | 那一則的每一行都與原版英文放大 3 倍差 0 |
| 前端 Xvfb（A1 `wall`、B `match`） | 5 行全部差 0 |
| 覆蓋率（重播 19 段，推進到 Zellwal） | 觸發 55 則（要翻 54）、不在文本檔 0 |
| 前端從開機走 01–09 段並打贏第一場（開中文疊字） | 疊字 27 筆；`missing-translation`、`too-long`、`missing-glyph` 都是 0 |
| 前端截圖 12 張的英文殘留 | 文字路徑 0；剩下的英文全部畫在圖檔上（#18） |

## 2. 逐像素情境

期望值不用轉譯層自己的紀錄：`dosgolem cmd/step` 以同一狀態、同一動作跑出原版畫面，`tools/overlay_check.py` 在原版畫面找到那一行英文
（字模比對，`FONT.BIN` 或 `PW_UNP.EXE` 的小字型指標表），取背景與前景色號，依 spec 009 §2 的幾何與透明格畫出期望區塊，再跟 `pwstep` 的截圖逐像素比。

| 情境 | 路徑 | key（行） | 起始狀態 → 動作 | 像素數 | 中文 | 反向對照 |
|---|---|---|---|---:|---:|---:|
| `wall` | A1 | `I_MENUH.BIN:0530`（0、1） | `overlay/wall-before` → `tap:Up,wait:4000` | 9,216＋8,640 | 0 | 0 |
| `direction` | A2 | `CODEH.BIN:02D5` | `states/07-first-play` → `tap:Left,wait:2500` | 4,608 | 0 | 0 |
| `place` | A2 | `I_MAP03.BIN:0238` | `overlay/zell-arrive-before` → `wait:4000` | 4,032 | 0 | 0 |
| `enemy` | A3 | `I_ENMY00.BIN:0052` | `overlay/encounter-before` → `wait:4000` | 4,608 | 0 | 0 |
| `surrender` | A3 | `PW.EXE:cs:4AFB` | `overlay/surrender-before` → `hold:Space:2000` | 8,640 | 0 | 0 |
| `match` | B | `PW.EXE:cs:65E5`（0–2） | `states/01-title` → `tap:Space,wait:4000` | 7,560 × 3 | 0 | 0 |
| `protection` | B | `PW.EXE:cs:6707`、`677F` | `states/02-match` → `tap:Return,wait:4000` | 3,024 × 2 | 0 | 0 |
| `name` | B（透明格） | `PW.EXE:cs:8AEF`（0、2） | `states/05-select` → `tap:Return,wait:3000,type:kai,wait:1500` | 7,560 × 2 | 0 | 0 |

- `protection` 的反向對照只拿掉 `6707`；`677F` 同一張圖仍是中文、差 0，確認拿掉一則不影響同畫面其他疊字。
- `name` 的第 2 行是輸入框：方括號與空白格位和原文相同，框內 `kai` 是原版畫的字（透明格），期望區塊在透明格直接取原版像素。
  原版位置搜尋在這一行要跳過原文空白格，否則框內的輸入字讓整行比不到。

## 3. 前端 Xvfb

`tools/frontend-overlay-check.sh <情境> <鍵>`：參照同樣由 `cmd/step` 產生；前端（Ebiten，scale 3）從同一狀態啟動，xdotool 送鍵，
等轉譯紀錄出現全部目標 key 後截圖（強制 PNG24，`import` 預設會存成調色盤 PNG）。

| 情境 | 行 | 差 |
|---|---|---:|
| `wall Up` | `0530` 第 0、1 行 | 0、0 |
| `match space` | `65E5` 第 0–2 行 | 0、0、0 |

規則：**畫面上一塊區域被原版逐步搬動（訊息框捲動）時，搬到一半的畫面不能拿來判斷疊字失效。** `pwstep` 每 16.7 ms 機器時間定一次色，
前端每幀跑的機器時間隨主機負載變動，捲動一步可能跨 3 幀以上；所以 spec 009 §4.6 在 `6273`（`sub_16783` 進入點）到 `6276`（retn）之間
凍結整個落在訊息框內的疊字（dosgolem `xlate.Layer.Frozen`）。捲動只搬、只凍結「整個」在框內的疊字：小字型的名字輸入區塊（x 100–220）左上角落在框內，但不屬於訊息框。

## 4. 覆蓋率（重播 19 段）

`tools/text_coverage.sh` → `tools/py.sh tools/text_coverage.py workplace/coverage/title-to-first-save`：

| 數字 | 值 |
|---|---:|
| 要翻 | 1,313 |
| 已翻 | 1,313 |
| 實跑觸發 | 55（要翻 54） |
| 觸發了卻不在文本檔 | **0** |

觸發來源：`I_MENUH` 27、`PW.EXE` 9、`I_MENU00` 6、`CODEH` 4、`I_ENMY00` 3、`I_MAP00` 2、`I_MAP01`／`I_MAP03`／`I_MENU01`／`I_MENU03` 各 1。
第 19 段 `19-zellwal` 抵達 Zellwal（區域 3），`I_MAP03`、`I_MENU03` 是本輪新增的來源。執行期產生、不列入文本的：輸入回顯與選擇游標（`B055`）、
腳本緩衝區的數字、`cs:2EE3`／`cs:2F3F` 只有顏色碼的字串。

## 5. 前端從開機實跑（開中文疊字）

`tools/frontend-playthrough.sh`（Xvfb、xdotool、`-audio null`、轉譯紀錄 `workplace/fe/play/text.jsonl`）：

- 檢查點 02–07 全部等到（檢查點是原版英文畫面，開疊字時容許值至少 3,000 像素，只用來抓送鍵時機；實際差 127–1,745）。
- 按住空白鍵 35 秒打贏 Shulosu（敵人消失、回到迷宮、玩家 HP 13）。F1 被攔截、轉向與 Esc 選單畫面都有變化。
- 轉譯紀錄：`stamp` 27、`drop` 18（`changed` 16：換畫面；`overlap` 2：方位換字、防拷名字蓋過區塊第 0 行）；三種問題事件 0。
- 量測時主機 load average 約 14–18（14 核，另有十幾個其他專案的容器），`underruns` 63、`dropped_ms` 350；節拍與音訊以 `docs/re/016` 為準。

截圖 12 張（`steps/00`–`05`、`08`、`09`、`15`–`17` 與 `final.png`：標題、配對說明、防拷、許可通過、轉場、主選單後的名字輸入、輸入名字、首次進迷宮、遭遇、戰後、轉向、Esc 選單）：

| 看到的英文 | 來源 | 處理 |
|---|---|---|
| `PSYCHIC WAR COSMIC SOLDIER`、`KYODAI`、`KGD SOFT`、`SF ROLEPLAYING` | 圖檔（`SCREEN.PBL` 等） | #18 |
| 操作面板 `ADVANCE`、`LOOK ASIDE`、`TURN`、`ESC OPERATION` | 圖檔 | #18 |
| 狀態欄標籤 `PLACE`、`DIRECTION` | 圖檔（值「北」「西」是疊字） | #18 |

文字路徑上的英文殘留 0。排版觀察：Esc 選單每行 8 像素高，放大後 24 點中文字上下緊貼，可讀但擁擠；原版行距就是字高，改善要動版面（M6 主題）。

## 6. 開放項

| 項目 | 結論 | 重開條件 |
|---|---|---|
| AdLib 路徑 60 秒欠載 | 本輪沒有重量。2026-09-17 22:19 load average 18.22（14 核），另有十幾個其他專案的容器在跑，前端拿不到 2 核，量到的欠載不能代表播放品質 | 主機 load average 連續 5 分鐘 < 2 時，`-adlib -audio null -stats` 從開機錄 60 秒，記 load 與 `cpu_ms` |
| #9 DOSBox-X 多場戰鬥分布 | 維持 3 場有效（平均 17.75 秒，`docs/re/010` §4.6）。DOSBox-X 以牆上時間跑，負載高時結果不可比 | 同上的空閒條件，走位改成等檢查點後補到 ≥ 5 場 |
| 倚天字模散布授權 | 選項與字形對照在 `docs/re/020`，等使用者決定 | 使用者選定 |

## 7. 重現

```
tools/go-ebiten.sh build -o /src/workplace/bin/pwstep ./cmd/pwstep
tools/go-ebiten.sh build -o /src/workplace/bin/psychicwar ./cmd/psychicwar
tools/go-ebiten.sh build -o /src/workplace/bin/step github.com/wicanr2/dosgolem/cmd/step
tools/overlay_run.sh && PSYCHICWAR_WITHOUT=1 tools/overlay_run.sh
tools/frontend-overlay-check.sh wall Up && tools/frontend-overlay-check.sh match space
tools/text_coverage.sh && tools/py.sh tools/text_coverage.py workplace/coverage/title-to-first-save
tools/frontend-playthrough.sh
```
