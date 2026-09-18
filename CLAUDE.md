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
`005-ega-palette-rgb-parity.md`（色盤）、`006-ibm-music-format-and-pc-speaker-parity.md`（PC 喇叭配樂）、`007-opl2-synth-skeleton-and-comparison.md`（OPL2，未通過）、`008-replay-first-battle-and-save-load.md`（重播、存讀檔）、`009-battle-hold-cheat-and-cpu-pacing.md`（按住攻擊、敵人 HP）、`010-cpu-speed-time-base.md`（CPU 速度與時間基準）、`011-observation-addresses.md`（位置、朝向、區域、角色數值位址，地圖檔格式）、`012-opl2-event-layer.md`（OPL2 事件層）、`013-replay-route-samar-to-sivad.md`（重播路線）、`014-print-routine.md`（印字常式）、`015-text-formats.md`（文字格式）、`016-frontend-prototype.md`（前端雛形驗收）、`017-text-coverage.md`（文字覆蓋率）、`018-first-chinese-overlay.md`（第一句中文）、`019-all-paths-overlay-verification.md`（所有路徑疊字、覆蓋率、前端實跑）、`020-font-license-options.md`（字型授權選項，待決）、`021-translation-first-pass.md`（全文翻譯第一版）、`022-simulated-player-playtest.md`（模擬玩家試玩）、`023-pbl-image-format.md`（`.PBL` 圖檔格式）、`024-baked-text-overlay.md`（圖檔內嵌文字）、`025-assist-hotkeys.md`（F1／F2／F10）、`026-adlib-underrun-followup.md`（AdLib 追量）、`027-sampling-playtest-and-fixes.md`（抽測試玩與修正）、`028-copy-protection.md`（防拷）、`029-cheats.md`（作弊）、`030-auto-map.md`（自動地圖）、`031-baked-text-rooms-and-map.md`（房間招牌、道具圖鑑、開場字幕）、`032-opl2-synth-fidelity.md`（OPL2 保真度）；
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
| 文字來源與格式見 `docs/spec/007`（READY）：`I_MENU*`（16 bytes 一列，訊息 32 bytes＝31 字＋類型碼，選單＝提問＋10 字選項）、`I_ENMY*` 名字、`I_MAP*` 偏移 200h 的地點名稱（64×8）、`CODE*` 指令 81h 內嵌字串（`CODEnn` 只有前 700h、`CODEH` 只有前 1200h 會留在記憶體）、`PW.EXE` 內嵌文字。`text/` 共 1,389 則、要翻 1,313；重播 19 段觸發 55 則、不在文本檔 0（`docs/re/017`、`docs/re/019` §4） | confirmed（I_MENU、CODE 指令語意）／強證據（I_ENMY、I_MAP） |
| 操作面板與狀態欄標籤畫在圖檔上（`SCREEN.PBL` #0／#1／#3，`MENU.PBL`）；標題 Logo 在 `LOGO.PBL` | confirmed（解碼後在實跑畫面上逐像素找到，`docs/re/023`） |
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
| 中文疊字（`docs/spec/009`，疊字層是 dosgolem `xlate`，規格 `202-translation-overlay`）：原版照畫英文，疊字層以該行英文的背景色蓋掉再畫中文。四條印字路徑：A1 `6289`、A2 `62A2`、A3 `62AF`（`FONT.BIN` 8×8 → 24×24）、小字型 `B0F1`（6×7 → 18×21，字 16×15）。游標 `cs:610E`（高位元組 X、低位元組 Y，×4 像素）。8 情境 13 行逐像素差 0、反向對照差 0、前端 A／B 差 0；訊息框捲動（`6273`–`6276`）期間框內疊字凍結（`docs/re/019`） | confirmed |
| 全文翻譯第一版：1,313 則全部有譯文（保留原文 38）、過長 0、非 Big5 0；譯名表 119 筆（說明書 40、暫譯 79）；字型子集 873 字（`docs/re/021`）。重播 19 段到 Zellwal 觸發 55 則、不在文本檔 0；前端開機到第一場戰鬥的轉譯紀錄缺譯文／過長／缺字 0，剩下的英文都在圖檔上（#18） | confirmed（數字）；譯文品質未逐則對畫面 |
| 模擬玩家（Sonnet 子代理只看 `pwstep` 截圖）84 步完成開機 → 打贏第一場 → 發射台選目的地，沒用攻略；卡住 1 次（降落平台與發射台的分辨）。找到的問號裁字、音效／音樂、BBS室已修（`docs/re/022`） | confirmed（一條路線） |
| 圖檔文字：`.PBL` ＝ 偏移表＋每張 `[寬÷8][高÷8][16 bytes CGA 表]`＋RLE，4bpp 逐列；貼圖位置 ＝ `CH×4`、`CL×4`。26 檔 537 張，含文字 50 張（`docs/re/023`、`024`） | confirmed |
| 圖檔內嵌文字的中文以「畫面內容比對」觸發（dosgolem `xlate.Watcher`）：面板與狀態欄 6 塊逐像素差 0；F1 說明、F2 中英切換、F10／F11 即時存檔都通過（`docs/re/024`、`025`） | confirmed |
| 防拷：出題 `sub_1695C`、答案表 `cs:6787`（11×7）、正確答案在 `cs:66E5`；**這一版的比對被一個位元組關掉**（`cmp` 之後是無條件跳躍），任何答案都會過（`docs/re/028`） | confirmed |
| 疊字失效是**逐格**判斷（原版會在一行的一部分上面畫別的東西）；watcher 的疊字例外，整筆失效再由 watcher 重蓋（`docs/re/027`） | confirmed |
| 疊字失效另有**錨定格**規則：定色時壓在原文墨跡上的格子全部失效就整筆移除。中文比原文寬時多出來的格子壓在純色背景上，指紋永遠不變，只靠逐格判斷會留下孤字（`docs/re/031` §7.3） | confirmed（重現前後對照） |
| 圖檔文字：含文字 50 張已疊 45 張、122 塊（面板 3、房間招牌 23、道具圖鑑 17、開場字幕 1，另 `MENU.PBL` 由 `SCREEN.PBL` 的 watcher 等價涵蓋）。剩 `OPEN.PBL` #7–#9（字是從雜訊底挖空的，要疊字層的新畫法）與兩張標題美術字（定案保留）（`docs/re/031`） | confirmed |
| OPL2 合成器的判準（`docs/spec/016`）：`spec` ≥ 0.8350、`env` ≥ 0.7548、`chroma` ≥ 0.9603，門檻由「DOSBox-X 錄兩次」的上界與兩種下界推出。**前 15 秒三項全部大幅超過門檻**（0.9502／0.8669／0.9798），15 秒之後 `spec`／`env` 掉下來而 `chroma` 全程 0.956 以上 → 純音色問題，已排除 KSL、震音、顫音、指數起音、時間軸速度差、段落間隔、混音截幅（`docs/re/032`）| confirmed（量測）|
| 原版**不支援滑鼠**：解壓後 IDA 認得的 93 處 `INT` 指令裡沒有 `33h`。滑鼠是轉譯層自己提供的（`docs/spec/018`）| confirmed |
| probe 的 `-steps` 是**絕對**指令數不是「再跑幾步」；`-hold` 要逗號分隔成一個旗標（Go 的 flag 對重複旗標只留一個）。兩者都會讓重播安靜地變成「什麼都沒按」（`docs/re/033` 待寫，細節在 `docs/spec/019` 與 #14 的留言）| confirmed（正對照：戰鬥按住空白鍵，敵人 HP 38 → 20）|
| **`font/charset.txt` 是烘字的產物，不是可用字的上限**：`tools/font/bake.sh` 掃 `text/*.json` 的譯文收字，再從倚天全字庫烘。寫譯文時不要為了避開字型換詞——缺字補烘一次就有（`docs/re/031` §3.1） | confirmed |
| A2／A3 的迴圈頭也會停在字串結尾的 0；那一次不能推進行追蹤，否則記憶體裡相鄰的下一個字串會被當成同一行（Game Over 三行只有第一行是中文）| confirmed（修正前後對照） |

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
| `tools/states.sh [--check] [重播檔]` | 依重播檔（預設 `replay/title-to-first-save.json`，格式 `docs/spec/003`）產生 19 段狀態檔（到 Zellwal）；`--check` 驗畫面雜湊；支援按住按鍵、暫存層、畫面與記憶體相等斷言 |
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
| `tools/frontend-playthrough.sh` | Xvfb＋xdotool 從開機打贏第一場戰鬥（按鍵按住 0.15 秒、等檢查點畫面再送）；預設開中文疊字，轉譯紀錄在 `workplace/fe/play/text.jsonl` |
| `tools/print_trace.py plan\|report` | 重播各段記錄印字函式的暫存器與字串，合併逐字元命中 |
| `tools/char_callers.py [紀錄目錄]` | 單字元輸出的呼叫端與字碼分布（`-call-args` 紀錄，`docs/re/014` §4） |
| `tools/text_coverage.sh [重播檔]`＋`tools/text_coverage.py` | 文字覆蓋率四個數字（`docs/spec/007` §6） |
| `tools/font/bake.sh` | 烘製 `font/cjk24.golemfnt`、`cjk16.golemfnt`（倚天＋Noto 補字；來源放 `workplace/font-src/`）；兩套字型都沒有的字結束碼 1 |
| `tools/overlay_run.sh [情境…]`＋`tools/overlay_cases.json`＋`tools/overlay_check.py` | 疊字逐像素驗收：`cmd/step` 出原版參照、`pwstep` 出中文截圖；`PSYCHICWAR_WITHOUT=1` 反向對照（`docs/spec/009` §6） |
| `tools/frontend-overlay-check.sh <情境> <xdotool 鍵>` | 同一情境改在 Xvfb 前端上驗 |
| `cmd/pwstep`、`tools/playstep.sh <試玩名> new\|next\|from NNN "<動作>"` | 逐步操作（dosgolem 規格 `201-step-actions`）加轉譯層：一步一個狀態檔＋960×600 中文截圖＋轉譯紀錄；模擬玩家試玩用，代理指令範本 `tools/playtest-instructions.md`（`docs/re/022`） |
| `tools/pbl.py list\|dump\|find\|check` | `.PBL` 圖檔解碼與定位（`docs/spec/010`） |
| `tools/baked_run.sh`、`tools/frontend-baked-check.sh` | 圖檔內嵌文字的疊字驗收（`docs/spec/011` §5）；`PSYCHICWAR_WITHOUT=<key>` 反向對照 |
| `tools/frontend-hotkeys-check.sh`、`tools/help_check.py` | F1／F2／F10／F11 的實跑驗收（`docs/spec/012` §5） |
| `tools/frontend-cheat-check.sh` | 作弊熱鍵驗收（`docs/spec/014`）：開與不開各一次，probe 從即時存檔讀值 |
| `tools/frontend-map-check.sh`、`tools/map_check.py` | F3 自動地圖驗收（`docs/spec/015`）：座標由存檔推、逐格比顏色 |
| `tools/baked_boxes.py`、`tools/baked_lint.py`、`tools/baked_preview.py` | 圖檔疊字的資料：量文字框、檢查一筆一筆、畫預覽（`tools/baked-authoring-instructions.md`）|
| `tools/baked_report.py` | 圖檔內嵌文字的覆蓋率（清冊 vs 已疊中文） |
| `tools/baked_find.py <截圖…>` | 哪一張截圖上有哪張圖：拿 `.PBL` 解出的原版圖塊比 `screen` 座標的像素。招牌多半在還沒走到的房間，逐像素驗收前要先知道去哪裡驗 |
| `tools/baked_merge.py <來源.json…>` | 分頭寫的疊字資料合併進 `text/baked.json`（排序後寫，diff 看得懂） |
| `tools/playtest-sampling-instructions.md` | 抽測試玩的代理指令範本（`docs/re/027`） |
| `tools/l10n_batches.py prep\|merge\|sweep`、`tools/l10n_check.py` | 分批翻譯、合併核對、一致性掃描；譯者自我檢查（`docs/re/021`） |
| `tools/text_extract.py lint` | 譯文行寬與 Big5 檢查（`docs/spec/009` §3） |
| `tools/text_extract.py build\|check\|stats <原版目錄>` | 產生／驗證 `text/`（`docs/spec/007`）；`menu\|enmy\|code` 是單檔除錯輸出 |

## 待決事項

- 目標平台清單（前端框架已定 Go／Ebiten，`docs/spec/006`；Windows／macOS 由 #35 追蹤）。
- **畫面上要出現的中文，來源一律放進 `text/` 的資料檔**（`help.json`、`baked.json`…），否則字型子集收不到、畫面缺字（`docs/re/030` §2）。
- 公開時機。授權一律採 RRSAL-1.0（`rulebook/85`，已定案不重問），放入 `LICENSE` 由 #36 追蹤。
- **中文字模用倚天字形發行（使用者定案 2026-09-18，`docs/re/020` §4，不重問）**；README 要寫明來源是倚天中文系統 3.53 的點陣子集與下架聯絡方式（#36）。
- **試玩用抽測（使用者定案 2026-09-18）**：不必從頭玩到結局，抽幾段正常玩家路徑確認畫面上是中文即可。
- **#8「從開頭跑到結局，沒有未實作的服務」不執行（使用者定案 2026-09-18）**，已移到 `docs/worklist.json` 的 `dropped`。
- **輸入名字只支援 ASCII（使用者定案 2026-09-18）**：存檔檔名沿用 `<名>.DAT`，提示文字中文化，不做中文名字的顯示層轉換。

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
