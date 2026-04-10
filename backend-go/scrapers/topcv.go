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

	"github.com/playwright-community/playwright-go"
)

type TopCvScraper struct {
	pw      *playwright.Playwright
	browser playwright.Browser
}

func NewTopCvScraper(pw *playwright.Playwright, browser playwright.Browser) *TopCvScraper {
	return &TopCvScraper{pw: pw, browser: browser}
}

func (s *TopCvScraper) SourceName() string { return "TopCV" }

func (s *TopCvScraper) Scrape(ctx context.Context, filter models.SearchFilter) ([]models.JobItem, error) {
	var jobs []models.JobItem

	bCtx, err := s.browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"),
	})
	if err != nil {
		return jobs, fmt.Errorf("topcv: new context: %w", err)
	}
	defer bCtx.Close()

	page, err := bCtx.NewPage()
	if err != nil {
		return jobs, fmt.Errorf("topcv: new page: %w", err)
	}
	defer page.Close()

	if err := page.Route("**/*", func(route playwright.Route) {
		rt := route.Request().ResourceType()
		if rt == "image" || rt == "stylesheet" || rt == "font" || rt == "media" {
			route.Abort()
		} else {
			route.Continue()
		}
	}); err != nil {
		slog.Warn("TopCV: route setup failed", "err", err)
	}

	encodedKeyword := url.QueryEscape(filter.Keyword)
	baseURL := fmt.Sprintf("https://www.topcv.vn/tim-viec-lam-%s?sort=new", encodedKeyword)

	slog.Info("TopCV URL", "url", baseURL)
	_, err = page.Goto(baseURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if err != nil {
		return jobs, fmt.Errorf("topcv: goto: %w", err)
	}

	pageLimit := 5
	for p := 1; p <= pageLimit; p++ {
		// Wait for job listings to appear
		page.WaitForSelector(".job-item-search-result", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(5000),
		})

		rawJobs, err := page.Evaluate(`() => {
			const arr = [];
			document.querySelectorAll(".job-item-search-result").forEach(el => {
				const titleE = el.querySelector("h3.title span[data-original-title]") || el.querySelector("h3.title span");
				const urlE = el.querySelector("h3.title a[href]");
				const companyE = el.querySelector(".company");
				const locationE = el.querySelector(".address");
				const salaryE = el.querySelector(".salary");
				const expE = el.querySelector(".experience");
				
				let title = titleE ? (titleE.getAttribute("data-original-title") || titleE.innerText || titleE.textContent) : "";
				if (!title && urlE) title = urlE.innerText || urlE.textContent;
				
				let loc = locationE ? locationE.innerText || locationE.textContent : "";
				// locationE inside the badge can contain weird html tooltip, just clean it
				if (loc.includes("\n")) loc = loc.split("\n")[0];
				
				arr.push({
					title: title,
					url: urlE ? urlE.href : "",
					company: companyE ? companyE.innerText || companyE.textContent : "",
					location: loc,
					salary: salaryE ? salaryE.innerText || salaryE.textContent : "",
					experience: expE ? expE.innerText || expE.textContent : "",
					cardText: el.innerText || el.textContent || ""
				});
			});
			return arr;
		}`)
		if err != nil {
			slog.Warn("TopCV evaluate failed", "err", err)
			break
		}

		list, ok := rawJobs.([]interface{})
		if !ok || len(list) == 0 {
			break
		}

		added := 0
		for _, raw := range list {
			m, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}

			title := strings.TrimSpace(m["title"].(string))
			if title == "" {
				continue
			}

			company := strings.TrimSpace(m["company"].(string))
			loc := strings.TrimSpace(m["location"].(string))
			salary := strings.TrimSpace(m["salary"].(string))
			expRaw := strings.TrimSpace(m["experience"].(string))
			cardText := strings.TrimSpace(m["cardText"].(string))
			href := strings.TrimSpace(m["url"].(string))

			// Clean salary multiline spacing
			salary = regexp.MustCompile(`\s+`).ReplaceAllString(salary, " ")
			expRaw = regexp.MustCompile(`\s+`).ReplaceAllString(expRaw, " ")
			cardText = regexp.MustCompile(`\s+`).ReplaceAllString(cardText, " ")

			experience := normalizeExperienceYears(expRaw)
			if experience == "" {
				experience = normalizeExperienceYears(cardText)
			}
			if experience == "" {
				experience = normalizeExperienceYears(title)
			}
			if experience == "" {
				experience = expRaw
			}

			jobURL := href
			if !strings.HasPrefix(href, "http") {
				jobURL = "https://www.topcv.vn" + href
			}

			jobs = append(jobs, models.JobItem{
				Title:      title,
				Company:    company,
				Location:   loc,
				Salary:     salary,
				Level:      helpers.NormalizeLevel(expRaw),
				Experience: experience,
				Source:     s.SourceName(),
				Url:        jobURL,
			})
			added++
		}

		slog.Info("TopCV page scraped", "page", p, "added", added)

		// Check if there is a next page
		nextBtn, err := page.QuerySelector("ul.pagination li a[rel='next']")
		if err != nil || nextBtn == nil {
			break
		}
		
		href, err := nextBtn.GetAttribute("href")
		if err != nil || href == "" {
			break
		}
		
		err = nextBtn.Click()
		if err != nil {
			break
		}
		
		page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
			State: playwright.LoadStateDomcontentloaded,
		})
	}

	slog.Info("TopCV scraped total", "count", len(jobs), "keyword", filter.Keyword)
	return jobs, nil
}

func normalizeExperienceYears(text string) string {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "" {
		return ""
	}
	t = strings.ReplaceAll(t, "n\u0103m", "nam")

	rangeRe := regexp.MustCompile(`(\d+)\s*[-~]\s*(\d+)\s*(?:\+)?\s*(?:nam|year|years)`)
	if m := rangeRe.FindStringSubmatch(t); len(m) == 3 {
		return fmt.Sprintf("%s-%s years", m[1], m[2])
	}

	singleRe := regexp.MustCompile(`(\d+)\s*(\+)?\s*(?:nam|year|years)`)
	if m := singleRe.FindStringSubmatch(t); len(m) >= 2 {
		if len(m) >= 3 && m[2] == "+" {
			return fmt.Sprintf("%s+ years", m[1])
		}
		return fmt.Sprintf("%s years", m[1])
	}

	return ""
}
