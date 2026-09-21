# go-fast-cli

A simple, fast, and elegant command-line tool written in Go to test your internet speed using Netflix's **Fast.com** service, featuring a real-time Terminal User Interface (TUI) powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- **Instant Connection**: Uses Fast.com's API with dynamic fallback token retrieval.
- **Full Dual-Direction Testing**: Measures both download and upload speeds with concurrent multi-stream throughput.
- **Interactive TUI**: Live animated spinner, progress bar, real-time speed meter, bytes counter, latency probe, and completion summary card.
- **Accurate Throughput**: Multi-stream concurrent transfers with exponential moving average (EMA) smoothing and atomic byte accounting.
- **Multiple Output Modes**:
  - Full interactive TUI (default)
  - Non-interactive plain text (`--simple`) for CI / logs
  - Structured JSON (`--json`) for scripting and automation
- **Clean Interrupts**: Abort gracefully anytime with `q`, `Esc`, or `Ctrl+C`.

---

## Installation

### Via Homebrew (macOS / Linux)

Install directly:
```bash
brew install orekasep/tap/go-fast-cli
```

Or tap first, then install:
```bash
brew tap orekasep/tap
brew install go-fast-cli
```

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
Downloading... 24.50 Mbps | 15.3 MB | 100%
Uploading...   18.20 Mbps | 11.4 MB | 100%

=== Results ===
Download Speed:   22.80 Mbps
Upload Speed:     18.20 Mbps
Latency (RTT):    18 ms
Data Transferred: 26.7 MB (down: 15.3 MB, up: 11.4 MB)
Total Time:       10.5s
Provider:         Viettel (171.253.233.49)
Client Location:  Phu Dien, VN
Server Node:      Singapore, SG
```

### 3. JSON Output (for scripts)
```bash
./fast -json -duration 5s
```
Output:
```json
{
  "download_speed_mbps": 22.8,
  "upload_speed_mbps": 18.2,
  "latency_ms": 18,
  "total_bytes": 28000000,
  "download_bytes": 16000000,
  "upload_bytes": 12000000,
  "duration_sec": 10,
  "client": {
    "ip": "171.253.233.49",
    "isp": "Viettel",
    "location": {
      "city": "Phu Dien",
      "country": "VN"
    }
  },
  "targets": [
    ...
  ]
}
```

---

## CLI Options

| Flag | Shorthand | Default | Description |
|---|---|---|---|
| `-duration` | `-d` | `10s` | Test duration per phase (e.g., `5s`, `10s`, `15s`) |
| `-threads` | `-t` | `4` | Number of concurrent streams |
| `-urls` | | `5` | Number of CDN target servers to query |
| `-upload` | `-u` | `true` | Measure upload speed in addition to download |
| `-no-upload` | | `false` | Disable upload speed test |
| `-proxy` | `-p` | `""` | Proxy URL (`http://`, `https://`, `socks5://`) |
| `-simple` | | `false` | Output simple text line (no TUI) |
| `-json` | | `false` | Output test results as JSON |
| `-version` | `-v` | `false` | Print version and exit |

### Proxy Support

You can route all traffic through an HTTP, HTTPS, or SOCKS5 proxy via the `-proxy` flag:

```bash
# HTTP proxy
fast -proxy http://127.0.0.1:8080

# Authenticated HTTP proxy
fast -proxy http://user:pass@127.0.0.1:8080

# SOCKS5 proxy
fast -proxy socks5://127.0.0.1:1080
```

Standard proxy environment variables (`HTTP_PROXY`, `HTTPS_PROXY`, `ALL_PROXY`) are also respected automatically when `-proxy` is not specified.

---

## Architecture

- [`pkg/fast/client.go`](pkg/fast/client.go): Fast.com API integration, CDN target querying, latency probe, and unit formatting.
- [`pkg/fast/tester.go`](pkg/fast/tester.go): Multi-stream download and upload engine, atomic rate accounting, connection keep-alive reuse, and progress channel stream.
- [`pkg/ui/tui.go`](pkg/ui/tui.go): Bubble Tea state machine, animated progress bar, responsive layouts, and dual-speed results card.
- [`main.go`](main.go): CLI flags and execution runner.
