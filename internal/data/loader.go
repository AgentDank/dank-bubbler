// Package data handles data loading from various sources
package data

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

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
	db, err := sql.Open("duckdb", l.dbPath+"?access_mode=read_only")
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
	defer func() { _ = rows.Close() }()

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

// LoadProducts loads the full browseable product set from the database.
func (l *Loader) LoadProducts() ([]models.Product, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	rows, err := l.db.Query(`
		SELECT
			registration_number,
			COALESCE(brand_name, 'Unknown') as brand_name,
			COALESCE(dosage_form, 'Unknown') as dosage_form,
			COALESCE(NULLIF(chemotype, ''), 'Unknown') as chemotype,
			COALESCE(branding_entity, '') as branding_entity,
			approval_date,
			COALESCE(tetrahydrocannabinol_thc, 0) as thc,
			COALESCE(cannabidiols_cbd, 0) as cbd
		FROM ct_brands
		ORDER BY brand_name, registration_number
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var products []models.Product
	for rows.Next() {
		var product models.Product
		if err := rows.Scan(
			&product.RegistrationNumber,
			&product.BrandName,
			&product.DosageForm,
			&product.Chemotype,
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
			COALESCE(NULLIF(chemotype, ''), 'Unknown') as chemotype,
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
	defer func() { _ = rows.Close() }()

	var products []models.Product
	for rows.Next() {
		var product models.Product
		if err := rows.Scan(
			&product.RegistrationNumber,
			&product.BrandName,
			&product.DosageForm,
			&product.Chemotype,
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

// LoadProductsByType loads products filtered by chemotype (product type).
func (l *Loader) LoadProductsByType(productType string) ([]models.Product, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	rows, err := l.db.Query(`
		SELECT
			registration_number,
			COALESCE(brand_name, 'Unknown') as brand_name,
			COALESCE(dosage_form, 'Unknown') as dosage_form,
			COALESCE(NULLIF(chemotype, ''), 'Unknown') as chemotype,
			COALESCE(branding_entity, '') as branding_entity,
			approval_date,
			COALESCE(tetrahydrocannabinol_thc, 0) as thc,
			COALESCE(cannabidiols_cbd, 0) as cbd
		FROM ct_brands
		WHERE LOWER(COALESCE(NULLIF(chemotype, ''), 'Unknown')) = LOWER(?)
		ORDER BY brand_name, registration_number
	`, productType)
	if err != nil {
		return nil, fmt.Errorf("failed to query products by type: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var products []models.Product
	for rows.Next() {
		var product models.Product
		if err := rows.Scan(
			&product.RegistrationNumber,
			&product.BrandName,
			&product.DosageForm,
			&product.Chemotype,
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

// LoadProductsByForm loads products filtered by dosage_form.
func (l *Loader) LoadProductsByForm(form string) ([]models.Product, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	rows, err := l.db.Query(`
		SELECT
			registration_number,
			COALESCE(brand_name, 'Unknown') as brand_name,
			COALESCE(dosage_form, 'Unknown') as dosage_form,
			COALESCE(NULLIF(chemotype, ''), 'Unknown') as chemotype,
			COALESCE(branding_entity, '') as branding_entity,
			approval_date,
			COALESCE(tetrahydrocannabinol_thc, 0) as thc,
			COALESCE(cannabidiols_cbd, 0) as cbd
		FROM ct_brands
		WHERE LOWER(COALESCE(dosage_form, 'Unknown')) = LOWER(?)
		ORDER BY brand_name, registration_number
	`, form)
	if err != nil {
		return nil, fmt.Errorf("failed to query products by form: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var products []models.Product
	for rows.Next() {
		var product models.Product
		if err := rows.Scan(
			&product.RegistrationNumber,
			&product.BrandName,
			&product.DosageForm,
			&product.Chemotype,
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
	defer func() { _ = rows.Close() }()

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

// GetDistinctTypes returns a list of unique product chemotypes.
func (l *Loader) GetDistinctTypes() ([]string, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	rows, err := l.db.Query(`
		SELECT DISTINCT COALESCE(NULLIF(chemotype, ''), 'Unknown') as product_type
		FROM ct_brands
		ORDER BY product_type
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query distinct product types: %w", err)
	}
	defer func() { _ = rows.Close() }()

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

// GetDistinctForms returns a list of unique dosage forms.
func (l *Loader) GetDistinctForms() ([]string, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	rows, err := l.db.Query(`
		SELECT DISTINCT COALESCE(dosage_form, 'Unknown') as form
		FROM ct_brands
		ORDER BY form
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query distinct forms: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var forms []string
	for rows.Next() {
		var form string
		if err := rows.Scan(&form); err != nil {
			continue
		}
		forms = append(forms, form)
	}

	return forms, rows.Err()
}

// GetDistinctNames returns a list of unique product names.
func (l *Loader) GetDistinctNames() ([]string, error) {
	return l.GetDistinctBrands()
}

// LoadProductsByName loads products filtered by exact product name.
func (l *Loader) LoadProductsByName(name string) ([]models.Product, error) {
	return l.LoadProductsByBrand(name)
}

// LoadProductWithCompounds loads a product and its compound data.
func (l *Loader) LoadProductWithCompounds(registrationNumber string) (*models.Product, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	var (
		regNum, brandName, dosageForm, brandingEntity string
		approvalDate                                   time.Time
		thc, thca, cbd, cbda                         float64
		market, chemotype, processingTechnique         string
		solventsUsed, nationalDrugCode                 string
		productImageURL, labelImageURL, labAnalysisURL string
		// Cannabinoids
		cbc, cbn, cbg, cbga, cbdv, thcv float64
		// Terpenes
		aPinene, bMyrcene, bCaryophyllene, bPinene, limonene, linalool, humulene, ocimene, terpinolene float64
		// Additional terpenes
		aBisabolol, aPhellandrene, aTerpinene, bEudesmol, bTerpinene, fenchone, pulegol, borneol, isopulegol float64
		carene, camphene, camphor, caryophylleneOxide, cedrol, eucalyptol, geraniol, guaiol, geranylAcetate float64
		isoborneol, menthol, lFenchone, nerol, sabinene, terpineol, transBFarnesene, valencene float64
		aCedrene, aFarnesene, bFarnesene, cisNerolidol, fenchol, transNerolidol float64
	)

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
			COALESCE(NULLIF(chemotype, ''), 'Unknown') as chemotype,
			COALESCE(CAST(processing_technique AS VARCHAR), '') as processing_technique,
			COALESCE(CAST(solvents_used AS VARCHAR), '') as solvents_used,
			COALESCE(CAST(national_drug_code AS VARCHAR), '') as national_drug_code,
			COALESCE(product_image_url, '') as product_image_url,
			COALESCE(label_image_url, '') as label_image_url,
			COALESCE(lab_analysis_url, '') as lab_analysis_url,
			COALESCE(cannabichromene_cbc, 0) as cbc,
			COALESCE(cannbinol_cbn, 0) as cbn,
			COALESCE(cbg, 0) as cbg,
			COALESCE(cbg_a, 0) as cbga,
			COALESCE(cannabavarin_cbdv, 0) as cbdv,
			COALESCE(tetrahydrocannabivarin_thcv, 0) as thcv,
			COALESCE(a_pinene, 0) as a_pinene,
			COALESCE(b_myrcene, 0) as b_myrcene,
			COALESCE(b_caryophyllene, 0) as b_caryophyllene,
			COALESCE(b_pinene, 0) as b_pinene,
			COALESCE(limonene, 0) as limonene,
			COALESCE(linalool_lin, 0) as linalool,
			COALESCE(humulene_hum, 0) as humulene,
			COALESCE(ocimene, 0) as ocimene,
			COALESCE(terpinolene, 0) as terpinolene,
			COALESCE(a_bisabolol, 0) as a_bisabolol,
			COALESCE(a_phellandrene, 0) as a_phellandrene,
			COALESCE(a_terpinene, 0) as a_terpinene,
			COALESCE(b_eudesmol, 0) as b_eudesmol,
			COALESCE(b_terpinene, 0) as b_terpinene,
			COALESCE(fenchone, 0) as fenchone,
			COALESCE(pulegol, 0) as pulegol,
			COALESCE(borneol, 0) as borneol,
			COALESCE(isopulegol, 0) as isopulegol,
			COALESCE(carene, 0) as carene,
			COALESCE(camphene, 0) as camphene,
			COALESCE(camphor, 0) as camphor,
			COALESCE(caryophyllene_oxide, 0) as caryophyllene_oxide,
			COALESCE(cedrol, 0) as cedrol,
			COALESCE(eucalyptol, 0) as eucalyptol,
			COALESCE(geraniol, 0) as geraniol,
			COALESCE(guaiol, 0) as guaiol,
			COALESCE(geranyl_acetate, 0) as geranyl_acetate,
			COALESCE(isoborneol, 0) as isoborneol,
			COALESCE(menthol, 0) as menthol,
			COALESCE(l_fenchone, 0) as l_fenchone,
			COALESCE(nerol, 0) as nerol,
			COALESCE(sabinene, 0) as sabinene,
			COALESCE(terpineol, 0) as terpineol,
			COALESCE(trans_b_farnesene, 0) as trans_b_farnesene,
			COALESCE(valencene, 0) as valencene,
			COALESCE(a_cedrene, 0) as a_cedrene,
			COALESCE(a_farnesene, 0) as a_farnesene,
			COALESCE(b_farnesene, 0) as b_farnesene,
			COALESCE(cis_nerolidol, 0) as cis_nerolidol,
			COALESCE(fenchol, 0) as fenchol,
			COALESCE(trans_nerolidol, 0) as trans_nerolidol
		FROM ct_brands
		WHERE registration_number = ?
	`, registrationNumber).Scan(
		&regNum, &brandName, &dosageForm, &brandingEntity, &approvalDate,
		&thc, &thca, &cbd, &cbda, &market, &chemotype, &processingTechnique,
		&solventsUsed, &nationalDrugCode, &productImageURL, &labelImageURL, &labAnalysisURL,
		&cbc, &cbn, &cbg, &cbga, &cbdv, &thcv,
		&aPinene, &bMyrcene, &bCaryophyllene, &bPinene, &limonene, &linalool, &humulene, &ocimene, &terpinolene,
		&aBisabolol, &aPhellandrene, &aTerpinene, &bEudesmol, &bTerpinene, &fenchone, &pulegol, &borneol, &isopulegol,
		&carene, &camphene, &camphor, &caryophylleneOxide, &cedrol, &eucalyptol, &geraniol, &guaiol, &geranylAcetate,
		&isoborneol, &menthol, &lFenchone, &nerol, &sabinene, &terpineol, &transBFarnesene, &valencene,
		&aCedrene, &aFarnesene, &bFarnesene, &cisNerolidol, &fenchol, &transNerolidol,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query product: %w", err)
	}
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("product not found")
	}

	b := models.BrandFromRaw(
		regNum, brandName, dosageForm, brandingEntity, approvalDate,
		productImageURL, labelImageURL, labAnalysisURL,
		thc, thca, cbd, cbda,
		market, chemotype, processingTechnique, solventsUsed, nationalDrugCode,
		cbc, cbn, cbg, cbga, cbdv, thcv,
		aPinene, bMyrcene, bCaryophyllene, bPinene, limonene, linalool, humulene, ocimene, terpinolene,
		aBisabolol, aPhellandrene, aTerpinene, bEudesmol, bTerpinene, fenchone, pulegol, borneol, isopulegol,
		carene, camphene, camphor, caryophylleneOxide, cedrol, eucalyptol, geraniol, guaiol, geranylAcetate,
		isoborneol, menthol, lFenchone, nerol, sabinene, terpineol, transBFarnesene, valencene,
		aCedrene, aFarnesene, bFarnesene, cisNerolidol, fenchol, transNerolidol,
	)
	product := models.ProductFromBrand(b)
	return &product, nil
}

