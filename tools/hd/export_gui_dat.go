// 研究038 §74：正式視窗F10狀態零步匯出，保留原始玩家bytes。
package main

import (
 "crypto/sha256"
 "encoding/json"
 "flag"
 "fmt"
 "log"
 "os"
 "path/filepath"
 "github.com/wicanr2/dosgolem/oracle"
)
func check(e error) { if e!=nil {log.Fatal(e)} }
func hash(b []byte) string {return fmt.Sprintf("%x",sha256.Sum256(b))}
func read(p string) []byte {b,e:=os.ReadFile(p);check(e);return b}
func write(p string,b []byte) {f,e:=os.OpenFile(p,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0644);check(e);_,e=f.Write(b);check(e);check(f.Close())}
func main(){
 out:=flag.String("out","","真正GUI輸出目錄");flag.Parse();if *out=="" {log.Fatal("缺輸出")}
 paths,e:=filepath.Glob(filepath.Join(*out,"*.state"));check(e)
 inputs:=map[string]string{};rows:=[]map[string]any{}
 for _,p:=range []string{"tools/hd/export_gui_dat.go","/orig/psychic-war/PW.EXE"}{inputs[p]=hash(read(p))}
 for _,p:=range paths{
  inputs[p]=hash(read(p));o,e:=oracle.Load("/orig/psychic-war/PW.EXE","/orig/psychic-war");check(e);check(o.LoadStateFile(p))
  w,h,rgb:=o.ScreenRGB();if w!=320||h!=200{log.Fatal("原版尺寸不符")}
  idx:=o.Indexed();player:=o.Bytes(oracle.Addr{Seg:0x1696,Off:6},52)
  write(p+".frame",idx);write(p+".rgb.bin",rgb);write(p+".player.bin",player)
  rows=append(rows,map[string]any{"state":p,"regs":o.Regs(),"steps":o.Steps(),"cycles":o.Cycles(),"area":o.Word(oracle.Addr{Seg:0x1696,Off:6}),"x":o.Word(oracle.Addr{Seg:0x1696,Off:8}),"y":o.Word(oracle.Addr{Seg:0x1696,Off:10}),"direction":o.Word(oracle.Addr{Seg:0x1696,Off:16}),"hp":o.Word(oracle.Addr{Seg:0x1699,Off:0}),"energy":o.Word(oracle.Addr{Seg:0x1699,Off:4}),"frame_sha256":hash(idx),"rgb_sha256":hash(rgb),"player_sha256":hash(player)})
  o.Close()
 }
 if len(rows)==0{log.Fatal("沒有F10狀態")}
 b,e:=json.MarshalIndent(map[string]any{"scope":"真正GUI F10狀態的原版零步匯出，52 bytes玩家區域線性16966","inputs_sha256":inputs,"results":rows,"limits":"PNG與state不保證同一幀；不比較GUI亂數／自然時序或整份RAM"},"","  ");check(e)
 write(filepath.Join(*out,"original-frames.json"),append(b,'\n'));fmt.Println("原版GUI狀態零步匯出",len(rows))
}
