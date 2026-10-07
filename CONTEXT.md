# 接手現況：中文化與HD化

核對日期：2026-10-07 17:14（台灣時間）。主repo建置基準e8a401e，dosgolem31242a9；本批ALLY實作與驗收納入本機提交。未完成權威：[docs/worklist.json](docs/worklist.json)。歷程：[WORKLOG.md](WORKLOG.md)。

## 最新結果

敵人來源接入360/360；ALLY來源接入31/31。新增十五張小圖與四張裝備，沿024 §1.53–§1.54 READY。小圖限定31個原版位置；裝備先與人物合成，再沿四個隊伍原位顯示。原版來源、RAM與存檔保持。敵人360／ALLY31美術均已定稿。

現行本機主題theme-enemy360-ally31-effects-maze-B-v1-20261006，5,746PBL＋78builtin_masks＋256MAZE／413PNG。31份小圖、192裝備與48人物原位的獨立全RGBA差0；144合成B場景、48獨立B圖面及60唯一隊伍工作源通過。六正常保存點12次44欄位與DOS相同；11舊RGBA相同，格斯丁提的一份真實中途用實際兩隊員基底重建，156格吻合、4格回退。

新正式前端的隊伍／道具24視窗、肖像切換、撤圖及F10／F11通過。普通Space戰鬥36視窗中35份、1,739效果格符合獨立B模板。真正DAT存讀後24視窗、66角色區全RGBA差0。存檔、讀檔、戰鬥三條實際GUI鍵序，各44機器欄位及完整DOS與原版精確重播相同；512-byte DAT相同，載回玩家52 bytes相同。正常GUI仍限既有保存點，敵人30/360、ALLY #0–#2保持。

效能三模式已量測，起跑load 6.26／4.29／4.74。未載入、載入關閉、開啟的Frame均值1.543／3.207／3.485ms，繪圖頻率40.58／53.23／38.14FPS。限兩CPU容器、軟體OpenGL、null音訊的正常道具畫面，沒有設定效能門檻。

## 目前狀態表

| 項目 | 目前狀態 | 證據與限制 |
|---|---|---|
| 中文化 | 1,313則、48圖檔／125塊及既有抽測完成 | 缺文本0；兩張標題美術字保留；未全程試玩 |
| 美術定稿 | 敵人360/360、ALLY31/31 | 身份、尺寸與SHA凍結，不重問美術 |
| 現行本機主題 | 5,746PBL＋78遮罩＋256MAZE／413PNG | theme-enemy360-ally31-effects-maze-B-v1-20261006；僅本機 |
| 敵人來源／接入 | 360/360 | 180身體＋180小圖；ENEMY02別名共用；ENEMY08未知尾三格保持原版 |
| 敵人正常GUI | 已驗30/360 | 原27身體及敏頓三小圖；受控／合成不增加計數 |
| ALLY來源／接入 | 31/31圖號 | 12人物、4裝備、15小圖；限READY原位與模式 |
| ALLY正常玩家／GUI | 仍只驗#0–#2 | 其他人物、小圖及正常選裝／招募待驗 |
| 裝備與隊伍基底 | 60唯一來源、四隊伍原位 | 192原版合成、48原位COPY與240獨立RGBA；正常選裝另驗 |
| 敵人冷載與中途 | 60組／180合成場景 | 八受控原版轉換、40短圖中途模型；全域8×8及未知三格保持 |
| 敏頓回歸 | 4,166邊界、2,083轉換及負對照通過 | ALLY31主題，兩核心清單均明示執行 |
| 六正常保存點 | 12次44欄位／DOS與獨立隊伍B通過 | 11舊RGBA相同；1真實中途156格吻合、4格原版回退 |
| 新版普通戰鬥GUI | 35/36畫面、1,739有限效果格 | 六類來源與F10；模板步數不是截圖時戳，整屏同幀另驗 |
| 隊伍／道具GUI | 24視窗、F10／F11與撤圖通過 | 新前端；#0／#2兩原位，保持既有正常GUI範圍 |
| 真正DAT | 新主題512 bytes存讀及原版對照通過 | 載回玩家52 bytes及24視窗／66角色區；其他角色／場景另驗 |
| GUI原版狀態不變 | 三條精確錄放各44欄位及DOS相同 | 同初始state／seed及原版鍵步數，沒有重擲 |
| 效能 | 三模式有數字與負對照 | 起跑load均<7；道具畫面、兩CPU／軟體OpenGL；戰鬥與封包另驗 |
| 一般套件 | 24通過、25專用fixture跳過 | 小圖31原位、裝備192、人物48、60敵人與兩敏頓清單另明示通過 |
| 房間／OVER／迷宮 | 既有來源與B v9保持 | ROOM0限定來源、OVER #0及單張256格MAZE；其他場景另驗 |
| 完整HD／交付 | 未完成 | 其他效果、圖像／動畫、正常抽樣、權利與封包待驗；#34 OPEN、#44不改 |

