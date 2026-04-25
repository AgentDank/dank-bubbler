// Package models defines data structures for the dank-bubbler application
package models

import (
	"sort"
	"time"

	"github.com/AgentDank/dank-extract/sources/us/ct"
	"github.com/relvacode/iso8601"
)

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
	SolventsUsed        string     // solvents_used
	NationalDrugCode    string     // national_drug_code
	ProductImageURL     string     // product_image_url
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

// ProductFromBrand converts a canonical ct.Brand into the UI-facing Product model.
func ProductFromBrand(b ct.Brand) Product {
	p := Product{
		ID:                  b.RegistrationNumber,
		BrandName:           b.BrandName,
		DosageForm:          b.DosageForm,
		BrandingEntity:      b.BrandingEntity,
		RegistrationNumber:  b.RegistrationNumber,
		ApprovalDate:        b.ApprovalDate.Time,
		Market:              b.Market,
		Chemotype:           b.Chemotype,
		ProcessingTechnique: b.ProcessingTechnique,
		SolventsUsed:        b.SolventsUsed,
		NationalDrugCode:    b.NationalDrugCode,
		ProductImageURL:     b.ProductImage.URL,
	}

	if v, _, empty := b.TetrahydrocannabinolThc.Amount(); !empty {
		p.THC = v
	}
	if v, _, empty := b.TetrahydrocannabinolAcidThca.Amount(); !empty {
		p.THCA = v
	}
	if v, _, empty := b.CannabidiolsCbd.Amount(); !empty {
		p.CBD = v
	}
	if v, _, empty := b.CannabidiolAcidCbda.Amount(); !empty {
		p.CBDA = v
	}

	addCann := func(name string, m ct.Measure) {
		if v, _, empty := m.Amount(); !empty && v > 0 {
			p.OtherCannabinoids = append(p.OtherCannabinoids, Compound{Name: name, Percentage: v})
		}
	}
	addCann("CBC", b.CannabichromeneCbc)
	addCann("CBN", b.CannbinolCbn)
	addCann("CBG", b.Cbg)
	addCann("CBGA", b.CbgA)
	addCann("CBDV", b.CannabavarinCbdv)
	addCann("THCV", b.TetrahydrocannabivarinThcv)

	addTerp := func(name string, m ct.Measure) {
		if v, _, empty := m.Amount(); !empty && v > 0 {
			p.Compounds = append(p.Compounds, Compound{Name: name, Percentage: v})
		}
	}
	addTerp("α-Pinene", b.APinene)
	addTerp("β-Myrcene", b.BMyrcene)
	addTerp("β-Caryophyllene", b.BCaryophyllene)
	addTerp("β-Pinene", b.BPinene)
	addTerp("Limonene", b.Limonene)
	addTerp("Linalool", b.LinaloolLin)
	addTerp("Humulene", b.HumuleneHum)
	addTerp("Ocimene", b.Ocimene)
	addTerp("Terpinolene", b.Terpinolene)
	addTerp("α-Bisabolol", b.ABisabolol)
	addTerp("α-Phellandrene", b.APhellandrene)
	addTerp("α-Terpinene", b.ATerpinene)
	addTerp("β-Eudesmol", b.BEudesmol)
	addTerp("β-Terpinene", b.BTerpinene)
	addTerp("Fenchone", b.Fenchone)
	addTerp("Pulegol", b.Pulegol)
	addTerp("Borneol", b.Borneol)
	addTerp("Isopulegol", b.Isopulegol)
	addTerp("Carene", b.Carene)
	addTerp("Camphene", b.Camphene)
	addTerp("Camphor", b.Camphor)
	addTerp("Caryophyllene Oxide", b.CaryophylleneOxide)
	addTerp("Cedrol", b.Cedrol)
	addTerp("Eucalyptol", b.Eucalyptol)
	addTerp("Geraniol", b.Geraniol)
	addTerp("Guaiol", b.Guaiol)
	addTerp("Geranyl Acetate", b.GeranylAcetate)
	addTerp("Isoborneol", b.Isoborneol)
	addTerp("Menthol", b.Menthol)
	addTerp("l-Fenchone", b.LFenchone)
	addTerp("Nerol", b.Nerol)
	addTerp("Sabinene", b.Sabinene)
	addTerp("Terpineol", b.Terpineol)
	addTerp("trans-β-Farnesene", b.TransBFarnesene)
	addTerp("Valencene", b.Valencene)
	addTerp("α-Cedrene", b.ACedrene)
	addTerp("α-Farnesene", b.AFarnesene)
	addTerp("β-Farnesene", b.BFarnesene)
	addTerp("cis-Nerolidol", b.CisNerolidol)
	addTerp("Fenchol", b.Fenchol)
	addTerp("trans-Nerolidol", b.TransNerolidol)

	sort.Slice(p.OtherCannabinoids, func(i, j int) bool {
		return p.OtherCannabinoids[i].Percentage > p.OtherCannabinoids[j].Percentage
	})
	sort.Slice(p.Compounds, func(i, j int) bool {
		return p.Compounds[i].Percentage > p.Compounds[j].Percentage
	})

	return p
}

