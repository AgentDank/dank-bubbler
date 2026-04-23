package ui

import (
	"reflect"
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

func TestRecomputeRetail(t *testing.T) {
	all := []models.RetailLocation{
		{Business: "ACME", City: "Hartford", Type: "Hybrid Retailer"},
		{Business: "Best", City: "Bristol", Type: "Adult-Use Cannabis Only"},
		{Business: "Carlos", City: "Bristol", Type: "Hybrid Retailer"},
		{Business: "Delta", City: "Ansonia", Type: "Medical Marijuana Only"},
		{Business: "Echo", City: "Ansonia", Type: "Hybrid Retailer"},
	}

	tests := []struct {
		name   string
		filter retailTypeFilter
		sort   retailSortKey
		query  string
		want   []string // expected Business order
	}{
		{"all, business asc", retailFilterAll, retailSortBusinessAsc, "",
			[]string{"ACME", "Best", "Carlos", "Delta", "Echo"}},
		{"all, business desc", retailFilterAll, retailSortBusinessDesc, "",
			[]string{"Echo", "Delta", "Carlos", "Best", "ACME"}},
		{"hybrid only", retailFilterHybrid, retailSortBusinessAsc, "",
			[]string{"ACME", "Carlos", "Echo"}},
		{"adult-use only", retailFilterAdultUseOnly, retailSortBusinessAsc, "",
			[]string{"Best"}},
		{"medical only", retailFilterMedicalOnly, retailSortBusinessAsc, "",
			[]string{"Delta"}},
		{"city asc", retailFilterAll, retailSortCityAsc, "",
			[]string{"Delta", "Echo", "Best", "Carlos", "ACME"}},
		{"city desc", retailFilterAll, retailSortCityDesc, "",
			[]string{"ACME", "Best", "Carlos", "Delta", "Echo"}},
		{"query matches city substring", retailFilterAll, retailSortBusinessAsc, "bristol",
			[]string{"Best", "Carlos"}},
		{"query matches business substring case-insensitive", retailFilterAll, retailSortBusinessAsc, "cme",
			[]string{"ACME"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := recomputeRetail(all, tc.filter, tc.sort, tc.query)
			var gotBiz []string
			for _, r := range got {
				gotBiz = append(gotBiz, r.Business)
			}
			if !reflect.DeepEqual(gotBiz, tc.want) {
				t.Fatalf("got %v, want %v", gotBiz, tc.want)
			}
		})
	}
}
