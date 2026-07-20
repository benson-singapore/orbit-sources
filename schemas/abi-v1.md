# Orbit WASM Plugin ABI v1

## Transport

- One JSON request line on **stdin**, one JSON response line on **stdout**.
- WASM entry: WASI `_start`; plugin `main` calls `sdk.Run(handler)`.
- Manifest metadata lives only in **`manifest.json`** on disk (not returned by WASM).

## Request envelope

```json
{ "action": "fetch", "data": { "channelId": "trending", "route": "/juejin/trending", "params": {}, "vars": {} } }
```

### Request data fields

| Field | Type | Description |
|-------|------|-------------|
| `channelId` | string | Channel identifier from manifest |
| `route` | string | Route pattern with params resolved |
| `params` | object | Channel-specific parameters from manifest |
| `vars` | object | User-provided plugin variables (from `config.variables`) |
| `secrets` | object | **Deprecated.** Same as `vars`; kept for compatibility |

### Actions (v1)

| Action | Status |
|--------|--------|
| `fetch` | Required |
| `playback_list` | Optional; when `config.playback.managedBy` is `plugin` |
| `playback_get` | Optional; when `config.playback.managedBy` is `plugin` |
| `playback_put` | Optional; when `config.playback.managedBy` is `plugin` |
| `playback_delete` | Optional; when `config.playback.managedBy` is `plugin` |
| `manifest` | Dev-only self-check; runtime does not call |
| `parse` | Reserved for hybrid browser mode (`executionMode: "hybrid"`); see `schemas/browser-preview.md` |

## Response envelope

```json
{ "ok": true, "data": { "title": "…", "description": "…", "items": [] } }
```

```json
{ "ok": false, "error": "human readable message" }
```

## Feed item fields

| Field | Type | Maps to runtime |
|-------|------|-----------------|
| `id` | string | Prefixed as `{pluginId}:{channelId}:{id}` |
| `title` | string | required |
| `url` | string | `sourceUrl` |
| `summary` | string | optional |
| `content` | string | optional |
| `author` | string | optional |
| `cover` / `image` | string | `image` |
| `published_at` | RFC3339 | `publishedAt` unix |
| `tags` | string[] | optional |
| `kind` | string | optional; `short` \| `long` for social notes |
| `author_avatar` | string | optional; author profile image URL |
| `author_handle` | string | optional; e.g. `aaronparnas` |
| `stats` | object | optional; `{ likes, replies, restacks }` |
| `media` | array | optional; image / video / link attachments |
| `quote` | object | optional; embedded quote note |

### Social media fields (`mediaType: social`)

Used by Substack Notes and similar tweet-style plugins.

**`stats`:**

| Field | Type |
|-------|------|
| `likes` | integer |
| `replies` | integer |
| `restacks` | integer |

**`media[]` item:**

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | `image` \| `video` \| `link` |
| `url` | string | optional |
| `thumbnail` | string | optional |
| `title` | string | optional |
| `playback_id` | string | optional; Mux HLS playback id |
| `width` / `height` | integer | optional |

**`quote`:**

| Field | Type |
|-------|------|
| `id` | string |
| `author` | string |
| `author_avatar` | string |
| `author_handle` | string |
| `body` | string |
| `url` | string |

**`content` convention:** ProseMirror `body_json` JSON string for rich text rendering; fallback to plain `body` in `summary`.

### Item ID prefixes (storage)

| Layer | fullId pattern |
|-------|----------------|
| feed list | `{pluginId}:{channelId}:{item.id}` |
| chapters list | `{pluginId}:{channelId}:{parentId}:{chapterItem.id}` |

`channelId` is always the list channel id, even when fetching `chapters.route` or `chapters.detail.route`.

## Feed result fields

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | Feed title |
| `description` | string | optional |
| `items` | array | Feed items |
| `hasMore` | boolean | optional; whether more pages exist |
| `next` | object | optional; params fragment for the next page (e.g. `{ "lastId": "123" }` or `{ "page": "2", "seenIds": "…" }`) |
| `tree` | array | optional; hierarchical category nodes (see TreeNode) |

### TreeNode fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Node identifier |
| `title` | string | Display label |
| `url` | string | optional; source URL |
| `site` | string | optional; site key for routing |
| `children` | array | optional; nested TreeNode[] |

## Channel features

