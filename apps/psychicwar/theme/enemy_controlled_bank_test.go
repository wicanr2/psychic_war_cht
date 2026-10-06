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

// §1.47：期望來自實際原版C6 RAM及獨立PBL收據；不是正常玩家驗收。
func TestControlledEnemyBankSources(t *testing.T) {
	path := os.Getenv("PSYCHICWAR_CONTROLLED_BANK_PROOF")
	orig := os.Getenv("PSYCHICWAR_TEST_ORIG")
	if path == "" || orig == "" {
		t.Skip("需原版及受控C6來源收據")
	}
	var proof struct {
		Schema string
		Rows   []struct {
			File, RAM string
			RAMSHA    string `json:"ram_sha256"`
			Groups    []struct {
				Group      int
				Eligible   bool
				Initial    int    `json:"initial_image"`
				InitialSHA string `json:"initial_sha256"`
				Phases     []struct {
					Image  int
					Unique string `json:"unique_source"`
					SHA    string `json:"packed_sha256"`
				}
			}
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &proof); err != nil || proof.Schema != "psychic-war-controlled-enemy-bank-proof/1" || len(proof.Rows) != 12 {
		t.Fatal("受控來源收據不符", err)
	}
	digest := func(p []byte) string { return fmt.Sprintf("%x", sha256.Sum256(p)) }
	groups, transitions := 0, 0
	for _, bank := range proof.Rows {
		ram, err := os.ReadFile(bank.RAM)
		if err != nil || len(ram) != 0xa0000 || digest(ram) != bank.RAMSHA {
			t.Fatal("原版受控RAM不符", bank.File, err)
		}
		for _, group := range bank.Groups {
			if !group.Eligible {
				for n := group.Group * 3; n < group.Group*3+3; n++ {
					if _, err := loadEnemy(orig, bank.File, n); err == nil {
						t.Fatal("別名或短圖被接受", bank.File, n)
					}
				}
				continue
			}
			groups++
			t.Run(fmt.Sprintf("%s/%d", bank.File, group.Group), func(t *testing.T) {
				if group.Group < 0 || group.Group >= 5 || len(group.Phases) != 4 {
					t.Fatal("組號或階段數不符")
				}
				start := 0x11750 + 0x63c6 + group.Group*0x650
				current := append([]byte(nil), ram[start:start+384]...)
				first, err := loadEnemy(orig, bank.File, group.Initial)
				if err != nil || digest(current) != group.InitialSHA || !bytes.Equal(first.packed, current) {
					t.Fatal("完整來源不同", err)
				}
				deltas := [][]byte{ram[start+0x240 : start+0x3c0], ram[0x1610+0x32ca+group.Group*384 : 0x1610+0x32ca+(group.Group+1)*384]}
				for phase, want := range group.Phases {
					transitions++
					if want.Unique != fmt.Sprintf("%s:%d", bank.File, want.Image) {
						t.Fatal("唯一身份不符")
					}
					frame := make([]byte, 64000)
					for i, v := range current {
						pixel := i * 2
						pos := (152+pixel/24)*320 + 32 + pixel%24
						frame[pos], frame[pos+1] = v>>4, v&15
					}
					target, err := loadEnemy(orig, bank.File, want.Image)
					if err != nil || !target.retainDelta {
						t.Fatal("來源或生命週期契約缺失", err)
					}
					which := 1
					if phase == 0 || phase == 3 {
						which = 0
					}
					regs := oracle.Regs{AX: 1, CX: 0x0826, DX: 0x0304}
					target.blitWithFrame(regs, func() []byte { return deltas[which] }, func() []byte { return frame })
					if !target.active || !target.inFlight {
						t.Fatal("拒絕原版來源")
					}
					for i := range current {
						current[i] ^= deltas[which][i]
					}
					if digest(current) != want.SHA || !bytes.Equal(target.packed, current) {
						t.Fatal("原版差分結果不同")
					}
					target.blitWithFrame(regs, func() []byte { return deltas[1-which] }, func() []byte { return frame })
					if target.active {
						t.Fatal("接受錯階段來源")
					}
					wrong := append([]byte(nil), deltas[which]...)
					wrong[0] ^= 1
					target.blitWithFrame(regs, func() []byte { return wrong }, func() []byte { return frame })
					if target.active {
						t.Fatal("接受一bit錯誤差分")
					}
				}
			})
		}
	}
	if groups != 58 || transitions != 232 {
		t.Fatal("來源覆蓋不符", groups, transitions)
	}
	// 已知檔案版本的一bit變異必須被拒絕。
	dir := t.TempDir()
	b, err := os.ReadFile(filepath.Join(orig, "ENEMY11.PBL"))
	if err != nil {
		t.Fatal(err)
	}
	b[len(b)-1] ^= 1
	if err = os.WriteFile(filepath.Join(dir, "ENEMY11.PBL"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = loadEnemy(dir, "ENEMY11.PBL", 0); err == nil {
		t.Fatal("接受錯誤來源SHA")
	}
}

func TestEnemySourceBounds(t *testing.T) {
	for _, name := range []string{"ENEMY02.PBL", "ENEMY08.PBL"} {
		for _, n := range []int{-1, 15, 29, 30} {
			if _, _, err := enemySource(name, n); err == nil {
				t.Fatal("接受未READY圖號", name, n)
			}
		}
	}
	for _, name := range []string{"ENEMY12.PBL", "enemy00.pbl", "ENEMY99.PBL", "../ENEMY00.PBL"} {
		if _, _, err := enemySource(name, 0); err == nil || knownEnemySource(name) {
			t.Fatal("接受未知來源", name)
		}
	}
}
