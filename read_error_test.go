package byline_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/msoap/byline"
)

func TestReadAllKeepsFilterOutputBeforeError(t *testing.T) {
	failure := errors.New("filter failed")
	for _, tc := range []struct {
		name    string
		stopErr error
		wantErr error
	}{
		{name: "EOF", stopErr: io.EOF},
		{name: "custom error", stopErr: failure, wantErr: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := strings.Repeat("x", 1101)
			calls := 0
			reader := byline.NewReader(strings.NewReader("first\nstop\ntail\n")).MapStringErr(func(line string) (string, error) {
				calls++
				if line == "stop\n" {
					return output, tc.stopErr
				}
				return line, nil
			})

			got, err := io.ReadAll(reader)
			if err != tc.wantErr {
				t.Fatalf("first ReadAll error = %v, want %v", err, tc.wantErr)
			}
			if want := "first\n" + output; string(got) != want {
				t.Fatalf("first ReadAll returned %d bytes, want %d", len(got), len(want))
			}
			if calls != 2 {
				t.Fatalf("filter called %d times before error, want 2", calls)
			}

			tail, err := io.ReadAll(reader)
			if err != nil || string(tail) != "tail\n" {
				t.Fatalf("second ReadAll = %q, %v; want tail and no error", tail, err)
			}
			if calls != 3 {
				t.Fatalf("filter called %d times after resume, want 3", calls)
			}
		})
	}
}

type readerWithFailure struct {
	remaining []byte
	failure   error
	combined  bool
}

func (r *readerWithFailure) Read(p []byte) (int, error) {
	if len(r.remaining) == 0 {
		return 0, r.failure
	}
	n := copy(p, r.remaining)
	r.remaining = r.remaining[n:]
	if r.combined && len(r.remaining) == 0 {
		return n, r.failure
	}
	return n, nil
}

func TestReadKeepsSourceOutputBeforeError(t *testing.T) {
	failure := errors.New("source failed")
	want := strings.Repeat("a", 700) + "\n" + strings.Repeat("b", 700)
	for _, combined := range []bool{false, true} {
		t.Run(fmt.Sprintf("errorWithData=%t", combined), func(t *testing.T) {
			reader := byline.NewReader(&readerWithFailure{
				remaining: []byte(want), failure: failure, combined: combined,
			})
			buf := make([]byte, 37)
			var got bytes.Buffer
			for reads := 0; reads <= len(want); reads++ {
				n, err := reader.Read(buf)
				if n > 0 {
					if err != nil {
						t.Fatalf("Read returned data and error together: %d, %v", n, err)
					}
					got.Write(buf[:n])
					continue
				}
				if err != failure {
					t.Fatalf("Read after source output = %d, %v; want 0, %v", n, err, failure)
				}
				if got.String() != want {
					t.Fatalf("read output has %d bytes, want %d", got.Len(), len(want))
				}
				if n, err := reader.Read(buf); n != 0 || err != failure {
					t.Fatalf("read after source error = %d, %v; want 0, %v", n, err, failure)
				}
				return
			}
			t.Fatal("Read never reported the source error")
		})
	}
}

func TestReadReportsFilterErrorAfterBufferedOutput(t *testing.T) {
	failure := errors.New("filter failed")
	for _, size := range []int{1, 512, 2048} {
		t.Run(fmt.Sprintf("bufferSize=%d", size), func(t *testing.T) {
			// The preceding line and filter result fill the internal 512-byte buffer exactly.
			want := "head\n" + strings.Repeat("x", 507)
			reader := byline.NewReader(strings.NewReader("head\nstop\ntail\n")).MapStringErr(func(line string) (string, error) {
				if line == "stop\n" {
					return strings.Repeat("x", 507), failure
				}
				return line, nil
			})
			buf := make([]byte, size)
			var got bytes.Buffer
			for reads := 0; reads <= len(want); reads++ {
				n, err := reader.Read(buf)
				if n > 0 {
					if err != nil {
						t.Fatalf("Read returned data and error together: %d, %v", n, err)
					}
					got.Write(buf[:n])
					continue
				}
				if err != failure {
					t.Fatalf("Read after output = %d, %v; want 0, %v", n, err, failure)
				}
				if got.String() != want {
					t.Fatalf("read output has %d bytes, want %d", got.Len(), len(want))
				}
				tail, err := io.ReadAll(reader)
				if err != nil || string(tail) != "tail\n" {
					t.Fatalf("read after filter error = %q, %v; want tail and no error", tail, err)
				}
				return
			}
			t.Fatal("Read never reported the filter error")
		})
	}
}

func TestEmptyReadDoesNotAdvanceReader(t *testing.T) {
	failure := errors.New("filter failed")
	calls := 0
	reader := byline.NewReader(strings.NewReader("stop\ntail\n")).MapStringErr(func(line string) (string, error) {
		calls++
		if line == "stop\n" {
			return strings.Repeat("x", 700), failure
		}
		return line, nil
	})
	if n, err := reader.Read(nil); n != 0 || err != nil || calls != 0 {
		t.Fatalf("empty read before scanning = %d, %v; filter calls = %d", n, err, calls)
	}
	buf := make([]byte, 1)
	if n, err := reader.Read(buf); n != 1 || err != nil || calls != 1 {
		t.Fatalf("first data read = %d, %v; filter calls = %d", n, err, calls)
	}
	if n, err := reader.Read(nil); n != 0 || err != nil || calls != 1 {
		t.Fatalf("empty read with pending error = %d, %v; filter calls = %d", n, err, calls)
	}
	remaining, err := io.ReadAll(reader)
	if err != failure || string(remaining) != strings.Repeat("x", 699) {
		t.Fatalf("remaining output = %d bytes, %v; want 699 bytes and %v", len(remaining), err, failure)
	}
	tail, err := io.ReadAll(reader)
	if err != nil || string(tail) != "tail\n" {
		t.Fatalf("read after filter error = %q, %v; want tail and no error", tail, err)
	}
}
