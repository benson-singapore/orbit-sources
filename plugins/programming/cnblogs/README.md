# 博客园 Orbit 插件

博客园技术文章聚合插件，抓取 `https://www.cnblogs.com/` 首页、分类列表和文章详情。

## 功能

- 首页、所有随笔、所有文章、后端开发、Java、.NET、前端开发、人工智能频道
- 基于 `features.pagination` 的页码翻页
- 文章详情解析 `#cnblogs_post_body` 正文
- 列表解析标题、摘要、作者、作者头像、发布时间和阅读/评论/推荐信息

## 路由

- `/cnblogs/list` - 列表页，参数：`section`、`page`
- `/cnblogs/detail/:id` - 文章详情，参数：`id`

## 验证

```bash
make try PLUGIN=cnblogs CHANNEL=home ROUTE=/cnblogs/list PARAMS='{"section":"home","page":"1"}'
make try PLUGIN=cnblogs CHANNEL=home ROUTE=/cnblogs/list PARAMS='{"section":"home","page":"2"}'
make orbit PLUGIN=cnblogs
```
