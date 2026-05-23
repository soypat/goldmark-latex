package latex_test

import (
	"bytes"
	_ "embed"
	"io"
	"os"
	"strings"
	"testing"

	latex "github.com/soypat/goldmark-latex"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

var _ renderer.NodeRenderer = &latex.Renderer{} // Compile time check of interface implementation.

//go:embed _data.md
var data []byte

func TestRenderer(t *testing.T) {
	os.Mkdir("testresult", 0777)
	fp, err := os.Create("testresult/result_test.tex")
	if err != nil {
		t.Fatal(err)
	}
	output := render(t, bytes.NewBuffer(data))
	defer fp.Close()
	io.Copy(fp, output)
}

func TestTableRendering(t *testing.T) {
	const input = "| Left | Center | Right |\n| :--- | :----: | ----: |\n| a    | b      | c     |\n"
	const want = `\begin{tabular}{lcr}
\hline
Left & Center & Right \\
\hline
a & b & c \\
\hline
\end{tabular}`

	lr := latex.NewRenderer(latex.Config{NoPreamble: true})
	rd := renderer.NewRenderer(renderer.WithNodeRenderers(util.Prioritized(lr, 1000)))
	md := goldmark.New(
		goldmark.WithRenderer(rd),
		goldmark.WithParserOptions(
			parser.WithParagraphTransformers(util.Prioritized(extension.NewTableParagraphTransformer(), 200)),
			parser.WithASTTransformers(util.Prioritized(extension.NewTableASTTransformer(), 0)),
		),
	)
	var out bytes.Buffer
	if err := md.Convert([]byte(input), &out); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(out.String())
	if got != want {
		t.Errorf("table output mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestTableCaptionRendering(t *testing.T) {
	const input = "| A | B |\n| - | - |\n| 1 | 2 |\n\n: My caption\n"
	const want = `\begin{table}[h!]
\centering
\begin{tabular}{ll}
\hline
A & B \\
\hline
1 & 2 \\
\hline
\end{tabular}
\caption{My caption}
\end{table}`

	lr := latex.NewRenderer(latex.Config{NoPreamble: true, EnableTableCaptions: true})
	rd := renderer.NewRenderer(renderer.WithNodeRenderers(util.Prioritized(lr, 1000)))
	md := goldmark.New(
		goldmark.WithRenderer(rd),
		goldmark.WithParserOptions(
			parser.WithParagraphTransformers(util.Prioritized(extension.NewTableParagraphTransformer(), 200)),
			parser.WithASTTransformers(util.Prioritized(extension.NewTableASTTransformer(), 0)),
			parser.WithASTTransformers(util.Prioritized(latex.TableCaptionTransformer, -1)),
		),
	)
	var out bytes.Buffer
	if err := md.Convert([]byte(input), &out); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(out.String())
	if got != want {
		t.Errorf("caption output mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func render(t *testing.T, markdown io.Reader) *bytes.Buffer {
	r := renderer.NewRenderer(renderer.WithNodeRenderers(util.Prioritized(latex.NewRenderer(latex.Config{
		NoHeadingNumbering: true,                                                                     // No heading numbers
		Preamble:           append(latex.DefaultPreamble(), []byte("\n\\usepackage{MnSymbol}\n")...), // add star symbols to preamble.
		DeclareUnicode: func(r rune) (raw string, isReplaced bool) {
			switch r {
			case '★':
				return `$\filledstar$`, true
			case '☆':
				return `$\smallstar$`, true
			}
			return "", false
		},
	}), 1000)))
	md := goldmark.New(goldmark.WithRenderer(r))
	var output, input bytes.Buffer
	_, err := io.Copy(&input, markdown)
	if err != nil {
		t.Error(err)
	}
	err = md.Convert(input.Bytes(), &output)
	if err != nil {
		t.Error(err)
	}
	return &output
}
