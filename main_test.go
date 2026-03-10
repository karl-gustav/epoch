package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

// withFixedLocal temporarily sets time.Local to a fixed zone and restores it after.
func withFixedLocal(offsetSeconds int, name string, fn func()) {
	orig := time.Local
	time.Local = time.FixedZone(name, offsetSeconds)
	defer func() { time.Local = orig }()
	fn()
}

// captureStdout captures stdout output produced inside fn and returns it as string.
func captureStdout(fn func()) string {
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestHasExplicitZone_CurrentImpl(t *testing.T) {
	// Matches the CURRENT implementation which treats any '-' as "has zone".
	// So date strings without timezone still return true due to '-'.
	cases := []struct {
		in   string
		want bool
	}{
		{"2023-11-14T22:13:20Z", true},
		{"2023-11-14T22:13:20+02:00", true},
		{"2023-11-14T22:13:20-07:00", true},
		{"2023-11-14T22:13:20", true}, // current impl: true (because of '-')
		{"2023-11-14 22:13:20", true}, // current impl: true (because of '-')
		{"20231114T221320", false},    // no Z, +, or -, so false
	}

	for _, c := range cases {
		got := hasExplicitZone(c.in)
		if got != c.want {
			t.Errorf("hasExplicitZone(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestHandleEpoch_PrintFormatsDeterministic(t *testing.T) {
	// Use a fixed local zone: UTC+02:00 to make "Local" outputs predictable
	withFixedLocal(2*3600, "TEST+02", func() {
		out := captureStdout(func() {
			handleEpoch(1700000000) // seconds since epoch
		})

		// Expected values:
		// Seconds (UTC):   2023-11-14T22:13:20Z
		// Seconds (Local): 2023-11-15T00:13:20+02:00
		//
		// Milliseconds: interpret n as milliseconds since epoch:
		// 1,700,000,000 ms = 1970-01-20T16:13:20Z
		// Local (+02:00):  1970-01-20T18:13:20+02:00
		//
		// Nanoseconds: interpret n as nanoseconds since epoch:
		// 1,700,000,000 ns = 1.7 seconds -> 1970-01-01T00:00:01.700000000Z
		// Local (+02:00):  1970-01-01T02:00:01.700000000+02:00

		wantSnippets := []string{
			"--- Seconds ---",
			"RFC3339 (UTC)  : 2023-11-14T22:13:20Z",
			"RFC3339 (Local): 2023-11-15T00:13:20+02:00",

			"--- Milliseconds ---",
			"RFC3339 (UTC)  : 1970-01-20T16:13:20.000Z",
			"RFC3339 (Local): 1970-01-20T18:13:20.000+02:00",

			"--- Nanoseconds ---",
			"RFC3339 (UTC)  : 1970-01-01T00:00:01.700000000Z",
			"RFC3339 (Local): 1970-01-01T02:00:01.700000000+02:00",
		}

		for _, s := range wantSnippets {
			if !strings.Contains(out, s) {
				t.Fatalf("handleEpoch output missing expected line:\nwant contains: %q\nfull output:\n%s", s, out)
			}
		}
	})
}

func TestHandleRFC_WithTimezoneZ(t *testing.T) {
	out := captureStdout(func() {
		handleRFC("2023-11-14T22:13:20Z")
	})

	// Expected epoch for 2023-11-14T22:13:20Z is 1700000000
	wantSnippets := []string{
		"--- epoch ---",
		"s  (UTC) 1700000000",
		"ms (UTC) 1700000000000",
		"ns (UTC) 1700000000000000000",
	}
	for _, s := range wantSnippets {
		if !strings.Contains(out, s) {
			t.Fatalf("handleRFC(Z) output missing expected line:\nwant contains: %q\nfull output:\n%s", s, out)
		}
	}
}

func TestHandleRFC_NoTimezone_LocalAssumed(t *testing.T) {
	withFixedLocal(2*3600, "TEST+02", func() {
		// Input without timezone should be treated as Local time.
		// 2023-11-14 22:13:20 LOCAL(+02:00) == 2023-11-14 20:13:20 UTC
		// epoch = 1700000000 - 7200 = 1699992800
		out := captureStdout(func() {
			handleRFC("2023-11-14 22:13:20")
		})

		wantSnippets := []string{
			"--- epoch ---",
			"s  (UTC) 1699992800",
			"ms (UTC) 1699992800000",
			"ns (UTC) 1699992800000000000",
		}
		for _, s := range wantSnippets {
			if !strings.Contains(out, s) {
				t.Fatalf("handleRFC(no tz) output missing expected line:\nwant contains: %q\nfull output:\n%s", s, out)
			}
		}
	})
}

func TestHandleRFC_SupportsMultipleNoTZLayouts(t *testing.T) {
	withFixedLocal(2*3600, "TEST+02", func() {
		cases := []struct {
			in          string
			expectEpoch int64
		}{
			// "2006-01-02T15:04:05.999999999"
			{"2023-11-14T22:13:20.123456789", 1699992800},
			// "2006-01-02 15:04:05.999999999"
			{"2023-11-14 22:13:20.999999999", 1699992800},
			// "2006-01-02T15:04:05"
			{"2023-11-14T22:13:20", 1699992800},
			// "2006-01-02 15:04:05"
			{"2023-11-14 22:13:20", 1699992800},
			// "2006-01-02T15:04"
			{"2023-11-14T22:13", 1699992780}, // 22:13:00 local => 20:13:00Z
			// "2006-01-02 15:04"
			{"2023-11-14 22:13", 1699992780},
			// "2006-01-02"
			{"2023-11-14", 1699912800}, // 00:00 local => 22:00Z previous day
		}

		for _, c := range cases {
			out := captureStdout(func() { handleRFC(c.in) })
			line := "s  (UTC) " + int64ToString(c.expectEpoch)
			if !strings.Contains(out, line) {
				t.Fatalf("handleRFC(%q) missing expected epoch line:\nwant contains: %q\nfull output:\n%s", c.in, line, out)
			}
		}
	})
}

// int64ToString without fmt to keep expectations explicit
func int64ToString(n int64) string {
	var b [20]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	if n == 0 {
		return "0"
	}
	for n > 0 && i > 0 {
		i--
		b[i] = byte('0' + (n % 10))
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
