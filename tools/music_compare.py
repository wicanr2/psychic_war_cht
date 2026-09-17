#!/usr/bin/env python3
"""music_compare.py — PC 喇叭配樂比對（docs/spec/001）。

    tools/py.sh tools/music_compare.py events <dosgolem 埠紀錄.tsv> <ibm 目錄> [--json out.json]
    tools/py.sh tools/music_compare.py audio  <dosgolem.wav> <dosboxx.wav> [--json out.json]
    tools/py.sh tools/music_compare.py opl    <dosgolem opl-log.txt> <dosgolem.wav> <dosboxx.wav> [--json out.json]

events（§2 A 層）：probe `-dump-ports "42,43,61=…"` 的紀錄解成音符事件，與 OPEN0.IBM、OPEN1.IBM…
依序串接的記錄逐筆比對（頻率、刻數）。
audio（§2 B 層）：兩個 WAV 用同一支分析器切成音符段，對齊後比音高與節奏（§3、§4）。
opl：OPL2（AdLib）複音配樂，以暫存器序列為樂譜檢查兩份 WAV 的起音與音高（docs/spec/002）。
"""
import array
import bisect
import json
import math
import struct
import sys
from pathlib import Path

PIT_HZ = 315e6 / 264
TICK_S = 16571 / PIT_HZ  # 遊戲把通道 0 設成 16571 分頻


# ---------------------------------------------------------------- A 層：事件

DRIVER_PIT = 0x1234DC  # 驅動換算分頻值用的常數 1193180，捨去小數（docs/re/006 §3）


def expected_divisor(hz):
    return DRIVER_PIT // hz if hz else 0


def port_events(tsv):
    """回 [(步數, 分頻值或 0)]：每次 61h 寫入時的發聲狀態。與 dosgolem ToneEvents 同一套規則。

    比分頻值不比 Hz：驅動用整數除法算分頻值，反算回 Hz 再四捨五入會差 1（988 → 989）。"""
    rows = [line.rstrip("\n").split("\t") for line in Path(tsv).read_text().splitlines()[1:]]
    out, lo, low_next, div, div_set = [], 0, True, 0, False
    for step, port, val in rows:
        s, v = int(step), int(val, 16)
        if port == "043":
            if v >> 6 & 3 == 2 and v >> 4 & 3:
                low_next = True
        elif port == "042":
            if low_next:
                lo, low_next = v, False
            else:
                div, div_set, low_next = lo | v << 8, True, True
        elif port == "061":
            on = v & 3 == 3 and div_set
            out.append((s, (div or 65536) if on else 0))
    return out


def ibm_records(path):
    b = Path(path).read_bytes()
    if len(b) % 3:
        raise SystemExit(f"{path}：{len(b)} bytes 不是 3 的倍數")
    return [(struct.unpack_from("<H", b, i)[0], b[i + 2]) for i in range(0, len(b), 3)]


