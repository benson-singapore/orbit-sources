package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"

	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const apiBase = "https://api.opennana.com/api/prompts"

func main() {
	sdk.Run(&OpenNanaPlugin{})
}

type OpenNanaPlugin struct{}

func (p *OpenNanaPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/opennana/list" || strings.HasPrefix(req.Route, "/opennana/list"):
		return fetchList(req)
	case strings.HasPrefix(req.Route, "/opennana/detail"):
		return fetchDetail(req)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

type listResponse struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
	Data   struct {
		Items      []listItem `json:"items"`
		Pagination struct {
			Page       int  `json:"page"`
			Limit      int  `json:"limit"`
			Total      int  `json:"total"`
			TotalPages int  `json:"total_pages"`
			HasMore    bool `json:"has_more"`
			ItemsCount int  `json:"items_count"`
		} `json:"pagination"`
	} `json:"data"`
}

type listItem struct {
	ID         int    `json:"id"`
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	MediaType  string `json:"media_type"`
	AccessType int    `json:"access_type"`
	PaidPoints int    `json:"paid_points"`
	CoverImage string `json:"cover_image"`
	IsSponsor  bool   `json:"_is_sponsor"`
}

type detailResponse struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
	Data   struct {
		ID          int            `json:"id"`
		Slug        string         `json:"slug"`
		Title       string         `json:"title"`
		Description string         `json:"description"`
		Model       string         `json:"model"`
		MediaType   string         `json:"media_type"`
		Images      []string       `json:"images"`
		Prompts     []promptEntry  `json:"prompts"`
		Tags        []string       `json:"tags"`
		VideoURLs   []string       `json:"video_urls"`
		AccessType  int            `json:"access_type"`
		PaidPoints  int            `json:"paid_points"`
		IsUnlocked  bool           `json:"is_unlocked"`
		SourceName  string         `json:"source_name"`
		SourceURL   string         `json:"source_url"`
	} `json:"data"`
}

