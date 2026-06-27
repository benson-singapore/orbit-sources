# 插件开发指南

本文面向第一次开发 Orbit WASM 插件的开发者。

## 1. 环境要求

- Go 1.22+
- `make`
- 可选：`wasmtime`，用于验证 WASM 产物
- 可选：Orbit Runtime，用于完整联调

## 2. 插件目录

每个插件是独立 Go module，放在 `plugins/<category>/<id>/`：

```text
plugins/programming/my-plugin/
  go.mod
  main.go
  manifest.json
  Makefile
  README.md
```

插件 ID 必须同时满足：

- 目录名是 `<id>`
- `manifest.json` 的 `id` 是 `<id>`
- Makefile 的 `PLUGIN_ID` 是 `<id>`

## 3. 最小插件

`go.mod`：

```go
module github.com/orbit-sources/plugin-my-plugin

go 1.22

require github.com/orbit-tauri-tools/plugin-sdk v0.0.0

replace github.com/orbit-tauri-tools/plugin-sdk => ../../../sdk
```

`main.go`：

```go
package main

import (
	"fmt"
	"time"

	sdk "github.com/orbit-tauri-tools/plugin-sdk"
)

func main() {
	sdk.Run(&Plugin{})
}

type Plugin struct{}

func (p *Plugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch req.Route {
	case "/my-plugin/feed":
		return &sdk.FeedResult{
			Title: "My Plugin",
			Items: []sdk.FeedItem{{
				ID:          "hello",
				Title:       "Hello Orbit",
				URL:         "https://example.com/hello",
				PublishedAt: time.Now().Format(time.RFC3339),
			}},
		}, nil
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}
```

## 4. manifest.json

`manifest.json` 是插件元数据唯一来源：

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
    "defaultChannel": "feed",
    "executionMode": "wasm",
    "channels": [
      {
        "id": "feed",
        "label": "Feed",
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
  "meta": {
    "description": "Example plugin",
    "icon": "text",
    "color": "bg-blue-500",
    "logoText": "M",
    "marketCategory": "blog",
    "categoryTag": "DEV",
    "official": false
  }
}
```

## 5. HTTP 请求

插件内 HTTP 请求统一使用 SDK host 包：

```go
body, status, err := host.HTTPGet("https://api.example.com/items", map[string]string{
	"Accept": "application/json",
})
```

WASM 环境下请求由 Runtime 代理；原生测试时走 Go 标准库。

## 6. 用户配置

需要 API Key、Token、地区等配置时，只在 `manifest.json` 声明变量，不提交真实值：

```json
"variables": {
  "apiKey": {
    "label": "API Key",
    "required": true,
    "secret": true
  }
}
```

代码中读取：

```go
apiKey := req.Var("apiKey")
```

## 7. 开发检查清单

- `manifest.id`、目录名、`PLUGIN_ID` 一致
- 所有 enabled channel 的 route 都有对应 Fetch 分支
- `FeedItem.ID` 在频道内稳定唯一
- `FeedItem.URL` 是完整 URL
- `FeedItem.PublishedAt` 是 RFC3339 格式
- 需要用户输入的值使用 `config.variables`
- 不提交成人内容、私钥、Token 或本地构建产物
