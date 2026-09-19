package psychicwar

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// docs/spec/006 §4 第 1 項。

func TestPacer(t *testing.T) {
	p := &Pacer{PerMs: 750}
	if got := p.Cycles(1000, 999); got != 750 {
		t.Errorf("落後 1 ms 給 %d cycles，要 750", got)
	}
	if got := p.Cycles(1000, 1000); got != 0 {
		t.Errorf("沒落後卻給 %d", got)
	}
	p = &Pacer{PerMs: 750}
	if got := p.Cycles(1000, 0); got != 75_000 || p.DroppedMs != 900 {
		t.Errorf("落後 1000 ms：給 %d、放棄 %.0f ms，要 75000、900", got, p.DroppedMs)
	}
	// 放棄的量之後不再追：機器跑到 100 ms 後、牆上 1016 ms，只差 16 ms。
	if got := p.Cycles(1016, 100); got != 12_000 {
		t.Errorf("放棄追趕後給 %d，要 12000", got)
	}
}

func TestRingUnderrun(t *testing.T) {
	q := NewRing(4096)
	in := make([]int16, 1000)
	for i := range in {
		in[i] = int16(i + 1)
	}
	q.Write(in)
	out := make([]int16, 1500)
	if k := q.Read(out); k != 1000 || out[999] != 1000 || out[1000] != 0 || out[1499] != 0 || q.Underruns != 1 {
		t.Errorf("讀 1500：真實 %d、第 1000 筆 %d、欠載 %d", k, out[1000], q.Underruns)
	}
	// 反向對照：夠讀時不算欠載。
	q = NewRing(4096)
	q.Write(in)
	if k := q.Read(make([]int16, 1000)); k != 1000 || q.Underruns != 0 {
		t.Errorf("讀 1000：真實 %d、欠載 %d", k, q.Underruns)
	}
}

func TestRingOverflowDropsOldest(t *testing.T) {
	q := NewRing(3)
	q.Write([]int16{1, 2, 3, 4})
	out := make([]int16, 3)
	q.Read(out)
	if out[0] != 2 || out[2] != 4 || q.Overflows != 1 {
		t.Errorf("溢位後讀到 %v，溢位 %d", out, q.Overflows)
	}
}

func TestKeyMap(t *testing.T) {
	for k, want := range map[ebiten.Key]uint8{
		ebiten.KeyArrowUp: 0x48, ebiten.KeyArrowDown: 0x50, ebiten.KeyArrowLeft: 0x4B, ebiten.KeyArrowRight: 0x4D,
		ebiten.KeyNumpad8: 0x48, ebiten.KeyNumpad2: 0x50, ebiten.KeySpace: 0x39, ebiten.KeyEnter: 0x1C, ebiten.KeyEscape: 0x01,
		ebiten.KeyA: 0x1E, ebiten.KeyZ: 0x2C, ebiten.KeyK: 0x25, ebiten.KeyI: 0x17, ebiten.KeyT: 0x14, ebiten.KeyDigit0: 0x0B, ebiten.KeyDigit9: 0x0A,
	} {
		if got, ok := ScanCode(k); !ok || got != want {
			t.Errorf("%v → %02X（%v），要 %02X", k, got, ok, want)
		}
	}
	// 輔助熱鍵：前端自己處理，不送進遊戲（docs/spec/012 §1）
	for _, k := range []ebiten.Key{ebiten.KeyF4, ebiten.KeyF5, ebiten.KeyF6,
		ebiten.KeyF7, ebiten.KeyF8, ebiten.KeyF10, ebiten.KeyF11} {
		if _, ok := ScanCode(k); ok || !Intercepted(k) {
			t.Errorf("%v 是輔助熱鍵，應該被攔下", k)
		}
	}
	// F1／F2／F3 是原版自己的功能鍵（issue #42）：不攔，而且要有掃描碼。
	// 少了掃描碼的話「不攔」只是空話——按了一樣送不進遊戲，症狀是沒反應不是報錯。
	for k, want := range map[ebiten.Key]uint8{
		ebiten.KeyF1: 0x3B, ebiten.KeyF2: 0x3C, ebiten.KeyF3: 0x3D, ebiten.KeyF9: 0x43,
	} {
		if Intercepted(k) {
			t.Errorf("%v 是原版的功能鍵，不可以攔", k)
		}
		if got, ok := ScanCode(k); !ok || got != want {
			t.Errorf("%v → %02X（%v），要 %02X", k, got, ok, want)
		}
	}
}

// docs/spec/012 §5 第 1 項：說明頁排版與即時存檔的版本比對。
func TestHelpFits(t *testing.T) {
	lines, err := LoadHelp("../../text")
	if err != nil {
		t.Fatalf("讀不到說明頁：%v", err)
	}
	if err := CheckHelp(lines); err != nil {
		t.Error(err)
	}
	if err := CheckHelp([]string{string([]rune(repeatRune('字', HelpCols+1)))}); err == nil {
		t.Error("超過行寬要回錯")
	}
	rows := make([]string, HelpRows+1)
	if err := CheckHelp(rows); err == nil {
		t.Error("超過行數要回錯")
	}
}

func repeatRune(r rune, n int) []rune {
	out := make([]rune, n)
	for i := range out {
		out[i] = r
	}
	return out
}

