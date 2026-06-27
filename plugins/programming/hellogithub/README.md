# HelloGitHub Plugin v2.0

HelloGitHub 开源项目推荐插件 - 支持多分类浏览和详情查询。

## Features

- ✅ **动态分类列表** - 实时获取所有项目分类
- ✅ **按分类浏览** - Python、Java、C++、JavaScript、教程、AI、算法、Rust、游戏等
- ✅ **精选/最新排序** - 每个分类支持精选和最新两种排序方式
- ✅ **项目详情** - 获取完整项目描述和 HTML 内容
- ✅ **丰富元数据** - 作者、发布时间、编程语言标签等

## Routes

### 1. 分类列表 `/hellogithub/tags`
获取所有可用的项目分类

**参数**: 无

**返回**: 分类列表，每个分类包含：
- `id`: 分类 ID (tid)
- `title`: 分类名称
- `url`: 分类链接

**示例**:
```bash
CHANNEL=tags ROUTE=/hellogithub/tags PARAMS='{}' make test-native
```

**响应示例**:
```json
{
  "ok": true,
  "data": {
    "title": "HelloGithub 分类",
    "description": "所有可用分类",
    "items": [
      {"id": "Z8PipJsHCX", "title": "Python", ...},
      {"id": "YgDkvUzLAC", "title": "Java", ...},
      {"id": "juBLV86qa5", "title": "AI", ...},
      ...
    ]
  }
}
```

### 2. 按分类查询 `/hellogithub/category/:tid`
查询特定分类下的精选项目

**参数**:
- `tid`: 分类 ID（从 /hellogithub/tags 获取）
- `sort`: 排序方式（推荐 "featured"，默认值）

**返回**: 该分类的项目列表，每个项目包含：
- `id`: **项目全名** (author/project_name，如 "Andyyyy64/whichllm")
- `title`: 项目标题
- `url`: 项目详情链接
- `summary`: 项目摘要
- `author`: 项目作者
- `published_at`: 更新时间
- `tags`: 编程语言标签

**示例**:
```bash
# Python - 精选
CHANNEL=python ROUTE=/hellogithub/category/:tid PARAMS='{"tid":"Z8PipJsHCX","sort":"featured"}' make try-hellogithub

# AI - 精选
CHANNEL=ai ROUTE=/hellogithub/category/:tid PARAMS='{"tid":"juBLV86qa5","sort":"featured"}' make try-hellogithub
```

### 3. 项目详情 `/hellogithub/detail/:id`
获取项目的完整详情和 HTML 内容

**参数**:
- `id`: 项目全名 (格式: "author/project_name"，如 "Andyyyy64/whichllm")
- 该参数与列表返回的 `id` 字段保持一致

**返回**: 单个项目的详细信息，包含：
- `id`: 项目全名
- `title`: 项目全名
- `url`: 详情页链接
- `content`: **完整 HTML 内容**，包含：
  - 项目展示图片（高清截图/演示图）
  - 项目完整描述文本

**示例**:
```bash
# 使用 id 参数（推荐，与列表返回的 id 一致）
CHANNEL=detail ROUTE=/hellogithub/detail/:id PARAMS='{"id":"Andyyyy64/whichllm"}' make test-native

# 或从列表中获取 id 后直接传递
# list 返回: { "id": "Andyyyy64/whichllm", "url": "...", ... }
# 使用该 id 查询详情
```

**响应示例**:
```json
{
  "ok": true,
  "data": {
    "title": "Andyyyy64/whichllm",
    "description": "项目详情",
    "items": [
      {
        "id": "Andyyyy64/whichllm",
        "title": "Andyyyy64/whichllm",
        "url": "https://hellogithub.com/repository/Andyyyy64/whichllm",
        "content": "<div class=\"flex cursor-zoom-in justify-center pt-2\"><div class=\"relative flex\"><img class=\"rounded-md border border-gray-200\" src=\"https://img.hellogithub.com/i/ys1NthP37oe5GbH_1779548072.gif\" alt=\"whichllm image\"/></div></div>\n<div class=\"w-full p-2 leading-8\">该项目能够自动检测本机 GPU/CPU/RAM 配置，并从 HuggingFace 中筛选出适合当前硬件的大模型...</div>"
      }
    ]
  }
}
```

## 预配置的 Channels

Manifest 中已包含以下预配置的频道：

| 分类 | Channel ID | 状态 | 说明 |
|------|------------|------|------|
| 分类列表 | `tags` | enabled | 获取所有分类 |
| Python | `python-featured` | enabled | Python 精选项目 |
| Java | `java-featured` | enabled | Java 精选项目 |
| C++ | `cpp-featured` | enabled | C++ 精选项目 |
| JavaScript | `javascript-featured` | enabled | JavaScript 精选项目 |
| 教程 | `tutorial-featured` | enabled | 教程精选项目 |
| AI | `ai-featured` | enabled | AI 精选项目 |
| 算法 | `algo-featured` | enabled | 算法精选项目 |
| Rust | `rust-featured` | enabled | Rust 精选项目 |
| 游戏 | `game-featured` | enabled | 游戏精选项目 |

各分类频道通过 `features.detail` 声明详情解析（非独立 channel）：

```json
"features": {
  "detail": {
    "route": "/hellogithub/detail/:id",
    "idParam": "id",
    "persist": true
  }
}
```

打开列表项时 Runtime 调用 `detail.route`，将 `content` 写回库中该条。使用列表返回的 `id` 字段（`full_name` 格式）。

## API Source

- Tags API: `https://abroad.hellogithub.com/v1/tag/`
- Category API: `https://abroad.hellogithub.com/v1/?sort_by=...&page=1&rank_by=newest&tid=...`
- Detail Pages: `https://hellogithub.com/repository/{full_name}`

## Development

### Native testing
```bash
# 获取分类列表
make try-hellogithub

# 查询 Python 分类 - 精选
CHANNEL=python ROUTE=/hellogithub/category/:tid PARAMS='{"tid":"Z8PipJsHCX","sort":"featured"}' make try-hellogithub

# 获取项目详情（使用 id 参数，与列表返回的 id 一致）
CHANNEL=detail ROUTE=/hellogithub/detail/:id PARAMS='{"id":"Andyyyy64/whichllm"}' make try-hellogithub
```

### WASM testing
```bash
brew install wasmtime
make try-wasm-hellogithub
```

### Full integration
```bash
# Terminal 1: Start Runtime
cd .. && make dev-go

# Terminal 2: Build and test
make dev-hellogithub
```

## Version History

- **v2.0** - 完全重新设计，支持动态分类、多种排序、详情查询
- **v1.0** - 初始版本，支持精选和全部两种频道
