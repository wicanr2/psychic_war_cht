package translator

// 圖檔內嵌文字的中文疊字（docs/spec/011）：畫面上某塊等於原版圖塊時就蓋中文
// （dosgolem 規格 203-baked-text-watchers）。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wicanr2/dosgolem/xlate"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
)

// BakedEntry 是 text/baked.json 的一筆（docs/spec/011 §3）。
type BakedEntry struct {
	Key         string `json:"key"`
	File        string `json:"file"`
	Image       int    `json:"image"`
	Screen      [2]int `json:"screen"` // 這張圖畫在畫面上的左上角
	Region      [4]int `json:"region"` // 圖內座標 x, y, w, h：要比對的區塊
	Text        [3]int `json:"text"`   // 圖內座標 x, y 與格數：中文蓋在哪
	Original    string `json:"original"`
	Translation string `json:"translation"`
	// Font：cjk24（一格 8×8，預設）或 cjk16（一格 6×7，字模 16×15 偏移 (1,3)）——
	// 招牌上的字常常只有 5 像素高，用大格會蓋掉上下的美術（docs/spec/011 §3）。
	Font string `json:"font"`
	// SwapColors：定色時把背景與前景對調（dosgolem `202-translation-overlay` §2.3）。
	// 用在字比底密的區塊——整條橫幅被字填滿時，字的像素多於底，「最多的當背景」會反過來。
	SwapColors bool   `json:"swap_colors,omitempty"`
	Note       string `json:"note"`
}

type bakedFile struct {
	Schema  string       `json:"schema"`
	Note    string       `json:"note"`
	Entries []BakedEntry `json:"entries"`
}

// LoadBaked 讀 <dir>/baked.json。檔案不存在時回 nil（沒有圖檔疊字，不是錯）。
func LoadBaked(dir string) ([]BakedEntry, error) {
	b, err := os.ReadFile(filepath.Join(dir, "baked.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var doc bakedFile
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	if doc.Schema != "psychic-war-baked/1" {
		return nil, fmt.Errorf("baked.json 的 schema 不是 psychic-war-baked/1：%q", doc.Schema)
	}
	return doc.Entries, nil
}

// AttachBaked 依資料檔登記 watcher；orig 是玩家自備的原版目錄。
// 原版檔缺少或資料檔有問題時記一筆紀錄並跳過那一筆，遊戲照樣跑（docs/spec/011 §4）。
func (t *Translator) AttachBaked(entries []BakedEntry, orig string) {
	cache := map[string][]byte{}
	for _, e := range entries {
		data, ok := cache[e.File]
		if !ok {
			b, err := os.ReadFile(filepath.Join(orig, e.File))
			if err != nil {
				t.once("baked-missing", e.File)
				cache[e.File] = nil
				continue
			}
			data, cache[e.File] = b, b
		}
		if data == nil {
			continue
		}
		w, h, px, err := pbl.Decode(data, e.Image)
		if err != nil {
			t.once("baked-decode", e.Key)
			continue
		}
		want, err := pbl.Region(px, w, h, e.Region[0], e.Region[1], e.Region[2], e.Region[3])
		if err != nil {
			t.once("baked-region", e.Key)
			continue
		}
		ent := e // 迴圈變數要複製一份給 closure
		t.Layer.Watch(&xlate.Watcher{
			Key:  ent.Key,
			X:    ent.Screen[0] + ent.Region[0],
			Y:    ent.Screen[1] + ent.Region[1],
			W:    ent.Region[2],
			H:    ent.Region[3],
			Want: want,
			Make: func() []*xlate.Stamp {
				s := t.bakedStamp(ent)
				if s == nil {
					return nil
				}
				return []*xlate.Stamp{s}
			},
		})
	}
}

// bakedStamp 依一筆資料建疊字；沒有譯文或譯文過長時記紀錄並回 nil。
func (t *Translator) bakedStamp(e BakedEntry) *xlate.Stamp {
	cells := e.Text[2]
	if e.Translation == "" {
		t.once("missing-translation", e.Key)
		return nil
	}
	text := []rune(e.Translation)
	if len(text) > cells {
		t.once("too-long", e.Key)
		return nil
	}
	s := &xlate.Stamp{
		Key: e.Key, X: e.Screen[0] + e.Text[0], Y: e.Screen[1] + e.Text[1],
		Cells: cells, CellW: bakedCell, CellH: bakedCell,
		Font: t.Font24, GlyphScale: t.Scale / 3, Text: text, State: xlate.Pending,
		SwapColors: e.SwapColors,
	}
	if e.Font == "cjk16" { // 小字型：與 docs/spec/009 的 B 路徑同一套幾何
		s.CellW, s.CellH, s.Font = smallCellW, smallCellH, t.Font16
		s.GlyphX, s.GlyphY = t.Scale/3, 3*t.Scale/3
	}
	t.event(map[string]any{"event": "stamp", "key": e.Key, "line": 0,
		"text": string(text), "x": s.X, "y": s.Y, "cells": cells})
	return s
}

// 圖檔疊字的格子（docs/spec/011 §3）：預設與 FONT.BIN 路徑相同的 8×8；
// font 指定 cjk16 時是小字型的 6×7。
const (
	bakedCell  = 8
	smallCellW = 6
	smallCellH = 7
)
