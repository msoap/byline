# byline Reader [![GoDoc](https://godoc.org/github.com/msoap/byline?status.svg)](https://godoc.org/github.com/msoap/byline) [![Build Status](https://travis-ci.org/msoap/byline.svg?branch=master)](https://travis-ci.org/msoap/byline) [![Coverage Status](https://coveralls.io/repos/github/msoap/byline/badge.svg?branch=master)](https://coveralls.io/github/msoap/byline?branch=master) [![Sourcegraph](https://sourcegraph.com/github.com/msoap/byline/-/badge.svg)](https://sourcegraph.com/github.com/msoap/byline?badge) [![Report Card](https://goreportcard.com/badge/github.com/msoap/byline)](https://goreportcard.com/report/github.com/msoap/byline)
Convert Go Reader to line-by-line Reader

Example, add line number to each line and add suffix at the end:
```Go
	reader := strings.NewReader("111\n222\n333")
    // or reader, err := os.Open("file")

	i := 1
	blr := byline.NewReader(reader).MapErr(func(line []byte) ([]byte, error) {
		newLine := fmt.Sprintf("(%d) %s", i, string(line))
		i++
		return []byte(newLine), nil
	}).MapErr(func(line []byte) ([]byte, error) {
		return regexp.MustCompile(`\n?$`).ReplaceAll(line, []byte(" suf\n")), nil
	})

	result, err := ioutil.ReadAll(blr)

```