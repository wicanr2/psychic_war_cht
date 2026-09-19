# Worklist

> 由 `tools/worklist.py render` 從 `docs/worklist.json` 產生，**不要手改**。
> 條目做完就從 JSON 移走或改 verify，不是在這裡打勾。

## M0：探勘與基線：證據、狀態檔、參照畫面

| issue | id | 標題 | label | 前置 | 完成訊號 |
|---|---|---|---|---|---|

## M1：原版在 golem 上完整可跑：畫面、音樂、決定性

| issue | id | 標題 | label | 前置 | 完成訊號 |
|---|---|---|---|---|---|

## M2：可遊玩前端：視窗、鍵盤、滑鼠、音訊、節拍

| issue | id | 標題 | label | 前置 | 完成訊號 |
|---|---|---|---|---|---|
| #9 | `realtime-pacing` | 牆上時間節拍：讓 72 Hz 的遊戲以原速執行 | golem-upstream, frontend | — | manual |
| #40 | `adjustable-speed` | 遊戲內可調執行速度，預設考慮調快 | frontend | — | manual |

## M3：文字攔截與文本抽取

| issue | id | 標題 | label | 前置 | 完成訊號 |
|---|---|---|---|---|---|

## M4：中文繪製與全文翻譯

| issue | id | 標題 | label | 前置 | 完成訊號 |
|---|---|---|---|---|---|
| #27 | `playtest-cht` | 中文版正常玩家路徑試玩 | verify, text | #24、#25 | manual |

## M5：輔助功能：F1／F2／F3／F10、防拷、作弊

| issue | id | 標題 | label | 前置 | 完成訊號 |
|---|---|---|---|---|---|
| #42 | `hotkey-conflict` | F2／F3 熱鍵與原版功能衝突 | frontend | — | manual |
| #41 | `help-page-design` | F1 說明頁的視覺設計 | frontend | — | manual |

## M6：主題、打包、授權與發行

| issue | id | 標題 | label | 前置 | 完成訊號 |
|---|---|---|---|---|---|
| #34 | `theme` | 主題替換 | graphics | #20 | manual |
| #39 | `windows-package` | Windows 發行包 | release | — | absent |
| #39 | `macos-real-run` | macOS 發行包的真機驗收 | release, verify | — | manual |

## 不做（使用者定案）

| issue | id | 標題 | 理由 | 定案日期 |
|---|---|---|---|---|
| #8 | `full-run-no-gaps` | 從開頭跑到結局，沒有未實作的服務 | 使用者定案：不執行。MVP 改以抽測驗收（幾段正常玩家路徑畫面上是中文），不做從開頭到結局的完整跑通。 | 2026-09-18 |

## 已完成

