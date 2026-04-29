// analyze reads every businesses/<slug>/data.json file, computes common
// acquisition stats, and prints a side-by-side comparison table.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	core "github.com/igor-nav/biz/internal/biz"
)

type analyzedCandidate struct {
	core.Candidate
	Metrics core.Metrics
}

func usd(v float64) string {
	return core.FormatUSD(v)
}

func pct(v float64) string { return fmt.Sprintf("%.1f%%", v*100) }
func f2(v float64) string  { return fmt.Sprintf("%.2f", v) }
func yn(v float64) string  { return fmt.Sprintf("%.1f yrs", v) }

func passFail(ok bool) string {
	if ok {
		return "yes"
	}
	return "no"
}

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

func printStats(entry analyzedCandidate, terms core.Terms) {
	b := entry.Biz
	m := entry.Metrics
	sep := strings.Repeat("-", 58)

	fmt.Printf("\n%s\n", sep)
	fmt.Printf("  %-24s %s\n", b.Name, entry.Slug)
	fmt.Printf("  %-24s %s\n", "Type:", b.Type)
	fmt.Printf("  %-24s %s\n", "Location:", b.Location)
	if b.URL != "" {
		fmt.Printf("  %-24s %s\n", "URL:", b.URL)
	}
	fmt.Printf("%s\n", sep)

	fmt.Printf("  %-30s %10s\n", "Asking price:", usd(b.AskingPrice))
	fmt.Printf("  %-30s %10s\n", "Latest SDE:", usd(m.LatestSDE))
	fmt.Printf("  %-30s %10s\n", "Latest revenue:", usd(m.LatestRevenue))
	fmt.Printf("  %-30s %10s\n", "Inventory:", usd(b.Inventory))
	fmt.Printf("  %-30s %10s\n", "FF&E:", usd(b.FFE))
	fmt.Println()

	fmt.Printf("  -- Valuation --------------------------------------\n")
	fmt.Printf("  %-30s %10s\n", "SDE multiple:", f2(m.SDEMultiple)+"x")
	fmt.Printf("  %-30s %10s\n", "SDE margin:", pct(m.SDEMargin))
	fmt.Printf("  %-30s %10s\n", "Revenue growth (YoY):", pct(m.RevenueGrowth))
	fmt.Printf("  %-30s %10s\n", "SDE growth (YoY):", pct(m.SDEGrowth))
	fmt.Println()

	fmt.Printf("  -- SBA 7(a) scenario  (%.0f%% down, %.2f%%, %d yr) --\n",
		terms.DownPct*100, terms.AnnualRate*100, terms.TermYears)
	fmt.Printf("  %-30s %10s\n", "Down payment:", usd(m.DownPayment))
	fmt.Printf("  %-30s %10s\n", "Loan amount:", usd(m.LoanAmount))
	fmt.Printf("  %-30s %10s\n", "Monthly loan payment:", usd(m.MonthlyPayment))
	fmt.Printf("  %-30s %10s\n", "Annual debt service:", usd(m.AnnualDebtService))
	fmt.Printf("  %-30s %10s   [%s - SBA min 1.25]\n",
		"DSCR:", f2(m.DSCR), dscrLabel(m.DSCR))
	fmt.Printf("  %-30s %10s   %s\n",
		"Down <= $250k?", usd(m.DownPayment), passFail(m.DownPayment <= 250_000))
	fmt.Println()

	fmt.Printf("  -- Returns ----------------------------------------\n")
	fmt.Printf("  %-30s %10s\n", "ROI on down payment:", pct(m.ROI))
	fmt.Printf("  %-30s %10s\n", "Payback period:", yn(m.PaybackYears))
	fmt.Println()

	if b.AIOpportunity != "" {
		fmt.Printf("  AI opportunity: %s\n\n", b.AIOpportunity)
	}
	if b.Notes != "" {
		fmt.Printf("  Notes: %s\n\n", b.Notes)
	}
	fmt.Printf("  Years in business: %d   Employees: %d   Real estate: %s\n",
		b.YearsInBusiness, b.Employees, b.RealEstate)
	if b.LeaseMonthly > 0 {
		lease := "-"
		if b.LeaseExpiresYear > 0 {
			lease = fmt.Sprintf("expires %d", b.LeaseExpiresYear)
		}
		fmt.Printf("  Lease: %s/mo  (%s)\n", usd(b.LeaseMonthly), lease)
	}
	fmt.Printf("  Reason for selling: %s\n", b.ReasonForSelling)
}

func printSummary(all []analyzedCandidate) {
	sep := strings.Repeat("=", 100)
	fmt.Printf("\n%s\n", sep)
	fmt.Printf("  SUMMARY  -  %d candidate(s)\n", len(all))
	fmt.Printf("%s\n", sep)
	fmt.Printf("  %-28s  %10s  %8s  %10s  %8s  %6s  %8s  %10s\n",
		"Name", "Price", "SDE x", "Down", "DSCR", "ROI", "Payback", "Down<=250k")
	fmt.Printf("  %s\n", strings.Repeat("-", 96))
	for _, entry := range all {
		b := entry.Biz
		m := entry.Metrics
		fmt.Printf("  %-28s  %10s  %8s  %10s  %8s  %6s  %8s  %10s\n",
			truncate(b.Name, 28),
			usd(b.AskingPrice),
			f2(m.SDEMultiple)+"x",
			usd(m.DownPayment),
			f2(m.DSCR),
			pct(m.ROI),
			yn(m.PaybackYears),
			passFail(m.DownPayment <= 250_000),
		)
	}
	fmt.Printf("%s\n", sep)
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "."
}

func main() {
	rate := flag.Float64("rate", 10.50, "SBA loan annual interest rate (%)")
	term := flag.Int("term", 10, "SBA loan term (years)")
	down := flag.Float64("down", 10.0, "Down payment percentage")
	dir := flag.String("dir", "businesses", "Root directory containing business sub-directories")
	flag.Parse()

	terms := core.Terms{
		DownPct:    *down / 100,
		AnnualRate: *rate / 100,
		TermYears:  *term,
	}

	candidates, err := core.LoadCandidates(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if len(candidates) == 0 {
		fmt.Fprintf(os.Stderr, "no businesses found in %q\n", *dir)
		os.Exit(1)
	}

	computed := make([]analyzedCandidate, len(candidates))
	for i, candidate := range candidates {
		computed[i] = analyzedCandidate{
			Candidate: candidate,
			Metrics:   core.ComputeMetrics(candidate.Biz, terms),
		}
	}
	sort.Slice(computed, func(i, j int) bool {
		return computed[i].Metrics.DSCR > computed[j].Metrics.DSCR
	})

	for _, entry := range computed {
		printStats(entry, terms)
	}
	printSummary(computed)
}
