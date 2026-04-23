# Development

## Project Structure

```
dank-bubbler/
├── cmd/
│   └── dank-bubbler-ct/   # CT cannabis data browser
├── internal/
│   ├── data/              # Data loading and DuckDB integration
│   ├── models/            # Data models
│   └── ui/                # BubbleTea UI components
├── assets/                # Static assets and data
├── tests/                 # Test fixtures and data
├── go.mod
├── Taskfile.yaml          # Task automation
└── README.md
```

## Setup

### Prerequisites

- Go 1.23 or later
- [Task](https://taskfile.dev) - Install with: `brew install go-task` or `choco install go-task` or `apt-get install go-task`

### Getting Started

1. **Initialize Go module** (already done):
   ```bash
   go mod init github.com/AgentDank/dank-bubbler
   ```

2. **Install dependencies**:
   ```bash
   task tidy
   ```

3. Task Commands

| Task | Description |
|------|-------------|
| `task` | Show available tasks (default) |
| `task build` | Build all tools |
| `task build-demo` | Build dank-bubbler-ct tool |
| `task clean` | Remove build artifacts |
| `task install` | Install tools into $GOPATH/bin |
| `task test` | Run tests |
| `task lint` | Run golangci-lint |
| `task fmt` | Format code |
| `task tidy` | Tidy go.mod |
| `task run -- --db <path>` | Run dank-bubbler-ct with database path |

## **Build the dank-bubbler-ct**:
   ```bash
   task build-demo
   # or run all builds
   task build
   ```

4. **List available tasks**:
   ```bash
   task
   # or
   task -l
   ```

## Dependencies

### Core
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/bubbles` - BubbleTea components
- `github.com/charmbracelet/lipgloss` - Styling

### Charts
- `github.com/NimbleMarkets/ntcharts` - Terminal charting library

### Data
- DuckDB (via Go driver - to be added)

### CLI
- `github.com/spf13/pflag` - Flag parsing

## Next Steps

1. Set up DuckDB Go driver
2. Implement data loader for products database
3. Build ProductBrowser UI component
4. Implement compound chart using NTCharts
5. Create filtering views (by brand, type, date)
6. Build info pane for product details
