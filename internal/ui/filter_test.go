package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

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

	pb.filterIdx = 1
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

func TestDateFilterApply(t *testing.T) {
	products := []models.Product{
		{BrandName: "Alpha", DosageForm: "Flower", RegistrationNumber: "1", ApprovalDate: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)},
		{BrandName: "Beta", DosageForm: "Vape", RegistrationNumber: "2", ApprovalDate: time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)},
		{BrandName: "Gamma", DosageForm: "Edible", RegistrationNumber: "3", ApprovalDate: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)},
	}

	pb := NewProductBrowser(products, nil, nil)

	pb.Update(keyPress("d"))

	if pb.filterMode != FilterModeByDate {
		t.Fatalf("expected date filter mode, got %v", pb.filterMode)
	}

	if len(pb.filterOptions) != 2 {
		t.Fatalf("expected 2 date options, got %d", len(pb.filterOptions))
	}

	if pb.filterOptions[0] != "2026-04-10" {
		t.Fatalf("expected newest date first, got %q", pb.filterOptions[0])
	}

	pb.Update(keyEnter())

	if pb.activeFilter != "date: 2026-04-10" {
		t.Fatalf("expected active date filter to be set, got %q", pb.activeFilter)
	}

	if len(pb.products) != 2 {
		t.Fatalf("expected 2 products on selected date, got %d", len(pb.products))
	}
}

func keyPress(text string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: text, Code: []rune(text)[0]})
}

func keyEnter() tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
}
