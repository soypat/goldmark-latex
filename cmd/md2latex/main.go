package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	latex "github.com/soypat/goldmark-latex"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

var (
	usehtml        bool
	print          bool
	unhead         bool
	outputFilename string
)

func main() {
	flag.BoolVar(&usehtml, "html", false, "Output html")
	flag.BoolVar(&print, "p", false, "Output to stdout")
	flag.BoolVar(&unhead, "unhead", false, "No heading numbering")
	flag.StringVar(&outputFilename, "o", "", "Output filename. By default just adds .tex to input filename.")
	flag.Parse()
	args := flag.Args()
	err := run(args)
	if err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("missing filename argument")
	}
	filename := args[0]
	fp, err := os.Open(filename)
	if err != nil {
		return err
	}
	input, err := io.ReadAll(fp)
	fp.Close()
	if err != nil {
		return err
	}
	output, err := renderGoldmark(input)
	if err != nil {
		return err
	}
	if print {
		fmt.Println(string(output))
		return nil
	}
	// Generate output file.
	ext := filepath.Ext(filename)
	if ext == "" && outputFilename == "" {
		outputFilename = filename + ".tex"
	} else if outputFilename == "" {
		outputFilename = strings.TrimSuffix(filename, ext) + ".tex"
	}
	outfp, err := os.Create(outputFilename)
	if err != nil {
		return err
	}
	defer outfp.Close()
	_, err = io.Copy(outfp, bytes.NewBuffer(output))
	return err
}

func renderGoldmark(input []byte) ([]byte, error) {
	var rd renderer.Renderer
	if usehtml {
		rd = goldmark.DefaultRenderer()
	} else {
		rd = renderer.NewRenderer(renderer.WithNodeRenderers(util.Prioritized(latex.NewRenderer(latex.Config{
			NoHeadingNumbering: unhead,
		}), 1000)))
	}
	md := goldmark.New(goldmark.WithRenderer(rd))
	var b bytes.Buffer
	err := md.Convert(input, &b)
	return b.Bytes(), err
}
