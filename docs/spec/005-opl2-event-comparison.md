# 005：OPL2 事件層比對（dosgolem 暫存器紀錄 vs DOSBox-X raw OPL）

狀態：**READY**
日期：2026-09-17
對應：issue #5；前一層 `docs/spec/002`（WAV 比對，方法分辨力不足，`docs/re/007`）

## 1. 目的

回答「遊戲的 AdLib 驅動在 dosgolem 上寫進 OPL2 的暫存器序列，跟在 DOSBox-X 上一樣嗎」。
這一層不經過合成器，與 dosgolem 合成器的品質無關。

## 2. 兩邊的輸入

| 來源 | 產生方式 | 內容 |
|---|---|---|
| dosgolem | probe `-adlib -opl-log`（標題畫面、不按鍵） | 每筆：指令步數、暫存器、值（全部寫入） |
| DOSBox-X | `DX-CAPTURE /O PW.EXE`（`sbtype=sb1`、`oplmode=opl2`），遊戲以 Ctrl+Q 正常結束讓擷取收尾 | DRO v2：暫存器轉換表＋（raw 索引, 值）與延遲命令 |

## 3. DOSBox-X 擷取的過濾（`src/hardware/adlib.cpp` `Adlib::Capture`），dosgolem 端要照做

1. 暫存器不在轉換表（01、04、05、08、BD、20–35／40–55／60–75／80–95／E0–F5 的有效槽、A0–A8、B0–B8、C0–C8）就不記。
2. 寫入值與暫存器目前的值相同就不記（快取在每次寫入後更新，不論有沒有記）。
3. 擷取在「第一個音」開始，開頭先寫一份快取裡不為 0 的暫存器（B0–B8 去掉 Key-On 位元、BD 去掉打擊樂位元），
   再寫觸發的那筆。dosgolem 端在同一個時點（第一次 B0–B8 Key-On 位元為 1 或 BD 打擊樂觸發的那筆）照同樣規則產生這份快取傾印，一起比。
4. 計時器暫存器 02、03、04 在 OPL2 模式下由 `Chip::Write` 攔下，不進快取也不擷取。
5. 兩筆之間超過 30 秒就停止擷取。

## 4. 判準（執行前定案）

以 dosgolem 過濾後的序列為基準，逐筆比（暫存器, 值）：

| 指標 | 通過 |
|---|---|
| 從起點算起、兩邊都有的筆數 N（取較短者，至少 5,000 筆） | 逐筆完全一致的前綴長度 ＝ N |
| 若不一致 | 回報第一個不一致的位置、前後 5 筆，以及以 `difflib` 對齊後的不一致筆數 |
| 時間（診斷，不是判準） | DRO 累計毫秒 vs dosgolem 步數換算毫秒的線性斜率 |

反向對照：把 dosgolem 序列中任一筆的值改掉，工具必須回報不一致並指出該位置。

### 4.1 修訂（2026-09-17，**看過第一次結果之後**加的）

DOSBox-X 的擷取要讓遊戲正常結束才會寫標頭，而遊戲結束（迷宮中 Ctrl+Q）時會把 9 個聲道的 A0–A8、B0–B8 全部寫 0；
dosgolem 端沒有結束遊戲。修訂：**DOSBox-X 擷取最後 10 ms 內、至少 9 筆、全是 A0–A8／B0–B8 寫 0 的結尾不計入 N**。
原判準的結果照樣回報，兩者並列（`docs/re/012`）。

## 5. 工具

`tools/opl_events.py <dosgolem opl-log> <DOSBox-X .dro> [--json out.json]`；
`tools/dosboxx-opl.sh [秒數]` 產生 `.dro`。