func TestCheckQuickMeta(t *testing.T) {
	m := NewQuickMeta("dosgolem-state/1", "exe-sha", "text-sha", "zh")
	if ok, why := CheckQuickMeta(m, "dosgolem-state/1", "exe-sha", "text-sha"); !ok || why != "" {
		t.Errorf("同一份應該可讀：%v %q", ok, why)
	}
	if ok, why := CheckQuickMeta(m, "dosgolem-state/2", "exe-sha", "text-sha"); ok || why == "" {
		t.Error("golem 版本不符要拒絕")
	}
	if ok, why := CheckQuickMeta(m, "dosgolem-state/1", "別的 exe", "text-sha"); ok || why == "" {
		t.Error("PW.EXE 不同要拒絕")
	}
	if ok, why := CheckQuickMeta(m, "dosgolem-state/1", "exe-sha", "別的譯文"); !ok || why == "" {
		t.Error("譯文不同只警告，仍可讀")
	}
	m.Schema = "psychic-war-quicksave/0"
	if ok, _ := CheckQuickMeta(m, "dosgolem-state/1", "exe-sha", "text-sha"); ok {
		t.Error("schema 不符要拒絕")
	}
}

func TestDrawTextPageFillsAndDraws(t *testing.T) {
	f := &Font8x8Stub
	w, h := 64, 32
	dst := make([]uint8, 4*w*h)
	DrawTextPage(dst, w, h, f, []string{"A"}, 8, [3]uint8{9, 9, 9}, [3]uint8{1, 2, 3}, 0xFF)
	if dst[0] != 1 || dst[1] != 2 || dst[2] != 3 || dst[3] != 0xFF {
		t.Errorf("背景沒填：%v", dst[:4])
	}
	drew := false
	for i := 0; i < w*h; i++ {
		if dst[4*i] == 9 && dst[4*i+1] == 9 && dst[4*i+2] == 9 {
			drew = true
			break
		}
	}
	if !drew {
		t.Error("字模沒畫上去")
	}
}

// docs/spec/013 §3 第 1 項：防拷答案的判讀。
func TestProtectionAnswer(t *testing.T) {
	if got := ProtectionAnswer([]byte("FREEZE          ")); got != "FREEZE" {
		t.Errorf("正常答案：%q", got)
	}
	if got := ProtectionAnswer([]byte("MIND-GRENADE    ")); got != "MIND-GRENADE" {
		t.Errorf("含連字號：%q", got)
	}
	for _, bad := range []string{
		"                ",  // 空白：一般畫面
		"Freeze          ",  // 小寫：不是答案的字集
		"FR              ",  // 太短
		"FREEZE\x00       ", // 控制碼
		"FREEZE!         ",  // 標點
	} {
		if got := ProtectionAnswer([]byte(bad)); got != "" {
			t.Errorf("%q 不該當成答案，卻回 %q", bad, got)
		}
	}
	if got := ProtectionAnswer([]byte("SHORT")); got != "" {
		t.Errorf("長度不足 16 要回空字串，卻回 %q", got)
	}
}

// docs/spec/014 §4 第 1 項：作弊的純函式。
func TestCheatValues(t *testing.T) {
	if got := FullValue(10, 40); got != 40 {
		t.Errorf("補到上限：%d", got)
	}
	if got := FullValue(40, 40); got != 0 {
		t.Errorf("already full 不用寫：%d", got)
	}
	if got := FullValue(10, 0); got != 0 {
		t.Errorf("上限 0（還沒開始遊戲）不要寫：%d", got)
	}
	if got := FullValue(10, 60000); got != 0 {
		t.Errorf("上限不合理時不要寫：%d", got)
	}
	for _, v := range []uint16{0, 1000, 65535} {
		if EnemyHPSane(v) {
			t.Errorf("%d 不該當成戰鬥中", v)
		}
	}
	for _, v := range []uint16{1, 38, 120, 999} {
		if !EnemyHPSane(v) {
			t.Errorf("%d 應該算戰鬥中", v)
		}
	}
}

// docs/spec/015 §4 第 1 項：自動地圖只記合理的格子、去重、分區域、快照往返。
func TestAutoMap(t *testing.T) {
	m := NewAutoMap()
	if !m.Note(0, 3, 4) || m.Note(0, 3, 4) {
		t.Error("同一格只算一次")
	}
	if m.Note(0, 64, 4) || m.Note(0, 3, 999) || m.Note(12, 1, 1) {
		t.Error("不合理的值不該記")
	}
	m.Note(1, 5, 6)
	if m.Count(0) != 1 || m.Count(1) != 1 || m.Seen(1, 3, 4) {
		t.Errorf("區域要分開：0 有 %d、1 有 %d", m.Count(0), m.Count(1))
	}
	b, err := m.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	m2 := NewAutoMap()
	if err := m2.UnmarshalJSON(b); err != nil {
		t.Fatal(err)
	}
	if !m2.Seen(0, 3, 4) || !m2.Seen(1, 5, 6) || m2.Count(0) != 1 {
		t.Error("快照往返後內容要相同")
	}
	m3 := NewAutoMap()
	if err := m3.UnmarshalJSON([]byte(`{"schema":"別的東西"}`)); err != nil || m3.Count(0) != 0 {
		t.Error("格式不對時當空的，不回錯")
	}
	for f, want := range map[uint16]rune{0: '↑', 1: '→', 2: '↓', 3: '←', 7: '←'} {
		if got := FacingMark(f); got != want {
			t.Errorf("朝向 %d → %c，要 %c", f, got, want)
		}
	}
}
