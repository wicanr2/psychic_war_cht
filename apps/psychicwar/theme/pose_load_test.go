package theme

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
	"os"
	"path/filepath"
	"testing"
)

// 正常來源state的接續回歸；原始bytes由獨立Python核對，入口研究038 §77。
func TestPoseSavedContinuation(t *testing.T) {
	orig, states, themeDir, out := os.Getenv("PSYCHICWAR_TEST_ORIG"), os.Getenv("PSYCHICWAR_POSE_STATES"), os.Getenv("PSYCHICWAR_POSE_THEME"), os.Getenv("PSYCHICWAR_POSE_OUT")
	if orig == "" || states == "" || themeDir == "" || out == "" {
		t.Skip("需要明示正常來源state、本機HD主題及證據輸出目錄")
	}
	for _, c := range []struct {
		name        string
		event, pose int
	}{{"start3", 0, 3}, {"start4", 1, 4}, {"start5", 2, 5}, {"return4", 3, 4}} {
		t.Run(c.name, func(t *testing.T) {
			check := func(e error) {
				if e != nil {
					t.Fatal(e)
				}
			}
			write := func(name string, b []byte) {
				p := filepath.Join(out, c.name+"-"+name)
				f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
				check(e)
				_, e = f.Write(b)
				check(e)
				check(f.Close())
			}
			o, e := oracle.Load(filepath.Join(orig, "PW.EXE"), orig)
			check(e)
			defer o.Close()
			state := filepath.Join(states, fmt.Sprintf("enemy-cycle-runtime-v1-20261001-event%02d-pose%d.state", c.event, c.pose))
			check(o.LoadStateFile(state))
			o.SetScratch(t.TempDir())
			o.SetDOSBoxCycles(750)
			hd, _, e := LoadTheme(themeDir, orig, "", 3)
			check(e)
			check(hd.Attach(o))
			var events []map[string]any
			pending := -1
			o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x8705}, func(o *oracle.Oracle) {
				r := o.Regs()
				if r.CX != 0x0826 || r.DX != 0x0304 {
					return
				}
				if pending >= 0 {
					t.Fatal("敵人貼圖尚未返回便再次進入")
				}
				pending = len(events)
				before, e := pbl.Region(o.Indexed(), 320, 200, 32, 152, 24, 32)
				check(e)
				raw := o.Bytes(oracle.Addr{Seg: r.DS, Off: r.BX}, 384)
				write(fmt.Sprintf("event%02d-before.bin", pending), before)
				write(fmt.Sprintf("event%02d-source.bin", pending), raw)
				events = append(events, map[string]any{"entry_step": o.Steps(), "entry_regs": r, "pending": true})
			})
			o.OnCall(oracle.Addr{Seg: 0x161, Off: 0x8751}, func(o *oracle.Oracle) {
				if pending < 0 {
					return
				}
				after, e := pbl.Region(o.Indexed(), 320, 200, 32, 152, 24, 32)
				check(e)
				write(fmt.Sprintf("event%02d-after.bin", pending), after)
				events[pending]["return_step"] = o.Steps()
				events[pending]["pending"] = false
				pending = -1
			})
			capture := func(phase string) {
				hd.Frame(o)
				write(phase+".frame", o.Indexed())
				over := make([]byte, 960*600*4)
				hd.Draw(over, 3)
				write(phase+".rgba", over)
				check(o.SaveStateFile(filepath.Join(out, c.name+"-"+phase+".state")))
				px, e := pbl.Region(o.Indexed(), 320, 200, 32, 152, 24, 32)
				check(e)
				var statuses []map[string]any
				for i := range hd.groups {
					g := &hd.groups[i]
					if g.sprite == nil || g.sprite.slot() != [4]int{32, 152, 24, 32} {
						continue
					}
					statuses = append(statuses, map[string]any{"group": i, "full": bytes.Equal(px, g.sprite.indexed), "active": g.sprite.active, "in_flight": g.sprite.inFlight})
				}
				b, e := json.MarshalIndent(map[string]any{"source_state": state, "phase": phase, "steps": o.Steps(), "cycles": o.Cycles(), "ip": o.IP(), "pending_event": pending, "events": events, "sprites": statuses}, "", "  ")
				check(e)
				write(phase+".json", b)
			}
			capture("initial")
			for _, phase := range []struct{ name, action string }{{"1000ms", "wait:1000"}, {"1050ms", "wait:50"}} {
				acts, e := oracle.ParseActions(phase.action)
				check(e)
				check(o.RunActions(acts, 0, func() { hd.Frame(o) }))
				capture(phase.name)
			}
			if pending >= 0 {
				t.Fatal("接續50ms後貼圖仍未返回")
			}
			t.Logf("%s: %d次貼圖，接續返回完成", c.name, len(events))
		})
	}
}
