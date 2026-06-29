# 歌曲宝
# URL： https://www.gequbao.com/


## channel 根据不同的分类自行构建
## 例如：
- 周杰伦：https://www.gequbao.com/topic/zhoujielun-3q7t5h?page=1
- 许嵩：https://www.gequbao.com/topic/xusong-2pifzk
- 毛不易：https://www.gequbao.com/topic/maobuyi-37knnq
- 陈奕迅：https://www.gequbao.com/topic/chenyixun-lsek3v
- 热门推荐：https://www.gequbao.com/hot-music
- 周排行：https://www.gequbao.com/top/week-download
- 抖音热歌：https://www.gequbao.com/s/%E6%8A%96%E9%9F%B3%E7%83%AD%E6%AD%8C

## 获取列表数据时需要 Cookie 和 User-Agent，这两个参数作为用户参数提供传递
## 参考下面的命令进行获取列表
```shell
curl --location --request GET 'https://www.gequbao.com/topic/chenyixun-lsek3v' \
--header 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36' \
--header 'Cookie: cf_clearance=divVPSg_B3XhOWLzW4Pg4jQLbbl86Jl_rHSY5lNohSw-1782723256-1.2.1.1-7jQYola4cJAyd46Fa6WpJLvugwuaXMYI_Yq5Teqjvd9Qth7RfPh3JAQPclm9y54bHeBAvf25QIzJX_Fyiw7RreT9qbWCP6g4b4CdudzsYM6i2qeTbydmiWJ_OdTtaIjFI2VMczVQX2X7GFWI_QF7EDOewnagtI9ssbyRqOAsO3HwygydLtgf3daJaXtkXakF0zOha4kUVCQw7u.ysoOCugZJ.CbgXio128Yi.pDYN2JgQTQjffRPaXuyk7HzJpa8hrCkqbZnhy_21iWy.qd5.F67Y.fuMSAqybKCYAfoH7ZoLOevpg_0FYI2vEksHUmAmDHwJ_Ld2mTY4Bjxr2W8pcejcImQ2ttpY35OHO5QTcgqwXhefmZYLg1jGBTZRP3lyJ9cwStwUEyGunXbr4rXiK.ALfRpAz.QjgQ5zMMkTsQQ8Mm7l5Su6lR19cCCoyiiAMUaF73iNSKmiVzkugGDqQzUxWeAq_121dcvC6_Ks8l83oK8SdELYo9uPcPCo_xR; first_referer=eyJpdiI6IlphUE5Fbi9yVFhYc3FPMEVhL3BrTlE9PSIsInZhbHVlIjoiWi83Q0VsOU9rZVZsN0wyeUdRQ0xBYmg4akkxVU5NMUlxcFVuZ3J1RHRwaG0xSFZocFVZRGZsU2M0OFhTdFE0MENvbmZ5QVRub1RxcTIwOUpld3BQSm9UVHRNUUhiekZrZ2dCTElnS2ZaaDhmdDIzVnQ0eHUxUGxDOEdENDF5eGE5YkFwajFQZHN2NXZpMVlaWVFrS29nPT0iLCJtYWMiOiI4MGU0YTY2MTY2ZDNjODBmMGQ2ODBiYmQyMGE5ZjgyOGQyZWQ5MjRlZGQwY2RkNzgwZjIxMGIwMzY3YWU1Njk1IiwidGFnIjoiIn0%3D; server_session_c355b968=f95807e1bc9a5bc4972966f5917f4e33; gequbao_session=eyJpdiI6IkJORWJTbU1DQ2NPTWkzTng1b0tBK2c9PSIsInZhbHVlIjoiRjdDcCtCV0gzWENIMFhDc3RFY0VENVJyV2xqVmRkUjNHNmxNRjJqWDFmbmd2ZisxNEtjbkRIblg4elJwT3IxZ3pMZ3NqczQxUmN5WXZQeTJ2eHhlVDUySWhjbzdaWlN4QXUzZEJ4Y0l4N1FtNWI1WEdHVk5zOWx6bU9MYnQ2RTAiLCJtYWMiOiI4MTE4OWEzYmIzYzIzN2I5ZjZjZGExYzhmZDJjYmUwZjEwYWQ2MzY2MDQ1MjZhYTFkNWQyYTBhODQyYmQ2NmI0IiwidGFnIjoiIn0%3D; gequbao_session=eyJpdiI6IjZQVTlyeFRRTGY5T1VhSi9xSm04MlE9PSIsInZhbHVlIjoiRTh0MmxoR2orV01ud2RIaHY4WUpjZmNMWkRSWGlQVjRIY0QvYlBER01HS1hiY1RvdzRpSENyT1A1QW92ekkyTVk0aHpmUlVPN2x4aU42N2w4R2FIT2dwL2dSUWhpc1Z6b2dLemRYVHFQeXpLdFBLeUpCY2U1V1dEZVF1aWYyL3MiLCJtYWMiOiJiNWE1ZDdmZTZhYWI2YWVjZDI2MjRmNDg1MjQ5MDgwYmY0MGQ1NzI4ZWM2ZjI5MDEzYmU4Mjc3YTJkNGRhNTg2IiwidGFnIjoiIn0%3D' \
--header 'Accept: */*' \
--header 'Host: www.gequbao.com' \
--header 'Connection: keep-alive'
```
## 解析： 
- 个人资料：
```html
<div class="card-header bg-white">
                <div class="media align-items-start">
                    <img class="artist-avatar mr-3"
                     src="https://img4.kuwo.cn/star/starheads/300/37/26/3816222178.jpg"
                     alt="陈奕迅歌曲合集">
                    <div class="media-body">
                        <h1 class="artist-name">
                            陈奕迅歌曲合集
                            <small class="badge badge-pill badge-dark">共300首</small>
                        </h1>
                        <p class="artist-desc scrollable">
                            陈奕迅（Eason Chan），1974年7月27日出生于香港，中国香港流行乐男歌手、演员，毕业于英国金斯顿大学。
                            1995年因获得第14届新秀歌唱大赛冠军而正式出道。1996年发行个人首张专辑《陈奕迅》。1997年主演个人首部电影《旺角大家姐》。1998年凭借歌曲《天下无双》在乐坛获得关注。2000年发行的歌曲《K歌之王》奠定其在歌坛的地位。2001年发行流行摇滚风格的专辑《反正是我》。2003年发行个人首张概念专辑《黑·白·灰》；专辑中的歌曲《十年》获得第4届百事音乐风云榜十大金曲奖。
                            20...
                        </p>
                    </div>
                </div>
            </div>
```
- 歌曲列表：
```html
<div class="row no-gutters py-2d5 border-top align-items-center">

                        <div class="col-8 col-md-7">
                            <a href="/music/6421" target="_blank" class="hover-zoom d-block text-decoration-none" title="十年 - 陈奕迅">

                                <div class="d-flex flex-column flex-md-row align-items-md-center">
                                    <div class="d-flex align-items-center mb-1 mb-md-0 min-w-0">
                                                                                <span class="text-primary font-weight-bold h6 mb-0 text-truncate">
                                        十年
                                    </span>
                                    </div>

                                    <div class="d-flex align-items-center ml-md-2 min-w-0">
                                        <span class="d-none d-md-inline text-muted mr-1 flex-shrink-0">-</span>
                                        
                                        <small class="text-jade font-weight-bold text-truncate">
                                            陈奕迅
                                        </small>
                                    </div>
                                </div>
                            </a>
                        </div>

                        <div class="col-md-2 text-center text-muted font-smaller d-none d-md-block">
                            03:25
                        </div>

                        <div class="col-4 col-md-3 text-right">
                            <a href="/music/6421" target="_blank" class="btn btn-sm btn-outline-blue rounded-pill font-sm" title="十年 - 陈奕迅">播放&amp;下载</a>
                        </div>
                    </div>
```


