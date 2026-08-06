package main

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestCleanMarkdownRemovesPromo(t *testing.T) {
	raw := `# 标题

(PS：扫描首页里面的二维码进群，分享技术资料)

正文第一段。

![](http://notfound9.github.io/interviewGuide/static/wdsfsdfsmaster.png)

### 多态是什么？

多态说明。
`
	out := cleanMarkdown(raw)
	if strings.Contains(out, "进群") {
		t.Fatalf("expected promo line removed, got:\n%s", out)
	}
	if strings.Contains(out, "wdsfsdfsmaster") {
		t.Fatalf("expected promo image removed, got:\n%s", out)
	}
	if !strings.Contains(out, "多态是什么") {
		t.Fatalf("expected content kept, got:\n%s", out)
	}
}

func TestCleanArticleHTMLRewritesLinks(t *testing.T) {
	html := `
<div class="markdown-body">
<p><a href="docs/JavaBasic.md">基础</a></p>
<p><img src="static/demo.png" alt="图"></p>
<p>扫描二维码进群学习</p>
</div>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	root := doc.Find(".markdown-body").First()
	cleanArticleHTML(root)
	out, _ := root.Html()

	if !strings.Contains(out, "interviewGuide/#/docs/JavaBasic") {
		t.Fatalf("expected docsify link rewrite, got:\n%s", out)
	}
	if !strings.Contains(out, "interviewGuide/static/demo.png") {
		t.Fatalf("expected absolute image url, got:\n%s", out)
	}
	if strings.Contains(out, "进群") {
		t.Fatalf("expected promo paragraph removed, got:\n%s", out)
	}
}

func TestMarkdownPathFromID(t *testing.T) {
	cases := map[string]string{
		"https://notfound9.github.io/interviewGuide/#/docs/JavaBasic": "docs/JavaBasic.md",
		"https://notfound9.github.io/interviewGuide/docs/Lock.md":     "docs/Lock.md",
		"docs/RedisBasic.md": "docs/RedisBasic.md",
		"README.md":          "README.md",
	}
	for in, want := range cases {
		got := markdownPathFromID(in)
		if got != want {
			t.Fatalf("markdownPathFromID(%q)=%q want %q", in, got, want)
		}
	}
}
