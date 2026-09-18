package psychicwar

// 輸入錄製（docs/spec/019）：把按鍵記成「指令數 ＋ 鍵」，之後能一模一樣放回去。
//
// 記指令數不記時間：dosgolem 的決定性來自指令數（docs/re/004），
// 同一個狀態、同一串按鍵落在同樣的指令數上，結果逐位元組相同。牆上時間辦不到——
// 主機忙一點，同樣的 100 ms 會跑到不同的指令數。這也是錄出來的檔案換一台機器仍能重現的原因。

import (
	"encoding/json"
	"os"
)

// RecordEvent 是一次按下或放開。
type RecordEvent struct {
	Step uint64 `json:"step"`
	Key  string `json:"key"`
	Down bool   `json:"down"`
}

// Recording 是一段錄下來的操作（docs/spec/019 §3）。
type Recording struct {
	Schema    string        `json:"schema"`
	Note      string        `json:"note,omitempty"`
	ExeSHA256 string        `json:"pw_exe_sha256"`
	Cycles    int           `json:"cycles"`
	From      string        `json:"from"`
	StartStep uint64        `json:"start_step"`
	Events    []RecordEvent `json:"events"`
}

// RecordingSchema 是格式版本。
const RecordingSchema = "psychic-war-recording/1"

// NewRecording 開一段錄製。
func NewRecording(exeSHA string, cycles int, from string, startStep uint64) *Recording {
	return &Recording{Schema: RecordingSchema, ExeSHA256: exeSHA, Cycles: cycles, From: from, StartStep: startStep}
}

// Add 記一筆。一個按鍵是**兩筆**（按下、放開），不是一筆帶長度：
// 原版有些操作要按住，而「按住多久」在錄製當下是未知的。
func (r *Recording) Add(step uint64, key string, down bool) {
	r.Events = append(r.Events, RecordEvent{Step: step, Key: key, Down: down})
}

// Save 寫成 JSON。
func (r *Recording) Save(path string) error {
	b, err := json.MarshalIndent(r, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// LoadRecording 讀一段錄製並檢查格式與執行檔雜湊（exeSHA 空字串表示不檢查）。
func LoadRecording(path, exeSHA string) (*Recording, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Recording
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	if r.Schema != RecordingSchema {
		return nil, &recordingError{"schema 不是 " + RecordingSchema + "：" + r.Schema}
	}
	if exeSHA != "" && r.ExeSHA256 != "" && r.ExeSHA256 != exeSHA {
		return nil, &recordingError{"錄製時的 PW.EXE 與現在的不同，指令數對不上，不能重播"}
	}
	return &r, nil
}

type recordingError struct{ msg string }

func (e *recordingError) Error() string { return "錄製檔：" + e.msg }
