package byline_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/msoap/byline"
)

func Example() {
	reader := strings.NewReader(`CSV Title
CSV description
ID,NAME,PRICE
A001,name one,12.3

A002,second row;7.1
A003,three row;15.51
Total: ....
Some text
`)

	lr := byline.NewReader(reader).
		GrepString(func(line string) bool {
			// skip empty lines
			return line != "" && line != "\n"
		}).
		Grep(func(line []byte) bool {
			return !bytes.HasPrefix(line, []byte("CSV"))
		}).
		SetFS(regexp.MustCompile(`[,;]`)).
		AWKMode(func(line string, fields []string, _ byline.AWKVars) (string, error) {
			// skip header
			if strings.HasPrefix(fields[0], "ID") {
				return "", byline.ErrOmitLine
			}
			// skip footer
			if strings.HasPrefix(fields[0], "Total:") {
				return "", io.EOF
			}
			return line, nil
		}).
		MapString(func(line string) string {
			return "Z" + line
		}).
		AWKMode(func(line string, fields []string, vars byline.AWKVars) (string, error) {
			if vars.NF < 3 {
				return "", fmt.Errorf("csv parse failed for %q", line)
			}

			return fmt.Sprintf("%s - %s (line:%d)", fields[0], fields[1], vars.NR), nil
		})

	result, err := lr.ReadAllString()
	fmt.Print("\n", result, err)
	// Output:
	// ZA001 - name one (line:4)
	// ZA002 - second row (line:6)
	// ZA003 - three row (line:7)
	// <nil>
}

func ExampleReader_AWKMode() {
	reader := strings.NewReader(`ID,NAME,PRICE
A001,name one,12.3
A002,second row;7.1
A003,three row;15.51
Total: ....
Some text
`)

	sum := 0.0
	lr := byline.NewReader(reader).
		SetFS(regexp.MustCompile(`[,;]`)).
		AWKMode(func(line string, fields []string, vars byline.AWKVars) (string, error) {
			if vars.NR == 1 {
				// skip first line
				return "", byline.ErrOmitLine
			}

			if vars.NF > 0 && strings.HasPrefix(fields[0], "Total:") {
				// skip rest of file
				return "", io.EOF
			}

			if vars.NF < 3 {
				return "", fmt.Errorf("csv parse failed for %q", line)
			}

			if price, err := strconv.ParseFloat(fields[2], 64); err != nil {
				return "", err
			} else if price < 10 {
				return "", byline.ErrOmitLine
			} else {
				sum += price
			}

			return fmt.Sprintf("line:%d. %s - %s", vars.NR, fields[0], fields[1]), nil
		})

	result, err := lr.ReadAllString()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Print(result)
	fmt.Printf("Sum: %.2f", sum)
	// Output: line:2. A001 - name one
	// line:4. A003 - three row
	// Sum: 27.81
}

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

	lr := byline.NewReader(file).Grep(sm.SMFilter).Map(func(line []byte) []byte {
		// and remove comments
		return regexp.MustCompile(`\s+//.+`).ReplaceAll(line, []byte{})
	})

	result, err := lr.ReadAllString()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Print("\n" + result)
	// Output:
	// type Reader struct {
	// 	scanner     *bufio.Scanner
	// 	buffer      bytes.Buffer
	// 	existsData  bool
	// 	filterFuncs []func(line []byte) ([]byte, error)
	// 	awkVars     AWKVars
	// }
	// type AWKVars struct {
	// 	NR int
	// 	NF int
	// 	RS byte
	// 	FS *regexp.Regexp
	// }
}

func ExampleReader_GrepByRegexp() {
	reader := strings.NewReader(`ID,NAME,PRICE
A001,name one,12.3
A002,second row;7.1
A003,three row;15.51
Total: ....
Some text
`)

	result, err := byline.NewReader(reader).GrepByRegexp(regexp.MustCompile(`^A\d+,`)).ReadAllString()
	fmt.Print("\n"+result, err)
	// Output:
	// A001,name one,12.3
	// A002,second row;7.1
	// A003,three row;15.51
	// <nil>
}

func ExampleReader_MapStringErr() {
	reader := strings.NewReader(`
100000
200000
300000
end ...
Some text
`)

	result, err := byline.NewReader(reader).
		MapStringErr(func(line string) (string, error) {
			switch {
			case line == "" || line == "\n":
				return "", byline.ErrOmitLine
			case strings.HasPrefix(line, "end "):
				return "", io.EOF
			default:
				return "<" + line, nil
			}
		}).
		ReadAllString()

	fmt.Print("\n"+result, err)
	// Output:
	// <100000
	// <200000
	// <300000
	// <nil>
}

func ExampleReader_Each() {
	reader := strings.NewReader(`1 1 1
2 2 2
3 3 3
`)

	spacesCount, bytesCount, linesCount := 0, 0, 0
	err := byline.NewReader(reader).
		Each(func(line []byte) {
			linesCount++
			bytesCount += len(line)
			for _, b := range line {
				if b == ' ' {
					spacesCount++
				}
			}
		}).Discard()

	if err == nil {
		fmt.Printf("spaces: %d, bytes: %d, lines: %d\n", spacesCount, bytesCount, linesCount)
	}
	// Output: spaces: 6, bytes: 18, lines: 3
}

func ExampleReader_EachString() {
	reader := strings.NewReader(`111
222
333
`)

	result := []string{}
	err := byline.NewReader(reader).
		EachString(func(line string) {
			result = append(result, line)
		}).Discard()

	if err == nil {
		fmt.Printf("%q\n", result)
	}
	// Output: ["111\n" "222\n" "333\n"]
}
