// Package models defines data structures for the dank-bubbler application
package models

import "time"

// Product represents a cannabis product from the brands_us_ct database
type Product struct {
	ID                  string             // registration_number
	BrandName           string             // brand_name
	DosageForm          string             // dosage_form
	BrandingEntity      string             // branding_entity
	RegistrationNumber  string             // registration_number (unique ID)
	ApprovalDate        time.Time          // approval_date
	THC                 float64            // tetrahydrocannabinol_thc
	THCA                float64            // tetrahydrocannabinol_acid_thca
	CBD                 float64            // cannabidiols_cbd
	CBDA                float64            // cannabidiol_acid_cbda
	Terpenes            map[string]float64 // Terpene compounds
	Cannabinoids        []Cannabinoid      // Top cannabinoids (derived)
	Market              string             // market (e.g., "CT")
	Chemotype           string             // chemotype
	ProcessingTechnique string             // processing_technique
}

// Cannabinoid represents a cannabinoid with its percentage
type Cannabinoid struct {
	Name       string
	Percentage float64
}

// Brand represents a cannabis brand (derived from unique brand_name values)
type Brand struct {
	ID           string
	Name         string
	Description  string
	ProductCount int
}
