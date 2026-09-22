---
name: orbit-plugin-builder
description: Build, update, test, and package Orbit Go/WASM feed plugins. Use for scaffolding plugins, implementing SDK Fetch routes, normalizing article/social/media items, configuring manifest channels and features, adding pagination/search/detail/chapters/playback, handling user variables, debugging HTTP or HTML parsing, validating schema/ABI compliance, and producing extension.orbit packages.
---

# Orbit Plugin Builder

Develop plugins in `plugins/<category>/<id>/` as independent Go modules. Treat the repository schemas and ABI as authoritative; this skill is an execution guide, not a replacement for them.

## Source Of Truth

Read only the references needed for the task:

- Manifest and channel feature fields: [references/manifest-quick-ref.md](references/manifest-quick-ref.md), then `schemas/manifest.wasm.schema.json` and `schemas/features.schema.json` for exact constraints.
- JSON stdin/stdout ABI, item fields, routes, pagination, and host behavior: [references/abi-quick-ref.md](references/abi-quick-ref.md), then `schemas/abi-v1.md`.
- Playback and browser/hybrid details: the same references, then `schemas/playback.schema.json` or `schemas/browser-preview.md`.
- Repository workflow: `docs/development.md`, `docs/testing.md`, `docs/packaging.md`, root `Makefile`, and `scripts/try.sh`.

When a reference example conflicts with a schema, follow the schema. Avoid deprecated `channel.type`, `channel.dynamic`, and `config.secrets` in new manifests.

## Plugin Shape

Use this runtime entry point:

```go
package main

import (
	"fmt"

	sdk "github.com/orbit-tauri-tools/plugin-sdk"
)

type Plugin struct{}

func (p *Plugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch req.Route {
	case "/my-plugin/feed":
		return fetchFeed(req)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func main() { sdk.Run(&Plugin{}) }
```

`FetchRequest` contains `ChannelID`, `Route`, string `Params`, string `Vars`, and deprecated compatibility `Secrets`. Use `req.Var("key")`; it checks `Vars` first and then `Secrets`. `FeedResult` contains `Title`, `Description`, `Items`, optional `Tree`, `HasMore`, and `Next`.

For HTTP, use `github.com/orbit-tauri-tools/plugin-sdk/host`. `host.HTTPGet` is proxied by the Orbit host in WASM and uses native HTTP during local development. Check errors and non-2xx status codes, set only necessary headers, and normalize relative URLs before returning them.

## Scaffold Or Update

1. Inspect a nearby reference plugin with a similar source: news/article, social, picture, audio/video, reading, or manga.
2. Copy the complete plugin directory when scaffolding:

   ```bash
   cp -R plugins/news/zaobao plugins/news/my-plugin
   cd plugins/news/my-plugin
   ```

3. Update all identity-bearing files: directory, `manifest.json.id`, `Makefile` `PLUGIN_ID`, Go type names, `go.mod` module path if needed, routes, and `README.md`.
4. Keep the SDK replacement relative to the plugin directory: `../../../sdk`.
5. Ensure every enabled channel route has a matching `Fetch` branch, and every route parameter is read from `req.Params`.
6. Keep IDs stable and unique within a channel. Return source URLs as absolute URLs and use RFC3339 timestamps for `published_at`, especially for persisted feeds and ordering.

Do not hardcode API keys, cookies, or tokens. Declare user inputs in `config.variables` and read them with `req.Var`.

## Manifest Workflow

Start from the repository schema. The top-level required keys are `id`, `name`, `version`, `source`, `capabilities`, `config`, and `meta`; `mediaType` is optional in the schema. `source` must be `wasm`, and `capabilities` must contain `feed`; add `playback` only when the plugin uses the playback contract.

`config` requires `channels` (at least one) and `wasm`. A channel requires `id`, `label`, and `route`. `params` values are strings. `features` may be omitted or `{}` for the default persisted/refreshed feed behavior.

Use this minimal shape, adding only fields required by the plugin:

