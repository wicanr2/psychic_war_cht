package psychicwar

import "github.com/hajimehoshi/ebiten/v2"

// ShiftInput 暫存Shift，待一般鍵或前端熱鍵確定用途。規格024 §4.1。
type ShiftInput struct{ keys [2]shiftState }
type shiftState struct{ held, sent, consumed bool }
type KeyEdge struct {
	Key  ebiten.Key
	Down bool
}

func shiftIndex(k ebiten.Key) int {
	switch k {
	case ebiten.KeyShiftLeft:
		return 0
	case ebiten.KeyShiftRight:
		return 1
	}
	return -1
}

func (s *ShiftInput) Press(k ebiten.Key) bool {
	i := shiftIndex(k)
	if i < 0 {
		return false
	}
	s.keys[i] = shiftState{held: true}
	return true
}

// Consume 只消耗尚未進入原版的修飾鍵，不撤回先前一般鍵已使用的Shift。
func (s *ShiftInput) Consume() {
	for i := range s.keys {
		if s.keys[i].held && !s.keys[i].sent {
			s.keys[i].consumed = true
		}
	}
}

func (s *ShiftInput) BeforeKey() []KeyEdge {
	var edges []KeyEdge
	for i := range s.keys {
		k := &s.keys[i]
		if !k.held || k.sent {
			continue
		}
		key := ebiten.KeyShiftLeft
		if i == 1 {
			key = ebiten.KeyShiftRight
		}
		edges = append(edges, KeyEdge{key, true})
		k.sent = true
	}
	return edges
}

func (s *ShiftInput) Release(k ebiten.Key) ([]KeyEdge, bool) {
	i := shiftIndex(k)
	if i < 0 {
		return nil, false
	}
	state := s.keys[i]
	s.keys[i] = shiftState{}
	if state.sent {
		return []KeyEdge{{k, false}}, true
	}
	if state.held && !state.consumed {
		return []KeyEdge{{k, true}, {k, false}}, true
	}
	return nil, true
}
