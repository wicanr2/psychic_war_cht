#!/bin/sh
# 推廣片合成（skill `game-promo-video-ffmpeg`）。在 tools/video.sh 的容器裡跑：
#
#   tools/video.sh sh tools/promo/make.sh
#
# 輸出 workplace/promo/psychic-war-promo.mp4。
#
# ⚠ 配樂是 workplace/dosboxx-audio/title-adlib.wav —— **DOSBox-X 跑原版錄下來的輸出**。
#   不可以換成 workplace/audio/golem-opl-title.wav，那是我們自己寫的合成器算出來的
#   （`rulebook/93` 鐵則 1：配樂必須用原版實際素材，自產的逼近渲染一律不行）。
# ⚠ 不用 zoompan：`-loop 1 -t` 加 `fps` 濾鏡會讓 zoompan 的 d 變成「每個輸入幀輸出 d 幀」，
#   6 秒 25fps 算成兩萬多幀（skill 的雷 #1）。靜態圖加淡入淡出就夠。
set -eu
cd /src
. tools/promo/theme.sh

SHOT=/src
MUSIC=workplace/dosboxx-audio/title-adlib.wav
OUT=workplace/promo
TMP=$OUT/tmp
rm -rf "$TMP"; mkdir -p "$TMP" "$OUT"

[ -f "$MUSIC" ] || { echo "缺配樂 $MUSIC（tools/dosboxx-audio.sh 90 adlib）"; exit 2; }
for f in "$FONT_TITLE" "$FONT_BODY"; do
  [ -f "$f" ] || { echo "缺字型 $f"; exit 2; }
done

# bg：深色徑向漸層加掃描線母題（遊戲是 200 線 EGA，橫向細線是它的質感）
bg() {
  convert -size ${W}x${H} "radial-gradient:${BG_LITE}-${BG_DEEP}" \
    \( -size ${W}x${H} xc:none -fill "#ffffff10" \
       -draw "line 0,0 ${W},0" -write mpr:line +delete \
       -size ${W}x${H} tile:mpr:line \) -compose over -composite "$1" 2>/dev/null \
  || convert -size ${W}x${H} "radial-gradient:${BG_LITE}-${BG_DEEP}" "$1"
}

# card：標題卡。$1 out $2 中標 $3 英標 $4 副標
# ⚠ gravity center 下 -annotate 的幾何是 +x+y，y 直接寫負號（+0-110）。
#   寫成 +0+0-110 不是「y = 0-110」，那個尾巴會被吃掉，字就疊在一起。
card() {
  bg "$TMP/bg.png"
  convert "$TMP/bg.png" -gravity center \
    -font "$FONT_TITLE" \
    -fill "$ACCENT_DIM" -pointsize 76 -annotate +5-105 "$2" \
    -fill "$ACCENT"     -pointsize 76 -annotate +0-110 "$2" \
    -font "$FONT_BODY" \
    -fill "$TEXT" -pointsize 34 -annotate +0+10 "$3" \
    -fill "$DIM"  -pointsize 26 -annotate +0+90 "$4" \
    -stroke "$ACCENT" -strokewidth 2 -fill none \
    -draw "rectangle 40,40 $((W-40)),$((H-40))" "$1"
}

# slide：截圖置中加青框，底部字幕條。$1 out $2 截圖 $3 字幕
slide() {
  bg "$TMP/bg.png"
  convert "$SHOT/$2" -resize x596 -bordercolor "$ACCENT" -border 2 "$TMP/sc.png"
  convert "$TMP/bg.png" "$TMP/sc.png" -gravity north -geometry +0+6 -composite \
    -fill "#000000cc" -draw "rectangle 0,610 ${W},${H}" \
    -stroke "$ACCENT" -strokewidth 1 -draw "line 0,610 ${W},610" -stroke none \
    -font "$FONT_BODY" -fill "$TEXT" -gravity south -pointsize 32 -annotate +0+36 "$3" "$1"
}

# split：左右對照。$1 out $2 左圖 $3 右圖 $4 左標 $5 右標 $6 字幕
split() {
  bg "$TMP/bg.png"
  half=$(( (W - 150) / 2 ))   # 中間留 150 給 F2
  convert "$SHOT/$2" -resize ${half}x -bordercolor "$DIM"    -border 2 "$TMP/l.png"
  convert "$SHOT/$3" -resize ${half}x -bordercolor "$ACCENT" -border 2 "$TMP/r.png"
  convert "$TMP/bg.png" \
    "$TMP/l.png" -gravity northwest -geometry +20+150 -composite \
    "$TMP/r.png" -gravity northeast -geometry +20+150 -composite \
    -font "$FONT_BODY" -gravity northwest -fill "$DIM"    -pointsize 30 -annotate +20+100 "$4" \
    -gravity northeast -fill "$ACCENT" -pointsize 30 -annotate +20+100 "$5" \
    -gravity center -font "$FONT_TITLE" -fill "$ACCENT" -pointsize 44 -annotate +0-130 "F2" \
    -fill "#000000cc" -gravity south -draw "rectangle 0,610 ${W},${H}" \
    -stroke "$ACCENT" -strokewidth 1 -draw "line 0,610 ${W},610" -stroke none \
    -font "$FONT_BODY" -fill "$TEXT" -gravity south -pointsize 32 -annotate +0+36 "$6" "$1"
}

