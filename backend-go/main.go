package main

import (
	"jobaggregator/models"
	"jobaggregator/scrapers"
	"jobaggregator/services"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/playwright-community/playwright-go"
)

func main() {
	// Khởi tạo Playwright browser (singleton) - dùng chung cho ItviecScraper
	pw, err := playwright.Run()
	if err != nil {
		slog.Error("Could not start playwright", "err", err)
		panic(err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		slog.Error("Could not launch browser", "err", err)
		panic(err)
	}
	defer browser.Close()

	scraperList := []scrapers.IScraper{
		scrapers.NewTopCvScraper(pw, browser),
		scrapers.NewJobsGoScraper(),
		scrapers.NewJobOkoScraper(),
		scrapers.NewTopDevScraper(),
		scrapers.NewGlintsScraper(),
		scrapers.NewItviecScraper(pw, browser),
	}

	searchSvc := services.NewSearchService(scraperList)

	// Gin router
	r := gin.Default()

	// CORS cho Next.js - tương đương CORS policy trong Program.cs
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: false,
	}))

	// API endpoint - tương đương /api/jobs/search trong JobController.cs
	r.GET("/api/jobs/search", func(c *gin.Context) {
		filter := parseFilter(c)
		result := searchSvc.Search(filter)
		c.JSON(http.StatusOK, result)
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	slog.Info("Go API server starting", "port", 8080)
	if err := r.Run(":8080"); err != nil {
		slog.Error("Server failed", "err", err)
	}
}

// parseFilter - đọc query params từ HTTP request,
// tương đương model binding trong C# ([FromQuery] SearchViewModel)
func parseFilter(c *gin.Context) models.SearchFilter {
	filter := models.SearchFilter{
		Keyword:     c.Query("keyword"),
		HideExpired: c.DefaultQuery("hideExpired", "true") == "true",
		Page:        1,
		PageSize:    20,
	}

	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		filter.Page = p
	}
	if ps, err := strconv.Atoi(c.Query("pageSize")); err == nil && ps > 0 {
		filter.PageSize = ps
	}

	// Multi-value params: ?levels=junior&levels=senior
	if lvls := c.QueryArray("levels"); len(lvls) > 0 {
		filter.Levels = lvls
	}
	if srcs := c.QueryArray("sources"); len(srcs) > 0 {
		filter.Sources = srcs
	}
	if locs := c.QueryArray("locations"); len(locs) > 0 {
		filter.Locations = locs
	}

	// Comma-separated fallback: ?levels=junior,senior
	if len(filter.Levels) == 0 && c.Query("levels") != "" {
		filter.Levels = strings.Split(c.Query("levels"), ",")
	}

	return filter
}
