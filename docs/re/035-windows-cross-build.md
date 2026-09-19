# 035：在 Linux 上交叉編出 Windows 版

狀態：量測紀錄（2026-09-19）。對應 issue #39、#45（§3.3、§4）。規格 `docs/spec/021`（READY）§1.1、§3、§4、§5；
紀律 `rulebook/82`。對照組是 `docs/re/033`（macOS）。

## 1. 結論

**`dist-all/PsychicWar-<版本>-win64.zip` 與 `-with-data-win64.zip` 都產得出來，
靜態驗收全過，wine 底下起得來、讀得到資料檔、畫得出中文。**

⚠ **沒有在真正的 Windows 上跑過。** wine 過關不等於 Windows 過關（§5 寫了哪些事驗不到）。

| 項目 | 結果 |
|---|---|
| 工具鏈 | 與 Linux 版同一份：`psychicwar-go-ebiten`（go1.24.13），**不需要 mingw** |
| 建置指令 | `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -H windowsgui -X main.version=$VER"` |
| 執行檔 | `PsychicWar.exe`，PE32+、machine `0x8664`、subsystem 2（GUI）、8,533,504 bytes |
| 匯入表 | 只有 `kernel32.dll`。其餘（`user32`、`opengl32`、`d3d11`…）由 purego 在執行期載入 |
| mingw／MSVC runtime | **沒有**（`libgcc_s_seh-1`、`libwinpthread-1`、`libstdc++-6`、`vcruntime140` 全是 0） |
| 發行包 | 可散布 3,413,548 bytes；`-with-data` 3,648,971 bytes（多 106 個原版檔） |
| 存檔落點 | `%APPDATA%\PsychicWar`（`apps/psychicwar/paths.go` 這一輪新增的 windows 分支） |
| 建置時間 | 暖快取 1.8 秒（冷快取未量）；`--cpus 4`，開工時 load average 1.27 |
| 實跑驗收 | wine-9.0（Ubuntu 24.04）＋ Xvfb：可散布版與 `-with-data` 版都起得來；同狀態畫面與 Linux 產物**逐像素差 0** |
| 致命錯誤（issue #45，2026-09-19 補） | 彈 `MessageBoxW` ＋ 寫 `%APPDATA%\PsychicWar\psychicwar-error.log`（§4.1）。wine 底下的彈窗驗過，截圖為證（§5.2 第 F 項） |
| 圖示 | **仍然沒有**（§3.3 評估了三條路與代價） |
| **真機驗收** | **沒有做**（沒有 Windows 機器） |

## 2. 產物

```
PsychicWar/
  PsychicWar.exe         ← GUI 子系統，靜態，無外部 DLL 相依
  text/                  ← 56 個 JSON（譯文與原文欄位）
  font/                  ← cjk16.golemfnt、cjk24.golemfnt
  README.md、LICENSE
  troubleshoot.bat       ← 把 stderr 導成檔案再 pause（§4）
  troubleshoot.txt       ← 玩家看的說明（UTF-8 ＋ BOM）
  original/              ← 只有 -with-data 變體有，106 個原版檔
```

