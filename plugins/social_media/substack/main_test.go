package main

import (
	"testing"

	"github.com/orbit-tauri-tools/plugin-sdk"
)

func TestClassifyKind(t *testing.T) {
	short := classifyKind("Hello world", nil, nil)
	if short != "short" {
		t.Fatalf("expected short, got %s", short)
	}

	longBody := classifyKind(string(make([]rune, 300)), nil, nil)
	if longBody != "long" {
		t.Fatalf("expected long for long body, got %s", longBody)
	}

	quote := classifyKind("hi", nil, &sdk.SocialQuote{ID: "1", Author: "a", Body: "q"})
	if quote != "long" {
		t.Fatalf("expected long with quote, got %s", quote)
	}
}

func TestNoteTitle(t *testing.T) {
	if got := noteTitle("First line\nSecond"); got != "First line" {
		t.Fatalf("unexpected title: %q", got)
	}
}

func TestNormalizeFeedCursor(t *testing.T) {
	if got := normalizeFeedCursor("1"); got != "" {
		t.Fatalf("numeric cursor should be ignored, got %q", got)
	}
	token := "eyJzZXNzaW9uX2lkIjoiYWJjIn0"
	if got := normalizeFeedCursor(token); got != token {
		t.Fatalf("opaque cursor should be kept, got %q", got)
	}
}
