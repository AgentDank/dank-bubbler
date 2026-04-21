# Zoning & Retail Pages Design

Date: 2026-04-21
Status: Approved (pending written-spec review)

## Goal

Add two new pages to dank-bubbler that surface the newly-available CT datasets:

1. **Zoning** — a browsable, sortable, filterable table of `ct_zoning` (169 CT towns with cannabis zoning status).
2. **Retail** — a list of `ct_retail_locations` (74 licensed retailers) paired with an interactive OpenStreetMap view via the local `mapview` package (a Bubble Tea v2 port of `mrusme/mercator`).

Out of scope for this design: multi-marker overlays on the map, population of `ct_applications` / `ct_lottery` into UI, any refactor of the existing Brands and Sales & Tax pages beyond the routing additions described below.

## App-level changes

### Page routing

`internal/ui/app.go` grows the `Page` enum and tab strip:

```go
PageBrands   Page = iota // "1" Brands       (existing)
PageSalesTax             // "2" Sales & Tax  (existing)
PageZoning               // "3" Zoning       (new)
PageRetail               // "4" Retail       (new)
```

`AppModel` gains `zoning *ZoningBrowser` and `retail *RetailBrowser` fields, constructed in `NewAppModel`. Both implement `Init/Update/View` and `SetActivePage(Page)` exactly like the existing pages. The `"1"`/`"2"`/`"3"`/`"4"` page-switch keys remain intercepted at `AppModel.Update` and never reach child pages — this preserves page switching even when a page's focus state would otherwise eat the key (e.g. when map focus is active on the Retail page).

### Data layer

`internal/data/loader.go` gains two methods:

```go
func (l *Loader) LoadZoning() ([]models.ZoningRow, error)
func (l *Loader) LoadRetailLocations() ([]models.RetailLocation, error)
```

- `LoadZoning` issues `SELECT town, COALESCE(status, '') FROM ct_zoning ORDER BY town`. NULL statuses come back as empty strings; the UI renders empty strings as "Unknown".
- `LoadRetailLocations` issues `SELECT type, business, dba, license, street, city, zipcode, website, longitude, latitude FROM ct_retail_locations ORDER BY business`.

### Models

New types in `internal/models/` (new file `zoning.go` and extension of the location shape into `retail.go` if `product.go` is already dense; final file placement decided during plan writing):

