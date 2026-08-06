package main

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var promoLinePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)扫描.*二维码`),
	regexp.MustCompile(`(?i)进群`),
	regexp.MustCompile(`(?i)微信群`),
	regexp.MustCompile(`(?i)技术交流群`),
	regexp.MustCompile(`(?i)公众号.?大厂面试`),
	regexp.MustCompile(`(?i)领取.*PDF`),
	regexp.MustCompile(`(?i)发红包`),
	regexp.MustCompile(`(?i)tctip`),
}

var blockedImagePathHints = []string{
	"wdsfsdfsmaster",
	"49160c2basfdsf",
	"image1.jpg",
	"qe222wewewqere",
	"master.png",
}

func cleanMarkdown(raw string) string {
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isPromoLine(trimmed) {
			continue
		}
		if isPromoImageMarkdown(trimmed) {
			continue
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func isPromoLine(line string) bool {
	if line == "" {
		return false
	}
	for _, re := range promoLinePatterns {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

func isPromoImageMarkdown(line string) bool {
	lower := strings.ToLower(line)
	if !strings.Contains(lower, "![](") && !strings.Contains(lower, "![") {
		return false
	}
	for _, hint := range blockedImagePathHints {
		if strings.Contains(lower, strings.ToLower(hint)) {
			return true
		}
	}
	return false
}

func cleanArticleHTML(root *goquery.Selection) *goquery.Selection {
	if root == nil || root.Length() == 0 {
		return root
	}

	root.Find("script, style, noscript, iframe").Remove()

	root.Find("p, li, blockquote").Each(func(_ int, s *goquery.Selection) {
		text := cleanText(s.Text())
		if isPromoLine(text) {
			s.Remove()
		}
	})

	root.Find("img[src]").Each(func(_ int, img *goquery.Selection) {
		src := strings.TrimSpace(img.AttrOr("src", ""))
		if isPromoImageURL(src) {
			img.Remove()
			return
		}
		if abs := absoluteAssetURL(src); abs != "" {
			img.SetAttr("src", abs)
		}
	})

	root.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		href := strings.TrimSpace(a.AttrOr("href", ""))
		if abs := rewriteDocLink(href); abs != "" {
			a.SetAttr("href", abs)
		}
	})

	root.Find("p, div").Each(func(_ int, s *goquery.Selection) {
		if strings.TrimSpace(s.Text()) == "" && s.Find("img, pre, code, table, video").Length() == 0 {
			s.Remove()
		}
	})

	return root
}

func isPromoImageURL(raw string) bool {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return false
	}
	for _, hint := range blockedImagePathHints {
		if strings.Contains(raw, strings.ToLower(hint)) {
			return true
		}
	}
	return false
}

func absoluteAssetURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	if strings.HasPrefix(raw, "/interviewGuide/") {
		return siteOrigin + raw
	}
	if strings.HasPrefix(raw, "/") {
		return siteOrigin + "/interviewGuide" + raw
	}
	if strings.HasPrefix(raw, "static/") {
		return baseURL + "/" + raw
	}
	return baseURL + "/" + strings.TrimPrefix(raw, "./")
}

func rewriteDocLink(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "#") {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	if strings.HasPrefix(raw, "mailto:") {
		return raw
	}

	path := raw
	fragment := ""
	if i := strings.Index(raw, "#"); i >= 0 {
		path = raw[:i]
		fragment = raw[i:]
	}
	path = strings.TrimPrefix(path, "./")

	switch {
	case path == "" && fragment != "":
		return fragment
	case strings.HasSuffix(strings.ToLower(path), ".md"):
		return docsifyURL(path) + fragment
	case strings.HasPrefix(path, "static/") || strings.HasPrefix(path, "/interviewGuide/static/"):
		return absoluteAssetURL(path)
	default:
		if u, err := url.Parse(raw); err == nil && u.Scheme != "" {
			return raw
		}
		return absoluteAssetURL(raw)
	}
}
