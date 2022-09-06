//go:build !tinygo

package latex_test

import (
	_ "embed"
)

// var hasLatex = exec.Command("latexmk", "-h").Run() == nil

// //go:embed list.md
// var list string

// func TestLists(t *testing.T) {
// 	// const list = listTest //"1. This is\n\tthe first paragraph  \nThis is new paragraph\n2. Another item"
// 	output := render(t, strings.NewReader(list))
// 	writeTexAndPDF(t, output, "testresult/list.tex")
// 	t.Log(output.String())
// }

// func writeTexAndPDF(t *testing.T, r io.Reader, texFilename string) {
// 	pdfFilename := strings.TrimSuffix(texFilename, filepath.Ext(texFilename)) + ".pdf"
// 	os.Remove(pdfFilename)
// 	txfp, _ := os.Create(texFilename)
// 	io.Copy(txfp, r)
// 	txfp.Close()
// 	cmd := exec.Command("latexmk", "-pdf", texFilename)
// 	cmd.Stdin = strings.NewReader("QQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQ") // Force batch mode
// 	cmdOut, err := cmd.CombinedOutput()
// 	if err != nil {
// 		t.Error(string(cmdOut))
// 		t.Fatal("running latexmk failed with error:", err)
// 	}
// }
