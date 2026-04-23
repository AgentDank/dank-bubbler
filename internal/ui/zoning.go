package ui

import (
	"fmt"
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
)

// zoningUpstreamURL points to the CT.gov dataset that feeds ct_zoning.
const zoningUpstreamURL = "https://data.ct.gov/Government/Cannabis-Zoning/khc7-gd9u"

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
// direction. Empty status values are bucketed as "Unknown" (index 3). When
// query is non-empty, rows whose town does not contain the (case-insensitive)
// substring are dropped before partitioning.
func zoningColumnRows(all []models.ZoningRow, order zoningSortOrder, query string) [zoningColumnCount][]models.ZoningRow {
	q := strings.ToLower(strings.TrimSpace(query))
	var cols [zoningColumnCount][]models.ZoningRow
	for _, r := range all {
		if q != "" && !strings.Contains(strings.ToLower(r.Town), q) {
			continue
		}
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
	zoningFilterKey   = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter"))
)

type zoningHelpKeyMap struct{}

func (zoningHelpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{pagesKey, moveKey, zoningFocusKey, zoningSortKeyBind, zoningFilterKey, quitKey}
}

func (zoningHelpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{pagesKey, moveKey, zoningFocusKey, zoningSortKeyBind, zoningFilterKey, quitKey}}
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

	query     string          // committed filter substring
	input     textinput.Model // prompt state when editing the query
	inputOpen bool
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

	z.input = textinput.New()
	z.input.Prompt = "/"
	z.input.Placeholder = "town name"
	z.input.CharLimit = 128
	z.input.SetVirtualCursor(true)

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
	z.cols = zoningColumnRows(z.all, z.sortOrder, z.query)
	for i := range z.cols {
		tRows := make([]table.Row, 0, len(z.cols[i]))
		for _, r := range z.cols[i] {
			tRows = append(tRows, table.Row{r.Town})
		}
		z.tbls[i].SetRows(tRows)
	}
}

func (z *ZoningBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// When the filter prompt is open it owns the keyboard: ESC cancels (and
	// clears any committed query), Enter commits, anything else edits.
	if z.inputOpen {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "esc":
				// ESC clears the committed query and closes the prompt.
				hadQuery := z.query != ""
				z.inputOpen = false
				z.input.Blur()
				if hadQuery {
					z.query = ""
					z.recompute()
				}
				z.relayout()
				return z, nil
			case "enter":
				// Enter seals the live-search query — query is already
				// applied, we just close the prompt.
				z.inputOpen = false
				z.input.Blur()
				z.relayout()
				return z, nil
			}
			// Live search: forward to textinput, then re-apply the filter
			// immediately on any value change.
			prev := z.input.Value()
			var cmd tea.Cmd
			z.input, cmd = z.input.Update(msg)
			if z.input.Value() != prev {
				z.query = z.input.Value()
				z.recompute()
			}
			return z, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		z.width = msg.Width
		z.height = msg.Height
		z.help.SetWidth(msg.Width)

		// Layout: header (1) + summary (1) + columns body + filter? (1) + pageFooter (1) + help (1).
		filterH := 0
		if z.inputOpen || z.query != "" {
			filterH = 1
		}
		bodyH := max(msg.Height-4-filterH, 4)
		// Each column's rendered block = table.View() (tH lines, header included
		// in the viewport) + 2 for the top/bottom border. We want that block to
		// be exactly bodyH tall, so tH = bodyH - 2.
		tH := max(bodyH-2, 3)
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
		case "/":
			z.input.SetValue(z.query)
			z.input.CursorEnd()
			z.inputOpen = true
			z.relayout()
			return z, z.input.Focus()
		case "ctrl+c", "q":
			return z, tea.Quit
		}
	}

	// Forward nav keys to the focused column's table only.
	var cmd tea.Cmd
	z.tbls[z.focus], cmd = z.tbls[z.focus].Update(msg)
	return z, cmd
}

// relayout re-runs the WindowSize resize math with the current dimensions —
// used after filter-row visibility toggles so the table heights re-sync.
func (z *ZoningBrowser) relayout() {
	if z.width == 0 || z.height == 0 {
		return
	}
	z.Update(tea.WindowSizeMsg{Width: z.width, Height: z.height})
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
	pieces := []string{header, summary, body}
	if z.inputOpen || z.query != "" {
		pieces = append(pieces, z.renderFilterRow(z.width))
	}
	pieces = append(pieces, z.renderPageFooterBar(), footer)
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, pieces...))
}

func (z *ZoningBrowser) renderPageFooterBar() string {
	total := 0
	for _, c := range z.cols {
		total += len(c)
	}
	direction := "asc"
	if z.sortOrder == zoningSortDesc {
		direction = "desc"
	}
	parts := []string{fmt.Sprintf("towns: %d", total), "sort: " + direction}
	if z.query != "" {
		parts = append(parts, "filter: "+z.query)
	}
	return renderPageFooter(z.width, strings.Join(parts, "  ·  "), zoningUpstreamURL)
}

func (z *ZoningBrowser) renderFilterRow(width int) string {
	style := lipgloss.NewStyle().Width(width).MaxWidth(width).MaxHeight(1)
	if z.inputOpen {
		z.input.SetWidth(max(width-2, 8))
		return style.Render(z.input.View())
	}
	return style.
		Foreground(lipgloss.Color("245")).
		Render("filter: " + z.query + "  (/: edit, esc-from-/: clear)")
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
