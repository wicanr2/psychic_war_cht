# psychic_war_cht — 給 Claude 的專案規則

## 這是什麼

《銀河超能力戰記》（Psychic War: Cosmic Soldier 2，工画堂スタジオ 1987）的繁體中文化。

走的路線**不是完整反組譯後重寫**，是把原版執行檔跑在
[`dosgolem`](https://github.com/wicanr2/dosgolem) 上，讓 golem 當**轉譯層**：

- 原版負責遊戲邏輯，一行都不重寫。
- 轉譯層在原版畫文字的那一刻攔下來，換成中文畫在放大後的畫布上。
- 同一層順便提供原版沒有的輔助功能：說明、切換語言、即時存檔、地圖、主題、作弊。

跟 ScummVM 的分工相同：**引擎是通用的，每款遊戲只要準備文本與少量位址表就能玩。**
這個專案是第一個案例，目標之一是把「老軟體轉譯層」做成可以複用的範例。

原始構想見 `IDEA.md`。

## 版本選擇（使用者定案 2026-09-17）

主要來源是 **DOS 英文版**，執行器是 **dosgolem**。

| 素材 | 角色 | 理由 |
|---|---|---|
| `Cosmic-Soldier-Psychic-War_DOS_EN.zip`（Kyodai Software 1989，IBM PC）| **主要來源**，跑在 dosgolem 上 | x86 實模式，dosgolem 已有 20 多個專案在用，是最成熟的執行器 |
| `…[PC88]…[SCP+MFI+HFE+D88].zip`、`…_PC-88_EN.zip` | 交叉 oracle（日文原文、原版畫面） | Z80 平台，pc98golem／dosgolem 都跑不了；只拿來對照文本與畫面 |
| `175-銀河超能力戰記.rar` | 譯名與術語參考 | 軟體世界代理版說明書掃描（19 頁 JPG）。人名、地名、超能力名稱優先沿用台灣當年的譯名 |

`IDEA.md` 寫的是 pc98golem。手上的日文映像都是 PC-88 版，pc98golem 的 CPU 是 x86，
兩者對不上，所以改走 DOS 版。PC-98 版（1988）目前手上沒有，不在範圍內。

## 目標：可以玩的中文版（MVP）

**MVP 的定義是「一般玩家能從開頭玩到結尾，所有遊戲內文字都是中文」**，不是對拍通過。

| 期 | 目標 |
|---|---|
| M0 | 探勘與基線：IDA 普查、狀態檔檢查點、DOSBox-X 參照 |
| M1 | 原版在 golem 上完整可跑：色盤、OPL2、PC 喇叭、決定性、全程無缺口 |
| M2 | 可遊玩前端：節拍、視窗、鍵盤、滑鼠、音訊、輸入錄放 |
| M3 | 文字攔截與文本抽取 |
| M4 | 中文繪製與全文翻譯 |
| M5 | 輔助功能：防拷、F1／F2／F3／F10、作弊 |
| M6 | 主題、打包、授權與發行 |

各期的完成定義與量測指標在 `docs/goal/`，逐條工作在 `docs/worklist.json`。
依賴是硬的：沒有 M3 的文本清單就不做 M4 的排版。

## 分層與程式碼歸屬（使用者定案 2026-09-17）

**判準：「換一款遊戲之後，這段程式碼還成立嗎？」** 成立就往 golem 上游送，不成立就留在本 repo。

| 層 | 放哪 | 內容 |
|---|---|---|
| 機器 | dosgolem `internal/` | CPU、DOS／BIOS、顯示卡、PIT、喇叭、滑鼠。缺的就補，依據是 DOSBox-X 原始碼（`~/cht/DOSBox-X-MCP-Debugger/dosbox-src/`），**不是改回去跑 DOSBox** |
| 觀測 | dosgolem `oracle/` | `Load`／`RunUntil`／`Save`／`Restore`／`OnCall`／`Indexed` |
| 轉譯層機制 | dosgolem（新增套件，spec 先寫） | 印字攔截的掛鉤介面、放大畫布與疊字、熱鍵分派、即時存檔落地、牆上時間節拍、輸入錄放 |
| 遊戲專屬 | **本 repo** | 印字常式位址、文本檔、譯名表、CJK 字型子集、地圖解讀、作弊位址、說明頁內容 |
| 前端 | **本 repo**（通用部分穩定後抽回 golem） | 視窗、音訊輸出、按鍵對應。預設 Go／Ebiten，定案寫進 spec |

⚠ **位址永遠是遊戲專屬的。** 印字常式的「演算法」可以通用，進入點、字串表偏移、
變數位置一律由本 repo 提供給 golem，不得寫死在 golem 裡。

### 預定目錄

```
cmd/psychicwar/    可遊玩的主程式（前端）
apps/psychicwar/   位址、狀態、攔截點、地圖與作弊（import dosgolem）
text/              文本檔：原文、譯文、來源位置（JSON，一則一筆）
font/              CJK 點陣字子集與烘製腳本
docs/spec/         規格，標 DRAFT／READY
docs/re/           反組譯與量測紀錄
tools/             docker 包裝
worktrees/         相關 repo 的本地 clone（gitignore）
workplace/         解開的原版、快照、暫存輸出（gitignore）
```

## 動手前

1. **SDD：spec 齊了才實作。只有標 `READY` 的規格可以動手。**
   RE 證據 → `READY` spec → 實作 → 同狀態驗證。流程見
   `~/.claude/knowledge-base/retro/retro-remake-spec-gated-workflow.md`。
2. 要跑原版或改 golem 之前，先讀 `worktrees/dosgolem/CLAUDE.md` 與 `docs/spec/000-index.md`。
   引用 dosgolem 規格要連號碼帶檔名（`015-execution-speed`，不是 `015`）。
3. 解讀未知函式之前，先把 compiler／runtime helper 分出來
   （`~/.claude/knowledge-base/retro/compiler-runtime-helper-fingerprints.md`）。
4. 任務屬於哪一類，先查 skill `re-retro-cht-rulebook` 的路由表再動手。

## `[HARD]` 硬規則

- **不得散布原版素材。** 本 repo 不含 `PW.EXE`、`.PBL`、`.BIN`、`.MID`、磁碟映像或說明書掃描。
  根目錄的 `*.zip`／`*.rar` 由使用者自備，已列入 `.gitignore`。
  需要原版的測試**缺檔就 skip**，不做代用品。
- **文本檔的原文欄位是原版素材。** 譯文可以進版控，原文欄位要能在建置時從玩家自備的執行檔
  重新抽出。在做到之前，含原文的文本檔不得進公開 repo（本 repo 目前 private）。
- **相關 repo 一律 clone 到 `worktrees/<repo>` 工作**，不動 `~/cht/dosgolem` 等主機上的原始工作區。
  在 worktree 內開分支，推上游前先回報使用者。
- **建置、分析、轉檔、測試一律走 docker**，`docker run --rm`、目前 UID/GID、限制記憶體與 CPU、
  預設 `--network none`，原版素材唯讀掛載。只清理自己建立的 container；
  禁止任何 `docker image/system/volume/builder prune` 或 `rmi`。
- **git 身分一律 `wicanr2@gmail.com`。** 進任何 repo（含 `worktrees/` 底下的每一個）先看
  `git config user.email`，再跑一次 `git log --format=%ae | sort -u` 看歷史。
- **推論標籤要誠實**：confirmed／強證據／假說／未知。反組譯產生的導覽名稱不是證據，
  要保留原始位址、operand、bytes。
- **驗證以原版為準，不以自己的內部訊號為準。** 「攔截器有觸發」不代表「那一則文字替換對了」；
  要拿快照解幀跟原版同一時點比。不用截圖量像素。

## 中文化的設計約束

- **畫布放大，不縮字**（`rulebook/81`）。原版底圖 nearest 整數倍放大，中文字用 16×16 以上點陣
  畫在放大後的畫布。不要把中文塞進原版的 8×8 字格。
- **替換單位是「一則訊息」，不是「一個字元」。** 中英文長度不對應，逐字元替換必然破版。
  文本檔的 key 用來源位置（檔名＋偏移），不用畫面上的字串內容——同一句英文可能出現在不同情境。
- **沒有翻譯的字串要看得出來。** 缺譯文時顯示原文並記錄，不能靜默略過。
- **完整性要有數字**（`rulebook/83`）：抽出幾則、翻了幾則、正常路徑實際觸發了幾則、
  觸發了卻不在文本檔裡的有幾則。最後一項必須是 0。
- 譯名統一：人名、地名、超能力名稱集中在一份譯名表，優先沿用軟體世界說明書的用法。

## 目前已知

完整證據、指令與數字見 `docs/re/001-pw-exe-first-look.md`。

| 事實 | 等級 |
|---|---|
| `PW.EXE` 在 dosgolem 上可跑到主畫面（EXEC `LOGO.EXE` → 標誌 → 標題 → 防拷 → 輸入名字） | confirmed |
| 版本是 Kyodai Software 1989 的英文移植（IBM VERSION） | confirmed |
| 顯示模式 EGA 0Dh（320×200 16 色），色盤用 `INT 10h AH=10h` 設屬性控制器 | confirmed |
| 自掛 `INT 08h`（PIT ≈72 Hz）與 `INT 09h`（直接讀掃描碼） | confirmed |
| 音樂兩條路徑：偵測不到 AdLib 載 `.IBM`（PC 喇叭），有 AdLib 載 `.MID`（OPL2） | confirmed |
| 文字散在 `PW.EXE`、`I_MENUH.BIN`、`I_MENU00–11.BIN`、`I_ENMY00–11.BIN`、`CODEH／2／11.BIN`，固定寬度欄位 | confirmed（格式未解） |
| 操作面板文字畫在圖檔上（`SCREEN.PBL` 等） | 假說 |
| `FONT.BIN` 是 8×8 字模，遊戲自己畫字 | 假說 |
| 開頭有手冊式防拷（盟友 ↔ ESP 數值） | confirmed（判定邏輯未讀） |
| `CD 75` 55 處都沒有被執行，抽查 6 處為 `DEC CH; JNZ`；byte pattern 計數不能當 `INT` 證據 | confirmed |

## 工作追蹤

- **未完成項的權威**：`docs/worklist.json`（每條對應一個 GitHub issue）。
  `tools/py.sh tools/worklist.py verify` 逐條檢查；`render` 產生 `docs/worklist.md`（不要手改）。
  條目做完就改 JSON 與關 issue，不在 markdown 打勾。
- **分期目標與驗收**：`docs/goal/`。只寫目標，不記進度。
- **Python 工具走 `tools/py.sh`**（docker）；dosgolem 的 Go 工具走 `worktrees/dosgolem/tools/go.sh`。
- 原版解壓到 `workplace/original/`，probe 輸出放 `workplace/probe/`（都 gitignore）。
  ⚠ probe 的 `-shots` 輸出是 64,000 bytes 色號陣列，不是 PNG。

## 待決事項

- 前端框架（預設 Go／Ebiten）與目標平台清單。
- 公開時機。授權一律採 RRSAL-1.0（`rulebook/85`，已定案不重問），放入 `LICENSE` 由 #36 追蹤。

## 按需載入

| 情境 | 讀 |
|---|---|
| 印字常式、字串表、位址溯源 | `rulebook/62` |
| 拿 PC-88 版畫面反推 DOS 版資料 | `rulebook/64` |
| 驗證「做完了」 | `rulebook/65` |
| CJK 畫布與字型 | `rulebook/81` |
| 跨平台打包 | `rulebook/82` |
| 素材與功能完整性 | `rulebook/83` |
| 正常玩家路徑、存讀檔試玩 | kb `retro-cht/retro-game-playtest` |
| 大量對白逐則翻譯 | kb `workflows/batch-subagent-localization.md` |
| 寫 README | `rulebook/80` |
