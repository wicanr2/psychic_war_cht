package psychicwar

// 致命錯誤要走得到玩家眼前（issue #45；rulebook/82 第 1 點）。
//
// Windows 版以 `-H windowsgui` 連結，行程沒有主控台。`log.Fatal` 的訊息不是「比較不明顯」，
// 是**一個字都不會出現**：雙擊的人只看到什麼都沒發生。同一段訊息在 Linux 只是終端機上的一行字。
// 所以致命錯誤一律走這裡，三條出口同時走：
//
//  1. stderr——從終端機或 troubleshoot.bat 啟動時看得到，行為與以前相同。
//  2. 存檔目錄下的紀錄檔——三個平台都寫，事後要人回報時有東西可以貼。
//  3. 平台專屬的彈窗——只有 Windows 有實作（showFatalDialog），其餘平台是 no-op。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrorLogName 是錯誤紀錄的檔名，放在存檔目錄（SaveDir，docs/spec/021 §3.2）底下。
// 不放執行檔旁邊：AppImage 與 .app 的內容是唯讀的，發行包也可能裝在 Program Files。
const ErrorLogName = "psychicwar-error.log"

// NoDialogEnv 設成 1 就不彈視窗。
//
// 自動驗收要的是結束碼，而模態視窗沒有人會按確定，會一路卡到 timeout。
// 彈窗本身另外跑一次不設這個變數的（docs/re/035 §4.4），不是靠這個變數繞過去不驗。
const NoDialogEnv = "PSYCHICWAR_NO_DIALOG"

// dialogTitle 與視窗標題一致，玩家看得出是同一支程式。
const dialogTitle = "銀河超能力戰記 Psychic War"

// errorLogDir 由 SetErrorLogDir 指定；空字串表示用 SaveDir。
var errorLogDir string

// SetErrorLogDir 指定錯誤紀錄寫到哪，通常是 `-scratch` 解析之後的存檔目錄。
// 給了 `-scratch` 卻把紀錄寫到別處，回報的人會找不到檔案。
func SetErrorLogDir(dir string) { errorLogDir = dir }

// ErrorLogPath 回錯誤紀錄的完整路徑；連存檔目錄都推不出來時回空字串。
func ErrorLogPath() string {
	dir := errorLogDir
	if dir == "" {
		dir = SaveDir("")
	}
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, ErrorLogName)
}

// Fatal 報告一個致命錯誤然後結束。code 通常是 1；用法問題（缺原版目錄）用 2。
func Fatal(code int, msg string) {
	fmt.Fprintln(os.Stderr, msg)
	path, err := writeErrorLog(ErrorLogPath(), msg)
	if err != nil {
		// 寫不出來不能蓋掉原本的錯誤——真正的問題是 msg，不是紀錄檔。
		fmt.Fprintf(os.Stderr, "（另外：錯誤紀錄寫不出來：%v）\n", err)
	}
	if os.Getenv(NoDialogEnv) != "1" {
		showFatalDialog(dialogTitle, dialogText(msg, path))
	}
	os.Exit(code)
}

// Fatalf 是 Fatal(1, fmt.Sprintf(...)) 的簡寫。
func Fatalf(format string, a ...any) { Fatal(1, fmt.Sprintf(format, a...)) }

// ClearErrorLog 在啟動成功之後刪掉上一次失敗留下的紀錄。
// 少了這一步，玩家修好問題之後還是會看到一個錯誤紀錄檔，照著已經不成立的訊息追下去。
func ClearErrorLog() {
	if p := ErrorLogPath(); p != "" {
		_ = os.Remove(p)
	}
}

// writeErrorLog 把訊息寫成紀錄檔（覆蓋，只留最後一次失敗），回寫到哪。
// path 是空字串時什麼都不做，不算失敗。
func writeErrorLog(path, msg string) (string, error) {
	if path == "" {
		return "", nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(&b, "命令列：%s\n", strings.Join(os.Args, " "))
	if wd, err := os.Getwd(); err == nil {
		fmt.Fprintf(&b, "工作目錄：%s\n", wd)
	}
	fmt.Fprintf(&b, "\n%s\n", msg)
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// dialogText 是彈窗裡的內容：錯誤本身，加上紀錄檔在哪。
func dialogText(msg, path string) string {
	if path == "" {
		return msg
	}
	return msg + "\n\n（同一份訊息也寫在 " + path + "）"
}
