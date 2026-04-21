package ui

import (
	"fmt"
	"sort"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/AgentDank/dank-bubbler/internal/data"
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

// zoningStatusRank returns a sort rank for the status string so that the
// display order is Approved < Moratorium < Prohibited < Unknown ("").
func zoningStatusRank(status string) int {
	switch status {
	case "Approved":
		return 0
	case "Moratorium":
		return 1
	case "Prohibited":
		return 2
	default: // "" (Unknown) and anything else sorts last
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
	loader        *data.Loader
	all           []models.ZoningRow
	view          []models.ZoningRow
	tbl           table.Model
	width, height int
	statusFilter  zoningStatusFilter
	sortBy        zoningSortKey
	help          help.Model
	activePage    Page
	loadErr       error
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
		z.tbl.SetWidth(msg.Width)
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
