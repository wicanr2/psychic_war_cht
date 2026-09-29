# AGENTS：《銀河超能力戰記》繁體中文化

這是這個專案的唯一規則文件：怎麼工作（§1–§11）、已經確認的事實（§14）、
有哪些工具（§15）。動手前先看 §4 的流程閘門與 §16 的按需載入表。

## 1. 定位與邊界

用 **dosgolem 的執行期攔截與覆繪**把《銀河超能力戰記》（Psychic War: Cosmic Soldier 2，
工画堂スタジオ 1987；本專案用 Kyodai Software 1989 的 DOS 英文移植版）中文化。
不是 remake、不改 EXE、不重作遊戲規則或資料格式。

- 原版 `PW.EXE`、`.PBL`、`.BIN`、存檔格式、條件比較、檔名一律保持原樣。
- 只在已證實的輸出路徑上，以「原文＋呼叫位置＋畫面座標」辨識字串，在覆繪層畫中文。
  原版的記憶體與畫面緩衝不寫入。
- 英文原文是穩定比對鍵。譯文只影響顯示，不得進入比較、查找、序列化、檔名或存檔。
- 原版遊戲、說明書掃描、雜誌攻略掃描是使用者本機輸入，不進 git、不進發行包。
  `175-*/`、`*.jpg`、`*.zip`、`*.rar` 都在 `.gitignore`。

- 開頭的手冊式防拷：F4 說明頁在那個畫面會多顯示一行答案
  （`text/help.json` 的 `protection_label`）。這一版原廠的比對本來就被一個位元組關掉，
  任何答案都會過（`docs/re/028`），顯示答案不改變任何判定。

## 2. 版本選擇與素材（使用者定案 2026-09-17）

主要來源是 **DOS 英文版**，執行器是 **dosgolem**。

| 素材 | 角色 | 理由 |
|---|---|---|
| `Cosmic-Soldier-Psychic-War_DOS_EN.zip`（Kyodai Software 1989，IBM PC）| **主要來源**，跑在 dosgolem 上 | x86 實模式，dosgolem 已有 20 多個專案在用，是最成熟的執行器 |
| `…[PC88]…[SCP+MFI+HFE+D88].zip`、`…_PC-88_EN.zip` | 交叉 oracle（日文原文、原版畫面） | Z80 平台，pc98golem／dosgolem 都跑不了；只拿來對照文本與畫面 |
| `175-銀河超能力戰記.rar` | 譯名與術語參考 | 軟體世界代理版說明書掃描（19 頁 JPG）。人名、地名、超能力名稱優先沿用台灣當年的譯名 |

`IDEA.md` 寫的是 pc98golem。手上的日文映像都是 PC-88 版，pc98golem 的 CPU 是 x86，
兩者對不上，所以改走 DOS 版。PC-98 版（1988）目前手上沒有，不在範圍內。


## 3. 架構

```text
玩家（鍵盤、滑鼠、螢幕、喇叭）
  │ 鍵盤 → 掃描碼（apps/psychicwar/keymap.go）   ▲ 原版畫格 ×3 ＋ 中文覆繪；OPL2／PC 喇叭
  ▼ 滑鼠 → 視窗座標 ÷ 倍率 = 原版座標            │
cmd/psychicwar（Go／Ebiten）：F4–F8、F10–F12 前端自己處理，不送進原版
  │                                              ▲
  ▼                                              │
dosgolem（worktrees/dosgolem，分支 psychic-war/r5-text）
  session／oracle：int 09h 鍵盤、int 33h 滑鼠、依 DOSBox 相容 cycles 推進
  xlate：疊字層（規格 202-translation-overlay）、畫面內容比對（203-baked-text-watchers）
  │ 唯讀觀察
  ▼
apps/psychicwar：印字常式位址、文本載入、疊字建立、地圖、作弊、戰鬥偵測
```

