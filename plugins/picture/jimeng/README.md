# 即梦 AI 图片插件

抓取 [即梦 AI](https://jimeng.jianying.com/ai-tool/home/?type=image) 发现页热门趋势 AI 图片，每条 feed 的 `content` 含大图与完整生成提示词。

## 特性

- 实时抓取（`refresh=true`）
- 不落库（`persist=false`）
- 无需登录或 API Key
- 列表封面 + 详情 `content` 展示图片与提示词

## 频道

| channel | 说明 |
|---------|------|
| `trending` | 发现页热门趋势（混合拉取，仅输出含图片的条目） |
| `discover-image` | 发现页图片（API 层过滤仅图片） |

## 参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `categoryId` | `11222` | 发现页分类 ID |
| `page` | `1` | 页码，从 1 开始；插件内部换算 `offset = (page - 1) × 40` |
| `workType` | `video,image,canvas` | 作品类型过滤，逗号分隔 |

## 分页与 `seenIds`

完整规范见 **[docs/pagination-seenids.md](../../docs/pagination-seenids.md)**（传参规则、manifest 声明、客户端合并 `next`、测试方法）。

简要规则：

| 场景 | `page` | `seenIds` |
|------|--------|-----------|
| 首屏 / 下拉刷新 | `1` | `""` |
| 加载更多 | `next.page` | `next.seenIds` |

## 数据源

- 接口：`POST /mweb/v1/get_explore`
- 提示词：`aigc_image_params.text2image_params.prompt`
- 原图：`image.large_images[0].image_url`（回退 `cover_url_map` / `cover_url`）

## 说明

请求仅使用 `sign` 头签名，不携带浏览器侧的 `msToken` / `a_bogus`（过期反而会导致空响应）。
