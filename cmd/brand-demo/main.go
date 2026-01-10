// Brand Demo - A BubbleTea-based product browser for cannabis brands
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/pflag"
)

const (
	version = "0.0.1"
)

var usageFormat = `usage:  %s [--help] [options]

A product browser for cannabis brands based on the dank-data repository.
Browse products by brand, name, cannabis type, or date.

Features:
  - Browse cannabis products with filtering
  - View product details including cannabinoid profiles
  - Horizontal bar chart showing top 8 cannabinoids
  - Data from dank-data repository (DuckDB format)

Example:  $ db-brand-demo --db us_ct_brands.duckdb.zst

`

type model struct {
	// TODO: Add UI components
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		// Handle window resize
	}
	return m, nil
}

func (m model) View() string {
	return "Brand Demo - Not Yet Implemented\n"
}

func main() {
	var dbPath string
	var verbose, showHelp, showVersion bool

	pflag.StringVarP(&dbPath, "db", "d", "", "Path to DuckDB database file")
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

	if len(dbPath) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: missing required argument: --db\n")
		fmt.Fprintf(os.Stderr, "usage:  %s [--help] [options]\n", os.Args[0])
		os.Exit(1)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "INFO: db=%s\n", dbPath)
	}

	// TODO: Load data from DuckDB
	// TODO: Initialize UI components
	// TODO: Run BubbleTea program

	m := model{}
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
