package psychicwar

// F10／F11 即時存檔的中繼資料與版本比對（docs/spec/012 §4）。

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// QuickMeta 是即時存檔旁邊的中繼資料。
type QuickMeta struct {
	Schema   string `json:"schema"`
	Golem    string `json:"golem"`
	ExeSHA   string `json:"exe_sha256"`
	TextSHA  string `json:"text_sha256"`
	Language string `json:"language"`
	SavedAt  string `json:"saved_at"`
}

// QuickSchema 是目前的中繼資料版本。
const QuickSchema = "psychic-war-quicksave/1"

// FileSHA256 回檔案內容的 SHA-256（十六進位）。
func FileSHA256(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// DirSHA256 回目錄下所有 *.json 內容的 SHA-256（檔名排序後串起來算）。
// 目錄不存在時回空字串（不是錯：可以不開中文疊字）。
func DirSHA256(dir string) (string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(files) == 0 {
		return "", nil
	}
	sort.Strings(files)
	h := sha256.New()
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return "", err
		}
		h.Write([]byte(filepath.Base(f)))
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// NewQuickMeta 組一份中繼資料。
func NewQuickMeta(golem, exeSHA, textSHA, lang string) QuickMeta {
	return QuickMeta{Schema: QuickSchema, Golem: golem, ExeSHA: exeSHA, TextSHA: textSHA,
		Language: lang, SavedAt: time.Now().Format(time.RFC3339)}
}

// WriteQuickMeta 寫出中繼資料。
func WriteQuickMeta(path string, m QuickMeta) error {
	b, err := json.MarshalIndent(m, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// ReadQuickMeta 讀回中繼資料。
func ReadQuickMeta(path string) (QuickMeta, error) {
	var m QuickMeta
	b, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	err = json.Unmarshal(b, &m)
	return m, err
}

// CheckQuickMeta 判斷這份存檔能不能讀（docs/spec/012 §4）。
//
// schema、golem 版本、PW.EXE 的 SHA-256 任一不符就不能讀，回 ok=false 與原因；
// 文本檔不同只回一句警告（ok 仍是 true）。
func CheckQuickMeta(m QuickMeta, golem, exeSHA, textSHA string) (ok bool, why string) {
	switch {
	case m.Schema != QuickSchema:
		return false, fmt.Sprintf("存檔格式不符（%s，要 %s）", m.Schema, QuickSchema)
	case m.Golem != golem:
		return false, fmt.Sprintf("golem 版本不符（存檔 %s，現在 %s）", m.Golem, golem)
	case m.ExeSHA != exeSHA:
		return false, "原版 PW.EXE 不是存檔時的那一份"
	case textSHA != "" && m.TextSHA != "" && m.TextSHA != textSHA:
		return true, "譯文檔和存檔時不同，畫面上的中文可能是舊的"
	}
	return true, ""
}