可散布版不含原版素材（`CLAUDE.md` [HARD]）。玩家把含 `PW.EXE` 的目錄複製成執行檔旁的
`original\`，或用 `-orig` 指過去。兩條都走 `OrigDir()`，與 AppImage、macOS 同一條程式碼。

## 3. 怎麼做的

| 檔案 | 做什麼 |
|---|---|
| `tools/windows-pack.sh <版本>` | 交叉編 → 組 portable 目錄 → 驗收 → `zip` |
| `tools/windows-verify.sh <目錄／exe／zip> [--run]` | 靜態驗收（解 PE）＋ wine 實跑 |
| `tools/docker/wine.Dockerfile` | wine ＋ Xvfb 的驗收 image（`psychicwar-wine`） |
| `tools/package.sh` | 多一個 `windows` 目標，`all` 也含它 |

### 3.1 先驗 `CGO_ENABLED=0` 行不行：行

這是這一輪唯一真正的未知數。macOS 那邊非要 osxcross 不可（`docs/re/033` §3.2），
Windows 不必：**Ebiten v2.9.9 的 Windows 後端用 purego 在執行期 `LoadLibrary`**，
連結期不需要任何 C 函式庫。

第一次試編的輸出（`CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./cmd/psychicwar`）：

```
# github.com/wicanr2/psychic_war_cht/cmd/psychicwar
cmd/psychicwar/main.go:360:14: undefined: syscall.Getrusage
cmd/psychicwar/main.go:360:32: undefined: syscall.RUSAGE_SELF
cmd/psychicwar/main.go:370:22: ru.Utime undefined (type syscall.Rusage has no field or method Utime)
cmd/psychicwar/main.go:370:40: ru.Stime undefined (type syscall.Rusage has no field or method Stime)
```

**四行全部是我們自己的程式碼，Ebiten 一行都沒有。** 這與 macOS 的失敗訊息
（`undefined: glfw.Window`、`v.initDisplayLink undefined`，全部在 Ebiten 內部）是兩件事：
那邊是後端本身要 cgo，這邊只是 `-stats` 的 `cpu_ms` 用了 POSIX 專屬的 `getrusage`。
`syscall.Getrusage` 在 `GOOS=windows` 底下不存在，**有沒有 cgo 都一樣**，
所以這不是「換成 mingw 就好」的問題。

修法是把取 CPU 時間拆成兩個檔：`apps/psychicwar/cputime_unix.go`（`getrusage`）與
`cputime_windows.go`（`GetProcessTimes` 的 kernel ＋ user FILETIME），
`cmd/psychicwar/main.go` 改呼叫 `psychicwar.CPUMillis()`。拆完就編得過。

### 3.2 存檔要走 `%APPDATA%`

`paths.go` 的 `SaveDir()` 原本只有 darwin 與 default 兩條，Windows 會掉進 default，
拿 `os.UserHomeDir()` 拼出 `C:\Users\<名>\.local\share\psychicwar`。
**那條路徑建得起來、寫得進去，所以不會報錯**，只是存檔落在沒有人會去看的地方，
這種「靜默走錯」比崩潰難發現（`rulebook/82` 第 4 點）。

新增的 windows 分支用 `os.UserConfigDir()`，它在 Windows 回的就是 `%APPDATA%`
（Roaming），拼成 `%APPDATA%\PsychicWar`。

### 3.3 沒做圖示（issue #45 第 3 項，評估過，這一輪仍然沒做）

`.exe` 的圖示要嵌成 PE 的 `.rsrc` 資源（`RT_GROUP_ICON` ＋ `RT_ICON`）。
**純 Go 工具鏈做不到**：`go build` 不會產生資源節，只會把它看得到的 `*.syso`
（COFF 目的檔）連進去。所以問題是「那個 `.syso` 從哪來」。

本機現況：`psychicwar-go-ebiten` 裡**沒有** `windres`、`llvm-rc`，也沒有任何 mingw 套件
（`ls /usr/bin | grep -c mingw` ＝ 0），`GOPROXY=off` 也裝不了 Go 寫的工具。三條路：

| 做法 | 代價 | 備註 |
|---|---|---|
| `binutils-mingw-w64-x86-64` 的 `windres` | 重建 image、要網路（這台容器網路 60–200 KB/s） | 只為了圖示裝一整套 binutils |
| `tc-hib/go-winres` 或 `akavel/rsrc` | 要網路 `go install`，多一個工具相依 | 產物一樣是 `.syso` |
| 自己用 Python 產 `.syso` | 不必網路，但要自己處理 COFF 節、資源目錄與 `IMAGE_REL_AMD64_ADDR32NB` 重定位 | 錯了會產出「連得起來但資源壞掉」的執行檔 |

**如果之後要做，建議第二條，而且只做一次**：`.syso` 產好就 commit 進
`cmd/psychicwar/`（Go 會自動連進 `GOOS=windows` 的建置），之後**建置端沒有任何新相依**，
只有換圖示的時候才需要那個工具。圖示素材已經有了：`tools/py.sh tools/appicon.py <輸出.png> [邊長]`
（macOS 的 `.icns` 就是用它組的），多一步 PNG → `.ico`。

沒有圖示不影響能不能跑，只影響檔案總管與工作列的長相。

## 4. `-H windowsgui` 與「看不見的錯誤」

`-H windowsgui` 讓 PE 的 subsystem 欄位從 3（主控台）變成 2（GUI），雙擊時不會多開一個
黑底主控台視窗。代價是 **`log.Fatal` 的訊息沒有地方可去**：缺原版、缺字型、狀態檔讀不動，
玩家看到的都是「雙擊之後什麼事都沒發生」。

`rulebook/82` 第 1 點講的就是這件事：同一段訊息在 Linux 是 stderr 上的噪音，
在 Windows 是玩家唯一的線索，而 GUI 子系統把那個線索整個拿掉。

### 4.1 做法：彈窗 ＋ 錯誤紀錄（issue #45，2026-09-19）

`apps/psychicwar/fatal.go` 的 `Fatal(code, msg)` 取代 `cmd/psychicwar` 裡每一處 `log.Fatal`，
三條出口同時走（規格 `docs/spec/021` §3.4）：

| 出口 | 平台 | 內容 |
|---|---|---|
| stderr | 全部 | 與以前相同 |
| `%APPDATA%\PsychicWar\psychicwar-error.log` | 全部（Linux 在 `~/.local/share/psychicwar`） | 時間、命令列、工作目錄、訊息。覆蓋寫 |
| `MessageBoxW` | 只有 Windows | 訊息 ＋ 紀錄檔路徑 |

分平台的只有彈窗：`dialog_windows.go` 與 `dialog_other.go`（no-op），build tag 分檔，
與 `cputime_*.go` 同一套做法。`MessageBoxW` 走 `golang.org/x/sys/windows`
（`x/sys` 本來就在相依樹裡，這一輪從 indirect 變成直接相依）：它用 `NewLazySystemDLL`
只從 system32 載，不會被執行檔旁邊的同名 DLL 攔截，而且沒有 cgo，`CGO_ENABLED=0` 照樣編得過。

兩條配套規則，少了任何一條都會讓玩家追錯方向：

- **缺資料檔的訊息要帶「怎麼修」。** `open …\cjk24.golemfnt: no such file or directory`
  每個字都看得懂，但沒有回答「我該做什麼」。現在的格式是「一句現況 ＋ 原始錯誤 ＋ 一句修法」。
- **啟動成功就刪掉上一次的紀錄檔**（`ClearErrorLog`）。不刪的話，玩家修好之後還是會
  看到那個檔案，照著已經不成立的訊息追下去。

`PSYCHICWAR_NO_DIALOG=1` 關掉彈窗，給無人看管的驗收用（§5.2）。

### 4.2 `troubleshoot.bat` 留著當後路

彈窗被擋掉（防毒、遠端桌面）、或要看 `log.Printf` 那些非致命訊息時還是用得上。
包裡附一支 ASCII 內容的 `.bat`：

```bat
PsychicWar.exe %* > psychicwar-log.txt 2>&1
echo Exit code: %ERRORLEVEL%
type psychicwar-log.txt
pause
```

**GUI 子系統的行程沒有自己的主控台，但照樣繼承 cmd 交給它的標準控制代碼**，
所以導向檔案收得到輸出。這是本輪實測的（§5.2 第 D 項），不是推論。

`.bat` 的內容全部用 ASCII：cmd.exe 以系統 ANSI 代碼頁逐行解讀批次檔，
中文寫進去在非 cp950 的機器上會變亂碼。中文說明另外放 `troubleshoot.txt`，
UTF-8 ＋ BOM（記事本沒有 BOM 會拿 ANSI 代碼頁去猜）。檔名一律 ASCII，
避開 zip 的檔名編碼問題。

### 4.3 順帶一提：Ebiten 本來就會藏主控台

`ebitengine/hideconsole`（Ebiten 的間接相依）在 `internal/ui` 的 `init()` 裡呼叫
`GetConsoleWindow` ＋ `GetWindowThreadProcessId`，**只有在「主控台是自己這支行程建的」
（＝雙擊啟動）時才 `FreeConsole`**。所以不加 `-H windowsgui` 也不會留一個黑框，
而且從 cmd 啟動時訊息還看得到。

兩者的差別其實很小：`-H windowsgui` 是雙擊時連一閃都沒有，hideconsole 是閃一下再消失。
規格與本輪定案採 `-H windowsgui`，驗收腳本把 subsystem 2 當硬條件，
旗標掉了會被擋下（§5.1 的反向對照驗過）。

## 5. 驗收：驗到了什麼

`rulebook/82` 的硬規則是「驗**實際打包產物**在**它自己的執行環境**」，所以下面每一項
都是**解開 zip 之後**的那份，不是 `workplace/bin/` 的建置輸出。

### 5.1 靜態（`tools/windows-verify.sh`，純 Python 解 PE）

| 道 | 檢查 | 結果 |
|---|---|---|
| 1 | PE 格式：`MZ` ＋ `PE\0\0` ＋ PE32+（`0x20B`） | 過 |
| 2 | 架構：COFF machine `0x8664` | 過 |
| 3 | 子系統 ＝ 2（GUI） | 過 |
| 4 | 匯入的 DLL | 只有 `kernel32.dll`；mingw／MSVC runtime 0 個 |
| 5 | 內容證據 | `PsychicWar` 1、`cjk24.golemfnt` 2、`銀河超能力戰記` 1、`original` 6；反向樣本 `SDL2` 0 |

第 4 道還另外確認執行期會載的 DLL 名字在不在 binary 裡（24 個系統 DLL 全部找得到，
含 `opengl32.dll`、`d3d11.dll`、`user32.dll`）。
⚠ **這裡不能用正規式去「列出所有 DLL 名字」**：Go 把字串常數黏成一整塊、沒有分隔符號，
撈出來的會是「前一個字串的結尾＋dll 名」這種東西（第一版就撈到 `foundwinmm.dll`、
`rocessmemoryinfobcryptprimitives.dll`）。**問「某個名字在不在」答得準，
問「總共有哪些名字」答不準**，所以改成拿已知名單逐個查。

反向對照（證明檢查不是空轉）：

| 餵給它 | 期望 | 實際 |
|---|---|---|
| 沒加 `-H windowsgui` 編出來的同一支程式 | 擋下 | 「subsystem 3 不是 2（GUI）：-H windowsgui 掉了」，結束碼 1 |
| 反向樣本字串 `SDL2` | 0 處 | 0 處 |

### 5.2 wine 實跑（`tools/windows-verify.sh <zip> --run`）

環境：`psychicwar-wine`（Ubuntu 24.04 ＋ wine-9.0 ＋ Xvfb），`--network none`，
`EBITENGINE_GRAPHICS_LIBRARY=opengl`（軟體 llvmpipe）。每一項都先解開 zip 再跑。

| 項 | 做什麼 | 期望 | 實際 |
|---|---|---|---|
| A | 可散布版 ＋ `-orig Z:\orig\psychic-war`，`-quit-after 12s` | 起得來 | **結束碼 0**，`%APPDATA%\PsychicWar` 被建出來（`C:\users\ubuntu\AppData\Roaming\PsychicWar`） |
| A2 | 同一個狀態檔，wine 的畫面 vs repo 建置的 Linux 執行檔 | 差 0 | `05-select`、`08-encounter` 兩張**逐像素差 0** |
| B | 可散布版，**不掛**原版也不給 `-orig` | 明確報錯 | **結束碼 2**，訊息是 §3.3 那段（「找不到原版遊戲檔案 PW.EXE」＋兩種指法，例子是 `PsychicWar.exe -orig D:\games\psychic-war`），`%APPDATA%\PsychicWar\psychicwar-error.log` 寫得出來 |
| C | `-with-data` 版，**不掛**原版也不給 `-orig` | 起得來 | **結束碼 0**，`%APPDATA%\PsychicWar` 被建出來，畫面是原版的 KOGADO 開場 |
| D | `wine cmd /c troubleshoot.bat`（不給原版） | 訊息進得了檔案 | `psychicwar-log.txt` **1,326 bytes**，內容是用法訊息 |
| E | 把 `font\` 改名再跑 | 明確報錯 | **結束碼 1**，`open font\cjk24.golemfnt: Path not found.` |
| E2 | E 的對照組（字型在） | 正常 | 結束碼 0 |
| F | 缺原版，**不設** `PSYCHICWAR_NO_DIALOG`（`--dialog`，2026-09-19） | 彈出 MessageBox | 程式**沒有自己結束**（停在模態視窗上）；視窗樹裡 `0x600005` 標題 `銀河超能力戰記 Psychic War`、437×296；截圖 `workplace/win-check-out/dialog.png` 裡看得到錯誤圖示、全文與 OK 鍵 |
| F2 | F 的反向對照：同一情境設 `PSYCHICWAR_NO_DIALOG=1` | 立刻結束 | 結束碼 2（＝上面 B 那一列） |
| G | A 那一輪跑完，`%APPDATA%\PsychicWar` 底下 | 沒有錯誤紀錄 | 目錄是空的（正常啟動不寫、也把舊的清掉） |

A2 是這一輪最強的證據：**Windows 產物畫出來的東西與已經驗過的 Linux 產物一模一樣**，
包含中文疊字（操作面板的「前進／轉向／向後轉／選單」、狀態欄的「地點／方向」）。
截圖見 `workplace/win-check-out/`（gitignore）。

`EBITENGINE_GRAPHICS_LIBRARY` 預設值（DirectX）也跑得起來（結束碼 0），
但**沒有辦法證明它真的走了 D3D11 而不是自己退回 OpenGL**，所以只當旁證。

### 5.3 wine 驗不到什麼

- **wine 不是 Windows。** 視窗行為、DPI、輸入法、音效卡、Direct3D 的實際路徑、
  檔案總管裡的圖示、防毒軟體的反應，全部未知。
- **沒有按過任何一個鍵。** A 與 C 只證明「開得起來、畫得出來、存檔目錄建得出來」，
  沒有走過玩家路徑。Linux 那邊有 `tools/frontend-playthrough.sh` 用 xdotool 實際打完第一場戰鬥，
  Windows 這邊沒有對應的東西。
- **`%APPDATA%` 只驗到目錄被建出來**，沒有實際存一次遊戲檔再讀回來。
- **彈窗只驗到 wine 的 MessageBox。** 真正的 Windows 上，彈窗會不會被防毒、遠端桌面或
  全螢幕的其他程式蓋住，沒有驗過。這也是 `troubleshoot.bat` 留著的理由（§4.2）。
- **彈窗的中文能不能顯示，這個環境答不了。** 這個 image 一個 CJK 字型都沒有，
  第一次截圖整段中文是豆腐格；把專案烘字用的 `NotoSansCJKtc-Regular.otf` 複製進
  prefix 的 `Fonts\` 並設 Wine 的字型 Replacements（`MS Shell Dlg`、`Tahoma`）之後才看得到字
  （`tools/windows-verify.sh --dialog` 已經內建這一步）。**豆腐格是驗收環境缺字型，
  不是訊息壞掉**，但這代表「Windows 的介面字型畫不畫得出這些字」在這裡驗不到。

### 5.4 踩到的兩個坑（都會給出「自洽但錯」的結論）

**一、Debian 12 的 wine 8.0 跑不動任何 Go 1.22 以後編的 `.exe`。**
第一版的驗收 image 以專案既有的 `psychicwar-go-ebiten`（Debian 12）為底，裝 Debian 的 wine 8.0，
結果是：

```
fatal error: bcryptprimitives.dll not found
runtime: panic before malloc heap initialized
runtime.loadOptionalSyscalls()
	runtime/os_windows.go:269
