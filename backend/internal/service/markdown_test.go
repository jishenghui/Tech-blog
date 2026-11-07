package service

import (
	"strings"
	"testing"
)

func TestMarkdownRenderer(t *testing.T) {
	renderer := NewMarkdownRenderer()
	rendered, err := renderer.Render("测试标题", `# 一级标题

这是正文，包含 **强调** 和 [链接](https://example.com)。

`+"```go"+`
fmt.Println("hello")
`+"```"+`

`+"```mermaid"+`
graph TD;
A-->B;
`+"```"+`

行内公式 $x^2 + y^2 = z^2$。

$$
\int_0^1 x^2 dx
$$

| 列1 | 列2 |
| --- | --- |
| a | b |
`)
	if err != nil {
		t.Fatalf("渲染 Markdown 失败: %v", err)
	}

	checks := []string{
		`<h1`,
		`class="chroma"`,
		`class="mermaid"`,
		`A--&gt;B;`,
		`class="math inline"`,
		`class="math display"`,
		`<table>`,
		`<a href="https://example.com">链接</a>`,
	}
	for _, check := range checks {
		if !strings.Contains(rendered.HTML, check) {
			t.Fatalf("渲染结果缺少 %q：\n%s", check, rendered.HTML)
		}
	}

	if rendered.WordCount == 0 {
		t.Fatal("字数统计不应为 0")
	}
	if rendered.ReadingTime < 1 {
		t.Fatal("阅读时长应至少为 1 分钟")
	}
	if !strings.Contains(rendered.SearchText, "测试标题") || !strings.Contains(rendered.SearchText, "正文") {
		t.Fatalf("搜索文本不正确: %q", rendered.SearchText)
	}
	if strings.Contains(rendered.SearchText, `\(`) || strings.Contains(rendered.Excerpt, `\(`) {
		t.Fatalf("摘要/搜索文本不应包含 LaTeX: excerpt=%q search=%q", rendered.Excerpt, rendered.SearchText)
	}
	if rendered.Excerpt == "" {
		t.Fatal("摘要不应为空")
	}
}

func TestExcerptTruncatesByRunes(t *testing.T) {
	got := excerpt("一二三四五六", 4)
	if got != "一二三四…" {
		t.Fatalf("摘要截断不正确: %q", got)
	}
}
