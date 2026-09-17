# M1：原版在 golem 上完整可跑

## 目標

**原版在 dosgolem 上從開頭跑到結局，畫面、顏色、音樂都與原版一致，而且同一組輸入永遠得到同一個結果。**
這是轉譯層的地基：底下的機器錯了，上面疊的中文再漂亮也是錯的。

## 起點（M0 留下的東西）

| 產物 | 位置 | 用法 |
|---|---|---|
| 解壓後的執行檔 | `workplace/unpacked/PW_UNP.EXE`（`tools/unexepack.py`） | 所有反組譯與位址一律用它；`PW.EXE` 是 EXEPACK 壓縮檔 |
| IDA 資料庫與普查 | `workplace/ida/PW_UNP.EXE.i64`、`docs/re/002` | 419 個函式、93 處 `INT`；遊戲本體在 `10000–1BCD5` |
| 狀態檔檢查點 | `tools/states.sh [--check]`、`docs/re/003` §2 | 19 秒到迷宮；實驗一律從檢查點展開，不從開機跑 |
| DOSBox-X 參照 | `tools/dosboxx-ref.sh`、`tools/frame_compare.py`、`docs/re/003` §3 | 版面已逐像素一致；色號 → RGB 對照表缺色號 `2`、`A` |
| 已知差異 | `docs/re/003` §2.2 | 防拷問哪位盟友隨按鍵時機改變；dosgolem 與 DOSBox-X 目前問的不同 |

## 完成定義

| 項目 | 驗收 | issue |
|---|---|---|
| 決定性與亂數 | 亂數常式位址、種子來源、每次呼叫的消費點寫進 `docs/re/`；dosgolem 同一組輸入跑兩次，戰鬥結果逐項相同；能說明為什麼防拷題目隨按鍵時機改變 | #7 |
| EGA 色盤 | mode 0Dh 的色號經屬性控制器對到 EGA 64 色；四個檢查點加上至少一個含色號 `2`、`A` 的畫面，dosgolem 輸出的 **RGB** 與 DOSBox-X 逐像素一致；dosgolem 端有測試 | #4 |
| PC 喇叭 | `.IBM` 格式寫進 `docs/re/`；標題音樂 WAV 與 DOSBox-X 錄音比對，音高與節奏一致（比對方法先寫成 spec） | #6 |
| OPL2 合成 | `-adlib` 路徑的標題音樂可合成成 WAV，與 DOSBox-X（`oplmode=opl2`）錄音比對一致 | #5 |
| 全程可跑 | 一份可重播的全程輸入（檢查點串）跑到結局，未實作服務為空，結局畫面存證 | #8 |

## 量測指標

- 檢查點畫面與 DOSBox-X 的 **RGB** 不一致像素數：**0**（防拷畫面在亂數對齊前除外，差異必須只在盟友名稱列）。
- 全程重播中的未實作服務數：**0**。
- 同一輸入兩次執行的畫面雜湊與記憶體差異：**0**。
- 音樂比對：音符起點與音高的偏差門檻由 #5／#6 的 spec 定案，本文件不預設數字。

## 執行順序

依賴決定順序，不是難易：

1. **#7 決定性與亂數**：純 RE，不改 golem。先做，因為 #8 的重播與之後的即時存檔都靠它，
   而且它能解釋防拷題目差異，讓 #4 的比對少一個例外。線索：`seg003` 的 5 處 `INT 1Ah`（沒執行過）、
   防拷題目隨按鍵指令數改變（推定亂數在等待按鍵的迴圈裡推進，假說）。
2. **#4 EGA 色盤**：先補一個含色號 `2`、`A` 的參照畫面（戰鬥或地圖畫面，延伸 `tools/states.sh`
   與 `tools/dosboxx-ref.sh`），再寫 dosgolem spec、改機器層。
3. **#6 PC 喇叭**：預設路徑，先做；`.IBM` 格式是 RE，WAV 比對需要 DOSBox-X 錄音。
4. **#5 OPL2 合成**：新元件，工程量最大；依據是 DOSBox-X 的 OPL 實作（`~/cht/DOSBox-X-MCP-Debugger/dosbox-src/`）。
5. **#8 全程可跑**：最後做，而且可能跨好幾輪。先定重播檔格式（檢查點串＋每段按鍵），
   一段一段往結局推進，每段都進 `tools/states.expected`。

## 本輪 `/goal` 的完成條件

M1 全部完成可能需要不只一輪。**本輪完成**＝以下全部成立：

- #7、#4、#6 達到上表驗收；
- #5 至少完成 spec（READY）與合成器骨架，能把 `-opl-log` 轉成 WAV，比對結果寫進 `docs/re/`（通過或不通過都要寫）；
- #8 定好重播檔格式，並把檢查點串從迷宮至少推進到**第一場戰鬥結束**與**一次存檔／讀檔**；
  沒走完的路段列在 issue #8 的留言草稿（見下方收尾）；
- `tools/py.sh tools/worklist.py verify` 沒有「可能已完成」的條目；做完的條目已移到 `done`。

## 工作邊界

- **SDD**：改 dosgolem 之前先在 `worktrees/dosgolem/docs/spec/` 寫 spec 並標 READY；本 repo 的格式解讀寫 `docs/spec/`。
- **dosgolem 的修改**只在 `worktrees/dosgolem` 的分支（`psychic-war/m1-<主題>`）上做，
  **不 push、不開 PR**；上游合併前先回報使用者。本 repo 暫時引用 worktree 裡的版本，狀態檔與期望雜湊記下 commit。
- dosgolem 的 CPU 驗收判準是「全部通過」；改機器層後跑 dosgolem 自己的測試，不能只跑本專案的檢查點。
- 全部走 docker；只清理自己建立的 container，禁止任何 prune／`rmi`。
- 原版素材只在 `workplace/`；dosgolem 的測試缺檔就 skip。
- 防拷相關：依 `~/.claude/knowledge-base/retro/ida-pro-9.4.md` 的老遊戲保存例外，可以讀判定邏輯；
  本輪只做 RE，不做略過（那是 M5 #28）。

## 收尾

- 每項完成：證據寫進 `docs/re/` 或 `docs/spec/`，`docs/worklist.json` 移到 `done`，`render`，本機 commit。
- **push、關 issue、在 issue 留言都先問使用者**；可以先把要留的內容整理在回報裡。
- 回報時附數字：不一致像素、未實作服務數、音樂比對結果、重播推進到哪一段。

## 前置

M0（#1–#3，已完成）。
