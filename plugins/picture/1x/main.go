package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

func main() {
	sdk.Run(&OneXPlugin{})
}

type OneXPlugin struct{}

func (p *OneXPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	category := req.Params["category"]
	if category == "" {
		category = "latest/awarded"
	}

	// 处理 magazine 路由
	if strings.HasPrefix(req.Route, "/1x/magazine") {
		return fetchMagazine(category)
	}

	// 获取分页参数
	page := req.Params["page"]
	if page == "" {
		page = "1"
	}
	size := req.Params["size"]
	if size == "" {
		size = "20"
	}

	return fetchGallery(category, page, size)
}

func fetchMagazine(category string) (*sdk.FeedResult, error) {
	rootURL := "https://1x.com"
	magazineURL := rootURL + "/magazine/" + category

	// Fetch magazine page
	body, status, err := host.HTTPGet(magazineURL, map[string]string{
		"Accept": "text/html",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch magazine page failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("magazine page http status %d", status)
	}

	// Parse items from magazine page
	items, err := parseMagazineItems(string(body), rootURL)
	if err != nil {
		return nil, fmt.Errorf("parse magazine items failed: %w", err)
	}

	// Extract metadata
	title := extractTitle(string(body))
	description := extractDescription(string(body))

	return &sdk.FeedResult{
		Title:       title,
		Description: description,
		Items:       items,
	}, nil
}

func parseMagazineItems(html string, rootURL string) ([]sdk.FeedItem, error) {
	items := []sdk.FeedItem{}

	// Magazine item pattern
	itemPattern := regexp.MustCompile(`<a[^>]*class="[^"]*magazine-thumb[^"]*"[^>]*href="([^"]*)"[^>]*>`)
	itemMatches := itemPattern.FindAllStringSubmatch(html, -1)

	for _, match := range itemMatches {
		if len(match) < 2 {
			continue
		}

		url := match[1]
		if !strings.HasPrefix(url, "http") {
			url = rootURL + url
		}

		// Extract ID from URL
		id := extractIDFromURL(url)
		if id == "" {
			continue
		}

		items = append(items, sdk.FeedItem{
			ID:        id,
			Title:     "Magazine Article",
			URL:       url,
			PublishedAt: time.Now().Format(time.RFC3339),
		})
	}

	return items, nil
}

func extractIDFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func fetchGallery(category string, page string, size string) (*sdk.FeedResult, error) {
	rootURL := "https://1x.com"
	galleryURL := rootURL + "/gallery/" + category

	// Fetch gallery page to extract mode value
	body, status, err := host.HTTPGet(galleryURL, map[string]string{
		"Accept": "text/html",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch gallery page failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("gallery page http status %d", status)
	}

	// Extract mode from lm_mode input
	mode := extractMode(string(body))
	if mode == "" {
		mode = "newest"
	}

	// Convert page and size to 'from' parameter (0-indexed offset)
	// from = (page - 1) * size
	// Note: 1x.com API has a hard limit of 20 items per request, regardless of size parameter
	pageNum := 1
	pageSize := 20
	if p, err := parseIntSafe(page); err == nil && p > 0 {
		pageNum = p
	}
	if s, err := parseIntSafe(size); err == nil && s > 0 {
		pageSize = s
	}
	// Cap size at 20 (API limitation)
	if pageSize > 20 {
		pageSize = 20
	}
	from := (pageNum - 1) * pageSize

	// Fetch API data with pagination support using 'from' parameter
	apiURL := fmt.Sprintf("%s/backend/lm2.php?style=normal&mode=%s&from=%d", rootURL, mode, from)
	apiBody, apiStatus, err := host.HTTPGet(apiURL, map[string]string{
		"Accept": "text/html",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch api failed: %w", err)
	}
	if apiStatus < 200 || apiStatus >= 300 {
		return nil, fmt.Errorf("api http status %d", apiStatus)
	}

	// Parse items from API response
	items, err := parseItems(string(apiBody), rootURL)
	if err != nil {
		return nil, fmt.Errorf("parse items failed: %w", err)
	}

	// Extract metadata from gallery page
	title := extractTitle(string(body))
	description := extractDescription(string(body))

	return &sdk.FeedResult{
		Title:       title,
		Description: description,
		Items:       items,
	}, nil
}

func extractMode(html string) string {
	re := regexp.MustCompile(`<input[^>]*id="lm_mode"[^>]*value="([^"]*)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func extractTitle(html string) string {
	re := regexp.MustCompile(`<title>([^<]+)</title>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}
	return "1x.com Gallery"
}

func extractDescription(html string) string {
	re := regexp.MustCompile(`<meta\s+name="description"\s+content="([^"]*)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func extractLogo(html string, rootURL string) string {
	re := regexp.MustCompile(`<img[^>]*class="themedlogo"[^>]*src="([^"]*)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		src := matches[1]
		if strings.HasPrefix(src, "http") {
			return src
		}
		return rootURL + src
	}
	return ""
}

func extractSiteName(html string) string {
	re := regexp.MustCompile(`<meta\s+property="og:site_name"\s+content="([^"]*)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}
	return "1x.com"
}

func parseItems(html string, rootURL string) ([]sdk.FeedItem, error) {
	items := []sdk.FeedItem{}

	// Split by item container
	itemPattern := regexp.MustCompile(`<div[^>]*class="[^"]*photos-feed-item[^"]*"[^>]*>(.*?)</div>\s*</div>`)
	itemMatches := itemPattern.FindAllStringSubmatch(html, -1)

	if len(itemMatches) == 0 {
		// Try alternative pattern
		itemMatches = findItemsByContainer(html)
	}

	for _, match := range itemMatches {
		if len(match) < 2 {
			continue
		}

		itemHTML := match[1]

		title := extractItemTitle(itemHTML)
		if title == "" {
			title = "Untitled"
		}

		author := extractItemAuthor(itemHTML)
		image := extractItemImage(itemHTML)
		id := extractItemID(itemHTML)

		if id == "" {
			continue
		}

		description := buildDescription(image, title, author)

		items = append(items, sdk.FeedItem{
			ID:        id,
			Title:     title,
			URL:       rootURL + "/photo/" + id,
			Summary:   title + " by " + author,
			Author:    author,
			Cover:     image,
			Image:     image,
			Content:   description,
			PublishedAt: time.Now().Format(time.RFC3339),
		})
	}

	return items, nil
}

func findItemsByContainer(html string) [][]string {
	// More flexible item extraction
	re := regexp.MustCompile(`(?s)<div[^>]*class="photos-feed-item"[^>]*>(.*?)<div[^>]*class="photos-feed-item"`)
	matches := re.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		// Try single item
		re = regexp.MustCompile(`(?s)<div[^>]*class="photos-feed-item"[^>]*>(.*?)(?:</div>)?$`)
		matches = re.FindAllStringSubmatch(html, -1)
	}
	return matches
}

func extractItemTitle(itemHTML string) string {
	re := regexp.MustCompile(`<span[^>]*class="[^"]*photos-feed-data-title[^"]*"[^>]*>([^<]+)</span>`)
	matches := re.FindStringSubmatch(itemHTML)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func extractItemAuthor(itemHTML string) string {
	re := regexp.MustCompile(`<span[^>]*class="[^"]*photos-feed-data-name[^"]*"[^>]*>([^<]+)</span>`)
	matches := re.FindStringSubmatch(itemHTML)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func extractItemImage(itemHTML string) string {
	re := regexp.MustCompile(`<img[^>]*src="([^"]*)"[^>]*>`)
	matches := re.FindStringSubmatch(itemHTML)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func extractItemID(itemHTML string) string {
	// Extract ID from img id attribute
	re := regexp.MustCompile(`<img[^>]*id="([^"]*)"`)
	matches := re.FindStringSubmatch(itemHTML)
	if len(matches) > 1 {
		id := matches[1]
		// Get last part after hyphen
		parts := strings.Split(id, "-")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
		return id
	}
	return ""
}

func buildDescription(image string, title string, author string) string {
	if image == "" {
		return fmt.Sprintf("%s by %s", title, author)
	}
	return fmt.Sprintf("<figure><img src=\"%s\" alt=\"%s\"/></figure><p>%s by %s</p>", 
		image, title, title, author)
}

func parseIntSafe(s string) (int, error) {
	return strconv.Atoi(s)
}

