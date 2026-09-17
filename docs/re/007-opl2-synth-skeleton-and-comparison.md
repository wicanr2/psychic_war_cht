# 007：OPL2 合成雛形與比對結果

狀態：量測紀錄（2026-09-17）。對應 issue #5。**比對不通過，issue 仍開著。**

## 1. 結論

- dosgolem 新增 OPL2 合成雛形（分支 `psychic-war/m1-opl2` `a70be7f`，規格 `196-opl2-synth-skeleton`，未推上游）：
  `-adlib` 路徑的暫存器序列可以合成成 WAV（`-dump-opl-wav`）。5 個單元測試通過，全套測試通過。
- 依 `docs/spec/002` 比對標題音樂：**dosgolem 與 DOSBox-X 兩份 WAV 都不通過**。
- 更關鍵的是：**dosgolem 自己的 WAV 對自己的樂譜也不通過**。這份 WAV 就是由同一份樂譜合成的，所以這次不通過主要反映
  **比對方法的分辨力不足**，還不能當作「dosgolem 與 DOSBox-X 不一致」的證據。
- 兩份 WAV 的速度斜率都在 1 ± 0.03% 內，節拍速度一致；這與 PC 喇叭路徑的結論相同（docs/re/006）。

## 2. 樂譜（dosgolem `-adlib`，1,100,000,000 步）

| 項目 | 值 |
|---|---:|
| OPL2 寫入 | 10,999 筆 |
| Key-On | 776 次，639 個起音群組（相差 ≤ 15 ms 合併） |
| 合成長度 | 75.3 秒 |
| 用到但雛形沒做的功能 | KSL 651 次、震音 32 次、顫音 4 次 |

暫存器使用範圍（節奏模式、波形、連接方式等）見 dosgolem 規格 `196` §1。

## 3. 比對結果（`docs/spec/002`，門檻執行前定案）

DOSBox-X：`tools/dosboxx-audio.sh 90 adlib`（`sbtype=sb1`、`oplmode=opl2`，22,050 Hz）。錄音時主機負載約 12.6（14 核）。

| 指標 | dosgolem WAV | DOSBox-X WAV | 判準 |
|---|---:|---:|---:|
| 對齊位移 | 0.12 秒 | 1.86 秒 | — |
| 起音命中率（群組） | 59.8% | 65.4% | ≥ 90% |
| 音高在場率（523 個音） | 49.3% | 39.6% | ≥ 80% |
| 速度斜率 | 1.00029 | 1.00014 | 與 1 差 ≤ 0.5% |
| 起音殘差 p95 | 31.0 ms | 31.8 ms | — |
| 判定 | **不通過** | **不通過** | |

### 3.1 診斷（不是判準）

| 指標 | 值 |
|---|---:|
| 兩份 WAV 起音函數的相關係數（對齊後） | 0.23 |
| 逐音「音高在場」判定一致的比例 | 76.7%（401／523） |

## 4. 為什麼方法本身不夠

- **起音被遮蔽**：OPL2 的音多半是持續音，新音起音時其他聲道還在響，能量的相對增量小，10 ms 能量差抓不到。
- **音高檢查受泛音干擾**：FM 音色的泛音與旁帶會落在相鄰半音附近，「基頻 ≥ 相鄰半音 2 倍」在複音中常常不成立。
- **雛形音色與真機差距大**：包絡是 dB 線性近似，KSL、震音、顫音沒做；起音函數相關係數只有 0.23，
  表示兩邊的能量起伏形狀不同。音高判定仍有 76.7% 一致，代表音高大致相符，差距主要在起音與包絡。

## 5. 下一步（建議，未做）

1. **加一層事件比對**：DOSBox-X 能擷取 OPL raw 暫存器序列（mapper 的 raw OPL capture，產出 `.dro`）。
   與 dosgolem `-opl-log` 逐筆比，就像 PC 喇叭的事件層（docs/re/006 §4.1）。這不依賴合成器品質，最能直接回答
   「dosgolem 上的驅動是否照原樣寫暫存器」。
2. **比對方法改為雙 WAV 直接比**：同一個起音群組在兩份 WAV 的頻譜（例如 12 平均律的 chroma）相似度，而不是各自對樂譜判定。
   門檻要先拿 dosgolem 自我比對當正對照定案。
3. **提升雛形**：KSL、震音／顫音、依 YM3812 時間表的指數包絡。做完再重跑本比對。

## 6. 重跑

```sh
worktrees/dosgolem/tools/go.sh run ./cmd/probe -exe /orig/psychic-war/PW.EXE -root /orig/psychic-war -adlib \
    -steps 1100000000 -opl-log /wp/audio/opl-title-full.txt -dump-opl-wav /wp/audio/golem-opl-title.wav
tools/dosboxx-audio.sh 90 adlib
tools/py.sh tools/music_compare.py opl workplace/audio/opl-title-full.txt \
    workplace/audio/golem-opl-title.wav workplace/dosboxx-audio/title-adlib.wav
```
