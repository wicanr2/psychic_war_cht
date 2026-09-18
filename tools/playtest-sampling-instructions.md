# 抽測試玩指令：《銀河超能力戰記》中文版（第二段）

你是玩家，接著上一次的進度玩。目的是**抽幾段正常玩家會走的路徑，確認畫面上是中文**，
不必從頭玩到結局（使用者定案 2026-09-18）。

## 起點

`tools/playstep.sh play2 …` 的第 001 步已經放好：上一次試玩結束的畫面——剛搭發射台抵達**賽瓦德**（Sivad）的降落平台。

## 三段抽測（依序做，每段有步數上限）

| 段 | 要做的事 | 上限 |
|---|---|---|
| A | **存檔與讀檔**：Esc → 選項 → 存檔，取個檔名存好；再走幾步，然後 Esc → 選項 → 讀取進度，把剛才的存檔讀回來 | 120 步 |
| B | **在賽瓦德找到一個重要地點**：治療室（可以補血）或下一個發射台，走到它、進去、看它的訊息 | 120 步 |
| C | **打一場戰鬥**：遇到敵人就打（按住空白鍵），打到分出勝負 | 120 步 |

每一段都要記錄畫面上看到的文字是不是中文、有沒有排版或操作問題。
段落之間不必重來；C 段常常在 B 段途中就自然發生，那就直接接著記。

## 操作方式

在 `/home/anr2/cht/psychic-war` 底下：

```
tools/playstep.sh play2 next "<動作>"          # 從上一步接著跑
tools/playstep.sh play2 from <NNN> "<動作>"    # 從第 NNN 步重來（像玩家讀進度）
```

- 每一步都會印出截圖路徑（`workplace/playtest/play2/NNN.png`）與這一步的轉譯紀錄，**每一步都要 Read 截圖**再決定下一步。
- 動作：`tap:<鍵>`、`hold:<鍵>:<毫秒>`、`wait:<毫秒>`、`type:<文字>`，逗號分隔。
  鍵名 `Return`、`Space`、`Esc`、`Up`、`Down`、`Left`、`Right`、`Backspace`、字母、數字。
- 遊戲很慢，按鍵後等 2–4 秒再看；一步不要超過 15 秒遊戲時間。
- 看不到結果的連續按鍵最多 3 個。
- 轉譯紀錄出現 `missing-translation`、`too-long`、`missing-glyph`、`unsupported-control`、`unknown-caller` 就是問題，記下來。

## 你事先知道的事

- ↑ 前進、←→ 轉向、↓ 向後轉、Esc 開選單、空白鍵攻擊（要按住）。
- Esc 選單裡有「選項」，底下有存檔與讀取。
- 每走一步 HP 與能量會慢慢回復；太空站裡有治療室。

**不准看**：`docs/`、`replay/`、`apps/`、`tools/` 的原始碼、`text/`、`workplace/states/`、上一次的試玩紀錄。
同一段卡住 60 步可以讀 `docs/re/013-replay-route-samar-to-sivad.md` 的 §2、§3 當攻略，讀之前先在紀錄寫明。

## 紀錄

寫 `workplace/playtest/play2/record.md`（繁體中文），每 10 步至少更新一次：

```markdown
# 抽測試玩紀錄 play2

## A 段：存讀檔
| 步 | 看到什麼 | 按什麼、為什麼 | 結果 |

## B 段：重要地點
## C 段：戰鬥

## 問題
| # | 步 | 類型 | 描述 | 截圖 |

## 摘要
- 每段步數、是否完成、卡住次數、重來次數、是否使用攻略
- 畫面上還有英文的地方（分成「遊戲文字」與「圖片上的字」）
```

問題類型：`英文殘留`、`圖片文字`、`排版`、`翻譯`、`操作`、`轉譯紀錄`、`其他`。同一個問題只記一次。

## 邊界（嚴格遵守）

- 只寫 `workplace/playtest/play2/` 底下的檔案（`record.md`；狀態與截圖由 `playstep.sh` 自己寫）。
- 不改其他檔案；不 git commit／push。
- 不要自己跑 docker 指令；**禁止任何 docker image／volume／system／builder prune、`docker rmi`、`docker rm` 別人的 container**。
- **要跑 Python 一律用 `tools/py.sh`（docker），不要在主機上跑 python3／pip。**
- 不動 `~/.cache/`、`~/.claude/`、其他 repo。
- 做完回報：三段各自的步數與是否完成、卡住與重來次數、是否使用攻略、問題清單（前 15 條）。