# clip：一張圖一段，靜態加淡入淡出。$1 png $2 mp4 $3 秒
clip() {
  fo=$(awk "BEGIN{print $3-0.6}")
  ffmpeg -y -loglevel error -loop 1 -i "$1" -t "$3" -r $FPS \
    -vf "fade=t=in:st=0:d=0.6,fade=t=out:st=$fo:d=0.6,format=yuv420p" \
    -threads 2 -c:v libx264 -preset veryfast -pix_fmt yuv420p "$2"
}

# ===== 分鏡 =====
card "$TMP/00.png" "銀河超能力戰記" "PSYCHIC WAR ── COSMIC SOLDIER 2" \
  "工画堂スタジオ 1987 ・ DOS 英文版 1989 ・ 繁體中文化"

split "$TMP/01.png" workplace/hotkeys/b-english.png workplace/hotkeys/a-chinese.png \
  "原版英文" "繁體中文" "同一個畫面，F2 隨時切換；切回英文時與原版逐像素相同"

slide "$TMP/02.png" workplace/playtest/opencheck/025.png \
  "西元 3656 年，太空戰艦正從 KGD 星團躍遷而來"

slide "$TMP/03.png" docs/images/opening-story.png \
  "開場故事：和生化複合人凱拉一起降落在貿易站薩瑪"

slide "$TMP/04.png" docs/images/maze-panel.png \
  "操作面板與狀態欄是畫進圖檔的字，靠畫面內容比對換成中文"

slide "$TMP/05.png" docs/images/dialogue-menu.png \
  "對白與選單：1,313 則文本全部有譯文"

slide "$TMP/06.png" docs/images/battle.png \
  "遭遇戰要按住空白鍵，連按打不贏 ── 原版的規則一行都沒改"

slide "$TMP/07.png" docs/images/help-f1.png \
  "原版沒有的：F1 說明、F3 自動地圖、F10／F11 即時存檔"

card "$TMP/99.png" "銀河超能力戰記" "繁體中文化 ・ RRSAL-1.0" \
  "原版執行檔跑在 dosgolem 上 ・ 需自備原版 ・ github.com/wicanr2"

# ===== 串起來 =====
LIST="$TMP/list.txt"; : > "$LIST"
for f in 00 01 02 03 04 05 06 07 99; do
  case "$f" in
    00|99) s=$PACE_CARD ;;
    01)    s=$PACE_SPLIT ;;
    *)     s=$PACE_SLIDE ;;
  esac
  clip "$TMP/$f.png" "$TMP/s_$f.mp4" "$s"
  echo "file 's_$f.mp4'" >> "$LIST"  # concat 的路徑是相對 list 檔所在目錄
done
ffmpeg -y -loglevel error -f concat -safe 0 -i "$LIST" \
  -threads 2 -c:v libx264 -preset veryfast -pix_fmt yuv420p "$TMP/silent.mp4"

DUR=$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$TMP/silent.mp4")
FO=$(awk "BEGIN{print $DUR-3}")
# 配樂先無限循環再剪到影片長度，不用 -shortest：配樂比影片短的話 -shortest 會把結尾卡砍掉。
ffmpeg -y -loglevel error -i "$TMP/silent.mp4" -i "$MUSIC" \
  -filter_complex "[1:a]aloop=loop=-1:size=2000000000,atrim=0:$DUR,afade=t=in:st=0:d=2,afade=t=out:st=$FO:d=3[a]" \
  -map 0:v -map "[a]" -threads 2 -c:v libx264 -preset veryfast -c:a aac -b:a 192k \
  -movflags +faststart "$OUT/psychic-war-promo.mp4"

echo "== 產出"
ffprobe -v error -show_entries format=duration -of csv=p=0 "$OUT/psychic-war-promo.mp4"
ffprobe -v error -select_streams v -show_entries stream=duration -of csv=p=0 "$OUT/psychic-war-promo.mp4"
ffprobe -v error -select_streams a -show_entries stream=duration -of csv=p=0 "$OUT/psychic-war-promo.mp4"
ls -la "$OUT/psychic-war-promo.mp4"
