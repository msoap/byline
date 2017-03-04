/*
Package byline implements Reader for process line-by-line another Reader
*/
package byline

import (
	"bufio"
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
type Reader struct {
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
func NewReader(reader io.Reader) *Reader {
	return &Reader{
		bufReader: bufio.NewReader(reader),
		awkVars: AWKVars{
			RS: defaultRS,
			FS: defaultFS,
		},
	}
}

// Read - implement io.Reader interface
func (lr *Reader) Read(p []byte) (n int, err error) {
	lineBytes, bufErr := lr.bufReader.ReadBytes(lr.awkVars.RS)
	lr.awkVars.NR++

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

// Map - set filter function for process each line
func (lr *Reader) Map(filterFn func(line []byte) []byte) *Reader {
	lr.filterFuncs = append(lr.filterFuncs, func(line []byte) ([]byte, error) {
		return filterFn(line), nil
	})
	return lr
}

// MapErr - set filter function for process each line, returns error if needed (io.EOF for example)
func (lr *Reader) MapErr(filterFn func(line []byte) ([]byte, error)) *Reader {
	lr.filterFuncs = append(lr.filterFuncs, filterFn)
	return lr
}
