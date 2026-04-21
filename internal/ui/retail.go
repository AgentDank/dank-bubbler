package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/AgentDank/dank-bubbler/internal/models"
)

// retailTypeFilter selects which retail locations the page shows.
type retailTypeFilter int

const (
	retailFilterAll retailTypeFilter = iota
	retailFilterHybrid
	retailFilterAdultUseOnly
	retailFilterMedicalOnly
)

// retailSortKey selects the list's row order.
type retailSortKey int

const (
	retailSortBusiness retailSortKey = iota
	retailSortCity
)

// recomputeRetail filters then sorts rows for the retail list.
func recomputeRetail(all []models.RetailLocation, filter retailTypeFilter, key retailSortKey) []models.RetailLocation {
	out := make([]models.RetailLocation, 0, len(all))
	for _, r := range all {
		if !retailRowMatches(r, filter) {
			continue
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		switch key {
		case retailSortCity:
			if out[i].City != out[j].City {
				return out[i].City < out[j].City
			}
			return out[i].Business < out[j].Business
		default:
			return out[i].Business < out[j].Business
		}
	})
	return out
}

func retailRowMatches(r models.RetailLocation, filter retailTypeFilter) bool {
	switch filter {
	case retailFilterHybrid:
		return r.Type == "Hybrid Retailer"
	case retailFilterAdultUseOnly:
		return r.Type == "Adult-Use Cannabis Only"
	case retailFilterMedicalOnly:
		return r.Type == "Medical Marijuana Only"
	default:
		return true
	}
}

// retailTypeBadge returns a compact 2-3 char badge for a retail location type.
// Returns "?" for empty or unrecognized types.
func retailTypeBadge(t string) string {
	switch t {
	case "Hybrid Retailer":
		return "HYB"
	case "Adult-Use Cannabis Only":
		return "AU"
	case "Medical Marijuana Only":
		return "MED"
	default:
		return "?"
	}
}

// formatRetailDetailBar returns the two lines of the detail bar for a
// selected retail location. Empty fields (and their leading separators) are
// omitted so rows with missing DBA/website/etc. still read cleanly.
//
//	line 1: BUSINESS · DBA  —  Type: <type>  —  Lic#<license>
//	line 2: street, city zipcode  —  website  —  (lat, lng)
func formatRetailDetailBar(loc models.RetailLocation) (string, string) {
	// Line 1
	var who []string
	if loc.Business != "" {
		who = append(who, loc.Business)
	}
	if loc.DBA != "" && loc.DBA != loc.Business {
		who = append(who, loc.DBA)
	}
	var line1Parts []string
	if len(who) > 0 {
		line1Parts = append(line1Parts, strings.Join(who, " · "))
	}
	if loc.Type != "" {
		line1Parts = append(line1Parts, "Type: "+loc.Type)
	}
	if loc.License != "" {
		line1Parts = append(line1Parts, "Lic#"+loc.License)
	}

	// Line 2
	var addr string
	switch {
	case loc.Street != "" && loc.City != "" && loc.Zipcode != "":
		addr = fmt.Sprintf("%s, %s %s", loc.Street, loc.City, loc.Zipcode)
	case loc.Street != "" && loc.City != "":
		addr = fmt.Sprintf("%s, %s", loc.Street, loc.City)
	case loc.City != "":
		addr = loc.City
	default:
		addr = loc.Street
	}
	var line2Parts []string
	if addr != "" {
		line2Parts = append(line2Parts, addr)
	}
	if loc.Website != "" {
		line2Parts = append(line2Parts, loc.Website)
	}
	if loc.Latitude != 0 || loc.Longitude != 0 {
		line2Parts = append(line2Parts, fmt.Sprintf("(%.3f, %.3f)", loc.Latitude, loc.Longitude))
	}

	return strings.Join(line1Parts, "  —  "), strings.Join(line2Parts, "  —  ")
}
