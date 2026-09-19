package psychicwar

// 玩家設定（docs/spec/023 §6）：目前只有速度檔位。
//
// 放在存檔目錄旁邊，跟即時存檔同一個可寫位置（docs/spec/021 §3.2）。

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SettingsSchema 是設定檔的格式版本。
const SettingsSchema = "psychic-war-settings/1"

// SettingsFile 是設定檔的檔名。
const SettingsFile = "settings.json"

// Settings 是設定檔的內容。
type Settings struct {
	Schema string `json:"schema"`
	Speed  int    `json:"speed"` // 速度檔位（SpeedGears 的其中一個值）
}

// LoadSettings 讀 <dir>/settings.json。
//
// **讀不到或壞掉一律回預設值，不回錯誤。** 設定檔只影響節奏，壞掉不該擋住遊戲；
// 而且玩家看到「設定檔第 3 行格式不對」也不知道要做什麼。
func LoadSettings(dir string) Settings {
	def := Settings{Schema: SettingsSchema, Speed: SpeedGears[0]}
	b, err := os.ReadFile(filepath.Join(dir, SettingsFile))
	if err != nil {
		return def
	}
	var s Settings
	if json.Unmarshal(b, &s) != nil || s.Schema != SettingsSchema {
		return def
	}
	for _, g := range SpeedGears {
		if s.Speed == g {
			return Settings{Schema: SettingsSchema, Speed: g}
		}
	}
	return def
}

// SaveSettings 寫 <dir>/settings.json。
func SaveSettings(dir string, s Settings) error {
	s.Schema = SettingsSchema
	b, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, SettingsFile), append(b, '\n'), 0o644)
}
