package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	apiBase     = "https://api.voronoiapp.com"
	siteBase    = "https://www.voronoiapp.com"
	cdnBase     = "https://cdn.voronoiapp.com/public"
	defaultSize = 20
	maxSize     = 50
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func main() {
	sdk.Run(&VoronoiPlugin{})
}

type VoronoiPlugin struct{}

var feedLabelMap = map[string]string{
	"latest":  "Latest",
	"popular": "Popular",
	"curated": "Editor's Pick",
	"home":    "Home",
}

var categoryLabelMap = map[string]string{
	"5":  "Economy",
	"12": "Markets",
	"19": "Technology",
	"2":  "Business",
	"6":  "Energy",
	"15": "Politics",
	"11": "Maps",
	"4":  "Demographics",
}

func (p *VoronoiPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/voronoi/list":
		return fetchList(req.Params)
	case req.Route == "/voronoi/creator":
		return fetchCreator(req.Params, req.Var("creator"))
	case strings.HasPrefix(req.Route, "/voronoi/detail"):
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchDetail(id)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

type voronoiPost struct {
	PID             int      `json:"pid"`
	UID             string   `json:"uid"`
	Headline        string   `json:"headline"`
	Link            string   `json:"link"`
	Description     string   `json:"description"`
	WebpImage       string   `json:"webp_image"`
	Image           string   `json:"image"`
	Dataset         string   `json:"dataset"`
	DatasetSettings string   `json:"dataset_settings"`
	Sources         []string `json:"sources"`
	Note            string   `json:"note"`
	Tags            []string `json:"tags"`
	Views           int      `json:"views"`
	Comments        int      `json:"comments"`
	Likes           int      `json:"likes"`
	PublishedAt     int64    `json:"published_at"`
	CategoryID      int      `json:"categoryId"`
	Author          struct {
		FirstName         string `json:"first_name"`
		LastName          string `json:"last_name"`
		PreferredUsername string `json:"preferred_username"`
		Sub               string `json:"sub"`
	} `json:"author"`
}

type datasetSettings struct {
	IsFirstRowHeader bool `json:"isFirstRowHeader"`
	IsLastRowFooter  bool `json:"isLastRowFooter"`
}

func fetchList(params map[string]string) (*sdk.FeedResult, error) {
	feed := strings.TrimSpace(params["feed"])
	if feed == "" {
		feed = "latest"
	}

	limit := parsePositiveInt(params["size"], defaultSize)
	if limit > maxSize {
		limit = maxSize
	}
	offset := parseNonNegativeInt(params["offset"], 0)

	query := url.Values{}
	query.Set("limit", strconv.Itoa(limit))
	query.Set("offset", strconv.Itoa(offset))

	title := "Voronoi"
	desc := "Data-driven visual stories from Visual Capitalist"

	switch feed {
	case "latest":
		query.Set("swimlane", "LATEST")
		title = "Voronoi · Latest"
	case "popular":
		query.Set("swimlane", "POPULAR")
		query.Set("tab", "POPULAR")
		query.Set("time_range", "MONTH")
		title = "Voronoi · Popular"
	case "curated":
		query.Set("swimlane", "CURATED")
		title = "Voronoi · Editor's Pick"
	case "home":
		query.Set("feed", "VORONOI")
		title = "Voronoi · Home"
	default:
		if categoryID, ok := categoryLabelMap[feed]; ok {
			query.Set("swimlane", "LATEST")
			query.Set("categoryIds", feed)
			title = "Voronoi · " + categoryID
		} else if strings.TrimSpace(params["query"]) != "" {
			query.Set("swimlane", "LATEST")
			query.Set("filter", strings.TrimSpace(params["query"]))
			title = "Voronoi · Search"
		} else {
			return nil, fmt.Errorf("unknown feed: %s", feed)
		}
	}

	if label := feedLabelMap[feed]; label != "" && feed != "home" {
		desc = label + " visualizations"
	}

	posts, err := fetchPosts(query)
	if err != nil {
		return nil, err
	}
	if len(posts) == 0 {
		return nil, fmt.Errorf("empty feed")
	}

	items := postsToItems(posts)
	result := &sdk.FeedResult{
		Title:       title,
		Description: desc,
		Items:       items,
		HasMore:     len(posts) == limit,
	}
	if result.HasMore {
		result.Next = map[string]string{
			"offset": strconv.Itoa(offset + len(posts)),
		}
	}
	return result, nil
}

func fetchCreator(params map[string]string, defaultCreator string) (*sdk.FeedResult, error) {
	uid := strings.TrimSpace(params["uid"])
	creator := strings.TrimSpace(params["creator"])
	if creator == "" {
		creator = strings.TrimSpace(defaultCreator)
	}
	if uid == "" && creator == "" {
		return nil, fmt.Errorf("creator or uid required (set channel param or plugin variable creator)")
	}

	authorUID := uid
	displayName := creator
	if authorUID == "" {
		resolved, name, err := resolveCreatorUID(creator)
		if err != nil {
			return nil, err
		}
		authorUID = resolved
		if name != "" {
			displayName = name
		}
	} else if displayName == "" {
		displayName = authorUID
	}

	limit := parsePositiveInt(params["size"], defaultSize)
	if limit > maxSize {
		limit = maxSize
	}
	offset := parseNonNegativeInt(params["offset"], 0)

	query := url.Values{}
	query.Set("limit", strconv.Itoa(limit))
	query.Set("offset", strconv.Itoa(offset))
	query.Set("author", authorUID)

	posts, err := fetchPosts(query)
	if err != nil {
		return nil, err
	}
	if len(posts) == 0 {
		return nil, fmt.Errorf("no posts for creator %s", displayName)
	}

	if displayName == "" && posts[0].Author.PreferredUsername != "" {
		displayName = posts[0].Author.PreferredUsername
	}

	items := postsToItems(posts)
	result := &sdk.FeedResult{
		Title:       "Voronoi · " + displayName,
		Description: fmt.Sprintf("Posts by %s on Voronoi", displayName),
		Items:       items,
		HasMore:     len(posts) == limit,
	}
	if result.HasMore {
		result.Next = map[string]string{
			"offset": strconv.Itoa(offset + len(posts)),
		}
	}
	return result, nil
}

func fetchDetail(id string) (*sdk.FeedResult, error) {
	query := url.Values{}
	query.Set("pid", id)

	body, status, err := host.HTTPGet(apiBase+"/post?"+query.Encode(), defaultHeaders())
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	var post voronoiPost
	if err := json.Unmarshal(body, &post); err != nil {
		return nil, fmt.Errorf("parse detail response: %w", err)
	}
	if post.PID == 0 || strings.TrimSpace(post.Headline) == "" {
		return nil, fmt.Errorf("post not found")
	}

	item := postToItem(post)
	datasetHTML, err := fetchDatasetHTML(post.Dataset, post.DatasetSettings)
	if err != nil {
		return nil, fmt.Errorf("fetch dataset: %w", err)
	}
	item.Content = buildContent(item.Cover, post.Description, datasetHTML, post.Sources, post.Note)

	return &sdk.FeedResult{
		Title:       item.Title,
		Description: item.Summary,
		Items:       []sdk.FeedItem{item},
	}, nil
}

func resolveCreatorUID(username string) (string, string, error) {
	username = strings.TrimSpace(strings.TrimPrefix(username, "@"))
	if username == "" {
		return "", "", fmt.Errorf("empty creator username")
	}

	candidates := []string{"@" + username, username}
	seen := make(map[string]struct{}, len(candidates))
	for _, filter := range candidates {
		if filter == "" {
			continue
		}
		if _, ok := seen[filter]; ok {
			continue
		}
		seen[filter] = struct{}{}

		query := url.Values{}
		query.Set("limit", "1")
		query.Set("offset", "0")
		query.Set("swimlane", "LATEST")
		query.Set("filter", filter)

		posts, err := fetchPosts(query)
		if err != nil {
			return "", "", err
		}
		if len(posts) == 0 {
			continue
		}
		uid := strings.TrimSpace(posts[0].Author.Sub)
		if uid == "" {
			uid = strings.TrimSpace(posts[0].UID)
		}
		if uid == "" {
			continue
		}
		name := strings.TrimSpace(posts[0].Author.PreferredUsername)
		return uid, name, nil
	}

	return "", "", fmt.Errorf("creator %q not found; use username from https://www.voronoiapp.com/creator/{username} or set uid directly", username)
}

func fetchPosts(query url.Values) ([]voronoiPost, error) {
	body, status, err := host.HTTPGet(apiBase+"/post?"+query.Encode(), defaultHeaders())
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d: %s", status, strings.TrimSpace(string(body)))
	}

	var posts []voronoiPost
	if err := json.Unmarshal(body, &posts); err != nil {
		return nil, fmt.Errorf("parse list response: %w", err)
	}
	return posts, nil
}

