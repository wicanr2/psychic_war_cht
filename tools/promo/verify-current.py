"""驗實際推廣MP4：影音、黑幀、靜音、靜態分幕凍結與逐幕抽幀。"""
from pathlib import Path
import argparse,hashlib,json,re,subprocess
p=argparse.ArgumentParser();p.add_argument('directory',type=Path);a=p.parse_args();O=a.directory;m=json.loads((O/'rights-and-timeline.json').read_text());movie=next(O.glob('*-promo.mp4'));sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
def call(*args):return subprocess.run(args,capture_output=True,text=True,check=True,timeout=150)
probe=json.loads(call('ffprobe','-v','error','-show_streams','-show_format','-of','json',str(movie)).stdout)
v=next(s for s in probe['streams'] if s['codec_type']=='video');audio=next(s for s in probe['streams'] if s['codec_type']=='audio')
assert v['codec_name']=='h264' and (v['width'],v['height'])==(1280,720) and v['r_frame_rate']=='25/1' and int(v['nb_frames'])==1675
assert audio['codec_name']=='aac' and audio['channels']==2 and abs(float(probe['format']['duration'])-67)<.05 and abs(float(audio['duration'])-67)<.05
result=call('ffmpeg','-hide_banner','-i',str(movie),'-af','volumedetect,silencedetect=n=-50dB:d=1','-vf','blackdetect=d=0.7:pix_th=0.08,freezedetect=n=-50dB:d=2','-threads','2','-f','null','-');log=result.stderr;(O/'av-check.log').write_text(log)
mean=float(re.search(r'mean_volume: ([-\d.]+)',log)[1]);maximum=float(re.search(r'max_volume: ([-\d.]+)',log)[1]);assert -50<mean<-5 and maximum<0
assert not re.findall(r'black_start:',log)
silences=[float(x) for x in re.findall(r'silence_duration: ([\d.]+)',log)];assert max(silences or [0])<6
freezes=[float(x) for x in re.findall(r'freeze_duration: ([\d.]+)',log)];assert len(freezes)>=8 and max(freezes)<8.1
frames=[]
for row in m['timeline']:
 assert row['intentional_static'];time=row['start']+row['duration']/2;f=O/f'frame-{row["index"]:02d}.png';call('ffmpeg','-y','-loglevel','error','-ss',str(time),'-i',str(movie),'-frames:v','1',str(f));frames.append(f)
assert len({sha(f) for f in frames})==9
for start in [0,3,6]:call('convert',*[str(f) for f in frames[start:start+3]],'-resize','640x360','+append',str(O/f'contact-{start//3}.png'))
for path,digest in m['inputs_sha256'].items():assert sha(path)==digest,path
(O/'ffprobe.json').write_text(json.dumps(probe,indent=2)+'\n')
(O/'verification.json').write_text(json.dumps(dict(status='PASS_67S_1675_FRAMES_STEREO_AUDIO_BLACK_SILENCE_AND_PLANNED_STATIC_FREEZES',version=m['version'],movie_sha256=sha(movie),mean_db=mean,max_db=maximum,silence_durations=silences,freeze_durations=freezes,frame_sha256={f.name:sha(f) for f in frames},rights=m['rights'],limits='各分幕刻意靜態已登錄timeline；九幕中心幀需目視字幕及版面。音訊技術驗證不等於人耳確認。'),ensure_ascii=False,indent=2)+'\n');print('67秒／1675幀、音訊與黑幀／靜音／已登錄靜態分幕驗收通過。')
