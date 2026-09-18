# 圖檔內嵌文字的中文化：資料檔怎麼寫

《銀河超能力戰記》中文化。畫在圖檔裡的英文（招牌、道具圖鑑）換成中文的方式**不是改圖**，
而是「畫面上那一塊等於原版圖塊時，把中文蓋上去」。你要做的是**寫資料**：每一塊文字在圖裡的位置與譯文。

工作目錄一律 `/home/anr2/cht/psychic-war`。

## 一筆資料長這樣

```json
{
  "key": "ROOM0.PBL:18:danger",
  "file": "ROOM0.PBL",
  "image": 18,
  "screen": [4, 124],
  "region": [20, 10, 31, 7],
  "text": [20, 10, 5],
  "original": "DANGER",
  "translation": " 危險  ",
  "font": "cjk16",
  "note": "房間招牌：藍底黃字的警示牌"
}
```

| 欄位 | 意義 |
|---|---|
| `key` | `<檔名>:<圖號>:<這塊的英文小寫、用 `-` 連接>`，同一張圖有好幾塊就取不同名字 |
| `file`、`image` | 來源圖 |
| `screen` | **這張圖畫在畫面上的左上角**：房間圖（`ROOM0`／`ROOM1`）是 `[4, 124]`；道具圖鑑（`MAP`）與開場（`OPEN`）要另外量，見下面「怎麼找 screen」 |
| `region` | **圖內座標** `[x, y, w, h]`：拿來比對的區塊，要把那段英文完整框住，**不要框到遊戲自己會重畫的地方**（數值、游標） |
| `text` | **圖內座標** `[x, y, 格數]`：中文蓋在哪。一格 `cjk16` 是 6×7 像素、`cjk24` 是 8×8 |
| `original` | 圖上的英文（清冊裡的寫法），程式不讀，給人對照 |
| `translation` | 譯文。**字數 ≤ 格數**；用半形空白調整位置（空白格會填上底色，等於把英文塗掉） |
| `font` | `cjk16`（招牌、小字，一格 6×7）或 `cjk24`（大字，一格 8×8）。字高不到 8 像素就用 `cjk16` |

**中文矩形（`text`）必須蓋住 `region` 裡的英文筆畫**，否則會看到中英文疊在一起；檢查工具會擋。

## 工具

```
tools/py.sh tools/pbl.py dump workplace/original/psychic-war/ROOM0.PBL workplace/pbl 18   # 解出 PNG（原始大小）
tools/py.sh tools/baked_boxes.py workplace/original/psychic-war/ROOM0.PBL 18 --art 16 6 44 18   # 印出那一塊的色號圖
tools/py.sh tools/baked_lint.py --file <你的 json>          # 檢查（結束碼 0 才算完成）
tools/py.sh tools/baked_preview.py --all --file <你的 json> --out workplace/pbl/preview-<你的名字>   # 畫預覽 PNG
```

- **色號圖**是最有用的：一個字元一個像素，`7` 是灰、`9` 是藍、`E` 是黃、`F` 是白…。
  由此可以數出招牌的邊界與字的邊界（`--art <x> <y> <寬> <高>` 的座標是圖內座標）。
- **預覽 PNG** 要用 Read 工具看過，確認中文蓋得剛好、沒有蓋掉招牌邊框或旁邊的美術。
- 譯文用到的字如果不在字型子集裡，lint 會說「字型子集沒有 X」。**不要自己跑 `tools/font/bake.sh`**（會動到共用檔案），
  改用別的字，或把需要的字列在回報裡。

## 怎麼找 `screen`

- 房間圖（`ROOM0.PBL`、`ROOM1.PBL`）：固定 `[4, 124]`（實跑觀測，`docs/re/024`）。
- 其他檔案：先用 `tools/py.sh tools/pbl.py find <畫面.png> <PBL 檔>` 在實跑畫面上找；
  找不到就在 `note` 寫「screen 未觀測」，並用該圖在畫面上**最可能**的位置（例如整頁的圖多半是 `[0, 0]`）。
  找不到不影響 lint，但要誠實寫在 `note` 裡。

## 譯名與風格

- 台灣繁體中文，語氣自然；不要中國大陸用語。
- 專有名詞先查 `text/glossary.json`（人名、地名、道具）。表上沒有的自己取一個，在 `note` 寫「新譯名：英文 → 中文」。
- **代號、型號、按鍵名保留原文**（例 `AT011`、`KDG216`、`ESC`）。
- 招牌類短詞的建議：`DANGER` 危險、`GUARD` 警衛、`ELEVATOR` 電梯、`DATA` 資料、`ESCAPE` 逃生口、
  `GATE OPEN` 閘門開啟、`COMPUTER` 電腦、`INFORMATION` 資訊。同一個英文在不同圖要用同一個譯法。
- 格數不夠時可以縮短（`INFORMATION` → 資訊），但不要改變意思。

## 產出

寫成一份 JSON（路徑在你的任務說明裡），格式：

```json
{ "schema": "psychic-war-baked/1", "note": "<你負責的範圍>", "entries": [ … ] }
```

完成的條件：`tools/py.sh tools/baked_lint.py --file <你的 json>` 結束碼 0，而且**每一張圖的預覽 PNG 你都看過**。

## 邊界（嚴格遵守）

- 只寫自己的 JSON 與自己的預覽目錄；**不要改 `text/baked.json`、`text/` 其他檔案、`font/`、`tools/`、`docs/`**。
- 不 git commit／push；不要自己跑 docker 指令（用上面的 `tools/py.sh` 包裝）；Python 一律走 `tools/py.sh`。
- 不動 `~/.cache/`、`~/.claude/`、其他 repo。
- 回報：處理了幾張圖、幾塊文字、lint 結果、看過預覽的張數、需要但字型子集沒有的字、不確定的原文。