func postsToItems(posts []voronoiPost) []sdk.FeedItem {
	items := make([]sdk.FeedItem, 0, len(posts))
	for _, post := range posts {
		item := postToItem(post)
		if item.Title == "" {
			continue
		}
		items = append(items, item)
	}
	return items
}

func postToItem(post voronoiPost) sdk.FeedItem {
	cover := imageURL(post.WebpImage, post.Image)
	summary := stripHTML(post.Description)
	if len(summary) > 280 {
		summary = summary[:277] + "..."
	}

	item := sdk.FeedItem{
		ID:          strconv.Itoa(post.PID),
		Title:       strings.TrimSpace(post.Headline),
		URL:         postURL(post.Link),
		Summary:     summary,
		Author:      authorName(post.Author.FirstName, post.Author.LastName, post.Author.PreferredUsername),
		Cover:       cover,
		Image:       cover,
		PublishedAt: msToRFC3339(post.PublishedAt),
		Tags:        append([]string(nil), post.Tags...),
	}

	if post.Views > 0 {
		item.Tags = append(item.Tags, fmt.Sprintf("views %d", post.Views))
	}
	if post.Likes > 0 {
		item.Tags = append(item.Tags, fmt.Sprintf("likes %d", post.Likes))
	}
	if post.Comments > 0 {
		item.Tags = append(item.Tags, fmt.Sprintf("comments %d", post.Comments))
	}

	return item
}

