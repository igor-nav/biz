// analyze reads every businesses/<slug>/data.json file, computes common
// acquisition stats, and prints a side-by-side comparison table.
//
// SBA loan assumptions (all configurable via flags):
//   -rate   annual interest rate on the SBA 7(a) loan  (default 10.50 %)
//   -term   loan term in years                          (default 10)
//   -down   down-payment percentage                     (default 10 %)
//
// Usage:
//
//	go run ./cmd/analyze
//	go run ./cmd/analyze -rate 11.0 -down 15
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ── data model ────────────────────────────────────────────────────────────────

// YearlyFigure pairs a calendar year with a dollar amount.
type YearlyFigure struct {
	Year   int     `json:"year"`
	Amount float64 `json:"amount"`
}

// Business mirrors the schema of each businesses/<slug>/data.json file.
type Business struct {
	Name             string         `json:"name"`
	Type             string         `json:"type"`
	Location         string         `json:"location"`
	URL              string         `json:"url"`
	AskingPrice      float64        `json:"asking_price"`
	Revenue          []YearlyFigure `json:"revenue"`
	SDE              []YearlyFigure `json:"sde"` // Seller's Discretionary Earnings
	Inventory        float64        `json:"inventory"`
	FFE              float64        `json:"ffe"` // Furniture, Fixtures & Equipment
	RealEstate       string         `json:"real_estate"`
	LeaseMonthly     float64        `json:"lease_monthly"`
	LeaseExpiresYear int            `json:"lease_expires_year"`
	YearsInBusiness  int            `json:"years_in_business"`
	Employees        int            `json:"employees"`
	ReasonForSelling string         `json:"reason_for_selling"`
	AIOpportunity    string         `json:"ai_opportunity"`
	Notes            string         `json:"notes"`
}

// ── stats ─────────────────────────────────────────────────────────────────────

// Stats holds derived acquisition metrics for one business.
type Stats struct {
	Slug string
	Biz  *Business

	// Trailing figures (most recent year in the data)
	LatestSDE     float64
	LatestRevenue float64

	// Valuation
	SDEMultiple float64 // AskingPrice / LatestSDE

	// SBA 7(a) scenario
	DownPayment    float64 // AskingPrice * downPct
	LoanAmount     float64
	MonthlyPayment float64 // fixed-rate amortising payment
	AnnualDebtSvc  float64

	// Health ratios
	DSCR         float64 // LatestSDE / AnnualDebtSvc  (≥ 1.25 required by SBA)
	ROI          float64 // LatestSDE / DownPayment
	PaybackYears float64 // DownPayment / LatestSDE

	// Trend metrics (require ≥ 2 years of data)
	SDEMargin     float64 // LatestSDE / LatestRevenue
	RevenueGrowth float64 // YoY growth of the two most-recent years (fraction)
	SDEGrowth     float64 // YoY growth of the two most-recent years (fraction)
}

// latestFigure returns the YearlyFigure with the highest year value.
func latestFigure(figures []YearlyFigure) (YearlyFigure, bool) {
	if len(figures) == 0 {
		return YearlyFigure{}, false
	}
	best := figures[0]
	for _, f := range figures[1:] {
		if f.Year > best.Year {
			best = f
		}
	}
	return best, true
}

// previousFigure returns the figure for the year immediately before current.
func previousFigure(figures []YearlyFigure, current YearlyFigure) (YearlyFigure, bool) {
	target := current.Year - 1
	for _, f := range figures {
		if f.Year == target {
			return f, true
		}
	}
	return YearlyFigure{}, false
}

// monthlyPayment computes the fixed amortising payment for a loan.
//
//	P   = principal
//	r   = annual interest rate (fraction, e.g. 0.105)
//	n   = term in years
func monthlyPayment(principal, annualRate float64, termYears int) float64 {
	if principal <= 0 {
		return 0
	}
	mr := annualRate / 12
	n := float64(termYears * 12)
	if mr == 0 {
		return principal / n
	}
	return principal * mr * math.Pow(1+mr, n) / (math.Pow(1+mr, n) - 1)
}

