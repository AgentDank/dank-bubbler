# dank-bubbler

A BubbleTea-based TUI component suite for AgentDank tools, featuring terminal charting with NTCharts.

## Overview

This project helps integrate AgentDank functionality into Golang [BubbleTea](https://github.com/charmbracelet/bubbletea) GUIs. We're building out a component suite starting with a tech demo, then extracting reusable components.

## Projects

### Brand Demo

A demo application showcasing a product browser for the brands database table.

**Features:**
- Product browsing by brand, name, cannabis type, or date
- Info pane displaying selected product details
- NTCharts horizontal bar chart showing top 8 cannabinoids for the product
- Data source: [dank-data repository](https://github.com/AgentDank/dank-data)

**Current Dataset:**
- Database: Connecticut brands (2025-04-03)
- Format: ZST-compressed DuckDB
- URL: https://github.com/AgentDank/dank-data/blob/main/snapshots/us/ct/2025-04-03/us_ct_brands.duckdb.zst

## Dependencies

- [BubbleTea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [NTCharts](https://github.com/NimbleMarkets/ntcharts) - Terminal charting library (maintained by us)
- DuckDB - Database engine for data queries

## Development

Refer to [PROMPTS.md](PROMPTS.md) for the complete development history and prompts used.
Refer to [AGENTS.md](AGENTS.md) for agent configuration and maintenance notes.
