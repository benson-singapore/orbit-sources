# TMDB 插件

基于 [The Movie Database (TMDB)](https://www.themoviedb.org/) API v3，提供影视排行榜订阅、分区榜单、搜索与详情展示。数据包括电影/剧集元信息、评分、演员、剧照、预告片与影评等。

---

## 快速开始

1. 在 [TMDB](https://www.themoviedb.org/signup) 注册账号并申请 API 凭证
2. 打开 Orbit → **插件设置** → TMDB，填写 **API Key / Access Token**
3. 按需调整语言、流媒体地区等参数
4. 订阅感兴趣的频道（如「近期热门韩剧」「Netflix 剧集」）
5. 点击条目查看详情；使用「搜索」频道按名称查找

---

## 获取 API 凭证

TMDB 提供两种认证方式，**任选其一**填入插件的 `apiKey` 字段即可。

### 方式一：Read Access Token（推荐）

适用于 TMDB 当前默认发放的 Bearer Token，以 `eyJ` 开头。

1. 登录 [TMDB](https://www.themoviedb.org/)
2. 进入 **头像 → Settings → API**
3. 在 **API Read Access Token** 区域点击 **Copy**，或按页面指引创建 Access Token
4. 将完整 Token 粘贴到 Orbit 插件设置的 **TMDB API Key / Access Token**

插件会自动识别 JWT 格式，使用 `Authorization: Bearer <token>` 请求 API。

### 方式二：API Key (v3 auth)

传统 32 位 API Key，通过查询参数 `api_key=` 认证。

1. 同上进入 [API 设置页](https://www.themoviedb.org/settings/api)
2. 在 **API Key (v3 auth)** 区域申请并复制 Key
3. 粘贴到插件设置的 `apiKey` 字段

### 注意事项

- **不要将真实密钥写入代码仓库**；manifest 仅声明字段，取值由用户在客户端填写
- 个人非商业用途通常免费；详见 [TMDB API 使用条款](https://www.themoviedb.org/api-terms-of-use)
- 请遵守 TMDB 速率限制，避免短时间内大量刷新

---

## 插件配置参数

在 Orbit **插件设置** 中可配置以下变量（对应 `manifest.json` → `config.variables`）：

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| **TMDB API Key / Access Token** | 是 | — | API 凭证。支持 v3 API Key 或 Read Access Token（`eyJ…`） |
| **语言** | 否 | `zh-CN` | 接口返回的标题、简介等文本语言。常用：`zh-CN`、`en-US`、`ja-JP` |
| **流媒体地区** | 否 | `US` | Netflix / Disney+ / Prime Video 等榜单的可用地区。常用：`US`、`CN`、`HK`、`KR` |
| **近期榜单起始日期** | 否 | `2023-01-01` | 「近期*」类频道只包含该日期之后上映/开播的内容。格式：`YYYY-MM-DD` |
| **近期高分最低评分数** | 否 | `200` | 「近期高分」「高分*」类频道要求的最少评分人数，过滤样本过少的虚高评分 |

### 配置建议

| 使用场景 | 推荐配置 |
|----------|----------|
| 只看最近一两年的内容 | `近期榜单起始日期` → `2024-01-01` 或 `2025-01-01` |
| 在国内查 Netflix 片单 | `流媒体地区` → `US` 或 `HK`（TMDB 按地区返回平台片库） |
| 英文原名与简介 | `语言` → `en-US` |
| 高分榜更严格 | `近期高分最低评分数` → `500` 或 `1000` |

---

## 频道分类

插件共 **37 个频道**，分为以下几类。榜单逻辑简述：

| 类型 | 排序依据 | 特点 |
|------|----------|------|
| **热门** | 全站热度 | 当前最受关注 |
| **高分** | 用户评分 | 经典作品偏多 |
| **趋势** | 近日关注度变化 | 偏「正在火」 |
| **近期** | 限定日期后的热度/评分 | 新片新剧优先 |
| **分区** | Discover 筛选 + 热度/评分 | 按语言、类型、平台等细分 |

### 官方榜单

| 频道 | 说明 |
|------|------|
| 热门电影 | TMDB 官方热门电影榜 |
| 热门电视剧 | TMDB 官方热门剧集榜 |
| 高分电影 | 历史高分电影（易含经典老片） |
| 高分电视剧 | 历史高分剧集（易含经典老剧） |
| 今日热榜 · 电影 / 剧集 / 全类型 | 24 小时趋势 |
| 本周热榜 · 电影 / 剧集 / 全类型 | 7 天趋势 |
| 即将上映 | 即将上映电影 |
| 正在上映 | 当前院线电影 |

### 近期最佳（通用）

受 **近期榜单起始日期**、**近期高分最低评分数** 影响。

| 频道 | 说明 |
|------|------|
| 近期高分剧集 | 指定日期后开播、按评分排序 |
| 近期热门剧集 | 指定日期后开播、按热度排序 |
| 近期高分电影 | 指定日期后上映、按评分排序 |
| 近期热门电影 | 指定日期后上映、按热度排序 |

### 国漫

| 频道 | 说明 |
|------|------|
| 国漫排行 | 中国动画，按热度 |
| 高分国漫 | 中国动画，按评分 |
| 近期热门国漫 | 近期中国动画，按热度 |
| 近期高分国漫 | 近期中国动画，按评分 |

### 韩剧

| 频道 | 说明 |
|------|------|
| 韩剧排行 | 韩语剧集，按热度 |
| 高分韩剧 | 韩语剧集，按评分 |
| 近期热门韩剧 | 近期韩语剧集，按热度 |
| 近期高分韩剧 | 近期韩语剧集，按评分 |

### Netflix

受 **流媒体地区** 影响（决定该平台在该地区可用的作品）。

| 频道 | 说明 |
|------|------|
| Netflix 剧集 | Netflix 剧集，按热度 |
| Netflix 高分剧集 | Netflix 剧集，按评分 |
| 近期 Netflix 剧集 | 近期 Netflix 剧集，按热度 |
| 近期 Netflix 高分 | 近期 Netflix 剧集，按评分 |
| Netflix 电影 | Netflix 电影，按热度 |

### 动漫与其他分区

| 频道 | 说明 |
|------|------|
| 动漫排行 / 动漫电影 / 高分动漫 | 日本动画（电影/剧集） |
| 日剧排行 | 日语剧集 |
| 华语剧集 | 中文剧集 |
| 欧美剧集 | 美国剧集 |
| Disney+ 剧集 | Disney+ 平台剧集 |
| Prime Video 剧集 | Amazon Prime 剧集 |
| 科幻奇幻剧集 | 科幻 & 奇幻类型 |
| 动作电影 | 动作类型电影 |

### 搜索

| 频道 | 说明 |
|------|------|
| 搜索 | 按名称搜索电影/剧集，支持分页 |

在客户端输入关键词即可；默认综合搜索电影与剧集，不持久化缓存。

---

## 使用说明

### 订阅频道

1. 在 Orbit 中添加 TMDB 插件
2. 完成 API 凭证与参数配置
3. 在频道列表中选择要订阅的榜单（可多选）
4. 刷新后即可在 Feed 中浏览条目

### 查看详情

点击任意电影/剧集条目，插件会请求详情并展示：

- 类型、评分、上映/开播日期、时长
- 导演、演员（含头像）
- 背景图、海报、剧照（最多 36 张）
- YouTube 预告片（如有）
- 用户影评（如有）

### 搜索

1. 订阅「搜索」频道
2. 在搜索框输入影视名称（如「星际穿越」「黑暗荣耀」）
3. 点击结果进入详情

### 分页

大部分榜单频道支持翻页；搜索频道同样支持分页加载更多结果。

---

## 本地开发与测试

### 环境要求

- Go 1.22+
- 已配置 TMDB API 凭证

### 常用命令

```bash
# 编译 WASM
make build-tmdb

# 打包到 dist/tmdb/
make package-tmdb

# 本地运行（无需 WASM）
cd plugins/social_media/tmdb
```

### 测试示例

将 `YOUR_API_KEY` 替换为你的 API Key 或 Read Access Token。

**列表（热门电影）：**

```bash
echo '{"action":"fetch","data":{"channelId":"movie_popular","route":"/tmdb/list","params":{"endpoint":"movie/popular","page":"1"},"vars":{"apiKey":"YOUR_API_KEY","language":"zh-CN"}}}' | go run .
```

**近期热门韩剧：**

```bash
echo '{"action":"fetch","data":{"channelId":"recent_korean_drama","route":"/tmdb/list","params":{"endpoint":"discover/tv","page":"1","sort_by":"popularity.desc","with_original_language":"ko","use_recent_since":"tv"},"vars":{"apiKey":"YOUR_API_KEY","recentSince":"2023-01-01","watchRegion":"US"}}}' | go run .
```

**搜索：**

```bash
echo '{"action":"fetch","data":{"channelId":"search","route":"/tmdb/search/:query","params":{"query":"星际穿越","page":"1","type":"multi"},"vars":{"apiKey":"YOUR_API_KEY"}}}' | go run .
```

**详情：**

```bash
echo '{"action":"fetch","data":{"channelId":"movie_popular","route":"/tmdb/detail/:id","params":{"id":"movie:550"},"vars":{"apiKey":"YOUR_API_KEY","language":"zh-CN"}}}' | go run .
```

### 接入 Orbit Runtime 开发

```bash
# 终端 1：父项目启动 Runtime
cd .. && make dev-go

# 终端 2：打包并刷新
make package-tmdb
make dev-tmdb   # 或手动刷新 feed

# manifest 变更后需 resync
curl -X POST http://127.0.0.1:17890/v1/plugins/resync
```

---

## 故障排查

### `TMDB API key required`

未配置 API 凭证。请在 Orbit 插件设置中填写 **TMDB API Key / Access Token**，或本地测试时在 `vars.apiKey` 中传入。

### `http status 401` / `Invalid API key`

- 凭证错误或已失效 → 到 [API 设置页](https://www.themoviedb.org/settings/api) 重新复制
- 若使用 Read Access Token，确保完整粘贴（以 `eyJ` 开头的一长串 JWT）
- 若使用 v3 API Key，确保未误填成 Access Token（或反之）

### Netflix / Disney+ 榜单结果很少或为空

- 调整 **流媒体地区**（如 `US`、`HK`）
- 部分地区平台片库在 TMDB 中数据较少，属正常现象

### 「高分」榜单全是老剧

高分榜按历史评分排序，经典作品占多数。请改用：

- **近期高分剧集 / 电影**
- **近期热门韩剧 / 国漫 / Netflix**
- **今日/本周热榜**

并适当提高 **近期榜单起始日期**（如 `2024-01-01`）。

### 详情页剧照很少

插件已通过 `include_image_language` 拉取多语言剧照。若仍偏少，多为 TMDB 源数据本身图片较少。

### `no results for: xxx`（搜索）

- 检查关键词拼写，可尝试英文原名
- 切换 **语言** 为 `en-US` 后再搜

---

## 参考链接

- [TMDB 官网](https://www.themoviedb.org/)
- [TMDB API 文档](https://developer.themoviedb.org/docs/getting-started)
- [API 设置（申请密钥）](https://www.themoviedb.org/settings/api)
- 插件内部分类与接口说明：[`docs/抓取/TMDB.md`](../../../docs/抓取/TMDB.md)
