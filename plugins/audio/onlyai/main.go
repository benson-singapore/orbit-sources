package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	baseURL   = "https://onlyai.fm"
	defaultUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"

	defaultLatestSize      = 120
	defaultLiveSize        = 60
	maxLatestSize          = 160
	defaultGenrePageSize = 20
)

var (
	reJSONLDBlocks      = regexp.MustCompile(`(?s)application/ld\+json">(\{.*?\})</script>`)
	reMoreAboutSong     = regexp.MustCompile(`moreAboutSong\\":(?:null|\\"((?:\\.|[^\\"])*)\\",\\"trackPlayCount)`)
	reCoverImageURL     = regexp.MustCompile(`coverImageUrl\\":\\"([^\\"]+)\\"`)
	reMusicRecordingLD  = regexp.MustCompile(`"@type":"MusicRecording"[^}]*"audio":\{"@type":"AudioObject","contentUrl":"([^"]+)"`)
	reMusicRecordingLD2 = regexp.MustCompile(`"contentUrl":"(https://onlyai\.fm/media/audio/[^"]+)"`)
)

func main() {
	sdk.Run(&OnlyAIPlugin{})
}

type OnlyAIPlugin struct{}

type radioResponse struct {
	Queue     []radioTrack `json:"queue"`
	Track     *radioTrack  `json:"track"`
	SessionID string       `json:"sessionId"`
}

type radioTrack struct {
	ID           string      `json:"id"`
	Title        string      `json:"title"`
	TrackURL     string      `json:"trackUrl"`
	ArtistName   string      `json:"artistName"`
	Genre        string      `json:"genre"`
	Mood         string      `json:"mood"`
	Duration     int         `json:"duration"`
	AudioURL     string      `json:"audioUrl"`
	Provider     string      `json:"provider"`
	ProviderID   string      `json:"providerTrackId"`
	Source       trackSource `json:"source"`
	ArtistResolved string    `json:"artistNameResolved"`
}

type trackSource struct {
	SourceURL string `json:"sourceUrl"`
}

type tracksResponse struct {
	Tracks []apiTrack `json:"tracks"`
}

type apiTrack struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Genre     string    `json:"genre"`
	AudioURL  string    `json:"audioUrl"`
	CreatedAt string    `json:"createdAt"`
	Artist    apiArtist `json:"artist"`
}

type apiArtist struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type ldItemList struct {
	Type            string       `json:"@type"`
	NumberOfItems   int          `json:"numberOfItems"`
	ItemListElement []ldListItem `json:"itemListElement"`
}

type ldListItem struct {
	Position int            `json:"position"`
	Item     ldMusicRecording `json:"item"`
}

type ldMusicRecording struct {
	Type          string        `json:"@type"`
	Name          string        `json:"name"`
	URL           string        `json:"url"`
	Image         string        `json:"image"`
	Genre         string        `json:"genre"`
	Duration      string        `json:"duration"`
	DatePublished string        `json:"datePublished"`
	ByArtist      ldMusicGroup  `json:"byArtist"`
	Audio         *ldAudioObject `json:"audio"`
}

type ldMusicGroup struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ldAudioObject struct {
	ContentURL string `json:"contentUrl"`
}

func (p *OnlyAIPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/onlyai/latest" || strings.HasPrefix(req.Route, "/onlyai/latest"):
		return fetchLatest(req.Params)
	case req.Route == "/onlyai/live" || strings.HasPrefix(req.Route, "/onlyai/live"):
		return fetchLive(req.Params)
	case req.Route == "/onlyai/charts" || strings.HasPrefix(req.Route, "/onlyai/charts"):
		return fetchCharts()
	case req.Route == "/onlyai/genre" || strings.HasPrefix(req.Route, "/onlyai/genre"):
		return fetchGenre(req.Params)
	case req.Route == "/onlyai/detail" || strings.HasPrefix(req.Route, "/onlyai/detail"):
		pageURL := strings.TrimSpace(req.Params["url"])
		if pageURL == "" {
			return nil, fmt.Errorf("missing url parameter")
		}
		return fetchDetail(pageURL)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchLatest(params map[string]string) (*sdk.FeedResult, error) {
	size := latestSize(params)
	prevSize := prevLatestSize(params)
	apiURL := fmt.Sprintf("%s/api/radio/latest?size=%d", baseURL, size)

	body, status, err := httpGet(apiURL)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("latest api status %d", status)
	}

	var resp radioResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse latest response: %w", err)
	}
	if len(resp.Queue) == 0 {
		return nil, fmt.Errorf("empty latest queue")
	}

	items := tracksToItems(resp.Queue, true)
	if prevSize > 0 {
		if prevSize >= len(items) {
			items = nil
		} else {
			items = items[prevSize:]
		}
	}
	if len(items) == 0 {
		if prevSize > 0 {
			return &sdk.FeedResult{
				Title:       "OnlyAI.fm · 最新发布",
				Description: "没有更多曲目",
				Items:       []sdk.FeedItem{},
			}, nil
		}
		return nil, fmt.Errorf("empty latest queue")
	}

	result := &sdk.FeedResult{
		Title:       "OnlyAI.fm · 最新发布",
		Description: fmt.Sprintf("共 %d 首 AI 音乐", len(items)),
		Items:       items,
	}
	if len(resp.Queue) >= size && size < maxLatestSize {
		result.HasMore = true
		result.Next = map[string]string{
			"size":     strconv.Itoa(min(size+defaultLatestSize, maxLatestSize)),
			"prevSize": strconv.Itoa(size),
		}
	}
	return result, nil
}

