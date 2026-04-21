# Zoning & Retail Pages Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add two new Bubble Tea pages to dank-bubbler — a Zoning table (`ct_zoning`) and a Retail+Map page (`ct_retail_locations` + vendored `mapview`).

**Architecture:** Each page is a `*Browser` struct implementing `Init/Update/View` + `SetActivePage`, routed by `AppModel` via numbered tab keys `1..4` intercepted at the top level. Filter/sort logic is lifted into pure functions so it can be unit-tested without a Bubble Tea runtime. The Retail page embeds `mapview.Model` and uses a `tab`-toggled focus flag to decide whether keystrokes drive the list or the map.

**Tech Stack:** Go 1.26, `charm.land/bubbletea/v2`, `charm.land/bubbles/v2/table`, `charm.land/lipgloss/v2`, local `mapview` package (v2 port of `mrusme/mercator`), DuckDB via `github.com/duckdb/duckdb-go/v2`.

**Reference spec:** `docs/superpowers/specs/2026-04-21-zoning-and-retail-pages-design.md`.

**Scope note on loader tests:** The existing `internal/data/loader.go` has no tests. Adding DuckDB fixtures would add significant machinery for trivial SQL. This plan intentionally skips loader unit tests and relies on manual verification + the UI-layer tests. Revisit if loader behavior becomes non-trivial.

---

## Task 1: Add `ZoningRow` and `RetailLocation` models

**Files:**
- Modify: `internal/models/product.go`

- [ ] **Step 1: Append the two new struct types to `internal/models/product.go`**

Append these to the end of the file (keep them in `product.go` alongside `TaxRecord` / `SalesRecord` — file is still small; no need for a split):

```go
// ZoningRow is one row from ct_zoning. Empty Status represents a SQL NULL
// (rendered as "Unknown" in the UI).
type ZoningRow struct {
	Town   string
	Status string
}

// RetailLocation is one row from ct_retail_locations.
type RetailLocation struct {
	Type      string
	Business  string
	DBA       string
	License   string
	Street    string
	City      string
	Zipcode   string
	Website   string
	Longitude float64
	Latitude  float64
}
```

- [ ] **Step 2: Verify the package still builds**

Run: `go build ./internal/models/...`
Expected: exits 0, no output.

- [ ] **Step 3: Commit**

```bash
git add internal/models/product.go
git commit -m "models: add ZoningRow and RetailLocation"
```

---

## Task 2: Add `LoadZoning` and `LoadRetailLocations` to Loader

**Files:**
- Modify: `internal/data/loader.go`

- [ ] **Step 1: Add `LoadZoning` at the end of `internal/data/loader.go`**

Add this function just before the closing `nullableTime` helper:

