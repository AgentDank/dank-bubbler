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

### 2026-01-09T22:50:00Z - Correct Table Name and Schema
you are using the table "brands" when it is "us_ct_brands" please correct that and also look at the us_ct_brands table structure

### 2026-01-09T22:55:00Z - Add Horizontal Bar Chart Panel
now put a panel in the lower right that is a horizontal barchart of the top 6 cannabinoids or terpenes of the selected product. Use NTCharts for this

### 2026-04-16T17:31:18Z - Review Sibling Repos For Latest State
it's been a while since i've worked on this repo.   look at the ../dank-data and ../dank-extract projects for the latest over there.   then we will work on expanding this demo

### 2026-04-16T17:31:18Z - Sync Demo To Current Data Contract
yes do that sync-up

### 2026-04-16T17:44:50Z - Commit Cleanup And Sync Work
ok go ahead and commit that

### 2026-04-16T18:44:55Z - Implement Real Filter UX
i fixed the test. proceed with #2

### 2026-04-16T18:52:03Z - Remove Product Browser Row Cap
go ahead

### 2026-04-16T18:53:27Z - Commit Checkpoint Then Continue
commit first then proceed

### 2026-04-16T19:09:28Z - Commit Browse Modes Checkpoint And Fix Lint
commit this as a checkpoint and then fix any linter errors

### 2026-04-16T19:14:26Z - Commit Lint Fixes
commit them
