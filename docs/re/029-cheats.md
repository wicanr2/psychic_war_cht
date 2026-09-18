# 029：作弊（補血、補能量、削弱敵人）

狀態：量測紀錄（2026-09-18）。對應 issue #33。規格 `docs/spec/014`；位址來源 `docs/re/011`、`docs/re/009` §3。

## 1. 結論

`-cheat` 打開後，前端多兩個熱鍵：**F5** 把 HP 與能量補到各自的上限、**F6** 把敵人 HP 設成 1。
一律寫記憶體（`oracle.SetWord`），不改原版檔案，位址表在 `apps/psychicwar/cheat.go`。

| 驗收（`docs/spec/014` §4） | 結果 |
|---|---|
| 單元測試（補到上限的換算、上限 0 或不合理時不寫、敵人 HP 的合理範圍） | 通過 |
| 實跑 `12-battle2`（HP 23/40、能量 1/30，不在戰鬥）＋`-cheat`：F5 → F10 存檔 → probe 讀 | HP **40/40**、能量 **30/30** |
| 同上不給 `-cheat`（反向對照） | HP **23/40**、能量 **1/30**，完全沒變 |
| 實跑 `08-encounter`（戰鬥中，敵人 HP 38）＋`-cheat`：F6 | 敵人 HP **1** |
| 同上不給 `-cheat`（反向對照） | 敵人 HP **38** |
| F6 在非戰鬥畫面（敵人 HP 字組是 0） | 不寫，顯示「現在不是戰鬥」 |

## 2. 做法上的兩個判斷

- **預設關閉**：作弊鍵平常不存在，避免正常遊玩時誤按。旗標 `-cheat`。
- **F6 要有安全條件**：不在戰鬥時 `0x509C` 是上一場留下來的值（常常是 0），直接寫 1 會把不相干的記憶體改掉。
  條件是「目前值在 1–999」；這同時也是「現在是不是戰鬥」的判準（與 `docs/re/028` 的教訓同一類：
  **殘留的資料不能當狀態**，但值域檢查可以擋掉明顯不合理的情況）。

## 3. 重現

```
tools/go-ebiten.sh build -o /src/workplace/bin/psychicwar ./cmd/psychicwar
tools/go-ebiten.sh build -o /src/workplace/bin/probe github.com/wicanr2/dosgolem/cmd/probe
tools/frontend-cheat-check.sh states/12-battle2.state   # 血量不滿的狀態
tools/frontend-cheat-check.sh                            # 預設 08-encounter（戰鬥中）
```