```go
// LoadZoning returns every row from ct_zoning, ordered by town. NULL status
// values come back as empty strings.
func (l *Loader) LoadZoning() ([]models.ZoningRow, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}
	rows, err := l.db.Query(`
		SELECT town, COALESCE(status, '')
		FROM ct_zoning
		ORDER BY town
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query zoning: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []models.ZoningRow
	for rows.Next() {
		var r models.ZoningRow
		if err := rows.Scan(&r.Town, &r.Status); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
```

- [ ] **Step 2: Add `LoadRetailLocations` right after `LoadZoning`**

```go
// LoadRetailLocations returns every row from ct_retail_locations, ordered
// by business. Missing string fields come back as empty strings.
func (l *Loader) LoadRetailLocations() ([]models.RetailLocation, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}
	rows, err := l.db.Query(`
		SELECT
			COALESCE(type, ''),
			COALESCE(business, ''),
			COALESCE(dba, ''),
			COALESCE(license, ''),
			COALESCE(street, ''),
			COALESCE(city, ''),
			COALESCE(zipcode, ''),
			COALESCE(website, ''),
			COALESCE(longitude, 0),
			COALESCE(latitude, 0)
		FROM ct_retail_locations
		ORDER BY business
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query retail locations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []models.RetailLocation
	for rows.Next() {
		var r models.RetailLocation
		if err := rows.Scan(
			&r.Type, &r.Business, &r.DBA, &r.License,
			&r.Street, &r.City, &r.Zipcode, &r.Website,
			&r.Longitude, &r.Latitude,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
```

- [ ] **Step 3: Verify the loader package builds**

Run: `go build ./internal/data/...`
Expected: exits 0, no output.

- [ ] **Step 4: Commit**

```bash
git add internal/data/loader.go
git commit -m "data: add LoadZoning and LoadRetailLocations"
```

---

## Task 3: Finish `mapview` v2 port

The vendored `mapview/mapview.go` already has v2 import paths and a v2-shaped `View() tea.View`. Two remaining issues: `io/ioutil` is deprecated (Go 1.26 keeps it but we should use `io`), and the package's deps need to resolve via `go.mod` / `go.sum`.

**Files:**
- Modify: `mapview/mapview.go`
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Replace `ioutil` with `io` in `mapview/mapview.go`**

In the imports block of `mapview/mapview.go`, remove `"io/ioutil"` and add `"io"`.

In the body of `func (m *Model) lookup` (near line 309), change:

```go
body, err := ioutil.ReadAll(resp.Body)
```

to:

```go
body, err := io.ReadAll(resp.Body)
```

- [ ] **Step 2: Run `go mod tidy` to resolve mapview deps**

Run: `go mod tidy`
Expected: exits 0. `go.mod` / `go.sum` will gain entries for `flopp/go-staticmaps`, `eliukblau/pixterm`, `golang/geo`, and their transitive deps.

- [ ] **Step 3: Build the mapview package**

Run: `go build ./mapview/...`
Expected: exits 0, no output.

- [ ] **Step 4: Run the mapview tests**

Run: `go test ./mapview/...`
Expected: PASS. The existing tests in `mapview/mapview_test.go` exercise `New`, `Update(MapCoordinates{...})`, `Update(MapRender(...))`, and zoom-in bound checking.

- [ ] **Step 5: Commit**

```bash
git add mapview/mapview.go go.mod go.sum
git commit -m "mapview: finish v2 port (io.ReadAll, resolve deps)"
```

---

## Task 4: Verify full repo builds & tests pass on main branch

Pre-integration checkpoint. Nothing to change — just confirm the repo is green before we start adding pages.

- [ ] **Step 1: Build everything**

Run: `go build ./...`
Expected: exits 0.

- [ ] **Step 2: Run all tests**

Run: `go test ./...`
Expected: all PASS. If any existing test fails, STOP and diagnose before proceeding.

---

## Task 5: Add `PageZoning` and `PageRetail` enum values + tab strip

This lands the page constants and tab labels, with stub page models so the app still compiles. The real browsers slot in under later tasks.

**Files:**
- Modify: `internal/ui/app.go`

- [ ] **Step 1: Extend the `Page` enum**

In `internal/ui/app.go`, change:

```go
const (
	PageBrands Page = iota
	PageSalesTax
)
```

to:

```go
const (
	PageBrands Page = iota
	PageSalesTax
	PageZoning
	PageRetail
)
```

- [ ] **Step 2: Extend `pageTabs`**

Change:

```go
var pageTabs = []struct {
	key   string
	label string
}{
	{"1", "Brands"},
	{"2", "Sales & Tax"},
}
```

to:

```go
var pageTabs = []struct {
	key   string
	label string
}{
	{"1", "Brands"},
	{"2", "Sales & Tax"},
	{"3", "Zoning"},
	{"4", "Retail"},
}
```

- [ ] **Step 3: Verify the app still builds**

Run: `go build ./...`
Expected: exits 0. (No routing wired yet, so pressing `3` / `4` will do nothing — that's fine for now.)

- [ ] **Step 4: Run existing tests**

Run: `go test ./...`
Expected: all PASS. Tab-strip rendering may differ in existing layout tests — if so, update expected values.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/app.go
git commit -m "ui: add PageZoning and PageRetail to tab strip"
```

---

## Task 6: Pure helper — `recomputeZoning` (TDD)

Lift filter + sort into a standalone function so it's testable without a Bubble Tea runtime.

**Files:**
- Create: `internal/ui/zoning.go`
- Create: `internal/ui/zoning_test.go`

- [ ] **Step 1: Write the failing test first**

Create `internal/ui/zoning_test.go`:

```go
package ui

import (
	"reflect"
	"testing"

	"github.com/AgentDank/dank-bubbler/internal/models"
)

func TestRecomputeZoningFilter(t *testing.T) {
	all := []models.ZoningRow{
		{Town: "Ansonia", Status: "Approved"},
		{Town: "Avon", Status: "Prohibited"},
		{Town: "Bethany", Status: "Moratorium"},
		{Town: "Andover", Status: ""}, // NULL -> Unknown
		{Town: "Bristol", Status: "Approved"},
	}

	tests := []struct {
		name   string
		filter zoningStatusFilter
		sort   zoningSortKey
		want   []string // expected town order
	}{
		{"all, sort by town", zoningFilterAll, zoningSortTown,
			[]string{"Andover", "Ansonia", "Avon", "Bethany", "Bristol"}},
		{"approved only", zoningFilterApproved, zoningSortTown,
			[]string{"Ansonia", "Bristol"}},
		{"prohibited only", zoningFilterProhibited, zoningSortTown,
			[]string{"Avon"}},
		{"moratorium only", zoningFilterMoratorium, zoningSortTown,
			[]string{"Bethany"}},
		{"unknown only", zoningFilterUnknown, zoningSortTown,
			[]string{"Andover"}},
		{"sort by status then town", zoningFilterAll, zoningSortStatus,
			[]string{"Ansonia", "Bristol", "Bethany", "Avon", "Andover"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := recomputeZoning(all, tc.filter, tc.sort)
			var gotTowns []string
			for _, r := range got {
				gotTowns = append(gotTowns, r.Town)
			}
			if !reflect.DeepEqual(gotTowns, tc.want) {
				t.Fatalf("got %v, want %v", gotTowns, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test, confirm it fails to compile**

Run: `go test ./internal/ui/ -run TestRecomputeZoning`
Expected: FAIL — `undefined: recomputeZoning`, `undefined: zoningStatusFilter`, etc.

- [ ] **Step 3: Create `internal/ui/zoning.go` with the minimum to compile**

```go
package ui

import (
	"sort"

	"github.com/AgentDank/dank-bubbler/internal/models"
)

// zoningStatusFilter selects which rows the zoning page shows.
type zoningStatusFilter int

const (
	zoningFilterAll zoningStatusFilter = iota
	zoningFilterApproved
	zoningFilterProhibited
	zoningFilterMoratorium
	zoningFilterUnknown
)

// zoningSortKey selects the table's row order.
type zoningSortKey int

const (
	zoningSortTown zoningSortKey = iota
	zoningSortStatus
)

// recomputeZoning filters then sorts rows for the zoning table. The status
// filter matches on the raw string ("" is the Unknown bucket). The sort is
// stable with Town as the tiebreaker when sorting by Status.
func recomputeZoning(all []models.ZoningRow, filter zoningStatusFilter, key zoningSortKey) []models.ZoningRow {
	out := make([]models.ZoningRow, 0, len(all))
	for _, r := range all {
		if !zoningRowMatches(r, filter) {
			continue
		}
		out = append(out, r)
	}

	sort.SliceStable(out, func(i, j int) bool {
		switch key {
		case zoningSortStatus:
			si, sj := zoningStatusRank(out[i].Status), zoningStatusRank(out[j].Status)
			if si != sj {
				return si < sj
			}
			return out[i].Town < out[j].Town
		default:
			return out[i].Town < out[j].Town
		}
	})
	return out
}

// zoningStatusRank returns a sort rank so display order is Approved <
// Moratorium < Prohibited < Unknown (empty string). Plain string `<` would
// put "" first, contradicting the spec.
func zoningStatusRank(status string) int {
	switch status {
	case "Approved":
		return 0
	case "Moratorium":
		return 1
	case "Prohibited":
		return 2
	default:
		return 3
	}
}

func zoningRowMatches(r models.ZoningRow, filter zoningStatusFilter) bool {
	switch filter {
	case zoningFilterApproved:
		return r.Status == "Approved"
	case zoningFilterProhibited:
		return r.Status == "Prohibited"
	case zoningFilterMoratorium:
		return r.Status == "Moratorium"
	case zoningFilterUnknown:
		return r.Status == ""
	default:
		return true
	}
}
```

- [ ] **Step 4: Run the test, confirm it passes**

Run: `go test ./internal/ui/ -run TestRecomputeZoning -v`
Expected: all 6 subtests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/zoning.go internal/ui/zoning_test.go
git commit -m "ui: add recomputeZoning (filter + sort) with tests"
```

---

## Task 7: `ZoningBrowser` — struct, Init/Update/View

Now build the actual page.

**Files:**
- Modify: `internal/ui/zoning.go`

- [ ] **Step 1: Append the browser to `internal/ui/zoning.go`**

Add these imports to the existing `import` block (merge with what's already there):

```go
"fmt"

"charm.land/bubbles/v2/help"
"charm.land/bubbles/v2/key"
"charm.land/bubbles/v2/table"
tea "charm.land/bubbletea/v2"
"charm.land/lipgloss/v2"

"github.com/AgentDank/dank-bubbler/internal/data"
```

Then append to the file:

```go
var (
	zoningCycleFilterKey = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "status"))
	zoningToggleSortKey  = key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "sort"))
)

type zoningHelpKeyMap struct{}

func (zoningHelpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{pagesKey, moveKey, zoningCycleFilterKey, zoningToggleSortKey, quitKey}
}

func (zoningHelpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{pagesKey, moveKey, zoningCycleFilterKey, zoningToggleSortKey, quitKey}}
}

// ZoningBrowser renders the Zoning page: filterable, sortable table of CT towns.
type ZoningBrowser struct {
	loader       *data.Loader
	all          []models.ZoningRow
	view         []models.ZoningRow
	tbl          table.Model
	width, height int
	statusFilter zoningStatusFilter
	sortBy       zoningSortKey
	help         help.Model
	activePage   Page
	loadErr      error
}

func NewZoningBrowser(loader *data.Loader) *ZoningBrowser {
	z := &ZoningBrowser{
		loader: loader,
	}
	z.help = help.New()
	z.help.ShortSeparator = "  "
	z.help.Styles.ShortKey = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Bold(true)
	z.help.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	z.help.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	z.tbl = table.New(
		table.WithColumns([]table.Column{
			{Title: "Town", Width: 30},
			{Title: "Status", Width: 12},
		}),
		table.WithFocused(true),
	)
	z.reload()
	return z
}

func (z *ZoningBrowser) SetActivePage(p Page) { z.activePage = p }

func (z *ZoningBrowser) Init() tea.Cmd { return nil }

func (z *ZoningBrowser) reload() {
	if z.loader == nil {
		return
	}
	rows, err := z.loader.LoadZoning()
	if err != nil {
		z.loadErr = err
		return
	}
	z.all = rows
	z.loadErr = nil
	z.recompute()
}

func (z *ZoningBrowser) recompute() {
	z.view = recomputeZoning(z.all, z.statusFilter, z.sortBy)
	tRows := make([]table.Row, 0, len(z.view))
	for _, r := range z.view {
		status := r.Status
		if status == "" {
			status = "Unknown"
		}
		tRows = append(tRows, table.Row{r.Town, status})
	}
	z.tbl.SetRows(tRows)
}

func (z *ZoningBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		z.width = msg.Width
		z.height = msg.Height
		z.help.SetWidth(msg.Width)
		// Resize table: subtract header (1) + status line (1) + help (1) + table border (2).
		tH := max(msg.Height-5, 3)
		z.tbl.SetHeight(tH)
		// Keep town column fat, status column lean.
		statusW := 12
		townW := max(msg.Width-statusW-4, 10) // -4 for borders/padding
		z.tbl.SetColumns([]table.Column{
			{Title: "Town", Width: townW},
			{Title: "Status", Width: statusW},
		})
	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			z.statusFilter = (z.statusFilter + 1) % 5
			z.recompute()
			return z, nil
		case "o":
			if z.sortBy == zoningSortTown {
				z.sortBy = zoningSortStatus
			} else {
				z.sortBy = zoningSortTown
			}
			z.recompute()
			return z, nil
		case "ctrl+c", "q":
			return z, tea.Quit
		}
	}
	var cmd tea.Cmd
	z.tbl, cmd = z.tbl.Update(msg)
	return z, cmd
}

func (z *ZoningBrowser) View() tea.View {
	header := renderAppHeader(z.width, z.activePage)
	status := z.renderStatusLine()
	footer := z.renderHelp()

	body := z.tbl.View()
	if z.loadErr != nil {
		body = "load error: " + z.loadErr.Error()
	}

	content := lipgloss.JoinVertical(lipgloss.Left, header, status, body, footer)
	return tea.NewView(content)
}

func (z *ZoningBrowser) renderStatusLine() string {
	filter := []string{"All", "Approved", "Prohibited", "Moratorium", "Unknown"}[z.statusFilter]
	sortName := []string{"Town", "Status"}[z.sortBy]
	line := fmt.Sprintf("Status: %s  ·  Sort: %s  ·  %d rows", filter, sortName, len(z.view))
	return lipgloss.NewStyle().
		Width(z.width).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1).
		Render(line)
}

func (z *ZoningBrowser) renderHelp() string {
	if z.width <= 0 {
		return ""
	}
	helpText := z.help.View(zoningHelpKeyMap{})
	return lipgloss.NewStyle().
		Width(z.width).
		MaxWidth(z.width).
		MaxHeight(1).
		Background(lipgloss.Color("238")).
		Foreground(lipgloss.Color("252")).
		Render(helpText)
}
```

- [ ] **Step 2: Verify the UI package builds**

Run: `go build ./internal/ui/...`
Expected: exits 0.

- [ ] **Step 3: Run the existing UI tests plus the zoning recompute test**

Run: `go test ./internal/ui/`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/zoning.go
git commit -m "ui: add ZoningBrowser (table + status/sort keys)"
```

---

## Task 8: Wire the `ZoningBrowser` into `AppModel`

**Files:**
- Modify: `internal/ui/app.go`

- [ ] **Step 1: Add a `zoning` field and constructor call**

In `AppModel`:

```go
type AppModel struct {
	page       Page
	brands     *ProductBrowser
	salesTax   *SalesTaxBrowser
	zoning     *ZoningBrowser   // NEW
	lastResize tea.WindowSizeMsg
}
```

In `NewAppModel`:

```go
func NewAppModel(products []models.Product, brands []models.Brand, loader *data.Loader) *AppModel {
	a := &AppModel{
		page:     PageBrands,
		brands:   NewProductBrowser(products, brands, loader),
		salesTax: NewSalesTaxBrowser(loader),
		zoning:   NewZoningBrowser(loader),          // NEW
	}
	a.syncActivePage()
	return a
}
```

- [ ] **Step 2: Update `syncActivePage`, `Init`, `Update`, `forwardToActive`, `View`**

`syncActivePage`:

```go
func (a *AppModel) syncActivePage() {
	a.brands.SetActivePage(a.page)
	a.salesTax.SetActivePage(a.page)
	a.zoning.SetActivePage(a.page)
}
```

`Init`:

```go
func (a *AppModel) Init() tea.Cmd {
	return tea.Batch(a.brands.Init(), a.salesTax.Init(), a.zoning.Init())
}
```

In `Update`, extend the `WindowSizeMsg` forwarding to hit the new page:

```go
case tea.WindowSizeMsg:
	a.lastResize = msg
	_, cmdA := a.brands.Update(msg)
	_, cmdB := a.salesTax.Update(msg)
	_, cmdC := a.zoning.Update(msg)
	return a, tea.Batch(cmdA, cmdB, cmdC)
```

Extend the `KeyMsg` page-switch block:

```go
case tea.KeyMsg:
	switch msg.String() {
	case "1":
		a.page = PageBrands
		a.syncActivePage()
		return a, nil
	case "2":
		a.page = PageSalesTax
		a.syncActivePage()
		return a, nil
	case "3":
		a.page = PageZoning
		a.syncActivePage()
		return a, nil
	}
```

(Note: `4` will be added in Task 13 when the RetailBrowser is wired; leave it off for now. Pressing `4` does nothing yet — that's OK.)

`forwardToActive`:

```go
func (a *AppModel) forwardToActive(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch a.page {
	case PageBrands:
		_, cmd = a.brands.Update(msg)
	case PageSalesTax:
		_, cmd = a.salesTax.Update(msg)
	case PageZoning:
		_, cmd = a.zoning.Update(msg)
	}
	return a, cmd
}
```

`View`:

```go
func (a *AppModel) View() tea.View {
	switch a.page {
	case PageSalesTax:
		return a.salesTax.View()
	case PageZoning:
		return a.zoning.View()
	default:
		return a.brands.View()
	}
}
```

- [ ] **Step 3: Build and test**

Run: `go build ./...`
Expected: exits 0.

Run: `go test ./...`
Expected: all PASS.

- [ ] **Step 4: Manual smoke test**

Run: `task run` (or `go run ./cmd/brand-demo`)
Expected:
- Tab strip shows `1 Brands · 2 Sales & Tax · 3 Zoning · 4 Retail` (4 is a dead tab for now).
- Pressing `3` shows the Zoning page with a sorted table of ~169 towns.
- Pressing `s` cycles the status filter; row count in the status line changes accordingly.
- Pressing `o` toggles sort between Town and Status; row order changes.
- `q` / `ctrl+c` quits.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/app.go
git commit -m "ui: route PageZoning through AppModel"
```

---

## Task 9: Retail helpers — badge formatter & detail-bar formatter (TDD)

Before building the Retail browser, write the two pure formatters so the browser's View method stays thin.

**Files:**
- Create: `internal/ui/retail.go` (just the helpers)
- Create: `internal/ui/retail_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/ui/retail_test.go`:

```go
package ui

import (
	"strings"
	"testing"

	"github.com/AgentDank/dank-bubbler/internal/models"
)

func TestRetailTypeBadge(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Hybrid Retailer", "HYB"},
		{"Adult-Use Cannabis Only", "AU"},
		{"Medical Marijuana Only", "MED"},
		{"", "?"},
		{"Unknown Whatever", "?"},
	}
	for _, tc := range tests {
		got := retailTypeBadge(tc.in)
		if got != tc.want {
			t.Errorf("retailTypeBadge(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatRetailDetailBar(t *testing.T) {
	loc := models.RetailLocation{
		Type:      "Hybrid Retailer",
		Business:  "ACME CANNABIS LLC",
		DBA:       "ACME DISPENSARY",
		License:   "ABC12345",
		Street:    "1 MAIN ST",
		City:      "HARTFORD",
		Zipcode:   "06103",
		Website:   "https://example.com",
		Longitude: -72.68,
		Latitude:  41.76,
	}

	line1, line2 := formatRetailDetailBar(loc)
	for _, want := range []string{"ACME CANNABIS LLC", "ACME DISPENSARY", "Hybrid Retailer", "ABC12345"} {
		if !strings.Contains(line1, want) {
			t.Errorf("line1 %q missing %q", line1, want)
		}
	}
	for _, want := range []string{"1 MAIN ST", "HARTFORD", "06103", "https://example.com", "41.760", "-72.680"} {
		if !strings.Contains(line2, want) {
			t.Errorf("line2 %q missing %q", line2, want)
		}
	}
}

func TestFormatRetailDetailBarOmitsEmptyFields(t *testing.T) {
	loc := models.RetailLocation{
		Type:     "Hybrid Retailer",
		Business: "ACME",
		License:  "ABC123",
		// DBA, street, city, zipcode, website all empty
	}
	line1, line2 := formatRetailDetailBar(loc)
	if strings.Contains(line1, "·") && !strings.Contains(line1, "ACME") {
		t.Errorf("line1 should contain ACME, got %q", line1)
	}
	// line2 should not be just separators
	if strings.Count(line2, "—") > 1 {
		t.Errorf("line2 should collapse empty fields, got %q", line2)
	}
}
```

- [ ] **Step 2: Run the tests, confirm they fail**

Run: `go test ./internal/ui/ -run 'TestRetail|TestFormatRetail'`
Expected: FAIL — `undefined: retailTypeBadge`, `undefined: formatRetailDetailBar`.

- [ ] **Step 3: Create `internal/ui/retail.go` with just the helpers**

```go
package ui

import (
	"fmt"
	"strings"

	"github.com/AgentDank/dank-bubbler/internal/models"
)

// retailTypeBadge returns a compact 2-3 char badge for a retail location type.
// Returns "?" for empty or unrecognized types.
func retailTypeBadge(t string) string {
	switch t {
	case "Hybrid Retailer":
		return "HYB"
	case "Adult-Use Cannabis Only":
		return "AU"
	case "Medical Marijuana Only":
		return "MED"
	default:
		return "?"
	}
}

// formatRetailDetailBar returns the two lines of the detail bar for a
// selected retail location. Empty fields (and their leading separators) are
// omitted so rows with missing DBA/website/etc. still read cleanly.
//
//	line 1: BUSINESS · DBA  —  Type: <type>  —  Lic#<license>
//	line 2: street, city zipcode  —  website  —  (lat, lng)
func formatRetailDetailBar(loc models.RetailLocation) (string, string) {
	// Line 1
	var who []string
	if loc.Business != "" {
		who = append(who, loc.Business)
	}
	if loc.DBA != "" && loc.DBA != loc.Business {
		who = append(who, loc.DBA)
	}
	var line1Parts []string
	if len(who) > 0 {
		line1Parts = append(line1Parts, strings.Join(who, " · "))
	}
	if loc.Type != "" {
		line1Parts = append(line1Parts, "Type: "+loc.Type)
	}
	if loc.License != "" {
		line1Parts = append(line1Parts, "Lic#"+loc.License)
	}

	// Line 2
	var addr string
	switch {
	case loc.Street != "" && loc.City != "" && loc.Zipcode != "":
		addr = fmt.Sprintf("%s, %s %s", loc.Street, loc.City, loc.Zipcode)
	case loc.Street != "" && loc.City != "":
		addr = fmt.Sprintf("%s, %s", loc.Street, loc.City)
	case loc.City != "":
		addr = loc.City
	default:
		addr = loc.Street
	}
	var line2Parts []string
	if addr != "" {
		line2Parts = append(line2Parts, addr)
	}
	if loc.Website != "" {
		line2Parts = append(line2Parts, loc.Website)
	}
	if loc.Latitude != 0 || loc.Longitude != 0 {
		line2Parts = append(line2Parts, fmt.Sprintf("(%.3f, %.3f)", loc.Latitude, loc.Longitude))
	}

	return strings.Join(line1Parts, "  —  "), strings.Join(line2Parts, "  —  ")
}
```

- [ ] **Step 4: Run the tests, confirm they pass**

Run: `go test ./internal/ui/ -run 'TestRetail|TestFormatRetail' -v`
Expected: 3 PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/retail.go internal/ui/retail_test.go
git commit -m "ui: add retail type badge and detail-bar formatters"
```

---

## Task 10: Retail recompute (TDD)

**Files:**
- Modify: `internal/ui/retail.go`
- Modify: `internal/ui/retail_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/ui/retail_test.go`:

```go
func TestRecomputeRetail(t *testing.T) {
	all := []models.RetailLocation{
		{Business: "ACME", City: "Hartford", Type: "Hybrid Retailer"},
		{Business: "Best", City: "Bristol", Type: "Adult-Use Cannabis Only"},
		{Business: "Carlos", City: "Bristol", Type: "Hybrid Retailer"},
		{Business: "Delta", City: "Ansonia", Type: "Medical Marijuana Only"},
		{Business: "Echo", City: "Ansonia", Type: "Hybrid Retailer"},
	}

	tests := []struct {
		name   string
		filter retailTypeFilter
		sort   retailSortKey
		want   []string // expected Business order
	}{
		{"all, sort by business", retailFilterAll, retailSortBusiness,
			[]string{"ACME", "Best", "Carlos", "Delta", "Echo"}},
		{"hybrid only", retailFilterHybrid, retailSortBusiness,
			[]string{"ACME", "Carlos", "Echo"}},
		{"adult-use only", retailFilterAdultUseOnly, retailSortBusiness,
			[]string{"Best"}},
		{"medical only", retailFilterMedicalOnly, retailSortBusiness,
			[]string{"Delta"}},
		{"sort by city", retailFilterAll, retailSortCity,
			[]string{"Delta", "Echo", "Best", "Carlos", "ACME"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := recomputeRetail(all, tc.filter, tc.sort)
			var gotBiz []string
			for _, r := range got {
				gotBiz = append(gotBiz, r.Business)
			}
			if !reflect.DeepEqual(gotBiz, tc.want) {
				t.Fatalf("got %v, want %v", gotBiz, tc.want)
			}
		})
	}
}
```

Also add `"reflect"` to the retail_test.go imports.

- [ ] **Step 2: Run test, confirm fail**

Run: `go test ./internal/ui/ -run TestRecomputeRetail`
Expected: FAIL — `undefined: recomputeRetail`, `undefined: retailTypeFilter`, etc.

- [ ] **Step 3: Add `recomputeRetail` and its enums to `internal/ui/retail.go`**

Add to the top of `internal/ui/retail.go` (after the existing imports — you may need to add `"sort"` to imports):

```go
// retailTypeFilter selects which retail locations the page shows.
type retailTypeFilter int

const (
	retailFilterAll retailTypeFilter = iota
	retailFilterHybrid
	retailFilterAdultUseOnly
	retailFilterMedicalOnly
)

// retailSortKey selects the list's row order.
type retailSortKey int

const (
	retailSortBusiness retailSortKey = iota
	retailSortCity
)

// recomputeRetail filters then sorts rows for the retail list.
func recomputeRetail(all []models.RetailLocation, filter retailTypeFilter, key retailSortKey) []models.RetailLocation {
	out := make([]models.RetailLocation, 0, len(all))
	for _, r := range all {
		if !retailRowMatches(r, filter) {
			continue
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		switch key {
		case retailSortCity:
			if out[i].City != out[j].City {
				return out[i].City < out[j].City
			}
			return out[i].Business < out[j].Business
		default:
			return out[i].Business < out[j].Business
		}
	})
	return out
}

func retailRowMatches(r models.RetailLocation, filter retailTypeFilter) bool {
	switch filter {
	case retailFilterHybrid:
		return r.Type == "Hybrid Retailer"
	case retailFilterAdultUseOnly:
		return r.Type == "Adult-Use Cannabis Only"
	case retailFilterMedicalOnly:
		return r.Type == "Medical Marijuana Only"
	default:
		return true
	}
}
```

- [ ] **Step 4: Run test, confirm pass**

Run: `go test ./internal/ui/ -run TestRecomputeRetail -v`
Expected: 5 PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/retail.go internal/ui/retail_test.go
git commit -m "ui: add recomputeRetail (filter + sort) with tests"
```

---

## Task 11: `RetailBrowser` — struct, list, no map yet

Build the browser structure and wire it up to use the table + recompute + detail bar, but stub the map area. The map comes in Task 12.

**Files:**
- Modify: `internal/ui/retail.go`

- [ ] **Step 1: Append the browser to `internal/ui/retail.go`**

Add these imports (merge with existing):

```go
"charm.land/bubbles/v2/help"
"charm.land/bubbles/v2/key"
"charm.land/bubbles/v2/table"
tea "charm.land/bubbletea/v2"
"charm.land/lipgloss/v2"

"github.com/AgentDank/dank-bubbler/internal/data"
"github.com/AgentDank/dank-bubbler/mapview"
```

Then append:

```go
type retailFocus int

const (
	focusList retailFocus = iota
	focusMap
)

var (
	retailCycleFilterKey = key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "type"))
	retailToggleSortKey  = key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "sort"))
	retailFocusKey       = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "focus"))
)