判準：**「換一款遊戲之後，這段程式碼還成立嗎？」** 成立就往 dosgolem 送，不成立留本 repo。
⚠ **位址永遠是遊戲專屬的**：演算法可以通用，進入點與偏移一律由本 repo 提供。

dosgolem 支援不到的，依 DOSBox-X 原始碼補進 dosgolem，**不是改回去跑 DOSBox**。
在 `worktrees/dosgolem` 的分支上工作，不動主機上的原始工作區；推上游前先回報使用者。

## 4. 工作流程與證據閘門

```text
原版證據 → docs/spec DRAFT → READY → 實作 → 同狀態驗證
```

- **規格沒標 READY 不寫實作程式。** 目前 23 份規格全部 READY。
- 設計類的規格（例如版面）先出 DRAFT 給使用者選方案，定案後才改 READY。
- **證據等級一律標示**：confirmed／強證據／假說／未知。反組譯產生的導覽名稱不是證據，
  要保留原始位址、operand、bytes。
- 未完成項的權威是 `docs/worklist.json`，每條對應一個 GitHub issue，每條掛一個可跑的 verify。
  條目做完改 JSON 並關 issue，不在 markdown 打勾。

## 5. 辨識輸出事件的規則

- 印字用**呼叫端位址**辨識（四條路徑：`FONT.BIN` 的 8×8 三條、內建 6×6 小字型一條）。
- **同一個呼叫端會服務多個畫面**，要再用游標座標或字串來源區分。
- 迴圈頭也會停在字串結尾的 0，那一次不能推進行追蹤，否則記憶體裡相鄰的下一個字串
  會被當成同一行。
- 畫在圖檔裡的文字沒有字串可攔，用**畫面內容比對**觸發（`xlate.Watcher`）。
- 疊字失效是**逐格**判斷，另有**錨定格**規則：定色時壓在原文墨跡上的格子全部失效才整筆移除。
  中文比原文寬時多出來的格子壓在純色背景上，指紋永遠不變，只靠逐格判斷會留下孤字。

## 6. 譯文與字型治理

- 譯文的唯一正式來源是 `text/*.json`（**JSON 不是 TSV**），一則一筆，key 用來源位置
  （檔名＋偏移）不用畫面上的字串內容。
- **替換單位是「一則訊息」不是「一個字元」。** 中英文長度不對應，逐字元替換必然破版。
- **缺譯文時顯示原文並記錄**，不能靜默略過。
- **畫面上要出現的中文一律放進 `text/` 的資料檔**，不寫死在程式裡——烘字型只掃 `text/`，
  寫死在程式裡的字會靜默缺字。
- **改完任何譯文都要重跑 `tools/font/bake.sh`。** 沒重烘不會報錯，新字在畫面上是靜默地畫不出來。
- 字型收全部可見 ASCII，理由是防拷答案在執行時才從記憶體讀出來，不在 `text/` 裡。
- 專名以軟體世界代理版說明書為準（`docs/manual-1989.md`），說明書沒給中文的才自訂並標
  `provisional`。**說明書給用途說明不等於給定名**，後者才能當定譯。

## 7. 驗收方法

- **同狀態逐像素**：同一個 state、同一組輸入，比新舊畫面。像素差異只能落在批准的覆繪格。
- **反向對照是必要的，不是加分**。每一項驗收都要問「這個測試在功能壞掉時會不會失敗」。
  例：說明頁的負對照是改一行資料，期望畫面要跟著變、比對要失敗——沒有它，「逐像素差 0」
  可能只是兩邊都算錯同一個樣子。
- **驗收以原版為準，不以自己的內部訊號為準。** 「攔截器有觸發」不代表「那一則文字替換對了」。
- **完整性要有數字**：抽出幾則、翻了幾則、正常路徑觸發了幾則、觸發了卻不在文本檔的有幾則。
  最後一項必須是 0。
- **驗收 oracle 要獨立**：期望值由資料算出來，不是拿截圖當答案。
- 驗**實際打包產物在它自己的執行環境**，不是驗 `workplace/bin/` 的建置輸出。

