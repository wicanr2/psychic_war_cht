package psychicwar

// 戰鬥偵測（docs/spec/023 §3，證據在 docs/re/037）。
//
// **用執行位址判斷，不用記憶體值判斷。** 兩個掛鉤剛好把戰鬥主迴圈 sub_14CE4 包起來，
// 而戰鬥的難度就是那個迴圈（docs/re/010 §3）。打贏與用 F3 逃走兩種出口都會走到返回點。
//
// 為什麼不用敵人 HP `0x509C`：**F3 逃走時它不會被清掉**，會永遠停在上一場的值
// （docs/re/037 §2.1）。下一場如果遇到同種敵人，值一樣沒變，輪詢分不出來，
// 那一場就會跑在加速檔位上——那是會改變難度的方向。

// 執行期位址。段值 0161 是這一版 PW.EXE 在 dosgolem 上 EXEPACK 解壓後的程式段，
// 與 apps/psychicwar/translator 的印字掛鉤同一個段。
const (
	CodeSeg = 0x0161

	// OffBattleLoop 是戰鬥主迴圈 sub_14CE4 的進入點（IDA 14CE4）。
	// 位元組碼直譯器有兩個處理常式都會佈置敵人，只有這一個真的打（docs/re/037 §3）。
	OffBattleLoop = 0x47D4
	// OffBattleDone 是 `call sub_14CE4` 的返回點（IDA 14CBD）。
	OffBattleDone = 0x47AD
)

// Battle 記現在在不在戰鬥中。
type Battle struct {
	in      bool
	changed bool
}

// Enter 由 OffBattleLoop 的掛鉤呼叫。
func (b *Battle) Enter() { b.set(true) }

// Leave 由 OffBattleDone 的掛鉤呼叫。
func (b *Battle) Leave() { b.set(false) }

func (b *Battle) set(in bool) {
	if b.in != in {
		b.in, b.changed = in, true
	}
}

// In 回在不在戰鬥中。
func (b *Battle) In() bool { return b.in }

// TakeChange 回「上次問過之後狀態有沒有變」，問完就清掉。
func (b *Battle) TakeChange() bool {
	c := b.changed
	b.changed = false
	return c
}

// Reseed 在讀狀態檔之後由記憶體重推（docs/spec/023 §3）。
//
// 掛鉤在載入的時候不會觸發——狀態檔還原的是記憶體與暫存器，不是「剛剛執行過哪些位址」。
// 這裡用敵人 HP 當替代訊號：18 個非戰鬥檢查點都是 0，戰鬥中的那個是 38（docs/re/037 §2.1）。
// 已知誤判：在「F3 逃走之後」存的檔會被當成戰鬥中，一路維持原速到下一場戰鬥結束為止。
// 誤判方向是慢，不會讓戰鬥變快，所以不另外加「位置變了就解除」的條件——
// 那會讓判斷多一個來源，換到的只是「該快的時候早一點快」。
func (b *Battle) Reseed(enemyHP uint16) { b.set(EnemyHPSane(enemyHP)) }