type retailHelpKeyMap struct{}

func (retailHelpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{pagesKey, moveKey, retailCycleFilterKey, retailToggleSortKey, retailFocusKey, quitKey}
}

func (retailHelpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{pagesKey, moveKey, retailCycleFilterKey, retailToggleSortKey, retailFocusKey, quitKey}}
}

type RetailBrowser struct {
	loader       *data.Loader
	all          []models.RetailLocation
	view         []models.RetailLocation
	tbl          table.Model
	mv           mapview.Model
	focus        retailFocus
	typeFilter   retailTypeFilter
	sortBy       retailSortKey
	width, height int
	help         help.Model
	activePage   Page
	loadErr      error
	lastSelected int
}

func NewRetailBrowser(loader *data.Loader) *RetailBrowser {
	r := &RetailBrowser{
		loader: loader,
		focus:  focusList,
		lastSelected: -1,
	}
	r.help = help.New()
	r.help.ShortSeparator = "  "
	r.help.Styles.ShortKey = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Bold(true)
	r.help.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	r.help.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	r.tbl = table.New(
		table.WithColumns([]table.Column{
			{Title: "Business", Width: 20},
			{Title: "City", Width: 12},
			{Title: "Type", Width: 5},
		}),
		table.WithFocused(true),
	)
	r.mv = mapview.New(40, 12) // replaced on first resize
	// Center on CT until first selection lands.
	r.mv.SetLatLng(41.6, -72.7, 8)
	r.reload()
	return r
}

