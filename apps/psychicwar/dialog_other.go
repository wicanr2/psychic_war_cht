//go:build !windows

package psychicwar

// showFatalDialog 在 Windows 以外是 no-op。
//
// Linux 與 macOS 的執行檔都連在主控台子系統上，stderr 到得了使用者手上
// （終端機、.app 的話是 Console.app 的系統紀錄），不需要另外彈窗；
// 要彈還得挑 zenity／kdialog／osascript，多一堆執行期相依卻換不到新資訊。
// 這兩個平台靠 Fatal 的另外兩條出口（stderr ＋ 錯誤紀錄檔）。
func showFatalDialog(title, msg string) {}
