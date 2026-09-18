# 023：`.PBL` 圖檔格式

狀態：量測紀錄（2026-09-18）。對應 issue #18。規格 `docs/spec/010`；工具 `tools/pbl.py`。

## 1. 結論

| 事實 | 等級 |
|---|---|
| `.PBL` ＝ u16 表長（＝第 1 張圖的偏移、張數 ＝ 表長÷2）＋ u16 偏移表 ＋ 每張圖 | confirmed |
| 每張圖 ＝ u8 寬÷8、u8 高÷8、16 bytes 的 CGA 4 色對照表、RLE 資料；解壓後 `w×h×32` bytes | confirmed |
| 像素是 4bpp（一 byte 兩個像素，高半位元組在左），整張圖逐列排列；EGA 下色號就是半位元組值 | confirmed（逐像素對原版） |
| 16 bytes 對照表只在 CGA／Tandy 4 色模式用，EGA 跳過（`cs:863E` ＝ 2 或 5 時不轉換） | 強證據（程式碼分支＋26 個檔的表值域都在 0–3） |
| 貼圖位置 ＝ (`CH`×4, `CL`×4) 像素，與印字游標 `cs:610E` 同一套單位 | confirmed（兩張圖在實跑畫面上的位置相符） |
| 26 個檔、537 張圖 | confirmed |

## 2. 怎麼找到的

靜態溯源（`rulebook/62`），沒有用動態追蹤：

1. 執行檔裡有 26 個 `*.pbl` 檔名字串（`enemy00.pbl`…`kgdlogo.pbl`）。
2. INT 呼叫點表（`workplace/ida/int-table-PW.md`）裡，`sub_189BF` 一支函式就做完
   開檔（3Dh）→ 讀（3Fh）→ `LSEEK`（42h）→ 讀 → 關（3Eh），形狀正好是「讀偏移表、跳到那一張、讀進來」。
3. `sub_189BF` 用 `AH` 查 `cs:915B` 的指標表拿檔名（每筆前面多一個 byte 是磁片編號），
   用 `AL` 查剛讀進來的偏移表；讀 1388h bytes 到 `cs:92BE`，再把 `min(w×h×32, 讀到的長度)` 複製到呼叫端給的緩衝區。
   → **`cs:92BE` 的前兩個 byte 就是 IDA 標成 `byte_197CE`／`byte_197CF` 的那兩個，也就是圖的寬高。**
4. 解壓在 `sub_18B76`；`sub_18A98` 是同一套解壓但邊解邊畫到畫面，位置從 `CH`／`CL` 各乘 4 算出來。
5. 色號轉換 `sub_18C62` 拿一個 byte 的兩個半位元組各查一次 16 bytes 的表 → **一個 byte 是兩個像素**，
   排除了「EGA 位元平面」的猜測（先試平面排列，解出來是雜訊）。

## 3. 驗收數字（`docs/spec/010` §6）

| 項目 | 結果 |
|---|---|
| 537 張全部解得開（吃掉的輸入 ＝ 檔案裡的長度 ±2、輸出 ＝ `w×h×32`） | 異常 0 張 |
| `SCREEN.PBL` 第 0、1、2 張在 `07-first-play` 的實跑畫面上 | (0,0)、(0,40)、(0,80)，逐像素相同 |
| 反向對照：重複段次數改成「次數＋1」 | 三張都找不到位置（`null`） |
| 位置換算：`logo.pbl` 第 0 張（`sub_116E0`：`CH=1Ah`、`CL=06h`） | 實跑畫面 `01-title` 上在 (104, 24)，與 26×4、6×4 相同 |
| 位置換算：`menu.pbl` 第 0 張 | 實跑畫面 `07-first-play` 上在 (160, 4) |

`SCREEN.PBL` 的第 3、4 張在 `07-first-play` 找不到位置：那是畫面下半部，實跑時已經被迷宮與訊息區蓋掉。

## 4. 走過的坑（寫成規則）

- **「壓縮率超過 100%」是格式讀對了的旁證**：`FIGHT.PBL` 的小圖有幾張壓縮後比原始還大（130%），
  因為 RLE 對雜亂的像素會膨脹。看到這種數字不要急著推翻格式假設。
- **先試排列、再試壓縮**：RLE 解出來的長度剛好等於 `w×h×32`、吃掉的輸入剛好等於檔案裡的長度，
  就代表壓縮讀對了；畫面是雜訊只可能是排列錯。把兩件事分開驗，比一起猜快得多。
- **一個 byte 兩個像素 vs 位元平面，看色號轉換就知道**：轉換常式對「半位元組」查表，
  平面排列下半位元組沒有意義。**顏色相關的常式是判斷像素排列的直接證據。**

## 5. 重現

```
tools/py.sh tools/pbl.py check workplace/original/psychic-war/*.PBL
tools/py.sh tools/pbl.py dump workplace/original/psychic-war/SCREEN.PBL workplace/pbl
tools/py.sh tools/pbl.py find workplace/states/07-first-play.rgb.png workplace/original/psychic-war/SCREEN.PBL
```
