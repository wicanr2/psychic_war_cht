"""retro-remake推廣片流程：新版擷取、多版面、指定原版配樂、67秒。Docker內執行。"""
from pathlib import Path
import argparse,hashlib,json,subprocess
p=argparse.ArgumentParser();p.add_argument('version');p.add_argument('out');a=p.parse_args()
R=Path('/src');B=R/'workplace/ida/hd-ally-recruit-20261004';G=B/'hd-full-boot-gui-v3-20261008';D=B/'hd-full-dat-load-v1-20261008';O=Path(a.out);assert not O.exists();O.mkdir(parents=True)
FONT='/fonts/opentype/noto/NotoSansCJK-Regular.ttc';BOLD='/fonts/opentype/noto/NotoSansCJK-Bold.ttc'
assert Path(FONT).is_file() and Path(BOLD).is_file()
MUSIC=R/'workplace/dosboxx-audio/title-adlib.wav';assert MUSIC.is_file()
inputs={};sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
def source(p):p=Path(p);assert p.is_file();inputs[str(p)]=sha(p);return str(p)
def run(*args):subprocess.run(args,check=True,timeout=100,stdout=subprocess.DEVNULL)
def bg(n,color='#072033'):
 f=O/f'{n:02d}.png';run('convert','-size','1280x720','radial-gradient:'+color+'-#03050c','-fill','#55ffff20','-draw','path "M 32,85 L 32,32 L 210,32 M 1070,688 L 1248,688 L 1248,635"',str(f));return f
def text(f,main,sub='',y=70):
 run('convert',str(f),'-font',BOLD,'-fill','#55ffff','-pointsize','42','-gravity','northwest','-annotate',f'+64+{y}',main,'-font',FONT,'-fill','#deeef6','-pointsize','27','-annotate',f'+66+{y+60}',sub,str(f))
def paste(f,p,size,at):
 tmp=O/'paste.png';run('convert',source(p),'-resize',size,'-bordercolor','#55ffff','-border','2',str(tmp));run('convert',str(f),str(tmp),'-gravity','northwest','-geometry','+'+at.replace(',', '+'),'-composite',str(f))
def footer(f,label):
 run('convert',str(f),'-fill','#03050c','-draw','rectangle 0,630 1280,720','-font',FONT,'-pointsize','32','-fill','#f3f5ff','-gravity','south','-annotate','+0+30',label,str(f))