Channel capabilities are declared in `manifest.json` under `channels[].features`. Omit `features` (or use `{}`) for a standard subscription: list results are persisted and refreshed on `refreshInterval`.

> **Deprecated:** `type` and `dynamic` are replaced by `features`. Do not use in new plugins.

### `feed` — list storage and scheduling

```json
"features": {
  "feed": { "persist": true, "refresh": true }
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `persist` | `true` | Store list results in the database |
| `refresh` | `true` | Include in `refreshInterval` scheduled fetch |
| `limit` | `100` | Max items kept per channel; prune oldest when exceeded |

When `persist` is `true`:

- Scheduled refresh (pagination at default) → **incremental** insert (new ids only)
- Load more (pagination past default) → **append** older records
- After each insert → **prune** to `limit` (delete oldest by `published_at`, else `created_at`)

When `persist` is `false`, results are shown in the UI only (typical for search).

### `pagination` — paging

```json
"pagination": { "style": "offset", "param": "page", "default": "1" }
```

| `style` | Load more params | Example |
|---------|------------------|---------|
| `offset` | increment `param` | `{ "page": "2" }` |
| `cursor` | increment `param`; plugin maps to source token | YouTube `pageToken` |
| `lastId` | `param` = last item id | `{ "lastId": "14590200" }` |

| Field | Description |
|-------|-------------|
| `param` | Key in `params` (default: `page` for offset/cursor, `lastId` for lastId) |
| `default` | Value used on scheduled refresh (e.g. `"1"` or `""`) |
| `idFrom` | For `lastId`: source field on list item (default `item.id`) |
| `sizeParam` / `defaultSize` | Optional page size |
| `carryParams` | Optional string array; extra keys from `next` merged on load-more (e.g. `["seenIds"]` for recommendation dedup) |

Example with session carry:

```json
"params": { "page": "1", "seenIds": "" },
"pagination": {
  "style": "offset",
  "param": "page",
  "default": "1",
  "carryParams": ["seenIds"]
}
```

Fetch returns `next: { "page": "2", "seenIds": "id1,id2,..." }`; runtime merges `carryParams` on load-more.

完整约定见 [docs/pagination-seenids.md](../docs/pagination-seenids.md)。

### `search` — user query

```json
"search": { "param": "query", "required": true }
```

Usually combined with `feed: { "persist": false, "refresh": false }`.

Example:

```json
{
  "id": "search",
  "route": "/youtube/search/:query",
  "params": { "query": "", "page": "1" },
  "features": {
    "feed": { "persist": false, "refresh": false },
    "search": { "param": "query", "required": true },
    "pagination": { "style": "cursor", "param": "page", "default": "1" }
  }
}
```

### `detail` — item resolver (two-level navigation)

Not a separate channel. Runtime calls `detail.route` when the user opens a feed item. **Mutually exclusive with `chapters`.**

```json
"detail": {
  "route": "/hellogithub/detail/:id",
  "idParam": "id",
  "idFrom": "item.id",
  "persist": true
}
```

| `persist` | Behavior |
|-----------|----------|
| `true` | Fetch content and write back to the stored item |
| `false` | Display only, no DB write |

### `chapters` — sub-list (three-level navigation)

For serial content: manga episodes, TV episodes, novel chapters. When present, feed item clicks open a sub-list instead of `detail`.

```json
"chapters": {
  "route": "/comic/:id/chapters",
  "idParam": "id",
  "label": "话数",
  "persist": true,
  "limit": 500,
  "detail": {
    "route": "/comic/:id/chapter/:chapterId",
    "idParam": "chapterId",
    "parentParam": "id",
    "persist": false
  }
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `route` | required | Sub-list route; parent id in params |
| `idParam` | `"id"` | Parent item id key |
| `label` / `itemLabel` | — | UI labels (话数/集数/章节) |
| `persist` | `true` | Cache sub-list per parent item |
| `limit` | `500` | Max chapters per parent when persisted |
| `detail` | — | Third-level fetch (same semantics as top-level `detail` + `parentParam`) |

**Navigation:**

```
feed → openChapters → chapters.route
     → openChapterDetail → chapters.detail.route
```

**FetchRequest examples:**

```json
{ "channelId": "guoman", "route": "/comic/doupo/chapters", "params": { "id": "doupo" } }
{ "channelId": "guoman", "route": "/comic/doupo/chapter/5", "params": { "id": "doupo", "chapterId": "5" } }
```

### Feature matrix (current plugins)

| Plugin | features |
|--------|----------|
| juejin, zaobao, douban | — (defaults) |
| hellogithub | detail |
| youtube (subscription) | — |
| youtube (search) | feed(no persist) + search + pagination |
| 1x | pagination |
| baozi, gman, yilin | feed + chapters (+ pagination?) |

See [docs/方案/manifest-features-v2.md](../docs/方案/manifest-features-v2.md) and [schemas/features.schema.json](./features.schema.json) for the full design.

## Playback history

For consumable plugins (`mediaType: video` / `audio` / `article` / `novel` / `manga`), declare policy in **`config.playback`** (plugin-wide). Optional per-channel overrides live in `channels[].features.playback`.

```json
{
  "capabilities": ["feed", "playback"],
  "mediaType": "video",
  "config": {
    "playback": {
      "history": true,
      "progress": true,
      "mode": "video",
      "limit": 200,
      "managedBy": "runtime"
    }
  }
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `history` | `false` | Record which work/chapter the user consumed |
| `progress` | `false` | Record resume position within a chapter (semantics depend on `mode`) |
| `mode` | from `mediaType` | `video` \| `audio` \| `article` \| `novel` \| `manga` |
| `limit` | `200` | Max history entries per plugin; prune oldest |
| `managedBy` | `runtime` | `runtime`: client persists on reader/player events; `plugin`: WASM handles `playback_*` |

**`mode` defaults when omitted:** `video`→`video`, `audio`→`audio`, `article`→`article`, `novel`→`novel`, `manga`→`manga`.

**Do not use `feed.persist` for consumption history.** `feed` only controls list ingestion; search channels often set `feed.persist: false` but users still consume content.

### Progress modes

| mode | Use case | `progress` fields |
|------|----------|-------------------|
| `video` | 影视续播 | `position`, `duration` (seconds) |
| `audio` | 音频续播 | `position`, `duration` (seconds) |
| `article` | 文章 / 杂志阅读 | `offset` (0-based char index), optional `total`, optional `anchor` |
| `novel` | 长篇小说阅读 | `offset` (0-based char index), optional `total`, optional `anchor` |
| `manga` | 漫画阅读 | `page` (1-based image index), optional `totalPages` |

Top-level `position` / `duration` are **deprecated** shorthand for `video`/`audio`; prefer `progress`.

### `managedBy: runtime`

Runtime/UI writes history when the reader/player reports progress. Plugins do not implement `playback_*` actions. Keys are scoped by `pluginId` in the client database.

### `managedBy: plugin`

Plugin WASM **implementer** handles `playback_*` actions in the plugin entry (dispatch by `action`). Persist via host `storage_*` when needed. Runtime forwards consumption events to `playback_put` and reads via `playback_get` / `playback_list`.

> **Note:** orbit-sources SDK `sdk.Run` only handles `fetch`. Playback routing is the plugin author's responsibility when `managedBy` is `plugin`.

### Playback record

| Field | Type | Description |
|-------|------|-------------|
| `parentId` | string | Feed item id (work / series / album) |
| `chapterId` | string | optional; episode / chapter / 话 id |
| `channelId` | string | optional; list channel where consumption started |
| `parentTitle` | string | optional; display title |
| `chapterTitle` | string | optional; chapter / episode title |
| `cover` | string | optional; cover image URL |
| `mode` | string | optional; `video` \| `audio` \| `article` \| `novel` \| `manga` |
| `progress` | object | optional; mode-specific resume payload (see Progress modes) |
| `position` | number | optional; **deprecated**; use `progress.position` for video/audio |
| `duration` | number | optional; **deprecated**; use `progress.duration` for video/audio |
| `updatedAt` | integer | unix timestamp of last consumption |

**`progress` examples:**

```json
{ "mode": "video", "progress": { "position": 1234.5, "duration": 3600 } }
{ "mode": "article", "progress": { "offset": 4523, "total": 12000 } }
{ "mode": "manga", "progress": { "page": 12, "totalPages": 45 } }
```

Storage keys (host prefixes `pluginId` automatically):

| Key | Value |
|-----|-------|
| `playback:{parentId}` | JSON `PlaybackRecord` |
| `playback:_index` | JSON array of `parentId`, newest first |

### `playback_*` request / response

**List**

```json
{ "action": "playback_list", "data": { "limit": 50, "offset": 0 } }
```

```json
{
  "ok": true,
  "data": {
    "items": [
      {
        "parentId": "79534",
        "chapterId": "3",
        "mode": "video",
        "progress": { "position": 1234.5, "duration": 3600 },
        "updatedAt": 1719043200
      }
    ],
    "total": 1
  }
}
```

**Get**

```json
{ "action": "playback_get", "data": { "parentId": "79534" } }
```

Missing entry: `{ "ok": true, "data": null }`.

**Put — video**

```json
{
  "action": "playback_put",
  "data": {
    "record": {
      "parentId": "79534",
      "chapterId": "3",
      "mode": "video",
      "progress": { "position": 1234.5, "duration": 3600 },
      "updatedAt": 1719043200
    }
  }
}
```

**Put — novel**

```json
{
  "action": "playback_put",
  "data": {
    "record": {
      "parentId": "doupo",
      "chapterId": "42",
      "mode": "novel",
      "progress": { "offset": 1520, "total": 8000 },
      "updatedAt": 1719043200
    }
  }
}
```

**Put — manga**

```json
{
  "action": "playback_put",
  "data": {
    "record": {
      "parentId": "one-piece",
      "chapterId": "1095",
      "mode": "manga",
      "progress": { "page": 8, "totalPages": 18 },
      "updatedAt": 1719043200
    }
  }
}
```

**Delete**

```json
{ "action": "playback_delete", "data": { "parentId": "79534" } }
```

```json
{ "ok": true, "data": { "ok": true } }
```

See [docs/方案/playback.md](../docs/方案/playback.md) for client/runtime integration.

## Host module `orbit`

Imported by plugins (`//go:wasmimport orbit …`). Implemented by the Go runtime (wazero).

| Export | Signature | Description |
|--------|-----------|-------------|
| `http_request` | `(req_ptr, req_len, resp_ptr, resp_cap) -> u32` | JSON in/out; returns bytes written |
| `storage_get` | `(key_ptr, key_len, resp_ptr, resp_cap) -> u32` | Read plugin-scoped KV; JSON out |
| `storage_set` | `(key_ptr, key_len, val_ptr, val_len, resp_ptr, resp_cap) -> u32` | Write plugin-scoped KV |
| `storage_delete` | `(key_ptr, key_len, resp_ptr, resp_cap) -> u32` | Delete plugin-scoped KV |
| `log` | `(level_ptr, level_len, msg_ptr, msg_len)` | Debug logging |
| `now_unix` | `() -> i64` | Current unix time |

### `http_request` input JSON

```json
{ "method": "GET", "url": "https://…", "headers": {}, "body": "" }
```

### `http_request` output JSON

```json
{ "status": 200, "body": "…" }
```

Non-text responses (images, `binary/octet-stream`, etc.) use base64:

```json
{ "status": 200, "body_base64": "…" }
```

```json
{ "error": "message" }
```

Network uses manifest `config.userAgent` when set.

### `storage_get` output JSON

```json
{ "value_base64": "eyJwYXJlbnRJZCI6Ijc5NTM0In0=" }
```

```json
{ "missing": true }
```

```json
{ "error": "message" }
```

### `storage_set` / `storage_delete` output JSON

```json
{ "ok": true }
```

```json
{ "error": "message" }
```

## Security (runtime)

- Per-invocation timeout (`config.wasm.timeoutMs`, default 30000)
- Memory cap (`config.wasm.maxMemoryMB`, default 64)
- Response body limit 8 MiB per HTTP call

## Native development

Without WASM, build tags use real `net/http`:

```bash
cd orbit-sources/plugins/programming/juejin
echo '{"action":"fetch","data":{"channelId":"trending","route":"/juejin/trending","params":{},"secrets":{}}}' | go run .
```

For plugins requiring API keys:

```bash
cd orbit-sources/plugins/social_media/youtube
echo '{"action":"fetch","data":{"channelId":"test","route":"/youtube/channel/:channelId","params":{"channelId":"UCxxx"},"vars":{"apiKey":"YOUR_API_KEY"}}}' | go run .
```

## WASM build

```bash
cd orbit-sources/plugins/programming/juejin
make build   # -> dist/juejin/plugin.wasm
make package # -> dist/juejin/
```
