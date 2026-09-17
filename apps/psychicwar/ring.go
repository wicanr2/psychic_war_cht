package psychicwar

import "sync"

// Ring 是音訊的環形緩衝（docs/spec/006 §3.5）：模擬執行緒寫、音效卡端讀，不夠時補 0 並計一次欠載。
type Ring struct {
	mu        sync.Mutex
	buf       []int16
	r, n      int
	Underruns int
	Overflows int // 寫滿時丟掉最舊取樣的次數
}

// NewRing 建一個容量 capacity 取樣的緩衝。
func NewRing(capacity int) *Ring { return &Ring{buf: make([]int16, capacity)} }

// Write 寫入取樣；滿了就丟掉最舊的。
func (q *Ring) Write(p []int16) {
	q.mu.Lock()
	defer q.mu.Unlock()
	dropped := false
	for _, v := range p {
		if q.n == len(q.buf) {
			q.r = (q.r + 1) % len(q.buf)
			q.n--
			dropped = true
		}
		q.buf[(q.r+q.n)%len(q.buf)] = v
		q.n++
	}
	if dropped {
		q.Overflows++
	}
}

// Read 填滿 dst；緩衝裡的取樣不夠時其餘補 0，Underruns 加一。回填入的真實取樣數。
func (q *Ring) Read(dst []int16) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	k := 0
	for ; k < len(dst) && q.n > 0; k++ {
		dst[k] = q.buf[q.r]
		q.r = (q.r + 1) % len(q.buf)
		q.n--
	}
	if k < len(dst) {
		for i := k; i < len(dst); i++ {
			dst[i] = 0
		}
		q.Underruns++
	}
	return k
}

// Len 回緩衝裡的取樣數。
func (q *Ring) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.n
}

// Stats 回欠載與溢位次數。
func (q *Ring) Stats() (underruns, overflows int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.Underruns, q.Overflows
}
