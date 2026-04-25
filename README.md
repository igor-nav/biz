# biz

Research repo for finding a small business to buy.

## Acquisition criteria

| Criterion | Target |
|---|---|
| Own money at risk | ~$100k, hard cap $250k |
| Financing | SBA 7(a) loan (10 % down, 10 yr, ~10.5 %) |
| Business type | AI-augmentable; services or software |
| DSCR | ≥ 1.25 (SBA minimum) |
| SDE multiple | ≤ 4× preferred |

## Repository layout

```
businesses/
  <slug>/          ← one directory per candidate
    data.json      ← financials + notes (schema below)
cmd/
  analyze/
    main.go        ← stats tool
go.mod
```

## Adding a candidate

1. Create `businesses/<slug>/data.json` using the schema below.
2. Run `go run ./cmd/analyze` to see updated stats.

### data.json schema

```jsonc
{
  "name": "Acme IT Services",          // display name
  "type": "IT Managed Services / MSP", // business category
  "location": "Austin, TX",
  "url": "https://bizbuysell.com/...", // listing URL (optional)
  "asking_price": 750000,

  // yearly financials – include as many years as available (most recent wins)
  "revenue": [
    {"year": 2023, "amount": 620000},
    {"year": 2022, "amount": 580000}
  ],
  "sde": [                             // Seller's Discretionary Earnings
    {"year": 2023, "amount": 195000},
    {"year": 2022, "amount": 178000}
  ],

  "inventory": 0,                      // current inventory value
  "ffe": 25000,                        // Furniture, Fixtures & Equipment
  "real_estate": "leased",             // "leased" | "owned" | "none"
  "lease_monthly": 2200,               // monthly rent (optional)
  "lease_expires_year": 2027,          // (optional)

  "years_in_business": 12,
  "employees": 6,
  "reason_for_selling": "Owner retiring",
  "ai_opportunity": "...",             // how AI/software expertise adds value
  "notes": "..."
}
```

## Running the analyzer

```sh
# default: 10 % down, 10.50 % rate, 10-year term
go run ./cmd/analyze

# custom parameters
go run ./cmd/analyze -down 20 -rate 11.0 -term 10
```

**Flags**

| Flag | Default | Description |
|---|---|---|
| `-dir` | `businesses` | root directory of candidate sub-dirs |
| `-down` | `10.0` | down payment % |
| `-rate` | `10.50` | SBA loan annual interest rate % |
| `-term` | `10` | loan term in years |

## Stats computed per candidate

| Metric | Description |
|---|---|
| SDE multiple | `asking_price / latest_SDE` – valuation ratio (2–4× is typical) |
| SDE margin | `SDE / revenue` |
| Revenue growth | YoY growth of the two most-recent years |
| SDE growth | YoY growth of the two most-recent years |
| Down payment | `asking_price × down%` |
| Loan amount | `asking_price − down payment` |
| Monthly payment | Fixed-rate amortising SBA loan payment |
| Annual debt service | `monthly_payment × 12` |
| DSCR | `SDE / annual_debt_service` – SBA requires ≥ 1.25 |
| ROI on down payment | `SDE / down_payment` |
| Payback period | `down_payment / SDE` |
