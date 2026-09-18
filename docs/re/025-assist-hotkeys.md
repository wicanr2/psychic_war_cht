# 025：輔助熱鍵 F1、F2、F10／F11

狀態：量測紀錄（2026-09-18）。對應 issue #29、#30、#31。規格 `docs/spec/012`。

## 1. 結論

| 驗收（`docs/spec/012` §5） | 結果 |
|---|---|
| 單元測試：說明頁排版、即時存檔的版本比對、文字繪製 | 通過 |
| F2 切到英文：畫面與 `cmd/step` 跑同一狀態的原版畫面（放大 3 倍） | 不同像素 **0**／576,000 |
| F2 切回中文：與切換前 | **0** |
| F1 說明頁：與 `text/help.json` 算出的期望值 | **0**（22 行） |
| F1 關閉：與開啟前 | **0** |
| F10 → 移動一步 → F11：畫面與存檔時 | **0** |
| 同上：觀測變數（`lin:16966` 起 52 bytes） | 相同 |
| 反向對照：把 `quick.json` 的 `exe_sha256` 改掉再按 F11 | 拒絕讀檔，畫面與按之前相同（**0**） |

## 2. 做法

- **F2** 只是「不畫」：轉譯層照常攔截、照常建疊字、照常 `Frame`，Draw 略過疊字層。
  所以切回中文時畫面立刻正確，不必重跑；英文模式下畫面就是原版像素（上表第 2 列是這件事的證據）。
- **F1** 說明頁的內容在 `text/help.json`（schema `psychic-war-help/1`），程式與驗收工具讀同一份，
  期望值才不是「前端自己說的」。排版：一格 8×scale 像素、**半形字佔半格**、整頁置中、固定黑底白字。
  排不下（超過 38 格寬或 22 行）在載入時就報錯。
- **F10／F11** 存的是 dosgolem 狀態檔 ＋ 疊字層快照 ＋ `quick.json`（schema、golem 狀態格式版本、
  `PW.EXE` 的 SHA-256、`text/` 的 SHA-256、語言、時間）。
  版本或執行檔不符就拒絕；譯文不同只警告。讀檔後重新登記圖檔疊字的 watcher。

## 3. 走過的坑（寫成規則）

- **載入狀態檔會把時鐘倒回**，而音訊是用「上次到現在的 cycles 差」算取樣數的：
  無號數相減變成天文數字，`make([]int16, n)` 直接 panic，前端在按下 F11 的那一刻整個掛掉。
  修在 dosgolem（`oracle.Audio.Render` 偵測倒回就重新對時、那一段不出聲，規格 `199`），
  因為**任何會載入狀態的前端都會踩到**。
  規則：**凡是「上次到現在」的差值，都要先想「時間會不會倒退」**——存讀檔、跨幀節流、重連都會。
- **讀檔之後不能直接換掉疊字層物件**：`Layer` 上掛著 `OnDrop`、`Frozen` 回呼，換掉就悄悄失去捲動凍結與紀錄。
  改成清內容、保留回呼（`Translator.ResetForLoad`）。
- **前端的牆上時間也要跟著重新對齊**：讀檔後 `startCyc` 不重設的話，節拍器會以為落後了幾十秒，
  拚命補跑，畫面會快轉。
- 說明頁的英文（`Esc`、`F10`、`Launch Pad`）用中文字型畫會一個字佔一整格，看起來很散；
  **半形字佔半格**之後才像正常排版。全形括號夾半形英文會擠在一起，改用半形括號。

## 4. 重現

```
tools/go-ebiten.sh build -o /src/workplace/bin/psychicwar ./cmd/psychicwar
tools/go-ebiten.sh build -o /src/workplace/bin/step github.com/wicanr2/dosgolem/cmd/step
tools/go-ebiten.sh build -o /src/workplace/bin/probe github.com/wicanr2/dosgolem/cmd/probe
tools/frontend-hotkeys-check.sh          # 產出 workplace/hotkeys/result.txt
```
