# byline Reader [![GoDoc](https://godoc.org/github.com/msoap/byline?status.svg)](https://godoc.org/github.com/msoap/byline) [![Build Status](https://travis-ci.org/msoap/byline.svg?branch=master)](https://travis-ci.org/msoap/byline) [![Coverage Status](https://coveralls.io/repos/github/msoap/byline/badge.svg?branch=master)](https://coveralls.io/github/msoap/byline?branch=master) [![Sourcegraph](https://sourcegraph.com/github.com/msoap/byline/-/badge.svg)](https://sourcegraph.com/github.com/msoap/byline?badge) [![Report Card](https://goreportcard.com/badge/github.com/msoap/byline)](https://goreportcard.com/report/github.com/msoap/byline)

Go library for convert io.Reader to line-by-line Reader. Now you can add UNIX text processing principles to its Reader (like with awk, grep, sed ...).

## Install

`go get -u github.com/msoap/byline`

## Usage

```Go
import "github.com/msoap/byline"

// Create new line-by-line Reader from io.Reader:
lr := byline.NewReader(reader)

// Add to the Reader stack of a filter functions:
lr.MapString(func(line string) string {return "prefix_" + line}).GrepByRegexp(regexp.MustCompile("only this"))

// Read all content
result, err := lr.ReadAll()

// Or in one place
result, err := byline.NewReader(reader).MapString(func(line string) string {return "prefix_" + line}).ReadAll()
```

## Filter functions

  * `Map(filterFn func([]byte) []byte)` - processing of each line as `[]byte`
  * `MapErr(filterFn func([]byte) ([]byte, error))` - processing of each line as `[]byte`, and you can return error, `io.EOF` or custom error
  * `MapString(filterFn func(string) string)` - processing of each line as `string`
  * `MapStringErr(filterFn func(string) (string, error))` - processing of each line as `string`, and you can return error
  * `Grep(filterFn func([]byte) bool)` - filtering lines by function
  * `GrepString(filterFn func(string) bool)` - filtering lines as `string` by function
  * `GrepByRegexp(re *regexp.Regexp)` - filtering lines by regexp
  * `AWKMode(filterFn func(line string, fields []string, vars AWKVars) (string, error))` - processing of each line in AWK mode.
    In addition to current line, `filterFn` gets slice with fields splitted by separator (default is `/\s+/`) and vars releated to awk (`NR`, `NF`, `RS`, `FS`)

`Map*Err`, `AWKMode` methods can return `byline.ErrOmitLine` - error for discard processing of current line.

## Helper methods

  * `SetRS(rs byte)` - set line (record) separator, default is newline - `\n`
  * `SetFS(fs *regexp.Regexp)` - set field separator for AWK mode, default is `\s+`
  * `Discard()` - discard all content from Reader only for side effect of filter functions
  * `ReadAll() ([]byte, error)` - return all content as slice of bytes
  * `ReadAllSlice() ([][]byte, error)` - return all content by lines as `[][]byte`
  * `ReadAllString() (string, error)` - return all content as string
  * `ReadAllSliceString() ([]string, error)` - return all content by lines as slice of strings

## Examples

Add line number to each line and add suffix at the end of line:

```Go
	reader := strings.NewReader("111\n222\n333")
    // or read file
    reader, err := os.Open("file.txt")
    // or process response from HTTP client
    reader := httpResponse.Body

	i := 0
	blr := byline.NewReader(reader).MapString(func(line string) string {
		i++
		return fmt.Sprintf("(%d) %s", i, string(line))
	}).Map(func(line []byte) []byte {
		return regexp.MustCompile(`\n?$`).ReplaceAll(line, []byte(" suf\n"))
	})

	result, err := blr.ReadAll()

```

<details><summary>Example grep Go types from source:</summary>
```Go
type StateMachine struct {
	beginRe *regexp.Regexp
	endRe   *regexp.Regexp
	inBlock bool
}

func (sm *StateMachine) SMFilter(line []byte) bool {
	switch {
	case sm.beginRe.Match(line):
		sm.inBlock = true
		return true
	case sm.inBlock && sm.endRe.Match(line):
		sm.inBlock = false
		return true
	default:
		return sm.inBlock
	}
}

func ExampleReader_Grep() {
	file, err := os.Open("byline.go")
	if err != nil {
		fmt.Println(err)
		return
	}

	// get all lines between "^type..." and "^}"
	sm := StateMachine{
		beginRe: regexp.MustCompile(`^type `),
		endRe:   regexp.MustCompile(`^}\s+$`),
	}

	blr := byline.NewReader(file).Grep(sm.SMFilter).Map(func(line []byte) []byte {
		// and remove comments
		return regexp.MustCompile(`\s+//.+`).ReplaceAll(line, []byte{})
	})

	result, err := blr.ReadAllString()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Print(result)
}
```
Output:
```
type Reader struct {
	bufReader   *bufio.Reader
	filterFuncs []func(line []byte) ([]byte, error)
	awkVars     AWKVars
}
type AWKVars struct {
	NR int
	NF int
	RS byte
	FS *regexp.Regexp
}
```
</details>

<details><summary>Example of AWK mode, summarize third column with filter (>10.0):</summary>
```Go
    // CSV with "#" instead of "\n"
	reader := strings.NewReader(`1,name one,12.3#2,second row;7.1#3,three row;15.51`)

	sum := 0.0
	err := byline.NewReader(reader).
		SetRS('#').
		SetFS(regexp.MustCompile(`,|;`)).
		AWKMode(func(line string, fields []string, vars byline.AWKVars) (string, error) {
			if vars.NF < 3 {
				return "", fmt.Errorf("csv parse failed for %q", line)
			}

			if price, err := strconv.ParseFloat(fields[2], 10); err != nil {
				return "", err
			} else if price < 10 {
				return "", byline.ErrOmitLine
			} else {
				sum += price
				return "", nil
			}
		}).Discard()

	fmt.Println("Price sum:", sum, err)

```
Output:
```
Price sum: 27.81 nil
```
</details>

## See also

  * [io](https://golang.org/pkg/io/), [ioutil](https://golang.org/pkg/io/ioutil/), [bufio](https://golang.org/pkg/bufio/) - Go packages for work with Readers
  * [AWK](https://en.wikipedia.org/wiki/AWK) - programming language and great UNIX tool
