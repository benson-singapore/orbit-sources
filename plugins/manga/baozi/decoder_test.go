package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDecodeChapterImages(t *testing.T) {
	raw, err := os.ReadFile(".tmp/chapter_api.json")
	if err != nil {
		t.Skip("sample not found")
	}
	var payload struct {
		Data struct {
			Info struct {
				Images struct {
					Images string `json:"images"`
				} `json:"images"`
			} `json:"info"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	pages, err := decodeChapterImages(payload.Data.Info.Images.Images)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 17 {
		t.Fatalf("want 17 pages, got %d", len(pages))
	}
}
