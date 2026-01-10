# Development

## Project Structure

```
dank-bubbler/
├── cmd/
│   └── brand-demo/        # Brand product browser demo
├── internal/
│   ├── data/              # Data loading and DuckDB integration
│   ├── models/            # Data models
│   └── ui/                # BubbleTea UI components
├── assets/                # Static assets and data
├── tests/                 # Test fixtures and data
├── dankbubbler.go         # Main package definition
├── go.mod
├── Makefile
└── README.md
```

## Setup

1. **Initialize Go module** (already done):
   ```bash
   go mod init github.com/AgentDank/dank-bubbler
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Build the brand-demo**:
   ```bash
   make build-demo
   # or
   go build -o ./bin/db-brand-demo ./cmd/brand-demo
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
2. Implement data loader for brands database
3. Build ProductBrowser UI component
4. Implement cannabinoid chart using NTCharts
5. Create filtering views (by brand, type, date)
6. Build info pane for product details
