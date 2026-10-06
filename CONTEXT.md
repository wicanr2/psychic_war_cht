# 接手現況：中文化與HD化

核對日期：2026-10-06 11:11（台灣時間）。本週暫停，提交前主repo基準527456b；dosgolem進度提交31242a9。
未完成權威：[docs/worklist.json](docs/worklist.json)。歷程：[WORKLOG.md](WORKLOG.md)。

## 最新結果

本週依使用者要求暫停開發，保存進度並commit／push。正式本機主題仍46PBL＋256MAZE／44PNG，敵人27/360、ALLY3/31，完整HD未完成。最後Sivad上層兩段普通GUI共13鍵，4次完整44欄位／DOS與負對照通過；新電梯下降段GUI自然退出0，尚未完整重播，末點area1(10,2)朝北仍在電梯。下週從保存點確認離開與防衛控制路線，不重試Celtac衛星砲分支；美術已定案，不重問。收工一般測試及兩個主程式編譯通過，研究038 §192。

沿正式定案風格新增ALLY #7第二稿與#8第二／第三稿，三份完整RGBA轉檔及負對照通過；#8左手對位改善，#7指端仍偏高，九人物草稿共24份原生，未正式接入。Celtac實際GUI選單與出發共7普通鍵，4次完整44欄位／DOS與負對照通過。選單重播的舊時鐘差異已依前端載入契約修正；原版出發遭衛星砲後返回Samar，停止重試同路線，下一步查Sivad防衛控制前置條件。正式46PBL＋256MAZE／44PNG、敵人27/360、ALLY3/31保持；研究038 §191。

HD美術正式定案與不再詢問的授權保持。Rusteck四段實際GUI共7個普通鍵／14事件、8次完整44欄位與DOS比對及有效負對照通過。F3分支戰敗後停止；既有F7／F8取得實際戰後保存點，但兩次輔助重播與GUI的6項機器欄位仍不同，不列通過。後續普通鍵走到另一座電梯並上樓，離開後仍為已接入ENEMY04 #9，不增加正式圖數。正式46PBL＋256MAZE／44PNG、敵人27/360、ALLY3/31保持；研究038 §190。

ENEMY04 #9–#11三張HD已限定選入正式本機主題，敵人由24/360增至27/360；主題46PBL＋256MAZE／44PNG、ALLY3/31保持。正常9→10→11→10→9來源沿§188，八份完整圖面、載回／冷載／開關與44欄位／DOS通過；實際Up進戰鬥的24張視窗抽樣中11張可見格吻合，三姿勢皆排除原版及其他兩姿勢，13張未列通過。GUI實際鍵另有兩次完整44欄位／DOS獨立比對，負對照有效。效果遮擋沿原8×8回退，沒有新增作弊；其餘33張既有候選、其他sprite、全動畫、新敵人DAT與封包仍待完成，研究038 §189。

ENEMY10／11三十姿勢草稿、十一原生／33轉檔及完整提示沿§187保全；正常來源、構圖及動作仍待驗，不列正式接入。

HD美術正式定案，沿AGENTS §12及024 §1.26自主製作與修正，不再詢問美術或逐張批准。原版位置、比例、姿勢、8×8、人物在後／框線在前、單張迷宮圖集、ROOM22統一、B微明暗與效果B透光保持。

現行本機主題為theme-enemy04-group3-maze-B-v1-20261006。新三姿勢限定接入沿研究038 §189；前43筆素材保持。中英電梯限定來源／圖面沿研究038 §175–176；ALLY2存讀14/14整屏、512DAT及52玩家bytes沿§178；電梯附近樓層存讀／再入10/10整屏沿§182。不得外推新敵人、效果中途DAT、完整動畫或封包。

敏頓B透光三份完整合成原型已驗，正式接入仍待內建遮罩資料表示回覆及READY來源／冷載／合成契約。既有A／B格式提問保持，藝術方向不重問。來源與101變數模型沿研究038 §180–181。其餘333敵人、28ALLY圖號、小圖、效果、未見迷宮、全動畫、效能、公開權利與HD封包仍待完成。

