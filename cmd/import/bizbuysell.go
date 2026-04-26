// bizbuysell.go implements the Provider interface for BizBuySell.com listings.
//
// Extraction strategy (applied in order; later passes fill only empty fields):
//  1. __NEXT_DATA__ – the JSON blob Next.js embeds in every page; most reliable.
//  2. JSON-LD       – schema.org structured data in <script type="application/ld+json">.
//  3. HTML patterns – regex fallbacks for when the above are absent or sparse.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// bizBuySellProvider scrapes business-for-sale listings from BizBuySell.com.
type bizBuySellProvider struct{}

// Supports reports whether the URL belongs to BizBuySell.
func (p *bizBuySellProvider) Supports(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	return host == "www.bizbuysell.com" || host == "bizbuysell.com"
}

// Fetch downloads and parses a BizBuySell listing page.
func (p *bizBuySellProvider) Fetch(rawURL string) (*Business, string, error) {
	body, err := fetchPage(rawURL)
	if err != nil {
		return nil, "", err
	}

	biz := &Business{URL: rawURL}

	parseNextData(body, biz) // Next.js embedded JSON (most reliable)
	parseJSONLD(body, biz)   // JSON-LD structured data
	parseBBSHTML(body, biz)  // HTML regex fallbacks

	slug := bizBuySellSlug(rawURL, biz.Name)
	return biz, slug, nil
}

// ── HTTP fetch ────────────────────────────────────────────────────────────────

// fetchPage downloads rawURL and returns the response body as a string.
func fetchPage(rawURL string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	// Mimic a real browser to avoid trivial bot-detection blocks.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, rawURL)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading body: %w", err)
	}
	return string(b), nil
}

// ── Next.js __NEXT_DATA__ extraction ─────────────────────────────────────────

var reNextData = regexp.MustCompile(`(?s)<script[^>]+id=["']__NEXT_DATA__["'][^>]*>(.*?)</script>`)

// parseNextData extracts business data from the Next.js __NEXT_DATA__ JSON blob.
func parseNextData(body string, biz *Business) {
	m := reNextData.FindStringSubmatch(body)
	if len(m) < 2 {
		return
	}

	var nd map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(m[1])), &nd); err != nil {
		return
	}

	listing := findListingMap(nd)
	if listing == nil {
		return
	}

	// Helpers that only write if the field is still empty/zero.
	setStr := func(dst *string, keys ...string) {
		if *dst != "" {
			return
		}
		for _, k := range keys {
			if v, _ := listing[k].(string); strings.TrimSpace(v) != "" {
				*dst = strings.TrimSpace(v)
				return
			}
		}
	}
	setFloat := func(dst *float64, keys ...string) {
		if *dst != 0 {
			return
		}
		for _, k := range keys {
			switch v := listing[k].(type) {
			case float64:
				if v != 0 {
					*dst = v
					return
				}
			case string:
				if f := parseDollar(v); f != 0 {
					*dst = f
					return
				}
			}
		}
	}
	setInt := func(dst *int, keys ...string) {
		if *dst != 0 {
			return
		}
		for _, k := range keys {
			switch v := listing[k].(type) {
			case float64:
				if int(v) != 0 {
					*dst = int(v)
					return
				}
			case string:
				if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n != 0 {
					*dst = n
					return
				}
			}
		}
	}

	setStr(&biz.Name, "businessName", "name", "title")
	setStr(&biz.Type, "businessCategory", "category", "businessType", "industryType")
	setStr(&biz.ReasonForSelling, "reasonForSelling", "sellingReason")
	setStr(&biz.Notes, "description", "listingDescription", "businessDescription")

	setFloat(&biz.AskingPrice, "askingPrice", "listingPrice", "price")
	setFloat(&biz.FFE, "ffe", "furnitureFixturesEquipment")
	setFloat(&biz.Inventory, "inventory", "inventoryValue")
	setFloat(&biz.LeaseMonthly, "leaseMonthly", "monthlyRent", "leaseAmount")

	setInt(&biz.Employees, "employees", "numberOfEmployees", "numEmployees")
	setInt(&biz.YearsInBusiness, "yearsInBusiness", "yearsEstablished")

	// Location (may be a nested object or flat string).
	if biz.Location == "" {
		biz.Location = extractLocationFromMap(listing)
	}

	// Financial figures – use the prior calendar year as a default.
	year := time.Now().Year() - 1
	if len(biz.SDE) == 0 {
		var cf float64
		setFloat(&cf, "cashFlow", "sde", "sellerDiscretionaryEarnings")
		if cf > 0 {
			biz.SDE = []YearlyFigure{{Year: year, Amount: cf}}
		}
	}
	if len(biz.Revenue) == 0 {
		var rev float64
		setFloat(&rev, "grossRevenue", "revenue", "annualRevenue")
		if rev > 0 {
			biz.Revenue = []YearlyFigure{{Year: year, Amount: rev}}
		}
	}

	// Established year → years in business.
	if biz.YearsInBusiness == 0 {
		switch v := listing["established"].(type) {
		case float64:
			if est := int(v); est > 1900 {
				biz.YearsInBusiness = time.Now().Year() - est
			}
		case string:
			if est, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && est > 1900 {
				biz.YearsInBusiness = time.Now().Year() - est
			}
		}
	}

	// Real estate.
	if biz.RealEstate == "" {
		if v, _ := listing["realEstate"].(string); v != "" {
			biz.RealEstate = strings.ToLower(strings.TrimSpace(v))
		}
	}
}

