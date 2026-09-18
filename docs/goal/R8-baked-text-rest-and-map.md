# 第 8 輪：圖檔文字剩下的部分、F3 自動地圖、防拷與作弊

## 目標

**把清冊裡剩下的 47 張圖檔文字（房間招牌、道具圖鑑、開場字幕）疊上中文，並補完 M5 的其餘輔助功能。**
做完之後，玩家在正常路徑上看到的英文只剩刻意保留的標題美術字（M4 收尾、M5 主體）。

使用者定案沿用：**dosgolem 的修改整個移植完成後一次發 PR**（分支已推 `psychic-war/r5-text`）；
**字型用倚天字形發行**；**試玩用抽測**。

## 起點（第 7 輪留下的東西）

| 產物 | 位置 |
|---|---|
| `.PBL` 格式與解碼工具 | `docs/spec/010`、`tools/pbl.py`、`apps/psychicwar/pbl` |
| 含文字圖清冊 537 筆（含文字 50 張 143 則） | `text/baked-inventory.json`、`docs/re/024` |
| 以畫面內容觸發的疊字機制 | `docs/spec/011`、dosgolem 規格 `203-baked-text-watchers` |
| 已疊：操作面板 4 塊、狀態欄 2 塊 | `text/baked.json`、`tools/baked_run.sh` |
| F1 說明、F2 中英切換、F10／F11 即時存檔 | `docs/spec/012`、`docs/re/025`、`tools/frontend-hotkeys-check.sh` |
| 覆蓋率工具 | `tools/baked_report.py` |

## 本輪完成條件

1. **房間招牌（#25）**
   - `ROOM0.PBL`／`ROOM1.PBL` 含文字的 23 張：`DANGER`、`GUARD`、`ELEVATOR`、`DATA`、`GATE OPEN`、`ESCAPE`、`COMPUTER`、`IMPERIAL CO. LTD.` 等，全部疊上中文。
   - 每張圖的畫面座標以實跑觀測或 `tools/pbl.py find` 取得，寫進 `text/baked.json`。
   - 驗收：每張至少一塊以 `tools/baked_run.sh` 逐像素差 0；抽 3 張做反向對照。
2. **道具圖鑑與拾獲地圖（#25）**
   - `MAP.PBL` 17 張（`TOP SECRET MAP xx-x-xx`、`ARTICLE nnn OF nnn`…）疊上中文。字多，先確認原文（清冊有 7 則標 `uncertain`，要放大重看）。
   - 譯文照既有流程（譯名表、行寬檢查）；`tools/baked_report.py` 的「已疊張數」達 17。
3. **開場字幕（#25）**
   - `OPEN.PBL` 4 張的字幕條疊上中文，或寫明不做的理由（開場動畫會捲動，疊字可能跟不上）。
4. **F3 自動地圖（#32）**
   - 規格先寫 READY：資料來源（`I_MAP` 或記憶體的區域與座標）、畫法、開關行為。
   - 驗收：走過的格子會標出來，玩家位置與朝向正確；開關前後記憶體相同。
5. **防拷（#28）**
   - 判定常式讀出來寫進 `docs/re/`；三種用法（照答、F1 查表、設定裡略過）各有測試。
6. **作弊（#33）**
   - 玩家 HP、能量的位址找出來；作弊一律透過記憶體寫入，位址表放 `apps/psychicwar`。
7. **抽測試玩**
   - 從第 7 輪的進度再抽兩段（含一次進房間看招牌、一次道具圖鑑），確認畫面上是中文；問題逐條處理。
   - ⚠ 第 7 輪的抽測卡在**等級**：3 級角色打不過賽瓦德的敵人（`docs/re/027`）。這一輪的抽測先用作弊（#33）
     或既有存檔把角色練起來再走，不要把時間花在重複戰死。
   - 判讀規則：**捲動中或轉場中的畫面不要拿來判斷有沒有英文**（`docs/re/027` §2.7）。
8. **開放項**
   - AdLib 60 秒欠載：主機空閒（load < 2 連續 5 分鐘）時重量一次，把第 7 輪的「50 秒無欠載、第 51–54 秒與負載尖峰同時」補成乾淨數字。
   - #9 DOSBox-X 多場戰鬥：同一條件下補到 ≥ 5 場。
9. `tools/py.sh tools/worklist.py verify` 沒有「可能已完成」；做完的條目移到 `done`；本機 commit。

## 量測指標

- 圖檔文字：已疊張數／含文字張數（目前 3／50）、每塊的不同像素數、反向對照結果。
- F3：地圖與實際位置的一致次數、開關前後的記憶體比對。
- 防拷：三種用法各自的測試結果。
- 抽測試玩：段數、步數、問題條數、卡住次數。

## 工作邊界

- SDD：改 dosgolem 前先寫規格標 READY；每次改 dosgolem 跑全套測試。
- 全部走 docker（Python 用 `tools/py.sh`，**子代理也一樣**）；只清理自己建立的 container，禁止 prune／`rmi`。
- 原版素材只放 `workplace/` 與 private repo 的 `text/`。
- **push、PR 先問**；issue 狀態與留言可以直接更新。

## 收尾

- 每項完成：證據進 `docs/re/` 或 `docs/spec/`，worklist 移到 `done` 或更新現況，`render`，本機 commit。
- 在對應 issue 留言進度；完成的關閉。
- 回報附數字；產生下一輪的 goal markdown。
