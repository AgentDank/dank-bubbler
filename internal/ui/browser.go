// Package ui provides BubbleTea UI components for the dank-bubbler application
package ui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/barchart"

	"github.com/AgentDank/dank-bubbler/internal/data"
	"github.com/AgentDank/dank-bubbler/internal/models"
)

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
	focused       bool
	loader        *data.Loader
	activeFilter  string
}

// FilterMode represents the current filter type
type FilterMode int

const (
	FilterModeNone FilterMode = iota
	FilterModeByBrand
	FilterModeByType
	FilterModeByDate
)

// ProductItem implements the list.Item interface for products
type ProductItem struct {
	product models.Product
}

func (p ProductItem) FilterValue() string {
	return strings.ToLower(p.product.BrandName)
}

func (p ProductItem) String() string {
	return fmt.Sprintf("%s (%s)", p.product.BrandName, p.product.DosageForm)
}

// NewProductBrowser creates a new product browser component
func NewProductBrowser(products []models.Product, brands []models.Brand, loader *data.Loader) *ProductBrowser {
	productCopy := append([]models.Product(nil), products...)
	pb := &ProductBrowser{
		products:    productCopy,
		allProducts: append([]models.Product(nil), products...),
		brands:      brands,
		selectedIdx: 0,
		focused:     true,
		filterMode:  FilterModeNone,
		loader:      loader,
	}
	pb.updateDimensions(80, 24)
	// Prime selected product details
	pb.loadSelectedProductDetails()
	return pb
}

// Init initializes the product browser
func (pb *ProductBrowser) Init() tea.Cmd {
	return nil
}

// Update handles messages for the product browser
func (pb *ProductBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		pb.width = msg.Width
		pb.height = msg.Height
		pb.updateDimensions(msg.Width, msg.Height)

	case tea.KeyMsg:
		if pb.filterMode != FilterModeNone {
			return pb.updateFilter(msg)
		}

		switch msg.String() {
		case "up", "k":
			if pb.selectedIdx > 0 {
				pb.selectedIdx--
				pb.loadSelectedProductDetails()
				pb.updateInfoPane()
			}

		case "down", "j":
			if pb.selectedIdx < len(pb.products)-1 {
				pb.selectedIdx++
				pb.loadSelectedProductDetails()
				pb.updateInfoPane()
			}

		case "home":
			pb.selectedIdx = 0
			pb.loadSelectedProductDetails()
			pb.updateInfoPane()

		case "end":
			if len(pb.products) > 0 {
				pb.selectedIdx = len(pb.products) - 1
				pb.loadSelectedProductDetails()
				pb.updateInfoPane()
			}

		case "b": // Filter by brand
			pb.openFilter(FilterModeByBrand)

		case "t": // Filter by type
			pb.openFilter(FilterModeByType)

		case "c":
			pb.clearFilter()

		case "f": // Toggle focused mode
			pb.focused = !pb.focused

		case "ctrl+c", "q":
			return pb, tea.Quit
		}
	}

	return pb, nil
}

// View renders the product browser
func (pb *ProductBrowser) View() tea.View {

	if len(pb.products) == 0 {
		return tea.NewView("No products loaded. Check your database connection.\n")
	}

	// Left pane: product list (1/3 width)
	leftPane := pb.renderProductList()

	// Right panes: top and bottom
	rightTopPane := pb.renderInfoPane()
	rightBottomPane := pb.renderCompoundsChart()

	// Combine right panes vertically
	rightPane := lipgloss.JoinVertical(
		lipgloss.Left,
		rightTopPane,
		rightBottomPane,
	)

	// Help footer
	helpText := pb.renderHelp()

	// Combine left and right horizontally
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPane,
		rightPane,
	)

	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		helpText,
	))
}

