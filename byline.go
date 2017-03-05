package byline

import (
	"bufio"
	"errors"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
)

var (
	// ErrOmitLine - error for Grep/AWK mode, for omit current line
	ErrOmitLine = errors.New("ErrOmitLine")

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

	for _, filterFunc := range lr.filterFuncs {
		var filterErr error
		lineBytes, filterErr = filterFunc(lineBytes)
		if filterErr != nil {
			switch {
			case filterErr == ErrOmitLine:
				lineBytes = nullBytes
			case filterErr != nil:
				bufErr = filterErr
			}
			break
		}
	}

	copy(p, lineBytes)
	return len(lineBytes), bufErr
}

// Map - set filter function for process each line
func (lr *Reader) Map(filterFn func([]byte) []byte) *Reader {
	return lr.MapErr(func(line []byte) ([]byte, error) {
		return filterFn(line), nil
	})
}

// MapErr - set filter function for process each line, returns error if needed (io.EOF for example)
func (lr *Reader) MapErr(filterFn func([]byte) ([]byte, error)) *Reader {
	lr.filterFuncs = append(lr.filterFuncs, filterFn)
	return lr
}

// MapString - set filter function for process each line as string
func (lr *Reader) MapString(filterFn func(string) string) *Reader {
	return lr.MapErr(func(line []byte) ([]byte, error) {
		return []byte(filterFn(string(line))), nil
	})
}

// MapStringErr - set filter function for process each line as string, returns error if needed (io.EOF for example)
func (lr *Reader) MapStringErr(filterFn func(string) (string, error)) *Reader {
	return lr.MapErr(func(line []byte) ([]byte, error) {
		newString, err := filterFn(string(line))
		return []byte(newString), err
	})
}

// Grep - grep lines by func
func (lr *Reader) Grep(filterFn func([]byte) bool) *Reader {
	return lr.MapErr(func(line []byte) ([]byte, error) {
		if filterFn(line) {
			return line, nil
		}

		return nullBytes, ErrOmitLine
	})
}

// GrepString - grep lines as string by func
func (lr *Reader) GrepString(filterFn func(string) bool) *Reader {
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

// SetRS - set lines (records) separator
func (lr *Reader) SetRS(rs byte) *Reader {
	lr.awkVars.RS = rs
	return lr
}

// SetFS - set field separator for AWK mode
func (lr *Reader) SetFS(fs *regexp.Regexp) *Reader {
	lr.awkVars.FS = fs
	return lr
}

// AWKMode - process lines with AWK like mode
func (lr *Reader) AWKMode(filterFn func(line string, fields []string, vars AWKVars) (string, error)) *Reader {
	return lr.MapStringErr(func(line string) (string, error) {
		addRS := ""
		if strings.HasSuffix(line, string(lr.awkVars.RS)) {
			addRS = string(lr.awkVars.RS)
			line = strings.TrimSuffix(line, string(lr.awkVars.RS))
		}

		fields := lr.awkVars.FS.Split(line, -1)
		lr.awkVars.NF = len(fields)
		result, err := filterFn(line, fields, lr.awkVars)
		if err != nil {
			return "", err
		}

		if !strings.HasSuffix(result, string(lr.awkVars.RS)) && addRS != "" {
			result += addRS
		}
		return result, nil
	})
}

// Discard - read all content from Reader for side effect from filter functions
func (lr *Reader) Discard() error {
	_, err := io.Copy(ioutil.Discard, lr)
	return err
}

// ReadAllSlice - read all content from Reader to []byte slice by lines
func (lr *Reader) ReadAllSlice() ([][]byte, error) {
	result := [][]byte{}
	err := lr.Map(func(line []byte) []byte {
		result = append(result, line)
		return nullBytes
	}).Discard()

	return result, err
}

// ReadAll - read all content from Reader to slice of bytes
func (lr *Reader) ReadAll() ([]byte, error) {
	return ioutil.ReadAll(lr)
}

// ReadAllSliceString - read all content from Reader to string slice by lines
func (lr *Reader) ReadAllSliceString() ([]string, error) {
	result := []string{}
	err := lr.MapString(func(line string) string {
		result = append(result, line)
		return ""
	}).Discard()

	return result, err
}

// ReadAllString - read all content from Reader to one string
func (lr *Reader) ReadAllString() (string, error) {
	result, err := ioutil.ReadAll(lr)
	return string(result), err
}
