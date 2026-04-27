package main

import (
	"strings"

	core "github.com/igor-nav/biz/internal/biz"
)

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

func ScoreBusiness(b core.Business, m core.Metrics) Score {
	text := searchableText(b)
	s := Score{
		Financial:      financialScore(m),
		AIFit:          aiFitScore(text),
		Budget:         budgetScore(b, m, text),
		Recurring:      recurringScore(b, text),
		DataConfidence: dataConfidenceScore(b, m),
	}
	s.Reasons = reasons(b, m, s, text)
	s.Warnings = warnings(b, m, text)
	s.Total = clamp(s.Financial+s.AIFit+s.Budget+s.Recurring+s.DataConfidence, 0, 100)
	return s
}

type compareOp int

const (
	atLeast compareOp = iota
	greaterThan
	atMost
	atMostPositive
)

type thresholdRule struct {
	Op     compareOp
	Value  float64
	Points float64
}

type keywordRule struct {
	Needles []string
	Points  float64
}

type factRule struct {
	Points float64
	Match  func(core.Business, core.Metrics) bool
}

var (
	dscrRules = []thresholdRule{
		{Op: atLeast, Value: 2.0, Points: 12},
		{Op: atLeast, Value: 1.5, Points: 10},
		{Op: atLeast, Value: 1.25, Points: 7},
		{Op: greaterThan, Value: 0, Points: 3},
	}
	sdeMultipleRules = []thresholdRule{
		{Op: atMostPositive, Value: 2.5, Points: 8},
		{Op: atMost, Value: 3.5, Points: 6},
		{Op: atMost, Value: 4, Points: 4},
		{Op: atMost, Value: 5, Points: 2},
	}
	sdeMarginRules = []thresholdRule{
		{Op: atLeast, Value: 0.30, Points: 4},
		{Op: atLeast, Value: 0.20, Points: 3},
		{Op: greaterThan, Value: 0, Points: 1.5},
	}
	paybackRules = []thresholdRule{
		{Op: atMostPositive, Value: 0.5, Points: 6},
		{Op: atMost, Value: 1.5, Points: 4},
		{Op: atMost, Value: 3, Points: 2},
	}
	askingPriceRules = []thresholdRule{
		{Op: atMost, Value: 500_000, Points: 12},
		{Op: atMost, Value: 750_000, Points: 9},
		{Op: atMost, Value: 1_000_000, Points: 6},
		{Op: atMost, Value: 1_500_000, Points: 3},
	}
	downPaymentRules = []thresholdRule{
		{Op: atMost, Value: 100_000, Points: 5},
		{Op: atMost, Value: 250_000, Points: 4},
	}
	yearsInBusinessRules = []thresholdRule{
		{Op: atLeast, Value: 20, Points: 3.5},
		{Op: atLeast, Value: 10, Points: 2.5},
		{Op: atLeast, Value: 5, Points: 1.5},
	}
	aiKeywordRules = []keywordRule{
		{Needles: []string{"ai ", " ai", "artificial intelligence", "machine learning"}, Points: 2.2},
		{Needles: []string{"automation", "automated", "workflow"}, Points: 2.2},
		{Needles: []string{"predictive", "forecast", "forecasting"}, Points: 2.2},
		{Needles: []string{"route", "routing", "scheduling"}, Points: 2.2},
		{Needles: []string{"iot", "sensor", "monitoring", "remote"}, Points: 2.2},
		{Needles: []string{"crm", "portal", "customer retention"}, Points: 2.2},
		{Needles: []string{"saas", "software", "platform", "e-commerce", "ecom"}, Points: 2.2},
		{Needles: []string{"diagnostic", "cybersecurity", "ticket"}, Points: 2.2},
		{Needles: []string{"dynamic pricing", "bidding", "lead scoring", "upsell"}, Points: 2.2},
	}
	aiCategoryRules = []keywordRule{
		{Needles: []string{"property management", "pest control", "computer repair", "telehealth", "streaming platform", "commercial cleaning", "managed it"}, Points: 3},
	}
	recurringKeywordRules = []keywordRule{
		{Needles: []string{"recurring", "contracts", "contracted", "membership", "arr"}, Points: 2.3},
		{Needles: []string{"essential", "recession-resistant", "housing always needed"}, Points: 2.3},
		{Needles: []string{"route", "routes", "accounts"}, Points: 2.3},
		{Needles: []string{"maintenance", "property management", "pest control", "commercial cleaning", "pool service"}, Points: 2.3},
		{Needles: []string{"b2b", "managed it", "document storage"}, Points: 2.3},
	}
	dataConfidenceRules = []factRule{
		{Points: 2, Match: func(b core.Business, _ core.Metrics) bool { return b.AskingPrice > 0 }},
		{Points: 3, Match: func(_ core.Business, m core.Metrics) bool { return m.LatestSDE > 0 }},
		{Points: 1.5, Match: func(_ core.Business, m core.Metrics) bool { return m.LatestRevenue > 0 }},
		{Points: 1, Match: func(b core.Business, _ core.Metrics) bool { return core.HasVerifiedSource(b) }},
		{Points: 1, Match: func(b core.Business, _ core.Metrics) bool { return b.Location != "" }},
		{Points: 1, Match: func(b core.Business, _ core.Metrics) bool { return b.YearsInBusiness > 0 || b.Employees > 0 }},
		{Points: 0.5, Match: func(b core.Business, _ core.Metrics) bool {
			return b.FFE > 0 || b.Inventory > 0 || b.LeaseMonthly > 0 || core.HasKnownRealEstate(b.RealEstate)
		}},
	}
)

