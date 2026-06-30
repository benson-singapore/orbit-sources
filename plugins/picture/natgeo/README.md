# National Geographic 图文插件

National Geographic 栏目文章，以图文内容展示：列表项上方为封面图，下方为描述文字。

## 特性

- 媒体类型：`article`（文章内容）
- 列表项内置 `content`（图片 + 描述 HTML）
- 实时刷新（`refresh=true`，按 `refreshInterval` 定时拉取）
- 不落库（`persist=false`）
- 支持栏目分类（`category`）
- `lastId` 分页，避免加载更多重复

## 列表内容格式

每条 `FeedItem` 的 `content` 字段结构：

1. 顶部：封面图 `<figure><img …/></figure>`
2. 底部：描述文字 `<p>…</p>`

列表已包含完整展示内容，无需 `detail` 二次请求。

## 支持分类

- `home`（首页）
- `animals`
- `travel`
- `science`
- `environment`
- `history`
- `health`

## 参数说明

| 参数 | 默认值 | 说明 |
|---|---|---|
| `category` | `home` | 栏目分类 |
| `lastId` | 空 | 加载更多时传入上一页最后一条的 `id` |
| `size` | `10` | 每页数量，最大 20 |

## 说明

插件流程：
1. 拉取分类 list 页（仅 1 次 HTTP 请求）。
2. 解析 `window['__natgeo__']` 主体内容，生成带 `content` 的列表项。
