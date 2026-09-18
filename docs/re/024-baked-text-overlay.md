# 024：圖檔內嵌文字的清冊與中文疊字

狀態：量測紀錄（2026-09-18）。對應 issue #18（清冊）、#25（替換）。
規格 `docs/spec/010`（`.PBL` 格式）、`docs/spec/011`（疊字）；dosgolem 規格 `203-baked-text-watchers`。

## 1. 結論

| 項目 | 結果 |
|---|---|
| 清冊 | 537 張圖全部判讀；**含文字 50 張、143 則**（`text/baked-inventory.json`） |
| 含文字的檔案 | `MAP.PBL` 17、`ROOM1.PBL` 15、`ROOM0.PBL` 8、`OPEN.PBL` 4、`SCREEN.PBL` 3、`MENU.PBL`／`LOGO.PBL`／`KGDLOGO.PBL` 各 1 |
| 本輪疊上中文 | 操作面板 4 則、狀態欄 2 則（`text/baked.json`） |
| 逐像素（`pwstep`） | 6 塊全部差 0 |
| 反向對照 | 拿掉 `ADVANCE` 的譯文，該塊與原版英文放大 3 倍差 0，其餘 5 塊仍是中文、差 0 |
| 前端（Xvfb） | 6 塊全部差 0 |
| 四個檢查點的英文圖字 | `07-first-play`、`08-encounter`、`09-battle-won`、`16-sivad`：文字類 0，只剩標題美術字（`docs/spec/011` §6 決定保留） |

## 2. 觸發方式

畫在圖裡的文字沒有「原版印字」那個時刻可以攔，而且同一張圖有三條繪圖路徑（實跑觀測）：

| 路徑 | 常式 | 本輪觀測到的圖 |
|---|---|---|
| 直接畫到畫面（帶位置 `CH`、`CL`） | `sub_18A7C` → `sub_18A98` | `screen.pbl` 5 張（(0,0)–(0,160)）、`room0.pbl` 的房間圖（(4,124)） |
| 解到緩衝區，之後由別的常式貼 | `sub_18B76` | `ally.pbl`、`beam.pbl`、`enemy*.pbl`、`fight.pbl` |
| 只載入不解壓 | `sub_189BF` | `menu.pbl`（面板重畫）、`open.pbl` |

所以觸發點改成**畫面內容**：登記「這塊區域應該長這樣」，比對到就蓋中文（dosgolem `xlate.Watcher`）。
好處是一份資料同時涵蓋三條路徑——面板像素同時來自 `SCREEN.PBL` 與 `MENU.PBL`（兩張圖那一塊完全相同），
一組 watcher 就夠，不必知道這一幀是誰畫的。

## 3. 六塊的資料（`text/baked.json`）

| key | 原文 | 譯文 | 比對區塊（圖內） | 中文位置（畫面） |
|---|---|---|---|---|
| `SCREEN.PBL:0:advance` | `ADVANCE` | 前進 | #0 (180,8,48,10) | (180,8) 6 格 |
| `SCREEN.PBL:0:look-aside` | `LOOK ASIDE` | 轉向 | #0 (178,32,52,8) | (180,32) 6 格 |
| `SCREEN.PBL:1:turn` | `TURN` | 向後轉 | #1 (186,14,40,10) | (184,56) 5 格 |
| `SCREEN.PBL:1:operation` | `ESC OPERATION` | 選單（`ESC` 是按鍵名，保留） | #1 (178,28,52,10) | (180,68) 6 格 |
| `SCREEN.PBL:3:place` | `PLACE` | 地點 | #3 (76,4,38,10) | (76,124) 4 格 |
| `SCREEN.PBL:3:direction` | `DIRECTION` | 方向 | #3 (82,12,36,10) | (84,132) 4 格 |

⚠ 從狀態檔起跑的截圖上，方位值會是英文（例 `North`）：那一則是**狀態存檔之前**印的，印字掛鉤沒有機會攔到。
正常遊玩時它是中文（`docs/re/022` 的試玩截圖）。看驗收截圖時不要把它當成漏譯。

## 4. 走過的坑（寫成規則）

- **比對區塊不能框到遊戲自己會重畫的地方。** 狀態欄第一版把 `PLACE` 左邊的框線與 `DIRECTION` 右邊的方位值框進來，
  比對永遠差 10 個像素、疊字一次都沒出現。**先拿實跑畫面跟解出的圖逐像素比一次，看差在哪幾格，再決定框多大。**
- **字型子集要收資料檔的字。** `baked.json` 的 schema 和文本檔不同，烘製腳本原本只收 `psychic-war-text/1`，
  新加的「前進」「向後轉」不會進子集，畫面上會是缺字。schema 一多就要回頭確認每個吃 `text/*.json` 的工具。
- 面板的 `ESC` 是鍵名不是英文單字，翻掉反而看不懂要按哪個鍵——**鍵名、代號保留原文**（與文本檔的「保留原文」規則一致）。

## 5. 清冊的判讀方式與限制

- 一個 Sonnet 子代理逐張看解出來的 PNG（角色與怪物類用整檔總覽判定，其餘 103 張逐張看），
  產出 537 筆，主迴圈再對 `tools/pbl.py list` 的尺寸、張數與抽樣內容核對過。
- `uncertain` 7 則：字被畫面邊緣截斷或字型太小（`MAP.PBL` 的 `KANSO`、`CHIOO`、`ROOM0.PBL` 的 `AT011`、`COMPUTE`、`INFORM` 等）。
  要翻譯到那幾則時再放大確認。
- 清冊只記「圖上有什麼字」，不含畫面座標；座標在需要疊中文時才量（§3 的六塊就是這樣做的）。
- **限制**：判讀是視覺的，沒有 OCR 或字模比對當交叉驗證；`MAP.PBL` 的道具說明頁字數多，翻譯前要再核一次原文。

## 6. 還沒疊中文的（下一輪）

| 來源 | 張數 | 內容 |
|---|---:|---|
| `MAP.PBL` #1–#17 | 17 | 拾獲地圖（`TOP SECRET MAP xx-x-xx`）與道具說明頁（`ARTICLE nnn OF nnn`…），字最多 |
| `ROOM0/ROOM1` | 23 | 房間招牌：`DANGER`、`GUARD`、`ELEVATOR`、`DATA`、`GATE OPEN`、`ESCAPE`、`COMPUTER`、`IMPERIAL CO. LTD.` |
| `OPEN.PBL` #7–#10 | 4 | 開場字幕條（`STANDARD 47600`、`SAMAR CITY STATION`、`SPACE JUMP IS READY`…） |
| `LOGO.PBL`、`KGDLOGO.PBL`、`SCREEN.PBL` 的標題字 | 3 | 美術字與商標，`docs/spec/011` §6 決定保留原樣 |

## 7. 重現

```
tools/py.sh tools/pbl.py dump workplace/original/psychic-war/SCREEN.PBL workplace/pbl
tools/baked_run.sh                                   # 逐像素
PSYCHICWAR_WITHOUT=SCREEN.PBL:0:advance tools/baked_run.sh   # 反向對照
tools/frontend-baked-check.sh                        # 前端 Xvfb
```
