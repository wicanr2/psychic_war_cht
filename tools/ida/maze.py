# IDA 9.4 唯讀匯出；入口 docs/re/038 §72。
# idat -A '-S/tools/maze.py /work/probe.json probe' <一次性DB>
import hashlib
import json
import os
import sys

import ida_auto
import ida_bytes
import ida_funcs
import ida_loader
import ida_pro
import ida_segment
import ida_ua
import idaapi
import idautils
import idc


def sha(path):
    with open(path, "rb") as stream:
        return hashlib.sha256(stream.read()).hexdigest()


def location(ea):
    seg = ida_segment.getseg(ea)
    return {
        "ida_ea": ea,
        "segment": ida_segment.get_segm_name(seg) if seg else None,
        "segment_base": ida_segment.get_segm_base(seg) if seg else None,
        "segment_offset": ea - ida_segment.get_segm_base(seg) if seg else None,
    }


def instruction(ea):
    value = location(ea)
    size = ida_bytes.get_item_size(ea)
    value.update(size=size, bytes=ida_bytes.get_bytes(ea, size).hex(),
                 disasm=" ".join(idc.GetDisasm(ea).split()))
    return value


def function(ea):
    f = ida_funcs.get_func(ea)
    if f is None:
        return {"query_ea": ea, "function": None}
    return {
        "query_ea": ea, "original_name": ida_funcs.get_func_name(f.start_ea),
        "start": location(f.start_ea), "end_ea": f.end_ea,
        "instructions": [instruction(h) for h in idautils.FuncItems(f.start_ea)],
        "xrefs_to_start": [dict(source=location(x.frm), type=x.type,
                                 is_code=bool(x.iscode))
                            for x in idautils.XrefsTo(f.start_ea)],
    }


def main():
    ida_auto.auto_wait()
    out, mode = idc.ARGV[1:3]
    result = {
        "schema": "psychic-war-ida-maze-evidence/1", "mode": mode,
        "ida_version": idaapi.get_kernel_version(), "python_version": sys.version,
        "address_space": "IDA linear ea and IDA segment offset, not DOS runtime linear addresses",
        "input_binary_sha256": sha("/input/PW_UNP.EXE"),
        "input_database_sha256": sha("/input/PW_UNP.EXE.i64"),
        "function_count": len(list(idautils.Functions())),
        "segments": [dict(location(s), end_ea=ida_segment.getseg(s).end_ea)
                     for s in idautils.Segments()],
    }
    if mode == "decode":
        lo, hi = [int(s, 16) for s in idc.ARGV[3].split("-")]
        result["original_region_bytes"] = ida_bytes.get_bytes(lo, hi - lo).hex()
        result["original_region_function"] = function(lo)
        result["region_start"] = location(lo)
        result["region_end_ea"] = hi
        # 正式DB沒有這段的函式邊界。僅在一次性副本建立指令，保留原bytes。
        ida_bytes.del_items(lo, ida_bytes.DELIT_SIMPLE, hi - lo)
        ea = lo
        items = []
        while ea < hi:
            size = ida_ua.create_insn(ea)
            if not size:
                raise ValueError("instruction decode failed at %X" % ea)
            items.append(instruction(ea))
            ea += size
        result["decoded_instructions"] = items
        result["limits"] = "instruction decoding on disposable database; no claimed original IDA function boundary"
    elif mode == "draw":
        queries = []
        for s in idc.ARGV[3:]:
            if s.startswith("file:"):
                offset = int(s[5:], 16)
                ea = ida_loader.get_fileregion_ea(offset)
                queries.append({"file_offset": offset, "ida_ea": ea})
            else:
                ea = int(s, 16)
                queries.append({"ida_ea": ea})
        result["queries"] = queries
        result["functions"] = [function(q["ida_ea"]) for q in queries]
        result["near_queries"] = [dict(query=q, instructions=[instruction(h) for h in
                                     idautils.Heads(q["ida_ea"] - 32, q["ida_ea"] + 48)])
                                  for q in queries]
    elif mode != "probe":
        raise ValueError("unknown export mode")
    with open(out, "x", encoding="utf-8") as stream:
        json.dump(result, stream, ensure_ascii=False, indent=2)
        stream.write("\n")
    ida_pro.qexit(0)


main()
