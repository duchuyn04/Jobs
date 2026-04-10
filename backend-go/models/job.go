package models

import "time"

// JobItem tương đương JobItem.cs
type JobItem struct {
	Title      string     `json:"title"`
	Company    string     `json:"company"`
	Location   string     `json:"location"`
	Salary     string     `json:"salary,omitempty"`
	Level      string     `json:"level"`
	Experience string     `json:"experience"`
	PostedDate *time.Time `json:"postedDate,omitempty"`
	Deadline   *time.Time `json:"deadline,omitempty"`
	DaysLeft   *int       `json:"daysLeft,omitempty"`
	Source     string     `json:"source"`
	Url        string     `json:"url"`
	LogoUrl    string     `json:"logoUrl,omitempty"`
}

// LocationDef defines a location with matching keywords
type LocationDef struct {
	Key      string
	Display  string
	Keywords []string
}

// SearchFilter tương đương SearchViewModel.cs
type SearchFilter struct {
	Keyword     string   `form:"keyword"`
	Levels      []string `form:"levels"`
	MinExp      *int     `form:"minExp"`
	MaxExp      *int     `form:"maxExp"`
	Sources     []string `form:"sources"`
	Locations   []string `form:"locations"`
	HideExpired bool     `form:"hideExpired"`
	MinDaysLeft *int     `form:"minDaysLeft"`
	Page        int      `form:"page"`
	PageSize    int      `form:"pageSize"`
}

// SearchResult tương đương JobResultViewModel.cs
type SearchResult struct {
	Jobs         []JobItem      `json:"jobs"`
	TotalCount   int            `json:"totalCount"`
	TotalPages   int            `json:"totalPages"`
	CountBySource map[string]int `json:"countBySource"`
	IsLoaded     bool           `json:"isLoaded"`
	Errors       []string       `json:"errors"`
	Filter       SearchFilter   `json:"filter"`
}

// AllSources là danh sách nguồn hỗ trợ - tương đương SearchViewModel.AllSources
var AllSources = []string{"JobOKO", "JobsGO", "TopDev", "TopCV", "ITviec", "Glints"}

// AllLevels - tương đương SearchViewModel.AllLevels
var AllLevels = []string{"intern", "fresher", "junior", "middle", "senior", "manager"}

// AllLocations - tương đương SearchViewModel.AllLocations
var AllLocations = []LocationDef{
	{
		Key:     "tphcm",
		Display: "TP.HCM",
		Keywords: []string{"hồ chí minh", "hcm", "tphcm", "tp.hcm", "ho chi minh"},
	},
	{
		Key:     "hanoi",
		Display: "Hà Nội",
		Keywords: []string{"hà nội", "ha noi", "hanoi"},
	},
	{
		Key:     "danang",
		Display: "Đà Nẵng",
		Keywords: []string{"đà nẵng", "da nang", "danang"},
	},
	{
		Key:     "remote",
		Display: "Remote",
		Keywords: []string{"remote", "wfh", "work from home", "toàn quốc", "anywhere"},
	},
}