## 获取详情
## HTTP
```shell
curl --location --request GET 'https://www.gequbao.com/music/6421' \
--header 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36' \
--header 'Cookie: cf_clearance=divVPSg_B3XhOWLzW4Pg4jQLbbl86Jl_rHSY5lNohSw-1782723256-1.2.1.1-7jQYola4cJAyd46Fa6WpJLvugwuaXMYI_Yq5Teqjvd9Qth7RfPh3JAQPclm9y54bHeBAvf25QIzJX_Fyiw7RreT9qbWCP6g4b4CdudzsYM6i2qeTbydmiWJ_OdTtaIjFI2VMczVQX2X7GFWI_QF7EDOewnagtI9ssbyRqOAsO3HwygydLtgf3daJaXtkXakF0zOha4kUVCQw7u.ysoOCugZJ.CbgXio128Yi.pDYN2JgQTQjffRPaXuyk7HzJpa8hrCkqbZnhy_21iWy.qd5.F67Y.fuMSAqybKCYAfoH7ZoLOevpg_0FYI2vEksHUmAmDHwJ_Ld2mTY4Bjxr2W8pcejcImQ2ttpY35OHO5QTcgqwXhefmZYLg1jGBTZRP3lyJ9cwStwUEyGunXbr4rXiK.ALfRpAz.QjgQ5zMMkTsQQ8Mm7l5Su6lR19cCCoyiiAMUaF73iNSKmiVzkugGDqQzUxWeAq_121dcvC6_Ks8l83oK8SdELYo9uPcPCo_xR; first_referer=eyJpdiI6IlphUE5Fbi9yVFhYc3FPMEVhL3BrTlE9PSIsInZhbHVlIjoiWi83Q0VsOU9rZVZsN0wyeUdRQ0xBYmg4akkxVU5NMUlxcFVuZ3J1RHRwaG0xSFZocFVZRGZsU2M0OFhTdFE0MENvbmZ5QVRub1RxcTIwOUpld3BQSm9UVHRNUUhiekZrZ2dCTElnS2ZaaDhmdDIzVnQ0eHUxUGxDOEdENDF5eGE5YkFwajFQZHN2NXZpMVlaWVFrS29nPT0iLCJtYWMiOiI4MGU0YTY2MTY2ZDNjODBmMGQ2ODBiYmQyMGE5ZjgyOGQyZWQ5MjRlZGQwY2RkNzgwZjIxMGIwMzY3YWU1Njk1IiwidGFnIjoiIn0%3D; server_session_c355b968=f95807e1bc9a5bc4972966f5917f4e33; gequbao_session=eyJpdiI6IkJORWJTbU1DQ2NPTWkzTng1b0tBK2c9PSIsInZhbHVlIjoiRjdDcCtCV0gzWENIMFhDc3RFY0VENVJyV2xqVmRkUjNHNmxNRjJqWDFmbmd2ZisxNEtjbkRIblg4elJwT3IxZ3pMZ3NqczQxUmN5WXZQeTJ2eHhlVDUySWhjbzdaWlN4QXUzZEJ4Y0l4N1FtNWI1WEdHVk5zOWx6bU9MYnQ2RTAiLCJtYWMiOiI4MTE4OWEzYmIzYzIzN2I5ZjZjZGExYzhmZDJjYmUwZjEwYWQ2MzY2MDQ1MjZhYTFkNWQyYTBhODQyYmQ2NmI0IiwidGFnIjoiIn0%3D; gequbao_session=eyJpdiI6IjhhSDlNcnN3dktqU2lYOElhUkFNYVE9PSIsInZhbHVlIjoiRWkwSjJiN3VOemlsZ0xReWR4RmEzbkhvTzdlVlBYWDVVb0FYTmpXbnUrRjV2dXlmSk9iSHdYZkw3a1BJYm1KR0V6SnZUczk2UE5YdnBtS284WDFpTXVQbjkyMDFNMmJpbGNJZHBzaXgrYVFQdVA1djJhdWlFSTduNXNMb0JjK20iLCJtYWMiOiI1ZTI1YjYwZTllZjk0OWQwMTg4NDdjOWUzMTgxODgyNTQ0MmRmZGQ3ZDU3YTdkNTI1OWYxOWE0MDY5ZWIyZWQ2IiwidGFnIjoiIn0%3D' \
--header 'Accept: */*' \
--header 'Host: www.gequbao.com' \
--header 'Connection: keep-alive'
```

