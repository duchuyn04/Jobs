package services

import (
	"context"
	"fmt"
	"jobaggregator/models"
	"jobaggregator/scrapers"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	timeoutSeconds = 60
	cacheMinutes   = 10
)

// cacheEntry lưu jobs + errors theo cache key
type cacheEntry struct {
	jobs      []models.JobItem
	errors    []string
	expiresAt time.Time
}

// SearchService - tương đương JobSearchService.cs
// Dùng sync.Map làm in-memory cache (thay IMemoryCache của C#)
// Dùng goroutines + WaitGroup thay Task.WhenAll
type SearchService struct {
	scrapers []scrapers.IScraper
	cache    sync.Map // map[string]*cacheEntry
}

func NewSearchService(scraperList []scrapers.IScraper) *SearchService {
	return &SearchService{scrapers: scraperList}
}

func (s *SearchService) Search(filter models.SearchFilter) models.SearchResult {
	// Default values
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	filter.PageSize = clamp(filter.PageSize, 5, 100)

	cacheKey := buildCacheKey(filter)

	// Kiểm tra cache
	var filteredJobs []models.JobItem
	var cachedErrors []string

	if entry, ok := s.cache.Load(cacheKey); ok {
		e := entry.(*cacheEntry)
		if time.Now().Before(e.expiresAt) {
			slog.Info("Cache hit", "key", cacheKey)
			filteredJobs = e.jobs
			cachedErrors = e.errors
		}
	}

	if filteredJobs == nil {
		// Xác định scrapers cần gọi - tương đương logic activeSources trong C#
		selectedScrapers := s.selectScrapers(filter.Sources)

		// Chạy song song với timeout 60s - tương đương Task.WhenAll
		ctx, cancel := context.WithTimeout(context.Background(), timeoutSeconds*time.Second)
		defer cancel()

		type scraperResult struct {
			source string
			jobs   []models.JobItem
			err    string
		}

		resultCh := make(chan scraperResult, len(selectedScrapers))
		var wg sync.WaitGroup

		for _, sc := range selectedScrapers {
			wg.Add(1)
			go func(sc scrapers.IScraper) {
				defer wg.Done()
				jobs, err := sc.Scrape(ctx, filter)
				if err != nil {
					slog.Warn("Scraper error", "source", sc.SourceName(), "err", err)
					resultCh <- scraperResult{
						source: sc.SourceName(),
						jobs:   []models.JobItem{},
						err:    fmt.Sprintf("%s: Lỗi kết nối", sc.SourceName()),
					}
					return
				}
				resultCh <- scraperResult{source: sc.SourceName(), jobs: jobs}
			}(sc)
		}

		// Đóng channel khi tất cả goroutines xong
		go func() {
			wg.Wait()
			close(resultCh)
		}()

		var allJobs []models.JobItem
		var errors []string
		for r := range resultCh {
			allJobs = append(allJobs, r.jobs...)
			if r.err != "" {
				errors = append(errors, r.err)
			}
		}

		filteredJobs = applyFilters(allJobs, filter)

		// Lưu cache
		s.cache.Store(cacheKey, &cacheEntry{
			jobs:      filteredJobs,
			errors:    errors,
			expiresAt: time.Now().Add(cacheMinutes * time.Minute),
		})
		cachedErrors = errors

		slog.Info("Scraped", "total", len(allJobs), "filtered", len(filteredJobs))
	}

	// Phân trang in-memory - tương đương C#
	totalCount := len(filteredJobs)
	pageSize := filter.PageSize
	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}
	page := clamp(filter.Page, 1, totalPages)

	sorted := make([]models.JobItem, len(filteredJobs))
	copy(sorted, filteredJobs)

	// Normalize missing PostedDate
	now := time.Now()
	for i := range sorted {
		if sorted[i].PostedDate == nil {
			sorted[i].PostedDate = &now
		}
	}

	// Sort: PostedDate desc (rounded to day), then Title asc (to interleave sources naturally)
	sort.Slice(sorted, func(i, j int) bool {
		ti := *sorted[i].PostedDate
		tj := *sorted[j].PostedDate

		dayI := time.Date(ti.Year(), ti.Month(), ti.Day(), 0, 0, 0, 0, time.UTC)
		dayJ := time.Date(tj.Year(), tj.Month(), tj.Day(), 0, 0, 0, 0, time.UTC)

		if dayI.Equal(dayJ) {
			return sorted[i].Title < sorted[j].Title
		}
		return dayI.After(dayJ)
	})

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(sorted) {
		start = len(sorted)
	}
	if end > len(sorted) {
		end = len(sorted)
	}
	pagedJobs := sorted[start:end]

	// Count by source
	countBySource := map[string]int{}
	for _, j := range filteredJobs {
		countBySource[j.Source]++
	}

	filter.Page = page
	filter.PageSize = pageSize

	return models.SearchResult{
		Jobs:          pagedJobs,
		TotalCount:    totalCount,
		TotalPages:    totalPages,
		CountBySource: countBySource,
		IsLoaded:      true,
		Errors:        cachedErrors,
		Filter:        filter,
	}
}