### 7.1 量測環境會偽造結論

即時性的量測（音訊欠載、節拍、速度）在高負載下量到的是機器狀態不是程式行為。
**開跑前先看 `uptime`**：這台 14 核，`docs/spec/020` 要求開跑時 load < 7。

實測：同一支程式同一個產物，load 6.83 起跑 0 項不符、load 19.55 起跑 2 項不符。

最容易誤導的形狀是「檔案看起來正常但中間有洞」：錄 70 秒的 WAV 只有 54.9 秒，
播得出來，拿去跑保真度指標三項全不過，看起來像合成器壞了。
**解讀音訊結果之前先確認取樣數 ＝ 秒數 × 取樣率。**

### 7.2 判準要對得上音源

`tools/music_compare.py audio` 是給 PC 喇叭（單音方波）用的，會去抽每一段的基頻；
OPL2 是複音 FM 抽不出音高，配對率只有 7%。**OPL2 的判準是 `docs/spec/016` 的三項指標**，
工具是 `tools/opl_wav_compare.py`。

## 8. 前端與執行期

- **F1、F2、F3、F9 是原版自己的功能鍵，不可以攔。** 輔助熱鍵從 F4 起算。
  攔一個鍵之前先確認原版有沒有在用它；**「目前沒有指派」不等於「原版沒用」**——
  沒有掃描碼的鍵原版永遠收不到，也就永遠看不出它其實會用（`docs/re/036`）。
- 按鍵要按住 0.15 秒以上。按下與放開落在同一個 1/60 秒的輪詢裡，Ebiten 會整個漏掉。
- **戰鬥的推進綁 CPU 指令數**，所以 F12 調速只改「一毫秒牆上時間跑幾個 cycle」，
  不碰 CPU 頻率。戰鬥長度不受檔位影響因此是結構上的保證，不是靠偵測補的。
- 存檔寫使用者資料目錄不是 cwd：Linux `$XDG_DATA_HOME`、macOS `Application Support`、
  Windows `%APPDATA%`。AppImage 與 `.app` 的內容是唯讀的。
- 缺關鍵檔時要看得到錯誤：所有平台寫一份錯誤紀錄檔，Windows 另外彈 MessageBox
  （`-H windowsgui` 的程式 `log.Fatal` 之後一個字都看不到）。

## 9. 目錄與文件

| 路徑 | 放什麼 |
|---|---|
| `cmd/psychicwar/` | 前端主程式 |
| `cmd/pwstep/` | 逐步操作（模擬玩家試玩用） |
| `apps/psychicwar/` | 位址、狀態、攔截點、地圖、作弊、戰鬥偵測 |
| `text/` | 譯文（JSON）、譯名表、說明頁、圖檔疊字資料 |
| `font/` | CJK 點陣字子集與字集清單 |
| `docs/spec/` | 規格，標 DRAFT／READY |
| `docs/re/` | 逆向與量測紀錄；`000-overturned-claims.md` 收被推翻的斷言 |
| `docs/goal/` | 分期目標與驗收，只寫目標不記進度 |
| `tools/` | docker 包裝與驗收腳本 |
| `worktrees/` | 相關 repo 的本地 clone（gitignore） |
| `workplace/` | 唯一可寫的研究工作區（gitignore） |
| `dist-all/` | 所有發行產物，每平台只留最新一份（gitignore） |

**正文只寫現況。** 推翻的紀錄集中到 `docs/re/000-overturned-claims.md`，
正文最多留一個指標。教訓寫成可重用的規則，不寫成會過期的事件敘述。

## 10. Docker、Git 與發行

- **分析、建置、轉檔、測試一律走 docker**：`--rm`、`-u "$(id -u):$(id -g)"`、
  `--memory`／`--cpus`／`--pids-limit`、`--log-opt max-size=10m --log-opt max-file=3`、
  預設 `--network none`，原版素材唯讀掛載。
