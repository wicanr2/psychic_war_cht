# 接手現況：中文化、完整HD與交付

核對日期：2026-10-08。本次版號 `v.0.3.1-20261008`。dosgolem固定 `31242a9`；原版、規則、RAM及DAT格式不修改。唯一工作清單：[docs/worklist.json](docs/worklist.json)；歷程：[WORKLOG.md](WORKLOG.md)；研究：[038 §207](docs/re/038-hd-theme-feasibility.md)。

## 有效決策

完成全部HD，再交付三平台本機完整版、公開Release與推廣片。公開包只帶程式與已確認可散布素材；原版與全部HD僅放本機，不進Git或Release。人物、效果、迷宮與場景美術已定案，不重問。原位、比例、動作、全域8×8及人物在後／框線在前保持。影片只用DOSBox-X原版實錄 `workplace/dosboxx-audio/title-adlib.wav`，含原版音樂及HD的影片僅留本機。

## 目前狀態表

| 項目 | 現況 | 證據與限制 |
|---|---|---|
| 中文化 | 1,313則、48圖檔／125塊與既有抽測完成 | 缺文本0；兩Logo保留；未全程試玩 |
| 完整HD | 531/531目標PBL圖號已接入 | 6,875個原位entries、78遮罩、256 MAZE、532 PNG；全部僅本機 |
| 人物與戰鬥 | 敵人360、ALLY31、四BEAM工作源及四隊伍FIGHT效果 | 來源、獨立RGBA及限定動作通過；正常GUI敵人30/360、ALLY #0–#2，不把受控來源算成全播放 |
| 房間 | ROOM0全部31、ROOM1全部32 | 63原版COPY、63獨立RGBA及負對照；#14/#17同源共用；ROOM22沿MAZE優先序 |
| 固定場景 | MAP18、OPEN6、END0兩圖、END1十八圖 | 816 COPY原位全RGBA與省略負對照；半列及左裁切沒有外推 |
| 開場親子圖 | OPEN3/5/6兩個完整模板 | 原版0／7像素變化、HD共用裁切及兩全RGBA通過 |
| 場景XOR | OPEN4的16原位、END1#18的76原位 | 92原版操作、雙XOR恢復、B全RGBA、失配／來源／色盤／冷載方向負對照通過 |
| 原版未知 | ENEMY08尾96 bytes、未辨識的中途XOR方向 | 尾三格與方向未知的效果保留原版；不以推測補洞 |
| 新版戰鬥GUI | 36視窗、1,827有限可見格 | 普通Space與F10；原版精確重播44欄位及完整DOS相同 |
| 正常開機 | 新版開機到迷宮及首場遭遇已擷取 | quiet等待原版COPY穩定後送鍵；有限正常路線，非全程試玩 |
| 真正DAT | 512-byte DAT、52-byte玩家資料、24視窗66角色區通過 | 存、讀、戰鬥三條實際鍵序各44欄位及DOS相同；負對照有效 |
| 成本 | 三模式Frame均值1.522／4.491／4.646ms | 繪圖45.20／39.46／29.38 FPS，更新約60次／秒；兩CPU、軟體OpenGL、load<7、null音訊 |
| 四張總覽 | 敵人360及ALLY31的原版／HD已完成 | workplace/hd/full-overviews-v1-20261008；本機，不當作GUI驗收 |
| 指定音源 | 原版AdLib錄音95.058秒 | 全取樣、非靜音、無削波、退出0；非dosgolem波形parity或人耳驗收 |
| 推廣片 | 67秒、多版面候選已合成及字卡目視 | workplace/promo/full-hd-v3-20261008；影音及正式交付收據見下述入口 |
| 三平台封包 | 乾淨tag建置及逐包驗收流程已備妥 | 實際結果以dist-all版本內收據為準；公開包不得有原版或HD |
| 平台限制 | Linux實跑、Windows Wine、macOS結構分開驗 | 真Windows／Mac仍屬#44，不能由容器代替 |
| 發布 | push／PR仍需使用者批准 | 先準備可審閱的公開附件與Release說明；本機完整版及影片不上傳 |

## 現行入口

本機主題：`workplace/hd/theme-full-scenes-ally31-maze-B-v1-20261008/`。前端用 `-theme` 指向此目錄。規格：[024](docs/spec/024-hd-theme.md) §1.55–§1.59、§6與§7；重生及來源入口：[038 §203–§207](docs/re/038-hd-theme-feasibility.md)。

本輪收據均在 `workplace/ida/hd-ally-recruit-20261004/`：

- `hd-full-scene-copy-art-independent-v1-20261008.json`：816 COPY來源及全RGBA。
- `hd-completion-opening-parent-art-independent-v1-20261008.json`：兩親子模板及負對照。
- `hd-completion-scene-effects-prototype-independent-v1-20261008.json`、`hd-full-final-effect-output-audit-v2-20261008.json`：92 B圖面、正式輸出與冷載方向。
- `hd-full-products-tests-v2-20261008.log`、`hd-full-cold-tests-v1-20261008.log`：全部產品套件與新效果fixture明示執行。
- `hd-completion-frontend-v4-20261008-build.json`：本輪冷載階段前端的352份來源及binary SHA；最後撤圖修正的正式程式以乾淨tag封包的HEAD、SHA與產品測試v2為準。
- `hd-completion-gui-machine-independent-v2-20261008.json`、`hd-completion-gui-cells-independent-v2-20261008.json`：普通戰鬥原版重播及有限可見格。
- `hd-full-gui-dat-machine-independent-v1-20261008.json`、`hd-full-dat-gui-crops-independent-v1-20261008.json`：DAT與三條GUI原版對照。
- `hd-full-performance-v1-20261008/verified.json`：三模式成本及計數器負對照。
- `hd-full-boot-gui-v3-20261008/`：正常開機、移動、戰鬥、F4及HD切換；v1／v2送鍵落於轉場，不當作到迷宮證據。

## 交付與下一步

唯一交付根為 `dist-all/v.0.3.1-20261008/`：公開包在 `patch/`，本機完整包在 `full-local/`，影片與影音收據在 `promo/`，實際封包驗收在 `smoke/`。`SHA256SUMS.json` 列出正式產物的大小、雜湊、建置HEAD及權利分類。產物不進Git。

重生：[tools/package.sh](tools/package.sh)、[release-stage.py](tools/release-stage.py)、[release-verify.py](tools/release-verify.py)。先完成全部commit、建立同名tag，再從乾淨輸入打包。Linux兩包驗實际GUI、DAT與缺字型負對照；Windows另驗Wine；macOS驗雙架構、簽章、最低版本與動態相依。實際結果以版本內 `smoke/package-verification.json` 及平台收據為準；不存在或失敗時不得宣稱交付驗收通過。

推廣片：[capture-current.py](tools/promo/capture-current.py)、[make-current.py](tools/promo/make-current.py)。67秒靜態分幕已在timeline明示，需驗影音、黑幀、凍結與字幕。四總覽由[overview.py](tools/hd/overview.py)重生。工具鏈沿專案Dockerfile，Go1.24.13、IDA9.4、固定DOSBox-X来源5fcf624b；macOS revision沿既有SDK工具鏈改為同版Go，SDK只留本機。

完成封包與影片收據後，公開附件及Release說明供使用者批准push／發布。#44真機驗收維持獨立未完成；不重新深挖硬體時序或已完成來源。
