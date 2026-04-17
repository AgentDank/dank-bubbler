package ui

import (
	"fmt"
	"math"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
	"github.com/charmbracelet/x/ansi"

	"github.com/AgentDank/dank-bubbler/internal/data"
	"github.com/AgentDank/dank-bubbler/internal/models"
)

// TimeRange selects the historical window shown on the Sales & Tax page.
type TimeRange int

const (
	TimeRange3Month TimeRange = iota
	TimeRange1Year
	TimeRange5Year
)

func (r TimeRange) String() string {
	switch r {
	case TimeRange3Month:
		return "3 Months"
	case TimeRange1Year:
		return "1 Year"
	case TimeRange5Year:
		return "5 Years"
	}
	return ""
}

func (r TimeRange) start(end time.Time) time.Time {
	switch r {
	case TimeRange3Month:
		return end.AddDate(0, -3, 0)
	case TimeRange1Year:
		return end.AddDate(-1, 0, 0)
	case TimeRange5Year:
		return end.AddDate(-5, 0, 0)
	}
	return time.Time{}
}

var (
	prevRangeKey = key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "prev range"))
	nextRangeKey = key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "next range"))
	pagesKey     = key.NewBinding(key.WithKeys("1", "2"), key.WithHelp("1/2", "page"))
)

type salesTaxHelpKeyMap struct{}

func (salesTaxHelpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{pagesKey, prevRangeKey, nextRangeKey, quitKey}
}

func (salesTaxHelpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{pagesKey, prevRangeKey, nextRangeKey, quitKey}}
}

// SalesTaxBrowser renders the Sales & Tax page: revenue line chart overlay,
// stacked products-sold bars, and an average-price line chart.
type SalesTaxBrowser struct {
	loader    *data.Loader
	width     int
	height    int
	timeRange TimeRange
	tax       []models.TaxRecord
	sales     []models.SalesRecord
	loadErr   error
	help      help.Model
}

// NewSalesTaxBrowser builds a new page bound to the given loader.
func NewSalesTaxBrowser(loader *data.Loader) *SalesTaxBrowser {
	b := &SalesTaxBrowser{
		loader:    loader,
		timeRange: TimeRange1Year,
	}
	b.help = help.New()
	b.help.ShortSeparator = "  "
	b.help.Styles.ShortKey = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Bold(true)
	b.help.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	b.help.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	b.reload()
	return b
}

func (s *SalesTaxBrowser) Init() tea.Cmd { return nil }

func (s *SalesTaxBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		s.help.SetWidth(msg.Width)
	case tea.KeyMsg:
		switch msg.String() {
		case "left":
			if s.timeRange > TimeRange3Month {
				s.timeRange--
				s.reload()
			}
		case "right":
			if s.timeRange < TimeRange5Year {
				s.timeRange++
				s.reload()
			}
		case "ctrl+c", "q":
			return s, tea.Quit
		}
	}
	return s, nil
}

func (s *SalesTaxBrowser) reload() {
	if s.loader == nil {
		return
	}
	end := time.Now()
	start := s.timeRange.start(end)
	tax, err := s.loader.LoadTaxHistory(start, time.Time{})
	if err != nil {
		s.loadErr = err
		return
	}
	sales, err := s.loader.LoadSalesHistory(start, time.Time{})
	if err != nil {
		s.loadErr = err
		return
	}
	s.tax = tax
	s.sales = sales
	s.loadErr = nil
}

// View renders the entire page.
func (s *SalesTaxBrowser) View() tea.View {
	header := s.renderHeader()
	footer := s.renderHelp()
	middleHeight := max(s.height-3, 0)

	revenueH := middleHeight * 2 / 5
	productsH := middleHeight / 3
	priceH := middleHeight - revenueH - productsH
	if revenueH < 5 || productsH < 5 || priceH < 5 {
		// Fallback for very small windows: split evenly.
		revenueH = middleHeight / 3
		productsH = middleHeight / 3
		priceH = middleHeight - revenueH - productsH
	}

	revenue := s.renderRevenueChart(s.width, revenueH)
	products := s.renderProductsSoldChart(s.width, productsH)
	price := s.renderAvgPriceChart(s.width, priceH)

	content := lipgloss.JoinVertical(lipgloss.Left, revenue, products, price)
	content = lipgloss.NewStyle().Width(s.width).MaxWidth(s.width).Render(content)

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, content, footer))
}

