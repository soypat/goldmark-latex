package latex

import (
	"bytes"
	_ "embed"
	"io"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

type Config struct {
	// Increase heading levels: if the offset is 1, \section (1) becomes \subsection (2) etc.
	// Negative offset is also valid.
	// Resulting levels are clipped between 1 and 6.
	HeadingLevelOffset int
	// Removes section numbering.
	NoHeadingNumbering bool
	// Replace the default preamble by setting this to a non-nil byte slice.
	// Should NOT end with \begin{document}, this is added automatically.
	Preamble []byte
	// If set renderer will render possibly unsafe elements, such as links.
	Unsafe bool
	// Declares all used unicode characters in the preamble
	// and replaces them with the result of this function.
	DeclareUnicode func(rune) (raw string, isReplaced bool)
}

//go:embed header.tex
var defaultHeader []byte

var _ renderer.NodeRenderer = &Renderer{}

type Renderer struct {
	Config Config
}

// An Option interface sets options for HTML based renderers.
type Option interface {
	SetLatexOption(*Config)
}

// NewRenderer returns a new Renderer with given options.
func NewRenderer(opts ...Option) renderer.NodeRenderer {
	r := &Renderer{
		Config: Config{},
	}
	for _, opt := range opts {
		opt.SetLatexOption(&r.Config)
	}
	return r
}

func (r *Renderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// blocks

	reg.Register(ast.KindDocument, r.renderDocument)
	reg.Register(ast.KindHeading, r.renderHeading)
	reg.Register(ast.KindBlockquote, r.renderBlockquote)
	reg.Register(ast.KindCodeBlock, r.renderCodeBlock)
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
	reg.Register(ast.KindHTMLBlock, r.renderHTMLBlock)
	reg.Register(ast.KindList, r.renderList)
	reg.Register(ast.KindListItem, r.renderListItem)
	reg.Register(ast.KindParagraph, r.renderParagraph)
	reg.Register(ast.KindTextBlock, r.renderTextBlock)
	reg.Register(ast.KindThematicBreak, r.renderThematicBreak)

	// inlines

	reg.Register(ast.KindAutoLink, r.renderAutoLink)
	reg.Register(ast.KindCodeSpan, r.renderCodeSpan)
	reg.Register(ast.KindEmphasis, r.renderEmphasis)
	reg.Register(ast.KindImage, r.renderImage)
	reg.Register(ast.KindLink, r.renderLink)
	reg.Register(ast.KindRawHTML, r.renderRawHTML)
	reg.Register(ast.KindText, r.renderText)
	reg.Register(ast.KindString, r.renderString)
}

func (r *Renderer) renderDocument(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		if r.Config.Preamble == nil {
			w.Write(defaultHeader)
		} else {
			w.Write(r.Config.Preamble)
		}
		if r.Config.DeclareUnicode != nil {
			_ = w.WriteByte('\n')
			const unicodeDecl = "\\DeclareUnicodeCharacter{"
			const zeropad = "00"
			declared := make(map[rune]struct{})
			n := len(source)
			i := 0
			for i < n {
				char, lchar := utf8.DecodeRune(source[i:])
				i += lchar
				if lchar == 1 {
					continue // ASCII character.
				}
				if _, ok := declared[char]; ok {
					continue
				}
				declared[char] = struct{}{}
				replace, ok := r.Config.DeclareUnicode(char)
				if !ok {
					continue
				}
				_, _ = w.WriteString(unicodeDecl)
				num := strconv.FormatUint(uint64(char), 16)
				_, _ = w.WriteString(zeropad[:2-(len(num)-2)])
				_, _ = w.WriteString(num)
				_, _ = w.WriteString("}{")
				_, _ = w.WriteString(replace)
				_, _ = w.WriteString("}\n")
			}
		}
		w.WriteString("\n\\begin{document}\n")
	} else {
		w.WriteString("\n\\end{document}\n")
		return ast.WalkStop, nil
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Heading)
	if entering {
		headingLevel := max(0, min(6, r.Config.HeadingLevelOffset+n.Level-1))
		start := headingTable[headingLevel][bool2int(r.Config.NoHeadingNumbering)]
		w.WriteByte('\n')
		w.Write(start)
		if headingLevel >= 5 {
			w.WriteByte('\n')
		}
	} else {
		w.WriteByte('}')
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderBlockquote(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.Write(blockQuoteStart)
	} else {
		_, _ = w.Write(blockQuoteEnd)
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderCodeBlock(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.Write(blockCodeStart)
		_ = w.WriteByte('\n')
		r.writeLines(w, source, n)
	} else {
		_, _ = w.Write(blockCodeEnd)
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.FencedCodeBlock)
	if entering {
		_, _ = w.Write(blockCodeStart)
		language := n.Language(source)
		language = language[:min(10, len(language))]
		_, supported := supportedLang[string(language)]
		if language != nil && supported {
			_, _ = w.WriteString("[language=")
			escapeLaTeX(w, language)
			_ = w.WriteByte(']')
		}
		_ = w.WriteByte('\n')
		r.writeLines(w, source, n)
	} else {
		_, _ = w.Write(blockCodeEnd)
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderHTMLBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	w.WriteString("\n% HTML block rendering unsupported, skipped\n")
	return ast.WalkSkipChildren, nil
}

func (r *Renderer) renderList(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.List)
	tag := "itemize"
	if n.IsOrdered() {
		tag = "enumerate"
	}
	if entering {
		_, _ = w.WriteString("\\begin{")
		_, _ = w.WriteString(tag)
		_, _ = w.WriteString("}\n")
	} else {
		_, _ = w.WriteString("\\end{")
		_, _ = w.WriteString(tag)
		_, _ = w.WriteString("}\n")
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderListItem(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.Write(itemCommand)
		fc := n.FirstChild()
		if fc != nil {
			if _, ok := fc.(*ast.TextBlock); !ok {
				// _ = w.WriteByte('\n')
			}
		}
	} else {
		_ = w.WriteByte('\n')
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderParagraph(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		parent := n.Parent()
		pkind := parent.Kind()
		if pkind != ast.KindList && pkind != ast.KindListItem {
			_, _ = w.Write(hardBreak)
		}
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderTextBlock(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		if n.NextSibling() != nil && n.FirstChild() != nil {
			_ = w.WriteByte('\n')
		}
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderThematicBreak(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.Write(hruleCommand)
		_ = w.WriteByte('\n')
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.AutoLink)
	if !entering {
		return ast.WalkContinue, nil
	}
	url := n.URL(source)
	label := n.Label(source)
	_, _ = w.WriteString(`\href{`)
	if n.AutoLinkType == ast.AutoLinkEmail && haslowerprefix(url, mailToPrefix) {
		_, _ = w.WriteString("mailto:")
	}
	escLink(w, url)
	_, _ = w.WriteString(`}{`)
	escapeLaTeX(w, label)
	_ = w.WriteByte('}')
	return ast.WalkContinue, nil
}

// haslowerprefix is an allocation free implementation of
//
//	bytes.HasPrefix(bytes.ToLower(a), bytes.ToLower(b))
func haslowerprefix(a, b []byte) bool {
	n := min(len(a), len(b))
	i := 0
	for i < n {
		ra, la := utf8.DecodeRune(a[i:])
		rb, lb := utf8.DecodeRune(b[i:])
		if la != lb || unicode.ToLower(ra) != unicode.ToLower(rb) {
			return false
		}
		i += la
	}
	return true
}

func (r *Renderer) renderCodeSpan(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		_ = w.WriteByte('}')
		return ast.WalkContinue, nil
	}

	// Render all children within code span. Should all be Text kind.
	_, _ = w.Write(codeSpanStart)
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		segment := c.(*ast.Text).Segment
		value := segment.Value(source)
		if bytes.HasSuffix(value, []byte("\n")) {
			escapeLaTeX(w, value[:len(value)-1])
			_ = w.WriteByte(' ')
		} else {
			escapeLaTeX(w, value)
		}
	}
	return ast.WalkSkipChildren, nil // Skip all of them after rendering.
}

func (r *Renderer) renderEmphasis(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		const (
			emph  = "\\textit{"
			bold  = "\\textbf{"
			emph3 = "\\emph{"
		)
		n := node.(*ast.Emphasis)
		tag := emph
		if n.Level == 2 {
			tag = bold
		} else if n.Level == 3 {
			tag = emph3
		}
		w.WriteString(tag)
	} else {
		w.WriteByte('}')
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if entering {
		_, _ = w.WriteString(`\href{`)
		if r.Config.Unsafe || !html.IsDangerousURL(n.Destination) {
			escapeLaTeX(w, n.Destination)
			// _, _ = w.Write(util.EscapeHTML(util.URLEscape(n.Destination, true)))
		}
		_, _ = w.WriteString("}{")
	} else {
		_ = w.WriteByte('}')
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	// No image rendering implemented yet.
	w.WriteString("\n% image rendering unsupported as of yet\n")
	return ast.WalkSkipChildren, nil
}

func (r *Renderer) renderRawHTML(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	// No rawHTML rendering supported
	w.WriteString("\n% raw HTML rendering unsupported\n")
	return ast.WalkSkipChildren, nil
}

func (r *Renderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Text)
	segment := n.Segment.Value(source)
	if n.IsRaw() {
		w.Write(segment)
		// r.Writer.RawWrite(w, segment.Value(source))
	} else {
		escapeLaTeX(w, segment)
		if n.HardLineBreak() {
			_, _ = w.Write(hardBreak)
		} else if n.SoftLineBreak() {
			_ = w.WriteByte('\n')
		}
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderString(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.String)
	if n.IsCode() || n.IsRaw() {
		_, _ = w.Write(n.Value)
	} else {
		escapeLaTeX(w, n.Value)
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) writeLines(w util.BufWriter, source []byte, n ast.Node) {
	l := n.Lines().Len()
	for i := 0; i < l; i++ {
		line := n.Lines().At(i)
		escapeLaTeX(w, line.Value(source))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func bool2int(b bool) int {
	if b {
		return 1
	}
	return 0
}

var (
	mailToPrefix    = []byte(":mailto")
	hardBreak       = []byte("\\\\\n\n")
	strikeStart     = []byte("\\sout{") // Using ulem package.
	hrefStart       = []byte("\\href{")
	codeSpanStart   = []byte("\\texttt{")
	blockQuoteStart = []byte("\n\\begin{lstlisting}[frame=none]\n")
	blockQuoteEnd   = []byte("\\end{lstlisting}\n")
	blockCodeStart  = []byte("\n\\begin{lstlisting}")
	blockCodeEnd    = []byte("\\end{lstlisting}\n")
	hruleCommand    = []byte("\n\\hrulefill\n")

	itemCommand  = []byte("\\item~ ")
	tableStart   = []byte("\n\\begin{table}\n")
	tableEnd     = []byte("\n\\end{table}\n")
	headingTable = [6][2][]byte{
		// {[]byte("\\part{"), []byte("\\part*{")},
		// {[]byte("\\chapter{"), []byte("\\chapter*{")},
		{[]byte("\\section{"), []byte("\\section*{")},
		{[]byte("\\subsection{"), []byte("\\subsection*{")},
		{[]byte("\\subsubsection{"), []byte("\\subsubsection*{")},
		{[]byte("\\paragraph{"), []byte("\\paragraph*{")},
		{[]byte("\\subparagraph{"), []byte("\\subparagraph*{")},
		{[]byte("\\textbf{"), []byte("\\textbf{")},
	}
)

var supportedLang = map[string]struct{}{
	"python": {},
	// "markdown": {}, // Breaks lstlisting
}

var escapeTable = [256][]byte{
	'\\': []byte("\\textbackslash~"),
	'~':  []byte("\\textasciitilde~"),
	'^':  []byte("\\textasciicircum~"),
	'&':  []byte("\\&"),
	'%':  []byte("\\%"),
	'$':  []byte("\\$"),
	'#':  []byte("\\#"),
	'_':  []byte("\\_"),
	'{':  []byte("\\{"),
	'}':  []byte("\\}"),
}

func escapeLaTeX(w io.Writer, s []byte) {
	var start, end int
	for end < len(s) {
		escSeq := escapeTable[s[end]]
		if escSeq != nil {
			w.Write(s[start:end])
			w.Write(escSeq)
			start = end + 1
		}
		end++
	}
	if start < len(s) && end <= len(s) {
		w.Write(s[start:end])
	}
}

func escLink(w io.Writer, text []byte) {
	escapeLaTeX(w, text)
}