func fetchDatasetHTML(datasetPath, settingsJSON string) (string, error) {
	datasetPath = strings.TrimSpace(datasetPath)
	if datasetPath == "" {
		return "", nil
	}

	body, status, err := host.HTTPGet(cdnBase+"/"+strings.TrimPrefix(datasetPath, "/"), defaultHeaders())
	if err != nil {
		return "", fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("http status %d", status)
	}

	settings := datasetSettings{IsFirstRowHeader: true}
	if strings.TrimSpace(settingsJSON) != "" {
		_ = json.Unmarshal([]byte(settingsJSON), &settings)
	}

	return csvToHTMLTable(body, settings), nil
}

func csvToHTMLTable(raw []byte, settings datasetSettings) string {
	reader := csv.NewReader(bytes.NewReader(raw))
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err != nil || len(rows) == 0 {
		return ""
	}

	if settings.IsLastRowFooter && len(rows) > 1 {
		footer := rows[len(rows)-1]
		rows = rows[:len(rows)-1]
		return renderTable(rows, settings.IsFirstRowHeader) + renderTableFooter(footer)
	}

	return renderTable(rows, settings.IsFirstRowHeader)
}

func renderTable(rows [][]string, header bool) string {
	if len(rows) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(`<section class="voronoi-dataset" style="margin-top:1.5rem;overflow-x:auto;">`)
	sb.WriteString(`<h3 style="margin:0 0 0.75rem;font-size:1rem;">Dataset</h3>`)
	sb.WriteString(`<table style="width:100%;border-collapse:collapse;font-size:0.9rem;">`)

	start := 0
	if header {
		sb.WriteString("<thead><tr>")
		for _, cell := range rows[0] {
			sb.WriteString(`<th style="border:1px solid #ddd;padding:0.5rem;text-align:left;background:#f5f5f5;">`)
			sb.WriteString(html.EscapeString(strings.TrimSpace(cell)))
			sb.WriteString("</th>")
		}
		sb.WriteString("</tr></thead>")
		start = 1
	}

	sb.WriteString("<tbody>")
	for _, row := range rows[start:] {
		sb.WriteString("<tr>")
		for _, cell := range row {
			sb.WriteString(`<td style="border:1px solid #ddd;padding:0.5rem;">`)
			sb.WriteString(html.EscapeString(strings.TrimSpace(cell)))
			sb.WriteString("</td>")
		}
		sb.WriteString("</tr>")
	}
	sb.WriteString("</tbody></table></section>")

	return sb.String()
}

