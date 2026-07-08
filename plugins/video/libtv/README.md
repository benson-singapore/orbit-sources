# LibLib TV Plugin

LibLib TV 是一个 AI 视频创作社区，聚合了 AI 漫剧、短剧、专业影视、商业广告等多类型视频内容。

## 功能特性

- **9 个视频频道** - 分类浏览不同类型的 AI 生成视频
- **分页加载** - 支持无限滚动加载更多内容
- **视频播放** - HTML5 视频播放器集成
- **元数据丰富** - 包含视频描述、作者、发布时间等信息

## 频道列表

| ID | 频道名 |
|----|--------|
| 3036 | AI漫剧精卫计划 |
| 3040 | 广告导演请就位 |
| 1000 | 精选画布 |
| 1200 | 专业影视 |
| 1800 | 短剧漫剧 |
| 1100 | 商业广告 |
| 1300 | 动漫游戏 |
| 1500 | 教育生活 |
| 1600 | TV工具箱 |

## 数据结构

### Feed Item

每个视频项包含：
- `id` - 视频唯一标识
- `title` - 视频标题
- `url` - 视频 MP4 URL (可直接播放)
- `cover` - 封面图 URL
- `summary` - 描述摘要
- `content` - 完整描述
- `published_at` - 发布时间 (RFC3339)
- `author` - 作者昵称
- `author_avatar` - 作者头像 URL
- `tags` - 分类标签

### 分页

- 参数: `page` (从 1 开始)
- 返回: `hasMore` + `next` 字段
- 每页: 20 条视频

## API 端点

```
POST https://api.liblib.tv/api/community/project/template/feed/stream

请求体:
{
  "page": 1,
  "pageSize": 20,
  "tagId": 3036
}

响应:
{
  "code": 0,
  "data": {
    "list": [...],
    "hasMore": true,
    "total": 20
  }
}
```

## 实现细节

- **语言**: Go + WASM
- **编译**: `GOOS=wasip1 GOARCH=wasm go build`
- **打包**: Brotli 压缩 + ZIP 封装
- **超时**: 120 秒
- **内存**: 64 MB

## 测试

```bash
# Native 测试
make test-native CHANNEL=3036

# WASM 测试
make try-wasm PLUGIN=libtv

# 完整集成测试
make dev PLUGIN=libtv
```

## 许可

LibLib TV API 内容遵循原平台的服务条款。

## 更新日志

### v1.0.0 (2026-07-08)
- 初始版本
- 9 个频道支持
- 分页加载
- HTML5 视频播放器
