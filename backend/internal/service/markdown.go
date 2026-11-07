package service

import (
	"bytes"
	"html"
	"regexp"
	"strings"
	"unicode"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	xhtml "golang.org/x/net/html"
)

// MarkdownRenderer 负责把 Markdown 渲染为 content_html，并计算搜索与统计字段。
type MarkdownRenderer struct {
	md        goldmark.Markdown
	style     *chroma.Style
	formatter *chromahtml.Formatter
}

// Rendered 是一次渲染得到的全部派生数据。
type Rendered struct {
	HTML        string
	PlainText   string
	WordCount   int
	ReadingTime int
	SearchText  string
	Excerpt     string
}

func NewMarkdownRenderer() *MarkdownRenderer {
	style := styles.Get("github-dark")
	if style == nil {
		style = styles.Get("github")
	}
	if style == nil {
		style = styles.Fallback
	}
	formatter := chromahtml.New(chromahtml.WithClasses(true))

	codeRenderer := &codeBlockRenderer{style: style, formatter: formatter}
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			mathjax.MathJax,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			gmhtml.WithHardWraps(),
			// goldmark 的默认 HTML 渲染器优先级为 1000，按降序注册，
			// 后注册者覆盖先注册者，因此这里必须用更低的优先级。
			renderer.WithNodeRenderers(util.Prioritized(codeRenderer, 500)),
		),
	)
	return &MarkdownRenderer{md: md, style: style, formatter: formatter}
}

// Render 输出 HTML，并计算纯文本、字数、阅读时长、搜索文本与摘要。
func (r *MarkdownRenderer) Render(title, markdown string) (Rendered, error) {
	var buf bytes.Buffer
	if err := r.md.Convert([]byte(markdown), &buf); err != nil {
		return Rendered{}, err
	}
	renderedHTML := buf.String()
	plain := plainTextFromHTML(renderedHTML)
	words := countWords(plain)
	readingTime := 0
	if words > 0 {
		readingTime = (words + 299) / 300
	}
	// 搜索文本统一小写，使英文搜索不区分大小写；中文不受影响。
	searchText := strings.ToLower(strings.TrimSpace(title + " " + plain))

	return Rendered{
		HTML:        renderedHTML,
		PlainText:   plain,
		WordCount:   words,
		ReadingTime: readingTime,
		SearchText:  searchText,
		Excerpt:     excerpt(plain, 120),
	}, nil
}

type codeBlockRenderer struct {
	style     *chroma.Style
	formatter *chromahtml.Formatter
}

func (r *codeBlockRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
}

func (r *codeBlockRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	fenced := node.(*ast.FencedCodeBlock)
	lang := strings.ToLower(strings.TrimSpace(string(fenced.Language(source))))
	code := fencedCodeText(fenced, source)

	// Mermaid 交给前台脚本渲染，后端只输出可识别的容器。
	if lang == "mermaid" {
		_, _ = w.WriteString(`<pre class="mermaid">`)
		_, _ = w.WriteString(html.EscapeString(code))
		_, _ = w.WriteString("</pre>\n")
		return ast.WalkSkipChildren, nil
	}

	if lang == "" {
		_, _ = w.WriteString(`<pre><code>`)
		_, _ = w.WriteString(html.EscapeString(code))
		_, _ = w.WriteString("</code></pre>\n")
		return ast.WalkSkipChildren, nil
	}

	lexer := lexers.Get(lang)
	if lexer == nil {
		_, _ = w.WriteString(`<pre><code class="language-`)
		_, _ = w.WriteString(html.EscapeString(lang))
		_, _ = w.WriteString(`">`)
		_, _ = w.WriteString(html.EscapeString(code))
		_, _ = w.WriteString("</code></pre>\n")
		return ast.WalkSkipChildren, nil
	}

	lexer = chroma.Coalesce(lexer)
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		_, _ = w.WriteString(`<pre><code>`)
		_, _ = w.WriteString(html.EscapeString(code))
		_, _ = w.WriteString("</code></pre>\n")
		return ast.WalkSkipChildren, nil
	}

	var highlighted bytes.Buffer
	if err := r.formatter.Format(&highlighted, r.style, iterator); err != nil {
		_, _ = w.WriteString(`<pre><code>`)
		_, _ = w.WriteString(html.EscapeString(code))
		_, _ = w.WriteString("</code></pre>\n")
		return ast.WalkSkipChildren, nil
	}

	_, _ = w.WriteString(`<div class="highlight">`)
	_, _ = w.Write(highlighted.Bytes())
	_, _ = w.WriteString("</div>\n")
	return ast.WalkSkipChildren, nil
}

func fencedCodeText(node *ast.FencedCodeBlock, source []byte) string {
	var b strings.Builder
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		b.Write(line.Value(source))
	}
	return strings.TrimRight(b.String(), "\n")
}

// plainTextFromHTML 提取可见文本，用于搜索与字数统计。
func plainTextFromHTML(rendered string) string {
	doc, err := xhtml.Parse(strings.NewReader(rendered))
	if err != nil {
		return strings.TrimSpace(rendered)
	}

	var b strings.Builder
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		switch n.Type {
		case xhtml.TextNode:
			b.WriteString(n.Data)
			b.WriteByte(' ')
		case xhtml.ElementNode:
			switch n.Data {
			case "pre", "code", "script", "style":
				// 代码/脚本不进入摘要与搜索文本。
				return
			case "span":
				// 数学公式的 TeX 原文不进入摘要与搜索文本。
				for _, attr := range n.Attr {
					if attr.Key == "class" && strings.Contains(attr.Val, "math") {
						return
					}
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	joined := strings.Join(strings.Fields(b.String()), " ")
	return punctuationSpacePattern.ReplaceAllString(joined, "$1")
}

var punctuationSpacePattern = regexp.MustCompile(`\s+([。，！？：；,.;:!?])`)

// countWords 把每个汉字计为一个词，把连续的拉丁字母/数字计为一个词。
func countWords(text string) int {
	count := 0
	inWord := false
	for _, r := range text {
		switch {
		case unicode.Is(unicode.Han, r):
			count++
			inWord = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if !inWord {
				count++
				inWord = true
			}
		default:
			inWord = false
		}
	}
	return count
}

func excerpt(text string, limit int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "…"
}
