# 005：EGA 色盤修正與 RGB 對拍

狀態：量測紀錄（2026-09-17）。對應 issue #4。

## 1. 結論

- 畫面全黑的成因是 dosgolem **設 mode 0Dh 時沒有載入 DAC**：遊戲只用 `INT 10h AH=10h AL=00` 設屬性暫存器、從不寫 DAC
  （EGA 卡沒有 DAC），色號經屬性暫存器對到的 DAC 全是 0。
- 修正在 dosgolem 分支 `psychic-war/m1-ega-palette`（`509a639`，**未推上游**），規格 `docs/spec/194-ega-200-line-mode-palette-defaults.md`：
  設 0Dh／0Eh 時 DAC 0–63 依 200 線 RGBI 規則載入，屬性暫存器 8–15 設成 `10h`–`17h`。
- 修正後，dosgolem 輸出的 **RGB** 與 DOSBox-X 逐像素比對：

| 畫面 | RGB 不一致像素 | 說明 |
|---|---:|---|
| `01-title` | 0 | |
| `03-protection` | 128 | 全在盟友名稱那一列；亂數不同，問的盟友不同（docs/re/004） |
| `06-name` | 0 | |
| `07-first-play` | 0 | 含色號 8 的黑色 |
| `08-encounter` | 0 | 固定時點是 46；逐格掃描後，dosgolem 有多格與 DOSBox-X 完全相同，差異是待機動畫的相位 |

- 色號 2、A 只出現在攻擊的雷射上。在這一段遊戲的色盤是**棕色 `AA5500`、黃色 `FFFF55`**，不是 EGA 預設的綠色。
  與 DOSBox-X 錄影比對，對齊良好的格子 100% 吻合；EGA 預設綠色在任何一格都是 0%（§4）。

## 2. 遊戲怎麼設色盤

dosgolem 記錄的 `AH=10h` 呼叫（`workplace/states/05-select.log`、`workplace/determinism/A1.log`）：

- 子功能只有 `AL=00`（單一屬性暫存器，`BL` ＝ 暫存器、`BH` ＝ 值），每次成對設 `n` 與 `n+8`。
- 值要照 200 線 RGBI 解讀：bit 0／1／2 ＝ B／G／R、bit 4 ＝ 高亮，bit 3、5、6 不起作用。
  例如 `0Fh ← 77h`（白）、`0Ah ← 76h`（黃）、`02h ← 06h`（棕）。
- 淡入淡出：先把 16 個暫存器全設 0，再分幾步寫到最終值。**暫存器 8 在淡入時沒有再設**，所以色號 8 維持黑色。
- 同一個色號在不同畫面可以是不同顏色：主畫面與戰鬥中 2／A 是棕／黃；Game Over 過場把它們改成 `02h`／`72h`（綠／亮綠）。

## 3. 修正與測試（dosgolem）

| 項目 | 結果 |
|---|---|
| 新增測試 | `TestMode0DAttributeDefaults`、`TestMode0DDACAllEntries`（64 筆逐一照表）、`TestOtherPlanarModesKeepDAC`（10h／12h 回歸）、`TestMode0DColorChainEndToEnd` |
| 反向對照 | 暫時拿掉修正：前三個 0Dh 測試失敗、回歸測試照常通過 |
| 全套 `go test ./...` | 通過（15 個套件；需要原版或 CPU 語料的測試 skip） |
| 其他案例的風險 | eob1 的色盤期望雜湊不是「全零 DAC」，推定它自己寫 DAC；沒有原版素材，**未實跑確認** |

修正只改色彩，不改色號：本專案 `tools/states.sh --check` 八段畫面雜湊不變。

## 4. 色號 2、A 的驗證

`tools/battle_palette_check.py`：從 `08-encounter` 在第 86,500,000 步按攻擊，dosgolem 每 100,000 步輸出一格（54 格含色號 2／A）；
DOSBox-X 按住空白鍵 0.4 秒後以 30 fps 錄 6 秒（178 格）。兩邊亂數不同、時間軸不同，所以只比 dosgolem 色號 2／A 所在的像素，
每格 dosgolem 取吻合最多的 DOSBox-X 格。

| 指標 | 值 |
|---|---|
| 比較的像素 | 54 格、5,290 像素 |
| 實際色盤（棕／黃）吻合 | 合計 54.3%；逐格中位數 96%、最高 100% |
| EGA 預設色（綠／亮綠）吻合 | 0.0% |
| dosgolem 輸出的顏色 | 每一格都是 2 ＝ `AA5500`、A ＝ `FFFF55` |

吻合率低的格子，是 DOSBox-X 錄影裡沒有同一個雷射位置（亂數不同，雷射與數值的變化不同步）。
對齊良好的格子（例如 dosgolem 第 89,700,000 步對 DOSBox-X 第 27 格，94 像素全部相同）證明顏色與位置都對。

## 5. 工具與重跑

```sh
tools/states.sh --check                      # 八段檢查點，另輸出 <段>.rgb.png（經屬性暫存器與 DAC 解色）
tools/dosboxx-ref.sh                          # DOSBox-X：四個檢查點、遭遇、攻擊錄影
tools/py.sh tools/frame_compare.py --rgb workplace/states/07-first-play.rgb.png workplace/dosboxx/07-first-play-*.rgb
tools/py.sh tools/battle_palette_check.py <dosgolem -dump-at 目錄> workplace/dosboxx/battle
```

DOSBox-X 送鍵的兩個坑：遭遇畫面要**按住**空白鍵（`xdotool key` 的點按沒有效果）；
Xvfb 沒有視窗管理員，要先把滑鼠移進視窗。

## 6. 限制與重開條件

- 修正只在 worktree 分支；**上游合併前**本專案的狀態檔與比對都依賴 `509a639`。
- 350 線 EGA（0Fh／10h）與 VGA 平面模式的 DAC 預設沒有做（`194` §3）。
- 防拷畫面要逐像素相同，需要讓兩邊的亂數狀態一致（在出題前把 `cs:41DF` 設成同一個值），尚未做。
