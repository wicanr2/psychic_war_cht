package theme

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/dosgolem/oracle"
)

// 024 §1.22：期望由原版RAM與獨立Python收據提供，不由載入器的edges生成。
func TestEnemy01NativeBankSources(t *testing.T) {
	ramPath := os.Getenv("PSYCHICWAR_ENEMY01_BANK_RAM")
	proofPath := os.Getenv("PSYCHICWAR_ENEMY01_BANK_PROOF")
	testNativeBankSources(t, "ENEMY01.PBL", ramPath, proofPath, "workplace/ida/hd-ally-recruit-20261004/body-bank-sivad-v1-20261005.ram")
}

// 024 §1.23：正常保存的三個原始圖庫，期望來自獨立Python模型。
func TestOtherNativeBankSources(t *testing.T) {
	dir := os.Getenv("PSYCHICWAR_BODY_BANK_DIR")
	if dir == "" {
		t.Skip("需明示正常保存圖庫收據目錄")
	}
	for _, suffix := range []string{"00", "03", "04"} {
		t.Run(suffix, func(t *testing.T) {
			prefix := "body-bank-" + suffix + "-v1-20261005"
			testNativeBankSources(t, "ENEMY"+suffix+".PBL", filepath.Join(dir, prefix+".ram"), filepath.Join(dir, prefix+".json"), "workplace/ida/hd-ally-recruit-20261004/"+prefix+".ram")
		})
	}
}

func testNativeBankSources(t *testing.T, file, ramPath, proofPath, ramKey string) {
	t.Helper()
	orig := os.Getenv("PSYCHICWAR_TEST_ORIG")
	if orig == "" || ramPath == "" || proofPath == "" {
		t.Skip("需明示原版、正常Sivad原始RAM與獨立來源收據")
	}
	ram, err := os.ReadFile(ramPath)
	if err != nil || len(ram) != 0xa0000 {
		t.Fatalf("原始RAM不符：%v", err)
	}
	proof, err := os.ReadFile(proofPath)
	if err != nil {
		t.Fatal(err)
	}
	var model struct {
		Schema string            `json:"schema"`
		Inputs map[string]string `json:"input_sha256"`
		Groups []struct {
			Group      int    `json:"group"`
			Initial    int    `json:"initial_image"`
			InitialSHA string `json:"initial_sha256"`
			Phases     []struct {
				Image  int    `json:"image"`
				Unique string `json:"unique_source"`
			} `json:"phases"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(proof, &model); err != nil {
		t.Fatal(err)
	}
	if model.Schema != "psychic-war-native-body-cycle-proof/1" || len(model.Groups) != 5 ||
		model.Inputs[ramKey] != fmt.Sprintf("%x", sha256.Sum256(ram)) {
		t.Fatal("獨立來源收據或原始RAM雜湊不符")
	}
	for _, group := range model.Groups {
		t.Run(fmt.Sprint(group.Group), func(t *testing.T) {
			if group.Group < 0 || group.Group >= 5 || len(group.Phases) != 4 {
				t.Fatal("原版組號或階段數不符")
			}
			start := 0x11750 + 0x63c6 + group.Group*0x650
			current := append([]byte(nil), ram[start:start+384]...)
			if fmt.Sprintf("%x", sha256.Sum256(current)) != group.InitialSHA {
				t.Fatal("初始來源與獨立收據不符")
			}
			deltas := [][]byte{ram[start+0x240 : start+0x3c0], ram[0x1610+0x32ca+group.Group*384 : 0x1610+0x32ca+(group.Group+1)*384]}
			frameFor := func(packed []byte) []byte {
				frame := make([]byte, 64000)
				for i, v := range packed {
					pixel := i * 2
					pos := (152+pixel/24)*320 + 32 + pixel%24
					frame[pos], frame[pos+1] = v>>4, v&15
				}
				return frame
			}
			first, err := loadEnemy(orig, file, group.Initial)
			if err != nil || !bytes.Equal(first.packed, current) {
				t.Fatalf("正式完整來源未對上原版圖庫：%v", err)
			}
			for phase, want := range group.Phases {
				if want.Unique != fmt.Sprintf("%s:%d", file, want.Image) {
					t.Fatal("來源唯一身份不符")
				}
				target, err := loadEnemy(orig, file, want.Image)
				if err != nil || !target.retainDelta {
					t.Fatalf("正式來源或差分保留契約缺失：%v", err)
				}
				which := 1
				if phase == 0 || phase == 3 {
					which = 0
				}
				before := frameFor(current)
				regs := oracle.Regs{AX: 1, CX: 0x0826, DX: 0x0304}
				target.blitWithFrame(regs, func() []byte { return deltas[which] }, func() []byte { return before })
				if !target.active || !target.inFlight {
					t.Fatal("原版實際來源被正式載入器拒絕", phase, want.Image)
				}
				for i := range current {
					current[i] ^= deltas[which][i]
				}
				if !bytes.Equal(target.packed, current) {
					t.Fatal("正式目標與原版差分結果不符")
				}
				target.blitWithFrame(regs, func() []byte { return deltas[1-which] }, func() []byte { return before })
				if target.active {
					t.Fatal("錯階段來源被接受")
				}
			}
		})
	}
}
