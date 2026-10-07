# 038 — HD 主題：現有架構接得住嗎

日期：2026-09-30
相關：`docs/re/023`（`.PBL` 格式）、`docs/re/024`（圖檔內嵌文字）、`docs/spec/011`、
dosgolem `202-translation-overlay`、`203-baked-text-watchers`

---

## 1. 問題

`#34` 原本寫的是「替換 UI 框線、配色或圖檔」。使用者要的是**基於原圖做成 HD 版**。
問的是同一件事的前半：**現在這套轉譯層撐不撐得住把圖換掉。**

## 2. 合成順序本來就留了位置

前端每幀的順序（`cmd/psychicwar/main.go:463`–`480`）：

```
320×200 色號 ──► RGB ──► nearest ×scale ──► 底圖
                                              ▼
                              960×600 RGBA 疊層（g.overPix）
```

放大倍率是旗標（`-scale`，預設 3），疊層是全解析度 RGBA。
**HD 圖要放的地方已經存在，而且已經有東西在用**（中文疊字、圖檔內嵌文字 125 塊）。

## 3. 量測：整個畫面背景是五張固定位置的圖

拿原版參照畫面（`workplace/overlay/ref-wall.rgb.png`，320×200 色號）
對 `SCREEN.PBL` 解出來的五張圖逐位置掃最小差異：

| 圖 | 尺寸 | 最佳位置 | 不同像素 | 佔比 |
|---|---|---|---|---|
| #0 | 320×40 | y=0 | 0 | 0.0% |
| #1 | 320×40 | y=40 | 0 | 0.0% |
| #2 | 320×40 | y=80 | 530 | 4.1% |
| #3 | 320×40 | y=120 | 2,408 | 18.8% |
| #4 | 320×40 | y=160 | 2,327 | 18.2% |

**五張恰好鋪滿 320×200，y ＝ 0／40／80／120／160。**
遊戲先把整個背景從 `SCREEN.PBL` 畫出來，再把迷宮視野、訊息框、狀態數值與角色貼上去。

差異率就是「被後續繪製蓋掉多少」：

- #0／#1 完全沒被蓋 → 整塊比對成立。
- #2 蓋掉 4.1%（訊息框上緣）。
- #3／#4 蓋掉約 18%（迷宮視野、地點／方位的值、走路中的角色）。

`MENU.PBL` #0（88×72）在 (160, 4)，是指令面板。

**結論：整塊比對只對 #0／#1 成立。** 這不是 bug，是 §4 要處理的事。

## 4. 逐格失效剛好就是需要的機制

`202` §2.3 的疊字是**逐格**判斷有效性：原版在某一格上面畫了別的東西，那一格就失效。
把 HD 圖也切成 8×8 格登記，得到的行為正是想要的——

**HD 背景整張蓋上去，原版畫迷宮視野的地方自動破洞露出原版。**

格線對得上：`.PBL` 的尺寸本來就以 8×8 格為單位（`寬÷8`、`高÷8`，`docs/re/023`）。
一張 320×40 的圖 ＝ 40×5 ＝ 200 格。五張 ＝ 1,000 格。
既有的疊字層本來就在算這種雜湊，多這些量不構成問題。

## 5. 擋路的是這一條

`xlate.Layer.Add`（`worktrees/dosgolem/xlate/layer.go`）：

```go
if old.Owner != "" || (nx0 <= ox0 && nx1 >= ox1 && ny0 <= oy0 && ny1 >= oy1) {
    l.drop(old, "overlap")
```

**任何 watcher 建立的疊字，只要跟新疊字重疊就整筆移除。**
面板上的中文（「前進」「轉向」「位置」「方位」）全部是 watcher 疊上去的。
一張蓋住整條 strip 的 HD 疊字加進去，會把那條上面的中文全部刪掉；
下一幀 watcher 又補回來，再把 HD 圖打成半透明。**兩邊互相拆台。**

`Stamp` 另有兩項不合用：

1. 內容只有 `Font` ＋ `Text`（`layer.go:18`–`40`），畫不了任意點陣圖。
2. 格子是**一維橫排**（`Cells`／`CellW`／`CellH`）。二維要拆成每列一筆。

## 6. 觸發方式：兩條路

| | 路線 A：畫面內容比對 | 路線 B：攔貼圖常式 |
|---|---|---|
| 機制 | `xlate.Watcher`（`203`） | `OnCall` 掛 `sub_18A98`／`sub_189BF` |
| 新反組譯 | 無 | 要讀 `sub_18A98` 的暫存器約定 |
| 色盤重映射 | 會中招（`OPEN.PBL` #10：檔案色號 10、畫面 15，`docs/re/024`） | 不受影響 |
| 被蓋住 | 整塊比對失敗（見 §3 的 18%） | 不受影響 |
| 成本 | 每個 watcher 每幀 8 次比較 | 每次貼圖一次回呼 |

`sub_18A98` 的既有證據（`docs/re/023` §4）：邊解壓邊畫到畫面，
位置 ＝ `CH`×4、`CL`×4；`sub_189BF` 用 `AH` 查 `cs:915B` 決定哪個 `.PBL`。
**缺的只有「來源指標怎麼換算回圖號」，是一個明確的小任務，不是未知數。**

第一版走 A（`SCREEN.PBL` 位置固定、`#0`／`#1` 整塊比對已驗證成立）；
範圍擴到會移動的東西時再補 B。

## 7. 素材盤點

537 張，分佈很集中：

| 來源 | 張數 | 尺寸 |
|---|---|---|
| `ENEMY00`–`11` | 360 | 72×72 |
| `ROOM0`／`ROOM1` | 63 | 72×72 |
| `ALLY` | 31 | 72×72 |
| `MAP`（第一人稱視野） | 18 | 152×80 |
| `OPEN`／`END1` 等 | 60 | 不一 |
| `SCREEN` | 5 | 320×40 |
| `MENU` | 1 | 88×72 |

3 倍：怪物 216×216、面板 960×120。

## 8. 三個限制

1. **倍率鎖 3。** 所有逐像素驗收都假設 `scale` ＝ 3；改倍率會讓整套驗收失效。
   HD 圖以 3 倍為準，`-scale` 其他值時退回原版圖。
2. **逐格失效會讓 HD 圖破洞。** 這是 §4 要的行為，但格子邊界會出現原版的鋸齒色塊。
   驗收要看的是「破洞落在該破的地方」，不是「整張都在」。
3. **授權。** 從原圖放大出來的是工画堂素材的衍生作品，性質與 `PW.EXE` 相同。
   `AGENTS.md` 的 `[HARD]`「不得散布原版素材」目前直接禁止它進 repo 與發行包。
   要放行必須明文改規則，否則 repo 裡會有兩條互相矛盾的規定。

## 9. 判定

**可行。** 要新增的程式集中在一處：疊字層要能畫點陣圖、要有一個不跟文字層互相移除的平面。
其餘（生命週期、逐格失效、快照往返、捲動凍結、`Watcher` 比對）全部沿用。

規格：dosgolem `204-art-plane`（DRAFT）、本 repo `docs/spec/024-hd-theme`（DRAFT）。

---

## 10. 第一次放大：三種放大器的實際結果

2026-09-30。六張（`SCREEN.PBL` #0–#4、`MENU.PBL` #0）各跑三種，輸出在
`workplace/hd/{near3,hq3x,xbr3}/`（本機，未進版控）。
放大器用 ffmpeg 內建的 `hqx=3`、`xbr=3`，與 `scale=flags=neighbor` 當基準線，
不需要新 image，也不需要模型權重。

整張畫面的疊合結果：`workplace/hd/full-{near3,hq3x,xbr3}.png`（960×600）。
三組細節對照（左 nearest、中 hq3x、右 xbr3）：`workplace/hd/cmp-{face,logo,mech}.png`。

### 10.1 結果不是一致的，分素材類型

| 素材 | nearest | hq3x | xbr3 |
|---|---|---|---|
| 機械管線（`cmp-mech`） | 階梯狀 | **明顯變好**，圓管變成圓管 | 與 hq3x 接近，稍微過頭 |
| 人物輪廓（`cmp-face`） | 階梯狀 | **變好**，臉部與髮線平滑 | 更圓，開始出現假細節 |
| 標題字（`cmp-logo`） | **最忠實** | 字角被磨圓，紅色格線微幅扭曲 | 磨得更圓，`KGD SOFT` 被讀成 `KGA SOFT` |

**放大器對曲線有利、對設計過的像素字與細格線有害。** 標題 Logo 是刻意畫成方角的美術字，
磨圓之後不是變高級，是變糊。

### 10.2 抖色沒有被解掉

人物皮膚是 EGA 的黃白交錯抖色（`cmp-face` 中段）。
hq3x 與 xbr3 都**把棋盤格原樣放大**，不會把它解讀成一塊中間色。
所以放大後仍然看得出來是 16 色抖色，只是格子變大。

要真的變成「HD」，抖色要先解成連續色調，那不是幾何放大器做得到的事。
這是 AI 放大在這批素材上唯一明確的優勢，也是它值得留在 `scaler` 欄位的理由。

### 10.3 對規格的影響

`docs/spec/024` §2 的 `scaler` 是**逐張**指定。實測顯示**同一張圖裡面就有兩種素材**
——`SCREEN.PBL` #0 同時有標題字（要 nearest）與人物和機械（要 hq3x）。

**`entries` 要能指定子矩形**，一張圖拆成數筆、各自選放大器。
規格照這個修正，不要用「一張一個放大器」的假設往下做。

---

## 11. 改走重繪：方法與第一個實例

使用者定案 2026-09-30：**基於原版重新繪製**，不是放大。
§10 的放大結果留作對照，不當成交付路線。

### 11.1 畫面的素材密度

把 `SCREEN.PBL` 五張疊成 320×200 背景，逐 8×8 格數相異色號（共 1,000 格）：

| 類別 | 格數 | 佔比 |
|---|---|---|
| ≤2 色（純底、單色框線） | 425 | 42.5% |
| 3–4 色（一般 UI 結構） | 220 | 22.0% |
| ≥5 色（細密美術） | 355 | 35.5% |

**約六成五是結構，三成五是細密美術。** 而且密度分佈很平均
（Logo 區 26.5%、指令／訊息區 26.9%、機械帶 30.9%、人物區 28.5%），
不存在「某一整塊全是結構」的乾淨切法。

### 11.2 結構元素可以自動抽出來

以 `MENU.PBL` #0（88×72 指令面板）為例，色號分布只有三種：
黑底 5,858 px、白色 308 px（英文字，本來就被中文疊字蓋掉）、青色 172 px。

青色連通塊（量自 3 倍圖，見 §11.3 的警告）：

| 元素 | 頭 | 刻線 |
|---|---|---|
| 上 | x 120–140、y 45–65 | x 126–134、y 69–71 |
| 下 | x 120–140、y 129–149 | x 126–134、y 123–125 |
| 左 | x 27–47、y 87–107 | x 51–53、y 93–101 |
| 右 | x 216–236、y 87–107 | x 210–212、y 93–101 |
| `ESC` 鍵帽 | x 15–53、y 195–215 | — |

**整個面板的圖形內容就是四個箭頭加一個鍵帽。**
連通塊分析（色號 ＋ 外框 ＋ 像素數）足以把這種元素自動列出來。

### 11.3 ⚠ `.idx` 前面有 4 bytes 檔頭

`tools/pbl.py dump` 寫出的 `.idx` 是 `寬×高 ＋ 4` bytes。
直接當成像素陣列讀，**所有座標會往左偏 4 px**，而且不會報錯——
第一列會多出兩個「色號 88」「色號 72」的單點，那其實是寬與高的數值。

第一次重繪就踩了，畫出來的箭頭壓在文字上。
**座標要量已經解出來的 PNG，不要從 `.idx` 推。**

### 11.4 兩種重繪風格

同一組座標畫了兩版（`workplace/hd/redraw/menu-cmp3.png`，左原版、中忠實、右相連）：

| 版本 | 做法 | 結果 |
|---|---|---|
| 忠實 | 三角形 ＋ 分離的小刻線，完全照原版的連通塊 | 讀起來像「▲ 加一條底線」，不像箭頭 |
| **相連** | 三角形 ＋ 相接的柄，維持原版的外框、位置與顏色 | **清楚是一支箭頭**，抗鋸齒乾淨 |

原版的頭與刻線中間隔 3 px，靠 nearest 放大的塊狀感才讀成一支箭頭；
畫成向量之後那個間隙變得明顯。**相連版採用。**

### 11.5 方法邊界

這套（連通塊 → 圖形基元 → 向量重畫）只對**幾何元素**成立：
箭頭、框線、鍵帽、格線、儀表、管線。

`docs/re/038` §11.1 的 355 格細密美術——人物插圖、機械帶的細節、標題美術字——
沒有可以自動擬合的基元，**需要真的重畫**。這一段不是工具問題，是美術工作。

---

## 12. 幾何那條做完了，而且它天生就小

### 12.1 先量「哪裡放大之後才會看起來鋸齒」

`hq3x` 只改變對角階梯，所以**它跟 nearest 的差異區，就是 nearest 放大會出現鋸齒的地方**。
逐 8×8 格比兩張 960×600（差 ≥24 px ＝ 一格 576 px 的 4% 才算）：

| 類別 | 格數 | 有鋸齒 |
|---|---|---|
| ≤2 色（純底、單線） | 425 | **30（7.1%）** |
| 3–4 色 | 220 | 86（39.1%） |
| ≥5 色（細密美術） | 355 | 194（54.6%） |

**軸對齊的直線用 nearest 放大是完美的——沒有鋸齒可以消。**
原版的 UI 框架幾乎全是軸對齊的矩形與直線（單色格有 344 個，其中 342 個是純黑），
所以「把框線與格線重畫成向量」在畫面上等於沒做。

值得重畫的幾何元素只剩 116 格，而且多半落在 Logo 區（已定案沿用原版）
與機械帶（屬於細密美術，交給畫圖代理）。

**這不是偷懶的結論，是原版的設計決定的**：1987 年的 UI 用直線是因為直線便宜，
而直線正好是整數倍放大唯一不會劣化的東西。

### 12.2 交付：基元渲染器

`tools/hd/render.py`：基元描述（JSON）→ 任意倍率的 RGBA PNG。

- 座標用**原版像素**（可小數），顏色用 **EGA 色號** —— 重繪的配色自動跟原版一致，
  不用另外維護色表，「風格跟原版相近」由格式本身保證。
- 4×4 超取樣求覆蓋率再合成，斜邊抗鋸齒。
- 基元：`poly`、`rect`、`grid`。

第一筆資料 `theme/hd/draw/MENU-00.json`（指令面板，12 個基元）：
四個黑色矩形蓋掉原版箭頭，四組「三角形 ＋ 相接的柄」重畫。
輸出與手寫 ImageMagick 的驗證版逐像素相同。

### 12.3 交付：自動去抖色

`tools/hd/dedither.py`：把兩色交錯的棋盤格換成真正的中間色。

判準是**四鄰中有 N 個同色、而且自己跟四鄰都不同色**。
「自己跟任一鄰居同色就跳過」這條擋掉線與塊——1 px 的線，左右鄰居就是線本身。

| 門檻 | 判定像素 | 結果 |
|---|---|---|
| 4 | 1,176（1.8%） | 抖色區**邊緣留一圈**沒解到的格子 |
| **3** | 2,119（3.3%） | 邊緣乾淨，Logo 與紅色格線未受影響 |

預設 3。輸出是 RGBA 疊層，只有判定為抖色的像素不透明，
**非抖色的部分一律透明讓原版透出來**，所以這支工具的作用範圍是可界定的。

實際效果最明顯的地方是人物皮膚（黃白交錯）與右緣的灰黑棋盤格儀表區
（`workplace/hd/redraw/dedither-cmp3.png`：左原版、中門檻 4、右門檻 3）。

### 12.4 沒做的與為什麼

| 項目 | 為什麼 |
|---|---|
| 框線、格線、鍵帽向量化 | §12.1：軸對齊直線 nearest 已經是完美的 |
| 標題 Logo | `docs/spec/011` §6 已定案沿用原版 |
| 右緣的藍色量表與紅燈 | 語意未查（是靜態裝飾還是真的量表），而且去抖色已經處理掉它最醜的部分 |
| 第一人稱迷宮的牆 | **程式即時畫的，不在 `.PBL` 裡**（`MAP.PBL` 是遊戲裡的地圖道具，不是視野）。要重畫得先讀迷宮繪製常式，是另一個題目 |

---

## 13. 第一次整屏合成：通過

2026-09-30。人物與外框由 codex（`gpt-6-sol`，內建 `imagegen`）重繪，
幾何與去抖色由本 repo 的工具產生，合成腳本 `tools/hd/compose.sh`。

成品 `workplace/hd/redraw/full-hd.png`，驗收 **無主的黑色像素被佔住 0 px**，
全畫面改動 159,337 px（27.7%）。

### 13.1 兩個 codex 的坑

- **`codex exec` 的 `-i/--image` 是可變長度參數**，會把後面的提示詞當成另一個圖檔吃掉。
  症狀是 codex 印「Reading prompt from stdin」然後空轉到被殺（exit 143），
  **看起來像模型當掉，其實是參數解析**。提示詞改用 stdin 餵。
- `pkill -f "codex exec"` 會匹配到自己的 shell 指令，**把自己殺掉**（exit 144）。

### 13.2 外框第一次失敗：不是代理不聽話，是遮罩不夠

第一版把英文字畫進去、標題 Logo 換成抽象圖形、配色從灰／桃紅／青變成深藍。

根因：那些像素在原版**不是黑的**，所以落在「要重畫」的範圍裡。
指示寫了「不要畫文字」，但沒告訴它那塊該留白——代理只好畫東西上去。

`mask.py` 加保護區之後，留給外框的範圍從 39.6% 降到 **12.7%**：

| 保護區 | 來源 |
|---|---|
| 標題與紅色格線 `0,0,160,80` | `docs/spec/011` §6 定案沿用原版 |
| 六個英文字框 | `text/baked.json` 的 `SCREEN.PBL`／`MENU.PBL` 條目算出來 |
| 四個箭頭與 `ESC` 鍵帽 | 本 repo 自己向量重畫 |
| 人物區 `232,0,88,152` | `girl.png` 另外疊 |

第二版守住範圍（代理自己比對：洋紅禁畫區的非黑像素 0），配色也回到原版的
灰白面板／桃紅管線／青色高光。

### 13.3 ⚠ 「不給代理畫」與「沒有人可以畫」是兩件事

第一次驗收失敗 1,775 px，全部來自把這兩份清單混用：

- 箭頭要**擋著不給代理畫**（進 `mask.py` 的保護區），但**我們自己會重畫它**。
- 人物區同理。
- 去抖色會把棋盤格的黑色相位換成中間灰，所以它也需要一塊「有主」的矩形。

分成兩份清單之後：`workplace/hd/keep-rects.txt` 餵 `mask.py`，
`workplace/hd/owned-rects.txt` 餵 `tools/hd/check.py`。

### 13.4 去抖色要限範圍

全畫面跑去抖色會零星佔住 432 px 的孤立黑點（機械帶裡的單像素黑）。
黑色像素在別處是遊戲的地盤，所以 `dedither.py` 加了矩形參數，
只在有主的區域作用（目前只有右緣儀表區 `304,145,16,38`，2,119 px → 295 px）。

### 13.5 人物要去背，不能用黑底

黑底的 `girl.png` 直接疊上去會把她背後原有的機械整塊蓋掉
（那塊在原版有管線與控制台，不是空的）。

改用**邊緣填充去背**（從四個角 floodfill，fuzz 6%），保留角色內部的黑色線條。
去背後改動像素從 173,933 降到 159,337，背後的機械回來了。

---

## 14. 人物收進框線、框線改三段漸層

### 14.1 框線的底邊在 y=143

可見人物裝飾區止於 y=143；y=144 整列全黑，留給原版遊戲。
2026-10-01 使用者定案後，人物保持原版 `(232,0)` 與 88 像素寬，
只裁掉超出裝飾區的部分，由前方控制台／框線自然遮住下緣。
先前等比縮到高 432 再靠右對齊的處理已被取代，舊產物留作歷史，
原因與新驗證見 §31、`000-overturned-claims.md`。

### 14.2 框線本來就是「亮色壓暗色」

原版的框線是一列亮色壓一列暗色（`9` 壓 `1`、`11` 壓 `3`、`15` 壓 `7`…），
那是 16 色下表現立體感的手法。整數倍放大會變成 3 px 亮加 3 px 暗的硬邊。

`tools/hd/bevels.py` 自動掃這種配對，輸出三段漸層基元：
**上緣高光（亮色再提一階）→ 本體（原版亮色）→ 下緣陰影（原版暗色）**。
位置與厚度完全照原版，所以不影響對位，改變的只有立體感的畫法。

最短長度 12 px 掃出 5 條：

| 位置 | 長度 | 色 |
|---|---|---|
| y=122 x=40 | 13 | 14→12→4 |
| y=122 x=83 | 20 | 11→9→1 |
| y=122 x=109 | 33 | 11→9→1 |
| y=142 x=82 | 94 | 11→9→1 |
| y=142 x=286 | 30 | 15→11→3 |

⚠ 掃出來的條數比肉眼看到的框少，因為判準要求**整段同色不中斷**。
原版的框線常被其他構件打斷（例如 y=122 那條被面板切成三段）。
這是保守的取捨：寧可漏掉，不要把不是框線的東西當框線改掉。

`render.py` 跟著支援多段漸層（`gradient` 給 2 個以上色號，`dir` 決定方向）。

## 15. 其餘 537 張圖的結構

**這批還沒開工。** 目前只重繪了人物與整屏外框。
動手之前先看清楚素材結構，因為它決定能不能逐張重畫：

| 來源 | 張數 | 尺寸 | 結構 |
|---|---|---|---|
| `ENEMY00`–`11` | 360 | **16×16** | 小人物／敵人 sprite，每個角色數個動作。不是單張大圖 |
| `ROOM0`／`ROOM1` | 63 | 72×72 | 場景插圖（太空船、星球、電梯面板…），可以逐張重畫 |
| `ALLY` | 31 | 72×72 | 同上 |
| `MAP` | 18 | 152×80 | 遊戲裡的地圖道具（「TOP SECRET MAP 02-V-11」） |
| `OPEN`／`END`／`BEAM`／`FIGHT`／`OVER` | 60 | 不一 | 開場、結局、特效 |

**`ENEMY*` 是 16×16 的 sprite，不是一張大圖。** 逐張丟給畫圖模型重畫，
拼回去會有接縫與比例不一致。這 360 張要當成「同一個角色的數個動作」一起重畫，
不能一張一張送。

`ROOM*`／`ALLY`（94 張 72×72）是完整的場景插圖，可以逐張重畫，
而且是玩家會停下來看的畫面，優先做這批的投資報酬最高。

---

## 16. 批次重繪：管線與進度

使用者定案 2026-09-30：依序做 `ROOM0`／`ROOM1`／`ALLY`／`MAP`／`OPEN`／`END`／
`BEAM`／`FIGHT`／`OVER`，最後才 `ENEMY00`–`11`。

### 16.1 管線

| 步驟 | 工具 |
|---|---|
| 解參照圖與尺寸清單 | `tools/pbl.py dump` → `workplace/hd/batches.json`（169 張，ENEMY 以外） |
| 產生逐張指示 | `tools/hd/batch_spec.py <名>` → `workplace/hd/spec-<名>.md` |
| 派工 | `tools/hd/batch_run.sh <名> [說明]` |
| 驗尺寸 | `tools/hd/verify_batch.py <名…>` |

`batch_spec.py` 的關鍵是**把 `text/baked.json` 的內嵌英文框逐張列進去**。
招牌上的英文執行時會被中文疊字蓋掉，重繪要留成乾淨的招牌底；
代理看參照圖不會知道哪些字之後會被蓋，這件事必須由資料告訴它。

### 16.2 進度

| 批次 | 張數 | 結果 |
|---|---|---|
| `ROOM0` | 31 | **31/31 尺寸正確**，8 張的招牌留白 |
| `ROOM1` | 32 | **32/32 尺寸正確**，19 張的招牌留白 |
| `ALLY` | 31 | 進行中 |
| 其餘 | 75 | 排隊 |
| `ENEMY00`–`11` | 360 | 最後，要整組處理 |

`ROOM0` 一批約 12 分鐘、8.3 萬 token。

### 16.3 已知的副作用

`ROOM0` #27／#28／#29（`DATA`／`COMPUTE`／`INFORM`）原版的招牌是**藍底白字的橫條**，
重繪把橫條換成控制台面板。中文疊字的顏色是從**原版**的像素取的（`Colors()` 取最多色當底、
次多色當字），所以字還是會畫出來，但底色可能跟新畫的面板不搭。

這是「疊字取色看原版、畫面看重繪」的結構性後果，不是這一批的失誤。
要解得讓疊字層知道該塊已被重繪並改用重繪後的顏色——留到實跑驗收時看嚴重程度再決定。

### 16.4 ⚠ ImageMagick 的 `-resize` 是套用到清單裡所有影像

做前後對照時寫成

```
convert 底圖.png 小圖.png -filter point -resize 300% -geometry +12+372 -composite 出.png
```

底圖會跟著被放大三倍，產出一張 2880×1800 的怪圖。要放大單一輸入得包起來：

```
convert 底圖.png \( 小圖.png -filter point -resize 300% \) -geometry +12+372 -composite 出.png
```

## 17. 接手後的素材與貼圖路徑核對

2026-09-30。以下追加目前證據，保留前面各階段的原始記錄。

### 17.1 素材尺寸與預覽

**已證實（資料解碼）**：十二個 ENEMY PBL 共 360 張，尺寸是 24×32 的 177 張、
24×24 的 3 張與 16×16 的 180 張。對尺寸與解碼色號陣列做 SHA-256 分組，
有 359 個不同組；唯一完全相同的一組是 ENEMY02 #12／#13。
這是資料相同的證據，不能單憑像素相似推定其他圖的角色或動作語意。
輸入逐檔雜湊、工具與資料基準見 `workplace/hd/redraw/enemy-input-hashes-20260930.json`。

**已證實（檔案尺寸）**：非 ENEMY 批次有 165 張，明確排除 OPEN #7–#10 後驗證通過。
`tools/hd/verify_batch.py` 已支援 `--skip OPEN:7,8,9,10`，未指定時仍檢查四張字幕，
不會自動忽略缺檔。缺檔、截短 PNG 標頭、錯尺寸、未知批次／圖號與全數排除的
正反對照均已跑過。這仍只是存在與 PNG 標頭尺寸檢查，不是美術或遊戲驗收。

ENEMY00 #0–#2 已用內建 `image_gen` 產生同角色三格 HD 預覽，
三格各 72×96，放在 `workplace/hd/redraw/ENEMY00-group0-frame-00.png` 至 `02.png`；
並排檢視入口是 `ENEMY00-group0-preview.png`，來源及輸出雜湊見
`ENEMY00-group0-provenance.json`。首版仍像像素放大，已排除；修正版改為平順的
賽璐珞輪廓。此為 **prototype**，未接入正式遊戲，未宣稱動畫或逐像素對拍通過。

### 17.2 IDA 輸入與定位

- 輸入：`PW_UNP.EXE`，SHA-256 `fd5115b91f014c47a1fb1a6e2ecfd645e293264fe614a4b350e92637cad2fbd9`。
- 正式資料庫：`PW_UNP.EXE.i64`，SHA-256 `4db51a199f5913c52dc4964557b9a4aafb965754d663cb0173c9dfdfe9a00c56`；查詢前後相同。
- 工具：IDA Pro 9.4，映像 `ida-pro-9.4-idapython:locked-v1`；先以最小 JSON 探針確認輸出，列得 417 個函式。
- 以下位址都是 **IDA linear ea**。`seg002` 的相對位移基準是 `0x10510`；這不是執行期線性位址。其他 segment 使用自己的 IDA base，不混用。
- 正式輸入唯讀掛載，查詢在容器 `/tmp` 的 `.i64` 副本上執行，沒有改名或覆寫原始資料庫。

查詢及探針收據位於 `workplace/ida/scratch/hd-*-20260930.*`。
文字匯出的每列保留原始位址、segment、bytes 與 operand，未附加語意的列明示「未知」，
不可把原始導覽名稱當成語意事實。

| 原始定位 | 附加語意 | 等級 | 證據 |
|---|---|---|---|
| `sub_189BF`，IDA ea `0x189BF`；`0x189C6` 的 `mov bl, ah`、`0x189CA` 的 `add bx, 915Bh`、`0x189F8` 的 `shl ax, 1` | AH 選擇檔名表條目，保存後的 AL 用於檔內 u16 偏移表索引；讀取選定資料，目的地由進入時的 BX 保存 | 已證實，限靜態指令與 DOS 檔案呼叫控制流；未做端到端參數對拍 | `hd-pbl-calls-20260930.txt` |
| `sub_18A7C`，IDA ea `0x18A7C`；`0x18A80` 的 `mov bx, 92BEh`、`0x18A8E` 的 `call sub_18A98` | 在 CS 資料緩衝讀取圖塊後，以 DS:BX 交給螢幕解碼器 | 已證實，限此包裝常式的靜態資料流 | `hd-pbl-source-20260930.txt` |
| `sub_18A98`，IDA ea `0x18A98`；`0x18A9F` 的 `mov al, [bx]`、`0x18AC8`–`0x18ADA` 的 CH／CL 乘四 | 輸入是 DS:BX 圖塊；目的座標由 CH／CL 換成像素；解碼途中呼叫底層畫點常式 | 已證實，限靜態指令；動態覆蓋範圍未知 | `hd-pbl-calls-20260930.txt` |
| `sub_18B76`，IDA ea `0x18B76`；`0x18B83` 的 `call sub_189BF`、`0x18BB3` 的 `mov di, bx`、`0x18BDB`／`0x18C05` 的 `mov [di], ah` | 另有先載入壓縮資料，再解碼至呼叫者記憶體目的地的路徑 | 已證實，限靜態解碼／寫入控制流；哪些正常畫面使用它仍未知 | `hd-pbl-dispatch-20260930.txt` |
| `sub_19187`，IDA ea `0x19187`；檔名表相對位移 `8DAFh`／`8D17h` | 本次抽查索引 4–8 是 `GAME0`–`GAME3`／`END0` 的 `.mid` 與 `.ibm`，不應作為貼圖快取證據 | 檔名 bytes 已證實；完整音樂載入語意不在本輪範圍 | `hd-pbl-cache-20260930.txt`、`hd-pbl-tables-20260930.json` |

**強推論**：只攔 `sub_18A98` 不足以直接宣稱涵蓋全部 HD 素材，因為存在另一條解碼到
記憶體的路徑。這並未證明特定角色一定走後者；下一步應追 `sub_18B76` 的呼叫與取址，
以正常重播核對圖塊、目的地與玩家畫面。取得這條證據前，不把全素材執行期接入標 READY。

這修正了 §6「缺的只有來源指標換算」的充分性判斷：那是第一條解碼路徑的缺口，
目前證據還顯示需要核對第二條路徑，原有 xref 與來源索引保留。

## 18. 正常玩家貼圖與中文 HD 合成原型

2026-09-30。輸入與 IDA 資料庫雜湊沿用 §17.2；正式資料庫仍唯讀，沒有改名。
dosgolem 使用 `f8c1a6ec5f069058f163ce8ddf345e24dc851de6`，Go 1.24.13。
本節的執行期位址以 **段:偏移** 表示，不與 IDA ea 混用。

### 18.1 正常重播與原始圖塊

**已證實（限兩段正常重播）**：從 `06-name.state` 依原重播輸入 KAI，
走到 42,000,000 道指令；另從 `07-first-play.state` 依原重播送出 11 次前進，
走到 86,000,000 道指令。沒有座標注入或直接呼叫遊戲常式。

| 重播終點 | 新收據與既有收據的 SHA-256 |
|---|---|
| 第一人稱迷宮 | `a17c8610ac338deddd3060238b4debe3e837df6c6f8c685d7d9162c0e4cedfc7` |
| 第一場遭遇 | `d0616fd57d1f7d078ea8abae989e6e901ff069f23890a0906e8cea845fb2e7d0` |

第一段觀察到記憶體解碼進入點 `0161:8666`（IDA `sub_18B76`）34 次。
第二段觀察到貼圖進入點 `0161:8705`（IDA `sub_18C15`）16 次。
掛鉤版與未掛鉤版在第二段的結果逐 byte 相同；兩段均與既有正常重播畫面相同。
這些數字只涵蓋列出的區間；`0161:8588` 沒在這兩段觸發，不能推論它在遊戲中沒有用途。

| 原始定位 | 附加語意 | 等級 | 證據 |
|---|---|---|---|
| IDA ea `0x18C15`／執行期 `0161:8705`；原始輸入 DS:BX、CH／CL、DH／DL | DS:BX 是已解碼的 packed 4bpp 圖像；CH／CL ×4 是位置，DH／DL ×8 是尺寸 | 已證實，限指令資料流與本次正常呼叫 | `hd-memory-dis-20260930.txt`、`hd-memory-encounter.log` |
| 第 83,190,318 道指令，`DS:BX=1175:A686`、`CX=0826`、`DX=0304` | 第一筆為 24×32 圖像，貼到 (32,152)；384 bytes 與 ENEMY00 #3 的獨立解碼資料完全相同 | 已證實，限這一筆來源、尺寸與位置；角色身分語意未定 | `hd-enemy-initial.bin`、`hd-draw-observation-20260930.json` |
| 第 83,373,343 道的 `1175:A8C6`、第 83,556,097 道的 `0161:344A` | 後續兩筆資料無法直接比對到完整原始圖塊 | **⚠ 未知**；不將最相近的像素候選當成來源 | `hd-enemy-memory.bin`、`hd-enemy-code.bin` |

固定初始狀態的 SHA-256、執行前已固定的 CS:41DF 種子、設定方式、完整輸入及對照結果，
保存在 `workplace/probe/hd-draw-observation-20260930.json`。
此為原版圖像來源觀察，不是 HD 動畫或全素材驗收。

【HD-XOR-01】本節兩筆「未知」保留為當時觀察；2026-10-01 的 §20 已證實它們
分別是 ENEMY00 #3／#4、#4／#5 的 XOR 差分，並以正常貼圖前後畫面驗證。
「不等於完整原圖」仍成立，新的證據補足其來源與轉換，沒有把近似比對當成同一圖。

### 18.2 IDA 無輸出的診斷與修正

連續空輸出後依診斷入口檢查正對照、啟動與原始紀錄。
最小探針的紀錄明示 `UnicodeEncodeError`：容器的預設文字編碼是 ASCII，中文寫檔失敗。
`tools/ida/dump.py` 改為明示 UTF-8；在同一映像、同一唯讀資料庫副本、同一查詢範圍重跑，
取得非空指令與交叉參照輸出。這是工具輸出修正，沒有改動原版或正式資料庫。

原始位元組探針也成功輸出五個查詢：四個有函式邊界，`0x1150C` 不在 IDA 已定義的函式內。
後續摘要程式曾因假設每筆都有函式名稱而報 KeyError；JSON 本體的完成狀態與證據仍有效。
「沒有函式邊界」不等於「沒有程式碼」，該段仍由有原始 bytes 的區間查詢保留。
匯出逐筆附加未知警示、原始定位與來源，不將工具名稱升格為語意。

### 18.3 單幀中文 HD 原型與待決取捨

從既有正常試玩 `workplace/shots29/maze.state` 以 `pwstep wait:17` 接續，
另存第 43,994,177 道指令的狀態。圖層匯出工具只讀該狀態，不推進或改寫遊戲。
HD 背景先疊，正式轉譯層匯出的獨立中文圖層最後疊。

| 原型 | HD 改動像素 | 中文不透明像素不符 | 原版已改動畫面像素被 HD 蓋掉 |
|---|---:|---:|---:|
| 8×8 原版像素格 | 145,353 | 0 | 0 |
| 逐像素 | 152,817 | 0 | 0 |

樣本是 `workplace/hd/redraw/normal-maze-hd-cell8.png` 與 `normal-maze-hd-cell1.png`，
收據為 `normal-maze-hd-verification.json`。原型程式在
`workplace/hd/hd_preview_layers.go` 與 `hd_preview.py`；入口與命令見 WORKLOG。
這只證明單幀的圖字順序與動態像素保護，不涵蓋正式執行期的跨幀失效、動畫、切換與存讀檔。

**待使用者決定**：8×8 格會多露出原版區塊，逐像素會留下較碎的文字周邊輪廓。
已提供實際原型並依 `grill-me` 提問；未將建議寫成已定案需求。024／204 仍為 DRAFT。

正式圖層審查另需處理以下既有草案缺口：

- 204 說逐格失效沿用 202，卻又要求原版改回來後自動恢復；既有 202 的透明格會永久保留，
  必須明定由哪個機制恢復，不能在實作時默默猜補。

- 既有 watcher 的 Owner 疊字是整筆失效，同一 Owner 有任何一筆存活就不重建。
  一個 watcher 產生多列圖面時，不能假設被移除的那一列會自行回來。
- 主題晚載入或讀檔時，不能把已被遊戲覆蓋的畫面當成 HD 圖的原始指紋；
  比對基準必須來自原版資料，避免首次登記就蓋住遊戲文字。
- Pix 的倍率必須有可驗證的紀錄；原始 RGBA 長度與 Draw 倍率的關係需完整定義。

【HD-LAYER-01】上述缺口後續以既有 API 的受控矩陣實跑量化，見 §23；
204 已回填 DRAFT 審查，不能將既有文字規則直接當成圖面的恢復契約。

現有背景重新驗證的結果是改動 153,375／576,000 像素，未授權黑色區域被佔用 0。
165 張非 ENEMY PNG 已用 ImageMagick 的像素平均值讀取全部解碼；紀錄是
`workplace/hd/redraw/non-enemy-png-decode-20260930.txt`。
以上均為素材或原型驗證，不把它們登錄成正式主題完成。

ENEMY00 #3–#5 的第二組重繪預覽也已產出，入口
`workplace/hd/redraw/ENEMY00-group1-preview.png`。第一版第三格手臂姿勢不符已排除，
修正版才保存三張 72×96；提示詞與雜湊見 `ENEMY00-group1-provenance.json`。
它未接入本節的正常重播，不把這六張預覽登錄成正式 ENEMY 批次完成。

## 19. ENEMY00 後續九格預覽與候選分組訂正

2026-09-30。原始輸入是 `ENEMY00.PBL`；檔案與解碼色號陣列 SHA-256、圖號、
完整提示詞及輸出雜湊分別保存在 `workplace/hd/redraw/ENEMY00-group2-provenance.json`
至 `group4-provenance.json`。定位基準是 **PBL 檔名／圖號**，不是程式位址。

### 19.1 新增預覽

使用內建 `image_gen`，以 #6–#8、#9–#11、#12–#14 的原版三格參照生成三組。
第二張輸入只作前組賽璐珞線條的風格參考，造型、配色與姿勢仍以各組原圖為準。
人工檢視排除兩個首版偏差：#6 的畫面右側手臂錯抬至頭頂；
#11 的眼睛仍是方塊輪廓，畫面左側手臂方向也不符。針對這些偏差重新編輯後保存。
這只屬視覺參照審閱，不是原版角色語意或正常動畫驗收。

**已證實（檔案與轉檔）**：三張選定圖集均為 1881×836，每格 627×836，
比例正好 3:4。在 `psychicwar-video:latest` 的 ImageMagick 內以
`-crop 3x1@ +repage -resize 72x96!` 切成九張 72×96；全數強制解碼通過。
原版與內建工具的原始生成檔均未被修改，產物 UID/GID 為 1000:1000。

並排預覽為 `workplace/hd/redraw/ENEMY00-group2-preview.png` 至 `group4-preview.png`。
ENEMY00 #0–#14 的原圖／HD 對照入口是 `ENEMY00-first15-comparison.png`：
由上往下五組，每列左三格是原圖、右三格是 HD。
五組共 15 格均為 **prototype**；正式 `art-in/ENEMY*.png` 仍為 0/360，尚未接入遊戲。
縮圖後輪廓、生成的局部造型與實際動畫一致性仍須驗證，不能從可解碼推論完成。

### 19.2 候選分組的證據限制

訂正 `tools/hd/enemy_groups.py` 先前「相鄰不同像素 <35% 即同角色」的敘述。
閾值只能證明兩張解碼圖像相近，無法證明原版如何將它們分配給角色或動作。
§15–§16 的舊說法保留作歷程，現況以 §17.1 的實際尺寸與本節限制為準。

工具仍使用原有閾值提出候選，每列明示 **⚠ 假說**，保留原始圖號、尺寸、
Python 版本、色號陣列基準及輸入檔案雜湊；缺輸入改為非零結束，避免靜默漏批。
`enemy_run.sh` 的提示詞也要求先看參照，不得因候選分組就合併不同角色。
本輪只在 Docker 內檢查 shell 語法，未執行該腳本的主機 `codex exec`。

**已證實（清單驗證）**：Python 3.13.15 重生 ENEMY00 清單，18 個候選組完整涵蓋
原始圖號 0–29；每列有假說警示，來源 SHA-256 相符。
不存在的原始輸入負對照以非零結束，沒有產出一份看似完成的空清單。

## 20. 正常 ENEMY 動畫的 XOR 差分與畫面原型

2026-10-01。輸入與 IDA 位址基準沿用 §17.2；未修改原版、正式資料庫或遊戲記憶體。
工具是 dosgolem `f8c1a6ec5f069058f163ce8ddf345e24dc851de6`、Go 1.24.13；
實際 `probe-hd-native` 的 Go 建置資訊相符且 `vcs.modified=false`。
以下執行期定位均為 **段:偏移**，不是 IDA ea。

### 20.1 有限差分核對

**已證實（限以下三次正常呼叫）**：以 §18 的同一初始狀態與十一個前進輸入，
在 `0161:8705` 進入前與 `0161:8751` 返回指令前取樣；沒有直接呼叫常式或座標注入。
獨立解碼 PBL 作畫面期望值，不以掛鉤訊號本身證明顯示正確。

| 進入／返回步數 | 原始來源 DS:BX／AL | 附加語意 | 等級與證據 |
|---|---|---|---|
| 83,190,318／83,214,859 | `1175:A686`／`00h` | 384 bytes 完整圖，畫面等於 ENEMY00 #3 | 已證實，限本筆來源與 24×32 畫面；`hd-enemy-xor.json` |
| 83,373,343／83,397,692 | `1175:A8C6`／`01h` | 384 bytes 等於 #3 XOR #4；前畫面 XOR 資料等於後畫面 #4 | 已證實，限本筆資料與畫面轉換；同上 |
| 83,556,097／83,580,446 | `0161:344A`／`01h` | 384 bytes 等於 #4 XOR #5；前畫面 XOR 資料等於後畫面 #5 | 已證實，限本筆資料與畫面轉換；同上 |

三個後畫面在 (32,152)、24×32 範圍內均與原始 PBL 圖逐色號相同，差異 0；
每次貼圖前後矩形外的畫面變動也為 0。忽略差分的負對照分別有 88／98 個像素不同。
重播終點仍是 `d0616fd57d1f7d078ea8abae989e6e901ff069f23890a0906e8cea845fb2e7d0`。
不是重新抽亂數或挑選通過結果；執行前載入的 state SHA-256 與 CS:41DF 種子
`86AFh` 沿用先前同一狀態的唯讀收據，並再次核對完整 state 雜湊。

原始檔、程式、工具與前後畫面雜湊都在 `workplace/probe/hd-enemy-xor.json`。
原始定位、AL 模式、步數與每項驗證結果也在同一收據；不把本次兩個 AL=1
的結果推廣成所有繪圖模式契約。全部角色、後續循環與 16×16 小圖塊用途仍未知。

【HD-SMALL-02】此處為2026-10-01的有限結論。後續§37已證實首場#18–#20來源；
§56新增敏頓正常路線ENEMY00 #21–#23的一般貼圖來源與位置證據，其他小圖塊仍未知。
不把這些來源升格為完整重疊、HD接入或能力語意驗收。

### 20.2 重生與負對照入口

`workplace/bin/probe-hd-native` 是既有 dosgolem `cmd/probe` 的建置產物。
在 `psychicwar-go-ebiten:latest` 容器內，專案掛 `/src`、原版掛 `/orig`（唯讀），
使用 `--rm`、目前 UID/GID、無網路及有界資源，於 `/src` 執行：

```text
workplace/bin/probe-hd-native -root /orig/psychic-war \
  -load-state workplace/states/07-first-play.state -steps 83600000 \
  -press up,up,up,up,up,up,up,up,up,up,up -press-at 43000000 -press-every 4000000 \
  -regs-at 0161:8705,0161:8751 -regs-max 3 > workplace/probe/hd-xor-return.log

workplace/bin/probe-hd-native -root /orig/psychic-war \
  -load-state workplace/states/07-first-play.state -steps 86000000 \
  -press up,up,up,up,up,up,up,up,up,up,up -press-at 43000000 -press-every 4000000 \
  -shots 83190318:workplace/probe/hd-xor-before0.frame,83214859:workplace/probe/hd-xor-after0.frame,83373343:workplace/probe/hd-xor-before1.frame,83397692:workplace/probe/hd-xor-after1.frame,83556097:workplace/probe/hd-xor-before2.frame,83580446:workplace/probe/hd-xor-after2.frame \
  -dump-mem-at '83190318:lin:1BDD6:384:workplace/probe/hd-xor-source0.bin;83373343:lin:1C016:384:workplace/probe/hd-xor-source1.bin;83556097:lin:4A5A:384:workplace/probe/hd-xor-source2.bin' \
  -dump-screen workplace/probe/hd-xor-normal-end.frame > workplace/probe/hd-xor-normal.log
```

驗證入口是 `tools/py.sh tools/hd/verify_enemy_xor.py`。它核對初始狀態、原版雜湊、
暫存器紀錄、三個獨立 PBL 期望值、差分資料、畫面外變動及原重播終點。
三份 `hd-xor-source*.bin` 在同一次重播、指定步數取樣，384 是十進位長度；
`lin` 後是執行期線性位址，與上述 DS:BX 可直接換算，非 IDA ea。
它們與 §18 的三份 `hd-enemy-*.bin` 逐 byte 相同，舊來源保留，兩次取樣雜湊可回查。
在 Docker `/tmp` 的獨立副本，改一個後畫面像素與一個差分來源 byte，
兩個負對照均以 1 結束且不寫通過收據。這不是 HD 執行期驗收。

### 20.3 HD 角色在正常畫面中的原型

三個正常後畫面以各自 `.pal` 的實際 256 色 RGB 陣列轉 PNG，整數放大三倍，
將既有 HD 三格依序放在 (96,456)、72×96。入口是
`workplace/hd/redraw/normal-enemy-frame3-hd-prototype.png` 至 `frame5-hd-prototype.png`。
原版色號與實際色盤獨立算出的角色矩形外差異均為 0，收據為
`normal-enemy-hd-prototype-verification.json`。

這是 **prototype**：原版英文顯示與背景保留，沒有接入中文圖層、正式主題、
跨幀失效、遊戲存讀檔或即時動畫；15 格素材仍不計入正式 ENEMY 完成數。
目前只讓角色位置與縮圖效果有具體玩家畫面可查，不宣稱中文 HD 遊戲已完成。

### 20.4 結論回查索引

| 不可變定位鍵 | 新證據與語意 | 舊入口及必要訂正標記 |
|---|---|---|
| DOS／`PW_UNP.EXE`（§17.2 雜湊）／`0161:8705`；同一固定 state 下的 `1175:A8C6`、`0161:344A` | 本節 §20.1，已證實兩筆 XOR 差分與正常畫面轉換 | §18.1 保留舊紀錄並補 `【HD-XOR-01】` |
| 同上；IDA ea `0x18C15`（base `0x10510`） | 第二版貼圖不能假設 DS:BX 永遠指向完整原圖 | `docs/spec/024` §8 補 `【HD-XOR-01】`；仍 DRAFT |

`verify_enemy_xor.py` 同時核對新證據保留原始定位、§18 的回查標記，以及舊規格的
訂正標記。缺任一入口就拒絕收據，不靠操作者記得回看。

## 21. ENEMY01 預覽與可重跑的中文遭遇戰原型

2026-10-01。本節是素材準備與 **DRAFT 單幀原型**，不代表正式主題接入完成。
路由沿用 `local/retro-remake-spec-gated-workflow.md`、文件職責與 §20 的回查契約。

### 21.1 新增十五格 ENEMY01 預覽

原版輸入 `workplace/original/psychic-war/ENEMY01.PBL`，SHA-256：
`22f664050ce7ffa4ea0f6941c9c91cd1ab43671ea5b53491f6799f78ba8e64af`。
定位基準是 PBL 原始圖號 #0–#14 及解碼色號陣列，沒有反組譯位址。
依相鄰三格提出五組候選；**角色身分與動作分配仍是假說**，沒有正常玩家映射證據。

每組用原圖作造型、配色與姿勢依據；ENEMY00-group1 只作賽璐珞線條參考。
內建 `image_gen` 生成後，以既有 `psychicwar-video:latest` 的 ImageMagick
等寬切三格，再縮為各 72×96。第零組初稿偏立體材質，另修平面色塊與線條；
初稿、修訂提示詞及選定結果的雜湊都保留，沒有以修訂結果抹除來源。

入口均在 `workplace/hd/redraw/`：

- `ENEMY01-group0-provenance.json` 至 `group4-provenance.json`：完整提示詞、
  參照角色、原版與解碼雜湊、選定圖及各格雜湊、轉換命令與限制。
- `ENEMY01-group{0..4}-frame-{00..02}.png`：十五格 72×96，Docker 內用
  Python 3.13.15／`tools/pbl.py` 完整解碼通過；檔案擁有者 UID/GID 1000。
- `ENEMY01-first15-comparison.png`：由上往下五組，左原圖、右 HD，已目視檢查。

**已證實的只有檔案、尺寸與可解碼性**。局部花紋、面部與服裝細節仍含生成解讀，
第五組細微動作的保持尚未驗證；不能宣稱造型忠實或動畫完成。
加上 ENEMY00 的十五格，目前共有 **30 格預覽，正式 ENEMY 仍 0/360**。
#15–#29 的 16×16 局部圖塊不猜補為完整角色。

### 21.2 正常中文遭遇戰的來源鏈

舊 `workplace/shots29/enc.state` 可用於查看畫面，但未找出完整產生鏈；
保留舊原型與收據，不單獨拿它宣稱正常玩家路徑完成。
本節重新由 `workplace/states/07-first-play.state` 經六次前進建立來源：

- 固定初始 state SHA-256：
  `56c489deaa57cd26fbd4f39f374e87db2f51e7d1275c440c9805321fb09bd532`。
- 執行前載入該狀態；CS:41DF 種子 `86AFh` 沿用同一狀態的唯讀收據，
  本次核對完整 state 雜湊，不重擲、改寫亂數或要求跨流程骰序一致。
- dosgolem `f8c1a6ec5f069058f163ce8ddf345e24dc851de6`、Go 1.24.13，cycles 750。
- 六次 `tap:Up:150,wait:300` 後 `wait:200`；未作弊、未直接進入戰鬥或注入座標。
- 終點 44,135,601 指令；七則轉譯事件都有正式文本資料，缺譯文事件 0。
- 獨立中文圖層匯出合成與 `pwstep` 實際截圖比較，RGB byte 不符 0。

在既有 `psychicwar-go-ebiten:latest` Docker 容器內，工作樹掛 `/src`、
原版父目錄掛 `/orig:ro`，於 `/src` 執行（所有容器仍須 `--rm`、目前 UID/GID、
無網路、資源上限及外層逾時；Go 快取沿用 `workplace/gocache`、`gomodcache`）：

```text
go run -mod=readonly ./cmd/pwstep -orig /orig/psychic-war \
  -load-state workplace/states/07-first-play.state \
  -do 'tap:Up:150,wait:300,tap:Up:150,wait:300,tap:Up:150,wait:300,tap:Up:150,wait:300,tap:Up:150,wait:300,tap:Up:150,wait:300,wait:200' \
  -scratch /tmp/pwstep-saves \
  -save-state workplace/hd/redraw/replay-encounter.state \
  -shot workplace/hd/redraw/replay-encounter-chinese-shot.png \
  -text-log workplace/hd/redraw/replay-encounter-text.jsonl

go run -mod=readonly workplace/hd/hd_preview_layers.go \
  -state workplace/hd/redraw/replay-encounter.state \
  -out workplace/hd/redraw/replay-encounter
```

第二支可丟棄工具只讀該 state 與同名 `.xlate.json`，不推進指令；
匯出原版 RGB、獨立中文 RGBA 與合成中文 PNG。來源、程式、字型、原版檔案
雜湊在 `replay-encounter-hd-verification.json`，狀態與畫面留在同一前綴。

### 21.3 HD 合成及反向對照

原型順序是 **HD 背景 → 已比對的左側角色 → 中文**。左側 (32,152)、24×32
色號陣列精確等於 ENEMY00 #4，再以既有來源紀錄的雜湊綁定
`ENEMY00-group1-frame-01.png`。右側角色仍原樣保留；有限搜尋沒有找到完整匹配，
不能據此斷言它在所有資料或姿勢中都無法識別。

【HD-ALLY0-01】後續擴大到全部 PBL 的限定區域搜尋，右側已精確匹配 ALLY #0，
並從更早的正常名字輸入路徑取得貼圖來源與前後畫面，見 §22。
本節舊原型與「只搜 ENEMY」的限制保留；新來源不會將舊原型追認為雙角色驗收。

在 `python:3.13-alpine` Docker 容器內掛工作樹 `/src`、原版唯讀，於 `/src` 執行：

```text
python workplace/hd/verify_replay_preview.py
```

該工具會重生兩張單幀原型、從原版色號獨立算動態保留範圍，並核對中文圖層：

| 原版粒度 | 相對中文截圖改動像素 | 不透明中文不符 | 角色外動態畫面不符 |
|---|---:|---:|---:|
| 8×8 格 | 153,439 | 0 | 0 |
| 逐像素 | 157,145 | 0 | 0 |

輸出是 `replay-encounter-hd-cell8.png`、`replay-encounter-hd-cell1.png`。
指定錯誤原圖 #3，或保留正確 #4 卻傳入 HD 第三動作格，兩個負對照都以 1
結束、訊息符合指定拒絕原因且沒有輸出，排除「其他異常也算拒絕成功」。
初次加護欄曾誤用另一份紀錄的 `frames` 欄位，造成 KeyError；修正為該組原有
`sha256` 格式後重新驗證，舊異常不列入有效負對照。

兩張圖都是 **單幀 DRAFT 原型**；沒有正式載入、跨幀失效、即時動畫、切換、
還原、存讀檔或音訊驗收。遮罩粒度仍待使用者選擇，024／204 不因此升為 READY，
也不擴張第一版 SCREEN／MENU 的範圍。

## 22. 右側 ALLY #0 的正常貼圖來源

2026-10-01。解決 §21 的右側來源未知，沒有改原版、IDA 資料庫或正式執行期。
使用 dosgolem `f8c1a6e`／Go 1.24.13 的既有 `probe-hd-native` 與
Python 3.13.15／`tools/pbl.py`。原版 EXE 雜湊及已建立的貼圖入口證據沿用 §17.2。
以下定位全是 **執行期段:偏移、線性位址或 PBL 檔案偏移**，不是 IDA ea。

### 22.1 完整圖匹配與首次貼圖

輸入 `ALLY.PBL` 的 SHA-256：
`c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219`。
全部 537 張圖中，435 張符合本次尺寸與非黑像素篩選；只搜尋右下區域的
完整矩形，包含原向及水平鏡像。唯一匹配是檔案偏移 `0x003E` 的 **#0**，
24×32、原向、(264,152)。篩選與座標範圍完整記在驗證收據，不外推為全畫面搜尋。

右側角色在 `07-first-play` 已存在；由迷宮開始六次前進不會重新畫它，
因而該次有限掛鉤觀察無事件。改用既有 `replay/title-to-first-save.json`
的正常名字輸入段，從 `06-name.state` 重播，不改掛鉤地址或猜測圖像座標。

| 原始定位與參數 | 附加語意 | 等級及證據 |
|---|---|---|
| `0161:8705`，步數 38,648,892；`DS:BX=1175:96C6`，`AL=00h`、`CX=4226h`、`DX=0304h` | 384 bytes 完整圖精確等於 ALLY #0 的打包色號資料（每像素 4 位元） | 已證實，限本筆；`hd-ally0-verification.json` |
| `0161:8751`，步數 38,673,433 | 返回前的 (264,152)、24×32 畫面等於獨立解碼的 ALLY #0 | 已證實，限本次前後畫面；同上 |
| 執行期線性 `0x1AE16`；PBL 檔案偏移 `0x003E` | 前者是 DS:BX 取樣位置，後者是原檔圖號 #0；不可混用 | 已證實，原始定位與換算見同一收據 |

來源 byte 不符 0，貼圖後原圖不符 0，貼圖矩形外變動 0。
正常名字重播終點與正式 `07-first-play.frame` 逐色號相同；後續中文遭遇戰的
右側仍等於 #0。這不證明角色身分、其他 ALLY 動作或 16×16 小圖用途。

固定 `06-name.state` SHA-256：
`ac8eec0bbc533b267782298c298ce4a60440d0568266e665c8a5badbf9e0ca4a`。
執行前載入 state，唯讀取得 `CS:41DF=86AFh`、步數 35,000,000；不推進或改寫亂數。
收據 `hd-rightactor-initial-seed.json`；主重播按鍵與時機完全沿用既有 JSON，沒有重擲。

### 22.2 重生與驗證入口

在既有 Go Docker 映像內，工作樹掛 `/src`，原版父目錄掛 `/orig:ro`，於 `/src`：

```text
go run -mod=readonly workplace/hd/inspect_state_seed.go workplace/states/06-name.state \
  > workplace/probe/hd-rightactor-initial-seed.json

workplace/bin/probe-hd-native -root /orig/psychic-war \
  -load-state workplace/states/06-name.state -steps 42000001 \
  -press 25,1E,17,enter -press-at 35500000 -press-every 500000 \
  -regs-at 0161:8705,0161:8751 -regs-max 200 \
  -dump-screen workplace/probe/hd-rightactor-name-end.frame \
  > workplace/probe/hd-rightactor-name.log

workplace/bin/probe-hd-native -root /orig/psychic-war \
  -load-state workplace/states/06-name.state -steps 42000001 \
  -press 25,1E,17,enter -press-at 35500000 -press-every 500000 \
  -shots 38648892:workplace/probe/hd-rightactor-before.frame,38673433:workplace/probe/hd-rightactor-after.frame \
  -dump-mem-at 38648892:lin:1AE16:384:workplace/probe/hd-rightactor-source.bin \
  -dump-screen workplace/probe/hd-rightactor-name-end.frame \
  > workplace/probe/hd-rightactor-source.log
```

資源與隔離契約沿用 §21：`--rm`、目前 UID/GID、無網路、CPU／記憶體／程序上限，
外層逾時。原版與研究輸入唯讀；只給明確輸出與快取可寫。
核對入口是 `tools/py.sh tools/hd/verify_ally0.py`；會獨立解碼、核對種子與原版雜湊、
來源參數、矩形內外畫面、正式終點及有限搜尋唯一性，完整原始定位與工具雜湊保存於
`workplace/probe/hd-ally0-verification.json`。忽略整次貼圖必須產生差異才算有效負對照。

【HD-ALLY0-01】舊入口 §21.3 與 `docs/spec/024` §8 同時回填來源；
驗證器缺任一訂正標記就拒絕收據。024 仍 DRAFT，第一版範圍不變。

在 Docker `/tmp` 的獨立輸入副本，改一個貼圖後像素或來源 byte，兩者都以 1
結束、原因符合指定錯誤且不寫通過收據。忽略整次貼圖的負對照差 **426 像素**。
紀錄為 `workplace/probe/hd-ally0-negative-controls.json`；未改原版輸入。

### 22.3 中文雙角色 HD 單幀原型

沿用 §21 的相同正常中文遭遇戰輸入與兩種背景遮罩，加入右側既有
`workplace/hd/art-in/ALLY-00.png`，在 (792,456) 畫成 72×96；中文最後疊上。
來源對位、原圖完整匹配與已選 HD 檔的 SHA-256 都必須通過，才允許合成。
沒有生成新 ALLY 圖或將素材準備追認為正式接入。

既有 ALLY 圖的批次規格、日誌、原參照及選定圖雜湊，保存於
`workplace/hd/redraw/ALLY-00-provenance.json`。日誌 `workplace/hd/log/ALLY.log`
可追到原生成結果及縮放命令，但沒有保存該張完整生成提示詞；**未知**，
不從批次摘要捏造。原版／重繪對照已查看，身形比例與細節仍待美術驗收。

Docker 重驗入口：

```text
python workplace/hd/verify_replay_preview.py --both
```

不加 `--both` 仍核對 §21 的舊單角色原型；兩套輸出與收據分開保存。
新產物前綴為 `workplace/hd/redraw/replay-encounter-hd-both`：

| 原版粒度 | 相對中文截圖改動像素 | 中文不符 | 雙角色 HD 不符 | 角色外動態不符 |
|---|---:|---:|---:|---:|
| 8×8 格 | 157,859 | 0 | 左 0／右 0 | 0 |
| 逐像素 | 161,565 | 0 | 左 0／右 0 | 0 |

角色期望值直接由兩張已選 HD PNG 及中文 RGBA 算出，逐像素檢查整個矩形，
不是只忽略角色區域就算通過。因此若程式漏畫角色，驗證會失敗。
四個負對照分別為錯誤 ENEMY 圖號、錯誤 ENEMY HD 格、錯誤 ALLY HD 圖、
被改壞的 ALLY 原始畫面；全部以指定原因拒絕、無 PNG 輸出。
完整來源與程式雜湊、結果在 `replay-encounter-hd-both-verification.json`。

新圖仍是 **DRAFT 單幀原型**。正式載入、跨幀失效、即時動畫、切換、存讀檔與
平台驗收未完成；遮罩粒度仍待使用者選擇，沒有代選或擴張第一版規格。

## 23. 圖層恢復、多列與首次基準的受控觀察

2026-10-01。這是 **既有 API 行為證據**，不是原版玩家路徑、HD 正式實作或 parity。
輸入為 16×16 人工色號矩陣、兩列、每列兩個 8×8 格；沒有原版執行、亂數或 RND 對拍。
工具為 dosgolem `f8c1a6e` 的現有 `xlate`、Go 1.24.13、Python 3.13.15。
模組定位採原始 Go 檔名及函式符號，沒有 DOS／IDA 位址；程式與規格雜湊在收據。

### 23.1 四個有限案例

`workplace/hd/probe_layer_recovery.go` 使用既有 `Add`、`Watch`、`Frame`、`Draw`、
`Snapshot`、`Restore`，不增加 `Art`／`Pix`，也沒有修改正式 xlate 程式。
每個階段保存人工色號、實際 RGBA 與疊字快照，核對器從原始輸入及繪圖 alpha
獨立算覆蓋範圍，不只依賴活列數或 Make 計數。

| 案例 | 輸入與實際結果 | 等級與限制 |
|---|---|---|
| 普通疊字部分格失效 | 第一格遮擋三幀，再恢復基準六幀；仍少 576 個放大像素 | 已證實，限既有 202 API；文字原本未承諾恢復，不能因此判為中文化故障 |
| 同 Owner 的兩列 watcher | 第一列失效，第二列存活；恢復後第一列沒補回，少 1,152 個放大像素 | 已證實，限此兩列案例；Make 仍只有一次 |
| 多列快照還原 | 在上述缺列狀態快照往返、重登記 watcher，再跑六幀；仍少 1,152 像素 | 已證實，限部分列存活的狀態；只保存 Owner 不足以證明圖面完整 |
| 單列 watcher 正對照 | 唯一一列失效後，基準恢復便重新 Make；完整恢復、缺像素 0 | 已證實，說明單列機制有恢復能力，不能把全部 watcher 都判為失效 |
| 比對區小於繪製區 | 只比第一列卻建立兩列；第二列第一格已被改變，初次定色仍蓋住 576 個非基準像素 | 已證實現有 API 允許此配置；若拿它畫 HD 背景，會缺完整繪製格的首次驗證 |

這些數字是三倍畫布的像素數，不是原版畫面的 576／1,152 個色號。
正式 `AttachBaked` 每個 watcher 只建立一筆文字疊字；本節沒有證明遊戲中的
中文 watcher 出現多列缺失。204 的每列一筆圖面，是需要額外契約的新用法。

原始符號與因果定位：

- `xlate/layer.go` 的 `(*Stamp).setTransparent` 只設 true；`(*Layer).Frame`
  跳過透明格，不再檢查是否恢復。沿用它不能達成 204 要求的「改回來恢復」。
- `xlate/watch.go` 的 `(*Layer).alive` 只需同 Owner 任一筆存在便回 true；
  `checkWatchers` 因而跳過 Make，單列被移除不會補回。
- `(*Watcher).match` 只驗證 Want 所涵蓋的矩形；Make 的額外覆蓋範圍由呼叫者負責。
  `(*Layer).Frame` 對新 Pending 疊字採當前畫面作指紋，並不自動證明它等於原始底圖。

### 23.2 重跑與反向對照

在既有 `psychicwar-go-ebiten:latest` Docker 容器內掛工作樹 `/src`，於 `/src`：

```text
go run -mod=readonly workplace/hd/probe_layer_recovery.go
```

容器仍須 `--rm`、目前 UID/GID、無網路、有界 CPU／記憶體／程序及外層逾時；
Go 快取沿用專案既有目錄。只寫 `workplace/probe/`，不掛原版輸入也不執行遊戲。
核對入口：

```text
tools/py.sh tools/hd/verify_layer_observation.py
tools/py.sh tools/hd/verify_layer_observation.py --require-hd-contract
```

第一個命令確認上述受控觀察，**不代表 HD 契約通過**。第二個要求圖面恢復契約，
目前以 1 結束並明示尚未通過；不是現有遊戲產品測試失敗。
在 Docker `/tmp` 的獨立 RGBA 副本，將本應透明的恢復格改為不透明，
觀察核對器以 2 拒絕、不寫通過收據，避免只相信自己記錄的狀態。

收據在 `workplace/probe/hd-layer-recovery-observation.json`、
`hd-layer-recovery-verification.json`、`hd-layer-recovery-negative-controls.json`。
各案例前綴 `hd-layer-<案例>-<階段>`，保存 `.idx`、`.rgba`、`.json`，
兩份工具以及文件職責與現行工作入口都掛在 CONTEXT。

### 23.3 DRAFT 回填與停止點

【HD-LAYER-01】已訂正 204「與 202 完全相同」的圖面敘述，保留原來的文字規則。
圖面的恢復基準、部分列完整性、首次繪製區驗證、倍率及 alpha 仍須在 READY 前明定；
本輪沒有選定新 API、快照格式、遮罩粒度或改寫正式圖層。
舊草案說法的追溯記錄在 `docs/re/000-overturned-claims.md`。

| 定位鍵 | 已證實語意 | 必須回查的舊入口 |
|---|---|---|
| dosgolem `f8c1a6e`／`xlate/layer.go`／`(*Layer).Frame`、`(*Stamp).setTransparent` | 普通疊字透明後不自動恢復 | `docs/spec/024` §9、dosgolem `204` 的 `【HD-LAYER-01】` |
| 同版本／`xlate/watch.go`／`(*Layer).alive`、`checkWatchers` | 任一同 Owner 列存活會阻止整組再 Make | 同上，研究 §18.3 回查本節 |

024／204 保持 DRAFT。正常玩家路徑、跨幀 HD 覆繪、切換、存讀檔與效能仍未驗，
不以人工矩陣取代它們；後續實作也須保留文字與圖面行為的界線。

## 24. ENEMY02 前十五圖的素材預覽（2026-10-01）

### 24.1 原始輸入與核對範圍

本節僅準備素材，不執行原版，不涉及亂數、CPU 位址或玩家動作映射。
來源是 `workplace/original/psychic-war/ENEMY02.PBL`，SHA-256：
`c38beb2879508466f0c316185eb7a489071279c34c2a89678bb1e80598de4c3f`。
與 §17 的原版清單 `enemy-input-hashes-20260930.json` 相同；原版全程唯讀掛載。

已證實：`tools/pbl.py` 解碼 #0–#14 皆為 24×32，#15–#29 皆為 16×16。
#12／#13 的解碼色號陣列完全相同，#14 不同。定位基準是 PBL 圖號及解碼色號陣列，
不是執行檔線性位址。逐圖雜湊保存在下列來源紀錄。

`tools/hd/enemy_groups.py /tmp/enemy02.md ENEMY02` 提出前五個三格候選；
**假說：連續圖號與像素相似不證明角色身分或動作語意。**檢視五張原版參照後，
以原圖可見形塊、構圖及配色準備研究預覽，未擅自命名角色或指定小圖塊用途。
本檔 #12–#14 也都是 24×32，沒有套用其他檔的 24×24 尺寸。

### 24.2 生成、轉換與來源入口

使用內建 `image_gen`，每組提供本組原圖拼圖，以及 ENEMY00 第 1 組生成圖作線條風格參照。
第二張只提供賽璐珞線條與陰影，不作角色造型來源。完整提示詞與參照角色已保存：

- `workplace/hd/redraw/ENEMY02-generation-jobs.json`：五組完整生成提示詞與參照路徑。
- `ENEMY02-selected.json`：內建工具保存的五張原始生成檔路徑。
- `ENEMY02-group0-provenance.json` 至 `group4-provenance.json`：每組原版、解碼圖、
  參照、生成圖、轉換腳本與輸出雜湊；每格附原始圖號。
- `ENEMY02-group0-reference.png` 至 `group4-reference.png`：原版三格放大參照。
- `ENEMY02-group0-generated.png` 至 `group4-generated.png`：原始生成結果的本機副本，
  每張 1881×836；內建工具保存的原檔保留。
- `ENEMY02-group<0–4>-frame-<00–02>.png`：15 格 RGB 預覽，各 72×96。
- `ENEMY02-group<0–4>-preview.png` 與 `ENEMY02-first15-comparison.png`：各組及全組對照。
- `process_enemy02.sh`、`verify_enemy02_previews.py` 與 `ENEMY02-preview-verification.json`：
  轉換、檔案核對與收據。以上短檔名皆位於 `workplace/hd/redraw/`，由 CONTEXT 掛入入口。

ImageMagick 6.9.11-60 Q16 將生成圖等分三格（各 627×836），再縮為 72×96。
#13 的輸出直接複製 #12，保留原版的重複圖關係；沒有採用生成圖第二格的細微差異。
核對使用 Python 3.13.15 與 `tools/pbl.py read_png`，檢查每個 PNG 區塊 CRC、完整解碼、
各列長度、尺寸、重複圖關係與來源雜湊。15/15 檔案核對通過，共 14 種不同預覽。
這是檔案契約核對，不是正式美術、同狀態或遊戲驗收，沒有據此升格 READY。

可重跑入口：在既有 `psychicwar-video:latest` Docker 執行
`sh workplace/hd/redraw/process_enemy02.sh`；預設拒絕覆寫已保存的生成圖與格。
重建時將 `ENEMY02_OUT_DIR` 指向容器內已存在的空輸出目錄；原版、參照與內建工具
生成目錄唯讀掛載。再於 `python:3.13-alpine`、`/src` 工作目錄執行
`python workplace/hd/redraw/verify_enemy02_previews.py` 核對正式保存的研究預覽。
兩者皆使用 `--rm`、UID/GID 1000、無網路、CPU 1、有界記憶體與外層逾時；
未在主機啟動 Python 或影像轉換。

### 24.3 看圖後的限制與下一閘門

| 候選組／圖號 | 預覽觀察 | 狀態 |
|---|---|---|
| 0／#0–#2 | 下方留白增加，底部構件位置與原圖不同 | 待修正 |
| 1／#3–#5 | 保留主要配色，但有立體斜面與輪廓詮釋 | 未通過造型審查 |
| 2／#6–#8 | 三格下方長條構件有差異，仍有立體表面詮釋 | 未通過造型與動作審查 |
| 3／#9–#11 | 增加臉部、手部與背後細節；頭頂第三格彎曲已有呈現 | 新增細節待修正，不能視為忠實完成 |
| 4／#12–#14 | 原圖黑白區被詮釋為眼睛；#12／#13 使用同一輸出 | 重複圖已保留，造型及角色種類未證實 |

目前 ENEMY00、ENEMY01、ENEMY02 各有 15 格研究預覽，合計 **45 格**。
本批沒有寫入 `art-in/`，正式 ENEMY 批次仍 **0/360**；正常玩家呈現、動畫一致性、
遮罩、存讀檔、效能及平台執行均未驗。本節不擴張 024 第一版 SCREEN／MENU 範圍。
024／204 保持 DRAFT，正式圖層分支仍待使用者選定 8×8 格或逐像素的遮罩取捨。

## 25. ENEMY02 局部修正與草案來源分類（2026-10-01）

### 25.1 第 0、3 組的修正候選

來源仍為 §24 的 `ENEMY02.PBL`，SHA-256
`c38beb2879508466f0c316185eb7a489071279c34c2a89678bb1e80598de4c3f`。
沒有原版執行、亂數、角色身分或動作映射的新結論；定位基準仍是 PBL 圖號。
內建 `image_gen` 以初版為修改目標、原版拼圖為形體與構圖參照，保留各版而未覆寫。

| 候選 | 核對結果 | 限制 |
|---|---|---|
| 第 0 組 v2，#0–#2 | 三格的底部黑色留白都由 13 列降為 0，與原圖貼底一致 | 整個機體輪廓、比例及色塊仍需審查，不由貼底數字證明忠實 |
| 第 3 組 v2，#9–#11 | 視覺上移除新增長翼片，爪狀指尖較簡化 | 白色眼縫仍在，不能將整次提示詞視為已完成 |
| 第 3 組 v3，#9–#11 | 再次只提示去除眼縫後，三格面板不再有白色眼縫 | 其餘形體忠實及姿勢對應未驗；生成不保證觀察區外逐像素不變 |

底部觀察以 72×96 輸出中 RGB 最大值大於 16 的像素為形體範圍。
這是測量留白的方法，**不是美術驗收門檻**；原版色號非 0 的形體延伸到最底列。
`ENEMY02-v2-before-layout.json`、`ENEMY02-v2-after-layout.json` 保留初版及 v2 的
每列占用數、邊界及對應檔案雜湊。

v2→v3 在固定臉部觀察矩形 `[20,26,33,10]` 外仍有 3,014／2,998／3,027 個
RGB 像素不同。此矩形只作觀察，沒有自動辨識臉部或批准逐像素修改範圍的含義；
結果不能用來宣稱只改了眼縫，也不等同原版／HD 的同狀態差異。
目前選入新對照的是第 0 組 v2、第 3 組 v3，其他三組沿用初版候選，皆尚未接入遊戲。

### 25.2 完整提示詞、逐版來源及重建入口

以下檔案皆在 `workplace/hd/redraw/`，入口另掛入 CONTEXT：

- `ENEMY02-v2-generation-jobs.json`：兩組完整修正提示詞與各參照角色。
- `ENEMY02-v3-generation-job.json`：眼縫局部修正的完整提示詞。
- `ENEMY02-v2-selected.json`、`ENEMY02-revisions-selected.json`：內建工具原始生成檔路徑。
- `ENEMY02-group0-v2-generated.png`、`ENEMY02-group3-v2-generated.png`、
  `ENEMY02-group3-v3-generated.png`：生成原檔的專案副本；工具保存的原檔也保留。
- 每個上述組／版的 `frame-00.png` 至 `frame-02.png`、`preview.png`、`provenance.json`：
  72×96 格、1296×576 拼圖，以及完整提示詞、原版圖號／解碼雜湊、修改目標、生成及輸出雜湊。
- `ENEMY02-first15-v2-comparison.png`：第一次修正的全組對照。
- `ENEMY02-first15-revised-comparison.png`：最新候選對照，1296×1440。
- `process_enemy02_v2.sh`、`process_enemy02_v3.sh`、`verify_enemy02_revisions.py`、
  `ENEMY02-revisions-verification.json`：重建與來源核對入口；未建立正式主題 manifest。

v2 生成尺寸為 1881×836，三格各 627×836。v3 為 **1882×836**，初次轉換因寬度
不可被 3 整除而停止；沒有錯尺寸輸出。後續以 `round(k×1882÷3)` 的整數邊界
`[0,627,1255,1882]` 切成 627／628／627 像素寬，完整保留輸入，再各自縮為 72×96。
部分產物重跑前先核對生成副本與來源相同，沒有覆寫其他候選或原版。

兩個轉換腳本直接讀專案保存的生成副本，重建不依賴內建工具快取仍存在。
在 `psychicwar-video:latest` 的 `/src` 工作目錄執行：

```sh
mkdir /tmp/enemy02-v2-rebuild /tmp/enemy02-v3-rebuild
ENEMY02_V2_OUT_DIR=/tmp/enemy02-v2-rebuild sh workplace/hd/redraw/process_enemy02_v2.sh
ENEMY02_V3_OUT_DIR=/tmp/enemy02-v3-rebuild sh workplace/hd/redraw/process_enemy02_v3.sh
```

這些命令只在 Docker 中執行。容器掛專案目錄唯讀，輸出放容器的 `/tmp`；
以 `--rm --network none --cpus 1 --memory 768m --pids-limit 96`、目前 UID/GID、
有界日誌及外層 60 秒逾時執行，不掛原版為可寫，也不掛主機 runtime。

九個修正候選格與最新對照圖在乾淨重建後，解碼的 RGB 像素全部相同。
**PNG 檔案位元組不相同**：ImageMagick 寫入的 `date:create`、`date:modify`、`tIME`
隨轉換時間改變；第 0 組首格的兩側像素 signature 同為
`a83839c722c7260f46a7aa33b715f0b3136ea5d9fe2910f4bba200e0a8c4d714`。
保存的檔案仍各有自己的 SHA-256，不把 RGB 重建一致說成完整 PNG 雜湊一致。

來源核對在 `python:3.13-alpine`、`/src` 工作目錄執行
`python workplace/hd/redraw/verify_enemy02_revisions.py`；專案研究輸出可寫、原版子目錄
另唯讀掛載。使用 `--rm --network none --cpus 1 --memory 384m --pids-limit 64`、
目前 UID/GID、有界日誌及外層 60 秒逾時。工具只載入既有核對器的 `sha`／`png` 函式，
不執行初版紀錄寫入；核對 CRC、完整解碼、尺寸、原版雜湊與初版 15 格未變。
其自有檔案及共用核對函式來源雜湊均保存於收據。

### 25.3 目前完成範圍

九個候選輸出涵蓋六個既有圖號，**不是新增九個完成圖號**。
ENEMY 預覽仍涵蓋 45 格，正式批次仍 0/360；#12／#13 仍共用原先相同輸出。
本輪解決已指出的底部留白、翼片與白色眼縫，沒有證明其餘造型、配色、動畫或
全角色覆蓋，也沒有新增正式動作語意。024／204 仍 DRAFT，遮罩取捨及正式接入未完成。

### 25.4 草案的製作來源與權利分類

【HD-RIGHTS-01】本日以真正主機 `gh auth status` 確認登入，再唯讀回查
[#34](https://github.com/wicanr2/psychic_war_cht/issues/34) 與
[#44](https://github.com/wicanr2/psychic_war_cht/issues/44)，兩者仍開啟。
#34 的最新留言仍是 2026-09-30 的可行性與 DRAFT，沒有新的遮罩或公開方式定案。

024 §2 原 `kind: redraw` 說明「新畫，可散布」容易將製作來源當作所有素材的權利證明。
已澄清為製作分類，並於 §7 明示細密美術及人物重繪的公開方式仍未定；
既有幾何結構元素的敘述不自動擴張到全部人物圖。
勘誤索引保留舊欄位說法；這不是新增法律判斷，也沒有選定公開方式或改變授權。
所有本輪生成圖、各版候選及原始素材仍留本機，沒有 Git、Issue 寫入或發行。

## 26. ENEMY03 前十五圖的研究預覽（2026-10-01）

### 26.1 原版來源與獨立參照核對

本節只準備本機素材，沒有執行原版、亂數或正式主題。
輸入 `workplace/original/psychic-war/ENEMY03.PBL` 的 SHA-256：
`ad6e8183bbf6c6fc258693b1f0ac726593feff5d052b5da18ac715cc2e63e5c1`，
與既有 `enemy-input-hashes-20260930.json` 相同。工具為 Python 3.13.15、
`tools/pbl.py` 及 ImageMagick 6.9.11-60 Q16；定位基準是 PBL 圖號、
檔案偏移及解碼色號陣列，不是 CPU／IDA 位址。

已證實：#0–#14 都是 24×32；#12–#14 的前 17 列全黑，放大三倍為 51 列。
`python tools/hd/enemy_groups.py /tmp/enemy03.md ENEMY03` 提出的前五組依序涵蓋
#0–#2、#3–#5、#6–#8、#9–#11、#12–#14。候選分組的角色身分及動作語意仍是
**假說**；第五組只依可見下半部形塊作圖，不補完整身體，也不命名其用途。

核對器直接解碼原版 PBL，再獨立比較快取參照 PNG 與三格放大拼圖：
15 張原版參照及 5 張拼圖的像素不符均為 0。這證明提供的參照正確，
不代表模型輸出的美術忠實。15 張預覽均為 72×96，CRC 與完整 PNG 解碼通過。

### 26.2 生成紀錄與重跑入口

使用內建 `image_gen`。前四組附 ENEMY00 第 1 組作線條與陰影參照，
角色造型仍只取本組原圖；第五組只附原圖，避免借用其他角色的完整身體。
所有檔案位於既有 `workplace/hd/redraw/`，下列清單是本批入口：

- `ENEMY03-generation-jobs.json`、`ENEMY03-selected.json`：五組完整提示詞、
  參照及內建工具原始生成路徑。
- `ENEMY03-group0-reference.png` 至 `group4-reference.png`、同前綴的
  `generated.png`、`frame-00.png` 至 `frame-02.png`、`preview.png`：原圖、生成副本與預覽。
- `ENEMY03-group0-provenance.json` 至 `group4-provenance.json`：原版檔案偏移、
  色號陣列、參照與輸出雜湊、完整提示詞、轉換與核對工具版本／雜湊。
- `process_enemy03.sh`、`verify_enemy03_previews.py`、`ENEMY03-preview-verification.json`：
  初版轉換、獨立參照核對及收據。
- `ENEMY03-group4-v2-generation-job.json`、`ENEMY03-group4-v2-provenance.json`、
  同前綴的生成副本／三格／預覽：第五組局部修正與逐版來源。
- `process_enemy03_v2.sh`、`verify_enemy03_revision.py`：修正版轉換及檔案核對。
- `ENEMY03-first15-comparison.png`、`ENEMY03-first15-revised-comparison.png`：
  每列左三格原版、右三格 HD；後者只選用第五組 v2，其餘四組仍為初版。

轉換在既有 `psychicwar-video:latest` Docker 中執行：

```text
sh workplace/hd/redraw/process_enemy03.sh
sh workplace/hd/redraw/process_enemy03_v2.sh
```

工作樹掛 `/src`、工作目錄 `/src`。從專案保存的生成副本切分，
邊界採 `(k×寬+1)÷3` 的整數值，以涵蓋不能整除三的實際寬度。
初版腳本拒絕覆寫已存在的格；乾淨重建時先在容器建立 `/tmp/enemy03-rebuild`，
再加 `ENEMY03_OUT_DIR=/tmp/enemy03-rebuild` 執行，專案掛唯讀。
PNG 排除 `date`、`time` 區塊；本批 15 格、5 張預覽及 1 張對照，共 **21 檔**
在相同映像乾淨重建後完整位元組相同。此結果限本批及相同工具，
不回溯宣稱 §25 的舊 PNG 已可逐位元組重建。

核對在既有 `python:3.13-alpine` Docker 中執行：

```text
python workplace/hd/redraw/verify_enemy03_previews.py
python workplace/hd/redraw/verify_enemy03_revision.py
```

核對器沿用受雜湊記錄的 ENEMY02 `sha`／`png` 函式，僅載入函式，
不執行舊紀錄寫入。原版子目錄唯讀，只有研究輸出可寫；容器使用目前 UID/GID、
`--rm --network none --cpus 1`、相稱的記憶體／程序／日誌上限與 60 秒外層逾時。

### 26.3 修正結果與限制

第五組初版上方留白為 48／49／46 列，比原版放大的 51 列短。
只提供初版修改目標與原版參照，要求校正上部位置；v2 實測為 **52／53／51 列**。
第三格符合此項量測，前兩格仍多出 1／2 列，未宣稱全部構圖已修正。
量測以 RGB 最大值 >16 為可見形塊，不能取代造型審查。
初版十五格雜湊均未變，三個修正版只涵蓋既有 #12–#14，不增加完成圖號。

看圖可確認主要配色與部分三格差異，但細部輪廓、陰影、臉部及動作仍有模型詮釋；
第五組細密尖形塊的忠實度也未驗收。預覽現在涵蓋 ENEMY00–ENEMY03 各前十五圖，
共 **60 個圖號**，正式批次仍 **0/360**。未寫入 `art-in`、manifest、正式執行期或發行包。
024／204 仍 DRAFT，遮罩粒度待使用者回答，公開方式仍未定。

## 27. ENEMY04 前十五圖與手繪風格修正（2026-10-01）

### 27.1 來源及核對範圍

輸入 `workplace/original/psychic-war/ENEMY04.PBL` 的 SHA-256：
`81cb62cf9a8b64a538d09e50b29cbf8ad123b39b6e86979af6423a2aed20f546`。
與既有 `enemy-input-hashes-20260930.json` 一致，原版唯讀掛載。
`tools/pbl.py` 解碼確認 #0–#14 都是 24×32，前五組各涵蓋三個連續圖號；
角色身分及動作語意仍是**假說**，不用模型判定補成原版事實。
定位採 PBL 圖號、檔案偏移及解碼色號陣列，不是 CPU／IDA 位址。

用 Python 3.13.15 直接解碼 PBL，獨立核對十五張快取原版參照及五張放大拼圖，
像素不符均 0。ImageMagick 6.9.11-60 Q16 將五張生成圖切成十五格 72×96；
尺寸、CRC 與完整 PNG 解碼通過。核對不涵蓋美術品質、原版動畫或正常玩家呈現。

### 27.2 初稿與兩組風格修正

內建 `image_gen` 以本組三格原版為唯一造型、位置及配色來源，
ENEMY00 第 1 組僅提供賽璐珞線條與陰影風格。五張初稿皆為 1881×836。
看圖後第 0、1 組仍帶大面積像素階梯，沒有達到已定案的手繪 HD 方向。
初稿保留，再提供修改目標、原圖及風格圖，作一次線條與色塊邊緣修正。

v2 的主要外輪廓已改成連續曲線，原版可見的棋盤配色區仍保留；
這是視覺觀察，不能證明造型或位置逐像素不變。第 0 組第三格上方留白
由初版 5 列變為 4 列，原圖放大應為 0，**構圖差異仍存在**。
其餘三組仍有切面式線條與細部詮釋，風格一致性及三格動作也未完成審查。
初版十五格雜湊均未變，六個修正格仍為 72×96，完整解碼與尺寸核對通過。

後續直接以原圖及風格參照重繪的候選與裁切核對見 §28；本節保留前輪的結果與限制。

### 27.3 本機產物與重跑入口

下列檔案都在既有 `workplace/hd/redraw/`，不在 Git 或公開包：

- `ENEMY04-generation-jobs.json`、`ENEMY04-selected.json`：五組完整提示詞、
  參照路徑及內建工具的原始輸出路徑。
- `ENEMY04-group0-reference.png` 至 `group4-reference.png`，各組同前綴的
  `generated.png`、`frame-00.png` 至 `frame-02.png`、`preview.png`：初稿及原版參照。
- `ENEMY04-group0-provenance.json` 至 `group4-provenance.json`：逐圖來源偏移、
  原版解碼／參照／生成／輸出雜湊、提示詞、轉換及核對工具版本與雜湊。
- `ENEMY04-v2-generation-jobs.json`、`ENEMY04-v2-selected.json`：兩組修正提示詞與輸出路徑。
- `ENEMY04-group0-v2-provenance.json`、`ENEMY04-group1-v2-provenance.json`，
  及各組同前綴的生成副本、三格、預覽：修改目標、原圖與風格圖的用途和逐版來源。
- `process_enemy04.sh`、`process_enemy04_v2.sh`：初稿／修正版轉換；
  `verify_enemy04_previews.py`、`verify_enemy04_revisions.py`：來源及完整檔案核對。
- `ENEMY04-preview-verification.json`、`ENEMY04-revisions-verification.json`：
  初稿與修正版收據，明示正式批次仍 0/360。
- `ENEMY04-first15-comparison.png`、`ENEMY04-first15-revised-comparison.png`：
  每列左三格原圖、右三格候選；後者選第 0、1 組 v2，其餘沿用初稿。

在既有影像 Docker 掛專案於 `/src`、工作目錄 `/src` 執行：

```text
sh workplace/hd/redraw/process_enemy04.sh
sh workplace/hd/redraw/process_enemy04_v2.sh
```

來源直接讀專案保存的生成副本，不依賴工具快取；使用實際圖寬按
`(k×寬+1)÷3` 的整數邊界切分，PNG 排除日期區塊。拒絕覆寫已存在的格，
乾淨轉換可先在容器 `/tmp` 建立輸出目錄，設定 `ENEMY04_OUT_DIR`。
本批未另宣稱完整 PNG 已通過乾淨重建比對。

在既有 Python Docker 的 `/src` 執行：

```text
python workplace/hd/redraw/verify_enemy04_previews.py
python workplace/hd/redraw/verify_enemy04_revisions.py
```

沿用受雜湊記錄的 ENEMY02 PNG 核對函式，只載入函式，不執行舊紀錄寫入。
容器均 `--rm --network none --cpus 1`、目前 UID/GID，有界記憶體、程序、日誌及
60 秒外層逾時；原版與原始生成輸入唯讀，明確研究輸出可寫。

預覽目前涵蓋 ENEMY00–ENEMY04 各前十五圖，共 **75 個圖號**；
這是來源涵蓋數，**沒有任何一批因此通過正式美術與動畫驗收**。
六個修正格不增加涵蓋數。正式 ENEMY 仍 0/360，024／204 仍 DRAFT；
遮罩粒度 pending，未寫入正式 `art-in`、manifest、執行期或發行包。

## 28. ENEMY04 的直接重繪候選與裁切核對（2026-10-01）

### 28.1 未採用的局部修正與量測基準

來源沿用 §27 的 ENEMY04.PBL 及其 SHA-256，定位仍是原版圖號、PBL 檔案偏移
與解碼色號陣列；沒有執行原版、亂數或修改正式程式。
工具沿用內建 `image_gen`、ImageMagick 6.9.11-60 Q16、Python 3.13.15。

初次追加的第 2–4 組 v2 仍保留許多切面，第 0 組 v3 則在第三格頂部新增了
原圖沒有的橫杆。**第 0 組 v3 不採用**：可見形塊上方留白量得 0，不能據此判定構圖正確。
與 v2 比較，三格 RGB 改變 5,620／5,757／5,398 像素；
固定第三格左上觀察矩形 `[0,0,6,18]` 外，仍改變 5,620／5,757／5,335 像素。
前兩格全部算作區外。提示詞「其餘不變」沒有達成，不把此矩形稱為批准覆繪區。

原版 #2 前兩列的最左一像素是色號 04h，接下來兩列最左兩像素同色。
v2 原生生成圖第三格在 y=0、1、10、20、30 都沒有 RGB 最大值 >16 的形塊，
y=40 才在 x=0–2 出現；上端缺少紅條存在於生成圖，不能只歸因於縮放。
72×96 版前六列 RGB 最大值依序是 **0、0、1、4、30、150**。
此前「上方留白 4 列」採 >16 的可見形塊判準；只有前兩列是全零，
不將微弱濾波值也稱為全黑。這些量測限定為構圖觀察，不是美術忠實度門檻。

### 28.2 從原圖直接重繪及最新候選

連續兩次修正未解決切面後，重查知識路由並讀取 `imagegen` 的
`references/prompting.md`，縮短為分段提示詞。先以第 3 組原圖與已修正第 1 組的
風格圖直接重繪，不沿用切面初稿作修改目標；結果的輪廓及陰影較連續。
再同法處理第 0、2、4 組。風格圖只提供線條及平塗，不借用角色身分或形狀。

最新對照採用下列版本，全部仍為**未驗收的研究候選**：

| 候選組／原版圖號 | 最新版本 | 本輪可確認的範圍與限制 |
|---|---|---|
| 0／#0–#2 | v4 | 三格可見形塊上方留白均 0，保留左上紅條且無獨立橫杆；整體形狀仍需審查 |
| 1／#3–#5 | 前輪 v2 | 沿用已保存的曲線版本，沒有本輪重畫 |
| 2／#6–#8 | v3 | 大輪廓及色區較接近手繪，保留中段色帶差異；細部造型未驗 |
| 3／#9–#11 | v3 | 改為連續線條與較大的陰影色塊；姿勢及輪廓忠實度未驗 |
| 4／#12–#14 | v3 | 保留三格長形塊的彎曲差異；比例及姿勢仍待審查 |

生成工具的服務端版本未提供；不保證重送同一提示詞能得到相同圖。
可重跑的轉換以專案保存的原始生成 PNG 為輸入。新候選共 24 格，
涵蓋八個版本、十二個既有圖號，沒有增加來源涵蓋數。
全部尺寸、CRC、完整 PNG 解碼通過；初版 15 格及前輪 v2 六格雜湊均未變。

### 28.3 產物及 Docker 入口

所有新增檔案都留在既有 `workplace/hd/redraw/`：

- `ENEMY04-more-generation-jobs.json`、`ENEMY04-more-selected.json`：
  初次四組追加修正的完整提示詞、參照用途及工具原始輸出路徑。
- `ENEMY04-group3-v3-generation-job.json`、`ENEMY04-group3-v3-selected.json`：
  第 3 組直接重繪提示詞及輸出路徑。
- `ENEMY04-direct-generation-jobs.json`、`ENEMY04-direct-selected.json`：
  第 0／2／4 組直接重繪提示詞及輸出路徑。
- 第 0 組 v3／v4、第 2／3／4 組 v2／v3 各有
  `ENEMY04-group<組>-<版本>-generated.png`、三格 `frame-00.png` 至 `frame-02.png`、
  `preview.png` 與 `provenance.json`；逐版來源保存原始偏移、解碼與輸出雜湊、
  完整提示詞及工具版本／雜湊，並附視覺採用狀態。
- `ENEMY04-more-review.json`、`ENEMY04-more-verification.json`：
  分別保存視覺審查及來源／檔案核對，兩者不混為完成驗收。
- `process_enemy04_more.sh`、`process_enemy04_group3_v3.sh`、`process_enemy04_direct.sh`：
  三批轉換入口；`verify_enemy04_more.py` 核對全部八個新增版本。
- `assemble_enemy04_refined.sh`、`ENEMY04-refined-selection.json`：
  最新十五格來源選擇、組裝入口與輸出雜湊。
- `ENEMY04-first15-refined-comparison.png`：1296×1440，每列左三格原版、右三格候選。
- `ENEMY04-first15-refined-cycle.gif`：432×1440、三個完整畫格，左原圖、右候選。

轉換及組裝在既有影像 Docker 的 `/src` 執行相應 `sh workplace/hd/redraw/<腳本>`；
工作樹或明確研究輸出可寫，原版參照目錄另唯讀。核對在既有 Python Docker 的 `/src` 執行：

```text
python workplace/hd/redraw/verify_enemy04_more.py
```

容器一律 `--rm --network none --cpus 1`、目前 UID/GID、相稱記憶體／程序／日誌上限，
外層 60 秒逾時；原版全程唯讀。轉換拒絕覆寫舊格，使用實際圖寬切分，
可先在容器 `/tmp` 建立乾淨目錄，設定 `ENEMY04_OUT_DIR` 重建。

輪播只按三個候選圖號順序播放，每格 32 個百分之一秒，每圈 0.96 秒。
三格完整解碼、尺寸、像素平均值及延遲資料已讀取；這個速度是研究設定，
沒有原版時序證據，也沒有正常遊戲動畫、中文覆繪或存讀檔驗收。
GIF 調色盤只是看圖產物，不作原版逐像素比較收據。

ENEMY 研究來源涵蓋仍為 **75 個圖號**，正式批次仍 **0/360**。
024／204 保持 DRAFT，遮罩粒度仍待回答；不將風格改善或檔案通過當成整體 HD 完成。

## 29. 正常 F3 路徑的背景恢復範圍

2026-10-01。本節補第一版 SCREEN／MENU 的跨幀驗收依據，不生成新美術或改正式圖層。
路由載入規格閘門、文件職責及 dosgolem README，沿用 `grill-me` 的決策閘門。
原版操作使用既有 dosgolem 分支 `f8c1a6e`、Go 1.24.13；核對使用 Python 3.13.15。
開跑主機負載 17.02／13.95／9.58，容器限 CPU 1。本節只做決定性機器時間重播，
沒有音訊、幀率或牆上速度量測，不把高負載結果當即時效能證據。

### 29.1 固定狀態與正常輸入

從 §21 的正常六次前進終點 `workplace/hd/redraw/replay-encounter.state` 開始：

- state SHA-256：`782405fd3ca55e61fafd3daa294b299911f1e7984199b62e71ee79f410ccd2b8`。
- 執行前唯讀觀察 `0161:41DF`，種子 **CAF0h**，初始指令數 **44,135,601**。
  這是本次遭遇終點的種子，不是更早 `07-first-play` 的 86AFh；沒有重擲或改寫。
- cycles 750；正常分支按住 F3 150 ms、等 500 ms，再等 2,000 ms。
  對照分支不按鍵等 650 ms，再等 2,000 ms。兩側從完全相同的 state 及種子開始。
- 正常分支終點 **46,108,635** 指令，畫面已回到迷宮且方向為南；
  對照終點 **46,123,101**，仍留在戰鬥。兩個原版終點有 **2,969** 個色號不同。
  不要求不同操作造成的結束種子或指令數相同。

首次嘗試 `pwstep -do 'tap:F3:150,wait:500'` 被既有 `oracle.ParseActions` 拒絕：
它的鍵名表不含 F3，也沒有任意十六進位掃描碼的語法。這是探針輸入格式限制，
不是原版不支援 F3。保留正式工具，從 `cmd/pwstep/main.go` 衍生可丟棄的
`workplace/hd/replay_escape.go`，僅為這條固定輸入建立 `ActionTap`、掃描碼 **3Dh**。
掃描碼及原版用途沿用 `docs/re/036`；正式前端、原版程式和動作解析器未修改。

最初 650 ms 終點尚在畫面重繪途中，不能拿來宣稱已回到迷宮。
保留中途 state、原版畫面、中文圖層及截圖；後續再等 2,000 ms 後才採用完整終點。
本節的等待是固定機器時間，不調牆上時間、CPU 頻率或亂數狀態。

### 29.2 背景基準與有限恢復觀察

直接由原版 PBL 重建背景：SCREEN 五張各 320×40，依圖號鋪在 (0,0/40/80/120/160)，
再於 (160,4) 貼 MENU #0，88×72；與既有 `bg.idx` 的 **64,000 個色號不符 0**。
來源及原檔偏移逐項記在 `normal-recovery-20261001.json`：

- `SCREEN.PBL` SHA-256：`302a9724ddbf1872cf6b6ccdad1e2611e0c94663e61b40cb1ee46322d78ddbd1`；
  #0–#4 的原檔偏移為 `000Ah`、`1357h`、`25EFh`、`343Ah`、`3C8Dh`。
- `MENU.PBL` SHA-256：`b4ef63944ca7a2f37fbd459356e064cab3f8d285296fc701d362ff99aed89641`；
  #0 原檔偏移 `0002h`。
- 上述偏移是 PBL 檔案偏移；遮罩位置是 320×200 原版畫面座標。
  種子是執行期段:偏移，不混為 IDA 線性位址。

【HD-RECOVERY-NORMAL-01】**已證實，限這條正常操作與保存的畫面**：
每張畫面獨立與原版背景比較；完全相同的格可顯示 HD，其他格露出原版，中文最後合成。

| 粒度 | 由遭遇到脫離後恢復的原版像素 | 其中非黑背景像素 | 新增遮蔽像素 | 不按鍵對照的恢復像素 | 永久遮格模型少畫的 HD 像素 |
|---|---:|---:|---:|---:|---:|
| 8×8 | 1,216 | 47 | 3,136 | 0 | 423 |
| 逐像素 | 549 | 0 | 2,278 | 27 | 0 |

8×8 的 47 個非黑背景像素位於原版 (48,192)、(56,192) 兩個格，屬 SCREEN #4。
如果故意讓遭遇畫面失效的格永遠不恢復，脫離後原型少畫 **423 個放大後 RGB 像素**。
這個錯誤消費者是受控模型，不是正式 `xlate` 或已接入 HD 的遊戲。
正確的兩個原型皆有不透明中文不符 **0**、原版動態畫面不符 **0**。

逐像素這次恢復的都是原版黑底，因此錯誤模型沒有可見 HD 差異。
**這條樣本不足以驗逐像素的非黑圖面恢復**，不能把數字 0 當成該契約完成，
也不因此替使用者選定遮罩。正式圖層、多列、首次基準、切換、存讀檔與效能仍待驗。

原版 RGB 匯出逐色號核對 `.frame`，獨立中文合成與實際 `pwstep` 截圖相同。
改壞一個原版畫面像素的負對照以指定原因「原始畫面與 RGB 匯出不符」拒絕，
不寫通過收據；記錄為 `normal-recovery-negative-20261001.json`。

### 29.3 產物、入口與回填

以下研究入口均位於本機 `workplace/hd/`，未加入正式封包：

- `replay_escape.go`、`replay_escape_run.sh`：固定 F3 的可丟棄輸入探針及重生腳本。
- `verify_normal_recovery.py`：由原版 PBL、正常原版畫面及中文圖層獨立核對；
  `--escape-frame`／`--receipt` 可指定負對照輸入與輸出。
- `redraw/replay-escape-initial-seed-20261001.json`：本次執行前固定 seed 收據。
- `redraw/replay-{escape,noescape}[-settled]-20261001.*`：四個終點的 state、
  `.xlate.json`、原版色號、原版 RGB、中文 RGBA、中文合成、實際截圖與轉譯紀錄。
  同名前綴 `-metadata.json` 保存 state 雜湊、指令數及唯讀種子觀察。
- `redraw/normal-recovery-20261001.json`：來源、工具／輸入雜湊、原檔偏移及數值結果。
- `redraw/normal-recovery-20261001-cell8.png`、`cell1.png`：脫離後的 HD 背景原型，
  動態角色維持原版；`permanent-hole-cell8.png`／`cell1.png` 是故意錯誤的對照。
- `redraw/normal-recovery-negative-20261001.json`：被改壞畫面的拒絕結果。

在 §21 的既有 Go Docker 掛載與快取設定中，於 `/src` 執行：

```text
sh workplace/hd/replay_escape_preserved.sh /out
```

再於既有 Python Docker 的 `/src` 執行：

```text
python workplace/hd/verify_normal_recovery.py
```

容器使用 `--rm --network none`、目前 UID/GID、CPU 1、程序及日誌上限；
Go 記憶體 2 GiB／外層 120 秒，Python 768 MiB／60 秒。所有原版素材唯讀。
現行保全入口的 `/out` 掛載及核對方式見 §30；舊 `replay_escape_run.sh` 保留為
歷史工具來源，直接使用會覆寫研究輸出，不再作新重播入口。不在主機直接跑探針腳本。
來源回填到 024 §9 與 dosgolem 204 §2.6，同時保留 §23 的既有 API 人工矩陣。
兩份正式規格仍 DRAFT，第一版範圍不變，沒有新增正式 API、HD manifest 或執行期實作。

### 29.4 保存腳本重生的實際範圍

以保存的 `replay_escape_run.sh` 在同一 Go 映像與相同固定初始 state 重生四個終點。
**29 個非 state 產物完整位元組相同**：原版色號／RGB、中文圖層、中文合成、
實際截圖、轉譯紀錄與初始種子收據。四個終點的指令數、觀察種子及其他中繼資料相同。

四個 `.state` 的完整 SHA-256 不同；四份中繼資料只有 `sha256` 欄位隨之改變。
初次核對錯把所有 37 檔都應位元組相同當成完成條件，因此拒絕；這項拒絕保留為驗證限制，
**不宣稱四個完整機器狀態或 RAM 相同**。`worktrees/dosgolem/internal/state/state.go`
證實格式是 gzip 壓縮的機器與 DOS 兩段 gob，序列化差異的完整原因仍未知。
此處沒有改保存格式、亂數或正式存檔，也沒有把這次差異當作 HD 產品故障。

首次原始 state bytes 已被重生覆寫，只能保留首次收據的雜湊與觀察資料，
無法補做舊／新 state 原始 bytes 差分。首次收據另存
`workplace/hd/redraw/normal-recovery-first-verification-20261001.json`；
重生比較及差異分類另存 `normal-recovery-replay-verification-20261001.json`。
重生後以現行 state／中繼資料重新執行同一 `verify_normal_recovery.py`，
結果維持 §29.2，現行 `normal-recovery-20261001.json` 的雜湊對應目前產物。
這次重生不升格成完整同狀態或正式 HD 驗收。

## 30. 正常重播的證據保全入口

2026-10-01。修正 §29.4 已發生的研究證據覆寫問題。路由命中文件職責與規格閘門；
本節只改本機研究工具，正式 HD 與原版程式維持原樣。

### 30.1 唯讀來源與獨立輸出

`workplace/hd/replay_escape_preserved.sh` 必須在既有 Go Docker 執行，工作樹掛 `/src:ro`、
原版父目錄掛 `/orig:ro`，全新空研究目錄獨立掛 `/out` 可寫，Go 快取另掛可寫。
它在 `/out` 建立最小模組入口，以唯讀來源的符號連結提供 Go 原始碼、文本、字型及固定 state；
四個正常終點與其原始 bytes 都寫入該次專用目錄。

舊 `replay_escape_run.sh` 的內容與雜湊保留，不改寫既有收據的工具來源。
新入口在隔離工作目錄呼叫它，所以相同的相對輸出路徑不再指向原來的證據。
`/out` 有任何既有項目時即以 **2** 拒絕，原因為「拒絕覆寫：/out 必須是全新空目錄。」
完成四個終點後才寫 `status.txt`；部分輸出不會被當成完成。

### 30.2 已執行的重生與核對

本次輸出目錄：`workplace/hd/replay-run.lp7nvs4c/`，屬既有 `workplace/hd/` 的單次研究輸出。
原始固定遭遇 state、CAF0h 種子、cycles 750 及正常 F3／不按鍵操作沿用 §29，沒有重擲。
Go 映像 `psychicwar-go-ebiten:latest` ID
`083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7`，Go 1.24.13；
Python 3.13.15 映像 ID `540c7d91f98ff6880174c40e99067bf5941eb54d818a7a5e094d188b196a934d`。
開跑負載 0.57／3.20／7.09；CPU 1，沒有即時音訊或速度量測。

**已證實，限本次隔離重播及證據保全**：

- 舊收據的 **57 個輸入**與 **4 個 HD 原型**全部保持原雜湊。
- 新產生的 **29 個非 state 檔**與 §29 完整位元組相同。
- 四個新終點的指令數、觀察種子及其他中繼資料相同；四個新 state 雜湊均不同，
  其原始 bytes 及新雜湊完整保留，不宣稱完整機器狀態或 RAM 相同。
- 對已完成 `/out` 再呼叫重播入口，以指定原因及結束碼 2 拒絕；
  **37 個新輸出檔**與全部舊收據輸入未被更動。

核對入口是 `workplace/hd/verify_preserved_replay.py`。
本次收據位於 `workplace/hd/replay-run.lp7nvs4c/verification.json`，包括工具來源、
新 state 與所有輸出雜湊、終點觀察值及拒絕覆寫負對照。
同一目錄的 `source-sha256.txt`、`status.txt` 記錄實際來源及完成標記；
四個正常終點位於該目錄的 `workplace/hd/redraw/`，不是正式交付或新增素材完成數。

後續重跑先在既有 Python Docker 中於 `workplace/hd/` 建立空的 `replay-run.*` 研究目錄，
確認來源路徑存在且 UID/GID 正確後，才把該目錄單獨掛成 `/out`。
Go 容器於 `/src` 執行：

```text
sh workplace/hd/replay_escape_preserved.sh /out
```

Python 容器也用工作樹 `/src:ro`、原版 `/orig:ro`、該次研究目錄 `/out` 可寫，於 `/src` 執行：

```text
python workplace/hd/verify_preserved_replay.py /out
```

容器一律 `--rm --network none`、目前 UID/GID、CPU 1 及程序／日誌上限；
Go 2 GiB／外層 120 秒，Python 512 MiB／60 秒；沿用已存在且擁有權正確的快取。
新入口、輸出目錄與核對器是本機研究工具，不把它們接進正式 `xlate`。
024／204 仍 DRAFT，遮罩取捨仍待使用者回答。


## 31. 8×8 定案與原版排版的前後修正

2026-10-01。使用者依序回答「8x8」、「美女圖放在後面，框線疊在上面」，
並明示 HD 框線與人物排版要跟原版一樣，避免玩家感覺是額外貼圖。
**已定案**：8×8 原版格遮罩、原點 `(0,0)`；不採逐像素。
人物在後，框架／控制台／框線在前，中文最後；位置與範圍沿用原版。
這解除舊遮罩決策阻塞，但不升格尚缺技術契約的 024／204。

### 31.1 舊合成為何偏離原版

**已證實，限已保存檔案與程式**：舊 `tools/hd/compose.sh` 將 264×456 人物
等比縮為 250×432 再靠右，並放在框線之後合成。
因此左緣由原版 x=232 的 696 放大像素移到 710，人物也蓋住前方控制台。
`chrome-masked.png` 已把人物矩形整塊排除，僅交換它與人物的順序不足以恢復框架。
舊工具、圖片、原始輸入與所有舊收據保留，不再以該流程代表現行排版。

新原型固定人物於 `(696,0)`，使用原本 264 像素寬，不縮小或平移；
裁掉 y≥432 的越界部分，以原版座標的頂部與下方框架遮擋人物。
前方框架分區（原版 x,y,w,h）：
`(232,0,88,6)`、`(232,131,14,13)`、`(246,133,28,11)`、
`(274,129,42,15)`、`(316,123,4,21)`。
**強證據**：人工核對原版色號與參照圖的美術分區，並非遊戲內物件語意或完整輪廓驗收。
部分框架暫沿用原版像素，五條既有 HD 漸層在其上；不宣稱完整手繪框架已完成。

### 31.2 輸入、工具及可重跑入口

- 原版來源：`SCREEN.PBL` SHA-256
  `302a9724ddbf1872cf6b6ccdad1e2611e0c94663e61b40cb1ee46322d78ddbd1`；
  `MENU.PBL` SHA-256 `b4ef63944ca7a2f37fbd459356e064cab3f8d285296fc701d362ff99aed89641`。
  五張 SCREEN 仍放 `(0,0/40/80/120/160)`，MENU #0 仍放 `(160,4)`。
- 位址基準：解碼後 320×200 原版像素座標；輸出 960×600，沒有新增 IDA 位址結論。
- 場景沿用 §21 正常六次前進的 `replay-encounter`；來源 state／輸入／種子鏈保持原樣。
  本次只讀取已保存畫面，沒有執行遊戲、重擲亂數或改寫 state。
- 工具：`tools/hd/preview_layout.py`、`tools/pbl.py`；Python 3.13.15，
  `python:3.13-alpine` 映像 ID
  `540c7d91f98ff6880174c40e99067bf5941eb54d818a7a5e094d188b196a934d`。
- 逐檔來源與輸出 SHA-256 見 `workplace/hd/redraw/layout-v2-20261001.json`，
  包含工具、原版 PBL、原型圖層與獨立中文圖層。

先確認專案及原版目錄存在且輸出目錄 UID/GID 正確，在既有 Python Docker 中執行：

```text
python tools/hd/preview_layout.py --output workplace/hd/redraw/<新的前綴>
```

工作樹掛 `/src`，原版父目錄另外掛 `/src/workplace/original:ro`；
`--rm --network none`、目前 UID/GID、CPU 1、512 MiB、64 個程序、外層 90 秒及日誌上限。
工具要求既有且由目前使用者擁有的輸出目錄，已有輸出即拒絕，不能覆寫證據。

### 31.3 有限驗證與限制

**已證實，限離線原型**：

| 項目 | 結果 |
|---|---|
| SCREEN／MENU 重建原版背景 | 0 色號不符 |
| 獨立原版 RGB 參照與 EGA 色號放大 | 0 像素不符 |
| 人物矩形以外相對舊背景 | 0 意外改動 |
| 前方框架的原座標期望值 | 0 像素不符 |
| 故意把人物放回最上方 | 10,204 個框架像素不符；負對照有效 |
| 人物與前方框架實際重疊 | 10,314 個放大像素 |
| 8×8 遮罩保留原版動態／中文 | 各 0 像素不符 |
| 未授權黑色功能區被佔用 | 0 像素 |
| 重用輸出前綴 | 以非 0 及指定原因拒絕，全部來源與既有輸出未變 |

現行畫面 `layout-v2-20261001-chinese.png`；`-comparison.png` 左為原版中文、
右為修正原型；`-background.png`、`-foreground.png`、`-reversed.png` 另保留各階段。
人物候選的造型品質、完整框架重繪、跨幀恢復、切換、存讀檔及效能尚未驗收。
原版角色保持原樣，沒有將後續版本的角色素材接入第一版 SCREEN／MENU。
024 §3.6／§9、204 的 `【HD-LAYOUT-01】`、CONTEXT 及權威 worklist 同步回查本節。


## 32. 正式圖面層：可恢復格、完整多列及成本修正

2026-10-01。前輪 sprite 範圍回填屬實際進展。本輪依規格閘門及文件職責路由，
將 204 的技術缺口閉合後才實作；沒有以自動續行選擇新產品分支。
遮罩、原版排版及完整 sprite 範圍沿用使用者已確認的決定。

### 32.1 READY 契約與實作

既有 202／203 的限制仍是真實歷史，不直接拿文字層承諾圖面恢復。
新的 204 明定 Reference 原版基準、PixScale、RGBA 長度、非預乘 alpha、Order、
可恢復遮格、整組啟用比對及 ArtKeys 完整集合。切片登記時複製，全部被遮也不丟掉基準；
缺一列時先驗 factory 完整成功才替換整組，錯誤輸出不留半組。
快照只存圖面中繼資料，不存資產；還原成功清除圖面，由已知素材及基準重登記。
文字重疊移除與捲動不作用於圖面，不同 Key 的圖面可共存，按 Order／Key 排序，中文最後。

**規格設計及內部驗證**：這些是覆繪層契約，不是新增的原版遊戲語意。
先將 204 設 READY，實作 `xlate/art.go` 並整合 `layer.go`／`watch.go`，
通過 §32.2–32.3 後回填 CONFORMED；範圍只限通用圖面層及本次有限驗證。
024 仍 DRAFT，主題資料、前端流程、全部 sprite、美術及平台驗收仍待完成。

舊 DRAFT 的兩項處理已訂正：圖面不能重用文字的永久透明格；同組圖面不能因重疊刪掉
後續需要恢復的背景。保留原先證據、舊來源及訂正原因，未重寫 §23 的歷史收據。

### 32.2 原版輸入、工具及正常保存幀

- 原版檔案、SHA-256、PBL 偏移與原座標依 §29／§31；本次再次獨立重建 SCREEN 5 張及 MENU 1 張，
  與 `bg.idx` 不符 0。沒有新增 IDA 位址或反組譯命名。
- 兩個分支各使用三幀：`replay-encounter`、F3 或不按鍵的過渡及完整終點。
  原始來源由 §29 執行前固定 CAF0h 的保存狀態與等價機器時間取得。
  本次只讀已保存 `.frame`、原版 PNG、獨立中文圖層及來源快照，未執行新遊戲指令或重擲亂數。
  舊 `normal-recovery-20261001.json` 的 57 個來源雜湊再核對未變。
- HD 背景採 §31 的 `layout-v2-20261001-background.png`，人物位置、前後及範圍沿用定案。
  原版 sprite 在本次正常幀中保持原樣，這不計入 sprite HD 的完成數。
- Go 工具 `tools/hd/verify_art_plane.go` 呼叫正式 xlate Art；Python 的
  `tools/hd/verify_art_plane.py` 從原版 PBL、當前色號及獨立中文圖層算期望值，
  不呼叫 Go 圖面、不拿成果截圖當答案。位址基準是 320×200 原版像素，輸出 960×600。
- Go 1.24.13；`psychicwar-go-ebiten:latest` ID
  `083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7`；
  Python 3.13.15，既有 `python:3.13-alpine`。
  每個 Docker 限 CPU 1、預設無網路、原版唯讀、目前 UID/GID、程序／日誌上限；
  Go 2 GiB／120 秒，Python 512 MiB／90 秒。開工 uptime load 6.16／5.53／5.23，
  沒有即時音訊或平台幀率結論。

**已證實，限本批正式圖面對保存正常幀的消費結果**：

| 檢查 | 結果 |
|---|---|
| 六幀圖面對獨立 8×8 色號比對 | 全部像素不符 0 |
| 六幀最終合成、中文、原版動態 | 各項不符 0 |
| 多列完整性 | 每幀 25 列；兩分支各只 Make 一次 |
| F3 後恢復 | 1,216 個原版像素範圍，其中 47 個非黑背景像素 |
| 不按鍵對照 | 恢復範圍 0 |
| 故意把初始遮格設永久 Transparent | 少畫 423 個放大像素，與正確輸出不同 |
| xlate 契約及既有文字測試 | 39 項通過，含子案例 53 筆 |
| 主專案轉譯層相容測試 | 8 項通過，含子案例 133 筆 |

快照、多列錯誤 factory、半透明及倍率、壞輸入、圖字順序由單元測試覆蓋。
正常幀驗證不代表真實前端已載入 HD、不代表實際 F10／F11 往返或整段機器狀態相同，
也不能替代完整 sprite 的來源、動畫與正常玩家驗收。

### 32.3 繪製成本的具體修正

首次完整 1,000 格加文字的 Draw 約 20.142 ms，超出 60 幀每幀 16.7 ms 的預算。
保留首次通過版本後，把登記時已驗全不透明的格，改為繪製時合併連續區段複製；
半透明格保留非預乘合成，遮格仍依每幀原版資料。

同一工具鏈重新跑必要測試、正常幀及成本，**19 個輸出檔完整位元組相同**，包含
兩分支六幀的 Art RGBA、合成 RGBA、PNG，加一個永久遮格負對照。
本次 v2 量測：

| 情境 | Frame | Draw |
|---|---:|---:|
| 只含文字 | 0.003192 ms | 0.031069 ms |
| 1,000 格 Art 加文字 | 0.099383 ms | 0.208764 ms |

成本是共用主機上的 10 次固定工作量樣本，沒有以兩批比值宣稱固定倍數加速或跨平台幀率。
完整輸出比對及基準／alpha 測試用來證明此次最佳化沒有改變已驗畫面。

### 32.4 證據版本、保全與重跑

現行證據：

- `workplace/hd/redraw/art-plane-v2-20261001.json`：Go 正式圖面來源與全部輸出 SHA-256。
- `workplace/hd/redraw/art-plane-v2-independent-20261001.json`：Python 獨立核對、正常恢復及負對照。
- `workplace/hd/art-plane-tests-v2-20261001.jsonl`、`art-plane-translator-tests-20261001.jsonl`：逐項測試結果。
- `workplace/hd/art-plane-benchmark-v2-20261001.txt`：本次成本。
- 畫面例 `workplace/hd/redraw/art-plane-v2-20261001-0-2-composed.png`：正常 F3 後迷宮。

逐檔來源快照的唯一入口為各自 `manifest.json`，都在既有研究工作區，非新權威文件：

1. `workplace/hd/source-before-art-20261001/`：本輪實作前 14 個來源，供既有文字 API 回查。
2. `workplace/hd/source-before-art-fast-20261001/`：首次正式 Art 通過後的 16 個來源，供慢版收據回查。
3. `workplace/hd/source-art-v2-20261001/`：已驗 v2 的 16 個來源，包含驗收當時 READY 的完整 204。

規格改為 CONFORMED 後，Go 收據的規格雜湊仍對應第 3 項的 READY bytes，
不把測試後的文件狀態回填偽裝成執行當時來源。更早的舊 DRAFT 不必然等於本輪保全的 DRAFT；
只以逐檔雜湊相等認定同版本，不把快照日期當作相等證據。
慢版原輸出及收據保留，已用第 2 項快照重新核對來源並驗獨立消費結果。

新重跑前綴必須位於既有、UID/GID 正確的可寫研究目錄；工作樹掛 `/src`，原版另掛唯讀，
Go 使用已存在的獨立快取，在 `/src` 執行：

```text
go run -mod=readonly tools/hd/verify_art_plane.go -out workplace/hd/redraw/<新前綴>
```

工具先核對整批輸出不存在，再寫排他檔案，不能覆寫證據。
核對現行保存 v2 的命令（Python Docker、`/src`）：

```text
python tools/hd/verify_art_plane.py   --receipt workplace/hd/redraw/art-plane-v2-20261001.json   --output workplace/hd/redraw/<新的核對收據>.json   --tests workplace/hd/art-plane-tests-v2-20261001.jsonl   --benchmark workplace/hd/art-plane-benchmark-v2-20261001.txt   --source-snapshot workplace/hd/source-art-v2-20261001/manifest.json
```

快照參數只允許驗證 input 雜湊明確相符的已保存來源；輸出或原版正常路徑雜湊不同仍拒絕，
不以快照繞過錯誤像素。主題載入器及前端生命週期是下一閘門；完整 sprite 範圍依 024 §1.1。
本輪不改 Issue、commit、push 或發行包。

## 33. 第一批主題載入與前端接合（2026-10-01）

024 §4.1 已補齊第一批技術契約並達 READY；原版座標、原始輸入雜湊與正常起點
仍沿用 §29／§31／§32。此批不改遊戲程式、存檔格式或全部 sprite 的範圍。

實作入口：

- `apps/psychicwar/theme/theme.go`：無視窗相依的清單／PNG 載入器、SCREEN／MENU
  基準重建、全畫面 8×8 補格及讀檔重登記；`theme_test.go` 為正反向測試。
- `apps/psychicwar/theme.go`：可遊玩前端的既有資料目錄解析接合。
- `cmd/psychicwar/main.go`、`cmd/pwstep/main.go`：`-theme`、原版 → HD → 中文的繪製順序；
  可遊玩前端的 Shift+F5 與 F5 獨立。PNG 使用非預乘合成，Ebiten 輸出前才預乘。
- `tools/hd/prepare_theme.go`：已驗合成背景切為六筆本機研究資產，禁止覆寫目錄。
- `tools/hd/verify_theme.go`：從執行前固定 CAF0h 的正常遭遇狀態，以相同 F3 輸入
  比較 HD 開關兩側；第一批載入與狀態驗收工具，不是全部素材驗收。

實作前的前端、原規格及字型保存在
`workplace/hd/source-before-theme-20261001/manifest.json`；現行本機資產位於
`workplace/hd/theme-v1-20261001/`，來源是 §31 的 `layout-v2-20261001-background.png`。
該圖仍含原版素材及未完成的框架美術，不加入 Git、發行包或宣稱可公開。

初次測試揭露說明頁新增一行會超高，已在既有末行加入 Shift+F5 說明，保留不暫停語意；
另修正遮格單元測試的錨點，避免故意改錨點而誤當單格遮擋失敗。後續 40 筆支援測試
含子案例通過，無略過。直接引用前端套件造成逐步工具需要 DISPLAY，已拆出不引用 Ebiten
的 theme 套件；第一次工具呼叫的 `wait:0` 不符合既有動作解析器，不是遊戲缺陷。
後續驗證狀態、成本與前端實跑結果另列於本節，不以建置成功當作驗收完成。

### 33.1 正式載入與正常重播

工具為既有 `psychicwar-go-ebiten`（Go 1.24.13）、Python 3.13.15，單容器一 CPU、
原版 `/orig` 唯讀。SCREEN／MENU 及 PW.EXE 原始雜湊沿用 §29；引用位址是
執行期 `0161:41DF` 的種子，原版畫面座標 320×200，不是 IDA 線性位址。

- 載入器五項主測試、含子案例 14 筆通過；另有前端支援 25 項通過，無失敗／略過。
  測試包含真實 PBL 重建、缺圖、錯尺寸、錯位、圖號與矩形越界、清單缺欄位／null、
  符號連結越界、透明度及 MENU 的全畫面格線。收據
  `theme-loader-tests-v4-20261001.jsonl`、`theme-frontend-support-tests-v3-20261001.jsonl`。
- 正常起點 state SHA-256 `782405fd3ca55e61fafd3daa294b299911f1e7984199b62e71ee79f410ccd2b8`，
  執行前兩側唯讀核對 CAF0h；F3 掃描碼 3Dh 按住 150 ms，等待 500＋2,000 ms。
  通用字串解析器不支援 F3 鍵名，使用已證實掃描碼的 `oracle.Action`，不改解析器。
- 開 HD 與不開的終點：指令數皆 **46,108,635**、cycles 皆 **182,882,451**，完整
  1 MiB 可讀記憶體 SHA-256 皆 `868af9c6212a42fe0bdf27ee8d7014d0b72464fa9cc690aad5cab3e89fcb5d3d`；
  原版色號、CPU 暫存器也相同。顯示回呼不改原版記憶體；正式載入起始圖與 §32 的獨立
  合成完全相同，實際載回 state 並重登記後亦相同。限本次正常路徑，未比較原版 DAT。
- 來源／輸出及結果：`workplace/hd/redraw/theme-runtime-v2-20261001.json`、
  `-initial.png`、`-off.png`、`-on.png`；v1 保留。固定工作量 100 次平均主題 Frame／Draw
  本次約 **3.199／0.731 ms**，共用主機高負載，不宣稱即時幀率或固定速度提升。

### 33.2 實際視窗、錯誤模式及驗證器訂正

`tools/hd/frontend_check.sh` 在已就緒 Xvfb 的容器內，以正常第一人稱起點操作
F5、Shift+F5、F10、移動、F11，再切回 HD，產生九張視窗圖。
`tools/hd/verify_frontend.go` 從原版存檔色號／RGB、不可變背景及合成資產獨立推導期望，
不以內部開關當作畫面驗收。產物入口
`workplace/hd/gui-theme-v1-20261001/verified.json`。

| 核對 | 結果 |
|---|---:|
| 九張圖的人物／框線原版矩形 `(232,0,88,144)` | 各自不符 0 |
| HD 與原版模式內的中英文切換負對照 | 各差 3,699 像素 |
| 切回原設定 | 完整畫面不符 0 |
| 移動負對照 | 差 5,517 像素 |
| F11 後原版中文畫面 | 完整畫面不符 0 |
| F11 後 HD，依原版完整格轉換推導 | 完整畫面不符 0 |
| 存讀前後區域、座標、角色數值 | 線性 `0x16966` 起 52 bytes 相同 |

初次驗證器錯把人物區內的能量數值當成靜態背景，差 987；改用保存原版資料後不符 0。
讀回 HD 的原始截圖差 254；唯讀探針在 `(160,80)` 找到 16 個原版色號由 0 改成 8，
不可直接假定兩個自然運行時點相同。最終驗證器從兩份 state 的全部色號推導格集合，
遮格轉換為 `(160,72)`、`(160,80)`；套用完整格後不符 0，沒有硬編碼位置或放寬容許像素數。
這只證實顯示符合格契約，不命名尚未追查的原版變化原因。

Xvfb 的 `xvfb-run` 首次卡在就緒訊號，沒有進遊戲；已確認並停止本次容器，
改用現有 socket 就緒檢查、背景程序 trap 與容器內外逾時，重跑相同操作通過。
環境失敗不記成產品缺陷。音訊使用 null，不宣稱音訊或高負載即時幀率驗收。

實際 `pwstep` 的缺圖負對照以 1 結束，指出第 1 筆及 `SCREEN-00.png`；
倍率 2 配主題倍率 3 記一次提示，PNG 與未選主題的同狀態輸出完整 bytes 相同。
紀錄：`theme-missing-20261001.log`、`theme-missing-status-20261001.txt`、
`theme-scale2-20261001.log`，及 `redraw/theme-scale2*-20261001.png`。
修改 `text/help.json` 後已以 `tools/font/bake.sh` 重烘兩種字型，各 1,004 字；
Pillow 鎖 11.3.0；烘製腳本已補上寫入前 UID/GID 檢查與原版另掛唯讀。

### 33.3 回查與接續入口

已驗程式及當時 READY 規格的 18 份完整來源，保存於
`workplace/hd/source-theme-v1-20261001/manifest.json`；追加結果後的規格文字與收據
輸入 bytes 不同時，必須透過該索引回查，不能改寫既有收據。早期程式與字型另在
`source-before-theme-20261001/manifest.json`。本機合成資產仍含原版像素，不公開。

重跑命令在既有 Go Docker、`/src` 工作目錄、原版父目錄唯讀掛 `/orig` 執行：

```text
go run -mod=readonly tools/hd/prepare_theme.go -source workplace/hd/redraw/layout-v2-20261001-background.png -out workplace/hd/<新的主題目錄>
go run -mod=readonly tools/hd/verify_theme.go -theme workplace/hd/<主題目錄> -out workplace/hd/redraw/<新的收據前綴>
sh tools/hd/frontend_check.sh workplace/hd/<新的視窗輸出目錄>
go run -mod=readonly tools/hd/verify_frontend.go -out workplace/hd/<視窗輸出目錄>
```

前兩項及最後一項不用顯示伺服器；視窗腳本必須先完成 Xvfb 就緒檢查，且目前指定
本機 `workplace/hd/theme-v1-20261001` 與 `psychicwar-theme-v1`。所有輸出禁止覆寫，
新目錄父層先核對擁有權。接續是第一批剩餘驗收與 §1.1 sprite 來源／素材接入；
原版 DAT、四個原有檢查點、幀率、全部美術／sprite、正式封包／真機尚未完成。
024 保持 READY，#34 保持未完成；本批無 commit／push、Issue 寫入或發行。

舊正常重播收據的 57 個輸入中，本批只有 `cmd/pwstep/main.go` 改動；其舊 bytes
與 `source-before-theme-20261001/manifest.json` 所保存來源相符。舊收據不改寫，
不能用新 pwstep 去宣稱舊來源全部仍與工作樹相同。


## 34. ALLY #0 的限定主題接入與正常路徑

### 34.1 證據與契約

本節沿用 §22 的原始定位與工具位址空間，不新增角色身分推論。ALLY.PBL SHA-256
為 `c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219`，31 張；
#0 是 24×32，檔案偏移 0x003E，原座標 (264,152)。觀察入口為執行期
0161:8705，不是 IDA 線性位址；DS:BX 的內容是已解碼 384 bytes 完整 4bpp，
不能把 RLE 或 PBL 檔頭當作貼圖來源。以上定位與原圖符合為已證實，限定此圖與位置。

實作前先補 024 §1.2 READY，才延伸 `apps/psychicwar/theme/ally.go`、載入器與兩個前端。
OnCall 追加觀察者，不改原版；固定來源及穩定錨點管理出現狀態，逐 8×8 格管遮擋，
原版全黑格透明。未支援的完整替換／模式撤下舊來源；讀回 state 後由完整原圖重建。
技術接入與美術驗收分開，現有 ALLY-00.png 的來源限制仍見 §22 的 provenance。

本機七筆主題：`workplace/hd/theme-ally0-v1-20261001/`，六筆背景與舊 theme-v1 的
PNG bytes 相同，另加入 ALLY-00.png（SHA-256
402481edf4549b943bfa001e0849240f8ebe80d3df339c90505eb31e2b69e749）。
這是本機研究資產，未進 Git 或公開包。

### 34.2 正常固定種子與獨立期望

入口 `tools/hd/verify_ally_runtime.go`：在既有 psychicwar-go-ebiten 容器、Go 1.24.13、
原版唯讀 /orig/psychic-war 及目前模組環境執行：

```sh
go run -mod=readonly tools/hd/verify_ally_runtime.go \
  -theme workplace/hd/theme-ally0-v1-20261001 \
  -out workplace/hd/ally-runtime-新的唯一前綴
```

每側在執行前載入 `workplace/states/06-name.state`，SHA-256
ac8eec0bbc533b267782298c298ce4a60440d0568266e665c8a5badbf9e0ca4a；
唯讀核對 0161:41DF＝86AFh、步數 35,000,000，沒有寫 seed 或重擲。
35,500,000 起每 1,000,000 指令按 K/A/I/Enter，各 500,000 後放開，至 42,000,001。
本次 Oracle 定時事件與歷史 probe 節流佇列各自記錄，不先假設傳輸時序相同；
實際來源事件仍同在 38,648,892，AX=4200、BX=96C6、CX=4226、DX=0304、DS=1175，
packed SHA-256＝927937937cd2781ec2471dc6d6283e6ddcb51a1363a251bfda42ee3d0f140950。

收據 `workplace/hd/ally-runtime-v1-20261001.json`：兩側完整 1 MiB 可讀記憶體 SHA-256
f94bd8605698aaa0abfb18ae4fb7e44a02876aa9e48e958dd25711d526c5cb34、CPU 暫存器、
步數 42,000,001、cycles 178,719,911 與原版畫面均相同；Frame 前後記憶體不變。
原版終點 SHA-256 a17c8610ac338deddd3060238b4debe3e837df6c6f8c685d7d9162c0e4cedfc7，
亦與 §22 相同。直接讀 PNG 與原版 PBL 推導期望格，未向載入器取答案：
期望不符 0、角色區外變動 0；忽略 ALLY 負對照差 4,420 像素；實際保存、推進後載回
新的 state 並 ResetForLoad，角色圖面完全相同。HD／原版／只有背景圖片與 state 以同前綴保存。

`ally-runtime-tests-v1-20261001.jsonl`：7 主測試，含子案例 16 筆，無失敗／略過。
含真實來源、重複 Attach、第二個 Oracle 拒絕、局部完整格遮擋與恢復、原版空格透明、
清除、未知全圖／AL 撤銷、無關貼圖、失去錨點及讀檔後重建。

### 34.3 實際視窗與驗證邊界

前端 `workplace/hd/psychicwar-ally0-v1`、逐步工具 `pwstep-ally0-v1` 都已建置。
在有界 Xvfb、DISPLAY=:99、原版唯讀的相同 Go 容器中重跑：

```sh
sh tools/hd/frontend_check.sh workplace/hd/gui-ally0-新的唯一目錄 \
  workplace/hd/psychicwar-ally0-v1 workplace/hd/theme-ally0-v1-20261001
go run -mod=readonly tools/hd/verify_frontend.go \
  -out workplace/hd/gui-ally0-新的唯一目錄 \
  -bin workplace/hd/psychicwar-ally0-v1 \
  -ally workplace/hd/theme-ally0-v1-20261001/ALLY-00.png
```

現行 `gui-ally0-v3-20261001/verified.json`：九張人物／框線 (232,0,88,144) 與
ALLY (264,152,24,32) 區域不符均 0；獨立來源／格線計算期望，不使用正式載入器。
HD 負對照 4,420、語言負對照各 3,699、移動 5,517；原版讀回整屏差 0，
F10／F11 的線性 0x16966 起 52 bytes 區域／座標／角色數值相同。

驗證腳本 v1 的跨時間 a/e 比較差 254；v2 擷取用 F10 覆蓋玩家存檔，導致位置不同，
已查明並在 v3 對 quick.state、quick.state.xlate.json、quick.json、quick.map.json
全部保全／還原，不修改產品讀檔。v3 的逐張 state 色號可相同而稍後截圖仍差 254，
故這批資料不足以作整屏同幀對拍；收據完整保存原始差異，未硬編碼排除格或容許像素數，
整屏跨時間兩項只作觀察，不當成完成證據。這是採樣限制，沒有證據判定遊戲故障或特定原版語意。
ALLY 區域、人物／框線、語言與快速存讀檔為本批視窗驗收範圍；完整狀態不變依 §34.2。
使用 null 音訊及共用主機，不宣稱音訊或幀率通過。

預設回歸 `ally-default-20261001/verified.json`：01-title、03-protection、05-select、
07-first-play 各用原版／中文、wait:1、不選主題，比較 pwstep-theme-v1 與 pwstep-ally0-v1，
八組 PNG 完整 bytes 相同。這是固定起點的開發產物回歸，不代替發行包驗收。

### 34.4 保全與剩餘範圍

接入前五個來源與 READY 規格在 `source-before-ally-runtime-20261001/manifest.json`；
已驗 81 個來源／資料／素材在 `source-ally-runtime-v1-20261001/manifest.json`，
最新 GUI 驗證器在 `source-ally-runtime-v2-20261001/manifest.json`。
兩份已驗 READY 規格 SHA-256 e3ba6d29e3e0aaed7b3b272c17f0364037b3439a42d8cc1f075dd82cb63d73cb；
後續規格回填依此快照回查。v1／v2 的程式、輸入與資產相同，只更新驗證器範圍；
舊 §22／§30–33 收據及資產均不覆寫。

ALLY #0 僅完成限定技術接入，候選造型／身形比例仍需美術審查；其餘 30 張、敵人差分、
小圖塊、效果及其他素材維持完整必要範圍。原版 DAT、同幀整屏視窗矩陣、低負載幀率、
公開權利分類、正式 HD 封包與平台真機仍待驗。024 維持 READY，#34 不關閉。

## 35. ENEMY00 #3–#5：正常中文路徑與貼圖完成閘門

日期：2026-10-01。**已證實（限定技術範圍）**，不代表候選美術或全部動作驗收。
依 §20 原版來源證據先補 024 §1.3 READY，再實作；貼圖中途的新觀察亦先補入 READY 契約。
目前原始函式定位、暫存器及 operand 均保留，不改原版 EXE、記憶體、色號或規則。

### 35.1 輸入、工具與原版定位

- DOS `PW.EXE` SHA-256：`88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49`。
- `ENEMY00.PBL` SHA-256：`8241b0ea73b1e7402b13adc434f10a5b1288f88f01fcfc3ba5710e4923fe6067`，實際 30 張；#3–#5 各 24×32，位置 `(32,152)`。
- `06-name.state` SHA-256：`ac8eec0bbc533b267782298c298ce4a60440d0568266e665c8a5badbf9e0ca4a`；`07-first-play.state` SHA-256：`56c489deaa57cd26fbd4f39f374e87db2f51e7d1275c440c9805321fb09bd532`。
- 工具：dosgolem 基準 `f8c1a6e` 加收據記錄的覆繪來源；Go 1.24.13；Go Docker 映像沿用 §34。Python 3.13.15 僅核對檔案及保全來源，均於 CPU 1、無網路、原版唯讀容器內執行。
- 位址空間：dosgolem 執行期 `CS:IP` 的 `0161:8705` 貼圖入口／`0161:8751` 返回；資料為執行期 `DS:BX`，不是 IDA 線性位址。

| 原圖 | 入口步數／返回步數 | 來源／模式 | 已證實用途 |
|---|---|---|---|
| #3 | 83,190,318／83,214,859 | `1175:A686`，AL=00h | 完整 384 bytes |
| #4 | 83,373,343／83,397,692 | `1175:A8C6`，AL=01h | #3 XOR #4 的 384 bytes |
| #5 | 83,556,097／83,580,446 | `0161:344A`，AL=01h | #4 XOR #5 的 384 bytes |

每個來源與前後原版整屏均由 §20 的獨立檔案核對，非只檢查內部觸發旗標。
候選 PNG 分別沿用 `redraw/ENEMY00-group1-frame-00.png` 至 `02.png`，72×96，雜湊依序為：

```
46749cdea1fc877cfbd94f6fa340dfd1cea2df0bf1ad89e56d8c57cd846431df
8c4bd802f64798a8d853d291fc46333ce82c9bd65850dd08ff0659e38607c3ae
e1c2e07a61d87581159a7f9fbba101de10e3ecfcf5d7baa71f7fec8a5da5d8bf
```

生成提示詞沿用 `redraw/ENEMY00-group1-provenance.json`。本機十筆主題位於
`workplace/hd/theme-enemy00-v1-20261001/`；既有 SCREEN／MENU／ALLY 七筆未變，追加上述三筆。
來源套件為 `apps/psychicwar/theme/`；未知圖號、錯尺寸、錯位置及同槽重複 Reference 拒絕載入。

### 35.2 中途貼圖的原版觀察與實作

唯讀工具 `tools/hd/observe_enemy_partial.go` 從固定 07 state 與正常前進輸入，取得
`workplace/hd/enemy-partial-observation-20261001.json`。原版 #3→#4 並非瞬間完成：

| 絕對步數 | 與前一姿勢差異 | 與目標姿勢差異 |
|---|---:|---:|
| 83,373,344 | 0 | 88 |
| 83,374,343 | 1 | 87 |
| 83,377,343 | 14 | 74 |
| 83,381,343 | 40 | 48 |
| 83,385,343 | 60 | 28 |
| 83,389,343 | 80 | 8 |
| 83,393,343 | 88 | 0 |
| 83,397,692 | 88 | 0 |

八個抽樣點角色區外變動均 0，原始色號 SHA-256 與暫存器保存在觀察收據，並由正式驗證器納入輸入雜湊。
入口後第一個指令仍有完整舊圖，若此時用全圖辨識重選姿勢，會撤銷已由來源確認的新姿勢。
因此貼圖入口至返回期間禁止全圖重新選擇；保留已證實的目標來源，仍逐格依原版 8×8 比對遮擋。
返回後恢復全圖辨識，支援載回既有 state。未知來源／模式撤銷，無關貼圖不改角色，
失去錨點、顯示模式變更及重設均清除生命週期。讀來源先檢查可讀 RAM 界線，避免碰 VGA 匯流排。
同位置姿勢互斥；相同參考格不能把兩張半透明 PNG 重複疊加。

9 主測試、含子案例 18 筆全部通過，無失敗／略過；收據 `enemy-runtime-tests-v3-20261001.jsonl`。
新增反向對照會在入口後仍為舊圖時誤選姿勢、同槽雙畫、來源或前姿勢不符時失敗，
不是僅把實作旗標當作畫面正確證據。

### 35.3 正常路徑、完整狀態與獨立畫面

入口 `tools/hd/verify_enemy_runtime.go`，現行收據 `workplace/hd/enemy-runtime-v3-20261001.json`。
執行前固定上述 state 雜湊，兩側 seed 皆 86AFh，唯讀核對；不寫 seed、不重擲、不挑結果。
由正常名字輸入 K/A/I/Enter 接續，按下自 35,500,000 每 1,000,000 一鍵，500,000 後放開；
到 42,000,001，完整 RAM、暫存器、cycles 與原版畫面等於既有 07 state 再推一指令。
接著 Up 按下自 43,000,000 每 8,000,000 一次，4,000,000 後放開，停在 86,000,000。
這是舊 §20 排隊輸入的等價**已執行前綴**：第六次已按下、第六次放開尚未到達；不宣稱十一組全部執行。

HD 開／關兩側三個返回點及終點完整 1 MiB 可讀記憶體、暫存器、色號畫面、步數與 cycles 相同。
終點 86,000,000 指令、395,444,093 cycles；RAM SHA-256
`882733e333961e74e8459071bf085d8d8815ac12dd13e64980dba30e7a69c5f7`，
色號 SHA-256 `d0616fd57d1f7d078ea8abae989e6e901ff069f23890a0906e8cea845fb2e7d0`，與舊原版終點相同。
中文攔截自名字階段啟用，不注入翻譯答案。

| 原圖 | 獨立整屏 HD 不符／角色區外差異 | 省略角色負對照 | 錯姿勢負對照 |
|---|---|---:|---:|
| #3 | 0／0 | 4,321 | 3,964 |
| #4 | 0／0 | 4,328 | 3,916 |
| #5 | 0／0 | 4,539 | 4,171 |

期望由背景／ALLY 主題、原始 PBL 非空 8×8 格、候選 PNG 及中文層獨立合成，
比較完整 960×600 RGBA，不以正式 HD 截圖當答案。八個中途樣本原版雜湊與觀察收據一致、
獨立合成不符均 0；誤用舊姿勢負對照差 2,833–3,964 像素。
三份實際保存 state 與文字快照載回後 HD 圖面完整相同，HD 關閉回原版、重開回當前姿勢亦通過。
`ally-runtime-after-enemy-v3-20261001.json` 的正常名字輸入、完整狀態與 ALLY 獨立期望回歸通過。

最新 `pwstep-enemy00-v3` 的三個實際 CLI 載回 PNG 與對應獨立驗證圖完整 bytes 相同，
收據 `pwstep-enemy00-v3-20261001/verified.json`。使用 `wait:0.001`：既有 parser 要求正時長，
750 cycles/ms 下未滿一個 cycle，不推進 CPU；只作同狀態畫面核對，正常路徑證據來自上述重播。
預設回歸 `enemy-default-20261001/verified.json`：未選主題，01-title／03-protection／05-select／
07-first-play，各原版與中文、wait:1，與接入前 `pwstep-ally0-v1` 的八組 PNG 完整相同。

### 35.4 保全、重跑入口與限制

接入前六個來源在 `source-before-enemy-runtime-20261001/manifest.json`；v1 已驗來源 94 檔保留。
最新來源、READY 規格、輸入及驗證收據共 102 檔保存於
`workplace/hd/source-enemy-runtime-v3-20261001/manifest.json`；READY 規格 SHA-256
`86c6dab1cd180d14a4013200ee65efe2f609aad5fe5189096287015fa09fc1e1`。
其中 `verify_cli_and_preserve.py` 保存本次 CLI 核對與保全步驟；原版素材不複製，必須由本機合法輸入提供。
後續規格現況回填只改說明，驗證時的 bytes 由此快照回查。全部舊收據／圖／來源均保留。

v1 從 07 state 啟用中文時，「North」已在更早畫過，故留英文；v2 改由正常名字段攔截後修正。
這是驗證起點缺少先前文字事件，沒有改產品翻譯。CLI 首次 `wait:0` 被既有 parser 拒絕，
改用正時長重跑；屬命令錯誤，未記成遊戲缺陷。v3 再補貼圖完成閘門與八個中途對照。

重跑沿用 §34 的 Docker 資源、UID/GID、原版唯讀掛載及快取設定，在 Go 容器 `/src` 執行：

```sh
go test -mod=readonly -json ./apps/psychicwar/theme
go run -mod=readonly tools/hd/verify_enemy_runtime.go \
  -theme workplace/hd/theme-enemy00-v1-20261001 \
  -out workplace/hd/enemy-runtime-新的唯一前綴
go run -mod=readonly tools/hd/verify_ally_runtime.go \
  -theme workplace/hd/theme-ally0-v1-20261001 \
  -out workplace/hd/ally-runtime-新的唯一前綴
```

兩個驗證器會拒絕已存在的輸出，禁止拿新程式覆寫歷史收據。觀察工具目前使用固定輸出名，
重生須先在獨立研究工作樹承接其輸出，不能直接覆寫本次八個原始樣本。

候選三張只完成技術接入；正式美術仍 0/360。反向差分、完整循環、其餘角色／小圖塊、
BEAM／FIGHT、框架手繪、原版 DAT、正式包與平台真機仍待完成，不由 XOR 可交換性猜補原版呼叫。
本輪未重跑實際視窗，§34 的 GUI 範圍與跨時間限制保留；開工 load 44.14，僅做決定性抽測，
不宣稱即時幀率、音訊或跨平台通過。024 保持 READY，#34 不關閉；沒有 commit／push／遠端寫入／發行。

## 36. ENEMY00 正常往返循環的原版來源

日期：2026-10-01。**已證實（此正常遭遇的有界 16 次貼圖）**。
只讀工具 `tools/hd/observe_enemy_cycle.go` 從 §35 的固定 `07-first-play.state`、86AFh
與相同正常 Up 已執行前綴，觀察至 86,000,000；不開主題、不寫原版或固定新的測試答案。
工具 Go 1.24.13、dosgolem `f8c1a6e`；位址空間仍為執行期 `0161:8705`／`0161:8751`，
資料為 `DS:BX`。原版 PW.EXE／ENEMY00.PBL／state 的 SHA-256 沿用 §35.1，
並逐項記錄於 `workplace/hd/enemy-cycle-observation-v1-20261001.json`。

原始 48 份前後色號畫面／384 bytes 來源為 `enemy-cycle-observation-v1-20261001-eventNN-*`。
動作序列確認為 `3,4,5,4,3,4,5,4,3,4,5,4,3,4,5,4`。
反向 #5→#4 首次入口 83,738,831／返回 83,763,180，差分 SHA-256
`f7a0e8cb2e71a7118f181c31b43d2b353029cac71262e1f6778cca4b38df488e`；
#4→#3 首次入口 83,921,553／返回 83,945,961，差分 SHA-256
`ca815ba0697e977a2b37c5ae407c09480f3ae470b055b79f03419b05a8980e9c`。
完整暫存器、各次指令／cycle、前後色號及來源雜湊均保存，未以導覽名稱代替定位。

`tools/hd/verify_enemy_cycle.py` 以獨立 Python PBL 解碼、每次實際來源推導與區外比對，
驗完 16 筆：來源／原圖不符及角色矩形外變動均 0，誤當完整圖或錯姿勢的負對照有效。
收據 `workplace/hd/enemy-cycle-independent-v1-20261001.json`。
原版終點 RAM、暫存器、cycles 及畫面與 §35 的正常 HD 開關收據相同。
這些新原版證據支持 024 §1.4 READY，先完成契約再擴充實作；不猜補其他角色與效果。

### 36.1 往返接入與正常中文驗證

先保存實作前九份來源及 READY 規格於 `source-before-enemy-cycle-20261001/manifest.json`。
接入器將單一前姿勢擴充為已證實的轉換清單：#3 由 #4、#4 由 #3／#5、#5 由 #4；
每條皆核對實際前圖及差分 bytes，同位置互斥、未知回退與貼圖完成閘門沿用 §35。
不改動 PNG、背景／人物層序、原版 RAM、存檔或動畫時序。

現行 `tools/hd/verify_enemy_runtime.go` 以新原版觀察作獨立來源，從固定 06-name 的正常名字
接續相同迷宮起點及 Up 已執行前綴，兩側均固定 86AFh；收據
`workplace/hd/enemy-cycle-runtime-v1-20261001.json`。
16 個返回點的來源、整份 1 MiB RAM、暫存器、原版色號及 cycles 與原版一致；
終點 86,000,000／395,444,093，RAM／色號 SHA-256 與 §35 相同。
來源比對不是只看觸發旗標，核對各次實際 `DS:BX` bytes 與 48 份原版觀察檔。
首次 #5→#4 來源為 `0161:344A`、#4→#3 為 `1175:A8C6`，AL 都為 01h，
不把相同差分內容當作前姿勢已證實。

16 張完整姿勢的獨立 HD／中文整屏期望不符及角色區外變動均 0，
省略角色負對照差 4,321–4,539、錯姿勢差 3,916–4,171。
四方向共 26 個中途抽樣，比對沒有 HD 的原版色號，以及由目標原圖／候選 PNG／
完整 8×8 格線推導的期望，不符均 0，錯用舊姿勢負對照差 2,514–3,964。
#3→#4 的八個樣本另與 §35 獨立唯讀觀察完全相同。
HD 停用／重開及 16 份實際 state＋文字快照載回後整屏相同；
`ally-runtime-after-cycle-v1-20261001.json` 的 ALLY 正常路徑、完整狀態及獨立期望回歸通過。

主題測試 `enemy-cycle-tests-v2-20261001.jsonl`：9 主測試／18 含子案例全過，無失敗／略過。
前端及逐步工具建置成功。最新 `pwstep-enemy-cycle-v1` 抽驗事件 03 的 #5→#4 與
事件 04 的 #4→#3，正時長 `wait:0.001` 不推進一個 cycle，載回 PNG 與獨立期望完整 bytes 相同；
收據 `pwstep-enemy-cycle-v1-20261001/verified.json`。正常路徑由前述完整重播證明，
此 CLI 抽樣只證明實際最新工具的同狀態合成。

### 36.2 保全、重跑與限制

已驗來源、原版觀察檔、工具及收據 153 檔保存於
`workplace/hd/source-enemy-cycle-v1-20261001/manifest.json`；READY 規格 SHA-256
`b16078cd134244be3f84e6a72633d660e11a05900b7265968aa858bce89b78c2`。
舊 §35 的 102 檔快照及全部收據不覆寫；新增技術範圍不重寫舊驗證的來源。
原始 PBL／EXE 仍須本機提供，不進 Git 或公開包。

使用 §35 Docker 資源、快取、UID/GID 與原版唯讀掛載，在 Go 容器 `/src` 重跑：

```sh
go run -mod=readonly tools/hd/observe_enemy_cycle.go \
  -out workplace/hd/enemy-cycle-observation-新的唯一前綴
```

在既有 Python Docker 內獨立核對（`--original /orig/psychic-war` 預設）：

```sh
python tools/hd/verify_enemy_cycle.py \
  --probe workplace/hd/enemy-cycle-observation-新的唯一前綴 \
  --out workplace/hd/enemy-cycle-independent-新的唯一檔名.json
```

執行期驗證沿用 §35 的 `verify_enemy_runtime.go -theme ... -out ...`，但這版驗完 16 次循環及四方向，
並引用此次固定原版觀察；各來源與完整步數逐項核對。主題真實背景測試須設
`PSYCHICWAR_TEST_ORIG=/src/workplace/original/psychic-war`，它依現有目錄關係讀取 `workplace/hd/bg.idx`。

首次 Go 容器的 login shell 清掉既有 Go PATH，改用相同映像的非 login shell；
Python 核對器首次誤用既有 decode API，改為以偏移表取位址並解包三元組後同命令重跑。
第一輪測試指定 `/orig/psychic-war`，相對基準因此落在不存在的 `/hd/bg.idx`；
改正上述既有工作區入口重跑，失敗 log 保留。這三項屬工具／環境錯誤，沒有產品或原版缺陷。

本輪開工 load 8.66，僅做 CPU 1 的決定性抽測，不宣稱即時幀率或音訊。
已證實的是此角色正常往返循環；其他敵人、攻擊／效果、小圖塊、造型比例、完整框架、
原版 DAT、發行包及真機仍待完成。024 保持 READY，#34 不關閉，沒有 commit／push／Issue 寫入／發行。

## 37. 正常攻擊：PBL 差分與內建位元遮罩（2026-10-01）

### 37.1 目前結論與證據範圍

**confirmed：此正常攻擊路徑 975 次一般貼圖及 106 次綠色位元遮罩的來源、
方向、重疊場景與消除，已由獨立 Python 解碼逐次核對。**
不是 HD 合成、完整戰鬥效果、美術品質或全遊戲驗收；024 §1.5 保持 DRAFT。
§35–36 的角色往返接入不重開，正式程式本輪沒有新增效果覆繪。

原版執行器為 dosgolem `f8c1a6e`，Go 1.24.13；獨立核對用 Python 3.13.15。
靜態定位使用 IDA Pro 9.4 的正式 `PW_UNP.EXE.i64` 唯讀輸入及本機複本，不改名或覆寫註記。
IDA ea、執行期 CS:IP、執行期線性位址與 EXE 檔案偏移分別記錄。

| 輸入 | SHA-256 |
|---|---|
| `PW.EXE` | `88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49` |
| `PW_UNP.EXE` | `fd5115b91f014c47a1fb1a6e2ecfd645e293264fe614a4b350e92637cad2fbd9` |
| 正式 `PW_UNP.EXE.i64` | `4db51a199f5913c52dc4964557b9a4aafb965754d663cb0173c9dfdfe9a00c56` |
| `08-encounter.state` | `02d5fdba9181cb7fbec6282d74efbc25396562ba1403893cdf4c579ddfe770ba` |
| `BEAM.PBL`，12 張 | `ac1cdff92a20a1ed68de7a228835af6962fa103751a2c427c21cff43abb46045` |
| `FIGHT.PBL`，12 張 | `5dbced5104985c23ba085855fed6d0a6ff27bdac2a180dd3b42eef8d8926ae9d` |
| `ENEMY00.PBL`，30 張 | `8241b0ea73b1e7402b13adc434f10a5b1288f88f01fcfc3ba5710e4923fe6067` |

固定起點為第 86,000,000 道指令；執行前唯讀取得原版種子 5447h。
第 86,500,000 道指令按下空白鍵，保持到第 112,000,000 道指令；不改既有 state 按鍵佇列、
不寫亂數狀態、不重擲或挑結果。這與舊重播的 `space@86500000+3000ms` 時長不同，
不能冒稱同組輸入或完整 RAM 相同。這次終點 cycles 為 515,130,237；
原版色號 SHA-256 為 `04449b264486b8daabd02398d4ca679c59218fa74cca11280f65081ac0adad63`，
完整可讀 RAM 為 `5e60e554b31c36138aac29ad8e8b4d5c88abb27d02b6ae43c9fc6cd252005ab1`。

### 37.2 一般貼圖：來源與方向

執行期 `0161:8705`／`0161:8751`；IDA 對應 `sub_18C15`。
來源 DS:BX，x=CH×4、y=CL×4、w=DH×8、h=DL×8；此路徑全部 AL=01h、4bpp XOR。
975 次呼叫均有來源 bytes 及前後 64,000 bytes 色號畫面，未知來源數為 0。

| 來源／尺寸 | 此路徑座標 | 完整來源及差分 |
|---|---|---|
| BEAM，16×16 | y=160；x=40,56,…,248 | #0：37 次；#1：43；#2：36；0^1：158；0^2：155；1^2：153 |
| ENEMY00 小彈體，16×16 | y=160；x=56,72,…,216 | #18：27；#19：36；#20：35；18^19：54；18^20：59；19^20：56 |
| FIGHT，24×32 | (256,144) 或 (40,144) | #0／#2 各 2；0^1／2^3 各 52 |
| ENEMY00 角色，24×32 | (32,152) | #3 完整圖 1（消除）；3^4：9；4^5：8 |

來源配對本身不證明 XOR 方向；獨立核對器由固定起點角色 #4 與前一位置狀態，
逐次確認唯一的前圖→目標圖或消除。BEAM、ENEMY00 小彈體與 FIGHT 可交錯疊到同一格，
FIGHT 亦疊到敵人／盟友；不能拿單張解碼 PBL 作整個重疊場景的 Reference。

### 37.3 內建 0Ah 遮罩：原始定位與有限語意

一般貼圖 v3 的事件 18 返回與事件 19 入口之間，(120,152,16,16) 有 113 個像素
與 0Ah XOR；一般來源完全辨識仍不足以解釋整個場景。
IDA 資料庫的函式入口及唯讀正常重播，定位到以下獨立輸出。

| 原始定位 | 附加語意／等級 | 證據 |
|---|---|---|
| IDA `sub_1530A`、`15341 call sub_18770`；執行期 `0161:4E31` | 此路徑的遮罩呼叫端，confirmed | `effects-green-kernel.json` callers；正常追蹤 |
| IDA `sub_18770`，`18778 jmp loc_16FD2`；執行期 `0161:8260` | EGA 輸出分派，confirmed | IDA 分支原始 bytes；DS=0161h、DX=4E36h |
| IDA `loc_16FD2`–`17023` | 16 列、每列兩個 bytes、以 EGA 平面 0Ah／XOR 輸出，confirmed | 原始輸出指令、每次 before／after 及獨立位元核對 |
| 執行期 `0161:4E34`；IDA `15344 pop ds` | 遮罩已畫完、尚未返回呼叫端，confirmed | 正常 106 次入口／返回畫面 |
| 執行期 `0161:4E36`；IDA ea `15346` | 32 bytes 位元遮罩，confirmed；其遊戲語意尚未知 | 靜態解壓 EXE bytes、所有正常來源 bytes 相同 |

32 bytes 為 `0fe01038403c402e4b46bb46ba47bac7bdc7bfc747c65bce291810580fe00000`，
SHA-256 `e1aa2b9ddb6488a70d573ca3a03e71028cd8a2dafa68087c6f6f3d7c60950a5a`。
本機 EXE 檔案偏移為 `0x5716`，由 MZ 表頭 `0x3D0`＋初始 CS `0x51×16`＋`0x4E36` 計算；
不是 IDA ea 或執行期線性位址。座標由 `(BX−C000h)%80×4` 與 `(BX−C000h)/80` 取得。
106 次均為 16×16，涵蓋 34 個位置，y=152／160／168；每次恰好 113 個像素變更，
來源不符及矩形外變動均 0。只確認 0Ah 遮罩輸出，不自行命名為傷害、命中或特定能力；
此戰鬥的 0Ah 是黃色，預設色盤的綠色誤稱訂正見 §38–39 與 `000-overturned-claims.md`。

原版核對器以 SCREEN／MENU、ALLY #0、當前敵人、各效果位置狀態及此靜態遮罩重建場景；
(32,144,256,40) 在全部 1,081 個輸出入口／返回不符均 0，結束時效果狀態全清除。
每次省略輸出均會造成可見差異；遮罩的省略負對照每次差 113 像素。
另逐次由 before XOR 原始來源核對整屏 after，矩形外變動均 0。

### 37.4 HD 候選與尚未完成項

使用內建 `image_gen`，未使用 CLI／API 備援；完整提示詞及預設保存來源已登錄：

- `tools/hd/battle-effects-prompts-20261001.json`：#18／#19／#20／CS:4E36 四張 v1。
- `tools/hd/battle-effects-prompts-v2-20261001.json`：#18 偏小、偏低的針對性重繪；其餘沿用 v1。
- 生成原圖已複製到 `workplace/hd/art-in/battle-effect-*-generated-v*-20261001.png`。
- 48×48 資產在 `workplace/hd/art-in/battle-effect-*-48-v2-20261001.png`；
  現行對照為 `workplace/hd/battle-effects-candidates-comparison-v2-20261001.png`。
  依序 #18／#19／#20／遮罩，上列原版，中列候選放大，下列實際 48×48。
- `tools/hd/prepare_battle_effects.go -version v2` 僅全圖面積縮放，沒有裁切／平移，保留真實 alpha；
  `battle-effects-candidates-v2-20261001.json` 保存來源、尺寸、透明通道及輸出雜湊。

四張有有效透明通道、48×48 可解碼；v2 改善 #18 的低位與尺寸，**造型及三張動作一致性尚未通過**。
既有 BEAM／FIGHT 24 張候選不重新生成；本輪四張不加入正式主題，不改正式 ENEMY 0/360 狀態。
下一步是具體重疊 HD 原型、中途貼圖、未知來源回退、開關與效果中途讀檔，
並審查內建遮罩的資料表示；達 READY 後才實作正式路徑，全部 sprite 目標維持。

### 37.5 重跑與來源保全

使用 §35 的 Docker 資源、UID/GID、快取及原版唯讀掛載；Go 使用非 login shell。
以下命令在既有 Go 容器 `/src` 執行，輸出前綴必須全新：

```sh
go run -mod=readonly tools/hd/observe_battle_effects.go -deltas \
  -out workplace/hd/battle-effects-observation-新的唯一前綴
```

在 Python Docker 內獨立核對：

```sh
python tools/hd/verify_battle_effects.py \
  --probe workplace/hd/battle-effects-observation-新的唯一前綴 \
  --out workplace/hd/battle-effects-independent-新的唯一檔名.json
```

已驗證的原版收據為 `battle-effects-observation-v4-20261001.json` 與
`battle-effects-independent-v4-20261001.json`。v4 觀察工具 bytes 另存
`workplace/hd/observe_battle_effects-v4-20261001.go`；目前工具只訂正範圍說明，
不拿現行 hash 冒充舊收據來源。v1–v3 原始收據、來源版本與失敗樣本保留。
來源、IDA 匯出、候選及原版 v4 幀的快照入口為
`workplace/hd/source-effects-v4-20261001/manifest.json`；全在本機，不進 Git 或公開包。

IDA 原始匯出在 `workplace/ida/hd-effects-20261001/effects-*-*.json`／`effects-kernel.json`，
包含輸入 EXE／正式 DB 雜湊、工具版本及位址空間。匯出腳本保存於
`workplace/hd/ida-effects-*-20261001.py`，由 `ida-pro-9.4-idapython:locked-v1` 的
`idat -A -L/out/私有log -S/scripts/腳本.py /out/PW_UNP.EXE.i64` 對本機 DB 複本重生；
正式 DB 與 EXE 以 `/in` 唯讀掛載，輸出用獨立 `/out`，不匯出私有授權 banner。

首輪一般來源未收 XOR、第二輪未收 ENEMY00 小彈體，擴充來源後全部辨識；
獨立場景 v3 在事件 19 檢出 113 像素缺口，補上已證實遮罩後同模型完整通過，沒有放寬比對範圍。
VGA `OnWrite` 返回零筆屬工具能力限制，不是原版沒有寫入；改以 IDA 函式入口及畫面定位。
IDA 首兩次匯出因版本 API 及 ASCII 編碼失敗，改用 `idaapi.get_kernel_version()`／UTF-8 後重跑。
候選轉檔最初誤選沒有 Python 的影片映像，查回既有工具鏈後改用 Go 標準圖片庫，沒有建立重複映像。
本輪 load 18.81，僅 CPU 1 決定性觀察，不宣稱即時幀率或音訊。

## 38. 效果場景唯一分解與實際重疊／中途讀檔原型（2026-10-01）

### 38.1 範圍、來源與證據等級

本節沿用 §37 的原版收據，不重新開啟已證實的 975 次一般貼圖及 106 次位元遮罩。
新問題是：不保存 HD 狀態，能否從被多重效果覆蓋的原版畫面恢復候選圖層，並維持
原點 `(0,0)` 的 8×8 格、原版位置及人物在後／框線在前。024 §1.5 仍為 DRAFT，
工具為可丟棄原型；沒有修改正式效果路徑、資料格式或原版記憶體。

原版／工具為 dosgolem `f8c1a6e`、Go 1.24.13、Python 3.13.15。PBL、EXE 與原版
遭遇 state 的 SHA-256 沿用 §37.1；原型收據逐項保存實際輸入雜湊。名字起點
`06-name.state` SHA-256 為 `ac8eec0bbc533b267782298c298ce4a60440d0568266e665c8a5badbf9e0ca4a`。
本節沒有新增反組譯定位；CS:IP、靜態 EXE 偏移與遮罩定位沿用 §37.3，不能混用位址基準。

| 結論 | 等級與限制 |
|---|---|
| 已驗原版場景的 114 個變數可唯一還原 | confirmed：僅限固定正常攻擊的 2,162 個完整原版幀；Python 獨立解碼核對已證實呼叫方向 |
| 原型 21 次實際存讀檔顯示一致 | confirmed：兩种原型 PNG、原版色號、暫存器、cycles 與 A0000h 以下 RAM；不冒稱全部硬體狀態逐項相同 |
| 中途／最多重疊各一份讀檔可接續到原版終點 | confirmed：112,000,000 步的色號、cycles 及終點複本 1MiB 可讀記憶體指標一致 |
| 單矩形投影取共識可供中途顯示 | hypothesis：已抽 15 個正常中途畫格；未獨立證明任意未知寫入／所有部分畫面的來源唯一性 |
| HD 透光混色更適合重疊效果 | 視覺建議，未定案；已展示實際對照，不進正式規格 |

### 38.2 完整場景的代數分解

以 `(32,144,256,40)` 的 10,240 個原版 4bpp 像素建立 40,960 個二進位方程。
不可變底圖由實際 SCREEN 五張、MENU #0、ALLY #0 重建；角色使用「底圖 XOR 完整角色」
的差量，效果使用原始來源像素。變數只取 §37 已觀察到的槽位與姿勢，不猜補其他位置。

| 種類 | 變數數量 |
|---|---:|
| BEAM，14 槽 × 3 圖 | 42 |
| ENEMY00 小圖塊，11 槽的實際姿勢 | 31 |
| FIGHT，2 槽 × 2 圖 | 4 |
| 敵人完整姿勢 | 3 |
| 內建位元遮罩位置 | 34 |
| 合計／矩陣秩 | 114／114 |

Python 分解與已獨立核對的呼叫序列逐次比對，全部 2,162 個入口／返回畫面相同。
在 `(280,144)` 注入未知像素，以及同槽兩張姿勢 XOR，兩個負對照均被拒絕。
相同位置最多一張完整姿勢；能解線性方程不代表是合法角色狀態。
這只證明此原版樣本的可辨識性，不代表全遊戲效果或 HD 美術已完成。

### 38.3 原型合成、部分畫面與讀檔

Go 原型用正常名字輸入 K／A／I／Enter、六次前進接到原版 `08-encounter.state`：
暫存器、cycles、原版畫面與 A0000h 以下 RAM 相同。起點種子 86AFh，攻擊起點 5447h；
載回已核對遭遇 state 以保留原有未來按鍵佇列，第 86,500,000 步按住空白鍵至
112,000,000。不重擲、不更改原版規則或動畫時序。原型 live 重播全部 2,162 個完整幀
雜湊與已驗原版相同，Go 分解亦與 Python 已驗方向相同。

完整解失敗時，依 51 個已知矩形投影，取所有一致解中的固定變數共識；歧義變數的
整個矩形保持原版。其餘格仍須完整符合重建色號模型，否則原版回退。這是待審查方法，
不能把一個未知像素直接解讀成「原版正在畫某特效」。每張候選僅在原版有墨跡的 8×8
格顯示，原型基底驗為全不透明，避免透明洞留下舊版角色或效果。

中途時點依各種類首個實際變更呼叫預先選定：一般貼圖入口後 +1／+1,000／+4,000／+8,000
指令，位元遮罩 +1／+128／+256；不挑選會通過的時點。BEAM、FIGHT、ENEMY00 小圖塊及
遮罩共 15 幀；其中 7 幀需投影共識，8×8 不符／歧義格均保持原版。
正常場景整屏有 34–48 格因原版動態文字、數值或部分貼圖不符而回退；不把這些格報為
HD 失效，也不宣稱中途已全 HD。

保存 21 組實際 `.state`／`.xlate.json`，另開 Oracle 載回，不保存任何 HD 姿勢索引：
兩种合成 PNG、原版畫面、暫存器、cycles 與安全 RAM 比對全相同。其中首個未完成 FIGHT
狀態（86,539,215）與最多效果狀態（91,366,425）讀回後繼續正常攻擊，兩份均到相同終點。
終點指標與 §37.1 完整相同；1MiB 指標僅在不再繼續執行的終點複本讀取。

一般透明合成依 FIGHT→BEAM→小圖塊→遮罩順序；透光混色以每色通道的透光率乘積
`255 − (255 − 底色) × Π(1 − 色值×alpha/255²)`，最後一次四捨五入。
兩者都是新的 HD 視覺處理，不冒稱原版 XOR 色號完全一致。對照圖
`workplace/hd/battle-effects-prototype-v4-20261001-overlap-comparison.png` 依序為原版中文、
一般透明、透光混色。已向使用者提出具體選擇；未回覆不視為批准。
完整中文中途圖例為 `battle-effects-prototype-v4-20261001-step86665204-screen.png`。

### 38.4 觀察工具失敗與有限 A/B 訂正

原型 v1 太早注入空白鍵，後續要求回到較早完整幀，檢出指令數倒退；修正採樣排程。
v2 在首個 FIGHT 返回 86,562,564 檢出原版幀不符；原因為讀檔核對對正在繪製的主 Oracle
呼叫 `Bytes(0,1MiB)`。此 API 逐位元組呼叫匯流排 `Read8`，VGA 區會更新 EGA 鎖存器，
不是沒有副作用的 RAM 快照。

有限 A/B 使用 v2 保存的同一 86,539,215 中途 state，各起一個 Oracle，唯一差別為是否
先做該 1MiB 匯流排讀取，再推到 86,562,564。乾淨分支色號 SHA-256 符合已獨立核對原版
`9af6bbbe75a7c07792a4f76ba69da5bee3155b12d3c649730c4563d7bc2f02e6`；讀取分支差 **2 像素**。
因此 confirmed 為觀察工具副作用，不是產品貼圖缺陷。v3／v4 改用 `Save()`＋
`SearchChanged()` 的 A0000h 以下安全 RAM 比對，所有原版完整幀及終點重新符合。
v1／v2 來源及失敗收據保留，沒有放寬原版期望，也不開硬體時序逆向切片。

### 38.5 重跑入口與剩餘閘門

使用 §35 的既有 Docker 映像、原版唯讀掛載、UID/GID 1000:1000、CPU 1、離線快取及
非 login shell；每次 `--rm`、有界逾時，輸出檔／前綴必須全新。工具均為研究入口：

```sh
# Python 容器：原版完整場景的獨立分解
python tools/hd/probe_effect_decomposition.py \
  --probe workplace/hd/battle-effects-observation-v4-20261001 \
  --verified workplace/hd/battle-effects-independent-v4-20261001.json \
  --out workplace/hd/battle-effects-decomposition-新的唯一檔名.json

# Go 容器：具體重疊、中途及實際讀檔原型
go run -mod=readonly tools/hd/prototype_battle_effects.go \
  -out workplace/hd/battle-effects-prototype-新的唯一前綴

# Go 容器：觀察副作用有限 A/B
go run -mod=readonly tools/hd/probe_vga_observation.go \
  -out workplace/hd/vga-observation-ab-新的唯一檔名.json

# Python 容器：獨立候選合成及 8×8 回退核對
python tools/hd/verify_effect_prototype.py \
  --receipt workplace/hd/battle-effects-prototype-v4-20261001.json \
  --source-snapshot workplace/hd/source-effects-prototype-v4-20261001/manifest.json \
  --out workplace/hd/battle-effects-prototype-independent-新的唯一檔名.json
```

本節原型收據為 `battle-effects-prototype-v4-20261001.json`；最新黃色候選原型 v5 見 §39。
Go v3／v4 已驗來源另存
`workplace/hd/prototype_battle_effects-v3-20261001.go`／`prototype_battle_effects-v4-20261001.go`。
來源、候選、原型及失敗收據的逐檔快照為
`workplace/hd/source-effects-prototype-v4-20261001/manifest.json`，原版 2,162 幀回查 §37 的快照；407 檔、38,449,196 bytes 均逐項雜湊及 UID/GID 通過。
Go／分解工具只訂正「獨立」用字，Python 核對器另支援明示舊來源快照；
舊收據來源 bytes 回查快照，不拿現行 hash 冒充舊來源。核對器快照版另存 `workplace/hd/verify_effect_prototype-v3-20261001.py`。
Python 獨立候選合成收據為 `battle-effects-prototype-independent-v2-20261001.json`；
快照支援版以 `battle-effects-prototype-independent-v3-20261001.json` 乾淨重跑，21 筆核對與負對照相同：
21 幀 × 兩種混色，在 `(32,144,256,40)` 戰鬥區的候選合成、原點 8×8 遮格及原版回退
不符均 0；省略特效負對照差 20,169 像素。完整幀來源取已獨立核對的 profile，部分幀
來源共識仍引用 Go 收據，未獨立證明其唯一性；不涵蓋區外合成、中文、美術或正式前端。
核對器 v1／診斷版在 86,702,889 的一般透明中途畫面檢出 540 像素，全部是原版回退格：
468 像素實際黃色、預期亮綠；72 像素實際棕色、預期綠。查回 `docs/re/005` §1–2，
原版已證實將屬性暫存器 02h→06h、0Ah→76h，戰鬥色號 2／A 為棕／黃。
修正核對器引用同狀態原版 RGB 畫面，戰鬥區域先驗沒有中文覆繪；不放寬合成或遮格條件。
§37 的「綠色遮罩」是依預設色盤的誤稱，來源色號及 bytes 不變；現行以「內建 0Ah 遮罩」
稱呼。當時 v2 候選為綠色；§39 的 v3 黃色候選已修正配色，造型及透明邊緣仍需審查。
舊候選及失敗收據保留，不能當作正式美術通過。
混色選擇、來源共識／未知回退、內建遮罩資料表示及正式開關契約尚待閉合；
候選造型、三張動作一致性、其他敵人／盟友／圖塊、完整 HD 與交付目標維持未完成。
本輪開工 load 6.458／8.446／10.873，僅 CPU 1 決定性驗證；不宣稱即時幀率或音訊。

## 39. 遮罩黃色候選與 sprite 色盤盤點（2026-10-01）

### 39.1 證據及範圍

上一輪是實際進展：§38 原型、獨立合成、讀檔及觀察副作用 A/B 已閉合。混色問題仍待
使用者回答；本輪只做不依賴該選擇的配色修正、資料盤點與候選驗證，不自行批准正式接入。
規格及文件職責路由重新載入，沿用 imagegen 技能與既有 Docker 工具鏈；未使用 CLI 備援。

配色依 `docs/re/005` 已證實的主畫面／戰鬥屬性暫存器：色號 2 為 `AA5500`、A 為 `FFFF55`，
8 為黑。本輪以 §38 保存的 86,702,889 中途原版色號及無中文字的戰鬥區 RGB，再確認
2／A 的實際棕／黃值。這是既有原版色盤證據的使用，不新增硬體時序或完整反組譯。
工具為 dosgolem `f8c1a6e`、Go 1.24.13、Python 3.13.15；來源 EXE／PBL／state 雜湊沿用
§37–38，新的收據逐項保存工具、色盤證據、實際樣本、輸入與輸出 SHA-256。

### 39.2 415 張來源盤點及正確參照

`tools/hd/audit_sprite_palette.py` 解碼 BEAM 12、FIGHT 12、ALLY 31、ENEMY 十二檔 360 張，
總共 **415 張**，逐檔核對實際圖數，盤點 2／8／A 的出現數。九張包含敏感色號：

| 來源 | 圖號 | 盤點用途 |
|---|---|---|
| FIGHT | #8–#11 | 後續按已證實玩家輸出路徑檢查配色，未接入本輪固定攻擊原型 |
| ALLY | #12–#15 | 有大面積 A 色號；不能直接把預設綠色解釋成透明背景或角色顏色 |
| ENEMY10 | #2 | 含兩個 2 色號像素，完整造型及輸出路徑仍待驗 |

BEAM、ALLY #0、ENEMY00 三姿勢及小圖塊均不含這次盤點的敏感色號。
含色號不等於候選錯色：其他狀態可以重設屬性暫存器，Game Over 已知會改成綠色。
本輪只建立普通主畫面／戰鬥色盤參照，不宣稱九張均已驗正常輸出或已修美術。

收據 `workplace/hd/sprite-battle-palette-audit-v1-20261001.json` 有九張來源的尺寸／數字及
所有輸入雜湊。九張參照為 `workplace/hd/<檔>-<兩位圖號>-reference-battle-v1-20261001.png`；
另建立 ENEMY00 #18–#20 與內建遮罩四張 `*-reference-v3-20261001.png`，共 **13 張**。
參照直接由原始色號／32 bytes 遮罩與已驗色盤繪出，未裁切或改變原版圖形。

### 39.3 黃色候選與幾何限制

內建 image_gen 對 `battle-effect-mask-generated-v1-20261001.png` 做單一配色修正，
保留不規則輪廓及透明孔洞，將綠色改為黃／金色。完整提示詞、原圖路徑、用途與候選狀態在
`tools/hd/battle-effects-prompts-v3-20261001.json`。生成原圖已複製到本機
`workplace/hd/art-in/battle-effect-mask-generated-v3-20261001.png`；48×48 面積縮放候選為
`battle-effect-mask-48-v3-20261001.png`，SHA-256
`f97b966fe77b0a32400da9623a2a7092f13301f5c66bb08cb3994d5dd6141af5`。
轉檔入口 `tools/hd/prepare_battle_effects.go -version v3 -reference-version v3`；
另外三張小圖塊 v3 與 v2 **完整 PNG bytes 相同**，沒有順便改造型。

候選對照為 `workplace/hd/battle-effects-candidates-comparison-v3-20261001.png`，
三列依序正確色盤原版、候選放大、實際 48×48。候選核對收據
`battle-effects-color-candidate-check-v1-20261001.json`：兩版 alpha 範圍均 0–253，
alpha≥128 的外框均為 `(0,0,48,46)`；覆蓋像素由 1,085 改為 1,097，
綠色優勢像素由 1,085 降至 0。**921 個像素的 alpha 值不同**，不是逐像素保持的純色值替換；
形狀、透明邊緣及造型仍需正式美術審查，不把外框相同或沒有綠色當作完成驗收。
舊綠色候选與舊原型收据保持不變，正式 ENEMY 美術仍 0/360。

### 39.4 修正後正常戰鬥原型

`tools/hd/prototype_battle_effects.go` 新增研究用 `-mask`，明示候選路徑，避免覆寫舊素材；
`verify_effect_prototype.py` 改由收據的唯一遮罩來源取候選，不假設固定 v2 檔名。
其他來源、原點 8×8 格、原版座標、兩種混色、正常輸入與期望維持 §38。

最新 `battle-effects-prototype-v5-20261001.json`：2,162 完整原版幀、15 中途樣本、21 次
實際存讀檔及兩份讀档後的112,000,000步終點均通過，原版色號、cycles與終點記憶體指標不變。
獨立候选合成 `battle-effects-prototype-independent-v4-20261001.json`：21幀×兩種混色，
戰斗區 `(32,144,256,40)` 合成、8×8遮格及原版回退不符均0；省略特效負對照差 **20,182**。
中途來源共識仍引用Go結果，區外合成、中文、美術及正式前端未在此獨立驗收，不能外推全部效果。

最新重疊图 `workplace/hd/battle-effects-prototype-v5-20261001-overlap-comparison.png` 左為
原版中文、中為一般透明、右為透光混色，使用修正後的黃色遮罩。混色選擇未獲回答，
024 §1.5保持DRAFT；沒有修改正式sprite程式、原版遊戲规则、資料格式或存檔。

### 39.5 重跑入口

使用 §35 的既有 Go／Python Docker、UID/GID 1000:1000、CPU 1、唯讀原版與有界逾時。
下列命令在容器內執行；所有輸出／前綴必須全新，生成圖以唯讀目錄掛載後複製到本機。
本次開跑 load 31.54／26.60／22.10，只作決定性工作，不宣稱即時音訊或幀率。

```sh
# 首次建立十三張參照；既有輸出不覆寫，重跑須使用來源快照／新工作樹
python tools/hd/audit_sprite_palette.py \
  --out workplace/hd/sprite-battle-palette-audit-新的唯一檔名.json

# 依生成資料清單建立版本化候選；v3 已存在時拒絕覆寫
go run -mod=readonly tools/hd/prepare_battle_effects.go -version v3 -reference-version v3

# 用已保存黃色候選做正常路徑重疊及讀檔原型
go run -mod=readonly tools/hd/prototype_battle_effects.go \
  -mask workplace/hd/art-in/battle-effect-mask-48-v3-20261001.png \
  -out workplace/hd/battle-effects-prototype-新的唯一前綴

python tools/hd/verify_effect_prototype.py \
  --receipt workplace/hd/battle-effects-prototype-v5-20261001.json \
  --out workplace/hd/battle-effects-prototype-independent-新的唯一檔名.json
```

完整 sprite 及美術目標維持；下一步為混色定案與中途來源共識／未知回退、內建遮罩表示契約
審查，並按玩家輸出路徑接續其他 FIGHT／ALLY／ENEMY。候選及原版全部留在本機，未發布。

本輪已驗來源保全入口為 `workplace/hd/source-effects-color-v5-20261001/manifest.json`；
211 檔、26,799,039 bytes 逐項雜湊、尺寸與 UID/GID 通過，上一份 §38 快照 hash 亦確認。

## 40. 首場遭遇的有限按鍵輸出核對（2026-10-01）

### 40.1 前置條件與固定輸入

沿用 §37 的 dosgolem `f8c1a6e`、Go 1.24.13、Python 3.13.15、原版 EXE 與
`08-encounter.state`，起點 86,000,000 指令、執行前種子 5447h、終點 112,000,000。
原版未來按鍵佇列保持原樣；不改記憶體、亂數或遊戲資料，不挑選結果。
原始貼圖定位仍為執行期 `0161:8705`／`8751`；遮罩定位沿用 §37.3。

依 `docs/manual-1989.md` 與 `docs/re/036`，F1 的強力加速砲需要彈匣，F2 的戰友／
投降輸出需要相應前提；首場遭遇不具備這些條件。因此本節只測 Enter、無新增按鍵、
以及空白鍵後再加入 Enter；沒有以缺前提的 F1／F2 當作能力覆蓋。
ALLY.PBL 實際 31 張、SHA-256
`c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219`，
與 BEAM／FIGHT／ENEMY00 一起作來源候選；其餘輸入雜湊見每份收據。

| 樣本 | 正常按鍵輸入 | 一般貼圖／遮罩 | 終點 cycles |
|---|---|---:|---:|
| enter | 86,500,000 按住 Enter 至終點 | 142／0 | 486,279,123 |
| none | 不加按鍵 | 142／0 | 486,284,024 |
| space-enter | 86,500,000 按住空白鍵；87,000,000 再按住 Enter | 975／106 | 515,129,890 |

### 40.2 獨立核對結果及停止線

`tools/hd/verify_battle_key_paths.py` 獨立解碼四個 PBL，核對原始來源 bytes、同尺寸
XOR 配對、每次 before／after 的整屏運算，以及位元遮罩。1,259 次一般貼圖及
106 次遮罩的來源與整屏像素不符均 **0**；每次省略輸出的負對照都產生可見差異。
未知一般來源及其他 DS:DX 遮罩來源數均 0。這是有限樣本的 confirmed 輸出證據，
不是所有能力、圖塊用途或 HD 完成證據。

Enter 與 none 皆只有 `(32,152,24,32,AL=01h)`：ENEMY00 #3^#4 共 72 次、#4^#5
共 70 次。忽略執行步數時，兩側來源、before、after 序列完整相同；入口／返回步數
並不相同。終點畫面均為
`d0616fd57d1f7d078ea8abae989e6e901ff069f23890a0906e8cea845fb2e7d0`；
RAM 分別為 `53440f018a4ca8ae7dab4639f202e58377b9b8d13d5c1ed97391069a21c6dfb5`
與 `5b321d21c1496b4832a5d838289e869126a64ab2c25e5ae04dc5c7881e3bd0f4`。
**畫面相同不能推論完整狀態相同或防護盾無效。**

空白鍵＋Enter 的來源族及計數與 §37 相同，仍只有 BEAM #0–#2、FIGHT #0–#3、
ENEMY00 #3–#5／#18–#20 及已證實遮罩。終點畫面與 §37 相同，cycles 少 347；
RAM 為 `e32401173f44a05c8c4f9ea379803cad998214e00a19724449dc74b8af1a62da`。
不宣稱與單按空白鍵同狀態，也不據此推論能力語意。
這三組沒有擴充 sprite 圖號覆蓋；不反覆重跑首場以尋找同樣缺前提的圖塊。
其餘 FIGHT #4–#11／ALLY #12–#15 需從具備實際前置條件的正常玩家狀態續查。

### 40.3 收據、來源版本與重跑入口

本機前綴依序為 `workplace/hd/battle-key-enter-v1-20261001`、
`battle-key-none-v1-20261001`、`battle-key-space-enter-v1-20261001`；各有 `.json`、
逐事件原始來源及 before／after 色號陣列。獨立收據為
`workplace/hd/battle-key-independent-v1-20261001.json`。
觀察器的已跑版本保存為 `observe_battle_effects-keys-v1-20261001.go`（前兩組）與
`observe_battle_effects-keys-v2-20261001.go`（雙鍵）；各與對應收據雜湊相符。
已驗 Python bytes 保存為 `verify_battle_key_paths-v1-20261001.py`，SHA-256
`10ec052663649151e4f2e93d5e851a78d2a7c3bc1146d4f2c3e51fdac131e570`。
現行工具後續只修正範圍文字／繁體字，不把新 hash 登記為舊收據來源。
逐檔回查入口為 `workplace/hd/battle-key-source-manifest-v1-20261001.json`：
4,103 檔、177,736,749 bytes，含版本化來源與三組原始色號／貼圖；
解碼器 bytes 另存 `battle-key-pbl-v1-20261001.py`。原檔保持各自版本，不複製為其他 oracle。

使用 §35 的既有離線 Docker、唯讀原版、UID/GID 1000:1000、CPU 1、資源上限及
有界逾時；Go 以非 login shell 啟動。輸出前綴必須全新：

```sh
# Go 容器；三種輸入分別執行，不覆寫既有收據
go run -mod=readonly tools/hd/observe_battle_effects.go -deltas -key enter -out workplace/hd/新的唯一前綴
go run -mod=readonly tools/hd/observe_battle_effects.go -deltas -key none -out workplace/hd/另一唯一前綴
go run -mod=readonly tools/hd/observe_battle_effects.go -deltas -key space -guard-at 87000000 -out workplace/hd/第三唯一前綴

# Python 容器：消費本節已保存三組樣本；/orig 掛原版唯讀目錄
python tools/hd/verify_battle_key_paths.py --out workplace/hd/新的唯一核對檔名.json
```

開跑 load 7.39／8.57／13.05，只作決定性核對，不宣稱即時音訊或幀率。
工具摘要曾把 JSON null 當陣列呼叫 len，修正摘要工具後讀取成功；原版樣本沒有失敗。
024 §1.5 仍 DRAFT；本節未改正式程式、原版規則、資料格式、存檔或美術驗收狀態。

## 41. ENEMY05 前十五圖號的 HD 候選（2026-10-01）

### 41.1 來源、參照與生成

原版 `ENEMY05.PBL` 實際 30 張，SHA-256
`2b390a5c4a5a2c6e49a9e27b02c89dd839c6c932dad0f568aee6b88e2397180a`。
本批只處理 #0–#14，均 24×32；定位採圖號及 PBL 檔案偏移，不新增反組譯位址。
每三張成組供視覺審查；不由像素相似度證明角色身分、動作語意或用途。
`workplace/hd/redraw/prepare_enemy05_references.py` 由原始色號及 `docs/re/005` 主畫面／
戰鬥色盤建立五張 1152×512 參照，本批不含 2／8／A 敏感色號；全部參照與解碼 RGB
逐像素相符。偏移、色號雜湊及工具來源見 `ENEMY05-references-v1-20261001.json`。
尺寸、來源與參照吻合為 confirmed；角色辨識、造型與動畫忠實度仍待審查。

使用 `imagegen` 技能及內建 `image_gen`，五組分別生成，不使用 CLI／API 備援。
原版參照提供構圖、配色與各格差異；ENEMY04 第 1 組 v2 只作輪廓及平塗風格參照。
要求原版裁切、純黑底、等寬三格、圖集 9:4，不縮小或平移人物避框。
服務端版本及固定生成 seed 未提供，不保證重送提示詞得到相同圖；可重跑的轉換
以已保存的生成 PNG 為輸入。七張原圖均已複製到專案，工具預設原圖保留。

### 41.2 版本及視覺限制

| 組／來源圖號 | 版本 | 限定審查 |
|---|---|---|
| 0／#0–#2 | v1 | 中央長形體有三格彎曲差異；大輪廓、比例與細節仍待核對 |
| 1／#3–#5 | v1／v2 | 原版已有分叉端部，不能一律判為新增指狀突起；輪廓及配色仍待審查，訂正證據見 §42 |
| 2／#6–#8 | v1 | 下半部三格青綠／紅藍／洋紅差異可見；部分輪廓重新解讀，未驗收 |
| 3／#9–#11 | v1 | 青綠黃色大輪廓保留；三格彎曲與色帶位置仍待核對 |
| 4／#12–#14 | v1／v2 | 原版已有頂端灰條及右側斜長形塊；兩版留白及配色問題已依原始色號續修，現行 v4 見 §42，整體未驗收 |

原版輪廓判讀依 §42 訂正，舊結論與形成原因保留於 `000-overturned-claims.md` 及 WORKLOG。
後續依原始色號核對，再作單項修正；不能把不確定的物件語意當成刪除形塊的理由。
本批歷史候選及收據保留，沒有新增正式接入或變更遊戲規則。

### 41.3 產物、核對與入口

全部產物在既有 `workplace/hd/redraw/`，以下為完整入口索引：

- `ENEMY05-generation-jobs-v1-20261001.json`／`ENEMY05-generation-jobs-v2-20261001.json`：七次生成的完整提示詞及參照用途。
- `ENEMY05-selected-v1-20261001.json`／`ENEMY05-selected-v2-20261001.json`：工具原始保存位置及專案路徑。
- `ENEMY05-group<組>-<版本>-generated-20261001.png`：七張原始生成圖。
- 每版本三張 `ENEMY05-group<組>-<版本>-frame-00/01/02-20261001.png`：共 21 張 72×96 候選。
- `ENEMY05-first15-comparison-v1-20261001.png`／`ENEMY05-first15-comparison-v2-20261001.png`：1296×1440；五列各左三格原版、右三格候選，v2 只換第 1／4 組。
- `process_enemy05_candidates.sh`：依實際圖寬等分、全圖縮放，沒有另裁人物、移位或手動改圖。
- `verify_enemy05_candidates.py`／`ENEMY05-candidates-verification-v1-20261001.json`：原版偏移、解碼來源、PNG CRC／完整解碼／尺寸／UID/GID、輸入輸出雜湊及相鄰格差異。

45 張 PNG 核對通過：五張原版參照、七張生成圖、21 張小圖、十張分組對照、兩張總對照。
相鄰候選格 RGB 差異 4,097–6,091 像素；第 4 組兩版頂端留白均 1 列，其餘均 0。
這些數值只描述圖像，不證明造型、動作或正常玩家路徑吻合。
研究來源涵蓋由 **75 增為 90 個圖號**（ENEMY00–05 各 #0–#14），正式仍 **0/360**。
其餘圖號、盟友、小圖塊、效果、遊戲內驗證及全部美術目標維持。

工具為 Python 3.13.15、ImageMagick **7.1.2-12 Q16-HDRI**；實際映像
`sha256:87998ec1b8127b2f73f626f74f7b05e8827f9d7605fa52da5370588f7e53cee1`。
使用既有 `dpokidov/imagemagick:latest`，不同於 §28 的 6.9 版本，不冒稱濾波相同。
`convert` 棄用警告不影響本次成功退出。映像檢查首命令因 `.Config.Cmd` 欄位不存在
而失敗，改讀實際 `.Config` 確認入口；轉換用 `--entrypoint sh`，沒有建立重複映像。

使用 §35 的離線 Docker、UID/GID 1000:1000、CPU 1、資源上限及有界逾時，原版唯讀。
下列在容器 `/src` 執行；已有輸出會拒絕覆寫，重跑使用乾淨研究工作樹／新輸出目錄：

```sh
python workplace/hd/redraw/prepare_enemy05_references.py
# 影像容器可指定 ENEMY05_OUT_DIR 為容器內已建立的空目錄
sh workplace/hd/redraw/process_enemy05_candidates.sh
# Python 容器：/orig 掛原版唯讀目錄，核對收據必須尚不存在
python workplace/hd/redraw/verify_enemy05_candidates.py
```

候選、參照及原版均只留本機；未公開、發行、commit、push 或寫入 Issue。

## 42. ENEMY05 輪廓判讀訂正與有限修正（2026-10-01）

### 42.1 原始色號證據

沿用 §41 的 ENEMY05.PBL、SHA-256、Python 3.13.15 及 PBL 圖號／檔案偏移基準，
沒有新增反組譯位址。`verify_enemy05_refinements.py` 獨立讀取 #3–#5、#12–#14：

- #3–#5 的左端第 21 列至少有兩個分離的青綠色號 3；原版已有分叉，不能把指狀端部一律當作新增。
- #12–#14 頂列 `(2,0)`／`(15,0)` 均為灰色色號 7；左側經 `(2,1)`、`(3,2–3)`、`(4,4–5)` 連接，右側經 `(14,1–2)` 連接。原版已有兩條頂端灰形塊，不能要求全部刪除。
- 右側 `(16,8,8,10)` 已有斜長形塊。#12 是 6／E 黃棕及 B 青綠，沒有 4／C 紅色；#13 有 C 紅色，#14 有 4／C 紅色。三格配色不同，不能都畫成黃紅條紋。

色號、座標、來源雜湊及解碼陣列是 confirmed；指、天線或武器的遊戲語意尚未證實。
這些原始來源足以否定 §41 當時「分叉、頂端細條、斜長形塊全屬新增」的判斷，
不是靠候選本身自證。舊判讀、提示詞及圖片不刪，勘誤入口 `000-overturned-claims.md`。

### 42.2 單項修正及結果

使用內建 image_gen，先以第 4 組 v2 作目標、原版作配色參照生成 v3；再以 v3 作目標，
原版作頂端座標參照生成 v4，不重新整組設計、不人工塗改資產。
第 0 格觀察矩形 `(48,24,24,30)` 的青綠主色像素 10→37→38，紅色主色 44→32→32；
矩形亦含既有其他色區，不能把這些計數直接當整張配色忠實度。
v2／v3 三格頂列非黑均 0，v4 為 **8／6／7**，灰條已接到頂邊，不以人物整體放大避框。

單項提示詞不代表區外逐像素不變：配色 v2→v3 在觀察矩形外差 4,797–4,912 像素；
灰條 v3→v4 在兩個頂端觀察矩形外差 4,436–4,585。矩形不是批准覆繪區。
本節只證明有限色彩與頂端改善，整體構圖、輪廓位置、動作與正常玩家呈現仍未驗收。
舊 45 PNG 雜湊未變，新 **11 PNG** 完整解碼、CRC、尺寸與 UID/GID 通過；
來源涵蓋仍 90 個圖號，沒有把修正版數量當成新圖號。

### 42.3 來源及重跑入口

全部留在 `workplace/hd/redraw/`：

- `ENEMY05-group4-v3-generation-job-20261001.json`／`ENEMY05-group4-v4-generation-job-20261001.json`：兩次完整提示詞、參照及編輯目標。
- 對應 `ENEMY05-group4-v3-selected-20261001.json`／`v4`：工具原生保存位置與專案路徑。
- 第 4 組 v3／v4 的 `generated-20261001.png`、三格 `frame-00/01/02-20261001.png` 及 `comparison-v3/v4-20261001.png`。
- `ENEMY05-first15-refined-comparison-v1-20261001.png`：最新十五格，依序第 0 組 v1、第 1 組 v2、第 2／3 組 v1、第 4 組 v4；左原版、右候選。
- `process_enemy05_group4_refined.sh`／`verify_enemy05_refinements.py`：轉換及原始色號／CRC／區外變動核對入口。
- `ENEMY05-refinements-verification-v1-20261001.json`：完整來源事實、雜湊及指標。
- 已驗 Python bytes 另存 `verify_enemy05_refinements-v1-20261001.py`，SHA-256 `7f6411eb0d7afd08fc7deabd49f5ff058cdb5834886151504fab4cf0294a9a4d`；現行工具僅修正繁體字，不把新 hash 冒充舊收據。

既有 ImageMagick 7.1.2-12 映像及 ID 沿用 §41，使用 `magick`；Python 3.13.15。
使用 §35 的離線 Docker、UID/GID 1000:1000、CPU 1、資源限制及有界逾時，原版唯讀。
容器 `/src` 入口：`sh workplace/hd/redraw/process_enemy05_group4_refined.sh`，可指定
已存在空目錄 `ENEMY05_REFINED_OUT_DIR`；核對入口為
`python workplace/hd/redraw/verify_enemy05_refinements.py`。已有輸出拒絕覆寫，重跑用乾淨研究工作樹。
不再用 §41 的廣泛 glob 核對器混入新增版本；本節核對器明確消費舊收據清單及新版本。

## 43. ENEMY06 前十五圖號的 HD 候選（2026-10-01）

### 43.1 來源、版本與有限視覺審查

原版 `ENEMY06.PBL` 實際 30 張，SHA-256
`489364bda2f9356d3b386ae71ad90cfc6064a1e04d33a5569e5e653a27d220a7`；
本批 #0–#14 均 24×32，不含色盤敏感色號 2／8／A。定位仍採 PBL 圖號與檔案偏移，
沒有新增反組譯位址。五組原版參照逐像素 RGB 核對相符；成組關係供視覺審查，
不由相似度證明角色身分、動作語意或用途。

使用 imagegen 技能及內建 image_gen，完成六次生成：v1 清單只執行第 0–2 組，
第 3／4 組 v1 未呼叫；v2 清單執行第 2–4 組。不用 CLI／API 備援。
前三次以原版作造型參照、ENEMY05 第 4 組 v4 只作畫法參照；
第 2 組 v1 卻借用了後者的兩條頂端細條，**不採用**，生成原圖保存作失敗對照。
原版 #6–#8 頂列只有 x=7–12 的灰白主輪廓，沒有兩側細條；獨立解碼確認。
改用單一原版參照後，第 2 組 v2 兩側頂列非黑像素由 5／6／8 改為 **0／0／0**，
不把此有限改進當作整體造型通過。第 3／4 組也只用各自原版參照。

| 組／來源圖號 | 最新候選 | 審查與限制 |
|---|---|---|
| 0／#0–#2 | v1 | 頂端直條及右側藍紫→紅→洋紅色區變化可見；形體比例仍待審查 |
| 1／#3–#5 | v1 | 中部洋紅→紅色區變化可見；黑色空間、灰白小形塊及輪廓仍待核對 |
| 2／#6–#8 | v2 | 移除錯借的頂端細條，右側下垂→平伸→上抬各自保留；v1 不採用，正常玩家動作未驗 |
| 3／#9–#11 | v2 | 灰色分叉、洋紅紅色橫帶及三格左上姿勢差異可見；比例、分叉數量與輪廓未驗收 |
| 4／#12–#14 | v2 | 黃棕主體及上方青綠／洋紅／紅色差異保留；內部細線與形體忠實度仍待審查 |

### 43.2 產物與技術核對

全部在既有 `workplace/hd/redraw/`；完整入口如下：

- `prepare_enemy06_references.py`／`ENEMY06-references-v1-20261001.json`：五組 1152×512 原版參照、來源偏移與色號雜湊，沿用 §41 的原始解碼方法，不改舊來源工具。
- `ENEMY06-generation-jobs-v1-20261001.json`／`ENEMY06-generation-jobs-v2-20261001.json`：完整提示詞及參照用途。
- `ENEMY06-selected-v1-20261001.json`：六次實際執行的原生保存位置、專案路徑、未採用標記與最新五組選擇；未呼叫項目明確註明。
- 六張 `ENEMY06-group<組>-<版本>-generated-20261001.png`、18 張 `frame-00/01/02-20261001.png`，六張分組對照及總對照。
- `ENEMY06-first15-comparison-v1-20261001.png`：1296×1440，五列各左原版三格、右最新候選三格。
- `process_enemy06_candidates.sh`／`verify_enemy06_candidates.py`：不覆寫的全圖切分／縮放及來源／檔案核對。
- `ENEMY06-candidates-verification-v1-20261001.json`：原版頂列證據、36 PNG 雜湊／CRC／完整解碼／尺寸／UID/GID、動作差異及工具來源。

36 PNG 均通過，六張圖集轉成 **18 個版本化 72×96 候選**，其中最新十五格涵蓋十五圖號。
所有格可見形塊頂部留白均 0；相鄰格 RGB 差異 3,479–5,489，只描述圖像，不是動作忠實度。
研究來源涵蓋 **90→105 個圖號**（ENEMY00–06 各 #0–#14），正式仍 **0/360**。
正常遊戲動畫、中文覆繪、8×8 動態接入、存讀檔及美術驗收尚未完成。

### 43.3 重跑與剩餘範圍

使用 §41 的 Python 3.13.15、固定 ImageMagick 7.1.2-12 映像、UID/GID 1000:1000、
CPU 1、離線、資源上限、有界逾時及原版唯讀掛載。容器 `/src` 入口：

```sh
python workplace/hd/redraw/prepare_enemy06_references.py
sh workplace/hd/redraw/process_enemy06_candidates.sh
python workplace/hd/redraw/verify_enemy06_candidates.py
```

已有輸出拒絕覆寫；可指定已存在空目錄 `ENEMY06_OUT_DIR` 重生轉換，核對收據及參照
重跑則使用乾淨研究工作樹。生成服務端版本／固定 seed 未提供，可重現範圍是已保存 PNG 的轉換。
首次核對器準備命令因 shell 引號造成語法錯誤，容器及 Python 未啟動，沒有寫入來源；
改用儲存庫編輯工具建立實際腳本，再由相同 Python 映像乾淨執行成功，未改產品或放寬期望。
開跑 load 26.74／16.03／10.76，只作決定性影像處理，不宣稱即時幀率／音訊。
ENEMY07–11、其餘圖號、全部小圖塊／盟友／特效等完整目標維持；
024 §1.5 混色問題仍未回答，未接入正式特效，未公開、發行、commit、push 或寫入 Issue。

本輪來源保全入口為 `workplace/hd/redraw/ENEMY05-ENEMY06-source-manifest-v1-20261001.json`：
118 檔、24,233,527 bytes，逐項尺寸、雜湊及 UID/GID 通過，含兩批歷史及修正版、
完整提示詞、實際生成來源路徑、原版解碼索引與已驗工具版本。
解碼器 bytes 另存 `pbl-enemy05-06-v1-20261001.py`；本機原檔按版本保存，不覆寫舊收據。

## 44. ENEMY07／08 前十五圖號的 HD 候選（2026-10-01）

### 44.1 原版來源與尺寸

兩個 PBL 各含 30 張；本次只處理各 #0–#14。來源定位採 PBL 圖號與檔案偏移，
沒有新增反組譯語意。輸入 SHA-256：

- `ENEMY07.PBL`：`c2924057e1d704d30be7a644c870b0879bb72e155183b6ab7c4c6e864904d519`。
- `ENEMY08.PBL`：`eb4e8673858a5425bba68edba74cad1138caf2c64d58d429e40c7d7a7ec072a7`。

ENEMY07 十五張及 ENEMY08 #0–#11 為 24×32，候選轉成 72×96；
ENEMY08 #12–#14 為 **24×24，候選維持 72×72**，不拉成直式。
本批不含色盤敏感色號 2／8／A。十組原版參照由色號逐像素核對 RGB，不符 0。
尺寸、偏移與色號為 confirmed；成組及形狀描述供視覺審查，不證明角色身分或動作語意。

### 44.2 生成及有限審查

沿用 imagegen 技能，以內建 image_gen 完成十次生成，每次只附該組原版參照，
不借用其他角色作形體或風格來源。完整提示詞要求保留三格差異、裁切、比例與空白，
採平滑輪廓及節制陰影，不新增部件；實際結果仍須獨立審查，提示詞不是符合證據。

實際檢視兩張總對照：ENEMY07 第0組中央白→青綠、第三組中央青藍→黃，以及
ENEMY08 第3組下方紅→黃、第4組灰紅→青藍變化仍可見；這只是有限觀察。
ENEMY07 第1組端部曲線、第2組右側小形塊、第4組零散細節，及 ENEMY08
內部陰影、色區與輪廓仍有推測成分，**本批均未通過美術驗收**。
不以原版已有分叉／長形部位本身判新增，不由生成圖推定遊戲用途。

62 PNG 來源／CRC／完整 RGB 解碼／尺寸／UID/GID 核對通過，含十張原版參照、
十張生成圖、三十張候選、十張分組對照及兩張總對照。方形錯尺寸負對照能失敗。
相鄰候選差異 2,319–5,782 像素，僅為圖像描述；ENEMY07 第1組中格頂部留白1列，
第4組三格各23列；ENEMY08 第4組各32列，不把非零差異或留白計數當動作忠實度。
研究來源涵蓋 **105→135 個圖號**（ENEMY00–08 各 #0–#14），正式美術仍 **0/360**。
全體造型、配色、正常玩家動畫、中文覆繪、8×8 接入及存讀檔未驗；沒有正式程式變更。

### 44.3 來源與重跑入口

所有新檔均在既有 `workplace/hd/redraw/`，完整索引：

- `prepare_enemy07_08_references.py`、`ENEMY07-references-v1-20261001.json`／`ENEMY08-references-v1-20261001.json`：來源偏移、尺寸、色號雜湊與參照。
- `ENEMY07-08-generation-jobs-v1-20261001.json`：十次完整提示詞及單一參照用途；工具為內建 image_gen。
- `ENEMY07-08-selected-v1-20261001.json`、`copy_enemy07_08_candidates.py`：實際工具保存位置、專案路徑及生成原圖雜湊。
- `ENEMY07/08-group<組>-v1-generated-20261001.png`、對應三張 `frame-00/01/02-20261001.png` 及 `group<組>-comparison-v1-20261001.png`：版本化原圖、候選與分組對照。
- `ENEMY07-first15-comparison-v1-20261001.png`／`ENEMY08-first15-comparison-v1-20261001.png`：1296×1440／1296×1368，五列各左原版三格、右候選三格；方形列高216。
- `process_enemy07_08_candidates.sh`、`verify_enemy07_08_candidates.py`、`ENEMY07-08-candidates-verification-v1-20261001.json`：切分及全圖縮放、獨立來源核對、PNG／尺寸／負對照與工具雜湊。
- `preserve_enemy07_08_sources.py`、`ENEMY07-ENEMY08-source-manifest-v1-20261001.json`、`pbl-enemy07-08-v1-20261001.py`：明確檔案清單的來源保全、工具 bytes、擁有權稽核。

Python 3.13.15；ImageMagick 7.1.2-12 Q16-HDRI，映像識別沿用 §41，
本輪確認入口仍為 convert，轉換用 `--entrypoint sh` 與 `magick`，沒有另建映像。
容器 `/src` 的重跑入口如下；已存在輸出拒絕覆寫，須使用乾淨研究工作樹，
原版 `/orig` 及 `/src/workplace/original` 均唯讀。工具預設路徑不以模型語意映射圖號。

```sh
python workplace/hd/redraw/prepare_enemy07_08_references.py
python workplace/hd/redraw/copy_enemy07_08_candidates.py
sh workplace/hd/redraw/process_enemy07_08_candidates.sh
python workplace/hd/redraw/verify_enemy07_08_candidates.py
python workplace/hd/redraw/preserve_enemy07_08_sources.py
```

複製工具須將既存工具輸出掛到 `/generated:ro`；轉換重現範圍是保存 PNG，
生成服務沒有提供固定 seed／服務版本，不承諾重新生成相同圖片。
Docker 依 §35 使用 --rm、CPU1、UID/GID1000、資源限制及有界逾時、離線；
起跑 load 4.05／6.51／12.23，只作影像處理，不下幀率／音訊結論。
8×8、美女在後、框在前及原版排版維持；其餘 ENEMY09–11、全部小圖塊／盟友／
特效與正式驗收繼續。混色選擇未回答，024 §1.5 維持 DRAFT；候選僅留本機，
未 commit、push、寫入 Issue、公開或發行。

本輪來源 manifest 明確列75檔、15,428,519 bytes，逐項雜湊／尺寸／UID核對通過；工作根67,127項未找到root-owned或誤建.md目錄。原版與舊版收據未覆寫。

## 45. ENEMY09–11 候選、戰鬥色盤與位置篩查（2026-10-01）

### 45.1 原版來源

三個 PBL 各含30張，本批處理各 #0–#14，全部24×32。SHA-256：

- `ENEMY09.PBL`：`f6e09a218300e9848360493ecac122617639475106753350944ef078b91e71d9`。
- `ENEMY10.PBL`：`d371d4065a76a374329128c6d4a35f52f578322f7f2e4458252592b26fa0a482`。
- `ENEMY11.PBL`：`f5c29f254baf0ec1ef2dc9db61596efbd2cfb941ed992534358899386a29dc44`。

採 PBL 圖號／檔案偏移定位，沒有新增反組譯語意。十五組原版參照由原始色號
逐像素核對 RGB，不符0。只有 ENEMY10 #2 含敏感色號2，實際2個原版像素；
使用005與本研究§39已驗主畫面／戰鬥色盤，2為棕色 `AA5500`，不是預設綠色。
參照×16後，錯用預設綠色負對照相差 **512像素**。這證明參照色盤檢查有效，
不證明該圖已在正常玩家路徑觸發，也不證明生成候選的全體配色。

十二個檔案重新盤點：177張24×32、3張24×24、180張16×16；
每個檔案 #0–#14 正好對應24像素寬，#15–#29 為16×16，不符0。
尺寸及色號為 confirmed；角色身分、每張用途與正常輸出路徑不能由尺寸推定。

### 45.2 初版生成、技術核對及邊界限制

imagegen 技能的內建 image_gen 完成十五次初版生成，每次只有該組原版作參照；
十五張原生圖、45張72×96候選、完整提示詞及原始保存位置均留本機。
93 PNG來源／CRC／完整RGB解碼／尺寸／UID/GID核對通過；錯尺寸負對照會失敗。
相鄰候選差異3,386–5,882像素，只描述圖片，不作原版動作忠實度判準。

實際檢視三張總對照：ENEMY09第2組的上方部位變化、ENEMY10第1組下方彎曲變化、
ENEMY11第1組紅→黃色區及下方伸展變化可見。內部陰影、黑色空隙、細部輪廓、
形體比例及生成解讀仍待審查，本批**未通過美術驗收**。

以原始非零色號的邊界×3，對照候選RGB大於16的可見邊界，找出具體位置偏差：

- ENEMY09第4組右界原版60／66／72，候選59／70／69；第2組後兩格原版頂端3，候選2。
- ENEMY10第0組三格原版底界96，候選95；第4組原版頂端57／54／48、底界96，候選54／52／46、底界93。
- ENEMY11第1組首格底界90，候選87；第2組原版底界93，候選95；第4組左右及底端亦有偏差。

邊界是篩查指標，不能因邊界相同就宣布全輪廓符合。只處理已量測差異，
不把原版本有長形／分叉部位一律刪除，亦不靠縮小整體補完原版裁切。
來源涵蓋 **135→180個圖號**（ENEMY00–11各 #0–#14），正式美術仍 **0/360**。
這個180不包含§37–39的 ENEMY00 #18–#20效果候選；後者保留，不因續做小圖塊而重生成。

### 45.3 ENEMY10 第4組單項位置修正

以保存的v1作編輯目標、原版作位置參照，內建 image_gen 生成v2，要求保留畫法並
向下修正，沒有直接塗改舊資產。新6 PNG技術核對通過，舊93 PNG雜湊全部未變。

| 格 | 原版邊界 `(左,上,右,下)` ×3 | v1 | v2 |
|---|---|---|---|
| 0 | `(0,57,72,96)` | `(0,54,72,93)` | `(0,55,72,94)` |
| 1 | `(0,54,72,96)` | `(0,52,72,93)` | `(0,53,72,94)` |
| 2 | `(0,48,72,96)` | `(0,46,72,93)` | `(0,46,72,94)` |

底部黑邊3→2列，位置只有有限改善，**未達原版精確邊界**。
把v1依要求平移3／2／2列後，與v2仍差3,263／3,319／3,622像素，不能稱純平移或
編輯區外逐像素保持。停止追加同類生成，回查 imagegen 單項修改指引；
保存兩版與未通過結果，其他素材工作繼續，沒有降低正式排版要求。
全體造型、動畫、中文覆繪、8×8接入與存讀檔仍未驗，v2並非正式接受版本。

### 45.4 產物、工具及重跑入口

全部在既有 `workplace/hd/redraw/`，完整入口索引：

- `prepare_enemy09_11_references.py`、各 `ENEMY09/10/11-references-v1-20261001.json`：十五組原版參照、圖號、偏移、色盤與輸入雜湊。
- 各 `ENEMY09/10/11-generation-jobs-v1-20261001.json`、`ENEMY09-11-selected-v1-20261001.json`：十五次完整提示詞、參照用途、實際工具路徑與保存原圖雜湊。
- `copy_enemy09_11_candidates.py`、`process_enemy09_11_candidates.sh`：原圖保存、等分全圖縮放，拒絕覆寫舊版。
- 各 `ENEMY09/10/11-group<組>-v1-generated-20261001.png`、三張 `frame-00/01/02-20261001.png` 與 `group<組>-comparison-v1-20261001.png`：原生圖、候選與分組對照。
- `ENEMY09-first15-comparison-v1-20261001.png`、`ENEMY10-first15-comparison-v1-20261001.png`、`ENEMY11-first15-comparison-v1-20261001.png`：1296×1440，五列各左原版三格、右候選三格。
- `verify_enemy09_11_candidates.py`、`ENEMY09-11-candidates-verification-v1-20261001.json`：來源RGB、93 PNG、負對照、邊界與相鄰格差異。
- `ENEMY10-group4-v2-generation-job-20261001.json`、`ENEMY10-group4-v2-selected-20261001.json`：單項編輯提示詞、目標／位置參照、原生保存路徑。
- `ENEMY10-group4-v2-generated-20261001.png`、三張v2小圖、`ENEMY10-group4-comparison-v2-20261001.png`、`ENEMY10-first15-comparison-v2-20261001.png`：有限修正版；總對照只換第4組。
- `process_enemy10_group4_refined.sh`、`verify_enemy10_group4_refined.py`、`ENEMY10-group4-refined-verification-v1-20261001.json`：單項轉換、舊93圖不變、精確邊界未符合及純平移差異收據。
- `preserve_enemy09_11_sources.py`、`preserve_enemy09_11_refined.py`、`ENEMY09-ENEMY11-source-manifest-v1-20261001.json`／`v2`、`pbl-enemy09-11-v1-20261001.py`：明確清單來源保全與已驗解碼器bytes；v1不覆寫。

Python3.13.15，映像 `sha256:540c7d91f98ff6880174c40e99067bf5941eb54d818a7a5e094d188b196a934d`；
ImageMagick7.1.2-12 Q16-HDRI，固定映像識別沿用§41。
首次同時讀Python及影像映像的Entrypoint模板失敗：Python映像沒有該欄，
改讀完整Config確認預設Cmd為python3後，以相同既有映像成功執行；這是檢查命令問題，
沒有重建映像或改產品。影像容器用 `--entrypoint sh` 及 `magick`。

依§35離線Docker、--rm、CPU1、UID/GID1000、資源限制及有界逾時，
原版 `/orig` 與 `/src/workplace/original` 均唯讀。容器 `/src` 入口：

```sh
python workplace/hd/redraw/prepare_enemy09_11_references.py
python workplace/hd/redraw/copy_enemy09_11_candidates.py
sh workplace/hd/redraw/process_enemy09_11_candidates.sh
python workplace/hd/redraw/verify_enemy09_11_candidates.py
python workplace/hd/redraw/preserve_enemy09_11_sources.py
# 保存v2生成原圖後，另跑限定修正版；不覆寫v1收據
sh workplace/hd/redraw/process_enemy10_group4_refined.sh
python workplace/hd/redraw/verify_enemy10_group4_refined.py
python workplace/hd/redraw/preserve_enemy09_11_refined.py
```

已有輸出拒絕覆寫；重跑用乾淨研究工作樹或已存在空目錄 `ENEMY09_11_OUT_DIR`。
初版核對器只在含初版93張PNG的乾淨工作樹重跑；新增v2後沿用限定修正版核對器，
由舊收據的明確清單驗93圖不變，不把新檔混入舊收據。
複製時工具輸出須掛到 `/generated:ro`；v2依selected內default_path讀同目錄原生檔、
另存project_path並核對sha256，原始檔保留。生成服務未提供版本／固定seed，
重現限於保存PNG的轉換。起跑load8.86／10.83／11.03，不作即時幀率或音訊結論。

初版來源manifest112檔、24,227,805 bytes；含修正版的v2明確列 **125檔、25,587,550 bytes**，
全部尺寸／雜湊／UID核對；工作根67,248項未發現root-owned或誤建.md目錄。
後續180個16×16圖號、盟友、全效果、正式驗收與交付維持；小圖塊按已驗來源續作，
不由畫圖模型猜玩法。8×8與原版排版／人物在後框在前維持；混色未回答，024 §1.5仍DRAFT。
沒有正式程式改動、commit、push、Issue寫入、公開或發行。

## 46. ENEMY00／01 的16×16小圖塊候選（2026-10-01）

本批接續已核對的實際尺寸，製作ENEMY00／01各 #15–#29；不是完整角色縮小版。
沿用imagegen技能及內建image_gen，九組各附唯一原版三格參照。ENEMY00 #18–#20已有
§37–39的候選與有限正常來源，本輪沿用，沒有重新生成或覆寫。先核對知識路由及
`local/project-document-responsibilities.md`，沿用既有研究檔、CONTEXT、worklist及唯一WORKLOG。

### 46.1 原版來源與證據層級

| 輸入 | SHA-256 | 本批 |
|---|---|---|
| ENEMY00.PBL | `8241b0ea73b1e7402b13adc434f10a5b1288f88f01fcfc3ba5710e4923fe6067` | 各檔30張，本批#15–#29均16×16 |
| ENEMY01.PBL | `22f664050ce7ffa4ea0f6941c9c91cd1ab43671ea5b53491f6799f78ba8e64af` | 各檔30張，本批#15–#29均16×16 |

圖號、PBL檔案偏移、尺寸、原始色號及非零外框為confirmed（獨立PBL解碼）。
每三個連續圖號僅為製圖分組，不證明全部圖號的玩法用途、實際出現順序或輸出模式。
每組參照1152×384：三張16×16按原序橫接、最近鄰×24。十組完整RGB重算不符0；
兩檔此批均無色號2／8／A，不重開已閉合的戰鬥色盤追查。

來源非零像素外框在收據以右／下界不含該座標表示。候選黑色空間轉透明只屬
素材預備，**不證明原版採透明疊圖**，未驗路徑及未知合成模式保持未知。
ENEMY00 #18–#20有限正常來源的已證實範圍依§37，不能推展至其餘27張。

### 46.2 生成、轉換與技術核對

九次內建image_gen各產出2172×724透明原生圖；全圖縮到144×48後按原序拆成
三張48×48 RGBA，不裁掉生成圖邊界、不改原生檔案。生成服務未提供版本／固定seed，
可重現的是保存原生PNG之後的裁切／縮放及獨立驗證，不保證重新生成同圖。
ENEMY00第1組由舊三張48×48候選橫接作對照，不製造另一組替代候選。

- **新增27個圖號**、沿用3個；16×16小圖塊候選涵蓋3→30/180，剩餘150個維持範圍。
- 新68張PNG：10參照、9原生圖、9縮放圖集、1沿用圖集、27小圖、10分組對照、2總對照。
  加上原位置的三張沿用圖，共71張通過CRC、完整解碼、尺寸、通道及UID/GID1000核對。
- 27張新圖均48×48、RGBA、可見且有透明空間；完全透明像素261–1439／2304。
  此數字不代表透明邊緣、輪廓或效果混色已通過。
- 原版參照完整RGB不符0；分組黑底對照由原版索引＋候選RGBA獨立推導，不符0，
  總對照的追加次序與完整RGB相同。參照水平位移一像素負對照差**90,696**像素；
  完全不透明候選的負對照被拒絕。
- 三張沿用圖的SHA-256與舊 `battle-effects-candidates-v3-20261001.json` 完全相同。

新圖尺寸及透明通道通過，只算檔案技術核對。ENEMY00–11各#0–#14的180圖號預覽
另計，不把小圖塊混入原本180計數；全體正式美術仍**0/360**，未正式接入本批。

### 46.3 精確位置未達整批驗收

以alpha≥128量測可見外框，原版非零外框×3獨立比對：新候選只有**9/27**完全相同，
沿用候選**1/3**，合計10/30。alpha≥16的低透明邊緣外框亦完整保存。
即使外框相同，內部形狀／配色仍可能偏離，不能升格為完整輪廓或美術驗收。

| 抽樣 | 原版外框×3 `(左,上,右,下)` | 候選alpha≥128 | 結果 |
|---|---|---|---|
| ENEMY00 #15 | `(0,0,48,42)` | `(1,2,48,44)` | 新增上下留白及高度偏差 |
| ENEMY00 #22 | `(0,0,48,39)` | `(1,6,47,38)` | 頂端多6列、左右各少1列 |
| ENEMY00 #25 | `(0,3,48,48)` | `(1,3,47,46)` | 底界少2列 |
| ENEMY01 #16 | `(0,6,48,36)` | `(0,12,48,35)` | 頂端多6列，原版高度關係未保持 |
| ENEMY01 #21 | `(0,6,48,36)` | `(0,6,41,38)` | 右界少7列、底界多2列 |
| ENEMY01 #24 | `(0,0,48,36)` | `(0,4,48,36)` | 頂端多4列 |

實際檢視兩張總對照，部分原版斷開形狀被連接成圓滑帶狀／枝狀圖案，另有留白及
高度偏差。這是本批未通過的美術問題，不能靠PNG尺寸正確掩蓋，不改使用者的原版
位置與比例要求；保留全部原圖及量測，沒有反覆生成相同組「直到看似通過」。
輪廓、細部配色、三張動作關係、正常玩家輸出、中文覆繪、8×8遮格及讀檔未驗。
8×8原點、美女在後框線在前、原版排版維持；一般透明／透光混色未回答，024 §1.5仍DRAFT。

### 46.4 完整入口索引與重跑

全部沿用 `workplace/hd/redraw/`，沒有新根目錄或交付目錄：

- `prepare_enemy00_01_small_references.py`、`ENEMY00-small-references-v1-20261001.json`、`ENEMY01-small-references-v1-20261001.json`：原版來源、偏移、尺寸、色盤、十組參照及其完整RGB。
- `ENEMY00-small-reuse-v1-20261001.json`：三張舊圖、舊收據雜湊與沿用範圍；實際資產仍在 `workplace/hd/art-in/battle-effect-<18或19或20>-48-v3-20261001.png`。
- `ENEMY00-01-small-generation-jobs-v1-20261001.json`、`ENEMY00-01-small-selected-v1-20261001.json`：九次完整提示詞、參照角色、實際原生工具路徑及專案保存路徑。
- `ENEMY00-01-small-worklist-body-v1-20261001.json`：回填唯一工作清單的本輪現況，不取代 `docs/worklist.json`。
- `copy_enemy00_01_small_candidates.py`、`process_enemy00_01_small_candidates.sh`：保留原生檔、Lanczos縮放全圖、等分拆圖與最近鄰放大對照；拒絕覆寫。
- `ENEMY00/01-smallgroup<組>-reference-v1-20261001.png`、`-generated-v1-20261001.png`、`-atlas48-v1-20261001.png`、`-comparison-v1-20261001.png`：分組來源、候選及展示圖。00第1組沒有新generated，改用 `ENEMY00-smallgroup1-atlas48-reused-v1-20261001.png`。
- `ENEMY00/01-small-<圖號>-48-v1-20261001.png`：27張新小圖；00 #18–#20沿用舊位置，不另存新候選。
- `ENEMY00-small15-comparison-v1-20261001.png`、`ENEMY01-small15-comparison-v1-20261001.png`：576×1920總對照，每組兩列依序原版／候選，單列三格。黑底只供展示，原圖RGBA保留。
- `verify_enemy00_01_small_candidates.py`、`ENEMY00-01-small-verification-v1-20261001.json`：明確71 PNG清單、原版RGB、拆圖與對照獨立核對、負對照、兩個alpha門檻外框；不掃舊批次。
- `preserve_enemy00_01_small_sources.py`、`pbl-enemy00-01-small-v1-20261001.py`、`ENEMY00-01-small-source-manifest-v1-20261001.json`：已驗來源與解碼器bytes保全、精確檔案雜湊／UID及原版輸入雜湊。

Python3.13.15，固定映像 `sha256:540c7d91f98ff6880174c40e99067bf5941eb54d818a7a5e094d188b196a934d`；
ImageMagick7.1.2-12 Q16-HDRI，映像 `sha256:87998ec1b8127b2f73f626f74f7b05e8827f9d7605fa52da5370588f7e53cee1`。
依§35的有界、離線、--rm、CPU1、UID/GID1000、記憶體／程序／日誌限制執行；
原版同時掛 `/orig:ro` 與 `/src/workplace/original:ro`，避免工作樹可寫別名暴露原始輸入。
複製時原生工具輸出目錄唯讀掛 `/generated:ro`。容器工作目錄 `/src`，保存輸出後可核對：

```sh
python workplace/hd/redraw/verify_enemy00_01_small_candidates.py --check
```

生成前準備、複製、處理及保全腳本拒絕覆寫，重建須用乾淨研究工作樹；
舊三張候選與其舊收據為明示輸入，不能再生成後換成相同檔名。
本批起跑load6.28／10.18／12.00，不作音訊或即時幀率結論。
來源manifest明確列85檔、10,012,447 bytes，完整雜湊及UID/GID核對符合；不覆寫前批manifest。
沒有正式程式改動、commit、push、Issue寫入、公開或發行；來源保全及清理結果見唯一WORKLOG。

## 47. 單格方法試驗與ENEMY02小圖塊（2026-10-01）

§46的三格生成仍有位置及形狀偏差。本輪先用同一原版ENEMY00 #22試單格，
再逐張製作ENEMY02 #15–#29，未覆寫舊候選。沿用imagegen技能及內建image_gen，
十六次完整提示詞、參照與實際工具原生路徑均保存。查表命中「CONTEXT、WORKLIST、
WORKLOG與研究紀錄分工」，實際載入 `local/project-document-responsibilities.md`；
沿用既有研究檔與唯一工作歷程，不建立新根目錄或交付結構。

### 47.1 原版來源及證據範圍

| 原版檔案 | SHA-256 | 本批 |
|---|---|---|
| ENEMY00.PBL | `8241b0ea73b1e7402b13adc434f10a5b1288f88f01fcfc3ba5710e4923fe6067` | #22同來源單例，16×16 |
| ENEMY02.PBL | `c38beb2879508466f0c316185eb7a489071279c34c2a89678bb1e80598de4c3f` | 30張中#15–#29均16×16、無2／8／A色號 |

圖號、PBL檔案偏移、解碼色號、尺寸及非零外框為confirmed；工具為已保存bytes的
`tools/pbl.py`，Python3.13.15。未證明玩法身分、實際動作順序、所有原版合成模式，
更未以透明候選當作原版alpha語意。ENEMY03另核對來源SHA-256
`ad6e8183bbf6c6fc258693b1f0ac726593feff5d052b5da18ac715cc2e63e5c1`、30張及後15張16×16，
尚未產生小圖塊候選，不能把來源盤點當作素材完成。

十六個單格參照各384×384，完整原版16×16最近鄰×24；ENEMY02另有五組1152×384
展示參照，各三格按圖號橫接，只供閱圖及對照，生成時每次只附自己的原版單格。
完整RGB獨立重算不符0，輸入雜湊、偏移、原版外框及解碼雜湊均存入收據。

### 47.2 同來源單格只有有限改善

ENEMY00 #22參照與§46來源相同；新提示詞強調單格、原版裁切、保留空間，不畫成
孤立裝飾圖形。保留三格舊候選與新單格，完整1254×1254原生圖縮到48×48，不補畫、
不平移或裁去邊界。以alpha≥128外框量測，右／下界不含該座標：

| 原版×3 | 三格舊候選 | 單格新候選 |
|---|---|---|
| `(0,0,48,39)` | `(1,6,47,38)` | `(0,0,48,37)` |

四座標絕對誤差總和**9→2**，頂端偏差**6→0列**、左右界相同；底界仍少2列。
alpha≥16的新外框為`(0,0,48,38)`，低透明邊緣也未達原版底界。
原版／舊三格／新單格1152×384對照由原版色號及兩張RGBA獨立推導，RGB不符0。

此例共同改變提示詞、分組方式、原生解析度及重取樣路徑，**不能隔離成單格方法的因果證明**，
也不證明全體輪廓、美術或正式路徑通過。圖片與有限量測完整保留，不追改同一圖直到碰巧過關。

### 47.3 ENEMY02逐張候選與未通過項

十五次內建image_gen生成各1254×1254原生圖，保留整個畫布縮到48×48 RGBA；
只用放大後原版形塊描述，不把抽象圖案猜成角色、武器或機械。十五張透明像素528–1755／2304，
可見且透明通道存在；這不證明透明邊緣或混色正確。完整原生圖、完整提示詞與原圖均保存本機。

新PNG共**65張**：單格試驗4張，加ENEMY02的20參照、15原生圖、15小圖、
5展示圖集、5分組對照、1總對照。含三格舊候選一張共**66張**，全部CRC、完整解碼、
尺寸、通道及UID/GID1000符合；不使用廣泛glob混入舊批次。獨立原版參照、黑底合成、
展示圖集追加次序及總對照RGB不符0。參照水平位移一像素負對照差**75,576**像素，
完全不透明候選被拒絕；§46舊85項來源雜湊全部未變。

以alpha≥128外框比對原版非零外框×3：**6/15相同**，最大單一邊界誤差3個HD像素。
外框相同不代表內部形狀及細部配色已通過：

| 抽樣 | 原版外框×3 | 候選外框 | 限制 |
|---|---|---|---|
| #17 | `(12,6,36,30)` | `(12,6,37,31)` | 原版窄形仍有右／下界各多1列 |
| #18 | `(0,0,48,42)` | `(0,1,48,44)` | 頂端多1列、底端多2列 |
| #24 | `(0,0,48,48)` | `(1,1,48,47)` | 完整範圍仍有額外留白 |
| #28 | `(0,6,42,36)` | `(0,6,45,37)` | 右界多3列，比例尚未保持 |
| #29 | `(12,6,30,36)` | `(12,6,31,37)` | 窄條右／下界各多1列 |

實際查看總對照：逐張生成避免三格間直接連接，但圓角化、局部高度、細部配色／
風格一致性仍待修正；斜向交錯組接近圓角像素塊，不能當作已完成HD線條處理。
白色區域透明孔洞尚見邊緣雜點，PNG通道通過不足以宣稱美術通過。
正常玩家輸出、中文覆繪、8×8遮格、動作時序及讀檔未驗；本批未正式接入。

小圖塊候選涵蓋**30→45/180**，其餘135個維持完整範圍；與#0–#14的180圖號預覽分開計，
正式美術仍**0/360**。8×8、美女在後框在前、原版排版與全部sprite要求維持；
混色未回答，024 §1.5仍DRAFT，不重開已閉合首場來源或把素材轉換當正式功能完成。

### 47.4 完整入口、來源保全及重跑

全部在既有 `workplace/hd/redraw/`：

- `prepare_small_single_trial.py`、`ENEMY00-small-22-single-reference-v1-20261001.json`、`ENEMY00-small-22-single-reference-v1-20261001.png`：同來源原版單格及舊候選位置證據。
- `ENEMY00-small-22-single-generation-job-v1-20261001.json`、`ENEMY00-small-22-single-selected-v1-20261001.json`：完整提示詞、實際原生工具及專案保存路徑。
- `copy_small_single_trial.py`、`process_small_single_trial.sh`、`ENEMY00-small-22-single-generated-v1-20261001.png`、`ENEMY00-small-22-single-48-v1-20261001.png`、`ENEMY00-small-22-single-comparison-v1-20261001.png`：單格原圖、完整畫布縮放及原版／舊三格／新單格對照。舊圖仍在§46原位置。
- `prepare_enemy02_small_single_references.py`、`ENEMY02-small-single-references-v1-20261001.json`：十五個原版來源、偏移、外框及圖集參照。
- `ENEMY02-small-<15–29>-single-reference-v1-20261001.png`、`ENEMY02-smallgroup<0–4>-single-reference-v1-20261001.png`：十五單格及五組展示參照，不將後者附進生成。
- `ENEMY02-small-single-generation-jobs-v1-20261001.json`、`ENEMY02-small-single-selected-v1-20261001.json`：十五次完整提示詞與實際工具／保存路徑。
- `copy_enemy02_small_single_candidates.py`、`process_enemy02_small_single_candidates.sh`：非破壞性原圖保存、Lanczos完整畫布縮放及展示圖集。
- `ENEMY02-small-<15–29>-single-generated-v1-20261001.png`、`ENEMY02-small-<15–29>-single-48-v1-20261001.png`：十五原生圖及48×48候選。
- `ENEMY02-smallgroup<0–4>-single-atlas48-v1-20261001.png`、`ENEMY02-smallgroup<0–4>-single-comparison-v1-20261001.png`、`ENEMY02-small15-single-comparison-v1-20261001.png`：144×48展示圖集、576×384分組對照及576×1920總對照。每組兩列原版／候選，黑底僅展示，RGBA原檔仍保留。
- `verify_enemy02_small_single_candidates.py`、`ENEMY02-small-single-verification-v1-20261001.json`：66張明確清單、原版與對照獨立核對、兩個alpha門檻外框、負對照及舊85項不變。
- `ENEMY02-small-single-worklist-body-v1-20261001.json`：回填唯一工作清單的現況，不取代 `docs/worklist.json`。
- `preserve_enemy02_small_single_sources.py`、`pbl-enemy02-small-single-v1-20261001.py`、`ENEMY02-small-single-source-manifest-v1-20261001.json`：85項、**9,422,762 bytes**精確來源／UID／雜湊與解碼器bytes保全；原版只記雜湊，不複製。§46舊manifest及驗證收據作明示輸入保留。

Python3.13.15與ImageMagick7.1.2-12 Q16-HDRI映像識別沿用§46；前者沒有Entrypoint欄，
完整Config確認預設Cmd為python3；後者用 `--entrypoint sh` 與 `magick`，沒有新建工具鏈。
執行依§35離線、CPU1、--rm、UID/GID1000、記憶體／程序／日誌限制及有界逾時。
原版同時掛 `/orig:ro` 與 `/src/workplace/original:ro`，工具原生輸出另以 `/generated:ro` 掛載。
生成服務未提供固定seed／版本，重現限於保存原生PNG以後的轉換及核對。容器內 `/src` 重跑：

```sh
python workplace/hd/redraw/verify_enemy02_small_single_candidates.py --check
```

準備、複製、處理與保全腳本拒絕覆寫；完整重建用乾淨研究工作樹，舊85項明確輸入保持原樣。
起跑load13.92／9.54／9.56，沒有音訊／即時幀率結論；本批只是候選素材及方法證據，
沒有正式程式改動、commit、push、Issue寫入、公開或發行。容器與檔案清理結果見唯一WORKLOG。

## 48. Kasuruji／Minton 正常身體來源與限定接入

2026-10-01路由命中資產考古／驗收及規格實作，載入逆向技能、
`local/retro-remake-spec-gated-workflow.md`；文件職責沿用已載入入口。
沿用原版正常重播，不注入座標、道具或勝利。新增來源與程式入口集中本節。

### 48.1 原版輸入與正常路線（已證實，限兩段）

- `PW.EXE` SHA-256 `88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49`。
- `ENEMY00.PBL` SHA-256 `8241b0ea73b1e7402b13adc434f10a5b1288f88f01fcfc3ba5710e4923fe6067`，30張。
- `10-saved.state` SHA-256 `abdbb98f66010d5680d3f3bfecead2c28ffd4f7f6fd625fc3b3103a62bcbd056`，執行前seed A71Dh。
- `13-healed.state` SHA-256 `d9c18d64a60ed678d34d72f00c199af51108c2a36931fdcaed650a110f23e7a0`，執行前seed A48Ch。

工具dosgolem `f8c1a6e`、Go1.24.13；位址空間為執行期CS:IP／DS:BX，
貼圖入口 `0161:8705`、返回 `0161:8751`，不是IDA線性位址。
兩段完全沿用 `replay/title-to-first-save.json` 的FIFO方向鍵、每掃描碼3,000,000步，
按住空白鍵238,000,000／382,200,000起各30,000,000步，typematic=true。
終點280,000,000／420,000,000步；完整鍵序列、來源雜湊及兩側seed寫入收據。

研究工具在容器 `/tmp/pwbody` 建立臨時子模組 `github.com/wicanr2/dosgolem/bodyprobe`，
require本專案與dosgolem並replace到 `/src` 與 `/src/worktrees/dosgolem`；
複製主專案go.sum，使用 `go build -mod=readonly`。這只使一次性工具可沿用probe
內部鍵盤佇列，不更改dosgolem或增加正式API。建置來源 `tools/hd/observe_next_bodies.go`。

### 48.2 身體來源與重疊限制

| 正常路線 | 已證實來源循環 | 核對樣本／總身體呼叫 | 完整畫面直接匹配 |
|---|---|---|---|
| Kasuruji，12-battle2 | #0→#1→#2→#1→#0 | 16／26 | 1／16 |
| Minton，14-minton1 | #6→#7→#8→#7→#6 | 16／18 | 3／16 |

位置(32,152)、24×32，AL00完整／AL01差分；32次原始384 bytes、前後色號畫面、
返回state全部保存。Go來源推導不符0、矩形外變動0；獨立Python對12檔177個24×32
來源解碼，由首個完整圖及每筆唯一打包差分追蹤身體，四方向均有實際呼叫證據。
重疊後額外XOR成分在身體轉換前後相同；其用途維持未知，不宣稱已辨識全部特效。
錯位置負對照兩段分別6,875／6,140像素；30次差分的錯模式負對照均有效。
首個完整圖畫在全黑區時，完整與XOR結果相同，因此兩筆錯模式負對照為0，
不冒稱該負對照有效；錯姿勢及位移負對照仍有效。

有／無觀察兩側終點暫存器、flags、步數、cycles、IRQ1、快照RAM及原版色號畫面相同；
兩段終點畫面亦與既有正常 `.frame` 完全相同。
此處RAM比較是兩次同版原版執行，不宣稱所有既有state欄位均相同。
Kasuruji終點cycles1,343,022,797、Minton2,035,693,244；不量測即時音訊或幀率。

### 48.3 訂正及入口

最初攻擊前前綴只得1／3筆，未達16筆門檻；v1原圖、來源及工具bytes保留，沒有通過收據。
改用已存在的完整正常攻擊段，觀察上限16、實際數明列，不用新增等待或改亂數湊數。
最初獨立核對要求每筆返回都等於完整姿勢，因效果重疊而失敗；
`next-bodies-verifier-fullpose-v1-20261001.py` 保留舊假設。
新驗證改核對原始來源差分與額外成分不變，仍不把重疊圖當完整圖。
規格024 §1.6已依來源審查達READY，正式擴充維持完整前姿勢門檻及未知回退。

入口：

- `tools/hd/observe_next_bodies.go`；本機 `workplace/hd/observe-next-bodies-v1-20261001`／`v2` 二進位。
- `workplace/hd/next-bodies-observer-prefix-v1-20261001.go`、`next-body-{kasuruji,minton}-v1-20261001-event*`：未達門檻的舊前綴，不當作驗收。
- `workplace/hd/next-body-{kasuruji,minton}-v2-20261001.json`、同前綴 `-event00`至`-event15` 的 `-before.frame`／`-after.frame`／`-source.bin`／`.state`、`-end.frame`：原版來源與兩側終點。
- `tools/hd/verify_next_bodies.py`、`workplace/hd/next-bodies-verification-v2-20261001.json`：獨立解碼、來源追蹤、負對照及既有原版終點。
- `workplace/hd/next-bodies-before-{enemy-go,theme-go,enemy_test-go}-20261001`：正式擴充前程式bytes。
- `tools/hd/prepare_next_body_theme.py`、`workplace/hd/theme-next-bodies-v1-20261001/`：舊十筆加六姿勢的本機候選主題；不覆寫舊主題，`asset-map.json`保存實際來源雜湊。
- `tools/hd/verify_next_body_runtime.go`／`tools/hd/verify_next_body_plane.py`：新增接入的保存狀態與獨立8×8圖面驗證入口。
- `workplace/hd/next-bodies-source-manifest-v1-20261001.json`：本輪來源及產物的精確保全入口。

原版及state僅限本機；美術未驗、效果混色仍DRAFT。正式接入及收尾結果續記於本節，
不能以來源已證實宣稱全部sprite完成。

### 48.4 限定正式接入及實際驗證

先保存舊enemy.go／theme.go／enemy_test.go，再依024 §1.6 READY擴充載入器接受#0–#8，
每組三姿勢只接受已證實相鄰往返；#9–#29、其他檔、跨角色前姿勢維持拒絕。
不修改原版EXE、規則、記憶體或存檔，也不把重疊分解接到正式路徑。
本機主題十→十六筆，六張直接複製既有group0／group2 PNG，`asset-map.json`明列來源。
舊十筆主題及首場動作資料保留；正式美術仍0/360。

`verify_next_body_runtime.go`以兩個Oracle逐一實際載回32個原版返回state，執行前同一state
固定seed、HD開關／重登記圖面相同；各接續100,000步、無新增鍵，有／無HD兩側暫存器、
步數、cycles、原版畫面及完整1 MiB匯流排讀值相同。
全記憶體只在兩側終止後取樣，不在貼圖中讀VGA或接續該取樣後狀態。
32張輸出PNG由 `verify_next_body_plane.py` 依原版PBL、候選PNG及(0,0)起8×8格獨立推導；
包括背景、MENU、ALLY及同位置敵人互斥，圖面不符全部0。
四張完整姿勢為Kasuruji#0、Minton#6／#7／#8；其省略負對照有效。
其餘28張重疊來源在重新載回後沒有完整姿勢證據，依契約回退原版，不能當作HD身體覆蓋完成。

真實素材主題測試9主測試／18含子案例通過，失敗／略過0；包含九圖號差分、跨角色拒絕、
未支援圖號、ALLY及既有格線／恢復回歸。首跑缺 `/hd/bg.idx` 唯讀掛載，背景測試失敗；
補掛存在的 `workplace/hd/` 到 `/hd:ro` 後，同一映像／同一測試命令乾淨重跑通過。
首次失敗JSONL保留，分類為驗證環境問題，沒有為它修改產品。

新增收據／入口：

- `workplace/hd/next-body-theme-tests-v1-20261001.jsonl`／`v2`：缺掛載失敗及乾淨通過。
- `workplace/hd/next-body-runtime-v1-20261001.json`、同前綴`-{kasuruji,minton}-{00–15}.png`：32次讀回、開關、原版接續及實際圖面。
- `workplace/hd/next-body-plane-verification-v1-20261001.json`：獨立圖面、四個省略負對照及28份回退。
- `tools/hd/preserve_next_bodies.py`：精確來源／產物清單及程式bytes保存，拒絕覆寫。

重跑須使用乾淨研究工作樹，不覆寫已有前綴。原版掛 `/orig:ro` 與
`/src/workplace/original:ro`；Go快取沿用既有兩目錄到 `/gocache`／`/gomodcache`。
Go映像 `golang:1.24-bookworm` SHA-256
`1a6d4452c65dea36aac2e2d606b01b4a029ec90cc1ae53890540ce6173ea77ac`、Go1.24.13；
Python沿用 `python:3.13-alpine`。全部--rm、network none、UID/GID1000、CPU1、
記憶體512MiB／Go1GiB、pids64／128、日誌10MiB×3、有界30–180秒逾時。
容器 `/src` 中的工作順序為：

```sh
# 原版探針需依48.1臨時子模組建置，輸出兩段全新前綴：
workplace/hd/observe-next-bodies-v2-20261001 -segment 12-battle2 -out workplace/hd/next-body-kasuruji-v2-20261001
workplace/hd/observe-next-bodies-v2-20261001 -segment 14-minton1 -out workplace/hd/next-body-minton-v2-20261001
python tools/hd/verify_next_bodies.py --out workplace/hd/next-bodies-verification-v2-20261001.json
python tools/hd/prepare_next_body_theme.py
go test -mod=readonly -json ./apps/psychicwar/theme
go run -mod=readonly tools/hd/verify_next_body_runtime.go -out workplace/hd/next-body-runtime-v1-20261001
python tools/hd/verify_next_body_plane.py --out workplace/hd/next-body-plane-verification-v1-20261001.json
```

已有輸出時只加`--check`重生兩份獨立Python收據，不執行拒絕覆寫的準備／原版擷取。
單元測試需 `PSYCHICWAR_TEST_ORIG=/orig/psychic-war` 及上述 `/hd:ro` 掛載。
新增角色完整HD正常玩家路徑、中途姿勢／清除及中文覆繪抽測尚未完成；§1.6維持READY，
不宣稱CONFORMED。候選造型及比例尚待美術驗收、效果混色仍DRAFT；沒有公開或發行。

## 49. 新增角色的正常遭遇／等待／F3路徑驗證

本輪以§48現行程式為準，補正常HD路徑、中途輸出及動作清除證據，不重開已閉合的特效來源。
路由命中資產驗收與規格實作，沿用逆向技能、規格閘門及文件職責入口。
既有正常方向鍵路線走到Kasuruji／Minton，等候4,000,000步後按住F3 500,000步，
再等候至F3起點後3,000,000步；不按空白鍵、不寫原版RAM或直接呼叫繪圖常式。
這是玩家等候再脫離的另一條正常輸入，不宣稱等於§48既有攻擊重播終點。

研究工具先以原FIFO執行器記錄實際IRQ1送出步數，再以Oracle的KeyDown／KeyUp重播相同
掃描碼；必須核對原版畫面、全部暫存器、steps、cycles、IRQ1及A0000h以下RAM，
不能因按鍵名稱相同便假定等價。Oracle.Run依指令數推進，不增加按住鍵的自動重複。
取樣原版貼圖返回及四種方向的首指令／512步中途；正式圖層與中文層沿用現行契約。
全1MiB匯流排只在終止後讀取、不接續該取樣後狀態。

新增入口：

- `tools/hd/verify_next_body_normal.go`：正常FIFO來源、實際按鍵送出收據、HD開關與中文路徑。
- `tools/hd/verify_next_body_normal.py`：由原版及候選獨立核對圖面、中途遮格、中文與清除。
- `tools/hd/preserve_next_body_normal.py`：精確保全來源、舊驗證器及本輪產物。
- `workplace/hd/next-body-normal-{kasuruji,minton}-v1-20261001*`、Kasuruji的`v2`及兩側`v3`：歷次來源及輸出前綴，不覆寫。
- `workplace/hd/next-body-normal-verification-v3-20261001.json`：獨立核對結果；v1／v2驗證失敗未產收據，舊工具另存。
- `workplace/hd/next-body-normal-source-manifest-v1-20261001.json`：來源與產物保全入口。

### 49.1 原版與正式接入的實跑

Minton有16筆身體貼圖，完整姿勢依#6→#7→#8→#7→#6循環；四種方向各取
入口後1步與512步，共8個中途樣本。F3掃描碼3Dh於386,200,000步送出，
500,000步後釋放，389,200,000步終止；原版戰鬥返回0161:47AD恰好一次。
Kasuruji有9筆身體貼圖，8個中途樣本；F3於242,000,000步送出、相同按住時長，
245,000,000步終止，原版返回次數0。實際正常IRQ1皆送完，36／16個make與break
掃描碼分別由原FIFO取得，再交Oracle重播。兩側HD開關與FIFO的原版終點暫存器、
steps、cycles、IRQ1、畫面及A0000h以下RAM一致；終止後全匯流排HD兩側亦相同。
此處是固定seed的玩家可見抽樣結果，不推定F3逃走規則，也沒有延長等待或換時機重擲。

最初v1驗證器誤要求每場必定返回一次；Kasuruji的0次使工具中止，該前綴只有原版來源、
沒有通過收據。舊工具保存於`next-body-normal-observer-before-diagnostic-v1-20261001.go`。
v2改為據實記錄0／1次返回，只有原版確實返回才要求HD敵人清除；用相同輸入乾淨重跑。
沒有為測試修改遊戲判定。Kasuruji首張完整#0，其後八張被原版自動特效遮住；
完整前姿勢門檻無法確認後續HD身份，因此回退原版，不能宣稱全部三姿勢HD正常顯示。

### 49.2 中文起點的缺口與正常重生

中文起點的補證入口：`tools/hd/rebuild_normal_translation.go`，及僅於研究編譯使用的
`workplace/hd/normal-chain-oracle-bridge-v1-20261001.go`。
後者以Go編譯覆映射（overlay）提供Oracle對已設好正常FIFO的Machine／DOS包裝；
不修改dosgolem工作樹、不直接呼叫遊戲函式或改原版記憶體，不進正式路徑。
由06-name經07／08／09／10／12／13既有正常重播，逐段核對既有原版state，
將正常印字產生的中文Layer保存於 `workplace/hd/normal-chain-translation-v1-20261001*`。
六段實跑全部與既有state的暫存器、steps、cycles、畫面及A0000h以下RAM相同。
不把CPU／RAM一致冒稱原始state檔bytes相同，也不宣稱06-name之前的全文覆繪。

Minton v1截圖實際檢視時，新敵人名已中文、舊治療訊息仍英文；原因是原版起始state
沒有中文Layer快照，掛上翻譯器不能重建過去已印的字。Kasuruji v2同樣只涵蓋新印字。
舊v2程式保存於`next-body-normal-observer-before-translation-v2-20261001.go`。
v3從上述正常重生的10-saved／13-healed Layer恢復，方向／等待／F3與前版完全相同，
兩條路徑原版及HD開關對照均通過。Minton event02實際圖像可見治療訊息、敵人名及
控制標籤中文，人物與框架沿用原版排版；圖像檢視本身不代替獨立像素驗證。
不能用猜補原文或人工注入疊字修正舊測試截圖。

獨立驗證以原版PBL、候選PNG、正式text及GOLEMFNT推導43幀期望，結果見49.3。
盟友的全部覆蓋貼圖未在本輪保存，不猜其持續身份；排除盟友(264,152,24,32)的
HD身份期望，該區只驗實際合成。既有盟友正式證據仍見§34。
024 §1.6仍READY，美術、全部sprite與效果混色未完成。

獨立驗證首版已完成Kasuruji18幀，Minton因原版返回迷宮清空訊息，而測試誤要求
終點繼續保留治療訊息失敗；保存工具於`next-body-normal-verifier-before-message-gate-v1-20261001.py`。
修正為戰鬥取樣必須保存正確譯文、返回終點必須移除已清空訊息，再以相同資料重跑。
沒有重生或覆寫原版幀，也沒有為測試改產品。

v2續跑完成兩側逐像素圖面、中文及合成，但末尾清除斷言誤要求敵人位置的HD圖面
全透明，忽略原版返回時HD走廊正常補回該區而失敗；工具保存於
`next-body-normal-verifier-before-background-clear-v2-20261001.py`。
重新讀路由與規格閘門後，v3以不含敵人的完整背景期望核對清除，
另故意把最後敵人候選疊回終點，要求殘留負對照確實不同。

### 49.3 獨立核對結果與保全入口

v3以同一映像／同一批原版輸出乾淨重跑通過：25筆原版貼圖逐像素套用實際AL模式，
來源、前後及矩形外不符0；23筆差分的錯模式負對照有效，首兩張黑底完整圖錯模式差0，
不宣稱那兩張有有效負對照。43幀皆核對原版雜湊、HD開關中文一致、正式GOLEMFNT
與Layer快照獨立渲染中文、原版RGB×3→HD→中文合成，不符均0。
依正式文本核對兩個敵人名；Minton戰鬥期間兩行治療訊息保留正確譯文，返回清空後移除。
43幀中文省略負對照全部有效，合計1,889,856個放大像素；不是全文翻譯品質驗收。

8×8圖面依原版完整前姿勢及raw差分決定唯一來源，中途只允許已確認目標姿勢的吻合格，
不依正式Theme內部訊號當答案。Minton16個入口可確認、24個戰鬥樣本有HD敵人格，
返回終點無敵人格，故意疊回最後敵人差5,906像素。Kasuruji只有首入口與首樣本完整HD，
其餘17樣本沿原版回退；兩側身份對照不符0但不能冒稱全部姿勢HD完成。
省略敵人負對照各6,336／138,240個放大像素，合計144,576；盟友持續身份期望排除，
實際盟友區合成仍逐像素核對，不能將此結果擴稱全部盟友正常動作通過。

起跑load8.35／9.95／11.52，CPU1；只做決定性指令及像素比較，不作音訊或即時幀率結論。
沿用§48既有Go及Python映像、快取、--rm、UID/GID1000、network none、記憶體與
pids／日誌限制，外層30–240秒逾時。原版在`/orig`與`/src/workplace/original`皆唯讀。
研究子模組沿用§48.1，正常工具及中文重播需`github.com/wicanr2/dosgolem/bodyprobe`
模組名稱以受控使用原FIFO；中文重播額外Go覆映射僅在編譯時將不存在的
`/src/worktrees/dosgolem/oracle/hd_research.go`指向上述已保存bridge，不修改dosgolem檔案。
實際建置來源與二進位均保存本機，不把bridge加入正式API。

Docker容器`/src`內主要入口如下；Go工具及保全工具拒絕覆寫，原版重跑須在乾淨研究輸入
使用相同前綴。既有獨立收據只以`--check`驗證，不覆寫原版輸出。

```sh
workplace/hd/rebuild-normal-translation-v1-20261001 -out workplace/hd/normal-chain-translation-v1-20261001
workplace/hd/verify-next-body-normal-v3-20261001 -label kasuruji -initial-xlate workplace/hd/normal-chain-translation-v1-20261001-10-saved.layer.json -out workplace/hd/next-body-normal-kasuruji-v3-20261001
workplace/hd/verify-next-body-normal-v3-20261001 -label minton -initial-xlate workplace/hd/normal-chain-translation-v1-20261001-13-healed.layer.json -out workplace/hd/next-body-normal-minton-v3-20261001
python tools/hd/verify_next_body_normal.py --out workplace/hd/next-body-normal-verification-v3-20261001.json
python tools/hd/preserve_next_body_normal.py
```

沒有修改正式產品、原版EXE或RAM，沒有注入道具／角色、強制勝利、commit、push、PR、
Issue寫入、公開或發行。024 §1.6仍READY、§1.5混色仍DRAFT，完整HD及中文化目標維持。

## 50. Sivad正常新遭遇的身體來源與限定接入

2026-10-02接續§49現行主題；先取正常原版證據，再審查024新範圍。
路由命中資產驗收與規格實作，載入逆向技能、證據reference及規格閘門。
既有`17-saved2.state`由正常流程抵達Sivad並存檔，研究011 §3.5指出出平台後很快遭遇。
本輪從該固定原版state以正常方向鍵前進，不注入座標／道具、seed或勝負，不重開碰撞考古。
目前已有原版觀察收據；首完整姿勢辨識為 ENEMY01 #6，差分方向的獨立核對及新接入仍待完成。
不能依區域編號猜圖號，也不能把貼圖觀察直接當作 HD 驗收。

本輪工具與產物入口：

- `tools/hd/observe_sivad_body.go`：有界正常方向鍵、原版貼圖入口／返回與觀察對照。
- `tools/hd/verify_sivad_body.py`：已建立的原版PBL、實際raw來源與前後畫面獨立核對工具，結果見§50.2。
- `workplace/hd/sivad-body-source-v1-20261002*`：固定來源、樣本、失敗或成功前綴均不覆寫。
- `workplace/hd/sivad-body-verification-v1-20261002.json`：實際結果收據，未產生前不宣稱通過。

正式實作需在來源與規格達READY後進行；§1.5特效混色仍DRAFT，整體美術與其他sprite範圍維持。

### 50.1 首次原版觀察的內容與界限

原始收據 `sivad-body-source-v1-20261002.json` 的 SHA-256 為
`79987c386258696348632f5638448ac21045c7ca3b7a154c423c1d5a98a020c3`。
工具 Go 1.24.13、dosgolem 基準 f8c1a6e；輸入 `17-saved2.state` 雜湊
`3c2ae7470600c61217060dc76843aed7c4097ac6e4d80a778cf3282b8afbc7e4`，
ENEMY01.PBL 雜湊 `22f664050ce7ffa4ea0f6941c9c91cd1ab43671ea5b53491f6799f78ba8e64af`。
位址基準為原版執行期 CS:IP／DS:BX，畫面為 320×200 色號座標；
兩側執行前唯讀核對 CS:41DF 種子 F95Bh，不改寫或重擲。

從 655,000,000 步的正常存檔開始，原 FIFO 送兩次向上鍵，起點 655,500,000、
鍵事件間隔 3,000,000，終點 695,000,000；實際 IRQ1 共四次按下／放開。
身體入口 CS:8705 至返回 CS:8751，矩形 (32,152,24,32)，取得 8 次完整呼叫與
8 個中途樣本。首筆 AL=00h，觀察器辨識完整圖 ENEMY01 #6；其餘七筆 AL=01h，
前後完整姿勢因重疊均未辨識。這份觀察收據本身不能證明其他姿勢及差分方向；
後續獨立 PBL／raw 核對見§50.2。

原版戰鬥進入／返回各一次，終點區域1、(5,4)、HP 0、能量8；不宣稱為 F3 逃走或勝利。
觀察與無觀察對照的全部 CPU 暫存器、步數、cycles、IRQ1、原版色號畫面及 A0000h 以下 RAM
收據相同。這證明本次觀察終點一致，不證明新角色 HD、中文或美術驗收通過。
首次觀察未更動正式主題載入器；後續來源審查及限定接入的現況見§50.2–50.3，
不將本小節的觀察收據單獨當作HD驗收。

### 50.2 獨立來源核對與接入入口

獨立Python PBL解碼核對八次來源為 ENEMY01 #6→#7→#8→#7→#6→#7→#8→#7；
四種相鄰有向差分均已證實。每次原版AL模式套用、完整圖辨識及矩形外不符0，
七次差分額外XOR成分在身體貼圖前後保持相同，用途未知。錯姿勢／位移負對照有效，
七次錯模式負對照有效；首筆黑底完整圖錯模式差0不宣稱有效。八個中途樣本來源已保全，
原版來源核對本身不證明HD中途輸出；正式限定驗證見§50.3。
收據 `sivad-body-verification-v1-20261002.json` 已產生。
證據審查後新增024 §1.7 READY；接入與驗證入口如下：

- `workplace/hd/source-before-sivad-v1-20261002/manifest.json`：舊正式程式及文件bytes。
- `tools/hd/prepare_sivad_theme.py`、`workplace/hd/theme-sivad-v1-20261002/`：舊十六筆＋三張既有候選；本機研究主題。
- `tools/hd/verify_sivad_runtime.go`、`workplace/hd/sivad-runtime-v1-20261002*`：完整／中途state載回及同狀態原版接續。
- `tools/hd/verify_sivad_plane.py`、`workplace/hd/sivad-plane-verification-v1-20261002.json`：獨立原版Reference與候選PNG推導8×8圖面。
- `tools/hd/rebuild_sivad_translation.go`、`workplace/hd/sivad-translation-v1-20261002*`：由已驗13-healed中文Layer接續正常原版段落到Sivad。
- `tools/hd/verify_sivad_normal.go`／`.py`、`workplace/hd/sivad-normal-v1-20261002*`：正常方向鍵的正式HD／中文抽測；未產成功收據前不宣稱通過。
- `apps/psychicwar/theme/sivad_test.go`：真實三姿勢、跨檔前姿勢、清單與錯雜湊回歸。
- `tools/hd/preserve_sivad.py`、`workplace/hd/sivad-source-manifest-v1-20261002.json`及`source-sivad-v1-20261002/`：本批精確來源／產物快照與舊來源完整性核對。
- `workplace/hd/preserve-sivad-before-path-normalization-v1-20261002.py`：首版保全工具，拒絕容器`/src`來源別名後的失敗來源bytes。
- `workplace/hd/preserve-sivad-before-source-set-normalization-v2-20261002.py`：次版保全工具，工具自身絕對路徑仍未正規化的失敗來源bytes。

以上工具／產物建立與執行結果見下節；正式美術0/360、其他sprite及§1.5 DRAFT不變。

### 50.3 正式限定接入、正常中文鏈與同狀態結果

024 §1.7 READY之後才保存舊正式程式及七份來源／文件bytes，擴充載入器接受
ENEMY01 #6–#8及已證實四條相鄰有向差分；ENEMY00 #0–#8保持。
按檔名選擇原版雜湊、圖號集合及前姿勢，拒絕未支援檔／圖號、錯雜湊、平移／裁切、
錯背景錨點及跨檔前姿勢。三張group2既有候選複製到新主題，舊十六筆不覆寫。
現行 `theme-sivad-v1-20261002/` 共19筆；候選美術及公開權利未驗，正式0/360。

真實PBL主題測試10個主測試／19含子案例，失敗／略過0，包含跨檔及遮擋前姿勢
負對照、舊ENEMY00／ALLY與格線回歸。Go 1.24.13、`PSYCHICWAR_TEST_ORIG=/orig/psychic-war`，
`/hd`掛既有工作素材唯讀；完整JSONL為 `sivad-theme-test-v1-20261002.jsonl`。
17個原版state（8完整／8中途／1終點）逐一載回，HD開關／重登記結果相同，
各接續100,000步，兩側CPU、steps、cycles、原版畫面、A0000h以下RAM與完整匯流排相同。
完整匯流排只在終止後讀取，不接續該state，避免VGA鎖存器取樣干擾。
獨立PBL／PNG推導8×8圖面不符0：首完整姿勢HD，15戰鬥樣本重疊回退，
終點沒有敵人。驗證器摘要把16份無完整敵人的樣本統稱回退；細分以收據kind為準，
不將原版結束畫面算成戰鬥重疊。收據 `sivad-runtime-v1-20261002.json`／
`sivad-plane-verification-v1-20261002.json`，不是正常玩家HD流程的替代驗收。

中文起點由§49已驗13-healed Layer接續14-minton1、15-minton2、16-sivad、17-saved2；
每段保持原FIFO方向鍵／按住，CPU、steps、cycles、原版色號畫面及A0000h以下RAM
與既有原版state相同，不猜補文字。新工具 `rebuild_sivad_translation.go` 使用既有
`normal-chain-oracle-bridge-v1-20261001.go` 的Go編譯覆映射，未寫入dosgolem正式API。
四段收據與Layer為 `sivad-translation-v1-20261002*`，舊中文鏈保留。

正常Sivad兩次前進以原FIFO實際IRQ1送鍵時点對Oracle重播，未按F3／Space、
未改原版種子、狀態或勝負。新FIFO終點的steps／cycles／IRQ1／RAM／原版畫面
先核對先前原始Sivad收據，HD開關兩側再核對FIFO全部CPU及終點，完整匯流排相同。
8次身體貼圖、8中途及終點共17幀，獨立核對AL模式、矩形外、背景／敵人8×8、
正式敵人文本 `I_ENMY01.BIN:00A2`「歐格斯」、GOLEMFNT中文及RGB→HD→中文合成，不符0。
盟友(264,152,24,32)持續身份期望排除，該區只核對實際合成；不外推所有盟友動作。
首姿勢有HD敵人格，其餘15戰鬥樣本重疊回退；原版HP0進入結束畫面，
敵人清除，故意保留最後邏輯姿勢#7的負對照差6,912放大像素。
敵人省略負對照6,912，中文省略負對照合計762,624；七次差分錯模式有效，
首黑底完整圖差0不冒稱有效。實際檢視首遭遇與終點中文圖，結束人物仍原版素材、
尚未HD，維持其他圖像完整範圍。正常收據 `sivad-normal-v1-20261002.json`／
`sivad-normal-verification-v1-20261002.json`；§1.7維持READY，未驗GUI／全動作／美術／封包。

Go編譯沿用§48研究子模組及§49的bridge覆映射；容器`/src`中的主要入口：

```sh
python tools/hd/verify_sivad_body.py --out workplace/hd/sivad-body-verification-v1-20261002.json --check
workplace/hd/verify-sivad-runtime-v1-20261002 -out workplace/hd/sivad-runtime-v1-20261002
python tools/hd/verify_sivad_plane.py --out workplace/hd/sivad-plane-verification-v1-20261002.json --check
workplace/hd/rebuild-sivad-translation-v1-20261002 -out workplace/hd/sivad-translation-v1-20261002
workplace/hd/verify-sivad-normal-v1-20261002 -initial-xlate workplace/hd/sivad-translation-v1-20261002-17-saved2.layer.json -out workplace/hd/sivad-normal-v1-20261002
python tools/hd/verify_sivad_normal.py --out workplace/hd/sivad-normal-verification-v1-20261002.json --check
python tools/hd/preserve_sivad.py
```

Go／複製／保全入口拒絕覆寫；上述Go重生須在乾淨研究輸入使用相同前綴。
舊收據只以--check回查，不覆寫原版幀或state。來源快照在
`sivad-source-manifest-v1-20261002.json`／`source-sivad-v1-20261002/`，保存當時文件bytes。
沿用既有Go／Python映像、UID/GID1000、離線、CPU1、--rm、記憶體／pids／日誌限制
及30–300秒外層逾時；原版兩個掛載別名均唯讀。起跑load15.48，只驗指令／像素，
不作即時幀率或音訊結論。首次保全腳本漏接Docker stdin，未產生檔案；補-i後
同一腳本乾淨重跑，屬控制面設定問題。讀舊manifest摘要誤以清單切片字典，
修正讀取方式後重跑；未因此修改產品或舊來源。

## 51. 原版陣亡結束畫面人物的HD接入

2026-10-02接續§50正常Sivad結束畫面；本批仍不改原版勝負或規則。
原版OVER.PBL有一張64×64圖，SHA-256
`57673c27c3c0b141182a8924ac2f2de8490f43b1958369926861f0eaf02b1d1c`。
獨立Python解碼在§50既有正常終點色號畫面只找到(128,48)一個完整匹配。
這是已證實的來源／位置，完整場景契約與正常HD接入結果見§51.1–51.2。
既有 `workplace/hd/art-in/OVER-00.png` 為原版像素放大，檢視後保留；
本批另以內建image_gen重繪，不覆寫舊圖，生成成功不代表美術驗收。

本批工具與產物入口（產生結果後據實回填）：

- `tools/hd/observe_over.go`、`workplace/hd/over-source-v1-20261002*`：正常Sivad原版貼圖入口、返回、中途及控制組。
- `tools/hd/verify_over_source.py`、`workplace/hd/over-source-verification-v1-20261002.json`：獨立PBL／raw來源、座標與前後圖核對。
- `tools/hd/prepare_over_theme.py`、`workplace/hd/theme-over-v1-20261002/`：沿用十九筆主題及一張本批新OVER候選。
- `tools/hd/verify_over_runtime.go`／`.py`、`workplace/hd/over-runtime-v1-20261002*`：正常出現／清除、讀檔重建、開關、中文及8×8期望。
- `workplace/hd/source-before-over-v1-20261002/`：新增正式接入前保存既有來源bytes。
- `tools/hd/preserve_over.py`、`workplace/hd/over-source-manifest-v1-20261002.json`：本批精確來源保全；舊§50來源保留。

正式實作在審查024 §1.8達READY後進行；本批只接入已證實原版場景，
不把結束畫面人物當成其他ALLY／END1圖號猜映射。完整sprite、美術及公開權利範圍不變。

第一版固定CX=200Ch／DX=0808h的8705探針沒有捕捉呼叫（0/0），兩側仍與§50原版終點一致。
不將零筆當通過，也不因此推定原版未畫OVER；獨立完整畫面匹配仍成立。
舊工具與0筆收據保留；後續補觀察入口清單與逐次場景出現，不以未知繪製模式猜補。

追加來源／圖像入口：

- `tools/hd/observe_over_scene.go`、`workplace/hd/over-scene-source-v2-20261002*`：所有一般貼圖位置摘要及原版完整OVER出現／變更。
- `workplace/hd/redraw/OVER-00-generation-job-v1-20261002.json`：本輪完整提示詞、輸入及工具模式。
- `workplace/hd/redraw/OVER-00-hd-native-v1-20261002.png`／`OVER-00-hd-v1-20261002.png`：內建imagegen原生及192×192候選，不覆寫舊點陣圖。

### 51.1 場景來源審查與READY範圍

獨立全部原版PBL的64×64圖核對，兩個完整OVER樣本均只在(128,48)匹配OVER #0。
首次完整取樣670400000步、前一取樣670300000尚未完整；695000000畫面與§50原版終點相同。
正常Enter在695500000按下、696000000放開，697900000取樣OVER已消失；
原版觀察／對照至705000000的全部CPU、steps、cycles、IRQ1、色號畫面及A0000h以下RAM相同。
461次8705摘要沒有64×64完整OVER呼叫，繪製路徑仍未知；不深挖不影響內容比對的硬體路徑。
正式採已證實完整4096色號的內容比對，未知／中途／錯圖整組回退；
證據審查新增024 §1.8 READY後才接入。全部sprite、美術與正式交付仍未完成。

追加正式測試入口 `apps/psychicwar/theme/over_test.go`；
主程式入口維持 `apps/psychicwar/theme/`，OVER靜態完整來源不猜8705模式。

本批接續驗證入口另列 `tools/hd/verify_over_reload.go` 與
`workplace/hd/over-reload-v1-20261002*`：三個原版場景state與終點實際載回、
HD切換／重登記及有界100,000步接續；與正常玩家路線分開記錄。
`verify_over_runtime.go` 負責正常兩次Up／陣亡／Enter清除的HD及中文，
`verify_over_runtime.py` 由原版PBL、PNG及GOLEMFNT獨立核對，不以正式圖面當期望。

獨立驗證器首版將人物矩形內全部圖面的alpha視為OVER；正常Enter後恢復的
背景圖面也在此矩形，形成假失敗。此前完整8×8期望已相符，原版狀態未受影響。
失敗版本保存於 `workplace/hd/verify-over-before-background-clear-check-v1-20261002.py`。
修正以有／無OVER的獨立圖面比較人物貢獻，另故意保留已出現人物作清除負對照；
不要求正常背景透明，也不因此修改正式程式或原版輸入。

中文內容補證入口 `tools/hd/verify_over_text.py`／
`workplace/hd/over-text-verification-v1-20261002.json`：兩個完整陣亡場景的三行訊息
直接與正式 `text/PW.EXE.json` 比較；字型與合成另依正常路線獨立收據核對。

### 51.2 正式限定接入與同狀態驗證

- 024 §1.8 READY後沿用靜態圖面內容比對。來源SHA／圖數／64×64尺寸、圖號#0、
  原版位置(128,48)及完整src／match均驗證；完整4096色號相符才啟用全部8列。
  任一像素失配整組退出；全黑8×8格保持透明。沒有猜測8705呼叫或新增原版狀態。
- 新主題 `workplace/hd/theme-over-v1-20261002/` 共20筆。前十九張及清單來源保存，
  新人物PNG192×192、原生圖1254×1254、完整提示詞及製作紀錄留本機。造型／比例未正式驗收。
- Go 1.24.13真實素材與回歸測試12主測試／21含子案例通過，失敗／略過0。
  收據 `over-theme-tests-v1-20261002.jsonl`，包含缺檔／錯SHA／圖號／位置／裁切／錨點／PNG尺寸、
  空格透明、三個單像素失配整組清除、恢復、切換及讀檔重建。
- 正常17-saved2／F95Bh起點不變，原FIFO兩次Up及695500000步正常Enter，持續500000步。
  實際IRQ1六筆；原FIFO與HD開關兩側至705000000步全部CPU、steps、cycles、IRQ1、
  原版色號畫面及A0000h以下RAM相同。完整匯流排只在終止後比較，讀取後不再執行。
- 正常8次身體貼圖／8中途／4個OVER場景／1終點，共21幀。獨立PBL差分、背景／敵人／OVER
  8×8圖面、GOLEMFNT中文字型及RGB→HD→中文合成不符0。盟友持續身份期望仍排除，
  該區只驗實際合成，沿用§50限制。首完整敵人HD，其餘15戰鬥樣本重疊回退。
- OVER在670400000及695000000的完整場景顯示HD；670300000未完整、697900000清除及
  705000000終點的OVER貢獻皆0。省略人物負對照各28,224放大像素；
  完整陣亡場景三行訊息直接對正式PW.EXE譯文不符0，省略中文字面各62,208。
- 另三個原版場景state及705000000終點，共4份實際載回。HD切換／重登記相同，各100,000步
  原版接續及完整匯流排相同，獨立完整圖面不符0。故意保留人物的清除／終點負對照
  各28,224／26,009。此項保存狀態抽測與正常路線分開。
- 收據 `over-runtime-v1-20261002.json`、`over-reload-v1-20261002.json`、
  `over-runtime-verification-v1-20261002.json`、`over-text-verification-v1-20261002.json`。
  實際檢視人物與Enter後合成圖，位置未縮小／平移。GUI、原版DAT、美術及封包仍待驗，
  §1.8保持READY，整項未完成；既有§49的43幀與§50的17幀保留，不混算完整覆蓋。

建置沿用Go研究子模組 `github.com/wicanr2/dosgolem/bodyprobe`，兩個replace指向
容器 `/src/worktrees/dosgolem` 及 `/src`，複製根go.sum；將正常／載回工具複製到子模組
各自子目錄的main.go，再 `go build -mod=mod`。直接以工作樹絕對檔名編譯會違反Go內部套件邊界，
首次建置因此失敗；修正一次性來源位置後同映像乾淨重建，沒有修改正式dosgolem API。

主要重生入口在容器 `/src`，均拒絕覆寫原結果：

```sh
python tools/hd/prepare_over_theme.py
workplace/hd/verify-over-runtime-v1-20261002 -initial-xlate workplace/hd/sivad-translation-v1-20261002-17-saved2.layer.json -out workplace/hd/over-runtime-v1-20261002
workplace/hd/verify-over-reload-v1-20261002 -out workplace/hd/over-reload-v1-20261002
python tools/hd/verify_over_runtime.py --out workplace/hd/over-runtime-verification-v1-20261002.json --check
python tools/hd/verify_over_text.py
python tools/hd/preserve_over.py
```

重生Go／主題／內容收據須使用乾淨研究輸入；現有來源不可覆寫。
沿用Go／Python映像、UID/GID1000、離線、CPU1、--rm、記憶體／pids／日誌限制及30–900秒外層逾時。
起跑load3.46，只驗指令／像素，不宣稱即時幀率或音訊驗收。

精確來源清單 `workplace/hd/over-source-manifest-v1-20261002.json`，快照
`workplace/hd/source-over-v1-20261002/`：516項、46,082,919 bytes。
前批551份snapshot雜湊未變；manifest保存收尾補記前文件bytes，歷史用snapshot回查。

## 52. OVER兩個正式前端與快速存讀檔

本批接續§51二十筆主題，驗正式 `cmd/psychicwar` 的Linux視窗及 `cmd/pwstep`。
原版起點沿用正常路線最後一次身體返回state，繼續等待原版陣亡出圖，不注入勝負或訊息。
Linux視窗以真正X按鍵切語言／HD、F10存檔、正常Enter清除及F11恢復；
保存狀態起點的UI抽測與§51固定種子正常流程分開，不冒稱從開機重跑或GUI亂數同狀態對拍。
開工load19.45，只驗像素及可用性，不宣稱幀率或音訊；音訊採null。

工具／產物入口：

- `tools/hd/over_frontend_check.sh`：容器內Xvfb按鍵與視窗擷取，子程序有trap與外層逾時。
- `tools/hd/export_over_frontend.go`：只讀實際F10 state，匯出原版色號／RGB供獨立對照。
- `tools/hd/over_pwstep_check.sh`：逐步前端從已驗陣亡state接續，相同動作分別開關HD，保存四組結果。
- `tools/hd/observe_over_frontend_colors.go`／`workplace/hd/observe-over-frontend-colors-v1-20261002`／`over-frontend-colors-v1-20261002.json`：以固定12500 cycles取樣原版RGB與三行疊字定色，不改原版。
- `tools/hd/verify_over_frontend.py`：獨立原版PBL／PNG／GOLEMFNT驗視窗圖與語言／HD／存讀檔。
- `workplace/hd/psychicwar-over-ui-v1-20261002`／`pwstep-over-ui-v1-20261002`：現行正式前端建置。
- `workplace/hd/export-over-frontend-v1-20261002`：前端真實保存狀態的唯讀匯出工具。
- `workplace/hd/over-frontend-v1-20261002/`：視窗／快速存檔／按鍵及原版輸出，全部本機。
- `workplace/hd/over-frontend-diagnostic-v1-20261002/`：首版等待不符後的真實快速存檔與語言／HD診斷，不當作通過收據。
- `workplace/hd/over-pwstep-v1-20261002*`：逐步工具HD開關等價流程與截圖，不取代視窗驗收。
- `workplace/hd/over-frontend-verification-v1-20261002.json`：獨立有限驗證收據，結果產生後回填。

沿用 `psychicwar-go-ebiten:latest`，映像識別
`sha256:083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7`，
可重現來源 `tools/docker/go-ebiten.Dockerfile`；Go1.24.13，無entrypoint，預設bash。
第一次inspect模板缺Entrypoint欄位是控制面格式錯誤，改讀完整Config後確認，沒有重建映像。

### 52.1 原版色盘恢復與中文缺行

首版視窗因三行期望不符而未進入切換流程，已終止；失敗視窗、原工具與診斷產物保留。
真正F10快照證實三行均Shown，前兩行FG／BG皆黑，第三行FG白；固定12500cycles重播
再次證實670711610及670724110步的色號0／15都映為黑色，670736610步色號15恢復白色，
前兩行仍保存黑色FG。這是定色後未追隨原版色盤的顯示層缺陷，已證實；原版內容指紋不變。
契約回填通用202 §2.3.1 READY後才修正，不改EXE／RAM／原版存檔或疊字快照格式。
所選色號只存記憶體，Shown僅從未改變的有效格更新RGB；讀檔後由吻合格恢復色號，
未知／失配時維持舊RGB並沿用失效機制。不重設透明格、錨點或失效次數。

修正與回歸入口：`worktrees/dosgolem/xlate/layer.go`、`xlate/layer_palette_test.go`，
收據 `workplace/hd/over-palette-tests-v1-20261002.jsonl`。修正後以前綴v2另建
`psychicwar-over-ui-v2-20261002`、`pwstep-over-ui-v2-20261002`、
`observe-over-frontend-colors-v2-20261002`，重新執行至
`over-frontend-colors-v2-20261002.json`、`over-frontend-v2-20261002/`、
`over-pwstep-v2-20261002/`、`over-frontend-verification-v2-20261002.json`；全部在本機workplace/hd。
既有v1圖、正常來源、快照及516份來源保全不覆寫。

黑屏繪製診斷另存 `workplace/hd/over-draw-diagnostic-v1-20261002.go`、同前綴二進位及目錄。
這是正式main的可丟棄複本，只增加CPU RGB與GPU讀回，不能當正式修正或正常效能證據。
第二版以同一完整陣亡state診斷圖面Draw回傳及GPU圖面，來源、二進位、輸出為
`workplace/hd/over-draw-diagnostic-v2-20261002.go`、無副檔名二進位及`-output/`目錄。
分離HD繪製緩衝的可丟棄原型為 `workplace/hd/over-draw-separate-v1-20261002.go`、
無副檔名二進位及`-output/`目錄；只驗英文HD人物缺失的修復，尚未正式接入。

Shift+F5退出陣亡畫面已由v2實際按鍵紀錄及F10原版state證實：F5本身不送原版，
ShiftLeft按下在678698824步送出，之後c取樣已非OVER，正常Enter尚未送出。
這是前端熱鍵修飾鍵漏送，不是渲染黑屏。024 §4.1限定契約補為READY：
暫存未確定用途的Shift，前端熱鍵消耗該修飾鍵；一般按鍵先送Shift，獨立Shift放開時送完整兩個事件。
已送原版的Shift保持到真實釋放，F1／F2／F3／F9與一般鍵映射不改。
實作及負對照入口 `apps/psychicwar/shift_input.go`、`shift_input_test.go`，正常GUI再驗。

英文HD缺人物另由隔離原型核對：原版CPU RGB與GPU上傳一致，HD圖面Draw為true，
但同一Ebiten圖像隨後被提示覆寫。分離HD圖像／RGBA的原型英文畫面恢復同位置人物。
依已確認前後順序回填024 §4.1 READY後接入正式前端，尚須獨立逐像素與真正按鍵重跑。
正式新建置 `workplace/hd/psychicwar-over-ui-v3-20261002`，視窗輸出
`over-frontend-v3-20261002/`、獨立收據 `over-frontend-verification-v3-20261002.json`；
逐步工具沿用未受輸入／GPU改動影響的v2。必要回歸收據 `over-ui-input-tests-v3-20261002.jsonl`。

### 52.2 本次進度核對：擷取完成，獨立驗證尚未通過

2026-10-02核對v3正式視窗九張、十份原版state匯出及v2逐步四組輸出均已產生。
原版按鍵紀錄只有Return按下／放開，無Shift或前端功能鍵。
色盤回歸41主測試／57含子案例、輸入27項通過，失敗／略過0。
主題／翻譯回歸14主測試／148含子案例通過，失敗0、略過6；略過不計完成。

現行 `tools/hd/verify_over_frontend.py` 獨立核對a至f六張視窗合成不符0。
g-after-enter第七張不符108個放大像素，驗證以ValueError中止。
差異原因未知，尚未判定是產品繪製或state與稍後視窗擷取時點不同。
未產生 `over-frontend-verification-v3-20261002.json` 通過收據。
h／i兩張F11及pwstep四組尚未完成此輪獨立核對，024 §1.8仍READY。
本次先依使用者要求同步CONTEXT及GitHub #34，不宣稱GUI整批通過。

### 52.3 選單游標相位與重跑入口

108像素差異定位在放大座標(372,288)至(386,299)，期望為藍底、截圖為黃色游標。
舊shot在F10之後等待三秒才抓圖，兩份輸入不保證游標相位相同。
保留v3失敗驗證器於 `workplace/hd/over-frontend-diagnostic-v1-20261002/verifier-v3-before-cursor.py`。
修改擷取工具只增加Enter後30張連續視窗樣本，每張都保留；獨立驗證器要求其中一張
與原版快照及正式中文字型的完整畫面完全吻合，不排除游標或放寬像素差門檻。
新輸出 `workplace/hd/over-frontend-v4-20261002/`，正式二進位沿用v3，沒有改正式程式。
新收據入口 `workplace/hd/over-frontend-verification-v4-20261002.json`，結果產生後回填。
來源保全入口 `tools/hd/preserve_over_frontend.py`，輸出
`workplace/hd/over-frontend-source-manifest-v1-20261002.json` 及
`workplace/hd/source-over-frontend-v1-20261002/`；舊516份快照先逐一核對，再保存本批。

v4獨立驗證已通過九張視窗、選單游標完整相位及F11原版位置／畫面核對，
但在pwstep的clear-off中止：色號矩形仍為完整OVER，RGB全黑，原版尚在淡出途中。
舊逐步腳本只有Enter150毫秒後等待1000毫秒，不能作為人物已清除的測試起點。
改為正常等待20000毫秒，維持相同起點與按鍵，不寫RAM或改判定；
輸出另存 `workplace/hd/over-pwstep-v3-20261002/`，二進位沿用v2。
舊腳本保存在診斷目錄 `pwstep-v2-before-wait.sh`，中止驗證器另存 `verifier-v4-before-wait.py`。
一次看圖曾誤認i畫面無字；最終PNG三行區域6202個非黑像素與獨立零差核對否定該判讀，
三行來源、字色及透明格亦與a／h相同，沒有因此再修改正式前端。

### 52.4 正式前端有限驗證結果

`over-frontend-verification-v4-20261002.json` 已產生，九張正式Linux視窗及四組pwstep的
獨立完整合成不符0。原版PBL、二十張PNG、正式文本與GOLEMFNT為期望來源，
所有state、色號／RGB匯出及輸入雜湊均核對；沒有以截圖當期望。
九張涵蓋中英／HD、Enter清除及F11恢復。F11的area／x／y／direction與原版畫面相同。
實際送入原版僅Return按下／放開，F5／Shift／F10／F11皆未洩漏。

新g畫面及30張相位共31張，30張完整零差、一張相位差135；沒有忽略游標區。
舊v3的108差異為F10與三秒後擷取的游標相位，屬驗證時點問題，未因此修改正式程式。
逐步四組使用相同起點，原版CPU／steps／cycles／seed／位置／畫面與A0000h以下RAM及
終止後匯流排在HD兩側相同；中文快照bytes相同。20秒正常等待後OVER已清除、回到選單，
原一秒資料停在淡出途中，舊版本保留，不重命名為成功資料。

HD完整人物省略負對照24,096，中文省略負對照22,772，均為放大像素；
這是前端收據範圍，與§51的原版遮格／字面數字分開。前後83筆色盤取樣的原版條件全相同，
670736610步舊前兩行FG黑、新三行FG白。色盤41主測試／57含子案例、輸入27項通過，
失敗／略過0；主題／翻譯14主測試／148含子案例通過，失敗0、略過6。

本批為保存狀態接續的真正X視窗按鍵及逐步有限驗證，不是從開機或固定GUI亂數對拍。
正式二進位沿用v3／v2，不含探針GPU讀回；無幀率／即時音訊或真機聲明。
024 §1.8維持READY：原版DAT、正式封包、美術及全部sprite仍待驗，其他角色GUI未外推完成。

本批精確保全654項、126,409,517 bytes，舊516份快照逐一SHA-256核對未變。
manifest保存收尾補記前文件bytes，歷史文件用snapshot回查，不能要求歷史文件等於新現況。
保全首輪因清單誤列不存在的dosgolem go.sum在預檢拒絕，尚未建立目錄；
移除不存在項後同映像重跑通過，沒有補造檔案或覆寫舊來源。

## 53. ENEMY03小圖塊候選

接續§47，目標為ENEMY03.PBL #15–#29十五個16×16來源的48×48候選。
2026-10-02最新結果：15張原版參照、完整提示詞、原生生成圖及48×48 RGBA均已保存。
內建image_gen逐張生成，共15次；原生圖完整畫布以ImageMagick 6.9.11-60 Lanczos縮放，
不裁切、平移或改用色鍵。已有小圖塊候選增加至60/180，另120個圖號待製作，正式美術仍0/360。
只按實際色號、輪廓及留白重繪，不推定角色、用途、動畫順序或透明合成模式。
同檔前三格等分組僅供展示；每次內建image_gen只附一張完整單格參照。
8×8正式覆繪與原版排版契約不變，本批不進正式執行期。

工具入口 `tools/hd/small_tile_assets.py`：容器內prepare產生唯讀原版解碼參照與完整提示詞；
convert保存已生成原生檔並以完整畫布縮為48×48；verify核對來源、RGBA與外框，不能當正式美術驗收。
全部產物沿用 `workplace/hd/redraw/`，前綴 `ENEMY03-small-*-v1-20261002`；
`ENEMY03-small-single-references-v1-20261002.json` 記來源SHA、PBL偏移、色號與參照SHA，
`ENEMY03-small-single-generation-jobs-v1-20261002.json` 保存完整提示詞，
`ENEMY03-small-single-selected-v1-20261002.json` 保存原生與專案路徑，
`ENEMY03-small-single-verification-v1-20261002.json` 保存技術核對及限制。
來源保全 `ENEMY03-small-single-source-manifest-v1-20261002.json` 保存55項、10,691,973 bytes，
包含解碼器、工具快照、提示詞、驗證收據及產物；前批85份來源雜湊逐一核對未變。

對照入口 `ENEMY03-small15-single-comparison-v1-20261002.png`，五欄、六列，每兩列依序
原版／候選，圖號依15–19、20–24、25–29排列。僅作展示，不推定動畫。
工具與解碼器的精確bytes另存 `ENEMY03-small-single-tool-source-v1-20261002.py`
及 `ENEMY03-small-single-decoder-source-v1-20261002.py`，由source-manifest掛接。

Python 3.13.15獨立核對46份PNG的CRC、完整解碼、尺寸、RGBA與UID/GID1000通過。
原版參照及展示圖RGB不符0，參照位移負對照40,824，不透明候選負對照拒絕。
外框以alpha≥128量測，僅7/15完全相同。#20底界少3列、#24少2列，#15–#19及#26少1列；
其餘7張精確。輪廓遮罩不同像素逐張列在收據，不視為造型通過。
目視仍有圓角化、局部色區平滑化及小塊配色差異；用途、原版合成、動畫、正常路徑與權利未驗。
本批不進正式執行期，024 §1.5維持DRAFT。邊界修整仍是必要未完成項，兩張局部試驗見§53.1；
先接續ENEMY04–11小圖塊與其他sprite，不能以候選數量代替完整HD驗收。

### 53.1 #20／#24底界修整

原生圖與48×48各以alpha 1／64／128／192／254核對。原生#20在alpha≥128的底界1106/1254，
原版要求15/16；#24為884/1254，原版要求12/16。48×48差異均已存在於原生幾何，
降低alpha門檻只能計入薄邊或零星殘點，不能作位置修正。舊圖及55項manifest保留。
兩張局部修整沿用完整原版參照與v1作圖面依據，不裁切或拉伸整張。
完整提示詞與實際路徑入口 `ENEMY03-small-boundary-generation-jobs-v2-20261002.json`
及 `ENEMY03-small-boundary-selected-v2-20261002.json`；候選原生／48×48分別採
`ENEMY03-small-{20,24}-single-generated-v2-20261002.png` 及同前綴 `single-48`。
容器核對入口 `tools/hd/refine_small_tile.py`，對照 `ENEMY03-small-boundary-comparison-v2-20261002.png`
每列原版／v1／v2，先#20後#24。技術收據與來源保全採同boundary前綴verification／source-manifest。
工具精確bytes採 `ENEMY03-small-boundary-tool-source-v2-20261002.py` 保存；產生結果後回填，未進正式執行期。

v2未採用。11份PNG技術核對通過，原版參照／對照RGB不符0，舊55項來源雜湊未變。
17項來源保全3,158,616 bytes，包括原生圖、48×48、完整提示詞、工具快照及驗證收據。
#20外框仍[0,6,48,42]，外框偏差3未改善、輪廓遮罩差157→167；#24變成[1,4,47,32]，
外框總偏差2→7、遮罩差180→321，新增四邊內縮。最上36列RGBA變動各1,571／1,569，
不符合「只改下方」的局部不變要求，不宣稱區外維持。兩張精確外框仍0/2。
單靠原版參照＋既有候選＋文字座標的局部編輯，本次未修正原生幾何；不調alpha門檻讓它通過。
原生圖、48×48、完整提示詞、收據及失敗版本均保留，現行候選仍v1，總數60/180、正式美術0/360。
後續修整保留為未完成項；先接續ENEMY04來源及其他sprite，不以重複同類生成代替解法。

## 54. ENEMY04小圖塊來源與候選

範圍為ENEMY04.PBL #15–#29的十五個16×16來源，沿用完整原版位置與留白，
不推定用途／動畫／透明合成。容器入口 `tools/hd/small_tile_assets.py prepare ENEMY04`，
參照及完整提示詞採 `ENEMY04-small-single-references-v1-20261002.json`／
`ENEMY04-small-single-generation-jobs-v1-20261002.json`；沿用 `workplace/hd/redraw/`。
候選原生／48×48採同small單格前綴；實際路徑由 `ENEMY04-small-single-selected-v1-20261002.json` 記錄。
展示、核對與來源保全採comparison／verification／source-manifest前綴，完整結果如下。

來源準備完成：15張384×384最近鄰參照與原版16×16解碼RGB逐像素不符0，
完整提示詞已保存。參照階段獨立來源清單 `ENEMY04-small-reference-source-manifest-v1-20261002.json`
保留原bytes，候選來源另建完整source-manifest。

內建image_gen逐張生成十五張原生圖，以完整畫布Lanczos轉為48×48 RGBA；不裁切／平移。
Python 3.13.15核對46 PNG CRC、完整解碼、尺寸、RGBA與UID/GID1000通過；
原版參照及展示RGB不符0，參照位移負對照42,816、不透明負對照拒絕，前批85來源未變。
完整來源保全55項、11,496,460 bytes，包含原生／48×48、完整提示詞及工具／解碼器精確快照。
提示詞 `ENEMY04-small-single-generation-jobs-v1-20261002.json`，收據同single前綴verification／source-manifest。
對照 `ENEMY04-small15-single-comparison-v1-20261002.png` 為五欄六列，每兩列原版／候選，
依序#15–#19、#20–#24、#25–#29；展示分組不推論動畫。

alpha≥128外框僅8/15精確。#19／#24／#28底界少1列、#27／#29少2列；
#25頂界提前1列且底界少1列，#21左右各內縮1列／頂界少2列。全部逐張遮罩差異列在收據，
僅#15／#23的二值遮罩不符0，不能由外框相同推論內部輪廓或色區相同。
目視仍有細條圓角化、密集色塊膠囊化及局部配色／漸層差異；用途、動畫、原版合成與正常路徑未驗。
候選總數75/180，剩餘105個圖號；正式美術仍0/360，未新增正式主題接入。

本輪依grilling展示既有v5特效重疊原型，透光混色／一般透明選擇已送出，尚未收到答案。
此為待決狀態，不將建議當作定案，024 §1.5保持DRAFT；不阻塞其他sprite製作。

## 55. ENEMY05小圖塊來源與候選

範圍為ENEMY05.PBL #15–#29十五個16×16來源。沿用原版位置、範圍、色區與留白，
不推定用途、動畫或合成模式。容器入口 `tools/hd/small_tile_assets.py prepare ENEMY05`，
單格參照及完整提示詞為 `ENEMY05-small-single-references-v1-20261002.json`／
`ENEMY05-small-single-generation-jobs-v1-20261002.json`，沿用 `workplace/hd/redraw/`。
內建image_gen逐張生成，原生與48×48採同small單格前綴，實際工具及專案保存路徑由
`ENEMY05-small-single-selected-v1-20261002.json` 記錄。convert以完整畫布Lanczos轉檔，
verify核對來源、PNG、alpha、輪廓、負對照及來源保全；不能當正式美術或正常路徑完成。
對照 `ENEMY05-small15-single-comparison-v1-20261002.png` 五欄六列，每兩列原版／候選、由#15排至#29。
verification／source-manifest與工具／解碼器快照採同single前綴，不覆寫舊圖。

### 55.1 十五張候選保存與技術核對

內建image_gen逐張生成十五張1254×1254 RGBA原生圖，完整提示詞與工具原生路徑由上述JSON保存。
Python 3.11.2／ImageMagick 6.9.11-60以完整畫布Lanczos轉成48×48 RGBA，不裁切、平移或去底色。
首次命令誤用映像沒有的python，容器未啟動；改python3後同映像、同素材轉檔完成。
Python 3.13.15核對來源、PNG CRC、完整解碼、尺寸、alpha及UID/GID1000。

**confirmed技術結果**：46 PNG通過，原版參照／展示RGB不符0；位移負對照41,112、
不透明負對照拒絕。原版ENEMY05.PBL SHA-256為
`2b390a5c4a5a2c6e49a9e27b02c89dd839c6c932dad0f568aee6b88e2397180a`。
新55項來源保全12,406,798 bytes，含工具及解碼器精確快照；十五張專案原生複製與工具輸出SHA相同。
舊ENEMY02的85份來源、ENEMY03的55／17及ENEMY04的55份來源雜湊未變。

以alpha≥128量測，12/15外框相同；#21／#22底界少1列、#29頂界少1列。
全部十五張二值輪廓仍有差異，外框相同不等於內部空隙、配色或造型通過。

| 圖號 | 原版外框×3 | 候選外框 | 二值輪廓差異像素 |
|---|---|---|---|
| #15 | 0,0,48,48 | 0,0,48,48 | 38 |
| #16 | 0,3,48,48 | 0,3,48,48 | 17 |
| #17 | 0,6,48,42 | 0,6,48,42 | 24 |
| #18 | 0,3,48,45 | 0,3,48,45 | 9 |
| #19 | 0,9,48,45 | 0,9,48,45 | 104 |
| #20 | 0,0,48,42 | 0,0,48,42 | 118 |
| #21 | 0,6,48,42 | 0,6,48,41 | 48 |
| #22 | 0,3,48,42 | 0,3,48,41 | 64 |
| #23 | 0,0,48,48 | 0,0,48,48 | 135 |
| #24 | 0,0,48,48 | 0,0,48,48 | 295 |
| #25 | 0,0,48,48 | 0,0,48,48 | 27 |
| #26 | 0,0,48,48 | 0,0,48,48 | 181 |
| #27 | 0,6,48,39 | 0,6,48,39 | 33 |
| #28 | 0,6,48,39 | 0,6,48,39 | 2 |
| #29 | 0,6,48,39 | 0,7,48,39 | 46 |

實際檢視五欄六列對照：細條與密集色塊圓角化、額外漸層／邊緣色暈及部分灰色區變白，
仍需美術修整。用途、動畫、原版透明合成、正常路徑及公開權利維持未知／未驗。
本批只增候選至90/180，剩餘90個圖號為ENEMY06–11；正式ENEMY美術仍0/360，未新增正式接入。
核對收據 `ENEMY05-small-single-verification-v1-20261002.json`，來源入口
`ENEMY05-small-single-source-manifest-v1-20261002.json`，均在 `workplace/hd/redraw/`。

## 56. 敏頓正常攻擊的小圖塊來源

本切片接續§48的既有14-minton1正常攻擊路線。目的為辨識實際16×16來源、位置與模式，
不由候選造型推定用途，不改原版規則或HD正式效果路徑；024 §1.5仍DRAFT。
原版13-healed.state、方向鍵FIFO、382,200,000步起按住空白鍵30,000,000步及
420,000,000步終點沿用§48.1。原版state與seed在執行前固定，不重擲。
新研究工具 `tools/hd/observe_small_projectiles.go` 保存所有16×16及24×32的一般貼圖
來源、入口／返回色號與原始定位，並以不觀察分支及既有§48終點核對觀察副作用。
獨立核對入口 `tools/hd/verify_small_projectiles.py` 從PBL解碼實際來源及同檔XOR配對，
不把無方向差分配對稱為已證實方向，不宣稱所有重疊場景或能力語意。
本機輸出採 `workplace/hd/small-projectiles-minton-v1-20261002*`，來源保全與結果見以下各節。
保全入口 `tools/hd/preserve_small_projectiles.py`，來源快照在
`workplace/hd/source-small-projectiles-v1-20261002/`，清單為
`workplace/hd/small-projectiles-source-manifest-v1-20261002.json`。

### 56.1 正常來源與獨立核對結果

**confirmed，限本路線**：Go1.24.13／dosgolem f8c1a6e保存1,009次一般貼圖。
觀察、不觀察與§48既有終點的暫存器、flags、steps、cycles、IRQ1、快照RAM及色號畫面相同。
終點420,000,000步、cycles2,035,693,244、IRQ1為39；原版色號SHA-256為
`ea17529f4b9b07828402fda8bde3f9e0908ddd76c1bc9220af8372489d4a1496`，
快照RAM SHA-256為 `9d66c8894ab1bd186c8c0ad34b0d2d0a60bece99349aa79119d615aff9b671a2`。
執行前state、A48Ch seed、方向鍵及攻擊時點沿用既有路線；沒有改RAM或選另一組結果。

Python3.13.15獨立解碼26檔PBL及同檔、同尺寸差分。1,009次來源皆唯一，未知／歧義0，
來源運算、整屏after及矩形外不符0。首個有效小圖塊負對照：改來源一個byte差1像素，
平移4像素差138；不以Go內部match訊號作答案。XOR配對只證明來源bytes，不推定方向。

| 來源 | 此路線次數 | 位置與範圍 |
|---|---:|---|
| BEAM #0／#1／#2完整來源 | 37／40／37 | 16×16，y=160，x=40至248、每16像素 |
| BEAM 0^1／0^2／1^2差分 | 187／186／183 | 同上，共670次BEAM |
| ENEMY00 #21／#22／#23完整來源 | 36／30／38 | 16×16，y=160，x=56至184、每16像素 |
| ENEMY00 21^22／21^23／22^23差分 | 37／35／33 | 同上，共209次敏頓小圖塊 |
| ENEMY00 #6完整來源／6^7／7^8 | 2／8／8 | 24×32，(32,152)，共18次身體 |
| FIGHT #0／#2完整來源、0^1／2^3 | 2／2、54／54 | 24×32，(40,144)及(256,144)，共112次 |

一般16×16共879次、24×32共130次，不把後者全部稱作角色身體。
#21–#23確實由敏頓正常攻擊路線使用；傷害、能力名稱、移動方向、完整重疊場景與內建遮罩未驗。
既有首場#18–#20證據維持，不重新開啟已完成來源研究；其他敵人的小圖塊用途仍須正常樣本。
本節沒有新美術或正式接入，候選維持90/180、正式美術0/360，024 §1.5仍DRAFT。
獨立收據 `workplace/hd/small-projectiles-minton-independent-v1-20261002.json`。

### 56.2 來源保全及重跑入口

3,149項、137,553,903 bytes來源及產物已保全，114份可變程式／文件精確快照。
原版27份輸入只記雜湊，不複製原版素材；所有state及研究產物只留本機。
Go映像ID `sha256:1a6d4452c65dea36aac2e2d606b01b4a029ec90cc1ae53890540ce6173ea77ac`。
研究子模組沿用§48.1；本輪精確go.mod在 `small-projectiles-build-v1-20261002.mod`，
二進位為 `observe-small-projectiles-v1-20261002`。兩者均在 `workplace/hd/`。
舊來源清單保存補記前文件bytes，歷史回查snapshot，不以現況文件代替舊bytes。

Docker內 `/src` 的重跑入口如下，輸出必須使用全新前綴，不覆寫現有收據：

```sh
workplace/hd/observe-small-projectiles-v1-20261002 -out workplace/hd/新的唯一前綴
python tools/hd/verify_small_projectiles.py --probe workplace/hd/新的唯一前綴 --out workplace/hd/新的獨立核對檔名.json
```

使用既有Go／Python Docker、UID/GID1000、CPU1、有界記憶體／pids／日誌及30–240秒逾時。
原版掛 `/orig:ro` 及 `/src/workplace/original:ro`；build沿用已存在gocache／gomodcache，離線。
此工具固定14-minton1正常輸入，不藉新前綴調整seed、攻擊時機或挑選通過結果。

### 56.3 小圖塊來源的回填護欄

【HD-SMALL-02】不可變鍵為DOS ENEMY00.PBL，SHA-256
`8241b0ea73b1e7402b13adc434f10a5b1288f88f01fcfc3ba5710e4923fe6067`。
下列offset為PBL檔案偏移，不是執行期DS:BX或IDA ea。

| 圖號／檔案偏移 | 已證實語意 | 新證據 | 較早入口與必要標記 |
|---|---|---|---|
| #21／0x1924 | 敏頓正常攻擊一般貼圖來源之一 | §56.1、獨立收據，36次完整來源及相關差分 | §20.1、024 §1.5，HD-SMALL-02 |
| #22／0x19A1 | 同上 | §56.1、獨立收據，30次完整來源及相關差分 | 同上 |
| #23／0x1A1B | 同上 | §56.1、獨立收據，38次完整來源及相關差分 | 同上 |

護欄 `tools/hd/verify_small_projectile_backlinks.py` 核對PBL雜湊／偏移／解碼、來源收據、
較早§20.1與024 §1.5的回填標記；移除標記的負對照必須失敗。
收據 `workplace/hd/small-projectiles-backlinks-v1-20261002.json`；工具及文件快照採同backlink-source前綴。
這次只閉合三個圖號的正常來源，不改玩家流程；其HD垂直鏈仍待DRAFT契約審查、正式接入與正常驗收。

## 57. ENEMY06小圖塊來源與候選

範圍為ENEMY06.PBL #15–#29十五個16×16來源，保留原版位置、色區、空隙與留白。
其他檔案已證實的小圖塊用途不套用到ENEMY06；本批用途、動作與原版合成仍待正常證據。
沿用 `tools/hd/small_tile_assets.py prepare ENEMY06`，產物在既有 `workplace/hd/redraw/`：
`ENEMY06-small-single-references-v1-20261002.json` 保存原版雜湊、偏移及單格參照，
`ENEMY06-small-single-generation-jobs-v1-20261002.json` 保存十五份完整提示詞。
內建image_gen逐張生成，實際原生工具／專案路徑由同single前綴selected記錄；
convert以完整畫布Lanczos轉成48×48，verify獨立核對來源、PNG、alpha、邊界與負對照。
對照 `ENEMY06-small15-single-comparison-v1-20261002.png` 為五欄六列，每兩列原版／候選。
verification／source-manifest及工具／解碼器快照採同single前綴，結果產生後回填；不覆寫舊來源。

### 57.1 進度核對

2026-10-02依使用者要求先同步進度。十五份參照的RGB與原版解碼不符0，完整提示詞已建立。
初次核對的selected只有#15、#16、#17三筆；唯讀檢查確認三份1254×1254工具原生輸出存在。
三筆預定專案PNG皆未建立，48×48轉檔、完整PNG／輪廓核對、展示圖與來源保全尚未完成。
因此本批只列進行中，不增加已保存並核對的90/180候選；正式美術0/360與現行20筆主題維持。
工具輸出可用性不證明造型、透明合成、動作或正常玩家路徑通過。

### 57.2 原版白色形塊補繪候選

十五張v1完成後，視覺對照發現#20漏掉原版白色形塊。白色是原版非零前景色號，不能當成透明底。
新提示詞另存 `workplace/hd/redraw/ENEMY06-small-20-white-generation-job-v2-20261002.json`，
只以原版整張RGB參照生成，不覆寫v1。原生／48×48候選採同white前綴，
selected／verification／source-manifest及三欄comparison採同前綴；結果產生後回填。
核對包含原版來源雜湊、白色位置覆蓋、PNG／alpha／邊界、v1對照及既有55份來源不變。
Docker重跑工具為 `workplace/hd/enemy06_white_v2.py convert` 與 `verify`；
convert沿用完整畫布Lanczos轉檔，verify以原版色號15的位置獨立量測白色覆蓋，不移動或補畫像素。
同white前綴的tool／decoder／base-tool-source `.py` 保存工具精確bytes；
pre-snapshot-source-manifest保存首次清單，source-manifest補入快照回查連結，不覆寫v1。

### 57.3 本批完成的技術核對與限制

十五張v1原生／48×48、參照、完整提示詞與實際工具路徑已保存。Python 3.13.15獨立核對46份PNG的
CRC、完整解碼、尺寸、RGBA與UID/GID通過；原版參照及720×864展示圖RGB不符0。
位移負對照44,088，不透明負對照拒絕；十五張專案原生複製與工具輸出SHA相同。
新55份來源共11,637,956 bytes，含精確工具／解碼器快照。ENEMY02既有85份、ENEMY03的55／17、
ENEMY04與ENEMY05各55份來源均核對未變，未覆寫舊候選或原版。

以alpha≥128量測，v1只有#21、#22、#24、#25、#26共5/15外框相同，只有#26二值輪廓完全相同。
其餘十張外框有偏差；#27頂界內縮3列且底界延伸2列，#28頂界內縮3列，#29頂界內縮1列。
每張原生圖均作完整畫布轉檔，沒有用裁切或平移修改候選位置。

| 圖號 | 原版外框×3 | v1外框 | 二值輪廓差 |
|---|---|---|---|
| #15 | 0,9,48,42 | 0,9,48,41 | 82 |
| #16 | 0,9,48,39 | 0,10,48,38 | 105 |
| #17 | 0,9,45,42 | 0,9,46,42 | 64 |
| #18 | 0,0,48,45 | 1,1,47,46 | 338 |
| #19 | 0,0,48,45 | 1,2,46,45 | 599 |
| #20 | 0,0,48,45 | 0,3,48,45 | 348 |
| #21 | 0,3,48,45 | 0,3,48,45 | 133 |
| #22 | 0,3,48,45 | 0,3,48,45 | 16 |
| #23 | 0,3,48,45 | 0,2,48,45 | 32 |
| #24 | 0,0,48,48 | 0,0,48,48 | 48 |
| #25 | 0,0,48,48 | 0,0,48,48 | 38 |
| #26 | 0,0,48,48 | 0,0,48,48 | 0 |
| #27 | 0,9,48,33 | 0,12,48,35 | 513 |
| #28 | 0,9,48,33 | 0,12,48,33 | 276 |
| #29 | 0,9,48,33 | 0,10,48,33 | 76 |

#20首次候選漏掉原版白色形塊，v2明示這些白色是不透明前景後另存原生與48×48。
以alpha≥128且RGB各分量≥220量測原版色號15位置，白色覆蓋由0/342增加至327/342，
白色出現在原版白色範圍外的像素1；外框總偏差3→0，二值輪廓差348→18。
原版／v1／v2三欄432×144對照RGB不符0，六份PNG技術核對通過，
省略白色負對照342、不透明負對照拒絕；v1的55份來源未變。
v2的19份來源共1,879,526 bytes，包含首次清單與工具精確快照。
這是白色缺項的有限改善，尚有15個原版白色位置未符合量測門檻，不能當作配色或完整美術通過。

v1的圓角化、色區形狀、額外漸層、白色／紫色透明邊緣噪點仍待修整；
v2也仍有圓角、輪廓及邊緣問題。兩版保留，不直接修改正式美術或產品接入。
候選總量增加至105/180，剩餘ENEMY07–11共75個小圖塊；正式美術0/360、現行20筆主題與
024 §1.5 DRAFT維持。用途、動作、原版合成與正常路徑仍未驗，全部素材留本機。

## 58. ENEMY07小圖塊來源與候選

範圍為ENEMY07.PBL #15–#29十五個16×16來源，保留原版位置、色區、空隙與留白。
十五張來源沒有色盤敏感色號2／8／10，可沿用已核對色盤；用途、動作與原版合成仍未知。
Docker入口 `tools/hd/small_tile_assets.py prepare ENEMY07`，產物在既有 `workplace/hd/redraw/`：
`ENEMY07-small-single-references-v1-20261002.json` 保存來源雜湊／偏移與十五張參照，
`ENEMY07-small-single-generation-jobs-v1-20261002.json` 保存十五份完整提示詞。
依§57.2實際漏白證據，本批提示詞另明示白色及灰色是原版不透明前景，只有黑色空隙透明。
內建image_gen逐張生成，selected記錄實際工具／專案路徑；完整畫布Lanczos轉48×48。
verification／source-manifest及tool／decoder-source快照採同single前綴；結果產生後回填。
對照 `ENEMY07-small15-single-comparison-v1-20261002.png` 五欄六列，每兩列原版／候選。
原版與候選留本機，不由上一檔已解出的用途推定本批語意，不覆寫舊來源。

### 58.1 技術核對與限制

ENEMY07.PBL SHA-256為 `c2924057e1d704d30be7a644c870b0879bb72e155183b6ab7c4c6e864904d519`。
十五張原版參照RGB不符0，十五份完整提示詞已保存。每張原版參照先查看後才用內建image_gen生成，
原生圖及整張Lanczos 48×48 RGBA保存於專案，沒有裁切或平移。
Python 3.13.15獨立核對46份PNG的CRC、完整解碼、尺寸、RGBA及UID/GID通過；
原版參照與720×864展示圖RGB不符0，位移負對照39,240、不透明負對照拒絕。
十五張原生複製與工具輸出SHA相同，完整來源55項共10,220,438 bytes，含工具／解碼器快照。
既有ENEMY02的85份、前批ENEMY06的55份與白色修整19份來源均核對未變。

以alpha≥128量測，外框7/15相同：#18、#21–#24、#26及#28；僅#22及#26二值輪廓相同。
#15底界少3列、#16少1列、#17少2列；#19底界超出1列，#20四邊總偏差5；
#25底界少1列、#27少2列、#29少1列。完整量測如下。

| 圖號 | 原版外框×3 | 候選外框 | 二值輪廓差 |
|---|---|---|---|
| #15 | 0,12,48,42 | 0,12,48,39 | 118 |
| #16 | 0,3,48,45 | 0,3,48,44 | 64 |
| #17 | 0,3,48,45 | 0,3,48,43 | 174 |
| #18 | 0,0,48,45 | 0,0,48,45 | 297 |
| #19 | 0,0,48,45 | 0,0,48,46 | 390 |
| #20 | 3,0,45,45 | 4,2,46,44 | 479 |
| #21 | 0,0,48,45 | 0,0,48,45 | 44 |
| #22 | 0,9,48,45 | 0,9,48,45 | 0 |
| #23 | 0,9,48,39 | 0,9,48,39 | 30 |
| #24 | 0,0,48,48 | 0,0,48,48 | 11 |
| #25 | 0,0,48,48 | 0,0,48,47 | 226 |
| #26 | 0,0,48,48 | 0,0,48,48 | 0 |
| #27 | 0,15,48,42 | 0,15,48,40 | 99 |
| #28 | 0,15,48,42 | 0,15,48,42 | 5 |
| #29 | 0,15,48,42 | 0,15,48,41 | 48 |

視覺對照仍有色帶位置／寬度偏差、密集色區圓角化、額外漸層、邊緣色暈及透明噪點。
不以外框相同推定色區、造型或動畫已通過；白色提示詞補強不證明整批方法因果。
候選總量120/180，剩餘ENEMY08–11共60個小圖塊；正式ENEMY美術0/360，現行主題20筆，
未新增正式接入。024 §1.5 DRAFT維持，用途、動作、原版合成與正常路徑未驗，全部留本機。

## 59. ENEMY08小圖塊來源與候選

範圍為ENEMY08.PBL #15–#29十五個16×16來源，保留原版位置、色區、空隙與留白。
來源未使用色盤敏感色號2／8／10，可沿用已核對色盤；用途、動作與原版合成仍未知。
Docker入口 `tools/hd/small_tile_assets.py prepare ENEMY08`，產物在既有 `workplace/hd/redraw/`：
`ENEMY08-small-single-references-v1-20261002.json` 保存來源雜湊／偏移與十五張參照，
`ENEMY08-small-single-generation-jobs-v1-20261002.json` 保存十五份完整提示詞。
提示詞沿用§58的白色／灰色不透明前景契約，只有黑色空隙透明，不由圖形猜完整物件。
內建image_gen逐張生成，selected記錄工具／專案路徑；完整畫布Lanczos轉48×48。
verification／source-manifest及tool／decoder-source快照採同single前綴，結果產生後回填。
對照 `ENEMY08-small15-single-comparison-v1-20261002.png` 五欄六列，每兩列原版／候選。
原版與候選留本機，不把其他檔案的已知用途套到本批，不覆寫舊來源。

### 59.1 進行中檢查點

2026-10-02核對：十五張原版參照與完整提示詞已準備；內建image_gen已生成#15–#20六張原生圖。
六張已複製到selected列出的專案路徑，逐張SHA與工具輸出相同，UID/GID均為1000:1000。
#21–#29九張尚待生成；尚無48×48轉檔、verification、source-manifest或整批對照圖。
本批不計入已保存且技術核對的120/180；正式ENEMY美術0/360、現行主題20筆不變。
上述後續產物名稱是既定入口，並非已完成結果；用途、造型、動作與正式接入仍未驗。

接續轉檔入口 `workplace/hd/enemy08_convert_resume_v1.py`，Docker內執行。
既存原生圖先核對SHA一致，缺少的複製；48×48只建立尚不存在的檔案，沿用完整畫布Lanczos。
本批source-manifest另納入此工具及同single前綴resume-source快照，不改寫其他批次工具。

### 59.2 候選保存與技術核對

十五張原生圖及完整畫布Lanczos 48×48 RGBA已保存，無裁切或平移。
轉檔使用既有psychicwar-go-ebiten映像內的ImageMagick 6.9.11-60 Q16 x86_64。
本輪接續九次內建image_gen；前六張工具輸出已保存，十五張專案原生SHA均與實際工具輸出相同。
Python 3.13.15獨立核對46 PNG的CRC、完整解碼、尺寸、RGBA及UID/GID通過。
原版參照與展示RGB不符0，位移負對照42,984，不透明負對照拒絕。
來源PBL SHA-256為 `eb4e8673858a5425bba68edba74cad1138caf2c64d58d429e40c7d7a7ec072a7`。

外框9/15相同，二值輪廓8/15相同。#17四邊各內縮2列；#19左右及頂端各內縮2列、底界少1列；
#18／#23底界少1列；#20右界超出2列、底界超出1列；#27頂界與底界各下移1列。
數值為48×48畫布、alpha≥128、右／下界不含邊界；二值輪廓只驗占用，不驗內部配色或造型。

| 圖號 | 原版外框×3 | 候選外框 | 二值輪廓差異像素 |
|---|---|---|---|
| #15 | 0,9,48,48 | 0,9,48,48 | 0 |
| #16 | 0,3,48,48 | 0,3,48,48 | 0 |
| #17 | 0,0,48,45 | 2,2,46,43 | 396 |
| #18 | 0,0,48,39 | 0,0,48,38 | 59 |
| #19 | 0,0,48,39 | 2,2,46,38 | 265 |
| #20 | 0,9,45,45 | 0,9,47,46 | 89 |
| #21 | 0,6,48,45 | 0,6,48,45 | 0 |
| #22 | 0,3,48,48 | 0,3,48,48 | 1 |
| #23 | 0,6,48,45 | 0,6,48,44 | 53 |
| #24 | 0,0,48,48 | 0,0,48,48 | 0 |
| #25 | 0,0,48,48 | 0,0,48,48 | 0 |
| #26 | 0,0,48,48 | 0,0,48,48 | 0 |
| #27 | 3,27,48,42 | 3,28,48,43 | 83 |
| #28 | 3,21,48,42 | 3,21,48,42 | 0 |
| #29 | 0,24,48,42 | 0,24,48,42 | 0 |

視覺對照仍有圓角化、交錯色塊造型／位置變動、額外漸層及邊緣透明噪點。
#24–#26部分直條末端圓角及長度與原版有差異；整張外框相同無法證明各色區完全一致。
本批正式美術未驗收，未新增正式接入，用途、動畫、原版合成、正常玩家路徑與公開權利仍未驗。

本批source-manifest保全57項、10,366,531 bytes，包含正式核對工具／解碼器與接續轉檔工具精確快照。
舊ENEMY02的85份、前批ENEMY07的55份來源雜湊未變；原版只記雜湊，不複製到來源保全包。
verification原始files表保存基礎55項，source-manifest另加接續工具與快照兩項，數字不同並非缺檔。

本批完成後小圖塊候選135/180、剩餘ENEMY09–11共45圖號；正式ENEMY美術0/360、現行主題20筆不變。
特效024 §1.5維持DRAFT，既定8×8、原版排版、人物在後框在前及全部sprite範圍維持。

## 60. ENEMY09小圖塊來源與候選

來源為ENEMY09.PBL #15–#29十五個16×16圖塊，保留原版位置、各色區、空隙與留白。
PBL SHA-256：`f6e09a218300e9848360493ecac122617639475106753350944ef078b91e71d9`。
全部來源未使用色盤敏感色號2／8／10；偏移、尺寸及色號已核對，用途、動作與合成仍未知。
Docker入口 `tools/hd/small_tile_assets.py prepare ENEMY09`，輸出沿用 `workplace/hd/redraw/`。
`ENEMY09-small-single-references-v1-20261002.json` 保存十五張原版參照、偏移與原始色號雜湊，
`ENEMY09-small-single-generation-jobs-v1-20261002.json` 保存十五份完整提示詞。
提示詞沿用§58–59的白色／灰色不透明前景契約，只有黑色空隙透明，不猜補物件身分。
內建image_gen逐張生成；selected記錄工具／專案路徑，完整畫布Lanczos轉48×48 RGBA。
verification／source-manifest及tool／decoder-source快照採同single前綴，結果產生後回填。
對照 `ENEMY09-small15-single-comparison-v1-20261002.png` 五欄六列，每兩列原版／候選。
原版與候選留本機，不覆寫前批來源，未產生的結果名稱只作接續入口，不當作完成證據。

### 60.1 #21右側局部色區修整入口

原版#21逐列色號確認左色區在[0,4,11,13]，右側裁切色區在[12,4,16,13]，x=11整欄為黑色空隙。
v1把兩塊改為置中的完整色區，右側裁切部分遺失。另一次內建image_gen只用原版參照生成v2，
完整提示詞 `ENEMY09-small-21-clipped-generation-job-v2-20261002.json`，selected記實際輸出與保存路徑。
generated／48／comparison／verification／source-manifest採同clipped前綴，全部在既有redraw目錄。
修整核對工具 `workplace/hd/enemy09_clipped_v2.py`，Docker限定；v1及其來源保留，結果產生後再判定是否改善。

### 60.2 十五張v1候選保存與技術核對

十五次內建image_gen輸出與48×48 RGBA已保存。ImageMagick 6.9.11-60 Q16完整畫布Lanczos縮放，
不裁切或平移，十五張原生複本SHA與實際工具輸出相同。
Python 3.13.15獨立核對46 PNG的CRC、完整解碼、尺寸、RGBA與UID/GID通過。
原版參照／展示RGB不符0，位移負對照43,344，不透明負對照拒絕；舊ENEMY02的85份來源未變。
55項來源11,005,995 bytes已保全；前批ENEMY08的57項SHA未變，原版只記雜湊。

v1外框5/15相同，二值輪廓只有#26相同。#15頂端內縮3列、底界少1列；#16頂端內縮3列、底界少2列；
#17底界少3列；#18／#22／#27少1列、#23少2列；#28上下界各內縮1列、#29底界超出1列。
#21v1外框總偏差4、輪廓差441，右側裁切色區遺失並置中，修整另見§60.3。
數值為48×48畫布、alpha≥128、右／下界不含邊界。

| 圖號 | 原版外框×3 | v1候選外框 | 二值輪廓差異像素 |
|---|---|---|---|
| #15 | 0,0,48,45 | 0,3,48,44 | 28 |
| #16 | 0,0,48,45 | 0,3,48,43 | 132 |
| #17 | 0,3,48,48 | 0,3,48,45 | 51 |
| #18 | 0,0,48,45 | 0,0,48,44 | 38 |
| #19 | 0,3,48,45 | 0,3,48,45 | 24 |
| #20 | 0,3,48,42 | 0,3,48,42 | 27 |
| #21 | 0,12,48,39 | 1,10,47,39 | 441 |
| #22 | 0,12,48,39 | 0,12,48,38 | 27 |
| #23 | 0,12,48,39 | 0,12,48,37 | 128 |
| #24 | 0,0,48,48 | 0,0,48,48 | 158 |
| #25 | 0,0,48,48 | 0,0,48,48 | 9 |
| #26 | 0,0,48,48 | 0,0,48,48 | 0 |
| #27 | 0,9,48,42 | 0,9,48,41 | 16 |
| #28 | 0,9,48,42 | 0,10,48,41 | 70 |
| #29 | 0,9,48,42 | 0,9,48,43 | 83 |

對照仍有圓角化、色區位置及連接關係變動、額外漸層與邊緣透明噪點。
外框相同不證明內部色區或造型一致。用途、動作、原版合成、正常玩家路徑及公開權利未驗，未新增正式接入。

### 60.3 #21裁切色區v2改善

從原版逐列色號取得兩個區域與x=11空隙後，另一次內建image_gen只使用原版參照生成v2。
v1與其55項來源保留。新原生圖／48×48、完整提示詞、selected及三欄對照依序原版／v1／v2已保存。
六份PNG的CRC、完整解碼、RGBA／尺寸與UID通過，參照與展示RGB不符0。

| 核對項 | v1 | v2 |
|---|---|---|
| 外框偏差總和 | 4 | 0 |
| 二值輪廓差異像素 | 441 | 2 |
| 全圖16×16格中心最接近EGA色號不符 | 129 | 14 |
| 右側四欄格中心色號不符 | 26 | 0 |
| x=11欄內不透明像素 | 87 | 0 |

格中心色號以候選48×48的每個3×3格中心對原版色盤計最近色，alpha<128視為空隙。
這是色區位置抽樣，不是完整RGB或美術驗收。v2右側裁切區及空隙已恢復，外框改善，
仍有兩個二值輪廓差異像素與十四個格中心色號不符，不能宣稱全部色區／造型一致。
省略右側負對照234、不透明負對照拒絕。候選改善true、正式美術false；v2列為較佳候選，未正式接入。

十八項來源1,254,646 bytes保全，含核對工具、通用工具及解碼器的精確快照。
十六張v1／v2原生複本SHA與工具輸出相同；v1來源55項及前批ENEMY08的57項未改寫。
本批完成後小圖塊候選150/180，剩餘ENEMY10–11共30圖號；正式ENEMY美術0/360、主題20筆不變。
特效024 §1.5維持DRAFT，全部sprite、美術修整、原版DAT與正式封包仍待完成。

## 61. ENEMY10小圖塊來源與候選

來源為ENEMY10.PBL #15–#29十五個16×16圖塊，PBL SHA-256：
`d371d4065a76a374329128c6d4a35f52f578322f7f2e4458252592b26fa0a482`。
這十五張不含色盤敏感色號2／8／10；大圖#2的色盤問題不套用到本批。
位置、各色區、空隙與留白沿用原版。用途、動作、正常輸出路徑與合成仍未知。
Docker入口 `tools/hd/small_tile_assets.py prepare ENEMY10`，輸出沿用既有 `workplace/hd/redraw/`。
`ENEMY10-small-single-references-v1-20261002.json` 保存原版參照、檔案偏移、尺寸及色號雜湊，
`ENEMY10-small-single-generation-jobs-v1-20261002.json` 保存逐張完整提示詞與原版定位資料。
提示詞沿用白色／灰色不透明前景契約；依§60.3的單例改善，另列出原版四鄰接占用區外框，避免省略裁切片段。
提示詞定位工具 `workplace/hd/enemy10_prompt_geometry_v1.py`，Docker限定；原版占用格及每個四鄰接區外框／像素數存入jobs。
本批source-manifest另保存定位工具與geometry-source精確快照，通用轉檔／核對工具不改寫。
四鄰接占用區只是色號幾何，不推論物件身分；不以提示詞補強證明整批方法或美術通過。
內建image_gen逐張生成，selected記工具／專案路徑，完整畫布Lanczos轉48×48 RGBA。
verification／source-manifest及tool／decoder-source快照採同single前綴，結果產生後回填。
對照 `ENEMY10-small15-single-comparison-v1-20261002.png` 五欄六列，每兩列原版／候選。
原版及候選留本機，不覆寫舊來源；未產生的結果名稱只作接續入口。

### 61.1 前一檢查點

2026-10-02依使用者要求先同步進度。十五張原版參照及完整提示詞已準備，
#15–#20六張1254×1254、8-bit RGBA工具原生PNG存在，專案複本已保存。
六份複本SHA-256與selected記錄的工具輸出相同，UID/GID均1000:1000；未覆寫既有素材。
#21–#29九張尚待生成，48×48轉檔、整批PNG／輪廓核對、對照與完整來源保全尚未完成。
PNG標頭與複本雜湊核對不等於整批技術或美術驗收；本批不增計150/180。
現行正式ENEMY美術0/360、主題20筆與特效024 §1.5 DRAFT維持。
下一步先查看其餘參照，再生成九張；已有六張原生複本核對SHA後保留，
接續轉檔沿用§59.2的保留流程並另存ENEMY10工具，不能直接重跑拒絕覆寫的通用convert。

接續轉檔工具 `workplace/hd/enemy10_convert_resume_v1.py` 核對既有複本SHA，另存缺項並完整畫布Lanczos轉48×48。
本批來源保全另加提示詞定位工具／geometry-source與接續轉檔工具／resume-source各兩項，合計59項。
精確快照採 `ENEMY10-small-single-geometry-source-v1-20261002.py` 與
`ENEMY10-small-single-resume-source-v1-20261002.py`，保留通用verification的55項基礎files表。

### 61.2 十五張候選保存與技術核對

接續九次內建image_gen完成#21–#29，六張既存原生圖核對SHA後保留；十五張原生／48×48 RGBA均保存。
ImageMagick 6.9.11-60 Q16 x86_64採完整畫布Lanczos縮放，不裁切或平移。
Python 3.13.15獨立核對46 PNG的CRC、完整解碼、尺寸、RGBA與UID/GID通過。
原版參照及展示RGB不符0，位移負對照39,792、不透明負對照拒絕；舊ENEMY02的85份來源未變。
59項來源9,821,057 bytes已保全，包含提示詞定位工具與接續轉檔工具精確快照。
前批ENEMY09的55份與#21修整18份來源未變；十五張原生複本SHA與工具輸出相同。

外框9/15相同，二值輪廓只有#26／#28相同。#20四邊總偏差4；#23／#24／#25底界少1列；
#27上下界各下移1列；#29左界內縮2列、頂界內縮1列、右界超出1列。
#19雖外框相同，二值輪廓差276；#20差499，直條內部位置及黑色空隙未保持。
數值為48×48畫布、alpha≥128、右／下界不含邊界。

| 圖號 | 原版外框×3 | 候選外框 | 二值輪廓差異像素 |
|---|---|---|---|
| #15 | 0,9,48,39 | 0,9,48,39 | 9 |
| #16 | 0,9,48,39 | 0,9,48,39 | 20 |
| #17 | 0,9,48,39 | 0,9,48,39 | 24 |
| #18 | 0,0,48,45 | 0,0,48,45 | 3 |
| #19 | 0,0,48,45 | 0,0,48,45 | 276 |
| #20 | 3,0,45,45 | 4,1,46,44 | 499 |
| #21 | 0,6,48,48 | 0,6,48,48 | 1 |
| #22 | 0,9,48,48 | 0,9,48,48 | 18 |
| #23 | 0,3,48,42 | 0,3,48,41 | 81 |
| #24 | 0,6,48,48 | 0,6,48,47 | 75 |
| #25 | 0,6,48,45 | 0,6,48,44 | 76 |
| #26 | 0,6,48,48 | 0,6,48,48 | 0 |
| #27 | 3,27,48,42 | 3,28,48,43 | 36 |
| #28 | 0,27,48,42 | 0,27,48,42 | 0 |
| #29 | 0,21,45,42 | 2,22,46,42 | 155 |

對照仍有圓角化、密集色區改為膠囊形、額外漸層與透明邊緣噪點。
提示詞已列原版占用格與四鄰接區，仍不足以保證內部幾何；不推論整批方法改善或美術通過。
用途、動作、原版合成、正常玩家路徑及公開權利未驗，未新增正式接入。
本批完成後小圖塊候選165/180，剩餘ENEMY11十五圖號；正式ENEMY美術0/360、主題20筆不變。
特效024 §1.5維持DRAFT，其他sprite、美術修整、原版DAT與正式封包仍待完成。

## 62. ENEMY11小圖塊來源與候選

來源為ENEMY11.PBL #15–#29十五個16×16圖塊，PBL SHA-256：
`f5c29f254baf0ec1ef2dc9db61596efbd2cfb941ed992534358899386a29dc44`。
十五張不含色盤敏感色號2／8／10，位置、比例、色區與留白沿用原版，不推論物件身分、用途或動作。
Docker入口 `tools/hd/small_tile_assets.py prepare ENEMY11`，沿用既有 `workplace/hd/redraw/`。
`ENEMY11-small-single-references-v1-20261002.json` 保存原版參照、偏移、尺寸與色號雜湊，
`ENEMY11-small-single-generation-jobs-v1-20261002.json` 保存十五張完整提示詞與定位資料。
定位工具 `workplace/hd/enemy11_prompt_geometry_v1.py` 沿用§61的原版占用格與四鄰接區外框；
白色／灰色為不透明前景，只將原版黑色轉透明。§61證實提示詞不足以保證內部幾何，不宣稱方法成功。
內建image_gen逐張生成，selected記實際工具及專案路徑，完整畫布Lanczos轉48×48 RGBA。
verification／source-manifest採同single前綴；另保存定位工具與
`ENEMY11-small-single-geometry-source-v1-20261002.py` 精確快照，完整來源預計57項。
對照 `ENEMY11-small15-single-comparison-v1-20261002.png` 五欄六列，每兩列原版／候選。
全部產物留本機，不覆寫前批；尚未產生的名稱只作接續入口，結果產生後回填。

### 62.1 全180小圖塊候選的逐來源盤點入口

工具 `workplace/hd/verify_small_candidate_inventory_v1.py`，Docker限定，十二個PBL唯讀。
按ENEMY00–11各#15–#29明示選圖，獨立解碼原版圖號與候選PNG，不以glob數量作完成證據。
00 #18–#20沿用§37–39的art-in v3，00 #22用§47單格較佳候選；
06 #20與09 #21沿用§57／§60較佳v2，不覆寫各v1來源，正式美術仍false。
輸出 `ENEMY00-ENEMY11-small-inventory-v1-20261002.json` 記180個來源鍵、候選SHA、PNG技術與幾何數字，
`ENEMY00-ENEMY11-small-inventory-source-manifest-v1-20261002.json` 保存180 PNG、收據及工具／解碼器精確快照。
同inventory前綴tool-source／decoder-source保存可變工具bytes；缺一圖號及重複圖號的負對照必須拒絕。
本項只證明候選存在、檔案技術與來源幾何，不驗正常用途、動畫、原版合成或正式接入。

### 62.2 ENEMY11十五張候選保存與技術核對

十五次內建image_gen產出與完整畫布Lanczos 48×48 RGBA已保存，無裁切或平移。
ImageMagick 6.9.11-60 Q16 x86_64，Python3.13.15獨立核對46 PNG的CRC／完整解碼／尺寸／RGBA／UID通過。
原版參照與展示RGB不符0，位移負對照33,216、不透明負對照拒絕；既有ENEMY02的85份來源未變。
57項來源10,056,861 bytes已保全，含定位工具精確快照；前批ENEMY10的59項SHA未變，十五張原生複本與工具SHA相同。

外框8/15相同，二值輪廓僅#22相同。#16上下各內縮2列；#19／#25／#28底界少1列；
#20右界超出2列；#23左界內縮2列、右界超出1列；#26左／上／下界各內縮1列。
數值為48×48畫布、alpha≥128、右／下界不含邊界。

| 圖號 | 原版外框×3 | 候選外框 | 二值輪廓差異像素 |
|---|---|---|---|
| #15 | 0,3,48,48 | 0,3,48,48 | 1 |
| #16 | 0,3,48,48 | 0,5,48,46 | 192 |
| #17 | 0,3,48,48 | 0,3,48,48 | 1 |
| #18 | 0,18,48,39 | 0,18,48,39 | 36 |
| #19 | 0,18,48,39 | 0,18,48,38 | 56 |
| #20 | 0,12,45,39 | 0,12,47,39 | 62 |
| #21 | 12,6,48,45 | 12,6,48,45 | 27 |
| #22 | 0,9,48,42 | 0,9,48,42 | 0 |
| #23 | 0,6,45,42 | 2,6,46,42 | 36 |
| #24 | 0,3,48,48 | 0,3,48,48 | 16 |
| #25 | 0,0,48,45 | 0,0,48,44 | 75 |
| #26 | 0,0,48,48 | 1,1,48,47 | 147 |
| #27 | 0,0,48,39 | 0,0,48,39 | 73 |
| #28 | 0,0,48,39 | 0,0,48,38 | 132 |
| #29 | 0,0,48,36 | 0,0,48,36 | 52 |

對照仍有圓角化、分離色區被連成曲線、內部色區與粗細變動、額外漸層與透明邊緣噪點。
完整提示詞定位不足以保證內部幾何；造型、動畫、原版合成及正常玩家路徑未驗，未新增正式接入。

### 62.3 全180候選逐圖號盤點結果

十二個原版PBL各30張，#15–#29全部16×16；180個來源鍵均有唯一明示候選檔，
180 PNG的CRC、完整解碼、48×48 RGBA、透明及可見通道、UID核對通過。
缺一圖號及重複圖號負對照均拒絕；不以glob數量或批次相加證明候選完整。

全180候選外框88/180相同，二值輪廓26/180相同。92個外框有差異，
其餘外框相同也不代表內部色區或動作關係通過。00 #18–#20沿用既有特效v3，
00 #22／06 #20／09 #21使用已記錄較佳候選，仍非正式採用；各v1與舊來源不改寫。
新185項來源1,013,615 bytes保全，含180候選、收據及工具／解碼器精確快照，十二個原版只記SHA。
完整逐圖號檔案映射、原版檔案偏移、解碼SHA、候選SHA與幾何差異在§62.1所列inventory收據。

本批完成後小圖塊候選180/180，候選製作無缺圖號；正式ENEMY美術0/360、主題20筆維持。
全部sprite正式接入、形狀修整、原版DAT正常存讀檔、四檢查點、幀率與正式封包仍待完成。
特效024 §1.5 DRAFT維持，不把候選齊備當作完整HD完成；後續先補現行HD主題的原版DAT正常存讀檔。

## 63. 現行HD主題的原版DAT正常存讀檔

驗證入口 `tools/hd/verify_dat_runtime.go`，研究編譯覆映射沿用§49的正常FIFO包裝。
現行原始收據為本機 `workplace/hd/dat-runtime-v6-20261002/verification.json`，
獨立核對入口 `tools/hd/verify_dat_runtime.py`，收據 `workplace/hd/dat-independent-v1-20261002.json`。
來源保全入口 `tools/hd/preserve_dat_runtime.py` 與本機 `workplace/hd/dat-source-manifest-v1-20261002.json`。

範圍為既有重播的10-saved／11-loaded／17-saved2／18-loaded2四段，
比較原版無覆繪、已載主題但關閉、開啟主題三條分支；兩段讀檔均重建執行器，
從05-select正常LOAD GAME輸入檔名，不以F10／F11替代DAT。
起點為已驗原版state及正常重播中文Layer，原版seed由固定state保留，沒有修改RAM或判定。
原版資料夾沒有TEST.DAT或TEST2.DAT，各分支使用新可寫暫存層。

依024 §6.2核對正常取樣、結束步數、cycles、原版畫面、CPU、RAM、IRQ1及DAT bytes。
中途RAM限A0000h以下；完整匯流排只在終點經共同畫面讀取後比較，沿用§49–51的位址範圍限制。
既有獨立原版state及52 bytes玩家資料作讀回期望，另要求HD實際有圖面、中文字面不受HD改變。
這是保存正常state接續的原版選單流程，仍須另做正式GUI、封包、全部sprite與美術驗收。

### 63.1 證據與測試輸入

【confirmed，有限正常重播】現行二十筆主題無新增正式接入。原版EXE SHA-256為
`88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49`，
Go1.24.13、Python3.13.15，沿用dosgolem基準f8c1a6e及既有專用工具映像。
固定state及seed先於執行確定；Samar存檔A71Dh、Sivad存檔F95Bh、兩段主選單讀檔86AFh。
SELECT中文起始Layer由04-cleared的既有正常空白鍵重畫，CPU、steps、cycles、原版畫面及
A0000h以下RAM與05-select.state相同，不人工猜補譯文。

磁碟state未保存IRQ1Delivered，與舊磁碟state對照不比較該計數；本輪三分支間完整比較。
原版重播單鍵的key_every=0表示probe省略旗標，研究工具沿用probe的500,000步預設值。
首版工具沒有還原這兩項契約，失敗來源與二進位保存，未改正式遊戲輸入。
主機落地檔名為test.dat／test2.dat，依既有DOS寫入服務辨識不分大小寫的唯一檔案；
第二版大寫路徑失敗亦保存，沒有變更原版檔名或存檔格式。

第三個阻塞是檔案修改時間。未固定時間的原版11-loaded與舊state只有線性
`0x1096`、`0x1097`、`0x1098`三個RAM bytes不同，CPU、steps、cycles、畫面及DAT bytes相同。
來源為dosgolem `internal/dos/find.go` 的emitFind，PSP DTA基址`0x1080`，時間偏移`+0x16`、
日期偏移`+0x18`。這些是執行期線性位址及DTA結構偏移，沒有覆寫原始遊戲定位資訊。
新測試DAT建立後，僅以os.Chtimes保存既有獨立參考檔的主機修改時間：
test.dat為2026-09-18T01:10:06.183146Z，test2.dat為2026-09-18T01:10:47.388458Z。
參考檔bytes／SHA及時間均記在收據；原版RAM、seed、EXE與參考DAT均未改寫，沒有排除DTA位元組。
未固定時間的v4／v5與RAM差異收據保留為負對照，不列為產品缺陷。

### 63.2 正常重播與獨立核對結果

| 段落 | 每分支取樣 | 結束steps | 結束cycles | 本段IRQ1 | DAT |
|---|---:|---:|---:|---:|---|
| 10-saved | 14 | 141000000 | 661741934 | 26 | test.dat，512 bytes |
| 11-loaded | 8 | 57000000 | 248543493 | 14 | 讀回test.dat |
| 17-saved2 | 17 | 655000000 | 3202734321 | 32 | test2.dat，512 bytes |
| 18-loaded2 | 9 | 58000000 | 253613469 | 16 | 讀回test2.dat |

原版／關閉／開啟三分支共12段、144取樣，每分支48取樣。每個取樣CPU、steps、cycles、
IRQ1、原版畫面及A0000h以下RAM一致；十二個完整終點匯流排一致。HD關閉／開啟兩側中文字面
逐bytes相同，四段開啟分支各有1／8／3／9個實際HD圖面取樣，未通過空測。
兩份DAT逐bytes與既有獨立原版DAT相同，三分支也相同：
test.dat SHA-256 `7c4ee8ff91822d6022877d21ff390c067039434d50f8a38d04b226656a9d49c2`；
test2.dat SHA-256 `781da2d6bc693fe370061b74d1cdefaf1472fd8c217e711908aaab36adb43acc`。

Python獨立讀取144份原版frame及十二份終點52 bytes玩家資料，對既有原版frame／mem不符0；
十二份PNG的CRC、IEND、尺寸及完整解壓通過，兩張HD讀檔終點已目視。
變更DAT一byte的負對照拒絕，未固定檔案時間的三個DTA差異亦獨立確認。
PNG並未獨立重建整屏HD合成，不能由此宣稱GUI或新整屏美術通過。

024 §6.2在上述正常原版DAT重播範圍通過；024仍READY，完整sprite、美術、四檢查點、
幀率、正式GUI存讀檔及實際封包仍待完成。候選180/180、正式ENEMY美術0/360及特效§1.5 DRAFT維持。

來源保全630項、55,084,771 bytes，含實際DAT與修改時間、原版state、失敗版本、工具與精確
程式／文件快照；原版輸入只記SHA，前批全180候選185份來源未變，全部留本機。
進度文件以本次source-manifest的snapshot回查，不以後續文字改動取代實際執行時的程式bytes。

### 63.3 重生入口

Docker內以Go工作區 `workplace/hd/dat-runtime-v1-20261002.go.work` 同時載入本專案與dosgolem。
`dat-runtime-overlay-v2-20261002.json` 將probe main映射到研究工具，並只在編譯覆映射排除
probe的observe.go；Oracle bridge沿用§49。不在正式API增加函式，不修改原probe。
由dosgolem模組建置整個probe包才能符合Go internal邊界，單檔建置及混入原probe輔助檔的
環境失敗不列為遊戲缺陷；舊覆映射、試驗複本及二進位保留。

```sh
# /src/worktrees/dosgolem，已載入兩模組的GOWORK
go build -overlay /src/workplace/hd/dat-runtime-overlay-v2-20261002.json -o /src/workplace/hd/dat-runtime-v6-20261002.bin ./cmd/probe
# /src，原版/orig唯讀；所有工具拒絕覆寫既有輸出
workplace/hd/dat-runtime-v6-20261002.bin -out workplace/hd/dat-runtime-v6-20261002
python tools/hd/verify_dat_runtime.py
python tools/hd/preserve_dat_runtime.py
```

重生使用乾淨研究輸出或精確來源快照；既有收據只核對，不覆寫。

## 64. 四個既有檢查點未選主題回歸

對應024 §6.1，檢查點為01-title／03-protection／05-select／07-first-play，原版與中文共八組。
正常中文輸入入口 `tools/hd/prepare_checkpoint_layers.go`；本機輸入目錄
`workplace/hd/checkpoints-input-v1-20261002/`，內含未改bytes的原版state及正常鍵序重建的中文Layer。
執行期比較工具 `tools/hd/checkpoints_run.py`，獨立核對 `tools/hd/verify_checkpoints.py`，
現行輸出 `workplace/hd/checkpoints-runtime-v1-20261002/` 與 `checkpoints-verification-v1-20261002.json`。
本節已完成下述有限回歸；不改既有8×8、人物／框線排版或sprite範圍。

加入HD前的逐步工具main來源取自§33的 `source-before-theme-20261001/manifest.json`，
該不可變來源只覆映射main，兩側共用目前dosgolem／翻譯器／字型及文本，隔離HD接合差異。
這個控制版本不是整個歷史工具鏈重建，也不由舊二進位的未知依賴推出目前完整回歸。
正常中文Layer由01-title依既有六段鍵序推進至07-first-play，每段核對既有原版state；
避免只載原版state、中文分支卻沒有先前印字而形成空測。標題Logo依既有定案保留。
逐步工具採wait:1且保存實際終點，獨立匯出原版索引／RGB與中文Layer後核對完整合成，
另以真正選用二十筆HD主題的初始迷宮作敏感性負對照；GUI、封包及幀率另驗。

### 64.1 正常中文起點與控制來源

【confirmed，有限正常流程】Go1.24.13、生成／執行工具Python3.11.2，獨立核對Python3.13.15；
沿用既有psychicwar-go-ebiten映像，開工load18.742，一CPU離線容器，不作即時性能聲明。
PW.EXE及四個state的完整SHA列於prepare的verification.json；原版輸入唯讀，沒有修改seed／RAM。
02-match至07-first-play六段正常FIFO鍵序，每段CPU、steps、cycles、畫面及A0000h以下RAM
與既有獨立原版state相同，所有中文Layer由實際印字產生；四份供逐步工具的state完整bytes不變。
標題沒有要替換的動態中文字面，Logo依既有定案保留原版。

控制main原始SHA-256 `217f6594432d12063292601ef6696cd679d79029a8310f51fe7564e77c71aad7`，
取自加入HD前保存的不可變來源。控制與現行binary都本輪重新建置，共用現行底層依賴，
控制沒有theme旗標，現行明示`-theme ""`。與舊§34的兩份post-HD二進位對照分開，
本輪不是完整歷史底層恢復，也不把共同依賴的問題當作已被回歸排除。

### 64.2 實際逐步輸出與獨立像素核對

| 檢查點 | 每語言兩版本終點steps | cycles | 原版／中文畫面差 | 中文省略負對照 |
|---|---:|---:|---:|---:|
| 01-title | 12000750 | 43405996 | 各0 | 0，標題Logo保留 |
| 03-protection | 18000750 | 72734960 | 各0 | 3779 |
| 05-select | 32000750 | 130240901 | 各0 | 8534 |
| 07-first-play | 42000750 | 178720659 | 各0 | 6635 |

四個檢查點、兩語言、兩版本共16份實際PNG；每組CPU、steps、cycles、seed、原版色號、
RGB、A0000h以下RAM與完整終點匯流排一致，中文字面快照bytes相同。
無副作用的state匯出器保存320×200色號與RGB，Python由原版RGB、正式GOLEMFNT與中文快照
獨立重建960×600完整畫面，16份合成不符0，沒有排除游標或容許差異。
防拷、主選單與迷宮中文均有實際中文字模，三張現行中文PNG已目視，沒有把空中文層算通過。

第17份為真正選用二十筆主題的初始迷宮中文：與未選主題差143971個放大像素，
原版state及中文字面仍相同，作敏感性負對照；未在本批獨立重建這張HD圖面。
另外變更一個RGBA通道的受控比較差1像素，確保比較器會拒絕單像素錯誤。
獨立收據 `workplace/hd/checkpoints-verification-v1-20261002.json`，實際執行argv、二進位SHA及
17份原版終點資料在runtime目錄的execution.json／original-frames.json。

024 §6.1在此限定逐步工具回歸範圍通過。GUI、完整歷史工具鏈、實際封包、幀率、
全部sprite、美術與公開權利仍待完成，024仍READY、特效§1.5仍DRAFT，#34不關閉。
下一步回到其他sprite的正常來源與接入，不重跑已通過的同一四檢查點；高負載不補幀率聲明。

### 64.3 保全與重生入口

保全工具 `tools/hd/preserve_checkpoints.py`，本機索引
`workplace/hd/checkpoints-source-manifest-v1-20261002.json`；原版檔只記SHA，全部留本機。
新444項來源共30,467,341 bytes，快照SHA及UID/GID1000:1000逐項核對相符；
前批DAT的630項來源55,084,771 bytes未變，未覆寫既有來源快照。
Go正常Layer工具沿用§63的兩模組GOWORK，probe覆映射入口為checkpoints-layer-overlay，
控制main覆映射為checkpoints-control-overlay。Docker `/src`內先建置四個二進位：

```sh
go build -C worktrees/dosgolem -overlay /src/workplace/hd/checkpoints-layer-overlay-v1-20261002.json -o /src/workplace/hd/checkpoints-layer-v1-20261002.bin ./cmd/probe
go build -o workplace/hd/pwstep-checkpoints-current-v1-20261002 ./cmd/pwstep
go build -overlay workplace/hd/checkpoints-control-overlay-v1-20261002.json -o workplace/hd/pwstep-checkpoints-control-v1-20261002 ./cmd/pwstep
go build -o workplace/hd/export-checkpoints-v1-20261002 tools/hd/export_over_frontend.go
workplace/hd/checkpoints-layer-v1-20261002.bin -out workplace/hd/checkpoints-input-v1-20261002
python3 tools/hd/checkpoints_run.py
python tools/hd/verify_checkpoints.py
python tools/hd/preserve_checkpoints.py
```

逐步執行用含Python3.11.2的既有Go映像，獨立核對／保全用Python3.13-alpine。
重生使用新的研究輸出或精確來源快照，既有原版與輸出不覆寫；這些路徑不是發行產物。

## 65. 正常治療路線的房間貼圖來源

【confirmed，有限正常來源】唯讀探針入口 `tools/hd/observe_room.go`，由既有正常
`12-battle2.state` 接續九個原版FIFO按鍵到 `13-healed.state`，不改RAM、seed或規則。
現行輸出前綴 `workplace/hd/room-normal-v2-20261002`。一般打包貼圖入口
CS:IP `0161:8705`／`0161:8751` 在本條路線0筆；實際走串流RLE入口
`0161:8588`，由原版near-return的SS:SP核對返回，記錄DS:BX、CX座標、
原始RLE與前後畫面，再對所有實際PBL完整解碼結果核對。
上限256次貼圖，與無觀察同鍵序終點及既有原版state比較；未驗HD或正式接入。
獨立核對入口 `tools/hd/verify_room.py`，由實際RLE bytes、原始PBL、保存畫面及
獨立舊終點重算來源與座標，收據 `workplace/hd/room-normal-independent-v1-20261002.json`。

### 65.1 實際來源與座標

| 項目 | 本條正常路線實測 |
|---|---|
| 固定起點 | 12-battle2.state，SHA94912dee5881b8f7b878fed44465a7afdbbe47d2affef5a19226a898e5f60993、seed A48Ch |
| 來源 | ROOM0.PBL #8，26檔537張唯一完整解碼匹配 |
| 原始檔位置 | 檔案偏移0x3C64，實際RLE及標頭1965 bytes與執行期bytes相同 |
| 原版座標／尺寸 | (4,124)、72×72，CX011Fh；DS:BX0161:92BE，執行期線性0xA8CE |
| 繪製步數 | 322515156進入、322911294返回 |
| 獨立核對 | 原版返回畫面及區外不符0，治療終點仍完整匹配 |
| 負對照 | X偏一像素1865，改成ROOM0 #7差4395，單像素1 |

九個FIFO鍵實際交付18次IRQ1。與無觀察同路線及既有13-healed原版終點的CPU、steps、
cycles、原版畫面、A0000h以下RAM及完整終點匯流排一致；磁碟state未保存IRQ1計數，
此欄只在兩條實際執行間比較。終點345000000步、1670272138 cycles；位置Samar(9,2)、HP40／能量30。
獨立Python只重算PBL來源、座標及保存畫面，不把原版探針的CPU／RAM比較當作獨立重算。

Go1.24.13、Python3.13.15，既有一核心離線Docker；開工load12.356，不作幀率／音訊／時序聲明。
初版只觀察打包貼圖，0筆而退出；原始來源與二進位另存，未覆寫。參照既有023的串流RLE路徑
及組語定位線索後補觀察入口，返回畫面與原始RLE已直接驗證。這是探針範圍修正，未改正式產品。
原版色號參照圖已目視，既有216×216 ROOM0-08候選也已查看；未在本批驗收候選造型或完整HD合成。

### 65.2 接入邊界與保全入口

(4,124)未落在原點(0,0)的8×8格界。後續限定接入必須保留原座標、部分邊界格及中文／框線順序，
不將位置取整，不自行將貼圖原點當成新格網原點。先核對現行圖面契約，再審查024的限定READY契約。
正常離房清除、繪製中途、主題開關、載回重建、HD圖面合成、美術及GUI仍待驗，現行主題20筆不變。
保全工具 `tools/hd/preserve_room.py`，索引 `workplace/hd/room-normal-source-manifest-v1-20261002.json`；
精確來源、原版state、失敗首版及既有候選均留本機，原版PBL／EXE只記SHA，不上傳。
新249項來源12,592,054 bytes，前批四檢查點的444項快照未變。ROOM0.log不存在，
保全在預檢階段停止後改收實際存在的檔案；首次保全工具也保存。未虛構當時生成紀錄，
保存既有候選、參照及製作spec不等於完整美術來源或散布權利已驗。

## 66. 治療房間的8×8邊界與限定接入

本節進行中，依§65已證實的ROOM0 #8、72×72與(4,124)核對既有204補齊方式。
邊界核對入口 `tools/hd/verify_room_grid.py`；原點(0,0)的覆蓋格範圍為
(0,120)至(80,200)，十列十欄，原圖外每邊四像素只補透明，不平移圖案。
正常離房先用現行 `cmd/probe` 從13-healed接續14-minton1的首個Up，
345500000步送出、3000000步鍵距，351000000步止；先確認原版圖像是否清除。
本輪不依地圖座標猜圖像生命週期，契約未READY前不修改正式主題載入器。

兩張原版完整房間各100格與SCREEN／MENU＋ROOM基準一致，36個邊界格；
外緣(0,120)一像素改動仍保有完整原圖，但有效格降至99，定位在原點(0,0)格網。
正常首鍵Up的原版probe已執行：345000000及346000000步完整，349000000步與房間原圖差4180。
probe於351000000上限停止，該上限的shots／save-state未執行，已保存三張實際畫面及argv／工具SHA；
不把未產生的終點state當產品缺陷。後續正常接入驗收自行保存實際終點並比較。
由來源、邊界及正常原版清除證據完成審查，024 §1.9 READY限定契約已載入；美術與GUI仍未驗。
實作前規格精確bytes在 `workplace/hd/room-ready-spec-v1-20261003.txt`；
接入前theme.go沿用§65來源快照，SHA12654ca980ccc218305ac5641e45d8cb5b2689202f93fcfb0fffe201f7774aa1。
主題準備入口 `tools/hd/prepare_room_theme.py`，正常接入驗收入口
`tools/hd/verify_room_runtime.go`，本機新主題 `workplace/hd/theme-room-v1-20261003/`。

### 66.1 本次進度核對

【confirmed，限定載入器與單元回歸】ROOM0 #8載入器與房間測試已實作，保留原座標與尺寸。
新主題21筆由既有20筆加ROOM0 #8候選組成，來源複製清單為該目錄的source-copies.json。
主題測試15主測試／29含子案例通過，失敗／略過0；收據
`workplace/hd/room-unit-v2-20261003.jsonl`。含真實ROOM0版本、格邊界、透明補齊、
單像素失配清除／恢復、開關／重登記、錯圖號／位置／裁切／錨點／來源及重複列拒絕。

首輪的房間測試均通過，既有TestThemeRealBackground因未掛載獨立bg.idx而失敗。
補上既有/hd唯讀掛載後，同一工具鏈完整重跑通過；首輪room-unit-v1收據保留。
主題準備程序已完成21筆與編譯覆映射，之後讀取首輪失敗結果而退出，現存輸出不覆寫。

該次進度同步時正常驗收工具只寫完，尚未建置／執行，因此已驗證主題仍20筆。
後續正常／載回／pwstep與獨立完整核對見§66.2–3，現行21筆主題在該有限範圍通過。
不把單元測試算成正常玩家流程或美術完成。
024 §1.9仍READY、§1.5 DRAFT，完整sprite、GUI、幀率、美術及封包仍待完成。

### 66.2 正常HD與載回驗收入口

正常工具 `tools/hd/verify_room_runtime.go` 已建置並執行，實際收據
`workplace/hd/room-runtime-v1-20261003.json`、執行紀錄room-normal-run-v1-20261003-execution.json。
十三個樣本的原版／HD關閉／HD開啟狀態一致、中文字面相同，十鍵交付20次IRQ1。
房間完整出現四個樣本，ALLY #0在十三個樣本均完整匹配；獨立像素結果見§66.3。

補驗入口：`tools/hd/verify_room_reload.go` 載回四份實際正常state並等步數接續；
`tools/hd/room_pwstep_run.py` 使用本輪新建置的正式pwstep與原版匯出器；
`tools/hd/verify_room_render.py` 由原版PBL、候選PNG及正式字型重算所有21筆圖面及合成，
不排除盟友或敵人區域。這些工具與後續收據均留本機研究範圍，美術及GUI另驗。

### 66.3 獨立結果與保全入口

【confirmed，有限正常接續與逐步工具】獨立收據
`workplace/hd/room-render-verification-v2-20261003.json` 已通過十三個正常樣本、
四份真實state載回前後八個圖面與八份實際pwstep完整合成；沒有排除角色區域。
四個完整房間樣本的省略負對照各43349像素，349000000及351000000步保留房間的負對照各288。
原版終點351000000步、1700303068 cycles、20次IRQ1，三分支一致；四份載回各100000步亦相同。
四則治療訊息直接對正式text/I_MENU00.BIN.json不符0，正常起點Layer對既有串接SHA相同；
收據room-provenance-verification-v1-20261003.json。房間完整／清除兩張中文HD圖已目視。

首版獨立工具把原圖完整與顯示錨點混為同一條件，332000000及345000000步的ALLY原圖完整，
但背景錨點與基準差588像素。保存首版工具後修正分類，以同一來源完整重跑；像素門檻未放寬。
這兩個治療選單樣本依既有契約回退SCREEN／MENU與盟友HD，ROOM0 #8仍正常顯示。
這是完整HD玩家體驗的剩餘限制；後續需依原版來源證據審查動態選單的背景錨點，不能直接改矩形猜補。
差異定位收據room-anchor-diagnostic-v1-20261003.json：兩個樣本差異邊界為(168,5)至(247,40)，
原版MENU矩形以外差0，左側160×40差0；這只支持後續審查，不授權新錨點契約。
新21筆主題只在上述有限接續／載回／逐步範圍驗證通過，不宣稱整屏處處HD或美術完成。

來源保全入口 `tools/hd/preserve_room_runtime.py`，本機索引
`workplace/hd/room-runtime-source-manifest-v1-20261003.json`，快照目錄source-room-runtime-v1-20261003。
沿用§65及文件職責的workplace研究目錄；預檢全部來源、SHA與UID後才建立新目錄，拒絕覆寫。
原版PBL／EXE只記雜湊，正常state、候選、二進位與精確Go來源留本機；不推送素材或發行。
本批539項來源120421754 bytes已保存，前批§65的249項快照未變。進度文件後續回填來源數字與遠端時間，
實際驗收時的工具／程式以manifest的不可變快照回查，不以後改的現況文件替代。

## 67. 動態選單與背景錨點

有限驗證通過，完整HD仍未完成。§66的兩個治療樣本已定位588個改畫像素，全部在原版MENU範圍。
先核對候選穩定矩形(0,0,160,40)對既有正常原版畫面、標題／防拷／主選單與OVER的辨識，
保留(0,0,320,40)舊契約及逐格不可變Reference；不改原版位置、資產或8×8格。
只補顯示辨識；證據審查與READY保存完成後才改正式載入器。證據入口 `tools/hd/verify_background_anchor.py`，
本機收據background-anchor-source-v1-20261003.json；後續契約見024 §1.10。

【confirmed，有限原版保存畫面】Go未推進本批原版，Python3.13.15獨立解碼SCREEN／MENU。
十九個既有原版檢查點、ENEMY00正常43樣本、Sivad17、OVER21與房間13，共113份原版frame。
左側160×40匹配105，舊320×40匹配102，新增三份皆為完整治療畫面；各588差異全位於MENU。
其餘105個匹配均包含已證實背景。八份未匹配為標題、防拷階段、OVER與Sivad結束；
SELECT與輸入名字在舊錨點原本也匹配，不新增未證實畫面類型。
首版誤要求SELECT失配，探針與診斷另存後修正假設，沒有改正式程式或放寬像素判斷。
原版單像素負對照有效；依此與§66的逐格不可變基準完成024 §1.10 READY審查。
READY只授權明示左側錨點，省略與舊全寬預設保持；格網、座標、角色來源與未知回退保持。
接入前theme.go SHA3c28f056f42e358f29da9b6cbda3598cbb63e2056ca4ce708102f4ebf84e9423與§66快照相同，
實作前READY規格保存為background-anchor-ready-spec-v1-20261003.txt。
後續入口 `tools/hd/verify_anchor_runtime.go`、`tools/hd/verify_anchor_render.py`、
`tools/hd/anchor_pwstep_run.py`、`tools/hd/verify_anchor_reload.go`；本機新主題theme-room-anchor-v1-20261003，舊21筆保留。
保存幀回歸入口 `tools/hd/prepare_anchor_saved.py` 與 `tools/hd/verify_anchor_saved.go`，
從十九原版檢查點與四組已支援敵人完整原圖選出可回查state，不以內部身份旗標選答案。
所有圖面仍由verify_anchor_render.py獨立核對；這項與正常玩家鍵序驗收分開。
本批精確來源保全入口 `tools/hd/preserve_anchor_runtime.py`，輸出於既有workplace/hd，
保留來源snapshot、候選、狀態及二進位；原版EXE／PBL僅記SHA，不複製進公開輸出。

限定實作與執行進度：明示左錨點載入器及新21筆主題已建立，全部PNG與舊主題相同。
18主測試／32含子案例通過，失敗／略過0；收據background-anchor-unit-v1-20261003.jsonl。
正常13樣本、四份載回與八份實際pwstep已執行，三項命令結束碼0；入口background-anchor-execution-v1-20261003.json。
十九原版檢查點與四組完整敵人來源去重為22份保存狀態，計畫background-anchor-saved-input-v1-20261003.json。
獨立整屏圖面／字型／合成與保存狀態回歸已完成，有限範圍結果如下。
舊21筆房間主題與§66來源保全維持，完整sprite、美術、GUI、幀率及正式交付仍待完成。

【confirmed，限定正常流程與保存狀態】Go1.24.13／Python3.13.15，原版位址空間與§66相同。
13正常樣本原版CPU、steps、cycles、IRQ1、RAM／frame及中文快照與舊批相同；中文字型與完整合成不符0。
332000000與345000000步各新增448704放大圖面像素，背景與完整ALLY #0恢復HD；其他11正常樣本圖面與舊批相同。
四份載回前後八圖面、八份真正pwstep完整合成均獨立不符0，未排除任何角色區域。
十九原版檢查點與四組已支援完整敵人來源去重22份實際state載回，原版frame對state相同，開關／重建及原版CPU／RAM不變。
22份獨立全21筆圖面不符0；四組敵人省略負對照6336／6912／6912／6912像素有效。
獨立收據room-anchor-render-verification-v1-20261003.json，保存狀態原始收據background-anchor-saved-v1-20261003.json。
345000000步完整中文HD圖已目視，框架及人物在原版位置，治療選單與房間中文仍可見；此為有限畫面觀察，非美術或全部GUI驗收。
不因此宣稱特效重疊、其他角色／姿勢、房間／迷宮視野、美術、幀率或正式封包完成。

本批來源保全721項、210578919 bytes，前批539份快照逐項SHA未變。
入口background-anchor-source-manifest-v1-20261003.json；manifest保存收尾補記前文件bytes，歷史以snapshot回查。
GitHub #34限定結果已同步並全文回讀，2026-10-02T17:57:04Z、仍OPEN，未改#44。

## 68. 交通路線與其他房間來源

進行中。十九個既有正常檢查點的獨立ROOM0／ROOM1解碼，除了已接入的ROOM0 #8，
還在16-sivad／19-zellwal找到ROOM0 #2，在17-saved2／18-loaded2找到ROOM0 #22。
以上完整原圖均在(4,124)、72×72；保存畫面匹配不單獨證明呼叫來源或正常切換時機。
本輪先追既有15-minton2→16-sivad的正常九鍵，入口 `tools/hd/observe_transport_rooms.go`，
原版RLE與返回畫面的獨立核對入口 `tools/hd/verify_transport_rooms.py`。
來源選擇不改原版RAM／seed／路線，也不因找到候選PNG直接擴充正式載入器。
產物留既有workplace/hd；來源保全入口 `tools/hd/preserve_transport_rooms.py`。
格網及候選技術入口 `tools/hd/verify_transport_room_grid.py`；限定實作負對照與生命週期在
`apps/psychicwar/theme/transport_room_test.go`，後續驗收不以單元測試替代正常玩家路線。
新24筆主題準備入口 `tools/hd/prepare_transport_room_theme.py`，原21筆主題保留。

【confirmed，限定正常交通來源】固定15-minton2.state、F95Bh與既有九鍵，18次IRQ1。
0161:8588、DS:BX0161:92BE，原版ROOM0 #0→#3→#2三次完整RLE；72×72、(4,124)。
26檔537圖獨立唯一匹配，來源RLE／返回原圖／矩形外不符0；位移負對照1595／1153／1445，錯圖號4734／4275／4877，單像素1。
終點614000000步、2995871801 cycles，觀察／無觀察／既有16-sivad的CPU、cycles、RAM與原版frame相同。
首版／第二版將交通MENU也假設為完整原始PBL複製，70像素不符；原始bytes、畫面與精確失敗來源保留。
第三版限定原版房間矩形，MENU呼叫另列unknown，房間像素門檻未放寬；不宣稱該MENU模式已解出。
獨立核對首版把圖表範圍與RLE讀取範圍混用。#3宣告1575 bytes，最後literal需要讀下一byte；
按既有RLE格式獨立重算1576 bytes，禁止不足零填充，原始檔bytes與source全部相同，兩版核對器保留。
來源收據transport-room-source-v3-20261003.json、獨立transport-room-independent-v2-20261003.json。
三張返回原圖各100格吻合／36邊界，外緣負對照99；三候選216×216完整解碼通過，收據transport-room-grid-v1-20261003.json。
DRAFT審查後024 §1.11轉READY，實作前完整規格與前版theme.go SHA保存；限定擴充#0／#2／#3及每圖號去重。
20主測試／34含子案例通過，失敗／略過0；收據transport-room-unit-v2-20261003.jsonl，舊#8／角色／背景回歸維持。
首版測試把舊圖不可見快取列當作仍顯示，保存後改核對完整輸出像素，未因假失敗再改產品；新測試涵蓋四房間切換及恢復、清空、開關／重建、非法圖號與重複列。
新本機theme-transport-room-v1-20261003共24筆，全部PNG bytes複製來源，舊21筆保留。
正常交通三分支HD／中文整屏、實際state載回接續與兩語言pwstep尚未建置／執行；不得用原版來源及單元綠色代替這些驗收。
ROOM0 #22仍只有保存畫面匹配；全部房間／sprite、美術、GUI、幀率及交付未完成。

- 交通房間來源與限定實作保全370項、20616341 bytes，前批721份snapshot未變；入口transport-room-source-manifest-v1-20261003.json。新24筆正常HD驗收仍待完成，來源manifest保存收尾補記前文件bytes。

失敗版本由來源manifest回查：observe-transport-rooms-source-v1-20261003.go／source-v2、
verify-transport-rooms-source-v1-20261003.py與transport-room-test-source-v1-20261003.go，皆在workplace/hd。
原版MENU模式未知，未進一步逆向；正常三房間來源證據已足夠限定接入。
本批GitHub #34全文回讀相符，2026-10-02T18:29:55Z、OPEN；現況文件補記不覆寫snapshot。

## 69. 交通房間HD正常呈現與接續驗證

限定驗證通過。依024 §1.11 READY及§68的三張房間來源，接續15-minton2的既有正常中文Layer，
先核對前批Layer與原版state來源，再走相同九鍵交通及到達後首個Up。
正常原版／HD關閉／HD開啟三分支，來源、原版狀態、中文字面與完整覆繪分開核對。
入口 `tools/hd/verify_transport_room_runtime.go`、`tools/hd/verify_transport_room_reload.go`、
`tools/hd/transport_room_pwstep_run.py`、`tools/hd/verify_transport_room_render.py`；
本批精確來源保全入口 `tools/hd/preserve_transport_runtime.py`。
新產物使用transport-room-runtime前綴，舊24筆準備與21筆驗證收據保留。

### 69.1 有限驗證結果

- 正常21取樣的三分支CPU、steps、cycles、IRQ1、原版frame及A0000h以下RAM一致；完整終點匯流排相同。HD兩側中文快照與字面相同，614000000步對既有16-sivad相符。十鍵20次IRQ1，最終620000000步、3025905141 cycles；原版seed F95Bh、RAM、EXE與規則未改。
- 獨立讀取24筆素材與原版七個PBL、正式GOLEMFNT及六份文本，全部圖面／字型／整屏合成不符0。20個實際中文來源鍵對正式譯文不符0，改錯I_MENUH:0850的負對照失敗1筆。未排除角色區域或游標。
- ROOM0 #0／#3／#2完整來源樣本各4／5／4個。房間省略負對照43491／26835／46612像素；617000000及620000000步已清除，故意保留前張房間負對照46656像素。
- 四份本批真實state載回／開關／重建及各100000步接續的原版狀態相同，前後八份圖面獨立不符0。三房間與清除狀態的兩語言、HD兩側共16份實際pwstep，原版終點、中文快照及完整合成不符0。
- 三張交通及清除的實際中文HD PNG已目視；保留原版位置與人物／框線順序。這是有限技術驗證，完整美術、其他sprite／房間、GUI、幀率與封包仍待完成。ROOM0 #22未接入，特效§1.5仍DRAFT。
- 執行收據 `workplace/hd/transport-runtime-execution-v1-20261003.json`、`transport-followup-execution-v1-20261003.json`；正常／載回主收據 `transport-room-runtime-v1-20261003.json`、`transport-room-reload-v1-20261003.json`，獨立完整核對 `transport-room-render-verification-v1-20261003.json`。16份實際逐步輸出及argv在 `transport-room-pwstep-v1-20261003/`，均在workplace/hd。
- 建置準備首輪研究Go檔遺留區塊使gofmt失敗，尚未建置或執行原版；修正同工具鏈後通過，沒有正式產品修正。精確來源保全工具見本節入口，結果補記於下方。

- 本批精確來源707項、179892360 bytes保全，前批370份快照逐項核對未變，入口 `workplace/hd/transport-runtime-source-manifest-v1-20261003.json`。來源snapshot保存收尾補記前文件bytes，不覆寫。GitHub #34全文回讀相符，2026-10-02T18:54:51Z、仍OPEN；CONTEXT同步，#44未改。

## 70. ROOM0 #22正常離開降落平台來源

進行中。沿16-sivad原版state及17-saved2第一個Up接續，到上一批620000000步終點。
只查(4,124)房間矩形的原始來源與返回，不重跑九鍵交通或存檔流程。
唯讀探針 `tools/hd/observe_room22.go`，獨立核對 `tools/hd/verify_room22.py`；
原版26檔537圖唯一來源核對、位移／錯圖號負對照及無觀察終點比較。
本批使用room22-source新前綴，不覆寫前批來源；達證據停止線後才審查限定規格。

來源路徑診斷入口 `tools/hd/observe_room22_paths.go`；正常內容比對入口 `tools/hd/observe_room22_content.go`。首版0整張入口，第二版畫面相同但預期原版state失配；首Up放開間隔與前批正常交通3百萬步不同，改用前批相同間隔查證。兩版保留，不據此宣稱來源完成。

獨立核對入口 `tools/hd/verify_room22_content.py`：19個內容變動樣本相符，17個中途未辨識完整原圖；兩份完整圖各100格／36邊界，外緣負對照99，位移927／錯圖號4642。原版觀察／控制／前批終點相同，已知入口0次，不再追未知driver。024 §1.12完成DRAFT證據審查轉READY，規格完整bytes與實作前theme.go SHA保存在room22-ready-spec／review-v1-20261003；尚未實作。

限定實作沿用完整比對，新增#22；既有五房間互斥／恢復測試包含#2→#22→#2。新25筆主題準備工具 `tools/hd/prepare_room22_theme.py`，正常驗收 `tools/hd/verify_room22_runtime.go`、載回 `tools/hd/verify_room22_reload.go`、實際逐步 `tools/hd/room22_pwstep_run.py`、獨立完整合成 `tools/hd/verify_room22_render.py`；本批工具接續限定規格，不重跑前批24筆交通路線。

正常Up／Down／Up13取樣、四份真實state載回接續與16份pwstep已執行，原版及中文相同。獨立整屏首版對終點的#2預期失敗，原版內容實際627000000為#2、630000000起又#22；保留核對器精確來源及失敗原因，第二版依實際切換在627000000驗#22清除與殘留負對照。未改產品／像素門檻，終點returned檔名只是首版標籤，實際內容仍獨立要求#22。

精確來源保全入口 `tools/hd/preserve_room22.py`，包括失敗版本、三版來源二進位與未執行的RLE核對準備稿；現行獨立來源入口為verify_room22_content.py，未執行稿不作證據。

### 70.1 限定實作與正常驗證結果

- 20主測試／34含子案例通過，失敗／略過0，五來源切換包含#2→#22→#2、失配清除／恢復、HD開關及重建；原版PBL版本／位置／src／match與其他未證實來源拒絕沿用。
- 25筆主題 `theme-room22-v1-20261003/` 的正常Up／Down／Up三分支13樣本，原版CPU、steps、cycles、IRQ1、RAM／frame相同，HD兩側中文快照相同，完整終點匯流排亦相同。六次IRQ1，640000000步、3126337427 cycles。起點16-sivad／F95Bh未改。
- 615380000首個完整抽樣#22；627000000步顯示#2，630000000至終點再顯示#22。正常#2／#22完整樣本2／8個；#22省略41721像素，627000000故意保留#22負對照42048。三個繪製中途無完整房間，回退原版。
- 四份本批真實state載回、開關、重建及各100000步接續，原版狀態一致，前後八圖面獨立不符0。16份實際pwstep原版／中文與HD兩側的原版終點與中文快照相同，完整合成獨立不符0。
- 全部25筆PBL／PNG、正式GOLEMFNT及六份文本獨立圖面／字型／整屏不符0，13個實際中文鍵相符，改錯I_MENU01:04F0負對照13筆。不排除角色區域或游標；#22與接續終點兩張中文HD PNG已目視，未當作正式美術通過。
- 正常、載回及pwstep執行argv／二進位SHA收據採room22-runtime／reload／pwstep-execution-v1-20261003，獨立完整核對 `room22-render-verification-v2-20261003.json`；全部在workplace/hd。首版核對器與失敗原因保存，未因驗證假設再改正式產品。
- 本批只驗正常平台分支及真實state接續；完整房間／sprite、美術、正式GUI、原版DAT正式GUI、幀率與封包仍待完成。特效024 §1.5維持DRAFT，#34 OPEN。精確來源保全及遠端同步結果補記於下方。

- ROOM0 #22及25筆主題來源保全645項、137010021 bytes，前批707份snapshot未變；入口 `workplace/hd/room22-source-manifest-v1-20261003.json`。保存收尾補記前文件bytes，不覆寫歷史。GitHub #34寫入前現況未變，更新後全文回讀相符，2026-10-02T19:27:19Z、仍OPEN；CONTEXT同步，#44未改，原版／候選未上傳。

## 71. 其他sprite的正常內容盤點與F2探索

本節為來源探索，不是正式HD契約或美術驗收。入口 `tools/hd/explore_sprite_sources.py --phase initial`。
在既有專用Go容器內執行，工作樹唯讀、原版唯讀，只有workplace/hd與Go快取可寫。
工具建置現行dosgolem probe，再由07-first-play及09-battle-won的原版state，
分別執行不新增鍵與真實F2按鍵。原版檔案、記憶體、seed與角色資料不注入。
全部來源SHA、實際argv、二進位SHA、原版畫面及state採companion-explore-v1-20261003前綴，拒絕覆寫。
PBL採獨立嚴格RLE解碼，缺byte即失敗，不用零填充；完整內容配對只查已有證據的座標。
未匹配不代表來源不存在，F2與控制輸入不同也不要求終點相同；精確貼圖入口與IRQ1序列另行查證。
執行結果與下一步待完成後補記，正式25筆主題未因本節改動。
鄰近房間探索入口 `tools/hd/explore_room_sources.py`，採既有地圖線索，
從正常07-first-play以真實左轉／前進鍵到BBS、前進／右轉／前進鍵到Radar，
實際房間來源與座標以新畫面查證，不把路線標籤當作來源證據。
迷宮圖塊獨立核對入口 `tools/hd/verify_maze_tiles.py`。先查MAZE.BIN候選4×4排列，
再以正常視野完整重建與不推進指令的原版RAM傾印確認內容來源；未辨明slot身份時保留歧義。
本節不把迷宮視野硬配成ROOM.PBL，也不把原版4×4素材尺寸改稱使用者的全域8×8覆繪格。
原版讀取端觀察入口 `tools/hd/observe_maze_reads.py`，使用既有probe的read-watch、8 bytes粒度。
只記圖塊切換，不把合併過的序列當完整貼圖次數；同鍵序的完整state欄位與原版frame必須和未觀察來源相同。
完整state核對入口 `tools/hd/compare_maze_saved.py`，依實際machineState全部欄位生成獨立Go解碼器，
核對gob映射內容及完整DOS區段，不排除CPU、RAM、埠或顯示器狀態；CPU／RAM／埠變更負對照必須失敗。

### 71.1 正常來源結果

【confirmed，原版內容與正常輸入】Go1.24.13、Python3.11.2及既有psychicwar-go-ebiten映像，
一CPU、開工load13.62，沒有作幀率或音訊時序聲明。
22份既有參考frame的固定位置完整配對，只找到已接入ALLY #0及既有ROOM0來源；
其中舊探索frame只作內容參考，不代替本節07-first-play與新BBS路線的正常輸入證據。
正常07-first-play／86AFh與09-battle-won／A71Dh各走一次不新增鍵及一次F2。
F2各2次IRQ1，沒有新sprite；與控制只差原版游標25像素，不推論F2在其他前提下無效。

從正常07-first-play，左轉42500000、前進48500000，按住各3000000指令、無typematic，
實際4次IRQ1後到BBS，區域0、(0,14)、朝西、地點07h，畫面已目視。
另一條前進／右轉／前進有6次IRQ1，實際停在(1,13)、朝東、走廊20h，
原版顯示不可前進；沒有到Radar，路線標籤不是到達證據。
BBS的72×72視野無完整ROOM.PBL來源，不能因此猜接新的ROOM圖號。
新收據 `companion-explore-v2-20261003.json`、`room-nearby-explore-v1-20261003.json`，均在workplace/hd。

【confirmed，MAZE原始內容】`MAZE.BIN`2048 bytes，SHA-256
`8b3b08ef483ff8c64beb04c69c861763c1258afba2348a7058e828bcf4060756`。
每8 bytes是4×4、4bpp逐列圖塊，高半位元組在左；256個slot、253種內容。
正常視野(4,124)、72×72，34份畫面中23份非空視野的324格全部可重建，不符0。
共觀察166種圖塊內容；6份全黑只屬內容配對，5份不完整配對不猜來源。
三份正常state不推進指令，A0000h以下RAM完整來源都在執行期線性0x12E16，原始2048 bytes相同。
一像素、source tile一byte及原版x移一像素負對照有效。
槽位3／235、6／201、70／220同bytes，畫面內容不能消除身份歧義。
ROOM0 #22也可分解為MAZE圖塊，完整配對不能單獨證明其繪製路徑。
原始收據 `maze-tile-sources-v2-20261003.json`。

【confirmed，執行期讀取】同正常BBS鍵序，唯讀監看0x12E16–0x13615、8 bytes粒度，
376次slot切換、59個slot，讀取觀察點CS:IP0161:4FFA。
粒度合併連續同slot，不宣稱376次完整貼圖；讀取PC不當作函式入口。
觀察與未觀察的66000000步原版frame相同；65999999步保存state的全部44個機器欄位，
含完整RAM、CPU、cycles、Ports／PortsIn、VGA平面、latch與AC，以及完整DOS gob區段相同。
state結束cycles299420040；CPU／RAM／埠變更負對照全部失敗。
gzip bytes差161、解壓機器gob差27，原因為埠map序列化順序；機器完整語意與DOS區段不變。
不是產品缺陷，也沒有排除任何欄位或調低像素門檻。
收據 `maze-read-source-v1-20261003.json`、`maze-saved-state-independent-v1-20261003.json`。

### 71.2 失敗分類與下一閘門

- 首版探索要求不存在的dosgolem/go.sum，原版尚未執行；缺檔明示記錄後重跑同工具鏈。
- 第二版probe存檔點在截止步數，迴圈沒有執行該保存點；改為截止前一指令，frame仍取實際截止，兩者時間分開記錄。舊輸出不覆寫。
- MAZE核對首版負對照假定所有完整配對視野非空；全黑畫面仍可核對純內容，不能當作迷宮功能。保存首版後修正，不改產品。
- `-dump-mem lin:...:<帶連字號的檔名>`被probe的字串判斷誤當範圍，exit0但沒有輸出；讀實際程式後使用已實作的`0-a0000:<路徑>`線性範圍契約。失敗log保留，不把這次工具故障寫成遊戲缺陷。
- 第一份讀取驗證錯要求state gzip bytes相同。保留後依完整機器gob欄位及DOS區段核對映射，不重跑已完成的原版觀察；完整比較及負對照通過。
- 精確失敗版為companion-explore-tool-source-v1／v2／v3、verify-maze-tiles-source-v1／v2與observe-maze-reads-source-v1，均在workplace/hd。

024 §1.13為DRAFT。下一步追4×4寫入的座標／中途與接合契約，出可審查圖面原型；
原版4×4尺寸不改使用者的全域8×8遮罩。完整來源、資料契約與美術未完成前不加入正式載入器。
正式主題仍25筆，ENEMY正式美術0/360、效果§1.5 DRAFT，全部sprite與HD交付未完成。
精確來源保全入口 `tools/hd/preserve_maze_sources.py`，結果於完成後補記。

- 本批MAZE／正常探索／完整state／失敗版本與精確程式來源保全283項、25424051 bytes，全部snapshot SHA／大小／UID/GID相符；前批ROOM22的645份快照未變。入口 `workplace/hd/maze-source-manifest-v1-20261003.json`。保存收尾補記前文件bytes，不覆寫歷史。
- GitHub #34更新前全文未變，更新後全文回讀相符，2026-10-02T20:22:37Z、仍OPEN；CONTEXT同步，#44未改。正式主題25筆維持，MAZE §1.13 DRAFT，原版、state、RAM與PNG未上傳。

## 72. MAZE 圖塊繪製端與中途狀態

進行中。由 §71 的原版讀取 PC `0161:4FFA` 查 IDA 函式邊界與資料流，再以相同正常 BBS 輸入核對每次圖塊寫入。
IDA 入口 `tools/ida/maze.py`，僅匯出原始名稱、位址、bytes、operand 與交叉參照，不修改正式資料庫。
正常路線唯讀觀察入口 `tools/hd/observe_maze_draw.go`。由 §71 的同一開局 state、左轉／前進與 seed 核對原始圖塊入口、四列中途及完整返回；全畫面前後、原始 slot 與返回堆疊留本機。
獨立核對入口 `tools/hd/verify_maze_draw.py`。期望值直接來自玩家自備 MAZE.BIN，重建完整前後畫面及四列中途；錯來源、錯 slot、位移、省略與提早完成的負對照必須失敗。
原始 selector 表核對入口 `tools/hd/verify_maze_table.py`，由窄範圍 IDA bytes、實際 AL 圖號及三份原版 RAM／畫面核對 `CS:307B`，不以重複圖塊的像素猜身份。
精確來源保全入口 `tools/hd/preserve_maze_draw.py`。本批snapshot與manifest在workplace/hd的source-maze-draw-v1-20261003／maze-draw-source-manifest-v1-20261003.json，原版輸入只記SHA。
一次性資料庫與輸出在 `workplace/ida/hd-maze-20261003/`；正式 `PW_UNP.EXE.i64` 與解壓執行檔唯讀掛載。
先跑最小輸出探針並核對輸入 SHA-256，再查窄範圍。
本節尚未變更 §1.13 DRAFT 或正式 25 筆主題，原版 4×4 素材尺寸不改全域 8×8 覆繪格。

### 72.1 原始定位與指令資料流

【confirmed，輸入與工具】IDA 9.4／Python 3.12.3，映像
`ida-pro-9.4-idapython:locked-v1`，ID `sha256:6f6d59af49d0008c4109a5295b5f374bdc007e2d1ab28cb9de08779584de2780`。
最小 JSON schema、417個現有函式、UID/GID1000與原始輸入雜湊核對通過。
`PW_UNP.EXE` SHA-256 `fd5115b91f014c47a1fb1a6e2ecfd645e293264fe614a4b350e92637cad2fbd9`；
正式 `PW_UNP.EXE.i64` SHA-256 `4db51a199f5913c52dc4964557b9a4aafb965754d663cb0173c9dfdfe9a00c56`。
只對一次性 DB 解碼，未改正式函式名、邊界、bytes或資料庫。

【confirmed，bytes與位址基準】原版 RAM 的 CS:4FE0–5007 共40 bytes在解壓檔唯一出現，
檔案偏移0x58C0。讀取 `mov dh,[bx]` 的runtime入口是0161:4FF8，read-watch回呼時IP已到4FFA。
同bytes對應IDA ea0x15508，IDA seg002基址0x10510；數值以各自位址空間標示。
原版4FD6–504F與4F60–4FD5兩段bytes各與IDA匯出完全相同。
正式DB沒有4FD6附近的函式；首次func與file-offset查詢均回報未知，沒有因此改名或猜補邊界。
窄範圍指令解碼來源 `decode.json`／`parent.json`，原本有函式的列分派器是
IDA `sub_1ACC2`、ea0x1ACC2、seg002:A7B2，對應runtime0161:A7B2，來源 `row.json`。

### 72.2 正常648次圖塊與列中途

【confirmed，限定正常路線】沿§71的07-first-play.state與86AFh，
42500000步左轉、48500000步前進，各按住3000000指令、不重複按鍵。
648次入口CS:IP0161:4FD6，AL為slot，X＝DH×4、Y＝DL×4；59種slot、324個座標，
X為4–72、Y為124–192，每次左轉／前進各完整遍歷324格。
每次來源DS:BX由DS:AED4指標與slot×8取得；DS1175h、指標16C6h，
來源base線性0x12E16，原始8 bytes皆與MAZE.BIN相同。
四列分別在runtime5002／5010／501E／502C完成，SI＝X、DI＝Y＋列序。
648次near返回皆為0161:4FBE，原始SS、返回SP與返回IP一同核對。

【confirmed，獨立oracle】Python 3.13.15由原始MAZE與完整前後畫面重算copy，
648次整屏重建不符0，圖塊區外不變；2,592次中途的4×4矩形與已完成列／未完成列相符。
中途只核對圖塊矩形，不外推整屏中途已驗。錯一像素、錯slot、位移、
省略與提早畫完的負對照分別差1／10／5／8／5像素。
648對完整畫面合併82,944,000 bytes，長度與SHA已核對，不以檔名或觀察器成功代替核對。
收據 `maze-draw-source-v2-20261003.json`、`maze-draw-verification-v1-20261003.json`；
來源frame stream與實際argv、工具二進位SHA均留在workplace/hd。

【confirmed，觀察不干擾原版】4次IRQ1發生於42500000／45500000／48500000／51500000步。
正常觀察與無觀察控制的終點fingerprint及完整machine.Snapshot相同，終點frame與既有BBS相同。
65999999步保存state的全部44個機器欄位與完整DOS區段亦和既有BBS相同，
canonical machine SHA `6ac1ff27c7292501415842660ea9230dc9a1c1aec4577c755f89cf35f3a750a9`，
cycles299420040；CPU／RAM／埠負對照有效。沿用已保存的獨立state解碼器，
收據 `maze-draw-saved-state-independent-v1-20261003.json`，沒有重生或覆寫舊解碼器。

### 72.3 原始slot表與保存狀態

【confirmed，所列三份狀態】runtime CS:307B、線性0x468B保存324 bytes的18×18表。
slot索引為row×18＋column；原版貼圖遍歷column在外、row在內。
原始bytes `BB7B30`、`2E8A07`、`B91200`、`03D9`與動態648次AL／座標相符。
開局、BBS與受阻Radar三份原版RAM表各重建72×72視野，不符0，單像素負對照有效。
BBS最後324個實際AL身份逐格和保存表相同。
槽位3／235、6／201、70／220仍同bytes，但原始表能保留本狀態實際身份；
不將僅59種動態來源宣稱為全部256種已實跑。收據 `maze-table-verification-v1-20261003.json`。
共同迴圈最後runtime4FD5是RET的靜態指令證據，其實際共同函式入口／返回尚未另行觀察。

### 72.4 限制與失敗分類

- 首版執行日誌使用觀察器的拒絕覆寫前綴，觀察器啟動即退出，原版未執行。保留首版log與execution；修正日誌命名後，用同一二進位正常跑通，產品未改。
- 初次輸出查詢結果的閱讀器錯假定unknown項目必有original_name，改按實際未知結構讀取。原始匯出未失敗，正式DB未改。
- 共同入口的前一RET線索搜尋沒有找到，沒有採用負數索引或據此命名。改從確定bytes的4F60窄範圍解碼，函式邊界維持unknown。
- 開工load9.22，沿用離線單CPU容器；本批不作幀率、音訊或牆上時間聲明。

本批來源已足以進入圖塊資料契約與視覺原型；先停下逐指令追查。
其他場景slot表有效條件、ROOM圖面優先序與正式HD素材尚未審查，024 §1.13維持DRAFT。
正式主題仍25筆；ENEMY正式美術0/360，完整sprite與交付未完成。

- 本批221項精確來源、98,379,975 bytes已保全，前批MAZE的283份snapshot逐項SHA／大小／UID未變。入口 `workplace/hd/maze-draw-source-manifest-v1-20261003.json`；正式DB與原版輸入保持唯讀，文件快照保存收尾補記前bytes。
- GitHub #34更新前全文未變，更新後全文回讀相符，updatedAt `2026-10-02T21:08:52Z`，仍OPEN；CONTEXT同步，#44未改。沒有新增正式迷宮HD或上傳本機素材。

## 73. MAZE資料表示與幾何重繪原型

進行中。入口 `tools/hd/prototype_maze_theme.py`，產物在 `workplace/hd/maze-theme-prototype-v2-20261003/`；首版來源與部分輸出保留。
使用§72的原始圖號表與來源，準備單張16×16圖集及256張獨立PNG兩份可丟棄資料表示。
兩份資料使用同一組幾何候選；不把MAZE偽裝成PBL，不修改正式載入器或§1.13 DRAFT。
幾何候選從原始色區輪廓建立多邊形，僅簡化格內階梯斜邊，保留圖塊邊緣的原版色號接合。
獨立核對入口 `tools/hd/verify_maze_prototype.py`，實際讀取兩份清單及PNG，核對原始格邊、三份正常狀態及648份實際after畫面的全域8×8遮罩，包含缺圖號、重複圖號、圖集錯尺寸與失配／恢復負對照。
重疊條件核對入口 `tools/hd/probe_maze_room_overlap.py`，不推進指令，讀取§70保存的13份正常state之MAZE表與frame；RAM／frame須和既有原版收據相同，不能把原型優先序當成原版事實。
回填護欄入口 `tools/hd/verify_maze_backlink.py`，核對原始0161:5002、證據等級與024 §1.12的來源別名標記；缺標記或混淆推論等級即失敗。
未經選定的候選不算正式HD美術。

### 73.1 可審查原型與獨立核對

兩份原型資料的256個PNG圖塊實際讀回相同。一張192×192 PNG包含16×16格；
另一份為256張12×12 PNG。完整提案保留現行25筆entries，另加MAZE資料，
分別在原型目錄的 `theme-atlas-proposal.json`／`theme-files-proposal.json`。
兩份均為未定案的 `psychic-war-theme/2`，正式載入器尚不接受；舊主題未改。
原型清單使用研究專用schema，不將其冒稱正式載入器已驗。

幾何候選將原始色區輪廓轉為多邊形，格內折線簡化門檻0.51原版像素，
4×4超取樣求覆蓋率；圖塊外緣一個HD像素保持原色，供相鄰圖塊接合。
73／256格的內部像素不同於原版nearest，其他格保留直線色區；這是候選，非完整美術通過。
開局、BBS與受阻Radar三份畫面的改變像素為453／44／0，畫面其餘部分未改。
`initial-comparison.png`／`bbs-comparison.png`左原版、右幾何候選，已目視；
全畫面與8×8圖面均在同一原型目錄，沒有改變原版座標或移動框架／人物。

Python3.13.15獨立解析兩份清單及PNG，256格邊緣色號與透明度相符；
三份正常圖面的100個原點8×8格及648份實際after frame的圖面兩種表示完全相同。
實際中途有效格為19–100，不以假造中途frame驗遮罩。
缺圖號、重複圖號、錯圖集布局與路徑越界均拒絕；改(0,120)的補邊一像素，
全域(0,120)格整格失效99格，撤回後恢復100格，圖面差144個HD像素。
原型 `receipt.json`、獨立 `verification.json`、兩份完整提案與既有素材副本全部留本機。

首版用bytearray累加多色覆蓋率，遇到四捨五入總量256而退出。保留首版來源
`workplace/hd/maze-prototype-source-v1-20261003.py`及部分PNG，改整數累加後正規化，
第二版同一工具鏈通過；正式產品未改，也未放寬像素比對。
開工load33.25、一CPU離線容器，不作幀率或音訊聲明。

### 73.2 MAZE與ROOM0 #22內容別名

【confirmed，既有正常state零步讀取】13份§70原版state的完整A0000h以下RAM與frame雜湊相同，
MAZE原始2048 bytes均保留；只讀資料，不重跑玩家路線或CPU指令。
兩份ROOM0 #2畫面仍有其他MAZE表，原始視野失配4643／4642像素；
三份中途與目標表視野依序差4351／1949／4像素。
八份完整#22畫面和原始MAZE表重建相同，表的目標像素也與PBL #22相同。
615210000步原版PC為runtime0161:5002，SI8、DI124、DS1175、BX16C6，
正處在§72已驗MAZE第一列返回。這是本路線MAZE正在繪製的證據。
【強推論】本路線完整#22可由這次MAZE繪製產生；沒有另行實測完整共同入口／返回，
不把此推論寫成原版RLE／PBL呼叫或所有場景正式優先序。

同一原始像素既能由MAZE表重建，也存在ROOM0.PBL #22，僅依完整內容的來源名不唯一。
既有§70及024 §1.12的限定內容驗證仍成立，但不能因表還在RAM就疊在ROOM0 #2上。
優先序與其他場景有效條件待審查，正式載入器及25筆主題沒有因此變更。
本機收據 `workplace/hd/maze-room-overlap-v1-20261003.json`，RAM／frame／log及實際argv均保留。

| 不可變原始鍵 | 已解出的內容 | 新證據 | 舊consumer與必備回填 |
|---|---|---|---|
| DOS PW.EXE SHA88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49，runtime0161:5002 | confirmed：正常#22中途在MAZE列返回；完整#22來源為強推論 | 本節及§72，615210000步原版state／RAM／frame | 024 §1.12，標記「MAZE／ROOM0 #22內容別名」；§70限定內容驗證不受影響 |

資料表示已透過grilling提出單張圖集／256PNG選擇，尚未收到回答；
024 §1.13維持DRAFT，沒有把建議當成同意。正式ENEMY美術0/360及全部sprite／交付仍未完成。

- 本批406項精確來源、103,156,186 bytes已保全，入口 `workplace/hd/maze-prototype-source-manifest-v1-20261003.json`；snapshot在 `workplace/hd/source-maze-prototype-v1-20261003/`。一次性保全程式同存為preserve-source.py，已逐項核對SHA、大小與UID/GID1000。原版只記SHA，文件保存此收尾補記前版本，不覆寫核對時輸入。

## 74. 現行25筆主題的正常視窗與DAT存讀檔

進行中。沿用已READY的顯示契約，不新增產品功能。工具入口 `tools/hd/gui_dat_run.py`，使用既有專用Go／Ebiten映像、Xvfb與真正視窗鍵盤事件，逐畫面確認原版存檔選單後才送下一鍵。
本批輸出位於 `workplace/hd/gui-dat-v1-20261003/`，現行前端與原版狀態匯出二進位位於workplace/hd；保存正常玩家state接續，不稱為從開機完整試玩。
擷取F10的原版state、中文sidecar及真正視窗PNG；兩者時間不同，逐份記錄相位，不排除游標或角色區域來通過。原版DAT只在獨立scratch落地，原版輸入唯讀，原版存讀檔與F10／F11分開判讀。
原版狀態匯出入口 `tools/hd/export_gui_dat.go`，零指令載回各份F10狀態，只匯出原版色號、RGB與原始52 bytes玩家區域，保留實際CS:IP及指令數；不據此宣稱整份GUI RAM同狀態。
獨立核對入口 `tools/hd/verify_gui_dat.py`；直接由原版PBL、PNG、正式文本及字模建立完整期望，再比實際視窗相位。存讀檔只驗正常選單DAT與明示的52 bytes玩家區域，不拿F11替代原版LOAD GAME。
獨立合成沿用§70的全部25筆素材、原始PBL、正式文本與字型；不是發行包、真機、幀率、音訊或全部sprite驗收。
精確來源保全入口 `tools/hd/preserve_gui_dat.py`，snapshot與manifest位於workplace/hd的source-gui-dat-v2-20261003／gui-dat-source-manifest-v2-20261003.json。兩版GUI工具以各自執行時SHA解決，不用新版本冒充首版；原版僅記雜湊。首版source及495份部分snapshot保留；DOS模組與go.work原本沒有go.sum，依實際結構明示缺少的可選metadata，不加入假檔。

### 74.1 正常視窗與原版DAT有限驗證

【confirmed，限定正常state接續】現行25筆主題、Go1.24.13／Ebiten2.9.9，Linux amd64真正960×600視窗。
映像psychicwar-go-ebiten:latest，ID sha256:083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7。
保存分支從§70 sample04正常state接續，含原版既有排定輸入；讀檔分支由05-select正常主選單state接續。
Esc→功能選項→儲存遊戲→當然要→HD25→繼續玩，每一階段先讀實際視窗再送下一鍵；
另一份執行器由LOAD GAME輸入HD25，沒有F11、RAM注入或作弊替代原版讀檔。

Python3.13.15直接解析原始PBL、25筆PNG、正式text及字模，11份保存路徑及5份讀取路徑共16份完整合成不符0。
每份預先擷取12個相位，找到整屏完全一致才通過，未排除角色、游標或任何區域；
各份選定相位、實際比較數及所有PNG都保留。14份原版frame的完整內容鍵為ROOM0 #22及ALLY #0，
兩份主選單／檔名框無角色來源，不稱全部25筆素材都在GUI出現。中文鍵及多行間隔對正式資料相符。
省略HD與中文各有非零差異，兩種原版英文樣本本來不畫覆繪，不把它們當非空負對照。

原版HD25.DAT為512 bytes，與既有獨立TEST2.DAT完全相同，
SHA-256 781da2d6bc693fe370061b74d1cdefaf1472fd8c217e711908aaab36adb43acc。
保存前及三份載入後的原始線性0x16966起52 bytes，與既有18-loaded2.mem相同；
區域1、位置(5,6)、方向0、HP28、能量6。只證明這個明示範圍，不稱GUI全部RAM或自然亂數相同。
DAT改一byte負對照有效。讀檔GUI由quit-after正常退出0，14筆原版按鍵完整，
Down／Return／H／D／Digit2／Digit5／Return各按下與放開，沒有F5／Shift+F5／F10外洩。
收據workplace/hd/gui-dat-independent-v2-20261003.json，229項輸入SHA。

### 74.2 工具失敗與限制

- 保存GUI首版用xdotool windowclose，造成XGetProperty BadWindow而退出1，record未保存；不是原版存檔失敗。DAT及保存成功畫面均保留，讀檔分支以相同產品二進位的quit-after正常結束。
- 首次工具source保全假定執行收據鍵為相對路徑，實際為/src絕對路徑，KeyError；按原始鍵及SHA精確恢復首版，不以新版來源冒充。
- 核對首版把PW.EXE:cs:8A4D的三行譯文當成單行，比到檔名框時自行ValueError終止。實際正式資料為第一行提示、空白第二行、第三行括號，側檔在y89／103；改按明示newline及cell_h驗各行內容／間隔，產品未改，未放寬像素門檻。
- 曾預計12相位全解碼會超時，準備停止時容器已因上項自行結束；stop回報No such container，首版沒有成功收據。來源與實際終止原因皆保留。新版找到完整一致相位即停止多餘解碼，接受條件仍是預先12張中至少一張整屏一致，並移除首版轉場容許分支。
- 保全首版假定DOS模組有go.sum而失敗，495份部分snapshot保留；實際DOS模組／go.work沒有該metadata，新版明示缺少的可選檔，不造假檔。
- 開工load20.58，使用單CPU離線容器及null音訊；不作幀率、速度或音訊聲明。本批是保存正常state接續的Linux視窗抽測，不是開機全流程、更多角色／F11 GUI、真機、全部sprite、美術、封包或完整HD完成。

精確來源693項、82,689,258 bytes保全，入口gui-dat-source-manifest-v2-20261003.json；
每份SHA、大小與UID/GID1000核對，原版僅記雜湊，文件snapshot在此結果補記前。
現行25筆與READY契約未改，迷宮§1.13與特效§1.5仍DRAFT，正式ENEMY美術0/360。

| 不可變原始鍵 | 已驗範圍 | 新證據 | 舊consumer與必備回填 |
|---|---|---|---|
| DOS PW.EXE SHA88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49，原始線性0x16966起52 bytes | confirmed：本批保存前及三份LOAD GAME後與獨立原版相同；不升級其他欄位語意或全RAM | 本節及gui-dat-independent-v2-20261003.json | 024 §1.12，標記「原版DAT正式GUI已補驗，見研究038 §74」 |

收尾回填護欄與進度核對的精確程式來源在 `workplace/hd/gui-dat-progress-close-source-v1-20261003.py`，收據 `gui-dat-progress-close-v1-20261003.json`。缺原始定位、較早規格或回填標記均失敗；負對照移除標記必須失敗。此檢查不宣稱新玩家規則或完整HD完成。

## 75. ENEMY00首組美術修整

2026-10-03，進行中。沿用024 §1.6的已證實#0／#1／#2三姿勢及原版24×32矩形，僅製作美術候選，不改執行期或現行25筆主題。
原版ENEMY00.PBL SHA-256 `8241b0ea73b1e7402b13adc434f10a5b1288f88f01fcfc3ba5710e4923fe6067`。
目視舊原生圖、原版參照及72×96預覽：舊圖有平順線條，卻加入多處圓管、球狀關節；縮圖後線條模糊。這是人工審查發現，不推定所有候選都有同樣問題。

輸入為 `workplace/hd/redraw/ENEMY00-group0-reference.png`，使用內建image_gen重畫三姿勢；保持造型、主要色區、位置、比例及動作差異，減少細節。原生圖與提示詞保存至既有redraw目錄的 `ENEMY00-group0-v2-generated-20261003.png`／`ENEMY00-group0-v2-provenance-20261003.json`，技術正規化與比較圖同用v2前綴。模型輸出未檢查前不計入正式美術驗收。
美術指引 `tools/hd-art-instructions.md` 已回填完整sprite定案、逐姿勢檢查及本機候選邊界，移除「怪物第二版才處理」與自動可公開的過期描述。

v2三格尺寸72×96已正規化；比較圖上原版、中舊候選、下v2。人工核對發現第三姿勢手臂過於接近第二姿勢，未保留原版水平伸出，因此未接受。v3只修第三格手臂後已恢復水平動作，但臂帶仍偏高，繼續局部修正。v3／v4原生圖、三格、比較圖與完整提示詞沿用 `ENEMY00-group0-v3-`／`v4-` 加日期20261003的版本化檔名，均留本機；前兩格最後選用v2固定圖格，避免模型局部修改時重畫其他姿勢。

v4已將第三格水平臂帶降低，三張72×96 PNG解碼、尺寸及全不透明核對通過；十二個角落取樣皆為RGB(0,0,0)，不外推整個背景。前兩張SHA與v2相同。人工審查仍發現手部輪廓與主色區偏差，中間姿勢的褐橙區也缺原版黃色小亮點，尚未達正式美術驗收。
最終比較圖 `workplace/hd/redraw/ENEMY00-group0-v4-comparison-20261003.png` 的三列為原版、舊候選、本批選用候選；SHA-256 `80b69230675d226a85090a80acedf1f96d07abf44843b9c3c27dad6dbdd3658b`。
三次內建image_gen提示詞及各版精確輸入／輸出SHA在對應v2／v3／v4 provenance JSON。候選生成與技術檢查均不計入正式0/360；下一步修手部與色區，未替換現行主題。

後續改用單姿勢參照：v5前綴的source／target檔案為固定圖格裁切，frame-01修褐橙區黃色亮點，frame-02修手部色區。生成、正規化與比較檔同保存在既有redraw目錄，提示詞及精確雜湊由v5-provenance記錄；任何生成偏差先檢查，不直接替換現行主題。

【HD-ENEMY00-ARMPOS-01】勘誤：v4提示詞曾把末姿勢的手部最右端寫成放大後x68–71，這是誤讀三格比較圖。原版24×32的#2在第18–21列最右墨跡為x16，放大後止於x50；不能因此把手伸至圖格右緣。v5兩份單姿勢參照縮回原尺寸後與既有原始PNG逐像素不符0；原始PBL獨立解碼、圖號偏移及臂部範圍核對記於 `workplace/hd/redraw/ENEMY00-group0-v5-source-check-20261003.json`。舊提示詞與候選保留，後續只以原版色號座標修正，物件語意不因此升級。
原始PBL的#1／#2檔案偏移420／768、尺寸24×32已核對；兩份參照與獨立解碼差0。#1黃色色號14只在(14,6)，#2第18–21列最右墨跡為16／16／16／15；單像素變更負對照有效。首次核對器不接受4位元調色盤PNG，另存8位元RGB後以同一Python3.13.15工具鏈重跑，原版與原參照均未改。

v5單姿勢生成#1／#2原生圖均1086×1448，比例3:4未變，直接縮至72×96。#1黃色亮點已出現，但量測只有(44,17)一個明顯黃色像素，原版三倍矩形(42,18,3,3)內仍為0；不能稱位置與面積已修正。#2手部較v4緊湊，但第54–59列最右墨跡仍到x54，原版止於50，仍待修。兩份提示詞、原生圖、正規化成品及比較圖由 `ENEMY00-group0-v5-provenance-20261003.json` 索引；不以生成成功或更接近便計入正式美術。
v6僅續修#1黃色亮點，版本化原生圖、frame-01、三姿勢比較與提示詞紀錄沿用 `ENEMY00-group0-v6-` 加日期20261003前綴；不得重畫其他姿勢後覆寫已選圖格。

v6亮點量測為(43,16)、(44,16)、(43,17)、(44,17)，原版三倍矩形內仍為0。局部修改連續兩次增加面積卻未修位置，不能宣稱有效。重查imagegen提示指引後，v7改用原版單姿勢作唯一圖像輸入，避免既有HD頭部結構固定錯誤位置；v7前綴的原生圖、frame-01、比較圖與provenance仍存既有redraw目錄。這是修法變更，不是降低驗收要求。

v7直接重畫後，#1黃色像素(42,18)、(43,18)、(42,19)、(43,19)落回原版矩形；#0／#2也依同一來源方式重畫。三圖72×96、不透明與PNG解碼通過。末格第54–59列最右墨跡仍到x52，比v5的54縮回；不是逐像素輪廓相同。三圖獨立生成仍有頭部／管線的靜態造型差異，不能忽略動畫一致性。
接續v8以v7首姿勢為共同底稿，只按原版差分修改其他兩姿勢的臉部色區與手臂；v8前綴原生圖、三格、比較及provenance保存於redraw。本批實際位置預覽採本機 `workplace/hd/art-group0-review-v8-20261003/`，依現行25筆清單另存美術候選，使用既有READY §1.6與正常來源state零步載回；不替換正式主題，不稱新正常玩家流程或全動作驗收。

### 75.1 v8共同底稿與原座標預覽

v8兩張原生圖1086×1448，固定整幅Lanczos縮至72×96，首張沿用v7。三張解碼與不透明核對通過，#1四個黃色像素仍落在原版(42,18,3,3)矩形。共同底稿並未保證其他部分完全不變：原版未變的頭頂、軀幹與腿部列區間，#1／#2相對首姿勢仍有2,158／2,145個HD像素差異。數字包含抗鋸齒及色彩差異，不能單憑此數字認定幾何改變，也不能宣稱動畫一致性已通過。手部輪廓與主要色區仍待修，正式美術維持0/360。

本機比較圖 `redraw/ENEMY00-group0-v8-comparison-20261003.png` 三列為原版／v7／v8；v6／v7／v8完整提示詞、圖檔SHA及量測存於各版provenance JSON，均在workplace/hd。原版PBL、候選及state未上傳。

【confirmed，限定零步預覽】使用既有pwstep-room22-v1-20261003與正常來源237,710,721步的kasuruji首姿勢state，另存25筆美術審查副本，僅換ENEMY00 #0–#2 PNG。清單與原座標未改，manifest SHA仍為f39be567ae334b9babe5ebd3448cab8a63a09aa1cd9e922a4d1def2b13405a46。實際畫面此時只呈現#0，不能稱三姿勢都在玩家流程出現。

兩側`-do type:`由既有解析器展開為零個動作，沒有執行CPU指令或鍵盤事件。新舊960×600畫面差4,035像素，全部在(96,456,72,96)角色矩形內，區外差0。沿用§71獨立保存狀態比較器，兩側44個機器欄位及完整DOS gob區段相同，CPU／RAM／埠變更負對照有效。這是單姿勢原座標預覽，不是正常新流程、動畫、視窗GUI、幀率或封包驗收。

預覽及實際argv、log、state、完整比較收據在 `art-group0-review-v8-20261003/`，入口`review.json`及`saved-state-comparison.json`。執行程式保存為`preview-v2-source.py`；正規化工具保存為`normalization-source.py`，是同一方法的快取讀取修正版，不冒稱首版執行時bytes。

工具紀錄：首版正規化重複解碼PNG，外層60秒timeout退出124，但全部產物及provenance已寫出；後續讀回核對完整。首版預覽誤用被既有契約拒絕的wait:0，在執行前退出1，失敗log保留；依解析器既有空文字動作，以新檔名及同一二進位乾淨重跑通過，產品未改。此批離線單CPU，開工高負載，不作時序量測聲明。

### 75.2 手臂局部修整

2026-10-03進行中。v9僅修v8的前臂與手部，其他造型沿用共同底稿；固定來源裁切的arm-reference、生成原生圖、正規化三格、比較及完整提示詞／SHA沿用既有redraw目錄的ENEMY00-group0-v9-加日期前綴。來源裁切只作近看參照，不縮小、移動角色或用原版輪廓遮住生成偏差。先以原版實際位置及動作關係審查，再決定是否替換本機候選；現行25筆主題不動。

v9三張尺寸與解碼通過。近看#1的左手已減少圓形拳套，但未完成全部手部審查；#2末端縮回，白色臂帶仍偏高。源色號列核對：#2白色橫帶在原版第18／19列，三倍後第54／57列；v8／v9的明顯白色橫帶在HD第48／51列。這是高度偏差，不能因端點縮回就當作動作完成。先前中途訊息把#1右手概稱垂得太低，並不足以定位全臂偏差；後續依原始列及色區核對，不採該概稱為證據。

v10要求只調整兩份候選前臂的原版高度及白色帶位置，其他部位保留。原生圖、固定正規化圖格、比較圖及完整提示詞／SHA沿用redraw的ENEMY00-group0-v10-加日期前綴。實際#2明顯白帶仍在HD第50與52／53列，未到原版第54／57列；不能把生成成功視為高度修正完成。候選達標前不改現行主題或計入0/360。

## 76. ENEMY00 #3–#5美術動作複核

2026-10-03進行中。沿用024 §1.3／§1.4 READY與研究§35–36的正常往返循環，原版矩形仍為24×32。現行三張PNG與§36 runtime收據的素材SHA相同，舊技術驗證沒有審查造型。目視發現#5的畫面左側手部在原版參照較高，舊provenance卻要求左臂下垂；先以原始PBL圖號及單姿勢參照核對，不能沿用舊提示詞當作原版動作證據。

本批固定source／target裁切、局部生成、72×96圖格、比較與來源核對記錄存於既有redraw目錄的ENEMY00-group1-v2-加日期20261003前綴。範圍僅訂正已證實姿勢的美術，不改時序、規則、資料格式或全身位置；確認修正並核對同狀態呈現後才替換本機主題候選。

【HD-ENEMY00-POSE5-LEFTARM-01，confirmed色號位置】原版ENEMY00.PBL SHA8241b0ea73b1e7402b13adc434f10a5b1288f88f01fcfc3ba5710e4923fe6067、#5檔案偏移1899、24×32。固定第3格裁切縮回原尺寸的全部768個取樣與獨立原版解碼差0。第9–13列的左側0–5欄已出現青色／白色墨跡，第20–22列此區全黑；舊HD把同一手部畫在腰下，動作未保留。舊ENEMY00-group1-provenance.json的edit_prompt誤稱原版左臂下垂，不能作證據；舊檔及當時候選保留，修正使用者可見動作不需改原版規則或既有READY顯示契約。研究§36的像素技術核對仍只證明合成，不證明此候選造型。

v2已恢復左臂抬起，但生成了原版沒有的長食指，手套與袖口也偏高。後續v3以v2單格為底，僅修左手緊湊形狀及來源高度，檔名沿用ENEMY00-group1-v3-加日期20261003；生成成功不算修正完成，現行主題暫不替換。

v3實際改動畫面右手，左手未完成指定修正，拒絕替換。回查imagegen的單一局部修正與不變條件後，v4回到v2底稿，附原版左側0–5欄、第8–19列的近圖，以畫布左邊界座標指定唯一修改區域。arm-reference、原生圖、圖格、比較與provenance採ENEMY00-group1-v4-加日期20261003前綴，錯側版本保留。

v4已恢復左手抬起與短小突出部，右手原動作保留；72×96的青色左手最上列30，原版色號最上列三倍為27，仍有細部高度及造型差異。v2／v3／v4比較圖由左至右為原版、舊候選、v2、v3、v4。來源核對收據ENEMY00-group1-v4-source-check-20261003.json驗全部248,832個放大參照像素不符0，單像素變更負對照差1。

首版青色門檻也可能納入灰色抗鋸齒邊緣，provenance初值不當作手部定位證據。後續採G及B大於180且各比R大於50，v2／v3／v4的左手頂列為21／21／30、右手為17／29／17；只量色區，不外推物件語意。修正量測另存ENEMY00-group1-v4-height-check-20261003.json，不覆寫原紀錄。

限定接續驗證保存於 `workplace/hd/art-pose5-runtime-v1-20261003/`，其中theme是另存的25筆候選副本，僅更新ENEMY00 #5，原manifest與其他24張PNG保持相同。入口runtime-source.py及各案例輸出；首版未完成，沒有成功verification.json。使用§36正常來源三姿勢與四方向相鄰state，既有pwstep零動作與正時長接續、兩側保存狀態及獨立PNG核對。這不是新GUI、全美術或發行包驗收；通過前現行主題維持ROOM22版本。

首版三個完整姿勢的原座標、兩側44個機器欄位及完整DOS區段通過，#5新舊差3,942像素，區外差0；接續1000ms時要求整個矩形等於一張PNG而失敗。原版獨立零步輸出與#4差27個三倍像素，差異是否源於繪製中途或重疊尚未分類。依8×8來源比對重建，當時保留#3的9個有效格，完整角色區期望不符0，另外兩個姿勢的期望均不相符。這只證明輸出相容於舊姿勢投影，尚不足以排除產品的姿勢更新問題。

新版在 `workplace/hd/art-pose5-runtime-v2-20261003/`，同一二進位、原版輸入與動作，另取未轉譯／未HD的原版畫面，由原始PBL及實際PNG獨立建立三個允許的完整8×8期望，要求唯一相符且整個矩形不符0。每個回退格仍比較原版像素，沒有排除區域。projection_image只表示唯一相符的顯示候選，不宣稱重新觀察原版差分呼叫或精確動作次序。可重現來源runtime-source.py，收據verification.json；首版來源與部分結果保留。

七份樣本的此一致性檢查通過，兩側及原版對照44個機器欄位、完整DOS區段相同。它單獨不證明載回後的動作選擇正確：四份接續的projection_image均仍為起始圖號。後續§77已由實際入口bytes證實各樣本落在下一次貼圖中途，並補驗返回後完整恢復；不再列為未解的姿勢更新缺口。現行原始碼與舊二進位畫面相同，實際來源及獨立期望在art-pose5-runtime-v3-20261003，單張相容性不代替來源證據。

## 77. 載回後的中途貼圖複核

2026-10-03複核完成，限以下四份正常來源state。§76的相同event03-pose4.state與wait:1000，現行原始碼新建置與舊二進位全畫面差0；獨立原版完整#5差0，HD角色區與#5候選差3,963像素，角色區外差0。這個差異落在已進入#5→#4、尚未返回的中途：原版還是完整前姿勢，HD依024 §1.3／§1.4只顯示吻合本次目標#4的原點8×8格。不是載回後保留上一個來源認定。全部保存狀態由既有獨立比較器核對相同，中文側檔相同；不作GUI、完整動作或美術完成聲明。

複核入口`workplace/hd/art-pose5-runtime-v3-20261003/runtime-source.py`與`verification.json`；來源、逐格及保存狀態獨立核對為`gate-independent-source-v2.py`／`gate-independent-v3.json`，正常來源觀察入口`apps/psychicwar/theme/pose_load_test.go`。原版只讀，固定各state與輸入，cycles750；開工load19.71，離線單CPU不作時序聲明。沒有改正式圖面實作、原版規則或主題。

【HD-POSE-CONTINUATION-GATE-01，confirmed限定四份接續】Go側保存實際0161:8705入口的DS:BX 384 bytes、完整前姿勢768色號與0161:8751返回的完整後姿勢。Python獨立解碼PBL #3／#4／#5，要求唯一完整前姿勢及已證實相鄰方向的打包XOR相符；已返回事件還要求完整後姿勢等於來源推導的目標。四份各有4次貼圖，總16次實際來源及返回後圖面核對通過，不使用Theme.active或投影相容性推定目標。

| 正常state | 1000ms原版完整圖 | 1000ms實際貼圖目標 | 目標可見非黑8×8格 | 再接續50ms |
|---|---|---|---:|---|
| event00-pose3 | 無完整圖 | #3 | 9 | 返回後完整#3，12格 |
| event01-pose4 | 無完整圖 | #4 | 9 | 返回後完整#4，12格 |
| event02-pose5 | #4 | #5 | 8 | 返回後完整#5，12格 |
| event03-pose4 | #5 | #4 | 8 | 返回後完整#4，12格 |

共12份initial／1000ms／1050ms的完整72×96角色覆繪區與獨立PBL、PNG及原點格期望不符0，包含所有透明及回退格，沒有排除區域；另外兩個姿勢的負對照均不相符。八份1000ms／1050ms的44個機器欄位及完整DOS區段與無HD／無中文的原版控制相同，CPU／RAM／埠負對照沿用獨立比較器有效。1000ms前4次循環使貼圖目標恰巧等於起始圖號，不能只用起始／終點圖號判定殘留。

首版診斷假定完整畫面一定推翻中途目標而失敗，違反既有READY閘門；精確來源及log保存為diagnostic-test-v1-source.go／diagnostic-test-v1.log。修正觀察只登錄正在核對的敵人入口／返回，避免把其他8751命中586次當作敵人返回次數。獨立核對首版把Go nil事件清單當陣列而TypeError，尚未執行控制分支；v2依空清單處理後乾淨重跑，來源與成功log分版本保存。正式產品未改，ENEMY美術仍0/360；後續回到sprite造型、色區與動作一致性修整。

## 78. ALLY #0美術修整

2026-10-03。依024 §1.2 READY完成本批ALLY #0單格造型與限定正常呈現核對，已選入新本機25筆主題；其他ALLY與完整sprite範圍未完成。來源為已證實的ALLY.PBL #0、24×32、原版(264,152)；位置、比例、姿勢及8×8格維持使用者定案。舊候選與缺完整提示詞的舊provenance保留，來源單格、完整生成提示詞及新版本保存於既有redraw目錄的ALLY-00-v2-加日期20261003前綴。source、生成原生圖、72×96圖格、比較、來源核對及provenance由本節索引；單格審查及選入紀錄為ALLY-00-v2-review-20261003.json。

原座標正常呈現使用另存的`workplace/hd/art-ally0-review-v2-20261003/`審查副本，僅替換ALLY #0，其他24筆及manifest不變。預覽及正常來源state／原版對照／載回收據由本節索引，單格生成與技術格式不代替完整美術或玩家呈現驗收。

新候選已逐單格查看，恢復原版較大的頭部、靠右的頭部位置、紅色胸甲、青色肩甲／靴子及兩腳落點；原版三倍墨跡外框與新圖均為[9,0,71,95]，此值只證明整體外框相同，不稱輪廓逐像素相同。內部線條與面部屬HD重繪。舊圖的小頭、白甲／黃色護膝偏差改善；其他ALLY圖號及動作未審查。

依既有`tools/hd/verify_ally_runtime.go`正常名字路徑，從固定06-name.state／86AFh按K／A／I／Enter到42,000,001步。兩側完整1MiB RAM、原版frame、暫存器、steps、cycles一致；唯一ALLY完整來源在38,648,892步。新圖獨立PBL／PNG／8×8角色期望不符0，省略盟友負對照4,340像素、角色區外差0，實際state載回後角色圖面相同。收據在art-ally0-review-v2-20261003/normal.json，來源與成功log同目錄；本次不是新GUI、原版DAT、其他盟友、動畫或正式封包驗收。

本批選入新的本機`workplace/hd/theme-ally0-v2-20261003/`，25筆及原manifest保持，僅ALLY-00.png更換。準備入口`tools/hd/prepare_ally0_theme.py`，主題載入器及原版不改；舊ROOM22版完整保留。舊ROOM22版本16份GUI證據不擴張新素材範圍。ALLY #0造型及限定正常呈現已核對，其他30張ALLY、ENEMY正式美術0/360與全部sprite目標仍未完成。

工具及限制：內建image_gen生成；Go1.24.13／Python3.11.2／ImageMagick6.9.11-60的既有psychicwar-go-ebiten容器。放大參照為4位元調色盤PNG，首版pbl.read_png只支援8位元而退出，精確來源保留；同容器ImageMagick讀RGB後重跑來源核對成功，未改參照或正式產品。初始load7.24，離線單CPU，不作幀率／音訊聲明。原版、PNG及state均留本機，公開權利與正式交付另驗。

## 79. ENEMY00 #6–#8姿勢與美術修整

2026-10-03。沿用024 §1.6 READY與§49的敏頓正常來源證據，先獨立核對ENEMY00.PBL #6–#8與既有三格參照，再以單姿勢重畫。原版位置(32,152)、24×32及全域8×8格不改。來源、單格放大、內建image_gen原生圖、72×96候選、完整提示詞、SHA與比較圖由既有`workplace/hd/redraw/`的`ENEMY00-group2-v2-`及日期`20261003`前綴索引；來源準備腳本同前綴保存。

限定正常呈現審查及精確工具放在`workplace/hd/art-minton-review-v2-20261003/`，使用現行25筆主題的審查副本，只替換#6–#8。本節為研究入口，未完成審查前不選入現行主題；原版、state及候選留本機，不新增正式格式。造型、動作與技術接入分別核對，整項HD及ENEMY正式美術仍未完成。

本批三張造型與限定正常呈現審查通過，選入新的本機`workplace/hd/theme-minton-v2-20261003/`；準備入口`tools/hd/prepare_minton_theme.py`，25筆只更新ENEMY00 #6–#8，其他22張及manifest不變。正式美術進度為本批3/360，並不表示完整角色重疊、GUI、所有sprite或HD交付完成。單格審查`art-review.json`、原始路線`replay.json`、獨立來源及24份角色區核對`independent.json`、脫離清除`independent-end.json`、四份實際載回`reload-verification-v2.json`、工具／來源保全`source-manifest.json`均由本節索引，在審查目錄。選入紀錄為新主題的`selection.json`。精確來源副本保存於審查目錄的`source-snapshot/`；`build-input/normal-source.go`為單一主程式的可重現建置入口，`go-dependencies.json`索引實際Go依賴。

已證實：原版#6／#7／#8的檔案偏移為2273／2675／3079，各24×32。三格放大參照各248,832像素不符0，變更單像素負對照有效。舊HD右手偏高、末姿勢左手下伸；新圖恢復原版右手高度及末姿勢左側上彎手臂。首版#8仍過於靠近臉部而拒絕，修正版移回左側；失敗圖與完整提示詞均保留。三張墨跡外框[0,0,71,95]與原版三倍相同。頭髮、面部、腿部、右手四個靜態區域的RGB不完全相同，最大通道差8–30，人工未見部位位移；不稱逐像素輪廓或RGB完全相同。

限定正常來源：13-healed.state固定SHA／A48Ch，以既有正常方向鍵及F3重播至389,200,000步，16次完整身體貼圖、8次中途及一次脫離。獨立PBL與入口384 bytes核對16次完整原版前後整屏、24份完整角色RGBA及合成、25份中文圖面不符0，錯姿勢負對照有效。脫離後整個舊角色區與獨立SCREEN8×8背景相同，未排除失效格；故意殘留敵人負對照5,931像素。兩側原版終點暫存器、steps、cycles、IRQ1、frame及1MiB RAM相同。

三個完整姿勢及脫離終點四份真正state零步載回，完整960×600投影與正常擷取不符0；開關HD兩側全部44個保存欄位及完整DOS區段相同，CPU／RAM／埠負對照有效。首版直接比較舊state與pwstep輸出因CPUHz／CycleClock／DOSBoxCost不同而失敗；已由既有pwstep載入後SetDOSBoxCycles呼叫證實為明示設定。第二版在兩側採相同設定，不省略這三欄，完整核對通過。沒有宣稱舊state全欄位在pwstep載入後保持原樣。

環境與研究工具：Go1.24.13、Python3.11.2／3.13.15、ImageMagick6.9.11-60，初始load9.01、離線單CPU。Docker首版新子掛載點無法在唯讀父掛載內建立；改用已存在的tools目錄掛載。兩次單檔Go建置不允許internal匯入，改以dosgolem模組的套件路徑建置成功。normal前綴與工具檔名碰撞，覆寫護欄在原版執行前拒絕；改replay前綴後同二進位正常完成。這些為研究環境／腳本問題，正式runtime未改；不作幀率或音訊聲明。新三張GUI／DAT未驗，其他357張ENEMY、其他盟友、迷宮／效果DRAFT與完整交付仍未完成。

## 80. 歐格斯三姿勢美術修整

2026-10-03。沿用024 §1.7 READY及§50的ENEMY01 #6–#8正常來源，原版(32,152)、24×32及全域8×8格不改。三張來源、單格放大、生成原生圖、72×96圖格、完整提示詞及比較由既有`workplace/hd/redraw/`的`ENEMY01-group2-v2-`與日期`20261003`前綴索引。

審查副本、正常路線、獨立圖面、state與精確工具在`workplace/hd/art-oogus-review-v2-20261003/`；只替換現行25筆主題的ENEMY01三張，其他22張不變。單一主程式建置目錄`build-input/`及`source-snapshot/`由本節索引，避免與診斷工具的main衝突。來源已足夠，不重開完整RE；重疊剩餘成分仍未知，正常路線15戰鬥樣本原本回退，不能把來源身份已證實當成完整HD動作。

本批已保存15份內建image_gen原生圖及72×96候選。初次五份suffix為06／07／07-revised／08／08-revised，其後十份修正版保留。完整提示詞在同前綴generation-records-20261003.json與corrections系列，各版provenance記錄原生圖、正規化方法及SHA。審查、正常重播、載回與選入收據已建立；#6正式美術及限定正常呈現通過，#7／#8仍待審查。

原版來源收據source-check-20261003.json記錄ENEMY01.PBL SHA-256 `22f664050ce7ffa4ea0f6941c9c91cd1ab43671ea5b53491f6799f78ba8e64af`，三個圖號檔案偏移2512／2890／3269，各24×32。放大參照各248,832像素不符0，單像素變更負對照差1。來源已核對不表示候選造型、所有重疊或動作呈現通過。

本批續作的正規化來源、全部補正提示詞、各候選provenance、獨立色區量測及三欄比較沿用同一ENEMY01-group2-v2-前綴。只用原版單姿勢參照的single-source三張為新審查候選；generated／frame、generation-records-corrections及corrections2–5、geometry-check-v1–v3均由本節索引。原生圖固定整幅縮至72×96，不用原版輪廓遮住偏差。多參照版本有頭部高度及突出部位置偏差，保留但不選入。

正常路線及17份完整角色區、中文字面獨立核對在art-oogus-review-v2-20261003的replay.json與independent.json。只有首姿勢#6有完整HD；其他15份戰鬥樣本依READY回退，正常原版HP0陣亡終點不強制改勝利。三張單格定位改善不代表動畫造型一致，#7／#8暫不作正式美術選入。

首姿勢選入審查另存同一審查目錄的selected-theme/，保持25筆並僅換ENEMY01-06.png，其他24張及manifest沿用敏頓版。四份零步載回已通過原始資料獨立完整期望與44欄／DOS核對；tools/hd/prepare_oogus_pose6_theme.py已另存本機workplace/hd/theme-oogus-pose6-v2-20261003/。26份檔案與selected-theme相同，selection.json記錄全部輸入／輸出SHA。正式ENEMY美術為4/360，其他356張未完成。這是既有READY限定單張修整，完整歐格斯動作、其他sprite與交付範圍維持。

**已證實的限定驗證**：17-saved2 SHA-256 `3c2ae7470600c61217060dc76843aed7c4097ac6e4d80a778cf3282b8afbc7e4`，執行前種子F95Bh，655,000,000至695,000,000步，兩次Up及四次IRQ1，不用F3／Space。八次完整身體貼圖邏輯圖號6／7／8／7／6／7／8／7；384 bytes來源及完整原版前後整屏不符0。17份完整角色RGBA及合成、17份完整中文字面不符0，錯姿勢與殘留負對照有效。只有首樣本完整HD，其他15份戰鬥樣本依來源護欄回退；終點原版HP0陣亡。HD兩側原版終點及完整1MiB RAM相同。正常HD合成只驗角色區與完整文字圖面，不宣稱整屏HD驗收。

零步載回event00／01／02／end，用實際pwstep及selected-theme。verify-reload-source-aware.py由完整原版PBL、25張PNG、正式字型、保存frame與8×8遮罩獨立重建HD開關兩側全部960×600像素，四份不符0，未排除ALLY或任何區域。兩側相同設定下全部44個機器欄位、完整DOS區段、中文快照相同；CPU／RAM／埠、省略HD負對照有效。既有pwstep載入後設定CPUHz／CycleClock／DOSBoxCost，未省略三欄。

載回與正常擷取的完整畫面差為0／1,601／1,562／0，event01／02差異全在ALLY區。正常觀察保留先前已證實盟友身份與有效格；新載回時光束遮住原版盟友，完整來源無法辨識，依READY回退。首版verify-reload-v2.py要求整張載回等於正常擷取，因此失敗；reload-v1-failure-note.json保存原因，沒有改實作或把ALLY排除後冒稱整屏相同。完整重疊HD仍未完成。

art-review.json接受單張#6：24×32來源(32,152)，HD72×96，墨跡外框[0,0,71,95]；紅眼高度24–29及左下突出部外框[0,63,11,83]恢復。紅眼左緣相差一個HD像素，色區外框相同不等於輪廓或RGB逐像素相同。#7／#8單格眼睛高度與突出部已改善，身體、腿部與跨姿勢一致性仍待審查，未選入。

source-manifest.json保存481項來源紀錄、473份副本、90份實際Go依賴，共53,977,400 bytes；原版八項只記SHA。source-snapshot/保存收尾更新前文件bytes。preserve-v3.py為成功來源；首版相對路徑及第二版假設dosgolem/go.sum存在，均在複製前退出，失敗腳本保留。dosgolem為標準函式庫模組，Go依賴metadata證實沒有go.sum需求，未改產品。內建image_gen完整提示詞、原生圖、正規化來源與SHA均留本機；GUI／DAT、其他sprite及交付未驗。

## 81. 歐格斯跨姿勢一致性修整

2026-10-03。沿用024 §1.7 READY，現行25筆及正式4/360以§80為準。本批候選入口為既有workplace/hd/redraw/的ENEMY01-group2-v3-及20261003前綴；生成紀錄、來源差分、正規化腳本、原生圖、72×96圖、provenance與比較均由本節索引。已驗收#6原生圖作編輯底稿，各自原版#7／#8只指定動作；不以另一姿勢取代原版動作，不平移、縮小或改位置。新候選尚未接受或選入，正常來源、重疊、GUI及交付範圍不擴張。

【confirmed，原版限定三姿勢】獨立ENEMY01.PBL解碼，#6→#7的77像素與#6→#8的84像素改動全部落在頭部x16–23／y0–12及左下突出部x0–4／y20–27。其餘軀幹、腿、腳與長手臂在原版逐像素相同；不因候選各自重畫便當作原版動作。原版SHA與檔案偏移沿用§80；色號陣列位址基準為24×32逐列索引。

內建image_gen新增四份原生1086×1448候選，完整整幅固定縮至72×96：07／08-consistent及各consistent-revised。generation-records與generation-records-corrections保存完整提示詞；normalization-source保存整幅Lanczos及PNG24方法，各版provenance記錄所有參照／輸出SHA。連同§80共19份候選，未替換現行主題。

獨立review與review-v2來源及JSON保存色區量測。首版#7眼睛y17–21、突出部y65–83，#8眼睛y19–24、突出部x0／y68–83，定位未通過。局部修正版#7眼睛y18–23符合原版高度，突出部外框[0,62,11,80]與原版[0,63,11,80]僅上緣一個HD像素差；#8眼睛[60,16,67,20]已改善，突出部[5,67,11,78]仍較原版[3,60,11,80]低，未接受。固定區5,616像素的RGB仍有3,181／3,185差，人工比較見軀幹與腿部造型較一致，不稱靜態RGB逐像素相同或正式美術通過。

比較入口ENEMY01-group2-v3-comparison-v2-20261003.png，三欄為原版／§80單一來源候選／本批局部修正版。首版montage因沒有可寫Fontconfig快取而SIGABRT；改用不需字型的convert整幅縮放、並列後成功，未改素材。正式ENEMY仍4/360，#7／#8的動作與正常HD仍未完成，不重跑不依賴新候選的原版來源RE。

## 82. 現行主題新盟友的有限視窗補驗

2026-10-03。原座標與圖層契約沿用024 READY，現行主題為theme-oogus-pose6-v2-20261003，仍25筆。工具、建置輸入與依賴metadata、二進位、Xvfb視窗擷取、F10 state、中文側檔、原版匯出、獨立完整合成與來源副本入口為workplace/hd/gui-current-ally-v1-20261003/。本批用既有正常07-first-play.state接續，驗新ALLY #0及既有背景的HD／語言切換與F11，不當作從開機、原版DAT、敵人GUI、全部sprite、幀率或交付驗收。§74的舊GUI及原版DAT結果維持原範圍。

【confirmed，限定六份真正視窗】目前原始碼重新建置前端，Xvfb／xdotool輸入、每鍵按住0.22秒；HD中文、HD英文、原版英文、原版中文、正常Up移動後原版中文及F11載回後HD中文，共六份960×600視窗，各保存12相位及F10 state、中文側檔、語言metadata。verify.py由原版七份PBL、現行25張PNG、正式text與兩個字型獨立重建全部像素，各至少一個實際游標相位完整相同，不符0，未排除區域。六份原版都含完整ALLY #0；三份HD樣本省略新ALLY負對照各4,340像素，HD及中文省略負對照亦有效。敵人本批未出現，不能因主題已載入便稱敵人GUI通過。

F10正式保存後正常Up使52 bytes玩家區與保存前不同，F11還原後與保存前完全相同；量測用F10不覆寫待載回的快速存檔。此負對照確保F11沒有作用時驗證會失敗。只驗明示玩家區，不稱自然GUI全部44機器欄位／RAM／亂數一致。前端quit-after正常退出0，按鍵紀錄及終止收據完整。初始debug state沒有既有中文側檔，未重新印出的方位North仍為原文，本批只核對已建立的正式中文，不證明全中文玩家路徑。

工具版本Go1.24.13、Python3.11.2、ImageMagick6.9.11-60；Docker影像SHA `083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7`。原版色號為320×200，GUI960×600，玩家區為dosgolem執行期線性0x16966、52 bytes，原版PBL檔案偏移不與之混用。首版Go建置gcc fork失敗，未限制Go並行度為強推論原因；相同image及原始碼加GOMAXPROCS=1、-p1乾淨重跑成功，build-v1-failure.json保留。正式runtime與原版未改，高負載不作幀率或音訊時序聲明。

preserve.py保存683項精確來源、675份副本、346份實際非標準函式庫Go／C／內嵌依賴，共69,318,887 bytes；含§81候選、完整生成紀錄及本批六份GUI／F11來源。原版八項只記雜湊。入口source-manifest.json與source-snapshot/，保存收尾前文件bytes；新盟友GUI有限補驗完成，原版DAT、敵人GUI、全部sprite與交付仍未完成。

## 83. 歐格斯末姿勢的突出部局部修整

2026-10-03。沿用024 §1.7 READY及§81的獨立原版動作區域，以08-consistent-revised原生圖作編輯目標，只伸長小突出部的向上青色尖端；#7另限定修嘴部下唇，軀幹、長手臂與腿部不改。原版嘴部參照為#7來源x17／y7、7×6格的完整放大裁切，參照只定位嘴部，不替換正式素材。候選原生圖、72×96圖、完整提示詞、provenance、正規化、獨立審查與比較均由既有workplace/hd/redraw/的ENEMY01-group2-v4-及20261003前綴索引；未接受前不替換現行25筆主題。現行正式4/360與GUI範圍以§80／§82為準。

本批內建image_gen新增08-tip、07-lip及07-mouth-guide三份候選，連同前批共22份。原生1086×1448整幅縮至72×96，未裁切或平移角色。各版generation-records保存完整提示詞，provenance保存參照及輸出SHA；review-source與review保存獨立原版色區量測，progress-source與progress-review保存本次嘴部核對、候選雜湊及進度同步方法。本節只記候選修整，沒有新增正式美術或正常呈現驗收。

【confirmed，限定色區量測】#8突出部由[5,67,11,78]改善為[5,62,11,79]，原版三倍為[3,60,11,80]；上緣偏低由7個HD像素減至2個，左緣仍差2個，未接受。#7兩次修嘴均未達到原版下唇y33–35：新圖下唇仍在y29–31，y33–35全黑。原版來源#7檔案偏移2890，x17–23／y7–12獨立色號核對；原版SHA沿用§80。07-mouth-guide改變嘴部形狀，沒有修正垂直位置，不以提示詞宣稱成功。

現行25筆與正式ENEMY美術4/360不變，#7／#8均未選入。Python局部修圖的必要明示授權已提出，尚未收到回答；未以等待時間作授權。此處不阻止其他已授權的HD工作。

§82冷載限制的原因補充：cmd/psychicwar/main.go的-load-state啟動路徑只載入機器狀態，沒有還原中文側檔的呼叫。即使另有側檔，也不能據此稱啟動時已還原中文；實際F11還原結果維持§82的限定證據。本次只核對程式路徑，未改產品。

## 84. 敏頓三姿勢的實際視窗補驗

2026-10-03。沿用024 §1.6 READY、§79正常敏頓路線與現行25筆主題。本批工具、輸入雜湊、擷取、F10狀態、中文側檔、原版匯出、獨立核對及來源保全入口為workplace/hd/gui-minton-v1-20261003/。沿用§82已建置前端，先核對實際專案依賴與二進位雜湊；以真正F11還原§79已核對的reload-event00.state及其中文側檔，繼續原版動畫。F10後短暫SIGSTOP只固定擷取時刻，不改機器資料；SIGCONT與終止由有界程序負責。驗收先核對完整72×96角色矩形與三個原版姿勢，再核對包含提示列的完整畫面；沒有同狀態證據的擷取保留並標未驗，不用不同姿勢圖面湊通過。其他擴張依實際證據另列。

【confirmed，限定36份完整視窗】現行theme-oogus-pose6-v2-20261003、960×600真正視窗，共40份F10擷取。v2/verified-with-toast.json證實36份全部像素不符0，包含HD17份、原版19份，敏頓ENEMY00 #6／#7／#8在中英文模式兩側均出現。期望由原版七份PBL、現行25 PNG、正式text與字型、保存原版frame及全域8×8遮罩重建，包含依輸入順序與兩秒契約決定的F10／F5提示列，未排除提示列、盟友、游標或其他區域。三姿勢省略敵人負對照分別4,867／4,888／5,556像素；錯姿勢負對照4,044／4,044／5,095像素，均有效。

另四份a-hd-chinese-00／01、b-hd-english-01、d-original-chinese-07未驗證。三份原版來源中途或無法完整辨識；b-hd-english-01完整原版#7與實際視窗角色差4,181像素，無法證明保存與擷取處於同一幀。這四份原始PNG、state及差異全部保留，不能當作通過或斷言產品缺陷。

首跑使用replay-event00.state及研究用metadata，F11恢復指令數時鐘後RunCycles退出。這份準備輸入不是前端F10存檔；按failure-note.json分類為量測前置條件不符，未改產品。第二跑改用§79已逐項核對的reload-event00.state，SHA `6579c73f8c3411ccfecd3d9e3fffd3fc803ab77278dd263e09f2d477b2f9a940`，由既有pwstep的SetDOSBoxCycles設定時鐘，其餘資料與中文已驗。真實F11之後繼續原版動畫，前端退出0；不稱從開機、原版DAT、完整RAM／亂數或自然硬體時序對拍。

第一版verify.py未建模前端短暫提示列，36份完整角色矩形已吻合，只有一份全屏吻合，其餘固定差13,782／13,848像素。核對cmd/psychicwar/main.go的drawToast及apps/psychicwar/help.go的DrawTextPage後，verify-with-toast.py由正式cjk24、HelpCols38、黃色前景、黑色底與按鍵順序獨立重建完整提示列。連續F10間隔均小於兩秒；第一份英文切換樣本使用F5提示，其餘由前一份F10決定，沒有從實際PNG取得期望。第二版36份全屏不符0，首版收據及腳本保留。

沿用§82前端SHA `da3a3ca23db75bb040a6c138da6ffc187e8ab3761aa0068c91cca8e5415d5e6c`，117份目前專案來源／二進位與前批保全SHA相符。Docker、Go與Python版本沿用§82；本批preserve.py另存776項來源、768份副本、346份實際非標準函式庫Go／C／內嵌依賴，共46,019,994 bytes。原版八項只記SHA，來源快照保存在最終進度更新前。現行25筆及正式美術4/360不變，歐格斯GUI、最新素材DAT、其他重疊、全部sprite及交付仍未完成。

## 85. 首組末姿勢的前臂對位修整

2026-10-03。沿用024 §1.6 READY、§75.2未通過的ENEMY00 #2候選，限定修整交叉前臂及所附色區；不平移、縮小或裁切整個角色。獨立原版ENEMY00.PBL的#2檔案偏移768、24×32逐列色號為依據，第一白色帶x5–10／y18、第二白色帶x10–14／y19；原版SHA沿用§75。完整原版、臂部參照與v10原生圖作內建image_gen的不同角色輸入。既有workplace/hd/redraw/的ENEMY00-group0-v11-及20261003前綴索引原生圖、整幅72×96正規化、完整提示詞、來源／輸出SHA、獨立色區量測、比較與進度同步。候選未通過審查前，現行25筆與正式美術4/360不變。

v11內建image_gen以v10原生圖、完整原版與臂部裁切三份參照編輯，輸出1086×1448整幅縮至72×96。x15–32的白色色區仍集中在y50–51，原版y54–56範圍仍為0；沒有修正前臂高度，未接受或選入。review.json記錄閾值、來源與輸出SHA；此量測只定位白色色區，不當作輪廓逐像素對拍。停止此位置的重複局部編輯，Python局部修圖授權問題尚待回答；先推進其他已READY的sprite。

## 86. ENEMY00 #3單姿勢的原版參照重畫

2026-10-03。沿用024 §1.3／§1.4 READY及§35–36正常循環；只重畫原版ENEMY00 #3完整24×32，位置(32,152)及HD72×96不變。既有workplace/hd/redraw/的ENEMY00-group1-v5-及20261003前綴索引完整單格原版參照、原生圖、整幅正規化、提示詞與SHA、獨立來源及造型審查。正常呈現與載回等驗證工具、收據與選入副本使用workplace/hd/art-group1-pose3-review-v1-20261003/，接受前不替換現行主題。#4／#5的跨姿勢修整及完整動作仍未完成，不能以一張審查通過便稱整組完成。

選入準備入口tools/hd/prepare_group1_pose3_theme.py；新本機主題workplace/hd/theme-group1-pose3-v1-20261003/只換ENEMY00-03.png，其餘24張及manifest維持。單張接受條件、來源保全與選入紀錄仍由本節的審查目錄索引，完整HD與素材公開權利未完成。

【confirmed，原版參照及限定正常呈現】#3檔案偏移1118，24×32；432×576原版參照248,832像素不符0，單像素變更負對照差1。新圖1086×1448整幅縮至72×96，亮度門檻60的墨跡外框[0,0,65,95]與原版三倍相同。人工審查抬高右臂、向左下伸出的左臂、大頭比例、藍／青／白色護甲、小紅色腰部及分腿靴子保留；細部輪廓及抗鋸齒重畫，不稱輪廓或RGB逐像素相同。只接受#3，第4／5張跨姿勢一致性仍未完成。

normal.json從06-name／86AFh正常K／A／I／Enter及前進前綴到86,000,000步，核對16次來源bytes、原版前後整屏及往返動作。HD兩側在16次返回點及終點的1MiB RAM、原版frame、暫存器與cycles相同。26份中途按8×8目標來源遮格、HD開關與16份真實state載回通過。背景比較工具使用同一圖面實作，只稱內部一致性；independent.py另由原版七份PBL、候選25 PNG、正式字型及中文側檔重建16份全部960×600像素，不符0，未排除區域。新#3出現四次，省略敵人負對照各4,279像素，錯姿勢各4,663像素。

art-review.json保存準備階段，art-review-accepted.json記單張接受及正常閘門；selection.json保存來源與26份manifest／PNG輸出SHA。現行本機改為theme-group1-pose3-v1-20261003，25筆只換ENEMY00-03.png，其他24張及manifest不變，敏頓三張、歐格斯#6與新ALLY #0保留。正式ENEMY美術5/360，其他355張未完成。新#3 GUI／DAT、其他重疊、#4／#5美術與完整HD交付未驗。

工具版本與Docker影像沿用§82，Go1.24.13、Python3.11.2、ImageMagick6.9.11-60，離線單CPU；開工load14.29，不作幀率或音訊時序聲明。初次原版PNG匯出誤用不存在的pbl.write_png，在產物建立前退出；核對API後改既有pbl.png，同一工具鏈重跑來源差0。preserve.py另存603項精確來源、595份副本、90份實際Go依賴，共53,245,686 bytes，包含§85失敗候選、兩批完整提示詞及參照SHA；原版八項只記SHA，快照保存最終進度更新前文件。原生圖與本機PNG由redraw對應前綴索引，來源保全入口source-manifest.json。

## 87. ENEMY00 #4／#5與已接受第3張的動作造型修整

2026-10-03。沿用024 §1.3／§1.4 READY，原版ENEMY00.PBL SHA及位置同§86。每張先獨立解碼完整24×32原圖，僅以§86已接受的第3張作造型與線條參照，原版各圖號決定手臂動作。保持原座標(32,152)及72×96，不平移、縮小角色或裁切修正偏差。

本批入口為既有workplace/hd/redraw/的ENEMY00-group1-v6-及20261003前綴，包含prepare-references.py的日期化檔名、原版單格與放大參照、完整提示詞、原生生成圖、整幅正規化、造型審查及SHA。內建image_gen逐張編輯；接受前不替換現行theme-group1-pose3-v1-20261003的25筆主題，正式ENEMY美術5/360維持。若候選通過，限定正常循環及載回的工具、收據、來源副本使用workplace/hd/art-group1-pose45-review-v1-20261003/，選入版本須另記於本節。

開工load41.68，離線單CPU，不作音訊、幀率或自然時序聲明。待審查結果追加本節；不得以候選產出或尺寸正確當作完成。

原版#4偏移1512、#5偏移1899，三份432×576參照各248,832像素不符0，單像素負對照差1。#3→#4有88像素、#4→#5有98像素變動，#3→#5有131像素變動；原版末八列在三張之間完全相同。v6第4張右側x57–71的亮度門檻60指尖上緣為HD y18，原版為y9，明顯偏低而拒絕。以該候選局部編輯的v7前綴亦由本節索引；只修右臂，保持其他部分。第5張仍使用v6前綴，獨立依其原版動作重畫。

v7第4張同一右側區域上緣y5，較原版y9高4；v6第5張右側上緣y3，較原版y15高12，右外緣x65也未達原版x71；左側x0–11上緣y21，原版y27，均拒絕。三份候選末24個HD列的二值墨跡與已接受第3張相同，RGB仍有708／761／718像素差，不能稱原生圖逐像素保持。初版左側x0–14區域包括耳部，該值不當作指尖量測；初版工具保存為review-source-initial-20261003.py，後續改x0–11排除耳部。右手拒絕依據不變。

連續局部編輯未精確對位，停止此方法。重新核對§1.3／§1.4及美術指引後，v8第4張改以原版單格為主要構圖參照，第3張只提供造型與風格；v8前綴及20261003同由本節索引。仍不修改原版、主題或正式完成數。

v8仍沿用第3張的直抬右臂及下垂左臂，未採原版第4張動作，拒絕。本輪停止手部生成迭代，四份失敗候選、完整提示詞及原生圖留本機；Python局部對位授權未收到回答，未執行。現行第3張仍可繼續補驗實際視窗，不能因第4／5張尚未完成而宣稱整組完成。

v8獨立右側x57–71非黑區上緣y2，原版第4張為y9，仍高7個HD像素；左側區域上緣相同只定位手臂區域，不能證明手指與動作吻合。v8-review-04-20261003.json保存拒絕理由、量測與完整輸入SHA；不改現行25筆及正式5/360。

四份均由內建image_gen產出，未選入。存檔與完整提示詞入口：

| 候選 | 原生圖 | 72×96 PNG | 完整提示詞與參照 |
|---|---|---|---|
| 第4張v6 | [原生圖](../../workplace/hd/redraw/ENEMY00-group1-v6-generated-04-20261003.png) | [PNG](../../workplace/hd/redraw/ENEMY00-group1-v6-frame-04-20261003.png) | [記錄](../../workplace/hd/redraw/ENEMY00-group1-v6-generation-records-04-20261003.json) |
| 第4張v7 | [原生圖](../../workplace/hd/redraw/ENEMY00-group1-v7-generated-04-20261003.png) | [PNG](../../workplace/hd/redraw/ENEMY00-group1-v7-frame-04-20261003.png) | [記錄](../../workplace/hd/redraw/ENEMY00-group1-v7-generation-records-04-20261003.json) |
| 第5張v6 | [原生圖](../../workplace/hd/redraw/ENEMY00-group1-v6-generated-05-20261003.png) | [PNG](../../workplace/hd/redraw/ENEMY00-group1-v6-frame-05-20261003.png) | [記錄](../../workplace/hd/redraw/ENEMY00-group1-v6-generation-records-05-20261003.json) |
| 第4張v8 | [原生圖](../../workplace/hd/redraw/ENEMY00-group1-v8-generated-04-20261003.png) | [PNG](../../workplace/hd/redraw/ENEMY00-group1-v8-frame-04-20261003.png) | [記錄](../../workplace/hd/redraw/ENEMY00-group1-v8-generation-records-04-20261003.json) |

## 88. 現行ENEMY00第3張的實際視窗接續補驗

2026-10-03。現行theme-group1-pose3-v1-20261003共25筆及024 §1.3／§1.4 READY不變。入口workplace/hd/gui-group1-pose3-v1-20261003/索引準備及啟動腳本、現行來源核對、真實F11／F10、視窗PNG、原版state／frame匯出、獨立全屏合成、來源保全及進度同步。沿用§82已驗前端與§84視窗量測方法。

先以目前pwstep將§86正常event00 state設定DOSBox cycles並另存準備checkpoint，不寫入原版遊戲資料或亂數；這不是原版DAT或從開機。真正F11恢復中文側檔後驗兩語言、HD兩側。畫面及state若不同步、來源仍在貼圖中途，保留為未驗，不排除區域強行通過。高負載下只核對像素與限定資料，不作音訊、幀率或自然時序聲明。結果待追加。

準備首版向independent.json查詢未列入的state SHA而退出，尚未建立prepared目錄；實際state SHA在該收據已綁定的original-frames.json。prepare-v1-failed.py保留，改從正式匯出索引核對state、從獨立收據核對中文側檔，同工具鏈重跑成功。prepared/initial.state由正常event00接續wait:1，另存時鐘設定及目前圖面；實際步數83,215,609。

真正前端退出0，40份F10及視窗保存。首版獨立驗證在b-hd-english-09.meta.json的0 bytes處退出；擷取工具只等mtime變動，SIGSTOP可能停在中繼資料寫入前，該樣本不能證明語言及HD狀態。capture-metadata-review.json只確認這一份無效，其餘39份JSON可讀。verify-v1-failed.py與原始樣本保留；後續驗證將此份明列未驗，再核對其餘完整畫面，不改正式產品或排除任何畫面區域。

後續擷取入口另存run-save-complete.py：等mtime更新且中繼資料及中文側檔均為可讀JSON，才暫停擷取；輸出使用capture-complete/子目錄，避免覆寫本輪證據。此修正版尚未執行，不用它追認本輪樣本。

source-manifest.json及source-snapshot/保全本批GUI與候選來源；preserve-execution.py另逐項核對execution.json及preparation.json的實際輸入SHA，補存主manifest未列的前端、匯出器與準備中繼資料至source-supplement.json。補充副本也放source-snapshot/，不覆寫原manifest或既有副本；總數依兩份manifest合計。

【confirmed，限定實際視窗】40份保存樣本中36份完整960×600不符0，HD16份、原版20份；提示列由正式字型、輸入順序及兩秒契約重建，未排除區域。新#3有五份完整HD，中文兩份、英文三份；省略角色各差4,279像素，錯姿勢各差4,663像素。兩語言及HD開關依實際meta核對。第4／5張的技術呈現亦有樣本，但不因此接受其美術。

另四份不能稱完整畫面通過：a-hd-chinese-04及b-hd-english-07原版來源中途；b-hd-english-09中繼資料0 bytes；a-hd-chinese-00角色區吻合，但完整畫面差13,857像素，差異原因未證實。37份角色區通過只限其矩形，不取代36份完整畫面數字。真正前端退出0；資料由準備正常checkpoint後F11接續，不稱從開機、原版DAT、全部RAM／亂數、自然時序或正式封包。

812項精確來源、804份副本、347份實際非標準函式庫Go／C／內嵌依賴，共58,730,007 bytes；原版八項只記SHA。主manifest與補充manifest保留，來源快照為最終進度編輯前；sync-progress.py為本批現況回填入口。沿用Go1.24.13、Python3.11.2、ImageMagick6.9.11-60及§82固定Docker image，單CPU。正式5/360及現行25筆不變，#4／#5美術及完整HD未完成。

## 89. Zellwal正常路線的其他角色來源

2026-10-03。上一輪第3張GUI補驗已完成，本批轉向尚未接入的sprite來源。入口tools/hd/observe_zellwal_sprites.go，工作區workplace/hd/zellwal-sprite-source-v1-20261003/保存建置、實際Go依賴、原版正常輸入、貼圖前後frame／384 bytes來源、state、獨立解碼及保全收據。只讀0161:8705／8751及既有戰鬥進出位址，不強制呼叫、不寫遊戲資料。

正常19-zellwal.state SHA-256為727873bc31df7737f3a825af4a70efa6aed2ec28a04d6d4de3115fddd700df38，原版frame SHA為9b695972cfab83f569d27dfa4faea2cf0f0a3cf276301b7b1998856a61dca65f。執行前核對並讀取原版seed，固定六次Up與有界推進；帶觀察與無觀察兩側相同輸入、同一初始state。每張依實際檔名、圖號及尺寸辨識，不用區域或相鄰圖號猜角色語意。原版、RAM及state留本機。

工具與新來源尚屬研究；現行25筆、正式5/360及既有READY範圍不變。來源證據充分後才另審查接入契約；不修改效果§1.5或迷宮§1.13的DRAFT。開工load20.39，單CPU，不作即時音訊或幀率聲明。本地建置為workplace/hd/observe-zellwal-v1-20261003，實際依賴清單為workplace/hd/zellwal-source-go-dependencies-v1-20261003.json。

本批執行結果已核對：620,000,000至670,000,000步，固定F95Bh及六次Up，區域3由(2,0)朝北至(2,1)朝南，HP28、能量6未變。角色貼圖0次，戰鬥進入及返回均0次，沒有新來源可驗收。觀察與控制兩側終點完整1MiB RAM、原版frame、全部暫存器、steps及cycles相同；本次核對保存end.frame SHA及觀察工具SHA與observation.json相符，不冒稱重跑遊戲。來源與路線是否足以觸發其他角色仍未知，下一步確認正常可通行方向，不增加正式5/360或25筆接入。進度同步正文保存在本目錄issue-progress-body.md，只含現況文字，不含原版資料。

## 90. Zellwal轉向後的正常sprite來源

2026-10-03。§89六次Up未取得來源，本批接續其observed-end.state，不改原版記憶體或重擲seed。研究工具tools/hd/observe_zellwal_route.go以明示起始state／frame／SHA及最多16個原版方向按鍵觀察來源，逐鍵記錄位置與通行旗標。工作區workplace/hd/zellwal-sprite-route-v1-20261003/的turn-east、advance-east等具名子目錄只放各段正常輸入及前後來源收據；source-observer-v1.go保留前版工具原始bytes，observer-source.go保存本批工具。建置與Go依賴分別為observer與go-dependencies.json。未知牆壁位元不猜補；先正常左轉觀察，再依遊戲回應選擇可行方向。每段控制使用相同初始state、seed及按鍵。候選、完整HD與現行READY範圍不因此改變。

本批獨立核對入口tools/hd/verify_zellwal_route.py，各段independent.json依原版PBL完整解碼與實際384 bytes貼圖來源重建64000 bytes整屏，不拿觀察器的圖號當期望。advance-east-four是已核對東向可通行後的四次Up路線收據。來源保全工具及輸出將存於本工作區preserve.py、source-manifest.json、source-snapshot/，只複製明確來源與輸出，原版只記SHA。

獨立核對已證實四次ENEMY03 #3→#4→#5→#4來源，首筆AL00，其餘AL01；第五筆ALLY原始來源等於#0，但被額外XOR成分遮住，完整目標身份仍未知。五筆整屏不符0，單nibble負對照均1像素，三筆敵人XOR錯模式差466／467／465像素。首筆黑底錯模式差0，不冒稱有效。原版正常終點HP0、戰鬥進出各1次；#4→#3尚無呼叫證據。首版驗證漏掛/output，次版將收據寫到唯讀/src；補齊既有可寫/output並指向该路徑後，同image同程式重跑通過，產品與來源未改。

另以同一advance-east起始state及F95Bh，使用原版Up後按住Space至第10,000,000步，正常攻擊分支只為取得剩餘動作，不強制勝利、不改血量、不重設seed。工具tools/hd/observe_zellwal_battle.go，工作區advance-east-attack、observer-battle及go-dependencies-battle.json；observer-battle-source.go保存原始bytes。新分支的完整結果尚待驗證，不以先前死亡或未出現圖號猜補差分边。

正常空白鍵攻擊分支已完成，五次敵人來源#3→#4→#5→#4→#3，再一次ALLY raw #0貼圖。六筆原版完整64000 bytes前後frame不符0，六筆單nibble負對照均1像素；敵人四筆XOR錯模式差466／467／465／467像素。第六筆ALLY完整來源身份仍未知，僅確認raw bytes及完整前後畫面。兩側1MiB RAM、原版frame、全部暫存器、steps、cycles相同；終點區域3 (4,1)、HP0、能量3，戰鬥進入及返回各1次。四條敵人有向差分已有實際正常呼叫，來源證據已足夠，停止此角色的逆向切片。技術來源清單審查後掛入024 §1.14 READY；實作、中途HD、正式美術、GUI與DAT仍未完成，現行25筆及5/360不變。

本批精確來源保全173份副本、79份實際Go／C／內嵌依賴，共15,288,403 bytes；原版14項只記SHA。source-manifest.json保留收尾進度更新前文件bytes，舊來源不覆寫。本輪GitHub正文放本工作區issue-progress-body.md，只含進度文字；驗證／型別／來源雜湊及研究工具透過本節索引回查。

本批工具鏈沿用psychicwar-go-ebiten:latest，映像sha256:083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7；觀察收據記錄Go1.24.13，獨立Python3.11.2。開跑load11.33、離線單CPU，只做決定性指令數驗證，不作音訊或幀率時序聲明。收尾173份快照及14項原版SHA相符，現行26份主題檔不變；本輪source-manifest保留此前文件bytes，現況以CONTEXT及唯一worklist為準。

## 91. ENEMY03三姿勢的限定HD接入

2026-10-03。依024 §1.14 READY實作ENEMY03 #3／#4／#5來源，四條有向差分由§90真實原版呼叫支持。研究輸出workplace/hd/zellwal-hd-runtime-v1-20261003/保存正式程式寫前bytes、建置／實際Go依賴、正常HD／中文圖面、中途state、載回及獨立驗證、來源保全與GitHub進度正文；before/保存寫前來源。工具tools/hd/verify_zellwal_hd.go、tools/hd/verify_zellwal_hd.py及tools/hd/prepare_zellwal_theme.py均由本節索引。候選分支workplace/hd/theme-zellwal-candidate-v1-20261003/只在現行25筆旁新增ENEMY03三張既有72×96候選，不覆寫現行主題。候選雙側白青圓塊位置及造型仍待美術審查，不作正式接受；正式5/360與現行25筆不因技術測試增加。

開工load17.99，離線單CPU；工具鏈沿用psychicwar-go-ebiten:latest，image sha256:083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7，Go1.24.13、Python3.11.2。原版、state及候選留本機，不改EXE、規則、亂數或存檔，不作音訊／幀率時序聲明。效果及迷宮仍DRAFT；#34維持OPEN。

新增來源驗收apps/psychicwar/theme/zellwal_test.go核對真實三姿勢、四差分及錯雜湊／跨檔／重疊／未支援圖號負對照。首輪全包測試缺既有/hd/bg.idx掛載，TestThemeRealBackground失敗；補上已存在的workplace/hd唯讀掛載後同image同程式重跑，35項通過、1項依前置條件略過，不改產品。unit.jsonl與unit-v2.jsonl保留。正常驗證保存每次Frame的原版frame及覆蓋sprite槽的貼圖入口／返回，獨立期望可追身份生存期，不排除盟友區或用正式HD圖面作答案。輸出normal-attack及normal-death子目錄；舊來源掛/output唯讀，本批輸出掛/hdout可寫，避免改寫前輪證據。

正式程式限定接入已實作：enemy.go新增ENEMY03來源SHA及3↔4↔5的四條已證實有向差分，theme.go同步載入、尺寸／位置與全黑格透明判斷。沒有改共用dosgolem或原版。技術候選28筆只新增三張既有PNG，現行25筆不覆寫。

兩條原版正常分支已驗：攻擊18樣本及未攻擊15樣本，共33份完整960×600圖面、中文字面、合成及33份真正state載回的獨立期望不符0。含18份貼圖中途；每份載回再接續100000步，HD與原版RAM／frame／全部暫存器／steps／cycles一致。兩條正常末段與§90原版終點相同，均HP0，不強制勝利。29份樣本的I_ENMY03.BIN:0052與正式資料「葛雷戈林」一致。冷載中文只證明新印字，未恢復前綴中文側檔，不稱全中文、從開機、原版DAT、GUI或封包驗收。

獨立期望讀取429次Frame的原版畫面及11次覆蓋slot的貼圖入口／返回，依原版PBL、實際384 bytes來源、候選PNG、正式text及字型重建，未排除盟友或其他區域。只有首姿勢#3兩份完整HD，省略新敵人負對照各6912像素；其他受遮姿勢依READY回退。21份正常／冷載圖面不同，差異1728／2304／3456像素等落在盟友身份失去的區域；各自完整期望已核對，不用連續觀察身份替載回猜補。這仍是完整重疊HD的限制。

來源保全工具及輸出為本工作區preserve.py、source-manifest.json、source-snapshot/；GitHub進度正文為issue-progress-body.md。保留舊來源§90的173份快照，不覆寫。候選三張白青圓塊與原版位置的差異仍待修整，美術未接受，正式5/360不變；§1.14維持READY，GUI／DAT、完整動作及美術尚未CONFORMED。

本批source-manifest.json保存948份精確副本、90份實際Go／C／內嵌依賴，共83,076,442 bytes；原版18項只記SHA，前批173份來源快照逐項未變。快照文件為最終進度更新前bytes。另逐像素核對21份正常／冷載差異，全部落在ALLY原版矩形三倍後的(792,456,72,96)，沒有其他區域差異；不把身份未知的回退改成產品缺陷或完整HD通過。

## 92. ENEMY03單姿勢美術修整

2026-10-03。前批§91來源接入及正常畫面有限驗證已完成，本批只修整白青圓塊與姿勢，不重開來源RE。依024 §1.14 READY與既有美術定案，以原版ENEMY03.PBL #3／#4／#5獨立解碼，原檔SHA沿用§90；單格24×32完整參照用nearest放大18倍至432×576，沒有修圖。來源偏移、色號SHA與參照SHA見workplace/hd/redraw/ENEMY03-group1-v2-source-check-20261003.json。內建image_gen單格候選原生圖、72×96整幅規格化、600%檢視圖及完整提示詞由同目錄ENEMY03-group1-v2-provenance-03-20261003.json索引；原生圖不覆寫舊候選。僅以ImageMagick整幅縮至72×96，不裁切／平移角色，不執行Python修圖。候選尚未接受或選入；現行25筆、正式5/360維持。第4／5張及正常HD驗證將依相同單格流程補齊。

本節追加候選索引：ENEMY03-group1-v2-generated／frame／preview／provenance-04／05-20261003，以及三張ENEMY03-group1-v3-generated／frame／preview／provenance-03／04／05-20261003。第3張v3左圓塊白緣由HD y33修到原版y36；第4張v3縮短中間彎條並抬高右下鉤，第5張v3下移前方彎條及短鉤。v2／v3測量記錄為同目錄ENEMY03-group1-v2／v3-measured-03-20261003.json，RGB分類只供導覽，不能單独證明像素忠實度。

本批正常驗證工作區workplace/hd/art-zellwal-group1-review-v1-20261003/，theme/只替換技術候選28筆中的三張ENEMY03 PNG；before/保存驗證工具修改前bytes。tools/hd/verify_zellwal_hd.go新增明示-theme選擇，用同一§90原版收據驗正常HD，不改正式產品程式。normal-attack／normal-death及go-dependencies.json、verify-normal、source-manifest.json／source-snapshot/、art-review.json與selection.json由本節索引，結果待完成後補記。

三姿勢新版28筆候選已完成正常兩分支33份完整合成、18份中途、33份真正state載回及接續，獨立不符0。只有第3張兩份完整HD，省略負對照有效；第4／5張仍重疊回退。art-review.json只接受第3張的造型及有限正常呈現；第4張墨跡右邊差1個HD像素及曲線、第5張短鉤下緣仍須修整，不選入正式主題。新26筆主題workplace/hd/theme-zellwal-pose3-v1-20261003/沿用現行25筆，只新增ENEMY03 #3；prepare_zellwal_pose3_theme.py保存可重現選入閘門，selection.json由本節索引。selected-attack／selected-death將驗實際選入26筆，independent-source.py等前版來源保存於before/，來源版本以各收據SHA識別。

26筆準備首版誤把sprite矩形寫成match，被正式載入器拒絕，兩分支退出1；這是候選清單準備錯誤，未改產品。核對已通過§91清單後，依READY省略match沿用已證實背景錨點，保留selected-theme／attack／death-v1-invalid-manifest及invalid-manifest.json，修正同工具鏈重跑。tools/hd/review_zellwal_art.py及art-measurements.json亦由本節索引；度量不修圖。

實際26筆selected-attack／selected-death已驗：18＋15，共33份完整960×600圖面、中文字面、合成與33份真正state載回的獨立期望不符0，含18份中途；每份接續100000步原版RAM／frame／全部暫存器／steps／cycles一致。29份正式「葛雷戈林」中文；第3張完整HD兩份，省略負對照各6912像素。兩條原版終點仍HP0，未重擲。prepare_zellwal_pose3_theme.py --select依已驗來源及實際26筆閘門保存theme-zellwal-pose3-v1-20261003，共26筆，只增加ENEMY03 #3。正式敵人美術6/360，其他354張未完成；第4／5張未選入。來源保全preserve.py、source-manifest.json／source-snapshot/及issue-progress-body.md由本節索引；本輪未改正式產品程式、原版、譯文或字型，未執行Python修圖。兩批33份不能當作全部敵人、完整中文、GUI／DAT或交付驗收。

本批保存1779份精確副本、90份實際Go／C／內嵌依賴，共161364328 bytes；原版18項只記SHA。source_version_bindings明示新版28筆收據使用的舊Python驗證器SHA與before/精確副本，不能用現行26筆工具版本代替。前批173份來源及948份技術接入快照均逐項未變。來源快照保存最終進度更新前文件bytes，現況仍以CONTEXT及唯一worklist為準。

收尾索引：同工作區final-audit.py／final-audit.json保存快照、原版雜湊、兩版來源綁定、新27檔及前26檔、原25張及欄位、擁有權掃描結果。首版內聯收尾已通過全部斷言並更新文件，最後輸出摘要時重用old字串造成TypeError；修正摘要變數後以同工具鏈乾淨重跑，與產品無關。

負對照口徑補充：同工作區negative-composition.json逐樣本另量測移除第3張HD而露出原版sprite的實際合成RGB差，區域(96,456,72,96)內中文alpha為0。此前6912表示圖面RGBA差異像素數；實際完整合成RGB負對照兩份各4499像素，以本收據分列兩種數字。首版量測假設全屏PNG是RGBA而退出，改用已驗RGB／RGBA正規化helper後同工具鏈通過，未修改圖片。

## 93. 現行26筆主題與ENEMY03第3張實際視窗

2026-10-03。§92新26筆及正式6/360已選入，本批補驗實際Linux視窗、F11／F10與適用的正常原版DAT，不重開來源RE、不改產品。工作區workplace/hd/gui-zellwal-pose3-v1-20261003/保存新建psychicwar-current／pwstep／export-state及各自go-dependencies、prepare.py／preparation.json與prepared/正常checkpoint、run.py／execution.json／captures.json／terminal.json、verify.py／verified.json／original-frames.json、正常DAT逐步紀錄、source-manifest.json／source-snapshot/與issue-progress-body.md。使用§92已驗ENEMY03第3張正常sample03.state及中文側檔，由現行pwstep wait1ms依既有API準備cycles checkpoint；真正視窗再F11接續，不稱從開機或自然時序。現行theme-zellwal-pose3-v1-20261003保持26筆，所有原版與源素材唯讀掛載。开跑load26.96、image sha256:083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7，不作效能或音訊聲明。正式6/360及第4／5張未完成狀態維持。全部sprite、美術、完整重疊、迷宮、效果与交付仍待完成。

首輪run-v1-failed.py將1MiB RAM誤當壓縮state檔案大小下限，F10資料已完整但量測條件拒絕，前端由工具有界終止、0份擷取，不列產品缺陷。實際gzip state87910 bytes及JSON保存完整，v1-failure.json與原始輸出留根目錄；改用gzip CRC／EOF完整性及JSON判斷後，同image同binary重跑至v2/，正式程式未改。

【confirmed，限定擷取與角色矩形】v2前端正常退出0，取得20份960×600實際視窗及完整F10保存資料。三份72×96角色矩形不符0：a-hd-chinese-00／01兩份中文HD，d-original-chinese-00一份中文原版。兩份HD省略角色負對照各4499像素，錯姿勢各4406像素。英文HD尚無來源可確定的完整樣本。

完整視窗通過0/20。上述三份全屏差異依序13476／275／275像素，其他17份因來源中途或未知維持未驗。verify.py已保存v2/verified.json後，於兩語言覆盖閘門退出1。差異原因尚未證實，不稱產品缺陷或GUI通過；提示列及其他區域未排除。v2/execution.json、captures.json、terminal.json、original-frames.json與verified.json為本輪實際收據，根目錄首輪輸出不混用。原版DAT工作尚未執行；本輪先依使用者要求更新CONTEXT與Issue #34，完整GUI、DAT及交付仍未完成。

本次進度同步收據由本節索引：同工作區issue-progress-body.md、issue-progress-before.json與issue-progress-after.json。Issue #34寫前全文／時間相同，寫後全文與body-file相同，2026-10-03T12:40:45Z維持OPEN。原版DAT逐步紀錄及本批source-manifest／source-snapshot仍為預定產物，尚未建立，不當作已有證據。

後續同步診斷：a-hd-chinese-01實際提示為「已讀檔」，保存檔則已是F10。正式Update先quickSave寫檔，再showToast，之後才推進及Draw；因此保存完成不能證明視窗已更新。v2完整差異保留，未修改期望來追認通過。續驗入口同工作區run-v3.py、verify-v3.py及v3/：每份先真正F11還原相同checkpoint，準備中繼資料只指定顯示語言，不修改原版state或seed；等保存完成後再給一次繪图機會，實際整屏是否吻合仍由獨立比對判定。F5與Shift+F5正常操作另保留記錄；不改正式產品。

v3前端與獨立驗證退出0，8份中7份完整960×600不符0，HD三份、原版四份；HD中文一份、英文兩份。首份a-hd-chinese-00角色區吻合但全屏仍差275像素，保留未通過。八份角色區不符0，四份HD省略角色負對照各4499、錯姿勢各4406像素。這是同一正常checkpoint經顯示語言準備與真正F11接續的限定核對，不稱從開機、全程中文或完整動作。觀察到起點已有未重印英文保留，正式側檔只驗其記錄的譯文，不能稱整屏文字全中文。

原版DAT續驗索引同工作區run-dat.py、dat-save-v1/與dat-load-v1/、dat-independent/、verify-dat.py及dat-verified.json。先用真正F11回到已驗原版checkpoint，再用原版F3脫離戰鬥；逐畫面選Esc／Options／Save Game，不寫HP或座標。載回由原版SELECT選單與LOAD GAME，另用pwstep自同一保存前GUI state走原版鍵序，產生獨立512 bytes DAT比較。結果未產生前不宣稱通過。

dat-save-v1實际F3已送達原版，後續自然推進仍走到Game Over；Esc擷取最後一份顯示遊戲選單／新遊戲，沒有Options或DAT。前端由量測工具有界終止，這條失敗玩家路徑保留，不用HP注入或重擲改成勝利。續驗採既有19-zellwal.state迷宮檢查點、現行26筆主題，工作區dat-prepared/、dat-save-v2/及dat-load-v1/；存讀檔範圍限迷宮與該主題，不因此證明新敵人戰鬥逃離。

dat-save-v2正常Up離開降落平台後，Esc／第五項功能選項／儲存遊戲／確認畫面皆已逐畫面核對；前端於300秒既有有界期限退出0，第四批命令未執行，不誤算成已輸入檔名。獨立原版由b-hd-current.state正常存出HD26.DAT 512 bytes。續驗工作區dat-save-v3/，以真正F11載回實際i-save-confirm.state及其側檔／中繼資料，接續檔名與原版保存確認；限定快照接續，不稱一次未中斷的玩家流程。run-dat-v1.py／run-dat-v2.py保留各已執行版本，現行run-dat.py只新增正常F11準備分支。

dat-save-v3實際HD26.DAT已與獨立原版512 bytes相同，SHA-256 ac6d2dff05041cb7a2fe2ba43d69383ee11e7c63c983d04d41efe0aedcaf475b。後續相位擷取遇到前端150秒期限，import等待已關閉視窗，外層210秒逾時退出124，terminal.json未完成；不能據此宣稱保存GUI退出0。八份已擷取相位保留，不補成十二份。工具run-dat-v3.py保留，現行run-dat.py為每個外部命令加10秒有界逾時。真正保存驗收重跑至dat-save-v4/，從已實際確認的j-save-name.state與側檔經F11接續，預先排入同一ASCII檔名鍵序與finish，以避免人工檢閱延遲碰到期限；不改原版資料或產品程式。

dat-save-v4正常保存與工具退出0，兩份完整十二相位擷取已保存，DAT與獨立原版相同。dat-load-v1原版LOAD GAME已執行完整十四個按下／放開事件，載回後中文與英文已擷取，但第三模式擷取碰到300秒期限；既有有界命令會退出，樣本保留不追認完整模式驗收。正式續驗dat-load-v2/預排同一已逐畫面確認的SELECT／LOAD GAME／HD26鍵序及兩種模式切換，正常quit-after140秒，工具outer210秒。run-dat-v4-before-auto.py與run-dat-v5-before-load-auto.py索引既有工具SHA；verify-dat.py只對實際完整相位與真實退出收據核對。preserve.py、source-manifest.json與source-snapshot/索引本批來源保全。

【confirmed，限定正常選單與完整像素】dat-save-v2／v4及dat-load-v2共12份真實960×600畫面，各保存12相位、至少一份全屏不符0，未排除任何區域。DAT512與獨立原版逐位元組相同，單byte負對照有效；保存前玩家52 bytes與獨立原版及三種載回模式相同。原版LOAD GAME十四個按下／放開事件符合Down／Return／H／D／Digit2／Digit6／Return，F5等熱鍵未外洩；保存名稱十個事件亦符合。保存及載回前端與工具退出0。保存經真實F11接續已確認的空檔名欄位，未覆寫原版資料或固定正式遊戲亂數；不稱未中斷流程、從開機、新敵人戰鬥DAT、完整中文或全部sprite。最終收據dat-verified.json，原版DAT SHA見上述；前批失敗各自保留。

來源保全1184份精確副本、348份實際非標準Go／C／內嵌依賴，共116941727 bytes；原版9項只記SHA。resolved_versions綁定各已執行工具的保存版本，不以現行run-dat.py代替舊SHA。來源快照是最終進度收尾前版本。最終同步入口同工作區issue-gui-dat-body.md、issue-gui-dat-before.json與issue-gui-dat-after.json；final-audit.py／final-audit.json記錄來源、主題及擁有權核對。

收尾首版已核對本批1184份，讀前批dict型files時誤以list解析而退出。final-audit-v1-failed.py保留；依舊manifest的snapshot／size欄位與其目錄基準修正讀取後，同image同命令重跑，不改舊manifest或副本。

第二版已核對本批1184與前批1779份，主題總檔案數假設27而退出。實際26 PNG、manifest及selection共28檔；回到路由與現行manifest，改由entry清單導出27份渲染輸入，selection另與已保全SHA核對，禁止忽略未知額外檔案。不改主題、素材或驗證期望。

## 94. ENEMY03第4／5張曲線與邊界修整

2026-10-03。上一批首姿勢GUI與26筆迷宮DAT已有限通過，不重跑已通過範圍。本批依024 §1.14 READY、原位置及同一造型定案，修整第4／5張候選，不重開來源RE。沿用只讀原版單格參照、內建image_gen與ImageMagick整幅72×96規格化，不執行Python修圖。

入口workplace/hd/redraw/ENEMY03-group1-v4-generated／frame／preview／provenance-04／05-20261003與來源核對ENEMY03-group1-v4-source-check-20261003.json。第4張原版下鉤紅色量測x45–62、y75–83，前版x43–65、y75–84；整體右界原版71、前版70。第5張原版短鉤紅色y75–77，前版只y75。以上色群是導覽，不取代人工造型及正常呈現審查。新候選與review-zellwal-v4-20261003.json量測結果從此入口追蹤，未產生或未接受前不選入正式主題。現行26筆、正式6/360及第3張接受狀態不变，完整重疊、其他sprite與交付仍待完成。

只讀核對工具入口workplace/hd/redraw/review_zellwal_v4.py，沿用tools/hd/review_zellwal_art.py的原版解碼、參照完整像素與色群量測，候選只指定第4／5張v4，第3張仍用已接受v3。不修改任何PNG或以量測自行接受美術。

v4三份完整參照共746496像素不符0，三份單像素負對照各差1。第4張下鉤紅色底緣84→83已回原版，但左緣43→42仍偏左，整體右界70仍少1個HD像素；第5張短鉤底緣75→76，原版77，仍不足。兩張未接受或選入。再各做一次局部端點修正，入口ENEMY03-group1-v5-generated／frame／preview／provenance-04／05-20261003與review_zellwal_v5.py／review-zellwal-v5-20261003.json；第4張右灰白突出部依原版碰到畫布右緣，不以額外留邊改變原版，第5張只調低短鉤。完整提示詞與原生圖保留，v4不覆寫。

依使用者要求先同步進度。第4張v5目前只完成原生生成，保存為ENEMY03-group1-v5-generated-04-20261003.png及provenance-04，SHA-256 fe3db27fc47b292ec354d64ca2784b06832ba8a42968703a3858139f6d687025；尚未規格化、量測或接受。第5張v5及v5量測工具／結果尚未產生，前段列名是預定入口，不能當成已有收據。現行26筆及正式6/360不變。

本次進度同步收據由本目錄issue-v4-progress-before.json、issue-v4-progress-body.md、issue-v4-progress-after.json及issue-v4-progress-audit.json索引，正文只含進度文字。原圖、提示詞、原版與state留本機；不重跑已通過的限定GUI／DAT，不改#44。

接續v5規格化與只讀審查：第4張原生圖已保存，整幅規格化的命令及SHA另存ENEMY03-group1-v5-normalization-04-20261003.json，不覆寫原生生成收據。第5張依v4及原版單格只修短鉤，原生圖、完整提示詞、72×96圖與六倍預覽依前段v5入口保存。review_zellwal_v5.py只改候選版本，原版參照與量測算法不變，尚未取得結果前不接受兩張美術。

v5只讀結果：三份原版完整參照746496像素不符0，三份單像素負對照各差1。第4張整體右界已到x71，但下鉤紅色色群[46,75,64,82]仍與原版[45,75,62,83]有差異；第5張短鉤紅色色群[49,75,56,76]，原版[51,75,59,77]，仍未達標。兩張不接受或選入，停止本輪這組端點生成，不以正常合成測試代替美術審查。

## 95. 首組敵人第0張的獨立美術與正常呈現審查

2026-10-03。沿用024 §1.6 READY、研究038 §48–49的正常來源，只審查ENEMY00 #0單姿勢；第1／2張、動畫一致性與完整特效仍另待完成。既有v7第0張為原版單格重畫，v8–v10沿用同一張；先核對原版比例、主要色區、下展手臂與端點，若不符合即保留候選，不因生成數量或其他姿勢曾失敗便跳過單張審查。現行26筆與正式6/360在審查、正常來源及獨立完整畫面通過前不變。

本批工作區workplace/hd/art-kasuruji-pose0-v1-20261003/保存measure.py／art-measurements.json、art-review.json、theme/候選、正常／載回收據、來源快照及選入紀錄。原版ENEMY00.PBL SHA沿用§75，單格參照ENEMY00-group0-v7-source-00-20261003.png、原生圖／提示詞與72×96圖沿用v7的完整provenance；不新生成或重畫其他姿勢。來源及原座標不變，僅替換審查副本ENEMY00-00.png，正式工具與主題暫不改。

只讀量測首版碰到既有4-bit調色盤參照，pbl.read_png只接受8-bit格式而退出；measure-v1-failed.py保留當時來源。新版以ImageMagick只讀解碼RGB串流，不寫回或修改參照；原版解碼、完整參照像素與負對照維持，屬驗證格式前置問題。

正常抽測入口run.py／execution.json及verify.py／independent.json，沿用§93保存並綁定來源的pwstep與export-state二進位。從§49真實kasuruji第0張state接續正時長等待，HD開關兩側用相同起點與等待；再由實際新保存state載回接續。完整原版frame、中文、26筆合成由原版PBL／PNG／字型獨立推導，不排除盟友或任何區域；保存機器欄位及DOS區段用既有獨立比較器核對，來源保全另由source-manifest.json／source-snapshot/索引。這不是新GUI或全部動作驗收。

【confirmed，原版參照與限定正常接續】#0原版檔案偏移60、24×32；完整432×576參照248832像素不符0，單像素負對照差1。HD墨跡外框[0,0,62,95]與原版三倍相同。人工核對雙天線與大頭、黑色下方面孔空洞、白灰青紅護甲、兩手向下外展與原端點、分腿靴子與原版畫布邊緣保留，只平順線條及有限陰影，不稱輪廓或RGB逐像素相同。

同正常event00.state接續1／5／20ms，三組HD兩側，另從真正新保存hd-5.state載回接續1ms，共八份完整960×600獨立合成不符0，四份完整第0張HD；未排除盟友或中文。四組全部44個機器欄位及DOS區段相同，CPU／RAM／port變更負對照有效。角色省略負對照每份3720像素，錯姿勢亦有效；實際執行與獨立驗證退出0，不改原版記憶體或seed。

只接受第0張，接受收據art-review-accepted.json；select.py檢查上述完整閘門後另存workplace/hd/theme-kasuruji-pose0-v1-20261003/，仍26筆，只換ENEMY00-00.png，原manifest及其他25張不變，selection.json保留來源與雜湊。舊26筆主題不覆寫。正式敵人美術由6增至7/360，其他353張及30張盟友仍未完成。第1／2張、整組動畫、新主題GUI／DAT、特效重疊、全部sprite與交付不由本次延伸。

來源保全入口preserve.py／source-manifest.json及source-snapshot/，保存本批執行工具、輸入與實際非標準Go依賴；沿用二進位的完整建置證據回查§93 source-manifest.json。遠端進度正文及回讀由issue-progress-before.json／issue-progress-body.md／issue-progress-after.json索引；final-audit.py／final-audit.json核對來源、現行與前批主題、擁有權與遠端全文。原版只記SHA，不上傳原圖、state或候選。

本批324份副本、91份實際非標準Go依賴、26513992 bytes已保全；前批1184份快照未變，原版九項SHA相符。source-manifest保存最終進度收尾前文件版本。Issue #34全文回讀與body-file一致，2026-10-03T14:36:22Z保持OPEN；第0張新主題GUI／DAT、完整動畫及其他sprite仍待完成。

## 96. 首組第0張新主題的真正視窗抽測

2026-10-03。沿用024 §1.6 READY及§95新主題theme-kasuruji-pose0-v1-20261003，正式7/360與26筆不變。工作區workplace/hd/gui-kasuruji-pose0-v1-20261003/保存run.py／run-source-v3.py、preparation.json、原版來源state、真正F11／F10視窗、原版零步匯出、完整獨立重建、來源保全及遠端進度。沿用§93精確保存來源的前端與匯出器，寫前核對二進位及實際Go依賴SHA；只把§95已驗正常hd-5.state及其中文側檔準備為快速讀檔，不注入原版數值或seed。

八份預定擷取含中英文HD與原版、語言及HD切換；每份均真正F11後F10，保存完成後讓Update／Draw前進，再用有界暫停固定視窗。以原版PBL、正式PNG／字型／text及F10保存state獨立推導全部960×600像素，包含前端提示列；不同步或未知來源保留未驗，不排除區域。此處只有原版checkpoint接續及首姿勢GUI目標，不稱從開機、整組動畫、原版DAT、全RAM／亂數或正式封包通過。

入口verify.py／verified.json、execution.json／terminal.json／captures.json、original-frames.json、source-manifest.json／source-snapshot/。外部命令設10秒逾時，Xvfb由有界程序及trap停止；高負載單CPU，不作速度或音訊聲明。

首版v1真正前端退出0、取得八份，但完整畫面0/8，只有第一份角色矩形一致，全屏仍差275像素，其餘七份角色／全屏不同步。verify.py保存verified.json後於兩語HD閘門退出1，不排除提示列或盟友。原版F10 frame仍完整第0張，實際視窗已出現射擊效果；cmd/psychicwar/main.go Update先hotkeys／quickSave，再RunCycles、Frame與Draw，保存完成不等於繪圖完成。

v2/run-v2.py改為外部暫停時排入真正F10與F11，恢復後保存再載回剛保存的state，最後提示依此輸入順序為「已讀檔」。只讓F11後Update／Draw短暫前進，再固定擷取，完整像素仍須獨立核對，未匹配就保留未驗。verify-v2.py、verify-source-v3.py及v2各收據由本節索引；只有真正前端與驗證器退出0且兩語完整HD門檻通過，才稱有限GUI通過。

【confirmed，限定視窗核對】v2前端退出0、八份擷取完整保存；八組F10／F11均由實際log證實同一次Update處理。六份完整角色矩形吻合，其中三份HD的省略角色負對照各3720像素、錯姿勢各4121像素。完整960×600畫面僅3/8不符0，均為原版模式；中文HD一份仍差13524、英文HD兩份仍差2301／3119像素。另兩份來源中途或未知，保留未驗。驗證器保存v2/verified.json後因兩語HD完整畫面閘門未通過而退出1，GUI仍未通過。未排除任何區域，未把角色區吻合延伸為全屏驗收；剩餘差異原因尚未證實。首版完整0/8收據與所有原始擷取保留。

本輪依使用者要求先同步CONTEXT.md及GitHub #34，不繼續修改擷取工具。現行26筆、正式美術7/360維持；第1／2張、完整重疊、其他sprite、迷宮與效果正式素材、新主題DAT及HD交付仍未完成。遠端正文與寫前／寫後回讀保存於本工作區issue-progress-body.md／issue-progress-before.json／issue-progress-after.json；#34保持OPEN，#44未改，原版、state及PNG未上傳。

後續全屏差異定位沿用本工作區，diagnose.py／difference-regions.json只讀八份已保存的原版、視窗與正式素材，依獨立全屏期望量測不同像素的位置及8×8格分布。診斷不修改原圖或降低verify-v2.py門檻，亦不另重跑CPU或重擲種子；結果用來區分擷取時序與載回後圖面辨識。

量測結果：中文HD首份的13524個不同像素全在y576–599提示列；英文HD兩份的2301／3119個不同像素全在ALLY #0矩形內。其餘三份完整原版差0。僅憑此位置仍不能區分前一個視窗畫格與本幀圖面錯誤。

繪圖觀察原型由prepare-draw-probe.py產生draw-probe-main.go，精確保留cmd/psychicwar/main.go並只在Draw結尾附加唯讀觀察呼叫。production檔案不改，原型依原版正常輸入推進，無規則、數值或seed注入。draw-probe.go最多保存每次請求16份繪圖當下原版frame／RGB／原始state／中文側檔與Ebiten ReadPixels原始RGBA及模式中繼資料。客戶端run-draw-probe.py依實際視窗與觀察RGBA完全相同才建立畫格對應，完整畫面仍由原版PBL／PNG／正式字型與text獨立核對，不能以觀察圖像自身當期望。此原型只診斷非同步擷取，不稱正式產品、自然時序或封包驗收。

replay-draw.go／replay-original.json重播每個觀察批次第一份實際state到其餘繪圖步數，唯讀匯出0161:8705貼圖入口的原暫存器、DS:BX來源bytes與0161:8751返回。各終點另存replay.state，與真實Draw原始state核對全部44個機器欄位及DOS區段，避免以相似畫面冒充同一路徑。verify-draw-context.py只從上述原版來源、024 §1.2／§1.3生命週期與正式Reference推導逐格期望，不讀Theme內部active旗標；負對照撤掉已證實角色與更換原版來源bytes。

【confirmed，觀察原型與原版重播】前端觀察原型退出0，取得44份Draw與四份完整視窗。原版重播44份frame／RGB逐位元組相同；44份replay.state的全部44個機器欄位及完整DOS區段與觀察時state相同，CPU／RAM／port負對照有效。獨立來源生命週期重建42/44份完整960×600畫面不符0，不讀正式Theme內部旗標；四份視窗分別與該模式第4份Draw原始RGBA逐位元組相同，中文HD／英文HD／中文原版／英文原版完整重建皆不符0。

這批原版入口證實角色區內的局部效果不覆蓋完整角色矩形。依既有READY規格，前面已證實的ENEMY00 #0與ALLY #0可保留其他完整吻合的8×8格。原art_plane只適用完整來源的獨立重建，將它直接用於整段GUI會整張撤掉角色，造成假差異。新的來源期望使用原版入口資料及初始完整原圖，PNG與Reference固定，未排除角色或任何區域。撤掉ALLY與變更其原版來源首byte負對照皆有效；更換敵人姿勢負對照有效。完整原型結果見probe-v1/context-verified.json，原始靜態期望與差異仍保留於probe-v1/verified.json及v1／v2。

其餘兩份b-hd-english-06與d-original-chinese-07仍各差13857像素，全在y576–599提示列。觀察器在ReadPixels之後才讀提示期限，可能跨過原本已繪出的提示期限；這是觀察時序限制，沒有從PNG猜回提示文字，也不把兩份列為通過。正常程式碼SHA仍為794565013a934a69e5cf7a98bde65104253b166d6fb58bb87b4688fb4dffcce0，原型只多一個唯讀觀察呼叫。結果不冒充無觀察器正式前端、實際封包、全部動作、原版DAT、自然時序或新效果素材通過；現行26筆及正式7/360維持。

工具前置失敗包括未初始化診斷empty畫布、零步匯出使用相對路徑、未載入text/help.json中的正式切換提示、Oracle.Close無回傳值，以及比較器路徑寫錯。核對實際API與正式資料後在同image修正重跑，均未改production、原版或畫面門檻。來源保全由preserve.py／source-manifest.json與source-snapshot/索引，保存實際建置依賴及研究工具，原版只記SHA；raw state／RGBA／視窗／機器收據亦在manifest列出雜湊，留本機。

本輪進度同步另存issue-draw-before.json／issue-draw-body.md／issue-draw-after.json，不覆蓋前次15:09進度收據；最終核對由final-audit.py／final-audit.json索引。#34仍OPEN、#44不改，不上傳原版或視窗圖像。

本批保存377份精確來源副本，347份實際非標準Go／C／內嵌依賴，共5458420 bytes；702項本機產物與素材雜湊列於manifest，原版九項只記SHA，前批324份來源快照未變。來源快照為最終進度收尾前版本。Issue #34寫前全文及時間未變，寫後正文與body-file一致，回讀2026-10-03T15:51:39Z，保持OPEN。這個差異定位切片證據已足夠，停止增加同類繪圖樣本，回到其他sprite美術與完整範圍；無觀察器正式前端與實際封包在交付驗收補齊。

## 97. 首組第1張單姿勢審查

2026-10-03。沿用024 §1.6 READY、§75.2既有v10第1張與§95已接受第0張作一致性參照。先以原版ENEMY00.PBL #1偏移420、24×32單格核對褐橙面孔條及黃色小亮點、前臂與手端、主要色区、整體比例，未完成審查前不換現行26筆或增計7/360。工作區workplace/hd/art-kasuruji-pose1-v1-20261003/保存measure.py／measurements.json、ImageMagick整幅最近鄰預覽、人工review.json；只讀量測，不用Python修圖。原版來源SHA沿用§95。若候選未符合動作，就保存差異而不接受；技術尺寸與單像素參照負對照不能取代美術審查。後續單張接受、來源保全、正常呈現與選入收據亦由本節索引。

2026-10-04接續。原版完整432×576參照248832像素不符0，单像素負對照差1；v10左臂區原版三倍外框[0,39,35,65]，候選閾值32外框[5,39,35,65]，右臂原版[36,39,53,65]、候選[36,39,55,65]。整張原版[0,0,59,95]、候選[3,0,58,95]，左靴區外框相同，右靴最右差1個HD像素。這些閾值只定位幾何，不能代替輪廓審查；原版左臂灰色外緣確在原版第0欄、第14／15列，不能因先前未完成手部審查而直接接受。

v12僅以內建image_gen修第1張前臂，輸入v10原生圖為編輯目標、原版單格為姿勢與色區權威。原生圖、完整提示詞、整幅72×96規格化與SHA保存於既有redraw的ENEMY00-group0-v12-及20261004前綴。不移動或縮小整個角色，不用Python修圖；取得圖後再以同一原版位置核對，未通過前不選入。

v12左臂邊界x5改善為x3，仍未到原版x0；右臂x55縮至x52，原版x53。整體外框仍[3,0,58,95]，兩側臂輪廓未符合原版，不接受或選入。停止以v10為底的局部修臂；此方法保留了原本偏窄的上臂外形。下一份v13改從原版第1張單格完整重畫，第0張已接受素材僅作造型與線條參照，不借用其手臂姿勢。原生圖、提示詞與量測沿用redraw的v13及20261004前綴，仍只做候選。

2026-10-04進度核對：v13已由內建image_gen產生，來源為`/home/anr2/.codex/generated_images/01a0f29c-ca42-7e30-8785-cd44d66ea00d/exec-0f257b86-571a-4a85-bac1-e478bd4908cb.png`。尚未複製到本專案、整幅規格化或審查，前段v13前綴是預定保存位置，不代表已有產物或量測。依使用者要求先同步CONTEXT.md及GitHub #34，現行26筆與正式美術7/360維持。同步收據沿用本工作區的`issue-pose1-progress-before-20261004.json`、`issue-pose1-progress-body-20261004.md`及`issue-pose1-progress-after-20261004.json`；只同步文字，不上傳原版、state或候選圖。

v13已接續保存至redraw版本化檔案，完整提示詞及來源／輸出SHA見ENEMY00-group0-v13-provenance-01-20261004.json。measure-v13.py／measurements-v13.json以同一原版參照核對，248832像素不符0、單像素負對照差1。整體及兩側臂外框已與原版相同，但左靴外框[0,78,35,95]原版為[3,78,35,95]，右靴下緣y90原版為y92。人工核對見左靴新增白色塊，整體仍保留大塊原版階梯輪廓，未符合一致的細密重畫。未接受或選入；本輪停止首組第1張迭代，返回其他已READY姿勢。審查收據為本工作區review-v13.json。

## 98. 歐格斯第7張下唇位置修整

2026-10-04。依024 §1.7 READY、§81原版限定動作區域及§83下唇差異，接續既有第7張，僅重畫頭部嘴部。原版ENEMY01.PBL #7檔案偏移2890，下唇在原版y11，三倍y33–35；目前候選仍在y29–31。製圖沿用內建image_gen，原版單格是姿勢與色區權威，已接受#6只作造型一致性參照，不複製閉嘴姿勢。

本批沿用既有redraw目錄的ENEMY01-group2-v5-及20261004前綴保存原版整幅最近鄰參照、原生圖、完整提示詞、整幅72×96規格化、預覽、SHA及只讀量測；審查與正常呈現證據沿用art-oogus-review-v2-20261003工作區。生成與量測不改production、原版、位置、比例、seed或存檔格式；未完成美術及正常來源審查前不替換現行26筆或增計7/360。

v5-07-mouth已保存，原版參照248832像素不符0、單像素負對照差1。新下唇僅降到y30–32，原版y33–35仍全黑；紅眼[59,18,67,23]保留，左下突出部[0,62,11,80]仍比原版高一個HD像素。人工審查嘴部仍不符，未接受或選入。編修區外3514個RGB像素不同只表示生成與縮圖改變細節，不能稱固定區逐像素相同。停止本輪局部修嘴，不依提示詞追認成功；量測入口ENEMY01-group2-v5-measure-07-20261004.py及measurements JSON。

## 99. ALLY全量候選美術盤點

2026-10-04。沿用024 §1.1完整範圍，先把其他盟友美術從「31張尺寸合格」推進到逐張原版參照與候選的幾何、主要色區審查。這是美術準備，不擴充§1.2只接受ALLY #0的正式來源契約。

輸入ALLY.PBL SHA為c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219，31張。實際#0–#15為24×32、#16–#30為16×16；相同畫布尺寸不能證明全部是完整人物。#12–#15雖為24×32，參照可見只有局部色塊，用途仍依來源證據判定，不猜為新的盟友。

只讀工具入口tools/hd/review_ally_art.py，本機輸出workplace/hd/ally-art-review-v1-20261004.json。逐張獨立PBL解碼核對既有ref PNG、候選尺寸及幾何，保留不同色門檻與單像素負對照；外框相等不自動接受。人工比較預覽沿用redraw的ALLY-review-v1-及20261004前綴，只做原版與候選的整幅最近鄰放大、並列，不修改素材。正式完成仍須造型審查、READY來源契約及正常玩家呈現；現行26筆、已選ALLY #0與正式ENEMY 7/360維持。

【confirmed，限定原版與候選量測】31份既有原版ref逐像素相同，31份候選尺寸／解碼通過，單像素負對照各差1。閾值16／32／60的非黑外框相同分別19／19／18張。#0–#11的舊身體候選全有外框差異，包含已被現行ALLY #0替換的舊art-in/ALLY-00.png；不因此重開目前已接受#0。#12–#30本來就有非黑底色或充滿畫布的像素，整框相同不能證明內部造型正確；本輪不以這19張數字增加完成度。

人工查看原版#30，見藍色髮形／頭部、黃白臉區及右側灰白構件。既有art-in/ALLY-30.png畫成索尼克，不能接受為原版HD。以原版16×16單格重新生成一份藍髮人像候選，完整原生圖、48×48整幅規格化、288×288預覽、提示詞與SHA保存在redraw的ALLY-review-v1-及30-20261004前綴，原版參照256×256不改構圖。新候選恢復人像與主要色區，尚須位置、造型及正常用途來源審查，未替換原候選或正式接入。

來源保全、現行主題不變與進度同步收據沿用workplace/hd/ally-art-review-v1-20261004-source-manifest.json、ally-art-review-v1-20261004-source-snapshot/及同前綴的issue-before／body／after與audit檔。來源副本只保存本輪實際量測、規格化與候選輸入；原版只記SHA，不上傳。

## 100. ALLY肖像來源定位

2026-10-04。前輪全量美術盤點發現#30候選偏離原版，已另存新肖像；正常用途與位置仍未知。本輪只追使用者要求的其他sprite接入所需資料流：ALLY檔名表索引、選圖與繪製呼叫端、正常玩家觸發。未證實前不猜角色身份、不修改原版或正式Theme。

IDA入口沿用技能use-ida-pro-9-4及工具專案/home/anr2/ida_94_official/knowledge-base/ida-94-tools.md。image為ida-pro-9.4-idapython:locked-v1，ID 6f6d59af49d0008c4109a5295b5f374bdc007e2d1ab28cb9de08779584de2780，UID/GID1000、HOME=/home/ubuntu。正式PW_UNP.EXE.i64唯讀，在容器/tmp副本查詢，不改名或寫入正式DB。

匯出工具tools/ida/ally_portrait.py；工作區workplace/ida/hd-ally-portrait-20261004/保存最小probe、query JSON、log與輸入身份。IDA位址使用linear ea及各segment base，執行期地址另列換算，不將IDA ea當作DOS線性位址。原始函式名、運算元與bytes完整保留，語意起始均為unknown；正常來源驗證後才升級。匯出成功以schema、非空內容及輸入SHA核對，不以exit code判斷。

進度核對：probe、query、callers、dispatch及game-draw五份JSON已保存，IDA kernel 9.4、SDK 940、417個函式。輸入PW_UNP.EXE SHA-256為fd5115b91f014c47a1fb1a6e2ecfd645e293264fe614a4b350e92637cad2fbd9；正式PW_UNP.EXE.i64 SHA-256為4db51a199f5913c52dc4964557b9a4aafb965754d663cb0173c9dfdfe9a00c56。輸入身份見input-identity.json。

【confirmed，僅靜態定位】query.json的26項PBL檔名表，索引12指向ally.pbl；表項IDA ea 0x19683、原始指標0x922B、descriptor ea 0x1973B。以上數字全部使用IDA位址空間，不能當作DOS線性位址。sub_189BF保留選檔與讀取指令，sub_18A7C、sub_18B76及其呼叫端已匯出；game-draw.json保留遊戲繪圖包裝常式。原始匯出不改名，靜態角色用途仍標unknown。

【unknown】第30張的正常玩家觸發、用途、選圖來源與座標尚未證實。直接交叉參照不足以涵蓋快取與間接繪圖，不據此宣稱素材未使用。下一步觀察正常遊戲的DOS檔案載入與選圖來源；不擴充ALLY #0以外的READY契約，不改現行26筆或正式ENEMY 7/360。

依使用者要求先同步CONTEXT.md及GitHub #34，本輪不續作來源探針。更新前／後全文、body-file及文字核對收據沿用本工作區issue-progress-before.json、issue-progress-after.json、issue-progress-body.md及issue-progress-audit.json。只更新進度文字，原版、資料庫、state與候選圖片留本機。

接續正常來源觀察：沿用已保存來源的companion-explore-v2-20261003-probe.bin，從既有06-name輸入kai及Enter至07-first-play，不改seed或狀態值。限定觀察原版ALLY檔案讀取、載入常式與打包繪圖暫存器；輸出沿用本工作區native-name-20261004前綴的log、state、frame及RAM。需要最小唯讀觀察器時，入口tools/hd/observe_ally_sources.go，以Go覆映射建研究二進位，不修改正式dosgolem或前端；來源與控制收據沿用同工作區ally-sources-20261004前綴。正常觸發與具體圖片比對成功前不新增正式契約。

觀察器建置補充：同工作區machine-schema.go由原始internal/machine/state.go的完整machineState欄位宣告生成，作為研究Gob解碼型別；不手選欄位或排除差異。overlay JSON、空observe檔、schema及各revision二進位均保留。保存檔包含map，原始Gob位元組順序不保證相同；正式比較全部機器欄位及完整DOS區段，另驗CPU／RAM／port負對照。

獨立核對入口tools/hd/verify_ally_sources.py。以strict_pbl獨立解碼原版31張，核對實際DS:BX來源、正常終點矩形、已知07-first-play原版畫面及IDA／執行期bytes；單像素負對照必須失敗。收據同工作區ally-sources-independent-20261004.json；來源保全沿用ally-sources-source-manifest-20261004.json及source-snapshot/，只保存本輪工具、實際建置依賴與研究輸入，原版只記SHA。

【confirmed，正常原版路線】固定原版06-name／07-first-play的86AFh，不改數值；開局、詢問凱拉、道具及ESP四條鍵序，觀察與無觀察兩側全部44個機器欄位及DOS區段相同，CPU／RAM／port負對照有效。開局終點完整64000色號與既有07-first-play.frame相同。道具選單確實重載ALLY #0，於(128,8)以AL=0、24×32完整貼圖；開局同圖於(264,152)。兩份384-byte打包來源經獨立解碼唯一匹配#0，正常終點各768像素不符0，单像素負對照各差1。詢問凱拉及ESP兩路在本鍵序內無新ALLY載入／完整來源貼圖，不推及其他角色或條件。

【confirmed，正常堆疊及靜態bytes互證】ALLY載入器正常返回0161:2CEE。IDA未為該處建立函式，故原先只查函式與直接xref漏掉此包裝。新span解碼保留原始資料定義：IDA ea 0x131F5對應runtime 0161:2CE5，bytes B40C設定AH=0Ch；0x131F7對應2CE7，8B1EE8AE讀DS:[AEE8h]至BX；0x131FB對應2CEB，E87859近呼叫runtime8666，返回2CEE；0x131FE再讀同一BX並ret。選圖AL保留呼叫前值，兩正常樣本皆0。17-byte查詢範圍從前一包裝的尾端開始，IDA span [0x131F3,0x13204)、runtime [0161:2CE3,0161:2CF4)逐bytes相同；不把前一條pop／ret歸入新包裝。正式DB未改名、未增函式。

停止此窄RE切片的結論：ALLY #0道具肖像是目前未支援的新原版位置，來源已足夠進入顯示生命週期與錨點契約審查。第30張用途仍未知，不能由兩個#0樣本推定。現行26筆、正式敵人7/360及ALLY #0原位置接入維持，不稱新位置已HD。下一步先閉合這條已證實的正常道具肖像路徑，再依實際新選圖事件追其他ALLY。

本批保全腳本ally-sources-preserve-20261004.py、實際依賴ally-sources-build-deps-20261004.json與來源manifest位於本工作區。遠端同步收據新增issue-source-before／prewrite／body／after及source-final-audit前綴，既有issue-progress收據保留。

## 101. ALLY #0道具肖像HD接入

2026-10-04。來源入口§100已證實原版(128,8)、24×32、AL00及完整384-byte來源。本輪先核對正常選單開啟／關閉、既有原版負對照及不可變SCREEN錨點，再審查024 §1.15。正式實作前保留DRAFT，不改原版或猜#30用途。

研究工作區workplace/hd/ally-items-v1-20261004/，原版來源唯讀；anchor-source.json保存錨點與正常frame的只讀量測，items-close前綴沿用既有已保存來源的觀察二進位。實作與驗收入口預留tools/hd/verify_ally_items_runtime.go、tools/hd/verify_ally_items.py；同工作區保存overlay、生成完整machine schema、建置版本、正常來源／圖面／中文／state載回、獨立核對及來源manifest。新本機主題theme-ally-items-v1-20261004/沿用已接受ALLY #0，增加一個已證實位置，其他26筆及PNG不變；未驗證前不選為現行主題。

截至本次進度同步，024 §1.15仍DRAFT，上述實作工具與新主題尚未建立。原版正常Down／Enter選擇Forget it後，選項關閉但兩位置肖像保留；Esc兩次不會關閉此選項，items-close不是正常離開證據。兩條接續與無觀察控制的全部44個機器欄位及DOS區段相同。

anchor-reviewed.json核對27份原版frame：SCREEN.PBL #0右側(248,0,72,40)在23份正對照完全相同，四份標題／防拷負對照各差1906像素，單像素負對照差1。道具頁左側舊錨點差4570像素，右側仍相同。這只證實有限原版狀態的候選錨點，不證明HD接入或全部場景完成。錨點原版色號SHA-256為8d831d84f2b6aa286dfa70cc5aaefccc88522352c5c53843c43c159e05a454c0。

新主題若通過後續契約，計畫保留全部PNG，調整既有背景與角色的明示match，ROOM／OVER沿用原契約，再新增道具位置。因此前段「其他26筆不變」只適用素材bytes，不適用manifest的match；現行26筆主題本身未改。下一閘門是審查024 §1.15，升READY後才實作並驗正常路徑、完整圖面及存讀接續。Issue同步收據沿用本工作區issue-progress-before／prewrite／body／after及progress-sync-audit.json，原版、state與圖片不隨進度上傳。

§1.15證據審查：正常來源已由§100的獨立PBL解碼、384-byte貼圖及兩位置原版終點閉合，27份錨點frame的SHA已於進度同步逐項核對。原版Forget it後保留肖像，來源失效及逐格遮擋沿用§1.2–1.4；每個位置的狀態分開，新增位置限定明示右側錨點。輸入版本、位址基準及工具沿用§100，不擴充ALLY圖號或資料格式。沒有需要猜補的來源語意，足以升READY；實際HD、載回與前端驗收仍待執行，不能提前標CONFORMED。

逐步前端抽測入口tools/hd/ally_items_pwstep_run.py，輸出同工作區pwstep-v1/；沿用tools/hd/export_over_frontend.go匯出每份真正保存產物的原版frame／RGB。道具頁與Forget it後各驗英文／中文、HD／原版及舊26筆回歸。這是pwstep正常checkpoint接續，不是Ebiten實際視窗或正式封包驗收。準備主題來源為同工作區prepare-theme.py，保全腳本preserve.py與source-manifest.json／source-snapshot/保存實際建置依賴；所有原版輸入只記SHA。

逐步前端獨立完整合成核對入口tools/hd/verify_ally_items_pwstep.py，沿用原版PBL及正式字型，不讀Theme內部來源旗標；收據同工作區pwstep-independent-v1.json。

§1.15已升READY並接入。theme.go的來源辨識與圖面共用已證實位置；上方必須明示右側錨點，同圖兩位置各自保存生命週期，同位置重複、未證實圖號、偏移、裁切及錯錨點拒絕。既有角色狀態機不改。真實來源theme測試通過，包含兩位置同時完整與冷載後只認定一位置的負對照。

正常06-name／86AFh輸入kai，再Esc→See Items→Enter、Down→Forget it→Enter及Up，共十鍵20個IRQ1事件。13份三分支取樣的全部44個機器欄位及DOS區段相同，HD兩側中文圖面與快照相同；每份真正保存state冷載後接續100000步，HD與無HD機器亦相同。CPU／RAM／port負對照有效。新增肖像原版貼圖入口54799098，返回54823639；包含中途取樣，未注入位置、數值或seed。

independent-v2.json依原版PBL、入口384-byte來源、正式PNG及GOLEMFNT獨立重建39份完整960×600圖面、中文與兩語合成，全部不符0；新肖像省略負對照23份有效。冷載中途沒有沿用未保存身份，逐格仍依原版；沒有排除任何畫面區域。pwstep-independent-v1.json另核對兩個正常checkpoint各三主題模式及兩語言，共十二份完整畫面不符0，四份新增肖像省略負對照有效。舊26筆及未選HD回歸均通過本批限定範圍。

本機theme-ally-items-v1-20261004/已選入27筆，新增ALLY #0的(128,8)位置及明示右側match。全部26張PNG與舊主題相同，舊主題28檔SHA未變；正式ENEMY美術維持7/360，ALLY正式美術仍僅#0。selection.json及select-theme.py保存限定選入條件。尚未驗Ebiten實際視窗、原版DAT、全敵人動畫、其他ALLY、迷宮、效果及封包；024 §1.15保持READY，尚未CONFORMED。

工具失敗保留：初建漏入既有Oracle研究包裝且把無回傳值Close當作error；加回既有覆映射及正確API後同image建置。首輪capture額外更新中文Frame，造成兩側呼叫數不同；保留首輪二進位、來源與收據，移除擷取的額外Frame後同鍵序完整通過。獨立像素核對外層期限124，但39份終端收據已完整寫出；同容器已終止，後續逐項核對收據內容，不以期限當作產品失敗。高負載單CPU，不作幀率或音訊時序聲明。

本次限定實作遠端同步收據同工作區issue-hd-before／prewrite／body／after；final-implementation-audit.json核對最終文件、來源保全、兩份完整驗收、27筆選入及容器清理。前次DRAFT進度收據不覆寫。

本批來源保全632份精確副本、94份實際非標準依賴，共236640899 bytes，原版28項只記SHA。收尾快照、原版及產物雜湊相符；現行27筆／28檔，全部26張PNG及舊28檔未變，正式前端main.go未改。#34同步2026-10-03T18:54:05Z，寫後正文一致且OPEN，#44未改。完整HD目標持續，下一閘門為新27筆正式視窗與原版DAT抽測。

## 102. 27筆道具肖像正式視窗與原版存讀檔抽測

27筆DAT續驗入口：tools/hd/gui_ally_items_dat_run.py使用原速及0.18秒實際按鍵，從本節run-v5正常道具退出state接續。工作區dat-save-v*、dat-load-v*、dat-independent-v*保存GUI及無HD原版存檔；dat-verified-v*.json與verify-dat-v*.py保存獨立完整畫面、512-byte DAT與52-byte玩家資料結果。dat-source-manifest-v*.json、dat-source-snapshot-v*/與preserve-dat-v*.py保存本輪新增來源，原source-manifest.json與1133份快照不覆寫。產物未經核對即維持未完成。

本節後續工具與收據：工作區內preserve-gui.py保存正式建置的Go／C／內嵌依賴、工具版本、來源快照與原版雜湊；source-manifest.json及source-snapshot/保存本輪限定來源。run-v4／run-v5保留本輪擷取與終端結果，reload-verified-v4.json與reload-check-*保留F11的52-byte資料比較。個別結果以下述實際驗證為準。

2026-10-04。沿用024 §1.15 READY，現行theme-ally-items-v1-20261004。工作區workplace/hd/gui-ally-items-v1-20261004/，正式main.go不加觀察器；tools/hd/gui_ally_items_run.py以實際xdotool操作並保存F10、12份全屏相位、語言中繼資料及原版按鍵紀錄。tools/hd/verify_gui_ally_items.py由原版PBL、正式PNG、GOLEMFNT及F10匯出獨立合成；tools/hd/export_gui_dat.go沿用零步原版frame／RGB／52-byte玩家資料匯出。

從前批正常名字路徑sample01的真正state起始。正式前端冷載-load-state不恢復中文側檔，因此先以實際F10建立相符中繼資料，再用真正F11載回準備起點與正式中文快照。首次run-v1起點仍使用指令時鐘，F11後RunCycles拒絕執行，前端退出1，沒有完整擷取。改由既有pwstep介面設定750 cycles並推進750條指令，產生start-cycle.state；fixture-verified.json證實52-byte玩家資料與原起點相同，不稱完整機器狀態相同。

run-v2以同一正式binary正常Esc→See Items→Enter、切換兩語與HD、Forget it接續。terminal.json保存七組完整擷取，每組12份全屏相位；移動後第八組state已保存，擷取不完整。前端於既定五分鐘期限正常退出0，擷取工具其後因視窗已關閉而逾時，屬測試生命週期問題。後續F11還原與DAT未執行。run-v3延長有界期限至十分鐘，自run-v2真正保存的道具state接續，僅完成初始道具樣本；依使用者要求先同步進度，本輪不增加驗收通過數。

尚未產生獨立GUI通過收據、新DAT存讀或封包結果。tools/hd/verify_gui_ally_items.py已補入正式字型與text/help.json的速度標記期望，但尚未完成驗證；不能稱全屏吻合。起點、前端／匯出binary、build-deps、execution.json、commands.json、各終端收據及原始相位均留本工作區。正式main.go本輪未改，不改原版值或seed，不稱從開機或GUI自然時序固定seed對拍。後續獨立verified.json、原版DAT及來源保全／稽核亦由本節索引；檔案未產生即維持未完成。

本輪進度同步：GitHub #34寫前全文與更新時間一致，寫後正文與issue-progress-body.md完全相同，2026-10-03T19:25:17Z保持OPEN。issue-progress-before.json、issue-progress-prewrite.json、issue-progress-body.md、issue-progress-after.json與issue-44-progress-after.json均留本工作區。#44仍OPEN且更新時間未變。gui-run-v2-source.py精確對應run-v1／v2執行雜湊，gui-run-v3-source.py保存本次延長期限版本；完整GUI建置來源保全尚待後續驗收收尾，不冒充已完成。

收尾：run-v3正式前端及外層命令退出0，一份初始道具擷取與零步原版匯出已保存，沒有執行獨立全屏驗收。pw-hd容器清單為空，本輪一次性容器均以--rm清除。

2026-10-04後續：修正獨立GUI工具的速度欄位speed_gear及缺字前進規則後，run-v2七份完整擷取與run-v3初始樣本均完整960×600不符0，含速度標記，未排除區域。但run-v2第八份不完整、run-v3缺英文HD，兩批驗證器均依既有閘門退出1，收據保留。正式中文字面另依text/*.json核對。

run-v4取得十組擷取及F11前後資料。reload-verified-v4.json證實52 bytes移動前後不同，真正F11還原後與移動前相同，單byte負對照有效；範圍限玩家資料。原訂Options擷取實際已選Cancel返回迷宮，文件依實際畫面記錄，不以檔名當成功證據。外層390秒逾時中止runner，沒有正常terminal.json及record.json，不列整批GUI通過。

期限真因confirmed：正式main.go的quickLoad以同時指派重設g.start，Update以now.Sub(g.start)判定quit-after。因此最後一次F11重新起算六分鐘，外層從啟動算390秒不足。run-v5沿用相同正式binary、原始起點與四模式／Forget it／移動／F11鍵序，quit-after三分鐘，外層430秒，涵蓋最後F11後的重新起算；正式main.go不改。先等待正常terminal與原版匯出，再獨立全屏驗收。

第五版結果：27筆主題的正式Ebiten道具路徑第五版有限通過：七份完整960×600視窗獨立不符0，涵蓋兩語、HD開關、Forget it、移動與真正F11。前端與驗證器退出0，未排除任何區域；52-byte玩家資料移動前後不同，F11後與移動前相同，負對照有效。新DAT、正式封包、全部sprite與完整動畫仍未完成，研究038 §102。
證據run-v5/verified.json、terminal.json、original-frames.json及reload-verified-v5.json。七組各保存十二相位，至少一份完整畫面吻合才通過；沒有排除速度標記、文字或盟友。中文字面依正式text/*.json核對，新肖像省略及單像素負對照有效。前端熱鍵F4／F5／F10／F11未出現在原版按鍵紀錄。run-v4兩份options命名樣本實際為Cancel後迷宮，非Options完成證據；原版DAT未產生。finish工具等待上限改650秒以涵蓋預設十分鐘，外層仍須容納最後F11前全部操作。本輪已驗版本精確副本保留，正式程式與26張PNG未改。

第五版進度同步：#34寫前全文與時間一致、寫後正文與issue-v5-body.md相同，2026-10-03T19:48:47Z保持OPEN。issue-v5-before.json、issue-v5-prewrite.json、issue-v5-body.md、issue-v5-after.json、issue-44-v5-after.json及launch-recipe.json由本節索引；#44未改。reload-verified-v5.json亦由本節索引，五份省略新肖像負對照各4340個RGBA像素。

來源保全：source-manifest.json共1133份精確副本、346份實際非標準Go／C／內嵌依賴、109261130 bytes，原版9項只記SHA。source-snapshot/逐份重驗雜湊及UID/GID1000通過；來源保全於最終收尾附記前，收尾文件另記SHA。final-audit.json保存本輪快照、原版、GUI、F11、擁有權與清理核對。

2026-10-04 DAT續驗結果：現行27筆主題的原版DAT正常選單有限通過：HD27.DAT 512 bytes與同一GUI存檔前起點、無HD原版正常鍵序產物相同；續接保存三份及LOAD GAME六份完整960×600視窗不符0，四模式載回52-byte玩家資料相同。載回後正常移動另驗兩份完整視窗，玩家資料與同起點無HD原版相同。保存由真正F11接續首批檔名頁；不稱一次未中斷或從開機，新敵人戰鬥DAT與封包未驗，研究038 §102。
首批dat-save-v1六份完整視窗差0，g-after-save-hd-chinese不完整，前端退出0但擷取工具退出1；保持整批未通過。dat-save-v2從首批真正保存的空白檔名頁經F11接續，三份完整視窗差0；dat-load-v1正常SELECT→LOAD GAME→HD27六份完整視窗差0，保存與載回前端及核對工具均退出0。DAT SHA-256 3468b6ec00439ceaad19bb121ec2f57214f80516d12ba3c289c58fe076d96df1，兩次GUI保存與獨立原版逐位元組相同；26／12／14個原版事件符合完整正常鍵序，前端熱鍵未外洩。byte負對照及五十二位元組玩家資料變更負對照有效。

載回後左上Logo rectangle(0,0,152,84)與未進道具頁的正常原版sample01共12768個色號相同；不是以檔名loaded推定成功。dat-load-move-v1由本次真正LOAD GAME的state經F11接續，正常Up後玩家資料與同起點無HD原版相同，兩份完整視窗差0。dat-move-verified-v1.json、dat-move-independent-v1/、dat-load-prepared-v1/selection.json及verify-dat-v1-failed-source.py亦由本節索引。SELECT未重印原文保留，不稱全程中文；原版正常Logo不強制改成道具狀態面板。初版aggregate的Esc名稱寫成Escape，未改鍵序，只修驗證器後同資料通過；已通過的九份全屏收據不重跑。素材與正式main.go未改，27筆與正式敵人美術7/360不變。

DAT同步收據：issue-dat-before.json、issue-dat-prewrite.json、issue-dat-body.md、issue-dat-after.json及issue-dat-44-after.json。#34寫前全文及時間一致、寫後正文完全相同，2026-10-03T20:28:49Z保持OPEN；#44未改。dat-move-independent-v1/與dat-move-verified-v1.json由本節索引，後續來源保全採preserve-dat-v1.py，final-dat-audit-v1.json保存最終快照及擁有權核對。

DAT收尾：新增425份精確副本、60725801 bytes；既有1133份來源快照全部重驗未變，沿用346份正式建置依賴。兩份manifest的原版雜湊均重驗相同。九份正常DAT視窗與兩份載回移動通過；正式main.go與27筆／26張PNG未改。快照保存於本段收尾前，文件最終SHA另記final-dat-audit-v1.json。一次性pw-hd容器均--rm清除，主機最後核對清單。

2026-10-04進度再同步：CONTEXT與#34的現行待辦已修正，不重開已有限通過的道具GUI、F11及正常迷宮DAT。正式敵人美術7/360、27筆及其他未完成範圍不變。#34寫前全文及時間一致，寫後正文相同，2026-10-03T20:45:26Z保持OPEN；#44未改。本工作區新增`issue-progress-refresh-before.json`、`issue-progress-refresh-prewrite.json`、`issue-progress-refresh-body.md`及`issue-progress-refresh-after.json`保存本次同步，舊收據維持原檔。

## 103. 歐格斯末姿勢兩份候選與越界核對

2026-10-04。沿用024 §1.7 READY及文件職責入口。使用內建image_gen局部重畫第8張突出部，新增v6／v7；原生1086×1448、完整提示詞及72×96候選均保存於既有workplace/hd/redraw/，只做ImageMagick整幅Lanczos規格化，沒有Python修圖、角色平移、裁切或縮小避位。兩份均未接受，歐格斯候選累計25份，正式7/360及27筆不變。

【confirmed，限定量測】原版ENEMY01.PBL SHA-256 `22f664050ce7ffa4ea0f6941c9c91cd1ab43671ea5b53491f6799f78ba8e64af`，第8張檔案偏移3269、24×32。獨立解碼與432×576參照248832個RGB像素完全相同，單像素負對照差1。工具沿用psychicwar-go-ebiten:latest中的Python3.11.2與ImageMagick6.9.11-60；來源／輸出SHA保存在下列provenance及量測收據，不涉及遊戲位址或狀態改寫。

- v6突出部[4,61,11,79]，原版三倍[3,60,11,80]。左、上偏差縮到一像素，下緣仍少一列；眼睛[60,16,67,20]與原版[60,15,68,20]不同。未接受。
- v7窄區量測為[3,60,11,81]，該區從y60開始，不能證明y60以上沒有越界。核對包含上下空白的[0,58,12,26]後，實際[3,58,11,81]，頂緣過高、下緣多一列，未接受。
- 原版上下空白區避開手臂及腿部，獨立解碼全黑；v7新增7個亮像素，單點資料負對照有效。較寬的[0,54,12,33]包含原版手臂及腿部，整框相同不能證明突出部，沒有採用為通過依據。
- 生成區外RGB亦有變動，v6／v7相對v4分別3693／3977像素，不稱其他部分逐像素固定。兩張主構圖仍由人工審查；外框量測不證明完整輪廓或正常呈現。

本節索引：`ENEMY01-group2-v6-`與`ENEMY01-group2-v7-`的`generated-08-tip-20261004.png`、`frame-08-tip-20261004.png`、`provenance-08-tip-20261004.json`、`measure-08-20261004.py`、`measurements-08-20261004.json`；越界方法及完整收據為`ENEMY01-group2-v6-v7-guard-08-20261004.py`／`.json`。提示詞與每項輸入／輸出SHA皆在provenance。停止本輪局部反覆生成，下一批依單張原版重新審查姿勢或接續其他已有READY來源的sprite；不放寬原版位置要求，不選入兩張候選。完整中文化與HD尚未完成。

收尾入口同工作區`ENEMY01-group2-v6-v7-audit-20261004.json`及`ENEMY01-group2-v6-v7-issue-before-20261004.json`、`issue-prewrite-20261004.json`、`issue-body-20261004.md`、`issue-after-20261004.json`；後三項亦帶同一`ENEMY01-group2-v6-v7-`前綴。11項候選輸入／輸出SHA及現行主題27項輸出SHA核對相同，正式main.go未改。Docker image ID `sha256:083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7`。#34寫前全文與時間一致，寫後正文相同，2026-10-03T20:57:24Z保持OPEN，#44未改。工作樹無root-owned項目或.md目錄，兩個儲存庫git diff --check通過，pw-hd容器清單為空。未commit／push／發行，原版、state與候選留本機。

## 104. 葛雷戈林末姿勢全新單格候選

2026-10-04。沿用024 §1.14 READY及imagegen技能，改以第5張唯一原版單格為構圖依據；已接受第3張僅作畫風參照，未再編輯舊短鉤端點。內建image_gen新增v6原生1086×1448，ImageMagick整幅Lanczos規格化72×96，沒有裁切、平移角色或Python修圖。原生圖、完整提示詞及來源／輸出SHA保存在既有workplace/hd/redraw/。

【confirmed，限定參照及色區】原版ENEMY03.PBL SHA-256 `ad6e8183bbf6c6fc258693b1f0ac726593feff5d052b5da18ac715cc2e63e5c1`，第5張檔案偏移2040、24×32、解碼色號SHA `e5be9d08953469dfeed83ca1aba9555f92b716e8a876b4a742d531a7ca6822e9`。原版432×576參照248832個像素不符0，單像素負對照差1。這只證明參照正確，不代表新候選通過。

- 下方豎條量測區的白色色群由v5的3降到0，青色50，原版青色63。候選在青色區增加高光仍須造型審查；不稱原版RGB相同。
- 左圓塊色區[4,33,33,62]，原版[3,36,32,59]；右圓塊[38,37,68,65]，原版[45,39,68,68]；頭部黃色[39,22,48,24]，原版[39,24,50,26]。手動審查另見左眼增加黑色瞳線，原版未有該線，未接受。
- 原版短鉤底端[51,75,59,77]有27個色號4／5／12像素，新候選在這27個位置的紅色色群為0；單點資料負對照檢出1。舊寬色群量測在[39,85,41,86]抓到主幹，不能把它當成短鉤端點。候選短鉤明顯偏高，未選入。
- 初版遮罩只列色號4／12，漏掉原版5而在18像素前置失敗；核對原始色號後修正，同image及候選重跑通過。前置inline命令括號語法錯誤未產生量測，改為保存腳本完成。兩者是驗證工具問題，未改圖片或驗收門檻，分類保存於provenance。

工具版本沿用§103的Python3.11.2、ImageMagick6.9.11-60及同Docker image。入口為`ENEMY03-group1-v6-generated-05-20261004.png`、`frame-05-20261004.png`、`provenance-05-20261004.json`、`measure-05-20261004.py`、`measurements-05-20261004.json`與`mask-review-05-20261004.py`，皆帶同一`ENEMY03-group1-v6-`前綴。只讀量測沿用`review_zellwal_v5.py`，精確依賴SHA已保存。原版位置(32,152)、全域8×8與完整動作目標不變。

本候選未接受，正式敵人美術7/360、27筆主題及已通過GUI／DAT範圍維持。這批不追加局部端點生成；後續審查依每一原版色區遮罩區分短鉤與主幹，不從寬色群名稱推定部位。全部sprite、完整動作與HD交付尚未完成；未commit／push／發行，原版、state及圖片留本機。

收尾入口同工作區的`ENEMY03-group1-v6-audit-source-20261004.py`、`ENEMY03-group1-v6-audit-20261004.json`；依賴精確副本`ENEMY03-group1-v6-dependency-review-20261004.py`及`ENEMY03-group1-v6-dependency-pbl-20261004.py`保存原始來源SHA。同步收據`ENEMY03-group1-v6-issue-before-20261004.json`、`ENEMY03-group1-v6-issue-prewrite-20261004.json`、`ENEMY03-group1-v6-issue-body-20261004.md`、`ENEMY03-group1-v6-issue-after-20261004.json`均由本節索引。十項候選輸入／輸出及現行主題27項輸出SHA相同，正式main.go未變；#34寫前全文與時間一致，寫後正文相同，2026-10-03T21:09:59Z保持OPEN，#44未改。首批收尾漏掛/orig，沒有產生稽核收據，補回唯讀掛載後同image重跑完成；未改產品。worklist render／verify及兩個儲存庫git diff --check通過，工作樹無root-owned項目或.md目錄，pw-hd容器清單為空，全部一次性容器已清除。

## 105. 現行27筆的Frame成本與每秒幀數

接續入口：本節既有工作區`rerun-v2/`另存三模式重跑，沿用同一psychicwar-measured.bin；run_performance.py新增--binary只分開輸出與二進位路徑，不改按鍵、區間或模式。verify_performance.py的計數負對照改破壞區間第二筆，讓前後單調性能確實檢出歸零；破壞第一筆不能保證違反單調性，未採用為驗收證據。原批來源與失敗保留。

來源保全入口`tools/hd/preserve_performance.py`，對實際build-deps.json的Go／C／內嵌檔及模組、文字、字型與現行主題逐項核對。相同來源引用§102精確快照，新工具與覆映射main另存於rerun-v2/source-snapshot/；source-manifest.json記錄二進位版本與SHA及每份來源。rerun-v2/verified.json為本輪三模式結果入口，產物未通過前不增加驗收數。

2026-10-04。024 §6第6項要求開主題及不開主題的Frame耗時與每秒幀數，不設門檻。工具入口`tools/hd/prepare_performance.py`與`tools/hd/run_performance.py`；本機工作區`workplace/hd/performance-v1-20261004/`。以正式main.go精確副本、Go覆映射加入計時與計數，不修改正式前端或素材。比未載主題、載入後關閉HD、開啟HD；起點是§102正常道具Forget it狀態，真正F11還原文字側檔後量測10–30秒。尚未產生結果即維持此項未完成，不宣稱封包、戰鬥、實機GPU或全部sprite效能。

本批未完成。未載主題開跑負載6.99，前端正常退出0，保存完整統計、按鍵、F10及視窗收據；第二模式開跑前負載10.64，工具依既有load < 7閘門停止，退出1，第三模式未執行。第二模式僅建立空白saves目錄，沒有啟動前端或執行收據。未產生verified.json，不報完整Frame／幀率比較，失敗分類為共用主機量測環境。

驗證工具入口另含`tools/hd/verify_performance.py`，本批因前置負載閘門停止而未執行。本工作區`batch-status.json`保存退出與停止原因，`source-progress-manifest.json`及`source-*.py`保存本次工具精確副本與SHA；Go覆映射、main-base.go、main-measured.go、main.diff及build-deps.json留本機。正式main.go SHA仍為`794565013a934a69e5cf7a98bde65104253b166d6fb58bb87b4688fb4dffcce0`。遠端同步收據使用`issue-progress-before.json`、`issue-progress-prewrite.json`、`issue-progress-body.md`、`issue-progress-after.json`；原版、state及圖片不上傳，#34保持OPEN。

進度同步收尾：#34寫前全文及時間一致，寫後正文完全相同，2026-10-03T21:29:22Z保持OPEN，#44未變。工作清單已改成現況摘要，render／verify及兩個儲存庫git diff --check通過；工作樹無root-owned項目或.md目錄，pw-hd容器清單為空。全部一次性容器已清除。前置收據核對誤將空白saves目錄當成完全空目錄，修正檔案核對後完成，未改產品或量測門檻。最終同步稽核入口為同工作區`progress-audit.json`，文件SHA記錄於本句追加前。

2026-10-04接續結果：rerun-v2三模式前端及驗證器均退出0，每模式固定10–30秒取19筆，實際比較跨度18.196／18.232／18.414秒；全部計數負對照檢出歸零。開跑負載均小於7，真正F11後中文、1倍速、非戰鬥及相同區域／座標條件成立。量測限正常Forget it後靜態畫面與游標，不是戰鬥、全部動作或正式封包。

| 模式 | 中文與主題Frame合計平均ms | 主題Frame平均ms | Draw CPU平均ms | Ebiten FPS中位數 |
|---|---:|---:|---:|---:|
| 未載主題 | 0.9804 | 0.0002 | 1.8620 | 46.0671 |
| 載入後關閉HD | 3.2060 | 1.6836 | 2.6898 | 22.9631 |
| 開啟HD | 3.1777 | 1.6674 | 5.1826 | 17.9337 |

【confirmed，限定本機量測】Frame計數區間涵蓋tr.Frame與theme.Frame，另記theme.Frame；Draw CPU不含完整GPU或螢幕呈現時間。2 CPU、軟體OpenGL、null音訊及計時開銷均有限制，主機負載及三模式分開執行也可能影響比較，不據此宣稱硬體因果比例或60 FPS。主題關閉仍執行來源觀察，因此關閉並不等於未載主題。此項現行27筆數據記錄有限完成，不擴張為完整HD完成或效能合格門檻。

來源保全source-manifest.json記錄355項實際非標準依賴，448份來源中442份重用§102精確快照、6份新來源另存，共同Go覆映射明示實際計時main。正式main.go及現行27項輸出SHA未改；原版9項只記SHA。rerun-v2/verified.json、三份terminal.json／execution.json／stats.jsonl、source-manifest.json及source-snapshot/為完整收據。後續同步收據為同目錄issue-before／prewrite／body／after，最終稽核為final-audit.json；全部原版、state及圖留本機。

本批收尾：三模式前端與驗證器退出0，計數負對照有效；355項實際依賴、448份來源、原版9項及所有驗證輸入SHA相符，正式main.go與現行27項主題輸出未改。#34寫前全文及時間一致、寫後正文相同，2026-10-03T21:36:49Z保持OPEN；#44未變。worklist render／verify與兩個儲存庫git diff --check通過，pw-hd容器清單為空；一次性容器均已清除，未commit／push／發行。稽核腳本final-audit.py與final-audit.json留本節rerun-v2工作區，文件SHA取本次追加後版本。

## 106. 首組第2張原版單格重畫

2026-10-04。沿用024 §1.6 READY與imagegen技能；從ENEMY00 #2原版單格重新生成，第0張已接受素材只供畫風，不編輯v11臂帶端點。工作區沿用workplace/hd/redraw/，新檔前綴ENEMY00-group0-v12-，圖號02及日期20261004。本節索引generated／frame／provenance／measure／measurements等候選與只讀量測；生成與接受分開，未審查前不改正式27筆或7/360。來源參照ENEMY00-group0-v5-source-02-20261003.png，唯一構圖依據為原版24×32完整圖，位置(32,152)與全域8×8不改。只做整幅ImageMagick規格化，沒有Python修圖或局部平移。

本輪新增v12及v13。v12仍移位，整體[0,0,59,94]、右靴下緣88，原版整體[0,0,59,95]、右靴下緣92，未接受。v13移除第0張畫風參照，只以原版第2張生成；整體、左靴與右靴非黑閾值32外框分別[0,0,59,95]、[3,78,35,95]、[36,78,59,92]，均與原版相同。左臂帶y54–56白色色群17／18／7，原版各18，舊v11皆0。這些是限定色區與外框量測，不代表輪廓或RGB逐像素相同。

v13造型初審通過：大頭與雙天線、黃下臉與黑色缺口、白灰青紅護甲、抬臂環繞腰部、左側灰色突出部及非對稱靴子保留。金屬高光畫在原有色塊內，沒有另增封閉裝甲板；保持原版部位範圍，平順外緣。初審不等於正式選入，完整動作一致性及正常HD呈現仍待驗。

【confirmed，限定原版參照】ENEMY00.PBL SHA`8241b0ea73b1e7402b13adc434f10a5b1288f88f01fcfc3ba5710e4923fe6067`，第2張檔案偏移768、24×32。獨立解碼與432×576參照248832像素不符0，單像素負對照差1。工具沿用同Docker image、Python3.11.2與ImageMagick6.9.11-60，生成原生1086×1448及72×96整幅規格化，完整提示詞在兩份generation-02-20261004.json。

正常來源下一閘門：既有Kasuruji保存frame限定抽查沒有完整第2張匹配，最少仍差72像素；原版額外成分的原因未在本批判定。不因檔名或差分呼叫就宣稱新第2張完整HD可見。停止本輪生成，下一步沿既有§96來源生命週期原型核對正常動作，完成新素材可見及載回後接續，再決定正式選入。正式7/360、現行27筆及所有既有GUI／DAT範圍不變。

本節追加索引：同工作區ENEMY00-group0-v13-generated／frame／generation／measure／measurements-02-20261004檔；ENEMY00-group0-v12-v13-normal-source-02-20261004.py／.json保存原版frame抽查，兩份provenance-02-20261004.json保存SHA與初審狀態。審查工具只讀RGB，原版未寫入。同步與收尾檔以ENEMY00-group0-v12-v13-issue及audit-20261004為前綴，原版、state及圖留本機，未commit／push／發行。

本輪收尾：兩候選的原生圖、72×96、完整提示詞與全部來源SHA相同，88份原版frame及兩份精確解碼器核對完成；現行27項主題輸出及正式main.go未改。#34寫前全文與時間一致、寫後正文相同，2026-10-03T21:52:26Z保持OPEN，#44未變。worklist render／verify與兩個儲存庫git diff --check通過，工作樹無root-owned項目或.md目錄，pw-hd容器清單為空；全部一次性容器已清除。GitHub首次讀取HTTP 504已終止，重新讀取完成後才寫入，未重送不確定的寫入。稽核腳本與結果為同工作區ENEMY00-group0-v12-v13-audit-20261004.py／.json，文件SHA取本段追加後版本。未commit／push／發行。

## 107. 首組正常動作與重疊來源生命週期

2026-10-04。第2張新候選初審通過後，先查正常來源；不改024 §1.6的正式辨識条件。入口tools/hd/observe_body_context.go，本機工作區workplace/hd/body-context-v1-20261004/。從既有正常next-body-normal-kasuruji-v3-20261001-event00.state接續，原版完整第0張為起點；原版正常等待15000000條指令，沒有新按鍵、改值或seed重擲。記錄所有與(32,152,24,32)相交的8705／8751貼圖來源、前後原版色號及終點state；另跑無觀察控制到相同步數，再用既有44欄位與DOS區段比較器核對。

先前只收24×32完整角色矩形，事件之間有245／175／57像素的區內變化，不能用那份事件清單重建全部寫入。已核對CS:32CA來源也匹配已證實第1／2張差分，不從來源位址把它猜成效果。晚期第2張差72像素的殘差位於原版局部[8,0,23,21]，只有色號1／3／9／11；來源含義仍未知。下一閘門是完整相交貼圖記錄及獨立來源模型，實際收據未通過前不宣稱可正式接入。

獨立驗證入口tools/hd/verify_body_context.py，設計為從不可變原版第0張及完整相交來源依AL=0／1逐像素重建；before／after只作oracle比較，不反填model。原型擬以已證實相鄰差分更新來源身份，未知完整覆蓋則清除；8×8只有來源原圖完全吻合且非全黑才列可見。驗證器尚未執行，independent.json與來源保全收據尚未產生，不宣稱來源模型已通過。

【confirmed，限定觀察不干擾原版】觀察工具與原版控制均正常退出0，記錄98次相交貼圖，起點237710721、終點252725753條指令。trace.json的原版畫面、暫存器、cycles與RAM對照相同；machine.json另核對全部44個機器欄位及DOS區段相同，CPU／port／RAM負對照有效。此結果只驗證觀察未干擾原版，不代表新第2張HD已正常可見或可正式接入。正式辨識條件、27筆主題與7/360不變。

進度同步收據沿用本工作區issue34-progress-before.json、issue34-progress-prewrite.json、issue34-progress-body.md及issue34-progress-after.json。原版、state、DAT與圖片不上傳。

### 107.1 相交貼圖外的缺口與第二版驗證

第一版獨立來源模型在第3次一般貼圖前失敗：113像素皆為XOR 0Ah。
第一版只收8705／8751，未收已證實的8260／4E34位元遮罩；未觀察不代表原版沒有寫入。
失敗工具及結果保留在body-context-v1-20261004/verifier-source-failed.py及independent-failed.json。
不从before畫面反填模型，不放寬逐像素條件。

第二版工作區workplace/hd/body-context-v2-20261004/，觀察入口仍為tools/hd/observe_body_context.go。
以相同起點、seed及無新輸入接續到相同252725753步；98次一般貼圖與19次16×16遮罩共117次。
遮罩DS:DX=0161:4E36、32 bytes與§37固定原版bytes完全相同，SHA
`e1aa2b9ddb6488a70d573ca3a03e71028cd8a2dafa68087c6f6f3d7c60950a5a`。
由完整原版第0張、所有原始packed來源及固定遮罩逐像素重建，每次before／after與終點均不符0。

【confirmed，限定正常保存起點無輸入接續】independent-v2.json驗證16次有向身體轉換，
為0→1→2→1→0四輪，全部原版完整前姿勢皆失配；仍可從先前唯一已認定來源追溯相鄰方向。
第2張有24份after樣本至少一個非空8×8格完整匹配不可變原圖。
這是可見格，不是24份完整HD畫面。錯來源像素、錯XOR模式、缺一般貼圖與缺遮罩四項負對照有效。
觀察與控制工具均退出0，原版全部44個機器欄位及DOS區段相同，三項機器負對照有效。

已建立024 §1.6.1 DRAFT，以此窄證據準備首組受遮擋前姿勢辨識；正式程式仍維持完整前姿勢雙條件。
唯一身份、未知覆蓋、HD開關、中途及載回清除須審查後才轉READY。
新第2張正常HD可見、載回接續、完整動畫、美術正式選入及GUI／DAT未由本原型驗證。
27筆主題、7/360與204的範圍不變，原版來源、位址及bytes未修改。

精確保全入口tools/hd/preserve_body_context.py及本工作區source-manifest.json：
80份實際非標準建置依賴，108份來源／收據，83份重用已核對快照、25份新副本，15727691 bytes。
保存觀察二進位、兩版驗證器、完整state比較器及第一版失敗來源；原版兩檔及正常起點只記SHA。
工具版本Go1.24.13、Python3.11.2，沿用psychicwar-go-ebiten映像，無新工具鏈。
最後輸入／文件／清理核對與同步由本工作區final-audit.json及issue34-source-model-*檔索引。

## 108. 首組受遮擋來源的正式接入與第2張

2026-10-04。沿024 §1.6.1及§107的原版來源模型，先審查唯一身份與清除邊界，再轉READY及實作。
工作區沿用workplace/hd/，本輪輸出body-runtime-v1-20261004/，第2張技術候選主題theme-kasuruji-pose2-v1-20261004/。
審查入口tools/hd/review_body_contract.py，正式接入位於apps/psychicwar/theme/ally.go及enemy.go，
驗證入口tools/hd/verify_body_runtime.go與verify_body_runtime.py。
未取得正常來源、獨立8×8輸出及載回接續收據前，不宣稱新素材接受或8/360。
原版檔案、座標、比例、全域8×8格與顯示先後不改，其他sprite的完整目標維持。

### 108.1 READY審查與正式實作

contract-review.json以13份已登記敵人原版來源核對同位置唯一性；13個契約案例涵蓋四條有向邊、
錯前身份、跨組／跨檔、歧義、未知來源／模式、覆蓋及載回清除，全部通過。024 §1.6.1依此轉READY，
原版來源雜湊與完整117次模型沿§107，未新增未證實轉換。
實作僅ENEMY00 #0–#2可使用持續身份；入口先保存同位置唯一有效身份，
同時核對當次錨點、沒有貼圖進行中，以及原版完整前圖是否與持續身份矛盾。
已證實差分仍須384 bytes完全吻合；其他ENEMY組維持完整前姿勢條件。
未知覆蓋清除、成功載回清除、入口／返回閘門、全黑格透明與原點8×8比對不變。
接入前ally.go／enemy.go／theme.go精確版本保存在本工作區*.go.before及production-before.json。

正式測試body_context_test.go新增14個實際原版生命週期案例，包括正反方向、錯來源、
未知完整覆蓋後差分、未知模式、冷載、歧義、跨組、錨點、貼圖進行中、完整原圖矛盾、
來源缺失、HD關閉及未擴充舊組；被改畫格不顯示HD、其他吻合格仍可見。
連同既有回歸共52個pass事件、1項缺外部pose-state環境的既有測試skip，沒有fail。
首批只缺/hd/bg.idx的唯讀掛載，環境失敗保存於tests.jsonl；補上既有workplace/hd掛載，
同程式與同原版乾淨重跑退出0，完整收據tests-rerun.jsonl。

### 108.2 正常接續與真正載回的完整圖面

【confirmed，限定正常保存起點接續】verify_body_runtime.go由與§107相同的完整第0張起點、
相同seed與無新輸入接續，14份正常／中途／返回／終點樣本與無HD原版控制相同。
每份獨立核對全部44個機器欄位及完整DOS區段，CPU／port／RAM負對照有效。
另以真正LoadStateFile及ResetForLoad載回完整起點，接續至第2張；
真正載回第2張貼圖中途並接續100000步各另取樣，共17份。
HD停用不畫圖，重啟圖面相同；原版EXE、RAM、色號、seed及存檔未改。

verify_body_runtime.py只讀原版PBL、PNG與§107的獨立來源證據，不使用正式active旗標。
17份完整960×600圖面逐像素不符0；第2張entry／mid／return及完整起點載回接續共4份HD可見，
省略第2張均有差異。完整起點載回後的第2張原版frame與圖面和暖接續完全相同。
冷載受遮擋中途清除身份，當下及100000步後按契約回退，沒有猜補或持續舊圖。
此為正式Theme輸出，不是Ebiten實際視窗、中文、DAT、全部動畫、實機或封包验收。
runtime.json及independent.json保存兩側狀態与獨立結果；Go1.24.13、Python3.11.2，
沿用既有Docker image與UID/GID1000，所有原版唯讀掛載。

### 108.3 第2張限定接受與現行主題

ENEMY00 #2 v13造型初審依§106，原版整體及兩靴外框、白臂帶高度已核對。
本輪檢視72×96完整候選、原版完整參照及正式圖面合成：大頭、雙天線、黃下臉與黑色缺口、
白灰青紅色區、抬臂環腰、左灰色突出部及非對稱靴子保持；高光在原色塊內。
角色在(32,152)、24×32，位置、比例及原點8×8格不變，框線先後維持原版。
造型與限定正常HD可見、真正完整起點載回接續通過，第2張接受並正式選入本機新主題。
正式敵人美術8/360：ENEMY00 #0／#2／#3／#6–#8、ENEMY01 #6、ENEMY03 #3；其他352張未完成。
第1張、整組動畫、其他重疊、冷載受遮擋姿勢完整HD、新主題GUI／DAT及交付未完成。

現行本機主題theme-kasuruji-pose2-v1-20261004/，27筆／26張PNG，只替換ENEMY00-02.png，
來源為redraw/ENEMY00-group0-v13-frame-02-20261004.png，其他素材bytes與舊27筆相同。
舊theme-ally-items-v1-20261004及其GUI／DAT／效能收據維持各自範圍，不延伸為新主題通過。
接受收據art-review-accepted.json與新主題selection.json；檢視圖由tools/hd/preview_body.go
直接將原版RGB與正式圖面合成，沒有生成或編輯美術。

精確保全入口tools/hd/preserve_body_runtime.py、source-manifest.json與source-supplement.json。
90份實際非標準建置依賴、286份來源／收據，131份重用精確快照、155份新副本，60719612 bytes。
原版及正常起點只記SHA；所有資產及狀態留本機。新目錄複製來的selection.json已有舊紀錄，
首次exclusive寫入因此停止；接受審查已保存。保留selection-before.json後，只修新目錄紀錄，
selection-current.json及supplement保存新SHA，舊主題未改。
最後來源、文件、同步及清理索引為本工作區final-audit.json與issue34-body-runtime-*檔。

## 109. 首組第1張原版單一參照重畫

2026-10-04。沿024 §1.6.1 READY及§97的第1張造型證據，處理v13左靴新增白塊與右靴縮短。
只用`redraw/ENEMY00-group0-v5-source-01-20261003.png`作構圖參照，不加入其他姿勢。
原版ENEMY00.PBL #1偏移420，24×32；原圖SHA及432×576參照核對沿§97。
候選、完整提示與生成紀錄保存於`workplace/hd/redraw/ENEMY00-group0-v14-*-01-20261004.*`；
只讀量測入口與結果為`workplace/hd/art-kasuruji-pose1-v1-20261003/measure-v14.py`及`measurements-v14.json`。
使用內建imagegen，整幅正規化沿既有ImageMagick流程；不以Python修改圖像。
尺寸及部位外框只定位差異，接受仍需完整造型審查及正常HD呈現。正式8/360與現行主題暫不變。

v14右靴回到y92，整體與左臂範圍吻合；左靴x4較原版x3內縮1像素，右臂x52較x53內縮1像素。
完整檢視另見右前臂白帶偏長；不因整體外框相同就接受。原版參照248832像素不符0，單像素負對照有效。
一次局部修正v15以v14為編輯目標、同原版為幾何參照；輸出及提示沿同目錄`ENEMY00-group0-v15-*-01-20261004.*`。
只讀量測沿`measure-v15.py`及`measurements-v15.json`，保留v14與原版，不覆寫現行主題。

v15局部修正仍未達指定範圍：左靴x4、右臂x52未變，右臂白色閾值下緣由y62變y61，原版為y59。
【confirmed，限定量測】兩份原版參照核對不符0及單像素負對照差1；右靴均回到原版y92。
【人工審查】v14／v15改善細密線條及右靴高度，但完整造型仍見右前臂白帶偏長，未接受或選入。
兩次同類修正後重新載入READY入口及024 §1／§6，保持原版範圍要求，不放寬閾值追認。
本批停止此局部修正方法；正式8/360、現行主題與程式均不變，沒有新增正常呈現驗收。
本批精確提示、生成原圖、正規化圖及工具SHA由各`generation-01`、`provenance-01`索引，
量測只讀圖像；同步及最後稽核沿本工作區`issue-v15-*`及`audit-v15-20261004.json`。

## 110. 第4張單一原版姿勢重畫

2026-10-04。沿024 §1.3／§1.4 READY及研究§87第4張拒絕證據，改以唯一原版第4張參照。
原版ENEMY00.PBL #4偏移1512、24×32，SHA沿§87；參照為`redraw/ENEMY00-group1-v6-reference-04-20261003.png`。
不使用第3張作風格輸入，避免借入直抬右臂。完整提示、生成原圖、72×96正規化候選、
只讀量測及審查沿`workplace/hd/redraw/ENEMY00-group1-v9-*-04-20261004.*`。
內建imagegen製圖，ImageMagick只作整幅正規化；沒有Python修圖。現行主題與正式8/360未變，
造型及原座標正常呈現驗收通過後才選入，不以技術尺寸或外框相同宣稱動作完成。

v9原版432×576參照248832像素不符0，單像素負對照差1；整體及兩側手部外框吻合，
右側區域上緣回到原版HD y9，左側y39。靴區亮度外框右緣x66，原版x65；量測不證明輪廓相同。
人工比較已接受第3張：v9頭部將黃白色區畫成棋盤形塊，線條及明暗尚未一致，未接受。
v10以v9原生圖為編輯目標、已接受第3張只作頭部畫風參照；保留第4張身體及雙手位置。
v10產物、完整提示與只讀核對沿同目錄`ENEMY00-group1-v10-*-04-20261004.*`，不覆寫v9或現行主題。

v10頭部與已接受第3張畫風一致，雙手量測外框維持v9；原版參照核對及負對照通過。
正常呈現工作區`workplace/hd/art-pose4-runtime-v1-20261004/`，其theme僅換現行27筆的ENEMY00-04.png。
入口`prepare.py`、`runtime-source.go`及`independent.py`；完整收據`runtime.json`、`independent.json`及`source-manifest.json`。
使用研究§36的正常完整第3張state，SHA `a248291429eed019a3bb3316565859486ae53012570e2d9757b868d5e2347da8`，
按既有原版16次有向事件取正常、中途及返回樣本，另真正載回及接續。造型與正常呈現尚未通過前，
此候選不選入現行主題；靴區局部亮度外框差1個HD像素，不稱精確輪廓或RGB相同。

### 110.1 正常呈現與限定接受

【confirmed，限定正常完整第3張起點接續】正式Theme取得17份正常／中途／返回與真正載回樣本，
每份全部44個原版機器欄位及完整DOS區段相同，CPU／RAM／port負對照有效。
獨立驗證以原版16次來源證據、PBL與27筆PNG重建完整960×600圖面，17份不符0，
第4張9份HD可見，省略負對照有效。完整起點載回至第4張與暖接續圖面完全相同；
冷載中途當下清除身份回退，接續100000步恢復完整原版第4張後可重新辨識。
沒有原版寫入、改seed或重擲，沒有Ebiten視窗、中文、DAT、全部動畫、平台或封包驗收。

第一版獨立期望在pose4-entry失敗：靜態合成helper把仍完整吻合的前姿勢也自動畫上，
遺漏024 §1.3已READY的同位置互斥與中途閘門。保存independent-before.py、independent-failed.log
及independent-failure-note.json；僅依原版來源選定唯一敵人候選再比較全部像素。
同一程式、PNG、原版收據與零容差重跑通過，不排除角色或其他區域，正式程式未改。

人工完整候選與兩份正常合成檢視：第4張保持原版彎曲抬起右臂與向下左臂、緊湊手端、
黃頭黑眼及嘴、橙灰耳側、白青藍盔甲、腰部紅色小塊與分開的兩靴；頭部線條沿已接受第3張。
全圖外框與雙手位置吻合，沒有縮放、平移或借用第3張直抬右臂。靴區局部亮度外框差1個HD像素
保留量測限制，不稱輪廓、固定區RGB或整組動畫逐像素相同。造型與限定正常呈現接受第4張。
接受收據art-review-accepted.json；原版RGB與正式圖面直接合成由preview-source.go及兩份preview.png索引。

現行主題另存`workplace/hd/theme-enemy00-pose4-v1-20261004/`，27筆／26PNG，只換ENEMY00-04.png，
其餘PNG與上個pose2主題相同。selection.json記錄可重現選入；舊主題保留。
正式敵人美術9/360：ENEMY00 #0／#2–#4／#6–#8、ENEMY01 #6、ENEMY03 #3。
其他351張、30張ALLY、全部動作與重疊、新主題GUI／DAT及交付仍未完成。
準備、驗收、精確依賴保全、選入与最終同步沿本工作區prepare.py、runtime-source.go、independent.py、
preserve.py、source-manifest.json、art-review-accepted.json、selection.json、issue34-*及final-audit.json。

精確來源保全：90份實際非標準建置依賴、375份來源／收據，125份重用已核對快照、250份新副本，
67482267 bytes。Go1.24.13、Python3.11.2，沿用既有Docker工具鏈、原版唯讀及UID/GID1000。
原版PBL與正常起點只記SHA；現行主題與已驗theme的27項輸入完全相同。
#34寫前全文及時間未變，寫後正文與body-file完全一致，2026-10-03T23:37:39Z保持OPEN；#44未修改。
保全文件為收尾同步前版本；最後文件SHA另存final-documents.json，原版、state、DAT與圖片不上傳。

## 111. 第5張單一原版姿勢重畫

2026-10-04。沿024 §1.3／§1.4 READY與研究§87第5張拒絕證據，採第4張已成功的單一原版構圖流程。
原版ENEMY00.PBL #5偏移1899，24×32，SHA沿§87；唯一姿勢參照為`redraw/ENEMY00-group1-v6-reference-05-20261003.png`。
舊第5張右手上緣HD y3與原版y15不同，左侧y21與原版y27不同；不延用其錯誤手勢。
新v11原生圖、完整提示、72×96正規化、只讀量測與審查沿`workplace/hd/redraw/ENEMY00-group1-v11-*-05-20261004.*`。
內建imagegen製圖，ImageMagick只作整幅正規化；沒有Python修圖。現行主題與正式9/360維持，
造型、動作關係及正常原座標呈現通過後才選入，不能以單張尺寸或外框相同宣稱整組動畫完成。

v11原版432×576參照248832像素不符0，單像素負對照差1；整體、兩側手部與靴區亮度外框吻合。
右側上緣HD y15、左側y27恢復；人工比較頭部線條與已接受第3／4張仍不一致。
v12以v11為編輯目標、已接受第4張只作頭部畫風參照，不借用其雙手位置。
v12原圖、提示、正規化與只讀核對沿同目錄`ENEMY00-group1-v12-*-05-20261004.*`，不覆寫v11或現行主題。

進度同步核對2026-10-04 07:50台灣時間：v12已由內建imagegen生成，仍未正規化、量測、造型審查或正常呈現驗證。
原生圖與完整提示保存在上述v12檔名前綴；生成不等於接受，正式9/360與現行第4張主題不變。
本次CONTEXT／唯一worklist／GitHub #34同步收據沿既有redraw目錄，
檔名前綴`progress-refresh-pose5-20261004-*`，保存寫前、正文、寫前再次核對、寫後及稽核結果。

v12已整幅正規化為72×96，整體、兩側手部與靴區亮度外框均吻合原版；參照248832像素不符0，單像素負對照差1。
人工檢視原生圖、正規化圖與原版第5張，雙手彎曲上舉位置保持，頭部畫風與已接受第4張一致。
這只支持候選造型審查，不是正常呈現或整組動畫完成。

限定正常呈現工作區沿既有職責設在`workplace/hd/art-pose5-runtime-v1-20261004/`。
prepare.py複製現行第4張主題，只替換ENEMY00-05.png；runtime-source.go沿真正完整第3張保存起點，
對既有16次原版事件取47份正常入口／中途／返回與終點，加3份真正完整或中途載回，共50份。
independent.py由原版PBL及已驗原版來源推導唯一動作，逐像素比較完整960×600圖面與44個機器欄位／DOS，
保持同位置互斥與冷載清除契約；preview-source.go只合成實際輸出，供首循環動作檢視。
工具、二進位、依賴、收據、審查及精確來源保全沿該目錄；未驗前不更新正式接受數。

完整圖面50份獨立不符0，第5張14份正常／真正載回HD可見，省略負對照有效；
全部44個機器欄位及DOS區段相同，三項機器負對照有效。真正完整起點載回接續相同，
冷載貼圖中途清除歷史身份，只辨識當下完整原圖；此樣本辨識到仍完整的第4張，
不沿用第5張身份，第5張完整原圖恢復後才重建。
首循環五份實際合成圖已檢視，頭部畫風與雙手動作關係保持；
原版第3／4／5張底部8列逐位元相同，HD靴部陰影仍有差異，完整動畫穩定性不列通過。
preview-independent.py以已驗圖面及原版RGB獨立核對八份實際PNG，收據preview-independent.json。
第5張限定接受與現行本機選入沿art-review-accepted.json及selection.json；
新主題位置為`workplace/hd/theme-enemy00-pose5-v1-20261004/`，只換第5張，舊主題與收據保留。

正式敵人美術由9/360增至10/360：ENEMY00 #0／#2–#8、ENEMY01 #6、ENEMY03 #3。
接受限各自造型與已驗正常來源範圍；其他350張敵人、30張ALLY與完整動畫仍未完成。
本批保全90份實際非標準建置依賴及572份來源／收據，177份重用精確快照、395份新副本，159523221 bytes。
原版PBL與正常完整起點只記SHA。source-manifest.json保存同步前精確來源，最後文件SHA另存final-documents.json。
CONTEXT、唯一worklist及GitHub #34同步沿該工作區issue34-*及final-audit.json，素材不上傳。
#34寫前全文與時間未變，寫後與body-file完全一致，2026-10-04T00:11:51Z保持OPEN；#44狀態與時間未改。

## 112. 現行第5張主題的正式修洛斯視窗抽測

2026-10-04。沿024 §1.3／§1.4 READY、§111現行主題及§84已通過的普通前端方法，
工作區`workplace/hd/gui-shulosu-v1-20261004/`。不延伸§96已停止的繪圖觀察原型。
prepare-start.go由已驗正常07-first-play.state及中文Layer，沿§36原版六次Up鍵序接續，
到§111完整第3張起點，與原版不可變state核對44個機器欄位及DOS後保存中文側檔。
run.py使用本次建置的未加觀察器正式前端，真正F11後取中英文／HD開關四模式各十份F10狀態及實際視窗。
F10保存與畫面可能不同步，全部擷取保留；verify.py依完整原圖、正式文本、字型、提示與8×8規則逐份核對全部像素。
入口還有Go依賴清單、export-state、準備／執行／零步匯出／驗證收據及精確來源保全。
未核對前不稱GUI通過，不代表從開機、整組動畫穩定性、DAT、自然時序、實機或封包。

起點準備首版退出1，診斷frame與A0000h以下RAM相同；原版不可變state為83214859步，
工具誤用§111第一份取樣的83214860步，已多執行RET，SP／IP與cycles因而不同。
before-diagnostic-*、diagnostic.json／state及preparation-diagnostic.log保留。
prepare-start-v2.go只改成原版state的實際步數，不改原版、控制state或其他條件。

v2的暫存器、步數、cycles、畫面與A0000h以下RAM吻合，但44欄位閘門仍退出2。
state-diagnostic.go／json證實唯一差異是Latch：正常重播F0／F0／F0／F0，舊state全0；DOS區段相同。
舊tools/hd/verify_enemy_runtime.go返回掛鉤在SaveStateFile前呼叫Bytes(Addr{},1<<20)，
oracle.Bytes經Machine.Read8與VGA.Read載入四個平面的Latch；Indexed走平面像素解碼，不走這條讀取。
validate-start.go以相同正常07起點及六次Up鍵序、相同83214859步重跑無中文控制，保存前不讀VRAM。
先比較準備起點與新控制的44欄位／DOS，再以獨立副本演示1 MB bus讀取是否只改Latch並吻合舊state；
示範副本不用於GUI，也不把Latch手動改成0或放寬正式比較器。舊試驗state／收據維持其原觀察條件。

無中文控制與準備state的44欄位／DOS皆相同，三項負對照有效；bus讀取副本與舊state也全部相同。
首次普通GUI在F11後退出1、擷取0份，captures/保留。輸入是舊probe的指令數時鐘，
F11還原後RunCycles無法執行；普通前端main.go載入後本來會設AdLib與750 cycles，測試fixture漏了此初始化。
prepare-frontend.go沿相同正式初始化，從已驗準備state建立週期時鐘GUI輸入，保留原版寄存器／RAM／畫面，
另起獨立控制核對44欄位／DOS。不手改序列化欄位，不修改quickLoad或正式玩法。
before-frontend-clock-run.py與before-frontend-clock-verify.py保留首次工具；captures-v2/沿同一40份門檻重跑。

重跑普通前端退出0，captures-v2/保存四模式各10張，共40張實際視窗。run-v2.log記錄擷取數；
frontend-preparation.json確認起點與獨立控制44個機器欄位及DOS相同。
獨立逐像素比對尚未執行，未產生verified.json；不稱GUI／F11通過，不增加正式美術10/360。
使用者要求先同步進度，CONTEXT及GitHub #34更新為此待驗狀態；
同工作區progress-refresh-issue34-before.json、progress-refresh-issue34-body.md及progress-refresh-issue34-after.json保存同步證據。
#34寫前全文與時間未變，寫後正文與body-file完全相同，2026-10-04T00:52:27Z保持OPEN；#44未修改。

獨立比對後，40份中35份完整960×600視窗不符0，涵蓋第3／4／5張、中英文與HD開关四模式。
五份位於貼圖中途或原版enemy／ALLY來源不完整，保留unverified，不排除區域或猜補來源。
HD三姿勢兩語全屏吻合，省略角色與錯姿勢負對照有效；35份均包含獨立正式譯文、字型及輸入推導的提示列期望。
八份實際PNG已檢視，含三姿勢兩語HD及兩個原版模式；原座標、人物在後框在前、文字及切換有限呈現正確。
verify.py原稿的else分支語法錯誤已保留為verify-before-syntax-fix.py，僅修original_covered分支；
export-state首次缺專案cwd而停止，改為/src後零步匯出40份，未重跑或改動原始擷取。
verified.json與verify.log保存完整結果。這是正常檢查點接續與真正F11的有限GUI驗證，
不代表從開機、全場／全部動畫、DAT、自然時序、實機或封包完成。
preserve.py、source-manifest.json及source-snapshot/保存本批實際來源與收據；原版素材僅記雜湊。

## 113. 首組第1張改用完整原版參照重畫

2026-10-04。沿§109停止舊局部修正的限制，使用唯一完整原版參照重新生成v16，
不把v15當編輯底圖。已授權的原座標、比例、色區與動作契約維持024 §1／§6。
原版參照為workplace/hd/redraw/ENEMY00-group0-v5-source-01-20261003.png；
本批ENEMY00-group0-v16-generation-01-20261004.json保存完整提示，generated／frame保存原生與整幅正規化候選。
量測來源及measurements收據沿同前綴保存。未審查前不接受、選入或增加10/360。

v16原生候選保留方塊階梯與逐塊倒角，尚未達已接受第0／2張的平滑畫風。
v17以v16為編輯底圖，第0張已接受PNG只供筆觸與陰影參照，原版第1張只供姿勢及色區；
完整提示與原生／正規化候選沿ENEMY00-group0-v17-*-01-20261004保存，未選入。

v16／v17整體、左右靴與左右臂的亮度外框均吻合原版，原版參照248832像素不符0，單像素負對照有效。
右臂白色亮度外框受灰色高光影響，不直接當白帶幾何。v17完整原生及72×96已檢視，色區與黑色缺口保留。
候選執行期工作區workplace/hd/art-pose1-runtime-v1-20261004/，prepare.py建立candidate-theme/，
從現行第5張主題僅替換第1張，驗證來源沿§108正常第0張起點與117次原版事件。
run.go／verify.py／preview.go及各收據保存正常接續、真正完整起點及第1張中途載回；未通過前不接受。

v17正常接續與真正載回共17份完整960×600正式圖面不符0，44個原版機器欄位及DOS區段相同，三項機器負對照有效。
第1張7份HD可見，省略負對照有效；正常與真正完整起點載回接續相同。
兩份完整RGB合成獨立不符0並已檢視，保留原版戰鬥效果覆蓋；不宣稱無遮擋全身、完整動畫、中文或實際視窗。
v17保留原版頭部、雙天線、棕黃下臉、黑色缺口、白灰青紅色區、雙臂與非對稱靴子，
左靴x3與右臂x53外緣恢復，右前臂L形白帶沿原版色區。灰色高光使亮度分類白框外擴，不能宣稱白色像素相同。
完整候選及正常呈現已限定接受，art-review-accepted.json保存審查；正式敵人美術11/360，其他349張與30張ALLY未完成。
新本機主題workplace/hd/theme-kasuruji-pose1-v1-20261004/維持27筆／26PNG，僅換ENEMY00-01.png；selection.json記錄來源及選入。
preserve.py、source-manifest.json及source-snapshot/保存本批實際建置依賴、來源與收據；原版素材僅記SHA。
此新主題未做Ebiten GUI／DAT驗收；§112的35份完整GUI屬前版第5張主題，不自動延伸。

本批來源保全331項，322份精確本機副本共63410417 bytes，實際非標準Go依賴88份，原版九項僅記SHA。
§112 GUI來源保全830項、821份副本共81302368 bytes，實際依賴351份；兩批source-manifest.json各記工具與輸入雜湊。
證據等級：上述限定機器／圖面／視窗實驗結果為confirmed；美術是限定審查接受，不稱原版RGB逐像素相同。
本批issue34-before.json／body.md／after.json保存遠端同步；寫前全文與時間未變，寫後正文完全一致，
2026-10-04T01:14:20Z保持OPEN。CONTEXT／024現況／worklist同步11/360；render／verify通過，#34及#44兩項仍未完成。
final-audit.py／json核對兩批精確快照、原版SHA及現行選入；source-supplement.json保存最後文件與同步收據精確SHA。
素材留本機，未commit／push／發行，#44未修改。

## 114. 修洛斯固定靴部的跨姿勢一致性

2026-10-04。沿024 §1.3／§1.4／§6與§111的已知靴部差異，現行主題仍為§113的第1張主題，11/360。
原版ENEMY00.PBL #3／#4／#5各24×32，底部8列逐byte相同，證據等級confirmed。
現行三張HD底部24列兩兩RGB像素不符1096／1068／1025，不能宣稱固定靴區動畫一致。
第3張原生參照為workplace/hd/redraw/ENEMY00-group1-v5-generated-03-20261003.png；原版色區仍以同組source／reference為準。
先用imagegen修第3張底部靴區，保留其頭部、身體、雙手及姿勢；再供其他姿勢比對，不修改原版或正式程式。
第3張本批generation／generated／frame／measurements沿ENEMY00-group1-v6-*-03-20261004保存；
只用ImageMagick整幅正規化，不做Python局部修圖。候選未通過前不選入，不增加完成數。

量測工具ENEMY00-group1-v6-measure-03-20261004.py讀取原版及三張現行PNG，
核對原版432×576參照、單像素負對照、底部色區與跨姿勢RGB差異。RGB最近色盤分類只作定位，不取代造型或動畫審查。

第3張v6雙靴及整體亮度外框吻合，最近色盤分類的底列藍色由38像素降至0，原版也是0；完整候選已檢視。
上部72列RGB有2694像素差異，不能稱上部逐像素保持；頭部、手臂、身體及姿勢需照常審查。
第4張v11及第5張v13以各自已接受原生圖為底，第3張v6只供固定靴區，原版各姿勢只供位置與色區。
完整提示、原生及正規化候選沿ENEMY00-group1-v11-*-04-20261004及v13-*-05-20261004保存，未選入。

本批候選整體核對與來源保全工作區workplace/hd/art-boots-v1-20261004/，
measure.py／measurements.json記三姿勢靴區、上部RGB差異與原版部位外框；本批候選尚未選入。

三張候選已檢視並量測，靴區RGB兩兩差異仍為1042／1015／1013；最近色盤分類第4／5張由149增至179，不能宣稱整組一致。
第4張右靴亮度外框多1個HD像素，上部RGB亦皆改變。保留改善與未達要求部分，本批不選入，11/360與現行主題不變。
停止用此獨立重生成方法追求固定區完全一致，接續其他已READY素材，不放寬位置或動畫要求。

## 115. 歐格斯第7張改用完整原版姿勢重畫

2026-10-04。沿024 §1.7 READY與§98／§103的停止線，從唯一原版第7張完整參照重画v6，不沿用局部修嘴方法。
原版ENEMY01.PBL第7張偏移2890，24×32；原圖下唇y11即三倍y33–35，右側黑色嘴部間隙y27–32。
紅眼原版色區三倍外框[60,18,68,23]；左下突出部[0,63,11,80]，來自原始色號，證據等級confirmed。
完整提示、原生／正規化候選沿workplace/hd/redraw/ENEMY01-group2-v6-*-07-20261004保存；
measure工具與收據同前綴保存。原版參照ENEMY01-group2-v5-source-preview-07-20261004.png。
候選未審查前不接受、選入或增加11/360；素材與來源不公開。

第7張v6原版參照248832像素不符0，單像素負對照有效。下唇確實位於y33–35，y36–38黑色；
紅眼[60,18,68,23]與左下突出部[0,63,11,80]外框吻合。原生候選仍保留方塊階梯，未接受。
v7以v6為編輯底圖、已接受第6張只供畫風，修平滑輪廓但保留下唇與嘴部間隙；各檔沿ENEMY01-group2-v7-*-07-20261004保存。

既有§1.7／§83正常收據只有第6張完整可見，七個差分返回因原版特效覆蓋而沒有完整來源，正式門檻回退。
ENEMY01-group2-source-gate-review-20261004.py／json沿redraw保存只讀核對：
以既有八次真實來源及原版差分解碼，再逐格計算邏輯目標可吻合的8×8格。這是候選來源模型，不是正式身份或HD呈現證據。
若需要擴充受遮擋身份，須補RE及DRAFT，不能直接把§1.6.1的ENEMY00限制解除。

v7平滑畫風候選已檢視，保持下唇y33–35、下方黑色、紅眼及左下突出部外框；仍未經正常HD呈現，不接受或選入。
既有八次正常原版來源與before／after逐像素重建不符0，七個受遮擋返回各仍有8個非空8×8格吻合邏輯目標。
此結果是confirmed的原版圖格條件，不证明正式身份可持续，下一步另核對完整相交貼圖。

## 116. 歐格斯受遮擋身體的來源與身份證據

2026-10-04。沿024 §1.7的已知回退與§115窄模型，從既有正常Sivad事件0完整第6張state接續，
工作區workplace/hd/oogus-context-v1-20261004/。prepare.py由§107既有觀察／獨立重建工具建立本批observe.go與verify.py，
只改原版來源檔、已知圖號6–8、正常第6張起點及證據索引，不改正式程式或原版資料。
紀錄15,000,000步內全部與角色區相交的8705／8751一般貼圖、8260／4E34固定遮罩，
原始before／after、來源bytes、CS:IP、DS:BX及矩形。原始來源指令及證據等級沿§107，新增Sivad結果獨立審查。
trace.json、independent-v2.json、build-deps.jsonl及兩側state由本節索引；原版僅記SHA。
未達證據與READY閘門前不擴充ENEMY01 retainDelta，不把來源模型當HD或GUI驗收。

首批15,000,000步觀察共39次一般貼圖、0次固定遮罩，兩側CPU／cycles／RAM／frame相同；
逐事件before／after重建吻合，但最後角色區全黑，較最後一次after多405像素清場，未走已記錄貼圖。
verify.py终點閘門退出1，保留trace與兩側state，不能稱完整來源模型通過。
另存narrow/核對最後已觀察貼圖返回667244777步的接續點667244778，範圍由原始trace決定，不改seed或原版狀態。
observe-narrow.go／verify-narrow.py保留限定片段工具與負對照；清場另列生命週期契約待審，不把縮窄片段冒充清場已解。

限定片段39次一般貼圖，逐事件與終點角色區來源重建不符0；七次身體轉換為7／8／7／6／7／8／7。
narrow/independent-v2.json記錄三項來源負對照有效；narrow/machine.json記44個機器欄位與DOS區段相同。
第7張邏輯目標在20次返回有可吻合原版圖格，這不證明正式HD身份或候選美術已接受。
本片段沒有固定遮罩事件，不宣稱遮罩負對照通過；較長測試未解405像素清場的失敗保留。
本次進度同步工具progress-refresh.py與progress-issue34-before.json／body.md／after.json及progress-sync.json沿同工作區保存，
只更新交接、唯一worklist及使用者授權的#34正文；原版、state、DAT與圖片未上傳。

## 117. 歐格斯戰鬥清場的首次寫入定位

2026-10-04。從§116最後已觀察貼圖返回後的narrow/control.state接續，定位未記錄的角色區清場。
沿同工作區保存clear-locate.go／json／log、before／after.frame與state、control.state及實際build-deps。
先以1024步片段找到角色區首次變化，再從同一不可變起點重播到安全邊界逐指令定位。
不改seed、原版RAM、按鍵、EXE或正式程式；不能把首次寫入當完整清場或身份生命週期已驗。

首次角色區改畫指令為執行期0161:7205、IDA線性17715h的F3 A4，原始sub_176CC。
首次改變405個角色像素，非空像素由405降為402，並非一道指令清空角色；
before／after整幅320×200逐像素證實向上移一列，底列保持，證據等級confirmed。
IDA 9.4從唯讀4db51a199f5913c52dc4964557b9a4aafb965754d663cb0173c9dfdfe9a00c56資料庫副本匯出，
clear-ida-probe.py／json核對版本及417函式，clear-ida-query.txt保留sub_176CC、原始bytes與sub_1AC50跳轉來源，不改名稱。

prepare-scroll.py／observe-scroll.go與scroll/保存同起點15,000,000步的39次packed與200次全畫面捲動。
prepare-scroll-verify.py／verify-scroll.py及scroll/independent-v3.json由原始來源重建全部239次角色區before／after及終點，不符0。
捲動整幅獨立期望不符0，五项來源負對照有效；44個機器欄位及DOS完全相同。
第一列捲動前錨點吻合、捲動後失配，既有錨點清除身份規則足以處理這條清場。
首版生成器註解定位錯誤退出1，精確來源存prepare-scroll-verify-before-marker-fix.py；只修生成器定位，同收據重跑通過。
§116未觀察捲動的405像素終點失敗仍保留，已由本批來源補齊，不再列為未知清場。

## 118. 歐格斯受遮擋動作的契約與正式接入

2026-10-04。沿§117已充分原版來源，024 §1.7.1先保存DRAFT，再以review-contract.py／contract-review.json審查。
15份同位置已知原圖唯一、四條本組有向差分及15項契約案例通過，包含真實第一列捲動錨點失配。
審查後標READY，只允許ENEMY01 #6–#8使用同位置唯一有效前身份與384 bytes原始差分。
未知來源、完整覆蓋、歧義、錨點失配及載回仍清除，其他組範圍不擴張；不加入效果分解或捲動掛鉤。

正式來源修改前bytes保存同工作區enemy-before.go／ally-before.go／body-context-test-before.go。
prepare-runtime.py／candidate-preparation.json建立本機candidate-theme，只換ENEMY01-07.png為§115 v7，尚未選入。
verify-runtime.go與runtime/保存正常第6張起點接續、貼圖中途、捲動及真正載回樣本，完整圖面與機器驗證須獨立核對。
原版、state、DAT及圖片留本機，不以技術接入或候選尺寸宣稱美術接受或完整HD完成。

實作及有限驗證已完成：ENEMY00與ENEMY01各16項生命週期及既有Theme回歸通過。
首跑未掛/hd/bg.idx造成既有背景測試失敗，tests-missing-background-fixture.log保留；補只讀掛載後同測試重跑通過，非產品缺陷。
27份完整960×600正式圖面不符0，所有44個機器欄位與DOS相同，三項機器負對照有效；
第7張13份HD可見，省略負對照有效。完整起點真正載回接續相同，中途冷載清除歷史身份，第一列捲動後HD圖面全空。
四份完整合成獨立不符0，原生、72×96、原版參照、正常／完整載回及捲動合成已檢視。
第7張v7保留開口下唇、紅眼、青紫藍色區、左側突出部與不對稱雙腿，平滑畫風沿第6張；
部位外框吻合不稱RGB或輪廓逐像素相同。art-review-accepted.json限定接受，accept-select.py建立新主題。
現行workplace/hd/theme-oogus-pose7-v1-20261004/維持27筆／26PNG，只換ENEMY01-07.png，selection.json記錄12/360。
其他348張敵人、30張ALLY、第8張、完整動畫、新主題GUI／中文／DAT、權利與交付仍未完成。
來源保全入口preserve.py／source-manifest.json及source-snapshot/，原版檔案與正式IDA資料庫僅記SHA；
遠端進度同步沿issue34-current-before.json／body.md／after.json與sync.json保存，#34保持OPEN，#44未修改。

本批479份精確本機來源／收據副本共167575312 bytes；原版定位及捲動觀察各80份實際非標準Go依賴，正式Theme驗證90份。
原版素材、正常起點state及正式IDA資料庫僅記SHA，DRAFT階段規格與READY後文件以精確副本與輸入解析對照保留。
final-audit.py／json核對來源快照、原版SHA、現行選入及正式main.go未改；最後文件另存final-documents.json。

## 119. 歐格斯第8張從唯一原版姿勢重畫

2026-10-04。沿024 §1.7／§1.7.1 READY及§103的局部反覆生成停止線，從唯一原版第8張完整參照重畫v8。
原版ENEMY01.PBL SHA與第8張檔案偏移3269、24×32沿§103；突出部三倍範圍[3,60,11,80]，紅眼[60,15,68,20]。
上下空白區須包含在量測內，不能用窄範圍掩蓋越界；不以外框相同宣稱輪廓或RGB相同。
提示與原生／正規化候選沿workplace/hd/redraw/ENEMY01-group2-v8-*-08-20261004保存，內建imagegen重画、ImageMagick只作整幅正規化。
候選未審查與正常呈現通過前不選入，現行§118主題及12/360維持；不改正式程式、原版或清場契約。

v8已生成並整幅正規化為72×96，ENEMY01-group2-v8-measurements-08-20261004.json保存唯讀靜態量測。
原版參照248832像素不符0，單像素負對照有效；突出部[3,60,11,80]、紅眼[60,15,68,20]及整體亮度外框與原版吻合。
擴大突出部量測範圍包含上下空白區，亮度大於60的越界像素0，非零RGB像素11；不能稱為全黑、輪廓或RGB逐像素相同。
完整造型、畫風與正常遊戲呈現尚未審查通過，accepted=false，現行主題與正式12/360不變。
本次使用者要求的進度同步收據沿同redraw目錄progress-v8-issue34-before.json／body.md／prewrite.json／after.json與sync.json保存。

原生v8審查發現色塊與右腿棋盤紋過於鼓起，與已接受第7張平滑畫風不同。
同目錄ENEMY01-group2-v9-*-08-20261004保存一次畫風修正：v8為編輯目標，第7張只供畫風，唯一原版第8張供姿勢、座標及色區。
全幅正規化與原版範圍重新量測後，再審查完整造型與正常呈現；未通過前不選入。

v9亮度外框與突出部吻合，原紅眼R>200量測的底列為y19，原版為y20。
直接讀回y20像素R178–191、G65–98、B68–102，顯示底列存在暗紅陰影；原量測不修改。
另以色相分類核對座標，並以平移眼區負對照排除分類永遠通過。初版R>G+B誤把(43,26)的暗洋紅列為紅色，範圍[43,15,68,26]不通過。
改依EGA12的R=3G=3B，用R>2G、R>2B且R>60區分紅色陰影與暗洋紅，加入洋紅與灰色負對照；原亮度量測不改。
唯讀工具與收據沿redraw/ENEMY01-group2-v9-color-review-08-20261004.py／json保存，不寫圖片。
正常正式呈現驗證沿workplace/hd/art-oogus-pose8-v1-20261004/保存prepare.py、verify-runtime.go、verify-plane.py、preview.go、verify-preview.py及runtime/。
候選主題只複製現行27份正式素材並替換ENEMY01-08.png，不改位置、規格、正式程式或原版起點；本批未通過前仍未選入。

色相核對[60,15,68,20]與原版相同，移位、洋紅及灰色負對照有效。首跑PBL tuple解包錯誤，修為既有offset索引後重跑；分類誤判另記上段。
v9保留第8張姿勢、色區、向下下唇、彎曲突出部及不對稱雙腿，平滑陰影與棋盤紋沿已接受第7張；原生及72×96已檢視。

## 120. 歐格斯第8張的限定正式呈現與選入

2026-10-04。沿§119工作區，同一正常完整第6張不可變起點、seed F95Bh，沒有改原版RAM、輸入或重擲。
正式Theme接續既有七次身體轉換、第一列捲動及終點，再真正載回完整起點和第8張中途，共27份完整960×600圖面。
runtime/independent.json記27份完整圖面不符0、44個機器欄位與DOS區段相同，三項機器負對照有效；第8張7份HD可見，省略負對照有效。
完整起點載回接續相同，中途冷載清除歷史身份，第一列捲動後HD圖面全空；不猜補身份。
四份完整合成不符0、單像素負對照有效，正常、重複動作、完整載回及捲動PNG均已檢視。
圖面驗證首跑錯把基準主題路徑改成尚未存在的第8張主題，屬驗證腳本錯誤；原稿verify-plane-before-baseline-fix.py保留，修正路徑後同一收據重跑，未改產品或重新擷取。
art-review-accepted.json與accept-select.py記第8張v9限定接受及選入；新主題workplace/hd/theme-oogus-pose8-v1-20261004/維持27筆／26PNG，只換ENEMY01-08.png，正式13/360。
部位亮度／色相外框不證明RGB或輪廓逐像素相同；R>200的紅眼陰影底列差異、10個空白區非零RGB像素保留。
這批不代表全部動畫、新主題Ebiten GUI、中文、DAT、效能、平台或HD封包通過。其他347張敵人、30張ALLY及完整動畫仍待完成。
本批來源保全與稽核沿preserve.py、source-manifest.json、source-snapshot/保存，原版素材及state只記雜湊；進度同步沿issue34-before／prewrite／body／after／sync保存。

來源保全為266份精確本機副本、88份實際非標準Go依賴，45份原版／衍生state與frame僅記SHA。最後核對沿final-audit.py／json及final-documents.json保存，不覆寫原始收據。

## 121. 現行歐格斯主題的普通前端驗證

2026-10-04。工作區workplace/hd/gui-oogus-v1-20261004/沿既有正常視窗流程，驗現行第8張主題。
prepare.py／prepare-frontend.go沿原版完整第6張replay-event00.state與同路徑正式中文側檔，來源為§50／§52的正常鍵序重播；不重新編造譯文或人物身份。
以正式前端載入後相同AdLib／cycles設定建立F11輸入，零步保持原版RAM、寄存器及畫面，兩側44欄位與DOS核對。
run.py／captures/以未加觀察器的正式cmd/psychicwar建置，真正F11後正常時間接續，四種語言／HD組合各10份F10狀態與實際視窗PNG。
外部SIGSTOP只固定擷取，不改原版資料；PNG與state是否同幀須由verify.py獨立完整合成核對。
export-state與build-deps-*.jsonl保存原版零步匯出及實際建置依賴；沒有完整來源的受遮擋樣本不稱完成。
DAT存讀檔將沿同工作區的正常鍵序另存收據。本批尚未執行／通過，正式13/360與主題27筆／26PNG不變，不改正式程式、文本或字型。

普通前端40份擷取完成、退出0，record沒有送入原版的新按鍵。完整原圖判準因40份均受效果遮擋而未驗，captures/verified.json保留，不能當呈現已通過。
prepare-replay.py／observe-replay.go／replay/沿同真正F11起點重播，逐擷取步數核對原版狀態，再由已證實來源歷史核對當下姿勢。
此為獨立驗證工具，不替正式前端加入觀察器、不改原版，且不把較窄的角色區通過當作完整視窗通過。

40份實際F10狀態與同起點無原版新按鍵重播的44欄位及DOS全部相同；239次來源模型與200次捲動重建不符0，五項來源負對照有效。
prepare-verification.py／verify-source-aware.py沿原始來源歷史認定受遮擋姿勢，再核對同一40份完整視窗，保留舊完整原圖判準的未驗結果。

來源歷史版證實40份完整敵人矩形吻合，20份關閉HD的完整視窗吻合；20份HD視窗尚有全圖差異，保留未通過。
prepare-replay-v2.py／observe-replay-v2.go／replay-v2/補記盟友矩形的完整覆蓋來源，釐清既有ALLY #0在局部遮擋下保留的身份，不改正式判定。

補記只見667513844步的一次ALLY完整覆蓋，先前局部效果不清除起點已完整的ALLY #0身份。
prepare-verification-v3.py／verify-source-aware-v3.py依同原版覆蓋事件保留／清除盟友身份；舊模型漏掉這層，不改產品以迎合測試。
首次F10固定的是上一張已繪視窗，F11提示可能仍在該幀；只從正式提示原文及實際F11輸入建立空白／已讀檔兩種合法期望，整幅比對及提示負對照仍須成立，不排除提示列。

最終captures/verified-source-aware-v3.json證實40/40完整960×600視窗不符0，20份HD與20份原版，涵蓋中英文及歐格斯#6／#7／#8。敵人省略、錯姿勢、ALLY省略及提示省略負對照有效，八份實際完整PNG已檢視。40份實際F10狀態與獨立原版重播的44個機器欄位及DOS均相同；replay-v2/independent.json的239次來源與200次全圖捲動重建通過，五項來源負對照有效。

此結果為confirmed的限定普通前端、真正F11及中文／HD開關抽測。起點是正常鍵序保存的完整第6張，seed F95Bh；正式前端未加觀察器，正式程式、文本、字型及現行主題未改。先前失敗收據保留，屬期望模型漏記盟友保留身份與首張提示，不是產品缺陷。未涵蓋從開機、整場戰鬥、完整動畫、DAT存讀、自然時序、效能、音訊、封包或真機。

本批來源保全沿同工作區preserve.py、source-manifest.json及source-snapshot/；採實際五份build-deps-*.jsonl的非標準依賴聯集。原版EXE／PBL、state、frame、RGB及玩家資料只記雜湊；正式程式、測試來源、40張PNG、主題及文字字型保存本機精確副本。進度同步收據沿issue34-before.json／prewrite.json／body.md／after.json／sync.json；最後文件沿final-documents.json及document-snapshot/。全部留本機，不上傳圖片或原版資料。

來源保全完成：709份精確本機副本、350份實際非標準Go依賴，264份原版／衍生資料只記雜湊。現行27項主題輸入及正式main.go SHA保持相同。

## 122. 現行歐格斯主題的正常 DAT 存讀抽測

2026-10-04。沿§121工作區gui-oogus-v1-20261004/，使用現行正式前端及第8張主題。prepare-dat.py／run-dat.py、dat-save-actions.json／dat-load-actions.json保存正常键序；dat-prepared-save/與dat-prepared-load/由§102正常Forget it後迷宮及SELECT狀態建立同正式初始化的F11輸入，原版RAM、寄存器、畫面不改，44欄位與DOS控制獨立核對。

dat-save-v1/走Esc→Options→Save Game→Definitely→HD13檔名，再返回迷宮；dat-load-v1/走SELECT→Load Game→HD13，再比較四模式與正常移動。dat-independent-v1/由同一實際保存前狀態及等價原版鍵序、不載HD／中文，重生512 bytes DAT作獨立對照；不使用舊主題收據直接宣稱新主題通過。

prepare-dat.py、verify-dat-screen.py、verify-dat.py、preserve-dat.py、build-deps-dat-*.jsonl與dat-verified.json為本批工具及證據入口。實際PNG、F10狀態、原版零步匯出、鍵盤紀錄、終止碼及原版DAT留各自輸出目錄。dat-source-manifest.json／dat-source-snapshot/保存實際工具及來源，原版素材、state、DAT與派生frame僅記SHA；最終文件沿dat-final-documents.json及dat-document-snapshot/，Issue同步沿dat-issue34-*。範圍限正常保存起點接續與迷宮DAT，不代表從開機、戰鬥DAT、全部sprite、自然亂數逐次相同、效能、音訊、真機或正式封包。本批已完成下列限定驗證。

本批confirmed：15/15完整960×600視窗不符0，七份完整Save Game選單與八份Load Game／四模式／正常前進。兩個普通正式前端均退出0，原版按鍵紀錄與指定鍵序相同；起點44欄位及DOS控制相同。五份實際PNG已檢視。HD13.DAT為512 bytes，SHA-256 3468b6ec00439ceaad19bb121ec2f57214f80516d12ba3c289c58fe076d96df1，與本次同實際保存前state的獨立無中文／HD原版重生結果相同。四模式載回52 bytes等於保存前，正常前進後與獨立原版相同且確實改變玩家資料；DAT及玩家一byte、完整圖面一像素負對照有效。

第一輪全屏驗證45秒逾時，未產生驗證收據；保留dat-verify-save-screen.log，使用同擷取及同判準以180秒期限重跑，七份全屏通過，不分類為產品缺陷。SELECT起點未重印原文保留既有局限，不列全中文證據。正式main.go SHA、主題、文本與字型未改，正式13/360與27筆／26PNG不變。735份精確來源副本含453份重用，353份實際非標準Go依賴，90份原版／衍生資料僅記SHA。工具鏈沿§121的既有image／Go1.24.13／Python3.11.2／ImageMagick6.9.11-60，原版素材唯讀，UID/GID1000。

## 123. 葛雷戈林第4張原版單格重新繪製

進度同步入口：2026-10-04 12:42依本節既有收據核對CONTEXT與GitHub #34，未新增測試或接受。同步正文與讀寫核對收據保存在同工作區`progress-sync-20261004-1242-body.md`及`progress-sync-20261004-1242-{before,prewrite,after,sync}.json`；未完成權威仍為`docs/worklist.json`。

2026-10-04。沿024 §1.14 READY、原版位置及已接受第3張畫風，原版第4張為唯一幾何／姿勢／色區參照。不沿舊v5端點局部微調；固定外框、左條與右鉤及底部範圍均由原版資料讀取。入口為既有redraw/下ENEMY03-group1-v6-generation／generated／frame／measure／measurements／review-04-20261004，完整提示及原生圖保留。只用內建image_gen與ImageMagick整幅72×96正規化，不裁切、平移或Python修圖；讀取像素及幾何量測不編修圖片。候選未接受前保持現行13/360、27筆／26PNG及§121–122通過範圍。若候選未達標，記錄具體差異，不以合成通過代替美術接受。

v6原版參照248832像素不符0，單像素及鉤端平移負對照有效。新候選左圓塊[4,34,31,56]，原版[3,36,32,59]；右圓塊[45,38,70,65]，原版[45,39,68,68]；黃條[40,23,50,24]，原版[39,24,50,26]；右鉤量測[39,75,63,86]，原版[45,75,62,83]。原生及72×96已檢視，未接受。再以原版單一參照建立v7，先只驗幾何及色區；不沿用其他姿勢的畫風圖。v7-generation／generated／frame／measure／measurements／review-04-20261004沿同命名入口，不覆寫v6。

v7完整參照及負對照通過，五組色區／亮度外框與原版吻合，鉤端量測ROI也相同；白條量測白色30像素，原版27。原生及72×96已檢視，圓角像素塊畫風未接受。v8僅以v7為目標調整線條與陰影，第3張只供畫風、原版第4張只供幾何；v8-generation／generated／frame／measure／measurements／review-04-20261004沿同入口，仍未接受或選入。外框吻合不宣稱輪廓或RGB逐像素相同。

v8原版參照與負對照通過，五組量測外框及右鉤ROI與原版相同，色區像素數仍不同，白條40對原版27。原生及72×96已檢視，姿勢與色區、黑色缺口及平滑線條造型審查通過；正式接受仍待正常HD呈現，不增加13/360。候選與接入驗證沿art-zellwal-pose4-v1-20261004/：prepare.py、theme/、build-deps.jsonl、verify-normal、normal-attack/、verify-plane.py、display-review.json、source-manifest.json／source-snapshot/、final-documents.json及issue34-*。正常原版模型沿§90的advance-east-attack/observation.json，原始按鍵、seed F95Bh及終點HP0保留；不靠改HP或注入姿勢驗候選。

候選準備首跑漏掛/orig而中止於來源SHA讀取，沒有準備收據；補原版唯讀掛載後同來源完成。正常Go驗證首跑漏掛原版收據的/output別名，runtime.log保留，normal-attack/只有空目錄。原版路徑不改寫，補掛zellwal-sprite-route-v1-20261003/至/output:ro後，以同二進位及鍵序存normal-attack-v2/與runtime-v2.log。兩者是環境掛載问题，不改產品、圖片、原版收據或判準。

正常候選18份完整圖面／中文／合成及18份真正載回獨立不符0，第3張僅一份HD可見；第4張需另驗可見性，不以總體通過接受。pose4-display-review.py與review-plane-model.py沿同工作區保存只讀可見性及已證實來源身份下的8×8可用格量測；後者是從verify_ally_items.py複製、僅明示本候選28筆，第一輪誤用只收27筆的模型已被拒絕，沒有產生接受收據。preserve.py保存本批精確工具／美術／收據及實際依賴，原版bytes只記雜湊。

第4張可見閘門未通過：sample06／12兩次原版第4張返回，正式完整圖面與獨立期望吻合，但第4張身份未建立，回退原版。依已證實原版來源身份計算，兩份各有4608個RGBA像素符合既定8×8格；此分析投影不是正式呈現，也不猜補身份或改判準。pose4-frame-audit.json另核對全部157份既有原版tick，第4張完整原图相符0，實際第3張作正對照。v8靜態造型／色區審查通過，仍未正式接受或選入，13/360不變。下一閘門為第5張同畫風候選與ENEMY03 #3–#5受遮擋前身份的來源／契約審查；達READY並驗真正HD可見後才選入，不重跑現行27筆已通過的GUI／DAT。

## 124. 葛雷戈林第5張原版單格重新繪製

2026-10-04。接續§123來源可見閘門，原版第5張為唯一幾何／色區參照，先重畫v7再依量測結果決定畫風調整。第5張下條為青色，短鉤原版三倍範圍[51,75,59,77]，不沿用第4張白條或長鉤。原版ENEMY03.PBL雜湊沿§123，檔案偏移2040、解碼色號SHA-256 e5be9d08953469dfeed83ca1aba9555f92b716e8a876b4a742d531a7ca6822e9，位址基準為PBL檔案偏移，非執行位址。

本批入口為既有redraw/下`ENEMY03-group1-v7-{generation,generated,frame,measure,measurements,review}-05-20261004`；若幾何通過後調畫風，以v8同命名保留，不覆寫v7。內建imagegen生成，ImageMagick只作整幅72×96正規化；Python只读量測與收據，不修圖。原版參照逐像素驗證、單像素及短鉤平移負對照、主要色區外框與整體視覺審查均須記錄。未實際通過前不接受或選入，正式13/360及現行27筆／26PNG不變。

v7原版參照248832像素不符0，單像素與短鉤平移負對照有效；整體、左圓塊、頭部黃條與短鉤外框吻合，鉤端27個原版位置均屬候選紅色群。右圓塊量測[45,39,71,68]對原版[45,39,68,68]，右灰塊亮度分類擴入白色群；下條白色2／青色43，原版白色0／青色63。原生及72×96已檢視，逐塊鼓起畫風、灰色高光與眼內黑縫不接受。v8以v7為編輯目標、第4張只供平滑畫風、原版第5張只供幾何及色區，重新核對全部外框；不能稱v7幾何全部通過。

v8原生及72×96已檢視，平滑線條、實心黄楔、青色下條及短鉤關係保留。四組量測外框吻合，右圓塊仍量到x71，下條白色5／青色49；短鉤原版27位置有26屬候選紅色群，外框相同不代表RGB相同。只讀逐點定位：右緣12個與下條5個量測白色像素全部對應原版色號7的灰塊。v9只調灰塊亮度，入口以`ENEMY03-group1-v9-{generation,generated,frame,measure,measurements,review}-05-20261004`保存。候選仍未接受或選入，不以量測分類取代原版色區與視覺關係審查。

## 125. 葛雷戈林受遮擋前身份契約審查

第5張v9最後結果：五組量測外框與短鉤ROI均與原版吻合，下條白色0／青色47，原版0／63；短鉤紅色群26對原版27。原生1086×1448及72×96已檢視，灰塊已調暗、青色下條與短鉤保留，造型審查通過。原版參照及兩項負對照仍通過，外框不代表RGB或輪廓逐像素相同，完整提示與原生圖沿§124入口保存。

第5張v9静態審查見§124，正式候選呈現另沿本工作區`prepare.py`、`candidate-preparation.json`、`theme/`、`build-deps.jsonl`、`verify-normal`、`normal-attack/`、`verify-plane.py`、`display-review.json`及`source-manifest.json`保存。若正常HD可見與完整合成通過，再記`art-review-accepted.json`及`selection.json`，不可提前選入。進度同步沿`issue34-{before,prewrite,body,after,sync}`，最終來源核對與清理沿`final-audit.json`。

本批附加工具入口：`display-review.py`核對各姿勢實際PNG及可用格的省略／錯圖負對照；`accept-select.py`通過前述閘門後才建立`workplace/hd/theme-zellwal-pose5-v1-20261004/`，新主題若成立為29筆／28PNG。`preserve.py`及`source-snapshot/`保存精確來源，原版與state只記SHA。`tests.log`首跑缺唯讀/hd/bg.idx掛載，未執行正常測試；補掛同一既有基準後`tests-v2.log`通過既有Theme回歸與三組各16項生命週期，一项缺外部state測試略過，由本批真正正常载回另驗。保留失敗收據，不改測試或產品迎合環境。

契約審查的spec雜湊指向DRAFT，升READY後接受腳本首跑讀現行spec而被拒絕，尚未建立新主題。修正為核對已保存的精確contract-draft.md，另存當次READY全文至contract-ready.md，不改審查收據或期待值；原版、圖片、正式程式及驗證結果維持。

2026-10-04。接續§90／§123，024 §1.14.1先列DRAFT，只限ENEMY03 #3–#5的原版身體來源；8×8、位置、比例、動作、清單格式與正式遊戲資料不改。沿既有唯一有效前身份及完整384 bytes差分契約，核對原版正常3→4→5→4→3，不猜效果或ALLY未知身份。

工作區為`workplace/hd/zellwal-context-v1-20261004/`，入口`review-contract.py`、`contract-draft.md`與`contract-review.json`。原版事件與所有輸入SHA沿§90不可變收據；原始來源、state與frame留本機。審查必須涵蓋已知同位置原圖唯一性、四條有向邊及來源／生命週期負對照，達READY後才實作與驗正式HD可見。候選美術未接受，正式13/360與現行27筆／26PNG維持。

契約審查通過：15份同位置原圖唯一、四條有向差分無歧義、16項契約與負對照成立；重新核對五次原版來源的完整64000 bytes前後畫面不符0，來源單像素負對照有效。四次差分前圖均受遮擋但原版raw確實對應3→4→5→4→3，當次背景錨點相同。第六筆右側ALLY未知身份不推論，原版終點HP0及seed F95Bh保留。DRAFT全文保存contract-draft.md，024 §1.14.1據此達READY；正式實作、正常HD呈現、美術與完整動畫仍須各自驗證。

限定正式驗證通過：三組各16項生命週期及既有Theme回歸通過，一項缺外部state條件略過。正常同起點、seed F95Bh及原版按鍵的29筆候選，18份完整960×600圖面／中文／合成及18份真正載回、各100000步接續獨立不符0；HD两側原版RAM、frame、寄存器、steps及cycles一致，原版HP0如實保留。16份新敵人中文名與正式資料吻合。

display-review.json證實第3／4／5張各5／6／3份HD可見，其中來源完整返回3／2／1份；可用8×8格均與各自PNG吻合。第4／5張省略與錯圖負對照有效，sample06／09／12／15四份完整合成已檢視。第4張v8及第5張v9據此限定接受，accept-select.py建立新主題theme-zellwal-pose5-v1-20261004/，29筆／28PNG，正式15/360。舊27筆主題與全部收據保留。這批只證明正常攻擊分支的正式Theme與真正載回，不證明新29筆普通GUI／DAT、全部動畫、自然時序、效能、平台或封包。冷載受遮擋身份依契約清除／回退，不稱完整HD載回。其他345張敵人、30張ALLY與完整sprite範圍仍待完成。

文件與最後稽核入口沿同工作區final-documents.json／document-snapshot/與final-audit.json，不覆寫DRAFT／READY及前期來源收據。來源保存使用實際build-deps.jsonl及本批腳本、原生圖、量測、主題與正常圖面；原版與衍生state／frame僅記雜湊。

來源保全完成：437份精確本機副本、90份實際非標準Go依賴，273份原版／衍生資料只記SHA。#34寫前全文與時間一致，写後正文與body-file及worklist相同，2026-10-04T05:13:04Z維持OPEN；#44未修改。未commit／push／發行，原版與圖片未上傳。

## 126. 現行葛雷戈林29筆主題的普通前端驗證

進度同步入口：`progress-sync-20261004-1328-{before,prewrite,after,sync}.json`與`progress-sync-20261004-1328-body.md`沿本節工作區保存。2026-10-04依使用者要求核對CONTEXT與#34，僅同步已驗結果及目前準備狀態，不新增GUI／DAT通過聲明。

2026-10-04。工作區`workplace/hd/gui-zellwal-v1-20261004/`沿§121普通前端流程，以§125正常完整第3張sample03.state及正式中文側檔為不可變起點，兩側只有相同AdLib／cycles初始化，不注入人物、HP、座標或亂數。現行29筆／28PNG及15/360不變。

工具與證據入口：`prepare.py`、`prepare-frontend.go`、`fixture-inputs.json`、`frontend-preparation.json`、`run.py`及`captures/`；實際建置沿`build-deps-*.jsonl`、`build-inputs.json`、`psychicwar-current`、`export-state`與`machine-compare`。獨立原版重播沿`observe-replay.go`、`observe-replay`、`verify-replay.py`與`replay/`，各擷取state核對44欄位及DOS，來源逐事件重建；期望沿`art-model.py`、`verify.py`與`captures/verified.json`，不得縮小全屏或拿正式HD內部訊號作來源證據。

本批建置工具為`build.py`，實際Go依賴逐項保存至`build-deps-*.jsonl`，二進位與全部輸入雜湊記於`build-inputs.json`。原版來源模型準備沿`prepare-replay.py`及`verify-replay.py`；建置及執行紀錄沿`build.log`、`gui.log`、`export.log`、`replay.log`及`verify.log`，失敗收據不覆寫。

普通前端首跑在輸入雜湊核對時缺唯讀`/gomod`掛載，尚未啟動遊戲。`gui.log`與空輸出`captures-initial-mount-failure/`保留；補相同建置使用的唯讀模組掛載後，以同二進位、輸入及判準重跑，紀錄為`gui-v2.log`。

零步匯出首次命令帶未支援的`-orig`旗標，退出2且未產生資料，`export.log`保留；依工具實際`-out`入口與固定唯讀原版路徑重跑，紀錄為`export-v2.log`。普通前端本身已正常退出0，完成40份擷取；像素與來源尚待獨立核對。

首輪完整比對`captures/verified.json`為39/40，三姿勢及兩HD語言均有完整樣本。唯一`a-hd-chinese-01`差275像素，角色矩形不符0，實際為底部「已讀檔」。正式Update依F10再F11處理，原版步數695714916的啟動兩鍵同次更新，前次F10提示被F11覆寫；期望模型漏掉這條實際順序。`prepare-verification-v2.py`、`verify-v2.py`及`verify-v2.log`依正式原文、實際相同步數鍵序與兩秒提示期限建立限定合法期望，整幅與省略提示負對照保持。新收據為`captures/verified-v2.json`，不得覆寫首輪或排除提示列。

本批confirmed：現行29筆主題與本輪正式前端40/40完整960×600視窗不符0，每模式10份，中英文、HD開關及真正F11涵蓋ENEMY03 #3／#4／#5。20份HD中三姿勢各6／8／6份，兩語言各有三姿勢。敌人省略、錯姿勢及ALLY省略負對照有效，39份非空提示的省略負對照有效。九份實際完整PNG已檢視。原版按鍵紀錄0筆，前端正常退出0；正式main.go及主題、文本與字型未修改。

`replay/gui-machine-audit.json`的40份實際F10狀態與同F11起點無新原版按鍵重播，44個機器欄位及DOS全部相同。`replay/independent.json`與`replay-verify.log`的219次來源重建包含200次全圖向上捲動，不符0；五項來源負對照有效。起點695714912步，原版執行前seed F95Bh，固定已保存起點、不寫值或重擲，已證實有向差分身份與原版raw核對。ALLY起點完整，僅一次後續完整覆蓋，局部效果不猜新身份。

範圍限普通前端的保存起點接續與這40份視窗。保存起點未重印的英文訊息保留，較後樣本正常重印的敵人名稱為中文；不作全文中文、從開機、整場戰鬥、完整動畫、DAT、自然時序、效能、音訊、封包或真機聲明。第3張控制初始化與44欄位／DOS核對完成，起點原版RAM／畫面／寄存器未改。完整HD仍未完成，15/360及29筆／28PNG不變，下一步適用DAT及剩餘sprite。

正式前端不加觀察器，四種語言／HD模式真正F11後正常接續，F10固定擷取實際960×600視窗。每份state與PNG是否同幀由獨立原版資料、來源與完整像素比較判定，不由擷取步驟保證。原版、state、DAT與圖面留本機。來源保全與收尾沿`preserve.py`、`source-manifest.json`、`source-snapshot/`、`final-documents.json`、`document-snapshot/`及`final-audit.json`；進度同步沿`issue34-{before,prewrite,body,after,sync}`。本批未通過前不宣稱新29筆GUI／DAT已完成，不外推從開機、全部動畫、效能、音訊或封包。

## 127. 現行葛雷戈林29筆主題的正常 DAT 存讀

2026-10-04。沿§126同正式二進位與29筆主題，工作區仍為`workplace/hd/gui-zellwal-v1-20261004/`。正常迷宮與SELECT來源沿§102／§122，`prepare-dat.py`建立`dat-prepared-save/`、`dat-prepared-load/`與同正式零步初始化控制，不注入HP、人物或亂數。

工具入口：`build-dat.py`、`build-deps-dat-*.jsonl`及`dat-build-inputs.json`；`run-dat.py`沿`dat-save-actions.json`、`dat-load-actions.json`正常按鍵進Save Game與Load Game，存檔名HD15，輸出`dat-save-v1/`、`dat-load-v1/`。同實際保存前state及等價原版鍵序，由`run-dat-independent.py`與`pwstep-current`重生`dat-independent-v1/`、`dat-move-independent-v1/`，不載中文或HD。原版EXE／PBL／state／DAT唯讀或只在本批scratch寫入，全部留本機。

全屏期望沿§126本機29筆`art-model.py`及`verify-dat-screen.py`，匯出用§126實際`export-state`，輸出`dat-verified.json`與`verify-dat.py`；每份完整視窗需有相符原版狀態、正式文字字型及PNG期望，不排除提示或其他區域。每份十二相位至少一份完整吻合，不能稱十二份均過。DAT512 bytes與四模式52 bytes玩家資料及正常移動另需獨立對照和負對照。

建置／執行紀錄沿`dat-build.log`、`dat-save.log`、`dat-load.log`、`dat-export-save.log`、`dat-export-load.log`、`dat-independent.log`、`dat-verify-save-screen.log`、`dat-verify-load-screen.log`及`dat-verify.log`。來源保全沿`preserve-dat.py`、`dat-source-manifest.json`及`dat-source-snapshot/`；最後文件與核對沿`dat-final-documents.json`、`dat-document-snapshot/`及`dat-final-audit.json`。進度同步沿`dat-issue34-{before,prewrite,body,after,sync}`。未通過前不新增DAT完成聲明；完整15/360與29筆／28PNG、§126普通GUI結果維持，不外推從開機、戰鬥DAT、完整sprite、時序、音訊、封包或真機。

本批confirmed：15/15完整960×600視窗不符0，七份Save Game選單與八份Load Game／四模式／正常前進；每份十二相位至少一份吻合，不稱180張均通過。六份實際PNG已檢視，兩個普通前端退出0，原版按鍵紀錄符合指定鍵序，保存26個／載回16個按下放開事件。初始化兩側RAM／寄存器／畫面未改，44欄位及DOS控制相同。

HD15.DAT為512 bytes，SHA-256 3468b6ec00439ceaad19bb121ec2f57214f80516d12ba3c289c58fe076d96df1；與同一當次實際保存前state及等價原版鍵序、停用中文與HD後重生結果相同。四模式載回52 bytes等於保存前，正常前進確實改變玩家資料，與本次獨立原版Up相同；DAT／玩家資料一byte與完整圖面一像素負對照有效。原版初始狀態在執行前由不可變保存state固定，不寫值或重擲；不要求GUI自然時序或整段亂數序列相同。

SELECT起點未重印的英文保留，不作全中文證據；正常保存／載回選單中文字面沿正式JSON與字型獨立核對。範圍限正常迷宮保存檢查點，未從開機、戰鬥DAT、全部sprite、完整動畫、效能、音訊、真機或正式封包。正式main.go、主題、文本與字型未改，15/360與29筆／28PNG維持。原版、DAT與圖片留本機。移動獨立執行紀錄為dat-move-independent.log，§126已保存GUI來源副本可在雜湊相同時重用，不覆寫。

2026-10-04 14:45依使用者要求同步交接與#34，僅核對既有GUI／DAT收據，不新增遊戲驗收。同步收據沿同工作區`progress-sync-20261004-1445-{before,prewrite,after,sync}.json`及`progress-sync-20261004-1445-body.md`，文件核對沿`progress-sync-20261004-1445-check.json`。CONTEXT明確區分§125美術來源保存437份與最新DAT批次758份，不能相加為唯一檔案總數。正式15/360、29筆／28PNG、GUI40/40與DAT15/15的限定結果維持；完整HD及其他sprite未完成。

## 128. 剩餘盟友的正常來源探查

2026-10-04。接續§100，只確認ALLY #0不代表其他圖號未使用。ALLY.PBL共31張，#0–#15為24×32，#16–#30為16×16，不能直接將相鄰圖號当作同一角色的動作。正常Kasuruji遭遇來源沿§54的`next-body-kasuruji-v2-20261001.json`與相应event state，不修改人物、HP、座標或亂數。攻略所述投降／招募只作正常按鍵線索，未執行前仍未知。

工作區入口`workplace/hd/ally-recruit-source-v1-20261004/`，準備輸入與實際命令沿`inputs.json`、`inputs-postbattle.json`、`run.py`、`run-postbattle.py`及`execution*.json`保存；觀察器沿§100的精確研究工具與建置依賴，可在來源雜湊相同時重用。原版CS:IP 0161:8666的AH=0Ch載入與0161:8705原始DS:BX內容需獨立配對ALLY.PBL，完整機器欄位與DOS控制另核對。收據沿`source-*.json`、原始來源與state、`verification.json`、`source-manifest.json`保存。圖號、位置或用途未確認前不擴張正式024契約、不接受候選、不增加完成數。

ALLY #1原版24×32、PBL檔案偏移441，單格候選沿既有`workplace/hd/redraw/ALLY-01-v1-{source,generated,frame,prompt,provenance,measurements,review}-20261004`保存。原版單一參照、內建imagegen重繪，ImageMagick只作整幅尺寸正規化；原版來源逐像素、主要色區與姿勢另核對，沒有局部修圖。來源用途與正常HD呈現未知，候選不得列為正式完成。

首版原生及72×96已檢視。原版參照442368像素不符0，單像素負對照有效；整體threshold16外框[9,0,71,95]相同，但左靴外緣x10對原版x12，右腿內緣x42對原版x39。原版整體最左其實為左手x9，首版提示整體x12的描述有誤，量測以實際解碼為準。v2只由imagegen依原版修正腿部轮廓，沿相同v2命名保存，不用局部程式修圖、不提高接受數。

v2左靴ROI為[11,69,37,95]，原版[12,69,38,95]；右側ROI最左仍x42。回查獨立解碼，原版x39–41／y69–71對應來源(13,23)色號Bh的腰帶，來源(13,24)已為黑色，不能稱缺少大腿。v2提示將腰帶邊緣誤解成腿部，不接受為幾何修正完成。兩輪差異後重讀原版幾何對拍路由；v3只用imagegen修腰帶及左靴輪廓，沿相同v3命名保留全部收據。

【confirmed，限定正常原版鍵序】由237710721步首張Kasuruji返回狀態，237800000步FIFO掃描碼3Ch，終點250000000；由正常12-battle2戰勝狀態，281000000步同鍵，終點294000000。兩側執行前seed分別A71Dh／A48Ch，不重擲，IRQ1各2次。兩分支ALLY載入、開檔及唯一完整來源貼圖皆0；只證明這兩條鍵序，不證明全部F2、投降或招募失敗。`verify.py`以另一獨立比較工具核對44個機器欄位及DOS完全相同，CPU／RAM／埠負對照有效；重驗§100既有ALLY #0原始384 bytes唯一與768個終點像素正對照、單像素負對照及161份精確來源副本，64份實際非標準Go依賴沿原快照保留。`export.py`零步匯出兩份原版完整frame相同，兩份實際PNG均已檢視。未驗中文、HD或全部盟友。

v3原生及72×96已檢視，原版參照與負對照通過，整體／左手／右手／軀幹threshold16外框相同。頭部[26,0,66,35]對原版[27,0,65,35]，左靴ROI[10,69,37,95]對[12,69,38,95]，右側ROI[42,69,68,95]對[39,69,68,95]，腰帶缺口仍未修正。候選未接受、未選入，三版完整提示與原生圖、整幅正規化、`measure-art*.py`、`*-measurements-*.json`與`*-review-*.json`均留本機；沒有局部修圖。正式ALLY仍只#0，敵人15/360與29筆／28PNG不變。

失敗分類：工具副本因copyfile未保留執行權限而未啟動原版，修正該副本755後同二進位與鍵序重跑，`run.log`／`run-v2.log`及`environment-failure.json`保留。獨立比較工具首查漏工作目錄，第二查誤帶不支援-h；依實際Go位置參數入口重跑成功。Python首版錯把Go的[]uint8 JSON值當list，實際為Base64的PA==，保存`verify-initial-key-format.py`後解碼為同掃描碼3Ch，沒有修改原版或輸入期待。

本批保全與同步入口`preserve.py`、`source-manifest.json`、`source-snapshot/`、`issue34-{before,prewrite,body,after,sync}`、`final-audit.py`、`final-audit.json`與`document-snapshot/`。保全82份本批精確副本，重用161份已驗精確來源，13份原版／state／frame只記SHA；數量按路徑計算，不宣稱唯一內容總數。下一閘門仍為正常投降／招募的實際圖號與位置，以及ALLY #1腰帶／靴子造型修正；不將候選生成或無新來源結果當作正式sprite完成。完整HD未完成。

## 129. 招募與盟友圖號的最小呼叫切片

2026-10-04。接續§128正常F2沒有新ALLY的限定結果，先追實際載入來源，不繼續猜鍵序。工作區`workplace/ida/hd-ally-recruit-20261004/`；原版解壓EXE與正式.i64唯讀，既有IDA9.4 locked-v1僅對/tmp副本查詢，不改名或新增函式。最小探針沿§100 `tools/ida/ally_portrait.py`，收據`probe.json`、`probe.log`、`input-identity.json`；實際成功查詢為`query-v2.json`／`query-v3.json`及各自log，條件切片為`conditions.json`／`conditions.log`。原始定位、bytes、operand及推論等級保留，靜態候選不當作正常來源或READY契約。

入口以實際動態ALLY載入堆疊為準：原版CS:IP 0161:8666、返回2CEEh，開局上一層返回2CC1h、道具返回2D98h；IDA ea基準為seg002 base10510h，與原版runtime CS基準1610h分列。此批僅完成靜態查詢，尚無新的正常操作或來源貼圖收據。核對與來源保全沿`progress-sync.py`、`verification.json`、`source-manifest.json`及`source-snapshot/`；交接與遠端同步沿`issue34-*`及`final-audit.json`。確認來源前不接受ALLY #1、不改正式契約或完成數。

輸入SHA-256：解壓`PW_UNP.EXE`為`fd5115b91f014c47a1fb1a6e2ecfd645e293264fe614a4b350e92637cad2fbd9`，正式`PW_UNP.EXE.i64`為`4db51a199f5913c52dc4964557b9a4aafb965754d663cb0173c9dfdfe9a00c56`。工具IDA9.4／SDK940，查詢417個既有函式；image為`ida-pro-9.4-idapython:locked-v1`，ID與UID/GID見`input-identity.json`。函式數只描述此份DB，不推翻先前不同普查範圍的419。

| 原始定位 | 附加語意與證據等級 | 證據 |
|---|---|---|
| runtime `0161:2CE5`；IDA ea `131F5h`，bytes `b40c8b1ee8aee878598b1ee8aec3` | 強證據：AH=0Ch載入包裝；兩處呼叫runtime `2CBE`／`2D95`返回`2CC1`／`2D98`，吻合§100動態堆疊。這兩處IDA原分類為data，保留分類 | `query-v3.json`、§100原始動態收據 |
| runtime `0161:254A`；IDA ea `12A5Ah`，原名 `sub_12A5A` | 強證據：兩條ALLY路徑在呼叫後把BL移至AL；helper搬移30 bytes、從原始`cs:[305Bh]`取BX。圖號來源的實際操作數與新角色用途仍未知 | `query-v3.json`的caller原始bytes及helper指令 |
| runtime `0161:48ED`；IDA ea `14DFDh`，原名 `sub_14CE4`，bytes `e8e901` | 強證據：戰鬥分支在原始`cs:3AAA`減1為0時呼叫`4AD9`；不能由此推論HP門檻、招募成功或該byte的完整語意 | `query-v3.json`完整戰鬥函式 |
| runtime `0161:4AD9`；IDA ea `14FE9h`，原DB歸於 `sub_14FC0` | 強證據：先讀`cs:[bx]`，BX=3AB3h，再寫1；舊值非0即返回，舊值0才印投降句。未做正常動態驗證 | `conditions.json`對齊指令 |
| runtime `0161:4AF8`與字串`4AFB`；IDA ea `15008h`／`1500Bh`，call bytes `e89917` | 強證據：呼叫既有inline印字`sub_167A4`，其後為控制碼與原文`I..I Surrender!`。該段是資料，不能當CPU指令 | `query-v3.json`字串bytes與xref；`conditions.json` |

限制與失敗分類：首查錯將segment base10510h當作有效item起點，AssertionError保留於`query.log`與`query-initial-segment.py`。改從已知入口取segment，實際start1051Ch；near call相對位移計算加入start與base的差，v2／v3才成功。沒有重建正式資料庫或改原始bytes。

`surrender-span.json`的起點14FA0h落在指令中段，且`conditions.json`把inline字串保留為原DB的假指令解碼；兩者不可逐列當成控制流。結論只引用已對齊call／branch、原始bytes與已知印字契約。未知`cs:3AAA`寫入端、正常投降與招募後的ALLY圖號／位置留待下一窄切片；不能把既有兩條F2無新來源等同於所有招募失敗。

本輪依使用者要求更新CONTEXT與#34，僅同步已存在的查詢與收據。正式main、29筆／28PNG主題及15/360接受數維持，ALLY #1仍未接受。沒有新增遊戲測試、正常招募、HD接入或發行完成聲明。原版、DB、state與圖片留本機。

同步與收尾：#34回讀正文與body-file及worklist相同，2026-10-04T07:48:25Z維持OPEN。21份精確來源與收據副本保全，原版EXE與正式DB雜湊未改；最後文件副本沿`document-snapshot/`，核對沿`final-audit.json`。兩repo git diff --check與worklist render／verify通過，工作根無root-owned項目或.md目錄，本輪一次性容器已清理。#44未修改，未commit／push／發行。

後續最小寫入查詢沿同工作區`surrender-writers.py`、`surrender-writers.json`與`surrender-writers.log`，僅查已知code item與相關函式。直接operand與立即值不能涵蓋全部間接寫入；缺xref不作不存在寫入的證據。新正常鍵序研究探針與建置／執行／來源核對另沿`recruit-observer.go`、`recruit-*.json`、`recruit-*.log`及`source-manifest-v2.json`保存，未驗證前不新增招募或HD完成聲明。

短攻擊工具準備沿`recruit-prepare.py`，支援正常Space按住／放開；招募接續工具沿`recruit-release-prepare.py`與`recruit-observer-v2.go`，只增加正常放開鍵排程。獨立資料／機器／零步畫面核對入口為`recruit-verify.py`；工具與收據留本機，不進正式前端。

本批來源保存入口`recruit-preserve.py`及`source-manifest-v2.json`，沿實際兩份`recruit-build-deps*.jsonl`與Go overlay解析非標準建置來源；原版EXE／PBL、state、frame與原始圖面只記雜湊。快照沿既有`source-snapshot/`，既有檔案須雜湊相同才重用，不覆寫歷史。

2026-10-04 16:17新結果：

- 【confirmed，限定此正常Kasuruji分支】同一不可變首張返回state、seed A71Dh，Space從238000000步按住6000000步，終點258000000。敵人初始化HP70，終點HP47，未出現投降；曾在(264,152)用ALLY #0原始384 bytes作AL=1的XOR貼圖。獨立PBL唯一、整幅64000像素差0，來源省略差426、單像素負對照差1。短攻擊沒有達本例30的比較欄位，不能當作所有投降失败。
- 【confirmed，限定此正常分支】同state／seed，Space按住30000000步，終點280000000。原版`0161:46EF`／IDA ea`14BFFh`，bytes `2ea2aa3a`，在254517771步以AL=1寫`cs:3AAA`；前值4、HP29、原始word `cs:3A62`=30。其前對齊指令`or al,al`清carry，`sbb bx,dx`及`jnb`構成此比較。保留原名`sub_14B94`，其他byte不推測命名。這是實際執行證據，不能外推全部敵人的門檻。
- 投降入口在254770896步呼叫，255582894步返回原始`0161:48F0`，SP恢復2 bytes；提示後HP27、`cs:3AAA=1`及`cs:3AB3=1`。原版完整PNG已檢視，直接取此正常state接續，不寫HP、角色、位置或seed；長攻擊原終點HP0如實保留，不挑結果。
- 首次提示後state的seed自然為6A75h，先正常放開Space，再按住／放開F2，IRQ1三次；終點263582894，原版出現Talk／Duplicate。此鍵序沒有新ALLY載入、開檔或來源貼圖。
- 同實際選單state、seed F52Ch分別選Talk與Duplicate。Talk以Enter取得BIO Room提示，終點277582894；Duplicate以Down／Enter進Kai／Cancel選單，終點281582894。選Kai的Enter分支終點301582894，原版顯示`This ally can't use that ESP.`，沒有新ALLY來源。再Down／Enter取消，終點315582894，回到Talk／Duplicate父選單。不能把此選單分支當作已招募新盟友或ALLY #1用途證據。
- 七份終點皆由另一工具核對44個機器欄位與完整DOS相同，CPU／RAM／埠負對照有效；每份零步匯出完整frame相同。當次IRQ1依序2／2／3／2／4／2／4。投降後、F2、Talk、Duplicate、Kai拒絕與取消六份實際PNG已檢視。訊息字形另外由原版FONT.BIN及正式JSON原文核對2048像素與can't／can負對照，入口`recruit-message.py`與`recruit-message-verified.json`。

實際收據沿`recruit-{short-attack,full-attack,surrender-f2,talk-event,duplicate-event,risk-event,cancel-event}{,-verified,-independent-machine}.json`及其state／frame／PNG。首次Talk輸出前綴與其inputs／run.log碰撞，防覆寫拒絕時原版尚未啟動；改為talk-event後同二進位、state、鍵序與步數重跑，初次失敗與v2紀錄皆保留。沒有修改產品或判準迎合測試。

此窄切片已足以定位投降及正常F2分支，停止深挖與ALLY新圖號無直接證據的其餘戰鬥內部。下一步回到ALLY圖號操作數的真實來源或其他sprite；不繼續重擲此Duplicate分支。正式ALLY只#0、敵人15/360與29筆／28PNG維持，完整HD未完成。024 §1.1仍停在13/360的現況註記修正為已有§124–125接受證據的15/360，技術契約未擴張。

本輪交接與授權GitHub同步沿`recruit-progress.py`、`recruit-issue34-*`及`recruit-final-audit.json`，保留前一輪同步收據。

本批收尾：167份精確來源與收據副本，其中166份新增，65份實際非標準Go來源；50份原版／state／frame／PNG只記SHA。#34回讀正文與body-file及worklist相同，2026-10-04T08:22:49Z保持OPEN。兩repo git diff --check與worklist render／verify通過，工作根無root-owned項目或.md目錄，本輪容器皆--rm退出清理；UID/GID1000。原版資料、正式main與29筆主題未改，未commit／push／發行，#44未修改。

## 130. 首次Shulosu交談與其他ALLY的正常來源

2026-10-04。§129只證明Kasuruji投降的Talk／Duplicate分支，沒有新ALLY；手冊p.17的Team up線索與密碼表中的Shulosu仍待正常操作查證。本節從既有`08-encounter.state`的首次正常Shulosu遭遇按F2，不以Kasuruji分支推論所有盟友。

沿用§129精確保存的`recruit-observer-v2.bin`及`recruit-verify.py`，正式Go／原版資料不改、不重建既有二進位。工作區沿`workplace/ida/hd-ally-recruit-20261004/`，新增收據以`recruit-shulosu-*`命名，包含inputs／execution／run.log、event.json、原始來源及state／frame，獨立核對沿`*-verified.json`及`*-independent-machine.json`。圖號或位置未確認前不擴張024 READY，不接受美術候選或增加HD完成數。

本批confirmed，限定首次Shulosu遭遇的正常F2→Team up→命名接續：

- 起點`08-encounter.state`為86000000步；86500000步按F2、終點100000000，seed5447h、IRQ1=2。原版HP38未變，實際選單為Talk／Team up，沒有ALLY載入或貼圖。
- 從同次F2終點，100500000步起Down／Enter、間距1000000步、終點118000000，seed5C74h、IRQ1=4。原版要求命名，沒有ALLY載入或貼圖。
- 從同次Team up終點，118500000步起正常掃描碼`1F,23,16,26,18,1F,16,1C`、間距1000000步、終點140000000，seed5C74h、IRQ1=16。沒有Shift，實際名字為小寫shulosu，不將大寫計畫字串當作畫面證據。
- 132501642步在原版runtime`0161:8666`以AX=0C01h載入；132555249步在`0161:8705`以AL=1 XOR貼圖。原始DS:BX=1175h:9906h、DX=0304h、CX=3A26h；獨立PBL唯一圖號1，24×32，(232,152)。原始384 bytes SHA-256為`8c25534e872e9e435681b9d7948c9279ae5eee02d8bd53ee0c18d688a98489a6`。runtime位址與320×200座標分列，不當作IDA ea。
- `recruit-verify.py`獨立解碼ALLY.PBL與一般AL=1 XOR，前後完整64000像素不符0；省略來源差402、單像素負對照差1。三份終點44個機器欄位與完整DOS和同鍵序無觀察器控制相同，CPU／RAM／埠負對照有效；三份零步匯出完整frame相同，實際完整PNG均已檢視。命名終點回到迷宮，隊伍欄新增#1；不把AL=1一律當作出現或消失。

實際入口`recruit-shulosu-{f2,team,name}-inputs.json`、`*-execution.json`、`*-run.log`及`*-event.json`，獨立核對沿`*-event-verified.json`與`*-event-independent-machine.json`。新來源保存沿`shulosu-preserve.py`與`source-manifest-v3.json`，只增新副本並核對既有來源，不覆寫§129快照；進度同步沿`shulosu-progress.py`及`shulosu-issue34-*`，最後核對沿`shulosu-final-audit.json`與既有`document-snapshot/`的新shulosu前綴。

此原版來源已足以證實ALLY #1的本正常加入位置。其他位置、生命週期、中文／HD玩家呈現與正式契約仍待驗。§128三版候選的幾何偏差維持，不接受、不選入；024 READY未擴張、正式ALLY仍只#0。敵人15/360、29筆／28PNG及§126–127GUI／DAT限定結果維持，完整HD未完成。原版、state與圖片留本機，未commit／push／發行，#44未修改。

本批收尾：新增35份精確本機副本，167份前批副本及65份實際Go來源重驗；22份原版／state／frame／PNG只記SHA。#34回讀與body-file及worklist相同，2026-10-04T08:44:25Z維持OPEN。worklist render／verify與兩repo git diff --check通過，工作根無root-owned項目或.md目錄；一次性容器已退出移除，輸出UID/GID1000。正式main及29筆主題雜湊相同，未commit／push／發行，#44未修改。

## 131. ALLY #1正常來源的HD契約與造型修正

2026-10-04。接續§130的正常F2→Team up→命名來源，ALLY #1原版24×32與(232,152)已確認。沿已接受的原版排版定案，不改人物位置或資料；先補正式來源／生命週期契約與候選幾何，READY前不實作。

來源工作區沿`workplace/ida/hd-ally-recruit-20261004/`，新增入口以`ally1-*`命名；美術沿既有`workplace/hd/redraw/ALLY-01-v4-{prompt,generated,frame,provenance,measurements,review}-20261004`，量測工具以來源工作區`ally1-measure-art-v4.py`保存。第一張為§128 v3編輯目標，第二張為唯一原版完整24×32幾何參照。內建imagegen修頭部、左靴與青色腰帶輪廓，ImageMagick只作整幅尺寸正規化，不做局部程式修圖。候選生成前仍未接受／選入，正式ALLY只#0、敵人15/360、29筆／28PNG維持。

技術審查入口`ally1-contract-evidence-v2.py/.json`與`ally1-review-contract.py`，精確DRAFT／READY副本為`ally1-contract-{draft,ready}.md`、審查收據`ally1-contract-review.json`。獨立PBL #1唯一、原版貼圖前全黑與後完整、右側錨點三份相同及正常機器／DOS對照已閉合。024 §1.16升READY，只支援(232,152)與明示右側錨點；完整原圖恢復、未知覆蓋清身份及讀檔重建沿既有READY生命週期，不建立新ALLY XOR差分語意。美術及正常HD呈現仍待驗，正式接受數不變。

v4／v5候選均由內建imagegen及整幅72×96正規化，原版442368參照像素差0與負對照通過。v5頭部低亮度外框[27,0,66,35]對原版[27,0,65,35]，左靴[12,69,37,95]對[12,69,38,95]，右側含腰帶[41,69,68,95]對[39,69,68,95]；未接受、未選入。整體／兩手／軀幹外框仍相同，不以較亮門檻的吻合取代低亮度偏差。

量測工具首查錯將strict_pbl的tuple當作dict，後續獨立腳本誤帶crop的frame-width參數；兩次均在資料讀取階段失敗，未改原版或產生接受收據。讀實際函式契約後用v2修正，SCREEN五張320×40串接成原版320×200基準，原始腳本保留，不用補零或錯誤裁切通過。

§1.16 READY後已實作：`apps/psychicwar/theme/ally.go`新增限定圖號載入，`theme.go`的來源與圖面共用#1的(232,152)及明示右側錨點，既有#0兩位置保留。新增`ally1_test.go`使用真實正常frame與固定384-byte SHA，37項Theme測試通過、0失敗；4項其他角色外部state測試略過，名稱沿`ally1-tests-v1.jsonl`，不宣稱略過項本輪通過。

實際執行入口`ally1-runtime.go`、`ally1-runtime.bin`，建置沿`ally1-build-deps.jsonl`及`ally1-build.log`，執行沿`ally1-run-inputs-v1.json`、`ally1-execution-v1.json`與`ally1-run-v1.log`。本工作區manifest.json只含未接受v5候選，屬技術原型，未取代現行29筆主題。從§130同實際Team up命名state、seed5C74h，正常按下／放開shulosu+Enter，各按住500000步，至140000000。與FIFO來源探針的鍵事件API分開記錄，不宣稱逐指令輸入完全相同；兩側使用同正常鍵序，沒有改HP、位置、人物、seed或資料。

獨立圖面入口`ally1-verify-runtime-v2.py`與`ally1-runtime-independent-v2.json`。12份完整960×600不符0，涵蓋兩側start／entry／return／end／reload／continue；原版貼圖前全黑、後#1完整。四份可見HD省略負對照各4027像素、換#0圖各4385，單像素差1；完整HD終點及v5縮圖已檢視。真正保存／載回後兩側各接續100000步，兩組44欄位及完整DOS相同，CPU／RAM／埠負對照有效，沿`ally1-machine-{end,continue}.json`。此為技術接入及正常state接續，未驗中文、普通GUI、原版DAT、其他位置／生命週期、造型或封包；完整HD未完成，024 §1.16維持READY。

初次獨立工具缺Pillow，在讀取資料前失敗；不另建image、不裝未鎖版library。保留`ally1-verify-runtime.py`，v2改用既有ImageMagick只讀RGBA，再依原版PBL獨立構造完整期望，兩張資產的alpha皆255。沒有修圖片、改驗收區域或重跑挑選原版結果。spec後續只回填§1.2及來源欄位表的§1.16連結，執行時完整原文另存`ally1-runtime-input-spec.md`，SHA與原執行收據相同，不覆寫原收據的規格版本。

本批精確來源／實際Go依賴保存入口`ally1-preserve.py`、`source-manifest-v4.json`及`ally1-source-snapshot/`；進度沿`ally1-progress.py`、`ally1-issue34-*`，最後文件與核對沿既有`document-snapshot/`的ally1前綴及`ally1-final-audit.json`。新程序與原型留本機，現行29筆／28PNG及15/360未變；#1候選未接受，#44未修改，未commit／push／發行。

本批收尾：156份精確本機副本、88份實際非標準runtime Go來源、53份原版／state／frame／PNG只記SHA；前批167及35份快照重驗，數量不相加為唯一總數。來源保存首跑假定dosgolem有go.sum，但該module實際無此檔；未影響遊戲或測試。實際保存入口ally1-preserve-v2.py，保留首跑71份部分副本，重用前逐項核對SHA，不覆寫歷史。#34回讀與body-file及worklist相同，2026-10-04T09:24:12Z保持OPEN。兩repo git diff --check與worklist render／verify通過；工作根無root-owned項目或.md目錄，本輪容器均退出移除，UID/GID1000。現行29筆主題與正式main未改，ALLY來源程式新增#1；敵人15/360與ALLY接受只#0不變，完整HD未完成。未commit／push／發行，#44未修改。

## 132. ALLY #1剩餘輪廓的原版來源核對

2026-10-04。接續§131已完成的限定技術接入與未接受v5。先查原始ALLY #1像素，不用ROI外框代替部位來源：原版row23的x11–14均為Bh青色，形成跨中央腿縫的連續腰帶；x12在row24及29–31為3h深青色，左ROI最右因此同時涵蓋腰帶與靴內側。原版頭部最右x21在row5–8是耳側裝置，不是臉部；v5在顯示x66／y15–19的亮度16以上殘邊對應該裝置。

工作區沿`workplace/ida/hd-ally-recruit-20261004/`。本節最初只建立`workplace/hd/redraw/ALLY-01-v6-prompt-20261004.txt`；後續已完成v6–v8生成及量測，結果如下。計畫以內建imagegen閉合原版青色腰帶橋，再以ImageMagick整幅72×96正規化；後續工具預定`ally1-art-normalize-v6.py`與`ally1-art-measure-v6.py`，後續均已建立。候選沿既有`workplace/hd/redraw/ALLY-01-v6-{generated,frame,provenance,measurements,review}-20261004`。未知或失配仍不接受、不選入，正式ALLY接受只#0、敵人15/360及29筆／28PNG維持。

本次進度同步入口沿同研究工作區`progress-only-*`，包含GitHub #34的before／prewrite／body／after／sync及本機evidence收據。只更新文件與議題，不新增HD完成數或改動程式。§131的37項通過、4項略過及12份完整圖面差0仍限其原技術驗證範圍。

後續美術批次沿`ALLY-01-v6-*`與`ALLY-01-v7-*`保存prompt／call／generated／source／frame／provenance／measurements／review；工具沿`ally1-art-normalize-v{6,7}.py`與`ally1-art-measure-v{6,7}.py`。v6已內建生成並整幅正規化，原版參照442368像素差0、單像素負對照差1；缺口填平後腰帶下緣抬高，左右腿區域外框仍失配，未接受／未選入。v7只調整腰帶下緣，生成與驗收結果另追加，不能由已寫提示當作完成。

v7原生及72×96已檢視；左右腿ROI外框吻合，但腰帶x36–42在y71仍近黑，右端x43–44仍為白色；左靴獨立區域[33,87,38,95]的外框只到[33,87,36,94]，原版54個非黑像素對候選27。未接受。v8沿`ALLY-01-v8-*`與`ally1-art-{normalize,measure}-v8.py`，只再延伸腰帶下緣；同步及保全收據沿同研究工作區`ally1-art-*`，原版、PNG及state保持本機。

2026-10-04 17:57：v8已內建生成、整幅72×96正規化與檢視；七個區域中六個外框吻合，頭部右緣仍多1像素。腰帶y69／70已為青色，但y71的x36–42仍近黑，x44仍為白色；左靴獨立區域仍27對原版54像素，外框到x36／y94，原版到x38／y95。因此v6–v8均未接受／未選入；沒有局部程式修圖，正式敵人15/360、ALLY接受只#0、現行29筆／28PNG保持。下一步仍需修原版腰帶下緣與靴內側，再處理耳側裝置，不能由外框吻合宣稱部位完成。來源清單、精確工具快照及最後核對沿ally1-art-source-manifest-v1.json／ally1-art-source-tools-v1.json／ally1-art-final-audit-v1.json。

後續v9–v12候選沿`ALLY-01-v{9,10,11,12}-*`，同樣保存prompt／call／generated／source／frame／provenance／measurements／review；工具沿`ally1-art-{normalize,measure}-v{9,10,11,12}.py`。本批保全、同步及最後核對沿同工作區`ally1-art2-*`。v9已生成但腰帶y71仍近黑，未接受。連續局部修正失配後重讀imagegen提示reference與024 §6／§9，不改門檻或原版參照。v10改由原版單一幾何參照作完整平滑重繪，v9只供畫風；以新候選重驗，未執行或未驗的候選不列成果。

v11只提供原版圖片，腰帶36份診斷均為青色、左靴內側54像素與原版區域外框吻合；整體、頭部、左臂與兩靴外緣仍偏移，未接受。原版row29–31核對不含9h藍色；row31左靴x5–10及右靴x16–22為Fh白色，另有灰色及青色側條。v11的兩塊藍色鞋尖不符此來源。v12只把鞋尖改回白色，保留上方藍色護膝與靴帶、青色側條及原版位置；結果另記。

接續v13–v16沿`ALLY-01-v{13,14,15,16}-*`保存prompt／call／generated／source／frame／provenance／measurements／review；工具沿`ally1-art-{normalize,measure}-v{13,14,15,16}.py`，保全／同步／最後核對沿同工作區`ally1-art3-*`。先修靴子外緣，保留v12已修的腰帶、靴內側與白色鞋尖；頭部及左臂分開檢查。尚未生成的版本不列成果。

v13右靴外框恢復；v14頭部外框恢復，剩餘偏移為左指x8／y61–64的4個像素與左鞋尖x11／y95的灰色像素RGB[25,24,24]。v15左手指外緣修正後，七個區域中六個外框吻合，只剩左鞋尖外緣；v16灰色鞋尖側緣修正過頭，最左x13對原版x12，仍未接受。v17只回補該灰色側緣，沿`ALLY-01-v17-*`與`ally1-art-{normalize,measure}-v17.py`；本批保全及同步同`ally1-art3-*`。既有亮度16／32／60診斷保持，不提高門檻宣稱通過。

v17靜態審查：七區亮度16外框均吻合，腰帶36個診斷點持續青色，左靴內側54像素及外框吻合，鞋尖固定區域藍色0／0；原生及72×96已檢視。鞋尖中性色數40／49對原版54／63，整體非黑數3697對原版3618，保留色彩及輪廓近似限制，不稱逐像素RGB相同。下一閘門為該圖正常呈現與正式選入。

本地正常呈現原型沿同工作區`ally1-v17-prototype/`的manifest及ALLY-01.png；首個執行入口`ally1-v17-runtime.go`／`.bin`因輸出前綴與工具本身同名，在載入遊戲前被防覆寫閘門擋下，原失敗紀錄保留。修正入口`ally1-v17-runtime-v2.go`／`.bin`，建置及執行收據沿`ally1-v17-*`，完整圖面及state沿`ally1-v17-frames-v1-*`，獨立比對入口`ally1-v17-verify-runtime-v2.py`及`ally1-v17-independent.json`，機器對照沿`ally1-v17-machine-*`。規格沿024 §1.16 READY，原型不取代現行29筆主題；未驗中文、普通GUI、DAT或完整HD。

2026-10-04 18:14：v9–v12原生及72×96均已檢視。v9／v10沒有解掉腰帶；v11移除HD畫風圖片、只提供原版後，腰帶與左靴內側區域恢復。v12將原版最下列鞋尖白色回填，兩鞋尖固定區域藍色數從19／14降為0／0，輕色中性色數40／52對原版54／63，不能稱RGB或顏色像素數相同。左靴內側54像素及外框保持；頭部／左臂左緣各超出1像素，左靴左緣超出2像素、右靴右緣超出1像素。完整造型仍未接受／選入；原版參照各442368像素差0、單像素負對照有效，沒有局部程式修圖。正式敵人15/360、ALLY接受只#0、現行29筆／28PNG維持。本批來源清單及精確工具快照沿ally1-art2-source-manifest-v1.json／ally1-art2-source-tools-v1.json，最後核對沿ally1-art2-final-audit-v1.json。

2026-10-04 18:44：v17在相同正常Team up命名state及seed5C74h、shulosu+Enter鍵序實跑。12份完整960×600獨立圖面差0，涵蓋貼圖入口／返回、終點、真正state載回及接續100000步；四份可見HD省略／錯圖負對照各4155／4582像素，單像素差1。兩組44欄位與完整DOS相同，CPU／RAM／埠負對照有效。實際完整HD終點已檢視。初次工具在遊戲載入前因檔名前綴碰撞退出，修正後用同工具鏈與條件重跑，不覆寫失敗來源／binary／log。v17靜態與本正常呈現通過，完整主題選入、中文、普通GUI、原版DAT及其他位置仍待驗；現行29筆／28PNG、敵人15/360與ALLY正式選入只#0維持。實際來源及工具保全沿ally1-art3-source-manifest-v1.json／ally1-art3-source-tools-v1.json；最後核對沿ally1-art3-final-audit-v1.json。

## 133. ALLY #1 v17完整主題合成與中文驗證

2026-10-04。沿024 §1.16 READY，將§132已通過靜態及正常呈現的v17放入完整候選主題，保留原29筆／28PNG，新增ALLY #1原版(232,152)與明示右側錨點。候選入口預定`workplace/hd/theme-ally1-candidate-v1-20261004/`，不代表已選入或完整HD完成。

工具與收據沿既有`workplace/ida/hd-ally-recruit-20261004/`的`ally1-full-*`：`prepare.py`建立候選及新的`runtime-v1.go/.bin`；`frames-v1-*`保存同正常命名、真正state載回與接續的原版／圖面／中文快照；`verify-v1.py`從原版PBL、PNG、正式字型獨立重建完整期望，`independent-v1.json`保存結果。建置、執行、來源保存與進度沿同前綴，不覆寫§132候選與收據。

本批預定驗證完整30筆圖面與中英文合成、#0／#1共存及HD切換；普通GUI、原版DAT及其他位置尚待後續驗證。未執行或未驗的項目不列完成。

正式選入工具沿同工作區`ally1-full-select-v1.py`，只有靜態、限定正常呈現及完整主題合成收據全部通過才建立`workplace/hd/theme-ally1-v1-20261004/`，逐張核對來源SHA，另存`selection.json`。此選入只接受#1已證實位置；普通GUI、DAT與其他ALLY仍各自驗證，不外推前批29筆主題的收據。

普通前端入口沿同工作區`ally1-full-gui-{build,run,model,verify}-v1.py`及`ally1-full-frontend-v1.bin`、`ally1-full-export-v1.bin`，輸出`ally1-full-gui-v1/`。Xvfb只在有界容器中執行，正常F10／F11與語言／HD切換後各擷取十二相位，以真正F10原版state零步匯出、正式中文快照、PBL與PNG獨立核對整幅視窗；只有實際通過的相位算證據。名稱提示起點未重新印出的英文保留，不作全中文或從開機驗收。

首輪F11載回研究用指令數時鐘state後，正式前端以`RunCycles需要先SetDOSBoxCycles`退出1，未完成任何視窗抽測。首輪程式、binary、state及log保留。修正入口`ally1-full-gui-setup-v2.py`沿§126的載入後初始化契約，使用本輪實際main預設AdLib=false、cycles750，零步兩側轉接並檢查RAM／寄存器／frame不變、44欄位及DOS相同；不修改正式前端。新來源`ally1-full-gui-prepare-v2.go/.bin`、`ally1-full-gui-{run,verify}-v2.py`與輸出`ally1-full-gui-v2/`分開保存，未重跑或未核對的項目不列通過。

v2普通前端退出0、六份完整視窗像素均相符，獨立工具最後把record的`events:null`當作非空事件而失敗。實際record有明示events欄位且為null，原版事件為零筆。保留v2工具及log，`ally1-full-gui-verify-v3.py`依實際格式接受null或空陣列，新增假事件必須拒絕的負對照；沿同一批視窗與state重核，不重跑挑選結果，收據`ally1-full-gui-v2/independent-v3.json`。

v3仍在F11檢查未通過：保存與載回後相隔2456333步的indexed SHA不同，完整RGB與52 bytes玩家資料相同。不能直接拿不同指令點當同狀態。沿既有固定state對拍契約新增`ally1-full-gui-replay-v1.go/.bin`，從實際初始state及真正F11的d保存state，無新原版按鍵重播到各實際F10的絕對指令數，逐份比完整indexed、RGB、RAM、寄存器、cycles及44欄位／DOS；獨立收據`ally1-full-gui-replay-v1.json`。不排除游標或放寬全畫面比對，原v3與log保留；後續驗證入口`ally1-full-gui-verify-v4.py`。

2026-10-04 19:18：完整30筆候選24份中英文合成及12份圖面差0，真正state載回及接續100000步；四組44欄位及DOS相同，HD兩側與前批原版終點均核對。#0／#1共存、HD切換、省略4155／錯圖4582、錯位／錯錨點／單像素及CPU／RAM／埠負對照有效。中文限正式六個SCREEN標籤，命名提示未重印的英文保留。v17已限定接受並選入現行30筆／29PNG主題；原29筆／28PNG SHA相同。完整中英文終點已檢視。

新30筆普通前端6/6完整960×600視窗相符，四模式及真正F11，#0／#1共存。每份十二相位至少一份吻合，不外推72張全過；省略#1與中文負對照有效，前端退出0、原版按鍵紀錄0筆，F11依實際保存state重播到同指令數，完整indexed／RGB、RAM、寄存器、cycles及44欄位／DOS相同。精確程式、模型與依賴沿ally1-full-gui-build-inputs-v1.json／setup-inputs-v2.json，首輪指令時鐘不相容失敗保留，factory初始化控制44欄位／DOS相同。新30筆原版DAT及其他位置仍待驗，§126–127只屬前版29筆；完整HD未完成，024維持READY。

GUI最終入口為ally1-full-gui-v2/independent-v4.json；v3未產出通過收據。六份同絕對指令數原版重播的44欄位及DOS、完整indexed／RGB／RAM／寄存器／cycles相同。v2的null事件與v3不同指令數indexed直接比較失敗均為驗證工具問題，來源及log保留，不修改產品或排除像素。最終來源與私有資料SHA沿ally1-full-source-tools-final-v1.json／ally1-full-private-hashes-v1.json；遠端同步及收尾沿ally1-full-issue34-sync-v1.json與ally1-full-final-audit-v1.json。

最後核對發現CONTEXT的遠端同步段仍留前批29筆與只#0，已依真正#34回讀修正；原v1文件快照保留，新的CONTEXT及本研究檔快照沿document-snapshot/ally1-full-v2-*，補充收據ally1-full-final-documents-v2.json。此為文件修正，不新增驗證完成數。

## 134. 新30筆主題的招募盟友DAT存讀

2026-10-04。沿§133現行30筆／29PNG及024 §1.16 READY，從實際ALLY #1命名終點用正常按鍵返回迷宮，普通前端Save Game保存，再由正常SELECT的Load Game載回。核對512 bytes DAT、原始玩家資料、#0／#1來源及完整中英文／HD畫面，不改原版資料或玩法。

工作區沿既有`workplace/ida/hd-ally-recruit-20261004/`，新增工具／收據沿`ally1-dat-*`；`ally1-dat-build-v1.py`重建當前pwstep並核對沿用的正式前端來源。原版返回迷宮的探針沿`ally1-dat-route-*`，普通前端輸出沿`ally1-dat-save-v1/`與`ally1-dat-load-v1/`，獨立原版保存／載回／移動沿`ally1-dat-{save,load,move}-independent-v1/`。建立前先檢查UID/GID及防覆寫，原版素材唯讀，DAT／state／PNG留本機。未取得實際收據前，不列存讀完成。

2026-10-04 19:49：原版DAT與正常GUI限定驗證通過。

新30筆主題正常DAT存讀15/15完整960×600視窗抽測通過，包含七份保存選單、八份載回／四模式／正常移動。
512 bytes的hd30.dat與同保存前state及等價原版鍵序獨立重生結果相同；原始檔案偏移166保有ASCII shulosu。
四模式載回52 bytes玩家資料與保存前相同，原版完整#0／#1來源均恢復，正常前進資料與獨立原版相同。
省略#1差4155像素，DAT名字一byte、玩家資料一byte與完整視窗一像素負對照有效。
兩次普通前端退出0，原版按鍵紀錄26／16筆符合指定鍵序，HD與語言熱鍵未送入原版。
每份十二相位至少一份完整吻合，不稱180張均通過；SELECT起點未重印的英文保留。
限定已招募盟友的迷宮保存與SELECT起點接續，未涵蓋從開機、其他ALLY位置、戰鬥DAT、完整動畫或封包。
完成收據ally1-dat-verified-v1.json；整幅視窗分項沿ally1-dat-{save,load}-v1/verified.json。

DAT SHA-256：23ebe0a003baf865c1a907746ebbddc1ba4e70e65e44f06a983b7e5496800ac7。檔案偏移為原始DAT的位元組基準；名字欄位的結構語意不外推。confirmed僅限實際原版存讀及所列資料／圖面。Go1.24.13、Python3.11.2與既有psychicwar-go-ebiten工具鏈，pwstep重建來源沿ally1-dat-build-inputs-v1.json，普通前端沿§133保存binary。等價原版鍵序未宣稱相同GUI牆上時間或完整骰序。

本輪進度、來源文字保全與遠端同步收據沿ally1-dat-{progress,source-text,issue34-sync,final-audit}-v1.json，輸入圖片、原版及DAT只留本機。

## 135. 招募後ALLY #1的道具肖像來源

2026-10-04。從§134實際招募與保存後的原版state，沿正常Esc→看道具及隊伍切換查原版來源。工作區沿`workplace/ida/hd-ally-recruit-20261004/`，本輪新工具、state、截圖及收據統一用`ally1-items-*`前綴。先確認原版PBL、貼圖位置、完整畫面及唯讀觀察控制，再審查024限定契約；未READY前不改正式接入。

本節追蹤正常鍵序與原版來源，不把既有ALLY #0位置直接當作#1證據。不修改原版RAM、玩家資料、亂數或DAT。既有美術沿v17，不新增造型、排版或manifest schema；其他ALLY位置、動畫、效果與交付仍依完整目標完成。

原版證據完成入口為`ally1-items-evidence-v3.json`。v1空Go byte切片為JSON空base64字串，v2誤認貼圖前全黑；兩個工具來源及失敗保留。實際貼圖前及道具列表肖像區為768個色號1，v3依真實資料核對，未重跑原版挑選結果。探索性SCREEN裁切曾把320×40當320×200，屬工具尺寸問題，右側錨點改以原PBL逐列獨立取樣核對。

024 §1.17的DRAFT／READY精確文字及證據審查沿`ally1-items-contract-review-v1.json`。後續新增位置主題沿既有`workplace/hd/theme-ally1-items-v1-20261004/`，31筆／29PNG，只有同一ALLY #1美術的新位置；原30筆素材保持。未通過實際顯示前不取代現行主題。建置／正常接續／前端收據沿`ally1-items-*`；普通前端、新主題DAT及交付僅依實際通過範圍宣稱。

2026-10-04 20:23：限定道具位置已接入並選入現行31筆／29PNG主題。

ALLY #1的道具肖像已依原版來源接入並選入。現行主題為
workplace/hd/theme-ally1-items-v1-20261004/，31筆／29PNG。只有同一v17美術的新位置，
原30筆及29張PNG逐張SHA相同；ALLY正式仍是#0／#1，共2/31，其他29張未完成，敵人15/360維持。

正常Esc→看道具→Enter確認ALLY.PBL #1在(128,8)，完整24×32、AL0、384 bytes唯一來源，
原版貼圖前為色號1，覆寫後64000像素不符0；右側錨點及下方#0／#1保留。
再按Enter至道具列表，原版及HD上方肖像均清除。Escape在此提示被原版忽略，不當作離開選單。
來源觀察44欄位及DOS控制相同，執行前固定seed5C74h，未改RAM、玩家資料、亂數或DAT。

024 §1.17經DRAFT與證據審查達READY。新位置沿既有AL0來源與8×8比對，
同位置#0／#1互斥、兩位置身份分開，沒有改排版、比例、原版資料、schema或貼圖生命週期。
Theme回歸38項通過，4項缺外部state條件略過；首輪測試誤把AL0完整after畫面當進行前畫面，修正為實際before後通過，失敗來源與log保留。

新31筆主題正常道具路徑16份完整960×600兩語合成差0，包含四模式、真正state載回及1ms接續；
十二組44欄位及DOS原版對照相同。普通前端7/7完整視窗通過，四模式、上方#1與下方兩名盟友、道具列表清除及真正F11均涵蓋。
每份十二相位至少一份相符，不稱84張均通過；省略上方#1差4155像素，單像素負對照有效。
七份實際F10 state依原版四筆Enter邊緣的絕對指令數重播，完整indexed／RGB、RAM、寄存器、cycles及44欄位／DOS相同。
前端退出0，原版只收到兩次Enter，HD與語言熱鍵未送入原版。實際HD中文道具頁與GUI已檢視。

來源入口與正常獨立收據沿本節ally1-items-*。新31筆／重建binary的DAT、其他ALLY來源、完整動畫與交付待驗；
§133–134的30筆GUI6/6與DAT15/15留在原保存binary及主題範圍，不自動延伸為31筆通過。

AL0首輪技術測試以完整after.frame要求進行前零輸出，與既有§1.2來源／格契約不同；v2改驗實際before.frame，正式生命週期不改。新位置載入器只加一項已READY的位置，其他拒絕及重複來源限制維持。Go重播首啟動因重複唯讀掛載在容器建立前被拒，移除重複項後同image／命令乾淨重跑通過，非產品失敗。選入收據ally1-items-selection-v1.json；同步與收尾入口ally1-items-{issue34-sync,final-audit}-v1.json。

最終來源保全沿ally1-items-source-snapshot-v1.json／v2.json、ally1-items-gui-replay-build-v1.json。正式前端347份與重播77份實際非標準Go來源各自保存精確文字，不作唯一檔案合計。編譯時完整規格文字已凍結，後續僅補新節索引，§1.17 READY正文相同。新31筆DAT及完整HD仍未完成。

## 136. 31筆主題的原版DAT存讀

2026-10-04。沿§135現行31筆／29PNG與實際重建binary，查正常道具頁退出後的Save Game、SELECT→Load Game與載回道具肖像。工具、原版控制、截圖與收據沿既有`workplace/ida/hd-ally-recruit-20261004/ally1-items-dat-*`；原版只讀、正常按鍵與存檔寫入獨立scratch，不改RAM、亂數或DAT格式。驗證聚焦新增位置及保存的盟友資料，不重開與本輪改動無關的既有矩陣。

## 137. ALLY #2的原版構圖與HD素材

2026-10-04。先核對原始ALLY.PBL #2、既有reference及候選，準備與已接受#0／#1風格一致的新素材。原版位置、比例、姿勢、色區及黑色缺口維持。工作區沿`workplace/hd/redraw/ALLY-02-v1-*-20261004`，精確提示、生成、原生輸出、整幅正規化與量測各自保存；本輪僅用內建imagegen，不做局部程式修圖。靜態候選不代表正常來源、正式接入或完整HD已完成。

2026-10-04 20:51：進度核對。§136目前僅完成道具頁按鍵探索，尚無新31筆Save Game／Load Game驗證收據。
Enter／Space切換角色與道具列表，Escape與小寫n的既有嘗試未退出提示，退出鍵未知。
探索收據沿ally1-items-dat-{exit,close,space}-v1/execution.json，執行退出0只表示探針正常結束，不表示玩家已離開選單或完成DAT。

§137原版參照RGB差異0；內建imagegen第一稿的原生1086×1448已整幅正規化為72×96，accepted=false。
證據為workplace/hd/redraw/ALLY-02-v1-{reference,generation}-20261004.json及同前綴prompt、generated、frame。
量測與視覺審查、正常來源、READY契約及正式接入尚未完成；ALLY正式仍2/31，敵人正式仍15/360。
本次文件與GitHub進度同步收據沿既有研究工作區progress-20261004-issue34-{before,body,after,sync}；圖片及原版資料不公開。

2026-10-04 21:27：§136新31筆DAT限定驗證完成。

### 最新：現行31筆主題的原版 DAT 存讀

- 正常已招募迷宮 Save Game → SELECT Load Game → 看道具，9/9 份完整 960×600 視窗相符，包含保存三份及載回六份；新增 ALLY #1 道具肖像涵蓋中英／HD 四模式。
- hd31.dat 共 512 位元組，與同實際保存前狀態及等價原版鍵序獨立重生結果相同。檔案偏移166保有 ASCII shulosu；載回52位元組玩家資料與保存前相同。
- 載回後原版完整 ALLY #0／#1 下方來源及 #1 上方道具來源均恢復。四模式道具資料與獨立原版相同；省略上方 #1 差4155像素，DAT名字、玩家資料及全圖負對照有效。
- 9份實際F10狀態依26／22筆原版鍵邊緣重播到同指令數，完整indexed／RGB、RAM、寄存器、cycles及44欄位／DOS相同；CPU／RAM／埠負對照有效。兩個前端正常退出0，HD／語言熱鍵未送入原版。
- 每份十二相位至少一份完整吻合，不稱108張均通過。限定正常保存起點接續，未驗從開機、道具頁退出鍵、戰鬥DAT、完整動畫、效能或HD封包。
- 首跑缺Go依賴掛載及載回截圖期限不足，已保留失敗並用同工具鏈重跑。原版重播曾因整體RAM觀察改變EGA Latch，以及複製DAT未保留修改時間而失敗；改用保存狀態完整Mem比較並保留實際mtime後通過，44欄位判準、正式程式及原版RAM不改。
- 證據：研究038 §136，本機 ally1-items-dat-verified-v1.json、ally1-items-dat-{save,load}-v2/verified.json、ally1-items-dat-save-replay-v2/verified.json、ally1-items-dat-load-replay-v3/verified.json。

### ALLY #2：三稿審查未接受

原版24×32參照已獨立核對；三稿原生1086×1448均整幅正規化至72×96，未局部修圖。
第三稿恢復上方銀白區，腰／臂區外框吻合；肩部向左多5像素、左腿向內5像素、左鞋尖向外4像素，腰帶仍為紅條。
三稿均未接受，正常來源與READY契約未完成，未接入。正式ALLY仍2/31、敵人仍15/360。
證據：研究038 §137，本機 ALLY-02-v{1,2,3}-{generation,measurements,review}-20261004.json。


確認工具問題：oracle.Bytes逐byte走Machine.Read8，涵蓋EGA時會改Latch。原版重播改用既有machine-compare保存狀態的完整Mem欄位，仍核對44欄位及DOS，未排除Latch。載回另有Mem線性位址0x1096／0x1097兩byte不同，位址基準為保存狀態machineState.Mem索引；dosgolem internal/dos/find.go的DTA時間欄位+16h來自檔案mtime，保留真實GUI DAT的mtime後全部相同。診斷沿ally1-items-dat-machine-diagnostic-v{1,2}.go/.bin及v2.log，重播來源及77份實際依賴沿ally1-items-dat-gui-replay-build-v{1,2,3}.json。首輪控制及失敗資料保留，不重寫原版或降低判準。

本輪精確研究工具與生成提示保存於ally1-items-dat-source-text-v1.json，私有資料只記雜湊於ally1-items-dat-private-hashes-v1.json；進度同步沿ally1-items-dat-issue34-{before,body,after,sync}，收尾沿ally1-items-dat-final-audit-v1.json。024限定READY範圍及現行主題未改，完整HD仍未完成。

## 138. 已招募修洛斯後的Kasuruji複製路徑

2026-10-04。接續§136實際載回迷宮state，沿既有12-battle2正常鍵序到Kasuruji，查招募修洛斯後可選的複製分支。只使用正常按鍵、固定既有seed及唯讀觀察，不寫HP、能力、角色、座標或RAM；未取得貼圖來源前不推定ALLY #2與Kasuruji的關係。既有§129主角不能使用的分支保持原結果，不重擲該state。工具與收據沿既有workplace/ida/hd-ally-recruit-20261004/ally2-kasuruji-*；美術沿§137的ALLY-02-v4-*，未審查前不接受或接入。

2026-10-04 22:00：§138本輪17鍵正常路線先遇到Shulosu，原版畫面已檢視，尚未到Kasuruji。原版Space按住30000000步後HP38→0，實際終點回到迷宮；沒有新ALLY draw／load／open或投降。路線與戰鬥兩份獨立收據各44機器欄位及DOS相同，CPU／RAM／埠負對照有效，零步匯出畫面相同。固定既有seed分別86AFh／74C6h，未重擲或改HP、ESP、角色、座標及RAM；不推定舊路線終點或ALLY #2對應。入口ally2-kasuruji-{route-v3,first-battle-v1}-event-verified.json及-end-zero.png。

工具v1因log撞output前綴而在原版啟動前退出；v2每條指令建立map，115秒到期未完成。v3改用固定9欄raw陣列比較，欄位改變時才建立紀錄；同正常鍵序、完整控制與44欄位判準不改。實際64份非標準Go來源精確保存於ally2-kasuruji-build-v3.json；失敗inputs、log與execution保留，正式程式未改。

§137第四／五稿均未接受。第四稿青色腰帶修正，亮度16頭髮[18,0,64,35]、肩[17,36,71,50]、腰臂[9,51,71,62]、腿[11,63,71,80]、靴[5,81,68,94]。第五稿頭髮、肩與腰臂外框相同，腿[13,63,71,80]、靴[8,81,68,95]；鞋底到原版下緣，鞋尖仍多1像素。兩稿原生1086×1448及72×96已檢視；內建imagegen與整幅Lanczos正規化，未局部修圖，正常來源與READY仍缺。正式主題及完成數不改。精確來源與提示沿ally2-kasuruji-source-text-v1.json，私有收據雜湊沿ally2-kasuruji-private-hashes-v1.json；進度同步沿ally2-kasuruji-issue34-*，最後核對沿ally2-kasuruji-final-audit-v1.json。

§138後續正常路線入口ally2-kasuruji-position-v1.go/.bin與position-after-first-battle-v1.json，直接唯讀解碼保存狀態Mem，不經Machine.Read8。戰鬥後實際區域0、(1,2)、北向、HP40、能量25；依既有路線接續北1／東6／南2，11鍵，收據沿ally2-kasuruji-route-remainder-v1-*。終點與第二個遭遇尚待實跑，不由路線推定。

2026-10-04 22:22：§138六段正常原版接續與獨立驗證完成。### ALLY #2：六稿審查未接受

原版24×32參照已獨立核對；六稿均整幅正規化至72×96，未局部修圖。
第四稿已修正青色腰帶。第六稿左腿範圍較接近原版，但鞋尖又向左多3像素，
頭髮、肩部及腰臂仍有偏差。六稿均未接受，未接入。
正常來源與READY契約仍未完成；正式ALLY維持2/31，敵人維持15/360。
證據：研究038 §137–138，本機 ALLY-02-v{1,2,3,4,5,6}-{generation,measurements,review}-20261004.json。

### Kasuruji：已招募修洛斯的正常複製分支

- 從真正DAT載回的隊伍接續。途中打贏修洛斯後，實際位置為(1,2)、北向；再用11鍵抵達(7,3)，原版畫面確認Kasuruji。
- 正常攻擊取得第一個投降返回state，敵人HP27。放開空白鍵後，F2 → Duplicate 的伙伴選單只有Kai／Cancel，沒有修洛斯。
- 正常選Kai仍顯示不能使用這項超能力；已選Cancel返回Talk／Duplicate。沒有取得新的ALLY來源，不重試同一不可用分支。
- 本輪六段正常輸入的44項機器欄位及DOS對照均相同，CPU／RAM／埠負對照有效；完整原版終點及投降PNG已檢視。固定各起始state與seed，沒有改HP、能力、角色、座標或RAM。
- 攻擊途中只有既有ALLY #1的XOR貼圖，獨立原版全圖64000像素相同。以上是原版來源研究，未作HD／中文GUI完成證據。
- 攻略指出Pionn有複製能力；本機DOS名字在I_ENMY07.BIN的位元組偏移242。正常取得路徑、能力及ALLY #2對應仍待驗，不由攻略或檔名推定。
- 證據：研究038 §138，本機 ally2-kasuruji-{route-remainder,attack,f2,duplicate,kai,cancel}-v1-event-verified.json、ally2-kasuruji-pionn-inventory-v1.json。

原版投降寫入runtime0161:46EF於288022226步AL=1，HP29；首次輸出runtime0161:4AD9於288194850步進入，289011834步返回0161:48F0，HP27。投降state只來自本輪實際執行，原長攻擊終點HP0保留，不選有利seed或改值。放開39h後F2，再Down／Enter選Duplicate，Enter選Kai；本輪明示不能使用，Down／Enter選Cancel返回。每段終點44欄位／DOS相同，不將第一投降state宣稱為獨立44欄位控制點；投降零步全frame核對與PNG已檢視。

Pionn名字確認限I_ENMY07.BIN原檔偏移242；攻略的能力與二訪Sivad路線未由DOS正常取得驗證，候選80-byte記錄首字組0100h不命名為肖像或能力。下一步轉查具複製能力伙伴正常取得，不重試目前Kai；ALLY #2對應仍unknown。第六稿局部輪廓改善但多區偏差，accepted=false；原版比例及排版不放寬。新增精確來源沿ally2-kasuruji-route-source-text-v1.json，私有雜湊沿route-private-hashes-v1.json，六段總收據route-summary-v1.json，同步與收尾沿route-issue34-*及route-final-audit-v1.json。

## 139. Pionn的正常取得與第7組載入前提

2026-10-04。§138已確認目前Kai／修洛斯隊伍無法複製；攻略Pionn能力與DOS名字已定位，但不能由I_ENMY07檔名推定地點或正常路線。先從既有正常15-minton2.state接續Samar發射台，觀察實際目的地選單及正常跨區載入，再追第7組來源。這是原版來源與HD取得前提研究，不改玩法、能力、HP、seed、角色或座標。新增入口沿既有workplace/ida/hd-ally-recruit-20261004/pionn-*；執行器沿已核對的ally2-kasuruji-observer-v3.bin與recruit-verify.py。原版腳本只讀，未知欄位保留原始定位，不猜判定或增加正式HD數。

§139新區域的MAZE來源抽樣入口pionn-maze-state-read-v1.go/.bin與pionn-maze-source-v1.json。接續已取得Samar發射台、Rusteck抵達與退出三份正常state，唯讀檢查已證實18×18原始表與2048-byte來源；不依賴待選A/B格式，不先改正式資料或ROOM優先序。

2026-10-04 23:13：§139正常來源與使用者格式定案。

### 迷宮資料格式已定案，新增正常來源抽樣

- 使用者選定 A：256 個 MAZE slot 集中在一張192×192圖集，每格12×12。新版主題提案保留既有 entries 與 /1 相容；全域8×8、原版位置及框線先後沿既有定案。這次只定資料表示，美術與正式接入仍待審查。
- 從既有正常15-minton2.state走Samar發射台，正常選Rusteck並離開降落平台。三段均固定seed F95Bh，44機器欄位及DOS與無觀察控制相同，CPU／RAM／埠負對照有效；原版終點PNG已檢視。
- Rusteck實際成功開啟I_MAP04.BIN、CODE4.BIN、I_MENU04.BIN、ENEMY04.PBL及I_ENMY04.BIN，實際區域4。沒有載入I_ENMY07，未取得Pionn、新ALLY來源或新正式HD素材。
- 三份保存狀態的2048位元組MAZE來源與原檔相同。Samar發射台／Rusteck降落平台的舊18×18表與畫面分別差4632／4643像素；離開平台後(1,6)北向的72×72視野重建差0，全部三份單像素負對照有效。表存在不代表可以顯示HD迷宮。
- 證據：研究038 §139，本機pionn-{launch,rusteck,rusteck-exit}-v1-event-verified.json及pionn-maze-source-v2.json。單張圖集原型沿workplace/hd/maze-theme-prototype-v2-20261003/；正式主題維持31筆／29PNG、敵人15/360、ALLY2/31。

本輪Pionn探索使用既有正常solo Kai的15-minton2.state，不宣稱接續§138的招募隊伍。Rusteck鍵序為發射台Down三次／Enter，退出平台Up；三段14／8／2次IRQ1均核對。原版OnOpen回呼在成功os.Open及handle建立後觸發，來源見worktrees/dosgolem/internal/dos/files.go；v4只增加成功開檔的唯讀紀錄，精確64份實際Go來源及工具版本保存於pionn-observer-build-v4.json。

MAZE核對位址為保存狀態Mem線性0x468B、324bytes，對應runtime0161:307B；來源線性0x12E16、2048bytes。原始MAZE.BIN SHA-256為8b3b08ef483ff8c64beb04c69c861763c1258afba2348a7058e828bcf4060756，Go1.24.13、Python3.11.2。首次v1負對照在降落平台改到原本失配的像素，總差異數未變；保留v1，v2改選吻合像素再變更，三份差異數各增加1。未排除任何像素或改期望迎合原版。精確Go讀取器與Python驗證來源、binary SHA及三份輸入SHA保存在pionn-maze-source-v2.json。

A只定單張圖集及既有提案的相容性，未定迷宮美術、ROOM優先序或戰鬥效果混色。正式loader與31筆主題均未改；來源、私有雜湊與同步沿pionn-source-text-v1.json、pionn-private-hashes-v1.json及pionn-issue34-*，收尾沿pionn-final-audit-v1.json。

2026-10-04 23:22：單張圖集的新區域原型與色盤限制。

- 單張圖集在新走廊的原型有100個有效8×8格；視野外差0，改一像素會移除(0,120)整格，恢復後原型相同。原型圖集偏暗：它採用早期BBS中途色盤，與BBS及Rusteck原版終點的六個亮色不同。保留舊圖集與失敗；美術仍未接受，需先修正來源基準再驗。證據pionn-atlas-preview-receipt-v1.json、pionn-atlas-palette-diagnostic-v1.json，比較圖pionn-atlas-comparison-v1.png。

首跑原PNG讀取器只接受RGB／RGBA，正常原版PNG為索引色，於輸出前失敗。次跑透過ImageMagick唯讀解碼後，發現舊原型色盤與原版不同，再次於輸出前失敗。兩次精確來源與失敗沿pionn-atlas-preview-failure-v{1,2}.json；回查路由、研究005與原版probe palette契約，沒有新增產品特例。第三次僅用真實原版PNG取得原圖色盤，未改atlas；範圍驗證明示只驗位置、整格退出及恢復，不降低美術判準。舊BBS實際終點PNG的六個亮色與新Rusteck終點相同，而原型採用BBS較早pal檔的低亮色，確認為基準選取差異；尚不宣稱精確淡入完成步數。原型與比較圖已檢視，色彩未接受。

## 140. 迷宮單張圖集的穩定色盤與房間重疊原型

2026-10-05。接續§139。先由真正正常Rusteck走廊與已驗Sivad平台出口state零步匯出完整色盤，核對44欄位／DOS與原始frame。保留256原始slot及既有幾何輪廓，從可編輯幾何來源重生圖集，不改舊PNG。研究工具沿既有workplace/ida/hd-ally-recruit-20261004/maze-stable-source-v1-20261005.py與maze-stable-source-v1-20261005.json；原型沿workplace/hd/maze-theme-prototype-v3-20261005/及make-maze-stable-prototype-v1-20261005.py。只產生單張圖集及31筆相容提案，不產256獨立PNG；正式loader／主題不改，未接受美術或決定ROOM優先序。

§140零步色盤匯出首跑將-save-state誤當終點路徑，於執行前退出1；原始source及log保留。既有probe排程保存只在迴圈內執行，零步時不會另存state，因此v2只核對零步frame及完整palette/PNG，正常state的44欄位證據沿§139與§73保留，不宣稱本輪零步重新作44欄位控制。實際入口maze-stable-source-v2-20261005.py/.json；未增加正式遊戲特例。

§140色盤映射核對入口maze-color-state-read-v1-20261005.go/.bin、maze-stable-source-v3-20261005.py/.json。gob唯讀取得DAC及AC，按既有internal/machine/vga.go的DACIndex契約映射；完整色盤比原始DAC與PNG，AC負對照驗證有效。v2留下完整DAC／frame／PNG，不符源於漏用映射，舊「中途淡入」判斷待勘誤；不重試同一錯誤解讀。

§140原型生成工具v1在PNG輸出前因JSON list與記憶體tuple的輪廓比較退出；沒有內容差異證據。保存精確失敗來源後，v2用JSON同型表示逐點比較，圖形及判準不變。實際入口make-maze-stable-prototype-v2-20261005.py、maze-native-prototype-failure-v1-20261005.json。空輸出目錄確認後移除，再沿原路徑乾淨重跑。

§140第二次生成失敗：來源導覽誤選歷史v1失敗副本，bytearray色區累加溢位。回查v2 receipt輸入索引，正式成功原型來源為tools/hd/prototype_maze_theme.py，以整數陣列累加。第三版make-maze-stable-prototype-v3-20261005.py改用此實際來源，不修改renderer；256份JSON輪廓逐點相同才通過。保留兩次精確失敗來源與空輸出範圍，原型v3仍未正式接受。正確映射可讓原本同RGB的明暗色區重新顯示，所以新可見差異slot數以實測為準，不硬套舊73值。

§140第三版圖集與Rusteck預覽已產生，重疊比較讀ROOM22正式PNG時發現它依原點8×8補成240×240；216×216為原始內容範圍。保留v3部分產物及失敗來源，第四版按正式padding取內容作技術比較，不改原PNG。完整原型入口workplace/hd/maze-theme-prototype-v4-20261005/與make-maze-stable-prototype-v4-20261005.py。

§140尺寸判讀勘誤：第三次失敗未讀出實際tuple，第四次錯推240×240。直接讀正式PNG確認ROOM0-22.png為216×216、RGB三通道；正式8×8padding由runtime建立，不在該PNG中。前三次「240×240 PNG」文字是未驗推斷，現在撤回。第五版只在技術比較中將原RGB通道展開，不修改原圖。完整可審閱原型改由workplace/hd/maze-theme-prototype-v5-20261005/與make-maze-stable-prototype-v5-20261005.py承接；前四版失敗及部分產物保留。

2026-10-05 01:45：§140新證據與完整原型。

### 迷宮單張圖集：EGA 映射勘誤與修正版

- 已證實偏暗原因是把原始 DAC 色盤當成 EGA 畫面色盤，漏用屬性暫存器映射。上一輪「中途淡入」成因撤回；原版PNG會先按VGA.DACIndex映射，遊戲與原版資料未改。
- 正常Rusteck走廊及Sivad平台出口兩份保存狀態零步核對，完整RGB與DAC／AC映射相同。錯把原始DAC當畫面色盤差21016／21266像素；改錯AC的負對照差6477／5327像素。完整色盤與兩份原始frame均已核對；本輪零步不宣稱重新作44欄位控制，原正常state證據沿§139及§73。
- 修正版從既有可編輯幾何來源重生，256個slot的輪廓逐點相同，全部11264個圖塊邊緣像素與正確原版色彩一致。76個slot有可見內部輪廓重繪，其餘保留直線色區；舊原型的73值不套用新色盤。
- 單張192×192圖集及新版/2可丟棄提案已完成。現行31筆entries與29張PNG逐張SHA不變，沒有正式切換主題或改loader。
- Rusteck及ROOM22內容別名兩份正常畫面，各100個8×8有效格、視野外差0。改一像素後整格移除，剩99格；恢復後相同。完整比較图已檢視，ROOM22高光與圖集平面風格不同，已依grilling提出單一重疊外觀選擇，尚待使用者回答。
- 現行正式仍31筆／29PNG、敵人15/360、ALLY2/31。迷宮美術、重疊優先序、原版淡入與其他場景尚未正式接受，024 §1.13仍DRAFT。
- 證據：研究038 §140、maze-stable-source-v3-20261005.json及workplace/hd/maze-theme-prototype-v5-20261005/receipt.json；room22-comparison.png依序原版／既有高光／修正版圖集。失敗與原型均保留本機。

本輪完成源資料與可撤回原型，不代表普通GUI／全動畫或完整HD完成。正式31筆主題未改，§1.13.2保持DRAFT，重疊決策未回答。精確來源與私有雜湊沿maze-stable-source-text-v1-20261005.json、maze-stable-private-hashes-v1-20261005.json，進度同步沿maze-stable-issue34-*，收尾沿maze-stable-final-audit-v1-20261005.json。

§140使用者已回答「統一用迷宮圖集」，重疊外觀決策已定，不重新詢問；舊/1外觀保持，新/2含maze時採圖集。來源閘門核對入口maze-source-gate-v1-20261005.py/.json，16份既有正常state的完整來源、slot表、RGB及負對照；此為DRAFT來源契約證據，尚非正式runtime。

§140來源生命周期原型入口maze-session-gate-v1-20261005.py/.json。初始來源仍要求完整視野，已有身份且原始表不變時沿獨立原版Reference逐格移除／恢復。來源／表／色盤／非#22已知房間／全黑或右側SCREEN錨點失效會清除身份，載回重新取得。MAZE作背景先畫，SCREEN／MENU與sprite在前；新/2含maze時停用ROOM22高光watcher，舊/1保持。這是待審查生命週期，不寫原版state，也不是正常GUI完成證據。

§140來源審查完成：16份正常狀態有9份符合完整來源、原始表、畫面與實際色盤，7份回原版；27個來源／表／色盤反向對照全部有效。生命週期原型通過冷載局部拒絕、已取得來源100→99→100、改表重新辨識，以及來源／色盤／已知房間／全黑／錨點失效。13份正常畫面的右側錨點全部與原始SCREEN相同。原始state雜湊不变；本輪沒有重新宣稱44欄位對拍。

審查結論：024 §1.13.3限定轉READY，接受完整原始內容取得來源、同表局部遮格、實際色盤不符回原版及新/2統一圖集。整張視野函式邊界、全場景、正式GUI、全動畫、效能、美術與交付仍未完成。實作入口apps/psychicwar/theme/maze.go；正式驗證需獨立合成及原版狀態控制，不以原型取代。來源／私有雜湊及收尾入口沿本節既有maze-stable-*。

§140正式技術接入：maze.go與maze_test.go沿READY契約實作，通用dosgolem未改。maze-runtime-test-v4-20261005.log通過獨立Python整屏期望、局部格移除／恢復、來源失效、16份正常state的9份顯示／7份回原版、RAM／暫存器／步數／cycles／色號緩衝不變、接續10000道相同正常指令及/1相容回歸。兩個既有ALLY #1專用收據測試未提供環境而跳過，不能宣稱本轮重跑它們。

前三次正式測試失敗為腳本環境與API誤用：原版EXE路徑、RunCycles要求原時鐘設定、Oracle.Steps是相對步數。各log保留，沒有修改原版CPU設定或玩法來通過；直接查Oracle.Run原始碼後，以同狀態兩側相對1000指令、共十次驗證。既有/hd/bg.idx測試輸入未掛載，後續依具體改動選必要回歸，不稱全套測試通過。

普通GUI及F11驗證候選沿workplace/hd/maze-theme-runtime-v1-20261005/，含原31筆與原29PNG，新增單張MAZE圖集；只是技術驗證選擇，不取代現行正式接受主題。GUI工具入口tools/hd/gui_maze_run.py與tools/hd/verify_gui_maze.py，執行／獨立期望及私有輸出沿workplace/ida/hd-ally-recruit-20261004/maze-gui-v1-20261005/。限定主題與正常原始輸入已可審查，完整美術與發行仍未完成。

§140 GUI首跑保存四份完整擷取，正式前端期限45秒結束且退出0，第五份擷取時視窗已消失。此為驗證期限問題，未有前端產品缺陷證據。保存maze-gui-failure-v1-20261005.json精確來源及前四份，改同工具鏈的有界期限，乾淨重跑入口maze-gui-v2-20261005/；不降低整屏、正常轉向或F11判準。

§140正式GUI獨立驗證完成：正常Rusteck起點、左轉、HD關閉、重開與真正F11共5/5完整960×600視窗相符，每份8相位至少1份吻合，不宣稱40張全相符。四份HD省略MAZE各差21／306／306／306像素，負對照有效；F11載回與保存的原版整屏相同。原始PBL、MAZE原表、圖集、正式文本及字型獨立合成，MAZE在既有框線與人物之前；主代理已檢視實際載回相符PNG。前端由有界腳本終止，terminal為-15，不宣稱自然退出0或原版自然亂數GUI重播通過。

§140完整機器控制入口maze-runtime-state-v1-20261005-*.state、maze-machine-compare-*-v1-20261005.json及maze-machine-summary-v1-20261005.json。16份正常狀態顯示前後零步，加一份10000指令接續，17組44欄位／DOS全部相同；51個CPU／RAM／埠反向對照有效。正式测试maze-runtime-test-v6-20261005.log亦通過透明圖集、未知來源、嚴格清單及舊/1回歸。此控制不冒稱GUI自然時序、原版所有函式或完整HD完成。

工具與版本、精確建置來源及二進位SHA入口maze-build-inputs-v1-20261005.json；Python／Go研究來源保存maze-stable-source-text-v1-20261005.json，原版與私有產物雜湊保存maze-stable-private-hashes-v1-20261005.json，遠端同步與擁有權／容器清理沿maze-stable-issue34-*及maze-stable-final-audit-v1-20261005.json。限定契約保留READY，完整美術、全場景、動畫、效能與交付未完成；現行接受主題31筆／29PNG不變。

## 141. ALLY #2重繪基準與第七稿

2026-10-05。上一輪§140為實際進展，迷宮已正式技術接入；CONTEXT目前狀態表仍有舊DRAFT列，本輪先按§140收據修正，未重開迷宮來源工作。其餘ALLY與敵人依完整HD目標接續。

ALLY #2原始色號只含0／1／3／4／5／7／9／11／12／13／14／15，沒有前輪MAZE映射出錯的2／8／10。現有六稿的頭髮、肩、腰臂及鞋尖偏差不能用MAZE色盤勘誤解釋。原始參照與第六稿已再次檢視；本輪改以原始放大參照作構圖主體，六稿只作已定案HD風格參照，避免把前稿的錯誤輪廓延續下去。只用內建imagegen重繪，整幅正規化，禁止局部程式修圖、搬移或縮小人物。正常ALLY #2來源與READY契約仍缺，未選入正式主題。

第七稿入口沿既有workplace/hd/redraw/ALLY-02-v7-{prompt,generation,generated,frame,measurements,review}-20261005；來源與版本、原版輸入、工具、進度同步及收尾沿既有workplace/ida/hd-ally-recruit-20261004/ally2-v7-*。原版PBL、參照與未確認公開權利的素材留本機，不進Git或公開封包。

第七稿原生1086×1448已完整保存，副本SHA相同，僅整幅正規化72×96。五區及整幅亮度16外框全部仍與第六稿相同，沒有改善原版輪廓的證據；未接受、未選入。測量入口ally2-v7-measure-20261005.py及ALLY-02-v7-measurements-20261005.json。回查imagegen的references/prompting.md保留約束與單項迭代指引，不以繼續同類提示代替來源或美術驗收。

### Minton正常加入路徑

原始ALLY #2與全部12個ENEMY檔的24×32圖面沒有逐像素相同來源，最接近ENEMY00 #6–#8也差471–473像素；這只能作造型線索，不能證實角色身份。既有replay/title-to-first-save.json記錄正常13-healed→14-minton1的Minton遭遇，§130已證實Shulosu的F2→Team up→命名可取得ALLY #1。本輪從13-healed接續相同前進鍵序，在攻擊前驗Minton的F2選單，不把Pionn複製能力研究當成唯一ALLY來源，也不預先宣稱Minton就是#2。

原版觀察沿已凍結的pionn-observer-v4.bin及recruit-verify.py。收據入口既有workplace/ida/hd-ally-recruit-20261004/ally2-minton-*-20261005；正常state、seed與鍵序固定後執行，不改角色、HP、能力、座標或原版RAM。取得來源後才審查READY及正式接入；本節不擴張正式完成數。

正常遭遇首跑終點382000000只涵蓋七鍵的13條邊緣；KeyEvery對每條按下／放開節流，最後放開應在384500000之後。觀察及控制state已保存且相同，但IRQ14判準退出1，event JSON未產生，不稱遭遇完成。保留v1，原鍵序與起點不變，v2把終點延至385000000，僅讓既有最後放開送完，不改期望或重擲seed。

【confirmed，限定正常招募來源】遭遇v2、F2、Team up、命名四段全部通過獨立44機器欄位及DOS控制，CPU／RAM／埠負對照有效。實際原版PNG逐段確認Minton、Talk／Team up、命名提示及加入後隊伍；命名按ASCII minton與Enter，起始seed ADBFh，終點445000000步。ALLY #2完整24×32來源唯一相符，原版在(232,152)以XOR貼圖；64000像素整屏重建不符0，省略來源差460像素、改一像素差1。沒有改HP、角色、座標、seed或RAM。這已取得#2正常來源，不等於HD美術或正式GUI驗收。

第八稿只提供原始放大參照，不提供第六／七稿風格圖。入口沿既有workplace/hd/redraw/ALLY-02-v8-{prompt,generation,generated,frame,measurements,review}-20261005；完整提示詞、原生圖、副本與整幅正規化各自保存。第七稿及其失敗保留；未驗收前不改正式主題數量。

第八稿測量工具、#2限定來源契約、技術驗證、進度同步與收尾入口沿既有workplace/ida/hd-ally-recruit-20261004/ally2-{v8,source-contract,runtime,issue34,final}-*-20261005。來源契約僅審查正常招募的(232,152)，不擴張到未驗道具位置或其他圖號；技術候選不代表美術接受。

第八稿原生1086×1448及整幅72×96已保存。肩部左界21→16、腿部左界9→16，青色腰帶改成紅色；單一原版參照仍未通過造型審查。ALLY-02-v8-review-20261005.json明示accepted:false，正式主題與美術數不改。本輪停止同一圖的重複生成，先完成已證實來源的技術接入。

024 §1.18經DRAFT及獨立證據審查，限定轉READY。ally2-source-contract-v1-20261005.json核對原始SHA／31圖／偏移796與唯一384-byte來源；貼圖前768像素全黑、前後及四段終點共六份完整右側SCREEN錨點相符。命名前不誤認#2，加入後#0共存，錯#1來源差410像素、錯位置差486像素、錨點及單像素負對照有效。四段正常觀察／控制各44欄位及DOS相同。本節只准(232,152)完整來源，沿既有8×8及冷載契約；道具位置、美術、GUI與交付待验。

## 142. Rusteck正常敵人來源與既有候選審查

2026-10-05。§141已取得ALLY #2來源及限定技術支援；第七／八稿未接受。本輪接續尚未完成的敵人範圍，不再重複生成同一ALLY。沿§139的正常Rusteck出口state，當前區域4、(1,6)、北向、HP28，終點660000000步。由正常方向鍵取得實際敵人及姿勢，不改角色、HP、seed、座標或RAM。既有ENEMY04候選只作美術審查，未有來源與READY契約前不進正式清單。

來源探針、候選測量、契約、同步及收尾入口沿既有workplace/ida/hd-ally-recruit-20261004/rusteck-sprites-*-20261005；既有細密素材沿workplace/hd/redraw/ENEMY04-refined-selection.json。原版、研究state及未確認公開權利的美術留本機。首段使用已凍結pionn-observer-v4.bin，六次Up，驗正常終點／機器控制，再按實際畫面決定下一個最小來源觀察；不先推定是哪一組。

【confirmed，限定正常來源】六次Up後仍(1,6)，北／東不可走；轉西後通行0，正常Up抵(0,6)。北側也不可走，Down轉南再Up後遇Jaxemo。六段正常輸入均與無觀察控制44欄位及DOS相同，CPU／RAM／埠負對照有效；不改原版判定。原版未攻擊分支最後陣亡，不用重擲或改HP延長來源序列。

新唯讀觀察器rusteck-sprites-observer-v1-20261005.go/.bin以既有pionn v4為底，僅增加原始13個PBL／391圖資料庫及首24次完整身體輸出，完整原生來源與64個實際Go輸入凍結。rusteck-sprites-southup-v1-20261005-event-verified.json獨立核對四次貼圖64000像素前後、原始source bank、機器及DOS。初次AL00在(32,152)唯一匹配ENEMY04 #3，packed384 bytes SHA a88303b7f1443ed2303fbe64f47f724d9a902425cfa76d090664e2bd8c3d4b6f，貼圖入口760686131、返回760710672步；完整#3 PNG已檢視。後續3→4及4→5各384-byte差分已獨立吻合，來源收據rusteck-sprites-body-source-v1-20261005.json。不推定反向邊或未知效果身份。

15張既有ENEMY04候選已按獨立PBL建立原版／候選對照。該15圖均無色號2／8／10，前輪MAZE映射勘誤不解釋本批輪廓偏差。原始對照預覽與PNG尺寸／SHA通過；對照拼接首跑因Fontconfig無可寫快取而SIGABRT，屬驗證環境失敗，保留精確腳本及30份預覽。改用不需文字的convert拼接並逐像素核對預覽縮放後，v2完成。入口rusteck-sprites-art-{failure-v1,measurements-v2,comparison-v2}-20261005。

已出場#3–#5的既有group1 v2主要配色沿原版，但三張頭部外框均不符：#3右界71→64，#4為67，且左側輪廓外擴；不因整幅外框相同就接受。一次新v3重繪只提供原始三格參照，保留每姿勢頭部及四肢範圍。原生圖、提示詞、逐格正規化、測量與審查入口沿既有workplace/hd/redraw/ENEMY04-group1-v3-*-20261005。未審查前不加入現行主題，正式敵人仍15/360。

§142三姿勢來源審查v2通過：原始SHA、30圖、391圖唯一、兩條有向差分、六份完整右錨點及六組44欄位／DOS控制均確認。原始DRAFT SHA a778e1a02da9e80cfb1bf0d51b3f149433b85955d5b52cce13aa1b3d9c3b9adb已保留精確副本；024 §1.19限定READY，僅授權來源技術。第一次審查裁切API誤用保留failure-v1，v2建立整屏參照後通過，不修改原版或期望。

正式enemy.go／theme.go接入ENEMY04 #3／#4／#5，明示右錨點及原位置；有向前驅僅#3→#4、#4→#5。新增apps/psychicwar/theme/jaxemo_test.go，測試入口PSYCHICWAR_TEST_ORIG與PSYCHICWAR_JAXEMO_EVIDENCE明示本節原版目錄及研究工作區，再執行go test ./apps/psychicwar/theme -run TestJaxemoOriginalEventsAndGrid。全部工作走既有psychicwar-go-ebiten容器，原版唯讀、UID/GID1000、1 CPU、有界期限；不在主機執行。

runtime-test-v2四項必要Go測試通過、零略過。正常三事件及獨立Python grid-eligibility-v2期望對照整幅960×600合成PNG圖面，12／8／8格相符；錯姿勢／省略／單像素、進行中、遮格／恢復、HD停用、冷載受遮擋、未觀察反向、錯來源／跨檔／歧義、未知覆蓋／模式、失效錨點及嚴格manifest均驗。正式Attach由原版正常#3返回state接續兩次動作，完整圖面吻合，attached-machine-{4,5}-v1各44欄位與DOS相同，CPU／RAM／埠負對照有效。尚非普通中文GUI、真正載回接續或完整動畫完成。

候選group1 v3原生1881×836與固定三格完整72×96已保存。整幅輪廓雙向Chebyshev距離最大2／2／3個HD像素，最多1個原版像素；這是測量，尚非容差規則或美術接受。比較rusteck-sprites-jaxemo-comparison-v3-20261005.png及style-proposal-v1已送使用者依grilling選曲邊平滑／嚴格原輪廓，回答仍待定。不增敵人15/360、ALLY2/31與現行31筆／29PNG。

精確223份來源及188項原版／私有雜湊保存runtime-source-inputs-v1，含實際非標準Go依賴、DRAFT／READY規格與正式技術來源。同步／收尾沿rusteck-sprites-issue34-*及final-audit-v1；資料保全與清理結果由實際收據記錄，不推定完成。

## 143. Jaxemo正常攻擊與動作來源補證

2026-10-05。§142已完成三姿勢及兩條有向差分的限定技術接入；候選美術與曲邊標準仍待使用者選擇。本輪從真正正常#3返回state接續，執行前固定起點、原有seed、Space按住時機與有界終點，驗原版攻擊時的來源及動作。不改HP、角色、seed、座標、原始資料或RAM，不重擲以挑選結果。

沿用已凍結rusteck-sprites-observer-v1-20261005.bin及獨立rusteck-sprites-verify-v1-20261005.py。原始13個PBL／391圖資料庫、完整64000像素貼圖核對、44機器欄位／DOS控制及負對照沿既有契約。新研究產物入口為既有workplace/ida/hd-ally-recruit-20261004/jaxemo-attack-*-20261005；需要補新動作時先DRAFT與來源審查，再READY接入，不以靜態XOR猜補方向。

這是完整HD中敵人動作的來源補證，不代替美術或普通GUI驗收。正式接受主題31筆／29PNG、敵人15/360、ALLY2/31維持；所有原版／state／未確認公開權利的圖片留本機。

【confirmed，來源技術限定】

### Gestinti：另一組敵人的完整動作來源與技術接入

- 舊Jaxemo起點的Space、Space＋Enter與F3三個正常分支均陣亡；三組44機器欄位／DOS相同，沒有新增反向動作，停止重試同一起點。
- 改從已正常招募敏頓的state接續，在Samar下一場戰鬥存活，記錄24次ENEMY00 #6–#8貼圖。隊伍到發射台、選Rusteck、退出平台後向西一步，在(0,6)遇Gestinti，原版為ENEMY04 #6／#7／#8。原先預定南一步未完成，不把它記成Jaxemo。
- Gestinti首次完整來源及十次差分證實6→7→8→7→6→7→8→7→6→7→8。三姿勢在391個原始來源中唯一；十三次身體／盟友貼圖各64000像素重建差0，22份右側錨點相同。八段正常輸入各有44欄位／DOS及CPU／RAM／埠負對照；原始seed與新隊伍最後陣亡如實保留。
- 024 §1.20經DRAFT與獨立來源審查限定READY，正式載入器新增#6／#7／#8及四條實際有向邊。原版位置、全域8×8及明示右側錨點保持，與Jaxemo組分開。合成PNG核對十一完整圖面，首份11格、其後各8格。
- 五項Go測試通過且零略過，涵蓋兩組來源、正常循環、遮格／恢復、冷載、HD開關、未知／跨組及嚴格清單。正式Attach接續Gestinti十次差分，加Jaxemo兩次差分回歸，三組44欄位／DOS與無主題控制相同，負對照有效。
- 現行接受主題31筆／29PNG、敵人15/360、ALLY2/31不變。Gestinti美術、普通中文GUI、真正玩家載回接續及完整HD尚未完成；sprite曲邊標準仍待使用者回答。證據研究038 §143及jaxemo-attack-gestinti-{body-source,contract-review,runtime-test}-v1-20261005、runtime-machine-summary-v1。

原版Gestinti名稱由first-zero.png確認。八段正常輸入各固定原state及seed，三Jaxemo分支F95Bh，敏頓路線ADBFh，其餘8CD8h；不聲稱跨seed骰序相同。敏頓路線實際終點(14,8)南向、HP40、ESP23；發射台(15,14)後實際恢復ESP30。平台退出(1,6)北向，向西一步就遇Gestinti，最終(0,6)西向、HP0，返回SELECT；其餘鍵已送到但未完成預定南步，不推定原版忽略原因。

§1.20 DRAFT全文與SHA e52fecdc9d44d35dff8e4003e7f834c7a2d391ff7f947666b7e5f8ce808f6da8沿contract-review-v1保存。正式新增apps/psychicwar/theme/gestinti_test.go，Docker內明示PSYCHICWAR_TEST_ORIG及PSYCHICWAR_GESTINTI_EVIDENCE為原版與本研究工作區，go test ./apps/psychicwar/theme -run TestGestintiOriginalLoopAndAttached。實際Go依賴及所有變動來源沿jaxemo-attack-source-manifest-v1-20261005.json；原版／state／私有衍生物只記SHA，保持本機。條目索引含session-scripts、issue34-*及final-audit-v1。新正式美術與普通GUI尚未驗，不升完整024為CONFORMED。

## 144. ALLY #2正常道具位置來源與接續

2026-10-05 05:46台灣時間。沿既有hd-ally-recruit-20261004工作區，原版唯讀、固定state及seed。以下均confirmed且限實際來源技術；候選美術與普通GUI未接受。

### ALLY #2：正常道具肖像的位置接入

- 正常已招募敏頓的See Items頁，從Kai按Enter後出現ALLY #2於(128,8)，完整24×32；384 bytes在391個來源中唯一，64000像素重建差0，省略來源差730像素、單像素負對照有效。
- 五份原版畫面的右側SCREEN錨點及下方#0／#2共存相符。再按Enter進道具列表，上方肖像恢復768個色號1；七段固定state／seed的正常輸入各有44欄位／DOS及CPU／RAM／埠負對照通過。
- 024 §1.21經DRAFT及來源審查限定READY，正式載入器支援#2道具位置，沿原版8×8及獨立身份。六項Go測試全過、零略過；正式正常Enter及真正保存／載回後接續100000步的兩組44欄位／DOS控制通過。
- 首兩次接續測試因缺少state記錄的/orig唯讀掛載而停在原版插入磁片提示。失敗來源／狀態／畫面保留，補掛載後用同起點、鍵序、期望及正式程式重跑通過，沒有改按鍵時序或降低判準。
- 現行接受主題31筆／29PNG、敵人15/360、ALLY2/31不變。#2美術仍未接受；此為合成PNG的來源技術，未當作普通HD中文GUI、完整動畫或交付完成。證據研究038 §144與ally2-items-{source-proof,contract-review,grid-proof,machine-summary}-v1-20261005、runtime-test-v3-20261005.log。

原版ALLY.PBL SHA c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219，共31圖；#2檔案偏移796、RLE終點1192。runtime原版CS:IP 0161:8705→8751、步數555800115→555824656，AX2000h、DS:BX1175h:53C6h、CX2002h、DX0304h。384 bytes SHA c0743ac3fa9ff26ea58843c2ceb9765803662214cd71ae89d62954f69202b985，768色號SHA 97f4ae6b77d1b89947acbc3a297ebe1afb8ff562de64eefbaa5f917fe6c14474。原始packed／前後frame及state留本機；貼圖前後原版上方從768個色號1變成唯一#2，12個格均完整可用。

實際原版See Items起點先Kai36格，再Enter後敏頓36格，下一次Enter至道具列表24格；數字包含下方Kai與敏頓各12格。期望由獨立Python原PBL模型提供；正式測試用三種不同合成色，不使用HD截圖自當答案。正常Enter的按下／放開時序與HD兩側相同，真正保存後載回並接續100000步；兩組機器控制均44欄位／DOS相同，CPU／RAM／埠負對照有效，不把此測試稱為普通視窗或真人試玩。

本輪Rusteck已招募隊伍的持續Space分支仍陣亡，實際終點(0,6)西向HP0、ESP17；已停止同一戰鬥重試。Sip Yontry及三次Down分支不作道具肖像來源。可見選單會略過未開放項目，文本表含See Maps不代表此state有該選項；See Items入口以實際PNG及原版來源確認。

§1.21 DRAFT全文與SHA 6c47b68c9cf7888edce119657954e68597107797369b22a6e27a1fb7a38417ba沿contract-review-v1保存。正式新增ally_items2_test.go，明示PSYCHICWAR_TEST_ORIG、PSYCHICWAR_ALLY2_ITEMS_EVIDENCE及既有ALLY1／Jaxemo／Gestinti證據，六項選定Go測試零略過。錯位置／裁切／錨點／重複／圖號拒絕；#2(128,8)原負對照改為未觀察(128,16)，原ALLYS來源拒絕#2改為仍未READY的#3，保持證據範圍。

索引：rusteck-active-attack-v1-20261005與ally2-{items-menu,items-select,see-items-menu,see-items-open,items-next,items-list}-v1-20261005保存七段正常輸入、run log、完整機器控制、原版終點及零步PNG。ally2-items-source-proof／contract-review／grid-proof／machine-summary-v1-20261005保存獨立模型、草案與來源審查。runtime-test-v1／v2及runtime-failed-source-v1／v2保留環境失敗；attached-v2-failed.png證實原版INSERT DISK提示，v3修正唯讀掛載後通過且不改正式程式及鍵序。同步沿issue34-*，精確來源沿ally2-items-source-manifest-v1-20261005.json及session-scripts-v1，擁有權／容器清理沿ally2-items-final-audit-v1-20261005.json。未commit／push／發行，完整024與HD目標維持未完成。

## 145. Sivad 抽樣與共用角色貼圖來源（2026-10-05）

### Sivad 抽樣與共用角色貼圖來源

- 已招募隊伍正常抵達 Sivad、離開平台，下一步遇到既有 Oogus。攻擊分支陣亡；三次 F3 的有界分支仍在戰鬥中，未取得新敵人來源，停止重試同一起點。
- 四段正常輸入的 44 項機器欄位與 DOS 對照均相同，CPU／RAM／埠負對照有效。保留的 24 次攻擊貼圖及 22 次退避貼圖，全屏 64000 像素重建差 0；觀察上限不代表完整動畫已驗。
- 從保存的 IDA 9.4 資料庫匯出共用角色載入、全圖與差分貼圖流程。原始資料庫雜湊未變；選擇器、650h 步距及 600h／50h 兩段複製已定位，尚未把全部圖號或動作關係列為 READY。
- 12 個敵人圖檔共 360 張：177 張 24×32、3 張 24×24、180 張 16×16。ENEMY08 #12–#14 為較短圖像；ENEMY02 #12／#13 像素相同。來源辨識需保留這些例外，不能只靠固定尺寸或像素唯一性推廣。
- 現行接受主題仍 31 筆／29 PNG、敵人 15/360、ALLY 2/31。證據入口：研究038 §145；本機 sivad-party-inventory-v1-20261005.json 與 sivad-party-body-pipeline-v1-20261005.json。完整 HD、美術與交付仍未完成。

輸入與工具：PW_UNP.EXE SHA-256 fd5115b91f014c47a1fb1a6e2ecfd645e293264fe614a4b350e92637cad2fbd9；保存i64 SHA-256 4db51a199f5913c52dc4964557b9a4aafb965754d663cb0173c9dfdfe9a00c56。IDA 9.4、Python 3.12.3，既有鎖定image 6f6d59af49d0008c4109a5295b5f374bdc007e2d1ab28cb9de08779584de2780，原始庫唯讀掛載後複製至容器/tmp。匯出五段指令與交叉參照，原始名稱、bytes及operand保留；沒有重新命名或改寫原庫。

位址均為IDA線性ea，與執行期CS:IP分開：sub_14038的14064h比較5，14072h以650h作步距；14085h及1409Eh分別指定600h與50h複製量，後一段目的為原版CS:3A4Ah。sub_1634B的1635Dh／1635Fh設定DH=3、DL=4，16361h設定AL=0；sub_16367的16379h／1637Bh同尺寸，1637Dh設定AL=1，兩者呼叫sub_18C15。以上指令定位為confirmed；全部PBL圖號、載入區塊與動作順序的泛化對應仍unknown，不因XOR對稱而猜補有向動作。

敵人圖像盤點由既有strict_pbl逐檔解碼，保存12份原檔SHA、360筆檔案偏移／RLE讀取終點／尺寸／像素SHA。ENEMY08 #12–#14為24×24；ENEMY02 #12／#13同像素。因此177張24×32中只有175張可由本次全庫像素單獨辨識。此為confirmed資產事實，不能取代載入器、動作、錨點、8×8生命週期或正常GUI驗證。

四段正常來源收據固定各自state、seed8CD8h及鍵序。Sivad到達與離開無身體貼圖，攻擊與三次F3分支分別保留24及22次；每次已保留貼圖的完整64000像素重建為0差異。兩名隊伍在下一步遇既有Oogus，攻擊分支陣亡，退避分支仍在戰鬥；不宣稱F3普遍失效，也不宣稱兩分支走到未實際完成的座標。觀察沒有新增正式美術，停止同起點戰鬥重試。

索引沿既有workplace/ida/hd-ally-recruit-20261004/：sivad-party-{arrival,exit,attack,retreat}-v1-20261005-event-{inputs,verified,independent-machine}.json及原始frame／state／PNG；sivad-party-ida-minimal-v1-20261005.{py,json,log}；sivad-party-body-pipeline-v1-20261005.{py,json,log}及command收據；sivad-party-inventory-v1-20261005.{py,json}；sivad-party-progress-v1-20261005.py、source-manifest、issue34同步與final-audit收據。全部原版與候選素材留本機。

本輪修正CONTEXT與Issue現況中仍稱ALLY #2道具位置拒絕的舊句，現行依024 §1.21。歷史研究／來源快照保持。未修改正式程式、擴大READY、接受新美術、commit／push或發行；下一閘門為共用載入器來源閉合，再審查限定契約。

## 146. ENEMY01 五組來源與原版四階段動作（2026-10-05）

### ENEMY01：五組身體來源與原版四階段動作

- 正常 Sivad 保存狀態中的五組資料已逐位元組對上原版 PBL 與 I_ENMY01.BIN。每組都有初始圖及兩份差分，15 張身體來源在 391 圖全庫中可唯一辨識。
- 原版 sub_1435B 的四階段分支決定動作順序。24 處指令在 IDA、原始 EXE 及保存 RAM 三側相符；獨立模型的五組 20 個階段與 20 個錯來源負對照通過。
- 024 §1.22 經 DRAFT 與證據審查限定 READY。正式載入器新增 ENEMY01 #0–#5、#9–#14，身體技術來源由 21 張增至 33 張；原版位置、完整來源、8×8 及生命週期保持。
- 四項必要 Go 測試通過、零略過，含五組原始 RAM 的 20 次轉換與既有 Oogus 遮格／恢復、未知來源、冷載與開關回歸。未把其餘四敵人的模型核對當作正常玩家實跑。
- 現行接受主題仍 31 筆／29 PNG、敵人 15/360、ALLY 2/31。新來源尚未完成美術與正常 HD 中文 GUI；其他圖庫、短圖、像素別名、小圖與完整交付仍待完成。證據：研究038 §146、body-bank-native-cycle-proof-v1-20261005.json、body-loader-go-tests-v1-20261005.log。

工具與輸入：沿§145的IDA 9.4鎖定image、原EXE及原i64雜湊，匯出body-loader-ida-v1-20261005.json。只用原始庫的容器/tmp複本；讀回schema、工具版本、輸出與原始庫SHA均相符，exit0。主機sandbox直接inspect Docker socket先拒絕，改用既有require_escalated後成功，沒有另建image。

位址空間：IDA程式段base10510h，MZ映像線性base10000h，原EXE標頭976 bytes；檔案偏移=976+IDA ea−10000h。原版CS0161h、IP=IDA ea−10510h。24處原始指令在IDA、原EXE及原始正常RAM三側逐bytes相同，不混用位址基準。IDA線性ea14DA6h的E8 b2 f5呼叫sub_1435B；14370h的and bl,3及14373h／14378h比較0／3，14376h／1437Bh跳往14387h取DS:[AF10h]，其他階段由CS:3B22h取指標、切DS至CS。14387h的導覽名word_1B420只是IDA推定DS，原始operand AF10h及實際DS保留；不把導覽名當新全域來源。

原版資料：零步載回真正Sivad首完整身體返回state，SHA01d377962d41f1b8f489db4e3b915a9c4009dae62e926e6f5c06b964ec5e87d4，步數650717685、CS:IP0161:8751、DS1175h。不可變probe SHA cd07e8789ceaaed785264d5289e21174368eb76a1aaae7125a275918acc043a6，直接匯出低於A0000h的Mem與原frame，零步frame完全相同；不經VGAlatch讀取，不改RAM、角色、座標或seed。body-bank-sivad-v1保存ram／frame／log、argv、原檔及來源hash。

五組DS:63C6h+650h×g的首384bytes為ENEMY01 #3g，+240h為#3g↔#3g+1的差分；CS:32CAh+180h×g為#3g+1↔#3g+2的差分；末80bytes逐字吻合I_ENMY01.BIN偏移80×g。每組16×16原圖另外位於+3C0h、+480h、+540h，實際依序對上#15+3g、#17+3g、#16+3g；本輪只保存原始身份，不接入小圖或推定其動作。最初按384／128連續切槽的探索有未匹配欄，保留body-bank-sivad-v1原始結果；後續以不預設位置的完整bytes搜尋查明布局，未把探索失配記成產品缺陷。

證據分級：以上尺寸、原始bytes、紀錄及指令分支均confirmed；五組20階段是原版分支與資料的獨立模型，包含20個錯差分負對照，15張來源各在391張全庫中唯一。不能稱其他四敵人已實跑全循環。首完整身體返回時原初始化尚未完成，CS:3B21h仍為33h；初始化歸零依sub_14038末端的原指令證明，不能把這個中途state當階段零。其他圖庫、ENEMY08短圖、ENEMY02別名與載入全部細節未在此閉合。

024 §1.22草案原文與SHA沿body-loader-contract-review-v1保存，經來源審查限定READY才修改正式程式。enemy.go新增ENEMY01 #0–#5／#9–#14及原版四條有向邊；現行技術身體來源33張，接受美術仍15/360。新的enemy01_bank_test.go直接取正常RAM的兩份差分，期望來自獨立收據；五組20次轉換、錯差分拒絕全過。既有Sivad非法圖號負對照由已READY的0／5／9改為仍不支援的15／29／30，清單負對照5改15，拒絕標準不降低。四主測試及21子案例全部通過且零略過；未宣稱GUI、逐像素HD圖面、當前binary的44機器欄位或全部動畫驗收。

索引：既有workplace/ida/hd-ally-recruit-20261004/下body-loader-ida-v1-20261005.{py,json,log}與command收據、body-bank-proof-v1-20261005.py、body-bank-sivad-v1-20261005.{json,ram,frame,log}、body-bank-{identity,cs-identities,native-cycle-proof}-v1-20261005.{py,json}、body-loader-contract-review-v1-20261005.{py,json}、body-loader-go-tests-v1-20261005.log、body-loader-progress-v1-20261005.py及source-manifest、issue34與final-audit收據。原版、RAM與候選圖片留本機，不加入Git或發行。

## 147. ENEMY00／03／04 正常保存圖庫與共用動作（2026-10-05）

### 四個敵人圖庫：60 張身體來源

- ENEMY00、ENEMY03、ENEMY04 各五組原始資料，在正常保存狀態中逐位元組對上原版圖庫與角色記錄；零步匯出的三份畫面均與保存畫面相同。
- 三檔各15張完整來源在391圖全庫唯一。原版四階段分支與實際兩份差分閉合，60個動作階段及60個錯來源負對照通過。
- 024 §1.23經DRAFT及來源審查限定READY，三檔各擴至#0–#14，連同ENEMY01共60張身體技術來源。ENEMY04反向邊及ENEMY00其他組的差分身份均有原版分支／資料依據；原版位置、完整24×32、8×8與錨點限制保持。
- 10項必要Go測試全部通過、零略過，共80個主／子案例。包含四檔原始RAM的80次轉換、未知／跨組拒絕、遮格／恢復、冷載、開關，以及正常Jaxemo／Gestinti正式Attach接續回歸。
- 首跑兩個測試仍把ENEMY00 #3–#8視為不保留差分身份，與新READY範圍衝突。依原版證據改為正對照後，同一批測試通過；首跑收據保留，未知來源與冷載負對照維持。
- 現行接受主題仍31筆／29PNG、敵人15/360、ALLY2/31。新來源未完成美術或普通HD中文GUI，未宣稱15名敵人全動畫已實跑。證據：研究038 §147、body-banks-native-proof-v1-20261005.json、body-banks-go-tests-v2-20261005.json。

工具：沿§146的不可變probe，SHA cd07e8789ceaaed785264d5289e21174368eb76a1aaae7125a275918acc043a6。既有psychicwar-go-ebiten:latest映像SHA 083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7；Go1.24.13、Python3.11.2。Docker限制1CPU、256m至1g、pids32至128、無網路，UID/GID1000。原版輸入唯讀；主機只用已授權gh更新Issue，未在主機執行分析或測試。

原版來源：ENEMY00取workplace/states/07-first-play.state，ENEMY03取19-zellwal.state；ENEMY04取§143正常Gestinti首次完整身體返回state，原Return.Step676717779、CS:IP0161:8751、DS1175h。全部以probe的絕對-steps 0零步載回，原始RAM只匯出0–A0000h，三份64000像素frame與既存frame完全相同。未注入座標、HP、角色或seed。三份state、frame、RAM、PBL、I_ENMY及輸出SHA逐項保存在body-bank-{00,03,04}-v1-20261005.json。

位址與證據等級：沿§146已核對的24筆IDA原始指令，三份RAM同樣逐bytes符合。IDA線性ea／EXE偏移／執行期CS:IP分開記錄，舊native-cycle-proof的所有輸入SHA本輪再次核對。資料定位為執行期DS:63C6h+650h×g、DS區塊+240h、CS:32CAh+180h×g、紀錄+600h；g0–4。原始尺寸、資料、指令與唯一身份均confirmed；60階段為原版分支／資料的獨立模型，沒有宣稱其餘敵人已正常跑完戰鬥。

024 §1.23的DRAFT原文與SHA沿body-banks-contract-review-v1-20261005.json保存，來源審查通過後才改READY及enemy.go。新增ENEMY00 #9–#14、ENEMY03 #0–#2／#6–#14、ENEMY04 #0–#2／#9–#14，共27張；ENEMY04 #3–#5反向邊由原版分支／資料補證。ENEMY00全五組保留已確認的差分身份，含既有#3–#8。既有右側SCREEN錨點與清單範圍未放寬。已READY圖號的負對照改為15／29／30，小圖與其他圖庫仍拒絕。

首跑body-banks-go-tests-v1保留兩個old-group失敗，為測試仍採舊READY範圍。改為other-ready-group正對照後，同命令v2全過；正對照仍驗遮格、圖面與冷載，未知／跨組等負對照未刪。Jaxemo舊反向拒絕案例由新原始RAM模型的兩條反向正對照取代，不靠XOR交換性猜補。十主測試含80主／子案例；四檔模型合計80轉換，全部錯階段拒絕。Jaxemo兩次與Gestinti十次正常Attach接續亦回歸通過，但本輪未產生新的44機器欄位控制收據，也未驗普通中文GUI或新美術。

索引沿workplace/ida/hd-ally-recruit-20261004/：body-banks-native-proof-v1-20261005.{py,json}、body-bank-{00,03,04}-v1-20261005.{json,ram,frame,log}、body-banks-contract-v1-20261005.py與contract-review-v1、body-banks-tests-v{1,2}-20261005.py、body-banks-go-tests-v{1,2}-20261005.log、body-banks-go-tests-v2-20261005.json、body-banks-progress-v1-20261005.py、source-manifest-v1、issue34-{before,precheck,body,after}-v1及final-audit-v1。正式測試入口為apps/psychicwar/theme/enemy01_bank_test.go的TestOtherNativeBankSources，明示PSYCHICWAR_BODY_BANK_DIR、PSYCHICWAR_TEST_ORIG及原版收據後執行；完整命令與環境沿tests-v2。原版、RAM、state與未確認權利的圖片保持本機。完整024未CONFORMED、#34保持OPEN；未commit／push／發行。

收尾腳本v1假定dosgolem有go.sum，該檔不存在，精確快照階段停止。這是腳本輸入列表錯誤；保留v1腳本及未完成tar，v2改依實際go list及存在的module檔核對，不更動正式程式或測試。現行保全／收尾索引為body-banks-source-{snapshot,manifest}-v2-20261005及body-banks-final-audit-v2-20261005.{py,json}。

## 148. 迷宮圖集的材質原型（2026-10-05）

沿§140已確認的256個slot、原版色盤及原始表，比較平面色區與局部線緣明暗。
這是可丟棄美術比較，不改正式來源契約、圖集格式、原版位置、8×8或ROOM22定案。
研究腳本為workplace/ida/hd-ally-recruit-20261004/maze-material-prototype-v1-20261005.py；
輸出索引為workplace/hd/maze-material-prototype-v1-20261005/receipt.json及兩份正常場景comparison.png。
原始向量輪廓與256個slot身份保持；材質選擇須依grilling展示比較後由使用者確認，未確認前不選入正式美術。

首跑v1在向量來源核對停止：執行期頂點是tuple，JSON來源是list，直接比較誤報。v2先正規化序列表示，仍逐點核對全部輪廓；v1腳本與空輸出目錄保留。現行原型入口為maze-material-prototype-v2-20261005.py及workplace/hd/maze-material-prototype-v2-20261005/receipt.json。

v2材質目視未通過：每格外緣強制原色切斷邊線明暗，出現規律斷線。保留v2比較與收據；v3只對原版穿越整格的連續水平／垂直線加明暗，短線及角點不猜延伸，取消材質的外緣RGB重設。原版來源與向量輪廓不變；新明暗屬未定案外觀，不能稱11264接縫RGB仍相同。新入口maze-material-prototype-v3-20261005.py及workplace/hd/maze-material-prototype-v3-20261005/receipt.json。

v3獨立PNG檢查抓到35個原本全黑的外緣像素被多色抗鋸齒填色，全部位於圖塊接合邊界。v4外緣先取原版色區，再計算同一條原版直線的明暗，不混合鄰色；原黑色保持全黑。v3比較及verify-maze-material-v1來源保留。現行比較為workplace/hd/maze-material-prototype-v4-20261005/，核對入口verify-maze-material-v2-20261005.py與maze-material-independent-v2-20261005.json；尚未選入正式主題。

2026-10-05 07:31。獨立核對v4通過：全部256格黑色範圍與A相同、27垂直／17水平原始條紋可連續重複，v2的36例失敗負對照有效；兩份原始72×72視野逐像素重建0差，對照PNG欄位及視野外不變相符。v4改變162個slot的材質；原始256份向量逐點相同，未改原版表或來源。以上confirmed且僅限原型幾何／PNG及條紋，其他接縫、普通GUI及美術接受仍unknown。

工具為既有隔離映像083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7、Python3.11.2；原始MAZE SHA8b3b08ef483ff8c64beb04c69c861763c1258afba2348a7058e828bcf4060756，位址／正常state沿§140，不重新解碼IDA或改RAM。本輪未跑遊戲或變更正式程式；黑色區域、條紋及比較欄位由tools/pbl.py獨立讀回，沒有呼叫材質生成器作期望。原型v1的序列表示假失敗、v2斷線與v3邊界失配均保留。

索引：同研究工作區maze-material-prototype-v{1,2,3,4}-20261005.py、verify-maze-material-v{1,2}-20261005.py、maze-material-independent-v2-20261005.json、maze-material-display-review-v1-20261005.json、maze-material-progress-v1-20261005.py、maze-material-issue34-{before,precheck,body,after}-v1-20261005、maze-material-final-audit-v1-20261005；原型輸出工作區v{2,3,4}各receipt.json保存完整來源／輸出SHA與精確生成器文本。材質方向仍待使用者決定，正式024 §1.13.4不升READY，#34保持OPEN，原版／圖片留本機。

進度腳本首跑缺少收據路徑/orig的唯讀掛載，讀取來源SHA時停止，未寫入文件。補同一個原始資料目錄掛載後，同腳本通過；此為工具環境問題，沒有改原圖或正式程式。精確原型／驗證器／文件來源保全沿maze-material-source-manifest-v1-20261005.json與maze-material-source-snapshot-v1-20261005.tar，收尾沿maze-material-final-audit-v1-20261005.json及其cleanup_receipt。

## 149. Sivad 隊伍 F2 的正常接續結果

2026-10-05 07:53。confirmed，限定正常 Sivad Oogus 分支。ALLY.PBL SHA-256 c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219；31 張中 #0–#15 為24×32、#16–#30為16×16。保存 RAM SHA-256 b77be8742959cb915917b939e0d77dd8ae2144e19236a89951da9cff59a1fee3，完整原始 packed 來源只在 linear 1AE16h／1B056h 對上 #0／#2。這是保存 RAM linear，非 IDA ea；無匹配不證明其餘來源永不出場。

第一段起點是§145正常首張返回 state，650717685步、runtime0161:8751；其 phase 尚未初始化，不能當作戰鬥就緒。651000000步按F2、終點666000000，起點seed8CD8h，畫面仍戰鬥。實際兩次ALLY來源為#2在(232,152)、#0在(264,152)，AL=1，完整384 bytes唯一與原檔相符。全部64000像素重建差0，省略來源差460／426、單像素差1。

從此實際終點接續，667000000步再按F2、終點680000000，保存seed6EB1h。實際畫面為Game Over，沒有招募選單、ALLY載入或新來源。兩次完整原版PNG均已檢視。不將F2未出選單推廣為所有Oogus不能招募；不從此分支猜補ALLY #3，不修改HP或重設seed重試。

兩段各IRQ1=2；獨立 machine-compare 核對44欄位與完整DOS相同，CPU／RAM／埠負對照有效；原版零步匯出frame相同。觀測二進位為既有pionn-observer-v4.bin，SHA9959142379701f695150fd0d0742adc14c6371e2bacef8eeb838585547d4a0e7，Go1.24.13；獨立recruit-verify.py使用Python3.11.2及原始PBL。Docker既有工具鏈083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7，UID/GID1000、network none。未變更正式程式、規格、美術或接受數，未驗HD／中文GUI／封包。

索引：既有工作區workplace/ida/hd-ally-recruit-20261004/，sivad-ally-{f2,steady-f2}-v1-20261005.py、各inputs.json／run.log、event.json及event-verified.json／independent-machine.json／state／frame／PNG；sivad-ally-inventory-v1-20261005.json、sivad-ally-progress-v1-20261005.py、sivad-ally-issue34-{before,precheck,body,after}-v1-20261005、sivad-ally-final-audit-v1-20261005.json。原版／state／圖片留本機，來源與收據保存於sivad-ally-source-snapshot-v1-20261005.tar及manifest。來源16／15分類未推論人物或效果用途。

本輪收尾：#34遠端全文與body-file及worklist相同，2026-10-04T23:54:09Z維持OPEN；224份Go／module來源與前次正式來源快照相同。兩repo git diff --check通過；worklist仍2條未完成。原版、既有二進位與接受主題未改，#44未改，未commit／push／發行。根擁有權與容器清理收據沿sivad-ally-final-audit-v1-20261005.json及sivad-ally-cleanup-v1-20261005.json。

## 150. 正常迷宮相鄰接縫診斷與 Celtac 交通限制

2026-10-05 08:10。confirmed，僅限本批保存場景與正常輸入。前次§148只驗27個垂直／17個水平條紋；本批獨立解碼PNG、讀原版保存表，沒有呼叫材質生成器作期望。§140的16份正常保存state中9份有效，另7份房間或不完整視野未當作迷宮接縫。每份2048 bytes來源與原始MAZE相同、324 bytes表重建72×72視野差0，保存SHA與先前來源閘門相同。

九份場景有273個不同有向相鄰slot／axis組合。只在原版兩側邊界同色色號時核對HD RGBA，共1821對：A平面原色0差，B微明暗261差、66組受影響。只改記憶體單邊像素的負對照能檢出，原型PNG沒有改。前六組差異最多的放大PNG已檢視，B在部分轉角／條紋接合有可見中斷。這是實際原型限制，未將純RGBA差異全部冒稱原版玩法缺陷，也未自訂視覺容差。v4仍非接受美術。重複條紋測試只足以支援其原範圍；新數字不代表遊戲全部接縫已驗。

原始MAZE SHA-256 8b3b08ef483ff8c64beb04c69c861763c1258afba2348a7058e828bcf4060756；A圖集50c9ff7fe25c83fdc55f8afd402b01530d409b19e7cc1c0a41a8388a84d35826；B圖集0f06ba471dec6513ca13d5282a87b9f1e35e0a2eaece8f53d67b543a92c2c824。資料位址沿§140，讀取工具是既有pionn-maze-state-read-v1.bin，保存來源與工具SHA見診斷inputs_sha256。Python3.11.2；Docker既有映像083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7，UID/GID1000、network none。放大僅供診斷，三欄原版／A／B，未局部修圖。

另從既有pionn-launch-v1-event-observed.state正常選Celtac。起點580000000步、seedF95Bh，580500000起四次Down及Enter、間距1000000，終點640000000。原版提示Watch it Celtac!及衛星攻擊，沒有新ENEMY來源；從實際終點640500000步Enter接續至680000000，仍固定保存seedF95Bh，畫面為back to Samar及Landing Pad。實際重新開啟I_MAP00／CODE0／I_MENU00／ENEMY00／I_ENMY00，不能記成已抵達Celtac或取得ENEMY05。兩段IRQ1=10／2，44欄位與完整DOS相同，CPU／RAM／埠負對照及零步frame相同，兩份PNG已檢視。原版state、角色、HP、位置及seed不改。

docs/walkthrough-1989.md §3.7所述Chia B前置只作後續線索，尚未動態驗證，不為HD來源抽樣重做全程故事或以RAM注入取得素材。停止同起點Celtac分支。正式Go、接受主題與規格READY範圍不變；材質方向因新證據重新呈現，AGENTS與grilling選擇問題已提出，024 §1.13.4保持DRAFT。完整HD、正式GUI、來源全庫、素材權利與交付仍未完成。

索引：既有workplace/ida/hd-ally-recruit-20261004/，maze-material-observed-seams-v1-20261005.{py,json}、maze-material-seam-review-v1-20261005.{py,json,png}；celtac-bank-{route,continue}-v1-20261005.py、inputs.json／run.log、event.json及event-verified.json／independent-machine.json／state／frame／PNG；maze-seams-progress-v1-20261005.py、maze-seams-issue34-{before,precheck,body,after}-v1-20261005、maze-seams-final-audit-v1-20261005.{py,json}、maze-seams-source-{snapshot,manifest}-v1-20261005及maze-seams-cleanup-v1-20261005.json。原版與圖像留本機。

2026-10-05 08:20 使用者明確選B：保留微明暗方向，先修接縫。方向已確認，未知項不包括A／B；AGENTS §12、024 §1.13.4、CONTEXT與worklist同步。v5先建立光柵化前的共用明暗場，涵蓋原版全部實際273組相鄰關係、所有條紋與同來源別名，1821對邊界差0。放大仍見內側突變，未接受；v6再以同色色區的明暗場平滑，保留同樣邊界約束，1821對差0，黑色區域／不透明／27垂直17水平／36舊負對照／三對別名均通過。未降低判準或修補原PNG。

v6六組最差接縫放大、Rusteck／ROOM22完整視野比較已檢視，過渡連續。188個slot材質變更；80次平滑迭代、保留向量明暗來源權重0.2，最後明暗場最大變動0.0000859784，這是原型計算參數，不是原版規則或美術容差。明暗場先按原版資料約束，再光柵化；未更動256套向量輪廓。兩場景原版72×72重建差0，100→99→100及視野外差0。此結果只涵蓋原型與九份有效保存場景；未見相鄰關係、普通HD中文GUI、正式素材選入與交付仍未驗，候選契約DRAFT。

最新候選workplace/hd/maze-material-prototype-v6-20261005/，material-atlas SHA-256 80c7c8da94768ff2b318dd21a15840c10e98c16e5cb29fb7f9f68d3fc30ddda4，receipt SHA 2bfc1ad0c046ea0f6081544a2cb8acbd696b1342b4b6f42b1aef9de54a4380f7。v5與舊v4保留。新增索引：同研究工作區maze-material-B-decision-v1-20261005.json、maze-material-v{5,6}-prepare-v1-20261005.py、maze-material-prototype-v{5,6}-20261005.py、maze-material-v{5,6}-verify-prepare-v1-20261005.py、maze-material-observed-seams-v{2,3}-20261005.{py,json}、verify-maze-material-v{3,4}-20261005.py、maze-material-independent-v{3,4}-20261005.json、maze-material-seam-review-v{2,3}-20261005.{py,json,png}、maze-material-B-progress-v1-20261005.py；本輪Issue最新正文為maze-seams-issue34-body-v2-20261005.md。已接受主題與15/360敵人、2/31ALLY、60來源維持。

本輪收尾：B方向已定案，v6候選的已觀察273組／1821對接縫差0，內側過渡已檢視；未稱完整迷宮或正式素材選入。#34全文與body-file／worklist相同，2026-10-05T00:21:29Z保持OPEN。224份Go／module來源與前批快照相同，兩repo git diff --check及worklist render／verify通過。#44未改，未commit／push／發行，原版／PNG留本機。來源與擁有權沿maze-seams-source-manifest-v1-20261005.json及maze-seams-final-audit-v1-20261005.json；容器清理沿maze-seams-cleanup-v1-20261005.json。

## 151. B 材質新轉向接縫修正與普通前端有限驗證

2026-10-05 12:13。confirmed，限原版正常保存起點的本批場景、原型幾何與普通GUI抽樣。§150的v6通過273組／1821對已觀察接縫；本批正常Left轉向新增證據，384組／2772對同色邊界有833差異。v6未選入接受主題，GUI 5/5相符不代表接縫良好。沒有以重跑舊場景掩蓋新差異。

v7在原版向量光柵化前增加兩批實際相鄰關係，共485個不同有向slot／axis組、3333對同色邊界約束。沿用同色平滑80次、來源錨定0.2，這是生成器工程參數，非遊戲規則或美術容差。原256套幾何及slot身份不變，194個slot有材質變更。九份舊有效場景與五份新GUI保存表獨立解碼原始MAZE／PNG，273／1821與384／2772兩批均差0。只在記憶體改一個邊緣像素的負對照有效，原PNG不改；舊v6新場景833差異保留。256格不透明、黑色範圍保持，27垂直／17水平條紋、三對來源別名、36個舊失敗負對照通過。兩份正常保存畫面原版72×72重建差0、100→99→100與視野外差0。未知場景接縫未驗。

完整主題候選maze-material-theme-v2-20261005為schema/2，原31筆entries及29PNG SHA全部保持，增加一張192×192圖集。MAZE SHA-256 8b3b08ef483ff8c64beb04c69c861763c1258afba2348a7058e828bcf4060756；v7圖集8884cc0c08e5f700b7cf39c96ff98a908f3eba67bc6c2a1febf04753932307b1。新接縫放大前六組與真正960×600 GUI的北向／西向畫面已檢視，保留微明暗方向，不更動人物與框線。

從現行正式Go來源重新建置cmd/psychicwar，Go1.24.13、347份實際非標準依賴已記錄SHA與來源，binary SHA e60e8078e1a7c3be4e2655cd13c3a90bba2659c111608e46b49590bb3d703fc6。普通GUI使用原正常pionn-rusteck-exit保存狀態，Left、Shift+F5關閉／重開、Right、F11；按鍵保持0.20秒。五份F10均保存真正原版state，獨立零步匯出RGB、indexed、玩家52 bytes與原版表。期望由原始PBL、MAZE、來源表、正式文本和字型合成，不用截圖當期望。每份八相位至少一張完整960×600差0，不能稱40張全部通過。四份HD的省略迷宮、換A平面原色、換舊B v4負對照全部有效。兩次45秒有界自然退出0，原版錄製各四筆Left／Right按鍵邊緣，HD及F10／F11未送入原版。

F11恢復52 bytes玩家資料、位置、朝向、HP、能量與完整RGB畫面。保存後已推進的indexed畫面有25點0／8差異，實際RGB完整相同，差異座標與色號保留；不稱indexed相同、整份RAM或44機器欄位對拍。錄製只記原版鍵，不記F11時間倒退；未用四筆鍵紀錄聲稱完整重播已對拍。此批中文只覆蓋正式可重建顯示，起點未重印的英文訊息仍可見，不稱全文或從開機驗收。音訊使用null，負載與軟體GL環境不作效能或真實音訊結論。

環境首跑缺xdpyinfo，在遊戲啟動前退出127；依image已有xdotool修正，不重建工具鏈。獨立驗證首跑把Path當作sha(bytes)輸入，隨後indexed相同假設與RGB契約不符；失敗來源與log保留，修正雜湊API與獨立完整RGB／玩家檢查。沒有改正式Go、原版資料、RAM、seed、HP或按鍵時序。v2 GUI由腳本SIGTERM退出−15；v3／v4改為正式-quit-after 45s自然退出0，資料保存完整。

Docker使用既有psychicwar-go-ebiten:latest，image SHA 083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7，UID/GID1000、network none、GUI CPU2／memory1536m，有界Xvfb及trap，原版雙路徑唯讀掛載。原始MAZE與表位址沿§140，不新增逆向定位。正式Go、原接受31筆／29PNG、敵人15/360、ALLY2/31及60身體技術來源不變；024 §1.13.4仍DRAFT，候選尚未選入。更多場景接縫、全部sprites／效果、來源權利、全動畫與HD交付未完成。

索引：workplace/ida/hd-ally-recruit-20261004/內maze-B-{theme-prepare,frontend-build,build-inputs}-v1-20261005、maze-B-theme-prepare-v2-20261005.py、maze-B-frontend-v1-20261005.bin、maze-B-gui-run-v{1,2,3,4}-20261005.sh、maze-B-gui-runner-v3-20261005.py、maze-B-gui-failure-v1-20261005.json、maze-B-gui-v{2,3,4}-20261005/，v2／v4含B-independent-verification.json、original-frames.json及真正state／PNG，v3／v4含terminal.json及record.json。獨立來源工具沿tools/hd/export_gui_dat.go與maze-B-gui-verify-v{1,2,3}-20261005.py，失敗log及export-verify-v{1,2}保存。

修正版索引：同工作區maze-B-gui-seams-v{1,2,3}-20261005.{py,json}、maze-B-combined-seams-v1-20261005.json、maze-material-prototype-v7-20261005.py、maze-material-observed-seams-v4-20261005.{py,json}、verify-maze-material-v5-20261005.py、maze-material-independent-v5-20261005.json、maze-material-seam-review-v4-20261005.{py,json,png}；原型workplace/hd/maze-material-prototype-v7-20261005/，完整候選workplace/hd/maze-material-theme-v2-20261005/。進度、遠端與收尾沿maze-B-progress-v1-20261005.py、maze-B-issue34-*、maze-B-source-{manifest,snapshot}-v1-20261005及maze-B-final-audit-v1-20261005.{py,json}。全部PNG、state及素材留本機，#44未改，未commit／push／發行。

本批收尾：#34全文與body-file／worklist相同，2026-10-05T04:13:27Z保持OPEN。224份正式Go／module與前批快照相同；重新建置347份實際非標準依賴的來源SHA全部相同。兩repo git diff --check與worklist render／verify通過。原版素材、PNG及state留本機，未commit／push／發行，#44未改。來源精確保存及容器／擁有權核對見maze-B-final-audit-v1-20261005.json。

補充索引：maze-B-edge-negative-v1-20261005.{py,json}獨立重算485組／3333對邊界，單像素記憶體負對照確實造成邊界比較失敗，恢復後差0，PNG不變。

## 152. B v8 四向接縫、原版平台轉場與本機素材選入

2026-10-05 12:57。confirmed，限本批原版來源、材質與正常保存起點的GUI。v7新增Right／Right／Up／Left五份F10；北、東、南、西四份table重建完整72×72相同。新東／南方向使相鄰聯集增至511組、3483對同色邊界，v7有61差異。v8以相同原版向量與光柵化前明暗場約束修正；舊九份273／1821與新四向506／3483兩批獨立差0。只在記憶體改一個HD邊界像素，重算511組／3483對實際檢出9處差異，恢復後0，原PNG不變。256格不透明、原版向量與黑色範圍保持，27垂直／17水平條紋、三對來源別名、36個失敗條紋負對照通過。195個slot有材質變更，兩正常場景100→99→100與視野外差0。新最差六組接縫放大、真正GUI東／南畫面已檢視；未知場景接縫不稱已驗。

第一次d-forward F10存於674850946指令，原版位置(1,7)、北向，完整ROOM0 #2；舊南向MAZE表與frame不符，迷宮格0。後續八份GUI相位沒有該保存畫面，最小整屏差52907，不能當作5/5通過。由真正state無新IRQ1接續，675350946及676350946均回到(1,6)、北向，原始表重建72×72差0；原版自行畫回迷宮。觀察程序沒有新增按鍵、改seed、角色或HP。不是新房間失效的證據。probe在指定最後指令上限前不匯出同一終點shot，因此此命令實際只產出兩份中途frame／state，請求三份不等於完成三份；保存log與缺項，未重新覆蓋。

第二次用現行正式binary e60e8078e1a7c3be4e2655cd13c3a90bba2659c111608e46b49590bb3d703fc6、同正常起點、同Right／Right／Up／Left，保持按鍵0.20秒。只在Up後多等2秒再F10，這是已證實轉場的穩定擷取，不改原版規則或改RAM。各自真正state零步匯出，五份北／東／南／平台返回北／西的完整960×600，每份八相位至少一張與獨立PBL／MAZE／slot表／文本／字型期望差0。五份省略迷宮、換A、換舊B負對照有效；八筆原版鍵邊緣與指定鍵序一致，自然退出0。只證明本批五份穩定畫面，未稱40張全部通過、完整機器／RAM／亂數重播、從開機或DAT驗收。初始尚未重印英文訊息仍可見，平台返回後正式中文訊息正常，不稱全文試玩。

原始MAZE SHA-256 8b3b08ef483ff8c64beb04c69c861763c1258afba2348a7058e828bcf4060756；v8圖集04eb8d3e57e0d860c0830d198ce77b333c3fdecf9628eb3a3f04bafa831b7b67。依使用者B選擇和單張圖集／ROOM22統一決定，來源與素材審查後先將024 §1.13.4限定READY，再選入workplace/hd/theme-ally1-items-maze-B-v1-20261005/，以-theme該目錄載入、Shift+F5切換。正式本機主題31筆PBL＋256格MAZE、30PNG；原31筆與29PNG SHA全部保持，舊主題及全部失敗候選保留。selected每檔SHA與真正GUI驗證候選完全相同，不將舊31筆的DAT／封包／機器對照外推新主題。

Docker仍用既有083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7、Python3.11.2、UID/GID1000、network none、有界Xvfb與trap。負載18.68，不作音訊／速度／效能驗收。正式Go與原版資料不改；敵人15/360、ALLY2/31、60身體技術來源未增加。素材接受是本機顯示層的限定接受，不宣稱全部迷宮接縫、全動畫、公開權利或完整HD完成。其餘345敵人、29ALLY、效果、DAT／效能／權利與HD交付仍待完成。

索引：既有workplace/ida/hd-ally-recruit-20261004/，maze-B-more-gui-runner-v{1,2}-20261005.py、maze-B-more-gui-run-v{1,2}-20261005.sh、maze-B-more-gui-v{1,2}-20261005/；首批export-verify-v1與verify-v2 log、room-diagnostic-v1.{py,png,json}入口為maze-B-more-room-diagnostic-v1-20261005.py及首批d-forward-*。無按鍵接續沿maze-B-room-steady-inputs-v1-20261005.json、maze-B-room-steady-v1-20261005.log及兩份{state,frame,pal}。

接縫／素材入口：maze-B-more-gui-seams-v{1,2,3,4}-20261005.{py,json}、maze-B-combined-seams-v2-20261005.json、maze-material-prototype-v8-20261005.py、verify-maze-material-v6-20261005.py、maze-material-independent-v6-20261005.json、maze-material-observed-seams-v5-20261005.{py,json}、maze-B-edge-negative-v{2,3}-20261005.{py,json}、maze-material-seam-review-v5-20261005.{py,json,png}、maze-B-theme-prepare-v3-20261005.py；原型workplace/hd/maze-material-prototype-v8-20261005/，候選maze-material-theme-v3-20261005/，正式theme-ally1-items-maze-B-v1-20261005/selection-receipt.json。首版負對照v2範圍數字標籤誤留485，實際重算3483對且檢出差異；v3修正標籤511，原版與PNG不改。

審查／同步入口：maze-B-contract-review-v1-20261005.json、maze-B-selection-v1-20261005.{py,json}、maze-B-more-progress-v1-20261005.py、maze-B-more-issue34-*、maze-B-more-final-audit-v1-20261005.{py,json}、maze-B-more-source-{manifest,snapshot}-v1-20261005。沒有新增public素材、commit／push／發行，#44未改。

本批收尾：本機31筆PBL＋256格MAZE／30PNG，原29PNG保持。#34全文與body-file／worklist相同，2026-10-05T04:58:37Z保持OPEN。224份正式Go／module與前批快照相同，347份實際非標準建置來源SHA相同；兩repo git diff --check與worklist render／verify通過。審查前DRAFT完整規格由前批受驗archive取回，保存maze-B-material-draft-spec-v1-20261005.md，與審查前SHA相同；READY規格SHA與selection收據相同。來源精確保存見maze-B-more-source-manifest-v1-20261005.json；Docker與擁有權沿maze-B-more-final-audit-v1-20261005.json。未commit／push／發行，#44未改。

## 153. B v9 正常DAT接縫修正、存讀檔及本機選入

2026-10-05 13:38。confirmed限本批原版資料、接縫、正式GUI與確定的DAT／玩家資料。沿使用者B方向，不新增取捨。v8正常Save Game三份與SELECT Load Game／ALLY #1道具頁四模式六份，整屏共9/9通過，512位元組原版DAT與獨立無覆繪控制相同。該批287組實際相鄰關係、2211對原版同色邊界卻有104對B材質差異；相同GUI期望並不能證明材質接縫合格。保留失敗診斷，回到原始256套向量與光柵化前共用明暗場，納入v8既有511組與新DAT證據聯集，共598組、4029對同色邊界。v9三批獨立原始table／frame／PNG檢查均差0：舊九有效場景273／1821、四向五份506／3483、DAT八有效迷宮287／2211。SELECT不是迷宮，不套MAZE。單像素只在記憶體改邊界，重新比較全部598組，0→15→0，原PNG不改。

v9全256格不透明、原始向量與黑色範圍保持，27垂直／17水平條紋、三對來源別名與36個舊失敗條紋負對照通過。204個slot有微明暗，兩保存場景100→99→100及視野外差0。最差六組接縫比較圖已檢視，未移動幾何。原MAZE SHA-256 8b3b08ef483ff8c64beb04c69c861763c1258afba2348a7058e828bcf4060756；v9圖集7dc7de0064339d9e54f958f3cbf5e00181c6aeb80484d3c3e54b5612cf71736f，A平面50c9ff7fe25c83fdc55f8afd402b01530d409b19e7cc1c0a41a8388a84d35826。未知相鄰關係仍是限制，未稱全部遊戲接縫通過。

修正版用相同正式前端e60e8078e1a7c3be4e2655cd13c3a90bba2659c111608e46b49590bb3d703fc6與正常迷宮／SELECT保存起點，重新以原版鍵盤Save Game→hd31、SELECT Load Game→hd31→See Items→ALLY #1；接著F5／Shift+F5／F5切四模式。原版鍵邊緣26／22筆，兩段前端自然退出0。零步原版frame／RGB／player匯出與獨立PBL／MAZE／表／正式字型、正式text期望核對，9/9完整960×600，每份12相位至少一份相同，省略迷宮／上方肖像與一像素負對照有效。八份迷宮為100格，SELECT0格；HD關閉不畫圖面。單元模型與照片存在不代替此次正常操作證據。

v9實際產生512位元組hd31.dat，SHA-256 23ebe0a003baf865c1a907746ebbddc1ba4e70e65e44f06a983b7e5496800ac7，偏移166為ASCII shulosu。與v8實際GUI及獨立無覆繪原版控制完全相同；載回52位元組玩家資料與控制相同。道具頁四模式52位元組及原#1於(128,8)／(232,152)、原#0於(264,152)相同，控制畫面亦同。道具操作後byte38原版由43變1，無覆繪控制同樣發生，不當作HD修改。DAT／player單位元組負對照各檢出1。無覆繪控制以v8實際GUI保存起點及相同鍵序獨立執行；只核對確定DAT與52位元組，不聲稱v9與該控制的RNG、同指令數全RAM／44欄位、全動畫或全部存檔狀態相同。

首批v8讀檔驗證器因70秒外層上限在5/6後終止、未寫收據。同六份原始擷取與不變驗證器加長有界執行至125秒後6/6通過；未重跑GUI、更換圖片或降低整屏判準。這是驗證腳本逾時，失敗log保留。首次GitHub讀取使用錯repo名稱，立即由git remote核對為wicanr2/psychic_war_cht，正式主機gh讀取成功；未當作憑證受阻。

審查來源、三批接縫、幾何、真正負對照及v9正常DAT GUI後，024 §1.13.4素材修正維持限定READY，再選入workplace/hd/theme-ally1-items-maze-B-v2-20261005/。以-theme該目錄載入、Shift+F5切HD；預設原版。現行31筆PBL＋256格MAZE／30PNG；原31筆與29PNG SHA保持，候選與選入每檔一致，舊主題與v8保留。347份正式前端非標準來源與91份pwstep依賴雜湊核對，正式Go／text／字型本輪未修改。敵人15/360、ALLY2/31、60張身體技術來源不增加；完整HD仍未完成，#34保持OPEN，#44不改。v8四向GUI5/5限歷史範圍，不外推v9全動作；未從開機、全場景、全相位、效能、公開權利或封包驗收。

所有下列入口位於既有workplace/ida/hd-ally-recruit-20261004/：maze-B-dat-{gui-run-v1,gui-run-v2,model-v1,verify-gui-v1,independent-v1,seams-v1,byte-proof-v1}-20261005.py及同名JSON；maze-B-dat-gui-v{1,2}-20261005.sh，maze-B-dat-{save,load,save-independent,load-independent,items-independent}-v1-20261005/。v8失敗驗證log為maze-B-dat-gui-verification-v1-20261005.log，重跑由load/verified.json與不變source SHA定位。正式pwstep來源入口maze-B-dat-pwstep-build-{v1}-20261005.py／.log與build-inputs-v1 JSON。

修正可重現入口：maze-B-v9-prepare-v1-20261005.py產生maze-B-combined-seams-v3-20261005.json與版本化生成／驗證器；依序執行maze-material-prototype-v9-20261005.py、verify-maze-material-v7-20261005.py、maze-material-observed-seams-v6-20261005.py、maze-B-more-gui-seams-v5-20261005.py、maze-B-dat-seams-v2-20261005.py、maze-B-edge-negative-v4-20261005.py、maze-material-seam-review-v6-20261005.py，收據各同名或maze-material-independent-v7，log為maze-B-v9-prototype-v1-20261005.log。圖集與完整比較在workplace/hd/maze-material-prototype-v9-20261005/；接縫放大maze-material-seam-review-v6-20261005.png。

候選／GUI入口：maze-B-v9-gui-prepare-v1-20261005.py、maze-B-theme-prepare-v4-20261005.py；workplace/hd/maze-material-theme-v4-20261005/。maze-B-v9-dat-{gui-run-v1,model-v1,verify-gui-v1,byte-proof-v1}-20261005.py、maze-B-v9-dat-gui-v1-20261005.sh，以及maze-B-v9-dat-{save,load}-v1-20261005/的execution／terminal／record／original-frames／verified.json。用同一正式maze-gui-export-v1-20261005.bin零步匯出。

素材審查／進度／保全入口：maze-B-v9-selection-v1-20261005.py／.json、maze-B-v9-contract-review-v1-20261005.json、maze-B-v8-ready-spec-snapshot-v1-20261005.md，現行主題selection-receipt.json；maze-B-v9-progress-v1-20261005.py、maze-B-v9-issue34-*、maze-B-v9-final-audit-v1-20261005.py／.json、maze-B-v9-source-manifest-v1-20261005.json與source-snapshot-v1.tar.gz。原版素材、states、DAT與未公開權利圖集留本機，不加入Git或發行包。

工具沿既有Docker image SHA 083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7、Go1.24.13、Python3.11.2、UID/GID1000、network none、1–2CPU及有界Xvfb／trap，原版雙路徑唯讀。軟體OpenGL與null音訊僅驗畫面，不作人耳、速度或即時效能結論。容器、root擁有權、來源與遠端全文收尾核對由本節final-audit保存。未commit／push／發行。

## 154. 四個載入庫的60張小圖來源及正常敏頓輸出銜接

2026-10-05 14:01。confirmed限原始EGA來源、資料存放偏移與本節原版輸出核對。沿現行CONTEXT的其他sprite範圍查核既有正常資料，未重開已完成B v9。既有21份Transport與13份ROOM22保存狀態只涵蓋區域0／1，未取得ENEMY02或其他新區域。正常發射台新區域接續未執行，不能把計畫寫成已取得新來源。進一步檢查四份既有正常RAM，發現五組小圖完整128位元組均已載入；原先把600h尾部按128連續切段，跨入64位元組尾段，不能作動畫差分來源。

執行期DS=1175h，組基底線性0x11750+0x63C6+g×0x650。每組的ENEMYxx #15+3g、#17+3g、#16+3g分別位於組內0x3C0、0x480、0x540，各完整128位元組。四檔00／01／03／04、20組、60張原始EGA與PBL精確相同，在ALLY加12檔ENEMY共391圖來源中唯一。原檔SHA與body-bank-{00,sivad,03,04}-v1-20261005.ram精確身份由source-proof記錄，原始保存起點沿§146–147；檔案偏移為各原PBL來源offset，RAM線性及DS偏移逐列標明，沒有把IDA EA混成RAM位址。60個單來源位元組負對照各檢出1，20個錯用0x500取第二張圖的128步距負對照全部不符。未改seed、角色、HP、位置、原始RAM或正式程式。存放順序不能推成動畫順序。

來源後64位元組按原PBL16色映射表生成2bpp CGA的初始假說檢查未全部相同，CGA_all_equal=false；原始失敗檢查逐筆保留。其格式仍unknown，不因段落長度符合便改稱confirmed CGA，也不把它當下一張EGA或XOR差分。目前正式遊戲只需EGA，這個未使用模式不阻塞完整128位元組來源核對，不為它擴大逆向範圍。

從四份載入源取ENEMY00 #21／#22／#23，重新獨立計算既有正常14-minton1中的209次16×16原版呼叫。104筆完整來源、105筆無方向XOR配對，來源唯一；各原始before／after／raw SHA核對，原版AL=1，完整64000像素paint重建差0。每筆在記憶體改raw第一個位元組，一個像素即不符，209個負對照各差1。事件位置、DS:BX、CX、DX、入／返回step逐列保存；未推論三張圖的動畫方向、HD透明或混色。既有1009筆整體收據的原始seed A48C、終點420000000、觀察／無觀察／舊基準相同保持；本輪只重算保存證據，沒有重跑原版戰鬥，沒有外推20名敵人全動作。

首版bridge使用ENEMY00字首篩事件，誤含24×32身體，讀出384 bytes後斷言失敗，沒有輸出通過收據。v2先按16×16尺寸篩選，用同原資料與209筆判準乾淨重跑通過。保存v1原始source及failure.json，屬驗證腳本分類錯誤；沒有更換原圖、原事件或放寬來源／整屏判準。讀既有資料時另有JSON鍵functions誤用及map.go不存在，改由實際keys／檔名確認；未當產品缺陷。

024 §1.24只追加來源DRAFT。正式小圖位置、模式、動作／重疊、效果透明／混色、美術、GUI與真正存讀契約仍未READY，本輪無正式Go實作。現行31筆PBL＋MAZE B v9／30PNG、敵人15/360、ALLY2/31、身體正式來源60張均不變；本批小圖60是資料核對數，不能相加為正式技術接入120或美術完成75。完整HD與其他圖庫仍未完成，#34保持OPEN，#44未改。

可重現入口：workplace/ida/hd-ally-recruit-20261004/small-banks-source-proof-v1-20261005.py／.json，以既有Dockerimage psychicwar-go-ebiten:latest、UID/GID1000、network none、原版／專案唯讀與該研究根可寫執行。小圖獨立來源使用tools/hd/explore_sprite_sources.py；正常輸出計算由small-banks-runtime-bridge-v{1,2}-20261005.py、v1-failure.json、v2.log／.json，呼叫tools/hd/verify_small_projectiles.py的原版AL0／AL1像素算式。舊正常事件入口workplace/hd/small-projectiles-minton-v1-20261002.json及independent-v1 JSON，該舊收據有879次全部小圖，其中本批ENEMY00為209，其他為BEAM。

進度與保全入口：同研究根small-banks-progress-v1-20261005.py、small-banks-issue34-*、small-banks-final-audit-v1-20261005.py／.json、small-banks-source-manifest-v1-20261005.json及source-snapshot-v1-20261005.tar.gz。已知正對照研究038 §153確有CONTEXT與spec入口，本節及新DRAFT同次掛入CONTEXT與worklist；沒有另建文件分類。

工具image SHA 083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7、Python3.11.2、1CPU與15–50秒外層有界逾時。load17.24以上，只驗靜態資料與保存輸出，不做音訊／速度／效能結論。正式前端347份與pwstep91份實際來源沿§153build-inputs核對未變；原版、RAM、DAT留本機。沒有新增接受美術、commit／push／發行，容器與擁有權收尾由final-audit記錄。

本節保全補記：追加§1.24來源DRAFT後，整份024 SHA改變；§153的B v9素材契約全文保持。由其精確source-snapshot archive恢復small-banks-prior-ready-spec-v1-20261005.md，SHA等於B v9 selection-receipt的spec_ready_sha256，沒有改寫舊收據。

## 155. 戰鬥效果混色比較重核（2026-10-05）

本節只將§38–39已獨立驗證的v5原型整理為可審閱的混色比較。confirmed限於保存收據與PNG身份、比較區精確像素、兩種方式的差異及範圍；建議B屬視覺判斷，尚非使用者定案。中途共識安全性仍待審查，效果美術、正式接入與完整HD尚未完成。

輸入是workplace/hd/battle-effects-prototype-v5-20261001.json、battle-effects-prototype-independent-v4-20261001.json及其21份frame／alpha／screen PNG。逐份indexed SHA與Go收據相同，PNG檔案SHA與獨立收據相同，RGBA SHA亦等於Go收據；原獨立驗證21份戰鬥區不符0及省略效果負對照20182仍有效。本輪沒有重新執行原版戰鬥，不將舊原版終點、RNG、RAM或真正讀檔當本輪新驗證。

比較選定兩個已記錄的語意時點：首次黃色MASK，step86703048；最多同時效果，step91366425。兩份皆完整分解，964／1000個全域8×8格有效，36格回退原版。原版／A一般透明疊色／B透光混色並列，相同原位置及美術。從960×600保存PNG精確取戰鬥區原版座標(32,144,256,40)，輸出2304×416帶標題版面，沒有縮放、改畫PNG或移動遊戲角色。兩模式分別差4664及10578個像素，全部在該戰鬥區；若標題A／B交換會檢出同數不符。加入標題後兩列比較區逐行與輸入RGBA相同，不符0。

比較圖SHA-256為6b80f972427cbbc9ccbb79e6c7557f49ef5d7ba1f21d1ae15f8127cc48aaa9c4。可重現入口是workplace/ida/hd-ally-recruit-20261004/effect-blend-review-v1-20261005.py／.json／.png，精確全部輸入SHA列於JSON；-panels.png保存無標題原始並列。以既有psychicwar-go-ebiten:latest、UID/GID1000、network none、1CPU、768MiB、50秒外層逾時，專案唯讀及既有研究根可寫執行；工具image SHA 083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7，Python3.11.2。ImageMagick只畫版面標題，Fontconfig可寫快取警告不影響PNG產出或比較區逐像素檢查。

已以此圖提出一個共同決策問題：A效果顏色較實，重疊會遮角色；B透光混色效果較亮，角色較清楚，建議B。這份舊原型的角色／背景只供混色比較，不宣稱現行31筆＋迷宮B v9全場景。單純選擇混色不改原版EXE、RAM或DAT，可撤回；正式資料格式仍需另經原始證據、DRAFT與必要確認。024 §1.5保持DRAFT，中途共識／未知回退、內建遮罩主題表示、正式HD切換、效果中途真正載回與美術驗收仍待完成。

本節精確來源與版面保全沿同研究根effect-blend-source-manifest-v1-20261005.json及effect-blend-source-snapshot-v1-20261005.tar.gz，包含腳本、原型輸入、進度文件與Issue全文；各檔及壓縮包SHA由final-audit核對。素材只保存於本機研究根。

CONTEXT與worklist同次掛入本節與比較圖。文件寫入來源effect-blend-progress-v1-20261005.py，遠端同步effect-blend-issue34-{before,body,after}-v1-20261005，收尾effect-blend-final-audit-v1-20261005.py／.json。已知正對照§154與其small-banks-source-proof入口確在CONTEXT，沒有另建文件分類。31筆PBL＋256格MAZE／30PNG、敵人15/360、ALLY2/31及正式身體技術來源60張維持；#34保持OPEN，#44不修改，未commit／push／發行。

## 156. 效果共識與回退的限定審查（2026-10-05）

本節解鎖024 §1.5的部分方法審查，維持DRAFT。confirmed限於已保存輸入、408個畫面副本負對照、八類副本回退及四個已受驗PNG的HD正對照。共識安全性的數學條件列為條件推導，不宣稱任意未知來源、正常執行身份、完整動畫、GUI或正式效果完成。混色與內建遮罩主題表示仍未定案，未問新產品問題。

工具介面由tools/hd/prototype_battle_effects.go精確SHA bf55289327419cf916eb93e7d1faa29f8e689fc70cb172a9bd873f98c247c232建立於既有本機研究根。原本正常重播main改名保留，原render函式委派給新增renderCopied參數介面；原渲染函式主體、decode、model及basis完全沿原碼。只在新Go檔追加負對照，不改正式Go或舊原型。v1只跑回退；v2增加正對照及錯回退負對照，兩版原碼、二進位與收據保留。

獨立真值仍是§38–39的原始PBL、PW_UNP.EXE CS:4E36位元遮罩、原版方向與唯一分解profile，114變數、51投影矩形。21份保存indexed SHA與舊收據相同，分解值／known／complete保持。兩份用step86538216與91366425定位的frame，按整幅indexed精確SHA連到已獨立驗證的原版完整邊界資料；前者可連到step86504412的相同畫面，不把這種畫面身份當整機同狀態。按原始資料重建(32,144,256,40)戰鬥區均差0，區外原版迷宮／數值輸出分別有490／477像素未建模，不能稱全屏來源重建通過。

固定這兩份真值畫面，在51矩形各中心依次XOR色號的四個位元，共408份副本，原始indexed與機器記憶體不改。每份SHA、唯獨一像素改變及位於宣告矩形內均獨立核對；Go均回到不完整共識，已知位元與原版真值差0，已啟用變數同Name／Rect無雙姿勢。檢查器不以Go輸出的active列表反填真值。將已知變數故意翻一位會被獨立真值檢出；將原版身體來源一像素翻轉，完整戰鬥區重建差1。

原始方法的條件推導：當有一個已知投影矩形包含全部未知繪製差異，且投影後的真實合法狀態仍為候選，該候選的固定變數必與真值相同。basis消去時的零向量組合產生nullspace生成元；其所有支援變數都保守列為歧義。跨各一致候選取fixed交集，並移除候選值有分歧的位元，因此共同保留的位元不會違背該真實候選。這個條件推導不證明未知輸出必然落在51矩形之一，也不證明每個真實中途只有一次繪製、所有像素等價的未知來源可辨識。完整向量可解且合法只證明已知原始圖面可重建觀測，不足以單獨確認原版執行來源；正式接入仍需原始來源與生命週期證據，不能把此有限審查當所有未知來源安全閘門。

八類固定副本案例為：區內未知像素、分解範圍外的像素、兩處分離污染、未知戰鬥圖樣、同位置兩張身體姿勢，以及ENEMY01／03／04原檔#6的未登錄完整身體。最後三份使用本場原型之外的真正24×32 PBL來源，沒有把它們登記為新HD素材。未知像素／兩處污染／未知圖樣整場景回退；區外變更回退其全域8×8格；雙姿勢及未登錄身體保留三個身體變數為未知，整個24×32矩形回退。八類×兩混色的指定回退區逐像素差0；故意在回退區改一個回傳像素，每份檢出1，共16份。此負對照使用人工色號標籤RGB區分每個色號，只驗原版raw回退與範圍，不宣稱原版色盤或GUI畫面。

為排除全部raw也空過的情況，兩份完整場景×alpha／screen，使用舊原版RGB PNG為輸入，戰鬥區輸出與原獨立核對過的v5 PNG相同，共四份差0。強制只返回原版會依序差8959／8959／35260／35661像素。這些正對照沿舊美術與舊原型，不外推現行31筆＋MAZE B v9全場景，未接受特效造型或混色。

首跑獨立檢查器v1誤以中途step查邊界profile，line72失敗；v2按精確frame SHA修正後，在line79誤將戰鬥區模型當全屏。回查profile domain與原獨立v4 scope，兩份戰鬥區差0，區外490／477有舊未建模輸出；v3按既有domain檢查後同資料乾淨通過。兩個失敗來源與failure JSON保留，沒有改原始資料、解碼器、共識、回退或既定像素判準。一次Docker heredoc未加-i只執行空標準輸入，退出0且無收據；加-i後才真正執行，不把退出0當通過。

可重現入口：workplace/ida/hd-ally-recruit-20261004/effect-fallback-prepare-v{1,2}-20261005.py、effect-fallback-v{1,2}-20261005.go／.bin／.json、effect-fallback-adapter-v{1,2}-20261005.json，獨立檢查effect-fallback-independent-v3-20261005.py／.json，失敗沿v{1,2}-failure-20261005.json。Go v2收據SHA 57ec5e40c667cf68fee9a0a0ce80652cd7f7e98f0bf5da9fc21ee42675bd8f5a，Python v3 SHA 60a3f91f939f9e05641153d45057d43ddf6aee3dea67854265e998c3705ec675，所有原圖與frame輸入SHA在Go收據內。

既有image psychicwar-go-ebiten:latest，SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7；Go1.24.13、Python3.11.2、UID/GID1000、1CPU、network none、1024MiB，Go建置／副本探針60秒外層逾時，Python30秒。原版與專案唯讀，研究根及既有Go快取可寫；沒有啟動遊戲、直接改記憶體、PNG修補或寫存檔。實際Go依賴、精確來源保存、檔案擁有權與容器狀態沿effect-fallback-final-audit-v1-20261005.py／.json；effect-fallback-source-manifest-v1-20261005.json與source-snapshot-v1-20261005.tar.gz保存本輪來源及文件，只留本機。

同次將本節與收據掛進CONTEXT、worklist與024 DRAFT，已知正對照研究038 §155及比較圖可由CONTEXT找到。進度寫入effect-fallback-progress-v1-20261005.py，Issue #34全文同步effect-fallback-issue34-{before,body,after}-v1-20261005。正式31筆PBL＋256格MAZE／30PNG、敵人15/360、ALLY2/31及身體來源60張保持；#34仍OPEN、#44未改，未commit／push／發行。

## 157. 電梯正常路線與HD美術草稿（2026-10-05）

confirmed限於原版有界輸入結果、保存frame來源比對及生成輸出檔案。ROOM0 #5的正常貼圖身份、位置、構圖忠實度、文字區、HD GUI與公開權利均未驗。正式素材與程式未變，024 §1.25僅DRAFT。

既有13-healed.state SHA d9c18d64a60ed678d34d72f00c199af51108c2a36931fdcaed650a110f23e7a0，起點Samar (9,2)朝南、HP40、能量30。v1依序down/up/right/up/up/right/up，hold起點345500000、間隔6000000、每鍵3000000指令，typematic=false；終點(9,1)朝南。v2由該實際state接續right/up/up/right/up，hold397500000起、同間隔及長度；終點(7,1)朝北。IRQ1分別14／10，佇列0。兩終點原版均提示不能前進，未抵原先假設的(11,2)或(7,0)，停止此探索。seed來自不變保存狀態，未重設seed、HP、角色、位置或RAM；沒有獨立機器控制對照，不稱同狀態HD驗收。

v1保存396999999、最終畫格397000000；v2保存436999999、最終畫格437000000。兩保存SHA為8137ebe0773737c7bb1b697de1234c94fed68176af8dceac44e297626bf7fc2f及ab5180b8b3686c563089a7bb6c0bc506c6cb63330e33ae51b1f6008735281143。26份PBL嚴格RLE來源庫537圖，比對視野(4,124,72,72)共16畫格，只有v1 step380000000命中既有ROOM0 #22；不是新ROOM0 #5。來源／鍵序／frame SHA／log見同研究根elevator-route-v{1,2}-20261005.py／.json／.log／.frame／.png／.state，逐步副檔同前綴。零步讀state工具沿既有pionn-maze-state-read-v1.go／.bin；原probe SHA cd07e8789ceaaed785264d5289e21174368eb76a1aaae7125a275918acc043a6。

原ROOM0.PBL SHA 2b2f58c9b716a54bf826dbc9c90237a32458fe49abab52869d5353bbff34d111，31張；#5為72×72。既有workplace/hd/ref/ROOM0-05.png及art-in/ROOM0-05.png保持。參照圖以ImageMagick point整幅放大三倍至elevator-reference-3x-v1-20261005.png，只供生成分析。內建image_gen兩次生成原生1254×1254。v1保留原版參照及舊候選風格，卻在頂部增加三黑色燈槽；v2以v1與原版參照，只要求移除該額外構件。兩份精確prompt與原始工具路徑保存workplace/hd/redraw/ROOM0-05-prompts-20261005.json。原生檔各自保存ROOM0-05-v{1,2}-native-20261005.png；ImageMagick Lanczos完整正方形轉216×216 PNG32至ROOM0-05-v{1,2}-20261005.png，沒有crop、shift或像素修補。

輸出RGBA216×216、不透明度均255。文字矩形(24,33,111,15)的max(R,G,B)<80計數v1=0、v2=1，粗略洋紅計數均0；這個閾值不辨識燈槽／構圖，不能當原版字區或招牌覆繪合格。已檢視原版／舊候選／v2比較圖elevator-art-comparison-v1-20261005.png；控制台、右側直槽與下方燈管仍待完整原版逐區核對，兩稿均DRAFT_NOT_ACCEPTED。原生／成品／比較SHA在redraw/ROOM0-05-art-review-v1-20261005.json；v2成品SHA399a9c26aaca1a78d20968a3e13e5b70d8a9ffc5eefde3c13b9202368ebdc83a。

Docker沿既有psychicwar-go-ebiten:latest SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7，UID/GID1000、1CPU、network none、pids有界、--rm。原版與專案唯讀，僅研究根／redraw可寫；主機不執行分析。Fontconfig快取警告不影響比較PNG產出；沒有量測音訊、FPS或遊戲牆上時序。後續程序與保全核對沿elevator-final-audit-v2-20261005.py／.json，來源索引elevator-source-manifest-v1-20261005.json。原生稿僅留本機，不加入Git或發行包。

文件職責依knowledge-base/local/project-document-responsibilities.md。已知正對照研究038 §156確有CONTEXT入口；本節與素材／路線同次掛入CONTEXT、024 DRAFT及worklist。進度寫入elevator-progress-v1-20261005.py，Issue #34全文沿elevator-issue34-{before,body,after}-v1-20261005。主題31筆PBL＋256格MAZE／30PNG、敵人15/360、ALLY2/31、60身體來源維持；完整HD未完成。#34保持OPEN，#44不修改，未commit／push／發行。

收尾驗證器v1直接匯出8-bit RGBA，未經原製程PNG32寫出再讀回；第一稿57724個通道值相差1，屬編碼量化的比較方式差異。用相同PNG32製程在容器/tmp整幅重生，第一稿像素差0。v2依原製程核對兩稿，原生與既有成品均不修改；v1腳本及failure JSON保留，不降低構圖或字區標準。

## 158. 電梯另一入口與正常Minton戰鬥（2026-10-05）

confirmed限於原版有界正常輸入、逐段保存的原始終點／frame及既有嚴格PBL來源庫。舊通路與房間注入紀錄僅為線索；未新增ROOM0 #5正常來源、HD素材或READY規格。

本輪按復古遊戲路由載入sources/claude/retro-cht/retro-game-playtest.md，回查研究011 §3.4、013 §3及workplace/scan2/run.sh／all.txt。scan2只驗走廊，房間最後一步未涵蓋；較早scan/cell.sh及explore/probe-cell.sh直接改座標／朝向，未重算兩組碰撞座標與前方通行。e27_4_2-f0.log雖開ROOM0.PBL、seek8958，不能由檔名斷言正常入口或貼圖身份。原始紀錄保留，本輪未執行注入腳本、不新解牆位元、不據此改遊戲規則。

從§157真正終點v2接續，v3 left/up/up/up/left/up，437500000起每6000000指令、hold3000000，typematic=false。第一步向西到(6,1)遇Minton，最終481000000仍戰鬥，敵人HP42，玩家HP40／能量30，朝西；後續方向鍵未完成預定路線。這是原版遭遇截住移動，不記為走廊牆壁。v3保存480999999 SHA38bd95df1692f8b37571e5b06f9a3bb02b46b01fd080f3e81d56534f984e49ad。

v4由上述真實戰鬥state，481500000起hold space40000000指令，固定原始seed，不重設任何RAM。531000000畫面已回走廊，原版敵人HP0，玩家HP29／能量8；勝利是已檢視PNG與實際終點，非由內部HD訊號推定。保存530999999 SHA3633578dff994fa812d03ef5471e8057a10ec24ee3446e1b457ba3154a85cf75，仍(6,1)西。

v5由真正勝利state，up/up/left/up，531500000起每6000000、hold3000000。正常西行兩格至(4,1)，朝南最後一步被原版擋住，565000000 PNG顯示You can’t move forward here，未抵(4,2)電梯線索。HP31／能量10為原版步行回復，沒有HD改值。保存564999999 SHAd0f8ab4c34d2a0ab407319cda29b3a438dbd3dd75dd712c6b8856e7f703b24c5。停止這批猜入口，後續需正常可達路線證據，不重試同一起點或改HP／seed。

三段IRQ1各12／2／8，佇列0；8＋4＋6共18份64000像素畫格，原始537圖來源庫的(4,124,72,72)完整房間比對均無命中，未取得ROOM0 #5。每段完整argv、初始state／最終state／seed所在log、原版26份PBL與PW.EXE SHA，以及probe／reader SHA保存elevator-route-v{3,4,5}-20261005.py／.json／.log／.state／.frame／.png；逐步frame／pal同前綴。每段保存state比最終畫格少1指令，不把相同frame當整機同狀態。沒有獨立44欄位機器控制／DOS對照，未驗中文／HD普通GUI。

沿既有psychicwar-go-ebiten:latest SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7，UID/GID1000、1CPU、768MiB、pids64、network none、--rm、95秒外層逾時；原版與專案唯讀，僅既有研究根可寫。Go probe未重新建置，二進位SHA沿§157已核對來源。沒有量測即時音訊或FPS。

本節與證據同次掛入CONTEXT、024 §1.25 DRAFT及worklist；正對照§157與比較圖確在CONTEXT。進度寫入elevator-follow-progress-v1-20261005.py；遠端全文elevator-follow-issue34-{before,body,after}-v1-20261005。保全與擁有權／Docker收尾沿elevator-follow-final-audit-v1-20261005.py／.json及elevator-follow-source-manifest-v1-20261005.json。兩份電梯美術維持DRAFT，現行31筆PBL＋256格MAZE／30PNG、敵人15/360、ALLY2/31及60張身體來源保持。#34 OPEN，#44未改，未commit／push／發行，完整HD未完成。

## 159. HD美術正式定案與Jaxemo三姿勢（2026-10-05）

使用者於2026-10-05 16:58明示接受先前同風格參考並正式定案，要求不再詢問HD美術。確切文字、範圍與來源SHA在同研究根hd-style-decision-v1-20261005.json，寫入來源hd-style-decision-v1-20261005.py。依grilling記錄既有已展示參考與明確授權，無須再次確認。AGENTS §12、024 §1.26、CONTEXT與worklist同步。

沿原版位置、比例、姿勢差異與8×8，人物在後、框線在前。自然曲邊沿既有參考；已展示效果B透光方向採用，效果來源與主題表示仍走技術審查。接受風格不把未證實來源、錯構圖、任意容差、完整GUI／動畫／封包或公開權利升格為confirmed。舊§142／155等當時等待紀錄保留，現況以本定案為準。

本批採ENEMY04-group1-v3-frame-00／01／02-20261005.png，image#3／#4／#5。三份精確SHA為7cabfa0164c287efbd87c803241cb6df8879c599a58e730f2ad95d8f1af7ce32、8faefbdc89d58fcb67f910137d631a3bd6eeb55d8ad3f30e2c005fcde7ca4497及來源審查JSON所列第三份；原72×96不做像素修改。原生／提示沿redraw/ENEMY04-group1-v3-generation-20261005.json與prompt-20261005.json；原版ENEMY04.PBL SHA81cb62cf9a8b64a538d09e50b29cbf8ad123b39b6e86979af6423a2aed20f546。原版來源契約024 §1.19／§1.23 READY，§142已有正常三事件與完整原版對照。

建立本機候選workplace/hd/theme-jaxemo-maze-B-v1-20261005/，保留原31筆／30PNG／256格MAZE，新增ENEMY04 #3–#5三筆，34筆／33PNG。新manifest仍/2，at(32,152)、match(248,0,72,40)，不改正式Go、玩法、DAT或EXE。asset-map.json記完整輸入／輸出SHA；正式選入等同狀態及正常前端結果，舊主題保留。

本節後續來源入口：jaxemo-style-runtime-v1-20261005.go／.bin及runtime收據，jaxemo-style-independent-v1-20261005.py及v1-failure-20261005.json、jaxemo-style-independent-v2-20261005.py／.json，jaxemo-style-gui-v1-20261005.py／.sh與GUI目錄；收尾hd-style-final-audit-v1-20261005.py／.json，Issue全文hd-style-issue34-{before,body,after}-v1-20261005。素材留本機，不commit／push／發行；#44不修改。

本批有限選入完成：六份全960×600圖面由原版PBL、已觀察貼圖事件與批准PNG推導，差0；44欄位／完整DOS與無HD控制相同，CPU／RAM／port負對照有效。Jaxemo三姿勢原座標12／8／8格，省略／錯姿勢／單像素負對照有效；真正載回後再次差分、冷載局部不猜身份、HD關閉／重開通過。原版seed從保存state讀取，未改動或重擲。

獨立檢查器v1的pose4差2880點落在右側ALLY五格，原因是只允許完整戰友圖，漏算已確認起點身份的逐格保留；Jaxemo自身格差0。v2從原版起點完整ALLY與各格原畫吻合推導期望，同一資料重跑六份通過，未改正式Go、PNG或原版state。失敗與修正版來源均保留。

普通Ebiten前端重用現行已建置二進位，347份實際非標準依賴SHA重新核對相同。從正常754000000保存點送Down／Up共四邊緣進入戰鬥，24張視窗中7張可見原座標的Jaxemo三姿勢非黑格；原版nearest與錯姿勢負對照有效。phase02／06已實際檢視，自然退出0。其餘17份不稱通過，不宣稱整屏同狀態、全部動畫、此34筆DAT或音訊。額外入口jaxemo-style-gui-independent-v1-20261005.py／.json及GUI目錄execution.json、record.json、terminal.json。

本機theme-jaxemo-maze-B-v1-20261005/selection-receipt.json已選入，34筆PBL＋256格MAZE／33PNG，敵人18/360、ALLY2/31。原31筆／30PNG逐檔相同，三張新增原72×96位元組不改；原ENEMY04.PBL與全部正式Go保持。完整HD、其餘342敵人／29ALLY、全動畫、效果、封包與公開權利尚未完成。收尾來源hd-style-finalize-v1-20261005.py，來源快照hd-style-source-snapshot-v1-20261005.tar.gz與hd-style-final-audit-v1-20261005.py／.json，遠端同步完整before／body／after保留。

## 160. Gestinti三姿勢美術接入（2026-10-05）

【confirmed，來源及已定案風格限定】目前以§159的34筆／33PNG為正式本機主題。依使用者正式定案，不重問美術；實際檢視ENEMY04-group2-v3-preview及原版reference，採原生1881×836衍生的完整72×96三姿勢。來源、提示、切格及精確SHA沿redraw/ENEMY04-group2-v3-provenance.json，原檔ENEMY04.PBL SHA81cb62cf9a8b64a538d09e50b29cbf8ad123b39b6e86979af6423a2aed20f546。

024 §1.27限定READY，原動作來源及生命周期沿§1.20／§1.23與研究§143。新增#6–#8至theme-gestinti-maze-B-v1-20261005/，候選37筆／36PNG，原34筆與33PNG保持；原版位置、比例、8×8及框線順序不改。asset-map.json記完整輸入與輸出SHA，正式選入依本節實際驗證結果。

工具與收據入口沿既有workplace/ida/hd-ally-recruit-20261004/：gestinti-style-prepare-v1-20261005.py、gestinti-style-runtime-v1-20261005.go／.bin及同名runtime目錄、gestinti-style-independent-v1-20261005.py／.json、gestinti-style-gui-v1-20261005.py／.sh與同名GUI目錄、gestinti-style-gui-independent-v1-20261005.py／.json。收尾gestinti-style-finalize-v1-20261005.py、gestinti-style-final-audit-v1-20261005.py／.json、gestinti-style-source-snapshot-v1-20261005.tar.gz與gestinti-style-issue34-{before,body,after}-v1-20261005。未commit／push／發行，#44不修改，原版／美術／state留本機。

【confirmed，有限本機選入】

Gestinti三姿勢ENEMY04 #6–#8已依使用者定案選入本機，現行主題37筆PBL＋256格MAZE／36PNG，敵人正式美術21/360、ALLY2/31。原34筆與33PNG逐檔相同，新增三姿勢直接採既有v3原圖；原版位置、比例、姿勢差異、8×8及人物／框線前後不改，未改正式Go、EXE、RAM、seed或DAT格式。

十一個正常動作事件6→7→8→7→6→7→8→7→6→7→8，加真正載回、載回接續與冷載共十四份960×600圖面，由原版PBL、正常貼圖事件與批准PNG獨立推導，差0；首三份有效格11／8／8。44機器欄位及完整DOS與無HD控制相同，CPU／RAM／port負對照有效；HD關閉／重開、省略、錯姿勢與單像素負對照有效。

正常中文前端從Rusteck保存點以Left／Up進入Gestinti戰鬥，24張視窗中19張驗到三姿勢的非黑8×8格，原版與錯姿勢負對照有效。phase04／08已實際檢視，自然退出0；其餘5張不列通過。此結果不稱整屏同狀態、全部動畫、37筆主題DAT或封包完成。此前Jaxemo六份圖面及GUI7/24、31筆迷宮DAT GUI9/9保持各自範圍。

原始起點seed8CD8h由保存state讀取，未改seed／HP／位置或重擲。前端347份實際非標準依賴與既有建置收據SHA相同，二進位沿maze-B-frontend-v1-20261005.bin；原生素材及三張新增PNG SHA與原provenance一致，原34筆／33PNG保持。完整HD、其餘339敵人／29ALLY、效果、DAT與封包尚未完成，#34保持OPEN。

## 161. ALLY #2構圖修正與同風格美術（2026-10-05）

沿使用者正式美術定案，不重新詢問風格。現行主題為§160的37筆／36PNG、敵人21/360、ALLY2/31。舊ALLY #2 v8肩向左多5個HD像素、腿部左界由9變16、腰帶由青改紅，不能因風格定案選入。原版24×32來源及隊伍／道具兩位置契約024 §1.18／§1.21已READY，不重做來源逆向。

本輪以原版ALLY-02-v1-source-20261004.png為唯一構圖／配色參照，已實際檢視；採imagegen內建工具重畫v9，要求大頭、短身、左右寬站姿、青腰帶及頂／右／底裁切保持。提示入口workplace/hd/redraw/ALLY-02-v9-prompt-20261005.json，原版ALLY.PBL SHA c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219。

輸出及研究入口沿既有redraw/ALLY-02-v9-{generated,frame,generation,review}-20261005；必要修正版另以v10同前綴保存，不覆寫舊稿。收尾沿workplace/ida/hd-ally-recruit-20261004/ally2-style-art-final-audit-v1-20261005.json與ally2-style-issue34-{before,body,after}-v1-20261005。尺寸／PNG／風格不代替構圖與正常GUI驗收，未通過前不選入。原版、美術與state留本機，未commit／push／發行，#44不修改。

第九稿原生1086×1448 SHA1a6857e34c68877ddd5b7f28d93f9567392b39cf7ff13fae7e4cf3a5f601db75，腰帶紅色未接受。第十稿以內建imagegen修正腰帶，原生同尺寸SHAa6323f60b26e30e19d84b060da2698c5b8806f711806296f361f02aa17aa3cd6；完整72×96與六區bbox沿generation／measurements／review三份JSON。原生及縮圖已實際檢視，提示的「只改腰帶」不當作其他像素沒變證明。沒有像素修補、縮小或平移。

024 §1.28限定READY，候選theme-ally2-maze-B-v1-20261005/新增兩位置，共39筆／37PNG，原37筆／36PNG保持。候選準備入口同研究根ally2-style-prepare-v1-20261005.py，驗證入口ally2-style-runtime-v1-20261005.go／.bin及runtime目錄、ally2-style-independent-v1-20261005.py／.json、ally2-style-gui-v1-20261005.py／.sh與GUI目錄、ally2-style-gui-independent-v1-20261005.py／.json。正式選入及來源保全由本節結果判斷，未把靜態美術接受當作完整HD。

【confirmed，有限本機選入】

ALLY #2第十稿已選入隊伍(232,152)與道具(128,8)，現行主題39筆PBL＋256格MAZE／37PNG，敵人HD21/360、盟友HD3/31。第九稿紅腰帶未接受；第十稿以內建imagegen恢復原版青腰帶，完整1086×1448與提示保存，僅整幅縮成72×96，沒有縮小、平移角色或修補像素。原37筆／36PNG與正式Go、EXE、RAM、seed、DAT格式保持。

六份正常來源接續的兩角色區由原版PBL獨立推導，差0；隊伍共存、Enter顯示道具肖像、再Enter清除、真正載回接續與HD關閉／重開通過。44機器欄位及完整DOS與無HD控制相同，CPU／RAM／port負對照有效；省略、錯圖及單像素負對照有效。範圍為兩個24×32角色區，不稱整圖面對拍。

普通中文GUI由正常Samar保存點以Esc／Down／Down／Enter進See Items，24/24隊伍角色區及8/8道具肖像區相符，撤圖、實際F10／F11與載回後再撤圖通過；自然退出0。items-minton及reload-clear已實際檢視。未宣稱整屏同狀態、全部動畫、現行39筆DAT或封包；前批Gestinti十四圖面／GUI19/24、Jaxemo六圖面／GUI7/24及31筆迷宮DAT GUI9/9保持各自範圍。

工具與精確來源：同研究根ally2-v9-measure-20261005.py、ally2-v10-measure-20261005.py、ally2-style-finalize-v1-20261005.py及ally2-style-art-final-audit-v1-20261005.py／.json。來源快照ally2-style-source-snapshot-v1-20261005.tar.gz。前端347份實際非標準依賴與既有建置來源SHA相同；原生及完整72×96 PNG保存，不公開。39筆主題selection-receipt.json為本次限定選入依據。

## 162. 四圖庫60張敵人美術批次接入與技術驗證（2026-10-05）

【confirmed，限定候選及合成圖面／既有正常保存點回歸】

批次候選新增39張敵人美術，共78筆PBL＋256格MAZE／76PNG，四圖庫60張身體美術均已備齊。原版來源、原生生成圖、完整提示及72×96轉檔雜湊相符，13組原版／HD比較已檢視；原39筆／37PNG保持，未改正式Go、EXE、RAM、seed或DAT格式。

60張由原版PBL與PNG獨立推導的合成來源圖面全部相符，8×8遮格／恢復、錨點、開關／冷載，以及省略與單像素負對照通過。六個既有正常保存點各載入及接續100,000指令，共12圖面與舊主題相同；44機器欄位及完整DOS與無HD控制相同，CPU／RAM／port負對照有效。

本批為候選接入與技術驗證，新增39張尚未正常呈現抽測，不計成完整HD完成。既有正式限定主題仍39筆／37PNG、敵人21/360、ALLY3/31；候選共有敵人美術60/360，其他300張敵人、28張ALLY及效果等仍待完成。現行前端347份實際依賴雜湊保持；沒有新增GUI、DAT、效能或封包完成聲明。

入口沿workplace/ida/hd-ally-recruit-20261004/：prepare-enemy-ready60-v1-20261005.py、enemy-ready60-review-inputs-v1-20261005.json、enemy-ready60-review-page1／page2-20261005.png、13個ENEMY*-batch-comparison-20261005.png。逐圖源檔／解碼色號SHA、原生、提示、固定切格與72×96 SHA由review所列13份redraw/ENEMY*-provenance.json回查，歷史「prototype」不覆寫。已實際檢視兩頁左原版／右HD；不生成新稿、不修補像素、不以相似度猜測原版動作語意。

批次候選：workplace/hd/theme-ready60-maze-B-v1-20261005/manifest.json及asset-map.json。來源契約沿024 §1.22／§1.23，限定批次說明§1.29。舊正式主題theme-ally2-maze-B-v1-20261005/未改，未提升未抽測的新39張正常呈現計數。

獨立fixture：同研究根enemy-ready60-fixtures-v1-20261005/plan.json、render.json、test.log及每來源.frame。這些frame是原版PBL重建的合成測試資料，不寫Oracle記憶體，不冒充正常玩家收據。正式測試apps/psychicwar/theme/ready60_art_test.go使用PSYCHICWAR_READY60_ART_PLAN指向上述plan、PSYCHICWAR_TEST_ORIG指向合法本機原版，PSYCHICWAR_READY60_ART_OUT指定現存可寫收據目錄；Docker內go test ./apps/psychicwar/theme -run '^TestReady60ArtBatch$' -count=1 -v。全部60張期望由Python PBL及PNG獨立推導；78筆候選亦實際LoadTheme通過，60個主子案例無失敗／略過。

正常保存點回歸：同研究根enemy-ready60-normal-v1-20261005.go及enemy-ready60-normal-v1-20261005/runtime.json、六起點各loaded／continued的state、control.state、frame、plane.rgba、machine.json。來源取既有ALLY隊伍、道具、Gestinti、Jaxemo、Kasuruji、Minton正常保存點；兩側seed取同一state，沒有改seed／HP／座標或重擲。每段接續100,000指令，HD新候選與舊正式主題圖面相同，對無HD控制44機器欄位與完整DOS相同。沒有實際GUI或DAT新收據。

收尾與權威：同研究根enemy-ready60-final-audit-v1-20261005.json、enemy-ready60-source-snapshot-v1-20261005.tar.gz、enemy-ready60-issue34-{before,body,after}-v1-20261005；CONTEXT目前狀態表與worklist同步，修正verify.note過時31筆／15敵人及效果等待選擇文字。完整HD仍未完成，#34保持OPEN，#44不改；未commit／push／發行。全部工作沿既有psychicwar-go-ebiten:latest，在有界非root Docker完成，只清理本輪具名pw-hd-ready60與pw-hd-body-review容器。

## 163. ENEMY05五組15張身體美術草稿（2026-10-05）

【confirmed，來源與技術轉檔；美術為DRAFT，原版動作語意及正常呈現未驗】

ENEMY05五組15張身體動作新增HD草稿，逐張原版PBL／PNG參照相符，內建imagegen分組生成並修正兩組。原生、完整提示、固定圖格及整幅72×96保存；ENEMY05 #6–#8中間胸部抖色改平滑紫色，ENEMY05 #12–#14輪廓變平順，原版資訊面板的三種方塊圖案保持。原版與HD比較已檢視，沒有縮小、平移角色或像素修補。

這批只有美術草稿，ENEMY05來源／動作／重疊契約仍DRAFT，尚未接正式Go或主題。既有78筆／76PNG候選及39筆／37PNG正式限定主題保持；候選敵人60/360、正式限定21/360、ALLY3/31數量不變。另15張新ENEMY05草稿待來源及正常呈現，不能加進已驗收數。347份前端實際依賴保持，沒有新的原版戰鬥、GUI、DAT、效能或封包聲明。

原版輸入workplace/original/psychic-war/ENEMY05.PBL，實際30圖，#0–#14各24×32。完整檔案SHA、偏移及逐張色號SHA在workplace/hd/redraw/ENEMY05-reference-proof-v1-20261005.json，既有ref/ENEMY05-00至14.png與PBL獨立RGB逐像素相符。逐張單格及五組原版參照已實際檢視；三格分組只提供同時繪圖布局，不由像素相似推定原版動作或角色名字。

素材與提示入口沿workplace/hd/redraw/：
- ENEMY05-prompts-v1-20261005.json為五個內建imagegen呼叫的完整提示，ENEMY05-prompts-v2-20261005.json為兩個修正呼叫；輸入角色為原版唯一構圖／配色，ENEMY00-group1-generated.png只供賽璐珞線條。
- ENEMY05-generation-outputs-v1／v2-20261005.json記工具原始生成路徑，七份原生已從Codex預設位置複製回專案，舊稿不刪不覆寫。
- ENEMY05-group0–4的reference-native／reference、generated-v1、frame-00／01／02-v1、preview-v1、comparison-v1及provenance-v1；group2與group4另保存v2系列。原生依固定三等分圖格切出，再將每格整幅縮為72×96 PNG32，沒有縮小／平移角色或像素修補。
- ENEMY05-selected-drafts-v1-20261005.json索引五組15張目前草稿；ENEMY05-all-comparison-v1與ENEMY05-all-comparison-selected-v1-20261005.png兩份全組比較已檢視。ENEMY05 #6–#8v1胸部棋盤抖色排除，v2改平滑紫色；ENEMY05 #12–#14v1有像素式階梯，v2輪廓較平順，原版藍白資訊面板圖案保持。提示詞的「只改」不當作其他像素不變證據。

正式程式與兩套既有主題未改，不能將新增草稿與四檔60張READY來源混計。現階段只確認檔案、原版色號參照、尺寸、全不透明及固定轉檔重生；造型／姿勢接受、來源及動作契約、碰撞／遮擋／撤圖、普通HD中文GUI與真正存讀還需驗證。來源契約024 §1.30保持DRAFT；#34不關閉，#44不修改。

收尾與重生入口workplace/ida/hd-ally-recruit-20261004/finalize-enemy05-art-v1-20261005.py、enemy05-art-final-audit-v1-20261005.json、enemy05-art-source-snapshot-v1-20261005.tar.gz、enemy05-art-issue34-{before,body,after}-v1-20261005。Python與ImageMagick都在有界非root psychicwar-go-ebiten:latest內執行，imagegen使用內建模式。起跑load16.94，未作即時效能或音訊結論；原版與候選均留本機，未commit／push／發行。

## 164. ENEMY05完整來源核對、欄位更正與正常載入限制（2026-10-05）

ENEMY05十五張完整來源在391圖全庫中唯一，15張原版PNG與兩套解碼的色號／RGB相符。五組20次PBL差分模型及20個錯差分負對照通過；模型沒有取ENEMY05正常RAM，不能當作原版動作實跑。

修正上一批收據的欄位誤標：end_offset實際存圖塊長度。追加ENEMY05-reference-proof-v2，分開絕對區塊尾端、區塊長度與解碼讀取位置；15筆原始來源及七份歷史生成收據的更正表沿研究038 §164。原圖、HD、提示及舊收據未修改。

一段正常Rusteck按鍵仍在(1,6)，沒有載入ENEMY05；44機器欄位、DOS及負對照通過。唯讀檢查66份既有觀察終點及32份檢查點，這98份中沒有區域5；不稱全專案狀態普查。ENEMY05來源與動作契約仍DRAFT，正式21敵人、3盟友及60敵人候選計數保持，下一步需有證據的正常圖庫切換。

來源與位址基準：原檔workplace/original/psychic-war/ENEMY05.PBL，SHA-256 2b390a5c4a5a2c6e49a9e27b02c89dd839c6c932dad0f568aee6b88e2397180a。30筆偏移表為檔案偏移，不是IDA位址或RAM位址；前15張各24×32。舊v1欄位end_offset全部等於下一筆偏移減當前偏移，誤稱尾端。v2的block_end_exclusive取偏移表下一筆，block_byte_length保留實際長度，decoder_read_end_exclusive取嚴格RLE讀取位置。部分最後literal為送出前一byte會前看下一圖標頭1byte，另記lookahead_bytes；不把前看位置當區塊尾端。15張兩套解碼色號、SHA及單格RGB均相符，原版與HD素材保持。

證據等級：尺寸、原始來源唯一性、參照像素及欄位更正為confirmed。五組20次轉換依既有sub_1435B分支次序，差分由PBL兩張XOR推導，20個錯差分均不符合期望；只證明代數資料相容，不稱ENEMY05載入RAM或正常動作confirmed。原版分支的IDA ea、MZ檔案偏移與CS:IP對照沿§146，這輪未新增IDA分析或改名。來源允許清單、8×8及lifecycle沒有變動。

正常輸入：起點pionn-rusteck-exit-v1-event-observed.state，SHA 990a39e7f186cfb5b497daa9f3c944c323e15578faa5f4bd134f8e8d9bb45e15，區域4、(1,6)、北、HP28、能量6。I_MAP04的(1,5)地點值39只標示電梯，沒有通行證據。660500000步開始以4000000步間距送Up／Down／Enter，絕對終點690000000，角色仍(1,6)、南，沒有開檔。原始seed F95B不修改、不重擲。不可把這段稱為進入電梯或ENEMY05；停止同起點猜鍵。既有pionn-observer-v4.bin SHA 9959142379701f695150fd0d0742adc14c6371e2bacef8eeb838585547d4a0e7，完整保存對照及register負對照通過；獨立比較器再驗44欄位／DOS及CPU／RAM／port負對照。首比較命令引用不存在的研究根bin，test -f先停止，未啟動容器；改用實際workplace/hd/maze-saved-state-independent-v1-20261003.bin，同份原始state比較通過，不更動輸入或判準。

保存點範圍：只讀研究根66份*observed.state與workplace/states/的32份.state，沒有執行或寫回。區域統計0有53、1有10、3有1、4有22、65535有12；最後一類保留原值，不當作有效迷宮。這98份中沒有區域5，不稱全部state或整個遊戲沒有ENEMY05。

工具沿既有Docker psychicwar-go-ebiten:latest，SHA 083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7；Python3.11.2，Go比較器原生產物版本依各既存來源收據。所有容器1CPU、256m至512m、pids32至64、無網路，UID/GID1000；原版唯讀。讀取首次忘加-i，容器沒有收到stdin，沒有輸出或變更；補加-i後正常讀取。首審閱依錯JSON欄位名輸出None，後續按實際ida_ea/raw_bytes/db_disassembly讀取；不把None當原版缺失。

索引：workplace/hd/redraw/ENEMY05-reference-proof-v2-20261005.json；研究根enemy05-source-review-v1-20261005.{py,json}、enemy05-source-progress-v1-20261005.py、enemy05-source-independent-machine-v1-20261005.json、enemy05-saved-state-survey-v1-20261005.json、enemy05-rusteck-elevator-v1-20261005-{inputs,position}.json及run.log、event原版observed／control.state、end.frame／json。收尾沿enemy05-source-final-audit-v1-20261005.json及source-snapshot-v1，遠端全文沿enemy05-source-issue34-{before,body,after}-v1。原版、RAM、state與圖片保留本機；未commit／push／發行，#34保持OPEN，#44未改。

## 165. ENEMY06十五張草稿與圖庫切換查詢邊界（2026-10-05）

【confirmed：來源參照、檔案與固定轉檔。美術風格已定案；原版動作語意、載入條件及正常呈現未知。】

ENEMY06新增五組15張HD身體圖草稿。原版#0–#14各24×32，逐張參照相符；六份內建imagegen原生、完整提示與18份含舊稿72×96均保存。#3–#5修正腿部未到底邊的裁切；#12–#14保留原版各格方塊圖案差異。五組原版／HD比較已檢視，15張固定三格整幅縮圖重生相同，錯誤縮圖負對照能偵測差異。

美術風格沿使用者正式定案，來源／動作／重疊契約仍待技術驗證。ENEMY05與ENEMY06各15張草稿未正式接入，不能加入已驗收數。現行39筆／37PNG主題、78筆／76PNG候選及347份前端實際依賴保持；正式敵人21/360、候選60/360、ALLY3/31。沒有新增正常戰鬥、GUI、DAT、效能或封包完成聲明。

原版圖庫切換只完成窄幅IDA查詢，尚未證實ENEMY05正常載入條件。間接分派區被資料庫標為資料，直接xref及程式旗標不能證明無路徑；跨內嵌表的線性解碼也不當作有效指令。停止重跑等價的Jaxemo失敗分支，來源未知時仍顯示原版。

原版輸入workplace/original/psychic-war/ENEMY06.PBL，實際30張，前15張各24×32。檔案SHA、偏移表界限、解碼讀取位置、15張色號及原版PNG SHA沿workplace/hd/redraw/ENEMY06-reference-proof-v1-20261005.json。容器/out參照路徑在同輪正規化為專案相對路徑，PNG及SHA未變。三格參照與整頁比較實際檢視；固定三等分每格整幅Lanczos轉為72×96 PNG32，不作格內平移、縮小或像素修補。重新轉檔比完整RGBA，PNG日期附加資訊不作像素判準。

素材索引沿workplace/hd/redraw/：ENEMY06-prompts-v1／v2-20261005.json保存六次呼叫完整提示及參照；generation-outputs-v1／v2記錄內建工具原生路徑，六份generated均保存。group0–4的reference-native／reference、generated-v1、frame-00／01／02-v1、preview-v1、comparison-v1及provenance-v1；group1另有v2全系列。selected-drafts-v1索引目前五組15張；all-comparison-v1與all-comparison-selected-v1兩份比較保留。#3–#5v1底邊黑帶與原版不合，v2腿部延續到底邊；提示的「沿第一張」不是其他像素不變證據。#12–#14原版方塊圖案逐格有差異，保留圖案但用途未知。

IDA輸入workplace/ida/PW_UNP.EXE.i64 SHA 4db51a199f5913c52dc4964557b9a4aafb965754d663cb0173c9dfdfe9a00c56；PW_UNP.EXE SHA fd5115b91f014c47a1fb1a6e2ecfd645e293264fe614a4b350e92637cad2fbd9。工具IDA9.4／Python3.12，既有ida-pro-9.4-idapython:locked-v1映像SHA 6f6d59af49d0008c4109a5295b5f374bdc007e2d1ab28cb9de08779584de2780，四次只分析/tmp資料庫複本，原庫SHA保持。IDA載入基址10000h、程式基址10510h；MZ標頭976位元組，檔案偏移＝976＋IDA ea−10000h；執行期CS:IP為0161:(IDA ea−10510h)。三種基準不互換。

confirmed的原始bytes：IDA sub_12B10以AL−80及CS:2612表取得間接分派；IDA ea12B22是該表的線性位置。sub_12A09取表項，sub_12A07以堆疊返回間接呼叫。sub_12A5A取出CS:305B／305D引數堆疊，不能當作變數字碼解析器。圖檔名稱實際小寫，最初大寫搜尋未命中不代表沒有圖庫名稱。DS:AEEA相關直接xref有限，不涵蓋間接寫入，這輪沒有證實區域5來源選擇或正常RAM差分。

v3的Heads依code旗標查IDA ea12CAD–12EB0得到0列，是原庫資料旗標造成；v4用decode_insn非破壞性取246列，不改旗標、函式名或資料庫。v4跨過IDA ea12E78–12E8E內嵌表後的線性解碼失去邊界，不當作有效指令；A7表項的候選入口12E8E須另取邊界才能再研究。db_disassembly仍顯示db資料，不當作decode_insn反組譯。牆位元讀取及旋轉只屬強推論，沒有新增正式迷宮解析器。Rusteck路線仍要經過既有Jaxemo失敗分支，沒有重試等價攻擊／防護盾／F3，也沒有新的正常原版輸入收據。

研究根workplace/ida/hd-ally-recruit-20261004/索引：enemy-bank-routing-ida-v1–v4-20261005.{py,json,log}及enemy-bank-routing-ida-command-v1–v4-20261005.json；prepare-enemy06-art-v1-20261005.py、finalize-enemy06-art-v1-20261005.py、enemy06-art-final-audit-v1-20261005.json、enemy06-art-source-snapshot-v1-20261005.tar.gz與enemy06-art-issue34-{before,body,after}-v1-20261005。347份實際前端依賴及兩套主題115檔PNG／manifest SHA保持，固定轉檔15/15、錯誤縮圖反向對照與私有快照逐檔SHA驗證通過。非root有界Docker，原版唯讀，未作即時音訊／效能。CONTEXT、工作清單與#34同步，#44未改；原版、生成圖及state留本機，未commit／push／發行，完整HD尚未完成。

## 166. 原版C6載入布局與十二圖庫模型（2026-10-05）

已定位原版C6圖庫載入迴圈，180筆指令與原始EXE、ENEMY00／01／03／04四份正常RAM逐bytes相符。原版先解碼第0、2、1張，建立兩份差分；20組正常來源的140段資料共32,320位元組吻合。

十二圖庫共59組完整尺寸模型通過236次動作轉換及236個負對照；180張身體中178張來源唯一，ENEMY02 #12／#13仍為像素別名。ENEMY08 #12–#14只有24×24，原版卻固定複製384位元組，其餘96位元組須正常RAM釐清，排除這組，不猜補。其他八圖庫仍只有原始PBL／靜態模型，未取得正常載入RAM，不能提升正式接入或美術接受計數。

原版F1強力加速砲加Space固定輸入最後進入Game Over，未載入新圖庫。seed F95B保持，44機器欄位／DOS及CPU、RAM、port負對照通過；不調時機重試。這輪先完成載入來源證據，沒有新生成美術。ENEMY05、ENEMY06各15張草稿、現行39筆／37PNG、候選78筆／76PNG及347份前端依賴保持；正式敵人21/360、候選60/360、ALLY3/31。

證據分級：confirmed為分派表原始指標、180筆C6指令三側bytes、兩套PBL解碼、四個已存在正常圖庫RAM的140段資料及檔案SHA。其他八圖庫與236次階段轉換是原版指令／來源模型，不稱正常載入、完整動畫或GUI。全庫391張包含12檔360張敵人及31張ALLY；180張身體中178張唯一，ENEMY02 #12／#13皆對到相同兩來源。ENEMY08短圖三張為唯一來源但尺寸不合普通384bytes模型，整組排除。十二檔僅59組普通尺寸模型，不將它寫成全部60組完成。

位址基準及輸入沿§165：原始PW_UNP.EXE SHA fd5115b91f014c47a1fb1a6e2ecfd645e293264fe614a4b350e92637cad2fbd9；PW_UNP.EXE.i64 SHA 4db51a199f5913c52dc4964557b9a4aafb965754d663cb0173c9dfdfe9a00c56。IDA線性base10000h、CS段base10510h，MZ標頭976位元組；檔案偏移＝976＋ea−10000h，執行期0161:(ea−10510h)。IDA9.4／Python3.12，既有鎖定image SHA 6f6d59af49d0008c4109a5295b5f374bdc007e2d1ab28cb9de08779584de2780。v5–v8只分析/tmp庫複本，原始庫SHA保持。

v5定位sub_14038，證實它讀DS:[AEEAh]後按650h取已載入組，不能把它當開檔入口。v6匯出sub_18884、sub_189BF及sub_18B76；18884由CS:8E47檔名表讀BIN，189BF由CS:915B圖庫表開檔與取圖號偏移。兩表IDA位置19357h／1966Bh，第一個檔名位置給出53／26項邊界，原始前綴byte與大小寫保存。前綴byte只確認被跳過及在失敗開檔分支比較，不推定圖像格式。v7的直接xref只定位部分開場圖，沒有敵人載入端，因間接分派區仍標data；不以缺xref稱無呼叫。

原始byte搜尋只作線索：enemy-bank-load-call-candidates-v1以線性signed near call漏掉CS內16位元回捲，v2依CS IP modulo65536修正，兩份均標非指令證據。v8從原版分派表核對A7／C6／C7入口，decode_insn與canonical mnemonic取正確邊界，不建立指令或改原庫旗標。A7為ALLY相關圖庫呼叫，C7為另一圖庫序列，此輪不擴張該兩類HD。C6是原始opcode C6、IDA ea13000h至13154h、執行期0161:2AF0，180筆與原始EXE及00／01／03／04 RAM各相符。

C6的具體布局：IDA ea1302Eh取BL入DH，1303Ah設五組，13045h取DL作圖號；13048h載#3g入DS組首，1304Dh移+240h，13056h加2再13059h載#3g+2，1306Fh將384bytes存CS:32CAh+180h×g；13083h加1及13086h載#3g+1覆寫DS:+240h。13126h將CS已存#3g+2與DS的#3g+1做XOR，13137h將DS的#3g+1與組首#3g做XOR。這些操作才支持兩份差分，不由鄰圖號或XOR交換性猜補。小圖原始次序#15+3g、#17+3g、#16+3g，DS:+3C0h／+480h／+540h，各寫128bytes，192bytes步距中的其餘64bytes未在本模型聲明。I_ENMY紀錄每組80bytes放+600h，rep movsb先前進50h、後續加600h，使有效步距650h。

四份RAM沿§146–147已有零步原版匯出：body-bank-00／sivad／03／04-v1-20261005.ram，原DS1175h、DS:[AEEAh]＝63C6h。全20組核對組首、01差分、12差分、三小圖及80byte紀錄，140段共32,320bytes完全相符；沒有新原版RAM、座標或種子注入。236次模型各以單bit錯差分為負對照，均與原版來源不同；別名組雖可代數閉合，覆繪唯一身份仍未解。ENEMY08 #12–#14只解碼288bytes，C6固定384bytes複製／差分的剩餘96來源須正常RAM，保留unknown，不清零。

新正常輸入只抽一段F1＋Space：起點rusteck-sprites-southup-v1-20261005-event-draw000.state SHA 2a86f77948d37fc9ac12a831d38c250b45b009be862935baed16ef304f15f78f，Step760710672；760800000開始按住掃描碼3Bh與39h各6000000指令，絕對終點770000000。既有pionn-observer-v4.bin與原版state使用相同seed F95B，不改HP／RAM／隊伍。開END0.IBM與OVER.PBL，終點原版Game Over畫面已實際檢視，沒有新圖庫載入。完整保存state控制與register負對照通過，另獨立44欄位／DOS及CPU、RAM、port負對照通過。不宣稱F1在所有狀態無效，也不依此重試時機。

索引沿workplace/ida/hd-ally-recruit-20261004/：enemy-bank-routing-ida-v5–v8-20261005.{py,json,log}及command收據；enemy-bank-load-call-candidates-v1／v2-20261005.json；enemy-bank-loader-proof-v1-20261005.{py,json}；case-failure-20261005.{py,json}保全首版BIN小寫假定失敗，實際I_ENMYNN.bin查明後只按DOS檔名casefold核對、原名保持。模型用Python3.11.2、既有psychicwar-go-ebiten:latest SHA 083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7，非root、1CPU、有界資源、原版唯讀，起跑load19.51，不作即時效能／音訊結論。

F1收據同根enemy-bank-f1-jaxemo-v1-20261005-event.json、observed／control.state、end.frame、source與draw文件；enemy-bank-f1-jaxemo-independent-v1-20261005.json、original／preview-v1。重生模型沿enemy-bank-loader-proof-v1.py；文件及保全沿enemy-bank-loader-progress-v1-20261005.py、enemy-bank-loader-final-audit-v1-20261005.json、enemy-bank-loader-source-snapshot-v1-20261005.tar.gz與enemy-bank-loader-issue34-{before,body,after}-v1。347依賴、兩套主題115檔及ENEMY05／06已選30單格SHA保持。CONTEXT／工作清單與#34同步，#44不改，未commit／push／發行；完整HD未完成。

## 167. ENEMY07十二張草稿、失敗稿保全與路線限制（2026-10-05）

【confirmed：原版來源、參照、檔案與固定轉檔。美術風格已定案，正常載入／動作／重疊與正式接受未知。】

ENEMY07新增四組12張身體HD草稿。原版#0–#14的24×32參照與來源色號逐張核對；十份內建imagegen原生、四份完整提示及24份72×96含失敗稿已保存。#12–#14四版仍有構圖或風格問題，未選入；不以縮小、平移或像素修補避開原版裁切。12張選用草稿與12張未採用圖的固定轉檔完整RGBA皆能重生，錯誤縮圖反向對照能偵測差異。

風格沿使用者正式定案，不重問。ENEMY05、ENEMY06各15張及ENEMY07這12張仍是本機草稿；正式敵人21/360、候選60/360、ALLY3/31保持。現行39筆／37PNG、候選78筆／76PNG及347份前端依賴未改。來源／正常RAM、動畫、GUI、DAT、效能及封包驗收未擴張。

本輪重用98份既有保存點清單，沒有找到比既有滿HP40／ESP30隊伍更強的已存起點。原版I_MAP04的(1,7)牆值F7只開北側，不能從降落點南側繞過既有Jaxemo分支。未再次執行等價F1、攻擊或防護盾試跑，未調整seed、記憶體或正式遊戲資料。原版C6載入證據沿研究038 §166；236次靜態模型含四正常圖庫80次與其餘八圖庫156次，模型不等於正常載入。

原版輸入workplace/original/psychic-war/ENEMY07.PBL SHA c2924057e1d704d30be7a644c870b0879bb72e155183b6ab7c4c6e864904d519。prepare-enemy07-reference-v1-20261005.py以兩份既有解碼器核對15份原始色號及workplace/hd/ref/ENEMY07-00–14.png的RGB；詳細區塊開始／終點／長度、解碼讀取終點與lookahead分列於ENEMY07-reference-proof-v1-20261005.json，不混用長度與絕對位址。原版每格非黑色外框僅作構圖參照，不能代替正式HD逐像素或動作驗收。

本輪使用內建imagegen，五次v1三格生成、group4兩次v2／v3三格修改、v4三次各單格生成。group4原版上方8列黑、第12張最末列黑、第13／14張被底邊裁切。v1／v3縮小並保留底黑，v2散落圓點，v4像素方塊階梯明顯，四版均未採用。十二張草稿比較已檢視；不宣稱輪廓精確保持或新圖已在玩家畫面顯示。整幅固定轉換由ImageMagick完成，沒有格內圖形變形或手工像素修補。RGBA重生24/24、錯誤point濾鏡反向對照通過。

素材索引在workplace/hd/redraw/：ENEMY07-reference-proof-v1、all-reference-v1、group0–4的reference-native-v1／reference-v1；ENEMY07-12／13／14-reference-enlarged-v4。ENEMY07-prompts-v1–v4保存完整十次提示與參照，generation-outputs-v1及generation-outputs-v2-v4保存工具與原生位置。group0–4 generated-v1／frame-00–02-v1／preview-v1／comparison-v1／provenance-v1，以及group4 v2–v4三版同名系列全部保存；v4原生改用ENEMY07-12／13／14-generated-v4。所有日期後綴為20261005。selected-drafts-v1只列#0–#11，#12–#14列為未採用；selected-comparison-v1及rejected-comparison-v1分別保存四組草稿與四版失敗比較，all-comparison-v1原始首稿不覆寫。

路線檢視重用enemy05-saved-state-survey-v1-20261005.json，並核對既有有效保存點SHA，未取得更強隊伍或新增正常試跑。原版I_MAP04.BIN的(1,7)牆值F7只開北側，南側不可通行；xy交換讀法不符合既有地圖契約，不另寫解析器猜補。C6四正常RAM與十二圖庫靜態模型沿§166，236次包含四正常圖庫的80次模型及其餘八圖庫的156次模型；沒有其餘八圖庫正常RAM證據。本輪不重試既有Game Over分支。

研究根workplace/ida/hd-ally-recruit-20261004/索引：prepare-enemy07-reference-v1-20261005.py、prepare-enemy07-art-v1-20261005.py、prepare-enemy07-rejected-v2-v4-20261005.py、finalize-enemy07-art-v1-20261005.py、enemy07-art-final-audit-v1-20261005.json、enemy07-art-source-snapshot-v1-20261005.tar.gz及enemy07-art-issue34-{before,body,after}-v1-20261005。既有兩主題115份PNG／manifest及347前端實際依賴SHA保持，私有快照逐檔SHA核對。全部分析、轉檔與驗證使用非root有界既有Docker，原版唯讀；未測即時音訊／效能。CONTEXT、工作清單與#34同步，#44未改；未commit／push／發行，完整HD未完成。

## 168. 滿體力濟爾沃隊伍與發射台路徑限制（2026-10-05）

【confirmed限正常保存起點、原版按鍵、機器控制與實際通行旗標。未取得其他八圖庫正常RAM。】

由已正常招募敏頓的薩瑪發射台保存點，兩次Down與Enter抵達濟爾沃area3，HP40／ESP30；新起點SHA 09c6d31f9aa7199ae29a55b84a13e6f161d73671f589181aff67a5c71aeb02f0，步數650000000。Up正常出降落點到(2,1)面南。往西到(1,1)或東到(3,1)皆遭遇ENEMY03 #3葛雷戈林，未攻擊終點HP0。十條正常分支來源SHA及入／返回步數由現有唯讀observer保存，44機器欄位／DOS及CPU、RAM、port負對照均通過；原始seed按各state唯讀核對，不改或重擲。

初次三個F3每事件間距4000000，QueueKey各鍵會排按下與放開，六事件最早末端688900000已超過終點680000000。完整機器控制相同，但事件數閘門失敗，不當作三次完整F3。依既有keyboard.go契約改為每事件400000，在同一保存起點及終點固定重跑，六事件閘門通過，原版仍HP0，沒有脫離成功。保留首版exit1、狀態及預覽，不繼續調時機挑成功結果。鏡像指令錯把名為celtac的探針從濟爾沃出口啟動，只做面南Up，仍area3，明確排除其塞爾塔克證據。

I_MAP03.BIN邏輯(2,1)原始byte C4；直接把高低位元同方向都1解為阻塞的探路假說，預測南側可走、東側阻塞。正常南向前進實際不動且線性16976h為2，左轉面東後16976h為0，正常前進到(3,1)。此假說被實測否定，不進正式解析器。已確認的旗標入口沿研究011，地圖collision座標偏移與編碼仍需獨立證據，不因單點相似推成四向契約。

由真正薩瑪發射台起點四次Down及Enter選Celtac，出現事件但area保持0；送一次Space確認後回到薩瑪降落點(1,15)，開I_MAP00／CODE0／I_MENU00／ENEMY00／I_ENMY00，沒有ENEMY06載入。不能把選項顯示當成抵達新區域。未取得護甲、新圖庫或新HD來源，不重試相同空裝備路徑。

索引在workplace/ida/hd-ally-recruit-20261004/：enemy-bank-zellwal-party-run-v1-20261005.py、enemy-bank-zellwal-step-v1-20261005.py；各分支party／exit／room52／gregrolin-f3／gregrolin-f3-complete／south／eastlook／eaststep／celtac／celtac-launch／celtac-story的event、run、position、preview與independent，原始body來源／draw前後frame／state均按同前綴保存。權威核對入口enemy-bank-normal-route-proof-v1-20261005.{py,json}。首個核對缺/orig掛載、第二次缺/src工作目錄，原始資料未改；同容器補正環境後通過，未當產品缺陷。保存environment-failure-v1，不以exit0代替收據存在。所有state與原版資料只留本機。

## 169. 現行39筆主題真正DAT存讀與九份整屏回歸（2026-10-05）

【confirmed限本正常ALLY1路徑的真正GUI、確定存檔bytes、原始肖像與完整畫面。完整HD與其他sprite未完成。】

現行39筆PBL＋256格MAZE／37PNG主題完成限定正常DAT存讀。真正視窗正常鍵盤保存hd39.dat，再由原版LOAD GAME載回，開道具與切換中英文／HD；九份完整960×600畫面皆與原始資料獨立合成差0，每份保留12相位並有負對照。512位元組DAT與既有無覆繪原版控制相同，52位元組玩家資料及15份原始肖像核對通過。

沿既有31筆通過的DAT操作方式，主題改為現行39筆PBL＋MAZE／37PNG，存檔名改hd39。重用ally1-dat-save-v1/g-after-save-hd-chinese.state及其正式中文字面側檔；讀檔起點ally1-dat-load-prepared-v1/ally1-full-gui-start.state。沒有注入座標、seed、HP、記憶體或原版DAT。既有maze-B-frontend-v1-20261005.bin SHA e60e8078e1a7c3be4e2655cd13c3a90bba2659c111608e46b49590bb3d703fc6，347實際依賴未變，實際旗標-theme指向theme-ally2-maze-B-v1-20261005。

保存三份、載回與道具／切換六份，合計九份960×600完整視窗逐像素差0，每份保留12相位。原始PBL用strict_pbl獨立解碼，迷宮從MAZE.BIN與18×18原始表還原，再依全域8×8對比；中文依正式text JSON與字型重建。F10側檔僅提供實際中文字面，不反填HD或迷宮期望。符合的相位逐份列於verified.json，未排除任何畫面區域。選擇畫面無迷宮，其餘八份迷宮各100格；HD有效畫面的省略迷宮／上方肖像與全部九份一像素反向對照能檢出差異。

hd39.dat實際512位元組，SHA 23ebe0a003baf865c1a907746ebbddc1ba4e70e65e44f06a983b7e5496800ac7，與原版無覆繪save／load／items控制的hd31.dat全bytes相同。保存前後及載回52玩家bytes相同；開道具只改byte38的43→1，與舊同路徑無覆繪控制相同。四道具畫面加原版控制共15份原始ALLY1／ALLY0肖像匹配。26保存與22讀檔原版鍵盤邊緣精確相符，F5／F10／F11等前端鍵沒有外洩。DAT與玩家各一byte副本負對照檢出1。兩個GUI程序按既有quit-after正常退出0，Xvfb trap收尾。

可重跑入口在workplace/ida/hd-ally-recruit-20261004/：hd39-dat-prepare-v1-20261005.py與prepare-inputs記錄舊模板SHA；hd39-dat-gui-v1-20261005.sh於Docker內傳save或load，使用hd39-dat-gui-run-v1-20261005.py及hd39-dat-{save,load}-actions-v1-20261005.json。hd39-dat-{save,load}-v1-20261005/保存execution、terminal、record、completed、commands、原版匯出original-frames、verified、九份state／側檔／玩家／raw／RGB及108份PNG；hd39.dat只在本機saves。零步匯出用既有maze-gui-export-v1-20261005.bin，原碼tools/hd/export_gui_dat.go。獨立完整畫面入口hd39-dat-model-v1-20261005.py、hd39-dat-verify-gui-v1-20261005.py；確定資料入口hd39-dat-byte-proof-v1-20261005.py／.json。收尾hd39-dat-progress-v1-20261005.py、hd39-dat-final-audit-v1-20261005.json、hd39-dat-source-snapshot-v1-20261005.tar.gz及hd39-dat-issue34-{before,body,after}-v1-20261005。

使用既有psychicwar-go-ebiten:latest SHA 083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7，Python3.11.2／Go1.24.13，UID/GID1000、1CPU、network none；GUI1GiB／128pids／210秒外層，匯出與畫面核對512MiB／64pids／60或90秒，原版／Go模組唯讀、研究根可寫。沒有重定義HOME，GUI快取指定XDG_CACHE_HOME／XDG_RUNTIME_DIR到容器/tmp，音訊null僅供視窗，不作音訊／幀率結論。正式39主題與60候選共115檔SHA保持。

收據封存首次遇到相對字型路徑未轉成/src絕對路徑，文件已寫入但封存與最終audit尚未建立。修正封存路徑後用hd39-dat-progress-v1-20261005.py --finalize-only續做，保留原GUI、DAT與依賴收據，不重寫歷史文件。環境失敗記錄同研究根hd39-dat-finalize-environment-failure-v1-20261005.json。

本節補齊現行主題的一條既有正常DAT回歸。原版控制重用舊同起點、確定資料與肖像；未比較自然RNG或全RAM同指令數，未把ALLY1證據外推ALLY2、新敵人或效果中途存讀。正式敵人21、候選60、ALLY3保持，完整sprite、動畫、效能、權利、封包與真機驗收待完成。CONTEXT與工作清單掛入本節與§168，同步#34 OPEN，#44不改；未commit／push／發行。

## 170. ALLY #3–#5新肖像草稿與幾何限制（2026-10-05）

【confirmed限原版靜態像素、原生保全與固定轉檔。原版角色用途、正常RAM與新肖像呈現未驗。】

新增ALLY #3–#5三張本機HD肖像草稿。兩份獨立解碼與原版24×32像素參照核對相同，七份1086×1448原生、完整提示及72×96縮圖保存；固定RGBA7/7與錯誤濾鏡／一byte負對照通過。#3第二稿修正靴子配色，手腳範圍仍待修；#4第二稿恢復黑色眼部空隙；#5第三稿左臂仍高2HD像素，三版未採用。沒有放寬原版位置、比例或裁切要求。正式敵人21/360、候選60/360、ALLY3/31及兩套主題115檔保持，正常來源與接入尚未完成。

原始ALLY.PBL SHA c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219，共31張。#3／#4／#5皆24×32，PBL檔案偏移分別1191／1564／1931，下一區塊邊界1564／1931／2300；strict_pbl讀到1565／1932／2301，各有1 byte lookahead，邊界與讀取端不混用。strict_pbl與pbl.decode的色號陣列、既有ref RGB及576×768最近鄰參照一致。每列原版非黑範圍、色號數與SHA保存在reference-proof，不用反編譯導覽名推用途。

內建imagegen產生七份原生：#3兩版、#4兩版、#5三版，均1086×1448且全幅3:4。每次參照角色分別記錄，原版是幾何、配色與姿勢來源，ALLY-02-v10只作定案風格，不借人物造型。#3首稿靴子錯混入紅色，第二稿修成白灰青。#4首稿在黑色機械眼部新增白瞳孔，第二稿恢復黑色。#5的首次左邊界非黑原生列463／497／558，縮圖以RGB最大值>48判定首次HD列30／33／37，原版y13對應39；第三稿雖下降仍高2HD像素，三稿全未採用。#3左手末端與靴子範圍仍有差異，未將修色當成完整構圖完成。#4還需正常來源與幾何審查。原生及原版／縮圖並排已實際檢視，不因提示說只改局部便宣稱其餘像素不變。

七份72×96完整RGBA皆以原生整幅Lanczos、8位元RGBA重生相同，各6912像素全不透明。七份換成point濾鏡的錯誤控制及一byte副本負對照皆能檢出差異。PNG沒有像素修補、裁切後重排或移動角色。分段bbox與閾值只作診斷，不制定容差；這些數據不證明原版幾何或玩家畫面已通過。兩套主題115檔SHA與之前核對保持。

檔案索引：workplace/hd/redraw/ALLY-03-05-reference-proof-v1-20261005.json，ALLY-03／04／05-source-v1-20261005.png；ALLY-03-05-prompts-v1／v2-20261005.json、generation-jobs-v1／v2、ALLY-05-prompt-v3與generation-job-v3；七份ALLY-03／04／05-v1／v2以及ALLY-05-v3-generated／frame／preview-20261005.png；三份ALLY-03／04／05-comparison-v1-20261005.png；ALLY-03-05-verification-v1-20261005.json及review-v1。初次ImageMagick輸出4位元索引PNG，pbl.read_png只接受8位元而拒讀，改用PNG24與depth8後同資料重跑RGB差0，初稿source-indexed-initial-v1保留；此為參照工具格式，不是遊戲缺陷。比較圖尚在生成時的一次查看查無檔，等待原工具句柄正常退出0後再讀，沒有重啟生成工作。

可重生與封存入口在workplace/ida/hd-ally-recruit-20261004/：prepare-ally3-5-reference-v1-20261005.py、verify-ally3-5-art-v1-20261005.py、finalize-ally3-5-art-v1-20261005.py。核對入口需要原生預設保存資料夾唯讀掛到/generated，對照generation-jobs中的basename；只讀原生、不搬移或刪除預設檔，專案已有逐SHA相同副本。容器沿psychicwar-go-ebiten:latest，Python3.11.2與ImageMagick6.9.11-60，UID/GID1000，1CPU、512MiB、64pids、network none、60或90秒外層；原版與預設原生唯讀，redraw輸出可寫。imagegen本身使用內建工具，不使用CLI/API。保全與遠端全文沿ally3-5-art-final-audit-v1-20261005.json、ally3-5-art-source-snapshot-v1-20261005.tar.gz、ally3-5-issue34-{before,body,after}-v1-20261005。

024 §1.35保持DRAFT；原版位置、比例、裁切與8×8要求不改，正式ALLY3/31、敵人21/360及候選60/360不增加。尚未核對ALLY3／4／5正常來源、圖層、清除、GUI或DAT；沒有正式程式／主題變更。這三張不算完整HD完成，其他25張盟友與其餘sprite繼續。素材及原版只在本機，#34保持OPEN，#44不改；未commit／push／發行。

## 171. 滿資源隊伍的正常Sivad與Samar接續（2026-10-05）

【confirmed限十三條固定state、原版按鍵、兩份已知完整身體來源與不干擾控制。未取得新敵人圖庫或房間圖號，未驗新HD／GUI。】

十三條正常按鍵分支的44機器欄位、完整DOS及CPU／RAM／port負對照通過。滿資源敏頓隊伍在Sivad遇ENEMY01 #6後仍HP0、敵HP69，停止該固定分支。另一條Samar正常戰勝Shulosu，保留HP40／ESP29的(7,14)西向起點，及HP40／ESP30的(6,15)南向接收裝置畫面。這批只取得已知敵人ENEMY00 #3與ENEMY01 #6，沒有新增39張候選的正常來源。ROOM0兩次開檔不證明圖號；身體入口0161:8705的房間觀察仍無捕獲，停止等價重試。

起點為前批正常取得的jaxemo-attack-party-launch-v1-20261005-event-observed.state，SHA 01f1c95928b3f037af6c17820ff2e27771ad2e4b6d2d9de582587c72ffea33ef，590,000,000指令、HP40／ESP30與敏頓隊伍，seed8CD8。Sivad依arrival、exit、encounter、attack四段；Samar依samar-exit、samar-port-exit、samar-west、samar-west-next、samar-shulosu-attack、samar-west-three、samar-west-six、samar-transport、samar-transport-examine九段。每段argv、起點SHA、按鍵／hold、seed與完整終點在相應run-v1及event.json；未注入座標、HP、RAM、seed或反覆重擲。兩側從相同正常起點建立分支，後續seed按原版推進，不要求不同分支骰序相同。

Sivad初次身體呼叫返回660,717,782指令，原始packed RAM唯一對應ENEMY01.PBL #6。從該返回state、660,800,000起hold space25,000,000，到692,000,000，終點HP0／ESP7、敵HP69。Samar的ENEMY00.PBL #3返回620,715,729；後續正常space從640,500,000持續15,000,000，到662,000,000，玩家HP40／ESP23、敵HP0。兩份來源由strict_pbl獨立解碼，391張全來源唯一，完整64,000色號的before加copy與after相同；各單像素負對照差1。此為兩張已有來源的正常重播，不算新增敵人素材。

Samar六步西向終點715,000,000指令：(7,14)、朝西、HP40／ESP29，state SHA b308871d813507e2a197474af8aac018ca194afef3e335d3959b476d67d2eab9。接續Up／Left／Up，key-at715,500,000、every3,000,000，到741,000,000，為(6,15)南向、HP40／ESP30，SHA a7f4850657e07371b0dac302d58bdeb8d053a731241ddd5dc6ea870caf079d22。正常Enter到751,000,000回報接收裝置，SHA e49ce6cdf4eb24c860ec5a76e27487a25ce24f7135108a9789e6af40676c4029，沒有取得傳送或新圖庫證據。這兩份是可接續的正常起點，不能把地圖20h走廊讀成十進位22或用未證實牆位元猜路。

研究房間觀察器v1將31張ROOM0加入既有391張參照，全來源422張，但仍有24×32身體尺寸篩選。v2修正為同時收72×72尺寸，以同一起點及三鍵乾淨重跑，仍無房間貼圖捕獲。兩版各與自己的無觀察控制及舊身體觀察終點核對44欄位／DOS及負對照相同，均只看到ROOM0.PBL兩次開檔。沒有據此推定房間缺圖、特定圖號、位置或源路徑已證實；0161:8705是已確認身體入口，房間需沿既有房間契約另查。兩份假設位置／遮罩的靜態掃描不足辨識ROOM圖號，未進正式規格。

全部位址為原版runtime CS:IP、DS:BX／SS:SP，非IDA ea；原版尺寸320×200，frame為64,000色號。診斷preview使用預設EGA色表，只供辨認場景，不宣稱當時AC色盤或RGB同狀態。身體控制不取代HD動畫、GUI或DAT驗收。正式主題115檔與前端實際347份依賴SHA保持。原始PW.EXE、ALLY、ROOM0及12敵人PBL與研究來源在本機archive，未公開。

索引在workplace/ida/hd-ally-recruit-20261004/：candidate-sivad-route-v1-20261005.py、candidate-sivad-{arrival,exit,encounter,attack,samar-*}-run-v1及event各輸出；candidate-sivad-proof-v1／v2-20261005.json、各machine-v1；rusteck-sprites-observer／build／overlay-v1；pionn-maze-state-read-v1.bin；prepare-transport-room-observer-v1／v2、transport-room-observer／overlay／build／run／source／machine-v1／v2；finalize-candidate-sivad-v1／v2；candidate-sivad-final-audit-v1及source-snapshot-v1。v1校驗十一條，v2補最後兩條，共十三條，不覆寫原收據。

環境失敗分開保留：transport-room-mount-failure-v2缺/orig；補回原版唯讀掛載後同命令重跑。candidate-sivad-finalize-mount-failure-v1缺/gomod；補回鎖版依賴唯讀掛載後v2重跑，保留原腳本與已驗兩份machine收據。沒有調seed或改素材讓它通過。兩個房間建置各64份實際非標準Go來源由go list -deps保存，不用泛搜Go檔取代依賴證據。

工具沿psychicwar-go-ebiten:latest，image SHA 083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7，Go1.24.13／Python3.11.2，UID/GID1000、1CPU、512MiB或1GiB、64或128pids、network none與90／120秒外層。原版及Go依賴唯讀，只研究根與既有cache可寫。封存658檔逐檔重讀相同，SHA 30fd0dd9293b3c22c0c0e5dfa48d4c6746aa91d2f8da82b193eb0563fccc149f。未新增READY範圍、正式程式、美術或完成數；#34保持OPEN，未commit／push／發行。

## 172. ALLY #6–#8三張草稿與原版構圖審查（2026-10-05）

【confirmed限靜態原版像素、原生保全與固定縮圖。新肖像的正常來源、角色用途、構圖及執行期未完成。】

新增ALLY #6–#8三張本機HD草稿。原版兩份解碼與放大參照相同，三份1086×1448原生、完整提示及72×96整幅縮圖保存；完整RGBA3/3、錯誤濾鏡與一byte負對照通過。#6手套與腿部仍待對位審查，#7、#8左手較原版偏高，未選入正式主題。#3–#5前批草稿與已列構圖差異保留。正式敵人21/360、候選60/360、ALLY3/31不增加。

原始ALLY.PBL SHA c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219，共31張。#6／#7／#8皆24×32，檔案偏移2300／2664／3061，下一區塊邊界2664／3061／3420；strict_pbl讀到2665／3062／3421，各1 byte lookahead。strict_pbl與pbl.decode、原版PNG完整RGB及576×768最近鄰參照相同，每列非黑範圍與原始色號SHA保存在reference-proof，沒有推測未驗角色名稱或用途。

三次內建imagegen分別用各自原版作幾何／配色來源，ALLY-02-v10只供已定案風格，未借藍髮或人物設計。#6白色雙角、黑色機械眼孔與紅面罩；#7紅紫頭部、黃黑面孔與藍白黃裝甲；#8青白頭部與原版上舉雙臂。三份原生1086×1448完整保留並實際檢視；固定整幅Lanczos至72×96，不裁切重排、不縮小或平移人物、不修補像素。三張皆6912像素不透明，完整RGBA差0；point錯濾鏡分別13137／13447／12340 byte不同，一byte副本負對照各差1。

原版／縮圖比較已實際檢視。#6手套及腿部輪廓仍待對位審查，#7、#8左手較原版偏高，草稿保持未採用。非黑閾值與分段bbox只診斷，不定容差，也不因同尺寸便宣稱構圖或正常呈現通過。前批#3手腳與#5左臂差異沿§170，不因本批新增草稿結案。其餘22張ALLY本輪未新增草稿，完整ALLY尚未完成。

素材索引workplace/hd/redraw/：ALLY-06-08-reference-proof-v1、prompts-v1、generation-jobs-v1、verification-v1、review-v1-20261005.json；ALLY-06／07／08-source-v1與v1-generated／frame／preview-20261005.png，三份comparison-v1。完整提示逐張保存，原生預設路徑在generation-jobs，預設檔唯讀、不刪除，專案副本SHA相同。重生入口同研究根prepare-ally6-8-reference-v1-20261005.py、verify-ally6-8-art-v1-20261005.py、finalize-ally6-8-art-v1-20261005.py；收尾ally6-8-art-final-audit-v1與source-snapshot-v1，27檔逐檔重讀相同，archive SHA 145b06ff19587183a448ab14ead6afe2afbae138d818977df6606c8f1b7150fd。

工具沿既有Docker image、Python3.11.2與ImageMagick6.9.11-60，UID/GID1000、1CPU、256或512MiB、32或64pids、network none、60或90秒外層；原版及預設生成資料夾唯讀，redraw與研究根可寫。全部PNG及原版留本機，115份現行／候選主題保持。024 §1.36 DRAFT只記新草稿與技術閘門；正式ALLY3/31、敵人21/360、候選60/360不增加。未驗招募、來源RAM、兩位置、清除、GUI或DAT，不稱完整HD完成。CONTEXT、worklist及#34同步，#44未改，未commit／push／發行。

文件與遠端同步入口：同研究根update-candidate-ally6-8-progress-v1-20261005.py、candidate-sivad-issue34-before-v1、ally6-8-issue34-prewrite／body／after-v1及ally6-8-progress-final-audit-v1-20261005.json。同步正文由tools/worklist.py的issue_body產生，主機gh回讀後逐字核對；擁有權與本輪Docker清理保存在progress-final-audit。原版與草稿的來源封存收據獨立，不用遠端Issue當像素或正常來源證據。

## 173. 既有F7／F8輔助起點與普通Up的新三姿勢來源（2026-10-06）

【confirmed限實際輔助按鍵、保存state、五個原版來源與完整機器控制。不稱自然戰鬥對拍。】

READY規格014允許既有F7補HP／ESP與F8弱化敵HP；本批沒有新增作弊規則。從完整初始化的670,000,000保存點進入實際前端，F10在670,000,003保存初始HP28／敵HP100；F7於670,666,841、F8於670,976,188。F8實際寫入前敵HP98，寫後1；原版繼續推進，到671,351,485的GUI F10保存時敵HP已0、玩家HP37。GUI v2舊腳本要求採樣時仍HP1而提前終止-15，不算自然退出通過；這份真實state仍完整保存。

研究重播只在實際鍵時点套用既有FullValue與敵HP範圍判斷；其餘原版正常推進。重播與實際GUI保存state的44欄位及完整DOS完全相同，CPU／RAM／port負對照有效。省略F8分支敵HP98，完整機器比較預期拒絕；沒有更換seed、座標、隊伍或增加作弊。

後續唯讀來源觀察只送普通Up：north到676,000,000為Sivad(5,4)北向；north-three到690,000,000為(5,2)北向、HP40／ESP30；north-four於690,500,000送Up，到697,000,000為(5,1)，玩家HP0。三段完整state與各自無觀察控制的44欄位／DOS及負對照相同。未反覆重擲，後續seed由原版推進。

第三段捕獲五個左側身體來源，入口690687279／691320523／692069470／692818359／693567235指令，runtime CS:IP 0161:8705，返回0161:8751；非IDA ea。完整首次copy唯一對應ENEMY01 #0，四次XOR推導0→1→2→1→0，391張獨立來源核對。每次before加原始copy／XOR重建完整64000色號與after差0，單像素負對照差1。後續四張與靜態身體各差72像素，屬原版交錯遮擋，未補畫或定容差。第六個來源在x264的ALLY位置，不當成左側敵人清除證據。

輸入與工具索引在workplace/ida/hd-ally-recruit-20261004/：hd-cheat-source-gui-v1／v2-20261005各原始state、PNG、keylog及terminal；prepare-cheat-source-tools-v2、cheat-source-state-read／build-v1；prepare-assisted-sprites-observer-v1、assisted-sprites-{observer,overlay,build}-v1；assisted-actions-{GUI-v2,omit-F8}-v1、assisted-GUI-v2-{positive,omit-F8}-v1全部輸出、positive-machine-v1、replay-v2；prepare-assisted-diagnostic-v1、assisted-state-diagnostic／build-v1與diagnostic-v1僅作診斷。普通鍵入口sivad-assisted-follow-v1、sivad-assisted-{north,north-three,north-four}-run-v1及event全部輸出、各machine-v1、route-control-v1；來源proof為prepare-sivad-group0-v1／v2／v3及sivad-group0-body-source-v2、source-check-failure-v1、verification-script-failures-v1-20261006。

先前GUI v1從未完成敵人初始化的初次貼圖返回開始，F8被後續原版初始化覆蓋，未勝利；55秒自然退出0只證明程序收尾，未算來源進展。GUI v2已取得勝利state但採樣假設錯誤；省略F8預期失敗曾被包裝器誤列主驗證失敗，診斷44欄位仍相同。上述失敗原檔保留，不覆寫歷史或調原版讓檢查通過。

## 174. ENEMY01 #0–#2本機限定選入與視窗抽測（2026-10-06）

【confirmed限三姿勢來源、八份完整圖面與GUI可見格；不稱全部動畫、自然戰鬥、整屏GUI或交付完成。】

ENEMY01 #0–#2三姿勢已限定選入本機主題。現行42筆PBL＋256格MAZE／40PNG，正式敵人24/360、ALLY3/31；候選總數仍60/360，新增36張候選的實際呈現尚待驗。原39筆與37PNG不變，新增三PNG與既有定案候選相同，未改正式Go或資料格式。

來源起點由實際中文前端的既有F7／F8取得。按鍵時點重播與GUI保存state的44欄位及完整DOS相同；省略F8負對照拒絕。從輔助起點後只送普通原版Up，取得0→1→2→1→0五個來源，copy／XOR重建完整64000色號差0；不稱自然戰鬥對拍。

五個姿勢加真正載回、載回接續與冷載共八份960×600圖面，原版PBL／來源／批准PNG獨立期望差0，首三份有效格12／8／8。44機器欄位、完整DOS與無HD控制相同，CPU／RAM／port及省略、錯姿勢、單像素負對照有效。HD關閉／重開通過。

實際960×600視窗以普通Up進入這場戰鬥，24張抽樣中11張驗到三姿勢的非黑8×8格，原版與錯姿勢負對照有效，自然退出0；phase01／02／04已直接檢視。其他13張不列HD通過。原版遮擋保留，未放寬位置、比例或8×8。整屏GUI、全部動畫、42筆DAT與封包仍未完成。

本機主題workplace/hd/theme-enemy01-group0-maze-B-v1-20261006/manifest.json、asset-map.json與selection-receipt.json；先前候選／父asset-map另保存在candidate-selection-receipt.json及parent-asset-map.json。READY §1.29已允許這三個來源，§1.37只記限定選入與證據範圍；沒有正式Go、EXE、DAT或覆繪格規則變更。迷宮圖集、ROOM22與原版美女／框線前後保持。

重生與保全索引在同研究根：sivad-group0-style-runtime-v1-20261006.go／bin與build-v1、runtime-v1目錄的八份state／control／frame／RGB／RGBA及machine；sivad-group0-style-independent-v1.py／json；sivad-group0-style-gui-v1.py／sh與目錄24份PNG、execution、record、terminal；gui-independent-v1.py／json；sivad-group0-controls-v2、finalize-sivad-group0-v1、preserve-sivad-group0-v1、final-audit-v1及source-snapshot-v1。實際Go依賴由go list -deps記錄，不以泛搜替代；前端沿347份實際依賴及既有maze-B-frontend-v1-20261005.bin，沒有另建工具image。

比較圖montage的stderr回報Fontconfig無可寫cache，程序收到SIGABRT；根因未定位，圖片未完成且未採用；三份原始PNG已直接檢視，不需重跑像素驗證。三段控制已完成，不因製圖環境失敗重測。工具沿psychicwar-go-ebiten:latest，image SHA 083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7，Go1.24.13／Python3.11.2；UID/GID1000、1CPU、256MiB–1GiB、32–128pids、network none、60／120秒外層。原版與鎖版依賴唯讀，只研究根、主題及既有cache可寫；有界Xvfb有trap。擁有權、archive逐SHA核對與Docker清理沿final-audit。

CONTEXT與worklist使用唯一現況表，Issue正文由tools/worklist.py的issue_body產生。遠端#34保持OPEN，全文索引sivad-group0-issue34-{before,body,after}-v1-20261006；#44未修改。原版、PNG、state、DAT與archive只留本機，未commit／push／發行。全部HD目標持續，剩餘敵人336張、ALLY28張、小圖塊與效果待完成。

§174勘誤：製圖失敗只證實Fontconfig訊息與SIGABRT同次發生，未定位因果。正文改為觀察結果，保留封存中的舊說法與失敗腳本。來源封存727檔／939次輸入SHA核對通過，archive SHA 254a09bd4cc555d2db160b63e1e0b846ca14c35e7ea36da5f08050f9e1e1de03；此句晚於archive建立，不回寫原封存。進度與Issue未受此製圖失敗影響。收尾入口同研究根sivad-group0-close-audit-v1-20261006.json。

## 175. 原位電梯來源、中文字幕材質與限定接入（2026-10-06）

**等級：confirmed，限已保存的原版來源與普通鍵接續。** 起點由實際既有READY014 F7／F8取得，不稱自然戰鬥。美術依使用者正式定案，不另問外觀。

| 輸入／工具 | 定位與結果 |
|---|---|
| PW.EXE | SHA88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49，原版不改 |
| ROOM0.PBL #5 | SHA2b2f58c9b716a54bf826dbc9c90237a32458fe49abab52869d5353bbff34d111，31圖，偏移8958，72×72 |
| 執行期原始定位 | CS:IP 0161:8588，DS:BX 0161:92BE；非IDA線性位址。718515137進入，718913424返回 |
| 原版貼圖 | at(4,124)，完整64000色號重建差0，來源唯一吻合全部537張中的ROOM0 #5；外部44機器欄位與DOS相同 |
| 下／上層與撤圖 | (9,2)與(9,10)原始圖各5184像素差0；離開後差3435，原圖移除。新(7,9)差3110，圖號未知 |
| 工具鏈 | Go1.24.13、Python3.11.2、ImageMagick6.9.11-60；psychicwar-go-ebiten:latest SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7 |
| 新前端 | SHA04cadc80fe62cb29d62ab907e64865159ea5c593e3352ac64ca423db59078ba0，347實際Go／C／標頭／組語依賴已保存；runtime94、source-export91依賴沿actual-inputs收據 |

### 175.1 來源與普通鍵

實際前端保存的獲勝起點經精確輔助按鍵重播，44欄位及DOS相同；省略F8負對照失敗。後續十四普通鍵分支同控制相同，保留一條再次遇已知Machechi後HP0的分支。其餘沒有再注入HP、座標、隊伍或seed，也未重擲。

ROOM5來源讀1972 bytes，原圖塊大小1971，末byte是下一圖的未使用lookahead，完整來源仍與原檔slice相同。改該byte不影響解碼；不放寬來源匹配或補像素。原版前屏加RLE貼圖的全64000期望差0；偏移負對照2249、錯ROOM6 3516、一像素1。入口elevator-source-proof-v2及elevator-route-controls-proof-v3-20261006.json。電梯上層西向到(7,9)的外觀不能當圖號證據。

### 175.2 美術與背景契約

ROOM0-05-v3／v4／v5完整原生、提示及216×216保留於workplace/hd/redraw/。v3有新增頂部燈槽未採用；v4下方紫管被拆段，v5修成原版三支連續紫管。全稿只整幅Lanczos縮圖，無裁切、平移或像素修補。v5完整PNG轉檔重生差0，Point濾鏡負對照41027像素、一byte負對照1。

text/baked.json原電梯七個6×7字格、cjk16、FG／BG與位置不改。中文字矩形在HD小圖[24,30,126,21]，166字模及2480背景像素由字型獨立核對。平色背景在HD上形成貼片；可丟棄合成原型只保留字模、採HD背景，期望差0。原型前後44機器欄位與DOS相同，GOB位元串不同不能當狀態差異。

證據審查後024 §1.38、dosgolem 204-art-plane §5先標READY才實作。通用Layer.DrawWithBackground只接受指定Shown文字與同位置alpha255圖面，其他像素回原Stamp.BG；原Draw、字模、透明格、狀態及快照不變。遊戲只授權已載ROOM5 kind=redraw鍵，每幀清圖面，沒有新JSON、save或玩法。原版source／PNG／state仍本機。ROOM5無字HD招牌在英文模式尚未補ELEVATOR，故接受範圍限中文HD。

### 175.3 限定驗證與現行主題

九份冷載下／上層、撤圖、新未知房間、普通Up→Go up→Leave及真正載回，完整960×600圖面及字面由PBL、MAZE、字型、批准PNG獨立推導，差0。每份44欄位／完整DOS、CPU／RAM／port負對照通過；五份完整ROOM5、HD關／開、撤圖、省略及一像素負對照有效。入口elevator-runtime-independent-v1-20261006.json。

真正視窗用普通鍵上下電梯，Shift+F5切HD、F5切語言、F10／F11保存載回，正常退出0。十二份中11份216×192可見區差0，含八份ROOM5；原版72×64範圍避開前端底部保存提示，底部8列另由九份完整圖面核對。過場leave-phase0未列通過。英文只核對中文字幕停畫；不聲明英文招牌、整屏GUI、43筆DAT、所有動畫或封包完成。入口elevator-gui-independent-v1與elevator-gui-v5-20261006/terminal.json，upper-phase1才有原版0B30往下新選單。

現行theme-elevator-maze-B-v1-20261006為43筆PBL＋256格MAZE／41PNG，限定中文電梯已選入；父42筆／40PNG不改。敵人24/360、ALLY3/31、候選60不增加。完整HD與#34保持未完成。

### 175.4 失敗分類與保全入口

研究驗證器的1971邊界假設、4-bit參照PNG限制、直接RGBA與PNG編碼量化不同、GOB逐byte比較、把(7,9)外觀猜為ROOM5、stdin的__file__、遺漏/hd測試掛載及歷史ResearchWrapMachine編譯入口，各保留舊稿與更正收據。未放寬原版／像素判準，沒有據此判為產品缺陷。

GUI v1缺/gomod尚未開遊戲，v2在F10保存完成前複製，v3區域變數遮蔽，v4在原版上樓過場未完成時送Leave，各保留。v5等待新quick.json及原版0B30 Shown，再操作離開，12份自然退出；不以等待秒數代替完成狀態。

重跑入口同研究根elevator-runtime-v2-20261006.go、elevator-runtime-independent-v1-20261006.py、elevator-gui-v5-20261006.sh及elevator-gui-independent-v1-20261006.py。實際建置入口沿elevator-runtime-execution-v2、elevator-frontend-build-v1及elevator-frontend-v2／runtime-v2／source-export-v1-actual-inputs-20261006.json。原版逐檔SHA沿來源proof；新原生、所有提示、失敗稿、來源state、控制及程式保全於elevator-source-snapshot-v1-20261006.tar.gz與elevator-final-audit-v1-20261006.json，全部本機。CONTEXT與唯一worklist及#34同步，不改#44，未commit／push／發行。

## 176. 原位電梯英文標籤（2026-10-06）

來源等級confirmed，限ROOM0 #5。獨立tools/pbl.py核對原檔SHA2b2f58c9b716a54bf826dbc9c90237a32458fe49abab52869d5353bbff34d111、偏移8958；來源[8,11,37,5]含76個色號12前景、109個色號11背景，字模逐列保存於同研究根elevator-english-source-v1-20261006.json。原位[12,135,37,5]，3倍684前景像素。兩份普通鍵電梯上下層全72×72與原檔相同，撤圖負對照3435。既有英文模式停中文字後HD招牌無字，需以原版前景保留原文。

024 §1.39先READY才實作；不新增資料格式、不改玩法或美術。本節的限定新前端驗證結果見下表。來源入口elevator-english-source-v1-20261006.py及JSON；完整HD仍未完成。

前批中文保全入口preserve-elevator-v1-20261006.py，elevator-source-snapshot-v1-20261006.tar.gz保存1220份、838輸入雜湊，SHA52eab90447ed0071281c68d428dab642272b029c513edcdb9aa6e44115091f3a。原腳本gzip隨機取檔逐份回捲，外層命令124；逐檔驗證及audit已完成後容器才退出。另用依封存順序串流重驗1220份通過，不稱首次終端0或原腳本足夠快。當時英文實作尚未開始，中文前端347、runtime94及來源匯出91實際依賴已完整保存。第五稿正式選用紀錄另存workplace/hd/redraw/ROOM0-05-v5-selection-review-20261006.json，生成時DRAFT收據保留。

### 176.1 實作與限定驗證

| 項目 | 結果與入口 |
|---|---|
| 新正式程式 | theme/labels.go只補原版前景，Load保存已核對ROOM5 source，前端只在英文模式呼叫。通用xlate此階段不改。未新增JSON／save欄位 |
| 新前端 | elevator-english-frontend-v1-20261006.bin，SHA2ee43d77763c89d46c1092a3269774049477acf09c36db08bd0d9034870c63a4；348實際非標準庫來源完整保存，未重寫前批347收據 |
| 完整圖面 | 九份完整960×600 art/text各差0，五份ROOM5各684英文字模像素、英文省略負對照684；一像素1。冷載、普通鍵、載回、撤圖均沿獨立PBL／MAZE／字型／PNG期望 |
| 狀態與邊界 | 畫字前後44欄位與DOS相同，普通鍵同控制相同；錯完整來源、HD關閉、alpha254、全透明皆不補字，原位與材質保持 |
| GUI | 真正視窗17份保存、自然退出0；16份原版72×64 ROI差0，13份ROOM5、6份HD英文各684像素。1份離開過場不列通過；F5、Shift+F5、F10／F11與原版英文HD關閉均核對 |
| 現行主題 | 43筆PBL＋256格MAZE／41PNG，不增敵人24/360、ALLY3/31。中英電梯限定通過，整屏GUI／43筆DAT／全部動畫／完整HD／交付未完成 |

來源及重跑入口同研究根elevator-english-source-v1-20261006.py與JSON、build-elevator-english-v1-20261006.py、elevator-english-runtime-v1-20261006.go、elevator-english-runtime-independent-v1-20261006.py／JSON、elevator-english-gui-v1-20261006.sh及elevator-english-gui-independent-v1-20261006.py／JSON。前端348、runtime95、runtime-export92、GUI-export92依賴與完整overlay沿各-build.json；私有保全elevator-english-source-snapshot-v1-20261006.tar.gz及elevator-english-final-audit-v1-20261006.json。

GUI來源包裝器v1誤用位置參數，實際tools/hd/export_gui_dat.go要求-out目錄，報缺輸出且未生成原版匯出。v2依實際契約修正後對現有17份state匯出，沒有重跑GUI。誤找不存在的私有Go檔亦只屬研究讀取路徑錯誤。入口prepare-elevator-english-gui-proof-v2-20261006.py及elevator-english-gui-export-wrapper-failure-v1-20261006.json。未放寬像素判準或新增產品缺陷。

### 176.2 收尾保全

新中英封存elevator-english-source-snapshot-v1-20261006.tar.gz保存916份、942輸入檢核，SHA4313db6a4d5dc5ad9ccfefbc394261501f95be8020099abfa205f63eb05e93b3。preserve-elevator-english-v1-20261006.py依tar順序一次解壓逐份核對，命令退出0；前批1220份中文封存保持。遠端#34全文讀回一致、保持OPEN，#44未改。根與dosgolem的git diff --check通過；本輪formal文件及輸出UID/GID1000，研究根、重繪及43主題沒有root-owned檔或*.md目錄，Docker本輪容器全部清除。未commit／push／發行。

## 177. ALLY續稿、三張新草稿與普通鍵來源更正（2026-10-06）

HD美術已正式定案，不再詢問取捨。這批只製作草稿、修正已見偏差與研究原版來源；沒有變更正式程式或43筆主題。正式敵人24/360、ALLY3/31維持。024 §1.35與§1.40 DRAFT列明剩餘幾何與來源閘門。

### 177.1 素材與構圖

本機ALLY.PBL SHA c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219，共31張。#9／#10／#11檔案偏移3420／3803／4203，圖塊末端3803／4203／4580；strict_pbl實際讀到3804／4204／4581，包含下一表頭的一byte lookahead。位址基準都是檔案偏移，不能混稱IDA或runtime位址。兩份解碼器逐像素相同，三份576×768最近鄰RGB參照差0，靜態來源為confirmed；正常RAM、招募用途與位置仍未知。

內建imagegen新增#9–#11首稿及#10盔甲第二稿，另保全#3第三稿、#5第四至七稿，共九份本批原生。全部1086×1448，整幅Lanczos到72×96，不裁切、平移或補像素。八組與新增#10第二稿一組的完整RGBA重生差0，各有錯濾鏡及一個位元組負對照；此結果只證明素材流程。Python3.11.2、ImageMagick6.9.11-60在既有psychicwar-go-ebiten:latest隔離容器執行，版本及完整輸入SHA沿下列JSON。#3–#11累計九張草稿、十九份原生，#12–#30十九張尚無草稿。

| 草稿 | 審查結果 | 下一閘門 |
|---|---|---|
| #3第三稿 | 手部仍偏高，左靴最後HD列91、左端9，原版限定92與12 | 修手腳範圍。x<9所得63列只涵蓋左緣，不稱完整左手末端 |
| #5第四至七稿 | 左緣首次非黑HD列37／38／40／34，原版39；第五稿臂段壓縮，第七稿偏高 | 全部未採用，停止同類提示微調，回查原版整段手臂骨架 |
| #9首稿 | 保留機械藍面罩，左緣首次非黑HD列30，原版39，腰部也偏高 | 修構圖與腿部比例 |
| #10首稿／第二稿 | 首稿偏用風格參照角色盔甲；第二稿恢復藍色盔甲、紅色胸部矩形、白腰帶與白青靴筒，去掉桃紅領口及紅膝環 | 配色修正已有實際檢視，頭部與手腳比例、正常來源另驗 |
| #11首稿 | 保留紅髮及抬左手姿勢 | 手部高度、肩線與腿部範圍待精確對位 |

色彩門檻max RGB >48的左緣量測是診斷，不代表完整造型或原版parity。#5第四稿返回句柄查無，記憶store遺失；產物與請求的關聯依時間及實際內容標strong inference，不宣稱精確工具返回或終端0。後續生成的工具返回、原生路徑與完整提示均已保存。所有美術未選入，不以放寬容差接受。

入口在workplace/hd/redraw/：ALLY-09-11-{reference-proof,prompts,generation-jobs}-v1-20261006.json，ALLY-art-{verification,review}-v1-20261006.json，ALLY-10-{generation-job,verification}-v2-20261006.json與ALLY-10-prompt-v2-20261006.txt。ALLY-03-v3、ALLY-05-v4–v7、ALLY-09／10／11-v1與ALLY-10-v2的generated、frame、preview、comparison皆本機保全；#5各稿generation-review與prompt檔保留。重跑入口在既有研究根：prepare-ally9-11-reference-v1-20261006.py、verify-ally-art-v1-20261006.py、verify-ally10-v2-v1-20261006.py。建立產物須新版本檔名，腳本的存在檢查不允許覆寫舊稿。

### 177.2 原版普通鍵與來源更正

起點ally2-minton-name-v1-20261005-event-observed.state SHA5945ca8eb94f73eb7021f3d2bb2fd9283168f98c375d67ef4ba644c26c5c2e43，445000000步、SAMAR(14,1)朝東；既有加入來源沿§141。五段普通方向鍵接續至510000000步、SAMAR(9,1)朝西，HP40／能量30。不注入RAM、HP、座標或seed；seed由ADBF自然進至D3B4，更早13-healed起點限制保留，不稱自然全程。

觀察器rusteck-sprites-observer-v1-20261005.bin SHA ae0e8d2f1d5d3424cf7fefe6d43a4a27ea4f7661dc35deb701ecb33e9f934a7d，原始Go來源965451e5b6a92eaf077d7a81dedf4422e55583b7335238551985fda6d7982749，sourcebank共391圖。五對保存狀態經獨立maze-saved-state-independent-v1-20261003.bin核對完整44機器欄位及DOS相同，CPU／RAM／port負對照有效。最後24次貼圖只有ENEMY00 #6一個完整匹配，其餘包含局部XOR；不當24張新ALLY，也沒有取得ALLY #3正常來源。

追加更正：東／南兩份sivad-assisted-ally3-*-run-v1-20261006.json沿用了舊wrapper，scope誤稱起點來自F7／F8 GUI。實際argv與SHA都是ALLY2加入命名後普通鍵。保留兩份原收據與舊wrapper，由ally3-normal-source-proof-v1-20261006.json明確更正。檔名中的sivad-assisted不是區域或輔助來源證據。新ally3-normal-follow-v1-20261006.py使用正確敘述。這項更正不影響§173–176真正F7／F8起點的電梯收據。

入口沿同研究根ally3-normal-source-proof-v1-20261006.json與verify-ally3-normal-v1-20261006.py。五組run、event、observed／control.state、end.frame、position、preview與independent JSON保存完整argv及SHA；獨立44欄位是判準，不以非canonical gob bytes當完整相等證據。未做新GUI、DAT或正式sprite接入。

### 177.3 收尾與封存

同研究根preserve-ally-art-v1-20261006.py封存344份、255輸入檢核，串流逐份驗證344/344，命令退出0。本機ally-art-source-snapshot-v1-20261006.tar.gz SHA d3f1de9ac4ef749d77391f35a655040c0516c8bceb4d75a6c9ae8b6d2b7da5d9，34,416,502 bytes；manifest保存原版掛載別名對應。兩次前置失敗分別為漏掛/orig及未解析相對來源路徑，皆在建tar前停止；補唯讀掛載與正規化後同image重跑，不降低SHA判準。封存只保存列明素材與凍結工具，不宣稱完整runtime建置依賴或公開散布權。

遠端#34全文與本機3399字一致，updatedAt 2026-10-05T19:38:12Z、保持OPEN；讀回ally-art-issue34-after-v1-20261006.json，#44未改。根與dosgolem git diff --check通過，既有image沒有殘留容器。檔案擁有權、主題未變與封存後文件SHA由同研究根ally-art-final-audit-v1-20261006.json保存。原版、美術、原生及state全部本機，未commit／push／發行。

## 178. 現行43筆主題的敏頓DAT存讀

日期2026-10-06。範圍confirmed僅限現行43筆PBL＋256格MAZE／41PNG在既有ALLY2 GUI檢查點後，經普通原版鍵盤存檔、SELECT讀檔及道具肖像。HD美術正式定案沿024 §1.26及AGENTS §12，不再詢問外觀。本輪沒有改正式程式、主題、譯文、字型或玩法。

### 178.1 輸入與工具

沿既有Docker image psychicwar-go-ebiten:latest，SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7；Go1.24.13、Python3.11.2、ImageMagick6.9.11-60及Xvfb。原版/orig唯讀，/gomod唯讀，研究輸出UID/GID1000；GUI每容器2CPU／1GiB／128程序，普通控制1CPU／512MiB。均有外層timeout與--rm，Xvfb有trap。

前端elevator-english-frontend-v1-20261006.bin SHA2ee43d77763c89d46c1092a3269774049477acf09c36db08bd0d9034870c63a4，實際348非標準來源逐份核對。新hd43-ally2-dat-pwstep-v1-20261006.bin SHAef5a43d9e123c3f617b24e537c39a0c73aae3eaed24c2837e38d59e581c3e6b0，92來源；新hd43-ally2-dat-export-v1-20261006.bin SHA00270638d4568d84a955541f094bdafb8d3d2ff7f964c598f8bb22f625fab1d4，77來源。三份-build.json保存實際編譯輸入全文／base64與SHA，標準庫由上述image固定。

起點ally2-style-gui-v1-20261005/saves/quick.state及正式中文字面，當時是See Items敏頓資訊，ALLY2位於(128,8)與(232,152)，Kai位於(264,152)。原始加入路徑與更早起點限制沿§141及§158，未增加從開機聲明。初始化F11只恢復正式快照與字面，後續沒有注入RAM、HP、座標或seed。SAVE與LOAD的輸入SHA、argv、load、原版按鍵紀錄保存在execution與record收據。

### 178.2 正常操作與限定結果

See Items以Down、Return離開，Esc後實際選單依序為詢問凱拉、喝體力恢復劑、看道具、使用超能力、功能選項、取消。四次Down及Return選到功能選項，這份真正GUI狀態另存為save-v3/d-options-hd-chinese.state。save-v4從此狀態接續，Return選儲存，Return選當然要，輸入hd43並Return。畫面顯示「遊戲已存檔」，再Return繼續。SELECT檢查點經Down、Return、hd43、Return讀回，再Esc、Down兩次、Return看Kai，Return看敏頓。鍵序有實際畫面核對，沒有沿用其他游標狀態的固定Down數。

同研究根hd43-ally2-dat-save-v4-20261006與hd43-ally2-dat-load-v1-20261006兩次真正前端均正常退出0。保存7份與讀回8份，共15份；檔名輸入modal僅路徑證據，另14/14完整960×600圖面與獨立原版PBL、MAZE、批准PNG及正式字型期望差0。每份擷取12相位，至少一相位全圖相同，不稱所有相位或完整動畫。F5中英切換、Shift+F5 HD開關及讀回後ALLY2兩位置通過。省略迷宮、省略上方肖像及一像素負對照有效。MAZE保持B v9 SHA7dc7de0064339d9e54f958f3cbf5e00181c6aeb80484d3c3e54b5612cf71736f。

DAT完整512位元組SHA b32d25e6298fb0dd7a14de3202cd80a53479826e0cda6793b94f5081030c58f8，真正Save、Load副本與無覆繪原版獨立存讀均完全相同。52位元組玩家資料相同；敏頓道具畫面及獨立原版控制的ALLY2上方、下方及Kai三處，五份共15次完整24×32來源匹配。Kai道具頁的臨時欄位byte38為0，敏頓為1，由兩側實際資料核對，不套用前批ALLY1暫時值。原版按鍵紀錄保存階段16邊緣、讀回階段24邊緣，全部與批准普通鍵序相同；DAT與玩家資料各一位元組負對照有效。獨立控制使用相同GUI起點及等價普通鍵序，不稱同指令數全RAM或完整亂數序列對拍。

### 178.3 未通過嘗試與入口

保存v1的160秒期限在檔名盲輸入中耗盡，前端退出0但擷取窗口失敗；其中d的state複製了既有c，沒有對應新PNG，不當存檔完成證據。v2舊鍵序選到超能力，沒有DAT，確認後停止並保留退出143。v3實際選單觀察完成至功能選項，退出0；第三批命令在退出後才送達，terminal只完成sequence2。三者不列DAT通過，也不分類成產品缺陷。

通用匯出器第一次因缺-w /src找不到相對來源，在寫出前失敗；同一image、同一binary補正cwd後重跑，完整匯出7份與8份，沒有放寬判準。獨立原版控制v1執行後曾修改道具鍵序；已恢復v1原文，正確鍵序另存v2並重跑Load，舊load-original-v1不列驗收。Save控制沿v1，Load及Items控制沿v2，各輸入SHA保持。

本節全部入口在workplace/ida/hd-ally-recruit-20261004/：prepare-hd43-ally2-dat-v1、build-hd43-ally2-dat-v1、hd43-ally2-dat-gui-run-v1、gui-v1–v4與load-gui-v1、save-actions-v1–v10、load-actions-v1–v3、model-v1、verify-gui-v1／v2、original-v1／v2、byte-proof-v1及對應-build、model-inputs、prepare-inputs與verified JSON。GUI腳本以已存在的檢查點、/src唯讀、研究根可寫、/orig及/gomod唯讀掛載執行；save-v4與load-v1拒絕覆寫，重跑須新版本輸出。獨立原版控制依序傳save、load或items；完整GUI模型傳輸出目錄及期望完成sequence，Save為4、Load為3。

電梯途中、新敵人DAT、其他ALLY來源、完整動作、效能、公開權利及封包未完成。正式敵人24/360、ALLY3/31不增加。完整HD與#34保持未完成，#44不改；原版、DAT、state及PNG全留本機。

### 178.4 收尾與本機保全

preserve-hd43-ally2-dat-v1-20261006.py逐份核對746項輸入，封存1166份並串流重驗1166/1166。保存真正與未通過GUI、原版控制、348／92／77實際建置來源、原版唯讀輸入、現行主題與驗證腳本；失敗收據不列驗收。私有hd43-ally2-dat-source-snapshot-v1-20261006.tar.gz為105793328位元組，SHA ac168ca1b60d3f74b6fc14bf239c7d56d06ace47bb3161a0e2cc74b295e314d6。檔案清單與原始SHA保存在source-manifest，核對結果source-preserved；均在本節同研究根。

主機gh auth通過，#34讀回全文與body檔完全相同，updatedAt 2026-10-05T20:47:09Z、保持OPEN，#44未改。根及dosgolem git diff --check通過，本輪專用容器均已清理。封存後文件、擁有權及限定結果入口hd43-ally2-dat-final-audit-v1-20261006.json。未commit／push／發行，原版、DAT、美術與state只留本機。

## 179. ALLY全圖號分類、第四稿及候選來源接續（2026-10-06）

使用者再次明示HD美術正式定案，不再詢問。原位、比例、8×8、人物在後及框線在前保持。本輪依READY規格閘門載入local/retro-remake-spec-gated-workflow.md，美術修稿依imagegen技能。全部來源、轉檔及核對沿既有psychicwar-go-ebiten:latest Docker，image SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7，Python3.11.2、ImageMagick6.9.11-60。觀察、分類及轉檔用原版唯讀掛載，研究根與redraw指定UID/GID1000可寫；文件整理與封存只寫明示現況檔及研究根，原始輸入SHA保持。有外層逾時、資源限制、network none及--rm。本輪沒有新正式程式或主題變更。

### 179.1 ALLY全31圖號的靜態事實

原始ALLY.PBL SHA c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219。31個偏移與表頭逐張讀取，strict_pbl與pbl.decode各自輸出相同；證據位址為檔案偏移，不是IDA或runtime位址。完整接觸表按行排列#0–#30，八欄四列、1152×768，原始索引像素最近鄰六倍，未重畫素材。

| 圖號 | 已證實靜態形狀 | 正常用途限制 |
|---|---|---|
| #0–#11 | 12張24×32完整人物參照 | #0–#2既有正常來源驗收保持，其餘逐張查角色與位置 |
| #12–#15 | 4張24×32局部圖形 | 大量色號10。不能僅憑綠色判定透明或XOR，正常呼叫與模式未知 |
| #16–#30 | 15張16×16小圖 | 正常圖號、來源RAM、位置及模式另驗，不放大補畫成完整人物 |

追加更正：§177及舊CONTEXT將剩餘28個ALLY圖號籠統稱為28盟友，並把#12–#30作為後續人物草稿清單，分類不準確。正式3/31是接受的archive圖號數，完整人物是3/12；剩餘28圖號為9完整人物及19其他圖形。全31圖號仍在HD範圍，本次不刪減工作，也不宣稱19圖形未使用。現況及024 §1.1／§1.41依來源分類更新，歷史§177保持並由本節更正。

分類v1／v2的31張個別圖檔已輸出，ImageMagick montage兩次SIGABRT6，沒有最終分類JSON；這是研究工具失敗，不是遊戲缺陷。第二次後回查規格流程入口，停止繼續調montage。v3以既有tools/pbl.py PNG writer直接組合原始索引參照，31份檢查及最終JSON完成，沒有修改原版或正式美術。

入口在workplace/ida/hd-ally-recruit-20261004/：prepare-ally12-14-reference-v1、classify-ally-archive-v1–v3-20261006.py，ally-archive-classification-v3-20261006.json與ALLY-all31-classification-v3-20261006.png；失敗版本個別參照保留。原版#12–#14參照及ALLY-12-14-reference-proof-v1-20261006.json在workplace/hd/redraw/，兩解碼器及色號已核對；非黑範圍包含色號10，不當人物輪廓。

### 179.2 ALLY #3第四稿

內建imagegen返回cell162，完整原生1086×1448，SHA7865ee3b5438658ac460f4b5efd79e276e96064152057f352c38bee4859c79c1。整幅Lanczos至72×96、不裁切平移或補像素，圖面SHA4f3142a9d8ac16c1af6143914ac90bd1d5d2849e187adc1fedd966640a1dc520；完整提示SHA23d4605f400f10fa6462b381d448a284f36f60fc75af386e9eaf419843f7f60e。工具返回原檔與本機保全一致，預設生成檔保持。

完整RGBA重生差0，錯Point濾鏡差12671個位元組；實際翻轉expected[0]後差1，負對照有效。補充訂正：generation-review先寫入單位元組數值但尚未實際翻轉；該欄不單獨作證，只有後續verification-v4實際操作才列有效負對照。原收據保留。

第四稿實際比較圖仍見手部及左靴端點差異。局部手部診斷末列70對原版68；左靴末列93對92、末列左端14對12。局部區域可能包含腿部，不以此宣稱完整手末端。完整造型及正常來源仍待驗，DRAFT、不接入。#3–#11仍九張人物草稿，累計二十份原生，正式完成數不增加。

入口workplace/hd/redraw/ALLY-03-v4-{generated,frame,preview,comparison}-20261006.png、ALLY-03-prompt-v4-20261006.txt、ALLY-03-generation-review-v4-20261006.json及ALLY-03-verification-v4-20261006.json。重跑核對入口在本節研究根verify-ready60-follow-v1-20261006.py；存在輸出即停止，重跑須新版本。

### 179.3 原版來源盤點及普通鍵接續

既有104份貼圖event盤點，實際(32,152)的24×32貼圖後完整frame與四圖庫#0–#14逐像素核對，扣除已接受24張，沒有待呈現36張的新完整匹配。只報此掃描範圍，不能說遊戲沒有這些圖。入口enemy-ready60-pending-source-inventory-v1-20261006.json。

沿§173–175實際F7／F8 GUI來源後的SIVAD(7,9)普通鍵狀態繼續，沒有新增RAM、HP、座標或seed寫入。北向牆阻後，普通選單取消並轉東，走到(10,9)再向南到(10,10)第二座電梯，原版選Go down到(10,2)，Leave後(10,1)向北。其後向東至(11,1)，向南至(11,3)，轉西到(10,3)遇既有Machipi；第一份完整身體匹配ENEMY01 #0，其餘包括已知局部XOR，HP最後0。由首張身體實際state分支按原版F3，仍HP0，不列逃走通過。未重擲seed或修HP挑結果，不稱自然戰鬥或從開機路徑。

十四份成功run均由凍結maze-saved-state-independent-v1-20261003.bin外部核對44機器欄位及完整DOS，各相同，CPU／RAM／port負對照有效。這十四段與§175前批十四段不同，不能混成同一收據。觀察器rusteck-sprites-observer-v1-20261005.bin SHA ae0e8d2f1d5d3424cf7fefe6d43a4a27ea4f7661dc35deb701ecb33e9f934a7d；保存原版391圖來源bank、實際鍵序、輸入SHA及前後state，不以事件數充作新美術數。

upper-east-exit首次只預留12m，八個鍵的按下及放開各需0.8m，正常鍵事件數不足而停止。觀察與控制state已輸出，不列該run通過。upper-east-exit-complete使用完全相同起點、key-at、key-every及keys，只延長end至789m完成；不是新seed或產品修復。舊失敗全部保留。

入口在本節研究根enemy-ready60-follow-v1-20261006.py，參數label、start相對路徑、end絕對指令數、key-at、key-every及逗號掃描碼。各段enemy-ready60-follow-{label}-run-v1-20261006.json保存精確argv；event、observed／control.state、frame、position、preview及independent收據同前綴。總核對enemy-ready60-follow-proof-v1-20261006.json，重跑verify-ready60-follow-v1-20261006.py。沒有取得新ALLY或候選36張來源，正式43筆主題／41PNG、敵人24/360及ALLY3/31保持；完整HD與#34未完成。

### 179.4 保全及同步

封存後核對重跑入口同研究根audit-ready60-follow-v1-20261006.py，保存ready60-follow-final-audit-v1-20261006.json。

同研究根finalize-ready60-follow-v1-20261006.py整理本機現況與#34正文，再封存306份，116項輸入檢核，逐份串流306/306通過；另以完整tar成員清單核對306，沒有多餘成員。私有ready60-follow-source-snapshot-v1-20261006.tar.gz為12473546位元組，SHA619dcac40740f8d1f1d97b3dab64f0ba3f5a38ff074b563a56bf08c1bc5c83db。manifest列來源路徑、逐份SHA、引用前批完整43筆DAT封存及保全範圍，不是公開發行包。本節收尾文件在封存後追加，最新文件SHA另存ready60-follow-final-audit-v1-20261006.json。

主機gh auth通過，#34讀回全文與本機4038字完全一致，updatedAt 2026-10-05T21:28:08Z、保持OPEN；before／body／after收據為enemy-ready60-follow-issue34-{before,body,after}-v1-20261006。#44未操作。根及dosgolem git diff --check通過，研究根、redraw與現行主題無root-owned檔或*.md目錄，本輪專用容器皆清除。未commit／push／發行，原版、原生、美術、提示與state保持本機。

## 180. 敏頓效果的正常來源方向、重疊及完整冷載（2026-10-06）

本輪以前一輪ALLY分類及現行43筆主題為準，沒有重問美術或重開已完成DAT。路由命中READY／實作閘門，載入local/retro-remake-spec-gated-workflow.md及local/project-document-responsibilities.md。小圖原版來源沿§56／154；這批補收先前未包含的內建遮罩，解開209次小圖的無方向限制。美術B透光及全部sprite範圍沿024 §1.26，正式接入仍依DRAFT閘門。

### 180.1 工具及可重播輸入

既有Docker psychicwar-go-ebiten:latest SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7、Go1.24.13、Python3.11.2，UID/GID1000，network none。原版及專案唯讀、研究根可寫；Go快取沿用，建置1CPU／1GiB，觀察1CPU／512MiB，均有外層逾時、程序及日誌限制、--rm。起跑load3.37，只做指令數決定性及像素來源驗證，沒有音訊或效能結論。

起點workplace/states/13-healed.state SHAd9c18d64a60ed678d34d72f00c199af51108c2a36931fdcaed650a110f23e7a0，原始seed A48C；更早13-healed來源限制保持，不稱從開機或自然戰鬥全流程。沿replay/title-to-first-save.json的14-minton1：up、right、五次up，key-at345500000、key-every3000000；space在382200000按住30000000，終點420000000。PW.EXE SHA88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49。沒有新RAM、HP、座標或seed寫入，不重擲結果。

新唯讀觀察器沿tools/hd/observe_small_projectiles.go精確SHA，補收0161:8260入口與0161:4E34返回。DS:DX限定0161:4E36，32 bytes，BX從C000h算原始畫面位置；位元遮罩以色號10 XOR。一般貼圖仍觀察0161:8705／8751，DS:BX為原始來源、CX為位置、DX為尺寸。以上是執行期CS:IP與段偏移；PW_UNP.EXE檔案偏移由MZ header及初始CS另算並保存在directed-proof，不把IDA EA混成檔案或RAM位址。

實際執行二進位minton-effects-observer-v4-20261006.bin SHA78eed91f7b8921b1ab304d58ed8c879abc4e97bb7595222b289f43ff06e19b1e。63份實際非標準編譯來源的全文／base64、SHA、Go overlay與命令保存於-build.json；主程式為observer-v2，沒有改正式Go或dosgolem工作樹。v3成功二進位SHA bf4efe35e7bc2f89e83ba979a9a81b5d9f8b07a46bcf26cd2e60397af69d4609只作建置保全、未執行遊戲；v4另保存無觀察控制state，以獨立44欄位核對。

### 180.2 原版完整垂直證據

本次真的重跑同一起點、原版鍵序與seed，1,009次一般貼圖及110次內建遮罩。1,009份一般事件的入口／返回指令數、暫存器、來源SHA、前後完整frame SHA與§56舊收據逐份相同，新增觀察沒有改變原版。觀察／無觀察／舊基準指紋相同；新保存兩側由凍結maze-saved-state-independent-v1-20261003.bin比較44機器欄位及完整DOS，各相同，CPU／RAM／port負對照有效。

獨立Python從實際26份PBL重解原圖、從PW_UNP.EXE重解32-byte遮罩。初始戰鬥背景只由SCREEN五張、MENU及ALLY #0建立，不由截圖反填期望。按真實事件次序維護(name,rect)唯一前圖；AL0初始身體、AL1完整來源新增／消除、AL1差分有向切換，以及每位置遮罩的新增／消除。原版色號XOR重建所有重疊。不同來源在同位置獨立保存，不把BEAM與敵人小圖混成一個槽。

2,238個入口及返回邊界的(32,144,256,40)戰鬥區與獨立原始素材模型差0。785次XOR差分方向全部以已知前圖及實際after驗證，含ENEMY00身體、小圖、BEAM及FIGHT，不能只由圖號或可交換性猜方向。最多24個同時存在圖元的見證為一般事件370、返回step386834245；這是同場景圖元數，不說單一像素有24層。

1,009個錯來源byte負對照重算完整原版輸出，各差1；785個反向目標負對照均檢出差異。首份遮罩負對照只翻輸出像素，另以verify-minton-mask-source-negative-v1實際翻遮罩來源byte0 bit0，再從before重算，110個完整畫面各差1。兩種結果分開保存，不把輸出比較當錯來源介入。

### 180.3 完整場景冷載唯一性

另建101個實際(name,rect,image)來源變數，向量只取原始圖及基底；GF(2)秩101。冷載求解器沒有使用事件真值來建basis，所有2,238個完整畫面的解與上述來源事件追蹤真值一致。每個(name,rect)最多一個姿勢。未知像素(280,180)、同位置兩個身體姿勢、原檔ENEMY01 #0外部身體均被拒絕；故意翻已知變數與獨立真值不符。完整解只證明這套已知來源的圖面可唯一重建，不證明任意未知來源執行身份、所有中途投影或部分繪製安全。

這批不生成美術、不做PNG補像素、不改正式主題／程式／文本／字型，沒有普通HD GUI或新DAT聲明。正式43筆PBL＋256格MAZE／41PNG、敵人24/360及ALLY3/31保持。024 §1.42與§1.5仍DRAFT；舊§1.5中等待混色回答的過期文字改為§1.26正式B定案。下一步用已解出的正常來源與101變數接入B透光，補正式資料表示、原位8×8回退、冷載、GUI及效果中途DAT；不再重跑相同來源矩陣。

入口均在workplace/ida/hd-ally-recruit-20261004/：prepare-minton-effects-v1、build-minton-effects-v1–v4、run-minton-effects-v1、verify-minton-effects-v1、verify-minton-cold-scene-v1及verify-minton-mask-source-negative-v1-20261006.py。新原始事件minton-effects-normal-v1-20261006.json與逐筆before／after／source、observed end／control state保持本機；directed-proof-v1、cold-scene-v1、mask-source-negatives-v1及machine-independent-v1記錄逐項輸入SHA、來源offset、方向、完整樣本及負對照。

### 180.4 研究工具失敗分類

建置v1只覆蓋probe main.go，忽略observe.go對die的依賴而失敗；v2改單檔建置，觸發Go internal package界限。兩次後回查規格入口，再查看既有rusteck-sprites-build與overlay：該成功工具同時覆蓋main.go、empty-observe.go及machine-schema.go。v3沿完整既有overlay成功，v4只加保存control state成功。沒有換image、host建置、改正式工具鏈或降低來源判準。第一次查錯rusteck build檔名只屬研究讀取錯誤；依實際既有檔名補讀。失敗來源及診斷保留。

進度、本機保全與收尾入口finalize-minton-effects-v1-20261006.py、minton-effects-source-manifest-v1-20261006.json、source-snapshot-v1及final-audit-v1。CONTEXT、024、WORKLOG、worklist與GitHub #34只同步上述有限結果；#34保持OPEN、#44不操作。原版、state及所有美術留本機，未commit／push／發行。

本輪保全3493份／3470項輸入檢核，串流3493/3493通過。私有封存SHA88412640e6fdd778d0d4989d3ac35b930b6827602e1851925ac5565e02d154c1，34426720位元組；精確清單同研究根minton-effects-source-manifest-v1-20261006.json。封存後文件與遠端讀回SHA由final-audit-v1保存。

最後#34全文相同、updatedAt 2026-10-05T21:56:12Z、保持OPEN。根及dosgolem diff --check通過，本輪pw-hd-small-*容器皆清理。收尾文件SHA同研究根minton-effects-closeout-docs-v1-20261006.json；沒有重新驗原版、改接受數或新增發行聲明。
## 181. 敏頓效果B透光與內建來源格式原型（2026-10-06）

### 181.1 範圍、決策及來源

HD美術、B透光、8×8、原位比例、人物在後與框線在前已定案。本批沒有外觀訪談、重新生成美術或改原版資料。新的內建遮罩主題表示依AGENTS與grilling送出一次A／B格式問題，尚無回覆；024 §2.0及§1.42保持DRAFT，原型不進正式路徑。

來源沿§180敏頓正常鍵路線、101變數／秩101完整冷載模型及獨立事件真值。沒有重新執行同一來源矩陣，也沒有新RAM、HP、座標或seed寫入。遮罩是解壓原版CS:4E36的32 bytes，不當PBL圖號或任意EXE offset。美術沿原有ENEMY00身體6–8、小圖21–23、BEAM0–2、FIGHT0–3及battle-effect-mask-48-v3；小圖原版參照是彩色星形／局部圖形，不能只因候選星形就當新增構圖。幾何與普通GUI未在本批宣布通過。

兩份原型在workplace/ida/hd-ally-recruit-20261004/minton-effects-format-prototype-v2-20261006/A與B。各52PNG完全相同；A共141筆entries，以pbl=PW.EXE／image=0標唯一已知遮罩。B共114筆entries＋27筆builtin_masks，以id=battle-mask-4e36標來源。正規化後98筆效果來源／位置／PNG相同；既有43筆及256格迷宮保持。正式LoadTheme尚不接受兩份效果表示，不能宣稱相容載入通過。格式原型與統計保存在同研究根minton-effects-format-prototype-v2-20261006.json。

### 181.2 私有渲染器及驗證

既有tools/hd/prototype_battle_effects.go SHA bf55289327419cf916eb93e7d1faa29f8e689fc70cb172a9bd873f98c247c232複製為私有minton-format-render-v1-20261006.go，保留B透光與8×8計算。敏頓的Kai-only基底只畫ALLY #0於(264,152)，不把正式主題其他位置ALLY或其他盟友誤畫進戰鬥。兩格式只以同一來源／位置查找PNG，拒絕不唯一及越界檔名。

用原版終點state讀取EGA DACIndex對應色盤，零指令載入。三份完整邊界的原始frame SHA先與§180核對，按最大同時圖元及其餘來源素材覆蓋選取，沒有依通過結果挑圖。步數386834245、383420111、382082211，分別24、20及1個圖元。涵蓋所有14種本模型來源素材；24指同場景圖元，不指單像素24層。三份A／B完整RGBA相同。

獨立Python從原PBL及32-byte遮罩重建原始模型，以已獨立驗過的事件真值挑來源，不用Go解的active_variables建期望；SCREEN／MENU／ALLY0及當前PNG重新合成B透光。每色頻道透光乘積、原版來源空8×8格禁畫、全域8×8比對及原版回退逐項核對。三份完整960×600 RGBA各差0，935格吻合、65格原版回退。固定終點色盤只證明兩格式合成相同，不證明每個原版階段的RGB色盤、中文疊字、普通GUI或所有中途投影安全。

實際合成負對照：省略BEAM差22,486像素，省略MASK差5,152；BEAM錯移4原版像素差26,405，僅原版輸出差180,011，改一RGBA byte差1。原版來源與全部參與PNG／JSON／工具SHA先逐份核對。正式接受數不增加：43筆PBL＋256MAZE／41PNG、敵人24/360、ALLY3/31。完整動畫、未知來源安全、正式GUI、效果途中DAT及封包仍待完成。

可重跑入口均在上述研究根：prepare-minton-format-prototype-v1／v2-20261006.py、prepare-minton-format-render-v1-20261006.py、build-minton-format-render-v1-20261006.py及verify-minton-format-render-v1-20261006.py。使用既有psychicwar-go-ebiten:latest image SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7，Go1.24.13／Python3.11.2。Go以既有三檔overlay在dosgolem ./cmd/probe建置，沒有修改工作樹。二進位SHA ba631366676339f84209944360d635ffe46794600b5a4ad1b1da3376b612f935；94份實際非標準來源全文／SHA與命令保存在minton-format-render-v1-20261006-build.json。執行-out必須用不同的新前綴，實際為minton-format-render-output-v1-20261006，避免撞到拒絕覆寫檢查。

渲染與獨立收據為minton-format-render-output-v1-20261006.json、三份sample00–02-{original,A,B}.png及minton-format-render-independent-v1-20261006.json。第一版格式原型誤讀maze.png，實際欄位為atlas，產生部分空目錄後停止；保留v1，v2修正後成功。收尾查錯-build檔名與Issue repo拼字各一次，改由實際檔案及git origin核對，沒有工具鏈或產品缺陷；首次錯repo未寫遠端。

### 181.3 進度、保全及後續

CONTEXT、WORKLOG、worklist及#34同步這個有限結果。#34保持OPEN，#44不操作；沒有commit、push、PR或發行。原版、PNG及state保持本機。新腳本、兩表示、PNG、94份建置來源與輸入雜湊保全由finalize-minton-format-v1-20261006.py及minton-format-source-{manifest,snapshot}-v1-20261006建立，原始2,238邊界保全仍沿§180私有封存，不重複宣稱新來源驗收。

待資料表示回覆後，依§180已有證據補正式來源、生命周期、冷載與8×8未知回退契約，READY後再實作。美術正式定案保持，不重新詢問。下一垂直鏈是實際普通中文HD GUI、來源消除與效果DAT，不能以本批三份原型取代。

本批私有增量封存247份／129輸入檢核，串流247/247通過，SHA 4bc8f65a317679a721349a1d071b64ed0b122df31554074a67f7c01f92d1df39。原始全邊界仍沿§180私有封存。#34全文讀回相同、updatedAt 2026-10-05T22:18:31Z、保持OPEN。封存後文件SHA與擁有權由closeout-minton-format-v1-20261006.py及minton-format-closeout-docs-v1-20261006.json保存。

## 182. ALLY #3指端更正與電梯樓層DAT垂直驗收（2026-10-06）

### 182.1 當前權威及工作邊界

上一輪§181新增三份B透光格式原型與獨立證據，屬進度。內建遮罩格式A／B問題仍待回覆，正式效果契約保持DRAFT；本批做不依賴該選擇的盟友素材及現行43筆電梯樓層DAT。HD美術已正式定案，不重新詢問，沒有改正式Go、主題、字型、原版資料或存檔格式。入口沿024 §1.35／§1.41 DRAFT及電梯§1.39既有READY契約。

### 182.2 ALLY #3第五稿與來源勘誤

ALLY.PBL SHA c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219，#3靜態24×32來源既有雙解碼已confirmed。原版第22列x0–6的色號為0、15、6、0、0、6、0，形成兩個分離指端；省略第二色區的負對照變為一區。這不證明實際手指總數。第四稿完整提示中的three block-colored fingers沒有原版依據，保留舊提示並以ALLY-03-fingertip-source-proof-v1-20261006.json追加更正，不再拿提示當原版事實。

以內建imagegen cell249編輯第四稿與已檢視的原版單格參照，只指定指端及左靴局部輪廓；不改風格、不平移或縮小整個角色。原生1086×1448 SHA326467b9864ece832dbb764ce965a31cdc56bbfc9c9e60e751221a7158a72ef4；完整72×96 SHA5110606e78dc4ff58e801298af591b35cfd664e9a4a07ff8782aa3d9d0a9c8c1。完整prompt-v5 SHA7c444368640c8ddf4efa08cca78a88ccd75ed86861b3927385b80b572215ff91。整幅Lanczos縮放，沒有裁切、平移、填像素或換背景。

完整RGBA重新生成差0，實際Point錯濾鏡差12,764 bytes，改一byte差1。原版／第四／第五稿比較圖1728×768全部RGB與來源拼合差0，已實際檢視。使用相同ROI及RGB>16，第四／第五手部局部末列皆69，左靴末列皆93；最低列左界11→10，原版最近鄰對應手68、靴92、左界12。這個診斷與前批不同ROI／閾值數字分開保存，不當完整幾何或手指身份證明。兩次端點改稿仍未穩定達成要求，停止同類提示微調，不放寬要求或宣稱已修好。九張人物草稿與二十一原生保全，未增加正式接受數。

入口在workplace/hd/redraw/：ALLY-03-prompt-v5、v5-{generated,frame,preview,comparison}-20261006.png、generation-review-v5、verification-v5及fingertip-source-proof-v1-20261006.json。重跑工具在workplace/ida/hd-ally-recruit-20261004/：prepare-ally3-v5-art-v1及verify-ally3-v5-art-v2-20261006.py。v1原生／轉檔已完成，寫比較圖時誤呼不存在pbl.write_png而退出1；v2使用既有ImageMagick RGB輸入只補比較圖，另重驗既有素材，退出0。失敗工具與已產生資料保留。

### 182.3 電梯正常存讀路徑與原版資料

起點是§176真正GUI的elevator-english-gui-v1-20261006/lower-chinese.state及正式中文字面。原版電梯選單對正常Escape未開Options，隨後Return選離開、Escape才進主選單。這三個正常按鍵結果保存keycheck-v1／v2，不用座標、HP或seed寫入補路徑。實際可驗路徑是離開電梯、附近樓層存檔、由SELECT載回，再普通Right／Right／Up回到ROOM0 #5。

沿現行前端elevator-english-frontend-v1-20261006.bin SHA2ee43d77763c89d46c1092a3269774049477acf09c36db08bd0d9034870c63a4，348份實際非標準來源在每次GUI執行前逐份重新核對；沒有重新建置或改正式程式。獨立原版pwstep SHAef5a43d9e123c3f617b24e537c39a0c73aae3eaed24c2837e38d59e581c3e6b0及92實際來源，零指令匯出器SHA00270638d4568d84a955541f094bdafb8d3d2ff7f964c598f8bb22f625fab1d4及77來源仍沿§178凍結-build.json。全部使用既有psychicwar-go-ebiten:latest image SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7、Go1.24.13／Python3.11.2／ImageMagick6.9.11-60，有界非root、原版唯讀、network none；本批沒有音訊或即時速度聲明。

GUI存檔13鍵／26邊緣，GUI讀檔及再入10鍵／20邊緣。兩次真正GUI都自然退出0、各完成sequence1；F5中英與Shift+F5 HD開關由正式前端處理，不混進原版鍵紀錄。512-byte hd43.dat SHA79412fc0a6e58e3b79f25e8bac1884ac6db725c9fb8ce8a3cdc7b5f1d45302bc，GUI存讀與兩次獨立原版控制完全相同。載回及再入兩階段各52玩家bytes相同，三份GUI與一份原版控制的ROOM5完整72×72及ALLY0完整24×32來源吻合。DAT／玩家資料實際翻byte負對照各檢出一處。這不是GUI時序、全RAM同指令數或完整自然亂數序列對拍。

### 182.4 全圖面、字面及失敗分類

存檔5/5、讀檔5/5，共10份完整960×600圖面，各從12相位中至少一份與獨立原版PBL、MAZE v9、正式font／text及批准PNG期望相同，差0。1份前端檔名輸入框只作路徑證據，不列通過。載回後再入的中文HD、英文HD及原版英文均核對；三份包含完整原版ROOM5。中文字模背景只依已READY的024 §1.38／§1.39取當幀不透明HD材質，英文沿原位76個原版前景像素的684最近鄰3倍像素。

省略迷宮差6,093像素；中文ROOM5省略差39,633、英文ROOM5省略差41,595；省略英文原位字模差684，一RGBA byte負對照差1。正式HD關閉的原版英文也完整圖面相同，沒有只驗可見格或將遊戲內部旗標當答案。只比較已捕捉畫面，不宣稱全部12相位或所有動畫。

首次GUI save-v1缺/gomod唯讀掛載，實際編譯來源稽核在開前端前停止，退出1；保存空saves目錄與錯誤，不當產品缺陷。save-v2及load-v2補齊既有模組cache唯讀掛載，使用同image、同鍵序與同GUI控制器，成功正常退出。原版Escape忽略是玩家路徑事實，先按原版離開後存檔，沒有改規則讓選單內Save成功。

全部新工具索引於上述研究根：prepare-elevator-dat-v1、prepare-elevator-dat-wrapper-v2、prepare-elevator-dat-verifier-v2及elevator-dat-original-v1-20261006.py；elevator-dat-gui-v1／v2-20261006.sh傳save或load；elevator-dat-{save,load}-actions-v1-20261006.json、verify-gui-v1／v2-20261006.py及byte-proof-v1-20261006.py。控制器沿hd43-ally2-dat-gui-run-v1，獨立全圖模型沿hd43-ally2-dat-model-v1，原位文字的增加均是私有驗證器，不是正式實作。收據在elevator-dat-{save,load}-v2-20261006/{execution,terminal,original-frames,verified}.json、兩份-original-v1/{execution,after.state}與elevator-dat-byte-proof-v1-20261006.json。完整原版、DAT、state與PNG不公開。

### 182.5 現況、保全及下一步

現行43PBL＋256MAZE／41PNG、敵人24/360、ALLY3/31保持；電梯附近樓層DAT／載回再入已驗，電梯選單內或動畫中DAT不稱通過。效果正式資料表示／接入、普通效果GUI／DAT、其餘sprites、全部動畫、效能、素材公開權利及封包未完成。#34保持OPEN、#44不改，沒有commit、push、PR或發行。HD美術不重問，遮罩資料表示仍待已有提問回覆。

CONTEXT、WORKLOG、worklist及#34同步有限新結果。收尾入口update-ally3-elevator-dat-docs-v1-20261006.py、finalize-ally3-elevator-dat-v1-20261006.py與ally3-elevator-dat-{source-manifest,source-snapshot,final-audit}-v1-20261006。本批新素材、失敗工具、GUI、原版控制、原始輸入及三份凍結建置來源保全在私有增量封存，封存後文件與遠端讀回由final-audit記錄，舊素材與父封存保持。

本輪私有保全776份／1025輸入檢核，串流776/776完整核對通過。封存SHA04f7e46692aff1293f5301175f095a983b9bf3e086341c10c62f3aea0ab618e8，58977136位元組。#34全文讀回一致、updatedAt 2026-10-05T22:44:48Z、保持OPEN。 收尾文件SHA與擁有權見ally3-elevator-dat-final-audit-v1-20261006.json；現況收斂工具curate-ally3-elevator-current-v1-20261006.py，全部入口仍沿本節研究根。

## 183. 美術再次確認與Rusteck輔助來源起點（2026-10-06 07:03）

【confirmed：使用者決策、真正GUI操作、原版保存點及四段控制比較。新sprite美術與正式呈現未驗。】

2026-10-06使用者再次確認HD美術正式定案，不再詢問美術，也不要求逐張批准。沿既有風格自主完成全部sprite，原版位置、比例、姿勢、8×8及人物在後／框線在前保持。決策入口AGENTS §12、024 §1.26。

Rusteck新增真正GUI F7／F8輔助戰勝起點：原位(0,7)、HP40／能量30、敵人HP0，前端正常退出0；更早Pionn起點限制保持。四段後續普通鍵路線的44機器欄位及完整DOS與獨立控制相同，CPU／RAM／port負對照有效，最後到(3,9)。四段沒有新完整身體貼圖來源；正式43PBL＋256MAZE／41PNG、敵人24/360、ALLY3/31保持。這是來源探索，未驗新HD、自然戰鬥、從開機或GUI整屏。研究038 §183。

既有低血量路線的等價F1／Space試跑停止線保持。本次依READY014已有玩家功能，在真正GUI按F7恢復HP28→40、能量6→30；Left／Up／Left／Up抵達(0,7)的Jaxemo，實際敵人HP350。F8於第665185906步削弱敵人，隨後原版完成戰鬥；再F7恢復到上限，F10於667962714保存。F7／F8的寫入依014明確標為輔助，不宣稱原版自然戰鬥或沒有HP寫入。GUI起點未另作同指令數全機器重播，螢幕只目視調查，未宣稱960×600整屏HD通過。

原始輸入PW.EXE SHA-256 88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49。Pionn起點pionn-rusteck-exit-v1-event-observed.state SHA 990a39e7f186cfb5b497daa9f3c944c323e15578faa5f4bd134f8e8d9bb45e15。新GUI保存點SHA c23794070b63872518e8718d21e788bc209e01d577dcd680db7c544f14ee81f6、seed4FB0；只固定保存點，不重擲。後續四段共13個原版鍵、26個按下／放開事件，沒有新輔助或RAM／HP／座標／seed注入。終點依序677000000、701000000、729000000及763000000；位置依序(0,8)、(0,8)、(3,8)、(3,9)。source觀察範圍沿原版runtime0161:8705→8751、DS:BX來源與CX×4座標，非IDA ea；沒有新貼圖，不推論尚未觀察的八圖庫或ALLY用途。

沿用image psychicwar-go-ebiten:latest SHA 083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7、Go1.24.13、Python3.11.2、ImageMagick6.9.11-60。新GUI核對既有前端348份實際來源；二進位SHA 2ee43d77763c89d46c1092a3269774049477acf09c36db08bd0d9034870c63a4。四段沿用唯讀來源觀察器，原版素材唯讀、全部容器非root有界，未測音訊／效能。獨立驗證首輪誤把Go []uint8的base64字串當陣列，第二輪容器cwd錯誤；失敗腳本與收據保留。按正確JSON格式及/src工作目錄在同image乾淨重跑，四組44欄位／DOS與有效負對照通過，沒有產品程式修正。

研究根workplace/ida/hd-ally-recruit-20261004/：rusteck-assisted-source-gui-v1-20261006.py／.sh與同名GUI目錄保存execution、terminal、按鍵record、前端log、八份state／PNG及座標讀值。rusteck-assisted-follow-v1-20261006.py、四份follow-*-run、event、observed／control、independent及preview保存逐段證據；verify-rusteck-assisted-follow-v1-20261006.py產生rusteck-assisted-follow-proof-v1-20261006.json，完整輸入SHA及限制在該檔。finalize-rusteck-style-reconfirmed-v1-20261006.py保存本批私有快照與清單。所有原版、PNG、state只留本機。#34保持OPEN、#44不改；未commit／push／PR／發行，正式接受數與完整HD驗收範圍不變。

## 184. ENEMY08十二姿勢草稿更新與素材保全（2026-10-06 07:38）

【confirmed：原版解碼、參照及檔案／固定轉檔。正常圖庫RAM、動作與正式美術接受未完成。】

ENEMY08十二個完整姿勢新增兩輪草稿：八份內建imagegen原生、二十四張72×96素材與完整提示已保全。原版兩份解碼器及既有參照逐像素相同；24/24份完整RGBA重生差0，錯濾鏡、單位元組及三姿勢相異檢查通過。第二輪修掉部分格子塗色與階梯輪廓，group1／2六張列優先草稿；group0仍有方格色區、group3有碎斜線，全部正常來源與構圖待驗，停止同類重抽。原版#12–#14解碼24×24但載入器讀384位元組，尾段正常RAM未知，保留原版。正式43PBL＋256MAZE／41PNG、敵人24/360、ALLY3/31保持；024 §1.43 DRAFT、研究038 §184。

原版ENEMY08.PBL SHA-256 eb4e8673858a5425bba68edba74cad1138caf2c64d58d429e40c7d7a7ec072a7，共30張。兩份既有解碼器tools/pbl.py與tools/hd/explore_sprite_sources.py逐份核對#0–#11，12份既有24×32參照RGB亦相同。檔案offset、區塊終點、區塊長度、實際解碼read-end及lookahead分列於ENEMY08-reference-proof-v1-20261006.json，工具版本與SHA在本批verification及final-audit保存。#12–#14短圖契約不因生成圖而升格；原版C6與尾段停止線沿§166。

第一輪使用原版位置／色區及既有ENEMY00-group1賽璐珞線條參照，四個內建imagegen三格輸出；原生仍有像素階梯與棋盤色面。第二輪以這四份原生為編輯目標，原版只提供位置、配色及裁切，既有定案素材只提供線條，修自然曲邊與平坦色面，不新增臉、牙齒、武器或玻璃。四份修稿均保全，兩輪提示完整保存。人工目視已看八份原生及四份第二輪比較圖；第二輪部分方格仍殘留，優先草稿不代表最終美術完成，不再以相同提示重抽。

全部八份原生按固定三等分格界裁成整格，再完整Lanczos轉72×96；沒有格內人物縮放、平移或手工像素修補。所有24份PNG32全不透明、整份RGBA重生差0；24份實際錯Point濾鏡皆有差異、24份實際expected[0]翻位元各差1，每組三份圖面兩兩不同。這只驗圖面流程，不稱原版動作或正常GUI通過。RGB>16包圍盒僅供診斷，不訂成全角色容差；第二輪圓形三格均[0,0,71,74]，保留原版底部7列最近鄰3倍的21列黑區，完整輪廓仍須正常來源驗收。

驗證首輪將PNG32檔案與ImageMagick直接RGBA量子輸出相比，首格4736 bytes差1；對同一完整預期先依素材契約編成PNG32再解回RGBA，差0。失敗版本與診斷保留，不放寬容差。修正同image／命令後24份完整差0及負對照通過；這是驗證腳本的8-bit序列化差異，沒有修改生成圖來湊比對。提示準備另誤呼不存在pbl.offsets，改用已驗strict_pbl提供的偏移，不另猜解析器。

原始及HD素材全部留本機。既有image psychicwar-go-ebiten:latest SHA 083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7、Python3.11.2、ImageMagick6.9.11-60；Docker非root、有界、原版與生成輸入唯讀，未測遊戲GUI、音訊／效能或新來源分支。正式程式、43筆主題、譯文與字型不變；美術已定案不重問。#34保持OPEN、#44不改；未commit／push／PR／發行。

素材入口workplace/hd/redraw/ENEMY08-reference-proof-v1、prompts-v2／v3、generation-outputs-v2／v3、group0–3-generated／frame／preview／comparison／provenance-v2／v3、all-comparison-v2-v3、draft-selection-v3及art-verification-v2-v3，日期後綴20261006。研究根workplace/ida/hd-ally-recruit-20261004/：prepare-enemy08-reference-v1、prepare-enemy08-prompts-v2、prepare-enemy08-art-v2-v3、verify-enemy08-art-v2-v3及finalize-enemy08-art-v1-20261006.py；失敗verify版本與enemy08-art-verifier-failure-v1保留。私有快照、完整輸入SHA、封存及最終文件SHA由enemy08-art-source-manifest-v1與final-audit-v1保存。舊20261001稿與其原生／提示保持，不覆寫。

## 185. Rusteck上下層真正GUI來源探索（2026-10-06 07:59）

【confirmed：實際GUI按鍵、原版狀態、三段同指令數重播及獨立44欄位／DOS。新sprite美術、全螢幕HD與從開機未驗。】

Rusteck新增三段真正GUI來源路線：下層(3,9)走到(1,10)電梯，正常往上到(1,2)並離開，再走到(3,2)第二座電梯。共26個原版鍵／52個事件，三個終點與無觀察器控制、實際GUI各44機器欄位及完整DOS相同，CPU／RAM／port負對照有效。三次GUI正常退出0；本批沒有新F7／F8、RAM、HP、座標或seed注入，沿038 §183輔助起點。未見新敵人或ALLY完整身體來源，正式43PBL＋256MAZE／41PNG、敵人24/360、ALLY3/31保持；研究038 §185。

美術方向正式定案，沿AGENTS §12與024 §1.26，不設美術提問或逐張批准。本批只執行既有前端、主題與原版鍵。正式程式、主題、譯文及字型不變，沒有新增DRAFT功能進production。先前Pionn起點及GUI F7／F8輔助的限制仍沿§183，不改稱自然戰鬥或全程未曾修改HP。

### 185.1 原始輸入與三段路線

PW.EXE SHA-256 88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49。初始rusteck-assisted-follow-southeast-v1-20261006-event-observed.state SHA 08f02cc2ed29348fdb535d50b896f1f6e4d2dc57cc36f7348041acfdf839a92f，第763000000步、area4、(3,9)、朝東、HP40／能量30、敵人HP0、seed4FB0。全部起點由前段實際原版保存點固定，不重擲。只讀player_bytes的已定位place欄位與轉向後forward_flag，不把未知其他座標或未證實牆位元當通行判準。

| 實際GUI | 原版鍵 | 保存終點 | 來源結果 |
|---|---:|---|---|
| 下層走廊v1 | 11／22事件 | 第771174319步，(1,10)電梯 | opened ROOM0.PBL；完整身體draw 0 |
| 電梯往上v2 | 3／6事件 | 第778113650步，(1,2)朝東離開 | opened ROOM0.PBL；完整身體draw 0 |
| 上層走廊v2 | 12／24事件 | 第788639566步，(3,2)電梯 | opened ROOM0.PBL；完整身體draw 0 |

第二段只有Down、Return、Return。到達上層後等原版離開選單完成才退出；第三段初始再等原版離開動畫完成，從(2,2)出發。三段各與實際GUI保存終點同指令數重播，再與沒有觀察器的控制比較，共6組獨立44欄位／完整DOS比較，實際CPU／RAM／port翻位元負對照有效。只比較三個終點，不稱全部31份中間保存點皆已全RAM驗證。沒有新敵人／ALLY完整身體貼圖；觀察器只保留最多24筆完整身體，並非ROOM一般貼圖來源收據。因此開過ROOM0.PBL不替代ROOM5完整來源或新迷宮全景驗收。

目前可接續分岔保存點rusteck-corridor-gui-v2-20261006/key-02.state，SHA a9a68aefa13dd9e5dbc007ea2df85f9a679d32b8910e2cbfdd116d5a451aa222，第781258772步、(2,1)朝北；向西尚未試。末點rusteck-corridor-gui-v2-20261006/key-12.state SHA 35ead62802b6d8f4db4946d8cdd4895e9bcce34cfa70c48edf9ea1041ee9ff27，剛進入電梯，不直接盲送選單鍵。更早已到達的電梯與單條走廊不重複當新sprite完成。

### 185.2 工具、失敗分類與停止線

image psychicwar-go-ebiten:latest SHA 083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7，Go1.24.13、Python3.11.2、ImageMagick6.9.11-60、真正Xvfb／xdotool視窗。既有前端SHA 2ee43d77763c89d46c1092a3269774049477acf09c36db08bd0d9034870c63a4，348份實際非標準編譯來源逐份核對。既有觀察器SHA ae0e8d2f1d5d3424cf7fefe6d43a4a27ea4f7661dc35deb701ecb33e9f934a7d；位址為runtime0161:8705→8751及DS:BX，不是IDA ea。全程非root、有界、network none、原版唯讀；每個Xvfb有trap。未測音訊／效能或封包。

電梯GUI v1在剛進入畫面的保存點過早送Down，畫面游標仍留「離開」，Return實際退出電梯，測試預期上層選單而失敗。v1的8份PNG、state、log與退出碼保留。v2在相同初始state等原版畫完再送鍵，三次GUI正常退出0；這是探索腳本時序問題，不寫成產品缺陷。電梯重播v1已完成兩組44欄位／DOS比較，但收尾讀取不存在的走廊graph欄位而失敗；改成可選graph，保留v1，於同image乾淨重跑v2並通過。來源及重播種子保持，不挑碰巧結果。

三批PNG只用於GUI路線目視調查，部分保留更早Pionn起點畫面內容。未對完整960×600獨立合成，不宣稱新HD整屏、自然戰鬥、完整動畫、從開機或平台驗收。正式數保持，全部336敵人、28ALLY圖號、小圖與效果等未完成項仍沿worklist。美術不重問。

### 185.3 精確入口與保全

研究根workplace/ida/hd-ally-recruit-20261004/：rusteck-corridor-gui-v1／v2-20261006.py及.sh；rusteck-elevator-up-gui-v1／v2-20261006.py及.sh；各同名目錄的execution、record、terminal、frontend.log、state、xlate與PNG。Docker掛/src唯讀、研究根可寫、workplace/gomodcache→/gomod唯讀、workplace/original→/orig唯讀，執行sh對應.sh；每批GUI memory1g、cpus1、pids128、外層timeout80s。重播執行python3 rusteck-corridor-replay-v1-20261006.py，或rusteck-normal-gui-replay-v2-20261006.py傳elevator-up-v2／corridor-v2；memory512m、cpus1、pids64、外層timeout80s、cwd/src。完整argv及實際按鍵絕對步數在各record／run中。

證據rusteck-corridor-proof-v1-20261006.json、rusteck-elevator-up-v2-replay-v2-20261006-event-proof.json、rusteck-corridor-v2-replay-v2-20261006-event-proof.json。私有保全與完整輸入SHA由finalize-rusteck-corridor-v1-20261006.py、rusteck-corridor-source-manifest-v1-20261006.json及rusteck-corridor-final-audit-v1-20261006.json保存。原版、PNG、state只留本機。#34同步進度並保持OPEN，#44不改；未commit／push／PR／發行。

## 186. ENEMY09同風格草稿與Rusteck分岔停止線（2026-10-06 08:20）

【confirmed：原圖解碼與唯一性、固定素材轉檔、實際GUI及同指令數狀態。正常ENEMY09圖庫RAM、構圖、動作與正式呈現未驗。】

ENEMY09新增五組十五姿勢草稿及第0組一輪修稿，六份內建imagegen原生、十八張72×96與完整提示已保全。15份原圖兩解碼／既有參照相同，全391圖庫唯一；18/18完整RGBA重生差0，錯濾鏡、錯裁格、實際單位元組及三姿勢相異檢查通過。第0組仍有方格感，停止同類重抽；全部正常RAM、構圖與動作待驗，024 §1.44 DRAFT。Rusteck上層(2,1)西側確定是牆、東側通往已走區，三個GUI轉向鍵與控制／實際GUI各44欄位及DOS相同，負對照有效，沒有新sprite來源。正式43PBL＋256MAZE／41PNG、敵人24/360、ALLY3/31保持；研究038 §186。

### 186.1 原始來源及製作

原檔ENEMY09.PBL SHA-256 f6e09a218300e9848360493ecac122617639475106753350944ef078b91e71d9，實際30張。tools/pbl.py與tools/hd/explore_sprite_sources.py的strict_pbl逐份核對#0–#14色號；既有24×32參照RGB相同，五份三格原版參照保存。每張原始file offset、block_end、解碼read_end及lookahead各自列在ENEMY09-reference-proof-v1-20261006.json，不混為runtime或IDA位址。全部391圖庫逐份解碼，這15張完整來源各只匹配自身檔／圖號；其他15張小圖沒有因此完成或排除。

依使用者已定案的自然曲邊、日式科幻賽璐珞及原位比例製作五組三格。原版參照只管造型、色區、姿勢與裁切；ENEMY00-group1-generated只管已接受風格，不借用其角色。五份內建imagegen原生1881×836，格界0／627／1254／1881，每格整幅Lanczos轉72×96，不作格內人物移位、縮放或手工像素修補。第0組另外以首稿為目標作一輪曲線修正；原生與舊稿保留。六份完整提示、全部輸入角色、PNG及原生SHA逐一保全，沒有換CLI或model。

人工已看五份原版參照、六份原生、五份首稿比較及第0組修稿比較。第0組修稿只有部分外緣較圓，內部方格及階梯仍明顯，停止同類提示重抽。第1組中部裝甲與黑缺口、第2組圓形細節、第3組頭盔／雙臂／胸腰及第4組手腳範圍仍待原版構圖核對。這是代理自主驗收的限制，不是待使用者批准。優先草稿為第0組v2與其他四組v1；沒有選入正式主題。

### 186.2 有限素材驗證

18/18完整PNG32八位元RGBA重生差0，全不透明、尺寸72×96；18份實際Point錯濾鏡及8原生像素錯裁格都有差異，18份預期首位元組翻位元各差1。六份原生內三張RGBA兩兩不同，只證明三張圖不同，不宣稱0→1→2→1→0原版動作。RGB>16包圍盒是診斷，原始bbox另存；第2組中／右格頂部與第3／4組底部等差異不靠通用容差放行。正常來源與全GUI尚未驗，不由重生差0增加正式敵人數。

沿既有image psychicwar-go-ebiten:latest SHA 083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7、Python3.11.2、ImageMagick6.9.11-60。原版與生成輸入唯讀，所有轉檔／驗證在非root、有界Docker執行；純黑不透明背景按既有素材契約，不把原版色號10猜成透明。

### 186.3 Rusteck已試分岔

起點沿§185上層key-02.state，SHA a9a68aefa13dd9e5dbc007ea2df85f9a679d32b8910e2cbfdd116d5a451aa222，第781258772步、area4、(2,1)朝北、HP40／能量30、enemy_HP0、seed4FB0。真正GUI只按Left、Right、Right，各按住至少0.18秒，另用既有F10保存。向西後forward_flag2，沒有送Up撞牆；朝東forward_flag0，通往既有(3,1)，沒有重走。沒有新的F7／F8、RAM、HP、座標或seed注入，§183更早輔助起點限制保持。

終點第785098263步、(2,1)朝東；原版觀察器保存終點與無觀察器控制、實際GUI各44欄位及完整DOS相同，實際CPU／RAM／port負對照通過。沒有新完整身體貼圖或開檔；三鍵六事件、GUI自然退出0。只驗該分岔，不稱整層／全部Rusteck已探索，也未驗完整960×600HD、自然戰鬥或從開機。原版PW.EXE SHA88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49；位址runtime0161:8705→8751、DS:BX、320×200座標，不是IDA ea。

西牆及東側已訪區不再重試。上層(2,2)向南尚未驗，保存點rusteck-corridor-gui-v2-20261006/initial.state SHA2339cb9e64e07249b3030d8d75291863d62c8d676b8f96ee4fc3e21be93c8c68，可用原版轉向後的通行旗標接續，不猜牆位元或其他座標。未把無新來源的走廊探索當作新HD完成。

### 186.4 重跑與保全入口

素材均在workplace/hd/redraw/：ENEMY09-reference-proof-v1、prompts-v1、group0-prompt-v2、generation-outputs-v1／v2、六份provenance、selected-drafts-v1、art-verification-v1及all-comparison-selected-v1，日期後綴20261006。完整提示在prompts／provenance／selected-drafts，不僅保存摘要。原生與18個完整PNG留本機。

研究根workplace/ida/hd-ally-recruit-20261004/：prepare-enemy09-reference-v1、prepare-enemy09-prompts-v1、prepare-enemy09-art-v1、prepare-enemy09-group0-art-v2及verify-enemy09-art-v1-20261006.py；readonly/src、redraw可寫、原生generated_images掛/generated唯讀。驗證memory512m、cpus1、pids64、外層timeout60s、network none。GUI重跑rusteck-corridor-gui-v3-20261006.sh，/src唯讀、研究根可寫、原版/orig與gomodcache唯讀；memory1g、cpus1、pids128、timeout80s，Xvfb有trap。重播python3 rusteck-normal-gui-replay-v3-20261006.py corridor-v3，cwd/src、memory512m、cpus1、pids64、timeout60s。完整argv、絕對指令數按鍵及工具SHA在execution／record／run／proof。

私有增量保全finalize-enemy09-art-v1-20261006.py、enemy09-art-source-manifest-v1-20261006.json與enemy09-art-source-snapshot-v1-20261006.tar.gz，最終文件／遠端／容器與擁有權核對enemy09-art-final-audit-v1-20261006.json。正式程式、43筆主題、譯文與字型未改；未測音訊／效能或封包。#34保持OPEN，#44不改；未commit／push／PR／發行，完整HD目標保持。

## 187. ENEMY10／11同風格草稿與記者室入口限制（2026-10-06 08:44）

【HD-RUSTECK-WAIT-VIEW-01】 本節「黑視野」及電梯仍阻塞的停止判斷已由§188訂正；保留當時圖片、輸入及原始收據。

【confirmed：原圖解碼、靜態來源唯一性、固定轉檔與實際GUI同指令數狀態。正常新圖庫RAM、構圖、動作及房間入口未知。】

ENEMY10／11新增十組三十姿勢草稿，另修ENEMY10第4組曲線；十一份內建imagegen原生、三十三張72×96及完整提示已存本機。30原圖兩解碼及既有參照相同、全391圖庫各自唯一；33/33完整RGBA重生差0，錯濾鏡、錯裁格、實際單位元組及三姿勢相異負對照有效。第4組修稿保留下緣片段，正常來源、構圖、動作與接入仍待技術驗證，024 §1.45 DRAFT。Rusteck南側22個實際GUI鍵走到(7,1)記者室位置，等待段及一次Enter均無新貼圖／開檔；三段各與控制、實際GUI44欄位及完整DOS相同，負對照有效。原電梯文字及黑視野仍在，房間入口未知，不重試猜鍵、不稱新房間美術完成。正式43PBL＋256MAZE／41PNG、敵人24/360、ALLY3/31保持；研究038 §187。

### 187.1 原始來源與完整素材

ENEMY10.PBL SHA-256 d371d4065a76a374329128c6d4a35f52f578322f7f2e4458252592b26fa0a482；ENEMY11.PBL SHA-256 f5c29f254baf0ec1ef2dc9db61596efbd2cfb941ed992534358899386a29dc44。每檔30圖，#0–#14兩份解碼器逐色號相同，既有24×32參照RGB相同；全ALLY／十二ENEMY共391圖逐份解碼，這30份各只匹配本檔本圖號。原始file offset、block_end、read_end及lookahead分列在兩份reference-proof，不混為runtime或IDA位址。#15–#29沒有排除於完整目標。

十份原版三格參照均已檢視。內建imagegen生成十份三姿勢原生，ENEMY10第4組另以首稿作單一曲線修正。第4組原版是下緣彩色拱形片段，上方大片黑色；首稿仍有階梯／棋盤格，修稿改為連續曲線與整片賽璐珞色區。沒有補成人物、加五官、移位、縮小或手工修像素。原版參照管構圖與色區，ENEMY00-group1-generated只管已定案風格。全部提示、參照角色與SHA、原生及轉檔argv均逐一保存，原生尺寸及三格界依實際PNG測量。

所有輸出與比較已檢視。優先草稿採ENEMY10第4組v2，其他九組v1；只是草稿選擇，沒有正式接入。ENEMY10其他組的青白／紅粉裝甲、黑缺口及動作，ENEMY11頭部外緣、手臂、下端裁切及色區仍須原版構圖核對。正式同風格美術不等待使用者逐張批准。沒有新增全部sprite的容差。

### 187.2 有限素材驗證

ENEMY10六原生／18轉檔、ENEMY11五原生／15轉檔，全33/33完整八位元PNG32 RGBA重生差0，尺寸72×96且alpha全255。實際Point錯濾鏡、裁格向右8原生像素均有差；預期第一位元組翻一位各差1。十一組三張RGBA兩兩不同，只證明畫面不同，不證明原版往返動作。RGB>16包圍盒僅作診斷，不能把構圖差異靠門檻放行。正常來源／GUI尚未驗，正式數不增加。

既有工具image SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7，Python3.11.2及ImageMagick6.9.11-60。所有搜尋、轉檔及驗證在非root Docker，原版及生成目錄唯讀，只有既有研究區可寫。純黑背景不把色號10猜成透明。

### 187.3 真實GUI南側通路及停止線

起點沿§185 rusteck-corridor-gui-v2-20261006/initial.state，SHA2339cb9e64e07249b3030d8d75291863d62c8d676b8f96ee4fc3e21be93c8c68，第779532087步、area4(2,2)朝東、HP40／能量30、enemy_HP0、seed4FB0。既有前端實際348依賴逐SHA核對，未加F7／F8、RAM、HP、座標或seed注入；更早§183輔助起點限制保持。

GUI沿原版轉向後forward_flag0／2決定是否前進，不猜牆位元。22鍵／44事件經(2,3)、(3,3)、(4,3)、(5,3)、(5,2)、(5,1)、(6,1)到(7,1)。終點第797150022步，place=1，HUD文本來源I_MAP04.BIN:0288顯示記者室；state SHA42b0642880ead06268d9b18309410a1fc7746f88080e669508ea8db0f523ce77。這是位置／名稱證據，非新房間源。畫面仍顯示原電梯『An elevator...Leave』與黑視野，觀察器沒有新完整身體貼圖或開檔。

再載該state，在GUI等待後保存第801346396步，SHA785ae4bd580eb0d287d5d58881dab82b38c73f46d43c5f15b2817ed827772bfb；不按原版鍵。下一段依畫面Leave按一次Return，保存第805241867步，SHAda32fb61e619f2871f8b8a164e7739fe46de3901a14fbe14586d654075172ade。三段均自然退出0，位置／HP／seed保持，沒有新貼圖或開檔。原版控制與實際GUI共6次完整44欄位及DOS比較相同，CPU／RAM／port實際負對照有效。只證明有限輸入與狀態，不宣稱已進入記者室、正常房間美術、整屏HD、自然戰鬥或從開機。

無鍵record.events由Go輸出null，首次重播讀取誤當list；Enter段首次重播的GUI路徑後綴錯誤。均為驗證腳本問題，修正後用相同容器、輸入與工具乾淨重跑通過，沒有更動產品或條件放寬。等待及Return皆未取得新來源後停止猜鍵，下一步先查該原版保存點的房間入口狀態與控制流，不用位置名字代替素材來源。

### 187.4 重生及索引

素材在workplace/hd/redraw/：ENEMY10／11-reference-proof-v1、prompts-v1、generation-outputs-v1、十份provenance-v1、selected-drafts-v1、all-comparison-selected-v1；ENEMY10-group4-prompt／generated／comparison／provenance-v2、generation-outputs-v2、art-verification-v2及ENEMY11-art-verification-v1，日期後綴20261006。全部完整提示在prompts及provenance，十一份原生／33PNG留本機。

研究根workplace/ida/hd-ally-recruit-20261004/：prepare-enemy10／11-reference-v1、prepare-enemy10-11-prompts-v1、prepare-enemy10／11-art-v1、prepare-enemy10-group4-art-v2、verify-enemy10-art-v1／v2及verify-enemy11-art-v1-20261006.py。readonly/src、redraw可寫、generated_images掛/generated唯讀，memory512m、cpus1、pids64、timeout45s、network none；完整PNG32驗證不借用截圖作答案。

GUI入口rusteck-corridor-gui-v4、rusteck-journalist-gui-v1／v2-20261006.sh，readonly/src、研究根可寫、原版/orig及gomodcache唯讀，memory1g、cpus1、pids128、timeout80s內，Xvfb有trap。重播rusteck-normal-gui-replay-v4-20261006.py corridor-v4，以及rusteck-journalist-gui-replay-v1／v2-20261006.py journalist-v1／v2；cwd/src、memory512m、cpus1、pids64、timeout85s內。execution／record／run／proof保存完整argv、絕對指令數按鍵及工具SHA。原版PW.EXE SHA88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49；觀察runtime0161:8705→8751、DS:BX、320×200座標，不是IDA ea。

本批收尾與保全入口finalize-enemy10-11-art-v1-20261006.py、enemy10-11-art-source-manifest-v1-20261006.json、source-snapshot與final-audit，已由CONTEXT／024索引。正式程式、主題、譯文及字型不改；#34保持OPEN、#44未修改，未commit／push／PR／發行。完整HD及來源／動作／GUI／DAT／封包閘門仍待完成。

## 188. 原版等待點勘誤與ENEMY04第三組正常來源（2026-10-06 09:28）

【confirmed：四保存點零步資料、原始指令、真正GUI及完整同指令數狀態。新HD呈現及DAT未驗。】

已訂正黑迷宮／電梯仍阻塞的判斷：原版迷宮在(4,124)、72×72，有8種色號；四份保存點均位於CODEH:01CC的opcode94輸入等待。沿正常GUI走到新電梯(7,0)，上樓到(15,0)，離開後在(14,1)取得ENEMY04 #9–#11。五段共18普通鍵／36事件，10次完整44欄位及DOS比對相同，CPU／RAM／port負對照有效；五次原版貼圖確認9→10→11→10→9、384-byte來源及完整貼圖差0、矩形外變動0。效果覆蓋四份身體各72像素，等待後戰敗；不稱完整HD、勝利或新DAT通過。正式43PBL＋256MAZE／41PNG、敵人24/360、ALLY3/31保持；研究038 §188。

### 188.1 最小原始證據與位址空間

原版PW.EXE SHA88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49；解壓PW_UNP.EXE SHAfd5115b91f014c47a1fb1a6e2ecfd645e293264fe614a4b350e92637cad2fbd9；正式417函式.i64 SHA4db51a199f5913c52dc4964557b9a4aafb965754d663cb0173c9dfdfe9a00c56。IDA9.4使用ida-pro-9.4-idapython:locked-v1 image SHA6f6d59af49d0008c4109a5295b5f374bdc007e2d1ab28cb9de08779584de2780，資料庫唯讀掛載後複製至容器/tmp，沒有改名或覆寫正式庫。

IDA ea 12CE0h–12CF1h的opcode94分支共17 bytes，六指令逐項與EXE及四份runtime CS相同。直接call的IDA ea12CE1h bytes E81338，返回12CE4h；runtime基底10510h，分別為0161:27D1h／27D4h。MZ file offset另以976+ea−10000h計，不與runtime混用。四份DS:30C6的CODEH前1200h bytes相同，DS:04C6的CODE4實際完整1471 bytes相同；DS:3292對應CODEH:01CCh、opcode94，下一BX3293h。

sub_164F7先保存CX／BX；stack3293h及27D0h是保存的暫存器，runtime0161:27D4h才是該直接call的return IP。初次誤把保存BX當return，查到空區；已由原始call bytes及push序訂正，不推論空區是遊戲缺口。只追查本次玩家等待點，未逐行逆向無關輸入helper。

原版迷宮座標(4,124)、72×72，色號0／1／3／4／9／11／12／14；全黑負對照被拒絕。31份ROOM0完整圖在此保存點均不匹配；HUD記者室名字只證明位置名稱。舊電梯訊息及全身體觀察器沒有新事件，不能證明還在電梯選單，也不能證明新房間美術完成。【HD-RUSTECK-WAIT-VIEW-01】同時掛入§187及000勘誤；CONTEXT及worklist現況已修正。

### 188.2 正常GUI接續與敵人差分

GUI v5四次轉向確認(7,1)南／北阻擋、西可通。v6恢復先前實際路徑stack，九鍵回(6,1)、北至(6,0)、東至新電梯(7,0)，原版開ROOM0.PBL。v7原版Down／Return選往上，等待完成後到(15,0)。v8先Return離開，原版自行移至(14,0)朝西；再Left／Up到(14,1)遇敵。每次啟動均核對正式前端348份實際編譯來源，沒有新F7／F8、RAM、HP、座標或seed注入；承接§183更早輔助起點，不稱自然戰鬥parity或從開機。

可續HD呈現驗證的起點rusteck-corridor-gui-v8-20261006/key-02.state，SHA190ead3529262c146f6d1a8f08a38a5cb332307d56434308e55589a0a786854a，第829390795步，area4(14,1)朝南、HP40／EP30、enemy_HP200、seed9BC1。v9沒有普通鍵，兩秒後HP24、再等待戰敗，不重試調時機或新增作弊。戰敗保存點不當可續戰鬥起點。

ENEMY04 #9／#10／#11在全391圖庫各自唯一。五次來源依序mode0／1／1／1／1，1175:A686、1175:A8C6、0161:374A、0161:374A、1175:A8C6；每次384 bytes與原版完整圖或相鄰XOR差分相同。原版貼圖runtime0161:8705→8751、(32,152)、24×32，完整前後64000色號依原mode重建差0，矩形外變動0。實際來源byte翻位各檢出1像素；省略貼圖差388／74／83／83／74。首份後畫面完整匹配#9，後四份效果覆蓋各72像素，不能把它們當乾淨完整身體或HD通過。四圖庫來源仍沿024 §1.22–1.23既有READY，本批沒有改正式程式／主題／資料格式。

### 188.3 工具、重生與限制

研究根workplace/ida/hd-ally-recruit-20261004/：rusteck-room-entry-ida-v1–v8-20261006.py／json／log、rusteck-room-state-read-v1／v2-20261006.go／bin／json、build-rusteck-room-reader-v1／v2、verify-rusteck-room-entry-v1及rusteck-room-entry-proof-v1-20261006.json。GUI入口rusteck-corridor-gui-v5／v6／v8、rusteck-elevator-gui-v7、rusteck-enemy04-g3-gui-v9-20261006.sh；各terminal／execution／record、完整state及PNG留本機。重播rusteck-normal-gui-replay-v5／v6／v7／v8／v9-20261006.py，參數依序corridor-v5、corridor-v6、elevator-v7、corridor-v8、enemy04-g3-v9。差分核對verify-rusteck-enemy04-g3-v1及rusteck-enemy04-g3-source-proof-v1-20261006.json。

既有psychicwar-go-ebiten:latest image SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7、Go1.24.13／Python3.11.2／ImageMagick6.9.11-60。主工作樹/src唯讀、研究根可寫、原版/orig及gomodcache/gomod唯讀、user1000:1000、network none、cpus1。GUI memory1g／pids128／timeout60s內、Xvfb trap；來源重播memory512m／pids64／timeout65s。IDA memory2g／pids128／timeout60s、官方入口與工具方法見/home/anr2/ida_94_official/skills/use-ida-pro-9-4/SKILL.md。

Go首次預設並行建置碰到pids64，改GOMAXPROCS1及go build -p1同image乾淨重建通過。IDA首次JSON未指定UTF-8、初次stack與flags/data導覽錯誤、CODE4錯假設固定700h長度，均保留失敗工具與收據；依實際bytes及1471長度修正，沒有放寬產品判準。v6本輪根節點曾漏掉先前東向已訪記錄，多轉四次，修復記錄後不重跑已知阻擋。

收尾與私有保全入口update-rusteck-room-entry-v1、preserve-rusteck-room-entry-v1-20261006.py、rusteck-room-entry-source-manifest-v1及final-audit-v1-20261006.json。本節新增工具全由此及CONTEXT索引，不建立第二份目前狀態表。正式43PBL＋256MAZE／41PNG、敵人24/360、ALLY3/31保持；ENEMY10／11草稿沿§187。下一步用上述有血量的正常來源驗ENEMY04 #9–#11候選HD／效果遮擋與GUI，再計正式數。HD美術不重問；#34保持OPEN、#44不改，未commit／push／PR／發行。

## 189. ENEMY04第三組HD限定接入（2026-10-06 09:47）

【confirmed：既有美術來源、原版正常三姿勢、八份完整圖面及有限實際GUI；完整HD、DAT與交付未完成。】

ENEMY04 #9–#11三張HD已限定選入正式本機主題，敵人由24/360增至27/360；主題46PBL＋256MAZE／44PNG、ALLY3/31保持。正常9→10→11→10→9來源沿§188，八份完整圖面、載回／冷載／開關與44欄位／DOS通過；實際Up進戰鬥的24張視窗抽樣中11張可見格吻合，三姿勢皆排除原版及其他兩姿勢，13張未列通過。GUI實際鍵另有兩次完整44欄位／DOS獨立比對，負對照有效。效果遮擋沿原8×8回退，沒有新增作弊；其餘33張既有候選、其他sprite、全動畫、新敵人DAT與封包仍待完成，研究038 §189。

### 189.1 READY契約與美術來源

本批沿024 §1.23及§1.29 READY，使用者HD風格決策沿§1.26，不新增資料格式、位置或玩法。原版ENEMY04.PBL SHA81cb62cf9a8b64a538d09e50b29cbf8ad123b39b6e86979af6423a2aed20f546；正常來源由§188的真正GUI取得，三圖在391圖庫各自唯一，runtime0161:8705→8751、(32,152)、24×32。384-byte完整來源及原版DS／CS XOR循環9→10→11→10→9已核對，沒有以XOR對稱猜動作。

沿既有ENEMY04-group3-v3原生SHA33dd6c2f39074fcd1484bde39a232c009b79d4b7bad47171dfe1a98ec5885c91及provenance SHA3ff1765be1e2ac247f1c9bc28427b2bc9539b674dd0864ec97150b06a3b48812。三份72×96 PNG依序SHAb4187c763ba278fe991da30662e36f2726ad5eca206b5fe460f6dc15c25fbee0、88a55e8193e260ec424092aab5976467cd8d61be6ca0cf33c67df63b7418cf41、31a246e1c6cc803a7073dd5a3c261be42dd43ee6682477b5889d914dc75f4b42。比較ENEMY04-group3-v3-batch-comparison-20261005.png已檢視，保留原版上下裁切、姿勢、原位及自然曲邊，不重畫或手工改像素。

新主題workplace/hd/theme-enemy04-group3-maze-B-v1-20261006/保持父theme-elevator-maze-B-v1-20261006/的43筆及41PNG逐SHA相同，只追加ENEMY04 #9／#10／#11、(32,152)及右上SCREEN錨點(248,0)、72×40。總46PBL＋256MAZE／44PNG，敵人27/360，ALLY3/31／3完整人物；PNG含一張B v9圖集。selection-receipt.json保存父素材SHA、驗證入口及限制，舊主題不覆寫。

### 189.2 同狀態圖面、真正載回及負對照

原版五份貼圖後state沿§188凍結收據；runtime-plan-v1合併原事件並依已核對target_image建索引，不注入原始資料。先冷載第一次完整#9，正常接續五次9→10→11→10→9，再真正載回完整#9、接續下一#10及冷載局部#10，共八份圖面。每份對無HD原版控制44欄位及完整DOS相同，CPU／RAM／port實際變異負對照有效。

完整960×600 RGBA圖面與獨立原版PBL／PNG／8×8期望差0，前五份身體有效格依序12／8／8／8／8。原版戰友初始完整身份按既有逐格規則保留；效果格不匹配時回退原版。冷載局部畫面不猜前姿勢，HD關閉圖面為透明，重開與原圖面相同。省略身體、錯姿勢及實際一byte負對照均有效。此為完整圖面，不是GUI整屏或全部動畫聲明。

Go二進位SHAc541c80c0d68c8978f97ed2b7479ae57a7648313aac1f616471d3da169586567，Go1.24.13，90份實際非標準編譯來源凍結於rusteck-enemy04-g3-style-runtime-v1-20261006-build.json。原版PW.EXE、dosgolem f8c1a6e、主repo527456b及現行正式前端348來源沿§188。正式Go、EXE、RAM、seed、文本與字型未改。

### 189.3 正常GUI呈現與範圍

真正GUI從§188的rusteck-corridor-gui-v8-20261006/key-01.state，第828653523步、(14,0)朝南、HP40／EP30、seed4FB0，以普通Up進入戰鬥。實際record的按下828884231、放開829089349，來源由原始保存點繼承，沒有新的F7／F8或其他RAM寫入。24張960×600視窗抽樣、正常退出0；最終等待戰敗，不稱勝利或自然戰鬥對拍。

11/24張含可辨識的HD原位格，#9／#10／#11皆至少一份。每個匹配24×24顯示格完整RGBA相同，且相同可見格與原版及其餘兩姿勢各有差，避免共用頭部被當成不同動作。其餘13份不列HD通過；數值、效果、舊英文訊息及整屏中文字面不在本次GUI驗收範圍。沒有將同狀態圖面當作整屏GUI，也沒有用攔截器已觸發代替可見圖片。

這段實際GUI Up由原版獨立重播到真正F10保存點，觀察／控制及觀察／實際GUI各44機器欄位與DOS相同，CPU／RAM／port負對照有效。五次敵人貼圖及戰敗戰友消除保留在來源事件，開END0.IBM；不另驗音訊。正常分支與HD可見性已足以限定選入這三張美術，真正DAT、更多效果、其他分支及完整動畫仍另驗。

### 189.4 重生、索引及下一步

研究根workplace/ida/hd-ally-recruit-20261004/：rusteck-enemy04-g3-runtime-plan-v1、style-runtime-v1.go／bin／-build.json、build-rusteck-enemy04-g3-style-v1-20261006.py；執行後style-runtime-v1-20261006/保存八份state、control、frame、rgb、plane及machine。獨立核對rusteck-enemy04-g3-style-independent-v1-20261006.py／json。

GUI入口rusteck-enemy04-g3-style-gui-v1-20261006.sh／py，視窗24PNG、execution／terminal／record、final.state及final-position.json在同名目錄。獨立可見格驗證style-gui-independent-v1-20261006.py／json；真正原版重播style-gui-replay-v1-20261006.py傳enemy04-g3-style-v1，收據rusteck-enemy04-g3-style-v1-replay-v1-20261006-event-proof.json及完整observed／control／GUI machine。正式選入select-rusteck-enemy04-g3-style-v1-20261006.py，資料與限定結果以新主題selection-receipt.json為準。

既有psychicwar-go-ebiten:latest image SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7、Python3.11.2／ImageMagick6.9.11-60；Docker user1000:1000、network none、cpus1，原版唯讀。GUI memory1g／pids128／timeout35s、Xvfb trap；來源及圖面memory512m／pids64／timeout65s，主工作樹唯讀、只寫研究根。建置GOMAXPROCS1／go build -p1、memory1g／pids128／timeout140s，Go依賴cache唯讀，實際来源全凍結，不靠不明主機runtime。

CONTEXT維持唯一目前狀態表，AGENTS、024 §1.29、worklist及#34同步46／44與27/360；#34保持OPEN、#44不改。前43筆ALLY2與電梯樓層DAT證據沿§178／§182，不擴稱新46筆或新敵人DAT通過。其餘33張候選、333敵人圖號、9ALLY完整人物／19其他圖形、小圖、效果、全場景、效能、權利與HD封包仍待完成。未commit／push／PR／tag／發行；原始圖、PNG、完整提示、state及DAT留本機。

本批更新入口update-rusteck-enemy04-g3-style-v1，私有保全preserve-rusteck-enemy04-g3-style-v1-20261006.py及rusteck-enemy04-g3-style-source-manifest-v1／final-audit-v1-20261006.json，全部工具由本節及CONTEXT索引。來源圖面新候選已完成限定接入，後續直接處理其餘sprite，不重新詢問美術。

## 190. HD定案保持及Rusteck續走停止線（2026-10-06 10:13）

【confirmed：四段普通GUI按鍵及獨立完整狀態；輔助段GUI重播差異原因unknown。沒有新HD美術接入。】

HD美術正式定案與不再詢問的授權保持。Rusteck四段實際GUI共7個普通鍵／14事件、8次完整44欄位與DOS比對及有效負對照通過。F3分支戰敗後停止；既有F7／F8取得實際戰後保存點，但兩次輔助重播與GUI的6項機器欄位仍不同，不列通過。後續普通鍵走到另一座電梯並上樓，離開後仍為已接入ENEMY04 #9，不增加正式圖數。正式46PBL＋256MAZE／44PNG、敵人27/360、ALLY3/31保持；研究038 §190。

### 190.1 授權、起點與普通F3分支

使用者再次明示HD美術正式定案，不再詢問。既有AGENTS §12及024 §1.26保持，造型、配色、自然曲邊、B微明暗、B透光、原版位置／比例／姿勢／8×8與人物在後／框線在前自主實作及修正；本批不新增美術或產品取捨。

沿§188的rusteck-corridor-gui-v8-20261006/key-02.state，SHA190ead3529262c146f6d1a8f08a38a5cb332307d56434308e55589a0a786854a，第829390795步、(14,1)朝南、HP40／EP30、enemy_HP200、seed9BC1。真正GUI只送一次F3，保存點第833264245步HP0，沒有成功續走，不調時機或重試。rusteck-escape-gui-v10-20261006/保存實際record、F10 state、PNG及自然退出0。原版重播1鍵／2事件，觀察／控制及觀察／實際GUI各44機器欄位和完整DOS相同，CPU／RAM／port負對照有效。此為原版分支結果，不記為前端漏鍵或HD缺陷。

### 190.2 既有輔助操作與未通過的重播

沿READY014，實際GUI F7→F8→Space→F7，未改座標、seed、隊伍、EXE或資料。前端紀錄記錄F7第829803966步、F8第830703924步、F7第833265296步；Space按下831754693／放開832502095。實際戰後F10第833635210步、(14,1)朝南、HP40／EP30、enemy_HP0、seed7582，SHA829da1908941ddf587b10642572cebe05a76362c98f7e1751c5b3450687becc1。後續普通Up確實能移動，只作來源探索起點，不稱自然勝利或整屏HD驗收。

assisted-replay-v2從before-assistance.state、第829390801步起跑；v3改用record.from的原始key-02.state。兩次觀察器與內部控制相同，但對實際GUI均有Mem／R／IP／Flags／PortTicks／PortsIn六項差異，完整DOS相同。獨立comparer沒有放寬欄位，兩次均拒絕通過。兩份diagnostic保留40個RAM byte及原始暫存器／port差異。v2輸出的觀察／控制另經獨立44欄位、DOS及CPU／RAM／port負對照通過，不能取代GUI不符。失敗原因unknown；不外推原版玩法差異，也不反覆調按鍵、seed或比較範圍。未執行省略F8分支，沒有宣稱該負對照通過。

### 190.3 戰後普通GUI三段續走

從實際戰後保存點啟動正式本機46筆主題，-cheat未啟用。v11三個普通鍵從(14,1)走到(14,2)再至(15,2)電梯，觀察ROOM0.PBL開檔。v12以原版Down／Return選上樓，到(7,2)朝東、place26h。v13以Return離開，到(6,2)朝西並遇敵，第一份384-byte完整身體唯一匹配ENEMY04.PBL #9，下一次為原版XOR差分；沒有新sprite身份，不追加新圖。

三段共6鍵／12事件，觀察／控制及觀察／各自實際GUI保存點共6次44欄位及完整DOS相同，CPU／RAM／port負對照有效。加普通F3分支，本批7鍵／14事件及8次獨立完整狀態比較。seed從各實際state固定，未修改或重擲。不能把這些普通續段的通過補到輔助前段，也不外推自然戰鬥、全動畫、DAT或封包。

v13末點after-leave.state第846609357步、HP32／EP30、enemy_HP199、seed2B33，SHAfc747d18aadf8fc388984570b95f90d86755c5e1400675f9a6640a4f6a00ce79。這條電梯路線已見同一ENEMY04第三組，停止重試同分岔。下一輪可從既有未訪分岔查其餘33張READY候選與ALLY正常來源；本批不將新圖像的數字增加。

### 190.4 來源、工具與重生入口

所有新證據在既有研究根workplace/ida/hd-ally-recruit-20261004/。GUI入口rusteck-escape-gui-v10、rusteck-enemy04-g3-assisted-win-gui-v2、rusteck-corridor-gui-v11／v13、rusteck-elevator-gui-v12的20261006.py／sh，同名目錄保存execution、terminal、record、F10 states、位置JSON與PNG。普通重播rusteck-normal-gui-replay-v10-20261006.py傳escape-v10；v11傳corridor-v11、elevator-v12、corridor-v13。四份*-event-proof.json保存8次comparer收據。輔助失敗入口rusteck-enemy04-g3-assisted-replay-v2／v3-20261006.py、positive-v1／v2事件、diagnostic-v1／v2及observer-control-v2-20261006.json。

正式前端SHA2ee43d77763c89d46c1092a3269774049477acf09c36db08bd0d9034870c63a4、348實際非標準編譯來源沿elevator-english-frontend-v1-20261006-build.json；普通來源觀察器SHAae0e8d2f1d5d3424cf7fefe6d43a4a27ea4f7661dc35deb701ecb33e9f934a7d，輔助觀察器SHA77d61c4acd799bdabb4ecae8b0a900c457dc57f0a1da198740936ced2db06bd8。獨立comparer SHA6e4a6979d5fdef33b4c204982583e9952af8dbab834ac39650c6ea7d009c1e73、診斷binary SHA2a1908ea92487eadac39efd1184a48a2aef89160eb7651653e53afdbc2799e9d。PW.EXE SHA88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49，原始資料唯讀。工具位址為原版runtime CS:IP／DS:BX及線性Mem，非IDA ea。

沿psychicwar-go-ebiten:latest SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7、Go1.24.13／Python3.11.2／ImageMagick6.9.11-60。Docker user1000:1000、network none、cpus1；GUI memory1g／pids128／外層timeout40–65s、Xvfb trap；重播memory512m／pids64／timeout80–160s；讀取及更新memory256m／pids32／timeout20s。主工作樹通常唯讀，只寫研究根；文件更新容器原始資料另唯讀掛載。未建新image。

更新／保全入口update-rusteck-continuation-v1-20261006.py、preserve-rusteck-continuation-v1-20261006.py；results、source-manifest／snapshot及final-audit-v1-20261006保存精確輸入SHA、封存、遠端#34讀回及擁有權。CONTEXT唯一目前狀態表、worklist及WORKLOG同步；正式46／44、27敵人／3ALLY保持。#34仍OPEN，#44不改，未commit／push／PR／tag／發行；原生圖、PNG、state、DAT與完整提示留本機。完整HD目標繼續，藝術不重問，未知技術條件另列。

封存時發現普通重播v11腳本在第一段完成後新增v13入口。已逐SHA復原先前版本rusteck-normal-gui-replay-v11-before-v13-20261006.py，SHA3581bb597512f9e640e5d203f29ea05b8fcc81ea7922f61217100aab76e3d82f，與第一段實際輸入收據相同；保留當時路徑及歷史版本映射，沒有改寫原收據或放寬核對。後兩段對目前v11來源核對，全部工具均列入私有增量封存。這是保全來源版本問題，GUI驗證結果不變。

## 191. ALLY左臂修稿與Celtac普通GUI時鐘契約（2026-10-06 10:53）

【confirmed：三份固定轉檔、兩段普通GUI及完整狀態比對。構圖與ALLY正常來源未完成；攻略前置條件為待驗線索。】

沿正式定案風格新增ALLY #7第二稿與#8第二／第三稿，三份完整RGBA轉檔及負對照通過；#8左手對位改善，#7指端仍偏高，九人物草稿共24份原生，未正式接入。Celtac實際GUI選單與出發共7普通鍵，4次完整44欄位／DOS與負對照通過。選單重播的舊時鐘差異已依前端載入契約修正；原版出發遭衛星砲後返回Samar，停止重試同路線，下一步查Sivad防衛控制前置條件。正式46PBL＋256MAZE／44PNG、敵人27/360、ALLY3/31保持；研究038 §191。

### 191.1 三份美術修稿

沿AGENTS §12及024 §1.26定案，未重問美術。內建imagegen以原版、首稿與既有同風格參考生成ALLY07 v2、ALLY08 v2及v3，原生皆1086×1448；整幅Lanczos至72×96，位置、比例與8×8要求不改。完整提示及引用檔在workplace/hd/redraw/ALLY-07-08-prompts-v2、generation-jobs-v2及ALLY-08-prompt-v3、generation-v3-20261006.json。三份原生SHA依序21af5114465255bab5849cfaba2260fd03a5fab49f6de91fe1111236f8b6dc1d、491af94c5ac560ac8dbbcd4933474c6e5ffb8d4de60c61ecdc37ecb28a1dabdc、6bc940e1463754a5c2821378c8f0e254a0ad865333fe86057797553e1440b6e6。

三份完整RGBA差0；錯縮放濾鏡差13573／12529／12902 bytes，單byte負對照各1。兩份PBL解碼仍相同，ALLY.PBL SHA c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219。正式主題每檔SHA保持。ALLY7左指灰色區域原版[9,60,14,65]、v1[10,53,13,55]、v2[8,55,14,60]，仍偏高。ALLY8左臂區域原版[9,18,23,32]、v1[11,14,23,32]、v2[13,18,23,32]、v3[8,18,23,32]。v3改善對位，區域界線只作診斷，不新增容差或宣稱全身構圖通過。九人物草稿共有24原生，正式ALLY仍3/31。

ALLY8 v3首次執行因相對__file__失敗，已將四個自建輸出SHA保存partial-run-v1並逐檔核對UID後重建。修正版使用resolve；第二次外層舊PNG檔案SHA斷言失敗，原生相同但三份壓縮檔bytes不同。當前完整comparison1728×768 RGBA獨立重建差0，frame完整RGBA差0。壓縮bytes差異原因未完全證實，不宣稱PNG檔案SHA不變。兩份失敗分類在ally8-art-v3-partial-run-v1與clean-rerun-proof-v1-20261006.json；原始失敗不記為產品缺陷，不再重複同類指端生成。

### 191.2 普通GUI選單與出發

沿既有jaxemo-attack-party-launch-v1-20261005-event-observed.state，第590000000步、area0(15,14)朝東、HP40／EP30，SHA01f1c95928b3f037af6c17820ff2e27771ad2e4b6d2d9de582587c72ffea33ef。承接歷史起點限制，沒有本批新作弊或座標／seed寫入。真正GUI五次Down各0.18秒並F10保存。選單為Exit Port／Sivad／Zellwal／Rusteck／Celtac；第四Down已選Celtac，第五仍同一項。更正本輪中途六項／第五Down才到Celtac的假說，舊四Down選擇沒有被推翻。已執行GUI腳本scope的六項敘述保留為歷史輸入，本節以實際PNG訂正，不修改凍結腳本。

selected-04.state第595547536步，SHA595d97b85b07a85ed4bfadf32dbd34bf95c4089a5fb7a7ed406c90047de2bc34；selected-05第596665035步，SHA30360da5e232fc907df927090b25181dcd5c048cc66ed87439c7103aeefc4cc8。出發GUI沿selected-04，兩次Return，實際畫面先顯示前往Celtac，再顯示遭衛星砲攻擊，最後返回Samar降落平台。after-continue.state第603116727步、area0(1,15)朝南、HP40／EP30、seed8CD8，SHAf1c708a6d06e1a2bf8a2b61668362be3a9e897337bfb5eb8b3b0fa297824be85。兩段GUI自然退出0。

本批7普通鍵／14事件，觀察／控制與觀察／真正GUI各兩份，共4次完整44欄位與完整DOS相同，CPU／RAM／port負對照有效；沒有新完整身體貼圖或正式HD選入。普通GUI重播不等於整屏HD／完整動畫、DAT或從開機。攻略docs/walkthrough-1989.md §3.7的Sivad防衛控制是下一步待驗前置線索，未以攻略補寫原版規則；停止調按鍵或重試Celtac出發。

### 191.3 失敗根因與最小研究修正

首版選單觀察器內部控制相同，對GUI卻有14欄位差異，DOS相同。獨立diagnostic記錄CPUHz33000000對750000、CycleClock與DOSBoxCost false對true、56RAM bytes及計時／暫存器／port差異。原版起點保存舊指令數時鐘；正式前端cmd/psychicwar/main.go載入後呼叫SetAdLib(false)及SetDOSBoxCycles(750)。主來源oracle/live.go與internal/machine/dosbox_cycles.go證實設定CycleClock、DOSBoxCost、CPUHz及recalcIRQ0。

新私有celtac-sprites-observer-v2-20261006.go只在兩側state.Load後套用上述前端兩個設定，不動正式Go、seed、RAM、按鍵時機或比較範圍。同一五次Down與相同絕對終點重播，44欄位／DOS全部相同及負對照通過，證實首版失敗是工具啟動契約差異。出發段起點已是GUI時鐘，原觀察器直接通過，未追加重跑。此結論不外推§190輔助段六欄位差異，該段原因仍unknown。

### 191.4 工具、索引與保全

原版PW.EXE SHA88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49唯讀。正式前端SHA2ee43d77763c89d46c1092a3269774049477acf09c36db08bd0d9034870c63a4、348實際來源沿§190；新研究觀察器SHA1ee5c203996b4c554b6492f979f608d1e07af84c655e0f0aa181a396dd7d335d、64實際非標準編譯來源保全於-build.json。位址依runtime CS:IP／DS:BX／線性Mem，未新增IDA語意。沿既有psychicwar-go-ebiten:latest SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7、Go1.24.13、Python3.11.2、ImageMagick6.9.11-60。

所有入口在既有研究根workplace/ida/hd-ally-recruit-20261004/：verify-ally7-8-art-v2、verify-ally8-art-v3、celtac-select／depart-gui-v1的20261006.py／sh；build-celtac-observer-v2、celtac-normal-gui-replay-v2-20261006.py傳celtac-select；首版replay-v1傳celtac-depart。event-proof與完整machine收據、首版選單失敗diagnostic一併保留。美術入口redraw/ALLY-07-08-verification-v2、ALLY-08-verification-v3、ALLY-07-08-review-v2及三份comparison。新增檔案索引由本節與CONTEXT提供。

Docker user1000:1000、network none、cpus1、有界timeout及memory／pids；建置memory1g／pids128／timeout160s，重播512m／64／80s，文件256m／32。Xvfb有trap、GUI自然退出。update／preserve-ally78-celtac-v1-20261006.py保存結果、私有快照及最終核對。CONTEXT、worklist、WORKLOG與#34同步，#34保持OPEN，#44不改；無commit／push／PR／tag／發行。原生、PNG、完整提示、原版與state純本機；完整HD繼續，來源／構圖限制仍須解決。

## 192. 本週收工、Sivad新分岔與提交（2026-10-06 11:11）

【confirmed：兩段普通GUI與完整狀態比對、第三段GUI自然退出、提交前一般測試。第三段完整重播及防衛控制仍未知。】

本週依使用者要求暫停開發，保存進度並commit／push。正式本機主題仍46PBL＋256MAZE／44PNG，敵人27/360、ALLY3/31，完整HD未完成。最後Sivad上層兩段普通GUI共13鍵，4次完整44欄位／DOS與負對照通過；新電梯下降段GUI自然退出0，尚未完整重播，末點area1(10,2)朝北仍在電梯。下週從保存點確認離開與防衛控制路線，不重試Celtac衛星砲分支；美術已定案，不重問。收工一般測試及兩個主程式編譯通過，研究038 §192。

### 192.1 最後三段的實際範圍

原起點sivad-assisted-elevator-upper-west-01-v1-20261006-event-observed.state第756000000步、area1(7,9)朝西、HP40／EP30、seed65DF，SHA987b20659eec1c89f7bcbeea8ff9a089453de4c1c2a4539345ac4e4d927c46d0。歷史F7／F8起點限制保持，本批只送普通GUI鍵，沒有新HP、座標、seed或隊伍寫入。v1確認(7,9)南側阻擋，經(8,9)返回既有(9,10)電梯；7鍵、兩次完整44欄位／DOS及有效負對照通過。

v2沿實際key-03.state，從(8,9)向東走到(9,9)、(10,9)，東與北阻擋，南側通到(10,10)另一座電梯；6鍵、兩次完整44欄位／DOS及有效負對照通過。兩段均沒有新完整敵人或ALLY來源，不增加正式HD數。重播沿§191修正版750cycles，原鍵與比較範圍不改。

使用者中斷時第三段容器已啟動，後續確認其24秒有界GUI自然退出0、容器自動清除。Down／Return下降到(10,2)，依新Go up選項0AE0 Shown等就緒才送Return。末點第772632132步、area1(10,2)朝北、HP40／EP30、seed65DF、place26h，SHA5141b5c957f134c188463a7959206f4270f6520f97c7e6a384fa2c09371bbd81。座標仍在電梯，不宣稱已成功離開；完整重播尚未執行，不為收工重新跑遊戲。

Sivad DEF COM與Celtac前置條件沿攻略待驗線索，沒有證明已破壞防衛控制。下次先判讀上述保存點及原版選單，接續尚未走過的出口；本週停止新的素材生成、探路與正式實作。完整HD、其餘33既有候選、其他圖庫、ALLY與效果的未完成範圍保持。

### 192.2 入口與私有資料

所有證據在workplace/ida/hd-ally-recruit-20261004/：prepare-sivad-defense-route-v1、sivad-defense-route-gui-v1及replay-v1、sivad-defense-east-gui-v2及replay-v2、prepare-sivad-defense-elevator-v3及gui-v3，檔名日期20261006。GUI各有record、execution、terminal、PNG、F10 state及位置JSON。兩份*-replay-v2-20261006-event-proof.json與machine收據只涵蓋前兩段。第三段續接入口sivad-defense-elevator-gui-v3-20261006/after-leave.state。

既有正式前端SHA2ee43d77763c89d46c1092a3269774049477acf09c36db08bd0d9034870c63a4、348實際來源不改。觀察器SHA1ee5c203996b4c554b6492f979f608d1e07af84c655e0f0aa181a396dd7d335d、64來源沿§191；比較器44欄位／DOS與CPU／RAM／port負對照沿既有固定SHA。工具位址為原版runtime CS:IP／DS:BX及線性Mem，沒有新IDA語意或原版規則。

原版、HD原生與PNG、完整生成提示、state、DAT與私有快照均不加入Git。tools/hd/battle-effects-prompts*.json保留本機，.gitignore追加精確類型規則。提交只收既有程式、規格、研究摘要、工作歷程與工具；不建立tag、Release或新發行包。本節與CONTEXT同步索引新增私有檔，不另建文件分類。

### 192.3 提交前驗證與暫停

使用既有psychicwar-go-ebiten:latest、Go1.24.13及Docker user1000:1000、network none、cpus1、memory1g／pids128、外層timeout160s。Xvfb有trap；非音訊或效能驗收。go test -p 1 ./xlate於dosgolem通過；主repo go test -p 1 ./apps/... ./cmd/...通過四個套件，兩個主程式編譯通過。沒有設定原版HD fixture環境，不把一般測試綠燈外推完整HD。收據weekly-commit-tests-v1-20261006.json及兩份log。

dosgolem通用疊圖、文字背景與對應測試提交31242a9，分支psychic-war/r5-text；本repo繼續沿go.mod本機replace及README克隆入口使用該分支。兩repo git diff --check通過，提交明確列出新增依賴檔。使用者本次已授權commit與push，目標為本repo main及dosgolem專案分支，沒有force push或改上游main。遠端核對與收工快照保存於weekly-closeout-final-audit-v1-20261006.json，文件與#34同步保持OPEN，#44不改。

Goal保持paused，中文化與完整HD沒有宣稱完成。本週提交收尾後停止工作，待使用者下次繼續。

## 193. 全部草稿明示定稿、391圖號保全與四圖庫60張接入（2026-10-06）

【confirmed，限定使用者決定、素材保全、合成圖面及既有正常保存點回歸】

使用者明示「草稿我都同意定稿，依序完成敵人圖號、盟友圖號」。全部現有美術草稿已接受，停止逐張美術批准及手腳端點微調；接入工作先敵人、後盟友。原版位置、比例、姿勢、8×8及人物在後／框線在前保持。

敵人360/360及盟友31/31圖號的既有美術已逐檔凍結定稿。敵人原60張READY身體來源中其餘33張加入現行本機主題，共79筆PBL＋256格MAZE／77PNG；接入敵人60/360、ALLY3/31。敵人正常呈現已驗仍27/360，新增33張普通中文GUI抽測及其餘300圖來源／接入尚未完成。

敵人360/360的來源PBL圖數各30、解碼色號SHA、PNG尺寸與既有SHA逐圖核對。四已接入圖庫的60張保持現行選稿；ENEMY02採group0 v2及group3 v3，其餘沿已選版本；ENEMY05／06／07已選組及ENEMY07 group4最新v4；ENEMY08 #0–#11採v3、#12–#14保持舊72×72；ENEMY09–11沿已選草稿，180小圖沿全量清冊。這是本次使用者接受，不改寫先前未接受／拒稿的診斷收據。ENEMY02像素別名與ENEMY08少96bytes的原版來源仍未知，不因美術定稿猜補。

ALLY31/31逐圖核對，#0–#2保持正式稿，#3 v5、#4 v2、#5 v6、#6 v1、#7 v2、#8 v3、#9 v1、#10 v2、#11 v1；#12–#29沿既有素材，#30採20261004藍髮人像修稿。九個新人物完整原生Lanczos72×96 RGBA重生差0，實際單位元組負對照差1。#5取左臂較接近原版的v6，歷史v7不刪除，不新增容差。12完整人物、4局部及15小圖分類保持，不從靜態圖形推斷玩法身份。

正式本機theme-enemy60-final-maze-B-v1-20261006保留前46筆／44PNG並新增33圖，使用既有024 §1.22–§1.23 READY來源，不改Go、EXE、RAM、seed或DAT格式。既有ready60_art_test.go以新主題plan重跑60張全部通過，包含獨立整圖面、8×8遮格恢復、錨點、開關、冷載、省略及單像素負對照。六個既有正常保存點各載入及接續100,000指令，12圖面與前46筆主題相同；獨立44機器欄位、完整DOS與CPU／RAM／port負對照通過。沒有新GUI、DAT、全動畫或封包驗收。

測試首命令將-p1合成一個參數，go test未取得套件參數並報no Go files，屬命令環境失敗。按既有成功命令改成-p與1兩個參數，同容器工具鏈／期望乾淨重跑通過；兩份收據保留，不記產品缺陷。成功命令：Docker內go test -p 1 ./apps/psychicwar/theme -run '^TestReady60ArtBatch$' -count=1 -v；明示PSYCHICWAR_READY60_ART_PLAN與PSYCHICWAR_TEST_ORIG，不設定舊硬寫78筆的render輸出欄位，實際79筆／77PNG以selection-receipt及test-result-v2記錄。

輸入位址基準：PBL原始檔偏移、解碼色號及PNG，沒有新增反組譯位址或原版行為推論。Go1.24.13、Python3.11.2及ImageMagick6.9.11-60，既有psychicwar-go-ebiten:latest SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7。主repo基準7deffb9、dosgolem31242a9，正式程式保持。

逐圖定稿入口workplace/hd/enemy-art-final-v1-20261006/art-index.json與ally-art-final-v1-20261006/art-index.json；這兩份研究清冊不能當production manifest。重跑、來源及收據沿workplace/ida/hd-ally-recruit-20261004/finalize-approved-enemy60-v1-20261006.py、freeze-approved-enemy-art-v1-20261006.py、freeze-approved-ally-art-v1-20261006.py、record-approved-sprite-final-v1-20261006.py，enemy60-final-v1-20261006/plan.json、test-result-v2.json及test-v2.log，enemy60-final-normal-v1-20261006.go及該目錄runtime.json。正常回歸保持兩側同保存點／seed，沒有改原版狀態。

CONTEXT及worklist更新唯一現況，AGENTS §12和024 §1.46保存新明示決定。工作接入先敵人、後ALLY，不重抽已定稿美術。#34更新保持OPEN，#44不改；完整HD及平台交付未完成。源、PNG、完整提示、state及保全留本機，未新增commit／push／發行。私有保全與Docker／擁有權沿sprite-art-final-audit-v1-20261006.json及sprite-art-final-source-snapshot-v1-20261006.tar.gz。

## 194. 十二敵人圖庫受控原版載入與174張限定來源契約（2026-10-06）

【confirmed：十二次原版C6執行與RAM資料。原版共用階段為已證實分支／資料模型；新增圖庫正常玩家載入及GUI仍未驗。】

沿主repo7deffb9及dosgolem31242a9，原版PW.EXE SHA88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49唯讀。起點06-name.state，每圖庫重新載入；研究程式明示設定runtime CS:IP=0161:2AF0及CS:305D的C6參數0–11，執行原版180條已核對C6指令至0161:2C43。這是direct-entry受控來源實驗，非正常玩家路徑，不宣稱沒有研究狀態設定。原版EXE及檔案不改，正式覆繪仍唯讀。

十二次均在有限指令內結束、SP平衡；實際開檔為I_MAP、CODE、I_MENU、同ENEMY PBL三十次及I_ENMY。原始RAM413段95,344 bytes的初始身體、DS／CS差分、16×16小圖及80bytes紀錄與兩份獨立PBL解碼相符。小圖只核對載入資料，不證實用途或接入位置。59完整組236階段由既有原版sub_1435B選DS／CS分支重建，所有一bit負對照有效，沒有把模型稱為自然戰鬥。

391原圖唯一身份審查留下58組174張完整24×32。ENEMY02 #12/#13像素別名，整組#12–#14排除；ENEMY08 #12–#14原24×24，排除。後者初始288bytes相符，尾端96bytes與假設的上一組中間姿勢不符，不能猜舊值、補零或拉長成24×32。其餘十檔各#0–#14，02／08各#0–#11。

實際原版執行補足024 §1.32只靠靜態模型的缺口。限定來源證據審查後，024 §1.47在實作前標READY；允許沿既有來源識別及生命週期擴張白名單。普通GUI、來源生命週期、動畫及DAT仍是完成閘門，未放寬為受控測試即可完成HD。此判定不猜新增圖號、位置或規則，不將其餘DRAFT整節改成已驗。

位址基準：runtime CS:IP及線性RAM；IDA EA=runtime IP+10510h，EXE檔案偏移=976+IDA EA−10000h。既有IDA Pro9.4匯出及原始bytes保留，不改導覽名稱；共用階段原始證據body-bank-native-cycle-proof-v1-20261005.json，C6入口enemy-bank-routing-ida-v8-20261005.json。工具Go1.24.13、Python3.11.2，既有psychicwar-go-ebiten:latest SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7。

新增入口同研究根workplace/ida/hd-ally-recruit-20261004/：enemy-allbanks-c6-controlled-v1-20261006.go、build-enemy-allbanks-c6-controlled-v1-20261006.py、-build.json、-overlay.json及.bin，verify-enemy-allbanks-c6-controlled-v1-20261006.py；目錄enemy-allbanks-c6-controlled-v1-20261006/的execution.json、十二份RAM／state及source-proof.json。binary SHA4cfba477505a0ab04b7d9ffc9d7f405049dfdeec7b012934fe05ffbe26b937ed、62份實際非標準編譯來源保存。來源審查enemy174-source-contract-review-v1-20261006.json包含READY當時spec、原版共用分支及source-proof SHA。編譯初次缺少既有probe檔依賴的die(error)，修正後成功；重複建置被避免覆寫斷言擋下，均屬研究工具問題。

174張獨立原版PBL／PNG合成期望、8×8遮格恢復、錨點、HD開關、冷載及省略／單像素負對照均通過；58組232條實際RAM差分與錯階段／一bit負對照通過。別名及短圖組、未知檔名與錯SHA被拒絕，既有Sivad／Zellwal來源與清單限制回歸通過。測試新增enemy_controlled_bank_test.go，ready60_art_test.go兼容兩種限定批次清冊，輸出筆數從實際manifest／PNG計算。舊60張獨立圖面另跑通過，不把174合成case稱正常GUI。

本機theme-enemy174-final-maze-B-v1-20261006新增114已定稿PNG，原79筆／77PNG逐SHA保持；共193PBL＋256MAZE／191PNG，敵人接入174/360，ALLY3/31。六個原有正常保存點各載入及接續100,000指令，12圖面與舊79筆主題相同，44機器欄位與完整DOS相同；CPU／RAM／port負對照有效。正常回歸二進位SHAb8d9ac64c399f96f0553849321476835a922200e3f4192e0663f333719bcd678，90份實際非標準編譯來源保存，沒有使用新受控state當正常起點。

來源／重跑索引同研究根：prepare-enemy174-final-v1-20261006.py、verify-enemy174-final-v1-20261006.py；enemy174-final-v1-20261006/plan.json、render.json、test-result.json及tests.log／old60-regression.log；run-enemy174-normal-v1-20261006.py、enemy174-final-normal-v1-20261006.go／.bin／-build.json及同名目錄runtime.json、12份-machine.json。受限sandbox第一次timeout docker無socket權限，使用既有主機Docker權限後執行；不記產品缺陷。Docker network none、user1000:1000、cpus1、有界timeout及memory／pids，原版唯讀。

新正式前端SHAa0f0688d92be28dbccf4404a310315b5ebe3c0976f8567f5d83fbc61be22a516，348實際編譯來源在enemy174-frontend-v1-20261006-build.json。GUI沿既有F7／F8輔助來源起點，只有普通Up送原版，不稱全程自然遭遇；960×600視窗24張，10張非黑8×8格與獨立原版／PNG期望相符，ENEMY04 #9／#10／#11三姿勢都可見，省略／錯姿勢負對照有效。其餘14份未宣稱HD通過，沒有挑選樣本掩蓋差異；此為有限可見格驗收，非整屏同幀。新前端末點依實際record鍵與相同絕對指令數重播，observer-control及actual-GUI兩次44欄位／完整DOS相同，CPU／RAM／port負對照有效。這是既有三姿勢回歸，正常圖號數仍27，不外推新增圖庫GUI。

GUI第一次來源保全檢查漏掛/gomod唯讀來源，遊戲未啟動；保留空輸出目錄及failure.json到enemy174-gui-preflight-failed-v1-20261006，補掛既有gomodcache後相同binary／script／輸入重跑通過。Xvfb由trap終止，前端自然退出0。入口prepare-enemy174-gui-v1-20261006.py、enemy174-gui-v1-20261006.sh／.py、-independent-v1-20261006.py／.json及同名GUI目錄；普通鍵重播enemy174-gui-replay-v1-20261006.py傳enemy174-v1，rusteck-enemy174-v1-replay-v1-20261006-event-proof.json及兩份-machine.json，原始按鍵與保存點皆保全。

完整主題套件回歸v2通過，頂層25項通過、15項未提供可選fixture而明示跳過。受控232轉換、獨立174圖面與舊60圖已另行無跳過執行。v1唯一失敗為TestThemeRealBackground從/orig相對推到/hd/bg.idx，不符合既有fixture位置；v2使用同一唯讀原版的儲存庫路徑，獨立背景fixture不改，同工具鏈／命令乾淨重跑通過。兩份log／json保留，不記產品缺陷。

最終本機驗證入口theme-enemy174-final-maze-B-v1-20261006/verification-receipt.json，建立時selection-receipt的pending歷史不重寫。preserve-enemy174-v1-20261006.py保存本輪增量enemy174-source-snapshot-v1-20261006.tar.gz及索引／逐SHA核對enemy174-final-audit-v1-20261006.json。前輪完整美術保全sprite-art-final-source-snapshot-v1-20261006.tar.gz仍保留，本輪索引引用其SHA，原生與生成提示不重抽。#34兩次讀回全文保持OPEN，#44未改；Git差異及擁有權核對，Docker容器均以--rm及GUI trap清理。未commit／push／PR／tag／發行，所有原版、PNG、state、完整編譯來源及archive純本機。

## 195. 180敵人小圖來源、原版共用動作與遮罩格式B定案（2026-10-06）

【confirmed：原始EGA來源、原版受控初始化及共用動作。受控槽值不是正常玩家路徑；來源READY不增加接入數。】

使用者選定「B：新增builtin_masks」。024 §2.0由DRAFT改成限定格式READY，AGENTS §12保存決定；不重新詢問。主題/2使用可省略的獨立清單，主題/1保持。每筆必須提供id、at、png、kind、match，正式id為battle-mask-4e36；不加入任意EXE位址或圖號。美術仍沿使用者已接受的B透光，不因技術格式改稿。

十二圖庫ENEMY00–11各#15–#29，180張16×16。既有原版C6的十二份RAM與兩份獨立PBL解碼核對23,040 EGA bytes，180來源在391清冊各自唯一；舊60正常來源SHA保持。各槽後64 bytes的11,520 bytes只作CGA候選診斷，並非全部相同，語意未知，不作EGA輸入。初次驗證器把未證實的CGA映射當斷言而失敗；v2刪除這項假設，保留原EGA期望與負對照，沒有改原始RAM。入口同研究根verify-small180-c6-v2-20261006.py、small180-c6-proof-v2-20261006.json，首次failure.json保留。

IDA原始函式sub_144DC，執行期0161:3FCC，才是敵人小圖。原版sub_14038初始化來源DS:AF12。17槽請求值CS:3ACA、存值CS:3AFD，0為空，1／2／3依序對應#15+3g／#17+3g／#16+3g。CS:40B1的16個u16偏移以4×前值＋新值索引，AL=1進行XOR新增、轉換及消除，相同值不貼圖。sub_14566／DS:AF0C是玩家BEAM分支，不能混用。

實際小圖位置為(8+16s,y,16,16)，s=0–16；y160或168。原版DX=F522h名義y170經sub_1632F取商、兩次右移及一般貼圖乘4，實際y168。受控v1選錯玩家分支，得到空來源；獨立驗證v2又把名義170當實際170而拒絕。兩次研究工具失敗保留，按原始資料流修正；同一份v2實際執行收據由v3驗證器核對，不改輸入或重擲。

12圖庫×5組×2位置，共120受控原版初始化及sub_144DC執行。每組明示CS:IP、組號、狀態槽及原版狀態參數，16前後值組合共1,920對，1,440次貼圖、480次不貼圖。239原始指令與EXE及執行期RAM逐項相同；1,440完整frame由原始PBL、前frame及XOR重建差0，錯來源bit與錯目標負對照各1,440有效。舊正常敏頓209小圖座標回歸保持。合成前狀態槽不表示畫面已含對應舊姿勢，因此這只證實原始差分來源與輸出，不冒稱自然整段動畫。

位址基準：IDA EA；runtime0161:IP=EA−10510h；MZ file offset=976+EA−10000h。PW_UNP.EXE SHAfd5115b91f014c47a1fb1a6e2ecfd645e293264fe614a4b350e92637cad2fbd9；唯讀資料庫SHA4db51a199f5913c52dc4964557b9a4aafb965754d663cb0173c9dfdfe9a00c56。IDA Pro9.4 image SHA6f6d59af49d0008c4109a5295b5f374bdc007e2d1ab28cb9de08779584de2780，複本於/tmp開啟，不改原名、註記或正式DB。

來源索引：small-motion-ida-v1／v2-20261006.py及.json、兩份run與-command收據；small-motion-controlled-v2-20261006.go／.bin／-build.json／-overlay.json、同名目錄execution.json與所有frame／work.bin；verify-small-motion-controlled-v3-20261006.py、small-motion-controlled-proof-v3-20261006.json。二進位SHAfe27fae11b1f7b5c70405044b618d980d3d3cc1db2a4f658a6c8d358e3faabb4，62份實際非標準編譯來源保全。024 §1.48限定來源／動作READY；177張其他小圖、兩組例外與GUI仍待完成。

## 196. builtin_masks正式接入、敏頓B效果與普通GUI限定驗收（2026-10-06）

【confirmed：已知來源分解、限定正式合成、可見效果格與原版無修改比較。未證實所有真實中途、其他敵人動畫、DAT或效能。】

首次101變數候選按§180來源接入正式效果層，2,238完整邊界與1,119來源轉換通過。普通GUI發現未列入的遮罩位置，完整模型因此回退原版效果；這份候選未通過GUI，不列正式完成。回到原版來源查證，再修024 §1.49，沒有在程式默補位置。

首次GUI錄製起點為正常Minton #6完整保存點，steps381716779；Space381977770按下、392944263放開，F10終點392954097。研究重播v1誤用typematic=false，與前端KeyDown的按住鍵重複契約不同，44欄位比較拒絕。v2改用已查證的重複契約，保持同一初始state／原鍵／終點，控制及實際GUI兩次44機器欄位與完整DOS相同。這是研究工具修正，不記產品缺陷。v2原版874一般貼圖、90遮罩，1,928前後邊界與方向由獨立PBL／EXE來源重建差0。

原版sub_1461B，runtime0161:410B，DX=F202h起17槽，每槽加4；sub_14685，runtime4175，使用CS:4197的八偏移[-642,640,-638,-2,642,-640,638,2]；sub_1530A，runtime4DFA，使用CS:4E36的32-byte遮罩。98原始指令與解壓EXE及C6執行期RAM一致。共88原版位置，78完整位於(32,144,256,40)域內；域外10位置保持回退。78是靜態原版位置契約，並非78正常GUI樣本。

域內y152／168的x32至272每8，y160的x32至272每16；敵人小圖沿原版17槽，批准域內y160、x40至264每16。BEAM沿已驗y160、x40至248每16；FIGHT #0–#1在(256,144)、#2–#3在(40,144)；身體#6–#8在(32,152)。3身體、45小圖原位、42BEAM原位、4FIGHT及78MASK，共172變數滿秩。新GUI重播1,928邊界唯一冷載解與原事件真值相同，未知像素、兩身體姿勢、外來身體及錯已知bit負對照有效。

遮罩只有SHAe1aa2b9ddb6488a70d573ca3a03e71028cd8a2dafa68087c6f6f3d7c60950a5a進入正式程式，原32 bytes從玩家執行期CS:4E36唯讀取得，沒有嵌入Git。PW.EXE SHA沿§194。原始場景由SCREEN五張、MENU及ALLY #0重建，身體變數與原版基底差分；效果為色號XOR。完整場景用GF(2)冷載唯一解，不依賴HD畫面或事件歷史。一般8705／8751及遮罩8260／4E34入口只在完整前場景與唯一來源下預測中途新場景；全域8×8格與不可變模型不符即回退。ResetForLoad、未知來源、錯遮罩及錯錨點清除預測。

新正式程式為apps/psychicwar/theme/battle.go，theme.go負責格式／載入／圖面，ally.go附加唯讀來源入口。HD先沿SCREEN／MENU／ALLY基底畫人物，效果沿B透光，以各通道透光係數相乘後一次取整；全黑原版8×8來源格不畫PNG。原位、比例、人物在後／框線在前保持，中文在HD之後。舊/1與沒有新清單的/2主題保持，錯欄位、null、來源SHA、尺寸、原位及重複變數拒絕。原版RAM／buffer／DOS／DAT不寫入。

兩條獨立原版清單共4,166完整邊界、2,083來源轉換及2,083錯來源負對照全部通過，另有每次測試12個合成中途8×8遮格／恢復樣本。12個樣本是前後原版frame間只改第一個變動像素，不稱真實貼圖中途收據。3正式768×120圖面的全RGBA與独立PBL／PNG／B公式差0，敵人小圖、BEAM、FIGHT、MASK的省略及錯原位負對照有效。174身體、58組232轉換、Sivad／Zellwal及格式必要欄位測試無跳過通過；完整主題套件26通過、16項可選fixture明示跳過，不把跳過當成功。

現行本機theme-enemy177-effects-maze-B-v2-20261006保留193PBL／191PNG，新增91PBL效果原位、78builtin_masks，共284PBL＋78內建遮罩＋256MAZE／202PNG。新增11個PNG含三張已凍結敵人小圖，其餘沿已展示BEAM／FIGHT／MASK稿，不重生美術。三敵人小圖SHA逐項與391美術清冊相同。敵人接入177/360，正常呈現由27增至30/360；ALLY仍3/31。剩183敵人為177小圖與6例外，28ALLY接入排在敵人後。

六既有正常保存點的載入及接續100,000指令，12次原版44欄位／DOS相同，HD開關恢復相同。新效果圖面會填入原始已知基底，原始透明通道可與舊圖面不同，因此驗收預先指定為最終合成畫面，12份與舊174主題相同，沒有事後降低比較。

新正式前端SHA08bb307118108a122233682d6dd1840d796c9815aa792efc4094bad7c5a53c3b，349實際非標準編譯來源保全。GUI v3在第44次抓圖時超過16秒自動退出；另一次前置檢查漏掛/gomod，未啟動遊戲。兩項屬驗證環境／預算問題，保留來源及輸出。v4用同一前端及正常起點，延長至30秒，36張960×600視窗、普通Space及F10保存，自然退出0；Xvfb trap清理。

v4實際Space381960769按下、388862996放開，F10終點388881409、cycles1861545003、seed5E38。初始seed從同保存點唯讀取得，不寫seed或重擲；750cycles沿前端載入後設定。原版重播551一般貼圖＋57MASK，共608事件、1,216完整邊界，原始來源／方向與172變數冷載全部相符。控制及實際GUI兩次完整44欄位、DOS與CPU／RAM／port負對照通過。純F10保存不算512-byte DAT驗收。

35/36份GUI視窗共2,436效果格與獨立原版來源／定稿PNG／B公式相同，新增#21／#22／#23各431／432／440個可見來源格，BEAM587、FIGHT682、MASK406；重疊格可同時計多類來源，不將各類相加當總格。逐來源省略負對照有效。未驗到效果格的1畫面保留，不宣稱通過。候選原版邊界步數只供找回模型，並非抓圖時間戳，因此本項是有限8×8格驗收，不稱整屏同狀態或全動畫。前端有中文靜態標籤，-load-state未恢復舊訊息／名字的中文歷史，不稱從開機普通中文全流程。

工具沿§194的Go1.24.13、Python3.11.2、ImageMagick6.9.11-60與Docker image SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7；IDA版本與位址基準沿§195。研究根仍為workplace/ida/hd-ally-recruit-20261004/，沒有新研究根或交付目錄。

可重現入口：

- 清單重建：prepare-minton-production-v1／v2-20261006.py，v2的--check-only逐bytes重建284PBL／78遮罩及202PNG相同；--out只建立workplace/hd/下全新本機目錄，不覆寫。實作前172來源READY完整快照minton-expanded-ready-spec-snapshot-v1-20261006.md及-review.json與原plan所錄SHA完全相同。minton-current-production-source-review-v1-20261006.json確認113專案來源與已驗GUI的349來源編譯版本逐SHA相同。
- 原版位置：battle-mask-ida-v1／v2／v3-20261006.py、.json及run／command；verify-minton-mask-contract-v2-20261006.py、minton-mask-coordinate-contract-v2-20261006.json。
- 初次GUI來源：minton-gui-replay-observer-v2-20261006.go／.bin／-build.json及minton-gui-replay-normal-v2-20261006.json，verify-minton-gui-replay-directed-v2-20261006.py、verify-minton-gui-cold-expanded-v1-20261006.py。重播binary SHAc010b2f0b49551e9ff189aabc5d1a0a39b5c0814d423b2f8ad7a34ed4a6181aa，63份實際來源。
- 正式測試：minton-production-tests-v3-20261006.bin／-build.json，SHA78694b127fedb113014e06b9473e1314b3d79db7a960854dae922eeee54df17a、111實際來源；兩份minton-production-plan-expanded-{old,gui}-v1-20261006.json及minton-production-expanded-{old,gui}-v3-20261006/tests.log／render.json／test-command.json。明示PSYCHICWAR_TEST_ORIG、PSYCHICWAR_MINTON_PLAN及PSYCHICWAR_MINTON_OUT後執行-test.run 'TestMintonProductionScene|TestBuiltinMaskRequiredFields'。測試檔後續只有兩個加號的空白格式修正，實際編譯原文仍完整保留。
- B合成：verify-minton-production-render-v3-20261006.py、minton-production-render-independent-v3-20261006.json。
- 正常回歸：minton-production-normal-v3-20261006.go／.bin／-build.json及同名目錄runtime.json，SHA1ccbebc3042b295089a5bea4a342f52a1b02411494be27a3045cb0a2042d70f8、91實際來源；12份-machine.json。
- 正式GUI：minton-production-gui-v4-20261006.sh／.py及同名目錄36視窗、record.json、terminal.json及final.state；新349來源前端同名前綴frontend-v3。GUI所用容器掛repo／原版／gomod唯讀、研究根可寫，CPU2、memory1g、pids128、有界逾時及Xvfb trap。
- 新GUI原版：minton-gui-replay-observer-v3-20261006.go／.bin／-build.json，SHA33cb9ff8a155bbe6fb6490d28115e5f99c1a6c74de2f4bf288b9e12a58694400、63實際來源；minton-gui-replay-normal-v3-20261006.json與所有frame，verify-minton-gui-replay-directed-v3-20261006.py、verify-minton-gui-cold-expanded-v2-20261006.py；minton-gui-replay-{control,final}-machine-v3-20261006.json。
- GUI格：verify-minton-gui-cells-v1-20261006.py、minton-gui-cells-independent-v1-20261006.json，記錄所有36張及未匹配畫面，不隱藏負結果。

現況由CONTEXT及worklist保存，#34保持OPEN，#44不改。完整效果圖號、真實中途／DAT、其他敵人、全動畫、效能、權利、封包與平台待完成；024 §1.49保持限定READY，不稱全項CONFORMED。前輪原版及391美術archive保持，新來源／完整編譯原文／PNG／state在本機增量保全；入口preserve-minton-runtime-v1-20261006.py、minton-runtime-source-snapshot-v1-20261006.tar.gz、minton-runtime-final-audit-v1-20261006.json。未commit／push／PR／tag／發行。
## 197. 十二圖庫58組小圖的來源選擇與合成

日期：2026-10-06。規格024 §1.50；本節入口為workplace/ida/hd-ally-recruit-20261004/explore-small-profiles-v1-20261006.py及small-profiles-exploration-v1-20261006.json。12原版圖庫SHA沿§194–195，輸入逐檔SHA記錄於收據；工具Python3.11.2，既有psychicwar-go-ebiten image，原始PBL唯讀。

已證實：三段工作源0／C0h／180h各128 bytes連接，在60組各自唯一，120受控原版初始化皆吻合。原版指標1175:AF12；sub_14038的IDA EA14081 bytes `8b160eaf`讀DS:AF0E，14092的`f3a4`複製600h；原始指令與執行期驗證沿§195保留，不改導覽名稱。12份既有正常保存點讀取工作源，對應敏頓、卡蘇魯奇、格斯丁提與賈克斯莫組別。入口small-profile-normal-source-v1-20261006.go／.bin／-build.json／.json，實際77來源及binary SHA在build收據。

已證實：58個有完整24×32身體的組，三身體＋BEAM42＋FIGHT4＋MASK78＋兩高度90小圖，各217變數獨立滿秩。原版像素來源模型範圍(32,144,256,40)，未含兩組身體例外、小圖x8／24、域外遮罩、其他ALLY與光束。這是合成來源與可唯一求解的證據，尚不稱58敵人正常GUI或完整HD完成。正式接入與必要回歸結果續記本節。

限定接入已實作：新增171小圖，敵人348/360，包含174身體及174小圖。現行本機theme-enemy348-effects-maze-B-v2-20261006有5,459PBL／78builtin_masks／256MAZE、373PNG，前193筆及202PNG保持。美術沿360／31凍結清冊。原版指標、工作源、畫面與遮罩全唯讀，未知或切組清空預測，冷解及中途全域8×8沿024 §1.49–§1.50。

以下入口均在workplace/ida/hd-ally-recruit-20261004/：

- prepare-small-profile-plan-v1-20261006.py產生58組、174合成frame及12,586單源變數。正式apps/psychicwar/theme/battle_profiles_test.go沿清單核對來源SHA、原位、滿秩、冷載、讀檔、開關及174錯工作源負對照。small-profile-tests-v1-20261006/保存58圖面；verify-small-profile-render-v1-20261006.py及small-profile-render-independent-v1-20261006.json，58完整RGBA差0、58省略小圖負對照有效。合成場景不增加GUI計數。
- small-profile-tests-v2-20261006.bin／-build.json保存112實際來源。新217模型的敏頓4,166邊界及2,083轉換／錯源負對照通過、24合成中途格線樣本，small-profile-regression-v1-20261006/{old,gui}.log。一般套件24通過／19可選fixture跳過，small-profile-suite-orig-v2-20261006.json；核心58組及兩敏頓清單另無跳過執行。
- small-profile-normal-v2-20261006.go／.bin／-build.json及同名目錄runtime.json：六起點載入與100,000指令接續，12次44欄位、RAM／CPU／port負對照及完整DOS相同。10舊合成相同；賈克斯莫新增#18–#20小圖／FIGHT，差12,578放大像素；卡蘇魯奇新增光束／FIGHT，差11,206。verify-small-profile-normal-v3-20261006.py及small-profile-normal-independent-v3-20261006.json，由PBL、前幀與B公式獨立核對，差異皆在已知来源矩形。
- 卡蘇魯奇一份真實中途：237809800步入口(72,160,16,16)、AL=1，237810721步終點。small-profile-normal-pending-v1-20261006.go／.bin／-build.json／.json保存独立原版前幀與來源；唯一推得BEAM #0目標，156格与新模型／B公式相同，4格回退且新舊合成相同。party／items／格斯丁提六份未知場景保持舊合成，不稱模型已解出。
- 新前端small-profile-frontend-v1-20261006.bin SHA-256 `767fcd0fc5eb97fa7344da1569c6d4a494f5a3b4b79daf4b4d98b02156cdccf2`，-build.json有349實際來源。small-profile-gui-v1-20261006.sh／.py跑既有正常敏頓state、普通Space與F10，36視窗、自然退出0。verify-small-profile-gui-cells-v1-20261006.py及small-profile-gui-cells-independent-v1-20261006.json核對35視窗2,387效果格、三小圖與BEAM／FIGHT／MASK省略負對照。模板引用前輪原版邊界，候選步數不是新截圖時間戳，未稱整屏同幀。
- GUI實際Space381950198–388541576，F10終點388575312、cycles1861238906、seed0EE7。small-profile-gui-control-v1-20261006.go／.bin／-build.json與small-profile-gui-control-normal-v1-20261006.json：原版523一般／55遮罩，觀察及控制終點相同；small-profile-gui-machine-v1-20261006.json核對新GUI44欄位及DOS相同。750cycles、同state、按住鍵重複，seed唯讀不重擲。

失敗與訂正保留：主題v1的三小圖新別名造成PNG376不符373預估，v2沿同SHA舊檔名乾淨重建；六狀態初版誤把新增效果要求為舊畫面，v2保存差異供獨立核對；normal verifier v1缺真實中途，v2誤把未改且未知的格斯丁提當可解，v3明示未知回退，改動畫面仍需唯一來源及中途證據。私有debug overlay只讀合成狀態，不進production。GUI缺/gomod、套件缺/hd/bg.idx是環境失敗，相同image／命令補唯讀掛載後重跑通過，不列產品缺陷。

敵人正常GUI仍30/360、ALLY3/31。最後12敵人、28ALLY、其他基底／效果、小圖域外、動畫、DAT、效能與交付未完成。024限定READY、#34 OPEN、#44不改。本輪保全入口preserve-small-profile-runtime-v1-20261006.py、small-profile-runtime-source-snapshot-v1-20261006.tar.gz及small-profile-runtime-final-audit-v1-20261006.json；沒有新commit／push／tag或發行。

## 198. 最後兩組敵人別名與短圖來源接入（2026-10-06 19:09）

依使用者定案續作，不重問美術。路由命中IDA／規格閘門／文件職責，載入use-ida-pro-9-4、grilling與對應入口。原版唯讀，正式PW_UNP.EXE.i64只在IDA容器/tmp複本查詢，沒有改名、分類或原始bytes。Docker沿psychicwar-go-ebiten:latest、Go1.24.13及Python3.11.2；IDA沿locked-v1 9.4。image ID沿§197及§100，UID/GID1000、network none、資源限制、逾時與--rm。

【confirmed，限定原始來源】ENEMY02 #12／#13的384-byte原圖相同，已凍結定稿PNG bytes也相同，SHA59ba5219b0ac242fdd4d95d9ac74073cc21e5ad3607802f94f6354c6febc35e5。只此對可共用圖面／模型，保留兩圖號；不同PNG及重複同圖號拒絕。ENEMY08 #12–#14為24×24，HD保持72×72；三個前288-byte來源在391圖的24×24前綴各自唯一。C6觀察的尾96 bytes均為零，但語意仍unknown，不升格成固定零契約。

【confirmed，限定受控原版執行】從既有C6 bank02／08 state明示初始化group4，再直接進sub_1435B，CS:3B20設1。八次原版AL=1、(32,152,24,32)貼圖全畫面XOR與原始PBL已知區域相同；ENEMY02首末兩個零差分也如實保存。原始IDA14038／1435B等rows與原版RAM逐bytes核對；runtime0161:IP=IDA EA−10510h，MZ fileoffset976+EA−10000h。輸入EXE、.i64及原版PW.EXE SHA沿§100／§194保持。這是明示direct-entry，不當正常玩家或GUI。

024 §1.51 READY後接入六身體及六小圖，敵人来源覆蓋360/360。別名組216變數滿秩；短圖忽略(32,176,24,8)的三個全域8×8格，所有該區HD及重疊效果都保留原版。#27／#29在(40,168)的墨跡全部落在這區，217登記變數的兩個不可見變數不參與冷解；其餘215滿秩。初版獨立矩陣報兩個零向量後先核對原版逐列墨跡，沒有選任意姿勢或補零。中途完整前圖仍可顯示，但不得覆寫已由入口確認的待完成目標；新增pendingValue保存這個既有契約。

60組／180合成場景、13,019登記變數及13,017可見變數冷載通過。原版八轉換、40短圖中途模型、錯來源、尾段、不同別名PNG、讀檔與開關負對照通過；兩份敏頓4,166邊界／2,083轉換保持。60独立B整圖面RGBA差0；新ALLY合併主題亦重新核對60份。正常GUI仍30/360，未算新12圖。六正常保存點12次44欄位與DOS、12舊圖面及合成相同，不稱全動畫、正常新圖號或DAT完成。

來源入口在既有研究根：explore-last-enemy-groups-v1-20261006.py／last-enemy-groups-source-review-v1-20261006.json、last-enemy-motion-v1-20261006.go／.bin／-build.json及同名目錄、verify-last-enemy-motion-v1-20261006.py／last-enemy-motion-independent-v1-20261006.json。實作apps/psychicwar/theme/enemy.go、ally.go、theme.go、battle.go；驗證enemy_last_groups_test.go與battle_profiles_test.go。合併期望enemy360-ally12-plan-v2-20261006.json；原型及先前360主題保留，不覆寫歷史收據。

## 199. ALLY九張完整道具肖像接入（2026-10-06 19:09）

敵人來源批次接入後，依使用者順序接ALLY #3–#11，沿既有定稿，不重生。先查既有普通來源盤點，未取得新正常ALLY圖號，不再猜步行或重擲seed。只追已知道具肖像共用載入／貼圖切片，停止於來源、原位、模式足以形成限定READY契約；正常角色選取、隊伍及其他19圖號另列unknown。

【confirmed，限定IDA與受控執行互證】原版ALLY.PBL SHA c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219，31圖數與兩解碼器核對。IDA132A5 bytes e84dff，runtime2D95呼叫2CE5；後者設定AH=0Ch並讀DS:AEE8。132A8 bytes baa0c2、132AB bytes e89d30，送sub_1634B原位(128,8)、24×32、AL=0。原DB把這些bytes列data，匯出保留分類及原始bytes，對齊decode只作附加證據，沒有改DB名稱。相關輸入SHA、工具9.4及位址基準沿§198／§100。

12次直接進2D95並明示AL0–11，全部384-byte來源與391圖唯一身份相符；全畫面COPY差0，SP保持，12個一bit負對照有效。這只證實受控原版共用來源／原位／模式，不證實#3–#11玩家選取、招募或隊伍位置。024 §1.52 READY後復用既有sprite生命週期，九張只允許(128,8)及右側錨點。ALLY接入12/31，#12–#15四局部24×32、#16–#30十五小圖仍未接入；全部31美術定稿保持。

九張正式圖面與原版實際來源frame的獨立全畫面RGBA差0；原位8×8、冷載、開關、省略圖面、錯來源、未READY位置與其他圖號負對照通過。第一版測試漏給Layer.Frame RGB，圖層按契約停止繪製；補相同原版RGB後同工具鏈乾淨重跑。一般套件舊負對照仍把已READY #3列拒絕，改成尚未READY #12後24通過／23專用fixture明示跳過；九ALLY、60敵人、原版八轉換、40中途及兩敏頓核心均另明示執行沒有跳過。兩個失敗都是驗證輸入／舊期望，未放寬產品來源或原位。

合併主題theme-enemy360-ally12-effects-maze-B-v1-20261006，5,654PBL＋78builtin_masks＋256MAZE／394PNG。六正常保存點12次44欄位／DOS及12舊圖面／合成相同。新版正式前端SHA9b5fc6609a4bd1831cf228a8810bd25290a1d58a224e2ff705717a27bc8c8c11，349實際非標準來源保存；普通Space GUI36張中35張共1,975效果格符合獨立B模板，BEAM／FIGHT／MASK及敏頓三小圖可見，F10保存、30秒自然退出碼0。GUI首版cwd為/，相對text／font找不到；v2用/src，相同binary、state與鍵序重跑。Xvfb有trap，沒有退回主機執行。有限格模板来自前批原版邊界，不作新時戳或整屏同幀；ALLY新九張正常GUI尚未驗，不增加計數。

入口均在既有研究根：ally-portraits-ida-v1-20261006.py／.json及run-ally-portraits-ida-v1-20261006.py，ally-portraits-controlled-v1-20261006.go／.bin／-build.json及同名execution.json，verify-ally-portraits-source-v1-20261006.py／ally-portraits-source-independent-v1-20261006.json。正式測試apps/psychicwar/theme/ally_portraits_test.go；九份RGBA在ally12-validation-v2-20261006/，獨立ally12-render-independent-v2-20261006.json。六正常保存點與12-machine.json在enemy360-ally12-normal-v1-20261006/；60圖面、敏頓與一般套件入口見CONTEXT。GUI入口enemy360-ally12-gui-v2-20261006.sh／.py及同名目錄，獨立enemy360-ally12-gui-cells-independent-v1-20261006.json。

本批來源、原版、PNG、state、提示及archive只留本機。保全入口preserve-enemy360-ally12-v1-20261006.py、enemy360-ally12-source-snapshot-v1-20261006.tar.gz及enemy360-ally12-final-audit-v1-20261006.json。CONTEXT／AGENTS／README／worklist與#34同步，#34保持OPEN，#44不改。完整HD、19ALLY來源、正常隊伍／GUI、其他基底／效果、DAT、動畫、效能、權利與封包仍待完成；沒有commit／push／PR／tag／發行。

## 200. ALLY十五小圖、四裝備與四隊伍原位的接手核對（2026-10-07）

本輪接手前輪未提交的024 §1.53–§1.54 READY實作及收據，沿已定稿美術。研究根仍為workplace/ida/hd-ally-recruit-20261004/，本節以下相對入口均在此。工具Go1.24.13、Python3.11.2、ImageMagick6.9.11-60，psychicwar-go-ebiten image SHA083e45e6bc0f01ca46ba0774581572c80a607120431b530de72cdd6ffb36f2f7；IDA證據沿前輪9.4，正式DB未改。原版PW.EXE SHA88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49，ALLY.PBL SHAc88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219。原版唯讀，輸出UID/GID1000；network none、--rm、CPU1–2、資源及外層逾時限制。

【confirmed，限既有原版受控來源】#16–#30十五張16×16小圖共31原位的RLE COPY、兩解碼器及單像素負對照通過。座標表與原始bytes沿024 §1.53；原版直接RLE畫圖，不經8705，正式程式以完整來源與原版全圖辨識，不猜掛鉤。入口ally-remaining-ida-v7-20261006.json、ally-remaining-controlled-v2-20261006/execution.json、ally-small-source-independent-v1-20261006.json。IDA EA換runtime0161:IP為EA−10510h，MZ fileoffset為976+EA−10000h；原名、bytes與分類保留。

【confirmed，限既有受控合成】四裝備#12–#15合入12人物、四隊伍槽，共192組RAM及完整COPY；另48人物原位COPY及256人數輸入通過。原版driver2跳過色號2／Ah，完整384-byte工作源共有60唯一身份。四原位依槽0–3為(264,152)、(232,152)、(200,152)、(168,152)。覆繪只讀1175:AEE6指標+46h的低兩bit、人數與1175:AF02工作源；未知來源／driver與指標越界回退。入口ally-equipment-controlled-v4-20261006、ally-equipment-source-independent-v2-20261006.json、ally-loader-return-ida-v1及ally-party-count-ida-v1；原始地址／合成規則沿024 §1.54。

【confirmed，限定圖面】裝備定稿PNG沿RGB(85,255,85)底色的最小alpha轉換，再與人物合成，原PNG不改。240份人物／裝備完整RGBA、31小圖原位完整RGBA、144合成B場景與48獨立B圖面通過，移位、錨點、未知來源／driver／人數、冷載、開關及省略負對照有效。正式實作ally.go／ally_equipment.go／theme.go／battle.go，驗證ally_small_test.go／ally_equipment_test.go。收據ally31-render-independent-v2、ally31-small-render-independent-v1、enemy360-ally31-render-independent-v1，日期20261006。來源接入ALLY31/31，正常選裝與其他人物GUI仍未知。

【confirmed，既有正常保存點接續】六起點載入及100,000步接續，12次44機器欄位與DOS相同。11舊RGBA相同；格斯丁提bank4/group2兩隊員ALLY0／2的真實中途，原版入口676817253步、核對點676817780步，156格符合獨立來源與B公式、4格保持原版，錯來源負對照有效。入口enemy360-ally31-normal-v2-20261006/runtime.json、ally31-normal-independent-v2-20261006.json。60敵人模型、八受控身體轉換、40短圖中途及敏頓4,166邊界／2,083轉換保持。一般套件24通過／25專用fixture跳過；來源清單另明示執行通過，不計跳過為成功。

現行本機主題theme-enemy360-ally31-effects-maze-B-v1-20261006，5,746PBL＋78builtin_masks＋256MAZE／413PNG。audit-ally31-handoff-v1-20261007.py核對31凍結素材、既有四份獨立收據及實際編譯來源。缺失的18份art-in PNG別名由完全相同SHA的現行素材復原；舊比較器由原碼重生，binary SHA6e4a6979d5fdef33b4c204982583e9952af8dbab834ac39650c6ea7d009c1e73與舊收據完全相同。入口restore-ally31-reference-inputs-v1-20261007.py／ally31-reference-inputs-restored-v1-20261007.json。歷史收據及原版不改；四份輸入直接路徑全部相符。

本輪核對後接手實作，沒有重新生成美術或改規則／schema。來源READY及受控RGBA不能替代正常選裝、全部圖號GUI、動畫或完整HD交付。下一節記本輪新的正式GUI／DAT／效能驗收。

## 201. ALLY31正式前端、原版DAT與效能（2026-10-07）

【confirmed，限定新建正式前端】enemy360-ally31-frontend-v1-20261007.bin SHA9cca6ad5ae93cbcba2ba67d3d034975dbf11f6a6009f2aece1f6f6bb7ad9ab59，350實際非標準來源全文及SHA保存在同名-build.json；建置入口build-ally31-frontend-v1-20261007.py。主題與工具版本沿§200。所有起點為既有正常玩家保存點，seed於執行前由同state固定，不寫seed、不重擲，750cycles沿正式前端讀檔後設定。

隊伍／道具24視窗的角色兩原位、肖像切換／撤圖、F10／F11與自然退出0通過，獨立PNG負對照有效。普通Space戰鬥36視窗中35份、1,739效果格吻合，FIGHT590、BEAM1048、MASK31及敏頓小圖#21／#22／#23各42／45／75來源格可見。重疊格不相加當總格。模板是前批原版邊界，不作新截圖時戳；仍限8×8格，整屏同幀及全動畫另驗。入口ally31-gui-v1-20261007.sh、ally31-gui-{party,battle}-v1-20261007/及獨立party／battle-cells收據。敵人正常GUI30/360、ALLY正常#0–#2保持。

【confirmed，真正原版存讀檔】從正常Options及SELECT起點，以正式F11還原中文字面，再用普通原版鍵保存及讀取hd31.dat。新主題GUI的512 bytes與原版精確鍵序保存結果完全相同，SHA b32d25e6298fb0dd7a14de3202cd80a53479826e0cda6793b94f5081030c58f8；載回玩家區52 bytes相同，DAT／玩家單byte負對照有效。載回後切換凱與敏頓道具肖像，24視窗、66個完整72×96角色區與獨立原版PBL／定稿PNG相同，HD關閉回到原版RGB的負對照有效。入口ally31-dat-gui-v1-20261007.sh／.py、ally31-dat-{save,load}-v1-20261007/、verify-ally31-dat-crops-v1-20261007.py及其獨立收據。

【confirmed，同起點／同原版鍵步數】存檔、讀檔與普通Space三段GUI錄製各自重播到實際F10終點，44個機器欄位、指令數及完整DOS全部相同。負對照翻動CPU、RAM及port均有效。入口prepare-ally31-control-v2、run-ally31-controls-v2及ally31-{save,load,battle}-control-v2，日期20261007；總收據ally31-gui-dat-machine-independent-v2-20261007.json。重播binary SHAdd64ec5cda24d667b53f8ca402b9d17f830fb048408af70cc7703fcb38f3128c，62實際來源保全。原版state解析工具完整44欄位結構與目前machineState相同，不略過欄位。

讀檔第一輪只有線性01096／01097兩byte不同，DOS相同。獨立差分ally31-load-machine-difference-v1-20261007.json保留。dosgolem internal/dos/find.go的emitFind在DTA+16h寫入dosDateTime(info.ModTime())；此點DTA為1000:0080，因此差異即01096。GUI複製DAT與重播複製DAT的時間不同。v2維持實際GUI讀檔輸入的mtime，重跑完整44欄位與DOS即相同，沒有排除這兩byte或改原版RAM。這是驗證環境差異，未列產品缺陷。

【confirmed，限定本機成本】正式main以Go覆映射加入計時，原碼不改；量測binary SHAd591b49ca38c21b1e21754536a5ef89175c81dc42867791a643f2c69a36c239d，350實際來源保全。用本輪正常DAT載回的道具畫面，三模式起跑load 6.26／4.29／4.74，統計10–30秒，所有程序自然退出0及計數破壞負對照有效。

| 模式 | Frame均值ms | Theme.Frame均值ms | Draw CPU均值ms | 實際繪圖FPS |
|---|---:|---:|---:|---:|
| 未載入 | 1.543 | 0.00024 | 3.197 | 40.58 |
| 載入關閉 | 3.207 | 2.037 | 2.380 | 53.23 |
| 開啟 | 3.485 | 2.213 | 4.348 | 38.14 |

限兩CPU、軟體OpenGL、null音訊，沒有速度門檻；各段共享主機負載不同，FPS不外推真機、戰鬥或封包。入口prepare-ally31-performance-v2、run-ally31-performance-v3、verify-ally31-performance-v1及ally31-performance-v1/verified.json，日期20261007。

研究工具失敗保留：probe覆映射缺die輔助函式、舊計時探針漏目前英文標籤分支、舊比較器／道具fixture二進位或路徑不存在、loaded-off起跑load8.24停止。前兩項依實際源碼修正；缺件用固定SHA來源重生；道具fixture改用本輪正常DAT載回；高負載段未開跑，保留已驗absent，降載後只續兩模式。沒有退回主機執行或降低驗收標準。初次DAT工具執行的自動審查逾時，依工具指示重試一次後核准，未造成檔案寫入。

保全入口preserve-ally31-handoff-v1-20261007.py、ally31-handoff-source-snapshot-v1-20261007.tar.gz、ally31-handoff-final-audit-v1-20261007.json，均在既有研究根。原版、PNG、state、DAT、完整來源及archive只留本機。CONTEXT、AGENTS、README、024及worklist回填；#34核讀OPEN、#44不改。本批僅本機提交，未push／PR／tag／發行。其他效果圖號、圖像／動畫、其他人物／選裝正常GUI、剩餘場景DAT、權利與封包仍待完成，完整HD不稱CONFORMED。