def cmd_events(tsv, ibm_dir, out_json):
    ev = port_events(tsv)
    if not ev:
        raise SystemExit("埠紀錄裡沒有 61h 寫入")
    tick_steps = None
    files = sorted(Path(ibm_dir).glob("OPEN[0-9].IBM"))
    expected = []  # (累計刻數, 頻率, 檔名, 第幾筆)
    t = 0
    for f in files:
        for k, (hz, dur) in enumerate(ibm_records(f)):
            expected.append((t, hz, f.name, k))
            t += dur
    # 每個換檔點，驅動會多寫一次 61h ← 00（docs/re/006 §3），先把它濾掉：
    # 規則是「關閉事件後，同一刻又有新的發聲事件」且預期序列在此沒有休止記錄。
    first = ev[0][0]
    # 一刻幾道指令：初值取 dosgolem 的時間模型（165,000 道對應分頻 17,000，換算到 16,571），
    # 再用所有事件間隔最小平方微調。不用「間隔中位數 ÷ 某個刻數」：曲子裡哪種音長最多因曲而異。
    t0 = 165000 * 16571 / 17000
    gaps = [ev[i + 1][0] - ev[i][0] for i in range(1, len(ev) - 1)]  # 第一個音從一刻中途開始，不用
    n_ticks = [round(g / t0) for g in gaps]
    tick_steps = sum(g for g, n in zip(gaps, n_ticks) if n) / sum(n for n in n_ticks if n)
    got = []
    for i, (s, hz) in enumerate(ev):
        tk = (s - first) / tick_steps
        if hz == 0 and i + 1 < len(ev) and abs(ev[i + 1][0] - s) < tick_steps * 0.1:
            continue  # 換檔時的額外關閉
        got.append((tk, hz))
    while got and got[-1][1] == 0:
        got.pop()  # 曲子播完時的關閉，不是一筆休止記錄
    n = min(len(got), len(expected))
    unplayed = sorted({e[2] for e in expected[n:]})
    # 第一個音從一刻的中途開始：以第二筆對齊時間
    offset = expected[1][0] - got[1][0] if n > 1 else 0
    mism = []
    for i in range(n):
        tk, dv = got[i]
        et, ehz, fname, k = expected[i]
        if dv != expected_divisor(ehz) or (i > 0 and abs(tk + offset - et) > 0.25):
            mism.append({"i": i, "file": fname, "record": k, "expected_tick": et, "expected_hz": ehz,
                         "expected_divisor": expected_divisor(ehz), "got_tick": round(tk + offset, 2),
                         "got_divisor": dv})
    covered_files = sorted({expected[i][2] for i in range(n)})
    partial = [f for f in covered_files if f in unplayed]
    result = {"layer": "events", "port_events": len(ev), "compared": n, "expected_records": len(expected),
              "mismatches": len(mism), "first_mismatches": mism[:20], "ticks_per_step_estimate": tick_steps,
              "files_covered": covered_files, "files_not_fully_played": unplayed}
    print(f"事件層：比對 {n} 筆（預期共 {len(expected)} 筆，涵蓋 {', '.join(covered_files)}），不一致 {len(mism)}")
    if unplayed:
        print(f"  紀錄結束時還沒播到（或沒播完）的檔：{', '.join(unplayed)}"
              + (f"；其中 {', '.join(partial)} 只播了一部分" if partial else ""))
    for m in mism[:5]:
        print("  ", m)
    if out_json:
        Path(out_json).write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    return 1 if mism else 0


# ---------------------------------------------------------------- B 層：聲音

