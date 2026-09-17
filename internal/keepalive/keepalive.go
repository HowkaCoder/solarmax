// Package keepalive - фоновый "будильник" приложения.
//
// Бесплатные хостинги (Render, Koyeb, Fly и т.п.) усыпляют сервис, если по нему
// давно не было активности. Раз в заданный интервал (по умолчанию - час)
// приложение само дёргает свой публичный адрес и пишет об этом в лог.
package keepalive

import (
	"context"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type Pinger struct {
	urls     []string
	interval time.Duration
	client   *http.Client

	mu     sync.RWMutex
	status Status
}

// Status отдаётся наружу через GET /api/keepalive.
type Status struct {
	Enabled   bool         `json:"enabled"`
	Interval  string       `json:"interval"`
	Targets   []string     `json:"targets"`
	Runs      int64        `json:"runs"`
	StartedAt time.Time    `json:"started_at"`
	LastRunAt *time.Time   `json:"last_run_at,omitempty"`
	NextRunAt *time.Time   `json:"next_run_at,omitempty"`
	Results   []PingResult `json:"results,omitempty"`
}

type PingResult struct {
	URL        string    `json:"url"`
	StatusCode int       `json:"status_code,omitempty"`
	DurationMS int64     `json:"duration_ms"`
	Error      string    `json:"error,omitempty"`
	At         time.Time `json:"at"`
}

func New(urls []string, interval time.Duration) *Pinger {
	if interval <= 0 {
		interval = time.Hour
	}
	return &Pinger{
		urls:     urls,
		interval: interval,
		client:   &http.Client{Timeout: 30 * time.Second},
		status: Status{
			Interval:  interval.String(),
			Targets:   urls,
			StartedAt: time.Now().UTC(),
		},
	}
}

// Start запускает цикл пингов. Первый пинг - через минуту после старта,
// чтобы не мешать запуску, дальше - строго по интервалу.
func (p *Pinger) Start(ctx context.Context) {
	if len(p.urls) == 0 {
		log.Println("keepalive: адреса не заданы, фоновые пинги выключены")
		return
	}

	p.mu.Lock()
	p.status.Enabled = true
	next := time.Now().UTC().Add(time.Minute)
	p.status.NextRunAt = &next
	p.mu.Unlock()

	log.Printf("keepalive: включён, интервал %s, адреса: %v", p.interval, p.urls)

	go func() {
		first := time.NewTimer(time.Minute)
		defer first.Stop()

		select {
		case <-ctx.Done():
			return
		case <-first.C:
			p.Run(ctx)
		}

		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("keepalive: остановлен")
				return
			case <-ticker.C:
				p.Run(ctx)
			}
		}
	}()
}

// Run выполняет один проход по всем адресам. Можно дёрнуть и вручную
// через POST /api/keepalive/ping.
func (p *Pinger) Run(ctx context.Context) []PingResult {
	results := make([]PingResult, 0, len(p.urls))

	for _, url := range p.urls {
		results = append(results, p.ping(ctx, url))
	}

	now := time.Now().UTC()
	next := now.Add(p.interval)

	p.mu.Lock()
	p.status.Runs++
	p.status.LastRunAt = &now
	p.status.NextRunAt = &next
	p.status.Results = results
	p.mu.Unlock()

	return results
}

func (p *Pinger) ping(ctx context.Context, url string) PingResult {
	started := time.Now()
	res := PingResult{URL: url, At: started.UTC()}

	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		res.Error = err.Error()
		res.DurationMS = time.Since(started).Milliseconds()
		log.Printf("keepalive: не удалось собрать запрос к %s: %v", url, err)
		return res
	}
	req.Header.Set("User-Agent", "solarmax-keepalive/1.0")

	resp, err := p.client.Do(req)
	if err != nil {
		res.Error = err.Error()
		res.DurationMS = time.Since(started).Milliseconds()
		log.Printf("keepalive: пинг %s не удался: %v", url, err)
		return res
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	res.StatusCode = resp.StatusCode
	res.DurationMS = time.Since(started).Milliseconds()
	log.Printf("keepalive: пинг %s -> %d (%d мс)", url, res.StatusCode, res.DurationMS)
	return res
}

func (p *Pinger) Status() Status {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}