func (s *SalesTaxBrowser) renderHeader() string {
	if s.width <= 0 {
		return ""
	}
	title := ansi.Truncate(appHeader, s.width, "")
	return lipgloss.NewStyle().
		Width(s.width).
		MaxWidth(s.width).
		MaxHeight(1).
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("230")).
		Bold(true).
		Render(lipgloss.PlaceHorizontal(s.width, lipgloss.Right, title))
}

func (s *SalesTaxBrowser) renderHelp() string {
	if s.width <= 0 {
		return ""
	}
	helpText := s.help.View(salesTaxHelpKeyMap{})
	return lipgloss.NewStyle().
		Width(s.width).
		MaxWidth(s.width).
		MaxHeight(1).
		Background(lipgloss.Color("238")).
		Foreground(lipgloss.Color("252")).
		Render(helpText)
}

func (s *SalesTaxBrowser) sectionStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(0, 1)
}

// renderRevenueChart overlays weekly sales total and monthly tax.
func (s *SalesTaxBrowser) renderRevenueChart(outerWidth, outerHeight int) string {
	style := s.sectionStyle()
	innerWidth := max(outerWidth-style.GetHorizontalFrameSize(), 0)
	innerHeight := max(outerHeight-style.GetVerticalFrameSize(), 0)

	if s.loadErr != nil {
		return style.Width(outerWidth).Height(outerHeight).Render("load error: " + s.loadErr.Error())
	}
	if len(s.sales) == 0 && len(s.tax) == 0 {
		return style.Width(outerWidth).Height(outerHeight).Render("No sales/tax data")
	}
	if innerWidth < 20 || innerHeight < 4 {
		return style.Width(outerWidth).Height(outerHeight).Render("window too small")
	}

	title := fmt.Sprintf("Revenue & Tax  —  %s", s.timeRange)
	titleStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true).Render(title)

	// Inside the border we reserve row 0 for the title; the linechart uses the rest.
	chartHeight := max(innerHeight-1, 1)

	minT, maxT := s.timeRangeBounds()
	maxY := 0.0
	for _, r := range s.sales {
		maxY = math.Max(maxY, r.Total)
	}
	for _, r := range s.tax {
		maxY = math.Max(maxY, r.TotalTax)
	}
	if maxY <= 0 {
		maxY = 1
	}

	const taxSet = "tax"
	salesStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	taxStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))    // magenta

	lc := timeserieslinechart.New(innerWidth, chartHeight,
		timeserieslinechart.WithTimeRange(minT, maxT),
		timeserieslinechart.WithYRange(0, maxY),
		timeserieslinechart.WithAxesStyles(
			lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
		),
		timeserieslinechart.WithStyle(salesStyle),
		timeserieslinechart.WithLineStyle(runes.ThinLineStyle),
		timeserieslinechart.WithDataSetStyle(taxSet, taxStyle),
		timeserieslinechart.WithDataSetLineStyle(taxSet, runes.ThinLineStyle),
		timeserieslinechart.WithXLabelFormatter(timeserieslinechart.DateTimeLabelFormatter()),
	)
	for _, r := range s.sales {
		lc.Push(timeserieslinechart.TimePoint{Time: r.WeekEnding, Value: r.Total})
	}
	for _, r := range s.tax {
		lc.PushDataSet(taxSet, timeserieslinechart.TimePoint{Time: r.PeriodEnd, Value: r.TotalTax})
	}
	lc.DrawXYAxisAndLabel()
	lc.DrawAll()

	legend := salesStyle.Render("— weekly sales") + "  " + taxStyle.Render("— monthly tax")
	body := lipgloss.JoinVertical(lipgloss.Left, titleStyled+"  "+legend, lc.View())
	return style.Width(outerWidth).Height(outerHeight).Render(body)
}

