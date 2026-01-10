// Package ui provides BubbleTea UI components for the dank-bubbler application
package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgentDank/dank-bubbler/internal/models"
)

// ProductBrowser is a BubbleTea component for browsing cannabis products
type ProductBrowser struct {
	products []models.Product
	brands   []models.Brand
	selected int
	width    int
	height   int
}

// NewProductBrowser creates a new product browser component
func NewProductBrowser(products []models.Product, brands []models.Brand) *ProductBrowser {
	return &ProductBrowser{
		products: products,
		brands:   brands,
		selected: 0,
	}
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
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if pb.selected > 0 {
				pb.selected--
			}
		case "down":
			if pb.selected < len(pb.products)-1 {
				pb.selected++
			}
		}
	}
	return pb, nil
}

// View renders the product browser
func (pb *ProductBrowser) View() string {
	if len(pb.products) == 0 {
		return "No products loaded"
	}

	// TODO: Implement product list view with NTCharts integration
	return "Product Browser - Not Yet Implemented\n"
}
