package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/AgentDank/dank-bubbler/internal/data"
	"github.com/AgentDank/dank-bubbler/internal/models"
	"github.com/AgentDank/dank-bubbler/mapview"
)

// Page identifies a top-level tab in the application.
type Page int

const (
	PageBrands Page = iota
	PageSalesTax
	PageZoning
	PageRetail
)

// pageTabs drives the header tab strip. The index must match the Page value.
var pageTabs = []struct {
	key   string
	label string
}{
	{"1", "Brands"},
	{"2", "Sales & Tax"},
	{"3", "Zoning"},
	{"4", "Retail"},
}

// AppModel is the top-level tea.Model. It holds one sub-browser per page and
// routes input to the active one, plus handles page-switching hotkeys.
type AppModel struct {
	page       Page
	brands     *ProductBrowser
	salesTax   *SalesTaxBrowser
	zoning     *ZoningBrowser
	retail     *RetailBrowser
	lastResize tea.WindowSizeMsg
}

// NewAppModel wires the pages.
func NewAppModel(products []models.Product, brands []models.Brand, loader *data.Loader) *AppModel {
	a := &AppModel{
		page:     PageBrands,
		brands:   NewProductBrowser(products, brands, loader),
		salesTax: NewSalesTaxBrowser(loader),
		zoning:   NewZoningBrowser(loader),
		retail:   NewRetailBrowser(loader),
	}
	a.syncActivePage()
	return a
}

func (a *AppModel) syncActivePage() {
	a.brands.SetActivePage(a.page)
	a.salesTax.SetActivePage(a.page)
	a.zoning.SetActivePage(a.page)
	a.retail.SetActivePage(a.page)
}

func (a *AppModel) Init() tea.Cmd {
	return tea.Batch(a.brands.Init(), a.salesTax.Init(), a.zoning.Init(), a.retail.Init())
}

func (a *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.lastResize = msg
		// Forward size changes to all pages so the inactive pages are ready
		// when the user switches.
		_, cmdA := a.brands.Update(msg)
		_, cmdB := a.salesTax.Update(msg)
		_, cmdC := a.zoning.Update(msg)
		_, cmdD := a.retail.Update(msg)
		return a, tea.Batch(cmdA, cmdB, cmdC, cmdD)

	case tea.KeyMsg:
		switch msg.String() {
		case "1":
			a.page = PageBrands
			a.syncActivePage()
			return a, nil
		case "2":
			a.page = PageSalesTax
			a.syncActivePage()
			return a, nil
		case "3":
			a.page = PageZoning
			a.syncActivePage()
			return a, nil
		case "4":
			a.page = PageRetail
			a.syncActivePage()
			return a, nil
		}
	}

	// Mapview-originated messages (async render results) must reach the retail
	// page regardless of which page is currently active — otherwise the result
	// of the initial WindowSize-triggered render goes to the wrong page and
	// the map stays blank until the user nudges retail into rendering again.
	if mapview.IsMapUpdate(msg) {
		_, cmd := a.retail.Update(msg)
		return a, cmd
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
	case PageZoning:
		_, cmd = a.zoning.Update(msg)
	case PageRetail:
		_, cmd = a.retail.Update(msg)
	}
	return a, cmd
}

func (a *AppModel) View() tea.View {
	switch a.page {
	case PageSalesTax:
		return a.salesTax.View()
	case PageZoning:
		return a.zoning.View()
	case PageRetail:
		return a.retail.View()
	default:
		return a.brands.View()
	}
}