## 解析：
- 歌曲基本信息：
```html
    <script type="text/javascript">
        window.appData = JSON.parse('{\u0022mp3_id\u0022:6421,\u0022play_id\u0022:\u0022eyJpdiI6IlRLTXFqVDBWZmUvQ1JRbjRQZnlMd2c9PSIsInZhbHVlIjoiNHFBTVpNUzMzNFFhSG1NMklEclJvcnBjZnJhemFOMnVhcCtvUTRsT1JhV3ZiUHc4RHhRUFNWOEs1NUhnbjV0bUNJYTJlOUNwcjVXOHNCcmR5QjhJN2c9PSIsIm1hYyI6IjVmMDExMWI1MTlkNTE5MmY0NjZhZWRjNjMzMWIxODkyNmVhZTdmZjIwMTRmYzAwYzhhM2M0MWNhMGRlNjAxNzIiLCJ0YWciOiIifQ==\u0022,\u0022mp3_title\u0022:\u0022\\u5341\\u5e74\u0022,\u0022mp3_author\u0022:\u0022\\u9648\\u5955\\u8fc5\u0022,\u0022mp3_type\u0022:0,\u0022mp3_cover\u0022:\u0022http:\\\/\\\/img1.kuwo.cn\\\/star\\\/albumcover\\\/500\\\/91\\\/30\\\/2595482136.jpg\u0022,\u0022mp3_duration\u0022:\u002203:25\u0022,\u0022mp3_extra_urls\u0022:[{\u0022id\u0022:1859208,\u0022share_link\u0022:\u0022aHR0cHM6Ly9wYW4ucXVhcmsuY24vcy9kNzZmOWJhMTY4NjQ=\u0022,\u0022type\u0022:\u0022\\u5938\\u514b\\u7f51\\u76d8\u0022,\u0022color\u0022:\u0022#118AB2\u0022,\u0022compel_wap\u0022:false,\u0022icon\u0022:\u0022\\\/static\\\/img\\\/quark-icon.png\u0022},{\u0022id\u0022:2003953,\u0022share_link\u0022:\u0022aHR0cHM6Ly9wYW4uYmFpZHUuY29tL3MvMUlHMHZxdEVlTkhHaDBUSWQ4TGhRS2c\\\/cHdkPXdmV1c=\u0022,\u0022type\u0022:\u0022\\u767e\\u5ea6\\u7f51\\u76d8\u0022,\u0022color\u0022:\u0022#118AB2\u0022,\u0022compel_wap\u0022:false,\u0022icon\u0022:\u0022\\\/static\\\/img\\\/baidupan-icon.png\u0022}],\u0022ap_preload\u0022:\u0022metadata\u0022,\u0022is_robot\u0022:false,\u0022extra_url_compel\u0022:false,\u0022lrc_is_empty\u0022:false,\u0022ad_type\u0022:1,\u0022extra_recommend_wap_url\u0022:\u0022aHR0cHM6Ly9wYW4ucXVhcmsuY24vcy9kNzZmOWJhMTY4NjQ=\u0022,\u0022vip_down_url\u0022:true}');
    </script>
```
- 歌词获取
```html
<div class="content-lrc mt-1" id="content-lrc">[00:00.00]十年 - 陈奕迅 (Eason Chan)<br />
[00:03.85]词：林夕<br />
[00:07.71]曲：陈小霞<br />
[00:11.58]编曲：陈辉阳<br />
[00:15.44]如果那两个字没有颤抖<br />
[00:19.19]我不会发现 我难受<br />
[00:22.55]怎么说出口<br />
[00:26.16]也不过是分手<br />
[00:30.76]如果对于明天没有要求<br />
[00:34.84]牵牵手就像旅游<br />
[00:37.88]成千上万个门口<br />
[00:41.75]总有一个人要先走<br />
[00:47.49]怀抱既然不能逗留<br />
[00:50.90]何不在离开的时候<br />
[00:53.79]一边享受 一边泪流<br />
[01:01.00]十年之前<br />
[01:02.96]我不认识你<br />
[01:04.90]你不属于我<br />
[01:06.84]我们还是一样<br />
[01:09.23]陪在一个陌生人左右<br />
[01:13.06]走过渐渐熟悉的街头<br />
[01:16.54]十年之后<br />
[01:18.40]我们是朋友<br />
[01:20.37]还可以问候<br />
[01:22.32]只是那种温柔<br />
[01:24.75]再也找不到拥抱的理由<br />
[01:28.62]情人最后难免沦为朋友<br />
[01:57.15]怀抱既然不能逗留<br />
[02:00.59]何不在离开的时候<br />
[02:03.56]一边享受 一边泪流<br />
[02:10.78]十年之前<br />
[02:12.68]我不认识你<br />
[02:14.61]你不属于我<br />
[02:16.46]我们还是一样<br />
[02:18.90]陪在一个陌生人左右<br />
[02:22.81]走过渐渐熟悉的街头<br />
[02:26.22]十年之后 我们是朋友<br />
[02:30.06]还可以问候 只是那种温柔<br />
[02:34.40]再也找不到拥抱的理由<br />
[02:38.31]情人最后难免沦为朋友<br />
[02:48.09]直到和你做了多年朋友<br />
[02:52.37]才明白我的眼泪<br />
[02:55.24]不是为你而流<br />
[02:59.08]也为别人而流</div>
```