def read_wav(path):
    """回 (去直流的單聲道浮點取樣, 取樣率)。用 array 讀，長的立體聲錄音才不會吃光記憶體。"""
    b = Path(path).read_bytes()
    if b[:4] != b"RIFF" or b[8:12] != b"WAVE":
        raise SystemExit(f"{path}：不是 WAV")
    i, fmt, data = 12, None, None
    while i + 8 <= len(b):
        cid, n = b[i:i + 4], struct.unpack_from("<I", b, i + 4)[0]
        if cid == b"fmt ":
            fmt = struct.unpack_from("<HHIIHH", b, i + 8)
        elif cid == b"data":
            data = b[i + 8:i + 8 + n]
        i += 8 + n + (n & 1)
    tag, ch, rate, _, _, bits = fmt
    if tag != 1 or bits not in (8, 16):
        raise SystemExit(f"{path}：只支援 PCM 8／16 位元（format={tag}、bits={bits}）")
    if bits == 8:
        raw = array.array("B", data)
        centre = 128
    else:
        raw = array.array("h")
        raw.frombytes(data[:len(data) // 2 * 2])
        centre = 0
    if ch == 1:
        s = [x - centre for x in raw]
    else:
        chans = [raw[k::ch] for k in range(ch)]
        s = [(sum(v) / ch) - centre for v in zip(*chans)]
    mean = sum(s) / len(s)
    return [x - mean for x in s], rate


def segments(samples, rate):
    peak = max(abs(x) for x in samples) or 1
    h = peak * 0.10
    armed = False
    cross = []
    for i in range(1, len(samples)):
        x = samples[i]
        if x < -h:
            armed = True
        elif armed and x > h:
            # 往回找過零點，線性內插
            j = i
            while j > 0 and samples[j - 1] > 0:
                j -= 1
            a, b2 = samples[j - 1], samples[j]
            frac = (-a / (b2 - a)) if b2 != a else 0.0
            cross.append((j - 1 + frac) / rate)
            armed = False
    segs = []
    cur = [cross[0]] if cross else []
    for t in cross[1:]:
        p = t - cur[-1]
        if len(cur) >= 2:
            periods = sorted(cur[k + 1] - cur[k] for k in range(len(cur) - 1))
            med = periods[len(periods) // 2]
        else:
            med = p
        # 門檻取「4%」與「1.5 個取樣」較大者：硬邊方波的過零點只能落在取樣格上，
        # 高音（988 Hz 一週期約 22.3 個取樣）的週期會在 22／23 之間跳，相差 4.5%。
        if p > 0.040 or abs(p - med) > max(0.04 * med, 1.5 / rate):
            segs.append(cur)
            cur = [t]
        else:
            cur.append(t)
    if cur:
        segs.append(cur)
    out = []
    for c in segs:
        if len(c) < 4:  # 少於 3 個週期
            continue
        span = c[-1] - c[0]
        f = (len(c) - 1) / span
        out.append({"start": c[0], "end": c[-1] + 1 / f, "hz": f})
    return out


def match(G, D, off, limit=None):
    """在位移 off 下逐段配對（§4 第 2 項）。回 (配對, 未配對, 比對範圍終點)。

    以**時間**找候選（二分搜尋 DOSBox-X 的起點），不以索引往後看幾段：
    中間缺一整塊時，索引式的配對會卡住，之後完全正常的段也跟著全部失敗，看不出缺的是哪裡。"""
    span_end = min(G[-1]["end"], D[-1]["end"] - off)
    starts = [d["start"] - off for d in D]
    used = set()
    pairs, unmatched = [], []
    for g in (G if limit is None else G[:limit]):
        if g["start"] > span_end:
            break
        if g["start"] < starts[0] - 0.050:
            continue  # DOSBox-X 還沒開始錄
        k = bisect.bisect_left(starts, g["start"] - 0.050)
        best = None
        while k < len(D) and starts[k] <= g["start"] + 0.050:
            if k not in used and abs(D[k]["hz"] - g["hz"]) / g["hz"] <= 0.02:
                best = k
                break
            k += 1
        if best is None:
            unmatched.append({"t": round(g["start"], 3), "hz": round(g["hz"], 1)})
        else:
            used.add(best)
            pairs.append((g, D[best]))
    return pairs, unmatched, span_end


def unmatched_dosboxx(D, pairs, off, span_start, span_end):
    """DOSBox-X 在比對範圍內、沒有被任何 dosgolem 段配對到的段（dosgolem 漏掉的音）。"""
    used = {id(d) for _, d in pairs}
    return [{"t": round(d["start"] - off, 3), "hz": round(d["hz"], 1)} for d in D
            if span_start - 0.050 <= d["start"] - off <= span_end and id(d) not in used]


def cmd_audio(gwav, dwav, out_json):
    gs, gr = read_wav(gwav)
    ds, dr = read_wav(dwav)
    G, D = segments(gs, gr), segments(ds, dr)
    if not G or not D:
        raise SystemExit(f"分析不出音符段：dosgolem {len(G)}、DOSBox-X {len(D)}")
    # 對齊：兩邊開始錄的時間不同（DOSBox-X 常漏掉開頭幾個音），所以不假設第一段對第一段。
    # 候選位移取兩邊前 10 段裡頻率相同的配對，挑讓前 50 段配對最多的那一個。
    cands = {round(d["start"] - g["start"], 4) for g in G[:10] for d in D[:10]
             if abs(d["hz"] - g["hz"]) / g["hz"] <= 0.02}
    if not cands:
        raise SystemExit("兩邊前 10 段沒有頻率相同的音，無法對齊")
    off = max(cands, key=lambda o: len(match(G, D, o, limit=50)[0]))
    pairs, unmatched, span_end = match(G, D, off)
    if not pairs:
        raise SystemExit(f"位移 {off:.3f} 秒下一段都沒配對到")
    span_start = max(G[0]["start"], D[0]["start"] - off)
    missing = unmatched_dosboxx(D, pairs, off, span_start, span_end)
    total_g = len(pairs) + len(unmatched)
    total_d = len(pairs) + len(missing)
    rate_g = len(pairs) / total_g if total_g else 0
    rate_d = len(pairs) / total_d if total_d else 0
    rate_ok = min(rate_g, rate_d)
    # 回歸
    xs = [g["start"] for g, _ in pairs]
    ys = [d["start"] for _, d in pairs]
    n = len(xs)
    mx, my = sum(xs) / n, sum(ys) / n
    sxx = sum((x - mx) ** 2 for x in xs)
    slope = sum((x - mx) * (y - my) for x, y in zip(xs, ys)) / sxx if sxx else 1.0
    icpt = my - slope * mx
    res = sorted(abs(y - (slope * x + icpt)) for x, y in zip(xs, ys))
    p95 = res[int(0.95 * (len(res) - 1))]
    ferr = max(abs(d["hz"] - g["hz"]) / g["hz"] for g, d in pairs)
    passed = rate_ok >= 0.98 and ferr <= 0.02 and abs(slope - 1) <= 0.005 and p95 <= 0.020
    result = {
        "layer": "audio", "golem_segments": len(G), "dosboxx_segments": len(D),
        "offset_s": round(off, 4), "compared_span_s": round(span_end - max(G[0]["start"], D[0]["start"] - off), 2),
        "matched": len(pairs), "unmatched_golem": len(unmatched), "unmatched_dosboxx": len(missing),
        "match_rate_golem": round(rate_g, 4), "match_rate_dosboxx": round(rate_d, 4), "match_rate": round(rate_ok, 4),
        "max_freq_error": round(ferr, 4), "slope": round(slope, 5),
        "onset_residual_p95_ms": round(p95 * 1000, 2), "onset_residual_max_ms": round(res[-1] * 1000, 2),
        "passed": passed, "first_unmatched_golem": unmatched[:20], "first_unmatched_dosboxx": missing[:20],
    }
    print(f"聲音層：dosgolem {len(G)} 段、DOSBox-X {len(D)} 段，比對範圍 {result['compared_span_s']} 秒")
    print(f"  配對 {len(pairs)}：dosgolem 段 {rate_g:.1%}（未配對 {len(unmatched)}）、DOSBox-X 段 {rate_d:.1%}（未配對 {len(missing)}）；頻率誤差最大 {ferr:.2%}，"
          f"斜率 {slope:.5f}，起點殘差 p95 {p95 * 1000:.1f} ms、最大 {res[-1] * 1000:.1f} ms")
    print("  判定：", "通過" if passed else "不通過")
    if unmatched:
        print("  dosgolem 多出來、DOSBox-X 沒有的段（前 5）：", unmatched[:5])
    if missing:
        print("  DOSBox-X 有、dosgolem 沒有的段（前 5）：", missing[:5])
    if out_json:
        Path(out_json).write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    return 0 if passed else 1


# ---------------------------------------------------------------- OPL2（docs/spec/002）

STEPS_PER_SECOND = 165000 * PIT_HZ / 17000  # 與 dosgolem machine.StepsPerSecond 相同


def opl_score(path):
    """-opl-log → [(秒, 聲道, Hz)]，時間以第一個 Key-On 為 0。"""
    fnum, block, on = [0] * 9, [0] * 9, [False] * 9
    ev = []
    for line in Path(path).read_text().splitlines():
        if line.startswith("#") or not line.strip():
            continue
        step, reg, val = line.split()
        step, reg, val = int(step), int(reg, 16), int(val, 16)
        if 0xA0 <= reg <= 0xA8:
            c = reg - 0xA0
            fnum[c] = fnum[c] & 0x300 | val
        elif 0xB0 <= reg <= 0xB8:
            c = reg - 0xB0
            fnum[c] = fnum[c] & 0xFF | (val & 3) << 8
            block[c] = val >> 2 & 7
            k = bool(val & 0x20)
            if k and not on[c]:
                ev.append((step, c, fnum[c] * 49716 / 2 ** (20 - block[c])))
            on[c] = k
    if not ev:
        raise SystemExit("樂譜沒有 Key-On")
    t0 = ev[0][0]
    return [((s - t0) / STEPS_PER_SECOND, c, hz) for s, c, hz in ev]


def onset_function(x, rate):
    hop = rate // 100
    le = []
    for k in range(0, len(x) - 2 * hop, hop):
        e = sum(v * v for v in x[k:k + 2 * hop])
        le.append(math.log(e + 1e-9))
    return [0.0] + [max(0.0, le[k] - le[k - 1]) for k in range(1, len(le))]


def goertzel(x, start, n, hz, rate, hann):
    if start < 0 or start + n > len(x):
        return 0.0
    coeff = 2 * math.cos(2 * math.pi * hz / rate)
    s1 = s2 = 0.0
    for k in range(n):
        s0 = x[start + k] * hann[k] + coeff * s1 - s2
        s2, s1 = s1, s0
    return s1 * s1 + s2 * s2 - coeff * s1 * s2


def opl_check(score, groups, wav):
    x, rate = read_wav(wav)
    o = onset_function(x, rate)
    pos = sorted(v for v in o if v > 0)
    thr = pos[int(0.75 * (len(pos) - 1))] if pos else 0.0
    frames = [round(g * 100) for g in groups]

    def local_max(idx):
        lo, hi = max(0, idx - 3), min(len(o), idx + 4)
        if lo >= hi:
            return -1, 0.0
        k = max(range(lo, hi), key=lambda j: o[j])
        return k, o[k]

    # 對齊：-5 到 +15 秒
    best_lag, best = 0, -1.0
    for lag in range(-500, 1501):
        sc = sum(local_max(f + lag)[1] for f in frames)
        if sc > best:
            best, best_lag = sc, lag
    hits, miss = [], []
    for g, f in zip(groups, frames):
        k, v = local_max(f + best_lag)
        if k >= 0 and v >= thr:
            hits.append((g, k / 100))
        else:
            miss.append({"t": round(g, 3)})
    xs = [g for g, _ in hits]
    ys = [t for _, t in hits]
    n = len(xs)
    if n >= 2:
        mx, my = sum(xs) / n, sum(ys) / n
        sxx = sum((u - mx) ** 2 for u in xs)
        slope = sum((u - mx) * (w - my) for u, w in zip(xs, ys)) / sxx
        icpt = my - slope * mx
        res = sorted(abs(w - (slope * u + icpt)) for u, w in zip(xs, ys))
        p95 = res[int(0.95 * (len(res) - 1))]
    else:
        slope, icpt, p95 = 0.0, best_lag / 100, float("inf")
    nwin = int(0.100 * rate)
    hann = [0.5 - 0.5 * math.cos(2 * math.pi * k / (nwin - 1)) for k in range(nwin)]
    present, absent, counted = 0, [], 0
    semi = 2 ** (1 / 12)
    for t, c, hz in score:
        if hz < 60 or hz > 4000:
            continue
        counted += 1
        start = int((slope * t + icpt + 0.030) * rate)
        p = goertzel(x, start, nwin, hz, rate, hann)
        ref = (goertzel(x, start, nwin, hz / semi, rate, hann) + goertzel(x, start, nwin, hz * semi, rate, hann)) / 2
        if p >= 2 * ref and p > 0:
            present += 1
        else:
            absent.append({"t": round(t, 3), "hz": round(hz, 1)})
    return {
        "wav": Path(wav).name, "offset_s": round(best_lag / 100, 2),
        "onset_hit_rate": round(len(hits) / len(groups), 4), "onset_threshold": round(thr, 4),
        "pitch_present_rate": round(present / counted, 4) if counted else 0.0, "pitch_counted": counted,
        "slope": round(slope, 5), "onset_residual_p95_ms": round(p95 * 1000, 1),
        "passed": len(hits) / len(groups) >= 0.90 and counted and present / counted >= 0.80 and abs(slope - 1) <= 0.005,
        "first_missed_onsets": miss[:20], "first_absent_pitches": absent[:20],
    }


def cmd_opl(log, gwav, dwav, out_json):
    score = opl_score(log)
    groups = []
    for t, _, _ in score:
        if not groups or t - groups[-1] > 0.015:
            groups.append(t)
    result = {"layer": "opl", "key_ons": len(score), "onset_groups": len(groups),
              "golem": opl_check(score, groups, gwav), "dosboxx": opl_check(score, groups, dwav)}
    print(f"OPL2：樂譜 {len(score)} 個 Key-On、{len(groups)} 個起音群組")
    for name in ("golem", "dosboxx"):
        r = result[name]
        print(f"  {name}：位移 {r['offset_s']} 秒；起音命中 {r['onset_hit_rate']:.1%}；音高在場 {r['pitch_present_rate']:.1%}"
              f"（{r['pitch_counted']} 個音）；斜率 {r['slope']}；起音殘差 p95 {r['onset_residual_p95_ms']} ms；"
              f"{'通過' if r['passed'] else '不通過'}")
    if out_json:
        Path(out_json).write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    return 0 if result["dosboxx"]["passed"] and result["golem"]["passed"] else 1


def main(argv):
    args = argv[1:]
    out_json = None
    if "--json" in args:
        k = args.index("--json")
        out_json = args[k + 1]
        del args[k:k + 2]
    if len(args) == 3 and args[0] == "events":
        return cmd_events(args[1], args[2], out_json)
    if len(args) == 3 and args[0] == "audio":
        return cmd_audio(args[1], args[2], out_json)
    if len(args) == 4 and args[0] == "opl":
        return cmd_opl(args[1], args[2], args[3], out_json)
    print(__doc__)
    return 2


if __name__ == "__main__":
    sys.exit(main(sys.argv))
