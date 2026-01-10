# Prompt Archive

This file maintains a history of all prompts used to develop the dank-bubbler project. All entries include ISO 8601 timestamps.

## Prompt History

### 2026-01-09T00:00:00Z - Prompt Zero
read AGENTS.md and operate the prompt there until you need more information from me

### 2026-01-09T22:16:00Z - Project Structure Setup
create the project structure.  you can model it after [ollamatea](https://github.com/NimbleMarkets/ollamatea) which is a similar project but for ollama inferencing and bubbletea components.    this project will be module `github.com/AgentDank/dank-bubbler`

### 2026-01-09T22:30:00Z - Taskfile Migration
I prefer Taskfile to makefile.  convert the makefile to a taskfile and update the documentation

### 2026-01-09T22:35:00Z - Auto Database Download
the --db path option is optional and should default to `dank-data.duckdb`.  If there is no brands table present, it should be downloaded from the https://github.com/AgentDank/dank-data/blob/main/snapshots/us/ct/2025-04-03/us_ct_brands.duckdb.zst

### 2026-01-09T22:40:00Z - Core Implementation Complete
go ahead and implement these next steps:
- Implement DuckDB table existence check
- Add data loader to query the brands table
- Build the UI with ProductBrowser component