// findListingMap walks common Next.js pageProps paths to find the listing object.
// It returns the first map that contains a recognisable business field.
func findListingMap(nd map[string]any) map[string]any {
	// Candidate key sequences to traverse.
	paths := [][]string{
		{"props", "pageProps", "listing"},
		{"props", "pageProps", "business"},
		{"props", "pageProps", "listingDetail"},
		{"props", "pageProps", "data", "listing"},
		{"props", "pageProps"},
	}
	indicators := []string{"askingPrice", "businessName", "cashFlow", "grossRevenue", "listingPrice"}

	for _, path := range paths {
		cur := nd
		for _, seg := range path {
			next, ok := cur[seg].(map[string]any)
			if !ok {
				cur = nil
				break
			}
			cur = next
		}
		if cur == nil {
			continue
		}
		for _, ind := range indicators {
			if _, ok := cur[ind]; ok {
				return cur
			}
		}
	}
	return nil
}

// extractLocationFromMap builds a "City, State" string from a listing map.
func extractLocationFromMap(listing map[string]any) string {
	if loc, ok := listing["location"].(map[string]any); ok {
		city, _ := loc["city"].(string)
		state, _ := loc["state"].(string)
		city = strings.TrimSpace(city)
		state = strings.TrimSpace(state)
		if city != "" && state != "" {
			return city + ", " + state
		}
		if city != "" {
			return city
		}
	}
	for _, key := range []string{"city", "location", "address", "businessLocation"} {
		if v, _ := listing[key].(string); strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// ── JSON-LD extraction ────────────────────────────────────────────────────────

var reJSONLD = regexp.MustCompile(`(?is)<script[^>]+type=["']application/ld\+json["'][^>]*>(.*?)</script>`)

// parseJSONLD looks for JSON-LD script tags and fills any still-empty fields.
func parseJSONLD(body string, biz *Business) {
	for _, m := range reJSONLD.FindAllStringSubmatch(body, -1) {
		var obj map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSpace(m[1])), &obj); err != nil {
			continue
		}

		if biz.Name == "" {
			if name, _ := obj["name"].(string); strings.TrimSpace(name) != "" {
				biz.Name = strings.TrimSpace(name)
			}
		}
		if biz.Type == "" {
			if cat, _ := obj["category"].(string); cat != "" {
				biz.Type = strings.TrimSpace(cat)
			}
		}
		if biz.AskingPrice == 0 {
			if offers, ok := obj["offers"].(map[string]any); ok {
				biz.AskingPrice = jsonFloat(offers, "price")
			}
		}
		if biz.Notes == "" {
			if desc, _ := obj["description"].(string); strings.TrimSpace(desc) != "" {
				biz.Notes = strings.TrimSpace(desc)
			}
		}
	}
}

// jsonFloat extracts a number from a JSON map value, accepting both float64 and string.
func jsonFloat(obj map[string]any, key string) float64 {
	switch v := obj[key].(type) {
	case float64:
		return v
	case string:
		return parseDollar(v)
	}
	return 0
}

// ── HTML pattern extraction ───────────────────────────────────────────────────