func fetchLive(params map[string]string) (*sdk.FeedResult, error) {
	size := liveSize(params)
	apiURL := fmt.Sprintf("%s/api/radio/live-now?size=%d", baseURL, size)

	body, status, err := httpGet(apiURL)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("live api status %d", status)
	}

	var resp radioResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse live response: %w", err)
	}
	if len(resp.Queue) == 0 {
		return nil, fmt.Errorf("empty live queue")
	}

	items := tracksToItems(resp.Queue, true)
	return &sdk.FeedResult{
		Title:       "OnlyAI.fm · 直播热门",
		Description: fmt.Sprintf("共 %d 首正在热播的 AI 音乐", len(items)),
		Items:       items,
	}, nil
}

func fetchGenre(params map[string]string) (*sdk.FeedResult, error) {
	genre := strings.TrimSpace(params["genre"])
	if genre == "" {
		return nil, fmt.Errorf("missing genre parameter")
	}

	offset := genreOffset(params)
	apiURL := fmt.Sprintf("%s/api/tracks?genre=%s", baseURL, url.QueryEscape(genre))

	body, status, err := httpGet(apiURL)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("genre tracks api status %d", status)
	}

	var resp tracksResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse genre tracks response: %w", err)
	}
	if len(resp.Tracks) == 0 {
		return nil, fmt.Errorf("empty genre tracks for %s", genre)
	}
	if offset >= len(resp.Tracks) {
		return &sdk.FeedResult{
			Title:       "OnlyAI.fm · " + genre,
			Description: "没有更多曲目",
			Items:       []sdk.FeedItem{},
		}, nil
	}

	end := min(offset+defaultGenrePageSize, len(resp.Tracks))
	items := apiTracksToItems(resp.Tracks[offset:end])
	result := &sdk.FeedResult{
		Title:       "OnlyAI.fm · " + genre,
		Description: fmt.Sprintf("共 %d 首", len(items)),
		Items:       items,
	}
	if end < len(resp.Tracks) {
		result.HasMore = true
		result.Next = map[string]string{
			"genre":  genre,
			"offset": strconv.Itoa(end),
		}
	}
	return result, nil
}

func fetchCharts() (*sdk.FeedResult, error) {
	body, status, err := httpGet(baseURL + "/charts")
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("charts page status %d", status)
	}

	records, err := parseChartsJSONLD(string(body))
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("no chart tracks found")
	}

	items := make([]sdk.FeedItem, 0, len(records))
	for _, rec := range records {
		items = append(items, chartRecordToItem(rec))
	}

	return &sdk.FeedResult{
		Title:       "OnlyAI.fm · 榜单",
		Description: fmt.Sprintf("Top %d AI 音乐", len(items)),
		Items:       items,
	}, nil
}

