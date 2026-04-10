package scrapers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"jobaggregator/helpers"
	"jobaggregator/models"
	"log/slog"
	"net/http"
	"net/url"
)

// GlintsScraper - dịch từ GlintsScraper.cs, dùng Glints REST API
type GlintsScraper struct {
	client *http.Client
}

func NewGlintsScraper() *GlintsScraper {
	return &GlintsScraper{client: &http.Client{}}
}

func (s *GlintsScraper) SourceName() string { return "Glints" }

func (s *GlintsScraper) Scrape(ctx context.Context, filter models.SearchFilter) ([]models.JobItem, error) {
	var jobs []models.JobItem

	apiURL := fmt.Sprintf(
		"https://glints.com/api/v2/jobs/search?keyword=%s&countryCode=VN&pageSize=20&pageIndex=0",
		url.QueryEscape(filter.Keyword),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return jobs, fmt.Errorf("glints: create request: %w", err)
	}
	req.Header.Set("x-glints-country", "VN")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		slog.Warn("Glints request failed", "err", err)
		return jobs, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("Glints non-200", "status", resp.StatusCode)
		return jobs, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return jobs, fmt.Errorf("glints: read body: %w", err)
	}

	var result struct {
		Data []map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return jobs, fmt.Errorf("glints: parse JSON: %w", err)
	}

	for _, item := range result.Data {
		title := glintStr(item, "title")
		if title == "" {
			continue
		}

		// company.name
		company := ""
		if compRaw, ok := item["company"]; ok {
			var comp map[string]json.RawMessage
			if json.Unmarshal(compRaw, &comp) == nil {
				company = glintStr(comp, "name")
			}
		}

		// city.name hoặc locationName
		city := ""
		if cityRaw, ok := item["city"]; ok {
			var c map[string]json.RawMessage
			if json.Unmarshal(cityRaw, &c) == nil {
				city = glintStr(c, "name")
			}
		}
		if city == "" {
			city = glintStr(item, "locationName")
		}
		if city == "" {
			city = "Vietnam"
		}

		salary := glintStr(item, "salaryEstimate")
		if salary == "" {
			salary = glintStr(item, "salaryRange")
		}
		expRaw := glintStr(item, "minYearsOfExperience")
		if expRaw == "" {
			expRaw = glintStr(item, "experienceLevel")
		}

		jobID := glintStr(item, "id")
		slug := glintStr(item, "slug")

		endDate := glintStr(item, "endDate")
		if endDate == "" {
			endDate = glintStr(item, "closingDate")
		}
		deadline := helpers.ParseDate(endDate)

		jobURL := "https://glints.com/vn"
		if slug != "" && jobID != "" {
			jobURL = fmt.Sprintf("https://glints.com/vn/opportunities/jobs/%s/%s", slug, jobID)
		}

		jobs = append(jobs, models.JobItem{
			Title:      title,
			Company:    company,
			Location:   city,
			Salary:     salary,
			Level:      helpers.NormalizeLevel(expRaw),
			Experience: expRaw,
			Deadline:   deadline,
			DaysLeft:   helpers.CalcDaysLeft(deadline),
			Source:     s.SourceName(),
			Url:        jobURL,
		})
	}

	slog.Info("Glints scraped", "count", len(jobs), "keyword", filter.Keyword)
	return jobs, nil
}

// glintStr - lấy string từ JSON raw message
func glintStr(m map[string]json.RawMessage, key string) string {
	if v, ok := m[key]; ok {
		var s string
		if json.Unmarshal(v, &s) == nil {
			return s
		}
	}
	return ""
}
