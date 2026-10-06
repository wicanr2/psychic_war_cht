"""研究038 §100：在唯讀正式DB的副本匯出原始ALLY相關定位。"""
import hashlib
import json

import ida_auto
import ida_bytes
import ida_funcs
import ida_idaapi
import ida_nalt
import ida_pro
import ida_segment
import ida_ua
import idautils
import idc


def instruction(ea):
    segment = ida_segment.getseg(ea)
    base = ida_segment.get_segm_base(segment) if segment else None
    size = ida_bytes.get_item_size(ea)
    function = ida_funcs.get_func(ea)
    return {'ida_ea': hex(ea), 'segment': ida_segment.get_segm_name(segment) if segment else None,
            'segment_base': hex(base) if base is not None else None,
            'segment_offset': hex(ea-base) if base is not None else None,
            'bytes': (ida_bytes.get_bytes(ea, size) or b'').hex(),
            'original_disassembly': idc.GetDisasm(ea),
            'original_function': ida_funcs.get_func_name(function.start_ea) if function else None,
            'semantics_level': 'unknown'}


def references(ea):
    return [{'from': instruction(x.frm), 'type': x.type, 'is_code': bool(x.iscode)} for x in idautils.XrefsTo(ea)]


def function(ea):
    f = ida_funcs.get_func(ea)
    if f is None:
        return {'query': hex(ea), 'found': False}
    return {'original_name': ida_funcs.get_func_name(f.start_ea),
            'start': hex(f.start_ea), 'end': hex(f.end_ea),
            'instructions': [instruction(p) for p in idautils.Heads(f.start_ea, f.end_ea)],
            'references': references(f.start_ea)}


def main():
    ida_auto.auto_wait()
    mode = idc.ARGV[2] if len(idc.ARGV) > 2 else 'probe'
    source = '/src/workplace/ida/PW_UNP.EXE'
    with open(source, 'rb') as f:
        input_sha = hashlib.sha256(f.read()).hexdigest()
    assert input_sha == 'fd5115b91f014c47a1fb1a6e2ecfd645e293264fe614a4b350e92637cad2fbd9'
    result = {'schema': 'psychic-war-ida-ally-portrait/1', 'mode': mode,
              'tool_version': ida_pro.IDA_SDK_VERSION,
              'kernel_version': idaapi_kernel_version(),
              'input_file': ida_nalt.get_input_file_path(), 'input_sha256': input_sha,
              'address_space': 'IDA linear ea; segment base retained per row, not runtime linear address',
              'function_count': len(list(idautils.Functions())),
              'limits': 'Static instructions and xrefs only; no gameplay identity or source usage inferred.'}
    if mode == 'query':
        base = 0x10510
        table = 0x1966b
        result['pbl_table'] = []
        for i in range(26):
            pointer = ida_bytes.get_word(table+i*2)
            descriptor = base+pointer
            raw = ida_bytes.get_bytes(descriptor, 24)
            result['pbl_table'].append({'index': i, 'table_ea': hex(table+i*2),
                                        'original_pointer': hex(pointer), 'descriptor_ea': hex(descriptor),
                                        'descriptor_bytes': raw.hex(), 'filename': raw[1:].split(b'\0')[0].decode('ascii','replace'),
                                        'semantics_level': 'unknown'})
        result['functions'] = [function(ea) for ea in (0x189bf, 0x18a7c, 0x18b76)]
        immediates = []
        wanted = {0xc00, 0xc01, 0xc10, 0xc18, 0xc1e}
        for f in idautils.Functions():
            function_end = ida_funcs.get_func(f).end_ea
            for ea in idautils.Heads(f, function_end):
                insn = ida_ua.insn_t()
                if not ida_ua.decode_insn(insn, ea):
                    continue
                for op in insn.ops:
                    if op.type == ida_ua.o_imm and op.value in wanted:
                        immediates.append(instruction(ea))
                        break
        result['candidate_immediates'] = immediates
    elif mode.startswith('func:'):
        result['functions'] = [function(int(ea,16)) for ea in mode[5:].split(',')]
    elif mode.startswith('span:'):
        start, end = (int(ea,16) for ea in mode[5:].split(':'))
        assert 0 < end-start <= 256
        result['span_bytes'] = ida_bytes.get_bytes(start,end-start).hex()
        result['decoded_instructions'] = []
        ea = start
        while ea < end:
            insn = ida_ua.insn_t()
            size = ida_ua.decode_insn(insn,ea)
            assert size > 0 and ea+size <= end
            row = instruction(ea)
            row['bytes'] = ida_bytes.get_bytes(ea,size).hex()
            row['decoded_mnemonic'] = insn.get_canon_mnem()
            row['decoded_operands'] = [
                {'type':op.type,'reg':op.reg,'addr':hex(op.addr),'value':hex(op.value)}
                for op in insn.ops if op.type != ida_ua.o_void]
            result['decoded_instructions'].append(row)
            ea += size
    with open(idc.ARGV[1], 'w', encoding='utf-8') as f:
        json.dump(result, f, ensure_ascii=False, indent=2)
        f.write('\n')
    ida_pro.qexit(0)


def idaapi_kernel_version():
    import ida_kernwin
    return ida_kernwin.get_kernel_version()


main()
