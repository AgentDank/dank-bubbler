package ui

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/AgentDank/dank-bubbler/internal/data"
	"github.com/AgentDank/dank-bubbler/internal/models"
	"github.com/AgentDank/dank-bubbler/mapview"
)

var (
	retailMarkerColor         = color.RGBA{0x33, 0x99, 0xff, 0xff} // blue
	retailSelectedMarkerColor = color.RGBA{0xff, 0x33, 0x00, 0xff} // red-orange
)

const (
	retailMarkerSize         = 10
	retailSelectedMarkerSize = 8 // small red dot that sits on top of the selected row's blue
)

// retailUpstreamURL points to the CT.gov dataset that feeds ct_retail_locations.
const retailUpstreamURL = "https://data.ct.gov/Health-and-Human-Services/Licensed-Cannabis-Retailers-and-Medical-Marijuana-/p4ks-rfxp"

// retailTypeFilter selects which retail locations the page shows.
type retailTypeFilter int

const (
	retailFilterAll retailTypeFilter = iota
	retailFilterHybrid
	retailFilterAdultUseOnly
	retailFilterMedicalOnly
)

// retailSortKey selects the list's row order. The cycle runs
//
//	DBA desc → DBA asc → Business desc → Business asc → City asc → City desc
//
// and then wraps. The zero value is retailSortDBADesc so a freshly-created
// browser starts on that step.
type retailSortKey int

const (
	retailSortDBAAsc retailSortKey = iota
	retailSortDBADesc
	retailSortBusinessAsc
	retailSortBusinessDesc
	retailSortCityAsc
	retailSortCityDesc
	retailSortCount
)

// effectiveDBA returns the row's DBA, falling back to Business when DBA is
// empty so sort and filter both treat them as equivalent (matches what the
// list column renders).
func effectiveDBA(r models.RetailLocation) string {
	if r.DBA != "" {
		return r.DBA
	}
	return r.Business
}

// recomputeRetail filters then sorts rows for the retail list. query is a
// case-insensitive substring filter applied against DBA (or Business when
// empty), Business, and City; an empty query disables the filter.
func recomputeRetail(all []models.RetailLocation, filter retailTypeFilter, key retailSortKey, query string) []models.RetailLocation {
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]models.RetailLocation, 0, len(all))
	for _, r := range all {
		if !retailRowMatches(r, filter) {
			continue
		}
		if q != "" && !retailMatchesQuery(r, q) {
			continue
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		switch key {
		case retailSortDBADesc:
			if effectiveDBA(a) != effectiveDBA(b) {
				return effectiveDBA(a) > effectiveDBA(b)
			}
			return a.Business < b.Business
		case retailSortDBAAsc:
			if effectiveDBA(a) != effectiveDBA(b) {
				return effectiveDBA(a) < effectiveDBA(b)
			}
			return a.Business < b.Business
		case retailSortBusinessDesc:
			return a.Business > b.Business
		case retailSortBusinessAsc:
			return a.Business < b.Business
		case retailSortCityDesc:
			if a.City != b.City {
				return a.City > b.City
			}
			return a.Business < b.Business
		default: // retailSortCityAsc
			if a.City != b.City {
				return a.City < b.City
			}
			return a.Business < b.Business
		}
	})
	return out
}

// retailMatchesQuery reports whether any of (DBA, Business, City) contains
// the lowercase substring q.
func retailMatchesQuery(r models.RetailLocation, q string) bool {
	return strings.Contains(strings.ToLower(effectiveDBA(r)), q) ||
		strings.Contains(strings.ToLower(r.Business), q) ||
		strings.Contains(strings.ToLower(r.City), q)
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

// retailSortLabel returns a human-readable description of the active sort.
func retailSortLabel(k retailSortKey) string {
	switch k {
	case retailSortDBAAsc:
		return "dba↑"
	case retailSortDBADesc:
		return "dba↓"
	case retailSortBusinessAsc:
		return "business↑"
	case retailSortBusinessDesc:
		return "business↓"
	case retailSortCityAsc:
		return "city↑"
	case retailSortCityDesc:
		return "city↓"
	default:
		return "?"
	}
}

// retailTypeFilterLabel returns a human-readable description of the type
// filter. Returns an empty string when no type filter is applied so the
// caller can suppress the pill entirely.
func retailTypeFilterLabel(f retailTypeFilter) string {
	switch f {
	case retailFilterHybrid:
		return "hybrid"
	case retailFilterAdultUseOnly:
		return "adult-use"
	case retailFilterMedicalOnly:
		return "medical"
	default:
		return ""
	}
}

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

type retailFocus int

const (
	focusList retailFocus = iota
	focusMap
)

var (
	retailCycleFilterKey = key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "type"))
	retailToggleSortKey  = key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "sort"))
	retailFocusKey       = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "focus"))
	retailToggleGfxKey   = key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "gfx"))
	retailZoomKey        = key.NewBinding(key.WithKeys("+", "=", "-", "_"), key.WithHelp("+/-", "zoom"))
	retailSatelliteKey   = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sat"))
	retailFilterKey      = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter"))
	retailResetKey       = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "recenter"))
)

