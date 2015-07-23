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
		scr.rsScanner = nil // Force a new input stream.
		scr.input = bufio.NewReader(strings.NewReader(allRecordsStr))
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
	scr.RS = " "
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
	scr.FS = ","
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
	scr.Run(strings.NewReader("dummy data"))
	if val != 1234 {
		t.Fatalf("Expected 1234 but received %d", val)
	}
}