## 目前狀態表

| 項目 | 目前狀態 | 證據與限制 |
|---|---|---|
| 中文化 | 1,313則譯文、48張圖檔／125塊疊字與既有正常路徑抽測完成 | 缺文本0；兩張標題美術字依定案保留。未全程試玩，本批未改文本／字型 |
| HD執行期 | dosgolem 204既有圖面CONFORMED、新文字背景§5 READY；024各限定契約READY | 完整HD未CONFORMED；原版程式、資料及玩法不改 |
| 現行本機主題 | 46筆PBL＋256格MAZE／44PNG，新ENEMY04 #9–#11 | workplace/hd/theme-enemy04-group3-maze-B-v1-20261006/；研究038 §189。八份完整圖面差0、GUI11/24可見格及三姿勢通過；前中英電梯限定證據沿§176 |
| DAT存讀 | 前43筆主題的ALLY2及電梯附近樓層存讀通過 | 真正視窗14/14完整960×600差0；512位元組DAT、52位元組玩家資料、15處肖像來源及負對照通過，研究038 §178。電梯附近樓層DAT／再入已限定驗，其他新敵人DAT另驗；前批39筆ALLY1證據沿§169 |
| 敵人美術 | 正式限定27/360；另33張候選備齊並技術驗證 | ENEMY00 #0–#8、ENEMY01 #0–#2／#6–#8、ENEMY03 #3–#5、ENEMY04 #3–#11；完整動畫另驗 |
| 批次候選 | 78筆PBL＋256格MAZE／76PNG，敵人美術60/360 | theme-ready60-maze-B-v1-20261005/；合成60張圖面及既有正常12圖面通過，剩餘33張呈現抽測待完成，研究038 §162 |
| ENEMY10／11美術 | 三十姿勢草稿、十一原生／33轉檔，未正式接入 | 原圖30份全391圖庫唯一；33/33完整RGBA與有效負對照通過。ENEMY10第4組v2曲線修稿，正常RAM／構圖／動作待驗；024 §1.45、研究038 §187 |
| ENEMY09美術 | 五組十五姿勢草稿、六份原生／十八轉檔，未正式接入 | 全15原圖391圖庫唯一；18/18完整RGBA與負對照通過。第0組方格感、正常來源／構圖／動作待驗；024 §1.44、研究038 §186 |
| ENEMY05美術 | 五組15張新草稿，未正式接入 | #0–#14的原生與72×96保全；完整來源唯一性及參照已核對；正常圖庫／RAM差分仍待驗。研究038 §163–164，024 §1.30 DRAFT |
| ENEMY06美術 | 五組15張草稿，未正式接入 | 六原生／18單格含舊稿保全；固定轉檔15/15及負對照通過。研究038 §165、024 §1.31 DRAFT |
| ENEMY07美術 | 四組12張草稿，#12–#14未採用 | 十原生／24單格含失敗稿保全；固定轉檔24/24及負對照通過，未正式接入。研究038 §167、024 §1.33 DRAFT |
| 圖庫載入證據 | C6原始載入與四份正常RAM吻合 | 180指令、140段／32,320 bytes；236次靜態模型中156次屬其餘八圖庫，未正常載入，短圖與別名保留，研究038 §166、024 §1.32 DRAFT |
| 來源接續 | 新電梯及ENEMY04 #9–#11正常來源與限定HD已驗 | 18普通鍵／36事件、10次44欄位／DOS與五次原版差分沿§188；本批GUI Up及三姿勢11/24可見格沿§189，另兩次完整狀態相同。更早F7／F8起點限制保持，全動畫及新DAT仍待驗 |
| 敵人來源 | 四檔60張身體來源技術已接入 | ENEMY00／01／03／04 #0–#14；不代表60張美術完成。ENEMY02像素別名、ENEMY08短圖例外保留 |
| ALLY人物草稿 | #3–#11九張，二十四份原生保全 | #3第五稿轉檔通過，端點微調停止；手腳及其他明示構圖差異仍待修；正常來源待驗。研究038 §177／179、024 §1.35–1.36／1.40–1.41 DRAFT |
| ALLY | 正式3/31圖號，完整人物3/12；剩餘9人物及19其他圖形 | #0／#1／#2各兩位置。#12–#15局部24×32，#16–#30小圖16×16，用途與模式未知；不當19個新人物。#2既有GUI、存讀及道具驗收保持 |
| Gestinti | #6–#8來源與美術限定接入 | §160；正常循環、載回／冷載、14份圖面與GUI19/24可見格通過；完整動畫另驗 |
| 房間與OVER | ROOM0 #0／#2／#3／#5／#8／#22、OVER #0限定接入 | #5中文材質與原位英文已驗；ROOM22採迷宮，原座標及8×8不變，研究038 §176 |
| 迷宮 | B v9單張256格圖集已選入 | 598組／4029對已觀察接縫差0；前批31筆正常DAT GUI9/9、512位元組與52位元組資料通過，§153；未見場景另驗 |
| 小圖塊 | 四檔60張16×16 EGA來源已核對，未正式HD | 209次正常敏頓貼圖全屏重建相同；尾段用途未知，024 §1.24 DRAFT、§154 |
| 戰鬥效果 | B透光已定案，敏頓來源／方向／完整冷載模型已驗；正式接入未完成 | 1,009一般＋110遮罩、2,238邊界、101變數／秩101，44欄位／DOS相同；研究038 §180–181、024 §1.5／1.42及§2.0 DRAFT。三份B合成原型已驗；未知中途來源、資料表示回覆、正式GUI與DAT仍待完成 |
| 交付／平台 | 完整HD封包未完成 | 全動作、效能、逐項素材公開權利、封包與平台驗收待完成；Windows wine與macOS結構驗證限舊中文版，真機#44 |

