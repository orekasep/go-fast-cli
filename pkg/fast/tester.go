package fast

import (
	"context"
	"fmt"
	"io"
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
	PhaseUploading
	PhaseCompleted
	PhaseError
)

// PhaseDownloading is an alias for PhaseTesting.
const PhaseDownloading = PhaseTesting

func (p Phase) String() string {
	switch p {
	case PhaseInit:
		return "Initializing"
	case PhaseConnecting:
		return "Connecting"
	case PhaseTesting:
		return "Downloading"
	case PhaseUploading:
		return "Uploading"
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
	Upload   bool
}

// DefaultConfig provides sensible defaults for quick and accurate testing.
func DefaultConfig() Config {
	return Config{
		Duration: 10 * time.Second,
		Threads:  4,
		URLCount: 5,
		Upload:   true,
	}
}

// Progress contains the live status emitted during a test.
type Progress struct {
	Phase             Phase
	InstantSpeedMbps  float64
	AverageSpeedMbps  float64
	DownloadSpeedMbps float64
	UploadSpeedMbps   float64
	BytesTransferred  int64
	DownloadBytes     int64
	UploadBytes       int64
	Elapsed           time.Duration
	TotalDuration     time.Duration
	Percent           float64
	Latency           time.Duration
	Proxy             string
	Client            *ClientInfo
	Targets           []TargetServer
	Err               error
}

var zeroChunk = make([]byte, 64*1024)

type zeroReader struct{}

func (z zeroReader) Read(p []byte) (int, error) {
	return copy(p, zeroChunk), nil
}

type countingReader struct {
	r      io.Reader
	copied *int64
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	if n > 0 {
		atomic.AddInt64(cr.copied, int64(n))
	}
	return n, err
}

const maxPayloadBytes = 25 * 1024 * 1024

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

		overallStartTime := time.Now()

		// 1. Initializing & token retrieval
		ch <- Progress{
			Phase:         PhaseConnecting,
			Proxy:         t.cfg.Proxy,
			TotalDuration: t.cfg.Duration,
		}

		var parsedProxy *url.URL
		if t.cfg.Proxy != "" {
			var err error
			parsedProxy, err = url.Parse(t.cfg.Proxy)
			if err != nil {
				ch <- Progress{
					Phase: PhaseError,
					Proxy: t.cfg.Proxy,
					Err:   fmt.Errorf("invalid proxy URL %q: %w", t.cfg.Proxy, err),
				}
				return
			}
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

		// Target retrieval
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
			go func(targetURL string) {
				if l, err := MeasureLatencyWithClient(ctx, latencyClient, targetURL); err == nil {
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

		// 3. Shared HTTP client for keep-alive connection reuse
		transport := &http.Transport{
			MaxIdleConnsPerHost: t.cfg.Threads * 2,
			DisableCompression:  true,
		}
		if parsedProxy != nil {
			transport.Proxy = http.ProxyURL(parsedProxy)
		} else {
			transport.Proxy = http.ProxyFromEnvironment
		}
		httpClient := &http.Client{
			Transport: transport,
		}

		// 4. Launch concurrent download streams
		var totalBytes int64
		downloadCtx, downloadCancel := context.WithTimeout(ctx, t.cfg.Duration)
		defer downloadCancel()

		downloadStartTime := time.Now()
		var downloadWg sync.WaitGroup

		// Distribute workers across available targets
		for i := 0; i < t.cfg.Threads; i++ {
			targetURL := targets[i%len(targets)].URL
			downloadWg.Add(1)
			go func(url string) {
				defer downloadWg.Done()
				buf := make([]byte, 64*1024)

				for {
					select {
					case <-downloadCtx.Done():
						return
					default:
					}

					req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, url, nil)
					if err != nil {
						return
					}
					req.Header.Set("User-Agent", userAgent)

					resp, err := httpClient.Do(req)
					if err != nil {
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

		// Download progress monitoring loop
		downloadTicker := time.NewTicker(100 * time.Millisecond)
		defer downloadTicker.Stop()

		var lastBytes int64
		var lastTime = downloadStartTime
		var smoothedSpeed float64

		running := true
		for running {
			select {
			case <-downloadCtx.Done():
				running = false
			case now := <-downloadTicker.C:
				currentBytes := atomic.LoadInt64(&totalBytes)
				elapsed := now.Sub(downloadStartTime)
				if elapsed > t.cfg.Duration {
					elapsed = t.cfg.Duration
				}

				deltaBytes := currentBytes - lastBytes
				deltaTime := now.Sub(lastTime).Seconds()

				var instantMbps float64
				if deltaTime > 0 {
					instantMbps = (float64(deltaBytes) * 8) / (deltaTime * 1_000_000)
				}

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
					Phase:             PhaseTesting,
					InstantSpeedMbps:  smoothedSpeed,
					AverageSpeedMbps:  avgMbps,
					DownloadSpeedMbps: avgMbps,
					BytesTransferred:  currentBytes,
					DownloadBytes:     currentBytes,
					Elapsed:           elapsed,
					TotalDuration:     t.cfg.Duration,
					Percent:           percent,
					Latency:           currentLatency,
					Proxy:             t.cfg.Proxy,
					Client:            &clientInfo,
					Targets:           targets,
				}
			}
		}

		// Wait for download workers to finish cleanly
		downloadWg.Wait()

		// Final download metrics
		finalDownloadBytes := atomic.LoadInt64(&totalBytes)
		totalDownloadElapsed := time.Since(downloadStartTime)
		if totalDownloadElapsed > t.cfg.Duration {
			totalDownloadElapsed = t.cfg.Duration
		}
		var finalDownloadAvgMbps float64
		if totalDownloadElapsed.Seconds() > 0 {
			finalDownloadAvgMbps = (float64(finalDownloadBytes) * 8) / (totalDownloadElapsed.Seconds() * 1_000_000)
		}

		// 5. Upload phase (if enabled and context not cancelled)
		var finalUploadBytes int64
		var finalUploadAvgMbps float64

		if t.cfg.Upload && ctx.Err() == nil {
			var totalUploadBytes int64
			uploadCtx, uploadCancel := context.WithTimeout(ctx, t.cfg.Duration)
			defer uploadCancel()

			uploadStartTime := time.Now()
			var uploadWg sync.WaitGroup

			for i := 0; i < t.cfg.Threads; i++ {
				targetURL := targets[i%len(targets)].URL
				uploadWg.Add(1)
				go func(url string) {
					defer uploadWg.Done()

					for {
						select {
						case <-uploadCtx.Done():
							return
						default:
						}

						body := &countingReader{
							r:      io.LimitReader(zeroReader{}, maxPayloadBytes),
							copied: &totalUploadBytes,
						}

						req, err := http.NewRequestWithContext(uploadCtx, http.MethodPost, url, body)
						if err != nil {
							return
						}
						req.ContentLength = maxPayloadBytes
						req.Header.Set("Content-Type", "application/octet-stream")
						req.Header.Set("User-Agent", userAgent)

						resp, err := httpClient.Do(req)
						if err != nil {
							continue
						}
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
					}
				}(targetURL)
			}

			uploadTicker := time.NewTicker(100 * time.Millisecond)
			defer uploadTicker.Stop()

			var lastUploadBytes int64
			var lastUploadTime = uploadStartTime
			var smoothedUploadSpeed float64

			uploadRunning := true
			for uploadRunning {
				select {
				case <-uploadCtx.Done():
					uploadRunning = false
				case now := <-uploadTicker.C:
					currentBytes := atomic.LoadInt64(&totalUploadBytes)
					elapsed := now.Sub(uploadStartTime)
					if elapsed > t.cfg.Duration {
						elapsed = t.cfg.Duration
					}

					deltaBytes := currentBytes - lastUploadBytes
					deltaTime := now.Sub(lastUploadTime).Seconds()

					var instantMbps float64
					if deltaTime > 0 {
						instantMbps = (float64(deltaBytes) * 8) / (deltaTime * 1_000_000)
					}

					if smoothedUploadSpeed == 0 {
						smoothedUploadSpeed = instantMbps
					} else {
						smoothedUploadSpeed = (0.75 * instantMbps) + (0.25 * smoothedUploadSpeed)
					}

					lastUploadBytes = currentBytes
					lastUploadTime = now

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
						Phase:             PhaseUploading,
						InstantSpeedMbps:  smoothedUploadSpeed,
						AverageSpeedMbps:  avgMbps,
						DownloadSpeedMbps: finalDownloadAvgMbps,
						UploadSpeedMbps:   avgMbps,
						BytesTransferred:  finalDownloadBytes + currentBytes,
						DownloadBytes:     finalDownloadBytes,
						UploadBytes:       currentBytes,
						Elapsed:           elapsed,
						TotalDuration:     t.cfg.Duration,
						Percent:           percent,
						Latency:           currentLatency,
						Proxy:             t.cfg.Proxy,
						Client:            &clientInfo,
						Targets:           targets,
					}
				}
			}

			uploadWg.Wait()

			finalUploadBytes = atomic.LoadInt64(&totalUploadBytes)
			totalUploadElapsed := time.Since(uploadStartTime)
			if totalUploadElapsed > t.cfg.Duration {
				totalUploadElapsed = t.cfg.Duration
			}
			if totalUploadElapsed.Seconds() > 0 {
				finalUploadAvgMbps = (float64(finalUploadBytes) * 8) / (totalUploadElapsed.Seconds() * 1_000_000)
			}
		}

		finalLatency := time.Duration(atomic.LoadInt64(&latencyAtomic))
		totalOverallElapsed := time.Since(overallStartTime)
		ch <- Progress{
			Phase:             PhaseCompleted,
			InstantSpeedMbps:  finalDownloadAvgMbps,
			AverageSpeedMbps:  finalDownloadAvgMbps,
			DownloadSpeedMbps: finalDownloadAvgMbps,
			UploadSpeedMbps:   finalUploadAvgMbps,
			BytesTransferred:  finalDownloadBytes + finalUploadBytes,
			DownloadBytes:     finalDownloadBytes,
			UploadBytes:       finalUploadBytes,
			Elapsed:           totalOverallElapsed,
			TotalDuration:     totalOverallElapsed,
			Percent:           1.0,
			Latency:           finalLatency,
			Proxy:             t.cfg.Proxy,
			Client:            &clientInfo,
			Targets:           targets,
		}
	}()

	return ch
}
