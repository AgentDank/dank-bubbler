package ui

import (
	"testing"

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

func keyPress(text string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: text, Code: []rune(text)[0]})
}

func keyEnter() tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
}
