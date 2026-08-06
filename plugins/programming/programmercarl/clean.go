package main

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Promo / commercial hosts and path prefixes to strip from article bodies.
var blockedHostSuffixes = []string{
	"kamacoder.com",
	"notes.kamacoder.com",
	"toudi.kamacoder.com",
	"union-click.jd.com",
	"jd.com",
	"mp.weixin.qq.com",
}

// CDN hosts that host article diagrams — keep these even if under kamacoder.
var allowedImageHostSuffixes = []string{
	"file1.kamacoder.com",
	"file.kamacoder.com",
	"programmercarl.com",
}

var blockedPathPrefixes = []string{
	"/other/kstar",
	"/xunlian/",
	"/about/gongkaike",
	"/qita/algo_pdf",
	"/other/jianlizhuanye",
	"/other/jianlixiangmu",
}

var blockedHeadingKeywords = []string{
	"算法公开课",
	"知识星球",
	"训练营",
	"PDF下载",
	"PDF 下载",
	"加群",
	"简历辅导",
}

func cleanArticleContent(root *goquery.Selection) *goquery.Selection {
	if root == nil || root.Length() == 0 {
		return root
	}

	// Drop script/style/nav chrome inside content.
	root.Find("script, style, noscript, iframe, .line-numbers-wrapper, .header-anchor").Remove()

	// Remove promo banner paragraphs (centered image linking to kstar / camps).
	root.Find("p").Each(func(_ int, p *goquery.Selection) {
		if isPromoParagraph(p) {
			p.Remove()
		}
	})

	// Remove promotional heading sections (e.g. 「算法公开课」).
	root.Find("h1, h2, h3, h4").Each(func(_ int, h *goquery.Selection) {
		text := cleanText(h.Text())
		if !isBlockedHeading(text) {
			return
		}
		removeSection(h)
	})

	// Remove remaining promo anchors / images.
	root.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		href := strings.TrimSpace(a.AttrOr("href", ""))
		if isPromoURL(href) {
			// Keep link text if it looks like normal copy without nested media.
			if a.Find("img").Length() > 0 || cleanText(a.Text()) == "" {
				a.Remove()
				return
			}
			a.ReplaceWithHtml(cleanText(a.Text()))
		}
	})

	root.Find("img[src]").Each(func(_ int, img *goquery.Selection) {
		src := strings.TrimSpace(img.AttrOr("src", ""))
		if isPromoImageURL(src) {
			img.Remove()
			return
		}
		// Normalize relative image URLs.
		if abs := absoluteURL(src); abs != "" {
			img.SetAttr("src", abs)
		}
	})

	// Drop empty leftover wrappers.
	root.Find("p, div").Each(func(_ int, s *goquery.Selection) {
		if strings.TrimSpace(s.Text()) == "" && s.Find("img, pre, code, table, video").Length() == 0 {
			s.Remove()
		}
	})

	return root
}

func isPromoParagraph(p *goquery.Selection) bool {
	align := strings.ToLower(p.AttrOr("align", ""))
	style := strings.ToLower(p.AttrOr("style", ""))
	centered := align == "center" || strings.Contains(style, "text-align:center") || strings.Contains(style, "text-align: center")

	hasPromoLink := false
	p.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		if isPromoURL(a.AttrOr("href", "")) {
			hasPromoLink = true
		}
	})
	if hasPromoLink && (centered || p.Find("img").Length() > 0) {
		return true
	}

	text := cleanText(p.Text())
	if text == "" && p.Find("a[href] img").Length() > 0 {
		href := p.Find("a[href]").First().AttrOr("href", "")
		return isPromoURL(href)
	}
	return false
}

func isBlockedHeading(text string) bool {
	for _, kw := range blockedHeadingKeywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

func removeSection(heading *goquery.Selection) {
	next := heading.Next()
	heading.Remove()
	for next.Length() > 0 {
		tag := goquery.NodeName(next)
		if tag == "h1" || tag == "h2" || tag == "h3" || tag == "h4" {
			break
		}
		cur := next
		next = next.Next()
		cur.Remove()
	}
}

func isPromoURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "#") {
		return false
	}

	abs := absoluteURL(raw)
	u, err := url.Parse(abs)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Host)
	path := u.Path

	// Same-site promo paths.
	if host == "" || strings.Contains(host, "programmercarl.com") {
		for _, prefix := range blockedPathPrefixes {
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
		return false
	}

	// Keep leetcode / bilibili learning links.
	if strings.Contains(host, "leetcode.cn") || strings.Contains(host, "leetcode.com") ||
		strings.Contains(host, "bilibili.com") || strings.Contains(host, "github.com") {
		return false
	}

	for _, suffix := range blockedHostSuffixes {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			// Image CDN hosts are handled separately for <img>, not anchors.
			if isAllowedImageHost(host) {
				return false
			}
			return true
		}
	}
	return false
}

func isPromoImageURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	abs := absoluteURL(raw)
	u, err := url.Parse(abs)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Host)
	path := strings.ToLower(u.Path)

	// Wide promo banners often live under /i/web/ with kstar campaigns.
	if strings.Contains(path, "/i/web/") && (strings.Contains(path, "2026-03-05") || strings.Contains(abs, "kstar")) {
		return true
	}

	// Non-CDN kamacoder / third-party promo images.
	if host != "" && !isAllowedImageHost(host) {
		for _, suffix := range blockedHostSuffixes {
			if host == suffix || strings.HasSuffix(host, "."+suffix) {
				return true
			}
		}
	}
	return false
}

func isAllowedImageHost(host string) bool {
	host = strings.ToLower(host)
	for _, suffix := range allowedImageHostSuffixes {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}