- 掛 `-v` 前確認來源存在且型態正確。來源不存在時 dockerd 會**以 root 建一個目錄頂替**，
  而容器內讀到的是空目錄，不會報錯。
- **只清理自己建立的 container。禁止任何 `docker image/system/volume/builder prune` 或 `rmi`。**
  這台機器有多個專案的 image。
- **這台是共用機器**，常有別的 session 在跑。開工前 `uptime`，`--cpus` 最多用一半。
- **git 身分一律 `wicanr2@gmail.com`**。進任何 repo（含 `worktrees/` 底下每一個）先看
  `git config user.email`，再跑 `git log --format=%ae | sort -u` 看歷史。
- commit message 用繁體中文，不放 session 連結。push、開 PR 前先問。
- **發行包分兩種**：可散布版不含原版素材，打包時做 leak-scan；`-with-data` 版含原版素材，
  **純本機自用，絕不推 git、絕不上傳**。release 只附可散布版。
- **先把所有 commit 做完再打 tag 再打包**。反過來會讓版號帶 `-dirty` 或落後的 commit 數。

## 11. 子代理分工

- 每個子代理的 prompt **寫死邊界**：可寫哪些檔、不可碰哪些檔、禁止 commit／push／動 issue、
  docker 規則、`--cpus` 上限。**沒寫的就等於允許**，agent 會為了「做得完整」自行加碼。
- 拆法：證據探針 → 規格 → 實作 → 驗收。主代理審閱報告後才 commit。
- **子代理回報「順便做了 X」一律先查影響範圍。**
- 並行派工時，**不同 agent 不要碰同一個檔**。主代理 commit 時用明確檔案清單，
  並確認新檔也收進去——收了改過的 `main.go` 卻漏掉它依賴的新檔，HEAD 會編不過。
- 大量對白逐則翻譯先讀 `~/.claude/knowledge-base/workflows/batch-subagent-localization.md`。

## 12. 定案事項

使用者定案的產品決定寫在這一節並附日期，實作時照定案走，**不重新詢問**。
與定案衝突的新需求**先指出衝突再動手**。

| 定案 | 日期 | 內容 |
|---|---|---|
| 版本與執行器 | 2026-09-17 | DOS 英文版跑在 dosgolem 上（§2）|
| 分層歸屬 | 2026-09-17 | 「換一款遊戲還成立嗎」決定程式碼放哪（§3）|
| 全程試玩不做 | 2026-09-18 | 改用抽測：抽幾段正常玩家路徑確認畫面是中文即可 |
| 輸入名字只支援 ASCII | 2026-09-18 | 存檔檔名沿用 `<名>.DAT`，提示文字中文化，不做顯示層轉換 |
| 中文字模用倚天字形發行 | 2026-09-18 | `docs/re/020` §4。README 寫明來源是倚天中文系統 3.53 的點陣子集與下架聯絡方式 |
| 授權 RRSAL-1.0 | — | `rulebook/85`，全文在 `LICENSE` |
| 文本檔的原文欄位進版控 | 2026-09-19 | 原本的規則是公開前要改成從玩家的執行檔重抽，**使用者明示解除**。repo 已公開，原文欄位照留 |
| 發行包直接帶原文欄位 | 2026-09-19 | 轉譯層要靠原文對位，而沒有 `PW.EXE` 遊戲本來就跑不起來（`docs/spec/021` §2）|
| Linux 只出 AppImage | 2026-09-19 | 不出 tar.gz（`docs/spec/021`）|
| 完整版與可散布版都要 | 2026-09-19 | `-with-data` 含原版素材，純本機自用；release 只附可散布版 |
| 速度只加速非戰鬥 | 2026-09-19 | 戰鬥維持原速，不是整體變快（`docs/re/037`）|
| 輔助熱鍵從 F4 起算 | 2026-09-19 | F1／F2／F3 是原版的功能鍵（`docs/re/036`）|
| 說明頁採雙欄版面 | 2026-09-19 | `docs/spec/022` 方案 B |

兩條不是產品決定但同樣不要重問：