func renderTableFooter(footer []string) string {
	if len(footer) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(`<table style="width:100%;border-collapse:collapse;font-size:0.85rem;margin-top:0.25rem;"><tfoot><tr>`)
	for _, cell := range footer {
		sb.WriteString(`<td style="border:1px solid #ddd;padding:0.5rem;background:#fafafa;color:#666;">`)
		sb.WriteString(html.EscapeString(strings.TrimSpace(cell)))
		sb.WriteString("</td>")
	}
	sb.WriteString("</tr></tfoot></table>")
	return sb.String()
}

func buildContent(cover, description, datasetHTML string, sources []string, note string) string {
	var sb strings.Builder
	if cover != "" {
		sb.WriteString(fmt.Sprintf(`<img src="%s" style="max-width:100%%;border-radius:8px;margin-bottom:1rem;"/>`, cover))
		sb.WriteString("\n")
	}
	if description != "" {
		sb.WriteString(description)
		sb.WriteString("\n")
	}
	if datasetHTML != "" {
		sb.WriteString(datasetHTML)
		sb.WriteString("\n")
	}
	if len(sources) > 0 {
		sb.WriteString(`<section style="margin-top:1.5rem;"><h3 style="margin:0 0 0.5rem;font-size:1rem;">Sources</h3><ul>`)
		for _, src := range sources {
			src = strings.TrimSpace(src)
			if src == "" {
				continue
			}
			escaped := html.EscapeString(src)
			sb.WriteString(`<li><a href="`)
			sb.WriteString(html.EscapeString(src))
			sb.WriteString(`" target="_blank" rel="noopener noreferrer">`)
			sb.WriteString(escaped)
			sb.WriteString("</a></li>")
		}
		sb.WriteString("</ul></section>\n")
	}
	if strings.TrimSpace(note) != "" {
		sb.WriteString(`<section style="margin-top:1rem;padding:0.75rem;background:#f9f9f9;border-radius:6px;font-size:0.9rem;color:#555;"><strong>Note:</strong> `)
		sb.WriteString(html.EscapeString(strings.TrimSpace(note)))
		sb.WriteString("</section>\n")
	}
	return sb.String()
}

func postURL(link string) string {
	link = strings.TrimSpace(link)
	if link == "" {
		return siteBase
	}
	return siteBase + "/" + strings.TrimPrefix(link, "/")
}

func imageURL(webp, fallback string) string {
	raw := strings.TrimSpace(webp)
	if raw == "" {
		raw = strings.TrimSpace(fallback)
	}
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	return cdnBase + "/" + strings.TrimPrefix(raw, "/")
}

func authorName(first, last, username string) string {
	name := strings.TrimSpace(strings.TrimSpace(first) + " " + strings.TrimSpace(last))
	if name != "" && name != "-" && name != "--" {
		return name
	}
	if username != "" {
		return username
	}
	return ""
}

func stripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return strings.Join(strings.Fields(s), " ")
}

func msToRFC3339(ms int64) string {
	if ms <= 0 {
		return time.Now().Format(time.RFC3339)
	}
	return time.UnixMilli(ms).Format(time.RFC3339)
}

func parsePositiveInt(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func parseNonNegativeInt(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func defaultHeaders() map[string]string {
	return map[string]string{
		"Accept":     "application/json",
		"User-Agent": "Mozilla/5.0 (compatible; OrbitVoronoiPlugin/1.0)",
	}
}
