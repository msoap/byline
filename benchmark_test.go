package byline_test

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"

	"github.com/msoap/byline"
	"github.com/stretchr/testify/require"
)

const linesCount = 10000

var (
	bytesSlice  = getBytes()
	prefixBytes = []byte("pre:")
)

func getBytes() []byte {
	var data bytes.Buffer
	for i := 0; i < linesCount; i++ {
		fmt.Fprintf(&data, fmt.Sprintf("%d. line data\n", i))
	}

	return data.Bytes()
}

func Benchmark_NativeScannerBytes(b *testing.B) {

	for i := 0; i < b.N; i++ {
		NR := 0
		reader := bytes.NewReader(bytesSlice)
		scanner := bufio.NewScanner(reader)
		res := []byte{}
		for scanner.Scan() {
			NR++
			if NR%2 != 0 {
				res = append(res, prefixBytes...)
				res = append(res, scanner.Bytes()...)
			}
		}

		require.NoError(b, scanner.Err())
		require.True(b, len(res) > linesCount*5)
	}
}

func Benchmark_MapBytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NR := 0
		reader := bytes.NewReader(bytesSlice)
		res, err := byline.NewReader(reader).MapErr(func(line []byte) ([]byte, error) {
			NR++
			if NR%2 == 0 {
				return nil, byline.ErrOmitLine
			} else {
				line = append(prefixBytes, line...)
				return line, nil
			}
		}).ReadAll()
		require.NoError(b, err)
		require.True(b, len(res) > linesCount*5)
	}
}

func Benchmark_MapString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NR := 0
		reader := bytes.NewReader(bytesSlice)
		res, err := byline.NewReader(reader).MapStringErr(func(line string) (string, error) {
			NR++
			if NR%2 == 0 {
				return "", byline.ErrOmitLine
			} else {
				return "pre:" + line, nil
			}
		}).ReadAll()
		require.NoError(b, err)
		require.True(b, len(res) > linesCount*5)
	}
}

func Benchmark_AWKMode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(bytesSlice)
		res, err := byline.NewReader(reader).AWKMode(func(line string, fields []string, vars byline.AWKVars) (string, error) {
			if vars.NR%2 == 0 {
				return "", byline.ErrOmitLine
			} else {
				return "pre:" + line, nil
			}
		}).ReadAll()
		require.NoError(b, err)
		require.True(b, len(res) > linesCount*5)
	}
}