- **畫面上要出現的中文一律放進 `text/` 的資料檔**，不寫死在程式裡（§6）。
- **推廣片的配樂只能用 `workplace/dosboxx-audio/title-adlib.wav`**（DOSBox-X 跑原版錄下來的），
  不可以用 `workplace/audio/golem-opl-title.wav`（自寫合成器算的）——`rulebook/93` 鐵則 1。
  影片要對外公開之前，先提醒使用者原版音樂的著作權面向。

## 13. 分期目標

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


## 14. 目前已知的事實

證據、指令與數字見 `docs/re/001-pw-exe-first-look.md`（實跑初探）、`002-pw-exe-function-census.md`（函式普查）、
`003-checkpoints-and-dosboxx-reference.md`（檢查點與 DOSBox-X 參照）、`004-rng-and-determinism.md`（亂數）、
`005-ega-palette-rgb-parity.md`（色盤）、`006-ibm-music-format-and-pc-speaker-parity.md`（PC 喇叭配樂）、`007-opl2-synth-skeleton-and-comparison.md`（OPL2，未通過）、`008-replay-first-battle-and-save-load.md`（重播、存讀檔）、`009-battle-hold-cheat-and-cpu-pacing.md`（按住攻擊、敵人 HP）、`010-cpu-speed-time-base.md`（CPU 速度與時間基準）、`011-observation-addresses.md`（位置、朝向、區域、角色數值位址，地圖檔格式）、`012-opl2-event-layer.md`（OPL2 事件層）、`013-replay-route-samar-to-sivad.md`（重播路線）、`014-print-routine.md`（印字常式）、`015-text-formats.md`（文字格式）、`016-frontend-prototype.md`（前端雛形驗收）、`017-text-coverage.md`（文字覆蓋率）、`018-first-chinese-overlay.md`（第一句中文）、`019-all-paths-overlay-verification.md`（所有路徑疊字、覆蓋率、前端實跑）、`020-font-license-options.md`（字型授權選項，待決）、`021-translation-first-pass.md`（全文翻譯第一版）、`022-simulated-player-playtest.md`（模擬玩家試玩）、`023-pbl-image-format.md`（`.PBL` 圖檔格式）、`024-baked-text-overlay.md`（圖檔內嵌文字）、`025-assist-hotkeys.md`（F1／F2／F10）、`026-adlib-underrun-followup.md`（AdLib 追量）、`027-sampling-playtest-and-fixes.md`（抽測試玩與修正）、`028-copy-protection.md`（防拷）、`029-cheats.md`（作弊）、`030-auto-map.md`（自動地圖）、`031-baked-text-rooms-and-map.md`（房間招牌、道具圖鑑、開場字幕）、`032-opl2-synth-fidelity.md`（OPL2 保真度）、`033-macos-cross-build.md`（macOS 交叉編譯）、
`034-promo-video.md`（推廣片）、`035-windows-cross-build.md`（Windows 交叉編譯）、
`036-original-function-keys.md`（F1／F2／F3 是原版的功能鍵）、`037-battle-detection.md`（戰鬥偵測與調速）；
被推翻的斷言集中在 `000-overturned-claims.md`。

