package main

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	homeURL       = "https://www.yilinzazhi.com/"
	coverIDPrefix = "cover:"
)

var (
	reYearSection = regexp.MustCompile(`(?is)<div\s+class=['"]year-section['"]>(.*?)</div>`)
	reYearTitle   = regexp.MustCompile(`(?is)<h2\s+class=['"]year-title['"]>([^<]+)</h2>`)
	reIssueLink   = regexp.MustCompile(`(?is)<li>\s*<a\s+href=['"]([^'"]+)['"][^>]*>([^<]+)</a>\s*</li>`)
	reArticleItem = regexp.MustCompile(`(?is)<div\s+class=['"]article-item['"]>\s*<div\s+class=['"]article-title['"]>\s*<a\s+href=['"]([^'"]+)['"][^>]*>([^<]+)</a>`)
	reCatalogSec  = regexp.MustCompile(`(?is)<section\s+class=['"]catalog-section['"]>\s*<h2\s+class=['"]catalog-section-title['"]>([^<]+)</h2>\s*<div\s+class=['"]article-list['"]>(.*?)</div>\s*</section>`)
	reArticleTitle = regexp.MustCompile(`(?is)<h1\s+class=['"]article-title['"]>([^<]+)</h1>`)
	reCategory    = regexp.MustCompile(`(?is)<span\s+class=['"]category['"]>([^<]+)</span>`)
	reAuthorAlt   = regexp.MustCompile(`(?is)<span\s+class=['"]author['"]>\s*<img[^>]*alt=['"]([^'"]*)['"]`)
	reArticleBody = regexp.MustCompile(`(?is)<div\s+class=['"]article-content[^'"]*['"][^>]*>(.*?)</div>`)
	reStripTags   = regexp.MustCompile(`(?s)<[^>]+>`)
)

func main() {
	sdk.Run(&YilinPlugin{})
}

type YilinPlugin struct{}

func (p *YilinPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/yilin/issues":
		return fetchIssues()
	case req.Route == "/yilin/chapters/:id":
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchChapters(id)
	case req.Route == "/yilin/detail/:id":
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchDetail(id)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchIssues() (*sdk.FeedResult, error) {
	body, status, err := httpGet(homeURL)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	items, tree, err := parseIssues(string(body))
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no issues found")
	}

	return &sdk.FeedResult{
		Title:       "意林杂志",
		Description: "意林杂志各年期数",
		Items:       items,
		Tree:        tree,
	}, nil
}

func parseIssues(pageHTML string) ([]sdk.FeedItem, []sdk.TreeNode, error) {
	items := make([]sdk.FeedItem, 0, 64)
	tree := make([]sdk.TreeNode, 0, 16)
	seen := make(map[string]bool)

	for _, section := range reYearSection.FindAllStringSubmatch(pageHTML, -1) {
		if len(section) < 2 {
			continue
		}
		sectionHTML := section[1]

		yearMatch := reYearTitle.FindStringSubmatch(sectionHTML)
		if len(yearMatch) < 2 {
			continue
		}
		yearTitle := cleanText(yearMatch[1])
		if yearTitle == "" {
			continue
		}

		yearNode := sdk.TreeNode{
			ID:    yearTitle,
			Title: yearTitle,
		}

		for _, link := range reIssueLink.FindAllStringSubmatch(sectionHTML, -1) {
			if len(link) < 3 {
				continue
			}
			href := strings.TrimSpace(link[1])
			title := cleanText(link[2])
			if href == "" || title == "" {
				continue
			}

			issueURL := absolutize(homeURL, href)
			id := itemIDFromURL(issueURL)
			if seen[id] {
				continue
			}
			seen[id] = true

			cover := issueCoverURL(href)
			item := sdk.FeedItem{
				ID:          id,
				Title:       title,
				URL:         issueURL,
				Cover:       cover,
				Image:       cover,
				PublishedAt: time.Now().Format(time.RFC3339),
			}
			items = append(items, item)
			yearNode.Children = append(yearNode.Children, sdk.TreeNode{
				ID:    id,
				Title: title,
				URL:   issueURL,
			})
		}

		if len(yearNode.Children) > 0 {
			tree = append(tree, yearNode)
		}
	}

	return items, tree, nil
}