// LoadTaxHistory returns monthly tax rows with period_end_date in [start, end],
// ordered chronologically. A zero start or end is treated as unbounded.
func (l *Loader) LoadTaxHistory(start, end time.Time) ([]models.TaxRecord, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}
	rows, err := l.db.Query(`
		SELECT period_end_date,
		       COALESCE(plant_material_tax, 0),
		       COALESCE(edible_products_tax, 0),
		       COALESCE(other_cannabis_tax, 0),
		       COALESCE(total_tax, 0)
		FROM ct_tax
		WHERE (? IS NULL OR period_end_date >= ?)
		  AND (? IS NULL OR period_end_date <= ?)
		ORDER BY period_end_date
	`, nullableTime(start), nullableTime(start), nullableTime(end), nullableTime(end))
	if err != nil {
		return nil, fmt.Errorf("failed to query tax: %w", err)
	}
	defer rows.Close()

	var out []models.TaxRecord
	for rows.Next() {
		var r models.TaxRecord
		if err := rows.Scan(&r.PeriodEnd, &r.PlantMaterialTax, &r.EdibleProductsTax, &r.OtherCannabisTax, &r.TotalTax); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LoadSalesHistory returns weekly sales rows with week_ending in [start, end],
// ordered chronologically. A zero start or end is treated as unbounded.
func (l *Loader) LoadSalesHistory(start, end time.Time) ([]models.SalesRecord, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}
	rows, err := l.db.Query(`
		SELECT week_ending,
		       COALESCE(adult_use, 0),
		       COALESCE(medical, 0),
		       COALESCE(total, 0),
		       COALESCE(adult_use_products_sold, 0),
		       COALESCE(medical_products_sold, 0),
		       COALESCE(total_products_sold, 0),
		       COALESCE(adult_use_avg_price, 0),
		       COALESCE(medical_avg_price, 0)
		FROM ct_weekly_sales
		WHERE (? IS NULL OR week_ending >= ?)
		  AND (? IS NULL OR week_ending <= ?)
		ORDER BY week_ending
	`, nullableTime(start), nullableTime(start), nullableTime(end), nullableTime(end))
	if err != nil {
		return nil, fmt.Errorf("failed to query sales: %w", err)
	}
	defer rows.Close()

	var out []models.SalesRecord
	for rows.Next() {
		var r models.SalesRecord
		if err := rows.Scan(
			&r.WeekEnding, &r.AdultUse, &r.Medical, &r.Total,
			&r.AdultUseProductsSold, &r.MedicalProductsSold, &r.TotalProductsSold,
			&r.AdultUseAvgPrice, &r.MedicalAvgPrice,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LoadZoning returns every row from ct_zoning, ordered by town. NULL status
// values come back as empty strings.
// The source CT API returns the literal string "null" for unreported
// statuses (not a SQL NULL), so NULLIF maps it back to NULL before
// COALESCE normalizes to an empty string.
func (l *Loader) LoadZoning() ([]models.ZoningRow, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}
	rows, err := l.db.Query(`
		SELECT town, COALESCE(NULLIF(status, 'null'), '')
		FROM ct_zoning
		ORDER BY town
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query zoning: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []models.ZoningRow
	for rows.Next() {
		var r models.ZoningRow
		if err := rows.Scan(&r.Town, &r.Status); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LoadRetailLocations returns every row from ct_retail_locations, ordered
// by business. Missing string fields come back as empty strings.
func (l *Loader) LoadRetailLocations() ([]models.RetailLocation, error) {
	if l.db == nil {
		return nil, fmt.Errorf("database not open")
	}
	rows, err := l.db.Query(`
		SELECT
			COALESCE(type, ''),
			COALESCE(business, ''),
			COALESCE(dba, ''),
			COALESCE(license, ''),
			COALESCE(street, ''),
			COALESCE(city, ''),
			COALESCE(zipcode, ''),
			COALESCE(website, ''),
			COALESCE(longitude, 0),
			COALESCE(latitude, 0)
		FROM ct_retail_locations
		ORDER BY business
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query retail locations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []models.RetailLocation
	for rows.Next() {
		var r models.RetailLocation
		if err := rows.Scan(
			&r.Type, &r.Business, &r.DBA, &r.License,
			&r.Street, &r.City, &r.Zipcode, &r.Website,
			&r.Longitude, &r.Latitude,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