func fetchDetail(pageURL string) (*sdk.FeedResult, error) {
	absPage := absURL(pageURL)
	body, status, err := httpGet(absPage)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("track page status %d", status)
	}
	htmlBody := string(body)

	audioURL := extractAudioURL(htmlBody)
	if audioURL == "" {
		return nil, fmt.Errorf("audio url not found for %s", absPage)
	}

	title := extractMetaTitle(htmlBody)
	if title == "" {
		title = "OnlyAI.fm Track"
	}
	artist := extractArtistFromPage(htmlBody)
	cover := extractCoverFromPage(htmlBody)
	about := extractMoreAboutSong(htmlBody)
	genre, durationLabel := extractGenreDurationFromLD(htmlBody)

	summary := joinNonEmpty(" · ", artist, genre, durationLabel)
	if about != "" {
		if summary != "" {
			summary += "\n\n" + about
		} else {
			summary = about
		}
	}

	tags := []string{}
	if genre != "" {
		tags = append(tags, genre)
	}

	item := sdk.FeedItem{
		ID:          trackIDFromURL(absPage),
		Title:       title,
		URL:         audioURL,
		Summary:     summary,
		Content:     buildAudioPlayerHTML(title, artist, audioURL, cover, about),
		Author:      artist,
		Cover:       cover,
		Image:       cover,
		Tags:        tags,
		PublishedAt: extractPublishedAt(htmlBody),
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: "OnlyAI.fm · 播放",
		Items:       []sdk.FeedItem{item},
	}, nil
}

func apiTracksToItems(tracks []apiTrack) []sdk.FeedItem {
	items := make([]sdk.FeedItem, 0, len(tracks))
	for _, track := range tracks {
		if strings.TrimSpace(track.ID) == "" {
			continue
		}
		items = append(items, apiTrackToItem(track))
	}
	return items
}

func apiTrackToItem(track apiTrack) sdk.FeedItem {
	artist := strings.TrimSpace(track.Artist.Name)

	tags := []string{}
	if track.Genre != "" {
		tags = append(tags, track.Genre)
	}

	return sdk.FeedItem{
		ID:          track.ID,
		Title:       strings.TrimSpace(track.Title),
		URL:         absURL(track.AudioURL),
		Summary:     joinNonEmpty(" · ", artist, track.Genre),
		Author:      artist,
		Tags:        tags,
		PublishedAt: strings.TrimSpace(track.CreatedAt),
	}
}

func tracksToItems(tracks []radioTrack, directAudio bool) []sdk.FeedItem {
	items := make([]sdk.FeedItem, 0, len(tracks))
	seen := make(map[string]bool)
	for _, track := range tracks {
		if track.ID == "" || seen[track.ID] {
			continue
		}
		seen[track.ID] = true
		items = append(items, trackToItem(track, directAudio))
	}
	return items
}

func trackToItem(track radioTrack, directAudio bool) sdk.FeedItem {
	artist := strings.TrimSpace(track.ArtistName)
	if artist == "" {
		artist = strings.TrimSpace(track.ArtistResolved)
	}

	cover := coverFromSource(track.Source.SourceURL)
	pageURL := absURL(track.TrackURL)
	audioURL := absURL(track.AudioURL)

	itemURL := pageURL
	if directAudio && audioURL != "" {
		itemURL = audioURL
	}

	tags := uniqueNonEmpty(track.Genre, track.Mood)
	summary := joinNonEmpty(" · ", artist, track.Genre, formatDuration(track.Duration))

	return sdk.FeedItem{
		ID:          track.ID,
		Title:       strings.TrimSpace(track.Title),
		URL:         itemURL,
		Summary:     summary,
		Author:      artist,
		Cover:       cover,
		Image:       cover,
		Tags:        tags,
		PublishedAt: "",
	}
}

func chartRecordToItem(rec ldMusicRecording) sdk.FeedItem {
	artist := strings.TrimSpace(rec.ByArtist.Name)
	cover := absURL(rec.Image)
	pageURL := strings.TrimSpace(rec.URL)
	if pageURL == "" {
		pageURL = baseURL
	}

	tags := []string{}
	if rec.Genre != "" {
		tags = append(tags, rec.Genre)
	}

	summary := joinNonEmpty(" · ", artist, rec.Genre, formatISODuration(rec.Duration))

	item := sdk.FeedItem{
		ID:          trackIDFromURL(pageURL),
		Title:       strings.TrimSpace(rec.Name),
		URL:         pageURL,
		Summary:     summary,
		Author:      artist,
		Cover:       cover,
		Image:       cover,
		Tags:        tags,
		PublishedAt: strings.TrimSpace(rec.DatePublished),
	}

	if rec.Audio != nil && rec.Audio.ContentURL != "" {
		item.URL = absURL(rec.Audio.ContentURL)
	}

	return item
}