// computeStats derives all metrics for one business given the SBA parameters.
func computeStats(slug string, b *Business, downPct, annualRate float64, termYears int) Stats {
	s := Stats{Slug: slug, Biz: b}

	if latestSDE, ok := latestFigure(b.SDE); ok {
		s.LatestSDE = latestSDE.Amount
		if prev, ok := previousFigure(b.SDE, latestSDE); ok && prev.Amount > 0 {
			s.SDEGrowth = (latestSDE.Amount - prev.Amount) / prev.Amount
		}
	}
	if latestRevenue, ok := latestFigure(b.Revenue); ok {
		s.LatestRevenue = latestRevenue.Amount
		if prev, ok := previousFigure(b.Revenue, latestRevenue); ok && prev.Amount > 0 {
			s.RevenueGrowth = (latestRevenue.Amount - prev.Amount) / prev.Amount
		}
	}

	if s.LatestSDE > 0 {
		s.SDEMultiple = b.AskingPrice / s.LatestSDE
	}

	s.DownPayment = b.AskingPrice * downPct
	s.LoanAmount = b.AskingPrice - s.DownPayment
	s.MonthlyPayment = monthlyPayment(s.LoanAmount, annualRate, termYears)
	s.AnnualDebtSvc = s.MonthlyPayment * 12

	if s.AnnualDebtSvc > 0 {
		s.DSCR = s.LatestSDE / s.AnnualDebtSvc
	}
	if s.DownPayment > 0 && s.LatestSDE > 0 {
		s.ROI = s.LatestSDE / s.DownPayment
		s.PaybackYears = s.DownPayment / s.LatestSDE
	}
	if s.LatestRevenue > 0 {
		s.SDEMargin = s.LatestSDE / s.LatestRevenue
	}

	return s
}

// ── loading ───────────────────────────────────────────────────────────────────

// loadBusinesses walks the businesses/ directory and parses every data.json.
func loadBusinesses(root string) ([]Stats, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", root, err)
	}

	var all []Stats
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		slug := e.Name()
		path := filepath.Join(root, slug, "data.json")
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue // directory without data.json – skip silently
			}
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		var b Business
		if err := json.Unmarshal(data, &b); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		all = append(all, Stats{Slug: slug, Biz: &b})
	}
	return all, nil
}

// ── rendering ─────────────────────────────────────────────────────────────────

