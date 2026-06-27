# Voronoi 插件

从 [Voronoi](https://www.voronoiapp.com/)（Visual Capitalist 数据可视化平台）抓取信息图资讯，支持分类、热门、编辑精选，以及**按创作者订阅作品**。

## 功能特性

- 公开 API，无需登录或 Token
- 内置频道：最新、热门、编辑精选、首页推荐、科技/经济/商业分类
- 支持关注指定创作者的作品流（无需 Voronoi 账号）
- 列表含封面、摘要、作者；详情含完整 HTML 描述与配图
- 基于 `offset` 分页

## 内置频道

| 频道 ID | 显示名称 | 说明 |
|---------|---------|------|
| `latest` | 最新 | 按发布时间排序 |
| `popular` | 热门 | 近 30 天最受欢迎 |
| `curated` | 编辑精选 | 编辑精选内容 |
| `home` | 首页推荐 | Voronoi 首页 feed |
| `technology` | 科技 | 分类 ID 19 |
| `economy` | 经济 | 分类 ID 5 |
| `business` | 商业 | 分类 ID 2 |
| `creator-follow` | 我的关注 | 使用插件变量 `creator` 指定的创作者 |
| `creator-visualcapitalist` | Visual Capitalist | 示例创作者频道 |

## 路由

| 路由 | 用途 |
|------|------|
| `/voronoi/list` | 列表（最新/热门/分类等） |
| `/voronoi/creator` | 指定创作者的作品列表 |
| `/voronoi/detail/:id` | 单条详情（按 `pid`） |

## 关注创作者（可配置）

**可以**在插件内订阅某个创作者的作品，**不需要** Voronoi 登录态，也**不能**同步 Voronoi App 里「Follow」按钮的关注列表（那需要用户 Token）。

实现方式：调用公开接口 `GET /post?author={uid}`，只拉取该创作者已发布的作品。

### 方式一：插件变量（推荐用于「我的关注」频道）

在插件设置中填写变量 **关注创作者**（`creator`）：

```
visualcapitalist
```

然后启用 **我的关注**（`creator-follow`）频道。频道 `params.creator` 为空时，会自动读取该变量。

### 方式二：为每个创作者单独建频道

复制 manifest 中的 `creator-visualcapitalist` 频道，修改：

```json
{
  "id": "creator-owid",
  "label": "Our World in Data",
  "route": "/voronoi/creator",
  "params": {
    "creator": "OWiDCharts",
    "offset": "0"
  }
}
```

### 方式三：直接使用创作者 UID（最稳定）

若用户名解析失败，可在 `params` 中填写 `uid`（作者 `sub` 字段）：

```json
"params": {
  "uid": "3c0da578-e0f1-700a-3188-da67e9645746",
  "offset": "0"
}
```

### 如何查找创作者标识

1. **用户名**：打开创作者主页  
   `https://www.voronoiapp.com/creator/visualcapitalist`  
   URL 最后一段即为 `creator` 参数。

2. **UID**：在任意该创作者作品 JSON 的 `author.sub` 字段，例如 Visual Capitalist 为  
   `3c0da578-e0f1-700a-3188-da67e9645746`。

3. **注意**：部分账号的 `preferred_username` 与搜索关键词不一致（如 Statista 需用 `Statista` 而非 `StatistaCharts`）。解析失败时请改用 `uid`。

### 创作者接口参数

| 参数 | 说明 |
|------|------|
| `creator` | 创作者用户名（优先从频道 params 读取，否则读插件变量） |
| `uid` | 创作者 UID，填写后忽略 `creator` |
| `offset` | 分页偏移，默认 `0` |
| `size` | 每页条数，默认 20，最大 50 |

## 列表参数 `/voronoi/list`

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `feed` | `latest` | `latest` / `popular` / `curated` / `home`，或分类 ID（如 `19`） |
| `query` | — | 关键词搜索（配合 `feed` 使用） |
| `offset` | `0` | 分页偏移 |
| `size` | `20` | 每页条数 |

## 本地开发

```bash
# 最新列表
make try-voronoi CHANNEL=latest ROUTE=/voronoi/list PARAMS='{"feed":"latest","offset":"0"}'

# 热门
make try-voronoi CHANNEL=popular ROUTE=/voronoi/list PARAMS='{"feed":"popular","offset":"0"}'

# 创作者（Visual Capitalist）
make try-voronoi CHANNEL=creator-visualcapitalist ROUTE=/voronoi/creator PARAMS='{"creator":"visualcapitalist","offset":"0"}'

# 详情
make try-voronoi ROUTE=/voronoi/detail/:id PARAMS='{"id":"8464"}'

# 编译
make build-voronoi
make package-voronoi
```

## 数据来源

- API：`https://api.voronoiapp.com`
- 图片 CDN：`https://cdn.voronoiapp.com/public/`
- 文章链接：`https://www.voronoiapp.com/{link}`

## 限制说明

- 非官方公开 API，参数可能随前端改版变化
- 「关注」仅为按创作者拉取其公开作品，与 Voronoi 账号内的 Follow 功能无关
- 同步 Voronoi 账号关注列表、书签、点赞等需登录 Token，本插件未实现