```json
{
  "id": "my-plugin",
  "name": "My Plugin",
  "version": "1.0.0",
  "mediaType": "article",
  "source": "wasm",
  "capabilities": ["feed"],
  "config": {
    "refreshInterval": 1800,
    "defaultChannel": "home",
    "executionMode": "wasm",
    "channels": [
      {
        "id": "home",
        "label": "Home",
        "route": "/my-plugin/feed",
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
  "meta": { "description": "Example plugin" }
}
```

Variable definitions require `label`; supported fields are `description`, `required`, `secret`, and string `default`. Do not add `type`; it is not accepted by the current `variableDef` schema:

```json
"variables": {
  "apiKey": {
    "label": "API key",
    "description": "Provider key",
    "required": true,
    "secret": true
  }
}
```

Use `config.browser` and `executionMode` only as forward-compatible configuration. Browser/hybrid execution is Phase 3 preview and is not implemented in Phase 1; `parse` is a reserved ABI action.

## Channel Features

Features are declared under `channels[].features`. `features.detail` and `features.chapters` are mutually exclusive.

- `feed`: controls list persistence, scheduled refresh, and retained item limit. Search feeds normally use `persist: false` and `refresh: false`.
- `pagination`: requires `style` (`offset`, `cursor`, or `lastId`). Optional `param`, `default`, `idFrom`, `sizeParam`, `defaultSize`, and `carryParams` must match the source and request params.
- `search`: configures the query param, normally `{ "param": "query", "required": true }`.
- `detail`: two-level item resolver with required `route`, optional `idParam`, `idFrom`, and `persist`.
- `chapters`: three-level feed -> chapter list -> chapter detail. It requires `route`, can contain its own pagination, and its nested `detail` requires `route`; use `parentParam`/`parentFrom` when the source needs the parent ID.
- `playback`: optional per-channel override of plugin-level playback policy.

For chapters, the runtime scopes stored IDs as `{pluginId}:{channelId}:{parentId}:{chapterId}`. Keep chapter IDs stable and return chapter content or media in the nested detail route.

## Implement Pagination

Choose the style from the upstream API:

| Style | Runtime value | Plugin behavior |
|---|---|---|
| `offset` | page/offset number | Parse the value and increment it. |
| `cursor` | opaque token | Pass the returned source token unchanged in `Next`. |
| `lastId` | last item ID | Use the configured `idFrom` field and request older/newer items from that boundary. |

Return the same key configured by `pagination.param`:

```go
result := &sdk.FeedResult{Title: "My Feed", Items: items}
if sourceHasMore && len(items) > 0 {
	result.HasMore = true
	result.Next = map[string]string{"page": strconv.Itoa(page + 1)}
}
return result, nil
```

Do not set `HasMore` without a usable `Next`. For recommendation feeds that repeat items, declare `carryParams` such as `seenIds`, initialize the key in channel `params`, and return the accumulated value in `Next`; see `docs/pagination-seenids.md`.

Test the first and next request explicitly:

```bash
make try PLUGIN=my-plugin CHANNEL=home ROUTE=/my-plugin/feed PARAMS='{"page":"1"}'
make try PLUGIN=my-plugin CHANNEL=home ROUTE=/my-plugin/feed PARAMS='{"page":"2"}'
```

## Normalize Items

The SDK type has `ID`, `Title`, `URL`, `PublishedAt`, `Cover`, `Image`, `Summary`, `Content`, `Author`, `Tags`, and social fields. At minimum, return a stable `id`, non-empty `title`, and absolute `url`; return a valid RFC3339 `published_at` for ordinary feed items. Do not use placeholder timestamps when the source timestamp can be parsed.

For `mediaType: social`, use `kind` (`short` or `long`), `author_avatar`, `author_handle`, `stats`, `media`, and `quote` according to [references/abi-quick-ref.md](references/abi-quick-ref.md). For rich social text, `content` may contain a ProseMirror `body_json` JSON string; keep plain text in `summary` as a fallback.

For `Tree`, return recursive nodes with `id`, `title`, optional `url`/`site`, and `children` when the source exposes hierarchical navigation.

