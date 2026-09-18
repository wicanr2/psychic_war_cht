package psychicwar

// 防拷畫面的答案（docs/spec/013）：原版出題時把正確答案（超能力名稱）複製到 cs:66E5，
// F1 說明頁直接讀那 16 bytes，不散布原版資料。

import "strings"

// ProtAnswerAddr 是正確答案在段 0161 裡的偏移與長度（docs/spec/013 §1）。
const (
	ProtAnswerSeg = 0x0161
	ProtAnswerOff = 0x66E5
	ProtAnswerLen = 16
)

// ProtectionAnswer 把那 16 bytes 轉成答案字串；看起來不像答案就回空字串。
//
// 判準（docs/spec/013 §2.2）：全部是可印字元；去掉尾端空白後長度 ≥ 3，
// 而且只含大寫字母、數字、`-`（超能力名稱的字集）。
func ProtectionAnswer(b []byte) string {
	if len(b) < ProtAnswerLen {
		return ""
	}
	s := string(b[:ProtAnswerLen])
	for _, c := range []byte(s) {
		if c < 0x20 || c > 0x7E {
			return ""
		}
	}
	s = strings.TrimRight(s, " ")
	if len(s) < 3 {
		return ""
	}
	for _, c := range []byte(s) {
		switch {
		case c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-':
		default:
			return ""
		}
	}
	return s
}
