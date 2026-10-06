// 研究038 §90：明示原版state與方向按鍵；只讀來源與逐鍵位置。
package main

import (
 "bytes"
 "crypto/sha256"
 "encoding/json"
 "flag"
 "fmt"
 "log"
 "os"
 "path/filepath"
 "runtime"
 "strings"
 "github.com/wicanr2/dosgolem/oracle"
 "github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
)

func check(e error){if e!=nil{log.Fatal(e)}}
func read(p string)[]byte{b,e:=os.ReadFile(p);check(e);return b}
func hash(b []byte)string{return fmt.Sprintf("%x",sha256.Sum256(b))}
func write(p string,b []byte){f,e:=os.OpenFile(p,os.O_WRONLY|os.O_CREATE|os.O_EXCL,0644);check(e);_,e=f.Write(b);check(e);check(f.Close())}
func jsonWrite(p string,v any){b,e:=json.MarshalIndent(v,"","  ");check(e);write(p,append(b,'\n'))}
type pose struct{Key string;W,H int;Pixels,Packed []byte}
type event struct{Entry,Return uint64;Regs oracle.Regs;X,Y,W,H int;Before,After,Source,State string;FullBefore,FullAfter,RawFull []string}
type fingerprint struct{Steps,Cycles uint64;Regs oracle.Regs;RAM,Frame string}
func fp(o *oracle.Oracle)fingerprint{return fingerprint{o.Steps(),o.Cycles(),o.Regs(),hash(o.Bytes(oracle.Addr{},1<<20)),hash(o.Indexed())}}
func location(o *oracle.Oracle)map[string]uint16{return map[string]uint16{"area":o.Word(oracle.Addr{Seg:0x1696,Off:6}),"x":o.Word(oracle.Addr{Seg:0x1696,Off:8}),"y":o.Word(oracle.Addr{Seg:0x1696,Off:10}),"facing":o.Word(oracle.Addr{Seg:0x1697,Off:0}),"hp":o.Word(oracle.Addr{Seg:0x1699,Off:0}),"energy":o.Word(oracle.Addr{Seg:0x1699,Off:4}),"forward":o.Word(oracle.Addr{Seg:0x1697,Off:6})}}
func main(){
 out:=flag.String("out","","全新輸出目錄，父目錄已存在");startFlag:=flag.String("state","","已存在的正常原版state");frameFlag:=flag.String("frame","","對應原版frame");shaFlag:=flag.String("sha","","起始state固定SHA");routeFlag:=flag.String("route","","最多16個Up/Left/Right/Down以逗號分隔");flag.Parse();if *out==""{log.Fatal("缺輸出")}
 if _,e:=os.Stat(*out);e==nil{log.Fatal("拒絕覆寫")};check(os.Mkdir(*out,0755))
 const orig="/orig/psychic-war";start:=*startFlag
 if start==""||*frameFlag==""||len(*shaFlag)!=64||*routeFlag==""{log.Fatal("缺明示原版起始或路線")}
 route:=strings.Split(*routeFlag,",");if len(route)>16{log.Fatal("路線超出16鍵")};scans:=map[string]uint8{"Up":0x48,"Left":0x4b,"Right":0x4d,"Down":0x50};for _,k:=range route{if _,ok:=scans[k];!ok{log.Fatal("未知方向鍵")}}
 inputs:=map[string]string{}
 for _,p:=range []string{start,*frameFlag,"replay/title-to-first-save.json","tools/hd/observe_zellwal_route.go",orig+"/PW.EXE"}{inputs[p]=hash(read(p))}
 if inputs[start]!=*shaFlag||inputs[orig+"/PW.EXE"]!="88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49"{log.Fatal("固定原版輸入不符")}
 names:=[]string{"ALLY.PBL"};for n:=0;n<12;n++{names=append(names,fmt.Sprintf("ENEMY%02d.PBL",n))}
 poses:=[]pose{}
 for _,name:=range names{
  path:=filepath.Join(orig,name);data:=read(path);inputs[path]=hash(data);offsets,e:=pbl.Offsets(data);check(e)
  count:=30;if name=="ALLY.PBL"{count=31};if len(offsets)!=count{log.Fatal("實際PBL圖數不符")}
  for n:=range offsets{w,h,px,e:=pbl.Decode(data,n);check(e);packed:=make([]byte,len(px)/2);for i:=range packed{packed[i]=px[i*2]<<4|px[i*2+1]};poses=append(poses,pose{fmt.Sprintf("%s:%d",name,n),w,h,px,packed})}
 }
 matches:=func(frame []byte,x,y,w,h int)[]string{
  px,e:=pbl.Region(frame,320,200,x,y,w,h);check(e);ids:=[]string{}
  for _,p:=range poses{if p.W==w&&p.H==h&&bytes.Equal(p.Pixels,px){ids=append(ids,p.Key)}};return ids
 }
 create:=func()*oracle.Oracle{o,e:=oracle.Load(orig+"/PW.EXE",orig);check(e);check(o.LoadStateFile(start));o.SetScratch(filepath.Join(*out,"scratch"));return o}
 observed:=create();defer observed.Close();initial:=fp(observed);startLoc:=location(observed)
 if startLoc["area"]!=3{log.Fatal("正常Zellwal區域不同")}
 if !bytes.Equal(observed.Indexed(),read(*frameFlag)){log.Fatal("state與原版frame不同")}
 seed:=observed.Word(oracle.Addr{Seg:0x161,Off:0x41df});events:=[]event{};total,entered,returned:=0,0,0
 var pending *event;var before,raw []byte
 observed.OnCall(oracle.Addr{Seg:0x161,Off:0x47d4},func(o *oracle.Oracle){entered++})
 observed.OnCall(oracle.Addr{Seg:0x161,Off:0x47ad},func(o *oracle.Oracle){returned++})
 observed.OnCall(oracle.Addr{Seg:0x161,Off:0x8705},func(o *oracle.Oracle){
  r:=o.Regs();x,y,w,h:=int(r.CX>>8)*4,int(r.CX&255)*4,int(r.DX>>8)*8,int(r.DX&255)*8
  if (x!=32&&x!=264)||y!=152||w<=0||h<=0||x+w>320||y+h>200{return};total++
  if len(events)>=16{return};if pending!=nil{log.Fatal("巢狀觀察貼圖")}
  a:=oracle.Addr{Seg:r.DS,Off:r.BX};size:=w*h/2;if a.Linear()>0xa0000-uint32(size){log.Fatal("來源越界")}
  raw=o.Bytes(a,size);before=o.Indexed();full:=[]string{}
  for _,p:=range poses{if p.W==w&&p.H==h&&bytes.Equal(raw,p.Packed){full=append(full,p.Key)}}
  pending=&event{Entry:o.Steps(),Regs:r,X:x,Y:y,W:w,H:h,FullBefore:matches(before,x,y,w,h),RawFull:full}
 })
 observed.OnCall(oracle.Addr{Seg:0x161,Off:0x8751},func(o *oracle.Oracle){
  if pending==nil{return};n:=len(events);prefix:=filepath.Join(*out,fmt.Sprintf("event%02d",n));after:=o.Indexed()
  pending.Return=o.Steps();pending.Before=prefix+"-before.frame";pending.After=prefix+"-after.frame";pending.Source=prefix+"-source.bin";pending.State=prefix+".state";pending.FullAfter=matches(after,pending.X,pending.Y,pending.W,pending.H)
  write(pending.Before,before);write(pending.After,after);write(pending.Source,raw);check(o.SaveStateFile(pending.State));events=append(events,*pending);pending=nil
 })
 end:=initial.Steps+uint64(len(route)-1)*4000000+12500000
 type key struct{At uint64;Scan uint8;Down bool};keys:=[]key{}
 for i,k:=range route{at:=initial.Steps+500000+uint64(i)*4000000;keys=append(keys,key{at,scans[k],true},key{at+1000000,scans[k],false})}
 trace:=[]map[string]any{};run:=func(o *oracle.Oracle,record bool){for _,k:=range keys{if o.Steps()>k.At{log.Fatal("輸入排序錯誤")};check(o.Run(k.At-o.Steps()));if record{trace=append(trace,map[string]any{"at":o.Steps(),"before_input":k,"location":location(o)})};if k.Down{o.KeyDown(k.Scan)}else{o.KeyUp(k.Scan)}};check(o.Run(end-o.Steps()))}
 run(observed,true);finish:=fp(observed);endLoc:=location(observed);write(filepath.Join(*out,"end.frame"),observed.Indexed());check(observed.SaveStateFile(filepath.Join(*out,"observed-end.state")))
 control:=create();defer control.Close();if fp(control)!=initial||control.Word(oracle.Addr{Seg:0x161,Off:0x41df})!=seed{log.Fatal("起始對照不同")};run(control,false);baseline:=fp(control);check(control.SaveStateFile(filepath.Join(*out,"control-end.state")))
 if baseline!=finish||!bytes.Equal(control.Indexed(),observed.Indexed()){log.Fatal("觀察干擾原版")}
 jsonWrite(filepath.Join(*out,"observation.json"),map[string]any{"scope":"normal Zellwal explicit direction route; original observation only; no HD or gameplay injection","go_version":runtime.Version(),"inputs_sha256":inputs,"address_space":"original runtime CS:IP/DS:BX; file offsets only in independent decoder","seed_before":fmt.Sprintf("%04X",seed),"seed_method":"same SHA state read before both runs; no reseed or reroll","key_events":keys,"route":route,"key_locations":trace,"initial":initial,"observed_end":finish,"control_end":baseline,"start_location":startLoc,"end_location":endLoc,"battle_entries":entered,"battle_returns":returned,"total_sprite_calls":total,"events":events,"pending_at_end":pending!=nil,"limits":"first16 calls at known enemy/ally slots; each independent PBL/byte/frame check required; no READY extension, art acceptance, translation, GUI or fromboot claim"})
 fmt.Printf("Zellwal來源%d/%d，戰鬥%d/%d，seed%04X，原版完整RAM/frame/regs/cycles相同\n",len(events),total,entered,returned,seed)
}
