package fast

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFormatSpeed(t *testing.T) {
	tests := []struct {
		mbps     float64
		expected string
	}{
		{0.5, "500.00 Kbps"},
		{10.25, "10.25 Mbps"},
		{150.0, "150.00 Mbps"},
		{1250.5, "1.25 Gbps"},
	}

	for _, tt := range tests {
		result := FormatSpeed(tt.mbps)
		if result != tt.expected {
			t.Errorf("FormatSpeed(%f) = %s; want %s", tt.mbps, result, tt.expected)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		result := FormatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("FormatBytes(%d) = %s; want %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestGetSpeedtestTargetsMock(t *testing.T) {
	mockResponse := SpeedtestResponse{
		Client: ClientInfo{
			IP:  "1.2.3.4",
			ASN: "12345",
			ISP: "Test ISP",
			Location: Location{
				City:    "Test City",
				Country: "TC",
			},
		},
		Targets: []TargetServer{
			{
				Name: "target1",
				URL:  "https://example.com/speedtest",
				Location: Location{
					City:    "Server City",
					Country: "SC",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Direct decode test of SpeedtestResponse
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error creating request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected error requesting mock: %v", err)
	}
	defer resp.Body.Close()

	var data SpeedtestResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if data.Client.ISP != "Test ISP" {
		t.Errorf("got ISP %s, want 'Test ISP'", data.Client.ISP)
	}
	if len(data.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(data.Targets))
	}
	if data.Targets[0].Name != "target1" {
		t.Errorf("got target %s, want 'target1'", data.Targets[0].Name)
	}
}

func TestDefaultTokenFallback(t *testing.T) {
	// Cancelled context should immediately fall back to DefaultToken without panic
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	token := GetToken(ctx)
	if token != DefaultToken {
		t.Errorf("GetToken with cancelled context = %s; want DefaultToken %s", token, DefaultToken)
	}
}

func TestCreateHTTPClientProxy(t *testing.T) {
	// Test HTTP proxy
	client, err := CreateHTTPClient("http://127.0.0.1:8080", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error creating HTTP proxy client: %v", err)
	}
	if client == nil || client.Transport == nil {
		t.Fatal("expected non-nil client and transport")
	}

	// Test SOCKS5 proxy
	socksClient, err := CreateHTTPClient("socks5://127.0.0.1:1080", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error creating SOCKS5 proxy client: %v", err)
	}
	if socksClient == nil || socksClient.Transport == nil {
		t.Fatal("expected non-nil socks5 client and transport")
	}

	// Test Invalid proxy URL
	_, err = CreateHTTPClient("://invalid-url", 5*time.Second)
	if err == nil {
		t.Fatal("expected error for invalid proxy URL, got nil")
	}
}
