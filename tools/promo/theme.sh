#!/bin/sh
# 推廣片的設計 token（skill `game-promo-video-ffmpeg`「每片一個 theme」）。
#
# 顏色不是憑喜好挑的，是從實機截圖的色號直方圖取的（`tools/py.sh` 掃 docs/images/*.png）：
# 黑 35,781 像素、灰 5,008、白 4,727、亮青 4,544、亮紅 3,606、亮藍 3,222。
# 亮青（EGA 11）本來就是遊戲 UI 的框線色，拿它當 accent，片子跟畫面是同一套顏色。
THEME_NAME="EGA 星圖"

BG_DEEP='#000000'      # EGA 0：遊戲畫面的底色就是純黑
BG_LITE='#001028'      # 亮藍（85,85,255）壓暗，當漸層另一端
ACCENT='#55ffff'       # EGA 11 亮青：遊戲的框線與箭頭
ACCENT_DIM='#008b8b'   # 青的暗階，做浮雕陰影
TEXT='#ffffff'         # EGA 15
DIM='#aaaaaa'          # EGA 7
WARN='#ff5555'         # EGA 12

# 1987 年的日系科幻，用黑體；Serif 是西方奇幻的氣質，套在這裡會不對味。
FONT_TITLE=/usr/share/fonts/opentype/noto/NotoSansCJK-Bold.ttc
FONT_BODY=/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc

W=1280; H=720; FPS=25
PACE_CARD=6            # 標題卡與結尾卡
PACE_SLIDE=7           # 內容段
PACE_SPLIT=8           # 中英對照段（要看兩邊，給長一點）
MOTIF=scanline         # 母題：EGA 掃描線
