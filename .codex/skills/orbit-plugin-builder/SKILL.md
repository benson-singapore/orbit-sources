---
name: orbit-plugin-builder
description: Build, scaffold, and test Orbit news/social feed plugins with complete manifest, pagination, and ABI compliance. Use when creating plugins from templates, implementing pagination with HasMore/Next, normalizing feed items, configuring channels and features, adding search/chapters/detail routes, managing variables/secrets, testing via make try/try-wasm/dev, debugging selectors, or packaging extension.orbit bundles. Covers full lifecycle: scaffolding, Go WASM build, manifest schema validation, route handling, feed normalization, pagination (offset/cursor/lastId), testing modes, and dist/ packaging.
---

# Orbit Plugin Builder

Complete workflow for developing Orbit feed plugins in Go + WASM with full schema compliance.

## Core Concepts

### Plugin ID & Discovery
- Location: `plugins/<category>/<id>/` (e.g. `plugins/news/zaobao`)
- Root Makefile auto-discovers `plugins/*/*/Makefile`
- ID pattern: `^[a-z0-9][a-z0-9_-]{1,63}$`

### Architecture
- **Runtime entry:** Go `main()` → `sdk.Run(&YourPlugin{})`
- **Transport:** stdin/stdout JSON (ABI v1)
- **Compilation:** `GOOS=wasip1 GOARCH=wasm go build` → `plugin.wasm`
- **Packaging:** Brotli compress + ZIP → `extension.orbit`

## Plugin Interface

### Implement Plugin

```go
package main
import sdk "github.com/orbit-tauri-tools/plugin-sdk"

type YourPlugin struct{}

func (p *YourPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
  switch {
  case req.Route == "/your/route/:param":
    return handleRoute(req.Params["param"])
  default:
    return nil, fmt.Errorf("unknown route: %s", req.Route)
  }
}

func main() {
  sdk.Run(&YourPlugin{})
}
```

### FetchRequest Fields

| Field | Type | Description |
|-------|------|-------------|
| `channelId` | string | Channel id from manifest |
| `route` | string | Route pattern with params resolved |
| `params` | object | Channel params + pagination params |
| `vars` | object | User-provided variables (from `config.variables`) |

### FeedResult Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | yes | Feed title |
| `description` | string | no | Feed description |
| `items` | FeedItem[] | yes | Feed items (empty OK) |
| `hasMore` | bool | no | True if more pages exist |
| `next` | map | no | Pagination params for next page |
| `tree` | TreeNode[] | no | Hierarchical categories |

### FeedItem Fields

| Field | Type | Note |
|-------|------|------|
| `id` | string | Unique within channel; becomes `{pluginId}:{channelId}:{id}` in DB |
| `title` | string | Required |
| `url` | string | Required; clickable link |
| `cover` / `image` | string | Thumbnail URL |
| `published_at` | RFC3339 | Timestamp (e.g. `2025-01-08T12:00:00Z`) |
| `summary` | string | Excerpt / preview |
| `content` | string | Full text or ProseMirror JSON |
| `author` | string | Author name |
| `tags` | []string | Categories / topics |
| `kind` | string | `"short"` or `"long"` for social; omit for articles |
| `author_avatar` | string | Author profile image URL |
| `author_handle` | string | Author username (e.g. `@handle`) |
| `stats` | SocialStats | `{likes, replies, restacks}` for social |
| `media` | []SocialMedia | Images, videos, links |
| `quote` | SocialQuote | Embedded repost/quote |

### Social Fields (mediaType: social)

**SocialStats:**
```json
{ "likes": 123, "replies": 45, "restacks": 12 }
```

**SocialMedia:**
```json
[
  { "type": "image", "url": "...", "thumbnail": "..." },
  { "type": "video", "url": "...", "playback_id": "..." },
  { "type": "link", "url": "...", "title": "..." }
]
```

**SocialQuote:**
```json
{
  "id": "...",
  "author": "...",
  "author_avatar": "...",
  "author_handle": "@...",
  "body": "...",
  "url": "..."
}
```

## Manifest Configuration

