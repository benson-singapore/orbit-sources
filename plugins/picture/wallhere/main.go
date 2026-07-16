package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	baseURL = "https://wallhere.com"
)

func main() {
	sdk.Run(&WallherePlugin{})
}

type WallherePlugin struct{}

func (p *WallherePlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/wallhere/list":
		return fetchList(req.Params, req.ChannelID)
	case req.Route == "/wallhere/search":
		return fetchSearch(req.Params)
	case strings.HasPrefix(req.Route, "/wallhere/detail/"):
		return fetchDetail(req.Route)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

type wallhereResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data string `json:"data"`
}

type WallpaperItem struct {
	ID              string
	Title           string
	ImageURL        string
	ImageURLHiRes   string
	Width           int
	Height          int
	WidthHiRes      int
	HeightHiRes     int
	Tags            string
	Username        string
	Likes           int
	Collects        int
}

func fetchList(params map[string]string, channelID string) (*sdk.FeedResult, error) {
	page := params["page"]
	if page == "" {
		page = "1"
	}

	pageNum, err := strconv.Atoi(page)
	if err != nil {
		pageNum = 1
	}

	var order string
	switch channelID {
	case "popular":
		order = "popular"
	case "random":
		order = "random"
	default:
		order = "latest"
	}

	// Build URL
	reqURL := fmt.Sprintf("%s/zh/wallpapers?order=%s&page=%d&format=json", baseURL, order, pageNum)
	if color := params["color"]; color != "" {
		reqURL += "&color=" + url.QueryEscape(color)
	}
	if nsfw := params["NSFW"]; nsfw != "" {
		reqURL += "&NSFW=" + url.QueryEscape(nsfw)
	}

	// Fetch
	body, statusCode, err := host.HTTPGet(reqURL, map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	if statusCode != 200 {
		return nil, fmt.Errorf("HTTP error: %d", statusCode)
	}

	// Parse JSON response
	var wallResp wallhereResponse
	if err := json.Unmarshal(body, &wallResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if wallResp.Code != 200 {
		return nil, fmt.Errorf("API error: %s", wallResp.Msg)
	}

	// Parse HTML in data field
	items, err := parseHTML(wallResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Convert to feed items (使用高清URL)
	feedItems := make([]sdk.FeedItem, 0)
	for _, item := range items {
		imageURL := item.ImageURL
		if item.ImageURLHiRes != "" {
			imageURL = item.ImageURLHiRes
		}
		
		feedItems = append(feedItems, sdk.FeedItem{
			ID:     item.ID,
			Title:  item.Tags,
			Image:  imageURL,
			URL:    fmt.Sprintf("%s/zh/wallpaper/%s", baseURL, item.ID),
			Author: item.Username,
			Stats: &sdk.SocialStats{
				Likes:    item.Likes,
				Replies:  item.Collects,
			},
			Media: []sdk.SocialMedia{
				{
					Type:   "image",
					URL:    imageURL,
					Width:  item.WidthHiRes,
					Height: item.HeightHiRes,
				},
			},
		})
	}

	// Determine hasMore
	hasMore := pageNum < 70025 // Max pages from API

	return &sdk.FeedResult{
		Items:   feedItems,
		HasMore: hasMore,
		Next: map[string]string{
			"page": strconv.Itoa(pageNum + 1),
		},
	}, nil
}

func fetchSearch(params map[string]string) (*sdk.FeedResult, error) {
	query := params["q"]
	if query == "" {
		return nil, fmt.Errorf("search query required (q parameter)")
	}

	page := params["page"]
	if page == "" {
		page = "1"
	}

	pageNum, err := strconv.Atoi(page)
	if err != nil {
		pageNum = 1
	}

	// Build URL with search query
	reqURL := fmt.Sprintf("%s/zh/wallpapers?q=%s&page=%d&format=json", baseURL, url.QueryEscape(query), pageNum)

	// Fetch
	body, statusCode, err := host.HTTPGet(reqURL, map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	if statusCode != 200 {
		return nil, fmt.Errorf("HTTP error: %d", statusCode)
	}

	// Parse JSON response
	var wallResp wallhereResponse
	if err := json.Unmarshal(body, &wallResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if wallResp.Code != 200 {
		return nil, fmt.Errorf("API error: %s", wallResp.Msg)
	}

	// Parse HTML in data field
	items, err := parseHTML(wallResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Convert to feed items
	feedItems := make([]sdk.FeedItem, 0)
	for _, item := range items {
		imageURL := item.ImageURL
		if item.ImageURLHiRes != "" {
			imageURL = item.ImageURLHiRes
		}
		
		feedItems = append(feedItems, sdk.FeedItem{
			ID:     item.ID,
			Title:  item.Tags,
			Image:  imageURL,
			URL:    fmt.Sprintf("%s/zh/wallpaper/%s", baseURL, item.ID),
			Author: item.Username,
			Stats: &sdk.SocialStats{
				Likes:    item.Likes,
				Replies:  item.Collects,
			},
			Media: []sdk.SocialMedia{
				{
					Type:   "image",
					URL:    imageURL,
					Width:  item.WidthHiRes,
					Height: item.HeightHiRes,
				},
			},
		})
	}

	hasMore := pageNum < 70025

	return &sdk.FeedResult{
		Items:   feedItems,
		HasMore: hasMore,
		Next: map[string]string{
			"page": strconv.Itoa(pageNum + 1),
		},
	}, nil
}

func fetchDetail(route string) (*sdk.FeedResult, error) {
	// Extract ID from route: /wallhere/detail/{id}
	parts := strings.Split(route, "/")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid detail route: %s", route)
	}
	id := parts[3]

	// Fetch detail page
	reqURL := fmt.Sprintf("%s/zh/wallpaper/%s", baseURL, id)
	body, statusCode, err := host.HTTPGet(reqURL, map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch detail: %w", err)
	}
	if statusCode != 200 {
		return nil, fmt.Errorf("HTTP error: %d", statusCode)
	}

	// Parse HTML detail page
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract hi-res image info from og:image or main image
	var imageURL string
	var width, height int
	var title string

	// Try og:image first
	doc.Find("meta[property='og:image']").Each(func(i int, s *goquery.Selection) {
		if content, exists := s.Attr("content"); exists && content != "" {
			imageURL = content
		}
	})

	// Extract from structured data or title
	doc.Find("script[type='application/ld+json']").Each(func(i int, s *goquery.Selection) {
		jsonStr := s.Text()
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &data); err == nil {
			if contentURL, ok := data["contentUrl"].(string); ok && contentURL != "" {
				imageURL = contentURL
			}
			if nameVal, ok := data["name"].(string); ok {
				title = nameVal
			}
			if widthStr, ok := data["width"].(string); ok {
				if w, err := strconv.Atoi(strings.TrimSuffix(widthStr, "px")); err == nil {
					width = w
				}
			}
			if heightStr, ok := data["height"].(string); ok {
				if h, err := strconv.Atoi(strings.TrimSuffix(heightStr, "px")); err == nil {
					height = h
				}
			}
		}
	})

	// Fallback: try main image tag
	if imageURL == "" {
		doc.Find("img[itemprop='contentURL']").Each(func(i int, s *goquery.Selection) {
			if src, exists := s.Attr("src"); exists && src != "" {
				imageURL = src
			}
		})
	}

	// Extract author
	var author string
	doc.Find("a.photo-author").Each(func(i int, s *goquery.Selection) {
		author = s.Text()
	})

	if imageURL == "" {
		return nil, fmt.Errorf("could not extract image URL from detail page")
	}

	feedItems := []sdk.FeedItem{
		{
			ID:     id,
			Title:  title,
			Image:  imageURL,
			URL:    fmt.Sprintf("%s/zh/wallpaper/%s", baseURL, id),
			Author: author,
			Media: []sdk.SocialMedia{
				{
					Type:   "image",
					URL:    imageURL,
					Width:  width,
					Height: height,
				},
			},
		},
	}

	return &sdk.FeedResult{
		Items: feedItems,
	}, nil
}

func parseHTML(htmlStr string) ([]WallpaperItem, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlStr))
	if err != nil {
		return nil, err
	}

	items := []WallpaperItem{}

	doc.Find("div.item").Each(func(i int, s *goquery.Selection) {
		// Extract ID from button data-id
		button := s.Find("button[data-id]")
		id, exists := button.Attr("data-id")
		if !exists || id == "" {
			return
		}

		// Extract image info
		img := s.Find("img")
		imageURL, _ := img.Attr("src")
		alt, _ := img.Attr("alt")
		width, _ := img.Attr("width")
		height, _ := img.Attr("height")

		// Extract username
		username := s.Find("a.username").Text()

		// Extract likes and collects
		likes := 0
		collects := 0
		
		// Find collection count from button
		button.Find("em").Each(func(j int, elem *goquery.Selection) {
			collectStr := elem.Text()
			if n, err := strconv.Atoi(collectStr); err == nil {
				collects = n
			}
		})

		// Find likes count from ajax-like
		s.Find("a.ajax-like em").Each(func(j int, elem *goquery.Selection) {
			likeStr := elem.Text()
			if n, err := strconv.Atoi(likeStr); err == nil {
				likes = n
			}
		})

		w, _ := strconv.Atoi(width)
		h, _ := strconv.Atoi(height)

		// Generate hi-res URL (replace !s1 with !d for high resolution)
		imageURLHiRes := strings.Replace(imageURL, "!s1", "!d", 1)
		
		// Estimate hi-res dimensions (typical 16:9 or similar, use a reasonable default)
		// Most wallpapers are 1920x1080 or similar, but we'll use 0 and let client resize
		widthHiRes := w * 2
		heightHiRes := h * 2

		items = append(items, WallpaperItem{
			ID:              id,
			Title:           alt,
			ImageURL:        imageURL,
			ImageURLHiRes:   imageURLHiRes,
			Width:           w,
			Height:          h,
			WidthHiRes:      widthHiRes,
			HeightHiRes:     heightHiRes,
			Tags:            alt,
			Username:        username,
			Likes:           likes,
			Collects:        collects,
		})
	})

	return items, nil
}
