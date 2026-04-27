package main

import "testing"

func TestScoreRewardsKnownFinancials(t *testing.T) {
	terms := Terms{DownPct: 0.10, AnnualRate: 0.105, TermYears: 10}
	withFinancials := Business{
		Name:        "Property Management Company",
		Type:        "Property Management",
		Location:    "Coral Springs, FL",
		URL:         "https://example.com",
		AskingPrice: 275_000,
		SDE:         []YearlyFigure{{Year: 2026, Amount: 114_000}},
		AIOpportunity: "Automated tenant screening, predictive maintenance, dynamic pricing, " +
			"smart lock integration, automated lease generation, and AI chatbot support.",
		Notes:           "Recurring contracts and recession-resistant housing need.",
		YearsInBusiness: 40,
		RealEstate:      "unknown",
	}
	missingFinancials := Business{
		Name:          "Computer Repair and Device Sales",
		Type:          "Computer Repair",
		Location:      "Sunrise, FL",
		URL:           "https://example.com",
		AskingPrice:   100_000,
		AIOpportunity: "AI diagnostics, remote monitoring, automated ticket systems, and cybersecurity.",
		RealEstate:    "unknown",
	}

	known := ScoreBusiness(withFinancials, ComputeMetrics(withFinancials, terms))
	missing := ScoreBusiness(missingFinancials, ComputeMetrics(missingFinancials, terms))

	if known.Total <= missing.Total {
		t.Fatalf("known financials score %v should beat missing financials score %v", known.Total, missing.Total)
	}
	if known.Financial <= missing.Financial {
		t.Fatalf("financial score %v should beat %v", known.Financial, missing.Financial)
	}
}

func TestScoreWarnsOnMissingFinancials(t *testing.T) {
	b := Business{Name: "Lead", AskingPrice: 100_000}
	score := ScoreBusiness(b, ComputeMetrics(b, Terms{DownPct: 0.10, AnnualRate: 0.105, TermYears: 10}))

	if len(score.Warnings) == 0 {
		t.Fatal("expected warnings")
	}
	if score.Warnings[0] != "SDE/cash flow is missing" {
		t.Fatalf("first warning = %q", score.Warnings[0])
	}
}
