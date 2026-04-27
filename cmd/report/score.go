package main

import (
	"math"
	"strings"
)

type Terms struct {
	DownPct    float64
	AnnualRate float64
	TermYears  int
}

type Metrics struct {
	LatestSDE     float64
	LatestRevenue float64
	SDEMultiple   float64
	SDEMargin     float64
	DownPayment   float64
	LoanAmount    float64
	MonthlyDebt   float64
	AnnualDebt    float64
	DSCR          float64
	PaybackYears  float64
}

type Score struct {
	Total          float64
	Financial      float64
	AIFit          float64
	Budget         float64
	Recurring      float64
	DataConfidence float64
	Reasons        []string
	Warnings       []string
}

func ComputeMetrics(b Business, terms Terms) Metrics {
	var m Metrics
	if latestSDE, ok := latestFigure(b.SDE); ok {
		m.LatestSDE = latestSDE.Amount
	}
	if latestRevenue, ok := latestFigure(b.Revenue); ok {
		m.LatestRevenue = latestRevenue.Amount
	}

	if m.LatestSDE > 0 {
		m.SDEMultiple = b.AskingPrice / m.LatestSDE
	}
	if m.LatestRevenue > 0 {
		m.SDEMargin = m.LatestSDE / m.LatestRevenue
	}

	m.DownPayment = b.AskingPrice * terms.DownPct
	m.LoanAmount = b.AskingPrice - m.DownPayment
	m.MonthlyDebt = monthlyPayment(m.LoanAmount, terms.AnnualRate, terms.TermYears)
	m.AnnualDebt = m.MonthlyDebt * 12

	if m.AnnualDebt > 0 {
		m.DSCR = m.LatestSDE / m.AnnualDebt
	}
	if m.LatestSDE > 0 && m.DownPayment > 0 {
		m.PaybackYears = m.DownPayment / m.LatestSDE
	}

	return m
}

func ScoreBusiness(b Business, m Metrics) Score {
	var s Score
	text := searchableText(b)

	s.Financial = financialScore(m)
	s.AIFit = aiFitScore(text)
	s.Budget = budgetScore(b, m, text)
	s.Recurring = recurringScore(b, text)
	s.DataConfidence = dataConfidenceScore(b, m)

	s.Reasons = reasons(b, m, s, text)
	s.Warnings = warnings(b, m, text)
	s.Total = clamp(s.Financial+s.AIFit+s.Budget+s.Recurring+s.DataConfidence, 0, 100)
	return s
}

func financialScore(m Metrics) float64 {
	if m.LatestSDE <= 0 {
		return 3
	}

	var score float64
	switch {
	case m.DSCR >= 2.0:
		score += 12
	case m.DSCR >= 1.5:
		score += 10
	case m.DSCR >= 1.25:
		score += 7
	case m.DSCR > 0:
		score += 3
	}

	switch {
	case m.SDEMultiple > 0 && m.SDEMultiple <= 2.5:
		score += 8
	case m.SDEMultiple <= 3.5:
		score += 6
	case m.SDEMultiple <= 4:
		score += 4
	case m.SDEMultiple <= 5:
		score += 2
	}

	switch {
	case m.SDEMargin >= 0.30:
		score += 4
	case m.SDEMargin >= 0.20:
		score += 3
	case m.SDEMargin > 0:
		score += 1.5
	}

	switch {
	case m.PaybackYears > 0 && m.PaybackYears <= 0.5:
		score += 6
	case m.PaybackYears <= 1.5:
		score += 4
	case m.PaybackYears <= 3:
		score += 2
	}

	return clamp(score, 0, 30)
}

func aiFitScore(text string) float64 {
	if strings.TrimSpace(text) == "" {
		return 0
	}

	score := 5.0
	groups := [][]string{
		{"ai ", " ai", "artificial intelligence", "machine learning"},
		{"automation", "automated", "workflow"},
		{"predictive", "forecast", "forecasting"},
		{"route", "routing", "scheduling"},
		{"iot", "sensor", "monitoring", "remote"},
		{"crm", "portal", "customer retention"},
		{"saas", "software", "platform", "e-commerce", "ecom"},
		{"diagnostic", "cybersecurity", "ticket"},
		{"dynamic pricing", "bidding", "lead scoring", "upsell"},
	}
	for _, group := range groups {
		if containsAny(text, group...) {
			score += 2.2
		}
	}

	if containsAny(text, "property management", "pest control", "computer repair", "telehealth", "streaming platform", "commercial cleaning", "managed it") {
		score += 3
	}
	return clamp(score, 0, 25)
}

