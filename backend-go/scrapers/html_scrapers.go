package scrapers

// htmlScraper.go - shared helpers cho các HTML scrapers (JobsGO, JobOKO, TopDev)
// Thay thế HtmlAgilityPack bằng golang.org/x/net/html

import (
	"context"
	"fmt"
	"io"
	"jobaggregator/helpers"
	"jobaggregator/models"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// htmlScraper là base struct cho các HTTP scraper đơn giản
type htmlScraper struct {
	client     *http.Client
	sourceName string
	baseURL    string
	searchURL  func(keyword string) string
	cardSelectors []string
}

func (s *htmlScraper) SourceName() string { return s.sourceName }

func (s *htmlScraper) fetchHTML(ctx context.Context, rawURL string) (*html.Node, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept-Language", "vi-VN,vi;q=0.9,en-US;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return html.Parse(strings.NewReader(string(body)))
}

// --- JobsGO Scraper ---

type JobsGoScraper struct {
	client *http.Client
}

func NewJobsGoScraper() *JobsGoScraper {
	return &JobsGoScraper{client: &http.Client{}}
}

func (s *JobsGoScraper) SourceName() string { return "JobsGO" }

func (s *JobsGoScraper) Scrape(ctx context.Context, filter models.SearchFilter) ([]models.JobItem, error) {
	searchURL := fmt.Sprintf(
		"https://jobsgo.vn/viec-lam/tim-kiem/?q=%s",
		url.QueryEscape(filter.Keyword),
	)
	return scrapeHTMLJobs(ctx, s.client, s.SourceName(), searchURL, "https://jobsgo.vn")
}

// --- JobOKO Scraper ---

type JobOkoScraper struct {
	client *http.Client
}

func NewJobOkoScraper() *JobOkoScraper {
	return &JobOkoScraper{client: &http.Client{}}
}

func (s *JobOkoScraper) SourceName() string { return "JobOKO" }

func (s *JobOkoScraper) Scrape(ctx context.Context, filter models.SearchFilter) ([]models.JobItem, error) {
	searchURL := fmt.Sprintf(
		"https://joboko.com/tim-kiem-viec-lam?q=%s",
		url.QueryEscape(filter.Keyword),
	)
	return scrapeHTMLJobs(ctx, s.client, s.SourceName(), searchURL, "https://joboko.com")
}

// --- TopDev Scraper ---

type TopDevScraper struct {
	client *http.Client
}

func NewTopDevScraper() *TopDevScraper {
	return &TopDevScraper{client: &http.Client{}}
}

func (s *TopDevScraper) SourceName() string { return "TopDev" }

func (s *TopDevScraper) Scrape(ctx context.Context, filter models.SearchFilter) ([]models.JobItem, error) {
	searchURL := fmt.Sprintf(
		"https://topdev.vn/viec-lam-it?q=%s&page=1",
		url.QueryEscape(filter.Keyword),
	)
	return scrapeHTMLJobs(ctx, s.client, s.SourceName(), searchURL, "https://topdev.vn")
}

// scrapeHTMLJobs - shared logic để scrape HTML job cards
// Tương đương HtmlAgilityPack scraping pattern trong C#
func scrapeHTMLJobs(ctx context.Context, client *http.Client, sourceName, searchURL, baseURL string) ([]models.JobItem, error) {
	var jobs []models.JobItem

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return jobs, fmt.Errorf("%s: %w", sourceName, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept-Language", "vi-VN,vi;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("scrape failed", "source", sourceName, "err", err)
		return jobs, nil
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return jobs, fmt.Errorf("%s: parse html: %w", sourceName, err)
	}

	// Tìm tất cả các node có class chứa "job-item" hoặc "job-card"
	cards := findNodes(doc, func(n *html.Node) bool {
		cls := getAttr(n, "class")
		return n.Type == html.ElementNode &&
			(strings.Contains(cls, "job-item") || strings.Contains(cls, "job-card"))
	})

	for _, card := range cards {
		title := innerText(findFirst(card, func(n *html.Node) bool {
			return n.Type == html.ElementNode &&
				(n.Data == "h3" || n.Data == "h2" ||
					strings.Contains(getAttr(n, "class"), "title") ||
					strings.Contains(getAttr(n, "class"), "job-name"))
		}))

		company := innerText(findFirst(card, func(n *html.Node) bool {
			cls := getAttr(n, "class")
			return n.Type == html.ElementNode &&
				(strings.Contains(cls, "company") || strings.Contains(cls, "employer") ||
					strings.Contains(cls, "company-name"))
		}))

		location := innerText(findFirst(card, func(n *html.Node) bool {
			cls := getAttr(n, "class")
			return n.Type == html.ElementNode &&
				(strings.Contains(cls, "location") || strings.Contains(cls, "address"))
		}))

		salary := innerText(findFirst(card, func(n *html.Node) bool {
			cls := getAttr(n, "class")
			return n.Type == html.ElementNode &&
				(strings.Contains(cls, "salary") || strings.Contains(cls, "wage"))
		}))

		expRaw := innerText(findFirst(card, func(n *html.Node) bool {
			cls := getAttr(n, "class")
			return n.Type == html.ElementNode &&
				(strings.Contains(cls, "exp") || strings.Contains(cls, "experience"))
		}))

		deadlineRaw := innerText(findFirst(card, func(n *html.Node) bool {
			cls := getAttr(n, "class")
			return n.Type == html.ElementNode &&
				(strings.Contains(cls, "deadline") || strings.Contains(cls, "date"))
		}))

		href := ""
		if linkNode := findFirst(card, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "a" && getAttr(n, "href") != ""
		}); linkNode != nil {
			href = getAttr(linkNode, "href")
		}

		if title == "" || strings.Contains(title, "{") {
			continue
		}

		deadline := helpers.ParseDate(deadlineRaw)
		jobURL := href
		if !strings.HasPrefix(href, "http") {
			jobURL = baseURL + href
		}

		jobs = append(jobs, models.JobItem{
			Title:      title,
			Company:    company,
			Location:   location,
			Salary:     salary,
			Level:      helpers.NormalizeLevel(expRaw),
			Experience: expRaw,
			Deadline:   deadline,
			DaysLeft:   helpers.CalcDaysLeft(deadline),
			Source:     sourceName,
			Url:        jobURL,
		})
	}

	slog.Info("scraped", "source", sourceName, "count", len(jobs))
	return jobs, nil
}

// --- HTML helper functions ---

func findNodes(n *html.Node, match func(*html.Node) bool) []*html.Node {
	var results []*html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if match(node) {
			results = append(results, node)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return results
}

func findFirst(n *html.Node, match func(*html.Node) bool) *html.Node {
	if n == nil {
		return nil
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if match(c) {
			return c
		}
		if found := findFirst(c, match); found != nil {
			return found
		}
	}
	return nil
}

func getAttr(n *html.Node, key string) string {
	if n == nil {
		return ""
	}
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func innerText(n *html.Node) string {
	if n == nil {
		return ""
	}
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(sb.String())
}
