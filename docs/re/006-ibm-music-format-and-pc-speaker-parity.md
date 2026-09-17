# 006：.IBM 格式與 PC 喇叭配樂對拍

狀態：量測紀錄（2026-09-17）。對應 issue #6。

## 1. 結論

- `.IBM` 是 PC 喇叭版的樂譜：**每筆 3 bytes ＝ 頻率 Hz（u16，小端序）＋ 音長刻數（u8）**，頻率 0 是休止，沒有檔頭。
- 驅動把頻率換成 PIT 通道 2 分頻值時用 **`1193180 ÷ Hz` 捨去小數**，每個音寫 `42h`（低、高）再把 `61h` 設成 `03h`，休止寫 `61h ← 00h`。
  一刻是遊戲自掛的 `INT 08h`（分頻 16571，約 72 Hz）。
- 標題畫面依序播 `OPEN0 → OPEN1 → OPEN2`，共 5,776 刻（約 80 秒），播完**不重播，也不播 `OPEN3`**（`OPEN3` 用在哪裡未查）。
- 事件層：dosgolem 寫出的 868 個音符事件與三個檔的 868 筆記錄**逐筆相同**（分頻值與刻數）。
- 聲音層：dosgolem 合成的 WAV 與 DOSBox-X 錄音比對通過——兩個方向配對率都是 100%，頻率誤差最大 0.89%，
  速度斜率 1.00004，起點殘差 p95 5.4 ms。

## 2. 格式

| 位移 | 大小 | 內容 |
|---:|---:|---|
| 0 | u16 LE | 頻率（Hz）；0 ＝ 休止 |
| 2 | u8 | 音長（刻） |

檔案長度是 3 的倍數，記錄數 ＝ 長度 ÷ 3。四個檔的記錄數與總長：

| 檔 | 記錄 | 總刻數 | 約 |
|---|---:|---:|---:|
| `OPEN0.IBM` | 335 | 1,530 | 21.2 秒 |
| `OPEN1.IBM` | 269 | 1,786 | 24.8 秒 |
| `OPEN2.IBM` | 264 | 2,460 | 34.2 秒 |
| `OPEN3.IBM` | 32 | 256 | 3.6 秒 |

音長只出現 2、4、6、8 等值；斷音是用「短音＋休止記錄」寫在資料裡的，不是驅動加的。
`GAME0–3.IBM`、`END0–3.IBM` 推定同格式（未驗證）。

## 3. 驅動行為（dosgolem 實跑，埠紀錄）

| 觀察 | 值 | 等級 |
|---|---|---|
| 載入 | 開機後依序讀 `OPEN0.IBM`–`OPEN3.IBM`（偵測不到 AdLib 時；有 AdLib 改讀 `.MID`） | confirmed |
| 一個音 | `43h ← B6h`（僅第一次）、`42h ← 低／高`、`61h ← 03h` | confirmed |
| 休止 | `61h ← 00h` | confirmed |
| 換檔 | 前一檔最後一筆結束時多寫一次 `61h ← 00h`，同一刻開始下一檔 | confirmed |
| 分頻值 | 37 種全部等於 `floor(1193180 ÷ Hz)`；用 1193182 會有 1 種對不上，四捨五入有 21 種對不上 | confirmed |
| 一刻 | 160,835 道指令（dosgolem；由 868 個間隔最小平方求得） | confirmed |
| 第一個音 | 從一刻的中途開始，比記錄短約 0.95 刻 | confirmed |

## 4. 比對

方法：`docs/spec/001-pc-speaker-music-comparison.md`；工具：`tools/music_compare.py`。

### 4.1 事件層

```sh
worktrees/dosgolem/tools/go.sh run ./cmd/probe -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war \
    -steps 1100000000 -dump-ports "42,43,61=/wp/audio/title-full-ports.tsv" -dump-tone-wav /wp/audio/golem-title-full.wav
tools/py.sh tools/music_compare.py events workplace/audio/title-full-ports.tsv workplace/original/psychic-war
# 事件層：比對 868 筆（預期共 900 筆，涵蓋 OPEN0.IBM, OPEN1.IBM, OPEN2.IBM），不一致 0
# 紀錄結束時還沒播到（或沒播完）的檔：OPEN3.IBM
```

1,100,000,000 步約 6,823 刻；最後一個音在第 5,775 刻結束，之後一千多刻都沒有發聲。

### 4.2 聲音層

dosgolem：`-dump-tone-wav`（dosgolem 分支 `psychic-war/m1-pit-tone` `6cf8a1b`，規格 `195`），22,050 Hz。
DOSBox-X：`tools/dosboxx-audio.sh`，不按鍵錄 95 秒，22,050 Hz。

| 指標 | 結果 | 判準 |
|---|---:|---:|
| 分析出的段數 | dosgolem 630、DOSBox-X 628 | — |
| 比對範圍 | 79.98 秒 | — |
| 配對率（以 dosgolem 段為分母） | 100.0% | ≥ 98% |
| 配對率（以 DOSBox-X 段為分母） | 100.0% | ≥ 98% |
| 頻率誤差最大 | 0.89% | ≤ 2% |
| 速度斜率 | 1.00004 | 與 1 差 ≤ 0.5% |
| 起點殘差 p95／最大 | 5.4 ms／14.9 ms | p95 ≤ 20 ms |

dosgolem 多出的 2 段是開頭兩個音：DOSBox-X 開始錄音時已經播過了，不在比對範圍內。

### 4.3 反向對照

| 竄改 | 結果 |
|---|---|
| dosgolem WAV 標頭取樣率改快 3%（音高、速度都偏） | 前 10 段找不到頻率相同的音，無法對齊 → 不通過 |
| dosgolem WAV 第 30–35 秒改成靜音 | DOSBox-X 段配對率 94.6%，34 段對不到、全部從第 30 秒起 → 不通過 |

比對方法在過程中修正過三處（對齊不假設第一段、高音的取樣量化門檻、配對率雙向計算），都寫進了規格 `001`；
第三處正是靠第二個反向對照才發現的。

## 5. 限制與重開條件

- 只驗了標題音樂。遊戲中的 `GAME0–3.IBM`、結局的 `END0–3.IBM`、`OPEN3.IBM` 的使用時機未驗。
- 方波合成只在 dosgolem 分支，未推上游。
- DOSBox-X 以牆上時間混音；本次主機負載下起點殘差最大 14.9 ms，負載很高時可能超過判準，重跑時先看負載。
