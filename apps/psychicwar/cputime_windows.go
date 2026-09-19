//go:build windows

package psychicwar

import "syscall"

// CPUMillis 回本行程用掉的 CPU 時間（毫秒，使用者態＋核心態），給 `-stats` 的 `cpu_ms` 用。
//
// Windows 沒有 `getrusage`，對應的是 `GetProcessTimes`：四個 FILETIME 裡的
// kernelTime 與 userTime 就是這支行程累計的 CPU 時間（100 奈秒為單位，
// `Filetime.Nanoseconds()` 已經換算好）。
func CPUMillis() int64 {
	h, err := syscall.GetCurrentProcess()
	if err != nil {
		return 0
	}
	var creation, exit, kernel, user syscall.Filetime
	if err := syscall.GetProcessTimes(h, &creation, &exit, &kernel, &user); err != nil {
		return 0
	}
	return (kernel.Nanoseconds() + user.Nanoseconds()) / 1e6
}
