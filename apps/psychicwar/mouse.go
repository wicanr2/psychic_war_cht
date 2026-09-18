package psychicwar

// 滑鼠：點操作面板等於按對應的鍵（docs/spec/018）。
//
// 原版不支援滑鼠——解壓後 IDA 認得的 93 處 INT 指令裡沒有 33h（docs/spec/018 §1）。
// 這是轉譯層自己提供的功能，不是在模擬原版。

import "github.com/hajimehoshi/ebiten/v2"

// Hotspot 是面板上的一塊熱區：原版像素座標 [X0,X1)×[Y0,Y1)，點下去送 Key。
type Hotspot struct {
	X0, Y0, X1, Y1 int
	Key            ebiten.Key
	Name           string
}

// Hotspots 是操作面板的五塊熱區（docs/spec/018 §2）。
// LOOK ASIDE 那一行左右各半：原版那一行兩端各有一個箭頭，左半向左看、右半向右看。
var Hotspots = []Hotspot{
	{168, 0, 232, 24, ebiten.KeyUp, "前進"},
	{168, 24, 200, 48, ebiten.KeyLeft, "向左看"},
	{200, 24, 232, 48, ebiten.KeyRight, "向右看"},
	{168, 48, 232, 64, ebiten.KeyDown, "向後轉"},
	{168, 64, 232, 80, ebiten.KeyEscape, "選單"},
}

// HotspotAt 回原版像素座標 (x, y) 落在哪一塊熱區；不在任何一塊回 nil。
func HotspotAt(x, y int) *Hotspot {
	for i := range Hotspots {
		h := &Hotspots[i]
		if x >= h.X0 && x < h.X1 && y >= h.Y0 && y < h.Y1 {
			return h
		}
	}
	return nil
}
