package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

func main() {
	sdk.Run(&HelloGitHubPlugin{})
}

type HelloGitHubPlugin struct{}

func (p *HelloGitHubPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/hellogithub/tags":
		return fetchTags()
	case req.Route == "/hellogithub/category/:tid":
		tid := req.Params["tid"]
		if tid == "" {
			return nil, fmt.Errorf("missing tid parameter")
		}
		sort := req.Params["sort"]
		if sort == "" {
			sort = "featured"
		}
		return fetchByCategory(tid, sort)
	case req.Route == "/hellogithub/detail/:id":
		// Extract full_name from URL or id parameter
		urlStr := req.Params["url"]
		id := req.Params["id"]
		
		var fullName string
		if urlStr != "" {
			// Extract full_name from URL: https://hellogithub.com/repository/author/project
			fullName = extractFullNameFromURL(urlStr)
		} else if id != "" {
			// Use id directly as full_name
			fullName = id
		}
		
		if fullName == "" {
			return nil, fmt.Errorf("missing url or id parameter")
		}
		return fetchDetail(fullName)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func extractFullNameFromURL(urlStr string) string {
	// Extract full_name from URL like: https://hellogithub.com/repository/Andyyyy64/whichllm
	parts := strings.Split(urlStr, "/repository/")
	if len(parts) != 2 {
		return ""
	}
	fullName := strings.TrimSpace(parts[1])
	fullName = strings.TrimRight(fullName, "/")
	return fullName
}

type TagInfo struct {
	Name      string `json:"name"`
	NameEn    string `json:"name_en"`
	Tid       string `json:"tid"`
	IconName  string `json:"icon_name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type TagsResponse struct {
	Success bool      `json:"success"`
	Data    []TagInfo `json:"data"`
}

func fetchTags() (*sdk.FeedResult, error) {
	body, status, err := host.HTTPGet("https://abroad.hellogithub.com/v1/tag/", map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	var tagsResp TagsResponse
	if err := json.Unmarshal(body, &tagsResp); err != nil {
		return nil, fmt.Errorf("parse tags response: %w", err)
	}

	if !tagsResp.Success || len(tagsResp.Data) == 0 {
		return nil, fmt.Errorf("empty tags data")
	}

	items := make([]sdk.FeedItem, 0, len(tagsResp.Data))
	for _, tag := range tagsResp.Data {
		item := sdk.FeedItem{
			ID:    tag.Tid,
			Title: tag.Name,
			URL:   fmt.Sprintf("https://hellogithub.com/tags/%s", tag.Tid),
		}
		items = append(items, item)
	}

	return &sdk.FeedResult{
		Title:       "HelloGithub 分类",
		Description: "所有可用分类",
		Items:       items,
	}, nil
}

type CategoryResponse struct {
	Success bool        `json:"success"`
	Page    int         `json:"page"`
	Data    []RepoItem  `json:"data"`
	HasMore bool        `json:"has_more"`
}

type RepoItem struct {
	ItemID      string `json:"item_id"`
	FullName    string `json:"full_name"`
	Title       string `json:"title"`
	TitleEn     string `json:"title_en"`
	Author      string `json:"author"`
	AuthorAvatar string `json:"author_avatar"`
	Name        string `json:"name"`
	Summary     string `json:"summary"`
	SummaryEn   string `json:"summary_en"`
	IsHot       bool   `json:"is_hot"`
	IsClaimed   bool   `json:"is_claimed"`
	PrimaryLang string `json:"primary_lang"`
	LangColor   string `json:"lang_color"`
	ClicksTotal int    `json:"clicks_total"`
	CommentTotal int   `json:"comment_total"`
	UpdatedAt   string `json:"updated_at"`
}

func fetchByCategory(tid, sort string) (*sdk.FeedResult, error) {
	apiURL := fmt.Sprintf("https://abroad.hellogithub.com/v1/?sort_by=%s&page=1&rank_by=newest&tid=%s", sort, tid)

	body, status, err := host.HTTPGet(apiURL, map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	var catResp CategoryResponse
	if err := json.Unmarshal(body, &catResp); err != nil {
		return nil, fmt.Errorf("parse category response: %w", err)
	}

	if !catResp.Success || len(catResp.Data) == 0 {
		return nil, fmt.Errorf("empty category data")
	}

	items := make([]sdk.FeedItem, 0, len(catResp.Data))
	for _, repo := range catResp.Data {
		pubDate := publishedAtRFC3339(repo.UpdatedAt)
		if pubDate == "" {
			pubDate = time.Now().Format(time.RFC3339)
		}

		item := sdk.FeedItem{
			ID:          repo.FullName,
			Title:       fmt.Sprintf("%s: %s", repo.Name, repo.Title),
			URL:         fmt.Sprintf("https://hellogithub.com/repository/%s", repo.FullName),
			Summary:     repo.Summary,
			Author:      repo.Author,
			PublishedAt: pubDate,
		}

		if repo.PrimaryLang != "" {
			item.Tags = append(item.Tags, repo.PrimaryLang)
		}

		items = append(items, item)
	}

	return &sdk.FeedResult{
		Title:       fmt.Sprintf("HelloGithub - %s", sort),
		Description: "开源项目推荐",
		Items:       items,
	}, nil
}

func fetchDetail(fullName string) (*sdk.FeedResult, error) {
	detailURL := fmt.Sprintf("https://hellogithub.com/repository/%s", fullName)

	body, status, err := host.HTTPGet(detailURL, map[string]string{
		"Accept": "text/html,application/xhtml+xml",
	})
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	content := extractDetailContent(string(body))
	title := fullName

	item := sdk.FeedItem{
		ID:      fullName,
		Title:   title,
		URL:     detailURL,
		Content: content,
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: "项目详情",
		Items:       []sdk.FeedItem{item},
	}, nil
}

func extractDetailContent(htmlBody string) string {
	var content strings.Builder

	// 1. Extract image
	imageHTML := extractImageDiv(htmlBody)
	if imageHTML != "" {
		content.WriteString(imageHTML)
		content.WriteString("\n")
	}

	// 2. Extract main description
	descriptionHTML := extractDescriptionDiv(htmlBody)
	if descriptionHTML != "" {
		content.WriteString(descriptionHTML)
	}

	result := content.String()
	if result == "" {
		return "项目详情"
	}

	return result
}

func extractImageDiv(htmlBody string) string {
	// Look for the image div container
	// Pattern: <div class="flex cursor-zoom-in justify-center pt-2"><div class="relative flex"><img ... /></div></div>
	
	startMarker := `<div class="flex cursor-zoom-in justify-center pt-2">`
	startIdx := strings.Index(htmlBody, startMarker)
	if startIdx < 0 {
		return ""
	}

	// Find img tag
	imgStart := strings.Index(htmlBody[startIdx:], `<img`)
	if imgStart < 0 {
		return ""
	}
	
	imgStart += startIdx
	imgEnd := strings.Index(htmlBody[imgStart:], `/>`)
	if imgEnd < 0 {
		return ""
	}
	
	imgEnd += imgStart + 2
	
	// Extract just the image tag with proper wrapping
	imgTag := htmlBody[imgStart:imgEnd]
	
	// Clean up the img tag - remove !gif suffix from src
	imgTag = strings.ReplaceAll(imgTag, "!gif\"", "\"")
	
	return fmt.Sprintf(`<div class="flex cursor-zoom-in justify-center pt-2"><div class="relative flex">%s</div></div>`, imgTag)
}

func extractDescriptionDiv(htmlBody string) string {
	// Look for the description paragraph
	// Target: <div class="w-full p-2 leading-8">内容</div>
	
	startMarker := `<div class="w-full p-2 leading-8">`
	startIdx := strings.Index(htmlBody, startMarker)
	if startIdx < 0 {
		return ""
	}

	startIdx += len(startMarker)
	
	// Find the closing div
	endMarker := `</div>`
	endIdx := strings.Index(htmlBody[startIdx:], endMarker)
	if endIdx < 0 {
		return ""
	}

	content := htmlBody[startIdx : startIdx+endIdx]
	
	return fmt.Sprintf(`<div class="w-full p-2 leading-8">%s</div>`, content)
}

func publishedAtRFC3339(dateStr string) string {
	if dateStr == "" {
		return ""
	}

	// Try parsing common date formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Format(time.RFC3339)
		}
	}

	return ""
}
