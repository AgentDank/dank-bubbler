// Package ui provides BubbleTea UI components for the dank-bubbler application
package ui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/charmbracelet/x/ansi"

	"github.com/AgentDank/dank-bubbler/internal/data"
	"github.com/AgentDank/dank-bubbler/internal/models"
)

// productsUpstreamURL points to the CT.gov dataset that feeds the products
// table (medical + adult-use product registrations).
const productsUpstreamURL = "https://data.ct.gov/Health-and-Human-Services/Medical-Marijuana-and-Adult-Use-Cannabis-Product-R/egd5-wb6r"

// ProductBrowser is a BubbleTea component for browsing cannabis products
type ProductBrowser struct {
	products      []models.Product
	allProducts   []models.Product
	brands        []models.Brand
	selectedIdx   int
	width         int
	height        int
	infoPaneView  viewport.Model
	filterMode    FilterMode
	filterOptions []string
	filterIdx     int
	filterTitle   string
	loader        *data.Loader
	activeFilter  string
	dateSort      ProductDateSort
	help          help.Model
	leftList      list.Model
	tbl           table.Model
	picture       pictureView
	activePage    Page
}

// SetActivePage tells the browser which tab is currently active so the header
// can render the tab strip with the correct highlight.
func (pb *ProductBrowser) SetActivePage(p Page) { pb.activePage = p }

// FilterMode represents the current filter type
type FilterMode int

const (
	// tableHeaderBg is the background color applied to table column headers
	// across the app (ANSI 22 — dark green).
	tableHeaderBg = "22"

	// tableSelectedBg is the background color applied to the selected row of
	// table widgets across the app (ANSI 34 — a brighter green that pairs
	// with the darker tableHeaderBg).
	tableSelectedBg = "34"

	FilterModeNone FilterMode = iota
	FilterModeByBrand
	FilterModeByName
	FilterModeByType
	FilterModeByForm
)

// ProductDateSort describes the optional approval-date ordering for the
// currently visible products.
type ProductDateSort int

const (
	ProductDateSortNone ProductDateSort = iota
	ProductDateSortNewest
	ProductDateSortOldest
)

type FilterOptionItem struct {
	value string
}

type browserHelpKeyMap struct {
	filterMode FilterMode
}

var (
	moveKey = key.NewBinding(
		key.WithKeys("up", "k", "down", "j"),
		key.WithHelp("↑/k ↓/j", "move"),
	)
	brandFilterKey = key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "brand"),
	)
	nameFilterKey = key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "name"),
	)
	typeFilterKey = key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "type"),
	)
	formFilterKey = key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "form"),
	)
	dateSortKey = key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "date sort"),
	)
	clearFilterKey = key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clear"),
	)
	pictureModeKey = key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "image"),
	)
	applyFilterKey = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "apply"),
	)
	cancelFilterKey = key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	)
	quitKey = key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	)
)

func (km browserHelpKeyMap) ShortHelp() []key.Binding {
	if km.filterMode != FilterModeNone {
		return []key.Binding{pagesKey, moveKey, applyFilterKey, cancelFilterKey, quitKey}
	}

	return []key.Binding{
		pagesKey,
		moveKey,
		quitKey,
		brandFilterKey,
		nameFilterKey,
		typeFilterKey,
		formFilterKey,
		dateSortKey,
		clearFilterKey,
		pictureModeKey,
	}
}

func (km browserHelpKeyMap) FullHelp() [][]key.Binding {
	if km.filterMode != FilterModeNone {
		return [][]key.Binding{{pagesKey, moveKey, applyFilterKey, cancelFilterKey, quitKey}}
	}

	return [][]key.Binding{{pagesKey, moveKey, quitKey, brandFilterKey, nameFilterKey, typeFilterKey, formFilterKey, dateSortKey, clearFilterKey, pictureModeKey}}
}

func (f FilterOptionItem) FilterValue() string {
	return strings.ToLower(f.value)
}

func (f FilterOptionItem) Title() string {
	return f.value
}

func (f FilterOptionItem) Description() string {
	return ""
}

// NewProductBrowser creates a new product browser component
func NewProductBrowser(products []models.Product, brands []models.Brand, loader *data.Loader) *ProductBrowser {
	productCopy := append([]models.Product(nil), products...)
	pb := &ProductBrowser{
		products:    productCopy,
		allProducts: append([]models.Product(nil), products...),
		brands:      brands,
		selectedIdx: 0,
		filterMode:  FilterModeNone,
		loader:      loader,
		dateSort:    ProductDateSortNewest,
	}
	pb.help = help.New()
	pb.help.ShortSeparator = "  "
	pb.help.FullSeparator = "  "
	pb.help.Styles.ShortKey = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Bold(true)
	pb.help.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	pb.help.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	pb.help.Styles.Ellipsis = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	pb.leftList = newBrowserList()
	pb.tbl = newProductTable()
	pb.picture = newPictureView()
	pb.applyDateSort()
	pb.setProductItems()
	pb.updateDimensions(80, 24)
	// Prime selected product details
	pb.loadSelectedProductDetails()
	return pb
}

// Init initializes the product browser. Kicks off the initial product image
// fetch so the picture pane lights up as soon as dimensions are known.
func (pb *ProductBrowser) Init() tea.Cmd {
	return pb.syncPictureURL()
}