```

Go 的執行期在 `osinit` 就要 `bcryptprimitives.dll` 的 `ProcessPrng`，**真正的 Windows 8 以後都有**，
wine 8.0 沒有。症狀長得像「我們的 exe 壞了」，其實是驗收工具太舊。
bookworm-backports 也沒有新版 wine（查過）；Ubuntu 24.04 的 libwine 9.0 檔案清單裡有這個 DLL
（查 packages.ubuntu.com 的 filelist，沒有先下載 100 MB 的 `.deb` 才知道），所以底改成 Ubuntu 24.04。
**要在 wine 上驗 Go 程式，wine 至少要 9.0。**

**二、`import -window root` 截到的不是視窗。** wine 把視窗放在 `+160+66`，不是 `0,0`；
直接拿 root 的左上角 960×600 去比，會得到「47% 的像素不同」這種看起來很嚴重、
其實只是沒對齊的數字。第一次比出 272,286、換位移試出 232,377 的「最小值」，
差一點就寫成「Windows 版畫面不對」。
**要先問視窗在哪**（`xwininfo -root -tree` 的**最後一欄**才是絕對座標，
倒數第二欄的 `+x+y` 是相對父視窗的），再照那個座標裁。對齊之後差 0。

## 6. 重現

```sh
tools/package.sh windows                                   # 可散布版
PSYCHICWAR_WITH_DATA=1 tools/package.sh windows            # 再多一份含原版素材的本機版
tools/windows-verify.sh dist-all/PsychicWar-<版本>-win64.zip --run
PSYCHICWAR_NO_ORIG=1 PSYCHICWAR_WINE_ARGS=" " \
  tools/windows-verify.sh dist-all/PsychicWar-<版本>-with-data-win64.zip --run
