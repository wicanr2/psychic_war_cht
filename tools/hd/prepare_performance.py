"""研究038 §105：以Go覆映射建立本機量測前端，正式main.go不改。"""
from pathlib import Path
import difflib
import hashlib
import json
import sys

root = Path('/src')
out = Path(sys.argv[1])
assert not out.exists() and out.parent.is_dir()
assert (out.parent.stat().st_uid,out.parent.stat().st_gid) == (1000,1000)
source = root/'cmd/psychicwar/main.go'
base = source.read_text()
assert hashlib.sha256(source.read_bytes()).hexdigest() == '794565013a934a69e5cf7a98bde65104253b166d6fb58bb87b4688fb4dffcce0'
new = base
def replace(before,after):
    global new
    assert new.count(before) == 1, before
    new = new.replace(before,after,1)

replace('type game struct {', '''type game struct {
	perfFrameCalls uint64
	perfFrameNS int64
	perfThemeFrameNS int64
	perfDrawCalls uint64
	perfDrawNS int64
	perfHDCalls uint64''')
replace('''	if g.tr != nil {
		g.tr.Frame(g.o)
	}
	g.theme.Frame(g.o)
	g.noteMap()''', '''	var perfFrameStart, perfThemeStart time.Time
	if g.stats != nil { perfFrameStart = time.Now() }
	if g.tr != nil {
		g.tr.Frame(g.o)
	}
	if g.stats != nil { perfThemeStart = time.Now() }
	g.theme.Frame(g.o)
	if g.stats != nil {
		g.perfThemeFrameNS += time.Since(perfThemeStart).Nanoseconds()
		g.perfFrameNS += time.Since(perfFrameStart).Nanoseconds()
		g.perfFrameCalls++
	}
	g.noteMap()''')
replace('''func (g *game) Draw(dst *ebiten.Image) {
	w, h, rgb := g.o.ScreenRGB()''', '''func (g *game) Draw(dst *ebiten.Image) {
	if g.stats != nil {
		perfDrawStart := time.Now()
		defer func() { g.perfDrawCalls++; g.perfDrawNS += time.Since(perfDrawStart).Nanoseconds() }()
	}
	w, h, rgb := g.o.ScreenRGB()''')
replace('''		if g.theme.Draw(g.artPix, g.scale) {
			psychicwar.PremultiplyRGBA(g.artPix)''', '''		if g.theme.Draw(g.artPix, g.scale) {
			if g.stats != nil { g.perfHDCalls++ }
			psychicwar.PremultiplyRGBA(g.artPix)''')
replace('''		"theme_enabled": g.theme != nil && g.theme.Enabled,''', '''		"theme_enabled": g.theme != nil && g.theme.Enabled,
		"perf_frame_calls": g.perfFrameCalls,
		"perf_frame_ns": g.perfFrameNS,
		"perf_theme_frame_ns": g.perfThemeFrameNS,
		"perf_draw_calls": g.perfDrawCalls,
		"perf_draw_ns": g.perfDrawNS,
		"perf_hd_calls": g.perfHDCalls,
		"perf_actual_fps": ebiten.ActualFPS(),
		"perf_actual_tps": ebiten.ActualTPS(),''')
out.mkdir()
(out/'main-base.go').write_bytes(source.read_bytes())
(out/'main-measured.go').write_text(new)
(out/'main.diff').write_text(''.join(difflib.unified_diff(base.splitlines(True),new.splitlines(True),fromfile='formal-main.go',tofile='measurement-main.go')))
(out/'overlay.json').write_text(json.dumps({'Replace':{str(source):str(out/'main-measured.go')}},indent=2)+'\n')
inputs = [source,Path(__file__),root/'go.mod',root/'go.sum',root/'workplace/hd/dat-runtime-v1-20261002.go.work']
(out/'preparation.json').write_text(json.dumps({'scope':'024 §6.6 current27 local telemetry build; formal source unchanged',
    'inputs_sha256':{str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in inputs},
    'added_fields':['Frame combined time','Theme Frame time','Draw CPU time','Draw count','HD Draw count','Ebiten actual FPS/TPS'],
    'limits':'Instrumentation adds timer/counter overhead; no GPU timing, package, hardware or all-sprite claim.'},ensure_ascii=False,indent=2)+'\n')
print('measurement source prepared',out)
