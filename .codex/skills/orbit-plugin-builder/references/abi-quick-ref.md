# ABI v1 Quick Reference

The normative document is `schemas/abi-v1.md`. The transport is one JSON request line on stdin and one JSON response line on stdout. Manifest metadata is read from `manifest.json`, not returned by the WASM program.

## Request And Response

```json
{
  "action": "fetch",
  "data": {
    "channelId": "home",
    "route": "/my-plugin/feed",
    "params": {},
    "vars": {}
  }
}
```

The request data has `channelId`, `route`, string-to-string `params`, `vars`, and deprecated `secrets` compatibility input. `fetch` is required. Playback actions are optional and only relevant when playback is plugin-managed. `manifest` is dev-only; `parse` is reserved for Phase 3 hybrid mode.

Success:

```json
{ "ok": true, "data": { "title": "Feed", "items": [] } }
```

Failure:

```json
{ "ok": false, "error": "human readable message" }
```

The SDK exposes `FetchRequest` as `ChannelID`, `Route`, `Params`, `Vars`, and `Secrets`. Use `req.Var("key")` for configured values. The basic `sdk.Run` dispatcher invokes `Fetch` and does not implement plugin-managed playback routing.

## Feed Result And Items

`FeedResult` contains `title`, optional `description`, `items`, optional `tree`, `hasMore`, and `next`. The SDK does not enforce item validation, so validate before appending.

Return stable, channel-unique `id`, non-empty `title`, an absolute `url`, and RFC3339 `published_at` for ordinary persisted feed items. Optional fields are `summary`, `content`, `author`, `cover`/`image`, `tags`, `kind`, `author_avatar`, `author_handle`, `stats`, `media`, and `quote`.

`TreeNode` has `id`, `title`, optional `url`, `site`, and recursive `children`.

Storage prefixes are:

- Feed item: `{pluginId}:{channelId}:{item.id}`
- Chapter item: `{pluginId}:{channelId}:{parentId}:{chapterItem.id}`

The `channelId` remains the list channel while fetching chapter routes.

## Social Fields

For social notes, `kind` is `short` or `long`. `stats` contains integer `likes`, `replies`, and `restacks`. Each `media` entry has `type` (`image`, `video`, or `link`) and optional `url`, `thumbnail`, `title`, `playback_id`, `width`, and `height`. `quote` contains `id`, `author`, `body`, and optional author/url fields.

Rich social text may use a ProseMirror `body_json` JSON string in `content`; retain plain `body` text in `summary` as a fallback.

## Pagination

Manifest and result keys must agree:

```json
"pagination": { "style": "offset", "param": "page", "default": "1" }
```

```json
{ "hasMore": true, "next": { "page": "2" } }
```

Styles are:

- `offset`: increment a page/offset value.
- `cursor`: return the upstream opaque token under the configured `param`.
- `lastId`: return the last item ID under the configured `param`; `idFrom` defaults to `item.id`.

`sizeParam`/`defaultSize` control optional page sizing. `carryParams` names extra keys that the runtime copies from `next` into the next load-more request. Use it for accumulated `seenIds` or equivalent source state.

Do not return `hasMore: true` with an empty or non-advancing `next` object.

## Navigation Features

`detail` is a two-level resolver called when a feed item opens. `chapters` inserts a sub-list before the final detail/content fetch. The features schema forbids declaring both on the same channel. Preserve parent IDs in chapter detail requests using `parentParam`/`parentFrom` when needed.

## Playback Actions

When `config.playback.managedBy` is `runtime`, the client handles history and progress; the plugin only needs feed/content routes. When it is `plugin`, implement these actions in a custom entry dispatcher:

- `playback_list`: input may include `limit` and `offset`.
- `playback_get`: input includes `parentId`.
- `playback_put`: input includes a playback `record`.
- `playback_delete`: input includes `parentId`.

Playback records require `parentId` and `updatedAt`. Prefer `progress` with time fields for video/audio, character offset for article/novel, or 1-based page for manga. See `schemas/playback.schema.json` for the complete shapes and defaults.

## Host HTTP

Use `host.HTTPGet` for plugin requests. The WASM host exposes HTTP, storage, logging, and time imports. HTTP text responses have `status` and `body`; binary responses may use `body_base64`. The runtime applies the manifest user agent and enforces per-invocation timeout, memory cap, and an 8 MiB HTTP response body limit.