## Playback And Execution Modes

For video, audio, article, novel, or manga consumption, configure `config.playback` and add the `playback` capability when appropriate. `history` and `progress` are independent of `feed.persist`. The default owner is `managedBy: runtime`; the client persists events and the plugin only implements `fetch`.

Use `managedBy: plugin` only when the WASM entry explicitly dispatches `playback_list`, `playback_get`, `playback_put`, and `playback_delete` and persists through host storage as needed. The repository SDK `sdk.Run` currently handles `fetch` only, so playback-owned plugins need a custom ABI dispatcher or SDK support beyond the basic template.

Use the mode-specific `progress` shape: time position for video/audio, character offset for article/novel, and 1-based page for manga. Avoid deprecated top-level `position` and `duration` fields in new records.

## Build, Test, Package

Run from the repository root:

```bash
make list
make build PLUGIN=my-plugin
make test-native PLUGIN=my-plugin
make try PLUGIN=my-plugin CHANNEL=home ROUTE=/my-plugin/feed PARAMS='{}'
make package PLUGIN=my-plugin
make orbit PLUGIN=my-plugin
```

Build output is `dist/<id>/plugin.wasm`; packaging copies the manifest, README, and assets; `make orbit` creates `dist/<id>/extension.orbit`. Use `make try-wasm PLUGIN=my-plugin` when `wasmtime` is installed. Use `make dev PLUGIN=my-plugin` only with the Orbit Runtime running at the configured local URL; the helper installs/resyncs and refreshes the plugin.

For direct native testing inside a plugin directory:

```bash
echo '{"action":"fetch","data":{"channelId":"home","route":"/my-plugin/feed","params":{},"vars":{}}}' | go run .
```

The ABI is one JSON request line in and one response line out. A successful response is `{ "ok": true, "data": ... }`; failures are `{ "ok": false, "error": "..." }`.

## Debugging

When items are empty or a route fails:

1. Compare the actual HTTP response with the selector assumptions. Check status, redirects, content type, anti-bot pages, and whether the data is embedded JSON rather than server-rendered HTML.
2. Add temporary `log.Printf` calls around status, response length, selector counts, and rejected-item reasons. Remove noisy diagnostics before packaging.
3. Test selectors against a saved response or a small fixture with `goquery`; keep parsing and normalization separate so each can be inspected.
4. Verify route strings exactly match `manifest.json`, required params exist, and pagination defaults are handled when `Params` omits them.
5. Check that source IDs, URLs, and timestamps are not empty and that a full page is not being mistaken for an item list.

Common failures:

| Symptom | Check |
|---|---|
| Missing from `make list` | `plugins/<category>/<id>/Makefile` and plugin ID spelling. |
| WASM build failure | Go version, `go.mod`, and `replace ... => ../../../sdk`. |
| Empty feed | Response status/body, selectors, API shape, and item validation. |
| Load more loops | `HasMore`/`Next`, configured param name, and cursor/ID progression. |
| Runtime failure | Runtime health, package installation/resync, and manifest route/feature config. |

## Final Checklist

- [ ] Directory, manifest ID, Makefile ID, and routes agree.
- [ ] Manifest has all schema-required top-level/config/channel fields; no unsupported variable `type` fields.
- [ ] `source` is `wasm`; capabilities contains `feed`; deprecated fields are avoided.
- [ ] Every enabled route is handled and errors are explicit.
- [ ] Items have stable IDs, titles, absolute URLs, and valid timestamps where applicable.
- [ ] Pagination config, request params, `HasMore`, and `Next` agree; `carryParams` is tested when used.
- [ ] `detail` and `chapters` are not declared together.
- [ ] Playback ownership and progress semantics match the implementation.
- [ ] `make test-native`, `make try`, and, when available, `make try-wasm` pass.
- [ ] `make package` and `make orbit` produce the expected `dist/<id>/` artifacts.
- [ ] Validate JSON with `jq empty` and inspect against `schemas/manifest.wasm.schema.json`; run the repository's schema validator when one is available.