type promptEntry struct {
	Text  string `json:"text"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

func fetchList(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	page := parseInt(req.Params["page"], 1)
	limit := 20

	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(limit))
	q.Set("sort", "reviewed_at")
	q.Set("order", "DESC")

	if v := strings.TrimSpace(req.Params["media_type"]); v != "" {
		q.Set("media_type", v)
	}
	if v := strings.TrimSpace(req.Params["model"]); v != "" {
		q.Set("model", v)
	}
	if v := strings.TrimSpace(req.Params["search"]); v != "" {
		q.Set("search", v)
	}
	if v := strings.TrimSpace(req.Params["tags"]); v != "" {
		q.Set("tags", v)
	}
	if v := strings.TrimSpace(req.Params["access_type"]); v != "" {
		q.Set("access_type", v)
	}

	rawURL := apiBase + "?" + q.Encode()
	body, status, err := host.HTTPGet(rawURL, map[string]string{
		"Accept":     "application/json",
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36",
	})
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d: %s", status, truncate(string(body), 200))
	}

	var resp listResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse list response: %w", err)
	}
	if resp.Status != 200 {
		return nil, fmt.Errorf("api error: %s", resp.Msg)
	}

	items := make([]sdk.FeedItem, 0, len(resp.Data.Items))
	for _, raw := range resp.Data.Items {
		if raw.IsSponsor || raw.Slug == "" {
			continue
		}
		title := strings.TrimSpace(raw.Title)
		if title == "" {
			title = raw.Slug
		}
		items = append(items, sdk.FeedItem{
			ID:    raw.Slug,
			Title: title,
			URL:   "https://opennana.com/awesome-prompt-gallery/" + raw.Slug,
			Cover: raw.CoverImage,
			Image: raw.CoverImage,
		})
	}

	channelTitle := "OpenNana"
	if v := strings.TrimSpace(req.Params["model"]); v != "" {
		channelTitle = "OpenNana · " + v
	}
	if v := strings.TrimSpace(req.Params["search"]); v != "" {
		channelTitle = "OpenNana · 搜索: " + v
	}

	result := &sdk.FeedResult{
		Title: channelTitle,
		Items: items,
	}

	if resp.Data.Pagination.HasMore {
		result.HasMore = true
		result.Next = copyParams(req.Params)
		result.Next["page"] = strconv.Itoa(page + 1)
	}

	return result, nil
}

func fetchDetail(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	slug := strings.TrimSpace(req.Params["slug"])
	if slug == "" {
		return nil, fmt.Errorf("missing slug parameter")
	}

	rawURL := apiBase + "/" + url.PathEscape(slug)
	body, status, err := host.HTTPGet(rawURL, map[string]string{
		"Accept":     "application/json",
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36",
	})
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d: %s", status, truncate(string(body), 200))
	}

	var resp detailResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse detail response: %w", err)
	}
	if resp.Status != 200 {
		return nil, fmt.Errorf("api error: %s", resp.Msg)
	}

	d := resp.Data
	title := strings.TrimSpace(d.Title)
	if title == "" {
		title = d.Slug
	}

	cover := ""
	if len(d.Images) > 0 {
		cover = d.Images[0]
	}

	item := sdk.FeedItem{
		ID:      d.Slug,
		Title:   title,
		URL:     "https://opennana.com/awesome-prompt-gallery/" + d.Slug,
		Cover:   cover,
		Image:   cover,
		Summary: formatPrompts(d.Prompts),
		Content: buildDetailContent(d.Images, d.VideoURLs, d.Prompts, d.Model, d.SourceName, d.SourceURL),
		Author:  d.SourceName,
		Tags:    d.Tags,
	}

	return &sdk.FeedResult{
		Title: title,
		Items: []sdk.FeedItem{item},
	}, nil
}

func buildDetailContent(images, videos []string, prompts []promptEntry, model, sourceName, sourceURL string) string {
	var b strings.Builder

	for _, img := range images {
		b.WriteString(`<figure><img src="`)
		b.WriteString(html.EscapeString(img))
		b.WriteString(`" alt=""/></figure>`)
	}

	for _, vid := range videos {
		b.WriteString(`<video controls src="`)
		b.WriteString(html.EscapeString(vid))
		b.WriteString(`"></video>`)
	}

	if model != "" {
		b.WriteString(`<p><strong>模型：</strong>`)
		b.WriteString(html.EscapeString(model))
		b.WriteString(`</p>`)
	}

	for _, p := range prompts {
		text := strings.TrimSpace(p.Text)
		if text == "" {
			continue
		}
		label := strings.TrimSpace(p.Label)
		if label == "" {
			label = strings.TrimSpace(p.Type)
		}
		if label != "" {
			b.WriteString(`<h3>`)
			b.WriteString(html.EscapeString(label))
			b.WriteString(`</h3>`)
		}
		b.WriteString(`<pre style="white-space:pre-wrap;">`)
		b.WriteString(html.EscapeString(text))
		b.WriteString(`</pre>`)
	}

	if sourceName != "" || sourceURL != "" {
		b.WriteString(`<p><strong>来源：</strong>`)
		if sourceURL != "" {
			b.WriteString(`<a href="`)
			b.WriteString(html.EscapeString(sourceURL))
			b.WriteString(`">`)
			if sourceName != "" {
				b.WriteString(html.EscapeString(sourceName))
			} else {
				b.WriteString(html.EscapeString(sourceURL))
			}
			b.WriteString(`</a>`)
		} else {
			b.WriteString(html.EscapeString(sourceName))
		}
		b.WriteString(`</p>`)
	}

	return b.String()
}

func formatPrompts(prompts []promptEntry) string {
	if len(prompts) == 0 {
		return ""
	}

	parts := make([]string, 0, len(prompts))
	for _, p := range prompts {
		text := strings.TrimSpace(p.Text)
		if text == "" {
			continue
		}
		label := strings.TrimSpace(p.Label)
		if label == "" {
			label = strings.TrimSpace(p.Type)
		}
		if label != "" {
			parts = append(parts, label+"\n"+text)
		} else {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n\n")
}

func copyParams(params map[string]string) map[string]string {
	out := make(map[string]string, len(params))
	for k, v := range params {
		out[k] = v
	}
	return out
}

func parseInt(raw string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
