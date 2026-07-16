package main

import "testing"

func TestParseFloorSections(t *testing.T) {
	lines := []string{
		"NodeSeek有没有官方API可以推送站内通知？",
		"# NodeSeek有没有官方API可以推送站内通知？",
		"https://www.nodeseek.com/post-134606-1",
		"728days ago in日常",
		"#0",
		"如题",
		"0",
		"0",
		"0",
		"0",
		"#1",
		"可以抓到正文，但没有官方 API。",
		"1",
		"2",
		"3",
		"4",
		"登录或者注册后评论.",
	}

	sections := parseFloorSections(lines)
	if got := sections["#0"]; got != "如题" {
		t.Fatalf("unexpected #0 body: %q", got)
	}
	if got := sections["#1"]; got != "可以抓到正文，但没有官方 API。" {
		t.Fatalf("unexpected #1 body: %q", got)
	}
}

func TestExtractPublishedMeta(t *testing.T) {
	lines := []string{
		"https://www.nodeseek.com/post-1519-1",
		"1252days ago edited 1252days ago in日常",
	}

	publishedAt, category := extractPublishedMeta(lines)
	if publishedAt == "" {
		t.Fatal("publishedAt should not be empty")
	}
	if category != "日常" {
		t.Fatalf("unexpected category: %q", category)
	}
}

func TestNormalizePostID(t *testing.T) {
	if got := normalizePostID("https://www.nodeseek.com/post-1519-1"); got != "1519" {
		t.Fatalf("unexpected post id from url: %q", got)
	}
	if got := normalizePostID("1519"); got != "1519" {
		t.Fatalf("unexpected plain post id: %q", got)
	}
	if got := normalizePostID("post-abc"); got != "" {
		t.Fatalf("expected invalid id to be empty, got %q", got)
	}
}

func TestClassifyKind(t *testing.T) {
	if got := classifyKind("hello", 0, 0); got != "short" {
		t.Fatalf("expected short, got %q", got)
	}
	if got := classifyKind(string(make([]rune, 320)), 0, 0); got != "long" {
		t.Fatalf("expected long for long text, got %q", got)
	}
	if got := classifyKind("short", 0, 1); got != "long" {
		t.Fatalf("expected long when comments exist, got %q", got)
	}
}
