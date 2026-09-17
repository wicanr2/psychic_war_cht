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

證據、指令與數字見 `docs/re/001-pw-exe-first-look.md`（實跑初探）、`002-pw-exe-function-census.md`（函式普查）、
`003-checkpoints-and-dosboxx-reference.md`（檢查點與 DOSBox-X 參照）、`004-rng-and-determinism.md`（亂數）、
`005-ega-palette-rgb-parity.md`（色盤）、`006-ibm-music-format-and-pc-speaker-parity.md`（PC 喇叭配樂）、`007-opl2-synth-skeleton-and-comparison.md`（OPL2，未通過）、`008-replay-first-battle-and-save-load.md`（重播、存讀檔）、`009-battle-hold-cheat-and-cpu-pacing.md`（按住攻擊、敵人 HP）、`010-cpu-speed-time-base.md`（CPU 速度與時間基準）、`011-observation-addresses.md`（位置、朝向、區域、角色數值位址，地圖檔格式）、`012-opl2-event-layer.md`（OPL2 事件層）、`013-replay-route-samar-to-sivad.md`（重播路線）、`014-print-routine.md`（印字常式）、`015-text-formats.md`（文字格式）、`016-frontend-prototype.md`（前端雛形驗收）、`017-text-coverage.md`（文字覆蓋率）、`018-first-chinese-overlay.md`（第一句中文）；
被推翻的斷言集中在 `000-overturned-claims.md`。

