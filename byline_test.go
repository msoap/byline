package byline_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/msoap/byline"
	"github.com/stretchr/testify/require"
)

func TestMap(t *testing.T) {
	reader := strings.NewReader("111\n222\n333")

	i := 1
	lr := byline.NewReader(reader).Map(func(line []byte) []byte {
		newLine := fmt.Sprintf("%d. %s", i, string(line))
		i++
		return []byte(newLine)
	})

	result, err := ioutil.ReadAll(lr)
	require.NoError(t, err)
	require.Equal(t, "1. 111\n2. 222\n3. 333", string(result))
}

func TestMapErr(t *testing.T) {
	reader := strings.NewReader("111\n222\n333")

	i := 1
	lr := byline.NewReader(reader).MapErr(func(line []byte) ([]byte, error) {
		newLine := fmt.Sprintf("(%d) %s", i, string(line))
		i++
		return []byte(newLine), nil
	}).MapErr(func(line []byte) ([]byte, error) {
		return regexp.MustCompile(`\n?$`).ReplaceAll(line, []byte(" suf\n")), nil
	})

	result, err := ioutil.ReadAll(lr)
	require.NoError(t, err)
	require.Equal(t, "(1) 111 suf\n(2) 222 suf\n(3) 333 suf\n", string(result))
}

// truncate stream
func TestMapErrWithError(t *testing.T) {
	reader := strings.NewReader("111\n222\n333")

	lr := byline.NewReader(reader).MapErr(func(line []byte) ([]byte, error) {
		if bytes.HasPrefix(line, []byte("222")) {
			return line, io.EOF
		}
		return line, nil
	})

	result, err := ioutil.ReadAll(lr)
	require.NoError(t, err)
	require.Equal(t, "111\n222\n", string(result))
}

func TestMapString(t *testing.T) {
	reader := strings.NewReader("111\n222\n333")

	i := 0
	lr := byline.NewReader(reader).MapString(func(line string) string {
		i++
		return fmt.Sprintf("%d. %s", i, line)
	})

	result, err := ioutil.ReadAll(lr)
	require.NoError(t, err)
	require.Equal(t, "1. 111\n2. 222\n3. 333", string(result))
}

func TestMapStringErr(t *testing.T) {
	reader := strings.NewReader("111\n222\n333")

	i := 0
	lr := byline.NewReader(reader).MapStringErr(func(line string) (string, error) {
		i++
		if i == 2 {
			return fmt.Sprintf("%d. %s", i, line), io.EOF
		}

		return fmt.Sprintf("%d. %s", i, line), nil
	})

	result, err := ioutil.ReadAll(lr)
	require.NoError(t, err)
	require.Equal(t, "1. 111\n2. 222\n", string(result))
}

func TestGrep(t *testing.T) {
	reader := strings.NewReader("111\n222\n333")

	i := 0
	lr := byline.NewReader(reader).Grep(func(line []byte) bool {
		i++
		return !(i == 2)
	})

	result, err := ioutil.ReadAll(lr)
	require.NoError(t, err)
	require.Equal(t, "111\n333", string(result))
}

func TestGrepString(t *testing.T) {
	reader := strings.NewReader("111\n222\n333")

	lr := byline.NewReader(reader).GrepString(func(line string) bool {
		return !strings.HasPrefix(line, "222\n")
	})

	result, err := ioutil.ReadAll(lr)
	require.NoError(t, err)
	require.Equal(t, "111\n333", string(result))
}

func TestGrepByRegexp(t *testing.T) {
	reader := strings.NewReader("111\n222\n333\n")

	lr := byline.NewReader(reader).GrepByRegexp(regexp.MustCompile(`111|222`))

	result, err := ioutil.ReadAll(lr)
	require.NoError(t, err)
	require.Equal(t, "111\n222\n", string(result))
}

func TestAWKMode(t *testing.T) {
	reader := strings.NewReader(`1,name one,12.3#2,second row;7.1#3,three row;15.51`)

	lr := byline.NewReader(reader).
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
			}

			return fmt.Sprintf("%s/%d - %s", fields[0], vars.NR, fields[1]), nil
		})

	result, err := ioutil.ReadAll(lr)
	require.NoError(t, err)
	require.Equal(t, "1/1 - name one#3/3 - three row", string(result))
}

func TestAWKModeWithError(t *testing.T) {
	reader := strings.NewReader(`1 name_one 12.3#2 error_row#3 three row  15.51`)

	lr := byline.NewReader(reader).
		SetRS('#').
		AWKMode(func(line string, fields []string, vars byline.AWKVars) (string, error) {
			if vars.NF < 3 {
				return "", fmt.Errorf("csv parse failed for %q", line)
			}
			return fields[0] + " - " + fields[1], nil
		})

	_, err := ioutil.ReadAll(lr)
	require.Error(t, err)
}
