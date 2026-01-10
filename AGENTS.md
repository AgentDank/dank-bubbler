# Agent Configuration & Maintenance

This file tracks agent operations and project maintenance notes.

## Key Directives

- **Prompt History**: Maintained in [PROMPTS.md](PROMPTS.md) with ISO 8601 timestamps
- **Documentation**: See [README.md](README.md) for project overview
- **Process**: Always append new prompts to PROMPTS.md; never delete history

# dank-bubbler Project

This is the folder `dank-bubbler` which is the name of this project.  It is part of AgentDank's suite of tools and this one helps use AgentDank in TUIs, particularly Golang [BubbleTea](https://github.com/charmbracelet/bubbletea) GUIs.  [NTCharts](https://github.com/NimbleMarkets/ntcharts) is our terminal charting library which we maintain and should use.

We will build out the component suite by creating a tech demo and then separating out the relevant components for other programs to use.

## brand demo

This is a demo of the brands database table.  Create a product browser based on the common selected aspect, such as brand, name, cannabis type, date.

Then we will have an info pane that shows the information for the selected product.

We will have an NTCharts horizontal bar chart showing the top 8 cannabinoids for the product.

we will load our data from the dank-data repo.  Here's the current dataset for brands:
https://github.com/AgentDank/dank-data/blob/main/snapshots/us/ct/2025-04-03/us_ct_brands.duckdb.zst

It is a ZST compressed DuckDB database.  We can use that as the source.
