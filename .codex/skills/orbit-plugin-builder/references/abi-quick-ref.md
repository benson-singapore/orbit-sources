# ABI v1 Quick Reference

## Request-Response Flow

**Request (stdin):**
```json
{
  "action": "fetch",
  "data": {
    "channelId": "home",
    "route": "/plugin/list/:section",
    "params": { "section": "home", "page": "1" },
    "vars": { "api_key": "..." }
  }
}
```

**Success response (stdout):**
```json
{
  "ok": true,
  "data": {
    "title": "Feed Title",
    "description": "...",
    "items": [
      {
        "id": "item-id-1",
        "title": "Article Title",
        "url": "https://...",
        "cover": "https://...",
        "published_at": "2025-01-08T12:00:00Z",
        "summary": "...",
        "author": "...",
        "tags": ["news"]
      }
    ],
    "hasMore": true,
    "next": { "page": "2" }
  }
}
```

**Error response (stdout):**
```json
{
  "ok": false,
  "error": "human readable error message"
}
```

## FeedItem Fields

| Field | Required | Type | Normalized name in DB |
|-------|----------|------|----------------------|
| `id` | yes | string | (prefixed as `{pluginId}:{channelId}:{id}`) |
| `title` | yes | string | — |
| `url` | yes | string | `sourceUrl` |
| `published_at` | yes | RFC3339 | (converted to unix timestamp) |
| `cover` or `image` | no | string | `image` |
| `summary` | no | string | — |
| `content` | no | string | — |
| `author` | no | string | — |
| `tags` | no | []string | — |
| `kind` | no | string | (social: "short" or "long") |
| `author_avatar` | no | string | — |
| `author_handle` | no | string | — |
| `stats` | no | object | (social: likes, replies, restacks) |
| `media` | no | []object | (social: images, videos, links) |
| `quote` | no | object | (social: embedded quote/repost) |

## Social Fields

**`stats` example:**
```json
{
  "likes": 123,
  "replies": 45,
  "restacks": 12
}
```

**`media[]` example:**
```json
[
  {
    "type": "image",
    "url": "https://...",
    "thumbnail": "https://..."
  },
  {
    "type": "video",
    "url": "https://...",
    "playback_id": "mux-id"
  }
]
```

**`quote` example:**
```json
{
  "id": "original-post-id",
  "author": "John Doe",
  "author_avatar": "https://...",
  "author_handle": "johndoe",
  "body": "Original post text",
  "url": "https://..."
}
```

## Pagination

### offset style
```json
{
  "hasMore": true,
  "next": { "page": "2" }
}
```
Load-more request: `params: { "page": "2" }`

### cursor style
```json
{
  "hasMore": true,
  "next": { "pageToken": "abc123def456..." }
}
```
Load-more request: `params: { "pageToken": "abc123def456..." }`

### lastId style
```json
{
  "hasMore": true,
  "next": { "lastId": "14590200" }
}
```
Load-more request: `params: { "lastId": "14590200" }`

### With carryParams (dedup)
```json
{
  "hasMore": true,
  "next": {
    "page": "2",
    "seenIds": "id1,id2,id3,..."
  }
}
```
Manifest declares: `"carryParams": ["seenIds"]`
Runtime merges seenIds on load-more.

## Code Example: Implement Fetch

```go
func (p *MyPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
  // 1. Extract params
  section := req.Params["section"]
  page := req.Params["page"]
  apiKey := req.Var("api_key")  // from config.variables

  // 2. Validate
  if section == "" {
    return nil, fmt.Errorf("missing section")
  }

  // 3. Fetch data (HTTP, parse HTML, etc.)
  items, hasMore, err := fetchList(section, page, apiKey)
  if err != nil {
    return nil, err
  }

  // 4. Normalize items
  // Ensure: id, title, url, published_at present

  // 5. Build result
  result := &sdk.FeedResult{
    Title:       "My Feed",
    Description: "...",
    Items:       items,
  }

  // 6. Pagination
  if hasMore {
    pageNum, _ := strconv.Atoi(page)
    result.HasMore = true
    result.Next = map[string]string{
      "page": strconv.Itoa(pageNum + 1),
    }
  }

  return result, nil
}
```
