package helpers

import "strings"

// NormalizeLevel - tương đương LevelMappingHelper.Normalize trong C#
func NormalizeLevel(raw string) string {
	if raw == "" {
		return "unknown"
	}
	t := strings.ToLower(raw)
	if strings.Contains(t, "intern") || strings.Contains(t, "thực tập") {
		return "intern"
	}
	if strings.Contains(t, "fresher") || strings.Contains(t, "entry") || strings.Contains(t, "graduate") {
		return "fresher"
	}
	if strings.Contains(t, "junior") || strings.Contains(t, "jr.") {
		return "junior"
	}
	if strings.Contains(t, "manager") || strings.Contains(t, "management") {
		return "manager"
	}
	if strings.Contains(t, "senior") || strings.Contains(t, "sr.") || strings.Contains(t, "lead") ||
		strings.Contains(t, "principal") || strings.Contains(t, "staff") {
		return "senior"
	}
	if strings.Contains(t, "mid") || strings.Contains(t, "middle") || strings.Contains(t, "associate") {
		return "middle"
	}
	return "unknown"
}

// ExtractLevelFromTitle - tương đương ItviecScraper.ExtractLevelFromTitle
func ExtractLevelFromTitle(title string) string {
	return NormalizeLevel(title)
}

// MatchesFilter - tương đương LevelMappingHelper.MatchesFilter
func MatchesFilter(jobLevel, filterLevel string) bool {
	return strings.EqualFold(jobLevel, filterLevel)
}

// ExtractCityFromText - tương đương ItviecScraper.ExtractCityFromText
func ExtractCityFromText(lowerText string) string {
	if strings.Contains(lowerText, "hồ chí minh") || strings.Contains(lowerText, "hcm") ||
		strings.Contains(lowerText, "hcmc") || strings.Contains(lowerText, "saigon") {
		return "Hồ Chí Minh"
	}
	if strings.Contains(lowerText, "hà nội") || strings.Contains(lowerText, "hanoi") {
		return "Hà Nội"
	}
	if strings.Contains(lowerText, "đà nẵng") || strings.Contains(lowerText, "da nang") {
		return "Đà Nẵng"
	}
	if strings.Contains(lowerText, "remote") || strings.Contains(lowerText, "work from home") ||
		strings.Contains(lowerText, "wfh") {
		return "Remote"
	}
	return "Vietnam"
}