func (r *RetailBrowser) SetActivePage(p Page) { r.activePage = p }

func (r *RetailBrowser) Init() tea.Cmd { return nil }

func (r *RetailBrowser) reload() {
	if r.loader == nil {
		return
	}
	rows, err := r.loader.LoadRetailLocations()
	if err != nil {
		r.loadErr = err
		return
	}
	r.all = rows
	r.loadErr = nil
	r.recompute()
}

func (r *RetailBrowser) recompute() {
	r.view = recomputeRetail(r.all, r.typeFilter, r.sortBy)
	tRows := make([]table.Row, 0, len(r.view))
	for _, loc := range r.view {
		tRows = append(tRows, table.Row{loc.Business, loc.City, retailTypeBadge(loc.Type)})
	}
	r.tbl.SetRows(tRows)
	r.lastSelected = -1 // force re-center on next Update
}

func (r *RetailBrowser) selectedLocation() (models.RetailLocation, bool) {
	if len(r.view) == 0 {
		return models.RetailLocation{}, false
	}
	idx := r.tbl.Cursor()
	if idx < 0 || idx >= len(r.view) {
		return models.RetailLocation{}, false
	}
	return r.view[idx], true
}

func (r *RetailBrowser) View() tea.View {
	header := renderAppHeader(r.width, r.activePage)
	footer := r.renderHelp()

	if r.width < 80 || r.height < 20 {
		small := lipgloss.NewStyle().Width(r.width).Height(max(r.height-2, 1)).
			Align(lipgloss.Center, lipgloss.Center).
			Render("window too small")
		return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, small, footer))
	}

	// Body layout: list (left) | map (right), then detail (2 rows), then help.
	detailH := 2
	bodyH := max(r.height-1-detailH-1, 4) // header + detail + footer
	listW := max(r.width*2/5, 30)
	mapW := r.width - listW

	listStyled := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("6")).
		Width(listW - 2).
		Height(bodyH - 2).
		Render(r.tbl.View())

	mapStyled := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("6")).
		Width(mapW - 2).
		Height(bodyH - 2).
		Render(r.mv.View().Content)

	body := lipgloss.JoinHorizontal(lipgloss.Top, listStyled, mapStyled)

	detail := r.renderDetailBar(r.width, detailH)

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, body, detail, footer))
}

