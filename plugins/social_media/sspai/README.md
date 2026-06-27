# 少数派插件

从 [少数派](https://sspai.com/) 抓取文章列表与正文，支持列表落库、分页加载与详情按需拉取。

## 功能特性

- 三个内置频道：推荐、全部、热门
- 列表仅入库元数据（标题、摘要、封面、作者等），不预拉正文
- 支持 `features.pagination` 翻页浏览历史文章
- 支持 `features.detail` 点击后动态请求完整 HTML 正文
- 基于少数派官方 JSON API，无需解析网页

## 内置频道

| 频道 ID | 显示名称 | `type` 参数 | 对应 API |
|---------|---------|-------------|----------|
| `recommended` | 推荐 | `index` | `/api/v1/article/index/page/get` |
| `all` | 全部 | `matrix` | `/api/v1/article/matrix/page/get` |
| `hot` | 热门 | `hot` | `/api/v1/article/hot/page/get` |

默认频道：`recommended`

## 路由

| 路由 | 用途 |
|------|------|
| `/sspai/list` | 文章列表（支持分页） |
| `/sspai/detail/:id` | 文章详情（按文章 ID 拉取正文） |

## 参数说明

### 列表 `/sspai/list`

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `type` | string | `index` | 列表类型：`index` / `matrix` / `hot` |
| `page` | string | `1` | 页码，从 1 开始 |
| `size` | string | `10` | 每页条数，最大 30 |

页码与 API `offset` 的换算：

```
offset = (page - 1) * size
```

### 详情 `/sspai/detail/:id`

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | string | 文章 ID，如 `110989` |

## Features 配置

```json
"features": {
  "feed": {
    "persist": true,
    "refresh": true,
    "limit": 200
  },
  "pagination": {
    "style": "offset",
    "param": "page",
    "default": "1",
    "sizeParam": "size",
    "defaultSize": 10
  },
  "detail": {
    "route": "/sspai/detail/:id",
    "idParam": "id",
    "persist": true
  }
}
```

- **feed** — 定时刷新拉第一页并增量入库
- **pagination** — 加载更多时递增 `page`，追加历史记录
- **detail** — 打开文章时调用详情接口，正文入库后复用

> `defaultSize` 在 manifest 中必须是 **整数**（如 `10`），不能写成字符串 `"10"`。

## 本地开发

```bash
# 列表第一页（推荐）
make try-sspai CHANNEL=recommended ROUTE=/sspai/list PARAMS='{"type":"index","page":"1"}'

# 分页
make try-sspai CHANNEL=recommended ROUTE=/sspai/list PARAMS='{"type":"index","page":"2"}'

# 热门频道
make try-sspai CHANNEL=hot ROUTE=/sspai/list PARAMS='{"type":"hot","page":"1"}'

# 文章详情
make try-sspai ROUTE=/sspai/detail/:id PARAMS='{"id":"110989"}'

# 编译与打包
make build-sspai
make package-sspai
```

在插件目录内也可直接：

```bash
cd plugins/social_media/sspai
make test-native
```

## 数据返回格式

### 列表项

| 字段 | 说明 |
|------|------|
| `id` | 文章 ID |
| `title` | 标题 |
| `url` | 原文链接 `https://sspai.com/post/{id}` |
| `summary` | 摘要 |
| `author` | 作者昵称 |
| `cover` / `image` | 封面图 URL |
| `published_at` | 发布时间（RFC3339） |
| `tags` | 点赞数、评论数、文章标签等 |

列表响应还包含：

| 字段 | 说明 |
|------|------|
| `hasMore` | 是否还有下一页 |
| `next` | 下一页参数，如 `{ "page": "2" }` |

### 详情项

在列表字段基础上增加：

| 字段 | 说明 |
|------|------|
| `content` | 完整文章 HTML（含封面图与正文） |

## 数据来源

列表与详情均调用少数派公开 API：

- 列表：`https://sspai.com/api/v1/article/{index|matrix|hot}/page/get`
- 详情：`https://sspai.com/api/v1/article/info/get?id={id}&support_webp=true&view=second`

更完整的接口说明见仓库文档：`docs/抓取/少数派.md`

## 常见问题

### Q: 为什么列表里没有正文？

A: 列表阶段只入库元数据，正文在用户打开文章时通过 `detail` 路由按需拉取，减少刷新时的请求量。

### Q: 如何修改每页条数？

A: 在频道 `params` 中设置 `size`，或在 `pagination.defaultSize` 中修改默认值（须为整数）。

### Q: manifest 修改后如何生效？

A: 重新打包后，若 Runtime 已在运行：

```bash
curl -X POST http://127.0.0.1:17890/v1/plugins/resync
```