var (
	// Title
	reTitleOG = regexp.MustCompile(`(?i)(?:<meta[^>]+property=["']og:title["'][^>]+content=["']([^"']+)["']|<meta[^>]+content=["']([^"']+)["'][^>]+property=["']og:title["'])`)
	reTitleH1 = regexp.MustCompile(`(?i)<h1[^>]*>\s*([^<]+?)\s*</h1>`)

	// Financial highlights – match "Label ... $X,XXX" allowing for intervening HTML.
	reAskingPrice = regexp.MustCompile(`(?i)asking\s*price[^$\d]{0,300}\$([\d,]+)`)
	reCashFlow    = regexp.MustCompile(`(?i)cash\s*flow[^$\d]{0,300}\$([\d,]+)`)
	reRevenue     = regexp.MustCompile(`(?i)gross\s*revenue[^$\d]{0,300}\$([\d,]+)`)

	// Location in plain text (after stripping tags).
	reLocationText = regexp.MustCompile(`(?i)location\s*:?\s+([A-Z][A-Za-z\s]+,\s*[A-Z]{2}\b)`)

	// Employees / years / type
	reEmployees   = regexp.MustCompile(`(?i)(\d+)\s+(?:full[- ]?time\s+)?employees`)
	reEstablished = regexp.MustCompile(`(?i)established[^:]{0,10}:\s*(\d{4})`)
	reYearsInBiz  = regexp.MustCompile(`(?i)(\d+)\s+years?\s+in\s+business`)
	reBizType     = regexp.MustCompile(`(?i)(?:type\s+of\s+business|business\s+type)[^:]{0,20}:\s*([^\n<]{3,80})`)

	// Strip tags / compress whitespace
	reScriptStyle = regexp.MustCompile(`(?is)<(?:script|style)[^>]*>.*?</(?:script|style)>`)
	reHTMLTag     = regexp.MustCompile(`<[^>]+>`)
	reExtraSpace  = regexp.MustCompile(`\s{2,}`)

	// Year adjacent to a financial keyword.
	reNearYear = regexp.MustCompile(`(?i)(20\d{2})\s+(?:cash\s*flow|gross\s*revenue|sde)|(?:cash\s*flow|gross\s*revenue|sde)\s*[^.]{0,30}?\(?(20\d{2})\)?`)
)

// parseBBSHTML fills still-empty Business fields using HTML regex patterns.
func parseBBSHTML(body string, biz *Business) {
	// Remove scripts and styles so their text doesn't interfere.
	stripped := reScriptStyle.ReplaceAllString(body, " ")

	// Plain text (all tags removed) for numeric/keyword matching.
	text := reHTMLTag.ReplaceAllString(stripped, " ")
	text = reExtraSpace.ReplaceAllString(text, " ")

	// Title
	if biz.Name == "" {
		if m := reTitleOG.FindStringSubmatch(body); len(m) > 0 {
			for _, g := range m[1:] {
				if g = strings.TrimSpace(g); g != "" {
					biz.Name = htmlUnescape(g)
					break
				}
			}
		}
		if biz.Name == "" {
			if m := reTitleH1.FindStringSubmatch(stripped); len(m) > 1 {
				biz.Name = htmlUnescape(strings.TrimSpace(m[1]))
			}
		}
	}

	// Asking price
	if biz.AskingPrice == 0 {
		if m := reAskingPrice.FindStringSubmatch(text); len(m) > 1 {
			biz.AskingPrice = parseDollar(m[1])
		}
	}

	// Determine the best year to tag financial figures.
	year := bestFinancialYear(text)

	// SDE (BizBuySell labels it "Cash Flow")
	if len(biz.SDE) == 0 {
		if m := reCashFlow.FindStringSubmatch(text); len(m) > 1 {
			if amt := parseDollar(m[1]); amt > 0 {
				biz.SDE = []YearlyFigure{{Year: year, Amount: amt}}
			}
		}
	}

	// Revenue
	if len(biz.Revenue) == 0 {
		if m := reRevenue.FindStringSubmatch(text); len(m) > 1 {
			if amt := parseDollar(m[1]); amt > 0 {
				biz.Revenue = []YearlyFigure{{Year: year, Amount: amt}}
			}
		}
	}

	// Location
	if biz.Location == "" {
		if m := reLocationText.FindStringSubmatch(text); len(m) > 1 {
			biz.Location = strings.TrimSpace(m[1])
		}
	}

	// Employees
	if biz.Employees == 0 {
		if m := reEmployees.FindStringSubmatch(text); len(m) > 1 {
			biz.Employees, _ = strconv.Atoi(m[1])
		}
	}

	// Years in business
	if biz.YearsInBusiness == 0 {
		if m := reEstablished.FindStringSubmatch(text); len(m) > 1 {
			if est, _ := strconv.Atoi(m[1]); est > 1900 {
				biz.YearsInBusiness = time.Now().Year() - est
			}
		}
		if biz.YearsInBusiness == 0 {
			if m := reYearsInBiz.FindStringSubmatch(text); len(m) > 1 {
				biz.YearsInBusiness, _ = strconv.Atoi(m[1])
			}
		}
	}

	// Business type
	if biz.Type == "" {
		if m := reBizType.FindStringSubmatch(text); len(m) > 1 {
			biz.Type = strings.TrimSpace(m[1])
		}
	}
}