### Minimum Structure

```json
{
  "id": "plugin-id",
  "name": "Display Name",
  "version": "1.0.0",
  "mediaType": "article",
  "source": "wasm",
  "capabilities": ["feed"],
  "config": {
    "refreshInterval": 1800,
    "defaultChannel": "channel-id",
    "executionMode": "wasm",
    "channels": [
      {
        "id": "channel-id",
        "label": "频道名",
        "route": "/plugin/route/:param",
        "params": { "param": "value" },
        "status": "enabled",
        "features": {
          "feed": { "persist": true, "refresh": true, "limit": 100 }
        }
      }
    ],
    "wasm": {
      "entry": "plugin.wasm",
      "timeoutMs": 120000,
      "maxMemoryMB": 64
    }
  },
  "meta": {
    "description": "Source description",
    "icon": "text",
    "color": "bg-red-500",
    "logoText": "早",
    "marketCategory": "news",
    "categoryTag": "NEWS",
    "official": true
  }
}
```

### mediaType Options

| Value | Use case |
|-------|----------|
| `article` | News, blogs, articles |
| `social` | Twitter-like, Substack Notes |
| `video` | YouTube, TikTok |
| `audio` | Podcasts, music |
| `manga` | Comics, manga |
| `novel` | E-books, serialized text |
| `rating` | Reviews, ratings |
| `image` | Photo galleries |

### capabilities

Array; always includes `"feed"`. Optional: `"playback"` for video/audio.

### config.variables & secrets

```json
{
  "config": {
    "variables": {
      "api_key": {
        "type": "string",
        "label": "API Key",
        "description": "Your provider API key",
        "default": "",
        "secret": true
      }
    }
  }
}
```

At runtime, user-provided values arrive in `FetchRequest.Vars["api_key"]`.

> **Deprecated:** `config.secrets` — use `variables` with `"secret": true` instead.

## Channel Features

### feed — list persistence & refresh

```json
"features": {
  "feed": {
    "persist": true,
    "refresh": true,
    "limit": 100
  }
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `persist` | true | Store results in DB |
| `refresh` | true | Include in `refreshInterval` scheduling |
| `limit` | 100 | Max items retained; prune oldest by `published_at` |

### pagination — load more

```json
"pagination": {
  "style": "offset",
  "param": "page",
  "default": "1"
}
```

**Styles:**

| style | Example | Use case |
|-------|---------|----------|
| `offset` | `{ "page": "1" }` → `{ "page": "2" }` | Page numbers (1, 2, 3...) |
| `cursor` | `{ "pageToken": "abc123..." }` | Opaque pagination token (YouTube) |
| `lastId` | `{ "lastId": "14590200" }` | Last seen item id |

**Config fields:**

| Field | Type | Description |
|-------|------|-------------|
| `style` | enum | Required: `offset` \| `cursor` \| `lastId` |
| `param` | string | Param key (default: `page` for offset/cursor, `lastId` for lastId) |
| `default` | string | Value on scheduled refresh (e.g. `"1"`) |
| `idFrom` | string | For `lastId`: field to read id (default: `item.id`) |
| `sizeParam` | string | Optional: page size param key |
| `defaultSize` | integer | Optional: default page size |
| `carryParams` | []string | Optional: extra params to merge on load-more (e.g. `["seenIds"]` for dedup) |

**In code — return pagination:**

```go
page := parsePage(params["page"])
items, hasMore := fetchPagedList(section, page)

