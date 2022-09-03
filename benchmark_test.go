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
	bytesSlice = getBytes()
)

func getBytes() []byte {
	var data bytes.Buffer
	for i := 0; i < linesCount; i++ {
		_, _ = fmt.Fprintf(&data, "%d line\n", i)
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
				res = append(res, scanner.Bytes()...)
				res = append(res, '\n')
			}
		}

		require.NoError(b, scanner.Err())
		require.True(b, len(res) > len(bytesSlice)/2-1)
	}
}

func Benchmark_NativeScannerOnlyCount(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NR := 0
		reader := bytes.NewReader(bytesSlice)
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			NR++
		}

		require.NoError(b, scanner.Err())
		require.Equal(b, linesCount, NR)
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
			}
			return line, nil
		}).ReadAll()
		require.NoError(b, err)
		require.True(b, len(res) > len(bytesSlice)/2-1)
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
			}
			return line, nil
		}).ReadAll()
		require.NoError(b, err)
		require.True(b, len(res) > len(bytesSlice)/2-1)
	}
}

func Benchmark_Grep(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NR := 0
		reader := bytes.NewReader(bytesSlice)
		res, err := byline.NewReader(reader).Grep(func([]byte) bool {
			NR++
			return NR%2 != 0
		}).ReadAll()
		require.NoError(b, err)
		require.True(b, len(res) > len(bytesSlice)/2-1)
	}
}

func Benchmark_GrepString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NR := 0
		reader := bytes.NewReader(bytesSlice)
		res, err := byline.NewReader(reader).GrepString(func(string) bool {
			NR++
			return NR%2 != 0
		}).ReadAll()
		require.NoError(b, err)
		require.True(b, len(res) > len(bytesSlice)/2-1)
	}
}

func Benchmark_Each(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NR := 0
		reader := bytes.NewReader(bytesSlice)
		err := byline.NewReader(reader).Each(func([]byte) {
			NR++
		}).Discard()
		require.NoError(b, err)
		require.Equal(b, linesCount, NR)
	}
}

func Benchmark_EachString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NR := 0
		reader := bytes.NewReader(bytesSlice)
		err := byline.NewReader(reader).EachString(func(string) {
			NR++
		}).Discard()
		require.NoError(b, err)
		require.Equal(b, linesCount, NR)
	}
}

func Benchmark_AWKMode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(bytesSlice)
		res, err := byline.NewReader(reader).AWKMode(func(line string, _ []string, vars byline.AWKVars) (string, error) {
			if vars.NR%2 == 0 {
				return "", byline.ErrOmitLine
			}
			return line, nil
		}).ReadAll()
		require.NoError(b, err)
		require.True(b, len(res) > len(bytesSlice)/2-1)
	}
}
