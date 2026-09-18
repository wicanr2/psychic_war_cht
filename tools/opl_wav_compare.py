"""兩份 OPL2 音訊有多接近（`docs/spec/016`，issue #5）。

    tools/py.sh tools/opl_wav_compare.py <a.wav> <b.wav> [--json out.json] [--secs 60]

事件層已經與 DOSBox-X 逐筆相同（`docs/re/012`），所以音高與節奏是由事件決定的，
不必再從音訊反推——那正是 `docs/re/007` 分辨力不足的來源。這支只回答一件事：
**同一串暫存器寫入，兩個合成器算出來的波形有多接近。**

三個指標（都在對齊之後、兩份共同涵蓋的區間內算，都取中位數／相關係數）：

- `spec`   STFT 每幀幅度譜 L2 正規化後的餘弦相似度中位數 → 音色與音高
- `env`    10 ms RMS 序列各自除以自身中位數後的 Pearson 相關 → 包絡形狀
- `chroma` 幅度譜摺到 12 個半音類別後的餘弦相似度中位數 → 音高（對音色不敏感）

指標都先正規化，所以音量差不會被抓到（那是前端的事）；用幅度譜，所以相位差也不會。
門檻的定法見 `docs/spec/016` §3.4：由「DOSBox-X 錄兩次」的上界與「不同曲子」的下界推出來。
"""
import cmath
import json
import math
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
from music_compare import read_wav  # noqa: E402

ROOT = pathlib.Path(__file__).resolve().parent.parent
N_FFT, HOP = 1024, 256
ENV_MS = 10.0
MAX_SHIFT_S = 5.0


def resample(x, src, dst):
    """線性內插重取樣（兩份錄音取樣率不同時才用到）。"""
    if src == dst:
        return x
    n = int(len(x) * dst / src)
    out = [0.0] * n
    for i in range(n):
        t = i * src / dst
        j = int(t)
        f = t - j
        out[i] = x[j] if j + 1 >= len(x) else x[j] * (1 - f) + x[j + 1] * f
    return out