func (pb *ProductBrowser) renderProductList() string {
	targetWidth := pb.width / 3
	listWidth := targetWidth - 4 // Border(2) + Padding(2)

	targetHeight := pb.height - 3  // Leave room for help
	listHeight := targetHeight - 2 // Border(2)

	if listWidth < 0 {
		listWidth = 0
	}
	if listHeight < 0 {
		listHeight = 0
	}

	if pb.filterMode != FilterModeNone {
		return pb.renderFilterList(listWidth, listHeight)
	}

	var lines []string
	header := "Products"
	if label := pb.currentFilterLabel(); label != "" {
		header = fmt.Sprintf("%s [%s]", header, label)
	}
	lines = append(lines, pb.styledHeader(header))

	rowsAvailable := max(listHeight-1, 0)
	start := listStart(pb.selectedIdx, len(pb.products), rowsAvailable)

	for i := start; i < len(pb.products) && i < start+rowsAvailable; i++ {
		product := pb.products[i]

		prefix := "  "
		if i == pb.selectedIdx {
			prefix = "> "
		}

		label := fmt.Sprintf("%s (%s)", product.BrandName, product.DosageForm)
		line := fmt.Sprintf("%s%-*s", prefix, max(listWidth-3, 0), label)
		if len(line) > listWidth {
			line = line[:listWidth]
		}
		lines = append(lines, line)
	}

	if len(pb.products) == 0 {
		lines = append(lines, "  No matching products")
	}

	for i := len(lines); i < listHeight; i++ {
		lines = append(lines, strings.Repeat(" ", listWidth))
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("4")).
		Width(listWidth).
		Height(listHeight).
		Padding(0, 1).
		Render(content)
}

func (pb *ProductBrowser) renderInfoPane() string {
	totalRightWidth := pb.width - (pb.width / 3)
	infoWidth := totalRightWidth - 6 // Border(2) + Padding(4)

	totalHeight := pb.height - 3
	topHeight := totalHeight / 2
	infoHeight := topHeight - 4 // Border(2) + Padding(2)

	if infoWidth < 0 {
		infoWidth = 0
	}
	if infoHeight < 0 {
		infoHeight = 0
	}

	if len(pb.products) == 0 {
		return lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("6")).
			Width(infoWidth).
			Height(infoHeight).
			Padding(1, 2).
			Render("No product selected")
	}

	product := pb.products[pb.selectedIdx]

	var info strings.Builder
	info.WriteString(pb.styledLabel("Brand: "))
	info.WriteString(product.BrandName)
	info.WriteString("\n")

	info.WriteString(pb.styledLabel("Type: "))
	info.WriteString(product.DosageForm)
	info.WriteString("\n")

	info.WriteString(pb.styledLabel("Registration: "))
	info.WriteString(product.RegistrationNumber)
	info.WriteString("\n")

	if !product.ApprovalDate.IsZero() {
		info.WriteString(pb.styledLabel("Approved: "))
		info.WriteString(product.ApprovalDate.Format("2006-01-02"))
		info.WriteString("\n")
	}

	if product.THC > 0 {
		info.WriteString(pb.styledLabel("THC: "))
		info.WriteString(fmt.Sprintf("%.2f%%", product.THC))
		info.WriteString("\n")
	}

	if product.THCA > 0 {
		info.WriteString(pb.styledLabel("THCA: "))
		info.WriteString(fmt.Sprintf("%.2f%%", product.THCA))
		info.WriteString("\n")
	}

	if product.CBD > 0 {
		info.WriteString(pb.styledLabel("CBD: "))
		info.WriteString(fmt.Sprintf("%.2f%%", product.CBD))
		info.WriteString("\n")
	}

	if product.CBDA > 0 {
		info.WriteString(pb.styledLabel("CBDA: "))
		info.WriteString(fmt.Sprintf("%.2f%%", product.CBDA))
		info.WriteString("\n")
	}

	if len(product.Compounds) > 0 {
		info.WriteString("\n")
		info.WriteString(pb.styledLabel("Top Compounds:"))
		info.WriteString("\n")
		for _, c := range product.Compounds {
			info.WriteString(fmt.Sprintf("  • %s: %.2f%%\n", c.Name, c.Percentage))
		}
	}

	content := info.String()
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("6")).
		Width(infoWidth).
		Height(infoHeight).
		Padding(1, 2).
		Render(content)
}

