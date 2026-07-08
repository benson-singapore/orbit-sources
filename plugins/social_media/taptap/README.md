# TapTap

TapTap game ranking feed plugin for Orbit.

## Channels

- 热门榜
- 预约榜
- 热玩榜
- 新品榜
- 热卖榜
- 新版本榜
- 独家榜
- 动作榜
- 策略榜
- 放置榜
- 单机榜
- 休闲榜
- 沙盒生存榜
- 模拟经营榜
- 解谜榜
- 射击榜
- 多人对战榜
- 二次元榜
- 音乐节奏榜
- 剧情榜
- 武侠榜
- 女性向榜
- 独立游戏榜
- Roguelike榜

Data source: `https://www.taptap.cn/webapiv2/app-top/v2/hits`.

## Detail

Detail route: `/taptap/detail/:id`.

Uses `https://www.taptap.cn/webapiv2/app/v2/detail-by-id/{id}` to return game intro, update notes, screenshots, and video metadata. If TapTap exposes a playable HLS URL in the video payload, it is rendered with an HTML `<video>` tag; current public payloads usually expose only video thumbnails.
