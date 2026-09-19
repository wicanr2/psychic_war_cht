//go:build windows

package psychicwar

import "golang.org/x/sys/windows"

// showFatalDialog 在 Windows 彈一個錯誤視窗。
//
// 為什麼非做不可：`-H windowsgui` 的行程沒有主控台，stderr 沒有地方可去，
// 雙擊的人看不到任何訊息（docs/re/035 §4）。這是這支程式在 Windows 上唯一
// 「使用者一定看得到」的輸出管道。
//
// 用 x/sys/windows 的 MessageBox 而不是自己 syscall.NewLazyDLL("user32.dll")：
// 它走 NewLazySystemDLL，只從 system32 載，不會被執行檔旁邊的同名 DLL 攔截。
// 這條路徑沒有 cgo（purego 以外的部分都是 syscall），CGO_ENABLED=0 照樣編得過。
//
// 視窗是模態的：函式會停在這裡等使用者按確定，之後呼叫端才 os.Exit。
// 無人看管的自動驗收要設 NoDialogEnv，否則會卡到 timeout。
func showFatalDialog(title, msg string) {
	t, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	m, err := windows.UTF16PtrFromString(msg)
	if err != nil {
		return
	}
	// MB_SETFOREGROUND ＋ MB_TOPMOST：遊戲視窗還沒開，沒有父視窗可以掛，
	// 不搶到前景的話視窗會被壓在別的程式底下，症狀又變回「雙擊沒反應」。
	_, _ = windows.MessageBox(0, m, t,
		windows.MB_OK|windows.MB_ICONERROR|windows.MB_SETFOREGROUND|windows.MB_TOPMOST)
}