func budgetScore(b Business, m Metrics, text string) float64 {
	if b.AskingPrice <= 0 {
		return 3
	}

	var score float64
	switch {
	case b.AskingPrice <= 500_000:
		score += 12
	case b.AskingPrice <= 750_000:
		score += 9
	case b.AskingPrice <= 1_000_000:
		score += 6
	case b.AskingPrice <= 1_500_000:
		score += 3
	}

	switch {
	case m.DownPayment <= 100_000:
		score += 5
	case m.DownPayment <= 250_000:
		score += 4
	}

	if containsAny(text, "sba pre-qualified", "sba approved", "sba-prequalified") {
		score += 3
	} else {
		score += 1
	}
	return clamp(score, 0, 20)
}

func recurringScore(b Business, text string) float64 {
	var score float64
	signals := [][]string{
		{"recurring", "contracts", "contracted", "membership", "arr"},
		{"essential", "recession-resistant", "housing always needed"},
		{"route", "routes", "accounts"},
		{"maintenance", "property management", "pest control", "commercial cleaning", "pool service"},
		{"b2b", "managed it", "document storage"},
	}
	for _, group := range signals {
		if containsAny(text, group...) {
			score += 2.3
		}
	}
	switch {
	case b.YearsInBusiness >= 20:
		score += 3.5
	case b.YearsInBusiness >= 10:
		score += 2.5
	case b.YearsInBusiness >= 5:
		score += 1.5
	}
	return clamp(score, 0, 15)
}

func dataConfidenceScore(b Business, m Metrics) float64 {
	var score float64
	if b.AskingPrice > 0 {
		score += 2
	}
	if m.LatestSDE > 0 {
		score += 3
	}
	if m.LatestRevenue > 0 {
		score += 1.5
	}
	if b.URL != "" {
		score += 1
	}
	if b.Location != "" {
		score += 1
	}
	if b.YearsInBusiness > 0 || b.Employees > 0 {
		score += 1
	}
	if b.FFE > 0 || b.Inventory > 0 || b.LeaseMonthly > 0 || knownRealEstate(b.RealEstate) {
		score += 0.5
	}
	return clamp(score, 0, 10)
}

func reasons(b Business, m Metrics, s Score, text string) []string {
	var out []string
	if m.DSCR >= 1.25 {
		out = append(out, "passes SBA DSCR threshold")
	}
	if m.SDEMultiple > 0 && m.SDEMultiple <= 4 {
		out = append(out, "valuation is within preferred SDE multiple")
	}
	if b.AskingPrice > 0 && b.AskingPrice <= 500_000 {
		out = append(out, "asking price fits the target budget")
	}
	if s.AIFit >= 18 {
		out = append(out, "strong AI/IT improvement surface")
	}
	if s.Recurring >= 8 {
		out = append(out, "recurring or defensible revenue signals")
	}
	if containsAny(text, "sba pre-qualified", "sba approved", "sba-prequalified") {
		out = append(out, "SBA pre-qualified signal")
	}
	if len(out) == 0 && b.AIOpportunity != "" {
		out = append(out, "AI/IT angle identified, but financial diligence is still needed")
	}
	return out
}

func warnings(b Business, m Metrics, text string) []string {
	var out []string
	if b.AskingPrice <= 0 {
		out = append(out, "asking price is missing")
	}
	if m.LatestSDE <= 0 {
		out = append(out, "SDE/cash flow is missing")
	}
	if m.LatestRevenue <= 0 {
		out = append(out, "revenue is missing")
	}
	if strings.Contains(text, "estimate") || strings.Contains(text, "estimated") || strings.Contains(text, "midpoint") {
		out = append(out, "some values are estimates from source research")
	}
	if b.AskingPrice > 750_000 {
		out = append(out, "above the preferred purchase-price range")
	}
	if strings.TrimSpace(b.RealEstate) == "unknown" {
		out = append(out, "real-estate/lease status is unknown")
	}
	return out
}

func latestFigure(figures []YearlyFigure) (YearlyFigure, bool) {
	if len(figures) == 0 {
		return YearlyFigure{}, false
	}
	best := figures[0]
	for _, figure := range figures[1:] {
		if figure.Year > best.Year {
			best = figure
		}
	}
	return best, true
}

func monthlyPayment(principal, annualRate float64, termYears int) float64 {
	if principal <= 0 {
		return 0
	}
	monthlyRate := annualRate / 12
	months := float64(termYears * 12)
	if monthlyRate == 0 {
		return principal / months
	}
	return principal * monthlyRate * math.Pow(1+monthlyRate, months) / (math.Pow(1+monthlyRate, months) - 1)
}

func searchableText(b Business) string {
	return strings.ToLower(strings.Join([]string{
		b.Name,
		b.Type,
		b.Location,
		b.URL,
		b.ReasonForSelling,
		b.AIOpportunity,
		b.Notes,
	}, " "))
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func knownRealEstate(realEstate string) bool {
	switch strings.ToLower(strings.TrimSpace(realEstate)) {
	case "leased", "owned", "none":
		return true
	default:
		return false
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
