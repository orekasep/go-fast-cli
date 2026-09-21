package fast

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// Phase represents the current stage of the speed test.
type Phase int

const (
	PhaseInit Phase = iota
	PhaseConnecting
	PhaseTesting
	PhaseCompleted
	PhaseError
)

func (p Phase) String() string {
	switch p {
	case PhaseInit:
		return "Initializing"
	case PhaseConnecting:
		return "Connecting"
	case PhaseTesting:
		return "Testing"
	case PhaseCompleted:
		return "Completed"
	case PhaseError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Config defines execution parameters for the speed test.
type Config struct {
	Duration time.Duration
	Threads  int
	URLCount int
	Proxy    string
}

// DefaultConfig provides sensible defaults for quick and accurate testing.
func DefaultConfig() Config {
	return Config{
		Duration: 10 * time.Second,
		Threads:  4,
		URLCount: 5,
	}
}

// Progress contains the live status emitted during a test.
type Progress struct {
	Phase            Phase
	InstantSpeedMbps float64
	AverageSpeedMbps float64
	BytesTransferred int64
	Elapsed          time.Duration
	TotalDuration    time.Duration
	Percent          float64
	Latency          time.Duration
	Proxy            string
	Client           *ClientInfo
	Targets          []TargetServer
	Err              error
}

// Tester manages the lifecycle of the Fast.com speed test.
type Tester struct {
	cfg Config
}

// NewTester creates a new Tester instance with the provided config.
func NewTester(cfg Config) *Tester {
	if cfg.Duration <= 0 {
		cfg.Duration = 10 * time.Second
	}
	if cfg.Threads <= 0 {
		cfg.Threads = 4
	}
	if cfg.URLCount <= 0 {
		cfg.URLCount = 5
	}
	return &Tester{cfg: cfg}
}

// Run executes the speed test and streams progress updates on the returned channel.
func (t *Tester) Run(ctx context.Context) <-chan Progress {
	ch := make(chan Progress, 16)

	go func() {
		defer close(ch)

		// 1. Initializing & token retrieval
		ch <- Progress{
			Phase:         PhaseConnecting,
			Proxy:         t.cfg.Proxy,
			TotalDuration: t.cfg.Duration,
		}

		apiClient, err := CreateHTTPClient(t.cfg.Proxy, 6*time.Second)
		if err != nil {
			ch <- Progress{
				Phase: PhaseError,
				Proxy: t.cfg.Proxy,
				Err:   fmt.Errorf("failed to configure proxy: %w", err),
			}
			return
		}

		// 1. Target retrieval
		targetsResp, err := GetSpeedtestTargetsWithClient(ctx, apiClient, DefaultToken, t.cfg.URLCount)
		if err != nil {
			ch <- Progress{
				Phase: PhaseError,
				Proxy: t.cfg.Proxy,
				Err:   fmt.Errorf("failed to get targets: %w", err),
			}
			return
		}

		clientInfo := targetsResp.Client
		targets := targetsResp.Targets

		// 2. Measure latency concurrently so download testing begins immediately
		var latencyAtomic int64
		if len(targets) > 0 {
			latencyClient, _ := CreateHTTPClient(t.cfg.Proxy, 4*time.Second)
			go func(url string) {
				if l, err := MeasureLatencyWithClient(ctx, latencyClient, url); err == nil {
					atomic.StoreInt64(&latencyAtomic, int64(l))
				}
			}(targets[0].URL)
		}

		ch <- Progress{
			Phase:         PhaseConnecting,
			Proxy:         t.cfg.Proxy,
			Client:        &clientInfo,
			Targets:       targets,
			TotalDuration: t.cfg.Duration,
		}

		// 3. Launch concurrent download streams
		var totalBytes int64
		testCtx, cancel := context.WithTimeout(ctx, t.cfg.Duration)
		defer cancel()

		startTime := time.Now()
		var wg sync.WaitGroup

		transport := &http.Transport{
			MaxIdleConnsPerHost: t.cfg.Threads,
			DisableCompression:  true,
		}
		if t.cfg.Proxy != "" {
			parsedProxy, err := url.Parse(t.cfg.Proxy)
			if err != nil {
				ch <- Progress{
					Phase: PhaseError,
					Proxy: t.cfg.Proxy,
					Err:   fmt.Errorf("invalid proxy URL %q: %w", t.cfg.Proxy, err),
				}
				return
			}
			transport.Proxy = http.ProxyURL(parsedProxy)
		} else {
			transport.Proxy = http.ProxyFromEnvironment
		}
		httpClient := &http.Client{
			Transport: transport,
		}

		// Distribute workers across available targets
		for i := 0; i < t.cfg.Threads; i++ {
			targetURL := targets[i%len(targets)].URL
			wg.Add(1)
			go func(url string) {
				defer wg.Done()
				buf := make([]byte, 64*1024)

				for {
					select {
					case <-testCtx.Done():
						return
					default:
					}

					req, err := http.NewRequestWithContext(testCtx, http.MethodGet, url, nil)
					if err != nil {
						return
					}
					req.Header.Set("User-Agent", userAgent)

					resp, err := httpClient.Do(req)
					if err != nil {
						// Context cancelled or network blip
						continue
					}

					for {
						n, readErr := resp.Body.Read(buf)
						if n > 0 {
							atomic.AddInt64(&totalBytes, int64(n))
						}
						if readErr != nil {
							break
						}
					}
					resp.Body.Close()
				}
			}(targetURL)
		}

		// 4. Progress monitoring loop
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		var lastBytes int64
		var lastTime = startTime
		var smoothedSpeed float64

		running := true
		for running {
			select {
			case <-testCtx.Done():
				running = false
			case now := <-ticker.C:
				currentBytes := atomic.LoadInt64(&totalBytes)
				elapsed := now.Sub(startTime)
				if elapsed > t.cfg.Duration {
					elapsed = t.cfg.Duration
				}

				deltaBytes := currentBytes - lastBytes
				deltaTime := now.Sub(lastTime).Seconds()

				var instantMbps float64
				if deltaTime > 0 {
					instantMbps = (float64(deltaBytes) * 8) / (deltaTime * 1_000_000)
				}

				// Exponential moving average for smooth display
				if smoothedSpeed == 0 {
					smoothedSpeed = instantMbps
				} else {
					smoothedSpeed = (0.75 * instantMbps) + (0.25 * smoothedSpeed)
				}

				lastBytes = currentBytes
				lastTime = now

				var avgMbps float64
				if elapsed.Seconds() > 0 {
					avgMbps = (float64(currentBytes) * 8) / (elapsed.Seconds() * 1_000_000)
				}

				percent := float64(elapsed) / float64(t.cfg.Duration)
				if percent > 1.0 {
					percent = 1.0
				}

				currentLatency := time.Duration(atomic.LoadInt64(&latencyAtomic))
				ch <- Progress{
					Phase:            PhaseTesting,
					InstantSpeedMbps: smoothedSpeed,
					AverageSpeedMbps: avgMbps,
					BytesTransferred: currentBytes,
					Elapsed:          elapsed,
					TotalDuration:    t.cfg.Duration,
					Percent:          percent,
					Latency:          currentLatency,
					Proxy:            t.cfg.Proxy,
					Client:           &clientInfo,
					Targets:          targets,
				}
			}
		}

		// Wait for download workers to finish cleanly
		wg.Wait()

		// Final metrics
		finalBytes := atomic.LoadInt64(&totalBytes)
		totalElapsed := time.Since(startTime)
		if totalElapsed > t.cfg.Duration {
			totalElapsed = t.cfg.Duration
		}
		var finalAvgMbps float64
		if totalElapsed.Seconds() > 0 {
			finalAvgMbps = (float64(finalBytes) * 8) / (totalElapsed.Seconds() * 1_000_000)
		}

		finalLatency := time.Duration(atomic.LoadInt64(&latencyAtomic))
		ch <- Progress{
			Phase:            PhaseCompleted,
			InstantSpeedMbps: finalAvgMbps,
			AverageSpeedMbps: finalAvgMbps,
			BytesTransferred: finalBytes,
			Elapsed:          totalElapsed,
			TotalDuration:    t.cfg.Duration,
			Percent:          1.0,
			Latency:          finalLatency,
			Proxy:            t.cfg.Proxy,
			Client:           &clientInfo,
			Targets:          targets,
		}
	}()

	return ch
}
