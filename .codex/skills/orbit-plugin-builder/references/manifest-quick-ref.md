# Manifest And Features Quick Reference

Use the repository files as the final authority:

- `schemas/manifest.wasm.schema.json`
- `schemas/features.schema.json`
- `schemas/playback.schema.json`
- `schemas/browser-preview.md`

## Required Shape

Top-level required keys are `id`, `name`, `version`, `source`, `capabilities`, `config`, and `meta`. `mediaType` is optional and, when present, must be one of `article`, `novel`, `manga`, `video`, `audio`, `rating`, `image`, or `social`.

`id` must match `^[a-z0-9][a-z0-9_-]{1,63}$`. `source` is exactly `wasm`. `capabilities` is an array whose values are `feed` or `playback` and which must contain `feed`.

`config` requires:

- `channels`: at least one channel.
- `wasm`: optional fields `entry`, `timeoutMs` (minimum 1000), and `maxMemoryMB` (minimum 4).
- Optional `refreshInterval` (minimum 60), `userAgent`, `defaultChannel`, `executionMode`, `variables`, `secrets`, `browser`, and `playback`.

Each channel requires `id`, `label`, and `route`. Optional fields include string-valued `params`, `status` (`enabled` or `disabled`), and `features`. `itemLimit`, `type`, and `dynamic` are accepted only for backward compatibility and are deprecated.

## Variables

The current `variableDef` does not have a `type` field. It requires `label` and accepts `description`, `required`, `secret`, and string `default`:

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

At runtime, values are in `FetchRequest.Vars`; `FetchRequest.Var(key)` also falls back to deprecated `Secrets` for compatibility. `config.secrets` is deprecated; use variable definitions instead.

## Feature Matrix

| Feature | Required fields | Purpose |
|---|---|---|
| `feed` | none | `persist`, `refresh`, and `limit`; defaults are true, true, and 100. |
| `pagination` | `style` | `offset`, `cursor`, or `lastId`; optional `param`, `default`, `idFrom`, `sizeParam`, `defaultSize`, `carryParams`. |
| `search` | none | `param` defaults to `query`; `required` defaults to true. |
| `detail` | `route` | Two-level resolver; optional `idParam`, `idFrom`, and `persist` (default true). |
| `chapters` | `route` | Three-level sub-list; optional labels, persistence, limit, pagination, and nested `detail`. |
| `playback` | none | Per-channel override of plugin-level playback policy. |

`features.detail` and `features.chapters` are mutually exclusive. `chapters.detail`, when present, requires `route`; its defaults are `idParam: chapterId`, `idFrom: item.id`, `parentParam: id`, `parentFrom: parent.id`, and `persist: false`.

## Pagination Example

```json
{
  "params": { "page": "1", "seenIds": "" },
  "features": {
    "feed": { "persist": true, "refresh": true, "limit": 100 },
    "pagination": {
      "style": "offset",
      "param": "page",
      "default": "1",
      "carryParams": ["seenIds"]
    }
  }
}
```

Return `next` with the configured key and any carried values. See `docs/pagination-seenids.md` for the complete merge behavior.

## Playback

Declare `capabilities: ["feed", "playback"]` and configure `config.playback` for history/progress. Supported modes are `video`, `audio`, `article`, `novel`, and `manga`. `managedBy` is `runtime` or `plugin`; runtime is the default and does not require plugin playback actions. Channel `features.playback` can override `history`, `progress`, `mode`, or `limit`.

Use [schemas/playback.schema.json](../../../../schemas/playback.schema.json) for exact progress and record fields. Prefer mode-specific `progress` over deprecated top-level `position`/`duration`.

## Browser And Hybrid Preview

`executionMode` accepts `wasm`, `browser`, or `hybrid`; `config.browser` accepts `required` and `fallbackOn`. Browser/hybrid execution is currently a Phase 3 preview and not implemented in Phase 1. `parse` is a reserved ABI action. Treat these fields as forward-compatible declarations, not a currently testable fallback.