func parseChartsJSONLD(pageHTML string) ([]ldMusicRecording, error) {
	blocks := reJSONLDBlocks.FindAllStringSubmatch(pageHTML, -1)
	for _, block := range blocks {
		if len(block) < 2 {
			continue
		}
		var list ldItemList
		if err := json.Unmarshal([]byte(block[1]), &list); err != nil {
			continue
		}
		if list.Type != "ItemList" || len(list.ItemListElement) == 0 {
			continue
		}
		out := make([]ldMusicRecording, 0, len(list.ItemListElement))
		for _, el := range list.ItemListElement {
			if strings.TrimSpace(el.Item.Name) == "" {
				continue
			}
			out = append(out, el.Item)
		}
		if len(out) > 0 {
			return out, nil
		}
	}
	return nil, fmt.Errorf("charts ItemList not found")
}

func extractAudioURL(pageHTML string) string {
	blocks := reJSONLDBlocks.FindAllStringSubmatch(pageHTML, -1)
	for _, block := range blocks {
		if len(block) < 2 {
			continue
		}
		var rec ldMusicRecording
		if err := json.Unmarshal([]byte(block[1]), &rec); err != nil {
			continue
		}
		if rec.Type == "MusicRecording" && rec.Audio != nil && rec.Audio.ContentURL != "" {
			return absURL(rec.Audio.ContentURL)
		}
	}
	if m := reMusicRecordingLD.FindStringSubmatch(pageHTML); len(m) > 1 {
		return absURL(m[1])
	}
	if m := reMusicRecordingLD2.FindStringSubmatch(pageHTML); len(m) > 1 {
		return absURL(m[1])
	}
	return ""
}

func extractMoreAboutSong(pageHTML string) string {
	m := reMoreAboutSong.FindStringSubmatch(pageHTML)
	if len(m) < 2 || strings.TrimSpace(m[1]) == "" {
		return ""
	}
	return decodeEscapedJSONString(m[1])
}

func extractCoverFromPage(pageHTML string) string {
	if m := reCoverImageURL.FindStringSubmatch(pageHTML); len(m) > 1 {
		return absURL(m[1])
	}
	blocks := reJSONLDBlocks.FindAllStringSubmatch(pageHTML, -1)
	for _, block := range blocks {
		if len(block) < 2 {
			continue
		}
		var rec ldMusicRecording
		if err := json.Unmarshal([]byte(block[1]), &rec); err != nil {
			continue
		}
		if rec.Type == "MusicRecording" && rec.Image != "" {
			return absURL(rec.Image)
		}
	}
	return ""
}

func extractMetaTitle(pageHTML string) string {
	const marker = "<title>"
	start := strings.Index(pageHTML, marker)
	if start < 0 {
		return ""
	}
	start += len(marker)
	end := strings.Index(pageHTML[start:], "</title>")
	if end < 0 {
		return ""
	}
	title := strings.TrimSpace(pageHTML[start : start+end])
	title = strings.TrimSuffix(title, " · OnlyAI.fm")
	if idx := strings.Index(title, " by "); idx > 0 {
		return strings.TrimSpace(title[:idx])
	}
	return title
}

func extractArtistFromPage(pageHTML string) string {
	blocks := reJSONLDBlocks.FindAllStringSubmatch(pageHTML, -1)
	for _, block := range blocks {
		if len(block) < 2 {
			continue
		}
		var rec ldMusicRecording
		if err := json.Unmarshal([]byte(block[1]), &rec); err != nil {
			continue
		}
		if rec.Type == "MusicRecording" && rec.ByArtist.Name != "" {
			return rec.ByArtist.Name
		}
	}
	return ""
}

func extractGenreDurationFromLD(pageHTML string) (string, string) {
	blocks := reJSONLDBlocks.FindAllStringSubmatch(pageHTML, -1)
	for _, block := range blocks {
		if len(block) < 2 {
			continue
		}
		var rec ldMusicRecording
		if err := json.Unmarshal([]byte(block[1]), &rec); err != nil {
			continue
		}
		if rec.Type == "MusicRecording" {
			return rec.Genre, formatISODuration(rec.Duration)
		}
	}
	return "", ""
}

func extractPublishedAt(pageHTML string) string {
	blocks := reJSONLDBlocks.FindAllStringSubmatch(pageHTML, -1)
	for _, block := range blocks {
		if len(block) < 2 {
			continue
		}
		var rec ldMusicRecording
		if err := json.Unmarshal([]byte(block[1]), &rec); err != nil {
			continue
		}
		if rec.Type == "MusicRecording" {
			return strings.TrimSpace(rec.DatePublished)
		}
	}
	return ""
}

