# IDAPython：一次處理多個查詢，結果寫成一份文字檔（避免對同一個 .i64 連續開關）。
#
#   tools/ida.sh script dump.py PW_UNP.EXE.i64 /work/out.txt <查詢…>
#
# 查詢：
#   xref:<起>-<迄>     列出這段位址每個 byte 的 data／code xref（引用點、所屬函式、反組譯）
#   func:<ea>          反組譯 ea 所在的整個函式（位址、bytes、反組譯）
#   imm:<值>           掃全部指令，列出 operand 立即數或位移等於此值的指令
#   callers:<ea>       列出呼叫 ea 的指令
#   refs:<ea>          只列引用端的位址與 xref 型別（不反組譯引用端）
#   dis:<起>-<迄>      反組譯一段位址（不管有沒有函式；func: 會讓 idat 異常結束的位址用這個）
#
# ⚠ 已知：對 PW_UNP.EXE.i64 的 14CE4、141B2 下 callers:／xref: 會讓 idat 異常結束
#   （rc=1、輸出 0 bytes、scratch 殘留 .id0），可穩定重現。先用 refs: 分辨是
#   列舉 xref 還是反組譯引用端出錯。
#
# 位址都是 IDA ea（16 進位）。⚠ headless 的 print 不進 stdout；唯一可信的訊號是輸出檔。
import ida_auto
import ida_bytes
import ida_funcs
import ida_pro
import ida_segment
import ida_ua
import idautils
import idc


def line(ea):
    seg = ida_segment.getseg(ea)
    so = "%s:%04X" % (ida_segment.get_segm_name(seg), ea - ida_segment.get_segm_base(seg)) if seg else "?"
    size = max(1, ida_bytes.get_item_size(ea))
    return "%05X %-13s %-24s %s" % (ea, so, ida_bytes.get_bytes(ea, min(size, 12)).hex(),
                                    " ".join(idc.GetDisasm(ea).split()))


def fname(ea):
    f = ida_funcs.get_func(ea)
    return ida_funcs.get_func_name(f.start_ea) if f else "—"


def q_xref(arg, out):
    lo, hi = [int(x, 16) for x in arg.split("-")]
    out.append("## xref %05X-%05X" % (lo, hi))
    for ea in range(lo, hi + 1):
        for x in idautils.XrefsTo(ea):
            out.append("  -> %05X  from %s  [%s] type=%d" % (ea, line(x.frm), fname(x.frm), x.type))


def q_func(arg, out):
    ea = int(arg, 16)
    f = ida_funcs.get_func(ea)
    if not f:
        out.append("## func %05X：不在函式內" % ea)
        return
    out.append("## func %s %05X-%05X" % (ida_funcs.get_func_name(f.start_ea), f.start_ea, f.end_ea))
    for h in idautils.Heads(f.start_ea, f.end_ea):
        out.append("  " + line(h))


def q_imm(arg, out):
    val = int(arg, 16)
    out.append("## imm %X" % val)
    for s in idautils.Segments():
        seg = ida_segment.getseg(s)
        for h in idautils.Heads(s, seg.end_ea):
            if not ida_bytes.is_code(ida_bytes.get_flags(h)):
                continue
            insn = ida_ua.insn_t()
            if ida_ua.decode_insn(insn, h) == 0:
                continue
            for op in insn.ops:
                if op.type == ida_ua.o_void:
                    break
                if (op.type == ida_ua.o_imm and op.value == val) or \
                   (op.type in (ida_ua.o_mem, ida_ua.o_displ) and op.addr == val):
                    out.append("  %s  [%s]" % (line(h), fname(h)))
                    break


def q_callers(arg, out):
    ea = int(arg, 16)
    out.append("## callers %05X" % ea)
    for x in idautils.CodeRefsTo(ea, 0):
        out.append("  %s  [%s]" % (line(x), fname(x)))


def q_refs(arg, out):
    ea = int(arg, 16)
    out.append("## refs %05X" % ea)
    for x in idautils.XrefsTo(ea):
        out.append("  %05X type=%d iscode=%d" % (x.frm, x.type, x.iscode))


def q_dis(arg, out):
    lo, hi = [int(x, 16) for x in arg.split("-")]
    out.append("## dis %05X-%05X" % (lo, hi))
    ea = lo
    while ea < hi:
        out.append("  %s  [%s]" % (line(ea), fname(ea)))
        ea = idc.next_head(ea, hi + 1)
        if ea == idc.BADADDR:
            break


def main():
    ida_auto.auto_wait()
    out_path = idc.ARGV[1]
    out = []
    handlers = {"xref": q_xref, "func": q_func, "imm": q_imm, "callers": q_callers, "refs": q_refs, "dis": q_dis}
    for q in idc.ARGV[2:]:
        kind, _, arg = q.partition(":")
        try:
            handlers[kind](arg, out)
        except Exception as e:  # noqa: BLE001 — 一個查詢壞掉不要拖垮整批
            out.append("## %s 失敗：%r" % (q, e))
    with open(out_path, "w") as f:
        f.write("\n".join(out) + "\n")
    ida_pro.qexit(0)


main()
