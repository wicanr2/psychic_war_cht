# 001：PW.EXE 初探——在 dosgolem 上實跑與靜態掃描

狀態：量測紀錄（2026-09-17）

## 1. 輸入

| 檔案 | 大小 | SHA-256 |
|---|---:|---|
| `PW.EXE` | 58,649 | `88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49` |
| `LOGO.EXE` | 23,136 | `08a4e38b051c29381296fdf3d075eac55dc215d52cf43b9caf955c8deccfc1ba` |

來源：`Cosmic-Soldier-Psychic-War_DOS_EN.zip`（106 個檔）。解壓到 `workplace/original/psychic-war/`（gitignore）。

執行器：`worktrees/dosgolem` @ `d9c0c27`。

## 2. 怎麼跑

```sh
cd worktrees/dosgolem
DOSGOLEM_ORIG=../../workplace/original \
DOSGOLEM_EXTRA_MOUNT=$PWD/../../workplace/probe:/probe \
tools/go.sh run ./cmd/probe -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war \
    -steps 400000000 \
    -press space,enter,space,enter,space,enter,space,enter \
    -press-at 30000000 -press-every 40000000 \
    -shots 40000000:/probe/k40000000.png,…  -coverage /probe/cov.json
```

加 `-adlib` 再跑一次，比較載入的音樂檔。

⚠ `-shots` 寫出的是 64,000 bytes 的**色號陣列**，不是 PNG，副檔名會誤導。
另存的 `.pal` 是 VGA DAC，在這支遊戲裡全部 ≤ 1（見 §4.3）。
看畫面時用 EGA 預設 16 色直接對色號上色。

## 3. 實跑結果

| 項目 | 結果 | 等級 |
|---|---|---|
| 能否啟動 | 能。4 億道指令內沒有停下，程式仍在執行 | confirmed |
| 啟動流程 | `PW.EXE` 以 `INT 21h AH=4Bh` EXEC `LOGO.EXE`（exit=0）→ Kogado 標誌 → 標題 → 防拷 → 主畫面「ENTER YOUR NAME」 | confirmed（畫面） |
| 顯示模式 | `INT 10h AH=00h` 設 **0Dh**（EGA 320×200 16 色），全程只切換一次 | confirmed |
| 平面寫入 | write mode 0 與 1 | confirmed |
| 色盤 | `INT 10h AH=10h` 呼叫 18 次（屬性控制器） | confirmed |
| 計時器 | 自掛 `INT 08h`，PIT 通道 0 分頻 16571（≈72 Hz） | confirmed |
| 鍵盤 | 自掛 `INT 09h`（`0161:B4FE`），讀掃描碼；沒有 `INT 16h` 位元組 | confirmed |
| 滑鼠 | 執行檔裡有 1 處 `CD 33`，4 億步內沒有執行 | 強證據（尚未確認是否為指令） |
| 音樂（無 AdLib） | 偵測 OPL2 失敗 → 載入 `OPEN0–3.IBM`、`GAME0–3.IBM`，PC 喇叭發聲 | confirmed |
| 音樂（`-adlib`） | 改載 `OPEN0–3.MID`、`GAME0–3.MID`；OPL2 暫存器寫入 1,751 筆 | confirmed |
| 未實作服務 | 這段路徑上沒有回報 | confirmed（僅限走過的路徑） |
| 速度 | 11.7–16.4 M 道指令／秒（`--cpus 2`） | confirmed |

進入主畫面前後載入的檔案：`CODEH.BIN`、`I_MENUH.BIN`、`MAZE.BIN`、`FONT.BIN`、`MENU.PBL`、
`FIGHT.PBL`、`SCREEN.PBL`、`I_DATA.BIN`。

## 4. 靜態掃描

### 4.1 `CD 75` 不能當中斷的證據

`PW.EXE` 裡 `CD 75` 出現 55 次，覆蓋率顯示沒有一處被當指令執行。抽查前 6 處，前後文都是
`FE CD 75 xx`，即 `DEC CH` 後接 `JNZ`。**這 55 處不能當 `INT 75h` 的證據**；其餘 49 處的解讀留給 IDA 普查（#1）。

同理，byte pattern 計數只能當線索：`INT 21h` 有 76 處但這段路徑只執行 3 處、
`INT 10h` 9 處執行 2 處、`INT 1Ah` 5 處執行 0 處。

### 4.2 文字放在哪裡

| 來源 | 內容 | 版面 |
|---|---|---|
| `PW.EXE` | 開場故事、防拷問答、輸入名字、Game Over、工作人員名單 | 固定寬度字串，以空白補齊 |
| `I_MENUH.BIN`（3,584 bytes） | 主選單問句與選項 | 問句 16 字 × 2 行，選項 10 字，後接控制位元組 |
| `I_MENU00–11.BIN`（各 2,816 bytes） | 各區域的問句與選項 | 同上 |
| `I_ENMY00–11.BIN`（各 400 bytes） | 敵人名稱 | 8 字，每檔 5 個 |
| `CODEH.BIN`、`CODE2.BIN`、`CODE11.BIN` | 方位名稱、部分訊息 | 部分為 8 字固定寬 |
| `CODE0–10.BIN`（除 2） | 沒有可辨識的 ASCII 字串 | 未知：可能是腳本位元組碼，也可能含編碼過的文字 |
| `SCREEN.PBL` 等圖檔 | 「ADVANCE」「LOOK ASIDE」「TURN」「OPERATION」 | 任何檔案的 ASCII 裡都找不到，推定是畫在圖上的文字（假說） |

以「6 字以上可列印字元」粗估，選單類檔案合計約 2 萬字元。這個數字混有雜訊，
正式清冊以格式解讀後的欄位為準。

`SHURI.BFS`（10,712 bytes）的內容是另一款戰爭遊戲的文字（沖繩戰），與本作無關，推定是發行時混入的檔案。

### 4.3 dosgolem 目前的畫面輸出缺口

- `-dump-screen-png`（經屬性控制器與 DAC）在同一時點輸出全黑，但平面上有 4,757 個非零像素。
- VGA DAC 全部 ≤ 1：mode 0Dh 下遊戲不寫 DAC，只用 `AH=10h` 設屬性控制器。
- `-dump-ega` 用固定的 EGA 預設 16 色，沒有套用遊戲設定的色盤。

所以目前看得到正確的**圖形**，但**顏色**還沒驗證。等級：強證據（成因未逐行確認）。

## 5. 防拷

主畫面之前有手冊式防拷：先要求「把盟友和基地配對」，再顯示盟友名稱並要求輸入 ESP 數值。
隨便按鍵時會顯示「YOU ARE CLEARED」並進入遊戲——判定邏輯與失敗分支都還沒讀。

## 6. 尚未回答

- 印字常式在哪、吃什麼參數（`FONT.BIN` 是否為 8×8 字模）。
- `CODE*.BIN` 的格式。
- `.PBL` 圖檔格式，以及哪些圖含文字。
- 亂數來源（`INT 1Ah` 未執行；是否只靠計時器）。
- OPL2 合成：dosgolem 有 OPL2 暫存器紀錄，沒有合成器。