type retailHelpKeyMap struct{}

func (retailHelpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{pagesKey, moveKey, retailCycleFilterKey, retailToggleSortKey, retailFilterKey, retailFocusKey, retailToggleGfxKey, retailZoomKey, retailResetKey, retailSatelliteKey, quitKey}
}

func (retailHelpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{pagesKey, moveKey, retailCycleFilterKey, retailToggleSortKey, retailFilterKey, retailFocusKey, retailToggleGfxKey, retailZoomKey, retailResetKey, retailSatelliteKey, quitKey}}
}

type RetailBrowser struct {
	loader        *data.Loader
	all           []models.RetailLocation
	view          []models.RetailLocation
	tbl           table.Model
	mv            mapview.Model
	focus         retailFocus
	typeFilter    retailTypeFilter
	sortBy        retailSortKey
	width, height int
	help          help.Model
	activePage    Page
	loadErr       error
	lastSelected  int

	query     string          // committed filter substring
	input     textinput.Model // prompt state when editing the query
	inputOpen bool            // true while the user is editing the filter
}

func NewRetailBrowser(loader *data.Loader) *RetailBrowser {
	r := &RetailBrowser{
		loader:       loader,
		focus:        focusList,
		lastSelected: -1,
	}
	r.help = help.New()
	r.help.ShortSeparator = "  "
	r.help.Styles.ShortKey = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Bold(true)
	r.help.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	r.help.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	headerStyles := table.DefaultStyles()
	headerStyles.Header = headerStyles.Header.
		Background(lipgloss.Color(tableHeaderBg)).
		Foreground(lipgloss.Color("230"))
	headerStyles.Selected = headerStyles.Selected.
		Background(lipgloss.Color(tableSelectedBg)).
		Foreground(lipgloss.Color("230"))
	r.tbl = table.New(
		table.WithColumns([]table.Column{
			{Title: "DBA", Width: 12},
			{Title: "Business", Width: 12},
			{Title: "City", Width: 16},
			{Title: "Type", Width: 4},
		}),
		table.WithFocused(true),
		table.WithStyles(headerStyles),
	)
	r.mv = mapview.New(40, 12) // replaced on first resize
	// Center on CT until first selection lands.
	r.mv.SetLatLng(41.6, -72.7, 8)

	r.input = textinput.New()
	r.input.Prompt = "/"
	r.input.Placeholder = "business, dba, or city"
	r.input.CharLimit = 128
	r.input.SetVirtualCursor(true)

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
	r.view = recomputeRetail(r.all, r.typeFilter, r.sortBy, r.query)
	tRows := make([]table.Row, 0, len(r.view))
	for _, loc := range r.view {
		dba := loc.DBA
		if dba == "" {
			dba = loc.Business
		}
		tRows = append(tRows, table.Row{dba, loc.Business, loc.City, retailTypeBadge(loc.Type)})
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

	// Body layout: list (left) | map (right), then detail (2 rows), then
	// page-state bar (1), then help.
	// listW formula must match the one in Update so the table inside fits.
	detailH := 2
	pageFooterH := 1
	filterH := 0
	if r.inputOpen || r.query != "" {
		filterH = 1
	}
	bodyH := max(r.height-1-filterH-detailH-pageFooterH-1, 4) // header + filter? + detail + pageFooter + help
	listW := max(r.width/2, 40)
	mapW := r.width - listW

	listBorder := lipgloss.Color("6")
	mapBorder := lipgloss.Color("6")
	switch r.focus {
	case focusList:
		listBorder = lipgloss.Color("3")
	case focusMap:
		mapBorder = lipgloss.Color("3")
	}
	listStyled := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(listBorder).
		Width(listW).
		Height(bodyH - 2).
		Render(r.tbl.View())

	mapStyled := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(mapBorder).
		Width(mapW).
		Height(bodyH - 2).
		Render(r.mv.View().Content)

	body := lipgloss.JoinHorizontal(lipgloss.Top, listStyled, mapStyled)

	detail := r.renderDetailBar(r.width, detailH)

	pieces := []string{header, body}
	if filterH > 0 {
		pieces = append(pieces, r.renderFilterRow(r.width))
	}
	pieces = append(pieces, detail, r.renderPageFooterBar(), footer)
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, pieces...))
}