result := &sdk.FeedResult{
  Title: "...",
  Items: items,
}
if hasMore {
  result.HasMore = true
  result.Next = map[string]string{"page": strconv.Itoa(page + 1)}
}
return result, nil
```

**With carryParams:**

```go
result.Next = map[string]string{
  "page": strconv.Itoa(page + 1),
  "seenIds": strings.Join(seenIds, ","),
}
```

### detail — item resolver (two-level nav)

```json
"detail": {
  "route": "/plugin/detail/:id",
  "idParam": "id",
  "idFrom": "item.id",
  "persist": true
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `route` | required | Route to fetch full item content |
| `idParam` | `id` | Param key for item id |
| `idFrom` | `item.id` | Source field on feed item |
| `persist` | true | Write fetched content back to DB |

### search — user query

```json
"search": {
  "param": "query",
  "required": true
}
```

Typically paired with `feed: { "persist": false, "refresh": false }`.

### chapters — sub-list (three-level nav)

For manga episodes, TV episodes, novel chapters. When present, feed items open a sub-list instead of detail.

```json
"chapters": {
  "route": "/comic/:id/chapters",
  "idParam": "id",
  "label": "话数",
  "itemLabel": "话",
  "persist": true,
  "limit": 500,
  "pagination": { "style": "offset", "param": "page" },
  "detail": {
    "route": "/comic/:id/chapter/:chapterId",
    "idParam": "chapterId",
    "parentParam": "id",
    "persist": false
  }
}
```

## Build Workflow

### 1. Scaffold

```bash
cd /repo/root
cp -r plugins/news/zaobao plugins/news/mynews
cd plugins/news/mynews
```

Edit:
- `go.mod` — change `module mynews`
- `manifest.json` — id, name, routes, channels
- `main.go` — type name, routes, fetch logic
- `README.md` — source docs

Verify `go.mod` replace:
```
replace github.com/orbit-tauri-tools/plugin-sdk => ../../../sdk
```

### 2. Build & Package

```bash
cd /repo/root
make build PLUGIN=mynews        # → dist/mynews/plugin.wasm
make package PLUGIN=mynews      # → copy manifest, README
make orbit PLUGIN=mynews        # → dist/mynews/extension.orbit
```

### 3. Verify

```bash
make list                        # mynews should appear
make try PLUGIN=mynews           # native go run
make try-wasm PLUGIN=mynews      # wasmtime on WASM
make dev PLUGIN=mynews           # Orbit runtime (if running)
```

## Testing & Debugging

### make try (fastest)

```bash
make try PLUGIN=mynews CHANNEL=channel-id ROUTE=/path PARAMS='{"key":"val"}'
```

Returns JSON to stdout. Check:
- No errors
- `items` array populated
- Each item has `id`, `title`, `url`, `published_at`

### make try-wasm

```bash
make try-wasm PLUGIN=mynews
```

Requires `wasmtime`. Validates WASM build and binary execution.

### make dev (runtime)

```bash
make dev PLUGIN=mynews
```

Requires Orbit dev instance at `http://127.0.0.1:17890`. Tests full host integration.

### Debug Selectors

Compare browser HTML vs parsed output:

```bash
# 1. Open browser DevTools; inspect real page HTML
# 2. Add temp logging to main.go:
log.Printf("Found items: %d", len(items))
# 3. Rebuild and test
make try PLUGIN=mynews | jq '.data.items | length'
```

### Common Issues

| Problem | Cause | Fix |
|---------|-------|-----|
| Plugin not in `make list` | Missing/wrong Makefile path | Check `plugins/<cat>/<id>/Makefile` exists |
| Build WASM fails | go.mod replace broken | Verify `replace github.com/orbit-tauri-tools/plugin-sdk => ../../../sdk` |
| Empty items | Selector error or API changed | Compare DevTools HTML; test `goquery.Find()` |
| Pagination not working | Missing `HasMore`/`Next` or manifest config | Return `hasMore: true` + `next: {...}` in code; add `features.pagination` to manifest |
| `make try` hangs | Infinite loop or network timeout | Add timeout; check HTTP requests |
| `make dev` fails | Runtime not running | Start `make dev-go` in another terminal |

## Verification Checklist

Before packaging:

- [ ] Plugin in `make list`
- [ ] `make try` returns valid JSON with items array
- [ ] Items have `id`, `title`, `url`, `published_at`
- [ ] Pagination: `hasMore=true` → `next.page` increments
- [ ] Cover/image URLs are reachable
- [ ] Routes in code match manifest exactly
- [ ] No secrets hardcoded (use `config.variables`)
- [ ] `make test-native PLUGIN=id` passes
- [ ] `make orbit` produces `dist/id/extension.orbit`
- [ ] Manifest schema valid (validate against `schemas/manifest.wasm.schema.json`)

## File Templates

### main.go template

```go
package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

func main() {
	sdk.Run(&MyPlugin{})
}

type MyPlugin struct{}

const baseURL = "https://example.com"

var sectionMap = map[string]string{
	"home": "/api/home",
	"tech": "/api/tech",
}

func (p *MyPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/myplugin/list/:section":
		section := req.Params["section"]
		if section == "" {
			section = "home"
		}
		page := req.Params["page"]
		if page == "" {
			page = "1"
		}
		return fetchList(section, page)
	case req.Route == "/myplugin/detail/:id":
		id := req.Params["id"]
		return fetchDetail(id)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchList(section, pageStr string) (*sdk.FeedResult, error) {
	pageNum, _ := strconv.Atoi(pageStr)
	if pageNum < 1 {
		pageNum = 1
	}

	url := fmt.Sprintf("%s%s?page=%d", baseURL, sectionMap[section], pageNum)
	body, status, err := host.HTTPGet(url, nil)
	if err != nil || status != 200 {
		return nil, fmt.Errorf("fetch failed: %v", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	var items []sdk.FeedItem
	doc.Find("article").Each(func(_ int, el *goquery.Selection) {
		id, _ := el.Attr("data-id")
		title := el.Find("h2").Text()
		url, _ := el.Find("a").Attr("href")
		cover, _ := el.Find("img").Attr("src")

		if id != "" && title != "" {
			items = append(items, sdk.FeedItem{
				ID:          id,
				Title:       title,
				URL:         url,
				Cover:       cover,
				PublishedAt: "2025-01-08T00:00:00Z", // parse real timestamp
			})
		}
	})

	result := &sdk.FeedResult{
		Title:       fmt.Sprintf("My Plugin - %s", section),
		Description: "Article feed",
		Items:       items,
	}

	// Pagination
	if len(items) >= 20 {
		result.HasMore = true
		result.Next = map[string]string{"page": strconv.Itoa(pageNum + 1)}
	}

	return result, nil
}

func fetchDetail(id string) (*sdk.FeedResult, error) {
	url := fmt.Sprintf("%s/article/%s", baseURL, id)
	body, status, err := host.HTTPGet(url, nil)
	if err != nil || status != 200 {
		return nil, fmt.Errorf("detail fetch failed: %v", err)
	}

	// Parse and return full article content
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	content := doc.Find("article").Text()

	return &sdk.FeedResult{
		Title: "Article Detail",
		Items: []sdk.FeedItem{
			{
				ID:      id,
				Title:   "Article",
				Content: content,
			},
		},
	}, nil
}
```

### Makefile template

```makefile
PLUGIN_ID = myplugin
DIST_DIR  = ../../../dist/$(PLUGIN_ID)
OUTPUT    = $(DIST_DIR)/plugin.wasm

.PHONY: build package clean test-native

build:
	@mkdir -p $(DIST_DIR)
	GOOS=wasip1 GOARCH=wasm go build -o $(OUTPUT) .
	@echo "built $(OUTPUT)"

package: build
	cp manifest.json $(DIST_DIR)/
	cp README.md $(DIST_DIR)/
	@mkdir -p $(DIST_DIR)/assets

clean:
	rm -rf $(DIST_DIR)

test-native:
	echo '{"action":"fetch","data":{"channelId":"home","route":"/myplugin/list/:section","params":{"section":"home","page":"1"}}}' | go run .

CHANNEL ?= home
ROUTE ?= /myplugin/list/:section
PARAMS ?= {"section":"home","page":"1"}
```

### go.mod template

```
module myplugin

go 1.22

require (
	github.com/PuerkitoBio/goquery v1.8.1
	github.com/orbit-tauri-tools/plugin-sdk v0.0.0
)

require (
	github.com/andybalholm/cascadia v1.3.1 // indirect
	golang.org/x/net v0.17.0 // indirect
)

replace github.com/orbit-tauri-tools/plugin-sdk => ../../../sdk
```

