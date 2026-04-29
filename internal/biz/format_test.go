package biz

import "testing"

func TestFormatUSD_Zero(t *testing.T) {
	if got := FormatUSD(0); got != "TBD" {
		t.Fatalf("got %q, want TBD", got)
	}
}

func TestFormatUSD_Negative(t *testing.T) {
	if got := FormatUSD(-100); got != "TBD" {
		t.Fatalf("got %q, want TBD", got)
	}
}

func TestFormatUSD_Small(t *testing.T) {
	if got := FormatUSD(500); got != "$500" {
		t.Fatalf("got %q, want $500", got)
	}
}

func TestFormatUSD_Thousands(t *testing.T) {
	if got := FormatUSD(1250); got != "$1,250" {
		t.Fatalf("got %q, want $1,250", got)
	}
}

func TestFormatUSD_Millions(t *testing.T) {
	if got := FormatUSD(1250000); got != "$1,250,000" {
		t.Fatalf("got %q, want $1,250,000", got)
	}
}

func TestFormatUSD_Rounds(t *testing.T) {
	if got := FormatUSD(999.6); got != "$1,000" {
		t.Fatalf("got %q, want $1,000", got)
	}
}
