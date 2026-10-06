# 接手現況：中文化與HD化

核對日期：2026-10-06 19:09（台灣時間）。主repo基準7deffb9，dosgolem31242a9；改動尚未提交。
未完成權威：[docs/worklist.json](docs/worklist.json)。歷程：[WORKLOG.md](WORKLOG.md)。

## 最新結果

最後12個敵人圖號已接入，敵人來源覆蓋360/360，含180身體及180小圖。ENEMY02 #12／#13共用相同原圖及定稿PNG；ENEMY08最後三身體保持24×24，未知尾段的三個8×8格保留原版。正常GUI已驗30/360，沒有把受控來源算成全動畫完成。

依使用者順序續接ALLY #3–#11道具肖像，盟友來源接入12/31。新增九張只限定(128,8)的24×32完整肖像；其正常玩家選圖、隊伍位置及GUI仍待驗。其餘19圖號為四張局部圖形與15小圖，不猜用途或貼圖模式。敵人360／ALLY31美術皆已定稿，不重生或重問。

現行本機主題theme-enemy360-ally12-effects-maze-B-v1-20261006，5,654PBL＋78builtin_masks＋256MAZE／394PNG。沿原版位置、全域8×8、B透光與人物在後／框線在前。60組冷載及180合成場景通過；58組217變數、別名組216、短圖已知區215變數滿秩。60敵人及九ALLY獨立全圖面RGBA差0，省略負對照有效。

八次原版受控身體切換、40個短圖中途模型、錯來源／別名PNG／尾段／冷載／HD開關通過。中途完整前圖不再覆寫入口已確認的待完成目標。敏頓4,166邊界／2,083轉換回歸通過。合併主題六正常保存點12次44機器欄位及DOS相同，12舊圖面及合成均相同。

新版普通Space GUI36視窗中35份有已驗效果格，共1,975格，六類敏頓效果可見，F10保存及自然退出通過。這是既有正常保存點接續與有限格核對，不是整屏同幀、從開機、全部新圖號GUI或DAT驗收。

## 目前狀態表

| 項目 | 目前狀態 | 證據與限制 |
|---|---|---|
| 中文化 | 1,313則、48圖檔／125塊及既有抽測完成 | 缺文本0；兩張標題美術字保留；未全程試玩 |
| 美術定稿 | 敵人360/360、ALLY31/31 | 身份、尺寸與SHA凍結，不重問美術 |
| 現行本機主題 | 5,654PBL＋78遮罩＋256MAZE／394PNG | theme-enemy360-ally12-effects-maze-B-v1-20261006；僅本機 |
| 敵人來源／接入 | 360/360 | 180身體＋180小圖；別名共用來源、短圖保持原尺寸 |
| 敵人正常GUI | 已驗30/360 | 原27身體及敏頓三小圖；受控／合成不增加計數 |
| 原版小圖来源／動作 | 180來源、120受控組已驗 | 23,040 EGA bytes；1,440貼圖、480不貼圖；沿既有證據 |
| 工作區選組／冷載 | 60唯一來源組 | 唯讀1175:AF12；180合成場景、13,019登記變數、13,017可見獨立變數 |
| 短圖與中途 | 下方三格永遠保留原版 | 尾96 bytes語意未知；40受控中途模型，不冒稱正常中途 |
| 共用效果 | BEAM #0–#2、FIGHT #0–#3及78遮罩位置 | ALLY #0基底；小圖x40–264、y160／168；域外保持原版 |
| 敏頓回歸 | 4,166邊界、2,083轉換及負對照通過 | 新合併主題；核心清單沒有跳過 |
| 六正常保存點 | 12次44欄位／DOS與12舊畫面相同 | 同起點及100,000步接續；不當新圖號正常呈現 |
| 新版普通GUI | 35/36畫面有已驗效果格，共1,975格 | 六類來源、F10及自然退出；原版同幀與DAT另驗 |
| ALLY來源／接入 | 12/31圖號 | #0–#2既有位置、#3–#11僅道具肖像；九張獨立RGBA差0 |
| ALLY正常玩家／GUI | 仍只驗#0–#2 | #3–#11隊伍位置、正常選圖及GUI待驗 |
| 剩餘ALLY來源 | 19圖號未接入 | #12–#15局部24×32、#16–#30小圖16×16；美術已定稿 |
| 一般套件 | 24通過、23專用fixture跳過 | 九ALLY／60敵人／八原版身體／40中途／兩敏頓清單另明示執行通過 |
| DAT／動畫／效能 | 新批完整驗收未完成 | 前43筆DAT限定驗證保持；F10不能代替DAT |
| 房間／OVER／迷宮 | 既有來源與B v9保持 | ROOM0限定來源、OVER #0及單張256格MAZE；未見場景另驗 |
| 交付／平台 | 完整HD封包未完成 | 權利、封包與平台待驗；#34 OPEN，#44不改 |