```go
type ZoningRow struct {
    Town   string
    Status string // "" represents NULL; UI renders as "Unknown"
}

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

## Zoning page (`internal/ui/zoning.go`)

### Widget

`charm.land/bubbles/v2/table.Model`, two columns:

- `Town` — ~⅔ of the available width.
- `Status` — ~⅓.

The always-"CT" `state` column is dropped as non-informative.

### State

```go
type ZoningBrowser struct {
    loader       *data.Loader
    all          []models.ZoningRow      // loaded once, full set
    view         []models.ZoningRow      // recomputed after filter/sort
    tbl          table.Model
    width, height int
    statusFilter zoningStatusFilter       // All | Approved | Prohibited | Moratorium | Unknown
    sortBy       zoningSortKey            // SortTown | SortStatus
    help         help.Model
    activePage   Page
    loadErr      error
}
```

### Behavior

- On construction: `LoadZoning()` populates `all`; `recompute()` produces `view` and pushes rows into `tbl`.
- `recompute()` = filter(`all`, `statusFilter`) → sort(by `sortBy`, with town as stable secondary) → update `tbl` rows.
- Null/empty status renders as `"Unknown"`. The "Unknown" filter bucket matches exactly the empty-string rows.
- Status sort order is alphabetical: Approved, Moratorium, Prohibited, Unknown.

### Keys

| Key | Action |
|---|---|
| `↑/↓` / `k/j` | Table row navigation (handled by `table.Model`) |
| `s` | Cycle `statusFilter`: All → Approved → Prohibited → Moratorium → Unknown → All |
| `o` | Toggle `sortBy`: Town ↔ Status |
| `1..4` | Page switch (intercepted by `AppModel`) |
| `q`, `ctrl+c` | Quit |

### View

- Header tab strip (shared `renderAppHeader`).
- A single status line above the table: `Status: All  ·  Sort: Town  ·  169 rows` — surfaces current filter/sort state so `s` and `o` have visible effect.
- Table body sized to `height - headerRows - statusLineRow - helpRow`.
- Short help footer: `1-4 · s status · o sort · q quit`.

## Retail + Map page (`internal/ui/retail.go`)

### Layout

Four vertical bands, top to bottom:

1. **Header** tab strip (1 row) — shared `renderAppHeader`.
2. **Main body** — `lipgloss.JoinHorizontal` of list (left, ~40% of width, min 30 cols) and map (right, remainder).
3. **Detail bar** (2 rows, bordered, distinct border color from list/map sections).
4. **Footer help** (1 row, shared style).

If `width < 80` or `height < 20`, the body renders a single "window too small" placeholder (same pattern as `salestax.go`).

### State

```go
type RetailBrowser struct {
    loader       *data.Loader
    all          []models.RetailLocation
    view         []models.RetailLocation
    tbl          table.Model
    mv           mapview.Model
    focus        retailFocus              // FocusList | FocusMap
    typeFilter   retailTypeFilter         // All | Hybrid | AdultUseOnly | MedicalOnly
    sortBy       retailSortKey            // SortBusiness | SortCity
    width, height int
    help         help.Model
    activePage   Page
    loadErr      error
    lastSelected int                      // guards re-centering the map
}
```

### List

`table.Model` with three columns:

- `Business`
- `City`
- `Type` — rendered as a short badge: `HYB` (Hybrid Retailer), `AU` (Adult-Use Cannabis Only), `MED` (Medical Marijuana Only).

### Map

`mapview.Model` (local package). On selection change, the page calls `mv.SetLatLng(loc.Latitude, loc.Longitude, zoom)` where `zoom` stays at the last user-set value (default `12`, state-scale view of CT).

Initial center: the first row's coordinates, or a hard-coded CT centroid (~41.6, -72.7) if the first row has no valid lat/lng.

No multi-marker overlay. The "selected retailer" is indicated by being the map's center.

### Detail bar

Renders from `view[selectedIdx]`, two lines:

```
BUSINESS · DBA  —  Type: Hybrid Retailer  —  Lic#XXXXXX
street, city zipcode  —  website  —  (lat, lng)
```

Each field and its leading separator is omitted when empty. On an empty filter result, the bar shows `no matching retailers` and the map keeps its last center and zoom (no automatic re-centering).

### Keys

| Context | Key | Action |
|---|---|---|
| Any | `tab` / `shift+tab` | Toggle `focus` between list and map |
| Any | `1..4` | Page switch (intercepted by `AppModel`; never reaches page) |
| Any | `q`, `ctrl+c` | Quit |
| `FocusList` | `↑/↓` / `k/j` | Table row nav; on change, page re-centers map via `mv.SetLatLng` |
| `FocusList` | `t` | Cycle `typeFilter`: All → Hybrid → Adult-Use Only → Medical Only → All. Re-center map on new top row. |
| `FocusList` | `o` | Toggle `sortBy`: Business ↔ City |
| `FocusMap` | `h/j/k/l` / arrows | Forwarded to `mv.Update` — map pan |
| `FocusMap` | `+` / `=` / `-` / `_` | Forwarded to `mv.Update` — map zoom |
| `FocusMap` | `t` / `o` | Intercepted by page (not forwarded), so filter/sort still work without toggling focus back |

### Async map render

`mapview` fetches OSM tiles over HTTP inside a `tea.Cmd`. The page returns that `cmd` up to Bubble Tea's runtime; during fetch, `mv.View()` is the last-rendered string (empty on first load). Map pane shows blank briefly on startup — acceptable, matches upstream mercator behavior.

## `mapview` package v2 port

`mapview/mapview.go` is mercator's source with only import paths swapped. The design requires these additional edits in the same file (targeted; not a rewrite):

- `ioutil.ReadAll` → `io.ReadAll`; drop `io/ioutil` import.
- Verify `key.NewBinding` / `key.Matches` against `charm.land/bubbles/v2/key` (expected to be unchanged).
- Leave `View() string` as-is — the Retail page wrapper wraps with `tea.NewView(...)` when composing the page's `tea.View`. This keeps `mapview` broadly reusable outside this app.

New Go module dependencies resolved via `go get`:

- `github.com/flopp/go-staticmaps`
- `github.com/eliukblau/pixterm`
- `github.com/golang/geo`

### Deliberately deferred

A real marker API on `mapview.Model`. When needed, a targeted extension exposes `SetMarkers([]Marker)` backed by `sm.Context.AddMarker`. Out of scope for this design.

## Testing

Matching the existing pattern (`internal/ui/filter_test.go`, `internal/ui/layout_test.go`), pure-logic unit tests only. No Bubble Tea integration, no real DuckDB in tests, no network.

- **`internal/ui/zoning_test.go`** — table-driven tests on a `recompute` helper lifted out of the browser method. Cases: each status-filter bucket returns the correct rows; sort-by-status is stable; "Unknown" bucket captures empty-string status rows.
- **`internal/ui/retail_test.go`** — same shape. Type-filter bucketing, sort-toggle correctness, badge-formatting helper (`"Hybrid Retailer"` → `"HYB"`, etc.), detail-bar formatter with field omission.
- **`internal/ui/layout_test.go`** — extend with size-calculation assertions for both new pages: list+map+detail+help fit within `height`; list/map horizontal split honors the 40/60 ratio and the 30-col min for the list.
- **Loader tests** — `internal/data/loader_test.go` does not exist today, and existing loader code has no tests. Adding loader tests is optional and will be decided during plan writing; if added, they should use a tempfile DuckDB seeded via raw SQL and verify NULL→"" mapping for `LoadZoning` and lat/lng round-trip for `LoadRetailLocations`.

### Manual verification checklist

Covered in the implementation plan, not automated:

- `task run` launches, all four tabs render.
- Zoning: `s` and `o` cycle filter/sort visibly; status line updates.
- Retail: `tab` toggles focus; row change re-centers map; `h/j/k/l` + `+/-` only respond when map has focus; `t`/`o` work regardless of focus.
- Map tiles actually load (network dependency).
- Terminal resize recomputes column widths and map dimensions without overflow.

## Open items for the implementation plan

- Final file placement of the `ZoningRow` / `RetailLocation` model types (`product.go` vs new files).
- Whether to add loader tests now or defer.
- Exact color choices for the detail bar border and the Type badges (should harmonize with the existing cyan/magenta palette in `salestax.go`).
