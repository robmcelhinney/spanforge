package generator

import (
	"math/rand"
	"sync"
)

type RNG struct {
	mu sync.Mutex
	r  *rand.Rand
}

func NewRNG(seed int64) *RNG {
	return &RNG{r: rand.New(rand.NewSource(seed))}
}

func (r *RNG) Float64() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.r.Float64()
}

func (r *RNG) Intn(n int) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.r.Intn(n)
}

func (r *RNG) Read(p []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range p {
		p[i] = byte(r.r.Intn(256))
	}
}
