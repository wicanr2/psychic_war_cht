# 026：AdLib 路徑的 60 秒欠載（追量）

狀態：量測紀錄（2026-09-18）。對應 issue #13（即時音訊輸出）、#5（OPL2）。承接 `docs/re/016` §2.4、`docs/re/019` §6。

## 1. 結論

| 數字 | 值 |
|---|---:|
| 牆上時間 | 65 秒 |
| 欠載總數 | 102 |
| 其中開場第 1 秒 | 34 |
| 第 2–50 秒 | **0** |
| 第 51–54 秒 | 68（同一時間主機 load average 由 5.95 升到 16.44） |
| 第 55–65 秒 | 0 |
| 放棄追趕 | 677 ms（其中 153 ms 在第 1 秒） |
| CPU 佔用 | 1.80 核 |

**AdLib 路徑可以連續 50 秒不斷音**（第 2–50 秒欠載 0）。第 51–54 秒的欠載與主機負載尖峰同時發生，
不是前端本身的問題——但**這一點不能只用相關性斷定**，要在空閒主機重量才算乾淨數字。
第 1 秒的 34 次與 `docs/re/016` §2.2 記的開場（第一次繪圖編譯 shader）一致。

## 2. 為什麼這次量得出東西

`docs/re/019` §6 記的重開條件是「load average 連續 5 分鐘 < 2」。這次開跑時 load 5.95（14 核），
不符合那個條件，但**負載下的「通過」仍然是證據，負載下的「不通過」才不是**：
CPU 被搶只會讓欠載變多，不會讓它變少。所以「連續 50 秒 0 次」這個數字成立；
第 51–54 秒那一段則不下結論。

規則：**在不理想的條件下量，先想清楚這個條件會讓結果偏哪一邊**。偏向「更難通過」時，通過就算數。

## 3. 還沒做的

| 項目 | 狀態 | 重開條件 |
|---|---|---|
| 乾淨的 60 秒數字（欠載 0、無負載干擾） | 未做 | load average 連續 5 分鐘 < 2 時重跑本文件 §4 的指令 |
| #9：DOSBox-X 多場戰鬥分布補到 ≥ 5 場 | 未做 | 同上；DOSBox-X 以牆上時間跑，負載高時場次不可比 |

## 4. 重現

```
tools/go-ebiten.sh build -o /src/workplace/bin/psychicwar ./cmd/psychicwar
uptime   # 記下負載
PSYCHICWAR_SH="workplace/bin/psychicwar -orig /orig/psychic-war -adlib -audio null \
  -scratch workplace/adlib60/saves -stats workplace/adlib60/stats.jsonl -quit-after 65s" tools/go-ebiten.sh
uptime
```

每秒一行的 `stats.jsonl` 裡 `underruns` 是累計值，逐秒相減才看得出欠載發生在哪一段。
