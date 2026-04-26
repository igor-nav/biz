package main

import (
	"net/url"
	"testing"
)

func TestParseDollarSupportsSuffixesAndRanges(t *testing.T) {
	tests := map[string]float64{
		"$1,250,000": 1_250_000,
		"$39.5K":     39_500,
		"$1.45M":     1_450_000,
		"$400-600K":  500_000,
	}

	for input, want := range tests {
		if got := parseDollar(input); got != want {
			t.Fatalf("parseDollar(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestParseGenericHTMLExtractsListingFields(t *testing.T) {
	body := `
		<html>
			<head><title>Computer Repair and Device Sales | BizQuest</title></head>
			<body>
				<h1>Computer Repair and Device Sales</h1>
				<p>Location: Sunrise, FL</p>
				<p>Asking Price: $100K</p>
				<p>Cash Flow: $114,000</p>
				<p>Gross Revenue: $250,000</p>
			</body>
		</html>`

	var biz Business
	parseGenericHTML(body, &biz, "BizQuest")

	if biz.Name != "Computer Repair and Device Sales" {
		t.Fatalf("Name = %q", biz.Name)
	}
	if biz.Location != "Sunrise, FL" {
		t.Fatalf("Location = %q", biz.Location)
	}
	if biz.AskingPrice != 100_000 {
		t.Fatalf("AskingPrice = %v", biz.AskingPrice)
	}
	if len(biz.SDE) != 1 || biz.SDE[0].Amount != 114_000 {
		t.Fatalf("SDE = %#v", biz.SDE)
	}
	if len(biz.Revenue) != 1 || biz.Revenue[0].Amount != 250_000 {
		t.Fatalf("Revenue = %#v", biz.Revenue)
	}
}

func TestGenericProviderRejectsSearchPages(t *testing.T) {
	provider := &genericSiteProvider{
		siteName:        "BusinessMart",
		searchPathHints: []string{"businesses-for-sale/florida/"},
	}

	u, err := url.Parse("https://www.businessmart.com/businesses-for-sale/florida/broward-county.php")
	if err != nil {
		t.Fatal(err)
	}
	if !provider.isSearchPage(u) {
		t.Fatal("expected search/results page to be rejected")
	}
}
