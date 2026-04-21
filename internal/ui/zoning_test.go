package ui

import (
	"reflect"
	"testing"

	"github.com/AgentDank/dank-bubbler/internal/models"
)

func TestRecomputeZoningFilter(t *testing.T) {
	all := []models.ZoningRow{
		{Town: "Ansonia", Status: "Approved"},
		{Town: "Avon", Status: "Prohibited"},
		{Town: "Bethany", Status: "Moratorium"},
		{Town: "Andover", Status: ""}, // NULL -> Unknown
		{Town: "Bristol", Status: "Approved"},
	}

	tests := []struct {
		name   string
		filter zoningStatusFilter
		sort   zoningSortKey
		want   []string // expected town order
	}{
		{"all, sort by town", zoningFilterAll, zoningSortTown,
			[]string{"Andover", "Ansonia", "Avon", "Bethany", "Bristol"}},
		{"approved only", zoningFilterApproved, zoningSortTown,
			[]string{"Ansonia", "Bristol"}},
		{"prohibited only", zoningFilterProhibited, zoningSortTown,
			[]string{"Avon"}},
		{"moratorium only", zoningFilterMoratorium, zoningSortTown,
			[]string{"Bethany"}},
		{"unknown only", zoningFilterUnknown, zoningSortTown,
			[]string{"Andover"}},
		{"sort by status then town", zoningFilterAll, zoningSortStatus,
			[]string{"Ansonia", "Bristol", "Bethany", "Avon", "Andover"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := recomputeZoning(all, tc.filter, tc.sort)
			var gotTowns []string
			for _, r := range got {
				gotTowns = append(gotTowns, r.Town)
			}
			if !reflect.DeepEqual(gotTowns, tc.want) {
				t.Fatalf("got %v, want %v", gotTowns, tc.want)
			}
		})
	}
}
