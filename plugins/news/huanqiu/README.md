# 环球网

Orbit WASM feed plugin for [环球网](https://www.huanqiu.com/).

## Routes

- `/huanqiu/list` with `section` and optional `page` (default `1`)
- `/huanqiu/detail/:id` with `id` from `item.url`

## Sections

- `home`
- `world`
- `china`
- `mil`
- `taiwan`
- `opinion`
- `finance`
- `tech`
- `society`
- `health`
- `sports`
- `auto`
- `ent`

## Notes

The plugin reads article metadata and body HTML from Huanqiu pages. Huanqiu content is copyrighted; use the feed output in accordance with the site's terms and copyright notice.