| 事實 | 等級 |
|---|---|
| `PW.EXE` 是 Microsoft EXEPACK 壓縮檔；靜態解壓結果與執行期解壓逐位元組相同。**反組譯與位址一律用解壓後的 `PW_UNP.EXE`** | confirmed |
| `PW.EXE` 在 dosgolem 上可走到第一人稱迷宮（標題 → 防拷 → SELECT 選單 → 輸入名字 → 迷宮），畫面版面與 DOSBox-X 逐像素一致 | confirmed |
| 以 Microsoft C 1988 年版程式庫連結；另連結 Covox 1989 年音效函式庫。函式 419 個：遊戲本體 263、C 啟動碼與程式庫 61、其他模組 95 | 強推論（分群邊界） |
| 版本是 Kyodai Software 1989 的英文移植（IBM VERSION） | confirmed |
| 顯示模式 EGA 0Dh（320×200 16 色），色盤用 `INT 10h AH=10h` 設屬性控制器 | confirmed |
| 自掛 `INT 08h`（PIT ≈72 Hz）與 `INT 09h`（直接讀掃描碼） | confirmed |
| 音樂兩條路徑：偵測不到 AdLib 載 `.IBM`（PC 喇叭），有 AdLib 載 `.MID`（OPL2） | confirmed |
| `.IBM` ＝ 每筆 3 bytes（頻率 Hz u16＋刻數 u8，0 ＝ 休止）；驅動以 `1193180÷Hz` 捨去算分頻值。標題播 `OPEN0→1→2` 不重播。dosgolem 事件逐筆相同，合成的 WAV 與 DOSBox-X 錄音音高／節奏比對通過 | confirmed |
| 文字散在 `PW.EXE`、`I_MENUH.BIN`、`I_MENU00–11.BIN`、`I_ENMY00–11.BIN`、`CODEH／2／11.BIN`，固定寬度欄位 | confirmed（格式見下一列） |
| 文字來源與格式見 `docs/spec/007`（READY）：`I_MENU*`（16 bytes 一列，訊息 32 bytes＝31 字＋類型碼，選單＝提問＋10 字選項）、`I_ENMY*` 名字、`I_MAP*` 偏移 200h 的地點名稱（64×8）、`CODE*` 指令 81h 內嵌字串（`CODEnn` 只有前 700h、`CODEH` 只有前 1200h 會留在記憶體）、`PW.EXE` 內嵌文字。`text/` 共 1,389 則、要翻 1,313；重播 18 段觸發 52 則、不在文本檔 0（`docs/re/017`） | confirmed（I_MENU、CODE 指令語意）／強證據（I_ENMY、I_MAP） |
| 操作面板文字畫在圖檔上（`SCREEN.PBL` 等） | 假說 |
| 遊戲自己畫字，**兩套字型**：介面用 `FONT.BIN`（8×8，單字元 `sub_16629`／`0161:6119`，字串迴圈 `sub_16799`、`sub_167B2`、`sub_167BF`）；防拷、選單、輸入名字、故事、製作群用程式內建 6×6 小字型（`sub_1B601`／`0161:B0F1`，60 bytes 區塊 `B016`）。重播 18 段的每個字都歸到呼叫端：文字字元全部來自字串層，其餘是游標閃爍、空白與箭頭（`docs/re/014`） | confirmed（動態監看＋呼叫端歸類） |
| 開頭有手冊式防拷（盟友 ↔ ESP 數值） | confirmed（判定邏輯未讀） |
| byte pattern 計數不能當 `INT` 證據；解壓後 IDA 認得的 `INT` 指令共 93 處（`21h` 74） | confirmed |
| 遊戲只用 `AH=10h AL=00` 設屬性暫存器（200 線 RGBI 解讀），從不寫 DAC。dosgolem 修正後（分支 `psychic-war/m1-ega-palette`，未推上游）四個檢查點與遭遇戰的 RGB 與 DOSBox-X 逐像素一致 | confirmed |
| 唯一的亂數產生器是 `sub_146B7`（狀態 `cs:41DF`，初值 `544Eh`），不讀時鐘；等待按鍵的迴圈每圈推進一次，所以「隨機」來自玩家反應時間。同一組輸入在 dosgolem 上逐位元組可重現 | confirmed |
| 第一個遭遇 Shulosu 固定在第 6 次移動，與位置、亂數無關（`docs/re/011` §2）。攻擊要**按住**空白鍵才打得贏（連按 32 種都輸）；DOSBox-X 同樣成立 | confirmed |
| 敵人 HP 字組在線性 `0x509C`（`0161:3A8C`），改成 1 可一擊打贏 | 強證據 |
| **戰鬥敵我雙方的進度都綁 CPU，不是計時器**；單場長短主要看亂數與按鍵時機。速度用 DOSBox 相容 cycles（dosgolem `-cycles`，字串指令每次迭代算一個 cycle）；預設 750（AT 8 MHz），選項 240（XT），見 `docs/spec/004` | confirmed（主迴圈靜態＋成對實驗） |
| 遊戲邏輯跑在位元組碼直譯器 `sub_12766`；腳本變數在 `0x16916 ＋ 2n`：區域 `0x16966`、X `0x16968`、Y `0x1696A`、朝向 `0x16970`、前方可否通行 `0x16976`、地點 `0x16978`、HP `0x16990`、能量 `0x16994`（`docs/re/011`） | 強證據（改值驗證） |
| AdLib 驅動寫進 OPL2 的暫存器序列與 DOSBox-X 逐筆相同（5,599 筆；`docs/re/012`） | confirmed |
| 重播推進到 Sivad（區域 1）：Samar 的 Launch Pad (15,14) 選 Sivad 即可，不需道具；遭遇看步數（`docs/re/013`） | confirmed |
| 存檔 `<名>.DAT` 512 bytes；Esc → Options → Save Game → Definitely → 檔名。讀檔後畫面與存檔時相同 | confirmed |
| 防拷題目由 `sub_1695C` 以亂數出題；空白答案會顯示 `YOU ARE CLEARED` | confirmed（行為），判定邏輯未讀 |
| 前端（Go／Ebiten，`docs/spec/006`）在 Xvfb 上畫面與檢查點逐像素相同、60 秒節拍誤差 0.000%、以 xdotool 從開機打贏第一場戰鬥；即時音訊只在無音效卡的 null 輸出驗過（`docs/re/016`） | confirmed |
| 中文疊字雛形（`docs/spec/008`）：原版照畫英文，疊字層以該行英文的背景色蓋掉再畫 24×24 中文。撞牆訊息與出發平台選單逐像素對原版推算的期望值差 0（`docs/re/018`）。游標 `cs:610E`（高位元組 X、低位元組 Y，×4 像素） | confirmed |

## 工作追蹤

- **未完成項的權威**：`docs/worklist.json`（每條對應一個 GitHub issue）。
  `tools/py.sh tools/worklist.py verify` 逐條檢查；`render` 產生 `docs/worklist.md`（不要手改）。
  條目做完就改 JSON 與關 issue，不在 markdown 打勾。
- **分期目標與驗收**：`docs/goal/`。只寫目標，不記進度。
- **Python 工具走 `tools/py.sh`**（docker）；dosgolem 的 Go 工具走 `worktrees/dosgolem/tools/go.sh`。
- 原版解壓到 `workplace/original/`，probe 輸出放 `workplace/probe/`（都 gitignore）。
  ⚠ probe 的 `-shots` 輸出是 64,000 bytes 色號陣列，不是 PNG；看圖用 `tools/frames.py montage`。

