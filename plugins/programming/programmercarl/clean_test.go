package main

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestCleanArticleContentRemovesPromoBanner(t *testing.T) {
	html := `
<div class="theme-default-content">
<p align="center"><a href="https://programmercarl.com/other/kstar.html"><img src="https://file1.kamacoder.com/i/web/banner.jpg" width="1000"></a></p>
<h1>704. 二分查找</h1>
<p><a href="https://leetcode.cn/problems/binary-search/">力扣题目链接</a></p>
<h2>算法公开课</h2>
<p><a href="https://programmercarl.com/about/gongkaike.html">公开课</a> <a href="https://www.bilibili.com/video/BV1fA4y1o715">视频</a></p>
<h2>思路</h2>
<p>正文说明</p>
<p><img src="https://file1.kamacoder.com/i/algo/demo.jpg" alt="图"></p>
<p><a href="https://kamacoder.com/courseshop.php">卡码网课程</a></p>
</div>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	root := doc.Find(".theme-default-content").First()
	cleanArticleContent(root)
	out, _ := root.Html()

	mustNotContain := []string{
		"kstar.html",
		"算法公开课",
		"gongkaike.html",
		"courseshop.php",
		"banner.jpg",
	}
	for _, s := range mustNotContain {
		if strings.Contains(out, s) {
			t.Fatalf("expected cleaned html to remove %q, got:\n%s", s, out)
		}
	}
	mustContain := []string{
		"704. 二分查找",
		"leetcode.cn",
		"思路",
		"正文说明",
		"i/algo/demo.jpg",
	}
	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Fatalf("expected cleaned html to keep %q, got:\n%s", s, out)
		}
	}
}

func TestIsPromoURL(t *testing.T) {
	cases := map[string]bool{
		"https://programmercarl.com/other/kstar.html": true,
		"https://programmercarl.com/xunlian/bagu.html": true,
		"https://kamacoder.com/":                      true,
		"https://leetcode.cn/problems/binary-search/": false,
		"https://www.bilibili.com/video/BV1fA4y1o715": false,
		"https://programmercarl.com/0704.二分查找.html":   false,
		"#思路": false,
	}
	for href, want := range cases {
		if got := isPromoURL(href); got != want {
			t.Fatalf("isPromoURL(%q)=%v want %v", href, got, want)
		}
	}
}

func TestCatalogHasArray(t *testing.T) {
	sec := catalogByID["array"]
	if sec == nil || len(sec.Items) == 0 {
		t.Fatal("array catalog missing")
	}
	if sec.Items[0].Path != "/数组理论基础.html" {
		t.Fatalf("unexpected first item: %+v", sec.Items[0])
	}
}