func coverFromSource(sourceURL string) string {
	const prefix = "COVER:"
	idx := strings.Index(sourceURL, prefix)
	if idx < 0 {
		return ""
	}
	return absURL(strings.TrimSpace(sourceURL[idx+len(prefix):]))
}

func buildAudioPlayerHTML(title, artist, audioURL, cover, about string) string {
	var sb strings.Builder
	sb.WriteString(`<article class="onlyai-player" style="color:#1f2937;line-height:1.6;">`)
	if cover != "" {
		sb.WriteString(`<div style="display:flex;gap:16px;align-items:flex-start;margin-bottom:16px;">`)
		sb.WriteString(`<img src="`)
		sb.WriteString(html.EscapeString(cover))
		sb.WriteString(`" alt="" style="width:96px;height:96px;object-fit:cover;border-radius:8px;flex-shrink:0;">`)
		sb.WriteString(`<div><h2 style="margin:0 0 8px;font-size:18px;">`)
		sb.WriteString(html.EscapeString(title))
		sb.WriteString(`</h2>`)
		if artist != "" {
			sb.WriteString(`<p style="margin:0;color:#6b7280;font-size:14px;">`)
			sb.WriteString(html.EscapeString(artist))
			sb.WriteString(`</p>`)
		}
		sb.WriteString(`</div></div>`)
	}
	sb.WriteString(`<audio controls preload="metadata" style="width:100%;display:block;">`)
	sb.WriteString(`<source src="`)
	sb.WriteString(html.EscapeString(audioURL))
	sb.WriteString(`" type="audio/mpeg">`)
	sb.WriteString(`</audio>`)
	if about != "" {
		sb.WriteString(`<p style="margin-top:16px;color:#4b5563;font-size:14px;white-space:pre-wrap;">`)
		sb.WriteString(html.EscapeString(about))
		sb.WriteString(`</p>`)
	}
	sb.WriteString(`</article>`)
	return sb.String()
}

func trackIDFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, baseURL)
	raw = strings.TrimPrefix(raw, "/")
	if raw == "" {
		return "track"
	}
	return strings.ReplaceAll(raw, "/", ":")
}

func formatDuration(seconds int) string {
	if seconds <= 0 {
		return ""
	}
	m := seconds / 60
	s := seconds % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func formatISODuration(iso string) string {
	iso = strings.TrimSpace(iso)
	if !strings.HasPrefix(iso, "PT") {
		return ""
	}
	iso = strings.TrimPrefix(iso, "PT")
	iso = strings.TrimSuffix(iso, "S")
	if iso == "" {
		return ""
	}
	if sec, err := strconv.Atoi(iso); err == nil {
		return formatDuration(sec)
	}
	return ""
}

func decodeEscapedJSONString(s string) string {
	wrapped := `"` + s + `"`
	var out string
	if err := json.Unmarshal([]byte(wrapped), &out); err == nil {
		return strings.TrimSpace(out)
	}
	s = strings.ReplaceAll(s, `\r\n`, "\n")
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	return strings.TrimSpace(s)
}

func absURL(raw string) string {
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
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	return baseURL + raw
}

func joinNonEmpty(sep string, parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, sep)
}

func uniqueNonEmpty(values ...string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func latestSize(params map[string]string) int {
	if params == nil {
		return defaultLatestSize
	}
	n, err := strconv.Atoi(strings.TrimSpace(params["size"]))
	if err != nil || n < 1 {
		return defaultLatestSize
	}
	if n > maxLatestSize {
		return maxLatestSize
	}
	return n
}

func prevLatestSize(params map[string]string) int {
	if params == nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(params["prevSize"]))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func genreOffset(params map[string]string) int {
	if params == nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(params["offset"]))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func liveSize(params map[string]string) int {
	if params == nil {
		return defaultLiveSize
	}
	n, err := strconv.Atoi(strings.TrimSpace(params["size"]))
	if err != nil || n < 1 {
		return defaultLiveSize
	}
	if n > 120 {
		return 120
	}
	return n
}

func httpGet(rawURL string) ([]byte, int, error) {
	return host.HTTPGet(rawURL, map[string]string{
		"User-Agent": defaultUA,
		"Accept":     "application/json, text/html, */*",
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
