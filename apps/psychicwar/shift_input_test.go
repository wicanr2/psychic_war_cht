package psychicwar

import (
	"github.com/hajimehoshi/ebiten/v2"
	"reflect"
	"testing"
)

func TestShiftFrontendHotkeyDoesNotReachOriginal(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyShiftLeft, ebiten.KeyShiftRight} {
		var input ShiftInput
		if !input.Press(key) {
			t.Fatal("Shift未暫存")
		}
		input.Consume()
		got, handled := input.Release(key)
		if !handled || len(got) != 0 {
			t.Fatalf("前端組合鍵送到原版：%v", got)
		}
	}
}

func TestShiftNormalKeyAndStandalone(t *testing.T) {
	var input ShiftInput
	input.Press(ebiten.KeyShiftLeft)
	input.Consume() // 熱鍵後仍按住Shift，再輸入正常字元，須保留原版大寫支援。
	got := input.BeforeKey()
	if !reflect.DeepEqual(got, []KeyEdge{{ebiten.KeyShiftLeft, true}}) {
		t.Fatal(got)
	}
	if len(input.BeforeKey()) != 0 {
		t.Fatal("重复送Shift")
	}
	input.Consume() // 已送原版的Shift不能被後續熱鍵撤回。
	up, handled := input.Release(ebiten.KeyShiftLeft)
	if !handled || !reflect.DeepEqual(up, []KeyEdge{{ebiten.KeyShiftLeft, false}}) {
		t.Fatal(up)
	}
	input.Press(ebiten.KeyShiftRight)
	alone, handled := input.Release(ebiten.KeyShiftRight)
	if !handled || !reflect.DeepEqual(alone, []KeyEdge{{ebiten.KeyShiftRight, true}, {ebiten.KeyShiftRight, false}}) {
		t.Fatal(alone)
	}
	for _, key := range []ebiten.Key{ebiten.KeyA, ebiten.KeyF1, ebiten.KeyF2, ebiten.KeyF3, ebiten.KeyF9} {
		if input.Press(key) {
			t.Fatal("把原版一般鍵當Shift攔下")
		}
		if _, handled := input.Release(key); handled {
			t.Fatal("攔下原版一般鍵放開")
		}
	}
}