func (r *RetailBrowser) renderDetailBar(width, _ int) string {
	if r.loadErr != nil {
		return lipgloss.NewStyle().Width(width).Render("load error: " + r.loadErr.Error())
	}
	loc, ok := r.selectedLocation()
	if !ok {
		return lipgloss.NewStyle().
			Width(width).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("3")).
			Render("no matching retailers")
	}
	l1, l2 := formatRetailDetailBar(loc)
	box := lipgloss.NewStyle().
		Width(width - 2).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("3")).
		Padding(0, 1)
	return box.Render(l1 + "\n" + l2)
}

func (r *RetailBrowser) renderHelp() string {
	if r.width <= 0 {
		return ""
	}
	helpText := r.help.View(retailHelpKeyMap{})
	return lipgloss.NewStyle().
		Width(r.width).
		MaxWidth(r.width).
		MaxHeight(1).
		Background(lipgloss.Color("238")).
		Foreground(lipgloss.Color("252")).
		Render(helpText)
}
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: exits 0.

- [ ] **Step 3: Run tests**

Run: `go test ./...`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/retail.go
git commit -m "ui: add RetailBrowser (list + map stub, no Update yet)"
```

---

## Task 12: `RetailBrowser.Update` — key handling, focus toggle, map re-centering

**Files:**
- Modify: `internal/ui/retail.go`

- [ ] **Step 1: Add `Update` to `RetailBrowser`**

Append to `internal/ui/retail.go`:

```go
func (r *RetailBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.width = msg.Width
		r.height = msg.Height
		r.help.SetWidth(msg.Width)

		detailH := 2
		bodyH := max(r.height-1-detailH-1, 4)
		listW := max(r.width*2/5, 30)
		mapW := r.width - listW

		r.tbl.SetHeight(max(bodyH-4, 3))
		businessW := max(listW-5-12-2, 10) // total minus borders, city col, type col, padding
		r.tbl.SetColumns([]table.Column{
			{Title: "Business", Width: businessW},
			{Title: "City", Width: 12},
			{Title: "Type", Width: 5},
		})

		r.mv.Width = max(mapW-2, 20)
		r.mv.Height = max(bodyH-2, 4)
		// Trigger a re-render at the new size by nudging via the current center.
		cmd := r.centerMapOnSelectionIfChanged(true)
		return r, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if r.focus == focusList {
				r.focus = focusMap
			} else {
				r.focus = focusList
			}
			return r, nil
		case "t":
			r.typeFilter = (r.typeFilter + 1) % 4
			r.recompute()
			return r, r.centerMapOnSelectionIfChanged(true)
		case "o":
			if r.sortBy == retailSortBusiness {
				r.sortBy = retailSortCity
			} else {
				r.sortBy = retailSortBusiness
			}
			r.recompute()
			return r, r.centerMapOnSelectionIfChanged(true)
		case "ctrl+c", "q":
			return r, tea.Quit
		}
	}

	// Route remaining messages based on focus.
	switch r.focus {
	case focusMap:
		var cmd tea.Cmd
		r.mv, cmd = r.mv.Update(msg)
		return r, cmd
	default: // focusList
		var cmd tea.Cmd
		r.tbl, cmd = r.tbl.Update(msg)
		recenterCmd := r.centerMapOnSelectionIfChanged(false)
		return r, tea.Batch(cmd, recenterCmd)
	}
}