// selectScrapers - tương đương activeSources logic trong C#
func (s *SearchService) selectScrapers(sources []string) []scrapers.IScraper {
	if len(sources) == 0 {
		return s.scrapers
	}
	sourceSet := map[string]bool{}
	for _, src := range sources {
		sourceSet[strings.ToLower(src)] = true
	}
	var selected []scrapers.IScraper
	for _, sc := range s.scrapers {
		if sourceSet[strings.ToLower(sc.SourceName())] {
			selected = append(selected, sc)
		}
	}
	return selected
}

// applyFilters - dịch nguyên vẹn ApplyFilters từ C#
func applyFilters(jobs []models.JobItem, filter models.SearchFilter) []models.JobItem {
	result := jobs

	// Filter by level
	if len(filter.Levels) > 0 {
		levelSet := map[string]bool{}
		for _, l := range filter.Levels {
			levelSet[strings.ToLower(l)] = true
		}
		var filtered []models.JobItem
		for _, j := range result {
			// ITviec level parser cannot always infer server-side level tags from card text.
			// When levels are fully supported by ITviec (fresher/junior/senior/manager),
			// scraper already pushed these filters to ITviec URL.
			if strings.EqualFold(j.Source, "ITviec") && canBypassItviecLocalLevelFilter(filter.Levels) {
				filtered = append(filtered, j)
				continue
			}
			if levelSet[strings.ToLower(j.Level)] {
				filtered = append(filtered, j)
			}
		}
		result = filtered
	}

	// Filter by location
	if len(filter.Locations) > 0 {
		var matchedLocs []models.LocationDef
		for _, locKey := range filter.Locations {
			for _, allLoc := range models.AllLocations {
				if allLoc.Key == locKey {
					matchedLocs = append(matchedLocs, allLoc)
					break
				}
			}
		}
		if len(matchedLocs) > 0 {
			var filtered []models.JobItem
			for _, j := range result {
				jobLoc := strings.ToLower(j.Location)
				for _, loc := range matchedLocs {
					matched := false
					for _, kw := range loc.Keywords {
						if strings.Contains(jobLoc, kw) {
							matched = true
							break
						}
					}
					if matched {
						filtered = append(filtered, j)
						break
					}
				}
			}
			result = filtered
		}
	}

	// HideExpired
	if filter.HideExpired {
		var filtered []models.JobItem
		for _, j := range result {
			if j.DaysLeft == nil || *j.DaysLeft >= 0 {
				filtered = append(filtered, j)
			}
		}
		result = filtered
	}

	// MinDaysLeft
	if filter.MinDaysLeft != nil {
		var filtered []models.JobItem
		for _, j := range result {
			if j.DaysLeft == nil || *j.DaysLeft >= *filter.MinDaysLeft {
				filtered = append(filtered, j)
			}
		}
		result = filtered
	}

	return result
}

// buildCacheKey - tương đương BuildResultCacheKey trong C#
func buildCacheKey(f models.SearchFilter) string {
	levels := strings.Join(f.Levels, ",")
	sources := strings.Join(f.Sources, ",")
	locs := strings.Join(f.Locations, ",")
	minExp := ""
	if f.MinExp != nil {
		minExp = fmt.Sprintf("%d", *f.MinExp)
	}
	maxExp := ""
	if f.MaxExp != nil {
		maxExp = fmt.Sprintf("%d", *f.MaxExp)
	}
	minDays := ""
	if f.MinDaysLeft != nil {
		minDays = fmt.Sprintf("%d", *f.MinDaysLeft)
	}
	return fmt.Sprintf("jobs:%s:%s:%s:%s:%s:%v:%s:%s",
		f.Keyword, levels, minExp, maxExp, sources, f.HideExpired, minDays, locs)
}

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func canBypassItviecLocalLevelFilter(levels []string) bool {
	if len(levels) == 0 {
		return false
	}
	supported := map[string]bool{
		"fresher": true,
		"junior":  true,
		"senior":  true,
		"manager": true,
	}
	for _, l := range levels {
		if !supported[strings.ToLower(strings.TrimSpace(l))] {
			return false
		}
	}
	return true
}