## 證據與執行入口

- 規格：[024 HD主題](docs/spec/024-hd-theme.md) §1.47–§1.52、§2.0；研究：[038 HD可行性](docs/re/038-hd-theme-feasibility.md) §194–§199。
- 現行主題：workplace/hd/theme-enemy360-ally12-effects-maze-B-v1-20261006/manifest.json及verification-receipt.json。啟動參數：`-theme workplace/hd/theme-enemy360-ally12-effects-maze-B-v1-20261006`。來源及PNG僅本機，未納入發行包。
- 本輪研究根：workplace/ida/hd-ally-recruit-20261004/。敵人來源explore-last-enemy-groups-v1-20261006.py、last-enemy-groups-source-review-v1-20261006.json；八原版動作last-enemy-motion-v1-20261006.go／.bin／-build.json及同名目錄；獨立核對verify-last-enemy-motion-v1-20261006.py。
- 敵人完整模型enemy360-ally12-plan-v2-20261006.json；正式測試[battle_profiles_test.go](apps/psychicwar/theme/battle_profiles_test.go)、[enemy_last_groups_test.go](apps/psychicwar/theme/enemy_last_groups_test.go)。明示PSYCHICWAR_TEST_ORIG、PSYCHICWAR_SMALL_PLAN、PSYCHICWAR_ENEMY360_PLAN、PSYCHICWAR_LAST_ENEMY_PROOF及輸出；合併60圖面與兩敏頓日志enemy360-ally12-validation-v2-20261006/，獨立enemy360-ally12-render-independent-v1-20261006.json。
- ALLY共用來源ally-portraits-ida-v1-20261006.json、ally-portraits-controlled-v1-20261006.go／.bin／-build.json及同名目錄execution.json；獨立ally-portraits-source-independent-v1-20261006.json。正式測試[ally_portraits_test.go](apps/psychicwar/theme/ally_portraits_test.go)，明示PSYCHICWAR_ALLY12_PROOF／OUT。九圖面ally12-validation-v2-20261006/，獨立ally12-render-independent-v2-20261006.json。
- 實際建置來源enemy360-ally12-tests-v2／v3-20261006-build.json；一般套件enemy360-ally12-suite-v3-20261006.json。六正常保存點enemy360-ally12-normal-v1-20261006.go／.bin／-build.json及同名目錄runtime.json與12份-machine.json。
- 新前端enemy360-ally12-frontend-v1-20261006.bin／-build.json，349實際來源。普通GUI入口enemy360-ally12-gui-v2-20261006.sh／.py及同名目錄36視窗、record.json、final.state；獨立enemy360-ally12-gui-cells-independent-v1-20261006.json。模板來自前輪原版邊界，不作新截圖時間戳。
- 定稿清冊workplace/hd/{enemy,ally}-art-final-v1-20261006/art-index.json；舊原版／391美術與敏頓收據保持。本輪保全入口preserve-enemy360-ally12-v1-20261006.py、enemy360-ally12-source-snapshot-v1-20261006.tar.gz、enemy360-ally12-final-audit-v1-20261006.json；原版、PNG、提示、state與archive只留本機。

## 下一步

1. 接ALLY #12–#30前核對四局部圖形及15小圖的原版來源、位置、模式與B合成。
2. 補ALLY #3–#11的正常隊伍／道具GUI、敵人與其他效果／基底正常抽樣。
3. 完成真實中途／DAT、動畫、效能、權利與封包。全部達024 §6才關#34。

沒有新commit／push／PR／tag／發行；完整HD尚未完成。
