/*
Package byline implements Reader for process line-by-line another Reader
*/
package byline

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
)

var (
	// default field separator
	defaultFS = regexp.MustCompile(`\s+`)
	// default line separator
	defaultRS byte = '\n'
)

// Reader - line by line Reader
type Reader interface {
	io.Reader

	MapErr(func(line []byte) ([]byte, error)) Reader
}

type linesReader struct {
	bufReader   *bufio.Reader
	filterFuncs []func(line []byte) ([]byte, error)
	awkVars     AWKVars
}

// AWKVars - settings for AWK mode, see man awk
type AWKVars struct {
	NR int            // number of current line (begin from 1)
	NF int            // fields count in curent line
	RS byte           // record separator, default is '\n'
	FS *regexp.Regexp // field separator, default is `\s+`
}

// NewReader - get new line by line Reader
func NewReader(reader io.Reader) Reader {
	return &linesReader{
		bufReader: bufio.NewReader(reader),
		awkVars: AWKVars{
			RS: defaultRS,
			FS: defaultFS,
		},
	}
}

// Read - implement io.Reader interface
func (lr *linesReader) Read(p []byte) (n int, err error) {
	lineBytes, bufErr := lr.bufReader.ReadBytes(lr.awkVars.RS)
	lr.awkVars.NR++
	fmt.Printf("NR: %d, %s", lr.awkVars.NR, string(lineBytes))

	var filterErr error
	for _, filterFunc := range lr.filterFuncs {
		lineBytes, filterErr = filterFunc(lineBytes)
		if bufErr != io.EOF && filterErr != nil {
			bufErr = filterErr
		}
	}

	copy(p, lineBytes)
	return len(lineBytes), bufErr
}

// MapErr - set filter function for process one line
func (lr *linesReader) MapErr(filterFn func(line []byte) ([]byte, error)) Reader {
	lr.filterFuncs = append(lr.filterFuncs, filterFn)
	return lr
}
