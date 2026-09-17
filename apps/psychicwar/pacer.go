// Package psychicwar 是《銀河超能力戰記》前端與轉譯層的遊戲專屬部分（docs/spec/006）。
package psychicwar

// MaxCatchUpMs 是單次最多補跑的機器時間（docs/spec/006 §3.2）。
const MaxCatchUpMs = 100.0

// Pacer 把牆上時間換成這一次要跑的 cycles（docs/spec/006 §3.2）。
//
// 機器時間由呼叫端從累計 cycles 換算後傳進來，不在這裡另外累加——兩個時鐘各算各的會漂移。
// 落後超過 MaxCatchUpMs 時放棄追趕，放棄的量記在 DroppedMs，之後的目標扣掉它。
type Pacer struct {
	PerMs     uint64  // 每毫秒 cycles
	DroppedMs float64 // 累計放棄追趕的毫秒數
}

// Cycles 回這一次要跑的 cycles。wallMs 是啟動後經過的牆上毫秒，machineMs 是啟動後已跑的機器毫秒。
func (p *Pacer) Cycles(wallMs, machineMs float64) uint64 {
	lag := wallMs - p.DroppedMs - machineMs
	if lag <= 0 {
		return 0
	}
	if lag > MaxCatchUpMs {
		p.DroppedMs += lag - MaxCatchUpMs
		lag = MaxCatchUpMs
	}
	return uint64(lag * float64(p.PerMs))
}
