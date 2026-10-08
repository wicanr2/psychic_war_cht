"""原版與HD全圖號總覽，原圖及HD PNG不修改；只輸出本機PNG與來源清單。"""
from pathlib import Path
import argparse,hashlib,importlib.util,json,struct,subprocess,sys,zlib
p=argparse.ArgumentParser();p.add_argument('theme',type=Path);p.add_argument('out',type=Path);a=p.parse_args();assert not a.out.exists();a.out.mkdir(parents=True)
R=Path('/src');sys.path.insert(0,str(R/'tools/hd'));from explore_sprite_sources import strict_pbl
sp=importlib.util.spec_from_file_location('pnghelper',R/'tools/hd/verify_effect_prototype.py');h=importlib.util.module_from_spec(sp);sp.loader.exec_module(h)
manifest=json.loads((a.theme/'manifest.json').read_text());entries={}
for e in manifest['entries']:entries.setdefault((e['pbl'],e['image']),e)
palette=json.loads((R/'workplace/ida/hd-ally-recruit-20261004/hd-completion-opening-parent-native-v1-20261008/palette.json').read_text());inputs={};sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
def save(path,w,hh,rgba):
 def chunk(tag,data):return struct.pack('>I',len(data))+tag+data+struct.pack('>I',zlib.crc32(tag+data)&0xffffffff)
 scan=b''.join(b'\0'+rgba[y*w*4:(y+1)*w*4] for y in range(hh));path.write_bytes(b'\x89PNG\r\n\x1a\n'+chunk(b'IHDR',struct.pack('>IIBBBBB',w,hh,8,6,0,0,0))+chunk(b'IDAT',zlib.compress(scan,6))+chunk(b'IEND',b''))
for family,archives,cols,count in [('enemy',[f'ENEMY{n:02d}.PBL' for n in range(12)],20,360),('ally',['ALLY.PBL'],8,31)]:
 sources=[]
 for name in archives:
  path=Path('/orig/psychic-war')/name;inputs[str(path)]=sha(path)
  for n,w,hh,px,_,_ in strict_pbl(path.read_bytes()):sources.append((name,n,w,hh,bytes(px)))
 assert len(sources)==count
 W,H=cols*160,((count+cols-1)//cols)*160
 for mode in ['original','HD']:
  out=bytearray(bytes([5,10,19,255])*(W*H));draw=[]
  for i,(name,n,w,hh,px) in enumerate(sources):
   if mode=='HD':
    png=a.theme/entries[name,n]['png'];inputs[str(png)]=sha(png);w,hh,pix=h.png(png)
   else:pix=b''.join(bytes(palette[v]+[255]) for v in px)
   factor=min(3 if mode=='original' else 1,130//w,130//hh)
   # 大型圖以最近鄰縮至單格，比例保持。
   nw,nh=(w*factor,hh*factor) if factor else (int(w*min(130/w,130/hh)),int(hh*min(130/w,130/hh)))
   ox,oy=(i%cols)*160+(160-nw)//2,(i//cols)*160+15+(130-nh)//2
   for y in range(nh):
    for x in range(nw):
     j=((y*hh//nh)*w+x*w//nw)*4;k=((oy+y)*W+ox+x)*4;out[k:k+4]=pix[j:j+4]
   draw.extend(['-annotate',f'+{i%cols*160+5}+{i//cols*160+150}',f'{name[:-4]} #{n:02d}'])
  file=a.out/(family+'-'+mode+'.png');save(file,W,H,out)
  subprocess.run(['convert',str(file),'-font','DejaVu-Sans','-pointsize','10','-fill','#55ffff',*draw,str(file)],check=True,timeout=30)
(a.out/'manifest.json').write_text(json.dumps(dict(rights='LOCAL_ONLY',counts={'enemy':360,'ally':31},original_palette='原版fixture的EGA色盤；並非全部色盤相位GUI',inputs_sha256=inputs,outputs={f.name:sha(f) for f in a.out.glob('*.png')}),ensure_ascii=False,indent=2)+'\n');print('四張全圖號總覽完成，只留本機。')
