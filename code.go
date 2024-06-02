// Code syntax highlighting support.
// This file was mostly copied from the gomarkdown docs.
package main

import (
	"fmt"
	"io"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/gomarkdown/markdown/ast"
)

var (
	htmlFormatter  *html.Formatter
	highlightStyle *chroma.Style
)

func init() {
	htmlFormatter = html.New(html.WithClasses(false), html.TabWidth(2))
	if htmlFormatter == nil {
		panic("couldn't create html formatter")
	}
	styleName := "xcode"
	highlightStyle = styles.Get(styleName)
	if highlightStyle == nil {
		panic(fmt.Sprintf("didn't find style '%s'", styleName))
	}
}

func htmlHighlight(w io.Writer, source, lang string) error {
	l := lexers.Get(lang)
	if l == nil {
		return fmt.Errorf("unknown syntax highlight language %q", lang)
	}
	l = chroma.Coalesce(l)

	it, err := l.Tokenise(nil, source)
	if err != nil {
		return err
	}
	return htmlFormatter.Format(w, highlightStyle, it)
}

func syntaxHighlightRenderHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	if code, ok := node.(*ast.CodeBlock); ok {
		if lang := string(code.Info); lang != "" {
			if err := htmlHighlight(w, string(code.Literal), lang); err != nil {
				panic(err)
			}
			return ast.GoToNext, true
		}
	}
	return ast.GoToNext, false
}
