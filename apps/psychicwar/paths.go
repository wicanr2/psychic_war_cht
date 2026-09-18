package psychicwar

// 發行包的路徑解析（docs/spec/021 §3）。
//
// 從 repo 根目錄跑的時候 cwd 就是資料所在，旗標的相對路徑剛好對；發行包不是——
// 玩家會從任何目錄啟動，AppImage 與 macOS 的 .app 內容還是唯讀的，
// 存檔寫回 cwd 會失敗或落在莫名其妙的地方（rulebook/82 第 2、4 點）。

import (
	"os"
	"path/filepath"
	"runtime"
)

// DataDir 找資料目錄（name 是 "text" 或 "font"）。
//
// 依序找執行檔旁、macOS .app 的 Contents/Resources、cwd，回第一個存在的；
// 都沒有就回 name 本身，讓呼叫端照現行行為報錯——這裡不做靜默回退。
func DataDir(name string) string {
	for _, d := range dataCandidates(name) {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			return d
		}
	}
	return name
}

func dataCandidates(name string) []string {
	var out []string
	if exe, err := os.Executable(); err == nil {
		if exe, err := filepath.EvalSymlinks(exe); err == nil {
			dir := filepath.Dir(exe)
			out = append(out, filepath.Join(dir, name))
			// macOS 的 .app：執行檔在 Contents/MacOS，資料在 Contents/Resources
			out = append(out, filepath.Join(dir, "..", "Resources", name))
		}
	}
	return append(out, name)
}

// SaveDir 回存檔要寫到哪（docs/spec/021 §3.2）。取不到家目錄時回 fallback。
func SaveDir(fallback string) string {
	switch runtime.GOOS {
	case "darwin":
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, "Library", "Application Support", "PsychicWar")
		}
	default:
		if d := os.Getenv("XDG_DATA_HOME"); d != "" {
			return filepath.Join(d, "psychicwar")
		}
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, ".local", "share", "psychicwar")
		}
	}
	return fallback
}

// OrigDir 找原版目錄：執行檔旁的 original/。沒有就回空字串，由呼叫端印用法。
func OrigDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return ""
	}
	for _, d := range []string{
		filepath.Join(filepath.Dir(exe), "original"),
		filepath.Join(filepath.Dir(exe), "..", "Resources", "original"),
	} {
		if st, err := os.Stat(filepath.Join(d, "PW.EXE")); err == nil && !st.IsDir() {
			return d
		}
	}
	return ""
}