func financialScore(m core.Metrics) float64 {
	if m.LatestSDE <= 0 {
		return 3
	}
	score := firstThresholdScore(m.DSCR, dscrRules)
	score += firstThresholdScore(m.SDEMultiple, sdeMultipleRules)
	score += firstThresholdScore(m.SDEMargin, sdeMarginRules)
	score += firstThresholdScore(m.PaybackYears, paybackRules)
	return clamp(score, 0, 30)
}

func aiFitScore(text string) float64 {
	if strings.TrimSpace(text) == "" {
		return 0
	}
	score := 5.0
	score += keywordScore(text, aiKeywordRules)
	score += keywordScore(text, aiCategoryRules)
	return clamp(score, 0, 25)
}

func budgetScore(b core.Business, m core.Metrics, text string) float64 {
	if b.AskingPrice <= 0 {
		return 3
	}
	score := firstThresholdScore(b.AskingPrice, askingPriceRules)
	score += firstThresholdScore(m.DownPayment, downPaymentRules)
	if containsAny(text, "sba pre-qualified", "sba approved", "sba-prequalified") {
		score += 3
	} else {
		score += 1
	}
	return clamp(score, 0, 20)
}

func recurringScore(b core.Business, text string) float64 {
	score := keywordScore(text, recurringKeywordRules)
	score += firstThresholdScore(float64(b.YearsInBusiness), yearsInBusinessRules)
	return clamp(score, 0, 15)
}

func dataConfidenceScore(b core.Business, m core.Metrics) float64 {
	return clamp(factScore(b, m, dataConfidenceRules), 0, 10)
}

func reasons(b core.Business, m core.Metrics, s Score, text string) []string {
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

func warnings(b core.Business, m core.Metrics, text string) []string {
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

func firstThresholdScore(value float64, rules []thresholdRule) float64 {
	for _, rule := range rules {
		if thresholdMatches(value, rule) {
			return rule.Points
		}
	}
	return 0
}

func thresholdMatches(value float64, rule thresholdRule) bool {
	switch rule.Op {
	case atLeast:
		return value >= rule.Value
	case greaterThan:
		return value > rule.Value
	case atMost:
		return value <= rule.Value
	case atMostPositive:
		return value > 0 && value <= rule.Value
	default:
		return false
	}
}

func keywordScore(text string, rules []keywordRule) float64 {
	var score float64
	for _, rule := range rules {
		if containsAny(text, rule.Needles...) {
			score += rule.Points
		}
	}
	return score
}

func factScore(b core.Business, m core.Metrics, rules []factRule) float64 {
	var score float64
	for _, rule := range rules {
		if rule.Match(b, m) {
			score += rule.Points
		}
	}
	return score
}

func searchableText(b core.Business) string {
	parts := []string{
		b.Name,
		b.Type,
		b.Location,
		b.URL,
		b.Links.Source,
		b.Links.Website,
		b.ReasonForSelling,
		b.AIOpportunity,
		b.Notes,
	}
	for _, review := range b.Links.Reviews {
		parts = append(parts, review.Label, review.URL)
	}
	return strings.ToLower(strings.Join(parts, " "))
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, strings.ToLower(needle)) {
			return true
		}
	}
	return false
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
