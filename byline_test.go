package byline_test

import (
	"bytes"
	"errors"
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

type errorAfterDataReader struct {
	data     *strings.Reader
	err      error
	together bool
}

func (r *errorAfterDataReader) Read(p []byte) (int, error) {
	n, err := r.data.Read(p)
	if err == io.EOF || r.together && r.data.Len() == 0 {
		err = r.err
	}
	return n, err
}

func TestReadScannerErrorDrainsOutput(t *testing.T) {
	want := strings.Repeat("a", 999) + "\n"
	wantErr := errors.New("source failure")
	for _, together := range []bool{false, true} {
		for _, size := range []int{1, 4, 511, 512, 513, 2048} {
			t.Run(fmt.Sprintf("together=%t/size=%d", together, size), func(t *testing.T) {
				lr := byline.NewReader(&errorAfterDataReader{
					data: strings.NewReader(want), err: wantErr, together: together,
				})
				var got bytes.Buffer
				buf := make([]byte, size)
				for reads := 0; ; reads++ {
					require.Less(t, reads, len(want)+1)
					n, err := lr.Read(buf)
					_, _ = got.Write(buf[:n])
					if err != nil {
						require.Equal(t, wantErr, err)
						break
					}
				}
				require.Equal(t, want, got.String())
				n, err := lr.Read(buf)
				require.Zero(t, n)
				require.Equal(t, wantErr, err)
			})
		}
	}
}

func TestReadFilterErrorDrainsOutput(t *testing.T) {
	for _, filterErr := range []error{io.EOF, errors.New("filter failure")} {
		for _, output := range []string{"", "stop\n", strings.Repeat("b", 1100) + "\n"} {
			for _, size := range []int{1, 4, 511, 512, 513, 2048} {
				t.Run(fmt.Sprintf("error=%s/output=%d/size=%d", filterErr, len(output), size), func(t *testing.T) {
					calls := 0
					lr := byline.NewReader(strings.NewReader("first\nstop\ntail\n")).MapStringErr(func(line string) (string, error) {
						calls++
						if line == "stop\n" {
							return output, filterErr
						}
						return line, nil
					})
					var got bytes.Buffer
					buf := make([]byte, size)
					for reads := 0; ; reads++ {
						require.Less(t, reads, len(output)+10)
						n, err := lr.Read(buf)
						_, _ = got.Write(buf[:n])
						if err != nil {
							require.Equal(t, filterErr, err)
							break
						}
						if reads == 0 {
							n, err = lr.Read(nil)
							require.Zero(t, n)
							require.NoError(t, err)
						}
					}
					require.Equal(t, "first\n"+output, got.String())
					require.Equal(t, 2, calls)
					tail, err := lr.ReadAllString()
					require.NoError(t, err)
					require.Equal(t, "tail\n", tail)
					require.Equal(t, 3, calls)
				})
			}
		}
	}
}

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
		return regexp.MustCompile(`(\n?)$`).ReplaceAll(line, []byte(" suf$1")), nil
	})

	result, err := ioutil.ReadAll(lr)
	require.NoError(t, err)
	require.Equal(t, "(1) 111 suf\n(2) 222 suf\n(3) 333 suf", string(result))
}

// truncate stream
func TestMapErrWithError(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{
			in:  "111\n222\n333\n444\n555",
			out: "111\n222\n333\n",
		},
		{
			in:  "111\n222\n333\n444\n",
			out: "111\n222\n333\n",
		},
		{
			in:  "111\n222\n333\n444",
			out: "111\n222\n333\n",
		},
		{
			in:  "111\n222\n333\n",
			out: "111\n222\n333\n",
		},
		{
			in:  "111\n222\n333",
			out: "111\n222\n333",
		},
		{
			in:  "111\n222\n",
			out: "111\n222\n",
		},
		{
			in:  "111\n222",
			out: "111\n222",
		},
		{
			in:  "111\n",
			out: "111\n",
		},
		{
			in:  "111",
			out: "111",
		},
		{
			in:  "\n",
			out: "\n",
		},
		{
			in:  "",
			out: "",
		},
		{
			in:  "1\n3333333333333333333333333333333333333333333333333333333333333333",
			out: "1\n3333333333333333333333333333333333333333333333333333333333333333",
		},
		{
			in:  "3333333333333333333333333333333333333333333333333333333333333333",
			out: "3333333333333333333333333333333333333333333333333333333333333333",
		},
	}

	for i, row := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			reader := strings.NewReader(row.in)

			lr := byline.NewReader(reader).MapErr(func(line []byte) ([]byte, error) {
				if bytes.HasPrefix(line, []byte("333")) {
					return line, io.EOF
				}
				return line, nil
			})

			result, err := lr.ReadAll()
			require.NoError(t, err)
			require.Equal(t, row.out, string(result))
		})
	}
}

