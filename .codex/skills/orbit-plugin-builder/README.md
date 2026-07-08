# Orbit Plugin Builder Skill

Complete workflow for developing Orbit feed plugins with Go + WASM.

## Contents

- **SKILL.md** — Main skill document (606 lines)
  - Core concepts: plugin anatomy, SDK interface, manifest structure
  - Build workflow: scaffold → build → package → orbit
  - Testing & debugging: make try, make try-wasm, make dev
  - Verification checklist
  - File templates: main.go, Makefile, go.mod, manifest.json

- **references/**
  - `manifest-quick-ref.md` — Manifest field reference, channel features matrix, common patterns
  - `abi-quick-ref.md` — ABI v1 request/response format, FeedItem fields, pagination modes, code example

- **evals/evals.json** — 5 test scenarios covering:
  1. New plugin scaffolding (templating, build, registration)
  2. Pagination implementation (3 styles: offset/cursor/lastId)
  3. Selector debugging (HTML vs parser output)
  4. Packaging & distribution (extension.orbit)
  5. User variables (config.variables, API key injection)

## How It Works

The skill provides:

1. **Schema compliance** — Aligns with Orbit's official schemas (manifest.wasm.schema.json, features.schema.json, abi-v1.md)
2. **Step-by-step guidance** — From empty directory to working WASM plugin
3. **Real examples** — Code templates, manifest patterns, test commands
4. **Troubleshooting** — Common issues and fixes
5. **Verification** — Checklists for build, test, and package stages

## When to Use

- Creating a new Orbit plugin from scratch or from existing template
- Implementing pagination with offset/cursor/lastId styles
- Adding search, chapters, or detail routes
- Normalizing feed items (FeedItem fields)
- Testing locally via make try / make try-wasm / make dev
- Debugging selector issues or API changes
- Packaging plugins for distribution
- Understanding Orbit manifest and ABI v1 contracts
- Configuring user variables and secrets

## Rules Captured

### Plugin Discovery
- Location: `plugins/<category>/<id>/` (exactly 2 levels deep)
- ID pattern: `^[a-z0-9][a-z0-9_-]{1,63}$`
- Makefile at `plugins/<category>/<id>/Makefile` required

### Build Pipeline
1. `make build PLUGIN=id` → WASM compilation (GOOS=wasip1 GOARCH=wasm)
2. `make package PLUGIN=id` → Copy manifest, README to dist/
3. `make orbit PLUGIN=id` → Brotli compress + ZIP → extension.orbit

### SDK Interfaces
- **Plugin.Fetch(FetchRequest) → (FeedResult, error)**
- **FetchRequest fields:** channelId, route, params, vars
- **FeedResult fields:** title, items[], hasMore, next, tree

### Feed Item Normalization
- Required: id, title, url, published_at
- Optional: cover/image, summary, author, tags, stats (for social), media, quote

### Manifest Schema
- Required: id, name, version, mediaType, source, capabilities, config, meta
- config.channels[] declares routes and features
- features: feed, pagination, search, detail, chapters, playback

### Pagination Contracts
- **offset:** page numbers (1, 2, 3...)
- **cursor:** opaque pagination token
- **lastId:** last seen item id
- Return: `FeedResult.HasMore = true` + `FeedResult.Next = {"param": "value"}`

### Testing
- `make try PLUGIN=id` — Native Go execution (stdin/stdout JSON)
- `make try-wasm PLUGIN=id` — Wasmtime on compiled WASM
- `make dev PLUGIN=id` — Orbit runtime integration test

## Skill Structure

```
orbit-plugin-builder/
├── SKILL.md                          # Main instruction (606 lines, 2042 words)
├── references/
│   ├── manifest-quick-ref.md         # Manifest fields & patterns
│   └── abi-quick-ref.md              # Request/response formats
├── evals/
│   └── evals.json                    # 5 test scenarios
└── scripts/                          # (empty; ready for helper scripts)
```

## Test Scenarios (evals.json)

1. **eval-1:** New plugin scaffolding
   - Prompt: Create testnews plugin from zaobao template
   - Expected: Step-by-step with file edits, build commands, verification

2. **eval-2:** Pagination implementation
   - Prompt: Add pagination to existing plugin (3 styles, features config, testing)
   - Expected: Config snippets, code examples, test commands

3. **eval-3:** Selector debugging
   - Prompt: Empty items list (selector error)
   - Expected: Debugging steps (DevTools, logs, goquery, API vs HTML comparison)

4. **eval-4:** Packaging & distribution
   - Prompt: Package plugin as extension.orbit
   - Expected: Build commands, dist/ structure, verification

5. **eval-5:** User variables (API key)
   - Prompt: Add config.variables for API key
   - Expected: Manifest config, FetchRequest.Var() usage, secrets deprecation

## Rule Sources

Rules derived from and validated against:
- `schemas/manifest.wasm.schema.json` — Plugin manifest schema
- `schemas/features.schema.json` — Channel features definitions
- `schemas/abi-v1.md` — ABI v1 transport and contracts
- `plugins/news/zaobao/` — Reference plugin (news + pagination)
- `plugins/news/huanqiu/` — Reference plugin (pagination + chapters)
- Root `Makefile` — Build discovery and pipeline
- `sdk/types.go` — SDK types (FetchRequest, FeedResult, FeedItem)

