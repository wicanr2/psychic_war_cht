# 020 — 真實音訊裝置上的欠載量測

狀態：**READY**
日期：2026-09-18
對應 issue：#13
前置：`docs/spec/006`（前端）、`docs/re/016` §2.4、`docs/re/026`（null 輸出的數字）

---

## 1. 問題

到目前為止的欠載數字都是 `-audio null` 量的：那條路徑**不開音訊裝置**，照牆上時間把取樣丟掉。
它量得到「合成與節拍跟不跟得上」，量不到**裝置緩衝那一段**——真正會讓玩家聽到斷音的地方。

`docs/re/026` §6 試過 ALSA 的 `null` PCM，不行：它不按取樣率消耗資料而是立刻吸收，
應用端的緩衝永遠是空的，65 秒量到 290,203 次欠載，那是它自己的行為不是待測系統的。

**dummy sink 要會按時序消耗，否則量到的是它自己。**

## 2. 怎麼接（使用者授權 2026-09-18）

主機跑的是 PipeWire（含 `pipewire-pulse`）。容器經由**它的 pulse socket** 出聲：

| 做法 | 為什麼選它 |
|---|---|
| 掛 `/run/user/<uid>/pulse/native`，ALSA 走 pulse plugin | 不獨佔音效卡，主機的其他聲音照常。時序由 PipeWire 決定，是真的 |
| ~~掛 `/dev/snd` 直接開 `hw:0`~~ | 會跟主機的音訊伺服器搶裝置。實測（2026-09-18，`--device /dev/snd --group-add audio`）：`oto: ALSA error at snd_pcm_hw_params: Invalid argument`，連開都開不起來 |
| ~~ALSA 的 `null` PCM~~ | 不按時序消耗（§1） |

容器需要 `libasound2-plugins`（ALSA 的 pulse plugin），已加進 `tools/docker/go-ebiten.Dockerfile`。
`tools/go-ebiten.sh` 的 `PSYCHICWAR_DOCKER_EXTRA` 負責把 socket 掛進去。

⚠ **這會從喇叭發出聲音**，量測期間主機會播 65 秒的遊戲音樂。這是刻意的——
沒有真的送到裝置就量不到裝置緩衝。

⚠ 這個掛載在硬規則的「不碰共用資源」上開了一個口。口要小：
只掛那一個 socket、只在這支量測腳本用、用完就沒有常駐的東西留在主機上。

## 3. 量什麼

`-stats` 每秒一行 JSON。判讀與 `docs/re/026` 相同：

| 指標 | 意義 |
|---|---|
| `underruns` 逐秒差 | 該秒緩衝被要資料卻空了幾次 |
| `dropped_ms` | 放棄追趕的機器時間 |
| `overflows` | 反過來的情形：產得比消耗快 |
| `cpu_ms / wall_ms` | CPU 佔用（核） |
| `machine_ms / wall_ms` | 機器時間有沒有跟上牆上時間 |

## 4. 驗收

1. **AdLib 路徑 65 秒**：開場第 1 秒以外欠載 0。
2. **PC 喇叭路徑 65 秒**：同上。
3. 量測當時的每核負載 < 0.5（這台 14 核 ＝ load < 7），開跑與結束都記下來。
4. 反向對照：同一次執行的 `machine_ms / wall_ms` 要落在 1 ± 0.01——
   機器時間沒跟上的話，「欠載 0」可能只是因為它跑得慢、產得少。
5. 與 `-audio null` 的數字並列（`docs/re/026` §5），差異要能解釋。

## 5. 不做

- 聽感比對（音色對不對是 `docs/spec/016` 的事）。
- 不同音效卡、不同緩衝大小的掃描。
