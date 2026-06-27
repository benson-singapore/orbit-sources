package main

import (
	"sync"

	"github.com/orbit-tauri-tools/plugin-sdk"
)

const enrichConcurrency = 8

func enrichFeedItems(items []sdk.FeedItem) []sdk.FeedItem {
	if len(items) == 0 {
		return items
	}
	out := make([]sdk.FeedItem, len(items))
	type slot struct {
		i int
	}
	ch := make(chan slot, len(items))
	sem := make(chan struct{}, enrichConcurrency)
	var wg sync.WaitGroup
	for i, item := range items {
		wg.Add(1)
		go func(i int, item sdk.FeedItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = enrichFeedItem(item)
			ch <- slot{i: i}
		}(i, item)
	}
	wg.Wait()
	close(ch)
	return out
}
