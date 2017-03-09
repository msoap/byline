/*
Package byline implements Reader interface for processing io.Reader line-by-line.
You can add UNIX text processing principles to its Reader (like with awk, grep, sed ...).

Install

 go get -u github.com/msoap/byline

Usage

 import "github.com/msoap/byline"

 // Create new line-by-line Reader from io.Reader:
 lr := byline.NewReader(reader)

 // Add to the Reader stack of a filter functions:
 lr.MapString(func(line string) string {return "prefix_" + line}).GrepByRegexp(regexp.MustCompile("only this"))

 // Read all content
 result, err := lr.ReadAll()

 // Use everywhere instead of io.Reader
 _, err := io.Copy(os.Stdout, lr)

 // Or in one place
 result, err := byline.NewReader(reader).MapString(func(line string) string {return "prefix_" + line}).ReadAll()

*/
package byline
