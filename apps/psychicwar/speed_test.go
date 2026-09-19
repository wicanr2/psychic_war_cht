package psychicwar

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// docs/spec/023 §8 第 1 項。

func TestSpeedGearCycles(t *testing.T) {
	s := NewSpeed(1)
	// 從原速按一圈：2、3、…、最後回到原速。
	for _, g := range append(append([]int{}, SpeedGears[1:]...), SpeedGears[0]) {
		if got := s.Next(); got != g {
			t.Fatalf("循環到 %d，要 %d", got, g)
		}
	}
}

func TestSpeedBattleForcesOriginal(t *testing.T) {
	s := NewSpeed(3)
	if s.Effective() != 3 {
		t.Fatalf("非戰鬥時 Effective()=%d，要 3", s.Effective())
	}
	s.SetBattle(true)
	if s.Effective() != 1 {
		t.Fatalf("戰鬥中 Effective()=%d，要 1", s.Effective())
	}
	if s.Gear() != 3 {
		t.Fatalf("戰鬥中玩家選的檔位不該被改掉：%d", s.Gear())
	}
	// 戰鬥中換檔只記下來，還是不生效。
	s.Next()
	if s.Effective() != 1 {
		t.Fatalf("戰鬥中換檔後 Effective()=%d，要 1", s.Effective())
	}
	gear := s.Gear()
	s.SetBattle(false)
	if s.Effective() != gear {
		t.Fatalf("離開戰鬥後 Effective()=%d，要 %d", s.Effective(), gear)
	}
}

func TestSpeedLabels(t *testing.T) {
	s := NewSpeed(1)
	if s.Badge() != "" {
		t.Errorf("原速不該畫常駐標記：%q", s.Badge())
	}
	s = NewSpeed(2)
	if s.Badge() != "2 倍" || s.Toast() != "速度 2 倍" {
		t.Errorf("2 倍：標記 %q、提示 %q", s.Badge(), s.Toast())
	}
	s.SetBattle(true)
	if s.Badge() != "2 倍→原速" || s.Toast() != "速度 2 倍（戰鬥維持原速）" {
		t.Errorf("戰鬥中 2 倍：標記 %q、提示 %q", s.Badge(), s.Toast())
	}
}

// 不在清單裡的倍率（壞設定檔、打錯的旗標）要退回原速，不是照用。
func TestSpeedUnknownGearFallsBack(t *testing.T) {
	if g := NewSpeed(7).Gear(); g != 1 {
		t.Errorf("未知檔位 7 → %d，要 1", g)
	}
}

// docs/spec/023 §5：一幀能寫多少取樣由牆上時間決定，多的平均抽掉。
func TestAudioBudget(t *testing.T) {
	const rate = 1000
	// 原速：一幀 100 ms 產生 100 個取樣，剛好用完 credit，整批照寫。
	var b AudioBudget
	pcm := make([]int16, 100)
	for i := range pcm {
		pcm[i] = int16(i)
	}
	if got := b.Take(pcm, 0.1, rate); len(got) != 100 {
		t.Errorf("原速一幀寫 %d 個，要 100", len(got))
	}
	// 3 倍：同樣 100 ms 牆上時間，機器產生 300 個取樣 → 只能寫 100 個，平均抽。
	b = AudioBudget{}
	long := make([]int16, 300)
	for i := range long {
		long[i] = int16(i)
	}
	got := b.Take(long, 0.1, rate)
	if len(got) != 100 || got[0] != 0 || got[1] != 3 || got[99] != 297 {
		t.Errorf("3 倍抽樣 %d 個、前兩筆 %d %d、最後 %d，要 100 個、0 3、297",
			len(got), got[0], got[1], got[99])
	}
	// 主機跟不上（機器只跑到 1.5 倍）：抽掉的比例要跟著變小，不是照檔位抽。
	b = AudioBudget{}
	mid := make([]int16, 150)
	if got := b.Take(mid, 0.1, rate); len(got) != 100 {
		t.Errorf("1.5 倍實際寫 %d 個，要 100（照檔位 3 抽只會剩 50）", len(got))
	}
	// 抖動：上一幀沒寫滿的額度留著，下一幀補得回來，不然原速也會持續欠載。
	b = AudioBudget{}
	b.Take(make([]int16, 50), 0.1, rate) // 只有 50 個，剩 50 額度
	if got := b.Take(make([]int16, 150), 0.1, rate); len(got) != 150 {
		t.Errorf("補回來時寫 %d 個，要 150", len(got))
	}
}

// docs/spec/023 §3：戰鬥狀態由掛鉤翻轉，讀檔後由敵人 HP 重推。
func TestBattleHooks(t *testing.T) {
	var b Battle
	if b.In() {
		t.Fatal("一開始不該在戰鬥中")
	}
	b.Enter()
	if !b.In() || !b.TakeChange() {
		t.Fatal("進戰鬥要翻轉而且要報告有變")
	}
	if b.TakeChange() {
		t.Error("問過一次就要清掉")
	}
	b.Enter() // 重複進入不算變化
	if b.TakeChange() {
		t.Error("同一個狀態再設一次不該報告有變")
	}
	b.Leave()
	if b.In() || !b.TakeChange() {
		t.Fatal("離開戰鬥要翻轉而且要報告有變")
	}
	// 讀檔：18 個非戰鬥檢查點的敵人 HP 都是 0，戰鬥中的是 38（docs/re/037 §2.1）
	b.Reseed(0)
	if b.In() {
		t.Error("敵人 HP 0 → 不在戰鬥中")
	}
	b.Reseed(38)
	if !b.In() {
		t.Error("敵人 HP 38 → 在戰鬥中")
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if got := LoadSettings(dir).Speed; got != 1 {
		t.Errorf("沒有設定檔時 %d，要 1", got)
	}
	if err := SaveSettings(dir, Settings{Speed: 3}); err != nil {
		t.Fatal(err)
	}
	if got := LoadSettings(dir).Speed; got != 3 {
		t.Errorf("存 3 讀回 %d", got)
	}
	// 壞掉的設定檔回預設值，不是錯誤：設定檔只影響節奏，不該擋住遊戲。
	for _, bad := range []string{"{", `{"schema":"別的","speed":3}`, `{"schema":"psychic-war-settings/1","speed":99}`} {
		if err := os.WriteFile(filepath.Join(dir, SettingsFile), []byte(bad), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := LoadSettings(dir).Speed; got != 1 {
			t.Errorf("壞設定檔 %q → %d，要 1", bad, got)
		}
	}
}

// docs/spec/023 §4：F12 要被攔，而且**不可以有掃描碼**。
// 兩句話缺一不可——攔了卻留著掃描碼是矛盾，沒攔卻沒有掃描碼是「按了沒反應」（docs/re/036 §3）。
func TestSpeedHotkey(t *testing.T) {
	if !Intercepted(ebiten.KeyF12) {
		t.Error("F12 是速度熱鍵，要攔下來")
	}
	if _, ok := scanCodes[ebiten.KeyF12]; ok {
		t.Error("F12 不可以有掃描碼：原版收不到這個鍵，攔它才不會拿掉原版功能")
	}
	if _, ok := ScanCode(ebiten.KeyF12); ok {
		t.Error("被攔下的鍵不該送進遊戲")
	}
}