// syncPictureURL pushes the currently-selected product's image URL into the
// picture view. Safe to call when the product list is empty.
func (pb *ProductBrowser) syncPictureURL() tea.Cmd {
	if len(pb.products) == 0 {
		return pb.picture.SetURL("")
	}
	return pb.picture.SetURL(pb.products[pb.selectedIdx].ProductImageURL)
}

// picturePaneInnerSize returns the inner cell dimensions of the picture pane
// for the current window size, matching what View() will render into. Used
// by Update when the window is resized, so the pictureView can re-render
// its Kitty frame at the right grid.
func (pb *ProductBrowser) picturePaneInnerSize() (int, int) {
	middleH := pb.middleHeight()
	_, bottomH := pb.middleSplit(middleH)
	paneW := pb.width / 4
	lastPaneW := max(pb.width-3*paneW, 0)
	style := pb.infoPaneStyle()
	innerW := max(lastPaneW-style.GetHorizontalFrameSize(), 0)
	innerH := max(bottomH-style.GetVerticalFrameSize(), 0)
	return innerW, innerH
}

// Update handles messages for the product browser
func (pb *ProductBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Async picture-pane results (image fetch completions, Kitty frames)
	// arrive here regardless of filter mode or keyboard focus.
	if isPictureMsg(msg) {
		return pb, pb.picture.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		pb.width = msg.Width
		pb.height = msg.Height
		pb.updateDimensions(msg.Width, msg.Height)
		picW, picH := pb.picturePaneInnerSize()
		return pb, tea.Batch(pb.picture.SetSize(picW, picH), pb.syncPictureURL())

	case tea.MouseWheelMsg:
		m := msg.Mouse()
		if pb.filterMode == FilterModeNone {
			oldIndex := pb.tbl.Cursor()
			switch m.Button {
			case tea.MouseWheelUp:
				pb.tbl.MoveUp(1)
			case tea.MouseWheelDown:
				pb.tbl.MoveDown(1)
			default:
				return pb, nil
			}
			newIndex := pb.tbl.Cursor()
			if newIndex != oldIndex {
				pb.selectedIdx = newIndex
				pb.loadSelectedProductDetails()
				pb.updateInfoPane()
				return pb, pb.syncPictureURL()
			}
			return pb, nil
		}
		oldIndex := pb.leftList.Index()
		switch m.Button {
		case tea.MouseWheelUp:
			pb.leftList.CursorUp()
		case tea.MouseWheelDown:
			pb.leftList.CursorDown()
		default:
			return pb, nil
		}
		newIndex := pb.leftList.Index()
		if newIndex != oldIndex {
			pb.filterIdx = newIndex
		}
		return pb, nil

	case tea.KeyMsg:
		if pb.filterMode != FilterModeNone {
			return pb.updateFilter(msg)
		}

		switch msg.String() {
		case "b": // Filter by brand
			pb.openFilter(FilterModeByBrand)
			return pb, nil

		case "n": // Filter by name
			pb.openFilter(FilterModeByName)
			return pb, nil

		case "t": // Filter by type
			pb.openFilter(FilterModeByType)
			return pb, nil

		case "f": // Filter by form
			pb.openFilter(FilterModeByForm)
			return pb, nil

		case "d": // Toggle approval-date sort
			pb.toggleDateSort()
			return pb, pb.syncPictureURL()

		case "c":
			pb.clearFilter()
			return pb, pb.syncPictureURL()

		case "g": // Toggle picture render mode (glyph ↔ kitty)
			return pb, pb.picture.Toggle()

		case "ctrl+c", "q":
			return pb, tea.Quit
		}

		oldIndex := pb.tbl.Cursor()
		var cmd tea.Cmd
		pb.tbl, cmd = pb.tbl.Update(msg)
		pb.selectedIdx = pb.tbl.Cursor()
		if pb.selectedIdx != oldIndex {
			pb.loadSelectedProductDetails()
			pb.updateInfoPane()
			return pb, tea.Batch(cmd, pb.syncPictureURL())
		}
		return pb, cmd
	}

	return pb, nil
}

// View renders the product browser
func (pb *ProductBrowser) View() tea.View {
	header := pb.renderHeader()
	footer := pb.renderHelp()
	middleHeight := pb.middleHeight()
	topHeight, bottomHeight := pb.middleSplit(middleHeight)
	pb.configureLeftList(pb.width, topHeight)

	// Top: product table spans the full width.
	topPane := pb.renderProductList(pb.width, topHeight)

	// Bottom: four equal-width panes side by side.
	paneW := pb.width / 4
	lastPaneW := max(pb.width-3*paneW, 0)
	infoPane := pb.renderInfoPane(paneW, bottomHeight)
	cannabinoidsPane := pb.renderCannabinoidsChart(paneW, bottomHeight)
	terpenesPane := pb.renderTerpenesChart(paneW, bottomHeight)
	picturePane := pb.renderPicturePane(lastPaneW, bottomHeight)
	bottomRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		infoPane,
		cannabinoidsPane,
		terpenesPane,
		picturePane,
	)

	content := lipgloss.JoinVertical(lipgloss.Left, topPane, bottomRow)
	content = lipgloss.NewStyle().
		Width(pb.width).
		MaxWidth(pb.width).
		Render(content)

	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		pb.renderPageFooterBar(),
		footer,
	))
}

