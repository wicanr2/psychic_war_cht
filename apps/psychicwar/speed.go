package psychicwar

// 速度檔位（docs/spec/023）：只有非戰鬥的部分變快，戰鬥一律原速。
//
// 檔位是原速的整數倍，而且**只改「一毫秒牆上時間要跑幾個 cycle」，不改機器本身**。
// 理由在 docs/spec/023 §2：`SetDOSBoxCycles` 會連帶改 IRQ0 的間隔，而計時器中斷插在
// 戰鬥迴圈的哪個位置會改變單場戰鬥的結果（docs/re/010 §4.5）。只動節拍的話，
// 機器的執行與檔位完全無關，「戰鬥長度不受檔位影響」就是結構上的保證。

import "fmt"

// SpeedGears 是可選的倍率，第一個一定是原速。
// 不做非整數倍：音訊抽樣要整數，而且檔位越多玩家越難知道自己在哪一檔。
var SpeedGears = []int{1, 2, 3}

// Speed 記玩家選的檔位與現在在不在戰鬥。
type Speed struct {
	idx    int  // SpeedGears 的索引
	battle bool // 戰鬥中
}

// NewSpeed 建一個用 gear 倍率的檔位狀態；gear 不在清單裡就用原速。
func NewSpeed(gear int) *Speed {
	s := &Speed{}
	for i, g := range SpeedGears {
		if g == gear {
			s.idx = i
		}
	}
	return s
}

// Gear 回玩家選的倍率（不管現在是不是戰鬥中）。
func (s *Speed) Gear() int { return SpeedGears[s.idx] }

// Effective 回現在真正要套用的倍率：**戰鬥中一律 1**。
func (s *Speed) Effective() int {
	if s.battle {
		return 1
	}
	return SpeedGears[s.idx]
}

// SetBattle 設定在不在戰鬥中。
func (s *Speed) SetBattle(in bool) { s.battle = in }

// InBattle 回在不在戰鬥中。
func (s *Speed) InBattle() bool { return s.battle }

// Next 循環到下一檔，回新的倍率。戰鬥中照樣記下來，只是還不生效。
func (s *Speed) Next() int {
	s.idx = (s.idx + 1) % len(SpeedGears)
	return s.Gear()
}

// 畫面上的字樣（LoadHelp 從 text/help.json 讀進來覆蓋，docs/spec/023 §7）。
// 中文一律放在 text/ 的資料檔，不寫死在這裡——烘字型只掃 text/，
// 寫死在程式裡的字會靜默缺字（CLAUDE.md 待決事項、docs/re/030 §2）。
var (
	SpeedOriginal       = "原速"
	SpeedGearFmt        = "%d 倍"
	SpeedToastFmt       = "速度 %s"
	SpeedToastBattleFmt = "速度 %s（戰鬥維持原速）"
	SpeedBadgeBattleFmt = "%s→原速"
)

// GearName 回檔位的中文名。
func GearName(gear int) string {
	if gear <= 1 {
		return SpeedOriginal
	}
	return fmt.Sprintf(SpeedGearFmt, gear)
}

// Toast 回切換時要顯示的那一行。戰鬥中要說出「還沒生效」——
// 玩家按了鍵、畫面上的檔位變了、遊戲卻沒變快，不說就只能當成壞掉。
func (s *Speed) Toast() string {
	if s.battle && s.Gear() != 1 {
		return fmt.Sprintf(SpeedToastBattleFmt, GearName(s.Gear()))
	}
	return fmt.Sprintf(SpeedToastFmt, GearName(s.Gear()))
}

// Badge 回常駐在畫面角落的字；原速回空字串（不畫，不佔畫面）。
func (s *Speed) Badge() string {
	if s.Gear() == 1 {
		return ""
	}
	if s.battle {
		return fmt.Sprintf(SpeedBadgeBattleFmt, GearName(s.Gear()))
	}
	return GearName(s.Gear())
}

// AudioBudget 決定一幀能寫多少取樣進環形緩衝（docs/spec/023 §5）。
//
// 檔位 n 時機器一秒牆上時間產生 n 秒份的取樣，全寫進去會一直溢位
// （聽起來是延遲半秒再一直斷），所以多的要抽掉。
//
// **抽多少看牆上時間，不看檔位。** 兩者在主機跟得上的時候一樣，跟不上的時候不一樣：
// 前端追不上就會放棄一部分虛擬時間（Pacer.DroppedMs），機器其實沒跑到 n 倍，
// 照 n 抽會抽掉本來聽得到的聲音，結果是快轉時一直破音。
type AudioBudget struct{ credit float64 }

// Take 回這一幀要寫進環形緩衝的取樣；wallSec 是這一幀真正經過的牆上秒數。
//
// credit 是「牆上時間允許但還沒用掉的取樣數」。不逐幀清零是因為一幀的長短本來就有抖動，
// 清零會讓每一幀都被削掉尖峰、補不回低谷，累積成持續欠載——那在原速下也會發生。
func (b *AudioBudget) Take(pcm []int16, wallSec float64, rate int) []int16 {
	b.credit += wallSec * float64(rate)
	if cap := float64(rate) / 2; b.credit > cap { // 最多累到半秒，與環形緩衝同尺寸
		b.credit = cap
	}
	want := int(b.credit)
	if want >= len(pcm) {
		b.credit -= float64(len(pcm))
		return pcm
	}
	out := make([]int16, want)
	for i := range out {
		out[i] = pcm[i*len(pcm)/want]
	}
	b.credit -= float64(want)
	return out
}
