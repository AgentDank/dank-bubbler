// Package data handles data loading from various sources
package data

import (
	"database/sql"
	"fmt"

	_ "github.com/marcboeker/go-duckdb"

	"github.com/AgentDank/dank-bubbler/internal/models"
)

const brandsTableName = "ct_brands"

// Loader handles loading cannabis product data from a DuckDB source
type Loader struct {
	dbPath string
	db     *sql.DB
}

// NewLoader creates a new data loader for the given DuckDB path
func NewLoader(dbPath string) *Loader {
	return &Loader{
		dbPath: dbPath,
	}
}

// Open opens the database connection
func (l *Loader) Open() error {
	db, err := sql.Open("duckdb", l.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	l.db = db
	return nil
}

// Close closes the database connection
func (l *Loader) Close() error {
	if l.db != nil {
		return l.db.Close()
	}
	return nil
}

// HasBrandsTable checks if the CT brands table exists in the database.
func (l *Loader) HasBrandsTable() (bool, error) {
	if l.db == nil {
		return false, fmt.Errorf("database not open")
	}

	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = ?
		)
	`
	err := l.db.QueryRow(query, brandsTableName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check for %s table: %w", brandsTableName, err)
	}
	return exists, nil
}

// LoadBrands loads all brands from the database
func (l *Loader) LoadBrands() ([]models.Brand, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	rows, err := l.db.Query(`
		SELECT DISTINCT
			COALESCE(brand_name, 'Unknown') as name,
			COUNT(*) as count
		FROM ct_brands
		GROUP BY brand_name
		ORDER BY brand_name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query brands: %w", err)
	}
	defer rows.Close()

	var brands []models.Brand
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			continue
		}

		brand := models.Brand{
			ID:           name,
			Name:         name,
			Description:  "",
			ProductCount: count,
		}
		brands = append(brands, brand)
	}

	return brands, rows.Err()
}

// LoadProducts loads all products from the database
func (l *Loader) LoadProducts() ([]models.Product, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	rows, err := l.db.Query(`
		SELECT
			registration_number,
			COALESCE(brand_name, 'Unknown') as brand_name,
			COALESCE(dosage_form, 'Unknown') as dosage_form,
			COALESCE(branding_entity, '') as branding_entity,
			approval_date,
			COALESCE(tetrahydrocannabinol_thc, 0) as thc,
			COALESCE(cannabidiols_cbd, 0) as cbd
		FROM ct_brands
		ORDER BY brand_name, registration_number
		LIMIT 1000
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var product models.Product
		if err := rows.Scan(
			&product.RegistrationNumber,
			&product.BrandName,
			&product.DosageForm,
			&product.BrandingEntity,
			&product.ApprovalDate,
			&product.THC,
			&product.CBD,
		); err != nil {
			continue
		}

		product.ID = product.RegistrationNumber
		product.Compounds = []models.Compound{}
		products = append(products, product)
	}

	return products, rows.Err()
}

// LoadProductsByBrand loads products filtered by brand
func (l *Loader) LoadProductsByBrand(brand string) ([]models.Product, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	rows, err := l.db.Query(`
		SELECT
			registration_number,
			COALESCE(brand_name, 'Unknown') as brand_name,
			COALESCE(dosage_form, 'Unknown') as dosage_form,
			COALESCE(branding_entity, '') as branding_entity,
			approval_date,
			COALESCE(tetrahydrocannabinol_thc, 0) as thc,
			COALESCE(cannabidiols_cbd, 0) as cbd
		FROM ct_brands
		WHERE LOWER(brand_name) = LOWER(?)
		ORDER BY registration_number
	`, brand)
	if err != nil {
		return nil, fmt.Errorf("failed to query products by brand: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var product models.Product
		if err := rows.Scan(
			&product.RegistrationNumber,
			&product.BrandName,
			&product.DosageForm,
			&product.BrandingEntity,
			&product.ApprovalDate,
			&product.THC,
			&product.CBD,
		); err != nil {
			continue
		}

		product.ID = product.RegistrationNumber
		product.Compounds = []models.Compound{}
		products = append(products, product)
	}

	return products, rows.Err()
}

// LoadProductsByType loads products filtered by dosage_form (type)
func (l *Loader) LoadProductsByType(dosageForm string) ([]models.Product, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	rows, err := l.db.Query(`
		SELECT
			registration_number,
			COALESCE(brand_name, 'Unknown') as brand_name,
			COALESCE(dosage_form, 'Unknown') as dosage_form,
			COALESCE(branding_entity, '') as branding_entity,
			approval_date,
			COALESCE(tetrahydrocannabinol_thc, 0) as thc,
			COALESCE(cannabidiols_cbd, 0) as cbd
		FROM ct_brands
		WHERE LOWER(dosage_form) = LOWER(?)
		ORDER BY brand_name, registration_number
	`, dosageForm)
	if err != nil {
		return nil, fmt.Errorf("failed to query products by dosage form: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var product models.Product
		if err := rows.Scan(
			&product.RegistrationNumber,
			&product.BrandName,
			&product.DosageForm,
			&product.BrandingEntity,
			&product.ApprovalDate,
			&product.THC,
			&product.CBD,
		); err != nil {
			continue
		}

		product.ID = product.RegistrationNumber
		product.Compounds = []models.Compound{}
		products = append(products, product)
	}

	return products, rows.Err()
}

// GetDistinctBrands returns a list of unique brands
func (l *Loader) GetDistinctBrands() ([]string, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	rows, err := l.db.Query(`
		SELECT DISTINCT COALESCE(brand_name, 'Unknown')
		FROM ct_brands
		ORDER BY brand_name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query distinct brands: %w", err)
	}
	defer rows.Close()

	var brands []string
	for rows.Next() {
		var brand string
		if err := rows.Scan(&brand); err != nil {
			continue
		}
		brands = append(brands, brand)
	}

	return brands, rows.Err()
}

