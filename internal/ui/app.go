package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/AgentDank/dank-bubbler/internal/data"
	"github.com/AgentDank/dank-bubbler/internal/models"
)

// Page identifies a top-level tab in the application.
type Page int

const (
	PageBrands Page = iota
	PageSalesTax
)

// AppModel is the top-level tea.Model. It holds one sub-browser per page and
// routes input to the active one, plus handles page-switching hotkeys.
type AppModel struct {
	page       Page
	brands     *ProductBrowser
	salesTax   *SalesTaxBrowser
	lastResize tea.WindowSizeMsg
}

// NewAppModel wires the two pages.
func NewAppModel(products []models.Product, brands []models.Brand, loader *data.Loader) *AppModel {
	return &AppModel{
		page:     PageBrands,
		brands:   NewProductBrowser(products, brands, loader),
		salesTax: NewSalesTaxBrowser(loader),
	}
}

func (a *AppModel) Init() tea.Cmd {
	return tea.Batch(a.brands.Init(), a.salesTax.Init())
}

func (a *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.lastResize = msg
		// Forward size changes to both pages so the inactive page is ready
		// when the user switches.
		_, cmdA := a.brands.Update(msg)
		_, cmdB := a.salesTax.Update(msg)
		return a, tea.Batch(cmdA, cmdB)

	case tea.KeyMsg:
		switch msg.String() {
		case "1":
			a.page = PageBrands
			return a, nil
		case "2":
			a.page = PageSalesTax
			return a, nil
		}
	}

	return a.forwardToActive(msg)
}

func (a *AppModel) forwardToActive(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch a.page {
	case PageBrands:
		_, cmd = a.brands.Update(msg)
	case PageSalesTax:
		_, cmd = a.salesTax.Update(msg)
	}
	return a, cmd
}

func (a *AppModel) View() tea.View {
	switch a.page {
	case PageSalesTax:
		return a.salesTax.View()
	default:
		return a.brands.View()
	}
}
