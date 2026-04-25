// import fetches a for-sale business listing from a supported URL and creates a
// businesses/<slug>/data.json file following the project schema.
//
// Usage:
//
//	go run ./cmd/import <URL>
//	go run ./cmd/import -dir /path/to/businesses <URL>
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

// ── data model ────────────────────────────────────────────────────────────────

// YearlyFigure pairs a calendar year with a dollar amount.
// It mirrors the schema used by businesses/<slug>/data.json.
type YearlyFigure struct {
	Year   int     `json:"year"`
	Amount float64 `json:"amount"`
}

// Business mirrors the schema of businesses/<slug>/data.json.
type Business struct {
	Name             string         `json:"name"`
	Type             string         `json:"type"`
	Location         string         `json:"location"`
	URL              string         `json:"url,omitempty"`
	AskingPrice      float64        `json:"asking_price"`
	Revenue          []YearlyFigure `json:"revenue,omitempty"`
	SDE              []YearlyFigure `json:"sde,omitempty"`
	Inventory        float64        `json:"inventory,omitempty"`
	FFE              float64        `json:"ffe,omitempty"`
	RealEstate       string         `json:"real_estate,omitempty"`
	LeaseMonthly     float64        `json:"lease_monthly,omitempty"`
	LeaseExpiresYear int            `json:"lease_expires_year,omitempty"`
	YearsInBusiness  int            `json:"years_in_business,omitempty"`
	Employees        int            `json:"employees,omitempty"`
	ReasonForSelling string         `json:"reason_for_selling,omitempty"`
	AIOpportunity    string         `json:"ai_opportunity,omitempty"`
	Notes            string         `json:"notes,omitempty"`
}

// ── provider registry ─────────────────────────────────────────────────────────

// Provider describes a site-specific listing scraper.
type Provider interface {
	// Supports reports whether this provider can handle the given URL.
	Supports(u *url.URL) bool
	// Fetch downloads the listing and returns the populated Business and a slug.
	Fetch(rawURL string) (*Business, string, error)
}

// registry is the ordered list of registered listing providers.
// Add new providers here to extend support for additional sites.
var registry = []Provider{
	&bizBuySellProvider{},
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	dir := flag.String("dir", "businesses", "root directory for candidate sub-dirs")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: import [flags] <URL>\n\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nSupported providers:")
		fmt.Fprintln(os.Stderr, "  BizBuySell  (bizbuysell.com)")
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	rawURL := flag.Arg(0)

	u, err := url.Parse(rawURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "import: invalid URL %q: %v\n", rawURL, err)
		os.Exit(1)
	}

	var provider Provider
	for _, p := range registry {
		if p.Supports(u) {
			provider = p
			break
		}
	}
	if provider == nil {
		fmt.Fprintf(os.Stderr, "import: no provider supports %q\n", u.Hostname())
		os.Exit(1)
	}

	fmt.Printf("import: fetching %s\n", rawURL)

	biz, slug, err := provider.Fetch(rawURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "import: fetch failed: %v\n", err)
		os.Exit(1)
	}

	outDir := filepath.Join(*dir, slug)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "import: mkdir %s: %v\n", outDir, err)
		os.Exit(1)
	}

	outPath := filepath.Join(outDir, "data.json")
	if _, err := os.Stat(outPath); err == nil {
		fmt.Fprintf(os.Stderr, "import: %s already exists; refusing to overwrite\n", outPath)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(biz, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "import: marshal failed: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "import: write %s: %v\n", outPath, err)
		os.Exit(1)
	}

	fmt.Printf("import: created %s\n", outPath)
	fmt.Println("import: review the file and fill in any missing fields before running analyze")
}
