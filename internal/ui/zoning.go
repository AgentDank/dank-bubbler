package ui

import (
	"sort"

	"github.com/AgentDank/dank-bubbler/internal/models"
)

// zoningStatusFilter selects which rows the zoning page shows.
type zoningStatusFilter int

const (
	zoningFilterAll zoningStatusFilter = iota
	zoningFilterApproved
	zoningFilterProhibited
	zoningFilterMoratorium
	zoningFilterUnknown
)

// zoningSortKey selects the table's row order.
type zoningSortKey int

const (
	zoningSortTown zoningSortKey = iota
	zoningSortStatus
)

// recomputeZoning filters then sorts rows for the zoning table. The status
// filter matches on the raw string ("" is the Unknown bucket). The sort is
// stable with Town as the tiebreaker when sorting by Status.
func recomputeZoning(all []models.ZoningRow, filter zoningStatusFilter, key zoningSortKey) []models.ZoningRow {
	out := make([]models.ZoningRow, 0, len(all))
	for _, r := range all {
		if !zoningRowMatches(r, filter) {
			continue
		}
		out = append(out, r)
	}

	sort.SliceStable(out, func(i, j int) bool {
		switch key {
		case zoningSortStatus:
			si, sj := zoningStatusRank(out[i].Status), zoningStatusRank(out[j].Status)
			if si != sj {
				return si < sj
			}
			return out[i].Town < out[j].Town
		default:
			return out[i].Town < out[j].Town
		}
	})
	return out
}

// zoningStatusRank returns a sort rank for the status string so that the
// display order is Approved < Moratorium < Prohibited < Unknown ("").
func zoningStatusRank(status string) int {
	switch status {
	case "Approved":
		return 0
	case "Moratorium":
		return 1
	case "Prohibited":
		return 2
	default: // "" (Unknown) and anything else sorts last
		return 3
	}
}

func zoningRowMatches(r models.ZoningRow, filter zoningStatusFilter) bool {
	switch filter {
	case zoningFilterApproved:
		return r.Status == "Approved"
	case zoningFilterProhibited:
		return r.Status == "Prohibited"
	case zoningFilterMoratorium:
		return r.Status == "Moratorium"
	case zoningFilterUnknown:
		return r.Status == ""
	default:
		return true
	}
}
