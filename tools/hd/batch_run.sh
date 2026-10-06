#!/usr/bin/env bash
# 派畫圖代理重繪一批圖檔（docs/spec/024 §3.2）。
#
#   tools/hd/batch_run.sh <PBL 名> [說明] [跳過的圖號,…]
#
# 產生規格 → 跑 codex → log 在 workplace/hd/log/<名>.log。
# ⚠ 提示詞一律用 stdin 餵：codex exec 的 -i/--image 是可變長度參數，
#   寫成位置參數會被它吃掉，症狀是「Reading prompt from stdin」然後空轉（docs/re/038 §13.1）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
NAME="$1"
NOTE="${2:-進入地點時顯示的場景插圖}"
SKIP="${3:-}"
VERIFY=(tools/py.sh tools/hd/verify_batch.py "$NAME")
if [[ -n "$SKIP" ]]; then
  VERIFY+=(--skip "$NAME:$SKIP")
fi

tools/py.sh tools/hd/batch_spec.py "$NAME" "workplace/hd/spec-$NAME.md" "$SKIP"
N=$(grep -c '^| [0-9]' "workplace/hd/spec-$NAME.md")

PROMPT=$(cat <<EOF
你是《銀河超能力戰記》(Psychic War: Cosmic Soldier 2, 工画堂 1987) 繁體中文化專案的美術。
工作目錄 $ROOT。

任務：把 $NAME.PBL 的 $N 張圖「基於原版重新繪製」成高解析度。
清單在 workplace/hd/spec-$NAME.md，逐張照做（每張的參照圖、輸出路徑與精確尺寸都在表裡）。

原版是 EGA 320x200 十六色的遊戲，這批是$NOTE。

硬性要求：
1. 尺寸照表，一個像素都不能差。模型原生尺寸不符時自己裁切縮放到位。
2. 構圖、主體位置、配色照原版。不要重新構圖、不要換視角、不要增刪主要物件。
3. 原版用棋盤格抖色表現中間色調，重繪要畫成真正的漸層。
4. **一個文字都不要畫。** 清單第五欄列出的文字框要畫成乾淨的招牌底
   （跟周圍一致的平面），遊戲執行時中文會蓋上去，畫了英文會從底下透出來。
5. 風格是 1987 年日式科幻動畫的賽璐珞質感，不要現代 3D、不要照片寫實、不要厚塗。
6. 原版是純黑的背景就畫純黑 #000000。
7. 這批有些輸出很小（例如 48x48、72x96）。**線條要簡潔、對比要夠**，
   細節堆太多在縮到目標尺寸之後會糊成一團。先想「縮到那個尺寸還看得出是什麼」。

做法：一張一張處理，每張畫完先確認尺寸再做下一張。
全部做完跑 ${VERIFY[*]}，把結果貼在最後。排除圖號必須照本次清單，不可自動略過缺檔。

邊界（違反會造成事故，務必遵守）：
- 只能寫 workplace/hd/art-in/ 底下的檔案。不要改動 repo 裡任何其他檔案。
- 不要 git add / commit / push / 開分支。
- 不要跑任何 docker image / system / volume / builder prune，不要 docker rmi，不要停別人的 container。
- 不要碰 ~/.cache/ 或 ~/.claude/。
- workplace/hd/art-in/ 底下已經有的檔案是別批的成品，不要動。
EOF
)
mkdir -p workplace/hd/log
printf '%s\n' "$PROMPT" | timeout 180m codex exec -s workspace-write \
  -c sandbox_workspace_write.network_access=true > "workplace/hd/log/$NAME.log" 2>&1
echo "$NAME 結束碼 $?"
"${VERIFY[@]}"
