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

var (
	reMetaDescription = regexp.MustCompile(`name="description"\s+content="([^"]+)"`)
	reMetaKeywords    = regexp.MustCompile(`itemprop="keywords"\s+content="([^"]+)"`)
	reHeadline        = regexp.MustCompile(`itemprop="headline"\s+content="([^"]+)"`)
	reArticleAuthor   = regexp.MustCompile(`itemprop="author"[\s\S]*?itemprop="name"\s+content="([^"]+)"`)
	reArticleInfo     = regexp.MustCompile(`article_info:\{`)
	reCoverImage      = regexp.MustCompile(`cover_image:"((?:\\.|[^"\\])*)"`)
	reCtime           = regexp.MustCompile(`ctime:"(\d+)"`)
	reWebHTML         = regexp.MustCompile(`web_html_content:"((?:\\.|[^"\\])*)"`)
	reBriefContent    = regexp.MustCompile(`brief_content:"((?:\\.|[^"\\])*)"`)
	reBriefContentVar = regexp.MustCompile(`brief_content:[a-zA-Z_$][\w]*,`)
	reFirstImg        = regexp.MustCompile(`<img[^>]+src="([^"]+)"`)
	reStripTags       = regexp.MustCompile(`<[^>]+>`)
)

var pageFetchHeaders = map[string]string{
	"Accept":          "text/html,application/xhtml+xml",
	"Accept-Language": "zh-CN,zh;q=0.9",
	"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Cache-Control":   "no-cache",
}

type articlePageDetail struct {
	Title       string
	Author      string
	Content     string
	Summary     string
	Cover       string
	PublishedAt int64
	Tags        []string
}

func fetchDetail(articleID string) (*sdk.FeedResult, error) {
	detail, err := fetchArticlePageDetail(articleID)
	if err != nil {
		return nil, err
	}
	if detail.Content == "" {
		return nil, fmt.Errorf("article content not found")
	}

	summary := detail.Summary
	if summary == "" {
		summary = textSummaryFromHTML(detail.Content, 400)
	}

	title := strings.TrimSpace(detail.Title)
	if title == "" {
		title = "文章"
	}

	cover := detail.Cover
	image := cover
	if image == "" {
		image = firstImageInHTML(detail.Content)
	}

	publishedAt := ""
	if detail.PublishedAt > 0 {
		publishedAt = time.Unix(detail.PublishedAt, 0).Format(time.RFC3339)
	}

	item := sdk.FeedItem{
		ID:          articleID,
		Title:       title,
		URL:         "https://juejin.cn/post/" + articleID,
		Summary:     summary,
		Author:      detail.Author,
		Cover:       cover,
		Image:       image,
		Content:     detail.Content,
		PublishedAt: publishedAt,
		Tags:        detail.Tags,
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: summary,
		Items:       []sdk.FeedItem{item},
	}, nil
}

func fetchArticlePageDetail(articleID string) (*articlePageDetail, error) {
	articleID = strings.TrimSpace(articleID)
	if articleID == "" {
		return nil, errEmptyArticleID
	}
	url := "https://juejin.cn/post/" + articleID
	body, status, err := host.HTTPGet(url, pageFetchHeaders)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, errHTTPStatus(status)
	}
	return parseArticlePage(string(body)), nil
}

func parseArticlePage(html string) *articlePageDetail {
	detail := &articlePageDetail{}

	if m := reHeadline.FindStringSubmatch(html); len(m) > 1 {
		detail.Title = strings.TrimSpace(htmlUnescapeAttr(m[1]))
	}
	if m := reArticleAuthor.FindStringSubmatch(html); len(m) > 1 {
		detail.Author = strings.TrimSpace(htmlUnescapeAttr(m[1]))
	}

	if m := reMetaKeywords.FindStringSubmatch(html); len(m) > 1 {
		for _, t := range strings.Split(m[1], ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				detail.Tags = append(detail.Tags, t)
			}
		}
	}

	block := extractArticleInfoBlock(html)
	if block == "" {
		return detail
	}

	if m := reCoverImage.FindStringSubmatch(block); len(m) > 1 {
		detail.Cover = unescapeJSString(m[1])
	}
	if m := reCtime.FindStringSubmatch(block); len(m) > 1 {
		if ts, err := strconv.ParseInt(m[1], 10, 64); err == nil && ts > 0 {
			detail.PublishedAt = ts
		}
	}
	if m := reWebHTML.FindStringSubmatch(block); len(m) > 1 {
		detail.Content = unescapeJSString(m[1])
	}
	if m := reBriefContent.FindStringSubmatch(block); len(m) > 1 {
		detail.Summary = strings.TrimSpace(unescapeJSString(m[1]))
	} else if reBriefContentVar.MatchString(block) {
		if m := reMetaDescription.FindStringSubmatch(html); len(m) > 1 {
			detail.Summary = strings.TrimSpace(htmlUnescapeAttr(m[1]))
		} else if detail.Content != "" {
			detail.Summary = textSummaryFromHTML(detail.Content, 400)
		}
	}

	return detail
}

func extractArticleInfoBlock(html string) string {
	loc := reArticleInfo.FindStringIndex(html)
	if loc == nil {
		return ""
	}
	start := loc[1] - 1 // points at '{'
	return extractBalancedJSObject(html, start)
}

func extractBalancedJSObject(s string, start int) string {
	if start < 0 || start >= len(s) || s[start] != '{' {
		return ""
	}
	depth := 0
	inString := false
	escape := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inString {
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

func unescapeJSString(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			b.WriteByte(s[i])
			continue
		}
		if i+1 >= len(s) {
			b.WriteByte('\\')
			break
		}
		switch s[i+1] {
		case '"':
			b.WriteByte('"')
			i++
		case '\\':
			b.WriteByte('\\')
			i++
		case '/':
			b.WriteByte('/')
			i++
		case 'n':
			b.WriteByte('\n')
			i++
		case 'r':
			b.WriteByte('\r')
			i++
		case 't':
			b.WriteByte('\t')
			i++
		case 'u':
			if i+6 <= len(s) {
				if r, err := strconv.ParseUint(s[i+2:i+6], 16, 16); err == nil {
					b.WriteRune(rune(r))
					i += 5
					continue
				}
			}
			b.WriteByte('\\')
		default:
			b.WriteByte('\\')
		}
	}
	return b.String()
}

func htmlUnescapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	return s
}

func textSummaryFromHTML(html string, maxLen int) string {
	html = unescapeJSString(html)
	// Drop style blocks from NUXT web_html_content prefix.
	if idx := strings.Index(html, "</style>"); idx >= 0 {
		html = html[idx+len("</style>"):]
	}
	text := reStripTags.ReplaceAllString(html, "")
	text = strings.Join(strings.Fields(text), " ")
	if maxLen > 0 && len([]rune(text)) > maxLen {
		runes := []rune(text)
		text = string(runes[:maxLen]) + "…"
	}
	return text
}

func firstImageInHTML(html string) string {
	if m := reFirstImg.FindStringSubmatch(html); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}