// centerMapOnSelectionIfChanged re-centers the map on the currently-selected
// row when the selection index has changed since the last call (or when
// forced). Returns the tea.Cmd mapview emits for its tile fetch.
func (r *RetailBrowser) centerMapOnSelectionIfChanged(force bool) tea.Cmd {
	loc, ok := r.selectedLocation()
	if !ok {
		return nil
	}
	idx := r.tbl.Cursor()
	if !force && idx == r.lastSelected {
		return nil
	}
	r.lastSelected = idx
	if loc.Latitude == 0 && loc.Longitude == 0 {
		return nil
	}
	r.mv.SetLatLng(loc.Latitude, loc.Longitude, 12)
	// mapview renders lazily via its own Update path. Force a render by
	// sending a MapCoordinates message through its Update loop.
	var cmd tea.Cmd
	r.mv, cmd = r.mv.Update(mapview.MapCoordinates{Lat: loc.Latitude, Lng: loc.Longitude})
	return cmd
}
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: exits 0.

- [ ] **Step 3: Run tests**

Run: `go test ./...`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/retail.go
git commit -m "ui: add RetailBrowser.Update with focus toggle and map recenter"
```

---

## Task 13: Wire `RetailBrowser` into `AppModel` and enable page 4

**Files:**
- Modify: `internal/ui/app.go`

- [ ] **Step 1: Add the `retail` field + construct it**

`AppModel`:

```go
type AppModel struct {
	page       Page
	brands     *ProductBrowser
	salesTax   *SalesTaxBrowser
	zoning     *ZoningBrowser
	retail     *RetailBrowser    // NEW
	lastResize tea.WindowSizeMsg
}
```

`NewAppModel`:

```go
retail: NewRetailBrowser(loader),
```

`syncActivePage`:

```go
a.retail.SetActivePage(a.page)
```

`Init`:

```go
return tea.Batch(a.brands.Init(), a.salesTax.Init(), a.zoning.Init(), a.retail.Init())
```

`Update` → `WindowSizeMsg`:

```go
_, cmdD := a.retail.Update(msg)
return a, tea.Batch(cmdA, cmdB, cmdC, cmdD)
```

`Update` → `KeyMsg`: add case for `"4"`:

```go
case "4":
	a.page = PageRetail
	a.syncActivePage()
	return a, nil