## 證據與執行入口

- 規格：[024 HD主題](docs/spec/024-hd-theme.md) §1.28–§1.45。研究：[038 HD可行性](docs/re/038-hd-theme-feasibility.md) §161–192。
- 新主題：workplace/hd/theme-enemy04-group3-maze-B-v1-20261006/manifest.json與selection-receipt.json。前端旗標：-theme workplace/hd/theme-enemy04-group3-maze-B-v1-20261006。
- 精確決定與素材：workplace/ida/hd-ally-recruit-20261004/hd-style-decision-v1-20261005.json。
- ALLY2同狀態及GUI：同研究根ally2-style-runtime-v1-20261005/runtime.json、ally2-style-independent-v1-20261005.json、ally2-style-gui-independent-v1-20261005.json。前批Gestinti／Jaxemo收據各沿§160／§159。
- ALLY2美術與完整提示：workplace/hd/redraw/ALLY-02-v10-{generated,frame,prompt,generation,measurements,review}-20261005；使用內建imagegen，舊稿保留。
- 前批迷宮B v9與DAT：研究038 §153。29筆葛雷戈林40/40整屏GUI及DAT15/15限§126–127；27筆歐格斯40/40限§121–122，不能外推現行43筆。
- 收尾與來源保全：同研究根ally2-style-art-final-audit-v1-20261005.json、ally2-style-source-snapshot-v1-20261005.tar.gz；執行規則依AGENTS，全部分析與驗證走既有Docker image psychicwar-go-ebiten:latest。

- 批次候選與驗證：workplace/hd/theme-ready60-maze-B-v1-20261005/manifest.json；同研究根enemy-ready60-fixtures-v1-20261005/{plan,render}.json、enemy-ready60-normal-v1-20261005/runtime.json、enemy-ready60-final-audit-v1-20261005.json。重跑入口apps/psychicwar/theme/ready60_art_test.go，明示PSYCHICWAR_READY60_ART_PLAN及PSYCHICWAR_TEST_ORIG；詳研究038 §162。

