package ui

import (
	"strings"
	"testing"

	"github.com/AgentDank/dank-bubbler/internal/models"
)

func TestRetailTypeBadge(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Hybrid Retailer", "HYB"},
		{"Adult-Use Cannabis Only", "AU"},
		{"Medical Marijuana Only", "MED"},
		{"", "?"},
		{"Unknown Whatever", "?"},
	}
	for _, tc := range tests {
		got := retailTypeBadge(tc.in)
		if got != tc.want {
			t.Errorf("retailTypeBadge(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatRetailDetailBar(t *testing.T) {
	loc := models.RetailLocation{
		Type:      "Hybrid Retailer",
		Business:  "ACME CANNABIS LLC",
		DBA:       "ACME DISPENSARY",
		License:   "ABC12345",
		Street:    "1 MAIN ST",
		City:      "HARTFORD",
		Zipcode:   "06103",
		Website:   "https://example.com",
		Longitude: -72.68,
		Latitude:  41.76,
	}

	line1, line2 := formatRetailDetailBar(loc)
	for _, want := range []string{"ACME CANNABIS LLC", "ACME DISPENSARY", "Hybrid Retailer", "ABC12345"} {
		if !strings.Contains(line1, want) {
			t.Errorf("line1 %q missing %q", line1, want)
		}
	}
	for _, want := range []string{"1 MAIN ST", "HARTFORD", "06103", "https://example.com", "41.760", "-72.680"} {
		if !strings.Contains(line2, want) {
			t.Errorf("line2 %q missing %q", line2, want)
		}
	}
}

func TestFormatRetailDetailBarOmitsEmptyFields(t *testing.T) {
	loc := models.RetailLocation{
		Type:     "Hybrid Retailer",
		Business: "ACME",
		License:  "ABC123",
		// DBA, street, city, zipcode, website all empty
	}
	line1, line2 := formatRetailDetailBar(loc)
	if strings.Contains(line1, "·") && !strings.Contains(line1, "ACME") {
		t.Errorf("line1 should contain ACME, got %q", line1)
	}
	// line2 should not be just separators
	if strings.Count(line2, "—") > 1 {
		t.Errorf("line2 should collapse empty fields, got %q", line2)
	}
}