另外兩份不是量測紀錄但同樣是一手資料：`docs/manual-1989.md`（軟體世界代理版說明書整理，
譯名的權威來源）、`docs/walkthrough-1989.md`（1989 年《軟體世界》19 期攻略整理，含與本專案
已驗事實的交叉核對）。

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
| 全文翻譯第一版：1,313 則全部有譯文（保留原文 38）、過長 0、非 Big5 0；譯名表 119 筆（說明書 40、暫譯 79）；字型子集當時 873 字（`font/charset.txt` 是產物，隨譯文增減，現況以它為準）。重播 19 段到 Zellwal 觸發 55 則、不在文本檔 0；前端開機到第一場戰鬥的轉譯紀錄缺譯文／過長／缺字 0，剩下的英文都在圖檔上（#18） | confirmed（數字）；譯文品質未逐則對畫面 |
| 模擬玩家（Sonnet 子代理只看 `pwstep` 截圖）84 步完成開機 → 打贏第一場 → 發射台選目的地，沒用攻略；卡住 1 次（降落平台與發射台的分辨）。找到的問號裁字、音效／音樂、BBS室已修（`docs/re/022`） | confirmed（一條路線） |
| 圖檔文字：`.PBL` ＝ 偏移表＋每張 `[寬÷8][高÷8][16 bytes CGA 表]`＋RLE，4bpp 逐列；貼圖位置 ＝ `CH×4`、`CL×4`。26 檔 537 張，含文字 50 張（`docs/re/023`、`024`） | confirmed |
| 圖檔內嵌文字的中文以「畫面內容比對」觸發（dosgolem `xlate.Watcher`）：面板與狀態欄 6 塊逐像素差 0；F1 說明、F2 中英切換、F10／F11 即時存檔都通過（`docs/re/024`、`025`） | confirmed |
| 防拷：出題 `sub_1695C`、答案表 `cs:6787`（11×7）、正確答案在 `cs:66E5`；**這一版的比對被一個位元組關掉**（`cmp` 之後是無條件跳躍），任何答案都會過（`docs/re/028`） | confirmed |
| 圖檔內嵌文字**收尾**：含文字 50 張已疊 **48 張、125 塊**；沒疊的只剩 `LOGO.PBL`／`KGDLOGO.PBL` 兩張標題美術字（`docs/spec/011` §6 定案保留）。開場字幕條 `OPEN.PBL` #7–#10 在 (24,120/128/136/144)，字比底密的兩行用 `swap_colors` 對調定色（`docs/re/024` §6） | confirmed |
| 發行包（`docs/spec/021`）：Linux tar.gz 與 AppImage 通過「解開產物從別的 cwd 跑」的驗收（畫面差 0、解開處零寫入、存檔落在使用者資料目錄、字型改名報錯）；macOS universal 只過靜態驗收五道，**沒有 Mac 可以實跑**（`docs/re/033`） | confirmed（Linux／AppImage）／結構驗證（macOS） |
| 疊字失效是**逐格**判斷（原版會在一行的一部分上面畫別的東西）；watcher 的疊字例外，整筆失效再由 watcher 重蓋（`docs/re/027`） | confirmed |
| 疊字失效另有**錨定格**規則：定色時壓在原文墨跡上的格子全部失效就整筆移除。中文比原文寬時多出來的格子壓在純色背景上，指紋永遠不變，只靠逐格判斷會留下孤字（`docs/re/031` §7.3） | confirmed（重現前後對照） |
| OPL2 合成器**三項指標全部通過**（`docs/spec/016`：spec 0.9117、env 0.8119、chroma 0.9836，門檻 0.8350／0.7548／0.9603）。根因是**相位單位**：真機一個正弦週期是 **1024** 個索引不是 512（正弦表 `i & 511`、正負半週 `(i >> 9) & 1`），所以 FM 是 `8π` 不是 `4π`、回授是 `2^(FB−5)` 不是 `2^(FB−4)`。兩個錯誤方向相反，逐項開關與逐聲道都測不出來（`docs/re/032`）| confirmed（量測）|
| 即時音訊在**真實裝置**上（PipeWire 的 pulse socket）：AdLib 與 PC 喇叭各 65 秒，欠載 6 次全在開場第 1 秒、第 2 秒之後 0；機器／牆上 0.9978（`docs/re/026` §6）。⚠ 直接掛 `/dev/snd` 開不起來（PipeWire 管著那張卡）、ALSA 的 `null` PCM 不按時序消耗（量到 29 萬次欠載是它自己的行為）| confirmed（量測）|
| 原版**不支援滑鼠**：解壓後 IDA 認得的 93 處 `INT` 指令裡沒有 `33h`。滑鼠是轉譯層自己提供的（`docs/spec/018`）| confirmed |
| probe 的 `-steps` 是**絕對**指令數不是「再跑幾步」；`-hold` 要逗號分隔成一個旗標（Go 的 flag 對重複旗標只留一個）。兩者都會讓重播安靜地變成「什麼都沒按」（`docs/re/033` 待寫，細節在 `docs/spec/019` 與 #14 的留言）| confirmed（正對照：戰鬥按住空白鍵，敵人 HP 38 → 20）|
| **`font/charset.txt` 是烘字的產物，不是可用字的上限**：`tools/font/bake.sh` 掃 `text/*.json` 的譯文收字，再從倚天全字庫烘。寫譯文時不要為了避開字型換詞——缺字補烘一次就有（`docs/re/031` §3.1） | confirmed |
| A2／A3 的迴圈頭也會停在字串結尾的 0；那一次不能推進行追蹤，否則記憶體裡相鄰的下一個字串會被當成同一行（Game Over 三行只有第一行是中文）| confirmed（修正前後對照） |
| **F1／F2／F3 是原版自己的功能鍵**（說明書 p.8：強力加速砲、與戰友交談、向後一步脫離戰鬥；Enter 是防護盾）。原本既被前端攔下、又沒有掃描碼可送，按了沒反應也不報錯。輔助熱鍵改從 F4 起算，F9 也留給原版（`docs/re/036`） | confirmed（端到端對照：同狀態起兩次，按 F3 差 27,438 像素、不按差 657）|
| 戰鬥的進出可用執行位址掛鉤判斷：`0161:47D4` 進入、`0161:47AD` 返回，剛好包住戰鬥主迴圈 `sub_14CE4`。**敵人 HP `0x509C` 不能當判斷點**——用 F3 逃走時它完全不會被寫（`docs/re/037`）| confirmed（三個情境各觸發恰好一次）|
| F12 調速只改「一毫秒牆上時間跑幾個 cycle」，不碰 `SetDOSBoxCycles`（那會改 IRQ0 間隔與戰鬥結果）。所以戰鬥長度不受檔位影響是**結構上的保證**：同一場戰鬥 1× 與 3× 的結束指令數完全相同（103,945,892）| confirmed（量測）|
| 發行包三個平台（AppImage、macOS universal、Windows zip）都產得出來，各有可散布版與 `-with-data` 版。AppImage 解開後從別的 cwd 跑與開發建置逐像素差 0；Windows 在 wine 9.0 下同狀態畫面差 0。**macOS 與 Windows 都沒有在真機上跑過**（`docs/re/033`、`035`，#44）| confirmed（Linux）／結構與 wine 驗證（macOS、Windows）|
| 發行包的 AdLib 音訊在真實裝置上：欠載 9 次全在開場第 1 秒、之後 0，機器／牆上 0.9921（`docs/re/026` §7）。⚠ 這類量測**開跑時** load 要 < 7；同一支程式 load 6.83 起跑 0 項不符、load 19.55 起跑 2 項不符 | confirmed（量測）|