// renderPageFooterBar composes the page-state + upstream-URL bar. Parts:
//   - rows: how many retailers pass the current filters
//   - type: filter label (only when non-All)
//   - sort: current sort key + direction
//   - filter: current search query (only when non-empty)
//   - map: glyph vs kitty, tile style
func (r *RetailBrowser) renderPageFooterBar() string {
	parts := []string{fmt.Sprintf("rows: %d", len(r.view))}
	if t := retailTypeFilterLabel(r.typeFilter); t != "" {
		parts = append(parts, "type: "+t)
	}
	parts = append(parts, "sort: "+retailSortLabel(r.sortBy))
	if r.query != "" {
		parts = append(parts, "filter: "+r.query)
	}
	mapMode := "glyph"
	if r.mv.RenderMode() == mapview.RenderKitty {
		mapMode = "kitty"
	}
	tileName := "osm"
	if r.mv.TileStyle() == mapview.ArcgisWorldImagery {
		tileName = "sat"
	}
	parts = append(parts, "map: "+mapMode+"/"+tileName)
	return renderPageFooter(r.width, strings.Join(parts, "  ·  "), retailUpstreamURL)
}

// relayout reruns the WindowSize-driven layout math with the current
// width/height. Use this after state changes (e.g. filter row appearing) that
// should resize the table/map. Safe to call before the first real resize,
// since it's a no-op when width is zero.
func (r *RetailBrowser) relayout() {
	if r.width == 0 || r.height == 0 {
		return
	}
	r.Update(tea.WindowSizeMsg{Width: r.width, Height: r.height})
}

