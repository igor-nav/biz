# biz

Research repo for finding a small business to buy.

Current ranked research report: [REPORT.md](REPORT.md)

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
research/
  *.md             ← supporting market, lender, broker, and diligence notes
cmd/
  analyze/
    main.go        ← stats tool
  import/
    main.go        ← import a listing from a URL
    bizbuysell.go  ← BizBuySell provider
    generic_listing.go ← shared broker-page parser
  report/
    main.go        ← generate REPORT.md
    score.go       ← heuristic quality scoring rules
go.mod
```

## Adding a candidate

### Option A – import from a listing URL (recommended)

```sh
go run ./cmd/import <URL>
```

The script fetches the listing, extracts as many fields as possible, and writes
`businesses/<slug>/data.json`.  Review the file afterwards and fill in any
fields that could not be scraped automatically (e.g. multi-year financials,
`ai_opportunity`, `notes`).

**Supported providers**

| Provider | URL notes |
|---|---|
| BizBuySell | `https://www.bizbuysell.com/Business-Opportunity/…/1234567/` |
| BizQuest | Detail listing pages on `bizquest.com`; search/results pages are rejected |
| BusinessMart | Detail listing pages on `businessmart.com`; search/results pages are rejected |
| Truforte | Detail listing pages on `trufortebusinessgroup.com`; search/results pages are rejected |
| KMF Business Advisors | Detail listing pages on `kmfbusinessadvisors.com`; search/results pages are rejected |

**Flags**

| Flag | Default | Description |
|---|---|---|
| `-dir` | `businesses` | root directory for candidate sub-dirs |

### Option B – create manually

1. Create `businesses/<slug>/data.json` using the schema below.
2. Run `go run ./cmd/analyze` to see updated stats.

### data.json schema

```jsonc
{
  "name": "Acme IT Services",          // display name
  "type": "IT Managed Services / MSP", // business category
  "location": "Austin, TX",
  "url": "https://bizbuysell.com/...", // listing URL (optional)
  "links": {                           // exact diligence links for REPORT.md; omit unknowns
    "source": "https://bizbuysell.com/...",
    "website": "https://acme.example.com/location",
    "google_maps": "https://www.google.com/maps/search/?api=1&query=Acme%20IT%20Services%20Austin%20TX",
    "yelp": "https://www.yelp.com/biz/acme-it-services-austin",
    "bbb": "https://www.bbb.org/us/tx/austin/profile/...",
    "reviews": [
      {"label": "Birdeye", "url": "https://reviews.example.com/acme-it-services"}
    ]
  },
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

## Generating the report

```sh
go run ./cmd/report
```

This rewrites `REPORT.md` with the current candidate list ordered by decreasing
quality score.