```

`forwardToActive` — add `PageRetail` case:

```go
case PageRetail:
	_, cmd = a.retail.Update(msg)
```

`View` — add `PageRetail` case:

```go
case PageRetail:
	return a.retail.View()
```

- [ ] **Step 2: Build & test**

Run: `go build ./... && go test ./...`
Expected: exits 0; all PASS.

- [ ] **Step 3: Manual smoke test**

Run: `task run`
Expected:
- Press `4` → Retail page loads. Left pane shows a table of ~74 retailers; right pane shows a map tile (may take 1-2 seconds for the first tile fetch).
- Up/down moves list selection; map re-centers on the selected retailer's coordinates.
- Detail bar above the footer shows two lines for the selected retailer.
- Press `t` → type filter cycles; row count + map recenter. Press `o` → sort toggles between business / city.
- Press `tab` → focus moves to the map. Now `h/j/k/l` / arrows pan the map, `+`/`-` zoom. Press `tab` again → focus returns to list.
- `t`/`o` work regardless of focus (they're intercepted before forwarding).
- `1`/`2`/`3`/`4` switch pages; `q` quits.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/app.go
git commit -m "ui: route PageRetail through AppModel"
```

---

## Task 14: Layout size assertions for new pages

Extend `layout_test.go` with size checks for the zoning and retail pages so future resize regressions surface.