func (pb *ProductBrowser) renderPageFooterBar() string {
	parts := []string{fmt.Sprintf("products: %d", len(pb.products))}
	if pb.activeFilter != "" {
		parts = append(parts, pb.activeFilter)
	}
	if label := pb.dateSortLabel(); label != "" {
		parts = append(parts, "sort: "+label)
	}
	return renderPageFooter(pb.width, strings.Join(parts, "  ·  "), productsUpstreamURL)
}

func (pb *ProductBrowser) renderProductList(outerWidth, outerHeight int) string {
	style := pb.listPaneStyle()
	if pb.filterMode != FilterModeNone {
		return style.
			Width(outerWidth).
			Height(outerHeight).
			Render(pb.leftList.View())
	}

	title := "Products"
	if label := pb.currentFilterLabel(); label != "" {
		title = fmt.Sprintf("%s [%s]", title, label)
	}
	titleLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")).
		Bold(true).
		Render(title)
	content := lipgloss.JoinVertical(lipgloss.Left, titleLine, pb.tbl.View())
	return style.
		Width(outerWidth).
		Height(outerHeight).
		Render(content)
}

func (pb *ProductBrowser) renderInfoPane(outerWidth, outerHeight int) string {
	style := pb.infoPaneStyle()
	innerHeight := max(outerHeight-style.GetVerticalFrameSize(), 0)

	if len(pb.products) == 0 {
		return style.
			Width(outerWidth).
			Height(outerHeight).
			Render("No product selected")
	}

	product := pb.products[pb.selectedIdx]

	var info strings.Builder
	writeProductDetailLine(&info, pb, "brand_name", product.BrandName)
	writeProductDetailLine(&info, pb, "dosage_form", product.DosageForm)
	writeProductDetailLine(&info, pb, "producer", productProducer(product))
	if !product.ApprovalDate.IsZero() {
		writeProductDetailLine(&info, pb, "approval_date", product.ApprovalDate.Format("2006-01-02"))
	} else {
		writeProductDetailLine(&info, pb, "approval_date", "")
	}
	writeProductDetailLine(&info, pb, "registration_number", product.RegistrationNumber)
	writeProductDetailLine(&info, pb, "market", product.Market)
	writeProductDetailLine(&info, pb, "chemotype", productType(product))
	writeProductDetailLine(&info, pb, "processing_technique", product.ProcessingTechnique)
	writeProductDetailLine(&info, pb, "solvents_user", product.SolventsUser)
	writeProductDetailLine(&info, pb, "national_drug_code", product.NationalDrugCode)

	if product.THC > 0 {
		info.WriteString("\n")
		info.WriteString(pb.styledLabel("THC: "))
		_, _ = fmt.Fprintf(&info, "%.2f%%", product.THC)
		info.WriteString("\n")
	}

	if product.THCA > 0 {
		info.WriteString(pb.styledLabel("THCA: "))
		_, _ = fmt.Fprintf(&info, "%.2f%%", product.THCA)
		info.WriteString("\n")
	}

	if product.CBD > 0 {
		info.WriteString(pb.styledLabel("CBD: "))
		_, _ = fmt.Fprintf(&info, "%.2f%%", product.CBD)
		info.WriteString("\n")
	}

	if product.CBDA > 0 {
		info.WriteString(pb.styledLabel("CBDA: "))
		_, _ = fmt.Fprintf(&info, "%.2f%%", product.CBDA)
		info.WriteString("\n")
	}

	if len(product.Compounds) > 0 {
		info.WriteString("\n")
		info.WriteString(pb.styledLabel("Top Compounds:"))
		info.WriteString("\n")
		for _, c := range product.Compounds {
			_, _ = fmt.Fprintf(&info, "  • %s: %.2f%%\n", c.Name, c.Percentage)
		}
	}

	innerWidth := max(outerWidth-style.GetHorizontalFrameSize(), 0)
	content := fitPaneContent(info.String(), innerWidth, innerHeight)
	return style.
		Width(outerWidth).
		Height(outerHeight).
		Render(content)
}

// renderPicturePane wraps pictureView.View in the pane border. The inner
// content — fetch state ("Loading…"), glyph render, or Kitty placeholder
// grid — is owned by pictureView; this function only applies the pane frame.
func (pb *ProductBrowser) renderPicturePane(outerWidth, outerHeight int) string {
	style := pb.infoPaneStyle()
	if len(pb.products) == 0 {
		return style.Width(outerWidth).Height(outerHeight).Render("No product selected")
	}
	return style.Width(outerWidth).Height(outerHeight).Render(pb.picture.View())
}

// fitPaneContent truncates each line to innerWidth (using ansi.Truncate so
// styled prefixes survive) and clips the line count to innerHeight. This
// prevents lipgloss from wrapping lines that are wider than the pane, which
// would otherwise grow the rendered height beyond Height(outerHeight).
func fitPaneContent(content string, innerWidth, innerHeight int) string {
	if innerWidth <= 0 || innerHeight <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	if len(lines) > innerHeight {
		lines = lines[:innerHeight]
	}
	for i, line := range lines {
		if lipgloss.Width(line) > innerWidth {
			lines[i] = ansi.Truncate(line, innerWidth, "")
		}
	}
	return strings.Join(lines, "\n")
}

type barEntry struct {
	name  string
	value float64
}

