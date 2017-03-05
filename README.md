# byline Reader [![GoDoc](https://godoc.org/github.com/msoap/byline?status.svg)](https://godoc.org/github.com/msoap/byline) [![Build Status](https://travis-ci.org/msoap/byline.svg?branch=master)](https://travis-ci.org/msoap/byline) [![Coverage Status](https://coveralls.io/repos/github/msoap/byline/badge.svg?branch=master)](https://coveralls.io/github/msoap/byline?branch=master) [![Sourcegraph](https://sourcegraph.com/github.com/msoap/byline/-/badge.svg)](https://sourcegraph.com/github.com/msoap/byline?badge) [![Report Card](https://goreportcard.com/badge/github.com/msoap/byline)](https://goreportcard.com/report/github.com/msoap/byline)
Convert Go Reader to line-by-line Reader

Example, add line number to each line and add suffix at the end:
```Go
	reader := strings.NewReader("111\n222\n333")
    // or reader, err := os.Open("file")

	i := 1
	blr := byline.NewReader(reader).Map(func(line []byte) []byte {
		newLine := fmt.Sprintf("(%d) %s", i, string(line))
		i++
		return []byte(newLine)
	}).MapErr(func(line []byte) ([]byte, error) {
		return regexp.MustCompile(`\n?$`).ReplaceAll(line, []byte(" suf\n")), nil
	})

	result, err := ioutil.ReadAll(blr)

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

	// get all types from Go-source
	sm := StateMachine{
		beginRe: regexp.MustCompile(`^type `),
		endRe:   regexp.MustCompile(`^}\s+$`),
	}

	lr := byline.NewReader(file).Grep(sm.SMFilter).Map(func(line []byte) []byte {
		// and remove comments
		return regexp.MustCompile(`\s+//.+`).ReplaceAll(line, []byte{})
	})

	result, err := ioutil.ReadAll(lr)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Print(string(result))
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