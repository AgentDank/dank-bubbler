# Prompt Zero

This is Evan's first prompt for this project.  You can erase this content after you copy it to PROMPTS.md.  Create an AGENTS.md and a README.md file for further maintenance

# Prompt Archive

You will maintain a PROMPTS.md file with timestamps in ISO 8601 format.  You will always append the last user prompt to this.  You will not delete prompt history.

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
