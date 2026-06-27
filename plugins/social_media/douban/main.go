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
	sdk.Run(&DoubanPlugin{})
}

type DoubanPlugin struct{}

type DoubanItem struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	OriginalTitle string `json:"original_title"`
	URL           string `json:"url"`
	Type          string `json:"type"`
	TypeName      string `json:"type_name"`
	Year          string `json:"year"`
	ReleaseDate   string `json:"release_date"`
	Pic           struct {
		Large  string `json:"large"`
		Normal string `json:"normal"`
	} `json:"pic"`
	Cover struct {
		URL string `json:"url"`
	} `json:"cover"`
	Rating struct {
		Value float64 `json:"value"`
		Count int     `json:"count"`
	} `json:"rating"`
	Rank         int    `json:"rank"`
	RankValue    int    `json:"rank_value"`
	CardSubtitle string `json:"card_subtitle"`
	Info         string `json:"info"`
	Comments     []struct {
		Comment string `json:"comment"`
	} `json:"comments"`
}

type DoubanResponse struct {
	SubjectCollectionItems []DoubanItem `json:"subject_collection_items"`
	Total                  int          `json:"total"`
}

func (p *DoubanPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	listType := req.Params["type"]
	if listType == "" {
		listType = "subject_real_time_hotest"
	}

	return fetchList(listType)
}

// Douban list types mapped to human-readable labels
var listLabels = map[string]string{
	"subject_real_time_hotest":  "实时热门书影音",
	"movie_showing":              "影院热映",
	"movie_real_time_hotest":     "实时热门电影",
	"tv_real_time_hotest":        "实时热门电视",
	"movie_weekly_best":          "一周口碑电影榜",
	"tv_chinese_best_weekly":     "华语口碑剧集榜",
	"tv_global_best_weekly":      "全球口碑剧集榜",
	"show_chinese_best_weekly":   "国内口碑综艺榜",
	"show_global_best_weekly":    "国外口碑综艺榜",
	"tv_domestic":                "热播新剧国产剧",
	"tv_american":                "热播新剧欧美剧",
	"tv_japanese":                "热播新剧日剧",
	"tv_korean":                  "热播新剧韩剧",
	"tv_animation":               "热播新剧动画",
	"book_fiction_hot_weekly":    "虚构类小说热门榜",
	"book_nonfiction_hot_weekly": "非虚构类小说热门榜",
	"music_single":               "热门单曲榜",
	"music_chinese":              "华语新碟榜",
}

func fetchList(listType string) (*sdk.FeedResult, error) {
	label, ok := listLabels[listType]
	if !ok {
		label = listType
	}

	url := fmt.Sprintf("https://m.douban.com/rexxar/api/v2/subject_collection/%s/items?playable=0&start=0&count=50", listType)

	body, status, err := host.HTTPGet(url, map[string]string{
		"Accept":     "application/json",
		"Referer":    fmt.Sprintf("https://m.douban.com/subject_collection/%s", listType),
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
	})
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	items, err := parseDoubanList(body)
	if err != nil {
		return nil, err
	}

	return &sdk.FeedResult{
		Title:       fmt.Sprintf("豆瓣 - %s", label),
		Description: "",
		Items:       items,
	}, nil
}

func parseDoubanList(body []byte) ([]sdk.FeedItem, error) {
	var response DoubanResponse

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(response.SubjectCollectionItems) == 0 {
		return nil, fmt.Errorf("empty list data")
	}

	feedItems := make([]sdk.FeedItem, 0, len(response.SubjectCollectionItems))

	for _, item := range response.SubjectCollectionItems {
		if strings.TrimSpace(item.Title) == "" {
			continue
		}

		// Get cover image from pic.large or cover.url (use original URLs)
		coverURL := item.Pic.Large
		if coverURL == "" {
			coverURL = item.Cover.URL
		}

		// Build summary from info and comments
		summary := item.Info
		if len(item.Comments) > 0 && item.Comments[0].Comment != "" {
			summary = item.Comments[0].Comment
		}

		// Build detailed content
		content := buildDetailedContent(item, coverURL)

		feedItem := sdk.FeedItem{
			ID:          item.ID,
			Title:       item.Title,
			URL:         item.URL,
			Summary:     summary,
			Cover:       coverURL,
			Image:       coverURL,
			Content:     content,
			PublishedAt: time.Now().Format(time.RFC3339),
		}

		if item.Rating.Value > 0 {
			feedItem.Tags = append(feedItem.Tags, fmt.Sprintf("评分: %.1f", item.Rating.Value))
		}

		if item.Rank > 0 {
			feedItem.Tags = append(feedItem.Tags, fmt.Sprintf("排名: #%d", item.Rank))
		}

		if strings.TrimSpace(item.CardSubtitle) != "" {
			feedItem.Tags = append(feedItem.Tags, item.CardSubtitle)
		}

		feedItems = append(feedItems, feedItem)
	}

	if len(feedItems) == 0 {
		return nil, fmt.Errorf("no valid items in list")
	}

	return feedItems, nil
}