| issue | id | 標題 | 證據 | 日期 |
|---|---|---|---|---|
| #1 | `ida-baseline` | 建立 PW.EXE／LOGO.EXE 的 IDA 資料庫並做函式普查 | `docs/re/002-pw-exe-function-census.md` | 2026-09-17 |
| #2 | `state-checkpoints` | 固定的狀態檔檢查點：標題、防拷、輸入名字、第一個可操作畫面 | `docs/re/003-checkpoints-and-dosboxx-reference.md` §2；`tools/states.sh --check` | 2026-09-17 |
| #3 | `dosbox-reference-frames` | DOSBox-X 參照畫面：同一個畫面的正確顏色與版面 | `docs/re/003-checkpoints-and-dosboxx-reference.md` §3；`tools/dosboxx-ref.sh`、`tools/frame_compare.py` | 2026-09-17 |
| #4 | `ega-palette-output` | EGA mode 0Dh 的畫面輸出沒有套用遊戲設定的色盤 | `docs/re/005-ega-palette-rgb-parity.md`；dosgolem 分支 `psychic-war/m1-ega-palette` `509a639`（未推上游） | 2026-09-17 |
| #7 | `determinism-rng` | 亂數來源與重播決定性 | `docs/re/004-rng-and-determinism.md`；`tools/determinism.sh` | 2026-09-17 |
| #6 | `pc-speaker-audio` | PC 喇叭音樂路徑（`.IBM`）驗證 | `docs/re/006-ibm-music-format-and-pc-speaker-parity.md`；`docs/spec/001`；`tools/music_compare.py`；dosgolem 分支 `psychic-war/m1-pit-tone` `6cf8a1b`（未推上游） | 2026-09-17 |
| #10 | `frontend-window` | 可遊玩前端：視窗顯示 320×200 EGA 畫面（整數倍放大） | `docs/spec/006`、`docs/re/016` §1–2；`cmd/psychicwar`；`03-protection`、`07-first-play` 截圖與檢查點不同像素 0 | 2026-09-17 |
| #11 | `keyboard-input` | 鍵盤輸入：前端按鍵轉成 IRQ1 掃描碼 | `docs/re/016` §2.2；`apps/psychicwar/keymap.go`；`tools/frontend-playthrough.sh`（Xvfb＋xdotool：防拷、SELECT、名字、前進、按住空白鍵、轉向、Esc；F1 攔下畫面不變） | 2026-09-17 |
| #15 | `print-routine` | 找出印字常式：FONT.BIN 字模與固定寬度文字框 | `docs/re/014-print-routine.md`（單字元 `sub_16629`、三支字串迴圈、字模 `FONT.BIN`、攔截點候選）；`tools/print_trace.py`。其餘 6 個單字元呼叫點由第 5 輪追蹤 | 2026-09-17 |
| #16 | `text-formats` | 文字資料格式：I_MENU*.BIN、I_ENMY*.BIN、CODE*.BIN、PW.EXE 字串表 | `docs/spec/007`（READY：I_MENU、I_ENMY、I_MAP 地點名、CODE 81h＝sub_167B2 內嵌字串、PW_UNP.EXE 位址表；PW.EXE／PW_UNP.EXE SHA-256 與逐項位元組簽章）；`docs/re/014`、`015` | 2026-09-17 |
| #17 | `text-extract` | 文本抽取工具：從玩家自備的原版產生文本檔 | `text/*.json` 1,389 則（要翻 1,313）；`tools/text_extract.py build／check／stats`；`tools/test_text_extract.py` 7 項通過（含雜湊不符停止、缺原版列出缺檔） | 2026-09-17 |
| #19 | `runtime-text-coverage` | 文字覆蓋率量測：實跑觸發但不在文本檔的訊息數 | `tools/text_coverage.sh`、`tools/text_coverage.py`：重播 18 段 要翻 1,313、已翻 0→11、觸發 52、不在文本檔 0；反向對照拿掉一則變 2（`docs/re/017`、`tools/test_text_coverage.py`） | 2026-09-17 |
| #20 | `hires-overlay-canvas` | 放大畫布＋疊字層：原版畫面放大後在上面畫中文 | `docs/spec/009`（READY）＋dosgolem 規格 `202-translation-overlay`（`xlate` 套件）：A1／A2／A3／小字型四條印字路徑。`pwstep` 8 情境 13 行逐像素差 0、反向對照差 0；前端 Xvfb `wall`（含訊息框捲動）、`match` 5 行差 0（`docs/re/019`） | 2026-09-17 |
| #22 | `layout-rules` | 固定寬度文字框的中文排版規則 | `docs/spec/009` §3：各種類行寬（訊息 16＋15、選項 10、區塊每行 20、其餘依原文可印字數）、一格一字、透明格、保留原文；`tools/text_extract.py lint` 列出過長譯文，1,313 則過長 0（`docs/re/021`）。選單游標與反白沿用原版像素，不另排 | 2026-09-17 |
| #23 | `glossary` | 譯名表：人名、地名、超能力名稱 | `text/glossary.json` 119 筆：說明書對照 40（記頁碼）、自訂 79（標 provisional，敵人名稱全部）；翻譯指令與 `tools/l10n_batches.py sweep` 都讀它（`docs/re/021`） | 2026-09-17 |
| #24 | `translation-pass` | 全文翻譯 | 要翻 1,313、已翻 1,313（保留原文 38）；10 批子代理＋合併核對＋一致性修正；lint 過長 0、非 Big5 0；字型子集 873 字（`docs/re/021`） | 2026-09-17 |
| #21 | `cjk-font` | CJK 點陣字：依譯文烘製子集 | `tools/font/bake.sh` 走 docker 烘 `font/cjk24.golemfnt`、`cjk16.golemfnt`，873 字（24 點倚天 815＋Noto 58、16 點倚天 825＋Noto 48）；兩套字型都沒有的字結束碼 1、不寫輸出檔（反向對照 U+1F600）；實跑缺字 0（`docs/re/019` §5、`docs/re/021`）。字型授權：使用者定案 2026-09-18 發行也用倚天字形（`docs/re/020` §4），README 的字模來源說明由 #36 追蹤 | 2026-09-18 |
| #18 | `baked-text-graphics` | 圖檔內嵌文字：PBL 格式與含文字的圖清冊 | `docs/spec/010`（READY）＋`tools/pbl.py`：26 個檔 537 張圖全部解得開，`SCREEN.PBL` 三張在實跑畫面逐像素相同、反向對照失敗（`docs/re/023`）。清冊 `text/baked-inventory.json`：537 筆、含文字 50 張 143 則（`docs/re/024` §1、§5） | 2026-09-18 |
| #29 | `f1-help` | F1 說明頁 | `docs/spec/012`（READY）＋`text/help.json`：任何畫面 F1 開關，說明頁與期望值逐像素差 0（22 行），關閉後與開啟前差 0（`docs/re/025`）。內容含「發射台才能選目的地」（docs/re/022 的卡住點） | 2026-09-18 |
| #30 | `f2-language` | F2 切換語言（中文／英文原文） | F2 切到英文：畫面與原版放大 3 倍差 0；切回中文與切換前差 0；轉譯層照常運作只是不畫（`docs/spec/012` §2、`docs/re/025`） | 2026-09-18 |
| #31 | `f10-quicksave` | F10 即時存檔／讀檔 | F10 存、F11 讀：狀態檔＋疊字層快照＋`quick.json`（golem 狀態格式版本、PW.EXE 與 text/ 的 SHA-256、語言）。讀回後畫面與觀測變數和存檔時相同；雜湊被改過時拒絕且畫面不動（`docs/re/025`） | 2026-09-18 |
| #28 | `copy-protection` | 防拷：翻譯、F1 查表、略過選項 | `docs/spec/013`（READY）＋`docs/re/028`：出題與答案表讀出來（11×7，答案在 `cs:66E5`）；**這一版的比對被一個位元組關掉**，任何答案都會過。三種用法：畫面中文化（`docs/spec/009` 的疊字）、F1 查表（從玩家自己的執行檔即時讀，掛在輸入常式上判斷狀態）、略過（不必做，空白就會過）。反向對照：迷宮畫面按 F1 沒有那一行（差 0） | 2026-09-18 |
| #33 | `cheats` | 作弊功能 | `docs/spec/014`（READY）＋`docs/re/029`：`-cheat` 打開 F5（HP 與能量補滿）、F6（敵人剩 1 點），一律寫記憶體、位址表在 `apps/psychicwar/cheat.go`。實跑 HP 23→40、能量 1→30、敵人 38→1；不給 `-cheat` 時完全沒變（反向對照） | 2026-09-18 |
| #32 | `f3-map` | F3 自動地圖 | `docs/spec/015`（READY）＋`docs/re/030`：走過才記的自動地圖（不解 I_MAP），F3 開關。實跑目前格子與存檔座標相同、走過的 3 格正確、沒走過的 5 格是背景色、開關前後記憶體相同 | 2026-09-18 |
| #12 | `mouse-input` | 滑鼠：確認原版是否支援，並在前端提供點選操作 | docs/spec/018、tools/frontend-mouse-check.sh：點「前進」與按 Up 之後的觀測變數逐位元組相同、點熱區外與不點相同 | 2026-09-18 |
| #14 | `input-record-replay` | 輸入錄放：以指令數記錄按鍵，可重播重現 | docs/spec/019、tools/record-replay-check.sh：錄 8 筆事件，重播後觀測變數逐位元組相同，反向對照不同 | 2026-09-18 |
| #26 | `name-entry` | 輸入名字：原版只收 ASCII，中文版怎麼處理 | docs/spec/017、tools/frontend-name-check.sh：名字欄位 play3 正確、送非 ASCII 之後逐位元組相同、NonASCII 單元測試通過 | 2026-09-18 |
| #13 | `audio-output` | 即時音訊輸出（OPL2 與 PC 喇叭） | docs/spec/020、docs/re/026 §6：真實裝置（PipeWire 的 pulse socket）上 AdLib 與 PC 喇叭各 65 秒，欠載都是 6 次全在開場第 1 秒、第 2 秒之後 0；機器／牆上 0.9978（反向對照） | 2026-09-18 |
| #5 | `opl2-synth` | OPL2（AdLib）合成：`.MID` 音樂路徑目前只有暫存器紀錄 | docs/spec/016 三項全過：spec 0.9117、env 0.8119、chroma 0.9836（門檻 0.8350／0.7548／0.9603）。根因是相位單位——真機一個正弦週期 1024 個索引不是 512，FM 少一半、回授多一倍（docs/re/032）。驗收的第二段「迷宮曲」依使用者定案 2026-09-19 不做 | 2026-09-19 |
| #25 | `baked-text-replacement` | 圖檔內嵌文字的中文替換圖層 | 含文字 50 張已疊 48 張、125 塊（tools/baked_report.py）；沒疊的只有 LOGO.PBL 與 KGDLOGO.PBL 兩張標題美術字，docs/spec/011 §6 定案保留。lint 0 問題、合成畫面測試 125 筆全過。開場字幕條 OPEN.PBL #7–#10 的畫面座標由實跑截圖逐像素定出（差 0），四行中文實跑觸發並蓋上；字比底密的兩行用 text/baked.json 的 swap_colors 對調定色（dosgolem xlate，規格 202 §2.3）。見 docs/re/024 §6 | 2026-09-19 |
| #36 | `license-readme` | LICENSE（RRSAL-1.0）與 README | LICENSE 是 RRSAL-1.0 全文；README 的授權段摘要第 2、6、12 條，寫明授權不涵蓋原版素材、中文字模來自倚天中文系統 3.53 的點陣子集與下架聯絡方式（wicanr2@gmail.com），並列出 PW.EXE 與 LOGO.EXE 的 SHA-256（取得管道刻意不寫） | 2026-09-19 |
| #37 | `readme` | README.md：遊戲介紹、中文截圖、怎麼玩 | README.md 173 行，五張中文截圖在 docs/images/（由 pwstep 以當前 HEAD 重跑產生）。涵蓋遊戲介紹、中文化做法、完成度數字（出處全部連到 docs/re/）、抽測試玩三次、輔助功能表、怎麼跑、發行包狀態、文件索引、授權與字型來源。截圖只含遊戲畫面，原版檔案沒有進版控 | 2026-09-19 |
| #35 | `cross-platform-package` | 跨平台打包：Linux／Windows／macOS | AppImage 與 macOS universal 都產出並集中在 dist-all/（docs/spec/021、docs/DEV-SETUP.md）。AppImage：解開產物從別的 cwd 執行、畫面與 repo 建置逐像素差 0、解開處零寫入、存檔落在 XDG 目錄、字型改名報錯。-with-data 變體在沒有掛任何原版目錄的容器裡、不給 -orig 跑到標題畫面；反向對照是可散布版印用法、結束碼 2。macOS 只過靜態驗收五道（docs/re/033），沒有 Mac 可實跑，那一項移到 macos-real-run。Windows 移到 #39 | 2026-09-19 |
| #38 | `walkthrough-1989` | 整理當年《軟體世界》19 期攻略（29–38 頁）成 markdown | docs/walkthrough-1989.md 381 行。十頁掃描逐頁判讀，事實性資料表格化不逐字轉錄。交叉核對：基地名表 cs:674F 的七筆順序、三組密碼逐字（含驚嘆號數量）、八張地圖與 CODEH.BIN 的道具名表都吻合；MELSER 結局流程對上 I_MENU11／I_MAP11，可推定區域 11 ＝ MELSER。找到的衝突拆成 #42（F2／F3 熱鍵）。未核實項目逐條標明 | 2026-09-19 |
