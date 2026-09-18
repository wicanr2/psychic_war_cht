package psychicwar

// 作弊（docs/spec/014）：一律寫記憶體，不改原版檔案。位址是這一支 PW.EXE 專屬的（docs/re/011）。

// 腳本變數的線性位址。
const (
	AddrHPMax     = 0x1698E
	AddrHP        = 0x16990
	AddrEnergyMax = 0x16992
	AddrEnergy    = 0x16994
	AddrEnemyHP   = 0x509C
)

// EnemyHPSane 回敵人 HP 這個值看起來是不是「正在戰鬥」。
// 不在戰鬥時那個字組是上一場留下來的（可能是 0）或還沒初始化的垃圾。
func EnemyHPSane(v uint16) bool { return v >= 1 && v <= 999 }

// FullValue 回「補到上限」之後該寫的值：上限是 0（讀不到或還沒開始遊戲）時回 0 表示不要寫。
func FullValue(cur, max uint16) uint16 {
	if max == 0 || max > 9999 {
		return 0
	}
	if cur >= max {
		return 0 // 已經滿了，不用寫
	}
	return max
}
