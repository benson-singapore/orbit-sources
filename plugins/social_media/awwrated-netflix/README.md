# awwrated · Netflix

按 [awwrated](https://awwrated.com) 页面筛选条件构建频道，结构与站点过滤栏一致。

## Channel 结构

| 分组 | 频道 |
|------|------|
| 形态 | 电影 / 电视影集 / 动画 / 动画电影 |
| 排序 | 新上架 / 近期好评 / 最好评 / 热门 aww 评 / 最热门 / 最新 / 最旧 / 最…!!? |
| 地区 | 美国、台湾、韩国、日本、英国、中国、香港、印度、泰国 |
| 类型 | 编辑推荐、剧情、恐怖…及改编自游戏/漫画/小说/真实故事等 |
| 年份 | 2025–2020、2010-2019、2000-2009、1980-1999 |
| 其它 | 即将下架、搜索 |

默认频道：`新上架`。详情含 aww / IMDb / 豆瓣 / 烂番茄 / Metascore / IGN 评分与预告片。

## 测试

```bash
make try PLUGIN=awwrated-netflix CHANNEL=sort_new ROUTE=/awwrated-netflix/list PARAMS='{"orderby":"release_date","order":"DESC"}'
make try PLUGIN=awwrated-netflix CHANNEL=genre_drama ROUTE=/awwrated-netflix/list PARAMS='{"tag":"Drama","orderby":"average_review","order":"DESC"}'
make try PLUGIN=awwrated-netflix CHANNEL=region_jp ROUTE=/awwrated-netflix/list PARAMS='{"country":"Japan","orderby":"release_date","order":"DESC"}'
make try PLUGIN=awwrated-netflix CHANNEL=year_2024 ROUTE=/awwrated-netflix/list PARAMS='{"yearFrom":"1704067200","yearTo":"1735689599","orderby":"release_date","order":"DESC"}'
```
