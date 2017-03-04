package byline_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
	"testing"

	"github.com/msoap/byline"
	"github.com/stretchr/testify/require"
)

func TestMapErr(t *testing.T) {
	reader := strings.NewReader("111\n222\n333")

	i := 0
	lr := byline.NewReader(reader).MapErr(func(line []byte) ([]byte, error) {
		newLine := fmt.Sprintf("(%d) %s", i, string(line))
		i++
		return []byte(newLine), nil
	}).MapErr(func(line []byte) ([]byte, error) {
		return regexp.MustCompile(`\n?$`).ReplaceAll(line, []byte(" suf\n")), nil
	})

	result, err := ioutil.ReadAll(lr)
	require.NoError(t, err)
	require.Equal(t, "(0) 111 suf\n(1) 222 suf\n(2) 333 suf\n", string(result))
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