# 缺原版的訊息與結束碼（§5.2 B、F2）
PSYCHICWAR_NO_ORIG=1 PSYCHICWAR_WINE_ARGS=" " \
  tools/windows-verify.sh dist-all/PsychicWar-<版本>-win64.zip --run
# 彈窗本身（§5.2 F）：不掛原版、不設 PSYCHICWAR_NO_DIALOG，截圖存 workplace/win-check-out/dialog.png
tools/windows-verify.sh dist-all/PsychicWar-<版本>-win64.zip --dialog
```

A2（跨平台逐像素比）沒有做成腳本，是臨時跑的，重點只有三步：

1. wine 那邊：Xvfb 開 1280×800，`PsychicWar.exe -load-state Z:\src\workplace\states\05-select.state
   -audio null -quit-after 9s`，6 秒時用 `xwininfo -root -tree` 找 `960x600` 那一行的**最後一欄**
   （絕對座標），`import -window root -crop 960x600+X+Y`。
2. Linux 那邊：同一個狀態檔、同一組旗標，`xdotool search --name 銀河超能力戰記` ＋
   `getwindowgeometry --shell` 取座標再裁。
3. `compare -metric AE a.png b.png null:`。

旋鈕：`PSYCHICWAR_WIN_CPUS`（預設 4）、`PSYCHICWAR_WIN_TIMEOUT`、`PSYCHICWAR_WINE_CPUS`、
`PSYCHICWAR_WINE_ARGS`、`PSYCHICWAR_NO_ORIG=1`（不掛原版目錄）、
`EBITENGINE_GRAPHICS_LIBRARY`（預設 `opengl`）。
第一次跑 `--run` 會 build `psychicwar-wine`（**那一步要網路**，之後一律 `--network none`）。

## 7. 踩到什麼（寫成規則）

- **「Ebiten 要不要 cgo」是逐平台的問題，不是一個答案。** macOS 要（Cocoa／Metal 後端），
  Windows 不要（purego 在執行期載 DLL）。拿 macOS 的結論去推 Windows 會白裝一套 mingw。
  先編一次看錯誤訊息，成本是幾十秒。
- **編不過的訊息要分清楚是誰的程式碼。** 這次四行全在 `cmd/psychicwar`，代表問題是
  「我們用了 POSIX 專屬 API」，換工具鏈解決不了。訊息裡的檔名就是分流的依據。
- **`syscall` 的平台差異要擋在 `cmd` 外面。** 一個 `getrusage` 就讓整個 Windows 版編不過。
  平台專屬的東西放進有 build tag 的檔案，`cmd` 只呼叫一個中性的函式。
- **`SaveDir()` 這種 switch 少一條分支不會報錯，只會靜默走錯。** Windows 掉進 Linux 那條
  仍然建得出目錄、寫得進去，症狀是「存檔不見了」。新增平台時要逐條看 switch，不能只看編不編得過。
- **不要用正規式從 Go binary 裡「撈出」字串清單。** 字串常數之間沒有分隔符號。
  撈清單會撈到黏在一起的東西；查單一名字在不在則是準的。
- **GUI 子系統把錯誤訊息整個拿掉。** 不是「訊息變得不明顯」，是**一個字都沒有**。
  包裡一定要附一條看得到訊息的路徑（`.bat` 重導向，或程式自己寫 log ／彈窗）。
- **拿 wine 當驗收環境，要先確認 wine 夠新。** Go 1.22 以後的 `.exe` 在 wine 8.0 上
  連 `main` 都進不去，而錯誤訊息指著我們的 binary。驗收工具的年紀會偽裝成產物的缺陷。
- **截圖比對之前先確定截到的是同一塊畫面。** 視窗不在 `0,0`。沒對齊的比對會產出
  一個很具體、很有說服力、而且完全錯的數字；「換個位移找最小值」只會讓它更像真的。
- **模態視窗與無人看管的驗收互斥。** `MessageBox` 會停在那裡等人按確定，自動化那邊
  看到的是「卡住」，不是結束碼。要有一個關掉彈窗的開關給驗收用，**而且彈窗本身要另外驗一次**，
  否則開關會變成「永遠沒驗過」的擋箭牌。
- **`xwininfo` 印不出非 ASCII 標題時，長相像「視窗不存在」。** 預設 locale 是
  `ANSI_X3.4-1968`，中文標題會被印成 `" (failure in conversion from UTF8_STRING to …)"`，
  grep 標題就落空。要嘛 `LC_ALL=C.UTF-8`，要嘛改用 `xprop` 讀 `_NET_WM_NAME`（UTF8_STRING 原樣印）。
- **容器裡的字型不是目標平台的字型。** 彈窗的中文在這個 image 裡全是豆腐格，
  因為 image 一個 CJK 字型都沒有——**是驗收環境的缺，不是產物的缺**，但兩者的截圖長得一樣。
  分辨方法：看 ASCII 的部分有沒有正常畫出來。