func (pb *ProductBrowser) renderCannabinoidsChart(outerWidth, outerHeight int) string {
	style := pb.chartPaneStyle()
	if len(pb.products) == 0 {
		return style.Width(outerWidth).Height(outerHeight).Render("No product selected")
	}

	product := pb.products[pb.selectedIdx]
	var entries []barEntry
	addFixed := func(name string, v float64) {
		if v > 0 {
			entries = append(entries, barEntry{name, v})
		}
	}
	addFixed("THC", product.THC)
	addFixed("CBD", product.CBD)
	addFixed("THCA", product.THCA)
	addFixed("CBDA", product.CBDA)

	others := make([]barEntry, 0, len(product.OtherCannabinoids))
	for _, c := range product.OtherCannabinoids {
		if c.Percentage > 0 {
			others = append(others, barEntry{c.Name, c.Percentage})
		}
	}
	sort.Slice(others, func(i, j int) bool {
		return others[i].value > others[j].value
	})
	if len(others) > 20 {
		others = others[:20]
	}
	entries = append(entries, others...)

	return pb.renderBarChartBox(outerWidth, outerHeight, entries, "No cannabinoid data available")
}

func (pb *ProductBrowser) renderTerpenesChart(outerWidth, outerHeight int) string {
	style := pb.chartPaneStyle()
	if len(pb.products) == 0 {
		return style.Width(outerWidth).Height(outerHeight).Render("No product selected")
	}

	product := pb.products[pb.selectedIdx]
	entries := make([]barEntry, 0, len(product.Compounds))
	for _, c := range product.Compounds {
		if c.Percentage > 0 {
			entries = append(entries, barEntry{c.Name, c.Percentage})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].value > entries[j].value
	})
	if len(entries) > 20 {
		entries = entries[:20]
	}

	// Normalize every terpene label to the visual width of the longest name
	// ("β-Caryophyllene" = 15 cells) so the axis position is stable regardless
	// of which terpenes are present.
	const terpLabelWidth = 15
	for i, e := range entries {
		w := lipgloss.Width(e.name)
		switch {
		case w > terpLabelWidth:
			entries[i].name = ansi.Truncate(e.name, terpLabelWidth, "")
		case w < terpLabelWidth:
			entries[i].name = e.name + strings.Repeat(" ", terpLabelWidth-w)
		}
	}

	return pb.renderBarChartBox(outerWidth, outerHeight, entries, "No terpene data available")
}

func (pb *ProductBrowser) renderBarChartBox(outerWidth, outerHeight int, entries []barEntry, emptyMsg string) string {
	style := pb.chartPaneStyle()
	innerWidth := max(outerWidth-style.GetHorizontalFrameSize(), 0)
	innerHeight := max(outerHeight-style.GetVerticalFrameSize(), 0)

	if len(entries) == 0 {
		return style.Width(outerWidth).Height(outerHeight).Render(emptyMsg)
	}

	maxBars := (innerHeight + 1) / 2
	if maxBars > 0 && len(entries) > maxBars {
		entries = entries[:maxBars]
	}

	var maxVal float64
	for _, e := range entries {
		if e.value > maxVal {
			maxVal = e.value
		}
	}

	barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Background(lipgloss.Color("46"))
	barData := make([]barchart.BarData, 0, len(entries))
	for _, e := range entries {
		barData = append(barData, barchart.BarData{
			Label:  e.name,
			Values: []barchart.BarValue{{Name: e.name, Value: e.value, Style: barStyle}},
		})
	}

	content := "Window too small for chart"
	if innerWidth >= 12 && innerHeight >= 4 {
		axisStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
		// Size the canvas to exactly what the bars need so the axis line
		// doesn't extend below the last bar.
		chartHeight := max(min(len(entries)*2-1, innerHeight), 1)
		chart := barchart.New(
			max(innerWidth, 1),
			chartHeight,
			barchart.WithNoAutoBarWidth(),
			barchart.WithBarWidth(1),
			barchart.WithBarGap(1),
			barchart.WithMaxValue(maxVal),
			barchart.WithStyles(axisStyle, labelStyle),
			barchart.WithDataSet(barData),
			barchart.WithHorizontalBars(),
		)
		chart.Draw()

		// ntcharts uses byte length to set origin.X (the axis column), so mirror
		// that here — otherwise multi-byte labels like β-Caryophyllene put the
		// overlay one column too far left.
		maxLabelLen := 0
		for _, e := range entries {
			if n := len(e.name); n > maxLabelLen {
				maxLabelLen = n
			}
		}
		barStartX := maxLabelLen + 1
		barMaxCells := max(innerWidth-barStartX, 0)
		// Inside the bar: dark text on the bar's green so it reads on the filled block.
		onBarStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("46")).
			Bold(true)
		// Past the bar end: green text on the terminal default so it stays
		// readable on dark and light terminals (no fixed background).
		offBarStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)
		rowStride := chart.BarWidth() + chart.BarGap()
		for i, e := range entries {
			text := fmt.Sprintf("%.2f%%", e.value)
			cells := 0
			if maxVal > 0 {
				cells = int(e.value / maxVal * float64(barMaxCells))
			}
			y := i * rowStride
			inside := min(cells, len(text))
			if inside > 0 {
				chart.Canvas.SetStringWithStyle(canvas.Point{X: barStartX, Y: y}, text[:inside], onBarStyle)
			}
			if inside < len(text) {
				chart.Canvas.SetStringWithStyle(canvas.Point{X: barStartX + inside, Y: y}, text[inside:], offBarStyle)
			}
		}

		content = chart.View()
	}

	return style.Width(outerWidth).Height(outerHeight).Render(content)
}

