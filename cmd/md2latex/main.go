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
	"time"

	latex "github.com/soypat/goldmark-latex"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

var (
	usehtml          bool
	verbose          bool
	print            bool
	unhead           bool
	unsafe           bool
	preambleFilename string
	outputFilename   string
	headingOffset    int
)

func main() {
	flag.BoolVar(&verbose, "v", false, "Verbose output.")
	flag.BoolVar(&usehtml, "html", false, "Output html")
	flag.BoolVar(&print, "p", false, "Output to stdout")
	flag.BoolVar(&unsafe, "unsafe", false, "Render unsafe segments of document such as links or verbatim.")
	flag.BoolVar(&unhead, "unhead", false, "No section numbering")
	flag.StringVar(&outputFilename, "o", "", "Output filename. By default just adds .tex to input filename.")
	flag.StringVar(&preambleFilename, "preamble", "", "Preamble filename. If not set uses a default preamble.")
	flag.IntVar(&headingOffset, "headingoffset", 0, "Section heading offset. Can be negative. Results are clipped between 1 and 6.")
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
	verb("beginning verbose run")
	filename := args[0]
	input, err := readFile(filename)
	if err != nil {
		return err
	}
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
	var preamble []byte
	if preambleFilename != "" {
		b, err := readFile(preambleFilename)
		if err != nil {
			return nil, err
		}
		verb("replacing default preamble with", preambleFilename, "of length", len(b))
		preamble = b
	}
	var rd renderer.Renderer
	if usehtml {
		verb("using html renderer")
		rd = goldmark.DefaultRenderer()
	} else {
		rd = renderer.NewRenderer(renderer.WithNodeRenderers(util.Prioritized(latex.NewRenderer(latex.Config{
			NoHeadingNumbering: unhead,
			Unsafe:             unsafe,
			Preamble:           preamble,
			HeadingLevelOffset: headingOffset,
		}), 1000)))
	}
	md := goldmark.New(goldmark.WithRenderer(rd))
	var b bytes.Buffer
	verb("start rendering using goldmark")
	start := time.Now()
	err := md.Convert(input, &b)
	verb("finished rendering in", time.Since(start))
	return b.Bytes(), err
}

// Opens, reads and closes file and returns contents.
func readFile(filename string) ([]byte, error) {
	verb("opening ", filename)
	fp, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	input, err := io.ReadAll(fp)
	if err != nil {
		return nil, err
	}
	return input, fp.Close()
}

func verb(a ...any) {
	if verbose {
		log.Println(a...)
	}
}
