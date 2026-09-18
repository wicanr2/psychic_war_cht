# 034：推廣片

狀態：量測紀錄（2026-09-19）。方法 skill `game-promo-video-ffmpeg`；素材來源的硬規則 `rulebook/93`。

## 1. 結論

`workplace/promo/psychic-war-promo.mp4`，62.000 秒，1280×720、25 fps，1.9 MB。
視訊與音訊等長（都是 62.000 秒）。

| 項目 | 值 |
|---|---|
| 分鏡 | 9 段：標題卡、中英對照、開場星圖字幕、開場故事、迷宮面板、對白選單、戰鬥、輔助功能、結尾卡 |
| 配樂 | `workplace/dosboxx-audio/title-adlib.wav` ── **DOSBox-X 跑原版錄下來的輸出** |
| 畫面 | 全部是實跑截圖（`pwstep`、前端 Xvfb），沒有 mockup、沒有重畫 |
| 音軌驗證 | 來源 mean −18.0 dB／max −1.1 dB；成片 mean −17.4 dB／max −1.1 dB（差在淡入淡出與剪裁），無 clipping |

**影片不進版控。** 它含原版的畫面與音樂，`workplace/` 已經 gitignore。
要對外公開之前先看 §4。

## 2. 配樂的來源（`rulebook/93` 鐵則 1）

這個專案裡有兩個聽起來都像「標題曲」的 WAV，只有一個能用：

| 檔案 | 怎麼來的 | 能不能當配樂 |
|---|---|---|
| `workplace/dosboxx-audio/title-adlib.wav` | DOSBox-X 執行原版 `PW.EXE`，`-soundrecord` 錄下來 | **可以** |
| `workplace/audio/golem-opl-title.wav` | 我們自己寫的 OPL2 合成器算出來的（`docs/re/032`） | **不可以** |

第二個即使三項保真度指標都過了門檻（`spec` 0.9117、`env` 0.8119、`chroma` 0.9836），
它仍然是「自產的逼近渲染」。鐵則不是拿相似度當判準，是拿**來源**當判準。
`tools/promo/make.sh` 的檔頭與 `CLAUDE.md` 的待決事項都寫死了這一條。

## 3. Theme：「EGA 星圖」

顏色不是挑的，是從實機截圖的色號直方圖取的（`tools/promo/theme.sh` 記著出處）：
黑 35,781 像素、灰 5,008、白 4,727、亮青 4,544、亮紅 3,606、亮藍 3,222。
亮青（EGA 11）本來就是遊戲 UI 的框線與箭頭色，拿它當 accent，片子和畫面是同一套顏色。

字體用黑體不用襯線：1987 年的日系科幻，襯線是西方奇幻的氣質。

三種版面輪流：標題卡、左右對照（中英，中間一個 `F2`）、截圖滿版加底部字幕條。
中英對照那一段用的是 `workplace/hotkeys/a-chinese.png` 與 `b-english.png`——
同一個狀態按 F2 切換出來的兩張，不是分別跑兩次。

## 4. 對外公開之前（`rulebook/93` 但書）

配樂是原版遊戲音樂，旋律的著作權屬於原作曲者。這與「品質上該不該用原版」是兩件事：

- 個人保存、內部 demo：照現在這樣用，素材與成片都不入庫。
- 上傳 YouTube 或社群：**先確認**要不要換成授權明確的曲子，或改用無音樂版本。

## 5. 踩到的

| 症狀 | 原因 |
|---|---|
| 標題與英文副標疊在一起 | `-annotate +0+0-110` 不是「y ＝ 0−110」。ImageMagick 的幾何是 `+x+y`，負號直接寫在 y：`+0-110`。多出來的那一段會被吃掉，不報錯 |
| `concat` 說 `Impossible to open 'tmp/tmp/s_00.mp4'` | concat demuxer 的 `file` 路徑是**相對 list 檔所在目錄**，不是相對 cwd。list 裡寫檔名就好 |
| 抽出來的幀全黑 | 每段之間淡出到黑再淡入，剛好抽在交界。抽幀要避開段落邊界 |
| 工具 image 建不起來 | 容器內 `apt-get update` 60 秒跑不完，`fonts-noto-cjk-extra` 那一包又特別大。改成只裝 `fonts-noto-cjk`；這台機器上已經有別的專案建好的同類 image，`PSYCHICWAR_VIDEO_IMAGE` 可以指過去（**只執行，不清理也不覆寫**） |

沒有用 `zoompan`（skill 的雷 #1：它的 `d` 是「每個輸入幀輸出 d 幀」，配上前置 `fps` 會把 6 秒算成兩萬多幀）。
靜態圖加淡入淡出就夠。

## 6. 重現

```sh
tools/video.sh sh tools/promo/make.sh
# 借用既有的同類 image：
PSYCHICWAR_VIDEO_IMAGE=<image> tools/video.sh sh tools/promo/make.sh
```
