package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/AgentDank/dank-bubbler/internal/models"
)

func TestBrandFilterApplyAndClear(t *testing.T) {
	products := []models.Product{
		{BrandName: "Alpha", DosageForm: "Flower", RegistrationNumber: "1"},
		{BrandName: "Beta", DosageForm: "Vape", RegistrationNumber: "2"},
		{BrandName: "Alpha", DosageForm: "Edible", RegistrationNumber: "3"},
	}

	pb := NewProductBrowser(products, nil, nil)

	pb.Update(keyPress("b"))

	if pb.filterMode != FilterModeByBrand {
		t.Fatalf("expected brand filter mode, got %v", pb.filterMode)
	}

	if len(pb.filterOptions) != 2 {
		t.Fatalf("expected 2 brand options, got %d", len(pb.filterOptions))
	}

	pb.Update(keyPress("j"))
	pb.Update(keyEnter())

	if pb.filterMode != FilterModeNone {
		t.Fatalf("expected filter picker to close after apply, got %v", pb.filterMode)
	}

	if pb.activeFilter != "brand: Beta" {
		t.Fatalf("expected active filter to be set, got %q", pb.activeFilter)
	}

	if len(pb.products) != 1 {
		t.Fatalf("expected filtered product count 1, got %d", len(pb.products))
	}

	if pb.products[0].BrandName != "Beta" {
		t.Fatalf("expected Beta to remain after filtering, got %q", pb.products[0].BrandName)
	}

	pb.Update(keyPress("c"))

	if pb.activeFilter != "" {
		t.Fatalf("expected active filter to clear, got %q", pb.activeFilter)
	}

	if len(pb.products) != len(products) {
		t.Fatalf("expected full product list after clear, got %d", len(pb.products))
	}
}

func TestDateSortToggle(t *testing.T) {
	products := []models.Product{
		{BrandName: "Alpha", DosageForm: "Flower", RegistrationNumber: "1", ApprovalDate: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)},
		{BrandName: "Beta", DosageForm: "Vape", RegistrationNumber: "2", ApprovalDate: time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)},
		{BrandName: "Gamma", DosageForm: "Edible", RegistrationNumber: "3", ApprovalDate: time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)},
	}

	pb := NewProductBrowser(products, nil, nil)

	if pb.dateSort != ProductDateSortNewest {
		t.Fatalf("expected initial newest-first date sort, got %v", pb.dateSort)
	}
	if got := pb.products[0].BrandName; got != "Gamma" {
		t.Fatalf("expected initial newest product first, got %q", got)
	}
	if got := pb.products[pb.selectedIdx].BrandName; got != "Gamma" {
		t.Fatalf("expected initial selected product to be newest row, got %q", got)
	}

	pb.Update(keyPress("d"))

	if pb.filterMode != FilterModeNone {
		t.Fatalf("expected date key to keep normal table mode, got %v", pb.filterMode)
	}
	if pb.dateSort != ProductDateSortOldest {
		t.Fatalf("expected oldest-first date sort, got %v", pb.dateSort)
	}
	if got := pb.products[0].BrandName; got != "Beta" {
		t.Fatalf("expected oldest product first, got %q", got)
	}
	if got := pb.products[pb.selectedIdx].BrandName; got != "Gamma" {
		t.Fatalf("expected selected product to stay Gamma after sort, got %q", got)
	}

	pb.Update(keyPress("d"))

	if pb.dateSort != ProductDateSortNewest {
		t.Fatalf("expected newest-first date sort after second toggle, got %v", pb.dateSort)
	}
	if got := pb.products[0].BrandName; got != "Gamma" {
		t.Fatalf("expected newest product first after second toggle, got %q", got)
	}
	if got := pb.products[pb.selectedIdx].BrandName; got != "Gamma" {
		t.Fatalf("expected selected product to stay Gamma after second sort, got %q", got)
	}
}

