package ui

import (
	"reflect"
	"testing"

	"github.com/AgentDank/dank-bubbler/internal/models"
)

func TestZoningColumnRows(t *testing.T) {
	all := []models.ZoningRow{
		{Town: "Ansonia", Status: "Approved"},
		{Town: "Avon", Status: "Prohibited"},
		{Town: "Bethany", Status: "Moratorium"},
		{Town: "Andover", Status: ""},
		{Town: "Bristol", Status: "Approved"},
		{Town: "Barkhamsted", Status: ""},
	}

	tests := []struct {
		name  string
		order zoningSortOrder
		query string
		want  [zoningColumnCount][]string
	}{
		{
			name:  "asc",
			order: zoningSortAsc,
			want: [zoningColumnCount][]string{
				{"Ansonia", "Bristol"},     // Approved
				{"Avon"},                   // Prohibited
				{"Bethany"},                // Moratorium
				{"Andover", "Barkhamsted"}, // Unknown
			},
		},
		{
			name:  "desc",
			order: zoningSortDesc,
			want: [zoningColumnCount][]string{
				{"Bristol", "Ansonia"},
				{"Avon"},
				{"Bethany"},
				{"Barkhamsted", "Andover"},
			},
		},
		{
			name:  "filter matches prefix case-insensitive",
			order: zoningSortAsc,
			query: "ANS",
			want: [zoningColumnCount][]string{
				{"Ansonia"}, // Approved
				nil,         // Prohibited
				nil,         // Moratorium
				nil,         // Unknown
			},
		},
		{
			name:  "filter matches substring",
			order: zoningSortAsc,
			query: "st",
			want: [zoningColumnCount][]string{
				{"Bristol"},     // Approved (Ansonia excluded)
				nil,             // Prohibited
				nil,             // Moratorium
				{"Barkhamsted"}, // Unknown
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := zoningColumnRows(all, tc.order, tc.query)
			var gotTowns [zoningColumnCount][]string
			for i, col := range got {
				for _, r := range col {
					gotTowns[i] = append(gotTowns[i], r.Town)
				}
			}
			if !reflect.DeepEqual(gotTowns, tc.want) {
				t.Fatalf("got %v, want %v", gotTowns, tc.want)
			}
		})
	}
}

func TestZoningColumnIndex(t *testing.T) {
	cases := map[string]int{
		"Approved":   0,
		"Prohibited": 1,
		"Moratorium": 2,
		"":           3,
		"Bogus":      3, // unrecognized falls through to Unknown
	}
	for in, want := range cases {
		if got := zoningColumnIndex(in); got != want {
			t.Errorf("zoningColumnIndex(%q) = %d, want %d", in, got, want)
		}
	}
}
