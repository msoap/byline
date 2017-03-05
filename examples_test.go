package byline_test

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/msoap/byline"
)

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
		SetFS(regexp.MustCompile(`,|;`)).
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

			if price, err := strconv.ParseFloat(fields[2], 10); err != nil {
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

	// get all types from Go-source
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
	// 	bufReader   *bufio.Reader
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