def fft(a):
    """就地 Cooley-Tukey；長度必須是 2 的冪。"""
    n = len(a)
    if n == 1:
        return a
    ev = fft(a[0::2])
    od = fft(a[1::2])
    out = [0j] * n
    for k in range(n // 2):
        t = cmath.exp(-2j * math.pi * k / n) * od[k]
        out[k] = ev[k] + t
        out[k + n // 2] = ev[k] - t
    return out


HANN = [0.5 - 0.5 * math.cos(2 * math.pi * i / (N_FFT - 1)) for i in range(N_FFT)]


def frames(x):
    """逐幀幅度譜（只取一半頻譜）。"""
    out = []
    for s in range(0, len(x) - N_FFT + 1, HOP):
        buf = [complex(x[s + i] * HANN[i], 0.0) for i in range(N_FFT)]
        sp = fft(buf)
        out.append([abs(v) for v in sp[: N_FFT // 2]])
    return out


def envelope(x, rate):
    w = max(int(rate * ENV_MS / 1000), 1)
    return [math.sqrt(sum(v * v for v in x[i:i + w]) / w) for i in range(0, len(x) - w + 1, w)]


def cosine(a, b):
    na = math.sqrt(sum(v * v for v in a))
    nb = math.sqrt(sum(v * v for v in b))
    if na == 0 or nb == 0:
        return 0.0
    return sum(p * q for p, q in zip(a, b)) / (na * nb)


def pearson(a, b):
    n = min(len(a), len(b))
    a, b = a[:n], b[:n]
    ma, mb = sum(a) / n, sum(b) / n
    va = math.sqrt(sum((v - ma) ** 2 for v in a))
    vb = math.sqrt(sum((v - mb) ** 2 for v in b))
    if va == 0 or vb == 0:
        return 0.0
    return sum((p - ma) * (q - mb) for p, q in zip(a, b)) / (va * vb)


def median(xs):
    if not xs:
        return 0.0
    s = sorted(xs)
    n = len(s)
    return s[n // 2] if n % 2 else (s[n // 2 - 1] + s[n // 2]) / 2


def align(ea, eb, max_shift):
    """用包絡互相關取最大位移（單位：包絡樣本 ＝ ENV_MS 毫秒）。回 (位移, 相關值)。"""
    best, best_r = 0, -2.0
    for sh in range(-max_shift, max_shift + 1):
        if sh >= 0:
            x, y = ea[sh:], eb
        else:
            x, y = ea, eb[-sh:]
        n = min(len(x), len(y))
        if n < 100:
            continue
        r = pearson(x[:n], y[:n])
        if r > best_r:
            best, best_r = sh, r
    return best, best_r


def chroma_of(frame, rate):
    """幅度譜摺到 12 個半音類別（A4 ＝ 440 Hz）。"""
    out = [0.0] * 12
    for k, v in enumerate(frame):
        if k == 0:
            continue
        f = k * rate / N_FFT
        if f < 55 or f > 8000:
            continue
        out[int(round(12 * math.log2(f / 440.0))) % 12] += v
    return out


def main(argv):
    paths = [a for a in argv if not a.startswith("--")]
    if len(paths) < 2:
        print(__doc__)
        return 2
    secs = float(argv[argv.index("--secs") + 1]) if "--secs" in argv else 0.0
    # --cut-b 先把 b 的前 N 秒丟掉，--no-align 不做互相關對齊。
    # 兩個合起來才做得出「同一份錄音、錯開 N 秒」這個下界對照（§3.4 的 L）：
    # 讓自動對齊去跑的話，它會把錯開的部分對回來，量到的就不是「不同內容」了。
    cut_b = float(argv[argv.index("--cut-b") + 1]) if "--cut-b" in argv else 0.0
    no_align = "--no-align" in argv
    xa, ra = read_wav(ROOT / paths[0] if not pathlib.Path(paths[0]).is_absolute() else paths[0])
    xb, rb = read_wav(ROOT / paths[1] if not pathlib.Path(paths[1]).is_absolute() else paths[1])
    rate = min(ra, rb)
    xa, xb = resample(xa, ra, rate), resample(xb, rb, rate)

    if cut_b:
        xb = xb[int(cut_b * rate):]
    if no_align:
        shift, shift_r = 0, float("nan")
    else:
        ea, eb = envelope(xa, rate), envelope(xb, rate)
        shift, shift_r = align(ea, eb, int(MAX_SHIFT_S * 1000 / ENV_MS))
        off = int(abs(shift) * ENV_MS / 1000 * rate)
        if shift >= 0:
            xa = xa[off:]
        else:
            xb = xb[off:]
    n = min(len(xa), len(xb))
    if secs:
        n = min(n, int(secs * rate))
    xa, xb = xa[:n], xb[:n]
    if n < rate:  # 共同區間不到 1 秒：對不上
        print("對齊後共同區間只有 %.2f 秒，兩份錄音大概不是同一段" % (n / rate))
        return 1

    fa, fb = frames(xa), frames(xb)
    m = min(len(fa), len(fb))
    spec = median([cosine(fa[i], fb[i]) for i in range(m)])
    ca = [chroma_of(f, rate) for f in fa[:m]]
    cb = [chroma_of(f, rate) for f in fb[:m]]
    chroma = median([cosine(ca[i], cb[i]) for i in range(m)])
    va, vb = envelope(xa, rate), envelope(xb, rate)
    mva, mvb = median(va) or 1.0, median(vb) or 1.0
    env = pearson([v / mva for v in va], [v / mvb for v in vb])

    out = {
        "a": paths[0], "b": paths[1], "rate": rate, "seconds": round(n / rate, 2),
        "shift_ms": round(shift * ENV_MS, 1), "shift_r": round(shift_r, 4),
        "spec": round(spec, 4), "env": round(env, 4), "chroma": round(chroma, 4),
    }
    print("%s vs %s" % (paths[0], paths[1]))
    print("  對齊位移 %+.0f ms（相關 %.3f），共同區間 %.1f 秒 @ %d Hz"
          % (out["shift_ms"], shift_r, out["seconds"], rate))
    print("  spec %.4f   env %.4f   chroma %.4f" % (spec, env, chroma))
    if "--json" in argv:
        p = ROOT / argv[argv.index("--json") + 1]
        p.write_text(json.dumps(out, ensure_ascii=False, indent=1) + "\n", encoding="utf-8")
        print("  寫出", p)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