func TestMapString(t *testing.T) {
	reader := strings.NewReader("111\n222\n333")

	i := 0
	lr := byline.NewReader(reader).MapString(func(line string) string {
		i++
		return fmt.Sprintf("%d. %s", i, line)
	})

	result, err := lr.ReadAll()
	require.NoError(t, err)
	require.Equal(t, "1. 111\n2. 222\n3. 333", string(result))
}

func TestMapStringErr(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{
			in:  "111\n222\n333\n",
			out: "1. 111\n2. 222\n",
		},
		{
			in:  "111\n222\n333",
			out: "1. 111\n2. 222\n",
		},
		{
			in:  "111\n222\n",
			out: "1. 111\n2. 222\n",
		},
		{
			in:  "111\n222",
			out: "1. 111\n2. 222",
		},
		{
			in:  "111\n",
			out: "1. 111\n",
		},
		{
			in:  "111",
			out: "1. 111",
		},
		{
			in:  "\n",
			out: "1. \n",
		},
		{
			in:  "",
			out: "",
		},
	}

	for i, row := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			reader := strings.NewReader(row.in)

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
			require.Equal(t, row.out, string(result))
		})
	}
}

func TestEach(t *testing.T) {
	cases := []struct {
		in  string
		out []string
	}{
		{
			in:  "111\n222\n333\n",
			out: []string{"111\n", "222\n", "333\n"},
		},
		{
			in:  "111\n222\n333",
			out: []string{"111\n", "222\n", "333"},
		},
		{
			in:  "111\n222\n",
			out: []string{"111\n", "222\n"},
		},
		{
			in:  "111\n222",
			out: []string{"111\n", "222"},
		},
		{
			in:  "111\n",
			out: []string{"111\n"},
		},
		{
			in:  "111",
			out: []string{"111"},
		},
		{
			in:  "",
			out: []string{},
		},
	}

	for i, row := range cases {
		t.Run(fmt.Sprintf("%d. on bytes", i), func(t *testing.T) {
			reader := strings.NewReader(row.in)

			out := []string{}
			err := byline.NewReader(reader).Each(func(line []byte) {
				out = append(out, string(line))
			}).Discard()

			require.NoError(t, err)
			require.Equal(t, row.out, out)
		})

		t.Run(fmt.Sprintf("%d. on string", i), func(t *testing.T) {
			reader := strings.NewReader(row.in)

			out := []string{}
			err := byline.NewReader(reader).EachString(func(line string) {
				out = append(out, line)
			}).Discard()

			require.NoError(t, err)
			require.Equal(t, row.out, out)
		})
	}

	t.Run("Each can change line inplace", func(t *testing.T) {
		reader := strings.NewReader("111\n222\n333")

		out, err := byline.NewReader(reader).Each(func(line []byte) {
			line[0] = 'A'
		}).ReadAllString()

		require.NoError(t, err)
		require.Equal(t, "A11\nA22\nA33", out)
	})
}

func TestGrep(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{
			in:  "111\n222\n333",
			out: "111\n333",
		},
		{
			in:  "111\n222\n333\n",
			out: "111\n333\n",
		},
		{
			in:  "111\n222\n",
			out: "111\n",
		},
		{
			in:  "111\n",
			out: "111\n",
		},
		{
			in:  "111",
			out: "111",
		},
		{
			in:  "\n",
			out: "\n",
		},
		{
			in:  "",
			out: "",
		},
	}

	for i, row := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			reader := strings.NewReader(row.in)

			i := 0
			lr := byline.NewReader(reader).Grep(func(_ []byte) bool {
				i++
				return !(i == 2)
			})

			result, err := ioutil.ReadAll(lr)
			require.NoError(t, err)
			require.Equal(t, row.out, string(result))

		})
	}
}