func (pb *ProductBrowser) renderCompoundsChart() string {
	totalRightWidth := pb.width - (pb.width / 3)
	chartWidth := totalRightWidth - 4 // Border(2) + Padding(2)

	totalHeight := pb.height - 3
	topHeight := totalHeight / 2
	bottomHeight := totalHeight - topHeight
	chartHeight := bottomHeight - 2 // Border(2)

	if chartWidth < 0 {
		chartWidth = 0
	}
	if chartHeight < 0 {
		chartHeight = 0
	}

	if len(pb.products) == 0 {
		return lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("5")).
			Width(chartWidth).
			Height(chartHeight).
			Padding(1, 2).
			Render("No product selected")
	}

	product := pb.products[pb.selectedIdx]

	// Collect cannabinoids and terpenes with their percentages
	type compound struct {
		name  string
		value float64
	}

	var compounds []compound

	// Add main cannabinoids
	if product.THC > 0 {
		compounds = append(compounds, compound{"THC", product.THC})
	}
	if product.THCA > 0 {
		compounds = append(compounds, compound{"THCA", product.THCA})
	}
	if product.CBD > 0 {
		compounds = append(compounds, compound{"CBD", product.CBD})
	}
	if product.CBDA > 0 {
		compounds = append(compounds, compound{"CBDA", product.CBDA})
	}

	// Add terpenes from the selected product's derived compounds list.
	for _, c := range product.Compounds {
		compounds = append(compounds, compound{c.Name, c.Percentage})
	}

	if len(compounds) == 0 {
		return lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("5")).
			Width(chartWidth).
			Height(chartHeight).
			Padding(1, 2).
			Render("No compound data available")
	}

	// Sort by value descending and keep top 6
	sort.Slice(compounds, func(i, j int) bool {
		return compounds[i].value > compounds[j].value
	})

	if len(compounds) > 6 {
		compounds = compounds[:6]
	}

	// Get max value for scaling
	var maxVal float64
	for _, c := range compounds {
		if c.value > maxVal {
			maxVal = c.value
		}
	}

	// Create bar chart data
	var barData []barchart.BarData
	for _, c := range compounds {
		barData = append(barData, barchart.BarData{
			Label: c.name,
			Values: []barchart.BarValue{
				{
					Name:  c.name,
					Value: c.value,
					Style: lipgloss.NewStyle().Foreground(lipgloss.Color("46")),
				},
			},
		})
	}

	// Create and configure the chart
	chart := barchart.New(
		chartWidth-4,
		chartHeight-2,
		barchart.WithHorizontalBars(),
		barchart.WithMaxValue(maxVal),
		barchart.WithDataSet(barData),
	)

	content := chart.View()
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("5")).
		Width(chartWidth).
		Height(chartHeight).
		Padding(0, 1).
		Render(content)
}

func (pb *ProductBrowser) renderHelp() string {
	help := "↑/k: up  ↓/j: down  b: filter by brand  t: filter by type  c: clear filter  f: toggle focus  q: quit"
	if pb.filterMode != FilterModeNone {
		help = "↑/k: up  ↓/j: down  enter: apply filter  esc: cancel  q: quit"
	}
	if label := pb.currentFilterLabel(); label != "" {
		help += "  active: " + label
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(0, 1).
		Width(pb.width - 2).
		Render(help)
}

func (pb *ProductBrowser) styledHeader(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")).
		Bold(true).
		Render(text)
}

func (pb *ProductBrowser) styledLabel(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("3")).
		Render(text)
}

func (pb *ProductBrowser) updateInfoPane() {
	// Update viewport if needed
	pb.infoPaneView.SetHeight(pb.height - 3)
	pb.infoPaneView.SetWidth(pb.width / 2)
}

func (pb *ProductBrowser) updateDimensions(width, height int) {
	pb.width = width
	pb.height = height
	pb.infoPaneView = viewport.New(viewport.WithWidth(width/2), viewport.WithHeight(height-3))
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
	case FilterModeByType:
		pb.filterTitle = "Filter By Type"
		err = pb.buildTypeFilterOptions()
	default:
		pb.filterMode = FilterModeNone
		return
	}

	if err != nil || len(pb.filterOptions) == 0 {
		pb.filterMode = FilterModeNone
		pb.filterTitle = ""
		pb.filterOptions = nil
		pb.filterIdx = 0
	}
}

