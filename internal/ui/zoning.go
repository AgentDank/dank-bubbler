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

// zoningSortOrder determines the alphabetical direction applied to each column.
type zoningSortOrder int

const (
	zoningSortAsc zoningSortOrder = iota
	zoningSortDesc
)

// zoningColumnCount is the number of status columns on the page.
const zoningColumnCount = 4

// zoningColumnStatuses is the fixed, ordered list of statuses shown as columns.
var zoningColumnStatuses = [zoningColumnCount]string{
	"Approved",
	"Prohibited",
	"Moratorium",
	"Unknown",
}

// zoningColumnRows partitions `all` into one slice per status column (in
// zoningColumnStatuses order) and sorts each slice by town in `order`
// direction. Empty status values are bucketed as "Unknown" (index 3).
func zoningColumnRows(all []models.ZoningRow, order zoningSortOrder) [zoningColumnCount][]models.ZoningRow {
	var cols [zoningColumnCount][]models.ZoningRow
	for _, r := range all {
		idx := zoningColumnIndex(r.Status)
		cols[idx] = append(cols[idx], r)
	}
	for i := range cols {
		sortRows(cols[i], order)
	}
	return cols
}

// zoningColumnIndex maps a status string to its column index. Empty string
// (representing NULL / literal "null" after normalization) maps to 3 (Unknown).
// Unrecognized non-empty statuses also fall through to Unknown so no data
// silently disappears.
func zoningColumnIndex(status string) int {
	switch status {
	case "Approved":
		return 0
	case "Prohibited":
		return 1
	case "Moratorium":
		return 2
	default: // "" and anything unrecognized
		return 3
	}
}

func sortRows(rows []models.ZoningRow, order zoningSortOrder) {
	sort.SliceStable(rows, func(i, j int) bool {
		if order == zoningSortDesc {
			return rows[i].Town > rows[j].Town
		}
		return rows[i].Town < rows[j].Town
	})
}

var (
	zoningFocusKey    = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "focus"))
	zoningSortKeyBind = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort"))
)

type zoningHelpKeyMap struct{}

func (zoningHelpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{pagesKey, moveKey, zoningFocusKey, zoningSortKeyBind, quitKey}
}

func (zoningHelpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{pagesKey, moveKey, zoningFocusKey, zoningSortKeyBind, quitKey}}
}

// ZoningBrowser shows four side-by-side tables, one per zoning status.
type ZoningBrowser struct {
	loader        *data.Loader
	all           []models.ZoningRow
	cols          [zoningColumnCount][]models.ZoningRow // view: partitioned + sorted
	tbls          [zoningColumnCount]table.Model
	focus         int // 0..zoningColumnCount-1
	sortOrder     zoningSortOrder
	width, height int
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

	headerStyles := table.DefaultStyles()
	headerStyles.Header = headerStyles.Header.
		Background(lipgloss.Color(tableHeaderBg)).
		Foreground(lipgloss.Color("230"))
	headerStyles.Selected = headerStyles.Selected.
		Background(lipgloss.Color(tableSelectedBg)).
		Foreground(lipgloss.Color("230"))
	for i := range z.tbls {
		z.tbls[i] = table.New(
			table.WithColumns([]table.Column{
				{Title: zoningColumnStatuses[i], Width: 16},
			}),
			table.WithFocused(i == 0),
			table.WithStyles(headerStyles),
		)
	}

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
	z.cols = zoningColumnRows(z.all, z.sortOrder)
	for i := range z.cols {
		tRows := make([]table.Row, 0, len(z.cols[i]))
		for _, r := range z.cols[i] {
			tRows = append(tRows, table.Row{r.Town})
		}
		z.tbls[i].SetRows(tRows)
	}
}

func (z *ZoningBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		z.width = msg.Width
		z.height = msg.Height
		z.help.SetWidth(msg.Width)

		// Layout: header (1) + summary (1) + columns body + footer (1).
		bodyH := max(msg.Height-3, 4)
		// Each column: subtract for the border frame (2) and the column header row (1).
		tH := max(bodyH-3, 3)
		// Each column gets an equal slice of the width; subtract 2 per column for border.
		colOuterW := msg.Width / zoningColumnCount
		colInnerW := max(colOuterW-2, 10)
		for i := range z.tbls {
			z.tbls[i].SetHeight(tH)
			z.tbls[i].SetWidth(colInnerW)
			z.tbls[i].SetColumns([]table.Column{
				{Title: zoningColumnStatuses[i], Width: colInnerW - 2},
			})
		}
		return z, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			z.tbls[z.focus].Blur()
			z.focus = (z.focus + 1) % zoningColumnCount
			z.tbls[z.focus].Focus()
			return z, nil
		case "shift+tab":
			z.tbls[z.focus].Blur()
			z.focus = (z.focus - 1 + zoningColumnCount) % zoningColumnCount
			z.tbls[z.focus].Focus()
			return z, nil
		case "s":
			if z.sortOrder == zoningSortAsc {
				z.sortOrder = zoningSortDesc
			} else {
				z.sortOrder = zoningSortAsc
			}
			z.recompute()
			return z, nil
		case "ctrl+c", "q":
			return z, tea.Quit
		}
	}

	// Forward nav keys to the focused column's table only.
	var cmd tea.Cmd
	z.tbls[z.focus], cmd = z.tbls[z.focus].Update(msg)
	return z, cmd
}

func (z *ZoningBrowser) View() tea.View {
	header := renderAppHeader(z.width, z.activePage)
	summary := z.renderSummary()
	footer := z.renderHelp()

	if z.width < 80 || z.height < 20 {
		small := lipgloss.NewStyle().Width(z.width).Height(max(z.height-2, 1)).
			Align(lipgloss.Center, lipgloss.Center).
			Render("window too small")
		return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, small, footer))
	}

	colOuterW := z.width / zoningColumnCount
	rendered := make([]string, zoningColumnCount)
	for i := range z.tbls {
		borderColor := lipgloss.Color("6")
		if i == z.focus {
			borderColor = lipgloss.Color("3")
		}
		rendered[i] = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(borderColor).
			Width(colOuterW - 2).
			Render(z.tbls[i].View())
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, summary, body, footer))
}

func (z *ZoningBrowser) renderSummary() string {
	if z.loadErr != nil {
		return lipgloss.NewStyle().Width(z.width).Render("load error: " + z.loadErr.Error())
	}
	total := 0
	for _, c := range z.cols {
		total += len(c)
	}
	line := fmt.Sprintf(
		"%d towns: %d approved · %d prohibited · %d moratorium · %d unknown",
		total, len(z.cols[0]), len(z.cols[1]), len(z.cols[2]), len(z.cols[3]),
	)
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
