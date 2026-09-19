package psychicwar

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// issue #45：致命錯誤要留下一份紀錄檔，內容要夠一個玩家回報用（命令列 ＋ 訊息本體）。
func TestWriteErrorLog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", ErrorLogName) // 父層不存在，要自己建
	got, err := writeErrorLog(path, "找不到原版遊戲檔案（PW.EXE）。")
	if err != nil {
		t.Fatalf("writeErrorLog：%v", err)
	}
	if got != path {
		t.Fatalf("回傳 %q，期望 %q", got, path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("讀不回來：%v", err)
	}
	for _, want := range []string{"PW.EXE", "命令列：", os.Args[0]} {
		if !strings.Contains(string(b), want) {
			t.Errorf("紀錄檔裡找不到 %q，內容：\n%s", want, b)
		}
	}

	// 覆蓋而不是附加：只留最後一次失敗，檔案不會越長越大。
	if _, err := writeErrorLog(path, "第二次"); err != nil {
		t.Fatalf("第二次 writeErrorLog：%v", err)
	}
	b, _ = os.ReadFile(path)
	if strings.Contains(string(b), "PW.EXE") {
		t.Errorf("第二次寫入沒有覆蓋掉第一次：\n%s", b)
	}
}

// 推不出存檔目錄時不寫、也不算失敗——真正的問題是原本那個錯誤，不是紀錄檔。
func TestWriteErrorLogNoPath(t *testing.T) {
	got, err := writeErrorLog("", "訊息")
	if got != "" || err != nil {
		t.Fatalf("got %q, err %v；期望空字串與 nil", got, err)
	}
}

// 啟動成功要清掉上一次失敗的紀錄，免得玩家照著已經修好的問題追。
func TestClearErrorLog(t *testing.T) {
	dir := t.TempDir()
	old := errorLogDir
	t.Cleanup(func() { SetErrorLogDir(old) })
	SetErrorLogDir(dir)

	path := ErrorLogPath()
	if path != filepath.Join(dir, ErrorLogName) {
		t.Fatalf("ErrorLogPath()＝%q", path)
	}
	if _, err := writeErrorLog(path, "上一次的失敗"); err != nil {
		t.Fatalf("writeErrorLog：%v", err)
	}
	ClearErrorLog()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("ClearErrorLog 之後 %s 還在（err=%v）", path, err)
	}
	ClearErrorLog() // 沒有檔案時不能炸
}

// 彈窗內容要帶紀錄檔路徑：玩家按掉視窗之後還找得到完整訊息。
func TestDialogText(t *testing.T) {
	if got := dialogText("壞了", ""); got != "壞了" {
		t.Errorf("沒有紀錄檔時多加了東西：%q", got)
	}
	got := dialogText("壞了", "/tmp/psychicwar-error.log")
	if !strings.Contains(got, "壞了") || !strings.Contains(got, "/tmp/psychicwar-error.log") {
		t.Errorf("彈窗內容少東西：%q", got)
	}
}