func (pb *ProductBrowser) renderHeader() string {
	return renderAppHeader(pb.width, pb.activePage)
}

// renderAppHeader renders the single-row top bar. Layout priority, from most
// to least important as width shrinks: decorative title (right) → page tabs
// → data-source label. The tabs highlight the active page.
func renderAppHeader(width int, active Page) string {
	if width <= 0 {
		return ""
	}
	barStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("230")).
		Bold(true)
	dimStyle := barStyle.Foreground(lipgloss.Color("245"))
	activeStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("230")).
		Foreground(lipgloss.Color("236")).
		Bold(true)

	var tabs []string
	for i, t := range pageTabs {
		text := " " + t.key + ":" + t.label + " "
		if Page(i) == active {
			tabs = append(tabs, activeStyle.Render(text))
		} else {
			tabs = append(tabs, barStyle.Render(text))
		}
	}
	tabStrip := strings.Join(tabs, dimStyle.Render(" "))

	rightRendered := barStyle.Render(ansi.Truncate(renderFaceSlot()+appTitle, width, "") + " ")
	rightWidth := lipgloss.Width(rightRendered)

	sourceLabel := barStyle.Render(" USA-CT Cannabis Data (data.ct.gov) ") + dimStyle.Render("│ ")
	sourceWidth := lipgloss.Width(sourceLabel)
	tabWidth := lipgloss.Width(tabStrip)

	// Try full layout; progressively drop pieces if over budget.
	left := sourceLabel + tabStrip
	if sourceWidth+tabWidth+rightWidth > width {
		left = tabStrip // drop the data source
	}
	if lipgloss.Width(left)+rightWidth > width {
		left = "" // drop tabs too; keep only the decorative title
	}

	pad := max(width-lipgloss.Width(left)-rightWidth, 0)
	return left + barStyle.Render(strings.Repeat(" ", pad)) + rightRendered
}

// renderPageFooter renders a single-row bar with page-state on the left and
// the upstream data URL right-justified. Both pieces are truncated to fit if
// the window is narrow; the URL is kept whole (no ellipsis) so it remains
// copy-pasteable, but will be dropped entirely before any other content
// spills off the right edge.
func renderPageFooter(width int, status, url string) string {
	if width <= 0 {
		return ""
	}
	barStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252"))
	urlStyle := barStyle.Foreground(lipgloss.Color("117")) // soft blue
	right := ""
	rightW := 0
	if url != "" {
		right = urlStyle.Render(" " + url + " ")
		rightW = lipgloss.Width(right)
	}
	// Drop the URL entirely if there isn't room for it plus at least a space.
	if rightW+1 > width {
		right = ""
		rightW = 0
	}
	leftBudget := max(width-rightW, 0)
	left := barStyle.Render(" " + ansi.Truncate(status, max(leftBudget-1, 0), "…"))
	leftW := lipgloss.Width(left)
	pad := max(width-leftW-rightW, 0)
	return left + barStyle.Render(strings.Repeat(" ", pad)) + right
}

func (pb *ProductBrowser) renderHelp() string {
	if pb.width <= 0 {
		return ""
	}

	helpText := pb.help.View(browserHelpKeyMap{filterMode: pb.filterMode})
	return lipgloss.NewStyle().
		Width(pb.width).
		MaxWidth(pb.width).
		MaxHeight(1).
		Background(lipgloss.Color("238")).
		Foreground(lipgloss.Color("252")).
		Render(helpText)
}

func (pb *ProductBrowser) listPaneStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("4")).
		Padding(0, 1)
}

func (pb *ProductBrowser) infoPaneStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(1, 2)
}

func (pb *ProductBrowser) chartPaneStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("5")).
		Padding(0, 1)
}

func (pb *ProductBrowser) styledLabel(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("3")).
		Render(text)
}

func (pb *ProductBrowser) updateInfoPane() {
	// Update viewport if needed
	pb.infoPaneView.SetHeight(pb.middleHeight())
	pb.infoPaneView.SetWidth(pb.width / 2)
}

func (pb *ProductBrowser) updateDimensions(width, height int) {
	pb.width = width
	pb.height = height
	pb.help.SetWidth(width)
	pb.infoPaneView = viewport.New(viewport.WithWidth(width/2), viewport.WithHeight(pb.middleHeight()))
}

func (pb *ProductBrowser) configureLeftList(outerWidth, outerHeight int) {
	style := pb.listPaneStyle()
	innerWidth := max(outerWidth-style.GetHorizontalFrameSize(), 0)
	innerHeight := max(outerHeight-style.GetVerticalFrameSize(), 0)

	if pb.filterMode != FilterModeNone {
		pb.leftList.SetSize(innerWidth, innerHeight)
		pb.leftList.Title = pb.filterTitle
		pb.leftList.SetShowTitle(true)
		pb.filterIdx = pb.leftList.Index()
		return
	}

	// Products mode: one-line title above the table.
	tableHeight := max(innerHeight-1, 1)
	pb.tbl.SetWidth(innerWidth)
	pb.tbl.SetHeight(tableHeight)
	pb.tbl.SetColumns(productTableColumns(innerWidth))
	pb.selectedIdx = pb.tbl.Cursor()
}

