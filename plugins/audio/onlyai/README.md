# OnlyAI.fm

[OnlyAI.fm](https://onlyai.fm/) AI 音乐电台插件。

## Channels

| Channel | Route | Source |
|---------|-------|--------|
| 最新发布 | `/onlyai/latest` | `/api/radio/latest` |
| 直播热门 | `/onlyai/live` | `/api/radio/live-now` |
| 榜单 | `/onlyai/charts` | `/charts` JSON-LD |
| 流派 | `/onlyai/genre` | `/api/radio?genre=...` |

## Native test

```bash
make try-onlyai
# or
cd plugins/audio/onlyai
echo '{"action":"fetch","data":{"channelId":"latest","route":"/onlyai/latest","params":{"size":"5"}}}' | go run .
```

## Notes

- 列表项 `url` 为 MP3 直链（榜单频道为曲目页，需 detail 解析播放地址）。
- 流派频道支持 `sessionId` 分页（`/api/radio/extend`）。
- 无登录要求。
