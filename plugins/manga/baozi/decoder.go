package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

const (
	encPrefix       = "J7r"
	encSuffix       = "nQ"
	encKey          = "kD"
	encChecksumLen  = 3
	encChunkSize    = 7
	encCountDivisor = 3

	stdAlphabet    = "_-9876543210abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	customAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
)

func decodeChapterImages(encrypted string) ([]decodedPage, error) {
	encrypted = strings.TrimSpace(encrypted)
	if !strings.HasPrefix(encrypted, encPrefix) || !strings.HasSuffix(encrypted, encSuffix) {
		return nil, fmt.Errorf("invalid encrypted payload")
	}

	body := encrypted[len(encPrefix) : len(encrypted)-len(encSuffix)]

	var lastErr error
	for searchFrom := 0; ; {
		kdIdx := strings.Index(body[searchFrom:], encKey)
		if kdIdx < 0 {
			break
		}
		kdIdx += searchFrom
		pages, err := decodeChapterBody(body, kdIdx)
		if err == nil {
			return pages, nil
		}
		lastErr = err
		searchFrom = kdIdx + len(encKey)
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("chapter key marker not found")
}

func decodeChapterBody(body string, kdIdx int) ([]decodedPage, error) {
	checksumLen := len(body) - len(encKey) - encChecksumLen
	if checksumLen <= 0 {
		return nil, fmt.Errorf("invalid chapter body")
	}

	cntLen := int(math.Floor(float64(checksumLen) / encCountDivisor))
	part2Len := len(body) - kdIdx - len(encKey) - encChecksumLen - cntLen
	if part2Len < 0 {
		return nil, fmt.Errorf("invalid chapter layout")
	}

	part1 := body[:kdIdx]
	part2 := body[kdIdx+len(encKey) : kdIdx+len(encKey)+part2Len]
	cnt := body[kdIdx+len(encKey)+part2Len+encChecksumLen:]

	if len(cnt) != cntLen {
		return nil, fmt.Errorf("chapter checksum mismatch")
	}

	joined := cnt + part1 + part2
	mapped := mapCustomAlphabet(reverseEveryOtherChunk(joined, encChunkSize))
	if mapped == "" {
		return nil, fmt.Errorf("invalid chapter alphabet")
	}
	plain, err := decodeURLSafeBase64(mapped)
	if err != nil {
		return nil, err
	}

	var pages []decodedPage
	if err := json.Unmarshal([]byte(plain), &pages); err != nil {
		return nil, fmt.Errorf("parse decoded chapter json: %w", err)
	}
	return pages, nil
}

func reverseEveryOtherChunk(s string, size int) string {
	if size <= 0 || s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i, chunk := 0, 0; i < len(s); i, chunk = i+size, chunk+1 {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		part := s[i:end]
		if chunk%2 == 1 {
			part = reverseString(part)
		}
		b.WriteString(part)
	}
	return b.String()
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func mapCustomAlphabet(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		idx := strings.IndexByte(stdAlphabet, ch)
		if idx < 0 {
			return ""
		}
		b.WriteByte(customAlphabet[idx])
	}
	return b.String()
}

func decodeURLSafeBase64(s string) (string, error) {
	if pad := (4 - len(s)%4) % 4; pad > 0 {
		s += strings.Repeat("=", pad)
	}
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	return string(raw), nil
}