func (r *RetailBrowser) renderFilterRow(width int) string {
	style := lipgloss.NewStyle().Width(width).MaxWidth(width).MaxHeight(1)
	if r.inputOpen {
		r.input.SetWidth(max(width-2, 8))
		return style.Render(r.input.View())
	}
	return style.
		Foreground(lipgloss.Color("245")).
		Render("filter: " + r.query + "  (/: edit, esc-from-/: clear)")
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
		Width(width).
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

func (r *RetailBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// When the filter prompt is open, it owns the keyboard: ESC cancels,
	// Enter commits, everything else edits. Non-KeyMsg messages continue
	// to fall through to the normal handlers so map renders still land.
	if r.inputOpen {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "esc":
				// ESC clears any committed query and closes the prompt.
				hadQuery := r.query != ""
				r.inputOpen = false
				r.input.Blur()
				if hadQuery {
					r.query = ""
					r.recompute()
				}
				r.relayout()
				return r, nil
			case "enter":
				// Enter seals the live-search query — the query is already
				// applied (live), we just close the prompt.
				r.inputOpen = false
				r.input.Blur()
				r.relayout()
				return r, nil
			}
			// Live search: forward to textinput, then if the value changed
			// re-apply the filter immediately and recenter on the (possibly
			// new) selected row.
			prev := r.input.Value()
			var cmd tea.Cmd
			r.input, cmd = r.input.Update(msg)
			if r.input.Value() != prev {
				r.query = r.input.Value()
				r.recompute()
				return r, tea.Batch(cmd, r.centerMapOnSelectionIfChanged(true))
			}
			return r, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.width = msg.Width
		r.height = msg.Height
		r.help.SetWidth(msg.Width)

		detailH := 2
		pageFooterH := 1
		filterH := 0
		if r.inputOpen || r.query != "" {
			filterH = 1
		}
		bodyH := max(r.height-1-filterH-detailH-pageFooterH-1, 4)
		listW := max(r.width/2, 40)
		mapW := r.width - listW

		// listStyled has outer width listW with a Border, so its content
		// area is (listW-2). The table render uses that full content width
		// and its 4 cells each consume +2 chars of Padding(0,1), so the
		// per-cell content budget is (listW - 2 - 8).
		const (
			cityW = 16
			typeW = 4 // header "Type" is 4 chars; badges HYB/AU/MED fit in 3
		)
		tblW := max(listW-2, 10)
		// Split remaining space between DBA and Business, clipping long names.
		nameBudget := max(tblW-8-cityW-typeW, 8)
		dbaW := max(nameBudget/2, 4)
		businessW := max(nameBudget-dbaW, 4)

		r.tbl.SetHeight(max(bodyH-4, 3))
		r.tbl.SetWidth(tblW)
		r.tbl.SetColumns([]table.Column{
			{Title: "DBA", Width: dbaW},
			{Title: "Business", Width: businessW},
			{Title: "City", Width: cityW},
			{Title: "Type", Width: typeW},
		})

		r.mv.Width = max(mapW-2, 20)
		r.mv.Height = max(bodyH-4, 4)
		// Trigger a re-render at the new size by nudging via the current center.
		cmd := r.centerMapOnSelectionIfChanged(true)
		return r, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab":
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
			r.sortBy = (r.sortBy + 1) % retailSortCount
			r.recompute()
			return r, r.centerMapOnSelectionIfChanged(true)
		case "g":
			mode := mapview.RenderGlyph
			if r.mv.RenderMode() == mapview.RenderGlyph {
				mode = mapview.RenderKitty
			}
			return r, r.mv.SetRenderMode(mode)
		case "s":
			style := mapview.OpenStreetMaps
			if r.mv.TileStyle() == mapview.OpenStreetMaps {
				style = mapview.ArcgisWorldImagery
			}
			return r, r.mv.SetStyle(style)
		case "/":
			r.input.SetValue(r.query)
			r.input.CursorEnd()
			r.inputOpen = true
			r.relayout()
			return r, r.input.Focus()
		case "r":
			// Snap the map back to the selected retailer. Forces a re-center
			// even if the cursor hasn't moved, so it also works as an "undo
			// my panning" shortcut.
			return r, r.centerMapOnSelectionIfChanged(true)
		case "+", "=", "-", "_":
			// Zoom centered on the selected retailer if there is one, otherwise
			// around the current map center. Clamp to mapview's [2, 16] range.
			delta := 1
			if msg.String() == "-" || msg.String() == "_" {
				delta = -1
			}
			newZoom := r.mv.Zoom() + delta
			if newZoom < 2 {
				newZoom = 2
			}
			if newZoom > 16 {
				newZoom = 16
			}
			lat, lng := r.mv.Center()
			if loc, ok := r.selectedLocation(); ok && (loc.Latitude != 0 || loc.Longitude != 0) {
				lat, lng = loc.Latitude, loc.Longitude
			}
			r.mv.SetLatLng(lat, lng, newZoom)
			var cmd tea.Cmd
			r.mv, cmd = r.mv.Update(mapview.MapCoordinates{Lat: lat, Lng: lng})
			return r, cmd
		case "ctrl+c", "q":
			return r, tea.Quit
		}
	}

	// Mapview output messages must reach r.mv regardless of which pane has
	// focus, otherwise the new render never updates r.mv.maprender and the
	// displayed image goes stale.
	if mapview.IsMapUpdate(msg) {
		var cmd tea.Cmd
		r.mv, cmd = r.mv.Update(msg)
		return r, cmd
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
	r.applyRetailMarkers()

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
	zoom := r.mv.Zoom()
	if zoom <= 0 {
		zoom = 12 // default if mapview hasn't been initialized with a zoom yet
	}
	r.mv.SetLatLng(loc.Latitude, loc.Longitude, zoom)
	// mapview renders lazily via its own Update path. Force a render by
	// sending a MapCoordinates message through its Update loop.
	var cmd tea.Cmd
	r.mv, cmd = r.mv.Update(mapview.MapCoordinates{Lat: loc.Latitude, Lng: loc.Longitude})
	return cmd
}

// applyRetailMarkers syncs the map's marker set with the current filtered
// view. Every retailer gets a small blue dot; the selected row also gets a
// smaller red dot drawn on top of its blue so the selection is obvious.
func (r *RetailBrowser) applyRetailMarkers() {
	selectedIdx := r.tbl.Cursor()
	markers := make([]mapview.Marker, 0, len(r.view)+1)
	var selected *mapview.Marker
	for i, loc := range r.view {
		if loc.Latitude == 0 && loc.Longitude == 0 {
			continue
		}
		markers = append(markers, mapview.Marker{
			Lat:   loc.Latitude,
			Lng:   loc.Longitude,
			Color: retailMarkerColor,
			Size:  retailMarkerSize,
		})
		if i == selectedIdx {
			selected = &mapview.Marker{
				Lat:   loc.Latitude,
				Lng:   loc.Longitude,
				Color: retailSelectedMarkerColor,
				Size:  retailSelectedMarkerSize,
			}
		}
	}
	if selected != nil {
		markers = append(markers, *selected)
	}
	r.mv.SetMarkers(markers)
}
