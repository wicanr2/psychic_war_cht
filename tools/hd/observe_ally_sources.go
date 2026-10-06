// 研究038 §100：正常開局及選單的唯讀ALLY來源觀察。
// 以Go覆映射建置於dosgolem的cmd/probe，不修改正式執行器。
package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"reflect"
	"strings"

	"github.com/wicanr2/dosgolem/internal/cpu"
	"github.com/wicanr2/dosgolem/internal/dos"
	"github.com/wicanr2/dosgolem/internal/machine"
	"github.com/wicanr2/dosgolem/internal/state"
	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
)

func check(err error) { if err != nil { log.Fatal(err) } }
func read(path string) []byte { b,err:=os.ReadFile(path);check(err);return b }
func hash(b []byte) string { return fmt.Sprintf("%x",sha256.Sum256(b)) }
func write(path string,b []byte) { f,err:=os.OpenFile(path,os.O_WRONLY|os.O_CREATE|os.O_EXCL,0644);check(err);_,err=f.Write(b);check(err);check(f.Close()) }
func jsonWrite(path string,v any) { b,err:=json.MarshalIndent(v,"","  ");check(err);write(path,append(b,'\n')) }
func saved(path string) (machineSaved,[]byte) {
	z,err:=gzip.NewReader(bytes.NewReader(read(path)));check(err);defer z.Close()
	section:=func() []byte { var n uint64;check(binary.Read(z,binary.LittleEndian,&n));if n>32*1024*1024 {log.Fatal("保存區段越界")};b:=make([]byte,n);_,e:=io.ReadFull(z,b);check(e);return b }
	mb:=section();db:=section();tail,e:=io.ReadAll(z);check(e);if len(tail)>0 {log.Fatal("未解析尾端")}
	var s machineSaved;check(gob.NewDecoder(bytes.NewReader(mb)).Decode(&s));return s,db
}
type fingerprint struct {
	R [8]uint16
	Seg [4]uint16
	IP,Flags uint16
	Steps,Cycles,IRQ1 uint64
	RAM,Bus,Frame string
}
func fp(m *machine.Machine) fingerprint { c:=m.CPU;return fingerprint{c.R,c.Seg,c.IP,c.Flags,m.Steps,c.Cycles,m.IRQ1Delivered(),hash(m.Mem[:0xa0000]),hash(m.Snapshot().Mem()),hash(m.Indexed())} }
type context struct {
	Step uint64
	CS,IP,AX,BX,CX,DX,DS,ES,SS,SP,BP uint16
	Stack []uint16
}
func ctx(m *machine.Machine) context {
	c:=m.CPU;v:=context{Step:m.Steps,CS:c.Seg[cpu.CS],IP:c.IP,AX:c.R[cpu.AX],BX:c.R[cpu.BX],CX:c.R[cpu.CX],DX:c.R[cpu.DX],DS:c.Seg[cpu.DS],ES:c.Seg[cpu.ES],SS:c.Seg[cpu.SS],SP:c.R[cpu.SP],BP:c.R[cpu.BP]}
	for i:=0;i<16;i++ { a:=uint32(v.SS)*16+uint32(uint16(v.SP+uint16(i*2)));if a+2>0xa0000 { log.Fatal("堆疊越界") };v.Stack=append(v.Stack,m.Read16(a)) };return v
}
type source struct { Index,W,H int; Pixels []byte }
type draw struct {
	Context context
	X,Y,W,H int
	Matches []int
	PackedSHA256,File string
}
func main() {
	out:=flag.String("out","","全新輸出前綴")
	start:=flag.String("start","workplace/states/06-name.state","原版正常狀態")
	end:=flag.Uint64("end",42000000,"絕對終點")
	keyAt:=flag.Uint64("key-at",35500000,"第一個FIFO鍵時機")
	keyEvery:=flag.Uint64("key-every",500000,"鍵距")
	keySpec:=flag.String("keys","25,1e,17,1c","逗號分隔兩位十六進位掃描碼")
	flag.Parse();if *out=="" { log.Fatal("缺輸出") };old,err:=filepath.Glob(*out+"*");check(err);if len(old)>0 {log.Fatal("拒絕覆寫")}
	const orig="/orig/psychic-war"
	inputs:=map[string]string{}
	for _,p:=range []string{*start,"tools/hd/observe_ally_sources.go",orig+"/PW.EXE",orig+"/ALLY.PBL"} {inputs[p]=hash(read(p))}
	if inputs[orig+"/PW.EXE"]!="88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49" || inputs[orig+"/ALLY.PBL"]!="c88ad34c06d3c5ab68ff2b4aafe2a0fde51b513c375c1d90924563f1b1557219" {log.Fatal("原版SHA不符")}
	data:=read(orig+"/ALLY.PBL");sources:=[]source{}
	for i:=0;i<31;i++ {w,h,px,e:=pbl.Decode(data,i);check(e);sources=append(sources,source{i,w,h,px})}
	keys:=[]uint8{}
	for _,k:=range strings.Split(*keySpec,",") { var scan uint8;n,e:=fmt.Sscanf(k,"%x",&scan);check(e);if n!=1||len(k)!=2 {log.Fatal("掃描碼格式")};keys=append(keys,scan) }
	newMachine:=func() (*machine.Machine,*dos.DOS) {m:=machine.New();d:=dos.New(m,orig);d.Install();m.KeyEvery=*keyEvery;m.SetNextKey(*keyAt);for _,k:=range keys {m.QueueKey(k)};check(state.Load(*start,m,d));d.Root=orig;d.Scratch="/tmp/pw-ally-source";return m,d}
	m,d:=newMachine();defer d.Close();if *end<=m.Steps {log.Fatal("終點未在起點之後")}
	initial:=fp(m);seed:=m.Read16(0x1610+0x41df)
	loads:=[]context{};opens:=[]context{};draws:=[]draw{}
	d.OnOpen=func(name string) {if strings.EqualFold(name,"ally.pbl") {if len(opens)>=128 {log.Fatal("開檔超出界限")};opens=append(opens,ctx(m))}}
	for m.Steps<*end {
		c:=m.CPU
		if c.Seg[cpu.CS]==0x161 {
			if c.IP==0x8666 && c.R[cpu.AX]>>8==12 {if len(loads)>=128 {log.Fatal("載入超出界限")};loads=append(loads,ctx(m))}
			if c.IP==0x8705 {
				w,h:=int(c.R[cpu.DX]>>8)*8,int(c.R[cpu.DX]&255)*8
				if w*h>0&&w*h<=64000 {
					a:=uint32(c.Seg[cpu.DS])*16+uint32(c.R[cpu.BX]);n:=uint32(w*h/2);if a+n>0xa0000 {log.Fatal("來源越界")}
					raw:=append([]byte(nil),m.Mem[a:a+n]...);px:=make([]byte,w*h);for i,b:=range raw {px[2*i]=b>>4;px[2*i+1]=b&15}
					matches:=[]int{};for _,s:=range sources {if s.W==w&&s.H==h&&bytes.Equal(px,s.Pixels) {matches=append(matches,s.Index)}}
					if len(matches)>0 {if len(draws)>=128 {log.Fatal("貼圖超出界限")};path:=fmt.Sprintf("%s-source%03d.bin",*out,len(draws));write(path,raw);draws=append(draws,draw{ctx(m),int(c.R[cpu.CX]>>8)*4,int(c.R[cpu.CX]&255)*4,w,h,matches,hash(raw),path})}
				}
			}
		}
		check(m.Step());if d.Exited||c.Halted {log.Fatal("原版提早終止")}
	}
	observed:=fp(m);write(*out+"-end.frame",m.Indexed());check(state.Save(*out+"-observed.state",m,d))
	control,cd:=newMachine();defer cd.Close();if fp(control)!=initial||control.Read16(0x1610+0x41df)!=seed {log.Fatal("控制起點不同")}
	for control.Steps<*end {check(control.Step())};if fp(control)!=observed {log.Fatal("觀察干擾機器狀態")}
	check(state.Save(*out+"-control.state",control,cd));sm,sd:=saved(*out+"-observed.state");cm,cs:=saved(*out+"-control.state");if !reflect.DeepEqual(sm,cm)||!bytes.Equal(sd,cs) {log.Fatal("全部機器欄位或DOS區段不同")}
	bad:=cm;bad.R[0]^=1;if reflect.DeepEqual(sm,bad) {log.Fatal("完整機器暫存器負對照未檢出")};bad=cm;bad.Mem=append([]byte(nil),cm.Mem...);bad.Mem[0]^=1;if reflect.DeepEqual(sm,bad) {log.Fatal("完整機器RAM負對照未檢出")};bad=cm;bad.Ports=map[uint16]uint8{};for k,v:=range cm.Ports {bad.Ports[k]=v};bad.Ports[0x3ce]^=1;if reflect.DeepEqual(sm,bad) {log.Fatal("完整機器port負對照未檢出")}
	jsonWrite(*out+"-machine.json",sm)
	negative:=observed;negative.R[0]^=1;if negative==fp(control) {log.Fatal("負對照未檢出")}
	if observed.IRQ1!=uint64(len(keys)*2) {log.Fatal("正常鍵事件數不同")}
	jsonWrite(*out+".json",map[string]any{"schema":"psychic-war-ally-source/1","go_version":runtime.Version(),"inputs_sha256":inputs,"start":*start,"end_step":*end,"key_at":*keyAt,"key_every":*keyEvery,"keys_scan":keys,"seed_before":fmt.Sprintf("%04X",seed),"seed_method":"state固定，未修改或重擲","address_space":"原版runtime CS:IP與DS:BX；堆疊SS:SP，非IDA ea；座標320×200","initial":initial,"observed_end":observed,"control_end":fp(control),"full_saved_state_equal":true,"register_negative_control":true,"ally_loads":loads,"ally_opens":opens,"ally_draws":draws,"dos_reads":d.Reads,"limits":"僅本鍵序來源觀察；未匹配到圖號不證明該圖未使用。未驗HD、中文、GUI或全部ALLY。"})
	fmt.Printf("ALLY：%d載入、%d開檔、%d原版來源貼圖；完整控制state相同，seed%04X\n",len(loads),len(opens),len(draws),seed)
}
