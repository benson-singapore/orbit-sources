package main

import (
	"os"
	"strings"
	"testing"
)

func TestParseArticlePage(t *testing.T) {
	html, err := os.ReadFile("testdata/post_7646780176223010826.html")
	if err != nil {
		t.Skip("testdata not present; run: curl -sL https://juejin.cn/post/7646780176223010826 -o plugins/programming/juejin/testdata/post_7646780176223010826.html")
	}
	detail := parseArticlePage(string(html))
	if detail.Content == "" {
		t.Fatal("expected content")
	}
	if !strings.Contains(detail.Content, "markdown-body") {
		t.Fatalf("content should include article html, got prefix %q", detail.Content[:80])
	}
	if strings.Contains(detail.Content, `\u003C`) {
		t.Fatalf("content should be unescaped, got prefix %q", detail.Content[:80])
	}
	if detail.Cover == "" {
		t.Fatal("expected cover")
	}
	if detail.PublishedAt != 1780454059 {
		t.Fatalf("publishedAt = %d, want 1780454059", detail.PublishedAt)
	}
	if len(detail.Tags) != 3 {
		t.Fatalf("tags = %v", detail.Tags)
	}
	if !strings.Contains(detail.Summary, "Monorepo") {
		t.Fatalf("summary = %q", detail.Summary)
	}
	if detail.Title != "单仓库下的四十模块 —— React Monorepo 工程架构拆解" {
		t.Fatalf("title = %q", detail.Title)
	}
	if detail.Author != "老王以为" {
		t.Fatalf("author = %q", detail.Author)
	}
}