- ENEMY05草稿及提示：workplace/hd/redraw/ENEMY05-selected-drafts-v1-20261005.json、ENEMY05-prompts-v1／v2-20261005.json；比較ENEMY05-all-comparison-selected-v1-20261005.png。同研究根enemy05-art-final-audit-v1-20261005.json；正式SOURCE／GUI尚未通過。

- ENEMY05來源更正入口：workplace/hd/redraw/ENEMY05-reference-proof-v2-20261005.json；同研究根enemy05-source-review-v1-20261005.json、enemy05-saved-state-survey-v1-20261005.json及enemy05-source-independent-machine-v1-20261005.json。研究038 §164保留原始位址基準與限制。

- ENEMY06素材、原生與完整提示：workplace/hd/redraw/ENEMY06-selected-drafts-v1-20261005.json、ENEMY06-prompts-v1／v2-20261005.json、ENEMY06-all-comparison-selected-v1-20261005.png；收尾同研究根enemy06-art-final-audit-v1-20261005.json。圖庫切換IDA匯出enemy-bank-routing-ida-v1–v4-20261005，證據限制見研究038 §165。

- 圖庫載入證據：workplace/ida/hd-ally-recruit-20261004/enemy-bank-loader-proof-v1-20261005.{py,json}、enemy-bank-routing-ida-v5–v8-20261005及enemy-bank-loader-final-audit-v1-20261005.json；F1原版終點與控制沿enemy-bank-f1-jaxemo-v1-20261005-event、independent-v1及original-v1。

- ENEMY07草稿與失敗版本：workplace/hd/redraw/ENEMY07-selected-drafts-v1-20261005.json、ENEMY07-prompts-v1–v4-20261005.json、ENEMY07-selected-comparison-v1-20261005.png與ENEMY07-rejected-comparison-v1-20261005.png；原版參照及檔案核對同研究根enemy07-art-final-audit-v1-20261005.json。

- 現行39筆真正DAT：同研究根hd39-dat-{save,load}-v1-20261005/{execution,terminal,verified}.json、hd39-dat-byte-proof-v1-20261005.json、hd39-dat-final-audit-v1-20261005.json及hd39-dat-source-snapshot-v1-20261005.tar.gz。重跑入口hd39-dat-gui-v1-20261005.sh，Docker內傳save或load；詳細容器與範圍見研究038 §169。
- 正常路線：同研究根enemy-bank-normal-route-proof-v1-20261005.{py,json}及enemy-bank-zellwal-*-20261005；滿體力新起點enemy-bank-zellwal-party-v1-20261005-event-observed.state，原版未載入其他八圖庫，詳§168。

- ALLY #3–#5草稿與提示：workplace/hd/redraw/ALLY-03-05-{reference-proof,prompts,verification,review}-v1-20261005.json、prompts-v2與ALLY-05-prompt-v3；三份ALLY-03／04／05-comparison-v1-20261005.png。重跑與保全入口同研究根prepare-ally3-5-reference-v1-20261005.py、verify-ally3-5-art-v1-20261005.py、finalize-ally3-5-art-v1-20261005.py、ally3-5-art-final-audit-v1-20261005.json及source-snapshot；詳研究038 §170。

- 新正常路線與保全：同研究根candidate-sivad-proof-v2-20261005.json、candidate-sivad-final-audit-v1-20261005.json及source-snapshot；重跑candidate-sivad-route-v1，十三段argv沿run收據。房間入口捕獲限制沿transport-room-source-v1／v2，詳§171。
- ALLY #6–#8素材與完整提示：workplace/hd/redraw/ALLY-06-08-{reference-proof,prompts,generation-jobs,verification,review}-v1-20261005.json與三份comparison-v1；同研究根prepare／verify／finalize-ally6-8-art-v1及ally6-8-art-final-audit-v1，詳§172。

- 新三姿勢：同研究根sivad-group0-body-source-v2、sivad-group0-style-{runtime,independent,gui-independent}-v1、assisted-GUI-v2-replay-v2、sivad-assisted-route-control-v1-20261006；保全與重生索引見研究038 §173–174。

