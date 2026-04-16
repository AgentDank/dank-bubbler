// Brand Demo - A BubbleTea-based product browser for cannabis brands
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/klauspost/compress/zstd"
	"github.com/spf13/pflag"

	"github.com/AgentDank/dank-bubbler/internal/data"
	"github.com/AgentDank/dank-bubbler/internal/ui"
)

const (
	version           = "0.0.1"
	defaultDBPath     = "dank-data.duckdb"
	brandsDatabaseURL = "https://github.com/AgentDank/dank-data/raw/main/snapshots/us/ct/dank-data.duckdb.zst"
)

var usageFormat = `usage:  %s [--help] [options]

A product browser for cannabis brands based on the dank-data repository.
Browse products by brand, name, cannabis type, or date.

Features:
  - Browse cannabis products with filtering
  - View product details including compound profiles
  - Horizontal bar chart showing top 6 compounds
  - Data from dank-data repository (DuckDB format)
  - Automatic download of database if not present

Example:  $ db-brand-demo
          $ db-brand-demo --db custom-brands.duckdb

`

type model struct {
	browser *ui.ProductBrowser
	loader  *data.Loader
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	browserModel, cmd := m.browser.Update(msg)
	m.browser = browserModel.(*ui.ProductBrowser)

	switch msg.(type) {
	case tea.KeyMsg:
		// Key handling is done in ProductBrowser.Update()
	case tea.WindowSizeMsg:
		// Window handling is done in ProductBrowser.Update()
	}

	return m, cmd
}

func (m model) View() tea.View {
	v := m.browser.View()
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// downloadDatabase downloads the zstd-compressed DuckDB file and decompresses it
func downloadDatabase(url, targetPath string) error {
	fmt.Fprintf(os.Stderr, "Downloading brands database...\n")

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download database: HTTP %d", resp.StatusCode)
	}

	// Decompress zstd stream
	decoder, err := zstd.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to create zstd decoder: %w", err)
	}
	defer decoder.Close()

	// Write decompressed data to file
	outFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, decoder); err != nil {
		os.Remove(targetPath) // Clean up on failure
		return fmt.Errorf("failed to write database: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Database downloaded to %s\n", targetPath)
	return nil
}

// ensureDatabase ensures the database exists and has the current brands table.
func ensureDatabase(dbPath string) error {
	// If database doesn't exist, download it
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Database not found at %s\n", dbPath)
		return downloadDatabase(brandsDatabaseURL, dbPath)
	}

	// Check if brands table exists
	loader := data.NewLoader(dbPath)
	if err := loader.Open(); err != nil {
		return err
	}
	defer loader.Close()

	hasBrands, err := loader.HasBrandsTable()
	if err != nil {
		return err
	}

	if !hasBrands {
		fmt.Fprintf(os.Stderr, "%s table not found, downloading fresh database...\n", "ct_brands")
		os.Remove(dbPath)
		return downloadDatabase(brandsDatabaseURL, dbPath)
	}

	return nil
}

func main() {
	var dbPath string
	var verbose, showHelp, showVersion bool

	pflag.StringVarP(&dbPath, "db", "d", defaultDBPath, "Path to DuckDB database file")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	pflag.BoolVarP(&showHelp, "help", "", false, "show help")
	pflag.BoolVarP(&showVersion, "version", "", false, "show version")
	pflag.Parse()

	if showHelp {
		fmt.Fprintf(os.Stdout, usageFormat, os.Args[0])
		pflag.PrintDefaults()
		os.Exit(0)
	}

	if showVersion {
		fmt.Fprintf(os.Stdout, "brand-demo v%s\n", version)
		os.Exit(0)
	}

	// Convert to absolute path
	absDbPath, err := filepath.Abs(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: invalid path: %s\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "INFO: db=%s\n", absDbPath)
	}

	// Ensure database exists and is accessible
	if err := ensureDatabase(absDbPath); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "INFO: Database ready at %s\n", absDbPath)
	}

	// Load data from DuckDB
	loader := data.NewLoader(absDbPath)
	if err := loader.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to open database: %s\n", err)
		os.Exit(1)
	}
	defer loader.Close()

	if verbose {
		fmt.Fprintf(os.Stderr, "INFO: Loading brands and products...\n")
	}

	products, err := loader.LoadProducts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to load products: %s\n", err)
		os.Exit(1)
	}

	brands, err := loader.LoadBrands()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to load brands: %s\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "INFO: Loaded %d products from %d brands\n", len(products), len(brands))
	}

	// Initialize UI components and BubbleTea program
	browser := ui.NewProductBrowser(products, brands, loader)
	m := model{
		browser: browser,
		loader:  loader,
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
