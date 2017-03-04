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
	// for Grep* methods
	nullBytes = []byte{}
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

// MapString - set filter function for process each line as string
func (lr *Reader) MapString(filterFn func(line string) string) *Reader {
	lr.filterFuncs = append(lr.filterFuncs, func(line []byte) ([]byte, error) {
		return []byte(filterFn(string(line))), nil
	})
	return lr
}

// MapStringErr - set filter function for process each line as string, returns error if needed (io.EOF for example)
func (lr *Reader) MapStringErr(filterFn func(line string) (string, error)) *Reader {
	lr.filterFuncs = append(lr.filterFuncs, func(line []byte) ([]byte, error) {
		newString, err := filterFn(string(line))
		return []byte(newString), err
	})
	return lr
}

// Grep - grep lines by func
func (lr *Reader) Grep(filterFn func(line []byte) bool) *Reader {
	lr.filterFuncs = append(lr.filterFuncs, func(line []byte) ([]byte, error) {
		if filterFn(line) {
			return line, nil
		}
		return nullBytes, nil
	})
	return lr
}

// GrepString - grep lines as string by func
func (lr *Reader) GrepString(filterFn func(line string) bool) *Reader {
	return lr.Grep(func(line []byte) bool {
		return filterFn(string(line))
	})
}

// GrepByRegexp - grep lines by regexp
func (lr *Reader) GrepByRegexp(re *regexp.Regexp) *Reader {
	return lr.Grep(func(line []byte) bool {
		return re.Match(line)
	})
}
