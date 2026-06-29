package main

import (
	"testing"
)

func video(id string) YouTubeVideo {
	return YouTubeVideo{
		ID: id,
		Snippet: VideoSnippet{
			Title: "video " + id,
		},
	}
}

func TestPaginateAfterLastIDFirstPage(t *testing.T) {
	pages := map[string]*YouTubeAPIResponse{
		"": {
			Items:         []YouTubeVideo{video("a"), video("b")},
			NextPageToken: "page2",
		},
	}

	items, hasMore, err := paginateAfterLastID("", func(pageToken string) (*YouTubeAPIResponse, error) {
		return pages[pageToken], nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 || items[0].ID != "a" || items[1].ID != "b" {
		t.Fatalf("unexpected items: %+v", items)
	}
	if !hasMore {
		t.Fatal("expected hasMore=true")
	}
}

func TestPaginateAfterLastIDLoadMore(t *testing.T) {
	pages := map[string]*YouTubeAPIResponse{
		"": {
			Items:         []YouTubeVideo{video("a"), video("b")},
			NextPageToken: "page2",
		},
		"page2": {
			Items: []YouTubeVideo{video("c"), video("d")},
		},
	}

	items, hasMore, err := paginateAfterLastID("b", func(pageToken string) (*YouTubeAPIResponse, error) {
		return pages[pageToken], nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 || items[0].ID != "c" || items[1].ID != "d" {
		t.Fatalf("unexpected items: %+v", items)
	}
	if hasMore {
		t.Fatal("expected hasMore=false")
	}
}

func TestPaginateAfterLastIDWithinPage(t *testing.T) {
	pages := map[string]*YouTubeAPIResponse{
		"": {
			Items:         []YouTubeVideo{video("a"), video("b"), video("c")},
			NextPageToken: "page2",
		},
	}

	items, hasMore, err := paginateAfterLastID("a", func(pageToken string) (*YouTubeAPIResponse, error) {
		return pages[pageToken], nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 || items[0].ID != "b" || items[1].ID != "c" {
		t.Fatalf("unexpected items: %+v", items)
	}
	if !hasMore {
		t.Fatal("expected hasMore=true")
	}
}

func TestPaginateAfterLastIDNotFound(t *testing.T) {
	pages := map[string]*YouTubeAPIResponse{
		"": {
			Items: []YouTubeVideo{video("a")},
		},
	}

	_, _, err := paginateAfterLastID("missing", func(pageToken string) (*YouTubeAPIResponse, error) {
		return pages[pageToken], nil
	})
	if err == nil {
		t.Fatal("expected error for missing cursor")
	}
}
