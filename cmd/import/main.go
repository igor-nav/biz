// import fetches a for-sale business listing from a supported URL and creates a
// businesses/<slug>/data.json file following the project schema.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	core "github.com/igor-nav/biz/internal/biz"
)

type Provider interface {
	Supports(u *url.URL) bool
	Fetch(rawURL string) (*core.Business, string, error)
}

var registry = []Provider{
	&bizBuySellProvider{},
	&genericSiteProvider{
		siteName:        "BizQuest",
		slugPrefix:      "bizquest",
		hosts:           []string{"bizquest.com"},
		searchPathHints: []string{"businesses-for-sale-in-", "-businesses-for-sale-in-"},
	},
	&genericSiteProvider{
		siteName:        "BusinessMart",
		slugPrefix:      "businessmart",
		hosts:           []string{"businessmart.com"},
		searchPathHints: []string{"businesses-for-sale/florida/"},
	},
	&genericSiteProvider{
		siteName:        "Truforte",
		slugPrefix:      "truforte",
		hosts:           []string{"trufortebusinessgroup.com", "truforte.com"},
		searchPathHints: []string{"property-management-businesses-for-sale", "bbf-sba-lender-pre-qualified"},
	},
	&genericSiteProvider{
		siteName:        "KMF Business Advisors",
		slugPrefix:      "kmf",
		hosts:           []string{"kmfbusinessadvisors.com"},
		searchPathHints: []string{"sba-approved-businesses-for-sale"},
	},
}

func main() {
	dir := flag.String("dir", "businesses", "root directory for candidate sub-dirs")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: import [flags] <URL>\n\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nSupported providers:")
		fmt.Fprintln(os.Stderr, "  BizBuySell  (bizbuysell.com)")
		fmt.Fprintln(os.Stderr, "  BizQuest    (bizquest.com)")
		fmt.Fprintln(os.Stderr, "  BusinessMart (businessmart.com)")
		fmt.Fprintln(os.Stderr, "  Truforte    (trufortebusinessgroup.com)")
		fmt.Fprintln(os.Stderr, "  KMF Business Advisors (kmfbusinessadvisors.com)")
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