**Files:**
- Modify: `internal/ui/layout_test.go`

- [ ] **Step 1: Append a new test for the two new pages**

Add to the end of `internal/ui/layout_test.go`:

```go
func TestZoningLayoutFitsWindow(t *testing.T) {
	z := NewZoningBrowser(nil)
	sizes := []struct{ w, h int }{{80, 24}, {100, 40}, {120, 50}}
	for _, sz := range sizes {
		z.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
		v := z.View()
		if w := lipgloss.Width(v.Content); w > sz.w {
			t.Errorf("zoning: size %dx%d overflow width %d", sz.w, sz.h, w)
		}
		if h := lipgloss.Height(v.Content); h > sz.h {
			t.Errorf("zoning: size %dx%d overflow height %d", sz.w, sz.h, h)
		}
	}
}

func TestRetailLayoutFitsWindow(t *testing.T) {
	r := NewRetailBrowser(nil)
	sizes := []struct{ w, h int }{{80, 24}, {100, 40}, {120, 50}}
	for _, sz := range sizes {
		r.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
		v := r.View()
		if w := lipgloss.Width(v.Content); w > sz.w {
			t.Errorf("retail: size %dx%d overflow width %d", sz.w, sz.h, w)
		}
		if h := lipgloss.Height(v.Content); h > sz.h {
			t.Errorf("retail: size %dx%d overflow height %d", sz.w, sz.h, h)
		}
	}
}
```

- [ ] **Step 2: Run the new tests**

Run: `go test ./internal/ui/ -run 'TestZoningLayout|TestRetailLayout' -v`
Expected: PASS. If either fails with overflow, adjust the corresponding browser's size math until it fits.

- [ ] **Step 3: Run the full suite**

Run: `go test ./...`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/layout_test.go
git commit -m "test: assert zoning and retail layouts fit window"
```

---

## Task 15: Final verification

- [ ] **Step 1: Full build**

Run: `go build ./...`
Expected: exits 0.

- [ ] **Step 2: Full tests**

Run: `go test ./...`
Expected: all PASS.

- [ ] **Step 3: Manual exercise of all four pages**

Run: `task run`

Checklist:
- [ ] `1` Brands — still works (browser + filters).
- [ ] `2` Sales & Tax — still works.
- [ ] `3` Zoning — table shows 169 rows. `s` cycles status filter (count visible in status line). `o` toggles sort. Status `Unknown` bucket appears when cycled to.
- [ ] `4` Retail — list + map side by side. Row change re-centers map. `tab` moves focus to map; `hjkl/arrows` pan, `+/-` zoom. `tab` again returns focus to list. `t` cycles type; `o` toggles sort — both work regardless of focus. Detail bar shows selected retailer on two lines with the fields from the spec.
- [ ] Resize the terminal: all four pages reflow without overflow. Small window (<80×20) on the retail page shows "window too small".
- [ ] `q` / `ctrl+c` quits from any page.

- [ ] **Step 4: Done — no commit needed (verification only).**
