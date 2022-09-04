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
	lr := latex.NewRenderer(latex.Config{
		Unsafe: true,
	})
	r := renderer.NewRenderer(renderer.WithNodeRenderers(util.Prioritized(lr, 1000)))
	md := goldmark.New(goldmark.WithRenderer(r))
	var output bytes.Buffer
	err := md.Convert(data, &output)
	if err != nil {
		t.Error(err)
	}
	os.Mkdir("testresult", 0777)
	fp, err := os.Create("testresult/result_test.tex")
	if err != nil {
		t.Fatal(err)
	}
	defer fp.Close()
	io.Copy(fp, &output)
}
