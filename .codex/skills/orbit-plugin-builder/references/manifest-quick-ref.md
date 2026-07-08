# Manifest Quick Reference

## Minimal Valid Manifest

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
    "defaultChannel": "home",
    "executionMode": "wasm",
    "channels": [
      {
        "id": "home",
        "label": "首页",
        "route": "/plugin/list/:section",
        "params": { "section": "home" },
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
    "description": "Plugin description",
    "icon": "text",
    "marketCategory": "news",
    "official": true
  }
}
```

## Config Fields

| Field | Required | Type | Notes |
|-------|----------|------|-------|
| `id` | yes | string | Pattern: `^[a-z0-9][a-z0-9_-]{1,63}$` |
| `name` | yes | string | Display name |
| `version` | yes | string | Semantic version |
| `mediaType` | yes | enum | article\|social\|video\|audio\|manga\|novel\|rating\|image |
| `source` | yes | const | Always `"wasm"` |
| `capabilities` | yes | array | Always includes `"feed"` |
| `config` | yes | object | Plugin configuration |
| `meta` | yes | object | UI metadata |

## Channel Features Matrix

| Feature | Use case | Key params |
|---------|----------|-----------|
| `feed` | List persistence | persist, refresh, limit |
| `pagination` | Load more | style (offset\|cursor\|lastId), param, default |
| `search` | User query | param, required |
| `detail` | Item resolver | route, idParam, idFrom, persist |
| `chapters` | Sub-list (3-level) | route, idParam, detail, pagination |

## Common Patterns

### News plugin (article + pagination)
```json
{
  "mediaType": "article",
  "config": {
    "channels": [{
      "features": {
        "feed": { "persist": true, "refresh": true },
        "pagination": { "style": "offset", "param": "page", "default": "1" },
        "detail": { "route": "/plugin/detail/:id" }
      }
    }]
  }
}
```

### Social plugin (tweets + stats)
```json
{
  "mediaType": "social",
  "config": {
    "channels": [{
      "features": {
        "feed": { "persist": true, "refresh": true }
      }
    }]
  }
}
```

### Search plugin (no persist on list)
```json
{
  "config": {
    "channels": [{
      "features": {
        "feed": { "persist": false, "refresh": false },
        "search": { "param": "query", "required": true },
        "pagination": { "style": "cursor", "param": "pageToken" }
      }
    }]
  }
}
```

### Manga plugin (chapters + detail)
```json
{
  "mediaType": "manga",
  "config": {
    "channels": [{
      "features": {
        "chapters": {
          "route": "/comic/:id/chapters",
          "label": "话数",
          "detail": {
            "route": "/comic/:id/chapter/:chapterId",
            "persist": false
          }
        }
      }
    }]
  }
}
```
