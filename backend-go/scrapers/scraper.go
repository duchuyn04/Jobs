package scrapers

import (
	"context"
	"jobaggregator/models"
)

// IScraper - tương đương interface IJobScraper trong C#
type IScraper interface {
	SourceName() string
	Scrape(ctx context.Context, filter models.SearchFilter) ([]models.JobItem, error)
}
