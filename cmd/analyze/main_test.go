package main

import (
	"math"
	"testing"

	core "github.com/igor-nav/biz/internal/biz"
)

func TestComputeStatsUsesLatestFiguresAndAdjacentGrowth(t *testing.T) {
	b := core.Business{
		AskingPrice: 1_000_000,
		Revenue: []core.YearlyFigure{
			{Year: 2022, Amount: 500_000},
			{Year: 2024, Amount: 700_000},
			{Year: 2023, Amount: 600_000},
		},
		SDE: []core.YearlyFigure{
			{Year: 2022, Amount: 150_000},
			{Year: 2024, Amount: 210_000},
			{Year: 2023, Amount: 180_000},
		},
	}

	stats := core.ComputeMetrics(b, core.Terms{DownPct: 0.10, AnnualRate: 0.105, TermYears: 10})

	assertEqual(t, stats.LatestRevenue, 700_000)
	assertEqual(t, stats.LatestSDE, 210_000)
	assertClose(t, stats.RevenueGrowth, (700_000-600_000)/600_000.0)
	assertClose(t, stats.SDEGrowth, (210_000-180_000)/180_000.0)
	assertClose(t, stats.SDEMultiple, 1_000_000/210_000.0)
	assertClose(t, stats.SDEMargin, 210_000/700_000.0)
}

func assertEqual(t *testing.T, got, want float64) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func assertClose(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("got %v, want %v", got, want)
	}
}
