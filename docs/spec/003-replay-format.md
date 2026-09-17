# 003：重播檔格式

狀態：**READY**
日期：2026-09-17
對應：issue #8（全程重播）；決定性依據 `docs/re/004`

## 1. 目的

把「從開機走到某個畫面」寫成資料，而不是寫死在腳本裡。一份重播檔可以：

- 在 dosgolem 上**逐位元組重現**（按鍵以指令數計，docs/re/004）；
- 產出一串狀態檔檢查點，之後的實驗從檢查點展開；
- 用畫面雜湊驗證沒有走偏。

## 2. 格式（JSON）

```json
{
  "schema": "psychic-war-replay/1",
  "note": "說明",
  "dosgolem": "psychic-war/m1-opl2",
  "pw_exe_sha256": "88321206…",
  "memory_ranges": ["lin:16966:52"],
  "segments": [
    {
      "name": "01-title",
      "from": "",
      "save_at": 12000000,
      "keys": [],
      "key_at": 0,
      "key_every": 0,
      "holds": [],
      "scratch": false,
      "expect_same_frame_as": "",
      "expect_same_memory_as": "",
      "comment": "標題，等按鍵"
    }
  ]
}
```

| 欄位 | 意義 |
|---|---|
| `name` | 段名，也是狀態檔名；依字典序即執行順序 |
| `from` | 從哪一段的狀態檔接著跑；空字串 ＝ 開機 |
| `save_at` | 在第幾道指令存狀態與畫面（絕對指令數） |
| `keys` | probe `-press` 的鍵名或兩位十六進位掃描碼 |
| `key_at` | 第一個事件的指令數 |
| `key_every` | **每個事件**的間隔：一個鍵是「按下」與「放開」兩個事件，所以相鄰兩鍵相隔 2 × `key_every`；0 ＝ probe 預設 |
| `holds` | 按住按鍵：`["<鍵>@<起點>+<長度>", …]`，起點與長度是指令數或加 `ms`（dosgolem `197`，probe `-hold`，typematic 開） |
| `scratch` | 是否開可寫暫存層（遊戲存檔要真的留下來時為 true）；整份重播共用一個暫存層目錄 |
| `expect_same_frame_as` | 這一段存的畫面必須與哪一段**完全相同**（例：讀檔後應回到存檔時的畫面）；空字串 ＝ 不檢查 |
| `expect_same_memory_as` | 這一段在 `save_at` 時 `memory_ranges` 的內容必須與哪一段**完全相同**；空字串 ＝ 不檢查。畫面會因訊息區不同而不相等、但遊戲狀態相同時用這個（例：在降落平台存檔，讀檔後訊息區是空的） |
| `comment` | 給人看的說明 |
| `memory_ranges`（頂層） | 每段在 `save_at` 傾印的記憶體範圍：`lin:<十六進位線性位址>:<十進位長度>`；位址意義見 `docs/re/011`。沒有此欄位時不傾印 |

期望的畫面雜湊不放在重播檔裡，放在 `tools/states.expected`（`name`、`save_at`、雜湊前 16 碼）。

## 3. 規則

- **改任何一段的按鍵或時機，之後所有段的畫面都可能改變**（亂數隨指令數推進），要一起更新期望雜湊。
- 重播檔綁 dosgolem 版本與 `PW.EXE` 雜湊；換版本先跑 `tools/states.sh --check`。
- `keys` 走 FIFO 佇列、會節流：排得太密的鍵會延後送出（docs/re/008）。需要按住或精確時機時用 `holds`（定時送出、不節流）。
- 舊的重播檔沒有 `holds` 欄位時視為空陣列；沒有 `expect_same_memory_as` 時視為空字串。
- 暫存層每次執行前清空，重播必須自己產生它需要讀的存檔。

## 4. 驗收

1. `tools/states.sh` 改讀重播檔，原有八段的畫面雜湊不變。
2. `expect_same_frame_as` 不相等時報錯（反向對照：故意指到別段要失敗）。
3. `expect_same_memory_as` 不相等時報錯（反向對照：故意指到別段要失敗）；指到的段必須在 `memory_ranges` 有傾印。
