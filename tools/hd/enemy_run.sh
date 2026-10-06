#!/usr/bin/env bash
# 派畫圖代理重繪一個 ENEMY 檔（成組，docs/spec/024 §3.2）。
#
#   tools/hd/enemy_run.sh ENEMY00
#
# 分組工具只提出像素相似的候選，須先檢視參照確認角色；
# 確認同角色後才成組重畫（docs/re/038 §17）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
NAME="$1"
tools/py.sh tools/hd/enemy_groups.py "workplace/hd/spec-$NAME.md" "$NAME"

PROMPT=$(cat <<EOF
你是《銀河超能力戰記》(Psychic War: Cosmic Soldier 2, 工画堂 1987) 繁體中文化專案的美術。
工作目錄 $ROOT。

任務：把 $NAME.PBL 的 30 張 sprite「基於原版重新繪製」成高解析度。
清單在 workplace/hd/spec-$NAME.md。

**表格每列只是像素相似的候選，角色／動作語意標成假說，不能當成已證實。**
先檢視每張參照，確認哪些屬於同角色；確認後保持原版的臉、髮色、服裝與配色，
同角色各格只有原本的姿勢差異。不要依候選門檻擅自把不同角色合併。

硬性要求：
1. 尺寸照表（72x96 或 48x48），一個像素都不能差。模型原生尺寸不符時自己裁切縮放。
2. 姿勢、造型、配色照原版。不要換角色設計、不要增刪配件。
3. 原版用棋盤格抖色表現中間色調，重繪要畫成真正的漸層。
4. 一個文字都不要畫。
5. 輸出很小（72x96、48x48），**線條要簡潔、對比要夠**，細節堆太多縮下去會糊。
6. 原版是純黑的背景就畫純黑 #000000。
7. 風格是 1987 年日式科幻動畫的賽璐珞質感，不要現代 3D、不要照片寫實。

做完跑 tools/py.sh tools/hd/verify_batch.py $NAME，把結果貼在最後。

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
tools/py.sh tools/hd/verify_batch.py "$NAME"
