// Package models defines data structures for the dank-bubbler application
package models

// Product represents a cannabis product from the brands database
type Product struct {
	ID           string
	Name         string
	Brand        string
	Type         string // cannabis type: flower, edible, concentrate, etc.
	Date         string
	Cannabinoids []Cannabinoid
	Description  string
	Price        float64
	THC          float64
	CBD          float64
	Terpenes     []string
}

// Cannabinoid represents a cannabinoid with its percentage
type Cannabinoid struct {
	Name       string
	Percentage float64
}

// Brand represents a cannabis brand
type Brand struct {
	ID           string
	Name         string
	Description  string
	ProductCount int
}
