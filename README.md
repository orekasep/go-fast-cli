# go-fast-cli

A simple, fast, and elegant command-line tool written in Go to test your internet speed using Netflix's **Fast.com** service, featuring a real-time Terminal User Interface (TUI) powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- **Instant Connection**: Uses Fast.com's API with dynamic fallback token retrieval.
- **Interactive TUI**: Live animated spinner, progress bar, real-time speed meter, bytes counter, latency probe, and completion summary card.
- **Accurate Throughput**: Multi-stream concurrent downloads with exponential moving average (EMA) smoothing and atomic byte accounting.
- **Multiple Output Modes**:
  - Full interactive TUI (default)
  - Non-interactive plain text (`--simple`) for CI / logs
  - Structured JSON (`--json`) for scripting and automation
- **Clean Interrupts**: Abort gracefully anytime with `q`, `Esc`, or `Ctrl+C`.

---

## Installation

### Download Prebuilt Binaries (Windows, macOS, Linux)
Pre-compiled standalone binaries for **Windows** (x86_64, ARM64), **macOS** (Apple Silicon, Intel), and **Linux** (x86_64, ARM64) are available on the [**Releases Page**](https://github.com/orekasep/go-fast-cli/releases/latest).

### Build from Source
```bash
git clone https://github.com/orekasep/go-fast-cli.git
cd go-fast-cli
go build -o fast .
```

---

## Usage

### 1. Interactive TUI (Default)
Run a speed test with the live interactive interface:
```bash
./fast
```

Custom duration (e.g. 5 seconds) and concurrency:
```bash
./fast -duration 5s -threads 4
```

### 2. Simple Output (for standard logs / dumb terminals)
```bash
./fast -simple -duration 5s
```
Output:
```text
Connecting to Fast.com...
Testing... 24.50 Mbps | 6.2 MB | 60%

=== Results ===
Download Speed:   22.80 Mbps
Latency (RTT):    18 ms
Data Transferred: 14.2 MB
Total Time:       5.0s
Provider:         Viettel (171.253.233.49)
Client Location:  Phu Dien, VN
Server Node:      Singapore, SG
```

### 3. JSON Output (for scripts)
```bash
./fast -json -duration 5s
```

---

## CLI Options

| Flag | Shorthand | Default | Description |
|---|---|---|---|
| `-duration` | `-d` | `10s` | Test duration (e.g., `5s`, `10s`, `15s`) |
| `-threads` | `-t` | `4` | Number of concurrent download streams |
| `-urls` | | `5` | Number of CDN target servers to query |
| `-simple` | | `false` | Output simple text line (no TUI) |
| `-json` | | `false` | Output test results as JSON |
| `-version` | `-v` | `false` | Print version and exit |

---

## Architecture

- [`pkg/fast/client.go`](pkg/fast/client.go): Fast.com API integration, CDN target querying, latency probe, and unit formatting.
- [`pkg/fast/tester.go`](pkg/fast/tester.go): Multi-threaded download engine, atomic rate accounting, and progress channel stream.
- [`pkg/ui/tui.go`](pkg/ui/tui.go): Bubble Tea state machine, animated progress bar, responsive layouts, and results card.
- [`main.go`](main.go): CLI flags and execution runner.
