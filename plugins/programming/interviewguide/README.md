# 大厂面试指北 Orbit 插件

抓取 [《大厂面试指北》](https://notfound9.github.io/interviewGuide/#/)（[GitHub](https://github.com/NotFound9/interviewGuide)）面试题解答，以文章形式展示。

## 功能

- 按站点侧栏菜单分频道：首页、Java、Redis、MySQL、JVM、Kafka、ZooKeeper、HTTP、Spring、Nginx、系统设计、算法、大厂面试公众号文章系列、读书笔记、好书推荐
- 列表使用内嵌目录（与 `_sidebar.md` 一致）
- 详情拉取 Markdown 源文件并渲染为 HTML，过滤微信群 / 二维码等推广内容

## 路由

- `/interviewguide/list` — 列表，参数：`section`
- `/interviewguide/detail/:id` — 详情，参数：`id`（Docsify 页面 URL）

## 验证

```bash
make try PLUGIN=interviewguide CHANNEL=java ROUTE=/interviewguide/list PARAMS='{"section":"java"}'
make try PLUGIN=interviewguide CHANNEL=java ROUTE=/interviewguide/detail/:id PARAMS='{"id":"https://notfound9.github.io/interviewGuide/#/docs/JavaBasic"}'
make orbit PLUGIN=interviewguide
```