- 電梯完整來源、圖面與GUI：同研究根elevator-source-proof-v2、elevator-route-controls-proof-v3、elevator-runtime-independent-v1、elevator-gui-independent-v1-20261006.json。重跑elevator-runtime-v2-20261006.go與elevator-gui-v5-20261006.sh；新前端elevator-frontend-v1-20261006.bin，實際347依賴elevator-frontend-v2-actual-inputs-20261006.json。原生／提示與五版素材在workplace/hd/redraw/ROOM0-05-*，私有保全elevator-source-snapshot-v1-20261006.tar.gz。

- 中英電梯：同研究根elevator-english-source-v1、runtime-independent-v1、gui-independent-v1-20261006.json。新前端elevator-english-frontend-v1-20261006.bin及-build.json保存348實際依賴；runtime95、兩個匯出器各92。重跑build-elevator-english-v1、elevator-english-gui-v1與prepare-elevator-english-gui-proof-v2-20261006。保全elevator-english-source-snapshot-v1-20261006.tar.gz，收尾elevator-english-final-audit-v1-20261006.json，沿研究038 §176。

- 新ALLY素材與完整提示：workplace/hd/redraw/ALLY-09-11-{reference-proof,prompts,generation-jobs}-v1-20261006.json，ALLY-10-prompt-v2、generation-job-v2、verification-v2，ALLY-art-{verification,review}-v1-20261006.json。原生ALLY-09／10／11-v1、ALLY-10-v2與#3第三稿、#5第四至七稿全部保留。重跑入口同研究根prepare-ally9-11-reference-v1、verify-ally-art-v1、verify-ally10-v2-v1-20261006.py；範圍見研究038 §177。
- 原版接續與更正：同研究根ally3-normal-source-proof-v1-20261006.json；ally3-normal-follow-v1及verify-ally3-normal-v1-20261006.py。五組44欄位／DOS與負對照入口*-independent-v1-20261006.json；沒有新ALLY來源。
- 本批私有封存：同研究根ally-art-source-snapshot-v1-20261006.tar.gz及.json，344份／255檢核、串流344/344通過，SHA d3f1de9ac4ef749d77391f35a655040c0516c8bceb4d75a6c9ae8b6d2b7da5d9。重跑preserve-ally-art-v1-20261006.py；收尾與封存後文件SHA沿ally-art-final-audit-v1-20261006.json，範圍限研究038 §177。

## 下一步

1. 抽測已READY其餘候選33張的正常呈現；核對ENEMY05來源，完成其他圖庫、剩餘28個ALLY圖號、小圖及效果。ALLY #3–#11修明示構圖差異並驗正常來源；#12–#15局部圖形及#16–#30小圖先查正常模式，不補畫成新人物。#5停止同類提示微調，依原版骨架重新檢查手臂整段範圍。ENEMY05及ENEMY06各十五草稿、ENEMY07十二草稿保留，技術契約待驗；ENEMY07 #12–#14先補構圖與造型證據，再續畫。
2. 將已核對的敏頓效果來源與101變數模型接到B透光合成，完成正式來源、資料表示、8×8回退與冷載契約，再驗GUI及真正DAT。正常1,009貼圖與110遮罩方向已解出，不再重跑相同來源矩陣。其他圖庫與完整動畫仍須完成；來源或分支未知時保留原版。
3. 電梯中英招牌已限定通過；接續其他sprite的正常來源與接入。電梯來源已解出，停止舊猜入口路線與F1固定分支。效果B方向已定，完成來源與遮罩契約後接入。
4. 抽測未見迷宮接縫，補驗新敵人與效果存讀，完成效能、逐項素材公開權利、HD封包與平台驗收。ALLY2存讀沿§178；電梯附近樓層存讀／再入沿§182已限定通過，按原版可達路徑驗收，不要求改原版選單允許動畫中Save。

## 遠端同步

