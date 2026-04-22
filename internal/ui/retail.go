package ui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/AgentDank/dank-bubbler/internal/data"
	"github.com/AgentDank/dank-bubbler/internal/models"
	"github.com/AgentDank/dank-bubbler/mapview"
)

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
)

type retailHelpKeyMap struct{}

func (retailHelpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{pagesKey, moveKey, retailCycleFilterKey, retailToggleSortKey, retailFocusKey, quitKey}
}

func (retailHelpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{pagesKey, moveKey, retailCycleFilterKey, retailToggleSortKey, retailFocusKey, quitKey}}
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
	headerStyles.Header = headerStyles.Header.Reverse(true)
	r.tbl = table.New(
		table.WithColumns([]table.Column{
			{Title: "Business", Width: 20},
			{Title: "City", Width: 12},
			{Title: "Type", Width: 5},
		}),
		table.WithFocused(true),
		table.WithStyles(headerStyles),
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
		r.tbl.SetWidth(max(listW-4, 10))
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