// bestFinancialYear finds the most recent year (20xx) mentioned near a financial
// keyword in the page text. Defaults to the previous calendar year.
func bestFinancialYear(text string) int {
	if m := reNearYear.FindStringSubmatch(text); len(m) > 0 {
		for _, s := range m[1:] {
			if len(s) == 4 {
				if y, err := strconv.Atoi(s); err == nil && y >= 2000 && y <= time.Now().Year() {
					return y
				}
			}
		}
	}
	return time.Now().Year() - 1
}

// ── helpers ───────────────────────────────────────────────────────────────────

// parseDollar converts "$1,250,000", "$39.5K", or "$400-600K" to float64.
// Ranges return their midpoint so estimated listing ranges remain sortable.
func parseDollar(s string) float64 {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer(
		"$", "",
		",", "",
		"~", "",
		"estimated", "",
		"est.", "",
		"est", "",
		"\u2013", "-",
		"\u2014", "-",
	)
	s = strings.TrimSpace(replacer.Replace(s))
	s = strings.ReplaceAll(s, " to ", "-")

	if strings.Contains(s, "-") {
		parts := strings.SplitN(s, "-", 2)
		suffix := moneySuffix(parts[1])
		low := parseSingleDollar(parts[0], suffix)
		high := parseSingleDollar(parts[1], suffix)
		if low > 0 && high > 0 {
			return (low + high) / 2
		}
	}

	return parseSingleDollar(s, "")
}

func parseSingleDollar(s, fallbackSuffix string) float64 {
	s = strings.Trim(strings.TrimSpace(s), " +")
	suffix := moneySuffix(s)
	if suffix == "" {
		suffix = fallbackSuffix
	} else {
		s = strings.TrimSpace(s[:len(s)-1])
	}

	v, _ := strconv.ParseFloat(s, 64)
	switch suffix {
	case "k":
		return v * 1_000
	case "m":
		return v * 1_000_000
	default:
		return v
	}
}

func moneySuffix(s string) string {
	s = strings.Trim(strings.TrimSpace(s), " +")
	if s == "" {
		return ""
	}
	switch s[len(s)-1] {
	case 'k', 'm':
		return string(s[len(s)-1])
	default:
		return ""
	}
}

// htmlUnescape replaces common HTML entities.
func htmlUnescape(s string) string {
	r := strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
		"&#x27;", "'",
		"&nbsp;", " ",
	)
	return r.Replace(s)
}

// ── slug generation ───────────────────────────────────────────────────────────

// bizBuySellSlug derives a filesystem-safe slug from the URL.
// BizBuySell listing URLs follow: /Business-Opportunity/<name-slug>/<id>/
func bizBuySellSlug(rawURL, name string) string {
	if u, err := url.Parse(rawURL); err == nil {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		// Expect at least: ["Business-Opportunity", "<name-slug>", "<id>"]
		if len(parts) >= 3 {
			nameSlug := strings.ToLower(parts[len(parts)-2])
			id := parts[len(parts)-1]
			if id != "" && nameSlug != "" {
				return nameSlug + "-" + id
			}
			if id != "" {
				return "bizbuysell-" + id
			}
		}
		if len(parts) >= 2 {
			if last := parts[len(parts)-1]; last != "" {
				return "bizbuysell-" + last
			}
		}
	}
	if name != "" {
		return slugify(name)
	}
	return "bizbuysell-import"
}

// slugify converts a display name to a lowercase hyphenated slug.
func slugify(s string) string {
	var b strings.Builder
	prevDash := true
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
