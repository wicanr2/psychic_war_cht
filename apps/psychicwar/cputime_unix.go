//go:build !windows

package psychicwar

import "syscall"

// CPUMillis 回本行程用掉的 CPU 時間（毫秒，使用者態＋核心態），給 `-stats` 的 `cpu_ms` 用。
//
// 為什麼要分平台：`syscall.Getrusage` 與 `RUSAGE_SELF` 在 GOOS=windows 底下不存在
// （有沒有 cgo 都一樣），寫在 cmd 裡會讓整個 Windows 版編不過。
func CPUMillis() int64 {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0
	}
	return (ru.Utime.Nano() + ru.Stime.Nano()) / 1e6
}
