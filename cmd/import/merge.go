package main

import (
	"strings"

	core "github.com/igor-nav/biz/internal/biz"
)

type extraction struct {
	Source   string
	Business core.Business
}

func baseExtraction(rawURL string) extraction {
	return extraction{
		Source: "request-url",
		Business: core.Business{
			URL:   rawURL,
			Links: core.Links{Source: rawURL},
		},
	}
}

func mergeExtractions(parts ...extraction) core.Business {
	var out core.Business
	for _, part := range parts {
		out = mergeBusiness(out, part.Business)
	}
	return out
}

func mergeBusiness(dst, src core.Business) core.Business {
	dst.Name = firstString(dst.Name, src.Name)
	dst.Type = firstString(dst.Type, src.Type)
	dst.Location = firstString(dst.Location, src.Location)
	dst.URL = firstString(dst.URL, src.URL)
	dst.Links = mergeLinks(dst.Links, src.Links)
	dst.AskingPrice = firstFloat(dst.AskingPrice, src.AskingPrice)
	dst.Revenue = firstFigures(dst.Revenue, src.Revenue)
	dst.SDE = firstFigures(dst.SDE, src.SDE)
	dst.Inventory = firstFloat(dst.Inventory, src.Inventory)
	dst.FFE = firstFloat(dst.FFE, src.FFE)
	dst.RealEstate = firstString(dst.RealEstate, src.RealEstate)
	dst.LeaseMonthly = firstFloat(dst.LeaseMonthly, src.LeaseMonthly)
	dst.LeaseExpiresYear = firstInt(dst.LeaseExpiresYear, src.LeaseExpiresYear)
	dst.YearsInBusiness = firstInt(dst.YearsInBusiness, src.YearsInBusiness)
	dst.Employees = firstInt(dst.Employees, src.Employees)
	dst.ReasonForSelling = firstString(dst.ReasonForSelling, src.ReasonForSelling)
	dst.AIOpportunity = firstString(dst.AIOpportunity, src.AIOpportunity)
	dst.Notes = firstString(dst.Notes, src.Notes)
	return dst
}

func mergeLinks(dst, src core.Links) core.Links {
	dst.Source = firstString(dst.Source, src.Source)
	dst.Website = firstString(dst.Website, src.Website)
	dst.GoogleMaps = firstString(dst.GoogleMaps, src.GoogleMaps)
	dst.Yelp = firstString(dst.Yelp, src.Yelp)
	dst.BBB = firstString(dst.BBB, src.BBB)
	dst.WebReviews = firstString(dst.WebReviews, src.WebReviews)
	if len(dst.Reviews) == 0 && len(src.Reviews) > 0 {
		dst.Reviews = append([]core.Link(nil), src.Reviews...)
	}
	return dst
}

func firstString(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return strings.TrimSpace(b)
}

func firstFloat(a, b float64) float64 {
	if a != 0 {
		return a
	}
	return b
}

func firstInt(a, b int) int {
	if a != 0 {
		return a
	}
	return b
}

func firstFigures(a, b []core.YearlyFigure) []core.YearlyFigure {
	if len(a) != 0 {
		return a
	}
	if len(b) == 0 {
		return nil
	}
	return append([]core.YearlyFigure(nil), b...)
}
