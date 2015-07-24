// This file tests script primitives.

package awk

import (
	"bufio"
	"regexp"
	"strings"
	"testing"
)

// TestReadRecordNewline tests reading newline-separated records.
func TestReadRecordNewline(t *testing.T) {
	// Define the basic test we plan to repeat.
	allRecords := []string{"X", "Word", "More than one word", "", "More text"}
	allRecordsStr := strings.Join(allRecords, "\n")
	scr := NewScript()
	doTest := func() {
		scr.input = bufio.NewReader(strings.NewReader(allRecordsStr))
		scr.SetRS("\n")
		for _, oneRecord := range allRecords {
			rec, err := scr.readRecord()
			if err != nil {
				t.Fatal(err)
			}
			if rec != oneRecord {
				t.Fatalf("Expected %q but received %q", oneRecord, rec)
			}
		}
	}

	// Test with no trailing newline.
	doTest()

	// Test with a trailing newline.
	allRecordsStr += "\n"
	doTest()
}

// TestReadRecordWhitespace tests reading whitespace-separated records.
func TestReadRecordWhitespace(t *testing.T) {
	allRecordsStr := "  banana banana banana  banana   banana banana\tbanana banana\nbanana banana"
	scr := NewScript()
	scr.input = bufio.NewReader(strings.NewReader(allRecordsStr))
	scr.SetRS(" ")
	for i := 0; i < 10; i++ {
		rec, err := scr.readRecord()
		if err != nil {
			t.Fatal(err)
		}
		if rec != "banana" {
			t.Fatalf("Expected %q but received %q", "banana", rec)
		}
	}
}

// TestSplitRecordWhitespace tests splitting a record into
// whitespace-separated fields.
func TestSplitRecordWhitespace(t *testing.T) {
	recordStr := "The woods are lovely,  dark and    deep,"
	fields := regexp.MustCompile(`\s+`).Split(recordStr, -1)
	scr := NewScript()
	scr.splitRecord(recordStr)
	for i, f := range fields {
		if scr.F[i+1].String() != f {
			t.Fatalf("Expected %q but received %q", f, scr.F[i+1])
		}
	}
}

// TestSplitRecordComma tests splitting a record into comma-separated fields.
func TestSplitRecordComma(t *testing.T) {
	recordStr := "The woods are lovely,  dark and    deep,"
	fields := strings.Split(recordStr, ",")
	scr := NewScript()
	scr.SetFS(",")
	scr.splitRecord(recordStr)
	for i, f := range fields {
		if scr.F[i+1].String() != f {
			t.Fatalf("Expected %q but received %q", f, scr.F[i+1])
		}
	}
}

// TestBeginEnd tests creating and running a script that contains a Begin
// action and an End action.
func TestBeginEnd(t *testing.T) {
	scr := NewScript()
	val := 123
	scr.AppendStmt(Begin, func(s *Script) { val *= 10 })
	scr.AppendStmt(End, func(s *Script) { val += 4 })
	err := scr.Run(strings.NewReader("dummy data"))
	if err != nil {
		t.Fatal(err)
	}
	if val != 1234 {
		t.Fatalf("Expected 1234 but received %d", val)
	}
}

// TestSimpleSum tests adding up a column of numbers.
func TestSimpleSum(t *testing.T) {
	scr := NewScript()
	sum := 0
	scr.AppendStmt(nil, func(s *Script) { sum += s.F[1].Int() })
	err := scr.Run(strings.NewReader("2\n4\n6\n8\n"))
	if err != nil {
		t.Fatal(err)
	}
	if sum != 20 {
		t.Fatalf("Expected 20 but received %d", sum)
	}
}

// TestRunTwice tests running the same script twice.
func TestRunTwice(t *testing.T) {
	// Run once.
	scr := NewScript()
	sum := 0
	scr.AppendStmt(nil, func(s *Script) { sum += s.F[1].Int() * s.NR })
	err := scr.Run(strings.NewReader("1\n3\n5\n7\n"))
	if err != nil {
		t.Fatal(err)
	}
	if sum != 50 {
		t.Fatalf("Expected 50 but received %d on the first trial", sum)
	}

	// Run again.
	sum = 0
	err = scr.Run(strings.NewReader("1\n3\n5\n7\n"))
	if err != nil {
		t.Fatal(err)
	}
	if sum != 50 {
		t.Fatalf("Expected 50 but received %d on the second trial", sum)
	}
}
