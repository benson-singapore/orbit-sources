package main

import "testing"

const sampleListItemPage = `
<div class="row no-gutters align-items-center py-2d5 border-bottom position-relative list-item-hover">
    <div class="col-2 col-md-1 d-flex justify-content-center align-items-center">
        <span class="badge badge-light-red shadow-sm badge-square">1</span>
    </div>
    <div class="col-8 col-md-9 px-2">
        <a href="/music/29353795" class="text-dark font-weight-bold text-decoration-none text-truncate d-block text-hover-orange stretched-link" title="亲爱的你啊 - 任素汐">
            亲爱的你啊
            <span class="text-muted font-weight-normal small ml-2">- 任素汐</span>
        </a>
    </div>
    <div class="col-2 col-md-2 text-right pr-2 pr-md-3">
        <a href="/music/29353795" class="btn btn-sm btn-outline-orange rounded-pill" title="下载/播放">下载</a>
    </div>
</div>
<title>周下载排行 - 歌曲宝</title>
<a class="page-link" href="https://www.gequbao.com/top/week-download?page=2" rel="next">下一页</a>
`

func TestParseListItems(t *testing.T) {
	items := parseListItems(sampleListItemPage, "", "")
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	item := items[0]
	if item.ID != "29353795" {
		t.Fatalf("unexpected id: %s", item.ID)
	}
	if item.Title != "亲爱的你啊" {
		t.Fatalf("unexpected title: %s", item.Title)
	}
	if item.Author != "任素汐" {
		t.Fatalf("unexpected author: %s", item.Author)
	}
	if item.URL != "https://www.gequbao.com/music/29353795" {
		t.Fatalf("unexpected url: %s", item.URL)
	}
}

func TestHasNextPage(t *testing.T) {
	if !hasNextPage(sampleListItemPage, 2) {
		t.Fatal("expected page 2 to be detected")
	}
}

func TestStripPageParam(t *testing.T) {
	got := stripPageParam("https://www.gequbao.com/top/week-download?page=2")
	want := "https://www.gequbao.com/top/week-download"
	if got != want {
		t.Fatalf("stripPageParam = %q, want %q", got, want)
	}
}
