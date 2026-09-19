#!/usr/bin/env bash
# Windows 交叉編產物的驗收（docs/spec/021 §5；docs/re/035）。
#
#   tools/windows-verify.sh workplace/pkg-stage/win64/PsychicWar      # 目錄（打包前自檢）
#   tools/windows-verify.sh workplace/win-check/PsychicWar/PsychicWar.exe
#   tools/windows-verify.sh dist-all/PsychicWar-<版本>-win64.zip --run # 解開 zip ＋ wine 實跑
#
# 參數可以是 portable 目錄、`.exe` 或 `.zip`；路徑要在本 repo 底下。
#
# 兩段：
#
# **靜態**（預設，純 Python 解 PE，不必執行）：
#   1. PE 格式：`MZ` ＋ `PE\0\0` ＋ PE32+（`0x20b`）。
#   2. 架構：COFF machine ＝ `0x8664`（x86-64）。
#   3. 子系統：2 ＝ GUI（`-H windowsgui`；3 ＝ CUI 表示旗標掉了，雙擊會多一個主控台視窗）。
#   4. 匯入的 DLL：只能是 Windows 自己就有的。mingw runtime（`libgcc_s_seh-1.dll`、
#      `libwinpthread-1.dll`、`libstdc++-6.dll`）出現就是漏了靜態連結，玩家那邊會
#      「缺少 DLL，無法執行」。CGO_ENABLED=0 的 Go 執行檔只匯入 `kernel32.dll`，
#      其餘（`user32`、`opengl32`、`d3d11`…）由 purego 在執行期 `LoadLibrary`，
#      所以另外把執行期會載的 DLL 名字一起列出來看（那些也必須是系統 DLL）。
#   5. 內容證據：這份程式一定會有的字串，證明編到的是這份原始碼。
#   （目錄或 zip 還會多驗：`text/` 56 個 JSON、`font/` 兩個字型、`troubleshoot.bat`。）
#
# **實跑**（加 `--run`）：解開到 `workplace/win-check/`，在 wine ＋ Xvfb 底下跑
#   `-quit-after`，截圖、看結束碼、看 `%APPDATA%` 底下有沒有寫出存檔。
#   ⚠ **wine 不是 Windows**。這一段只證明「不是立刻崩潰、路徑解析成立」，
#   真正的 Windows 上沒有跑過（docs/re/035 §5）。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
TARGET="${1:?用法：windows-verify.sh <目錄、.exe 或 .zip> [--run]}"
MODE="${2:-}"
IMAGE="${PSYCHICWAR_WINE_IMAGE:-psychicwar-wine}"
EXE_NAME="PsychicWar.exe"

