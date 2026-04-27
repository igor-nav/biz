package biz

import "math"

type Terms struct {
	DownPct    float64
	AnnualRate float64
	TermYears  int
}

type Metrics struct {
	LatestSDE         float64
	LatestRevenue     float64
	SDEMultiple       float64
	SDEMargin         float64
	RevenueGrowth     float64
	SDEGrowth         float64
	DownPayment       float64
	LoanAmount        float64
	MonthlyPayment    float64
	AnnualDebtService float64
	DSCR              float64
	ROI               float64
	PaybackYears      float64
}

func ComputeMetrics(b Business, terms Terms) Metrics {
	var m Metrics
	if latestSDE, ok := LatestFigure(b.SDE); ok {
		m.LatestSDE = latestSDE.Amount
		if prev, ok := PreviousFigure(b.SDE, latestSDE); ok && prev.Amount > 0 {
			m.SDEGrowth = (latestSDE.Amount - prev.Amount) / prev.Amount
		}
	}
	if latestRevenue, ok := LatestFigure(b.Revenue); ok {
		m.LatestRevenue = latestRevenue.Amount
		if prev, ok := PreviousFigure(b.Revenue, latestRevenue); ok && prev.Amount > 0 {
			m.RevenueGrowth = (latestRevenue.Amount - prev.Amount) / prev.Amount
		}
	}

	if m.LatestSDE > 0 {
		m.SDEMultiple = b.AskingPrice / m.LatestSDE
	}
	if m.LatestRevenue > 0 {
		m.SDEMargin = m.LatestSDE / m.LatestRevenue
	}

	m.DownPayment = b.AskingPrice * terms.DownPct
	m.LoanAmount = b.AskingPrice - m.DownPayment
	m.MonthlyPayment = MonthlyPayment(m.LoanAmount, terms.AnnualRate, terms.TermYears)
	m.AnnualDebtService = m.MonthlyPayment * 12

	if m.AnnualDebtService > 0 {
		m.DSCR = m.LatestSDE / m.AnnualDebtService
	}
	if m.DownPayment > 0 && m.LatestSDE > 0 {
		m.ROI = m.LatestSDE / m.DownPayment
		m.PaybackYears = m.DownPayment / m.LatestSDE
	}

	return m
}

func LatestFigure(figures []YearlyFigure) (YearlyFigure, bool) {
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

func PreviousFigure(figures []YearlyFigure, current YearlyFigure) (YearlyFigure, bool) {
	target := current.Year - 1
	for _, figure := range figures {
		if figure.Year == target {
			return figure, true
		}
	}
	return YearlyFigure{}, false
}

func MonthlyPayment(principal, annualRate float64, termYears int) float64 {
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
