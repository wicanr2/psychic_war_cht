package psychicwar

// F3 自動地圖（docs/spec/015）：走過才記，不解地圖檔。

import (
	"encoding/json"
	"fmt"
	"sort"
)

// 腳本變數的線性位址（docs/re/011）。
const (
	AddrArea   = 0x16966
	AddrMapX   = 0x16968
	AddrMapY   = 0x1696A
	AddrFacing = 0x16970
)

// MapMax 是地圖格子的上限（座標超過就不記；載入畫面時那幾個字組可能是垃圾）。
const MapMax = 63

// AreaMax 是區域編號的上限。
const AreaMax = 11

// AutoMap 記每個區域走過的格子。
type AutoMap struct {
	seen map[uint16]map[[2]uint16]bool
}

// NewAutoMap 建一張空地圖。
func NewAutoMap() *AutoMap { return &AutoMap{seen: map[uint16]map[[2]uint16]bool{}} }

// Note 記下一格；值不合理就不記，回是不是新的格子。
func (m *AutoMap) Note(area, x, y uint16) bool {
	if area > AreaMax || x > MapMax || y > MapMax {
		return false
	}
	g := m.seen[area]
	if g == nil {
		g = map[[2]uint16]bool{}
		m.seen[area] = g
	}
	if g[[2]uint16{x, y}] {
		return false
	}
	g[[2]uint16{x, y}] = true
	return true
}

// Seen 回這一格走過沒有。
func (m *AutoMap) Seen(area, x, y uint16) bool { return m.seen[area][[2]uint16{x, y}] }

// Count 回這個區域走過幾格。
func (m *AutoMap) Count(area uint16) int { return len(m.seen[area]) }

// Cells 回這個區域走過的格子（排序過，畫圖與測試用）。
func (m *AutoMap) Cells(area uint16) [][2]uint16 {
	out := make([][2]uint16, 0, len(m.seen[area]))
	for c := range m.seen[area] {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i][1] != out[j][1] {
			return out[i][1] < out[j][1]
		}
		return out[i][0] < out[j][0]
	})
	return out
}

// MarshalJSON 存成 {"<區域>": [[x, y], …]}（docs/spec/015 §3）。
func (m *AutoMap) MarshalJSON() ([]byte, error) {
	out := map[string][][2]uint16{}
	for area := range m.seen {
		out[fmt.Sprintf("%d", area)] = m.Cells(area)
	}
	return json.Marshal(map[string]any{"schema": "psychic-war-automap/1", "areas": out})
}

// UnmarshalJSON 讀回；格式不對就當空的（讀檔不該因為地圖壞掉而失敗）。
func (m *AutoMap) UnmarshalJSON(b []byte) error {
	var doc struct {
		Schema string                 `json:"schema"`
		Areas  map[string][][2]uint16 `json:"areas"`
	}
	if err := json.Unmarshal(b, &doc); err != nil || doc.Schema != "psychic-war-automap/1" {
		m.seen = map[uint16]map[[2]uint16]bool{}
		return nil
	}
	m.seen = map[uint16]map[[2]uint16]bool{}
	for k, cells := range doc.Areas {
		var area uint16
		if _, err := fmt.Sscanf(k, "%d", &area); err != nil {
			continue
		}
		for _, c := range cells {
			m.Note(area, c[0], c[1])
		}
	}
	return nil
}

// FacingMark 回朝向的記號（0 北、1 東、2 南、3 西，docs/re/011）。
func FacingMark(f uint16) rune {
	switch f & 3 {
	case 0:
		return '↑'
	case 1:
		return '→'
	case 2:
		return '↓'
	}
	return '←'
}
