package biz

import (
	"math"
	"testing"
)

func assertClose(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestLatestFigure_Empty(t *testing.T) {
	_, ok := LatestFigure(nil)
	if ok {
		t.Fatal("expected false for nil")
	}
}

func TestLatestFigure_Single(t *testing.T) {
	f, ok := LatestFigure([]YearlyFigure{{2024, 100}})
	if !ok || f.Year != 2024 || f.Amount != 100 {
		t.Fatalf("got %v %v", f, ok)
	}
}

func TestLatestFigure_Unordered(t *testing.T) {
	figs := []YearlyFigure{
		{2022, 80}, {2024, 120}, {2023, 100},
	}
	f, ok := LatestFigure(figs)
	if !ok || f.Year != 2024 {
		t.Fatalf("got year %d", f.Year)
	}
}

func TestPriorYearFigure_Found(t *testing.T) {
	figs := []YearlyFigure{
		{2022, 80}, {2023, 100}, {2024, 120},
	}
	f, ok := PriorYearFigure(figs, YearlyFigure{2024, 120})
	if !ok || f.Year != 2023 {
		t.Fatalf("got %v %v", f, ok)
	}
}

func TestPriorYearFigure_Gap(t *testing.T) {
	figs := []YearlyFigure{
		{2022, 80}, {2024, 120},
	}
	_, ok := PriorYearFigure(figs, YearlyFigure{2024, 120})
	if ok {
		t.Fatal("expected false for non-adjacent years")
	}
}

func TestPriorYearFigure_Empty(t *testing.T) {
	_, ok := PriorYearFigure(nil, YearlyFigure{2024, 120})
	if ok {
		t.Fatal("expected false for nil")
	}
}

func TestMonthlyPayment_Zero(t *testing.T) {
	p := MonthlyPayment(0, 0.10, 10)
	if p != 0 {
		t.Fatalf("got %v, want 0", p)
	}
}

func TestMonthlyPayment_ZeroRate(t *testing.T) {
	p := MonthlyPayment(120000, 0, 10)
	assertClose(t, p, 1000)
}

func TestMonthlyPayment_Normal(t *testing.T) {
	p := MonthlyPayment(900000, 0.105, 10)
	if p < 12000 || p > 13000 {
		t.Fatalf("monthly payment %v out of range", p)
	}
}