func issueCoverURL(href string) string {
	href = strings.TrimSpace(href)
	href = strings.TrimPrefix(href, "/")
	parts := strings.Split(strings.TrimSuffix(href, "/index.html"), "/")
	if len(parts) < 2 {
		return ""
	}
	year, folder := parts[0], parts[1]
	if strings.HasPrefix(folder, "yl") {
		return homeURL + "upload/image/" + year + "/" + folder + ".jpg"
	}
	return homeURL + year + "/" + folder + "/" + folder + ".jpg"
}

func fetchChapters(issueID string) (*sdk.FeedResult, error) {
	issueURL, err := urlFromItemID(issueID)
	if err != nil {
		return nil, err
	}

	body, status, err := httpGet(issueURL)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	htmlBody := string(body)
	items := parseArticleList(htmlBody, issueURL)
	if len(items) == 0 {
		return nil, fmt.Errorf("no articles found")
	}

	if coverURL := coverURLFromIssueID(issueID); coverURL != "" {
		coverItem := sdk.FeedItem{
			ID:          coverItemID(issueID),
			Title:       "封面",
			URL:         coverURL,
			Cover:       coverURL,
			Image:       coverURL,
			PublishedAt: time.Now().Format(time.RFC3339),
		}
		items = append([]sdk.FeedItem{coverItem}, items...)
	}

	title := extractIssueTitle(htmlBody)
	if title == "" {
		title = "文章目录"
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: "本期文章目录",
		Items:       items,
	}, nil
}

func parseArticleList(htmlBody, issueURL string) []sdk.FeedItem {
	items := make([]sdk.FeedItem, 0, 64)
	seen := make(map[string]bool)
	issueBase := strings.TrimSuffix(issueURL, "/index.html")

	for _, section := range reCatalogSec.FindAllStringSubmatch(htmlBody, -1) {
		if len(section) < 3 {
			continue
		}
		category := cleanText(section[1])
		listHTML := section[2]

		for _, match := range reArticleItem.FindAllStringSubmatch(listHTML, -1) {
			if len(match) < 3 {
				continue
			}
			href := strings.TrimSpace(match[1])
			title := cleanText(match[2])
			if href == "" || title == "" {
				continue
			}

			articleURL := resolveIssueRelative(issueBase, href)
			id := itemIDFromURL(articleURL)
			if seen[id] {
				continue
			}
			seen[id] = true

			item := sdk.FeedItem{
				ID:          id,
				Title:       title,
				URL:         articleURL,
				PublishedAt: time.Now().Format(time.RFC3339),
			}
			if category != "" {
				item.Tags = []string{category}
			}
			items = append(items, item)
		}
	}

	if len(items) > 0 {
		return items
	}

	for _, match := range reArticleItem.FindAllStringSubmatch(htmlBody, -1) {
		if len(match) < 3 {
			continue
		}
		href := strings.TrimSpace(match[1])
		title := cleanText(match[2])
		if href == "" || title == "" {
			continue
		}

		articleURL := resolveIssueRelative(issueBase, href)
		id := itemIDFromURL(articleURL)
		if seen[id] {
			continue
		}
		seen[id] = true

		items = append(items, sdk.FeedItem{
			ID:          id,
			Title:       title,
			URL:         articleURL,
			PublishedAt: time.Now().Format(time.RFC3339),
		})
	}

	return items
}

func resolveIssueRelative(issueBase, href string) string {
	href = strings.TrimSpace(href)
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "/") {
		return absolutize(homeURL, href)
	}
	base := issueBase
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return base + href
}

func coverItemID(issueID string) string {
	return coverIDPrefix + issueID
}

func coverURLFromIssueID(issueID string) string {
	slash := strings.Index(issueID, "/")
	if slash <= 0 {
		return ""
	}
	path := strings.TrimSuffix(issueID[slash+1:], "/")
	return issueCoverURL(path + "/index.html")
}

func fetchCoverDetail(coverID string) (*sdk.FeedResult, error) {
	issueID := strings.TrimPrefix(coverID, coverIDPrefix)
	coverURL := coverURLFromIssueID(issueID)
	if coverURL == "" {
		return nil, fmt.Errorf("cover not found")
	}

	item := sdk.FeedItem{
		ID:      coverID,
		Title:   "封面",
		URL:     coverURL,
		Content: fmt.Sprintf(`<div class="cover-content"><img src="%s" alt="封面" style="max-width:100%%;"/></div>`, coverURL),
		Cover:   coverURL,
		Image:   coverURL,
	}

	return &sdk.FeedResult{
		Title:       "封面",
		Description: "杂志封面",
		Items:       []sdk.FeedItem{item},
	}, nil
}

