"""研究038 §71：完整比較原版state，機器gob的map排序不當作遊戲變化。"""
import hashlib
import json
from pathlib import Path
import subprocess


def compare_saved(root, out, baseline, observed, inputs):
    prefix=out/'maze-saved-state-independent-v1-20261003'
    source=prefix.with_suffix('.go')
    binary=prefix.with_suffix('.bin')
    report=prefix.with_suffix('.json')
    log=prefix.with_suffix('.log')
    assert not list(out.glob(prefix.name+'*')), '拒絕覆寫獨立state核對'
    state_source=root/'worktrees/dosgolem/internal/machine/state.go'
    text=state_source.read_text()
    start=text.index('type machineState struct {')
    end=text.index('\n}',start)+2
    struct=text[start:end]
    go='''package main
import("bytes";"compress/gzip";"crypto/sha256";"encoding/binary";"encoding/gob";"encoding/json";"fmt";"io";"os";"reflect")
'''+struct+'''
func check(e error){if e!=nil{panic(e)}}
func hash(b []byte)string{return fmt.Sprintf("%x",sha256.Sum256(b))}
func load(p string)(machineState,[]byte,[]byte){
 b,e:=os.ReadFile(p);check(e); z,e:=gzip.NewReader(bytes.NewReader(b));check(e)
 raw,e:=io.ReadAll(z);check(e);check(z.Close());r:=bytes.NewReader(raw)
 var n uint64;check(binary.Read(r,binary.LittleEndian,&n));if n>uint64(r.Len()){panic("機器區段越界")}
 m:=make([]byte,n);_,e=io.ReadFull(r,m);check(e);var s machineState;check(gob.NewDecoder(bytes.NewReader(m)).Decode(&s))
 check(binary.Read(r,binary.LittleEndian,&n));if n!=uint64(r.Len()){panic("DOS區段或尾端不同")}
 d:=make([]byte,n);_,e=io.ReadFull(r,d);check(e);return s,m,d
}
func main(){
 a,am,ad:=load(os.Args[1]);b,bm,bd:=load(os.Args[2]);if !reflect.DeepEqual(a,b){panic("完整機器欄位不同")}
 if !bytes.Equal(ad,bd){panic("完整DOS gob bytes不同")}
 aj,e:=json.Marshal(a);check(e);bj,e:=json.Marshal(b);check(e);if !bytes.Equal(aj,bj){panic("完整JSON不同")}
 typ:=reflect.TypeOf(a);fields:=[]string{};for i:=0;i<typ.NumField();i++{fields=append(fields,typ.Field(i).Name)}
 if len(fields)<40{panic("欄位核對範圍不足")}
 originalR:=a.R[0];a.R[0]^=1;negativeCPU:=!reflect.DeepEqual(a,b);a.R[0]=originalR
 originalByte:=a.Mem[0];a.Mem[0]^=1;negativeRAM:=!reflect.DeepEqual(a,b);a.Mem[0]=originalByte
 var port uint16;for p:=range a.Ports{port=p;break};v:=a.Ports[port];a.Ports[port]^=1;negativePort:=!reflect.DeepEqual(a,b);a.Ports[port]=v
 if !negativeCPU||!negativeRAM||!negativePort{panic("負對照無效")}
 doc:=map[string]any{"scope":"原版state全部機器欄位及完整DOS區段","fields":fields,"field_count":len(fields),"canonical_machine_sha256":hash(aj),"dos_gob_sha256":hash(ad),"machine_fields_equal":true,"dos_gob_equal":true,"machine_gob_bytes_equal":bytes.Equal(am,bm),"negative_cpu_changed":negativeCPU,"negative_ram_changed":negativeRAM,"negative_port_changed":negativePort,"step":a.Steps,"cycles":a.Cycles}
 out,e:=json.MarshalIndent(doc,"","  ");check(e);f,e:=os.OpenFile(os.Args[3],os.O_CREATE|os.O_EXCL|os.O_WRONLY,0644);check(e);_,e=f.Write(append(out,10));check(e);check(f.Close());fmt.Println("完整state欄位核對",len(fields),"通過")
}
'''
    with source.open('x') as f:
        f.write(go)
    commands=[['go','build','-buildvcs=false','-o',str(binary),str(source)],
              [str(binary),str(baseline),str(observed),str(report)]]
    with log.open('xb') as f:
        for argv in commands:
            process=subprocess.run(argv,stdout=f,stderr=subprocess.STDOUT,timeout=90)
            assert process.returncode==0,argv
    for path in [Path(__file__),state_source,source,binary,report,log]:
        assert path.stat().st_uid==path.stat().st_gid==1000
        inputs[str(path.relative_to(root))]=hashlib.sha256(path.read_bytes()).hexdigest()
    result=json.loads(report.read_text())
    assert result['machine_fields_equal'] and result['dos_gob_equal']
    return {'receipt':str(report.relative_to(root)),'argv':commands,'result':result}
