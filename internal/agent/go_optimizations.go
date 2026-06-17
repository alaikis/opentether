package agent

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type WorkerPool struct {
	tasks   chan func()
	workers int
	wg      sync.WaitGroup
	active  int64
}

func NewWorkerPool(size int) *WorkerPool {
	if size <= 0 {
		size = 4
	}
	pool := &WorkerPool{tasks: make(chan func(), size*2), workers: size}
	for i := 0; i < size; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}
	return pool
}

func (p *WorkerPool) worker() {
	defer p.wg.Done()
	for task := range p.tasks {
		atomic.AddInt64(&p.active, 1)
		task()
		atomic.AddInt64(&p.active, -1)
	}
}

func (p *WorkerPool) Submit(task func()) {
	p.tasks <- task
}

func (p *WorkerPool) Active() int64 { return atomic.LoadInt64(&p.active) }
func (p *WorkerPool) Shutdown()     { close(p.tasks); p.wg.Wait() }

type Metrics struct {
	FastPathHits    int64
	AgentLoopCalls  int64
	LLMCalls        int64
	SQLCalls        int64
	ToolCalls       int64
	CacheHits       int64
	TotalTokens     int64
	TotalDurationMs int64
}

func (m *Metrics) IncFastPath()         { atomic.AddInt64(&m.FastPathHits, 1) }
func (m *Metrics) IncAgentLoop()        { atomic.AddInt64(&m.AgentLoopCalls, 1) }
func (m *Metrics) IncLLM()              { atomic.AddInt64(&m.LLMCalls, 1) }
func (m *Metrics) IncSQL()              { atomic.AddInt64(&m.SQLCalls, 1) }
func (m *Metrics) IncTool()             { atomic.AddInt64(&m.ToolCalls, 1) }
func (m *Metrics) IncCacheHit()         { atomic.AddInt64(&m.CacheHits, 1) }
func (m *Metrics) AddTokens(n int)      { atomic.AddInt64(&m.TotalTokens, int64(n)) }
func (m *Metrics) AddDuration(ms int64) { atomic.AddInt64(&m.TotalDurationMs, ms) }
func (m *Metrics) Snapshot() map[string]int64 {
	return map[string]int64{
		"fast_path_hits":    atomic.LoadInt64(&m.FastPathHits),
		"agent_loop_calls":  atomic.LoadInt64(&m.AgentLoopCalls),
		"llm_calls":         atomic.LoadInt64(&m.LLMCalls),
		"sql_calls":         atomic.LoadInt64(&m.SQLCalls),
		"tool_calls":        atomic.LoadInt64(&m.ToolCalls),
		"cache_hits":        atomic.LoadInt64(&m.CacheHits),
		"total_tokens":      atomic.LoadInt64(&m.TotalTokens),
		"total_duration_ms": atomic.LoadInt64(&m.TotalDurationMs),
	}
}

var bufPool = sync.Pool{New: func() interface{} { return make([]byte, 0, 4096) }}

func GetBuffer() []byte  { return bufPool.Get().([]byte)[:0] }
func PutBuffer(b []byte) { if cap(b) <= 16384 { bufPool.Put(b[:0]) } }

type LLMCache struct {
	mu    sync.Mutex
	items map[string]llmCacheEntry
}

type llmCacheEntry struct {
	Response  string
	ExpiresAt time.Time
}

func NewLLMCache() *LLMCache { return &LLMCache{items: make(map[string]llmCacheEntry)} }

func (c *LLMCache) Get(key string) (string, bool) {
	c.mu.Lock()
	e, ok := c.items[key]
	c.mu.Unlock()
	if ok && time.Now().Before(e.ExpiresAt) {
		return e.Response, true
	}
	return "", false
}

func (c *LLMCache) Set(key, value string, ttl time.Duration) {
	c.mu.Lock()
	c.items[key] = llmCacheEntry{Response: value, ExpiresAt: time.Now().Add(ttl)}
	c.mu.Unlock()
}

func WithDeadline(ctx context.Context, duration time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil { ctx = context.Background() }
	return context.WithTimeout(ctx, duration)
}
