package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	core "github.com/igor-nav/biz/internal/biz"
)

// genericSiteProvider handles broker/listing sites that expose useful fields in
// ordinary HTML instead of a site-specific structured data model.
type genericSiteProvider struct {
	siteName        string
	slugPrefix      string
	hosts           []string
	searchPathHints []string
}

func (p *genericSiteProvider) Supports(u *url.URL) bool {
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	for _, supported := range p.hosts {
		supported = strings.TrimPrefix(strings.ToLower(supported), "www.")
		if host == supported {
			return true
		}
	}
	return false
}

func (p *genericSiteProvider) Fetch(rawURL string) (*core.Business, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", fmt.Errorf("parsing URL: %w", err)
	}
	if p.isSearchPage(u) {
		return nil, "", fmt.Errorf("%s search/results pages are not importable; use a detail listing URL or create the candidate manually", p.siteName)
	}

	body, err := fetchPage(rawURL)
	if err != nil {
		return nil, "", err
	}

	biz := mergeExtractions(
		baseExtraction(rawURL),
		extraction{Source: "json-ld", Business: parseJSONLD(body)},
		extraction{Source: "html", Business: parseGenericHTML(body, p.siteName)},
	)

	hasFinancials := biz.AskingPrice > 0 || len(biz.SDE) > 0 || len(biz.Revenue) > 0
	if biz.Name == "" || !hasFinancials {
		return nil, "", fmt.Errorf("could not extract listing fields from %s; use a detail page instead of a search/results page", p.siteName)
	}

	return &biz, genericSlug(rawURL, p.slugPrefix, biz.Name), nil
}

func (p *genericSiteProvider) isSearchPage(u *url.URL) bool {
	path := strings.ToLower(strings.Trim(u.Path, "/"))
	for _, hint := range p.searchPathHints {
		if strings.Contains(path, strings.ToLower(strings.Trim(hint, "/"))) {
			return true
		}
	}
	return false
}

var (
	reHTMLTitle = regexp.MustCompile(`(?is)<title[^>]*>\s*(.*?)\s*</title>`)

	reGenericPrice = regexp.MustCompile(`(?i)(?:asking\s*price|listing\s*price|sale\s*price|price)[^$0-9]{0,120}\$?\s*([\d,.]+(?:\s*(?:-|to)\s*\$?\s*[\d,.]+)?\s*[kKmM]?\+?)`)
	reGenericSDE   = regexp.MustCompile(`(?i)(?:cash\s*flow|seller'?s?\s*discretionary\s*earnings|sde|owner\s*benefit|net\s*profit)[^$0-9]{0,120}\$?\s*([\d,.]+(?:\s*(?:-|to)\s*\$?\s*[\d,.]+)?\s*[kKmM]?\+?)`)
	reGenericRev   = regexp.MustCompile(`(?i)(?:gross\s*revenue|annual\s*revenue|revenue)[^$0-9]{0,120}\$?\s*([\d,.]+(?:\s*(?:-|to)\s*\$?\s*[\d,.]+)?\s*[kKmM]?\+?)`)
	reGenericLoc   = regexp.MustCompile(`(?i)(?:location|located\s+in)\s*:?\s*([A-Z][A-Za-z .'-]+,\s*[A-Z]{2}\b)`)
)

func parseGenericHTML(body string, siteName string) core.Business {
	var biz core.Business
	stripped := reScriptStyle.ReplaceAllString(body, " ")
	text := reHTMLTag.ReplaceAllString(stripped, " ")
	text = htmlUnescape(reExtraSpace.ReplaceAllString(text, " "))

	if biz.Name == "" {
		biz.Name = genericTitle(body, stripped, siteName)
	}
	if biz.AskingPrice == 0 {
		biz.AskingPrice = firstMoney(text, reGenericPrice)
	}

	year := bestFinancialYear(text)
	if len(biz.SDE) == 0 {
		if sde := firstMoney(text, reGenericSDE); sde > 0 {
			biz.SDE = []core.YearlyFigure{{Year: year, Amount: sde}}
		}
	}
	if len(biz.Revenue) == 0 {
		if revenue := firstMoney(text, reGenericRev); revenue > 0 {
			biz.Revenue = []core.YearlyFigure{{Year: year, Amount: revenue}}
		}
	}
	if biz.Location == "" {
		if m := reGenericLoc.FindStringSubmatch(text); len(m) > 1 {
			biz.Location = strings.TrimSpace(m[1])
		}
	}
	return biz
}

func genericTitle(body, stripped, siteName string) string {
	var title string
	if m := reTitleOG.FindStringSubmatch(body); len(m) > 0 {
		for _, g := range m[1:] {
			if g = strings.TrimSpace(g); g != "" {
				title = htmlUnescape(g)
				break
			}
		}
	}
	if title == "" {
		if m := reTitleH1.FindStringSubmatch(stripped); len(m) > 1 {
			title = htmlUnescape(strings.TrimSpace(m[1]))
		}
	}
	if title == "" {
		if m := reHTMLTitle.FindStringSubmatch(stripped); len(m) > 1 {
			title = htmlUnescape(strings.TrimSpace(m[1]))
		}
	}
	return cleanGenericTitle(title, siteName)
}

func cleanGenericTitle(title, siteName string) string {
	title = strings.TrimSpace(title)
	for _, sep := range []string{" | ", " - "} {
		if before, after, ok := strings.Cut(title, sep); ok && strings.Contains(strings.ToLower(after), strings.ToLower(siteName)) {
			return strings.TrimSpace(before)
		}
	}
	return title
}

func firstMoney(text string, re *regexp.Regexp) float64 {
	if m := re.FindStringSubmatch(text); len(m) > 1 {
		return parseDollar(m[1])
	}
	return 0
}

func genericSlug(rawURL, prefix, name string) string {
	if name != "" {
		return prefix + "-" + slugify(name)
	}
	if u, err := url.Parse(rawURL); err == nil {
		path := strings.Trim(u.Path, "/")
		if path != "" {
			return prefix + "-" + slugify(path)
		}
	}
	return fmt.Sprintf("%s-import-%d", prefix, time.Now().Unix())
}
