// Package data handles data loading from various sources
package data

import (
	"fmt"

	"github.com/AgentDank/dank-bubbler/internal/models"
)

// Loader handles loading cannabis product data from a DuckDB source
type Loader struct {
	dbPath string
}

// NewLoader creates a new data loader for the given DuckDB path
func NewLoader(dbPath string) *Loader {
	return &Loader{
		dbPath: dbPath,
	}
}

// LoadBrands loads all brands from the database
func (l *Loader) LoadBrands() ([]models.Brand, error) {
	// TODO: Implement DuckDB loading
	return nil, fmt.Errorf("not yet implemented")
}

// LoadProducts loads all products from the database
func (l *Loader) LoadProducts() ([]models.Product, error) {
	// TODO: Implement DuckDB loading
	return nil, fmt.Errorf("not yet implemented")
}

// LoadProductsByBrand loads products filtered by brand
func (l *Loader) LoadProductsByBrand(brand string) ([]models.Product, error) {
	// TODO: Implement DuckDB loading
	return nil, fmt.Errorf("not yet implemented")
}

// LoadProductsByType loads products filtered by cannabis type
func (l *Loader) LoadProductsByType(productType string) ([]models.Product, error) {
	// TODO: Implement DuckDB loading
	return nil, fmt.Errorf("not yet implemented")
}