// renderProductsSoldChart shows stacked vertical bars (adult+medical) per week.
func (s *SalesTaxBrowser) renderProductsSoldChart(outerWidth, outerHeight int) string {
	style := s.sectionStyle()
	innerWidth := max(outerWidth-style.GetHorizontalFrameSize(), 0)
	innerHeight := max(outerHeight-style.GetVerticalFrameSize(), 0)

	if len(s.sales) == 0 {
		return style.Width(outerWidth).Height(outerHeight).Render("No sales data")
	}
	if innerWidth < 20 || innerHeight < 4 {
		return style.Width(outerWidth).Height(outerHeight).Render("window too small")
	}

	title := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true).Render("Products Sold (stacked)")
	adultStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Background(lipgloss.Color("4")) // blue
	medStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Background(lipgloss.Color("9"))   // red

	// Keep the most recent weeks that fit in the canvas.
	sales := s.sales
	if len(sales) > innerWidth {
		sales = sales[len(sales)-innerWidth:]
	}

	var data []barchart.BarData
	for _, r := range sales {
		data = append(data, barchart.BarData{
			Label: "",
			Values: []barchart.BarValue{
				{Name: "adult", Value: float64(r.AdultUseProductsSold), Style: adultStyle},
				{Name: "medical", Value: float64(r.MedicalProductsSold), Style: medStyle},
			},
		})
	}

	chartH := max(innerHeight-1, 1)
	bc := barchart.New(innerWidth, chartH,
		barchart.WithNoAutoBarWidth(),
		barchart.WithBarWidth(1),
		barchart.WithBarGap(0),
		barchart.WithNoAxis(),
		barchart.WithDataSet(data),
	)
	bc.Draw()

	legend := adultStyle.Render("  ") + " adult-use  " + medStyle.Render("  ") + " medical"
	body := lipgloss.JoinVertical(lipgloss.Left, title+"  "+legend, bc.View())
	return style.Width(outerWidth).Height(outerHeight).Render(body)
}

// renderAvgPriceChart overlays adult-use and medical average prices.
func (s *SalesTaxBrowser) renderAvgPriceChart(outerWidth, outerHeight int) string {
	style := s.sectionStyle()
	innerWidth := max(outerWidth-style.GetHorizontalFrameSize(), 0)
	innerHeight := max(outerHeight-style.GetVerticalFrameSize(), 0)

	if len(s.sales) == 0 {
		return style.Width(outerWidth).Height(outerHeight).Render("No sales data")
	}
	if innerWidth < 20 || innerHeight < 4 {
		return style.Width(outerWidth).Height(outerHeight).Render("window too small")
	}

	title := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true).Render("Average Price")
	chartHeight := max(innerHeight-1, 1)

	minT, maxT := s.timeRangeBounds()
	maxY := 0.0
	for _, r := range s.sales {
		maxY = math.Max(maxY, math.Max(r.AdultUseAvgPrice, r.MedicalAvgPrice))
	}
	if maxY <= 0 {
		maxY = 1
	}

	const medSet = "medical"
	adultStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4")) // blue
	medStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))   // red

	lc := timeserieslinechart.New(innerWidth, chartHeight,
		timeserieslinechart.WithTimeRange(minT, maxT),
		timeserieslinechart.WithYRange(0, maxY),
		timeserieslinechart.WithAxesStyles(
			lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
		),
		timeserieslinechart.WithStyle(adultStyle),
		timeserieslinechart.WithLineStyle(runes.ThinLineStyle),
		timeserieslinechart.WithDataSetStyle(medSet, medStyle),
		timeserieslinechart.WithDataSetLineStyle(medSet, runes.ThinLineStyle),
		timeserieslinechart.WithXLabelFormatter(timeserieslinechart.DateTimeLabelFormatter()),
	)
	for _, r := range s.sales {
		lc.Push(timeserieslinechart.TimePoint{Time: r.WeekEnding, Value: r.AdultUseAvgPrice})
		lc.PushDataSet(medSet, timeserieslinechart.TimePoint{Time: r.WeekEnding, Value: r.MedicalAvgPrice})
	}
	lc.DrawXYAxisAndLabel()
	lc.DrawAll()

	legend := adultStyle.Render("— adult-use") + "  " + medStyle.Render("— medical")
	body := lipgloss.JoinVertical(lipgloss.Left, title+"  "+legend, lc.View())
	return style.Width(outerWidth).Height(outerHeight).Render(body)
}

// timeRangeBounds returns the X-axis bounds that match the selected view.
// Prefers actual data min/max when both tax and sales are populated.
func (s *SalesTaxBrowser) timeRangeBounds() (time.Time, time.Time) {
	end := time.Now()
	start := s.timeRange.start(end)

	// Clamp to actual data if available.
	var dataMin, dataMax time.Time
	update := func(t time.Time) {
		if t.IsZero() {
			return
		}
		if dataMin.IsZero() || t.Before(dataMin) {
			dataMin = t
		}
		if dataMax.IsZero() || t.After(dataMax) {
			dataMax = t
		}
	}
	for _, r := range s.sales {
		update(r.WeekEnding)
	}
	for _, r := range s.tax {
		update(r.PeriodEnd)
	}
	if !dataMin.IsZero() && dataMin.After(start) {
		start = dataMin
	}
	if !dataMax.IsZero() && dataMax.Before(end) {
		end = dataMax
	}
	return start, end
}

