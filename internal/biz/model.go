package biz

import "strings"

type YearlyFigure struct {
	Year   int     `json:"year"`
	Amount float64 `json:"amount"`
}

type Link struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type Links struct {
	Source     string `json:"source,omitempty"`
	Website    string `json:"website,omitempty"`
	GoogleMaps string `json:"google_maps,omitempty"`
	Yelp       string `json:"yelp,omitempty"`
	BBB        string `json:"bbb,omitempty"`
	WebReviews string `json:"web_reviews,omitempty"`
	Reviews    []Link `json:"reviews,omitempty"`
}

type Business struct {
	Name             string         `json:"name"`
	NameEvidence     string         `json:"name_evidence,omitempty"`
	Type             string         `json:"type"`
	Location         string         `json:"location"`
	URL              string         `json:"url,omitempty"`
	Links            Links          `json:"links,omitempty"`
	AskingPrice      float64        `json:"asking_price"`
	Revenue          []YearlyFigure `json:"revenue,omitempty"`
	SDE              []YearlyFigure `json:"sde,omitempty"`
	Inventory        float64        `json:"inventory,omitempty"`
	FFE              float64        `json:"ffe,omitempty"`
	RealEstate       string         `json:"real_estate,omitempty"`
	LeaseMonthly     float64        `json:"lease_monthly,omitempty"`
	LeaseExpiresYear int            `json:"lease_expires_year,omitempty"`
	YearsInBusiness  int            `json:"years_in_business,omitempty"`
	Employees        int            `json:"employees,omitempty"`
	ReasonForSelling string         `json:"reason_for_selling,omitempty"`
	AIOpportunity    string         `json:"ai_opportunity,omitempty"`
	Notes            string         `json:"notes,omitempty"`
}

type Candidate struct {
	Slug string
	Path string
	Biz  Business
}

func HasVerifiedSource(b Business) bool {
	return strings.TrimSpace(b.Links.Source) != ""
}

func HasKnownRealEstate(realEstate string) bool {
	switch strings.ToLower(strings.TrimSpace(realEstate)) {
	case "leased", "owned", "none":
		return true
	default:
		return false
	}
}
