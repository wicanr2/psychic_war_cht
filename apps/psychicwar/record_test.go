package psychicwar

import (
	"path/filepath"
	"testing"
)

// docs/spec/019 §3、§5 第 3 項：往返與雜湊檢查。
func TestRecordingRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "r.json")
	r := NewRecording("abc123", 750, "states/07-first-play.state", 1000)
	r.Add(1200, "Up", true)
	r.Add(1500, "Up", false)
	if err := r.Save(p); err != nil {
		t.Fatal(err)
	}
	got, err := LoadRecording(p, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if got.Cycles != 750 || got.StartStep != 1000 || len(got.Events) != 2 {
		t.Fatalf("往返後不一樣：%+v", got)
	}
	if got.Events[0].Step != 1200 || !got.Events[0].Down || got.Events[1].Down {
		t.Errorf("事件不對：%+v", got.Events)
	}
	// 執行檔換了就拒絕：指令數對不上，硬播只會得到看起來很像但其實不同的結果
	if _, err := LoadRecording(p, "different"); err == nil {
		t.Error("雜湊不符應該拒絕")
	}
	// 不檢查雜湊時照樣讀得到
	if _, err := LoadRecording(p, ""); err != nil {
		t.Errorf("空字串應該略過檢查：%v", err)
	}
}
