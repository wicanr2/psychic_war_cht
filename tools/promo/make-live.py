"""實際遊玩影片：四秒片頭、65秒原版／HD切換遊玩、四秒片尾。"""
from pathlib import Path
import argparse,hashlib,json,os,subprocess
p=argparse.ArgumentParser();p.add_argument('version');p.add_argument('capture',type=Path);p.add_argument('out',type=Path);a=p.parse_args();C=a.capture.resolve();O=a.out.resolve();assert not O.exists();O.mkdir(parents=True)
capture=json.loads((C/'execution.json').read_text());assert capture['version']==a.version and capture['returncode']==0
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
assert sha(C/'gameplay.mp4')==capture['capture_sha256']
fontroot=os.environ.get('PW_PROMO_FONT_ROOT','/usr/share/fonts/opentype/noto')
FONT=fontroot+'/NotoSansCJK-Regular.ttc';BOLD=fontroot+'/NotoSansCJK-Bold.ttc';assert Path(FONT).is_file() and Path(BOLD).is_file()
MUSIC=Path('/src/workplace/dosboxx-audio/title-adlib.wav');assert sha(MUSIC)=='70260f193ac8f22592e20858d2ea0708fd3050a8b8a9bdff053570f59524873a'
def run(*args):subprocess.run(args,check=True,timeout=180,stdout=subprocess.DEVNULL)
def card(name,title,sub):
 f=O/(name+'.png');run('convert','-size','1280x720','radial-gradient:#142137-#03050c','-font',BOLD,'-fill','#55ffff','-pointsize','76','-gravity','center','-annotate','+0-90',title,'-font',FONT,'-fill','#eb93c6','-pointsize','34','-annotate','+0+12',sub,'-fill','#c3d5e3','-pointsize','24','-annotate','+0+82',a.version,str(f));return f
intro=card('intro','銀河超能力戰記','繁體中文化 × 完整 HD · 實際遊玩')
outro=card('outro','銀河超能力戰記','Linux · Windows · macOS')
clips=[]
for name,image in [('intro',intro),('outro',outro)]:
 out=O/(name+'.mp4');run('ffmpeg','-y','-loglevel','error','-loop','1','-framerate','25','-i',str(image),'-t','4','-vf','fade=t=in:d=0.25,fade=t=out:st=3.75:d=0.25,format=yuv420p','-threads','2','-c:v','libx264','-preset','veryfast','-crf','20',str(out))
switches=[e['video_time'] for e in capture['events'] if e['kind']=='key' and e['key']=='F5' and e['shift'] and e['video_time'] is not None]
assert len(switches)==4 and all(0<t<65 for t in switches)
edges=[0,*switches,65];filters=['fps=25','pad=1280:720:160:32:color=0x03050c',f"drawtext=fontfile={FONT}:text='Shift+F5  原版 ↔ HD':fontcolor=white:fontsize=30:x=(w-tw)/2:y=660"]
for n,(lo,hi) in enumerate(zip(edges,edges[1:])):
 label='HD THEME' if n%2==0 else 'ORIGINAL';color='0x55ffff' if n%2==0 else 'white'
 filters.append(f"drawtext=fontfile={BOLD}:text='{label}':fontcolor={color}:fontsize=24:x=170:y=5:enable='gte(t,{lo})*lt(t,{hi})'")
live=O/'live.mp4';run('ffmpeg','-y','-loglevel','error','-i',str(C/'gameplay.mp4'),'-t','65','-vf',','.join(filters)+',format=yuv420p','-an','-threads','2','-c:v','libx264','-preset','veryfast','-crf','20',str(live))
(O/'list.txt').write_text("file 'intro.mp4'\nfile 'live.mp4'\nfile 'outro.mp4'\n");silent=O/'silent.mp4';run('ffmpeg','-y','-loglevel','error','-f','concat','-safe','0','-i',str(O/'list.txt'),'-c','copy',str(silent))
movie=O/('PsychicWar-'+a.version+'-promo-live.mp4');run('ffmpeg','-y','-loglevel','error','-i',str(silent),'-i',str(MUSIC),'-filter_complex','[1:a]atrim=0:73,afade=t=in:d=2,afade=t=out:st=70:d=3[a]','-map','0:v','-map','[a]','-c:v','copy','-c:a','aac','-b:a','192k','-movflags','+faststart',str(movie))
metadata=dict(version=a.version,duration=73,fps=25,live_seconds=65,capture_sha256=sha(C/'gameplay.mp4'),capture_execution_sha256=sha(C/'execution.json'),package_sha256=capture['package_sha256'],build_head=capture['build_head'],switch_video_times=[4+t for t in switches],timeline=[dict(start=0,duration=4,kind='title',intentional_static=True),dict(start=4,duration=65,kind='actual-gameplay',intentional_static=False),dict(start=69,duration=4,kind='outro',intentional_static=True)],inputs_sha256={str(p):sha(p) for p in [MUSIC,C/'gameplay.mp4',C/'execution.json',Path(__file__),Path(FONT),Path(BOLD)]},rights='LOCAL_ONLY: 原版資料、HD及原版音樂；影片不附公開Release',limits='Actual packaged normal boot and keyboard gameplay. No injected state/RAM/position. Music is designated original DOSBox-X recording; not captured device audio.')
(O/'rights-and-timeline.json').write_text(json.dumps(metadata,ensure_ascii=False,indent=2)+'\n');print(movie)
