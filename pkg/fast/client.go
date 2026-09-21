package fast

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

const (
	// DefaultToken is the known fallback token for Fast.com API.
	DefaultToken = "YXNkZmFzZGxmbnNkYWZoYXNkZmhrYWxm"
	fastURL      = "https://fast.com"
	apiBaseURL   = "https://api.fast.com/netflix/speedtest/v2"
	userAgent    = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

var (
	jsAppRegex = regexp.MustCompile(`app-[a-f0-9]+\.js`)
	tokenRegex = regexp.MustCompile(`token:"([a-zA-Z0-9]+)"`)
)

// Location contains geographical info about client or server.
type Location struct {
	City    string `json:"city"`
	Country string `json:"country"`
}

// ClientInfo contains metadata about the client connection.
type ClientInfo struct {
	IP       string   `json:"ip"`
	ASN      string   `json:"asn"`
	ISP      string   `json:"isp"`
	Location Location `json:"location"`
}

// TargetServer represents a Netflix Open Connect CDN speed test target.
type TargetServer struct {
	Name     string   `json:"name"`
	URL      string   `json:"url"`
	Location Location `json:"location"`
}

// SpeedtestResponse is the payload returned by api.fast.com.
type SpeedtestResponse struct {
	Client  ClientInfo     `json:"client"`
	Targets []TargetServer `json:"targets"`
}

// TestSummary summarizes the result of an internet speed test.
type TestSummary struct {
	DownloadSpeed float64        `json:"download_speed_mbps"`
	Latency       time.Duration  `json:"latency_ms"`
	TotalBytes    int64          `json:"total_bytes"`
	Duration      time.Duration  `json:"duration_sec"`
	Client        ClientInfo     `json:"client"`
	Targets       []TargetServer `json:"targets"`
}

// GetToken attempts to dynamically fetch the Fast.com API token from the app JS bundle.
// If extraction fails for any reason, it returns the reliable DefaultToken.
func GetToken(ctx context.Context) string {
	client := &http.Client{Timeout: 4 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fastURL, nil)
	if err != nil {
		return DefaultToken
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return DefaultToken
	}
	defer resp.Body.Close()

	htmlBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return DefaultToken
	}

	jsMatch := jsAppRegex.Find(htmlBytes)
	if len(jsMatch) == 0 {
		return DefaultToken
	}

	jsURL := fmt.Sprintf("%s/%s", fastURL, string(jsMatch))
	jsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, jsURL, nil)
	if err != nil {
		return DefaultToken
	}
	jsReq.Header.Set("User-Agent", userAgent)

	jsResp, err := client.Do(jsReq)
	if err != nil {
		return DefaultToken
	}
	defer jsResp.Body.Close()

	jsBytes, err := io.ReadAll(jsResp.Body)
	if err != nil {
		return DefaultToken
	}

	tokenMatch := tokenRegex.FindSubmatch(jsBytes)
	if len(tokenMatch) > 1 {
		return string(tokenMatch[1])
	}

	return DefaultToken
}

// GetSpeedtestTargets queries the Fast.com API for available download targets and client metadata.
// It tries with the default token first for instant response, and dynamically fetches a fresh token if unauthorized.
func GetSpeedtestTargets(ctx context.Context, token string, count int) (*SpeedtestResponse, error) {
	if token == "" {
		token = DefaultToken
	}
	if count <= 0 {
		count = 5
	}

	res, err := fetchTargets(ctx, token, count)
	if err == nil {
		return res, nil
	}

	// If failed, try fetching a dynamic token from fast.com and retry
	freshToken := GetToken(ctx)
	if freshToken != token && freshToken != "" {
		return fetchTargets(ctx, freshToken, count)
	}

	return nil, err
}

func fetchTargets(ctx context.Context, token string, count int) (*SpeedtestResponse, error) {
	endpoint := fmt.Sprintf("%s?https=true&token=%s&urlCount=%d", apiBaseURL, token, count)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build targets request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query Fast.com API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Fast.com API returned HTTP %d", resp.StatusCode)
	}

	var data SpeedtestResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode Fast.com API response: %w", err)
	}

	if len(data.Targets) == 0 {
		return nil, fmt.Errorf("no test targets returned from Fast.com")
	}

	return &data, nil
}

// MeasureLatency measures the round-trip latency to a target URL.
func MeasureLatency(ctx context.Context, targetURL string) (time.Duration, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, targetURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", userAgent)

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		// Fallback to GET with small Range if HEAD fails
		reqGet, errGet := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if errGet != nil {
			return 0, err
		}
		reqGet.Header.Set("Range", "bytes=0-0")
		reqGet.Header.Set("User-Agent", userAgent)

		start = time.Now()
		respGet, errGet := client.Do(reqGet)
		if errGet != nil {
			return 0, errGet
		}
		defer respGet.Body.Close()
		return time.Since(start), nil
	}
	defer resp.Body.Close()

	return time.Since(start), nil
}

// FormatSpeed returns a human-readable speed formatted with units (e.g. "12.34 Mbps").
func FormatSpeed(mbps float64) string {
	if mbps >= 1000 {
		return fmt.Sprintf("%.2f Gbps", mbps/1000)
	}
	if mbps >= 1 {
		return fmt.Sprintf("%.2f Mbps", mbps)
	}
	return fmt.Sprintf("%.2f Kbps", mbps*1000)
}

// FormatBytes returns human-readable byte sizes (e.g. "45.2 MB").
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
