# 代码随想录 Orbit 插件

抓取 [代码随想录](https://programmercarl.com/) 算法题解与编程语言基础课，以文章形式展示。

## 功能

- 按侧栏专题分频道：算法性能分析、编程素养、求职、编程语言基础、数组、链表、哈希表、字符串、双指针、栈与队列、二叉树、回溯、贪心、动态规划、单调栈、图论、额外题目、感悟
- 列表使用内嵌目录（与站点侧栏一致）；语言基础频道额外包含 `/ke/*` 课程页
- 详情解析 `.theme-default-content` 正文，并过滤广告 / 营销外链

## 广告过滤

详情页会移除：

- 居中知识星球 / 训练营横幅图
- `kstar`、`xunlian`、卡码网营销页、京东联盟、微信公众号推广等外链
- 「算法公开课」等推广小节
- 保留：力扣题目链接、站内题解、B 站讲解、GitHub，以及题解配图 CDN

## 路由

- `/programmercarl/list` — 列表，参数：`section`
- `/programmercarl/detail/:id` — 详情，参数：`id`（文章 URL）

## 验证

```bash
make try PLUGIN=programmercarl CHANNEL=array ROUTE=/programmercarl/list PARAMS='{"section":"array"}'
make try PLUGIN=programmercarl CHANNEL=array ROUTE=/programmercarl/detail/:id PARAMS='{"id":"https://programmercarl.com/0704.%E4%BA%8C%E5%88%86%E6%9F%A5%E6%89%BE.html"}'
make orbit PLUGIN=programmercarl
```
