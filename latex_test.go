package latex_test

import (
	"bytes"
	_ "embed"
	"io"
	"os"
	"testing"

	latex "github.com/soypat/goldmark-latex"
	"github.com/yuin/goldmark"
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
