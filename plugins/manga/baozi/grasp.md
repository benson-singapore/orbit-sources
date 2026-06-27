# 包子漫画
# 网址：https://baozimh.org/
# API：https://v2.apikk.top
# 封面 CDN：https://c-nc-1.6wm.top/manga/


# channel 分类
- 热门漫画：https://baozimh.org/manga-genre/hots
- 国漫：https://baozimh.org/manga-genre/cn
- 日漫：https://baozimh.org/manga-genre/jp
- 韩漫：https://baozimh.org/manga-genre/kr
- 欧美：https://baozimh.org/manga-genre/ou-mei
- 恋爱：https://baozimh.org/manga-tag/lianai
- 古风：https://baozimh.org/manga-tag/gufeng
- 玄幻：https://baozimh.org/manga-tag/xuanhuan
- 异能：https://baozimh.org/manga-tag/yineng
- 悬疑：https://baozimh.org/manga-tag/xuanyi
- 科幻：https://baozimh.org/manga-tag/kehuan
- 穿越：https://baozimh.org/manga-tag/chuanyue
- 冒险：https://baozimh.org/manga-tag/mouxian
- 热血：https://baozimh.org/manga-tag/rexie
- 搞笑：https://baozimh.org/manga-tag/gaoxiao
- 都市：https://baozimh.org/manga-tag/dushi
- 后宫：https://baozimh.org/manga-tag/hougong
- 其他：https://baozimh.org/manga-genre/qita
- 搜索：https://baozimh.org/s?q=%E5%87%A1%E4%BA%BA%E4%BF%AE%E4%BB%99%E4%BC%A0

分页：第 1 页为裸路径（如 `https://baozimh.org/manga-genre/hots`），第 2 页起路径段比页码少 1（第 2 页 → `/page/1`，第 3 页 → `/page/2`）


# channel 列表获取
## URL 示例
- 热门漫画：`https://baozimh.org/manga-genre/hots`
- 国漫：`https://baozimh.org/manga-genre/cn`
- 标签类：`https://baozimh.org/manga-tag/{slug}`
- 搜索：`https://baozimh.org/s?q={关键词}`

## 解析
正则匹配漫画卡片（`main.go` → `reComicCard`）：

```
<a href="/manga/{slug}"> ... <img class="card" src="..."> ... <h3 class="cardtitle">{标题}</h3>
```

示例 HTML：

```html
<a href="/manga/taijianzhuanshi">
  <img class="card" src="https://c-nc-1.6wm.top/manga/taijianzhuanshi/ca08948a6d1c60e15b34135d9b98acbd.webp">
  <h3 class="cardtitle">太监转世</h3>
</a>
```

字段映射：
- `id` / `slug`：`href` 中 `/manga/` 后的路径
- `title`：`h3.cardtitle` 文本
- `cover`：`img.card` 的 `src`（CDN 域名 `c-nc-1.6wm.top`）

下一页判断：页面 HTML 是否包含 `href="{listPath}/page/{page}"`（即用户第 N 页的下一页链接为 `/page/N`）


# chapter 列表获取
## 步骤
1. 请求漫画页：`https://baozimh.org/manga/{slug}`
2. 从 HTML 提取 `data-mid="{mid}"`（正则 `reMangaMID`）
3. 请求章节 API：`https://v2.apikk.top/api/manga/get?mid={mid}&mode=all`
   - 请求头需带 `Referer: https://baozimh.org/manga/{slug}`

## API 响应示例

```json
{
  "status": true,
  "data": {
    "id": "12",
    "title": "武炼巅峰",
    "slug": "wuliandianfeng-pikapi",
    "cover": "...",
    "desc": "...",
    "chapters": [
      {
        "id": "907630",
        "attributes": {
          "title": "1 扫地小厮",
          "slug": "0_0",
          "order": 0,
          "updatedAt": "2024-09-03T04:48:26Z"
        }
      }
    ]
  }
}
```

字段映射：
- 章节 `id`：API `chapters[].id`（用于阅读接口的 `chapterId`）
- 章节标题：`chapters[].attributes.title`
- 章节 URL：`https://baozimh.org/manga/{slug}/{chapter_slug}`


# 获取章节内容（图片）
## 步骤
1. 同上获取漫画页 `data-mid`
2. 请求：`https://v2.apikk.top/api/v2/chapter/getinfo?m={mid}&c={chapterId}`
   - 请求头需带 `Referer: https://baozimh.org/manga/{slug}`

## API 响应关键字段

```json
{
  "status": true,
  "data": {
    "info": {
      "title": "1 扫地小厮",
      "slug": "0_0",
      "images": {
        "line": 2,
        "images": "J7r...nQ"
      }
    }
  }
}
```

- `images.images`：加密字符串，需解码（见 `decoder.go`）
- `images.line`：图片 CDN 线路
  - `line == 2` → `https://f40-1-4.g-mh.online`
  - 其他 → `https://t40-1-4.g-mh.online`

## 图片解密（decoder.go）
加密串格式：`J7r` + body + `nQ`

解密流程：
1. 在 body 中遍历所有 `kD` 标记位置（部分章节含多个干扰 `kD`）
2. 对每个位置按布局切分：`part1 | kD | part2 | checksum(3) | cnt`
3. 拼接 `cnt + part1 + part2`，每 7 字符一组、奇数组反转
4. 字母表映射后 URL-safe Base64 解码，得到 JSON 数组

解码后 JSON 示例：

```json
[
  { "order": 0, "url": "/scomic/wuliandianfeng-pikapi/0/0-98bj/0.jpg" },
  { "order": 1, "url": "/scomic/wuliandianfeng-pikapi/0/0-98bj/1.jpg" }
]
```

最终图片 URL：`{cdn}{url}`（`url` 已是绝对地址则直接使用）


# 介绍内容查询
## URL
`https://baozimh.org/manga/{slug}`

## 解析（优先 og meta）
- 名称：`meta[property="og:title"]`（去掉后缀 `-包子漫畫 - 包子漫畫`）
- 封面：`meta[property="og:image"]`
- 简介：`meta[property="og:description"]`
- 备用标题：`h1.title`

插件在章节列表第一项返回 `intro` 虚拟章节，点击后渲染简介页。