## 15. 工作追蹤與工具

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
| `tools/opl_wav_compare.py <a.wav> <b.wav> [--segments] [--drift]` | OPL2 合成器的保真度（`docs/spec/016`）：頻譜、包絡、chroma 三個指標。整段一個數字看不出「哪一段不像」——`--segments` 逐 5 秒印，`--drift` 看位移是不是隨時間漂 |
| probe 的 `-opl-disable`／`-opl-only`／`-opl-except` | OPL2 診斷：關掉單一功能、只混一個聲道、排除一個聲道。**一次只改一個變因**；`-opl-except` 比 `-opl-only` 有用，因為參照那邊是混音拆不開 |
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
| `tools/font/bake.sh` | 烘製 `font/cjk24.golemfnt`、`cjk16.golemfnt`（倚天＋Noto 補字；來源放 `workplace/font-src/`）；兩套字型都沒有的字結束碼 1。**改完任何譯文都要重跑**——沒重烘不會報錯，新字在畫面上是靜默地畫不出來（2026-09-19 發現 `charset.txt` 還留著早就沒人用的字、卻缺 `help.json` 裡的招／募／情）|
| `tools/overlay_run.sh [情境…]`＋`tools/overlay_cases.json`＋`tools/overlay_check.py` | 疊字逐像素驗收：dosgolem 的 `cmd/step` 出原版參照、`pwstep` 出中文截圖；`PSYCHICWAR_WITHOUT=1` 反向對照（`docs/spec/009` §6） |
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
| `tools/package.sh appimage\|macos\|promo\|all` | 發行包（`docs/spec/021`）。**產物一律在 `dist-all/`**（gitignore），每平台只留最新一份，慣例見 `docs/DEV-SETUP.md`。`PSYCHICWAR_WITH_DATA=1` 另出含原版素材的本機版（**絕不推 git、絕不上傳**）|
| `tools/package-check.sh <產物>` | 發行包驗收：解開後從**別的 cwd** 跑，比畫面、比解開處有沒有被寫入、字型改名的反向對照 |
| `tools/macos-pack.sh`、`tools/macos-verify.sh` | macOS universal（osxcross 兩弧＋lipo）與五道靜態驗收（`docs/re/033`） |
| `tools/windows-pack.sh`、`tools/windows-verify.sh` | Windows（**不需要 mingw**，Ebiten 走 purego）與靜態五道＋wine 實跑（`docs/re/035`）。⚠ wine 8.0 跑不動 Go 1.22 以後編的 `.exe`，要 9.0 |
| `tools/frontend-speed-check.sh`、`tools/speed_report.py` | F12 調速驗收（`docs/spec/023`）：非戰鬥要變快、**戰鬥長度不受檔位影響** |
| `tools/audio-device-check.sh [adlib\|speaker] [秒] [產物]` | 真實音訊裝置上的欠載（`docs/spec/020`）。第三個參數給 AppImage 就驗發行包。⚠ **開跑時 load 要 < 7**，否則量到的是機器狀態不是程式行為 |
| `tools/appimagetool.sh`、`tools/appicon.py` | AppDir → AppImage（type2 runtime 串 squashfs）；自製圖示（不用原版 Logo） |
| `tools/video.sh`、`tools/promo/make.sh`、`tools/promo/theme.sh` | 推廣片合成（ffmpeg／ImageMagick，skill `game-promo-video-ffmpeg`）。⚠ 配樂只能用 `workplace/dosboxx-audio/title-adlib.wav`（DOSBox-X 錄的原版輸出），**不可以用自寫合成器的 `workplace/audio/golem-opl-title.wav`**（`rulebook/93`） |
| `tools/l10n_batches.py prep\|merge\|sweep`、`tools/l10n_check.py` | 分批翻譯、合併核對、一致性掃描；譯者自我檢查（`docs/re/021`） |
| `tools/text_extract.py lint` | 譯文行寬與 Big5 檢查（`docs/spec/009` §3） |
| `tools/text_extract.py build\|check\|stats <原版目錄>` | 產生／驗證 `text/`（`docs/spec/007`）；`menu\|enmy\|code` 是單檔除錯輸出 |


## 16. 按需載入

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
| 推廣片的素材來源 | `rulebook/93`（配樂必須用原版實際素材，不可自產）|
| 選授權條款、寫 LICENSE | `rulebook/85`（已定案 RRSAL-1.0，不重問）|
| 沒有 Mac 要出 macOS 版 | skill `osxcross-macos-cross-build` |
| 查詢回空、要下「不存在」結論前 | `~/diagnosis-notes/docs/02-query-returned-empty/` |
| 長時間自動化實驗、批次回報「沒觸發」 | `~/diagnosis-notes/docs/03-silence-is-not-success/` |

路由表在 `~/.claude/rules/00-rules-index.md`；復古遊戲類的細分規則收在
skill `re-retro-cht-rulebook`，命中就直接 invoke。
