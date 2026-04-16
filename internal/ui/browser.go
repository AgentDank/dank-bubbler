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
	brands        []models.Brand
	selectedIdx   int
	width         int
	height        int
	infoPaneView  viewport.Model
	filterMode    FilterMode
	filterOptions []string
	filterIdx     int
	focused       bool
	loader        *data.Loader
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
	pb := &ProductBrowser{
		products:    products,
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
			pb.updateInfoPane()

		case "end":
			if len(pb.products) > 0 {
				pb.selectedIdx = len(pb.products) - 1
				pb.updateInfoPane()
			}

		case "b": // Filter by brand
			pb.filterMode = FilterModeByBrand
			pb.buildBrandFilterOptions()

		case "t": // Filter by type
			pb.filterMode = FilterModeByType
			pb.buildTypeFilterOptions()

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

	var lines []string
	lines = append(lines, pb.styledHeader("Products"))

	for i, product := range pb.products {
		if i >= listHeight-1 {
			break
		}

		prefix := "  "
		if i == pb.selectedIdx {
			prefix = "> "
		}

		line := fmt.Sprintf("%s%-*s", prefix, listWidth-3, product.BrandName)
		if len(line) > listWidth {
			line = line[:listWidth]
		}
		lines = append(lines, line)
	}

	for i := len(pb.products); i < listHeight-1; i++ {
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
	help := "↑/k: up  ↓/j: down  b: filter by brand  t: filter by type  f: toggle focus  q: quit"
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

func (pb *ProductBrowser) buildBrandFilterOptions() {
	brandMap := make(map[string]bool)
	for _, p := range pb.products {
		brandMap[p.BrandName] = true
	}
	for brand := range brandMap {
		pb.filterOptions = append(pb.filterOptions, brand)
	}
}

func (pb *ProductBrowser) buildTypeFilterOptions() {
	typeMap := make(map[string]bool)
	for _, p := range pb.products {
		typeMap[p.DosageForm] = true
	}
	for t := range typeMap {
		pb.filterOptions = append(pb.filterOptions, t)
	}
}
