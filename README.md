# lx (Log X-Ray)

A fast, ephemeral log viewer with offline analytics. Paste a prod error, get clarity in 30 seconds.

![lx demo](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

## Features

- **Zero Config** - No setup, no database, no state. Just logs.
- **Smart Parsing** - JSON, plain text, stack traces auto-detected
- **Powerful Filtering** - Inclusive and exclusive patterns with AND logic
- **Level Filter** - Quick filtering by log level (ERROR/WARN/INFO/DEBUG)
- **Signal Booster** - Offline analytics for error patterns
- **Notes** - Annotate log lines during investigation
- **HTTP Lookup** - Quick reference for HTTP status codes
- **Vim-style Navigation** - j/k, g/G, familiar keybindings

## Installation

### From Source

```bash
# Clone and build
git clone https://github.com/kalayciburak/lx
cd lx
go build -o lx ./cmd/lx

# Optional: Install globally
sudo mv lx /usr/local/bin/
```

### Go Install

```bash
go install github.com/kalayciburak/lx/cmd/lx@latest
```

### Requirements

- Go 1.21+
- Terminal with Unicode support

## Usage

### Read from File

```bash
lx app.log
lx /var/log/nginx/error.log
```

### Pipe from Command

```bash
cat logs.txt | lx
kubectl logs pod-name | lx
docker logs container | lx
tail -f app.log | lx
```

### Interactive Mode

```bash
lx
# Then press 'p' to paste from clipboard
```

## Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `k` | Move down / up |
| `g` / `G` | Jump to top / bottom |
| `Enter` | Toggle detail view |

### Filter

| Key | Action |
|-----|--------|
| `/` | Open filter modal |
| `Tab` | Cycle level filter (ALL/ERROR/WARN/INFO/DEBUG) |
| `ESC` | Apply filter / exit |

### Signal Booster (Analytics)

| Key | Action |
|-----|--------|
| `1` | Error frequency - top errors by count |
| `2` | Lifetime - first/last seen for selected message |
| `3` | Burst detector - detect error spikes |
| `4` | Diversity - error variety analysis |

### Notes

| Key | Action |
|-----|--------|
| `N` | Write/edit note for current line |
| `n` | Toggle note display |
| `m` | Show/hide all notes |
| `]` / `[` | Jump to next/prev noted line |

### Copy & Edit

| Key | Action |
|-----|--------|
| `c` | Copy current line (with note if exists) |
| `y` | Yank all visible logs + notes |
| `p` | Paste from clipboard |
| `d` | Delete current line |
| `x` | Clear all |

### Tools

| Key | Action |
|-----|--------|
| `Ctrl+L` | HTTP status code lookup |
| `?` | Toggle help |
| `q` | Quit |

## Filter Syntax

| Pattern | Meaning |
|---------|---------|
| `error` | Lines containing "error" |
| `!debug` | Lines NOT containing "debug" |
| `error timeout` | Lines with both terms (AND) |
| `error !debug` | "error" but NOT "debug" |

All filters are **case-insensitive**.

## Signal Booster

Offline analytics that run entirely in RAM - no network, no database.

### 1. Error Frequency (`1`)
Shows top 10 most frequent ERROR messages:
```
TOP ERROR SIGNALS

timeout contacting downstream    x12
HTTP 503 Service Unavailable      x9
connection pool exhausted         x4
```

### 2. Lifetime Analysis (`2`)
Shows when a specific error first/last appeared:
```
SIGNAL LIFETIME

Message: connection timeout
First seen: 2024-01-15T10:30:45Z
Last seen:  2024-01-15T10:45:22Z
Occurrences: 15
```

### 3. Burst Detector (`3`)
Detects unusual error spikes:
```
BURST ANALYSIS

Message: database connection failed
BURST DETECTED
8 occurrences in 10s window
```

### 4. Error Diversity (`4`)
Analyzes error variety:
```
ERROR DIVERSITY

Total ERROR lines:     45
Unique ERROR messages: 3

Signal quality: HIGH
Repetitive errors - clear pattern
```

## Notes System

Annotate log lines during investigation:

1. Press `N` on any line to add a note
2. Press `n` to toggle note visibility
3. Press `m` to show/hide all notes
4. Notes are included when copying with `y` or `c`

Export format:
```
=== NOTE (lx) ===
• [line 42] This is the root cause

=== LOG ===
line 42: {"level":"error","msg":"connection refused"}
```

## Supported Log Formats

### JSON Logs
```json
{"msg":"connection failed","level":"error","service":"api"}
{"message":"request completed","severity":"info","duration_ms":42}
```

### Plain Text
```
2024-01-15 10:30:45 ERROR Database connection timeout
[WARN] Disk space low on /dev/sda1
```

### Stack Traces
Auto-detected patterns:
- Java: `at com.example.Foo.bar(Foo.java:123)`
- Go: `main.go:45`, `goroutine 1 [running]`
- Python: `File "/app/main.py", line 42`

## Philosophy

**lx is not a log viewer. It's a thinking accelerator.**

When debugging at 3 AM, you don't need:
- A database to query later
- Config files to manage
- State to persist

You need:
1. Copy error from Slack
2. Understand what happened
3. Add notes during investigation
4. Copy relevant logs with annotations
5. Paste into incident report
6. Move on

**Input -> RAM -> Brain -> Clipboard -> Done.**

No setup. No cleanup. No "let me save this for later."

Parse the chaos. Find the signal. Copy. Exit. Sleep.

## License

MIT

## Author

[kalayciburak](https://github.com/kalayciburak)