func extractIssueTitle(htmlBody string) string {
	re := regexp.MustCompile(`(?is)<h1\s+class=['"]magazine-title['"]>([^<]+)</h1>`)
	if match := re.FindStringSubmatch(htmlBody); len(match) > 1 {
		return cleanText(match[1])
	}
	return extractPageTitle(htmlBody)
}

func fetchDetail(articleID string) (*sdk.FeedResult, error) {
	if strings.HasPrefix(articleID, coverIDPrefix) {
		return fetchCoverDetail(articleID)
	}

	articleURL, err := urlFromItemID(articleID)
	if err != nil {
		return nil, err
	}

	body, status, err := httpGet(articleURL)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	htmlBody := string(body)
	title, content, category, author := extractArticle(htmlBody)
	if title == "" {
		title = extractPageTitle(htmlBody)
	}
	if title == "" {
		title = "正文"
	}
	if content == "" {
		return nil, fmt.Errorf("article content not found")
	}

	item := sdk.FeedItem{
		ID:      articleID,
		Title:   title,
		URL:     articleURL,
		Content: content,
		Author:  author,
	}
	if category != "" {
		item.Tags = []string{category}
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: "意林文章正文",
		Items:       []sdk.FeedItem{item},
	}, nil
}

func extractArticle(htmlBody string) (title, content, category, author string) {
	if match := reArticleTitle.FindStringSubmatch(htmlBody); len(match) > 1 {
		title = cleanText(match[1])
	}

	if match := reCategory.FindStringSubmatch(htmlBody); len(match) > 1 {
		category = cleanText(match[1])
	}

	if match := reAuthorAlt.FindStringSubmatch(htmlBody); len(match) > 1 {
		alt := cleanText(match[1])
		if alt != "" && alt != "作者信息" {
			author = alt
		}
	}

	if match := reArticleBody.FindStringSubmatch(htmlBody); len(match) > 1 {
		bodyHTML := strings.TrimSpace(match[1])
		content = fmt.Sprintf(`<div class="article-content">%s</div>`, bodyHTML)
	}

	return title, content, category, author
}

func extractPageTitle(htmlBody string) string {
	re := regexp.MustCompile(`(?is)<title>([^<]+)</title>`)
	match := re.FindStringSubmatch(htmlBody)
	if len(match) < 2 {
		return ""
	}
	title := cleanText(match[1])
	title = strings.TrimSuffix(title, " - 意林杂志")
	title = strings.TrimPrefix(title, "意林杂志 - ")
	return title
}

func itemIDFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	host := u.Host
	path := strings.TrimPrefix(u.Path, "/")
	path = strings.TrimSuffix(path, "/")
	if u.RawQuery != "" {
		return host + "/" + path + "?" + u.RawQuery
	}
	return host + "/" + path
}

func urlFromItemID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("empty id")
	}
	if strings.HasPrefix(id, "http://") || strings.HasPrefix(id, "https://") {
		return id, nil
	}

	slash := strings.Index(id, "/")
	if slash <= 0 {
		return "", fmt.Errorf("invalid item id: %s", id)
	}
	host := id[:slash]
	path := id[slash+1:]
	if !strings.HasSuffix(path, ".html") && !strings.HasSuffix(path, "/") {
		path += "/index.html"
	}
	return "https://" + host + "/" + path, nil
}

func cleanText(s string) string {
	s = reStripTags.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func absolutize(base, href string) string {
	href = strings.TrimSpace(href)
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	ref, err := url.Parse(href)
	if err != nil {
		return href
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return href
	}
	return baseURL.ResolveReference(ref).String()
}

func httpGet(rawURL string) ([]byte, int, error) {
	body, status, err := host.HTTPGet(rawURL, map[string]string{
		"Accept": "text/html,application/xhtml+xml",
	})
	if err != nil {
		return nil, 0, fmt.Errorf("http get failed: %w", err)
	}
	return body, status, nil
}
