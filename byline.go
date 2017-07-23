package byline

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"regexp"
)

var (
	// ErrOmitLine - error for Map*Err/AWKMode, for omitting current line
	ErrOmitLine = errors.New("ErrOmitLine")

	// default field separator
	defaultFS = regexp.MustCompile(`\s+`)
	// default line separator
	defaultRS byte = '\n'
	// for Grep* methods
	nullBytes = []byte{}
	// bytes.Buffer growth to this limit
	bufferSizeLimit = 512
)

// Reader - line by line Reader
type Reader struct {
	scanner     *bufio.Scanner
	buffer      bytes.Buffer
	existsData  bool
	filterFuncs []func(line []byte) ([]byte, error)
	awkVars     AWKVars
}

// AWKVars - settings for AWK mode, see man awk
type AWKVars struct {
	NR int            // number of the current line (begin from 1)
	NF int            // number of fields in the current line
	RS byte           // record separator, default is '\n'
	FS *regexp.Regexp // field separator, default is `\s+`
}

// NewReader - get new line by line Reader
func NewReader(reader io.Reader) *Reader {
	lr := &Reader{
		scanner:    bufio.NewScanner(reader),
		existsData: true,
		awkVars: AWKVars{
			RS: defaultRS,
			FS: defaultFS,
		},
	}

	lr.scanner.Split(lr.scanLinesBySep)
	lr.buffer.Grow(bufferSizeLimit)

	return lr
}

func (lr *Reader) scanLinesBySep(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, lr.awkVars.RS); i >= 0 {
		// We have a full RS-terminated line.
		return i + 1, data[0 : i+1], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}

	// Request more data.
	return 0, nil, nil
}

// Read - implement io.Reader interface
func (lr *Reader) Read(p []byte) (n int, err error) {
	var (
		bufErr, filterErr error
		lineBytes         []byte
	)

	for lr.existsData && bufErr == nil && lr.buffer.Len() < bufferSizeLimit {
		if lr.existsData = lr.scanner.Scan(); !lr.existsData {
			break
		}

		lineBytes = lr.scanner.Bytes()
		lr.awkVars.NR++

		for _, filterFunc := range lr.filterFuncs {
			lineBytes, filterErr = filterFunc(lineBytes)
			if filterErr != nil {
				switch filterErr {
				case ErrOmitLine:
					lineBytes = nullBytes
				default:
					bufErr = filterErr
				}
				break
			}
		}

		_, _ = lr.buffer.Write(lineBytes) // #nosec - err always is nil
	}

	if !lr.existsData && bufErr == nil {
		bufErr = lr.scanner.Err()
	}

	n, err = lr.buffer.Read(p)
	if err != nil && bufErr == nil {
		bufErr = err
	}

	return n, bufErr
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

// Each - processing each line.
// Do not save the value of the byte slice, since it can change in the next filter-steps.
func (lr *Reader) Each(filterFn func([]byte)) *Reader {
	return lr.MapErr(func(line []byte) ([]byte, error) {
		filterFn(line)
		return line, nil
	})
}

// EachString - processing each line as string
func (lr *Reader) EachString(filterFn func(string)) *Reader {
	return lr.MapErr(func(line []byte) ([]byte, error) {
		filterFn(string(line))
		return line, nil
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
	return lr.MapErr(func(line []byte) ([]byte, error) {
		addRS := false
		RS := []byte{lr.awkVars.RS}
		if bytes.HasSuffix(line, RS) {
			addRS = true
			line = bytes.TrimSuffix(line, RS)
		}

		lineStr := string(line)
		fields := lr.awkVars.FS.Split(lineStr, -1)
		lr.awkVars.NF = len(fields)
		result, err := filterFn(lineStr, fields, lr.awkVars)
		if err != nil {
			return nullBytes, err
		}

		resultBytes := []byte(result)
		if !bytes.HasSuffix(resultBytes, RS) && addRS {
			resultBytes = append(resultBytes, lr.awkVars.RS)
		}
		return resultBytes, nil
	})
}

// Discard - read all content from Reader for side effect from filter functions
func (lr *Reader) Discard() error {
	_, err := io.Copy(ioutil.Discard, lr)
	return err
}

// ReadAllSlice - read all content from Reader by lines to slice of []byte
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