| 工具 | 用途 |
|---|---|
| `tools/unexepack.py` | EXEPACK 解壓，`--verify` 對執行期傾印逐位元組比 |
| `tools/ida.sh`＋`tools/ida/census.py` | IDA 建庫與函式普查（`workplace/ida/`）。批次跑一律驗輸出檔 |
| `tools/census_report.py` | 普查 JSON＋覆蓋率 → 執行過的函式、`INT` 呼叫點表 |
| `tools/states.sh [--check] [重播檔]` | 依重播檔（預設 `replay/title-to-first-save.json`，格式 `docs/spec/003`）產生 18 段狀態檔；`--check` 驗畫面雜湊；支援按住按鍵、暫存層、畫面與記憶體相等斷言 |
| `tools/route_keys.py <朝向> <路線>` | 迷宮路線（NESW 字串）轉成按鍵序列 |
| `tools/determinism.sh` | 第一場戰鬥：同輸入兩次逐位元組相同、晚按鍵必須不同 |
| `tools/ida/dump.py` | 一次跑多個 IDA 查詢（xref／func／imm／callers／refs／dis）；⚠ 少數位址（例：14CE4、128C1）會讓 idat 異常結束，改用 `refs:` 或讀 `workplace/ida/PW_UNP.EXE.asm` |
| `tools/dosboxx-ref.sh` | DOSBox-X 走同樣四個畫面＋遭遇＋攻擊錄影，存 640×400 RGB |
| `tools/battle-pace.sh <cycles> [按住起點]` | dosgolem 量第一場戰鬥：DOSBox 相容 cycles、秒數、玩家 HP 寫入 |
| `tools/dosboxx-battle-speed.sh <cycles>` | DOSBox-X 以 8000 cycles 走到遭遇（看畫面變化送鍵）、F12＋減號降速、按住攻擊錄影 |
| `tools/battle_frames.py <rgb 目錄> [fps] [按下格]` | 戰鬥錄影判讀：敵人圖像消失的時點 |
| `tools/frame_compare.py [--rgb]` | dosgolem vs DOSBox-X：預設比版面（色號對應），`--rgb` 直接比 RGB |
| `tools/battle_palette_check.py` | 攻擊雷射上色號 2／A 的顏色驗證（時間軸不對齊時用） |
| `tools/music_compare.py events\|audio\|opl` | 配樂比對：埠紀錄 vs `.IBM`（逐筆）；兩個 WAV 的音高與節奏（`docs/spec/001`）；OPL2 樂譜 vs WAV（`docs/spec/002`，方法分辨力不足，見 docs/re/007） |
| `tools/dosboxx-audio.sh [秒] [speaker\|adlib]` | DOSBox-X 錄標題音樂 |
| `tools/dosboxx-opl.sh [秒]` | DOSBox-X 擷取 raw OPL（`DX-CAPTURE /O`），走進迷宮後 Ctrl+Q 收尾 |
| `tools/opl_events.py <opl-log> <.dro>` | OPL2 事件層逐筆比對（`docs/spec/005`），含原判準與修訂判準 |
| `tools/frames.py` | 色號陣列的變化摘要與總覽圖 |
| `tools/go-ebiten.sh` | 前端建置與 Xvfb 實跑（`PSYCHICWAR_SH` 在容器內執行指令；`PSYCHICWAR_GO_NETWORK=1` 抓 module） |
| `tools/frontend-playthrough.sh` | Xvfb＋xdotool 從開機打贏第一場戰鬥（按鍵按住 0.15 秒、等檢查點畫面再送） |
| `tools/print_trace.py plan\|report` | 重播各段記錄印字函式的暫存器與字串，合併逐字元命中 |
| `tools/char_callers.py [紀錄目錄]` | 單字元輸出的呼叫端與字碼分布（`-call-args` 紀錄，`docs/re/014` §4） |
| `tools/text_coverage.sh [重播檔]`＋`tools/text_coverage.py` | 文字覆蓋率四個數字（`docs/spec/007` §6） |
| `tools/font/bake.sh` | 烘製 `font/cjk24.bin`（倚天 24 點＋Noto 補字；來源放 `workplace/font-src/`） |
| `tools/frontend-overlay-check.sh wall\|launchpad [--expect-english KEY]`＋`tools/overlay_check.py` | 中文疊字實跑驗收，期望值由原版畫面推算（`docs/spec/008` §4） |
| `tools/text_extract.py build\|check\|stats <原版目錄>` | 產生／驗證 `text/`（`docs/spec/007`）；`menu\|enmy\|code` 是單檔除錯輸出 |

## 待決事項

- 目標平台清單（前端框架已定 Go／Ebiten，`docs/spec/006`；Windows／macOS 由 #35 追蹤）。
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
