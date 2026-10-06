"""研究038 §105：驗完整統計區間及正反對照，不設定速度門檻。"""
from pathlib import Path
import copy
import hashlib
import json
import statistics
import sys

work = Path(sys.argv[1])
counters = ['perf_frame_calls','perf_frame_ns','perf_theme_frame_ns','perf_draw_calls','perf_draw_ns','perf_hd_calls']

def measure(rows,mode):
    ticks = [r for r in rows if r['event']=='tick' and 10000 <= r['wall_ms'] <= 30000]
    assert len(ticks) >= 15
    elapsed = (ticks[-1]['wall_ms']-ticks[0]['wall_ms'])/1000
    assert elapsed >= 17
    for row in ticks:
        assert row['theme_enabled'] == (mode=='on')
        assert row['english'] is False and row['gear']==1 and row['effective']==1 and not row['battle']
        assert row['area']==ticks[0]['area'] and row['map_x']==ticks[0]['map_x'] and row['map_y']==ticks[0]['map_y']
        assert row['perf_actual_fps'] > 0 and row['perf_actual_tps'] > 0
    for left,right in zip(ticks,ticks[1:]):
        assert right['wall_ms'] > left['wall_ms']
        assert all(right[k]>=left[k] for k in counters)
        assert right['perf_frame_calls'] > left['perf_frame_calls']
        assert right['perf_draw_calls'] > left['perf_draw_calls']
    delta = {k:ticks[-1][k]-ticks[0][k] for k in counters}
    assert delta['perf_frame_calls']>0 and delta['perf_draw_calls']>0
    if mode=='on': assert delta['perf_hd_calls']>0
    else: assert delta['perf_hd_calls']==0
    return {'samples':len(ticks),'elapsed_seconds':elapsed,
            'frame_mean_ms':delta['perf_frame_ns']/delta['perf_frame_calls']/1e6,
            'theme_frame_mean_ms':delta['perf_theme_frame_ns']/delta['perf_frame_calls']/1e6,
            'draw_cpu_mean_ms':delta['perf_draw_ns']/delta['perf_draw_calls']/1e6,
            'draws_per_second':delta['perf_draw_calls']/elapsed,
            'updates_per_second':delta['perf_frame_calls']/elapsed,
            'reported_actual_fps_median':statistics.median(r['perf_actual_fps'] for r in ticks),
            'reported_actual_tps_median':statistics.median(r['perf_actual_tps'] for r in ticks),
            'hd_draws':delta['perf_hd_calls'],'counter_delta':delta}

results = {}
inputs = {str(Path(__file__)):hashlib.sha256(Path(__file__).read_bytes()).hexdigest()}
negative = {}
for mode in ['absent','loaded-off','on']:
    folder = work/mode
    terminal = json.loads((folder/'terminal.json').read_text())
    assert terminal['exit_code']==0
    run = json.loads((folder/'execution.json').read_text())
    assert run['start_load']<7 and run['interval_wall_ms']==[10000,30000]
    for name,want in run['inputs_sha256'].items(): assert hashlib.sha256(Path(name).read_bytes()).hexdigest()==want
    rows = [json.loads(line) for line in (folder/'stats.jsonl').read_text().splitlines() if line.strip()]
    assert rows[-1]['event']=='quit'
    results[mode] = measure(rows,mode)
    for f in ['terminal.json','execution.json','stats.jsonl','record.json','quick.state','quick.state.xlate.json','quick.json','window.png']:
        path = folder/f
        assert path.is_file() and path.stat().st_size>0
        inputs[str(path)] = hashlib.sha256(path.read_bytes()).hexdigest()
    bad = copy.deepcopy(rows)
    # Corrupt an interior sample so monotonicity must reject a reset counter.
    changed = [r for r in bad if r['event']=='tick' and 10000 <= r['wall_ms'] <= 30000][1]
    changed['perf_draw_calls'] = 0
    try: measure(bad,mode)
    except AssertionError: negative[mode] = True
    else: raise AssertionError('broken draw counter negative control passed')
result = {'scope':'024 §6.6 local measured frontend current27; fixed normal inventory checkpoint',
          'results':results,'negative_counter_detected':negative,'inputs_sha256':inputs,
          'limits':'Counters/timers add overhead. Two container CPUs, software OpenGL, null audio. No performance threshold, game rule parity, GPU timings, real device, battle/all-sprite or package claim.'}
out = work/'verified.json'
assert not out.exists()
out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({'results':results,'negative_counter_detected':negative},ensure_ascii=False))