func buildDetailedContent(item DoubanItem, coverURL string) string {
	var sb strings.Builder

	// 封面图
	if coverURL != "" {
		sb.WriteString(fmt.Sprintf("<img src=\"%s\" style=\"max-width: 100%%; border-radius: 8px; margin-bottom: 1rem;\"/>\n", coverURL))
	}

	// 类型和年份
	sb.WriteString("<div style=\"margin-bottom: 1rem;\">\n")
	if item.TypeName != "" {
		sb.WriteString(fmt.Sprintf("<p><strong>类型:</strong> %s</p>\n", item.TypeName))
	}
	if item.Year != "" {
		sb.WriteString(fmt.Sprintf("<p><strong>年份:</strong> %s", item.Year))
		if item.ReleaseDate != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", item.ReleaseDate))
		}
		sb.WriteString("</p>\n")
	}
	sb.WriteString("</div>\n")

	// 评分信息
	if item.Rating.Value > 0 {
		sb.WriteString(fmt.Sprintf("<div style=\"margin-bottom: 1rem;\">\n"))
		sb.WriteString(fmt.Sprintf("<p><strong>评分:</strong> %.1f/10", item.Rating.Value))
		if item.Rating.Count > 0 {
			sb.WriteString(fmt.Sprintf(" (%d人评分)", item.Rating.Count))
		}
		sb.WriteString("</p>\n")
		sb.WriteString("</div>\n")
	}

	// 排名
	if item.Rank > 0 {
		sb.WriteString(fmt.Sprintf("<div style=\"margin-bottom: 1rem; background-color: #f5f5f5; padding: 0.5rem; border-radius: 4px;\">\n"))
		sb.WriteString(fmt.Sprintf("<p style=\"margin: 0;\"><strong>实时排名: #%d</strong></p>\n", item.Rank))
		sb.WriteString("</div>\n")
	}

	// 详细信息 (来自 card_subtitle)
	if item.CardSubtitle != "" {
		sb.WriteString("<div style=\"margin-bottom: 1rem;\">\n")
		sb.WriteString("<p><strong>详细信息:</strong></p>\n")
		sb.WriteString(fmt.Sprintf("<p style=\"color: #666; line-height: 1.6;\">%s</p>\n", item.CardSubtitle))
		sb.WriteString("</div>\n")
	}

	// 描述/简介 (来自 info 字段或第一条评论)
	if item.Info != "" {
		sb.WriteString("<div style=\"margin-bottom: 1rem;\">\n")
		sb.WriteString("<p><strong>内容:</strong></p>\n")
		sb.WriteString(fmt.Sprintf("<p style=\"color: #666; line-height: 1.6;\">%s</p>\n", item.Info))
		sb.WriteString("</div>\n")
	}

	// 最新评论
	if len(item.Comments) > 0 && item.Comments[0].Comment != "" {
		sb.WriteString("<div style=\"margin-top: 1.5rem; padding-top: 1rem; border-top: 1px solid #eee;\">\n")
		sb.WriteString("<p><strong>热门评论:</strong></p>\n")
		sb.WriteString(fmt.Sprintf("<p style=\"color: #666; font-style: italic; line-height: 1.6;\">\"%s\"</p>\n", item.Comments[0].Comment))
		sb.WriteString("</div>\n")
	}

	return sb.String()
}