// BrandFromRaw creates a ct.Brand from raw scalar values (e.g. DB scan results).
// Measure fields are built with ct.NewMeasure; images and approval date are
// assembled from their parts.  This is a convenience helper for the data layer.
func BrandFromRaw(
	regNum, brandName, dosageForm, brandingEntity string,
	approvalDate time.Time,
	productImageURL, labelImageURL, labAnalysisURL string,
	thc, thca, cbd, cbda float64,
	market, chemotype, processingTechnique, solventsUsed, nationalDrugCode string,
	// Cannabinoids
	cbc, cbn, cbg, cbga, cbdv, thcv float64,
	// Terpenes
	aPinene, bMyrcene, bCaryophyllene, bPinene, limonene, linalool, humulene, ocimene, terpinolene float64,
	aBisabolol, aPhellandrene, aTerpinene, bEudesmol, bTerpinene, fenchone, pulegol, borneol, isopulegol float64,
	carene, camphene, camphor, caryophylleneOxide, cedrol, eucalyptol, geraniol, guaiol, geranylAcetate float64,
	isoborneol, menthol, lFenchone, nerol, sabinene, terpineol, transBFarnesene, valencene float64,
	aCedrene, aFarnesene, bFarnesene, cisNerolidol, fenchol, transNerolidol float64,
) ct.Brand {
	return ct.Brand{
		RegistrationNumber:  regNum,
		BrandName:           brandName,
		DosageForm:          dosageForm,
		BrandingEntity:      brandingEntity,
		ApprovalDate:        iso8601.Time{Time: approvalDate},
		ProductImage:        ct.Image{URL: productImageURL},
		LabelImage:          ct.Image{URL: labelImageURL},
		LabAnalysis:         ct.Image{URL: labAnalysisURL},
		Market:              market,
		Chemotype:           chemotype,
		ProcessingTechnique: processingTechnique,
		SolventsUsed:        solventsUsed,
		NationalDrugCode:    nationalDrugCode,
		TetrahydrocannabinolThc:      ct.NewMeasure(thc),
		TetrahydrocannabinolAcidThca: ct.NewMeasure(thca),
		CannabidiolsCbd:              ct.NewMeasure(cbd),
		CannabidiolAcidCbda:          ct.NewMeasure(cbda),
		CannabichromeneCbc:           ct.NewMeasure(cbc),
		CannbinolCbn:                 ct.NewMeasure(cbn),
		Cbg:                          ct.NewMeasure(cbg),
		CbgA:                         ct.NewMeasure(cbga),
		CannabavarinCbdv:             ct.NewMeasure(cbdv),
		TetrahydrocannabivarinThcv:   ct.NewMeasure(thcv),
		APinene:                      ct.NewMeasure(aPinene),
		BMyrcene:                     ct.NewMeasure(bMyrcene),
		BCaryophyllene:               ct.NewMeasure(bCaryophyllene),
		BPinene:                      ct.NewMeasure(bPinene),
		Limonene:                     ct.NewMeasure(limonene),
		LinaloolLin:                  ct.NewMeasure(linalool),
		HumuleneHum:                  ct.NewMeasure(humulene),
		Ocimene:                      ct.NewMeasure(ocimene),
		Terpinolene:                  ct.NewMeasure(terpinolene),
		ABisabolol:                   ct.NewMeasure(aBisabolol),
		APhellandrene:                ct.NewMeasure(aPhellandrene),
		ATerpinene:                   ct.NewMeasure(aTerpinene),
		BEudesmol:                    ct.NewMeasure(bEudesmol),
		BTerpinene:                   ct.NewMeasure(bTerpinene),
		Fenchone:                     ct.NewMeasure(fenchone),
		Pulegol:                      ct.NewMeasure(pulegol),
		Borneol:                      ct.NewMeasure(borneol),
		Isopulegol:                   ct.NewMeasure(isopulegol),
		Carene:                       ct.NewMeasure(carene),
		Camphene:                     ct.NewMeasure(camphene),
		Camphor:                      ct.NewMeasure(camphor),
		CaryophylleneOxide:           ct.NewMeasure(caryophylleneOxide),
		Cedrol:                       ct.NewMeasure(cedrol),
		Eucalyptol:                   ct.NewMeasure(eucalyptol),
		Geraniol:                     ct.NewMeasure(geraniol),
		Guaiol:                       ct.NewMeasure(guaiol),
		GeranylAcetate:               ct.NewMeasure(geranylAcetate),
		Isoborneol:                   ct.NewMeasure(isoborneol),
		Menthol:                      ct.NewMeasure(menthol),
		LFenchone:                    ct.NewMeasure(lFenchone),
		Nerol:                        ct.NewMeasure(nerol),
		Sabinene:                     ct.NewMeasure(sabinene),
		Terpineol:                    ct.NewMeasure(terpineol),
		TransBFarnesene:              ct.NewMeasure(transBFarnesene),
		Valencene:                    ct.NewMeasure(valencene),
		ACedrene:                     ct.NewMeasure(aCedrene),
		AFarnesene:                   ct.NewMeasure(aFarnesene),
		BFarnesene:                   ct.NewMeasure(bFarnesene),
		CisNerolidol:                 ct.NewMeasure(cisNerolidol),
		Fenchol:                      ct.NewMeasure(fenchol),
		TransNerolidol:               ct.NewMeasure(transNerolidol),
	}
}