// productTableColumns distributes innerW across brand / approval date / type.
// The bubbles/v2 table widget renders each cell with 1-char L/R padding, so
// the natural width is sum(widths) + 2*ncols; overflow wraps via lipgloss and
// doubles the pane height. The sum-of-widths budget here is innerW - 6 (3
// columns × 2-char padding). When space allows, the date column is pinned to
// its header width (13, "approval date"); otherwise it shrinks to 10 (body
// "YYYY-MM-DD") and the header truncates with an ellipsis.
func productTableColumns(innerW int) []table.Column {
	const (
		ncols       = 3
		dateHeaderW = 13 // "approval date"
		dateBodyW   = 10 // "YYYY-MM-DD"
	)
	budget := max(innerW-2*ncols, 3)

	dateW := dateBodyW
	if budget >= dateHeaderW+6 {
		dateW = dateHeaderW
	}
	if dateW > budget-2 {
		dateW = max(budget-2, 1)
	}

	remainder := max(budget-dateW, 2)
	brandW := max(remainder*3/5, 1)
	typeW := max(remainder-brandW, 1)
	return []table.Column{
		{Title: "brand", Width: brandW},
		{Title: "approval date", Width: dateW},
		{Title: "type", Width: typeW},
	}
}

// loadSelectedProductDetails enriches the currently selected product with compound data.
func (pb *ProductBrowser) loadSelectedProductDetails() {
	if pb.loader == nil || len(pb.products) == 0 {
		return
	}
	reg := pb.products[pb.selectedIdx].RegistrationNumber
	if reg == "" {
		return
	}
	p, err := pb.loader.LoadProductWithCompounds(reg)
	if err == nil && p != nil {
		pb.products[pb.selectedIdx] = *p
	}
}

func (pb *ProductBrowser) openFilter(mode FilterMode) {
	var err error
	pb.filterMode = mode
	pb.filterIdx = 0
	pb.filterOptions = nil

	switch mode {
	case FilterModeByBrand:
		pb.filterTitle = "Filter By Brand"
		err = pb.buildBrandFilterOptions()
	case FilterModeByName:
		pb.filterTitle = "Filter By Name"
		err = pb.buildNameFilterOptions()
	case FilterModeByType:
		pb.filterTitle = "Filter By Type"
		err = pb.buildTypeFilterOptions()
	case FilterModeByForm:
		pb.filterTitle = "Filter By Form"
		err = pb.buildFormFilterOptions()
	default:
		pb.filterMode = FilterModeNone
		return
	}

	if err != nil || len(pb.filterOptions) == 0 {
		pb.filterMode = FilterModeNone
		pb.filterTitle = ""
		pb.filterOptions = nil
		pb.filterIdx = 0
		pb.setProductItems()
		return
	}

	pb.setFilterItems()
}

func (pb *ProductBrowser) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		pb.applySelectedFilter()
		return pb, pb.syncPictureURL()
	case "esc":
		pb.cancelFilter()
		return pb, nil
	case "ctrl+c", "q":
		return pb, tea.Quit
	}

	oldIndex := pb.leftList.Index()
	var cmd tea.Cmd
	pb.leftList, cmd = pb.leftList.Update(msg)
	pb.filterIdx = pb.leftList.Index()
	if pb.filterIdx != oldIndex {
		return pb, cmd
	}
	return pb, cmd
}

func (pb *ProductBrowser) applySelectedFilter() {
	pb.filterIdx = pb.leftList.Index()
	if pb.filterIdx < 0 || pb.filterIdx >= len(pb.filterOptions) {
		pb.cancelFilter()
		return
	}

	mode := pb.filterMode
	value := pb.filterOptions[pb.filterIdx]
	pb.cancelFilter()

	var (
		products []models.Product
		err      error
	)
	switch mode {
	case FilterModeByBrand:
		products, err = pb.loadProductsByBrand(value)
	case FilterModeByName:
		products, err = pb.loadProductsByName(value)
	case FilterModeByType:
		products, err = pb.loadProductsByType(value)
	case FilterModeByForm:
		products, err = pb.loadProductsByForm(value)
	default:
		return
	}

	if err != nil {
		return
	}

	pb.products = products
	pb.selectedIdx = 0
	pb.activeFilter = pb.filterLabel(mode, value)
	pb.applyDateSort()
	pb.setProductItems()
	if len(pb.products) > 0 {
		pb.loadSelectedProductDetails()
	}
	pb.updateInfoPane()
}

func (pb *ProductBrowser) cancelFilter() {
	pb.filterMode = FilterModeNone
	pb.filterOptions = nil
	pb.filterIdx = 0
	pb.filterTitle = ""
	pb.setProductItems()
}

func (pb *ProductBrowser) clearFilter() {
	pb.cancelFilter()
	pb.activeFilter = ""
	pb.products = append([]models.Product(nil), pb.allProducts...)
	pb.selectedIdx = 0
	pb.applyDateSort()
	pb.setProductItems()
	if len(pb.products) > 0 {
		pb.loadSelectedProductDetails()
	}
	pb.updateInfoPane()
}

func (pb *ProductBrowser) toggleDateSort() {
	selectedID := ""
	if len(pb.products) > 0 && pb.selectedIdx >= 0 && pb.selectedIdx < len(pb.products) {
		selectedID = pb.products[pb.selectedIdx].RegistrationNumber
	}

	switch pb.dateSort {
	case ProductDateSortNewest:
		pb.dateSort = ProductDateSortOldest
	default:
		pb.dateSort = ProductDateSortNewest
	}
	pb.applyDateSort()
	if selectedID != "" {
		for i, product := range pb.products {
			if product.RegistrationNumber == selectedID {
				pb.selectedIdx = i
				break
			}
		}
	}
	pb.setProductItems()
	pb.updateInfoPane()
}

