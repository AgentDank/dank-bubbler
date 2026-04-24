# dank-bubbler

A BubbleTea-based TUI component suite for AgentDank tools, featuring terminal charting with NTCharts.

## Overview

This project helps integrate AgentDank functionality into Golang [BubbleTea](https://github.com/charmbracelet/bubbletea) GUIs. We're building out a component suite starting with a tech demo, then extracting reusable components.

## Projects

### Product Demo

A demo application showcasing a product browser for the Connecticut cannabis
dataset.

**Features:**
- Product browsing by brand, name, cannabis type, or date
- Info pane displaying selected product details
- NTCharts horizontal bar chart showing the top 6 compounds for the product
- Data source: [dank-data repository](https://github.com/AgentDank/dank-data)

**Current Dataset:**
- Database: latest Connecticut cannabis snapshot
- Format: ZST-compressed DuckDB
- URL: https://github.com/AgentDank/dank-data/blob/main/snapshots/us/ct/dank-data.duckdb.zst

## Development

Refer to [DEVELOPMENT.md](DEVELOPMENT.md) for development setup, available tasks, and contribution guidelines.

Use [Task](https://taskfile.dev) to automate common workflows:
```bash
task              # List all available tasks
task build        # Build all tools
task test         # Run tests
task fmt          # Format code
task lint         # Run linter
task run -- --db <path>  # Run dank-bubbler-ct
```

## Dependencies

- [BubbleTea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [NTCharts](https://github.com/NimbleMarkets/ntcharts) - Terminal charting library (maintained by us)
- DuckDB - Database engine for data queries

## Development

Refer to [AGENTS.md](AGENTS.md) for agent configuration and maintenance notes.