## 證據與執行入口

- 規格：[024 HD主題](docs/spec/024-hd-theme.md) §1.47–§1.54、§2.0；本批研究：[038](docs/re/038-hd-theme-feasibility.md) §200–§201。前批敵人入口保留§194–§199。
- 現行主題：workplace/hd/theme-enemy360-ally31-effects-maze-B-v1-20261006/。啟動參數：`-theme workplace/hd/theme-enemy360-ally31-effects-maze-B-v1-20261006`。定稿清冊workplace/hd/{enemy,ally}-art-final-v1-20261006/art-index.json。
- 以下入口均在workplace/ida/hd-ally-recruit-20261004/。原版來源：ally-remaining-ida-v7、ally-remaining-controlled-v2、ally-small-source-independent-v1、ally-equipment-controlled-v4、ally-equipment-source-independent-v2，日期20261006。正式實作／測試：[ally_equipment.go](apps/psychicwar/theme/ally_equipment.go)、[ally_equipment_test.go](apps/psychicwar/theme/ally_equipment_test.go)、[ally_small_test.go](apps/psychicwar/theme/ally_small_test.go)。
- 既有驗證：ally31-validation-v1、ally31-small-validation-v1、enemy360-ally31-regression-v2及enemy360-ally31-normal-v2，日期20261006。獨立收據ally31-render-independent-v2、ally31-small-render-independent-v1、enemy360-ally31-render-independent-v1、ally31-normal-independent-v2。跳過不計通過。
- 接手核對：audit-ally31-handoff-v1-20261007.py／ally31-handoff-audit-v1-20261007.json。restore-ally31-reference-inputs-v1-20261007.py精確復原18份缺失PNG別名及相同SHA舊比較器；四份既有獨立收據直接路徑全部相符，歷史保留。
- 新正式前端：build-ally31-frontend-v1-20261007.py、enemy360-ally31-frontend-v1-20261007.bin／-build.json，350實際來源。GUI：ally31-gui-v1-20261007.sh、ally31-gui-{party,battle}-v1-20261007/；獨立party及battle-cells收據同日期。沒有新作弊或位置／seed注入。
- DAT：ally31-dat-gui-v1-20261007.sh／.py、ally31-dat-{save,load}-v1-20261007/，含實際PNG、錄製、原版state／frame／player。對照：run-ally31-controls-v2-20261007.py、ally31-{save,load,battle}-control-v2-20261007/；收據ally31-gui-dat-machine-independent-v2-20261007.json。角色區verify-ally31-dat-crops-v1-20261007.py／ally31-dat-gui-crops-independent-v1-20261007.json。
- 效能：ally31-performance-v1-20261007/verified.json及三模式execution／stats／terminal。準備／重生入口prepare-ally31-performance-v2、run-ally31-performance-v3及verify-ally31-performance-v1，日期20261007。正式程式不含計時探針。
- 保全與最後稽核：preserve-ally31-handoff-v1-20261007.py、ally31-handoff-source-snapshot-v1-20261007.tar.gz、ally31-handoff-final-audit-v1-20261007.json。原版、PNG、state、DAT及archive僅本機。

## 下一步

1. 依024 §1.1核對尚未接入的BEAM／FIGHT圖號、ROOM1／MAP／OPEN／END等圖像與動畫。先查既有證據，再補最小來源切片。
2. 補其他人物、小圖、裝備與敵人的正常玩家GUI抽樣，以及剩餘場景DAT與戰鬥成本。
3. 完成權利與實際封包驗收。全部達024 §6才關#34；#44真機驗收獨立保持。
4. HD完成後製作四張README總覽：我方原版、我方HD、敵人原版、敵人HD，保留圖號並說明HD貢獻。此展示已於2026-10-06授權；原版資料包仍留本機。

本批未push、PR、tag或發行；完整HD尚未完成。
