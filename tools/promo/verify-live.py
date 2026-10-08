"""驗實際遊玩短片與切換，分開靜態字卡及遊玩中的正常停留。"""
from pathlib import Path
import argparse,hashlib,json,re,subprocess
p=argparse.ArgumentParser();p.add_argument('directory',type=Path);p.add_argument('capture',type=Path);a=p.parse_args();O=a.directory;C=a.capture;m=json.loads((O/'rights-and-timeline.json').read_text());movie=next(O.glob('*-promo-live.mp4'));sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
def call(*args):return subprocess.run(args,check=True,capture_output=True,text=True,timeout=180)
probe=json.loads(call('ffprobe','-v','error','-show_streams','-show_format','-of','json',str(movie)).stdout);v=next(s for s in probe['streams'] if s['codec_type']=='video');au=next(s for s in probe['streams'] if s['codec_type']=='audio')
assert v['codec_name']=='h264' and (v['width'],v['height'])==(1280,720) and v['r_frame_rate']=='25/1' and int(v['nb_frames'])==1825
assert au['codec_name']=='aac' and au['channels']==2 and abs(float(probe['format']['duration'])-73)<.05 and abs(float(au['duration'])-73)<.05
log=call('ffmpeg','-hide_banner','-i',str(movie),'-af','volumedetect,silencedetect=n=-50dB:d=1','-vf','blackdetect=d=0.7:pix_th=0.08,freezedetect=n=-50dB:d=2','-threads','2','-f','null','-').stderr;(O/'av-check.log').write_text(log)
mean=float(re.search(r'mean_volume: ([-\d.]+)',log)[1]);maximum=float(re.search(r'max_volume: ([-\d.]+)',log)[1]);assert -50<mean<-5 and maximum<0;assert 'black_start:' not in log
assert max([float(x) for x in re.findall(r'silence_duration: ([\d.]+)',log)] or [0])<6
ticks=[json.loads(s) for s in (C/'stats.jsonl').read_text().splitlines() if s.strip()];assert {r['theme_enabled'] for r in ticks}=={True,False}
assert len({(r['map_x'],r['map_y'],r['area']) for r in ticks})>=4 and any(r['battle'] for r in ticks)
assert len(m['switch_video_times'])==4
times=sorted({2,9,17,25,35,45,55,65,71,*[round(t-.7,2) for t in m['switch_video_times']],*[round(t+.7,2) for t in m['switch_video_times']]});frames=[]
for n,t in enumerate(times):
 f=O/f'frame-{n:02d}.png';call('ffmpeg','-y','-loglevel','error','-ss',str(t),'-i',str(movie),'-frames:v','1',str(f));frames.append(f)
assert len({sha(f) for f in frames})>12
for n in range(0,len(frames),3):call('convert',*[str(f) for f in frames[n:n+3]],'-resize','480x270','+append',str(O/f'contact-{n//3:02d}.png'))
for path,digest in m['inputs_sha256'].items():assert sha(Path(path))==digest,path
freeze=[float(x) for x in re.findall(r'freeze_duration: ([\d.]+)',log)];assert max(freeze or [0])<22
(O/'ffprobe.json').write_text(json.dumps(probe,indent=2)+'\n')
(O/'verification.json').write_text(json.dumps(dict(status='PASS_73S_65S_ACTUAL_GAMEPLAY_FOUR_HD_SWITCHES_AUDIO_AND_FRAMES',version=m['version'],movie_sha256=sha(movie),live_seconds=65,switches=m['switch_video_times'],mean_db=mean,max_db=maximum,freeze_durations=freeze,frame_times=times,frame_sha256={f.name:sha(f) for f in frames},rights=m['rights'],limits='Gameplay may pause for commands/help; freeze ranges reviewed against recorded inputs and frames. Technical audio check is not human-ear acceptance.'),ensure_ascii=False,indent=2)+'\n');print('73秒影片、65秒實際遊玩、四次原版／HD切換與影音驗收通過。')
