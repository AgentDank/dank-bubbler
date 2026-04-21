// Package models defines data structures for the dank-bubbler application
package models

import "time"

// Product represents a cannabis product from the CT brands dataset.
type Product struct {
	ID                  string     // registration_number
	BrandName           string     // brand_name
	DosageForm          string     // dosage_form
	BrandingEntity      string     // branding_entity
	RegistrationNumber  string     // registration_number (unique ID)
	ApprovalDate        time.Time  // approval_date
	THC                 float64    // tetrahydrocannabinol_thc
	THCA                float64    // tetrahydrocannabinol_acid_thca
	CBD                 float64    // cannabidiols_cbd
	CBDA                float64    // cannabidiol_acid_cbda
	OtherCannabinoids   []Compound // Cannabinoids beyond THC/CBD/THCA/CBDA
	Compounds           []Compound // Terpenes (derived)
	Market              string     // market (e.g., "CT")
	Chemotype           string     // chemotype
	ProcessingTechnique string     // processing_technique
}

// Compound represents a cannabinoid or terpene with its percentage.
type Compound struct {
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

// TaxRecord is a monthly cannabis tax-revenue row from ct_tax.
type TaxRecord struct {
	PeriodEnd         time.Time
	PlantMaterialTax  float64
	EdibleProductsTax float64
	OtherCannabisTax  float64
	TotalTax          float64
}

// SalesRecord is a weekly retail sales row from ct_weekly_sales.
type SalesRecord struct {
	WeekEnding           time.Time
	AdultUse             float64
	Medical              float64
	Total                float64
	AdultUseProductsSold int
	MedicalProductsSold  int
	TotalProductsSold    int
	AdultUseAvgPrice     float64
	MedicalAvgPrice      float64
}

// ZoningRow is one row from ct_zoning. Empty Status represents a SQL NULL
// (rendered as "Unknown" in the UI).
type ZoningRow struct {
	Town   string
	Status string
}

// RetailLocation is one row from ct_retail_locations.
type RetailLocation struct {
	Type      string
	Business  string
	DBA       string
	License   string
	Street    string
	City      string
	Zipcode   string
	Website   string
	Longitude float64
	Latitude  float64
}
