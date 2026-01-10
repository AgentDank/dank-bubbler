// Package ui provides BubbleTea UI components for the dank-bubbler application
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
func NewProductBrowser(products []models.Product, brands []models.Brand) *ProductBrowser {
	pb := &ProductBrowser{
		products:    products,
		brands:      brands,
		selectedIdx: 0,
		focused:     true,
		filterMode:  FilterModeNone,
	}
	pb.updateDimensions(80, 24)
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
				pb.updateInfoPane()
			}

		case "down", "j":
			if pb.selectedIdx < len(pb.products)-1 {
				pb.selectedIdx++
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
func (pb *ProductBrowser) View() string {
	if len(pb.products) == 0 {
		return "No products loaded. Check your database connection.\n"
	}

	// Left pane: product list
	leftPane := pb.renderProductList()

	// Right pane: product info
	rightPane := pb.renderInfoPane()

	// Help footer
	helpText := pb.renderHelp()

	// Combine panes
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPane,
		rightPane,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		helpText,
	)
}

func (pb *ProductBrowser) renderProductList() string {
	listWidth := pb.width / 2
	listHeight := pb.height - 3

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
	infoWidth := pb.width / 2
	infoHeight := pb.height - 3

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

	if len(product.Cannabinoids) > 0 {
		info.WriteString("\n")
		info.WriteString(pb.styledLabel("Top Cannabinoids:"))
		info.WriteString("\n")
		for _, c := range product.Cannabinoids {
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

func (pb *ProductBrowser) renderHelp() string {
	help := "↑/k: up  ↓/j: down  b: filter by brand  t: filter by type  f: toggle focus  q: quit"
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(0, 1).
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
	pb.infoPaneView.Height = pb.height - 3
	pb.infoPaneView.Width = pb.width / 2
}

func (pb *ProductBrowser) updateDimensions(width, height int) {
	pb.width = width
	pb.height = height
	pb.infoPaneView = viewport.New(width/2, height-3)
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
