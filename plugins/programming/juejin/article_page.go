package main

import (
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
	Content     string
	Summary     string
	Cover       string
	PublishedAt int64
	Tags        []string
}

func enrichFeedItem(item sdk.FeedItem) sdk.FeedItem {
	detail, err := fetchArticlePageDetail(item.ID)
	if err != nil {
		return item
	}
	if detail.Content != "" {
		item.Content = detail.Content
	}
	if detail.Summary != "" {
		item.Summary = detail.Summary
	} else if item.Summary == "" && detail.Content != "" {
		item.Summary = textSummaryFromHTML(detail.Content, 400)
	}
	if detail.Cover != "" {
		item.Cover = detail.Cover
		item.Image = detail.Cover
	} else if detail.Content != "" {
		if img := firstImageInHTML(detail.Content); img != "" {
			item.Image = img
		}
	}
	if detail.PublishedAt > 0 {
		item.PublishedAt = time.Unix(detail.PublishedAt, 0).Format(time.RFC3339)
	}
	if len(detail.Tags) > 0 {
		item.Tags = detail.Tags
	}
	return item
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
