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

// renderCite returns the LaTeX output for input using cfg, with the citation
// inline parser registered. Once the parser is implemented, citations in input
// will be converted; until then [@key] renders as plain text and these tests fail.
func renderCite(t *testing.T, cfg latex.Config, input string) string {
	t.Helper()
	lr := latex.NewRenderer(cfg)
	rd := renderer.NewRenderer(renderer.WithNodeRenderers(util.Prioritized(lr, 1000)))
	md := goldmark.New(
		goldmark.WithRenderer(rd),
		goldmark.WithParserOptions(
			parser.WithInlineParsers(util.Prioritized(latex.CitationParser, 150)),
		),
	)
	var out bytes.Buffer
	if err := md.Convert([]byte(input), &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestCitationSingle(t *testing.T) {
	got := strings.TrimSpace(renderCite(t, latex.Config{NoPreamble: true}, "See [@darwin1859]."))
	want := `See \cite{darwin1859}.\\`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestCitationMultipleKeys(t *testing.T) {
	got := strings.TrimSpace(renderCite(t, latex.Config{NoPreamble: true}, "[@key1; @key2]"))
	want := `\cite{key1,key2}\\`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestCitationCustomCmd(t *testing.T) {
	got := strings.TrimSpace(renderCite(t, latex.Config{NoPreamble: true, CiteCmd: "citep"}, "[@key]"))
	want := `\citep{key}\\`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestBibliography(t *testing.T) {
	got := renderCite(t, latex.Config{BibFile: "refs"}, "text")
	if !strings.Contains(got, "\\bibliographystyle{plain}\n\\bibliography{refs}") {
		t.Errorf("bibliography block missing or malformed:\n%s", got)
	}
}

func TestBibliographyCustomStyle(t *testing.T) {
	got := renderCite(t, latex.Config{BibFile: "refs", BibStyle: "ieeetr"}, "text")
	if !strings.Contains(got, "\\bibliographystyle{ieeetr}\n\\bibliography{refs}") {
		t.Errorf("bibliography block missing or malformed:\n%s", got)
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