func (pb *ProductBrowser) applyDateSort() {
	switch pb.dateSort {
	case ProductDateSortNewest:
		sort.SliceStable(pb.products, func(i, j int) bool {
			return productDateNewer(pb.products[i], pb.products[j])
		})
	case ProductDateSortOldest:
		sort.SliceStable(pb.products, func(i, j int) bool {
			return productDateOlder(pb.products[i], pb.products[j])
		})
	}
}

func productDateNewer(a, b models.Product) bool {
	aZero := a.ApprovalDate.IsZero()
	bZero := b.ApprovalDate.IsZero()
	switch {
	case aZero && bZero:
		return false
	case aZero:
		return false
	case bZero:
		return true
	default:
		return a.ApprovalDate.After(b.ApprovalDate)
	}
}

func productDateOlder(a, b models.Product) bool {
	aZero := a.ApprovalDate.IsZero()
	bZero := b.ApprovalDate.IsZero()
	switch {
	case aZero && bZero:
		return false
	case aZero:
		return false
	case bZero:
		return true
	default:
		return a.ApprovalDate.Before(b.ApprovalDate)
	}
}

func (pb *ProductBrowser) dateSortLabel() string {
	switch pb.dateSort {
	case ProductDateSortNewest:
		return "approval_date desc"
	case ProductDateSortOldest:
		return "approval_date asc"
	default:
		return ""
	}
}

func (pb *ProductBrowser) currentFilterLabel() string {
	return pb.activeFilter
}

func (pb *ProductBrowser) filterLabel(mode FilterMode, value string) string {
	switch mode {
	case FilterModeByBrand:
		return "brand: " + value
	case FilterModeByName:
		return "name: " + value
	case FilterModeByType:
		return "type: " + value
	case FilterModeByForm:
		return "form: " + value
	default:
		return value
	}
}

func (pb *ProductBrowser) buildBrandFilterOptions() error {
	if pb.loader != nil {
		options, err := pb.loader.GetDistinctBrands()
		if err != nil {
			return err
		}
		pb.filterOptions = options
		pb.preselectActiveFilter(FilterModeByBrand)
		return nil
	}

	brandMap := make(map[string]struct{})
	for _, p := range pb.allProducts {
		brandMap[p.BrandName] = struct{}{}
	}
	for brand := range brandMap {
		pb.filterOptions = append(pb.filterOptions, brand)
	}
	sort.Strings(pb.filterOptions)
	pb.preselectActiveFilter(FilterModeByBrand)
	return nil
}

func (pb *ProductBrowser) buildNameFilterOptions() error {
	if pb.loader != nil {
		options, err := pb.loader.GetDistinctNames()
		if err != nil {
			return err
		}
		pb.filterOptions = options
		pb.preselectActiveFilter(FilterModeByName)
		return nil
	}

	nameMap := make(map[string]struct{})
	for _, p := range pb.allProducts {
		nameMap[p.BrandName] = struct{}{}
	}
	for name := range nameMap {
		pb.filterOptions = append(pb.filterOptions, name)
	}
	sort.Strings(pb.filterOptions)
	pb.preselectActiveFilter(FilterModeByName)
	return nil
}

func (pb *ProductBrowser) buildTypeFilterOptions() error {
	if pb.loader != nil {
		options, err := pb.loader.GetDistinctTypes()
		if err != nil {
			return err
		}
		pb.filterOptions = options
		pb.preselectActiveFilter(FilterModeByType)
		return nil
	}

	typeMap := make(map[string]struct{})
	for _, p := range pb.allProducts {
		typeMap[productType(p)] = struct{}{}
	}
	for t := range typeMap {
		pb.filterOptions = append(pb.filterOptions, t)
	}
	sort.Strings(pb.filterOptions)
	pb.preselectActiveFilter(FilterModeByType)
	return nil
}

func (pb *ProductBrowser) buildFormFilterOptions() error {
	if pb.loader != nil {
		options, err := pb.loader.GetDistinctForms()
		if err != nil {
			return err
		}
		pb.filterOptions = options
		pb.preselectActiveFilter(FilterModeByForm)
		return nil
	}

	formMap := make(map[string]struct{})
	for _, p := range pb.allProducts {
		formMap[productForm(p)] = struct{}{}
	}
	for form := range formMap {
		pb.filterOptions = append(pb.filterOptions, form)
	}
	sort.Strings(pb.filterOptions)
	pb.preselectActiveFilter(FilterModeByForm)
	return nil
}

func (pb *ProductBrowser) preselectActiveFilter(mode FilterMode) {
	wantPrefix := ""
	switch mode {
	case FilterModeByBrand:
		wantPrefix = "brand: "
	case FilterModeByName:
		wantPrefix = "name: "
	case FilterModeByType:
		wantPrefix = "type: "
	case FilterModeByForm:
		wantPrefix = "form: "
	}

	if wantPrefix == "" || !strings.HasPrefix(pb.activeFilter, wantPrefix) {
		return
	}

	currentValue := strings.TrimPrefix(pb.activeFilter, wantPrefix)
	for i, option := range pb.filterOptions {
		if option == currentValue {
			pb.filterIdx = i
			return
		}
	}
}

