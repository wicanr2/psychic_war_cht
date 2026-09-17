# Worklist

> 由 `tools/worklist.py render` 從 `docs/worklist.json` 產生，**不要手改**。
> 條目做完就從 JSON 移走或改 verify，不是在這裡打勾。

## M0：探勘與基線：證據、狀態檔、參照畫面

| issue | id | 標題 | label | 前置 | 完成訊號 |
|---|---|---|---|---|---|
| #1 | `ida-baseline` | 建立 PW.EXE／LOGO.EXE 的 IDA 資料庫並做函式普查 | re | — | absent |
| #2 | `state-checkpoints` | 固定的狀態檔檢查點：標題、防拷、輸入名字、第一個可操作畫面 | verify | — | absent |
| #3 | `dosbox-reference-frames` | DOSBox-X 參照畫面：同一個畫面的正確顏色與版面 | verify, graphics | — | manual |

## M1：原版在 golem 上完整可跑：畫面、音樂、決定性

| issue | id | 標題 | label | 前置 | 完成訊號 |
|---|---|---|---|---|---|
| #4 | `ega-palette-output` | EGA mode 0Dh 的畫面輸出沒有套用遊戲設定的色盤 | golem-upstream, graphics | #3 | manual |
| #5 | `opl2-synth` | OPL2（AdLib）合成：`.MID` 音樂路徑目前只有暫存器紀錄 | golem-upstream, audio | — | manual |
| #6 | `pc-speaker-audio` | PC 喇叭音樂路徑（`.IBM`）驗證 | audio, verify | — | absent |
| #7 | `determinism-rng` | 亂數來源與重播決定性 | re, verify | — | absent |
| #8 | `full-run-no-gaps` | 從開頭跑到結局，沒有未實作的服務 | verify | #2 | manual |

## M2：可遊玩前端：視窗、鍵盤、滑鼠、音訊、節拍

| issue | id | 標題 | label | 前置 | 完成訊號 |
|---|---|---|---|---|---|
| #9 | `realtime-pacing` | 牆上時間節拍：讓 72 Hz 的遊戲以原速執行 | golem-upstream, frontend | — | absent |
| #10 | `frontend-window` | 可遊玩前端：視窗顯示 320×200 EGA 畫面（整數倍放大） | frontend, graphics | #4 | manual |
| #11 | `keyboard-input` | 鍵盤輸入：前端按鍵轉成 IRQ1 掃描碼 | input, frontend | #10 | manual |
| #12 | `mouse-input` | 滑鼠：確認原版是否支援，並在前端提供點選操作 | input, re | — | absent |
| #13 | `audio-output` | 即時音訊輸出（OPL2 與 PC 喇叭） | audio, frontend | #5、#9 | manual |
| #14 | `input-record-replay` | 輸入錄放：以指令數記錄按鍵，可重播重現 | golem-upstream, verify | #7 | manual |

## M3：文字攔截與文本抽取

| issue | id | 標題 | label | 前置 | 完成訊號 |
|---|---|---|---|---|---|
| #15 | `print-routine` | 找出印字常式：FONT.BIN 字模與固定寬度文字框 | re, text | #1 | absent |
| #16 | `text-formats` | 文字資料格式：I_MENU*.BIN、I_ENMY*.BIN、CODE*.BIN、PW.EXE 字串表 | re, text | — | absent |
| #17 | `text-extract` | 文本抽取工具：從玩家自備的原版產生文本檔 | text | #16 | absent |
| #18 | `baked-text-graphics` | 圖檔內嵌文字：PBL 格式與含文字的圖清冊 | re, graphics, text | — | absent |
| #19 | `runtime-text-coverage` | 文字覆蓋率量測：實跑觸發但不在文本檔的訊息數 | verify, text | #15、#17 | manual |

## M4：中文繪製與全文翻譯

| issue | id | 標題 | label | 前置 | 完成訊號 |
|---|---|---|---|---|---|
| #20 | `hires-overlay-canvas` | 放大畫布＋疊字層：原版畫面放大後在上面畫中文 | golem-upstream, graphics, text | #15、#10 | manual |
| #21 | `cjk-font` | CJK 點陣字：依譯文烘製子集 | text, graphics | — | manual |
| #22 | `layout-rules` | 固定寬度文字框的中文排版規則 | text, graphics | #20 | absent |
| #23 | `glossary` | 譯名表：人名、地名、超能力名稱 | text | — | absent |
| #24 | `translation-pass` | 全文翻譯 | text | #17、#23、#22 | manual |
| #25 | `baked-text-replacement` | 圖檔內嵌文字的中文替換圖層 | graphics, text | #18、#20 | manual |
| #26 | `name-entry` | 輸入名字：原版只收 ASCII，中文版怎麼處理 | text, input | #15 | manual |
| #27 | `playtest-cht` | 中文版正常玩家路徑試玩 | verify, text | #24、#25 | manual |

## M5：輔助功能：F1／F2／F3／F10、防拷、作弊

| issue | id | 標題 | label | 前置 | 完成訊號 |
|---|---|---|---|---|---|
| #28 | `copy-protection` | 防拷：翻譯、F1 查表、略過選項 | assist, re, text | #1 | absent |
| #29 | `f1-help` | F1 說明頁 | assist, frontend | #20 | manual |
| #30 | `f2-language` | F2 切換語言（中文／英文原文） | assist, text | #20 | manual |
| #31 | `f10-quicksave` | F10 即時存檔／讀檔 | assist, golem-upstream | #7 | manual |
| #32 | `f3-map` | F3 自動地圖 | assist, re | — | absent |
| #33 | `cheats` | 作弊功能 | assist, re | — | manual |

## M6：主題、打包、授權與發行

| issue | id | 標題 | label | 前置 | 完成訊號 |
|---|---|---|---|---|---|
| #34 | `theme` | 主題替換 | graphics | #20 | manual |
| #35 | `cross-platform-package` | 跨平台打包：Linux／Windows／macOS | release | #27 | manual |
| #36 | `license-readme` | LICENSE（RRSAL-1.0）與 README | release | — | absent |
