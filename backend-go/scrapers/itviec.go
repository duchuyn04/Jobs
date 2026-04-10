package scrapers

import (
	"context"
	"fmt"
	"jobaggregator/helpers"
	"jobaggregator/models"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

// ItviecScraper scrapes jobs from ITviec.
type ItviecScraper struct {
	pw      *playwright.Playwright
	browser playwright.Browser
}

func NewItviecScraper(pw *playwright.Playwright, browser playwright.Browser) *ItviecScraper {
	return &ItviecScraper{pw: pw, browser: browser}
}

func (s *ItviecScraper) SourceName() string { return "ITviec" }

func (s *ItviecScraper) Scrape(ctx context.Context, filter models.SearchFilter) ([]models.JobItem, error) {
	var jobs []models.JobItem
	seenURLs := make(map[string]bool)
	itviecLevels, useServerLevelFilter := resolveItviecLevelParams(filter.Levels)
	appendUnique := func(newJobs []models.JobItem) int {
		added := 0
		for _, j := range newJobs {
			if j.Url != "" && seenURLs[j.Url] {
				continue
			}
			if j.Url != "" {
				seenURLs[j.Url] = true
			}
			jobs = append(jobs, j)
			added++
		}
		return added
	}

	citySlug, cityDisplayName := resolveCitySlug(filter.Locations)

	const maxPages = 10
	const expectedPageSize = 20
	for p := 1; p <= maxPages; p++ {
		select {
		case <-ctx.Done():
			return jobs, nil
		default:
		}

		pageJobs, err := s.loadPageJobs(ctx, filter.Keyword, p, citySlug, cityDisplayName, itviecLevels, useServerLevelFilter)
		if err != nil {
			if p == 1 {
				slog.Warn("ITviec: page 1 failed", "err", err)
				return jobs, nil
			}
			slog.Warn("ITviec: next page failed", "page", p, "err", err)
			break
		}
		if len(pageJobs) == 0 {
			if p == 1 {
				slog.Warn("ITviec: page 1 returned 0 jobs")
			}
			slog.Info("ITviec: no more pages", "page", p)
			break
		}

		added := appendUnique(pageJobs)
		slog.Info("ITviec", "page", p, "page_jobs", len(pageJobs), "added_unique", added)

		if added == 0 {
			break
		}
		if len(pageJobs) < expectedPageSize {
			break
		}
	}

	slog.Info("ITviec scraped", "total", len(jobs), "keyword", filter.Keyword)
	return jobs, nil
}

func (s *ItviecScraper) loadPageJobs(
	ctx context.Context,
	keyword string,
	pageNum int,
	citySlug, cityDisplayName string,
	itviecLevels []string,
	useServerLevelFilter bool,
) ([]models.JobItem, error) {
	// Fresh context per page avoids ITviec 403 on page transitions.
	bCtx, err := s.browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"),
	})
	if err != nil {
		return nil, fmt.Errorf("itviec: new context (page %d): %w", pageNum, err)
	}
	defer bCtx.Close()

	page, err := bCtx.NewPage()
	if err != nil {
		return nil, fmt.Errorf("itviec: new page (page %d): %w", pageNum, err)
	}
	defer page.Close()

	if err := page.AddInitScript(playwright.Script{
		Content: playwright.String(`
			Object.defineProperty(navigator, 'webdriver', { get: () => undefined });
			window.chrome = { runtime: {} };
		`),
	}); err != nil {
		slog.Warn("ITviec: init script failed", "page", pageNum, "err", err)
	}

	select {
	case <-ctx.Done():
		return nil, nil
	default:
	}

	query := url.Values{}
	query.Set("page", fmt.Sprintf("%d", pageNum))
	if citySlug != "" {
		query.Set("city", citySlug)
	}

	var pageURL string
	alphaNum := regexp.MustCompile(`^[a-zA-Z0-9\s]+$`)
	if keyword != "" && alphaNum.MatchString(keyword) {
		slug := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(keyword), " ", "-"))
		pageURL = fmt.Sprintf("https://itviec.com/it-jobs/%s", slug)
	} else {
		pageURL = "https://itviec.com/it-jobs"
		if keyword != "" {
			query.Set("query", keyword)
		}
	}

	if useServerLevelFilter {
		for _, level := range itviecLevels {
			query.Add("job_level_names[]", level)
		}
	}

	encodedQuery := query.Encode()
	if encodedQuery != "" {
		pageURL += "?" + encodedQuery
	}

	if _, err := page.Goto(pageURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(35000),
	}); err != nil {
		return nil, fmt.Errorf("goto %s: %w", pageURL, err)
	}

	if _, err := page.WaitForSelector(".job-card", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(25000),
		State:   playwright.WaitForSelectorStateAttached,
	}); err != nil {
		slog.Warn("ITviec: .job-card timeout", "page", pageNum)
		return nil, nil
	}

	return s.scrapeDom(page, cityDisplayName)
}

