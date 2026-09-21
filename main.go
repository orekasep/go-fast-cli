package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"go-fast-cli/pkg/fast"
	"go-fast-cli/pkg/ui"
)

const version = "1.0.1"

func main() {
	var (
		durationFlag time.Duration
		threadsFlag  int
		urlCountFlag int
		proxyFlag    string
		simpleFlag   bool
		jsonFlag     bool
		versionFlag  bool
	)

	flag.DurationVar(&durationFlag, "duration", 10*time.Second, "Test duration (e.g., 5s, 10s)")
	flag.DurationVar(&durationFlag, "d", 10*time.Second, "Test duration (shorthand)")
	flag.IntVar(&threadsFlag, "threads", 4, "Number of concurrent download streams")
	flag.IntVar(&threadsFlag, "t", 4, "Number of concurrent download streams (shorthand)")
	flag.IntVar(&urlCountFlag, "urls", 5, "Number of CDN target servers to request")
	flag.StringVar(&proxyFlag, "proxy", "", "Proxy URL (e.g., http://127.0.0.1:8080, socks5://127.0.0.1:1080)")
	flag.StringVar(&proxyFlag, "p", "", "Proxy URL (shorthand)")
	flag.BoolVar(&simpleFlag, "simple", false, "Output in simple text format (no TUI)")
	flag.BoolVar(&jsonFlag, "json", false, "Output results as JSON")
	flag.BoolVar(&versionFlag, "version", false, "Print version and exit")
	flag.BoolVar(&versionFlag, "v", false, "Print version and exit (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of go-fast-cli:\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  go-fast-cli [options]\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if versionFlag {
		fmt.Printf("go-fast-cli v%s\n", version)
		os.Exit(0)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := fast.Config{
		Duration: durationFlag,
		Threads:  threadsFlag,
		URLCount: urlCountFlag,
		Proxy:    proxyFlag,
	}
	tester := fast.NewTester(cfg)

	// Mode 1: JSON output
	if jsonFlag {
		runJSON(ctx, tester, cfg)
		return
	}

	// Mode 2: Simple plain-text output
	if simpleFlag {
		runSimple(ctx, tester)
		return
	}

	// Mode 3: Interactive Bubble Tea TUI
	runTUI(ctx, stop, tester)
}

func runTUI(ctx context.Context, cancel context.CancelFunc, tester *fast.Tester) {
	model := ui.NewModel(ctx, cancel, tester)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func runSimple(ctx context.Context, tester *fast.Tester) {
	fmt.Println("Connecting to Fast.com...")
	ch := tester.Run(ctx)

	var lastProgress fast.Progress
	for p := range ch {
		lastProgress = p
		if p.Err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", p.Err)
			os.Exit(1)
		}

		if p.Phase == fast.PhaseTesting {
			fmt.Printf("\rTesting... %s | %s | %.0f%%",
				fast.FormatSpeed(p.InstantSpeedMbps),
				fast.FormatBytes(p.BytesTransferred),
				p.Percent*100)
		}
	}

	fmt.Printf("\n\n=== Results ===\n")
	fmt.Printf("Download Speed:   %s\n", fast.FormatSpeed(lastProgress.AverageSpeedMbps))
	if lastProgress.Latency > 0 {
		fmt.Printf("Latency (RTT):    %d ms\n", lastProgress.Latency.Milliseconds())
	}
	fmt.Printf("Data Transferred: %s\n", fast.FormatBytes(lastProgress.BytesTransferred))
	fmt.Printf("Total Time:       %.1fs\n", lastProgress.Elapsed.Seconds())
	if lastProgress.Client != nil {
		fmt.Printf("Provider:         %s (%s)\n", lastProgress.Client.ISP, lastProgress.Client.IP)
		fmt.Printf("Client Location:  %s, %s\n", lastProgress.Client.Location.City, lastProgress.Client.Location.Country)
	}
	if len(lastProgress.Targets) > 0 {
		fmt.Printf("Server Node:      %s, %s\n", lastProgress.Targets[0].Location.City, lastProgress.Targets[0].Location.Country)
	}
	if lastProgress.Proxy != "" {
		fmt.Printf("Proxy:            %s\n", lastProgress.Proxy)
	}
}

func runJSON(ctx context.Context, tester *fast.Tester, cfg fast.Config) {
	ch := tester.Run(ctx)

	var lastProgress fast.Progress
	for p := range ch {
		lastProgress = p
		if p.Err != nil {
			errMap := map[string]string{"error": p.Err.Error()}
			_ = json.NewEncoder(os.Stderr).Encode(errMap)
			os.Exit(1)
		}
	}

	var client fast.ClientInfo
	if lastProgress.Client != nil {
		client = *lastProgress.Client
	}

	summary := fast.TestSummary{
		DownloadSpeed: lastProgress.AverageSpeedMbps,
		Latency:       time.Duration(lastProgress.Latency.Milliseconds()),
		TotalBytes:    lastProgress.BytesTransferred,
		Duration:      time.Duration(lastProgress.Elapsed.Seconds()),
		Proxy:         cfg.Proxy,
		Client:        client,
		Targets:       lastProgress.Targets,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(summary); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode JSON: %v\n", err)
		os.Exit(1)
	}
}