func TestTypeDisplayAndFilterUseChemotype(t *testing.T) {
	products := []models.Product{
		{
			BrandName:           "Alpha",
			Chemotype:           "Hybrid",
			DosageForm:          "Flower",
			BrandingEntity:      "Alpha Producer",
			RegistrationNumber:  "1",
			ApprovalDate:        time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC),
			Market:              "CT",
			ProcessingTechnique: "Solventless",
			SolventsUsed:        "None",
			NationalDrugCode:    "12345-6789",
		},
		{BrandName: "Beta", Chemotype: "Indica", DosageForm: "Vape", RegistrationNumber: "2"},
		{BrandName: "Gamma", Chemotype: "Hybrid", DosageForm: "Edible", RegistrationNumber: "3"},
	}

	pb := NewProductBrowser(products, nil, nil)

	columns := pb.tbl.Columns()
	if len(columns) != 3 {
		t.Fatalf("expected 3 product table columns, got %d", len(columns))
	}
	if columns[2].Title != "type" {
		t.Fatalf("expected third column title type, got %q", columns[2].Title)
	}

	rows := pb.tbl.Rows()
	if rows[0][2] != "Hybrid" {
		t.Fatalf("expected table type to use chemotype, got %q", rows[0][2])
	}

	infoText := ansi.Strip(pb.renderInfoPane(80, 24))
	for _, want := range []string{
		"brand_name: Alpha",
		"dosage_form: Flower",
		"branding_entity: Alpha Producer",
		"approval_date: 2026-04-10",
		"registration_number: 1",
		"market: CT",
		"chemotype: Hybrid",
		"processing_technique: Solventless",
		"solvents_used: None",
		"national_drug_code: 12345-6789",
	} {
		if !strings.Contains(infoText, want) {
			t.Fatalf("expected details to include %q, got:\n%s", want, infoText)
		}
	}

	pb.Update(keyPress("t"))
	if pb.filterMode != FilterModeByType {
		t.Fatalf("expected type filter mode, got %v", pb.filterMode)
	}
	if pb.filterOptions[0] != "Hybrid" {
		t.Fatalf("expected first type option Hybrid, got %q", pb.filterOptions[0])
	}

	pb.Update(keyEnter())
	if pb.activeFilter != "type: Hybrid" {
		t.Fatalf("expected active type filter to use chemotype, got %q", pb.activeFilter)
	}
	if len(pb.products) != 2 {
		t.Fatalf("expected 2 hybrid products, got %d", len(pb.products))
	}
}

func TestFormFilterUsesDosageForm(t *testing.T) {
	products := []models.Product{
		{BrandName: "Alpha", Chemotype: "Hybrid", DosageForm: "Flower", RegistrationNumber: "1", ApprovalDate: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)},
		{BrandName: "Beta", Chemotype: "Hybrid", DosageForm: "Vape", RegistrationNumber: "2", ApprovalDate: time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)},
		{BrandName: "Gamma", Chemotype: "Indica", DosageForm: "Flower", RegistrationNumber: "3", ApprovalDate: time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)},
	}

	pb := NewProductBrowser(products, nil, nil)

	pb.Update(keyPress("f"))
	if pb.filterMode != FilterModeByForm {
		t.Fatalf("expected form filter mode, got %v", pb.filterMode)
	}
	if len(pb.filterOptions) != 2 {
		t.Fatalf("expected 2 form options, got %d", len(pb.filterOptions))
	}
	if pb.filterOptions[0] != "Flower" {
		t.Fatalf("expected first form option Flower, got %q", pb.filterOptions[0])
	}

	pb.Update(keyEnter())
	if pb.activeFilter != "form: Flower" {
		t.Fatalf("expected active form filter, got %q", pb.activeFilter)
	}
	if len(pb.products) != 2 {
		t.Fatalf("expected 2 flower products, got %d", len(pb.products))
	}
	if got := pb.products[0].BrandName; got != "Gamma" {
		t.Fatalf("expected filtered products to keep date-desc sort, got first product %q", got)
	}
}

func keyPress(text string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: text, Code: []rune(text)[0]})
}

func keyEnter() tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
}