func (s *ItviecScraper) scrapeDom(page playwright.Page, cityDisplayName string) ([]models.JobItem, error) {
	var jobs []models.JobItem

	cards, err := page.QuerySelectorAll(".job-card")
	if err != nil {
		return jobs, fmt.Errorf("itviec: query .job-card: %w", err)
	}

	for _, card := range cards {
		title := elemText(card, "h3, h2, [class*='title']")
		if title == "" || strings.Contains(title, "{{") {
			continue
		}

		company := elemText(card, "[class*='company'], [class*='employer']")
		salary := elemText(card, "[class*='salary'], [class*='wage']")
		cardText, _ := card.InnerText()
		cardText = strings.TrimSpace(cardText)
		cardTextLower := strings.ToLower(cardText)

		expRaw := strings.TrimSpace(elemText(card, "[class*='exp'], [class*='experience']"))
		if expRaw == "" {
			expRaw = extractExperienceFromText(title)
		}
		if expRaw == "" {
			expRaw = extractExperienceFromText(cardText)
		}

		levelRaw := title
		// ITviec "Level = Fresher" is often represented by the "Fresher Accepted" badge,
		// not always by title text.
		if strings.Contains(cardTextLower, "fresher accepted") || strings.Contains(cardTextLower, "fresher/junior") {
			levelRaw = "fresher"
		} else if strings.Contains(cardTextLower, "junior") {
			levelRaw = "junior"
		} else if strings.Contains(cardTextLower, "senior") {
			levelRaw = "senior"
		} else if strings.Contains(cardTextLower, "manager") {
			levelRaw = "manager"
		}
		level := helpers.NormalizeLevel(levelRaw)

		jobURL := ""
		if aEls, err := card.QuerySelectorAll("a[href]"); err == nil {
			for _, aEl := range aEls {
				href, _ := aEl.GetAttribute("href")
				if strings.Contains(href, "/sign_in?job=") {
					jobURL = itviecDirectJobURL("https://itviec.com" + href)
					break
				}
			}
		}

		if jobURL == "" {
			continue
		}

		location := cityDisplayName
		if location == "" {
			location = helpers.ExtractCityFromText(cardTextLower)
		}

		now := time.Now()
		jobs = append(jobs, models.JobItem{
			Title:      title,
			Company:    company,
			Location:   location,
			Salary:     salary,
			Level:      level,
			Experience: expRaw,
			Source:     s.SourceName(),
			PostedDate: &now,
			Url:        jobURL,
		})
	}

	return jobs, nil
}

func elemText(card playwright.ElementHandle, sel string) string {
	el, err := card.QuerySelector(sel)
	if err != nil || el == nil {
		return ""
	}
	text, err := el.InnerText()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(text)
}

func resolveCitySlug(locations []string) (string, string) {
	if len(locations) != 1 {
		return "", ""
	}
	switch locations[0] {
	case "tphcm":
		return "ho-chi-minh-hcm", "Ho Chi Minh"
	case "hanoi":
		return "ha-noi", "Ha Noi"
	case "danang":
		return "da-nang", "Da Nang"
	default:
		return "", ""
	}
}

func extractExperienceFromText(text string) string {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "" {
		return ""
	}

	rangeRe := regexp.MustCompile(`(\d+)\s*[-–~]\s*(\d+)\s*(?:\+)?\s*(?:nam|năm|year|years)`)
	if m := rangeRe.FindStringSubmatch(t); len(m) == 3 {
		return fmt.Sprintf("%s-%s years", m[1], m[2])
	}

	singleRe := regexp.MustCompile(`(\d+)\s*(\+)?\s*(?:nam|năm|year|years)`)
	if m := singleRe.FindStringSubmatch(t); len(m) >= 2 {
		if len(m) >= 3 && m[2] == "+" {
			return fmt.Sprintf("%s+ years", m[1])
		}
		return fmt.Sprintf("%s years", m[1])
	}

	return ""
}

// itviecDirectJobURL converts /sign_in?job=<slug> to direct job URL.
func itviecDirectJobURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	slug := u.Query().Get("job")
	if slug != "" {
		return "https://itviec.com/it-jobs/" + slug
	}

	slug = u.Query().Get("job_selected")
	if slug != "" {
		return "https://itviec.com/it-jobs/" + slug
	}

	return rawURL
}

func resolveItviecLevelParams(levels []string) ([]string, bool) {
	if len(levels) == 0 {
		return nil, false
	}

	mapping := map[string]string{
		"fresher": "Fresher",
		"junior":  "Junior",
		"senior":  "Senior",
		"manager": "Manager",
	}

	seen := map[string]bool{}
	var params []string
	for _, level := range levels {
		n := strings.ToLower(strings.TrimSpace(level))
		val, ok := mapping[n]
		if !ok {
			// If any requested level is unsupported by ITviec server filter,
			// keep existing behavior to avoid partial/incorrect filtering.
			return nil, false
		}
		if !seen[val] {
			seen[val] = true
			params = append(params, val)
		}
	}
	return params, len(params) > 0
}