func (pb *ProductBrowser) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if pb.filterIdx > 0 {
			pb.filterIdx--
		}
	case "down", "j":
		if pb.filterIdx < len(pb.filterOptions)-1 {
			pb.filterIdx++
		}
	case "home":
		pb.filterIdx = 0
	case "end":
		if len(pb.filterOptions) > 0 {
			pb.filterIdx = len(pb.filterOptions) - 1
		}
	case "enter":
		pb.applySelectedFilter()
	case "esc":
		pb.cancelFilter()
	case "ctrl+c", "q":
		return pb, tea.Quit
	}

	return pb, nil
}

func (pb *ProductBrowser) renderFilterList(listWidth, listHeight int) string {
	var lines []string
	lines = append(lines, pb.styledHeader(pb.filterTitle))

	rowsAvailable := max(listHeight-1, 0)
	start := listStart(pb.filterIdx, len(pb.filterOptions), rowsAvailable)

	for i := start; i < len(pb.filterOptions) && i < start+rowsAvailable; i++ {
		prefix := "  "
		if i == pb.filterIdx {
			prefix = "> "
		}

		line := fmt.Sprintf("%s%-*s", prefix, max(listWidth-3, 0), pb.filterOptions[i])
		if len(line) > listWidth {
			line = line[:listWidth]
		}
		lines = append(lines, line)
	}

	if len(pb.filterOptions) == 0 {
		lines = append(lines, "  No options available")
	}

	for i := len(lines); i < listHeight; i++ {
		lines = append(lines, strings.Repeat(" ", listWidth))
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("4")).
		Width(listWidth).
		Height(listHeight).
		Padding(0, 1).
		Render(content)
}

func (pb *ProductBrowser) applySelectedFilter() {
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
	case FilterModeByType:
		products, err = pb.loadProductsByType(value)
	default:
		return
	}

	if err != nil {
		return
	}

	pb.products = products
	pb.selectedIdx = 0
	pb.activeFilter = pb.filterLabel(mode, value)
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
}

func (pb *ProductBrowser) clearFilter() {
	pb.cancelFilter()
	pb.activeFilter = ""
	pb.products = append([]models.Product(nil), pb.allProducts...)
	pb.selectedIdx = 0
	if len(pb.products) > 0 {
		pb.loadSelectedProductDetails()
	}
	pb.updateInfoPane()
}

func (pb *ProductBrowser) currentFilterLabel() string {
	return pb.activeFilter
}

func (pb *ProductBrowser) filterLabel(mode FilterMode, value string) string {
	switch mode {
	case FilterModeByBrand:
		return "brand: " + value
	case FilterModeByType:
		return "type: " + value
	case FilterModeByDate:
		return "date: " + value
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
		typeMap[p.DosageForm] = struct{}{}
	}
	for t := range typeMap {
		pb.filterOptions = append(pb.filterOptions, t)
	}
	sort.Strings(pb.filterOptions)
	pb.preselectActiveFilter(FilterModeByType)
	return nil
}

func (pb *ProductBrowser) preselectActiveFilter(mode FilterMode) {
	wantPrefix := ""
	switch mode {
	case FilterModeByBrand:
		wantPrefix = "brand: "
	case FilterModeByType:
		wantPrefix = "type: "
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

func (pb *ProductBrowser) loadProductsByType(productType string) ([]models.Product, error) {
	if pb.loader != nil {
		return pb.loader.LoadProductsByType(productType)
	}

	var products []models.Product
	for _, product := range pb.allProducts {
		if strings.EqualFold(product.DosageForm, productType) {
			products = append(products, product)
		}
	}
	return products, nil
}

func listStart(selectedIdx, totalItems, visibleRows int) int {
	if visibleRows <= 0 || totalItems <= visibleRows || selectedIdx < visibleRows {
		return 0
	}
	return selectedIdx - visibleRows + 1
}