func TestGrepString(t *testing.T) {
	reader := strings.NewReader("111\n222\n333\n")

	lr := byline.NewReader(reader).GrepString(func(line string) bool {
		return !strings.HasPrefix(line, "222")
	})

	result, err := ioutil.ReadAll(lr)
	require.NoError(t, err)
	require.Equal(t, "111\n333\n", string(result))
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
		SetFS(regexp.MustCompile(`[,;]`)).
		AWKMode(func(line string, fields []string, vars byline.AWKVars) (string, error) {
			if vars.NF < 3 {
				return "", fmt.Errorf("csv parse failed for %q", line)
			}

			if price, err := strconv.ParseFloat(fields[2], 64); err != nil {
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

func TestReadAll(t *testing.T) {
	t.Run("ReadAllNilReader", func(t *testing.T) {
		var reader io.Reader
		_, err := byline.NewReader(reader).
			SetRS('#').
			MapString(func(line string) string { return "<" + line }).
			ReadAll()
		require.Exactly(t, byline.ErrNilReader, err, "ReadAllNilReader")

	})

	t.Run("ReadAll", func(t *testing.T) {
		reader := strings.NewReader(`1 name_one 12.3#2 error_row#3 three row  15.51#4 row#5 row end`)
		result, err := byline.NewReader(reader).
			SetRS('#').
			MapString(func(line string) string { return "<" + line }).
			ReadAll()
		require.NoError(t, err)
		require.EqualValues(t, "<1 name_one 12.3#<2 error_row#<3 three row  15.51#<4 row#<5 row end", result)
	})

	t.Run("ReadAllString", func(t *testing.T) {
		reader := strings.NewReader(`1 name_one 12.3#2 error_row#3 three row  15.51#4 row#5 row end`)
		result, err := byline.NewReader(reader).
			SetRS('#').
			MapString(func(line string) string { return "<" + line }).
			ReadAllString()
		require.NoError(t, err)
		require.EqualValues(t, "<1 name_one 12.3#<2 error_row#<3 three row  15.51#<4 row#<5 row end", result)
	})

	t.Run("ReadAllSlice", func(t *testing.T) {
		reader := strings.NewReader(`1 name_one 12.3#2 error_row#`)
		result, err := byline.NewReader(reader).
			SetRS('#').
			MapString(func(line string) string { return "<" + line }).
			ReadAllSlice()
		require.NoError(t, err)
		require.EqualValues(t, [][]byte{[]byte("<1 name_one 12.3#"), []byte("<2 error_row#")}, result)
	})

	t.Run("ReadAllSliceString", func(t *testing.T) {
		reader := strings.NewReader(`1 name_one 12.3#2 error_row`)
		result, err := byline.NewReader(reader).
			SetRS('#').
			MapString(func(line string) string { return "<" + line }).
			ReadAllSliceString()
		require.NoError(t, err)
		require.EqualValues(t, []string{"<1 name_one 12.3#", "<2 error_row"}, result)
	})

	t.Run("ReadAllSliceStringWithLastNL", func(t *testing.T) {
		reader := strings.NewReader(`1 name_one 12.3#2 error_row#`)
		result, err := byline.NewReader(reader).
			SetRS('#').
			MapString(func(line string) string { return "<" + line }).
			ReadAllSliceString()
		require.NoError(t, err)
		require.EqualValues(t, []string{"<1 name_one 12.3#", "<2 error_row#"}, result)
	})

	t.Run("ReadAllSliceStringWithLastEmptyLine", func(t *testing.T) {
		reader := strings.NewReader(`1 name_one 12.3#2 error_row##`)
		result, err := byline.NewReader(reader).
			SetRS('#').
			MapString(func(line string) string { return "<" + line }).
			ReadAllSliceString()
		require.NoError(t, err)
		require.EqualValues(t, []string{"<1 name_one 12.3#", "<2 error_row#", "<#"}, result)
	})
}

func TestLongLines(t *testing.T) {
	reader := strings.NewReader("01234567890123456789012345678901234567890123456789\n01234567890123456789012345678901234567890123456789")
	lr := byline.NewReader(reader).MapString(func(line string) string { return "<" + line })

	smallBuf := make([]byte, 10)
	n, err := lr.Read(smallBuf)
	require.NoError(t, err)
	require.Equal(t, 10, n)
	require.EqualValues(t, []byte("<012345678"), smallBuf)

	rest, err := ioutil.ReadAll(lr)
	require.NoError(t, err)
	require.EqualValues(t, []byte("90123456789012345678901234567890123456789\n<01234567890123456789012345678901234567890123456789"), rest)
}
