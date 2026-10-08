#!/bin/bash
# 在psychicwar-dosboxx:source-5fcf624b-r1內執行；/orig唯讀、/out唯一輸出。
# 入口沿tools/dosboxx-audio.sh與docs/re/034；只錄原版，不作golem parity收據。
set -euo pipefail
export HOME=/tmp DISPLAY=:99 SDL_AUDIODRIVER=dummy
test -f /orig/PW.EXE
test ! -e /out/title-adlib.wav
test "$(stat -c '%u:%g' /out)" = "$(id -u):$(id -g)"
uptime | tee /out/capture-start-load.txt
awk '{if ($1>=7) exit 1}' /proc/loadavg || { echo '起跑load>=7，保留未開跑記錄。'; exit 75; }
mkdir /out/capture /tmp/game
cp -r /orig/. /tmp/game/
chmod -R u+w /tmp/game
task_dbx_pid='';task_xvfb_pid=''
cleanup() {
    if test -n "$task_dbx_pid"; then
        kill "$task_dbx_pid" 2>/dev/null || true
        for task_close_try in {1..20}; do kill -0 "$task_dbx_pid" 2>/dev/null || break; sleep .1; done
        if kill -0 "$task_dbx_pid" 2>/dev/null; then kill -KILL "$task_dbx_pid" 2>/dev/null || true; fi
        wait "$task_dbx_pid" 2>/dev/null || true
    fi
    if test -n "$task_xvfb_pid"; then kill "$task_xvfb_pid" 2>/dev/null || true; wait "$task_xvfb_pid" 2>/dev/null || true; fi
}
trap cleanup EXIT INT TERM
mkdir -p /tmp/.X11-unix
chmod 1777 /tmp/.X11-unix
Xvfb :99 -screen 0 1024x768x24 -nolisten tcp >/out/xvfb.log 2>&1 & task_xvfb_pid=$!
for task_try in {1..30}; do xdotool getdisplaygeometry >/dev/null 2>&1 && break; sleep .1; done
xdotool getdisplaygeometry >/dev/null
cat > /out/original-adlib.conf <<'CONF'
[sdl]
output=surface
autolock=false
windowresolution=original
[dosbox]
machine=svga_s3
memsize=4
hostkey=ctrlalt
captures=/out/capture
[render]
aspect=false
scaler=none
[cpu]
core=normal
cputype=286
cycles=fixed 8000
[mixer]
rate=22050
[sblaster]
sbtype=sb1
oplmode=opl2
[gus]
gus=false
[speaker]
pcspeaker=true
[autoexec]
mount c /tmp/game
c:
PW.EXE
CONF
dosbox-x -conf /out/original-adlib.conf -nomenu >/out/dosbox.log 2>&1 & task_dbx_pid=$!
task_window=''
for task_try in {1..100}; do
    task_window=$(xdotool search --name 'DOSBox-X' 2>/dev/null | tail -1 || true)
    test -n "$task_window" && break
    kill -0 "$task_dbx_pid" 2>/dev/null || { echo 'DOSBox-X已退出。'; exit 1; }
    sleep .1
done
test -n "$task_window"
eval "$(xdotool getwindowgeometry --shell "$task_window")"
xdotool mousemove "$((X+WIDTH/2))" "$((Y+HEIGHT/2))"
xdotool key --clearmodifiers ctrl+alt+w
sleep 95
xdotool key --clearmodifiers ctrl+alt+w
sleep 2
cleanup;task_dbx_pid='';task_xvfb_pid=''
task_wav=$(find /out/capture -maxdepth 1 -type f -iname '*.wav' | sort | tail -1)
test -n "$task_wav" && test -s "$task_wav"
cp "$task_wav" /out/title-adlib.wav
python3 - <<'PY'
from pathlib import Path
import hashlib,json,subprocess,wave
p=Path('/out/title-adlib.wav')
with wave.open(str(p),'rb') as w:
    frames=w.getnframes();rate=w.getframerate();channels=w.getnchannels();width=w.getsampwidth()
    duration=frames/rate
assert rate==22050 and channels in [1,2] and width==2
assert 93<=duration<=99,(frames,rate,duration)
assert frames==int(duration*rate)
source=Path('/orig/PW.EXE')
result=dict(status='PASS_ORIGINAL_DOSBOXX_ADLIB_CAPTURE_PENDING_AUDIO_SIGNALS',
    original_sha256=hashlib.sha256(source.read_bytes()).hexdigest(),wav_sha256=hashlib.sha256(p.read_bytes()).hexdigest(),
    frames=frames,rate=rate,channels=channels,sample_width=width,duration=duration,
    source='DOSBox-X executing original PW.EXE; native wave recorder, no custom FM synthesis.',
    script_sha256=hashlib.sha256(Path('/tools/capture-original-adlib.sh').read_bytes()).hexdigest(),
    rights='local-only original music',limits='Auxiliary original recording; no dosgolem parity, human-ear or public music-license claim.')
Path('/out/original-adlib-capture.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
print('原版AdLib錄音',duration,'秒，取樣數相符。')
PY
ffprobe -v error -show_streams -show_format -of json /out/title-adlib.wav > /out/ffprobe.json
ffmpeg -hide_banner -i /out/title-adlib.wav -af volumedetect -f null - > /out/volume.log 2>&1
stat -c '%u:%g %n' /out/title-adlib.wav