- 音频播放地址获取：
```shell
curl 'https://www.gequbao.com/member/common-play-url' \
  -H 'accept: application/json, text/javascript, */*; q=0.01' \
  -H 'accept-language: zh,en;q=0.9,zh-CN;q=0.8' \
  -H 'cache-control: no-cache' \
  -H 'content-type: application/x-www-form-urlencoded; charset=UTF-8' \
  -b 'cf_clearance=divVPSg_B3XhOWLzW4Pg4jQLbbl86Jl_rHSY5lNohSw-1782723256-1.2.1.1-7jQYola4cJAyd46Fa6WpJLvugwuaXMYI_Yq5Teqjvd9Qth7RfPh3JAQPclm9y54bHeBAvf25QIzJX_Fyiw7RreT9qbWCP6g4b4CdudzsYM6i2qeTbydmiWJ_OdTtaIjFI2VMczVQX2X7GFWI_QF7EDOewnagtI9ssbyRqOAsO3HwygydLtgf3daJaXtkXakF0zOha4kUVCQw7u.ysoOCugZJ.CbgXio128Yi.pDYN2JgQTQjffRPaXuyk7HzJpa8hrCkqbZnhy_21iWy.qd5.F67Y.fuMSAqybKCYAfoH7ZoLOevpg_0FYI2vEksHUmAmDHwJ_Ld2mTY4Bjxr2W8pcejcImQ2ttpY35OHO5QTcgqwXhefmZYLg1jGBTZRP3lyJ9cwStwUEyGunXbr4rXiK.ALfRpAz.QjgQ5zMMkTsQQ8Mm7l5Su6lR19cCCoyiiAMUaF73iNSKmiVzkugGDqQzUxWeAq_121dcvC6_Ks8l83oK8SdELYo9uPcPCo_xR; first_referer=eyJpdiI6IlphUE5Fbi9yVFhYc3FPMEVhL3BrTlE9PSIsInZhbHVlIjoiWi83Q0VsOU9rZVZsN0wyeUdRQ0xBYmg4akkxVU5NMUlxcFVuZ3J1RHRwaG0xSFZocFVZRGZsU2M0OFhTdFE0MENvbmZ5QVRub1RxcTIwOUpld3BQSm9UVHRNUUhiekZrZ2dCTElnS2ZaaDhmdDIzVnQ0eHUxUGxDOEdENDF5eGE5YkFwajFQZHN2NXZpMVlaWVFrS29nPT0iLCJtYWMiOiI4MGU0YTY2MTY2ZDNjODBmMGQ2ODBiYmQyMGE5ZjgyOGQyZWQ5MjRlZGQwY2RkNzgwZjIxMGIwMzY3YWU1Njk1IiwidGFnIjoiIn0%3D; server_session_c355b968=f95807e1bc9a5bc4972966f5917f4e33; gequbao_session=eyJpdiI6IllwSmFOVFBsaE9ua0tvWERWdUM3bXc9PSIsInZhbHVlIjoibXJMSFJIWTVPeDRnNkpxSUQvWVJURHBkcXNWdC9JWHg2UDFhTTdaYlplR3hHUW5ETWNVUTl2Z1ozQi83UER3NVljZUpyTjlMaHlROEdzSlZOZ0pQT05FSlpjQUViYVk2dEtqcUhoVWlwUHFZeEZ0TG1hTVJsYjYwNTkyZjJOcGQiLCJtYWMiOiI3YWQ1MWIxMzhlZmMxOWY3ZTViM2ZmYzhjYzY1ZDY4OTIxYmRhMmVjODMzZGEzMjFlZDA0OTlkYzVmMjRkZDNhIiwidGFnIjoiIn0%3D' \
  -H 'origin: https://www.gequbao.com' \
  -H 'pragma: no-cache' \
  -H 'priority: u=1, i' \
  -H 'referer: https://www.gequbao.com/music/6421' \
  -H 'sec-ch-ua: "Google Chrome";v="149", "Chromium";v="149", "Not)A;Brand";v="24"' \
  -H 'sec-ch-ua-arch: "arm"' \
  -H 'sec-ch-ua-bitness: "64"' \
  -H 'sec-ch-ua-full-version: "149.0.7827.155"' \
  -H 'sec-ch-ua-full-version-list: "Google Chrome";v="149.0.7827.155", "Chromium";v="149.0.7827.155", "Not)A;Brand";v="24.0.0.0"' \
  -H 'sec-ch-ua-mobile: ?0' \
  -H 'sec-ch-ua-model: ""' \
  -H 'sec-ch-ua-platform: "macOS"' \
  -H 'sec-ch-ua-platform-version: "26.2.0"' \
  -H 'sec-fetch-dest: empty' \
  -H 'sec-fetch-mode: cors' \
  -H 'sec-fetch-site: same-origin' \
  -H 'user-agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36' \
  -H 'x-requested-with: XMLHttpRequest' \
  --data-raw 'id=eyJpdiI6InNQRE8yeUxLQjA0U1NMazgybzdXcXc9PSIsInZhbHVlIjoiekx3a1ZaL3pPWXlZM0RzQkUwOElNTnJ1SkdxTGpxWFZPTWdEdWJ6d2lpb3VxTmhZOGFNYVFRMk1iMnh3Z1NKWnlPSGpSNGZBWDl2bG1JUmNEbWxOWnc9PSIsIm1hYyI6ImMzMGIyNDFkZmFmZTRjNjMwZmY4ZmM1ZGZhZmZmMzQ2OTAyODQ2ZTIzZGEwMzEwYTMwMjNmODU3Mjc1Mzc2NDQiLCJ0YWciOiIifQ%3D%3D'
```
响应：
```json
{
    "code": 1,
    "data": {
        "url": "https:\/\/kw-er.kuwo.cn\/270c333671128873e2e2e8176c6715c5\/6a423515\/resource\/30106\/trackmedia\/M500001OyHbk2MSIi4.mp3?bitrate$128&from=vip",
        "is_white_url": false,
        "ut": false
    },
    "msg": "\u64cd\u4f5c\u6210\u529f"
}
```