# goldmark-latex
A LaTeX renderer for [Goldmark](https://github.com/yuin/goldmark). Produce `.tex` files from markdown.

This renderer seeks to be as extensible as Goldmark itself. Please file an issue if it does not meet your requirements.

## Results
So far this implementation renders the CommonMark specification with the exception of embedded HTML. It does have some bugs related to undefined ASCII sequences. Any help is appreciated.

![result](https://user-images.githubusercontent.com/26156425/188299284-8dd2fca1-dc50-4574-8128-c78017b42e73.png)

## md2latex program
This command converts a single markdown file to latex and writes to contents to a new .text file or to stdout.