func (pb *ProductBrowser) loadProductsByBrand(brand string) ([]models.Product, error) {
	if pb.loader != nil {
		return pb.loader.LoadProductsByBrand(brand)
	}

	var products []models.Product
	for _, product := range pb.allProducts {
		if strings.EqualFold(product.BrandName, brand) {
			products = append(products, product)
		}
	}
	return products, nil
}

func (pb *ProductBrowser) loadProductsByName(name string) ([]models.Product, error) {
	if pb.loader != nil {
		return pb.loader.LoadProductsByName(name)
	}

	var products []models.Product
	for _, product := range pb.allProducts {
		if strings.EqualFold(product.BrandName, name) {
			products = append(products, product)
		}
	}
	return products, nil
}

func (pb *ProductBrowser) loadProductsByType(selectedType string) ([]models.Product, error) {
	if pb.loader != nil {
		return pb.loader.LoadProductsByType(selectedType)
	}

	var products []models.Product
	for _, product := range pb.allProducts {
		if strings.EqualFold(productType(product), selectedType) {
			products = append(products, product)
		}
	}
	return products, nil
}

func (pb *ProductBrowser) loadProductsByForm(selectedForm string) ([]models.Product, error) {
	if pb.loader != nil {
		return pb.loader.LoadProductsByForm(selectedForm)
	}

	var products []models.Product
	for _, product := range pb.allProducts {
		if strings.EqualFold(productForm(product), selectedForm) {
			products = append(products, product)
		}
	}
	return products, nil
}

func (pb *ProductBrowser) middleHeight() int {
	// header (1) + page footer (1) + help (1) = 3
	return max(pb.height-3, 0)
}

// middleSplit partitions the middle vertical space between the top (products
// table) and bottom (four detail panes). The bottom gets enough height for
// one row of bordered panes plus a few content rows; everything else goes to
// the table so the user can see more products at once.
func (pb *ProductBrowser) middleSplit(totalHeight int) (int, int) {
	if totalHeight <= 0 {
		return 0, 0
	}
	bottomHeight := min(totalHeight/2, 12)
	bottomHeight = max(bottomHeight, min(totalHeight, 6))
	topHeight := max(totalHeight-bottomHeight, 0)
	return topHeight, bottomHeight
}

func newProductTable() table.Model {
	styles := table.DefaultStyles()
	styles.Header = styles.Header.
		Background(lipgloss.Color(tableHeaderBg)).
		Foreground(lipgloss.Color("230")).
		Bold(true)
	styles.Selected = styles.Selected.
		Background(lipgloss.Color(tableSelectedBg)).
		Foreground(lipgloss.Color("230"))
	return table.New(
		table.WithColumns([]table.Column{
			{Title: "brand", Width: 12},
			{Title: "approval date", Width: 13},
			{Title: "type", Width: 8},
		}),
		table.WithFocused(true),
		table.WithStyles(styles),
	)
}

func newBrowserList() list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)
	delegate.Styles.NormalTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).PaddingLeft(2)
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")).
		Bold(true).
		PaddingLeft(2)
	delegate.Styles.DimmedTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).PaddingLeft(2)

	l := list.New(nil, delegate, 0, 0)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()
	l.SetStatusBarItemName("product", "products")
	l.Styles.TitleBar = lipgloss.NewStyle()
	l.Styles.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	l.Styles.NoItems = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).PaddingLeft(2)
	return l
}

func (pb *ProductBrowser) setProductItems() {
	rows := make([]table.Row, 0, len(pb.products))
	for _, product := range pb.products {
		brand := product.BrandName
		if brand == "" {
			brand = product.BrandingEntity
		}
		approved := ""
		if !product.ApprovalDate.IsZero() {
			approved = product.ApprovalDate.Format("2006-01-02")
		}
		rows = append(rows, table.Row{brand, approved, productType(product)})
	}
	pb.tbl.SetRows(rows)
	if len(rows) == 0 {
		pb.tbl.SetCursor(0)
		pb.selectedIdx = 0
		return
	}

	pb.selectedIdx = min(pb.selectedIdx, len(rows)-1)
	pb.tbl.SetCursor(pb.selectedIdx)
}

func (pb *ProductBrowser) setFilterItems() {
	items := make([]list.Item, 0, len(pb.filterOptions))
	for _, option := range pb.filterOptions {
		items = append(items, FilterOptionItem{value: option})
	}
	pb.leftList.SetItems(items)
	if len(items) == 0 {
		pb.leftList.Select(0)
		pb.filterIdx = 0
		return
	}

	pb.filterIdx = min(pb.filterIdx, len(items)-1)
	pb.leftList.Select(pb.filterIdx)
}

func productType(product models.Product) string {
	value := productTypeValue(product)
	if value == "" {
		return "Unknown"
	}
	return value
}

func productTypeValue(product models.Product) string {
	return strings.TrimSpace(product.Chemotype)
}

func productForm(product models.Product) string {
	value := strings.TrimSpace(product.DosageForm)
	if value == "" {
		return "Unknown"
	}
	return value
}

func productProducer(product models.Product) string {
	if strings.TrimSpace(product.Producer) != "" {
		return product.Producer
	}
	return product.BrandingEntity
}

func writeProductDetailLine(info *strings.Builder, pb *ProductBrowser, label, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "-"
	}
	info.WriteString(pb.styledLabel(label + ": "))
	info.WriteString(value)
	info.WriteString("\n")
}
