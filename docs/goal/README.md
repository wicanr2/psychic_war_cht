# 專案目標

## 終點

**一般玩家拿自己的 DOS 版《銀河超能力戰記》，能從開頭玩到結局，所有遊戲內文字都是繁體中文。**
同時提供原版沒有的輔助功能（說明、切換語言、即時存檔、地圖、作弊），並能在 Linux、Windows、macOS 上執行。

達成方式不是重寫遊戲，是讓原版執行檔跑在 dosgolem 上，由轉譯層攔截畫字、換成中文。
這條路能不能縮短中文化工期，是本專案要回答的問題。所以每一期的驗收都要附數字，不能只說「可以用」。

## 三個不變的判準

1. **原版是唯一的真相。** 畫面、顏色、音樂、文字覆蓋率都要拿原版或交叉 oracle（DOSBox-X）比過；
   只看自己的內部訊號（攔截器有觸發、測試綠燈）不算完成。
2. **不散布原版素材。** 發行物與版控裡只有自製的程式、譯文、字型、替換圖；原版由玩家自備，啟動時驗 SHA-256。
3. **通用的歸 golem，專屬的留這裡。** 判準是「換一款遊戲之後還成立嗎」。

## 分期

| 期 | 目標 | 文件 | issue |
|---|---|---|---|
| M0 | 探勘與基線 | [M0](M0-baseline.md) | #1–#3 |
| M1 | 原版在 golem 上完整可跑 | [M1](M1-golem-runs-original.md) | #4–#8 |
| M2 | 可遊玩前端 | [M2](M2-playable-frontend.md) | #9–#14 |
| M3 | 文字攔截與文本抽取 | [M3](M3-text-capture.md) | #15–#19 |
| M4 | 中文繪製與全文翻譯 | [M4](M4-chinese-rendering.md) | #20–#27 |
| M5 | 輔助功能 | [M5](M5-assist-features.md) | #28–#33 |
| M6 | 主題、打包、授權與發行 | [M6](M6-release.md) | #34–#36 |

## 輪次（給 `/goal` 用）

分期（M0–M6）是完成定義；輪次是每次 `/goal` 要做到哪裡，可以跨分期。

| 輪 | 文件 | 範圍 |
|---|---|---|
| 1 | [M0](M0-baseline.md) | 探勘與基線 |
| 2 | [M1](M1-golem-runs-original.md) §本輪完成條件 | 亂數、色盤、PC 喇叭、OPL2 雛形、重播到第一場戰鬥 |
| 3 | [R3](R3-full-run-and-pacing.md) | CPU 速度定案、觀測位址、重播推進到離開起始區域、OPL2 事件層 |
| 4 | [R4](R4-playable-frontend-and-text.md) | 可遊玩前端雛形（750 cycles、視窗、鍵盤、音訊）、印字常式、文字格式、第 3 輪開放項 |

依賴關係：M0 → M1 → M2；M3 可以在 M1 之後與 M2 並行；M4 需要 M2 的前端與 M3 的文本；M5 大多需要 M4 的疊字層；M6 最後。
逐條的前置關係以 issue 的「前置」欄與 [`docs/worklist.json`](../worklist.json) 為準。

## 進度看哪裡

- **未完成項的權威**：`docs/worklist.json`，`tools/py.sh tools/worklist.py verify` 逐條檢查。
- **討論與指派**：GitHub issue／milestone。
- 本目錄只寫目標與驗收，**不記進度**——進度寫在這裡會過期。

## 量測起點

`PW.EXE` 已在 dosgolem 上實跑到主畫面，證據與數字見 [`docs/re/001`](../re/001-pw-exe-first-look.md)。