[GitHub #34](https://github.com/wicanr2/psychic_war_cht/issues/34)同步ENEMY04 #9–#11限定接入、46／44與27/360，研究038 §189，保持OPEN。本批全文讀回rusteck-enemy04-g3-style-issue34-after-v1-20261006.json；前批來源、素材及DAT限制保持。
#44未修改；未commit／push／發行。原版、PNG、state與DAT留本機。

- 現行43筆敏頓DAT：同研究根hd43-ally2-dat-{save-v4,load-v1}-20261006/{execution,terminal,verified}.json、hd43-ally2-dat-byte-proof-v1-20261006.json。GUI入口hd43-ally2-dat-gui-v4-20261006.sh傳save、hd43-ally2-dat-load-gui-v1-20261006.sh傳load；獨立原版控制hd43-ally2-dat-original-v1／v2-20261006.py，完整畫面驗收hd43-ally2-dat-verify-gui-v2-20261006.py。資料與素材全本機，來源建置及限制見研究038 §178。

- 本輪存讀檔保全：同研究根hd43-ally2-dat-source-{manifest,preserved,snapshot}-v1-20261006與hd43-ally2-dat-final-audit-v1-20261006.json，1166份／746輸入核對，研究038 §178.4。#34讀回全文一致、保持OPEN，專用容器已清理。

- ALLY全31圖號分類：同研究根classify-ally-archive-v1–v3-20261006.py、ally-archive-classification-v3-20261006.json與ALLY-all31-classification-v3-20261006.png；#12–#14來源參照及proof在workplace/hd/redraw/，只作靜態分類。原生第四稿ALLY-03-v4-{generated,frame,preview,comparison}-20261006.png、prompt與generation-review保留，實際負對照入口ALLY-03-verification-v4-20261006.json。
- 候選來源盤點與十四段接續：同研究根enemy-ready60-pending-source-inventory-v1、enemy-ready60-follow-proof-v1-20261006.json；重跑enemy-ready60-follow-v1與verify-ready60-follow-v1-20261006.py。各段run、event、observed／control、independent及失敗收據全部保留，範圍見研究038 §179。

- 本批增量保全：同研究根ready60-follow-source-{manifest,snapshot}-v1-20261006，306份／116檢核、串流及完整成員核對通過；封存後文件與最終核對ready60-follow-final-audit-v1-20261006.json。#34全文4038字讀回相同、保持OPEN，updatedAt 2026-10-05T21:28:08Z；研究038 §179.4。完整HD未完成。

- 敏頓效果來源與冷載：同研究根minton-effects-normal-v1-20261006.json、minton-effects-directed-proof-v1-20261006.json、minton-effects-cold-scene-v1-20261006.json與machine-independent-v1。重跑prepare-minton-effects-v1、build-minton-effects-v3／v4、run-minton-effects-v1、verify-minton-effects-v1及verify-minton-cold-scene-v1-20261006.py。v4二進位SHA78eed91f7b8921b1ab304d58ed8c879abc4e97bb7595222b289f43ff06e19b1e、63實際編譯來源凍結於-build.json。來源、失敗建置與收尾保全沿研究038 §180。

- 本批效果保全與收尾：同研究根finalize-minton-effects-v1-20261006.py及minton-effects-source-{manifest,snapshot}-v1、final-audit-v1。本輪保全3493份／3470項輸入檢核，串流3493/3493通過。私有封存SHA88412640e6fdd778d0d4989d3ac35b930b6827602e1851925ac5565e02d154c1，34426720位元組；精確清單同研究根minton-effects-source-manifest-v1-20261006.json。封存後文件與遠端讀回SHA由final-audit-v1保存。

- 敏頓B透光格式原型：同研究根minton-effects-format-prototype-v2-20261006/{A,B}/manifest.json、minton-format-render-output-v1-20261006.json及minton-format-render-independent-v1-20261006.json。三份完整RGBA差0，14種素材／101來源變數；固定終點色盤，只驗原型。重跑prepare-minton-format-render-v1、build-minton-format-render-v1及verify-minton-format-render-v1-20261006.py，94份實際來源-build.json保全，研究038 §181。格式回覆待完成，美術不重問。

- 本批B透光原型私有保全：同研究根minton-format-source-{manifest,snapshot}-v1-20261006及final-audit-v1。247份／129檢核，串流247/247；SHA 4bc8f65a317679a721349a1d071b64ed0b122df31554074a67f7c01f92d1df39。封存後文件與擁有權同研究根minton-format-closeout-docs-v1-20261006.json，研究038 §181。#34全文一致、保持OPEN。

- 本批電梯樓層DAT／再入：同研究根elevator-dat-{save,load}-v2-20261006/{terminal,verified}.json及elevator-dat-byte-proof-v1-20261006.json，10/10完整GUI、512-byte DAT、兩階段52玩家bytes及原位中英來源已驗。GUI入口elevator-dat-gui-v2-20261006.sh傳save／load；原版控制elevator-dat-original-v1、獨立圖面verify-gui-v1／v2及byte-proof-v1，研究038 §182。
- ALLY3第五稿與指端更正：workplace/hd/redraw/ALLY-03-v5-{generated,frame,preview,comparison}-20261006.png、prompt-v5及generation-review-v5／verification-v5；來源更正fingertip-source-proof-v1。九人物草稿／二十一原生保持DRAFT，停止同類端點微調，研究038 §182。

- 本批ALLY3／電梯DAT保全：同研究根ally3-elevator-dat-source-{manifest,snapshot}-v1-20261006與final-audit-v1；本輪私有保全776份／1025輸入檢核，串流776/776完整核對通過。封存SHA04f7e46692aff1293f5301175f095a983b9bf3e086341c10c62f3aea0ab618e8，58977136位元組。#34全文讀回一致、updatedAt 2026-10-05T22:44:48Z、保持OPEN。 原版、PNG、state、DAT純本機；研究038 §182。

- 美術再次確認與Rusteck新起點：同研究根rusteck-assisted-source-gui-v1-20261006/及rusteck-assisted-follow-proof-v1-20261006.json。GUI入口rusteck-assisted-source-gui-v1-20261006.sh；後續來源rusteck-assisted-follow-v1、獨立核對verify-rusteck-assisted-follow-v1、文件與保全finalize-rusteck-style-reconfirmed-v1-20261006.py。新起點與四段收據均只作來源探索，完整HD未完成，研究038 §183。

- ENEMY08十二姿勢新稿：workplace/hd/redraw/ENEMY08-prompts-v2／v3、generation-outputs-v2／v3、group0–3-provenance-v2／v3、draft-selection-v3及art-verification-v2-v3-20261006.json。八份原生、24份完整轉檔與負對照已驗；正常來源與構圖仍DRAFT。重跑入口同研究根prepare-enemy08-reference-v1、prepare-enemy08-prompts-v2、prepare-enemy08-art-v2-v3、verify-enemy08-art-v2-v3、finalize-enemy08-art-v1-20261006.py，研究038 §184。

- Rusteck新GUI路線：同研究根rusteck-corridor-gui-v1／v2-20261006/及rusteck-elevator-up-gui-v2-20261006/。下層重播rusteck-corridor-proof-v1-20261006.json；電梯與上層重播rusteck-{elevator-up-v2,corridor-v2}-replay-v2-20261006-event-proof.json。重跑rusteck-corridor-gui-v1／v2.sh、rusteck-elevator-up-gui-v2.sh及rusteck-normal-gui-replay-v2.py，檔名日期均20261006。上層可用分岔保存點rusteck-corridor-gui-v2-20261006/key-02.state，(2,1)西側已驗阻擋、東側通往已走區，見研究038 §186；末點(3,2)為剛進入電梯，送選單鍵前須等原版完成。研究038 §185保存範圍、失敗分類、精確入口與私有保全。

- ENEMY09本批草稿：workplace/hd/redraw/ENEMY09-reference-proof-v1、prompts-v1、group0-prompt-v2、generation-outputs-v1／v2、group0–4-provenance-v1與group0-provenance-v2、selected-drafts-v1及art-verification-v1-20261006.json。全提示、六份原生及十八張轉檔均留本機。重跑與私有快照入口同研究根prepare-enemy09-reference／prompts／art-v1、prepare-enemy09-group0-art-v2、verify-enemy09-art-v1及finalize-enemy09-art-v1-20261006.py，研究038 §186。

- Rusteck分岔停止線：同研究根rusteck-corridor-gui-v3-20261006/與rusteck-corridor-v3-replay-v3-20261006-event-proof.json。西側撞牆及東側已訪區不重試。上層(2,2)向南仍未驗，可從rusteck-corridor-gui-v2-20261006/initial.state接續；不稱整層探索完成。

- ENEMY10／11原生與完整提示：workplace/hd/redraw/ENEMY10／11-selected-drafts-v1-20261006.json、prompts-v1、all-comparison-selected-v1及ENEMY10-group4-provenance-v2。研究根素材、GUI南側／等待／Enter三段重播及私有保全入口沿研究038 §187。
- Rusteck前批三姿勢起點：同研究根rusteck-corridor-gui-v8-20261006/key-02.state，(14,1)朝南、HP40／EP30、enemy_HP200、seed9BC1。三姿勢來源已驗；v9等待後HP0，不使用戰敗點續戰鬥。重生、勘誤及私有保全沿研究038 §188。

- 本批等待點與三姿勢：同研究根rusteck-room-entry-proof-v1、rusteck-enemy04-g3-source-proof-v1-20261006.json；零步讀取／IDA原始bytes、GUI v5–v9、重播及負對照入口見研究038 §188.3。私有保全rusteck-room-entry-source-manifest-v1與final-audit-v1-20261006.json。

- 新ENEMY04第三組HD：同研究根rusteck-enemy04-g3-style-{runtime,independent,gui,gui-independent}-v1-20261006；八份完整圖面、GUI11/24可見格及原版實際鍵重播已驗。新主題46筆／44PNG，selection-receipt.json限定選入。重生、完整提示索引及私有保全沿研究038 §189。

- 本批Rusteck停止線與重播：同研究根rusteck-{escape-v10,corridor-v11,elevator-v12,corridor-v13}-replay-*-20261006-event-proof.json；7普通鍵、8次完整44欄位／DOS相同。實際F7／F8前段重播兩次有六欄位差異，未列通過；diagnostic-v1／v2沿研究038 §190。新電梯離開仍為ENEMY04 #9，不重試同分岔。更新及私有保全update／preserve-rusteck-continuation-v1-20261006.py；正式數保持46／44、敵人27、ALLY3。

- 本批普通重播歷史版本：同研究根rusteck-normal-gui-replay-v11-before-v13-20261006.py與source-manifest的historical_source_inputs。原始執行SHA與復原版本完全相同，研究038 §190。

- 本批ALLY7／8與Celtac：redraw/ALLY-07-08-review-v2-20261006.json保存三份修稿、24原生及構圖限制；研究根celtac-{select,depart}-replay-v*-20261006-event-proof.json保存7普通鍵、4次完整44欄位／DOS與負對照。選單時鐘契約修正版及衛星砲返回Samar停止線見研究038 §191，來源與保全入口update／preserve-ally78-celtac-v1-20261006.py。正式46／44、敵人27／ALLY3保持。

- 本週收工：研究038 §192；Sivad前兩段13普通鍵／4次完整狀態通過，第三段只驗GUI自然退出。下次入口研究根sivad-defense-elevator-gui-v3-20261006/after-leave.state，area1(10,2)朝北仍在電梯，先確認選單與離開，不猜新路。提交測試weekly-commit-tests-v1-20261006.json、遠端及私有快照weekly-closeout-final-audit-v1-20261006.json。Goal暫停，藝術不重問。