scenes=[]
f=bg(0,'#142137');run('convert',str(f),'-font',BOLD,'-fill','#55ffff','-pointsize','78','-gravity','center','-annotate','+0-80','銀河超能力戰記','-font',FONT,'-fill','#eb93c6','-pointsize','34','-annotate','+0+24','繁體中文化 × 完整 HD 圖面','-fill','#a6b9cf','-pointsize','26','-annotate','+0+95','PSYCHIC WAR · COSMIC SOLDIER 2',str(f));scenes.append((f,6,'title'))
f=bg(1);text(f,'經典畫面，原位升級','原版與 HD 的同場景對照');paste(f,G/'maze-original.png','530x','64,220');paste(f,G/'maze-hd.png','530x','682,220');footer(f,'位置、比例與原版排版保持');scenes.append((f,8,'split'))
f=bg(2);paste(f,G/'maze-hd.png','960x600!','158,18');footer(f,'走進薩瑪，從第一人稱迷宮展開旅程');scenes.append((f,8,'game-frame'))
f=bg(3,'#22132c');paste(f,D/'c-items-kai-phase03.png','780x','40,170');text(f,'戰友與裝備','人物、道具與小圖',y=44);run('convert',str(f),'-font',FONT,'-fill','#f0d8ea','-pointsize','32','-gravity','northwest','-annotate','+872+265','31 個盟友圖號','-annotate','+872+335','360 個敵人圖號','-annotate','+872+405','原版資料保持',str(f));footer(f,'同一套日式科幻美術，沿原版造型與動作');scenes.append((f,8,'sidebar'))
f=bg(4,'#271121');text(f,'按住空白鍵，迎戰超能力者','目前版本的普通戰鬥 GUI 抽樣');paste(f,B/'hd-completion-gui-battle-v3-20261008/phase05.png','550x','50,210');paste(f,B/'hd-completion-gui-battle-v3-20261008/phase24.png','550x','676,210');footer(f,'光束、攻擊與遮罩效果，採透光混色');scenes.append((f,8,'battle-pair'))
f=bg(5,'#182336');text(f,'圖面收錄','完整 HD 以原版圖號對應');T=R/'workplace/hd/theme-full-scenes-ally31-maze-B-v1-20261008'
for i,name in enumerate(['ALLY-00.png','ALLY-02.png','ALLY-03.png','ALLY-04.png']):paste(f,T/name,'135x180',''+str(116+i*286)+',230')
footer(f,'原位人物、原位框線，全域 8×8 遮罩');scenes.append((f,8,'character-gallery'))
f=bg(6);text(f,'場景圖面','房間、地圖、開場與結局');plan=json.loads(source(B/'hd-completion-room63-art-plan-v1-20261008.json') and (B/'hd-completion-room63-art-plan-v1-20261008.json').read_text())
for i,n in enumerate([3,11,25,29]):paste(f,B/'hd-completion-room63-art-outputs-v1-20261008'/f'ROOM0.PBL-{n:02d}.png','230x230',''+str(80+i*302)+',240')
footer(f,'場景候選的實際圖面驗證輸出');scenes.append((f,8,'scene-gallery'))
f=bg(7);paste(f,G/'help.png','960x600!','158,18');footer(f,'F4 說明 · F5 中英 · F10／F11 即時存讀');scenes.append((f,7,'help'))
f=bg(8,'#1d1430');run('convert',str(f),'-font',BOLD,'-fill','#55ffff','-pointsize','65','-gravity','center','-annotate','+0-98','銀河超能力戰記','-font',FONT,'-fill','#ffffff','-pointsize','32','-annotate','+0+0','Linux · Windows · macOS','-fill','#eb93c6','-pointsize','28','-annotate','+0+68','公開程式包需自備原版；完整 HD 僅本機','-fill','#a6b9cf','-pointsize','24','-annotate','+0+130','github.com/wicanr2/psychic_war_cht · '+a.version,str(f));scenes.append((f,6,'outro'))
concat=O/'list.txt';lines=[];timeline=[];start=0
for n,(f,duration,layout) in enumerate(scenes):
 clip=O/f's{n:02d}.mp4';run('ffmpeg','-y','-loglevel','error','-loop','1','-framerate','25','-i',str(f),'-t',str(duration),'-vf',f'fade=t=in:d=0.25,fade=t=out:st={duration-.25}:d=0.25,format=yuv420p','-threads','2','-c:v','libx264','-preset','veryfast',str(clip));lines.append(f"file '{clip.name}'");timeline.append(dict(index=n,start=start,duration=duration,layout=layout,frame=f.name,intentional_static=True));start+=duration
concat.write_text('\n'.join(lines)+'\n');silent=O/'silent.mp4';run('ffmpeg','-y','-loglevel','error','-f','concat','-safe','0','-i',str(concat),'-c','copy',str(silent))
movie=O/('PsychicWar-'+a.version+'-promo.mp4');source(MUSIC);run('ffmpeg','-y','-loglevel','error','-i',str(silent),'-i',str(MUSIC),'-filter_complex',f'[1:a]atrim=0:{start},afade=t=in:d=2,afade=t=out:st={start-3}:d=3[a]','-map','0:v','-map','[a]','-c:v','copy','-c:a','aac','-b:a','192k','-movflags','+faststart',str(movie))
inputs[str(Path(__file__))]=sha(Path(__file__));inputs[FONT]=sha(FONT);inputs[BOLD]=sha(BOLD)
(O/'rights-and-timeline.json').write_text(json.dumps(dict(version=a.version,duration=start,theme='1987星圖與桃紅科幻',differences_from_previous=['HD原位對照敘事','六種內容版面','新版擷取及HD場景圖面'],timeline=timeline,inputs_sha256=inputs,rights='LOCAL_ONLY: 原版音樂及HD素材；公開前需提醒並處理著作權。',music='DOSBox-X原版實錄，非自寫合成器',limits='以正常GUI擷取及標明的圖面驗證輸出製作靜態短片；不宣稱全程試玩或所有動畫正常GUI已驗。'),ensure_ascii=False,indent=2)+'\n');print(movie)