// GetDistinctTypes returns a list of unique dosage forms
func (l *Loader) GetDistinctTypes() ([]string, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	rows, err := l.db.Query(`
		SELECT DISTINCT COALESCE(dosage_form, 'Unknown')
		FROM ct_brands
		ORDER BY dosage_form
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query distinct dosage forms: %w", err)
	}
	defer rows.Close()

	var types []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			continue
		}
		types = append(types, t)
	}

	return types, rows.Err()
}

// LoadProductWithCompounds loads a product and its compound data.
func (l *Loader) LoadProductWithCompounds(registrationNumber string) (*models.Product, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	var product models.Product
	err := l.db.QueryRow(`
		SELECT
			registration_number,
			COALESCE(brand_name, 'Unknown') as brand_name,
			COALESCE(dosage_form, 'Unknown') as dosage_form,
			COALESCE(branding_entity, '') as branding_entity,
			approval_date,
			COALESCE(tetrahydrocannabinol_thc, 0) as thc,
			COALESCE(tetrahydrocannabinol_acid_thca, 0) as thca,
			COALESCE(cannabidiols_cbd, 0) as cbd,
			COALESCE(cannabidiol_acid_cbda, 0) as cbda,
			COALESCE(market, '') as market,
			COALESCE(chemotype, '') as chemotype
		FROM ct_brands
		WHERE registration_number = ?
	`, registrationNumber).Scan(
		&product.RegistrationNumber,
		&product.BrandName,
		&product.DosageForm,
		&product.BrandingEntity,
		&product.ApprovalDate,
		&product.THC,
		&product.THCA,
		&product.CBD,
		&product.CBDA,
		&product.Market,
		&product.Chemotype,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query product: %w", err)
	}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("product not found")
	}

	product.ID = product.RegistrationNumber
	product.Compounds = []models.Compound{}

	// Load additional terpene compounds for the detail pane and chart.
	cannaRow := l.db.QueryRow(`
		SELECT
			COALESCE(a_pinene, 0) as a_pinene,
			COALESCE(b_myrcene, 0) as b_myrcene,
			COALESCE(b_caryophyllene, 0) as b_caryophyllene,
			COALESCE(limonene, 0) as limonene,
			COALESCE(linalool_lin, 0) as linalool,
			COALESCE(humulene_hum, 0) as humulene,
			COALESCE(ocimene, 0) as ocimene,
			COALESCE(terpinolene, 0) as terpinolene
		FROM ct_brands
		WHERE registration_number = ?
	`, registrationNumber)

	var aPinene, bMyrcene, bCaryophyllene, limonene, linalool, humulene, ocimene, terpinolene float64
	if err := cannaRow.Scan(&aPinene, &bMyrcene, &bCaryophyllene, &limonene, &linalool, &humulene, &ocimene, &terpinolene); err == nil {
		if aPinene > 0 {
			product.Compounds = append(product.Compounds, models.Compound{"α-Pinene", aPinene})
		}
		if bMyrcene > 0 {
			product.Compounds = append(product.Compounds, models.Compound{"β-Myrcene", bMyrcene})
		}
		if bCaryophyllene > 0 {
			product.Compounds = append(product.Compounds, models.Compound{"β-Caryophyllene", bCaryophyllene})
		}
		if limonene > 0 {
			product.Compounds = append(product.Compounds, models.Compound{"Limonene", limonene})
		}
		if linalool > 0 {
			product.Compounds = append(product.Compounds, models.Compound{"Linalool", linalool})
		}
		if humulene > 0 {
			product.Compounds = append(product.Compounds, models.Compound{"Humulene", humulene})
		}
		if ocimene > 0 {
			product.Compounds = append(product.Compounds, models.Compound{"Ocimene", ocimene})
		}
		if terpinolene > 0 {
			product.Compounds = append(product.Compounds, models.Compound{"Terpinolene", terpinolene})
		}
	}

	return &product, nil
}