case "$TARGET" in /*) ABS="$TARGET" ;; *) ABS="$ROOT/$TARGET" ;; esac
[[ -e "$ABS" ]] || { echo "找不到 $ABS" >&2; exit 1; }
case "$ABS" in "$ROOT"/*) REL="${ABS#"$ROOT"/}" ;; *) echo "目標要在 repo 底下（$ROOT）" >&2; exit 1 ;; esac

# zip：先解開，後面都對解開的那份驗（rulebook/82：驗實際打包產物）。
UNPACK=""
case "$REL" in
  *.zip)
    UNPACK="workplace/win-check"
    rm -rf "$ROOT/$UNPACK"; mkdir -p "$ROOT/$UNPACK"
    unzip -q "$ROOT/$REL" -d "$ROOT/$UNPACK"
    REL="$UNPACK/PsychicWar"
    echo "== 解開 $TARGET → $REL"
    ;;
esac

# --- 靜態：PE 標頭、匯入表、內容證據 ---------------------------------------------
tools/py.sh - "$REL" "$EXE_NAME" <<'PY'
import pathlib, struct, sys

target = pathlib.Path("/src") / sys.argv[1]
exe_name = sys.argv[2]
fail = []

if target.is_dir():
    print("== portable 目錄：%s" % sys.argv[1])
    exe = target / exe_name
    if not exe.is_file():
        print("缺 %s" % exe_name, file=sys.stderr); sys.exit(1)
    njson = len(list((target / "text").glob("*.json"))) if (target / "text").is_dir() else 0
    nfont = len(list((target / "font").glob("*.golemfnt"))) if (target / "font").is_dir() else 0
    print("-- text/ %d 個 JSON、font/ %d 個字型" % (njson, nfont))
    if njson < 50: fail.append("text/ 只有 %d 個 JSON" % njson)
    if nfont != 2: fail.append("font/ 應該有 cjk16、cjk24 兩個，實際 %d" % nfont)
    for f in ("troubleshoot.bat", "troubleshoot.txt", "README.md", "LICENSE"):
        if not (target / f).is_file(): fail.append("缺 " + f)
        else: print("-- 有 %s" % f)
    if (target / "original" / "PW.EXE").is_file():
        print("-- original\\PW.EXE 在裡面（-with-data 變體）")
else:
    exe = target

data = exe.read_bytes()
print("== 執行檔：%s（%d bytes）" % (exe.name, len(data)))

# 1. PE 格式
if data[:2] != b"MZ": fail.append("開頭不是 MZ")
pe = struct.unpack_from("<I", data, 0x3C)[0]
if data[pe:pe+4] != b"PE\0\0": fail.append("找不到 PE 簽章")
machine, nsec, _, _, _, optsz, _ = struct.unpack_from("<HHIIIHH", data, pe + 4)
opt = pe + 24
magic = struct.unpack_from("<H", data, opt)[0]
print("-- machine 0x%04X、optional magic 0x%03X、%d 個 section" % (machine, magic, nsec))
if magic != 0x20B: fail.append("不是 PE32+（magic 0x%03X）" % magic)

# 2. 架構
if machine != 0x8664: fail.append("架構不是 x86-64（machine 0x%04X）" % machine)

# 3. 子系統
subsystem = struct.unpack_from("<H", data, opt + 68)[0]
print("-- subsystem %d（%s）" % (subsystem, {2: "GUI", 3: "主控台"}.get(subsystem, "?")))
if subsystem != 2: fail.append("subsystem %d 不是 2（GUI）：-H windowsgui 掉了" % subsystem)

# 4. 匯入表
secs = []
for i in range(nsec):
    off = opt + optsz + 40 * i
    name = data[off:off+8].rstrip(b"\0").decode("latin1")
    vsz, va, rawsz, raw = struct.unpack_from("<IIII", data, off + 8)
    secs.append((name, va, vsz, raw, rawsz))

def rva2off(rva):
    for _, va, vsz, raw, rawsz in secs:
        if va <= rva < va + max(vsz, rawsz):
            return raw + (rva - va)
    return None

imp_rva, imp_sz = struct.unpack_from("<II", data, opt + 112 + 8 * 1)
dlls = []
if imp_rva:
    off = rva2off(imp_rva)
    while True:
        desc = data[off:off+20]
        if len(desc) < 20 or desc == b"\0" * 20:
            break
        name_rva = struct.unpack_from("<I", desc, 12)[0]
        if not name_rva:
            break
        no = rva2off(name_rva)
        dlls.append(data[no:data.index(b"\0", no)].decode("latin1"))
        off += 20
print("-- 匯入的 DLL（%d）：%s" % (len(dlls), ", ".join(dlls) or "（無）"))
bad = [d for d in dlls if d.lower().startswith(("libgcc", "libwinpthread", "libstdc++", "libssp", "msvcp", "vcruntime"))]
if bad: fail.append("匯入了非系統 DLL（要靜態連結或收進 zip）：%s" % ", ".join(bad))

# 執行期才載的 DLL（purego／LoadLibrary）不在匯入表裡，只能從字串常數找。
# ⚠ Go 的字串全部黏成一大塊、沒有分隔，所以**不能**用正規式撈出名單（會撈到
# 「前一個字串的結尾＋dll 名」這種黏在一起的東西）。改成逐個名字去查有沒有出現：
# 問「這個名字在不在」答得準，問「總共有哪些名字」答不準。
blob = data.lower()
known = ["kernel32.dll", "user32.dll", "gdi32.dll", "opengl32.dll", "d3d11.dll", "d3d12.dll",
         "dxgi.dll", "d3dcompiler_47.dll", "winmm.dll", "ole32.dll", "shell32.dll",
         "advapi32.dll", "ntdll.dll", "imm32.dll", "shcore.dll", "dwmapi.dll",
         "xinput1_4.dll", "dinput8.dll", "gameinput.dll", "bcryptprimitives.dll",
         "ws2_32.dll", "psapi.dll", "version.dll", "sechost.dll"]
found = [d for d in known if d.encode() in blob]
print("-- 執行期會載的系統 DLL（字串裡找得到的，%d／%d）：%s" % (len(found), len(known), " ".join(found)))
for d in ("opengl32.dll", "user32.dll"):
    if d not in found: fail.append("字串裡找不到 %s（Ebiten 的 Windows 後端靠 purego 載它）" % d)
late_bad = [d for d in ("libgcc_s_seh-1.dll", "libgcc_s_dw2-1.dll", "libwinpthread-1.dll",
                        "libstdc++-6.dll", "vcruntime140.dll", "msvcp140.dll")
            if d.encode() in blob]
if late_bad: fail.append("執行期會載 mingw／MSVC runtime：%s" % ", ".join(late_bad))

# 5. 內容證據
for s, why in (("PsychicWar", "%APPDATA%\\PsychicWar 存檔落點"),
               ("cjk24.golemfnt", "中文字型子集"),
               ("銀河超能力戰記", "視窗標題"),
               ("original", "OrigDir() 找執行檔旁的 original\\")):
    n = data.count(s.encode("utf-8"))
    print("-- 內容證據 %r（%s）：%d 處" % (s, why, n))
    if n == 0: fail.append("binary 裡找不到 %r" % s)
n = data.count(b"SDL2")
print("-- 反向樣本 'SDL2'：%d 處（要是 0，證明這個檢查不是什麼都找得到）" % n)
if n: fail.append("反向樣本 SDL2 竟然找得到")

if fail:
    print("\n".join("✗ " + f for f in fail), file=sys.stderr)
    sys.exit(1)
print("靜態五道全過（**結構驗收，不是功能驗收**）")
PY

[[ "$MODE" == "--run" ]] || exit 0

# --- 實跑：wine ＋ Xvfb ----------------------------------------------------------
if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
  echo "[windows-verify] 第一次使用，先 build $IMAGE（要網路）" >&2
  docker build -t "$IMAGE" -f "$ROOT/tools/docker/wine.Dockerfile" "$ROOT/tools/docker"
fi

OUT="workplace/win-check-out"
mkdir -p "$ROOT/$OUT"
# 原版素材：可散布版要靠它才跑得起來；-with-data 版刻意**不掛**，驗包內的 original\。
ORIGMOUNT=()
if [[ "${PSYCHICWAR_NO_ORIG:-}" != 1 && -d "$ROOT/workplace/original" ]]; then
  ORIGMOUNT=(-v "$ROOT/workplace/original:/orig:ro")
fi
# ⚠ WINEPREFIX 不能放 /tmp（sticky bit，wine 拒絕），所以另外掛一個 /wine tmpfs 當 HOME。
# ⚠ `WINEDLLOVERRIDES=mscoree,mshtml=` 停掉 Mono 與 Gecko：少了它 `wineboot` 會跳
#    「要不要安裝」的對話框，而這裡是 `--network none` ＋ 無人看管的 Xvfb，會一路卡到 timeout。
timeout "${PSYCHICWAR_WINE_TIMEOUT:-10m}" docker run --rm --network none \
  --memory 4g --cpus "${PSYCHICWAR_WINE_CPUS:-2}" --pids-limit 512 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -e HOME=/wine -e WINEDEBUG="${WINEDEBUG:--all}" \
  -e WINEDLLOVERRIDES="mscoree,mshtml=" \
  -e "PW_PKG=$REL" -e "PW_OUT=$OUT" -e "PW_ARGS=${PSYCHICWAR_WINE_ARGS:--orig Z:\\orig\\psychic-war}" \
  -e "PW_GL=${EBITENGINE_GRAPHICS_LIBRARY:-opengl}" \
  -v "$ROOT:/src" "${ORIGMOUNT[@]}" \
  --tmpfs "/wine:exec,uid=$(id -u),gid=$(id -g),size=2g" \
  -w /src "$IMAGE" \
  bash -c '
    set -uo pipefail
    export WINEPREFIX=/wine/prefix EBITENGINE_GRAPHICS_LIBRARY=$PW_GL
    WINE=$(command -v wine || command -v wine64)
    [ -n "$WINE" ] || { echo "image 裡沒有 wine" >&2; exit 1; }
    mkdir -p /tmp/.X11-unix
    Xvfb :99 -screen 0 1280x800x24 -nolisten tcp -ac >/tmp/xvfb.log 2>&1 &
    xvfb_pid=$!
    i=0; while [ ! -S /tmp/.X11-unix/X99 ] && [ $i -lt 50 ]; do sleep 0.1; i=$((i+1)); done
    export DISPLAY=:99
    echo "== $($WINE --version)"
    $WINE wineboot -i >/dev/null 2>&1
    cd "/src/$PW_PKG"
    # ⚠ 截圖的子行程要記下 PID 單獨 wait。直接 `wait` 會連 Xvfb 一起等，而它不會結束，
    #    整個驗收會安靜地卡到 timeout。
    ( sleep 8; import -window root "/src/$PW_OUT/win.png" 2>/dev/null ) &
    shot_pid=$!
    $WINE ./PsychicWar.exe $PW_ARGS -audio null -quit-after 12s \
      -text-log "/src/$PW_OUT/text.jsonl" > "/src/$PW_OUT/run.log" 2>&1
    code=$?
    wait $shot_pid 2>/dev/null || true
    kill $xvfb_pid 2>/dev/null || true
    echo "結束碼 $code" | tee -a "/src/$PW_OUT/run.log"
    echo "== %APPDATA% 底下"
    find /wine/prefix/drive_c/users -iname "*.DAT" -o -iname "quick*" -o -ipath "*PsychicWar*" 2>/dev/null \
      | sed "s|/wine/prefix/drive_c/users|%USERPROFILE%|" | tee "/src/$PW_OUT/appdata.txt"
  '
echo "=== 執行紀錄（$OUT/run.log）"; tail -12 "$OUT/run.log" 2>/dev/null || echo "（沒有）"
echo "=== 轉譯紀錄筆數：$(wc -l < "$OUT/text.jsonl" 2>/dev/null || echo 0)"
ls -l "$OUT"/win.png 2>/dev/null || echo "（沒有截圖）"
