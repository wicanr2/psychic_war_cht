# 011：圖檔內嵌文字的中文疊字

狀態：**READY**
日期：2026-09-18
對應：issue #18、#25；`.PBL` 格式 `docs/spec/010`；印字路徑的疊字 `docs/spec/009`；
dosgolem 規格 `203-baked-text-watchers`（`xlate.Watcher`）、`202-translation-overlay`

## 1. 範圍

畫在 `.PBL` 圖檔裡的英文（操作面板、狀態欄標籤、房間招牌、道具圖鑑的說明）換成中文。
不在範圍：標題 Logo 的美術字（見 §6）、程式畫的文字（`docs/spec/009`）。

## 2. 做法

**不做替換圖，也不改原版檔案**：沿用 `docs/spec/009` 的疊字——同一套中文字型、同一套定色與失效規則，
只是換一個觸發點。

- 觸發點是「畫面上那一塊等於原版圖塊」（dosgolem `xlate.Watcher`，規格 `203`）。
  原版圖塊由 `apps/psychicwar/pbl` 在啟動時從玩家自備的 `.PBL` 解出來（`docs/spec/010`）。
- 比對到就蓋上中文；那一塊被蓋掉、換頁、捲走之後疊字失效，畫面回到原樣會再蓋一次。
- 好處：不必攔三條繪圖路徑（直接畫、解到緩衝區再貼、腳本決定位置），同一張圖在不同畫面出現也一樣有效。

## 3. 資料檔

`text/baked.json`，schema `psychic-war-baked/1`（與 `docs/spec/007` 的文本檔分開，`LoadText` 不會讀到）：

```json
{
  "schema": "psychic-war-baked/1",
  "entries": [
    {
      "key": "SCREEN.PBL:0:advance",
      "file": "SCREEN.PBL",
      "image": 0,
      "screen": [0, 0],
      "region": [176, 8, 56, 8],
      "text": [200, 8, 2],
      "original": "ADVANCE",
      "translation": "前進",
      "note": ""
    }
  ]
}
```

| 欄位 | 意義 |
|---|---|
| `key` | `<檔名>:<圖號>:<區塊名>` |
| `file`、`image` | 來源圖（`docs/spec/010` 的檔案索引與圖號） |
| `screen` | 這張圖畫在畫面上的左上角（原版像素）。實跑觀測或由 `CH×4`、`CL×4` 推得 |
| `region` | 要比對的區塊，**圖內座標** `[x, y, w, h]`。比對用的是整塊像素，所以要把文字完整框住 |
| `text` | 中文要蓋的位置與格數，**圖內座標** `[x, y, 格數]`；一格 8×8 原版像素（放大 3 倍後 24×24） |
| `original` | 圖上的英文（清冊用，程式不看） |
| `translation` | 譯文；空字串＝還沒翻，記 `missing-translation`，不蓋 |
| `swap_colors` | 定色時把背景與前景對調（dosgolem `202-translation-overlay` §2.3）。整條橫幅被字填滿時，字的像素比底多，「最多的當背景」會反過來 |

- 畫面座標 ＝ `screen` ＋ 區塊或文字的圖內座標。
- 顏色由疊字層自己從區塊像素取（背景 ＝ 最多的色號、前景 ＝ 第二多），與 `docs/spec/009` 相同。
  取色看的是**疊字自己蓋住的那塊**（`screen` ＋ `text` 起算，格數 × 格寬），不是 `region`。
  字比底密時設 `swap_colors`。底紋的顏色比字色還多時，前景會抓到底紋——這種區塊目前沒有指定色號的辦法。
- 譯文長度超過格數：記 `too-long`，該筆不蓋（與 `009` 一致）。

## 4. 執行期

- `apps/psychicwar/pbl`：`.PBL` 解碼（`docs/spec/010`），只讀玩家自備的原版檔。
- `translator.LoadBaked(dir)` 讀資料檔；`Translator.AttachBaked(orig string)` 對每一筆登記 `xlate.Watcher`：
  - `Want` ＝ 解出來的圖在 `region` 內的色號。
  - `Make` ＝ 建一筆疊字（字型 `cjk24`、一格 8×8、位置 `screen + text`）。
- 前端（`cmd/psychicwar`）與 `cmd/pwstep` 都在建立轉譯層之後呼叫 `AttachBaked`；狀態檔還原之後 watcher 要重新登記（規格 `203` §2.3）。
- 原版檔案缺少時：不登記、記一筆 `baked-missing`，遊戲照樣跑。

## 5. 驗收

1. 單元測試：`.PBL` 解碼（Go 版）與 `tools/pbl.py` 對同一張圖的色號陣列逐 byte 相同；資料檔的 `region`、`text` 超出圖範圍時載入失敗並指出是哪一筆。
2. 逐像素（`pwstep` ＋ `tools/overlay_check.py --baked`）：
   - 操作面板的 4 個區塊（`ADVANCE`、`LOOK ASIDE`、`TURN`、`ESC OPERATION`）與狀態欄的 `PLACE`、`DIRECTION`，
     在 `07-first-play` 的畫面上與期望值差 0。期望值 ＝ 原版畫面那一塊 ＋ 依 §3 畫上中文。
   - 反向對照：清掉其中一筆的譯文，該塊與原版英文放大 3 倍差 0。
3. 前端（Xvfb）：同一個情境用 `tools/frontend-overlay-check.sh` 再驗一次。
4. 檢查點畫面沒有英文圖字：`07-first-play`、`08-encounter`、`09-battle-won`、`16-sivad` 四張截圖，
   清冊裡列為「有文字」且出現在該畫面的區塊，都已經蓋上中文（以轉譯紀錄的 `stamp` 事件與截圖判讀）。
5. 合成畫面（`apps/psychicwar/translator` 的 `TestBakedEntriesOnSyntheticScreen`）：**每一筆**資料都要通過。
   做法是把 `.PBL` 解出來的原版圖塊貼到空白畫面上該在的座標，跑一次 `Frame`，
   檢查 watcher 有沒有比對到、疊字的座標與格數對不對、字有沒有畫進矩形裡，並以空白畫面做反向對照。
   像素仍然是原版的，所以這一項驗的是「圖出現時會不會蓋、蓋在哪裡」；
   沒驗到的只有「那個房間會不會出現在正常路徑上」。
   - 大部分招牌在還沒走到的房間，第 2 項的實跑只涵蓋當下畫面上有的圖。
     要知道某張圖該在哪個狀態驗，用 `tools/py.sh tools/baked_find.py <截圖…>`：
     它拿原版圖塊比截圖上 `screen` 座標的像素，回答「哪一張截圖上有這張圖」。
     已經蓋上中文的區塊不會命中——這支工具吃的是還沒疊字時留下的截圖。

## 6. 不做

- **標題 Logo 與商標**（`PSYCHIC WAR COSMIC SOLDIER`、`KYODAI`、`KGD SOFT`、`SF ROLEPLAYING`）：
  那是美術字，換成中文要重畫且會破壞原版觀感；保留原樣，理由記在此。
- 改寫 `.PBL` 檔案（不散布原版素材，玩家自備）。
- 4 色模式。
