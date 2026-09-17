# 第 6 輪：所有文字路徑都能疊中文、全文翻譯第一版

## 目標

**把第 5 輪「一行 I_MENU 訊息」的疊字，擴到遊戲裡每一條印字路徑，並完成 1,313 則的第一版譯文。**
做完之後，重播路線上看到的所有遊戲內文字都應該是中文，缺的要能列出來（M4 主體）。

使用者定案沿用：**dosgolem 的修改在整個遊戲移植完成後一次發 PR**；本輪繼續疊在本機分支上。

## 起點（第 5 輪留下的東西）

| 產物 | 位置 |
|---|---|
| 兩套字型與全部印字呼叫端的歸類；控制碼表 | `docs/re/014` |
| 文字來源規格（I_MENU、I_ENMY、I_MAP、CODE、PW.EXE）與文本檔 1,389 則（要翻 1,313） | `docs/spec/007`、`text/`、`tools/text_extract.py` |
| 覆蓋率工具：重播 18 段觸發 52 則、不在文本檔 0 | `docs/re/017`、`tools/text_coverage.*` |
| 疊字層雛形：I_MENU 定長字串，逐像素對原版推算的期望值差 0 | `docs/spec/008`、`docs/re/018`、`apps/psychicwar/overlay`、`translate.go` |
| 字型子集烘製（倚天 24 點＋Noto 補字） | `tools/font/bake.sh`、`font/cjk24.bin` |
| dosgolem `SetRegs`、IRQ0 絕對排程 | 規格 `200-hook-register-write`、`198-dosbox-cycles`（分支 `psychic-war/r5-text`，未推） |

## 本輪完成條件

以下全部成立才算完成：

1. **其餘印字路徑的疊字（#20）**
   - `FONT.BIN` 路徑：`sub_167B2`（CODE 內嵌、地點名稱緩衝區）、`sub_167BF`（`PW.EXE` 字串、敵人名稱緩衝區）接上疊字，key 換算沿用 `docs/spec/007` §6.2。
   - 小字型路徑（`B016` 區塊、`0788` 故事、`07C1` 版權、`0BA3`／`124B` 製作群、`6504` 防拷名字）：規格先定格子大小與中文字級（6×6 字格放大 3 倍是 18×18，要決定用 16 還是 24 點、怎麼對齊），再實作。
   - 每條路徑至少一則以 `tools/frontend-overlay-check.sh` 的方法驗收（期望值由原版畫面推算，逐像素差 0），附反向對照。
   - 規格更新：`docs/spec/008` 升版或另寫 `009`，標 READY。
2. **通用部分抽回 dosgolem（規格先寫）**
   - 疊字層（排版、定色、指紋失效、捲動）與「印字常式攔截的掛鉤介面」寫成 dosgolem 規格 READY，位址由本 repo 提供；本 repo 改用 dosgolem 的套件。全套測試通過。
3. **全文翻譯第一版（#24）**
   - 1,313 則要翻的都有譯文；方法依 knowledge-base `workflows/batch-subagent-localization.md`（分批、譯名表先行、一致性檢查）。
   - 譯名表（#23）補齊人名、超能力、道具與說明書其他頁；說明書沒有的標 provisional。
   - 排版檢查工具：每則譯文依 `docs/spec/008` §3.3 排得下（「譯文過長」0 則），字型子集缺字 0。
4. **覆蓋率與實跑**
   - 重播檔推進到至少再一個區域（Zellwal 或 Rusteck），重跑 `tools/text_coverage.sh`：「不在文本檔」0。
   - 前端從開機走重播 01–09 段（`tools/frontend-playthrough.sh`），轉譯紀錄的 `missing-translation`、`too-long`、`missing-glyph` 都是 0；抽 10 張截圖看有沒有英文殘留。
5. **模擬人手操作（使用者 2026-09-17 要求）**
   - dosgolem 加「逐步操作」介面（規格先寫 READY）：載入狀態 → 以牆上時間語意按住或點按鍵、跑一段機器時間 → 存狀態、輸出放大畫面與中文疊字後的截圖、印出本段轉譯紀錄。
     每一步是一次獨立的指令，不需要常駐行程；同一組步驟重播結果逐位元組相同。
   - 用這個介面由代理像玩家一樣看畫面操作：從開機到打贏第一場戰鬥、再到出發平台選目的地，每一步記「看到什麼、決定按什麼、結果」，寫成試玩紀錄（取代 spec 006 §4 第 6 項的人手操作）。
   - 試玩中看到的英文殘留、排版問題、操作卡住的地方逐條列出。
6. **第 5 輪留下的開放項（各給結論或重開條件）**
   - AdLib 路徑 60 秒欠載：主機空閒時重量（load average 與佔用核數一起記）。
   - #9：DOSBox-X 多場戰鬥分布（若第 5 輪沒做完）。
   - 倚天字模散布授權：列出可替代的授權清楚字型與字形差異，給使用者決定。
7. `tools/py.sh tools/worklist.py verify` 沒有「可能已完成」；做完的條目移到 `done`；本機 commit。

## 量測指標

- 疊字：每條路徑驗收的不同像素數；反向對照結果。
- 翻譯：要翻則數、已翻則數、譯文過長則數、缺字數、fallback 字數。
- 覆蓋率：四個數字（新路線）。
- 前端實跑：轉譯紀錄三種問題的筆數。
- 模擬試玩：步數、看到的問題條數、卡住次數。

## 工作邊界

- SDD：改 dosgolem 前先寫規格標 READY；本 repo 的格式與方法寫 `docs/spec/`。每次改 dosgolem 跑全套測試。
- 全部走 docker（含 Python 臨時分析，用 `tools/py.sh`）；只清理自己建立的 container，禁止 prune／`rmi`。
- 原版文字只在 `workplace/` 與 private repo 的 `text/`；說明書只摘要，不轉錄。
- **push、PR 先問**；issue 狀態與留言可以直接更新。

## 收尾

- 每項完成：證據進 `docs/re/` 或 `docs/spec/`，worklist 移到 `done` 或更新現況，`render`，本機 commit。
- 在對應 issue 留言進度；完成的關閉。
- 回報附數字；產生下一輪的 goal markdown。
