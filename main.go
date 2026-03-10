package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `epoch - epoch ↔ RFC3339

Usage:
  epoch <value>

Rules:
  - If <value> is a pure number (optional leading +/-), it's treated as epoch.
    The tool will show interpretations for seconds, milliseconds, and nanoseconds.
  - If <value> contains ":" it's treated as RFC3339-like (with or without timezone).
  - Otherwise, the program fails.

Outputs:
  - For epoch input:
      • RFC3339 (UTC and Local) for each possible unit (s, ms, ns).
  - For RFC input:
      • Epoch (UTC) in s, ms, and ns.

Examples:
  epoch 1700000000
  epoch 2023-11-14T22:13:20Z
`)
	}
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		fail("provide exactly one value to convert")
	}
	input := strings.TrimSpace(args[0])

	if number, err := strconv.ParseInt(input, 10, 0); err == nil {
		handleEpoch(int64(number))
		return
	}
	if strings.Contains(input, ":") {
		handleRFC(input)
		return
	}
	fail("input must be an epoch or RFC3339")
}

func handleEpoch(n int64) {
	tS := time.Unix(n, 0)
	fmt.Println("--- Seconds ---")
	fmt.Printf("RFC3339 (UTC)  : %s\n", tS.UTC().Format(time.RFC3339))
	fmt.Printf("RFC3339 (Local): %s\n", tS.In(time.Local).Format(time.RFC3339))
	fmt.Println()

	tMS := time.Unix(0, n*int64(time.Millisecond))
	const rfc3339ms = "2006-01-02T15:04:05.000Z07:00"
	fmt.Println("--- Milliseconds ---")
	fmt.Printf("RFC3339 (UTC)  : %s\n", tMS.UTC().Format(rfc3339ms))
	fmt.Printf("RFC3339 (Local): %s\n", tMS.In(time.Local).Format(rfc3339ms))
	fmt.Println()

	tNS := time.Unix(0, n)
	const rfc3339ns = "2006-01-02T15:04:05.000000000Z07:00"
	fmt.Println("--- Nanoseconds ---")
	fmt.Printf("RFC3339 (UTC)  : %s\n", tNS.UTC().Format(rfc3339ns))
	fmt.Printf("RFC3339 (Local): %s\n", tNS.In(time.Local).Format(rfc3339ns))
}

func handleRFC(s string) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		hasZone := hasExplicitZone(s)
		printEpochFromTime(t, hasZone)
		return
	}

	layoutsNoTZ := []string{
		"2006-01-02T15:04:05.999999999",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, l := range layoutsNoTZ {
		if t, err := time.ParseInLocation(l, s, time.Local); err == nil {
			printEpochFromTime(t, false)
			return
		}
	}

	fail("unable to parse RFC3339 / timestamp-like value")
}

func printEpochFromTime(t time.Time, hasZone bool) {
	utc := t.UTC()

	fmt.Println("--- epoch ---")
	fmt.Println("s  (UTC)", utc.Unix())
	fmt.Println("ms (UTC)", utc.UnixMilli())
	fmt.Println("ns (UTC)", utc.UnixNano())
}

func hasExplicitZone(s string) bool {
	return strings.Contains(s, "Z") || strings.Contains(s, "+") || strings.Contains(s, "-")
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "error:", msg)
	os.Exit(1)
}