// usd formats v as a dollar amount with thousands separators, e.g. $1,250,000.
func usd(v float64) string {
	s := fmt.Sprintf("%.0f", math.Round(v))
	// insert commas
	n := len(s)
	out := make([]byte, 0, n+n/3+1)
	for i, c := range s {
		pos := n - i
		if i > 0 && pos%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return "$" + string(out)
}

func pct(v float64) string { return fmt.Sprintf("%.1f%%", v*100) }
func f2(v float64) string  { return fmt.Sprintf("%.2f", v) }
func yn(v float64) string  { return fmt.Sprintf("%.1f yrs", v) }

// passFail returns a ✓ or ✗ indicator.
func passFail(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}

// dscrLabel annotates the DSCR value.
func dscrLabel(dscr float64) string {
	switch {
	case dscr >= 1.5:
		return "strong"
	case dscr >= 1.25:
		return "ok"
	default:
		return "FAIL"
	}
}

// printStats renders a single business's stats block.
func printStats(s Stats, downPct, rate float64, term int) {
	b := s.Biz
	sep := strings.Repeat("─", 58)

	fmt.Printf("\n%s\n", sep)
	fmt.Printf("  %-24s %s\n", b.Name, s.Slug)
	fmt.Printf("  %-24s %s\n", "Type:", b.Type)
	fmt.Printf("  %-24s %s\n", "Location:", b.Location)
	if b.URL != "" {
		fmt.Printf("  %-24s %s\n", "URL:", b.URL)
	}
	fmt.Printf("%s\n", sep)

	fmt.Printf("  %-30s %10s\n", "Asking price:", usd(b.AskingPrice))
	fmt.Printf("  %-30s %10s\n", "Latest SDE:", usd(s.LatestSDE))
	fmt.Printf("  %-30s %10s\n", "Latest revenue:", usd(s.LatestRevenue))
	fmt.Printf("  %-30s %10s\n", "Inventory:", usd(b.Inventory))
	fmt.Printf("  %-30s %10s\n", "FF&E:", usd(b.FFE))
	fmt.Println()

	fmt.Printf("  ── Valuation ──────────────────────────────────────\n")
	fmt.Printf("  %-30s %10s\n", "SDE multiple:", f2(s.SDEMultiple)+"×")
	fmt.Printf("  %-30s %10s\n", "SDE margin:", pct(s.SDEMargin))
	fmt.Printf("  %-30s %10s\n", "Revenue growth (YoY):", pct(s.RevenueGrowth))
	fmt.Printf("  %-30s %10s\n", "SDE growth (YoY):", pct(s.SDEGrowth))
	fmt.Println()

	fmt.Printf("  ── SBA 7(a) scenario  (%.0f%% down, %.2f%%, %d yr) ──\n",
		downPct*100, rate*100, term)
	fmt.Printf("  %-30s %10s\n", "Down payment:", usd(s.DownPayment))
	fmt.Printf("  %-30s %10s\n", "Loan amount:", usd(s.LoanAmount))
	fmt.Printf("  %-30s %10s\n", "Monthly loan payment:", usd(s.MonthlyPayment))
	fmt.Printf("  %-30s %10s\n", "Annual debt service:", usd(s.AnnualDebtSvc))
	fmt.Printf("  %-30s %10s   [%s – SBA min 1.25]\n",
		"DSCR:", f2(s.DSCR), dscrLabel(s.DSCR))
	fmt.Printf("  %-30s %10s   %s\n",
		"Down ≤ $250k?", usd(s.DownPayment), passFail(s.DownPayment <= 250_000))
	fmt.Println()

	fmt.Printf("  ── Returns ────────────────────────────────────────\n")
	fmt.Printf("  %-30s %10s\n", "ROI on down payment:", pct(s.ROI))
	fmt.Printf("  %-30s %10s\n", "Payback period:", yn(s.PaybackYears))
	fmt.Println()

	if b.AIOpportunity != "" {
		fmt.Printf("  AI opportunity: %s\n", b.AIOpportunity)
		fmt.Println()
	}
	if b.Notes != "" {
		fmt.Printf("  Notes: %s\n", b.Notes)
		fmt.Println()
	}
	fmt.Printf("  Years in business: %d   Employees: %d   Real estate: %s\n",
		b.YearsInBusiness, b.Employees, b.RealEstate)
	if b.LeaseMonthly > 0 {
		lease := "—"
		if b.LeaseExpiresYear > 0 {
			lease = fmt.Sprintf("expires %d", b.LeaseExpiresYear)
		}
		fmt.Printf("  Lease: %s/mo  (%s)\n", usd(b.LeaseMonthly), lease)
	}
	fmt.Printf("  Reason for selling: %s\n", b.ReasonForSelling)
}

// printSummary renders a compact comparison table of all candidates.
func printSummary(all []Stats) {
	sep := strings.Repeat("═", 100)
	fmt.Printf("\n%s\n", sep)
	fmt.Printf("  SUMMARY  –  %d candidate(s)\n", len(all))
	fmt.Printf("%s\n", sep)
	fmt.Printf("  %-28s  %10s  %8s  %10s  %8s  %6s  %6s  %8s\n",
		"Name", "Price", "SDE×", "Down", "DSCR", "ROI", "Payback", "Down≤250k")
	fmt.Printf("  %s\n", strings.Repeat("─", 96))
	for _, s := range all {
		fmt.Printf("  %-28s  %10s  %8s  %10s  %8s  %6s  %6s  %8s\n",
			truncate(s.Biz.Name, 28),
			usd(s.Biz.AskingPrice),
			f2(s.SDEMultiple)+"×",
			usd(s.DownPayment),
			f2(s.DSCR),
			pct(s.ROI),
			yn(s.PaybackYears),
			passFail(s.DownPayment <= 250_000),
		)
	}
	fmt.Printf("%s\n", sep)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	rate := flag.Float64("rate", 10.50, "SBA loan annual interest rate (%)")
	term := flag.Int("term", 10, "SBA loan term (years)")
	down := flag.Float64("down", 10.0, "Down payment percentage")
	dir := flag.String("dir", "businesses", "Root directory containing business sub-directories")
	flag.Parse()

	annualRate := *rate / 100
	downPct := *down / 100

	stats, err := loadBusinesses(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if len(stats) == 0 {
		fmt.Fprintf(os.Stderr, "no businesses found in %q\n", *dir)
		os.Exit(1)
	}

	// Recompute stats with the CLI parameters and sort by DSCR descending.
	computed := make([]Stats, len(stats))
	for i, s := range stats {
		computed[i] = computeStats(s.Slug, s.Biz, downPct, annualRate, *term)
	}
	sort.Slice(computed, func(i, j int) bool {
		return computed[i].DSCR > computed[j].DSCR
	})

	for _, s := range computed {
		printStats(s, downPct, annualRate, *term)
	}
	printSummary(computed)
}
